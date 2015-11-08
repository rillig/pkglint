package main

import (
	"os"
)

type CmdOpts struct {
	optCheckAlternatives,
	optCheckBuildlink3,
	optCheckDescr,
	optCheckDistinfo,
	optCheckExtra,
	optCheckGlobal,
	optCheckInstall,
	optCheckMakefile,
	optCheckMessage,
	optCheckMk,
	optCheckPatches,
	optCheckPlist bool

	optDebugInclude,
	optDebugMisc,
	optDebugPatches,
	optDebugQuoting,
	optDebugShell,
	optDebugTools,
	optDebugTrace,
	optDebugUnchecked,
	optDebugUnused,
	optDebugVartypes,
	optDebugVaruse bool

	optWarnAbsname,
	optWarnDirectcmd,
	optWarnExtra,
	optWarnOrder,
	optWarnPerm,
	optWarnPlistDepr,
	optWarnPlistSort,
	optWarnQuoting,
	optWarnSpace,
	optWarnStyle,
	optWarnTypes,
	optWarnVarorder bool

	optExplain,
	optAutofix,
	optGccOutput,
	optPrintHelp,
	optDumpMakefile,
	optImport,
	optQuiet,
	optRecursive,
	optPrintSource,
	optPrintVersion bool
	optPkgsrcdir,
	optRcsIds string

	args []string
}

func ParseCommandLine(args []string) CmdOpts {
	result := CmdOpts{}
	opts := &Options{}

	check := opts.AddFlagGroup('C', "check", "check,...", "Enable or disable specific checks")
	check.AddFlagVar("ALTERNATIVES", &result.optCheckAlternatives, true, "check ALTERNATIVES files")
	check.AddFlagVar("bl3", &result.optCheckBuildlink3, true, "check buildlink3.mk files")
	check.AddFlagVar("DESCR", &result.optCheckDescr, true, "check DESCR file")
	check.AddFlagVar("distinfo", &result.optCheckDistinfo, true, "check distinfo file")
	check.AddFlagVar("extra", &result.optCheckExtra, false, "check various additional files")
	check.AddFlagVar("global", &result.optCheckGlobal, false, "inter-package checks")
	check.AddFlagVar("INSTALL", &result.optCheckInstall, true, "check INSTALL and DEINSTALL scripts")
	check.AddFlagVar("Makefile", &result.optCheckMakefile, true, "check Makefiles")
	check.AddFlagVar("MESSAGE", &result.optCheckMessage, true, "check MESSAGE file")
	check.AddFlagVar("mk", &result.optCheckMk, true, "check other .mk files")
	check.AddFlagVar("patches", &result.optCheckPatches, true, "check patches")
	check.AddFlagVar("PLIST", &result.optCheckPlist, true, "check PLIST files")

	debug := opts.AddFlagGroup('D', "debugging", "debug,...", "Enable or disable debugging categories")
	debug.AddFlagVar("include", &result.optDebugInclude, false, "included files")
	debug.AddFlagVar("misc", &result.optDebugMisc, false, "all things that didn't fit elsewhere")
	debug.AddFlagVar("patches", &result.optDebugPatches, false, "the states of the patch parser")
	debug.AddFlagVar("quoting", &result.optDebugQuoting, false, "additional information about quoting")
	debug.AddFlagVar("shell", &result.optDebugShell, false, "the parsers for shell words and shell commands")
	debug.AddFlagVar("tools", &result.optDebugTools, false, "the tools framework")
	debug.AddFlagVar("trace", &result.optDebugTrace, false, "follow subroutine calls")
	debug.AddFlagVar("unchecked", &result.optDebugUnchecked, false, "show the current limitations of pkglint")
	debug.AddFlagVar("unused", &result.optDebugUnused, false, "unused variables")
	debug.AddFlagVar("vartypes", &result.optDebugVartypes, false, "additional type information")
	debug.AddFlagVar("varuse", &result.optDebugVaruse, false, "contexts where variables are used")

	opts.AddFlagVar('e', "explain", &result.optExplain, false, "Explain the diagnostics or give further help")
	opts.AddFlagVar('F', "autofix", &result.optAutofix, false, "Try to automatically fix some errors (experimental)")
	opts.AddFlagVar('g', "gcc-output-format", &result.optGccOutput, false, "Mimic the gcc output format")
	opts.AddFlagVar('h', "help", &result.optPrintHelp, false, "print a detailed usage message")
	opts.AddFlagVar('I', "dumpmakefile", &result.optDumpMakefile, false, "Dump the Makefile after parsing")
	opts.AddFlagVar('i', "import", &result.optImport, false, "Prepare the import of a wip package")
	opts.AddStrVar('p', "pkgsrcdir", &result.optPkgsrcdir, "", "Set the root directory of pkgsrc explicitly")
	opts.AddFlagVar('q', "quiet", &result.optQuiet, false, "Don't print a summary line when finishing")
	opts.AddStrVar('R', "rcsidstring", &result.optRcsIds, "NetBSD", "Set the allowed RCS Id strings")
	opts.AddFlagVar('r', "recursive", &result.optRecursive, false, "Recursive---check subdirectories, too")
	opts.AddFlagVar('s', "source", &result.optPrintSource, false, "Show the source lines together with diagnostics")
	opts.AddFlagVar('V', "version", &result.optPrintVersion, false, "print the version number of pkglint")

	warn := opts.AddFlagGroup('W', "warning", "warning,...", "Enable or disable groups of warnings")
	warn.AddFlagVar("absname", &result.optWarnAbsname, true, "warn about use of absolute file names")
	warn.AddFlagVar("directcmd", &result.optWarnDirectcmd, true, "warn about use of direct command names instead of Make variables")
	warn.AddFlagVar("extra", &result.optWarnExtra, false, "enable some extra warnings")
	warn.AddFlagVar("order", &result.optWarnOrder, true, "warn if Makefile entries are unordered")
	warn.AddFlagVar("perm", &result.optWarnPerm, false, "warn about unforeseen variable definition and use")
	warn.AddFlagVar("plist-depr", &result.optWarnPlistDepr, false, "warn about deprecated paths in PLISTs")
	warn.AddFlagVar("plist-sort", &result.optWarnPlistSort, false, "warn about unsorted entries in PLISTs")
	warn.AddFlagVar("quoting", &result.optWarnQuoting, false, "warn about quoting issues")
	warn.AddFlagVar("space", &result.optWarnSpace, false, "warn about inconsistent use of white-space")
	warn.AddFlagVar("style", &result.optWarnStyle, false, "warn about stylistic issues")
	warn.AddFlagVar("types", &result.optWarnTypes, true, "do some simple type checking in Makefiles")
	warn.AddFlagVar("varorder", &result.optWarnVarorder, false, "warn about the ordering of variables")

	opts.Parse(args)
	if result.optPrintHelp {
		opts.Help("pkglint [options] dir...")
		os.Exit(0)
	}
	return result
}
