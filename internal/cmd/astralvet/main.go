// Command astralvet runs the module's codec lint over a source tree and exits non-zero
// when it reports anything.
//
//	go run ./internal/cmd/astralvet .
//
// The check itself, and why it is drawn where it is, lives in internal/codecguard.
package main

import (
	"flag"
	"fmt"
	"go/token"
	"os"

	"github.com/astralp2p/astral-go/internal/codecguard"
)

func main() {
	flag.Parse()

	root := "."
	if flag.NArg() > 0 {
		root = flag.Arg(0)
	}

	found, err := codecguard.CheckTree(token.NewFileSet(), root)
	if err != nil {
		fmt.Fprintln(os.Stderr, "astralvet:", err)
		os.Exit(2)
	}

	for _, f := range found {
		fmt.Println(f)
	}
	if len(found) > 0 {
		fmt.Fprintf(os.Stderr, "astralvet: %d finding(s)\n", len(found))
		os.Exit(1)
	}
}
