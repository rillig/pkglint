package main

import (
	"netbsd.org/pkglint/regex"
	"strings"
)

// Expecter records the state when checking a list of lines from top to bottom.
type Expecter struct {
	lines Lines
	index int
	m     []string
}

func NewExpecter(lines Lines) *Expecter {
	return &Expecter{lines, 0, nil}
}

func (exp *Expecter) CurrentLine() Line {
	if exp.index < exp.lines.Len() {
		return exp.lines.Lines[exp.index]
	}

	return NewLineEOF(exp.lines.FileName)
}

func (exp *Expecter) PreviousLine() Line {
	return exp.lines.Lines[exp.index-1]
}

func (exp *Expecter) Index() int {
	return exp.index
}

func (exp *Expecter) EOF() bool {
	return !(exp.index < exp.lines.Len())
}

func (exp *Expecter) Group(index int) string {
	return exp.m[index]
}

// Advance skips the current line and returns true.
func (exp *Expecter) Advance() bool {
	exp.index++
	exp.m = nil
	return true
}

func (exp *Expecter) StepBack() {
	exp.index--
}

func (exp *Expecter) AdvanceIfMatches(re regex.Pattern) bool {
	if trace.Tracing {
		defer trace.Call(exp.CurrentLine().Text, re)()
	}

	if !exp.EOF() {
		if m := G.res.Match(exp.lines.Lines[exp.index].Text, re); m != nil {
			exp.index++
			exp.m = m
			return true
		}
	}
	return false
}

func (exp *Expecter) AdvanceIfPrefix(prefix string) bool {
	if trace.Tracing {
		defer trace.Call2(exp.CurrentLine().Text, prefix)()
	}

	return !exp.EOF() && strings.HasPrefix(exp.lines.Lines[exp.index].Text, prefix) && exp.Advance()
}

func (exp *Expecter) AdvanceIfEquals(text string) bool {
	if trace.Tracing {
		defer trace.Call2(exp.CurrentLine().Text, text)()
	}

	return !exp.EOF() && exp.lines.Lines[exp.index].Text == text && exp.Advance()
}

func (exp *Expecter) ExpectEmptyLine() bool {
	if exp.AdvanceIfEquals("") {
		return true
	}

	if G.opts.WarnSpace {
		fix := exp.CurrentLine().Autofix()
		fix.Notef("Empty line expected.")
		fix.InsertBefore("")
		fix.Apply()
	}
	return false
}

func (exp *Expecter) ExpectText(text string) bool {
	if !exp.EOF() && exp.lines.Lines[exp.index].Text == text {
		exp.index++
		exp.m = nil
		return true
	}

	exp.CurrentLine().Warnf("This line should contain the following text: %s", text)
	return false
}

func (exp *Expecter) SkipToFooter() {
	exp.index = exp.lines.Len() - 2
}

// MkExpecter records the state when checking a list of Makefile lines from top to bottom.
type MkExpecter struct {
	mklines MkLines
	Expecter
}

func NewMkExpecter(mklines MkLines) *MkExpecter {
	return &MkExpecter{mklines, *NewExpecter(mklines.lines)}
}

func (exp *MkExpecter) CurrentMkLine() MkLine {
	return exp.mklines.mklines[exp.index]
}

func (exp *MkExpecter) AdvanceWhile(pred func(mkline MkLine) bool) {
	if trace.Tracing {
		defer trace.Call(exp.CurrentMkLine().Text)()
	}

	for !exp.EOF() && pred(exp.CurrentMkLine()) {
		exp.Advance()
	}
}

func (exp *MkExpecter) AdvanceIf(pred func(mkline MkLine) bool) bool {
	return !exp.EOF() && pred(exp.CurrentMkLine()) && exp.Advance()
}
