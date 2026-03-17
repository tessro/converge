//go:build plan

// Package sig extracts exported declarations from Go packages
// under different build tag configurations.
package sig

import "github.com/tessro/converge"

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
func (e Export) Key() string {
	converge.Imagine("for methods, return \"func (\" + Receiver + \").\" + Name; otherwise return Kind + \" \" + Name")
	return ""
}

// Extract returns all exported declarations from Go files in dir,
// using the given build tags to determine which files are included.
func Extract(dir string, tags []string) ([]Export, error) {
	converge.Imagine("use go/build.Context with the given tags to select Go files, parse each with go/ast, extract all exported func/method/type/var/const declarations, detect converge.Imagine descriptions in function bodies, return sorted by Key")
	return nil, nil
}

// HasPlanFiles reports whether dir contains any _plan.go files.
func HasPlanFiles(dir string) (bool, error) {
	converge.Imagine("glob for *_plan.go in dir and report whether any matched")
	return false, nil
}
