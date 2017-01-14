package main

import (
	"netbsd.org/pkglint/textproc"
	"strings"
)

type Parser struct {
	Line         ILine
	repl         *textproc.PrefixReplacer
	EmitWarnings bool
}

func NewParser(line ILine, s string, emitWarnings bool) *Parser {
	return &Parser{line, textproc.NewPrefixReplacer(s), emitWarnings}
}

func (p *Parser) EOF() bool {
	return p.repl.EOF()
}

func (p *Parser) Rest() string {
	return p.repl.Rest()
}

func (p *Parser) PkgbasePattern() (pkgbase string) {
	repl := p.repl

	for {
		if repl.AdvanceRegexp(`^\$\{\w+\}`) ||
			repl.AdvanceRegexp(`^[\w.*+,{}]+`) ||
			repl.AdvanceRegexp(`^\[[\d-]+\]`) {
			pkgbase += repl.Group(0)
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

type DependencyPattern struct {
	pkgbase  string // "freeciv-client", "{gcc48,gcc48-libs}", "${EMACS_REQD}"
	lowerOp  string // ">=", ">"
	lower    string // "2.5.0", "${PYVER}"
	upperOp  string // "<", "<="
	upper    string // "3.0", "${PYVER}"
	wildcard string // "[0-9]*", "1.5.*", "${PYVER}"
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
		op := repl.Str()
		if repl.AdvanceRegexp(`^(?:(?:\$\{\w+\})+|\d[\w.]*)`) {
			dp.lowerOp = op
			dp.lower = repl.Group(0)
		} else {
			repl.Reset(mark2)
		}
	}
	if repl.AdvanceStr("<=") || repl.AdvanceStr("<") {
		op := repl.Str()
		if repl.AdvanceRegexp(`^(?:(?:\$\{\w+\})+|\d[\w.]*)`) {
			dp.upperOp = op
			dp.upper = repl.Group(0)
		} else {
			repl.Reset(mark2)
		}
	}
	if dp.lowerOp != "" || dp.upperOp != "" {
		return &dp
	}
	if repl.AdvanceStr("-") && !repl.EOF() {
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
