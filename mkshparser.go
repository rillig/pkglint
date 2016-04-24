package main

type MkShParser struct {
	tok  *ShTokenizer
	curr *ShToken
}

func NewMkShParser(line *Line, text string) *MkShParser {
	tok := NewShTokenizer(line, text)
	return &MkShParser{tok, nil}
}

// See Shell Command Language grammar.
type MkShTokenType uint8

const (
	msttWORD            MkShTokenType = iota
	msttASSIGNMENT_WORD               // Only in commands.
	msttNAME                          // ???
	msttNEWLINE
	msttIO_NUMBER // Only in commands, redirection.

	msttAND_IF // &&
	msttOR_IF  // ||
	msttDSEMI  // ;;

	msttDLESS     // <<
	msttDGREAT    // >>
	msttLESSAND   // <&
	msttGREATAND  // >&
	msttLESSGREAT // <>
	msttDLESSDASH // <<-
	msttCLOBBER   // >|
	msttEOF

	msttSEMI // ;
)

// ::= List Separator?
func (p *MkShParser) CompleteCommand() *MkShCompleteCmd {
	p.List()
	p.Separator()
	return nil
}

// ::= AndOr (SeparatorOp AndOr)*
func (p *MkShParser) List() *MkShList {
	p.AndOr()
	p.SeparatorOp()
	return nil
}

// ::= Pipeline
// ::= AndOr msttAND_IF Linebreak Pipeline
// ::= AndOr msttOR_IF Linebreak Pipeline
func (p *MkShParser) AndOr() *MkShAndOr {
	var pipes []*MkShPipeline
	var ops []MkShTokenType
	pipe := p.Pipeline()
	if pipe == nil {
		return nil
	}

	pipes = append(pipes, pipe)
nextpipe:
	switch op := p.peekType(); op {
	case msttAND_IF, msttOR_IF:
		p.Linebreak()
		pipe := p.Pipeline()
		if pipe != nil {
			pipes = append(pipes, pipe)
			ops = append(ops, op)
			goto nextpipe
		}
	}
	return &MkShAndOr{pipes, ops}
}

// ::= msttBang?  Command (msttPipe Linebreak Command)*
func (p *MkShParser) Pipeline() *MkShPipeline {
	p.Command()
	p.Linebreak()
	return nil
}

// ::= SimpleCommand
// ::= CompoundCommand RedirectList?
// ::= FunctionDefinition
func (p *MkShParser) Command() *MkShCommand {
	p.SimpleCommand()
	p.CompoundCommand()
	p.RedirectList()
	p.FunctionDefinition()
	return nil
}

// ::= BraceGroup
// ::= "(" CompoundList ")"
// ::= ForClause
// ::= CaseClause
// ::= IfClause
// ::= WhileClause
// ::= UntilClause
func (p *MkShParser) CompoundCommand() *MkShCompoundCmd {
	p.BraceGroup()
	p.CompoundList()
	p.ForClause()
	p.CaseClause()
	p.IfClause()
	p.WhileClause()
	p.UntilClause()
	return nil
}

// ::= NewlineList? Term Separator?
func (p *MkShParser) CompoundList() *MkShCompoundList {
	p.NewlineList()
	p.Term()
	p.Separator()
	return nil
}

// ::= Term Separator AndOr
// ::= AndOr
func (p *MkShParser) Term() *MkShTerm {
	p.Term()
	p.Separator()
	p.AndOr()
	return nil
}

// ::= "for" msttWORD Linebreak DoGroup
// ::= "for" msttWORD Linebreak "in" SequentialSep DoGroup
// ::= "for" msttWORD Linebreak "in" Wordlist SequentialSep DoGroup
func (p *MkShParser) ForClause() *MkShForClause {
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
	return nil
}

// ::= "case" msttWORD Linebreak "in" Linebreak CaseItem* "esac"
func (p *MkShParser) CaseClause() *MkShCaseClause {
	p.Linebreak()
	p.CaseItem()
	return nil
}

// ::= "("? Pattern ")" (CompoundList | Linebreak) msttDSEMI? Linebreak
func (p *MkShParser) CaseItem() *MkShCaseItem {
	p.Pattern()
	p.Linebreak()
	p.CompoundList()
	return nil
}

// ::= msttWORD
// ::= Pattern "|" msttWORD
func (p *MkShParser) Pattern() []*ShToken {
	p.Pattern()
	return nil
}

// ::= "if" CompoundList "then" CompoundList ("elif" CompoundList "then" CompoundList)* ("else" CompoundList)? "fi"
func (p *MkShParser) IfClause() *MkShIfClause {
	p.CompoundList()
	return nil
}

// ::= "while" CompoundList DoGroup
func (p *MkShParser) WhileClause() *MkShLoopClause {
	p.CompoundList()
	p.DoGroup()
	return nil
}

// ::= "until" CompoundList DoGroup
func (p *MkShParser) UntilClause() *MkShLoopClause {
	p.CompoundList()
	p.DoGroup()
	return nil
}

// ::= msttNAME "(" ")" Linebreak FunctionBody
func (p *MkShParser) FunctionDefinition() *MkShFunctionDef {
	p.Linebreak()
	p.FunctionBody()
	return nil
}

// ::= CompoundCommand
// ::= CompoundCommand RedirectList
func (p *MkShParser) FunctionBody() *MkShFunctionBody {
	p.CompoundCommand()
	p.RedirectList()
	return nil
}

// ::= "{" CompoundList "}"
func (p *MkShParser) BraceGroup() *MkShCompoundList {
	p.CompoundList()
	return nil
}

// ::= "do" CompoundList "done"
func (p *MkShParser) DoGroup() *MkShCompoundList {
	p.CompoundList()
	return nil
}

func (p *MkShParser) SimpleCommand() *MkShSimpleCmd {
	var words []*ShToken

nextword:
	word := p.tok.ShToken()
	if word != nil {
		words = append(words, word)
		goto nextword
	}
	if len(words) > 0 {
		return &MkShSimpleCmd{words}
	}
	return nil
}

// ::= IoRedirect+
func (p *MkShParser) RedirectList() []*ShToken {
	p.IoRedirect()
	return nil
}

// ::= msttIO_NUMBER? (IoFile | IoHere)
func (p *MkShParser) IoRedirect() *ShToken {
	p.IoFile()
	return nil
}

// ::= ("<"  | "<&" | ">" | ">&" | ">>" | "<>" | ">|") msttWORD
func (p *MkShParser) IoFile() *ShToken {
	// rule 2
	return nil
}

// ::= "<<" msttWORD
// ::= "<<-" msttWORD
func (p *MkShParser) IoHere() *ShToken {
	// rule 3
	return nil
}

// ::= "\n"+
func (p *MkShParser) NewlineList() bool {
	ok := false
	for p.peekType() == msttNEWLINE {
		ok = true
		p.skip()
	}
	return ok
}

func (p *MkShParser) Linebreak() {
	for p.peekType() == msttNEWLINE {
		p.skip()
	}
}

// ::= "&" | ";"
func (p *MkShParser) SeparatorOp() *MkShSeparator {
	return nil
}

// ::= SeparatorOp Linebreak
// ::= NewlineList
func (p *MkShParser) Separator() *MkShSeparator {
	p.SeparatorOp()
	p.Linebreak()
	p.NewlineList()
	return nil
}

func (p *MkShParser) SequentialSep() bool {
	if p.peekType() == msttSEMI {
		p.skip()
		p.Linebreak()
		return true
	} else {
		return p.NewlineList()
	}
}

func (p *MkShParser) mark() PrefixReplacerMark {
	return p.tok.Mark()
}

func (p *MkShParser) peek() *ShToken {
	if p.curr == nil {
		p.curr = p.tok.ShToken()
	}
	return p.curr
}

func (p *MkShParser) peekType() MkShTokenType {
	if curr := p.peek(); curr != nil {
		return curr.Type
	}
	return msttEOF
}

func (p *MkShParser) skip() {
	p.curr = nil
}
