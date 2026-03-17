// Package converge provides primitives for specifying function behavior
// in plan files. Plan files (_plan.go) use build tags to define the
// exported API surface alongside semantic descriptions of desired behavior.
// Code generation tools then produce implementations that must match the
// plan's exported signatures exactly.
package converge

// Imagine describes the intended behavior of a function body.
// It is used in _plan.go files as a semantic specification for
// LLM-driven code generation.
//
// Imagine is a no-op at runtime. Its value lies in the description
// string, which communicates intent to code generation tools.
//
// Example usage in a _plan.go file:
//
//	//go:build plan
//
//	package mypackage
//
//	import "github.com/tessro/converge"
//
//	func ProcessData(input []byte) (Result, error) {
//	    converge.Imagine("parse the input as JSON, validate against the schema, and return a structured Result")
//	    return Result{}, nil
//	}
func Imagine(description string) {}
