package main

// When files are read in by pkglint, they are interpreted in terms of
// lines. For Makefiles, line continuations are handled properly, allowing
// multiple physical lines to end in a single logical line. For other files
// there is a 1:1 translation.
//
// A difference between the physical and the logical lines is that the
// physical lines include the line end sequence, whereas the logical lines
// do not.
//
// Some methods allow modification of the physical lines contained in the
// logical line, but leave the C<text> field untouched. These methods are
// used in the --autofix mode.
//
// A line can have some "extra" fields that allow the results of parsing to
// be saved under a name.
//

import (
	"fmt"
	"io"
	"os"
	"strings"
)

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
	extra     map[string]interface{}
}

func NewLine(fname, linenos, text string, physlines []PhysLine) *Line {
	return &Line{fname, linenos, text, physlines, false, []PhysLine{}, []PhysLine{}, make(map[string]interface{})}
}

func (self *Line) physicalLines() []PhysLine {
	return append(self.before, append(self.physlines, self.after...)...)
}
func (self *Line) printSource(out io.Writer) {
	if G.opts.optPrintSource {
		io.WriteString(out, "\n")
		for _, physline := range self.physicalLines() {
			fmt.Fprintf(out, "> %s\n", physline.textnl)
		}
	}
}
func (self *Line) logFatal(format string, args ...interface{}) bool {
	self.printSource(os.Stderr)
	return logFatal(self.fname, self.lines, format, args...)
}
func (self *Line) logError(format string, args ...interface{}) bool {
	self.printSource(os.Stdout)
	return logError(self.fname, self.lines, format, args...)
}
func (self *Line) logWarning(format string, args ...interface{}) bool {
	self.printSource(os.Stdout)
	return logWarning(self.fname, self.lines, format, args...)
}
func (self *Line) logNote(format string, args ...interface{}) bool {
	self.printSource(os.Stdout)
	return logNote(self.fname, self.lines, format, args)
}
func (self *Line) logDebug(format string, args ...interface{}) bool {
	self.printSource(os.Stdout)
	return logDebug(self.fname, self.lines, format, args...)
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

func (line *Line) trace(funcname string, args ...interface{}) {
	if G.opts.optDebugTrace {
		line.logDebug("%s(%s)", funcname, argsStr(args...))
	}
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
			if replaced := reCompile(from).ReplaceAllString(physline.textnl, to); replaced != physline.textnl {
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
