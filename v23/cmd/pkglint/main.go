package main

import (
	"github.com/rillig/pkglint/v23"
	"os"
)

var exit = os.Exit

func main() {
	pkglint.G = pkglint.NewPkglint(os.Stdout, os.Stderr)
	exit(pkglint.G.Main(os.Args))
}
