package main

// Checks concerning single lines in Makefiles.

import (
	"fmt"
	"os"
	"strconv"
	"strings"
)

type MkLine struct {
	line *Line

	xtype uint8
	xb1   bool
	xs1   string
	xs2   string
	xs3   string
	xs4   string
	xs5   string
	xs6   string
}

func (mkline *MkLine) error1(format, arg1 string)       { mkline.line.error1(format, arg1) }
func (mkline *MkLine) warn0(format string)              { mkline.line.warn0(format) }
func (mkline *MkLine) warn1(format, arg1 string)        { mkline.line.warn1(format, arg1) }
func (mkline *MkLine) warn2(format, arg1, arg2 string)  { mkline.line.warn2(format, arg1, arg2) }
func (mkline *MkLine) note0(format string)              { mkline.line.note0(format) }
func (mkline *MkLine) note2(format, arg1, arg2 string)  { mkline.line.note2(format, arg1, arg2) }
func (mkline *MkLine) debug1(format, arg1 string)       { mkline.line.debug1(format, arg1) }
func (mkline *MkLine) debug2(format, arg1, arg2 string) { mkline.line.debug2(format, arg1, arg2) }

func NewMkLine(line *Line) (mkline *MkLine) {
	mkline = &MkLine{line: line}

	text := line.Text

	if hasPrefix(text, " ") {
		mkline.warn0("Makefile lines should not start with space characters.")
		explain3(
			"If you want this line to contain a shell program, use a tab",
			"character for indentation. Otherwise please remove the leading",
			"white-space.")
	}

	if m, varname, op, value, comment := matchVarassign(text); m {
		value = strings.Replace(value, "\\#", "#", -1)
		varparam := varnameParam(varname)

		mkline.xtype = 1
		mkline.xs1 = varname
		mkline.xs2 = varnameCanon(varname)
		mkline.xs3 = varparam
		mkline.xs4 = op
		mkline.xs5 = value
		mkline.xs6 = comment
		return
	}

	if hasPrefix(text, "\t") {
		mkline.xtype = 2
		mkline.xs1 = text[1:]
		return
	}

	if index := strings.IndexByte(text, '#'); index != -1 && strings.TrimSpace(text[:index]) == "" {
		mkline.xtype = 3
		mkline.xs6 = text[index+1:]
		return
	}

	if strings.TrimSpace(text) == "" {
		mkline.xtype = 4
		return
	}

	if m, indent, directive, args := matchMkCond(text); m {
		mkline.xtype = 5
		mkline.xs1 = indent
		mkline.xs2 = directive
		mkline.xs3 = args
		return
	}

	if m, directive, includefile := match2(text, reMkInclude); m {
		mkline.xtype = 6
		mkline.xb1 = directive == "include"
		mkline.xs1 = includefile
		return
	}

	if m, directive, includefile := match2(text, `^\.\s*(s?include)\s+<([^>]+)>\s*(?:#.*)?$`); m {
		mkline.xtype = 7
		mkline.xb1 = directive == "include"
		mkline.xs1 = includefile
		return
	}

	if m, targets, whitespace, sources := match3(text, `^([^\s:]+(?:\s*[^\s:]+)*)(\s*):\s*([^#]*?)(?:\s*#.*)?$`); m {
		mkline.xtype = 8
		mkline.xs1 = targets
		mkline.xs2 = sources
		if whitespace != "" {
			line.warn0("Space before colon in dependency line.")
		}
		return
	}

	if matches(text, `^(<<<<<<<|=======|>>>>>>>)`) {
		return
	}

	line.error0("Unknown Makefile line format.")
	return mkline
}

func (mkline *MkLine) IsVarassign() bool   { return mkline.xtype == 1 }
func (mkline *MkLine) Varname() string     { return mkline.xs1 }
func (mkline *MkLine) Varcanon() string    { return mkline.xs2 }
func (mkline *MkLine) Varparam() string    { return mkline.xs3 }
func (mkline *MkLine) Op() string          { return mkline.xs4 }
func (mkline *MkLine) Value() string       { return mkline.xs5 }
func (mkline *MkLine) Comment() string     { return mkline.xs6 }
func (mkline *MkLine) IsShellcmd() bool    { return mkline.xtype == 2 }
func (mkline *MkLine) Shellcmd() string    { return mkline.xs1 }
func (mkline *MkLine) IsComment() bool     { return mkline.xtype == 3 }
func (mkline *MkLine) IsEmpty() bool       { return mkline.xtype == 4 }
func (mkline *MkLine) IsCond() bool        { return mkline.xtype == 5 }
func (mkline *MkLine) Indent() string      { return mkline.xs1 }
func (mkline *MkLine) Directive() string   { return mkline.xs2 }
func (mkline *MkLine) Args() string        { return mkline.xs3 }
func (mkline *MkLine) IsInclude() bool     { return mkline.xtype == 6 }
func (mkline *MkLine) MustExist() bool     { return mkline.xb1 }
func (mkline *MkLine) Includefile() string { return mkline.xs1 }
func (mkline *MkLine) IsSysinclude() bool  { return mkline.xtype == 7 }
func (mkline *MkLine) IsDependency() bool  { return mkline.xtype == 8 }
func (mkline *MkLine) Targets() string     { return mkline.xs1 }
func (mkline *MkLine) Sources() string     { return mkline.xs2 }

func (mkline *MkLine) checkVardef(varname, op string) {
	if G.opts.DebugTrace {
		defer tracecall2("MkLine.checkVardef", varname, op)()
	}

	defineVar(mkline, varname)
	mkline.checkVardefPermissions(varname, op)
}

func (mkline *MkLine) checkVardefPermissions(varname, op string) {
	if !G.opts.WarnPerm {
		return
	}

	vartype := mkline.getVariableType(varname)
	if vartype == nil {
		if G.opts.DebugMisc {
			mkline.debug1("No type definition found for %q.", varname)
		}
		return
	}

	perms := vartype.effectivePermissions(mkline.line.Fname)
	var needed AclPermissions
	switch op {
	case "=", "!=", ":=":
		needed = aclpSet
	case "?=":
		needed = aclpSetDefault
	case "+=":
		needed = aclpAppend
	}

	switch {
	case perms.contains(needed):
		break
	case perms == aclpUnknown:
		if G.opts.DebugUnchecked {
			mkline.line.debug1("Unknown permissions for %q.", varname)
		}
	default:
		alternativeActions := perms & aclpAllWrite
		alternativeFiles := vartype.allowedFiles(needed)
		switch {
		case alternativeActions != 0 && alternativeFiles != "":
			mkline.line.warnf("The variable %s may not be %s (only %s) in this file; it would be ok in %s.",
				varname, needed.HumanString(), alternativeActions.HumanString(), alternativeFiles)
		case alternativeFiles != "":
			mkline.line.warnf("The variable %s may not be %s in this file; it would be ok in %s.",
				varname, needed.HumanString(), alternativeFiles)
		case alternativeActions != 0:
			mkline.line.warnf("The variable %s may not be %s (only %s) in this file.",
				varname, needed.HumanString(), alternativeActions.HumanString())
		default:
			mkline.line.warnf("The variable %s may not be %s by any package.",
				varname, needed.HumanString())
		}
		explain4(
			"The allowed actions for a variable are determined based on the file",
			"name in which the variable is used or defined. The exact rules are",
			"hard-coded into pkglint. If they seem to be incorrect, please ask on",
			"the tech-pkg@NetBSD.org mailing list.")
	}
}

func (mkline *MkLine) checkVaruse(varname string, mod string, vuc *VarUseContext) {
	if G.opts.DebugTrace {
		defer tracecall("MkLine.checkVaruse", mkline, varname, mod, *vuc)()
	}

	vartype := mkline.getVariableType(varname)
	if G.opts.WarnExtra &&
		(vartype == nil || vartype.guessed == guGuessed) &&
		!varIsUsed(varname) &&
		!(G.mk != nil && G.mk.forVars[varname]) {
		mkline.warn1("%s is used but not defined. Spelling mistake?", varname)
	}

	mkline.checkVarusePermissions(varname, vuc)

	if varname == "LOCALBASE" && !G.isInfrastructure {
		mkline.warnVaruseLocalbase()
	}

	needsQuoting := mkline.variableNeedsQuoting(varname, vuc)

	if vuc.quoting == vucQuotFor {
		mkline.checkVaruseFor(varname, vartype, needsQuoting)
	}

	if G.opts.WarnQuoting && vuc.quoting != vucQuotUnknown && needsQuoting != nqDontKnow {
		mkline.checkVaruseShellword(varname, vartype, vuc, mod, needsQuoting)
	}

	if G.globalData.userDefinedVars[varname] != nil && !G.globalData.systemBuildDefs[varname] && !G.mk.buildDefs[varname] {
		mkline.warn1("The user-defined variable %s is used but not added to BUILD_DEFS.", varname)
		explain(
			"When a pkgsrc package is built, many things can be configured by the",
			"pkgsrc user in the mk.conf file. All these configurations should be",
			"recorded in the binary package, so the package can be reliably rebuilt.",
			"The BUILD_DEFS variable contains a list of all these user-settable",
			"variables, so please add your variable to it, too.")
	}
}

func (mkline *MkLine) checkVarusePermissions(varname string, vuc *VarUseContext) {
	if !G.opts.WarnPerm {
		return
	}

	// This is the type of the variable that is being used. Not to
	// be confused with vuc.vartype, which is the type of the
	// context in which the variable is used (often a ShellCommand
	// or, in an assignment, the type of the left hand side variable).
	vartype := mkline.getVariableType(varname)
	if vartype == nil {
		if G.opts.DebugMisc {
			mkline.debug1("No type definition found for %q.", varname)
		}
		return
	}

	perms := vartype.effectivePermissions(mkline.line.Fname)

	isLoadTime := false // Will the variable be used at load time?

	// Might the variable be used indirectly at load time, for example
	// by assigning it to another variable which then gets evaluated?
	isIndirect := false

	switch {
	case vuc.vartype != nil && vuc.vartype.guessed == guGuessed:
		// Don't warn about unknown variables.

	case vuc.time == vucTimeParse && !perms.contains(aclpUseLoadtime):
		isLoadTime = true

	case vuc.vartype != nil && vuc.vartype.union().contains(aclpUseLoadtime) && !perms.contains(aclpUseLoadtime):
		isLoadTime = true
		isIndirect = true
	}

	if isLoadTime && !isIndirect {
		mkline.warn1("%s should not be evaluated at load time.", varname)
		explain(
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
		mkline.warn1("%s should not be evaluated indirectly at load time.", varname)
		explain4(
			"The variable on the left-hand side may be evaluated at load time, but",
			"the variable on the right-hand side may not. Due to this assignment, it",
			"might be used indirectly at load-time, when it is not guaranteed to be",
			"properly defined.")
	}

	if !perms.contains(aclpUseLoadtime) && !perms.contains(aclpUse) {
		mkline.warn1("%s may not be used in this file.", varname)
	}
}

func (mkline *MkLine) warnVaruseLocalbase() {
	mkline.warn0("The LOCALBASE variable should not be used by packages.")
	explain(
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

func (mkline *MkLine) checkVaruseFor(varname string, vartype *Vartype, needsQuoting NeedsQuoting) {
	switch {
	case vartype == nil:
		// Cannot check anything here.

	case vartype.kindOfList == lkSpace:
		// Fine

	case needsQuoting == nqDoesntMatter || needsQuoting == nqNo:
		// Fine, this variable is not supposed to contain special characters.

	default:
		mkline.warn1("The variable %s should not be used in .for loops.", varname)
		explain4(
			"The .for loop splits its argument at sequences of white-space, as",
			"opposed to all other places in make(1), which act like the shell.",
			"Therefore only variables that are specifically designed to match this",
			"requirement should be used here.")
	}
}

func (mkline *MkLine) checkVaruseShellword(varname string, vartype *Vartype, vuc *VarUseContext, mod string, needsQuoting NeedsQuoting) {

	// In GNU configure scripts, a few variables need to be
	// passed through the :M* operator before they reach the
	// configure scripts.
	//
	// When doing checks outside a package, the :M* operator is needed for safety.
	needMstar := matches(varname, `^(?:.*_)?(?:CFLAGS||CPPFLAGS|CXXFLAGS|FFLAGS|LDFLAGS|LIBS)$`) &&
		(G.pkg == nil || G.pkg.vardef["GNU_CONFIGURE"] != nil)

	strippedMod := mod
	if m, stripped := match1(mod, `(.*?)(?::M\*)?(?::Q)?$`); m {
		strippedMod = stripped
	}

	if mod == ":M*:Q" && !needMstar {
		mkline.line.note0("The :M* modifier is not needed here.")

	} else if needsQuoting == nqYes {
		correctMod := strippedMod + ifelseStr(needMstar, ":M*:Q", ":Q")
		if mod != correctMod {
			if vuc.quoting == vucQuotPlain {
				if !mkline.line.autofixReplace("${"+varname+mod+"}", "${"+varname+correctMod+"}") {
					mkline.line.warnf("Please use ${%s%s} instead of ${%s%s}.", varname, correctMod, varname, mod)
				}
			} else {
				mkline.line.warnf("Please use ${%s%s} instead of ${%s%s} and make sure"+
					" the variable appears outside of any quoting characters.", varname, correctMod, varname, mod)
			}
			explain1(
				"See the pkgsrc guide, section \"quoting guideline\", for details.")
		}
	}

	if hasSuffix(mod, ":Q") && (needsQuoting == nqNo || needsQuoting == nqDoesntMatter) {
		bad := "${" + varname + mod + "}"
		good := "${" + varname + strings.TrimSuffix(mod, ":Q") + "}"
		needExplain := false
		if needsQuoting == nqNo && !mkline.line.autofixReplace(bad, good) {
			needExplain = true
			mkline.warn1("The :Q operator should not be used for ${%s} here.", varname)
		}
		if needsQuoting == nqDoesntMatter && !mkline.line.autofixReplace(bad, good) {
			needExplain = true
			mkline.line.note1("The :Q operator isn't necessary for ${%s} here.", varname)
		}
		if needExplain {
			explain(
				"Many variables in pkgsrc do not need the :Q operator, since they",
				"are not expected to contain white-space or other special characters.",
				"Examples for these \"safe\" variables are:",
				"",
				"\t* filenames",
				"\t* directory names",
				"\t* user and group names",
				"\t* tool names and tool paths",
				"\t* variable names",
				"\t* PKGNAME")
		}
	}
}

func (mkline *MkLine) checkDecreasingOrder(varname, value string) {
	if G.opts.DebugTrace {
		defer tracecall2("MkLine.checkDecreasingOrder", varname, value)()
	}

	strversions := splitOnSpace(value)
	intversions := make([]int, len(strversions))
	for i, strversion := range strversions {
		iver, err := strconv.Atoi(strversion)
		if err != nil || !(iver > 0) {
			mkline.error1("All values for %s must be positive integers.", varname)
			return
		}
		intversions[i] = iver
	}

	for i, ver := range intversions {
		if i > 0 && ver >= intversions[i-1] {
			mkline.warn1("The values for %s should be in decreasing order.", varname)
			explain2(
				"If they aren't, it may be possible that needless versions of packages",
				"are installed.")
		}
	}
}

func (mkline *MkLine) checkVarassign() {
	if G.opts.DebugTrace {
		defer tracecall0("MkLine.checkVarassign")()
	}

	varname := mkline.Varname()
	op := mkline.Op()
	value := mkline.Value()
	comment := mkline.Comment()
	varcanon := varnameCanon(varname)

	mkline.checkVardef(varname, op)
	mkline.checkVarassignBsdPrefs()

	mkline.checkText(value)
	mkline.checkVartype(varname, op, value, comment)

	// If the variable is not used and is untyped, it may be a spelling mistake.
	if op == ":=" && varname == strings.ToLower(varname) {
		if G.opts.DebugUnchecked {
			mkline.debug1("%s might be unused unless it is an argument to a procedure file.", varname)
		}

	} else if !varIsUsed(varname) {
		if vartypes := G.globalData.vartypes; vartypes[varname] != nil || vartypes[varcanon] != nil {
			// Ok
		} else if deprecated := G.globalData.deprecated; deprecated[varname] != "" || deprecated[varcanon] != "" {
			// Ok
		} else {
			mkline.warn1("%s is defined but not used. Spelling mistake?", varname)
		}
	}

	if matches(value, `/etc/rc\.d`) {
		mkline.warn0("Please use the RCD_SCRIPTS mechanism to install rc.d scripts automatically to ${RCD_SCRIPTS_EXAMPLEDIR}.")
	}

	if hasPrefix(varname, "_") && !G.isInfrastructure {
		mkline.warn1("Variable names starting with an underscore (%s) are reserved for internal pkgsrc use.", varname)
	}

	if varname == "PERL5_PACKLIST" && G.pkg != nil {
		if m, p5pkgname := match1(G.pkg.effectivePkgbase, `^p5-(.*)`); m {
			guess := "auto/" + strings.Replace(p5pkgname, "-", "/", -1) + "/.packlist"

			ucvalue, ucguess := strings.ToUpper(value), strings.ToUpper(guess)
			if ucvalue != ucguess && ucvalue != "${PERL5_SITEARCH}/"+ucguess {
				mkline.warn1("Unusual value for PERL5_PACKLIST -- %q expected.", guess)
			}
		}
	}

	if varname == "CONFIGURE_ARGS" && matches(value, `=\$\{PREFIX\}/share/kde`) {
		mkline.note0("Please .include \"../../meta-pkgs/kde3/kde3.mk\" instead of this line.")
		explain3(
			"That file probably does many things automatically and consistently that",
			"this package also does. When using kde3.mk, you can probably also leave",
			"out some explicit dependencies.")
	}

	if varname == "EVAL_PREFIX" {
		if m, evalVarname := match1(value, `^([\w_]+)=`); m {

			// The variables mentioned in EVAL_PREFIX will later be
			// defined by find-prefix.mk. Therefore, they are marked
			// as known in the current file.
			G.mk.vardef[evalVarname] = mkline
		}
	}

	if varname == "PYTHON_VERSIONS_ACCEPTED" {
		mkline.checkDecreasingOrder(varname, value)
	}

	if comment == "# defined" && !hasSuffix(varname, "_MK") && !hasSuffix(varname, "_COMMON") {
		mkline.note0("Please use \"# empty\", \"# none\" or \"yes\" instead of \"# defined\".")
		explain(
			"The value #defined says something about the state of the variable, but",
			"not what that _means_. In some cases a variable that is defined means",
			"\"yes\", in other cases it is an empty list (which is also only the",
			"state of the variable), whose meaning could be described with \"none\".",
			"It is this meaning that should be described.")
	}

	if m, revvarname := match1(value, `\$\{(PKGNAME|PKGVERSION)[:\}]`); m {
		if varname == "DIST_SUBDIR" || varname == "WRKSRC" {
			mkline.line.warnf("%s should not be used in %s, as it includes the PKGREVISION. Please use %s_NOREV instead.", revvarname, varname, revvarname)
		}
	}

	if fix := G.globalData.deprecated[varname]; fix != "" {
		mkline.warn2("Definition of %s is deprecated. %s", varname, fix)
	} else if fix := G.globalData.deprecated[varcanon]; fix != "" {
		mkline.warn2("Definition of %s is deprecated. %s", varname, fix)
	}

	if hasPrefix(varname, "SITES_") {
		mkline.warn0("SITES_* is deprecated. Please use SITES.* instead.")
	}

	mkline.checkVarassignPlistComment(varname, value)

	time := vucTimeRun
	if op == ":=" || op == "!=" {
		time = vucTimeParse
	}

	usedVars := mkline.extractUsedVariables(value)
	vuc := &VarUseContext{mkline.getVariableType(varname), time, vucQuotUnknown, vucExtentUnknown}
	for _, usedVar := range usedVars {
		mkline.checkVaruse(usedVar, "", vuc)
	}
}

func (mkline *MkLine) checkVarassignBsdPrefs() {
	if G.opts.WarnExtra && mkline.Op() == "?=" && G.pkg != nil && !G.pkg.seenBsdPrefsMk {
		switch mkline.Varcanon() {
		case "BUILDLINK_PKGSRCDIR.*", "BUILDLINK_DEPMETHOD.*", "BUILDLINK_ABI_DEPENDS.*":
			return
		}

		mkline.warn0("Please include \"../../mk/bsd.prefs.mk\" before using \"?=\".")
		explain(
			"The ?= operator is used to provide a default value to a variable. In",
			"pkgsrc, many variables can be set by the pkgsrc user in the mk.conf",
			"file. This file must be included explicitly. If a ?= operator appears",
			"before mk.conf has been included, it will not care about the user's",
			"preferences, which can result in unexpected behavior. The easiest way",
			"to include the mk.conf file is by including the bsd.prefs.mk file,",
			"which will take care of everything.")
	}
}

func (mkline *MkLine) checkVarassignPlistComment(varname, value string) {
	if matches(value, `^[^=]@comment`) {
		mkline.warn1("Please don't use @comment in %s.", varname)
		explain(
			"Here you are defining a variable containing @comment. As this value",
			"typically includes a space as the last character you probably also used",
			"quotes around the variable. This can lead to confusion when adding this",
			"variable to PLIST_SUBST, as all other variables are quoted using the :Q",
			"operator when they are appended. As it is hard to check whether a",
			"variable that is appended to PLIST_SUBST is already quoted or not, you",
			"should not have pre-quoted variables at all.",
			"",
			"To solve this, you should directly use PLIST_SUBST+= ${varname}=${value}",
			"or use any other variable for collecting the list of PLIST substitutions",
			"and later append that variable with PLIST_SUBST+= ${MY_PLIST_SUBST}.")
	}

	// Mark the variable as PLIST condition. This is later used in checkfile_PLIST.
	if G.pkg != nil {
		if m, plistVarname := match1(value, `(.+)=.*@comment.*`); m {
			G.pkg.plistSubstCond[plistVarname] = true
		}
	}
}

const reVarnamePlural = "^(?:" +
	".*[Ss]" +
	"|.*LIST" +
	"|.*_AWK" +
	"|.*_ENV" +
	"|.*_OVERRIDE" +
	"|.*_PREREQ" +
	"|.*_REQD" +
	"|.*_SED" +
	"|.*_SKIP" +
	"|.*_SRC" +
	"|.*_SUBST" +
	"|.*_TARGET" +
	"|.*_TMPL" +
	"|BROKEN_EXCEPT_ON_PLATFORM" +
	"|BROKEN_ON_PLATFORM" +
	"|BUILDLINK_DEPMETHOD" +
	"|BUILDLINK_LDADD" +
	"|BUILDLINK_TRANSFORM" +
	"|COMMENT" +
	"|CRYPTO" +
	"|DEINSTALL_TEMPLATE" +
	"|EVAL_PREFIX" +
	"|EXTRACT_ONLY" +
	"|FETCH_MESSAGE" +
	"|FIX_RPATH" +
	"|GENERATE_PLIST" +
	"|INSTALL_TEMPLATE" +
	"|INTERACTIVE_STAGE" +
	"|LICENSE" +
	"|MASTER_SITE_.*" +
	"|MASTER_SORT_REGEX" +
	"|NOT_FOR_COMPILER" +
	"|NOT_FOR_PLATFORM" +
	"|ONLY_FOR_COMPILER" +
	"|ONLY_FOR_PLATFORM" +
	"|PERL5_PACKLIST" +
	"|PLIST_CAT" +
	"|PLIST_PRE" +
	"|PKG_FAIL_REASON" +
	"|PKG_SKIP_REASON" +
	"|PREPEND_PATH" +
	"|PYTHON_VERSIONS_INCOMPATIBLE" +
	"|REPLACE_INTERPRETER" +
	"|REPLACE_PERL" +
	"|REPLACE_RUBY" +
	"|RESTRICTED" +
	"|SITES_.*" +
	"|TOOLS_ALIASES\\.*" +
	"|TOOLS_BROKEN" +
	"|TOOLS_CREATE" +
	"|TOOLS_GNU_MISSING" +
	"|TOOLS_NOOP" +
	")$"

func (mkline *MkLine) checkVartype(varname, op, value, comment string) {
	if G.opts.DebugTrace {
		defer tracecall("MkLine.checkVartype", varname, op, value, comment)()
	}

	if !G.opts.WarnTypes {
		return
	}

	varbase := varnameBase(varname)
	vartype := mkline.getVariableType(varname)

	if op == "+=" {
		if vartype != nil {
			if !vartype.mayBeAppendedTo() {
				mkline.warn0("The \"+=\" operator should only be used with lists.")
			}
		} else if !hasPrefix(varbase, "_") && !matches(varbase, reVarnamePlural) {
			mkline.warn1("As %s is modified using \"+=\", its name should indicate plural.", varname)
		}
	}

	switch {
	case vartype == nil:
		// Cannot check anything if the type is not known.
		if G.opts.DebugUnchecked {
			mkline.debug1("Unchecked variable assignment for %s.", varname)
		}

	case op == "!=":
		if G.opts.DebugMisc {
			mkline.debug1("Use of !=: %q", value)
		}

	case vartype.kindOfList == lkNone:
		mkline.checkVartypePrimitive(varname, vartype.checker, op, value, comment, vartype.isConsideredList(), vartype.guessed)

	case vartype.kindOfList == lkSpace:
		for _, word := range splitOnSpace(value) {
			mkline.checkVartypePrimitive(varname, vartype.checker, op, word, comment, true, vartype.guessed)
		}

	case vartype.kindOfList == lkShell:
		shline := NewMkShellLine(mkline)
		words, _ := splitIntoShellWords(mkline.line, value)
		for _, word := range words {
			mkline.checkVartypePrimitive(varname, vartype.checker, op, word, comment, true, vartype.guessed)
			shline.checkShellword(word, true)
		}
	}
}

// The `op` parameter is one of `=`, `+=`, `:=`, `!=`, `?=`, `use`, `pp-use`, ``.
// For some variables (like BuildlinkDepth), the operator influences the valid values.
// The `comment` parameter comes from a variable assignment, when a part of the line is commented out.
func (mkline *MkLine) checkVartypePrimitive(varname string, checker *VarChecker, op, value, comment string, isList bool, guessed Guessed) {
	if G.opts.DebugTrace {
		defer tracecall("MkLine.checkVartypePrimitive", varname, op, value, comment, isList, guessed)()
	}

	ctx := &VartypeCheck{mkline, mkline.line, varname, op, value, "", comment, isList, guessed == guGuessed}
	ctx.valueNovar = mkline.withoutMakeVariables(value, isList)

	checker.checker(ctx)
}

func (mkline *MkLine) withoutMakeVariables(value string, qModifierAllowed bool) string {
	valueNovar := value
	for {
		var m []string
		if m, valueNovar = replaceFirst(valueNovar, `\$\{([^{}]*)\}`, ""); m != nil {
			varuse := m[1]
			if !qModifierAllowed && hasSuffix(varuse, ":Q") {
				mkline.warn0("The :Q operator should only be used in lists and shell commands.")
			}
		} else {
			return valueNovar
		}
	}
}

func (mkline *MkLine) checkVaralign() {
	if !G.opts.WarnSpace {
		return
	}

	if m, prefix, align := match2(mkline.line.Text, `^( *[-*+A-Z_a-z0-9.${}\[]+\s*[!:?]?=)(\s*)`); m {
		if align != " " && strings.Trim(align, "\t") != "" {
			alignedWidth := tabLength(prefix + align)
			tabs := ""
			for tabLength(prefix+tabs) < alignedWidth {
				tabs += "\t"
			}
			if !mkline.line.autofixReplace(prefix+align, prefix+tabs) {
				mkline.note0("Alignment of variable values should be done with tabs, not spaces.")
			}
		}
	}
}

func (mkline *MkLine) checkText(text string) {
	if G.opts.DebugTrace {
		defer tracecall1("MkLine.checkText", text)()
	}

	if m, varname := match1(text, `^(?:[^#]*[^\$])?\$(\w+)`); m {
		mkline.warn1("$%[1]s is ambiguous. Use ${%[1]s} if you mean a Makefile variable or $$%[1]s if you mean a shell variable.", varname)
	}

	if mkline.line.firstLine == 1 {
		checklineRcsid(mkline.line, `# `, "# ")
	}

	if strings.Contains(text, "${WRKSRC}/../") {
		mkline.warn0("Using \"${WRKSRC}/..\" is conceptually wrong. Please use a combination of WRKSRC, CONFIGURE_DIRS and BUILD_DIRS instead.")
		explain2(
			"You should define WRKSRC such that all of CONFIGURE_DIRS, BUILD_DIRS",
			"and INSTALL_DIRS are subdirectories of it.")
	}

	// Note: A simple -R is not detected, as the rate of false positives is too high.
	if m, flag := match1(text, `\b(-Wl,--rpath,|-Wl,-rpath-link,|-Wl,-rpath,|-Wl,-R)\b`); m {
		mkline.warn1("Please use ${COMPILER_RPATH_FLAG} instead of %q.", flag)
	}

	rest := text
	for {
		m, r := replaceFirst(rest, `(?:^|[^$])\$\{([-A-Z0-9a-z_]+)(\.[\-0-9A-Z_a-z]+)?(?::[^\}]+)?\}`, "")
		if m == nil {
			break
		}
		rest = r

		varbase, varext := m[1], m[2]
		varname := varbase + varext
		varcanon := varnameCanon(varname)
		instead := G.globalData.deprecated[varname]
		if instead == "" {
			instead = G.globalData.deprecated[varcanon]
		}
		if instead != "" {
			mkline.warn2("Use of %q is deprecated. %s", varname, instead)
		}
	}
}

func (mkline *MkLine) checkIf() {
	if G.opts.DebugTrace {
		defer tracecall0("MkLine.checkIf")()
	}

	tree := mkline.parseMkCond(mkline.Args())

	{
		var pvarname, ppattern *string
		if tree.Match(NewTree("not", NewTree("empty", NewTree("match", &pvarname, &ppattern)))) {
			vartype := mkline.getVariableType(*pvarname)
			if vartype != nil && vartype.checker.IsEnum() {
				if !matches(*ppattern, `[\$\[*]`) && !vartype.checker.HasEnum(*ppattern) {
					mkline.warn2("Invalid :M value %q. Only { %s } are allowed.", *ppattern, vartype.checker.AllowedEnums())
				}
			}
			return
		}
	}

	{
		var pop, pvarname, pvalue *string
		if tree.Match(NewTree("compareVarStr", &pvarname, &pop, &pvalue)) {
			mkline.checkVartype(*pvarname, "use", *pvalue, "")
		}
	}
}

func (mkline *MkLine) checklineValidCharactersInValue(reValid string) {
	rest := regcomp(reValid).ReplaceAllString(mkline.Value(), "")
	if rest != "" {
		uni := ""
		for _, c := range rest {
			uni += fmt.Sprintf(" %U", c)
		}
		mkline.warn2("%s contains invalid characters (%s).", mkline.Varname(), uni[1:])
	}
}

func (mkline *MkLine) explainRelativeDirs() {
	explain3(
		"Directories in the form \"../../category/package\" make it easier to",
		"move a package around in pkgsrc, for example from pkgsrc-wip to the",
		"main pkgsrc repository.")
}

func (mkline *MkLine) checkRelativePkgdir(pkgdir string) {
	mkline.checkRelativePath(pkgdir, true)
	pkgdir = resolveVarsInRelativePath(pkgdir, false)

	if m, otherpkgpath := match1(pkgdir, `^(?:\./)?\.\./\.\./([^/]+/[^/]+)$`); m {
		if !fileExists(G.globalData.pkgsrcdir + "/" + otherpkgpath + "/Makefile") {
			mkline.error1("There is no package in %q.", otherpkgpath)
		}

	} else if !containsVarRef(pkgdir) {
		mkline.warn1("%q is not a valid relative package directory.", pkgdir)
		explain3(
			"A relative pathname always starts with \"../../\", followed",
			"by a category, a slash and a the directory name of the package.",
			"For example, \"../../misc/screen\" is a valid relative pathname.")
	}
}

func (mkline *MkLine) checkRelativePath(path string, mustExist bool) {
	if !G.isWip && strings.Contains(path, "/wip/") {
		mkline.line.error0("A main pkgsrc package must not depend on a pkgsrc-wip package.")
	}

	resolvedPath := resolveVarsInRelativePath(path, true)
	if containsVarRef(resolvedPath) {
		return
	}

	abs := resolvedPath
	if !hasPrefix(abs, "/") {
		abs = G.currentDir + "/" + abs
	}
	if _, err := os.Stat(abs); err != nil {
		if mustExist {
			mkline.error1("%q does not exist.", resolvedPath)
		}
		return
	}

	if hasPrefix(path, "../") &&
		!matches(path, `^\.\./\.\./[^/]+/[^/]`) &&
		!(G.curPkgsrcdir == ".." && hasPrefix(path, "../mk/")) && // For category Makefiles.
		!hasPrefix(path, "../../mk/") {
		mkline.warn1("Invalid relative path %q.", path)
	}
}

func matchMkCond(text string) (m bool, indent, directive, args string) {
	i, n := 0, len(text)
	if i < n && text[i] == '.' {
		i++
	} else {
		return
	}

	indentStart := i
	for i < n && (text[i] == ' ' || text[i] == '\t') {
		i++
	}
	indentEnd := i

	directiveStart := i
	for i < n && 'a' <= text[i] && text[i] <= 'z' {
		i++
	}
	directiveEnd := i
	directive = text[directiveStart:directiveEnd]
	switch directive {
	case "if", "ifdef", "ifndef", "else", "elif", "endif", "for", "endfor", "undef":
		break
	default:
		return
	}

	for i < n && (text[i] == ' ' || text[i] == '\t') {
		i++
	}

	argsStart := i
	for i < n && text[i] != '#' {
		i++
	}
	for i > argsStart && (text[i-1] == ' ' || text[i-1] == '\t') {
		i--
	}
	argsEnd := i

	m = true
	indent = text[indentStart:indentEnd]
	args = text[argsStart:argsEnd]
	return
}

func (mkline *MkLine) parseMkCond(cond string) *Tree {
	if G.opts.DebugTrace {
		defer tracecall1("parseMkCond", cond)()
	}

	const (
		repartVarname = `[A-Z_][A-Z0-9_]*(?:\.[\w_+\-]+)?`
		reDefined     = `^defined\((` + repartVarname + `)\)`
		reEmpty       = `^empty\((` + repartVarname + `)\)`
		reEmptyMatch  = `^empty\((` + repartVarname + `):M([^\$:{})]+)\)`
		reCompare     = `^\$\{(` + repartVarname + `)\}\s+(==|!=)\s+"([^"\$\\]*)"`
	)

	if m, rest := replaceFirst(cond, `^!`, ""); m != nil {
		return NewTree("not", mkline.parseMkCond(rest))
	}
	if m, rest := replaceFirst(cond, reDefined, ""); m != nil {
		return NewTree("defined", mkline.parseMkCond(rest))
	}
	if m, _ := replaceFirst(cond, reEmpty, ""); m != nil {
		return NewTree("empty", m[1])
	}
	if m, _ := replaceFirst(cond, reEmptyMatch, ""); m != nil {
		return NewTree("empty", NewTree("match", m[1], m[2]))
	}
	if m, _ := replaceFirst(cond, reCompare, ""); m != nil {
		return NewTree("compareVarStr", m[1], m[2], m[3])
	}
	return NewTree("unknown", cond)
}

type NeedsQuoting uint8

const (
	nqNo NeedsQuoting = iota
	nqYes
	nqDoesntMatter
	nqDontKnow
)

func (mkline *MkLine) variableNeedsQuoting(varname string, vuc *VarUseContext) NeedsQuoting {
	if G.opts.DebugTrace {
		defer tracecall("variableNeedsQuoting", varname, *vuc)()
	}

	vartype := mkline.getVariableType(varname)
	if vartype == nil || vuc.vartype == nil {
		return nqDontKnow
	}

	isPlainWord := vartype.checker.IsEnum()
	if c := vartype.checker; false ||
		c == CheckvarDistSuffix ||
		c == CheckvarFileMode ||
		c == CheckvarFilename ||
		c == CheckvarIdentifier ||
		c == CheckvarOption ||
		c == CheckvarPathname ||
		c == CheckvarPkgName ||
		c == CheckvarPkgOptionsVar ||
		c == CheckvarPkgRevision ||
		c == CheckvarRelativePkgDir ||
		c == CheckvarRelativePkgPath ||
		c == CheckvarUserGroupName ||
		c == CheckvarVarname ||
		c == CheckvarVersion ||
		c == CheckvarWrkdirSubdirectory {
		isPlainWord = true
	}
	if isPlainWord {
		if vartype.kindOfList == lkNone {
			return nqDoesntMatter
		}
		if vartype.kindOfList == lkShell && vuc.extent != vucExtentWordpart {
			return nqNo
		}
	}

	// In .for loops, the :Q operator is always misplaced, since
	// the items are broken up at white-space, not as shell words
	// like in all other parts of make(1).
	if vuc.quoting == vucQuotFor {
		return nqNo
	}

	// Determine whether the context expects a list of shell words or not.
	wantList := vuc.vartype.isConsideredList() && (vuc.quoting == vucQuotBackt || vuc.extent != vucExtentWordpart)
	haveList := vartype.isConsideredList()

	if G.opts.DebugQuoting {
		mkline.line.debugf("variableNeedsQuoting: varname=%q, context=%v, type=%v, wantList=%v, haveList=%v",
			varname, vuc, vartype, wantList, haveList)
	}

	// A shell word may appear as part of a shell word, for example COMPILER_RPATH_FLAG.
	if vuc.extent == vucExtentWordpart && vuc.quoting == vucQuotPlain {
		if vartype.kindOfList == lkNone && vartype.checker == CheckvarShellWord {
			return nqNo
		}
	}

	// Assuming the tool definitions don't include very special characters,
	// so they can safely be used inside any quotes.
	if G.globalData.varnameToToolname[varname] != "" {
		switch vuc.quoting {
		case vucQuotPlain:
			if vuc.extent != vucExtentWordpart {
				return nqNo
			}
		case vucQuotBackt:
			return nqNo
		case vucQuotDquot, vucQuotSquot:
			return nqDoesntMatter
		}
	}

	// Variables that appear as parts of shell words generally need
	// to be quoted. An exception is in the case of backticks,
	// because the whole backticks expression is parsed as a single
	// shell word by pkglint.
	if vuc.extent == vucExtentWordpart && vuc.quoting != vucQuotBackt {
		return nqYes
	}

	// Assigning lists to lists does not require any quoting, though
	// there may be cases like "CONFIGURE_ARGS+= -libs ${LDFLAGS:Q}"
	// where quoting is necessary.
	if wantList && haveList {
		return nqDoesntMatter
	}

	if wantList != haveList {
		return nqYes
	}

	if G.opts.DebugQuoting {
		mkline.line.debug1("Don't know whether :Q is needed for %q", varname)
	}
	return nqDontKnow
}

// Returns the type of the variable (maybe guessed based on the variable name),
// or nil if the type cannot even be guessed.
func (mkline *MkLine) getVariableType(varname string) *Vartype {

	if vartype := G.globalData.vartypes[varname]; vartype != nil {
		return vartype
	}
	if vartype := G.globalData.vartypes[varnameCanon(varname)]; vartype != nil {
		return vartype
	}

	if G.globalData.varnameToToolname[varname] != "" {
		return &Vartype{lkNone, CheckvarShellCommand, []AclEntry{{"*", aclpUse}}, guNotGuessed}
	}

	if m, toolvarname := match1(varname, `^TOOLS_(.*)`); m && G.globalData.varnameToToolname[toolvarname] != "" {
		return &Vartype{lkNone, CheckvarPathname, []AclEntry{{"*", aclpUse}}, guNotGuessed}
	}

	allowAll := []AclEntry{{"*", aclpAll}}
	allowRuntime := []AclEntry{{"*", aclpAllRuntime}}

	// Guess the datatype of the variable based on naming conventions.
	varbase := varnameBase(varname)
	var gtype *Vartype
	switch {
	case hasSuffix(varbase, "DIRS"):
		gtype = &Vartype{lkShell, CheckvarPathmask, allowRuntime, guGuessed}
	case hasSuffix(varbase, "DIR"), hasSuffix(varname, "_HOME"):
		gtype = &Vartype{lkNone, CheckvarPathname, allowRuntime, guGuessed}
	case hasSuffix(varbase, "FILES"):
		gtype = &Vartype{lkShell, CheckvarPathmask, allowRuntime, guGuessed}
	case hasSuffix(varbase, "FILE"):
		gtype = &Vartype{lkNone, CheckvarPathname, allowRuntime, guGuessed}
	case hasSuffix(varbase, "PATH"):
		gtype = &Vartype{lkNone, CheckvarPathlist, allowRuntime, guGuessed}
	case hasSuffix(varbase, "PATHS"):
		gtype = &Vartype{lkShell, CheckvarPathname, allowRuntime, guGuessed}
	case hasSuffix(varbase, "_USER"):
		gtype = &Vartype{lkNone, CheckvarUserGroupName, allowAll, guGuessed}
	case hasSuffix(varbase, "_GROUP"):
		gtype = &Vartype{lkNone, CheckvarUserGroupName, allowAll, guGuessed}
	case hasSuffix(varbase, "_ENV"):
		gtype = &Vartype{lkShell, CheckvarShellWord, allowRuntime, guGuessed}
	case hasSuffix(varbase, "_CMD"):
		gtype = &Vartype{lkNone, CheckvarShellCommand, allowRuntime, guGuessed}
	case hasSuffix(varbase, "_ARGS"):
		gtype = &Vartype{lkShell, CheckvarShellWord, allowRuntime, guGuessed}
	case hasSuffix(varbase, "_CFLAGS"), hasSuffix(varname, "_CPPFLAGS"), hasSuffix(varname, "_CXXFLAGS"), hasSuffix(varname, "_LDFLAGS"):
		gtype = &Vartype{lkShell, CheckvarShellWord, allowRuntime, guGuessed}
	case hasSuffix(varbase, "_MK"):
		gtype = &Vartype{lkNone, CheckvarUnchecked, allowAll, guGuessed}
	case hasPrefix(varbase, "PLIST."):
		gtype = &Vartype{lkNone, CheckvarYes, allowAll, guGuessed}
	}

	if G.opts.DebugVartypes {
		if gtype != nil {
			mkline.line.debug2("The guessed type of %q is %q.", varname, gtype.String())
		} else {
			mkline.line.debug1("No type definition found for %q.", varname)
		}
	}
	return gtype
}

// TODO: merge with determineUsedVariables
func (mkline *MkLine) extractUsedVariables(text string) []string {
	re := regcomp(`^(?:[^\$]+|\$[\$*<>?@]|\$\{([.0-9A-Z_a-z]+)(?::(?:[^\${}]|\$[^{])+)?\})`)
	rest := text
	var result []string
	for {
		m := re.FindStringSubmatchIndex(rest)
		if m == nil {
			break
		}
		varname := rest[negToZero(m[2]):negToZero(m[3])]
		rest = rest[:m[0]] + rest[m[1]:]
		if varname != "" {
			result = append(result, varname)
		}
	}

	if rest != "" && G.opts.DebugMisc {
		mkline.debug1("extractUsedVariables: rest=%q", rest)
	}
	return result
}

func (mkline *MkLine) determineUsedVariables() (varnames []string) {
	rest := mkline.line.Text

	if strings.HasPrefix(rest, "#") {
		return
	}

	for {
		p1 := strings.Index(rest, "${")
		p2 := strings.Index(rest, "$(")
		p3 := strings.Index(rest, "defined(")
		p4 := strings.Index(rest, "empty(")
		if p1 == -1 && p2 == -1 && p3 == -1 && p4 == -1 {
			return
		}
		min := -1
		if min == -1 || (p1 != -1 && p1 < min) {
			min = p1
		}
		if min == -1 || (p2 != -1 && p2 < min) {
			min = p2
		}
		if min == -1 || (p3 != -1 && p3 < min) {
			min = p3
		}
		if min == -1 || (p4 != -1 && p4 < min) {
			min = p4
		}
		rest = rest[min:]

		m := regcomp(`(?:\$\{|\$\(|defined\(|empty\()([0-9+.A-Z_a-z]+)[:})]`).FindStringSubmatchIndex(rest)
		if m == nil {
			return
		}
		varname := rest[m[2]:m[3]]
		varnames = append(varnames, varname)
		rest = rest[:m[0]] + rest[m[1]:]
	}
	return
}

// VarUseContext defines the context in which a variable is defined
// or used. Whether that is allowed depends on:
//
// * The variable’s data type, as defined in vardefs.go.
// * When used on the right-hand side of an assigment, the variable can
//   represent a list of words, a single word or even only part of a
//   word. This distinction decides upon the correct use of the :Q
//   operator.
// * When used in preprocessing statements like .if or .for, the other
//   operands of that statement should fit to the variable and are
//   checked against the variable type. For example, comparing OPSYS to
//   x86_64 doesn’t make sense.
type VarUseContext struct {
	vartype *Vartype
	time    vucTime
	quoting vucQuoting
	extent  vucExtent
}

type vucTime uint8

const (
	vucTimeUnknown vucTime = iota

	// When Makefiles are loaded, the operators := and != are evaluated,
	// as well as the conditionals .if, .elif and .for.
	// During loading, not all variables are available yet.
	// Variable values are still subject to change, especially lists.
	vucTimeParse

	// All files have been read, all variables can be referenced.
	// Variable values don’t change anymore.
	vucTimeRun
)

func (t vucTime) String() string { return [...]string{"unknown", "parse", "run"}[t] }

// The quoting context in which the variable is used.
// Depending on this context, the modifiers :Q or :M can be allowed or not.
type vucQuoting uint8

const (
	vucQuotUnknown vucQuoting = iota
	vucQuotPlain              // Example: echo LOCALBASE=${LOCALBASE}
	vucQuotDquot              // Example: echo "The version is ${PKGVERSION}."
	vucQuotSquot              // Example: echo 'The version is ${PKGVERSION}.'
	vucQuotBackt              // Example: echo \`sed 1q ${WRKSRC}/README\`

	// The .for loop in Makefiles. This is the only place where
	// variables are split on whitespace. Everywhere else (:Q, :M)
	// they are split like in the shell.
	//
	// Example: .for f in ${EXAMPLE_FILES}
	vucQuotFor
)

func (q vucQuoting) String() string {
	return [...]string{"unknown", "plain", "dquot", "squot", "backt", "mk-for"}[q]
}

type vucExtent uint8

const (
	vucExtentUnknown  vucExtent = iota
	vucExtentWord               // Example: echo ${LOCALBASE}
	vucExtentWordpart           // Example: echo LOCALBASE=${LOCALBASE}
)

func (e vucExtent) String() string {
	return [...]string{"unknown", "word", "wordpart"}[e]
}

func (vuc *VarUseContext) String() string {
	typename := "no-type"
	if vuc.vartype != nil {
		typename = vuc.vartype.String()
	}
	return fmt.Sprintf("(%s %s %s %s)", vuc.time, typename, vuc.quoting, vuc.extent)
}
