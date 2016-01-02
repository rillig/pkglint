package main

import (
	"regexp"
	"strings"
)

func ChecklinesBuildlink3Mk(mklines *MkLines) {
	if G.opts.DebugTrace {
		defer tracecall1("checklinesBuildlink3Mk", mklines.lines[0].Fname)()
	}

	mklines.Check()

	exp := NewExpecter(mklines.lines)

	for exp.AdvanceIfPrefix("#") {
		line := exp.PreviousLine()
		// See pkgtools/createbuildlink/files/createbuildlink
		if hasPrefix(line.Text, "# XXX This file was created automatically") {
			line.Note0("Please read this comment and remove it if appropriate.")
		}
	}

	exp.ExpectEmptyLine()

	if exp.advanceIfMatches(`^BUILDLINK_DEPMETHOD\.(\S+)\?=.*$`) {
		exp.PreviousLine().Warn0("This line belongs inside the .ifdef block.")
		for exp.AdvanceIfEquals("") {
		}
	}

	pkgbaseLine, pkgbase := exp.CurrentLine(), ""
	var abiLine, apiLine *Line
	var abi, api *DependencyPattern

	// First paragraph: Introduction of the package identifier
	if !exp.advanceIfMatches(`^BUILDLINK_TREE\+=\s*(\S+)$`) {
		exp.CurrentLine().Warn0("Expected a BUILDLINK_TREE line.")
		return
	}
	pkgbase = exp.m[1]

	exp.ExpectEmptyLine()

	// Second paragraph: multiple inclusion protection and introduction
	// of the uppercase package identifier.
	if !exp.advanceIfMatches(`^\.if !defined\((\S+)_BUILDLINK3_MK\)$`) {
		return
	}
	pkgupperLine, pkgupper := exp.PreviousLine(), exp.m[1]

	if !exp.ExpectText(pkgupper + "_BUILDLINK3_MK:=") {
		return
	}
	exp.ExpectEmptyLine()

	// See pkgtools/createbuildlink/files/createbuildlink, keyword PKGUPPER
	ucPkgbase := strings.ToUpper(strings.Replace(pkgbase, "-", "_", -1))
	if ucPkgbase != pkgupper {
		pkgupperLine.Error2("Package name mismatch between multiple-inclusion guard %q (expected %q) ...", pkgupper, ucPkgbase)
		pkgbaseLine.Error1("... and package name %q.", pkgbase)
	}
	if G.Pkg != nil {
		if mkbase := G.Pkg.EffectivePkgbase; mkbase != "" && mkbase != pkgbase {
			pkgbaseLine.Error1("Package name mismatch between %q in this file ...", pkgbase)
			G.Pkg.EffectivePkgnameLine.Line.Error1("... and %q from the package Makefile.", mkbase)
		}
	}

	// Third paragraph: Package information.
	indentLevel := 1 // The first .if is from the second paragraph.
	for {
		if exp.EOF() {
			exp.CurrentLine().Warn0("Expected .endif")
			return
		}

		line := exp.CurrentLine()
		mkline := mklines.mklines[exp.index]

		if mkline.IsVarassign() {
			exp.Advance()
			varname, value := mkline.Varname(), mkline.Value()
			doCheck := false

			const (
				reDependencyCmp      = `^((?:\$\{[\w_]+\}|[\w_\.+]|-[^\d])+)[<>]=?(\d[^-*?\[\]]*)$`
				reDependencyWildcard = `^(-(?:\[0-9\]\*|\d[^-]*)$`
			)

			if varname == "BUILDLINK_ABI_DEPENDS."+pkgbase {
				abiLine = line
				parser := NewParser(value)
				if dp := parser.Dependency(); dp != nil && parser.EOF() {
					abi = dp
				} else {
					line.Warn1("Unknown dependency pattern %q.", value)
				}
				doCheck = true
			}
			if varname == "BUILDLINK_API_DEPENDS."+pkgbase {
				apiLine = line
				parser := NewParser(value)
				if dp := parser.Dependency(); dp != nil && parser.EOF() {
					api = dp
				} else {
					line.Warn1("Unknown dependency pattern %q.", value)
				}
				doCheck = true
			}
			if doCheck && abi != nil && api != nil && abi.pkgbase != api.pkgbase && !hasPrefix(api.pkgbase, "{") {
				abiLine.Warn1("Package name mismatch between ABI %q ...", abi.pkgbase)
				apiLine.Warn1("... and API %q.", api.pkgbase)
			}
			if doCheck {
				if abi != nil && abi.lower != "" && !containsVarRef(abi.lower) {
					if api != nil && api.lower != "" && !containsVarRef(api.lower) {
						if pkgverCmp(abi.lower, api.lower) < 0 {
							abiLine.Warn1("ABI version %q should be at least ...", abi.lower)
							apiLine.Warn1("... API version %q.", api.lower)
						}
					}
				}
			}

			if varparam := mkline.Varparam(); varparam != "" && varparam != pkgbase {
				if hasPrefix(varname, "BUILDLINK_") && mkline.Varcanon() != "BUILDLINK_API_DEPENDS.*" {
					line.Warn2("Only buildlink variables for %q, not %q may be set in this file.", pkgbase, varparam)
				}
			}

			if varname == "pkgbase" {
				exp.advanceIfMatches(`^\.\s*include "../../mk/pkg-build-options\.mk"$`)
			}

		} else if exp.AdvanceIfEquals("") || exp.AdvanceIfPrefix("#") {
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
				exp.CurrentLine().Warn0("Unchecked line in third paragraph.")
			}
			exp.Advance()
		}
	}
	if apiLine == nil {
		exp.CurrentLine().Warn0("Definition of BUILDLINK_API_DEPENDS is missing.")
	}
	exp.ExpectEmptyLine()

	// Fourth paragraph: Cleanup, corresponding to the first paragraph.
	if !exp.advanceIfMatches(`^BUILDLINK_TREE\+=\s*-` + regexp.QuoteMeta(pkgbase) + `$`) {
		exp.CurrentLine().Warn0("Expected BUILDLINK_TREE line.")
	}

	if !exp.EOF() {
		exp.CurrentLine().Warn0("The file should end here.")
	}

	if G.Pkg != nil {
		G.Pkg.checklinesBuildlink3Inclusion(mklines)
	}

	SaveAutofixChanges(mklines.lines)
}
