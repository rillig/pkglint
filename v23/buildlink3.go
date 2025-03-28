package pkglint

import (
	"github.com/rillig/pkglint/v23/pkgver"
	"strings"
)

// Checks for 'buildlink3.mk' files, which manage those dependencies of a
// package that provide C/C++ header files or shared libraries, to only
// provide these files to packages that explicitly declare this dependency.

type Buildlink3Checker struct {
	mklines          *MkLines
	pkgbase          string
	pkgbaseLine      *MkLine
	abiLine, apiLine *MkLine
	abi, api         *PackagePattern
}

func CheckLinesBuildlink3Mk(mklines *MkLines) {
	(&Buildlink3Checker{mklines: mklines}).Check()
}

func (ck *Buildlink3Checker) Check() {
	mklines := ck.mklines
	if trace.Tracing {
		defer trace.Call(mklines.lines.Filename)()
	}

	mklines.Check()

	llex := NewMkLinesLexer(mklines)

	for llex.SkipIf((*MkLine).IsComment) {
		line := llex.PreviousLine()
		// See pkgtools/createbuildlink/files/createbuildlink
		if hasPrefix(line.Text, "# XXX This file was created automatically") {
			line.Errorf("This comment indicates unfinished work (url2pkg).")
		}
	}

	llex.SkipEmptyOrNote()

	if llex.SkipRegexp(`^BUILDLINK_DEPMETHOD\.([^\t ]+)\?=.*$`) {
		llex.PreviousLine().Warnf("This line belongs inside the .ifdef block.")
		for llex.SkipText("") {
		}
	}

	if !ck.checkFirstParagraph(llex) {
		return
	}
	if !ck.checkSecondParagraph(llex) {
		return
	}
	if !ck.checkMainPart(llex) {
		return
	}

	// Fourth paragraph: Cleanup, corresponding to the first paragraph.
	if !llex.SkipTextOrWarn("BUILDLINK_TREE+=\t-" + ck.pkgbase) {
		return
	}

	if !llex.EOF() {
		llex.CurrentLine().Warnf("The file should end here.")
	}

	pkg := ck.mklines.pkg
	if pkg != nil {
		pkg.checkLinesBuildlink3Inclusion(mklines)
	}

	mklines.SaveAutofixChanges()
}

func (ck *Buildlink3Checker) checkFirstParagraph(mlex *MkLinesLexer) bool {

	for mlex.SkipPrefix("#") {
	}

	// First paragraph: Introduction of the package identifier
	m := mlex.NextRegexp(`^BUILDLINK_TREE\+=[\t ]*([^\t ]+)$`)
	if m == nil {
		mlex.CurrentLine().Warnf("Expected a BUILDLINK_TREE line.")
		return false
	}

	pkgbase := m[1]
	pkgbaseLine := mlex.PreviousMkLine()

	if containsExpr(pkgbase) {
		ck.checkExprInPkgbase(pkgbaseLine)
	}

	ck.checkUniquePkgbase(pkgbase, pkgbaseLine)

	mlex.SkipEmptyOrNote()
	ck.pkgbase = pkgbase
	if pkg := ck.mklines.pkg; pkg != nil {
		pkg.buildlinkID = ck.pkgbase
	}
	ck.pkgbaseLine = pkgbaseLine
	return true
}

func (ck *Buildlink3Checker) checkUniquePkgbase(pkgbase string, mkline *MkLine) {
	prev := G.InterPackage.Bl3(pkgbase, &mkline.Location)
	if prev == nil {
		return
	}

	dirname := G.Pkgsrc.Rel(mkline.Filename().Dir()).Base()
	base, name := trimCommon(pkgbase, dirname.String())
	if base == "" && matches(name, `^(\d*|-cvs|-fossil|-git|-hg|-svn|-devel|-snapshot)$`) {
		return
	}

	mkline.Errorf("Duplicate package identifier %q already appeared in %s.",
		pkgbase, mkline.RelLocation(*prev))
	mkline.Explain(
		"Each buildlink3.mk file must have a unique identifier.",
		"These identifiers are used for multiple-inclusion guards,",
		"and using the same identifier for different packages",
		"(often by copy-and-paste) may change the dependencies",
		"of a package in subtle and unexpected ways.")
}

// checkSecondParagraph checks the multiple inclusion protection and
// introduces the uppercase package identifier.
func (ck *Buildlink3Checker) checkSecondParagraph(mlex *MkLinesLexer) bool {
	pkgbase := ck.pkgbase
	m := mlex.NextRegexp(`^\.if !defined\(([^\t ]+)_BUILDLINK3_MK\)$`)
	if m == nil {
		return false
	}
	pkgupperLine, pkgupper := mlex.PreviousMkLine(), m[1]

	if !mlex.SkipTextOrWarn(pkgupper + "_BUILDLINK3_MK:=") {
		return false
	}
	mlex.SkipEmptyOrNote()

	// See pkgtools/createbuildlink/files/createbuildlink, keyword PKGUPPER
	ucPkgbase := strings.ToUpper(strings.Replace(pkgbase, "-", "_", -1))
	if ucPkgbase != pkgupper && !containsExpr(pkgbase) {
		pkgupperLine.Errorf("Package name mismatch between multiple-inclusion guard %q (expected %q) and package name %q (from %s).",
			pkgupper, ucPkgbase, pkgbase, pkgupperLine.RelMkLine(ck.pkgbaseLine))
	}
	ck.checkPkgbaseMismatch(pkgbase)

	return true
}

func (ck *Buildlink3Checker) checkPkgbaseMismatch(bl3base string) {
	pkg := ck.mklines.pkg
	if pkg == nil {
		return
	}

	mkbase := pkg.EffectivePkgbase
	if mkbase == "" || mkbase == bl3base || strings.TrimPrefix(mkbase, "lib") == bl3base {
		return
	}

	if hasPrefix(mkbase, bl3base) && matches(mkbase[len(bl3base):], `^\d+$`) {
		return
	}

	ck.pkgbaseLine.Errorf("Package name mismatch between %q in this file and %q from %s.",
		bl3base, mkbase, ck.pkgbaseLine.RelMkLine(pkg.EffectivePkgnameLine))
}

// Third paragraph: Package information.
func (ck *Buildlink3Checker) checkMainPart(mlex *MkLinesLexer) bool {
	pkgbase := ck.pkgbase

	// The first .if is from the second paragraph.
	indentLevel := 1

	for !mlex.EOF() && indentLevel > 0 {
		mkline := mlex.CurrentMkLine()
		mlex.Skip()

		switch {
		case mkline.IsVarassign():
			ck.checkVarassign(mkline, pkgbase)

		case mkline.IsDirective() && mkline.Directive() == "if":
			indentLevel++

		case mkline.IsDirective() && mkline.Directive() == "endif":
			indentLevel--
		}

		mkline.ForEachUsed(func(expr *MkExpr, time EctxTime) {
			ck.checkExpr(expr, mkline)
		})
	}

	if indentLevel > 0 {
		return false
	}

	if ck.apiLine == nil {
		mlex.CurrentLine().Warnf("Definition of BUILDLINK_API_DEPENDS is missing.")
	}
	mlex.SkipEmptyOrNote()
	return true
}

func (ck *Buildlink3Checker) checkExpr(expr *MkExpr, mkline *MkLine) {
	varname := expr.varname
	if varname == "PKG_OPTIONS" {
		mkline.Errorf("PKG_OPTIONS is not available in buildlink3.mk files.")
		mkline.Explain(
			"The buildlink3.mk file of a package is only ever included",
			"by other packages, never by the package itself.",
			"Therefore, it does not make sense to use the variable PKG_OPTIONS",
			"in this place since it contains the package options of a random",
			"package that happens to include this file.",
			"",
			"To access the options of this package, see mk/pkg-build-options.mk.")
	}

	if varnameBase(varname) == "PKG_BUILD_OPTIONS" {
		param := varnameParam(varname)
		if param != "" && param != ck.pkgbase {
			mkline.Warnf("Wrong PKG_BUILD_OPTIONS, expected %q instead of %q.",
				ck.pkgbase, param)
			mkline.Explain(
				"The variable parameter for PKG_BUILD_OPTIONS must correspond",
				"to the value of \"pkgbase\" above.")
		}
	}
}

func (ck *Buildlink3Checker) checkVarassign(mkline *MkLine, pkgbase string) {
	varname, value := mkline.Varname(), mkline.Value()
	doCheck := false

	if varname == "BUILDLINK_ABI_DEPENDS."+pkgbase {
		ck.abiLine = mkline
		parser := NewMkParser(nil, value)
		pattern := ParsePackagePattern(parser)
		if pattern != nil && parser.EOF() {
			ck.abi = pattern
			doCheck = true
		}
	}

	if varname == "BUILDLINK_API_DEPENDS."+pkgbase {
		ck.apiLine = mkline
		parser := NewMkParser(nil, value)
		pattern := ParsePackagePattern(parser)
		if pattern != nil && parser.EOF() {
			ck.api = pattern
			doCheck = true
		}
	}

	if doCheck && ck.abi != nil && ck.api != nil &&
		ck.abi.Pkgbase != ck.api.Pkgbase {
		ck.abiLine.Warnf("Package name mismatch between ABI %q and API %q (from %s).",
			ck.abi.Pkgbase, ck.api.Pkgbase, ck.abiLine.RelMkLine(ck.apiLine))
	}

	if doCheck && ck.abi != nil && ck.api != nil &&
		ck.abi.Lower != "" && ck.api.Lower != "" &&
		!containsExpr(ck.abi.Lower) && !containsExpr(ck.api.Lower) &&
		pkgver.Compare(ck.abi.Lower, ck.api.Lower) < 0 {
		ck.abiLine.Warnf("ABI version %q should be at least API version %q (see %s).",
			ck.abi.Lower, ck.api.Lower, ck.abiLine.RelMkLine(ck.apiLine))
	}

	if varparam := mkline.Varparam(); varparam != "" && varparam != pkgbase {
		if hasPrefix(varname, "BUILDLINK_") && mkline.Varcanon() != "BUILDLINK_API_DEPENDS.*" {
			mkline.Warnf("Only buildlink variables for %q, not %q may be set in this file.", pkgbase, varparam)
		}
	}

	if varname == "pkgbase" && value != ck.pkgbase {
		mkline.Errorf("A buildlink3.mk file must only query its own PKG_BUILD_OPTIONS.%s, not PKG_BUILD_OPTIONS.%s.",
			ck.pkgbase, value)
	}

	ck.checkVarassignPkgsrcdir(mkline, pkgbase, varname, value)
}

func (ck *Buildlink3Checker) checkVarassignPkgsrcdir(
	mkline *MkLine,
	pkgbase string,
	varname string,
	value string,
) {

	if varname != "BUILDLINK_PKGSRCDIR."+pkgbase {
		return
	}
	if containsExpr(value) {
		return
	}

	pkgdir := mkline.Filename().Dir()
	expected := "../../" + G.Pkgsrc.Rel(pkgdir).String()
	if value == expected {
		return
	}

	mkline.Errorf("%s must be set to the package's own path (%s), not %s.",
		varname, expected, value)
}

func (ck *Buildlink3Checker) checkExprInPkgbase(pkgbaseLine *MkLine) {
	tokens, _ := pkgbaseLine.ValueTokens()
	for _, token := range tokens {
		if token.Expr == nil {
			continue
		}

		replacement := ""
		switch token.Expr.varname {
		case "PYPKGPREFIX":
			replacement = "py"
		case "RUBY_BASE", "RUBY_PKGPREFIX":
			replacement = "ruby"
		case "PHP_PKG_PREFIX":
			replacement = "php"
		}

		if replacement != "" {
			pkgbaseLine.Warnf("Use %q instead of %q (also in other variables in this file).",
				replacement, token.Text)
		} else {
			pkgbaseLine.Warnf(
				"Replace %q with a simple string (also in other variables in this file).",
				token.Text)
		}

		pkgbaseLine.Explain(
			"The identifiers in the BUILDLINK_TREE variable should be plain",
			"strings that do not refer to any variable.",
			"",
			"Even for packages that depend on a specific version of a",
			"programming language, the plain name is enough since",
			"the version number of the programming language is stored elsewhere.",
			"Furthermore, these package identifiers are only used at build time,",
			"after the specific version has been decided.")
	}
}

type Buildlink3Data struct {
	id             Buildlink3ID
	prefix         Path
	pkgsrcdir      PackagePath
	apiDepends     *PackagePattern
	apiDependsLine *MkLine
	abiDepends     *PackagePattern
	abiDependsLine *MkLine
}

// Buildlink3ID is the identifier that is used in the BUILDLINK_TREE
// for referring to a dependent package.
//
// It almost uniquely identifies a package.
// Packages that are alternatives to each other may use the same identifier.
type Buildlink3ID string

func LoadBuildlink3Data(mklines *MkLines) *Buildlink3Data {
	var data Buildlink3Data

	mklines.ForEach(func(mkline *MkLine) {
		if !mkline.IsVarassign() {
			return
		}

		varname := mkline.Varname()
		varbase := varnameBase(varname)
		varid := Buildlink3ID(varnameParam(varname))

		if varname == "BUILDLINK_TREE" {
			value := mkline.Value()
			if !hasPrefix(value, "-") {
				data.id = Buildlink3ID(mkline.Value())
			}
		}

		if varbase == "BUILDLINK_API_DEPENDS" && varid == data.id {
			p := NewMkParser(nil, mkline.Value())
			pattern := ParsePackagePattern(p)
			if pattern != nil && p.EOF() {
				data.apiDepends = pattern
				data.apiDependsLine = mkline
			}
		}

		if varbase == "BUILDLINK_ABI_DEPENDS" && varid == data.id {
			p := NewMkParser(nil, mkline.Value())
			pattern := ParsePackagePattern(p)
			if pattern != nil && p.EOF() {
				data.abiDepends = pattern
				data.abiDependsLine = mkline
			}
		}

		if varbase == "BUILDLINK_PREFIX" && varid == data.id {
			data.prefix = NewPath(mkline.Value())
		}
		if varbase == "BUILDLINK_PKGSRCDIR" && varid == data.id {
			data.pkgsrcdir = NewPackagePathString(mkline.Value())
		}
	})

	if data.id != "" {
		return &data
	}
	return nil
}

func isBuildlink3Guard(mkline *MkLine) bool {
	if mkline.Basename == "buildlink3.mk" && mkline.NeedsCond() {
		cond := mkline.Cond()
		if cond != nil && cond.Not != nil && hasSuffix(cond.Not.Defined, "_MK") {
			return true
		}
	}
	return false
}
