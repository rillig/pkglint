package main

import (
	"path"
	"strings"
)

const (
	reMkDependency = `^([^\s:]+(?:\s*[^\s:]+)*)(\s*):\s*([^#]*?)(?:\s*#.*)?$`
	reMkSysinclude = `^\.\s*s?include\s+<([^>]+)>\s*(?:#.*)?$`
)

func readMakefile(fname string, mainLines *[]*Line, allLines *[]*Line) bool {
	defer tracecall("readMakefile", fname)()

	fileLines := LoadNonemptyLines(fname, true)
	if fileLines == nil {
		return false
	}

	isMainMakefile := len(*mainLines) == 0

	for _, mkline := range NewMkLines(fileLines).mklines {
		line := mkline.Line
		text := line.text

		if isMainMakefile {
			*mainLines = append(*mainLines, line)
		}
		*allLines = append(*allLines, line)

		var includeFile, incDir, incBase string
		if hasPrefix(text, ".") && hasSuffix(text, "\"") {
			if m, inc := match1(text, `^\.\s*include\s+\"(.*)\"$`); m {
				includeFile = resolveVariableRefs(resolveVarsInRelativePath(inc, true))
				if containsVarRef(includeFile) {
					if !contains(fname, "/mk/") {
						line.notef("Skipping include file %q. This may result in false warnings.", includeFile)
					}
					includeFile = ""
				}
				incDir, incBase = path.Split(includeFile)
			}
		}

		if includeFile != "" {
			if path.Base(fname) != "buildlink3.mk" {
				if m, bl3File := match1(includeFile, `^\.\./\.\./(.*)/buildlink3\.mk$`); m {
					G.pkg.bl3[bl3File] = line
					_ = G.opts.DebugMisc && line.debugf("Buildlink3 file in package: %q", bl3File)
				}
			}
		}

		if includeFile != "" && G.pkg.included[includeFile] == nil {
			G.pkg.included[includeFile] = line

			if matches(includeFile, `^\.\./[^./][^/]*/[^/]+`) {
				mkline.warnf("References to other packages should look like \"../../category/package\", not \"../package\".")
				mkline.explainRelativeDirs()
			}

			if !hasPrefix(incDir, "../../mk/") && incBase != "buildlink3.mk" && incBase != "builtin.mk" && incBase != "options.mk" {
				_ = G.opts.DebugInclude && line.debugf("Including %q sets seenMakefileCommon.", includeFile)
				G.pkg.seenMakefileCommon = true
			}

			if !contains(incDir, "/mk/") {
				dirname, _ := path.Split(fname)
				dirname = cleanpath(dirname)

				// Only look in the directory relative to the
				// current file and in the current working directory.
				// Pkglint doesn’t have an include dir list, like make(1) does.
				if !fileExists(dirname + "/" + includeFile) {
					dirname = G.currentDir
				}
				if !fileExists(dirname + "/" + includeFile) {
					line.errorf("Cannot read %q.", dirname+"/"+includeFile)
					return false
				}

				_ = G.opts.DebugInclude && line.debugf("Including %q.", dirname+"/"+includeFile)
				lengthBeforeInclude := len(*allLines)
				if !readMakefile(dirname+"/"+includeFile, mainLines, allLines) {
					return false
				}

				if incBase == "Makefile.common" && incDir != "" {
					makefileCommonLines := (*allLines)[lengthBeforeInclude:]
					NewMkLines(makefileCommonLines).checkForUsedComment(relpath(G.globalData.pkgsrcdir, fname))
				}
			}
		}

		if mkline.IsVarassign() {
			varname, op, value := mkline.Varname(), mkline.Op(), mkline.Value()

			if op != "?=" || G.pkg.vardef[varname] == nil {
				_ = G.opts.DebugMisc && line.debugf("varassign(%q, %q, %q)", varname, op, value)
				G.pkg.vardef[varname] = NewMkLine(line) // XXX
			}
		}
	}

	return true
}

func resolveVarsInRelativePath(relpath string, adjustDepth bool) string {

	tmp := relpath
	tmp = strings.Replace(tmp, "${PKGSRCDIR}", G.curPkgsrcdir, -1)
	tmp = strings.Replace(tmp, "${.CURDIR}", ".", -1)
	tmp = strings.Replace(tmp, "${.PARSEDIR}", ".", -1)
	tmp = strings.Replace(tmp, "${LUA_PKGSRCDIR}", "../../lang/lua52", -1)
	tmp = strings.Replace(tmp, "${PHPPKGSRCDIR}", "../../lang/php54", -1)
	tmp = strings.Replace(tmp, "${SUSE_DIR_PREFIX}", "suse100", -1)
	tmp = strings.Replace(tmp, "${PYPKGSRCDIR}", "../../lang/python27", -1)
	if G.pkg != nil {
		tmp = strings.Replace(tmp, "${FILESDIR}", G.pkg.filesdir, -1)
		tmp = strings.Replace(tmp, "${PKGDIR}", G.pkg.pkgdir, -1)
	}

	if adjustDepth {
		if m, pkgpath := match1(tmp, `^\.\./\.\./([^.].*)$`); m {
			tmp = G.curPkgsrcdir + "/" + pkgpath
		}
	}

	_ = G.opts.DebugMisc && dummyLine.debugf("resolveVarsInRelativePath: %q => %q", relpath, tmp)
	return tmp
}
