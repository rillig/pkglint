package main

import "gopkg.in/check.v1"

// VaralignTester reduces the amount of test code for variable
// aligning in Makefiles.
type VaralignTester struct {
	suite               *Suite
	tester              *Tester
	actualInput         []string
	expectedAutofixes   []string
	expectedDiagnostics []string
	expectedResult      []string
}

func NewVaralignTester(s *Suite, c *check.C) *VaralignTester {
	t := s.Init(c)

	return &VaralignTester{suite: s, tester: t}
}

func (vt *VaralignTester) Input(lines ...string) {
	vt.actualInput = lines
}

func (vt *VaralignTester) Diagnostics(diagnostics ...string) {
	vt.expectedDiagnostics = diagnostics
}

func (vt *VaralignTester) Autofixes(diagnostics ...string) {
	vt.expectedAutofixes = diagnostics
}

func (vt *VaralignTester) Result(lines ...string) {
	vt.expectedResult = lines
}

func (vt *VaralignTester) Run() {
	vt.runDefault()
	vt.runAutofix()
}

func (vt *VaralignTester) runDefault() {
	vt.tester.SetupCommandLine("-Wall")

	lines := vt.tester.SetupFileLinesContinuation("Makefile", vt.actualInput...)
	mklines := NewMkLines(lines)

	varalign := VaralignBlock{}
	for _, mkline := range mklines.mklines {
		varalign.Check(mkline)
	}
	varalign.Finish()

	vt.tester.CheckOutputLines(vt.expectedDiagnostics...)
}

func (vt *VaralignTester) runAutofix() {
	vt.tester.SetupCommandLine("-Wall", "--autofix")

	lines := vt.tester.SetupFileLinesContinuation("Makefile", vt.actualInput...)

	mklines := NewMkLines(lines)

	var varalign VaralignBlock
	for _, mkline := range mklines.mklines {
		varalign.Check(mkline)
	}
	varalign.Finish()

	vt.tester.CheckOutputLines(vt.expectedAutofixes...)

	SaveAutofixChanges(mklines.lines)
	vt.tester.CheckFileLinesDetab("Makefile", vt.expectedResult...)
}

func (s *Suite) Test_Varalign__one_var_tab(c *check.C) {
	vt := NewVaralignTester(s, c)
	vt.Input(
		"VAR=\tone tab")
	vt.Diagnostics()
	vt.Autofixes()
	vt.Result(
		"VAR=    one tab")
	vt.Run()
}

func (s *Suite) Test_Varalign__one_var_tabs(c *check.C) {
	vt := NewVaralignTester(s, c)
	vt.Input(
		"VAR=\t\t\tseveral tabs")
	vt.Diagnostics()
	vt.Autofixes()
	vt.Result(
		"VAR=                    several tabs")
	vt.Run()
}

func (s *Suite) Test_Varalign__one_var_space(c *check.C) {
	vt := NewVaralignTester(s, c)
	vt.Input(
		"VAR= indented with one space")
	vt.Diagnostics(
		"NOTE: ~/Makefile:1: This variable value should be aligned with tabs, not spaces, to column 9.")
	vt.Autofixes(
		"AUTOFIX: ~/Makefile:1: Replacing \" \" with \"\\t\".")
	vt.Result(
		"VAR=    indented with one space")
	vt.Run()
}

func (s *Suite) Test_Varalign__one_var_spaces(c *check.C) {
	vt := NewVaralignTester(s, c)
	vt.Input(
		"VAR=   several spaces")
	vt.Diagnostics(
		"NOTE: ~/Makefile:1: This variable value should be aligned with tabs, not spaces, to column 9.")
	vt.Autofixes(
		"AUTOFIX: ~/Makefile:1: Replacing \"   \" with \"\\t\".")
	vt.Result(
		"VAR=    several spaces")
	vt.Run()
}

// Inconsistently aligned lines for variables of the same length are
// autofixed to the next tab.
func (s *Suite) Test_Varalign__two_vars__spaces(c *check.C) {
	vt := NewVaralignTester(s, c)
	vt.Input(
		"VAR= indented with one space",
		"VAR=  indented with two spaces")
	vt.Diagnostics(
		"NOTE: ~/Makefile:1: This variable value should be aligned with tabs, not spaces, to column 9.",
		"NOTE: ~/Makefile:2: This variable value should be aligned with tabs, not spaces, to column 9.")
	vt.Autofixes(
		"AUTOFIX: ~/Makefile:1: Replacing \" \" with \"\\t\".",
		"AUTOFIX: ~/Makefile:2: Replacing \"  \" with \"\\t\".")
	vt.Result(
		"VAR=    indented with one space",
		"VAR=    indented with two spaces")
	vt.Run()
}

// The values in a block should be aligned.
func (s *Suite) Test_Varalign__several_vars__spaces(c *check.C) {
	vt := NewVaralignTester(s, c)
	vt.Input(
		"GRP_A= value",
		"GRP_AA= value",
		"GRP_AAA= value",
		"GRP_AAAA= value")
	vt.Diagnostics(
		"NOTE: ~/Makefile:1: This variable value should be aligned with tabs, not spaces, to column 17.",
		"NOTE: ~/Makefile:2: This variable value should be aligned with tabs, not spaces, to column 17.",
		"NOTE: ~/Makefile:3: This variable value should be aligned with tabs, not spaces, to column 17.",
		"NOTE: ~/Makefile:4: This variable value should be aligned with tabs, not spaces, to column 17.")
	vt.Autofixes(
		"AUTOFIX: ~/Makefile:1: Replacing \" \" with \"\\t\\t\".",
		"AUTOFIX: ~/Makefile:2: Replacing \" \" with \"\\t\\t\".",
		"AUTOFIX: ~/Makefile:3: Replacing \" \" with \"\\t\".",
		"AUTOFIX: ~/Makefile:4: Replacing \" \" with \"\\t\".")
	vt.Result(
		"GRP_A=          value",
		"GRP_AA=         value",
		"GRP_AAA=        value",
		"GRP_AAAA=       value")
	vt.Run()
}

// Continuation lines may be indented with a single space.
func (s *Suite) Test_Varalign__continuation(c *check.C) {
	vt := NewVaralignTester(s, c)
	vt.Input(
		"VAR= \\",
		"\tvalue")
	vt.Diagnostics()
	vt.Autofixes()
	vt.Result(
		"VAR= \\",
		"        value")
	vt.Run()
}

// To align these two lines, the first line needs more more tab.
// The second line is further to the right but doesn't count as
// an outlier since it is not far enough.
// Adding one more tab to the indentation is generally considered ok.
func (s *Suite) Test_Varalign__short_tab__long_space(c *check.C) {
	vt := NewVaralignTester(s, c)
	vt.Input(
		"BLOCK=\tindented with tab",
		"BLOCK_LONGVAR= indented with space")
	vt.Diagnostics(
		"NOTE: ~/Makefile:1: This variable value should be aligned to column 17.",
		"NOTE: ~/Makefile:2: This variable value should be aligned with tabs, not spaces, to column 17.")
	vt.Autofixes(
		"AUTOFIX: ~/Makefile:1: Replacing \"\\t\" with \"\\t\\t\".",
		"AUTOFIX: ~/Makefile:2: Replacing \" \" with \"\\t\".")
	vt.Result(
		"BLOCK=          indented with tab",
		"BLOCK_LONGVAR=  indented with space")
	vt.Run()
}

func (s *Suite) Test_Varalign__short_long__tab(c *check.C) {
	vt := NewVaralignTester(s, c)
	vt.Input(
		"BLOCK=\tshort",
		"BLOCK_LONGVAR=\tlong")
	vt.Diagnostics(
		"NOTE: ~/Makefile:1: This variable value should be aligned to column 17.")
	vt.Autofixes(
		"AUTOFIX: ~/Makefile:1: Replacing \"\\t\" with \"\\t\\t\".")
	vt.Result(
		"BLOCK=          short",
		"BLOCK_LONGVAR=  long")
	vt.Run()
}

func (s *Suite) Test_Varalign__space_and_tab(c *check.C) {
	vt := NewVaralignTester(s, c)
	vt.Input(
		"VAR=    space",
		"VAR=\ttab ${VAR}")
	vt.Diagnostics(
		"NOTE: ~/Makefile:1: Variable values should be aligned with tabs, not spaces.")
	vt.Autofixes(
		"AUTOFIX: ~/Makefile:1: Replacing \"    \" with \"\\t\".")
	vt.Result(
		"VAR=    space",
		"VAR=    tab ${VAR}")
	vt.Run()
}

func (s *Suite) Test_Varalign__no_space_at_all(c *check.C) {
	vt := NewVaralignTester(s, c)
	vt.Input(
		"PKG_FAIL_REASON+=\"Message\"")
	vt.Diagnostics(
		"NOTE: ~/Makefile:1: This variable value should be aligned to column 25.")
	vt.Autofixes(
		"AUTOFIX: ~/Makefile:1: Replacing \"\" with \"\\t\".")
	vt.Result(
		"PKG_FAIL_REASON+=       \"Message\"")
	vt.Run()
}

// Continuation lines without any content on the first line may use
// a space for variable value alignment.
// They are ignored when calculating the preferred alignment depth.
func (s *Suite) Test_Varalign__continuation_lines(c *check.C) {
	vt := NewVaralignTester(s, c)
	vt.Input(
		"DISTFILES+=\tvalue",
		"DISTFILES+= \\", // Continuation lines with small indentation must be aligned.
		"\t\t\tvalue",
		"DISTFILES+=\t\t\tvalue",
		"DISTFILES+= value",
		"",
		"DISTFILES= \\",
		"value")
	vt.Diagnostics(
		"NOTE: ~/Makefile:2--3: This variable value should be aligned with tabs, not spaces, to column 17.",
		"WARN: ~/Makefile:2--3: This line should be aligned with \"\\t\\t\".",
		"NOTE: ~/Makefile:4: This variable value should be aligned to column 17.",
		"NOTE: ~/Makefile:5: This variable value should be aligned with tabs, not spaces, to column 17.")
	vt.Autofixes(
		"AUTOFIX: ~/Makefile:2: Replacing \" \" with \"\\t\".",
		"AUTOFIX: ~/Makefile:3: Replacing indentation \"\\t\\t\\t\" with \"\\t\\t\".",
		"AUTOFIX: ~/Makefile:4: Replacing \"\\t\\t\\t\" with \"\\t\".",
		"AUTOFIX: ~/Makefile:5: Replacing \" \" with \"\\t\".")
	vt.Result(
		"DISTFILES+=     value",
		"DISTFILES+=     \\",
		"                value",
		"DISTFILES+=     value",
		"DISTFILES+=     value",
		"",
		"DISTFILES= \\",
		"value")
	vt.Run()
}

// Ensures that a wrong warning introduced in ccb56a5 is not logged.
func (s *Suite) Test_Varalign__aligned_continuation(c *check.C) {
	vt := NewVaralignTester(s, c)
	vt.Input(
		"USE_TOOLS+=\t[ awk \\",
		"\t\tsed")
	vt.Diagnostics()
	vt.Autofixes()
	vt.Result(
		"USE_TOOLS+=     [ awk \\",
		"                sed")
	vt.Run()
}

// Shell commands are assumed to be already nicely indented.
// This particular example is not, but pkglint cannot decide this.
func (s *Suite) Test_Varalign__shell_command(c *check.C) {
	vt := NewVaralignTester(s, c)
	vt.Input(
		"USE_BUILTIN.Xfixes=\tyes",
		"USE_BUILTIN.Xfixes!=\t\t\t\t\t\t\t\\",
		"\tif ${PKG_ADMIN} pmatch ...; then\t\t\t\t\t\\",
		"\t\t:; else :; fi")
	vt.Diagnostics()
	vt.Autofixes()
	vt.Result(
		"USE_BUILTIN.Xfixes=     yes",
		"USE_BUILTIN.Xfixes!=                                                    \\",
		"        if ${PKG_ADMIN} pmatch ...; then                                        \\",
		"                :; else :; fi")
	vt.Run()
}

// The most common pattern is to have all values in the
// continuation lines, all indented to the same depth.
// The depth is either a single tab or aligns with the
// other variables in the paragraph.
func (s *Suite) Test_Varalign__continuation_value_starts_in_second_line(c *check.C) {
	vt := NewVaralignTester(s, c)
	vt.Input(
		"WRKSRC=\t${WRKDIR}",
		"DISTFILES=\tdistfile-1.0.0.tar.gz",
		"SITES.distfile-1.0.0.tar.gz= \\",
		"\t\t\t${MASTER_SITES_SOURCEFORGE} \\",
		"\t\t\t${MASTER_SITES_GITHUB}")
	vt.Diagnostics(
		"NOTE: ~/Makefile:1: This variable value should be aligned to column 17.",
		"WARN: ~/Makefile:3--5: This line should be aligned with \"\\t\\t\".")
	vt.Autofixes(
		"AUTOFIX: ~/Makefile:1: Replacing \"\\t\" with \"\\t\\t\".",
		"AUTOFIX: ~/Makefile:4: Replacing indentation \"\\t\\t\\t\" with \"\\t\\t\".",
		"AUTOFIX: ~/Makefile:5: Replacing indentation \"\\t\\t\\t\" with \"\\t\\t\".")
	vt.Result(
		"WRKSRC=         ${WRKDIR}",
		"DISTFILES=      distfile-1.0.0.tar.gz",
		"SITES.distfile-1.0.0.tar.gz= \\",
		"                ${MASTER_SITES_SOURCEFORGE} \\",
		"                ${MASTER_SITES_GITHUB}")
	vt.Run()
}

// Another common pattern is to have the first value
// in the first line and subsequent values indented to
// the same depth as the value in the first line.
func (s *Suite) Test_Varalign__continuation_value_starts_in_first_line(c *check.C) {
	vt := NewVaralignTester(s, c)
	vt.Input(
		"WRKSRC=\t${WRKDIR}",
		"DISTFILES=\tdistfile-1.0.0.tar.gz",
		"SITES.distfile-1.0.0.tar.gz=\t${MASTER_SITES_SOURCEFORGE} \\",
		"\t\t\t\t${MASTER_SITES_GITHUB}")
	vt.Diagnostics(
		"NOTE: ~/Makefile:1: This variable value should be aligned to column 17.",
		"NOTE: ~/Makefile:3--4: This variable value should be aligned to column 17.",
		"WARN: ~/Makefile:3--4: This line should be aligned with \"\\t\\t\".")
	vt.Autofixes(
		"AUTOFIX: ~/Makefile:1: Replacing \"\\t\" with \"\\t\\t\".",
		"AUTOFIX: ~/Makefile:3: Replacing \"\\t\" with \" \".",
		"AUTOFIX: ~/Makefile:4: Replacing indentation \"\\t\\t\\t\\t\" with \"\\t\\t\".")
	vt.Result(
		"WRKSRC=         ${WRKDIR}",
		"DISTFILES=      distfile-1.0.0.tar.gz",
		"SITES.distfile-1.0.0.tar.gz= ${MASTER_SITES_SOURCEFORGE} \\",
		"                ${MASTER_SITES_GITHUB}")
	vt.Run()
}

// Continued lines that have mixed indentation are
// probably on purpose. Their minimum indentation should
// be aligned to the indentation of the other lines. The
// lines that are indented further should keep their
// relative indentation depth, no matter if that is done
// with spaces or with tabs.
func (s *Suite) Test_Varalign__continuation_mixed_indentation_in_second_line(c *check.C) {
	vt := NewVaralignTester(s, c)
	vt.Input(
		"WRKSRC=\t${WRKDIR}",
		"DISTFILES=\tdistfile-1.0.0.tar.gz",
		"AWK_PROGRAM+= \\",
		"\t\t\t\t  /search/ { \\",
		"\t\t\t\t    action(); \\",
		"\t\t\t\t  }")
	vt.Diagnostics(
		"NOTE: ~/Makefile:1: This variable value should be aligned to column 17.",
		"NOTE: ~/Makefile:3--6: This variable value should be aligned with tabs, not spaces, to column 17.",
		"WARN: ~/Makefile:3--6: This line should be aligned with \"\\t\\t\".")
	vt.Autofixes(
		"AUTOFIX: ~/Makefile:1: Replacing \"\\t\" with \"\\t\\t\".",
		"AUTOFIX: ~/Makefile:3: Replacing \" \" with \"\\t\".",
		"AUTOFIX: ~/Makefile:4: Replacing indentation \"\\t\\t\\t\\t  \" with \"\\t\\t\".",
		"AUTOFIX: ~/Makefile:5: Replacing indentation \"\\t\\t\\t\\t    \" with \"\\t\\t  \".",
		"AUTOFIX: ~/Makefile:6: Replacing indentation \"\\t\\t\\t\\t  \" with \"\\t\\t\".")
	vt.Result(
		"WRKSRC=         ${WRKDIR}",
		"DISTFILES=      distfile-1.0.0.tar.gz",
		"AWK_PROGRAM+=   \\",
		"                /search/ { \\",
		"                  action(); \\",
		"                }")
	vt.Run()
}

// Continuation lines may also start their values in the first line.
func (s *Suite) Test_Varalign__continuation_mixed_indentation_in_first_line(c *check.C) {
	vt := NewVaralignTester(s, c)
	vt.Input(
		"WRKSRC=\t${WRKDIR}",
		"DISTFILES=\tdistfile-1.0.0.tar.gz",
		"AWK_PROGRAM+=\t\t\t  /search/ { \\",
		"\t\t\t\t    action(); \\",
		"\t\t\t\t  }")
	vt.Diagnostics(
		"NOTE: ~/Makefile:1: This variable value should be aligned to column 17.",
		"NOTE: ~/Makefile:3--5: This variable value should be aligned with tabs, not spaces, to column 17.",
		"WARN: ~/Makefile:3--5: This line should be aligned with \"\\t\\t\".")
	vt.Autofixes(
		"AUTOFIX: ~/Makefile:1: Replacing \"\\t\" with \"\\t\\t\".",
		"AUTOFIX: ~/Makefile:3: Replacing \"\\t\\t\\t  \" with \"\\t\".",
		"AUTOFIX: ~/Makefile:4: Replacing indentation \"\\t\\t\\t\\t    \" with \"\\t\\t  \".",
		"AUTOFIX: ~/Makefile:5: Replacing indentation \"\\t\\t\\t\\t  \" with \"\\t\\t\".")
	vt.Result(
		"WRKSRC=         ${WRKDIR}",
		"DISTFILES=      distfile-1.0.0.tar.gz",
		"AWK_PROGRAM+=   /search/ { \\",
		"                  action(); \\",
		"                }")
	vt.Run()
}

// When there is an outlier, no matter whether indented using space or tab,
// fix the whole block to use the indentation of the second-longest line.
// Since all of the remaining lines have the same indentation (there is
// only 1 line at all), that existing indentation is used instead of the
// minimum necessary, which would only be a single tab.
func (s *Suite) Test_Varalign__tab_outlier(c *check.C) {
	vt := NewVaralignTester(s, c)
	vt.Input(
		"DISTFILES=\t\tvery-very-very-very-long-distfile-name",
		"SITES.very-very-very-very-long-distfile-name=\t${MASTER_SITE_LOCAL}")
	vt.Diagnostics(
		"NOTE: ~/Makefile:2: This variable value should be aligned to column 25.")
	vt.Autofixes(
		"AUTOFIX: ~/Makefile:2: Replacing \"\\t\" with \" \".")
	vt.Result(
		"DISTFILES=              very-very-very-very-long-distfile-name",
		"SITES.very-very-very-very-long-distfile-name= ${MASTER_SITE_LOCAL}")
	vt.Run()
}

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
func (s *Suite) Test_Varalign__multiline(c *check.C) {
	vt := NewVaralignTester(s, c)
	vt.Input(
		"DIST_SUBDIR=            asc",
		"DISTFILES=              ${DISTNAME}${EXTRACT_SUFX} frontiers.mp3 \\",
		"                        machine_wars.mp3 time_to_strike.mp3",
		".for file in frontiers.mp3 machine_wars.mp3 time_to_strike.mp3",
		"SITES.${file}=  http://asc-hq.org/",
		".endfor",
		"WRKSRC=                 ${WRKDIR}/${PKGNAME_NOREV}")
	vt.Diagnostics(
		"NOTE: ~/Makefile:1: This variable value should be aligned with tabs, not spaces, to column 17.",
		"NOTE: ~/Makefile:2--3: This variable value should be aligned with tabs, not spaces, to column 17.",
		"WARN: ~/Makefile:2--3: This line should be aligned with \"\\t\\t\".",
		"NOTE: ~/Makefile:5: Variable values should be aligned with tabs, not spaces.",
		"NOTE: ~/Makefile:7: This variable value should be aligned with tabs, not spaces, to column 17.")
	vt.Autofixes(
		"AUTOFIX: ~/Makefile:1: Replacing \"            \" with \"\\t\".",
		"AUTOFIX: ~/Makefile:2: Replacing \"              \" with \"\\t\".",
		"AUTOFIX: ~/Makefile:3: Replacing indentation \"                        \" with \"\\t\\t\".",
		"AUTOFIX: ~/Makefile:5: Replacing \"  \" with \"\\t\".",
		"AUTOFIX: ~/Makefile:7: Replacing \"                 \" with \"\\t\\t\".")
	vt.Result(
		"DIST_SUBDIR=    asc",
		"DISTFILES=      ${DISTNAME}${EXTRACT_SUFX} frontiers.mp3 \\",
		"                machine_wars.mp3 time_to_strike.mp3",
		".for file in frontiers.mp3 machine_wars.mp3 time_to_strike.mp3",
		"SITES.${file}=  http://asc-hq.org/",
		".endfor",
		"WRKSRC=         ${WRKDIR}/${PKGNAME_NOREV}")
	vt.Run()
}

// The CDROM variables align exactly at a tab position, therefore they must
// be indented by at least one more space. Since that one space is not
// enough to count as an outlier, everything is indented by one more tab.
func (s *Suite) Test_Varalign__single_space(c *check.C) {
	vt := NewVaralignTester(s, c)
	vt.Input(
		"RESTRICTED=\tDo not sell, do not rent",
		"NO_BIN_ON_CDROM= ${RESTRICTED}",
		"NO_BIN_ON_FTP=\t${RESTRICTED}",
		"NO_SRC_ON_CDROM= ${RESTRICTED}",
		"NO_SRC_ON_FTP=\t${RESTRICTED}")
	vt.Diagnostics(
		"NOTE: ~/Makefile:1: This variable value should be aligned to column 25.",
		"NOTE: ~/Makefile:2: This variable value should be aligned with tabs, not spaces, to column 25.",
		"NOTE: ~/Makefile:3: This variable value should be aligned to column 25.",
		"NOTE: ~/Makefile:4: This variable value should be aligned with tabs, not spaces, to column 25.",
		"NOTE: ~/Makefile:5: This variable value should be aligned to column 25.")
	vt.Autofixes(
		"AUTOFIX: ~/Makefile:1: Replacing \"\\t\" with \"\\t\\t\".",
		"AUTOFIX: ~/Makefile:2: Replacing \" \" with \"\\t\".",
		"AUTOFIX: ~/Makefile:3: Replacing \"\\t\" with \"\\t\\t\".",
		"AUTOFIX: ~/Makefile:4: Replacing \" \" with \"\\t\".",
		"AUTOFIX: ~/Makefile:5: Replacing \"\\t\" with \"\\t\\t\".")
	vt.Result(
		"RESTRICTED=             Do not sell, do not rent",
		"NO_BIN_ON_CDROM=        ${RESTRICTED}",
		"NO_BIN_ON_FTP=          ${RESTRICTED}",
		"NO_SRC_ON_CDROM=        ${RESTRICTED}",
		"NO_SRC_ON_FTP=          ${RESTRICTED}")
	vt.Run()
}

// These variables all look nicely aligned, but they use spaces instead
// of tabs for alignment. The spaces are replaced with tabs, making the
// indentation a little deeper.
func (s *Suite) Test_Varalign__only_space(c *check.C) {
	vt := NewVaralignTester(s, c)
	vt.Input(
		"DISTFILES+= space",
		"DISTFILES+= space",
		"",
		"REPLACE_PYTHON+= *.py",
		"REPLACE_PYTHON+= lib/*.py",
		"REPLACE_PYTHON+= src/*.py")
	vt.Diagnostics(
		"NOTE: ~/Makefile:1: This variable value should be aligned with tabs, not spaces, to column 17.",
		"NOTE: ~/Makefile:2: This variable value should be aligned with tabs, not spaces, to column 17.",
		"NOTE: ~/Makefile:4: This variable value should be aligned with tabs, not spaces, to column 25.",
		"NOTE: ~/Makefile:5: This variable value should be aligned with tabs, not spaces, to column 25.",
		"NOTE: ~/Makefile:6: This variable value should be aligned with tabs, not spaces, to column 25.")
	vt.Autofixes(
		"AUTOFIX: ~/Makefile:1: Replacing \" \" with \"\\t\".",
		"AUTOFIX: ~/Makefile:2: Replacing \" \" with \"\\t\".",
		"AUTOFIX: ~/Makefile:4: Replacing \" \" with \"\\t\".",
		"AUTOFIX: ~/Makefile:5: Replacing \" \" with \"\\t\".",
		"AUTOFIX: ~/Makefile:6: Replacing \" \" with \"\\t\".")
	vt.Result(
		"DISTFILES+=     space",
		"DISTFILES+=     space",
		"",
		"REPLACE_PYTHON+=        *.py",
		"REPLACE_PYTHON+=        lib/*.py",
		"REPLACE_PYTHON+=        src/*.py")
	vt.Run()
}

// The indentation is deeper than necessary, but all lines agree on
// the same column. Therefore this indentation depth is kept.
func (s *Suite) Test_Varalign__mixed_tabs_and_spaces_same_column(c *check.C) {
	vt := NewVaralignTester(s, c)
	vt.Input(
		"DISTFILES+=             space",
		"DISTFILES+=\t\ttab")
	vt.Diagnostics(
		"NOTE: ~/Makefile:1: Variable values should be aligned with tabs, not spaces.")
	vt.Autofixes(
		"AUTOFIX: ~/Makefile:1: Replacing \"             \" with \"\\t\\t\".")
	vt.Result(
		"DISTFILES+=             space",
		"DISTFILES+=             tab")
	vt.Run()
}

func (s *Suite) Test_Varalign__outlier_1(c *check.C) {
	vt := NewVaralignTester(s, c)
	vt.Input(
		"V= 3",  // Adjust from 3 to 8 (+ 1 tab)
		"V=\t4") // Keep at 8
	vt.Diagnostics(
		"NOTE: ~/Makefile:1: This variable value should be aligned with tabs, not spaces, to column 9.")
	vt.Autofixes(
		"AUTOFIX: ~/Makefile:1: Replacing \" \" with \"\\t\".")
	vt.Result("V=      3",
		"V=      4")
	vt.Run()
}

func (s *Suite) Test_Varalign__outlier_2(c *check.C) {
	vt := NewVaralignTester(s, c)
	vt.Input(
		"V.0008= 6", // Keep at 8 (space to tab)
		"V=\t7")     // Keep at 8
	vt.Diagnostics(
		"NOTE: ~/Makefile:1: Variable values should be aligned with tabs, not spaces.")
	vt.Autofixes(
		"AUTOFIX: ~/Makefile:1: Replacing \" \" with \"\\t\".")
	vt.Result(
		"V.0008= 6",
		"V=      7")
	vt.Run()
}

func (s *Suite) Test_Varalign__outlier_3(c *check.C) {
	vt := NewVaralignTester(s, c)
	vt.Input(
		"V.00009= 9", // Adjust from 9 to 16 (+ 1 tab)
		"V=\t10")     // Adjust from 8 to 16 (+1 tab)
	vt.Diagnostics(
		"NOTE: ~/Makefile:1: This variable value should be aligned with tabs, not spaces, to column 17.",
		"NOTE: ~/Makefile:2: This variable value should be aligned to column 17.")
	vt.Autofixes(
		"AUTOFIX: ~/Makefile:1: Replacing \" \" with \"\\t\".",
		"AUTOFIX: ~/Makefile:2: Replacing \"\\t\" with \"\\t\\t\".")
	vt.Result(
		"V.00009=        9",
		"V=              10")
	vt.Run()
}

func (s *Suite) Test_Varalign__outlier_4(c *check.C) {
	vt := NewVaralignTester(s, c)
	vt.Input(
		"V.000000000016= 12", // Keep at 16 (space to tab)
		"V=\tvalue")          // Adjust from 8 to 16 (+ 1 tab)
	vt.Diagnostics(
		"NOTE: ~/Makefile:1: Variable values should be aligned with tabs, not spaces.",
		"NOTE: ~/Makefile:2: This variable value should be aligned to column 17.")
	vt.Autofixes(
		"AUTOFIX: ~/Makefile:1: Replacing \" \" with \"\\t\".",
		"AUTOFIX: ~/Makefile:2: Replacing \"\\t\" with \"\\t\\t\".")
	vt.Result(
		"V.000000000016= 12",
		"V=              value")
	vt.Run()
}

func (s *Suite) Test_Varalign__outlier_5(c *check.C) {
	vt := NewVaralignTester(s, c)
	vt.Input(
		"V.0000000000017= 15", // Keep at 17 (outlier)
		"V=\tvalue")           // Keep at 8 (would require + 2 tabs)
	vt.Diagnostics()
	vt.Autofixes()
	vt.Result(
		"V.0000000000017= 15",
		"V=      value")
	vt.Run()
}

func (s *Suite) Test_Varalign__outlier_6(c *check.C) {
	vt := NewVaralignTester(s, c)
	vt.Input(
		"V= 18",            // Adjust from 3 to 16 (+ 2 tabs)
		"V.000010=\tvalue") // Keep at 16
	vt.Diagnostics(
		"NOTE: ~/Makefile:1: This variable value should be aligned with tabs, not spaces, to column 17.")
	vt.Autofixes("AUTOFIX: ~/Makefile:1: Replacing \" \" with \"\\t\\t\".")
	vt.Result(
		"V=              18",
		"V.000010=       value")
	vt.Run()
}

func (s *Suite) Test_Varalign__outlier_7(c *check.C) {
	vt := NewVaralignTester(s, c)
	vt.Input(
		"V.00009= 21",      // Adjust from 9 to 16 (+ 1 tab)
		"V.000010=\tvalue") // Keep at 16
	vt.Diagnostics(
		"NOTE: ~/Makefile:1: This variable value should be aligned with tabs, not spaces, to column 17.")
	vt.Autofixes("AUTOFIX: ~/Makefile:1: Replacing \" \" with \"\\t\".")
	vt.Result(
		"V.00009=        21",
		"V.000010=       value")
	vt.Run()
}

func (s *Suite) Test_Varalign__outlier_8(c *check.C) {
	vt := NewVaralignTester(s, c)
	vt.Input(
		"V.000000000016= 24", // Keep at 16 (space to tab)
		"V.000010=\tvalue")   // Keep at 16
	vt.Diagnostics(
		"NOTE: ~/Makefile:1: Variable values should be aligned with tabs, not spaces.")
	vt.Autofixes(
		"AUTOFIX: ~/Makefile:1: Replacing \" \" with \"\\t\".")
	vt.Result(
		"V.000000000016= 24",
		"V.000010=       value")
	vt.Run()
}

func (s *Suite) Test_Varalign__outlier_9(c *check.C) {
	vt := NewVaralignTester(s, c)
	vt.Input(
		"V.0000000000017= 27", // Adjust from 17 to 24 (+ 1 tab)
		"V.000010=\tvalue")    // Adjust from 16 to 24 (+ 1 tab)
	vt.Diagnostics(
		"NOTE: ~/Makefile:1: This variable value should be aligned with tabs, not spaces, to column 25.",
		"NOTE: ~/Makefile:2: This variable value should be aligned to column 25.")
	vt.Autofixes(
		"AUTOFIX: ~/Makefile:1: Replacing \" \" with \"\\t\".",
		"AUTOFIX: ~/Makefile:2: Replacing \"\\t\" with \"\\t\\t\".")
	vt.Result(
		"V.0000000000017=        27",
		"V.000010=               value")
	vt.Run()
}

func (s *Suite) Test_Varalign__outlier_10(c *check.C) {
	vt := NewVaralignTester(s, c)
	vt.Input(
		"V.0000000000000000023= 30", // Adjust from 23 to 24 (+ 1 tab)
		"V.000010=\tvalue")          // Adjust from 16 to 24 (+ 1 tab)
	vt.Diagnostics(
		"NOTE: ~/Makefile:1: This variable value should be aligned with tabs, not spaces, to column 25.",
		"NOTE: ~/Makefile:2: This variable value should be aligned to column 25.")
	vt.Autofixes(
		"AUTOFIX: ~/Makefile:1: Replacing \" \" with \"\\t\".",
		"AUTOFIX: ~/Makefile:2: Replacing \"\\t\" with \"\\t\\t\".")
	vt.Result(
		"V.0000000000000000023=  30",
		"V.000010=               value")
	vt.Run()
}

func (s *Suite) Test_Varalign__outlier_11(c *check.C) {
	vt := NewVaralignTester(s, c)
	vt.Input(
		"V.00000000000000000024= 33", // Keep at 24 (space to tab)
		"V.000010=\tvalue")           // Adjust from 16 to 24 (+ 1 tab)
	vt.Diagnostics(
		"NOTE: ~/Makefile:1: Variable values should be aligned with tabs, not spaces.",
		"NOTE: ~/Makefile:2: This variable value should be aligned to column 25.")
	vt.Autofixes(
		"AUTOFIX: ~/Makefile:1: Replacing \" \" with \"\\t\".",
		"AUTOFIX: ~/Makefile:2: Replacing \"\\t\" with \"\\t\\t\".")
	vt.Result(
		"V.00000000000000000024= 33",
		"V.000010=               value")
	vt.Run()
}

func (s *Suite) Test_Varalign__outlier_12(c *check.C) {
	vt := NewVaralignTester(s, c)
	vt.Input(
		"V.000000000000000000025= 36", // Keep at 25 (outlier)
		"V.000010=\tvalue")            // Keep at 16 (would require + 2 tabs)
	vt.Diagnostics()
	vt.Autofixes()
	vt.Result(
		"V.000000000000000000025= 36",
		"V.000010=       value")
	vt.Run()
}

func (s *Suite) Test_Varalign__outlier_13(c *check.C) {
	vt := NewVaralignTester(s, c)
	vt.Input(
		"V.00008=\t39",          // Keep at 16
		"V.00008=\t\t\t\tvalue") // Adjust from 40 to 16 (removes 3 tabs)
	vt.Diagnostics(
		"NOTE: ~/Makefile:2: This variable value should be aligned to column 17.")
	vt.Autofixes(
		"AUTOFIX: ~/Makefile:2: Replacing \"\\t\\t\\t\\t\" with \"\\t\".")
	vt.Result(
		"V.00008=        39",
		"V.00008=        value")
	vt.Run()
}

func (s *Suite) Test_Varalign__outlier_14(c *check.C) {
	vt := NewVaralignTester(s, c)
	vt.Input(
		"V.00008=\t\t42",        // Adjust from 24 to 16 (removes 1 tab)
		"V.00008=\t\t\t\tvalue") // Adjust from 40 to 16 (removes 3 tabs)
	vt.Diagnostics(
		"NOTE: ~/Makefile:1: This variable value should be aligned to column 17.",
		"NOTE: ~/Makefile:2: This variable value should be aligned to column 17.")
	vt.Autofixes(
		"AUTOFIX: ~/Makefile:1: Replacing \"\\t\\t\" with \"\\t\".",
		"AUTOFIX: ~/Makefile:2: Replacing \"\\t\\t\\t\\t\" with \"\\t\".")
	vt.Result(
		"V.00008=        42",
		"V.00008=        value")
	vt.Run()
}
