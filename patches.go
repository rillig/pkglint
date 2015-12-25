package main

// Checks for patch files.

import (
	"path"
	"strings"
)

func checklinesPatch(lines []*Line) {
	defer tracecall("checklinesPatch", lines[0].fname)()

	(&PatchChecker{lines, NewExpecter(lines), false, false}).check()
}

type PatchChecker struct {
	lines             []*Line
	exp               *Expecter
	seenDocumentation bool
	previousLineEmpty bool
}

const (
	rePatchUniFileDel = `^---\s(\S+)(?:\s+(.*))?$`
	rePatchUniFileAdd = `^\+\+\+\s(\S+)(?:\s+(.*))?$`
	rePatchUniHunk    = `^@@ -(\d+)(?:,(\d+))? \+(\d+)(?:,(\d+))? @@(.*)$`
)

func (ck *PatchChecker) check() {
	defer tracecall("PatchChecker.check")()

	if checklineRcsid(ck.lines[0], ``, "") {
		ck.exp.advance()
	}
	ck.previousLineEmpty = ck.exp.expectEmptyLine()

	patchedFiles := 0
	for !ck.exp.eof() {
		line := ck.exp.currentLine()
		if ck.exp.advanceIfMatches(rePatchUniFileDel) {
			if ck.exp.advanceIfMatches(rePatchUniFileAdd) {
				ck.checkBeginDiff(line, patchedFiles)
				ck.checkUnifiedDiff(ck.exp.m[1])
				patchedFiles++
				continue
			}

			ck.exp.stepBack()
		}

		if ck.exp.advanceIfMatches(rePatchUniFileAdd) {
			patchedFile := ck.exp.m[1]
			if ck.exp.advanceIfMatches(rePatchUniFileDel) {
				ck.checkBeginDiff(line, patchedFiles)
				ck.exp.previousLine().warn0("Unified diff headers should be first ---, then +++.")
				ck.checkUnifiedDiff(patchedFile)
				patchedFiles++
				continue
			}

			ck.exp.stepBack()
		}

		if ck.exp.advanceIfMatches(`^\*\*\*\s(\S+)(.*)$`) {
			if ck.exp.advanceIfMatches(`^---\s(\S+)(.*)$`) {
				ck.checkBeginDiff(line, patchedFiles)
				line.warn0("Please use unified diffs (diff -u) for patches.")
				return
			}

			ck.exp.stepBack()
		}

		ck.exp.advance()
		ck.previousLineEmpty = line.text == "" || hasPrefix(line.text, "diff ") || hasPrefix(line.text, "=============")
		if !ck.previousLineEmpty {
			ck.seenDocumentation = true
		}
	}

	if patchedFiles > 1 {
		warnf(ck.lines[0].fname, noLines, "Contains patches for %d files, should be only one.", patchedFiles)
	} else if patchedFiles == 0 {
		errorf(ck.lines[0].fname, noLines, "Contains no patch.")
	}

	checklinesTrailingEmptyLines(ck.lines)
	saveAutofixChanges(ck.lines)
}

// See http://www.gnu.org/software/diffutils/manual/html_node/Detailed-Unified.html
func (ck *PatchChecker) checkUnifiedDiff(patchedFile string) {
	defer tracecall("PatchChecker.checkUnifiedDiff")()

	patchedFileType := guessFileType(ck.exp.currentLine(), patchedFile)
	if G.opts.DebugMisc {
		ck.exp.currentLine().debugf("guessFileType(%q) = %s", patchedFile, patchedFileType)
	}

	hasHunks := false
	for ck.exp.advanceIfMatches(rePatchUniHunk) {
		hasHunks = true
		linesToDel := toInt(ck.exp.m[2], 1)
		linesToAdd := toInt(ck.exp.m[4], 1)
		if G.opts.DebugMisc {
			ck.exp.previousLine().debugf("hunk -%d +%d", linesToDel, linesToAdd)
		}
		ck.checktextUniHunkCr()

		for linesToDel > 0 || linesToAdd > 0 || hasPrefix(ck.exp.currentLine().text, "\\") {
			line := ck.exp.currentLine()
			ck.exp.advance()
			text := line.text
			switch {
			case text == "":
				linesToDel--
				linesToAdd--
			case hasPrefix(text, " "), hasPrefix(text, "\t"):
				linesToDel--
				linesToAdd--
				ck.checklineContext(text[1:], patchedFileType)
			case hasPrefix(text, "-"):
				linesToDel--
			case hasPrefix(text, "+"):
				linesToAdd--
				ck.checklineAdded(text[1:], patchedFileType)
			case hasPrefix(text, "\\"):
				// \ No newline at end of file
			default:
				line.error0("Invalid line in unified patch hunk")
				return
			}
		}
	}
	if !hasHunks {
		ck.exp.currentLine().error1("No patch hunks for %q.", patchedFile)
	}
	if !ck.exp.eof() {
		line := ck.exp.currentLine()
		if line.text != "" && !matches(line.text, rePatchUniFileDel) && !hasPrefix(line.text, "Index:") && !hasPrefix(line.text, "diff ") {
			line.warn0("Empty line or end of file expected.")
			explain3(
				"This empty line makes the end of the patch clearly visible.",
				"Otherwise the reader would have to count lines to see where",
				"the patch ends.")
		}
	}
}

func (ck *PatchChecker) checkBeginDiff(line *Line, patchedFiles int) {
	defer tracecall("PatchChecker.checkBeginDiff")()

	if !ck.seenDocumentation && patchedFiles == 0 {
		line.error0("Each patch must be documented.")
		explain(
			"Each patch must document why it is necessary. If it has been applied",
			"because of a security issue, a reference to the CVE should be mentioned",
			"as well.",
			"",
			"Since it is our goal to have as few patches as possible, all patches",
			"should be sent to the upstream maintainers of the package. After you",
			"have done so, you should add a reference to the bug report containing",
			"the patch.")
	}
	if G.opts.WarnSpace && !ck.previousLineEmpty {
		if !line.autofixInsertBefore("") {
			line.note0("Empty line expected.")
		}
	}
}

func (ck *PatchChecker) checklineContext(text string, patchedFileType FileType) {
	defer tracecall("PatchChecker.checklineContext", text, patchedFileType)()

	if G.opts.WarnExtra {
		ck.checklineAdded(text, patchedFileType)
	} else {
		ck.checktextRcsid(text)
	}
}

func (ck *PatchChecker) checklineAdded(addedText string, patchedFileType FileType) {
	defer tracecall("PatchChecker.checklineAdded", addedText, patchedFileType)()

	ck.checktextRcsid(addedText)

	line := ck.exp.previousLine()
	switch patchedFileType {
	case ftShell:
	case ftMakefile:
		// This check is not as accurate as the similar one in MkLine.checkShelltext.
		shellwords, _ := splitIntoShellwords(line, addedText)
		for _, shellword := range shellwords {
			if !hasPrefix(shellword, "#") {
				line.checkAbsolutePathname(shellword)
			}
		}
	case ftSource:
		checklineSourceAbsolutePathname(line, addedText)
	case ftConfigure:
		if hasPrefix(addedText, ": Avoid regenerating within pkgsrc") {
			line.error0("This code must not be included in patches.")
			explain(
				"It is generated automatically by pkgsrc after the patch phase.",
				"",
				"For more details, look for \"configure-scripts-override\" in",
				"mk/configure/gnu-configure.mk.")
		}
	case ftIgnore:
		break
	default:
		checklineOtherAbsolutePathname(line, addedText)
	}
}

func (ck *PatchChecker) checktextUniHunkCr() {
	defer tracecall("PatchChecker.checktextUniHunkCr")()

	line := ck.exp.previousLine()
	if hasSuffix(line.text, "\r") {
		if !line.autofixReplace("\r\n", "\n") {
			line.error0("The hunk header must not end with a CR character.")
			explain1(
				"The MacOS X patch utility cannot handle these.")
		}
	}
}

func (ck *PatchChecker) checktextRcsid(text string) {
	if strings.IndexByte(text, '$') == -1 {
		return
	}
	if m, tagname := match1(text, `\$(Author|Date|Header|Id|Locker|Log|Name|RCSfile|Revision|Source|State|NetBSD)(?::[^\$]*)?\$`); m {
		if matches(text, rePatchUniHunk) {
			ck.exp.previousLine().warn1("Found RCS tag \"$%s$\". Please remove it.", tagname)
		} else {
			ck.exp.previousLine().warn1("Found RCS tag \"$%s$\". Please remove it by reducing the number of context lines using pkgdiff or \"diff -U[210]\".", tagname)
		}
	}
}

type FileType uint8

const (
	ftSource FileType = iota
	ftShell
	ftMakefile
	ftText
	ftConfigure
	ftIgnore
	ftUnknown
)

func (ft FileType) String() string {
	return [...]string{
		"source code",
		"shell code",
		"Makefile",
		"text file",
		"configure file",
		"ignored",
		"unknown",
	}[ft]
}

// This is used to select the proper subroutine for detecting absolute pathnames.
func guessFileType(line *Line, fname string) FileType {
	basename := path.Base(fname)
	basename = strings.TrimSuffix(basename, ".in") // doesn’t influence the content type
	ext := strings.ToLower(strings.TrimLeft(path.Ext(basename), "."))

	switch {
	case matches(basename, `^I?[Mm]akefile|\.ma?k$`):
		return ftMakefile
	case basename == "configure" || basename == "configure.ac":
		return ftConfigure
	}

	switch ext {
	case "m4", "sh":
		return ftShell
	case "c", "cc", "cpp", "cxx", "el", "h", "hh", "hpp", "l", "pl", "pm", "py", "s", "t", "y":
		return ftSource
	case "conf", "html", "info", "man", "po", "tex", "texi", "texinfo", "txt", "xml":
		return ftText
	case "":
		return ftUnknown
	}

	if G.opts.DebugMisc {
		line.debug1("Unknown file type for %q", fname)
	}
	return ftUnknown
}

func checkwordAbsolutePathname(line *Line, word string) {
	defer tracecall("checkwordAbsolutePathname", word)()

	switch {
	case matches(word, `^/dev/(?:null|tty|zero)$`):
		// These are defined by POSIX.
	case word == "/bin/sh":
		// This is usually correct, although on Solaris, it's pretty feature-crippled.
	case matches(word, `^/(?:[a-z]|\$[({])`):
		// Absolute paths probably start with a lowercase letter.
		line.warn1("Found absolute pathname: %s", word)
		explain(
			"Absolute pathnames are often an indicator for unportable code. As",
			"pkgsrc aims to be a portable system, absolute pathnames should be",
			"avoided whenever possible.",
			"",
			"A special variable in this context is ${DESTDIR}, which is used in GNU",
			"projects to specify a different directory for installation than what",
			"the programs see later when they are executed. Usually it is empty, so",
			"if anything after that variable starts with a slash, it is considered",
			"an absolute pathname.")
	}
}

// Looks for strings like "/dev/cd0" appearing in source code
func checklineSourceAbsolutePathname(line *Line, text string) {
	if !strings.ContainsAny(text, "\"'") {
		return
	}
	if matched, before, _, str := match3(text, `^(.*)(["'])(/\w[^"']*)["']`); matched {
		if G.opts.DebugMisc {
			line.debug2("checklineSourceAbsolutePathname: before=%q, str=%q", before, str)
		}

		switch {
		case matches(before, `[A-Z_]\s*$`):
			// ok; C example: const char *echo_cmd = PREFIX "/bin/echo";

		case matches(before, `\+\s*$`):
			// ok; Python example: libdir = prefix + '/lib'

		default:
			checkwordAbsolutePathname(line, str)
		}
	}
}

func checklineOtherAbsolutePathname(line *Line, text string) {
	defer tracecall("checklineOtherAbsolutePathname", text)()

	if hasPrefix(text, "#") && !hasPrefix(text, "#!") {
		// Don't warn for absolute pathnames in comments, except for shell interpreters.

	} else if m, before, path, _ := match3(text, `^(.*?)((?:/[\w.]+)*/(?:bin|dev|etc|home|lib|mnt|opt|proc|sbin|tmp|usr|var)\b[\w./\-]*)(.*)$`); m {
		switch {
		case hasSuffix(before, "@"): // Example: @PREFIX@/bin
		case matches(before, `[)}]$`): // Example: ${prefix}/bin
		case matches(before, `\+\s*["']$`): // Example: prefix + '/lib'
		case matches(before, `\w$`): // Example: libdir=$prefix/lib
		case hasSuffix(before, "."): // Example: ../dir
		// XXX new: case matches(before, `s.$`): // Example: sed -e s,/usr,@PREFIX@,
		default:
			if G.opts.DebugMisc {
				line.debug1("before=%q", before)
			}
			checkwordAbsolutePathname(line, path)
		}
	}
}
