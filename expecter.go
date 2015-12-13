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

func (ctx *Expecter) currentLine() *Line {
	if ctx.index < len(ctx.lines) {
		return ctx.lines[ctx.index]
	}

	return NewLineEof(ctx.lines[0].fname)
}

func (ctx *Expecter) previousLine() *Line {
	return ctx.lines[ctx.index-1]
}

func (ctx *Expecter) eof() bool {
	return !(ctx.index < len(ctx.lines))
}

func (ctx *Expecter) advance() bool {
	ctx.index++
	ctx.m = nil
	return true
}

func (exp *Expecter) stepBack() {
	exp.index--
}

func (ctx *Expecter) advanceIfMatches(re string) bool {
	defer tracecall("Expecter.advanceIfMatches", ctx.currentLine().text, re)()

	if !ctx.eof() {
		if m := match(ctx.lines[ctx.index].text, re); m != nil {
			ctx.index++
			ctx.m = m
			return true
		}
	}
	return false
}

func (exp *Expecter) advanceIfPrefix(prefix string) bool {
	defer tracecall("Expecter.advanceIfPrefix", exp.currentLine().text, prefix)()

	return !exp.eof() && hasPrefix(exp.lines[exp.index].text, prefix) && exp.advance()
}

func (exp *Expecter) advanceIfEquals(text string) bool {
	defer tracecall("Expecter.advanceIfEquals", exp.currentLine().text, text)()

	return !exp.eof() && exp.lines[exp.index].text == text && exp.advance()
}

func (ctx *Expecter) expectEmptyLine() bool {
	if ctx.advanceIfMatches(`^$`) {
		return true
	}

	if G.opts.WarnSpace {
		ctx.currentLine().notef("Empty line expected.")
		ctx.currentLine().insertBefore("")
	}
	return false
}

func (ctx *Expecter) expectText(text string) bool {
	if !ctx.eof() && ctx.lines[ctx.index].text == text {
		ctx.index++
		ctx.m = nil
		return true
	}

	ctx.currentLine().warnf("This line should contain the following text: %s", text)
	return false
}
