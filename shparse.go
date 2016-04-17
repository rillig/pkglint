package main

// See ShQuote.Feed
func (p *Parser) ShAtom(quoting ShQuoting) *ShAtom {
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

func (p *Parser) shAtomPlain() *ShAtom {
	q := shqPlain
	repl := p.repl
	switch {
	case repl.AdvanceRegexp(`^[ \t]+`):
		return &ShAtom{shtSpace, repl.m[0], q, nil}
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

func (p *Parser) shAtomDquot() *ShAtom {
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

func (p *Parser) shAtomSquot() *ShAtom {
	repl := p.repl
	switch {
	case repl.AdvanceStr("'"):
		return &ShAtom{shtWord, repl.s, shqPlain, nil}
	case repl.AdvanceRegexp(`^([\t !"#%&()*+,\-./0-9:;<=>?@A-Z\[\\\]^_` + "`" + `a-z{|}~]+|\$\$)+`):
		return &ShAtom{shtWord, repl.m[0], shqSquot, nil}
	}
	return nil
}

func (p *Parser) shAtomBackt() *ShAtom {
	q := shqBackt
	repl := p.repl
	switch {
	case repl.AdvanceStr("\""):
		return &ShAtom{shtWord, repl.s, shqBacktDquot, nil}
	case repl.AdvanceStr("`"):
		return &ShAtom{shtWord, repl.s, shqPlain, nil}
	case repl.AdvanceStr("'"):
		return &ShAtom{shtWord, repl.s, shqBacktSquot, nil}
	case repl.AdvanceRegexp(`^[ \t]+`):
		return &ShAtom{shtSpace, repl.m[0], shqBackt, nil}
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
		return &ShAtom{shtRedirect, repl.m[0], q, nil}
	case repl.AdvanceRegexp("^#[^`]*"):
		return &ShAtom{shtComment, repl.m[0], q, nil}
	case repl.AdvanceRegexp(`^(?:[!#%*+,\-./0-9:=?@A-Z_a-z~]+|\\[^$]|\\\$\$|` + reShVaruse + `|\$\$)+`):
		return &ShAtom{shtWord, repl.m[0], q, nil}
	}
	return nil
}

func (p *Parser) shAtomDquotBackt() *ShAtom {
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
		return &ShAtom{shtComment, repl.m[0], q, nil}
	case repl.AdvanceRegexp(`^(?:[!#%*+,\-./0-9:=?@A-Z_a-z~]+|\\[^$]|\\\$\$)+`):
		return &ShAtom{shtWord, repl.m[0], q, nil}
	case repl.AdvanceRegexp(`^[ \t]+`):
		return &ShAtom{shtSpace, repl.m[0], q, nil}
	case repl.AdvanceStr("|"):
		return &ShAtom{shtPipe, repl.s, q, nil}
	}
	return nil
}

func (p *Parser) shAtomBacktDquot() *ShAtom {
	repl := p.repl
	switch {
	case repl.AdvanceStr("\""):
		return &ShAtom{shtWord, repl.s, shqBackt, nil}
	case repl.AdvanceRegexp(`^(?:[\t !%&()*+,\-./0-9:;<=>?@A-Z\[\]^_a-z{|}~]+|\\[^$]|\\\$\$)+`):
		return &ShAtom{shtWord, repl.m[0], shqBacktDquot, nil}
	}
	return nil
}

func (p *Parser) shAtomBacktSquot() *ShAtom {
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

func (p *Parser) shAtomDquotBacktDquot() *ShAtom {
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

func (p *Parser) shAtomDquotBacktSquot() *ShAtom {
	repl := p.repl
	switch {
	case repl.AdvanceStr("'"):
		return &ShAtom{shtWord, repl.s, shqDquotBackt, nil}
	case repl.AdvanceRegexp(`^(?:[\t !"#%()*+,\-./0-9:;<=>?@A-Z\[\]^_a-z{|}~]+|\\[^$]|\\\$\$|\$\$)+`):
		return &ShAtom{shtWord, repl.m[0], shqDquotBacktSquot, nil}
	}
	return nil
}

func (p *Parser) ShAtoms() []*ShAtom {
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

func (p *Parser) ShWord() *ShWord {
	shword := &ShWord{}
	inimark := p.repl.Mark()
	q := shqPlain
nextatom:
	mark := p.repl.Mark()
	atom := p.ShAtom(q)

	if atom == nil {
		goto end
	}
	if atom.Quoting == shqPlain {
		switch atom.Type {
		case shtSpace, shtSemicolon, shtPipe, shtBackground, shtOr, shtAnd:
			goto end
		}
	}

	switch {
	case atom.Type == shtComment:
		goto nextatom
	case atom.Type == shtVaruse,
		atom.Type == shtWord,
		atom.Type == shtSpace,
		atom.Quoting != shqPlain:
		shword.Atoms = append(shword.Atoms, atom)
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
	if len(shword.Atoms) != 0 {
		return shword
	}
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

// @Beta
func (p *Parser) ShSimpleCmd() *ShSimpleCmd {
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
		return &ShSimpleCmd{varassigns, command, args}
	}

	p.repl.Reset(mark)
	return nil
}

// @Beta
func (p *Parser) ShSimpleCmds() []*ShSimpleCmd {
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
nextatom:
	atom := p.ShAtom(shqPlain)
	if atom != nil {
		switch atom.Type {
		case shtSpace:
			goto nextatom
		case shtSemicolon, shtBackground, shtAnd, shtOr:
			goto nextcommand
		}
	}
	p.repl.Reset(mark)
	return cmds
}
