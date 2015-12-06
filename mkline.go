package main

// Checks concerning single lines in Makefiles.

import (
	"strconv"
	"strings"
)

type MkLine struct {
	*Line
}

func NewMkLine(line *Line) *MkLine {
	parselineMk(line)
	return &MkLine{line}
}

func (mkline *MkLine) checkVardef(varname, op string) {
	defer tracecall("MkLine.checkVardef", varname, op)()

	defineVar(mkline, varname)
	mkline.checkVardefPermissions(varname, op)
}

func (mkline *MkLine) checkVardefPermissions(varname, op string) {
	if !G.opts.WarnPerm {
		return
	}

	perms := getVariablePermissions(mkline.Line, varname)
	var needed string
	switch op {
	case "=", "!=", ":=":
		needed = "s"
	case "?=":
		needed = "d"
	case "+=":
		needed = "a"
	}

	if !contains(perms, needed) {
		mkline.warnf("Permission %q requested for %s, but only { %s } are allowed.",
			ReadableVartypePermissions(needed), varname, ReadableVartypePermissions(perms))
		mkline.explain(
			"Pkglint restricts the allowed actions on variables based on the filename.",
			"",
			"The available permissions are:",
			"\tappend       append something using +=",
			"\tdefault      set a default value using ?=",
			"\tpreprocess   use a variable during preprocessing (e.g. .if, .for)",
			"\truntime      use a variable at runtime",
			"\t             (when the shell commands are run)",
			"\tset          set a variable using :=, =, !=",
			"\t             (which happens during preprocessing)",
			"",
			"A \"?\" means that pkglint doesn't know which permissions are allowed",
			"and which are not.")
	}
}

func (mkline *MkLine) checkVaruse(varname string, mod string, vuc *VarUseContext) {
	defer tracecall("MkLine.checkVaruse", mkline, varname, mod, *vuc)()

	vartype := getVariableType(mkline.Line, varname)
	if G.opts.WarnExtra &&
		(vartype == nil || vartype.guessed == guGuessed) &&
		!varIsUsed(varname) &&
		!(G.mkContext != nil && G.mkContext.forVars[varname]) {
		mkline.warnf("%s is used but not defined. Spelling mistake?", varname)
	}

	mkline.checkVarusePermissions(varname, vuc)

	if varname == "LOCALBASE" && !G.isInfrastructure {
		mkline.warnVaruseLocalbase()
	}

	needsQuoting := variableNeedsQuoting(mkline.Line, varname, vuc)

	if vuc.shellword == vucQuotFor {
		mkline.checkVaruseFor(varname, vartype, needsQuoting)
	}

	if G.opts.WarnQuoting && vuc.shellword != vucQuotUnknown && needsQuoting != nqDontKnow {
		mkline.checkVaruseShellword(varname, vartype, vuc, mod, needsQuoting)
	}

	if G.globalData.userDefinedVars[varname] != nil && !G.globalData.systemBuildDefs[varname] && !G.mkContext.buildDefs[varname] {
		mkline.warnf("The user-defined variable %s is used but not added to BUILD_DEFS.", varname)
		mkline.explain(
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

	perms := getVariablePermissions(mkline.Line, varname)

	isLoadTime := false // Will the variable be used at load time?

	// Might the variable be used indirectly at load time, for example
	// by assigning it to another variable which then gets evaluated?
	isIndirect := false

	switch {
	case vuc.vartype != nil && vuc.vartype.guessed == guGuessed:
		// Don't warn about unknown variables.

	case vuc.time == vucTimeParse && !contains(perms, "p"):
		isLoadTime = true

	case vuc.vartype != nil && contains(vuc.vartype.union(), "p") && !contains(perms, "p"):
		isLoadTime = true
		isIndirect = true
	}

	if isLoadTime && !isIndirect {
		mkline.warnf("%s should not be evaluated at load time.", varname)
		mkline.explain(
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
		mkline.warnf("%s should not be evaluated indirectly at load time.", varname)
		mkline.explain(
			"The variable on the left-hand side may be evaluated at load time, but",
			"the variable on the right-hand side may not. Due to this assignment, it",
			"might be used indirectly at load-time, when it is not guaranteed to be",
			"properly defined.")
	}

	if !contains(perms, "p") && !contains(perms, "u") {
		mkline.warnf("%s may not be used in this file.", varname)
	}
}

func (mkline *MkLine) warnVaruseLocalbase() {
	mkline.warnf("The LOCALBASE variable should not be used by packages.")
	mkline.explain(
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
		mkline.warnf("The variable %s should not be used in .for loops.", varname)
		mkline.explain(
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
		(G.pkgContext == nil || G.pkgContext.vardef["GNU_CONFIGURE"] != nil)

	strippedMod := mod
	if m, stripped := match1(mod, `(.*?)(?::M\*)?(?::Q)?$`); m {
		strippedMod = stripped
	}
	correctMod := strippedMod + ifelseStr(needMstar, ":M*:Q", ":Q")

	if mod == ":M*:Q" && !needMstar {
		mkline.notef("The :M* modifier is not needed here.")

	} else if mod != correctMod && needsQuoting == nqYes {
		if vuc.shellword == vucQuotPlain {
			mkline.warnf("Please use ${%s%s} instead of ${%s%s}.", varname, correctMod, varname, mod)
		} else {
			mkline.warnf("Please use ${%s%s} instead of ${%s%s} and make sure the variable appears outside of any quoting characters.", varname, correctMod, varname, mod)
		}
		mkline.explain(
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
		case nqNo:
			mkline.warnf("The :Q operator should not be used for ${%s} here.", varname)
			mkline.explain(expl...)
		case nqDoesntMatter:
			mkline.notef("The :Q operator isn't necessary for ${%s} here.", varname)
			mkline.explain(expl...)
		}
	}
}

func (mkline *MkLine) checkDecreasingOrder(varname, value string) {
	defer tracecall("MkLine.checkDecreasingOrder", varname, value)()

	strversions := splitOnSpace(value)
	intversions := make([]int, len(strversions))
	for i, strversion := range strversions {
		iver, err := strconv.Atoi(strversion)
		if err != nil || !(iver > 0) {
			mkline.errorf("All values for %s must be positive integers.", varname)
			return
		}
		intversions[i] = iver
	}

	for i, ver := range intversions {
		if i > 0 && ver >= intversions[i-1] {
			mkline.warnf("The values for %s should be in decreasing order.", varname)
			mkline.explain(
				"If they aren't, it may be possible that needless versions of packages",
				"are installed.")
		}
	}
}

func (mkline *MkLine) checkVarassign() {
	defer tracecall("MkLine.checkVarassign")()

	varname := mkline.extra["varname"].(string)
	op := mkline.extra["op"].(string)
	value := mkline.extra["value"].(string)
	comment := mkline.extra["comment"].(string)
	varbase := varnameBase(varname)
	varcanon := varnameCanon(varname)

	mkline.checkVardef(varname, op)

	if G.opts.WarnExtra && op == "?=" && G.pkgContext != nil && !G.pkgContext.seenBsdPrefsMk {
		switch varbase {
		case "BUILDLINK_PKGSRCDIR", "BUILDLINK_DEPMETHOD", "BUILDLINK_ABI_DEPENDS":
			// FIXME: What about these ones? They occur quite often.

		default:
			mkline.warnf("Please include \"../../mk/bsd.prefs.mk\" before using \"?=\".")
			mkline.explain(
				"The ?= operator is used to provide a default value to a variable. In",
				"pkgsrc, many variables can be set by the pkgsrc user in the mk.conf",
				"file. This file must be included explicitly. If a ?= operator appears",
				"before mk.conf has been included, it will not care about the user's",
				"preferences, which can result in unexpected behavior. The easiest way",
				"to include the mk.conf file is by including the bsd.prefs.mk file,",
				"which will take care of everything.")
		}
	}

	mkline.checkText(value)
	mkline.checkVartype(varname, op, value, comment)

	// If the variable is not used and is untyped, it may be a spelling mistake.
	if op == ":=" && varname == strings.ToLower(varname) {
		_ = G.opts.DebugUnchecked && mkline.debugf("%s might be unused unless it is an argument to a procedure file.", varname)

	} else if !varIsUsed(varname) {
		if vartypes := G.globalData.vartypes; vartypes[varname] != nil || vartypes[varcanon] != nil {
			// Ok
		} else if deprecated := G.globalData.deprecated; deprecated[varname] != "" || deprecated[varcanon] != "" {
			// Ok
		} else {
			mkline.warnf("%s is defined but not used. Spelling mistake?", varname)
		}
	}

	if matches(value, `/etc/rc\.d`) {
		mkline.warnf("Please use the RCD_SCRIPTS mechanism to install rc.d scripts automatically to ${RCD_SCRIPTS_EXAMPLEDIR}.")
	}

	if hasPrefix(varname, "_") && !G.isInfrastructure {
		mkline.warnf("Variable names starting with an underscore (%s) are reserved for internal pkgsrc use.", varname)
	}

	if varname == "PERL5_PACKLIST" && G.pkgContext != nil {
		if m, p5pkgname := match1(G.pkgContext.effectivePkgbase, `^p5-(.*)`); m {
			guess := "auto/" + strings.Replace(p5pkgname, "-", "/", -1) + "/.packlist"

			ucvalue, ucguess := strings.ToUpper(value), strings.ToUpper(guess)
			if ucvalue != ucguess && ucvalue != "${PERL5_SITEARCH}/"+ucguess {
				mkline.warnf("Unusual value for PERL5_PACKLIST -- %q expected.", guess)
			}
		}
	}

	if varname == "CONFIGURE_ARGS" && matches(value, `=\$\{PREFIX\}/share/kde`) {
		mkline.notef("Please .include \"../../meta-pkgs/kde3/kde3.mk\" instead of this line.")
		mkline.explain(
			"That file probably does many things automatically and consistently that",
			"this package also does. When using kde3.mk, you can probably also leave",
			"out some explicit dependencies.")
	}

	if varname == "EVAL_PREFIX" {
		if m, evalVarname := match1(value, `^([\w_]+)=`); m {

			// The variables mentioned in EVAL_PREFIX will later be
			// defined by find-prefix.mk. Therefore, they are marked
			// as known in the current file.
			G.mkContext.vardef[evalVarname] = mkline
		}
	}

	if varname == "PYTHON_VERSIONS_ACCEPTED" {
		mkline.checkDecreasingOrder(varname, value)
	}

	if comment == "# defined" && !matches(varname, `.*(?:_MK|_COMMON)$`) {
		mkline.notef("Please use \"# empty\", \"# none\" or \"yes\" instead of \"# defined\".")
		mkline.explain(
			"The value #defined says something about the state of the variable, but",
			"not what that _means_. In some cases a variable that is defined means",
			"\"yes\", in other cases it is an empty list (which is also only the",
			"state of the variable), whose meaning could be described with \"none\".",
			"It is this meaning that should be described.")
	}

	if m, revvarname := match1(value, `\$\{(PKGNAME|PKGVERSION)[:\}]`); m {
		if varname == "DIST_SUBDIR" || varname == "WRKSRC" {
			mkline.warnf("%s should not be used in %s, as it includes the PKGREVISION. Please use %s_NOREV instead.", revvarname, varname, revvarname)
		}
	}

	if fix := G.globalData.deprecated[varname]; fix != "" {
		mkline.warnf("Definition of %s is deprecated. %s", varname, fix)
	} else if fix := G.globalData.deprecated[varcanon]; fix != "" {
		mkline.warnf("Definition of %s is deprecated. %s", varname, fix)
	}

	if hasPrefix(varname, "SITES_") {
		mkline.warnf("SITES_* is deprecated. Please use SITES.* instead.")
	}

	if matches(value, `^[^=]@comment`) {
		mkline.warnf("Please don't use @comment in %s.", varname)
		mkline.explain(
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
	if G.pkgContext != nil && G.pkgContext.plistSubstCond != nil {
		if m, plistVarname := match1(value, `(.+)=.*@comment.*`); m {
			G.pkgContext.plistSubstCond[plistVarname] = true
		}
	}

	time := vucTimeRun
	switch op {
	case ":=", "!=":
		time = vucTimeParse
	}

	usedVars := extractUsedVariables(mkline.Line, value)
	vuc := &VarUseContext{
		time,
		getVariableType(mkline.Line, varname),
		vucQuotUnknown,
		vucExtentUnknown}
	for _, usedVar := range usedVars {
		mkline.checkVaruse(usedVar, "", vuc)
	}
}

const reVarnamePlural = "^(?:" +
	".*S" +
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
	defer tracecall("MkLine.checkVartype", varname, op, value, comment)()

	if !G.opts.WarnTypes {
		return
	}

	varbase := varnameBase(varname)
	vartype := getVariableType(mkline.Line, varname)

	if op == "+=" {
		if vartype != nil {
			if !vartype.mayBeAppendedTo() {
				mkline.warnf("The \"+=\" operator should only be used with lists.")
			}
		} else if !matches(varbase, `^_`) && !matches(varbase, reVarnamePlural) {
			mkline.warnf("As %s is modified using \"+=\", its name should indicate plural.", varname)
		}
	}

	switch {
	case vartype == nil:
		// Cannot check anything if the type is not known.
		_ = G.opts.DebugUnchecked && mkline.debugf("Unchecked variable assignment for %s.", varname)

	case op == "!=":
		_ = G.opts.DebugMisc && mkline.debugf("Use of !=: %q", value)

	case vartype.kindOfList == lkNone:
		mkline.checkVartypePrimitive(varname, vartype.checker, op, value, comment, vartype.isConsideredList(), vartype.guessed)

	default:
		var words []string
		if vartype.kindOfList == lkSpace {
			words = splitOnSpace(value)
		} else {
			words, _ = splitIntoShellwords(mkline.Line, value)
		}

		for _, word := range words {
			mkline.checkVartypePrimitive(varname, vartype.checker, op, word, comment, true, vartype.guessed)
			if vartype.kindOfList != lkSpace {
				NewMkShellLine(mkline).checkShellword(word, true)
			}
		}
	}
}

// The `op` parameter is one of `=`, `+=`, `:=`, `!=`, `?=`, `use`, `pp-use`, ``.
// For some variables (like BuildlinkDepth), the operator influences the valid values.
// The `comment` parameter comes from a variable assignment, when a part of the line is commented out.
func (mkline *MkLine) checkVartypePrimitive(varname string, checker *VarChecker, op, value, comment string, isList bool, guessed Guessed) {
	defer tracecall("MkLine.checkVartypePrimitive", varname, op, value, comment, isList, guessed)()

	ctx := &VartypeCheck{mkline, varname, op, value, "", comment, isList, guessed == guGuessed}
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
				mkline.warnf("The :Q operator should only be used in lists and shell commands.")
			}
		} else {
			return valueNovar
		}
	}
}

func (mkline *MkLine) checkVaralign() {
	text := mkline.text
	if m := regcomp(reVarassign).FindStringSubmatchIndex(text); m != nil {
		varname := text[m[2]:m[3]]
		space1 := text[m[3]:m[4]]
		op := text[m[4]:m[5]]
		align := text[m[5]:m[6]]

		if G.opts.WarnSpace && align != " " && strings.Trim(align, "\t") != "" {
			mkline.notef("Alignment of variable values should be done with tabs, not spaces.")
			prefix := varname + space1 + op
			alignedWidth := tabLength(prefix + align)
			tabs := ""
			for tabLength(prefix+tabs) < alignedWidth {
				tabs += "\t"
			}
			mkline.replace(prefix+align, prefix+tabs)
		}
	}
}

func (mkline *MkLine) checkText(text string) {
	defer tracecall("MkLine.checkText", text)()

	if m, varname := match1(text, `^(?:[^#]*[^\$])?\$(\w+)`); m {
		mkline.warnf("$%s is ambiguous. Use ${%s} if you mean a Makefile variable or $$%s if you mean a shell variable.", varname, varname, varname)
	}

	if mkline.lines == "1" {
		checklineRcsid(mkline.Line, `# `, "# ")
	}

	if contains(text, "${WRKSRC}/../") {
		mkline.warnf("Using \"${WRKSRC}/..\" is conceptually wrong. Please use a combination of WRKSRC, CONFIGURE_DIRS and BUILD_DIRS instead.")
		mkline.explain(
			"You should define WRKSRC such that all of CONFIGURE_DIRS, BUILD_DIRS",
			"and INSTALL_DIRS are subdirectories of it.")
	}

	// Note: A simple -R is not detected, as the rate of false positives is too high.
	if m, flag := match1(text, `\b(-Wl,--rpath,|-Wl,-rpath-link,|-Wl,-rpath,|-Wl,-R)\b`); m {
		mkline.warnf("Please use ${COMPILER_RPATH_FLAG} instead of %q.", flag)
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
			mkline.warnf("Use of %q is deprecated. %s", varname, instead)
		}
	}
}

func (mkline *MkLine) checkIf() {
	defer tracecall("MkLine.checkIf")()

	condition := mkline.extra["args"].(string)
	tree := parseMkCond(mkline.Line, condition)

	{
		var pvarname, ppattern *string
		if tree.Match(NewTree("not", NewTree("empty", NewTree("match", &pvarname, &ppattern)))) {
			vartype := getVariableType(mkline.Line, *pvarname)
			if vartype != nil && vartype.checker.IsEnum() {
				if !matches(*ppattern, `[\$\[*]`) && !vartype.checker.HasEnum(*ppattern) {
					mkline.warnf("Invalid :M value %q. Only { %s } are allowed.", *ppattern, vartype.checker.AllowedEnums())
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

func (mkline *MkLine) explainRelativeDirs() {
	mkline.explain(
		"Directories in the form \"../../category/package\" make it easier to",
		"move a package around in pkgsrc, for example from pkgsrc-wip to the",
		"main pkgsrc repository.")
}

func (mkline *MkLine) checkRelativePkgdir(pkgdir string) {
	mkline.checkRelativePath(pkgdir, true)
	pkgdir = resolveVarsInRelativePath(pkgdir, false)

	if m, otherpkgpath := match1(pkgdir, `^(?:\./)?\.\./\.\./([^/]+/[^/]+)$`); m {
		if !fileExists(G.globalData.pkgsrcdir + "/" + otherpkgpath + "/Makefile") {
			mkline.errorf("There is no package in %q.", otherpkgpath)
		}

	} else {
		mkline.warnf("%q is not a valid relative package directory.", pkgdir)
		mkline.explain(
			"A relative pathname always starts with \"../../\", followed",
			"by a category, a slash and a the directory name of the package.",
			"For example, \"../../misc/screen\" is a valid relative pathname.")
	}
}

func (mkline *MkLine) checkRelativePath(path string, mustExist bool) {
	if !G.isWip && contains(path, "/wip/") {
		mkline.errorf("A main pkgsrc package must not depend on a pkgsrc-wip package.")
	}

	resolvedPath := resolveVarsInRelativePath(path, true)
	if containsVarRef(resolvedPath) {
		return
	}

	abs := ifelseStr(hasPrefix(resolvedPath, "/"), "", G.currentDir+"/") + resolvedPath
	if !dirExists(abs) {
		if mustExist {
			mkline.errorf("%q does not exist.", resolvedPath)
		}
		return
	}

	switch {
	case matches(path, `^\.\./\.\./[^/]+/[^/]`):
	case hasPrefix(path, "../../mk/"):
		// There need not be two directory levels for mk/ files.
	case matches(path, `^\.\./mk/`) && G.curPkgsrcdir == "..":
		// That's fine for category Makefiles.
	case matches(path, `^\.\.`):
		mkline.warnf("Invalid relative path %q.", path)
	}
}
