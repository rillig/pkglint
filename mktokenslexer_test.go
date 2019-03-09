package pkglint

import (
	"gopkg.in/check.v1"
	"netbsd.org/pkglint/textproc"
)

func (s *Suite) Test_MkTokensLexer__empty_slice_returns_EOF(c *check.C) {
	t := s.Init(c)

	lexer := NewMkTokensLexer(nil)

	t.Check(lexer.EOF(), equals, true)
}

// A slice of a single token behaves like textproc.Lexer.
func (s *Suite) Test_MkTokensLexer__single_plain_text_token(c *check.C) {
	t := s.Init(c)

	lexer := NewMkTokensLexer([]*MkToken{{"\\# $$ [#] $V", nil}})

	t.Check(lexer.SkipByte('\\'), equals, true)
	t.Check(lexer.Rest(), equals, "# $$ [#] $V")
	t.Check(lexer.SkipByte('#'), equals, true)
	t.Check(lexer.NextHspace(), equals, " ")
	t.Check(lexer.NextBytesSet(textproc.Space.Inverse()), equals, "$$")
	t.Check(lexer.Skip(len(lexer.Rest())), equals, true)
	t.Check(lexer.EOF(), equals, true)
}

// If the first element of the slice is a variable use, none of the plain
// text patterns matches.
//
// The code that uses the MkTokensLexer needs to distinguish these cases
// anyway, therefore it doesn't make sense to treat variable uses as plain
// text.
func (s *Suite) Test_MkTokensLexer__single_varuse_token(c *check.C) {
	t := s.Init(c)

	lexer := NewMkTokensLexer([]*MkToken{{"${VAR:Mpattern}", NewMkVarUse("VAR", "Mpattern")}})

	t.Check(lexer.EOF(), equals, false)
	t.Check(lexer.PeekByte(), equals, -1)
	t.Check(lexer.NextVarUse(), deepEquals, NewMkVarUse("VAR", "Mpattern"))
}

func (s *Suite) Test_MkTokensLexer__plain_then_varuse(c *check.C) {
	t := s.Init(c)

	lexer := NewMkTokensLexer([]*MkToken{
		{"plain text", nil},
		{"${VAR:Mpattern}", NewMkVarUse("VAR", "Mpattern")}})

	t.Check(lexer.NextBytesSet(textproc.Digit.Inverse()), equals, "plain text")
	t.Check(lexer.NextVarUse(), deepEquals, NewMkVarUse("VAR", "Mpattern"))
	t.Check(lexer.EOF(), equals, true)
}

func (s *Suite) Test_MkTokensLexer__varuse_varuse_varuse(c *check.C) {
	t := s.Init(c)

	lexer := NewMkTokensLexer([]*MkToken{
		{"${dirs:O:u}", NewMkVarUse("dirs", "O", "u")},
		{"${VAR:Mpattern}", NewMkVarUse("VAR", "Mpattern")},
		{"${.TARGET}", NewMkVarUse(".TARGET")}})

	t.Check(lexer.NextVarUse(), deepEquals, NewMkVarUse("dirs", "O", "u"))
	t.Check(lexer.NextVarUse(), deepEquals, NewMkVarUse("VAR", "Mpattern"))
	t.Check(lexer.NextVarUse(), deepEquals, NewMkVarUse(".TARGET"))
}

func (s *Suite) Test_MkTokensLexer__mark_reset_since_in_initial_state(c *check.C) {
	t := s.Init(c)

	lexer := NewMkTokensLexer([]*MkToken{
		{"${dirs:O:u}", NewMkVarUse("dirs", "O", "u")},
		{"${VAR:Mpattern}", NewMkVarUse("VAR", "Mpattern")},
		{"${.TARGET}", NewMkVarUse(".TARGET")}})

	start := lexer.Mark()
	t.Check(lexer.NextVarUse(), deepEquals, NewMkVarUse("dirs", "O", "u"))
	middle := lexer.Mark()
	t.Check(lexer.Rest(), equals, "${VAR:Mpattern}${.TARGET}")
	lexer.Reset(start)
	t.Check(lexer.Rest(), equals, "${dirs:O:u}${VAR:Mpattern}${.TARGET}")
	lexer.Reset(middle)
	t.Check(lexer.Rest(), equals, "${VAR:Mpattern}${.TARGET}")
}

func (s *Suite) Test_MkTokensLexer__mark_reset_since_inside_plain_text(c *check.C) {
	t := s.Init(c)

	lexer := NewMkTokensLexer([]*MkToken{
		{"plain text", nil},
		{"${VAR:Mpattern}", NewMkVarUse("VAR", "Mpattern")},
		{"rest", nil}})

	start := lexer.Mark()
	t.Check(lexer.NextBytesSet(textproc.Alpha), equals, "plain")
	middle := lexer.Mark()
	t.Check(lexer.Rest(), equals, " text${VAR:Mpattern}rest")
	lexer.Reset(start)
	t.Check(lexer.Rest(), equals, "plain text${VAR:Mpattern}rest")
	lexer.Reset(middle)
	t.Check(lexer.Rest(), equals, " text${VAR:Mpattern}rest")
}

func (s *Suite) Test_MkTokensLexer__mark_reset_since_after_plain_text(c *check.C) {
	t := s.Init(c)

	lexer := NewMkTokensLexer([]*MkToken{
		{"plain text", nil},
		{"${VAR:Mpattern}", NewMkVarUse("VAR", "Mpattern")},
		{"rest", nil}})

	start := lexer.Mark()
	t.Check(lexer.SkipString("plain text"), equals, true)
	end := lexer.Mark()
	t.Check(lexer.Rest(), equals, "${VAR:Mpattern}rest")
	lexer.Reset(start)
	t.Check(lexer.Rest(), equals, "plain text${VAR:Mpattern}rest")
	lexer.Reset(end)
	t.Check(lexer.Rest(), equals, "${VAR:Mpattern}rest")
}

func (s *Suite) Test_MkTokensLexer__mark_reset_since_after_varuse(c *check.C) {
	t := s.Init(c)

	lexer := NewMkTokensLexer([]*MkToken{
		{"${VAR:Mpattern}", NewMkVarUse("VAR", "Mpattern")},
		{"rest", nil}})

	start := lexer.Mark()
	t.Check(lexer.NextVarUse(), deepEquals, NewMkVarUse("VAR", "Mpattern"))
	end := lexer.Mark()
	t.Check(lexer.Rest(), equals, "rest")
	lexer.Reset(start)
	t.Check(lexer.Rest(), equals, "${VAR:Mpattern}rest")
	lexer.Reset(end)
	t.Check(lexer.Rest(), equals, "rest")
}

func (s *Suite) Test_MkTokensLexer__multiple_marks_in_same_plain_text(c *check.C) {
	t := s.Init(c)

	lexer := NewMkTokensLexer([]*MkToken{
		{"plain text", nil},
		{"${VAR:Mpattern}", NewMkVarUse("VAR", "Mpattern")},
		{"rest", nil}})

	start := lexer.Mark()
	t.Check(lexer.NextString("plain "), equals, "plain ")
	middle := lexer.Mark()
	t.Check(lexer.NextString("text"), equals, "text")
	end := lexer.Mark()
	t.Check(lexer.Rest(), equals, "${VAR:Mpattern}rest")
	lexer.Reset(start)
	t.Check(lexer.Rest(), equals, "plain text${VAR:Mpattern}rest")
	lexer.Reset(middle)
	t.Check(lexer.Rest(), equals, "text${VAR:Mpattern}rest")
	lexer.Reset(end)
	t.Check(lexer.Rest(), equals, "${VAR:Mpattern}rest")
}

func (s *Suite) Test_MkTokensLexer__multiple_marks_in_varuse(c *check.C) {
	t := s.Init(c)

	lexer := NewMkTokensLexer([]*MkToken{
		{"${VAR1}", NewMkVarUse("VAR1")},
		{"${VAR2}", NewMkVarUse("VAR2")},
		{"${VAR3}", NewMkVarUse("VAR3")}})

	start := lexer.Mark()
	t.Check(lexer.NextVarUse(), deepEquals, NewMkVarUse("VAR1"))
	middle := lexer.Mark()
	t.Check(lexer.NextVarUse(), deepEquals, NewMkVarUse("VAR2"))
	further := lexer.Mark()
	t.Check(lexer.NextVarUse(), deepEquals, NewMkVarUse("VAR3"))
	end := lexer.Mark()
	t.Check(lexer.Rest(), equals, "")
	lexer.Reset(middle)
	t.Check(lexer.Rest(), equals, "${VAR2}${VAR3}")
	lexer.Reset(further)
	t.Check(lexer.Rest(), equals, "${VAR3}")
	lexer.Reset(start)
	t.Check(lexer.Rest(), equals, "${VAR1}${VAR2}${VAR3}")
	lexer.Reset(end)
	t.Check(lexer.Rest(), equals, "")
}

func (s *Suite) Test_MkTokensLexer__EOF_before_plain_text(c *check.C) {
	t := s.Init(c)

	lexer := NewMkTokensLexer([]*MkToken{{"rest", nil}})

	t.Check(lexer.EOF(), equals, false)
}

func (s *Suite) Test_MkTokensLexer__EOF_before_varuse(c *check.C) {
	t := s.Init(c)

	lexer := NewMkTokensLexer([]*MkToken{{"${VAR}", NewMkVarUse("VAR")}})

	t.Check(lexer.EOF(), equals, false)
}

// When the MkTokensLexer is constructed, it gets a copy of the tokens array.
// In theory it would be possible to change the tokens after starting lexing,
// but there is no practical case where that would be useful.
//
// Since each slice is a separate view on the underlying array, modifying the
// size of the outside slice does not affect parsing. This is also only a
// theoretical case.
//
// Because all these cases are only theoretical, the MkTokensLexer doesn't
// bother to make this unnecessary copy and works on the shared slice.
func (s *Suite) Test_MkTokensLexer__constructor_uses_shared_array(c *check.C) {
	t := s.Init(c)

	tokens := []*MkToken{{"${VAR}", NewMkVarUse("VAR")}}
	lexer := NewMkTokensLexer(tokens)

	t.Check(lexer.Rest(), equals, "${VAR}")

	tokens[0].Text = "modified text"
	tokens[0].Varuse = NewMkVarUse("MODIFIED", "Mpattern")
	tokens = tokens[0:0]

	t.Check(lexer.Rest(), equals, "modified text")
}

// TODO: even without the append() calls the lexer and the marks are independent of each other

func (s *Suite) Test_MkTokensLexer__peek_after_varuse(c *check.C) {
	t := s.Init(c)

	lexer := NewMkTokensLexer([]*MkToken{
		{"${VAR}", NewMkVarUse("VAR")},
		{"${VAR}", NewMkVarUse("VAR")},
		{"text", nil}})

	t.Check(lexer.NextVarUse(), deepEquals, NewMkVarUse("VAR"))
	t.Check(lexer.PeekByte(), equals, -1)

	t.Check(lexer.NextVarUse(), deepEquals, NewMkVarUse("VAR"))
	t.Check(lexer.PeekByte(), equals, int('t'))
}

func (s *Suite) Test_MkTokensLexer__varuse_when_plain_text(c *check.C) {
	t := s.Init(c)

	lexer := NewMkTokensLexer([]*MkToken{{"text", nil}})

	t.Check(lexer.NextVarUse(), check.IsNil)
	t.Check(lexer.NextString("te"), equals, "te")
	t.Check(lexer.NextVarUse(), check.IsNil)
	t.Check(lexer.NextString("xt"), equals, "xt")
	t.Check(lexer.NextVarUse(), check.IsNil)
}
