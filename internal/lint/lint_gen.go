//go:build !plan

// Package lint implements converge's lint mode, which checks that
// converge conventions are followed in a Go project.
package lint

import (
	"bufio"
	"fmt"
	"go/ast"
	"go/build/constraint"
	"go/parser"
	"go/token"
	"io/fs"
	"os"
	"path/filepath"
	"strconv"
	"strings"
)

// Severity indicates how serious an issue is.
type Severity int

const (
	Error   Severity = iota // must be fixed
	Warning                 // should be fixed
)

// Issue represents a single lint finding.
type Issue struct {
	File     string   // file path (relative to root)
	Line     int      // line number (0 if not applicable)
	Severity Severity // error or warning
	Message  string   // human-readable description
	Fixed    bool     // true if -fix resolved it
}

// Result holds all lint findings.
type Result struct {
	Issues []Issue
}

const convergeImportPath = "github.com/tessro/converge"

func (s Severity) String() string {
	switch s {
	case Error:
		return "error"
	case Warning:
		return "warning"
	default:
		return "unknown"
	}
}

// HasErrors reports whether any issues are errors.
func (r *Result) HasErrors() bool {
	for _, iss := range r.Issues {
		if iss.Severity == Error && !iss.Fixed {
			return true
		}
	}
	return false
}

// Run performs lint checks across all Go files under root.
// If fix is true, it attempts to auto-fix issues where possible.
func Run(root string, fix bool) (*Result, error) {
	root, err := filepath.Abs(root)
	if err != nil {
		return nil, err
	}

	var result Result

	// Track which packages have plan files, for the "other files" warning.
	planPackages := make(map[string]bool)

	// First pass: find all plan packages and check individual files.
	err = filepath.WalkDir(root, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if !d.IsDir() {
			return nil
		}
		name := d.Name()
		if name != "." && strings.HasPrefix(name, ".") {
			return filepath.SkipDir
		}
		if name == "vendor" || name == "testdata" {
			return filepath.SkipDir
		}

		entries, err := os.ReadDir(path)
		if err != nil {
			return err
		}

		for _, entry := range entries {
			if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".go") {
				continue
			}
			if strings.HasSuffix(entry.Name(), "_test.go") {
				continue
			}

			filePath := filepath.Join(path, entry.Name())
			issues, isPlan, err := lintFile(root, filePath, fix)
			if err != nil {
				return fmt.Errorf("linting %s: %w", filePath, err)
			}
			result.Issues = append(result.Issues, issues...)
			if isPlan {
				planPackages[path] = true
			}
		}

		return nil
	})
	if err != nil {
		return nil, err
	}

	// Second pass: warn about non-plan, non-gen files in plan packages.
	for pkgDir := range planPackages {
		entries, err := os.ReadDir(pkgDir)
		if err != nil {
			continue
		}
		for _, entry := range entries {
			name := entry.Name()
			if entry.IsDir() || !strings.HasSuffix(name, ".go") {
				continue
			}
			if strings.HasSuffix(name, "_test.go") {
				continue
			}
			if strings.HasSuffix(name, "_plan.go") || strings.HasSuffix(name, "_gen.go") {
				continue
			}
			relPath, _ := filepath.Rel(root, filepath.Join(pkgDir, name))
			result.Issues = append(result.Issues, Issue{
				File:     relPath,
				Severity: Warning,
				Message:  "file is not a _plan.go or _gen.go file in a converge-managed package",
			})
		}
	}

	return &result, nil
}

// lintFile checks a single Go file for converge convention violations.
// It returns issues found, whether the file is a plan file, and any error.
func lintFile(root, path string, fix bool) ([]Issue, bool, error) {
	relPath, _ := filepath.Rel(root, path)
	filename := filepath.Base(path)
	isPlanName := strings.HasSuffix(filename, "_plan.go")

	fset := token.NewFileSet()
	f, err := parser.ParseFile(fset, path, nil, parser.ParseComments)
	if err != nil {
		return nil, false, err
	}

	importsConverge := fileImportsConverge(f)
	hasPlanTag := fileHasPlanTag(path)

	var issues []Issue
	isPlan := isPlanName || importsConverge || hasPlanTag

	// Check: converge import in non-plan-named file.
	if importsConverge && !isPlanName {
		issues = append(issues, Issue{
			File:     relPath,
			Line:     convergeImportLine(fset, f),
			Severity: Error,
			Message:  "converge package imported in file not named *_plan.go",
		})
	}

	// Check: converge import without //go:build plan.
	if importsConverge && !hasPlanTag {
		fixed := false
		if fix {
			if err := addPlanBuildTag(path); err == nil {
				fixed = true
			}
		}
		issues = append(issues, Issue{
			File:     relPath,
			Severity: Error,
			Message:  "file imports converge but lacks //go:build plan",
			Fixed:    fixed,
		})
	}

	// Check: file named _plan.go without //go:build plan.
	if isPlanName && !hasPlanTag {
		fixed := false
		if fix {
			if err := addPlanBuildTag(path); err == nil {
				fixed = true
			}
		}
		issues = append(issues, Issue{
			File:     relPath,
			Severity: Warning,
			Message:  "file named *_plan.go but lacks //go:build plan",
			Fixed:    fixed,
		})
	}

	// Check: file has //go:build plan but isn't named _plan.go.
	if hasPlanTag && !isPlanName {
		issues = append(issues, Issue{
			File:     relPath,
			Severity: Warning,
			Message:  "file has //go:build plan but is not named *_plan.go",
		})
	}

	return issues, isPlan, nil
}

func fileImportsConverge(f *ast.File) bool {
	for _, imp := range f.Imports {
		path, err := strconv.Unquote(imp.Path.Value)
		if err != nil {
			continue
		}
		if path == convergeImportPath {
			return true
		}
	}
	return false
}

func convergeImportLine(fset *token.FileSet, f *ast.File) int {
	for _, imp := range f.Imports {
		path, err := strconv.Unquote(imp.Path.Value)
		if err != nil {
			continue
		}
		if path == convergeImportPath {
			return fset.Position(imp.Pos()).Line
		}
	}
	return 0
}

// fileHasPlanTag checks whether a file contains a //go:build constraint
// that requires the "plan" build tag. It properly evaluates the constraint
// expression — "//go:build plan" returns true, "//go:build !plan" returns false.
func fileHasPlanTag(path string) bool {
	f, err := os.Open(path)
	if err != nil {
		return false
	}
	defer f.Close()

	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())

		// Build constraints must appear before the package clause.
		if strings.HasPrefix(line, "package ") {
			break
		}

		if constraint.IsGoBuild(line) {
			expr, err := constraint.Parse(line)
			if err != nil {
				continue
			}
			return requiresPlanTag(expr)
		}
	}
	return false
}

// requiresPlanTag reports whether a build constraint expression requires
// the "plan" tag — i.e., the file is included with plan=true and excluded
// with plan=false.
func requiresPlanTag(expr constraint.Expr) bool {
	withPlan := expr.Eval(func(tag string) bool { return true })
	withoutPlan := expr.Eval(func(tag string) bool { return tag != "plan" })
	return withPlan && !withoutPlan
}

// addPlanBuildTag inserts "//go:build plan" at the top of a Go file.
func addPlanBuildTag(path string) error {
	content, err := os.ReadFile(path)
	if err != nil {
		return err
	}

	lines := strings.SplitAfter(string(content), "\n")

	// Find the insertion point: before the package declaration,
	// after any existing //go:build lines.
	insertAt := 0
	for i, line := range lines {
		trimmed := strings.TrimSpace(line)
		if strings.HasPrefix(trimmed, "package ") {
			insertAt = i
			break
		}
		// Skip past existing build constraints and blank lines.
		if strings.HasPrefix(trimmed, "//go:build ") || strings.HasPrefix(trimmed, "// +build ") {
			insertAt = i + 1
		}
	}

	// Build the new content.
	var b strings.Builder
	for i, line := range lines {
		if i == insertAt {
			b.WriteString("//go:build plan\n\n")
		}
		b.WriteString(line)
	}

	return os.WriteFile(path, []byte(b.String()), 0o644)
}

// Format renders lint results as human/LLM-readable text.
func Format(r *Result) string {
	if len(r.Issues) == 0 {
		return "ok — no issues found\n"
	}

	var b strings.Builder
	errors := 0
	warnings := 0
	fixed := 0

	for _, iss := range r.Issues {
		var prefix string
		switch iss.Severity {
		case Error:
			prefix = "ERROR"
			if !iss.Fixed {
				errors++
			}
		case Warning:
			prefix = "WARN"
			if !iss.Fixed {
				warnings++
			}
		}

		loc := iss.File
		if iss.Line > 0 {
			loc = fmt.Sprintf("%s:%d", iss.File, iss.Line)
		}

		status := ""
		if iss.Fixed {
			status = " (fixed)"
			fixed++
		}

		fmt.Fprintf(&b, "  %s  %s: %s%s\n", prefix, loc, iss.Message, status)
	}

	b.WriteString("\n")
	parts := []string{}
	if errors > 0 {
		parts = append(parts, fmt.Sprintf("%d %s", errors, plural(errors, "error", "errors")))
	}
	if warnings > 0 {
		parts = append(parts, fmt.Sprintf("%d %s", warnings, plural(warnings, "warning", "warnings")))
	}
	if fixed > 0 {
		parts = append(parts, fmt.Sprintf("%d fixed", fixed))
	}
	if len(parts) == 0 {
		b.WriteString("ok — all issues fixed\n")
	} else {
		fmt.Fprintf(&b, "%s\n", strings.Join(parts, ", "))
	}

	return b.String()
}

func plural(n int, singular, p string) string {
	if n == 1 {
		return singular
	}
	return p
}
