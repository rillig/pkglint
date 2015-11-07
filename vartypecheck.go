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

func (cv *CheckVartype) Comment() {
	line, value := cv.line, cv.value

	if value == "SHORT_DESCRIPTION_OF_THE_PACKAGE" {
		line.logError("COMMENT must be set.")
	}
	if m, first := match1(value, `^(?i)(a|an)\s`); m {
		line.logWarning("COMMENT should not begin with %q.", first)
	}
	if match0(value, `^[a-z]`) {
		line.logWarning("COMMENT should start with a capital letter.")
	}
	if strings.HasSuffix(value, ".") {
		line.logWarning("COMMENT should not end with a period.")
	}
	if len(value) > 70 {
		line.logWarning("COMMENT should not be longer than 70 characters.")
	}
}

func (cv *CheckVartype) Dependency() {
	line, value := cv.line, cv.value

	if m, depbase, depop, depversion := match3(value, `^(`+rePkgbase+`)(<|=|>|<=|>=|!=|-)(`+rePkgversion+`)$`); m {
		_, _, _ = depbase, depop, depversion
		return
	}

	if m, depbase, bracket, version, versionWildcard, other := match5(value, `^(`+rePkgbase+`)-(?:\[(.*)\]\*|(\d+(?:\.\d+)*(?:\.\*)?)(\{,nb\*\}|\*|)|(.*))?$`); m {
		if bracket != "" {
			if bracket != "0-9" {
				line.logWarning("Only [0-9]* is allowed in the numeric part of a dependency.")
			}

		} else if version != "" && versionWildcard != "" {
			// Fine.

		} else if version != "" {
			line.logWarning("Please append \"{,nb*}\" to the version number of this dependency.")
			line.explainWarning(
				"Usually, a dependency should stay valid when the PKGREVISION is",
				"increased, since those changes are most often editorial. In the",
				"current form, the dependency only matches if the PKGREVISION is",
				"undefined.")

		} else if other == "*" {
			line.logWarning("Please use %s-[0-9]* instead of %s-*.", depbase, depbase)
			line.explainWarning(
				"If you use a * alone, the package specification may match other",
				"packages that have the same prefix, but a longer name. For example,",
				"foo-* matches foo-1.2, but also foo-client-1.2 and foo-server-1.2.")

		} else {
			line.logError("Unknown dependency pattern %q.", value)
		}
		return
	}

	if strings.Contains(value, "{") {
		// No check yet for alternative dependency patterns.
		_ = G.opts.optDebugUnchecked && line.logDebug("Unchecked alternative dependency pattern: %s", value)

	} else if value != cv.valueNovar {
		_ = G.opts.optDebugUnchecked && line.logDebug("Unchecked dependency: %s", value)

	} else {
		line.logWarning("Unknown dependency format: %s", value)
		line.explainWarning(
			"Typical dependencies have the following forms:",
			"",
			"* package>=2.5",
			"* package-[0-9]*",
			"* package-3.141")
	}
}

func (cv *CheckVartype) DependencyWithPath() {
	line, value := cv.line, cv.value
	if value != cv.valueNovar {
		return // It's probably not worth checking this.
	}

	if m, pattern, relpath, _, pkg := match4(value, `(.*):(\.\./\.\./([^/]+)/([^/]+))$`); m {
		checklineRelativePkgdir(line, relpath)

		if pkg == "msgfmt" || pkg == "gettext" {
			line.logWarning("Please use USE_TOOLS+=msgfmt instead of this dependency.")
		} else if pkg == "perl5" {
			line.logWarning("Please use USE_TOOLS+=perl:run instead of this dependency.")

		} else if pkg == "gmake" {
			line.logWarning("Please use USE_TOOLS+=gmake instead of this dependency.")
		}

		if !match0(pattern, reDependencyCmp) && !match0(pattern, reDependencyWildcard) {
			line.logError("Unknown dependency pattern %q.", pattern)
		}
		return
	}

	if match0(value, `:\.\./[^/]+$`) {
		line.logWarning("Dependencies should have the form \"../../category/package\".")
		line.explainWarning(explanationRelativeDirs()...)
		return
	}

	line.logWarning("Unknown dependency format.")
	line.explainWarning(
		"Examples for valid dependencies are:",
		"  package-[0-9]*:../../category/package",
		"  package>=3.41:../../category/package",
		"  package-2.718:../../category/package")
}

func (cv *CheckVartype) DistSuffix() {
	if cv.value == ".tar.gz" {
		line.logNote("%s is \".tar.gz\" by default, so this definition may be redundant.",cv.varname)
	}
}

func (cv *CheckVartype) EmulPlatform() {
	
			if (value =~ m"^(\w+)-(\w+)$") {
				my (opsys, arch) = (1, 2)

				if (opsys !~ m"^(?:bsdos|cygwin|darwin|dragonfly|freebsd|haiku|hpux|interix|irix|linux|netbsd|openbsd|osf1|sunos|solaris)$") {
					line.logWarning("Unknown operating system: ${opsys}")
				}
				// no check for os_version
				if (arch !~ m"^(?:i386|alpha|amd64|arc|arm|arm32|cobalt|convex|dreamcast|hpcmips|hpcsh|hppa|ia64|m68k|m88k|mips|mips64|mipsel|mipseb|mipsn32|ns32k|pc532|pmax|powerpc|rs6000|s390|sparc|sparc64|vax|x86_64)$") {
					line.logWarning("Unknown hardware architecture: ${arch}")
				}

			} else {
				line.logWarning("\"${value}\" is not a valid emulation platform.")
				line.explainWarning(
"An emulation platform has the form <OPSYS>-<MACHINE_ARCH>.",
"OPSYS is the lower-case name of the operating system, and MACHINE_ARCH",
"is the hardware architecture.",
"",
"Examples: linux-i386, irix-mipsel.")
			}
}
func (cv *CheckVartype) FetchURL() {
			checkline_mk_vartype_basic(line, varname, "URL", op, value, comment, list_context, is_guessed)

			my sites = get_dist_sites()
			foreach my site (keys(%{sites})) {
				if (index(value, site) == 0) {
					my subdir = substr(value, length(site))
					my is_github = value =~ m"^https://github\.com/"
					if (is_github) {
						subdir =~ s|/.*|/|
					}
					line.logWarning(sprintf("Please use \${%s:=%s} instead of \"%s\".", sites.{site}, subdir, value))
					if (is_github) {
						line.logWarning("Run \"".conf_make." help topic=github\" for further tips.")
					}
					last
				}
			}
}
func (cv *CheckVartype) Filename() {
			if (value_novar =~ m"/") {
				line.logWarning("A filename should not contain a slash.")

			} else if (value_novar !~ m"^[-0-9\@A-Za-z.,_~+%]*$") {
				line.logWarning("\"${value}\" is not a valid filename.")
			}
}
func (cv *CheckVartype) Filemask() {
			if (value_novar !~ m"^[-0-9A-Za-z._~+%*?]*$") {
				line.logWarning("\"${value}\" is not a valid filename mask.")
			}
}
func (cv *CheckVartype) FileMode() {
			if (value ne "" && value_novar eq "") {
				// Fine.
			} else if (value =~ m"^[0-7]{3,4}") {
				// Fine.
			} else {
				line.logWarning("Invalid file mode ${value}.")
			}
}
func (cv *CheckVartype) Identifier() {
			if (value ne value_novar) {
				//line.logWarning("Identifiers should be given directly.")
			}
			if (value_novar =~ m"^[+\-.0-9A-Z_a-z]+$") {
				// Fine.
			} else if (value ne "" && value_novar eq "") {
				// Don't warn here.
			} else {
				line.logWarning("Invalid identifier \"${value}\".")
			}
}
func (cv *CheckVartype) Integer() {
			if (value !~ m"^\d+$") {
				line.logWarning("${varname} must be a valid integer.")
			}
}
func (cv *CheckVartype) LdFlag() {
			if (value =~ m"^-L(.*)") {
				my (dirname) = (1)

				opt_debug_unchecked and line.logDebug("Unchecked directory ${dirname} in ${varname}.")

			} else if (value =~ m"^-l(.*)") {
				my (libname) = (1)

				opt_debug_unchecked and line.logDebug("Unchecked library name ${libname} in ${varname}.")

			} else if (value =~ m"^(?:-static)$") {
				// Assume that the wrapper framework catches these.

			} else if (value =~ m"^(-Wl,(?:-R|-rpath|--rpath))") {
				my (rpath_flag) = (1)
				line.logWarning("Please use \${COMPILER_RPATH_FLAG} instead of ${rpath_flag}.")

			} else if (value =~ m"^-.*") {
				line.logWarning("Unknown linker flag \"${value}\".")

			} else if (value =~ regex_unresolved) {
				opt_debug_unchecked and line.logDebug("Unchecked LDFLAG: ${value}")

			} else {
				line.logWarning("Linker flag \"${value}\" does not start with a dash.")
			}
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

func (cv *CheckVartype) Pathlist() {
			if (value !~ m":" && is_guessed) {
				checkline_mk_vartype_basic(line, varname, "Pathname", op, value, comment, list_context, is_guessed)

			} else {

				// XXX: The splitting will fail if value contains any
				// variables with modifiers, for example :Q or :S/././.
				foreach my p (split(qr":", value)) {
					my p_novar = remove_variables(p)

					if (p_novar !~ m"^[-0-9A-Za-z._~+%/]*$") {
						line.logWarning("\"${p}\" is not a valid pathname.")
					}

					if (p !~ m"^[\$/]") {
						line.logWarning("All components of ${varname} (in this case \"${p}\") should be an absolute path.")
					}
				}
			}
}
func (cv *CheckVartype) Pathmask() {
			if (value_novar !~ m"^[#\-0-9A-Za-z._~+%*?/\[\]]*$") {
				line.logWarning("\"${value}\" is not a valid pathname mask.")
			}
			checkline_mk_absolute_pathname(line, value)
}
func (cv *CheckVartype) Pathname() {
			if (value_novar !~ m"^[#\-0-9A-Za-z._~+%/]*$") {
				line.logWarning("\"${value}\" is not a valid pathname.")
			}
			checkline_mk_absolute_pathname(line, value)
}
func (cv *CheckVartype) Perl5Packlist() {
			if (value ne value_novar) {
				line.logWarning("${varname} should not depend on other variables.")
			}
}
func (cv *CheckVartype) PkgName() {if (value eq value_novar && value !~ regex_pkgname) {
				line.logWarning("\"${value}\" is not a valid package name. A valid package name has the form packagename-version, where version consists only of digits, letters and dots.")
			}
}
func (cv *CheckVartype) PkgPath() {
			checkline_relative_pkgdir(line, "cur_pkgsrcdir/value")
}
