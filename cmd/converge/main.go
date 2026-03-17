// Command converge checks alignment between plan files and their
// implementations in Go packages.
//
// Usage:
//
//	converge check [dir]     Compare plan and impl export signatures
//	converge lint [dir]      Check converge conventions
//	converge lint -fix [dir] Auto-fix convention issues
package main

import (
	"flag"
	"fmt"
	"os"

	"github.com/tessro/converge/internal/check"
	"github.com/tessro/converge/internal/lint"
)

func main() {
	if len(os.Args) < 2 {
		usage()
		os.Exit(2)
	}

	switch os.Args[1] {
	case "check":
		os.Exit(runCheck(os.Args[2:]))
	case "lint":
		os.Exit(runLint(os.Args[2:]))
	case "help", "-h", "--help":
		usage()
		os.Exit(0)
	default:
		fmt.Fprintf(os.Stderr, "converge: unknown command %q\n\n", os.Args[1])
		usage()
		os.Exit(2)
	}
}

func runCheck(args []string) int {
	fs := flag.NewFlagSet("check", flag.ExitOnError)
	fs.Usage = func() {
		fmt.Fprintf(os.Stderr, "Usage: converge check [dir]\n\n")
		fmt.Fprintf(os.Stderr, "Compare exported signatures between plan and impl builds.\n")
		fmt.Fprintf(os.Stderr, "Exits 0 if they match exactly, 1 if they differ.\n\n")
		fs.PrintDefaults()
	}
	fs.Parse(args)

	dir := "."
	if fs.NArg() > 0 {
		dir = fs.Arg(0)
	}

	result, err := check.Run(dir)
	if err != nil {
		fmt.Fprintf(os.Stderr, "converge check: %v\n", err)
		return 2
	}

	fmt.Print(check.Format(result))

	if !result.OK() {
		return 1
	}
	return 0
}

func runLint(args []string) int {
	fs := flag.NewFlagSet("lint", flag.ExitOnError)
	fix := fs.Bool("fix", false, "auto-fix issues where possible")
	fs.Usage = func() {
		fmt.Fprintf(os.Stderr, "Usage: converge lint [-fix] [dir]\n\n")
		fmt.Fprintf(os.Stderr, "Check that converge conventions are followed.\n\n")
		fs.PrintDefaults()
	}
	fs.Parse(args)

	dir := "."
	if fs.NArg() > 0 {
		dir = fs.Arg(0)
	}

	result, err := lint.Run(dir, *fix)
	if err != nil {
		fmt.Fprintf(os.Stderr, "converge lint: %v\n", err)
		return 2
	}

	fmt.Print(lint.Format(result))

	if result.HasErrors() {
		return 1
	}
	return 0
}

func usage() {
	fmt.Fprintf(os.Stderr, `converge — continuous alignment between humans and agents

Usage:
  converge <command> [flags] [dir]

Commands:
  check    Compare exported signatures between plan and impl builds
  lint     Check that converge conventions are followed

Run 'converge <command> -h' for command-specific help.
`)
}
