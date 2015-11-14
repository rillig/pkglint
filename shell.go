package main

// Parsing and checking shell commands embedded in Makefiles

import (
	"path"
	"strings"
	"unicode"
)

func checklineMkShellword(line *Line, word string, checkQuoting bool) {
	(&MkShellLine{line}).checklineMkShellword(word, checkQuoting)
}
func checklineMkShellcmdUse(line *Line, shellcmd string) {
	(&MkShellLine{line}).checkCommandUse(shellcmd)
}
func checklineMkShellcmd(line *Line, shellcmd string) {
	checklineMkText(line, shellcmd)
	(&MkShellLine{line}).checklineMkShelltext(shellcmd)
}

type ShellCommandState string

const (
	SCST_START           ShellCommandState = "start"
	SCST_CONT            ShellCommandState = "continuation"
	SCST_INSTALL         ShellCommandState = "install"
	SCST_INSTALL_D       ShellCommandState = "install -d"
	SCST_MKDIR           ShellCommandState = "mkdir"
	SCST_PAX             ShellCommandState = "pax"
	SCST_PAX_S           ShellCommandState = "pax -s"
	SCST_SED             ShellCommandState = "sed"
	SCST_SED_E           ShellCommandState = "sed -e"
	SCST_SET             ShellCommandState = "set"
	SCST_SET_CONT        ShellCommandState = "set-continuation"
	SCST_COND            ShellCommandState = "cond"
	SCST_COND_CONT       ShellCommandState = "cond-continuation"
	SCST_CASE            ShellCommandState = "case"
	SCST_CASE_IN         ShellCommandState = "case in"
	SCST_CASE_LABEL      ShellCommandState = "case label"
	SCST_CASE_LABEL_CONT ShellCommandState = "case-label-continuation"
	SCST_CASE_PAREN      ShellCommandState = "case-paren"
	SCST_FOR             ShellCommandState = "for"
	SCST_FOR_IN          ShellCommandState = "for-in"
	SCST_FOR_CONT        ShellCommandState = "for-continuation"
	SCST_ECHO            ShellCommandState = "echo"
	SCST_INSTALL_DIR     ShellCommandState = "install-dir"
	SCST_INSTALL_DIR2    ShellCommandState = "install-dir2"
)

type MkShellLine struct {
	line *Line
}

func (msline *MkShellLine) checklineMkShellword(shellword string, checkQuoting bool) {
	line := msline.line
	defer tracecall("checklineMkShellword", shellword, checkQuoting)()

	if shellword == "" {
		return
	}

	shellcommandContextType := newBasicVartype(LK_NONE, "ShellCommand", []AclEntry{{"*", "adsu"}}, NOT_GUESSED)
	shellwordVuc := &VarUseContext{VUC_TIME_UNKNOWN, shellcommandContextType, VUC_SHW_PLAIN, VUC_EXT_WORD}

	if m, varname, mod := match2(shellword, `^\$\{(`+reVarname+`)(:[^{}]+)?\}$`); m {
		checklineMkVaruse(line, varname, mod, shellwordVuc)
		return
	}

	if matches(shellword, `\$\{PREFIX\}/man(?:$|/)`) {
		line.warnf("Please use ${PKGMANDIR} instead of \"man\".")
	}
	if contains(shellword, "etc/rc.d") {
		line.warnf("Please use the RCD_SCRIPTS mechanism to install rc.d scripts automatically to ${RCD_SCRIPTS_EXAMPLEDIR}.")
	}

	type ShellwordState string
	const (
		SWST_PLAIN       ShellwordState = "plain"
		SWST_SQUOT       ShellwordState = "squot"
		SWST_DQUOT       ShellwordState = "dquot"
		SWST_DQUOT_BACKT ShellwordState = "dquot+backt"
		SWST_BACKT       ShellwordState = "backt"
	)

	rest := shellword
	state := SWST_PLAIN
outer:
	for rest != "" {
		_ = G.opts.optDebugShell && line.debugf("shell state %s: %q", state, rest)

		var m []string
		switch {
		// When parsing inside backticks, it is more
		// reasonable to check the whole shell command
		// recursively, instead of splitting off the first
		// make(1) variable.
		case state == SWST_BACKT || state == SWST_DQUOT_BACKT:
			// Scan for the end of the backticks, checking
			// for single backslashes and removing one level
			// of backslashes. Backslashes are only removed
			// before a dollar, a backslash or a backtick.
			//
			// References:
			// * http://www.opengroup.org/onlinepubs/009695399/utilities/xcu_chap02.html#tag_02_06_03
			stripped := ""
			for rest != "" {
				switch {
				case replacestart(&rest, &m, "^`"):
					if state == SWST_BACKT {
						state = SWST_PLAIN
					} else {
						state = SWST_DQUOT
					}
					goto endOfBackticks

				case replacestart(&rest, &m, "^\\\\([\\\\`$])"):
					stripped += m[1]

				case replacestart(&rest, &m, `^(\\)`):
					line.warnf("Backslashes should be doubled inside backticks.")
					stripped += m[1]

				case state == SWST_DQUOT_BACKT && replacestart(&rest, &m, `^"`):
					line.warnf("Double quotes inside backticks inside double quotes are error prone.")
					line.explain(
						"According to the SUSv3, they produce undefined results.",
						"",
						"See the paragraph starting \"Within the backquoted ...\" in",
						"http://www.opengroup.org/onlinepubs/009695399/utilities/xcu_chap02.html")

				case replacestart(&rest, &m, "^([^\\\\`]+)"):
					stripped += m[1]

				default:
					line.errorf("Internal pkglint error: checklineMkShellword shellword=%q rest=%q", shellword, rest)
				}
			}
			line.errorf("Unfinished backquotes: rest=%q", rest)

		endOfBackticks:
			msline.checklineMkShelltext(stripped)

		// Make(1) variables have the same syntax, no matter in which state we are currently.
		case replacestart(&rest, &m, `^\$\{(`+reVarname+`|@)(:[^\{]+)?\}`),
			replacestart(&rest, &m, `^\$\((`+reVarname+`|@])(:[^\)]+)?\)`),
			replacestart(&rest, &m, `^\$([\w@])()`):
			varname, mod := m[1], m[2]

			if varname == "@" {
				line.warnf("Please use \"${.TARGET}\" instead of \"$@\".")
				line.explain(
					"The variable $@ can easily be confused with the shell variable of the",
					"same name, which has a completely different meaning.")
				varname = ".TARGET"
			}

			switch {
			case state == SWST_PLAIN && hasSuffix(mod, ":Q"):
				// Fine.
			case state == SWST_BACKT:
				// Don't check anything here, to avoid false positives for tool names.
			case (state == SWST_SQUOT || state == SWST_DQUOT) && matches(varname, `^(?:.*DIR|.*FILE|.*PATH|.*_VAR|PREFIX|.*BASE|PKGNAME)$`):
				// This is ok if we don't allow these variables to have embedded [\$\\\"\'\`].
			case state == SWST_DQUOT && hasSuffix(mod, ":Q"):
				line.warnf("Please don't use the :Q operator in double quotes.")
				line.explain(
					"Either remove the :Q or the double quotes. In most cases, it is more",
					"appropriate to remove the double quotes.")
			}

			if varname != "@" {
				vucstate := VUC_SHW_UNKNOWN
				switch state {
				case SWST_PLAIN:
					vucstate = VUC_SHW_PLAIN
				case SWST_DQUOT:
					vucstate = VUC_SHW_DQUOT
				case SWST_SQUOT:
					vucstate = VUC_SHW_SQUOT
				case SWST_BACKT:
					vucstate = VUC_SHW_BACKT
				}
				vuc := &VarUseContext{VUC_TIME_UNKNOWN, shellcommandContextType, vucstate, VUC_EXT_WORDPART}
				checklineMkVaruse(line, varname, mod, vuc)
			}

		// The syntax of the variable modifiers can get quite
		// hairy. In lack of motivation, we just skip anything
		// complicated, hoping that at least the braces are balanced.
		case replacestart(&rest, &m, `^\$\{`):
			braces := 1
		skip:
			for rest != "" && braces > 0 {
				switch {
				case replacestart(&rest, &m, `^\}`):
					braces--
				case replacestart(&rest, &m, `^\{`):
					braces++
				case replacestart(&rest, &m, `^[^{}]+`):
				// skip
				default:
					break skip
				}
			}

		case state == SWST_PLAIN:
			switch {
			case replacestart(&rest, &m, `^[!#\%&\(\)*+,\-.\/0-9:;<=>?\@A-Z\[\]^_a-z{|}~]+`),
				replacestart(&rest, &m, `^\\(?:[ !"#'\(\)*;?\\^{|}]|\$\$)`):
			case replacestart(&rest, &m, `^'`):
				state = SWST_SQUOT
			case replacestart(&rest, &m, `^"`):
				state = SWST_DQUOT
			case replacestart(&rest, &m, "^`"):
				state = SWST_BACKT
			case replacestart(&rest, &m, `^\$\$([0-9A-Z_a-z]+|\#)`),
				replacestart(&rest, &m, `^\$\$\{([0-9A-Z_a-z]+|\#)\}`),
				replacestart(&rest, &m, `^\$\$(\$)\$`):
				shvarname := m[1]
				if G.opts.optWarnQuoting && checkQuoting {
					line.warnf("Unquoted shell variable %q.", shvarname)
					line.explain(
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
			case replacestart(&rest, &m, `^\$\@`):
				line.warnf("Please use %q instead of %q.", "${.TARGET}", "$@")
				line.explain(
					"It is more readable and prevents confusion with the shell variable of",
					"the same name.")

			case replacestart(&rest, &m, `^\$\$@`):
				line.warnf("The $@ shell variable should only be used in double quotes.")

			case replacestart(&rest, &m, `^\$\$\?`):
				line.warnf("The $? shell variable is often not available in \"set -e\" mode.")

			case replacestart(&rest, &m, `^\$\$\(`):
				line.warnf("Invoking subshells via $(...) is not portable enough.")
				line.explain(
					"The Solaris /bin/sh does not know this way to execute a command in a",
					"subshell. Please use backticks (`...`) as a replacement.")

			default:
				break outer
			}

		case state == SWST_SQUOT:
			switch {
			case replacestart(&rest, &m, `^'`):
				state = SWST_PLAIN
			case replacestart(&rest, &m, `^[^\$\']+`):
				// just skip
			case replacestart(&rest, &m, `^\$\$`):
				// just skip
			default:
				break outer
			}

		case state == SWST_DQUOT:
			switch {
			case replacestart(&rest, &m, `^"`):
				state = SWST_PLAIN
			case replacestart(&rest, &m, "^`"):
				state = SWST_DQUOT_BACKT
			case replacestart(&rest, &m, "^[^$\"\\\\`]+"):
				// just skip
			case replacestart(&rest, &m, "^\\\\(?:[\\\\\"`]|\\$\\$)"):
				// just skip
			case replacestart(&rest, &m, `^\$\$\{([0-9A-Za-z_]+)\}`),
				replacestart(&rest, &m, `^\$\$([0-9A-Z_a-z]+|[!#?\@]|\$\$)`):
				shvarname := m[1]
				_ = G.opts.optDebugShell && line.debugf("checklineMkShellword: found double-quoted variable %q.", shvarname)
			case replacestart(&rest, &m, `^\$\$`):
				line.warnf("Unquoted $ or strange shell variable found.")
			case replacestart(&rest, &m, `^\\(.)`):
				char := m[1]
				line.warnf("Please use \"%s\" instead of \"%s\".", "\\\\"+char, "\\"+char)
				line.explain(
					"Although the current code may work, it is not good style to rely on",
					"the shell passing this escape sequence exactly as is, and not",
					"discarding the backslash. Alternatively you can use single quotes",
					"instead of double quotes.")
			default:
				break outer
			}
		}
	}

	if strings.TrimSpace(rest) != "" {
		line.errorf("Internal pkglint error: checklineMkShellword state=%s, rest=%q, shellword=%q", state, rest, shellword)
	}
}

type ShelltextContext struct {
	line      *Line
	state     ShellCommandState
	shellword string
}

func (msline *MkShellLine) checklineMkShelltext(shelltext string) {
	line := msline.line
	defer tracecall("checklineMkShelltext", shelltext)()

	if contains(shelltext, "${SED}") || contains(shelltext, "${MV}") {
		line.notef("Please use the SUBST framework instead of ${SED} and ${MV}.")
		line.explain(
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
		line.notef("You don't need to use \"-\" before %q.", cmd)
	}

	rest := shelltext

	setE := false
	var m []string
	if replacestart(&rest, &m, `^\s*([-@]*)(\$\{_PKG_SILENT\}\$\{_PKG_DEBUG\}|\$\{RUN\}|)`) {
		hidden, macro := m[1], m[2]
		msline.checkLineStart(hidden, macro, rest, &setE)
	}

	state := SCST_START
	for replacestart(&rest, &m, reShellword) {
		shellword := m[1]
		st := &ShelltextContext{line, state, shellword}

		_ = G.opts.optDebugShell && line.debugf("checklineMkShelltext state=%v shellword=%q", state, shellword)

		{
			quotingNecessary := state != SCST_CASE &&
				state != SCST_FOR_CONT &&
				state != SCST_SET_CONT &&
				!(state == SCST_START && matches(shellword, reShVarassign))
			msline.checklineMkShellword(shellword, quotingNecessary)
		}

		st.checkCommandStart()
		st.checkConditionalCd()
		if state != SCST_PAX_S && state != SCST_SED_E && state != SCST_CASE_LABEL {
			checklineMkAbsolutePathname(line, shellword)
		}
		st.checkAutoMkdirs()
		st.checkInstallMulti()
		st.checkPaxPe()
		st.checkQuoteSubstitution()
		st.checkEchoN()
		st.checkPipeExitcode()
		st.checkSetE(setE)

		if state == SCST_SET && matches(shellword, `^-.*e`) || state == SCST_START && shellword == "${RUN}" {
			setE = true
		}

		state = nextState(line, state, shellword)
	}

	if !matches(rest, `^\s*$`) {
		line.errorf("Internal pkglint error: checklineMkShelltext state=%s rest=%q shellword=%q", state, rest, shelltext)
	}

}

func (msline *MkShellLine) checkLineStart(hidden, macro, rest string, eflag *bool) {
	line := msline.line

	switch {
	case !contains(hidden, "@"):
		// Nothing is hidden at all.

	case hasPrefix(G.mkContext.target, "show-") || hasSuffix(G.mkContext.target, "-message"):
		// In these targets commands may be hidden.

	case hasPrefix(rest, "#"):
		// Shell comments may be hidden, since they cannot have side effects.

	default:
		if m, cmd := match1(rest, reShellword); m {
			switch cmd {
			case "${DELAYED_ERROR_MSG}", "${DELAYED_WARNING_MSG}",
				"${DO_NADA}",
				"${ECHO}", "${ECHO_MSG}", "${ECHO_N}", "${ERROR_CAT}", "${ERROR_MSG}",
				"${FAIL_MSG}",
				"${PHASE_MSG}", "${PRINTF}",
				"${SHCOMMENT}", "${STEP_MSG}",
				"${WARNING_CAT}", "${WARNING_MSG}":
			default:
				line.warnf("The shell command %q should not be hidden.", cmd)
				line.explain(
					"Hidden shell commands do not appear on the terminal or in the log file",
					"when they are executed. When they fail, the error message cannot be",
					"assigned to the command, which is very difficult to debug.")
			}
		}
	}

	if contains(hidden, "-") {
		line.warnf("The use of a leading \"-\" to suppress errors is deprecated.")
		line.explain(
			"If you really want to ignore any errors from this command (including",
			"all errors you never thought of), append \"|| ${TRUE}\" to the",
			"command.")
	}

	if macro == "${RUN}" {
		*eflag = true
	}
}

func (ctx *ShelltextContext) checkCommandStart() {
	line, state, shellword := ctx.line, ctx.state, ctx.shellword

	if state != SCST_START && state == SCST_COND {
		return
	}

	switch {
	case shellword == "${RUN}":
		// Just skip this one.

	case isForbiddenShellCommand(shellword):
		line.errorf("%q must not be used in Makefiles.", shellword)
		line.explain(
			"This command must appear in INSTALL scripts, not in the package",
			"Makefile, so that the package also works if it is installed as a binary",
			"package via pkg_add.")

	case G.globalData.tools[shellword]:
		if !G.mkContext.tools[shellword] && !G.mkContext.tools["g"+shellword] {
			line.warnf("The %q tool is used but not added to USE_TOOLS.", shellword)
		}

		if G.globalData.varRequiredTools[shellword] {
			line.warnf("Please use ${%s} instead of %q.", G.globalData.vartools[shellword], shellword)
		}

		checklineMkShellcmdUse(line, shellword)

	case matches(shellword, `^(?:\(|\)|:|;|;;|&&|\|\||\{|\}|break|case|cd|continue|do|done|elif|else|esac|eval|exec|exit|export|fi|for|if|read|set|shift|then|umask|unset|while)$`):
		// Shell builtins are fine.

	case matches(shellword, `^[\w_]+=.*$`):
		// Variable assignment

	case hasPrefix(shellword, "./"):
		// All commands from the current directory are fine.

	case hasPrefix(shellword, "#"):
		semicolon := contains(shellword, ";")
		multiline := contains(line.lines, "--")

		if semicolon {
			line.warnf("A shell comment should not contain semicolons.")
		}
		if multiline {
			line.warnf("A shell comment does not stop at the end of line.")
		}

		if semicolon || multiline {
			line.explain(
				"When you split a shell command into multiple lines that are continued",
				"with a backslash, they will nevertheless be converted to a single line",
				"before the shell sees them. That means that even if it _looks_ like that",
				"the comment only spans one line in the Makefile, in fact it spans until",
				"the end of the whole shell command. To insert a comment into shell code,",
				"you can pass it as an argument to the ${SHCOMMENT} macro, which expands",
				"to a command doing nothing. Note that any special characters are",
				"nevertheless interpreted by the shell.")
		}

	default:
		if m, vartool := match1(shellword, `^\$\{([\w_]+)\}$`); m {
			plainTool := G.globalData.varnameToToolname[vartool]
			vartype := G.globalData.getVartypes()[vartool]
			switch {
			case plainTool != "" && !G.mkContext.tools[plainTool]:
				line.warnf("The %q tool is used but not added to USE_TOOLS.", plainTool)
			case vartype != nil && vartype.basicType == "ShellCommand":
				checklineMkShellcmdUse(line, shellword)
			case G.pkgContext.vardef[vartool] != nil:
				// This command has been explicitly defined in the package; assume it to be valid.
			default:
				if G.opts.optWarnExtra {
					line.warnf("Unknown shell command %q.", shellword)
					line.explain(
						"If you want your package to be portable to all platforms that pkgsrc",
						"supports, you should only use shell commands that are covered by the",
						"tools framework.")
				}
				checklineMkShellcmdUse(line, shellword)
			}
		}
	}
}

func (ctx *ShelltextContext) checkConditionalCd() {
	line, state, shellword := ctx.line, ctx.state, ctx.shellword

	if state == SCST_COND && shellword == "cd" {
		line.errorf("The Solaris /bin/sh cannot handle \"cd\" inside conditionals.")
		line.explain(
			"When the Solaris shell is in \"set -e\" mode and \"cd\" fails, the",
			"shell will exit, no matter if it is protected by an \"if\" or the",
			"\"||\" operator.")
	}
}

func (ctx *ShelltextContext) checkAutoMkdirs() {
	line, state, shellword := ctx.line, ctx.state, ctx.shellword

	if (state == SCST_INSTALL_D || state == SCST_MKDIR) && matches(shellword, `^(?:\$\{DESTDIR\})?\$\{PREFIX(?:|:Q)\}/`) {
		line.warnf("Please use AUTO_MKDIRS instead of %q.",
			ifelseStr(state == SCST_MKDIR, "${MKDIR}", "${INSTALL} -d"))
		line.explain(
			"Setting AUTO_MKDIRS=yes automatically creates all directories that are",
			"mentioned in the PLIST. If you need additional directories, specify",
			"them in INSTALLATION_DIRS, which is a list of directories relative to",
			"${PREFIX}.")
	}

	if (state == SCST_INSTALL_DIR || state == SCST_INSTALL_DIR2) && !matches(shellword, reMkShellvaruse) {
		if m, dirname := match1(shellword, `^(?:\$\{DESTDIR\})?\$\{PREFIX(?:|:Q)\}/(.*)`); m {
			line.notef("You can use AUTO_MKDIRS=yes or \"INSTALLATION_DIRS+= %s\" instead of this command.", dirname)
			line.explain(
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
	line, state, shellword := ctx.line, ctx.state, ctx.shellword

	if state == SCST_INSTALL_DIR2 && hasPrefix(shellword, "$") {
		line.warnf("The INSTALL_*_DIR commands can only handle one directory at a time.")
		line.explain(
			"Many implementations of install(1) can handle more, but pkgsrc aims at",
			"maximum portability.")
	}
}

func (ctx *ShelltextContext) checkPaxPe() {
	line, state, shellword := ctx.line, ctx.state, ctx.shellword

	if state == SCST_PAX && shellword == "-pe" {
		line.warnf("Please use the -pp option to pax(1) instead of -pe.")
		line.explain(
			"The -pe option tells pax to preserve the ownership of the files, which",
			"means that the installed files will belong to the user that has built",
			"the package.")
	}
}

func (ctx *ShelltextContext) checkQuoteSubstitution() {
	line, state, shellword := ctx.line, ctx.state, ctx.shellword

	if state == SCST_PAX_S || state == SCST_SED_E {
		if false && !matches(shellword, `"^[\"\'].*[\"\']$`) {
			line.warnf("Substitution commands like %q should always be quoted.", shellword)
			line.explain(
				"Usually these substitution commands contain characters like '*' or",
				"other shell metacharacters that might lead to lookup of matching",
				"filenames and then expand to more than one word.")
		}
	}
}

func (ctx *ShelltextContext) checkEchoN() {
	line, state, shellword := ctx.line, ctx.state, ctx.shellword

	if state == SCST_ECHO && shellword == "-n" {
		line.warnf("Please use ${ECHO_N} instead of \"echo -n\".")
	}
}

func (ctx *ShelltextContext) checkPipeExitcode() {
	line, state, shellword := ctx.line, ctx.state, ctx.shellword

	if G.opts.optWarnExtra && state != SCST_CASE_LABEL_CONT && shellword == "|" {
		line.warnf("The exitcode of the left-hand-side command of the pipe operator is ignored.")
		line.explain(
			"If you need to detect the failure of the left-hand-side command, use",
			"temporary files to save the output of the command.")
	}
}

func (ctx *ShelltextContext) checkSetE(eflag bool) {
	line, state, shellword := ctx.line, ctx.state, ctx.shellword

	if G.opts.optWarnExtra && shellword == ";" && state != SCST_COND_CONT && state != SCST_FOR_CONT && !eflag {
		line.warnf("Please switch to \"set -e\" mode before using a semicolon to separate commands.")
		line.explain(
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

func isForbiddenShellCommand(cmd string) bool {
	switch path.Base(cmd) {
	case "ktrace", "mktexlsr", "strace", "texconfig", "truss":
		return true
	}
	return false
}

// Some shell commands should not be used in the install phase.
func (msline *MkShellLine) checkCommandUse(shellcmd string) {
	line := msline.line

	if G.mkContext == nil || !matches(G.mkContext.target, `^(?:pre|do|post)-install$`) {
		return
	}

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
		line.warnf("The shell command %q should not be used in the install phase.", shellcmd)
		line.explain(
			"In the install phase, the only thing that should be done is to install",
			"the prepared files to their final location. The file's contents should",
			"not be changed anymore.")

	case "cp", "${CP}":
		line.warnf("${CP} should not be used to install files.")
		line.explain(
			"The ${CP} command is highly platform dependent and cannot overwrite",
			"files that don't have write permission. Please use ${PAX} instead.",
			"",
			"For example, instead of",
			"\t${CP} -R ${WRKSRC}/* ${PREFIX}/foodir",
			"you should use",
			"\tcd ${WRKSRC} && ${PAX} -wr * ${PREFIX}/foodir")
	}
}

func nextState(line *Line, state ShellCommandState, shellword string) ShellCommandState {
	switch {
	case shellword == ";;":
		return SCST_CASE_LABEL
	case state == SCST_CASE_LABEL_CONT && shellword == "|":
		return SCST_CASE_LABEL
	case matches(shellword, `^[;&\|]+$`):
		return SCST_START
	case state == SCST_START:
		switch shellword {
		case "${INSTALL}":
			return SCST_INSTALL
		case "${MKDIR}":
			return SCST_MKDIR
		case "${PAX}":
			return SCST_PAX
		case "${SED}":
			return SCST_SED
		case "${ECHO}", "echo":
			return SCST_ECHO
		case "${RUN}", "then", "else", "do", "(":
			return SCST_START
		case "set":
			return SCST_SET
		case "if", "elif", "while":
			return SCST_COND
		case "case":
			return SCST_CASE
		case "for":
			return SCST_FOR
		default:
			switch {
			case matches(shellword, `^\$\{INSTALL_[A-Z]+_DIR\}$`):
				return SCST_INSTALL_DIR
			case matches(shellword, reShVarassign):
				return SCST_START
			default:
				return SCST_CONT
			}
		}
	case state == SCST_MKDIR:
		return SCST_MKDIR
	case state == SCST_INSTALL && shellword == "-d":
		return SCST_INSTALL_D
	case state == SCST_INSTALL, state == SCST_INSTALL_D:
		if matches(shellword, `^-[ogm]$`) {
			return SCST_CONT // XXX: why not state?
		}
		return state
	case state == SCST_INSTALL_DIR && hasPrefix(shellword, "-"):
		return SCST_CONT
	case state == SCST_INSTALL_DIR && hasPrefix(shellword, "$"):
		return SCST_INSTALL_DIR2
	case state == SCST_INSTALL_DIR || state == SCST_INSTALL_DIR2:
		return state
	case state == SCST_PAX && shellword == "-s":
		return SCST_PAX_S
	case state == SCST_PAX && matches(shellword, `^-`):
		return SCST_PAX
	case state == SCST_PAX:
		return SCST_CONT
	case state == SCST_PAX_S:
		return SCST_PAX
	case state == SCST_SED && shellword == "-e":
		return SCST_SED_E
	case state == SCST_SED && matches(shellword, `^-`):
		return SCST_SED
	case state == SCST_SED:
		return SCST_CONT
	case state == SCST_SED_E:
		return SCST_SED
	case state == SCST_SET:
		return SCST_SET_CONT
	case state == SCST_SET_CONT:
		return SCST_SET_CONT
	case state == SCST_CASE:
		return SCST_CASE_IN
	case state == SCST_CASE_IN && shellword == "in":
		return SCST_CASE_LABEL
	case state == SCST_CASE_LABEL && shellword == "esac":
		return SCST_CONT
	case state == SCST_CASE_LABEL:
		return SCST_CASE_LABEL_CONT
	case state == SCST_CASE_LABEL_CONT && shellword == ")":
		return SCST_START
	case state == SCST_CONT:
		return SCST_CONT
	case state == SCST_COND:
		return SCST_COND_CONT
	case state == SCST_COND_CONT:
		return SCST_COND_CONT
	case state == SCST_FOR:
		return SCST_FOR_IN
	case state == SCST_FOR_IN && shellword == "in":
		return SCST_FOR_CONT
	case state == SCST_FOR_CONT:
		return SCST_FOR_CONT
	case state == SCST_ECHO:
		return SCST_CONT
	default:
		line.errorf("Internal pkglint error: shellword.nextState state=%s shellword=%q", state, shellword)
		return SCST_START
	}
}

func splitIntoShellwords(line *Line, text string) ([]string, string) {
	words := make([]string, 0)

	rest := text
	var m []string
	for replacestart(&rest, &m, reShellword) {
		words = append(words, m[1])
	}

	rest = strings.TrimLeftFunc(rest, unicode.IsSpace)
	if rest != "" && rest != "\\" {
		line.errorf("Internal pkglint error: splitIntoShellwords rest=%q text=%q", rest, text)
	}
	return words, rest
}
