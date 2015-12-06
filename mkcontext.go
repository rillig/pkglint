package main

// MkContext contains data for the Makefile (or *.mk) that is currently checked.
type MkContext struct {
	forVars     map[string]bool    // The variables currently used in .for loops
	indentation []int              // Indentation depth of preprocessing directives
	target      string             // Current make(1) target
	vardef      map[string]*MkLine // varname => line; for all variables that are defined in the current file
	varuse      map[string]*MkLine // varname => line; for all variables that are used in the current file
	buildDefs   map[string]bool    // Variables that are registered in BUILD_DEFS, to ensure that all user-defined variables are added to it.
	plistVars   map[string]bool    // Variables that are registered in PLIST_VARS, to ensure that all user-defined variables are added to it.
	tools       map[string]bool    // Set of tools that are declared to be used.
}

func newMkContext() *MkContext {
	forVars := make(map[string]bool)
	indentation := make([]int, 1)
	vardef := make(map[string]*MkLine)
	varuse := make(map[string]*MkLine)
	buildDefs := make(map[string]bool)
	plistVars := make(map[string]bool)
	tools := make(map[string]bool)
	for tool := range G.globalData.predefinedTools {
		tools[tool] = true
	}
	return &MkContext{forVars, indentation, "", vardef, varuse, buildDefs, plistVars, tools}
}

func (ctx *MkContext) indentDepth() int {
	return ctx.indentation[len(ctx.indentation)-1]
}
func (ctx *MkContext) popIndent() {
	ctx.indentation = ctx.indentation[:len(ctx.indentation)-1]
}
func (ctx *MkContext) pushIndent(indent int) {
	ctx.indentation = append(ctx.indentation, indent)
}

func (ctx *MkContext) defineVar(mkline *MkLine, varname string) {
	if ctx.vardef[varname] == nil {
		ctx.vardef[varname] = mkline
	}
	varcanon := varnameCanon(varname)
	if ctx.vardef[varcanon] == nil {
		ctx.vardef[varcanon] = mkline
	}
}

func (ctx *MkContext) varValue(varname string) (value string, found bool) {
	if mkline := ctx.vardef[varname]; mkline != nil {
		return mkline.extra["value"].(string), true
	}
	return "", false
}
