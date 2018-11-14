package textproc

import (
	"gopkg.in/check.v1"
	"netbsd.org/pkglint/intqa"
	"regexp"
	"testing"
	"unicode"
)

type Suite struct{}

var equals = check.Equals

func Test(t *testing.T) {
	check.Suite(new(Suite))
	check.TestingT(t)
}

func (s *Suite) Test_NewLexer(c *check.C) {
	lexer := NewLexer("text")

	c.Check(lexer.rest, equals, "text")
}

func (s *Suite) Test_Lexer_Rest__initial(c *check.C) {
	lexer := NewLexer("text")

	c.Check(lexer.Rest(), equals, "text")
}

func (s *Suite) Test_Lexer_Rest__end(c *check.C) {
	lexer := NewLexer("")

	c.Check(lexer.Rest(), equals, "")
}

func (s *Suite) Test_Lexer_EOF__initial(c *check.C) {
	lexer := NewLexer("text")

	c.Check(lexer.EOF(), equals, false)
}

func (s *Suite) Test_Lexer_EOF__end(c *check.C) {
	lexer := NewLexer("")

	c.Check(lexer.EOF(), equals, true)
}

func (s *Suite) Test_Lexer_PeekByte(c *check.C) {
	lexer := NewLexer("text")

	c.Check(lexer.PeekByte(), equals, int('t'))

	c.Check(lexer.NextString("text"), equals, "text")

	c.Check(lexer.PeekByte(), equals, -1)
}

func (s *Suite) Test_Lexer_Skip(c *check.C) {
	lexer := NewLexer("example text")

	c.Check(lexer.Skip(7), equals, true)

	c.Check(lexer.Rest(), equals, " text")

	c.Check(lexer.Skip(0), equals, false)

	c.Check(lexer.Rest(), equals, " text")

	// Skipping a fixed number of bytes only makes sense when the
	// lexer has examined every one of them before. Therefore no
	// extra check is done here, and panicking here is intentional.
	c.Check(
		func() { lexer.Skip(6) },
		check.PanicMatches,
		`^runtime error: slice bounds out of range$`)
}

func (s *Suite) Test_Lexer_NextString(c *check.C) {
	lexer := NewLexer("text")

	c.Check(lexer.NextString("te"), equals, "te")
	c.Check(lexer.NextString("st"), equals, "") // Did not match.
	c.Check(lexer.NextString("xt"), equals, "xt")
}

func (s *Suite) Test_Lexer_SkipString(c *check.C) {
	lexer := NewLexer("text")

	c.Check(lexer.SkipString("te"), equals, true)
	c.Check(lexer.SkipString("st"), equals, false)
	c.Check(lexer.SkipString("xt"), equals, true)
}

func (s *Suite) Test_Lexer_SkipHspace(c *check.C) {
	lexer := NewLexer("spaces   \t \t  and tabs\n")

	c.Check(lexer.NextString("spaces"), equals, "spaces")
	c.Check(lexer.SkipHspace(), equals, true)
	c.Check(lexer.Rest(), equals, "and tabs\n")
	c.Check(lexer.SkipHspace(), equals, false) // No space left.
	c.Check(lexer.NextString("and tabs"), equals, "and tabs")
	c.Check(lexer.SkipHspace(), equals, false) // Newline is not a horizontal space.
	c.Check(lexer.Rest(), equals, "\n")
}

func (s *Suite) Test_Lexer_NextHspace(c *check.C) {
	lexer := NewLexer("spaces   \t \t  and tabs\n")

	c.Check(lexer.NextString("spaces"), equals, "spaces")
	c.Check(lexer.NextHspace(), equals, "   \t \t  ")
	c.Check(lexer.NextHspace(), equals, "") // No space left.
	c.Check(lexer.NextString("and tabs"), equals, "and tabs")
	c.Check(lexer.NextHspace(), equals, "") // Newline is not a horizontal space.
}

func (s *Suite) Test_Lexer_SkipByte(c *check.C) {
	lexer := NewLexer("byte")

	c.Check(lexer.SkipByte('b'), equals, true)
	c.Check(lexer.SkipByte('b'), equals, false) // The b is already chopped off.
	c.Check(lexer.SkipByte('y'), equals, true)
	c.Check(lexer.SkipByte('t'), equals, true)
	c.Check(lexer.SkipByte('e'), equals, true)
	c.Check(lexer.SkipByte(0), equals, false) // This is not a C string.
}

func (s *Suite) Test_Lexer_NextBytesFunc(c *check.C) {
	lexer := NewLexer("an alphanumerical string")

	c.Check(lexer.NextBytesFunc(func(b byte) bool { return 'A' <= b && b <= 'Z' }), equals, "")
	c.Check(lexer.NextBytesFunc(func(b byte) bool { return !unicode.IsSpace(rune(b)) }), equals, "an")
	c.Check(lexer.NextHspace(), equals, " ")
	c.Check(lexer.NextBytesFunc(func(b byte) bool { return 'a' <= b && b <= 'z' }), equals, "alphanumerical")
	c.Check(lexer.NextBytesFunc(func(b byte) bool { return true }), equals, " string")
}

func (s *Suite) Test_Lexer_NextByteSet(c *check.C) {
	lexer := NewLexer("an a\n")

	c.Check(lexer.NextByteSet(Alnum), equals, int('a'))
	c.Check(lexer.NextByteSet(Alnum), equals, int('n'))
	c.Check(lexer.NextByteSet(Space), equals, int(' '))
	c.Check(lexer.NextByteSet(Alnum), equals, int('a'))
	c.Check(lexer.NextByteSet(Space), equals, int('\n'))
	c.Check(lexer.NextByteSet(Alnum), equals, -1)
}

func (s *Suite) Test_Lexer_NextBytesSet(c *check.C) {
	lexer := NewLexer("an alphanumerical 90_ \tstring\t\t \n")

	c.Check(lexer.NextBytesSet(Alnum), equals, "an")
	c.Check(lexer.NextBytesSet(Alnum), equals, "")
	c.Check(lexer.NextBytesSet(Space), equals, " ")
	c.Check(lexer.NextBytesSet(Alnum), equals, "alphanumerical")
	c.Check(lexer.NextBytesSet(Space), equals, " ")
	c.Check(lexer.NextBytesSet(AlnumU), equals, "90_")
	c.Check(lexer.NextBytesSet(Space), equals, " \t")
	c.Check(lexer.NextBytesSet(Alnum), equals, "string")
	c.Check(lexer.NextBytesSet(Hspace), equals, "\t\t ")
	c.Check(lexer.NextBytesSet(Space), equals, "\n")
}

func (s *Suite) Test_Lexer_SkipRegexp(c *check.C) {
	lexer := NewLexer("an alphanumerical 90_ \tstring\t\t \n")

	c.Check(lexer.SkipRegexp(regexp.MustCompile(`^\w+`)), equals, true)
	c.Check(lexer.Rest(), equals, " alphanumerical 90_ \tstring\t\t \n")
	c.Check(lexer.SkipRegexp(regexp.MustCompile(`^.+`)), equals, true)
	c.Check(lexer.Rest(), equals, "\n")
	c.Check(lexer.SkipRegexp(regexp.MustCompile(`^.+`)), equals, false)
	// This call returns false since the matched string was empty.
	c.Check(lexer.SkipRegexp(regexp.MustCompile(`^.*`)), equals, false)
	c.Check(lexer.Rest(), equals, "\n")
}

func (s *Suite) Test_Lexer_NextRegexp(c *check.C) {
	lexer := NewLexer("an alphanumerical 90_ \tstring\t\t \n")

	c.Check(lexer.NextRegexp(regexp.MustCompile(`^\w+`))[0], equals, "an")
	c.Check(lexer.NextRegexp(regexp.MustCompile(`^[\w ]+`))[0], equals, " alphanumerical 90_ ")
	c.Check(lexer.NextRegexp(regexp.MustCompile(`^.+`))[0], equals, "\tstring\t\t ")
	c.Check(lexer.NextRegexp(regexp.MustCompile(`^.+`)), check.IsNil)
	c.Check(lexer.NextRegexp(regexp.MustCompile(`^.*`))[0], equals, "")
	c.Check(lexer.Rest(), equals, "\n")
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

func (s *Suite) Test_Lexer__NextString_then_EOF(c *check.C) {
	lexer := NewLexer("text")
	lexer.NextString("text")

	c.Check(lexer.EOF(), equals, true)
}

func (s *Suite) Test_Lexer_Since(c *check.C) {
	lexer := NewLexer("text")
	mark := lexer.Mark()

	c.Check(lexer.NextString("te"), equals, "te")
	c.Check(lexer.NextString("st"), equals, "") // Did not match.
	c.Check(lexer.Since(mark), equals, "te")
	c.Check(lexer.NextString("xt"), equals, "xt")
	c.Check(lexer.Since(mark), equals, "text")
}

func (s *Suite) Test_Lexer_Copy(c *check.C) {
	lexer := NewLexer("text")
	copied := lexer.Copy()

	c.Check(copied.Rest(), equals, lexer.Rest())

	copied.NextString("te")

	c.Check(copied.Rest(), equals, "xt")
	c.Check(lexer.Rest(), equals, "text") // The original is not yet affected.
}

func (s *Suite) Test_Lexer_Commit(c *check.C) {
	lexer := NewLexer("text")
	copied := lexer.Copy()
	copied.NextString("te")

	c.Check(lexer.Rest(), equals, "text") // The original is not yet affected.

	lexer.Commit(copied)

	c.Check(lexer.Rest(), equals, "xt")
}

func (s *Suite) Test_NewByteSet(c *check.C) {
	set := NewByteSet("A-Za-z0-9_\xFC")

	c.Check(set.bits, equals, [4]uint64{
		0x03ff000000000000, // 9-0
		0x07fffffe87fffffe, // z-a _ Z-A
		0x0000000000000000,
		0x1000000000000000}) // \xFC
}

// Ensures that the bit manipulations work when a range spans
// multiple of the uint64 words.
func (s *Suite) Test_NewByteSet__large_range(c *check.C) {
	set := NewByteSet("\x01-\xFE")

	c.Check(set.bits, equals, [4]uint64{
		0xfffffffffffffffe,
		0xffffffffffffffff,
		0xffffffffffffffff,
		0x7fffffffffffffff})
}

// Demonstrates how to specify a byte set that includes a hyphen,
// since that is also used for byte ranges.
// The hyphen must be written as ---, and it must be at the beginning.
func (s *Suite) Test_NewByteSet__range_hyphen(c *check.C) {
	set := NewByteSet("---a-z")

	c.Check(set.bits, equals, [4]uint64{
		0x0000200000000000,
		0x07fffffe00000000,
		0x0000000000000000,
		0x0000000000000000})
}

func (s *Suite) Test_ByteSet_Inverse(c *check.C) {
	set := NewByteSet("A-Za-z0-9_\xFC")
	inverse := set.Inverse()

	c.Check(inverse.bits, equals, [4]uint64{
		0xfc00ffffffffffff,
		0xf800000178000001,
		0xffffffffffffffff,
		0xefffffffffffffff})

	c.Check(inverse.Inverse().bits, equals, set.bits)
}

func (s *Suite) Test_ByteSet_Contains(c *check.C) {
	set := NewByteSet("A-Za-z0-9_\xFC")

	c.Check(set.Contains(0x00), equals, false)
	c.Check(set.Contains('-'), equals, false)
	c.Check(set.Contains('A'), equals, true)
	c.Check(set.Contains('Z'), equals, true)
	c.Check(set.Contains('['), equals, false)
	c.Check(set.Contains(0xFC), equals, true)
	c.Check(set.Contains(0xFD), equals, false)
}

func (s *Suite) Test__test_names(c *check.C) {
	ck := intqa.NewTestNameChecker(c)
	ck.AllowCamelCaseDescriptions(
		"NextString_then_EOF")
	ck.ShowWarnings(false)
	ck.Check()
}
