package main

import (
	"strings"
)

func checklineMkVardef(line *Line, varname, op string) {
	_ = G.opts.optDebugTrace && line.logDebug("checkline_mk_vardef(%v, %v)", varname, op)

	if G.pkgContext != nil && G.pkgContext.vardef[varname] == nil {
		G.pkgContext.vardef[varname] = line
	}

	if G.mkContext.vardef[varname] == nil {
		G.mkContext.vardef[varname] = line
	}

	if !G.opts.optWarnPerm {
		return
	}

	perms := getVariablePermissions(line, varname)
	var needed string
	switch op {
	case "=":
		needed = "s"
	case "!=":
		needed = "s"
	case "?=":
		needed = "d"
	case "+=":
		needed = "a"
	case ":=":
		needed = "s"
	}

	if !strstr(perms, needed) {
		expandPermission := func(perm string) string {
			result := ""
			for _, c := range perm {
				switch c {
				case 'a':
					result += "append, "
				case 'd':
					result += "default, "
				case 'p':
					result += "preprocess, "
				case 's':
					result += "set, "
				case 'u':
					result += "runtime-use, "
				case '?':
					result += "unknown, "
				}
			}
			return strings.TrimRight(result, ", ")
		}

		line.logWarning("Permission [%s] requested for %s but only [%s] is allowed.",
			expandPermission(needed), varname, expandPermission(perms))
		line.explainWarning(
			"The available permissions are:",
			"\tappend\t\tappend something using +=",
			"\tdefault\t\tset a default value using ?=",
			"\tpreprocess\tuse a variable during preprocessing",
			"\truntime\t\tuse a variable at runtime",
			"\tset\t\tset a variable using :=, =, !=",
			"",
			"A \"?\" means that pkglint doesn't know which permissions are allowed",
			"and which are not.")
	}
}

func checklineMkVaruse(line *Line, varname string, mod string, vuc *VarUseContext) {
	_ = G.opts.optDebugTrace && line.logDebug("checklineMkVaruse(%q, %q, %q)", varname, mod, *vuc)

	vartype := getVariableType(line, varname)
	if G.opts.optWarnExtra &&
		!(vartype != nil && vartype.guessed == NOT_GUESSED) &&
		!varIsUsed(varname) &&
		!(G.mkContext != nil && G.mkContext.forVars[varname]) {
		line.logWarning("%s is used but not defined. Spelling mistake?", varname)
	}

	if G.opts.optWarnPerm {
		checklineMkVarusePerm(line, varname, vuc)
	}

	if varname == "LOCALBASE" && !G.isInternal {
		checklineMkVaruseLocalbase(line)
	}

	needsQuoting := variableNeedsQuoting(line, varname, vuc)

	if vuc.shellword == VUC_SHW_FOR {
		checklineMkVaruseFor(line, varname, vartype, needsQuoting)
	}

	if G.opts.optWarnQuoting && vuc.shellword != VUC_SHW_UNKNOWN && needsQuoting != NQ_DONT_KNOW {
		checklineMkVaruseShellword(line, varname, vartype, vuc, mod, needsQuoting)
	}

	if G.globalData.userDefinedVars[varname] != nil && !G.globalData.systemBuildDefs[varname] && !G.mkContext.buildDefs[varname] {
		line.logWarning("The user-defined variable ${varname} is used but not added to BUILD_DEFS.")
		line.explainWarning(
			"When a pkgsrc package is built, many things can be configured by the",
			"pkgsrc user in the mk.conf file. All these configurations should be",
			"recorded in the binary package, so the package can be reliably rebuilt.",
			"The BUILD_DEFS variable contains a list of all these user-settable",
			"variables, so please add your variable to it, too.")
	}
}

func checklineMkVarusePerm(line *Line, varname string, vuc *VarUseContext) {
	perms := getVariablePermissions(line, varname)

	isLoadTime := false // Will the variable be used at load time?

	// Might the variable be used indirectly at load time, for example
	// by assigning it to another variable which then gets evaluated?
	isIndirect := false

	switch {
	case vuc.vartype != nil && vuc.vartype.isGuessed():
		// Don't warn about unknown variables.

	case vuc.time == VUC_TIME_LOAD && !strings.Contains(perms, "p"):
		isLoadTime = true

	case vuc.vartype != nil && strings.Contains(vuc.vartype.union(), "p") && !strings.Contains(perms, "p"):
		isLoadTime = true
		isIndirect = true
	}

	if isLoadTime && !isIndirect {
		line.logWarning("%s should not be evaluated at load time.", varname)
		line.explainWarning(
			"Many variables, especially lists of something, get their values",
			"incrementally. Therefore it is generally unsafe to rely on their value",
			"until it is clear that it will never change again. This point is",
			"reached when the whole package Makefile is loaded and execution of the",
			"shell commands starts, in some cases earlier.",
			"",
			"Additionally, when using the \":=\" operator, each $$ is replaced",
			"with a single $, so variables that have references to shell variables",
			"or regular expressions are modified in a subtle way.")
	}

	if isLoadTime && isIndirect {
		line.logWarning("%s should not be evaluated indirectly at load time.", varname)
		line.explainWarning(
			"The variable on the left-hand side may be evaluated at load time, but",
			"the variable on the right-hand side may not. Due to this assignment, it",
			"might be used indirectly at load-time, when it is not guaranteed to be",
			"properly defined.")
	}

	if !strstr(perms, "p") && !strstr(perms, "u") {
		line.logWarning("%s must not be used in this file.")
	}
}

func checklineMkVaruseLocalbase(line *Line) {
	line.logWarning("The LOCALBASE variable should not be used by packages.")
	line.explainWarning(
		// from jlam via private mail.
		"Currently, LOCALBASE is typically used in these cases:",
		"",
		"(1) To locate a file or directory from another package.",
		"(2) To refer to own files after installation.",
		"",
		"In the first case, the example is:",
		"",
		"	STRLIST=        ${LOCALBASE}/bin/strlist",
		"	do-build:",
		"		cd ${WRKSRC} && ${STRLIST} *.str",
		"",
		"This should really be:",
		"",
		"	EVAL_PREFIX=    STRLIST_PREFIX=strlist",
		"	STRLIST=        ${STRLIST_PREFIX}/bin/strlist",
		"	do-build:",
		"		cd ${WRKSRC} && ${STRLIST} *.str",
		"",
		"In the second case, the example is:",
		"",
		"	CONFIGURE_ENV+= --with-datafiles=${LOCALBASE}/share/battalion",
		"",
		"This should really be:",
		"",
		"	CONFIGURE_ENV+= --with-datafiles=${PREFIX}/share/battalion")
}

func checklineMkVaruseFor(line *Line, varname string, vartype *Type, needsQuoting NeedsQuoting) {
	switch {
	case vartype == nil:
		// Cannot check anything here.

	case vartype.kindOfList == LK_INTERNAL:
		// Fine

	case needsQuoting == NQ_DOESNT_MATTER || needsQuoting == NQ_NO:
		// Fine, this variable is not supposed to contain special characters.

	default:
		line.logWarning("The variable %s should not be used in .for loops.", varname)
		line.explainWarning(
			"The .for loop splits its argument at sequences of white-space, as",
			"opposed to all other places in make(1), which act like the shell.",
			"Therefore only variables that are specifically designed to match this",
			"requirement should be used here.")
	}
}

func checklineMkVaruseShellword(line *Line, varname string, vartype *Type, vuc *VarUseContext, mod string, needsQuoting NeedsQuoting) {

	// In GNU configure scripts, a few variables need to be
	// passed through the :M* operator before they reach the
	// configure scripts.
	//
	// When doing checks outside a package, the :M* operator is needed for safety.
	needMstar := match0(varname, reGnuConfigureVolatileVars) &&
		(G.pkgContext == nil || G.pkgContext.vardef["GNU_CONFIGURE"] != nil)

	strippedMod := mod
	if m, stripped := match1(mod, `(.*?)(?::M\*)?(?::Q)?$`); m {
		strippedMod = stripped
	}
	correctMod := strippedMod + ifelseStr(needMstar, ":M*:Q", ":Q")

	if mod == ":M*:Q" && !needMstar {
		line.logNote("The :M* modifier is not needed here.")

	} else if mod != correctMod && needsQuoting == NQ_YES {
		if vuc.shellword == VUC_SHW_PLAIN {
			line.logWarning("Please use ${%s%s} instead of ${%s%s}.", varname, correctMod, varname, mod)
		} else {
			line.logWarning("Please use ${%s%s} instead of ${%s%s} and make sure the variable appears outside of any quoting characters.", varname, correctMod, varname, mod)
		}
		line.explainWarning(
			"See the pkgsrc guide, section \"quoting guideline\", for details.")
	}

	if strings.HasSuffix(mod, ":Q") {
		expl := []string{
			"Many variables in pkgsrc do not need the :Q operator, since they",
			"are not expected to contain white-space or other special characters.",
			"",
			"Another case is when a variable of type ShellWord appears in a context",
			"that expects a shell word, it does not need to have a :Q operator. Even",
			"when it is concatenated with another variable, it still stays _one_ word.",
			"",
			"Example:",
			"\tWORD1=  Have\\ fun             # 1 word",
			"\tWORD2=  \"with BSD Make\"       # 1 word, too",
			"",
			"\tdemo:",
			"\t\techo ${WORD1}${WORD2} # still 1 word",
		}

		switch needsQuoting {
		case NQ_NO:
			line.logWarning("The :Q operator should not be used for ${%s} here.", varname)
			line.explainWarning(expl...)
		case NQ_DOESNT_MATTER:
			line.logNote("The :Q operator isn't necessary for ${%s} here.", varname)
			line.explainWarning(expl...)
		}
	}
}
