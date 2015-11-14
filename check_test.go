package main

// Implementation details for integrating gocheck

import (
	"bytes"
	check "gopkg.in/check.v1"
	"os"
	"testing"
)

var equals = check.Equals
var deepEquals = check.DeepEquals

type Suite struct{}

func (s *Suite) SetUpTest(c *check.C) {
	G = &GlobalVars{}
	G.logOut, G.logErr = os.Stdout, os.Stderr
}

func (s *Suite) TearDownTest(c *check.C) {
	G = nil
}

var _ = check.Suite(&Suite{})

type CaptureOutputSuite struct {
	stdout bytes.Buffer
	stderr bytes.Buffer
}

func (s *CaptureOutputSuite) Stdout() string {
	defer s.stdout.Reset()
	return s.stdout.String()
}

func (s *CaptureOutputSuite) Stderr() string {
	defer s.stderr.Reset()
	return s.stderr.String()
}

func (s *CaptureOutputSuite) Output() string {
	return s.Stdout() + s.Stderr()
}

func (s *CaptureOutputSuite) SetUpTest(c *check.C) {
	G = &GlobalVars{}
	G.logOut, G.logErr = &s.stdout, &s.stderr
}

func (s *CaptureOutputSuite) TearDownTest(c *check.C) {
	G = nil
	if out := s.Stdout(); out != "" {
		c.Errorf("Unchecked output on stdout: %q", out)
	}
	if err := s.Stderr(); err != "" {
		c.Errorf("Unchecked output on stderr: %q", err)
	}
}

var _ = check.Suite(&CaptureOutputSuite{})

func Test(t *testing.T) { check.TestingT(t) }
