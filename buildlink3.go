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
			exp.previousLine().note0("Please read this comment and remove it if appropriate.")
		} else if !exp.advanceIfPrefix("#") {
			break
		}
	}

	exp.expectEmptyLine()

	if exp.advanceIfMatches(`^BUILDLINK_DEPMETHOD\.(\S+)\?=.*$`) {
		exp.previousLine().warn0("This line belongs inside the .ifdef block.")
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
		exp.currentLine().warn0("Expected a BUILDLINK_TREE line.")
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
		exp.currentLine().error0("Expected the multiple-inclusion guard.")
		return
	}
	exp.expectEmptyLine()

	ucPkgid := strings.ToUpper(strings.Replace(pkgid, "-", "_", -1))
	if ucPkgid != pkgbase {
		pkgbaseLine.error1("Package name mismatch between %q ...", pkgbase)
		pkgidLine.error1("... and %q.", pkgid)
	}
	if G.pkg != nil {
		if mkbase := G.pkg.effectivePkgbase; mkbase != "" && mkbase != pkgid {
			pkgidLine.error1("Package name mismatch between %q ...", pkgid)
			G.pkg.effectivePkgnameLine.line.error1("... and %q.", mkbase)
		}
	}

	// Third paragraph: Package information.
	indentLevel := 1 // The first .if is from the second paragraph.
	for {
		if exp.eof() {
			exp.currentLine().warn0("Expected .endif")
			return
		}

		line := exp.currentLine()
		mkline := mklines.mklines[exp.index]

		if mkline.IsVarassign() {
			exp.advance()
			varname, value := mkline.Varname(), mkline.Value()
			doCheck := false

			if varname == "BUILDLINK_ABI_DEPENDS."+pkgid {
				abiLine = line
				if m, p, v := match2(value, reDependencyCmp); m {
					abiPkg, abiVersion = p, v
				} else if m, p := match1(value, reDependencyWildcard); m {
					abiPkg, abiVersion = p, ""
				} else {
					if G.opts.DebugUnchecked {
						line.debug1("Unchecked dependency pattern %q.", value)
					}
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
					if G.opts.DebugUnchecked {
						line.debug1("Unchecked dependency pattern %q.", value)
					}
				}
				doCheck = true
			}
			if doCheck && abiPkg != "" && apiPkg != "" && abiPkg != apiPkg {
				abiLine.warn1("Package name mismatch between %q ...", abiPkg)
				apiLine.warn1("... and %q.", apiPkg)
			}
			if doCheck && abiVersion != "" && apiVersion != "" && pkgverCmp(abiVersion, apiVersion) < 0 {
				abiLine.warn1("ABI version (%s) should be at least ...", abiVersion)
				apiLine.warn1("... API version (%s).", apiVersion)
			}

			if m, varparam := match1(varname, `^BUILDLINK_[\w_]+\.(.*)$`); m {
				if varparam != pkgid {
					line.warn2("Only buildlink variables for %q, not %q may be set in this file.", pkgid, varparam)
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
			if G.opts.DebugUnchecked {
				exp.currentLine().warn0("Unchecked line in third paragraph.")
			}
			exp.advance()
		}
	}
	if apiLine == nil {
		exp.currentLine().warn0("Definition of BUILDLINK_API_DEPENDS is missing.")
	}
	exp.expectEmptyLine()

	// Fourth paragraph: Cleanup, corresponding to the first paragraph.
	if !exp.advanceIfMatches(`^BUILDLINK_TREE\+=\s*-` + regexp.QuoteMeta(pkgid) + `$`) {
		exp.currentLine().warn0("Expected BUILDLINK_TREE line.")
	}

	if !exp.eof() {
		exp.currentLine().warn0("The file should end here.")
	}

	if G.pkg != nil {
		G.pkg.checklinesBuildlink3Inclusion(mklines)
	}

	saveAutofixChanges(mklines.lines)
}
