package main

import (
	"path"
)

func checkpackagePossibleDowngrade() {
	trace("checkpackagePossibleDowngrade")

	m, _, pkgversion := match2(*G.pkgContext.effective_pkgname, rePkgname)
	if !m {
		return
	}

	line := *G.pkgContext.effective_pkgname_line

	change := G.globalData.lastChange[G.pkgContext.pkgpath]
	if change == nil {
		_ = G.opts.optDebugMisc && line.logDebug("No change log for package %v", G.pkgContext.pkgpath)
		return
	}

	if change.action == "Updated" {
		if pkgverCmp(pkgversion, "<", change.version) {
			line.logWarning("The package is being downgraded from %v to %v", change.version, pkgversion)
		}
	}
}

func checklinesBuildlink3Inclusion(lines []*Line) {
	lines[0].trace("checklinesbuildlink3Inclusion")

	if G.pkgContext == nil {
		return
	}

	// Collect all the included buildlink3.mk files from the file.
	includedFiles := make(map[string]*Line)
	for _, line := range lines {
		if m, _, file := match2(line.text, reMkInclude); m {
			if m, bl3 := match1(file, `^\.\./\.\./(.*)/buildlink3\.mk`); m {
				includedFiles[bl3] = line
				if G.pkgContext.bl3[bl3] == nil {
					line.logWarning("%s/buildlink3.mk is included by this file but not by the package.", bl3)
				}
			}
		}
	}

	// Print debugging messages for all buildlink3.mk files that are
	// included by the package but not by this buildlink3.mk file.
	for packageBl3, line := range G.pkgContext.bl3 {
		if includedFiles[packageBl3] == nil {
			_ = G.opts.optDebugMisc && line.logDebug("%s/buildlink3.mk is included by the package but not by the buildlink3.mk file.", packageBl3)
		}
	}
}

func checkdirPackage(pkgpath string) {
	ctx := newPkgContext(pkgpath)
	G.pkgContext = ctx
	defer func() { G.pkgContext = nil }()

	// we need to handle the Makefile first to get some variables
	lines := loadPackageMakefile(G.currentDir + "/Makefile")
	if lines == nil {
		logError(G.currentDir+"/Makefile", NO_LINES, "Cannot be read.")
		G.pkgContext = nil
		return
	}

	files := dirglob(G.currentDir)
	if *ctx.pkgdir != "." {
		files = append(files, dirglob(G.currentDir+"/"+*ctx.pkgdir)...)
	}
	if G.opts.optCheckExtra {
		files = append(files, dirglob(G.currentDir+"/"+ctx.filesdir)...)
	}
	files = append(files, dirglob(G.currentDir+"/"+ctx.patchdir)...)
	if ctx.distinfoFile != "distinfo" && ctx.distinfoFile != "./distinfo" {
		files = append(files, G.currentDir+"/"+ctx.distinfoFile)
	}
	haveDistinfo := false
	havePatches := false

	// Determine the used variables before checking any of the Makefile fragments.
	for _, fname := range files {
		if (hasPrefix(path.Base(fname), "Makefile.") || hasSuffix(fname, ".mk")) &&
			!match0(fname, `patch-`) &&
			!match0(fname, `${pkgdir}/`) &&
			!match0(fname, `${filesdir}/`) {
			if lines, err := loadLines(fname, true); err == nil && lines != nil {
				parselinesMk(lines)
				determineUsedVariables(lines)
			}
		}
	}

	for _, fname := range files {
		if fname == G.currentDir+"/Makefile" {
			_ = G.opts.optCheckMakefile && checkfilePackageMakefile(fname, lines)
		} else {
			checkfile(fname)
		}
		if match0(fname, `/patches/patch-*$`) {
			havePatches = true
		} else if hasSuffix(fname, "/distinfo") {
			haveDistinfo = true
		}
	}

	if G.opts.optCheckDistinfo && G.opts.optCheckPatches {
		if havePatches && !haveDistinfo {
			logWarning(G.currentDir+"/"+ctx.distinfoFile, NO_LINES, "File not found. Please run \"%s makepatchsum\".", confMake)
		}
	}

	if !isEmptyDir(G.currentDir + "/scripts") {
		logWarning(G.currentDir+"/scripts", NO_LINES, "This directory and its contents are deprecated! Please call the script(s) explicitly from the corresponding target(s) in the pkg's Makefile.")
	}
}
