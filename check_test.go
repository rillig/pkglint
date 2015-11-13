package main

// Implementation details for integrating gocheck

import (
	"bytes"
	check "gopkg.in/check.v1"
	"testing"
)

var equals = check.Equals
var deepEquals = check.DeepEquals

type Suite struct{}

func (s *Suite) SetUpTest(c *check.C) {
	G = &GlobalVars{}
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
	return s.stdout.String()
}

func (s *CaptureOutputSuite) Stderr() string {
	return s.stderr.String()
}

func (s *CaptureOutputSuite) SetUpTest(c *check.C) {
	G = &GlobalVars{}
	G.logOut = &s.stdout
	G.logErr = &s.stderr
}

func (s *CaptureOutputSuite) TearDownTest(c *check.C) {
	G = nil
	s.stdout.Reset()
	s.stderr.Reset()
}

var _ = check.Suite(&CaptureOutputSuite{})

func Test(t *testing.T) { check.TestingT(t) }
