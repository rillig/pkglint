package main

import (
	"path"
	"regexp"
	"strings"
)

func checkpackagePossibleDowngrade() {
	defer tracecall("checkpackagePossibleDowngrade")()

	if G.pkgContext.effectivePkgname == nil {
		return
	}
	m, _, pkgversion := match2(*G.pkgContext.effectivePkgname, rePkgname)
	if !m {
		return
	}

	line := *G.pkgContext.effectivePkgnameLine

	change := G.globalData.lastChange[G.pkgContext.pkgpath]
	if change == nil {
		_ = G.opts.optDebugMisc && line.logDebug("No change log for package %v", G.pkgContext.pkgpath)
		return
	}

	if change.action == "Updated" {
		if pkgverCmp(pkgversion, change.version) < 0 {
			line.logWarning("The package is being downgraded from %v to %v", change.version, pkgversion)
		}
	}
}

func checklinesBuildlink3Inclusion(lines []*Line) {
	defer lines[0].tracecall("checklinesbuildlink3Inclusion")()

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
	trace("G.pkgContext set")
	defer func() { G.pkgContext = nil; trace("G.pkgContext unset") }()

	// we need to handle the Makefile first to get some variables
	lines := loadPackageMakefile(G.currentDir + "/Makefile")
	if lines == nil {
		logError(G.currentDir+"/Makefile", NO_LINES, "Cannot be read.")
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
			if G.opts.optCheckMakefile {
				checkperms(fname)
				checkfilePackageMakefile(fname, lines)
			}
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

func checkfilePackageMakefile(fname string, lines []*Line) {
	defer tracecall("checkfilePackageMakefile", fname, len(lines))()

	vardef := G.pkgContext.vardef
	if vardef["PLIST_SRC"] == nil &&
		vardef["GENERATE_PLIST"] == nil &&
		vardef["META_PACKAGE"] == nil &&
		G.pkgContext.pkgdir != nil {
		if dir := G.currentDir + "/" + *G.pkgContext.pkgdir; !fileExists(dir+"/PLIST") && !fileExists(dir+"/PLIST.common") {
			logWarning(fname, NO_LINES, "Neither PLIST nor PLIST.common exist, and PLIST_SRC is unset. Are you sure PLIST handling is ok?")
		}

		if (vardef["NO_CHECKSUM"] != nil || vardef["META_PACKAGE"] != nil) && isEmptyDir(G.currentDir+"/"+G.pkgContext.patchdir) {
			if distinfoFile := G.currentDir + "/" + G.pkgContext.distinfoFile; fileExists(distinfoFile) {
				logWarning(distinfoFile, NO_LINES, "This file should not exist if NO_CHECKSUM or META_PACKAGE is set.")
			}
		} else {
			if distinfoFile := G.currentDir + "/" + G.pkgContext.distinfoFile; !fileExists(distinfoFile) {
				logWarning(distinfoFile, NO_LINES, "File not found. Please run \"%s makesum\".", confMake)
			}
		}

		if vardef["REPLACE_PERL"] != nil && vardef["NO_CONFIGURE"] != nil {
			vardef["REPLACE_PERL"].logWarning("REPLACE_PERL is ignored when ...")
			vardef["NO_CONFIGURE"].logWarning("... NO_CONFIGURE is set.")
		}

		if vardef["LICENSE"] == nil {
			logError(fname, NO_LINES, "Each package must define its LICENSE.")
		}

		if vardef["GNU_CONFIGURE"] != nil && vardef["USE_LANGUAGES"] != nil {
			languagesLine := vardef["USE_LANGUAGES"]
			value := languagesLine.extra["value"].(string)

			if languagesLine.extra["comment"] != nil && match0(languagesLine.extra["comment"].(string), `(?-i)\b(?:c|empty|none)\b`) {
				// Don't emit a warning, since the comment
				// probably contains a statement that C is
				// really not needed.

			} else if !match0(value, `(?:^|\s+)(?:c|c99|objc)(?:\s+|$)`) {
				vardef["GNU_CONFIGURE"].logWarning("GNU_CONFIGURE almost always needs a C compiler, ...")
				languagesLine.logWarning("... but \"c\" is not added to USE_LANGUAGES.")
			}
		}

		distnameLine := vardef["DISTNAME"]
		pkgnameLine := vardef["PKGNAME"]

		distname := ""
		if distnameLine != nil {
			distname = distnameLine.extra["value"].(string)
		}
		pkgname := ""
		if pkgnameLine != nil {
			pkgname = pkgnameLine.extra["value"].(string)
		}

		if distname != "" && pkgname != "" {
			pkgname = pkgnameFromDistname(pkgname, distname)
		}

		if pkgname != "" && pkgname == distname {
			pkgnameLine.logNote("PKGNAME is ${DISTNAME} by default. You probably don't need to define PKGNAME.")
		}

		if pkgname == "" && distname != "" && !match0(distname, reUnresolvedVar) && !match0(distname, rePkgname) {
			distnameLine.logWarning("As DISTNAME is not a valid package name, please define the PKGNAME explicitly.")
		}

		if G.pkgContext.effectivePkgnameLine != nil {
			_ = G.opts.optDebugMisc && G.pkgContext.effectivePkgnameLine.logDebug("Effective name=%q base=%q version=%q",
				G.pkgContext.effectivePkgname, G.pkgContext.effectivePkgbase, G.pkgContext.effectivePkgversion)
		}

		checkpackagePossibleDowngrade()

		if vardef["COMMENT"] == nil {
			logWarning(fname, NO_LINES, "No COMMENT given.")
		}

		if vardef["USE_IMAKE"] != nil && vardef["USE_X11"] != nil {
			vardef["USE_IMAKE"].logNote("USE_IMAKE makes ...")
			vardef["USE_X11"].logNote("... USE_X11 superfluous.")
		}

		if G.pkgContext.effectivePkgbase != nil {
			for _, sugg := range G.globalData.suggestedUpdates {
				if *G.pkgContext.effectivePkgbase != sugg.pkgname {
					continue
				}

				suggver, comment := sugg.version, sugg.comment
				if comment != "" {
					comment = " (" + comment + ")"
				}

				pkgnameLine := G.pkgContext.effectivePkgnameLine
				cmp := pkgverCmp(*G.pkgContext.effectivePkgversion, suggver)
				switch {
				case cmp < 0:
					pkgnameLine.logWarning("This package should be updated to %s%s", sugg.version, comment)
					pkgnameLine.explainWarning(
						"The wishlist for package updates in doc/TODO mentions that a newer",
						"version of this package is available.")
				case cmp > 0:
					pkgnameLine.logNote("This package is newer than the update request to ${suggver}${comment}.")
				default:
					pkgnameLine.logNote("The update request to ${suggver} from doc/TODO${comment} has been done.")
				}
			}
		}

		checklinesMk(lines)
		checklinesPackageMakefileVarorder(lines)
		autofix(lines)
	}
}

func pkgnameFromDistname(pkgname, distname string) string {
	pkgname = strings.Replace(pkgname, "${DISTNAME}", distname, -1)

	if m, before, sep, subst, after := match4(pkgname, `^(.*)\$\{DISTNAME:S(.)([^\\}:]+)\}(.*)$`); m {
		qsep := regexp.QuoteMeta(sep)
		if m, left, from, right, to, mod := match5(subst, `^(\^?)([^:]*)(\$?)`+qsep+`([^:]*)`+qsep+`(g?)$`); m {
			newPkgname := before + mkopSubst(distname, left != "", from, right != "", to, mod != "") + after
			_ = G.opts.optDebugMisc && G.pkgContext.vardef["PKGNAME"].logDebug("pkgnameFromDistname %q => %q", pkgname, newPkgname)
			pkgname = newPkgname
		}
	}
	return pkgname
}
