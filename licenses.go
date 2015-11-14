package main

import (
	"io/ioutil"
	"path"
	"strings"
)

func parseLicenses(licenses string) []string {
	noPerl := strings.Replace(licenses, "${PERL5_LICENSE}", "gnu-gpl-v2 OR artistic", -1)
	noOps := reCompile(`[()]|AND|OR`).ReplaceAllString(noPerl, "") // cheated
	return splitOnSpace(noOps)
}

func checktoplevelUnusedLicenses() {
	if G.ipcUsedLicenses == nil {
		return
	}

	licensedir := G.globalData.pkgsrcdir + "/licenses"
	files, _ := ioutil.ReadDir(licensedir)
	for _, licensefile := range files {
		licensename := licensefile.Name()
		licensepath := licensedir + "/" + licensename
		if fileExists(licensepath) {
			if !G.ipcUsedLicenses[licensename] {
				logWarning(licensepath, NO_LINES, "This license seems to be unused.")
			}
		}
	}
}

func checklineLicense(line *Line, value string) {
	licenses := parseLicenses(value)
	for _, license := range licenses {
		licenseFile := G.globalData.pkgsrcdir + "/licenses/" + license
		if licenseFileLine := G.pkgContext.vardef["LICENSE_FILE"]; licenseFileLine != nil {
			licenseFile = G.currentDir + "/" + resolveVarsInRelativePath(licenseFileLine.extra["value"].(string), false)
		} else if G.ipcUsedLicenses != nil {
			G.ipcUsedLicenses[license] = true
		}

		if !fileExists(licenseFile) {
			line.logWarning("License file %s does not exist.", path.Clean(licenseFile))
		}

		switch license {
		case "fee-based-commercial-use",
			"no-commercial-use",
			"no-profit",
			"no-redistribution",
			"shareware":
			line.logWarning("License %s is deprecated.", license)
		}
	}
}
