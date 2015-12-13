package main

import (
	"regexp"
	"strings"
)

func checklinesBuildlink3Mk(mklines *MkLines) {
	defer tracecall("checklinesBuildlink3Mk", mklines.lines[0].fname)()

	mklines.check()

	exp := NewExpecter(mklines.lines)

	for {
		if exp.advanceIfPrefix("# XXX") {
			exp.previousLine().notef("Please read this comment and remove it if appropriate.")
		} else if !exp.advanceIfPrefix("#") {
			break
		}
	}

	exp.expectEmptyLine()

	if exp.advanceIfMatches(`^BUILDLINK_DEPMETHOD\.(\S+)\?=.*$`) {
		exp.previousLine().warnf("This line belongs inside the .ifdef block.")
		for exp.advanceIfEquals("") {
		}
	}

	pkgbaseLine, pkgbase := (*Line)(nil), ""
	pkgidLine, pkgid := exp.currentLine(), ""
	abiLine, abiPkg, abiVersion := (*Line)(nil), "", ""
	apiLine, apiPkg, apiVersion := (*Line)(nil), "", ""

	// First paragraph: Introduction of the package identifier
	if exp.advanceIfMatches(`^BUILDLINK_TREE\+=\s*(\S+)$`) {
		pkgid = exp.m[1]
	} else {
		exp.currentLine().warnf("Expected a BUILDLINK_TREE line.")
		return
	}
	exp.expectEmptyLine()

	// Second paragraph: multiple inclusion protection and introduction
	// of the uppercase package identifier.
	if exp.advanceIfMatches(`^\.if !defined\((\S+)_BUILDLINK3_MK\)$`) {
		pkgbase = exp.m[1]
		pkgbaseLine = exp.previousLine()
	} else {
		return
	}
	if !exp.expectText(pkgbase + "_BUILDLINK3_MK:=") {
		exp.currentLine().errorf("Expected the multiple-inclusion guard.")
		return
	}
	exp.expectEmptyLine()

	ucPkgid := strings.ToUpper(strings.Replace(pkgid, "-", "_", -1))
	if ucPkgid != pkgbase {
		pkgbaseLine.errorf("Package name mismatch between %q ...", pkgbase)
		pkgidLine.errorf("... and %q.", pkgid)
	}
	if G.pkg != nil {
		if mkbase := G.pkg.effectivePkgbase; mkbase != "" && mkbase != pkgid {
			pkgidLine.errorf("Package name mismatch between %q ...", pkgid)
			G.pkg.effectivePkgnameLine.errorf("... and %q.", mkbase)
		}
	}

	// Third paragraph: Package information.
	indentLevel := 1 // The first .if is from the second paragraph.
	for {
		if exp.eof() {
			exp.currentLine().warnf("Expected .endif")
			return
		}

		line := exp.currentLine()

		if exp.advanceIfMatches(reVarassign) {
			varname, value := exp.m[1], exp.m[3]
			doCheck := false

			if varname == "BUILDLINK_ABI_DEPENDS."+pkgid {
				abiLine = line
				if m, p, v := match2(value, reDependencyCmp); m {
					abiPkg, abiVersion = p, v
				} else if m, p := match1(value, reDependencyWildcard); m {
					abiPkg, abiVersion = p, ""
				} else {
					_ = G.opts.DebugUnchecked && line.debugf("Unchecked dependency pattern %q.", value)
				}
				doCheck = true
			}
			if varname == "BUILDLINK_API_DEPENDS."+pkgid {
				apiLine = line
				if m, p, v := match2(value, reDependencyCmp); m {
					apiPkg, apiVersion = p, v
				} else if m, p := match1(value, reDependencyWildcard); m {
					apiPkg, apiVersion = p, ""
				} else {
					_ = G.opts.DebugUnchecked && line.debugf("Unchecked dependency pattern %q.", value)
				}
				doCheck = true
			}
			if doCheck && abiPkg != "" && apiPkg != "" && abiPkg != apiPkg {
				abiLine.warnf("Package name mismatch between %q ...", abiPkg)
				apiLine.warnf("... and %q.", apiPkg)
			}
			if doCheck && abiVersion != "" && apiVersion != "" && pkgverCmp(abiVersion, apiVersion) < 0 {
				abiLine.warnf("ABI version (%s) should be at least ...", abiVersion)
				apiLine.warnf("... API version (%s).", apiVersion)
			}

			if m, varparam := match1(varname, `^BUILDLINK_[\w_]+\.(.*)$`); m {
				if varparam != pkgid {
					line.warnf("Only buildlink variables for %q, not %q may be set in this file.", pkgid, varparam)
				}
			}

			if varname == "pkgbase" {
				exp.advanceIfMatches(`^\.\s*include "../../mk/pkg-build-options\.mk"$`)
			}

		} else if exp.advanceIfEquals("") || exp.advanceIfPrefix("#") {
			// Comments and empty lines are fine here.

		} else if exp.advanceIfMatches(`^\.\s*include "\.\./\.\./([^/]+/[^/]+)/buildlink3\.mk"$`) ||
			exp.advanceIfMatches(`^\.\s*include "\.\./\.\./mk/(\S+)\.buildlink3\.mk"$`) {
			// TODO: Maybe check dependency lines.

		} else if exp.advanceIfMatches(`^\.if\s`) {
			indentLevel++

		} else if exp.advanceIfMatches(`^\.endif.*$`) {
			indentLevel--
			if indentLevel == 0 {
				break
			}

		} else {
			_ = G.opts.DebugUnchecked && exp.currentLine().warnf("Unchecked line in third paragraph.")
			exp.advance()
		}
	}
	if apiLine == nil {
		exp.currentLine().warnf("Definition of BUILDLINK_API_DEPENDS is missing.")
	}
	exp.expectEmptyLine()

	// Fourth paragraph: Cleanup, corresponding to the first paragraph.
	if !exp.advanceIfMatches(`^BUILDLINK_TREE\+=\s*-` + regexp.QuoteMeta(pkgid) + `$`) {
		exp.currentLine().warnf("Expected BUILDLINK_TREE line.")
	}

	if !exp.eof() {
		exp.currentLine().warnf("The file should end here.")
	}

	if G.pkg != nil {
		G.pkg.checklinesBuildlink3Inclusion(mklines)
	}

	saveAutofixChanges(mklines.lines)
}
