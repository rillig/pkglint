package pkglint

import "strings"

// VargroupsChecker checks that the _VARGROUPS section of an infrastructure
// file matches the rest of the file content:
//
// - All variables that are used in the file should also be declared as used.
//
// - All variables that are declared to be used should actually be used.
//
// See mk/misc/show.mk, keyword _VARGROUPS.
type VargroupsChecker struct {
	mklines *MkLines
	skip    bool

	registered map[string]*MkLine
	userVars   map[string]*MkLine
	pkgVars    map[string]*MkLine
	sysVars    map[string]*MkLine
	defVars    map[string]*MkLine
	useVars    map[string]*MkLine
	ignVars    map[string]*MkLine
	sortedVars map[string]*MkLine
	listedVars map[string]*MkLine

	undefinedVars map[string]*MkLine
	unusedVars    map[string]*MkLine
}

func NewVargroupsChecker(mklines *MkLines) *VargroupsChecker {
	ck := VargroupsChecker{mklines: mklines}
	ck.init()
	return &ck
}

func (ck *VargroupsChecker) init() {
	mklines := ck.mklines
	scope := mklines.vars
	if !scope.Defined("_VARGROUPS") {
		ck.skip = true
		return
	}

	ck.registered = make(map[string]*MkLine)
	ck.userVars = make(map[string]*MkLine)
	ck.pkgVars = make(map[string]*MkLine)
	ck.sysVars = make(map[string]*MkLine)
	ck.defVars = make(map[string]*MkLine)
	ck.useVars = make(map[string]*MkLine)
	ck.ignVars = make(map[string]*MkLine)
	ck.sortedVars = make(map[string]*MkLine)
	ck.listedVars = make(map[string]*MkLine)

	group := ""

	checkGroupName := func(mkline *MkLine) {
		varname := mkline.Varname()
		if varnameParam(varname) != group {
			mkline.Warnf("Expected %s.%s, but found %q.",
				varnameBase(varname), group, varnameParam(varname))
		}
	}

	appendTo := func(vars map[string]*MkLine, mkline *MkLine) {
		checkGroupName(mkline)

		for _, varname := range mkline.ValueFields(mkline.Value()) {
			if containsVarRef(varname) {
				continue
			}

			if ck.registered[varname] != nil {
				mkline.Warnf("Duplicate variable name %s, already appeared in %s.",
					varname, mkline.RefTo(ck.registered[varname]))
			} else {
				ck.registered[varname] = mkline
			}

			vars[varname] = mkline
		}
	}

	appendToStyle := func(vars map[string]*MkLine, mkline *MkLine) {
		checkGroupName(mkline)

		for _, varname := range mkline.ValueFields(mkline.Value()) {
			if !containsVarRef(varname) {
				vars[varname] = mkline
			}
		}
	}

	mklines.ForEach(func(mkline *MkLine) {
		if mkline.IsVarassign() {
			switch varnameCanon(mkline.Varname()) {
			case "_VARGROUPS":
				group = mkline.Value()
			case "_USER_VARS.*":
				appendTo(ck.userVars, mkline)
			case "_PKG_VARS.*":
				appendTo(ck.pkgVars, mkline)
			case "_SYS_VARS.*":
				appendTo(ck.sysVars, mkline)
			case "_DEF_VARS.*":
				appendTo(ck.defVars, mkline)
			case "_USE_VARS.*":
				appendTo(ck.useVars, mkline)
			case "_IGN_VARS.*":
				appendTo(ck.ignVars, mkline)
			case "_SORTED_VARS.*":
				appendToStyle(ck.sortedVars, mkline)
			case "_LISTED_VARS.*":
				appendToStyle(ck.listedVars, mkline)
			}
		}
	})

	ck.undefinedVars = copyStringMkLine(ck.defVars)
	ck.unusedVars = copyStringMkLine(ck.useVars)
}

// CheckVargroups checks that each variable that is used or defined
// somewhere in the file is also registered in the _VARGROUPS section,
// in order to make it discoverable by "bmake show-all".
//
// This check is intended mainly for infrastructure files and similar
// support files, such as lang/*/module.mk.
func (ck *VargroupsChecker) Check(mkline *MkLine) {
	if ck.skip {
		return
	}

	ck.checkVarDef(mkline)
	ck.checkVarUse(mkline)
}

func (ck *VargroupsChecker) checkVarDef(mkline *MkLine) {
	if !mkline.IsVarassignMaybeCommented() {
		return
	}

	varname := mkline.Varname()
	if containsVarRef(varname) {
		return
	}
	varcanon := varnameCanon(varname)

	delete(ck.undefinedVars, varname)

	switch {
	case ck.registered[varname] != nil,
		varname == "_VARGROUPS",
		varcanon == "_USER_VARS.*",
		varcanon == "_PKG_VARS.*",
		varcanon == "_SYS_VARS.*",
		varcanon == "_DEF_VARS.*",
		varcanon == "_USE_VARS.*",
		varcanon == "_IGN_VARS.*",
		varcanon == "_LISTED_VARS.*",
		varcanon == "_SORTED_VARS.*":
		return
	}

	mkline.Warnf("Variable %s is defined but not mentioned in the _VARGROUPS section.", varname)
}

func (ck *VargroupsChecker) checkVarUse(mkline *MkLine) {
	mkline.ForEachUsed(func(varUse *MkVarUse, time VucTime) {
		varname := varUse.varname
		delete(ck.unusedVars, varname)

		switch {
		case containsVarRef(varname),
			hasSuffix(varname, "_MK"),
			varname == ".TARGET",
			varname == "RUN",
			varname == "TOUCH_FLAGS",
			varname == strings.ToLower(varname),
			G.Pkgsrc.Tools.ExistsVar(varname),
			ck.isShellCommand(varname),
			ck.registered[varname] != nil,
			!ck.mklines.once.FirstTimeSlice("_VARGROUPS", "use", varname):
			return
		}

		mkline.Warnf("Variable %s is used but not mentioned in the _VARGROUPS section.", varname)
	})
}

func (ck *VargroupsChecker) isShellCommand(varname string) bool {
	vartype := G.Pkgsrc.VariableType(ck.mklines, varname)
	if vartype != nil && vartype.basicType == BtShellCommand {
		return true
	}

	if varname == "TOOLS_SHELL" {
		return true
	}

	return false
}

func (ck *VargroupsChecker) Finish(mkline *MkLine) {
	if ck.skip {
		return
	}

	forEachStringMkLine(ck.undefinedVars, func(varname string, mkline *MkLine) {
		mkline.Warnf("The variable %s is not actually defined in this file.", varname)
	})
	forEachStringMkLine(ck.unusedVars, func(varname string, mkline *MkLine) {
		mkline.Warnf("The variable %s is not actually used in this file.", varname)
	})
}
