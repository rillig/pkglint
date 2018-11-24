package main

import "gopkg.in/check.v1"

// Since the --source option generates multi-line diagnostics,
// they are separated by an empty line.
//
// Whether the quoted source code is written above or below the
// diagnostics depends on the --show-autofix and --autofix options.
// When any of them is given, the general rule is given first, followed
// by a description of the fix ("replacing A with B"), finally followed
// by the actual changes to the code.
//
// In default mode, without any autofix options, the usual order is
// to first show the code and then show the diagnostic. This allows
// the diagnostics to underline the relevant part of the source code
// and reminds of the squiggly line used for spellchecking.
func (s *Suite) Test__show_source_separator(c *check.C) {
	t := s.Init(c)

	t.SetupCommandLine("--source")
	lines := t.SetupFileLines("DESCR",
		"The first line",
		"The second line",
		"The third line")

	fix := lines.Lines[1].Autofix()
	fix.Warnf("Using \"second\" is deprecated.")
	fix.Replace("second", "silver medal")
	fix.Apply()

	lines.Lines[2].Warnf("Dummy warning.")

	fix = lines.Lines[2].Autofix()
	fix.Warnf("Using \"third\" is deprecated.")
	fix.Replace("third", "bronze medal")
	fix.Apply()

	t.CheckOutputLines(
		">\tThe second line",
		"WARN: ~/DESCR:2: Using \"second\" is deprecated.",
		"",
		">\tThe third line",
		"WARN: ~/DESCR:3: Dummy warning.",
		"",
		">\tThe third line",
		"WARN: ~/DESCR:3: Using \"third\" is deprecated.")
}

func (s *Suite) Test__show_source_separator_show_autofix(c *check.C) {
	t := s.Init(c)

	t.SetupCommandLine("--source", "--show-autofix")
	lines := t.SetupFileLines("DESCR",
		"The first line",
		"The second line",
		"The third line")

	fix := lines.Lines[1].Autofix()
	fix.Warnf("Using \"second\" is deprecated.")
	fix.Replace("second", "silver medal")
	fix.Apply()

	lines.Lines[2].Warnf("Dummy warning.")

	fix = lines.Lines[2].Autofix()
	fix.Warnf("Using \"third\" is deprecated.")
	fix.Replace("third", "bronze medal")
	fix.Apply()

	t.CheckOutputLines(
		"WARN: ~/DESCR:2: Using \"second\" is deprecated.",
		"AUTOFIX: ~/DESCR:2: Replacing \"second\" with \"silver medal\".",
		"-\tThe second line",
		"+\tThe silver medal line",
		"",
		"WARN: ~/DESCR:3: Using \"third\" is deprecated.",
		"AUTOFIX: ~/DESCR:3: Replacing \"third\" with \"bronze medal\".",
		"-\tThe third line",
		"+\tThe bronze medal line")
}

func (s *Suite) Test__show_source_separator_autofix(c *check.C) {
	t := s.Init(c)

	t.SetupCommandLine("--source", "--autofix")
	lines := t.SetupFileLines("DESCR",
		"The first line",
		"The second line",
		"The third line")

	fix := lines.Lines[1].Autofix()
	fix.Warnf("Using \"second\" is deprecated.")
	fix.Replace("second", "silver medal")
	fix.Apply()

	lines.Lines[2].Warnf("Dummy warning.")

	fix = lines.Lines[2].Autofix()
	fix.Warnf("Using \"third\" is deprecated.")
	fix.Replace("third", "bronze medal")
	fix.Apply()

	t.CheckOutputLines(
		"AUTOFIX: ~/DESCR:2: Replacing \"second\" with \"silver medal\".",
		"-\tThe second line",
		"+\tThe silver medal line",
		"",
		"AUTOFIX: ~/DESCR:3: Replacing \"third\" with \"bronze medal\".",
		"-\tThe third line",
		"+\tThe bronze medal line")
}

func (s *Suite) Test_Pkglint_Explain__only(c *check.C) {
	t := s.Init(c)

	t.SetupCommandLine("--only", "interesting", "--explain")
	line := t.NewLine("Makefile", 27, "The old song")

	line.Warnf("Filtered warning.")                 // Is not logged.
	G.Explain("Explanation for the above warning.") // Neither is this explanation logged.

	line.Warnf("What an interesting line.")
	G.Explain("This explanation is logged.")

	t.CheckOutputLines(
		"WARN: Makefile:27: What an interesting line.",
		"",
		"\tThis explanation is logged.",
		"")
}

func (s *Suite) Test_Pkglint_Explain__long_lines(c *check.C) {
	t := s.Init(c)

	G.Explain(
		"123456789 12345678. abcdefghi. 123456789 123456789 123456789 123456789 123456789")

	t.CheckOutputLines(
		"Long explanation line: 123456789 12345678. abcdefghi. 123456789 123456789 123456789 123456789 123456789",
		"Break after: 123456789 12345678. abcdefghi. 123456789 123456789 123456789",
		"Short space after period: 123456789 12345678. abcdefghi. 123456789 123456789 123456789 123456789 123456789")
}

func (s *Suite) Test_Pkglint_Explain__trailing_whitespace(c *check.C) {
	t := s.Init(c)

	G.Explain(
		"This is a space: ")

	t.CheckOutputLines(
		"Trailing whitespace: \"This is a space: \"")
}

// TODO: Add tests for SeparatorWriter.
