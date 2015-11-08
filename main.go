package main

import (
	"fmt"
	"os"
)

func main() {
	G = &GlobalVarsType{}
	defer func() { G = nil }()

	G.opts = ParseCommandLine(os.Args)
	if G.opts.optPrintVersion {
		fmt.Printf("%s\n", confVersion)
		os.Exit(0)
	}

	G.globalData.Initialize(findPkgsrcTopdir())

	G.todo = append(G.todo, G.opts.args...)
	if len(G.todo) == 0 {
		G.todo = append(G.todo, ".")
	}

	for len(G.todo) != 0 {
		item := G.todo[0]
		G.todo = G.todo[1:]
		checkItem(item)
	}
	if G.ipcCheckingRootRecursively {
		checkUnusedLicenses()
	}
	printSummary()
	if G.errors != 0 {
		os.Exit(1)
	}
}
