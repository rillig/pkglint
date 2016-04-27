package main

type MkShList struct {
	AndOrs     []*MkShAndOr
	Separators []MkShSeparator
}

type MkShAndOr struct {
	Pipes []*MkShPipeline
	Ops   string // Either '&' or '|'
}

type MkShPipeline struct {
	Negated bool
	Cmds    []*MkShCommand
}

type MkShCommand struct {
	Simple    *MkShSimpleCommand
	Compound  *MkShCompoundCommand
	Redirects []*MkShRedirection // For Compound
	FuncDef   *MkShFunctionDef
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

type MkShForClause struct {
	Varname string
	Values  []*ShToken
	Body    *MkShList
}
type MkShCaseClause struct {
	Word  *ShToken
	Cases []*MkShCaseItem
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

type MkShLoopClause struct {
	Cond   *MkShList
	Action *MkShList
	Until  bool
}

type MkShFunctionDef struct {
	Name      string
	Body      *MkShCompoundCommand
	Redirects []*MkShRedirection
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
