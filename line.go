package main

// When files are read in by pkglint, they are interpreted in terms of
// lines. For Makefiles, line continuations are handled properly, allowing
// multiple raw lines to end in a single logical line. For other files
// there is a 1:1 translation.
//
// A difference between the raw and the logical lines is that the
// raw lines include the line end sequence, whereas the logical lines
// do not.
//
// Some methods allow modification of the raw lines contained in the
// logical line, but leave the “text” field untouched. These methods are
// used in the --autofix mode.

import (
	"fmt"
	"io"
	"strconv"
	"strings"
)

type RawLine struct {
	lineno int
	orignl string
	textnl string
}

func (rline *RawLine) String() string {
	return strconv.Itoa(rline.lineno) + ":" + rline.textnl
}

type Line struct {
	fname          string
	firstLine      int32 // Zero means not applicable, -1 means EOF
	lastLine       int32 // Usually the same as firstLine, may differ in Makefiles
	text           string
	raw            []*RawLine
	changed        bool
	before         []*RawLine
	after          []*RawLine
	autofixMessage *string
}

func NewLine(fname string, lineno int, text string, rawLines []*RawLine) *Line {
	return NewLineMulti(fname, lineno, lineno, text, rawLines)
}

// NewLineMulti is for logical Makefile lines that end with backslash.
func NewLineMulti(fname string, firstLine, lastLine int, text string, rawLines []*RawLine) *Line {
	return &Line{fname, int32(firstLine), int32(lastLine), text, rawLines, false, nil, nil, nil}
}

// NewLineEOF creates a dummy line for logging, with the “line number” EOF.
func NewLineEOF(fname string) *Line {
	return NewLineMulti(fname, -1, 0, "", nil)
}

func (line *Line) rawLines() []*RawLine {
	switch { // prevent inlining
	}
	return append(append(append([]*RawLine(nil), line.before...), line.raw...), line.after...)
}

func (line *Line) linenos() string {
	switch {
	case line.firstLine == -1:
		return "EOF"
	case line.firstLine == 0:
		return ""
	case line.firstLine == line.lastLine:
		return strconv.Itoa(int(line.firstLine))
	default:
		return strconv.Itoa(int(line.firstLine)) + "--" + strconv.Itoa(int(line.lastLine))
	}
}

func (line *Line) IsMultiline() bool {
	return line.firstLine > 0 && line.firstLine != line.lastLine
}

func (line *Line) printSource(out io.Writer) {
	if G.opts.PrintSource {
		io.WriteString(out, "\n")
		for _, rawLine := range line.rawLines() {
			if rawLine.textnl != rawLine.orignl {
				if rawLine.orignl != "" {
					io.WriteString(out, "- "+rawLine.orignl)
				}
				if rawLine.textnl != "" {
					io.WriteString(out, "+ "+rawLine.textnl)
				}
			} else {
				io.WriteString(out, "> "+rawLine.orignl)
			}
		}
	}
}

func (line *Line) fatalf(format string, args ...interface{}) {
	line.printSource(G.logErr)
	fatalf(line.fname, line.linenos(), format, args...)
}

func (line *Line) errorf(format string, args ...interface{}) {
	line.printSource(G.logOut)
	errorf(line.fname, line.linenos(), format, args...)
	line.logAutofix()
}
func (line *Line) error0(format string)             { line.errorf(format) }
func (line *Line) error1(format, arg1 string)       { line.errorf(format, arg1) }
func (line *Line) error2(format, arg1, arg2 string) { line.errorf(format, arg1, arg2) }

func (line *Line) warnf(format string, args ...interface{}) {
	line.printSource(G.logOut)
	warnf(line.fname, line.linenos(), format, args...)
	line.logAutofix()
}
func (line *Line) warn0(format string)             { line.warnf(format) }
func (line *Line) warn1(format, arg1 string)       { line.warnf(format, arg1) }
func (line *Line) warn2(format, arg1, arg2 string) { line.warnf(format, arg1, arg2) }

func (line *Line) notef(format string, args ...interface{}) {
	line.printSource(G.logOut)
	notef(line.fname, line.linenos(), format, args...)
	line.logAutofix()
}
func (line *Line) note0(format string)             { line.notef(format) }
func (line *Line) note1(format, arg1 string)       { line.notef(format, arg1) }
func (line *Line) note2(format, arg1, arg2 string) { line.notef(format, arg1, arg2) }

func (line *Line) debugf(format string, args ...interface{}) {
	line.printSource(G.logOut)
	debugf(line.fname, line.linenos(), format, args...)
	line.logAutofix()
}
func (line *Line) debug1(format, arg1 string)       { line.debugf(format, arg1) }
func (line *Line) debug2(format, arg1, arg2 string) { line.debugf(format, arg1, arg2) }

func (line *Line) String() string {
	return line.fname + ":" + line.linenos() + ": " + line.text
}

func (line *Line) logAutofix() {
	if line.autofixMessage != nil {
		autofixf(line.fname, line.linenos(), "%s", *line.autofixMessage)
		line.autofixMessage = nil
	}
}

func (line *Line) autofixInsertBefore(text string) bool {
	if G.opts.PrintAutofix || G.opts.Autofix {
		line.before = append(line.before, &RawLine{0, "", text + "\n"})
	}
	return line.noteAutofix("Inserting a line %q before this line.", text)
}

func (line *Line) autofixInsertAfter(text string) bool {
	if G.opts.PrintAutofix || G.opts.Autofix {
		line.after = append(line.after, &RawLine{0, "", text + "\n"})
	}
	return line.noteAutofix("Inserting a line %q after this line.", text)
}

func (line *Line) autofixDelete() bool {
	if G.opts.PrintAutofix || G.opts.Autofix {
		for _, rawLine := range line.raw {
			rawLine.textnl = ""
		}
	}
	return line.noteAutofix("Deleting this line.")
}

func (line *Line) autofixReplace(from, to string) bool {
	for _, rawLine := range line.raw {
		if rawLine.lineno != 0 {
			if replaced := strings.Replace(rawLine.textnl, from, to, 1); replaced != rawLine.textnl {
				if G.opts.PrintAutofix || G.opts.Autofix {
					rawLine.textnl = replaced
				}
				return line.noteAutofix("Replacing %q with %q.", from, to)
			}
		}
	}
	return false
}

func (line *Line) autofixReplaceRegexp(from, to string) bool {
	for _, rawLine := range line.raw {
		if rawLine.lineno != 0 {
			if replaced := regcomp(from).ReplaceAllString(rawLine.textnl, to); replaced != rawLine.textnl {
				if G.opts.PrintAutofix || G.opts.Autofix {
					rawLine.textnl = replaced
				}
				return line.noteAutofix("Replacing regular expression %q with %q.", from, to)
			}
		}
	}
	return false
}

func (line *Line) noteAutofix(format string, args ...interface{}) (hasBeenFixed bool) {
	if line.firstLine < 1 {
		return false
	}
	line.changed = true
	if G.opts.Autofix {
		autofixf(line.fname, line.linenos(), format, args...)
		return true
	}
	if G.opts.PrintAutofix {
		msg := fmt.Sprintf(format, args...)
		line.autofixMessage = &msg
	}
	return false
}

func (line *Line) checkAbsolutePathname(text string) {
	if G.opts.DebugTrace {
		defer tracecall1("Line.checkAbsolutePathname", text)()
	}

	// In the GNU coding standards, DESTDIR is defined as a (usually
	// empty) prefix that can be used to install files to a different
	// location from what they have been built for. Therefore
	// everything following it is considered an absolute pathname.
	//
	// Another context where absolute pathnames usually appear is in
	// assignments like "bindir=/bin".
	if m, path := match1(text, `(?:^|\$[{(]DESTDIR[)}]|[\w_]+\s*=\s*)(/(?:[^"'\s]|"[^"*]"|'[^']*')*)`); m {
		if matches(path, `^/\w`) {
			checkwordAbsolutePathname(line, path)
		}
	}
}

func (line *Line) checkLength(maxlength int) {
	if len(line.text) > maxlength {
		line.warnf("Line too long (should be no more than %d characters).", maxlength)
		explain3(
			"Back in the old time, terminals with 80x25 characters were common.",
			"And this is still the default size of many terminal emulators.",
			"Moderately short lines also make reading easier.")
	}
}

func (line *Line) checkValidCharacters(reChar string) {
	rest := regcomp(reChar).ReplaceAllString(line.text, "")
	if rest != "" {
		uni := ""
		for _, c := range rest {
			uni += fmt.Sprintf(" %U", c)
		}
		line.warn1("Line contains invalid characters (%s).", uni[1:])
	}
}

func (line *Line) checkTrailingWhitespace() {
	if hasSuffix(line.text, " ") || hasSuffix(line.text, "\t") {
		if !line.autofixReplaceRegexp(`\s+\n$`, "\n") {
			line.note0("Trailing white-space.")
			explain2(
				"When a line ends with some white-space, that space is in most cases",
				"irrelevant and can be removed.")
		}
	}
}

func checklineRcsid(line *Line, prefixRe, suggestedPrefix string) bool {
	if G.opts.DebugTrace {
		defer tracecall2("checklineRcsid", prefixRe, suggestedPrefix)()
	}

	if matches(line.text, `^`+prefixRe+`\$`+`NetBSD(?::[^\$]+)?\$$`) {
		return true
	}

	if !line.autofixInsertBefore(suggestedPrefix + "$" + "NetBSD$") {
		line.error1("Expected %q.", suggestedPrefix+"$"+"NetBSD$")
		explain3(
			"Several files in pkgsrc must contain the CVS Id, so that their current",
			"version can be traced back later from a binary package. This is to",
			"ensure reproducible builds, for example for finding bugs.")
	}
	return false
}
