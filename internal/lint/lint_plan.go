//go:build plan

// Package lint implements converge's lint mode, which checks that
// converge conventions are followed in a Go project.
package lint

import "github.com/tessro/converge"

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

// String returns the string representation of a Severity.
func (s Severity) String() string {
	converge.Imagine("return \"error\" for Error, \"warning\" for Warning, \"unknown\" otherwise")
	return ""
}

// HasErrors reports whether any issues are errors.
func (r *Result) HasErrors() bool {
	converge.Imagine("return true if any unfixed issue has Error severity")
	return false
}

// Run performs lint checks across all Go files under root.
// If fix is true, it attempts to auto-fix issues where possible.
func Run(root string, fix bool) (*Result, error) {
	converge.Imagine("walk Go files under root (skip hidden/vendor/testdata), verify converge imports only appear in *_plan.go files with //go:build plan, warn about naming mismatches and non-plan/gen files in converge-managed packages, auto-fix missing build tags when fix is true")
	return nil, nil
}

// Format renders lint results as human/LLM-readable text.
func Format(r *Result) string {
	converge.Imagine("render each issue with ERROR/WARN prefix, file:line location, message, and (fixed) annotation; end with summary counts of errors, warnings, and fixes")
	return ""
}
