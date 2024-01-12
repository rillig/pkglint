package pkglint

import (
	"github.com/rillig/pkglint/v23/textproc"
	"strings"
)

// MkExprChecker checks a single expression in a line from a makefile.
type MkExprChecker struct {
	expr    *MkExpr
	vartype *Vartype

	MkLines *MkLines
	MkLine  *MkLine
}

func NewMkExprChecker(use *MkExpr, mklines *MkLines, mkline *MkLine) *MkExprChecker {
	vartype := G.Pkgsrc.VariableType(mklines, use.varname)

	return &MkExprChecker{use, vartype, mklines, mkline}
}

// Check checks a single use of a variable in a specific context.
func (ck *MkExprChecker) Check(ectx *ExprContext) {
	if ck.expr.IsExpression() {
		return
	}

	ck.checkUndefined()
	ck.checkPermissions(ectx)

	ck.checkVarname(ectx.time)
	ck.checkModifiers()
	ck.checkAssignable(ectx)
	ck.checkQuoting(ectx)

	if G.Pkgsrc == nil {
		goto checkExpr
	}
	ck.checkToolsPlatform()
	ck.checkBuildDefs()
	ck.checkDeprecated()
	ck.checkPkgBuildOptions()

checkExpr:
	NewMkLineChecker(ck.MkLines, ck.MkLine).
		checkTextExpr(ck.expr.varname, ck.vartype, ectx.time)
}

func (ck *MkExprChecker) checkUndefined() {
	expr := ck.expr
	vartype := ck.vartype
	varname := expr.varname

	switch {
	case !G.WarnExtra,
		// Well-known variables are probably defined by the infrastructure.
		vartype != nil && !vartype.IsGuessed(),
		// TODO: At load time, check ck.MkLines.loadVars instead of allVars.
		ck.MkLines.allVars.IsDefinedSimilar(varname),
		ck.MkLines.checkAllData.forVars[varname],
		ck.MkLines.allVars.Mentioned(varname) != nil,
		ck.MkLines.pkg != nil && ck.MkLines.pkg.vars.IsDefinedSimilar(varname),
		containsExpr(varname),
		G.Project.Types().IsDefinedCanon(varname),
		varname == "":
		return
	}

	if ck.MkLines.once.FirstTimeSlice("used but not defined", varname) {
		ck.MkLine.Warnf("%s is used but not defined.", varname)
	}
}

func (ck *MkExprChecker) checkModifiers() {
	if len(ck.expr.modifiers) == 0 {
		return
	}

	ck.checkModifiersSuffix()
	ck.checkModifiersRange()

	for _, mod := range ck.expr.modifiers {
		ck.checkModifierLoop(mod)
	}
	// TODO: Investigate why :Q is not checked at this exact place.
}

func (ck *MkExprChecker) checkModifiersSuffix() {
	expr := ck.expr
	vartype := ck.vartype

	if !expr.modifiers[0].IsSuffixSubst() || vartype == nil || vartype.IsList() {
		return
	}

	ck.MkLine.Warnf("The :from=to modifier should only be used with lists, not with %s.", expr.varname)
	ck.MkLine.Explain(
		"Instead of (for example):",
		"\tMASTER_SITES=\t${HOMEPAGE:=repository/}",
		"",
		"Write:",
		"\tMASTER_SITES=\t${HOMEPAGE}repository/",
		"",
		"This expresses the intention of the code more clearly.")
}

// checkModifiersRange suggests to replace
// ${VAR:S,^,__magic__,1:M__magic__*:S,^__magic__,,} with the simpler ${VAR:[1]}.
func (ck *MkExprChecker) checkModifiersRange() {
	expr := ck.expr
	mods := expr.modifiers

	if len(mods) != 3 {
		return
	}

	m, _, from, to, options := mods[0].MatchSubst()
	if !m || from != "^" || !matches(to, `^\w+$`) || options != "1" {
		return
	}

	magic := to
	m, positive, pattern, _ := mods[1].MatchMatch()
	if !m || !positive || pattern != magic+"*" {
		return
	}

	m, _, from, to, options = mods[2].MatchSubst()
	if !m || from != "^"+magic || to != "" || options != "" {
		return
	}

	fix := ck.MkLine.Autofix()
	fix.Notef("The modifier %q can be written as %q.", expr.Mod(), ":[1]")
	fix.Explain(
		"The range modifier is much easier to understand than the",
		"complicated regular expressions, which were needed before",
		"the year 2006.")
	fix.Replace(expr.Mod(), ":[1]")
	fix.Apply()
}

func (ck *MkExprChecker) checkModifierLoop(mod MkExprModifier) {
	str := mod.String()
	if !hasSuffix(str, "@") {
		return
	}
	lex := textproc.NewLexer(str[:len(str)-1])
	if !lex.SkipByte('@') {
		return
	}
	varname := lex.NextBytesSet(textproc.Alnum)
	if varname == "" || !lex.SkipByte('@') {
		return
	}
	body := lex.Rest()
	// TODO: Are MkExpr interpreted the same in the before/after modifiers?
	if !matches(body, `^[-A-Za-z0-9 $();_{}]+$`) {
		return
	}
	varnameUse := "${" + varname + "}"

	n := 0
	ck.MkLine.ForEachUsedText(str, EctxRunTime, func(expr *MkExpr, time EctxTime) {
		if expr.varname == varname {
			n++
		}
	})
	if n != 1 {
		return
	}

	if rest := strings.TrimSuffix(body, varnameUse); len(rest) < len(body) {
		simpler := "S,^," + rest + ","
		fix := ck.MkLine.Autofix()
		fix.Notef("The modifier %q can be replaced with the simpler %q.",
			str, simpler)
		fix.Replace(str, simpler)
		fix.Apply()
	}

	if rest := strings.TrimPrefix(body, varnameUse); len(rest) < len(body) {
		simpler := "=" + rest
		fix := ck.MkLine.Autofix()
		fix.Notef("The modifier %q can be replaced with the simpler %q.",
			str, simpler)
		fix.Replace(str, simpler)
		fix.Apply()
	}
}

func (ck *MkExprChecker) checkVarname(time EctxTime) {
	varname := ck.expr.varname
	if varname == "@" {
		ck.MkLine.Warnf("Use %q instead of %q.", "${.TARGET}", "$@")
		ck.MkLine.Explain(
			"It is more readable and prevents confusion with the shell variable",
			"of the same name.")
	}

	if varname == "LOCALBASE" && !G.Infrastructure && time == EctxRunTime {
		fix := ck.MkLine.Autofix()
		fix.Warnf("Use PREFIX instead of LOCALBASE.")
		fix.ReplaceAfter("${", "LOCALBASE", "PREFIX")
		fix.Explain(
			"LOCALBASE is the user-settable variable,",
			"while PREFIX is the effective base directory,",
			"as determined by the pkgsrc infrastructure.",
			"While both variables contain the same directory,",
			"only PREFIX should be used, for consistency.")
		fix.Apply()
	}

	ck.checkVarnameBuildlink(varname)
}

func (ck *MkExprChecker) checkVarnameBuildlink(varname string) {
	pkg := ck.MkLines.pkg
	if pkg == nil {
		return
	}

	if !hasPrefix(varname, "BUILDLINK_PREFIX.") {
		return
	}

	varparam := varnameParam(varname)
	id := Buildlink3ID(varparam)
	if pkg.bl3Data[id] != nil || containsExpr(varparam) {
		return
	}

	// lang/lua/buildlink3.mk defines BUILDLINK_PREFIX.lua but
	// doesn't add it to BUILDLINK_TREE. Only the versioned
	// buildlink identifier (lua53) is added to BUILDLINK_TREE.
	// This is unusual.
	if pkg.vars.IsDefined(varname) {
		return
	}

	// Several packages contain makefile fragments that are more related
	// to the buildlink3.mk file than to the package Makefile.
	// These may use the buildlink identifier from the package itself.
	basename := ck.MkLine.Basename
	if basename != "Makefile" && basename != "options.mk" {
		bl3 := LoadMk(pkg.File("buildlink3.mk"), pkg, 0)
		if bl3 != nil {
			bl3Data := LoadBuildlink3Data(bl3)
			if bl3Data != nil && bl3Data.id == id {
				return
			}
		}
	}

	// TODO: Generalize the following paragraphs
	if id == "curses" && pkg.Includes("../../mk/curses.buildlink3.mk") != nil {
		return
	}
	if id == "mysql-client" && pkg.Includes("../../mk/mysql.buildlink3.mk") != nil {
		return
	}

	ck.MkLine.Warnf("Buildlink identifier %q is not known in this package.",
		varparam)
}

// checkPermissions checks the permissions when a variable is used,
// be it in a variable assignment, in a shell command, a conditional, or
// somewhere else.
//
// See MkAssignChecker.checkLeftPermissions.
func (ck *MkExprChecker) checkPermissions(ectx *ExprContext) {
	if !G.WarnPerm {
		return
	}
	if G.Infrastructure {
		// As long as vardefs.go doesn't explicitly define permissions for
		// infrastructure files, skip the check completely. This avoids
		// many wrong warnings.
		return
	}

	if trace.Tracing {
		defer trace.Call(ectx)()
	}

	// This is the type of the variable that is being used. Not to
	// be confused with ectx.vartype, which is the type of the
	// context in which the variable is used (often a ShellCommand
	// or, in an assignment, the type of the left-hand side variable).
	varname := ck.expr.varname
	vartype := ck.vartype
	if vartype == nil {
		if trace.Tracing {
			trace.Step1("No type definition found for %q.", varname)
		}
		return
	}

	if vartype.IsGuessed() {
		return
	}

	// Do not warn about unknown infrastructure variables.
	// These have all permissions to prevent warnings when they are used.
	// But when other variables are assigned to them, it would seem as if
	// these other variables could become evaluated at load time.
	// And this is something that most variables do not allow.
	if ectx.vartype != nil && ectx.vartype.basicType == BtUnknown {
		return
	}

	basename := ck.MkLine.Basename
	if basename == "hacks.mk" {
		return
	}

	effPerms := vartype.EffectivePermissions(basename)
	if effPerms.Contains(aclpUseLoadtime) {
		if ectx.time == EctxLoadTime {
			ck.checkUseAtLoadTime()
		}

		// Since the variable may be used at load time, it probably
		// may be used at run time as well. If it weren't, that would
		// be a rather strange permissions set.
		return
	}

	// At this point, the variable must not be used at load time.
	// Now determine whether it is directly used at load time because
	// the context already says so or, a little trickier, if it might
	// be used at load time somewhere in the future because it is
	// assigned to another variable, and that variable is allowed
	// to be used at load time.
	directly := ectx.time == EctxLoadTime
	indirectly := !directly && ectx.vartype != nil &&
		ectx.vartype.Union().Contains(aclpUseLoadtime)

	if !directly && !indirectly && effPerms.Contains(aclpUse) {
		// At this point, the variable is either used at run time, or the
		// time is not known.
		return
	}

	if directly || indirectly {
		// At this point, the variable is used at load time, although that
		// is not allowed by the permissions. The variable could be a tool
		// variable, and these tool variables have special rules.
		tool := G.ToolByVarname(ck.MkLines, varname)
		if tool != nil {

			// Whether a tool variable may be used at load time depends on
			// whether bsd.prefs.mk has been included before. That file
			// examines the tools that have been added to USE_TOOLS up to
			// this point and makes their variables available for use at
			// load time.
			if !tool.UsableAtLoadTime(ck.MkLines.Tools.SeenPrefs) {
				ck.warnToolLoadTime(varname, tool)
			}
			return
		}
	}

	if ck.MkLines.once.FirstTimeSlice("checkPermissions", varname) {
		ck.warnPermissions(ectx.vartype, varname, vartype, directly, indirectly)
	}
}

func (ck *MkExprChecker) warnPermissions(
	ectxVartype *Vartype,
	varname string,
	vartype *Vartype,
	directly, indirectly bool,
) {
	mkline := ck.MkLine

	anyPerms := vartype.Union()
	if !anyPerms.Contains(aclpUse) && !anyPerms.Contains(aclpUseLoadtime) {
		mkline.Warnf("%s should not be used in any file; it is a write-only variable.", varname)
		ck.explainPermissions(varname, vartype)
		return
	}

	if indirectly {
		// Some of the guessed variables may be used at load time. But since the
		// variable type and these permissions are guessed, pkglint should not
		// issue the following warning, since it is often wrong.
		if ectxVartype.IsGuessed() {
			return
		}

		mkline.Warnf("%s should not be used indirectly at load time (via %s).",
			varname, mkline.Varname())
		ck.explainPermissions(varname, vartype,
			"The variable on the left-hand side may be evaluated at load time,",
			"but the variable on the right-hand side should not.",
			"Because of the assignment in this line, the variable might be",
			"used indirectly at load time, before it is guaranteed to be",
			"properly initialized.")
		return
	}

	needed := aclpUse
	if directly {
		needed = aclpUseLoadtime
	}
	alternativeFiles := vartype.AlternativeFiles(needed)

	loadTimeExplanation := []string{
		"Many variables, especially lists of something, get their values incrementally.",
		"Therefore, it is generally unsafe to rely on their",
		"value until it is clear that it will never change again.",
		"This point is reached when the whole package Makefile is loaded and",
		"execution of the shell commands starts; in some cases earlier.",
		"",
		"Additionally, when using the \":=\" operator, each $$ is replaced",
		"with a single $, so variables that have references to shell",
		"variables or regular expressions are modified in a subtle way."}

	switch {
	case alternativeFiles == "" && directly:
		mkline.Warnf("%s should not be used at load time in any file.", varname)
		ck.explainPermissions(varname, vartype, loadTimeExplanation...)

	case alternativeFiles == "":
		mkline.Warnf("%s should not be used in any file.", varname)
		ck.explainPermissions(varname, vartype, loadTimeExplanation...)

	case directly:
		mkline.Warnf(
			"%s should not be used at load time in this file; "+
				"it would be ok in %s.",
			varname, alternativeFiles)
		ck.explainPermissions(varname, vartype, loadTimeExplanation...)

	default:
		mkline.Warnf(
			"%s should not be used in this file; it would be ok in %s.",
			varname, alternativeFiles)
		ck.explainPermissions(varname, vartype)
	}
}

func (ck *MkExprChecker) explainPermissions(varname string, vartype *Vartype, intro ...string) {
	if !G.Logger.Opts.Explain {
		return
	}

	// TODO: Starting with the second explanation, omit the common part. Instead, only list the permission rules.

	var expl []string

	if len(intro) > 0 {
		expl = append(expl, intro...)
		expl = append(expl, "")
	}

	expl = append(expl,
		"The allowed actions for a variable are determined based on the file",
		"name in which the variable is used or defined.",
		sprintf("The rules for %s are:", varname),
		"")

	for _, rule := range vartype.aclEntries {
		perms := rule.permissions.HumanString()

		files := rule.matcher.originalPattern
		if files == "*" {
			files = "any file"
		}

		if perms != "" {
			expl = append(expl, sprintf("* in %s, it may be %s", files, perms))
		} else {
			expl = append(expl, sprintf("* in %s, it should not be accessed at all", files))
		}
	}

	expl = append(expl,
		"",
		"If these rules seem to be incorrect, please ask on the tech-pkg@NetBSD.org mailing list.")

	ck.MkLine.Explain(expl...)
}

func (ck *MkExprChecker) checkUseAtLoadTime() {
	if ck.vartype.IsAlwaysInScope() || ck.MkLines.Tools.SeenPrefs {
		return
	}
	if ck.MkLines.pkg != nil && ck.MkLines.pkg.seenPrefs {
		return
	}
	mkline := ck.MkLine
	basename := mkline.Basename
	if basename == "builtin.mk" {
		return
	}

	if ck.vartype.IsPackageSettable() {
		// For package-settable variables, the explanation below
		// doesn't make sense since including bsd.prefs.mk won't
		// define any package-settable variables.
		return
	}

	if !ck.MkLines.once.FirstTime("bsd.prefs.mk") {
		return
	}

	include := condStr(
		basename == "buildlink3.mk",
		"mk/bsd.fast.prefs.mk",
		"mk/bsd.prefs.mk")
	currInclude := G.Pkgsrc.File(NewPkgsrcPath(NewPath(include)))

	mkline.Warnf("To use %s at load time, .include %q first.",
		ck.expr.varname, mkline.Rel(currInclude))
	mkline.Explain(
		"The user-settable variables and several other variables",
		"from the pkgsrc infrastructure are only available",
		"after the preferences have been loaded.",
		"",
		"Before that, these variables are undefined.")
}

// warnToolLoadTime logs a warning that the tool ${varname}
// should not be used at load time.
func (ck *MkExprChecker) warnToolLoadTime(varname string, tool *Tool) {
	// TODO: While using a tool by its variable name may be ok at load time,
	//  doing the same with the plain name of a tool is never ok.
	//  "VAR!= cat" is never guaranteed to call the correct cat.
	//  Even for shell builtins like echo and printf, bmake may decide
	//  to skip the shell and execute the commands via execve, which
	//  means that even echo is not a shell-builtin anymore.

	if tool.Validity == AfterPrefsMk {
		ck.MkLine.Warnf("To use the tool ${%s} at load time, bsd.prefs.mk has to be included before.", varname)
		return
	}

	if ck.MkLine.Basename == "Makefile" {
		pkgsrcTool := G.Pkgsrc.Tools.ByName(tool.Name)
		if pkgsrcTool != nil && pkgsrcTool.Validity == Nowhere {
			// The tool must have been added too late to USE_TOOLS,
			// i.e. after bsd.prefs.mk has been included.
			ck.MkLine.Warnf("To use the tool ${%s} at load time, it has to be added to USE_TOOLS before including bsd.prefs.mk.", varname)
			return
		}
	}

	ck.MkLine.Warnf("The tool ${%s} cannot be used at load time.", varname)
	ck.MkLine.Explain(
		"To use a tool at load time, it must be declared in the package",
		"Makefile by adding it to USE_TOOLS.",
		"After that, bsd.prefs.mk must be included.",
		"Adding the tool to USE_TOOLS at any later time has no effect,",
		"which means that the tool can only be used at run time.",
		"That's the rule for the package Makefiles.",
		"",
		"Since any other .mk file can be included from anywhere else, there",
		"is no guarantee that the tool is properly defined for using it at",
		"load time (see above for the tricky rules).",
		"Therefore, the tools can only be used at run time,",
		"except in the package Makefile itself.")
}

func (ck *MkExprChecker) checkAssignable(ectx *ExprContext) {
	leftType := ectx.vartype
	if leftType == nil || leftType.basicType != BtPathname {
		return
	}
	rightType := G.Pkgsrc.VariableType(ck.MkLines, ck.expr.varname)
	if rightType == nil || rightType.basicType != BtShellCommand {
		return
	}

	mkline := ck.MkLine
	if mkline.Varcanon() == "PKG_SHELL.*" {
		switch ck.expr.varname {
		case "SH", "BASH", "TOOLS_PLATFORM.sh":
			return
		}
	}

	mkline.Warnf(
		"Incompatible types: %s (type %q) cannot be assigned to type %q.",
		ck.expr.varname, rightType.basicType.name, leftType.basicType.name)
	mkline.Explain(
		"Shell commands often start with a pathname.",
		"They could also start with a list of environment variable",
		"definitions, since that is accepted by the shell.",
		"They can also contain addition command line arguments",
		"that are not filenames at all.")
}

// checkExprWords checks whether an expression of the form ${VAR}
// or ${VAR:modifiers} (without ':Q') is allowed in a certain context.
func (ck *MkExprChecker) checkQuoting(ectx *ExprContext) {
	if !G.WarnQuoting || ectx.quoting == EctxQuotUnknown {
		return
	}

	expr := ck.expr
	vartype := ck.vartype

	needsQuoting := ck.MkLine.VariableNeedsQuoting(ck.MkLines, expr, vartype, ectx)
	if needsQuoting == unknown {
		return
	}

	mod := expr.Mod()

	// In GNU configure scripts, a few variables need to be passed through
	// the :M* modifier before they reach the configure scripts. Otherwise,
	// the leading or trailing spaces will lead to strange caching errors
	// since the GNU configure scripts cannot handle these space characters.
	//
	// When doing checks outside a package, the :M* modifier is needed for safety.
	needMstar := (ck.MkLines.pkg == nil || ck.MkLines.pkg.vars.IsDefined("GNU_CONFIGURE")) &&
		matches(expr.varname, `^(?:.*_)?(?:CFLAGS|CPPFLAGS|CXXFLAGS|FFLAGS|LDFLAGS|LIBS)$`)

	mkline := ck.MkLine
	if mod == ":M*:Q" && !needMstar {
		if !vartype.IsGuessed() {
			mkline.Notef("The :M* modifier is not needed here.")
		}

	} else if needsQuoting == yes {
		ck.checkQuotingQM(mod, needMstar, ectx)
	}

	if hasSuffix(mod, ":Q") && needsQuoting == no {
		ck.warnRedundantModifierQ(mod)
	}
}

func (ck *MkExprChecker) checkQuotingQM(mod string, needMstar bool, ectx *ExprContext) {
	vartype := ck.vartype
	varname := ck.expr.varname

	modNoQ := strings.TrimSuffix(mod, ":Q")
	modNoM := strings.TrimSuffix(modNoQ, ":M*")
	correctMod := modNoM + condStr(needMstar, ":M*:Q", ":Q")

	if correctMod == mod+":Q" && ectx.IsWordPart && !vartype.IsShell() {

		isSingleWordConstant := func() bool {
			if ck.MkLines.pkg == nil {
				return false
			}

			varinfo := ck.MkLines.pkg.redundant.vars[varname]
			if varinfo == nil || !varinfo.vari.IsConstant() {
				return false
			}

			value := varinfo.vari.ConstantValue()
			return len(ck.MkLine.ValueFields(value)) == 1
		}

		if vartype.IsList() && isSingleWordConstant() {
			// Do not warn in this special case, which typically occurs
			// for BUILD_DIRS or similar package-settable variables.

		} else if vartype.IsList() {
			ck.warnListVariableInWord()
		} else {
			ck.warnMissingModifierQInWord()
		}

	} else if mod != correctMod {
		if ectx.quoting == EctxQuotPlain {
			ck.fixQuotingModifiers(correctMod, mod)
		} else {
			ck.warnWrongQuotingModifiers(correctMod, mod)
		}

	} else if ectx.quoting != EctxQuotPlain {
		ck.warnModifierQInQuotes(mod)
	}
}

func (ck *MkExprChecker) warnListVariableInWord() {
	mkline := ck.MkLine

	mkline.Warnf("The list variable %s should not be embedded in a word.",
		ck.expr.varname)
	mkline.Explain(
		"When a list variable has multiple elements, this expression expands",
		"to something unexpected:",
		"",
		"Example: ${MASTER_SITE_SOURCEFORGE}directory/ expands to",
		"",
		"\thttps://mirror1.sf.net/ https://mirror2.sf.net/directory/",
		"",
		"The first URL is missing the directory.",
		"To fix this, write",
		"\t${MASTER_SITE_SOURCEFORGE:=directory/}.",
		"",
		"Example: -l${LIBS} expands to",
		"",
		"\t-llib1 lib2",
		"",
		"The second library is missing the -l.",
		"To fix this, write ${LIBS:S,^,-l,}.")
}

func (ck *MkExprChecker) warnMissingModifierQInWord() {
	mkline := ck.MkLine

	mkline.Warnf("The variable %s should be quoted as part of a shell word.",
		ck.expr.varname)
	mkline.Explain(
		"This variable can contain spaces or other special characters.",
		"Therefore, it should be quoted by replacing ${VAR} with ${VAR:Q}.")
}

func (ck *MkExprChecker) fixQuotingModifiers(correctMod string, mod string) {
	varname := ck.expr.varname

	fix := ck.MkLine.Autofix()
	fix.Warnf("Please use ${%s%s} instead of ${%s%s}.", varname, correctMod, varname, mod)
	fix.Explain(
		seeGuide("Echoing a string exactly as-is", "echo-literal"))
	fix.Replace("${"+varname+mod+"}", "${"+varname+correctMod+"}")
	fix.Apply()
}

func (ck *MkExprChecker) warnWrongQuotingModifiers(correctMod string, mod string) {
	mkline := ck.MkLine
	varname := ck.expr.varname

	mkline.Warnf("Please use ${%s%s} instead of ${%s%s} and make sure"+
		" the variable appears outside of any quoting characters.", varname, correctMod, varname, mod)
	mkline.Explain(
		"The :Q modifier only works reliably when it is used outside of any",
		"quoting characters like 'single' or \"double\" quotes or `backticks`.",
		"",
		"Examples:",
		"Instead of CFLAGS=\"${CFLAGS:Q}\",",
		"     write CFLAGS=${CFLAGS:Q}.",
		"Instead of 's,@CFLAGS@,${CFLAGS:Q},',",
		"     write 's,@CFLAGS@,'${CFLAGS:Q}','.",
		"",
		seeGuide("Echoing a string exactly as-is", "echo-literal"))
}

func (ck *MkExprChecker) warnModifierQInQuotes(mod string) {
	mkline := ck.MkLine

	mkline.Warnf("Please move ${%s%s} outside of any quoting characters.",
		ck.expr.varname, mod)
	mkline.Explain(
		"The :Q modifier only works reliably when it is used outside of any",
		"quoting characters like 'single' or \"double\" quotes or `backticks`.",
		"",
		"Examples:",
		"Instead of CFLAGS=\"${CFLAGS:Q}\",",
		"     write CFLAGS=${CFLAGS:Q}.",
		"Instead of 's,@CFLAGS@,${CFLAGS:Q},',",
		"     write 's,@CFLAGS@,'${CFLAGS:Q}','.",
		"",
		seeGuide("Echoing a string exactly as-is", "echo-literal"))
}

func (ck *MkExprChecker) warnRedundantModifierQ(mod string) {
	varname := ck.expr.varname

	bad := "${" + varname + mod + "}"
	good := "${" + varname + strings.TrimSuffix(mod, ":Q") + "}"

	fix := ck.MkLine.Line.Autofix()
	fix.Notef("The :Q modifier isn't necessary for ${%s} here.", varname)
	fix.Explain(
		"Many variables in pkgsrc do not need the :Q modifier since they",
		"are not expected to contain whitespace or other special characters.",
		"Examples for these \"safe\" variables are:",
		"",
		"\t* filenames",
		"\t* directory names",
		"\t* user and group names",
		"\t* tool names and tool paths",
		"\t* variable names",
		"\t* package names (but not dependency patterns like pkg>=1.2)")
	fix.Replace(bad, good)
	fix.Apply()
}

func (ck *MkExprChecker) checkToolsPlatform() {
	if ck.MkLine.IsDirective() {
		return
	}

	varname := ck.expr.varname
	if varnameCanon(varname) != "TOOLS_PLATFORM.*" {
		return
	}

	indentation := ck.MkLines.indentation
	switch {
	case indentation.DependsOn("OPSYS"),
		indentation.DependsOn("MACHINE_PLATFORM"),
		indentation.DependsOn(varname):
		// TODO: Only return if the conditional is on the correct OPSYS.
		return
	}

	toolName := varnameParam(varname)
	tool := G.Pkgsrc.Tools.ByName(toolName)
	if tool == nil {
		return
	}

	conditional := false
	slice := tool.undefinedOn
	if len(slice) == 0 {
		conditional = true
		slice = tool.conditionalOn
	}
	if len(slice) == 0 {
		return
	}
	if ck.MkLine.IsVarassign() {
		param := varnameParam(ck.MkLine.Varname())
		if G.Pkgsrc.VariableType(nil, "OPSYS").basicType.HasEnum(param) &&
			!containsStr(slice, param) {
			return
		}
	}
	if conditional {
		ck.MkLine.Warnf("%s may be undefined on %s.",
			varname, joinCambridge("and", tool.conditionalOn...))
	} else {
		ck.MkLine.Warnf("%s is undefined on %s.",
			varname, joinCambridge("and", tool.undefinedOn...))
	}
}

func (ck *MkExprChecker) checkBuildDefs() {
	varname := ck.expr.varname

	if !G.Pkgsrc.UserDefinedVars.IsDefined(varname) || G.Pkgsrc.IsBuildDef(varname) {
		return
	}
	if ck.MkLines.buildDefs[varname] {
		return
	}
	if !ck.MkLines.once.FirstTimeSlice("BUILD_DEFS", varname) {
		return
	}

	ck.MkLine.Warnf("The user-defined variable %s is used but not added to BUILD_DEFS.", varname)
	ck.MkLine.Explain(
		"When a pkgsrc package is built, many things can be configured by the",
		"pkgsrc user in the mk.conf file.",
		"All these configurations should be recorded in the binary package",
		"so the package can be reliably rebuilt.",
		"The BUILD_DEFS variable contains a list of all these",
		"user-settable variables, so please add your variable to it, too.")
}

func (ck *MkExprChecker) checkDeprecated() {
	varname := ck.expr.varname
	instead := G.Project.Deprecated(varname)
	if instead == "" {
		return
	}

	ck.MkLine.Warnf("Use of %q is deprecated. %s", varname, instead)
}

func (ck *MkExprChecker) checkPkgBuildOptions() {
	pkg := ck.MkLines.pkg
	if pkg == nil {
		return
	}
	varname := ck.expr.varname
	if !hasPrefix(varname, "PKG_BUILD_OPTIONS.") {
		return
	}
	param := varnameParam(varname)
	if pkg.seenPkgbase.Seen(param) {
		return
	}

	ck.MkLine.Warnf("The PKG_BUILD_OPTIONS for %q are not available to this package.",
		param)
	ck.MkLine.Explain(
		"The variable parameter for PKG_BUILD_OPTIONS must correspond",
		"to the value of \"pkgbase\" above.",
		"",
		"For more information, see mk/pkg-build-options.mk.")
}
