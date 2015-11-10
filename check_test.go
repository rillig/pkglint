package main

// Implementation details for integrating gocheck

import (
	check "gopkg.in/check.v1"
	"testing"
)

func Test(t *testing.T) { check.TestingT(t) }

type Suite struct{}

func (s *Suite) SetUpTest(c *check.C) {
	G = &GlobalVars{}
}

func (s *Suite) TearDownTest(c *check.C) {
	G = nil
}

var _ = check.Suite(&Suite{})
var equals = check.Equals
