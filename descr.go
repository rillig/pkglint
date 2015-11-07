package main

import (
	"strings"
)

func checkfileDescr(fname string) {
	const (
		maxchars = 80
		maxlines = 24
	)

	_ = G.opts.optDebugTrace && logDebug(fname, NO_LINES, "checkfile_DESCR()")

	checkperms(fname)
	lines, err := loadLines(fname, false)
	if err != nil {
		logError(NO_FILE, NO_LINES, "Cannot be read.")
		return
	}

	if len(lines) == 0 {
		logError(NO_FILE, NO_LINES, "Must not be empty.")
		return
	}

	for _, line := range lines {
		checklineLength(line, maxchars)
		checklineTrailingWhitespace(line)
		checklineValidCharacters(line, reValidchars)
		if strings.Contains(line.text, "${") {
			line.logWarning("Variables are not expanded in the DESCR file.")
		}
	}
	checklinesTrailingEmptyLines(lines)

	if len(lines) > maxlines {
		line := lines[maxlines]

		line.logWarning("File too long (should be no more than %d lines).", maxlines)
		line.explainWarning(
			`A common terminal size is 80x25 characters. The DESCR file should
fit on one screen. It is also intended to give a _brief_ summary
about the package's contents.`)
	}

	autofix(lines)
}
