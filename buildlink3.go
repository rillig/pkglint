package main

import (
	"strings"
)

func checkfileBuildlink3Mk(fname string) {
	_ = G.opts.optDebugTrace && logDebug(fname, NO_LINES, "checkfileBuildlink3Mk()")

	checkperms(fname)

	lines, err := loadLines(fname, true)
	if err != nil {
		logError(fname, NO_LINES, "Cannot be read.")
		return
	}
	if len(lines) == 0 {
		logError(fname, NO_LINES, "Must not be empty.")
		return
	}

	parselinesMk(lines)
	checklinesMk(lines)

	exp := &ExpectContext{lines, 0}

x:
	if m := exp.advanceIfMatches(`^#`); m != nil {
		if strings.HasPrefix(m[0], "# XXX") {
			exp.previousLine().logNote("Please read this comment and remove it if appropriate.")
		}
		goto x
	}

	exp.expectEmptyLine()

	if exp.advanceIfMatches(`^BUILDLINK_DEPMETHOD\.(\S+)\?=.*$`) != nil {
		exp.previousLine().logWarning("This line belongs inside the .ifdef block.")
		for exp.advanceIfMatches(`^$`) != nil {
		}
	}

	m := exp.advanceIfMatches(`^BUILDLINK_TREE\+=\s*(\S+)$`)
	if m == nil {
		exp.currentLine().logWarning("Expected a BUILDLINK_TREE line.")
		return
	}

	checklinesBuildlink3Mk(exp.lines, exp.index, m[1])
}
