package main

import "gopkg.in/check.v1"

func (s *Suite) Test_Autofix_ReplaceRegex(c *check.C) {
	t := s.Init(c)

	t.SetupCommandLine("--show-autofix")
	lines := t.SetupFileLines("Makefile",
		"line1",
		"line2",
		"line3")

	fix := lines[1].Autofix()
	fix.Warnf("Something's wrong here.")
	fix.ReplaceRegex(`.`, "X")
	fix.Apply()
	SaveAutofixChanges(lines)

	c.Check(lines[1].raw[0].textnl, equals, "XXXXX\n")
	t.CheckFileLines("Makefile",
		"line1",
		"line2",
		"line3")
	t.CheckOutputLines(
		"WARN: ~/Makefile:2: Something's wrong here.",
		"AUTOFIX: ~/Makefile:2: Replacing regular expression \".\" with \"X\".")
}

func (s *Suite) Test_Autofix_ReplaceRegex_with_autofix(c *check.C) {
	t := s.Init(c)

	t.SetupCommandLine("--autofix", "--source")
	lines := t.SetupFileLines("Makefile",
		"line1",
		"line2",
		"line3")

	fix := lines[1].Autofix()
	fix.Warnf("Something's wrong here.")
	fix.ReplaceRegex(`.`, "X")
	fix.Apply()

	fix.Warnf("Use Y instead of X.")
	fix.Replace("X", "Y")
	fix.Apply()

	SaveAutofixChanges(lines)

	t.CheckFileLines("Makefile",
		"line1",
		"YXXXX",
		"line3")
	t.CheckOutputLines(
		"AUTOFIX: ~/Makefile:2: Replacing regular expression \".\" with \"X\".",
		"- line2",
		"+ XXXXX",
		"",
		"AUTOFIX: ~/Makefile:2: Replacing \"X\" with \"Y\".",
		"- line2",
		"+ YXXXX",
		"",
		"AUTOFIX: ~/Makefile: Has been auto-fixed. Please re-run pkglint.")
}

func (s *Suite) Test_Autofix_ReplaceRegex_with_show_autofix(c *check.C) {
	t := s.Init(c)

	t.SetupCommandLine("--show-autofix", "--source")
	lines := t.SetupFileLines("Makefile",
		"line1",
		"line2",
		"line3")

	fix := lines[1].Autofix()
	fix.Warnf("Something's wrong here.")
	fix.ReplaceRegex(`.`, "X")
	fix.Apply()

	fix.Warnf("Use Y instead of X.")
	fix.Replace("X", "Y")
	fix.Apply()

	SaveAutofixChanges(lines)

	t.CheckOutputLines(
		"WARN: ~/Makefile:2: Something's wrong here.",
		"AUTOFIX: ~/Makefile:2: Replacing regular expression \".\" with \"X\".",
		"- line2",
		"+ XXXXX",
		"",
		"WARN: ~/Makefile:2: Use Y instead of X.",
		"AUTOFIX: ~/Makefile:2: Replacing \"X\" with \"Y\".",
		"- line2",
		"+ YXXXX")
}

func (s *Suite) Test_autofix_MkLines(c *check.C) {
	t := s.Init(c)

	t.SetupCommandLine("--autofix")
	t.SetupFileLines("Makefile",
		"line1 := value1",
		"line2 := value2",
		"line3 := value3")
	pkg := NewPackage("category/basename")
	G.Pkg = pkg
	mklines := pkg.loadPackageMakefile(t.TempFilename("Makefile"))
	G.Pkg = nil

	fix := mklines.mklines[1].Autofix()
	fix.Warnf("Something's wrong here.")
	fix.ReplaceRegex(`.`, "X")
	fix.Apply()

	SaveAutofixChanges(mklines.lines)

	t.CheckFileLines("Makefile",
		"line1 := value1",
		"XXXXXXXXXXXXXXX",
		"line3 := value3")
	t.CheckOutputLines(
		"AUTOFIX: ~/Makefile:2: Replacing regular expression \".\" with \"X\".",
		"AUTOFIX: ~/Makefile: Has been auto-fixed. Please re-run pkglint.")
}

func (s *Suite) Test_Autofix_multiple_modifications(c *check.C) {
	t := s.Init(c)

	t.SetupCommandLine("--show-autofix", "--explain")

	line := t.NewLine("fname", 1, "original")

	c.Check(line.autofix, check.IsNil)
	c.Check(line.raw, check.DeepEquals, t.NewRawLines(1, "original\n"))

	{
		fix := line.Autofix()
		fix.Warnf("Silent-Magic-Diagnostic")
		fix.ReplaceRegex(`(.)(.*)(.)`, "$3$2$1")
		fix.Apply()
	}

	c.Check(line.autofix, check.NotNil)
	c.Check(line.raw, check.DeepEquals, t.NewRawLines(1, "original\n", "lriginao\n"))
	t.CheckOutputLines(
		"AUTOFIX: fname:1: Replacing regular expression \"(.)(.*)(.)\" with \"$3$2$1\".")

	{
		fix := line.Autofix()
		fix.Warnf("Silent-Magic-Diagnostic")
		fix.Replace("i", "u")
		fix.Apply()
	}

	c.Check(line.autofix, check.NotNil)
	c.Check(line.raw, check.DeepEquals, t.NewRawLines(1, "original\n", "lruginao\n"))
	c.Check(line.raw[0].textnl, equals, "lruginao\n")
	t.CheckOutputLines(
		"AUTOFIX: fname:1: Replacing \"i\" with \"u\".")

	{
		fix := line.Autofix()
		fix.Warnf("Silent-Magic-Diagnostic")
		fix.Replace("lruginao", "middle")
		fix.Apply()
	}

	c.Check(line.autofix, check.NotNil)
	c.Check(line.raw, check.DeepEquals, t.NewRawLines(1, "original\n", "middle\n"))
	c.Check(line.raw[0].textnl, equals, "middle\n")
	t.CheckOutputLines(
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
		fix.Explain(
			"When inserting multiple lines, Apply must be called in-between.",
			"Otherwise the changes are not described to the human reader.")
		fix.InsertAfter("after")
		fix.Apply()
	}

	c.Check(line.autofix.linesBefore, check.DeepEquals, []string{
		"before\n",
		"between before and middle\n"})
	c.Check(line.autofix.lines[0].textnl, equals, "middle\n")
	c.Check(line.autofix.linesAfter, deepEquals, []string{
		"between middle and after\n",
		"after\n"})
	t.CheckOutputLines(
		"AUTOFIX: fname:1: Inserting a line \"before\" before this line.",
		"AUTOFIX: fname:1: Inserting a line \"between before and middle\" before this line.",
		"AUTOFIX: fname:1: Inserting a line \"between middle and after\" after this line.",
		"NOTE: fname:1: This diagnostic is necessary for the following explanation.",
		"AUTOFIX: fname:1: Inserting a line \"after\" after this line.",
		"",
		"\tWhen inserting multiple lines, Apply must be called in-between.",
		"\tOtherwise the changes are not described to the human reader.",
		"")

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
	t.CheckOutputLines(
		"AUTOFIX: fname:1: Deleting this line.")
}

func (s *Suite) Test_Autofix_show_source_code(c *check.C) {
	t := s.Init(c)

	t.SetupCommandLine("--show-autofix", "--source")
	lines := t.SetupFileLinesContinuation("Makefile",
		mkrcsid,
		"before \\",
		"The old song \\",
		"after")
	line := lines[1]

	{
		fix := line.Autofix()
		fix.Warnf("Using \"old\" is deprecated.")
		fix.Replace("old", "new")
		fix.Apply()
	}

	t.CheckOutputLines(
		"WARN: ~/Makefile:2--4: Using \"old\" is deprecated.",
		"AUTOFIX: ~/Makefile:2--4: Replacing \"old\" with \"new\".",
		"> before \\",
		"- The old song \\",
		"+ The new song \\",
		"> after")
}

func (s *Suite) Test_Autofix_InsertBefore(c *check.C) {
	t := s.Init(c)

	t.SetupCommandLine("--show-autofix", "--source")
	line := t.NewLine("Makefile", 30, "original")

	fix := line.Autofix()
	fix.Warnf("Dummy")
	fix.InsertBefore("inserted")
	fix.Apply()

	t.CheckOutputLines(
		"WARN: Makefile:30: Dummy",
		"AUTOFIX: Makefile:30: Inserting a line \"inserted\" before this line.",
		"+ inserted",
		"> original")
}

func (s *Suite) Test_Autofix_Delete(c *check.C) {
	t := s.Init(c)

	t.SetupCommandLine("--show-autofix", "--source")
	line := t.NewLine("Makefile", 30, "to be deleted")

	fix := line.Autofix()
	fix.Warnf("Dummy")
	fix.Delete()
	fix.Apply()

	t.CheckOutputLines(
		"WARN: Makefile:30: Dummy",
		"AUTOFIX: Makefile:30: Deleting this line.",
		"- to be deleted")
}

// Demonstrates that the --show-autofix option only shows those diagnostics
// that would be fixed.
func (s *Suite) Test_Autofix_suppress_unfixable_warnings(c *check.C) {
	t := s.Init(c)

	t.SetupCommandLine("--show-autofix", "--source")
	lines := t.NewLines("Makefile",
		"line1",
		"line2",
		"line3")

	lines[0].Warnf("This warning is not shown since it is not automatically fixed.")

	fix := lines[1].Autofix()
	fix.Warnf("Something's wrong here.")
	fix.ReplaceRegex(`.`, "X")
	fix.Apply()

	fix.Warnf("The XXX marks are usually not fixed, use TODO instead.")
	fix.Replace("XXX", "TODO")
	fix.Apply()

	lines[2].Warnf("Neither is this warning shown.")

	t.CheckOutputLines(
		"WARN: Makefile:2: Something's wrong here.",
		"AUTOFIX: Makefile:2: Replacing regular expression \".\" with \"X\".",
		"- line2",
		"+ XXXXX",
		"",
		"WARN: Makefile:2: The XXX marks are usually not fixed, use TODO instead.",
		"AUTOFIX: Makefile:2: Replacing \"XXX\" with \"TODO\".",
		"- line2",
		"+ TODOXX")
}

func (s *Suite) Test_SaveAutofixChanges(c *check.C) {
	t := s.Init(c)

	t.SetupCommandLine("--autofix")
	lines := t.SetupFileLines("DESCR",
		"Line 1",
		"Line 2")

	fix := lines[0].Autofix()
	fix.Warnf("Dummy warning.")
	fix.Replace("X", "Y")
	fix.Apply()

	// Since nothing has been effectively changed,
	// nothing needs to be saved.
	SaveAutofixChanges(lines)

	// And therefore, no AUTOFIX action must appear in the log.
	t.CheckOutputEmpty()
}
