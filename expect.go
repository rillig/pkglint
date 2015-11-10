package main

// High-level iterating through lines and checking them.
type ExpectContext struct {
	lines []*Line
	index int
}

func (ctx *ExpectContext) currentLine() *Line {
	if ctx.index < len(ctx.lines) {
		return ctx.lines[ctx.index]
	}

	return NewLine(ctx.lines[0].fname, "EOF", "", nil) // dummy
}

func (ctx *ExpectContext) previousLine() *Line {
	return ctx.lines[ctx.index-1]
}

func (ctx *ExpectContext) eof() bool {
	return !(ctx.index < len(ctx.lines))
}
func (ctx *ExpectContext) advance() {
	ctx.index++
}

func (ctx *ExpectContext) advanceIfMatches(re string) []string {
	if ctx.index < len(ctx.lines) {
		if m := match(ctx.lines[ctx.index].text, re); m != nil {
			ctx.index++
			return m
		}
	}
	return nil
}

func (ctx *ExpectContext) expectEmptyLine() bool {
	if ctx.advanceIfMatches(`^$`) != nil {
		return true
	}

	_ = G.opts.optWarnSpace && ctx.currentLine().logNote("Empty line expected.")
	return false
}

func (ctx *ExpectContext) expectText(text string) bool {
	if ctx.index < len(ctx.lines) && ctx.lines[ctx.index].text == text {
		ctx.index++
		return true
	}

	ctx.currentLine().logWarning("This line should contain the following text: %s", text)
	return false
}
