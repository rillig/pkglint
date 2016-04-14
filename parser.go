package main

import (
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

// Whether the varname is interpreted as a variable name (the usual case)
// or as a full expression (rare).
func (vu *MkVarUse) IsExpression() bool {
	if len(vu.modifiers) == 0 {
		return false
	}
	mod := vu.modifiers[0]
	return mod == "L" || hasPrefix(mod, "?")
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

func (p *Parser) ShLexemes() []*ShLexeme {
	var shlexemes []*ShLexeme
	q := shqPlain
	for {
		lex := p.ShLexeme(q)
		if lex == nil {
			return shlexemes
		}
		shlexemes = append(shlexemes, lex)
		q = lex.Quoting
	}
}

// See ShQuote.Feed
func (p *Parser) ShLexeme(quoting ShQuoting) *ShLexeme {
	if p.EOF() {
		return nil
	}

	repl := p.repl
	mark := repl.Mark()

	if varuse := p.VarUse(); varuse != nil {
		return &ShLexeme{shlVaruse, repl.Since(mark), quoting, varuse}
	}

	var lex *ShLexeme
	switch quoting {
	case shqPlain:
		lex = p.shLexemePlain()
	case shqDquot:
		lex = p.shLexemeDquot()
	case shqSquot:
		lex = p.shLexemeSquot()
	case shqBackt:
		lex = p.shLexemeBackt()
	case shqDquotBackt:
		lex = p.shLexemeDquotBackt()
	case shqBacktDquot:
		lex = p.shLexemeBacktDquot()
	case shqDquotBacktSquot:
		lex = p.shLexemeDquotBacktSquot()
	}

	if lex == nil {
		p.repl.Reset(mark)
		p.line.Warnf("Parser.ShLexeme.stuck quoting=%s rest=%s", quoting, repl.rest)
	}
	return lex
}

func (p *Parser) shLexemePlain() *ShLexeme {
	q := shqPlain
	repl := p.repl
	switch {
	case repl.AdvanceRegexp(`^[ \t]+`):
		return &ShLexeme{shlSpace, repl.m[0], q, nil}
	case repl.AdvanceStr(";"):
		return &ShLexeme{shlSemicolon, repl.s, q, nil}
	case repl.AdvanceStr("("):
		return &ShLexeme{shlParenOpen, repl.s, q, nil}
	case repl.AdvanceStr(")"):
		return &ShLexeme{shlParenClose, repl.s, q, nil}
	case repl.AdvanceStr("||"):
		return &ShLexeme{shlOr, repl.s, q, nil}
	case repl.AdvanceStr("&&"):
		return &ShLexeme{shlAnd, repl.s, q, nil}
	case repl.AdvanceStr("|"):
		return &ShLexeme{shlPipe, repl.s, q, nil}
	case repl.AdvanceStr("&"):
		return &ShLexeme{shlBackground, repl.s, q, nil}
	case repl.AdvanceStr("\""):
		return &ShLexeme{shlText, repl.s, shqDquot, nil}
	case repl.AdvanceStr("'"):
		return &ShLexeme{shlText, repl.s, shqSquot, nil}
	case repl.AdvanceStr("`"):
		return &ShLexeme{shlText, repl.s, shqBackt, nil}
	case repl.AdvanceRegexp(`^(?:[!%*+,\-./0-9:=?@A-Z^_a-z~]+|\\.|\$\$\w+|\$\$\{\w+\})+`):
		return &ShLexeme{shlText, repl.m[0], q, nil}
	case repl.AdvanceRegexp(`^(?:<|<<|>|>>|>&)`):
		return &ShLexeme{shlRedirect, repl.m[0], q, nil}
	}
	return nil
}

func (p *Parser) shLexemeDquot() *ShLexeme {
	repl := p.repl
	switch {
	case repl.AdvanceStr("\""):
		return &ShLexeme{shlText, repl.s, shqPlain, nil}
	case repl.AdvanceStr("`"):
		return &ShLexeme{shlText, repl.s, shqDquotBackt, nil}
	case repl.AdvanceRegexp(`^(?:[\t !#%&'()*+,\-./0-9:;<=>?@A-Z\[\]^_a-z{|}~]+|\\.|\$\$\w+|\$\$\{\w+\})+`):
		return &ShLexeme{shlText, repl.m[0], shqDquot, nil} // XXX: unescape?
	}
	return nil
}

func (p *Parser) shLexemeSquot() *ShLexeme {
	repl := p.repl
	switch {
	case repl.AdvanceStr("'"):
		return &ShLexeme{shlText, repl.s, shqPlain, nil}
	case repl.AdvanceRegexp(`^([\t !"#%&()*+,\-./0-9:;<=>?@A-Z\[\\\]^_` + "`" + `a-z{|}~]+|\$\$)+`):
		return &ShLexeme{shlText, repl.m[0], shqSquot, nil}
	}
	return nil
}

func (p *Parser) shLexemeBackt() *ShLexeme {
	repl := p.repl
	switch {
	case repl.AdvanceStr("\""):
		return &ShLexeme{shlText, repl.s, shqBacktDquot, nil}
	case repl.AdvanceStr("`"):
		return &ShLexeme{shlText, repl.s, shqPlain, nil}
	case repl.AdvanceStr("'"):
		return &ShLexeme{shlText, repl.s, shqBacktSquot, nil}
	case repl.AdvanceRegexp(`^[ \t]+`):
		return &ShLexeme{shlSpace, repl.m[0], shqBackt, nil}
	case repl.AdvanceRegexp(`^(?:[!%*+,\-./0-9:=?@A-Z_a-z~]+|\\.|\$\$\w+|\$\$\{\w+\})+`):
		return &ShLexeme{shlText, repl.m[0], shqBackt, nil}
	}
	return nil
}

func (p *Parser) shLexemeDquotBackt() *ShLexeme {
	const q = "\"`"
	repl := p.repl
	switch {
	case repl.AdvanceStr("`"):
		return &ShLexeme{shlText, repl.s, shqDquot, nil}
	case repl.AdvanceStr("'"):
		return &ShLexeme{shlText, repl.s, shqDquotBacktSquot, nil}
	case repl.AdvanceRegexp(`^(?:[!%*+,\-./0-9:=?@A-Z_a-z~]+|\\.)+`):
		return &ShLexeme{shlText, repl.m[0], shqDquotBackt, nil}
	case repl.AdvanceRegexp(`^[ \t]+`):
		return &ShLexeme{shlSpace, repl.m[0], shqDquotBackt, nil}
	case repl.AdvanceStr("|"):
		return &ShLexeme{shlPipe, repl.s, shqDquotBackt, nil}
	}
	return nil
}

func (p *Parser) shLexemeBacktDquot() *ShLexeme {
	repl := p.repl
	switch {
	case repl.AdvanceStr("\""):
		return &ShLexeme{shlText, repl.s, shqBackt, nil}
	case repl.AdvanceRegexp(`^(?:[\t !%&()*+,\-./0-9:;<=>?@A-Z\[\]^_a-z{|}~]+|\\.)+`):
		return &ShLexeme{shlText, repl.m[0], shqBacktDquot, nil}
	}
	return nil
}

func (p *Parser) shLexemeDquotBacktSquot() *ShLexeme {
	repl := p.repl
	switch {
	case repl.AdvanceStr("'"):
		return &ShLexeme{shlText, repl.s, shqDquotBackt, nil}
	case repl.AdvanceRegexp(`^(?:[\t !"#%*+,\-./0-9:;<=>?@A-Z\[\]^_a-z{|}~]+|\\.|\$\$)+`):
		return &ShLexeme{shlText, repl.m[0], shqDquotBacktSquot, nil}
	}
	return nil
}

func (p *Parser) Hspace() bool {
	return p.repl.AdvanceRegexp(`^[ \t]+`)
}

// @Beta
func (p *Parser) ShCommand() *ShCommand {
	repl := p.repl
	mark := repl.Mark()

	var varassigns []*ShVarassign
	var command *ShWord
	var args []*ShWord

	_ = p.Hspace()

nextvarassign:
	if varassign := p.ShVarassign(); varassign != nil {
		varassigns = append(varassigns, varassign)
		if !p.Hspace() {
			goto end
		}
		goto nextvarassign
	}

	command = p.ShWord()
	if command == nil || !p.Hspace() {
		goto end
	}

nextarg:
	if arg := p.ShWord(); arg != nil {
		args = append(args, arg)
		if !p.Hspace() {
			goto end
		}
		goto nextarg
	}

end:
	if len(varassigns) != 0 || command != nil {
		return &ShCommand{varassigns, command, args}
	}

	p.repl.Reset(mark)
	return nil
}

func (p *Parser) ShVarassign() *ShVarassign {
	mark := p.repl.Mark()
	if p.repl.AdvanceRegexp(`^(\w+)=`) {
		varname := p.repl.m[1]
		value := p.ShWord()
		if value == nil {
			value = &ShWord{} // Assignment of empty value
		}
		return &ShVarassign{varname, value}
	}
	p.repl.Reset(mark)
	return nil
}

func (p *Parser) ShWord() *ShWord {
	shword := &ShWord{}
	inimark := p.repl.Mark()
	q := shqPlain
nextlex:
	mark := p.repl.Mark()
	lex := p.ShLexeme(q)
	if lex != nil {
		switch {
		case lex.Quoting == shqPlain && (lex.Type == shlSpace || lex.Type == shlSemicolon):
			break
		case lex.Type == shlVaruse,
			lex.Type == shlText,
			lex.Type == shlSpace,
			lex.Quoting != shqPlain:
			shword.Atoms = append(shword.Atoms, lex)
			q = lex.Quoting
			goto nextlex
		default:
			dummyLine.Errorf("Parse error at %q", mark)
			p.repl.Reset(inimark)
			return nil
		}
	}
	p.repl.Reset(mark)

	if len(shword.Atoms) != 0 {
		return shword
	}
	return nil
}
