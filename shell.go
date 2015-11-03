package main

// Parsing and checking shell commands embedded in Makefiles

import (
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
		SWST_PLAIN ShellwordState = "plain"
		SWST_SQUOT ShellwordState = "squot"
		SWST_DQUOT ShellwordState = "dquot"
		SWST_DQUOT_BACKT ShellwordState = "dquot+backt"
		SWST_BACKT ShellwordState="backt"
	)

	rest := shellword
	state := SWST_PLAIN
	for rest != "" {
		_ = GlobalVars.opts.optDebugShell && line.logDebugF("shell state %s: %q", state, rest)

		var m []string
		switch {
		// When parsing inside backticks, it is more
		// reasonable to check the whole shell command
		// recursively, instead of splitting off the first
		// make(1) variable (see the elsif below).
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
				notImplemented()
			}
			notImplemented()
			_ = stripped // XXX
/*-
				if ($rest =~ s/^\`//) {
					$state = ($state == SWST_BACKT) ? SWST_PLAIN : SWST_DQUOT;
					goto end_of_backquotes;
				} elsif ($rest =~ s/^\\([\\\`\$])//) {
					$stripped .= $1;
				} elsif ($rest =~ s/^(\\)//) {
					$line->log_warning("Backslashes should be doubled inside backticks.");
					$stripped .= $1;
				} elsif ($state == SWST_DQUOT_BACKT && $rest =~ s/^"//) {
					$line->log_warning("Double quotes inside backticks inside double quotes are error prone.");
					$line->explain_warning(
"According to the SUSv3, they produce undefined results.",
"",
"See the paragraph starting \"Within the backquoted ...\" in",
"http://www.opengroup.org/onlinepubs/009695399/utilities/xcu_chap02.html");
				} elsif ($rest =~ s/^([^\\\`]+)//) {
					$stripped .= $1;
				} else {
					assert(false, "rest=$rest");
				}

			}
			$line->log_error("Unfinished backquotes: rest=$rest");

		end_of_backquotes:
			# Check the resulting command.
			checkline_mk_shelltext($line, $stripped);
-*/
		case replacestart(&rest, &m, `^\$\{(${regex_varname}|[\@])(:[^\{]+)?\}`),
		replacestart(&rest, &m, `^\$\((${regex_varname}|[\@])(:[^\)]+)?\)`),
		replacestart(&rest, &m, `^\$([\w\@])()`):
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

			CONT_HERE
		}
	}
}
