package main

import "gopkg.in/check.v1"

// Since the --source option generates multi-line diagnostics,
// they are separated by an empty line.
//
// The quoted source code is written below the diagnostics.
// In the --show-autofix and --autofix modes, this order
// is the most useful since it first states the general rule,
// then states how to fix this instance and then shows a concrete
// example. Understanding the general rule is considered most
// important of these three.
//
// To keep the output layout consistent between all these
// modes, the source code is written below the diagnostic
// even in the default (check-only) mode, for consistency.
func (s *Suite) Test_show_source_separator(c *check.C) {
	s.Init(c)
	s.UseCommandLine("--source")
	fileName := s.CreateTmpFileLines("DESCR",
		"The first line",
		"The second line",
		"The third line")
	lines := LoadExistingLines(fileName, true)

	fix := lines[1].Autofix()
	fix.Warnf("Using \"second\" is deprecated.")
	fix.Replace("second", "silver medal")
	fix.Apply()

	lines[2].Warnf("Dummy warning.")

	fix = lines[2].Autofix()
	fix.Warnf("Using \"third\" is deprecated.")
	fix.Replace("third", "bronze medal")
	fix.Apply()

	s.CheckOutputLines(
		"WARN: ~/DESCR:2: Using \"second\" is deprecated.",
		"> The second line",
		"",
		"WARN: ~/DESCR:3: Dummy warning.",
		"> The third line",
		"",
		"WARN: ~/DESCR:3: Using \"third\" is deprecated.",
		"> The third line")
}

func (s *Suite) Test_show_source_separator_show_autofix(c *check.C) {
	s.Init(c)
	s.UseCommandLine("--source", "--show-autofix")
	fileName := s.CreateTmpFileLines("DESCR",
		"The first line",
		"The second line",
		"The third line")
	lines := LoadExistingLines(fileName, true)

	fix := lines[1].Autofix()
	fix.Warnf("Using \"second\" is deprecated.")
	fix.Replace("second", "silver medal")
	fix.Apply()

	lines[2].Warnf("Dummy warning.")

	fix = lines[2].Autofix()
	fix.Warnf("Using \"third\" is deprecated.")
	fix.Replace("third", "bronze medal")
	fix.Apply()

	s.CheckOutputLines(
		"WARN: ~/DESCR:2: Using \"second\" is deprecated.",
		"AUTOFIX: ~/DESCR:2: Replacing \"second\" with \"silver medal\".",
		"- The second line",
		"+ The silver medal line",
		"",
		"WARN: ~/DESCR:3: Using \"third\" is deprecated.",
		"AUTOFIX: ~/DESCR:3: Replacing \"third\" with \"bronze medal\".",
		"- The third line",
		"+ The bronze medal line")
}

func (s *Suite) Test_show_source_separator_autofix(c *check.C) {
	s.Init(c)
	s.UseCommandLine("--source", "--autofix")
	fileName := s.CreateTmpFileLines("DESCR",
		"The first line",
		"The second line",
		"The third line")
	lines := LoadExistingLines(fileName, true)

	fix := lines[1].Autofix()
	fix.Warnf("Using \"second\" is deprecated.")
	fix.Replace("second", "silver medal")
	fix.Apply()

	lines[2].Warnf("Dummy warning.")

	fix = lines[2].Autofix()
	fix.Warnf("Using \"third\" is deprecated.")
	fix.Replace("third", "bronze medal")
	fix.Apply()

	s.CheckOutputLines(
		"AUTOFIX: ~/DESCR:2: Replacing \"second\" with \"silver medal\".",
		"- The second line",
		"+ The silver medal line",
		"",
		"AUTOFIX: ~/DESCR:3: Replacing \"third\" with \"bronze medal\".",
		"- The third line",
		"+ The bronze medal line")
}
