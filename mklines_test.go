package main

import (
	"gopkg.in/check.v1"
)

func (s *Suite) Test_MkLines_Check__autofix_conditional_indentation(c *check.C) {
	t := s.Init(c)

	t.SetupCommandLine("--autofix", "-Wspace")
	lines := t.SetupFileLines("fname.mk",
		MkRcsId,
		".if defined(A)",
		".for a in ${A}",
		".if defined(C)",
		".endif",
		".endfor",
		".endif")
	mklines := NewMkLines(lines)

	mklines.Check()

	t.CheckOutputLines(
		"AUTOFIX: ~/fname.mk:3: Replacing \".\" with \".  \".",
		"AUTOFIX: ~/fname.mk:4: Replacing \".\" with \".    \".",
		"AUTOFIX: ~/fname.mk:5: Replacing \".\" with \".    \".",
		"AUTOFIX: ~/fname.mk:6: Replacing \".\" with \".  \".")
	t.CheckFileLines("fname.mk",
		"# $"+"NetBSD$",
		".if defined(A)",
		".  for a in ${A}",
		".    if defined(C)",
		".    endif",
		".  endfor",
		".endif")
}

func (s *Suite) Test_MkLines_Check__unusual_target(c *check.C) {
	t := s.Init(c)

	mklines := t.NewMkLines("Makefile",
		MkRcsId,
		"",
		"echo: echo.c",
		"\tcc -o ${.TARGET} ${.IMPSRC}")

	mklines.Check()

	t.CheckOutputLines(
		"WARN: Makefile:3: Unusual target \"echo\".")
}

func (s *Suite) Test_MkLineChecker_checkInclude__Makefile(c *check.C) {
	t := s.Init(c)

	mkline := t.NewMkLine("Makefile", 2, ".include \"../../other/package/Makefile\"")

	MkLineChecker{mkline}.checkInclude()

	t.CheckOutputLines(
		"ERROR: Makefile:2: \"/other/package/Makefile\" does not exist.",
		"ERROR: Makefile:2: Other Makefiles must not be included directly.")
}

func (s *Suite) Test_MkLines_quoting_LDFLAGS_for_GNU_configure(c *check.C) {
	t := s.Init(c)

	t.SetupCommandLine("-Wall")
	G.globalData.InitVartypes()
	G.Pkg = NewPackage("category/pkgbase")
	mklines := t.NewMkLines("Makefile",
		MkRcsId,
		"GNU_CONFIGURE=\tyes",
		"CONFIGURE_ENV+=\tX_LIBS=${X11_LDFLAGS:Q}")

	mklines.Check()

	t.CheckOutputLines(
		"WARN: Makefile:3: Please use ${X11_LDFLAGS:M*:Q} instead of ${X11_LDFLAGS:Q}.",
		"WARN: Makefile:3: Please use ${X11_LDFLAGS:M*:Q} instead of ${X11_LDFLAGS:Q}.")
}

func (s *Suite) Test_MkLines__variable_alignment_advanced(c *check.C) {
	t := s.Init(c)

	t.SetupCommandLine("-Wspace")
	lines := t.SetupFileLinesContinuation("Makefile",
		MkRcsId,
		"",
		"VAR= \\", // In continuation lines, indenting with spaces is ok
		"\tvalue",
		"",
		"VAR= indented with one space",   // Exactly one space is ok in general
		"VAR=  indented with two spaces", // Two spaces are uncommon
		"",
		"BLOCK=\tindented with tab",          // To align these two lines, this line needs more more tab.
		"BLOCK_LONGVAR= indented with space", // This still keeps the indentation at an acceptable level.
		"",
		"BLOCK=\tshort",
		"BLOCK_LONGVAR=\tlong",
		"",
		"GRP_A= avalue", // The values in a block should be aligned
		"GRP_AA= value",
		"GRP_AAA= value",
		"GRP_AAAA= value",
		"",
		"VAR=\t${VAR}${BLOCK}${BLOCK_LONGVAR} # suppress warnings about unused variables",
		"VAR=\t${GRP_A}${GRP_AA}${GRP_AAA}${GRP_AAAA}")
	mklines := NewMkLines(lines)

	mklines.Check()

	t.CheckOutputLines(
		"NOTE: ~/Makefile:6: This variable value should be aligned with tabs, not spaces, to column 9.",
		"NOTE: ~/Makefile:7: This variable value should be aligned with tabs, not spaces, to column 9.",
		"NOTE: ~/Makefile:9: This variable value should be aligned to column 17.",
		"NOTE: ~/Makefile:10: This variable value should be aligned with tabs, not spaces, to column 17.",
		"NOTE: ~/Makefile:12: This variable value should be aligned to column 17.",
		"NOTE: ~/Makefile:15: This variable value should be aligned with tabs, not spaces, to column 17.",
		"NOTE: ~/Makefile:16: This variable value should be aligned with tabs, not spaces, to column 17.",
		"NOTE: ~/Makefile:17: This variable value should be aligned with tabs, not spaces, to column 17.",
		"NOTE: ~/Makefile:18: This variable value should be aligned with tabs, not spaces, to column 17.")

	t.SetupCommandLine("-Wspace", "--autofix")

	mklines.Check()

	t.CheckOutputLines(
		"AUTOFIX: ~/Makefile:6: Replacing \" \" with \"\\t\".",
		"AUTOFIX: ~/Makefile:7: Replacing \"  \" with \"\\t\".",
		"AUTOFIX: ~/Makefile:9: Replacing \"\\t\" with \"\\t\\t\".",
		"AUTOFIX: ~/Makefile:10: Replacing \" \" with \"\\t\".",
		"AUTOFIX: ~/Makefile:12: Replacing \"\\t\" with \"\\t\\t\".",
		"AUTOFIX: ~/Makefile:15: Replacing \" \" with \"\\t\\t\".",
		"AUTOFIX: ~/Makefile:16: Replacing \" \" with \"\\t\\t\".",
		"AUTOFIX: ~/Makefile:17: Replacing \" \" with \"\\t\".",
		"AUTOFIX: ~/Makefile:18: Replacing \" \" with \"\\t\".")
	t.CheckFileLines("Makefile",
		MkRcsId,
		"",
		"VAR= \\",
		"\tvalue",
		"",
		"VAR=\tindented with one space",
		"VAR=\tindented with two spaces",
		"",
		"BLOCK=\t\tindented with tab",
		"BLOCK_LONGVAR=\tindented with space",
		"",
		"BLOCK=\t\tshort",
		"BLOCK_LONGVAR=\tlong",
		"",
		"GRP_A=\t\tavalue",
		"GRP_AA=\t\tvalue",
		"GRP_AAA=\tvalue",
		"GRP_AAAA=\tvalue",
		"",
		"VAR=\t${VAR}${BLOCK}${BLOCK_LONGVAR} # suppress warnings about unused variables",
		"VAR=\t${GRP_A}${GRP_AA}${GRP_AAA}${GRP_AAAA}")
}

func (s *Suite) Test_MkLines__variable_alignment_space_and_tab(c *check.C) {
	t := s.Init(c)

	t.SetupCommandLine("-Wspace")
	mklines := t.NewMkLines("Makefile",
		MkRcsId,
		"",
		"VAR=    space",
		"VAR=\ttab ${VAR}")

	mklines.Check()

	t.CheckOutputLines(
		"NOTE: Makefile:3: Variable values should be aligned with tabs, not spaces.")
}

func (s *Suite) Test_MkLines__variable_alignment_outlier(c *check.C) {
	t := s.Init(c)

	t.SetupCommandLine("-Wspace")
	G.globalData.InitVartypes()
	lines := t.SetupFileLinesContinuation("Makefile",
		MkRcsId,
		"",
		"V= 3",  // Adjust from 3 to 8 (+ 1 tab)
		"V=\t4", // Keep at 8
		"",
		"V.0008= 6", // Keep at 8 (space to tab)
		"V=\t7",     // Keep at 8
		"",
		"V.00009= 9", // Adjust from 9 to 16 (+ 1 tab)
		"V=\t10",     // Adjust from 8 to 16 (+1 tab)
		"",
		"V.000000000016= 12", // Keep at 16 (space to tab)
		"V=\tvalue",          // Adjust from 8 to 16 (+ 1 tab)
		"",
		"V.0000000000017= 15", // Keep at 17 (outlier)
		"V=\tvalue",           // Keep at 8 (would require + 2 tabs)
		"",
		"V= 18",            // Adjust from 3 to 16 (+ 2 tabs)
		"V.000010=\tvalue", // Keep at 16
		"",
		"V.00009= 21",      // Adjust from 9 to 16 (+ 1 tab)
		"V.000010=\tvalue", // Keep at 16
		"",
		"V.000000000016= 24", // Keep at 16 (space to tab)
		"V.000010=\tvalue",   // Keep at 16
		"",
		"V.0000000000017= 27", // Adjust from 17 to 24 (+ 1 tab)
		"V.000010=\tvalue",    // Adjust from 16 to 24 (+ 1 tab)
		"",
		"V.0000000000000000023= 30", // Adjust from 23 to 24 (+ 1 tab)
		"V.000010=\tvalue",          // Adjust from 16 to 24 (+ 1 tab)
		"",
		"V.00000000000000000024= 33", // Keep at 24 (space to tab)
		"V.000010=\tvalue",           // Adjust from 16 to 24 (+ 1 tab)
		"",
		"V.000000000000000000025= 36", // Keep at 25 (outlier)
		"V.000010=\tvalue",            // Keep at 16 (would require + 2 tabs)
		"",
		"V.00008=\t39",          // Keep at 16
		"V.00008=\t\t\t\tvalue", // Adjust from 40 to 16 (removes 3 tabs)
		"",
		"V.00008=\t\t42",        // Adjust from 24 to 16 (removes 1 tab)
		"V.00008=\t\t\t\tvalue", // Adjust from 40 to 16 (removes 3 tabs)
		"",
		"X=\t${X} ${V} ${V.*}") // To avoid "defined but not used" warnings
	mklines := NewMkLines(lines)

	mklines.Check()

	t.CheckOutputLines(
		"NOTE: ~/Makefile:3: This variable value should be aligned with tabs, not spaces, to column 9.",
		"NOTE: ~/Makefile:6: Variable values should be aligned with tabs, not spaces.",
		"NOTE: ~/Makefile:9: This variable value should be aligned with tabs, not spaces, to column 17.",
		"NOTE: ~/Makefile:10: This variable value should be aligned to column 17.",
		"NOTE: ~/Makefile:12: Variable values should be aligned with tabs, not spaces.",
		"NOTE: ~/Makefile:13: This variable value should be aligned to column 17.",
		"NOTE: ~/Makefile:18: This variable value should be aligned with tabs, not spaces, to column 17.",
		"NOTE: ~/Makefile:21: This variable value should be aligned with tabs, not spaces, to column 17.",
		"NOTE: ~/Makefile:24: Variable values should be aligned with tabs, not spaces.",
		"NOTE: ~/Makefile:27: This variable value should be aligned with tabs, not spaces, to column 25.",
		"NOTE: ~/Makefile:28: This variable value should be aligned to column 25.",
		"NOTE: ~/Makefile:30: This variable value should be aligned with tabs, not spaces, to column 25.",
		"NOTE: ~/Makefile:31: This variable value should be aligned to column 25.",
		"NOTE: ~/Makefile:33: Variable values should be aligned with tabs, not spaces.",
		"NOTE: ~/Makefile:34: This variable value should be aligned to column 25.",
		"NOTE: ~/Makefile:40: This variable value should be aligned to column 17.",
		"NOTE: ~/Makefile:42: This variable value should be aligned to column 17.",
		"NOTE: ~/Makefile:43: This variable value should be aligned to column 17.")
}

func (s *Suite) Test_MkLines__variable_alignment__nospace(c *check.C) {
	t := s.Init(c)

	t.SetupCommandLine("-Wspace")
	G.globalData.InitVartypes()
	lines := t.SetupFileLinesContinuation("Makefile",
		MkRcsId,
		"PKG_FAIL_REASON+=\"Message\"")
	mklines := NewMkLines(lines)

	mklines.Check()

	t.CheckOutputLines(
		"NOTE: ~/Makefile:2: This variable value should be aligned to column 25.")
}

// Continuation lines without any content on the first line are ignored.
// Even when they appear in a paragraph of their own.
func (s *Suite) Test_MkLines__variable_alignment__continuation_lines(c *check.C) {
	t := s.Init(c)

	t.SetupCommandLine("-Wspace", "--autofix")
	G.globalData.InitVartypes()
	lines := t.SetupFileLinesContinuation("Makefile",
		MkRcsId,
		"DISTFILES+=\tvalue",
		"DISTFILES+= \\",
		"\t\t\tvalue",
		"DISTFILES+=\t\t\tvalue",
		"DISTFILES+= value",
		"",
		"DISTFILES= \\",
		"value")
	mklines := NewMkLines(lines)

	mklines.Check()

	t.CheckOutputLines(
		"AUTOFIX: ~/Makefile:5: Replacing \"\\t\\t\\t\" with \"\\t\".",
		"AUTOFIX: ~/Makefile:6: Replacing \" \" with \"\\t\".")
	t.CheckFileLines("Makefile",
		MkRcsId,
		"DISTFILES+=\tvalue",
		"DISTFILES+= \\",
		"\t\t\tvalue",
		"DISTFILES+=\tvalue",
		"DISTFILES+=\tvalue",
		"",
		"DISTFILES= \\",
		"value")
}

// When there is an outlier, no matter whether indented using space or tab,
// fix the whole block to use the indentation of the second-longest line.
// Since all of the remaining lines have the same indentation (there is
// only 1 line at all), that existing indentation is used instead of the
// minimum necessary, which would only be a single tab.
func (s *Suite) Test_MkLines__variable_alignment__autofix_tab_outlier(c *check.C) {
	t := s.Init(c)

	t.SetupCommandLine("-Wspace", "--autofix")
	G.globalData.InitVartypes()
	lines := t.SetupFileLinesContinuation("Makefile",
		MkRcsId,
		"DISTFILES=\t\tvery-very-very-very-long-distfile-name",
		"SITES.very-very-very-very-long-distfile-name=\t${MASTER_SITE_LOCAL}")
	mklines := NewMkLines(lines)

	mklines.Check()

	t.CheckOutputLines(
		"AUTOFIX: ~/Makefile:3: Replacing \"\\t\" with \" \".")
	t.CheckFileLines("Makefile",
		MkRcsId,
		"DISTFILES=\t\tvery-very-very-very-long-distfile-name",
		"SITES.very-very-very-very-long-distfile-name= ${MASTER_SITE_LOCAL}")
}

func (s *Suite) Test_MkLines__for_loop_multiple_variables(c *check.C) {
	t := s.Init(c)

	t.SetupCommandLine("-Wall")
	t.SetupTool(&Tool{Name: "echo", Varname: "ECHO", Predefined: true})
	t.SetupTool(&Tool{Name: "find", Varname: "FIND", Predefined: true})
	t.SetupTool(&Tool{Name: "pax", Varname: "PAX", Predefined: true})
	mklines := t.NewMkLines("Makefile", // From audio/squeezeboxserver
		MkRcsId,
		"",
		".for _list_ _dir_ in ${SBS_COPY}",
		"\tcd ${WRKSRC} && ${FIND} ${${_list_}} -type f ! -name '*.orig' 2>/dev/null "+
			"| pax -rw -pm ${DESTDIR}${PREFIX}/${${_dir_}}",
		".endfor")

	mklines.Check()

	t.CheckOutputLines(
		"WARN: Makefile:3: Variable names starting with an underscore (_list_) are reserved for internal pkgsrc use.",
		"WARN: Makefile:3: Variable names starting with an underscore (_dir_) are reserved for internal pkgsrc use.",
		"WARN: Makefile:4: The exitcode of \"${FIND}\" at the left of the | operator is ignored.")
}

func (s *Suite) Test_MkLines__alignment_autofix_multiline(c *check.C) {
	t := s.Init(c)

	t.SetupCommandLine("--autofix", "-Wall")

	// The SITES.* definition is indented less than the other lines,
	// therefore the whole block will be realigned.
	//
	// In the multiline definition for DISTFILES, the second line is
	// indented the same as all the other lines but pkglint (at least up
	// to 5.5.1) doesn't notice this because in multiline definitions it
	// discards the physical lines early and only works with the logical
	// line, in which the line continuation has been replaced with a
	// single space.
	//
	// Because of this limited knowledge, pkglint only realigns the
	// first physical line of the continued line.
	lines := t.SetupFileLinesContinuation("Makefile",
		MkRcsId,
		"",
		"DIST_SUBDIR=            asc",
		"DISTFILES=              ${DISTNAME}${EXTRACT_SUFX} frontiers.mp3 \\",
		"                        machine_wars.mp3 time_to_strike.mp3",
		".for file in frontiers.mp3 machine_wars.mp3 time_to_strike.mp3",
		"SITES.${file}=  http://asc-hq.org/",
		".endfor",
		"WRKSRC=                 ${WRKDIR}/${PKGNAME_NOREV}")
	mklines := NewMkLines(lines)

	mklines.Check()

	t.CheckOutputLines(
		"AUTOFIX: ~/Makefile:3: Replacing \"            \" with \"\\t\".",
		"AUTOFIX: ~/Makefile:4--5: Replacing \"              \" with \"\\t\".",
		"AUTOFIX: ~/Makefile:7: Replacing \"  \" with \"\\t\".",
		"AUTOFIX: ~/Makefile:9: Replacing \"                 \" with \"\\t\\t\".")
	t.CheckFileLinesDetab("Makefile",
		"# $NetBSD$",
		"",
		"DIST_SUBDIR=    asc",
		"DISTFILES=      ${DISTNAME}${EXTRACT_SUFX} frontiers.mp3 \\",
		"                        machine_wars.mp3 time_to_strike.mp3",
		".for file in frontiers.mp3 machine_wars.mp3 time_to_strike.mp3",
		"SITES.${file}=  http://asc-hq.org/",
		".endfor",
		"WRKSRC=         ${WRKDIR}/${PKGNAME_NOREV}")
}

func (s *Suite) Test_MkLines__alignment_space(c *check.C) {
	t := s.Init(c)

	t.SetupCommandLine("-Wall", "--autofix")

	lines := t.SetupFileLines("Makefile",
		MkRcsId,
		"RESTRICTED=\tDo not sell, do not rent",
		"NO_BIN_ON_CDROM= ${RESTRICTED}",
		"NO_BIN_ON_FTP=\t${RESTRICTED}",
		"NO_SRC_ON_CDROM= ${RESTRICTED}",
		"NO_SRC_ON_FTP=\t${RESTRICTED}")
	mklines := NewMkLines(lines)

	mklines.Check()

	t.CheckOutputLines(
		"AUTOFIX: ~/Makefile:2: Replacing \"\\t\" with \"\\t\\t\".",
		"AUTOFIX: ~/Makefile:3: Replacing \" \" with \"\\t\".",
		"AUTOFIX: ~/Makefile:4: Replacing \"\\t\" with \"\\t\\t\".",
		"AUTOFIX: ~/Makefile:5: Replacing \" \" with \"\\t\".",
		"AUTOFIX: ~/Makefile:6: Replacing \"\\t\" with \"\\t\\t\".")
}

func (s *Suite) Test_MkLines__alignment__only_space(c *check.C) {
	t := s.Init(c)

	t.SetupCommandLine("-Wall", "--autofix")

	lines := t.SetupFileLines("Makefile",
		MkRcsId,
		"DISTFILES+= space",
		"DISTFILES+= space",
		"",
		"REPLACE_PYTHON+= *.py",
		"REPLACE_PYTHON+= lib/*.py",
		"REPLACE_PYTHON+= src/*.py")
	mklines := NewMkLines(lines)

	mklines.Check()

	t.CheckOutputLines(
		"AUTOFIX: ~/Makefile:2: Replacing \" \" with \"\\t\".",
		"AUTOFIX: ~/Makefile:3: Replacing \" \" with \"\\t\".",
		"AUTOFIX: ~/Makefile:5: Replacing \" \" with \"\\t\".",
		"AUTOFIX: ~/Makefile:6: Replacing \" \" with \"\\t\".",
		"AUTOFIX: ~/Makefile:7: Replacing \" \" with \"\\t\".")
}

// The indentation is deeper than necessary, but all lines agree on
// the same column. Therefore this column should be kept.
func (s *Suite) Test_MkLines__alignment__mixed_tabs_and_spaces_same_column(c *check.C) {
	t := s.Init(c)

	t.SetupCommandLine("-Wall", "--autofix")

	lines := t.SetupFileLines("Makefile",
		MkRcsId,
		"DISTFILES+=             space",
		"DISTFILES+=\t\tspace")
	mklines := NewMkLines(lines)

	mklines.Check()

	t.CheckOutputLines(
		"AUTOFIX: ~/Makefile:2: Replacing \"             \" with \"\\t\\t\".")
}

func (s *Suite) Test_MkLines__comparing_YesNo_variable_to_string(c *check.C) {
	t := s.Init(c)

	t.SetupCommandLine("-Wall")
	G.globalData.InitVartypes()
	mklines := t.NewMkLines("databases/gdbm_compat/builtin.mk",
		MkRcsId,
		".if ${USE_BUILTIN.gdbm} == \"no\"",
		".endif",
		".if ${USE_BUILTIN.gdbm:tu} == \"no\"", // Can never be true, since "no" is not uppercase.
		".endif")

	mklines.Check()

	t.CheckOutputLines(
		"WARN: databases/gdbm_compat/builtin.mk:2: " +
			"USE_BUILTIN.gdbm should be matched against \"[yY][eE][sS]\" or \"[nN][oO]\", not compared with \"no\".")
}

func (s *Suite) Test_MkLines__varuse_sh_modifier(c *check.C) {
	t := s.Init(c)

	t.SetupCommandLine("-Wall")
	G.globalData.InitVartypes()
	mklines := t.NewMkLines("lang/qore/module.mk",
		MkRcsId,
		"qore-version=\tqore --short-version | ${SED} -e s/-.*//",
		"PLIST_SUBST+=\tQORE_VERSION=\"${qore-version:sh}\"")

	vars2 := mklines.mklines[1].DetermineUsedVariables()

	c.Check(vars2, deepEquals, []string{"SED"})

	vars3 := mklines.mklines[2].DetermineUsedVariables()

	c.Check(vars3, deepEquals, []string{"qore-version"})

	mklines.Check()

	// No warnings about defined but not used or vice versa
	t.CheckOutputEmpty()
}

func (s *Suite) Test_MkLines__varuse_parameterized(c *check.C) {
	t := s.Init(c)

	t.SetupCommandLine("-Wall")
	G.globalData.InitVartypes()
	mklines := t.NewMkLines("converters/wv2/Makefile",
		MkRcsId,
		"CONFIGURE_ARGS+=\t\t${CONFIGURE_ARGS.${ICONV_TYPE}-iconv}",
		"CONFIGURE_ARGS.gnu-iconv=\t--with-libiconv=${BUILDLINK_PREFIX.iconv}")

	mklines.Check()

	// No warnings about defined but not used or vice versa
	t.CheckOutputEmpty()
}

func (s *Suite) Test_MkLines__loop_modifier(c *check.C) {
	t := s.Init(c)

	t.SetupCommandLine("-Wall")
	G.globalData.InitVartypes()
	mklines := t.NewMkLines("chat/xchat/Makefile",
		MkRcsId,
		"GCONF_SCHEMAS=\tapps_xchat_url_handler.schemas",
		"post-install:",
		"\t${GCONF_SCHEMAS:@.s.@"+
			"${INSTALL_DATA} ${WRKSRC}/src/common/dbus/${.s.} ${DESTDIR}${GCONF_SCHEMAS_DIR}/@}")

	mklines.Check()

	t.CheckOutputLines(
		// No warning about missing @ at the end
		"WARN: chat/xchat/Makefile:4: " +
			"Unknown shell command \"${GCONF_SCHEMAS:@.s.@" +
			"${INSTALL_DATA} ${WRKSRC}/src/common/dbus/${.s.} ${DESTDIR}${GCONF_SCHEMAS_DIR}/@}\".")
}

// PR 46570
func (s *Suite) Test_MkLines__PKG_SKIP_REASON_depending_on_OPSYS(c *check.C) {
	t := s.Init(c)

	G.globalData.InitVartypes()
	mklines := t.NewMkLines("Makefile",
		MkRcsId,
		"PKG_SKIP_REASON+=\t\"Fails everywhere\"",
		".if ${OPSYS} == \"Cygwin\"",
		"PKG_SKIP_REASON+=\t\"Fails on Cygwin\"",
		".endif")

	mklines.Check()

	t.CheckOutputLines(
		"NOTE: Makefile:4: Consider defining NOT_FOR_PLATFORM instead of setting PKG_SKIP_REASON depending on ${OPSYS}.")
}

// PR 46570, item "15. net/uucp/Makefile has a make loop"
func (s *Suite) Test_MkLines__indirect_variables(c *check.C) {
	t := s.Init(c)

	t.SetupCommandLine("-Wall")
	mklines := t.NewMkLines("net/uucp/Makefile",
		MkRcsId,
		"",
		"post-configure:",
		".for var in MAIL_PROGRAM CMDPATH",
		"\t"+`${RUN} ${ECHO} "#define ${var} \""${UUCP_${var}}"\"`,
		".endfor")

	mklines.Check()

	// No warning about UUCP_${var} being used but not defined.
	t.CheckOutputLines(
		"WARN: net/uucp/Makefile:5: Unknown shell command \"${ECHO}\".")
}

func (s *Suite) Test_MkLines_Check__list_variable_as_part_of_word(c *check.C) {
	t := s.Init(c)

	t.SetupCommandLine("-Wall")
	mklines := t.NewMkLines("converters/chef/Makefile",
		MkRcsId,
		"\tcd ${WRKSRC} && tr '\\r' '\\n' < ${DISTDIR}/${DIST_SUBDIR}/${DISTFILES} > chef.l")

	mklines.Check()

	t.CheckOutputLines(
		"WARN: converters/chef/Makefile:2: Unknown shell command \"tr\".",
		"WARN: converters/chef/Makefile:2: The list variable DISTFILES should not be embedded in a word.")
}

func (s *Suite) Test_MkLines_Check__absolute_pathname_depending_on_OPSYS(c *check.C) {
	t := s.Init(c)

	t.SetupCommandLine("-Wall")
	G.globalData.InitVartypes()
	mklines := t.NewMkLines("games/heretic2-demo/Makefile",
		MkRcsId,
		".if ${OPSYS} == \"DragonFly\"",
		"TOOLS_PLATFORM.gtar=\t/usr/bin/bsdtar",
		".endif",
		"TOOLS_PLATFORM.gtar=\t/usr/bin/bsdtar")

	mklines.Check()

	// No warning about an unknown shell command in line 3,
	// since that line depends on OPSYS.
	t.CheckOutputLines(
		"WARN: games/heretic2-demo/Makefile:3: The variable TOOLS_PLATFORM.gtar may not be set by any package.",
		"WARN: games/heretic2-demo/Makefile:5: The variable TOOLS_PLATFORM.gtar may not be set by any package.",
		"WARN: games/heretic2-demo/Makefile:5: Unknown shell command \"/usr/bin/bsdtar\".")
}

func (s *Suite) Test_MkLines_checkForUsedComment(c *check.C) {
	t := s.Init(c)

	t.SetupCommandLine("--show-autofix")
	t.NewMkLines("Makefile.common",
		MkRcsId,
		"",
		"# used by sysutils/mc",
	).checkForUsedComment("sysutils/mc")

	t.CheckOutputEmpty()

	t.NewMkLines("Makefile.common").checkForUsedComment("category/package")

	t.CheckOutputEmpty()

	t.NewMkLines("Makefile.common",
		MkRcsId,
	).checkForUsedComment("category/package")

	t.CheckOutputEmpty()

	t.NewMkLines("Makefile.common",
		MkRcsId,
		"",
	).checkForUsedComment("category/package")

	t.CheckOutputEmpty()

	t.NewMkLines("Makefile.common",
		MkRcsId,
		"",
		"VARNAME=\tvalue",
	).checkForUsedComment("category/package")

	t.CheckOutputLines(
		"WARN: Makefile.common:2: Please add a line \"# used by category/package\" here.",
		"AUTOFIX: Makefile.common:2: Inserting a line \"# used by category/package\" before this line.")

	t.NewMkLines("Makefile.common",
		MkRcsId,
		"#",
		"#",
	).checkForUsedComment("category/package")

	t.CheckOutputLines(
		"WARN: Makefile.common:3: Please add a line \"# used by category/package\" here.",
		"AUTOFIX: Makefile.common:3: Inserting a line \"# used by category/package\" before this line.")

	c.Check(G.autofixAvailable, equals, true)
}

func (s *Suite) Test_MkLines_DetermineUsedVariables__simple(c *check.C) {
	t := s.Init(c)

	mklines := t.NewMkLines("fname",
		"\t${VAR}")
	mkline := mklines.mklines[0]
	G.Mk = mklines

	mklines.DetermineUsedVariables()

	c.Check(len(mklines.varuse), equals, 1)
	c.Check(mklines.varuse["VAR"], equals, mkline)
}

func (s *Suite) Test_MkLines_DetermineUsedVariables__nested(c *check.C) {
	t := s.Init(c)

	mklines := t.NewMkLines("fname",
		"\t${outer.${inner}}")
	mkline := mklines.mklines[0]
	G.Mk = mklines

	mklines.DetermineUsedVariables()

	c.Check(len(mklines.varuse), equals, 3)
	c.Check(mklines.varuse["inner"], equals, mkline)
	c.Check(mklines.varuse["outer."], equals, mkline)
	c.Check(mklines.varuse["outer.*"], equals, mkline)
}

func (s *Suite) Test_MkLines_PrivateTool_Undefined(c *check.C) {
	t := s.Init(c)

	t.SetupCommandLine("-Wall")
	G.globalData.InitVartypes()
	mklines := t.NewMkLines("fname",
		MkRcsId,
		"",
		"\tmd5sum filename")

	mklines.Check()

	t.CheckOutputLines(
		"WARN: fname:3: Unknown shell command \"md5sum\".")
}

func (s *Suite) Test_MkLines_PrivateTool_Defined(c *check.C) {
	t := s.Init(c)

	t.SetupCommandLine("-Wall")
	G.globalData.InitVartypes()
	mklines := t.NewMkLines("fname",
		MkRcsId,
		"TOOLS_CREATE+=\tmd5sum",
		"",
		"\tmd5sum filename")

	mklines.Check()

	t.CheckOutputEmpty()
}

func (s *Suite) Test_MkLines_Check_indentation(c *check.C) {
	t := s.Init(c)

	t.SetupCommandLine("-Wall")
	mklines := t.NewMkLines("options.mk",
		MkRcsId,
		". if !defined(GUARD_MK)",
		". if ${OPSYS} == ${OPSYS}",
		".   for i in ${FILES}",
		".     if !defined(GUARD2_MK)",
		".     else",
		".     endif",
		".   endfor",
		".   if ${COND1}",
		".   elif ${COND2}",
		".   else ${COND3}",
		".   endif",
		". endif",
		". endif",
		". endif")

	mklines.Check()

	t.CheckOutputLines(""+
		"NOTE: options.mk:2: This directive should be indented by 0 spaces.",
		"NOTE: options.mk:3: This directive should be indented by 0 spaces.",
		"NOTE: options.mk:4: This directive should be indented by 2 spaces.",
		"NOTE: options.mk:5: This directive should be indented by 4 spaces.",
		"NOTE: options.mk:6: This directive should be indented by 4 spaces.",
		"NOTE: options.mk:7: This directive should be indented by 4 spaces.",
		"NOTE: options.mk:8: This directive should be indented by 2 spaces.",
		"NOTE: options.mk:9: This directive should be indented by 2 spaces.",
		"NOTE: options.mk:10: This directive should be indented by 2 spaces.",
		"NOTE: options.mk:11: This directive should be indented by 2 spaces.",
		"ERROR: options.mk:11: \".else\" does not take arguments.",
		"NOTE: options.mk:11: If you meant \"else if\", use \".elif\".",
		"NOTE: options.mk:12: This directive should be indented by 2 spaces.",
		"NOTE: options.mk:13: This directive should be indented by 0 spaces.",
		"NOTE: options.mk:14: This directive should be indented by 0 spaces.",
		"ERROR: options.mk:15: Unmatched .endif.",
		"NOTE: options.mk:15: This directive should be indented by 0 spaces.")
}

// Demonstrates how to define your own make(1) targets.
func (s *Suite) Test_MkLines_wip_category_Makefile(c *check.C) {
	t := s.Init(c)

	t.SetupCommandLine("-Wall")
	G.globalData.InitVartypes()
	t.SetupTool(&Tool{Name: "rm", Varname: "RM", Predefined: true})
	mklines := t.NewMkLines("Makefile",
		MkRcsId,
		"",
		"COMMENT=\tWIP pkgsrc packages",
		"",
		"SUBDIR+=\taaa",
		"SUBDIR+=\tzzz",
		"",
		"${.CURDIR}/PKGDB:",
		"\t${RM} -f ${.CURDIR}/PKGDB",
		"",
		"${.CURDIR}/INDEX:",
		"\t${RM} -f ${.CURDIR}/INDEX",
		"",
		".include \"../../mk/misc/category.mk\"")

	mklines.Check()

	t.CheckOutputLines(
		"ERROR: Makefile:14: \"/mk/misc/category.mk\" does not exist.")
}
