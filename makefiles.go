package main

import (
	"path"
	"strings"
)

func readMakefile(fname string, mainLines *MkLines, allLines *MkLines, includingFnameForUsedCheck string) bool {
	if G.opts.DebugTrace {
		defer tracecall1("readMakefile", fname)()
	}

	fileLines := LoadNonemptyLines(fname, true)
	if fileLines == nil {
		return false
	}
	fileMklines := NewMkLines(fileLines)

	isMainMakefile := len(mainLines.mklines) == 0

	for _, mkline := range fileMklines.mklines {
		line := mkline.line

		if isMainMakefile {
			mainLines.mklines = append(mainLines.mklines, mkline)
			mainLines.lines = append(mainLines.lines, line)
		}
		allLines.mklines = append(allLines.mklines, mkline)
		allLines.lines = append(allLines.lines, line)

		var includeFile, incDir, incBase string
		if mkline.IsInclude() {
			inc := mkline.Includefile()
			includeFile = resolveVariableRefs(resolveVarsInRelativePath(inc, true))
			if containsVarRef(includeFile) {
				if !strings.Contains(fname, "/mk/") {
					line.note1("Skipping include file %q. This may result in false warnings.", includeFile)
				}
				includeFile = ""
			}
			incDir, incBase = path.Split(includeFile)
		}

		if includeFile != "" {
			if path.Base(fname) != "buildlink3.mk" {
				if m, bl3File := match1(includeFile, `^\.\./\.\./(.*)/buildlink3\.mk$`); m {
					G.Pkg.bl3[bl3File] = line
					if G.opts.DebugMisc {
						line.debug1("Buildlink3 file in package: %q", bl3File)
					}
				}
			}
		}

		if includeFile != "" && G.Pkg.included[includeFile] == nil {
			G.Pkg.included[includeFile] = line

			if matches(includeFile, `^\.\./[^./][^/]*/[^/]+`) {
				mkline.warn0("References to other packages should look like \"../../category/package\", not \"../package\".")
				mkline.explainRelativeDirs()
			}

			if path.Base(fname) == "Makefile" && !hasPrefix(incDir, "../../mk/") && incBase != "buildlink3.mk" && incBase != "builtin.mk" && incBase != "options.mk" {
				if G.opts.DebugInclude {
					line.debug1("Including %q sets seenMakefileCommon.", includeFile)
				}
				G.Pkg.seenMakefileCommon = true
			}

			if !strings.Contains(incDir, "/mk/") || strings.HasSuffix(includeFile, "/mk/haskell.mk") {
				dirname, _ := path.Split(fname)
				dirname = cleanpath(dirname)

				// Only look in the directory relative to the
				// current file and in the current working directory.
				// Pkglint doesn’t have an include dir list, like make(1) does.
				if !fileExists(dirname + "/" + includeFile) {
					dirname = G.CurrentDir
				}
				if !fileExists(dirname + "/" + includeFile) {
					line.error1("Cannot read %q.", dirname+"/"+includeFile)
					return false
				}

				if G.opts.DebugInclude {
					line.debug1("Including %q.", dirname+"/"+includeFile)
				}
				includingFname := ifelseStr(incBase == "Makefile.common" && incDir != "", fname, "")
				if !readMakefile(dirname+"/"+includeFile, mainLines, allLines, includingFname) {
					return false
				}
			}
		}

		if mkline.IsVarassign() {
			varname, op, value := mkline.Varname(), mkline.Op(), mkline.Value()

			if op != "?=" || G.Pkg.vardef[varname] == nil {
				if G.opts.DebugMisc {
					line.debugf("varassign(%q, %q, %q)", varname, op, value)
				}
				G.Pkg.vardef[varname] = mkline
			}
		}
	}

	if includingFnameForUsedCheck != "" {
		fileMklines.checkForUsedComment(relpath(G.globalData.Pkgsrcdir, includingFnameForUsedCheck))
	}

	return true
}

func resolveVarsInRelativePath(relpath string, adjustDepth bool) string {

	tmp := relpath
	tmp = strings.Replace(tmp, "${PKGSRCDIR}", G.CurPkgsrcdir, -1)
	tmp = strings.Replace(tmp, "${.CURDIR}", ".", -1)
	tmp = strings.Replace(tmp, "${.PARSEDIR}", ".", -1)
	tmp = strings.Replace(tmp, "${LUA_PKGSRCDIR}", "../../lang/lua52", -1)
	tmp = strings.Replace(tmp, "${PHPPKGSRCDIR}", "../../lang/php55", -1)
	tmp = strings.Replace(tmp, "${SUSE_DIR_PREFIX}", "suse100", -1)
	tmp = strings.Replace(tmp, "${PYPKGSRCDIR}", "../../lang/python27", -1)
	if G.Pkg != nil {
		tmp = strings.Replace(tmp, "${FILESDIR}", G.Pkg.filesdir, -1)
		tmp = strings.Replace(tmp, "${PKGDIR}", G.Pkg.pkgdir, -1)
	}

	if adjustDepth {
		if m, pkgpath := match1(tmp, `^\.\./\.\./([^.].*)$`); m {
			tmp = G.CurPkgsrcdir + "/" + pkgpath
		}
	}

	if G.opts.DebugMisc {
		dummyLine.debug2("resolveVarsInRelativePath: %q => %q", relpath, tmp)
	}
	return tmp
}
