package main

import (
	"netbsd.org/pkglint/textproc"
	"strings"
)

type Parser struct {
	Line         Line
	repl         *textproc.PrefixReplacer
	EmitWarnings bool
}

func NewParser(line Line, s string, emitWarnings bool) *Parser {
	return &Parser{line, G.NewPrefixReplacer(s), emitWarnings}
}

func (p *Parser) EOF() bool {
	return p.repl.Rest() == ""
}

func (p *Parser) Rest() string {
	return p.repl.Rest()
}

func (p *Parser) PkgbasePattern() (pkgbase string) {
	repl := p.repl

	for {
		mark := repl.Mark()

		if repl.SkipRegexp(`^\$\{\w+\}`) ||
			repl.SkipRegexp(`^[\w.*+,{}]+`) ||
			repl.SkipRegexp(`^\[[\d-]+\]`) {
			pkgbase += repl.Since(mark)
			continue
		}

		if repl.SkipByte('-') {
			if repl.SkipRegexp(`^\d`) ||
				repl.SkipRegexp(`^\$\{\w*VER\w*\}`) ||
				repl.SkipByte('[') {
				repl.Reset(mark)
				return
			}
			pkgbase += "-"
			continue
		}

		repl.Reset(mark)
		return
	}
}

type DependencyPattern struct {
	Pkgbase  string // "freeciv-client", "{gcc48,gcc48-libs}", "${EMACS_REQD}"
	LowerOp  string // ">=", ">"
	Lower    string // "2.5.0", "${PYVER}"
	UpperOp  string // "<", "<="
	Upper    string // "3.0", "${PYVER}"
	Wildcard string // "[0-9]*", "1.5.*", "${PYVER}"
}

func (p *Parser) Dependency() *DependencyPattern {
	repl := p.repl

	var dp DependencyPattern
	mark := repl.Mark()
	dp.Pkgbase = p.PkgbasePattern()
	if dp.Pkgbase == "" {
		return nil
	}

	mark2 := repl.Mark()
	op := repl.NextString(">=")
	if op == "" {
		op = repl.NextString(">")
	}
	if op != "" {
		if m := repl.NextRegexp(`^(?:(?:\$\{\w+\})+|\d[\w.]*)`); m != nil {
			dp.LowerOp = op
			dp.Lower = m[0]
		} else {
			repl.Reset(mark2)
		}
	}

	op = repl.NextString("<=")
	if op == "" {
		op = repl.NextString("<")
	}
	if op != "" {
		if m := repl.NextRegexp(`^(?:(?:\$\{\w+\})+|\d[\w.]*)`); m != nil {
			dp.UpperOp = op
			dp.Upper = m[0]
		} else {
			repl.Reset(mark2)
		}
	}
	if dp.LowerOp != "" || dp.UpperOp != "" {
		return &dp
	}
	if repl.SkipByte('-') && repl.Rest() != "" {
		dp.Wildcard = repl.NextRest()
		return &dp
	}
	if hasPrefix(dp.Pkgbase, "${") && hasSuffix(dp.Pkgbase, "}") {
		return &dp
	}
	if hasSuffix(dp.Pkgbase, "-*") {
		dp.Pkgbase = strings.TrimSuffix(dp.Pkgbase, "-*")
		dp.Wildcard = "*"
		return &dp
	}

	repl.Reset(mark)
	return nil
}
