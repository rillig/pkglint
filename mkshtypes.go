package main

import (
	"fmt"
)

type MkShList struct {
	AndOrs     []*MkShAndOr
	Separators []MkShSeparator
}

func (list *MkShList) String() string {
	return fmt.Sprintf("MkShList(%v)", list.AndOrs)
}

type MkShAndOr struct {
	Pipes []*MkShPipeline
	Ops   string // Either '&' or '|'
}

func NewMkShAndOr1(pipeline *MkShPipeline) *MkShAndOr {
	return &MkShAndOr{[]*MkShPipeline{pipeline}, ""}
}

func (andor *MkShAndOr) String() string {
	return fmt.Sprintf("MkShAndOr(%v)", andor.Pipes)
}

type MkShPipeline struct {
	Negated bool
	Cmds    []*MkShCommand
}

func NewMkShPipeline(negated bool, cmds ...*MkShCommand) *MkShPipeline {
	return &MkShPipeline{negated, cmds}
}

func (pipe *MkShPipeline) String() string {
	return fmt.Sprintf("MkShPipeline(%v)", pipe.Cmds)
}

type MkShCommand struct {
	Simple    *MkShSimpleCommand
	Compound  *MkShCompoundCommand
	Redirects []*MkShRedirection // For Compound
	FuncDef   *MkShFunctionDef
}

func (cmd *MkShCommand) String() string {
	switch {
	case cmd.Simple != nil:
		return cmd.Simple.String()
	case cmd.Compound != nil:
		return cmd.Compound.String()
	case cmd.FuncDef != nil:
		return cmd.FuncDef.String()
	}
	return "MkShCommand(?)"
}

type MkShCompoundCommand struct {
	Brace    *MkShList
	Subshell *MkShList
	For      *MkShForClause
	Case     *MkShCaseClause
	If       *MkShIfClause
	While    *MkShLoopClause
	Until    *MkShLoopClause
}

func (cmd *MkShCompoundCommand) String() string {
	switch {
	case cmd.Brace != nil:
		return cmd.Brace.String()
	case cmd.Subshell != nil:
		return cmd.Subshell.String()
	case cmd.For != nil:
		return cmd.For.String()
	case cmd.Case != nil:
		return cmd.Case.String()
	case cmd.If != nil:
		return cmd.If.String()
	case cmd.While != nil:
		return cmd.While.String()
	case cmd.Until != nil:
		return cmd.Until.String()
	}
	return "MkShCompoundCommand(?)"
}

type MkShForClause struct {
	Varname string
	Values  []*ShToken
	Body    *MkShList
}

func (cl *MkShForClause) String() string {
	return fmt.Sprintf("MkShForClause(%v, %v, %v)", cl.Varname, cl.Values, cl.Body)
}

type MkShCaseClause struct {
	Word  *ShToken
	Cases []*MkShCaseItem
}

func (cl *MkShCaseClause) String() string {
	return fmt.Sprintf("MkShCaseClause(...)")
}

type MkShCaseItem struct {
	Patterns []*ShToken
	Action   *MkShList
}

type MkShIfClause struct {
	Conds   []*MkShList
	Actions []*MkShList
	Else    *MkShList
}

func (cl *MkShIfClause) String() string {
	return "MkShIf(...)"
}

type MkShLoopClause struct {
	Cond   *MkShList
	Action *MkShList
	Until  bool
}

func (cl *MkShLoopClause) String() string {
	return "MkShLoop(...)"
}

type MkShFunctionDef struct {
	Name      string
	Body      *MkShCompoundCommand
	Redirects []*MkShRedirection
}

func (def *MkShFunctionDef) String() string {
	return "MkShFunctionDef(...)"
}

type MkShSimpleCommand struct {
	Words []*ShToken // Can be redirects, too.
}

func NewMkShSimpleCommand(words ...*ShToken) *MkShSimpleCommand {
	return &MkShSimpleCommand{words}
}

func (scmd *MkShSimpleCommand) String() string {
	str := "SimpleCommand("
	for i, word := range scmd.Words {
		if i != 0 {
			str += ", "
		}
		str += word.MkText
	}
	return str
}

type MkShRedirection ShToken

// One of ';', '&', '\n'
type MkShSeparator rune

func (sep *MkShSeparator) String() string {
	return fmt.Sprintf("%q", sep)

}
