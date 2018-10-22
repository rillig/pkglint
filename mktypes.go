package main

import (
	"netbsd.org/pkglint/textproc"
	"unicode"
)

// MkToken represents a contiguous string from a Makefile.
// It is either a literal string or a variable use.
//
// Example (3 tokens): /usr/share/${PKGNAME}/data
type MkToken struct {
	Text   string // Used for both literals and varuses.
	Varuse *MkVarUse
}

// MkVarUse represents a reference to a Make variable, with optional modifiers.
//
// For nested variable expressions, the variable name can contain references
// to other variables. For example, ${TOOLS.${t}} is a MkVarUse with varname
// "TOOLS.${t}" and no modifiers.
//
// Example: ${PKGNAME}
//
// Example: ${PKGNAME:S/from/to/}
type MkVarUse struct {
	varname   string             // E.g. "PKGNAME", or "${BUILD_DEFS}"
	modifiers []MkVarUseModifier // E.g. "Q", "S/from/to/"
}

type MkVarUseModifier struct {
	Text string
}

func (m MkVarUseModifier) IsQ() bool { return m.Text == "Q" }

func (m MkVarUseModifier) IsSuffixSubst() bool {
	// XXX: There are other cases
	return hasPrefix(m.Text, "=")
}

func (m MkVarUseModifier) MatchSubst() (ok bool, regex bool, from string, to string, options string) {
	l := textproc.NewLexer(m.Text)
	regex = l.PeekByte() == 'C'
	if l.NextByte('S') || l.NextByte('C') {
		separator := l.PeekByte()
		l.Skip(1)
		if unicode.IsPunct(rune(separator)) || separator == '|' {
			fromStart := l.Mark()
			noSeparator := func(b byte) bool { return int(b) != separator && b != '\\' }
			for l.NextBytesFunc(noSeparator) != "" {
				if l.PeekByte() == '\\' && len(l.Rest()) >= 2 {
					// TODO: Compare with devel/bmake for the exact behavior
					l.Skip(2)
				}
			}
			from = l.Since(fromStart)
			if from != "" && l.NextByte(byte(separator)) {
				toStart := l.Mark()
				for l.NextBytesFunc(noSeparator) != "" {
					if l.PeekByte() == '\\' && len(l.Rest()) >= 2 {
						// TODO: Compare with devel/bmake for the exact behavior
						l.Skip(2)
					}
				}
				to = l.Since(toStart)
				if l.NextByte(byte(separator)) {
					options = l.Rest()
					ok = true
					return
				}
			}
		}
	}
	return
}

func (m MkVarUseModifier) MatchMatch() (ok bool, positive bool, pattern string) {
	if hasPrefix(m.Text, "M") || hasPrefix(m.Text, "N") {
		return true, m.Text[0] == 'M', m.Text[1:]
	}
	return false, false, ""
}

func (m MkVarUseModifier) IsToLower() bool { return m.Text == "tl" }

func (vu *MkVarUse) Mod() string {
	mod := ""
	for _, modifier := range vu.modifiers {
		mod += ":" + modifier.Text
	}
	return mod
}

// IsExpression returns whether the varname is interpreted as a variable
// name (the usual case) or as a full expression (rare, only the modifiers
// "?:" and "L" do this).
func (vu *MkVarUse) IsExpression() bool {
	if len(vu.modifiers) == 0 {
		return false
	}
	mod := vu.modifiers[0]
	return mod.Text == "L" || hasPrefix(mod.Text, "?")
}

func (vu *MkVarUse) IsQ() bool {
	mlen := len(vu.modifiers)
	return mlen > 0 && vu.modifiers[mlen-1].IsQ()
}
