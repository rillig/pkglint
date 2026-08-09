package pkglint

import (
	"github.com/rillig/pkglint/v23/regex"
	"testing"
)

func Test_RawLine_Orig(tt *testing.T) {
	t := NewTester(tt)
	defer t.Done()

	line := t.NewLine("filename", 123, "text")

	t.CheckEquals(line.raw[0].orignl, "text\n")
	t.CheckEquals(line.raw[0].Orig(), "text")
}

func Test_NewLocation(tt *testing.T) {
	t := NewTester(tt)
	defer t.Done()

	loc := NewLocation("subdir/filename", 123)

	t.CheckEquals(loc.Filename, NewCurrPathString("subdir/filename"))
	t.CheckEquals(loc.lineno, 123)
}

func Test_Location_Lineno(tt *testing.T) {
	t := NewTester(tt)
	defer t.Done()

	loc := NewLocation("subdir/filename", 123)

	t.CheckEquals(loc.Lineno(0), 123)
}

func Test_Location_File(tt *testing.T) {
	t := NewTester(tt)
	defer t.Done()

	loc := NewLocation("subdir/filename", 123)

	t.CheckEquals(loc.File("other"), NewCurrPathString("subdir/other"))
}

func Test_NewLine__assertion(tt *testing.T) {
	t := NewTester(tt)
	defer t.Done()

	t.ExpectAssert(func() { NewLine("filename", 123, "text", nil) })
}

func Test_NewLineMulti(tt *testing.T) {
	t := NewTester(tt)
	defer t.Done()

	line := NewLineMulti("subdir/filename", 123, "", []*RawLine{nil, nil, nil})

	t.CheckEquals(line.Filename(), NewCurrPathString("subdir/filename"))
	t.CheckEquals(line.Linenos(), "123--125")
}

func Test_NewLineEOF(tt *testing.T) {
	t := NewTester(tt)
	defer t.Done()

	line := NewLineEOF("subdir/filename")

	t.CheckEquals(line.Filename(), NewCurrPathString("subdir/filename"))
	t.CheckEquals(line.Linenos(), "EOF")
}

func Test_NewLineWhole(tt *testing.T) {
	t := NewTester(tt)
	defer t.Done()

	line := NewLineWhole("subdir/filename")

	t.CheckEquals(line.Filename(), NewCurrPathString("subdir/filename"))
	t.CheckEquals(line.Linenos(), "")
}

func Test_Line_Filename(tt *testing.T) {
	t := NewTester(tt)
	defer t.Done()

	line := t.NewLine("subdir/filename", 123, "")

	t.CheckEquals(line.Filename(), NewCurrPathString("subdir/filename"))
}

func Test_Line_File(tt *testing.T) {
	t := NewTester(tt)
	defer t.Done()

	line := t.NewLine("subdir/filename", 123, "")

	t.CheckEquals(line.File("subsub/nested"),
		NewCurrPathString("subdir/subsub/nested"))
}

func Test_Line_Linenos(tt *testing.T) {
	t := NewTester(tt)
	defer t.Done()

	single := t.NewLine("filename", 123, "")
	whole := NewLineWhole("filename")
	multi := NewLineMulti("filename", 123, "text", []*RawLine{nil, nil, nil})

	t.CheckEquals(single.Linenos(), "123")
	t.CheckEquals(whole.Linenos(), "")
	t.CheckEquals(multi.Linenos(), "123--125")
}

func Test_Line_RelLine__assertion(tt *testing.T) {
	t := NewTester(tt)
	defer t.Done()

	line := t.NewLine("filename", 123, "")
	whole := NewLineWhole("filename")

	t.ExpectAssert(func() { line.RelLine(whole) })
}

func Test_Line_RelLocation__assertion(tt *testing.T) {
	t := NewTester(tt)
	defer t.Done()

	line := t.NewLine("filename", 123, "")
	whole := NewLineWhole("filename")

	t.ExpectAssert(func() { line.RelLocation(whole.Location) })
}

func Test_Line_Rel(tt *testing.T) {
	t := NewTester(tt)
	defer t.Done()

	t.Chdir(".")
	line1 := t.NewLine("subdir/filename", 123, "from")

	rel := line1.Rel("other/filename")

	t.CheckEquals(rel, NewRelPathString("../other/filename"))
}

func Test_Line_IsMultiline(tt *testing.T) {
	t := NewTester(tt)
	defer t.Done()

	t.CheckEquals(t.NewLine("filename", 123, "text").IsMultiline(), false)

	t.CheckEquals(NewLineEOF("filename").IsMultiline(), false)

	multi := NewLineMulti("filename", 123, "text", []*RawLine{nil, nil, nil})
	t.CheckEquals(multi.IsMultiline(), true)
}

func Test_Line_RawText(tt *testing.T) {
	t := NewTester(tt)
	defer t.Done()

	t.SetUpCommandLine("--show-autofix")
	line := t.NewLine("filename", 123, "text")

	t.CheckEquals(line.RawText(0), "text")

	fix := line.Autofix()
	fix.Notef("Replacing.")
	fix.Replace("text", "replaced")
	fix.Apply()

	t.CheckEquals(line.RawText(0), "replaced")

	t.CheckOutputLines(
		"NOTE: filename:123: Replacing.",
		"AUTOFIX: filename:123: Replacing \"text\" with \"replaced\".")
}

func Test_Line_IsCvsID(tt *testing.T) {
	t := NewTester(tt)
	defer t.Done()

	test := func(text string, prefix regex.Pattern, expectedID bool, expectedExpanded bool) {
		line := t.NewLine("filename", 123, text)

		id, expanded := line.IsCvsID(prefix)

		t.CheckEquals(id, expectedID)
		t.CheckEquals(expanded, expectedExpanded)
	}

	test(CvsID, ``, true, false)

	test("$"+"NetBSD: ... $", ``, true, true)
}

func Test_Line_Errorf(tt *testing.T) {
	t := NewTester(tt)
	defer t.Done()

	line := t.NewLine("filename", 123, "text")

	line.Errorf("For your information.")

	t.CheckOutputLines(
		"ERROR: filename:123: For your information.")
}

func Test_Line_Warnf(tt *testing.T) {
	t := NewTester(tt)
	defer t.Done()

	line := t.NewLine("filename", 123, "text")

	line.Warnf("For your information.")

	t.CheckOutputLines(
		"WARN: filename:123: For your information.")
}

func Test_Line_Notef(tt *testing.T) {
	t := NewTester(tt)
	defer t.Done()

	line := t.NewLine("filename", 123, "text")

	line.Notef("For your information.")

	t.CheckOutputLines(
		"NOTE: filename:123: For your information.")
}

func Test_Line_Explain(tt *testing.T) {
	t := NewTester(tt)
	defer t.Done()

	t.SetUpCommandLine("--explain")
	line := t.NewLine("filename", 123, "text")

	line.Notef("For your information.")
	line.Explain(
		"First line",
		"continues.")

	t.CheckOutputLines(
		"NOTE: filename:123: For your information.",
		"",
		"\tFirst line continues.",
		"")
}

func Test_Line_Explain__without_diagnostic(tt *testing.T) {
	t := NewTester(tt)
	defer t.Done()

	t.SetUpCommandLine("--explain")
	line := t.NewLine("filename", 123, "text")

	// It is unusual to have an explanation without any preceding
	// diagnostic.
	line.Explain(
		"First line",
		"continues.")

	// There is no empty line above the explanation since that is the
	// job of the SeparatorWriter, which only steps in if there is a
	// diagnostic above the explanation.
	t.CheckOutputLines(
		"\tFirst line continues.",
		"")
}

func Test_Line_String(tt *testing.T) {
	t := NewTester(tt)
	defer t.Done()

	line := t.NewLine("filename", 123, "text")
	t.CheckEquals(line.String(), "filename:123: text")

	// Since Line.String is only used for debugging purposes, trailing
	// whitespace is OK.
	t.CheckEquals(NewLineEOF("filename").String(), "filename:EOF: ")

	multi := NewLineMulti("filename", 123, "text", []*RawLine{nil, nil, nil})
	t.CheckEquals(multi.String(), "filename:123--125: text")
}

func Test_Line_Autofix__reuse_incomplete(tt *testing.T) {
	t := NewTester(tt)
	defer t.Done()

	line := t.NewLine("filename.mk", 1, "")

	fix := line.Autofix()
	fix.Warnf("Warning.")
	// For some reason, the other required calls are left out.

	t.ExpectAssert(func() { _ = line.Autofix() })

	// Properly finish the standard call sequence for an Autofix.
	fix.Apply()

	t.CheckOutputLines(
		"WARN: filename.mk:1: Warning.")
}
