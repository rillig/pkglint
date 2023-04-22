package main

import (
	"github.com/rillig/pkglint/v23"
	"os"
)

var exit = os.Exit

func main() {
	exit(pkglint.G.Main(os.Stdout, os.Stderr, os.Args))
}
