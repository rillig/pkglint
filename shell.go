package main

// Parsing and checking shell commands embedded in Makefiles

import (
	"fmt"
	"path"
	"strings"
)

func checklineMkShellword(line *Line, word string, checkQuoting bool) {
	(&MkShellLine{line}).checklineMkShellword(word, checkQuoting)
}
func checklineMkShellcmdUse(line *Line, shellcmd string) {
	(&MkShellLine{line}).checkCommandUse(shellcmd)
}

type MkShellLine struct {
	line *Line
}

func (msline *MkShellLine) checklineMkShellword(shellword string, checkQuoting bool) {
	line := msline.line
	_ = GlobalVars.opts.optDebugTrace && line.logDebug("checklineMkShellword(%q, %q)", shellword, checkQuoting)

	if shellword == "" {
		return
	}

	shellcommandContextType := &Type{LK_NONE, "ShellCommand", nil, []AclEntry{{"*", "adsu"}}, NOT_GUESSED}
	shellwordVuc := &VarUseContext{VUC_TIME_UNKNOWN, shellcommandContextType, VUC_SHW_PLAIN, VUC_EXT_WORD}

	if m, varname, mod := match2(shellword, `^\$\{(${regex_varname})(:[^{}]+)?\}$`); m {
		checklineMkVaruse(line, varname, mod, shellwordVuc)
		return
	}

	if match(shellword, `\$\{PREFIX\}/man(?:$|/)`) != nil {
		line.logWarning("Please use ${PKGMANDIR} instead of \"man\".")
	}
	if strings.Contains(shellword, "etc/rc.d") {
		line.logWarning("Please use the RCD_SCRIPTS mechanism to install rc.d scripts automatically to ${RCD_SCRIPTS_EXAMPLEDIR}.")
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
		_ = GlobalVars.opts.optDebugShell && line.logDebug("shell state %s: %q", state, rest)

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
				if replacestart(&rest, &m, "^`") {
					if state == SWST_BACKT {
						state = SWST_PLAIN
					} else {
						state = SWST_DQUOT
					}
					goto endOfBackticks
				}
				if replacestart(&rest, &m, "^\\\\([\\\\`$])") {
					stripped += m[1]
				} else if replacestart(&rest, &m, `^(\\)`) {
					line.logWarning("Backslashes should be doubled inside backticks.")
					stripped += m[1]
				} else if state == SWST_DQUOT_BACKT && replacestart(&rest, &m, `^"`) {
					line.logWarning("Double quotes inside backticks inside double quotes are error prone.")
					line.explainWarning(
						"According to the SUSv3, they produce undefined results.",
						"",
						"See the paragraph starting \"Within the backquoted ...\" in",
						"http://www.opengroup.org/onlinepubs/009695399/utilities/xcu_chap02.html")
				} else if replacestart(&rest, &m, "^([^\\\\`]+)") {
					stripped += m[1]
				} else {
					panic(fmt.Sprintf("rest=%v", rest))
				}
			}
			line.logError("Unfinished backquotes: rest=%v", rest)

		endOfBackticks:
			msline.checklineMkShelltext(stripped)

		// Make(1) variables have the same syntax, no matter in which state we are currently.
		case replacestart(&rest, &m, `^\$\{(`+reVarname+`|@)(:[^\{]+)?\}`),
			replacestart(&rest, &m, `^\$\((`+reVarname+`|@])(:[^\)]+)?\)`),
			replacestart(&rest, &m, `^\$([\w@])()`):
			varname, mod := m[1], m[2]

			if varname == "@" {
				line.logWarning("Please use \"${.TARGET}\" instead of \"$@\".")
				line.explainWarning(
					"The variable $@ can easily be confused with the shell variable of the",
					"same name, which has a completely different meaning.")
				varname = ".TARGET"
			}

			switch {
			case state == SWST_PLAIN && strings.HasSuffix(mod, ":Q"):
				// Fine.
			case state == SWST_BACKT:
				// Don't check anything here, to avoid false positives for tool names.
			case (state == SWST_SQUOT || state == SWST_DQUOT) && match(varname, `^(?:.*DIR|.*FILE|.*PATH|.*_VAR|PREFIX|.*BASE|PKGNAME)$`) != nil:
				// This is ok if we don't allow these variables to have embedded [\$\\\"\'\`].
			case state == SWST_DQUOT && strings.HasSuffix(mod, ":Q"):
				line.logWarning("Please don't use the :Q operator in double quotes.")
				line.explainWarning(
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
				if GlobalVars.opts.optWarnQuoting && checkQuoting {
					line.logWarning("Unquoted shell variable %q.", shvarname)
					line.explainWarning(
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
				line.logWarning("Please use %q instead of %q.", "${.TARGET}", "$@")
				line.explainWarning(
					"It is more readable and prevents confusion with the shell variable of",
					"the same name.")

			case replacestart(&rest, &m, `^\$\$@`):
				line.logWarning("The $@ shell variable should only be used in double quotes.")

			case replacestart(&rest, &m, `^\$\$\?`):
				line.logWarning("The $? shell variable is often not available in \"set -e\" mode.")

			case replacestart(&rest, &m, `^\$\$\(`):
				line.logWarning("Invoking subshells via $(...) is not portable enough.")
				line.explainWarning(
					"The Solaris /bin/sh does not know this way to execute a command in a",
					"subshell. Please use backticks (`...`) as a replacement.")

			default:
				break outer
			}

		case state == SWST_SQUOT:
			if replacestart(&rest, &m, `^'`) {
				state = SWST_PLAIN
			} else if replacestart(&rest, &m, `^[^\$\']+`) {
				// just skip
			} else if replacestart(&rest, &m, `^\$\$`) {
				// just skip
			} else {
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
				_ = GlobalVars.opts.optDebugShell && line.logDebug("checklineMkShellword: found double-quoted variable %q.", shvarname)
			case replacestart(&rest, &m, `^\$\$`):
				line.logWarning("Unquoted $ or strange shell variable found.")
			case replacestart(&rest, &m, `^\\(.)`):
				char := m[1]
				line.logWarning("Please use %q instead of %q.", "\\\\"+char, "\\"+char)
				line.explainWarning(
					"Although the current code may work, it is not good style to rely on",
					"the shell passing \"\\${char}\" exactly as is, and not discarding the",
					"backslash. Alternatively you can use single quotes instead of double",
					"quotes.")
			default:
				break outer
			}
		}
	}

	if match(rest, `^\s*$`) == nil {
		line.logError("Internal pkglint error: %q: rest=%q", state, rest)
	}
}

func (msline *MkShellLine) checklineMkShelltext(shelltext string) {
	line := msline.line
	_ = GlobalVars.opts.optDebugTrace && line.logDebug("checklineMkShelltext: %v", shelltext)

	type State string
	const (
		S_START           State = "start"
		S_CONT            State = "continuation"
		S_INSTALL         State = "install"
		S_INSTALL_D       State = "install -d"
		S_MKDIR           State = "mkdir"
		S_PAX             State = "pax"
		S_PAX_S           State = "pax -s"
		S_SED             State = "sed"
		S_SED_E           State = "sed -e"
		S_SET             State = "set"
		S_SET_CONT        State = "set-continuation"
		S_COND            State = "cond"
		S_COND_CONT       State = "cond-continuation"
		S_CASE            State = "case"
		S_CASE_IN         State = "case in"
		S_CASE_LABEL      State = "case label"
		S_CASE_LABEL_CONT State = "case-label-continuation"
		S_CASE_PAREN      State = "case-paren"
		S_FOR             State = "for"
		S_FOR_IN          State = "for-in"
		S_FOR_CONT        State = "for-continuation"
		S_ECHO            State = "echo"
		S_INSTALL_DIR     State = "install-dir"
		S_INSTALL_DIR2    State = "install-dir2"
	)

	if strings.Contains(shelltext, "${SED}") || strings.Contains(shelltext, "${MV}") {
		line.logNote("Please use the SUBST framework instead of ${SED} and ${MV}.")
		line.explainNote(
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

	if m := match(shelltext, `^@*-(.*MKDIR|INSTALL.*-d|INSTALL_.*_DIR).*)`); m != nil {
		line.logNote("You don't need to use \"-\" before %v.", m[1])
	}

	rest := shelltext

	eflagMode := false
	var m []string
	if replacestart(&rest, &m, `^\s*([-@]*)(\$\{_PKG_SILENT\}\$\{_PKG_DEBUG\}|\$\{RUN\}|)`) {
		hidden, macro := m[1], m[2]
		msline.checkLineStart(hidden, macro, rest, &eflagMode)
	}

	state := S_START
	for replacestart(&rest, &m, reShellword) {
		shellword := m[1]

		_ = GlobalVars.opts.optDebugShell && line.logDebug("checklineMkShelltext state=%v shellword=%v", state, shellword)

		msline.checklineMkShellword(shellword, !(state == S_CASE ||
			state == S_FOR_CONT ||
			state == S_SET_CONT ||
			(state == S_START && match(shellword, reShVarassign) != nil)))

		if state == S_START || state == S_COND {
			msline.checkCommandStart(shellword)
		}

		if state == S_COND && shellword == "cd" {
			line.logError("The Solaris /bin/sh cannot handle \"cd\" inside conditionals.")
			line.explainError(
				"When the Solaris shell is in \"set -e\" mode and \"cd\" fails, the",
				"shell will exit, no matter if it is protected by an \"if\" or the",
				"\"||\" operator.")
		}

		if state != S_PAX_S && state != S_SED_E && state != S_CASE_LABEL {
			checklineMkAbsolutePathname(line, shellword)
		}

		if (state == S_INSTALL_D || state == S_MKDIR) && match(shellword, `^(?:\$\{DESTDIR\})?\$\{PREFIX(?:|:Q)\}/`) != nil {
			line.logWarning("Please use AUTO_MKDIRS instead of %q.",
				ifelseStr(state == S_MKDIR, "${MKDIR}", "${INSTALL} -d"))
			line.explainWarning(
				"Setting AUTO_MKDIRS=yes automatically creates all directories that are",
				"mentioned in the PLIST. If you need additional directories, specify",
				"them in INSTALLATION_DIRS, which is a list of directories relative to",
				"${PREFIX}.")
		}

		if (state == S_INSTALL_DIR || state == S_INSTALL_DIR2) && match(shellword, reMkShellvaruse) == nil {
			if m, dirname := match1(shellword, `^(?:\$\{DESTDIR\})?\$\{PREFIX(?:|:Q)\}/(.*)`); m {
				line.logNote("You can use AUTO_MKDIRS=yes or \"INSTALLATION_DIRS+= %s\" instead of this command.", dirname)
				line.explainNote(
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
		
		if state == S_INSTALL_DIR2 && strings.HasPrefix(shellword, "$") {
			line.logWarning("The INSTALL_*_DIR commands can only handle one directory at a time.")
			line.explainWarning(
"Many implementations of install(1) can handle more, but pkgsrc aims at",
"maximum portability.")
					}
		
		
		//AAAAAASDFDHFJDFSDGSDGSDGSD
		
	}
}

func (msline *MkShellLine) checkLineStart(hidden, macro, rest string, eflag *bool) {
	line := msline.line

	switch {
	case !strings.Contains(hidden, "@"):
		// Nothing is hidden at all.

	case strings.HasPrefix(GlobalVars.mkContext.target, "show-") || strings.HasSuffix(GlobalVars.mkContext.target, "-message"):
		// In these targets commands may be hidden.

	case strings.HasPrefix(rest, "#"):
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
				line.logWarning("The shell command %q should not be hidden.", cmd)
				line.explainWarning(
					"Hidden shell commands do not appear on the terminal or in the log file",
					"when they are executed. When they fail, the error message cannot be",
					"assigned to the command, which is very difficult to debug.")
			}
		}
	}

	if strings.Contains(hidden, "-") {
		line.logWarning("The use of a leading \"-\" to suppress errors is deprecated.")
		line.explainWarning(
			"If you really want to ignore any errors from this command (including",
			"all errors you never thought of), append \"|| ${TRUE}\" to the",
			"command.")
	}

	if macro == "${RUN}" {
		*eflag = true
	}
}

func (msline *MkShellLine) checkCommandStart(shellword string) {
	line := msline.line

	switch {
	case shellword == "${RUN}":
		// Just skip this one.

	case msline.isForbiddenShellCommand(shellword):
		line.logError("%q must not be used in Makefiles.", shellword)
		line.explainError(
			"This command must appear in INSTALL scripts, not in the package",
			"Makefile, so that the package also works if it is installed as a binary",
			"package via pkg_add.")

	case GlobalVars.globalData.tools[shellword]:
		if !GlobalVars.mkContext.tools[shellword] && !GlobalVars.mkContext.tools["g"+shellword] {
			line.logWarning("The %q tool is used but not added to USE_TOOLS.", shellword)
		}

		if GlobalVars.globalData.varRequiredTools[shellword] {
			line.logWarning("Please use ${%s} instead of %q.", GlobalVars.globalData.vartools[shellword], shellword)
		}

		checklineMkShellcmdUse(line, shellword)

	case match(shellword, `^(?:\(|\)|:|;|;;|&&|\|\||\{|\}|break|case|cd|continue|do|done|elif|else|esac|eval|exec|exit|export|fi|for|if|read|set|shift|then|umask|unset|while)$`) != nil:
		// Shell builtins are fine.

	case match(shellword, `^[\w_]+=.*$`) != nil:
		// Variable assignment

	case match(shellword, `^\./`) != nil:
		// All commands from the current directory are fine.

	case strings.HasPrefix(shellword, "#"):
		semicolon := strings.Contains(shellword, ";")
		multiline := strings.Contains(line.lines, "--")

		if semicolon {
			line.logWarning("A shell comment should not contain semicolons.")
		}
		if multiline {
			line.logWarning("A shell comment does not stop at the end of line.")
		}

		if semicolon || multiline {
			line.explainWarning(
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
			plainTool := GlobalVars.globalData.varnameToToolname[vartool]
			vartype := GlobalVars.globalData.vartypes[vartool]
			switch {
			case plainTool != "" && GlobalVars.mkContext.tools[plainTool]:
				line.logWarning("The %q tool is used but not added to USE_TOOLS.", plainTool)
			case vartype.basicType == "ShellCommand":
				checklineMkShellcmdUse(line, shellword)
			case GlobalVars.pkgContext.vardef[vartool] != nil:
				// This command has been explicitly defined in the package; assume it to be valid.
			default:
				if GlobalVars.opts.optWarnExtra {
					line.logWarning("Unknown shell command %q.", shellword)
					line.explainWarning(
						"If you want your package to be portable to all platforms that pkgsrc",
						"supports, you should only use shell commands that are covered by the",
						"tools framework.")
				}
				checklineMkShellcmdUse(line, shellword)
			}
		}
	}
}

func (msline *MkShellLine) isForbiddenShellCommand(cmd string) bool {
	switch path.Base(cmd) {
	case "ktrace", "mktexlsr", "strace", "texconfig", "truss":
		return true
	}
	return false
}

func shellSplit(text string) []string {
	words := make([]string, 0)

	rest := text
	var m []string
	for replacestart(&rest, &m, reShellword) {
		words = append(words, m[1])
	}
	if match0(rest, `^\s*$`) {
		return words
	} else {
		return nil
	}
}

// Some shell commands should not be used in the install phase.
func (msline *MkShellLine) checkCommandUse(shellcmd string) {
	line := msline.line

	if G.mkContext == nil || !match0(G.mkContext.target, `^(?:pre|do|post)-install$`) {
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
		line.logWarning("The shell command %q should not be used in the install phase.", shellcmd)
		line.explainWarning(
			"In the install phase, the only thing that should be done is to install",
			"the prepared files to their final location. The file's contents should",
			"not be changed anymore.")

	case "cp", "${CP}":
		line.logWarning("${CP} should not be used to install files.")
		line.explainWarning(
			"The ${CP} command is highly platform dependent and cannot overwrite",
			"files that don't have write permission. Please use ${PAX} instead.",
			"",
			"For example, instead of",
			"\t${CP} -R ${WRKSRC}/* ${PREFIX}/foodir",
			"you should use",
			"\tcd ${WRKSRC} && ${PAX} -wr * ${PREFIX}/foodir")
	}
}
