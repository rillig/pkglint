package main

import (
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"regexp"
	"strconv"
	"strings"
)

// When files are read in by pkglint, they are interpreted in terms of
// lines. For Makefiles, line continuations are handled properly, allowing
// multiple physical lines to end in a single logical line. For other files
// there is a 1:1 translation.
//
// A difference between the physical and the logical lines is that the
// physical lines include the line end sequence, whereas the logical lines
// do not.
//
// A logical line is a class having the read-only fields C<file>,
// C<lines>, C<text>, C<physlines> and C<is_changed>, as well as some
// methods for printing diagnostics easily.
//
// Some other methods allow modification of the physical lines, but leave
// the logical line (the C<text>) untouched. These methods are used in the
// --autofix mode.
//
// A line can have some "extra" fields that allow the results of parsing to
// be saved under a name.
//
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
	extra     map[string]string
}

func NewLine(fname, linenos, text string, physlines []PhysLine) *Line {
	return &Line{fname, linenos, text, physlines, false, []PhysLine{}, []PhysLine{}, make(map[string]string, 1)}
}
func (self *Line) physicalLines() []PhysLine {
	return append(self.before, append(self.physlines, self.after...)...)
}
func (self *Line) printSource(out io.Writer) {
	if GlobalVars.opts.optPrintSource {
		io.WriteString(out, "\n")
		for _, physline := range self.physicalLines() {
			fmt.Fprintf(out, "> %s", physline.textnl)
		}
	}
}
func (self *Line) logFatal(msg string) {
	self.printSource(os.Stderr)
	logFatal(self.fname, self.lines, msg)
}
func (self *Line) logError(msg string) {
	self.printSource(os.Stdout)
	logError(self.fname, self.lines, msg)
}
func (self *Line) logWarning(msg string) {
	self.printSource(os.Stdout)
	logWarning(self.fname, self.lines, msg)
}
func (self *Line) logNote(msg string) {
	self.printSource(os.Stdout)
	logNote(self.fname, self.lines, msg)
}
func (self *Line) logDebug(msg string) {
	self.printSource(os.Stdout)
	logDebug(self.fname, self.lines, msg)
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
			if replaced := regexp.MustCompile(from).ReplaceAllString(physline.textnl, to); replaced != physline.textnl {
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

func loadRawLines(fname string) ([]PhysLine, error) {
	physlines := make([]PhysLine, 50)
	rawtext, err := ioutil.ReadFile(fname)
	if err != nil {
		logError(fname, NO_LINES, "Cannot be read")
		return nil, err
	}
	for lineno, physline := range strings.SplitAfter(string(rawtext), "\n") {
		if physline != "" {
			physlines = append(physlines, PhysLine{lineno, physline})
		}
	}
	return physlines, nil
}

func getLogicalLine(fname string, physlines []PhysLine, pLineno *int) *Line {
	value := ""
	first := true
	lineno := *pLineno
	firstlineno := physlines[lineno].lineno
	lphyslines := make([]PhysLine, 1)

	for _, physline := range physlines {
		m := regexp.MustCompile(`^([ \t]*)(.*?)([ \t]*)(\\?)\n?$`).FindStringSubmatch(physline.textnl)
		indent, text, outdent, cont := m[1], m[2], m[3], m[4]

		if first {
			value += indent
			first = false
		}

		value += text
		lphyslines = append(lphyslines, physline)

		if cont == "\\" {
			value += " "
		} else {
			value += outdent
			break
		}
	}

	if lineno >= len(physlines) { // The last line in the file is a continuation line
		lineno--
	}
	lastlineno := physlines[lineno].lineno
	*pLineno = lineno + 1

	slineno := ifelseStr(firstlineno == lastlineno, fmt.Sprintf("%d", firstlineno), fmt.Sprintf("%d–%d", firstlineno, lastlineno))
	return NewLine(fname, slineno, value, physlines)
}

func loadLines(fname string, joinContinuationLines bool) ([]*Line, error) {
	physlines, err := loadRawLines(fname)
	if err != nil {
		return nil, err
	}
	return convertToLogicalLines(fname, physlines, joinContinuationLines)
}

func convertToLogicalLines(fname string, physlines []PhysLine, joinContinuationLines bool) ([]*Line, error) {
	loglines := make([]*Line, 0, len(physlines))
	if joinContinuationLines {
		for lineno := 0; lineno < len(physlines); {
			loglines = append(loglines, getLogicalLine(fname, physlines, &lineno))
		}
	} else {
		for _, physline := range physlines {
			loglines = append(loglines, NewLine(fname, strconv.Itoa(physline.lineno), strings.TrimSuffix(physline.textnl, "\n"), []PhysLine{physline}))
		}
	}

	if 0 < len(physlines) && !strings.HasSuffix(physlines[len(physlines)-1].textnl, "\n") {
		logError(fname, strconv.Itoa(physlines[len(physlines)-1].lineno), "File must end with a newline.")
	}

	return loglines, nil
}

func saveAutofixChanges(lines []Line) {
	changes := make(map[string][]PhysLine)
	changed := make(map[string]bool)
	for _, line := range lines {
		if line.changed {
			changed[line.fname] = true
		}
		changes[line.fname] = append(changes[line.fname], line.physicalLines()...)
	}

	for fname := range changed {
		physlines := changes[fname]
		tmpname := fname + ".pkglint.tmp"
		text := ""
		for _, physline := range physlines {
			text += physline.textnl
		}
		err := ioutil.WriteFile(tmpname, []byte(text), 0777)
		if err != nil {
			logError(tmpname, NO_LINES, "Cannot write.")
			continue
		}
		err = os.Rename(tmpname, fname)
		if err != nil {
			logError(fname, NO_LINES, "Cannot overwrite with auto-fixed content.")
			continue
		}
		logNote(fname, NO_LINES, "Has been auto-fixed. Please re-run pkglint.")
	}
}

func loadExistingLines(fname string, foldBackslashLines bool) []*Line {
	lines, err := loadLines(fname, foldBackslashLines)
	if lines == nil || err != nil {
		logFatal(fname, NO_LINES, "Cannot be read.")
	}
	return lines
}
