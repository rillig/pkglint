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

// ShWord combines tokens to form (roughly speaking) space-separated items.
type ShWord struct {
	Atoms []*ShToken
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
type ShToken struct {
	Type    ShTokenType
	Text    string
	Quoting ShQuoting
	Data    interface{}
}

func (token *ShToken) String() string {
	if token.Type == shtWord && token.Quoting == shqPlain && token.Data == nil {
		return fmt.Sprintf("%q", token.Text)
	}
	if token.Type == shtVaruse {
		varuse := token.Data.(*MkVarUse)
		return fmt.Sprintf("varuse(%q)", varuse.varname+varuse.Mod())
	}
	return fmt.Sprintf("ShToken(%v, %q, %s)", token.Type, token.Text, token.Quoting)
}

type ShTokenType uint8

const (
	shtSpace         ShTokenType = iota
	shtVaruse                    // ${PREFIX}
	shtWord                      //
	shtSemicolon                 // ;
	shtCaseSeparator             // ;;
	shtParenOpen                 // (
	shtParenClose                // )
	shtBraceOpen                 // {
	shtBraceClose                // }
	shtBacktOpen                 // `
	shtBacktClose                // `
	shtSubshellOpen              // $(
	shtPipe                      // |
	shtBackground                // &
	shtOr                        // ||
	shtAnd                       // &&
	shtRedirect                  // >, <, >>
	shtComment                   // # ...
)

func (t ShTokenType) String() string {
	return [...]string{
		"space",
		"varuse",
		"word",
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
