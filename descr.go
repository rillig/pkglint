package main

func checkfileDescr(fname string) {
	defer tracecall("checkfileDescr", fname)()

	const (
		maxchars = 80
		maxlines = 24
	)

	checkperms(fname)
	lines := loadNonemptyLines(fname, false)
	if lines == nil {
		return
	}

	for _, line := range lines {
		checklineLength(line, maxchars)
		checklineTrailingWhitespace(line)
		checklineValidCharacters(line, reValidchars)
		if contains(line.text, "${") {
			line.logWarning("Variables are not expanded in the DESCR file.")
		}
	}
	checklinesTrailingEmptyLines(lines)

	if len(lines) > maxlines {
		line := lines[maxlines]

		line.logWarning("File too long (should be no more than %d lines).", maxlines)
		line.explainWarning(
			"A common terminal size is 80x25 characters. The DESCR file should",
			"fit on one screen. It is also intended to give a _brief_ summary",
			"about the package's contents.")
	}

	autofix(lines)
}
