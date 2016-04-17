package main

type ShParser struct {
	*Parser
}

func NewShParser(line *Line, text string) *ShParser {
	return &ShParser{NewParser(line, text)}
}

// See ShQuote.Feed
func (p *ShParser) ShAtom(quoting ShQuoting) *ShAtom {
	if p.EOF() {
		return nil
	}

	repl := p.repl
	mark := repl.Mark()

	if varuse := p.VarUse(); varuse != nil {
		return &ShAtom{shtVaruse, repl.Since(mark), quoting, varuse}
	}

	var atom *ShAtom
	switch quoting {
	case shqPlain:
		atom = p.shAtomPlain()
	case shqDquot:
		atom = p.shAtomDquot()
	case shqSquot:
		atom = p.shAtomSquot()
	case shqBackt:
		atom = p.shAtomBackt()
	case shqDquotBackt:
		atom = p.shAtomDquotBackt()
	case shqBacktDquot:
		atom = p.shAtomBacktDquot()
	case shqBacktSquot:
		atom = p.shAtomBacktSquot()
	case shqDquotBacktDquot:
		atom = p.shAtomDquotBacktDquot()
	case shqDquotBacktSquot:
		atom = p.shAtomDquotBacktSquot()
	}

	if atom == nil {
		p.repl.Reset(mark)
		p.line.Warnf("Pkglint parse error in Parser.ShAtom at %q (quoting=%s)", repl.rest, quoting)
	}
	return atom
}

func (p *ShParser) shAtomPlain() *ShAtom {
	q := shqPlain
	repl := p.repl
	switch {
	case repl.AdvanceHspace():
		return &ShAtom{shtSpace, repl.s, q, nil}
	case repl.AdvanceStr(";;"):
		return &ShAtom{shtCaseSeparator, repl.s, q, nil}
	case repl.AdvanceStr(";"):
		return &ShAtom{shtSemicolon, repl.s, q, nil}
	case repl.AdvanceStr("("):
		return &ShAtom{shtParenOpen, repl.s, q, nil}
	case repl.AdvanceStr(")"):
		return &ShAtom{shtParenClose, repl.s, q, nil}
	case repl.AdvanceStr("||"):
		return &ShAtom{shtOr, repl.s, q, nil}
	case repl.AdvanceStr("&&"):
		return &ShAtom{shtAnd, repl.s, q, nil}
	case repl.AdvanceStr("|"):
		return &ShAtom{shtPipe, repl.s, q, nil}
	case repl.AdvanceStr("&"):
		return &ShAtom{shtBackground, repl.s, q, nil}
	case repl.AdvanceStr("\""):
		return &ShAtom{shtWord, repl.s, shqDquot, nil}
	case repl.AdvanceStr("'"):
		return &ShAtom{shtWord, repl.s, shqSquot, nil}
	case repl.AdvanceStr("`"):
		return &ShAtom{shtWord, repl.s, shqBackt, nil}
	case repl.AdvanceRegexp(`^(?:<|<<|>|>>|>&)`):
		return &ShAtom{shtRedirect, repl.m[0], q, nil}
	case repl.AdvanceRegexp(`^#.*`):
		return &ShAtom{shtComment, repl.m[0], q, nil}
	case repl.AdvanceRegexp(`^(?:[!#%*+,\-./0-9:=?@A-Z\[\]^_a-z{}~]+|\\[^$]|\\\$\$|` + reShVaruse + `|\$\$)+`):
		return &ShAtom{shtWord, repl.m[0], q, nil}
	}
	return nil
}

func (p *ShParser) shAtomDquot() *ShAtom {
	repl := p.repl
	switch {
	case repl.AdvanceStr("\""):
		return &ShAtom{shtWord, repl.s, shqPlain, nil}
	case repl.AdvanceStr("`"):
		return &ShAtom{shtWord, repl.s, shqDquotBackt, nil}
	case repl.AdvanceRegexp(`^(?:[\t !#%&'()*+,\-./0-9:;<=>?@A-Z\[\]^_a-z{|}~]+|\\[^$]|\\\$\$|` + reShVaruse + `|\$\$)+`):
		return &ShAtom{shtWord, repl.m[0], shqDquot, nil} // XXX: unescape?
	}
	return nil
}

func (p *ShParser) shAtomSquot() *ShAtom {
	repl := p.repl
	switch {
	case repl.AdvanceStr("'"):
		return &ShAtom{shtWord, repl.s, shqPlain, nil}
	case repl.AdvanceRegexp(`^([\t !"#%&()*+,\-./0-9:;<=>?@A-Z\[\\\]^_` + "`" + `a-z{|}~]+|\$\$)+`):
		return &ShAtom{shtWord, repl.m[0], shqSquot, nil}
	}
	return nil
}

func (p *ShParser) shAtomBackt() *ShAtom {
	const q = shqBackt
	repl := p.repl
	switch {
	case repl.AdvanceStr("\""):
		return &ShAtom{shtWord, repl.s, shqBacktDquot, nil}
	case repl.AdvanceStr("`"):
		return &ShAtom{shtWord, repl.s, shqPlain, nil}
	case repl.AdvanceStr("'"):
		return &ShAtom{shtWord, repl.s, shqBacktSquot, nil}
	case repl.AdvanceHspace():
		return &ShAtom{shtSpace, repl.s, q, nil}
	case repl.AdvanceStr(";;"):
		return &ShAtom{shtCaseSeparator, repl.s, q, nil}
	case repl.AdvanceStr(";"):
		return &ShAtom{shtSemicolon, repl.s, q, nil}
	case repl.AdvanceStr("("):
		return &ShAtom{shtParenOpen, repl.s, q, nil}
	case repl.AdvanceStr(")"):
		return &ShAtom{shtParenClose, repl.s, q, nil}
	case repl.AdvanceStr("||"):
		return &ShAtom{shtOr, repl.s, q, nil}
	case repl.AdvanceStr("&&"):
		return &ShAtom{shtAnd, repl.s, q, nil}
	case repl.AdvanceStr("|"):
		return &ShAtom{shtPipe, repl.s, q, nil}
	case repl.AdvanceStr("&"):
		return &ShAtom{shtBackground, repl.s, q, nil}
	case repl.AdvanceRegexp(`^(?:<|<<|>|>>|>&)`):
		return &ShAtom{shtRedirect, repl.s, q, nil}
	case repl.AdvanceRegexp("^#[^`]*"):
		return &ShAtom{shtComment, repl.s, q, nil}
	case repl.AdvanceRegexp(`^(?:[!#%*+,\-./0-9:=?@A-Z_a-z~]+|\\[^$]|\\\$\$|` + reShVaruse + `|\$\$)+`):
		return &ShAtom{shtWord, repl.s, q, nil}
	}
	return nil
}

func (p *ShParser) shAtomDquotBackt() *ShAtom {
	const q = shqDquotBackt
	repl := p.repl
	switch {
	case repl.AdvanceStr("`"):
		return &ShAtom{shtWord, repl.s, shqDquot, nil}
	case repl.AdvanceStr("\""):
		return &ShAtom{shtWord, repl.s, shqDquotBacktDquot, nil}
	case repl.AdvanceStr("'"):
		return &ShAtom{shtWord, repl.s, shqDquotBacktSquot, nil}
	case repl.AdvanceRegexp("^#[^`]*"):
		return &ShAtom{shtComment, repl.s, q, nil}
	case repl.AdvanceStr(";;"):
		return &ShAtom{shtCaseSeparator, repl.s, q, nil}
	case repl.AdvanceStr(";"):
		return &ShAtom{shtSemicolon, repl.s, q, nil}
	case repl.AdvanceStr("("):
		return &ShAtom{shtParenOpen, repl.s, q, nil}
	case repl.AdvanceStr(")"):
		return &ShAtom{shtParenClose, repl.s, q, nil}
	case repl.AdvanceStr("||"):
		return &ShAtom{shtOr, repl.s, q, nil}
	case repl.AdvanceStr("&&"):
		return &ShAtom{shtAnd, repl.s, q, nil}
	case repl.AdvanceStr("|"):
		return &ShAtom{shtPipe, repl.s, q, nil}
	case repl.AdvanceStr("&"):
		return &ShAtom{shtBackground, repl.s, q, nil}
	case repl.AdvanceRegexp(`^(?:<|<<|>|>>|>&)`):
		return &ShAtom{shtRedirect, repl.s, q, nil}
	case repl.AdvanceRegexp(`^(?:[!#%*+,\-./0-9:=?@A-Z_a-z~]+|\\[^$]|\\\$\$)+`):
		return &ShAtom{shtWord, repl.s, q, nil}
	case repl.AdvanceHspace():
		return &ShAtom{shtSpace, repl.s, q, nil}
	}
	return nil
}

func (p *ShParser) shAtomBacktDquot() *ShAtom {
	repl := p.repl
	switch {
	case repl.AdvanceStr("\""):
		return &ShAtom{shtWord, repl.s, shqBackt, nil}
	case repl.AdvanceRegexp(`^(?:[\t !%&()*+,\-./0-9:;<=>?@A-Z\[\]^_a-z{|}~]+|\\[^$]|\\\$\$)+`):
		return &ShAtom{shtWord, repl.m[0], shqBacktDquot, nil}
	}
	return nil
}

func (p *ShParser) shAtomBacktSquot() *ShAtom {
	const q = shqBacktSquot
	repl := p.repl
	switch {
	case repl.AdvanceStr("'"):
		return &ShAtom{shtWord, repl.s, shqBackt, nil}
	case repl.AdvanceRegexp(`^([\t !"#%&()*+,\-./0-9:;<=>?@A-Z\[\\\]^_` + "`" + `a-z{|}~]+|\$\$)+`):
		return &ShAtom{shtWord, repl.m[0], q, nil}
	}
	return nil
}

func (p *ShParser) shAtomDquotBacktDquot() *ShAtom {
	const q = shqDquotBacktDquot
	repl := p.repl
	switch {
	case repl.AdvanceStr("\""):
		return &ShAtom{shtWord, repl.s, shqDquotBackt, nil}
	case repl.AdvanceRegexp(`^(?:[\t !%&()*+,\-./0-9:;<=>?@A-Z\[\]^_a-z{|}~]+|\\[^$]|\\\$\$|` + reShVaruse + `)+`):
		return &ShAtom{shtWord, repl.m[0], q, nil}
	}
	return nil
}

func (p *ShParser) shAtomDquotBacktSquot() *ShAtom {
	repl := p.repl
	switch {
	case repl.AdvanceStr("'"):
		return &ShAtom{shtWord, repl.s, shqDquotBackt, nil}
	case repl.AdvanceRegexp(`^(?:[\t !"#%()*+,\-./0-9:;<=>?@A-Z\[\]^_a-z{|}~]+|\\[^$]|\\\$\$|\$\$)+`):
		return &ShAtom{shtWord, repl.m[0], shqDquotBacktSquot, nil}
	}
	return nil
}

func (p *ShParser) ShAtoms() []*ShAtom {
	var atoms []*ShAtom
	q := shqPlain
	for {
		atom := p.ShAtom(q)
		if atom == nil {
			return atoms
		}
		atoms = append(atoms, atom)
		q = atom.Quoting
	}
}

func (p *ShParser) ShToken() *ShToken {
	inimark := p.repl.Mark()
	q := shqPlain
	var atoms []*ShAtom

nextatom:
	mark := p.repl.Mark()
	atom := p.ShAtom(q)

	if atom == nil || atom.Quoting == shqPlain && atom.Type.IsTokenDelimiter() {
		goto end
	}

	switch {
	case atom.Type == shtComment:
		goto nextatom
	case atom.Type == shtVaruse,
		atom.Type == shtWord,
		atom.Type == shtSpace,
		atom.Quoting != shqPlain:
		atoms = append(atoms, atom)
		q = atom.Quoting
		goto nextatom
	default:
		p.repl.Reset(mark)
		p.line.Warnf("Pkglint parse error in Parser.ShWord at %q (atomtype=%s quoting=%s)", p.repl.rest, atom.Type, atom.Quoting)
		p.repl.Reset(inimark)
		return nil
	}

end:
	p.repl.Reset(mark)
	if len(atoms) != 0 {
		return &ShToken{p.repl.Since(inimark), atoms}
	}
	return nil
}

// @Beta
func (p *ShParser) ShSimpleCmd() *ShSimpleCmd {
	repl := p.repl
	mark := repl.Mark()

	var tokens []*ShToken
	_ = p.Hspace()

nexttoken:
	if token := p.ShToken(); token != nil {
		tokens = append(tokens, token)
		if !p.Hspace() {
			goto end
		}
		goto nexttoken
	}

end:
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

	p.repl.Reset(mark)
	return nil
}

// @Beta
func (p *ShParser) ShSimpleCmds() []*ShSimpleCmd {
	var cmds []*ShSimpleCmd

nextcommand:
	cmd := p.ShSimpleCmd()
	if cmd == nil && p.repl.AdvanceRegexp(`^#.*`) {
		goto nextcommand
	}
	if cmd == nil {
		return cmds
	}
	cmds = append(cmds, cmd)
	mark := p.repl.Mark()
	_ = p.Hspace()
	atom := p.ShAtom(shqPlain)
	if atom != nil && atom.Type.IsCommandDelimiter() {
		goto nextcommand
	}
	p.repl.Reset(mark)
	return cmds
}
