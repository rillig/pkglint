package main

// Expecter records the state when checking a list of lines from top to bottom.
type Expecter struct {
	lines []*Line
	index int
	m     []string
}

func NewExpecter(lines []*Line) *Expecter {
	return &Expecter{lines, 0, nil}
}

func (exp *Expecter) CurrentLine() *Line {
	if exp.index < len(exp.lines) {
		return exp.lines[exp.index]
	}

	return NewLineEOF(exp.lines[0].Fname)
}

func (exp *Expecter) PreviousLine() *Line {
	return exp.lines[exp.index-1]
}

func (exp *Expecter) EOF() bool {
	return !(exp.index < len(exp.lines))
}

func (exp *Expecter) Advance() bool {
	exp.index++
	exp.m = nil
	return true
}

func (exp *Expecter) stepBack() {
	exp.index--
}

func (exp *Expecter) advanceIfMatches(re string) bool {
	if G.opts.DebugTrace {
		defer tracecall2("Expecter.advanceIfMatches", exp.CurrentLine().Text, re)()
	}

	if !exp.EOF() {
		if m := match(exp.lines[exp.index].Text, re); m != nil {
			exp.index++
			exp.m = m
			return true
		}
	}
	return false
}

func (exp *Expecter) AdvanceIfPrefix(prefix string) bool {
	if G.opts.DebugTrace {
		defer tracecall2("Expecter.advanceIfPrefix", exp.CurrentLine().Text, prefix)()
	}

	return !exp.EOF() && hasPrefix(exp.lines[exp.index].Text, prefix) && exp.Advance()
}

func (exp *Expecter) AdvanceIfEquals(text string) bool {
	if G.opts.DebugTrace {
		defer tracecall2("Expecter.advanceIfEquals", exp.CurrentLine().Text, text)()
	}

	return !exp.EOF() && exp.lines[exp.index].Text == text && exp.Advance()
}

func (exp *Expecter) ExpectEmptyLine() bool {
	if exp.AdvanceIfEquals("") {
		return true
	}

	if G.opts.WarnSpace {
		if !exp.CurrentLine().autofixInsertBefore("") {
			exp.CurrentLine().note0("Empty line expected.")
		}
	}
	return false
}

func (exp *Expecter) ExpectText(text string) bool {
	if !exp.EOF() && exp.lines[exp.index].Text == text {
		exp.index++
		exp.m = nil
		return true
	}

	exp.CurrentLine().warn1("This line should contain the following text: %s", text)
	return false
}
