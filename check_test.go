package main

import (
	"bytes"
	"testing"

	check "gopkg.in/check.v1"
)

var equals = check.Equals
var deepEquals = check.DeepEquals

type Suite struct {
	stdout bytes.Buffer
	stderr bytes.Buffer
}

func (s *Suite) Stdout() string {
	defer s.stdout.Reset()
	return s.stdout.String()
}

func (s *Suite) Stderr() string {
	defer s.stderr.Reset()
	return s.stderr.String()
}

func (s *Suite) Output() string {
	return s.Stdout() + s.Stderr()
}

func (s *Suite) NewLines(fname string, lines ...string) []*Line {
	result := make([]*Line, len(lines))
	for i, line := range lines {
		result[i] = NewLine(fname, sprintf("%d", i+1), line, []*RawLine{{i + 1, line + "\n"}})
	}
	return result
}

func (s *Suite) UseCommandLine(args ...string) {
	G.opts = new(Pkglint).ParseCommandLine(append([]string{"pkglint"}, args...), G.logOut)
}

func (s*Suite) ExpectFatalError(action func()) {
	if r := recover(); r != nil {
		if _, ok := r.(pkglintFatal); ok {
			action()
			return
		}
		panic(r)
	}
}

func (s *Suite) SetUpTest(c *check.C) {
	G = new(GlobalVars)
	G.logOut, G.logErr, G.traceOut = &s.stdout, &s.stderr, &s.stdout
}

func (s *Suite) TearDownTest(c *check.C) {
	G = nil
	if out := s.Output(); out != "" {
		c.Errorf("Unchecked output; check with: c.Check(s.Output(), equals, %q)", out)
	}
}

var _ = check.Suite(new(Suite))

func Test(t *testing.T) { check.TestingT(t) }
