package main

import (
	"gopkg.in/check.v1"
	"strings"
)

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

// When the --show-autofix option is given, the warning is shown first,
// without the affected source, even if the --source option is also given.
// This is because the original and the modified source are shown after
// the "Replacing" message. Since these are shown in diff style, they
// must be kept together. And since the "+" line must be below the "Replacing"
// line, this order of lines seems to be the most intuitive.
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

// See Test__show_source_separator_show_autofix for the ordering of the
// output lines.
//
// TODO: Giving the diagnostics again would be useful, but the warning and
// error counters should not be affected, as well as the exitcode.
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

	// Neither the warning nor the corresponding explanation are logged.
	line.Warnf("Filtered warning.")
	G.Explain("Explanation for the above warning.")

	line.Notef("What an interesting line.")
	G.Explain("This explanation is logged.")

	t.CheckOutputLines(
		"NOTE: Makefile:27: What an interesting line.",
		"",
		"\tThis explanation is logged.",
		"")
}

func (s *Suite) Test_SeparatorWriter(c *check.C) {
	var sb strings.Builder
	wr := NewSeparatorWriter(&sb)

	wr.WriteLine("a")
	wr.WriteLine("b")

	c.Check(sb.String(), equals, "a\nb\n")

	wr.Separate()

	c.Check(sb.String(), equals, "a\nb\n")

	wr.WriteLine("c")

	c.Check(sb.String(), equals, "a\nb\n\nc\n")
}

func (s *Suite) Test_SeparatorWriter_Printf(c *check.C) {
	var sb strings.Builder
	wr := NewSeparatorWriter(&sb)

	wr.Printf("a")
	wr.Printf("b")

	c.Check(sb.String(), equals, "ab")

	wr.Separate()

	// The current line is terminated immediately, but the empty line for
	// separating two paragraphs is kept in mind. It will be added later,
	// before the next non-newline character.
	c.Check(sb.String(), equals, "ab\n")

	wr.Printf("c")

	c.Check(sb.String(), equals, "ab\n\nc")
}

func (s *Suite) Test_SeparatorWriter_Separate(c *check.C) {
	var sb strings.Builder
	wr := NewSeparatorWriter(&sb)

	wr.WriteLine("a")
	wr.Separate()

	c.Check(sb.String(), equals, "a\n")

	// The call to Separate had requested an empty line. That empty line
	// can either be given explicitly (like here), or it will be written
	// implicitly before the next non-newline character.
	wr.WriteLine("")
	wr.Separate()

	c.Check(sb.String(), equals, "a\n\n")

	wr.WriteLine("c")
	wr.Separate()

	c.Check(sb.String(), equals, "a\n\nc\n")
}
