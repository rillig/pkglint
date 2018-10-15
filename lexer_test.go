package main

import "gopkg.in/check.v1"

func (s *Suite) Test_NewLexer(c *check.C) {
	lexer := NewLexer("text")

	c.Check(lexer.EOF(), equals, false)
}

func (s *Suite) Test_Lexer_Rest(c *check.C) {
	c.Check(NewLexer("").Rest(), equals, "")
	c.Check(NewLexer("text").Rest(), equals, "text")
}

func (s *Suite) Test_Lexer_EOF(c *check.C) {
	c.Check(NewLexer("").EOF(), equals, true)
	c.Check(NewLexer("text").EOF(), equals, false)

	lexer := NewLexer("text")
	lexer.NextString("text")

	c.Check(lexer.EOF(), equals, true)
}

func (s *Suite) Test_Lexer_Mark__beginning(c *check.C) {
	lexer := NewLexer("text")

	mark := lexer.Mark()
	c.Check(lexer.NextString("text"), equals, true)

	c.Check(lexer.Rest(), equals, "")

	lexer.Reset(mark)

	c.Check(lexer.Rest(), equals, "text")
}

func (s *Suite) Test_Lexer_Mark__middle(c *check.C) {
	lexer := NewLexer("text")
	lexer.NextString("te")

	mark := lexer.Mark()
	c.Check(lexer.NextString("x"), equals, true)

	c.Check(lexer.Rest(), equals, "t")

	lexer.Reset(mark)

	c.Check(lexer.Rest(), equals, "xt")
}
