package main

type MkShParser struct {
	tok  *ShTokenizer
	curr *ShToken
}

func NewMkShParser(line *Line, text string) *MkShParser {
	shp := NewShTokenizer(line, text)
	return &MkShParser{shp, nil}
}

func (p *MkShParser) Program() *MkShList {
	ok := false
	defer p.rollback(&ok)()

	list := p.List()
	separator := p.Separator()
	if list == nil {
		return nil
	}
	ok = true
	if separator == nil {
		return list
	}
	return &MkShList{list.AndOrs, append(list.Separators, *separator)}
}

// ::= AndOr (SeparatorOp AndOr)*
func (p *MkShParser) List() *MkShList {
	panic("List")
	p.AndOr()
	p.SeparatorOp()
	return nil
}

func (p *MkShParser) AndOr() *MkShAndOr {
	ok := false
	defer p.rollback(&ok)

	var pipes []*MkShPipeline
	var ops string
nextpipe:
	pipe := p.Pipeline()
	if pipe == nil {
		return nil
	}

	pipes = append(pipes, pipe)
	switch op := p.peekText(); op {
	case "&&", "||":
		p.Linebreak()
		ops += ifelseStr(op == "&&", "&", "|")
		goto nextpipe
	}
	ok = true
	return &MkShAndOr{pipes, ops}
}

// ::= Command (msttPipe Linebreak Command)*
func (p *MkShParser) Pipeline() *MkShPipeline {
	ok := false
	defer p.rollback(&ok)()

	bang := p.eat("!")
	var cmds []*MkShCommand
nextcmd:
	cmd := p.Command()
	if cmd == nil {
		return nil
	}
	if p.eat("|") {
		p.Linebreak()
		goto nextcmd
	}
	return &MkShPipeline{bang, cmds}
}

func (p *MkShParser) Command() *MkShCommand {
	if simple := p.SimpleCommand(); simple != nil {
		return &MkShCommand{Simple: simple}
	}
	if compound := p.CompoundCommand(); compound != nil {
		redirects := p.RedirectList()
		return &MkShCommand{Compound: compound, Redirects: redirects}
	}
	if funcdef := p.FunctionDefinition(); funcdef != nil {
		return &MkShCommand{FuncDef: funcdef}
	}
	return nil
}

func (p *MkShParser) CompoundCommand() *MkShCompoundCommand {
	if brace := p.BraceGroup(); brace != nil {
		return &MkShCompoundCommand{Brace: brace}
	}
	if subshell := p.Subshell(); subshell != nil {
		return &MkShCompoundCommand{Subshell: subshell}
	}
	if forclause := p.ForClause(); forclause != nil {
		return &MkShCompoundCommand{For: forclause}
	}
	if caseclause := p.CaseClause(); caseclause != nil {
		return &MkShCompoundCommand{Case: caseclause}
	}
	if ifclause := p.IfClause(); ifclause != nil {
		return &MkShCompoundCommand{If: ifclause}
	}
	if whileclause := p.WhileClause(); whileclause != nil {
		return &MkShCompoundCommand{While: whileclause}
	}
	if untilclause := p.UntilClause(); untilclause != nil {
		return &MkShCompoundCommand{Until: untilclause}
	}
	return nil
}

func (p *MkShParser) Subshell() *MkShList {
	ok := false
	defer p.rollback(&ok)()

	ok1 := p.eat("(")
	list := p.CompoundList()
	ok2 := p.eat(")")
	if ok1 && list != nil && ok2 {
		ok = true
		return list
	}
	return nil
}

// ::= Newline* AndOr (Separator AndOr)* Separator?
func (p *MkShParser) CompoundList() *MkShList {
	ok := false
	defer p.rollback(&ok)()

	p.Linebreak()
	var andors []*MkShAndOr
	var separators []MkShSeparator
nextandor:
	andor := p.AndOr()
	if andor != nil {
		andors = append(andors, andor)
		if sep := p.Separator(); sep != nil {
			separators = append(separators, *sep)
			goto nextandor
		}
	}
	if len(andors) == 0 {
		return nil
	}
	ok = true
	return &MkShList{andors, separators}
}

// ::= "for" msttWORD Linebreak DoGroup
// ::= "for" msttWORD Linebreak "in" SequentialSep DoGroup
// ::= "for" msttWORD Linebreak "in" Wordlist SequentialSep DoGroup
func (p *MkShParser) ForClause() *MkShForClause {
	panic("ForClause")
	p.Linebreak()
	p.DoGroup()
	p.SequentialSep()
	p.Wordlist()
	// See rule 6 for "in"
	return nil
}

// ::= Wordlist msttWord
// ::= msttWord
func (p *MkShParser) Wordlist() []*ShToken {
	panic("Wordlist")
	return nil
}

// ::= "case" msttWORD Linebreak "in" Linebreak CaseItem* "esac"
func (p *MkShParser) CaseClause() *MkShCaseClause {
	panic("CaseClause")
	p.Linebreak()
	p.CaseItem()
	return nil
}

// ::= "("? Pattern ")" (CompoundList | Linebreak) msttDSEMI? Linebreak
func (p *MkShParser) CaseItem() *MkShCaseItem {
	panic("CaseItem")
	p.Pattern()
	p.Linebreak()
	p.CompoundList()
	return nil
}

// ::= msttWORD
// ::= Pattern "|" msttWORD
func (p *MkShParser) Pattern() []*ShToken {
	ok := false
	defer p.rollback(&ok)()

	var words []*ShToken
nextword:
	word := p.Word()
	if word == nil {
		return nil
	}
	words = append(words, word)
	if p.eat("|") {
		goto nextword

	}
	ok = true
	return words
}

func (p *MkShParser) IfClause() *MkShIfClause {
	ok := false
	defer p.rollback(&ok)()

	var conds []*MkShList
	var actions []*MkShList
	var elseaction *MkShList
	if !p.eat("if") {
		return nil
	}

nextcond:
	cond := p.CompoundList()
	if cond == nil || !p.eat("then") {
		return nil
	}
	action := p.CompoundList()
	if action == nil {
		return nil
	}
	conds = append(conds, cond)
	actions = append(actions, action)
	if p.eat("elif") {
		goto nextcond
	}
	if p.eat("else") {
		elseaction = p.CompoundList()
		if elseaction == nil {
			return nil
		}
	}
	if !p.eat("fi") {
		return nil
	}
	ok = true
	return &MkShIfClause{conds, actions, elseaction}
}

// ::= "while" CompoundList DoGroup
func (p *MkShParser) WhileClause() *MkShLoopClause {
	panic("WhileClause")
	p.CompoundList()
	p.DoGroup()
	return nil
}

// ::= "until" CompoundList DoGroup
func (p *MkShParser) UntilClause() *MkShLoopClause {
	panic("UntilClause")
	p.CompoundList()
	p.DoGroup()
	return nil
}

// ::= msttNAME "(" ")" Linebreak CompoundCommand Redirect*
func (p *MkShParser) FunctionDefinition() *MkShFunctionDef {
	panic("FunctionDefinition")
	p.Linebreak()
	p.CompoundCommand()
	p.RedirectList()
	return nil
}

// ::= "{" CompoundList "}"
func (p *MkShParser) BraceGroup() *MkShList {
	panic("BraceGroup")
	p.CompoundList()
	return nil
}

// ::= "do" CompoundList "done"
func (p *MkShParser) DoGroup() *MkShList {
	panic("DoGroup")
	p.CompoundList()
	return nil
}

func (p *MkShParser) SimpleCommand() *MkShSimpleCommand {
	ok := false
	defer p.rollback(&ok)()

	var words []*ShToken
nextword:
	word := p.Word()
	//dummyLine.Notef("words=%v word=%v",words,word)
	if word != nil {
		words = append(words, word)
		goto nextword
	}
	//dummyLine.Warnf("words=%v",words)
	if len(words) == 0 {
		return nil
	}
	ok = true
	return &MkShSimpleCommand{words}
}

// ::= IoRedirect+
func (p *MkShParser) RedirectList() []*MkShRedirection {
	panic("RedirectList")
	p.IoRedirect()
	return nil
}

// ::= msttIO_NUMBER? (IoFile | IoHere)
func (p *MkShParser) IoRedirect() *ShToken {
	panic("IoRedirect")
	p.IoFile()
	return nil
}

// ::= ("<"  | "<&" | ">" | ">&" | ">>" | "<>" | ">|") msttWORD
func (p *MkShParser) IoFile() *ShToken {
	panic("IoFile")
	// rule 2
	return nil
}

// ::= "<<" msttWORD
// ::= "<<-" msttWORD
func (p *MkShParser) IoHere() *ShToken {
	panic("IoHere")
	// rule 3
	return nil
}

func (p *MkShParser) NewlineList() bool {
	ok := false
	for p.eat("\n") {
		ok = true
	}
	return ok
}

func (p *MkShParser) Linebreak() {
	for p.eat("\n") {
	}
}

func (p *MkShParser) SeparatorOp() *MkShSeparator {
	if p.eat(";") {
		op := MkShSeparator(';')
		return &op
	}
	if p.eat("&") {
		op := MkShSeparator('&')
		return &op
	}
	return nil
}

func (p *MkShParser) Separator() *MkShSeparator {
	op := p.SeparatorOp()
	if op == nil && p.eat("\n") {
		sep := MkShSeparator('\n')
		op = &sep
	}
	if op != nil {
		p.Linebreak()
	}
	return op
}

func (p *MkShParser) SequentialSep() bool {
	if p.peekText() == ";" {
		p.skip()
		p.Linebreak()
		return true
	} else {
		return p.NewlineList()
	}
}

func (p *MkShParser) Word() *ShToken {
	if token := p.peek(); token != nil && token.IsWord() {
		p.skip()
		return token
	}
	return nil
}

func (p *MkShParser) peek() *ShToken {
	if p.curr == nil {
		p.curr = p.tok.ShToken()
		if p.curr == nil && !p.tok.parser.EOF() {
			panic("parse error at " + p.tok.parser.Rest())
		}
	}
	return p.curr
}

func (p *MkShParser) peekText() string {
	if next := p.peek(); next != nil {
		return next.MkText
	}
	return ""
}

func (p *MkShParser) skip() {
	p.curr = nil
}

func (p *MkShParser) eat(s string) bool {
	if p.peek() == nil {
		panic("p.peek is nil")
	}
	if p.peek().MkText == s {
		p.skip()
		return true
	}
	return false
}

func (p *MkShParser) rollback(pok *bool) func() {
	mark := p.tok.parser.repl.Mark()
	return func() {
		if !*pok {
			p.tok.parser.repl.Reset(mark)
		}
	}
}
