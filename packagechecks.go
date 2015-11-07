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
