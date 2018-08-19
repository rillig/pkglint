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
	Name             string // e.g. "sed", "gzip"
	Varname          string // e.g. "SED", "GZIP_CMD"
	MustUseVarForm   bool   // True for `echo`, because of many differing implementations.
	UsableAtLoadTime bool   // May be used after including `bsd.prefs.mk`.
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

// DefineName registers the tool by its name. After this tool is added to
// USE_TOOLS, it may be used by this name (e.g. "awk"), but not by a
// corresponding variable (e.g. ${AWK}).
//
// See MakeUsable.
func (tr *Tools) DefineName(name string, mkline MkLine) *Tool {
	if trace.Tracing {
		defer trace.Call(name, mkline)()
	}

	tr.validateToolName(name, mkline)

	tool := tr.byName[name]
	if tool == nil {
		tool = &Tool{Name: name}
		tr.byName[name] = tool
	}
	return tool
}

// DefineVarname registers the tool by its name and the corresponding
// variable name. After this tool is added to USE_TOOLS, it may be used
// by this name (e.g. "awk") or by its variable (e.g. ${AWK}).
//
// The toolname may include the scope (:pkgsrc, :run, etc.).
func (tr *Tools) DefineVarname(name, varname string, mkline MkLine) *Tool {
	if trace.Tracing {
		defer trace.Call(name, varname, mkline)()
	}

	tool := tr.DefineName(name, mkline)
	tool.Varname = varname
	tr.byVarname[varname] = tool
	return tool
}

func (tr *Tools) DefineTool(tool *Tool, mkline MkLine) {
	if trace.Tracing {
		defer trace.Call(tool, mkline)()
	}

	tr.validateToolName(tool.Name, mkline)

	if tool.Name != "" && tr.byName[tool.Name] == nil {
		tr.byName[tool.Name] = tool
	}
	if tool.Varname != "" && tr.byVarname[tool.Varname] == nil {
		tr.byVarname[tool.Varname] = tool
	}
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
func (tr *Tools) ParseToolLine(mkline MkLine) {
	if mkline.IsVarassign() {
		varname := mkline.Varname()
		value := mkline.Value()
		if varname == "TOOLS_CREATE" && (value == "[" || matches(value, `^[-\w.]+$`)) {
			tr.DefineName(value, mkline)

		} else if m, toolname := match1(varname, `^_TOOLS_VARNAME\.([-\w.]+|\[)$`); m {
			tr.DefineVarname(toolname, value, mkline)

		} else if m, toolname = match1(varname, `^(?:TOOLS_PATH|_TOOLS_DEPMETHOD)\.([-\w.]+|\[)$`); m {
			tr.DefineName(toolname, mkline)

		} else if m, toolname = match1(varname, `^_TOOLS\.(.*)`); m {
			tr.DefineName(toolname, mkline)
			for _, tool := range splitOnSpace(value) {
				tr.DefineName(tool, mkline)
			}
		}
	}
}

func (tr *Tools) ByVarname(varname string) *Tool {
	return tr.byVarname[varname]
}

func (tr *Tools) ByName(name string) *Tool {
	return tr.byName[name]
}

func (tr *Tools) ByCommand(cmd *ShToken) *Tool {
	if tool := tr.byName[cmd.MkText]; tool != nil {
		return tool
	}
	if len(cmd.Atoms) == 1 {
		if varuse := cmd.Atoms[0].VarUse(); varuse != nil {
			if tool := tr.byVarname[varuse.varname]; tool != nil {
				return tool
			}
		}
	}
	return nil
}

// MakeUsable declares the tool as usable in the current scope.
// This usually happens because the tool is mentioned in USE_TOOLS.
func (tr *Tools) MakeUsable(tool *Tool) {
	tr.usable[tool] = true
}

func (tr *Tools) Usable(tool *Tool) bool {
	return tr.usable[tool]
}

func (tr *Tools) AddAll(other Tools) {
	for _, tool := range other.byName {
		tr.DefineTool(tool, nil)
		if other.Usable(tool) {
			tr.MakeUsable(tool)
		}
	}
}

func (tr *Tools) validateToolName(toolName string, mkline MkLine) {
	if mkline != nil && toolName != "echo -n" && !matches(toolName, `^([-a-z0-9.]+|\[)$`) {
		mkline.Errorf("Invalid tool name %q.", toolName)
	}
}
