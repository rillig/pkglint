package main

import (
	"fmt"
	"os"
	"path"
	"strings"
)

const (
	reDependencyCmp      = `^((?:\$\{[\w_]+\}|[\w_\.+]|-[^\d])+)[<>]=?(\d[^-*?\[\]]*)$`
	reDependencyWildcard = `^((?:\$\{[\w_]+\}|[\w_\.+]|-[^\d\[])+)-(?:\[0-9\]\*|\d[^-]*)$`
	reMkCond             = `^\.(\s*)(if|ifdef|ifndef|else|elif|endif|for|endfor|undef)(?:\s+([^\s#][^#]*?))?\s*(?:#.*)?$`
	reMkInclude          = `^\.\s*(s?include)\s+\"([^\"]+)\"\s*(?:#.*)?$`
	rePkgname            = `^([\w\-.+]+)-(\d(?:\w|\.\d)*)$`
	rePkgbase            = `(?:[+.\w]|-[A-Z_a-z])+`
	rePkgversion         = `\d(?:\w|\.\d)*`
)

// Returns the pkgsrc top-level directory, relative to the given file or directory.
func findPkgsrcTopdir(fname string) string {
	for _, dir := range []string{".", "..", "../..", "../../.."} {
		if fileExists(fname + "/" + dir + "/mk/bsd.pkg.mk") {
			return dir
		}
	}
	return ""
}

func (pkg *Package) loadPackageMakefile(fname string) *MkLines {
	defer tracecall("loadPackageMakefile", fname)()

	mainLines, allLines := NewMkLines(nil), NewMkLines(nil)
	if !readMakefile(fname, mainLines, allLines, "") {
		return nil
	}

	if G.opts.DumpMakefile {
		debugf(G.currentDir, noLines, "Whole Makefile (with all included files) follows:")
		for _, line := range allLines.lines {
			fmt.Printf("%s\n", line.String())
		}
	}

	allLines.determineUsedVariables()

	pkg.pkgdir = expandVariableWithDefault("PKGDIR", ".")
	pkg.distinfoFile = expandVariableWithDefault("DISTINFO_FILE", "distinfo")
	pkg.filesdir = expandVariableWithDefault("FILESDIR", "files")
	pkg.patchdir = expandVariableWithDefault("PATCHDIR", "patches")

	if varIsDefined("PHPEXT_MK") {
		if !varIsDefined("USE_PHP_EXT_PATCHES") {
			pkg.patchdir = "patches"
		}
		if varIsDefined("PECL_VERSION") {
			pkg.distinfoFile = "distinfo"
		}
	}

	_ = G.opts.DebugMisc &&
		dummyLine.debugf("DISTINFO_FILE=%s", pkg.distinfoFile) &&
		dummyLine.debugf("FILESDIR=%s", pkg.filesdir) &&
		dummyLine.debugf("PATCHDIR=%s", pkg.patchdir) &&
		dummyLine.debugf("PKGDIR=%s", pkg.pkgdir)

	return mainLines
}

func extractUsedVariables(line *Line, text string) []string {
	re := regcomp(`^(?:[^\$]+|\$[\$*<>?@]|\$\{([.0-9A-Z_a-z]+)(?::(?:[^\${}]|\$[^{])+)?\})`)
	rest := text
	var result []string
	for {
		m := re.FindStringSubmatchIndex(rest)
		if m == nil {
			break
		}
		varname := rest[negToZero(m[2]):negToZero(m[3])]
		rest = rest[:m[0]] + rest[m[1]:]
		if varname != "" {
			result = append(result, varname)
		}
	}

	if rest != "" {
		_ = G.opts.DebugMisc && line.debugf("extractUsedVariables: rest=%q", rest)
	}
	return result
}

// Returns the type of the variable (maybe guessed based on the variable name),
// or nil if the type cannot even be guessed.
func getVariableType(line *Line, varname string) *Vartype {

	if vartype := G.globalData.vartypes[varname]; vartype != nil {
		return vartype
	}
	if vartype := G.globalData.vartypes[varnameCanon(varname)]; vartype != nil {
		return vartype
	}

	if G.globalData.varnameToToolname[varname] != "" {
		return &Vartype{lkNone, CheckvarShellCommand, []AclEntry{{"*", "u"}}, guNotGuessed}
	}

	if m, toolvarname := match1(varname, `^TOOLS_(.*)`); m && G.globalData.varnameToToolname[toolvarname] != "" {
		return &Vartype{lkNone, CheckvarPathname, []AclEntry{{"*", "u"}}, guNotGuessed}
	}

	allowAll := []AclEntry{{"*", "adpsu"}}
	allowRuntime := []AclEntry{{"*", "adsu"}}

	// Guess the datatype of the variable based on naming conventions.
	varbase := varnameBase(varname)
	var gtype *Vartype
	switch {
	case hasSuffix(varbase, "DIRS"):
		gtype = &Vartype{lkShell, CheckvarPathmask, allowRuntime, guGuessed}
	case hasSuffix(varbase, "DIR"), hasSuffix(varname, "_HOME"):
		gtype = &Vartype{lkNone, CheckvarPathname, allowRuntime, guGuessed}
	case hasSuffix(varbase, "FILES"):
		gtype = &Vartype{lkShell, CheckvarPathmask, allowRuntime, guGuessed}
	case hasSuffix(varbase, "FILE"):
		gtype = &Vartype{lkNone, CheckvarPathname, allowRuntime, guGuessed}
	case hasSuffix(varbase, "PATH"):
		gtype = &Vartype{lkNone, CheckvarPathlist, allowRuntime, guGuessed}
	case hasSuffix(varbase, "PATHS"):
		gtype = &Vartype{lkShell, CheckvarPathname, allowRuntime, guGuessed}
	case hasSuffix(varbase, "_USER"):
		gtype = &Vartype{lkNone, CheckvarUserGroupName, allowAll, guGuessed}
	case hasSuffix(varbase, "_GROUP"):
		gtype = &Vartype{lkNone, CheckvarUserGroupName, allowAll, guGuessed}
	case hasSuffix(varbase, "_ENV"):
		gtype = &Vartype{lkShell, CheckvarShellWord, allowRuntime, guGuessed}
	case hasSuffix(varbase, "_CMD"):
		gtype = &Vartype{lkNone, CheckvarShellCommand, allowRuntime, guGuessed}
	case hasSuffix(varbase, "_ARGS"):
		gtype = &Vartype{lkShell, CheckvarShellWord, allowRuntime, guGuessed}
	case hasSuffix(varbase, "_CFLAGS"), hasSuffix(varname, "_CPPFLAGS"), hasSuffix(varname, "_CXXFLAGS"), hasSuffix(varname, "_LDFLAGS"):
		gtype = &Vartype{lkShell, CheckvarShellWord, allowRuntime, guGuessed}
	case hasSuffix(varbase, "_MK"):
		gtype = &Vartype{lkNone, CheckvarUnchecked, allowAll, guGuessed}
	case hasPrefix(varbase, "PLIST."):
		gtype = &Vartype{lkNone, CheckvarYes, allowAll, guGuessed}
	}

	if gtype != nil {
		_ = G.opts.DebugVartypes && line.debugf("The guessed type of %q is %v.", varname, gtype)
	} else {
		_ = G.opts.DebugVartypes && line.debugf("No type definition found for %q.", varname)
	}
	return gtype
}

func resolveVariableRefs(text string) string {
	defer tracecall("resolveVariableRefs", text)()

	visited := make(map[string]bool) // To prevent endless loops

	str := text
	for {
		replaced := regcomp(`\$\{([\w.]+)\}`).ReplaceAllStringFunc(str, func(m string) string {
			varname := m[2 : len(m)-1]
			if !visited[varname] {
				visited[varname] = true
				if G.pkg != nil {
					if value, ok := G.pkg.varValue(varname); ok {
						return value
					}
				}
				if G.mk != nil {
					if value, ok := G.mk.varValue(varname); ok {
						return value
					}
				}
			}
			return sprintf("${%s}", varname)
		})
		if replaced == str {
			return replaced
		}
		str = replaced
	}
}

func expandVariableWithDefault(varname, defaultValue string) string {
	mkline := G.pkg.vardef[varname]
	if mkline == nil {
		return defaultValue
	}

	value := mkline.Value()
	value = resolveVarsInRelativePath(value, true)
	if containsVarRef(value) {
		value = resolveVariableRefs(value)
	}
	_ = G.opts.DebugMisc && mkline.debugf("Expanded %q to %q", varname, value)
	return value
}

func getVariablePermissions(line *Line, varname string) string {
	if vartype := getVariableType(line, varname); vartype != nil {
		return vartype.effectivePermissions(line.fname)
	}

	_ = G.opts.DebugMisc && line.debugf("No type definition found for %q.", varname)
	return "adpsu"
}

func checklineLength(line *Line, maxlength int) {
	if len(line.text) > maxlength {
		line.warnf("Line too long (should be no more than %d characters).", maxlength)
		explain3(
			"Back in the old time, terminals with 80x25 characters were common.",
			"And this is still the default size of many terminal emulators.",
			"Moderately short lines also make reading easier.")
	}
}

func checklineValidCharacters(line *Line, reChar string) {
	rest := regcomp(reChar).ReplaceAllString(line.text, "")
	if rest != "" {
		uni := ""
		for _, c := range rest {
			uni += sprintf(" %U", c)
		}
		line.warn1("Line contains invalid characters (%s).", uni[1:])
	}
}

func checklineTrailingWhitespace(line *Line) {
	if hasSuffix(line.text, " ") || hasSuffix(line.text, "\t") {
		if !line.autofixReplaceRegexp(`\s+\n$`, "\n") {
			line.notef("Trailing white-space.")
			explain2(
				"When a line ends with some white-space, that space is in most cases",
				"irrelevant and can be removed.")
		}
	}
}

func checklineRcsid(line *Line, prefixRe, suggestedPrefix string) bool {
	defer tracecall("checklineRcsid", prefixRe, suggestedPrefix)()

	if !matches(line.text, `^`+prefixRe+`\$NetBSD(?::[^\$]+)?\$$`) {
		if !line.autofixInsertBefore(suggestedPrefix + "$" + "NetBSD$") {
			line.errorf("Expected %q.", suggestedPrefix+"$"+"NetBSD$")
			explain3(
				"Several files in pkgsrc must contain the CVS Id, so that their current",
				"version can be traced back later from a binary package. This is to",
				"ensure reproducible builds, for example for finding bugs.")
		}
		return false
	}
	return true
}

func checkfileExtra(fname string) {
	defer tracecall("checkfileExtra", fname)()

	if lines := LoadNonemptyLines(fname, false); lines != nil {
		checklinesTrailingEmptyLines(lines)
	}
}

func checklinesMessage(lines []*Line) {
	defer tracecall("checklinesMessage", lines[0].fname)()

	explainMessage := func() {
		explain(
			"A MESSAGE file should consist of a header line, having 75 \"=\"",
			"characters, followed by a line containing only the RCS Id, then an",
			"empty line, your text and finally the footer line, which is the",
			"same as the header line.")
	}

	if len(lines) < 3 {
		lastLine := lines[len(lines)-1]
		lastLine.warn0("File too short.")
		explainMessage()
		return
	}

	hline := strings.Repeat("=", 75)
	if line := lines[0]; line.text != hline {
		line.warn0("Expected a line of exactly 75 \"=\" characters.")
		explainMessage()
	}
	checklineRcsid(lines[1], ``, "")
	for _, line := range lines {
		checklineLength(line, 80)
		checklineTrailingWhitespace(line)
		checklineValidCharacters(line, `[\t -~]`)
	}
	if lastLine := lines[len(lines)-1]; lastLine.text != hline {
		lastLine.warn0("Expected a line of exactly 75 \"=\" characters.")
		explainMessage()
	}
	checklinesTrailingEmptyLines(lines)
}

func checkfileMk(fname string) {
	defer tracecall("checkfileMk", fname)()

	lines := LoadNonemptyLines(fname, true)
	if lines == nil {
		return
	}

	NewMkLines(lines).check()
	saveAutofixChanges(lines)
}

func checkfile(fname string) {
	defer tracecall("checkfile", fname)()

	basename := path.Base(fname)
	if hasPrefix(basename, "work") || hasSuffix(basename, "~") || hasSuffix(basename, ".orig") || hasSuffix(basename, ".rej") {
		if G.opts.Import {
			errorf(fname, noLines, "Must be cleaned up before committing the package.")
		}
		return
	}

	st, err := os.Lstat(fname)
	if err != nil {
		errorf(fname, noLines, "%s", err)
		return
	}

	if st.Mode().IsRegular() && st.Mode().Perm()&0111 != 0 && !isCommitted(fname) {
		line := NewLine(fname, 0, "", nil)
		line.warn0("Should not be executable.")
		explain(
			"No package file should ever be executable. Even the INSTALL and",
			"DEINSTALL scripts are usually not usable in the form they have in the",
			"package, as the pathnames get adjusted during installation. So there is",
			"no need to have any file executable.")
	}

	switch {
	case st.Mode().IsDir():
		switch {
		case basename == "files" || basename == "patches" || basename == "CVS":
			// Ok
		case matches(fname, `(?:^|/)files/[^/]*$`):
			// Ok
		case !isEmptyDir(fname):
			warnf(fname, noLines, "Unknown directory name.")
		}

	case st.Mode()&os.ModeSymlink != 0:
		if !matches(basename, `^work`) {
			warnf(fname, noLines, "Unknown symlink name.")
		}

	case !st.Mode().IsRegular():
		errorf(fname, noLines, "Only files and directories are allowed in pkgsrc.")

	case basename == "ALTERNATIVES":
		if G.opts.CheckAlternatives {
			checkfileExtra(fname)
		}

	case basename == "buildlink3.mk":
		if G.opts.CheckBuildlink3 {
			if lines := LoadNonemptyLines(fname, true); lines != nil {
				checklinesBuildlink3Mk(NewMkLines(lines))
			}
		}

	case hasPrefix(basename, "DESCR"):
		if G.opts.CheckDescr {
			if lines := LoadNonemptyLines(fname, false); lines != nil {
				checklinesDescr(lines)
			}
		}

	case basename == "distinfo":
		if G.opts.CheckDistinfo {
			if lines := LoadNonemptyLines(fname, false); lines != nil {
				checklinesDistinfo(lines)
			}
		}

	case basename == "DEINSTALL" || basename == "INSTALL":
		if G.opts.CheckInstall {
			checkfileExtra(fname)
		}

	case hasPrefix(basename, "MESSAGE"):
		if G.opts.CheckMessage {
			if lines := LoadNonemptyLines(fname, false); lines != nil {
				checklinesMessage(lines)
			}
		}

	case matches(basename, `^patch-[-A-Za-z0-9_.~+]*[A-Za-z0-9_]$`):
		if G.opts.CheckPatches {
			if lines := LoadNonemptyLines(fname, false); lines != nil {
				checklinesPatch(lines)
			}
		}

	case matches(fname, `(?:^|/)patches/manual[^/]*$`):
		_ = G.opts.DebugUnchecked && debugf(fname, noLines, "Unchecked file %q.", fname)

	case matches(fname, `(?:^|/)patches/[^/]*$`):
		warnf(fname, noLines, "Patch files should be named \"patch-\", followed by letters, '-', '_', '.', and digits only.")

	case matches(basename, `^(?:.*\.mk|Makefile.*)$`) && !matches(fname, `files/`) && !matches(fname, `patches/`):
		if G.opts.CheckMk {
			checkfileMk(fname)
		}

	case hasPrefix(basename, "PLIST"):
		if G.opts.CheckPlist {
			if lines := LoadNonemptyLines(fname, false); lines != nil {
				checklinesPlist(lines)
			}
		}

	case basename == "TODO" || basename == "README":
		// Ok

	case hasPrefix(basename, "CHANGES-"):
		// This only checks the file, but doesn’t register the changes globally.
		G.globalData.loadDocChangesFromFile(fname)

	case matches(fname, `(?:^|/)files/[^/]*$`):
		// Skip

	default:
		warnf(fname, noLines, "Unexpected file found.")
		if G.opts.CheckExtra {
			checkfileExtra(fname)
		}
	}
}

func checklinesTrailingEmptyLines(lines []*Line) {
	max := len(lines)
	last := max
	for last > 1 && lines[last-1].text == "" {
		last--
	}
	if last != max {
		lines[last].notef("Trailing empty lines.")
	}
}

func matchVarassign(text string) (m bool, varname, op, value, comment string) {
	if !contains(text, "=") {
		return
	}

	var space string
	trimmed := strings.TrimSpace(text)
	if m, varname, space, op = match3(trimmed, `^([-*+A-Z_a-z0-9.${}\[]+)(\s*)([!:?]?=)`); m {
		rest := trimmed[len(varname)+len(space)+len(op):]
		valuebuf := make([]rune, 0, len(rest))
		for i, r := range rest {
			if r == '#' && (i == 0 || rest[i-1] != '\\') {
				comment = rest[i:]
				break
			} else if r != '\\' || i+1 >= len(rest) || rest[i+1] != '#' {
				valuebuf = append(valuebuf, r)
			}
		}

		value = strings.TrimSpace(string(valuebuf))
		if hasSuffix(varname, "+") && space == "" && op == "=" {
			varname = varname[:len(varname)-1]
			op = "+="
		}
		value = strings.TrimSpace(value)
	}
	return
}
