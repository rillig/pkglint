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

// ShWord combines lexemes to form (roughly speaking) space-separated items.
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
	if shlex.Type == shlText && shlex.Quoting == shqPlain && shlex.Data == nil {
		return fmt.Sprintf("%q", shlex.Text)
	}
	if shlex.Type == shlVaruse {
		varuse := shlex.Data.(*MkVarUse)
		return fmt.Sprintf("varuse(%q)", varuse.varname+varuse.Mod())
	}
	return fmt.Sprintf("ShLexeme(%v, %q, %s)", shlex.Type, shlex.Text, shlex.Quoting)
}

type ShLexemeType uint8

const (
	shlSpace         ShLexemeType = iota
	shlVaruse                     // ${PREFIX}
	shlText                       //
	shlSemicolon                  // ;
	shlCaseSeparator              // ;;
	shlParenOpen                  // (
	shlParenClose                 // )
	shlBraceOpen                  // {
	shlBraceClose                 // }
	shlBacktOpen                  // `
	shlBacktClose                 // `
	shlSubshellOpen               // $(
	shlPipe                       // |
	shlBackground                 // &
	shlOr                         // ||
	shlAnd                        // &&
	shlRedirect                   // >, <, >>
	shlComment                    // # ...
)

func (t ShLexemeType) String() string {
	return [...]string{
		"space",
		"varuse",
		"text",
		"semicolon",
		"caseSeparator",
		"parenOpen", "parenClose",
		"braceOpen", "braceClose",
		"backtOpen", "backtClose",
		"subshellOpen",
		"pipe", "background",
		"or", "and",
		"redirect",
		"comment",
	}[t]
}

// ShQuoting describes the context in which a string appears
// and how it must be unescaped to get its literal value.
type ShQuoting uint8

const (
	shqPlain ShQuoting = iota
	shqDquot
	shqSquot
	shqBackt
	shqDquotBackt
	shqBacktDquot
	shqBacktSquot
	shqDquotBacktDquot
	shqDquotBacktSquot
	shqUnknown
)

func (q ShQuoting) String() string {
	return [...]string{
		"plain",
		"d", "s", "b",
		"db", "bd", "bs",
		"dbd", "dbs",
		"unknown",
	}[q]
}

func (q ShQuoting) ToVarUseContext() vucQuoting {
	switch q {
	case shqPlain:
		return vucQuotPlain
	case shqDquot:
		return vucQuotDquot
	case shqSquot:
		return vucQuotSquot
	case shqBackt:
		return vucQuotBackt
	}
	return vucQuotUnknown
}
