package main

import (
	"regexp"
	"strings"
)

func checkfileBuildlink3Mk(fname string) {
	trace("checkfileBuildlink3Mk", fname)

	checkperms(fname)

	lines, err := loadLines(fname, true)
	if err != nil {
		logError(fname, NO_LINES, "Cannot be read.")
		return
	}
	if len(lines) == 0 {
		logError(fname, NO_LINES, "Must not be empty.")
		return
	}

	parselinesMk(lines)
	checklinesMk(lines)

	exp := &ExpectContext{lines, 0}

	for exp.advanceIfMatches(`^#`) != nil {
		if hasPrefix(exp.previousLine().text, "# XXX") {
			exp.previousLine().logNote("Please read this comment and remove it if appropriate.")
		}
	}

	exp.expectEmptyLine()

	if exp.advanceIfMatches(`^BUILDLINK_DEPMETHOD\.(\S+)\?=.*$`) != nil {
		exp.previousLine().logWarning("This line belongs inside the .ifdef block.")
		for exp.advanceIfMatches(`^$`) != nil {
		}
	}

	pkgbaseLine, pkgbase := (*Line)(nil), ""
	pkgidLine, pkgid := exp.currentLine(), ""
	abiLine, abiPkg, abiVersion := (*Line)(nil), "", ""
	apiLine, apiPkg, apiVersion := (*Line)(nil), "", ""

	// First paragraph: Introduction of the package identifier
	if m := exp.advanceIfMatches(`^BUILDLINK_TREE\+=\s*(\S+)$`); m != nil {
		pkgid = m[1]
	} else {
		exp.currentLine().logWarning("Expected a BUILDLINK_TREE line.")
		return
	}
	exp.expectEmptyLine()

	// Second paragraph: multiple inclusion protection and introduction
	// of the uppercase package identifier.
	if m := exp.advanceIfMatches(`^\.if !defined\((\S+)_BUILDLINK3_MK\)$`); m != nil {
		pkgbaseLine = exp.previousLine()
		pkgbase = m[1]
	} else {
		return
	}
	if !exp.expectText(pkgbase + "_BUILDLINK3_MK:=") {
		exp.currentLine().logError("Expected the multiple-inclusion guard.")
		return
	}
	exp.expectEmptyLine()

	ucPkgid := strings.ToUpper(strings.Replace(pkgid, "-", "_", -1))
	if ucPkgid != pkgbase {
		pkgbaseLine.logError("Package name mismatch between %q ...", pkgbase)
		pkgidLine.logError("... and %q.", pkgid)
	}
	if G.pkgContext != nil && G.pkgContext.effective_pkgbase != nil {
		if mkbase := *G.pkgContext.effective_pkgbase; mkbase != "" && mkbase != pkgid {
			pkgidLine.logError("Package name mismatch between %q ...", pkgid)
			G.pkgContext.effective_pkgname_line.logError("... and %q.", mkbase)
		}
	}

	// Third paragraph: Package information.
	indentLevel := 1 // The first .if is from the second paragraph.
	for {
		if exp.eof() {
			exp.currentLine().logWarning("Expected .endif")
			return
		}

		line := exp.currentLine()

		if m := exp.advanceIfMatches(reVarassign); m != nil {
			varname, value := m[1], m[3]
			doCheck := false

			if varname == "BUILDLINK_ABI_DEPENDS."+pkgid {
				abiLine = line
				if m, p, v := match2(value, reDependencyCmp); m {
					abiPkg, abiVersion = p, v
				} else if m, p := match1(value, reDependencyWildcard); m {
					abiPkg, abiVersion = p, ""
				} else {
					_ = G.opts.optDebugUnchecked && line.logDebug("Unchecked dependency pattern %q.", value)
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
					_ = G.opts.optDebugUnchecked && line.logDebug("Unchecked dependency pattern %q.", value)
				}
				doCheck = true
			}
			if doCheck && abiPkg != "" && apiPkg != "" && abiPkg != apiPkg {
				abiLine.logWarning("Package name mismatch between %q ...", abiPkg)
				apiLine.logWarning("... and %q.", apiPkg)
			}
			if doCheck && abiVersion != "" && apiVersion != "" && !pkgverCmp(abiVersion, ">=", apiVersion) {
				abiLine.logWarning("ABI version (%s) should be at least ...", abiVersion)
				apiLine.logWarning("... API version (%s).", apiVersion)
			}

			if m, varparam := match1(varname, `^BUILDLINK_[\w_]+\.(.*)$`); m {
				if varparam != pkgid {
					line.logWarning("Only buildlink variables for %q, not %q may be set in this file.", pkgid, varparam)
				}
			}

			if varname == "pkgbase" {
				exp.advanceIfMatches(`^\.\s*include "../../mk/pkg-build-options\.mk"$`)
			}

		} else if exp.advanceIfMatches(`^(?:#.*)?$`) != nil {
			// Comments and empty lines are fine here.

		} else if exp.advanceIfMatches(`^\.\s*include "\.\./\.\./([^/]+/[^/]+)/buildlink3\.mk"$`) != nil ||
			exp.advanceIfMatches(`^\.\s*include "\.\./\.\./mk/(\S+)\.buildlink3\.mk"$`) != nil {
			// TODO: Maybe check dependency lines.

		} else if exp.advanceIfMatches(`^\.if\s`) != nil {
			indentLevel++

		} else if exp.advanceIfMatches(`^\.endif.*$`) != nil {
			indentLevel--
			if indentLevel == 0 {
				break
			}

		} else {
			_ = G.opts.optDebugUnchecked && exp.currentLine().logWarning("Unchecked line in third paragraph.")
			exp.advance()
		}
	}
	if apiLine == nil {
		exp.currentLine().logWarning("Definition of BUILDLINK_API_DEPENDS is missing.")
	}
	exp.expectEmptyLine()

	// Fourth paragraph: Cleanup, corresponding to the first paragraph.
	exp.advanceIfMatches(`^BUILDLINK_TREE\+=\s*-` + regexp.QuoteMeta(pkgbase) + `$`)

	if !exp.eof() {
		exp.currentLine().logWarning("The file should end here.")
	}

	checklinesBuildlink3Inclusion(lines)
}
