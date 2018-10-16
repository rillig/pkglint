package textproc

import (
	"gopkg.in/check.v1"
	"testing"
	"unicode"
)

type Suite struct{}

var equals = check.Equals

func Test(t *testing.T) {
	check.Suite(Suite{})
	check.TestingT(t)
}

func (s *Suite) Test_NewLexer(c *check.C) {
	lexer := NewLexer("text")

	c.Check(lexer.EOF(), equals, false)
}

func (s *Suite) Test_Lexer_Rest__end(c *check.C) {
	lexer := NewLexer("")

	c.Check(lexer.Rest(), equals, "")
}

func (s *Suite) Test_Lexer_Rest__middle(c *check.C) {
	lexer := NewLexer("text")

	c.Check(lexer.Rest(), equals, "text")
}

func (s *Suite) Test_Lexer_EOF__end(c *check.C) {
	lexer := NewLexer("")

	c.Check(lexer.EOF(), equals, true)
}

func (s *Suite) Test_Lexer_EOF__middle(c *check.C) {
	lexer := NewLexer("text")

	c.Check(lexer.EOF(), equals, false)
}

func (s *Suite) Test_Lexer_EOF__reached(c *check.C) {
	lexer := NewLexer("text")
	lexer.NextString("text")

	c.Check(lexer.EOF(), equals, true)
}

func (s *Suite) Test_Lexer_Mark__beginning(c *check.C) {
	lexer := NewLexer("text")

	mark := lexer.Mark()
	c.Check(lexer.NextString("text"), equals, "text")

	c.Check(lexer.Rest(), equals, "")

	lexer.Reset(mark)

	c.Check(lexer.Rest(), equals, "text")
}

func (s *Suite) Test_Lexer_Mark__middle(c *check.C) {
	lexer := NewLexer("text")
	lexer.NextString("te")

	mark := lexer.Mark()
	c.Check(lexer.NextString("x"), equals, "x")

	c.Check(lexer.Rest(), equals, "t")

	lexer.Reset(mark)

	c.Check(lexer.Rest(), equals, "xt")
}

// Demonstrates that multiple marks can be taken at the same time and that
// the lexer can be reset to any of them in any order.
func (s *Suite) Test_Lexer_Reset__multiple(c *check.C) {
	lexer := NewLexer("text")

	mark0 := lexer.Mark()
	c.Check(lexer.NextString("te"), equals, "te")
	mark2 := lexer.Mark()
	c.Check(lexer.NextString("x"), equals, "x")
	mark3 := lexer.Mark()
	c.Check(lexer.NextString("t"), equals, "t")
	mark4 := lexer.Mark()

	c.Check(lexer.Rest(), equals, "")

	lexer.Reset(mark0)

	c.Check(lexer.Rest(), equals, "text")

	lexer.Reset(mark3)

	c.Check(lexer.Rest(), equals, "t")

	lexer.Reset(mark2)

	c.Check(lexer.Rest(), equals, "xt")

	lexer.Reset(mark4)

	c.Check(lexer.Rest(), equals, "")
}

func (s *Suite) Test_Lexer_NextString(c *check.C) {
	lexer := NewLexer("text")

	c.Check(lexer.NextString("te"), equals, "te")
	c.Check(lexer.NextString("st"), equals, "") // Did not match.
	c.Check(lexer.NextString("xt"), equals, "xt")
}

func (s *Suite) Test_Lexer_NextHspace(c *check.C) {
	lexer := NewLexer("spaces   \t \t  and tabs\n")

	c.Check(lexer.NextString("spaces"), equals, "spaces")
	c.Check(lexer.NextHspace(), equals, "   \t \t  ")
	c.Check(lexer.NextHspace(), equals, "") // No space left.
	c.Check(lexer.NextString("and tabs"), equals, "and tabs")
	c.Check(lexer.NextHspace(), equals, "") // Newline is not a horizontal space.
}

func (s *Suite) Test_Lexer_NextByte(c *check.C) {
	lexer := NewLexer("byte")

	c.Check(lexer.NextByte('b'), equals, true)
	c.Check(lexer.NextByte('b'), equals, false) // The b is already chopped off.
	c.Check(lexer.NextByte('y'), equals, true)
	c.Check(lexer.NextByte('t'), equals, true)
	c.Check(lexer.NextByte('e'), equals, true)
	c.Check(lexer.NextByte(0), equals, false) // This is not a C string.
}

func (s *Suite) Test_Lexer_NextBytesFunc(c *check.C) {
	lexer := NewLexer("an alphanumerical string")

	c.Check(lexer.NextBytesFunc(func(b byte) bool { return 'A' <= b && b <= 'Z' }), equals, "")
	c.Check(lexer.NextBytesFunc(func(b byte) bool { return !unicode.IsSpace(rune(b)) }), equals, "an")
	c.Check(lexer.NextHspace(), equals, " ")
	c.Check(lexer.NextBytesFunc(func(b byte) bool { return 'a' <= b && b <= 'z' }), equals, "alphanumerical")
	c.Check(lexer.NextBytesFunc(func(b byte) bool { return true }), equals, " string")
}
