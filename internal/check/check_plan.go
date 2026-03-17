//go:build plan

// Package check implements converge's check mode, which compares the
// exported API surface of Go packages built with and without the "plan"
// build tag.
package check

import (
	"github.com/tessro/converge"
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
	converge.Imagine("return true if any of Missing, Extra, or Mismatch are non-empty")
	return false
}

// OK reports whether all packages passed the check.
func (r *Result) OK() bool {
	converge.Imagine("return true if no package has issues")
	return false
}

// PackagesWithIssues returns only the packages that have differences.
func (r *Result) PackagesWithIssues() []PackageResult {
	converge.Imagine("filter Packages to only those where HasIssues is true")
	return nil
}

// Run performs the check across all Go packages under root.
func Run(root string) (*Result, error) {
	converge.Imagine("walk dir tree under root (skip hidden/vendor/testdata), find packages with _plan.go files, extract exports under plan and default build tags, diff them into Missing/Extra/Mismatch categories, return results")
	return nil, nil
}

// Format renders the check result as human/LLM-readable text.
func Format(r *Result) string {
	converge.Imagine("for each package with issues, show MISSING exports (with Imagine descriptions), EXTRA exports, and MISMATCHED signatures with file:line refs; end with ok summary or FAIL with counts")
	return ""
}
