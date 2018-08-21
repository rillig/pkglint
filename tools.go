package main

import (
	"netbsd.org/pkglint/trace"
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
}

func NewTools() Tools {
	return Tools{
		make(map[string]*Tool),
		make(map[string]*Tool),
		make(map[*Tool]bool)}
}

// Define registers the tool by its name and the corresponding
// variable name (if nonempty). After this tool is added to USE_TOOLS, it
// may be used by this name (e.g. "awk") or by its variable (e.g. ${AWK}).
//
// See MakeUsable.
func (tr *Tools) Define(name, varname string, mkline MkLine, makeUsable bool) *Tool {
	tool := tr.DefineTool(&Tool{name, varname, false, NeverValid}, mkline)
	if varname != "" {
		tool.Varname = varname
	}
	if makeUsable {
		tr.MakeUsable(tool)
	}
	return tool
}

func (tr *Tools) DefineTool(tool *Tool, mkline MkLine) *Tool {
	if trace.Tracing {
		trace.Stepf("Tools.DefineTool: %+v in %s", tool, mkline)
	}

	return tr.defineTool(tool, mkline)
}

func (tr *Tools) defineTool(tool *Tool, mkline MkLine) *Tool {
	tr.validateToolName(tool.Name, mkline)

	rv := tool
	if tool.Name != "" {
		if existing := tr.byName[tool.Name]; existing != nil {
			rv = existing
		} else {
			tr.byName[tool.Name] = tool
		}
	}
	if tool.Varname != "" {
		if existing := tr.byVarname[tool.Varname]; existing != nil {
			rv = existing
		} else {
			tr.byVarname[tool.Varname] = tool
		}
	}
	return rv
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
	if mkline.IsVarassign() {
		varname := mkline.Varname()
		value := mkline.Value()
		if varname == "TOOLS_CREATE" && (value == "[" || matches(value, `^[-\w.]+$`)) {
			tr.Define(value, "", mkline, makeUsable)

		} else if m, toolname := match1(varname, `^_TOOLS_VARNAME\.([-\w.]+|\[)$`); m {
			tool := tr.Define(toolname, value, mkline, makeUsable)
			if makeUsable {
				tr.MakeUsable(tool)
			}

		} else if m, toolname = match1(varname, `^(?:TOOLS_PATH|_TOOLS_DEPMETHOD)\.([-\w.]+|\[)$`); m {
			tool := tr.Define(toolname, "", mkline, makeUsable)
			if makeUsable {
				tr.MakeUsable(tool)
			}

		} else if m, toolname = match1(varname, `^_TOOLS\.(.*)`); m {
			tr.Define(toolname, "", mkline, makeUsable)
			for _, tool := range splitOnSpace(value) {
				tr.Define(tool, "", mkline, makeUsable)
			}
		}
	}
}

func (tr *Tools) ByVarname(varname string) (tool *Tool, usable bool) {
	tool = tr.byVarname[varname]
	usable = tr.Usable(tool)
	return
}

func (tr *Tools) ByName(name string) (tool *Tool, usable bool) {
	tool = tr.byName[name]
	usable = tr.Usable(tool)
	return
}

// MakeUsable declares the tool as usable in the current scope.
// This usually happens because the tool is mentioned in USE_TOOLS.
func (tr *Tools) MakeUsable(tool *Tool) {
	if trace.Tracing && !tr.usable[tool] {
		trace.Stepf("Tools.MakeUsable %s", tool.Name)
	}

	tr.usable[tool] = true
}

func (tr *Tools) Usable(tool *Tool) bool {
	return tr.usable[tool]
}

func (tr *Tools) AddAll(other Tools) {
	for _, tool := range other.byName {
		tr.defineTool(tool, nil)
		if other.Usable(tool) {
			tr.usable[tool] = true
		}
	}
}

func (tr *Tools) validateToolName(toolName string, mkline MkLine) {
	if mkline != nil && toolName != "echo -n" && !matches(toolName, `^([-a-z0-9.]+|\[)$`) {
		mkline.Errorf("Invalid tool name %q.", toolName)
	}
}

type Validity uint8

const (
	// NeverValid means that the tool has not been added
	// to USE_TOOLS and therefore cannot be used at all.
	NeverValid Validity = iota

	// AfterPrefsMk means that the tool has been added to USE_TOOLS
	// before including bsd.prefs.mk and therefore can be used at
	// load time after bsd.prefs.mk has been included.
	AfterPrefsMk

	// AtRunTime means that the tool has been added to USE_TOOLS
	// after including bsd.prefs.mk and therefore cannot be used
	// at load time.
	AtRunTime
)

func (time Validity) String() string {
	return [...]string{"NeverValid", "AfterPrefsMk", "AtRunTime"}[time]
}
