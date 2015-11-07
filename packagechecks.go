package main

func checkpackagePossibleDowngrade() {
	_ = GlobalVars.opts.optDebugTrace && logDebug(NO_FILE, NO_LINES, "checkpackagePossibleDowngrade")

	m, _, pkgversion := match2(*GlobalVars.pkgContext.effective_pkgname, rePkgname)
	if !m {
		return
	}

	line := *GlobalVars.pkgContext.effective_pkgname_line

	change := GlobalVars.globalData.lastChange[*GlobalVars.pkgContext.pkgpath]
	if change == nil {
		_ = GlobalVars.opts.optDebugMisc && line.logDebug("No change log for package %v", GlobalVars.pkgContext.pkgpath)
		return
	}

	if change.action == "Updated" {
		if pkgverCmp(pkgversion, "<", change.version) {
			line.logWarning("The package is being downgraded from %v to %v", change.version, pkgversion)
		}
	}
}
