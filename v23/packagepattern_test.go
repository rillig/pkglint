package pkglint

import "gopkg.in/check.v1"

func (s *Suite) Test_ParsePackagePattern(c *check.C) {
	t := s.Init(c)

	testRest := func(pattern string, expected PackagePattern, rest string) {
		parser := NewMkParser(nil, pattern)
		dp := ParsePackagePattern(parser)
		if t.CheckNotNil(dp) {
			t.CheckEquals(*dp, expected)
			t.CheckEquals(parser.Rest(), rest)
		}
	}

	testNil := func(pattern string) {
		parser := NewMkParser(nil, pattern)
		dp := ParsePackagePattern(parser)
		if t.CheckNil(dp) {
			t.CheckEquals(parser.Rest(), pattern)
		}
	}

	test := func(pattern string, expected PackagePattern) {
		testRest(pattern, expected, "")
	}

	test("pkgbase>=1.0",
		PackagePattern{"pkgbase", ">=", "1.0", "", "", ""})

	test("pkgbase>1.0",
		PackagePattern{"pkgbase", ">", "1.0", "", "", ""})

	test("pkgbase<=1.0",
		PackagePattern{"pkgbase", "", "", "<=", "1.0", ""})

	test("pkgbase<1.0",
		PackagePattern{"pkgbase", "", "", "<", "1.0", ""})

	test("fltk>=1.1.5rc1<1.3",
		PackagePattern{"fltk", ">=", "1.1.5rc1", "<", "1.3", ""})

	test("libwcalc-1.0*",
		PackagePattern{"libwcalc", "", "", "", "", "1.0*"})

	test("${PHP_PKG_PREFIX}-pdo-5.*",
		PackagePattern{"${PHP_PKG_PREFIX}-pdo", "", "", "", "", "5.*"})

	test("${PYPKGPREFIX}-metakit-[0-9]*",
		PackagePattern{"${PYPKGPREFIX}-metakit", "", "", "", "", "[0-9]*"})

	test("boost-build-1.59.*",
		PackagePattern{"boost-build", "", "", "", "", "1.59.*"})

	test("${_EMACS_REQD}",
		PackagePattern{"${_EMACS_REQD}", "", "", "", "", ""})

	test("{gcc46,gcc46-libs}>=4.6.0",
		PackagePattern{"{gcc46,gcc46-libs}", ">=", "4.6.0", "", "", ""})

	test("perl5-*",
		PackagePattern{"perl5", "", "", "", "", "*"})

	test("verilog{,-current}-[0-9]*",
		PackagePattern{"verilog{,-current}", "", "", "", "", "[0-9]*"})

	test("mpg123{,-esound,-nas}>=0.59.18",
		PackagePattern{"mpg123{,-esound,-nas}", ">=", "0.59.18", "", "", ""})

	test("mysql*-{client,server}-[0-9]*",
		PackagePattern{"mysql*-{client,server}", "", "", "", "", "[0-9]*"})

	test("postgresql8[0-35-9]-${module}-[0-9]*",
		PackagePattern{"postgresql8[0-35-9]-${module}", "", "", "", "", "[0-9]*"})

	test("ncurses-${NC_VERS}{,nb*}",
		PackagePattern{"ncurses", "", "", "", "", "${NC_VERS}{,nb*}"})

	test("xulrunner10>=${MOZ_BRANCH}${MOZ_BRANCH_MINOR}",
		PackagePattern{"xulrunner10", ">=", "${MOZ_BRANCH}${MOZ_BRANCH_MINOR}", "", "", ""})

	test("${_EMACS_CONFLICTS.${_EMACS_FLAVOR}}",
		PackagePattern{"${_EMACS_CONFLICTS.${_EMACS_FLAVOR}}", "", "", "", "", ""})

	test("${DISTNAME:S/gnome-vfs/gnome-vfs2-${GNOME_VFS_NAME}/}",
		PackagePattern{"${DISTNAME:S/gnome-vfs/gnome-vfs2-${GNOME_VFS_NAME}/}", "", "", "", "", ""})

	// FIXME
	testRest("${LUA_PKGPREFIX}-std-_debug-[0-9]*",
		PackagePattern{"${LUA_PKGPREFIX}-std", "", "", "", "", "_debug"},
		"-[0-9]*")

	// FIXME
	testRest("rt-*-[0-9]*",
		PackagePattern{"rt", "", "", "", "", "*"},
		"-[0-9]*")

	testRest("gnome-control-center>=2.20.1{,nb*}",
		PackagePattern{"gnome-control-center", ">=", "2.20.1", "", "", ""},
		"{,nb*}")

	testRest("R-jsonlite>=0.9.6*",
		PackagePattern{"R-jsonlite", ">=", "0.9.6", "", "", ""},
		"*")

	testRest("tex-pst-3d-[0-9]*",
		PackagePattern{"tex-pst", "", "", "", "", "3d"},
		"-[0-9]*")

	testRest("font-adobe-100dpi-[0-9]*",
		PackagePattern{"font-adobe", "", "", "", "", "100dpi"},
		"-[0-9]*")

	testNil("pkgbase")

	testNil("pkgbase-")

	testNil("pkgbase-client")

	testNil(">=2.20.1{,nb*}")

	testNil("pkgbase<=")

	// Package patterns with curly braces are handled by expandCurlyBraces.
	testNil("{ezmlm>=0.53,ezmlm-idx>=0.40}")
	testNil("{mecab-ipadic>=2.7.0,mecab-jumandic>=5.1}")
	testNil("{samba>=2.0,ja-samba>=2.0}")
	testNil("{ssh{,6}-[0-9]*,openssh-[0-9]*}")
}

func (s *Suite) Test_PackagePatternChecker_Check(c *check.C) {
	vt := NewVartypeCheckTester(s.Init(c), BtPackagePattern)

	vt.Varname("CONFLICTS")
	vt.Op(opAssignAppend)

	// alternative patterns, using braces or brackets
	vt.Values(
		"mpg123{,-esound,-nas}>=0.59.18",
		"seamonkey-{,-bin,-gtk1}<2.0",
		"mysql*-{client,server}-[0-9]*",
		"{ssh{,6}-[0-9]*,openssh-[0-9]*}",
		"libao-[a-z]*-[0-9]*")
	vt.Output(
		"ERROR: filename.mk:2: Invalid dependency pattern \"seamonkey-<2.0\".",
		"ERROR: filename.mk:2: Invalid dependency pattern \"seamonkey--bin<2.0\".",
		"ERROR: filename.mk:2: Invalid dependency pattern \"seamonkey--gtk1<2.0\".")

	// expressions
	vt.Values(
		"{${NETSCAPE_PREFERRED:C/:/,/g}}-[0-9]*")
	vt.OutputEmpty()
}

func (s *Suite) Test_PackagePatternChecker_checkSingle(c *check.C) {
	vt := NewVartypeCheckTester(s.Init(c), BtPackagePattern)

	vt.Varname("CONFLICTS")
	vt.Op(opAssignAppend)

	// numeric version comparison operators
	vt.Values(
		"perl5>=5.22",
		"libkipi>=0.1.5<4.0",
		"gtk2+>=2.16")
	vt.OutputEmpty()

	// textual pattern matching
	vt.Values(
		"perl-5*",
		"perl5-*",
		"perl-5.22",
		"perl5-5.22.*",
		"gtksourceview-sharp-2.0-[0-9]*",
		"eboard-[0-9\\.]*")
	vt.Output(
		"WARN: filename.mk:11: Use \"5.*\" instead of \"5*\" as the version pattern.",
		"WARN: filename.mk:12: Use \"perl5-[0-9]*\" instead of \"perl5-*\".",
		"WARN: filename.mk:13: Use \"5.22{,nb*}\" instead of \"5.22\" as the version pattern.",
		"ERROR: filename.mk:15: Dependency pattern \"gtksourceview-sharp-2.0\" is followed by extra text \"-[0-9]*\".",
		"WARN: filename.mk:16: Only \"[0-9]*\" is allowed as the numeric part of a dependency, not \"[0-9\\.]*\".",
		"WARN: filename.mk:16: The version pattern \"[0-9\\.]*\" should not contain a hyphen.")

	// nb suffix
	vt.Values(
		"perl5-5.22.*{,nb*}",
		"perl-5.22{,nb*}",
		"perl-5.22{,nb[0-9]*}",
		"mbrola-301h{,nb[0-9]*}",
		"ncurses-${NC_VERS}{,nb*}",
		"gnome-control-center>=2.20.1{,nb*}",
		"gnome-control-center>=2.20.1{,nb[0-9]*}")
	vt.Output(
		"WARN: filename.mk:26: Dependency patterns of the form pkgbase>=1.0 don't need the \"{,nb*}\" extension.",
		"WARN: filename.mk:27: Dependency patterns of the form pkgbase>=1.0 don't need the \"{,nb*}\" extension.")

	// expressions
	vt.Values(
		"postgresql8[0-35-9]-${module}-[0-9]*",
		"${_EMACS_CONFLICTS.${_EMACS_FLAVOR}}",
		"${PYPKGPREFIX}-sqlite3",
		"${PYPKGPREFIX}-sqlite3-${VERSION}",
		"${PYPKGPREFIX}-sqlite3-${PYSQLITE_REQD}",
		"${PYPKGPREFIX}-sqlite3>=${PYSQLITE_REQD}",
		"${EMACS_PACKAGE}>=${EMACS_MAJOR}",

		// The "*" is ambiguous. It could either continue the PKGBASE or
		// start the version number.
		"${PKGNAME_NOREV:S/jdk/jre/}*",

		// The canonical form is "{,nb*}" instead of "{nb*,}".
		// Plus, mentioning nb* is not necessary when using >=.
		"dovecot>=${PKGVERSION_NOREV}{nb*,}",

		"oxygen-icons>=${KF5VER}{,nb[0-9]*}",

		// The following pattern should have "]*}" instead of "]}*".
		"ja-vflib-lib-${VFLIB_VERSION}{,nb[0-9]}*",

		// The following pattern uses both ">=" and "*", which doesn't make sense.
		"${PYPKGPREFIX}-sphinx>=1.2.3nb1*",

		// These patterns are valid, assuming that DISTNAME is a valid PKGNAME.
		"${DISTNAME}{,nb*}",
		"${DISTNAME:S/-/-base-/}{,nb[0-9]*}",
		"${RUBY_PKGPREFIX}-${DISTNAME}{,nb*}",

		// A base version may have trailing version parts.
		"atril>=${VERSION:R}.2")

	vt.Output(
		"ERROR: filename.mk:33: Invalid dependency pattern \"${PYPKGPREFIX}-sqlite3\".",
		"ERROR: filename.mk:38: Invalid dependency pattern \"${PKGNAME_NOREV:S/jdk/jre/}*\".",
		"WARN: filename.mk:39: The nb version part should have the form \"{,nb*}\" or \"{,nb[0-9]*}\", not \"{nb*,}\".",
		"WARN: filename.mk:40: Dependency patterns of the form pkgbase>=1.0 don't need the \"{,nb*}\" extension.",
		"WARN: filename.mk:41: The nb version part should have the form \"{,nb*}\" or \"{,nb[0-9]*}\", not \"{,nb[0-9]}\".",
		"ERROR: filename.mk:42: Dependency pattern \"${PYPKGPREFIX}-sphinx>=1.2.3nb1\" is followed by extra text \"*\".")

	// invalid dependency patterns
	vt.Values(
		"Perl",
		"py-docs",
		"perl5-[5.10-5.22]*",
		"package-1.0|garbage",
		"package>=1.0:../../category/package",
		"package-1.0>=1.0.3",
		// This should be regarded as invalid since the [a-z0-9] might either
		// continue the PKGBASE or start the version number.
		"${RUBY_PKGPREFIX}-theme-[a-z0-9]*",
		"package>=2.9.0,<3",
		"package>=2.16>=0")
	vt.Output(
		"ERROR: filename.mk:51: Invalid dependency pattern \"Perl\".",
		"ERROR: filename.mk:52: Invalid dependency pattern \"py-docs\".",
		"WARN: filename.mk:53: Only \"[0-9]*\" is allowed as the numeric part of a dependency, not \"[5.10-5.22]*\".",
		"WARN: filename.mk:53: The version pattern \"[5.10-5.22]*\" should not contain a hyphen.",
		"ERROR: filename.mk:54: Dependency pattern \"package-1.0\" is followed by extra text \"|garbage\".",
		"ERROR: filename.mk:55: Dependency pattern \"package>=1.0\" is followed by extra text \":../../category/package\".",
		// TODO: Mention that version numbers in a pkgbase must be appended directly, without hyphen.
		"ERROR: filename.mk:56: Dependency pattern \"package-1.0\" is followed by extra text \">=1.0.3\".",
		"ERROR: filename.mk:57: Invalid dependency pattern \"${RUBY_PKGPREFIX}-theme-[a-z0-9]*\".",
		"ERROR: filename.mk:58: Dependency pattern \"package>=2.9.0\" is followed by extra text \",<3\".",
		"ERROR: filename.mk:59: Dependency pattern \"package>=2.16\" is followed by extra text \">=0\".")
}

func (s *Suite) Test_PackagePatternChecker_checkDepends__smaller_version(c *check.C) {
	t := s.Init(c)

	t.SetUpPackage("category/package",
		"LIB_VERSION_SMALL=\t1.0",
		"LIB_VERSION_LARGE=\t10.0",
		"",
		".include \"../../category/lib/buildlink3.mk\"",
		"BUILDLINK_API_DEPENDS.lib+=\tlib>=1.0pkg",
		"BUILDLINK_API_DEPENDS.lib+=\tlib>=${LIB_VERSION_SMALL}",
		"BUILDLINK_API_DEPENDS.lib+=\tlib>=${LIB_VERSION_LARGE}",
		"BUILDLINK_ABI_DEPENDS.lib+=\tlib>=1.1pkg")
	t.SetUpPackage("category/lib")
	t.CreateFileBuildlink3("category/lib/buildlink3.mk",
		"BUILDLINK_API_DEPENDS.lib+=\tlib>=1.3api",
		"BUILDLINK_ABI_DEPENDS.lib+=\tlib>=1.4abi")
	t.Chdir("category/package")
	t.FinishSetUp()

	G.checkdirPackage(".")

	t.CheckOutputLines(
		"NOTE: Makefile:24: The requirement >=1.0pkg is already guaranteed "+
			"by the >=1.3api from ../../category/lib/buildlink3.mk:12.",
		"ERROR: Makefile:27: Packages must only require API versions, "+
			"not ABI versions of dependencies.",
		"NOTE: Makefile:27: The requirement >=1.1pkg is already guaranteed "+
			"by the >=1.4abi from ../../category/lib/buildlink3.mk:13.")
}

func (s *Suite) Test_PackagePatternChecker_checkDepends__different_operators(c *check.C) {
	t := s.Init(c)

	t.SetUpPackage("category/package",
		".include \"../../category/lib/buildlink3.mk\"",
		"BUILDLINK_API_DEPENDS.lib+=\tlib>=1.0pkg",
		"BUILDLINK_ABI_DEPENDS.lib+=\tlib>=1.1pkg")
	t.SetUpPackage("category/lib")
	t.CreateFileBuildlink3("category/lib/buildlink3.mk",
		"BUILDLINK_API_DEPENDS.lib+=\tlib>1.3api",
		"BUILDLINK_ABI_DEPENDS.lib+=\tlib>1.4abi")
	t.Chdir("category/package")
	t.FinishSetUp()

	G.checkdirPackage(".")

	t.CheckOutputLines(
		"NOTE: Makefile:21: The requirement >=1.0pkg is already guaranteed "+
			"by the >1.3api from ../../category/lib/buildlink3.mk:12.",
		"ERROR: Makefile:22: Packages must only require API versions, "+
			"not ABI versions of dependencies.",
		"NOTE: Makefile:22: The requirement >=1.1pkg is already guaranteed "+
			"by the >1.4abi from ../../category/lib/buildlink3.mk:13.")
}

func (s *Suite) Test_PackagePatternChecker_checkDepends__additional_greater(c *check.C) {
	t := s.Init(c)

	t.SetUpPackage("category/package",
		".include \"../../category/lib/buildlink3.mk\"",
		"BUILDLINK_API_DEPENDS.lib+=\tlib>1.0pkg",
		"BUILDLINK_ABI_DEPENDS.lib+=\tlib>1.1pkg")
	t.SetUpPackage("category/lib")
	t.CreateFileBuildlink3("category/lib/buildlink3.mk",
		"BUILDLINK_API_DEPENDS.lib+=\tlib>=1.3api",
		"BUILDLINK_ABI_DEPENDS.lib+=\tlib>=1.4abi")
	t.Chdir("category/package")
	t.FinishSetUp()

	G.checkdirPackage(".")

	t.CheckOutputLines(
		"NOTE: Makefile:21: The requirement >1.0pkg is already guaranteed "+
			"by the >=1.3api from ../../category/lib/buildlink3.mk:12.",
		"ERROR: Makefile:22: Packages must only require API versions, "+
			"not ABI versions of dependencies.",
		"NOTE: Makefile:22: The requirement >1.1pkg is already guaranteed "+
			"by the >=1.4abi from ../../category/lib/buildlink3.mk:13.")
}

func (s *Suite) Test_PackagePatternChecker_checkDepends__upper_limit(c *check.C) {
	t := s.Init(c)

	t.SetUpPackage("category/package",
		".include \"../../category/lib/buildlink3.mk\"",
		"BUILDLINK_API_DEPENDS.lib+=\tlib<2.0",
		"BUILDLINK_ABI_DEPENDS.lib+=\tlib<2.1")
	t.SetUpPackage("category/lib")
	t.CreateFileBuildlink3("category/lib/buildlink3.mk",
		"BUILDLINK_API_DEPENDS.lib+=\tlib>1.3api",
		"BUILDLINK_ABI_DEPENDS.lib+=\tlib>1.4abi")
	t.Chdir("category/package")
	t.FinishSetUp()

	G.checkdirPackage(".")

	// If the additional constraint doesn't have a lower bound,
	// there are no version numbers to compare and warn about.
	t.CheckOutputLines(
		"ERROR: Makefile:22: Packages must only require API versions, " +
			"not ABI versions of dependencies.")
}

// Having an upper bound for a library dependency is unusual.
// A combined lower and upper bound makes sense though.
func (s *Suite) Test_PackagePatternChecker_checkDepends__upper_limit_in_buildlink3(c *check.C) {
	t := s.Init(c)

	t.SetUpPackage("category/package",
		".include \"../../category/lib/buildlink3.mk\"",
		"BUILDLINK_API_DEPENDS.lib+=\tlib>=16",
		"BUILDLINK_ABI_DEPENDS.lib+=\tlib>=16.1")
	t.SetUpPackage("category/lib")
	t.CreateFileBuildlink3("category/lib/buildlink3.mk",
		"BUILDLINK_API_DEPENDS.lib+=\tlib<7",
		"BUILDLINK_ABI_DEPENDS.lib+=\tlib<6")
	t.Chdir("category/package")
	t.FinishSetUp()

	G.checkdirPackage(".")

	// If the additional constraint doesn't have a lower bound,
	// there are no version numbers to compare and warn about.
	t.CheckOutputLines(
		"ERROR: Makefile:22: Packages must only require API versions, " +
			"not ABI versions of dependencies.")
}

func (s *Suite) Test_PackagePatternChecker_checkDepends__API_ABI(c *check.C) {
	t := s.Init(c)

	t.SetUpPackage("category/package",
		".include \"../../category/lib/buildlink3.mk\"",
		"BUILDLINK_API_DEPENDS.lib+=\tlib>=1.0pkg",
		".include \"../../category/indirect/buildlink3.mk\"",
		"BUILDLINK_API_DEPENDS.indirect+=\tindirect>=${:U1.4api}")
	t.SetUpPackage("category/lib")
	t.CreateFileBuildlink3("category/lib/buildlink3.mk",
		"BUILDLINK_ABI_DEPENDS.lib+=\tlib>=1.4abi")
	t.CreateFileBuildlink3("category/indirect/buildlink3.mk",
		"BUILDLINK_API_DEPENDS.indirect+=\tindirect>=${:U2.0}",
		"BUILDLINK_ABI_DEPENDS.indirect+=\tindirect>=${:U2.2}")
	t.Chdir("category/package")
	t.FinishSetUp()

	G.checkdirPackage(".")

	t.CheckOutputEmpty()
}

func (s *Suite) Test_PackagePatternChecker_checkDepends__ABI_API(c *check.C) {
	t := s.Init(c)

	t.SetUpPackage("category/package",
		".include \"../../category/lib/buildlink3.mk\"",
		"BUILDLINK_ABI_DEPENDS.lib+=\tlib>=1.1pkg")
	t.SetUpPackage("category/lib")
	t.CreateFileBuildlink3("category/lib/buildlink3.mk",
		"BUILDLINK_API_DEPENDS.lib+=\tlib>=1.3api")
	t.Chdir("category/package")
	t.FinishSetUp()

	G.checkdirPackage(".")

	t.CheckOutputLines(
		"ERROR: Makefile:21: Packages must only require API versions, " +
			"not ABI versions of dependencies.")
}
