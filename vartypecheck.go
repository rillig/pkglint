package main

	var deprecatedLicenses = map[string]bool  {
		"fee-based-commercial-use":true,
		"no-commercial-use":true,
		"no-profit":true,
		"no-redistribution":true,
		"shareware":true,
	}

func CheckvartypeLicense(line *Line, varname, varvalue string) {
	licenses := parseLicenses(varvalue)
	for _, license := range licenses {
		licenseFile := *GlobalVars.cwdPkgsrcdir + "/licenses/" + license
		if licenseFileLine := GlobalVars.pkgContext.vardef["LICENSE_FILE"]; licenseFileLine != nil {
			licenseFile = GlobalVars.currentDir + "/" + resolveVarsInRelativePath(licenseFileLine.get("value"), false)
		} else {
			GlobalVars.ipcUsedLicenses[license] = true
		}

		if !fileExists(licenseFile) {
			line.logWarning("License file " + normalizePathname(licenseFile) + " does not exist.")
		}

		if deprecatedLicenses[license] {
			line.logWarning("License " + license + " is deprecated.")
		}
	}
}
