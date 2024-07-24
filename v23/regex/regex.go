// Package regex provides a registry of precompiled regular expressions
// to allow reusing them without the syntactic overhead of declaring
// pattern variables everywhere in the code.
// The registry is not thread-safe, but the precompiled patterns are.
package regex

import (
	"fmt"
	"regexp"
)

type Pattern string

type Registry struct {
	res map[Pattern]*regexp.Regexp
}

func NewRegistry() Registry {
	return Registry{make(map[Pattern]*regexp.Regexp)}
}

func (r *Registry) Compile(re Pattern) *regexp.Regexp {
	cre := r.res[re]
	if cre == nil {
		cre = regexp.MustCompile(string(re))
		r.res[re] = cre
	}
	return cre
}

// Consider defining an alternative CompileX method that implements the
// /x modifier to allow whitespace in the regular expression.
// This makes the regular expressions more readable.

func (r *Registry) Match(s string, re Pattern) []string {
	return r.Compile(re).FindStringSubmatch(s)
}

func (r *Registry) Matches(s string, re Pattern) bool {
	return r.Compile(re).MatchString(s)
}

func (r *Registry) Match1(s string, re Pattern) (matched bool, m1 string) {
	if m := r.matchn(s, re, 1); m != nil {
		return true, m[1]
	}
	return
}

func (r *Registry) Match2(s string, re Pattern) (matched bool, m1, m2 string) {
	if m := r.matchn(s, re, 2); m != nil {
		return true, m[1], m[2]
	}
	return
}

func (r *Registry) Match3(s string, re Pattern) (matched bool, m1, m2, m3 string) {
	if m := r.matchn(s, re, 3); m != nil {
		return true, m[1], m[2], m[3]
	}
	return
}

func (r *Registry) ReplaceFirst(s string, re Pattern, replacement string) ([]string, string) {
	if m := r.Compile(re).FindStringSubmatchIndex(s); m != nil {
		replaced := s[:m[0]] + replacement + s[m[1]:]
		mm := make([]string, len(m)/2)
		for i := 0; i < len(m); i += 2 {
			if m[i] < 0 {
				mm[i/2] = ""
			} else {
				mm[i/2] = s[m[i]:m[i+1]]
			}
		}
		return mm, replaced
	}
	return nil, s
}

func (r *Registry) matchn(s string, re Pattern, n int) []string {
	if m := r.Match(s, re); m != nil {
		if len(m) != 1+n {
			panic(fmt.Sprintf("expected match%d, got match%d for %q", len(m)-1, n, re))
		}
		return m
	}
	return nil
}
