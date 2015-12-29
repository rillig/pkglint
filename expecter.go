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

func (exp *Expecter) currentLine() *Line {
	if exp.index < len(exp.lines) {
		return exp.lines[exp.index]
	}

	return NewLineEOF(exp.lines[0].fname)
}

func (exp *Expecter) previousLine() *Line {
	return exp.lines[exp.index-1]
}

func (exp *Expecter) eof() bool {
	return !(exp.index < len(exp.lines))
}

func (exp *Expecter) advance() bool {
	exp.index++
	exp.m = nil
	return true
}

func (exp *Expecter) stepBack() {
	exp.index--
}

func (exp *Expecter) advanceIfMatches(re string) bool {
	if G.opts.DebugTrace {
		defer tracecall2("Expecter.advanceIfMatches", exp.currentLine().text, re)()
	}

	if !exp.eof() {
		if m := match(exp.lines[exp.index].text, re); m != nil {
			exp.index++
			exp.m = m
			return true
		}
	}
	return false
}

func (exp *Expecter) advanceIfPrefix(prefix string) bool {
	if G.opts.DebugTrace {
		defer tracecall2("Expecter.advanceIfPrefix", exp.currentLine().text, prefix)()
	}

	return !exp.eof() && hasPrefix(exp.lines[exp.index].text, prefix) && exp.advance()
}

func (exp *Expecter) advanceIfEquals(text string) bool {
	if G.opts.DebugTrace {
		defer tracecall2("Expecter.advanceIfEquals", exp.currentLine().text, text)()
	}

	return !exp.eof() && exp.lines[exp.index].text == text && exp.advance()
}

func (exp *Expecter) expectEmptyLine() bool {
	if exp.advanceIfEquals("") {
		return true
	}

	if G.opts.WarnSpace {
		if !exp.currentLine().autofixInsertBefore("") {
			exp.currentLine().note0("Empty line expected.")
		}
	}
	return false
}

func (exp *Expecter) expectText(text string) bool {
	if !exp.eof() && exp.lines[exp.index].text == text {
		exp.index++
		exp.m = nil
		return true
	}

	exp.currentLine().warn1("This line should contain the following text: %s", text)
	return false
}
