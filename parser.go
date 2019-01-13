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
