package main

type MkShParser struct {
	*Parser
	*ShParser
	next *ShToken
}

func NewMkShParser(line *Line, text string) *MkShParser {
	p := NewParser(line, text)
	mkp := &MkParser{p}
	shp := &ShParser{p, mkp}
	next := shp.ShToken()
	return &MkShParser{p, shp, next}
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

type MkShDummy struct {
}

// ::= List Separator?
func (p *MkShParser) CompleteCommand() *MkShDummy {
	p.List()
	p.Separator()
	return nil
}

// ::= AndOr (SeparatorOp AndOr)*
func (p *MkShParser) List() *MkShDummy {
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
	switch op := p.peek(); op {
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

// ::= PipeSequence
// ::= msttBang PipeSequence
func (p *MkShParser) Pipeline() *MkShPipeline {
	p.PipeSequence()
	return nil
}

// ::= Command
// ::= PipeSequence msttPipe Linebreak Command
func (p *MkShParser) PipeSequence() *MkShDummy {
	p.Command()
	p.PipeSequence()
	p.Linebreak()
	p.Command()
	return nil
}

// ::= SimpleCommand
// ::= CompoundCommand
// ::= CompoundCommand RedirectList
// ::= FunctionDefinition
func (p *MkShParser) Command() *MkShDummy {
	p.SimpleCommand()
	p.CompoundCommand()
	p.RedirectList()
	p.FunctionDefinition()
	return nil
}

// ::= BraceGroup
// ::= Subshell
// ::= ForClause
// ::= CaseClause
// ::= IfClause
// ::= WhileClause
// ::= UntilClause
func (p *MkShParser) CompoundCommand() *MkShDummy {
	p.BraceGroup()
	p.Subshell()
	p.ForClause()
	p.CaseClause()
	p.IfClause()
	p.WhileClause()
	p.UntilClause()
	return nil
}

// ::= "(" CompoundList ")"
func (p *MkShParser) Subshell() *MkShDummy {
	p.CompoundList()
	return nil
}

// ::= NewlineList? Term Separator?
func (p *MkShParser) CompoundList() *MkShDummy {
	p.NewlineList()
	p.Term()
	p.Separator()
	return nil
}

// ::= Term Separator AndOr
// ::= AndOr
func (p *MkShParser) Term() *MkShDummy {
	p.Term()
	p.Separator()
	p.AndOr()
	return nil
}

// ::= "for" Name Linebreak DoGroup
// ::= "for" Name Linebreak "in" SequentialSep DoGroup
// ::= "for" Name Linebreak "in" Wordlist SequentialSep DoGroup
func (p *MkShParser) ForClause() *MkShDummy {
	p.Name()
	p.Linebreak()
	p.DoGroup()
	p.SequentialSep()
	p.Wordlist()
	// See rule 6 for "in"
	return nil
}

func (p *MkShParser) Name() *MkShDummy {
	// See rule 5
	return nil
}

// ::= Wordlist msttWord
// ::= msttWord
func (p *MkShParser) Wordlist() *MkShDummy {
	return nil
}

// ::= "case" msttWORD Linebreak "in" Linebreak CaseList "esac"
// ::= "case" msttWORD Linebreak "in" Linebreak CaseListNs "esac"
// ::= "case" msttWORD Linebreak "in" Linebreak "esac"
func (p *MkShParser) CaseClause() *MkShDummy {
	p.Linebreak()
	p.CaseList()
	p.CaseListNs()
	return nil
}

// ::= CaseList CaseItemNs
// ::= CaseItemNs
func (p *MkShParser) CaseListNs() *MkShDummy {
	p.CaseList()
	p.CaseItemNs()
	return nil
}

// ::= CaseList CaseItem
// ::= CaseItem
func (p *MkShParser) CaseList() *MkShDummy {
	p.CaseList()
	p.CaseItem()
	return nil
}

// ::= "("? Pattern ")" CompoundList? Linebreak
func (p *MkShParser) CaseItemNs() *MkShDummy {
	p.Pattern()
	p.CompoundList()
	p.Linebreak()
	return nil
}

// ::= "("? Pattern ")" Linebreak msttDSEMI Linebreak
// ::= "("? Pattern ")" CompoundList msttDSEMI Linebreak
func (p *MkShParser) CaseItem() *MkShDummy {
	p.Pattern()
	p.Linebreak()
	p.CompoundList()
	return nil
}

// ::= msttWORD
// ::= Pattern "|" msttWORD
func (p *MkShParser) Pattern() *MkShDummy {
	p.Pattern()
	return nil
}

// ::= "if" CompoundList "then" CompoundList ElsePart? "fi"
func (p *MkShParser) IfClause() *MkShDummy {
	p.CompoundList()
	p.ElsePart()
	return nil
}

// ::= ("elif" CompoundList "then" CompoundList)+
// ::= "else" CompoundList
func (p *MkShParser) ElsePart() *MkShDummy {
	p.CompoundList()
	return nil
}

// ::= "while" CompoundList DoGroup
func (p *MkShParser) WhileClause() *MkShDummy {
	p.CompoundList()
	p.DoGroup()
	return nil
}

// ::= "until" CompoundList DoGroup
func (p *MkShParser) UntilClause() *MkShDummy {
	p.CompoundList()
	p.DoGroup()
	return nil
}

// ::= msttNAME "(" ")" Linebreak FunctionBody
func (p *MkShParser) FunctionDefinition() *MkShDummy {
	p.Linebreak()
	p.FunctionBody()
	return nil
}

// ::= CompoundCommand
// ::= CompoundCommand RedirectList
func (p *MkShParser) FunctionBody() *MkShDummy {
	p.CompoundCommand()
	p.RedirectList()
	return nil
}

// ::= "{" CompoundList "}"
func (p *MkShParser) BraceGroup() *MkShDummy {
	p.CompoundList()
	return nil
}

// ::= "do" CompoundList "done"
func (p *MkShParser) DoGroup() *MkShDummy {
	p.CompoundList()
	return nil
}

func (p *MkShParser) SimpleCommand() *MkShSimpleCmd {
	var words []*ShToken

nextword:
	word := p.ShToken()
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
func (p *MkShParser) RedirectList() *MkShDummy {
	p.IoRedirect()
	return nil
}

// ::= msttIO_NUMBER? (IoFile | IoHere)
func (p *MkShParser) IoRedirect() *MkShDummy {
	p.IoFile()
	return nil
}

// ::= ("<"  | "<&" | ">" | ">&" | ">>" | "<>" | ">|") msttWORD
func (p *MkShParser) IoFile() *MkShDummy {
	// rule 2
	return nil
}

// ::= "<<" msttWORD
// ::= "<<-" msttWORD
func (p *MkShParser) IoHere() *MkShDummy {
	// rule 3
	return nil
}

// ::= "\n"+
func (p *MkShParser) NewlineList() bool {
	ok := false
	for p.repl.AdvanceStr("\n") {
		ok = true
	}
	return ok
}

func (p *MkShParser) Linebreak() {
	for p.repl.AdvanceStr("\n") {
	}
}

// ::= "&" | ";"
func (p *MkShParser) SeparatorOp() *MkShDummy {
	return nil
}

// ::= SeparatorOp Linebreak
// ::= NewlineList
func (p *MkShParser) Separator() *MkShDummy {
	p.SeparatorOp()
	p.Linebreak()
	p.NewlineList()
	return nil
}

func (p *MkShParser) SequentialSep() bool {
	if p.peek() == msttSEMI {
		p.skip()
		p.Linebreak()
		return true
	} else {
		return p.NewlineList()
	}
}

func (p *MkShParser) mark() PrefixReplacerMark {
	return p.repl.Mark()
}

func (p *MkShParser) peek() MkShTokenType {
	if p.next != nil {
		return p.next.Type
	}
	return msttEOF
}

func (p *MkShParser) skip() {
	p.next = p.ShToken()
}
