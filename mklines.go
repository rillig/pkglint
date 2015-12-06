package main

import (
	"path"
	"strings"
)

type MkLines struct {
	mklines []*MkLine
	lines   []*Line
}

func NewMkLines(lines []*Line) *MkLines {
	mklines := make([]*MkLine, len(lines))
	for i, line := range lines {
		parselineMk(line)
		mklines[i] = NewMkLine(line)
	}
	return &MkLines{mklines, lines}
}

func (mklines *MkLines) check() {
	defer tracecall("MkLines.check", mklines.lines[0].fname)()

	allowedTargets := make(map[string]bool)
	substcontext := new(SubstContext)

	ctx := newMkContext()
	G.mkContext = ctx
	defer func() { G.mkContext = nil }()

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
		text := mkline.text

		checklineTrailingWhitespace(mkline.Line)

		if mkline.extra["is_empty"] != nil {
			substcontext.Finish(mkline)

		} else if mkline.extra["is_comment"] != nil {
			// No further checks.

		} else if mkline.extra["is_varassign"] != nil {
			mkline.checkVaralign()
			mkline.checkVarassign()
			substcontext.Varassign(mkline)

		} else if hasPrefix(text, "\t") {
			shellcmd := text[1:]
			mkline.checkText(shellcmd)
			NewMkShellLine(mkline).checkShelltext(shellcmd)

		} else if m, include, includefile := match2(text, reMkInclude); m {
			mklines.checklineInclude(mkline, includefile, include == "include")

		} else if matches(text, reMkSysinclude) {

		} else if m, indent, directive, args := match3(text, reMkCond); m {
			mklines.checklinePreproc(ctx, mkline, indent, directive, args)

		} else if m, targets, _, dependencies := match3(text, reMkDependency); m {
			mklines.checklineDependencyRule(ctx,mkline,targets,dependencies,allowedTargets)

		} else if m, directive := match1(text, `^\.\s*(\S*)`); m {
			mkline.errorf("Unknown directive \".%s\".", directive)

		} else if hasPrefix(text, " ") {
			mkline.warnf("Makefile lines should not start with space characters.")
			mkline.explain(
				"If you want this line to contain a shell program, use a tab",
				"character for indentation. Otherwise please remove the leading",
				"white-space.")

		} else {
			_ = G.opts.DebugMisc && mkline.debugf("Unknown line format")
		}
	}
	lastMkline := mklines.mklines[len(mklines.mklines)-1]
	substcontext.Finish(lastMkline)

	checklinesTrailingEmptyLines(mklines.lines)

	if len(ctx.indentation) != 1 {
		lastMkline.errorf("Directive indentation is not 0, but %d.", ctx.indentDepth())
	}

	G.mkContext = nil
}

func (mklines *MkLines) determineDefinedVariables() {
	for _, mkline := range mklines.mklines {
		if mkline.extra["is_varassign"] == nil {
			continue
		}

		varcanon := mkline.extra["varcanon"].(string)
		switch varcanon {
		case "BUILD_DEFS", "PKG_GROUPS_VARS", "PKG_USERS_VARS":
			for _, varname := range splitOnSpace(mkline.extra["value"].(string)) {
				G.mkContext.buildDefs[varname] = true
				_ = G.opts.DebugMisc && mkline.debugf("%q is added to BUILD_DEFS.", varname)
			}

		case "PLIST_VARS":
			for _, id := range splitOnSpace(mkline.extra["value"].(string)) {
				G.mkContext.plistVars["PLIST."+id] = true
				_ = G.opts.DebugMisc && mkline.debugf("PLIST.%s is added to PLIST_VARS.", id)
				useVar(mkline, "PLIST."+id)
			}

		case "USE_TOOLS":
			for _, tool := range splitOnSpace(mkline.extra["value"].(string)) {
				tool = strings.Split(tool, ":")[0]
				G.mkContext.tools[tool] = true
				_ = G.opts.DebugMisc && mkline.debugf("%s is added to USE_TOOLS.", tool)
			}

		case "SUBST_VARS.*":
			for _, svar := range splitOnSpace(mkline.extra["value"].(string)) {
				useVar(mkline, varnameCanon(svar))
				_ = G.opts.DebugMisc && mkline.debugf("varuse %s", svar)
			}

		case "OPSYSVARS":
			for _, osvar := range splitOnSpace(mkline.extra["value"].(string)) {
				useVar(mkline, osvar+".*")
				defineVar(mkline, osvar)
			}
		}
	}
}

func (mklines *MkLines) determineUsedVariables() {
	re := regcomp(`(?:\$\{|\$\(|defined\(|empty\()([0-9+.A-Z_a-z]+)[:})]`)
	for _, mkline := range mklines.mklines {
		rest := mkline.text
		for {
			m := re.FindStringSubmatchIndex(rest)
			if m == nil {
				break
			}
			varname := rest[m[2]:m[3]]
			useVar(mkline, varname)
			rest = rest[:m[0]] + rest[m[1]:]
		}
	}
}

func (mklines *MkLines) checklinePreproc(ctx *MkContext, mkline *MkLine, indent, directive, args string) {
	if matches(directive, `^(?:endif|endfor|elif|else)$`) {
		if len(ctx.indentation) > 1 {
			ctx.popIndent()
		} else {
			mkline.errorf("Unmatched .%s.", directive)
		}
	}

	// Check the indentation
	if indent != strings.Repeat(" ", ctx.indentDepth()) {
		_ = G.opts.WarnSpace && mkline.notef("This directive should be indented by %d spaces.", ctx.indentDepth())
	}

	if directive == "if" && matches(args, `^!defined\([\w]+_MK\)$`) {
		ctx.pushIndent(ctx.indentDepth())

	} else if matches(directive, `^(?:if|ifdef|ifndef|for|elif|else)$`) {
		ctx.pushIndent(ctx.indentDepth() + 2)
	}

	reDirectivesWithArgs := `^(?:if|ifdef|ifndef|elif|for|undef)$`
	if matches(directive, reDirectivesWithArgs) && args == "" {
		mkline.errorf("\".%s\" requires arguments.", directive)

	} else if !matches(directive, reDirectivesWithArgs) && args != "" {
		mkline.errorf("\".%s\" does not take arguments.", directive)

		if directive == "else" {
			mkline.notef("If you meant \"else if\", use \".elif\".")
		}

	} else if directive == "if" || directive == "elif" {
		mkline.checkIf()

	} else if directive == "ifdef" || directive == "ifndef" {
		if matches(args, `\s`) {
			mkline.errorf("The \".%s\" directive can only handle _one_ argument.", directive)
		} else {
			mkline.warnf("The \".%s\" directive is deprecated. Please use \".if %sdefined(%s)\" instead.",
				directive, ifelseStr(directive == "ifdef", "", "!"), args)
		}

	} else if directive == "for" {
		if m, vars, values := match2(args, `^(\S+(?:\s*\S+)*?)\s+in\s+(.*)$`); m {
			for _, forvar := range splitOnSpace(vars) {
				if !G.isInfrastructure && hasPrefix(forvar, "_") {
					mkline.warnf("Variable names starting with an underscore (%s) are reserved for internal pkgsrc use.", forvar)
				}

				if matches(forvar, `^[_a-z][_a-z0-9]*$`) {
					// Fine.
				} else if matches(forvar, `[A-Z]`) {
					mkline.warnf(".for variable names should not contain uppercase letters.")
				} else {
					mkline.errorf("Invalid variable name %q.", forvar)
				}

				ctx.forVars[forvar] = true
			}

			// Check if any of the value's types is not guessed.
			guessed := guGuessed
			for _, value := range splitOnSpace(values) {
				if m, vname := match1(value, `^\$\{(.*)\}`); m {
					vartype := getVariableType(mkline.Line, vname)
					if vartype != nil && !vartype.guessed {
						guessed = guNotGuessed
					}
				}
			}

			forLoopType := &Vartype{lkSpace, CheckvarUnchecked, []AclEntry{{"*", "pu"}}, guessed}
			forLoopContext := &VarUseContext{
				vucTimeParse,
				forLoopType,
				vucQuotFor,
				vucExtentWord,
			}
			for _, fvar := range extractUsedVariables(mkline.Line, values) {
				mkline.checkVaruse(fvar, "", forLoopContext)
			}
		}

	} else if directive == "undef" && args != "" {
		for _, uvar := range splitOnSpace(args) {
			if ctx.forVars[uvar] {
				mkline.notef("Using \".undef\" after a \".for\" loop is unnecessary.")
			}
		}
	}
}

func (mklines *MkLines) checklineDependencyRule(ctx*MkContext,mkline *MkLine,targets,dependencies string, allowedTargets map[string]bool) {
	_ = G.opts.DebugMisc && mkline.debugf("targets=%q, dependencies=%q", targets, dependencies)
	ctx.target = targets

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
			mkline.warnf("Unusual target %q.", target)
			mkline.explain(
				"If you want to define your own targets, you can \"declare\"",
				"them by inserting a \".PHONY: my-target\" line before this line. This",
				"will tell make(1) to not interpret this target's name as a filename.")
		}
	}
}

func (mklines *MkLines) checklineInclude(mkline *MkLine, includefile string, mustExist bool) {
	_ = G.opts.DebugInclude && mkline.debugf("includefile=%s", includefile)
	mkline.checkRelativePath(includefile, mustExist)

	if hasSuffix(includefile, "../Makefile") {
		mkline.errorf("Other Makefiles must not be included directly.")
		mkline.explain(
			"If you want to include portions of another Makefile, extract",
			"the common parts and put them into a Makefile.common. After",
			"that, both this one and the other package should include the",
			"Makefile.common.")
	}

	if includefile == "../../mk/bsd.prefs.mk" {
		if path.Base(mkline.fname) == "buildlink3.mk" {
			mkline.notef("For efficiency reasons, please include bsd.fast.prefs.mk instead of bsd.prefs.mk.")
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
		mkline.errorf("%s must not be included directly. Include \"../../mk/x11.buildlink3.mk\" instead.", includefile)
	}
	if matches(includefile, `/jpeg/buildlink3\.mk$`) {
		mkline.errorf("%s must not be included directly. Include \"../../mk/jpeg.buildlink3.mk\" instead.", includefile)
	}
	if matches(includefile, `/intltool/buildlink3\.mk$`) {
		mkline.warnf("Please write \"USE_TOOLS+= intltool\" instead of this line.")
	}
	if m, dir := match1(includefile, `(.*)/builtin\.mk$`); m {
		mkline.errorf("%s must not be included directly. Include \"%s/buildlink3.mk\" instead.", includefile, dir)
	}
}
