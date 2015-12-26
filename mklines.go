package main

import (
	"path"
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
	for tool := range G.globalData.predefinedTools {
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

func (mklines *MkLines) defineVar(mkline *MkLine, varname string) {
	if mklines.vardef[varname] == nil {
		mklines.vardef[varname] = mkline
	}
	varcanon := varnameCanon(varname)
	if mklines.vardef[varcanon] == nil {
		mklines.vardef[varcanon] = mkline
	}
}

func (mklines *MkLines) useVar(mkline *MkLine, varname string) {
	varcanon := varnameCanon(varname)
	mklines.varuse[varname] = mkline
	mklines.varuse[varcanon] = mkline
	if G.pkg != nil {
		G.pkg.varuse[varname] = mkline
		G.pkg.varuse[varcanon] = mkline
	}
}

func (mklines *MkLines) varValue(varname string) (value string, found bool) {
	if mkline := mklines.vardef[varname]; mkline != nil {
		return mkline.Value(), true
	}
	return "", false
}

func (mklines *MkLines) check() {
	defer tracecall1("MkLines.check", mklines.lines[0].fname)()

	allowedTargets := make(map[string]bool)
	substcontext := new(SubstContext)

	G.mk = mklines
	defer func() { G.mk = nil }()

	mklines.determineUsedVariables()

	prefixes := splitOnSpace("pre do post")
	actions := splitOnSpace("fetch extract patch tools wrapper configure build test install package clean")
	for _, prefix := range prefixes {
		for _, action := range actions {
			allowedTargets[prefix+"-"+action] = true
		}
	}

	// In the first pass, all additions to BUILD_DEFS and USE_TOOLS
	// are collected to make the order of the definitions irrelevant.
	mklines.determineDefinedVariables()

	// In the second pass, the actual checks are done.

	checklineRcsid(mklines.lines[0], `#\s+`, "# ")

	for _, mkline := range mklines.mklines {
		text := mkline.line.text

		checklineTrailingWhitespace(mkline.line)

		switch {
		case mkline.IsEmpty():
			substcontext.Finish(mkline)

		case mkline.IsVarassign():
			mkline.checkVaralign()
			mkline.checkVarassign()
			substcontext.Varassign(mkline)

		case mkline.IsShellcmd():
			shellcmd := text[1:]
			mkline.checkText(shellcmd)
			NewMkShellLine(mkline).checkShellCommandLine(shellcmd)

		case mkline.IsInclude():
			mklines.checklineInclude(mkline)

		case mkline.IsCond():
			mklines.checklineCond(mkline)

		case mkline.IsDependency():
			mklines.checklineDependencyRule(mkline, mkline.Targets(), mkline.Sources(), allowedTargets)
		}
	}
	lastMkline := mklines.mklines[len(mklines.mklines)-1]
	substcontext.Finish(lastMkline)

	checklinesTrailingEmptyLines(mklines.lines)

	if len(mklines.indentation) != 1 {
		lastMkline.line.errorf("Directive indentation is not 0, but %d.", mklines.indentDepth())
	}
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
					mkline.debug1("%q is added to BUILD_DEFS.", varname)
				}
			}

		case "PLIST_VARS":
			for _, id := range splitOnSpace(mkline.Value()) {
				mklines.plistVars["PLIST."+id] = true
				if G.opts.DebugMisc {
					mkline.debug1("PLIST.%s is added to PLIST_VARS.", id)
				}
				mklines.useVar(mkline, "PLIST."+id)
			}

		case "USE_TOOLS":
			for _, tool := range splitOnSpace(mkline.Value()) {
				tool = strings.Split(tool, ":")[0]
				mklines.tools[tool] = true
				if G.opts.DebugMisc {
					mkline.debug1("%s is added to USE_TOOLS.", tool)
				}
			}

		case "SUBST_VARS.*":
			for _, svar := range splitOnSpace(mkline.Value()) {
				mklines.useVar(mkline, varnameCanon(svar))
				if G.opts.DebugMisc {
					mkline.debug1("varuse %s", svar)
				}
			}

		case "OPSYSVARS":
			for _, osvar := range splitOnSpace(mkline.Value()) {
				mklines.useVar(mkline, osvar+".*")
				defineVar(mkline, osvar)
			}
		}
	}
}

func (mklines *MkLines) determineUsedVariables() {
	re := regcomp(`(?:\$\{|\$\(|defined\(|empty\()([0-9+.A-Z_a-z]+)[:})]`)
	for _, mkline := range mklines.mklines {
		rest := mkline.line.text
		for {
			m := re.FindStringSubmatchIndex(rest)
			if m == nil {
				break
			}
			varname := rest[m[2]:m[3]]
			mklines.useVar(mkline, varname)
			rest = rest[:m[0]] + rest[m[1]:]
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
			mkline.error1("Unmatched .%s.", directive)
		}
	}

	// Check the indentation
	if indent != strings.Repeat(" ", mklines.indentDepth()) {
		if G.opts.WarnSpace {
			mkline.line.notef("This directive should be indented by %d spaces.", mklines.indentDepth())
		}
	}

	if directive == "if" && matches(args, `^!defined\([\w]+_MK\)$`) {
		mklines.pushIndent(mklines.indentDepth())

	} else if matches(directive, `^(?:if|ifdef|ifndef|for|elif|else)$`) {
		mklines.pushIndent(mklines.indentDepth() + 2)
	}

	reDirectivesWithArgs := `^(?:if|ifdef|ifndef|elif|for|undef)$`
	if matches(directive, reDirectivesWithArgs) && args == "" {
		mkline.error1("\".%s\" requires arguments.", directive)

	} else if !matches(directive, reDirectivesWithArgs) && args != "" {
		mkline.error1("\".%s\" does not take arguments.", directive)

		if directive == "else" {
			mkline.note0("If you meant \"else if\", use \".elif\".")
		}

	} else if directive == "if" || directive == "elif" {
		mkline.checkIf()

	} else if directive == "ifdef" || directive == "ifndef" {
		if matches(args, `\s`) {
			mkline.error1("The \".%s\" directive can only handle _one_ argument.", directive)
		} else {
			mkline.line.warnf("The \".%s\" directive is deprecated. Please use \".if %sdefined(%s)\" instead.",
				directive, ifelseStr(directive == "ifdef", "", "!"), args)
		}

	} else if directive == "for" {
		if m, vars, values := match2(args, `^(\S+(?:\s*\S+)*?)\s+in\s+(.*)$`); m {
			for _, forvar := range splitOnSpace(vars) {
				if !G.isInfrastructure && hasPrefix(forvar, "_") {
					mkline.warn1("Variable names starting with an underscore (%s) are reserved for internal pkgsrc use.", forvar)
				}

				if matches(forvar, `^[_a-z][_a-z0-9]*$`) {
					// Fine.
				} else if matches(forvar, `[A-Z]`) {
					mkline.warn0(".for variable names should not contain uppercase letters.")
				} else {
					mkline.error1("Invalid variable name %q.", forvar)
				}

				mklines.forVars[forvar] = true
			}

			// Check if any of the value's types is not guessed.
			guessed := guGuessed
			for _, value := range splitOnSpace(values) {
				if m, vname := match1(value, `^\$\{(.*)\}`); m {
					vartype := getVariableType(mkline.line, vname)
					if vartype != nil && !vartype.guessed {
						guessed = guNotGuessed
					}
				}
			}

			forLoopType := &Vartype{lkSpace, CheckvarUnchecked, []AclEntry{{"*", aclpUseLoadtime | aclpUse}}, guessed}
			forLoopContext := &VarUseContext{forLoopType, vucTimeParse, vucQuotFor, vucExtentWord}
			for _, forLoopVar := range extractUsedVariables(mkline.line, values) {
				mkline.checkVaruse(forLoopVar, "", forLoopContext)
			}
		}

	} else if directive == "undef" && args != "" {
		for _, uvar := range splitOnSpace(args) {
			if mklines.forVars[uvar] {
				mkline.note0("Using \".undef\" after a \".for\" loop is unnecessary.")
			}
		}
	}
}

func (mklines *MkLines) checklineDependencyRule(mkline *MkLine, targets, dependencies string, allowedTargets map[string]bool) {
	if G.opts.DebugMisc {
		mkline.debug2("targets=%q, dependencies=%q", targets, dependencies)
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
			mkline.warn1("Unusual target %q.", target)
			explain3(
				"If you want to define your own targets, you can \"declare\"",
				"them by inserting a \".PHONY: my-target\" line before this line. This",
				"will tell make(1) to not interpret this target's name as a filename.")
		}
	}
}

func (mklines *MkLines) checklineInclude(mkline *MkLine) {
	includefile := mkline.Includefile()
	mustExist := mkline.MustExist()
	if G.opts.DebugInclude {
		mkline.debug1("includefile=%s", includefile)
	}
	mkline.checkRelativePath(includefile, mustExist)

	if hasSuffix(includefile, "/Makefile") {
		mkline.line.error0("Other Makefiles must not be included directly.")
		explain4(
			"If you want to include portions of another Makefile, extract",
			"the common parts and put them into a Makefile.common. After",
			"that, both this one and the other package should include the",
			"Makefile.common.")
	}

	if includefile == "../../mk/bsd.prefs.mk" {
		if path.Base(mkline.line.fname) == "buildlink3.mk" {
			mkline.note0("For efficiency reasons, please include bsd.fast.prefs.mk instead of bsd.prefs.mk.")
		}
		if G.pkg != nil {
			G.pkg.seenBsdPrefsMk = true
		}
	} else if includefile == "../../mk/bsd.fast.prefs.mk" {
		if G.pkg != nil {
			G.pkg.seenBsdPrefsMk = true
		}
	}

	if matches(includefile, `/x11-links/buildlink3\.mk$`) {
		mkline.error1("%s must not be included directly. Include \"../../mk/x11.buildlink3.mk\" instead.", includefile)
	}
	if matches(includefile, `/jpeg/buildlink3\.mk$`) {
		mkline.error1("%s must not be included directly. Include \"../../mk/jpeg.buildlink3.mk\" instead.", includefile)
	}
	if matches(includefile, `/intltool/buildlink3\.mk$`) {
		mkline.warn0("Please write \"USE_TOOLS+= intltool\" instead of this line.")
	}
	if m, dir := match1(includefile, `(.*)/builtin\.mk$`); m {
		mkline.line.error2("%s must not be included directly. Include \"%s/buildlink3.mk\" instead.", includefile, dir)
	}
}
