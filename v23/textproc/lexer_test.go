package textproc

import (
	"github.com/rillig/pkglint/v23/intqa"
	"regexp"
	"strings"
	"testing"
	"unicode"
)

func Test_NewLexer(t *testing.T) {
	lexer := NewLexer("text")

	if have, want := lexer.rest, "text"; have != want {
		t.Errorf("want %#v, have %#v", want, have)
	}
}

func Test_Lexer_Rest__initial(t *testing.T) {
	lexer := NewLexer("text")

	if have, want := lexer.Rest(), "text"; have != want {
		t.Errorf("want %#v, have %#v", want, have)
	}
}

func Test_Lexer_Rest__end(t *testing.T) {
	lexer := NewLexer("")

	if have, want := lexer.rest, ""; have != want {
		t.Errorf("want %#v, have %#v", want, have)
	}
}

func Test_Lexer_EOF__initial(t *testing.T) {
	lexer := NewLexer("text")

	if lexer.EOF() {
		t.Errorf("unfinished lexer is at EOF")
	}
}

func Test_Lexer_EOF__end(t *testing.T) {
	lexer := NewLexer("")

	if !lexer.EOF() {
		t.Errorf("empty lexer is not at EOF")
	}
}

func Test_Lexer_PeekByte(t *testing.T) {
	lexer := NewLexer("text")

	if have, want := lexer.PeekByte(), int('t'); have != want {
		t.Errorf("want %#v, have %#v", want, have)
	}

	if have, want := lexer.NextString("text"), "text"; have != want {
		t.Errorf("want %#v, have %#v", want, have)
	}

	if have, want := lexer.PeekByte(), -1; have != want {
		t.Errorf("want %#v, have %#v", want, have)
	}
}

func Test_Lexer_TestByteSet(t *testing.T) {
	lexer := NewLexer("text")

	if lexer.TestByteSet(Upper) {
		t.Errorf("upper")
	}
	if !lexer.TestByteSet(Lower) {
		t.Errorf("lower")
	}
	if !lexer.TestByteSet(NewByteSet("t")) {
		t.Errorf("t")
	}

	if have, want := lexer.NextString("text"), "text"; have != want {
		t.Errorf("want %#v, have %#v", want, have)
	}

	if lexer.TestByteSet(Upper) {
		t.Errorf("upper")
	}
	if lexer.TestByteSet(Lower) {
		t.Errorf("lower")
	}
}

func Test_Lexer_Skip(t *testing.T) {
	lexer := NewLexer("example text")

	if !lexer.Skip(7) {
		t.Error("skip 7")
	}

	if have := lexer.Rest(); have != " text" {
		t.Errorf("rest %#v", have)
	}

	if lexer.Skip(0) {
		t.Error("skip 0")
	}

	if have := lexer.Rest(); have != " text" {
		t.Errorf("rest %#v", have)
	}

	// Skipping a fixed number of bytes only makes sense when the
	// lexer has examined every one of them before. Therefore, no
	// extra check is done here, and panicking here is intentional.
	var p error
	func() {
		defer func() {
			p = recover().(error)
		}()
		lexer.Skip(6)
	}()
	if have := p.Error(); !regexp.MustCompile(`runtime error: slice bounds out of range.*`).MatchString(have) {
		t.Errorf("%#v", have)
	}
}

func Test_Lexer_NextString(t *testing.T) {
	lexer := NewLexer("text")

	if have, want := lexer.NextString("te"), "te"; have != want {
		t.Errorf("want %#v, have %#v", want, have)
	}
	if have, want := lexer.NextString("st") /* Does not match */, ""; have != want {
		t.Errorf("want %#v, have %#v", want, have)
	}
	if have, want := lexer.NextString("xt"), "xt"; have != want {
		t.Errorf("want %#v, have %#v", want, have)
	}
}

func Test_Lexer_NextString__EOF(t *testing.T) {
	lexer := NewLexer("text")
	lexer.NextString("text")

	if !lexer.EOF() {
		t.Error("want EOF")
	}
}

func Test_Lexer_SkipString(t *testing.T) {
	lexer := NewLexer("text")

	if !lexer.SkipString("te") {
		t.Error("want true")
	}
	if lexer.SkipString("st") {
		t.Error("want false")
	}
	if !lexer.SkipString("xt") {
		t.Error("want true")
	}
}

func Test_Lexer_SkipHspace(t *testing.T) {
	lexer := NewLexer("spaces   \t \t  and tabs\n\t ")

	if have, want := lexer.NextString("spaces"), "spaces"; have != want {
		t.Errorf("want %#v, have %#v", want, have)
	}
	if !lexer.SkipHspace() {
		t.Error("want true")
	}
	if have, want := lexer.Rest(), "and tabs\n\t "; have != want {
		t.Errorf("want %#v, have %#v", want, have)
	}
	if have := lexer.SkipHspace(); have { // No space left.
		t.Error("want false")
	}
	if have, want := lexer.NextString("and tabs"), "and tabs"; have != want {
		t.Errorf("want %#v, have %#v", want, have)
	}
	if have := lexer.SkipHspace(); have { // Newline is not a horizontal space.
		t.Error("want false")
	}
	if have, want := lexer.NextString("\n"), "\n"; have != want {
		t.Errorf("want %#v, have %#v", want, have)
	}
	if !lexer.SkipHspace() {
		t.Error("want true")
	}
}

func Test_Lexer_NextHspace(t *testing.T) {
	lexer := NewLexer("spaces   \t \t  and tabs\n\t ")

	if have, want := lexer.NextString("spaces"), "spaces"; have != want {
		t.Errorf("want %#v, have %#v", want, have)
	}
	if have, want := lexer.NextHspace(), "   \t \t  "; have != want {
		t.Errorf("want %#v, have %#v", want, have)
	}
	if have, want := lexer.NextHspace(), ""; have != want { // No space left.
		t.Errorf("want %#v, have %#v", want, have)
	}
	if have, want := lexer.NextString("and tabs"), "and tabs"; have != want {
		t.Errorf("want %#v, have %#v", want, have)
	}
	if have, want := lexer.NextHspace(), ""; have != want { // Newline is not a horizontal space.
		t.Errorf("want %#v, have %#v", want, have)
	}
	if have, want := lexer.NextString("\n"), "\n"; have != want {
		t.Errorf("want %#v, have %#v", want, have)
	}
	if have, want := lexer.NextHspace(), "\t "; have != want {
		t.Errorf("want %#v, have %#v", want, have)
	}
}

func Test_Lexer_SkipByte(t *testing.T) {
	lexer := NewLexer("byte")

	if !lexer.SkipByte('b') {
		t.Error("want true")
	}
	if have := lexer.SkipByte('b'); have { // The b is already chopped off.
		t.Error("want false")
	}
	if !lexer.SkipByte('y') {
		t.Error("want true")
	}
	if !lexer.SkipByte('t') {
		t.Error("want true")
	}
	if !lexer.SkipByte('e') {
		t.Error("want true")
	}
	if have := lexer.SkipByte(0); have { // This is not a C string.
		t.Error("want false")
	}
}

func Test_Lexer_NextByte(t *testing.T) {
	lexer := NewLexer("byte")

	if have, want := lexer.NextByte(), byte('b'); have != want {
		t.Errorf("want %#v, have %#v", want, have)
	}
	if have, want := lexer.NextByte(), byte('y'); have != want {
		t.Errorf("want %#v, have %#v", want, have)
	}
	if have, want := lexer.NextByte(), byte('t'); have != want {
		t.Errorf("want %#v, have %#v", want, have)
	}
	if have, want := lexer.NextByte(), byte('e'); have != want {
		t.Errorf("want %#v, have %#v", want, have)
	}

	var p error
	func() {
		defer func() {
			p = recover().(error)
		}()
		lexer.NextByte()
	}()
	if !regexp.MustCompile(`^runtime error: index out of range.*`).MatchString(p.Error()) {
		t.Error(p)
	}
}

func Test_Lexer_SkipBytesFunc(t *testing.T) {
	lexer := NewLexer("an alphanumerical string")

	if lexer.SkipBytesFunc(func(b byte) bool { return 'A' <= b && b <= 'Z' }) {
		t.Error("want false")
	}
	if !lexer.SkipBytesFunc(func(b byte) bool { return !unicode.IsSpace(rune(b)) }) {
		t.Error("want true")
	}
	if !lexer.SkipHspace() {
		t.Error("want true")
	}
	if !lexer.SkipBytesFunc(func(b byte) bool { return 'a' <= b && b <= 'z' }) {
		t.Error("want true")
	}
	if !lexer.SkipBytesFunc(func(b byte) bool { return true }) {
		t.Error("want true")
	}
}

func Test_Lexer_NextBytesFunc(t *testing.T) {
	lexer := NewLexer("an alphanumerical string")

	if have, want := lexer.NextBytesFunc(func(b byte) bool { return 'A' <= b && b <= 'Z' }), ""; have != want {
		t.Errorf("want %#v, have %#v", want, have)
	}
	if have, want := lexer.NextBytesFunc(func(b byte) bool { return !unicode.IsSpace(rune(b)) }), "an"; have != want {
		t.Errorf("want %#v, have %#v", want, have)
	}
	if have, want := lexer.NextHspace(), " "; have != want {
		t.Errorf("want %#v, have %#v", want, have)
	}
	if have, want := lexer.NextBytesFunc(func(b byte) bool { return 'a' <= b && b <= 'z' }), "alphanumerical"; have != want {
		t.Errorf("want %#v, have %#v", want, have)
	}
	if have, want := lexer.NextBytesFunc(func(b byte) bool { return true }), " string"; have != want {
		t.Errorf("want %#v, have %#v", want, have)
	}
}

func Test_Lexer_NextByteSet(t *testing.T) {
	lexer := NewLexer("an a\n")

	if have, want := lexer.NextByteSet(Alnum), int('a'); have != want {
		t.Errorf("want %#v, have %#v", want, have)
	}
	if have, want := lexer.NextByteSet(Alnum), int('n'); have != want {
		t.Errorf("want %#v, have %#v", want, have)
	}
	if have, want := lexer.NextByteSet(Space), int(' '); have != want {
		t.Errorf("want %#v, have %#v", want, have)
	}
	if have, want := lexer.NextByteSet(Space), -1; have != want {
		t.Errorf("want %#v, have %#v", want, have)
	}
	if have, want := lexer.NextByteSet(Alnum), int('a'); have != want {
		t.Errorf("want %#v, have %#v", want, have)
	}
	if have, want := lexer.NextByteSet(Space), int('\n'); have != want {
		t.Errorf("want %#v, have %#v", want, have)
	}
	if have, want := lexer.NextByteSet(Alnum), -1; have != want {
		t.Errorf("want %#v, have %#v", want, have)
	}
}

func Test_Lexer_NextBytesSet(t *testing.T) {
	lexer := NewLexer("an alphanumerical 90_ \tstring\t\t \n")

	if have, want := lexer.NextBytesSet(Alnum), "an"; have != want {
		t.Errorf("want %#v, have %#v", want, have)
	}
	if have, want := lexer.NextBytesSet(Alnum), ""; have != want {
		t.Errorf("want %#v, have %#v", want, have)
	}
	if have, want := lexer.NextBytesSet(Space), " "; have != want {
		t.Errorf("want %#v, have %#v", want, have)
	}
	if have, want := lexer.NextBytesSet(Alnum), "alphanumerical"; have != want {
		t.Errorf("want %#v, have %#v", want, have)
	}
	if have, want := lexer.NextBytesSet(Space), " "; have != want {
		t.Errorf("want %#v, have %#v", want, have)
	}
	if have, want := lexer.NextBytesSet(AlnumU), "90_"; have != want {
		t.Errorf("want %#v, have %#v", want, have)
	}
	if have, want := lexer.NextBytesSet(Space), " \t"; have != want {
		t.Errorf("want %#v, have %#v", want, have)
	}
	if have, want := lexer.NextBytesSet(Alnum), "string"; have != want {
		t.Errorf("want %#v, have %#v", want, have)
	}
	if have, want := lexer.NextBytesSet(Space), "\t\t \n"; have != want {
		t.Errorf("want %#v, have %#v", want, have)
	}
}

func Test_Lexer_SkipRegexp(t *testing.T) {
	lexer := NewLexer("an alphanumerical 90_ \tstring\t\t \n")

	if !lexer.SkipRegexp(regexp.MustCompile(`^\w+`)) {
		t.Error("want true")
	}
	if have, want := lexer.Rest(), " alphanumerical 90_ \tstring\t\t \n"; have != want {
		t.Errorf("want %#v, have %#v", want, have)
	}
	if !lexer.SkipRegexp(regexp.MustCompile(`^.+`)) {
		t.Error("want true")
	}
	if have, want := lexer.Rest(), "\n"; have != want {
		t.Errorf("want %#v, have %#v", want, have)
	}
	if lexer.SkipRegexp(regexp.MustCompile(`^.+`)) {
		t.Error("want false")
	}
	// This call returns false since the matched string was empty.
	if lexer.SkipRegexp(regexp.MustCompile(`^.*`)) {
		t.Error("want false")
	}
	if have, want := lexer.Rest(), "\n"; have != want {
		t.Errorf("want %#v, have %#v", want, have)
	}
}

func Test_Lexer_SkipRegexp__panic(t *testing.T) {
	lexer := NewLexer("an alphanumerical 90_ \tstring\t\t \n")

	var p string
	func() {
		defer func() {
			p = recover().(string)
		}()
		lexer.SkipRegexp(regexp.MustCompile(`\w+`))
	}()
	if p != "Lexer.SkipRegexp: regular expression \"\\\\w+\" must have prefix \"^\"." {
		t.Error(p)
	}
}

func Test_Lexer_NextRegexp(t *testing.T) {
	lexer := NewLexer("an alphanumerical 90_ \tstring\t\t \n")

	if have, want := lexer.NextRegexp(regexp.MustCompile(`^\w+`))[0], "an"; have != want {
		t.Errorf("want %#v, have %#v", want, have)
	}
	if have, want := lexer.NextRegexp(regexp.MustCompile(`^[\w ]+`))[0], " alphanumerical 90_ "; have != want {
		t.Errorf("want %#v, have %#v", want, have)
	}
	if have, want := lexer.NextRegexp(regexp.MustCompile(`^.+`))[0], "\tstring\t\t "; have != want {
		t.Errorf("want %#v, have %#v", want, have)
	}
	if have := lexer.NextRegexp(regexp.MustCompile(`^.+`)); have != nil {
		t.Errorf("want nil, have %#v", have)
	}
	if have, want := lexer.NextRegexp(regexp.MustCompile(`^.*`))[0], ""; have != want {
		t.Errorf("want %#v, have %#v", want, have)
	}
	if have, want := lexer.Rest(), "\n"; have != want {
		t.Errorf("want %#v, have %#v", want, have)
	}
}

func Test_Lexer_NextRegexp__panic(t *testing.T) {
	lexer := NewLexer("an alphanumerical 90_ \tstring\t\t \n")

	var p string
	func() {
		defer func() {
			p = recover().(string)
		}()
		lexer.NextRegexp(regexp.MustCompile(`\w+`))
	}()
	if p != "Lexer.NextRegexp: regular expression \"\\\\w+\" must have prefix \"^\"." {
		t.Error(p)
	}
}

func Test_Lexer_Mark__beginning(t *testing.T) {
	lexer := NewLexer("text")

	mark := lexer.Mark()
	if have, want := lexer.NextString("text"), "text"; have != want {
		t.Errorf("want %#v, have %#v", want, have)
	}

	if have, want := lexer.Rest(), ""; have != want {
		t.Errorf("want %#v, have %#v", want, have)
	}

	lexer.Reset(mark)

	if have, want := lexer.Rest(), "text"; have != want {
		t.Errorf("want %#v, have %#v", want, have)
	}
}

func Test_Lexer_Mark__middle(t *testing.T) {
	lexer := NewLexer("text")
	lexer.NextString("te")

	mark := lexer.Mark()
	if have, want := lexer.NextString("x"), "x"; have != want {
		t.Errorf("want %#v, have %#v", want, have)
	}

	if have, want := lexer.Rest(), "t"; have != want {
		t.Errorf("want %#v, have %#v", want, have)
	}

	lexer.Reset(mark)

	if have, want := lexer.Rest(), "xt"; have != want {
		t.Errorf("want %#v, have %#v", want, have)
	}
}

// Demonstrates that multiple marks can be taken at the same time and that
// the lexer can be reset to any of them in any order.
func Test_Lexer_Reset__multiple(t *testing.T) {
	lexer := NewLexer("text")

	mark0 := lexer.Mark()
	if have, want := lexer.NextString("te"), "te"; have != want {
		t.Errorf("want %#v, have %#v", want, have)
	}
	mark2 := lexer.Mark()
	if have, want := lexer.NextString("x"), "x"; have != want {
		t.Errorf("want %#v, have %#v", want, have)
	}
	mark3 := lexer.Mark()
	if have, want := lexer.NextString("t"), "t"; have != want {
		t.Errorf("want %#v, have %#v", want, have)
	}
	mark4 := lexer.Mark()

	if have, want := lexer.Rest(), ""; have != want {
		t.Errorf("want %#v, have %#v", want, have)
	}

	lexer.Reset(mark0)

	if have, want := lexer.Rest(), "text"; have != want {
		t.Errorf("want %#v, have %#v", want, have)
	}

	lexer.Reset(mark3)

	if have, want := lexer.Rest(), "t"; have != want {
		t.Errorf("want %#v, have %#v", want, have)
	}

	lexer.Reset(mark2)

	if have, want := lexer.Rest(), "xt"; have != want {
		t.Errorf("want %#v, have %#v", want, have)
	}

	lexer.Reset(mark4)

	if have, want := lexer.Rest(), ""; have != want {
		t.Errorf("want %#v, have %#v", want, have)
	}
}

func Test_Lexer_Since(t *testing.T) {
	lexer := NewLexer("text")
	mark := lexer.Mark()

	if have, want := lexer.NextString("te"), "te"; have != want {
		t.Errorf("want %#v, have %#v", want, have)
	}
	if have, want := lexer.NextString("st"), ""; have != want { // Did not match.
		t.Errorf("want %#v, have %#v", want, have)
	}
	if have, want := lexer.Since(mark), "te"; have != want {
		t.Errorf("want %#v, have %#v", want, have)
	}
	if have, want := lexer.NextString("xt"), "xt"; have != want {
		t.Errorf("want %#v, have %#v", want, have)
	}
	if have, want := lexer.Since(mark), "text"; have != want {
		t.Errorf("want %#v, have %#v", want, have)
	}
}

func Test_Lexer_Copy(t *testing.T) {
	lexer := NewLexer("text")
	copied := lexer.Copy()

	if have, want := copied.Rest(), lexer.Rest(); have != want {
		t.Errorf("want %#v, have %#v", want, have)
	}

	copied.NextString("te")

	if have, want := copied.Rest(), "xt"; have != want {
		t.Errorf("want %#v, have %#v", want, have)
	}
	if have, want := lexer.Rest(), "text"; have != want { // The original is not yet affected.
		t.Errorf("want %#v, have %#v", want, have)
	}
}

func Test_Lexer_Commit(t *testing.T) {
	lexer := NewLexer("text")
	copied := lexer.Copy()
	copied.NextString("te")

	if have, want := lexer.Rest(), "text"; have != want { // The original is not yet affected.
		t.Errorf("want %#v, have %#v", want, have)
	}

	lexer.Commit(copied)

	if have, want := lexer.Rest(), "xt"; have != want {
		t.Errorf("want %#v, have %#v", want, have)
	}
}

func Test_NewByteSet(t *testing.T) {
	set := NewByteSet("A-Za-z0-9_\xFC")

	expected := "" +
		"0123456789" +
		"ABCDEFGHIJKLMNOPQRSTUVWXYZ" +
		"_" +
		"abcdefghijklmnopqrstuvwxyz" +
		"\xFC"
	if have, want := set.String(), expected; have != want {
		t.Errorf("want %#v, have %#v", want, have)
	}
}

// Demonstrates how to specify a byte set that includes a hyphen,
// since that character is also used for byte ranges.
// The hyphen is written as ---, which is a range from hyphen to hyphen.
func Test_NewByteSet__range_hyphen_beginning(t *testing.T) {
	set := NewByteSet("---a-z")

	if have, want := set.String(), "-abcdefghijklmnopqrstuvwxyz"; have != want {
		t.Errorf("want %#v, have %#v", want, have)
	}
}

// The byte set "x---a" is interpreted as 3 parts: "x--", "-", "a".
//
// The range "x--" is empty since U+0078 'x' comes later in Unicode than
// U+002D '-'.
func Test_NewByteSet__range_hyphen_middle(t *testing.T) {
	set := NewByteSet("x---a")

	if have, want := set.String(), "-a"; have != want {
		t.Errorf("want %#v, have %#v", want, have)
	}
}

// When a hyphen is listed at the very beginning, it is not considered part
// of a byte range but interpreted literally.
func Test_NewByteSet__hyphen_beginning(t *testing.T) {
	set := NewByteSet("-ax")

	if have, want := set.String(), "-ax"; have != want {
		t.Errorf("want %#v, have %#v", want, have)
	}
}

// When a hyphen is listed directly after a range ('+-+' in this case),
// it is interpreted literally.
func Test_NewByteSet__hyphen_after_range(t *testing.T) {
	set := NewByteSet("+-+-")

	if have, want := set.String(), "+-"; have != want {
		t.Errorf("want %#v, have %#v", want, have)
	}
}

// When a hyphen is listed at the very end, it is not considered part
// of a byte range but interpreted literally.
func Test_NewByteSet__hyphen_end(t *testing.T) {
	set := NewByteSet("ax-")

	if have, want := set.String(), "-ax"; have != want {
		t.Errorf("want %#v, have %#v", want, have)
	}
}

func Test_ByteSet_Inverse(t *testing.T) {
	set := NewByteSet("A-Za-z0-9_\xFC")
	inverse := set.Inverse()

	for i := range 256 {
		if inverse.Contains(byte(i)) == set.Contains(byte(i)) {
			t.Errorf("equal at %#v", i)
		}
	}
}

func Test_ByteSet_Contains(t *testing.T) {
	set := NewByteSet("A-Za-z0-9_\xFC")

	if have, want := set.String(), "0123456789ABCDEFGHIJKLMNOPQRSTUVWXYZ_abcdefghijklmnopqrstuvwxyz\xFC"; have != want {
		t.Errorf("want %#v, have %#v", want, have)
	}
}

func Test__XPrint(t *testing.T) {
	set := XPrint

	if have, want := set.String(), "\t\n !\"#$%&'()*+,-./0123456789:;<=>?@ABCDEFGHIJKLMNOPQRSTUVWXYZ[\\]^_`abcdefghijklmnopqrstuvwxyz{|}~"; have != want {
		t.Errorf("want %#v, have %#v", want, have)
	}
}

func Test__Alpha(t *testing.T) {
	set := Alpha

	if have, want := set.String(), "ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz"; have != want {
		t.Errorf("want %#v, have %#v", want, have)
	}
}

func Test__qa(t *testing.T) {
	ck := intqa.NewQAChecker(t.Errorf)
	ck.Configure("*", "*", "*", -intqa.EMissingTest)
	ck.Check()
}

func (bs *ByteSet) String() string {
	var actual strings.Builder
	for i := range 256 {
		if bs.Contains(byte(i)) {
			actual.WriteByte(byte(i))
		}
	}
	return actual.String()
}
