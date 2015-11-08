package main

// Checks for patch files

import (
	"path"
	"regexp"
	"strings"
)

type FileType int

const (
	FT_SOURCE FileType = iota
	FT_SHELL
	FT_MAKE
	FT_TEXT
	FT_CONFIGURE
	FT_IGNORE
	FT_UNKNOWN
)

// This is used to select the proper subroutine for detecting absolute pathnames.
func guessFileType(line *Line, fname string) FileType {
	basename := path.Base(fname)
	basename = strings.TrimSuffix(basename, ".in") // doesn’t influence the content type

	if match(basename, `^I?[Mm]akefile(\..*)?|\.ma?k$`) != nil {
		return FT_MAKE
	}
	if basename == "configure" || basename == "configure.ac" {
		return FT_CONFIGURE
	}
	if match(basename, `\.(?:m4|sh)$`) != nil {
		return FT_SHELL
	}
	if match(basename, `\.(?:cc?|cpp|cxx|el|hh?|hpp|l|pl|pm|py|s|t|y)$`) != nil {
		return FT_SOURCE
	}
	if match(basename, `.+\.(?:\d+|conf|html|info|man|po|tex|texi|texinfo|txt|xml)$`) != nil {
		return FT_TEXT
	}
	if !contains(basename, ".") {
		return FT_UNKNOWN
	}

	_ = G.opts.optDebugMisc && line.logDebug("Unknown file type.")
	return FT_UNKNOWN
}

var goodCppMacros = stringset(`
		__STDC__

		__GNUC__ __GNUC_MINOR__
		__SUNPRO_C

		__i386
		__mips
		__sparc

		__APPLE__
		__bsdi__
		__CYGWIN__
		__DragonFly__
		__FreeBSD__ __FreeBSD_version
		__INTERIX
		__linux__
		__MINGW32__
		__NetBSD__ __NetBSD_Version__
		__OpenBSD__
		__SVR4
		__sgi
		__sun

		__GLIBC__
`)
var badCppMacros = map[string]string{
	"__sgi__":      "__sgi",
	"__sparc__":    "__sparc",
	"__sparc_v9__": "__sparcv9",
	"__sun__":      "__sun",
	"__svr4__":     "__SVR4",
}

func checklineCppMacroNames(line *Line, text string) {
	for _, m := range regexp.MustCompile(`defined\((__[\w_]+)\)|\b(_\w+)\(`).FindAllStringSubmatch(text, -1) {
		macro := m[1] + m[2]

		if goodCppMacros[macro] {
			// nice
		} else if better := badCppMacros[macro]; better != "" {
			line.logWarning("The macro %q is not portable enough. Please use %q instead", macro, better)
			line.explainWarning("See the pkgsrc guide, section \"CPP defines\" for details.")
		} else if match(macro, `(?i)^_+NetBSD_+Version_+$`) != nil && macro != "__NetBSD_Version__" {
			line.logWarning("Misspelled variant %q of %q.", macro, "__NetBSD_Version__")
		}
	}
}

func checkwordAbsolutePathname(line *Line, word string) {
	line.trace("checkwordAbsolutePathname", word)

	switch {
	case match(word, `^/dev/(?:null|tty|zero)$`) != nil:
		// These are defined by POSIX.
	case word == "/bin/sh":
		// This is usually correct, although on Solaris, it's pretty feature-crippled.
	case match(word, `/(?:[a-z]|\$[({])`) == nil:
		// Assume that all pathnames start with a lowercase letter.
	default:
		line.logWarning("Found absolute pathname: %s", word)
		line.explainWarning(
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
	if matched, before, _, str := match3(text, `(.*)([\"'])(/\w[^\"']*)\2`); matched {
		_ = G.opts.optDebugMisc && line.logDebug("checklineSourceAbsolutePathname: before=%q, str=%q", before, str)

		if match(before, `[A-Z_]+\s*$`) != nil {
			// ok; C example: const char *echo_cmd = PREFIX "/bin/echo";
		} else if match(before, `\+\s*$`) != nil {
			// ok; Python example: libdir = prefix + '/lib'
		} else {
			checkwordAbsolutePathname(line, str)
		}
	}
}

func checklineOtherAbsolutePathname(line *Line, text string) {
	line.trace("checklineOtherAbsolutePathname", text)

	if match(text, `^#[^!]`) != nil {
		// Don't warn for absolute pathnames in comments, except for shell interpreters.

	} else if m, before, path, _ := match3(text, `^(.*?)((?:/[\w.]+)*/(?:bin|dev|etc|home|lib|mnt|opt|proc|sbin|tmp|usr|var)\b[\w./\-]*)(.*)$`); m {
		if hasSuffix(before, "@") {
			// ok; autotools example: @PREFIX@/bin

		} else if match(before, `[)}]`) != nil {
			// ok; autotools example: ${prefix}/bin

		} else if match(before, `\+\s*["']$`) != nil {
			// ok; Python example: prefix + '/lib'

		} else if match(before, `\w$`) != nil {
			// ok; shell example: libdir=$prefix/lib

		} else {
			_ = G.opts.optDebugMisc && line.logDebug("before=%q", before)
			checkwordAbsolutePathname(line, path)
		}
	}
}

func checkfilePatch(fname string) {
	trace("checkfilePatch", fname)

	checkperms(fname)
	lines, err := loadLines(fname, false)
	if err != nil {
		logError(fname, NO_LINES, "Cannot be read.")
		return
	}
	if len(lines) == 0 {
		logError(fname, NO_LINES, "Must not be empty.")
		return
	}

	checklinesPatch(lines)
}

const (
	rePatchRcsid            = `^\$.*\$$`
	rePatchText             = `^(.+)$`
	rePatchEmpty            = `^$`
	rePatchCtxFileDel       = `^\*\*\*\s(\S+)(.*)$`
	rePatchCtxFileAdd       = `^---\s(\S+)(.*)$`
	rePatchCtxHunk          = `^\*{15}(.*)$`
	rePatchCtxHunkDel       = `^\*\*\*\s(\d+)(?:,(\d+))?\s\*\*\*\*$`
	rePatchCtxHunkAdd       = `^-{3}\s(\d+)(?:,(\d+))?\s----$`
	rePatchCtxLineDel       = `^(?:-\s(.*))?$`
	rePatchCtxLineMod       = `^(?:!\s(.*))?$`
	rePatchCtxLineAdd       = `^(?:\+\s(.*))?$`
	rePatchCtxLineContext   = `^(?:\s\s(.*))?$`
	rePatchUniFileDel       = `^---\s(\S+)(?:\s+(.*))?$`
	rePatchUniFileAdd       = `^\+\+\+\s(\S+)(?:\s+(.*))?$`
	rePatchUniHunk          = `^\@\@\s-(?:(\d+),)?(\d+)\s\+(?:(\d+),)?(\d+)\s\@\@(.*)$`
	rePatchUniLineDel       = `^-(.*)$`
	rePatchUniLineAdd       = `^\+(.*)$`
	rePatchUniLineContext   = `^\s(.*)$`
	rePatchUniLineNoNewline = `^\\ No newline at end of file$`
)

type State int

const (
	PST_START State = iota
	PST_CENTER
	PST_TEXT
	PST_CTX_FILE_ADD
	PST_CTX_HUNK
	PST_CTX_HUNK_DEL
	PST_CTX_LINE_DEL0
	PST_CTX_LINE_DEL
	PST_CTX_LINE_ADD0
	PST_CTX_LINE_ADD
	PST_UNI_FILE_ADD
	PST_UNI_HUNK
	PST_UNI_LINE
)

func ptNop(ctx *CheckPatchContext) {}

type transition struct {
	re     string
	next   State
	action func(*CheckPatchContext)
}

var patchTransitions = map[State][]transition{
	PST_START: {
		{rePatchRcsid, PST_CENTER, func(ctx *CheckPatchContext) {
			checklineRcsid(ctx.line, ``, "")
		}},
		{"", PST_CENTER, func(ctx *CheckPatchContext) {
			checklineRcsid(ctx.line, ``, "")
		}},
	},

	PST_CENTER: {
		{rePatchEmpty, PST_TEXT, func(ctx *CheckPatchContext) {
			//
		}},
		{rePatchCtxFileDel, PST_CTX_FILE_ADD, func(ctx *CheckPatchContext) {
			if ctx.seenComment {
				ctx.expectEmptyLine()
			} else {
				ctx.expectComment()
			}
			ctx.line.logWarning("Please use unified diffs (diff -u) for patches.")
		}},
		{rePatchUniFileDel, PST_UNI_FILE_ADD, func(ctx *CheckPatchContext) {
			if ctx.seenComment {
				ctx.expectEmptyLine()
			} else {
				ctx.expectComment()
			}
		}},
		{"", PST_TEXT, func(ctx *CheckPatchContext) {
			_ = G.opts.optWarnSpace && ctx.line.logNote("Empty line expected.")
		}},
	},

	PST_TEXT: {
		{rePatchCtxFileDel, PST_CTX_FILE_ADD, func(ctx *CheckPatchContext) {
			if !ctx.seenComment {
				ctx.expectComment()
			}
			ctx.useUnifiedDiffs()
		}},
		{rePatchUniFileDel, PST_UNI_FILE_ADD, func(ctx *CheckPatchContext) {
			if !ctx.seenComment {
				ctx.expectComment()
			}
		}},
		{rePatchText, PST_TEXT, func(ctx *CheckPatchContext) {
			ctx.seenComment = true
		}},
		{"", PST_TEXT, func(ctx *CheckPatchContext) {
		}},
	},

	PST_CTX_FILE_ADD: {
		{rePatchCtxFileAdd, PST_CTX_HUNK, func(ctx *CheckPatchContext) {
			ctx.currentFilename = &ctx.m[1]
			ctx.currentFiletype = new(FileType)
			*ctx.currentFiletype = guessFileType(ctx.line, *ctx.currentFilename)
			_ = G.opts.optDebugPatches && ctx.line.logDebug("filename=%q filetype=%q", *ctx.currentFilename, *ctx.currentFiletype)
			ctx.patchedFiles++
			ctx.hunks = 0
		}},
	},

	PST_CTX_HUNK: {
		{rePatchCtxHunk, PST_CTX_HUNK_DEL, func(ctx *CheckPatchContext) {
			ctx.hunks++
		}},
		{"", PST_TEXT, func(ctx *CheckPatchContext) {
		}},
	},

	PST_CTX_HUNK_DEL: {
		{rePatchCtxHunkDel, PST_CTX_LINE_DEL0, func(ctx *CheckPatchContext) {
			ctx.dellines = new(int)
			if ctx.m[2] != "" {
				*ctx.dellines = (1 + toInt(ctx.m[2]) - toInt(ctx.m[1]))
			} else {
				*ctx.dellines = toInt(ctx.m[1])
			}
		}},
	},

	PST_CTX_LINE_DEL0: {
		{rePatchCtxLineContext, PST_CTX_LINE_DEL, func(ctx *CheckPatchContext) {
			ctx.checkHunkLine(1, 0, PST_CTX_LINE_DEL0)
		}},
		{rePatchCtxLineDel, PST_CTX_LINE_DEL, func(ctx *CheckPatchContext) {
			ctx.checkHunkLine(1, 0, PST_CTX_LINE_DEL0)
		}},
		{rePatchCtxLineMod, PST_CTX_LINE_DEL, func(ctx *CheckPatchContext) {
			ctx.checkHunkLine(1, 0, PST_CTX_LINE_DEL0)
		}},
		{rePatchCtxHunkAdd, PST_CTX_LINE_ADD0, func(ctx *CheckPatchContext) {
			ctx.dellines = nil
			ctx.addlines = new(int)
			if 2 < len(ctx.m) {
				*ctx.addlines = 1 + toInt(ctx.m[2]) - toInt(ctx.m[1])
			} else {
				*ctx.addlines = toInt(ctx.m[1])
			}
		}},
	},

	PST_CTX_LINE_DEL: {
		{rePatchCtxLineContext, PST_CTX_LINE_DEL, func(ctx *CheckPatchContext) {
			ctx.checkHunkLine(1, 0, PST_CTX_LINE_DEL0)
		}},
		{rePatchCtxLineDel, PST_CTX_LINE_DEL, func(ctx *CheckPatchContext) {
			ctx.checkHunkLine(1, 0, PST_CTX_LINE_DEL0)
		}},
		{rePatchCtxLineMod, PST_CTX_LINE_DEL, func(ctx *CheckPatchContext) {
			ctx.checkHunkLine(1, 0, PST_CTX_LINE_DEL0)
		}},
		{"", PST_CTX_LINE_DEL0, func(ctx *CheckPatchContext) {
			if nilToZero(ctx.dellines) != 0 {
				ctx.line.logWarning("Invalid number of deleted lines (%d missing).", ctx.dellines)
			}
		}},
	},

	PST_CTX_LINE_ADD0: {
		{rePatchCtxLineContext, PST_CTX_LINE_ADD, func(ctx *CheckPatchContext) {
			ctx.checkHunkLine(0, 1, PST_CTX_HUNK)
		}},
		{rePatchCtxLineMod, PST_CTX_LINE_ADD, func(ctx *CheckPatchContext) {
			ctx.checkHunkLine(0, 1, PST_CTX_HUNK)
		}},
		{rePatchCtxLineAdd, PST_CTX_LINE_ADD, func(ctx *CheckPatchContext) {
			ctx.checkHunkLine(0, 1, PST_CTX_HUNK)

		}},
		{"", PST_CTX_HUNK, func(ctx *CheckPatchContext) {
		}},
	},

	PST_CTX_LINE_ADD: {
		{rePatchCtxLineContext, PST_CTX_LINE_ADD, func(ctx *CheckPatchContext) {
			ctx.checkHunkLine(0, 1, PST_CTX_HUNK)
		}},
		{rePatchCtxLineMod, PST_CTX_LINE_ADD, func(ctx *CheckPatchContext) {
			ctx.checkHunkLine(0, 1, PST_CTX_HUNK)
		}},
		{rePatchCtxLineAdd, PST_CTX_LINE_ADD, func(ctx *CheckPatchContext) {
			ctx.checkHunkLine(0, 1, PST_CTX_HUNK)
		}},
		{"", PST_CTX_LINE_ADD0, func(ctx *CheckPatchContext) {
			if nilToZero(ctx.addlines) != 0 {
				ctx.line.logWarning("Invalid number of added lines (%d missing).", ctx.addlines)
			}
		}},
	},

	PST_UNI_FILE_ADD: {
		{rePatchUniFileAdd, PST_UNI_HUNK, func(ctx *CheckPatchContext) {
			ctx.currentFilename = new(string)
			*ctx.currentFilename = ctx.m[1]
			ctx.currentFiletype = new(FileType)
			*ctx.currentFiletype = guessFileType(ctx.line, *ctx.currentFilename)
			_ = G.opts.optDebugPatches && ctx.line.logDebug("filename=%q filetype=%q", ctx.currentFilename, ctx.currentFiletype)
			ctx.patchedFiles++
			ctx.hunks = 0
		}},
	},

	PST_UNI_HUNK: {
		{rePatchUniHunk, PST_UNI_LINE, func(ctx *CheckPatchContext) {
			m := ctx.m
			ctx.dellines = new(int)
			if m[1] != "" {
				*ctx.dellines = toInt(m[2])
			} else {
				*ctx.dellines = 1
			}
			ctx.addlines = new(int)
			if m[3] != "" {
				*ctx.addlines = toInt(m[4])
			} else {
				*ctx.addlines = 1
			}
			ctx.checkText(ctx.line.text)
			if hasSuffix(ctx.line.text, "\r") {
				ctx.line.logError("The hunk header must not end with a CR character.")
				ctx.line.explainError(
					"The MacOS X patch utility cannot handle these.")
			}
			ctx.hunks++
			if m[1] != "" && m[1] != "1" {
				ctx.contextScanningLeading = new(bool)
				*ctx.contextScanningLeading = true
			} else {
				ctx.contextScanningLeading = nil
			}
			ctx.leadingContextLines = 0
			ctx.trailingContextLines = 0
		}},
		{"", PST_TEXT, func(ctx *CheckPatchContext) {
			if ctx.hunks == 0 {
				ctx.line.logWarning("No hunks for file %q.", ctx.currentFilename)
			}
		}},
	},

	PST_UNI_LINE: {
		{rePatchUniLineDel, PST_UNI_LINE, func(ctx *CheckPatchContext) {
			ctx.checkHunkLine(1, 0, PST_UNI_HUNK)
		}},
		{rePatchUniLineAdd, PST_UNI_LINE, func(ctx *CheckPatchContext) {
			ctx.checkHunkLine(0, 1, PST_UNI_HUNK)
		}},
		{rePatchUniLineContext, PST_UNI_LINE, func(ctx *CheckPatchContext) {
			ctx.checkHunkLine(1, 1, PST_UNI_HUNK)
		}},
		{rePatchUniLineNoNewline, PST_UNI_LINE, func(ctx *CheckPatchContext) {
		}},
		{rePatchEmpty, PST_UNI_LINE, func(ctx *CheckPatchContext) {
			_ = G.opts.optWarnSpace && ctx.line.logNote("Leading white-space missing in hunk.")
			ctx.checkHunkLine(1, 1, PST_UNI_HUNK)
		}},
		{"", PST_UNI_HUNK, func(ctx *CheckPatchContext) {
			if nilToZero(ctx.dellines) != 0 || nilToZero(ctx.addlines) != 0 {
				ctx.line.logWarning("Unexpected end of hunk (-%d,+%d expected).", nilToZero(ctx.dellines), nilToZero(ctx.addlines))
			}
		}},
	},
}

func checklinesPatch(lines []*Line) {
	ctx := CheckPatchContext{state: PST_START}
	for lineno := 0; lineno < len(lines); {
		line := lines[lineno]
		text := line.text

		_ = G.opts.optDebugPatches &&
			line.logDebug("state %s hunks %d del %d add %d text %s",
				ctx.state, ctx.hunks, ctx.dellines, ctx.addlines, text)

		found := false
		for _, t := range patchTransitions[ctx.state] {
			if t.re == "" {
				ctx.m = ctx.m[:0]
			} else if ctx.m = match(text, t.re); ctx.m == nil {
				continue
			}

			ctx.redostate = nil
			ctx.nextstate = t.next
			t.action(&ctx)
			if ctx.redostate != nil {
				ctx.state = *ctx.redostate
			} else {
				ctx.state = ctx.nextstate
				if t.re != "" {
					lineno++
				}
			}
			found = true
			break
		}

		if !found {
			line.logError("Internal error: state=%q", ctx.state)
			ctx.state = PST_TEXT
			lineno++
		}
	}

	fname := lines[0].fname
	for ctx.state != PST_TEXT {
		_ = G.opts.optDebugPatches &&
			logDebug(fname, "EOF", "state %s hunks %d del %d add %d",
				ctx.state, ctx.hunks, ctx.dellines, ctx.addlines)

		found := false
		for _, t := range patchTransitions[ctx.state] {
			if t.re == "" {
				ctx.m = ctx.m[:0]
				ctx.redostate = nil
				ctx.nextstate = t.next
				t.action(&ctx)
				if ctx.redostate != nil {
					ctx.state = *ctx.redostate
				} else {
					ctx.state = ctx.nextstate
				}
				found = true
			}
		}

		if !found {
			logError(fname, "EOF", "Internal error: state=%q", ctx.state)
			break
		}
	}

	if ctx.patchedFiles > 1 {
		logWarning(fname, NO_LINES, "Contains patches for %d files, should be only one.", ctx.patchedFiles)
	} else if ctx.patchedFiles == 0 {
		logError(fname, NO_LINES, "Contains no patch.")
	}

	checklinesTrailingEmptyLines(lines)
}

type CheckPatchContext struct {
	state                  State
	redostate              *State
	nextstate              State
	dellines               *int
	addlines               *int
	hunks                  int
	seenComment            bool
	currentFilename        *string
	currentFiletype        *FileType
	patchedFiles           int
	leadingContextLines    int
	trailingContextLines   int
	contextScanningLeading *bool
	line                   *Line
	m                      []string
}

func (ctx *CheckPatchContext) expectEmptyLine() {
	_ = G.opts.optWarnSpace && ctx.line.logNote("Empty line expected.")
}

func (ctx *CheckPatchContext) expectComment() {
	ctx.line.logError("Comment expected.")
	ctx.line.explainError(
		"Each patch must document why it is necessary. If it has been applied",
		"because of a security issue, a reference to the CVE should be mentioned",
		"as well.",
		"",
		"Since it is our goal to have as few patches as possible, all patches",
		"should be sent to the upstream maintainers of the package. After you",
		"have done so, you should add a reference to the bug report containing",
		"the patch.")
}

func (ctx *CheckPatchContext) useUnifiedDiffs() {
	ctx.line.logWarning("Please use unified diffs (diff -u) for patches.")
}

func (ctx *CheckPatchContext) checkText(text string) {
	if m, tagname := match1(text, `\$(Author|Date|Header|Id|Locker|Log|Name|RCSfile|Revision|Source|State|`+G.opts.optRcsIds+`)(?::[^\$]*)?\$`); m {
		if match(text, rePatchUniHunk) != nil {
			ctx.line.logWarning("Found RCS tag \"$%s$\". Please remove it.", tagname)
		} else {
			ctx.line.logWarning("Found RCS tag \"$%s$\". Please remove it by reducing the number of context lines using pkgdiff or \"diff -U[210]\".", tagname)
		}
	}
}

func (ctx *CheckPatchContext) checkContents() {
	if 1 < len(ctx.m) {
		ctx.checkText(ctx.m[1])
	}
}

func (ctx *CheckPatchContext) checkAddedContents() {
	if !(1 < len(ctx.m)) {
		return
	}

	line := ctx.line
	addedText := ctx.m[1]

	checklineCppMacroNames(line, addedText)

	switch *ctx.currentFiletype {
	case FT_SHELL:
	case FT_MAKE:
		// This check is not as accurate as the similar one in checklineMkShelltext.
		for _, shellword := range regexp.MustCompile(reShellword).FindAllString(addedText, -1) {
			if !hasPrefix(shellword, "#") {
				checklineMkAbsolutePathname(line, shellword)
			}
		}
	case FT_SOURCE:
		checklineSourceAbsolutePathname(line, addedText)
	case FT_CONFIGURE:
		if match(addedText, `: Avoid regenerating within pkgsrc$`) != nil {
			line.logError("This code must not be included in patches.")
			line.explainError(
				"It is generated automatically by pkgsrc after the patch phase.",
				"",
				"For more details, look for \"configure-scripts-override\" in",
				"mk/configure/gnu-configure.mk.")
		}
	case FT_IGNORE:
		break
	default:
		checklineOtherAbsolutePathname(line, addedText)
	}
}

func (ctx *CheckPatchContext) checkHunkEnd(deldelta, adddelta int, newstate State) {
	if deldelta > 0 && nilToZero(ctx.dellines) == 0 {
		ctx.redostate = &newstate
		if nilToZero(ctx.addlines) > 0 {
			ctx.line.logError("Expected %d more lines to be added.", *ctx.addlines)
		}
		return
	}

	if adddelta > 0 && nilToZero(ctx.addlines) == 0 {
		ctx.redostate = &newstate
		if nilToZero(ctx.dellines) > 0 {
			ctx.line.logError("Expected %d more lines to be deleted.", *ctx.dellines)
		}
		return
	}

	if ctx.contextScanningLeading != nil {
		if deldelta != 0 && adddelta != 0 {
			if *ctx.contextScanningLeading {
				ctx.leadingContextLines++
			} else {
				ctx.trailingContextLines++
			}
		} else {
			if *ctx.contextScanningLeading {
				*ctx.contextScanningLeading = false
			} else {
				ctx.trailingContextLines = 0
			}
		}
	}

	if deldelta > 0 {
		*ctx.dellines -= deldelta
	}
	if adddelta > 0 {
		*ctx.addlines -= adddelta
	}

	if nilToZero(ctx.dellines) == 0 && nilToZero(ctx.addlines) == 0 {
		if ctx.contextScanningLeading != nil {
			if ctx.leadingContextLines != ctx.trailingContextLines {
				_ = G.opts.optDebugPatches && ctx.line.logWarning(
					"The hunk that ends here does not have as many leading (%d) as trailing (%d) lines of context.",
					ctx.leadingContextLines, ctx.trailingContextLines)
			}
		}
		ctx.nextstate = newstate
	}
}

func (ctx *CheckPatchContext) checkHunkLine(deldelta, adddelta int, newstate State) {
	ctx.checkContents()
	ctx.checkHunkEnd(deldelta, adddelta, newstate)

	// If -Wextra is given, the context lines are checked for
	// absolute paths and similar things. If it is not given,
	// only those lines that really add something to the patched
	// file are checked.
	if adddelta > 0 && (deldelta == 0 || G.opts.optWarnExtra) {
		ctx.checkAddedContents()
	}
}
