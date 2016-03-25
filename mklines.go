package main

import (
	"strings"
)

// MkLines contains data for the Makefile (or *.mk) that is currently checked.
type MkLines struct {
	mklines     []*MkLine
	lines       []*Line
	forVars     map[string]bool    // The variables currently used in .for loops
	indentation []int              // Indentation depth of preprocessing directives
	target      string             // Current make(1) target
	vardef      map[string]*MkLine // varname => line; for all variables that are defined in the current file
	varuse      map[string]*MkLine // varname => line; for all variables that are used in the current file
	buildDefs   map[string]bool    // Variables that are registered in BUILD_DEFS, to ensure that all user-defined variables are added to it.
	plistVars   map[string]bool    // Variables that are registered in PLIST_VARS, to ensure that all user-defined variables are added to it.
	tools       map[string]bool    // Set of tools that are declared to be used.
}

func NewMkLines(lines []*Line) *MkLines {
	mklines := make([]*MkLine, len(lines))
	for i, line := range lines {
		mklines[i] = NewMkLine(line)
	}
	tools := make(map[string]bool)
	for tool := range G.globalData.PredefinedTools {
		tools[tool] = true
	}

	return &MkLines{
		mklines,
		lines,
		make(map[string]bool),
		make([]int, 1),
		"",
		make(map[string]*MkLine),
		make(map[string]*MkLine),
		make(map[string]bool),
		make(map[string]bool),
		tools}
}

func (mklines *MkLines) indentDepth() int {
	return mklines.indentation[len(mklines.indentation)-1]
}
func (mklines *MkLines) popIndent() {
	mklines.indentation = mklines.indentation[:len(mklines.indentation)-1]
}
func (mklines *MkLines) pushIndent(indent int) {
	mklines.indentation = append(mklines.indentation, indent)
}

func (mklines *MkLines) DefineVar(mkline *MkLine, varname string) {
	if mklines.vardef[varname] == nil {
		mklines.vardef[varname] = mkline
	}
	varcanon := varnameCanon(varname)
	if mklines.vardef[varcanon] == nil {
		mklines.vardef[varcanon] = mkline
	}
}

func (mklines *MkLines) UseVar(mkline *MkLine, varname string) {
	varcanon := varnameCanon(varname)
	mklines.varuse[varname] = mkline
	mklines.varuse[varcanon] = mkline
	if G.Pkg != nil {
		G.Pkg.varuse[varname] = mkline
		G.Pkg.varuse[varcanon] = mkline
	}
}

func (mklines *MkLines) VarValue(varname string) (value string, found bool) {
	if mkline := mklines.vardef[varname]; mkline != nil {
		return mkline.Value(), true
	}
	return "", false
}

func (mklines *MkLines) Check() {
	if G.opts.DebugTrace {
		defer tracecall1(mklines.lines[0].Fname)()
	}

	G.Mk = mklines
	defer func() { G.Mk = nil }()

	allowedTargets := make(map[string]bool)
	prefixes := [...]string{"pre", "do", "post"}
	actions := [...]string{"fetch", "extract", "patch", "tools", "wrapper", "configure", "build", "test", "install", "package", "clean"}
	for _, prefix := range prefixes {
		for _, action := range actions {
			allowedTargets[prefix+"-"+action] = true
		}
	}

	// In the first pass, all additions to BUILD_DEFS and USE_TOOLS
	// are collected to make the order of the definitions irrelevant.
	mklines.DetermineUsedVariables()
	mklines.determineDefinedVariables()

	// In the second pass, the actual checks are done.

	mklines.lines[0].CheckRcsid(`#\s+`, "# ")

	var substcontext SubstContext
	var varalign VaralignBlock
	for _, mkline := range mklines.mklines {
		mkline.Check()
		varalign.Check(mkline)

		switch {
		case mkline.IsEmpty():
			substcontext.Finish(mkline)

		case mkline.IsVarassign():
			mklines.target = ""
			substcontext.Varassign(mkline)

		case mkline.IsInclude():
			mklines.target = ""

		case mkline.IsCond():
			mklines.checklineCond(mkline)

		case mkline.IsDependency():
			mklines.checklineDependencyRule(mkline, mkline.Targets(), mkline.Sources(), allowedTargets)
		}
	}
	lastMkline := mklines.mklines[len(mklines.mklines)-1]
	substcontext.Finish(lastMkline)
	varalign.Finish()

	ChecklinesTrailingEmptyLines(mklines.lines)

	if len(mklines.indentation) != 1 && mklines.indentDepth() != 0 {
		lastMkline.Line.Errorf("Directive indentation is not 0, but %d.", mklines.indentDepth())
	}

	SaveAutofixChanges(mklines.lines)
}

func (mklines *MkLines) determineDefinedVariables() {
	for _, mkline := range mklines.mklines {
		if !mkline.IsVarassign() {
			continue
		}

		varcanon := mkline.Varcanon()
		switch varcanon {
		case "BUILD_DEFS", "PKG_GROUPS_VARS", "PKG_USERS_VARS":
			for _, varname := range splitOnSpace(mkline.Value()) {
				mklines.buildDefs[varname] = true
				if G.opts.DebugMisc {
					mkline.Debug1("%q is added to BUILD_DEFS.", varname)
				}
			}

		case "PLIST_VARS":
			for _, id := range splitOnSpace(mkline.Value()) {
				mklines.plistVars["PLIST."+id] = true
				if G.opts.DebugMisc {
					mkline.Debug1("PLIST.%s is added to PLIST_VARS.", id)
				}
				mklines.UseVar(mkline, "PLIST."+id)
			}

		case "USE_TOOLS":
			tools := mkline.Value()
			if matches(tools, `\bautoconf213\b`) {
				tools += " autoconf autoheader-2.13 autom4te-2.13 autoreconf-2.13 autoscan-2.13 autoupdate-2.13 ifnames-2.13"
			}
			if matches(tools, `\bautoconf\b`) {
				tools += " autoheader autom4te autoreconf autoscan autoupdate ifnames"
			}
			for _, tool := range splitOnSpace(tools) {
				tool = strings.Split(tool, ":")[0]
				mklines.tools[tool] = true
				if G.opts.DebugMisc {
					mkline.Debug1("%s is added to USE_TOOLS.", tool)
				}
			}

		case "SUBST_VARS.*":
			for _, svar := range splitOnSpace(mkline.Value()) {
				mklines.UseVar(mkline, varnameCanon(svar))
				if G.opts.DebugMisc {
					mkline.Debug1("varuse %s", svar)
				}
			}

		case "OPSYSVARS":
			for _, osvar := range splitOnSpace(mkline.Value()) {
				mklines.UseVar(mkline, osvar+".*")
				defineVar(mkline, osvar)
			}
		}
	}
}

func (mklines *MkLines) DetermineUsedVariables() {
	for _, mkline := range mklines.mklines {
		varnames := mkline.determineUsedVariables()
		for _, varname := range varnames {
			mklines.UseVar(mkline, varname)
		}
	}
}

func (mklines *MkLines) checklineCond(mkline *MkLine) {
	indent, directive, args := mkline.Indent(), mkline.Directive(), mkline.Args()

	switch directive {
	case "endif", "endfor", "elif", "else":
		if len(mklines.indentation) > 1 {
			mklines.popIndent()
		} else {
			mkline.Error1("Unmatched .%s.", directive)
		}
	}

	// Check the indentation
	if expected := strings.Repeat(" ", mklines.indentDepth()); indent != expected {
		if G.opts.WarnSpace && !mkline.Line.AutofixReplace("."+indent, "."+expected) {
			mkline.Line.Notef("This directive should be indented by %d spaces.", mklines.indentDepth())
		}
	}

	if directive == "if" && matches(args, `^!defined\([\w]+_MK\)$`) {
		mklines.pushIndent(mklines.indentDepth())

	} else if matches(directive, `^(?:if|ifdef|ifndef|for|elif|else)$`) {
		mklines.pushIndent(mklines.indentDepth() + 2)
	}

	reDirectivesWithArgs := `^(?:if|ifdef|ifndef|elif|for|undef)$`
	if matches(directive, reDirectivesWithArgs) && args == "" {
		mkline.Error1("\".%s\" requires arguments.", directive)

	} else if !matches(directive, reDirectivesWithArgs) && args != "" {
		mkline.Error1("\".%s\" does not take arguments.", directive)

		if directive == "else" {
			mkline.Note0("If you meant \"else if\", use \".elif\".")
		}

	} else if directive == "if" || directive == "elif" {
		mkline.CheckCond()

	} else if directive == "ifdef" || directive == "ifndef" {
		if matches(args, `\s`) {
			mkline.Error1("The \".%s\" directive can only handle _one_ argument.", directive)
		} else {
			mkline.Line.Warnf("The \".%s\" directive is deprecated. Please use \".if %sdefined(%s)\" instead.",
				directive, ifelseStr(directive == "ifdef", "", "!"), args)
		}

	} else if directive == "for" {
		if m, vars, values := match2(args, `^(\S+(?:\s*\S+)*?)\s+in\s+(.*)$`); m {
			for _, forvar := range splitOnSpace(vars) {
				if !G.Infrastructure && hasPrefix(forvar, "_") {
					mkline.Warn1("Variable names starting with an underscore (%s) are reserved for internal pkgsrc use.", forvar)
				}

				if matches(forvar, `^[_a-z][_a-z0-9]*$`) {
					// Fine.
				} else if matches(forvar, `[A-Z]`) {
					mkline.Warn0(".for variable names should not contain uppercase letters.")
				} else {
					mkline.Error1("Invalid variable name %q.", forvar)
				}

				mklines.forVars[forvar] = true
			}

			// Check if any of the value's types is not guessed.
			guessed := true
			for _, value := range splitOnSpace(values) {
				if m, vname := match1(value, `^\$\{(.*)\}`); m {
					vartype := mkline.getVariableType(vname)
					if vartype != nil && !vartype.guessed {
						guessed = false
					}
				}
			}

			forLoopType := &Vartype{lkSpace, CheckvarUnchecked, []AclEntry{{"*", aclpAllRead}}, guessed}
			forLoopContext := &VarUseContext{forLoopType, vucTimeParse, vucQuotFor, vucExtentWord}
			for _, forLoopVar := range mkline.extractUsedVariables(values) {
				mkline.CheckVaruse(forLoopVar, "", forLoopContext)
			}
		}

	} else if directive == "undef" && args != "" {
		for _, uvar := range splitOnSpace(args) {
			if mklines.forVars[uvar] {
				mkline.Note0("Using \".undef\" after a \".for\" loop is unnecessary.")
			}
		}
	}
}

func (mklines *MkLines) checklineDependencyRule(mkline *MkLine, targets, dependencies string, allowedTargets map[string]bool) {
	if G.opts.DebugMisc {
		mkline.Debug2("targets=%q, dependencies=%q", targets, dependencies)
	}
	mklines.target = targets

	for _, source := range splitOnSpace(dependencies) {
		if source == ".PHONY" {
			for _, target := range splitOnSpace(targets) {
				allowedTargets[target] = true
			}
		}
	}

	for _, target := range splitOnSpace(targets) {
		if target == ".PHONY" {
			for _, dep := range splitOnSpace(dependencies) {
				allowedTargets[dep] = true
			}

		} else if target == ".ORDER" {
			// TODO: Check for spelling mistakes.

		} else if !allowedTargets[target] {
			mkline.Warn1("Unusual target %q.", target)
			Explain3(
				"If you want to define your own targets, you can \"declare\"",
				"them by inserting a \".PHONY: my-target\" line before this line.  This",
				"will tell make(1) to not interpret this target's name as a filename.")
		}
	}
}

type VaralignBlock struct {
	info []struct {
		mkline *MkLine
		prefix string
		align  string
	}
	skip           bool
	differ         bool
	maxPrefixWidth int
	maxSpaceWidth  int
	maxTabWidth    int
}

func (va *VaralignBlock) Check(mkline *MkLine) {
	if !G.opts.WarnSpace || mkline.Line.IsMultiline() || mkline.IsComment() || mkline.IsCond() {
		return
	}
	if mkline.IsEmpty() {
		va.Finish()
		return
	}
	if !mkline.IsVarassign() {
		va.skip = true
		return
	}

	valueAlign := mkline.ValueAlign()
	prefix := strings.TrimRight(valueAlign, " \t")
	align := valueAlign[len(prefix):]

	va.info = append(va.info, struct {
		mkline *MkLine
		prefix string
		align  string
	}{mkline, prefix, align})

	alignedWidth := tabLength(valueAlign)
	if contains(align, " ") {
		if va.maxSpaceWidth != 0 && alignedWidth != va.maxSpaceWidth {
			va.differ = true
		}
		if alignedWidth > va.maxSpaceWidth {
			va.maxSpaceWidth = alignedWidth
		}
	} else {
		if va.maxTabWidth != 0 && alignedWidth != va.maxTabWidth {
			va.differ = true
		}
		if alignedWidth > va.maxTabWidth {
			va.maxTabWidth = alignedWidth
		}
	}

	va.maxPrefixWidth = imax(va.maxPrefixWidth, tabLength(prefix))
}

func (va *VaralignBlock) Finish() {
	if !va.skip {
		for _, info := range va.info {
			if !info.mkline.Line.IsMultiline() {
				va.fixalign(info.mkline, info.prefix, info.align)
			}
		}
	}
	*va = VaralignBlock{}
}

func (va *VaralignBlock) fixalign(mkline *MkLine, prefix, oldalign string) {
	if mkline.Value() == "" && mkline.Comment() == "" {
		return
	}

	hasSpace := contains(oldalign, " ")
	if hasSpace &&
		va.maxTabWidth != 0 &&
		va.maxSpaceWidth > va.maxTabWidth &&
		tabLength(prefix+oldalign) == va.maxSpaceWidth {
		return
	}

	goodWidth := va.maxTabWidth
	if goodWidth == 0 && va.differ {
		goodWidth = va.maxSpaceWidth
	}
	minWidth := va.maxPrefixWidth + 1
	if goodWidth == 0 || minWidth < goodWidth && va.differ {
		goodWidth = minWidth
	}
	goodWidth = (goodWidth + 7) & -8

	newalign := ""
	for tabLength(prefix+newalign) < goodWidth {
		newalign += "\t"
	}
	if newalign == oldalign {
		return
	}

	if !mkline.Line.AutofixReplace(prefix+oldalign, prefix+newalign) {
		wrongColumn := tabLength(prefix+oldalign) != tabLength(prefix+newalign)
		switch {
		case hasSpace && wrongColumn:
			mkline.Line.Notef("This variable value should be aligned with tabs, not spaces, to column %d.", goodWidth+1)
		case hasSpace:
			mkline.Line.Notef("Variable values should be aligned with tabs, not spaces.")
		case wrongColumn:
			mkline.Line.Notef("This variable value should be aligned to column %d.", goodWidth+1)
		}
		if wrongColumn {
			Explain(
				"Normally, all variable values in a block should start at the same",
				"column.  There are some exceptions to this rule:",
				"",
				"Definitions for long variable names may be indented with a single",
				"space instead of tabs, but only if they appear in a block that is",
				"otherwise indented using tabs.",
				"",
				"Variable definitions that span multiple lines are not checked for",
				"alignment at all.",
				"",
				"When the block contains something else than variable definitions,",
				"it is not checked at all.")
		}
	}
}
