package main

import (
	"io"
)

type GlobalVars struct {
	curPkgsrcdir *string // The pkgsrc directory, relative to the directory that is currently checked.
	opts         CmdOpts
	globalData   GlobalData
	pkgContext   *PkgContext
	mkContext    *MkContext

	currentDir       string // The currently checked directory, relative to the cwd
	isWip            bool   // Is the current directory from pkgsrc-wip?
	isInfrastructure bool   // Is the currently checked item from the pkgsrc infrastructure?

	ipcDistinfo     map[string]*Hash // Maps "alg:fname" => "checksum".
	ipcUsedLicenses map[string]bool  // Maps "license name" => true
	todo            []string         // The list of directory entries that still need to be checked. Mostly relevant with --recursive.

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
