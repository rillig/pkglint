package main

import (
	"fmt"
)

type ShCommand struct {
	Varassigns []*ShVarassign
	Command    *ShWord
	Args       []*ShWord
}

func (shcmd *ShCommand) String() string {
	return fmt.Sprintf("ShCommand(%v, %v, %v)", shcmd.Varassigns, shcmd.Command, shcmd.Args)
}

type ShWord struct {
	Atoms []*ShLexeme
}

func (shword *ShWord) String() string {
	return fmt.Sprintf("ShWord(%v)", shword.Atoms)
}

type ShVarassign struct {
	Name  string
	Value *ShWord // maybe
}

func (shva *ShVarassign) String() string {
	return fmt.Sprintf("ShVarassign(%q, %v)", shva.Name, shva.Value)
}

// @Beta
type ShLexeme struct {
	Type    ShLexemeType
	Text    string
	Quoting ShQuoting
	Data    interface{}
}

func (shlex *ShLexeme) String() string {
	if shlex.Type == shlText && shlex.Quoting == "" && shlex.Data == nil {
		return fmt.Sprintf("%q", shlex.Text)
	}
	if shlex.Type == shlVaruse {
		varuse := shlex.Data.(*MkVarUse)
		return fmt.Sprintf("varuse(%q)", varuse.varname+varuse.Mod())
	}
	return fmt.Sprintf("ShLexeme(%v, %q, %q)", shlex.Type, shlex.Text, shlex.Quoting)
}

type ShLexemeType uint8

const (
	shlSpace        ShLexemeType = iota
	shlVaruse                    // ${PREFIX}
	shlText                      //
	shlSemicolon                 // ;
	shlParenOpen                 // (
	shlParenClose                // )
	shlBraceOpen                 // {
	shlBraceClose                // }
	shlBacktOpen                 // `
	shlBacktClose                // `
	shlSubshellOpen              // $(
	shlPipe                      // |
	shlBackground                // &
	shlOr                        // ||
	shlAnd                       // &&
)

func (t ShLexemeType) String() string {
	return [...]string{
		"space",
		"varuse",
		"text",
		"semicolon",
		"parenOpen", "parenClose",
		"braceOpen", "braceClose",
		"backtOpen", "backtClose",
		"subshellOpen",
		"pipe", "background",
		"or", "and",
	}[t]
}

// ShQuoting describes the context in which a string appears
// and how its literal value is retained.
// It is a sequence of ", ', `; maybe ( for subshell.
type ShQuoting string
