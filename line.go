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
	"strings"
)

type RawLine struct {
	lineno int
	textnl string
}

func (rline *RawLine) String() string {
	return sprintf("%d:%s", rline.lineno, rline.textnl)
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

// NewLineEof creates a dummy line for logging.
func NewLineEof(fname string) *Line {
	return NewLineMulti(fname, -1, 0, "", nil)
}

func (ln *Line) rawLines() []*RawLine {
	return append(append(append([]*RawLine(nil), ln.before...), ln.raw...), ln.after...)
}

func (ln *Line) linenos() string {
	switch {
	case ln.firstLine == -1:
		return "EOF"
	case ln.firstLine == 0:
		return ""
	case ln.firstLine == ln.lastLine:
		return sprintf("%d", ln.firstLine)
	default:
		return sprintf("%d--%d", ln.firstLine, ln.lastLine)
	}
}

func (ln *Line) IsMultiline() bool {
	return ln.firstLine > 0 && ln.firstLine != ln.lastLine
}

func (ln *Line) printSource(out io.Writer) {
	if G.opts.PrintSource {
		io.WriteString(out, "\n")
		for _, rawLine := range ln.rawLines() {
			fmt.Fprintf(out, "> %s", rawLine.textnl)
		}
	}
}

func (ln *Line) fatalf(format string, args ...interface{}) {
	ln.printSource(G.logErr)
	fatalf(ln.fname, ln.linenos(), format, args...)
}

func (ln *Line) errorf(format string, args ...interface{}) bool {
	ln.printSource(G.logOut)
	return errorf(ln.fname, ln.linenos(), format, args...) && ln.logAutofix()
}

func (ln *Line) warnf(format string, args ...interface{}) bool {
	ln.printSource(G.logOut)
	return warnf(ln.fname, ln.linenos(), format, args...) && ln.logAutofix()
}
func (ln *Line) warn0(format string) bool             { return ln.warnf(format) }
func (ln *Line) warn1(format, arg1 string) bool       { return ln.warnf(format, arg1) }
func (ln *Line) warn2(format, arg1, arg2 string) bool { return ln.warnf(format, arg1, arg2) }

func (ln *Line) notef(format string, args ...interface{}) bool {
	ln.printSource(G.logOut)
	return notef(ln.fname, ln.linenos(), format, args...) && ln.logAutofix()
}

func (ln *Line) debugf(format string, args ...interface{}) bool {
	ln.printSource(G.logOut)
	return debugf(ln.fname, ln.linenos(), format, args...) && ln.logAutofix()
}
func (ln *Line) debug1(format, arg1 string) bool { return ln.debugf(format, arg1) }

func (ln *Line) String() string {
	return ln.fname + ":" + ln.linenos() + ": " + ln.text
}

func (ln *Line) recordAutofixf(format string, args ...interface{}) {
	msg := sprintf(format, args...)
	ln.autofixMessage = &msg
}

func (ln *Line) logAutofix() bool {
	if ln.autofixMessage != nil {
		notef(ln.fname, ln.linenos(), "%s", *ln.autofixMessage)
		ln.autofixMessage = nil
	}
	return true
}

func (ln *Line) autofixInsertBefore(line string) bool {
	if G.opts.PrintAutofix || G.opts.Autofix {
		ln.before = append(ln.before, &RawLine{0, line + "\n"})
	}
	return ln.noteAutofix("Autofix: inserting a line %q before this line.", line)
}

func (ln *Line) autofixInsertAfter(line string) bool {
	if G.opts.PrintAutofix || G.opts.Autofix {
		ln.after = append(ln.after, &RawLine{0, line + "\n"})
	}
	return ln.noteAutofix("Autofix: inserting a line %q after this line.", line)
}

func (ln *Line) autofixDelete() bool {
	if G.opts.PrintAutofix || G.opts.Autofix {
		ln.raw = nil
	}
	return ln.noteAutofix("Autofix: deleting this line.")
}

func (ln *Line) autofixReplace(from, to string) bool {
	for _, rawLine := range ln.raw {
		if rawLine.lineno != 0 {
			if replaced := strings.Replace(rawLine.textnl, from, to, 1); replaced != rawLine.textnl {
				if G.opts.PrintAutofix || G.opts.Autofix {
					rawLine.textnl = replaced
				}
				return ln.noteAutofix("Autofix: replacing %q with %q.", from, to)
			}
		}
	}
	return false
}

func (ln *Line) autofixReplaceRegexp(from, to string) bool {
	for _, rawLine := range ln.raw {
		if rawLine.lineno != 0 {
			if replaced := regcomp(from).ReplaceAllString(rawLine.textnl, to); replaced != rawLine.textnl {
				if G.opts.PrintAutofix || G.opts.Autofix {
					rawLine.textnl = replaced
				}
				return ln.noteAutofix("Autofix: replacing regular expression %q with %q.", from, to)
			}
		}
	}
	return false
}

func (ln *Line) noteAutofix(format string, args ...interface{}) (hasBeenFixed bool) {
	if ln.firstLine < 1 {
		return false
	}
	ln.changed = true
	if G.opts.Autofix {
		ln.notef(format, args...)
		return true
	}
	if G.opts.PrintAutofix {
		ln.recordAutofixf(format, args...)
	}
	return false
}

func (ln *Line) checkAbsolutePathname(text string) {
	defer tracecall("Line.checkAbsolutePathname", text)()

	// In the GNU coding standards, DESTDIR is defined as a (usually
	// empty) prefix that can be used to install files to a different
	// location from what they have been built for. Therefore
	// everything following it is considered an absolute pathname.
	//
	// Another context where absolute pathnames usually appear is in
	// assignments like "bindir=/bin".
	if m, path := match1(text, `(?:^|\$[{(]DESTDIR[)}]|[\w_]+\s*=\s*)(/(?:[^"'\s]|"[^"*]"|'[^']*')*)`); m {
		if matches(path, `^/\w`) {
			checkwordAbsolutePathname(ln, path)
		}
	}
}
