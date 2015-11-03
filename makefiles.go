package main

import (
	"fmt"
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
					line.logNote("Skipping include file \"" + includeFile + "\". This may result in false warnings.")
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
					_ = GlobalVars.opts.optDebugMisc && line.logDebug("Buildlink3 file in package: "+bl3File)
				}
			}
		}

		if isIncludeLine && GlobalVars.pkgContext.included[includeFile] == nil {
			GlobalVars.pkgContext.included[includeFile] = line

			if match(includeFile, `^\.\./[^./][^/]*/[^/]+`) != nil {
				line.logWarning("References to other packages should look like \"../../category/package\", not \"../package\".")
				line.explainWarning(explanationRelativeDirs()...)
			}
			if path.Base(includeFile) == "Makefile.common" {
				_ = GlobalVars.opts.optDebugInclude && line.logDebug("Including \""+includeFile+"\" sets seenMakefileCommon.")
				GlobalVars.pkgContext.seenMakefileCommon = true
			}
			if m := match(includeFile, `^(?:\.\./(\.\./[^/]+/)?[^/]+/)?([^/]+)$`); m != nil {
				if m[2] != "buildlink3.mk" && m[2] != "options.mk" {
					_ = GlobalVars.opts.optDebugInclude && line.logDebug("Including \""+includeFile+"\" sets seenMakefileCommon.")
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
					line.logError("Cannot read " + dirname + "/" + includeFile + ".")
					return false
				} else {
					_ = GlobalVars.opts.optDebugInclude && line.logDebug(fmt.Sprintf("Including \"%s/%s\".", dirname, includeFile))
					lengthBeforeInclude := len(allLines)
					if !readMakefile(dirname+"/"+includeFile, mainLines, allLines) {
						return false
					}

					if path.Base(includeFile) == "Makefile.common" {
						makefileCommonLines := allLines[lengthBeforeInclude:]
						relpath, err := filepath.Rel(*GlobalVars.cwdPkgsrcdir, fname)
						if err != nil {
							line.logError("Cannot determine relative path.")
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
				_ = GlobalVars.opts.optDebugMisc && line.logDebug(fmt.Sprintf("varassign(%s, %s, %s)", varname, op, value))
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
	insertLine.logWarning("Please add a line \"" + expected + "\" here.")
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

func parselinesMk(lines []*Line) {
	panic("not implemented")
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

	_ = GlobalVars.opts.optDebugMisc && logDebug(NO_FILE, NO_LINES, "resolveVarsInRelativePath: "+relpath+" => "+tmp)
	return tmp
}
