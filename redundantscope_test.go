package pkglint

import "gopkg.in/check.v1"

// In a single file, five variables get a default value and are later overridden
// with the same value using the five different assignments operators.
func (s *Suite) Test_RedundantScope__single_file_default(c *check.C) {
	t := s.Init(c)

	mklines := t.NewMkLines("file.mk",
		"DEFAULT?=\tvalue",
		"ASSIGN?=\tvalue",
		"APPEND?=\tvalue",
		"EVAL?=\tvalue",
		"SHELL?=\tvalue",
		"",
		"DEFAULT?=\tvalue",
		"ASSIGN=\tvalue",
		"APPEND+=\tvalue",
		"EVAL:=\tvalue",
		"SHELL!=\tvalue")

	mklines.CheckRedundantAssignments(NewRedundantScope())

	t.CheckOutputLines(
		"NOTE: file.mk:7: Default assignment of DEFAULT has no effect because of line 1.",
		"NOTE: file.mk:8: Definition of ASSIGN is redundant because of line 2.")
	// TODO: "4: is overwritten later",
	// TODO: "5: is overwritten later"
}

// In a single file, five variables get assigned are value and are later overridden
// with the same value using the five different assignments operators.
func (s *Suite) Test_RedundantScope__single_file_assign(c *check.C) {
	t := s.Init(c)

	mklines := t.NewMkLines("file.mk",
		"DEFAULT=\tvalue",
		"ASSIGN=\tvalue",
		"APPEND=\tvalue",
		"EVAL=\tvalue",
		"SHELL=\tvalue",
		"",
		"DEFAULT?=\tvalue",
		"ASSIGN=\tvalue",
		"APPEND+=\tvalue",
		"EVAL:=\tvalue",
		"SHELL!=\tvalue")

	mklines.CheckRedundantAssignments(NewRedundantScope())

	t.CheckOutputLines(
		"NOTE: file.mk:7: Default assignment of DEFAULT has no effect because of line 1.",
		"NOTE: file.mk:8: Definition of ASSIGN is redundant because of line 2.")
	// TODO: "4: is overwritten later",
	// TODO: "5: is overwritten later"
}

// In a single file, five variables get appended a value and are later overridden
// with the same value using the five different assignments operators.
func (s *Suite) Test_RedundantScope__single_file_append(c *check.C) {
	t := s.Init(c)

	mklines := t.NewMkLines("file.mk",
		"DEFAULT+=\tvalue",
		"ASSIGN+=\tvalue",
		"APPEND+=\tvalue",
		"EVAL+=\tvalue",
		"SHELL+=\tvalue",
		"",
		"DEFAULT?=\tvalue",
		"ASSIGN=\tvalue",
		"APPEND+=\tvalue",
		"EVAL:=\tvalue",
		"SHELL!=\tvalue")

	mklines.CheckRedundantAssignments(NewRedundantScope())

	t.CheckOutputLines(
		"NOTE: file.mk:7: Default assignment of DEFAULT has no effect because of line 1.",
		"WARN: file.mk:2: Variable ASSIGN is overwritten in line 8.")
	// TODO: "4: is overwritten later",
	// TODO: "5: is overwritten later"
}

// In a single file, five variables get assigned a value using the := operator,
// which in this simple case is equivalent to the = operator. The variables are
// later overridden with the same value using the five different assignments operators.
func (s *Suite) Test_RedundantScope__single_file_eval(c *check.C) {
	t := s.Init(c)

	mklines := t.NewMkLines("file.mk",
		"DEFAULT:=\tvalue",
		"ASSIGN:=\tvalue",
		"APPEND:=\tvalue",
		"EVAL:=\tvalue",
		"SHELL:=\tvalue",
		"",
		"DEFAULT?=\tvalue",
		"ASSIGN=\tvalue",
		"APPEND+=\tvalue",
		"EVAL:=\tvalue",
		"SHELL!=\tvalue")

	mklines.CheckRedundantAssignments(NewRedundantScope())

	t.CheckOutputLines(
		"NOTE: file.mk:7: Default assignment of DEFAULT has no effect because of line 1.",
		"NOTE: file.mk:8: Definition of ASSIGN is redundant because of line 2.")
	// TODO: "4: is overwritten later",
	// TODO: "5: is overwritten later"
}

// In a single file, five variables get assigned a value using the != operator,
// which runs a shell command. As of March 2019 pkglint doesn't try to evaluate
// the shell commands, therefore the variable values are unknown. The variables
// are later overridden using the five different assignments operators.
func (s *Suite) Test_RedundantScope__single_file_shell(c *check.C) {
	t := s.Init(c)

	mklines := t.NewMkLines("file.mk",
		"DEFAULT!=\tvalue",
		"ASSIGN!=\tvalue",
		"APPEND!=\tvalue",
		"EVAL!=\tvalue",
		"SHELL!=\tvalue",
		"",
		"DEFAULT?=\tvalue",
		"ASSIGN=\tvalue",
		"APPEND+=\tvalue",
		"EVAL:=\tvalue",
		"SHELL!=\tvalue")

	mklines.CheckRedundantAssignments(NewRedundantScope())

	t.CheckOutputLines(
		"NOTE: file.mk:7: Default assignment of DEFAULT has no effect because of line 1.",
		"WARN: file.mk:2: Variable ASSIGN is overwritten in line 8.")
	// TODO: "4: is overwritten later",
	// TODO: "5: is overwritten later"
}

// In a single file, five variables get a default value and are later overridden
// with the same value using the five different assignments operators.
func (s *Suite) Test_RedundantScope__single_file_default_ref(c *check.C) {
	t := s.Init(c)

	mklines := t.NewMkLines("file.mk",
		"DEFAULT?=\t${OTHER}",
		"ASSIGN?=\t${OTHER}",
		"APPEND?=\t${OTHER}",
		"EVAL?=\t${OTHER}",
		"SHELL?=\t${OTHER}",
		"",
		"DEFAULT?=\t${OTHER}",
		"ASSIGN=\t${OTHER}",
		"APPEND+=\t${OTHER}",
		"EVAL:=\t${OTHER}",
		"SHELL!=\t${OTHER}")

	mklines.CheckRedundantAssignments(NewRedundantScope())

	t.CheckOutputLines(
		"NOTE: file.mk:7: Default assignment of DEFAULT has no effect because of line 1.",
		"NOTE: file.mk:8: Definition of ASSIGN is redundant because of line 2.")
	// TODO: "4: is overwritten later",
	// TODO: "5: is overwritten later"
}

// In a single file, five variables get assigned are value and are later overridden
// with the same value using the five different assignments operators.
func (s *Suite) Test_RedundantScope__single_file_assign_ref(c *check.C) {
	t := s.Init(c)

	mklines := t.NewMkLines("file.mk",
		"DEFAULT=\t${OTHER}",
		"ASSIGN=\t${OTHER}",
		"APPEND=\t${OTHER}",
		"EVAL=\t${OTHER}",
		"SHELL=\t${OTHER}",
		"",
		"DEFAULT?=\t${OTHER}",
		"ASSIGN=\t${OTHER}",
		"APPEND+=\t${OTHER}",
		"EVAL:=\t${OTHER}",
		"SHELL!=\t${OTHER}")

	mklines.CheckRedundantAssignments(NewRedundantScope())

	t.CheckOutputLines(
		"NOTE: file.mk:7: Default assignment of DEFAULT has no effect because of line 1.",
		"NOTE: file.mk:8: Definition of ASSIGN is redundant because of line 2.")
	// TODO: "4: is overwritten later",
	// TODO: "5: is overwritten later"
}

// In a single file, five variables get appended a value and are later overridden
// with the same value using the five different assignments operators.
func (s *Suite) Test_RedundantScope__single_file_append_ref(c *check.C) {
	t := s.Init(c)

	mklines := t.NewMkLines("file.mk",
		"DEFAULT+=\t${OTHER}",
		"ASSIGN+=\t${OTHER}",
		"APPEND+=\t${OTHER}",
		"EVAL+=\t${OTHER}",
		"SHELL+=\t${OTHER}",
		"",
		"DEFAULT?=\t${OTHER}",
		"ASSIGN=\t${OTHER}",
		"APPEND+=\t${OTHER}",
		"EVAL:=\t${OTHER}",
		"SHELL!=\t${OTHER}")

	mklines.CheckRedundantAssignments(NewRedundantScope())

	t.CheckOutputLines(
		"NOTE: file.mk:7: Default assignment of DEFAULT has no effect because of line 1.",
		"WARN: file.mk:2: Variable ASSIGN is overwritten in line 8.")
	// TODO: "4: is overwritten later",
	// TODO: "5: is overwritten later"
}

// In a single file, five variables get assigned a value using the := operator,
// which in this simple case is equivalent to the = operator. The variables are
// later overridden with the same value using the five different assignments operators.
func (s *Suite) Test_RedundantScope__single_file_eval_ref(c *check.C) {
	t := s.Init(c)

	mklines := t.NewMkLines("file.mk",
		"DEFAULT:=\t${OTHER}",
		"ASSIGN:=\t${OTHER}",
		"APPEND:=\t${OTHER}",
		"EVAL:=\t${OTHER}",
		"SHELL:=\t${OTHER}",
		"",
		"DEFAULT?=\t${OTHER}",
		"ASSIGN=\t${OTHER}",
		"APPEND+=\t${OTHER}",
		"EVAL:=\t${OTHER}",
		"SHELL!=\t${OTHER}")

	mklines.CheckRedundantAssignments(NewRedundantScope())

	t.CheckOutputLines(
		"NOTE: file.mk:7: Default assignment of DEFAULT has no effect because of line 1.",
		"NOTE: file.mk:8: Definition of ASSIGN is redundant because of line 2.")
	// TODO: "4: is overwritten later",
	// TODO: "5: is overwritten later"
}

// In a single file, five variables get assigned a value using the != operator,
// which runs a shell command. As of March 2019 pkglint doesn't try to evaluate
// the shell commands, therefore the variable values are unknown. The variables
// are later overridden using the five different assignments operators.
func (s *Suite) Test_RedundantScope__single_file_shell_ref(c *check.C) {
	t := s.Init(c)

	mklines := t.NewMkLines("file.mk",
		"DEFAULT!=\t${OTHER}",
		"ASSIGN!=\t${OTHER}",
		"APPEND!=\t${OTHER}",
		"EVAL!=\t${OTHER}",
		"SHELL!=\t${OTHER}",
		"",
		"DEFAULT?=\t${OTHER}",
		"ASSIGN=\t${OTHER}",
		"APPEND+=\t${OTHER}",
		"EVAL:=\t${OTHER}",
		"SHELL!=\t${OTHER}")

	mklines.CheckRedundantAssignments(NewRedundantScope())

	t.CheckOutputLines(
		"NOTE: file.mk:7: Default assignment of DEFAULT has no effect because of line 1.",
		"WARN: file.mk:2: Variable ASSIGN is overwritten in line 8.")
	// TODO: "4: is overwritten later",
	// TODO: "5: is overwritten later"
}

func (s *Suite) Test_RedundantScope__after_including_same_value(c *check.C) {
	t := s.Init(c)

	// Only test the ?=, = and += operators since the others are ignored,
	// as of March 2019.
	t.CreateFileLines("included.mk",
		"VAR.def.def?= ${OTHER}",
		"VAR.def.asg?= ${OTHER}",
		"VAR.def.app?= ${OTHER}",
		"VAR.asg.def=  ${OTHER}",
		"VAR.asg.asg=  ${OTHER}",
		"VAR.asg.app=  ${OTHER}",
		"VAR.app.def+= ${OTHER}",
		"VAR.app.asg+= ${OTHER}",
		"VAR.app.app+= ${OTHER}")
	t.CreateFileLines("including.mk",
		".include \"included.mk\"",
		"VAR.def.def?= ${OTHER}",
		"VAR.def.asg=  ${OTHER}",
		"VAR.def.app+= ${OTHER}",
		"VAR.asg.def?= ${OTHER}",
		"VAR.asg.asg=  ${OTHER}",
		"VAR.asg.app+= ${OTHER}",
		"VAR.app.def?= ${OTHER}",
		"VAR.app.asg=  ${OTHER}",
		"VAR.app.app+= ${OTHER}")
	mklines := t.LoadMkInclude("including.mk")

	mklines.CheckRedundantAssignments(NewRedundantScope())

	t.CheckOutputLines(
		"NOTE: ~/including.mk:2: Default assignment of VAR.def.def has no effect because of included.mk:1.",
		"NOTE: ~/including.mk:3: Definition of VAR.def.asg is redundant because of included.mk:2.",
		// In line 4, defining a default value and then appending to it is a common pattern.
		// Appending the same value feels redundant but probably doesn't happen in practice.
		// If it does, there should be a note for it.
		"NOTE: ~/including.mk:5: Default assignment of VAR.asg.def has no effect because of included.mk:4.",
		"NOTE: ~/including.mk:6: Definition of VAR.asg.asg is redundant because of included.mk:5.",
		// In line 7, setting a variable and later appending to it is a common pattern.
		// Appending the same value feels redundant but probably doesn't happen in practice.
		// If it does, there should be a note for it.
		"NOTE: ~/including.mk:8: Default assignment of VAR.app.def has no effect because of included.mk:7.",
		// In line 9, overwriting an appended variable might be a mistake.
		// TODO: Find out whether this case happens in actual pkgsrc and if it's accidental.
		// In line 10, appending further is a common pattern.
	)
}

func (s *Suite) Test_RedundantScope__after_including_different_value(c *check.C) {
	t := s.Init(c)

	// Only test the ?=, = and += operators since the others are ignored,
	// as of March 2019.
	t.CreateFileLines("included.mk",
		"VAR.def.def?= ${VALUE}",
		"VAR.def.asg?= ${VALUE}",
		"VAR.def.app?= ${VALUE}",
		"VAR.asg.def=  ${VALUE}",
		"VAR.asg.asg=  ${VALUE}",
		"VAR.asg.app=  ${VALUE}",
		"VAR.app.def+= ${VALUE}",
		"VAR.app.asg+= ${VALUE}",
		"VAR.app.app+= ${VALUE}")
	t.CreateFileLines("including.mk",
		".include \"included.mk\"",
		"VAR.def.def?= ${OTHER}",
		"VAR.def.asg=  ${OTHER}",
		"VAR.def.app+= ${OTHER}",
		"VAR.asg.def?= ${OTHER}",
		"VAR.asg.asg=  ${OTHER}",
		"VAR.asg.app+= ${OTHER}",
		"VAR.app.def?= ${OTHER}",
		"VAR.app.asg=  ${OTHER}",
		"VAR.app.app+= ${OTHER}")
	mklines := t.LoadMkInclude("including.mk")

	mklines.CheckRedundantAssignments(NewRedundantScope())

	t.CheckOutputLines(
		"NOTE: ~/including.mk:2: Default assignment of VAR.def.def has no effect because of included.mk:1.",
		"NOTE: ~/including.mk:5: Default assignment of VAR.asg.def has no effect because of included.mk:4.",
		"NOTE: ~/including.mk:8: Default assignment of VAR.app.def has no effect because of included.mk:7.")
}

func (s *Suite) Test_RedundantScope__before_including_same_value(c *check.C) {
	t := s.Init(c)

	// Only test the ?=, = and += operators since the others are ignored,
	// as of March 2019.
	t.CreateFileLines("including.mk",
		"VAR.def.def?= ${OTHER}",
		"VAR.def.asg?= ${OTHER}",
		"VAR.def.app?= ${OTHER}",
		"VAR.asg.def=  ${OTHER}",
		"VAR.asg.asg=  ${OTHER}",
		"VAR.asg.app=  ${OTHER}",
		"VAR.app.def+= ${OTHER}",
		"VAR.app.asg+= ${OTHER}",
		"VAR.app.app+= ${OTHER}",
		".include \"included.mk\"")
	t.CreateFileLines("included.mk",
		"VAR.def.def?= ${OTHER}",
		"VAR.def.asg=  ${OTHER}",
		"VAR.def.app+= ${OTHER}",
		"VAR.asg.def?= ${OTHER}",
		"VAR.asg.asg=  ${OTHER}",
		"VAR.asg.app+= ${OTHER}",
		"VAR.app.def?= ${OTHER}",
		"VAR.app.asg=  ${OTHER}",
		"VAR.app.app+= ${OTHER}")
	mklines := t.LoadMkInclude("including.mk")

	mklines.CheckRedundantAssignments(NewRedundantScope())

	t.CheckOutputLines(
		"NOTE: ~/including.mk:1: Default assignment of VAR.def.def has no effect because of included.mk:1.",
		"NOTE: ~/including.mk:2: Default assignment of VAR.def.asg has no effect because of included.mk:2.",
		"NOTE: ~/including.mk:4: Definition of VAR.asg.def is redundant because of included.mk:4.",
		"NOTE: ~/including.mk:5: Definition of VAR.asg.asg is redundant because of included.mk:5.",
		"WARN: ~/including.mk:8: Variable VAR.app.asg is overwritten in included.mk:8.")
}

func (s *Suite) Test_RedundantScope__before_including_different_value(c *check.C) {
	t := s.Init(c)

	// Only test the ?=, = and += operators since the others are ignored,
	// as of March 2019.
	t.CreateFileLines("including.mk",
		"VAR.def.def?= ${VALUE}",
		"VAR.def.asg?= ${VALUE}",
		"VAR.def.app?= ${VALUE}",
		"VAR.asg.def=  ${VALUE}",
		"VAR.asg.asg=  ${VALUE}",
		"VAR.asg.app=  ${VALUE}",
		"VAR.app.def+= ${VALUE}",
		"VAR.app.asg+= ${VALUE}",
		"VAR.app.app+= ${VALUE}",
		".include \"included.mk\"")
	t.CreateFileLines("included.mk",
		"VAR.def.def?= ${OTHER}",
		"VAR.def.asg=  ${OTHER}",
		"VAR.def.app+= ${OTHER}",
		"VAR.asg.def?= ${OTHER}",
		"VAR.asg.asg=  ${OTHER}",
		"VAR.asg.app+= ${OTHER}",
		"VAR.app.def?= ${OTHER}",
		"VAR.app.asg=  ${OTHER}",
		"VAR.app.app+= ${OTHER}")
	mklines := t.LoadMkInclude("including.mk")

	mklines.CheckRedundantAssignments(NewRedundantScope())

	t.CheckOutputLines(
		"WARN: ~/including.mk:2: Variable VAR.def.asg is overwritten in included.mk:2.",
		"WARN: ~/including.mk:5: Variable VAR.asg.asg is overwritten in included.mk:5.",
		"WARN: ~/including.mk:8: Variable VAR.app.asg is overwritten in included.mk:8.")
}

func (s *Suite) Test_RedundantScope__independent_same_value(c *check.C) {
	t := s.Init(c)

	// Only test the ?=, = and += operators since the others are ignored,
	// as of March 2019.
	t.CreateFileLines("including.mk",
		".include \"included1.mk\"",
		".include \"included2.mk\"")
	t.CreateFileLines("included1.mk",
		"VAR.def.def?= ${OTHER}",
		"VAR.def.asg?= ${OTHER}",
		"VAR.def.app?= ${OTHER}",
		"VAR.asg.def=  ${OTHER}",
		"VAR.asg.asg=  ${OTHER}",
		"VAR.asg.app=  ${OTHER}",
		"VAR.app.def+= ${OTHER}",
		"VAR.app.asg+= ${OTHER}",
		"VAR.app.app+= ${OTHER}")
	t.CreateFileLines("included2.mk",
		"VAR.def.def?= ${OTHER}",
		"VAR.def.asg=  ${OTHER}",
		"VAR.def.app+= ${OTHER}",
		"VAR.asg.def?= ${OTHER}",
		"VAR.asg.asg=  ${OTHER}",
		"VAR.asg.app+= ${OTHER}",
		"VAR.app.def?= ${OTHER}",
		"VAR.app.asg=  ${OTHER}",
		"VAR.app.app+= ${OTHER}")
	mklines := t.LoadMkInclude("including.mk")

	mklines.CheckRedundantAssignments(NewRedundantScope())

	// Since the two included files are independent, there cannot be any
	// redundancies between them. These redundancies can only be discovered
	// when one of them includes the other.
	t.CheckOutputEmpty()
}

func (s *Suite) Test_RedundantScope__independent_different_value(c *check.C) {
	t := s.Init(c)

	// Only test the ?=, = and += operators since the others are ignored,
	// as of March 2019.
	t.CreateFileLines("including.mk",
		".include \"included1.mk\"",
		".include \"included2.mk\"")
	t.CreateFileLines("included1.mk",
		"VAR.def.def?= ${VALUE}",
		"VAR.def.asg?= ${VALUE}",
		"VAR.def.app?= ${VALUE}",
		"VAR.asg.def=  ${VALUE}",
		"VAR.asg.asg=  ${VALUE}",
		"VAR.asg.app=  ${VALUE}",
		"VAR.app.def+= ${VALUE}",
		"VAR.app.asg+= ${VALUE}",
		"VAR.app.app+= ${VALUE}")
	t.CreateFileLines("included2.mk",
		"VAR.def.def?= ${OTHER}",
		"VAR.def.asg=  ${OTHER}",
		"VAR.def.app+= ${OTHER}",
		"VAR.asg.def?= ${OTHER}",
		"VAR.asg.asg=  ${OTHER}",
		"VAR.asg.app+= ${OTHER}",
		"VAR.app.def?= ${OTHER}",
		"VAR.app.asg=  ${OTHER}",
		"VAR.app.app+= ${OTHER}")
	mklines := t.LoadMkInclude("including.mk")

	mklines.CheckRedundantAssignments(NewRedundantScope())

	// Since the two included files are independent, there cannot be any
	// redundancies between them. Redundancies can only be discovered
	// when one of them includes the other.
	t.CheckOutputEmpty()
}

// FIXME: Continue the systematic redundancy tests.
//
// Tests involving variables that are defined and overwritten in independent files.
// including.mk
// +-- included1.mk
// `-- included2.mk
//
// Tests involving several files, both dependent and independent.
// This is to test includePath.includesAll and includePath.includesAny.
//
// A test where the operators = and += define a variable that afterwards
// is assigned the same value using the ?= operator.
//
// Tests where the variables refer to other variables. These variables may
// be read and written between the relevant assignments.
//
// Tests where the variables are defined conditionally using .if, .else, .endif.
//
// Tests where the variables are defined in a .for loop that might not be
// evaluated at all.
//
// Tests where files are included conditionally and additionally have conditional
// sections, arbitrarily nested.
//
// Tests that show how to suppress the notes about redundant assignments
// and overwritten variables. The explanation must be helpful.

func (s *Suite) Test_RedundantScope__override_after_including(c *check.C) {
	t := s.Init(c)
	t.CreateFileLines("included.mk",
		"OVERRIDE=\tprevious value",
		"REDUNDANT=\tredundant")
	t.CreateFileLines("including.mk",
		".include \"included.mk\"",
		"OVERRIDE=\toverridden value",
		"REDUNDANT=\tredundant")
	t.Chdir(".")
	mklines := t.LoadMkInclude("including.mk")

	// XXX: The warnings from here are not in the same order as the other warnings.
	// XXX: There may be some warnings for the same file separated by warnings for other files.
	mklines.CheckRedundantAssignments(NewRedundantScope())

	t.CheckOutputLines(
		"NOTE: including.mk:3: Definition of REDUNDANT is redundant because of included.mk:2.")
}

func (s *Suite) Test_RedundantScope__redundant_assign_after_including(c *check.C) {
	t := s.Init(c)
	t.CreateFileLines("included.mk",
		"REDUNDANT=\tredundant")
	t.CreateFileLines("including.mk",
		".include \"included.mk\"",
		"REDUNDANT=\tredundant")
	t.Chdir(".")
	mklines := t.LoadMkInclude("including.mk")

	mklines.CheckRedundantAssignments(NewRedundantScope())

	t.CheckOutputLines(
		"NOTE: including.mk:2: Definition of REDUNDANT is redundant because of included.mk:1.")
}

func (s *Suite) Test_RedundantScope__override_in_Makefile_after_including(c *check.C) {
	t := s.Init(c)
	t.CreateFileLines("module.mk",
		"VAR=\tvalue ${OTHER}",
		"VAR?=\tvalue ${OTHER}",
		"VAR=\tnew value")
	t.CreateFileLines("Makefile",
		".include \"module.mk\"",
		"VAR=\tthe package may overwrite variables from other files")
	t.Chdir(".")

	mklines := t.LoadMkInclude("Makefile")

	// XXX: The warnings from here are not in the same order as the other warnings.
	// XXX: There may be some warnings for the same file separated by warnings for other files.
	mklines.CheckRedundantAssignments(NewRedundantScope())

	// No warning for VAR=... in Makefile since it makes sense to have common files
	// with default values for variables, overriding some of them in each package.
	t.CheckOutputLines(
		"NOTE: module.mk:2: Default assignment of VAR has no effect because of line 1.",
		"WARN: module.mk:2: Variable VAR is overwritten in line 3.")
}

func (s *Suite) Test_RedundantScope__default_value_definitely_unused(c *check.C) {
	t := s.Init(c)
	mklines := t.NewMkLines("module.mk",
		"VAR=\tvalue ${OTHER}",
		"VAR?=\tdifferent value")

	mklines.CheckRedundantAssignments(NewRedundantScope())

	// A default assignment after an unconditional assignment is redundant.
	// Even more so when the variable is not used between the two assignments.
	t.CheckOutputLines(
		"NOTE: module.mk:2: Default assignment of VAR has no effect because of line 1.")
}

func (s *Suite) Test_RedundantScope__default_value_overridden(c *check.C) {
	t := s.Init(c)
	mklines := t.NewMkLines("module.mk",
		"VAR?=\tdefault value",
		"VAR=\toverridden value")

	mklines.CheckRedundantAssignments(NewRedundantScope())

	t.CheckOutputLines(
		"WARN: module.mk:1: Variable VAR is overwritten in line 2.")
}

func (s *Suite) Test_RedundantScope__overwrite_same_value(c *check.C) {
	t := s.Init(c)
	mklines := t.NewMkLines("module.mk",
		"VAR=\tvalue ${OTHER}",
		"VAR=\tvalue ${OTHER}")

	mklines.CheckRedundantAssignments(NewRedundantScope())

	t.CheckOutputLines(
		"NOTE: module.mk:2: Definition of VAR is redundant because of line 1.")
}

func (s *Suite) Test_RedundantScope__conditional_overwrite(c *check.C) {
	t := s.Init(c)
	mklines := t.NewMkLines("module.mk",
		"VAR=\tdefault",
		".if ${OPSYS} == NetBSD",
		"VAR=\topsys",
		".endif")

	mklines.CheckRedundantAssignments(NewRedundantScope())

	t.CheckOutputEmpty()
}

func (s *Suite) Test_RedundantScope__overwrite_inside_conditional(c *check.C) {
	t := s.Init(c)
	mklines := t.NewMkLines("module.mk",
		"VAR=\tgeneric",
		".if ${OPSYS} == NetBSD",
		"VAR=\tignored",
		"VAR=\toverwritten",
		".endif")

	mklines.CheckRedundantAssignments(NewRedundantScope())

	// TODO: expected a warning "WARN: module.mk:4: line 3 is ignored"
	// Since line 3 and line 4 are in the same basic block, line 3 is definitely ignored.
	t.CheckOutputEmpty()
}

func (s *Suite) Test_RedundantScope__conditionally_include(c *check.C) {
	t := s.Init(c)
	t.CreateFileLines("module.mk",
		"VAR=\tgeneric",
		".if ${OPSYS} == NetBSD",
		".  include \"included.mk\"",
		".endif")
	t.CreateFileLines("included.mk",
		"VAR=\tignored",
		"VAR=\toverwritten")
	mklines := t.LoadMkInclude("module.mk")

	mklines.CheckRedundantAssignments(NewRedundantScope())

	// TODO: expected a warning "WARN: module.mk:4: line 3 is ignored"
	//  Since line 3 and line 4 are in the same basic block, line 3 is definitely ignored.
	t.CheckOutputEmpty()
}

func (s *Suite) Test_RedundantScope__conditional_default(c *check.C) {
	t := s.Init(c)
	mklines := t.NewMkLines("module.mk",
		"VAR=\tdefault",
		".if ${OPSYS} == NetBSD",
		"VAR?=\topsys",
		".endif")

	mklines.CheckRedundantAssignments(NewRedundantScope())

	// TODO: WARN: module.mk:3: The value \"opsys\" will never be assigned to VAR because it is defined unconditionally in line 1.
	t.CheckOutputEmpty()
}

// These warnings are precise and accurate since the value of VAR is not used between line 2 and 4.
func (s *Suite) Test_RedundantScope__overwrite_same_variable_different_value(c *check.C) {
	t := s.Init(c)
	mklines := t.NewMkLines("module.mk",
		"OTHER=\tvalue before",
		"VAR=\tvalue ${OTHER}",
		"OTHER=\tvalue after",
		"VAR=\tvalue ${OTHER}")

	mklines.CheckRedundantAssignments(NewRedundantScope())

	// Strictly speaking, line 1 is redundant because OTHER is not evaluated
	// at load time and then immediately overwritten in line 3. If the operator
	// in line 2 were a := instead of a =, the situation would be clear.
	// Pkglint doesn't warn about the redundancy in line 1 because it prefers
	// to omit warnings instead of giving wrong advice.
	t.CheckOutputLines(
		"NOTE: module.mk:4: Definition of VAR is redundant because of line 2.")
}

func (s *Suite) Test_RedundantScope__overwrite_different_value_used_between(c *check.C) {
	t := s.Init(c)
	mklines := t.NewMkLines("module.mk",
		"OTHER=\tvalue before",
		"VAR=\tvalue ${OTHER}",

		// VAR is used here at load time, therefore it must be defined at this point.
		// At this point, VAR uses the \"before\" value of OTHER.
		"RESULT1:=\t${VAR}",

		"OTHER=\tvalue after",

		// VAR is used here again at load time, this time using the \"after\" value of OTHER.
		"RESULT2:=\t${VAR}",

		// Still this definition is redundant.
		"VAR=\tvalue ${OTHER}")

	mklines.CheckRedundantAssignments(NewRedundantScope())

	// There is nothing redundant here. Each write is followed by a
	// corresponding read, except for the last one. That is ok though
	// because in pkgsrc the last action of a package is to include
	// bsd.pkg.mk, which reads almost all variables.
	t.CheckOutputEmpty()
}

func (s *Suite) Test_RedundantScope__procedure_call(c *check.C) {
	t := s.Init(c)
	mklines := t.NewMkLines("mk/pthread.buildlink3.mk",
		"CHECK_BUILTIN.pthread:=\tyes",
		".include \"../../mk/pthread.builtin.mk\"",
		"CHECK_BUILTIN.pthread:=\tno")

	mklines.CheckRedundantAssignments(NewRedundantScope())

	t.CheckOutputEmpty()
}

func (s *Suite) Test_RedundantScope__procedure_call_implemented(c *check.C) {
	t := s.Init(c)

	t.SetUpPkgsrc()
	t.SetUpPackage("devel/gettext-lib")
	t.SetUpPackage("x11/Xaos",
		".include \"../../devel/gettext-lib/buildlink3.mk\"")
	t.CreateFileLines("devel/gettext-lib/builtin.mk",
		MkRcsID,
		"",
		".include \"../../mk/bsd.fast.prefs.mk\"",
		"",
		"CHECK_BUILTIN.gettext?=\tno",
		".if !empty(CHECK_BUILTIN.gettext:M[nN][oO])",
		".endif")
	t.CreateFileLines("devel/gettext-lib/buildlink3.mk",
		MkRcsID,
		"CHECK_BUILTIN.gettext:=\tyes",
		".include \"builtin.mk\"",
		"CHECK_BUILTIN.gettext:=\tno")
	G.Pkgsrc.LoadInfrastructure()

	// Checking x11/Xaos instead of devel/gettext-lib avoids warnings
	// about the minimal buildlink3.mk file.
	G.Check(t.File("x11/Xaos"))

	// There is nothing redundant here.
	// Up to March 2019, pkglint didn't pass the correct pathnames to Package.included,
	// which triggered a wrong note here.
	t.CheckOutputEmpty()
}

func (s *Suite) Test_RedundantScope__shell_and_eval(c *check.C) {
	t := s.Init(c)
	mklines := t.NewMkLines("module.mk",
		"VAR:=\tvalue ${OTHER}",
		"VAR!=\tvalue ${OTHER}")

	mklines.CheckRedundantAssignments(NewRedundantScope())

	// As of November 2018, pkglint doesn't check redundancies that involve the := or != operators.
	//
	// What happens here is:
	//
	// Line 1 evaluates OTHER at load time.
	// Line 1 assigns its value to VAR.
	// Line 2 evaluates OTHER at load time.
	// Line 2 passes its value through the shell and assigns the result to VAR.
	//
	// Since VAR is defined in line 1, not used afterwards and overwritten in line 2, it is redundant.
	// Well, not quite, because evaluating ${OTHER} might have side-effects from :sh or ::= modifiers,
	// but these are so rare that they are frowned upon and are not considered by pkglint.
	//
	// Expected result:
	// WARN: module.mk:2: Previous definition of VAR in line 1 is unused.

	t.CheckOutputEmpty()
}

func (s *Suite) Test_RedundantScope__shell_and_eval_literal(c *check.C) {
	t := s.Init(c)
	mklines := t.NewMkLines("module.mk",
		"VAR:=\tvalue",
		"VAR!=\tvalue")

	mklines.CheckRedundantAssignments(NewRedundantScope())

	// Even when := is used with a literal value (which is usually
	// only done for procedure calls), the shell evaluation can have
	// so many different side effects that pkglint cannot reliably
	// help in this situation.
	//
	// TODO: Why not? The evaluation in line 1 is trivial to analyze.
	t.CheckOutputEmpty()
}

func (s *Suite) Test_RedundantScope__included_OPSYS_variable(c *check.C) {
	t := s.Init(c)

	t.SetUpPackage("category/package",
		".include \"../../category/dependency/buildlink3.mk\"",
		"CONFIGURE_ARGS+=\tone",
		"CONFIGURE_ARGS=\ttwo",
		"CONFIGURE_ARGS+=\tthree")
	t.SetUpPackage("category/dependency")
	t.CreateFileDummyBuildlink3("category/dependency/buildlink3.mk")
	t.CreateFileLines("category/dependency/builtin.mk",
		MkRcsID,
		"CONFIGURE_ARGS.Darwin+=\tdarwin")

	G.Check(t.File("category/package"))

	t.CheckOutputLines(
		"WARN: ~/category/package/Makefile:21: Variable CONFIGURE_ARGS is overwritten in line 22.")
}

func (s *Suite) Test_RedundantScope__if_then_else(c *check.C) {
	t := s.Init(c)

	mklines := t.SetUpFileMkLines("if-then-else.mk",
		MkRcsID,
		".if exists(${FILE})",
		"OS=\tNetBSD",
		".else",
		"OS=\tOTHER",
		".endif")

	mklines.CheckRedundantAssignments(NewRedundantScope())

	// These two definitions are of course not redundant since they happen in
	// different branches of the same .if statement.
	t.CheckOutputEmpty()
}

func (s *Suite) Test_RedundantScope__if_then_else_without_variable(c *check.C) {
	t := s.Init(c)

	mklines := t.SetUpFileMkLines("if-then-else.mk",
		MkRcsID,
		".if exists(/nonexistent)",
		"IT=\texists",
		".else",
		"IT=\tdoesn't exist",
		".endif")

	mklines.CheckRedundantAssignments(NewRedundantScope())

	// These two definitions are of course not redundant since they happen in
	// different branches of the same .if statement.
	// Even though the .if condition does not refer to any variables,
	// this still means that the variable assignments are conditional.
	t.CheckOutputEmpty()
}

func (s *Suite) Test_RedundantScope__append_then_default(c *check.C) {
	t := s.Init(c)

	mklines := t.SetUpFileMkLines("append-then-default.mk",
		MkRcsID,
		"VAR+=\tvalue",
		"VAR?=\tvalue")

	mklines.CheckRedundantAssignments(NewRedundantScope())

	t.CheckOutputLines(
		"NOTE: ~/append-then-default.mk:3: Default assignment of VAR has no effect because of line 2.")
}

func (s *Suite) Test_RedundantScope__assign_then_default_in_same_file(c *check.C) {
	t := s.Init(c)

	mklines := t.SetUpFileMkLines("assign-then-default.mk",
		MkRcsID,
		"VAR=\tvalue",
		"VAR?=\tvalue")

	mklines.CheckRedundantAssignments(NewRedundantScope())

	t.CheckOutputLines(
		"NOTE: ~/assign-then-default.mk:3: " +
			"Default assignment of VAR has no effect because of line 2.")
}

func (s *Suite) Test_RedundantScope__eval_then_eval(c *check.C) {
	t := s.Init(c)

	mklines := t.SetUpFileMkLines("filename.mk",
		MkRcsID,
		"VAR:=\tvalue",
		"VAR:=\tvalue",
		"VAR:=\tother")

	mklines.CheckRedundantAssignments(NewRedundantScope())

	// TODO: Add redundancy check for the := operator.
	t.CheckOutputEmpty()
}

func (s *Suite) Test_RedundantScope__shell_then_assign(c *check.C) {
	t := s.Init(c)

	mklines := t.SetUpFileMkLines("filename.mk",
		MkRcsID,
		"VAR!=\techo echo",
		"VAR=\techo echo")

	mklines.CheckRedundantAssignments(NewRedundantScope())

	// Although the two variable assignments look very similar, they do
	// something entirely different. The first executes the echo command,
	// and the second just assigns a string. Therefore the actual variable
	// values are different, and the second assignment is not redundant.
	// It assigns a different value. Nevertheless, the shell command is
	// redundant and can be removed since its result is never used.
	t.CheckOutputLines(
		"WARN: ~/filename.mk:2: Variable VAR is overwritten in line 3.")
}

func (s *Suite) Test_RedundantScope__shell_then_read_then_assign(c *check.C) {
	t := s.Init(c)

	mklines := t.SetUpFileMkLines("filename.mk",
		MkRcsID,
		"VAR!=\techo echo",
		"OUTPUT:=${VAR}",
		"VAR=\techo echo")

	mklines.CheckRedundantAssignments(NewRedundantScope())

	// No warning since the value is used in-between.
	t.CheckOutputEmpty()
}

func (s *Suite) Test_RedundantScope__assign_then_default_in_included_file(c *check.C) {
	t := s.Init(c)

	t.CreateFileLines("assign-then-default.mk",
		MkRcsID,
		"VAR=\tvalue",
		".include \"included.mk\"")
	t.CreateFileLines("included.mk",
		MkRcsID,
		"VAR?=\tvalue")
	mklines := t.LoadMkInclude("assign-then-default.mk")

	mklines.CheckRedundantAssignments(NewRedundantScope())

	// If assign-then-default.mk:2 is deleted, VAR still has the same value.
	t.CheckOutputLines(
		"NOTE: ~/assign-then-default.mk:2: Definition of VAR is redundant because of included.mk:2.")
}

func (s *Suite) Test_RedundantScope__conditionally_included_file(c *check.C) {
	t := s.Init(c)

	t.CreateFileLines("including.mk",
		MkRcsID,
		"VAR=\tvalue",
		".if ${COND}",
		".  include \"included.mk\"",
		".endif")
	t.CreateFileLines("included.mk",
		MkRcsID,
		"VAR?=\tvalue")
	mklines := t.LoadMkInclude("including.mk")

	mklines.CheckRedundantAssignments(NewRedundantScope())

	// The assignment in including.mk:2 is only redundant if included.mk is actually included.
	// Therefore both included.mk:2 nor including.mk:2 are relevant.
	t.CheckOutputEmpty()
}

func (s *Suite) Test_RedundantScope__procedure_parameters(c *check.C) {
	t := s.Init(c)

	t.CreateFileLines("mk/pkg-build-options.mk",
		MkRcsID,
		"USED:=\t${pkgbase}")
	t.CreateFileLines("including.mk",
		MkRcsID,
		"pkgbase=\tpackage1",
		".include \"mk/pkg-build-options.mk\"",
		"",
		"pkgbase=\tpackage2",
		".include \"mk/pkg-build-options.mk\"",
		"",
		"pkgbase=\tpackage3",
		".include \"mk/pkg-build-options.mk\"")
	mklines := t.LoadMkInclude("including.mk")

	mklines.CheckRedundantAssignments(NewRedundantScope())

	// This variable is not overwritten since it is used in-between
	// by the included file.
	t.CheckOutputEmpty()
}

func (s *Suite) Test_RedundantScope__overwrite_definition_from_included_file(c *check.C) {
	t := s.Init(c)

	t.CreateFileLines("included.mk",
		MkRcsID,
		"WRKSRC=\t${WRKDIR}/${PKGBASE}")
	t.CreateFileLines("including.mk",
		MkRcsID,
		"SUBDIR=\t${WRKSRC}",
		".include \"included.mk\"",
		"WRKSRC=\t${WRKDIR}/overwritten")
	mklines := t.LoadMkInclude("including.mk")

	mklines.CheckRedundantAssignments(NewRedundantScope())

	// Before pkglint 5.7.2 (2019-03-09), including.mk:2 used WRKSRC for the first time.
	// At that point the include path for that variable was fixed once and for all.
	// Later in RedundantScope.handleVarassign, there was a check that was supposed to
	// prevent all warnings in included files, if there was such a relation.
	//
	// In this case no such inclusion hierarchy was visible since the include path of
	// WRKSRC was [including.mk], which was the same as the include path at including.mk:4,
	// therefore the lines were in the same file and the earlier line got the warning.
	//
	// Except that the earlier line was not related in any way to WRKSRC.includePath.
	//
	// This revealed an imprecise handling of these includePaths. They have been changed
	// to be more precise. Now every access to the variable is recorded, and the
	// conditions have been changed to "all of" or "any of", as appropriate.
	t.CheckOutputEmpty()
}

func (s *Suite) Test_RedundantScope_handleVarassign__conditional(c *check.C) {
	t := s.Init(c)

	scope := NewRedundantScope()
	mklines := t.NewMkLines("filename.mk",
		MkRcsID,
		"VAR=\tvalue",
		".if 1",
		"VAR=\tconditional",
		".endif")

	mklines.ForEach(func(mkline MkLine) {
		scope.Handle(mkline, mklines.indentation)
	})

	t.Check(
		scope.get("VAR").vari.WriteLocations(),
		deepEquals,
		[]MkLine{mklines.mklines[1], mklines.mklines[3]})
}

func (s *Suite) Test_includePath_includes(c *check.C) {
	t := s.Init(c)

	path := func(locations ...string) includePath {
		return includePath{locations}
	}

	var (
		m   = path("Makefile")
		mc  = path("Makefile", "Makefile.common")
		mco = path("Makefile", "Makefile.common", "other.mk")
		mo  = path("Makefile", "other.mk")
	)

	t.Check(m.includes(m), equals, false)

	t.Check(m.includes(mc), equals, true)
	t.Check(m.includes(mco), equals, true)
	t.Check(mc.includes(mco), equals, true)

	t.Check(mc.includes(m), equals, false)
	t.Check(mc.includes(mo), equals, false)
	t.Check(mo.includes(mc), equals, false)
}

func (s *Suite) Test_includePath_equals(c *check.C) {
	t := s.Init(c)

	path := func(locations ...string) includePath {
		return includePath{locations}
	}

	var (
		m   = path("Makefile")
		mc  = path("Makefile", "Makefile.common")
		mco = path("Makefile", "Makefile.common", "other.mk")
		mo  = path("Makefile", "other.mk")
	)

	t.Check(m.equals(m), equals, true)

	t.Check(m.equals(mc), equals, false)
	t.Check(m.equals(mco), equals, false)
	t.Check(mc.equals(mco), equals, false)

	t.Check(mc.equals(m), equals, false)
	t.Check(mc.equals(mo), equals, false)
	t.Check(mo.equals(mc), equals, false)
}
