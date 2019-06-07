package pkglint

import "gopkg.in/check.v1"

func (s *Suite) Test_Paragraph_Clear(c *check.C) {
	t := s.Init(c)

	para := NewParagraph(nil)

	para.Clear()

	t.Check(para.mklines, check.IsNil)

	para.Add(t.NewMkLine("filename.mk", 123, ""))

	para.Clear()

	t.Check(para.mklines, check.IsNil)
}

func (s *Suite) Test_Paragraph_Align(c *check.C) {
	t := s.Init(c)

	t.SetUpCommandLine("-Wall", "--autofix")
	mklines := t.SetUpFileMkLines("filename.mk",
		MkRcsID,
		"",
		"# comment",
		"VAR=value",
		"VAR=\t\t\tvalue")
	para := NewParagraph(nil)
	for _, mkline := range mklines.mklines {
		// Strictly speaking, lines 1 and 2 don't belong to the paragraph,
		// but aligning the lines works nevertheless.
		para.Add(mkline)
	}

	para.Align()
	mklines.SaveAutofixChanges()

	t.CheckOutputLines(
		"AUTOFIX: ~/filename.mk:4: Replacing \"\" with \"\\t\".",
		"AUTOFIX: ~/filename.mk:5: Replacing \"\\t\\t\\t\" with \"\\t\".")

	t.CheckFileLinesDetab("filename.mk",
		MkRcsID,
		"",
		"# comment",
		"VAR=    value",
		"VAR=    value")
}

func (s *Suite) Test_Paragraph_AlignTo(c *check.C) {
	t := s.Init(c)

	t.SetUpCommandLine("-Wall", "--autofix")
	mklines := t.SetUpFileMkLines("filename.mk",
		MkRcsID,
		"",
		"# comment",
		"VAR=value",
		"VAR=\t\tvalue",
		"VAR=\t \tvalue",
		"VAR=\t\t\tvalue")
	para := NewParagraph(nil)
	for _, mkline := range mklines.mklines {
		// Strictly speaking, lines 1 and 2 don't belong to the paragraph,
		// but aligning the lines works nevertheless.
		para.Add(mkline)
	}

	para.AlignTo(16)
	mklines.SaveAutofixChanges()

	t.CheckOutputLines(
		"AUTOFIX: ~/filename.mk:4: Replacing \"\" with \"\\t\\t\".",
		"AUTOFIX: ~/filename.mk:6: Replacing \"\\t \\t\" with \"\\t\\t\".",
		"AUTOFIX: ~/filename.mk:7: Replacing \"\\t\\t\\t\" with \"\\t\\t\".")

	t.CheckFileLinesDetab("filename.mk",
		MkRcsID,
		"",
		"# comment",
		"VAR=            value",
		"VAR=            value",
		"VAR=            value",
		"VAR=            value")
}