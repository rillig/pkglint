// based on NetBSD: pkglint.pl,v 1.893 2015/10/15 03:00:56 rillig Exp $
package main

import (
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"path"
	"regexp"
	"strconv"
	"strings"
)

const confMake = "@BMAKE@"
const confDatadir = "@DATADIR@"
const confVersion = "@VERSION@"

func intMax(a, b int) int {
	if a > b {
		return a
	}
	return b
}
func ifelseStr(cond bool, a, b string) string {
	if cond {
		return a
	}
	return b
}

type QuotingResult struct{ name string }

var (
	QR_FALSE         = QuotingResult{"false"}
	QR_TRUE          = QuotingResult{"true"}
	QR_DONT_KNOW     = QuotingResult{"don’t know"}
	QR_DOESNT_MATTER = QuotingResult{"doesn’t matter"}
)

const NO_FILE string = ""
const NO_LINES string = ""

type GlobalVarsType struct {
	errors          int
	warnings        int
	gccOutputFormat bool
	explainFlag     bool
	showSourceFlag  bool
	quiet           bool
	optWarnExtra    bool
	cwdPkgsrcdir    *string // The pkgsrc directory, relative to the current working directory of pkglint.
	curPkgsrcdir    *string // The pkgsrc directory, relative to the directory that is currently checked.
	opts            *CmdOpts

	currentDir string // The currently checked directory.
	isWip      bool   // Is the current directory from pkgsrc-wip?
	isInternal bool   // Is the currently checked item from the pkgsrc infrastructure?

	ipcDistinfo                map[string]string // Maps "alg:fname" => "checksum".
	ipcUsedLicenses            map[string]bool
	ipcCheckingRootRecursively bool // Only in this case is ipcUsedLicenses filled.
  todo []string // The list of directory entries that still need to be checked. Mostly relevant with --recursive.
}

var GlobalVars = GlobalVarsType{}

type LogLevel struct{ traditionalName, gccName string }

var (
	LL_FATAL = LogLevel{"FATAL", "fatal"}
	LL_ERROR = LogLevel{"ERROR", "error"}
	LL_WARN  = LogLevel{"WARN", "warning"}
	LL_NOTE  = LogLevel{"NOTE", "note"}
	LL_DEBUG = LogLevel{"DEBUG", "debug"}
)

func logMessage(level LogLevel, fname, lineno, message string) {
	if fname != NO_FILE {
		fname = path.Clean(fname)
	}

	text, sep := "", ""
	if !GlobalVars.gccOutputFormat {
		text += sep + level.traditionalName + ":"
		sep = " "
	}
	if fname != NO_FILE {
		text += sep + fname
		sep = ": "
		if lineno != NO_LINES {
			text += ":" + lineno
		}
	}
	if GlobalVars.gccOutputFormat {
		text += sep + level.gccName + ":"
		sep = " "
	}
	text += sep + message + "\n"
	if level == LL_FATAL {
		io.WriteString(os.Stderr, text)
	} else {
		io.WriteString(os.Stdout, text)
	}
}

func logFatal(fname, lineno, message string) {
	logMessage(LL_FATAL, fname, lineno, message)
	os.Exit(1)
}
func logError(fname, lineno, message string) {
	logMessage(LL_ERROR, fname, lineno, message)
	GlobalVars.errors++
}
func logWarning(fname, lineno, message string) {
	logMessage(LL_WARN, fname, lineno, message)
	GlobalVars.warnings++
}
func logNote(fname, lineno, message string) {
	logMessage(LL_NOTE, fname, lineno, message)
}
func logDebug(fname, lineno, message string) {
	logMessage(LL_DEBUG, fname, lineno, message)
}

func explain(level LogLevel, fname, lineno string, explanation []string) {
	if GlobalVars.explainFlag {
		out := os.Stdout
		if level == LL_FATAL {
			out = os.Stderr
		}
		for _, explanationLine := range explanation {
			io.WriteString(out, "\t"+explanationLine+"\n")
		}
	}
}

func printSummary() {
	if !GlobalVars.quiet {
		if GlobalVars.errors != 0 && GlobalVars.warnings != 0 {
			fmt.Printf("%d errors and %d warnings found.", GlobalVars.errors, GlobalVars.warnings)
			if !GlobalVars.explainFlag {
				fmt.Printf("(Use -e for more details.)")
			}
			fmt.Printf("\n")
		} else {
			fmt.Printf("looks fine.\n")
		}
	}
}

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

// When files are read in by pkglint, they are interpreted in terms of
// lines. For Makefiles, line continuations are handled properly, allowing
// multiple physical lines to end in a single logical line. For other files
// there is a 1:1 translation.
//
// A difference between the physical and the logical lines is that the
// physical lines include the line end sequence, whereas the logical lines
// do not.
//
// A logical line is a class having the read-only fields C<file>,
// C<lines>, C<text>, C<physlines> and C<is_changed>, as well as some
// methods for printing diagnostics easily.
//
// Some other methods allow modification of the physical lines, but leave
// the logical line (the C<text>) untouched. These methods are used in the
// --autofix mode.
//
// A line can have some "extra" fields that allow the results of parsing to
// be saved under a name.
//
type PhysLine struct {
	lineno int
	textnl string
}

type Line struct {
	fname     string
	lines     string
	text      string
	physlines []PhysLine
	changed   bool
	before    []PhysLine
	after     []PhysLine
	extra     map[string]string
}

func NewLine(fname, linenos, text string, physlines []PhysLine) *Line {
	return &Line{fname, linenos, text, physlines, false, []PhysLine{}, []PhysLine{}, make(map[string]string, 1)}
}
func (self *Line) physicalLines() []PhysLine {
	return append(self.before, append(self.physlines, self.after...)...)
}
func (self *Line) printSource(out io.Writer) {
	if GlobalVars.showSourceFlag {
		io.WriteString(out, "\n")
		for _, physline := range self.physicalLines() {
			fmt.Fprintf(out, "> %s", physline.textnl)
		}
	}
}
func (self *Line) logFatal(msg string) {
	self.printSource(os.Stderr)
	logFatal(self.fname, self.lines, msg)
}
func (self *Line) logError(msg string) {
	self.printSource(os.Stdout)
	logError(self.fname, self.lines, msg)
}
func (self *Line) logWarning(msg string) {
	self.printSource(os.Stdout)
	logWarning(self.fname, self.lines, msg)
}
func (self *Line) logNote(msg string) {
	self.printSource(os.Stdout)
	logNote(self.fname, self.lines, msg)
}
func (self *Line) logDebug(msg string) {
	self.printSource(os.Stdout)
	logDebug(self.fname, self.lines, msg)
}
func (self *Line) explainError(explanation ...string) {
	explain(LL_ERROR, self.fname, self.lines, explanation)
}
func (self *Line) explainWarning(explanation ...string) {
	explain(LL_WARN, self.fname, self.lines, explanation)
}
func (self *Line) explainNote(explanation ...string) {
	explain(LL_NOTE, self.fname, self.lines, explanation)
}
func (self *Line) String() string {
	return self.fname + ":" + self.lines + ": " + self.text
}
func (self *Line) prependBefore(line string) {
	self.before = append([]PhysLine{{0, line + "\n"}}, self.before...)
	self.changed = true
}
func (self *Line) appendBefore(line string) {
	self.before = append(self.before, PhysLine{0, line + "\n"})
	self.changed = true
}
func (self *Line) prependAfter(line string) {
	self.after = append([]PhysLine{{0, line + "\n"}}, self.after...)
	self.changed = true
}
func (self *Line) appendAfter(line string) {
	self.after = append(self.after, PhysLine{0, line + "\n"})
	self.changed = true
}
func (self *Line) delete() {
	self.physlines = []PhysLine{}
	self.changed = true
}
func (self *Line) replace(from, to string) {
	for _, physline := range self.physlines {
		if physline.lineno != 0 {
			if replaced := strings.Replace(physline.textnl, from, to, 1); replaced != physline.textnl {
				physline.textnl = replaced
				self.changed = true
			}
		}
	}
}
func (self *Line) replaceRegex(from, to string) {
	for _, physline := range self.physlines {
		if physline.lineno != 0 {
			if replaced := regexp.MustCompile(from).ReplaceAllString(physline.textnl, to); replaced != physline.textnl {
				physline.textnl = replaced
				self.changed = true
			}
		}
	}
}
func (line *Line) setText(text string) {
	line.physlines = []PhysLine{{0, text + "\n"}}
	line.changed = true
}

func loadRawLines(fname string) ([]PhysLine, error) {
	physlines := make([]PhysLine, 50)
	rawtext, err := ioutil.ReadFile(fname)
	if err != nil {
		logError(fname, NO_LINES, "Cannot be read")
		return nil, err
	}
	for lineno, physline := range strings.SplitAfter(string(rawtext), "\n") {
		physlines = append(physlines, PhysLine{lineno, physline})
	}
	return physlines, nil
}

func getLogicalLine(fname string, physlines []PhysLine, pLineno *int) *Line {
	value := ""
	first := true
	lineno := *pLineno
	firstlineno := physlines[lineno].lineno
	lphyslines := make([]PhysLine, 1)

	for _, physline := range physlines {
		m := regexp.MustCompile(`^([ \t]*)(.*?)([ \t]*)(\\?)\n?$`).FindStringSubmatch(physline.textnl)
		indent, text, outdent, cont := m[1], m[2], m[3], m[4]

		if first {
			value += indent
			first = false
		}

		value += text
		lphyslines = append(lphyslines, physline)

		if cont == "\\" {
			value += " "
		} else {
			value += outdent
			break
		}
	}

	if lineno >= len(physlines) { // The last line in the file is a continuation line
		lineno--
	}
	lastlineno := physlines[lineno].lineno
	*pLineno = lineno + 1

	slineno := ifelseStr(firstlineno == lastlineno, fmt.Sprintf("%d", firstlineno), fmt.Sprintf("%d–%d", firstlineno, lastlineno))
	return NewLine(fname, slineno, value, physlines)
}

func loadLines(fname string, joinContinuationLines bool) ([]*Line, error) {
	physlines, err := loadRawLines(fname)
	if err != nil {
		return nil, err
	}
	return convertToLogicalLines(fname, physlines, joinContinuationLines)
}

func convertToLogicalLines(fname string, physlines []PhysLine, joinContinuationLines bool) ([]*Line, error) {
	loglines := make([]*Line, 0, len(physlines))
	if joinContinuationLines {
		for lineno := 0; lineno < len(physlines); {
			loglines = append(loglines, getLogicalLine(fname, physlines, &lineno))
		}
	} else {
		for _, physline := range physlines {
			loglines = append(loglines, NewLine(fname, strconv.Itoa(physline.lineno), strings.TrimSuffix(physline.textnl, "\n"), []PhysLine{physline}))
		}
	}

	if 0 < len(physlines) && !strings.HasSuffix(physlines[len(physlines)-1].textnl, "\n") {
		logError(fname, strconv.Itoa(physlines[len(physlines)-1].lineno), "File must end with a newline.")
	}

	return loglines, nil
}

func saveAutofixChanges(lines []Line) {
	changes := make(map[string][]PhysLine)
	changed := make(map[string]bool)
	for _, line := range lines {
		if line.changed {
			changed[line.fname] = true
		}
		changes[line.fname] = append(changes[line.fname], line.physicalLines()...)
	}

	for fname := range changed {
		physlines := changes[fname]
		tmpname := fname + ".pkglint.tmp"
		text := ""
		for _, physline := range physlines {
			text += physline.textnl
		}
		err := ioutil.WriteFile(tmpname, []byte(text), 0777)
		if err != nil {
			logError(tmpname, NO_LINES, "Cannot write.")
			continue
		}
		err = os.Rename(tmpname, fname)
		if err != nil {
			logError(fname, NO_LINES, "Cannot overwrite with auto-fixed content.")
			continue
		}
		logNote(fname, NO_LINES, "Has been auto-fixed. Please re-run pkglint.")
	}
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
func TestConvertToLogicalLines() {
	var physlines = []PhysLine{{1, "continuation in last line\\"}}
	lines, _ := convertToLogicalLines("fname", physlines, true)
	fmt.Printf("%s\n", lines)
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
	subjectPattern regexp.Regexp
	permissions    string
}
type Type struct {
	kindOfList KindOfList
	basicType  string
	aclEntries []AclEntry
	guessed    bool
}

func (self *Type) effectivePermissions(fname string) string {
	for _, aclEntry := range self.aclEntries {
		if aclEntry.subjectPattern.MatchString(fname) {
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
	}
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

// Records the state of a block of variable assignments that make up a SUBST
// class (see mk/subst.mk).
type SubstContext struct {
	id        *string
	class     *string
	stage     *string
	message   *string
	files     []string
	sed       []string
	vars      []string
	filterCmd *string
}

func (self *SubstContext) isComplete() bool {
	return self.id != nil && self.class != nil && len(self.files) != 0 && (len(self.sed) != 0 || len(self.vars) != 0 || self.filterCmd != nil)
}
func (self *SubstContext) checkVarassign(line *Line, varname, op, value string) {
	if !GlobalVars.optWarnExtra {
		return
	}

	if varname == "SUBST_CLASSES" {
		classes := regexp.MustCompile(`\s+`).Split(value, -1)
		if len(classes) > 1 {
			line.logWarning("Please add only one class at a time to SUBST_CLASSES.")
		}
		if self.class != nil {
			line.logWarning("SUBST_CLASSES should only appear once in a SUBST block.")
		}
		self.id = &classes[0]
		self.class = &classes[0]
		return
	}

	var varbase, varparam string
	if m := regexp.MustCompile(`^(SUBST_(?:STAGE|MESSAGE|FILES|SED|VARS|FILTER_CMD))\.([\-\w_]+)$`).FindStringSubmatch(varname); m != nil {
		varbase, varparam = m[1], m[2]
		if self.id == nil {
			line.logWarning("SUBST_CLASSES should precede the definition of " + varname + ".")
			self.id = &varparam
		}
	} else if self.id != nil {
		line.logWarning("Foreign variable in SUBST block.")
	}

	if varparam != *self.id {
		if self.isComplete() {
			// XXX: This code sometimes produces weird warnings. See
			// meta-pkgs/xorg/Makefile.common 1.41 for an example.
			self.finish(line)

			// The following assignment prevents an additional warning,
			// but from a technically viewpoint, it is incorrect.
			self.class = &varparam
			self.id = &varparam
		} else {
			line.logWarning(fmt.Sprintf("Variable parameter \"%s\" does not match SUBST class \"%s\".", varparam, self.id))
		}
		return
	}

	switch varbase {
	case "SUBST_STAGE":
		if self.stage != nil {
			line.logWarning("Duplicate definition of " + varname + ".")
		}
		self.stage = &value
	case "SUBST_MESSAGE":
		if self.message != nil {
			line.logWarning("Duplicate definition of " + varname + ".")
		}
		self.message = &value
	case "SUBST_FILES":
		if len(self.files) > 0 && op != "+=" {
			line.logWarning("All but the first SUBST_FILES line should use the \"+=\" operator.")
		}
		self.files = append(self.files, value)
	case "SUBST_SED":
		if len(self.sed) > 0 && op != "+=" {
			line.logWarning("All but the first SUBST_SED line should use the \"+=\" operator.")
		}
		self.sed = append(self.sed, value)
	case "SUBST_FILTER_CMD":
		if self.filterCmd != nil {
			line.logWarning("Duplicate definition of " + varname + ".")
		}
		self.filterCmd = &value
	case "SUBST_VARS":
		if len(self.vars) > 0 && op != "+=" {
			line.logWarning("All but the first SUBST_VARS line should use the \"+=\" operator.")
		}
		self.vars = append(self.vars, value)
	default:
		line.logWarning("Foreign variable in SUBST block.")
	}
}
func (self *SubstContext) finish(line *Line) {
	if self.id == nil || !GlobalVars.optWarnExtra {
		return
	}
	if self.class == nil {
		line.logWarning("Incomplete SUBST block: SUBST_CLASSES missing.")
	}
	if self.stage == nil {
		line.logWarning("Incomplete SUBST block: SUBST_STAGE missing.")
	}
	if len(self.files) == 0 {
		line.logWarning("Incomplete SUBST block: SUBST_FILES missing.")
	}
	if len(self.sed) == 0 && len(self.vars) == 0 && self.filterCmd == nil {
		line.logWarning("Incomplete SUBST block: SUBST_SED, SUBST_VARS or SUBST_FILTER_CMD missing.")
	}
	self.id = nil
	self.class = nil
	self.stage = nil
	self.message = nil
	self.files = self.files[:0]
	self.sed = self.sed[:0]
	self.vars = self.vars[:0]
	self.filterCmd = nil
}

type CvsEntry struct {
	fname    string
	revision string
	mtime    string
	tag      string
}

// A change entry from doc/CHANGES-*
type Change struct {
	line    *Line
	action  string
	pkgpath string
	version string
	author  string
	date    string
}

func match(re string, s string) []string {
	return regexp.MustCompile(re).FindStringSubmatch(s)
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
	reVarname    = `(?:[-*+.0-9A-Z_a-z{}\[]+|\$\{[\w_]+\})+`
	rePkgbase    = `(?:[+.0-9A-Z_a-z]|-[A-Z_a-z])+`
	rePkgversion = `\d(?:\w|\.\d)*`
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

func help(out io.Writer, _ int, _ int) {
	panic("not implemented")
}

// Context of the package that is currently checked.
type PkgContext struct {
	pkgpath                *string // The relative path to the package within PKGSRC
	pkgdir                 *string // PKGDIR from the package Makefile
	filesdir               *string // FILESDIR from the package Makefile
	patchdir               *string // PATCHDIR from the package Makefile
	distinfo_file          *string // DISTINFO_FILE from the package Makefile
	effective_pkgname      *string // PKGNAME or DISTNAME from the package Makefile
	effective_pkgbase      *string // The effective PKGNAME without the version
	effective_pkgversion   *string // The version part of the effective PKGNAME
	effective_pkgname_line *string // The origin of the three effective_* values
	seen_bsd_prefs_mk      bool    // Has bsd.prefs.mk already been included?

	vardef             map[string]*Line // varname => line
	varuse             map[string]*Line // varname => line
	bl3                map[string]*Line // buildlink3.mk name => line; contains only buildlink3.mk files that are directly included.
	plistSubstCond     map[string]bool  // varname => true; list of all variables that are used as conditionals (@comment or nothing) in PLISTs.
	included           map[string]*Line // fname => line
	seenMakefileCommon bool             // Does the package have any .includes?
}

// Context of the Makefile that is currently checked.
type MkContext struct {
	forVars     map[string]bool  // The variables currently used in .for loops
	indentation []int            // Indentation depth of preprocessing directives
	target      string           // Current make(1) target
	vardef      map[string]*Line // varname => line; for all variables that are defined in the current file
	varuse      map[string]*Line // varname => line; for all variables that are used in the current file
	buildDefs   map[string]bool  // Variables that are registered in BUILD_DEFS, to assure that all user-defined variables are added to it.
	plistVars   map[string]bool  // Variables that are registered in PLIST_VARS, to assure that all user-defined variables are added to it.
	tools       map[string]bool  // Set of tools that are declared to be used.
}

func main() {
	GlobalVars.opts = ParseCommandLine(os.Args)
	if GlobalVars.opts.optPrintHelp {
		help(os.Stdout, 0, 1)
	}
	if GlobalVars.opts.optPrintVersion {
		fmt.Printf("%s\n", confVersion)
		os.Exit(0)
	}
	fmt.Printf("%#v\n", GlobalVars.opts)
}
