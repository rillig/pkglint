package pkglint

import (
	"github.com/rillig/pkglint/v23/textproc"
	"strconv"
	"strings"
)

// MkAssignChecker checks a variable assignment line in a makefile.
type MkAssignChecker struct {
	MkLine  *MkLine
	MkLines *MkLines
}

func NewMkAssignChecker(mkLine *MkLine, mkLines *MkLines) *MkAssignChecker {
	return &MkAssignChecker{mkLine, mkLines}
}

func (ck *MkAssignChecker) check() {
	ck.checkLeft()
	ck.checkOp()
	ck.checkRight()
}

// checkLeft checks everything to the left of the assignment operator.
func (ck *MkAssignChecker) checkLeft() {
	varname := ck.MkLine.Varname()
	if G.Pkgsrc == nil {
		goto checkExpr
	}

	if !ck.mayBeDefined(varname) {
		ck.MkLine.Warnf("Variable names starting with an underscore (%s) are reserved for internal pkgsrc use.", varname)
	}
	ck.checkLeftNotUsed()
	ck.checkLeftOpsys()
	ck.checkLeftDeprecated()
	ck.checkLeftBsdPrefs()
	if !ck.checkLeftUserSettable() {
		ck.checkLeftPermissions()
		ck.checkLeftAbiDepends()
	}
	ck.checkLeftRationale()

checkExpr:
	NewMkLineChecker(ck.MkLines, ck.MkLine).checkTextExpr(
		varname,
		NewVartype(BtVariableName, NoVartypeOptions, NewACLEntry("*", aclpAll)),
		EctxLoadTime)
}

// checkLeftNotUsed checks whether the left-hand side of a variable
// assignment is not used. If it is unused and also doesn't have a predefined
// data type, it may be a spelling mistake.
func (ck *MkAssignChecker) checkLeftNotUsed() {
	varname := ck.MkLine.Varname()
	varcanon := varnameCanon(varname)

	if ck.MkLine.Op() == opAssignEval && varname == strings.ToLower(varname) {
		if trace.Tracing {
			trace.Step1("%s might be unused unless it is an argument to a procedure file.", varname)
		}
		return
	}

	if ck.MkLines.allVars.IsUsedSimilar(varname) {
		return
	}

	pkg := ck.MkLines.pkg
	if pkg != nil && pkg.vars.IsUsedSimilar(varname) {
		return
	}

	vartypes := G.Project.Types()
	if vartypes.IsDefinedExact(varname) || vartypes.IsDefinedExact(varcanon) {
		return
	}

	if G.Project.Deprecated(varname) != "" {
		return
	}

	if !ck.MkLines.warnedAboutDefinedNotUsed.FirstTime(varname) {
		return
	}

	ck.MkLine.Warnf("Variable \"%s\" is defined but not used.", varname)
	ck.MkLine.Explain(
		"This might be a simple typo.",
		"",
		"If a package provides a file containing several related variables",
		"(such as module.mk, app.mk, extension.mk), that file may define",
		"variables that look unused since they are only used by other packages.",
		"These variables should be documented at the head of the file;",
		"see mk/subst.mk for an example of such a documentation comment.")
}

// checkLeftOpsys checks whether the variable name is one of the OPSYS
// variables, which get merged with their corresponding VAR.${OPSYS} in
// bsd.pkg.mk.
func (ck *MkAssignChecker) checkLeftOpsys() {
	varname := ck.MkLine.Varname()
	varbase := varnameBase(varname)
	if !G.Pkgsrc.IsOpsysVar(varbase) {
		return
	}

	varparam := varnameParam(varname)
	if varparam == "" || varparam == "*" ||
		textproc.Lower.Contains(varparam[0]) {
		return
	}

	platforms := G.Pkgsrc.VariableType(ck.MkLines, "OPSYS").basicType
	if platforms.HasEnum(varparam) {
		return
	}

	ck.MkLine.Warnf(
		"Since %s is an OPSYS variable, "+
			"its parameter %q should be one of %s.",
		varbase, varparam, platforms.AllowedEnums())
}

func (ck *MkAssignChecker) checkLeftDeprecated() {
	if instead := G.Project.Deprecated(ck.MkLine.Varname()); instead != "" {
		ck.MkLine.Warnf("Definition of %s is deprecated. %s", ck.MkLine.Varname(), instead)
	}
}

func (ck *MkAssignChecker) checkLeftBsdPrefs() {
	mkline := ck.MkLine

	switch mkline.Varcanon() {
	case "BUILDLINK_PKGSRCDIR.*",
		"BUILDLINK_DEPMETHOD.*",
		"BUILDLINK_ABI_DEPENDS.*",
		"BUILDLINK_INCDIRS.*",
		"BUILDLINK_LIBDIRS.*":
		return
	}

	if !G.WarnExtra ||
		G.Infrastructure ||
		mkline.Op() != opAssignDefault ||
		ck.MkLines.Tools.SeenPrefs {
		return
	}

	// Package-settable variables may use the ?= operator before including
	// bsd.prefs.mk in situations like the following:
	//
	//  Makefile:  LICENSE=       package-license
	//             .include "module.mk"
	//  module.mk: LICENSE?=      default-license
	//
	vartype := G.Pkgsrc.VariableType(nil, mkline.Varname())
	if vartype != nil && vartype.IsPackageSettable() {
		return
	}

	if !ck.MkLines.warnedAboutDefaultAssignment.FirstTime() {
		return
	}
	mkline.Warnf("Include \"../../mk/bsd.prefs.mk\" before using \"?=\".")
	mkline.Explain(
		"The ?= operator is used to provide a default value to a variable.",
		"In pkgsrc, many variables can be set by the pkgsrc user in the",
		"mk.conf file.",
		"This file must be included explicitly.",
		"If a ?= operator appears before mk.conf has been included,",
		"it will not care about the user's preferences,",
		"which can result in unexpected behavior.",
		"",
		"The easiest way to include the mk.conf file is by including the",
		"bsd.prefs.mk file, which will take care of everything.")
}

// checkLeftUserSettable checks whether a package defines a
// variable that is marked as user-settable since it is defined in
// mk/defaults/mk.conf.
func (ck *MkAssignChecker) checkLeftUserSettable() bool {
	mkline := ck.MkLine
	varname := mkline.Varname()

	defaultMkline := G.Pkgsrc.UserDefinedVars.Mentioned(varname)
	if defaultMkline == nil {
		return false
	}
	defaultValue := defaultMkline.Value()

	// A few of the user-settable variables can also be set by packages.
	// That's an unfortunate situation since there is no definite source
	// of truth, but luckily only a few variables make use of it.
	vartype := G.Pkgsrc.VariableType(ck.MkLines, varname)
	if vartype.IsPackageSettable() {
		return true
	}

	switch {
	case G.Infrastructure:
		// No warnings, as the usage patterns between the packages
		// and the pkgsrc infrastructure differ a lot.

	case mkline.HasComment():
		// Assume that the comment contains a rationale for disabling
		// this particular check.

	case mkline.Op() == opAssignAppend:
		mkline.Warnf("Packages should not append to user-settable %s.", varname)

	case defaultValue != mkline.Value():
		mkline.Warnf(
			"Package sets user-defined %q to %q, which differs "+
				"from the default value %q from mk/defaults/mk.conf.",
			varname, mkline.Value(), defaultValue)

	case defaultMkline.IsCommentedVarassign():
		// Since the variable assignment is commented out in
		// mk/defaults/mk.conf, the package has to define it.

	default:
		mkline.Notef("Redundant definition for %s from mk/defaults/mk.conf.", varname)
		if !ck.MkLines.Tools.SeenPrefs {
			mkline.Explain(
				"Instead of defining the variable redundantly, it suffices to include",
				"../../mk/bsd.prefs.mk, which provides all user-settable variables.")
		}
	}

	return true
}

// checkLeftPermissions checks the permissions for the left-hand side
// of a variable assignment line.
//
// See checkPermissions.
func (ck *MkAssignChecker) checkLeftPermissions() {
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
		defer trace.Call0()()
	}

	mkline := ck.MkLine
	if ck.MkLine.Basename == "hacks.mk" {
		return
	}

	varname := mkline.Varname()
	op := mkline.Op()
	vartype := G.Pkgsrc.VariableType(ck.MkLines, varname)
	if vartype == nil {
		return
	}

	perms := vartype.EffectivePermissions(mkline.Basename)

	// E.g. USE_TOOLS:= ${USE_TOOLS:Nunwanted-tool}
	if op == opAssignEval && perms&aclpAppend != 0 {
		tokens, _ := mkline.ValueTokens()
		if len(tokens) == 1 && tokens[0].Expr != nil && tokens[0].Expr.varname == varname {
			return
		}
	}

	needed := aclpSet
	switch op {
	case opAssignDefault:
		needed = aclpSetDefault
	case opAssignAppend:
		needed = aclpAppend
	}

	switch {
	case perms.Contains(needed):
		break
	default:
		alternativeActions := perms & aclpAllWrite
		alternativeFiles := vartype.AlternativeFiles(needed)
		switch {
		case alternativeActions != 0 && alternativeFiles != "":
			mkline.Warnf("The variable %s should not be %s (only %s) in this file; it would be ok in %s.",
				varname, needed.HumanString(), alternativeActions.HumanString(), alternativeFiles)
		case alternativeFiles != "":
			mkline.Warnf("The variable %s should not be %s in this file; it would be ok in %s.",
				varname, needed.HumanString(), alternativeFiles)
		case alternativeActions != 0:
			mkline.Warnf("The variable %s should not be %s (only %s) in this file.",
				varname, needed.HumanString(), alternativeActions.HumanString())
		default:
			mkline.Warnf("The variable %s should not be %s by any package.",
				varname, needed.HumanString())
		}

		// XXX: explainPermissions doesn't really belong to MkExprChecker.
		(&MkExprChecker{nil, nil, ck.MkLines, mkline}).
			explainPermissions(varname, vartype)
	}
}

func (ck *MkAssignChecker) checkLeftAbiDepends() {
	mkline := ck.MkLine

	varname := mkline.Varname()
	if !hasPrefix(varname, "BUILDLINK_ABI_DEPENDS.") {
		return
	}

	basename := mkline.Basename
	if basename == "buildlink3.mk" {
		varparam := varnameParam(varname)
		bl3id := ""
		for _, bl3line := range ck.MkLines.mklines {
			if bl3line.IsVarassign() && bl3line.Varname() == "BUILDLINK_TREE" {
				bl3id = bl3line.Value()
				break
			}
		}
		if varparam == bl3id {
			return
		}
	}

	fix := mkline.Autofix()
	fix.Errorf("Packages must only require API versions, not ABI versions of dependencies.")
	fix.Explain(
		"When building a package from the sources,",
		"the version of the installed package does not matter.",
		"That version is specified by BUILDLINK_ABI_VERSION.",
		"",
		"The only version that matters is the API of the dependency,",
		"which is selected by specifying BUILDLINK_API_DEPENDS.")
	fix.Replace("BUILDLINK_ABI_DEPENDS", "BUILDLINK_API_DEPENDS")
	fix.Apply()
}

func (ck *MkAssignChecker) checkLeftRationale() {
	if !G.WarnExtra {
		return
	}

	mkline := ck.MkLine
	varname := mkline.Varname()

	if varname == "BUILD_DEPENDS" && !G.Infrastructure {
		mkline.Warnf("BUILD_DEPENDS should be TOOL_DEPENDS.")
		mkline.Explain(
			"When cross-building a package,",
			"BUILD_DEPENDS means that",
			"the dependency is needed for the target platform.",
			"These dependencies are handled by the buildlink",
			"mechanism.",
			"",
			"TOOL_DEPENDS, on the other hand,",
			"means that building the package",
			"needs the dependency on the native platform.",
			"",
			"Either replace BUILD_DEPENDS with TOOL_DEPENDS,",
			"or add a rationale explaining why BUILD_DEPENDS",
			"is the correct choice in this particular case.")
	}

	vartype := G.Pkgsrc.VariableType(ck.MkLines, varname)
	if vartype == nil || !vartype.NeedsRationale() {
		return
	}

	if mkline.HasRationale() {
		return
	}

	if varname == "PYTHON_VERSIONS_INCOMPATIBLE" && mkline.Value() == "27" {
		// No warning since it is rather common that a modern Python
		// package supports all Python versions starting with 3.0.
		return
	}

	mkline.Warnf("Setting variable %s should have a rationale.", varname)
	mkline.Explain(
		"Since this variable prevents the package from being built in some situations,",
		"the reasons for this restriction should be documented.",
		"Otherwise it becomes too difficult to check whether these restrictions still apply",
		"when the package is updated by someone else later.",
		"",
		"To add the rationale, put it in a comment at the end of this line,",
		"or in a separate comment in the line above.",
		"The rationale should try to answer these questions:",
		"",
		"* which specific aspects of the package are affected?",
		"* if it's a dependency, is the dependency too old or too new?",
		"* in which situations does a crash occur, if any?",
		"* has it been reported upstream?")
}

func (ck *MkAssignChecker) checkOp() {
	ck.checkOpShell()
	ck.checkOpAppendOnly()
}

func (ck *MkAssignChecker) checkOpShell() {
	mkline := ck.MkLine

	switch {
	case mkline.Op() != opAssignShell:
		return

	case mkline.HasComment():
		return

	case mkline.Basename == "builtin.mk":
		// These are typically USE_BUILTIN.* and BUILTIN_VERSION.*.
		// Authors of builtin.mk files usually know what they're doing.
		return

	case ck.MkLines.pkg == nil || ck.MkLines.pkg.vars.IsUsedAtLoadTime(mkline.Varname()):
		return
	}

	mkline.Notef("Consider the :sh modifier instead of != for %q.", mkline.Value())
	mkline.Explain(
		"For variable assignments using the != operator, the shell command",
		"is run every time the file is parsed.",
		"In some cases this is too early, and the command may not yet be installed.",
		"In other cases the command is executed more often than necessary.",
		"Most commands don't need to be executed for \"make clean\", for example.",
		"",
		"The :sh modifier defers execution until the variable value is actually needed.",
		"On the other hand, this means the command is executed each time the variable",
		"is evaluated.",
		"",
		"Example:",
		"",
		"\tEARLY_YEAR!=    date +%Y",
		"",
		"\tLATE_YEAR_CMD=  date +%Y",
		"\tLATE_YEAR=      ${LATE_YEAR_CMD:sh}",
		"",
		"\t# or, in a single line:",
		"\tLATE_YEAR=      ${date +%Y:L:sh}",
		"",
		"To suppress this note, provide an explanation in a comment at the end",
		"of the line, or force the variable to be evaluated at load time,",
		"by using it at the right-hand side of the := operator, or in an .if",
		"or .for directive.")
}

// https://gnats.netbsd.org/56352
func (ck *MkAssignChecker) checkOpAppendOnly() {

	if ck.MkLine.Op() != opAssign {
		return
	}

	varname := ck.MkLine.Varname()
	varbase := varnameBase(varname)

	// See pkgtools/bootstrap-mk-files/files/sys.mk
	switch varbase {
	case
		"CFLAGS",    // C
		"OBJCFLAGS", // Objective-C
		"FFLAGS",    // Fortran
		"RFLAGS",    // Ratfor
		"LFLAGS",    // Lex
		"LDFLAGS",   // Linker
		"LINTFLAGS", // Lint
		"PFLAGS",    // Pascal
		"YFLAGS",    // Yacc
		"LDADD":     // Just for symmetry
		break
	default:
		if hasSuffix(varbase, "_REQD") {
			vartype := G.Pkgsrc.VariableType(ck.MkLines, varname)
			if vartype != nil && vartype.IsList() == yes {
				break
			}
		}
		return
	}

	// At this point, it does not matter whether bsd.prefs.mk has been
	// included or not since the above variables get their default values
	// in sys.mk already, which is loaded even before the very first line
	// of the package Makefile.

	// The parameterized OPSYS variants do not get any default values before
	// the package Makefile is included.  Therefore, as long as bsd.prefs.mk
	// has not been included, the operator '=' can still be used.  Testing for
	// bsd.prefs.mk is only half the story, any other accidental overwrites
	// are caught by RedundantScope.
	if varbase != varname && !ck.MkLines.Tools.SeenPrefs {
		return
	}

	ck.MkLine.Warnf("Assignments to %q should use \"+=\", not \"=\".", varname)
}

// checkLeft checks everything to the right of the assignment operator.
func (ck *MkAssignChecker) checkRight() {
	mkline := ck.MkLine
	varname := mkline.Varname()
	op := mkline.Op()
	value := mkline.Value()
	comment := condStr(mkline.HasComment(), "#", "") + mkline.Comment()

	if trace.Tracing {
		defer trace.Call(varname, op, value)()
	}

	mkLineChecker := NewMkLineChecker(ck.MkLines, ck.MkLine)
	mkLineChecker.checkText(value)
	mkLineChecker.checkVartype(varname, op, value, comment)
	if mkline.IsEmpty() {
		// The line type can change due to an Autofix, see for example
		// VartypeCheck.WrkdirSubdirectory.
		return
	}

	ck.checkMisc()

	ck.checkRightExpr()
	ck.checkRightConfigureArgs()
	ck.checkRightUseLanguages()
}

func (ck *MkAssignChecker) checkRightCategory() {
	mkline := ck.MkLine
	if mkline.Op() != opAssign && mkline.Op() != opAssignDefault {
		return
	}

	categories := mkline.ValueFields(mkline.Value())
	primary := categories[0]
	dir := G.Pkgsrc.Rel(mkline.Filename()).Dir().Dir().Base()

	if primary == dir.String() || dir == "wip" || dir == "regress" {
		return
	}

	fix := mkline.Autofix()
	fix.Warnf("The primary category should be %q, not %q.", dir, primary)
	fix.Explain(
		"The primary category of a package should be its location in the",
		"pkgsrc directory tree, to make it easy to find the package.",
		"All other categories may be added after this primary category.")
	if len(categories) > 1 && categories[1] == dir.String() {
		fix.Replace(primary+" "+categories[1], categories[1]+" "+primary)
	}
	fix.Apply()
}

// checkRightConfigureArgs checks that a package does not try to override GNU
// configure options that are already handled by the infrastructure.
func (ck *MkAssignChecker) checkRightConfigureArgs() {
	pkg := ck.MkLines.pkg
	if G.Pkgsrc == nil || pkg == nil || !pkg.vars.IsDefined("GNU_CONFIGURE") {
		return
	}
	mkline := ck.MkLine
	if mkline.Varname() != "CONFIGURE_ARGS" {
		return
	}
	value := mkline.Value()
	m, mkOpt := match1(value, `^(--[\w-]+)=?`)
	if !m {
		return
	}

	gnuConfigure := G.Pkgsrc.File("mk/configure/gnu-configure.mk")
	gnuConfigureMk := LoadMk(gnuConfigure, pkg, MustSucceed|NotEmpty)
	for _, gnuMkline := range gnuConfigureMk.mklines {
		if !gnuMkline.IsVarassign() || gnuMkline.Varname() != "CONFIGURE_ARGS" {
			continue
		}
		m, gnuOpt := match1(gnuMkline.Value(), `^(--[\w-]+)=?`)
		if !m || mkOpt != gnuOpt {
			continue
		}
		mkline.Warnf("The option %q is already handled by %s.",
			mkOpt, mkline.RelMkLine(gnuMkline))
		mkline.Explain(
			"Packages should not specify this option directly,",
			"as the pkgsrc infrastructure may override its value",
			"based on other variables.",
			"See mk/configure/gnu-configure.mk for further details.")
		break
	}
}

func (ck *MkAssignChecker) checkRightUseLanguages() {
	mkline := ck.MkLine
	if mkline.Varname() != "USE_LANGUAGES" || G.Pkgsrc == nil {
		return
	}
	cc := G.Pkgsrc.VariableType(ck.MkLines, "USE_CC_FEATURES").basicType
	cxx := G.Pkgsrc.VariableType(ck.MkLines, "USE_CXX_FEATURES").basicType

	for _, word := range mkline.Fields() {
		if containsExpr(word) {
			continue
		}
		var varname string
		if cc.HasEnum(word) {
			varname = "USE_CC_FEATURES"
		} else if cxx.HasEnum(word) {
			varname = "USE_CXX_FEATURES"
		} else {
			continue
		}
		mkline.Warnf("The feature %q should be added "+
			"to %s instead of USE_LANGUAGES.", word, varname)
		mkline.Explain(
			"Specifying a C/C++ language version in USE_LANGUAGES",
			"is deprecated.",
			"Set USE_CC_FEATURES (for C)",
			"or USE_CXX_FEATURES (for C++).",
			"If forcing a specific language version is necessary",
			"for the build to succeed,",
			"also set FORCE_C_STD or FORCE_CXX_STD.")
	}
}

func (ck *MkAssignChecker) checkMisc() {
	mkline := ck.MkLine
	varname := mkline.Varname()
	value := mkline.Value()

	if G.Pkgsrc != nil && contains(value, "/etc/rc.d") && mkline.Varname() != "RPMIGNOREPATH" {
		mkline.Warnf("Use the RCD_SCRIPTS mechanism to install rc.d scripts automatically to ${RCD_SCRIPTS_EXAMPLEDIR}.")
	}

	if varname == "PYTHON_VERSIONS_ACCEPTED" {
		ck.checkDecreasingVersions()
	}

	if mkline.Comment() == " defined" && !hasSuffix(varname, "_MK") && !hasSuffix(varname, "_COMMON") {
		mkline.Notef("Use \"# empty\", \"# none\" or \"# yes\" instead of \"# defined\".")
		mkline.Explain(
			"The value #defined says something about the state of the variable,",
			"but not what that _means_.",
			"In some cases a variable that is defined",
			"means \"yes\", in other cases it is an empty list (which is also",
			"only the state of the variable), whose meaning could be described",
			"with \"none\".",
			"It is this meaning that should be described.")
	}

	switch varname {
	case "DIST_SUBDIR", "WRKSRC", "MASTER_SITES":
		// TODO: Replace regex with proper Expr matching.
		if m, revVarname := match1(value, `\$\{(PKGNAME|PKGVERSION)[:\}]`); m {
			mkline.Warnf("%s should not be used in %s as it includes the PKGREVISION. "+
				"Use %[1]s_NOREV instead.", revVarname, varname)
		}
	}

	if hasPrefix(varname, "SITES_") {
		mkline.Warnf("SITES_* is deprecated. Use SITES.* instead.")
		// No autofix since it doesn't occur anymore.
	}

	if varname == "PKG_SKIP_REASON" && ck.MkLines.indentation.DependsOn("OPSYS") {
		// TODO: Provide autofix for simple cases, like ".if ${OPSYS} == SunOS".
		// This needs support for high-level refactoring tools though.
		// As of June 2020, refactoring is limited to text replacements in single lines.
		mkline.Notef("Consider setting NOT_FOR_PLATFORM instead of " +
			"PKG_SKIP_REASON depending on ${OPSYS}.")
	}

	ck.checkMiscRedundantInstallationDirs()
}

func (ck *MkAssignChecker) checkDecreasingVersions() {
	mkline := ck.MkLine
	strVersions := mkline.Fields()
	intVersions := make([]int, len(strVersions))
	for i, strVersion := range strVersions {
		iver, err := strconv.Atoi(strVersion)
		if err != nil || !(iver > 0) {
			mkline.Errorf("Value %q for %s must be a positive integer.", strVersion, mkline.Varname())
			return
		}
		intVersions[i] = iver
	}

	for i, ver := range intVersions {
		if i > 0 && ver >= intVersions[i-1] {
			mkline.Warnf("The values for %s should be in decreasing order (%d before %d).",
				mkline.Varname(), ver, intVersions[i-1])
			mkline.Explain(
				"If they aren't, it may be possible that needless versions of",
				"packages are installed.")
		}
	}
}

func (ck *MkAssignChecker) checkMiscRedundantInstallationDirs() {
	mkline := ck.MkLine
	varname := mkline.Varname()
	pkg := ck.MkLines.pkg

	switch {
	case pkg == nil,
		varname != "INSTALLATION_DIRS",
		!matches(pkg.vars.LastValue("AUTO_MKDIRS"), `^[Yy][Ee][Ss]$`):
		return
	}

	for _, dir := range mkline.ValueFields(mkline.Value()) {
		if NewPath(dir).IsAbs() {
			continue
		}

		rel := NewRelPathString(dir)
		if pkg.Plist.UnconditionalDirs[rel] != nil {
			mkline.Notef("The directory %q is redundant in %s.", rel, varname)
			mkline.Explain(
				"This package defines AUTO_MKDIR, and the directory is contained in the PLIST.",
				"Therefore, it will be created anyway.")
		}
	}
}

// checkRightExpr checks that in a variable assignment,
// each expression on the right-hand side of the assignment operator
// has the correct data type and quoting.
func (ck *MkAssignChecker) checkRightExpr() {
	if trace.Tracing {
		defer trace.Call0()()
	}

	mkline := ck.MkLine
	op := mkline.Op()

	time := EctxRunTime
	if op == opAssignEval || op == opAssignShell {
		time = EctxLoadTime
	}

	vartype := G.Pkgsrc.VariableType(ck.MkLines, mkline.Varname())
	if op == opAssignShell {
		vartype = shellCommandsType
	}

	if vartype != nil && vartype.IsShell() {
		ck.checkExprShell(vartype, time)
	} else {
		mkLineChecker := NewMkLineChecker(ck.MkLines, ck.MkLine)
		mkLineChecker.checkTextExpr(ck.MkLine.Value(), vartype, time)
	}
}

// checkExprShell checks that in a variable assignment, each
// expression on the right-hand side of the assignment operator has the
// correct data type and quoting.
//
// See checkRightExpr for non-shell variables.
func (ck *MkAssignChecker) checkExprShell(vartype *Vartype, time EctxTime) {
	if trace.Tracing {
		defer trace.Call(vartype, time)()
	}

	isWordPart := func(tokens []*ShAtom, i int) bool {
		if i-1 >= 0 && tokens[i-1].Type.IsWord() {
			return true
		}
		if i+1 < len(tokens) && tokens[i+1].Type.IsWord() {
			return true
		}
		return false
	}

	mkline := ck.MkLine
	atoms := NewShTokenizer(mkline.Line, mkline.Value()).ShAtoms()
	for i, atom := range atoms {
		if expr := atom.Expr(); expr != nil {
			wordPart := isWordPart(atoms, i)
			ectx := ExprContext{vartype, time, atom.Quoting.ToExprContext(), wordPart}
			NewMkExprChecker(expr, ck.MkLines, mkline).Check(&ectx)
		}
	}
}

func (ck *MkAssignChecker) mayBeDefined(varname string) bool {
	if !hasPrefix(varname, "_") {
		return true
	}
	if G.Infrastructure {
		return true
	}
	if G.Pkgsrc.Types().Canon(varname) != nil {
		return true
	}

	// Defining the group 'cmake' allows the variable names '_CMAKE_*',
	// it's kind of a namespace declaration.
	vargroups := ck.MkLines.allVars.FirstDefinition("_VARGROUPS")
	if vargroups != nil {
		prefix := "_" + strings.ToUpper(vargroups.Value()) + "_"
		if hasPrefix(varname, prefix) {
			return true
		}
	}

	return false
}
