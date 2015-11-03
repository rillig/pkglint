package main

import (
	"strings"
)

var deprecatedLicenses = map[string]bool{
	"fee-based-commercial-use": true,
	"no-commercial-use":        true,
	"no-profit":                true,
	"no-redistribution":        true,
	"shareware":                true,
}

func CheckvartypeLicense(line *Line, varname, value string) {
	licenses := parseLicenses(value)
	for _, license := range licenses {
		licenseFile := *GlobalVars.cwdPkgsrcdir + "/licenses/" + license
		if licenseFileLine := GlobalVars.pkgContext.vardef["LICENSE_FILE"]; licenseFileLine != nil {
			licenseFile = GlobalVars.currentDir + "/" + resolveVarsInRelativePath(licenseFileLine.extra["value"].(string), false)
		} else {
			GlobalVars.ipcUsedLicenses[license] = true
		}

		if !fileExists(licenseFile) {
			line.logWarningF("License file %s does not exist.", normalizePathname(licenseFile))
		}

		if deprecatedLicenses[license] {
			line.logWarningF("License %s is deprecated.", license)
		}
	}
}

func CheckvartypeMailAddress(line *Line, value string) {
	if m := match(value, `^([+\-.0-9A-Z_a-z]+)\@([-\w\d.]+)$`); m != nil {
		_, domain := m[1], m[2]

		if strings.EqualFold(domain, "NetBSD.org") && domain != "NetBSD.org" {
			line.logWarningF("Please write NetBSD.org instead of %q.", domain)
		}
		if match(value, `(?i)^(tech-pkg|packages)\@NetBSD\.org$`) != nil {
			line.logErrorF("This mailing list address is obsolete. Use pkgsrc-users@NetBSD.org instead.")
		}

	} else {
		line.logWarningF("\"%s\" is not a valid mail address.", value)
	}
}

func CheckvartypeMessage(line *Line, varname, value string) {
	if match(value, `^[\"'].*[\"']$`) != nil {
		line.logWarningF("%s should not be quoted.", varname)
		line.explainWarning(
			"The quoting is only needed for variables which are interpreted as",
			"multiple words (or, generally speaking, a list of something). A single",
			"text message does not belong to this class, since it is only printed",
			"as a whole.",
			"",
			"On the other hand, PKG_FAIL_REASON is a _list_ of text messages, so in",
			"that case, the quoting has to be done.`")
	}
}

func CheckvartypeOption(line *Line, varvalue string, varvalueNovar string) {
	if varvalue != varvalueNovar {
		_ = GlobalVars.opts.optDebugUnchecked && line.logDebugF("Unchecked option name: %q", varvalue)
		return
	}

	if m := match(varvalue, `^-?([a-z][-0-9a-z\+]*)$`); m != nil {
		optname := m[1]

		if GlobalVars.globalData.pkgOptions[optname] == "" {
			line.logWarningF("Unknown option \"%s\".", optname)
			line.explainWarning(
				"This option is not documented in the mk/defaults/options.description",
				"file. If this is not a typo, please think of a brief but precise",
				"description and either update that file yourself or ask on the",
				"tech-pkg@NetBSD.org mailing list.")
		}
		return
	}

	if match(varvalue, `^-?([a-z][-0-9a-z_\+]*)$`) != nil {
		line.logWarningF("Use of the underscore character in option names is deprecated.")
		return
	}

	line.logErrorF("Invalid option name.")
}

func CheckvartypePrefixPathname(line *Line, value string) {
	if m := match(value, `^man/(.*)`); m != nil {
		line.logWarningF("Please use \"${PKGMANDIR}/%s\" instead of \"%s\".", m[1], value)
	}
}
