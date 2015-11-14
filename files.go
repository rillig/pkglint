package main

import (
	"io/ioutil"
	"os"
	"strconv"
	"strings"
)

func checkperms(fname string) {
	st, err := os.Stat(fname)
	if err == nil && st.Mode().IsRegular() && (st.Mode().Perm()&0111 != 0) {
		line := NewLine(fname, NO_LINES, "", nil)
		line.logWarning("Should not be executable.")
		line.explainWarning(
			"No package file should ever be executable. Even the INSTALL and",
			"DEINSTALL scripts are usually not usable in the form they have in the",
			"package, as the pathnames get adjusted during installation. So there is",
			"no need to have any file executable.")
	}
}

func loadRawLines(fname string) ([]PhysLine, error) {
	physlines := make([]PhysLine, 0)
	rawtext, err := ioutil.ReadFile(fname)
	if err != nil {
		return nil, err
	}
	for lineno, physline := range strings.SplitAfter(string(rawtext), "\n") {
		if physline != "" {
			physlines = append(physlines, PhysLine{1 + lineno, physline})
		}
	}
	return physlines, nil
}

func getLogicalLine(fname string, physlines []PhysLine, pindex *int) *Line {
	text := ""
	index := *pindex
	firstlineno := physlines[index].lineno
	lphyslines := make([]PhysLine, 0)
	interestingPhyslines := physlines[index:]

	for i, physline := range interestingPhyslines {
		_, indent, phystext, outdent, cont := match4(physline.textnl, `^([ \t]*)(.*?)([ \t]*)(\\?)\n?$`)

		if text == "" {
			text += indent
		}
		text += phystext
		lphyslines = append(lphyslines, physline)

		if cont == "\\" && i != len(interestingPhyslines)-1 {
			text += " "
			index++
		} else {
			text += outdent + cont
			break
		}
	}

	lastlineno := physlines[index].lineno
	*pindex = index + 1

	slineno := ifelseStr(firstlineno == lastlineno,
		sprintf("%d", firstlineno),
		sprintf("%d--%d", firstlineno, lastlineno))
	return NewLine(fname, slineno, text, lphyslines)
}

func loadLines(fname string, joinContinuationLines bool) ([]*Line, error) {
	physlines, err := loadRawLines(fname)
	if err != nil {
		return nil, err
	}
	return convertToLogicalLines(fname, physlines, joinContinuationLines), nil
}

func loadNonemptyLines(fname string, joinContinuationLines bool) []*Line {
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

	if 0 < len(physlines) && !hasSuffix(physlines[len(physlines)-1].textnl, "\n") {
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
