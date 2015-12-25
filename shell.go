package main

// Parsing and checking shell commands embedded in Makefiles

import (
	"path"
	"strings"
)

const (
	reMkShellvaruse = `(?:^|[^\$])\$\$\{?(\w+)\}?`
	reVarnameDirect = `(?:[-*+.0-9A-Z_a-z{}\[]+)`
	reShellToken    = `^\s*(` +
		`#.*` + // shell comment
		`|(?:` +
		`'[^']*'` + // single quoted string
		`|"(?:\\.|[^"\\])*"` + // double quoted string
		"|`[^`]*`" + // backticks command execution
		`|\\\$\$` + // a shell-escaped dollar sign
		`|\\[^\$]` + // other escaped characters
		`|\$[\w_]` + // one-character make(1) variable
		`|\$\{[^{}]+\}` + // make(1) variable, ${...}
		`|\$\([^()]+\)` + // make(1) variable, $(...)
		`|\$[/@<^]` + // special make(1) variables
		`|\$\$[0-9A-Z_a-z]+` + // shell variable
		`|\$\$[#?@]` + // special shell variables
		`|\$\$[./]` + // unescaped dollar in shell, followed by punctuation
		`|\$\$\$\$` + // the special pid shell variable
		`|\$\$\{[0-9A-Z_a-z]+\}` + // shell variable in braces
		`|\$\$\(` + // POSIX-style backticks replacement
		`|[^\(\)'\"\\\s;&\|<>` + "`" + `\$]` + // non-special character
		`|\$\{[^\s\"'` + "`" + `]+` + // HACK: nested make(1) variables
		`)+` + // any of the above may be repeated
		`|;;?` +
		`|&&?` +
		`|\|\|?` +
		`|\(` +
		`|\)` +
		`|>&` +
		`|<<?` +
		`|>>?` +
		`|#.*)`
	reShVarassign = `^([A-Z_a-z]\w*)=`
)

// ShellCommandState
type scState uint8

const (
	scstStart scState = iota
	scstCont
	scstInstall
	scstInstallD
	scstMkdir
	scstPax
	scstPaxS
	scstSed
	scstSedE
	scstSet
	scstSetCont
	scstCond
	scstCondCont
	scstCase
	scstCaseIn
	scstCaseLabel
	scstCaseLabelCont
	scstFor
	scstForIn
	scstForCont
	scstEcho
	scstInstallDir
	scstInstallDir2
)

func (st scState) String() string {
	return [...]string{
		"start",
		"continuation",
		"install",
		"install -d",
		"mkdir",
		"pax",
		"pax -s",
		"sed",
		"sed -e",
		"set",
		"set-continuation",
		"cond",
		"cond-continuation",
		"case",
		"case in",
		"case label",
		"case-label-continuation",
		"for",
		"for-in",
		"for-continuation",
		"echo",
		"install-dir",
		"install-dir2",
	}[st]
}

type MkShellLine struct {
	line   *Line
	mkline *MkLine
}

func NewMkShellLine(mkline *MkLine) *MkShellLine {
	return &MkShellLine{mkline.line, mkline}
}

type ShellwordState uint8

const (
	swstPlain ShellwordState = iota
	swstSquot
	swstDquot
	swstDquotBackt
	swstBackt
)

func (st ShellwordState) String() string {
	return [...]string{"plain", "squot", "dquot", "dquot+backt", "backt"}[st]
}

func (msline *MkShellLine) checkShellword(shellword string, checkQuoting bool) {
	defer tracecall("MkShellLine.checklineMkShellword", shellword, checkQuoting)()

	if shellword == "" || hasPrefix(shellword, "#") {
		return
	}

	line := msline.line
	shellcommandsContextType := &Vartype{lkNone, CheckvarShellCommands, []AclEntry{{"*", aclpAllRuntime}}, guNotGuessed}
	shellwordVuc := &VarUseContext{shellcommandsContextType, vucTimeUnknown, vucQuotPlain, vucExtentWord}

	if m, varname, mod := match2(shellword, `^\$\{(`+reVarnameDirect+`)(:[^{}]+)?\}$`); m {
		msline.mkline.checkVaruse(varname, mod, shellwordVuc)
		return
	}

	if matches(shellword, `\$\{PREFIX\}/man(?:$|/)`) {
		line.warn0("Please use ${PKGMANDIR} instead of \"man\".")
	}
	if strings.Contains(shellword, "etc/rc.d") {
		line.warn0("Please use the RCD_SCRIPTS mechanism to install rc.d scripts automatically to ${RCD_SCRIPTS_EXAMPLEDIR}.")
	}

	repl := NewPrefixReplacer(shellword)
	state := swstPlain
outer:
	for repl.rest != "" {
		if G.opts.DebugShell {
			line.debugf("shell state %s: %q", state, repl.rest)
		}

		switch {
		// When parsing inside backticks, it is more
		// reasonable to check the whole shell command
		// recursively, instead of splitting off the first
		// make(1) variable.
		case state == swstBackt || state == swstDquotBackt:
			var backtCommand string
			backtCommand, state = msline.unescapeBackticks(shellword, repl, state)
			setE := true
			msline.checkShellCommand(backtCommand, &setE)

		// Make(1) variables have the same syntax, no matter in which state we are currently.
		case repl.advanceRegexp(`^\$\{(` + reVarnameDirect + `|@)(:[^\{]+)?\}`),
			repl.advanceRegexp(`^\$\((` + reVarnameDirect + `|@])(:[^\)]+)?\)`),
			repl.advanceRegexp(`^\$([\w@])()`):
			varname, mod := repl.m[1], repl.m[2]

			if varname == "@" {
				line := msline.line
				line.warn0("Please use \"${.TARGET}\" instead of \"$@\".")
				explain2(
					"The variable $@ can easily be confused with the shell variable of the",
					"same name, which has a completely different meaning.")
				varname = ".TARGET"
			}

			switch {
			case state == swstPlain && hasSuffix(mod, ":Q"):
				// Fine.
			case state == swstBackt:
				// Don't check anything here, to avoid false positives for tool names.
			case (state == swstSquot || state == swstDquot) && matches(varname, `^(?:.*DIR|.*FILE|.*PATH|.*_VAR|PREFIX|.*BASE|PKGNAME)$`):
				// This is ok if we don't allow these variables to have embedded [\$\\\"\'\`].
			case state == swstDquot && hasSuffix(mod, ":Q"):
				line.warn0("Please don't use the :Q operator in double quotes.")
				explain2(
					"Either remove the :Q or the double quotes. In most cases, it is more",
					"appropriate to remove the double quotes.")
			}

			if varname != "@" {
				vucstate := vucQuotUnknown
				switch state {
				case swstPlain:
					vucstate = vucQuotPlain
				case swstDquot:
					vucstate = vucQuotDquot
				case swstSquot:
					vucstate = vucQuotSquot
				case swstBackt:
					vucstate = vucQuotBackt
				}
				vuc := &VarUseContext{shellcommandsContextType, vucTimeUnknown, vucstate, vucExtentWordpart}
				msline.mkline.checkVaruse(varname, mod, vuc)
			}

		// The syntax of the variable modifiers can get quite
		// hairy. In lack of motivation, we just skip anything
		// complicated, hoping that at least the braces are balanced.
		case repl.advanceStr("${"):
			braces := 1
		skip:
			for repl.rest != "" && braces > 0 {
				switch {
				case repl.advanceStr("}"):
					braces--
				case repl.advanceStr("{"):
					braces++
				case repl.advanceRegexp(`^[^{}]+`):
				// skip
				default:
					break skip
				}
			}

		case state == swstPlain:
			switch {
			case repl.advanceRegexp(`^[!#\%&\(\)*+,\-.\/0-9:;<=>?@A-Z\[\]^_a-z{|}~]+`),
				repl.advanceRegexp(`^\\(?:[ !"#'\(\)*;?\\^{|}]|\$\$)`):
			case repl.advanceStr("'"):
				state = swstSquot
			case repl.advanceStr("\""):
				state = swstDquot
			case repl.advanceStr("`"):
				state = swstBackt
			case repl.advanceRegexp(`^\$\$([0-9A-Z_a-z]+|\#)`),
				repl.advanceRegexp(`^\$\$\{([0-9A-Z_a-z]+|\#)\}`),
				repl.advanceRegexp(`^\$\$(\$)\$`):
				shvarname := repl.m[1]
				if G.opts.WarnQuoting && checkQuoting && msline.variableNeedsQuoting(shvarname) {
					line.warn1("Unquoted shell variable %q.", shvarname)
					explain(
						"When a shell variable contains white-space, it is expanded (split into",
						"multiple words) when it is written as $variable in a shell script.",
						"If that is not intended, you should add quotation marks around it,",
						"like \"$variable\". Then, the variable will always expand to a single",
						"word, preserving all white-space and other special characters.",
						"",
						"Example:",
						"\tfname=\"Curriculum vitae.doc\"",
						"\tcp $fname /tmp",
						"\t# tries to copy the two files \"Curriculum\" and \"Vitae.doc\"",
						"\tcp \"$fname\" /tmp",
						"\t# copies one file, as intended")
				}
			case repl.advanceStr("$@"):
				line.warn2("Please use %q instead of %q.", "${.TARGET}", "$@")
				explain2(
					"It is more readable and prevents confusion with the shell variable of",
					"the same name.")

			case repl.advanceStr("$$@"):
				line.warn0("The $@ shell variable should only be used in double quotes.")

			case repl.advanceStr("$$?"):
				line.warn0("The $? shell variable is often not available in \"set -e\" mode.")

			case repl.advanceStr("$$("):
				line.warn0("Invoking subshells via $(...) is not portable enough.")
				explain2(
					"The Solaris /bin/sh does not know this way to execute a command in a",
					"subshell. Please use backticks (`...`) as a replacement.")

			default:
				break outer
			}

		case state == swstSquot:
			switch {
			case repl.advanceRegexp(`^'`):
				state = swstPlain
			case repl.advanceRegexp(`^[^\$\']+`):
				// just skip
			case repl.advanceRegexp(`^\$\$`):
				// just skip
			default:
				break outer
			}

		case state == swstDquot:
			switch {
			case repl.advanceStr("\""):
				state = swstPlain
			case repl.advanceStr("`"):
				state = swstDquotBackt
			case repl.advanceRegexp("^[^$\"\\\\`]+"):
				// just skip
			case repl.advanceRegexp("^\\\\(?:[\\\\\"`]|\\$\\$)"):
				// just skip
			case repl.advanceRegexp(`^\$\$\{([0-9A-Za-z_]+)\}`),
				repl.advanceRegexp(`^\$\$([0-9A-Z_a-z]+|[!#?@]|\$\$)`):
				shvarname := repl.m[1]
				if G.opts.DebugShell {
					line.debug1("checklineMkShellword: found double-quoted variable %q.", shvarname)
				}
			case repl.advanceStr("$$"):
				line.warn0("Unquoted $ or strange shell variable found.")
			case repl.advanceRegexp(`^\\(.)`):
				char := repl.m[1]
				line.warn2("Please use \"%s\" instead of \"%s\".", "\\\\"+char, "\\"+char)
				explain(
					"Although the current code may work, it is not good style to rely on",
					"the shell passing this escape sequence exactly as is, and not",
					"discarding the backslash. Alternatively you can use single quotes",
					"instead of double quotes.")
			default:
				break outer
			}
		}
	}

	if strings.TrimSpace(repl.rest) != "" {
		line.errorf("Internal pkglint error: checklineMkShellword state=%s, rest=%q, shellword=%q", state, repl.rest, shellword)
	}
}

// Scan for the end of the backticks, checking for single backslashes
// and removing one level of backslashes. Backslashes are only removed
// before a dollar, a backslash or a backtick.
//
// See http://www.opengroup.org/onlinepubs/009695399/utilities/xcu_chap02.html#tag_02_06_03
func (msline *MkShellLine) unescapeBackticks(shellword string, repl *PrefixReplacer, state ShellwordState) (unescaped string, newState ShellwordState) {
	line := msline.line
	for repl.rest != "" {
		switch {
		case repl.advanceStr("`"):
			if state == swstBackt {
				state = swstPlain
			} else {
				state = swstDquot
			}
			return unescaped, state

		case repl.advanceRegexp("^\\\\([\\\\`$])"):
			unescaped += repl.m[1]

		case repl.advanceStr("\\"):
			line.warn0("Backslashes should be doubled inside backticks.")
			unescaped += "\\"

		case state == swstDquotBackt && repl.advanceStr("\""):
			line.warn0("Double quotes inside backticks inside double quotes are error prone.")
			explain(
				"According to the SUSv3, they produce undefined results.",
				"",
				"See the paragraph starting \"Within the backquoted ...\" in",
				"http://www.opengroup.org/onlinepubs/009695399/utilities/xcu_chap02.html")

		case repl.advanceRegexp("^([^\\\\`]+)"):
			unescaped += repl.m[1]

		default:
			line.errorf("Internal pkglint error: checklineMkShellword shellword=%q rest=%q", shellword, repl.rest)
		}
	}
	line.error1("Unfinished backquotes: rest=%q", repl.rest)
	return unescaped, state
}

func (msline *MkShellLine) variableNeedsQuoting(shvarname string) bool {
	switch shvarname {
	case "#", "?":
		return false // Definitely ok
	case "d", "f", "i", "dir", "file", "src", "dst":
		return false // Probably ok
	}
	return true
}

type ShelltextContext struct {
	msline    *MkShellLine
	state     scState
	shellword string
}

func (msline *MkShellLine) checkShellCommandLine(shelltext string) {
	defer tracecall1("MkShellLine.checkShelltext", shelltext)()
	line := msline.line

	if strings.Contains(shelltext, "${SED}") && strings.Contains(shelltext, "${MV}") {
		line.note0("Please use the SUBST framework instead of ${SED} and ${MV}.")
		explain(
			"When converting things, pay attention to \"#\" characters. In shell",
			"commands make(1) does not interpret them as comment character, but",
			"in other lines it does. Therefore, instead of the shell command",
			"",
			"\tsed -e 's,#define foo,,'",
			"",
			"you need to write",
			"",
			"\tSUBST_SED.foo+=\t's,\\#define foo,,'")
	}

	if m, cmd := match1(shelltext, `^@*-(.*(?:MKDIR|INSTALL.*-d|INSTALL_.*_DIR).*)`); m {
		line.note1("You don't need to use \"-\" before %q.", cmd)
	}

	setE := false
	repl := NewPrefixReplacer(shelltext)
	if repl.advanceRegexp(`^\s*([-@]*)(\$\{_PKG_SILENT\}\$\{_PKG_DEBUG\}|\$\{RUN\}|)`) {
		hidden, macro := repl.m[1], repl.m[2]
		msline.checkLineStart(hidden, macro, repl.rest, &setE)
	}

	msline.checkShellCommand(repl.rest, &setE)
}

func (msline *MkShellLine) checkShellCommand(shellcmd string, pSetE *bool) {
	state := scstStart
	shellTokens, rest := splitIntoShellTokens(msline.line, shellcmd)
	for _, shellword := range shellTokens {
		if G.opts.DebugShell {
			msline.line.debugf("checkShellCommand state=%v shellword=%q", state, shellword)
		}

		{
			quotingNecessary := state != scstCase &&
				state != scstForCont &&
				state != scstSetCont &&
				!(state == scstStart && matches(shellword, reShVarassign))
			msline.checkShellword(shellword, quotingNecessary)
		}

		st := &ShelltextContext{msline, state, shellword}
		st.checkCommandStart()
		st.checkConditionalCd()
		if state != scstPaxS && state != scstSedE && state != scstCaseLabel {
			msline.line.checkAbsolutePathname(shellword)
		}
		st.checkAutoMkdirs()
		st.checkInstallMulti()
		st.checkPaxPe()
		st.checkQuoteSubstitution()
		st.checkEchoN()
		st.checkPipeExitcode()
		st.checkSetE(*pSetE)

		if state == scstSet && matches(shellword, `^-.*e`) || state == scstStart && shellword == "${RUN}" {
			*pSetE = true
		}

		state = msline.nextState(state, shellword)
	}

	if rest != "" {
		msline.line.errorf("Internal pkglint error: checklineMkShelltext state=%s rest=%q shellword=%q", state, rest, shellcmd)
	}
}

func (msline *MkShellLine) checkShellCommands(shellcmds string) {
	setE := true
	msline.checkShellCommand(shellcmds, &setE)
	if !hasSuffix(shellcmds, ";") {
		msline.line.warn0("This shell command list should end with a semicolon.")
	}
}

func (msline *MkShellLine) checkLineStart(hidden, macro, rest string, eflag *bool) {
	defer tracecall("MkShellLine.checkLineStart", hidden, macro, rest, eflag)()
	line := msline.line

	switch {
	case !strings.Contains(hidden, "@"):
		// Nothing is hidden at all.

	case hasPrefix(G.mk.target, "show-") || hasSuffix(G.mk.target, "-message"):
		// In these targets commands may be hidden.

	case hasPrefix(rest, "#"):
		// Shell comments may be hidden, since they cannot have side effects.

	default:
		if m, cmd := match1(rest, reShellToken); m {
			switch cmd {
			case "${DELAYED_ERROR_MSG}", "${DELAYED_WARNING_MSG}",
				"${DO_NADA}",
				"${ECHO}", "${ECHO_MSG}", "${ECHO_N}", "${ERROR_CAT}", "${ERROR_MSG}",
				"${FAIL_MSG}",
				"${PHASE_MSG}", "${PRINTF}",
				"${SHCOMMENT}", "${STEP_MSG}",
				"${WARNING_CAT}", "${WARNING_MSG}":
			default:
				line.warn1("The shell command %q should not be hidden.", cmd)
				explain3(
					"Hidden shell commands do not appear on the terminal or in the log file",
					"when they are executed. When they fail, the error message cannot be",
					"assigned to the command, which is very difficult to debug.")
			}
		}
	}

	if strings.Contains(hidden, "-") {
		line.warn0("The use of a leading \"-\" to suppress errors is deprecated.")
		explain3(
			"If you really want to ignore any errors from this command (including",
			"all errors you never thought of), append \"|| ${TRUE}\" to the",
			"command.")
	}

	if macro == "${RUN}" {
		*eflag = true
	}
}

func (ctx *ShelltextContext) checkCommandStart() {
	defer tracecall2("ShelltextContext.checkCommandStart", ctx.state.String(), ctx.shellword)()

	state, shellword := ctx.state, ctx.shellword
	if state != scstStart && state != scstCond {
		return
	}

	switch {
	case shellword == "${RUN}":
	case ctx.handleForbiddenCommand():
	case ctx.handleTool():
	case ctx.handleCommandVariable():
	case matches(shellword, `^(?:\(|\)|:|;|;;|&&|\|\||\{|\}|break|case|cd|continue|do|done|elif|else|esac|eval|exec|exit|export|fi|for|if|read|set|shift|then|umask|unset|while)$`):
	case matches(shellword, `^[\w_]+=.*$`): // Variable assignment
	case hasPrefix(shellword, "./"): // All commands from the current directory are fine.
	case ctx.handleComment():
	default:
		if G.opts.WarnExtra {
			line := ctx.msline.line
			line.warn1("Unknown shell command %q.", shellword)
			explain3(
				"If you want your package to be portable to all platforms that pkgsrc",
				"supports, you should only use shell commands that are covered by the",
				"tools framework.")
		}
	}
}

func (ctx *ShelltextContext) handleTool() bool {
	defer tracecall1("ShelltextContext.handleTool", ctx.shellword)()

	shellword := ctx.shellword
	if !G.globalData.tools[shellword] {
		return false
	}

	if !G.mk.tools[shellword] && !G.mk.tools["g"+shellword] {
		ctx.msline.line.warn1("The %q tool is used but not added to USE_TOOLS.", shellword)
	}

	if G.globalData.toolsVarRequired[shellword] {
		ctx.msline.line.warn2("Please use \"${%s}\" instead of %q.", G.globalData.vartools[shellword], shellword)
	}

	ctx.msline.checkCommandUse(shellword)
	return true
}

func (ctx *ShelltextContext) handleForbiddenCommand() bool {
	switch path.Base(ctx.shellword) {
	case "ktrace", "mktexlsr", "strace", "texconfig", "truss":
	default:
		return false
	}

	ctx.msline.line.error1("%q must not be used in Makefiles.", ctx.shellword)
	explain3(
		"This command must appear in INSTALL scripts, not in the package",
		"Makefile, so that the package also works if it is installed as a binary",
		"package via pkg_add.")
	return true
}

func (ctx *ShelltextContext) handleCommandVariable() bool {
	defer tracecall1("ShelltextContext.handleCommandVariable", ctx.shellword)()

	shellword := ctx.shellword
	if m, varname := match1(shellword, `^\$\{([\w_]+)\}$`); m {

		if toolname := G.globalData.varnameToToolname[varname]; toolname != "" {
			if !G.mk.tools[toolname] {
				ctx.msline.line.warn1("The %q tool is used but not added to USE_TOOLS.", toolname)
			}
			ctx.msline.checkCommandUse(shellword)
			return true
		}

		if vartype := getVariableType(ctx.msline.line, varname); vartype != nil && vartype.checker.name == "ShellCommand" {
			ctx.msline.checkCommandUse(shellword)
			return true
		}

		// When the package author has explicitly defined a command
		// variable, assume it to be valid.
		if G.pkg != nil && G.pkg.vardef[varname] != nil {
			return true
		}
	}
	return false
}

func (ctx *ShelltextContext) handleComment() bool {
	defer tracecall1("ShelltextContext.handleComment", ctx.shellword)()
	line := ctx.msline.line

	shellword := ctx.shellword
	if !hasPrefix(shellword, "#") {
		return false
	}

	semicolon := strings.Contains(shellword, ";")
	multiline := ctx.msline.line.IsMultiline()

	if semicolon {
		line.warn0("A shell comment should not contain semicolons.")
	}
	if multiline {
		line.warn0("A shell comment does not stop at the end of line.")
	}

	if semicolon || multiline {
		explain(
			"When you split a shell command into multiple lines that are continued",
			"with a backslash, they will nevertheless be converted to a single line",
			"before the shell sees them. That means that even if it _looks_ like that",
			"the comment only spans one line in the Makefile, in fact it spans until",
			"the end of the whole shell command. To insert a comment into shell code,",
			"you can pass it as an argument to the ${SHCOMMENT} macro, which expands",
			"to a command doing nothing. Note that any special characters are",
			"nevertheless interpreted by the shell.")
	}
	return true
}

func (ctx *ShelltextContext) checkConditionalCd() {
	if ctx.state == scstCond && ctx.shellword == "cd" {
		ctx.msline.line.error0("The Solaris /bin/sh cannot handle \"cd\" inside conditionals.")
		explain3(
			"When the Solaris shell is in \"set -e\" mode and \"cd\" fails, the",
			"shell will exit, no matter if it is protected by an \"if\" or the",
			"\"||\" operator.")
	}
}

func (ctx *ShelltextContext) checkAutoMkdirs() {
	state, shellword := ctx.state, ctx.shellword

	line := ctx.msline.line
	if (state == scstInstallD || state == scstMkdir) && matches(shellword, `^(?:\$\{DESTDIR\})?\$\{PREFIX(?:|:Q)\}/`) {
		line.warn1("Please use AUTO_MKDIRS instead of %q.",
			ifelseStr(state == scstMkdir, "${MKDIR}", "${INSTALL} -d"))
		explain(
			"Setting AUTO_MKDIRS=yes automatically creates all directories that are",
			"mentioned in the PLIST. If you need additional directories, specify",
			"them in INSTALLATION_DIRS, which is a list of directories relative to",
			"${PREFIX}.")
	}

	if (state == scstInstallDir || state == scstInstallDir2) && !matches(shellword, reMkShellvaruse) {
		if m, dirname := match1(shellword, `^(?:\$\{DESTDIR\})?\$\{PREFIX(?:|:Q)\}/(.*)`); m {
			line.note1("You can use AUTO_MKDIRS=yes or \"INSTALLATION_DIRS+= %s\" instead of this command.", dirname)
			explain(
				"This saves you some typing. You also don't have to think about which of",
				"the many INSTALL_*_DIR macros is appropriate, since INSTALLATION_DIRS",
				"takes care of that.",
				"",
				"Note that you should only do this if the package creates _all_",
				"directories it needs before trying to install files into them.",
				"",
				"Many packages include a list of all needed directories in their PLIST",
				"file. In that case, you can just set AUTO_MKDIRS=yes and be done.")
		}
	}
}

func (ctx *ShelltextContext) checkInstallMulti() {
	if ctx.state == scstInstallDir2 && hasPrefix(ctx.shellword, "$") {
		line := ctx.msline.line
		line.warn0("The INSTALL_*_DIR commands can only handle one directory at a time.")
		explain2(
			"Many implementations of install(1) can handle more, but pkgsrc aims at",
			"maximum portability.")
	}
}

func (ctx *ShelltextContext) checkPaxPe() {
	if ctx.state == scstPax && ctx.shellword == "-pe" {
		line := ctx.msline.line
		line.warn0("Please use the -pp option to pax(1) instead of -pe.")
		explain3(
			"The -pe option tells pax to preserve the ownership of the files, which",
			"means that the installed files will belong to the user that has built",
			"the package.")
	}
}

func (ctx *ShelltextContext) checkQuoteSubstitution() {
	if ctx.state == scstPaxS || ctx.state == scstSedE {
		if false && !matches(ctx.shellword, `"^[\"\'].*[\"\']$`) {
			line := ctx.msline.line
			line.warn1("Substitution commands like %q should always be quoted.", ctx.shellword)
			explain3(
				"Usually these substitution commands contain characters like '*' or",
				"other shell metacharacters that might lead to lookup of matching",
				"filenames and then expand to more than one word.")
		}
	}
}

func (ctx *ShelltextContext) checkEchoN() {
	if ctx.state == scstEcho && ctx.shellword == "-n" {
		ctx.msline.line.warn0("Please use ${ECHO_N} instead of \"echo -n\".")
	}
}

func (ctx *ShelltextContext) checkPipeExitcode() {
	if G.opts.WarnExtra && ctx.state != scstCaseLabelCont && ctx.shellword == "|" {
		line := ctx.msline.line
		line.warn0("The exitcode of the left-hand-side command of the pipe operator is ignored.")
		explain(
			"In a shell command like \"cat *.txt | grep keyword\", if the command",
			"on the left side of the \"|\" fails, this failure is ignored.",
			"",
			"If you need to detect the failure of the left-hand-side command, use",
			"temporary files to save the output of the command.")
	}
}

func (ctx *ShelltextContext) checkSetE(eflag bool) {
	if G.opts.WarnExtra && ctx.shellword == ";" && ctx.state != scstCondCont && ctx.state != scstForCont && !eflag {
		line := ctx.msline.line
		line.warn0("Please switch to \"set -e\" mode before using a semicolon to separate commands.")
		explain(
			"Older versions of the NetBSD make(1) had run the shell commands using",
			"the \"-e\" option of /bin/sh. In 2004, this behavior has been changed to",
			"follow the POSIX conventions, which is to not use the \"-e\" option.",
			"The consequence of this change is that shell programs don't terminate",
			"as soon as an error occurs, but try to continue with the next command.",
			"Imagine what would happen for these commands:",
			"    cd \"HOME\"; cd /nonexistent; rm -rf *",
			"To fix this warning, either insert \"set -e\" at the beginning of this",
			"line or use the \"&&\" operator instead of the semicolon.")
	}
}

// Some shell commands should not be used in the install phase.
func (msline *MkShellLine) checkCommandUse(shellcmd string) {
	if G.mk == nil || !matches(G.mk.target, `^(?:pre|do|post)-install$`) {
		return
	}

	line := msline.line
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
		line.warn1("The shell command %q should not be used in the install phase.", shellcmd)
		explain3(
			"In the install phase, the only thing that should be done is to install",
			"the prepared files to their final location. The file's contents should",
			"not be changed anymore.")

	case "cp", "${CP}":
		line.warn0("${CP} should not be used to install files.")
		explain(
			"The ${CP} command is highly platform dependent and cannot overwrite",
			"files that don't have write permission. Please use ${PAX} instead.",
			"",
			"For example, instead of",
			"\t${CP} -R ${WRKSRC}/* ${PREFIX}/foodir",
			"you should use",
			"\tcd ${WRKSRC} && ${PAX} -wr * ${PREFIX}/foodir")
	}
}

func (msline *MkShellLine) nextState(state scState, shellword string) scState {
	switch {
	case shellword == ";;":
		return scstCaseLabel
	case state == scstCaseLabelCont && shellword == "|":
		return scstCaseLabel
	case matches(shellword, `^[;&\|]+$`):
		return scstStart
	case state == scstStart:
		switch shellword {
		case "${INSTALL}":
			return scstInstall
		case "${MKDIR}":
			return scstMkdir
		case "${PAX}":
			return scstPax
		case "${SED}":
			return scstSed
		case "${ECHO}", "echo":
			return scstEcho
		case "${RUN}", "then", "else", "do", "(":
			return scstStart
		case "set":
			return scstSet
		case "if", "elif", "while":
			return scstCond
		case "case":
			return scstCase
		case "for":
			return scstFor
		default:
			switch {
			case matches(shellword, `^\$\{INSTALL_[A-Z]+_DIR\}$`):
				return scstInstallDir
			case matches(shellword, reShVarassign):
				return scstStart
			default:
				return scstCont
			}
		}
	case state == scstMkdir:
		return scstMkdir
	case state == scstInstall && shellword == "-d":
		return scstInstallD
	case state == scstInstall, state == scstInstallD:
		if matches(shellword, `^-[ogm]$`) {
			return scstCont // XXX: why not keep the state?
		}
		return state
	case state == scstInstallDir && hasPrefix(shellword, "-"):
		return scstCont
	case state == scstInstallDir && hasPrefix(shellword, "$"):
		return scstInstallDir2
	case state == scstInstallDir || state == scstInstallDir2:
		return state
	case state == scstPax && shellword == "-s":
		return scstPaxS
	case state == scstPax && hasPrefix(shellword, "-"):
		return scstPax
	case state == scstPax:
		return scstCont
	case state == scstPaxS:
		return scstPax
	case state == scstSed && shellword == "-e":
		return scstSedE
	case state == scstSed && hasPrefix(shellword, "-"):
		return scstSed
	case state == scstSed:
		return scstCont
	case state == scstSedE:
		return scstSed
	case state == scstSet:
		return scstSetCont
	case state == scstSetCont:
		return scstSetCont
	case state == scstCase:
		return scstCaseIn
	case state == scstCaseIn && shellword == "in":
		return scstCaseLabel
	case state == scstCaseLabel && shellword == "esac":
		return scstCont
	case state == scstCaseLabel:
		return scstCaseLabelCont
	case state == scstCaseLabelCont && shellword == ")":
		return scstStart
	case state == scstCont:
		return scstCont
	case state == scstCond:
		return scstCondCont
	case state == scstCondCont:
		return scstCondCont
	case state == scstFor:
		return scstForIn
	case state == scstForIn && shellword == "in":
		return scstForCont
	case state == scstForCont:
		return scstForCont
	case state == scstEcho:
		return scstCont
	default:
		if G.opts.DebugShell {
			msline.line.errorf("Internal pkglint error: shellword.nextState state=%s shellword=%q", state, shellword)
		}
		return scstStart
	}
}

func splitIntoShellTokens(line *Line, text string) (words []string, rest string) {
	repl := NewPrefixReplacer(text)
	for repl.advanceRegexp(reShellToken) {
		words = append(words, repl.m[1])
	}
	repl.advanceRegexp(`^\s+`)
	return words, repl.rest
}

// Compare devel/bmake/files/str.c, function brk_string.
func splitIntoShellWords(line *Line, text string) (words []string, rest string) {
	reShellWord := `^\s*(` +
		`#.*` + // shell comment
		`|(?:` +
		`'[^']*'` + // single quoted string
		`|"(?:\\.|[^"\\])*"` + // double quoted string
		"|`[^`]*`" + // backticks command execution
		"|[^\\s\"'`\\\\]+" + // normal characters
		`|\\.` + // escaped character
		`)+)` // any of the above may be repeated
	repl := NewPrefixReplacer(text)
	for repl.advanceRegexp(reShellWord) {
		words = append(words, repl.m[1])
	}
	repl.advanceRegexp(`^\s+`)
	return words, repl.rest
}
