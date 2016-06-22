package main

import (
	"encoding/json"
	"gopkg.in/check.v1"
	"strings"
)

func (s *Suite) Test_ShellYacc_Program(c *check.C) {
	testParser := func(program string, expected *MkShList) {
		lexer := &ShellLexer{
			tokens:         strings.Split(program, " "),
			atCommandStart: true,
			error:          ""}
		parser := &shyyParserImpl{}

		succeeded := parser.Parse(lexer)

		if c.Check(succeeded, equals, 0) && c.Check(lexer.error, equals, "") {
			if !c.Check(parser.stack[1].List, deepEquals, expected) {
				actualJson, actualErr := json.Marshal(parser.stack[1].List)
				expectedJson, expectedErr := json.Marshal(expected)
				if c.Check(actualErr, check.IsNil) && c.Check(expectedErr, check.IsNil) {
					c.Check(string(actualJson), deepEquals, string(expectedJson))
				}
			}
		}
	}

	testParser("echo hello, world",
		NewMkShList().AddSimple(NewSimpleCommandBuilder(c).Name("echo").Arg("hello,").Arg("world").Build()))

	testParser("if true ; then echo yes ; else echo no ; fi",
		NewMkShList().AddAndOr(
			NewMkShAndOr(
				NewMkShPipeline(false,
					&MkShCommand{Compound: &MkShCompoundCommand{
						If: &MkShIfClause{
							Conds:   []*MkShList{NewMkShList().AddSimple(NewSimpleCommandBuilder(c).Name("true").Build()).AddSeparator(";")},
							Actions: []*MkShList{NewMkShList().AddSimple(NewSimpleCommandBuilder(c).Name("echo").Arg("yes").Build()).AddSeparator(";")},
							Else:    NewMkShList().AddSimple(NewSimpleCommandBuilder(c).Name("echo").Arg("no").Build()).AddSeparator(";"),
						}}}))))
}

type ShellLexer struct {
	tokens         []string
	atCommandStart bool
	error          string
}

func (lex *ShellLexer) Lex(lval *shyySymType) int {
	if len(lex.tokens) == 0 {
		return 0
	}
	token := lex.tokens[0]
	lex.tokens = lex.tokens[1:]
	switch token {
	case ";":
		lex.atCommandStart = true
		return tkSEMI
	case ";;":
		lex.atCommandStart = true
		return tkSEMISEMI
	case "\n":
		lex.atCommandStart = true
		return tkNEWLINE
	case "&":
		lex.atCommandStart = true
		return tkBACKGROUND
	case "|":
		lex.atCommandStart = true
		return tkPIPE
	case "&&":
		lex.atCommandStart = true
		return tkAND
	case "||":
		lex.atCommandStart = true
		return tkOR
	case ">":
		lex.atCommandStart = false
		return tkGT
	case ">&":
		lex.atCommandStart = false
		return tkGTAND
	case "<":
		lex.atCommandStart = false
		return tkLT
	case "<&":
		lex.atCommandStart = false
		return tkLTAND
	case "<>":
		lex.atCommandStart = false
		return tkLTGT
	case ">>":
		lex.atCommandStart = false
		return tkGTGT
	case "<<":
		lex.atCommandStart = false
		return tkLTLT
	case "<<-":
		lex.atCommandStart = false
		return tkLTLTDASH
	case ">|":
		lex.atCommandStart = false
		return tkGTPIPE
	}
	if lex.atCommandStart {
		switch token {
		case "if":
			return tkIF
		case "then":
			return tkTHEN
		case "elif":
			return tkELIF
		case "else":
			return tkELSE
		case "fi":
			lex.atCommandStart = false
			return tkFI
		case "for":
			lex.atCommandStart = false
			return tkFOR
		case "while":
			return tkWHILE
		case "until":
			return tkUNTIL
		case "do":
			return tkDO
		case "done":
			lex.atCommandStart = false
			return tkDONE
		case "in":
			lex.atCommandStart = false
			return tkIN
		case "case":
			lex.atCommandStart = false
			return tkCASE
		case "esac":
			lex.atCommandStart = false
			return tkESAC
		case "(":
			return tkLPAREN
		case ")":
			lex.atCommandStart = false
			return tkRPAREN
		case "{":
			return tkLBRACE
		case "}":
			lex.atCommandStart = false
			return tkRBRACE
		case "!":
			return tkEXCLAM
		}
	}
	//fmt.Printf("word token: %q\n", token)
	lval.Word = (&MkShTester{nil}).Token(token)
	return tkWORD
}

func (lex *ShellLexer) Error(s string) {
	lex.error = s
}
