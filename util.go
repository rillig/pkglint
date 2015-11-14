package main

import (
	"io/ioutil"
	"os"
	"path"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
)

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

func mustMatch(pattern string, s string) []string {
	if m := reCompile(pattern).FindStringSubmatch(s); m != nil {
		return m
	}
	panic(sprintf("mustMatch %#v %#v", pattern, s))
}

func isEmptyDir(fname string) bool {
	dirents, err := ioutil.ReadDir(fname)
	if err != nil {
		return true
	}
	for _, dirent := range dirents {
		name := dirent.Name()
		if name == "." || name == ".." || name == "CVS" {
			continue
		}
		if dirent.IsDir() && isEmptyDir(fname+"/"+name) {
			continue
		}
		return false
	}
	return true
}

func getSubdirs(fname string) []string {
	dirents, err := ioutil.ReadDir(fname)
	if err != nil {
		logFatal(fname, NO_LINES, "Cannot be read.")
	}

	subdirs := make([]string, 0)
	for _, dirent := range dirents {
		name := dirent.Name()
		if name != "." && name != ".." && name != "CVS" && dirent.IsDir() && !isEmptyDir(fname+"/"+name) {
			subdirs = append(subdirs, name)
		}
	}
	return subdirs
}

// Checks whether a file is already committed to the CVS repository.
func isCommitted(fname string) bool {
	basename := path.Base(fname)
	lines, err := loadLines(path.Dir(fname)+"/CVS/Entries", false)
	if err != nil {
		return false
	}
	for _, line := range lines {
		if hasPrefix(line.text, "/"+basename+"/") {
			return true
		}
	}
	return false
}

// No package file should ever be executable. Even the INSTALL and
// DEINSTALL scripts are usually not usable in the form they have in the
// package, as the pathnames get adjusted during installation. So there is
// no need to have any file executable.
func checkPermissions(fname string) {
	st, err := os.Stat(fname)
	if err != nil && st.Mode().IsRegular() && st.Mode()&0111 != 0 {
		logWarning(fname, NO_LINES, "Should not be executable.")
	}
}

func removeVariableReferences(expr string) string {
	replaced := reCompile(`\$\{([^{}]*)\}`).ReplaceAllString(expr, "")
	if replaced != expr {
		return removeVariableReferences(replaced)
	}
	return replaced
}

// Returns the number of columns that a string occupies when printed with
// a tabulator size of 8.
func tabLength(s string) int {
	length := 0
	for _, r := range s {
		if r == '\t' {
			length = (length + 7) % 8
		} else {
			length++
		}
	}
	return length
}

func varnameBase(varname string) string {
	return strings.Split(varname, ".")[0]
}
func varnameCanon(varname string) string {
	parts := strings.SplitN(varname, ".", 2)
	if len(parts) == 2 {
		return parts[0] + ".*"
	}
	return parts[0]
}
func varnameParam(varname string) string {
	parts := strings.SplitN(varname, ".", 2)
	return parts[len(parts)-1]
}

func defineVar(line *Line, varname string) {
	varcanon := varnameCanon(varname)
	mk := G.mkContext
	if mk != nil {
		mk.vardef[varname] = line
		mk.vardef[varcanon] = line
	}
	pkg := G.pkgContext
	if pkg != nil {
		pkg.vardef[varname] = line
		pkg.vardef[varcanon] = line
	}
}
func varIsDefined(varname string) bool {
	varcanon := varnameCanon(varname)
	mk := G.mkContext
	if mk != nil && (mk.vardef[varname] != nil || mk.vardef[varcanon] != nil) {
		return true
	}
	pkg := G.pkgContext
	return pkg != nil && (pkg.vardef[varname] != nil || pkg.vardef[varcanon] != nil)
}
func useVar(line *Line, varname string) {
	varcanon := varnameCanon(varname)
	mk := G.mkContext
	if mk != nil {
		mk.varuse[varname] = line
		mk.varuse[varcanon] = line
	}
	pkg := G.pkgContext
	if pkg != nil {
		pkg.varuse[varname] = line
		pkg.varuse[varcanon] = line
	}
}

func varIsUsed(varname string) bool {
	varcanon := varnameCanon(varname)
	if mk := G.mkContext; mk != nil && (mk.varuse[varname] != nil || mk.varuse[varcanon] != nil) {
		return true
	}
	if pkg := G.pkgContext; pkg != nil && (pkg.varuse[varname] != nil || pkg.varuse[varcanon] != nil) {
		return true
	}
	return false
}

func splitOnSpace(s string) []string {
	return reCompile(`\s+`).Split(s, -1)
}

func fileExists(fname string) bool {
	st, err := os.Stat(fname)
	return err == nil && st.Mode().IsRegular()
}

func dirExists(fname string) bool {
	st, err := os.Stat(fname)
	return err == nil && st.Mode().IsDir()
}

func stringset(s string) map[string]bool {
	result := make(map[string]bool)
	for _, m := range reCompile(`\S+`).FindAllString(s, -1) {
		result[m] = true
	}
	return result
}

var res = make(map[string]*regexp.Regexp)

func reCompile(re string) *regexp.Regexp {
	cre := res[re]
	if cre == nil {
		cre = regexp.MustCompile(re)
		res[re] = cre
	}
	return cre
}

func match(s, re string) []string {
	return reCompile(re).FindStringSubmatch(s)
}
func matchn(s, re string, n int) []string {
	if m := match(s, re); m != nil {
		if len(m) != 1+n {
			panic(sprintf("expected match%d, got match%d for %q", len(m)-1, n, re))
		}
		return m
	}
	return nil
}

func match0(s, re string) bool {
	return matchn(s, re, 0) != nil
}
func match1(s, re string) (bool, string) {
	if m := matchn(s, re, 1); m != nil {
		return true, m[1]
	} else {
		return false, ""
	}
}
func match2(s, re string) (bool, string, string) {
	if m := matchn(s, re, 2); m != nil {
		return true, m[1], m[2]
	} else {
		return false, "", ""
	}
}
func match3(s, re string) (bool, string, string, string) {
	if m := matchn(s, re, 3); m != nil {
		return true, m[1], m[2], m[3]
	} else {
		return false, "", "", ""
	}
}
func match4(s, re string) (bool, string, string, string, string) {
	if m := matchn(s, re, 4); m != nil {
		return true, m[1], m[2], m[3], m[4]
	} else {
		return false, "", "", "", ""
	}
}
func match5(s, re string) (bool, string, string, string, string, string) {
	if m := matchn(s, re, 5); m != nil {
		return true, m[1], m[2], m[3], m[4], m[5]
	} else {
		return false, "", "", "", "", ""
	}
}

func replaceFirst(s, re, replacement string) ([]string, string) {
	defer tracecall("replaceFirst", s, re, replacement)()

	if m := reCompile(re).FindStringSubmatchIndex(s); m != nil {
		replaced := s[:m[0]] + replacement + s[m[1]:]
		mm := make([]string, len(m)/2)
		for i := 0; i < len(m); i += 2 {
			mm[i/2] = s[negToZero(m[i]):negToZero(m[i+1])]
		}
		return mm, replaced
	}
	return nil, s
}
func replacestart(ps *string, pm *[]string, re string) bool {
	if m := reCompile(re).FindStringSubmatch(*ps); m != nil {
		*ps = (*ps)[len(m[0]):]
		*pm = m
		return true
	}
	return false
}

func nilToZero(pi *int) int {
	if pi != nil {
		return *pi
	}
	return 0
}

// Useful in combination with regex.Find*Index
func negToZero(i int) int {
	if i >= 0 {
		return i
	}
	return 0
}

func toInt(s string) int {
	n, err := strconv.Atoi(s)
	if err != nil {
		panic("toInt " + s)
	}
	return n
}

func newInt(i int) *int {
	return &i
}

func newStr(s string) *string {
	return &s
}

func dirglob(dirname string) []string {
	fis, err := ioutil.ReadDir(dirname)
	if err != nil {
		return make([]string, 0)
	}
	fnames := make([]string, len(fis))
	for i, fi := range fis {
		fnames[i] = dirname + "/" + fi.Name()
	}
	return fnames
}

func argsStr(args ...interface{}) string {
	argsStr := ""
	for i, arg := range args {
		if i != 0 {
			argsStr += ", "
		}
		argsStr += sprintf("%#v", arg)
	}
	return argsStr
}

func trace(action, funcname string, args ...interface{}) {
	if G.opts.optDebugTrace {
		logDebug(NO_FILE, NO_LINES, "%s%s%s(%s)", strings.Repeat("| ", G.traceDepth), action, funcname, argsStr(args...))
	}
}
func tracecall(funcname string, args ...interface{}) func() {
	if G.opts.optDebugTrace {
		trace("+ ", funcname, args...)
		G.traceDepth++
		return func() {
			G.traceDepth--
			trace("- ", funcname, args...)
		}
	} else {
		return func() {}
	}
}

// Emulates make(1)’s :S substitution operator.
func mkopSubst(s string, left bool, from string, right bool, to string, all bool) string {
	re := ifelseStr(left, "^", "") + regexp.QuoteMeta(from) + ifelseStr(right, "$", "")
	done := false
	return reCompile(re).ReplaceAllStringFunc(s, func(match string) string {
		if all || !done {
			done = !all
			return to
		}
		return match
	})
}

func relpath(from, to string) string {
	absFrom, err1 := filepath.Abs(from)
	absTo, err2 := filepath.Abs(to)
	rel, err3 := filepath.Rel(absFrom, absTo)
	if err1 != nil || err2 != nil || err3 != nil {
		panic("relpath" + argsStr(from, to, err1, err2, err3))
	}
	return filepath.ToSlash(rel)
}
