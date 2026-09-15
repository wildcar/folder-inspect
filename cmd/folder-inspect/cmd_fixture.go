package main

import (
	"flag"
	"fmt"
	"os"

	"github.com/wildcar/folder-inspect/internal/fixture"
)

func runFixture(args []string) int {
	fs := flag.NewFlagSet("fixture", flag.ContinueOnError)
	fs.SetOutput(os.Stderr)
	scale := fs.Int64("scale", 1, "divide big-file sizes by this factor (1 = realistic sizes, ~170 MB total)")
	fs.Usage = func() {
		fmt.Fprintln(os.Stderr, "Usage: folder-inspect fixture [options] <empty-folder>")
		fs.PrintDefaults()
	}
	if err := fs.Parse(args); err != nil {
		return exitUsage
	}
	if fs.NArg() != 1 {
		fs.Usage()
		return exitUsage
	}
	dir := fs.Arg(0)
	if err := fixture.Generate(dir, fixture.Options{Scale: *scale}); err != nil {
		fmt.Fprintln(os.Stderr, "fixture:", err)
		return exitError
	}
	fmt.Printf("fixture written to %s (%d files, %d empty folders)\n", dir, len(fixture.Files), len(fixture.EmptyDirs))
	return exitOK
}
