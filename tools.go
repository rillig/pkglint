package main

import (
	"netbsd.org/pkglint/trace"
	"path"
	"sort"
)

// Tool is one of the many standard shell utilities that are typically
// provided by the operating system, or, if missing, are installed via
// pkgsrc.
//
// See `mk/tools/`.
type Tool struct {
	Name           string // e.g. "sed", "gzip"
	Varname        string // e.g. "SED", "GZIP_CMD"
	MustUseVarForm bool   // True for `echo`, because of many differing implementations.
	Validity       Validity
}

// Tools collects all tools for a certain scope (global, package, file)
// and remembers whether these tools are defined at all,
// and whether they are declared to be used via USE_TOOLS.
type Tools struct {
	byName    map[string]*Tool
	byVarname map[string]*Tool
	usable    map[*Tool]bool
	SeenPrefs bool
}

func NewTools() Tools {
	return Tools{
		make(map[string]*Tool),
		make(map[string]*Tool),
		make(map[*Tool]bool),
		false}
}

// Define registers the tool by its name and the corresponding
// variable name (if nonempty). After this tool is added to USE_TOOLS, it
// may be used by this name (e.g. "awk") or by its variable (e.g. ${AWK}).
//
// See MakeUsable.
func (tr *Tools) Define(name, varname string, mkline MkLine, makeUsable bool) *Tool {
	if trace.Tracing {
		trace.Stepf("Tools.Define: %q %q in %s", name, varname, mkline)
	}

	tool := tr.def(name, varname, mkline)
	if varname != "" {
		tool.Varname = varname
	}
	if makeUsable {
		tr.MakeUsable(tool)
	}
	return tool
}

func (tr *Tools) def(name, varname string, mkline MkLine) *Tool {
	if mkline != nil && !tr.IsValidToolName(name) {
		mkline.Errorf("Invalid tool name %q.", name)
	}

	validity := Nowhere
	if mkline != nil && path.Base(mkline.Filename) == "bsd.prefs.mk" {
		validity = AfterPrefsMk
	}
	tool := &Tool{name, varname, false, validity}

	if name != "" {
		if existing := tr.byName[name]; existing != nil {
			tool = existing
		} else {
			tr.byName[name] = tool
		}
	}

	if varname != "" {
		if existing := tr.byVarname[varname]; existing != nil {
			tool = existing
		} else {
			tr.byVarname[varname] = tool
		}
	}

	return tool
}

func (tr *Tools) Trace() {
	if trace.Tracing {
		defer trace.Call0()()
	} else {
		return
	}

	var keys []string
	for k := range tr.byName {
		keys = append(keys, k)
	}
	sort.Strings(keys)

	for _, toolname := range keys {
		trace.Stepf("tool %+v", tr.byName[toolname])
	}
}

// ParseToolLine parses a tool definition from the pkgsrc infrastructure,
// e.g. in mk/tools/replace.mk.
func (tr *Tools) ParseToolLine(mkline MkLine, makeUsable bool) {
	switch {

	case mkline.IsVarassign():
		varparam := mkline.Varparam()
		value := mkline.Value()

		switch mkline.Varcanon() {
		case "TOOLS_CREATE":
			if tr.IsValidToolName(value) {
				tr.Define(value, "", mkline, makeUsable)
			}

		case "_TOOLS_VARNAME.*":
			if !containsVarRef(varparam) {
				tr.Define(varparam, value, mkline, makeUsable)
			}

		case "TOOLS_PATH.*", "_TOOLS_DEPMETHOD.*":
			if !containsVarRef(varparam) {
				tr.Define(varparam, "", mkline, makeUsable)
			}

		case "_TOOLS.*":
			if !containsVarRef(varparam) {
				tr.Define(varparam, "", mkline, makeUsable)
				for _, tool := range splitOnSpace(value) {
					tr.Define(tool, "", mkline, makeUsable)
				}
			}

		case "USE_TOOLS":
			if !containsVarRef(value) {
				for _, name := range splitOnSpace(value) {
					if tool := tr.ByNameTool(name); tool != nil {
						if path.Base(mkline.Filename) == "bsd.prefs.mk" {
							tool.Validity = AfterPrefsMk
						} else {
							tool.Validity = AtRunTime
						}
					}
				}
			}
		}

	case mkline.IsInclude():
		if path.Base(mkline.IncludeFile()) == "bsd.prefs.mk" {
			tr.SeenPrefs = true
		}
	}
}

// @deprecated
func (tr *Tools) ByVarname(varname string) (tool *Tool, usable bool) {
	tool = tr.byVarname[varname]
	usable = tr.Usable(tool)
	return
}

func (tr *Tools) ByVarnameTool(varname string) (tool *Tool) { return tr.byVarname[varname] }

// @deprecated
func (tr *Tools) ByName(name string) (tool *Tool, usable bool) {
	tool = tr.byName[name]
	usable = tr.Usable(tool)
	return
}

func (tr *Tools) ByNameTool(name string) (tool *Tool) { return tr.byName[name] }

// MakeUsable declares the tool as usable in the current scope.
// This usually happens because the tool is mentioned in USE_TOOLS.
func (tr *Tools) MakeUsable(tool *Tool) {
	if trace.Tracing && !tr.usable[tool] {
		trace.Stepf("Tools.MakeUsable %s", tool.Name)
	}

	tr.usable[tool] = true
}

// @deprecated
func (tr *Tools) Usable(tool *Tool) bool {
	return tr.usable[tool]
}

// UsableAtLoadTime means that the tool may be used by its variable
// name after bsd.prefs.mk has been included.
//
// Additionally, all allowed cases from UsableAtRunTime are allowed.
//
//  VAR:=   ${TOOL}           # Not allowed since bsd.prefs.mk is not
//                            # included yet.
//
//  .include "../../bsd.prefs.mk"
//
//  VAR:=   ${TOOL}           # Allowed.
//  VAR!=   ${TOOL}           # Allowed.
//
//  VAR=    ${${TOOL}:sh}     # Allowed; the :sh modifier is evaluated
//                            # lazily, but when VAR should ever be
//                            # evaluated at load time, this still means
//                            # load time.
//
//  .if ${TOOL:T} == "tool"   # Allowed.
//  .endif
func (tool *Tool) UsableAtLoadTime(seenPrefs bool) bool {
	return seenPrefs && tool.Validity == AfterPrefsMk
}

// UsableAtRunTime means that the tool may be used by its simple name
// in all {pre,do,post}-* targets, and by its variable name in all
// runtime contexts.
//
//  VAR:=   ${TOOL}           # Not allowed; TOOL might not be initialized yet.
//  VAR!=   ${TOOL}           # Not allowed; TOOL might not be initialized yet.
//
//  VAR=    ${${TOOL}:sh}     # Tricky; pkglint doesn't know enough context
//                            # to check this reliably, therefore it doesn't
//                            # produce any warnings. This pattern fails if
//                            # VAR is evaluated at load time.
//
//  own-target:
//          ${TOOL}           # Allowed.
//          tool              # Not allowed because the PATH might not be set
//                            # up for this target.
//
//  pre-configure:
//          ${TOOL}           # Allowed.
//          tool              # Allowed.
func (tool *Tool) UsableAtRunTime() bool {
	return tool.Validity == AtRunTime || tool.Validity == AfterPrefsMk
}

func (tr *Tools) AddAll(other Tools) {
	for _, tool := range other.byName {
		tr.def(tool.Name, tool.Varname, nil)
		if other.Usable(tool) {
			tr.usable[tool] = true
		}
	}
}
func (tr *Tools) IsValidToolName(name string) bool {
	return name == "[" || name == "echo -n" || matches(name, `^[-0-9a-z.]+$`)
}

type Validity uint8

const (
	// Nowhere means that the tool has not been added
	// to USE_TOOLS and therefore cannot be used at all.
	Nowhere Validity = iota

	// AfterPrefsMk means that the tool has been added to USE_TOOLS
	// before including bsd.prefs.mk and therefore can be used at
	// load time after bsd.prefs.mk has been included.
	//
	// The tool may be used as ${TOOL} everywhere.
	// The tool may be used by its plain name in {pre,do,post}-* targets.
	AfterPrefsMk

	// AtRunTime means that the tool has been added to USE_TOOLS
	// after including bsd.prefs.mk and therefore cannot be used
	// at load time.
	//
	// The tool may be used as ${TOOL} in all targets.
	// The tool may be used by its plain name in {pre,do,post}-* targets.
	AtRunTime
)

func (time Validity) String() string {
	return [...]string{"Nowhere", "AfterPrefsMk", "AtRunTime"}[time]
}
