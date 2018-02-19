package main

import (
	"strings"

	"gopkg.in/check.v1"
	"netbsd.org/pkglint/trace"
	"os"
)

func (s *Suite) Test_Pkglint_Main_help(c *check.C) {
	t := s.Init(c)

	exitcode := new(Pkglint).Main("pkglint", "-h")

	c.Check(exitcode, equals, 0)
	c.Check(t.Output(), check.Matches, `^\Qusage: pkglint [options] dir...\E\n(?s).+`)
}

func (s *Suite) Test_Pkglint_Main_version(c *check.C) {
	t := s.Init(c)

	exitcode := new(Pkglint).Main("pkglint", "--version")

	c.Check(exitcode, equals, 0)
	t.CheckOutputLines(
		confVersion)
}

func (s *Suite) Test_Pkglint_Main_no_args(c *check.C) {
	t := s.Init(c)

	exitcode := new(Pkglint).Main("pkglint")

	c.Check(exitcode, equals, 1)
	t.CheckOutputLines(
		"FATAL: \".\" is not inside a pkgsrc tree.")
}

func (s *Suite) Test_Pkglint_Main__only(c *check.C) {
	t := s.Init(c)

	exitcode := new(Pkglint).ParseCommandLine([]string{"pkglint", "-Wall", "-o", ":Q", "--version"})

	if c.Check(exitcode, check.NotNil) {
		c.Check(*exitcode, equals, 0)
	}
	c.Check(G.opts.LogOnly, deepEquals, []string{":Q"})
	t.CheckOutputLines(
		"@VERSION@")
}

func (s *Suite) Test_Pkglint_Main__unknown_option(c *check.C) {
	t := s.Init(c)

	exitcode := new(Pkglint).Main("pkglint", "--unknown-option")

	c.Check(exitcode, equals, 1)
	t.CheckOutputLines(
		"pkglint: unknown option: --unknown-option",
		"",
		"usage: pkglint [options] dir...",
		"",
		"  -C, --check=check,...       enable or disable specific checks",
		"  -d, --debug                 log verbose call traces for debugging",
		"  -e, --explain               explain the diagnostics or give further help",
		"  -f, --show-autofix          show what pkglint can fix automatically",
		"  -F, --autofix               try to automatically fix some errors (experimental)",
		"  -g, --gcc-output-format     mimic the gcc output format",
		"  -h, --help                  print a detailed usage message",
		"  -I, --dumpmakefile          dump the Makefile after parsing",
		"  -i, --import                prepare the import of a wip package",
		"  -m, --log-verbose           allow the same log message more than once",
		"  -o, --only                  only log messages containing the given text",
		"  -p, --profiling             profile the executing program",
		"  -q, --quiet                 don't print a summary line when finishing",
		"  -r, --recursive             check subdirectories, too",
		"  -s, --source                show the source lines together with diagnostics",
		"  -V, --version               print the version number of pkglint",
		"  -W, --warning=warning,...   enable or disable groups of warnings",
		"",
		"  Flags for -C, --check:",
		"    all            all of the following",
		"    none           none of the following",
		"    ALTERNATIVES   check ALTERNATIVES files (enabled)",
		"    bl3            check buildlink3.mk files (enabled)",
		"    DESCR          check DESCR file (enabled)",
		"    distinfo       check distinfo file (enabled)",
		"    extra          check various additional files (disabled)",
		"    global         inter-package checks (disabled)",
		"    INSTALL        check INSTALL and DEINSTALL scripts (enabled)",
		"    Makefile       check Makefiles (enabled)",
		"    MESSAGE        check MESSAGE file (enabled)",
		"    mk             check other .mk files (enabled)",
		"    patches        check patches (enabled)",
		"    PLIST          check PLIST files (enabled)",
		"",
		"  Flags for -W, --warning:",
		"    all          all of the following",
		"    none         none of the following",
		"    absname      warn about use of absolute file names (enabled)",
		"    directcmd    warn about use of direct command names instead of Make variables (enabled)",
		"    extra        enable some extra warnings (disabled)",
		"    order        warn if Makefile entries are unordered (disabled)",
		"    perm         warn about unforeseen variable definition and use (disabled)",
		"    plist-depr   warn about deprecated paths in PLISTs (disabled)",
		"    plist-sort   warn about unsorted entries in PLISTs (disabled)",
		"    quoting      warn about quoting issues (disabled)",
		"    space        warn about inconsistent use of white-space (disabled)",
		"    style        warn about stylistic issues (disabled)",
		"    types        do some simple type checking in Makefiles (enabled)",
		"",
		"  (Prefix a flag with \"no-\" to disable it.)")
}

// go test -c -covermode count
// pkgsrcdir=...
// env PKGLINT_TESTCMDLINE="$pkgsrcdir -r" ./pkglint.test -test.coverprofile pkglint.cov
// go tool cover -html=pkglint.cov -o coverage.html
func (s *Suite) Test_Pkglint_coverage(c *check.C) {
	cmdline := os.Getenv("PKGLINT_TESTCMDLINE")
	if cmdline != "" {
		G.logOut, G.logErr, trace.Out = NewSeparatorWriter(os.Stdout), NewSeparatorWriter(os.Stderr), os.Stdout
		new(Pkglint).Main(append([]string{"pkglint"}, splitOnSpace(cmdline)...)...)
	}
}

func (s *Suite) Test_Pkglint_CheckDirent__outside(c *check.C) {
	t := s.Init(c)

	t.SetupFileLines("empty")

	new(Pkglint).CheckDirent(t.TmpDir())

	t.CheckOutputLines(
		"ERROR: ~: Cannot determine the pkgsrc root directory for \"~\".")
}

func (s *Suite) Test_Pkglint_CheckDirent(c *check.C) {
	t := s.Init(c)

	t.SetupFileLines("mk/bsd.pkg.mk")
	t.SetupFileLines("category/package/Makefile")
	t.SetupFileLines("category/Makefile")
	t.SetupFileLines("Makefile")
	G.globalData.Pkgsrcdir = t.TmpDir()
	pkglint := new(Pkglint)

	pkglint.CheckDirent(t.TmpDir())

	t.CheckOutputLines(
		"ERROR: ~/Makefile: Must not be empty.")

	pkglint.CheckDirent(t.TempFilename("category"))

	t.CheckOutputLines(
		"ERROR: ~/category/Makefile: Must not be empty.")

	pkglint.CheckDirent(t.TempFilename("category/package"))

	t.CheckOutputLines(
		"ERROR: ~/category/package/Makefile: Must not be empty.")

	pkglint.CheckDirent(t.TempFilename("category/package/nonexistent"))

	t.CheckOutputLines(
		"ERROR: ~/category/package/nonexistent: No such file or directory.")
}

func (s *Suite) Test_resolveVariableRefs__circular_reference(c *check.C) {
	t := s.Init(c)

	mkline := t.NewMkLine("fname", 1, "GCC_VERSION=${GCC_VERSION}")
	G.Pkg = NewPackage(".")
	G.Pkg.vardef["GCC_VERSION"] = mkline

	resolved := resolveVariableRefs("gcc-${GCC_VERSION}")

	c.Check(resolved, equals, "gcc-${GCC_VERSION}")
}

func (s *Suite) Test_resolveVariableRefs__multilevel(c *check.C) {
	t := s.Init(c)

	mkline1 := t.NewMkLine("fname", 10, "_=${SECOND}")
	mkline2 := t.NewMkLine("fname", 11, "_=${THIRD}")
	mkline3 := t.NewMkLine("fname", 12, "_=got it")
	G.Pkg = NewPackage(".")
	defineVar(mkline1, "FIRST")
	defineVar(mkline2, "SECOND")
	defineVar(mkline3, "THIRD")

	resolved := resolveVariableRefs("you ${FIRST}")

	c.Check(resolved, equals, "you got it")
}

func (s *Suite) Test_resolveVariableRefs__special_chars(c *check.C) {
	t := s.Init(c)

	mkline := t.NewMkLine("fname", 10, "_=x11")
	G.Pkg = NewPackage("category/pkg")
	G.Pkg.vardef["GST_PLUGINS0.10_TYPE"] = mkline

	resolved := resolveVariableRefs("gst-plugins0.10-${GST_PLUGINS0.10_TYPE}/distinfo")

	c.Check(resolved, equals, "gst-plugins0.10-x11/distinfo")
}

func (s *Suite) Test_ChecklinesDescr(c *check.C) {
	t := s.Init(c)

	lines := t.NewLines("DESCR",
		strings.Repeat("X", 90),
		"", "", "", "", "", "", "", "", "10",
		"Try ${PREFIX}",
		"", "", "", "", "", "", "", "", "20",
		"", "", "", "", "", "", "", "", "", "30")

	ChecklinesDescr(lines)

	t.CheckOutputLines(
		"WARN: DESCR:1: Line too long (should be no more than 80 characters).",
		"NOTE: DESCR:11: Variables are not expanded in the DESCR file.",
		"WARN: DESCR:25: File too long (should be no more than 24 lines).")
}

func (s *Suite) Test_ChecklinesMessage__short(c *check.C) {
	t := s.Init(c)

	lines := t.NewLines("MESSAGE",
		"one line")

	ChecklinesMessage(lines)

	t.CheckOutputLines(
		"WARN: MESSAGE:1: File too short.")
}

func (s *Suite) Test_ChecklinesMessage__malformed(c *check.C) {
	t := s.Init(c)

	lines := t.NewLines("MESSAGE",
		"1",
		"2",
		"3",
		"4",
		"5")

	ChecklinesMessage(lines)

	t.CheckOutputLines(
		"WARN: MESSAGE:1: Expected a line of exactly 75 \"=\" characters.",
		"ERROR: MESSAGE:1: Expected \"$"+"NetBSD$\".",
		"WARN: MESSAGE:5: Expected a line of exactly 75 \"=\" characters.")
}

func (s *Suite) Test_ChecklinesMessage__autofix(c *check.C) {
	t := s.Init(c)

	t.SetupCommandLine("-Wall", "--autofix")
	lines := t.SetupFileLines("MESSAGE",
		"1",
		"2",
		"3",
		"4",
		"5")

	ChecklinesMessage(lines)

	t.CheckOutputLines(
		"AUTOFIX: ~/MESSAGE:1: Inserting a line \"===================================="+
			"=======================================\" before this line.",
		"AUTOFIX: ~/MESSAGE:1: Inserting a line \"$NetBSD$\" before this line.",
		"AUTOFIX: ~/MESSAGE:5: Inserting a line \"===================================="+
			"=======================================\" after this line.")
	t.CheckFileLines("MESSAGE",
		"===========================================================================",
		"$NetBSD$",
		"1",
		"2",
		"3",
		"4",
		"5",
		"===========================================================================")
}

func (s *Suite) Test_GlobalData_Latest(c *check.C) {
	t := s.Init(c)

	G.globalData.Pkgsrcdir = t.TmpDir()

	latest1 := G.globalData.Latest("lang", `^python[0-9]+$`, "../../lang/$0")

	c.Check(latest1, equals, "")
	t.CheckOutputLines(
		"ERROR: Cannot find latest version of \"^python[0-9]+$\" in \"~\".")

	t.SetupFileLines("lang/Makefile")
	G.globalData.latest = nil

	latest2 := G.globalData.Latest("lang", `^python[0-9]+$`, "../../lang/$0")

	c.Check(latest2, equals, "")
	t.CheckOutputLines(
		"ERROR: Cannot find latest version of \"^python[0-9]+$\" in \"~\".")

	t.SetupFileLines("lang/python27/Makefile")
	G.globalData.latest = nil

	latest3 := G.globalData.Latest("lang", `^python[0-9]+$`, "../../lang/$0")

	c.Check(latest3, equals, "../../lang/python27")
	t.CheckOutputEmpty()

	t.SetupFileLines("lang/python35/Makefile")
	G.globalData.latest = nil

	latest4 := G.globalData.Latest("lang", `^python[0-9]+$`, "../../lang/$0")

	c.Check(latest4, equals, "../../lang/python35")
	t.CheckOutputEmpty()
}
