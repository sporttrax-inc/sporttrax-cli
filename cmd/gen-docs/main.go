// gen-docs regenerates the markdown command reference in docs/ from the
// cobra command definitions — the single source of truth for CLI
// documentation. Run via `make docs`; never edit docs/ by hand.
package main

import (
	"flag"
	"fmt"
	"os"

	"github.com/spf13/cobra/doc"

	"github.com/sporttrax-inc/sporttrax-cli/internal/cli"
)

func main() {
	out := flag.String("out", "docs", "output directory")
	flag.Parse()

	root := cli.Root()
	root.DisableAutoGenTag = true // no timestamps: regeneration must be diff-stable

	if err := os.MkdirAll(*out, 0o750); err != nil {
		fmt.Fprintln(os.Stderr, "error:", err)
		os.Exit(1)
	}
	if err := doc.GenMarkdownTree(root, *out); err != nil {
		fmt.Fprintln(os.Stderr, "error:", err)
		os.Exit(1)
	}
	fmt.Println("command reference generated in", *out)
}
