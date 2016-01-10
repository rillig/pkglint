package main

import (
	"strings"
)

type Parser struct {
	repl *PrefixReplacer
}

func NewParser(s string) *Parser {
	return &Parser{NewPrefixReplacer(s)}
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
	literal string
	varuse  MkVarUse
}
type MkVarUse struct {
	varname   string
	modifiers []string
}

func (p *Parser) MkTokens() []*MkToken {
	var tokens []*MkToken

	for {
		if varuse := p.VarUse(); varuse != nil {
			tokens = append(tokens, &MkToken{varuse: *varuse})
		} else if p.repl.AdvanceRegexp(`^([^$\\]+|\$\$|\\.|\$$)+`) {
			tokens = append(tokens, &MkToken{literal: p.repl.m[0]})
		} else {
			return tokens
		}
	}
}

func (p *Parser) Varname() string {
	repl := p.repl

	mark := repl.Mark()
	if repl.AdvanceRegexp(`^\.?\w+`) {
		if repl.AdvanceStr(".") {
			if repl.AdvanceRegexp(`^[\w-+.]+`) || p.VarUse() != nil {
				return repl.Since(mark)
			}
		} else {
			if hasSuffix(repl.Since(mark), "_") {
				p.VarUse()
			}
			return repl.Since(mark)
		}
	}

	repl.Reset(mark)
	return ""
}

func (p *Parser) VarUse() *MkVarUse {
	repl := p.repl

	mark := repl.Mark()
	if repl.AdvanceStr("${") {
		varname := p.Varname()
		if varname != "" {
			var modifiers []string
			for repl.AdvanceStr(":") {
				modifierMark := repl.Mark()
				switch {
				case repl.AdvanceRegexp(`^[MN][\w*\-=?\[\]]+`),
					repl.AdvanceRegexp(`^(E|H|L|Ox?|Q|R|T|sh|tA|tW|tl|ts.|tu|tw|u)`),
					repl.AdvanceRegexp(`^C/\^?([^/$\\]*|\\.|\$\{\w+\})*\$?/(\\\d|[^/$\\]+|\$\{\w+\})*/[1gW]?`),
					repl.AdvanceRegexp(`^=[\w-./]+`): // Special form of ${VAR:.c=.o}
					modifiers = append(modifiers, repl.Since(modifierMark))
					continue
				}

				if repl.AdvanceStr("M") {
					for p.VarUse() != nil || repl.AdvanceRegexp(`^[\w*\-=?\[\]]+`) {
					}
					modifiers = append(modifiers, repl.Since(modifierMark))
					continue
				}
				repl.Reset(modifierMark)

				if repl.AdvanceRegexp(`^S([,/|])`) {
					separator := repl.m[1]
					re := `^([^` + separator + `$\\]|\\.)+`
					repl.AdvanceStr("^")
					for p.VarUse() != nil || repl.AdvanceRegexp(re) {
					}
					repl.AdvanceStr("$")
					if repl.AdvanceStr(separator) {
						for p.VarUse() != nil || repl.AdvanceRegexp(re) {
						}
						if repl.AdvanceStr(separator) {
							repl.AdvanceRegexp(`^[1gW]`)
							modifiers = append(modifiers, repl.Since(modifierMark))
							continue
						}
					}
				}
				repl.Reset(modifierMark)

				if repl.AdvanceStr("D") || repl.AdvanceStr("U") {
					if p.VarUse() != nil || repl.AdvanceRegexp(`^\w+`) {
						modifiers = append(modifiers, repl.Since(modifierMark))
						continue
					}
				}

				goto fail
			}
			if repl.AdvanceStr("}") {
				return &MkVarUse{varname, modifiers}
			}
		}
	}
fail:
	repl.Reset(mark)
	return nil
}
