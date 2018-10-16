package textproc

import "strings"

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

// Rest returns the part of the string that has not yet been chopped off.
func (l *Lexer) Rest() string { return l.rest }

// EOF returns whether the whole input has been consumed.
func (l *Lexer) EOF() bool { return l.rest == "" }

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

// NextString tests whether the remaining text has the given prefix,
// and if so, chops it.
func (l *Lexer) NextString(prefix string) string {
	if strings.HasPrefix(l.rest, prefix) {
		l.rest = l.rest[len(prefix):]
		return prefix
	}
	return ""
}

// NextHspace chops off the longest prefix consisting solely of horizontal
// whitespace, which is space (U+0020) and tabs (U+0009).
func (l *Lexer) NextHspace() string {
	// The same code as in NextBytesFunc, inlined here for performance reasons.
	// The Go 1.11 compiler does not inline constant function arguments.
	i := 0
	rest := l.rest
	for len(rest) > i && (rest[i] == ' ' || rest[i] == '\t') {
		i++
	}
	if i != 0 {
		l.rest = rest[i:]
	}
	return rest[:i]
}

// NextByte chops off a single byte and returns true if the rest starts
// with the given byte.
//
// The return type differs from the other methods since creating a string
// would be too much work for such a simple operation.
func (l *Lexer) NextByte(b byte) bool {
	if len(l.rest) > 0 && l.rest[0] == b {
		l.rest = l.rest[1:]
		return true
	}
	return false
}

// NextBytesFunc chops off the longest prefix consisting solely of bytes
// for which fn returns true.
func (l *Lexer) NextBytesFunc(fn func(b byte) bool) string {
	i := 0
	rest := l.rest
	for len(rest) > i && fn(rest[i]) {
		i++
	}
	if i != 0 {
		l.rest = rest[i:]
	}
	return rest[:i]
}
