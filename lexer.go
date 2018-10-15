package main

// Lexer provides a flexible way of splitting a string into several parts
// by repeatedly chopping off a prefix that matches a string, a function
// or a regular expression.
type Lexer struct {
	rest string
}

type LexerMark string

func NewLexer(text string) *Lexer {
	return &Lexer{text}
}

// Mark returns the current position of the lexer,
// which can later be restored by calling Reset.
func (l *Lexer) Mark() LexerMark {
	return LexerMark(l.rest)
}

// Reset sets the lexer back to the position where
// the corresponding Mark was called.
func (l *Lexer) Reset(mark LexerMark) {
	l.rest = string(mark)
}

// EOF returns whether the whole input has been consumed.
func (l *Lexer) EOF() bool { return l.rest == "" }

// Rest returns the part of the string that has not yet been chopped off.
func (l *Lexer) Rest() string { return l.rest }

// NextString tests whether the remaining text has the given prefix,
// and if so, chops it.
func (l *Lexer) NextString(prefix string) bool {
	if hasPrefix(l.rest, prefix) {
		l.rest = l.rest[len(prefix):]
		return true
	}
	return false
}
