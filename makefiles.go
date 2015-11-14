package main

import (
	"path"
	"strings"
)

func readMakefile(fname string, mainLines *[]*Line, allLines *[]*Line) bool {
	fileLines, err := loadLines(fname, true)
	if err != nil {
		logError(fname, NO_LINES, "Cannot be read")
		return false
	}

	parselinesMk(fileLines)
	isMainMakefile := len(*mainLines) == 0

	for _, line := range fileLines {
		text := line.text

		if isMainMakefile {
			*mainLines = append(*mainLines, line)
		}
		*allLines = append(*allLines, line)

		isIncludeLine := false
		includeFile := ""
		if m, inc := match1(text, `^\.\s*include\s+\"(.*)\"$`); m {
			includeFile = resolveVarsInRelativePath(inc, true)
			if matches(includeFile, reUnresolvedVar) {
				if !contains(fname, "/mk/") {
					line.logNote("Skipping include file %q. This may result in false warnings.", includeFile)
				}
			} else {
				isIncludeLine = true
			}
		}

		if isIncludeLine {
			if path.Base(fname) == "buildlink3.mk" {
				if m, bl3File := match1(includeFile, `^\.\./\.\./(.*)/buildlink3\.mk$`); m {
					G.pkgContext.bl3[bl3File] = line
					_ = G.opts.optDebugMisc && line.logDebug("Buildlink3 file in package: %v", bl3File)
				}
			}
		}

		if isIncludeLine && G.pkgContext.included[includeFile] == nil {
			G.pkgContext.included[includeFile] = line

			if matches(includeFile, `^\.\./[^./][^/]*/[^/]+`) {
				line.logWarning("References to other packages should look like \"../../category/package\", not \"../package\".")
				line.explainWarning(explanationRelativeDirs()...)
			}
			if path.Base(includeFile) == "Makefile.common" {
				_ = G.opts.optDebugInclude && line.logDebug("Including %q sets seenMakefileCommon.", includeFile)
				G.pkgContext.seenMakefileCommon = true
			}
			if m, _, mkfile := match2(includeFile, `^(?:\.\./(\.\./[^/]+/)?[^/]+/)?([^/]+)$`); m {
				if mkfile != "buildlink3.mk" && mkfile != "options.mk" {
					_ = G.opts.optDebugInclude && line.logDebug("Including %q sets seenMakefileCommon.", includeFile)
					G.pkgContext.seenMakefileCommon = true
				}
			}

			if !contains(includeFile, "/mk/") {
				dirname := path.Dir(fname)

				// Only look in the directory relative to the
				// current file and in the current working directory.
				// We don't have an include dir list, like make(1) does.
				if !fileExists(dirname + "/" + includeFile) {
					dirname = G.currentDir
				}
				if !fileExists(dirname + "/" + includeFile) {
					line.logError("Cannot read %q.", dirname+"/"+includeFile)
					return false
				} else {
					_ = G.opts.optDebugInclude && line.logDebug("Including %q.", dirname+"/"+includeFile)
					lengthBeforeInclude := len(*allLines)
					if !readMakefile(dirname+"/"+includeFile, mainLines, allLines) {
						return false
					}

					if path.Base(includeFile) == "Makefile.common" {
						makefileCommonLines := (*allLines)[lengthBeforeInclude:]
						checkForUsedComment(makefileCommonLines, relpath(G.globalData.pkgsrcdir, fname))
					}
				}
			}
		}

		if line.extra["is_varassign"] != nil {
			varname, op, value := line.extra["varname"].(string), line.extra["op"].(string), line.extra["value"].(string)

			if op != "?=" || G.pkgContext.vardef[varname] == nil {
				_ = G.opts.optDebugMisc && line.logDebug("varassign(%q, %q, %q)", varname, op, value)
				G.pkgContext.vardef[varname] = line
			}
		}
	}

	return true
}

func checkForUsedComment(lines []*Line, relativeName string) {
	expected := "# used by " + relativeName

	for _, line := range lines {
		if line.text == expected {
			return
		}
	}

	lastCommentLine := 0
	for i, line := range lines {
		if m, _ := match1(line.text, reMkComment); m {
			break
		}
		lastCommentLine = i
	}

	insertLine := lines[lastCommentLine+1]
	insertLine.logWarning("Please add a line %q here.", expected)
	insertLine.explainWarning(
		`Since Makefile.common files usually don't have any comments and
therefore not a clearly defined interface, they should at least contain
references to all files that include them, so that it is easier to see
what effects future changes may have.

If there are more than five packages that use a Makefile.common,
you should think about giving it a proper name (maybe plugin.mk) and
documenting its interface.`)
	insertLine.appendBefore(expected)
	if G.opts.optAutofix {
		saveAutofixChanges(lines)
	}
}

func resolveVarsInRelativePath(relpath string, adjustDepth bool) string {

	tmp := relpath
	tmp = strings.Replace(tmp, "${PKGSRCDIR}", *G.curPkgsrcdir, -1)
	tmp = strings.Replace(tmp, "${.CURDIR}", ".", -1)
	tmp = strings.Replace(tmp, "${.PARSEDIR}", ".", -1)
	tmp = strings.Replace(tmp, "${LUA_PKGSRCDIR}", "../../lang/lua52", -1)
	tmp = strings.Replace(tmp, "${PHPPKGSRCDIR}", "../../lang/php54", -1)
	tmp = strings.Replace(tmp, "${SUSE_DIR_PREFIX}", "suse100", -1)
	tmp = strings.Replace(tmp, "${PYPKGSRCDIR}", "../../lang/python27", -1)
	if G.pkgContext != nil {
		tmp = strings.Replace(tmp, "${FILESDIR}", G.pkgContext.filesdir, -1)
	}
	if G.pkgContext != nil && G.pkgContext.pkgdir != nil {
		tmp = strings.Replace(tmp, "${PKGDIR}", *G.pkgContext.pkgdir, -1)
	}

	if adjustDepth {
		if m, pkgpath := match1(tmp, `^\.\./\.\./([^.].*)$`); m {
			tmp = *G.curPkgsrcdir + "/" + pkgpath
		}
	}

	_ = G.opts.optDebugMisc && logDebug(NO_FILE, NO_LINES, "resolveVarsInRelativePath: %q => %q", relpath, tmp)
	return tmp
}

func parselineMk(line *Line) {
	text := line.text

	if m, varname, op, value, comment := match4(text, reVarassign); m {

		// In variable assignments, a '#' character is preceded
		// by a backslash. In shell commands, it is interpreted
		// literally.
		value = strings.Replace(value, "\\#", "#", -1)
		varparam := varnameParam(varname)

		line.extra["is_varassign"] = true
		line.extra["varname"] = varname
		line.extra["varcanon"] = varnameCanon(varname)
		line.extra["varparam"] = varparam
		line.extra["op"] = op
		line.extra["value"] = value
		line.extra["comment"] = comment
		return
	}

	if m, shellcmd := match1(text, reMkShellcmd); m {
		line.extra["is_shellcmd"] = true
		line.extra["shellcmd"] = shellcmd
		return
	}

	if m, comment := match1(text, reMkComment); m {
		line.extra["is_comment"] = true
		line.extra["comment"] = comment
		return
	}

	if matches(text, `^\s*$`) {
		line.extra["is_empty"] = true
		return
	}

	if m, indent, directive, args := match3(text, reMkCond); m {
		line.extra["is_cond"] = true
		line.extra["indent"] = indent
		line.extra["directive"] = directive
		line.extra["args"] = args
		return
	}

	if m, _, includefile := match2(text, reMkInclude); m {
		line.extra["is_include"] = true
		line.extra["includefile"] = includefile
		return
	}

	if m, includefile, comment := match2(text, reMkSysinclude); m {
		line.extra["is_sysinclude"] = true
		line.extra["includefile"] = includefile
		line.extra["comment"] = comment
		return
	}

	if m, targets, whitespace, sources := match3(text, reMkDependency); m {
		line.extra["is_dependency"] = true
		line.extra["targets"] = targets
		line.extra["sources"] = sources
		if whitespace != "" {
			line.logWarning("Space before colon in dependency line.")
		}
		return
	}

	if matches(text, reConflict) {
		return
	}

	line.logFatal("Unknown Makefile line format.")
}

func parselinesMk(lines []*Line) {
	for _, line := range lines {
		parselineMk(line)
	}
}

func checklineMkText(line *Line, text string) {
	defer tracecall("checklineMkText", text)()

	if m, varname := match1(text, `^(?:[^#]*[^\$])?\$(\w+)`); m {
		line.logWarning("$%s is ambiguous. Use ${%s} if you mean a Makefile variable or $$%s if you mean a shell variable.", varname, varname, varname)
	}

	if line.lines == "1" {
		checklineRcsid(line, `# `, "# ")
	}

	if contains(text, "${WRKSRC}/../") {
		line.logWarning("Using \"${WRKSRC}/..\" is conceptually wrong. Please use a combination of WRKSRC, CONFIGURE_DIRS and BUILD_DIRS instead.")
		line.explainWarning(
			"You should define WRKSRC such that all of CONFIGURE_DIRS, BUILD_DIRS",
			"and INSTALL_DIRS are subdirectories of it.")
	}

	// Note: A simple -R is not detected, as the rate of false positives is too high.
	if m, flag := match1(text, `\b(-Wl,--rpath,|-Wl,-rpath-link,|-Wl,-rpath,|-Wl,-R)\b`); m {
		line.logWarning("Please use ${COMPILER_RPATH_FLAG} instead of %q.", flag)
	}

	rest := text
	for {
		m, r := replaceFirst(rest, `(?:^|[^$])\$\{([-A-Z0-9a-z_]+)(\.[\-0-9A-Z_a-z]+)?(?::[^\}]+)?\}`, "")
		if m == nil {
			break
		}
		rest = r

		varbase, varext := m[1], m[2]
		varname := varbase + varext
		varcanon := varnameCanon(varname)
		instead := G.globalData.deprecated[varname]
		if instead == "" {
			instead = G.globalData.deprecated[varcanon]
		}
		if instead != "" {
			line.logWarning("Use of %q is deprecated. %s", varname, instead)
		}
	}
}

func checklinesMk(lines []*Line) {
	defer tracecall("checklinesMk", lines[0].fname)()

	allowedTargets := make(map[string]bool)
	substcontext := &SubstContext{}

	ctx := newMkContext()
	G.mkContext = ctx
	defer func() { G.mkContext = nil }()

	determineUsedVariables(lines)

	prefixes := splitOnSpace("pre do post")
	actions := splitOnSpace("fetch extract patch tools wrapper configure build test install package clean")
	for _, prefix := range prefixes {
		for _, action := range actions {
			allowedTargets[prefix+"-"+action] = true
		}
	}

	// In the first pass, all additions to BUILD_DEFS and USE_TOOLS
	// are collected to make the order of the definitions irrelevant.

	for _, line := range lines {
		if line.extra["is_varassign"] == nil {
			continue
		}

		varcanon := line.extra["varcanon"].(string)
		switch varcanon {
		case "BUILD_DEFS", "PKG_GROUPS_VARS", "PKG_USERS_VARS":
			for _, varname := range splitOnSpace(line.extra["value"].(string)) {
				G.mkContext.buildDefs[varname] = true
				_ = G.opts.optDebugMisc && line.logDebug("%q is added to BUILD_DEFS.", varname)
			}

		case "PLIST_VARS":
			for _, id := range splitOnSpace(line.extra["value"].(string)) {
				G.mkContext.plistVars["PLIST."+id] = true
				_ = G.opts.optDebugMisc && line.logDebug("PLIST.%s is added to PLIST_VARS.", id)
				useVar(line, "PLIST."+id)
			}

		case "USE_TOOLS":
			for _, tool := range splitOnSpace(line.extra["value"].(string)) {
				tool = strings.Split(tool, ":")[0]
				G.mkContext.tools[tool] = true
				_ = G.opts.optDebugMisc && line.logDebug("%s is added to USE_TOOLS.", tool)
			}

		case "SUBST_VARS.*":
			for _, svar := range splitOnSpace(line.extra["value"].(string)) {
				useVar(line, varnameCanon(svar))
				_ = G.opts.optDebugMisc && line.logDebug("varuse %s", svar)
			}

		case "OPSYSVARS":
			for _, osvar := range splitOnSpace(line.extra["value"].(string)) {
				useVar(line, osvar+".*")
				defineVar(line, osvar)
			}
		}
	}

	// In the second pass, the actual checks are done.

	checklineRcsid(lines[0], `#\s+`, "# ")

	for _, line := range lines {
		text := line.text

		checklineTrailingWhitespace(line)

		if line.extra["is_empty"] != nil {
			substcontext.finish(line)

		} else if line.extra["is_comment"] != nil {
			// No further checks.

		} else if m := reCompile(reVarassign).FindStringSubmatchIndex(text); m != nil {
			varname := text[m[2]:m[3]]
			space1 := text[m[3]:m[4]]
			op := text[m[4]:m[5]]
			align := text[m[5]:m[6]]
			value := line.extra["value"].(string)
			comment := text[negToZero(m[8]):negToZero(m[9])]

			if align != " " && !matches(align, `^\t*$`) {
				_ = G.opts.optWarnSpace && line.logNote("Alignment of variable values should be done with tabs, not spaces.")
				prefix := varname + space1 + op
				alignedLen := tabLength(prefix + align)
				if alignedLen%8 == 0 {
					tabalign := strings.Repeat("\t", ((alignedLen - tabLength(prefix) + 7) / 8))
					line.replace(prefix+align, prefix+tabalign)
				}
			}
			checklineMkVarassign(line, varname, op, value, comment)
			substcontext.checkVarassign(line, varname, op, value)

		} else if m, shellcmd := match1(text, reMkShellcmd); m {
			checklineMkShellcmd(line, shellcmd)

		} else if m, include, includefile := match2(text, reMkInclude); m {
			_ = G.opts.optDebugInclude && line.logDebug("includefile=%s", includefile)
			checklineRelativePath(line, includefile, include == "include")

			if hasSuffix(includefile, "../Makefile") {
				line.logError("Other Makefiles must not be included directly.")
				line.explainError(
					"If you want to include portions of another Makefile, extract",
					"the common parts and put them into a Makefile.common. After",
					"that, both this one and the other package should include the",
					"Makefile.common.")
			}

			if includefile == "../../mk/bsd.prefs.mk" {
				if path.Base(line.fname) == "buildlink3.mk" {
					line.logNote("For efficiency reasons, please include bsd.fast.prefs.mk instead of bsd.prefs.mk.")
				}
				G.pkgContext.seen_bsd_prefs_mk = true
			} else if includefile == "../../mk/bsd.fast.prefs.mk" {
				G.pkgContext.seen_bsd_prefs_mk = true
			}

			if matches(includefile, `/x11-links/buildlink3\.mk$`) {
				line.logError("%s must not be included directly. Include \"../../mk/x11.buildlink3.mk\" instead.", includefile)
			}
			if matches(includefile, `/jpeg/buildlink3\.mk$`) {
				line.logError("%s must not be included directly. Include \"../../mk/jpeg.buildlink3.mk\" instead.", includefile)
			}
			if matches(includefile, `/intltool/buildlink3\.mk$`) {
				line.logWarning("Please write \"USE_TOOLS+= intltool\" instead of this line.")
			}
			if m, dir := match1(includefile, `(.*)/builtin\.mk$`); m {
				line.logError("%s must not be included directly. Include \"%s/buildlink3.mk\" instead.", includefile, dir)
			}

		} else if matches(text, reMkSysinclude) {

		} else if m, indent, directive, args := match3(text, reMkCond); m {
			if matches(directive, `^(?:endif|endfor|elif|else)$`) {
				if len(ctx.indentation) > 1 {
					ctx.popIndent()
				} else {
					line.logError("Unmatched .%s.", directive)
				}
			}

			// Check the indentation
			if indent != strings.Repeat(" ", ctx.indentDepth()) {
				_ = G.opts.optWarnSpace && line.logNote("This directive should be indented by %d spaces.", ctx.indentDepth())
			}

			if directive == "if" && matches(args, `^!defined\([\w]+_MK\)$`) {
				ctx.pushIndent(ctx.indentDepth())

			} else if matches(directive, `^(?:if|ifdef|ifndef|for|elif|else)$`) {
				ctx.pushIndent(ctx.indentDepth() + 2)
			}

			reDirectivesWithArgs := `^(?:if|ifdef|ifndef|elif|for|undef)$`
			if matches(directive, reDirectivesWithArgs) && args == "" {
				line.logError("\".%s\" requires arguments.", directive)

			} else if !matches(directive, reDirectivesWithArgs) && args != "" {
				line.logError("\".%s\" does not take arguments.", directive)

				if directive == "else" {
					line.logNote("If you meant \"else if\", use \".elif\".")
				}

			} else if directive == "if" || directive == "elif" {
				checklineMkCond(line, args)

			} else if directive == "ifdef" || directive == "ifndef" {
				if matches(args, `\s`) {
					line.logError("The \".%s\" directive can only handle _one_ argument.", directive)
				} else {
					line.logWarning("The \".%s\" directive is deprecated. Please use \".if %sdefined(%s)\" instead.",
						directive, ifelseStr(directive == "ifdef", "", "!"), args)
				}

			} else if directive == "for" {
				if m, vars, values := match2(args, `^(\S+(?:\s*\S+)*?)\s+in\s+(.*)$`); m {
					for _, forvar := range splitOnSpace(vars) {
						if !G.isInfrastructure && hasPrefix(forvar, "_") {
							line.logWarning("Variable names starting with an underscore are reserved for internal pkgsrc use.")
						}

						if matches(forvar, `^[_a-z][_a-z0-9]*$`) {
							// Fine.
						} else if matches(forvar, `[A-Z]`) {
							line.logWarning(".for variable names should not contain uppercase letters.")
						} else {
							line.logError("Invalid variable name %q.", forvar)
						}

						ctx.forVars[forvar] = true
					}

					// Check if any of the value's types is not guessed.
					guessed := GUESSED
					for _, value := range splitOnSpace(values) {
						if m, vname := match1(value, `^\$\{(.*)\}`); m {
							vartype := getVariableType(line, vname)
							if vartype != nil && !vartype.guessed {
								guessed = NOT_GUESSED
							}
						}
					}

					forLoopType := newBasicVartype(LK_SPACE, "Unchecked", []AclEntry{{"*", "pu"}}, guessed)
					forLoopContext := &VarUseContext{
						VUC_TIME_LOAD,
						forLoopType,
						VUC_SHW_FOR,
						VUC_EXT_WORD,
					}
					for _, fvar := range extractUsedVariables(line, values) {
						checklineMkVaruse(line, fvar, "", forLoopContext)
					}
				}

			} else if directive == "undef" && args != "" {
				for _, uvar := range splitOnSpace(args) {
					if ctx.forVars[uvar] {
						line.logNote("Using \".undef\" after a \".for\" loop is unnecessary.")
					}
				}
			}

		} else if m, targets, _, dependencies := match3(text, reMkDependency); m {
			_ = G.opts.optDebugMisc && line.logDebug("targets=%v, dependencies=%v", targets, dependencies)
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
					line.logWarning("Unusual target %q.", target)
					line.explainWarning(
						"If you want to define your own targets, you can \"declare\"",
						"them by inserting a \".PHONY: my-target\" line before this line. This",
						"will tell make(1) to not interpret this target's name as a filename.")
				}
			}

		} else if m, directive := match1(text, `^\.\s*(\S*)`); m {
			line.logError("Unknown directive \".%s\".", directive)

		} else if hasPrefix(text, " ") {
			line.logWarning("Makefile lines should not start with space characters.")
			line.explainWarning(
				"If you want this line to contain a shell program, use a tab",
				"character for indentation. Otherwise please remove the leading",
				"white-space.")

		} else {
			_ = G.opts.optDebugMisc && line.logDebug("Unknown line format")
		}
	}
	substcontext.finish(lines[len(lines)-1])

	checklinesTrailingEmptyLines(lines)

	if len(ctx.indentation) != 1 {
		lines[len(lines)-1].logError("Directive indentation is not 0, but %d.", ctx.indentDepth())
	}

	G.mkContext = nil
}
