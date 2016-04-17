package main

// See ShQuote.Feed
func (p *Parser) ShToken(quoting ShQuoting) *ShToken {
	if p.EOF() {
		return nil
	}

	repl := p.repl
	mark := repl.Mark()

	if varuse := p.VarUse(); varuse != nil {
		return &ShToken{shtVaruse, repl.Since(mark), quoting, varuse}
	}

	var token *ShToken
	switch quoting {
	case shqPlain:
		token = p.shTokenPlain()
	case shqDquot:
		token = p.shTokenDquot()
	case shqSquot:
		token = p.shTokenSquot()
	case shqBackt:
		token = p.shTokenBackt()
	case shqDquotBackt:
		token = p.shTokenDquotBackt()
	case shqBacktDquot:
		token = p.shTokenBacktDquot()
	case shqBacktSquot:
		token = p.shTokenBacktSquot()
	case shqDquotBacktDquot:
		token = p.shTokenDquotBacktDquot()
	case shqDquotBacktSquot:
		token = p.shTokenDquotBacktSquot()
	}

	if token == nil {
		p.repl.Reset(mark)
		p.line.Warnf("Pkglint parse error in Parser.ShToken at %q (quoting=%s)", repl.rest, quoting)
	}
	return token
}

func (p *Parser) shTokenPlain() *ShToken {
	q := shqPlain
	repl := p.repl
	switch {
	case repl.AdvanceRegexp(`^[ \t]+`):
		return &ShToken{shtSpace, repl.m[0], q, nil}
	case repl.AdvanceStr(";;"):
		return &ShToken{shtCaseSeparator, repl.s, q, nil}
	case repl.AdvanceStr(";"):
		return &ShToken{shtSemicolon, repl.s, q, nil}
	case repl.AdvanceStr("("):
		return &ShToken{shtParenOpen, repl.s, q, nil}
	case repl.AdvanceStr(")"):
		return &ShToken{shtParenClose, repl.s, q, nil}
	case repl.AdvanceStr("||"):
		return &ShToken{shtOr, repl.s, q, nil}
	case repl.AdvanceStr("&&"):
		return &ShToken{shtAnd, repl.s, q, nil}
	case repl.AdvanceStr("|"):
		return &ShToken{shtPipe, repl.s, q, nil}
	case repl.AdvanceStr("&"):
		return &ShToken{shtBackground, repl.s, q, nil}
	case repl.AdvanceStr("\""):
		return &ShToken{shtWord, repl.s, shqDquot, nil}
	case repl.AdvanceStr("'"):
		return &ShToken{shtWord, repl.s, shqSquot, nil}
	case repl.AdvanceStr("`"):
		return &ShToken{shtWord, repl.s, shqBackt, nil}
	case repl.AdvanceRegexp(`^(?:<|<<|>|>>|>&)`):
		return &ShToken{shtRedirect, repl.m[0], q, nil}
	case repl.AdvanceRegexp(`^#.*`):
		return &ShToken{shtComment, repl.m[0], q, nil}
	case repl.AdvanceRegexp(`^(?:[!#%*+,\-./0-9:=?@A-Z\[\]^_a-z{}~]+|\\[^$]|\\\$\$|` + reShVaruse + `|\$\$)+`):
		return &ShToken{shtWord, repl.m[0], q, nil}
	}
	return nil
}

func (p *Parser) shTokenDquot() *ShToken {
	repl := p.repl
	switch {
	case repl.AdvanceStr("\""):
		return &ShToken{shtWord, repl.s, shqPlain, nil}
	case repl.AdvanceStr("`"):
		return &ShToken{shtWord, repl.s, shqDquotBackt, nil}
	case repl.AdvanceRegexp(`^(?:[\t !#%&'()*+,\-./0-9:;<=>?@A-Z\[\]^_a-z{|}~]+|\\[^$]|\\\$\$|` + reShVaruse + `|\$\$)+`):
		return &ShToken{shtWord, repl.m[0], shqDquot, nil} // XXX: unescape?
	}
	return nil
}

func (p *Parser) shTokenSquot() *ShToken {
	repl := p.repl
	switch {
	case repl.AdvanceStr("'"):
		return &ShToken{shtWord, repl.s, shqPlain, nil}
	case repl.AdvanceRegexp(`^([\t !"#%&()*+,\-./0-9:;<=>?@A-Z\[\\\]^_` + "`" + `a-z{|}~]+|\$\$)+`):
		return &ShToken{shtWord, repl.m[0], shqSquot, nil}
	}
	return nil
}

func (p *Parser) shTokenBackt() *ShToken {
	q := shqBackt
	repl := p.repl
	switch {
	case repl.AdvanceStr("\""):
		return &ShToken{shtWord, repl.s, shqBacktDquot, nil}
	case repl.AdvanceStr("`"):
		return &ShToken{shtWord, repl.s, shqPlain, nil}
	case repl.AdvanceStr("'"):
		return &ShToken{shtWord, repl.s, shqBacktSquot, nil}
	case repl.AdvanceRegexp(`^[ \t]+`):
		return &ShToken{shtSpace, repl.m[0], shqBackt, nil}
	case repl.AdvanceStr(";;"):
		return &ShToken{shtCaseSeparator, repl.s, q, nil}
	case repl.AdvanceStr(";"):
		return &ShToken{shtSemicolon, repl.s, q, nil}
	case repl.AdvanceStr("("):
		return &ShToken{shtParenOpen, repl.s, q, nil}
	case repl.AdvanceStr(")"):
		return &ShToken{shtParenClose, repl.s, q, nil}
	case repl.AdvanceStr("||"):
		return &ShToken{shtOr, repl.s, q, nil}
	case repl.AdvanceStr("&&"):
		return &ShToken{shtAnd, repl.s, q, nil}
	case repl.AdvanceStr("|"):
		return &ShToken{shtPipe, repl.s, q, nil}
	case repl.AdvanceStr("&"):
		return &ShToken{shtBackground, repl.s, q, nil}
	case repl.AdvanceRegexp(`^(?:<|<<|>|>>|>&)`):
		return &ShToken{shtRedirect, repl.m[0], q, nil}
	case repl.AdvanceRegexp("^#[^`]*"):
		return &ShToken{shtComment, repl.m[0], q, nil}
	case repl.AdvanceRegexp(`^(?:[!#%*+,\-./0-9:=?@A-Z_a-z~]+|\\[^$]|\\\$\$|` + reShVaruse + `|\$\$)+`):
		return &ShToken{shtWord, repl.m[0], q, nil}
	}
	return nil
}

func (p *Parser) shTokenDquotBackt() *ShToken {
	const q = shqDquotBackt
	repl := p.repl
	switch {
	case repl.AdvanceStr("`"):
		return &ShToken{shtWord, repl.s, shqDquot, nil}
	case repl.AdvanceStr("\""):
		return &ShToken{shtWord, repl.s, shqDquotBacktDquot, nil}
	case repl.AdvanceStr("'"):
		return &ShToken{shtWord, repl.s, shqDquotBacktSquot, nil}
	case repl.AdvanceRegexp("^#[^`]*"):
		return &ShToken{shtComment, repl.m[0], q, nil}
	case repl.AdvanceRegexp(`^(?:[!#%*+,\-./0-9:=?@A-Z_a-z~]+|\\[^$]|\\\$\$)+`):
		return &ShToken{shtWord, repl.m[0], q, nil}
	case repl.AdvanceRegexp(`^[ \t]+`):
		return &ShToken{shtSpace, repl.m[0], q, nil}
	case repl.AdvanceStr("|"):
		return &ShToken{shtPipe, repl.s, q, nil}
	}
	return nil
}

func (p *Parser) shTokenBacktDquot() *ShToken {
	repl := p.repl
	switch {
	case repl.AdvanceStr("\""):
		return &ShToken{shtWord, repl.s, shqBackt, nil}
	case repl.AdvanceRegexp(`^(?:[\t !%&()*+,\-./0-9:;<=>?@A-Z\[\]^_a-z{|}~]+|\\[^$]|\\\$\$)+`):
		return &ShToken{shtWord, repl.m[0], shqBacktDquot, nil}
	}
	return nil
}

func (p *Parser) shTokenBacktSquot() *ShToken {
	const q = shqBacktSquot
	repl := p.repl
	switch {
	case repl.AdvanceStr("'"):
		return &ShToken{shtWord, repl.s, shqBackt, nil}
	case repl.AdvanceRegexp(`^([\t !"#%&()*+,\-./0-9:;<=>?@A-Z\[\\\]^_` + "`" + `a-z{|}~]+|\$\$)+`):
		return &ShToken{shtWord, repl.m[0], q, nil}
	}
	return nil
}

func (p *Parser) shTokenDquotBacktDquot() *ShToken {
	const q = shqDquotBacktDquot
	repl := p.repl
	switch {
	case repl.AdvanceStr("\""):
		return &ShToken{shtWord, repl.s, shqDquotBackt, nil}
	case repl.AdvanceRegexp(`^(?:[\t !%&()*+,\-./0-9:;<=>?@A-Z\[\]^_a-z{|}~]+|\\[^$]|\\\$\$|` + reShVaruse + `)+`):
		return &ShToken{shtWord, repl.m[0], q, nil}
	}
	return nil
}

func (p *Parser) shTokenDquotBacktSquot() *ShToken {
	repl := p.repl
	switch {
	case repl.AdvanceStr("'"):
		return &ShToken{shtWord, repl.s, shqDquotBackt, nil}
	case repl.AdvanceRegexp(`^(?:[\t !"#%()*+,\-./0-9:;<=>?@A-Z\[\]^_a-z{|}~]+|\\[^$]|\\\$\$|\$\$)+`):
		return &ShToken{shtWord, repl.m[0], shqDquotBacktSquot, nil}
	}
	return nil
}

func (p *Parser) ShTokens() []*ShToken {
	var tokens []*ShToken
	q := shqPlain
	for {
		token := p.ShToken(q)
		if token == nil {
			return tokens
		}
		tokens = append(tokens, token)
		q = token.Quoting
	}
}

func (p *Parser) ShWord() *ShWord {
	shword := &ShWord{}
	inimark := p.repl.Mark()
	q := shqPlain
nexttoken:
	mark := p.repl.Mark()
	token := p.ShToken(q)

	if token == nil {
		goto end
	}
	if token.Quoting == shqPlain {
		switch token.Type {
		case shtSpace, shtSemicolon, shtPipe, shtBackground, shtOr, shtAnd:
			goto end
		}
	}

	switch {
	case token.Type == shtComment:
		goto nexttoken
	case token.Type == shtVaruse,
		token.Type == shtWord,
		token.Type == shtSpace,
		token.Quoting != shqPlain:
		shword.Atoms = append(shword.Atoms, token)
		q = token.Quoting
		goto nexttoken
	default:
		p.repl.Reset(mark)
		p.line.Warnf("Pkglint parse error in Parser.ShWord at %q (tokentype=%s quoting=%s)", p.repl.rest, token.Type, token.Quoting)
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
nexttoken:
	token := p.ShToken(shqPlain)
	if token != nil {
		switch token.Type {
		case shtSpace:
			goto nexttoken
		case shtSemicolon, shtBackground, shtAnd, shtOr:
			goto nextcommand
		}
	}
	p.repl.Reset(mark)
	return cmds
}
