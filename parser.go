package main

import (
	"fmt"
	"strings"
)

type Parser struct {
	line *Line
	repl *PrefixReplacer
}

func NewParser(line *Line, s string) *Parser {
	return &Parser{line, NewPrefixReplacer(s)}
}

func (p *Parser) EOF() bool {
	return p.repl.rest == ""
}

func (p *Parser) Rest() string {
	return p.repl.rest
}

func (p *Parser) PkgbasePattern() (pkgbase string) {
	repl := p.repl

	for {
		if repl.AdvanceRegexp(`^\$\{\w+\}`) ||
			repl.AdvanceRegexp(`^[\w.*+,{}]+`) ||
			repl.AdvanceRegexp(`^\[[\d-]+\]`) {
			pkgbase += repl.m[0]
			continue
		}

		mark := repl.Mark()
		if repl.AdvanceStr("-") {
			if repl.AdvanceRegexp(`^\d`) ||
				repl.AdvanceRegexp(`^\$\{\w*VER\w*\}`) ||
				repl.AdvanceStr("[") {
				repl.Reset(mark)
				return
			}
			pkgbase += "-"
		} else {
			return
		}
	}
}

func (p *Parser) Dependency() *DependencyPattern {
	repl := p.repl

	var dp DependencyPattern
	mark := repl.Mark()
	dp.pkgbase = p.PkgbasePattern()
	if dp.pkgbase == "" {
		return nil
	}

	mark2 := repl.Mark()
	if repl.AdvanceStr(">=") || repl.AdvanceStr(">") {
		op := repl.s
		if repl.AdvanceRegexp(`^(?:(?:\$\{\w+\})+|\d[\w.]*)`) {
			dp.lowerOp = op
			dp.lower = repl.m[0]
		} else {
			repl.Reset(mark2)
		}
	}
	if repl.AdvanceStr("<=") || repl.AdvanceStr("<") {
		op := repl.s
		if repl.AdvanceRegexp(`^(?:(?:\$\{\w+\})+|\d[\w.]*)`) {
			dp.upperOp = op
			dp.upper = repl.m[0]
		} else {
			repl.Reset(mark2)
		}
	}
	if dp.lowerOp != "" || dp.upperOp != "" {
		return &dp
	}
	if repl.AdvanceStr("-") && repl.rest != "" {
		dp.wildcard = repl.AdvanceRest()
		return &dp
	}
	if hasPrefix(dp.pkgbase, "${") && hasSuffix(dp.pkgbase, "}") {
		return &dp
	}
	if hasSuffix(dp.pkgbase, "-*") {
		dp.pkgbase = strings.TrimSuffix(dp.pkgbase, "-*")
		dp.wildcard = "*"
		return &dp
	}

	repl.Reset(mark)
	return nil
}

type MkToken struct {
	Text   string // Used for both literals and varuses.
	Varuse *MkVarUse
}
type MkVarUse struct {
	varname   string
	modifiers []string // E.g. "Q", "S/from/to/"
}

func (vu *MkVarUse) Mod() string {
	mod := ""
	for _, modifier := range vu.modifiers {
		mod += ":" + modifier
	}
	return mod
}

func (vu *MkVarUse) HasL() bool {
	for _, mod := range vu.modifiers {
		if mod == "L" {
			return true
		}
	}
	return false
}
func (vu *MkVarUse) IsQ() bool {
	mlen := len(vu.modifiers)
	return mlen > 0 && vu.modifiers[mlen-1] == "Q"
}

func (p *Parser) MkTokens() []*MkToken {
	repl := p.repl

	var tokens []*MkToken
	for !p.EOF() {
		if repl.AdvanceStr("#") {
			repl.AdvanceRest()
		}

		mark := repl.Mark()
		if varuse := p.VarUse(); varuse != nil {
			tokens = append(tokens, &MkToken{Text: repl.Since(mark), Varuse: varuse})
			continue
		}

		needsReplace := false
	again:
		dollar := strings.IndexByte(repl.rest, '$')
		if dollar == -1 {
			dollar = len(repl.rest)
		}
		repl.Skip(dollar)
		if repl.AdvanceStr("$$") {
			needsReplace = true
			goto again
		}
		text := repl.Since(mark)
		if needsReplace {
			text = strings.Replace(text, "$$", "$", -1)
		}
		if text != "" {
			tokens = append(tokens, &MkToken{Text: text})
			continue
		}

		break
	}
	return tokens
}

func (p *Parser) Varname() string {
	repl := p.repl

	mark := repl.Mark()
	repl.AdvanceStr(".")
	isVarnameChar := func(c byte) bool {
		return 'A' <= c && c <= 'Z' || c == '_' || 'a' <= c && c <= 'z' || '0' <= c && c <= '9' || c == '+' || c == '-' || c == '.'
	}
	for p.VarUse() != nil || repl.AdvanceBytesFunc(isVarnameChar) {
	}
	return repl.Since(mark)
}

func (p *Parser) VarUse() *MkVarUse {
	repl := p.repl

	mark := repl.Mark()
	if repl.AdvanceStr("${") || repl.AdvanceStr("$(") {
		usingRoundParen := repl.Since(mark) == "$("
		closing := ifelseStr(usingRoundParen, ")", "}")

		varnameMark := repl.Mark()
		varname := p.Varname()
		if varname != "" {
			if usingRoundParen {
				p.line.Warn1("Please use curly braces {} instead of round parentheses () for %s.", varname)
			}
			modifiers := p.VarUseModifiers(varname, closing)
			if repl.AdvanceStr(closing) {
				return &MkVarUse{varname, modifiers}
			}
		}

		for p.VarUse() != nil || repl.AdvanceRegexp(`^([^$:`+closing+`]|\$\$)+`) {
		}
		rest := p.Rest()
		if hasPrefix(rest, ":L") || hasPrefix(rest, ":sh") || hasPrefix(rest, ":?") {
			varexpr := repl.Since(varnameMark)
			modifiers := p.VarUseModifiers(varexpr, closing)
			if repl.AdvanceStr(closing) {
				return &MkVarUse{varexpr, modifiers}
			}
		}
		repl.Reset(mark)
	}

	if repl.AdvanceStr("$@") {
		return &MkVarUse{"@", nil}
	}
	if repl.AdvanceStr("$<") {
		return &MkVarUse{"<", nil}
	}
	if repl.AdvanceRegexp(`^\$(\w)`) {
		varname := repl.m[1]
		p.line.Warn1("$%[1]s is ambiguous. Use ${%[1]s} if you mean a Makefile variable or $$%[1]s if you mean a shell variable.", varname)
		return &MkVarUse{varname, nil}
	}
	return nil
}

func (p *Parser) VarUseModifiers(varname, closing string) []string {
	repl := p.repl

	var modifiers []string
	mayOmitColon := false
	for repl.AdvanceStr(":") || mayOmitColon {
		mayOmitColon = false
		modifierMark := repl.Mark()

		switch repl.PeekByte() {
		case 'E', 'H', 'L', 'O', 'Q', 'R', 'T', 's', 't', 'u':
			if repl.AdvanceRegexp(`^(E|H|L|Ox?|Q|R|T|sh|tA|tW|tl|tu|tw|u)`) {
				modifiers = append(modifiers, repl.Since(modifierMark))
				continue
			}
			if repl.AdvanceStr("ts") {
				rest := repl.rest
				if len(rest) >= 2 && (rest[1] == closing[0] || rest[1] == ':') {
					repl.Skip(1)
				} else if len(rest) >= 1 && (rest[0] == closing[0] || rest[0] == ':') {
				} else if repl.AdvanceRegexp(`^\\\d+`) {
				} else {
					break
				}
				modifiers = append(modifiers, repl.Since(modifierMark))
				continue
			}

		case '=', 'D', 'M', 'N', 'U':
			if repl.AdvanceRegexp(`^[=DMNU]`) {
				for p.VarUse() != nil || repl.AdvanceRegexp(`^([^$:`+closing+`]|\$\$)+`) {
				}
				modifiers = append(modifiers, repl.Since(modifierMark))
				continue
			}

		case 'C', 'S':
			if repl.AdvanceRegexp(`^[CS]([%,/:;@^|])`) {
				separator := repl.m[1]
				repl.AdvanceStr("^")
				re := `^([^\` + separator + `$` + closing + `\\]|\$\$|\\.)+`
				for p.VarUse() != nil || repl.AdvanceRegexp(re) {
				}
				repl.AdvanceStr("$")
				if repl.AdvanceStr(separator) {
					for p.VarUse() != nil || repl.AdvanceRegexp(re) {
					}
					if repl.AdvanceStr(separator) {
						repl.AdvanceRegexp(`^[1gW]`)
						modifiers = append(modifiers, repl.Since(modifierMark))
						mayOmitColon = true
						continue
					}
				}
			}

		case '@':
			if repl.AdvanceRegexp(`^@([\w.]+)@`) {
				loopvar := repl.m[1]
				for p.VarUse() != nil || repl.AdvanceRegexp(`^([^$:@`+closing+`\\]|\$\$|\\.)+`) {
				}
				if !repl.AdvanceStr("@") {
					p.line.Warn2("Modifier ${%s:@%s@...@} is missing the final \"@\".", varname, loopvar)
				}
				modifiers = append(modifiers, repl.Since(modifierMark))
				continue
			}

		case '[':
			if repl.AdvanceRegexp(`^\[[-.\d]+\]`) {
				modifiers = append(modifiers, repl.Since(modifierMark))
				continue
			}

		case '?':
			repl.AdvanceStr("?")
			re := `^([^$:` + closing + `]|\$\$)+`
			for p.VarUse() != nil || repl.AdvanceRegexp(re) {
			}
			if repl.AdvanceStr(":") {
				for p.VarUse() != nil || repl.AdvanceRegexp(re) {
				}
				modifiers = append(modifiers, repl.Since(modifierMark))
				continue
			}
		}

		repl.Reset(modifierMark)
		for p.VarUse() != nil || repl.AdvanceRegexp(`^([^:$`+closing+`]|\$\$)+`) {
		}
		if suffixSubst := repl.Since(modifierMark); contains(suffixSubst, "=") {
			modifiers = append(modifiers, suffixSubst)
			continue
		}
	}
	return modifiers
}

func (p *Parser) MkCond() *Tree {
	return p.mkCondOr()
}

func (p *Parser) mkCondOr() *Tree {
	and := p.mkCondAnd()
	if and == nil {
		return nil
	}

	ands := append([]interface{}(nil), and)
	for {
		mark := p.repl.Mark()
		if !p.repl.AdvanceRegexp(`^\s*\|\|\s*`) {
			break
		}
		next := p.mkCondAnd()
		if next == nil {
			p.repl.Reset(mark)
			break
		}
		ands = append(ands, next)
	}
	if len(ands) == 1 {
		return and
	}
	return NewTree("or", ands...)
}

func (p *Parser) mkCondAnd() *Tree {
	atom := p.mkCondAtom()
	if atom == nil {
		return nil
	}

	atoms := append([]interface{}(nil), atom)
	for {
		mark := p.repl.Mark()
		if !p.repl.AdvanceRegexp(`^\s*&&\s*`) {
			break
		}
		next := p.mkCondAtom()
		if next == nil {
			p.repl.Reset(mark)
			break
		}
		atoms = append(atoms, next)
	}
	if len(atoms) == 1 {
		return atom
	}
	return NewTree("and", atoms...)
}

func (p *Parser) mkCondAtom() *Tree {
	if G.opts.Debug {
		defer tracecall1(p.Rest())()
	}

	repl := p.repl
	mark := repl.Mark()
	repl.SkipSpace()
	switch {
	case repl.AdvanceStr("!"):
		cond := p.mkCondAtom()
		if cond != nil {
			return NewTree("not", cond)
		}
	case repl.AdvanceStr("("):
		cond := p.MkCond()
		if cond != nil {
			repl.SkipSpace()
			if repl.AdvanceStr(")") {
				return cond
			}
		}
	case repl.AdvanceRegexp(`^defined\s*\(`):
		if varname := p.Varname(); varname != "" {
			if repl.AdvanceStr(")") {
				return NewTree("defined", varname)
			}
		}
	case repl.AdvanceRegexp(`^empty\s*\(`):
		if varname := p.Varname(); varname != "" {
			modifiers := p.VarUseModifiers(varname, ")")
			if repl.AdvanceStr(")") {
				return NewTree("empty", MkVarUse{varname, modifiers})
			}
		}
	case repl.AdvanceRegexp(`^(commands|exists|make|target)\s*\(`):
		funcname := repl.m[1]
		argMark := repl.Mark()
		for p.VarUse() != nil || repl.AdvanceRegexp(`^[^$)]+`) {
		}
		arg := repl.Since(argMark)
		if repl.AdvanceStr(")") {
			return NewTree(funcname, arg)
		}
	default:
		lhs := p.VarUse()
		mark := repl.Mark()
		if lhs == nil && repl.AdvanceStr("\"") {
			if quotedLHS := p.VarUse(); quotedLHS != nil && repl.AdvanceStr("\"") {
				lhs = quotedLHS
			} else {
				repl.Reset(mark)
			}
		}
		if lhs != nil {
			if repl.AdvanceRegexp(`^\s*(<|<=|==|!=|>=|>)\s*(\d+(?:\.\d+)?)`) {
				return NewTree("compareVarNum", *lhs, repl.m[1], repl.m[2])
			}
			if repl.AdvanceRegexp(`^\s*(<|<=|==|!=|>=|>)\s*`) {
				op := repl.m[1]
				if (op == "!=" || op == "==") && repl.AdvanceRegexp(`^"([^"\$\\]*)"`) {
					return NewTree("compareVarStr", *lhs, op, repl.m[1])
				} else if repl.AdvanceRegexp(`^\w+`) {
					return NewTree("compareVarStr", *lhs, op, repl.m[0])
				} else if rhs := p.VarUse(); rhs != nil {
					return NewTree("compareVarVar", *lhs, op, *rhs)
				}
			} else {
				return NewTree("not", NewTree("empty", *lhs)) // See devel/bmake/files/cond.c:/\* For \.if \$/
			}
		}
		if repl.AdvanceRegexp(`^\d+(?:\.\d+)?`) {
			return NewTree("literalNum", repl.m[0])
		}
	}
	repl.Reset(mark)
	return nil
}

type ShToken struct {
	Text        string
	StateChange bool
	NewState    string
}

func (st *ShToken) String() string {
	if st.StateChange {
		return st.Text + "[" + st.NewState + "]"
	}
	return st.Text
}

// See ShQuote.Feed
func (p *Parser) ShTokens() []*ShToken {
	const (
		reSkip = "^[^\"'`\\\\]+" // Characters that don’t influence the quoting mode.
		S      = "'"
		D      = "\""
		B      = "`"
	)

	var tokens []*ShToken
	repl := p.repl
	qstate := ""

	emitText := func() {
		tokens = append(tokens, &ShToken{repl.m[0], false, qstate})
	}
	emitState := func(newstate string) {
		qstate = newstate
		tokens = append(tokens, &ShToken{repl.s, true, qstate})
	}

	for repl.rest != "" {
		mark := repl.Mark()
		switch qstate {
		case "":
			switch {
			case repl.AdvanceStr(D):
				emitState(D)
			case repl.AdvanceStr(S):
				emitState(S)
			case repl.AdvanceStr(B):
				emitState(B)
			case repl.AdvanceRegexp(`^(?:` + reSkip + `|\\.)+`):
				emitText()
			}

		case D:
			switch {
			case repl.AdvanceStr(D):
				emitState("")
			case repl.AdvanceStr(B):
				emitState(D + B)
			case repl.AdvanceStr(S):
				emitState(D + S)
			case repl.AdvanceRegexp(`^(?:` + reSkip + `|\\.)+`):
				emitText()
			}

		case S:
			switch {
			case repl.AdvanceStr(S):
				emitState("")
			case repl.AdvanceRegexp(`^[^']+`):
				emitText()
			}

		case B:
			switch {
			case repl.AdvanceStr(B):
				emitState("")
			case repl.AdvanceStr(S):
				emitState(B + S)
			case repl.AdvanceRegexp(`^(?:` + reSkip + `|\\.)+`): // TODO: Lookup the exact rules
				emitText()
			}

		case D + B:
			switch {
			case repl.AdvanceStr(B):
				emitState(D)
			case repl.AdvanceStr(S):
				emitState(D + B + S)
			case repl.AdvanceRegexp(`^(?:` + reSkip + `|\\.)+`): // TODO: Lookup the exact rules
				emitText()
			}
		case D + B + S:
			switch {
			case repl.AdvanceStr(S):
				emitState(D + B)
			case repl.AdvanceRegexp(`^(?:` + reSkip + `|\\.)+`): // TODO: Lookup the exact rules
				emitText()
			}
		}

		if repl.Since(mark) == "" {
			traceStep2("Parser.ShTokens.stuck qstate=%s rest=%s", qstate, repl.rest)
			return append(tokens, &ShToken{repl.rest, true, "?"})
		}
	}
	return tokens
}

// @Beta
type ShLexemeType uint8

const (
	shlSpace  ShLexemeType = iota
	shlVaruse              // ${PREFIX}
	shlPlain
	shlDquot        // "..."
	shlSquot        // '...'
	shlBackt        // `...`
	shlSubshellOpen // $(
	shlSemicolon    // ;
	shlParenOpen    // (
	shlParenClose   // )
	shlBraceOpen    // {
	shlBraceClose   // }
	shlPipe         // |
	shlBackground   // &
	shlOr           // ||
	shlAnd          // &&
)

func (t ShLexemeType) String() string {
	return [...]string{
		"space",
		"varuse",
		"dquot", "squot", "backt",
		"subshellOpen",
		"semicolon",
		"parenOpen", "parenClose",
		"braceOpen", "braceClose",
		"pipe",
		"background",
		"or",
		"and",
	}[t]
}

type ShLexeme struct {
	Type ShLexemeType
	Text string
	Data interface{}
}

func (shl *ShLexeme) String() string {
	return shl.Text
}

func (p *Parser) ShLexeme() *ShLexeme {
	const rePlain = `^([\w-\[\]=/]|\$\$)+`
	repl := p.repl
	mark := repl.Mark()

	if varuse := p.VarUse(); varuse != nil {
		return &ShLexeme{shlVaruse, repl.Since(mark), varuse}
	}

	switch {
	case repl.AdvanceRegexp(rePlain):
		return &ShLexeme{shlPlain, repl.Since(mark), nil}
	case repl.AdvanceRegexp(`^[ \t]+`):
		return &ShLexeme{shlSpace, repl.Since(mark), nil}

	case repl.AdvanceStr(";"):
		return &ShLexeme{shlSemicolon, ";", nil}
	case repl.AdvanceStr("("):
		return &ShLexeme{shlParenOpen, "(", nil}
	case repl.AdvanceStr(")"):
		return &ShLexeme{shlParenClose, ")", nil}

	case repl.AdvanceRegexp("^\"(?:\\\\.|[^\"\\\\`$]|`[^\"\\\\`$']*`)*\""):
		return &ShLexeme{shlDquot, ")", nil} // TODO: unescape

	case repl.AdvanceRegexp("^`(?:\\\\.|[^\"\\\\`])*`"):
		return &ShLexeme{shlBackt, ")", nil} // TODO: unescape
	}
	repl.Reset(mark)
	return nil
}

func (p *Parser) ShLexemes() []*ShLexeme {
	var result []*ShLexeme

nextshlexeme:
	if !p.EOF() {
		shlexeme := p.ShLexeme()
		if shlexeme != nil {
			result = append(result, shlexeme)
			goto nextshlexeme
		}
	}
	return result
}

type ShCommand struct {
	Varassigns []*ShVarassign
	Command    *ShWord
	Args       []*ShWord
}

func (shcmd *ShCommand) String() string {
	return fmt.Sprintf("ShCommand(%v, %v, %v)", shcmd.Varassigns, shcmd.Command, shcmd.Args)
}

type ShWord struct {
	Atoms []*ShLexeme
}

func (shword *ShWord) String() string {
	return fmt.Sprintf("ShWord(%q)", shword.Atoms)
}

type ShVarassign struct {
	Name  string
	Value *ShWord // maybe
}

func (shva *ShVarassign) String() string {
	return fmt.Sprintf("ShVarassign(%s=%v)", shva.Name, shva.Value)
}

// @Beta
func (p *Parser) ShCommand() *ShCommand {
	var varassigns []*ShVarassign
nextvarassign:
	if varassign := p.ShVarassign(); varassign != nil {
		varassigns = append(varassigns, varassign)
		goto nextvarassign
	}

	command := p.ShWord()

	var args []*ShWord
nextarg:
	if arg := p.ShWord(); arg != nil {
		args = append(args, arg)
		goto nextarg
	}
	if len(varassigns) != 0 || command != nil {
		return &ShCommand{varassigns, command, args}
	}
	return nil
}

func (p *Parser) ShVarassign() *ShVarassign {
	mark := p.repl.Mark()
	if p.repl.AdvanceRegexp(`^(\w+)=`) {
		varname := p.repl.m[1]
		value := p.ShWord()
		if value != nil {
			return &ShVarassign{varname, value}
		}
	}
	p.repl.Reset(mark)
	return nil
}

func (p *Parser) ShWord() *ShWord {
	shword := &ShWord{}
nextlex:
	lex := p.ShLexeme()
	if lex != nil {
		switch lex.Type {
		case shlVaruse, shlPlain, shlDquot, shlSquot, shlBackt:
			shword.Atoms = append(shword.Atoms, lex)
			goto nextlex
		}
	}
	if len(shword.Atoms) != 0 {
		return shword
	}
	return nil
}
