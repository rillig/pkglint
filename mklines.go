package main

import (
	"netbsd.org/pkglint/trace"
	"path"
	"strings"
)

// MkLines contains data for the Makefile (or *.mk) that is currently checked.
type MkLines struct {
	mklines        []MkLine
	lines          []Line
	forVars        map[string]bool   // The variables currently used in .for loops
	target         string            // Current make(1) target
	vardef         map[string]MkLine // varname => line; for all variables that are defined in the current file
	varuse         map[string]MkLine // varname => line; for all variables that are used in the current file
	buildDefs      map[string]bool   // Variables that are registered in BUILD_DEFS, to ensure that all user-defined variables are added to it.
	plistVars      map[string]bool   // Variables that are registered in PLIST_VARS, to ensure that all user-defined variables are added to it.
	tools          map[string]bool   // Set of tools that are declared to be used.
	toolRegistry   ToolRegistry      // Tools defined in file scope.
	SeenBsdPrefsMk bool
	indentation    Indentation // Indentation depth of preprocessing directives
}

func NewMkLines(lines []Line) *MkLines {
	mklines := make([]MkLine, len(lines))
	for i, line := range lines {
		mklines[i] = NewMkLine(line)
	}
	tools := make(map[string]bool)
	for toolname, tool := range G.globalData.Tools.byName {
		if tool.Predefined {
			tools[toolname] = true
		}
	}

	return &MkLines{
		mklines,
		lines,
		make(map[string]bool),
		"",
		make(map[string]MkLine),
		make(map[string]MkLine),
		make(map[string]bool),
		make(map[string]bool),
		tools,
		NewToolRegistry(),
		false,
		Indentation{}}
}

func (mklines *MkLines) DefineVar(mkline MkLine, varname string) {
	if mklines.vardef[varname] == nil {
		mklines.vardef[varname] = mkline
	}
	varcanon := varnameCanon(varname)
	if mklines.vardef[varcanon] == nil {
		mklines.vardef[varcanon] = mkline
	}
}

func (mklines *MkLines) UseVar(mkline MkLine, varname string) {
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
	if trace.Tracing {
		defer trace.Call1(mklines.lines[0].Filename)()
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
	mklines.DetermineDefinedVariables()

	// In the second pass, the actual checks are done.

	CheckLineRcsid(mklines.lines[0], `#\s+`, "# ")

	substcontext := NewSubstContext()
	var varalign VaralignBlock
	indentation := &mklines.indentation
	indentation.Push(0)
	for _, mkline := range mklines.mklines {
		ck := MkLineChecker{mkline}
		ck.Check()
		varalign.Check(mkline)

		switch {
		case mkline.IsEmpty():
			substcontext.Finish(mkline)

		case mkline.IsVarassign():
			mklines.target = ""
			substcontext.Varassign(mkline)

		case mkline.IsInclude():
			mklines.target = ""
			switch path.Base(mkline.Includefile()) {
			case "bsd.prefs.mk", "bsd.fast.prefs.mk", "bsd.builtin.mk":
				mklines.setSeenBsdPrefsMk()
			}
			if G.Pkg != nil {
				G.Pkg.CheckInclude(mkline, indentation)
			}

		case mkline.IsCond():
			ck.checkCond(mklines.forVars, indentation)
			substcontext.Conditional(mkline)

		case mkline.IsDependency():
			ck.checkDependencyRule(allowedTargets)
			mklines.target = mkline.Targets()
		}
	}
	lastMkline := mklines.mklines[len(mklines.mklines)-1]
	substcontext.Finish(lastMkline)
	varalign.Finish()

	ChecklinesTrailingEmptyLines(mklines.lines)

	if indentation.Len() != 1 && indentation.Depth() != 0 {
		lastMkline.Errorf("Directive indentation is not 0, but %d.", indentation.Depth())
	}

	SaveAutofixChanges(mklines.lines)
}

func (mklines *MkLines) DetermineDefinedVariables() {
	if trace.Tracing {
		defer trace.Call0()()
	}

	for _, mkline := range mklines.mklines {
		if !mkline.IsVarassign() {
			continue
		}

		varcanon := mkline.Varcanon()
		switch varcanon {
		case "BUILD_DEFS", "PKG_GROUPS_VARS", "PKG_USERS_VARS":
			for _, varname := range splitOnSpace(mkline.Value()) {
				mklines.buildDefs[varname] = true
				if trace.Tracing {
					trace.Step1("%q is added to BUILD_DEFS.", varname)
				}
			}

		case "PLIST_VARS":
			for _, id := range splitOnSpace(mkline.Value()) {
				mklines.plistVars["PLIST."+id] = true
				if trace.Tracing {
					trace.Step1("PLIST.%s is added to PLIST_VARS.", id)
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
				if trace.Tracing {
					trace.Step1("%s is added to USE_TOOLS.", tool)
				}
			}

		case "SUBST_VARS.*":
			for _, svar := range splitOnSpace(mkline.Value()) {
				mklines.UseVar(mkline, varnameCanon(svar))
				if trace.Tracing {
					trace.Step1("varuse %s", svar)
				}
			}

		case "OPSYSVARS":
			for _, osvar := range splitOnSpace(mkline.Value()) {
				mklines.UseVar(mkline, osvar+".*")
				defineVar(mkline, osvar)
			}
		}

		mklines.toolRegistry.ParseToolLine(mkline.Line)
	}
}

func (mklines *MkLines) DetermineUsedVariables() {
	for _, mkline := range mklines.mklines {
		varnames := mkline.DetermineUsedVariables()
		for _, varname := range varnames {
			mklines.UseVar(mkline, varname)
		}
	}
}

func (mklines *MkLines) setSeenBsdPrefsMk() {
	if !mklines.SeenBsdPrefsMk {
		mklines.SeenBsdPrefsMk = true
		if trace.Tracing {
			trace.Stepf("Mk.setSeenBsdPrefsMk")
		}
	}
}

// VaralignBlock checks that all variable assignments from a paragraph
// use the same indentation depth for their values.
// It also checks that the indentation uses tabs instead of spaces.
//
// In general, all values should be indented using tabs.
// As an exception, very long lines may be indented with a single space.
// A typical example is a SITES.very-long-filename.tar.gz variable
// between HOMEPAGE and DISTFILES.
type VaralignBlock struct {
	info           []*varalignBlockInfo
	skip           bool
	differ         bool
	maxPrefixWidth int
	maxSpaceWidth  int
	maxTabWidth    int
}

type varalignBlockInfo struct {
	mkline MkLine
	prefix string
	align  string
	ignore bool
}

func (va *VaralignBlock) Check(mkline MkLine) {
	if !G.opts.WarnSpace || mkline.IsComment() || mkline.IsCond() {
		return
	}
	if mkline.IsEmpty() {
		va.Finish()
		return
	}
	if !mkline.IsVarassign() || mkline.IsMultiline() {
		trace.Stepf("Skipping")
		va.skip = true
		return
	}

	if trace.Tracing {
		defer trace.Call(mkline)()
	}

	valueAlign := mkline.ValueAlign()
	prefix := strings.TrimRight(valueAlign, " \t")
	align := valueAlign[len(prefix):]

	va.info = append(va.info, &varalignBlockInfo{mkline, prefix, align, false})

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
	trace.Stepf("%+v", va)
}

func (va *VaralignBlock) Finish() {
	if !va.skip {
		if trace.Tracing {
			defer trace.Call0()()
		}

		width := va.calculateOptimalWidth()

		if width != 0 {
			for _, info := range va.info {
				if !(info.align == " " && tabLength(info.mkline.ValueAlign()) > width) {
					if !(info.mkline.Op() == opAssignEval && matches(info.mkline.Varname(), `^[a-z]`)) {
						va.fixalign(info.mkline, info.prefix, info.align, width)
					}
				}
			}
		}
	}
	*va = VaralignBlock{}
}

func (va *VaralignBlock) calculateOptimalWidth() int {
	maxSingleSpaceAlignWidth := 0
	for _, info := range va.info {
		if info.align == " " {
			maxSingleSpaceAlignWidth = imax(maxSingleSpaceAlignWidth, tabLength(info.mkline.ValueAlign()))
		}
	}

	minSpaceAlignWidth := 0
	maxSpaceAlignWidth := 0
	minTabAlignWidth := 0
	maxTabAlignWidth := 0

	maxPrefixWidth := 0
	for _, info := range va.info {
		if info.mkline.Value() == "" && info.mkline.VarassignComment() == "" {
			continue
		}
		if info.align == " " && tabLength(info.mkline.ValueAlign()) == maxSingleSpaceAlignWidth {
			continue
		}
		if info.mkline.Op() == opAssignEval && matches(info.mkline.Varname(), `^[a-z]`) {
			continue
		}
		maxPrefixWidth = imax(maxPrefixWidth, tabLength(info.prefix))
		if contains(info.align, " ") {
			maxSpaceAlignWidth = imax(maxSpaceAlignWidth, tabLength(info.mkline.ValueAlign()))
			if minSpaceAlignWidth == 0 || tabLength(info.mkline.ValueAlign()) < minSpaceAlignWidth {
				minSpaceAlignWidth = tabLength(info.mkline.ValueAlign())
			}
		} else {
			maxTabAlignWidth = imax(maxTabAlignWidth, tabLength(info.mkline.ValueAlign()))
			if minTabAlignWidth == 0 || tabLength(info.mkline.ValueAlign()) < minTabAlignWidth {
				minTabAlignWidth = tabLength(info.mkline.ValueAlign())
			}
		}
	}
	if maxPrefixWidth == 0 {
		return 0
	}
	if len(va.info) == 1 && va.info[0].align == "" {
		return 0
	}

	// Test_MkLines__alignment_space
	if maxSpaceAlignWidth == 0 && minTabAlignWidth == maxTabAlignWidth && minTabAlignWidth > maxPrefixWidth &&
		(maxSingleSpaceAlignWidth == 0 || maxSingleSpaceAlignWidth >= maxPrefixWidth+4) {
		return 0
	}

	maxPrefixWidth = (maxPrefixWidth + 7) & -8

	if maxSingleSpaceAlignWidth != 0 &&
		maxPrefixWidth <= maxSingleSpaceAlignWidth &&
		maxSingleSpaceAlignWidth < maxPrefixWidth+4 {

		maxPrefixWidth += 8
	}

	return maxPrefixWidth
}

func (va *VaralignBlock) fixalign(mkline MkLine, prefix, oldalign string, goodWidth int) {
	hasSpace := contains(oldalign, " ")

	newalign := ""
	for tabLength(prefix+newalign) < goodWidth {
		newalign += "\t"
	}
	if newalign == oldalign {
		return
	}

	fix := mkline.Line.Autofix()
	wrongColumn := tabLength(prefix+oldalign) != tabLength(prefix+newalign)
	switch {
	case hasSpace && wrongColumn:
		fix.Notef("This variable value should be aligned with tabs, not spaces, to column %d.", goodWidth+1)
	case hasSpace:
		fix.Notef("Variable values should be aligned with tabs, not spaces.")
	case wrongColumn:
		fix.Notef("This variable value should be aligned to column %d.", goodWidth+1)
	}
	if wrongColumn {
		fix.Explain(
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
			"When the block contains something else than variable definitions",
			"and conditionals, it is not checked at all.")
	}
	fix.Replace(prefix+oldalign, prefix+newalign)
	fix.Apply()
}
