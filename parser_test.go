package main

import (
	check "gopkg.in/check.v1"
)

func (s *Suite) TestParser_PkgbasePattern(c *check.C) {
	test := func(pattern, expected, rest string) {
		parser := NewParser(pattern)
		actual := parser.PkgbasePattern()
		c.Check(actual, equals, expected)
		c.Check(parser.Rest(), equals, rest)
	}

	test("fltk", "fltk", "")
	test("fltk|", "fltk", "|")
	test("boost-build-1.59.*", "boost-build", "-1.59.*")
	test("${PHP_PKG_PREFIX}-pdo-5.*", "${PHP_PKG_PREFIX}-pdo", "-5.*")
	test("${PYPKGPREFIX}-metakit-[0-9]*", "${PYPKGPREFIX}-metakit", "-[0-9]*")
}

func (s *Suite) TestParser_Dependency(c *check.C) {

	testDependencyRest := func(pattern string, expected DependencyPattern, rest string) {
		parser := NewParser(pattern)
		dp := parser.Dependency()
		if c.Check(dp, check.NotNil) {
			c.Check(*dp, equals, expected)
			c.Check(parser.Rest(), equals, rest)
		}
	}
	testDependency := func(pattern string, expected DependencyPattern) {
		testDependencyRest(pattern, expected, "")
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
	testDependency("xulrunner10>=${MOZ_BRANCH}${MOZ_BRANCH_MINOR}", DependencyPattern{"xulrunner10", ">=", "${MOZ_BRANCH}${MOZ_BRANCH_MINOR}", "", "", ""})
	testDependencyRest("gnome-control-center>=2.20.1{,nb*}", DependencyPattern{"gnome-control-center", ">=", "2.20.1", "", "", ""}, "{,nb*}")
	// "{ssh{,6}-[0-9]*,openssh-[0-9]*}" is not representable using the current data structure
}

func (s *Suite) TestParser_MkTokens(c *check.C) {
	test := func(input string, expectedTokens []*MkToken, expectedRest string) {
		p := NewParser(input)
		actualTokens := p.MkTokens()
		c.Check(actualTokens, deepEquals, expectedTokens)
		for i, expectedToken := range expectedTokens {
			if i < len(actualTokens) {
				c.Check(*actualTokens[i], deepEquals, *expectedToken)
			}
		}
		c.Check(p.Rest(), equals, expectedRest)
	}
	token := func(input string, expectedToken MkToken) {
		test(input, []*MkToken{&expectedToken}, "")
	}
	literal := func(literal string) MkToken {
		return MkToken{literal: literal}
	}
	varuse := func(varname string, modifiers ...string) MkToken {
		return MkToken{varuse: MkVarUse{varname: varname, modifiers: modifiers}}
	}

	token("literal", literal("literal"))
	token("\\/share\\/ { print \"share directory\" }", literal("\\/share\\/ { print \"share directory\" }"))
	token("${VARIABLE}", varuse("VARIABLE"))
	token("${VARIABLE.param}", varuse("VARIABLE.param"))
	token("${VARIABLE.${param}}", varuse("VARIABLE.${param}"))
	token("${VARIABLE.hicolor-icon-theme}", varuse("VARIABLE.hicolor-icon-theme"))
	token("${VARIABLE.gtk+extra}", varuse("VARIABLE.gtk+extra"))
	token("${VARIABLE:S/old/new/}", varuse("VARIABLE", "S/old/new/"))
	token("${GNUSTEP_LFLAGS:S/-L//g}", varuse("GNUSTEP_LFLAGS", "S/-L//g"))
	token("${SUSE_VERSION:S/.//}", varuse("SUSE_VERSION", "S/.//"))
	token("${MASTER_SITE_GNOME:=sources/alacarte/0.13/}", varuse("MASTER_SITE_GNOME", "=sources/alacarte/0.13/"))
	token("${INCLUDE_DIRS:H:T}", varuse("INCLUDE_DIRS", "H", "T"))
	token("${A.${B.${C.${D}}}}", varuse("A.${B.${C.${D}}}"))
	token("${RUBY_VERSION:C/([0-9]+)\\.([0-9]+)\\.([0-9]+)/\\1/}", varuse("RUBY_VERSION", "C/([0-9]+)\\.([0-9]+)\\.([0-9]+)/\\1/"))
}
