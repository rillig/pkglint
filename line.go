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
			fmt.Fprintf(out, "> %s", physline.textnl)
		}
	}
}
func (self *Line) fatalf(format string, args ...interface{}) bool {
	self.printSource(G.logErr)
	return fatalf(self.fname, self.lines, format, args...)
}
func (self *Line) errorf(format string, args ...interface{}) bool {
	self.printSource(G.logOut)
	return errorf(self.fname, self.lines, format, args...)
}
func (self *Line) warnf(format string, args ...interface{}) bool {
	self.printSource(G.logOut)
	return warnf(self.fname, self.lines, format, args...)
}
func (self *Line) notef(format string, args ...interface{}) bool {
	self.printSource(G.logOut)
	return notef(self.fname, self.lines, format, args...)
}
func (self *Line) debugf(format string, args ...interface{}) bool {
	self.printSource(G.logOut)
	return debugf(self.fname, self.lines, format, args...)
}
func (self *Line) explain(explanation ...string) {
	if G.opts.optExplain {
		io.WriteString(G.logOut, "\n")
		for _, explanationLine := range explanation {
			io.WriteString(G.logOut, "\t"+explanationLine+"\n")
		}
		io.WriteString(G.logOut, "\n")
	}
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
