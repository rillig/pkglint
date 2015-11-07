package main

func checkpackagePossibleDowngrade() {
	_ = G.opts.optDebugTrace && logDebug(NO_FILE, NO_LINES, "checkpackagePossibleDowngrade")

	m, _, pkgversion := match2(*G.pkgContext.effective_pkgname, rePkgname)
	if !m {
		return
	}

	line := *G.pkgContext.effective_pkgname_line

	change := G.globalData.lastChange[*G.pkgContext.pkgpath]
	if change == nil {
		_ = G.opts.optDebugMisc && line.logDebug("No change log for package %v", G.pkgContext.pkgpath)
		return
	}

	if change.action == "Updated" {
		if pkgverCmp(pkgversion, "<", change.version) {
			line.logWarning("The package is being downgraded from %v to %v", change.version, pkgversion)
		}
	}
}

func checklinesBuildlink3Inclusion(lines []*Line) {
	_ = G.opts.optDebugTrace && logDebug(lines[0].fname, NO_LINES, "checklinesBuildlink3Inclusion()")

	if G.pkgContext == nil {
		return
	}

	// Collect all the included buildlink3.mk files from the file.
	includedFiles := make(map[string]*Line)
	for _, line := range lines {
		if m, _, file, _ := match3(line.text, reMkInclude); m {
			if m, bl3 := match1(file, `^\.\./\.\./(.*)/buildlink3\.mk`); m {
				includedFiles[bl3] = line
				if G.pkgContext.bl3[bl3] == nil {
					line.logWarning("%s/buildlink3.mk is included by this file but not by the package.", bl3)
				}
			}
		}
	}

	// Print debugging messages for all buildlink3.mk files that are
	// included by the package but not by this buildlink3.mk file.
	for packageBl3, line := range G.pkgContext.bl3 {
		if includedFiles[packageBl3] == nil {
			_ = G.opts.optDebugMisc && line.logDebug("%s/buildlink3.mk is included by the package but not by the buildlink3.mk file.", packageBl3)
		}
	}
}
