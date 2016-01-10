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
	literal   string
	varname   string
	modifiers []string
}

func (p *Parser) MkTokens() []*MkToken {
	repl := p.repl

	var tokens []*MkToken

next:
	mark := repl.Mark()
	switch {
	case repl.AdvanceStr("${"):
		var token MkToken
		if varname := p.Varname(); varname != "" {
			token.varname = varname
			for repl.AdvanceStr(":") {
				var modifier string
				switch {
				case repl.AdvanceRegexp(`^M(\*|[\w-]+)`),
					repl.AdvanceRegexp(`^Q`),
					repl.AdvanceRegexp(`^S/\^?[\w+\-.]*\$?/[\w+\-.]*/g?`),
					repl.AdvanceRegexp(`^=[\w-./]+`): // Special form of ${VAR:.c=.o}
					modifier = repl.m[0]
				}
				if modifier != "" {
					token.modifiers = append(token.modifiers, modifier)
				} else {
					goto failvaruse
				}
			}
			if p.repl.AdvanceStr("}") {
				tokens = append(tokens, &token)
				goto next
			}
		}
	failvaruse:
		repl.Reset(mark)
		break

	case repl.AdvanceRegexp(`^([^$\\]+|\$\$|\\[\w"./()]|\$$)+`):
		tokens = append(tokens, &MkToken{literal: repl.m[0]})
		goto next

	}
	return tokens
}

func (p *Parser) Varname() string {
	repl := p.repl

	mark := repl.Mark()
	if repl.AdvanceRegexp(`^\.?\w+`) {
		varbase := repl.m[0]
		if !repl.AdvanceStr(".") {
			return varbase
		}
		if repl.AdvanceRegexp(`^[\w-+.]+`) {
			return varbase + "." + repl.m[0]
		}
		if repl.AdvanceRegexp(`^\$\{\w+\}`) {
			return varbase + "." + repl.m[0]
		}
	}

	repl.Reset(mark)
	return ""
}
