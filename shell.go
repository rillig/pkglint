package main

// Parsing and checking shell commands embedded in Makefiles

import (
	"fmt"
	"strings"
)

func checklineMkShellword(line *Line, shellword string, checkQuoting bool) {
	_ = GlobalVars.opts.optDebugTrace && line.logDebugF("checklineMkShellword(%q, %q)", shellword, checkQuoting)

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
		line.logWarningF("Please use ${PKGMANDIR} instead of \"man\".")
	}
	if strings.Contains(shellword, "etc/rc.d") {
		line.logWarningF("Please use the RCD_SCRIPTS mechanism to install rc.d scripts automatically to ${RCD_SCRIPTS_EXAMPLEDIR}.")
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
		_ = GlobalVars.opts.optDebugShell && line.logDebugF("shell state %s: %q", state, rest)

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
					line.logWarningF("Backslashes should be doubled inside backticks.")
					stripped += m[1]
				} else if state == SWST_DQUOT_BACKT && replacestart(&rest, &m, `^"`) {
					line.logWarningF("Double quotes inside backticks inside double quotes are error prone.")
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
			line.logErrorF("Unfinished backquotes: rest=%v", rest)

		endOfBackticks:
			checklineMkShelltext(line, stripped)

		// Make(1) variables have the same syntax, no matter in which state we are currently.
		case replacestart(&rest, &m, `^\$\{(`+reVarname+`|@)(:[^\{]+)?\}`),
			replacestart(&rest, &m, `^\$\((`+reVarname+`|@])(:[^\)]+)?\)`),
			replacestart(&rest, &m, `^\$([\w@])()`):
			varname, mod := m[1], m[2]

			if varname == "@" {
				line.logWarningF("Please use \"${.TARGET}\" instead of \"$@\".")
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
				line.logWarningF("Please don't use the :Q operator in double quotes.")
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
					line.logWarningF("Unquoted shell variable %q.", shvarname)
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
				line.logWarningF("Please use %q instead of %q.", "${.TARGET}", "$@")
				line.explainWarning(
					"It is more readable and prevents confusion with the shell variable of",
					"the same name.")

			case replacestart(&rest, &m, `^\$\$@`):
				line.logWarningF("The $@ shell variable should only be used in double quotes.")

			case replacestart(&rest, &m, `^\$\$\?`):
				line.logWarningF("The $? shell variable is often not available in \"set -e\" mode.")

			case replacestart(&rest, &m, `^\$\$\(`):
				line.logWarningF("Invoking subshells via $(...) is not portable enough.")
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
				_ = GlobalVars.opts.optDebugShell && line.logDebugF("checklineMkShellword: found double-quoted variable %q.", shvarname)
			case replacestart(&rest, &m, `^\$\$`):
				line.logWarningF("Unquoted $ or strange shell variable found.")
			case replacestart(&rest, &m, `^\\(.)`):
				char := m[1]
				line.logWarningF("Please use %q instead of %q.", "\\\\"+char, "\\"+char)
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
		line.logErrorF("Internal pkglint error: %q: rest=%q", state, rest)
	}
}
