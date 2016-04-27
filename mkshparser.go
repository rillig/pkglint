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
	defer p.trace()()
	ok := false
	defer p.rollback(&ok)()

	list := p.List()
	if list == nil {
		return nil
	}
	separator := p.Separator()
	if separator == nil {
		ok = true
		return list
	}

	ok = true
	return &MkShList{list.AndOrs, append(list.Separators, *separator)}
}

// ::= AndOr (SeparatorOp AndOr)*
func (p *MkShParser) List() *MkShList {
	defer p.trace()()
	ok := false
	defer p.rollback(&ok)()

	var andors []*MkShAndOr
	var seps []MkShSeparator

	if andor := p.AndOr(); andor != nil {
		andors = append(andors, andor)
	} else {
		return nil
	}

next:
	mark := p.mark()
	if sep := p.SeparatorOp(); sep != nil {
		if andor := p.AndOr(); andor != nil {
			andors = append(andors, andor)
			seps = append(seps, *sep)
			goto next
		}
	}
	p.reset(mark)

	ok = true
	return &MkShList{andors, seps}
}

func (p *MkShParser) AndOr() *MkShAndOr {
	defer p.trace()()
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
	defer p.trace()()
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
	defer p.trace()()
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
	defer p.trace()()
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
	defer p.trace()()
	ok := false
	defer p.rollback(&ok)()

	if !p.eat("(") {
		return nil
	}
	list := p.CompoundList()
	if list == nil {
		return nil
	}
	if !p.eat(")") {
		return nil
	}
	ok = true
	return list
}

// ::= Newline* AndOr (Separator AndOr)* Separator?
func (p *MkShParser) CompoundList() *MkShList {
	defer p.trace()()
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
	defer p.trace()()
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
	defer p.trace()()
	panic("Wordlist")
	return nil
}

// ::= "case" msttWORD Linebreak "in" Linebreak CaseItem* "esac"
func (p *MkShParser) CaseClause() *MkShCaseClause {
	defer p.trace()()
	panic("CaseClause")
	p.Linebreak()
	p.CaseItem()
	return nil
}

// ::= "("? Pattern ")" (CompoundList | Linebreak) msttDSEMI? Linebreak
func (p *MkShParser) CaseItem() *MkShCaseItem {
	defer p.trace()()
	panic("CaseItem")
	p.Pattern()
	p.Linebreak()
	p.CompoundList()
	return nil
}

// ::= msttWORD
// ::= Pattern "|" msttWORD
func (p *MkShParser) Pattern() []*ShToken {
	defer p.trace()()
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
	defer p.trace()()
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
	defer p.trace()()
	panic("WhileClause")
	p.CompoundList()
	p.DoGroup()
	return nil
}

// ::= "until" CompoundList DoGroup
func (p *MkShParser) UntilClause() *MkShLoopClause {
	defer p.trace()()
	panic("UntilClause")
	p.CompoundList()
	p.DoGroup()
	return nil
}

// ::= msttNAME "(" ")" Linebreak CompoundCommand Redirect*
func (p *MkShParser) FunctionDefinition() *MkShFunctionDef {
	defer p.trace()()
	panic("FunctionDefinition")
	p.Linebreak()
	p.CompoundCommand()
	p.RedirectList()
	return nil
}

func (p *MkShParser) BraceGroup() *MkShList {
	defer p.trace()()
	ok := false
	defer p.rollback(&ok)()

	if !p.eat("{") {
		return nil
	}
	list := p.CompoundList()
	if list == nil {
		return nil
	}
	if !p.eat("}") {
		return nil
	}
	ok = true
	return list
}

func (p *MkShParser) DoGroup() *MkShList {
	ok := false
	defer p.rollback(&ok)()

	if !p.eat("do") {
		return nil
	}
	list := p.CompoundList()
	if list == nil {
		return nil
	}
	if !p.eat("done") {
		return nil
	}
	ok = true
	return list
}

func (p *MkShParser) SimpleCommand() *MkShSimpleCommand {
	defer p.trace()()
	ok := false
	defer p.rollback(&ok)()

	var words []*ShToken
nextword:
	if word := p.Word(); word != nil {
		words = append(words, word)
		goto nextword
	}
	if len(words) == 0 {
		return nil
	}
	ok = true
	return &MkShSimpleCommand{words}
}

// ::= IoRedirect+
func (p *MkShParser) RedirectList() []*MkShRedirection {
	defer p.trace()()
	panic("RedirectList")
	p.IoRedirect()
	return nil
}

// ::= msttIO_NUMBER? (IoFile | IoHere)
func (p *MkShParser) IoRedirect() *ShToken {
	defer p.trace()()
	panic("IoRedirect")
	p.IoFile()
	return nil
}

// ::= ("<"  | "<&" | ">" | ">&" | ">>" | "<>" | ">|") msttWORD
func (p *MkShParser) IoFile() *ShToken {
	defer p.trace()()
	panic("IoFile")
	// rule 2
	return nil
}

// ::= "<<" msttWORD
// ::= "<<-" msttWORD
func (p *MkShParser) IoHere() *ShToken {
	defer p.trace()()
	panic("IoHere")
	// rule 3
	return nil
}

func (p *MkShParser) NewlineList() bool {
	defer p.trace()()
	ok := false
	for p.eat("\n") {
		ok = true
	}
	return ok
}

func (p *MkShParser) Linebreak() {
	defer p.trace()()
	for p.eat("\n") {
	}
}

func (p *MkShParser) SeparatorOp() *MkShSeparator {
	defer p.trace()()
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
	defer p.trace()()
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
	defer p.trace()()
	if p.peekText() == ";" {
		p.skip()
		p.Linebreak()
		return true
	} else {
		return p.NewlineList()
	}
}

func (p *MkShParser) Word() *ShToken {
	defer p.trace()()
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
	traceStep("MkShParser.peek %v at %v", p.curr, p.tok.mkp.repl.rest)
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
	mark := p.mark()
	return func() {
		if !*pok {
			p.reset(mark)
		}
	}
}

func (p *MkShParser) trace() func() {
	if G.opts.Debug {
		return tracecallInternal(p.peek(), p.tok.mkp.repl.rest)
	} else {
		return func() {}
	}
}

func (p *MkShParser) mark() MkShParserMark {
	return MkShParserMark{p.tok.parser.repl.Mark(), p.curr}
}

func (p *MkShParser) reset(mark MkShParserMark) {
	p.tok.parser.repl.Reset(mark.rest)
	p.curr = mark.curr
}

type MkShParserMark struct {
	rest PrefixReplacerMark
	curr *ShToken
}
