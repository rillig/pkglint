package main

type MkShParser struct {
	*Parser
}

func NewMkShParser(line *Line, text string) *MkShParser {
	return &MkShParser{NewParser(line, text)}
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
func (p *MkShParser) AndOr() *MkShDummy {
	p.Pipeline()
	p.AndOr()
	p.Linebreak()
	return nil
}

// ::= PipeSequence
// ::= msttBang PipeSequence
func (p *MkShParser) Pipeline() *MkShDummy {
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

// ::= CmdPrefix CmdWord CmdSuffix
// ::= CmdPrefix CmdWord?
// ::= CmdName CmdSuffix?
func (p *MkShParser) SimpleCommand() *MkShDummy {
	p.CmdPrefix()
	p.CmdWord()
	p.CmdSuffix()
	p.CmdName()
	return nil
}

// ::= msttWORD
func (p *MkShParser) CmdName() *MkShDummy {
	// rule 7a
	return nil
}

// ::= msttWORD
func (p *MkShParser) CmdWord() *MkShDummy {
	// rule 7b
	return nil
}

// ::= (IoRedirect | msttASSIGNMENT_WORD)+
func (p *MkShParser) CmdPrefix() *MkShDummy {
	p.IoRedirect()
	return nil
}

// ::= (IoRedirect | msttWORD)+
func (p *MkShParser) CmdSuffix() *MkShDummy {
	p.IoRedirect()
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

// ::= "<<" WORD
// ::= "<<-" WORD
func (p *MkShParser) IoHere() *MkShDummy {
	// rule 3
	return nil
}

// ::= "\n"+
func (p *MkShParser) NewlineList() *MkShDummy {
	return nil
}

// ::= "\n"*
func (p *MkShParser) Linebreak() *MkShDummy {
	return nil
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

// ::= ";" Linebreak
// ::= NewlineList
func (p *MkShParser) SequentialSep() *MkShDummy {
	p.Linebreak()
	p.NewlineList()
	return nil
}
