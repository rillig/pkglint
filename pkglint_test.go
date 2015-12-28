package main

import (
	"strings"

	check "gopkg.in/check.v1"
)

func (s *Suite) TestDetermineUsedVariables_simple(c *check.C) {
	mklines := s.NewMkLines("fname",
		"\t${VAR}")
	mkline := mklines.mklines[0]
	G.mk = mklines

	mklines.determineUsedVariables()

	c.Check(len(mklines.varuse), equals, 1)
	c.Check(mklines.varuse["VAR"], equals, mkline)
}

func (s *Suite) TestDetermineUsedVariables_nested(c *check.C) {
	mklines := s.NewMkLines("fname",
		"\t${outer.${inner}}")
	mkline := mklines.mklines[0]
	G.mk = mklines

	mklines.determineUsedVariables()

	c.Check(len(mklines.varuse), equals, 3)
	c.Check(mklines.varuse["inner"], equals, mkline)
	c.Check(mklines.varuse["outer."], equals, mkline)
	c.Check(mklines.varuse["outer.*"], equals, mkline)
}

func (s *Suite) TestResolveVariableRefs_CircularReference(c *check.C) {
	mkline := NewMkLine(NewLine("fname", 1, "GCC_VERSION=${GCC_VERSION}", nil))
	G.pkg = NewPackage(".")
	G.pkg.vardef["GCC_VERSION"] = mkline

	resolved := resolveVariableRefs("gcc-${GCC_VERSION}")

	c.Check(resolved, equals, "gcc-${GCC_VERSION}")
}

func (s *Suite) TestResolveVariableRefs_Multilevel(c *check.C) {
	mkline1 := NewMkLine(NewLine("fname", 10, "_=${SECOND}", nil))
	mkline2 := NewMkLine(NewLine("fname", 11, "_=${THIRD}", nil))
	mkline3 := NewMkLine(NewLine("fname", 12, "_=got it", nil))
	G.pkg = NewPackage(".")
	defineVar(mkline1, "FIRST")
	defineVar(mkline2, "SECOND")
	defineVar(mkline3, "THIRD")

	resolved := resolveVariableRefs("you ${FIRST}")

	c.Check(resolved, equals, "you got it")
}

func (s *Suite) TestResolveVariableRefs_SpecialChars(c *check.C) {
	mkline := NewMkLine(NewLine("fname", 10, "_=x11", nil))
	G.pkg = NewPackage("category/pkg")
	G.pkg.vardef["GST_PLUGINS0.10_TYPE"] = mkline

	resolved := resolveVariableRefs("gst-plugins0.10-${GST_PLUGINS0.10_TYPE}/distinfo")

	c.Check(resolved, equals, "gst-plugins0.10-x11/distinfo")
}

func (s *Suite) TestChecklineRcsid(c *check.C) {
	lines := s.NewLines("fname",
		"$"+"NetBSD: dummy $",
		"$"+"NetBSD$",
		"$"+"Id: dummy $",
		"$"+"Id$",
		"$"+"FreeBSD$")

	for _, line := range lines {
		checklineRcsid(line, ``, "")
	}

	c.Check(s.Output(), equals, ""+
		"ERROR: fname:3: Expected \"$"+"NetBSD$\".\n"+
		"ERROR: fname:4: Expected \"$"+"NetBSD$\".\n"+
		"ERROR: fname:5: Expected \"$"+"NetBSD$\".\n")
}

func (s *Suite) TestMatchVarassign(c *check.C) {
	checkVarassign := func(text string, ck check.Checker, varname, op, value, comment string) {
		type va struct {
			varname, op, value, comment string
		}
		expected := va{varname, op, value, comment}
		am, avarname, aop, avalue, acomment := matchVarassign(text)
		if !am {
			c.Errorf("Text %q doesn’t match variable assignment", text)
			return
		}
		actual := va{avarname, aop, avalue, acomment}
		c.Check(actual, ck, expected)
	}
	checkNotVarassign := func(text string) {
		m, _, _, _, _ := matchVarassign(text)
		if m {
			c.Errorf("Text %q matches variable assignment, but shouldn’t.", text)
		}
	}

	checkVarassign("C++=c11", equals, "C+", "+=", "c11", "")
	checkVarassign("V=v", equals, "V", "=", "v", "")
	checkVarassign("VAR=#comment", equals, "VAR", "=", "", "#comment")
	checkVarassign("VAR=\\#comment", equals, "VAR", "=", "#comment", "")
	checkVarassign("VAR=\\\\\\##comment", equals, "VAR", "=", "\\\\#", "#comment")
	checkVarassign("VAR=\\", equals, "VAR", "=", "\\", "")
	checkVarassign("VAR += value", equals, "VAR", "+=", "value", "")
	checkVarassign(" VAR=value", equals, "VAR", "=", "value", "")
	checkNotVarassign("\tVAR=value")
	checkNotVarassign("?=value")
	checkNotVarassign("<=value")
}

func (s *Suite) TestPackage_LoadPackageMakefile(c *check.C) {
	makefile := s.CreateTmpFile(c, "category/package/Makefile", ""+
		"# $"+"NetBSD$\n"+
		"\n"+
		"PKGNAME=pkgname-1.67\n"+
		"DISTNAME=distfile_1_67\n"+
		".include \"../../category/package/Makefile\"\n")
	pkg := NewPackage("category/package")
	G.currentDir = s.tmpdir + "/category/package"
	G.curPkgsrcdir = "../.."
	G.pkg = pkg

	pkg.loadPackageMakefile(makefile)

	c.Check(s.OutputCleanTmpdir(), equals, "")
}

func (s *Suite) TestChecklinesDescr(c *check.C) {
	lines := s.NewLines("DESCR",
		strings.Repeat("X", 90),
		"", "", "", "", "", "", "", "", "10",
		"Try ${PREFIX}",
		"", "", "", "", "", "", "", "", "20",
		"", "", "", "", "", "", "", "", "", "30")

	checklinesDescr(lines)

	c.Check(s.Output(), equals, ""+
		"WARN: DESCR:1: Line too long (should be no more than 80 characters).\n"+
		"NOTE: DESCR:11: Variables are not expanded in the DESCR file.\n"+
		"WARN: DESCR:25: File too long (should be no more than 24 lines).\n")
}

func (s *Suite) TestChecklinesMessage_short(c *check.C) {
	lines := s.NewLines("MESSAGE",
		"one line")

	checklinesMessage(lines)

	c.Check(s.Output(), equals, "WARN: MESSAGE:1: File too short.\n")
}

func (s *Suite) TestChecklinesMessage_malformed(c *check.C) {
	lines := s.NewLines("MESSAGE",
		"1",
		"2",
		"3",
		"4",
		"5")

	checklinesMessage(lines)

	c.Check(s.Output(), equals, ""+
		"WARN: MESSAGE:1: Expected a line of exactly 75 \"=\" characters.\n"+
		"ERROR: MESSAGE:2: Expected \"$"+"NetBSD$\".\n"+
		"WARN: MESSAGE:5: Expected a line of exactly 75 \"=\" characters.\n")
}

func (s *Suite) TestParseDependency(c *check.C) {

	testDependency := func(pattern string, expected DependencyPattern) {
		repl := NewPrefixReplacer(pattern)
		dp := ParseDependency(repl)
		if c.Check(dp, check.NotNil) {
			c.Check(*dp, equals, expected)
			c.Check(repl.rest, equals, "")
		}
	}

	testDependency("fltk>=1.1.5rc1<1.3", DependencyPattern{"fltk", ">=", "1.1.5rc1", "<", "1.3", ""})
	testDependency("libwcalc-1.0*", DependencyPattern{"libwcalc", "", "", "", "", "1.0*"})
	testDependency("${PHP_PKG_PREFIX}-pdo-5.*", DependencyPattern{"${PHP_PKG_PREFIX}-pdo", "", "", "", "", "5.*"})
	testDependency("${PYPKGPREFIX}-metakit-[0-9]*", DependencyPattern{"${PYPKGPREFIX}-metakit", "", "", "", "", "[0-9]*"})
	testDependency("boost-build-1.59.*", DependencyPattern{"boost-build", "", "", "", "", "1.59.*"})
	testDependency("${_EMACS_REQD}", DependencyPattern{"${_EMACS_REQD}", "", "", "", "", ""})
	testDependency("{gcc46,gcc46-libs}>=4.6.0", DependencyPattern{"{gcc46,gcc46-libs}", ">=", "4.6.0", "", "", ""})
	testDependency("perl5-*", DependencyPattern{"perl5", "", "", "", "", "*"})
	testDependency("verilog{,-current}-[0-9]*", DependencyPattern{"verilog{,-current}", "", "", "", "", "[0-9]*"})
	testDependency("mpg123{,-esound,-nas}>=0.59.18", DependencyPattern{"mpg123{,-esound,-nas}", ">=", "0.59.18", "", "", ""})
	testDependency("mysql*-{client,server}-[0-9]*", DependencyPattern{"mysql*-{client,server}", "", "", "", "", "[0-9]*"})
	testDependency("postgresql8[0-35-9]-${module}-[0-9]*", DependencyPattern{"postgresql8[0-35-9]-${module}", "", "", "", "", "[0-9]*"})
	testDependency("ncurses-${NC_VERS}{,nb*}", DependencyPattern{"ncurses", "", "", "", "", "${NC_VERS}{,nb*}"})
	// "{ssh{,6}-[0-9]*,openssh-[0-9]*}" is not representable using the current data structure
}

func (s *Suite) TestParsePkgbasePattern(c *check.C) {
	test := func(pattern, expected, rest string) {
		repl := NewPrefixReplacer(pattern)
		actual := ParsePkgbasePattern(repl)
		c.Check(actual, equals, expected)
		c.Check(repl.rest, equals, rest)
	}

	test("fltk", "fltk", "")
	test("fltk|", "fltk", "|")
	test("boost-build-1.59.*", "boost-build", "-1.59.*")
	test("${PHP_PKG_PREFIX}-pdo-5.*", "${PHP_PKG_PREFIX}-pdo", "-5.*")
	test("${PYPKGPREFIX}-metakit-[0-9]*", "${PYPKGPREFIX}-metakit", "-[0-9]*")
}
