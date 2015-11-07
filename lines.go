package main

import (
	"fmt"
	"io/ioutil"
	"os"
	"regexp"
	"strconv"
	"strings"
)

func loadRawLines(fname string) ([]PhysLine, error) {
	physlines := make([]PhysLine, 0)
	rawtext, err := ioutil.ReadFile(fname)
	if err != nil {
		logError(fname, NO_LINES, "Cannot be read")
		return nil, err
	}
	for lineno, physline := range strings.SplitAfter(string(rawtext), "\n") {
		if physline != "" {
			physlines = append(physlines, PhysLine{1 + lineno, physline})
		}
	}
	return physlines, nil
}

func getLogicalLine(fname string, physlines []PhysLine, pLineno *int) *Line {
	value := ""
	first := true
	lineno := *pLineno
	firstlineno := physlines[lineno].lineno
	lphyslines := make([]PhysLine, 0)

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
	return convertToLogicalLines(fname, physlines, joinContinuationLines), nil
}

func loadNonemptyLines(fname string, joinContinuationLines bool) []*Line {
	checkperms(fname)
	lines, err := loadLines(fname, joinContinuationLines)
	if err != nil {
		logError(fname, NO_LINES, "Cannot be read.")
		return nil
	}
	if len(lines) == 0 {
		logError(fname, NO_LINES, "Must not be empty.")
		return nil
	}
	return lines
}

func convertToLogicalLines(fname string, physlines []PhysLine, joinContinuationLines bool) []*Line {
	loglines := make([]*Line, 0)
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

	return loglines
}

func saveAutofixChanges(lines []*Line) {
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

func autofix(lines []*Line) {
	if G.opts.optAutofix {
		saveAutofixChanges(lines)
	}
}

func checklinesTrailingEmptyLines(lines []*Line) {
	max := len(lines)
	last := max
	for last > 1 && lines[last-1].text == "" {
		last--
	}
	if last != max {
		lines[last].logNote("Trailing empty lines.")
	}
}
