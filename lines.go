package main

import (
	"netbsd.org/pkglint/regex"
	"path"
)

type Lines = *LinesImpl

type LinesImpl struct {
	FileName string
	BaseName string
	Lines    []Line
}

func NewLines(filename string, lines []Line) Lines {
	return &LinesImpl{filename, path.Base(filename), lines}
}

func (ls *LinesImpl) Len() int { return len(ls.Lines) }

func (ls *LinesImpl) LastLine() Line { return ls.Lines[ls.Len()-1] }

func (ls *LinesImpl) EOFLine() Line { return NewLineMulti(ls.FileName, -1, -1, "", nil) }

func (ls *LinesImpl) Errorf(format string, args ...interface{}) {
	NewLineWhole(ls.FileName).Errorf(format, args...)
}

func (ls *LinesImpl) Warnf(format string, args ...interface{}) {
	NewLineWhole(ls.FileName).Warnf(format, args...)
}

func (ls *LinesImpl) SaveAutofixChanges() {
	SaveAutofixChanges(ls)
}

func (ls *LinesImpl) CheckRcsID(index int, prefixRe regex.Pattern, suggestedPrefix string) bool {
	if trace.Tracing {
		defer trace.Call(prefixRe, suggestedPrefix)()
	}

	line := ls.Lines[index]
	if matches(line.Text, `^`+prefixRe+`\$`+`NetBSD(?::[^\$]+)?\$$`) {
		return true
	}

	fix := line.Autofix()
	fix.Errorf("Expected %q.", suggestedPrefix+"$"+"NetBSD$")
	fix.Explain(
		"Several files in pkgsrc must contain the CVS Id, so that their",
		"current version can be traced back later from a binary package.",
		"This is to ensure reproducible builds, for example for finding bugs.")
	fix.InsertBefore(suggestedPrefix + "$" + "NetBSD$")
	fix.Apply()

	return false
}
