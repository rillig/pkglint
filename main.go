package main

import (
	"fmt"
	"os"
)

func main() {
	pkgsrcdir := findPkgsrcTopdir()
	GlobalVars.opts = ParseCommandLine(os.Args)
	if GlobalVars.opts.optPrintVersion {
		fmt.Printf("%s\n", confVersion)
		os.Exit(0)
	}

	GlobalVars.cwdPkgsrcdir = &pkgsrcdir
	GlobalVars.globalData.Initialize(pkgsrcdir)
	initacls()

	GlobalVars.todo = append(GlobalVars.todo, GlobalVars.opts.args...)
	if len(GlobalVars.todo) == 0 {
		GlobalVars.todo = append(GlobalVars.todo, ".")
	}

	for len(GlobalVars.todo) != 0 {
		item := GlobalVars.todo[0]
		GlobalVars.todo = GlobalVars.todo[1:]
		checkItem(item)
	}
	if GlobalVars.ipcCheckingRootRecursively {
		checkUnusedLicenses()
	}
	printSummary()
	if GlobalVars.errors != 0 {
		os.Exit(1)
	}
}
