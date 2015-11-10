package main

import (
	"os"
	"runtime"
)

func notImplemented(funcname string) {
	logError(NO_FILE, NO_LINES, "not implemented: %s", funcname)
	if G.opts.optDebugUnchecked {
		bytes := make([]byte, 4096)
		if n := runtime.Stack(bytes, false); n < len(bytes) {
			os.Stdout.Write(bytes[:n])
		}
		os.Stdout.Write([]byte("\n"))
	}
}

func checklineMkCond(line *Line, args string) {
	notImplemented("checklineMkCond")
}
func loadDocChanges(fname string) {
	notImplemented("loadDocChanges")
}
