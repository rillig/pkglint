package main

type MkShList struct {
	AndOrs     []*MkShAndOr
	Separators []MkShSeparator
}

func NewMkShList() *MkShList {
	return &MkShList{nil, nil}
}

func (list *MkShList) AddAndOr(andor *MkShAndOr) *MkShList {
	list.AndOrs = append(list.AndOrs, andor)
	return list
}

func (list *MkShList) AddSeparator(separator MkShSeparator) *MkShList {
	list.Separators = append(list.Separators, separator)
	return list
}

type MkShAndOr struct {
	Pipes []*MkShPipeline
	Ops   []string // Either "&&" or "||"
}

func NewMkShAndOr(pipeline *MkShPipeline) *MkShAndOr {
	return &MkShAndOr{[]*MkShPipeline{pipeline}, nil}
}

func (andor *MkShAndOr) Add(op string, pipeline *MkShPipeline) *MkShAndOr {
	andor.Pipes = append(andor.Pipes, pipeline)
	andor.Ops = append(andor.Ops, op)
	return andor
}

type MkShPipeline struct {
	Negated bool
	Cmds    []*MkShCommand
}

func NewMkShPipeline(negated bool, cmds ...*MkShCommand) *MkShPipeline {
	return &MkShPipeline{negated, cmds}
}

func (pipe *MkShPipeline) Add(cmd *MkShCommand) *MkShPipeline {
	pipe.Cmds = append(pipe.Cmds, cmd)
	return pipe
}

type MkShCommand struct {
	Simple    *MkShSimpleCommand
	Compound  *MkShCompoundCommand
	FuncDef   *MkShFunctionDefinition
	Redirects []*MkShRedirection // For Compound and FuncDef
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

func (cl *MkShIfClause) Prepend(cond *MkShList, action *MkShList) {
	cl.Conds = append([]*MkShList{cond}, cl.Conds...)
	cl.Actions = append([]*MkShList{action}, cl.Actions...)
}

type MkShLoopClause struct {
	Cond   *MkShList
	Action *MkShList
	Until  bool
}

type MkShFunctionDefinition struct {
	Name      string
	Body      *MkShCompoundCommand
	Redirects []*MkShRedirection
}

type MkShSimpleCommand struct {
	Assignments  []*ShToken
	Name         *ShToken
	Args         []*ShToken
	Redirections []*MkShRedirection
}

type MkShRedirection struct {
	Fd     int // Or -1
	Op     string
	Target *ShToken
}

// One of ";", "&", "\n"
type MkShSeparator string
