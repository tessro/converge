//go:build !plan

// Package check implements converge's check mode, which compares the
// exported API surface of Go packages built with and without the "plan"
// build tag.
package check

import (
	"fmt"
	"io/fs"
	"path/filepath"
	"strings"

	"github.com/tessro/converge/internal/sig"
)

// PackageResult holds the comparison result for a single package.
type PackageResult struct {
	Package  string             // package name
	Dir      string             // directory (relative to root)
	Missing  []sig.Export        // in plan build but not in default build
	Extra    []sig.Export        // in default build but not in plan build
	Mismatch []MismatchedExport // same key, different signature
	Matched  int                // count of matching exports
}

// MismatchedExport pairs a plan export with an impl export that share
// the same key but have different signatures.
type MismatchedExport struct {
	Plan sig.Export
	Impl sig.Export
}

// Result holds the overall check result across all packages.
type Result struct {
	Packages []PackageResult
}

// HasIssues reports whether this package has any differences.
func (r *PackageResult) HasIssues() bool {
	return len(r.Missing) > 0 || len(r.Extra) > 0 || len(r.Mismatch) > 0
}

// OK reports whether all packages passed the check.
func (r *Result) OK() bool {
	for i := range r.Packages {
		if r.Packages[i].HasIssues() {
			return false
		}
	}
	return true
}

// PackagesWithIssues returns only the packages that have differences.
func (r *Result) PackagesWithIssues() []PackageResult {
	var out []PackageResult
	for _, p := range r.Packages {
		if p.HasIssues() {
			out = append(out, p)
		}
	}
	return out
}

// Run performs the check across all Go packages under root.
func Run(root string) (*Result, error) {
	root, err := filepath.Abs(root)
	if err != nil {
		return nil, err
	}

	var result Result

	err = filepath.WalkDir(root, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if !d.IsDir() {
			return nil
		}
		// Skip hidden directories, vendor, testdata.
		name := d.Name()
		if name != "." && strings.HasPrefix(name, ".") {
			return filepath.SkipDir
		}
		if name == "vendor" || name == "testdata" {
			return filepath.SkipDir
		}

		// Only check packages that have plan files.
		hasPlan, err := sig.HasPlanFiles(path)
		if err != nil {
			return err
		}
		if !hasPlan {
			return nil
		}

		pkgResult, err := checkPackage(root, path)
		if err != nil {
			return fmt.Errorf("checking %s: %w", path, err)
		}
		if pkgResult != nil {
			result.Packages = append(result.Packages, *pkgResult)
		}
		return nil
	})

	return &result, err
}

func checkPackage(root, dir string) (*PackageResult, error) {
	planExports, err := sig.Extract(dir, []string{"plan"})
	if err != nil {
		return nil, fmt.Errorf("extracting plan exports: %w", err)
	}

	implExports, err := sig.Extract(dir, nil)
	if err != nil {
		return nil, fmt.Errorf("extracting impl exports: %w", err)
	}

	relDir, err := filepath.Rel(root, dir)
	if err != nil {
		relDir = dir
	}
	relDir = "./" + relDir

	// Build maps by key.
	planMap := make(map[string]sig.Export, len(planExports))
	for _, e := range planExports {
		planMap[e.Key()] = e
	}
	implMap := make(map[string]sig.Export, len(implExports))
	for _, e := range implExports {
		implMap[e.Key()] = e
	}

	result := &PackageResult{
		Package: filepath.Base(dir),
		Dir:     relDir,
	}

	// Find missing (in plan but not impl).
	for _, pe := range planExports {
		ie, found := implMap[pe.Key()]
		if !found {
			result.Missing = append(result.Missing, pe)
		} else if pe.Signature != ie.Signature {
			result.Mismatch = append(result.Mismatch, MismatchedExport{Plan: pe, Impl: ie})
		} else {
			result.Matched++
		}
	}

	// Find extra (in impl but not plan).
	for _, ie := range implExports {
		if _, found := planMap[ie.Key()]; !found {
			result.Extra = append(result.Extra, ie)
		}
	}

	return result, nil
}

// Format renders the check result as human/LLM-readable text.
func Format(r *Result) string {
	var b strings.Builder

	issues := r.PackagesWithIssues()
	total := len(r.Packages)

	if len(issues) == 0 {
		fmt.Fprintf(&b, "ok — %d %s checked, all exports match\n",
			total, plural(total, "package", "packages"))
		return b.String()
	}

	for _, pkg := range issues {
		fmt.Fprintf(&b, "=== package %s (%s) ===\n", pkg.Package, pkg.Dir)

		if len(pkg.Missing) > 0 {
			b.WriteString("\n  MISSING (in plan, not yet implemented):\n\n")
			for _, e := range pkg.Missing {
				fmt.Fprintf(&b, "    %s\n", e.Signature)
				fmt.Fprintf(&b, "        %s:%d\n", e.File, e.Line)
				if e.Imagine != "" {
					fmt.Fprintf(&b, "        %q\n", e.Imagine)
				}
			}
		}

		if len(pkg.Extra) > 0 {
			b.WriteString("\n  EXTRA (implemented but not in plan):\n\n")
			for _, e := range pkg.Extra {
				fmt.Fprintf(&b, "    %s\n", e.Signature)
				fmt.Fprintf(&b, "        %s:%d\n", e.File, e.Line)
			}
		}

		if len(pkg.Mismatch) > 0 {
			b.WriteString("\n  MISMATCHED:\n\n")
			for _, m := range pkg.Mismatch {
				fmt.Fprintf(&b, "    %s:\n", m.Plan.Name)
				fmt.Fprintf(&b, "      plan: %s\n", m.Plan.Signature)
				fmt.Fprintf(&b, "            %s:%d\n", m.Plan.File, m.Plan.Line)
				fmt.Fprintf(&b, "      impl: %s\n", m.Impl.Signature)
				fmt.Fprintf(&b, "            %s:%d\n", m.Impl.File, m.Impl.Line)
			}
		}

		if pkg.Matched > 0 {
			fmt.Fprintf(&b, "\n  (%d %s matched)\n", pkg.Matched, plural(pkg.Matched, "export", "exports"))
		}
		b.WriteString("\n")
	}

	failed := len(issues)
	fmt.Fprintf(&b, "FAIL — %d of %d %s %s issues\n",
		failed, total, plural(total, "package", "packages"),
		plural(failed, "has", "have"))

	return b.String()
}

func plural(n int, singular, p string) string {
	if n == 1 {
		return singular
	}
	return p
}
