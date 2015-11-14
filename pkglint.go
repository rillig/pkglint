package main

// based on NetBSD: pkglint.pl,v 1.893 2015/10/15 03:00:56 rillig Exp $

import (
	"fmt"
	"os"
	"path"
	"path/filepath"
	"strconv"
	"strings"
)

type QuotingResult string

const (
	QR_FALSE         QuotingResult = "false"
	QR_TRUE          QuotingResult = "true"
	QR_DONT_KNOW     QuotingResult = "don’t know"
	QR_DOESNT_MATTER QuotingResult = "doesn’t matter"
)

// A SimpleMatch is the result of applying a regular expression to a Perl
// scalar value. It can return the range and the text of the captured
// groups.
//
type SimpleMatch struct {
	str    string
	starts []int
	ends   []int
	n      int
}

func NewSimpleMatch(str string, starts, ends []int, n int) *SimpleMatch {
	return &SimpleMatch{str, starts, ends, n}
}
func (self *SimpleMatch) has(i int) bool {
	return 0 <= i && i <= self.n && self.starts[i] != -1 && self.ends[i] != -1
}
func (self *SimpleMatch) text(i int) string {
	start, end := self.starts[i], self.ends[i]
	return self.str[start : end-start]
}

// A Vartype in pkglint is a combination of a data type and a permission
// specification. Further details can be found in the chapter ``The pkglint
// type system'' of the pkglint book.

type KindOfList struct{ name string }

var LK_NONE = KindOfList{"none"}
var LK_INTERNAL = KindOfList{"internal"}
var LK_EXTERNAL = KindOfList{"external"}

type AclEntry struct {
	glob        string
	permissions string
}

// The various contexts in which make(1) variables can appear in pkgsrc.
// Further details can be found in the chapter “The pkglint type system”
// of the pkglint book.
type VarUseContextTime int

const (
	VUC_TIME_UNKNOWN VarUseContextTime = iota
	VUC_TIME_LOAD
	VUC_TIME_RUN
)

type VarUseContextShellword int

const (
	VUC_SHW_UNKNOWN VarUseContextShellword = iota
	VUC_SHW_PLAIN
	VUC_SHW_DQUOT
	VUC_SHW_SQUOT
	VUC_SHW_BACKT
	VUC_SHW_FOR
)

type VarUseContextExtent int

const (
	VUC_EXTENT_UNKNOWN VarUseContextExtent = iota
	VUC_EXT_FULL
	VUC_EXT_WORD
	VUC_EXT_WORDPART
)

type CvsEntry struct {
	fname    string
	revision string
	mtime    string
	tag      string
}

const (
	reDependencyCmp            = `^((?:\$\{[\w_]+\}|[\w_\.+]|-[^\d])+)[<>]=?(\d[^-*?\[\]]*)$`
	reDependencyWildcard       = `^((?:\$\{[\w_]+\}|[\w_\.+]|-[^\d\[])+)-(?:\[0-9\]\*|\d[^-]*)$`
	reGnuConfigureVolatileVars = `^(?:.*_)?(?:CFLAGS||CPPFLAGS|CXXFLAGS|FFLAGS|LDFLAGS|LIBS)$`
	reMkComment                = `^ *\s*#(.*)$`
	reMkCond                   = `^\.(\s*)(if|ifdef|ifndef|else|elif|endif|for|endfor|undef)(?:\s+([^\s#][^#]*?))?\s*(?:#.*)?$`
	reMkDependency             = `^([^\s:]+(?:\s*[^\s:]+)*)(\s*):\s*([^#]*?)(?:\s*#.*)?$`
	reMkInclude                = `^\.\s*(s?include)\s+\"([^\"]+)\"\s*(?:#.*)?$`
	reMkSysinclude             = `^\.\s*s?include\s+<([^>]+)>\s*(?:#.*)?$`
	reMkShellvaruse            = `(?:^|[^\$])\$\$\{?(\w+)\}?`
	rePkgname                  = `^([\w\-.+]+)-(\d(?:\w|\.\d)*)$`
	reMkShellcmd               = `^\t(.*)$`
	reConflict                 = `^(<<<<<<<|=======|>>>>>>>)`
	reUnresolvedVar            = `\$\{`
	reValidchars               = `[\011\040-\176]`
	reVarassign                = `^ *([-*+A-Z_a-z0-9.${}\[]+?)\s*([!+:?]?=)\s*(.*?)(?:\s*(#.*))?$`
	reShVarassign              = `^([A-Z_a-z][0-9A-Z_a-z]*)=`
	// This regular expression cannot parse all kinds of shell programs, but
	// it will catch almost all shell programs that are portable enough to be
	// used in pkgsrc.
	reShellword = `\s*(` +
		`#.*` + // shell comment
		`|(?:` +
		`'[^']*'` + // single quoted string
		`|"(?:\\.|[^"\\])*"` + // double quoted string
		"|`[^`]*`" + // backticks command execution
		`|\\\$\$` + // a shell-escaped dollar sign
		`|\\[^\$]` + // other escaped characters
		`|\$[\w_]` + // one-character make(1) variable
		`|\$\{[^{}]+\}` + // make(1) variable, ${...}
		`|\$\([^()]+\)` + // make(1) variable, $(...)
		`|\$[/\@<^]` + // special make(1) variables
		`|\$\$[0-9A-Z_a-z]+` + // shell variable
		`|\$\$[#?@]` + // special shell variables
		`|\$\$\$\$` + // the special pid shell variable
		`|\$\$\{[0-9A-Z_a-z]+\}` + // shell variable in braces
		`|\$\$\(` + // POSIX-style backticks replacement
		`|[^\(\)'\"\\\s;&\|<>` + "`" + `\$]` + // non-special character
		`|\$\{[^\s\"'` + "`" + `]+` + // HACK: nested make(1) variables
		`)+` + // any of the above may be repeated
		`|;;?` +
		`|&&?` +
		`|\|\|?` +
		`|\(` +
		`|\)` +
		`|>&` +
		`|<<?` +
		`|>>?` +
		`|#.*)`
	reVarname    = `(?:[-*+.0-9A-Z_a-z{}\[]+|\$\{[\w_]+\})+`
	rePkgbase    = `(?:[+.0-9A-Z_a-z]|-[A-Z_a-z])+`
	rePkgversion = `\d(?:\w|\.\d)*`
)

func explanationRelativeDirs() []string {
	return []string{
		"Directories in the form \"../../category/package\" make it easier to",
		"move a package around in pkgsrc, for example from pkgsrc-wip to the",
		"main pkgsrc repository."}
}

func checkItem(fname string) {
	defer tracecall("checkItem", fname)()

	st, err := os.Stat(fname)
	if err != nil || (!st.Mode().IsDir() && !st.Mode().IsRegular()) {
		logError(fname, NO_LINES, "No such file or directory.")
		return
	}
	isDir := st.Mode().IsDir()
	isReg := st.Mode().IsRegular()

	G.currentDir = fname
	if isReg {
		G.currentDir = path.Dir(fname)
	}
	abs, err := filepath.Abs(G.currentDir)
	if err != nil {
		logFatal(G.currentDir, NO_LINES, "Cannot determine absolute path.")
	}
	absCurrentDir := filepath.ToSlash(abs)
	G.isWip = !G.opts.optImport && match0(absCurrentDir, `/wip/|/wip$`)
	G.isInfrastructure = match0(absCurrentDir, `/mk/|/mk$`)
	G.curPkgsrcdir = nil
	pkgpath := ""
	for _, dir := range []string{".", "..", "../..", "../../.."} {
		if fileExists(G.currentDir + "/" + dir + "/mk/bsd.pkg.mk") {
			G.curPkgsrcdir = newStr(dir)
			pkgpath = relpath(G.currentDir, G.currentDir+"/"+dir)
		}
	}

	if pkgpath == "" {
		logError(fname, NO_LINES, "Cannot determine the pkgsrc root directory.")
		return
	}

	if isDir && isEmptyDir(fname) {
		return
	}

	if isReg {
		checkfile(fname)
	} else if *G.curPkgsrcdir == "" {
		logError(fname, NO_LINES, "Cannot check directories outside a pkgsrc tree.")
	} else if *G.curPkgsrcdir == "../.." {
		checkdirPackage(pkgpath)
	} else if *G.curPkgsrcdir == ".." {
		checkdirCategory()
	} else if *G.curPkgsrcdir == "." {
		checkdirToplevel()
	} else {
		logError(fname, NO_LINES, "Don't know how to check this directory.")
	}
}

func loadPackageMakefile(fname string) []*Line {
	G.pkgContext.included = make(map[string]*Line)

	mainLines := make([]*Line, 0)
	allLines := make([]*Line, 0)
	if !readMakefile(fname, &mainLines, &allLines) {
		logError(fname, NO_LINES, "Cannot be read.")
		return nil
	}

	if G.opts.optDumpMakefile {
		logDebug(G.currentDir, NO_LINES, "Whole Makefile (with all included files) follows:")
		for _, line := range allLines {
			fmt.Printf("%s\n", line.String())
		}
	}

	determineUsedVariables(allLines)

	G.pkgContext.pkgdir = newStr(expandVariableWithDefault("PKGDIR", "."))
	G.pkgContext.distinfoFile = expandVariableWithDefault("DISTINFO_FILE", "distinfo")
	G.pkgContext.filesdir = expandVariableWithDefault("FILESDIR", "files")
	G.pkgContext.patchdir = expandVariableWithDefault("PATCHDIR", "patches")

	if varIsDefined("PHPEXT_MK") {
		if !varIsDefined("USE_PHP_EXT_PATCHES") {
			G.pkgContext.patchdir = ("patches")
		}
		if varIsDefined("PECL_VERSION") {
			G.pkgContext.distinfoFile = ("distinfo")
		}
	}

	_ = G.opts.optDebugMisc &&
		logDebug(NO_FILE, NO_LINES, "DISTINFO_FILE=%s", G.pkgContext.distinfoFile) &&
		logDebug(NO_FILE, NO_LINES, "FILESDIR=%s", G.pkgContext.filesdir) &&
		logDebug(NO_FILE, NO_LINES, "PATCHDIR=%s", G.pkgContext.patchdir) &&
		logDebug(NO_FILE, NO_LINES, "PKGDIR=%s", *G.pkgContext.pkgdir)

	return mainLines
}

func determineUsedVariables(lines []*Line) {
	re := reCompile(`(?:\$\{|\$\(|defined\(|empty\()([0-9+.A-Z_a-z]+)[:})]`)
	for _, line := range lines {
		rest := line.text
		for {
			m := re.FindStringSubmatchIndex(rest)
			if m == nil {
				break
			}
			varname := rest[m[2]:m[3]]
			useVar(line, varname)
			rest = rest[:m[0]] + rest[m[1]:]
		}
	}
}

func extractUsedVariables(line *Line, text string) []string {
	re := reCompile(`^(?:[^\$]+|\$[\$*<>?\@]|\$\{([.0-9A-Z_a-z]+)(?::(?:[^\${}]|\$[^{])+)?\})`)
	rest := text
	result := make([]string, 0)
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
		_ = G.opts.optDebugMisc && line.logDebug("extractUsedVariables: rest=%v", rest)
	}
	return result
}

func getNbpart() string {
	line := G.pkgContext.vardef["PKGREVISION"]
	if line != nil {
		pkgrevision, err := strconv.Atoi(line.extra["value"].(string))
		if err != nil && pkgrevision != 0 {
			return sprintf("nb%d", pkgrevision)
		}
	}
	return ""
}

// Returns the type of the variable (maybe guessed based on the variable name),
// or nil if the type cannot even be guessed.
func getVariableType(line *Line, varname string) *Vartype {

	if vartype := G.globalData.getVartypes()[varname]; vartype != nil {
		return vartype
	}
	if vartype := G.globalData.getVartypes()[varnameCanon(varname)]; vartype != nil {
		return vartype
	}

	if G.globalData.varnameToToolname[varname] != "" {
		return newBasicVartype(LK_NONE, "ShellCommand", []AclEntry{{"*", "u"}}, NOT_GUESSED)
	}

	if m, toolvarname := match1(varname, `^TOOLS_(.*)`); m && G.globalData.varnameToToolname[toolvarname] != "" {
		return newBasicVartype(LK_NONE, "Pathname", []AclEntry{{"*", "u"}}, NOT_GUESSED)
	}

	allowAll := []AclEntry{{"*", "adpsu"}}
	allowRuntime := []AclEntry{{"*", "adsu"}}

	// Guess the datatype of the variable based on naming conventions.
	var gtype *Vartype
	switch {
	case hasSuffix(varname, "DIRS"):
		gtype = newBasicVartype(LK_EXTERNAL, "Pathmask", allowRuntime, GUESSED)
	case hasSuffix(varname, "DIR"), hasSuffix(varname, "_HOME"):
		gtype = newBasicVartype(LK_NONE, "Pathname", allowRuntime, GUESSED)
	case hasSuffix(varname, "FILES"):
		gtype = newBasicVartype(LK_EXTERNAL, "Pathmask", allowRuntime, GUESSED)
	case hasSuffix(varname, "FILE"):
		gtype = newBasicVartype(LK_NONE, "Pathname", allowRuntime, GUESSED)
	case hasSuffix(varname, "PATH"):
		gtype = newBasicVartype(LK_NONE, "Pathlist", allowRuntime, GUESSED)
	case hasSuffix(varname, "PATHS"):
		gtype = newBasicVartype(LK_EXTERNAL, "Pathname", allowRuntime, GUESSED)
	case hasSuffix(varname, "_USER"):
		gtype = newBasicVartype(LK_NONE, "UserGroupName", allowAll, GUESSED)
	case hasSuffix(varname, "_GROUP"):
		gtype = newBasicVartype(LK_NONE, "UserGroupName", allowAll, GUESSED)
	case hasSuffix(varname, "_ENV"):
		gtype = newBasicVartype(LK_EXTERNAL, "ShellWord", allowRuntime, GUESSED)
	case hasSuffix(varname, "_CMD"):
		gtype = newBasicVartype(LK_NONE, "ShellCommand", allowRuntime, GUESSED)
	case hasSuffix(varname, "_ARGS"):
		gtype = newBasicVartype(LK_EXTERNAL, "ShellWord", allowRuntime, GUESSED)
	case hasSuffix(varname, "_CFLAGS"), hasSuffix(varname, "_CPPFLAGS"), hasSuffix(varname, "_CXXFLAGS"), hasSuffix(varname, "_LDFLAGS"):
		gtype = newBasicVartype(LK_EXTERNAL, "ShellWord", allowRuntime, GUESSED)
	case hasSuffix(varname, "_MK"):
		gtype = newBasicVartype(LK_NONE, "Unchecked", allowAll, GUESSED)
	case hasPrefix(varname, "PLIST."):
		gtype = newBasicVartype(LK_NONE, "Yes", allowAll, GUESSED)
	}

	if gtype != nil {
		_ = G.opts.optDebugVartypes && line.logDebug("The guessed type of %v is %v.", varname, gtype)
	} else {
		_ = G.opts.optDebugVartypes && line.logDebug("No type definition found for %v.", varname)
	}
	return gtype
}

func resolveVariableRefs(text string) string {
	defer tracecall("resolveVariableRefs", text)()

	visited := make(map[string]bool) // To prevent endless loops

	str := text
	re := reCompile(`\$\{(\w+)\}`)
	for {
		replaced := re.ReplaceAllStringFunc(str, func(m string) string {
			varname := m[2 : len(m)-1]
			if !visited[varname] {
				visited[varname] = true
				trace("", "resolve.visit", varname)
				if G.pkgContext != nil && G.pkgContext.vardef[varname] != nil {
					return G.pkgContext.vardef[varname].extra["value"].(string)
				}
				if G.mkContext != nil && G.mkContext.vardef[varname] != nil {
					return G.mkContext.vardef[varname].extra["value"].(string)
				}
			}
			return sprintf("${%s}", varname)
		})
		trace("", "resolveStep", str, replaced)
		if replaced == str {
			return replaced
		}
		str = replaced
	}
}

func expandVariableWithDefault(varname, defaultValue string) string {
	line := G.pkgContext.vardef[varname]
	if line == nil {
		return defaultValue
	}

	value := line.extra["value"].(string)
	value = resolveVarsInRelativePath(value, true)
	if match0(value, reUnresolvedVar) {
		value = resolveVariableRefs(value)
		_ = G.opts.optDebugMisc && logDebug(NO_FILE, NO_LINES, "expandVariableWithDefault: failed varname=%q value=%q", varname, value)
	}
	return value
}

func getVariablePermissions(line *Line, varname string) string {
	vartype := getVariableType(line, varname)
	if vartype == nil {
		_ = G.opts.optDebugMisc && line.logDebug("No type definition found for %q.", varname)
		return "adpsu"
	}
	return vartype.effectivePermissions(line.fname)
}

func checklineLength(line *Line, maxlength int) {
	if len(line.text) > maxlength {
		line.logWarning("Line too long (should be no more than maxlength characters).")
		line.explainWarning(
			"Back in the old time, terminals with 80x25 characters were common.",
			"And this is still the default size of many terminal emulators.",
			"Moderately short lines also make reading easier.")
	}
}

func checklineValidCharacters(line *Line, reValidchars string) {
	rest := reCompile(reValidchars).ReplaceAllString(line.text, "")
	if rest != "" {
		uni := ""
		for _, c := range rest {
			uni += sprintf(" %U", c)
		}
		line.logWarning("Line contains invalid characters (%s).", uni[1:])
	}
}

func checklineValidCharactersInValue(line *Line, reValidchars string) {
	varname := line.extra["varname"].(string)
	value := line.extra["value"].(string)
	rest := reCompile(reValidchars).ReplaceAllString(value, "")
	if rest != "" {
		uni := ""
		for _, c := range rest {
			uni += sprintf(" %U", c)
		}
		line.logWarning("%s contains invalid characters (%s).", varname, uni[1:])
	}
}

func checklineTrailingWhitespace(line *Line) {
	if match0(line.text, `\s$`) {
		line.logNote("Trailing white-space.")
		line.explainNote(
			"When a line ends with some white-space, that space is in most cases",
			"irrelevant and can be removed, leading to a \"normal form\" syntax.")
		line.replaceRegex(`\s+\n$`, "\n")
	}
}

func checklineRcsid(line *Line, prefixRe, suggestedPrefix string) bool {
	defer tracecall("checklineRcsid", prefixRe, suggestedPrefix)()

	rcsid := "NetBSD"
	if G.isWip {
		rcsid = "Id"
	}

	if !match0(line.text, `^`+prefixRe+`\$`+rcsid+`(?::[^\$]+)?\$$`) {
		line.logError("Expected %s.", suggestedPrefix+"$"+rcsid+"$")
		line.explainError(
			"Several files in pkgsrc must contain the CVS Id, so that their current",
			"version can be traced back later from a binary package. This is to",
			"ensure reproducible builds, for example for finding bugs.",
			"",
			"Please insert the text from the above error message (without the quotes)",
			"at this position in the file.")
		return false
	}
	return true
}

func checklineMkAbsolutePathname(line *Line, text string) {
	defer tracecall("checklineMkAbsolutePathname", text)()

	// In the GNU coding standards, DESTDIR is defined as a (usually
	// empty) prefix that can be used to install files to a different
	// location from what they have been built for. Therefore
	// everything following it is considered an absolute pathname.
	//
	// Another context where absolute pathnames usually appear is in
	// assignments like "bindir=/bin".
	if m, path := match1(text, `(?:^|\$[({]DESTDIR[)}]|[\w_]+\s*=\s*)(/(?:[\w/*]|\"[\w/*]*\"|'[\w/*]*')*)`); m {
		if match0(path, `^/\w`) {
			checkwordAbsolutePathname(line, path)
		}
	}
}

func checklineRelativePath(line *Line, path string, mustExist bool) {
	if !G.isWip && contains(path, "/wip/") {
		line.logError("A main pkgsrc package must not depend on a pkgsrc-wip package.")
	}

	resolvedPath := resolveVarsInRelativePath(path, true)
	if match0(resolvedPath, reUnresolvedVar) {
		return
	}

	abs := ifelseStr(hasPrefix(resolvedPath, "/"), "", G.currentDir+"/") + resolvedPath
	if _, err := os.Stat(abs); err != nil {
		if mustExist {
			line.logError("%v does not exist.", resolvedPath)
		}
		return
	}

	switch {
	case match0(path, `^\.\./\.\./[^/]+/[^/]`):
	case hasPrefix(path, "../../mk/"):
		// There need not be two directory levels for mk/ files.
	case match0(path, `^\.\./mk/`) && *G.curPkgsrcdir == "..":
		// That's fine for category Makefiles.
	case match0(path, `^\.\.`):
		line.logWarning("Invalid relative path %q.", path)
	}
}

func checkfileExtra(fname string) {
	defer tracecall("checkfileExtra", fname)()

	checkperms(fname)
	lines := loadNonemptyLines(fname, false)
	if lines == nil {
		return
	}
	checklinesTrailingEmptyLines(lines)
}

func checkfileMessage(fname string) {
	defer tracecall("checkfileMessage", fname)()

	explanation := []string{
		"A MESSAGE file should consist of a header line, having 75 \"=\"",
		"characters, followed by a line containing only the RCS Id, then an",
		"empty line, your text and finally the footer line, which is the",
		"same as the header line."}

	checkperms(fname)
	lines := loadNonemptyLines(fname, false)
	if lines == nil {
		return
	}

	if len(lines) < 3 {
		lastLine := lines[len(lines)-1]
		lastLine.logWarning("File too short.")
		lastLine.explainWarning(explanation...)
		return
	}

	hline := strings.Repeat("=", 75)
	if line := lines[0]; line.text != hline {
		line.logWarning("Expected a line of exactly 75 \"=\" characters.")
		line.explainWarning(explanation...)
	}
	checklineRcsid(lines[1], ``, "")
	for _, line := range lines {
		checklineLength(line, 80)
		checklineTrailingWhitespace(line)
		checklineValidCharacters(line, reValidchars)
	}
	if lastLine := lines[len(lines)-1]; lastLine.text != hline {
		lastLine.logWarning("Expected a line of exactly 75 \"=\" characters.")
		lastLine.explainWarning(explanation...)
	}
	checklinesTrailingEmptyLines(lines)
}

func checklineRelativePkgdir(line *Line, pkgdir string) {
	checklineRelativePath(line, pkgdir, true)
	pkgdir = resolveVarsInRelativePath(pkgdir, false)

	if m, otherpkgpath := match1(pkgdir, `^(?:\./)?\.\./\.\./([^/]+/[^/]+)$`); m {
		if !fileExists(G.globalData.pkgsrcdir + "/" + otherpkgpath + "/Makefile") {
			line.logError("There is no package in otherpkgpath.")
		}

	} else {
		line.logWarning("%q is not a valid relative package directory.", pkgdir)
		line.explainWarning(
			"A relative pathname always starts with \"../../\", followed",
			"by a category, a slash and a the directory name of the package.",
			"For example, \"../../misc/screen\" is a valid relative pathname.")
	}
}

func checkfileMk(fname string) {
	defer tracecall("checkfileMk", fname)()

	checkperms(fname)
	lines := loadNonemptyLines(fname, true)
	if lines == nil {
		return
	}

	parselinesMk(lines)
	checklinesMk(lines)
	autofix(lines)
}

func checkfile(fname string) {
	defer tracecall("checkfile", fname)()

	basename := path.Base(fname)
	if match0(basename, `^(?:work.*|.*~|.*\.orig|.*\.rej)$`) {
		if G.opts.optImport {
			logError(fname, NO_LINES, "Must be cleaned up before committing the package.")
		}
		return
	}

	st, err := os.Lstat(fname)
	if err != nil {
		logError(fname, NO_LINES, "%s", err)
		return
	}

	switch {
	case st.Mode().IsDir():
		if basename == "files" || basename == "patches" || basename == "CVS" {
			// Ok
		} else if match0(fname, `(?:^|/)files/[^/]*$`) {
			// Ok

		} else if !isEmptyDir(fname) {
			logWarning(fname, NO_LINES, "Unknown directory name.")
		}

	case (st.Mode() & os.ModeSymlink) != 0:
		if !match0(basename, `^work`) {
			logWarning(fname, NO_LINES, "Unknown symlink name.")
		}

	case !st.Mode().IsRegular():
		logError(fname, NO_LINES, "Only files and directories are allowed in pkgsrc.")

	case basename == "ALTERNATIVES":
		if G.opts.optCheckAlternatives {
			checkfileExtra(fname)
		}
	case basename == "buildlink3.mk":
		if G.opts.optCheckBuildlink3 {
			checkfileBuildlink3Mk(fname)
		}
	case hasPrefix(basename, "DESCR"):
		if G.opts.optCheckDescr {
			checkfileDescr(fname)
		}

	case match0(basename, `^distinfo`):
		if G.opts.optCheckDistinfo {
			checkfileDistinfo(fname)
		}

	case basename == "DEINSTALL" || basename == "INSTALL":
		if G.opts.optCheckInstall {
			checkfileExtra(fname)
		}

	case match0(basename, `^MESSAGE`):
		if G.opts.optCheckMessage {
			checkfileMessage(fname)
		}

	case match0(basename, `^patch-[-A-Za-z0-9_.~+]*[A-Za-z0-9_]$`):
		if G.opts.optCheckPatches {
			checkfilePatch(fname)
		}

	case match0(fname, `(?:^|/)patches/manual[^/]*$`):
		if G.opts.optDebugUnchecked {
			logDebug(fname, NO_LINES, "Unchecked file %q.", fname)
		}

	case match0(fname, `(?:^|/)patches/[^/]*$`):
		logWarning(fname, NO_LINES, "Patch files should be named \"patch-\", followed by letters, '-', '_', '.', and digits only.")

	case match0(basename, `^(?:.*\.mk|Makefile.*)$`) && !match0(fname, `files/`) && !match0(fname, `patches/`):
		if G.opts.optCheckMk {
			checkfileMk(fname)
		}

	case match0(basename, `^PLIST`):
		if G.opts.optCheckPlist {
			checkfilePlist(fname)
		}

	case (basename == "TODO" || basename == "README"):
		// Ok

	case match0(basename, `^CHANGES-.*`):
		loadDocChanges(fname)

	case match0(fname, `(?:^|/)files/[^/]*$`):
		// Ok
	default:
		logWarning(fname, NO_LINES, "Unexpected file found.")
		if G.opts.optCheckExtra {
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
		lines[last].logNote("Trailing empty lines.")
	}
}
