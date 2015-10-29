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
	if !strings.Contains(basename, ".") {
		return FT_UNKNOWN
	}

	_ = GlobalVars.opts.optDebugMisc && line.logDebug("Unknown file type.")
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
			line.logWarningF("The macro %q is not portable enough. Please use %q instead", macro, better)
			line.explainWarning("See the pkgsrc guide, section \"CPP defines\" for details.")
		} else if match(macro, `(?i)^_+NetBSD_+Version_+$`) != nil && macro != "__NetBSD_Version__" {
			line.logWarningF("Misspelled variant %q of %q.", macro, "__NetBSD_Version__")
		}
	}
}

// Looks for strings like "/dev/cd0" appearing in source code
func checklineSourceAbsolutePathname(line *Line, text string) {
	if matched, before, _, str := match3(text, `(.*)([\"'])(/\w[^\"']*)\2`); matched {
		_ = GlobalVars.opts.optDebugMisc && line.logDebugF("checklineSourceAbsolutePathname: before=%q, str=%q", before, str)

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
	_ = GlobalVars.opts.optDebugTrace && line.logDebugF("checklineOtherAbsolutePathname %q", text)

	if match(text, `^#[^!]`) != nil {
		// Don't warn for absolute pathnames in comments, except for shell interpreters.

	} else if m, before, path, _ := match3(text, `^(.*?)((?:/[\w.]+)*/(?:bin|dev|etc|home|lib|mnt|opt|proc|sbin|tmp|usr|var)\b[\w./\-]*)(.*)$`); m {
		if strings.HasSuffix(before, "@") {
			// ok; autotools example: @PREFIX@/bin

		} else if match(before, `[)}]`) != nil {
			// ok; autotools example: ${prefix}/bin

		} else if match(before, `\+\s*["']$`) != nil {
			// ok; Python example: prefix + '/lib'

		} else if match(before, `\w$`) != nil {
			// ok; shell example: libdir=$prefix/lib

		} else {
			_ = GlobalVars.opts.optDebugMisc && line.logDebugF("before=%q", before)
			checkwordAbsolutePathname(line, path)
		}
	}
}

func checkfilePatch(fname string) {
	_ = GlobalVars.opts.optDebugTrace && logDebugF(fname, NO_LINES, "checkfilePatch()")

	checkperms(fname)
	lines, err := loadLines(fname, false)
	if err != nil {
		logErrorF(fname, NO_LINES, "Cannot be read.")
		return
	}
	if len(lines) == 0 {
		logErrorF(fname, NO_LINES, "Must not be empty.")
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

func checklinesPatch(lines []*Line) {
	ctx := CheckPatchContext{state: PST_START}
	for lineno := 0; lineno < len(lines); {
		line := lines[lineno]
		text := line.text

		_ = GlobalVars.opts.optDebugPatches &&
			line.logDebugF("state %s hunks %d del %d add %d %s",
				ctx.state, ctx.hunks, ctx.dellines, ctx.addlines, text)

		lines[0].logErrorF("not implemented")

	}

	lines[0].logErrorF("not implemented")
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
	currentFiletype        *string
	patchedFiles           int
	leadingContextLines    int
	trailingContextLines   int
	contextScanningLeading *bool
	line                   *Line
	m                      []string
}

func (ctx *CheckPatchContext) logErrorCommentExpected() {
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

func (ctx *CheckPatchContext) checkText(text string) {
	if m := match(text, `\$(Author|Date|Header|Id|Locker|Log|Name|RCSfile|Revision|Source|State|`+GlobalVars.opts.optRcsIds+`)(?::[^\$]*)?\$`); m != nil {
		tagname := m[1]

		if match(text, rePatchUniHunk) != nil {
			ctx.line.logWarningF("Found RCS tag \"$%s$\". Please remove it.", tagname)
		} else {
			ctx.line.logWarningF("Found RCS tag \"$%s$\". Please remove it by reducing the number of context lines using pkgdiff or \"diff -U[210]\".", tagname)
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
	case "shell":
	case "make":
		// This check is not as accurate as the similar one in checklineMkShelltext.
		for _, shellword := range regexp.MustCompile(reShellword).FindAllString(addedText, -1) {
			if !strings.HasPrefix(shellword, "#") {
				checklineMkAbsolutePathname(line, shellword)
			}
		}
	case "source":
		checklineSourceAbsolutePathname(line, addedText)
	case "configure":
		if match(addedText, `: Avoid regenerating within pkgsrc$`) != nil {
			line.logErrorF("This code must not be included in patches.")
			line.explainError(
				"It is generated automatically by pkgsrc after the patch phase.",
				"",
				"For more details, look for \"configure-scripts-override\" in",
				"mk/configure/gnu-configure.mk.")
		}
	case "ignore":
		break
	default:
		checklineOtherAbsolutePathname(line, addedText)
	}
}

func (ctx *CheckPatchContext) checkHunkEnd(deldelta, adddelta int, newstate State) {
	if deldelta > 0 && nilToZero(ctx.dellines) == 0 {
		ctx.redostate = &newstate
		if nilToZero(ctx.addlines) > 0 {
			ctx.line.logErrorF("Expected %d more lines to be added.", *ctx.addlines)
		}
		return
	}

	if adddelta > 0 && nilToZero(ctx.addlines) == 0 {
		ctx.redostate = &newstate
		if nilToZero(ctx.dellines) > 0 {
			ctx.line.logErrorF("Expected %d more lines to be deleted.", *ctx.dellines)
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
				_ = GlobalVars.opts.optDebugPatches && ctx.line.logWarningF(
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
	if adddelta > 0 && (deldelta == 0 || GlobalVars.opts.optWarnExtra) {
		ctx.checkAddedContents()
	}
}
