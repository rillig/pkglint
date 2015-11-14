package main

import (
	"fmt"
	"os"
)

func main() {
	G = &GlobalVars{}

	G.opts = ParseCommandLine(os.Args, os.Stdout)
	if G.opts.optPrintVersion {
		fmt.Printf("%s\n", confVersion)
		return
	}

	G.globalData.Initialize(findPkgsrcTopdir())

	G.todo = append(G.todo, G.opts.args...)
	if len(G.todo) == 0 {
		G.todo = []string{"."}
	}

	for len(G.todo) != 0 {
		item := G.todo[0]
		G.todo = G.todo[1:]
		checkItem(item)
	}

	checktoplevelUnusedLicenses()
	printSummary()
	if G.errors != 0 {
		os.Exit(1)
	}
}
