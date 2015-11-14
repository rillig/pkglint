package main

// Implementation details for integrating gocheck

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

func (s *Suite) SetUpTest(c *check.C) {
	G = new(GlobalVars)
	G.logOut, G.logErr = &s.stdout, &s.stderr
}

func (s *Suite) TearDownTest(c *check.C) {
	G = nil
	if out := s.Stdout(); out != "" {
		c.Errorf("Unchecked output on stdout: %q", out)
	}
	if err := s.Stderr(); err != "" {
		c.Errorf("Unchecked output on stderr: %q", err)
	}
}

var _ = check.Suite(new(Suite))

func Test(t *testing.T) { check.TestingT(t) }
