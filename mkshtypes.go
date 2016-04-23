package main

type MkShCompleteCmd struct {
	Lists      []*MkShList
	Separators []MkShSeparator
}

type MkShList struct {
	AndOrs     []*MkShAndOr
	Separators []MkShSeparator
}

type MkShAndOr struct {
	Pipes []*MkShPipeline
	Ops   []MkShTokenType
}

type MkShPipeline struct {
}

type MkShCommand struct {
	Simple    *MkShSimpleCmd
	Compound  *MkShCompoundCmd
	Redirects *MkShRedirectList // For Compound
	FuncDef   *MkShFunctionDef
}

type MkShRedirectList struct {
}

type MkShFunctionDef struct {
	Name string
	Body *MkShFunctionBody
}

type MkShSimpleCmd struct {
	Words []*ShToken
}

type MkShCompoundCmd struct {
	Brace    *MkShBraceGroup
	Subshell *MkShCompoundList
}

type MkShBraceGroup struct {
}

type MkShCompoundList struct {
}

type MkShTerm struct {
	AndOrs     []*MkShAndOr
	Separators []MkShSeparator
}

type MkShForClause struct {
	Varname string
	Ins     []*ShToken
	Body    *MkShCompoundList
}
type MkShSeparator uint8

const (
	mssSemicolon MkShSeparator = iota
	mssAmpersand
	mssNewlines
)

type MkShFunctionBody struct {
}

type MkShCaseClause struct {
	Word  *ShToken
	Cases []*MkShCaseItem
}

type MkShCaseItem struct {
	Patterns []*ShToken
	Action   *MkShCompoundList
}

type MkShIfClause struct {
	Conds   []*MkShCompoundList
	Actions []*MkShCompoundList
	Else    *MkShCompoundList
}

type MkShLoopClause struct {
	Cond   *MkShCompoundList
	Action *MkShCompoundList
	Until  bool
}
