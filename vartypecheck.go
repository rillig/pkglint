package main

import (
	"strings"
)

type CheckVartype struct {
	line       *Line
	varname    string
	op         string
	value      string
	valueNovar string
}

func (cv *CheckVartype) License() {
	licenses := parseLicenses(cv.value)
	for _, license := range licenses {
		licenseFile := *GlobalVars.cwdPkgsrcdir + "/licenses/" + license
		if licenseFileLine := GlobalVars.pkgContext.vardef["LICENSE_FILE"]; licenseFileLine != nil {
			licenseFile = GlobalVars.currentDir + "/" + resolveVarsInRelativePath(licenseFileLine.extra["value"].(string), false)
		} else {
			GlobalVars.ipcUsedLicenses[license] = true
		}

		if !fileExists(licenseFile) {
			cv.line.logWarning("License file %s does not exist.", normalizePathname(licenseFile))
		}

		switch license {
		case "fee-based-commercial-use",
			"no-commercial-use",
			"no-profit",
			"no-redistribution",
			"shareware":
			cv.line.logWarning("License %s is deprecated.", license)
		}
	}
}

func (cv *CheckVartype) MailAddress() {
	line, value := cv.line, cv.value

	if m := match(value, `^([+\-.0-9A-Z_a-z]+)\@([-\w\d.]+)$`); m != nil {
		_, domain := m[1], m[2]

		if strings.EqualFold(domain, "NetBSD.org") && domain != "NetBSD.org" {
			line.logWarning("Please write NetBSD.org instead of %q.", domain)
		}
		if match(value, `(?i)^(tech-pkg|packages)\@NetBSD\.org$`) != nil {
			line.logError("This mailing list address is obsolete. Use pkgsrc-users@NetBSD.org instead.")
		}

	} else {
		line.logWarning("\"%s\" is not a valid mail address.", value)
	}
}

func (cv *CheckVartype) Message() {
	line, varname, value := cv.line, cv.varname, cv.value

	if match(value, `^[\"'].*[\"']$`) != nil {
		line.logWarning("%s should not be quoted.", varname)
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

func (cv *CheckVartype) Option() {
	line, value, valueNovar := cv.line, cv.value, cv.valueNovar

	if value != valueNovar {
		_ = GlobalVars.opts.optDebugUnchecked && line.logDebug("Unchecked option name: %q", value)
		return
	}

	if m := match(value, `^-?([a-z][-0-9a-z\+]*)$`); m != nil {
		optname := m[1]

		if GlobalVars.globalData.pkgOptions[optname] == "" {
			line.logWarning("Unknown option \"%s\".", optname)
			line.explainWarning(
				"This option is not documented in the mk/defaults/options.description",
				"file. If this is not a typo, please think of a brief but precise",
				"description and either update that file yourself or ask on the",
				"tech-pkg@NetBSD.org mailing list.")
		}
		return
	}

	if match(value, `^-?([a-z][-0-9a-z_\+]*)$`) != nil {
		line.logWarning("Use of the underscore character in option names is deprecated.")
		return
	}

	line.logError("Invalid option name.")
}

func (cv *CheckVartype) PrefixPathname() {
	if m := match(cv.value, `^man/(.*)`); m != nil {
		cv.line.logWarning("Please use \"${PKGMANDIR}/%s\" instead of \"%s\".", m[1], cv.value)
	}
}

func (cv *CheckVartype) AwkCommand() {
	_ = G.opts.optDebugUnchecked && cv.line.logDebug("Unchecked AWK command: %q", cv.value)
}

func (cv *CheckVartype) BrokenIn() {
	if !match0(cv.value, `^pkgsrc-20\d\d\dQ[1234]$`) {
		cv.line.logWarning("Invalid value %q for %s.", cv.value, cv.varname)
	}
}

func (cv *CheckVartype) BuildlinkDepmethod() {
	if !match0(cv.value, reUnresolvedVar) && cv.value != "build" && cv.value != "full" {
		cv.line.logWarning("Invalid dependency method %q. Valid methods are \"build\" or \"full\".", cv.value)
	}
}

func (cv *CheckVartype) BuildlinkDepth() {
	if (cv.op != "use" || cv.value != "+") &&
		cv.value != "${BUILDLINK_DEPTH}+" &&
		cv.value != "${BUILDLINK_DEPTH:S/+$//}" {
		cv.line.logWarning("Invalid value.")
	}
}

func (cv *CheckVartype) Category() {
	switch cv.value {
	case "archivers", "audio",
		"benchmarks", "biology",
		"cad", "chat", "chinese", "comms", "converters", "cross", "crosspkgtools",
		"databases", "devel",
		"editors", "emulators",
		"filesystems", "finance", "fonts",
		"games", "geography", "gnome", "gnustep", "graphics",
		"ham",
		"inputmethod",
		"japanese", "java",
		"kde", "korean",
		"lang", "linux", "local",
		"mail", "math", "mbone", "meta-pkgs", "misc", "multimedia",
		"net", "news",
		"packages", "parallel", "perl5", "pkgtools", "plan9", "print", "python",
		"ruby",
		"scm", "security", "shells", "sysutils",
		"tcl", "textproc", "time", "tk",
		"windowmaker", "wm", "www",
		"x11", "xmms":
	default:
		cv.line.logError("Invalid category %q.", cv.value)
	}
}

func (cv *CheckVartype) CFlag() {
	line, value := cv.line, cv.value

	if match0(value, `^(-[DILOUWfgm].|-std=`) {
		return
	}
	if value == "-c99" {
		return // Only useful for the IRIX C compiler
	}
	if strings.HasPrefix(value, "-") {
		line.logWarning("Unknown compiler flag %q.", value)
		return
	}
	if !match0(value, reUnresolvedVar) {
		line.logWarning("Compiler flag %q should start with a hyphen.")
	}
}
