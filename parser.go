package pkglint

import (
	"netbsd.org/pkglint/textproc"
)

type Parser struct {
	Line         Line
	lexer        *textproc.Lexer
	EmitWarnings bool
}

func NewParser(line Line, s string, emitWarnings bool) *Parser {
	return &Parser{line, textproc.NewLexer(s), emitWarnings}
}

func (p *Parser) EOF() bool {
	return p.lexer.EOF()
}

func (p *Parser) Rest() string {
	return p.lexer.Rest()
}

func (p *Parser) PkgbasePattern() (pkgbase string) {
	lexer := p.lexer

	for {
		mark := lexer.Mark()

		if lexer.SkipRegexp(G.res.Compile(`^\$\{\w+\}`)) ||
			lexer.SkipRegexp(G.res.Compile(`^[\w.*+,{}]+`)) ||
			lexer.SkipRegexp(G.res.Compile(`^\[[\d-]+\]`)) {
			pkgbase += lexer.Since(mark)
			continue
		}

		if lexer.SkipByte('-') {
			if lexer.SkipRegexp(G.res.Compile(`^\d`)) ||
				lexer.SkipRegexp(G.res.Compile(`^\$\{\w*VER\w*\}`)) ||
				lexer.SkipByte('[') {
				lexer.Reset(mark)
				return
			}
			pkgbase += "-"
			continue
		}

		lexer.Reset(mark)
		return
	}
}

type DependencyPattern struct {
	Pkgbase  string // "freeciv-client", "{gcc48,gcc48-libs}", "${EMACS_REQD}"
	LowerOp  string // ">=", ">"
	Lower    string // "2.5.0", "${PYVER}"
	UpperOp  string // "<", "<="
	Upper    string // "3.0", "${PYVER}"
	Wildcard string // "[0-9]*", "1.5.*", "${PYVER}"
}
