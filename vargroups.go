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

	registered StringSet
	userVars   StringSet
	pkgVars    StringSet
	sysVars    StringSet
	defVars    StringSet
	useVars    StringSet
	ignVars    StringSet
	sortedVars StringSet
	listedVars StringSet

	unusedVars map[string]bool
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

	ck.registered = NewStringSet()
	ck.userVars = NewStringSet()
	ck.pkgVars = NewStringSet()
	ck.sysVars = NewStringSet()
	ck.defVars = NewStringSet()
	ck.useVars = NewStringSet()
	ck.ignVars = NewStringSet()
	ck.sortedVars = NewStringSet()
	ck.listedVars = NewStringSet()

	group := ""

	appendTo := func(ss *StringSet, mkline *MkLine) {
		varname := mkline.Varname()
		if varnameParam(varname) != group {
			mkline.Warnf("Expected %s.%s, but found %s.",
				varnameBase(varname), group, varnameParam(varname))
		}

		for _, varname := range mkline.ValueFields(mkline.Value()) {
			if !containsVarRef(varname) {
				if ck.registered.Contains(varname) {
					mkline.Warnf("Duplicate variable name %s in _VARGROUPS.", varname)
				}

				ss.Add(varname)
				ck.registered.Add(varname)
			}
		}
	}

	appendToStyle := func(ss *StringSet, mkline *MkLine) {
		varname := mkline.Varname()
		if varnameParam(varname) != group {
			mkline.Warnf("Expected %s.%s, but found %s.",
				varnameBase(varname), group, varnameParam(varname))
		}

		for _, varname := range mkline.ValueFields(mkline.Value()) {
			if !containsVarRef(varname) {
				ss.Add(varname)
			}
		}
	}

	mklines.ForEach(func(mkline *MkLine) {
		if mkline.IsVarassign() {
			switch varnameCanon(mkline.Varname()) {
			case "_VARGROUPS":
				group = mkline.Value()
			case "_USER_VARS.*":
				appendTo(&ck.userVars, mkline)
			case "_PKG_VARS.*":
				appendTo(&ck.pkgVars, mkline)
			case "_SYS_VARS.*":
				appendTo(&ck.sysVars, mkline)
			case "_DEF_VARS.*":
				appendTo(&ck.defVars, mkline)
			case "_USE_VARS.*":
				appendTo(&ck.useVars, mkline)
			case "_IGN_VARS.*":
				appendTo(&ck.ignVars, mkline)
			case "_SORTED_VARS.*":
				appendToStyle(&ck.sortedVars, mkline)
			case "_LISTED_VARS.*":
				appendToStyle(&ck.listedVars, mkline)
			}
		}
	})

	ck.unusedVars = make(map[string]bool)
	for _, varname := range ck.useVars.Elements {
		ck.unusedVars[varname] = true
	}
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
			ck.registered.Contains(varname),
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
	unusedVarnames := keysJoined(ck.unusedVars)
	if unusedVarnames != "" {
		mkline.Warnf("The variables %s are declared in _VARGROUPS but not actually used.",
			unusedVarnames)
	}
}
