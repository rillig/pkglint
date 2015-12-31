package main

import (
	"path"
	"regexp"
	"strconv"
	"strings"
)

// Package contains data for the pkgsrc package that is currently checked.
type Package struct {
	pkgpath              string  // e.g. "category/pkgdir"
	pkgdir               string  // PKGDIR from the package Makefile
	filesdir             string  // FILESDIR from the package Makefile
	patchdir             string  // PATCHDIR from the package Makefile
	distinfoFile         string  // DISTINFO_FILE from the package Makefile
	effectivePkgname     string  // PKGNAME or DISTNAME from the package Makefile, including nb13
	effectivePkgbase     string  // The effective PKGNAME without the version
	effectivePkgversion  string  // The version part of the effective PKGNAME, excluding nb13
	effectivePkgnameLine *MkLine // The origin of the three effective_* values
	seenBsdPrefsMk       bool    // Has bsd.prefs.mk already been included?

	vardef             map[string]*MkLine // (varname, varcanon) => line
	varuse             map[string]*MkLine // (varname, varcanon) => line
	bl3                map[string]*Line   // buildlink3.mk name => line; contains only buildlink3.mk files that are directly included.
	plistSubstCond     map[string]bool    // varname => true; list of all variables that are used as conditionals (@comment or nothing) in PLISTs.
	included           map[string]*Line   // fname => line
	seenMakefileCommon bool               // Does the package have any .includes?
}

func NewPackage(pkgpath string) *Package {
	pkg := &Package{
		pkgpath:        pkgpath,
		vardef:         make(map[string]*MkLine),
		varuse:         make(map[string]*MkLine),
		bl3:            make(map[string]*Line),
		plistSubstCond: make(map[string]bool),
		included:       make(map[string]*Line),
	}
	for varname, line := range G.globalData.userDefinedVars {
		pkg.vardef[varname] = line
	}
	return pkg
}

func (pkg *Package) defineVar(mkline *MkLine, varname string) {
	if pkg.vardef[varname] == nil {
		pkg.vardef[varname] = mkline
	}
	varcanon := varnameCanon(varname)
	if pkg.vardef[varcanon] == nil {
		pkg.vardef[varcanon] = mkline
	}
}

func (pkg *Package) varValue(varname string) (string, bool) {
	if mkline := pkg.vardef[varname]; mkline != nil {
		return mkline.Value(), true
	}
	return "", false
}

func (pkg *Package) checkPossibleDowngrade() {
	if G.opts.DebugTrace {
		defer tracecall0("Package.checkPossibleDowngrade")()
	}

	m, _, pkgversion := match2(pkg.effectivePkgname, rePkgname)
	if !m {
		return
	}

	mkline := pkg.effectivePkgnameLine

	change := G.globalData.lastChange[pkg.pkgpath]
	if change == nil {
		if G.opts.DebugMisc {
			mkline.debug1("No change log for package %q", pkg.pkgpath)
		}
		return
	}

	if change.action == "Updated" {
		if pkgverCmp(pkgversion, change.version) < 0 {
			mkline.warn2("The package is being downgraded from %s to %s", change.version, pkgversion)
		}
	}
}

func (pkg *Package) checklinesBuildlink3Inclusion(mklines *MkLines) {
	if G.opts.DebugTrace {
		defer tracecall0("checklinesbuildlink3Inclusion")()
	}

	// Collect all the included buildlink3.mk files from the file.
	includedFiles := make(map[string]*MkLine)
	for _, mkline := range mklines.mklines {
		if mkline.IsInclude() {
			file := mkline.Includefile()
			if m, bl3 := match1(file, `^\.\./\.\./(.*)/buildlink3\.mk`); m {
				includedFiles[bl3] = mkline
				if pkg.bl3[bl3] == nil {
					mkline.warn1("%s/buildlink3.mk is included by this file but not by the package.", bl3)
				}
			}
		}
	}

	if G.opts.DebugMisc {
		for packageBl3, line := range pkg.bl3 {
			if includedFiles[packageBl3] == nil {
				line.debug1("%s/buildlink3.mk is included by the package but not by the buildlink3.mk file.", packageBl3)
			}
		}
	}
}

func checkdirPackage(pkgpath string) {
	if G.opts.DebugTrace {
		defer tracecall1("checkdirPackage", pkgpath)()
	}

	G.pkg = NewPackage(pkgpath)
	defer func() { G.pkg = nil }()
	pkg := G.pkg

	// we need to handle the Makefile first to get some variables
	lines := pkg.loadPackageMakefile(G.currentDir + "/Makefile")
	if lines == nil {
		return
	}

	files := dirglob(G.currentDir)
	if pkg.pkgdir != "." {
		files = append(files, dirglob(G.currentDir+"/"+pkg.pkgdir)...)
	}
	if G.opts.CheckExtra {
		files = append(files, dirglob(G.currentDir+"/"+pkg.filesdir)...)
	}
	files = append(files, dirglob(G.currentDir+"/"+pkg.patchdir)...)
	if pkg.distinfoFile != "distinfo" && pkg.distinfoFile != "./distinfo" {
		files = append(files, G.currentDir+"/"+pkg.distinfoFile)
	}
	haveDistinfo := false
	havePatches := false

	// Determine the used variables before checking any of the Makefile fragments.
	for _, fname := range files {
		if (hasPrefix(path.Base(fname), "Makefile.") || hasSuffix(fname, ".mk")) &&
			!matches(fname, `patch-`) &&
			!strings.Contains(fname, pkg.pkgdir+"/") &&
			!strings.Contains(fname, pkg.filesdir+"/") {
			if lines, err := readLines(fname, true); err == nil && lines != nil {
				NewMkLines(lines).determineUsedVariables()
			}
		}
	}

	for _, fname := range files {
		if fname == G.currentDir+"/Makefile" {
			if G.opts.CheckMakefile {
				pkg.checkfilePackageMakefile(fname, lines)
			}
		} else {
			checkfile(fname)
		}
		if strings.Contains(fname, "/patches/patch-") {
			havePatches = true
		} else if hasSuffix(fname, "/distinfo") {
			haveDistinfo = true
		}
	}

	if G.opts.CheckDistinfo && G.opts.CheckPatches {
		if havePatches && !haveDistinfo {
			warnf(G.currentDir+"/"+pkg.distinfoFile, noLines, "File not found. Please run \"%s makepatchsum\".", confMake)
		}
	}

	if !isEmptyDir(G.currentDir + "/scripts") {
		warnf(G.currentDir+"/scripts", noLines, "This directory and its contents are deprecated! Please call the script(s) explicitly from the corresponding target(s) in the pkg's Makefile.")
	}
}

func (pkg *Package) checkfilePackageMakefile(fname string, mklines *MkLines) {
	if G.opts.DebugTrace {
		defer tracecall1("Package.checkfilePackageMakefile", fname)()
	}

	vardef := pkg.vardef
	if vardef["PLIST_SRC"] == nil &&
		vardef["GENERATE_PLIST"] == nil &&
		vardef["META_PACKAGE"] == nil &&
		!fileExists(G.currentDir+"/"+pkg.pkgdir+"/PLIST") &&
		!fileExists(G.currentDir+"/"+pkg.pkgdir+"/PLIST.common") {
		warnf(fname, noLines, "Neither PLIST nor PLIST.common exist, and PLIST_SRC is unset. Are you sure PLIST handling is ok?")
	}

	if (vardef["NO_CHECKSUM"] != nil || vardef["META_PACKAGE"] != nil) && isEmptyDir(G.currentDir+"/"+pkg.patchdir) {
		if distinfoFile := G.currentDir + "/" + pkg.distinfoFile; fileExists(distinfoFile) {
			warnf(distinfoFile, noLines, "This file should not exist if NO_CHECKSUM or META_PACKAGE is set.")
		}
	} else {
		if distinfoFile := G.currentDir + "/" + pkg.distinfoFile; !containsVarRef(distinfoFile) && !fileExists(distinfoFile) {
			warnf(distinfoFile, noLines, "File not found. Please run \"%s makesum\".", confMake)
		}
	}

	if vardef["REPLACE_PERL"] != nil && vardef["NO_CONFIGURE"] != nil {
		vardef["REPLACE_PERL"].warn0("REPLACE_PERL is ignored when ...")
		vardef["NO_CONFIGURE"].warn0("... NO_CONFIGURE is set.")
	}

	if vardef["LICENSE"] == nil {
		errorf(fname, noLines, "Each package must define its LICENSE.")
	}

	if vardef["GNU_CONFIGURE"] != nil && vardef["USE_LANGUAGES"] != nil {
		languagesLine := vardef["USE_LANGUAGES"]

		if matches(languagesLine.Comment(), `(?-i)\b(?:c|empty|none)\b`) {
			// Don't emit a warning, since the comment
			// probably contains a statement that C is
			// really not needed.

		} else if !matches(languagesLine.Value(), `(?:^|\s+)(?:c|c99|objc)(?:\s+|$)`) {
			vardef["GNU_CONFIGURE"].warn0("GNU_CONFIGURE almost always needs a C compiler, ...")
			languagesLine.warn0("... but \"c\" is not added to USE_LANGUAGES.")
		}
	}

	pkg.determineEffectivePkgVars()
	pkg.checkPossibleDowngrade()

	if vardef["COMMENT"] == nil {
		warnf(fname, noLines, "No COMMENT given.")
	}

	if vardef["USE_IMAKE"] != nil && vardef["USE_X11"] != nil {
		vardef["USE_IMAKE"].note0("USE_IMAKE makes ...")
		vardef["USE_X11"].note0("... USE_X11 superfluous.")
	}

	pkg.checkUpdate()
	mklines.check()
	pkg.ChecklinesPackageMakefileVarorder(mklines)
	saveAutofixChanges(mklines.lines)
}

func (pkg *Package) getNbpart() string {
	line := pkg.vardef["PKGREVISION"]
	if line == nil {
		return ""
	}
	pkgrevision := line.Value()
	if rev, err := strconv.Atoi(pkgrevision); err == nil {
		return "nb" + strconv.Itoa(rev)
	}
	return ""
}

func (pkg *Package) determineEffectivePkgVars() {
	distnameLine := pkg.vardef["DISTNAME"]
	pkgnameLine := pkg.vardef["PKGNAME"]

	distname := ""
	if distnameLine != nil {
		distname = distnameLine.Value()
	}
	pkgname := ""
	if pkgnameLine != nil {
		pkgname = pkgnameLine.Value()
	}

	if distname != "" && pkgname != "" {
		pkgname = pkg.pkgnameFromDistname(pkgname, distname)
	}

	if pkgname != "" && pkgname == distname && pkgnameLine.Comment() == "" {
		pkgnameLine.note0("PKGNAME is ${DISTNAME} by default. You probably don't need to define PKGNAME.")
	}

	if pkgname == "" && distname != "" && !containsVarRef(distname) && !matches(distname, rePkgname) {
		distnameLine.warn0("As DISTNAME is not a valid package name, please define the PKGNAME explicitly.")
	}

	if pkgname != "" && !containsVarRef(pkgname) {
		if m, m1, m2 := match2(pkgname, rePkgname); m {
			pkg.effectivePkgname = pkgname + pkg.getNbpart()
			pkg.effectivePkgnameLine = pkgnameLine
			pkg.effectivePkgbase = m1
			pkg.effectivePkgversion = m2
		}
	}
	if pkg.effectivePkgnameLine == nil && distname != "" && !containsVarRef(distname) {
		if m, m1, m2 := match2(distname, rePkgname); m {
			pkg.effectivePkgname = distname + pkg.getNbpart()
			pkg.effectivePkgnameLine = distnameLine
			pkg.effectivePkgbase = m1
			pkg.effectivePkgversion = m2
		}
	}
	if pkg.effectivePkgnameLine != nil {
		if G.opts.DebugMisc {
			pkg.effectivePkgnameLine.line.debugf("Effective name=%q base=%q version=%q",
				pkg.effectivePkgname, pkg.effectivePkgbase, pkg.effectivePkgversion)
		}
	}
}

func (pkg *Package) pkgnameFromDistname(pkgname, distname string) string {
	pkgname = strings.Replace(pkgname, "${DISTNAME}", distname, -1)

	if m, before, sep, subst, after := match4(pkgname, `^(.*)\$\{DISTNAME:S(.)([^\\}:]+)\}(.*)$`); m {
		qsep := regexp.QuoteMeta(sep)
		if m, left, from, right, to, mod := match5(subst, `^(\^?)([^:]*)(\$?)`+qsep+`([^:]*)`+qsep+`(g?)$`); m {
			newPkgname := before + mkopSubst(distname, left != "", from, right != "", to, mod != "") + after
			if G.opts.DebugMisc {
				pkg.vardef["PKGNAME"].debug2("pkgnameFromDistname %q => %q", pkgname, newPkgname)
			}
			pkgname = newPkgname
		}
	}
	return pkgname
}

func (pkg *Package) checkUpdate() {
	if pkg.effectivePkgbase != "" {
		for _, sugg := range G.globalData.getSuggestedPackageUpdates() {
			if pkg.effectivePkgbase != sugg.pkgname {
				continue
			}

			suggver, comment := sugg.version, sugg.comment
			if comment != "" {
				comment = " (" + comment + ")"
			}

			pkgnameLine := pkg.effectivePkgnameLine
			cmp := pkgverCmp(pkg.effectivePkgversion, suggver)
			switch {
			case cmp < 0:
				pkgnameLine.warn2("This package should be updated to %s%s.", sugg.version, comment)
				explain2(
					"The wishlist for package updates in doc/TODO mentions that a newer",
					"version of this package is available.")
			case cmp > 0:
				pkgnameLine.note2("This package is newer than the update request to %s%s.", suggver, comment)
			default:
				pkgnameLine.note2("The update request to %s from doc/TODO%s has been done.", suggver, comment)
			}
		}
	}
}

func (pkg *Package) ChecklinesPackageMakefileVarorder(mklines *MkLines) {
	if G.opts.DebugTrace {
		defer tracecall0("ChecklinesPackageMakefileVarorder")()
	}

	if !G.opts.WarnOrder || pkg.seenMakefileCommon {
		return
	}

	type OccCount uint8
	const (
		once OccCount = iota
		optional
		many
	)
	type OccDef struct {
		varname string
		count   OccCount
	}
	type OccGroup struct {
		name  string
		count OccCount
		occ   []OccDef
	}

	var sections = []OccGroup{
		{"Initial comments", once,
			[]OccDef{},
		},
		{"Unsorted stuff, part 1", once,
			[]OccDef{
				{"DISTNAME", optional},
				{"PKGNAME", optional},
				{"PKGREVISION", optional},
				{"CATEGORIES", once},
				{"MASTER_SITES", optional},
				{"DIST_SUBDIR", optional},
				{"EXTRACT_SUFX", optional},
				{"DISTFILES", many},
				{"SITES.*", many},
			},
		},
		{"Distribution patches", optional,
			[]OccDef{
				{"PATCH_SITES", optional}, // or once?
				{"PATCH_SITE_SUBDIR", optional},
				{"PATCHFILES", optional}, // or once?
				{"PATCH_DIST_ARGS", optional},
				{"PATCH_DIST_STRIP", optional},
				{"PATCH_DIST_CAT", optional},
			},
		},
		{"Unsorted stuff, part 2", once,
			[]OccDef{
				{"MAINTAINER", optional},
				{"OWNER", optional},
				{"HOMEPAGE", optional},
				{"COMMENT", once},
				{"LICENSE", once},
			},
		},
		{"Legal issues", optional,
			[]OccDef{
				{"LICENSE_FILE", optional},
				{"RESTRICTED", optional},
				{"NO_BIN_ON_CDROM", optional},
				{"NO_BIN_ON_FTP", optional},
				{"NO_SRC_ON_CDROM", optional},
				{"NO_SRC_ON_FTP", optional},
			},
		},
		{"Technical restrictions", optional,
			[]OccDef{
				{"BROKEN_EXCEPT_ON_PLATFORM", many},
				{"BROKEN_ON_PLATFORM", many},
				{"NOT_FOR_PLATFORM", many},
				{"ONLY_FOR_PLATFORM", many},
				{"NOT_FOR_COMPILER", many},
				{"ONLY_FOR_COMPILER", many},
				{"NOT_FOR_UNPRIVILEGED", optional},
				{"ONLY_FOR_UNPRIVILEGED", optional},
			},
		},
		{"Dependencies", optional,
			[]OccDef{
				{"BUILD_DEPENDS", many},
				{"TOOL_DEPENDS", many},
				{"DEPENDS", many},
			},
		},
	}

	lineno := 0
	sectindex := -1
	varindex := 0
	nextSection := true
	var vars []OccDef
	below := make(map[string]string)
	var belowWhat string

	// If the current section is optional but contains non-optional
	// fields, the complete section may be skipped as long as there
	// has not been a non-optional variable.
	maySkipSection := false

	// In each iteration, one of the following becomes true:
	// - new lineno > old lineno
	// - new sectindex > old sectindex
	// - new sectindex == old sectindex && new varindex > old varindex
	// - new nextSection == true && old nextSection == false
	for lineno < len(mklines.lines) {
		mkline := mklines.mklines[lineno]
		line := mklines.lines[lineno]
		text := line.Text

		if G.opts.DebugMisc {
			line.debugf("[varorder] section %d variable %d vars %v", sectindex, varindex, vars)
		}

		if nextSection {
			nextSection = false
			sectindex++
			if !(sectindex < len(sections)) {
				break
			}
			vars = sections[sectindex].occ
			maySkipSection = sections[sectindex].count == optional
			varindex = 0
		}

		switch {
		case hasPrefix(text, "#"):
			lineno++

		case mkline.IsVarassign():
			varcanon := mkline.Varcanon()

			if belowText, exists := below[varcanon]; exists {
				if belowText != "" {
					line.warn2("%s appears too late. Please put it below %s.", varcanon, belowText)
				} else {
					line.warn1("%s appears too late. It should be the very first definition.", varcanon)
				}
				lineno++
				continue
			}

			for varindex < len(vars) && varcanon != vars[varindex].varname && (vars[varindex].count != once || maySkipSection) {
				if vars[varindex].count == once {
					maySkipSection = false
				}
				below[vars[varindex].varname] = belowWhat
				varindex++
			}
			switch {
			case !(varindex < len(vars)):
				if sections[sectindex].count != optional {
					line.warn0("Empty line expected.")
				}
				nextSection = true

			case varcanon != vars[varindex].varname:
				line.warn2("Expected %s, but found %s.", vars[varindex].varname, varcanon)
				lineno++

			default:
				if vars[varindex].count != many {
					below[vars[varindex].varname] = belowWhat
					varindex++
				}
				lineno++
			}
			belowWhat = varcanon

		default:
			for varindex < len(vars) {
				if vars[varindex].count == once && !maySkipSection {
					line.warn1("%s should be set here.", vars[varindex].varname)
				}
				below[vars[varindex].varname] = belowWhat
				varindex++
			}
			nextSection = true
			if text == "" {
				belowWhat = "the previous empty line"
				lineno++
			}
		}
	}
}

func (mklines *MkLines) checkForUsedComment(relativeName string) {
	lines := mklines.lines
	if len(lines) < 3 {
		return
	}

	expected := "# used by " + relativeName
	for _, line := range lines {
		if line.Text == expected {
			return
		}
	}

	i := 0
	for i < 2 && hasPrefix(lines[i].Text, "#") {
		i++
	}

	insertLine := lines[i]
	if !insertLine.autofixInsertBefore(expected) {
		insertLine.warn1("Please add a line %q here.", expected)
		explain(
			"Since Makefile.common files usually don't have any comments and",
			"therefore not a clearly defined interface, they should at least contain",
			"references to all files that include them, so that it is easier to see",
			"what effects future changes may have.",
			"",
			"If there are more than five packages that use a Makefile.common,",
			"you should think about giving it a proper name (maybe plugin.mk) and",
			"documenting its interface.")
	}
	saveAutofixChanges(lines)
}
