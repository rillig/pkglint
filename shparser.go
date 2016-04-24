package main

type ShParser struct {
	tok  *ShTokenizer
	curr *ShToken
}

func NewShParser(line *Line, text string) *ShParser {
	tok := NewShTokenizer(line, text)
	return &ShParser{tok, nil}
}

// @Beta
func (p *ShParser) ShSimpleCmd() *ShSimpleCmd {
	mark := p.tok.Mark()

	var tokens []*ShToken

nexttoken:
	if token := p.Word(); token != nil {
		tokens = append(tokens, token)
		goto nexttoken
	}

	if len(tokens) != 0 {
		i := 0
		for i < len(tokens) && tokens[i].IsAssignment() {
			i++
		}
		var cmd *ShToken
		if i < len(tokens) {
			cmd = tokens[i]
		}
		return &ShSimpleCmd{tokens, cmd}
	}

	p.tok.Reset(mark)
	return nil
}

// @Beta
func (p *ShParser) ShSimpleCmds() []*ShSimpleCmd {
	var cmds []*ShSimpleCmd

nextcommand:
	cmd := p.ShSimpleCmd()
	if cmd == nil {
		return cmds
	}
	cmds = append(cmds, cmd)
	mark := p.tok.Mark()
	atom := p.tok.ShAtom(shqPlain)
	if atom != nil && atom.Type.IsCommandDelimiter() {
		goto nextcommand
	}
	p.tok.Reset(mark)
	return cmds
}

func (p *ShParser) Word() *ShToken {
	curr := p.peek()
	if curr != nil && curr.Atoms[0].Type.IsWord() {
		return p.next()
	}
	return nil
}

func (p *ShParser) Rest() string {
	return p.tok.Rest()
}

func (p *ShParser) peek() *ShToken {
	if p.curr == nil {
		p.curr = p.tok.ShToken()
	}
	return p.curr
}

func (p *ShParser) next() *ShToken {
	next := p.peek()
	p.curr = nil
	return next
}
