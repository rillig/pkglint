package main

import (
	"fmt"
	"os"
	"path"
	"strings"
)

const (
	reMkInclude  = `^\.\s*(s?include)\s+\"([^\"]+)\"\s*(?:#.*)?$`
	rePkgname    = `^([\w\-.+]+)-(\d(?:\w|\.\d)*)$`
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
	if G.opts.DebugTrace {
		defer tracecall1("loadPackageMakefile", fname)()
	}

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

	if G.opts.DebugMisc {
		dummyLine.debug1("DISTINFO_FILE=%s", pkg.distinfoFile)
		dummyLine.debug1("FILESDIR=%s", pkg.filesdir)
		dummyLine.debug1("PATCHDIR=%s", pkg.patchdir)
		dummyLine.debug1("PKGDIR=%s", pkg.pkgdir)
	}

	return mainLines
}

func resolveVariableRefs(text string) string {
	if G.opts.DebugTrace {
		defer tracecall1("resolveVariableRefs", text)()
	}

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
			return "${" + varname + "}"
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
	if G.opts.DebugMisc {
		mkline.debug2("Expanded %q to %q", varname, value)
	}
	return value
}

func checkfileExtra(fname string) {
	if G.opts.DebugTrace {
		defer tracecall1("checkfileExtra", fname)()
	}

	if lines := LoadNonemptyLines(fname, false); lines != nil {
		checklinesTrailingEmptyLines(lines)
	}
}

func checklinesDescr(lines []*Line) {
	if G.opts.DebugTrace {
		defer tracecall1("checklinesDescr", lines[0].fname)()
	}

	for _, line := range lines {
		line.checkLength(80)
		line.checkTrailingWhitespace()
		line.checkValidCharacters(`[\t -~]`)
		if strings.Contains(line.text, "${") {
			line.note0("Variables are not expanded in the DESCR file.")
		}
	}
	checklinesTrailingEmptyLines(lines)

	if maxlines := 24; len(lines) > maxlines {
		line := lines[maxlines]

		line.warnf("File too long (should be no more than %d lines).", maxlines)
		explain3(
			"A common terminal size is 80x25 characters. The DESCR file should",
			"fit on one screen. It is also intended to give a _brief_ summary",
			"about the package's contents.")
	}

	saveAutofixChanges(lines)
}

func checklinesMessage(lines []*Line) {
	if G.opts.DebugTrace {
		defer tracecall1("checklinesMessage", lines[0].fname)()
	}

	explainMessage := func() {
		explain4(
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
		line.checkLength(80)
		line.checkTrailingWhitespace()
		line.checkValidCharacters(`[\t -~]`)
	}
	if lastLine := lines[len(lines)-1]; lastLine.text != hline {
		lastLine.warn0("Expected a line of exactly 75 \"=\" characters.")
		explainMessage()
	}
	checklinesTrailingEmptyLines(lines)
}

func checkfileMk(fname string) {
	if G.opts.DebugTrace {
		defer tracecall1("checkfileMk", fname)()
	}

	lines := LoadNonemptyLines(fname, true)
	if lines == nil {
		return
	}

	NewMkLines(lines).check()
	saveAutofixChanges(lines)
}

func checkfile(fname string) {
	if G.opts.DebugTrace {
		defer tracecall1("checkfile", fname)()
	}

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
		explain4(
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
		if G.opts.DebugUnchecked {
			debugf(fname, noLines, "Unchecked file %q.", fname)
		}

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
		lines[last].note0("Trailing empty lines.")
	}
}

func matchVarassign(text string) (m bool, varname, op, value, comment string) {
	i, n := 0, len(text)

	for i < n && text[i] == ' ' {
		i++
	}

	varnameStart := i
	for ; i < n; i++ {
		b := text[i]
		mask := uint(0)
		switch b & 0xE0 {
		case 0x20:
			mask = 0x03ff6c10
		case 0x40:
			mask = 0x8ffffffe
		case 0x60:
			mask = 0x2ffffffe
		}
		if (mask>>(b&0x1F))&1 == 0 {
			break
		}
	}
	varnameEnd := i

	if varnameEnd == varnameStart {
		return
	}

	for i < n && (text[i] == ' ' || text[i] == '\t') {
		i++
	}

	opStart := i
	if i < n {
		if b := text[i]; b&0xE0 == 0x20 && (uint(0x84000802)>>(b&0x1F))&1 != 0 {
			i++
		}
	}
	if i < n && text[i] == '=' {
		i++
	} else {
		return
	}
	opEnd := i

	if text[varnameEnd-1] == '+' && varnameEnd == opStart && text[opStart] == '=' {
		varnameEnd--
		opStart--
	}

	for i < n && (text[i] == ' ' || text[i] == '\t') {
		i++
	}

	valueStart := i
	valuebuf := make([]byte, n-valueStart)
	j := 0
	for ; i < n; i++ {
		b := text[i]
		if b == '#' && (i == valueStart || text[i-1] != '\\') {
			break
		} else if b != '\\' || i+1 >= n || text[i+1] != '#' {
			valuebuf[j] = b
			j++
		}
	}

	commentStart := i
	commentEnd := n

	m = true
	varname = text[varnameStart:varnameEnd]
	op = text[opStart:opEnd]
	value = strings.TrimSpace(string(valuebuf[:j]))
	comment = text[commentStart:commentEnd]
	return
}

type DependencyPattern struct {
	pkgbase  string // "freeciv-client", "{gcc48,gcc48-libs}", "${EMACS_REQD}"
	lowerOp  string // ">=", ">"
	lower    string // "2.5.0", "${PYVER}"
	upperOp  string // "<", "<="
	upper    string // "3.0", "${PYVER}"
	wildcard string // "[0-9]*", "1.5.*", "${PYVER}"
}

func ParsePkgbasePattern(repl *PrefixReplacer) (pkgbase string) {
	for {
		if repl.advanceRegexp(`^\$\{\w+\}`) ||
			repl.advanceRegexp(`^[\w.*+,{}]+`) ||
			repl.advanceRegexp(`^\[[\d-]+\]`) {
			pkgbase += repl.m[0]
			continue
		}

		mark := repl.Mark()
		if repl.advanceStr("-") {
			if repl.advanceRegexp(`^\d`) ||
				repl.advanceRegexp(`^\$\{\w*VER\w*\}`) ||
				repl.advanceStr("[") {
				repl.Reset(mark)
				return
			}
			pkgbase += "-"
		} else {
			return
		}
	}
}

func ParseDependency(repl *PrefixReplacer) *DependencyPattern {
	var dp DependencyPattern
	mark := repl.Mark()
	dp.pkgbase = ParsePkgbasePattern(repl)
	if dp.pkgbase == "" {
		return nil
	}

	mark2 := repl.Mark()
	if repl.advanceStr(">=") || repl.advanceStr(">") {
		op := repl.s
		if repl.advanceRegexp(`^(?:\$\{\w+\}|\d[\w.]*)`) {
			dp.lowerOp = op
			dp.lower = repl.m[0]
		} else {
			repl.Reset(mark2)
		}
	}
	if repl.advanceStr("<=") || repl.advanceStr("<") {
		op := repl.s
		if repl.advanceRegexp(`^(?:\$\{\w+\}|\d[\w.]*)`) {
			dp.upperOp = op
			dp.upper = repl.m[0]
		} else {
			repl.Reset(mark2)
		}
	}
	if dp.lowerOp != "" || dp.upperOp != "" {
		return &dp
	}
	if repl.advanceStr("-") && repl.rest != "" {
		dp.wildcard = repl.AdvanceRest()
		return &dp
	}
	if hasPrefix(dp.pkgbase, "${") && hasSuffix(dp.pkgbase, "}") {
		return &dp
	}
	if hasSuffix(dp.pkgbase, "-*") {
		dp.pkgbase = strings.TrimSuffix(dp.pkgbase, "-*")
		dp.wildcard = "*"
		return &dp
	}

	repl.Reset(mark)
	return nil
}
