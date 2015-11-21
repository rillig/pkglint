package main

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
)

const confMake = "@BMAKE@"
const confVersion = "@VERSION@"

func main() {
	G = new(GlobalVars)
	G.logOut, G.logErr, G.traceOut = os.Stdout, os.Stderr, os.Stdout

	defer func() {
		if r := recover(); r != nil {
			if _, ok := r.(pkglintFatal); ok {
				os.Exit(1)
			}
			panic(r)
		}
	}()
	new(Pkglint).Main(os.Args...)

	if G.errors != 0 {
		os.Exit(1)
	}
}

type Pkglint struct{}

func (p *Pkglint) Main(args ...string) {
	G.opts = p.ParseCommandLine(args, G.logOut)
	if G.opts.PrintHelp {
		return
	}
	if G.opts.PrintVersion {
		fmt.Fprintf(G.logOut, "%s\n", confVersion)
		return
	}

	for _, arg := range G.opts.args {
		G.todo = append(G.todo, filepath.ToSlash(arg))
	}
	if len(G.todo) == 0 {
		G.todo = []string{"."}
	}

	G.globalData.Initialize()

	for len(G.todo) != 0 {
		item := G.todo[0]
		G.todo = G.todo[1:]
		CheckDirent(item)
	}

	checktoplevelUnusedLicenses()
	printSummary()
}

func (p *Pkglint) ParseCommandLine(args []string, out io.Writer) CmdOpts {
	result := CmdOpts{}
	opts := NewOptions(out)

	check := opts.AddFlagGroup('C', "check", "check,...", "enable or disable specific checks")
	check.AddFlagVar("ALTERNATIVES", &result.CheckAlternatives, true, "check ALTERNATIVES files")
	check.AddFlagVar("bl3", &result.CheckBuildlink3, true, "check buildlink3.mk files")
	check.AddFlagVar("DESCR", &result.CheckDescr, true, "check DESCR file")
	check.AddFlagVar("distinfo", &result.CheckDistinfo, true, "check distinfo file")
	check.AddFlagVar("extra", &result.CheckExtra, false, "check various additional files")
	check.AddFlagVar("global", &result.CheckGlobal, false, "inter-package checks")
	check.AddFlagVar("INSTALL", &result.CheckInstall, true, "check INSTALL and DEINSTALL scripts")
	check.AddFlagVar("Makefile", &result.CheckMakefile, true, "check Makefiles")
	check.AddFlagVar("MESSAGE", &result.CheckMessage, true, "check MESSAGE file")
	check.AddFlagVar("mk", &result.CheckMk, true, "check other .mk files")
	check.AddFlagVar("patches", &result.CheckPatches, true, "check patches")
	check.AddFlagVar("PLIST", &result.CheckPlist, true, "check PLIST files")

	debug := opts.AddFlagGroup('D', "debugging", "debug,...", "enable or disable debugging categories")
	debug.AddFlagVar("include", &result.DebugInclude, false, "included files")
	debug.AddFlagVar("misc", &result.DebugMisc, false, "all things that didn't fit elsewhere")
	debug.AddFlagVar("patches", &result.DebugPatches, false, "the states of the patch parser")
	debug.AddFlagVar("quoting", &result.DebugQuoting, false, "additional information about quoting")
	debug.AddFlagVar("shell", &result.DebugShell, false, "the parsers for shell words and shell commands")
	debug.AddFlagVar("tools", &result.DebugTools, false, "the tools framework")
	debug.AddFlagVar("trace", &result.DebugTrace, false, "follow subroutine calls")
	debug.AddFlagVar("unchecked", &result.DebugUnchecked, false, "show the current limitations of pkglint")
	debug.AddFlagVar("unused", &result.DebugUnused, false, "unused variables")
	debug.AddFlagVar("vartypes", &result.DebugVartypes, false, "additional type information")
	debug.AddFlagVar("varuse", &result.DebugVaruse, false, "contexts where variables are used")

	opts.AddFlagVar('e', "explain", &result.Explain, false, "explain the diagnostics or give further help")
	opts.AddFlagVar('F', "autofix", &result.Autofix, false, "try to automatically fix some errors (experimental)")
	opts.AddFlagVar('g', "gcc-output-format", &result.GccOutput, false, "mimic the gcc output format")
	opts.AddFlagVar('h', "help", &result.PrintHelp, false, "print a detailed usage message")
	opts.AddFlagVar('I', "dumpmakefile", &result.DumpMakefile, false, "dump the Makefile after parsing")
	opts.AddFlagVar('i', "import", &result.Import, false, "prepare the import of a wip package")
	opts.AddFlagVar('q', "quiet", &result.Quiet, false, "don't print a summary line when finishing")
	opts.AddFlagVar('r', "recursive", &result.Recursive, false, "check subdirectories, too")
	opts.AddFlagVar('s', "source", &result.PrintSource, false, "show the source lines together with diagnostics")
	opts.AddFlagVar('V', "version", &result.PrintVersion, false, "print the version number of pkglint")

	warn := opts.AddFlagGroup('W', "warning", "warning,...", "enable or disable groups of warnings")
	warn.AddFlagVar("absname", &result.WarnAbsname, true, "warn about use of absolute file names")
	warn.AddFlagVar("directcmd", &result.WarnDirectcmd, true, "warn about use of direct command names instead of Make variables")
	warn.AddFlagVar("extra", &result.WarnExtra, false, "enable some extra warnings")
	warn.AddFlagVar("order", &result.WarnOrder, true, "warn if Makefile entries are unordered")
	warn.AddFlagVar("perm", &result.WarnPerm, false, "warn about unforeseen variable definition and use")
	warn.AddFlagVar("plist-depr", &result.WarnPlistDepr, false, "warn about deprecated paths in PLISTs")
	warn.AddFlagVar("plist-sort", &result.WarnPlistSort, false, "warn about unsorted entries in PLISTs")
	warn.AddFlagVar("quoting", &result.WarnQuoting, false, "warn about quoting issues")
	warn.AddFlagVar("space", &result.WarnSpace, false, "warn about inconsistent use of white-space")
	warn.AddFlagVar("style", &result.WarnStyle, false, "warn about stylistic issues")
	warn.AddFlagVar("types", &result.WarnTypes, true, "do some simple type checking in Makefiles")
	warn.AddFlagVar("varorder", &result.WarnVarorder, false, "warn about the ordering of variables")

	result.args, _ = opts.Parse(args) // XXX: error handling

	if result.PrintHelp {
		opts.Help("pkglint [options] dir...")
	}
	return result
}
