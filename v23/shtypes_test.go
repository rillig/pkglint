package pkglint

import "testing"

func NewShAtom(typ ShAtomType, text string, quoting ShQuoting) *ShAtom {
	return &ShAtom{typ, text, quoting, nil}
}

func Test_ShAtomType_String(tt *testing.T) {
	t := NewTester(tt)
	defer t.Done()

	t.CheckEquals(shtComment.String(), "comment")
}

func Test_ShAtom_String(tt *testing.T) {
	t := NewTester(tt)
	defer t.Done()

	line := t.NewLine("filename.mk", 1, "")
	tokenizer := NewShTokenizer(line, "${ECHO} \"hello, world\"")

	atoms := tokenizer.ShAtoms()

	t.CheckEquals(len(atoms), 5)
	t.CheckEquals(atoms[0].String(), "expr(\"ECHO\")")
	t.CheckEquals(atoms[1].String(), "ShAtom(space, \" \", plain)")
	t.CheckEquals(atoms[2].String(), "ShAtom(text, \"\\\"\", d)")
	t.CheckEquals(atoms[3].String(), "ShAtom(text, \"hello, world\", d)")
	t.CheckEquals(atoms[4].String(), "\"\\\"\"")
}

func Test_ShQuoting_String(tt *testing.T) {
	t := NewTester(tt)
	defer t.Done()

	t.CheckEquals(shqDquotBacktSquot.String(), "dbs")
}

func Test_NewShToken__no_atoms(tt *testing.T) {
	t := NewTester(tt)
	defer t.Done()

	t.ExpectAssert(func() { NewShToken("", NewShAtom(shtText, "text", shqPlain)) })
	t.ExpectAssert(func() { NewShToken(" ", nil...) })
}

func Test_ShToken_String(tt *testing.T) {
	t := NewTester(tt)
	defer t.Done()

	line := t.NewLine("filename.mk", 1, "")
	tokenizer := NewShTokenizer(line, "${ECHO} \"hello, world\"")

	t.CheckEquals(tokenizer.ShToken().String(), "ShToken([expr(\"ECHO\")])")
	t.CheckEquals(tokenizer.ShToken().String(), "ShToken([ShAtom(text, \"\\\"\", d) ShAtom(text, \"hello, world\", d) \"\\\"\"])")
}
