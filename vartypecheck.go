package main

import (
	"path"
	"strings"
)

type CheckVartype struct {
	line        *Line
	varname     string
	op          string
	value       string
	valueNovar  string
	comment     string
	listContext bool
	guessed     bool
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
	if hasPrefix(value, "-") {
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
	if hasSuffix(value, ".") {
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

	if contains(value, "{") {
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
		cv.line.logNote("%s is \".tar.gz\" by default, so this definition may be redundant.", cv.varname)
	}
}

func (cv *CheckVartype) EmulPlatform() {

	if m, opsys, arch := match2(cv.value, `^(\w+)-(\w+)$`); m {
		if !match0(opsys, `^(?:bsdos|cygwin|darwin|dragonfly|freebsd|haiku|hpux|interix|irix|linux|netbsd|openbsd|osf1|sunos|solaris)$`) {
			cv.line.logWarning("Unknown operating system: %s", opsys)
		}
		// no check for os_version
		if !match0(arch, `^(?:i386|alpha|amd64|arc|arm|arm32|cobalt|convex|dreamcast|hpcmips|hpcsh|hppa|ia64|m68k|m88k|mips|mips64|mipsel|mipseb|mipsn32|ns32k|pc532|pmax|powerpc|rs6000|s390|sparc|sparc64|vax|x86_64)"`) {
			cv.line.logWarning("Unknown hardware architecture: %s", arch)
		}

	} else {
		cv.line.logWarning("%q is not a valid emulation platform.", cv.value)
		cv.line.explainWarning(
			"An emulation platform has the form <OPSYS>-<MACHINE_ARCH>.",
			"OPSYS is the lower-case name of the operating system, and MACHINE_ARCH",
			"is the hardware architecture.",
			"",
			"Examples: linux-i386, irix-mipsel.")
	}
}

func (cv *CheckVartype) FetchURL() {
	checklineMkVartypeSimple(cv.line, cv.varname, "URL", cv.op, cv.value, cv.comment, cv.listContext, cv.guessed)

	for siteUrl, siteName := range G.globalData.masterSiteUrls {
		if hasPrefix(cv.value, siteUrl) {
			subdir := cv.value[len(siteUrl):]
			isGithub := hasPrefix(cv.value, "https://github.com/")
			if isGithub {
				subdir = strings.Split(subdir, "/")[0]
			}
			cv.line.logWarning("Please use ${%s:=%s} instead of %q.", siteName, subdir, cv.value)
			if isGithub {
				cv.line.logWarning("Run \"%s help topic=github\" for further tips.", confMake)
			}
			return
		}
	}
}

func (cv *CheckVartype) Filename() {
	if contains(cv.valueNovar, "/") {
		cv.line.logWarning("A filename should not contain a slash.")

	} else if !match0(cv.valueNovar, `^[-0-9\@A-Za-z.,_~+%]*$`) {
		cv.line.logWarning("%q is not a valid filename.", cv.value)
	}
}

func (cv *CheckVartype) Filemask() {
	if !match0(cv.valueNovar, `^[-0-9A-Za-z._~+%*?]*$`) {
		cv.line.logWarning("%q is not a valid filename mask.", cv.value)
	}
}

func (cv *CheckVartype) FileMode() {
	if cv.value != "" && cv.valueNovar == "" {
		// Fine.
	} else if match0(cv.value, `^[0-7]{3,4}`) {
		// Fine.
	} else {
		cv.line.logWarning("Invalid file mode %q.", cv.value)
	}
}

func (cv *CheckVartype) Identifier() {
	if cv.value != cv.valueNovar {
		//line.logWarning("Identifiers should be given directly.")
	}
	if match0(cv.valueNovar, `^[+\-.0-9A-Z_a-z]+$`) {
		// Fine.
	} else if cv.value != "" && cv.valueNovar == "" {
		// Don't warn here.
	} else {
		cv.line.logWarning("Invalid identifier %q.", cv.value)
	}
}

func (cv *CheckVartype) Integer() {
	if !match0(cv.value, `^\d+$`) {
		cv.line.logWarning("Invalid integer %q.")
	}
}

func (cv *CheckVartype) LdFlag() {
	if match0(cv.value, `^-[Ll]`) || cv.value == "-static" {
		return
	} else if m, rpathFlag := match1(cv.value, `^(-Wl,(?:-R|-rpath|--rpath))`); m {
		cv.line.logWarning("Please use ${COMPILER_RPATH_FLAG} instead of %s.", rpathFlag)

	} else if hasPrefix(cv.value, "-") {
		cv.line.logWarning("Unknown linker flag %q.", cv.value)

	} else if cv.value == cv.valueNovar {
		cv.line.logWarning("Linker flag %q does not start with a dash.", cv.value)
	}
}

func (cv *CheckVartype) License() {
	checklineLicense(cv.line, cv.value)
}

func (cv *CheckVartype) MailAddress() {
	line, value := cv.line, cv.value

	if m, _, domain := match2(value, `^([+\-.0-9A-Z_a-z]+)\@([-\w\d.]+)$`); m {
		if strings.EqualFold(domain, "NetBSD.org") && domain != "NetBSD.org" {
			line.logWarning("Please write NetBSD.org instead of %q.", domain)
		}
		if match0(value, `(?i)^(tech-pkg|packages)\@NetBSD\.org$`) {
			line.logError("This mailing list address is obsolete. Use pkgsrc-users@NetBSD.org instead.")
		}

	} else {
		line.logWarning("\"%s\" is not a valid mail address.", value)
	}
}

func (cv *CheckVartype) Message() {
	line, varname, value := cv.line, cv.varname, cv.value

	if match0(value, `^[\"'].*[\"']$`) {
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
		_ = G.opts.optDebugUnchecked && line.logDebug("Unchecked option name: %q", value)
		return
	}

	if m, optname := match1(value, `^-?([a-z][-0-9a-z\+]*)$`); m {
		if G.globalData.pkgOptions[optname] == "" {
			line.logWarning("Unknown option \"%s\".", optname)
			line.explainWarning(
				"This option is not documented in the mk/defaults/options.description",
				"file. If this is not a typo, please think of a brief but precise",
				"description and either update that file yourself or ask on the",
				"tech-pkg@NetBSD.org mailing list.")
		}
		return
	}

	if match0(value, `^-?([a-z][-0-9a-z_\+]*)$`) {
		line.logWarning("Use of the underscore character in option names is deprecated.")
		return
	}

	line.logError("Invalid option name.")
}

func (cv *CheckVartype) Pathlist() {
	if !contains(cv.value, ":") && cv.guessed {
		checklineMkVartypeSimple(cv.line, cv.varname, "Pathname", cv.op, cv.value, cv.comment, cv.listContext, cv.guessed)
		return
	}

	for _, path := range strings.Split(cv.value, ":") {
		pathNovar := removeVariableReferences(path)

		if !match0(pathNovar, `^[-0-9A-Za-z._~+%/]*$`) {
			cv.line.logWarning("%q is not a valid pathname.", path)
		}

		if !match0(path, `^[$/]`) {
			cv.line.logWarning("All components of %q (in this case %q) should be an absolute path.", cv.value, path)
		}
	}
}

func (cv *CheckVartype) Pathmask() {
	if !match0(cv.valueNovar, `^[#\-0-9A-Za-z._~+%*?/\[\]]*`) {
		cv.line.logWarning("%q is not a valid pathname mask.", cv.value)
	}
	checklineMkAbsolutePathname(cv.line, cv.value)
}

func (cv *CheckVartype) Pathname() {
	if !match0(cv.valueNovar, `^[#\-0-9A-Za-z._~+%/]*$`) {
		cv.line.logWarning("%q is not a valid pathname.", cv.value)
	}
	checklineMkAbsolutePathname(cv.line, cv.value)
}

func (cv *CheckVartype) Perl5Packlist() {
	if cv.value != cv.valueNovar {
		cv.line.logWarning("%s should not depend on other variables.", cv.varname)
	}
}

func (cv *CheckVartype) PkgName() {
	if cv.value == cv.valueNovar && !match0(cv.value, rePkgname) {
		cv.line.logWarning("%q is not a valid package name. A valid package name has the form packagename-version, where version consists only of digits, letters and dots.", cv.value)
	}
}

func (cv *CheckVartype) PkgPath() {
	checklineRelativePkgdir(cv.line, *G.curPkgsrcdir+"/"+cv.value)
}

func (cv *CheckVartype) PkgOptionsVar() {
	checklineMkVartypeSimple(cv.line, cv.varname, "Varname", cv.op, cv.value, cv.comment, false, cv.guessed)
	if !match0(cv.value, `\$\{PKGBASE[:\}]`) {
		cv.line.logError("PKGBASE must not be used in PKG_OPTIONS_VAR.")
		cv.line.explainError(
			"PKGBASE is defined in bsd.pkg.mk, which is included as the",
			"very last file, but PKG_OPTIONS_VAR is evaluated earlier.",
			"Use ${PKGNAME:C/-[0-9].*//} instead.")
	}
}

func (cv *CheckVartype) PkgRevision() {
	if !match0(cv.value, `^[1-9]\d*$`) {
		cv.line.logWarning("%s must be a positive integer number.", cv.varname)
	}
	if path.Base(cv.line.fname) != "Makefile" {
		cv.line.logError("%s only makes sense directly in the package Makefile.", cv.varname)
		cv.line.explainError(
			"Usually, different packages using the same Makefile.common have",
			"different dependencies and will be bumped at different times (e.g. for",
			"shlib major bumps) and thus the PKGREVISIONs must be in the separate",
			"Makefiles. There is no practical way of having this information in a",
			"commonly used Makefile.")
	}
}

func (cv *CheckVartype) PlatformTriple() {
	rePart := `(?:\[[^\]]+\]|[^-\[])+`
	reTriple := `^(` + rePart + `)-(` + rePart + `)-(` + rePart + `)$`
	if m, opsys, _, arch := match3(cv.value, reTriple); m {
		if !match0(opsys, `^(\*|BSDOS|Cygwin|Darwin|DragonFly|FreeBSD|Haiku|HPUX|Interix|IRIX|Linux|NetBSD|OpenBSD|OSF1|QNX|SunOS)$`) {
			cv.line.logWarning("Unknown operating system: %s", opsys)
		}
		// no check for os_version
		if !match0(arch, `^(\*|i386|alpha|amd64|arc|arm|arm32|cobalt|convex|dreamcast|hpcmips|hpcsh|hppa|ia64|m68k|m88k|mips|mips64|mipsel|mipseb|mipsn32|ns32k|pc532|pmax|powerpc|rs6000|s390|sparc|sparc64|vax|x86_64)$`) {
			cv.line.logWarning("Unknown hardware architecture: %s", arch)
		}

	} else {
		cv.line.logWarning("%q is not a valid platform triple.", cv.value)
		cv.line.explainWarning(
			"A platform triple has the form <OPSYS>-<OS_VERSION>-<MACHINE_ARCH>.",
			"Each of these components may be a shell globbing expression.",
			"Examples: NetBSD-*-i386, *-*-*, Linux-*-*.")
	}
}

func (cv *CheckVartype) PrefixPathname() {
	if m, mansubdir := match1(cv.value, `^man/(.+)`); m {
		cv.line.logWarning("Please use \"${PKGMANDIR}/%s\" instead of %q.", mansubdir, cv.value)
	}
}

func (cv *CheckVartype) PythonDependency() {
	if cv.value != cv.valueNovar {
		cv.line.logWarning("Python dependencies should not contain variables.")
	}
	if !match0(cv.valueNovar, `^[+\-.0-9A-Z_a-z]+(?:|:link|:build)$`) {
		cv.line.logWarning("Invalid Python dependency %q.", cv.value)
		cv.line.explainWarning(
			"Python dependencies must be an identifier for a package, as specified",
			"in lang/python/versioned_dependencies.mk. This identifier may be",
			"followed by :build for a build-time only dependency, or by :link for",
			"a run-time only dependency.")
	}
}

func (cv *CheckVartype) RelativePkgDir() {
	checklineRelativePkgdir(cv.line, cv.value)
}

func (cv *CheckVartype) RelativePkgPath() {
	checklineRelativePath(cv.line, cv.value, true)
}

func (cv *CheckVartype) Restricted() {
	if cv.value != "${RESTRICTED}" {
		cv.line.logWarning("The only valid value for %s is ${RESTRICTED}.", cv.varname)
		cv.line.explainWarning(
			"These variables are used to control which files may be mirrored on FTP",
			"servers or CD-ROM collections. They are not intended to mark packages",
			"whose only MASTER_SITES are on ftp.NetBSD.org.")
	}
}

func (cv *CheckVartype) SedCommand() {
}

func (cv *CheckVartype) SedCommands() {
	line := cv.line

	words := shellSplit(cv.value)
	if words == nil {
		line.logError("Invalid shell words in sed commands.")
		line.explainError(
			"If your sed commands have embedded \"#\" characters, you need to escape",
			"them with a backslash, otherwise make(1) will interpret them as a",
			"comment, no matter if they occur in single or double quotes or",
			"whatever.")

	} else {
		nwords := len(words)
		ncommands := 0

		for i := 0; i < nwords; i++ {
			word := words[i]
			checklineMkShellword(cv.line, word, true)

			if word == "-e" {
				if i+1 < nwords {
					// Check the real sed command here.
					i++
					ncommands++
					if ncommands > 1 {
						line.logWarning("Each sed command should appear in an assignment of its own.")
						line.explainWarning(
							"For example, instead of",
							"    SUBST_SED.foo+=        -e s,command1,, -e s,command2,,",
							"use",
							"    SUBST_SED.foo+=        -e s,command1,,",
							"    SUBST_SED.foo+=        -e s,command2,,",
							"",
							"This way, short sed commands cannot be hidden at the end of a line.")
					}
					checklineMkShellword(line, words[i-1], true)
					checklineMkShellword(line, words[i], true)
					checklineMkVartypeSimple(line, cv.varname, "SedCommand", cv.op, words[i], cv.comment, cv.listContext, cv.guessed)
				} else {
					line.logError("The -e option to sed requires an argument.")
				}
			} else if word == "-E" {
				// Switch to extended regular expressions mode.

			} else if word == "-n" {
				// Don't print lines per default.

			} else if i == 0 && match0(word, `^([\"']?)(?:\d*|/.*/)s(.).*\2g?\1$`) {
				line.logWarning("Please always use \"-e\" in sed commands, even if there is only one substitution.")

			} else {
				line.logWarning("Unknown sed command %q.", word)
			}
		}
	}
}

func (cv *CheckVartype) ShellCommand() {
	(&MkShellLine{cv.line}).checklineMkShelltext(cv.value)
}

func (cv *CheckVartype) ShellWord() {
	if !cv.listContext {
		checklineMkShellword(cv.line, cv.value, true)
	}
}

func (cv *CheckVartype) Stage() {
	if !match0(cv.value, `^(?:pre|do|post)-(?:extract|patch|configure|build|install)`) {
		cv.line.logWarning("Invalid stage name. Use one of {pre,do,post}-{extract,patch,configure,build,install}.")
	}
}

func (cv *CheckVartype) String() {
	// No further checks possible.
}

func (cv *CheckVartype) Tool() {
	if cv.varname == "TOOLS_NOOP" && cv.op == "+=" {
		// no warning for package-defined tool definitions

	} else if m, toolname, tooldep := match2(cv.value, `^([-\w]+|\[)(?::(\w+))?$`); m {
		if !G.globalData.tools[toolname] {
			cv.line.logError("Unknown tool %q.", toolname)
		}
		if tooldep != "" && !match0(tooldep, `^(bootstrap|build|pkgsrc|run)`) {
			cv.line.logError("Unknown tool dependency %q. Use one of \"build\", \"pkgsrc\" or \"run\".", tooldep)
		}
	} else {
		cv.line.logError("Invalid tool syntax: %q.", cv.value)
	}
}

func (cv *CheckVartype) Unchecked() {
	// Do nothing, as the name says.
}

func (cv *CheckVartype) URL() {
	line, value := cv.line, cv.value

	if value == "" && hasPrefix(cv.comment, "#") {
		// Ok

	} else if m, name, subdir := match2(value, `\$\{(MASTER_SITE_[^:]*).*:=(.*)\}$`); m {
		if !G.globalData.masterSiteVars[name] {
			line.logError("%s does not exist.", name)
		}
		if !hasSuffix(subdir, "/") {
			line.logError("The subdirectory in %s must end with a slash.", name)
		}

	} else if match0(value, reUnresolvedVar) {
		// No further checks

	} else if m, _, host, _, _ := match4(value, `^(https?|ftp|gopher)://([-0-9A-Za-z.]+)(?::(\d+))?/([-%&+,./0-9:=?\@A-Z_a-z~]|#)*$`); m {
		if match0(host, `(?i)\.NetBSD\.org$`) && !match0(host, `\.NetBSD\.org$`) {
			line.logWarning("Please write NetBSD.org instead of %s.", host)
		}

	} else if m, scheme, _, absPath := match3(value, `^([0-9A-Za-z]+)://([^/]+)(.*)$`); m {
		if scheme != "ftp" && scheme != "http" && scheme != "https" && scheme != "gopher" {
			line.logWarning("%q is not a valid URL. Only ftp, gopher, http, and https URLs are allowed here.", value)

		} else if absPath == "" {
			line.logNote("For consistency, please add a trailing slash to %q.", value)

		} else {
			line.logWarning("%q is not a valid URL.", value)
		}

	} else {
		line.logWarning("%q is not a valid URL.", value)
	}
}

func (cv *CheckVartype) UserGroupName() {
	if cv.value != cv.valueNovar {
		// No checks for now.
	} else if !match0(cv.value, `^[0-9_a-z]+$`) {
		cv.line.logWarning("Invalid user or group name %q.", cv.value)
	}
}

func (cv *CheckVartype) Varname() {
	if cv.value != "" && cv.valueNovar == "" {
		// The value of another variable

	} else if !match0(cv.valueNovar, `^[A-Z_][0-9A-Z_]*(?:[.].*)?$`) {
		cv.line.logWarning("%q is not a valid variable name.", cv.value)
	}
}

func (cv *CheckVartype) Version() {
	if !match0(cv.value, `^([\d.])+$`) {
		cv.line.logWarning("Invalid version number %q.", cv.value)
	}
}

func (cv *CheckVartype) WrapperReorder() {
	if !match0(cv.value, `^reorder:l:([\w\-]+):([\w\-]+)$`) {
		cv.line.logWarning("Unknown wrapper reorder command %q.", cv.value)
	}
}

func (cv *CheckVartype) WrapperTransform() {
	switch {
	case match0(cv.value, `^rm:(?:-[DILOUWflm].*|-std=.*)$`):
	case match0(cv.value, `^l:([^:]+):(.+)$`):
	case match0(cv.value, `^'?(?:opt|rename|rm-optarg|rmdir):.*$`):
	case cv.value == "-e":
	case match0(cv.value, `^\"?'?s[|:,]`):
	default:
		cv.line.logWarning("Unknown wrapper transform command %q.", cv.value)
	}
}

func (cv *CheckVartype) WrkdirSubdirectory() {
	checklineMkVartypeSimple(cv.line, cv.varname, "Pathname", cv.op, cv.value, cv.comment, cv.listContext, cv.guessed)
}

func (cv *CheckVartype) WrksrcSubdirectory() {
	if m, _, rest := match2(cv.value, `^(\$\{WRKSRC\})(?:/(.*))?`); m {
		if rest == "" {
			rest = "."
		}
		cv.line.logNote("You can use %q instead of %q.", rest, cv.value)

	} else if cv.value != "" && cv.valueNovar == "" {
		// The value of another variable

	} else if !match0(cv.valueNovar, `^(?:\.|[0-9A-Za-z_\@][-0-9A-Za-z_\@./+]*)$`) {
		cv.line.logWarning("%q is not a valid subdirectory of ${WRKSRC}.", cv.value)
	}
}

func (cv *CheckVartype) Yes() {
	if !match0(cv.value, `^(?:YES|yes)(?:\s+#.*)?$`) {
		cv.line.logWarning("%s should be set to YES or yes.", cv.varname)
		cv.line.explainWarning(
			"This variable means \"yes\" if it is defined, and \"no\" if it is",
			"undefined. Even when it has the value \"no\", this means \"yes\".",
			"Therefore when it is defined, its value should correspond to its",
			"meaning.")
	}
}

func (cv *CheckVartype) YesNo() {
	if !match0(cv.value, `^(?:YES|yes|NO|no)(?:\s+#.*)?$`) {
		cv.line.logWarning("%s should be set to YES, yes, NO, or no.", cv.varname)
	}
}

func (cv *CheckVartype) YesNo_Indirectly() {
	if cv.valueNovar != "" && !match0(cv.value, `^(?:YES|yes|NO|no)(?:\s+#.*)?$`) {
		cv.line.logWarning("%s should be set to YES, yes, NO, or no.", cv.varname)
	}
}
