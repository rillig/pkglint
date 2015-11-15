package main

import (
	"io"
)

type GlobalVars struct {
	opts       CmdOpts
	globalData GlobalData
	pkgContext *PkgContext
	mkContext  *MkContext

	todo             []string // The items that still need to be checked.
	currentDir       string   // The currently checked directory, relative to the cwd
	curPkgsrcdir     string   // The pkgsrc directory, relative to currentDir
	isWip            bool     // Is the currently checked directory from pkgsrc-wip?
	isInfrastructure bool     // Is the currently checked item from the pkgsrc infrastructure?

	ipcDistinfo     map[string]*Hash // Maps "alg:fname" => "checksum".
	ipcUsedLicenses map[string]bool  // Maps "license name" => true

	errors     int
	warnings   int
	traceDepth int
	logOut     io.Writer
	logErr     io.Writer
}

type Hash struct {
	hash string
	line *Line
}

var G *GlobalVars
