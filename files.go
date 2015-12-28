package main

import (
	"io/ioutil"
	"os"
	"strconv"
	"strings"
)

func LoadNonemptyLines(fname string, joinContinuationLines bool) []*Line {
	lines, err := readLines(fname, joinContinuationLines)
	if err != nil {
		errorf(fname, noLines, "Cannot be read.")
		return nil
	}
	if len(lines) == 0 {
		errorf(fname, noLines, "Must not be empty.")
		return nil
	}
	return lines
}

func LoadExistingLines(fname string, foldBackslashLines bool) []*Line {
	lines, err := readLines(fname, foldBackslashLines)
	if err != nil {
		fatalf(fname, noLines, "Cannot be read.")
	}
	if lines == nil {
		fatalf(fname, noLines, "Must not be empty.")
	}
	return lines
}

func getLogicalLine(fname string, rawLines []*RawLine, pindex *int) *Line {
	{ // Handle the common case efficiently
		index := *pindex
		rawLine := rawLines[*pindex]
		textnl := rawLine.textnl
		if hasSuffix(textnl, "\n") && !hasSuffix(textnl, "\\\n") {
			*pindex = index + 1
			return NewLine(fname, rawLine.lineno, textnl[:len(textnl)-1], rawLines[index:index+1])
		}
	}

	text := ""
	index := *pindex
	firstlineno := rawLines[index].lineno
	var lineRawLines []*RawLine
	interestingRawLines := rawLines[index:]

	for i, rawLine := range interestingRawLines {
		indent, rawText, outdent, cont := splitRawLine(rawLine.textnl)

		if text == "" {
			text += indent
		}
		text += rawText
		lineRawLines = append(lineRawLines, rawLine)

		if cont != "" && i != len(interestingRawLines)-1 {
			text += " "
			index++
		} else {
			text += outdent + cont
			break
		}
	}

	lastlineno := rawLines[index].lineno
	*pindex = index + 1

	return NewLineMulti(fname, firstlineno, lastlineno, text, lineRawLines)
}

func splitRawLine(textnl string) (leadingWhitespace, text, trailingWhitespace, cont string) {
	i, m := 0, len(textnl)

	if m > i && textnl[m-1] == '\n' {
		m--
	}

	if m > i && textnl[m-1] == '\\' {
		m--
		cont = textnl[m : m+1]
	}

	trailingEnd := m
	for m > i && (textnl[m-1] == ' ' || textnl[m-1] == '\t') {
		m--
	}
	trailingStart := m
	trailingWhitespace = textnl[trailingStart:trailingEnd]

	leadingStart := i
	for i < m && (textnl[i] == ' ' || textnl[i] == '\t') {
		i++
	}
	leadingEnd := i
	leadingWhitespace = textnl[leadingStart:leadingEnd]

	text = textnl[leadingEnd:trailingStart]
	return
}

func readLines(fname string, joinContinuationLines bool) ([]*Line, error) {
	rawText, err := ioutil.ReadFile(fname)
	if err != nil {
		return nil, err
	}

	return convertToLogicalLines(fname, string(rawText), joinContinuationLines), nil
}

func convertToLogicalLines(fname string, rawText string, joinContinuationLines bool) []*Line {
	var rawLines []*RawLine
	for lineno, rawLine := range strings.SplitAfter(rawText, "\n") {
		if rawLine != "" {
			rawLines = append(rawLines, &RawLine{1 + lineno, rawLine, rawLine})
		}
	}

	var loglines []*Line
	if joinContinuationLines {
		for lineno := 0; lineno < len(rawLines); {
			loglines = append(loglines, getLogicalLine(fname, rawLines, &lineno))
		}
	} else {
		for _, rawLine := range rawLines {
			text := strings.TrimSuffix(rawLine.textnl, "\n")
			logline := NewLine(fname, rawLine.lineno, text, []*RawLine{rawLine})
			loglines = append(loglines, logline)
		}
	}

	if 0 < len(rawLines) && !hasSuffix(rawLines[len(rawLines)-1].textnl, "\n") {
		errorf(fname, strconv.Itoa(rawLines[len(rawLines)-1].lineno), "File must end with a newline.")
	}

	return loglines
}

func saveAutofixChanges(lines []*Line) (autofixed bool) {
	if !G.opts.Autofix {
		for _, line := range lines {
			if line.changed {
				G.autofixAvailable = true
			}
		}
		return
	}

	changes := make(map[string][]*RawLine)
	changed := make(map[string]bool)
	for _, line := range lines {
		if line.changed {
			changed[line.fname] = true
		}
		changes[line.fname] = append(changes[line.fname], line.rawLines()...)
	}

	for fname := range changed {
		rawLines := changes[fname]
		tmpname := fname + ".pkglint.tmp"
		text := ""
		for _, rawLine := range rawLines {
			text += rawLine.textnl
		}
		err := ioutil.WriteFile(tmpname, []byte(text), 0666)
		if err != nil {
			errorf(tmpname, noLines, "Cannot write.")
			continue
		}
		err = os.Rename(tmpname, fname)
		if err != nil {
			errorf(fname, noLines, "Cannot overwrite with auto-fixed content.")
			continue
		}
		autofixf(fname, noLines, "Has been auto-fixed. Please re-run pkglint.")
		autofixed = true
	}
	return
}
