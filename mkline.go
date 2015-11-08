package main

import (
	"strconv"
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

	if !contains(perms, needed) {
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

	if !contains(perms, "p") && !contains(perms, "u") {
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

func checklineMkVaruseFor(line *Line, varname string, vartype *Vartype, needsQuoting NeedsQuoting) {
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

func checklineMkVaruseShellword(line *Line, varname string, vartype *Vartype, vuc *VarUseContext, mod string, needsQuoting NeedsQuoting) {

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

	if hasSuffix(mod, ":Q") {
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

// @param op
//	The operator that is used for reading or writing to the variable.
//	One of: "=", "+=", ":=", "!=", "?=", "use", "pp-use", "".
//	For some variables (like BuildlinkDepth or BuildlinkPackages), the
//	operator influences the valid values.
// @param comment
//	In assignments, a part of the line may be commented out. If there
//	is no comment, pass C<undef>.
//
func checklineMkVartypeSimple(line *Line, varname string, basicType string, op, value, comment string, listContext, guessed bool) {

	_ = G.opts.optDebugTrace && line.logDebug("checklineMkVartypeBasic(%v, %v, %v, %v, %v, %v, %v)",
		varname, basicType, op, value, comment, listContext, guessed)

	valueNovar := value
	for {
		var m []string
		if m, valueNovar = replaceFirst(valueNovar, `\$\{([^{}]*)\}`, ""); m != nil {
			varuse := m[1]
			if !listContext && hasSuffix(varuse, ":Q") {
				line.logWarning("The :Q operator should only be used in lists and shell commands.")
			}
		} else {
			break
		}
	}

	notImplemented()
	_ = valueNovar
	// fn := basicCheck(vartype.basicType)
	// TODO: basic check(vartype)
	// fn()
}

func checklineMkVartypeEnum(line *Line, varname string, enumValues map[string]bool, enumValuesStr, op, value, comment string, listContext, guessed bool) {
	if !enumValues[value] {
		line.logWarning("%q is not valid for %s. Use one of { %s } instead.", value, varname, enumValuesStr)
	}
}

func checklineMkDecreasingOrder(line *Line, varname, value string) {
	strversions := splitOnSpace(value)
	intversions := make([]int, len(strversions))
	for i, strversion := range strversions {
		iver, err := strconv.Atoi(strversion)
		if err != nil || !(iver > 0) {
			line.logError("All values for %s must be positive integers.", varname)
			return
		}
		intversions[i] = iver
	}

	for i, ver := range intversions[1:] {
		if ver >= intversions[i-1] {
			line.logWarning("The values for %s should be in decreasing order.", varname)
			line.explainWarning(
				"If they aren't, it may be possible that needless versions of packages",
				"are installed.")
		}
	}
}

func checklineMkVarassign(line *Line, varname, op, value, comment string) {
	_ = G.opts.optDebugTrace && line.logDebug("checklineMkVarassign(%v, %v, %v)", varname, op, value)

	varbase := varnameBase(varname)
	varcanon := varnameCanon(varname)

	checklineMkVardef(line, varname, op)

	if G.opts.optWarnExtra && op == "?=" && G.pkgContext != nil && !G.pkgContext.seen_bsd_prefs_mk {
		switch varbase {
		case "BUILDLINK_PKGSRCDIR", "BUILDLINK_DEPMETHOD", "BUILDLINK_ABI_DEPENDS":
			// FIXME: What about these ones? They occur quite often.

		default:
			line.logWarning("Please include \"../../mk/bsd.prefs.mk\" before using \"?=\".")
			line.explainWarning(
				"The ?= operator is used to provide a default value to a variable. In",
				"pkgsrc, many variables can be set by the pkgsrc user in the mk.conf",
				"file. This file must be included explicitly. If a ?= operator appears",
				"before mk.conf has been included, it will not care about the user's",
				"preferences, which can result in unexpected behavior. The easiest way",
				"to include the mk.conf file is by including the bsd.prefs.mk file,",
				"which will take care of everything.")
		}
	}

	checklineMkText(line, value)
	checklineMkVartype(line, varname, op, value, comment)

	// If the variable is not used and is untyped, it may be a spelling mistake.
	if op == ":=" && varname == strings.ToLower(varname) {
		_ = G.opts.optDebugUnchecked && line.logDebug("%s might be unused unless it is an argument to a procedure file.", varname)

	} else if !varIsUsed(varname) {
		if vartypes := G.globalData.vartypes; vartypes[varname] != nil || vartypes[varcanon] != nil {
			// Ok
		} else if deprecated := deprecatedVars; deprecated[varname] != "" || deprecated[varcanon] != "" {
			// Ok
		} else {
			line.logWarning("%s is defined but not used. Spelling mistake?", varname)
		}
	}

	if match0(value, `/etc/rc\.d`) {
		line.logWarning("Please use the RCD_SCRIPTS mechanism to install rc.d scripts automatically to ${RCD_SCRIPTS_EXAMPLEDIR}.")
	}

	if hasPrefix(varname, "_") && !G.isInternal {
		line.logWarning("Variable names starting with an underscore are reserved for internal pkgsrc use.")
	}

	if varname == "PERL5_PACKLIST" && G.pkgContext.effective_pkgbase != nil {
		if m, p5pkgname := match1(*G.pkgContext.effective_pkgbase, `^p5-(.*)`); m {
			guess := "auto/" + strings.Replace(p5pkgname, "-", "/", -1) + "/.packlist"

			ucvalue, ucguess := strings.ToUpper(value), strings.ToUpper(guess)
			if ucvalue != ucguess && ucvalue != "${PERL5_SITEARCH}/"+ucguess {
				line.logWarning("Unusual value for PERL5_PACKLIST -- %q expected.", guess)
			}
		}
	}

	if varname == "CONFIGURE_ARGS" && match0(value, `=\$\{PREFIX\}/share/kde`) {
		line.logNote("Please .include \"../../meta-pkgs/kde3/kde3.mk\" instead of this line.")
		line.explainNote(
			"That file probably does many things automatically and consistently that",
			"this package also does. When using kde3.mk, you can probably also leave",
			"out some explicit dependencies.")
	}

	if varname == "EVAL_PREFIX" {
		if m, evalVarname := match1(value, `^([\w_]+)=`); m {

			// The variables mentioned in EVAL_PREFIX will later be
			// defined by find-prefix.mk. Therefore, they are marked
			// as known in the current file.
			G.mkContext.vardef[evalVarname] = line
		}
	}

	if varname == "PYTHON_VERSIONS_ACCEPTED" {
		checklineMkDecreasingOrder(line, varname, value)
	}

	if comment == "# defined" && !match0(varname, `.*(?:_MK|_COMMON)$`) {
		line.logNote("Please use \"# empty\", \"# none\" or \"yes\" instead of \"# defined\".")
		line.explainNote(
			"The value #defined says something about the state of the variable, but",
			"not what that _means_. In some cases a variable that is defined means",
			"\"yes\", in other cases it is an empty list (which is also only the",
			"state of the variable), whose meaning could be described with \"none\".",
			"It is this meaning that should be described.")
	}

	if m, pkgvarname := match1(value, `\$\{(PKGNAME|PKGVERSION)[:\}]`); m {
		if match0(varname, `^PKG_.*_REASON$`) {
			// ok
		} else if match0(varname, `^(?:DIST_SUBDIR|WRKSRC)$`) {
			line.logWarning("%s should not be used in %s, as it sometimes includes the PKGREVISION. Please use %s_NOREV instead.", pkgvarname, varname, pkgvarname)
		} else {
			_ = G.opts.optDebugMisc && line.logDebug("Use of PKGNAME in %s.", varname)
		}
	}

	if fix := deprecatedVars[varname]; fix != "" {
		line.logWarning("Definition of %s is deprecated. %s", varname, fix)
	} else if fix := deprecatedVars[varcanon]; fix != "" {
		line.logWarning("Definition of %s is deprecated. %s", varname, fix)
	}

	if hasPrefix(varname, "SITES_") {
		line.logWarning("SITES_* is deprecated. Please use SITES.* instead.")
	}

	if match0(value, `^[^=]\@comment`) {
		line.logWarning("Please don't use @comment in %s.", varname)
		line.explainWarning(
			"Here you are defining a variable containing @comment. As this value",
			"typically includes a space as the last character you probably also used",
			"quotes around the variable. This can lead to confusion when adding this",
			"variable to PLIST_SUBST, as all other variables are quoted using the :Q",
			"operator when they are appended. As it is hard to check whether a",
			"variable that is appended to PLIST_SUBST is already quoted or not, you",
			"should not have pre-quoted variables at all. To solve this, you should",
			"directly use PLIST_SUBST+= ${varname}=${value} or use any other",
			"variable for collecting the list of PLIST substitutions and later",
			"append that variable with PLIST_SUBST+= ${MY_PLIST_SUBST}.")
	}

	// Mark the variable as PLIST condition. This is later used in
	// checkfile_PLIST.
	if G.pkgContext.plistSubstCond != nil {
		if m, plistVarname := match1(value, `(.+)=.*\@comment.*`); m {
			G.pkgContext.plistSubstCond[plistVarname] = true
		}
	}

	time := VUC_TIME_RUN
	switch op {
	case ":=", "!=":
		time = VUC_TIME_LOAD
	}

	usedVars := extractUsedVariables(line, value)
	vuc := &VarUseContext{
		time,
		getVariableType(line, varname),
		VUC_SHW_UNKNOWN,
		VUC_EXTENT_UNKNOWN}
	for _, usedVar := range usedVars {
		checklineMkVaruse(line, usedVar, "", vuc)
	}
}

func checklineMkVartype(line *Line, varname, op, value, comment string) {
	if !G.opts.optWarnTypes {
		return
	}

	varbase := varnameBase(varname)
	vartype := getVariableType(line, varname)

	if op == "+=" {
		if vartype != nil {
			if !vartype.mayBeAppendedTo() {
				line.logWarning("The \"+=\" operator should only be used with lists.")
			}
		} else if !match0(varbase, `^_`) && !match0(varbase, reVarnamePlural) {
			line.logWarning("As ${varname} is modified using \"+=\", its name should indicate plural.")
		}
	}

	if vartype == nil {
		// Cannot check anything if the type is not known.
		_ = G.opts.optDebugUnchecked && line.logDebug("Unchecked variable assignment for %s.", varname)

	} else if op == "!=" {
		_ = G.opts.optDebugMisc && line.logDebug("Use of !=: %q", value)

	} else if vartype.kindOfList != LK_NONE {
		words := make([]string, 0)
		rest := ""

		if vartype.kindOfList == LK_INTERNAL {
			words = splitOnSpace(value)
		} else {
			rest = value
			for {
				m, r := replaceFirst(rest, reShellword, "")
				if m == nil {
					break
				}
				rest = r

				word := m[1]
				if match0(word, `^#`) {
					break
				}
				words = append(words, word)
			}
		}

		for _, word := range words {
			checklineMkVartypeBasic(line, varname, vartype, op, word, comment, true, vartype.isGuessed())
			if vartype.kindOfList != LK_INTERNAL {
				checklineMkShellword(line, word, true)
			}
		}

		if !match0(rest, `^\s*$`) {
			line.logError("Internal pkglint error: rest=%q", rest)
		}

	} else {
		checklineMkVartypeBasic(line, varname, vartype, op, value, comment, vartype.isConsideredList(), vartype.isGuessed())
	}
}

func checklineMkVartypeBasic(line *Line, varname string, vartype *Vartype, op, value, comment string, isList, isGuessed bool) {
	notImplemented()
}
