package main

import (
	"path"
	"strings"
)

type VartypeCheck struct {
	mkline      *MkLine
	line        *Line
	varname     string
	op          string
	value       string
	valueNovar  string
	comment     string
	listContext bool
	guessed     Guessed
}

func (cv *VartypeCheck) AwkCommand() {
	if G.opts.DebugUnchecked {
		cv.line.debug1("Unchecked AWK command: %q", cv.value)
	}
}

func (cv *VartypeCheck) BasicRegularExpression() {
	if G.opts.DebugUnchecked {
		cv.line.debug1("Unchecked basic regular expression: %q", cv.value)
	}
}

func (cv *VartypeCheck) BuildlinkDepmethod() {
	if !containsVarRef(cv.value) && cv.value != "build" && cv.value != "full" {
		cv.line.warn1("Invalid dependency method %q. Valid methods are \"build\" or \"full\".", cv.value)
	}
}

func (cv *VartypeCheck) Category() {
	if fileExists(G.currentDir + "/" + G.curPkgsrcdir + "/" + cv.value + "/Makefile") {
		return
	}
	switch cv.value {
	case
		"chinese", "crosspkgtools",
		"gnome", "gnustep",
		"japanese", "java",
		"kde", "korean",
		"linux", "local",
		"packages", "perl5", "plan9", "python",
		"ruby",
		"scm",
		"tcl", "tk",
		"windowmaker",
		"xmms":
	default:
		cv.line.error1("Invalid category %q.", cv.value)
	}
}

// A single option to the C/C++ compiler.
func (cv *VartypeCheck) CFlag() {
	line, value := cv.line, cv.value

	if hasPrefix(value, "-") {
		if !matches(value, `^-[DILOUWfgm]`) && !hasPrefix(value, "-std=") && value != "-c99" {
			line.warn1("Unknown compiler flag %q.", value)
		}
	} else {
		if !containsVarRef(value) && !(hasPrefix(value, "`") && hasSuffix(value, "`")) {
			line.warn1("Compiler flag %q should start with a hyphen.", value)
		}
	}
}

// The single-line description of the package.
func (cv *VartypeCheck) Comment() {
	line, value := cv.line, cv.value

	if value == "TODO: Short description of the package" { // See pkgtools/url2pkg/files/url2pkg.pl, keyword "COMMENT".
		line.error0("COMMENT must be set.")
	}
	if m, first := match1(value, `^(?i)(a|an)\s`); m {
		line.warn1("COMMENT should not begin with %q.", first)
	}
	if matches(value, `^[a-z]`) {
		line.warn0("COMMENT should start with a capital letter.")
	}
	if hasSuffix(value, ".") {
		line.warn0("COMMENT should not end with a period.")
	}
	if len(value) > 70 {
		line.warn0("COMMENT should not be longer than 70 characters.")
	}
}

func (cv *VartypeCheck) Dependency() {
	line, value := cv.line, cv.value

	if matches(value, `^(`+rePkgbase+`)(<|=|>|<=|>=|!=|-)(`+rePkgversion+`)$`) {
		return
	}
	if matches(value, `^(`+rePkgbase+`)>=?(`+rePkgversion+`)<=?(`+rePkgversion+`)$`) {
		return
	}

	if m, depbase, bracket, version, versionWildcard, other := match5(value, `^(`+rePkgbase+`)-(?:\[(.*)\]\*|(\d+(?:\.\d+)*(?:\.\*)?)(\{,nb\*\}|\*|)|(.*))?$`); m {
		switch {
		case bracket != "":
			if bracket != "0-9" {
				line.warn0("Only [0-9]* is allowed in the numeric part of a dependency.")
			}

		case version != "" && versionWildcard != "":
			// Fine.

		case version != "":
			line.warn0("Please append \"{,nb*}\" to the version number of this dependency.")
			explain4(
				"Usually, a dependency should stay valid when the PKGREVISION is",
				"increased, since those changes are most often editorial. In the",
				"current form, the dependency only matches if the PKGREVISION is",
				"undefined.")

		case other == "*":
			line.warn2("Please use \"%s-[0-9]*\" instead of \"%s-*\".", depbase, depbase)
			explain3(
				"If you use a * alone, the package specification may match other",
				"packages that have the same prefix, but a longer name. For example,",
				"foo-* matches foo-1.2, but also foo-client-1.2 and foo-server-1.2.")

		default:
			line.error1("Unknown dependency pattern %q.", value)
		}
		return
	}

	switch {
	case strings.Contains(value, "{"):
		// No check yet for alternative dependency patterns.
		if G.opts.DebugUnchecked {
			line.debug1("Unchecked alternative dependency pattern: %s", value)
		}

	case value != cv.valueNovar:
		if G.opts.DebugUnchecked {
			line.debug1("Unchecked dependency: %s", value)
		}

	default:
		line.warn1("Unknown dependency pattern %q.", value)
		explain(
			"Typical dependencies have the following forms:",
			"",
			"\tpackage>=2.5",
			"\tpackage-[0-9]*",
			"\tpackage-3.141",
			"\tpackage>=2.71827<=3.1415")
	}
}

func (cv *VartypeCheck) DependencyWithPath() {
	line, value := cv.line, cv.value
	if value != cv.valueNovar {
		return // It's probably not worth checking this.
	}

	if m, pattern, relpath, pkg := match3(value, `(.*):(\.\./\.\./[^/]+/([^/]+))$`); m {
		cv.mkline.checkRelativePkgdir(relpath)

		switch pkg {
		case "msgfmt", "gettext":
			line.warn0("Please use USE_TOOLS+=msgfmt instead of this dependency.")
		case "perl5":
			line.warn0("Please use USE_TOOLS+=perl:run instead of this dependency.")
		case "gmake":
			line.warn0("Please use USE_TOOLS+=gmake instead of this dependency.")
		}

		cv.mkline.checkVartypePrimitive(cv.varname, CheckvarDependency, cv.op, pattern, cv.comment, cv.listContext, cv.guessed)
		return
	}

	if matches(value, `:\.\./[^/]+$`) {
		line.warn0("Dependencies should have the form \"../../category/package\".")
		cv.mkline.explainRelativeDirs()
		return
	}

	line.warn1("Unknown dependency pattern with path %q.", value)
	explain4(
		"Examples for valid dependency patterns with path are:",
		"  package-[0-9]*:../../category/package",
		"  package>=3.41:../../category/package",
		"  package-2.718:../../category/package")
}

func (cv *VartypeCheck) DistSuffix() {
	if cv.value == ".tar.gz" {
		cv.line.note1("%s is \".tar.gz\" by default, so this definition may be redundant.", cv.varname)
	}
}

func (cv *VartypeCheck) EmulPlatform() {

	if m, opsys, arch := match2(cv.value, `^(\w+)-(\w+)$`); m {
		if !matches(opsys, `^(?:bsdos|cygwin|darwin|dragonfly|freebsd|haiku|hpux|interix|irix|linux|netbsd|openbsd|osf1|sunos|solaris)$`) {
			cv.line.warn1("Unknown operating system: %s", opsys)
		}
		// no check for os_version
		if !matches(arch, `^(?:i386|alpha|amd64|arc|arm|arm32|cobalt|convex|dreamcast|hpcmips|hpcsh|hppa|ia64|m68k|m88k|mips|mips64|mipsel|mipseb|mipsn32|ns32k|pc532|pmax|powerpc|rs6000|s390|sparc|sparc64|vax|x86_64)$`) {
			cv.line.warn1("Unknown hardware architecture: %s", arch)
		}

	} else {
		cv.line.warn1("%q is not a valid emulation platform.", cv.value)
		explain(
			"An emulation platform has the form <OPSYS>-<MACHINE_ARCH>.",
			"OPSYS is the lower-case name of the operating system, and MACHINE_ARCH",
			"is the hardware architecture.",
			"",
			"Examples: linux-i386, irix-mipsel.")
	}
}

func (cv *VartypeCheck) FetchURL() {
	cv.mkline.checkVartypePrimitive(cv.varname, CheckvarURL, cv.op, cv.value, cv.comment, cv.listContext, cv.guessed)

	for siteUrl, siteName := range G.globalData.masterSiteUrls {
		if hasPrefix(cv.value, siteUrl) {
			subdir := cv.value[len(siteUrl):]
			isGithub := hasPrefix(cv.value, "https://github.com/")
			if isGithub {
				subdir = strings.SplitAfter(subdir, "/")[0]
			}
			cv.line.warnf("Please use ${%s:=%s} instead of %q.", siteName, subdir, cv.value)
			if isGithub {
				cv.line.warn1("Run \"%s help topic=github\" for further tips.", confMake)
			}
			return
		}
	}
}

// See Pathname
// See http://www.opengroup.org/onlinepubs/009695399/basedefs/xbd_chap03.html#tag_03_169
func (cv *VartypeCheck) Filename() {
	switch {
	case strings.Contains(cv.valueNovar, "/"):
		cv.line.warn0("A filename should not contain a slash.")
	case !matches(cv.valueNovar, `^[-0-9@A-Za-z.,_~+%]*$`):
		cv.line.warn1("%q is not a valid filename.", cv.value)
	}
}

func (cv *VartypeCheck) Filemask() {
	if !matches(cv.valueNovar, `^[-0-9A-Za-z._~+%*?]*$`) {
		cv.line.warn1("%q is not a valid filename mask.", cv.value)
	}
}

func (cv *VartypeCheck) FileMode() {
	switch {
	case cv.value != "" && cv.valueNovar == "":
		// Fine.
	case matches(cv.value, `^[0-7]{3,4}`):
		// Fine.
	default:
		cv.line.warn1("Invalid file mode %q.", cv.value)
	}
}

func (cv *VartypeCheck) Identifier() {
	if cv.value != cv.valueNovar {
		//line.logWarning("Identifiers should be given directly.")
	}
	switch {
	case matches(cv.valueNovar, `^[+\-.0-9A-Z_a-z]+$`):
		// Fine.
	case cv.value != "" && cv.valueNovar == "":
		// Don't warn here.
	default:
		cv.line.warn1("Invalid identifier %q.", cv.value)
	}
}

func (cv *VartypeCheck) Integer() {
	if !matches(cv.value, `^\d+$`) {
		cv.line.warn1("Invalid integer %q.", cv.value)
	}
}

func (cv *VartypeCheck) LdFlag() {
	if ldflag := cv.value; hasPrefix(ldflag, "-") {
		if m, rpathFlag := match1(ldflag, `^(-Wl,(?:-R|-rpath|--rpath))`); m {
			cv.line.warn1("Please use \"${COMPILER_RPATH_FLAG}\" instead of %q.", rpathFlag)
		} else if !hasPrefix(ldflag, "-L") && !hasPrefix(ldflag, "-l") && ldflag != "-static" {
			cv.line.warn1("Unknown linker flag %q.", cv.value)
		}
	} else {
		if ldflag == cv.valueNovar && !(hasPrefix(ldflag, "`") && hasSuffix(ldflag, "`")) {
			cv.line.warn1("Linker flag %q should start with a hypen.", cv.value)
		}
	}
}

func (cv *VartypeCheck) License() {
	checklineLicense(cv.mkline, cv.value)
}

func (cv *VartypeCheck) MailAddress() {
	line, value := cv.line, cv.value

	if m, _, domain := match2(value, `^([+\-.0-9A-Z_a-z]+)@([-\w\d.]+)$`); m {
		if strings.EqualFold(domain, "NetBSD.org") && domain != "NetBSD.org" {
			line.warn1("Please write \"NetBSD.org\" instead of %q.", domain)
		}
		if matches(value, `(?i)^(tech-pkg|packages)@NetBSD\.org$`) {
			line.error0("This mailing list address is obsolete. Use pkgsrc-users@NetBSD.org instead.")
		}

	} else {
		line.warn1("\"%s\" is not a valid mail address.", value)
	}
}

// See ${STEP_MSG}, ${PKG_FAIL_REASON}
func (cv *VartypeCheck) Message() {
	line, varname, value := cv.line, cv.varname, cv.value

	if matches(value, `^[\"'].*[\"']$`) {
		line.warn1("%s should not be quoted.", varname)
		explain(
			"The quoting is only needed for variables which are interpreted as",
			"multiple words (or, generally speaking, a list of something). A single",
			"text message does not belong to this class, since it is only printed",
			"as a whole.",
			"",
			"On the other hand, PKG_FAIL_REASON is a _list_ of text messages, so in",
			"that case, the quoting has to be done.`")
	}
}

// A package option from options.mk
func (cv *VartypeCheck) Option() {
	line, value, valueNovar := cv.line, cv.value, cv.valueNovar

	if value != valueNovar {
		if G.opts.DebugUnchecked {
			line.debug1("Unchecked option name: %q", value)
		}
		return
	}

	if m, optname := match1(value, `^-?([a-z][-0-9a-z+]*)$`); m {
		if _, found := G.globalData.pkgOptions[optname]; !found { // There’s a difference between empty and absent here.
			line.warn1("Unknown option \"%s\".", optname)
			explain4(
				"This option is not documented in the mk/defaults/options.description",
				"file. If this is not a typo, please think of a brief but precise",
				"description and either update that file yourself or ask on the",
				"tech-pkg@NetBSD.org mailing list.")
		}
		return
	}

	if matches(value, `^-?([a-z][-0-9a-z_\+]*)$`) {
		line.warn0("Use of the underscore character in option names is deprecated.")
		return
	}

	line.error1("Invalid option name %q. Option names must start with a lowercase letter and be all-lowercase.", value)
}

// The PATH environment variable
func (cv *VartypeCheck) Pathlist() {
	if !strings.Contains(cv.value, ":") && cv.guessed == guGuessed {
		cv.mkline.checkVartypePrimitive(cv.varname, CheckvarPathname, cv.op, cv.value, cv.comment, cv.listContext, cv.guessed)
		return
	}

	for _, path := range strings.Split(cv.value, ":") {
		if strings.Contains(path, "${") {
			continue
		}

		if !matches(path, `^[-0-9A-Za-z._~+%/]*$`) {
			cv.line.warn1("%q is not a valid pathname.", path)
		}

		if !hasPrefix(path, "/") {
			cv.line.warn2("All components of %s (in this case %q) should be absolute paths.", cv.varname, path)
		}
	}
}

// Shell globbing including slashes.
// See Filemask
func (cv *VartypeCheck) Pathmask() {
	if !matches(cv.valueNovar, `^[#\-0-9A-Za-z._~+%*?/\[\]]*`) {
		cv.line.warn1("%q is not a valid pathname mask.", cv.value)
	}
	cv.line.checkAbsolutePathname(cv.value)
}

// Like Filename, but including slashes
// See http://www.opengroup.org/onlinepubs/009695399/basedefs/xbd_chap03.html#tag_03_266
func (cv *VartypeCheck) Pathname() {
	if !matches(cv.valueNovar, `^[#\-0-9A-Za-z._~+%/]*$`) {
		cv.line.warn1("%q is not a valid pathname.", cv.value)
	}
	cv.line.checkAbsolutePathname(cv.value)
}

func (cv *VartypeCheck) Perl5Packlist() {
	if cv.value != cv.valueNovar {
		cv.line.warn1("%s should not depend on other variables.", cv.varname)
	}
}

func (cv *VartypeCheck) PkgName() {
	if cv.value == cv.valueNovar && !matches(cv.value, rePkgname) {
		cv.line.warn1("%q is not a valid package name. A valid package name has the form packagename-version, where version consists only of digits, letters and dots.", cv.value)
	}
}

func (cv *VartypeCheck) PkgOptionsVar() {
	cv.mkline.checkVartypePrimitive(cv.varname, CheckvarVarname, cv.op, cv.value, cv.comment, false, cv.guessed)
	if matches(cv.value, `\$\{PKGBASE[:\}]`) {
		cv.line.error0("PKGBASE must not be used in PKG_OPTIONS_VAR.")
		explain3(
			"PKGBASE is defined in bsd.pkg.mk, which is included as the",
			"very last file, but PKG_OPTIONS_VAR is evaluated earlier.",
			"Use ${PKGNAME:C/-[0-9].*//} instead.")
	}
}

// A directory name relative to the top-level pkgsrc directory.
// Despite its name, it is more similar to RelativePkgDir than to RelativePkgPath.
func (cv *VartypeCheck) PkgPath() {
	cv.mkline.checkRelativePkgdir(G.curPkgsrcdir + "/" + cv.value)
}

func (cv *VartypeCheck) PkgRevision() {
	if !matches(cv.value, `^[1-9]\d*$`) {
		cv.line.warn1("%s must be a positive integer number.", cv.varname)
	}
	if path.Base(cv.line.fname) != "Makefile" {
		cv.line.error1("%s only makes sense directly in the package Makefile.", cv.varname)
		explain(
			"Usually, different packages using the same Makefile.common have",
			"different dependencies and will be bumped at different times (e.g. for",
			"shlib major bumps) and thus the PKGREVISIONs must be in the separate",
			"Makefiles. There is no practical way of having this information in a",
			"commonly used Makefile.")
	}
}

func (cv *VartypeCheck) PlatformTriple() {
	if cv.value != cv.valueNovar {
		return
	}

	rePart := `(?:\[[^\]]+\]|[^-\[])+`
	reTriple := `^(` + rePart + `)-(` + rePart + `)-(` + rePart + `)$`
	if m, opsys, _, arch := match3(cv.value, reTriple); m {
		if !matches(opsys, `^(?:\*|BSDOS|Cygwin|Darwin|DragonFly|FreeBSD|Haiku|HPUX|Interix|IRIX|Linux|NetBSD|OpenBSD|OSF1|QNX|SunOS)$`) {
			cv.line.warn1("Unknown operating system: %s", opsys)
		}
		// no check for os_version
		if !matches(arch, `^(?:\*|i386|alpha|amd64|arc|arm|arm32|cobalt|convex|dreamcast|hpcmips|hpcsh|hppa|ia64|m68k|m88k|mips|mips64|mipsel|mipseb|mipsn32|ns32k|pc532|pmax|powerpc|rs6000|s390|sparc|sparc64|vax|x86_64)$`) {
			cv.line.warn1("Unknown hardware architecture: %s", arch)
		}

	} else {
		cv.line.warn1("%q is not a valid platform triple.", cv.value)
		explain3(
			"A platform triple has the form <OPSYS>-<OS_VERSION>-<MACHINE_ARCH>.",
			"Each of these components may be a shell globbing expression.",
			"Examples: NetBSD-*-i386, *-*-*, Linux-*-*.")
	}
}

func (cv *VartypeCheck) PrefixPathname() {
	if m, mansubdir := match1(cv.value, `^man/(.+)`); m {
		cv.line.warn2("Please use \"${PKGMANDIR}/%s\" instead of %q.", mansubdir, cv.value)
	}
}

func (cv *VartypeCheck) PythonDependency() {
	if cv.value != cv.valueNovar {
		cv.line.warn0("Python dependencies should not contain variables.")
	}
	if !matches(cv.valueNovar, `^[+\-.0-9A-Z_a-z]+(?:|:link|:build)$`) {
		cv.line.warn1("Invalid Python dependency %q.", cv.value)
		explain4(
			"Python dependencies must be an identifier for a package, as specified",
			"in lang/python/versioned_dependencies.mk. This identifier may be",
			"followed by :build for a build-time only dependency, or by :link for",
			"a run-time only dependency.")
	}
}

// Refers to a package directory.
func (cv *VartypeCheck) RelativePkgDir() {
	cv.mkline.checkRelativePkgdir(cv.value)
}

// Refers to a file or directory.
func (cv *VartypeCheck) RelativePkgPath() {
	cv.mkline.checkRelativePath(cv.value, true)
}

func (cv *VartypeCheck) Restricted() {
	if cv.value != "${RESTRICTED}" {
		cv.line.warn1("The only valid value for %s is ${RESTRICTED}.", cv.varname)
		explain3(
			"These variables are used to control which files may be mirrored on FTP",
			"servers or CD-ROM collections. They are not intended to mark packages",
			"whose only MASTER_SITES are on ftp.NetBSD.org.")
	}
}

func (cv *VartypeCheck) SedCommand() {
}

func (cv *VartypeCheck) SedCommands() {
	line := cv.line
	mkline := cv.mkline
	shline := NewMkShellLine(mkline)

	words, rest := splitIntoShellTokens(line, cv.value)
	if rest != "" {
		if strings.Contains(cv.value, "#") {
			line.error0("Invalid shell words in sed commands.")
			explain4(
				"When sed commands have embedded \"#\" characters, they need to be",
				"escaped with a backslash, otherwise make(1) will interpret them as a",
				"comment, no matter if they occur in single or double quotes or",
				"whatever.")
		}
		return
	}

	nwords := len(words)
	ncommands := 0

	for i := 0; i < nwords; i++ {
		word := words[i]
		shline.checkShellword(word, true)

		switch {
		case word == "-e":
			if i+1 < nwords {
				// Check the real sed command here.
				i++
				ncommands++
				if ncommands > 1 {
					line.note0("Each sed command should appear in an assignment of its own.")
					explain(
						"For example, instead of",
						"    SUBST_SED.foo+=        -e s,command1,, -e s,command2,,",
						"use",
						"    SUBST_SED.foo+=        -e s,command1,,",
						"    SUBST_SED.foo+=        -e s,command2,,",
						"",
						"This way, short sed commands cannot be hidden at the end of a line.")
				}
				shline.checkShellword(words[i-1], true)
				shline.checkShellword(words[i], true)
				mkline.checkVartypePrimitive(cv.varname, CheckvarSedCommand, cv.op, words[i], cv.comment, cv.listContext, cv.guessed)
			} else {
				line.error0("The -e option to sed requires an argument.")
			}
		case word == "-E":
			// Switch to extended regular expressions mode.

		case word == "-n":
			// Don't print lines per default.

		case i == 0 && matches(word, `^(["']?)(?:\d*|/.*/)s.+["']?$`):
			line.note0("Please always use \"-e\" in sed commands, even if there is only one substitution.")

		default:
			line.warn1("Unknown sed command %q.", word)
		}
	}
}

func (cv *VartypeCheck) ShellCommand() {
	setE := true
	NewMkShellLine(cv.mkline).checkShellCommand(cv.value, &setE)
}

// Zero or more shell commands, each terminated with a semicolon.
func (cv *VartypeCheck) ShellCommands() {
	NewMkShellLine(cv.mkline).checkShellCommands(cv.value)
}

func (cv *VartypeCheck) ShellWord() {
	if !cv.listContext {
		NewMkShellLine(cv.mkline).checkShellword(cv.value, true)
	}
}

func (cv *VartypeCheck) Stage() {
	if !matches(cv.value, `^(?:pre|do|post)-(?:extract|patch|configure|build|test|install)`) {
		cv.line.warn1("Invalid stage name %q. Use one of {pre,do,post}-{extract,patch,configure,build,test,install}.", cv.value)
	}
}

func (cv *VartypeCheck) String() {
	// No further checks possible.
}

func (cv *VartypeCheck) Tool() {
	if cv.varname == "TOOLS_NOOP" && cv.op == "+=" {
		// no warning for package-defined tool definitions

	} else if m, toolname, tooldep := match2(cv.value, `^([-\w]+|\[)(?::(\w+))?$`); m {
		if !G.globalData.tools[toolname] {
			cv.line.error1("Unknown tool %q.", toolname)
		}
		switch tooldep {
		case "", "bootstrap", "build", "pkgsrc", "run":
		default:
			cv.line.error1("Unknown tool dependency %q. Use one of \"build\", \"pkgsrc\" or \"run\".", tooldep)
		}
	} else {
		cv.line.error1("Invalid tool syntax: %q.", cv.value)
	}
}

func (cv *VartypeCheck) Unchecked() {
	// Do nothing, as the name says.
}

func (cv *VartypeCheck) URL() {
	line, value := cv.line, cv.value

	if value == "" && hasPrefix(cv.comment, "#") {
		// Ok

	} else if m, name, subdir := match2(value, `\$\{(MASTER_SITE_[^:]*).*:=(.*)\}$`); m {
		if !G.globalData.masterSiteVars[name] {
			line.error1("%s does not exist.", name)
		}
		if !hasSuffix(subdir, "/") {
			line.error1("The subdirectory in %s must end with a slash.", name)
		}

	} else if containsVarRef(value) {
		// No further checks

	} else if m, _, host, _, _ := match4(value, `^(https?|ftp|gopher)://([-0-9A-Za-z.]+)(?::(\d+))?/([-%&+,./0-9:;=?@A-Z_a-z~]|#)*$`); m {
		if matches(host, `(?i)\.NetBSD\.org$`) && !matches(host, `\.NetBSD\.org$`) {
			line.warn1("Please write NetBSD.org instead of %s.", host)
		}

	} else if m, scheme, _, absPath := match3(value, `^([0-9A-Za-z]+)://([^/]+)(.*)$`); m {
		switch {
		case scheme != "ftp" && scheme != "http" && scheme != "https" && scheme != "gopher":
			line.warn1("%q is not a valid URL. Only ftp, gopher, http, and https URLs are allowed here.", value)

		case absPath == "":
			line.note1("For consistency, please add a trailing slash to %q.", value)

		default:
			line.warn1("%q is not a valid URL.", value)
		}

	} else {
		line.warn1("%q is not a valid URL.", value)
	}
}

func (cv *VartypeCheck) UserGroupName() {
	if cv.value == cv.valueNovar && !matches(cv.value, `^[0-9_a-z]+$`) {
		cv.line.warn1("Invalid user or group name %q.", cv.value)
	}
}

func (cv *VartypeCheck) Varname() {
	if cv.value == cv.valueNovar && !matches(cv.value, `^[A-Z_][0-9A-Z_]*(?:[.].*)?$`) {
		cv.line.warn1("%q is not a valid variable name.", cv.value)
		explain(
			"Variable names are restricted to only uppercase letters and the",
			"underscore in the basename, and arbitrary characters in the",
			"parameterized part, following the dot.",
			"",
			"Examples:",
			"\t* PKGNAME",
			"\t* PKG_OPTIONS.gnuchess")
	}
}

func (cv *VartypeCheck) Version() {
	if !matches(cv.value, `^([\d.])+$`) {
		cv.line.warn1("Invalid version number %q.", cv.value)
	}
}

func (cv *VartypeCheck) WrapperReorder() {
	if !matches(cv.value, `^reorder:l:([\w\-]+):([\w\-]+)$`) {
		cv.line.warn1("Unknown wrapper reorder command %q.", cv.value)
	}
}

func (cv *VartypeCheck) WrapperTransform() {
	switch {
	case matches(cv.value, `^rm:(?:-[DILOUWflm].*|-std=.*)$`):
	case matches(cv.value, `^l:([^:]+):(.+)$`):
	case matches(cv.value, `^'?(?:opt|rename|rm-optarg|rmdir):.*$`):
	case cv.value == "-e":
	case matches(cv.value, `^\"?'?s[|:,]`):
	default:
		cv.line.warn1("Unknown wrapper transform command %q.", cv.value)
	}
}

func (cv *VartypeCheck) WrkdirSubdirectory() {
	cv.mkline.checkVartypePrimitive(cv.varname, CheckvarPathname, cv.op, cv.value, cv.comment, cv.listContext, cv.guessed)
}

// A directory relative to ${WRKSRC}, for use in CONFIGURE_DIRS and similar variables.
func (cv *VartypeCheck) WrksrcSubdirectory() {
	if m, _, rest := match2(cv.value, `^(\$\{WRKSRC\})(?:/(.*))?`); m {
		if rest == "" {
			rest = "."
		}
		cv.line.note2("You can use %q instead of %q.", rest, cv.value)
		explain1(
			"These directories are interpreted relative to ${WRKSRC}.")

	} else if cv.value != "" && cv.valueNovar == "" {
		// The value of another variable

	} else if !matches(cv.valueNovar, `^(?:\.|[0-9A-Za-z_@][-0-9A-Za-z_@./+]*)$`) {
		cv.line.warn1("%q is not a valid subdirectory of ${WRKSRC}.", cv.value)
	}
}

// Used for variables that are checked using `.if defined(VAR)`.
func (cv *VartypeCheck) Yes() {
	if !matches(cv.value, `^(?:YES|yes)(?:\s+#.*)?$`) {
		cv.line.warn1("%s should be set to YES or yes.", cv.varname)
		explain4(
			"This variable means \"yes\" if it is defined, and \"no\" if it is",
			"undefined. Even when it has the value \"no\", this means \"yes\".",
			"Therefore when it is defined, its value should correspond to its",
			"meaning.")
	}
}

// The type YesNo is used for variables that are checked using
//     .if defined(VAR) && !empty(VAR:M[Yy][Ee][Ss])
//
func (cv *VartypeCheck) YesNo() {
	if !matches(cv.value, `^(?:YES|yes|NO|no)(?:\s+#.*)?$`) {
		cv.line.warn1("%s should be set to YES, yes, NO, or no.", cv.varname)
	}
}

// Like YesNo, but the value may be produced by a shell command using the
// != operator.
func (cv *VartypeCheck) YesNoIndirectly() {
	if cv.valueNovar != "" && !matches(cv.value, `^(?:YES|yes|NO|no)(?:\s+#.*)?$`) {
		cv.line.warn1("%s should be set to YES, yes, NO, or no.", cv.varname)
	}
}
