// based on NetBSD: pkglint.pl,v 1.893 2015/10/15 03:00:56 rillig Exp $
package main

import (
	"fmt"
	"os"
	"path"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
)

type QuotingResult struct{ name string }

var (
	QR_FALSE         = QuotingResult{"false"}
	QR_TRUE          = QuotingResult{"true"}
	QR_DONT_KNOW     = QuotingResult{"don’t know"}
	QR_DOESNT_MATTER = QuotingResult{"doesn’t matter"}
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

func TestPrintTable() {
	printTable(os.Stdout, [][]string{{"hello", "world"}, {"how", "are", "you?"}})
}
func TestLogFatal() {
	(&Line{fname: "fname", lines: "13"}).logFatal("msg")
}
func TestGetLogicalLine() {
	var physlines = []PhysLine{{1, "first\\"}, {2, "second"}, {3, "third"}}
	var lineno int = 1
	fmt.Printf("%v\n", getLogicalLine("fname", physlines, &lineno))
	fmt.Printf("%#v\n", getLogicalLine("fname", physlines, &lineno))
}

// A Type in pkglint is a combination of a data type and a permission
// specification. Further details can be found in the chapter ``The pkglint
// type system'' of the pkglint book.

type KindOfList struct{ name string }

var LK_NONE = KindOfList{"none"}
var LK_INTERNAL = KindOfList{"internal"}
var LK_EXTERNAL = KindOfList{"external"}

type Guessed struct{ name string }

var GUESSED = Guessed{"guessed"}
var NOT_GUESSED = Guessed{"not guessed"}

type AclEntry struct {
	glob        string
	permissions string
}
type Type struct {
	kindOfList    KindOfList
	basicType     string
	enumValues    map[string]bool
	enumValuesStr string
	aclEntries    []AclEntry
	guessed       Guessed
}

func (self *Type) effectivePermissions(fname string) string {
	for _, aclEntry := range self.aclEntries {
		if m, _ := path.Match(aclEntry.glob, fname); m {
			return aclEntry.permissions
		}
	}
	return ""
}

// Returns the union of all possible permissions. This can be used to
// check whether a variable may be defined or used at all, or if it is
// read-only.
func (self *Type) union() (perms string) {
	for _, aclEntry := range self.aclEntries {
		perms += aclEntry.permissions
	}
	return
}

// This distinction between “real lists” and “considered a list” makes
// the implementation of checklineMkVartype easier.
func (self *Type) isConsideredList() bool {
	switch {
	case self.kindOfList == LK_EXTERNAL:
		return true
	case self.kindOfList == LK_INTERNAL:
		return false
	case self.basicType == "BuildlinkPackages":
		return true
	case self.basicType == "SedCommands":
		return true
	case self.basicType == "ShellCommand":
		return true
	default:
		return false
	}
}
func (self *Type) mayBeAppendedTo() bool {
	return self.kindOfList != LK_NONE ||
		self.basicType == "AwkCommand" ||
		self.basicType == "BuildlinkPackages" ||
		self.basicType == "SedCommands"
}
func (self *Type) String() string {
	switch self.kindOfList {
	case LK_NONE:
		return self.basicType
	case LK_INTERNAL:
		return "InternalList of " + self.basicType
	case LK_EXTERNAL:
		return "List of " + self.basicType
	default:
		panic("")
		return ""
	}
}
func (t *Type) isGuessed() bool {
	return t.guessed == GUESSED
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

type VarUseContext struct {
	time      VarUseContextTime
	vartype   *Type
	shellword VarUseContextShellword
	extent    VarUseContextExtent
}

func (self *VarUseContext) String() string {
	typename := "no-type"
	if self.vartype != nil {
		typename = self.vartype.String()
	}
	return fmt.Sprintf("(%s %s %s %s)",
		[]string{"unknown-time", "load-time", "run-time"}[self.time],
		typename,
		[]string{"none", "plain", "dquot", "squot", "backt", "for"}[self.shellword],
		[]string{"unknown", "full", "word", "word-part"}[self.extent])
}

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
	// Note: the following regular expression looks more complicated than
	// necessary to avoid a stack overflow in the Perl interpreter.
	// The leading white-space may only consist of \040 characters, otherwise
	// the order of regex_varassign and regex_mk_shellcmd becomes important.
	reVarassign   = `^ *([-*+A-Z_a-z0-9.\${}\[]+?)\s*(=|\?=|\+=|:=|!=)\s*((?:[^\\#\s]+|\s+?|(?:\\#)+|\\)*?)(?:\s*(#.*))?$`
	reShVarassign = `^([A-Z_a-z][0-9A-Z_a-z]*)=`
	// This regular expression cannot parse all kinds of shell programs, but
	// it will catch almost all shell programs that are portable enough to be
	// used in pkgsrc.
	reShellword = `(?sx)\s*(
	\#.*				# shell comment
	|
	(?:	'[^']*'			# single quoted string
	|	\"(?:\\.|[^\"\\])*\"	# double quoted string
	` + "|	`[^`]*`" + `		# backticks string
	|	\\\$\$			# an escaped dollar sign
	|	\\[^\$]			# other escaped characters
	|	\$[\w_]			# one-character make(1) variable
	|	\$\{[^{}]+\}		# make(1) variable
	|	\$\([^()]+\)		# make(1) variable, $(...)
	|	\$[/\@<^]		# special make(1) variables
	|	\$\$[0-9A-Z_a-z]+	# shell variable
	|	\$\$[\#?@]		# special shell variables
	|	\$\$\$\$		# the special pid shell variable
	|	\$\$\{[0-9A-Z_a-z]+\}	# shell variable in braces
	|	\$\$\(			# POSIX-style backticks replacement
	|	[^\(\)'\"\\\s;&\|<>` + "`" + `\$] # non-special character
	|	\$\{[^\s\"'` + "`" + `]+		# HACK: nested make(1) variables
	)+ | ;;? | &&? | \|\|? | \( | \) | >& | <<? | >>? | \#.*)`
	reVarname       = `(?:[-*+.0-9A-Z_a-z{}\[]+|\$\{[\w_]+\})+`
	rePkgbase       = `(?:[+.0-9A-Z_a-z]|-[A-Z_a-z])+`
	rePkgversion    = `\d(?:\w|\.\d)*`
	reVarnamePlural = `(?x:
		| .*S
		| .*LIST
		| .*_AWK
		| .*_ENV
		| .*_REQD
		| .*_SED
		| .*_SKIP
		| BUILDLINK_LDADD
		| COMMENT
		| EXTRACT_ONLY
		| FETCH_MESSAGE
		| GENERATE_PLIST
		| PLIST_CAT
		| PLIST_PRE
		| PREPEND_PATH

	# Existing plural variables whose name doesn’t indicate plural
		| .*_OVERRIDE
		| .*_PREREQ
		| .*_SRC
		| .*_SUBST
		| .*_TARGET
		| .*_TMPL
		| BROKEN_EXCEPT_ON_PLATFORM
		| BROKEN_ON_PLATFORM
		| BUILDLINK_DEPMETHOD
		| BUILDLINK_TRANSFORM
		| EVAL_PREFIX
		| INTERACTIVE_STAGE
		| LICENSE
		| MASTER_SITE_.*
		| MASTER_SORT_REGEX
		| NOT_FOR_COMPILER
		| NOT_FOR_PLATFORM
		| ONLY_FOR_COMPILER
		| ONLY_FOR_PLATFORM
		| PERL5_PACKLIST
		| PKG_FAIL_REASON
		| PKG_SKIP_REASON
		| CRYPTO
		| DEINSTALL_TEMPLATE
		| FIX_RPATH
		| INSTALL_TEMPLATE
		| PYTHON_VERSIONS_INCOMPATIBLE
		| REPLACE_INTERPRETER
		| REPLACE_PERL
		| REPLACE_RUBY
		| RESTRICTED
		| SITES_.*
		| TOOLS_ALIASES\.*
		| TOOLS_BROKEN
		| TOOLS_CREATE
		| TOOLS_GNU_MISSING
		| TOOLS_NOOP)`
)

func explanationRelativeDirs() []string {
	return []string{
		"Directories in the form \"../../category/package\" make it easier to",
		"move a package around in pkgsrc, for example from pkgsrc-wip to the",
		"main pkgsrc repository."}
}

func checkItem(fname string) {
	st, err := os.Stat(fname)
	if err != nil || (!st.Mode().IsDir() && !st.Mode().IsRegular()) {
		logError(fname, NO_LINES, "No such file or directory.")
		return
	}
	isDir := st.Mode().IsDir()
	isReg := st.Mode().IsRegular()

	currentDir := fname
	if isReg {
		currentDir = path.Dir(fname)
	}
	abs, err := filepath.Abs(currentDir)
	if err != nil {
		logFatal(currentDir, NO_LINES, "Cannot determine absolute path.")
	}
	absCurrentDir := filepath.ToSlash(abs)
	G.isWip = !G.opts.optImport && match(absCurrentDir, `/wip/|/wip$`) != nil
	G.isInternal = match(absCurrentDir, `/mk/|/mk$`) != nil
	G.curPkgsrcdir = nil
	G.pkgContext.pkgpath = nil
	for _, dir := range []string{".", "..", "../..", "../../.."} {
		fname := currentDir + "/" + dir + "/mk/bsd.pkg.mk"
		if fst, err := os.Stat(fname); err == nil && fst.Mode().IsRegular() {
			*G.curPkgsrcdir = dir
			*G.pkgContext.pkgpath, err = filepath.Rel(currentDir, currentDir+"/"+dir)
			if err != nil {
				logFatal(currentDir, NO_LINES, "Cannot determine relative dir.")
			}
		}
	}
	if *G.cwdPkgsrcdir == "" && *G.curPkgsrcdir != "" {
		*G.cwdPkgsrcdir = currentDir + "/" + *G.curPkgsrcdir
	}

	if *G.cwdPkgsrcdir == "" {
		logError(fname, NO_LINES, "Cannot determine the pkgsrc root directory.")
		return
	}

	if isDir && isEmptyDir(fname) {
		return
	}

	if isDir {
		checkdirCvs(fname)
	}

	if isReg {
		checkfile(fname)
	} else if *G.curPkgsrcdir == "" {
		logError(fname, NO_LINES, "Cannot check directories outside a pkgsrc tree.")
	} else if *G.curPkgsrcdir == "../.." {
		checkdirPackage()
	} else if *G.curPkgsrcdir == ".." {
		checkdirCategory()
	} else if *G.curPkgsrcdir == "." {
		checkdirToplevel()
	} else {
		logError(fname, NO_LINES, "Don't know how to check this directory.")
	}
}

func loadPackageMakefile(fname string) []*Line {
	lines := make([]*Line, 0)
	allLines := make([]*Line, 0)
	G.pkgContext.included = make(map[string]*Line)

	if !readMakefile(fname, lines, allLines) {
		logError(fname, NO_LINES, "Cannot be read.")
		return nil
	}

	if G.opts.optDumpMakefile {
		logDebug(NO_FILE, NO_LINES, "Whole Makefile (with all included files) follows:")
		for _, line := range lines {
			fmt.Printf("%s\n", line.String())
		}
	}

	determineUsedVariables(allLines)

	G.pkgContext.pkgdir = newStr(expandVariableWithDefault("PKGDIR", "."))
	G.pkgContext.distinfoFile = (expandVariableWithDefault("DISTINFO_FILE", "distinfo"))
	G.pkgContext.filesdir = (expandVariableWithDefault("FILESDIR", "files"))
	G.pkgContext.patchdir = (expandVariableWithDefault("PATCHDIR", "patches"))

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

	return lines
}

func findPkgsrcTopdir() string {
	return "C:/Users/rillig/Desktop/pkgsrc/pkgsrc"
}

func determineUsedVariables(lines []*Line) {
	re := regexp.MustCompile(`(?:\$\{|\$\(|defined\(|empty\()([0-9+.A-Z_a-z]+)[:})]`)
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
	re := regexp.MustCompile(`^(?:[^\$]+|\$[\$*<>?\@]|\$\{([.0-9A-Z_a-z]+)(?::(?:[^\${}]|\$[^{])+)?\})`)
	rest := text
	result := make([]string, 0)
	for {
		m := re.FindStringSubmatchIndex(rest)
		if m == nil {
			break
		}
		varname := rest[m[2]:m[3]]
		rest = rest[:m[0]] + rest[m[1]:]
		result = append(result, varname)
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
			return fmt.Sprintf("nb%d", pkgrevision)
		}
	}
	return ""
}

// Returns the type of the variable (maybe guessed based on the variable name),
// or nil if the type cannot even be guessed.
func getVariableType(line *Line, varname string) *Type {

	vartype := G.globalData.vartypes[varname]
	if vartype == nil {
		vartype = G.globalData.vartypes[varnameCanon(varname)]
	}

	if G.globalData.varnameToToolname[varname] != "" {
		return &Type{LK_NONE, "ShellCommand", nil, "", []AclEntry{{"*", "u"}}, NOT_GUESSED}
	}

	if m, toolvarname := match1(varname, `^TOOLS_(.*)`); m && G.globalData.varnameToToolname[toolvarname] != "" {
		return &Type{LK_NONE, "Pathname", nil, "", []AclEntry{{"*", "u"}}, NOT_GUESSED}
	}

	allowAll := []AclEntry{{"*", "adpsu"}}
	allowRuntime := []AclEntry{{"*", "adsu"}}

	// Guess the datatype of the variable based on naming conventions.
	var gtype *Type
	if m, suffix := match1(varname, `(DIRS|DIR|FILES|FILE|PATH|PATHS|_USER|_GROUP|_ENV|_CMD|_ARGS|_CFLAGS|_CPPFLAGS|_CXXFLAGS|_LDFLAGS|_MK)$`); m {
		switch suffix {
		case "DIRS":
			gtype = &Type{LK_EXTERNAL, "Pathmask", nil, "", allowRuntime, GUESSED}
		case "DIR", "_HOME":
			gtype = &Type{LK_NONE, "Pathname", nil, "", allowRuntime, GUESSED}
		case "FILES":
			gtype = &Type{LK_EXTERNAL, "Pathmask", nil, "", allowRuntime, GUESSED}
		case "FILE":
			gtype = &Type{LK_NONE, "Pathname", nil, "", allowRuntime, GUESSED}
		case "PATH":
			gtype = &Type{LK_NONE, "Pathlist", nil, "", allowRuntime, GUESSED}
		case "PATHS":
			gtype = &Type{LK_EXTERNAL, "Pathname", nil, "", allowRuntime, GUESSED}
		case "_USER":
			gtype = &Type{LK_NONE, "UserGroupName", nil, "", allowAll, GUESSED}
		case "_GROUP":
			gtype = &Type{LK_NONE, "UserGroupName", nil, "", allowAll, GUESSED}
		case "_ENV":
			gtype = &Type{LK_EXTERNAL, "ShellWord", nil, "", allowRuntime, GUESSED}
		case "_CMD":
			gtype = &Type{LK_NONE, "ShellCommand", nil, "", allowRuntime, GUESSED}
		case "_ARGS":
			gtype = &Type{LK_EXTERNAL, "ShellWord", nil, "", allowRuntime, GUESSED}
		case "_CFLAGS", "_CPPFLAGS", "_CXXFLAGS", "_LDFLAGS":
			gtype = &Type{LK_EXTERNAL, "ShellWord", nil, "", allowRuntime, GUESSED}
		case "_MK":
			gtype = &Type{LK_NONE, "Unchecked", nil, "", allowAll, GUESSED}
		}
	} else if strings.HasPrefix(varname, "PLIST.") {
		gtype = &Type{LK_NONE, "Yes", nil, "", allowAll, GUESSED}
	}

	if gtype != nil {
		_ = G.opts.optDebugVartypes && line.logDebug("The guessed type of %v is %v.", varname, gtype)
	} else {
		_ = G.opts.optDebugVartypes && line.logDebug("No type definition found for %v.", varname)
	}
	return gtype
}

func resolveVariableRefs(text string) string {
	visited := make(map[string]bool) // To prevent endless loops

	str := text
	re := regexp.MustCompile(`\$\{(\w+)\}`)
	for {
		replaced := re.ReplaceAllStringFunc(str, func(varname string) string {
			if !visited[varname] {
				visited[varname] = true
				if G.pkgContext != nil && G.pkgContext.vardef[varname] != nil {
					return G.pkgContext.vardef[varname].extra["value"].(string)
				}
				if G.mkContext != nil && G.mkContext.vardef[varname] != nil {
					return G.mkContext.vardef[varname].extra["value"].(string)
				}
			}
			return fmt.Sprintf("${%s}", varname)
		})
		if replaced == str {
			return replaced
		}
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

func checklineValidCharacters(line *Line, subject, reValidchars string) {
	rest := reCompile(reValidchars).ReplaceAllString(line.text, "")
	if rest != "" {
		uni := ""
		for _, c := range rest {
			uni += fmt.Sprintf(" %U", c)
		}
		line.logWarning("%s contains invalid characters (%s).", subject, uni[1:])
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
	id := G.opts.optRcsIds
	if G.isWip {
		id += "|Id"
	}

	_ = G.opts.optDebugTrace && line.logDebug("checkline_rcsid_regex(%v, %v)", prefixRe, suggestedPrefix)

	if !match0(line.text, `^`+prefixRe+`$(`+id+`)(?::[^\$]+|)\$$`) {
		line.logError("Expected %s.", suggestedPrefix+"$"+G.opts.optRcsIds+"$")
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
	_ = G.opts.optDebugTrace && line.logDebug("checkline_mk_absolute_pathname(%v)", text)

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
	if !G.isWip && strstr(path, "/wip/") {
		line.logError("A main pkgsrc package must not depend on a pkgsrc-wip package.")
	}

	resolvedPath := resolveVarsInRelativePath(path, true)
	if match0(resolvedPath, reUnresolvedVar) {
		return
	}

	abs := ifelseStr(strings.HasPrefix(resolvedPath, "/"), "", G.currentDir+"/") + resolvedPath
	if _, err := os.Stat(abs); err != nil {
		if mustExist {
			line.logError("%v does not exist.", resolvedPath)
		}
		return
	}

	switch {
	case match0(path, `^\.\./\.\./[^/]+/[^/]`):
	case strings.HasPrefix(path, "../../mk/"):
		// There need not be two directory levels for mk/ files.
	case match0(path, `^\.\./mk/`) && *G.curPkgsrcdir == "..":
		// That's fine for category Makefiles.
	case match0(path, `^\.\.`):
		line.logWarning("Invalid relative path %q.", path)
	}
}

func checkfileAlternatives(fname string) {
	_ = G.opts.optDebugTrace && logDebug(fname, NO_LINES, "checkfileAlternatives()")

	checkperms(fname)
	if _, err := loadLines(fname, false); err != nil {
		logError(fname, NO_LINES, "Cannot be read.")
	}
}

func checkfileExtra(fname string) {
	_ = G.opts.optDebugTrace && logDebug(fname, NO_LINES, "checkfileExtra()")

	checkperms(fname)
	lines := loadNonemptyLines(fname, false)
	if lines == nil {
		return
	}
	checklinesTrailingEmptyLines(lines)
}

var checkfileInstall = checkfileExtra

func checkfileMessage(fname string) {
	_ = G.opts.optDebugTrace && logDebug(fname, NO_LINES, "checkfileMessage()")

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
		checklineValidCharacters(line, "Line", reValidchars)
	}
	if lastLine := lines[len(lines)-1]; lastLine.text != hline {
		lastLine.logWarning("Expected a line of exactly 75 \"=\" characters.")
		lastLine.explainWarning(explanation...)
	}
	checklinesTrailingEmptyLines(lines)
}

func parseLicenses(licenses string) []string {
	noPerl := strings.Replace(licenses, "${PERL5_LICENSE}", "gnu-gpl-v2 OR artistic", -1)
	noOps := reCompile(`[()]|AND|OR`).ReplaceAllString(noPerl, "") // cheated
	return splitOnSpace(noOps)
}

func checklineRelativePkgdir(line *Line, pkgdir string) {
	checklineRelativePath(line, pkgdir, true)
	pkgdir = resolveVarsInRelativePath(pkgdir, false)

	if m, otherpkgpath := match1(pkgdir, `^(?:\./)?\.\./\.\./([^/]+/[^/]+)$`); m {
		if !fileExists(*G.cwdPkgsrcdir + "/" + otherpkgpath + "/Makefile") {
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
	_ = G.opts.optDebugTrace && logDebug(fname, NO_LINES, "checkfileMk()")

	checkperms(fname)
	lines := loadNonemptyLines(fname,true)
	if lines == nil {
		return
		}

	parselinesMk(lines)
	checklinesMk(lines)
	autofix(lines)
}
