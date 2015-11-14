package main

import (
	"io"
	"os"
)

type CmdOpts struct {
	CheckAlternatives,
	CheckBuildlink3,
	CheckDescr,
	CheckDistinfo,
	CheckExtra,
	CheckGlobal,
	CheckInstall,
	CheckMakefile,
	CheckMessage,
	CheckMk,
	CheckPatches,
	CheckPlist bool

	DebugInclude,
	DebugMisc,
	DebugPatches,
	DebugQuoting,
	DebugShell,
	DebugTools,
	DebugTrace,
	DebugUnchecked,
	DebugUnused,
	DebugVartypes,
	DebugVaruse bool

	WarnAbsname,
	WarnDirectcmd,
	WarnExtra,
	WarnOrder,
	WarnPerm,
	WarnPlistDepr,
	WarnPlistSort,
	WarnQuoting,
	WarnSpace,
	WarnStyle,
	WarnTypes,
	WarnVarorder bool

	Explain,
	Autofix,
	GccOutput,
	PrintHelp,
	DumpMakefile,
	Import,
	Quiet,
	Recursive,
	PrintSource,
	PrintVersion bool
	Pkgsrcdir string

	args []string
}

func ParseCommandLine(args []string, out io.Writer) CmdOpts {
	result := CmdOpts{}
	opts := NewOptions(out)

	check := opts.AddFlagGroup('C', "check", "check,...", "Enable or disable specific checks")
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

	debug := opts.AddFlagGroup('D', "debugging", "debug,...", "Enable or disable debugging categories")
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

	opts.AddFlagVar('e', "explain", &result.Explain, false, "Explain the diagnostics or give further help")
	opts.AddFlagVar('F', "autofix", &result.Autofix, false, "Try to automatically fix some errors (experimental)")
	opts.AddFlagVar('g', "gcc-output-format", &result.GccOutput, false, "Mimic the gcc output format")
	opts.AddFlagVar('h', "help", &result.PrintHelp, false, "print a detailed usage message")
	opts.AddFlagVar('I', "dumpmakefile", &result.DumpMakefile, false, "Dump the Makefile after parsing")
	opts.AddFlagVar('i', "import", &result.Import, false, "Prepare the import of a wip package")
	opts.AddStrVar('p', "pkgsrcdir", &result.Pkgsrcdir, "", "Set the root directory of pkgsrc explicitly")
	opts.AddFlagVar('q', "quiet", &result.Quiet, false, "Don't print a summary line when finishing")
	opts.AddFlagVar('r', "recursive", &result.Recursive, false, "Recursive---check subdirectories, too")
	opts.AddFlagVar('s', "source", &result.PrintSource, false, "Show the source lines together with diagnostics")
	opts.AddFlagVar('V', "version", &result.PrintVersion, false, "print the version number of pkglint")

	warn := opts.AddFlagGroup('W', "warning", "warning,...", "Enable or disable groups of warnings")
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

	result.args, _ = opts.Parse(args)

	if result.PrintHelp {
		opts.Help("pkglint [options] dir...")
		os.Exit(0)
	}
	return result
}
