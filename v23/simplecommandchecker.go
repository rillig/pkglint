package pkglint

import "path"

// SimpleCommandChecker checks shell commands that consist of variable
// assignments, the command name and arguments, but no redirections.
type SimpleCommandChecker struct {
	cmd    *MkShSimpleCommand
	strcmd *StrCommand
	time   ToolTime

	mkline  *MkLine
	mklines *MkLines
}

func NewSimpleCommandChecker(cmd *MkShSimpleCommand, time ToolTime, mkline *MkLine, mklines *MkLines) *SimpleCommandChecker {
	strcmd := NewStrCommand(cmd)
	return &SimpleCommandChecker{cmd, strcmd, time, mkline, mklines}

}

func (scc *SimpleCommandChecker) Check() {
	if trace.Tracing {
		defer trace.Call(scc.strcmd)()
	}

	scc.checkCommandStart()
	scc.checkRegexReplace()
	scc.checkAutoMkdirs()
	scc.checkInstallMulti()
	scc.checkPaxPe()
	scc.checkEchoN()
}

func (scc *SimpleCommandChecker) checkCommandStart() {
	if trace.Tracing {
		defer trace.Call0()()
	}

	shellword := scc.strcmd.Name
	scc.checkInstallCommand(shellword)

	switch {
	case shellword == "":
		break
	case scc.handleForbiddenCommand():
		break
	case scc.handleTool():
		break
	case scc.handleCommandVariable():
		break
	case scc.handleShellBuiltin():
		break
	case hasPrefix(shellword, "./"): // All commands from the current directory are fine.
		break
	default:
		if G.WarnExtra && !scc.mklines.indentation.DependsOn("OPSYS") {
			scc.mkline.Warnf("Unknown shell command %q.", shellword)
			scc.mkline.Explain(
				"To make the package portable to all platforms that pkgsrc supports,",
				"it should only use shell commands that are covered by the tools framework.",
				"",
				"To run custom shell commands, prefix them with \"./\" or with \"${PREFIX}/\".")
		}
	}
}

// Some shell commands should not be used in the install phase.
func (scc *SimpleCommandChecker) checkInstallCommand(shellcmd string) {
	if trace.Tracing {
		defer trace.Call0()()
	}

	if !matches(scc.mklines.checkAllData.target, `^(?:pre|do|post)-install$`) {
		return
	}

	line := scc.mkline.Line
	switch shellcmd {
	case "${INSTALL}",
		"${INSTALL_DATA}", "${INSTALL_DATA_DIR}",
		"${INSTALL_LIB}", "${INSTALL_LIB_DIR}",
		"${INSTALL_MAN}", "${INSTALL_MAN_DIR}",
		"${INSTALL_PROGRAM}", "${INSTALL_PROGRAM_DIR}",
		"${INSTALL_SCRIPT}",
		"${LIBTOOL}",
		"${LN}",
		"${PAX}":
		return

	case "sed", "${SED}",
		"tr", "${TR}":
		// TODO: Pkglint should not complain when sed and tr are used to transform filenames.
		line.Warnf("The shell command %q should not be used in the install phase.", shellcmd)
		line.Explain(
			"In the install phase, the only thing that should be done is to",
			"install the prepared files to their final location.",
			"The file's contents should not be changed anymore.")

	case "cp", "${CP}":
		line.Warnf("${CP} should not be used to install files.")
		line.Explain(
			"The ${CP} command is highly platform dependent and cannot overwrite read-only files.",
			"Use ${PAX} instead.",
			"",
			"For example, instead of:",
			"\t${CP} -R ${WRKSRC}/* ${PREFIX}/foodir",
			"use:",
			"\tcd ${WRKSRC} && ${PAX} -wr * ${PREFIX}/foodir")
	}
}

func (scc *SimpleCommandChecker) handleForbiddenCommand() bool {
	if trace.Tracing {
		defer trace.Call0()()
	}

	shellword := scc.strcmd.Name
	switch path.Base(shellword) {
	case "mktexlsr", "texconfig":
		scc.Errorf("%q must not be used in Makefiles.", shellword)
		scc.Explain(
			"This command may only appear in INSTALL scripts, not in the package Makefile,",
			"so that the package also works if it is installed as a binary package.")
		return true
	}
	return false
}

// handleTool tests whether the shell command is one of the recognized pkgsrc tools
// and whether the package has added it to USE_TOOLS.
func (scc *SimpleCommandChecker) handleTool() bool {
	if trace.Tracing {
		defer trace.Call0()()
	}

	command := scc.strcmd.Name

	tool, usable := G.Tool(scc.mklines, command, scc.time)

	if tool != nil && !usable {
		scc.mkline.Warnf("The %q tool is used but not added to USE_TOOLS.", command)
	}

	if tool != nil && tool.MustUseVarForm && !containsExpr(command) {
		scc.mkline.Warnf("Use \"${%s}\" instead of %q.", tool.Varname, command)
	}

	return tool != nil
}

func (scc *SimpleCommandChecker) handleCommandVariable() bool {
	if trace.Tracing {
		defer trace.Call0()()
	}

	shellword := scc.strcmd.Name
	expr := NewMkLexer(shellword, nil).Expr()
	if expr == nil {
		return false
	}

	varname := expr.varname

	vartype := G.Pkgsrc.VariableType(scc.mklines, varname)
	if vartype != nil && (vartype.basicType == BtShellCommand || vartype.basicType == BtPathname) {
		scc.checkInstallCommand(shellword)
		return true
	}

	// When the package author has explicitly defined a command
	// variable, assume it to be valid.
	if scc.mklines.allVars.IsDefinedSimilar(varname) {
		return true
	}

	return scc.mklines.pkg != nil && scc.mklines.pkg.vars.IsDefinedSimilar(varname)
}

func (scc *SimpleCommandChecker) handleShellBuiltin() bool {
	switch scc.strcmd.Name {
	case ":", "break", "cd", "continue", "eval", "exec", "exit", "export", "read", "set", "shift", "umask", "unset":
		return true
	}
	return false
}

func (scc *SimpleCommandChecker) checkRegexReplace() {
	if trace.Tracing {
		defer trace.Call0()()
	}

	if !G.Testing {
		return
	}

	checkArg := func(arg string) {
		if matches(arg, `^["'].*["']$`) {
			return
		}

		// Substitution commands that consist only of safe characters cannot
		// have any side effects, therefore they don't need to be quoted.
		if matches(arg, `^([\w,.]|\\.)+$`) {
			return
		}

		scc.Warnf("Substitution commands like %q should always be quoted.", arg)
		scc.Explain(
			"Usually these substitution commands contain characters like '*' or",
			"other shell metacharacters that might lead to lookup of matching",
			"filenames and then expand to more than one word.")
	}

	checkArgAfter := func(opt string) {
		args := scc.strcmd.Args
		for i, arg := range args {
			if i > 0 && args[i-1] == opt {
				checkArg(arg)
			}
		}
	}

	switch scc.strcmd.Name {
	case "${PAX}", "pax":
		checkArgAfter("-s")
	case "${SED}", "sed":
		checkArgAfter("-e")
	}
}

func (scc *SimpleCommandChecker) checkAutoMkdirs() {
	if trace.Tracing {
		defer trace.Call0()()
	}

	cmdname := scc.strcmd.Name
	switch {
	case cmdname == "${MKDIR}":
		break
	case cmdname == "${INSTALL}" && scc.strcmd.HasOption("-d"):
		cmdname = "${INSTALL} -d"
	case matches(cmdname, `^\$\{INSTALL_.*_DIR\}$`):
		// TODO: Replace regex with proper Expr.
		break
	default:
		return
	}

	containsIgnoredVar := func(arg string) bool {
		for _, token := range scc.mkline.Tokenize(arg, false) {
			if token.Expr != nil && matches(token.Expr.varname, `^[_.]*[a-z]`) {
				return true
			}
		}
		return false
	}

	for _, arg := range scc.strcmd.Args {
		if contains(arg, "$$") || containsIgnoredVar(arg) {
			continue
		}

		m, dirname := match1(arg, `^(?:\$\{DESTDIR\})?\$\{PREFIX(?:|:Q)\}/+([^/]\S*)$`)
		if !m {
			continue
		}

		prefixRel := NewRelPathString(dirname).Clean()
		if prefixRel == "." {
			continue
		}

		autoMkdirs := false
		if scc.mklines.pkg != nil {
			plistLine := scc.mklines.pkg.Plist.UnconditionalDirs[prefixRel]
			if plistLine != nil && !containsExpr(plistLine.Line.Text) {
				autoMkdirs = true
			}
		}

		if autoMkdirs {
			scc.Notef("You can use AUTO_MKDIRS=yes or \"INSTALLATION_DIRS+= %s\" instead of %q.",
				prefixRel.String(), cmdname)
			scc.Explain(
				"Many packages include a list of all needed directories in their",
				"PLIST file.",
				"In such a case, you can just set AUTO_MKDIRS=yes and be done.",
				"The pkgsrc infrastructure will then create all directories in advance.",
				"",
				"To create directories that are not mentioned in the PLIST file,",
				"it is easier to just list them in INSTALLATION_DIRS than to execute the",
				"commands explicitly.",
				"That way, you don't have to think about which",
				"of the many INSTALL_*_DIR variables is appropriate, since",
				"INSTALLATION_DIRS takes care of that.")
		} else {
			scc.Notef("You can use \"INSTALLATION_DIRS+= %s\" instead of %q.",
				prefixRel.String(), cmdname)
			scc.Explain(
				"To create directories during installation, it is easier to just",
				"list them in INSTALLATION_DIRS than to execute the commands",
				"explicitly.",
				"That way, you don't have to think about which",
				"of the many INSTALL_*_DIR variables is appropriate,",
				"since INSTALLATION_DIRS takes care of that.")
		}
	}
}

func (scc *SimpleCommandChecker) checkInstallMulti() {
	if trace.Tracing {
		defer trace.Call0()()
	}

	cmd := scc.strcmd

	if hasPrefix(cmd.Name, "${INSTALL_") && hasSuffix(cmd.Name, "_DIR}") {
		prevdir := ""
		for i, arg := range cmd.Args {
			switch {
			case hasPrefix(arg, "-"):
				break
			case i > 0 && (cmd.Args[i-1] == "-m" || cmd.Args[i-1] == "-o" || cmd.Args[i-1] == "-g"):
				break
			default:
				if prevdir != "" {
					scc.mkline.Warnf("The INSTALL_*_DIR commands can only handle one directory at a time.")
					scc.mkline.Explain(
						"Many implementations of install(1) can handle more, but pkgsrc aims",
						"at maximum portability.")
					return
				}
				prevdir = arg
			}
		}
	}
}

func (scc *SimpleCommandChecker) checkPaxPe() {
	if trace.Tracing {
		defer trace.Call0()()
	}

	if (scc.strcmd.Name == "${PAX}" || scc.strcmd.Name == "pax") && scc.strcmd.HasOption("-pe") {
		scc.Warnf("Use the -pp option to pax(1) instead of -pe.")
		scc.Explain(
			"The -pe option tells pax to preserve the ownership of the files.",
			"",
			"When extracting distfiles as root user, this means that whatever numeric uid was",
			"used by the upstream package will also appear in the filesystem during the build.",
			"",
			"The {pre,do,post}-install targets are usually run as root.",
			"When pax -pe is used in these targets, this means that the installed files will",
			"belong to the user that has built the package.")
	}
}

func (scc *SimpleCommandChecker) checkEchoN() {
	if trace.Tracing {
		defer trace.Call0()()
	}

	if scc.strcmd.Name == "${ECHO}" && scc.strcmd.HasOption("-n") {
		scc.Warnf("Use ${ECHO_N} instead of \"echo -n\".")
	}
}

func (scc *SimpleCommandChecker) Errorf(format string, args ...interface{}) {
	scc.mkline.Errorf(format, args...)
}
func (scc *SimpleCommandChecker) Warnf(format string, args ...interface{}) {
	scc.mkline.Warnf(format, args...)
}
func (scc *SimpleCommandChecker) Notef(format string, args ...interface{}) {
	scc.mkline.Notef(format, args...)
}
func (scc *SimpleCommandChecker) Explain(explanation ...string) {
	scc.mkline.Explain(explanation...)
}
