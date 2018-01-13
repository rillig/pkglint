package main

import "gopkg.in/check.v1"

func (s *Suite) Test_Autofix_ReplaceRegex(c *check.C) {
	s.Init(c)
	s.UseCommandLine("--show-autofix")
	fname := s.CreateTmpFile("Makefile", ""+
		"line1\n"+
		"line2\n"+
		"line3\n")
	lines := LoadExistingLines(fname, true)

	fix := lines[1].Autofix()
	fix.Warnf("Something's wrong here.")
	fix.ReplaceRegex(`.`, "X")
	fix.Apply()
	SaveAutofixChanges(lines)

	c.Check(lines[1].raw[0].textnl, equals, "XXXXX\n")
	c.Check(s.LoadTmpFile("Makefile"), equals, "line1\nline2\nline3\n")
	s.CheckOutputLines(
		"WARN: ~/Makefile:2: Something's wrong here.",
		"AUTOFIX: ~/Makefile:2: Replacing regular expression \".\" with \"X\".")
}

func (s *Suite) Test_Autofix_ReplaceRegex_with_show_autofix(c *check.C) {
	s.Init(c)
	s.UseCommandLine("--autofix")
	fname := s.CreateTmpFile("Makefile", ""+
		"line1\n"+
		"line2\n"+
		"line3\n")
	lines := LoadExistingLines(fname, true)

	fix := lines[1].Autofix()
	fix.Warnf("Something's wrong here.")
	fix.ReplaceRegex(`.`, "X")
	fix.Apply()
	SaveAutofixChanges(lines)

	c.Check(s.LoadTmpFile("Makefile"), equals, "line1\nXXXXX\nline3\n")
	s.CheckOutputLines(
		"AUTOFIX: ~/Makefile:2: Replacing regular expression \".\" with \"X\".",
		"AUTOFIX: ~/Makefile: Has been auto-fixed. Please re-run pkglint.")
}

func (s *Suite) Test_autofix_MkLines(c *check.C) {
	s.Init(c)
	s.UseCommandLine("--autofix")
	fname := s.CreateTmpFile("Makefile", ""+
		"line1 := value1\n"+
		"line2 := value2\n"+
		"line3 := value3\n")
	pkg := NewPackage("category/basename")
	G.Pkg = pkg
	mklines := pkg.loadPackageMakefile(fname)
	G.Pkg = nil

	fix := mklines.mklines[1].Autofix()
	fix.Warnf("Something's wrong here.")
	fix.ReplaceRegex(`.`, "X")
	fix.Apply()
	SaveAutofixChanges(mklines.lines)

	c.Check(s.LoadTmpFile("Makefile"), equals, ""+
		"line1 := value1\n"+
		"XXXXXXXXXXXXXXX\n"+
		"line3 := value3\n")
	s.CheckOutputLines(
		"AUTOFIX: ~/Makefile:2: Replacing regular expression \".\" with \"X\".",
		"AUTOFIX: ~/Makefile: Has been auto-fixed. Please re-run pkglint.")
}

func (s *Suite) Test_Autofix_multiple_modifications(c *check.C) {
	s.Init(c)
	s.UseCommandLine("--show-autofix", "--explain")

	line := NewLine("fname", 1, "dummy", T.NewRawLines(1, "original\n"))

	c.Check(line.autofix, check.IsNil)
	c.Check(line.raw, check.DeepEquals, T.NewRawLines(1, "original\n"))

	{
		fix := line.Autofix()
		fix.Warnf("Silent-Magic-Diagnostic")
		fix.ReplaceRegex(`(.)(.*)(.)`, "$3$2$1")
		fix.Apply()
	}

	c.Check(line.autofix, check.NotNil)
	c.Check(line.raw, check.DeepEquals, T.NewRawLines(1, "original\n", "lriginao\n"))
	s.CheckOutputLines(
		"AUTOFIX: fname:1: Replacing regular expression \"(.)(.*)(.)\" with \"$3$2$1\".")

	{
		fix := line.Autofix()
		fix.Warnf("Silent-Magic-Diagnostic")
		fix.Replace("i", "u")
		fix.Apply()
	}

	c.Check(line.autofix, check.NotNil)
	c.Check(line.raw, check.DeepEquals, T.NewRawLines(1, "original\n", "lruginao\n"))
	c.Check(line.raw[0].textnl, equals, "lruginao\n")
	s.CheckOutputLines(
		"AUTOFIX: fname:1: Replacing \"i\" with \"u\".")

	{
		fix := line.Autofix()
		fix.Warnf("Silent-Magic-Diagnostic")
		fix.Replace("lruginao", "middle")
		fix.Apply()
	}

	c.Check(line.autofix, check.NotNil)
	c.Check(line.raw, check.DeepEquals, T.NewRawLines(1, "original\n", "middle\n"))
	c.Check(line.raw[0].textnl, equals, "middle\n")
	s.CheckOutputLines(
		"AUTOFIX: fname:1: Replacing \"lruginao\" with \"middle\".")

	{
		fix := line.Autofix()
		fix.Warnf("Silent-Magic-Diagnostic")
		fix.InsertBefore("before")
		fix.Apply()

		fix.Warnf("Silent-Magic-Diagnostic")
		fix.InsertBefore("between before and middle")
		fix.Apply()

		fix.Warnf("Silent-Magic-Diagnostic")
		fix.InsertAfter("between middle and after")
		fix.Apply()

		fix.Notef("This diagnostic is necessary for the following explanation.")
		fix.InsertAfter("after")
		fix.Explain(
			"When inserting multiple lines, Apply must be called in-between.",
			"Otherwise the changes are not described to the human reader.")
		fix.Apply()
	}

	c.Check(line.autofix.linesBefore, check.DeepEquals, []string{
		"before\n",
		"between before and middle\n"})
	c.Check(line.autofix.lines[0].textnl, equals, "middle\n")
	c.Check(line.autofix.linesAfter, deepEquals, []string{
		"between middle and after\n",
		"after\n"})
	s.CheckOutputLines(
		"AUTOFIX: fname:1: Inserting a line \"before\" before this line.",
		"AUTOFIX: fname:1: Inserting a line \"between before and middle\" before this line.",
		"AUTOFIX: fname:1: Inserting a line \"between middle and after\" after this line.",
		"NOTE: fname:1: This diagnostic is necessary for the following explanation.",
		"",
		"\tWhen inserting multiple lines, Apply must be called in-between.",
		"\tOtherwise the changes are not described to the human reader.",
		"",
		"AUTOFIX: fname:1: Inserting a line \"after\" after this line.")

	{
		fix := line.Autofix()
		fix.Warnf("Silent-Magic-Diagnostic")
		fix.Delete()
		fix.Apply()
	}

	c.Check(line.autofix.linesBefore, check.DeepEquals, []string{
		"before\n",
		"between before and middle\n"})
	c.Check(line.autofix.lines[0].textnl, equals, "")
	c.Check(line.autofix.linesAfter, deepEquals, []string{
		"between middle and after\n",
		"after\n"})
	s.CheckOutputLines(
		"AUTOFIX: fname:1: Deleting this line.")
}

func (s *Suite) Test_Autofix_show_source_code(c *check.C) {
	s.Init(c)
	s.UseCommandLine("--show-autofix", "--source")
	line := NewLineMulti("Makefile", 27, 29, "# old", T.NewRawLines(
		27, "before\n",
		28, "The old song\n",
		29, "after\n"))

	{
		fix := line.Autofix()
		fix.Warnf("Using \"old\" is deprecated.")
		fix.Replace("old", "new")
		fix.Apply()
	}

	s.CheckOutputLines(
		"WARN: Makefile:27--29: Using \"old\" is deprecated.",
		"AUTOFIX: Makefile:27--29: Replacing \"old\" with \"new\".",
		"> before",
		"- The old song",
		"+ The new song",
		"> after")
}

func (s *Suite) Test_Autofix_InsertBefore(c *check.C) {
	s.Init(c)
	s.UseCommandLine("--show-autofix", "--source")
	line := NewLine("Makefile", 30, "original", T.NewRawLines(30, "original\n"))

	fix := line.Autofix()
	fix.Warnf("Dummy")
	fix.InsertBefore("inserted")
	fix.Apply()

	s.CheckOutputLines(
		"WARN: Makefile:30: Dummy",
		"AUTOFIX: Makefile:30: Inserting a line \"inserted\" before this line.",
		"+ inserted",
		"> original")
}

func (s *Suite) Test_Autofix_Delete(c *check.C) {
	s.Init(c)
	s.UseCommandLine("--show-autofix", "--source")
	line := NewLine("Makefile", 30, "to be deleted", T.NewRawLines(30, "to be deleted\n"))

	fix := line.Autofix()
	fix.Warnf("Dummy")
	fix.Delete()
	fix.Apply()

	s.CheckOutputLines(
		"WARN: Makefile:30: Dummy",
		"AUTOFIX: Makefile:30: Deleting this line.",
		"- to be deleted")
}
