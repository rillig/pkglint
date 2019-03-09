package pkglint

import (
	"netbsd.org/pkglint/textproc"
	"strings"
)

// MkTokensLexer parses a sequence of variable uses (like ${VAR:Mpattern})
// interleaved with other text that is uninterpreted by bmake.
type MkTokensLexer struct {
	// The lexer for the current text-only token.
	// If the current token is a variable use, the lexer will always return
	// EOF internally. That is not visible from the outside though, as EOF is
	// overridden in this type.
	*textproc.Lexer

	// The remaining tokens.
	tokens []*MkToken
}

func NewMkTokensLexer(tokens []*MkToken) *MkTokensLexer {
	lexer := &MkTokensLexer{nil, tokens}
	lexer.next()
	return lexer
}

func (m *MkTokensLexer) next() {
	if len(m.tokens) > 0 && m.tokens[0].Varuse == nil {
		m.Lexer = textproc.NewLexer(m.tokens[0].Text)
		m.tokens = m.tokens[1:]
	} else {
		m.Lexer = textproc.NewLexer("")
	}
}

func (m *MkTokensLexer) EOF() bool { return m.Lexer.EOF() && len(m.tokens) == 0 }

func (m *MkTokensLexer) Rest() string {
	var sb strings.Builder
	sb.WriteString(m.Lexer.Rest())
	for _, token := range m.tokens {
		sb.WriteString(token.Text)
	}
	return sb.String()
}

func (m *MkTokensLexer) NextVarUse() *MkToken {
	if m.Lexer.EOF() && len(m.tokens) > 0 && m.tokens[0].Varuse != nil {
		token := m.tokens[0]
		m.tokens = m.tokens[1:]
		m.next()
		return token
	}
	return nil
}

func (m *MkTokensLexer) Mark() MkTokensLexerMark {
	return MkTokensLexerMark{m.Lexer.Rest(), append([]*MkToken(nil), m.tokens...)}
}

func (m *MkTokensLexer) Since(mark MkTokensLexerMark) string {
	early := (&MkTokensLexer{textproc.NewLexer(mark.rest), mark.tokens}).Rest()
	late := m.Rest()

	return strings.TrimSuffix(early, late)
}

func (m *MkTokensLexer) Reset(mark MkTokensLexerMark) {
	m.Lexer = textproc.NewLexer(mark.rest)
	m.tokens = append([]*MkToken(nil), mark.tokens...)
}

type MkTokensLexerMark struct {
	rest   string
	tokens []*MkToken
}
