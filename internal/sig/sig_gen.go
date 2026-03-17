//go:build !plan

// Package sig extracts exported declarations from Go packages
// under different build tag configurations.
package sig

import (
	"bytes"
	"fmt"
	"go/ast"
	"go/build"
	"go/format"
	"go/parser"
	"go/token"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
)

// Export represents a single exported declaration in a Go package.
type Export struct {
	Name      string // symbol name, e.g. "Foo"
	Receiver  string // receiver type for methods, e.g. "*Server"; empty for non-methods
	Kind      string // "func", "type", "var", "const"
	Signature string // canonical rendered declaration (no body for funcs)
	File      string // source filename (relative to package dir)
	Line      int    // line number in source file
	Imagine   string // converge.Imagine description, if found in the function body
}

// Key returns a unique identifier for matching this export across builds.
// For methods it includes the receiver type; for all others just kind+name.
func (e Export) Key() string {
	if e.Receiver != "" {
		return "func (" + e.Receiver + ")." + e.Name
	}
	return e.Kind + " " + e.Name
}

// Extract returns all exported declarations from Go files in dir,
// using the given build tags to determine which files are included.
func Extract(dir string, tags []string) ([]Export, error) {
	ctx := build.Default
	ctx.BuildTags = tags

	pkg, err := ctx.ImportDir(dir, 0)
	if err != nil {
		if _, ok := err.(*build.NoGoError); ok {
			return nil, nil
		}
		return nil, fmt.Errorf("importing %s: %w", dir, err)
	}

	fset := token.NewFileSet()
	var exports []Export

	goFiles := append([]string{}, pkg.GoFiles...)
	goFiles = append(goFiles, pkg.CgoFiles...)

	for _, filename := range goFiles {
		fullpath := filepath.Join(dir, filename)
		f, err := parser.ParseFile(fset, fullpath, nil, parser.ParseComments)
		if err != nil {
			return nil, fmt.Errorf("parsing %s: %w", filename, err)
		}

		// Determine the converge import alias (if any) for Imagine extraction.
		convergeAlias := convergeImportAlias(f)

		for _, decl := range f.Decls {
			switch d := decl.(type) {
			case *ast.FuncDecl:
				if !d.Name.IsExported() {
					continue
				}
				e := exportFromFunc(fset, d, filename)
				if convergeAlias != "" && d.Body != nil {
					e.Imagine = findImagineDesc(d.Body, convergeAlias)
				}
				exports = append(exports, e)
			case *ast.GenDecl:
				exports = append(exports, exportsFromGenDecl(fset, d, filename)...)
			}
		}
	}

	sort.Slice(exports, func(i, j int) bool {
		return exports[i].Key() < exports[j].Key()
	})

	return exports, nil
}

// HasPlanFiles reports whether dir contains any _plan.go files.
func HasPlanFiles(dir string) (bool, error) {
	matches, err := filepath.Glob(filepath.Join(dir, "*_plan.go"))
	if err != nil {
		return false, err
	}
	return len(matches) > 0, nil
}

func exportFromFunc(fset *token.FileSet, decl *ast.FuncDecl, file string) Export {
	e := Export{
		Name: decl.Name.Name,
		Kind: "func",
		File: file,
		Line: fset.Position(decl.Pos()).Line,
	}

	if decl.Recv != nil && len(decl.Recv.List) > 0 {
		e.Receiver = renderNode(fset, decl.Recv.List[0].Type)
	}

	// Render function signature without body.
	stripped := *decl
	stripped.Body = nil
	stripped.Doc = nil
	e.Signature = renderNode(fset, &stripped)

	return e
}

func exportsFromGenDecl(fset *token.FileSet, decl *ast.GenDecl, file string) []Export {
	var exports []Export

	switch decl.Tok {
	case token.TYPE:
		for _, spec := range decl.Specs {
			ts := spec.(*ast.TypeSpec)
			if !ts.Name.IsExported() {
				continue
			}
			e := Export{
				Name: ts.Name.Name,
				Kind: "type",
				File: file,
				Line: fset.Position(ts.Pos()).Line,
			}
			single := &ast.GenDecl{
				Tok:   token.TYPE,
				Specs: []ast.Spec{ts},
			}
			e.Signature = renderNode(fset, single)
			exports = append(exports, e)
		}

	case token.VAR:
		exports = append(exports, exportsFromValueSpecs(fset, decl, "var", file)...)

	case token.CONST:
		exports = append(exports, exportsFromConstSpecs(fset, decl, file)...)
	}

	return exports
}

func exportsFromValueSpecs(fset *token.FileSet, decl *ast.GenDecl, kind string, file string) []Export {
	var exports []Export

	for _, spec := range decl.Specs {
		vs := spec.(*ast.ValueSpec)
		for i, name := range vs.Names {
			if !name.IsExported() {
				continue
			}
			e := Export{
				Name: name.Name,
				Kind: kind,
				File: file,
				Line: fset.Position(name.Pos()).Line,
			}

			singleSpec := &ast.ValueSpec{
				Names: []*ast.Ident{name},
				Type:  vs.Type,
			}
			if i < len(vs.Values) {
				singleSpec.Values = []ast.Expr{vs.Values[i]}
			}
			single := &ast.GenDecl{
				Tok:   decl.Tok,
				Specs: []ast.Spec{singleSpec},
			}
			e.Signature = renderNode(fset, single)
			exports = append(exports, e)
		}
	}

	return exports
}

func exportsFromConstSpecs(fset *token.FileSet, decl *ast.GenDecl, file string) []Export {
	var exports []Export

	// Track inherited type for iota-style const blocks.
	var lastType ast.Expr
	var lastValues []ast.Expr

	for _, spec := range decl.Specs {
		vs := spec.(*ast.ValueSpec)

		if vs.Type != nil {
			lastType = vs.Type
		}
		if vs.Values != nil {
			lastValues = vs.Values
		}

		for i, name := range vs.Names {
			if !name.IsExported() {
				continue
			}
			e := Export{
				Name: name.Name,
				Kind: "const",
				File: file,
				Line: fset.Position(name.Pos()).Line,
			}

			// Build the effective spec, expanding inherited type/values.
			effectiveType := vs.Type
			if effectiveType == nil {
				effectiveType = lastType
			}
			singleSpec := &ast.ValueSpec{
				Names: []*ast.Ident{name},
				Type:  effectiveType,
			}
			if i < len(vs.Values) {
				singleSpec.Values = []ast.Expr{vs.Values[i]}
			} else if vs.Values == nil && lastValues != nil && i < len(lastValues) {
				// Inherit value expression (iota-style).
				singleSpec.Values = []ast.Expr{lastValues[i]}
			}

			single := &ast.GenDecl{
				Tok:   token.CONST,
				Specs: []ast.Spec{singleSpec},
			}
			e.Signature = renderNode(fset, single)
			exports = append(exports, e)
		}
	}

	return exports
}

// convergeImportAlias returns the local name used for the converge package
// import, or "" if converge is not imported.
func convergeImportAlias(f *ast.File) string {
	for _, imp := range f.Imports {
		path, err := strconv.Unquote(imp.Path.Value)
		if err != nil {
			continue
		}
		if path == "github.com/tessro/converge" {
			if imp.Name != nil {
				if imp.Name.Name == "." {
					return "." // dot import
				}
				return imp.Name.Name
			}
			return "converge"
		}
	}
	return ""
}

// findImagineDesc looks for a converge.Imagine("...") call in a function
// body and returns the description string. Returns "" if not found.
func findImagineDesc(body *ast.BlockStmt, alias string) string {
	if body == nil {
		return ""
	}
	for _, stmt := range body.List {
		exprStmt, ok := stmt.(*ast.ExprStmt)
		if !ok {
			continue
		}
		call, ok := exprStmt.X.(*ast.CallExpr)
		if !ok {
			continue
		}

		if alias == "." {
			// Dot import: Imagine("...")
			ident, ok := call.Fun.(*ast.Ident)
			if !ok || ident.Name != "Imagine" {
				continue
			}
		} else {
			// Qualified: converge.Imagine("...") or alias.Imagine("...")
			sel, ok := call.Fun.(*ast.SelectorExpr)
			if !ok {
				continue
			}
			ident, ok := sel.X.(*ast.Ident)
			if !ok || ident.Name != alias || sel.Sel.Name != "Imagine" {
				continue
			}
		}

		if len(call.Args) > 0 {
			if lit, ok := call.Args[0].(*ast.BasicLit); ok && lit.Kind == token.STRING {
				s, err := strconv.Unquote(lit.Value)
				if err == nil {
					return s
				}
			}
		}
	}
	return ""
}

func renderNode(fset *token.FileSet, node ast.Node) string {
	var buf bytes.Buffer
	if err := format.Node(&buf, fset, node); err != nil {
		// Fallback: shouldn't happen for valid AST nodes.
		return fmt.Sprintf("<render error: %v>", err)
	}
	return strings.TrimSpace(buf.String())
}
