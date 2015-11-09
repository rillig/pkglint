package main

import (
	"os"
	"runtime"
)

func notImplemented() {
	logError(NO_FILE, NO_LINES, "not implemented")
	if G.opts.optDebugUnchecked {
		bytes := make([]byte, 4096)
		if n := runtime.Stack(bytes, false); n < len(bytes) {
			os.Stdout.Write(bytes[:n])
		}
		os.Stdout.Write([]byte("\n"))
	}
}

func checkperms(fname string) {
	notImplemented()
}
func normalizePathname(fname string) string {
	notImplemented()
	return fname
}
func checkUnusedLicenses() {
	notImplemented()
}
func varIsDefined(varname string) bool {
	notImplemented()
	return false
}
func pkgverCmp(left, op, right string) bool {
	notImplemented()
	return false
}
func varIsUsed(varname string) bool {
	notImplemented()
	return false
}

func checklineMkCond(line *Line, args string) {
	notImplemented()
}
func checklinesBuildlink3Mk(lines []*Line, index int, pkgbase string) {
	notImplemented()
}
func loadDocChanges(fname string) {
	notImplemented()
}
