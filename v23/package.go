package pkglint

import (
	"github.com/rillig/pkglint/v23/pkgver"
	"os"
	"strconv"
	"strings"
)

// Package is the pkgsrc package that is currently checked.
//
// Most of the information is loaded first, and after loading the actual checks take place.
// This is necessary because variables in Makefiles may be used before they are defined,
// and such dependencies often span multiple files that are included indirectly.
type Package struct {
	dir     CurrPath   // The directory of the package, for resolving files
	Pkgpath PkgsrcPath // e.g. "category/pkgdir"

	Makefile *MkLines

	EffectivePkgname     string  // PKGNAME or DISTNAME from the package Makefile, including nb13, can be empty
	EffectivePkgbase     string  // EffectivePkgname without the version
	EffectivePkgversion  string  // The version part of the effective PKGNAME, excluding nb13
	EffectivePkgnameLine *MkLine // The origin of the three Effective* values
	buildlinkID          string
	optionsID            string

	Pkgdir       PackagePath  // PKGDIR from the package Makefile
	Filesdir     PackagePath  // FILESDIR from the package Makefile
	Patchdir     PackagePath  // PATCHDIR from the package Makefile
	DistinfoFile PackagePath  // DISTINFO_FILE from the package Makefile
	Plist        PlistContent // Files and directories mentioned in the PLIST files
	PlistLines   *PlistLines

	vars      Scope
	redundant *RedundantScope

	// bl3 contains the buildlink3.mk files that are included by the
	// package, including any transitively included files.
	//
	// This is later compared to those buildlink3.mk files that are
	// included by the package's own buildlink3.mk file.
	// These included files should match.
	bl3 map[PackagePath]*MkLine

	// The default dependencies from the buildlink3.mk files.
	bl3Data map[Buildlink3ID]*Buildlink3Data

	// Remembers the makefile fragments that have already been included.
	// Typical keys are "../../category/package/buildlink3.mk".
	//
	// TODO: Include files with multiple-inclusion guard only once.
	//
	// TODO: Include files without multiple-inclusion guard as often as needed.
	included IncludedMap

	// Does the package have any .includes?
	//
	// TODO: Be more precise about the purpose of this field.
	seenInclude bool

	// The identifiers for which PKG_BUILD_OPTIONS is defined.
	seenPkgbase map[string]struct{}

	// During both load() and check(), tells whether bsd.prefs.mk has
	// already been loaded directly or indirectly.
	//
	// At the end of load(), it is reset to false.
	seenPrefs bool

	// The first line of the package Makefile at which bsd.prefs.mk is
	// guaranteed to be loaded.
	prefsLine *MkLine

	// Files from .include lines that are nested inside .if.
	// They often depend on OPSYS or on the existence of files in the build environment.
	conditionalIncludes map[PackagePath]*MkLine
	// Files from .include lines that are not nested.
	// These are cross-checked with buildlink3.mk whether they are unconditional there, too.
	unconditionalIncludes map[PackagePath]*MkLine

	IgnoreMissingPatches bool // In distinfo, don't warn about patches that cannot be found.

	warnedAboutConditionalInclusion OncePerStringSlice
	checkedPolicyUpdateLimited      Once

	// Contains the basenames of the distfiles that are mentioned in distinfo,
	// for example "package-1.0.tar.gz", even if that file is in a DIST_SUBDIR.
	distinfoDistfiles map[RelPath]bool
}

func NewPackage(dir CurrPath) *Package {
	pkgpath := G.Pkgsrc.Rel(dir)

	// Package directory must be two subdirectories below the pkgsrc root.
	// As of November 2019, it is technically possible to create packages
	// on different levels, but that is not used at all. Therefore, all
	// relative directories are in the form "../../category/package".
	assert(pkgpath.Count() == 2)

	pkg := Package{
		dir:                   dir,
		Pkgpath:               pkgpath,
		Pkgdir:                ".",
		Filesdir:              "files",              // TODO: Redundant, see the vars.Fallback below.
		Patchdir:              "patches",            // TODO: Redundant, see the vars.Fallback below.
		DistinfoFile:          "${PKGDIR}/distinfo", // TODO: Redundant, see the vars.Fallback below.
		Plist:                 NewPlistContent(),
		PlistLines:            NewPlistLines(),
		vars:                  NewScope(),
		bl3:                   make(map[PackagePath]*MkLine),
		bl3Data:               make(map[Buildlink3ID]*Buildlink3Data),
		included:              NewIncludedMap(),
		seenPkgbase:           make(map[string]struct{}),
		conditionalIncludes:   make(map[PackagePath]*MkLine),
		unconditionalIncludes: make(map[PackagePath]*MkLine),
	}
	pkg.vars.DefineAll(&G.Pkgsrc.UserDefinedVars)

	pkg.vars.Fallback("PKGDIR", ".")
	pkg.vars.Fallback("DISTINFO_FILE", "${PKGDIR}/distinfo")
	pkg.vars.Fallback("FILESDIR", "files")
	pkg.vars.Fallback("PATCHDIR", "patches")
	pkg.vars.Fallback("KRB5_TYPE", "heimdal")
	pkg.vars.Fallback("PGSQL_VERSION", "95")
	pkg.vars.Fallback("EXTRACT_SUFX", ".tar.gz")

	// In reality, this is an absolute pathname. Since this variable is
	// typically used in the form ${.CURDIR}/../../somewhere, this doesn't
	// matter much.
	pkg.vars.Fallback(".CURDIR", ".")

	return &pkg
}

func (pkg *Package) Check() {
	files, mklines, allLines := pkg.load()
	if files == nil {
		return
	}
	pkg.check(files, mklines, allLines)
}

func (pkg *Package) load() ([]CurrPath, *MkLines, *MkLines) {
	// Load the package Makefile and all included files,
	// to collect all used and defined variables and similar data.
	mklines, allLines := pkg.loadPackageMakefile()
	if mklines == nil {
		return nil, nil, nil
	}

	files := pkg.File(".").ReadPaths()
	if pkg.Pkgdir != "." && pkg.Rel(pkg.File(pkg.Pkgdir)) != "." {
		files = append(files, pkg.File(pkg.Pkgdir).ReadPaths()...)
	}
	files = append(files, pkg.File(pkg.Patchdir).ReadPaths()...)
	defaultDistinfoFile := NewPackagePathString(pkg.vars.create("DISTINFO_FILE").fallback)
	if pkg.DistinfoFile != defaultDistinfoFile {
		resolved := func(p PackagePath) PkgsrcPath { return G.Pkgsrc.Rel(pkg.File(p)) }
		if resolved(pkg.DistinfoFile) != resolved(defaultDistinfoFile) {
			files = append(files, pkg.File(pkg.DistinfoFile))
		}
	}

	isRelevantMk := func(filename CurrPath, basename RelPath) bool {
		if !hasPrefix(basename.String(), "Makefile.") && !filename.HasSuffixText(".mk") {
			return false
		}
		if filename.Dir().HasBase("patches") {
			return false
		}
		if pkg.Pkgdir == "." {
			return true
		}
		return !filename.ContainsPath(pkg.Pkgdir.AsPath())
	}

	// Determine the used variables and PLIST directories before checking any of the makefile fragments.
	// TODO: Why is this code necessary? What effect does it have?
	pkg.collectConditionalIncludes(mklines)
	for _, filename := range files {
		basename := filename.Base()
		if isRelevantMk(filename, basename) {
			fragmentMklines := LoadMk(filename, pkg, MustSucceed)
			pkg.collectConditionalIncludes(fragmentMklines)
			pkg.loadBuildlink3Pkgbase(filename, fragmentMklines)
		}
		if basename.HasPrefixText("PLIST") {
			pkg.loadPlistDirs(filename)
		}
	}

	pkg.seenPrefs = false
	return files, mklines, allLines
}

func (pkg *Package) loadBuildlink3Pkgbase(filename CurrPath, fragmentMklines *MkLines) {
	if !filename.HasBase("buildlink3.mk") {
		return
	}
	fragmentMklines.ForEach(func(mkline *MkLine) {
		if mkline.IsVarassign() && mkline.Varname() == "pkgbase" {
			pkg.seenPkgbase[mkline.Value()] = struct{}{}
		}
	})
}

func (pkg *Package) loadPackageMakefile() (*MkLines, *MkLines) {
	filename := pkg.File("Makefile")
	if trace.Tracing {
		defer trace.Call(filename)()
	}

	mainLines := LoadMk(filename, pkg, NotEmpty|LogErrors)
	if mainLines == nil {
		return nil, nil
	}
	pkg.Makefile = mainLines

	G.checkRegCvsSubst(filename)
	allLines := NewMkLines(NewLines("", nil), pkg, &pkg.vars)
	if !pkg.parse(mainLines, allLines, "", true) {
		return nil, nil
	}

	// See mk/bsd.hacks.mk, which is included by mk/bsd.pkg.mk.
	hacks := LoadMk(pkg.File("${PKGDIR}/hacks.mk"), pkg, NotEmpty)
	if hacks != nil {
		_ = pkg.parse(hacks, allLines, "", false)
	}

	// TODO: Is this still necessary? This code is 20 years old and was introduced
	//  when pkglint loaded the package Makefile including all included files into
	//  a single string. Maybe it makes sense to print the file inclusion hierarchy
	//  to quickly see files that cannot be included because of unresolved variables.
	if G.DumpMakefile {
		G.Logger.out.WriteLine("Whole Makefile (with all included files) follows:")
		for _, line := range allLines.lines.Lines {
			G.Logger.out.WriteLine(line.String())
		}
	}

	// See mk/tools/cmake.mk
	if pkg.vars.IsDefined("USE_CMAKE") {
		allLines.Tools.def("cmake", "", false, AtRunTime, nil)
		allLines.Tools.def("cpack", "", false, AtRunTime, nil)
	}

	allLines.collectUsedVariables()

	pkg.Pkgdir = NewPackagePathString(pkg.vars.LastValue("PKGDIR"))
	pkg.DistinfoFile = NewPackagePathString(pkg.vars.LastValue("DISTINFO_FILE"))
	pkg.Filesdir = NewPackagePathString(pkg.vars.LastValue("FILESDIR"))
	pkg.Patchdir = NewPackagePathString(pkg.vars.LastValue("PATCHDIR"))

	// See lang/php/ext.mk
	if pkg.vars.IsDefinedSimilar("PHPEXT_MK") {
		if !pkg.vars.IsDefinedSimilar("USE_PHP_EXT_PATCHES") {
			pkg.Patchdir = "patches"
		}
		if pkg.vars.IsDefinedSimilar("PECL_VERSION") {
			pkg.DistinfoFile = "distinfo"
		} else {
			pkg.IgnoreMissingPatches = true
		}

		// For PHP modules that are not PECL, this combination means that
		// the patches in the distinfo cannot be found in PATCHDIR.
	}

	if trace.Tracing {
		trace.Stepf("DISTINFO_FILE=%s", pkg.DistinfoFile)
		trace.Stepf("FILESDIR=%s", pkg.Filesdir)
		trace.Stepf("PATCHDIR=%s", pkg.Patchdir)
		trace.Stepf("PKGDIR=%s", pkg.Pkgdir)
	}

	return mainLines, allLines
}

func (pkg *Package) parse(mklines *MkLines, allLines *MkLines, includingFileForUsedCheck CurrPath, main bool) bool {
	if trace.Tracing {
		defer trace.Call(mklines.lines.Filename)()
	}

	result := mklines.ForEachEnd(
		func(mkline *MkLine) bool {
			return pkg.parseLine(mklines, mkline, allLines, main)
		},
		func(mkline *MkLine) {})

	if includingFileForUsedCheck != "" {
		mklines.CheckUsedBy(G.Pkgsrc.Rel(includingFileForUsedCheck))
	}

	// For every included buildlink3.mk, include the corresponding builtin.mk
	// automatically since the pkgsrc infrastructure does the same.
	filename := mklines.lines.Filename
	if mklines.lines.BaseName == "buildlink3.mk" {
		builtin := filename.Dir().JoinNoClean("builtin.mk").CleanPath()
		builtinRel := NewPackagePath(G.Pkgsrc.Relpath(pkg.dir, builtin))
		if pkg.included.FirstTime(builtinRel) && builtin.IsFile() {
			builtinMkLines := LoadMk(builtin, pkg, MustSucceed|LogErrors)
			pkg.parse(builtinMkLines, allLines, "", false)
		}

		data := LoadBuildlink3Data(mklines)
		if data != nil {
			pkg.bl3Data[data.id] = data
		}
	}

	return result
}

func (pkg *Package) parseLine(mklines *MkLines, mkline *MkLine, allLines *MkLines, main bool) bool {
	allLines.mklines = append(allLines.mklines, mkline)
	allLines.lines.Lines = append(allLines.lines.Lines, mkline.Line)

	if mkline.IsInclude() {
		includingFile := mkline.Filename()
		includedFile := mkline.IncludedFile()
		includedMkLines, skip := pkg.loadIncluded(mkline, includingFile)

		if includedMkLines == nil {
			pkgsrcPath := G.Pkgsrc.Rel(mkline.File(includedFile))
			if skip || mklines.indentation.HasExists(pkgsrcPath) {
				return true // See https://github.com/rillig/pkglint/issues/1
			}
			mkline.Errorf("Cannot read %q.", includedFile)
			return false
		}

		filenameForUsedCheck := NewCurrPath("")
		dir, base := includedFile.Split()
		if dir != "" && base == "Makefile.common" && dir.String() != "../../"+pkg.Pkgpath.String()+"/" {
			filenameForUsedCheck = includingFile
		}
		if !pkg.parse(includedMkLines, allLines, filenameForUsedCheck, false) {
			return false
		}
		if main && pkg.seenPrefs && pkg.prefsLine == nil {
			pkg.prefsLine = mkline
		}
	}

	if mkline.IsVarassign() {
		varname, op, value := mkline.Varname(), mkline.Op(), mkline.Value()

		if op != opAssignDefault || !pkg.vars.IsDefined(varname) {
			if trace.Tracing {
				trace.Stepf("varassign(%q, %q, %q)", varname, op, value)
			}
			pkg.vars.Define(varname, mkline)
		}

		if varname == "pkgbase" {
			pkg.seenPkgbase[mkline.Value()] = struct{}{}
		}
	}
	return true
}

// loadIncluded loads the lines from the file given by the .include directive
// in mkline.
//
// Returning (nil, true) means that the file was deemed irrelevant.
//
// Returning (nil, false) means there was an error loading the file, and that
// error has already been logged.
func (pkg *Package) loadIncluded(mkline *MkLine, includingFile CurrPath) (includedMklines *MkLines, skip bool) {
	includedFile := pkg.resolveIncludedFile(mkline, includingFile)
	if includedFile.IsEmpty() {
		return nil, true
	}

	// TODO: .Dir? Add test before changing this.
	// pkglint -Wall x11/kde-runtime4
	dirname, _ := includingFile.Split()
	dirname = dirname.CleanPath()
	fullIncluded := dirname.JoinNoClean(includedFile)

	if !pkg.shouldDiveInto(includingFile, includedFile) {
		return nil, true
	}

	// XXX: Depending on the current working directory, the filename
	// that is added to pkg.included differs. Running pkglint
	// from the pkgsrc root directory resolves relative paths, while
	// running pkglint from the package directory keeps the "../.."
	// prefix.
	relIncludedFile := NewPackagePath(G.Pkgsrc.Relpath(pkg.dir, fullIncluded))
	if !pkg.included.FirstTime(relIncludedFile) {
		return nil, true
	}

	pkg.collectSeenInclude(mkline, includedFile)

	if trace.Tracing {
		trace.Stepf("Including %q.", fullIncluded)
	}
	includedMklines = LoadMk(fullIncluded, pkg, 0)
	if includedMklines != nil {
		return includedMklines, false
	}

	// Only look in the directory relative to the current file
	// and in the package directory; see
	// devel/bmake/files/parse.c, function Parse_include_file.
	//
	// Bmake has a list of include directories that can be specified
	// on the command line using the -I option, but pkgsrc doesn't
	// make use of that, so pkglint also doesn't need this extra
	// complexity.
	pkgBasedir := pkg.File(".")

	// Prevent unnecessary syscalls
	if dirname == pkgBasedir {
		return nil, false
	}

	dirname = pkgBasedir

	fullIncludedFallback := dirname.JoinNoClean(includedFile)
	includedMklines = LoadMk(fullIncludedFallback, pkg, 0)
	if includedMklines != nil {
		pkg.checkIncludePath(mkline, fullIncludedFallback)
	}
	return includedMklines, false
}

// checkIncludePath checks that the relative path in an '.include' directive
// has the canonical form, see Pkgsrc.Relpath.
// In particular, the path must not point outside the pkgsrc top directory.
func (pkg *Package) checkIncludePath(mkline *MkLine, canonicalRel CurrPath) {
	if containsExpr(mkline.IncludedFile().String()) {
		return
	}

	// TODO: Remove this paragraph after 2023Q3 and fix the affected files
	// from the pkgsrc infrastructure.
	if !G.Testing && G.Pkgsrc.IsInfra(mkline.Filename()) {
		return
	}

	mkline.Warnf("The path to the included file should be %q.",
		mkline.Rel(canonicalRel))
	mkline.Explain(
		"The .include directive first searches the file relative to the including file.",
		"And if that doesn't exist, falls back to the current directory, which in the",
		"case of a pkgsrc package is the package directory.",
		"",
		"This fallback mechanism is not necessary for pkgsrc,",
		"therefore it should not be used.",
		"One less thing to learn for package developers.")
}

// resolveIncludedFile resolves makefile variables such as ${PKGPATH} to
// their actual values, returning an empty path if there are any expressions
// left that cannot be resolved.
func (pkg *Package) resolveIncludedFile(mkline *MkLine, includingFilename CurrPath) RelPath {

	// TODO: Try to combine resolveExprs and ResolveExprsInRelPath.
	resolved := mkline.ResolveExprsInRelPath(mkline.IncludedFile(), pkg)
	includedText := resolveExprs(resolved.String(), nil, pkg)
	includedFile := NewRelPathString(includedText)
	if containsExpr(includedText) {
		if trace.Tracing && !includingFilename.ContainsPath("mk") {
			trace.Stepf("%s:%s: Skipping unresolvable include file %q.",
				mkline.Filename(), mkline.Linenos(), includedFile)
		}
		return ""
	}

	if mkline.Basename != "buildlink3.mk" {
		if includedFile.HasSuffixPath("buildlink3.mk") {
			curr := mkline.File(includedFile)
			if !curr.IsFile() {
				curr = pkg.File(PackagePath(includedFile))
			}
			packagePath := pkg.Rel(curr)
			pkg.bl3[packagePath] = mkline
			if trace.Tracing {
				trace.Stepf("Buildlink3 file in package: %q", packagePath)
			}
		}
	}

	return includedFile
}

// shouldDiveInto decides whether to load the includedFile.
func (pkg *Package) shouldDiveInto(includingFile CurrPath, includedFile RelPath) bool {

	if includedFile.HasSuffixPath("bsd.pkg.mk") || IsPrefs(includedFile) {
		pkg.seenPrefs = true
		return false
	}

	if includedFile.HasSuffixText(".buildlink3.mk") {
		return true
	}
	if G.Pkgsrc.IsInfraMain(includingFile) {
		if !G.Pkgsrc.IsInfra(includingFile.Dir().JoinNoClean(includedFile)) {
			return true
		}
		return includingFile.HasSuffixText(".buildlink3.mk") &&
			includedFile.HasSuffixText(".builtin.mk")
	}

	return true
}

func (pkg *Package) collectSeenInclude(mkline *MkLine, includedFile RelPath) {
	if mkline.Basename != "Makefile" {
		return
	}

	incDir, incBase := includedFile.Split()
	switch {
	case
		incDir.HasPrefixPath("../../mk"),
		incBase == "buildlink3.mk",
		incBase == "builtin.mk",
		incBase == "options.mk":
		return
	}

	if trace.Tracing {
		trace.Stepf("Including %q sets seenInclude.", includedFile)
	}
	pkg.seenInclude = true
}

func (pkg *Package) collectConditionalIncludes(mklines *MkLines) {
	mklines.ForEach(func(mkline *MkLine) {
		if mkline.IsInclude() {
			mkline.SetConditionalVars(mklines.indentation.Varnames())

			includedFile := pkg.Rel(mkline.IncludedFileFull())
			if mklines.indentation.IsConditional() {
				pkg.conditionalIncludes[includedFile] = mkline
			} else {
				pkg.unconditionalIncludes[includedFile] = mkline
			}
		}
	})
}

func (pkg *Package) loadPlistDirs(plistFilename CurrPath) {
	lines := Load(plistFilename, MustSucceed)
	ck := PlistChecker{
		pkg,
		make(map[RelPath]*PlistLine),
		make(map[RelPath]*PlistLine),
		"",
		Once{},
		Once{},
		Once{},
		OncePerString{},
		false}
	plistLines := ck.Load(lines)

	for filename, pline := range ck.allFiles {
		pkg.Plist.Files[filename] = pline
	}
	for dirname, pline := range ck.allDirs {
		if len(pline.conditions) == 0 {
			pkg.Plist.UnconditionalDirs[dirname] = pline
		}
	}
	checkDuplicatesAcrossRanks := !pkg.vars.IsDefined("PLIST_SRC")
	for _, plistLine := range plistLines {
		if plistLine.HasPath() {
			rank := NewPlistRank(plistLine.Line.Basename)
			pkg.PlistLines.Add(plistLine, rank, checkDuplicatesAcrossRanks)
		}
		for _, cond := range plistLine.conditions {
			pkg.Plist.Conditions[strings.TrimPrefix(cond, "PLIST.")] = true
		}
	}
}

func (pkg *Package) check(filenames []CurrPath, mklines, allLines *MkLines) {
	haveDistinfo := false
	havePatches := false

	for _, filename := range filenames {
		if containsExpr(filename.String()) {
			if trace.Tracing {
				trace.Stepf("Skipping file %q because the name contains an unresolved variable.", filename)
			}
			continue
		}

		st, err := filename.Lstat()
		switch {
		case err != nil:
			// For a missing custom distinfo file, an error message is already generated
			// for the line where DISTINFO_FILE is defined.
			//
			// For all other cases it is next to impossible to reach this branch
			// since all those files come from calls to dirglob.
			break

		case filename.HasBase("Makefile") && pkg.Rel(filename) == "Makefile":
			G.checkExecutable(filename, st.Mode())
			pkg.checkfilePackageMakefile(filename, mklines, allLines)

		default:
			pkg.checkDirent(filename, st.Mode())
		}

		if filename.ContainsText("/patches/patch-") {
			havePatches = true
		} else if filename.HasSuffixPath("distinfo") {
			haveDistinfo = true
		}
		pkg.checkOwnerMaintainer(filename)
		pkg.checkPolicyUpdateLimited()
		pkg.checkFreeze(filename)
	}

	if pkg.Pkgdir == "." {
		if havePatches && !haveDistinfo {
			line := NewLineWhole(pkg.File(pkg.DistinfoFile))
			line.Warnf("A package with patches should have a distinfo file.")
			line.Explain(
				"To generate a distinfo file for the existing patches, run",
				sprintf("%q.", bmake("makepatchsum")))
		}

		pkg.checkDescr(filenames, mklines)
	}

	pkg.checkDistinfoFileAndPatchdir()
	pkg.checkDistfilesInDistinfo(allLines)
	pkg.checkPkgConfig(allLines)
	pkg.checkWipCommitMsg()
}

func (pkg *Package) checkDescr(filenames []CurrPath, mklines *MkLines) {
	for _, filename := range filenames {
		if filename.HasBase("DESCR") {
			return
		}
	}
	if pkg.vars.IsDefined("DESCR_SRC") {
		return
	}
	mklines.Whole().Errorf("Each package must have a DESCR file.")
}

func (pkg *Package) checkDistinfoFileAndPatchdir() {
	distLoc := pkg.vars.LastDefinition("DISTINFO_FILE")
	patchLoc := pkg.vars.LastDefinition("PATCHDIR")
	diag := distLoc
	if distLoc != nil && patchLoc != nil {
		dist := distLoc.Value()
		patch := patchLoc.Value()
		distSuff, patchSuff := trimCommonPrefix(dist, patch)
		if !contains(distSuff, "/") && !contains(patchSuff, "/") {
			return
		}
		distLoc.Warnf(
			"DISTINFO_FILE %q does not match PATCHDIR %q from %s.",
			dist, patch, distLoc.RelMkLine(patchLoc))
	} else if distLoc != nil {
		distLoc.Warnf("DISTINFO_FILE %q has no corresponding PATCHDIR.", distLoc.Value())
	} else if patchLoc != nil {
		patchLoc.Warnf("PATCHDIR %q has no corresponding DISTINFO_FILE.", patchLoc.Value())
		diag = patchLoc
	} else {
		return
	}
	diag.Explain(
		"The distinfo file records the checksums of all available patches.",
		"Only those patches that are listed in the distinfo file are applied.",
		"To make the relationship between the distinfo file",
		"and the corresponding patches obvious,",
		"both should be in the same package.",
		"",
		"A typical definition is:",
		"\tDISTINFO_FILE=\t../../category/package/distinfo",
		"\tPATCHDIR=\t../../category/package/patches")
}

func (pkg *Package) checkDistfilesInDistinfo(mklines *MkLines) {
	// Needs more work; see MkLines.IsUnreachable.
	if !G.Experimental {
		return
	}

	if len(pkg.distinfoDistfiles) == 0 {
		return
	}

	redundant := pkg.redundant
	distfiles := redundant.get("DISTFILES")
	if len(distfiles.vari.WriteLocations()) == 0 {
		return
	}

	for _, mkline := range distfiles.vari.WriteLocations() {
		unreachable := newLazyBool(
			func() bool { return mklines.IsUnreachable(mkline) })
		resolved := resolveExprs(mkline.Value(), nil, pkg)

		for _, distfile := range mkline.ValueFields(resolved) {
			if containsExpr(distfile) {
				continue
			}
			if pkg.distinfoDistfiles[NewPath(distfile).Base()] {
				continue
			}
			if unreachable.get() {
				continue
			}
			mkline.Warnf("Distfile %q is not mentioned in %s.",
				distfile, mkline.Rel(pkg.File(pkg.DistinfoFile)))
		}
	}
}

func (pkg *Package) checkPkgConfig(allLines *MkLines) {
	pkgConfig := allLines.Tools.ByName("pkg-config")
	if pkgConfig == nil || !pkgConfig.UsableAtRunTime() {
		return
	}

	for included := range pkg.included.m {
		included := included.String()
		if hasSuffix(included, "buildlink3.mk") ||
			hasSuffix(included, "/mk/apache.mk") ||
			hasSuffix(included, "/mk/apache.module.mk") {
			return
		}
	}

	mkline := allLines.mklines[0]
	mkline.Warnf("The package uses the tool \"pkg-config\" " +
		"but doesn't include any buildlink3 file.")
	mkline.Explain(
		"The pkgsrc tool wrappers replace the \"pkg-config\" command",
		"with a pkg-config implementation that looks in the buildlink3",
		"directory.",
		"This directory is populated by including the dependencies via",
		"the buildlink3.mk files.",
		"Since this package does not include any such files, the buildlink3",
		"directory will be empty and pkg-config will not find anything.")
}

func (pkg *Package) checkWipCommitMsg() {
	if !G.Wip {
		return
	}
	if pkg.File("TODO").IsFile() {
		return
	}
	file := pkg.File("COMMIT_MSG")
	lines := Load(file, NotEmpty)
	if lines == nil {
		line := NewLineWhole(file)
		line.Warnf("Every work-in-progress package should have a COMMIT_MSG file.")
		line.Explain(
			"A wip package should have a file COMMIT_MSG",
			"that contains exactly the text",
			"that should be used for importing the package to main pkgsrc,",
			"or for updating the main pkgsrc package from the wip version.",
			"Someone with main pkgsrc write access should be able",
			"to simply run 'cvs commit -F COMMIT_MSG'.",
			"",
			"Line 1 should have one of these forms:",
			"\tcategory/pkgpath: Add foo version 1.2.3",
			"\tcategory/pkgpath: Update foo to 4.5.6",
			"",
			"The next paragraph gives credit",
			"to the work-in-progress packager, such as:",
			"\tPackaged in wip by Alyssa P. Hacker",
			"\tUpdate prepared in wip by Ben Bitdiddle",
			"",
			"The next paragraph describes the pkgsrc-specific",
			"packaging changes, if any.",
			"",
			"The next paragraph summarizes the upstream changes",
			"on a high level.",
			"In packages following the GNU Coding Standards,",
			"these changes are in the NEWS file, see",
			"https://www.gnu.org/prep/standards/html_node/NEWS-File.html.",
			"",
			"See https://www.pkgsrc.org/wip/users/#COMMIT_MSG.")
		return
	}
}

func (pkg *Package) checkfilePackageMakefile(filename CurrPath, mklines *MkLines, allLines *MkLines) {
	if trace.Tracing {
		defer trace.Call(filename)()
	}

	pkg.checkPlist()

	pkg.checkDistinfoExists()

	pkg.checkReplaceInterpreter()

	vars := pkg.vars
	if !vars.IsDefined("LICENSE") && !vars.IsDefined("META_PACKAGE") {
		line := NewLineWhole(filename)
		line.Errorf("Each package must define its LICENSE.")
		// TODO: Explain why the LICENSE is necessary.
		line.Explain(
			"To take a good guess on the license of a package,",
			sprintf("run %q.", bmake("guess-license")))
	}

	pkg.redundant = NewRedundantScope()
	pkg.redundant.IsRelevant = func(mkline *MkLine) bool {
		// As of December 2019, the RedundantScope is only used for
		// checking a whole package. Therefore, G.Infrastructure can
		// never be true and there is no point testing it.
		//
		// If the RedundantScope is applied also to individual files,
		// it would have to be added here.
		return G.CheckGlobal || !G.Pkgsrc.IsInfra(mkline.Filename())
	}
	pkg.redundant.Check(allLines) // Updates the variables in the scope
	pkg.checkGnuConfigureUseLanguages()
	pkg.checkUseLanguagesCompilerMk(allLines)

	pkg.determineEffectivePkgVars()
	pkg.checkPossibleDowngrade()
	pkg.checkOptionsMk()

	if !vars.IsDefined("COMMENT") {
		NewLineWhole(filename).Warnf("Each package should define a COMMENT.")
	}

	if imake := vars.FirstDefinition("USE_IMAKE"); imake != nil {
		if x11 := vars.FirstDefinition("USE_X11"); x11 != nil {
			if !x11.Filename().HasSuffixPath("mk/x11.buildlink3.mk") {
				imake.Notef("USE_IMAKE makes USE_X11 in %s redundant.", imake.RelMkLine(x11))
			}
		}
	}

	pkg.checkUpdate()

	// TODO: Maybe later collect the conditional includes from allLines
	//  instead of mklines. This will lead to about 6000 new warnings
	//  though.
	// pkg.collectConditionalIncludes(allLines)

	allLines.collectVariables(false, true) // To get the tool definitions
	mklines.Tools = allLines.Tools         // TODO: also copy the other collected data

	// TODO: Checking only mklines instead of allLines ignores the
	//  .include lines. For example, including "options.mk" does not
	//  set Tools.SeenPrefs, but it should.
	//
	// See Test_Package_checkfilePackageMakefile__options_mk.
	mklines.checkAllData.postLine = func(mkline *MkLine) {
		if mkline == pkg.prefsLine {
			pkg.seenPrefs = true
		}
	}
	mklines.Check()

	// This check is experimental because it's not yet clear how to
	// classify the various Python packages and whether all Python
	// packages really need the prefix.
	if G.Experimental && pkg.EffectivePkgname != "" && pkg.Includes("../../lang/python/extension.mk") != nil {
		pkg.EffectivePkgnameLine.Warnf("The PKGNAME of Python extensions should start with ${PYPKGPREFIX}.")
	}

	if !pkg.seenInclude {
		NewVarorderChecker(mklines).Check()
	}

	pkg.checkMeson(mklines)

	SaveAutofixChanges(mklines.lines)
}

func (pkg *Package) checkReplaceInterpreter() {
	vars := pkg.vars
	noConfigureLine := vars.FirstDefinition("NO_CONFIGURE")
	if noConfigureLine == nil {
		return
	}

	// See mk/configure/replace-interpreter.mk.
	varnames := [...]string{
		"REPLACE_AWK",
		"REPLACE_BASH",
		"REPLACE_CSH",
		"REPLACE_KSH",
		"REPLACE_PERL",
		"REPLACE_PERL6",
		"REPLACE_SH",
		"REPLACE_INTERPRETER"}

	for _, varname := range varnames {
		mkline := vars.FirstDefinition(varname)
		if mkline == nil {
			continue
		}
		mkline.Warnf("%s is ignored when NO_CONFIGURE is set (in %s).",
			varname, mkline.RelMkLine(noConfigureLine))
	}
}

func (pkg *Package) checkDistinfoExists() {
	vars := pkg.vars

	want := pkg.wantDistinfo(vars)

	if !want {
		distinfoFile := pkg.File(pkg.DistinfoFile)
		if distinfoFile.IsFile() {
			line := NewLineWhole(distinfoFile)
			line.Warnf("This file should not exist.")
			line.Explain(
				"This package neither downloads external files (distfiles),",
				"nor has it any patches that would need to be validated.")
		}
	} else {
		distinfoFile := pkg.File(pkg.DistinfoFile)
		if !containsExpr(distinfoFile.String()) && !distinfoFile.IsFile() {
			line := NewLineWhole(distinfoFile)
			line.Warnf("A package that downloads files should have a distinfo file.")
			line.Explain(
				sprintf("To generate the distinfo file, run %q.", bmake("makesum")),
				"",
				"To mark the package as not needing a distinfo file, set",
				"NO_CHECKSUM=yes in the package Makefile.")
		}
	}
}

func (pkg *Package) wantDistinfo(vars Scope) bool {
	switch {
	case vars.IsDefined("DISTINFO_FILE"):
		return true
	case vars.IsDefined("DISTFILES") && vars.LastValue("DISTFILES") != "":
		return true
	case vars.IsDefined("DISTFILES"):
		break
	case vars.IsDefined("NO_CHECKSUM"):
		break
	case vars.IsDefined("META_PACKAGE"):
		break // see Test_Package_checkfilePackageMakefile__META_PACKAGE_with_patch
	case !vars.IsDefined("DISTNAME"):
		break
	default:
		return true
	}

	return !isEmptyDir(pkg.File(pkg.Patchdir))
}

// checkPlist checks whether the package needs a PLIST file,
// or whether that file should be omitted since it is autogenerated.
func (pkg *Package) checkPlist() {
	vars := pkg.vars
	if vars.IsDefined("PLIST_SRC") || vars.IsDefined("GENERATE_PLIST") {
		return
	}

	needsPlist, line := pkg.needsPlist()
	hasPlist := pkg.File(pkg.Pkgdir.JoinNoClean("PLIST")).IsFile() ||
		pkg.File(pkg.Pkgdir.JoinNoClean("PLIST.common")).IsFile()

	if needsPlist && !hasPlist {
		line.Warnf("This package should have a PLIST file.")
		line.Explain(
			"The PLIST file provides the list of files that will be",
			"installed by the package. Having this list ensures that",
			"a package update doesn't accidentally modify the list",
			"of installed files.",
			"",
			seeGuide("PLIST issues", "plist"))
	}

	if hasPlist && !needsPlist {
		line.Warnf("This package should not have a PLIST file.")
	}
}

func (pkg *Package) needsPlist() (bool, *Line) {
	vars := pkg.vars

	// TODO: In the below code, it shouldn't be necessary to mention
	//  each variable name twice.

	if vars.IsDefined("PERL5_PACKLIST") {
		return false, vars.LastDefinition("PERL5_PACKLIST").Line
	}

	if vars.IsDefined("PERL5_USE_PACKLIST") {
		needed := strings.ToLower(vars.LastValue("PERL5_USE_PACKLIST")) == "no"
		return needed, vars.LastDefinition("PERL5_USE_PACKLIST").Line
	}

	if vars.IsDefined("META_PACKAGE") {
		return false, vars.LastDefinition("META_PACKAGE").Line
	}

	return true, NewLineWhole(pkg.File("Makefile"))
}

func (pkg *Package) checkGnuConfigureUseLanguages() {
	s := pkg.redundant

	gnuConfigure := s.vars["GNU_CONFIGURE"]
	if gnuConfigure == nil || !gnuConfigure.vari.IsConstant() {
		return
	}

	useLanguages := s.vars["USE_LANGUAGES"]
	if useLanguages == nil || !useLanguages.vari.IsConstant() {
		return
	}

	var wrongLines []*MkLine
	for _, mkline := range useLanguages.vari.WriteLocations() {

		if G.Pkgsrc.IsInfra(mkline.Filename()) {
			continue
		}

		if matches(mkline.Comment(), `(?-i)\b(?:c|empty|none)\b`) {
			// Don't emit a warning since the comment probably contains a
			// statement that C is really not needed.
			return
		}

		languages := mkline.Value()
		if matches(languages, `(?:^|[\t ]+)(?:c|c99|objc)(?:[\t ]+|$)`) {
			return
		}

		wrongLines = append(wrongLines, mkline)
	}

	gnuLine := gnuConfigure.vari.WriteLocations()[0]
	for _, useLine := range wrongLines {
		gnuLine.Warnf(
			"GNU_CONFIGURE almost always needs a C compiler, "+
				"but \"c\" is not added to USE_LANGUAGES in %s.",
			gnuLine.RelMkLine(useLine))
	}
}

// checkUseLanguagesCompilerMk checks that after including mk/compiler.mk
// or mk/endian.mk for the first time, there are no more changes to
// USE_LANGUAGES, as these would be ignored by the pkgsrc infrastructure.
func (pkg *Package) checkUseLanguagesCompilerMk(mklines *MkLines) {

	var seen OncePerString

	handleVarassign := func(mkline *MkLine) {
		if mkline.Varname() != "USE_LANGUAGES" {
			return
		}

		if !seen.Seen("../../mk/compiler.mk") && !seen.Seen("../../mk/endian.mk") {
			return
		}

		if mkline.Basename == "compiler.mk" {
			if G.Pkgsrc.Relpath(pkg.dir, mkline.Filename()) == "../../mk/compiler.mk" {
				return
			}
		}

		mkline.Warnf("Modifying USE_LANGUAGES after including ../../mk/compiler.mk has no effect.")
		mkline.Explain(
			"The file compiler.mk guards itself against multiple inclusion.")
	}

	handleInclude := func(mkline *MkLine) {
		_ = seen.FirstTime(pkg.Rel(mkline.IncludedFileFull()).String())
	}

	mklines.ForEach(func(mkline *MkLine) {
		switch {
		case mkline.IsVarassign():
			handleVarassign(mkline)

		case mkline.IsInclude():
			handleInclude(mkline)
		}
	})
}

// checkMeson checks for typical leftover snippets from packages that used
// GNU autotools or another build system, before being migrated to Meson.
func (pkg *Package) checkMeson(mklines *MkLines) {

	mkline := pkg.Includes("../../devel/meson/build.mk")
	if mkline == nil {
		return
	}

	pkg.checkMesonGnuMake(mklines)
	pkg.checkMesonConfigureArgs()
	pkg.checkMesonPython(mklines, mkline)
}

func (pkg *Package) checkMesonGnuMake(mklines *MkLines) {
	gmake := mklines.Tools.ByName("gmake")
	if gmake != nil && gmake.UsableAtRunTime() {
		mkline := NewLineWhole(pkg.File("."))
		mkline.Warnf("Meson packages usually don't need GNU make.")
		mkline.Explain(
			"After migrating a package from GNU make to Meson,",
			"GNU make is typically not needed anymore.")
	}
}

func (pkg *Package) checkMesonConfigureArgs() {
	mkline := pkg.vars.FirstDefinition("CONFIGURE_ARGS")
	if mkline == nil {
		return
	}

	if pkg.Rel(mkline.Location.Filename).HasPrefixPath("..") {
		return
	}

	mkline.Warnf("Meson packages usually don't need CONFIGURE_ARGS.")
	mkline.Explain(
		"After migrating a package from GNU make to Meson,",
		"CONFIGURE_ARGS are typically not needed anymore.")
}

func (pkg *Package) checkMesonPython(mklines *MkLines, mkline *MkLine) {

	if mklines.allVars.IsDefined("PYTHON_FOR_BUILD_ONLY") {
		return
	}
	if mklines.allVars.IsDefined("REPLACE_PYTHON") {
		return
	}

	for path := range pkg.unconditionalIncludes {
		if path.ContainsPath("lang/python") && !path.AsPath().HasSuffixPath("lang/python/tool.mk") {
			goto warn
		}
	}
	return

warn:
	mkline.Warnf("Meson packages usually need Python only at build time.")
	mkline.Explain(
		"The Meson build system is implemented in Python,",
		"therefore packages that use Meson need Python",
		"as a build-time dependency.",
		"After building the package, it is typically independent from Python.",
		"",
		"To change the Python dependency to build-time,",
		"set PYTHON_FOR_BUILD_ONLY=tool in the package Makefile.")
}

func (pkg *Package) determineEffectivePkgVars() {
	distnameLine := pkg.vars.FirstDefinition("DISTNAME")
	pkgnameLine := pkg.vars.FirstDefinition("PKGNAME")

	distname := ""
	if distnameLine != nil {
		distname = distnameLine.Value()
	}

	pkgname := ""
	if pkgnameLine != nil {
		pkgname = pkgnameLine.Value()
	}

	effname := pkgname
	if distname != "" && effname != "" {
		merged, ok := pkg.pkgnameFromDistname(effname, distname, pkgnameLine)
		if ok {
			effname = merged
		}
	}

	pkg.checkPkgnameRedundant(pkgnameLine, pkgname, distname)

	if pkgname == "" && distnameLine != nil && !containsExpr(distname) && !matchesPkgname(distname) {
		distnameLine.Warnf("As DISTNAME is not a valid package name, define the PKGNAME explicitly.")
	}

	if pkgname != "" {
		distname = ""
	}

	if effname != "" && !containsExpr(effname) {
		if m, m1, m2 := matchPkgname(effname); m {
			pkg.EffectivePkgname = effname + pkg.nbPart()
			pkg.EffectivePkgnameLine = pkgnameLine
			pkg.EffectivePkgbase = m1
			pkg.EffectivePkgversion = m2
		}
	}

	if pkg.EffectivePkgnameLine == nil && distname != "" && !containsExpr(distname) {
		if m, m1, m2 := matchPkgname(distname); m {
			pkg.EffectivePkgname = distname + pkg.nbPart()
			pkg.EffectivePkgnameLine = distnameLine
			pkg.EffectivePkgbase = m1
			pkg.EffectivePkgversion = m2
		}
	}

	if pkg.EffectivePkgnameLine != nil {
		if trace.Tracing {
			trace.Stepf("Effective name=%q base=%q version=%q",
				pkg.EffectivePkgname, pkg.EffectivePkgbase, pkg.EffectivePkgversion)
		}
	}
}

func (pkg *Package) checkPkgnameRedundant(pkgnameLine *MkLine, pkgname string, distname string) {
	if pkgnameLine == nil || pkgnameLine.HasComment() {
		return
	}
	if pkgname != distname && pkgname != "${DISTNAME}" {
		return
	}
	pkgnameInfo := pkg.redundant.vars["PKGNAME"]
	if len(pkgnameInfo.vari.WriteLocations()) >= 2 {
		return
	}
	pkgnameLine.Notef("This assignment is probably redundant " +
		"since PKGNAME is ${DISTNAME} by default.")
	pkgnameLine.Explain(
		"To mark this assignment as necessary, add a comment to the end of this line.")
}

// nbPart determines the smallest part of the package version number,
// typically "nb13" or an empty string.
//
// It is only used inside pkgsrc to mark changes that are
// independent of the upstream package.
func (pkg *Package) nbPart() string {
	pkgrevision := pkg.vars.LastValue("PKGREVISION")
	if rev, err := strconv.Atoi(pkgrevision); err == nil {
		return "nb" + strconv.Itoa(rev)
	}
	return ""
}

func (pkg *Package) pkgnameFromDistname(pkgname, distname string, diag Diagnoser) (string, bool) {
	tokens, rest := NewMkLexer(pkgname, nil).MkTokens()
	if rest != "" {
		return "", false
	}

	// TODO: Make this resolving of variable references available to all other variables as well.

	result := NewLazyStringBuilder(pkgname)
	for _, token := range tokens {
		if token.Expr != nil {
			if token.Expr.varname != "DISTNAME" {
				return "", false
			}

			newDistname := distname
			for _, mod := range token.Expr.modifiers {
				if mod.IsToLower() {
					newDistname = strings.ToLower(newDistname)
				} else if ok, subst := mod.Subst(newDistname); ok {
					if subst == newDistname && !containsExpr(subst) {
						diag.Notef("The modifier :%s does not have an effect.", mod.String())
					}
					newDistname = subst
				} else {
					return "", false
				}
			}
			result.WriteString(newDistname)
		} else {
			result.WriteString(token.Text)
		}
	}
	return result.String(), true
}

func (pkg *Package) checkPossibleDowngrade() {
	if trace.Tracing {
		defer trace.Call0()()
	}

	m, _, pkgversion := matchPkgname(pkg.EffectivePkgname)
	if !m {
		return
	}

	mkline := pkg.EffectivePkgnameLine

	change := G.Pkgsrc.changes.LastChange[pkg.Pkgpath]
	if change == nil {
		if trace.Tracing {
			trace.Stepf("No change log for package %q", pkg.Pkgpath)
		}
		return
	}

	if change.Action == Updated {
		pkgversionNorev := replaceAll(pkgversion, `nb\d+$`, "")
		changeNorev := replaceAll(change.Version(), `nb\d+$`, "")
		cmp := pkgver.Compare(pkgversionNorev, changeNorev)
		switch {
		case cmp < 0:
			mkline.Warnf("The package is being downgraded from %s (see %s) to %s.",
				change.Version(), mkline.Line.RelLocation(change.Location), pkgversion)
			mkline.Explain(
				"The files in doc/CHANGES-*, in which all version changes are",
				"recorded, have a higher version number than what the package says.",
				"This is unusual, since packages are typically upgraded instead of",
				"downgraded.")

		case cmp > 0 && !isLocallyModified(mkline.Filename()):
			mkline.Notef("Package version %q is greater than the latest %q from %s.",
				pkgversion, change.Version(), mkline.Line.RelLocation(change.Location))
			mkline.Explain(
				"Each update to a package should be mentioned in the doc/CHANGES file.",
				"That file is used for the quarterly statistics of updated packages.",
				"",
				"To do this after updating a package, run",
				sprintf("%q,", bmake("cce")),
				"which is the abbreviation for commit-changes-entry.")
		}
	}
}

func (pkg *Package) checkOptionsMk() {
	for f := range pkg.included.m {
		if f.AsPath().HasBase("options.mk") {
			return
		}
	}
	if pkg.File("options.mk").Exists() {
		mkline := pkg.Makefile.Whole()
		mkline.Errorf("Each package must include its own options.mk file.")
		mkline.Explain(
			"When a package defines an options.mk file,",
			"that means that the package has some",
			"build-time options.",
			"To make these options available to the pkgsrc user,",
			"either the package Makefile or some other makefile",
			"that is included by the package Makefile must have",
			"this line:",
			"\t.include \"options.mk\"")
	}
}

func (pkg *Package) checkUpdate() {
	if pkg.EffectivePkgbase == "" {
		return
	}

	for _, sugg := range G.Pkgsrc.SuggestedUpdates() {
		if pkg.EffectivePkgbase != sugg.Pkgname {
			continue
		}

		suggver, comment := sugg.Version, sugg.Comment

		commentSuffix := func() string {
			if comment != "" {
				return " (" + comment + ")"
			}
			return ""
		}

		mkline := pkg.EffectivePkgnameLine
		cmp := pkgver.Compare(pkg.EffectivePkgversion, suggver)
		ref := mkline.RelLocation(sugg.Line)
		switch {

		case cmp < 0:
			if comment != "" {
				mkline.Warnf("This package should be updated to %s (%s; see %s).",
					sugg.Version, comment, ref)
			} else {
				mkline.Warnf("This package should be updated to %s (see %s).",
					sugg.Version, ref)
			}

		case cmp > 0:
			mkline.Notef("This package is newer than the update request to %s%s from %s.",
				suggver, commentSuffix(), ref)

		default:
			mkline.Notef("The update request to %s%s from %s has been done.",
				suggver, commentSuffix(), ref)
		}
	}
}

// checkDirent checks a directory entry based on its filename and its mode
// (regular file, directory, symlink).
func (pkg *Package) checkDirent(dirent CurrPath, mode os.FileMode) {
	// TODO: merge duplicate code in Pkglint.checkMode

	basename := dirent.Base()

	switch {

	case mode.IsRegular():
		G.checkReg(dirent, basename, G.Pkgsrc.Rel(dirent).Count(), pkg)

	case basename.HasPrefixText("work"):
		if G.Import {
			NewLineWhole(dirent).Errorf("Must be cleaned up before committing the package.")
		}
		return

	case mode.IsDir():
		switch {
		case basename == "files",
			basename == "patches",
			dirent.Dir().HasBase("files"),
			isEmptyDir(dirent):
			break

		default:
			NewLineWhole(dirent).Warnf("Unknown directory name.")
		}

	case mode&os.ModeSymlink != 0:
		line := NewLineWhole(dirent)
		line.Warnf("Invalid symlink name.")
		line.Explain(
			"The only symlinks that pkglint ever expects are those to",
			"WRKDIR, which are usually named 'work' or 'work.*'.")

	default:
		NewLineWhole(dirent).Errorf("Only files and directories are allowed in pkgsrc.")
	}
}

// checkOwnerMaintainer checks files that are about to be committed.
// Depending on whether the package has a MAINTAINER or an OWNER,
// the wording differs.
//
// Pkglint assumes that the local username is the same as the NetBSD
// username, which fits most scenarios.
func (pkg *Package) checkOwnerMaintainer(filename CurrPath) {
	if trace.Tracing {
		defer trace.Call(filename)()
	}

	owner := pkg.vars.LastValue("OWNER")
	maintainer := pkg.vars.LastValue("MAINTAINER")
	if maintainer == "pkgsrc-users@NetBSD.org" || maintainer == "INSERT_YOUR_MAIL_ADDRESS_HERE" {
		maintainer = ""
	}
	if owner == "" && maintainer == "" {
		return
	}

	username := G.Username
	if trace.Tracing {
		trace.Stepf("user=%q owner=%q maintainer=%q", username, owner, maintainer)
	}

	if username == strings.Split(owner, "@")[0] || username == strings.Split(maintainer, "@")[0] {
		return
	}

	if !isLocallyModified(filename) {
		return
	}

	if owner != "" {
		line := NewLineWhole(filename)
		line.Warnf("Don't commit changes to this file without asking the OWNER, %s.", owner)
		line.Explain(
			seeGuide("Package components, Makefile", "components.Makefile"))
		return
	}

	line := NewLineWhole(pkg.File("."))
	line.Notef("Only commit changes that %s would approve.", maintainer)
	line.Explain(
		"See the pkgsrc guide, section \"Package components\",",
		"keyword \"maintainer\", for more information.")
}

func (pkg *Package) checkPolicyUpdateLimited() {
	if !pkg.checkedPolicyUpdateLimited.FirstTime() {
		return
	}

	varname := "POLICY_UPDATE_LIMITED"
	limits := pkg.vars.LastValue(varname)
	if limits == "" {
		return
	}
	mkline := pkg.vars.LastDefinition(varname)
	if containsExpr(limits) {
		mkline.Errorf("The value for \"%s\" must be given directly.", varname)
		return
	}

	if isLocallyModified(mkline.Filename()) {
		line := NewLineWhole(pkg.File("."))
		line.Warnf("Changes to this package require extensive testing.")
		line.Explain(
			seeGuide("pkgsrc Policies", "policies"))
	}
}

func (pkg *Package) checkFreeze(filename CurrPath) {
	freezeStart := G.Pkgsrc.changes.LastFreezeStart
	if freezeStart == "" || G.Pkgsrc.changes.LastFreezeEnd != "" {
		return
	}

	if !isLocallyModified(filename) {
		return
	}

	line := NewLineWhole(filename)
	line.Notef("Pkgsrc is frozen since %s.", freezeStart)
	line.Explain(
		"During a pkgsrc freeze, changes to pkgsrc should only be made very carefully.",
		"See https://www.NetBSD.org/developers/pkgsrc/ for the exact rules.")
}

// TODO: Move to MkLinesChecker.
func (*Package) checkFileMakefileExt(filename CurrPath) {
	base := filename.Base()
	if !base.HasPrefixText("Makefile.") || base == "Makefile.common" {
		return
	}
	ext := strings.TrimPrefix(base.String(), "Makefile.")

	line := NewLineWhole(filename)
	line.Notef("Consider renaming %q to %q.", base, ext+".mk")
	line.Explain(
		"The main definition of a pkgsrc package should be in the Makefile.",
		"Common definitions for a few very closely related packages can be",
		"placed in a Makefile.common, these may cover various topics.",
		"",
		"All other definitions should be grouped by topics and implemented",
		"in separate files named *.mk after their topics. Typical examples",
		"are extension.mk, module.mk, version.mk.",
		"",
		"These topic files should be documented properly so that their",
		sprintf("content can be queried using %q.", bmakeHelp("help")))
}

// checkLinesBuildlink3Inclusion checks whether the package Makefile includes
// at least those buildlink3.mk files that are included by the buildlink3.mk
// file of the package.
//
// The other direction is not checked since it is perfectly fine for a package
// to have more dependencies than are needed for buildlink the package.
// (This might be worth re-checking though.)
func (pkg *Package) checkLinesBuildlink3Inclusion(mklines *MkLines) {
	if trace.Tracing {
		defer trace.Call0()()
	}

	// Collect all the included buildlink3.mk files from the file.
	includedFiles := make(map[PackagePath]*MkLine)
	for _, mkline := range mklines.mklines {
		if mkline.IsInclude() {
			included := pkg.Rel(mkline.IncludedFileFull())
			if included.AsPath().HasSuffixPath("buildlink3.mk") {
				includedFiles[included] = mkline
				if pkg.bl3[included] == nil {
					mkline.Warnf("%s is included by this file but not by the package.",
						mkline.IncludedFile())
				}
			}
		}
	}

	if trace.Tracing {
		for packageBl3 := range pkg.bl3 {
			if includedFiles[packageBl3] == nil {
				trace.Stepf("%s is included by the package but not by the buildlink3.mk file.", packageBl3)
			}
		}
	}
}

func (pkg *Package) checkIncludeConditionally(mkline *MkLine, indentation *Indentation) {
	if IsPrefs(mkline.IncludedFile()) {
		return
	}

	key := pkg.Rel(mkline.IncludedFileFull())

	explainPkgOptions := func(uncond *MkLine, cond *MkLine) {
		if uncond.Basename == "buildlink3.mk" && containsStr(cond.ConditionalVars(), "PKG_OPTIONS") {
			mkline.Explain(
				"When including a dependent file, the conditions in the",
				"buildlink3.mk file should be the same as in options.mk",
				"or the Makefile.",
				"",
				"To find out the PKG_OPTIONS of this package at build time,",
				"have a look at mk/pkg-build-options.mk.")
		}
	}

	dependingOn := func(varnames []string) string {
		if len(varnames) == 0 {
			return ""
		}
		return sprintf(" (depending on %s)", strings.Join(varnames, ", "))
	}

	if indentation.IsConditional() {
		if other := pkg.unconditionalIncludes[key]; other != nil {
			if !pkg.warnedAboutConditionalInclusion.FirstTime(mkline.String(), other.String()) {
				return
			}

			mkline.Warnf(
				"%q is included conditionally here%s "+
					"and unconditionally in %s.",
				mkline.IncludedFile().CleanPath(),
				dependingOn(mkline.ConditionalVars()),
				mkline.RelMkLine(other))

			explainPkgOptions(other, mkline)
		}

	} else {
		if other := pkg.conditionalIncludes[key]; other != nil {
			if !pkg.warnedAboutConditionalInclusion.FirstTime(other.String(), mkline.String()) {
				return
			}

			mkline.Warnf(
				"%q is included unconditionally here "+
					"and conditionally in %s%s.",
				mkline.IncludedFile().CleanPath(),
				mkline.RelMkLine(other),
				dependingOn(other.ConditionalVars()))

			explainPkgOptions(mkline, other)
		}
	}

	// TODO: Check whether the conditional variables are the same on both places.
	//  Ideally they should match, but there may be some differences in internal
	//  variables, which need to be filtered out before comparing them, like it is
	//  already done with *_MK variables.
}

func (pkg *Package) matchesLicenseFile(basename RelPath) bool {
	licenseFile := NewPath(pkg.vars.LastValue("LICENSE_FILE"))
	return basename == licenseFile.Base()
}

func (pkg *Package) AutofixDistinfo(oldSha1, newSha1 string) {
	distinfoFilename := pkg.File(pkg.DistinfoFile)
	if lines := Load(distinfoFilename, NotEmpty|LogErrors); lines != nil {
		for _, line := range lines.Lines {
			fix := line.Autofix()
			fix.Warnf(SilentAutofixFormat)
			fix.Replace(oldSha1, newSha1)
			fix.Apply()
		}
		lines.SaveAutofixChanges()
	}
}

// File returns the (possibly absolute) path to relativeFileName,
// as resolved from the package's directory.
// Variables that are known in the package are resolved, e.g. ${PKGDIR}.
func (pkg *Package) File(relativeFileName PackagePath) CurrPath {
	joined := pkg.dir.JoinNoClean(NewRelPath(relativeFileName.AsPath()))
	resolved := resolveExprs(joined.String(), nil, pkg)
	return NewCurrPathString(resolved).CleanPath()
}

// Rel returns the path by which the given filename (as seen from the
// current working directory) can be reached as a relative path from
// the package directory.
//
// Example:
//
//	NewPackage("category/package").Rel("other/package") == "../../other/package"
func (pkg *Package) Rel(filename CurrPath) PackagePath {
	return NewPackagePath(G.Pkgsrc.Relpath(pkg.dir, filename))
}

// Includes returns whether the given file
// is included somewhere in the package, either directly or indirectly.
func (pkg *Package) Includes(filename PackagePath) *MkLine {
	mkline := pkg.unconditionalIncludes[filename]
	if mkline == nil {
		mkline = pkg.conditionalIncludes[filename]
	}
	return mkline
}

// CanFixAddInclude tests whether the package Makefile follows the standard
// form and thus allows to add additional '.include' directives.
func (pkg *Package) CanFixAddInclude() bool {
	mklines := pkg.Makefile
	lastLine := mklines.mklines[len(mklines.mklines)-1]
	return lastLine.Text == ".include \"../../mk/bsd.pkg.mk\""
}

// FixAddInclude adds an '.include' directive at the end of the package
// Makefile.
func (pkg *Package) FixAddInclude(includedFile PackagePath) {
	mklines := pkg.Makefile

	alreadyThere := false
	mklines.ForEach(func(mkline *MkLine) {
		if mkline.IsInclude() && mkline.IncludedFile() == includedFile.AsRelPath() {
			alreadyThere = true
		}

		// XXX: This low-level code should not be necessary.
		// Instead, the added '.include' line should be a MkLine of its own.
		if mkline.fix != nil {
			expected := ".include \"" + includedFile.String() + "\"\n"
			for _, rawLine := range mkline.fix.above {
				if rawLine == expected {
					alreadyThere = true
				}
			}
		}
	})
	if alreadyThere {
		return
	}

	lastLine := mklines.mklines[len(mklines.mklines)-1]
	fix := lastLine.Autofix()
	fix.Silent()
	fix.InsertAbove(".include \"" + includedFile.String() + "\"")
	fix.Apply()

	mklines.SaveAutofixChanges()
}

// PlistContent lists the directories and files that appear in the
// package's PLIST files. It serves two purposes:
//
// 1. Decide whether AUTO_MKDIRS can be used instead of listing
// the INSTALLATION_DIRS redundantly.
//
// 2. Ensure that the entries mentioned in the ALTERNATIVES file
// also appear in the PLIST files.
type PlistContent struct {
	UnconditionalDirs map[RelPath]*PlistLine
	Files             map[RelPath]*PlistLine
	Conditions        map[string]bool // each ${PLIST.id} sets ["id"] = true.
}

func NewPlistContent() PlistContent {
	return PlistContent{
		make(map[RelPath]*PlistLine),
		make(map[RelPath]*PlistLine),
		make(map[string]bool)}
}

// IncludedMap remembers which files the package Makefile has included,
// including indirect files.
// See OncePerString.
type IncludedMap struct {
	m     map[PackagePath]struct{}
	Trace bool
}

func NewIncludedMap() IncludedMap {
	return IncludedMap{make(map[PackagePath]struct{}), false}
}

func (im *IncludedMap) FirstTime(p PackagePath) bool {
	_, found := im.m[p]
	if !found {
		im.m[p] = struct{}{}
		if im.Trace {
			G.Logger.out.WriteLine("FirstTime: " + p.String())
		}
	}
	return !found
}

func (im *IncludedMap) Seen(p PackagePath) bool {
	_, seen := im.m[p]
	return seen
}

// matchPkgname tests whether the string has the form of a package name that
// does not contain any variable expressions.
func matchPkgname(s string) (m bool, base string, version string) {
	// TODO: Allow a hyphen in the middle of a version number.
	return match2(s, `^([\w\-.+]+)-([0-9][.0-9A-Z_a-z]*)$`)
}

// matchPkgname tests whether the string has the form of a package name that
// does not contain any variable expressions.
func matchesPkgname(s string) bool {
	m, _, _ := matchPkgname(s)
	return m
}
