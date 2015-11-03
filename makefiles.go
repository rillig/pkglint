package main

import (
	"path"
	"path/filepath"
	"strings"
)

func readMakefile(fname string, mainLines []*Line, allLines []*Line) bool {
	lines, err := loadLines(fname, true)
	if err != nil {
		return false
	}

	parselinesMk(lines)
	isMainMakefile := len(mainLines) == 0

	for _, line := range lines {
		text := line.text

		if isMainMakefile {
			mainLines = append(mainLines, line)
		}
		allLines = append(allLines, line)

		isIncludeLine := false
		includeFile := ""
		if m := match(text, `^\.\s*include\s+\"(.*)\"$`); m != nil {
			includeFile = resolveVarsInRelativePath(m[1], true)
			if match(includeFile, reUnresolvedVar) != nil {
				if match(fname, `/mk/`) == nil {
					line.logNoteF("Skipping include file %q. This may result in false warnings.", includeFile)
				}
			} else {
				isIncludeLine = true
			}
		}

		if isIncludeLine {
			if path.Base(fname) == "buildlink3.mk" {
				if m := match(includeFile, `^\.\./\.\./(.*)/buildlink3\.mk$`); m != nil {
					bl3File := m[1]

					GlobalVars.pkgContext.bl3[bl3File] = line
					_ = GlobalVars.opts.optDebugMisc && line.logDebugF("Buildlink3 file in package: %v", bl3File)
				}
			}
		}

		if isIncludeLine && GlobalVars.pkgContext.included[includeFile] == nil {
			GlobalVars.pkgContext.included[includeFile] = line

			if match(includeFile, `^\.\./[^./][^/]*/[^/]+`) != nil {
				line.logWarningF("References to other packages should look like \"../../category/package\", not \"../package\".")
				line.explainWarning(explanationRelativeDirs()...)
			}
			if path.Base(includeFile) == "Makefile.common" {
				_ = GlobalVars.opts.optDebugInclude && line.logDebugF("Including %q sets seenMakefileCommon.", includeFile)
				GlobalVars.pkgContext.seenMakefileCommon = true
			}
			if m := match(includeFile, `^(?:\.\./(\.\./[^/]+/)?[^/]+/)?([^/]+)$`); m != nil {
				if m[2] != "buildlink3.mk" && m[2] != "options.mk" {
					_ = GlobalVars.opts.optDebugInclude && line.logDebugF("Including %q sets seenMakefileCommon.", includeFile)
					GlobalVars.pkgContext.seenMakefileCommon = true
				}
			}

			if match(includeFile, `/mk/`) == nil {
				dirname := path.Dir(fname)

				// Only look in the directory relative to the
				// current file and in the current working directory.
				// We don't have an include dir list, like make(1) does.
				if !fileExists(dirname + "/" + includeFile) {
					dirname = GlobalVars.currentDir
				}
				if !fileExists(dirname + "/" + includeFile) {
					line.logErrorF("Cannot read %q.", dirname+"/"+includeFile)
					return false
				} else {
					_ = GlobalVars.opts.optDebugInclude && line.logDebugF("Including %q.", dirname+"/"+includeFile)
					lengthBeforeInclude := len(allLines)
					if !readMakefile(dirname+"/"+includeFile, mainLines, allLines) {
						return false
					}

					if path.Base(includeFile) == "Makefile.common" {
						makefileCommonLines := allLines[lengthBeforeInclude:]
						relpath, err := filepath.Rel(*GlobalVars.cwdPkgsrcdir, fname)
						if err != nil {
							line.logErrorF("Cannot determine relative path.")
							return false
						}
						checkForUsedComment(makefileCommonLines, relpath)
					}
				}
			}
		}

		if line.extra["is_varassign"] != nil {
			varname, op, value := line.extra["varname"].(string), line.extra["op"].(string), line.extra["value"].(string)

			if op != "?=" || GlobalVars.pkgContext.vardef[varname] == nil {
				_ = GlobalVars.opts.optDebugMisc && line.logDebugF("varassign(%q, %q, %q)", varname, op, value)
				GlobalVars.pkgContext.vardef[varname] = line
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
		if match(line.text, reMkComment) == nil {
			break
		}
		lastCommentLine = i
	}

	insertLine := lines[lastCommentLine+1]
	insertLine.logWarningF("Please add a line %q here.", expected)
	insertLine.explainWarning(
		`Since Makefile.common files usually don't have any comments and
therefore not a clearly defined interface, they should at least contain
references to all files that include them, so that it is easier to see
what effects future changes may have.

If there are more than five packages that use a Makefile.common,
you should think about giving it a proper name (maybe plugin.mk) and
documenting its interface.`)
	insertLine.appendBefore(expected)
	if GlobalVars.opts.optAutofix {
		saveAutofixChanges(lines)
	}
}

func resolveVarsInRelativePath(relpath string, adjustDepth bool) string {

	tmp := relpath
	tmp = strings.Replace(tmp, "${PKGSRCDIR}", *GlobalVars.curPkgsrcdir, -1)
	tmp = strings.Replace(tmp, "${.CURDIR}", ".", -1)
	tmp = strings.Replace(tmp, "${.PARSEDIR}", ".", -1)
	tmp = strings.Replace(tmp, "${LUA_PKGSRCDIR}", "../../lang/lua52", -1)
	tmp = strings.Replace(tmp, "${PHPPKGSRCDIR}", "../../lang/php54", -1)
	tmp = strings.Replace(tmp, "${SUSE_DIR_PREFIX}", "suse100", -1)
	tmp = strings.Replace(tmp, "${PYPKGSRCDIR}", "../../lang/python27", -1)
	if GlobalVars.pkgContext.filesdir != nil {
		tmp = strings.Replace(tmp, "${FILESDIR}", *GlobalVars.pkgContext.filesdir, -1)
	}
	if adjustDepth {
		if m := match(tmp, `^\.\./\.\./([^.].*)$`); m != nil {
			tmp = *GlobalVars.curPkgsrcdir + "/" + m[1]
		}
	}
	if GlobalVars.pkgContext.pkgdir != nil {
		tmp = strings.Replace(tmp, "${PKGDIR}", *GlobalVars.pkgContext.pkgdir, -1)
	}

	_ = GlobalVars.opts.optDebugMisc && logDebugF(NO_FILE, NO_LINES, "resolveVarsInRelativePath: %q => %q", relpath, tmp)
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

		shellwords, rest := matchAll(shellcmd, reShellword)
		line.extra["shellwords"] = shellwords
		if match(rest, `^\s*$`) == nil {
			line.extra["shellwords_rest"] = rest
		}
		return
	}

	if m, comment := match1(text, reMkComment); m {
		line.extra["is_comment"] = true
		line.extra["comment"] = comment
		return
	}

	if m := match(text, `^\s*$`); m != nil {
		line.extra["is_empty"] = true
		return
	}

	if m, indent, directive, args, comment := match4(text, reMkCond); m {
		line.extra["is_cond"] = true
		line.extra["indent"] = indent
		line.extra["directive"] = directive
		line.extra["args"] = args
		line.extra["comment"] = comment
		return
	}

	if m, _, includefile, comment := match3(text, reMkInclude); m {
		line.extra["is_include"] = true
		line.extra["includefile"] = includefile
		line.extra["comment"] = comment
		return
	}

	if m, includefile, comment := match2(text, reMkSysinclude); m {
		line.extra["is_sysinclude"] = true
		line.extra["includefile"] = includefile
		line.extra["comment"] = comment
		return
	}

	if m, targets, whitespace, sources, comment := match4(text, reMkDependency); m {
		line.extra["is_dependency"] = true
		line.extra["targets"] = targets
		line.extra["sources"] = sources
		line.extra["comment"] = comment
		if whitespace != "" {
			line.logWarningF("Space before colon in dependency line.")
		}
		return
	}

	if match(text, reConflict) != nil {
		return
	}

	line.logFatalF("Unknown Makefile line format.")
}

func parselinesMk(lines []*Line) {
	for _, line := range lines {
		parselineMk(line)
	}
}
