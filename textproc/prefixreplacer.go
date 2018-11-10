package textproc

import (
	"fmt"
	"netbsd.org/pkglint/regex"
	"strings"
)

var Testing bool

type PrefixReplacerMark string

// PrefixReplacer parses an arbitrary string into its components by repeatedly
// stripping off a prefix matched by a literal string or a regular expression.
type PrefixReplacer struct {
	rest string
	m    []string
	res  *regex.Registry
}

func NewPrefixReplacer(s string, res *regex.Registry) *PrefixReplacer {
	return &PrefixReplacer{s, nil, res}
}

func (pr *PrefixReplacer) Rest() string {
	return pr.rest
}

// Group returns a matching group from the last matched AdvanceRegexp.
func (pr *PrefixReplacer) Group(index int) string {
	return pr.m[index]
}

func (pr *PrefixReplacer) NextString(prefix string) string {
	if strings.HasPrefix(pr.rest, prefix) {
		pr.Skip(len(prefix))
		return prefix
	}
	return ""
}

func (pr *PrefixReplacer) SkipString(prefix string) bool {
	skipped := strings.HasPrefix(pr.rest, prefix)
	if skipped {
		pr.Skip(len(prefix))
	}
	return skipped
}

func (pr *PrefixReplacer) NextByte(b byte) bool {
	if pr.PeekByte() == int(b) {
		pr.Skip(1)
		return true
	}
	return false
}

func (pr *PrefixReplacer) NextHspace() string {
	i := initialHspace(pr.rest)
	if i != 0 {
		hspace := pr.rest[:i]
		pr.rest = pr.rest[i:]
		return hspace
	}
	return ""
}

func (pr *PrefixReplacer) AdvanceRegexp(re regex.Pattern) bool {
	pr.m = nil
	if !strings.HasPrefix(string(re), "^") {
		panic(fmt.Sprintf("PrefixReplacer.AdvanceRegexp: regular expression %q must have prefix %q.", re, "^"))
	}
	if Testing && pr.res.Matches("", re) {
		panic(fmt.Sprintf("PrefixReplacer.AdvanceRegexp: the empty string must not match the regular expression %q.", re))
	}
	if m := pr.res.Match(pr.rest, re); m != nil {
		pr.rest = pr.rest[len(m[0]):]
		pr.m = m
		return true
	}
	return false
}

func (pr *PrefixReplacer) NextRegexp(re regex.Pattern) []string {
	pr.m = nil
	if !strings.HasPrefix(string(re), "^") {
		panic(fmt.Sprintf("PrefixReplacer.AdvanceRegexp: regular expression %q must have prefix %q.", re, "^"))
	}
	if Testing && pr.res.Matches("", re) {
		panic(fmt.Sprintf("PrefixReplacer.AdvanceRegexp: the empty string must not match the regular expression %q.", re))
	}
	m := pr.res.Match(pr.rest, re)
	if m != nil {
		pr.rest = pr.rest[len(m[0]):]
	}
	return m
}

func (pr *PrefixReplacer) SkipRegexp(re regex.Pattern) bool {
	return pr.NextRegexp(re) != nil
}

// NextBytesFunc chops off the longest prefix (possibly empty) consisting
// solely of bytes for which fn returns true.
func (pr *PrefixReplacer) NextBytesFunc(fn func(b byte) bool) string {
	i := 0
	rest := pr.rest
	for i < len(rest) && fn(rest[i]) {
		i++
	}
	if i != 0 {
		pr.rest = rest[i:]
	}
	return rest[:i]
}

func (pr *PrefixReplacer) NextBytesSet(set *ByteSet) string {
	return pr.NextBytesFunc(set.Contains)
}

func (pr *PrefixReplacer) PeekByte() int {
	rest := pr.rest
	if rest == "" {
		return -1
	}
	return int(rest[0])
}

func (pr *PrefixReplacer) Mark() PrefixReplacerMark {
	return PrefixReplacerMark(pr.rest)
}

func (pr *PrefixReplacer) Reset(mark PrefixReplacerMark) {
	pr.rest = string(mark)
}

func (pr *PrefixReplacer) Skip(n int) {
	pr.rest = pr.rest[n:]
}

func (pr *PrefixReplacer) SkipHspace() {
	pr.rest = pr.rest[initialHspace(pr.rest):]
}

// Since returns the substring between the mark and the current position.
func (pr *PrefixReplacer) Since(mark PrefixReplacerMark) string {
	return string(mark[:len(mark)-len(pr.rest)])
}

func (pr *PrefixReplacer) AdvanceRest() string {
	rest := pr.rest
	pr.rest = ""
	return rest
}

func (pr *PrefixReplacer) HasPrefix(str string) bool {
	return strings.HasPrefix(pr.rest, str)
}

func (pr *PrefixReplacer) HasPrefixRegexp(re regex.Pattern) bool {
	return pr.res.Matches(pr.rest, re)
}

func initialHspace(s string) int {
	i := 0
	for i < len(s) && (s[i] == ' ' || s[i] == '\t') {
		i++
	}
	return i
}
