package intqa

import (
	"bytes"
	"fmt"
	"gopkg.in/check.v1"
	"io/ioutil"
	"testing"
)

type Suite struct {
	c       *check.C
	ck      *TestNameChecker
	summary string
}

func Test(t *testing.T) {
	check.Suite(&Suite{})
	check.TestingT(t)
}

func (s *Suite) Init(c *check.C) *TestNameChecker {
	errorf := func(format string, args ...interface{}) {
		s.summary = fmt.Sprintf(format, args...)
	}

	s.c = c
	s.ck = NewTestNameChecker(errorf)
	s.ck.ShowWarnings(true)
	s.ck.out = ioutil.Discard
	return s.ck
}

func (s *Suite) TearDownTest(c *check.C) {
	s.c = c
	s.CheckErrors(nil...)
	s.CheckWarnings(nil...)
	s.CheckSummary("")
}

func (s *Suite) CheckWarnings(warnings ...string) {
	s.c.Check(s.ck.warnings, check.DeepEquals, warnings)
	s.ck.warnings = nil
}

func (s *Suite) CheckErrors(errors ...string) {
	s.c.Check(s.ck.errors, check.DeepEquals, errors)
	s.ck.errors = nil
}

func (s *Suite) CheckSummary(summary string) {
	s.c.Check(s.summary, check.Equals, summary)
	s.summary = ""
}

func (s *Suite) Test_TestNameChecker_Check(c *check.C) {
	ck := s.Init(c)

	ck.Check()

	s.CheckWarnings(
		"W: Missing unit test \"Test_NewTestNameChecker\" for \"NewTestNameChecker\".",
		"W: Missing unit test \"Test_TestNameChecker_IgnoreFiles\" for \"TestNameChecker.IgnoreFiles\".",
		"W: Missing unit test \"Test_TestNameChecker_ShowWarnings\" for \"TestNameChecker.ShowWarnings\".",
		"W: Missing unit test \"Test_TestNameChecker_load\" for \"TestNameChecker.load\".",
		"W: Missing unit test \"Test_TestNameChecker_loadDecl\" for \"TestNameChecker.loadDecl\".",
		"W: Missing unit test \"Test_TestNameChecker_addCode\" for \"TestNameChecker.addCode\".",
		"W: Missing unit test \"Test_TestNameChecker_addTestee\" for \"TestNameChecker.addTestee\".",
		"W: Missing unit test \"Test_TestNameChecker_addTest\" for \"TestNameChecker.addTest\".",
		"W: Missing unit test \"Test_TestNameChecker_nextOrder\" for \"TestNameChecker.nextOrder\".",
		"W: Missing unit test \"Test_TestNameChecker_relate\" for \"TestNameChecker.relate\".",
		"W: Missing unit test \"Test_TestNameChecker_checkTests\" for \"TestNameChecker.checkTests\".",
		"W: Missing unit test \"Test_TestNameChecker_checkTestFile\" for \"TestNameChecker.checkTestFile\".",
		"W: Missing unit test \"Test_TestNameChecker_checkTestName\" for \"TestNameChecker.checkTestName\".",
		"W: Missing unit test \"Test_TestNameChecker_checkTestees\" for \"TestNameChecker.checkTestees\".",
		"W: Missing unit test \"Test_TestNameChecker_isIgnored\" for \"TestNameChecker.isIgnored\".",
		"W: Missing unit test \"Test_TestNameChecker_checkOrder\" for \"TestNameChecker.checkOrder\".",
		"W: Missing unit test \"Test_TestNameChecker_addError\" for \"TestNameChecker.addError\".",
		"W: Missing unit test \"Test_TestNameChecker_addWarning\" for \"TestNameChecker.addWarning\".",
		"W: Missing unit test \"Test_code_fullName\" for \"code.fullName\".",
		"W: Missing unit test \"Test_plural\" for \"plural\".",
		"W: Missing unit test \"Test_Test\" for \"Test\".",
		"W: Missing unit test \"Test_Suite_Init\" for \"Suite.Init\".",
		"W: Missing unit test \"Test_Suite_TearDownTest\" for \"Suite.TearDownTest\".",
		"W: Missing unit test \"Test_Suite_CheckWarnings\" for \"Suite.CheckWarnings\".",
		"W: Missing unit test \"Test_Suite_CheckErrors\" for \"Suite.CheckErrors\".",
		"W: Missing unit test \"Test_Suite_CheckSummary\" for \"Suite.CheckSummary\".",
		"W: Missing unit test \"Test_Value_Method\" for \"Value.Method\".")
	s.CheckSummary("27 warnings.")
}

func (s *Suite) Test_TestNameChecker_checkTestTestee__global(c *check.C) {
	ck := s.Init(c)

	ck.checkTestTestee(&test{
		code{"demo_test.go", "Suite", "Test__Global", 0},
		"",
		"",
		nil})

	s.CheckErrors(
		nil...)
}

func (s *Suite) Test_TestNameChecker_checkTestTestee__no_testee(c *check.C) {
	ck := s.Init(c)

	ck.checkTestTestee(&test{
		code{"demo_test.go", "Suite", "Test_Missing", 0},
		"Missing",
		"",
		nil})

	s.CheckErrors(
		"E: Missing testee \"Missing\" for test \"Suite.Test_Missing\".")
}

func (s *Suite) Test_TestNameChecker_checkTestTestee__testee_exists(c *check.C) {
	ck := s.Init(c)

	ck.checkTestTestee(&test{
		code{"demo_test.go", "Suite", "Test_Missing", 0},
		"Missing",
		"",
		&testee{}})

	s.CheckErrors(
		nil...)
}

func (s *Suite) Test_TestNameChecker_print__empty(c *check.C) {
	var out bytes.Buffer
	ck := s.Init(c)
	ck.out = &out

	ck.print()

	c.Check(out.String(), check.Equals, "")
}

func (s *Suite) Test_TestNameChecker_print__errors_and_warnings(c *check.C) {
	var out bytes.Buffer
	ck := s.Init(c)
	ck.out = &out

	ck.addError("1")
	ck.addWarning("2")
	ck.print()

	c.Check(out.String(), check.Equals, "E: 1\nW: 2\n")
	s.CheckErrors("E: 1")
	s.CheckWarnings("W: 2")
	s.CheckSummary("1 error and 1 warning.")
}

func (s *Suite) Test_isCamelCase(c *check.C) {
	_ = s.Init(c)

	c.Check(isCamelCase(""), check.Equals, false)
	c.Check(isCamelCase("Word"), check.Equals, false)
	c.Check(isCamelCase("Ada_Case"), check.Equals, false)
	c.Check(isCamelCase("snake_case"), check.Equals, false)
	c.Check(isCamelCase("CamelCase"), check.Equals, true)

	// After the first underscore of the description, any CamelCase
	// is ignored because there is no danger of confusing the method
	// name with the description.
	c.Check(isCamelCase("Word_CamelCase"), check.Equals, false)
}

func (s *Suite) Test_join(c *check.C) {
	_ = s.Init(c)

	c.Check(join("", " and ", ""), check.Equals, "")
	c.Check(join("one", " and ", ""), check.Equals, "one")
	c.Check(join("", " and ", "two"), check.Equals, "two")
	c.Check(join("one", " and ", "two"), check.Equals, "one and two")
}

type Value struct{}

// Method has no star on the receiver,
// for code coverage of TestNameChecker.loadDecl.
func (Value) Method() {}
