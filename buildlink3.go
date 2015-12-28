package main

import (
	"regexp"
	"strings"
)

func checklinesBuildlink3Mk(mklines *MkLines) {
	if G.opts.DebugTrace {
		defer tracecall1("checklinesBuildlink3Mk", mklines.lines[0].fname)()
	}

	mklines.check()

	exp := NewExpecter(mklines.lines)

	for exp.advanceIfPrefix("#") {
		line := exp.previousLine()
		// See pkgtools/createbuildlink/files/createbuildlink
		if hasPrefix(line.text, "# XXX This file was created automatically") {
			line.note0("Please read this comment and remove it if appropriate.")
		}
	}

	exp.expectEmptyLine()

	if exp.advanceIfMatches(`^BUILDLINK_DEPMETHOD\.(\S+)\?=.*$`) {
		exp.previousLine().warn0("This line belongs inside the .ifdef block.")
		for exp.advanceIfEquals("") {
		}
	}

	pkgbaseLine, pkgbase := exp.currentLine(), ""
	abiLine, abiPkg, abiVersion := (*Line)(nil), "", ""
	apiLine, apiPkg, apiVersion := (*Line)(nil), "", ""

	// First paragraph: Introduction of the package identifier
	if !exp.advanceIfMatches(`^BUILDLINK_TREE\+=\s*(\S+)$`) {
		exp.currentLine().warn0("Expected a BUILDLINK_TREE line.")
		return
	}
	pkgbase = exp.m[1]

	exp.expectEmptyLine()

	// Second paragraph: multiple inclusion protection and introduction
	// of the uppercase package identifier.
	if !exp.advanceIfMatches(`^\.if !defined\((\S+)_BUILDLINK3_MK\)$`) {
		return
	}
	pkgupperLine, pkgupper := exp.previousLine(), exp.m[1]

	if !exp.expectText(pkgupper + "_BUILDLINK3_MK:=") {
		return
	}
	exp.expectEmptyLine()

	// See pkgtools/createbuildlink/files/createbuildlink, keyword PKGUPPER
	ucPkgbase := strings.ToUpper(strings.Replace(pkgbase, "-", "_", -1))
	if ucPkgbase != pkgupper {
		pkgupperLine.error1("Package name mismatch between multiple-inclusion guard %q ...", pkgupper)
		pkgbaseLine.error1("... and package name %q.", pkgbase)
	}
	if G.pkg != nil {
		if mkbase := G.pkg.effectivePkgbase; mkbase != "" && mkbase != pkgbase {
			pkgbaseLine.error1("Package name mismatch between %q in this file ...", pkgbase)
			G.pkg.effectivePkgnameLine.line.error1("... and %q from the package Makefile.", mkbase)
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

			if varname == "BUILDLINK_ABI_DEPENDS."+pkgbase {
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
			if varname == "BUILDLINK_API_DEPENDS."+pkgbase {
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
				abiLine.warn1("Package name mismatch between ABI %q ...", abiPkg)
				apiLine.warn1("... and API %q.", apiPkg)
			}
			if doCheck && abiVersion != "" && apiVersion != "" && pkgverCmp(abiVersion, apiVersion) < 0 {
				abiLine.warn1("ABI version (%s) should be at least ...", abiVersion)
				apiLine.warn1("... API version (%s).", apiVersion)
			}

			if varparam := mkline.Varparam(); varparam != "" && varparam != pkgbase {
				if hasPrefix(varname, "BUILDLINK_") && mkline.Varcanon() != "BUILDLINK_API_DEPENDS.*" {
					line.warn2("Only buildlink variables for %q, not %q may be set in this file.", pkgbase, varparam)
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
	if !exp.advanceIfMatches(`^BUILDLINK_TREE\+=\s*-` + regexp.QuoteMeta(pkgbase) + `$`) {
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
