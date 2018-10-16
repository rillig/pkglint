package textproc

import "strings"

// Lexer provides a flexible way of splitting a string into several parts
// by repeatedly chopping off a prefix that matches a string, a function
// or a regular expression.
type Lexer struct {
	rest string
}

// LexerMark remembers a position in the string being parsed, to be able
// to revert to that position, should a complex expression not match in
// its entirety.
type LexerMark string

// ByteSet is a subset of all 256 possible byte values.
// It is used for matching character classes efficiently.
type ByteSet struct {
	bits [4]uint64
}

func NewLexer(text string) *Lexer {
	return &Lexer{text}
}

// Rest returns the part of the string that has not yet been chopped off.
func (l *Lexer) Rest() string { return l.rest }

// EOF returns whether the whole input has been consumed.
func (l *Lexer) EOF() bool { return l.rest == "" }

// PeekByte returns the next byte, or -1 at the end.
func (l *Lexer) PeekByte() int {
	if l.rest != "" {
		return int(l.rest[0])
	}
	return -1
}

// Skip skips the next n bytes.
func (l *Lexer) Skip(n int) {
	l.rest = l.rest[n:]
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
	for i < len(rest) && (rest[i] == ' ' || rest[i] == '\t') {
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
	for i < len(rest) && fn(rest[i]) {
		i++
	}
	if i != 0 {
		l.rest = rest[i:]
	}
	return rest[:i]
}

// NextByteSet chops off and returns the first byte if the set contains it, otherwise -1.
func (l *Lexer) NextByteSet(bytes *ByteSet) int {
	rest := l.rest
	if 0 < len(rest) && bytes.bits[rest[0]/64]&(1<<(rest[0]%64)) != 0 {
		l.rest = rest[1:]
		return int(rest[0])
	}
	return -1
}

// NextBytesSet chops off the longest prefix consisting solely of bytes
// from the given set.
func (l *Lexer) NextBytesSet(bytes *ByteSet) string {
	i := 0
	rest := l.rest
	for i < len(rest) && bytes.bits[rest[i]/64]&(1<<(rest[i]%64)) != 0 {
		i++
	}
	if i != 0 {
		l.rest = rest[i:]
	}
	return rest[:i]
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

// Since returns the text between the given mark and the current position.
func (l *Lexer) Since(mark LexerMark) string {
	return string(mark)[0 : len(mark)-len(l.rest)]
}

// Copy returns a copy of this lexer.
// It can be used to try one path of parsing and then either discard the
// result or commit it back by calling Commit.
func (l *Lexer) Copy() *Lexer { return &Lexer{l.rest} }

// Commit copies the state of the other lexer into this lexer.
// It always returns true so that it can be used in conditions.
func (l *Lexer) Commit(other *Lexer) bool { l.rest = other.rest; return true }

// NewByteSet creates a bit mask out of a string like "0-9A-Za-z_".
// The bit mask can be used with Lexer.NextBytesSet.
func NewByteSet(chars string) *ByteSet {
	var set ByteSet
	i := 0

	for i < len(chars) {
		switch {
		case i+2 < len(chars) && chars[i+1] == '-':
			for ch := uint16(chars[i]); ch <= uint16(chars[i+2]); ch++ {
				set.bits[ch/64] |= 1 << (ch % 64)
			}
			i += 3
		default:
			ch := chars[i]
			set.bits[ch/64] |= 1 << (ch % 64)
			i++
		}
	}
	return &set
}

// Inverse returns a byte set that matches the inverted set of bytes.
func (bs *ByteSet) Inverse() *ByteSet {
	return &ByteSet{[4]uint64{^bs.bits[0], ^bs.bits[1], ^bs.bits[2], ^bs.bits[3]}}
}

// Predefined byte classes for parsing ASCII text.
var (
	Alnum  = NewByteSet("A-Za-z0-9")  // Alphanumerical, without underscore
	AlnumU = NewByteSet("A-Za-z0-9_") // Alphanumerical, including underscore
	Digit  = NewByteSet("0-9")        // The digits zero to nine
	Space  = NewByteSet("\t\n ")      // Tab, newline, space
	Hspace = NewByteSet("\t ")        // Tab, space
)
