package main

import (
	"fmt"
	"io/ioutil"
	"os"
	"path"
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
	if m := regexp.MustCompile(pattern).FindStringSubmatch(s); m != nil {
		return m
	}
	panic(fmt.Sprintf("mustMatch %#v %#v", pattern, s))
	return nil
}

func isEmptyDir(fname string) bool {
	dirents, err := ioutil.ReadDir(fname)
	if err != nil {
		logFatal(fname, NO_LINES, "Cannot be read.")
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
		if strings.HasPrefix(line.text, "/"+basename+"/") {
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
		logWarning(fname, NO_LINES, "Should no be executable.")
	}
}

func removeVariableReferences(expr string) string {
	replaced := regexp.MustCompile(`\$\{([^{}]*)\}`).ReplaceAllString(expr, "")
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
	mk := GlobalVars.mkContext
	if mk != nil {
		mk.vardef[varname] = line
		mk.vardef[varcanon] = line
	}
	pkg := GlobalVars.pkgContext
	if pkg != nil {
		pkg.vardef[varname] = line
		pkg.vardef[varcanon] = line
	}
}
func isVarDefined(varname string) bool {
	varcanon := varnameCanon(varname)
	mk := GlobalVars.mkContext
	if mk != nil && (mk.vardef[varname] != nil || mk.vardef[varcanon] != nil) {
		return true
	}
	pkg := GlobalVars.pkgContext
	return pkg != nil && (pkg.vardef[varname] != nil || pkg.vardef[varcanon] != nil)
}
func useVar(line *Line, varname string) {
	varcanon := varnameCanon(varname)
	mk := GlobalVars.mkContext
	if mk != nil {
		mk.varuse[varname] = line
		mk.varuse[varcanon] = line
	}
	pkg := GlobalVars.pkgContext
	if pkg != nil {
		pkg.varuse[varname] = line
		pkg.varuse[varcanon] = line
	}
}

func isVarUsed(varname string) bool {
	varcanon := varnameCanon(varname)
	mk := GlobalVars.mkContext
	if mk != nil && (mk.varuse[varname] != nil || mk.varuse[varcanon] != nil) {
		return true
	}
	pkg := GlobalVars.pkgContext
	return pkg != nil && (pkg.varuse[varname] != nil || pkg.varuse[varcanon] != nil)
}

func splitOnSpace(s string) []string {
	return regexp.MustCompile(`\s+`).Split(s, -1)
}

func fileExists(fname string) bool {
	st, err := os.Stat(fname)
	return err == nil && st.Mode().IsRegular()
}

func stringset(s string) map[string]bool {
	result := make(map[string]bool)
	for _, m := range regexp.MustCompile(`\S+`).FindAllString(s, -1) {
		result[m] = true
	}
	return result
}

var res = make(map[string]*regexp.Regexp)
func match(s, re string) []string {
	cre := res[re]
	if cre == nil {
		cre = regexp.MustCompile(re)
		res[re] = cre
	}
	return cre.FindStringSubmatch(s)
}

func match3(s, re string) (bool, string, string, string) {
	if m := match(s, re); m != nil {
		return true, m[1], m[2], m[3]
	} else {
		return false, "", "", ""
	}
}

func nilToZero(pi *int) int {
	if pi != nil {
		return *pi
	}
	return 0
}

func toInt(s string) int {
	n, err := strconv.Atoi(s)
	if err != nil {
		logFatalF(NO_FILE, NO_LINES, "Internal error: %q", err)
	}
	return n
}
