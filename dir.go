package main

import (
	"os"
	"path"
)

func CheckDirent(fname string) {
	if G.opts.DebugTrace {
		defer tracecall1("CheckDirent", fname)()
	}

	st, err := os.Lstat(fname)
	if err != nil || !st.Mode().IsDir() && !st.Mode().IsRegular() {
		errorf(fname, noLines, "No such file or directory.")
		return
	}
	isDir := st.Mode().IsDir()
	isReg := st.Mode().IsRegular()

	G.CurrentDir = ifelseStr(isReg, path.Dir(fname), fname)
	absCurrentDir := abspath(G.CurrentDir)
	G.Wip = !G.opts.Import && matches(absCurrentDir, `/wip/|/wip$`)
	G.Infrastructure = matches(absCurrentDir, `/mk/|/mk$`)
	G.CurPkgsrcdir = findPkgsrcTopdir(G.CurrentDir)
	if G.CurPkgsrcdir == "" {
		errorf(fname, noLines, "Cannot determine the pkgsrc root directory for %q.", G.CurrentDir)
		return
	}

	switch {
	case isDir && isEmptyDir(fname):
		return
	case isReg:
		checkfile(fname)
		return
	}

	switch G.CurPkgsrcdir {
	case "../..":
		checkdirPackage(relpath(G.globalData.Pkgsrcdir, G.CurrentDir))
	case "..":
		CheckdirCategory()
	case ".":
		checkdirToplevel()
	default:
		errorf(fname, noLines, "Cannot check directories outside a pkgsrc tree.")
	}
}
