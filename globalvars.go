package main

type GlobalVarsType struct {
	errors       int
	warnings     int
	cwdPkgsrcdir *string // The pkgsrc directory, relative to the current working directory of pkglint.
	curPkgsrcdir *string // The pkgsrc directory, relative to the directory that is currently checked.
	opts         *CmdOpts
	globalData   GlobalData
	pkgContext   *PkgContext
	mkContext    *MkContext

	currentDir string // The currently checked directory, relative to the cwd
	isWip      bool   // Is the current directory from pkgsrc-wip?
	isInternal bool   // Is the currently checked item from the pkgsrc infrastructure?

	ipcDistinfo                map[string]string // Maps "alg:fname" => "checksum".
	ipcUsedLicenses            map[string]bool   // asdf
	ipcCheckingRootRecursively bool              // Only in this case is ipcUsedLicenses filled.
	todo                       []string          // The list of directory entries that still need to be checked. Mostly relevant with --recursive.
}

var G = &GlobalVarsType{}

const confMake = "@BMAKE@"
const confDatadir = "@DATADIR@"
const confVersion = "@VERSION@"
