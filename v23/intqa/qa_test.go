package intqa

import (
	"bytes"
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"io"
	"os"
	"path"
	"reflect"
	"regexp"
	"strings"
	"testing"
)

type QATester struct {
	t       *testing.T
	ck      *QAChecker
	summary string
}

func NewQATester(tt *testing.T) *QATester {
	var s QATester
	errorf := func(format string, args ...any) {
		s.summary = fmt.Sprintf(format, args...)
	}

	s.t = tt
	s.ck = NewQAChecker(errorf)
	s.ck.out = io.Discard
	return &s
}

func (s *QATester) Done() {
	// The testees and tests are not validated to be checked
	// since that would create too much boilerplate code.
	s.CheckErrors(nil...)
	s.CheckSummary("")
}

func (s *QATester) CheckTestees(testees ...*testee) {
	if !reflect.DeepEqual(s.ck.testees, testees) {
		s.t.Errorf("want %#v, have %#v", testees, s.ck.testees)
	}
	s.ck.testees = nil
}

func (s *QATester) newTestee(filename, typeName, funcName string, order int) *testee {
	return &testee{code{filename, typeName, funcName, order}}
}

func (s *QATester) CheckTests(tests ...*test) {
	if !reflect.DeepEqual(s.ck.tests, tests) {
		s.t.Errorf("want %#v, have %#v", tests, s.ck.tests)
	}
	s.ck.tests = nil
}

func (*QATester) newTest(filename, typeName, funcName string, order int, testeeName, descr string, testee *testee) *test {
	c := code{filename, typeName, funcName, order}
	return &test{c, testeeName, descr, testee}
}

func (s *QATester) CheckErrors(errors ...string) {
	if !reflect.DeepEqual(s.ck.errors, errors) {
		s.t.Errorf("want %#v, have %#v", errors, s.ck.errors)
	}
	s.ck.errors = nil
}

func (s *QATester) CheckSummary(summary string) {
	if s.summary != summary {
		s.t.Errorf("have %#v, want %#v", s.summary, summary)
	}
	s.summary = ""
}

func Test_NewQAChecker(tt *testing.T) {
	s := NewQATester(tt)
	defer s.Done()
	ck := s.ck
	if !ck.isRelevant("*_test.go", "Suite", "SetUpTest", EAll) {
		tt.Error("want relevant")
	}
	if ck.isRelevant("*_test.go", "Suite", "SetUpTest", EMissingTest) {
		tt.Error("want not relevant")
	}

	if !ck.isRelevant("*_test.go", "Suite", "TearDownTest", EAll) {
		tt.Error("want relevant")
	}
	if ck.isRelevant("*_test.go", "Suite", "TearDownTest", EMissingTest) {
		tt.Error("want not relevant")
	}
}

func Test_QAChecker_Configure(tt *testing.T) {
	s := NewQATester(tt)
	defer s.Done()
	ck := s.ck

	ck.Configure("*", "*", "*", ENone) // overwrite initialization from Suite.Init

	if ck.isRelevant("", "", "", EAll) {
		tt.Error("want false")
	}
	if ck.isRelevant("", "", "", EMissingTestee) {
		tt.Error("want false")
	}
	if ck.isRelevant("", "", "", EMissingTest) {
		tt.Error("want false")
	}

	ck.Configure("*", "*", "*", EAll)

	if !ck.isRelevant("", "", "", EAll) {
		tt.Error("want true")
	}
	if !ck.isRelevant("", "", "", EMissingTestee) {
		tt.Error("want true")
	}
	if !ck.isRelevant("", "", "", EMissingTest) {
		tt.Error("want true")
	}

	ck.Configure("*", "*", "*", -EMissingTestee)

	if !ck.isRelevant("", "", "", EAll) {
		tt.Error("want true")
	}
	if ck.isRelevant("", "", "", EMissingTestee) {
		tt.Error("want false")
	}
	if !ck.isRelevant("", "", "", EMissingTest) {
		tt.Error("want true")
	}

	ck.Configure("*", "*", "*", ENone, EMissingTest)

	if !ck.isRelevant("", "", "", EAll) {
		tt.Error("want true")
	}
	if ck.isRelevant("", "", "", EMissingTestee) {
		tt.Error("want false")
	}
	if !ck.isRelevant("", "", "", EMissingTest) {
		tt.Error("want true")
	}

	ck.Configure("*", "*", "*", EAll, -EMissingTest)

	if !ck.isRelevant("", "", "", EAll) {
		tt.Error("want true")
	}
	if !ck.isRelevant("", "", "", EMissingTestee) {
		tt.Error("want true")
	}
	if ck.isRelevant("", "", "", EMissingTest) {
		tt.Error("want false")
	}
}

func Test_QAChecker_Configure__ignore_single_function(tt *testing.T) {
	s := NewQATester(tt)
	defer s.Done()
	ck := s.ck

	ck.Configure("*", "*", "*", EAll)

	// The intention of this rule is that this particular function is ignored.
	// Everything else from that file is still processed.
	ck.Configure("*_test.go", "", "TestMain", ENone)

	if !ck.isRelevant("file_test.go", "", "", EAll) {
		tt.Error("want true")
	}
	if !ck.isRelevant("file_test.go", "*", "*", EAll) {
		tt.Error("want true")
	}
	if !ck.isRelevant("file_test.go", "*", "Other", EAll) {
		tt.Error("want true")
	}
	if ck.isRelevant("file_test.go", "", "TestMain", EAll) {
		tt.Error("want false")
	}
	if !ck.isRelevant("file_test.go", "*", "TestMain", EAll) {
		tt.Error("want true")
	}
}

func Test_QAChecker_Check(tt *testing.T) {
	s := NewQATester(tt)
	defer s.Done()
	ck := s.ck

	ck.Configure("*", "*", "", -EMissingTest)
	ck.Configure("*", "Suite", "*", -EMissingTest)

	by, err := os.ReadFile("qa_test.go")
	if err != nil {
		tt.Fatal(err)
	}
	defer func() {
		err := os.WriteFile("qa_test.go", by, 0666)
		if err != nil {
			tt.Fatal(err)
		}
	}()

	ck.Check()

	s.CheckErrors(
		"Missing unit test \"Test_QAChecker_isRelevant\" for \"QAChecker.isRelevant\" in \"qa.go\". "+
			"Inserted it in \"qa_test.go\", above \"Test_QAChecker_errorsMask\".",
		"Missing unit test \"Test_skeleton\" for \"skeleton\" in \"qa.go\". "+
			"Inserted it in \"qa_test.go\", above \"Test_code_fullName\".",
		"Missing unit test \"Test_NewQATester\" for \"NewQATester\" in \"qa_test.go\". "+
			"Inserted it in \"qa_test.go\", above \"Test_Value_Method\".",
		"Missing unit test \"Test_QATester_Done\" for \"QATester.Done\" in \"qa_test.go\". "+
			"Inserted it in \"qa_test.go\", above \"Test_Value_Method\".",
		"Missing unit test \"Test_QATester_CheckTestees\" for \"QATester.CheckTestees\" in \"qa_test.go\". "+
			"Inserted it in \"qa_test.go\", above \"Test_Value_Method\".",
		"Missing unit test \"Test_QATester_newTestee\" for \"QATester.newTestee\" in \"qa_test.go\". "+
			"Inserted it in \"qa_test.go\", above \"Test_Value_Method\".",
		"Missing unit test \"Test_QATester_CheckTests\" for \"QATester.CheckTests\" in \"qa_test.go\". "+
			"Inserted it in \"qa_test.go\", above \"Test_Value_Method\".",
		"Missing unit test \"Test_QATester_newTest\" for \"QATester.newTest\" in \"qa_test.go\". "+
			"Inserted it in \"qa_test.go\", above \"Test_Value_Method\".",
		"Missing unit test \"Test_QATester_CheckErrors\" for \"QATester.CheckErrors\" in \"qa_test.go\". "+
			"Inserted it in \"qa_test.go\", above \"Test_Value_Method\".",
		"Missing unit test \"Test_QATester_CheckSummary\" for \"QATester.CheckSummary\" in \"qa_test.go\". "+
			"Inserted it in \"qa_test.go\", above \"Test_Value_Method\".",
	)
	s.CheckSummary("10 errors.")
}

func Test_QAChecker_load__filtered_nothing(tt *testing.T) {
	s := NewQATester(tt)
	defer s.Done()
	ck := s.ck

	ck.Configure("*", "*", "*", ENone)

	ck.load(".")

	s.CheckTestees(
		nil...)
	s.CheckTests(
		nil...)
}

func Test_QAChecker_load__filtered_only_Value(tt *testing.T) {
	s := NewQATester(tt)
	defer s.Done()
	ck := s.ck

	ck.Configure("*", "*", "*", ENone)
	ck.Configure("*", "Value", "*", EAll)

	ck.load(".")

	s.CheckTestees(
		s.newTestee("qa_test.go", "Value", "", 0),
		s.newTestee("qa_test.go", "Value", "Method", 1))
	s.CheckTests(
		nil...)
}

func Test_QAChecker_load__panic(tt *testing.T) {
	s := NewQATester(tt)
	defer s.Done()
	ck := s.ck

	var p any
	func() {
		defer func() {
			p = recover()
		}()
		ck.load("does-not-exist")
	}()
	if have, want := p.(error).Error(), `^open does-not-exist\b.*`; !regexp.MustCompile(want).MatchString(have) {
		tt.Errorf("want %#v, have %#v", want, have)
	}
}

func Test_QAChecker_loadDecl(tt *testing.T) {
	s := NewQATester(tt)
	defer s.Done()
	ck := s.ck

	load := func(filename, decl string) {
		fset := token.NewFileSet()
		file, err := parser.ParseFile(fset, filename, "package p; "+decl, 0)
		if err != nil {
			tt.Fatal(err)
		}
		ck.loadDecl(file.Decls[0], filename)
	}

	load("file_test.go", "type TypeName int")

	s.CheckTestees(
		s.newTestee("file_test.go", "TypeName", "", 0))

	// The freestanding TestMain function is ignored by a hardcoded rule,
	// independently of the configuration.
	load("file_test.go", "func TestMain() {}")

	s.CheckTestees(
		nil...)
	s.CheckTests(
		nil...)

	// The TestMain method on a type is relevant, but violates the naming rule.
	// Therefore, it is ignored.
	load("file_test.go", "func (Suite) TestMain(*check.C) {}")

	s.CheckTestees(
		nil...)
	s.CheckTests(
		nil...)
	s.CheckErrors(
		"Test \"Suite.TestMain\" must start with \"Test_\".")

	// The above error can be disabled, and then the method is handled
	// like any other test method.
	ck.Configure("*", "Suite", "*", -EName)
	load("file_test.go", "func (Suite) TestMain(*check.C) {}")

	s.CheckTestees(
		nil...)
	s.CheckTests(
		s.newTest("file_test.go", "Suite", "TestMain", 1, "Main", "", nil))

	// There is no special handling for TestMain method with a description.
	load("file_test.go", "func (Suite) TestMain__descr(*check.C) {}")

	s.CheckTestees(
		nil...)
	s.CheckTests(
		s.newTest("file_test.go", "Suite", "TestMain__descr", 2, "Main", "descr", nil))
	s.CheckErrors(
		nil...)
}

func Test_QAChecker_parseFuncDecl(tt *testing.T) {

	testFunc := func(filename, decl, typeName, funcName string) {
		fset := token.NewFileSet()
		file, err := parser.ParseFile(fset, filename, "package p; "+decl, 0)
		if err != nil {
			tt.Fatal(err)
		}
		fn := file.Decls[0].(*ast.FuncDecl)
		actual := (*QAChecker).parseFuncDecl(nil, filename, fn)
		if have, want := actual, (code{filename, typeName, funcName, 0}); have != want {
			tt.Errorf("want %#v, have %#v", want, have)
		}
	}

	testFunc("f_test.go", "func (t Type) Test() {}",
		"Type", "Test")
	testFunc("f_test.go", "func (t Type) Test_Type_Method() {}",
		"Type", "Test_Type_Method")
	testFunc("f_test.go", "func Test() {}",
		"", "Test")
	testFunc("f_test.go", "func Test_Type_Method() {}",
		"", "Test_Type_Method")
}

func Test_QAChecker_addCode(tt *testing.T) {
	s := NewQATester(tt)
	defer s.Done()
	ck := s.ck

	// TestMain is a special function that does not test the Main function.
	ck.addCode(code{"main_test.go", "", "TestMain", 123}, &ast.FuncDecl{})

	if len(ck.tests) > 0 {
		tt.Errorf("%#v", ck.tests)
	}
}

func Test_QAChecker_isTest(tt *testing.T) {
	test := func(filename, typeName, funcName string, isTest bool) {
		code := code{filename, typeName, funcName, 0}
		if have := (*QAChecker).isTest(nil, code, nil); have != isTest {
			tt.Errorf("want %#v, have %#v", isTest, have)
		}
	}

	testFunc := func(filename, decl string, isTest bool) {
		fset := token.NewFileSet()
		file, err := parser.ParseFile(fset, filename, "package p; "+decl, 0)
		if err != nil {
			tt.Fatal(err)
		}
		fn := file.Decls[0].(*ast.FuncDecl)
		code := (*QAChecker).parseFuncDecl(nil, filename, fn)
		if have := (*QAChecker).isTest(nil, code, fn); have != isTest {
			tt.Errorf("want %#v, have %#v", isTest, have)
		}
	}

	test("f.go", "Type", "", false)
	test("f.go", "", "Func", false)
	test("f.go", "Type", "Method", false)
	test("f.go", "Type", "Test", false)
	test("f.go", "Type", "Test_Type_Method", false)
	test("f.go", "", "Test_Type_Method", false)

	testFunc("f_test.go", "func (t Type) Test(tt *testing.T) {}", true)
	testFunc("f_test.go", "func (t Type) Test_Type_Method(tt *testing.T) {}", true)
	testFunc("f_test.go", "func Test_Type_Method(tt *testing.T) {}", true)
	testFunc("f_test.go", "func Test_Type_Method(tt *T) {}", true)
	testFunc("f_test.go", "func Test_Type_Method(tt T) {}", true)
	testFunc("f_test.go", "func Test_Type_Method(t *testing.T) {}", true)
	testFunc("f_test.go", "func Test_Type_Method(X) {}", false)
	testFunc("f_test.go", "func Test_Type_Method(int) {}", false)

	// Having such a method in a test suite is unlikely since the testing
	// package checks the number of parameters.
	testFunc("f_test.go", "func Test_Type_Method() {}", false)
	testFunc("f_test.go", "func Test(int, int) {}", false)

	// In a test helper type, it is reasonable that some of the method
	// names start with "Test". This doesn't make them tests though.
	testFunc("f_test.go", "func (h *Helper) TestEqInt(int, int) {}", false)
}

func Test_QAChecker_addTestee(tt *testing.T) {
	s := NewQATester(tt)
	defer s.Done()
	ck := s.ck

	ck.addTestee(code{"filename.go", "Type", "Method", 0})

	s.CheckTestees(
		s.newTestee("filename.go", "Type", "Method", 0))
}

func Test_QAChecker_addTest(tt *testing.T) {
	s := NewQATester(tt)
	defer s.Done()
	ck := s.ck

	ck.addTest(code{"filename.go", "Type", "Method", 0})

	s.CheckTests(
		nil...)
	s.CheckErrors(
		"Test \"Type.Method\" must start with \"Test_\".")
}

func Test_QAChecker_addTest__empty_description(tt *testing.T) {
	s := NewQATester(tt)
	defer s.Done()
	ck := s.ck

	ck.addTest(code{"f_test.go", "Suite", "Test_Method__", 0})

	s.CheckErrors(
		"Test \"Suite.Test_Method__\" must have a nonempty description.")

	// The test is not registered and thus cannot complain about its missing
	// testee.
	ck.checkTests()

	s.CheckErrors(
		nil...)
}

func Test_QAChecker_addTest__suppressed_empty_description(tt *testing.T) {
	s := NewQATester(tt)
	defer s.Done()
	ck := s.ck

	ck.Configure("*", "*", "*", -EName)
	ck.addTest(code{"f_test.go", "Suite", "Test_Method__", 0})

	s.CheckErrors(
		nil...)

	// Since there was no error above, the test is added normally
	// and can complain about its missing testee.
	ck.checkTests()

	s.CheckErrors(
		"Missing testee \"Method\" for test \"Suite.Test_Method__\".")
}

func Test_QAChecker_nextOrder(tt *testing.T) {
	s := NewQATester(tt)
	defer s.Done()
	ck := s.ck

	for i := 0; i < 5; i++ {
		if have := ck.nextOrder(); have != i {
			tt.Errorf("want %#v, have %#v", i, have)
		}
	}
}

func Test_QAChecker_relate(tt *testing.T) {
	ck := NewQAChecker(tt.Errorf)
	ck.addTestee(code{"main.go", "", "abs", 0})
	ck.addTest(code{"main_test.go", "", "Test_abs", 2})

	ck.relate()

	if ck.tests[0].testee == nil {
		tt.Errorf("not related")
	}
}

func Test_QAChecker_checkTests(tt *testing.T) {
	s := NewQATester(tt)
	defer s.Done()
	ck := s.ck

	ck.tests = append(ck.tests,
		s.newTest("wrong_test.go", "", "Test_Func", 0, "Func", "",
			s.newTestee("source.go", "", "Func", 1)))

	ck.checkTests()

	s.CheckErrors(
		"Test \"Test_Func\" for \"Func\" " +
			"must be in source_test.go instead of wrong_test.go.")
}

func Test_QAChecker_checkTestFile__global(tt *testing.T) {
	s := NewQATester(tt)
	defer s.Done()
	ck := s.ck

	ck.checkTestFile(&test{
		code{"demo_test.go", "Suite", "Test__Global", 0},
		"",
		"",
		&testee{code{"other.go", "", "Global", 0}}})

	s.CheckErrors(
		"Test \"Suite.Test__Global\" for \"Global\" " +
			"must be in other_test.go instead of demo_test.go.")
}

func Test_QAChecker_checkTestTestee__global(tt *testing.T) {
	s := NewQATester(tt)
	defer s.Done()
	ck := s.ck

	ck.checkTestTestee(&test{
		code{"demo_test.go", "Suite", "Test__Global", 0},
		"",
		"",
		nil})

	s.CheckErrors(
		nil...)
}

func Test_QAChecker_checkTestTestee__no_testee(tt *testing.T) {
	s := NewQATester(tt)
	defer s.Done()
	ck := s.ck

	ck.checkTestTestee(&test{
		code{"demo_test.go", "Suite", "Test_Missing", 0},
		"Missing",
		"",
		nil})

	s.CheckErrors(
		"Missing testee \"Missing\" for test \"Suite.Test_Missing\".")
}

func Test_QAChecker_checkTestTestee__testee_exists(tt *testing.T) {
	s := NewQATester(tt)
	defer s.Done()
	ck := s.ck

	ck.checkTestTestee(&test{
		code{"demo_test.go", "Suite", "Test_Missing", 0},
		"Missing",
		"",
		&testee{}})

	s.CheckErrors(
		nil...)
}

func Test_QAChecker_checkTestDescr__camel_case(tt *testing.T) {
	s := NewQATester(tt)
	defer s.Done()
	ck := s.ck

	ck.checkTestDescr(&test{
		code{"demo_test.go", "Suite", "Test_Missing__CamelCase", 0},
		"Missing",
		"CamelCase",
		&testee{}})

	s.CheckErrors(
		"Suite.Test_Missing__CamelCase: Test description \"CamelCase\" " +
			"must not use CamelCase in the first word.")
}

func Test_QAChecker_checkTesteesTest(tt *testing.T) {
	s := NewQATester(tt)
	defer s.Done()
	ck := s.ck

	ck.addTestee(code{"demo.go", "Type", "", 0})
	ck.addTestee(code{"demo.go", "", "Func", 0})
	ck.addTestee(code{"demo.go", "Type", "Method", 0})
	ck.addTestee(code{"demo.go", "OkType", "", 0})
	ck.addTestee(code{"demo.go", "", "OkFunc", 0})
	ck.addTestee(code{"demo.go", "OkType", "Method", 0})
	ck.addTestee(code{"demo.go", "Bottom", "Method", 0})
	ck.addTest(code{"demo_test.go", "", "Test_OkType", 0})
	ck.addTest(code{"demo_test.go", "", "Test_OkFunc", 0})
	ck.addTest(code{"demo_test.go", "", "Test_OkType_Method", 0})
	ck.relate()

	_ = os.WriteFile("demo_test.go", []byte(""), 0666)
	defer func() {
		_ = os.Remove("demo_test.go")
	}()

	ck.checkTesteesTest()

	s.CheckErrors(
		"Missing unit test \"Test_Type\" for \"Type\" in \"demo.go\". "+
			"Insert it in \"demo_test.go\", above \"Test_OkType\".",
		"Missing unit test \"Test_Func\" for \"Func\" in \"demo.go\". "+
			"Insert it in \"demo_test.go\", above \"Test_OkType\".",
		"Missing unit test \"Test_Type_Method\" for \"Type.Method\" in \"demo.go\". "+
			"Insert it in \"demo_test.go\", above \"Test_OkType\".",
		"Missing unit test \"Test_Bottom_Method\" for \"Bottom.Method\" in \"demo.go\". "+
			"Inserted it at the bottom of \"demo_test.go\".")
}

func Test_QAChecker_checkMethodsSameFile(tt *testing.T) {
	s := NewQATester(tt)
	defer s.Done()
	ck := s.ck

	ck.addTestee(code{"main.go", "Main", "", 0})
	ck.addTestee(code{"main.go", "Main", "MethodOk", 1})
	ck.addTestee(code{"other.go", "Main", "MethodWrong", 2})
	ck.addTestee(code{"main_test.go", "Main", "MethodOkTest", 3})
	ck.addTestee(code{"other_test.go", "Main", "MethodWrongTest", 4})
	ck.addTestee(code{"other_test.go", "Elsewhere", "Func", 4})
	ck.addTestee(code{"main_test.go", "T", "", 100})
	ck.addTestee(code{"main_test.go", "T", "MethodOk", 101})
	ck.addTestee(code{"other_test.go", "T", "MethodWrong", 102})
	ck.addTest(code{"main_test.go", "T", "Test_MethodOk", 101})
	ck.addTest(code{"other_test.go", "T", "Test_MethodWrong", 102})

	ck.checkMethodsSameFile()

	s.CheckErrors(
		"Method Main.MethodWrong must be in main.go, like its type.",
		"Method Main.MethodWrongTest must be in main_test.go, "+
			"corresponding to its type.",
		"Method T.MethodWrong must be in main_test.go, like its type.",
		"Method T.Test_MethodWrong must be in main_test.go, like its type.")
}

func Test_QAChecker_errorsMask(tt *testing.T) {
	s := NewQATester(tt)
	defer s.Done()
	ck := s.ck

	if have, want := ck.errorsMask(0, EAll), ^uint64(0); have != want {
		tt.Errorf("want %#v, have %#v", want, have)
	}
	if have, want := ck.errorsMask(12345, ENone), uint64(0); have != want {
		tt.Errorf("want %#v, have %#v", want, have)
	}
	if have, want := ck.errorsMask(12345, ENone, EMissingTest), uint64(8); have != want {
		tt.Errorf("want %#v, have %#v", want, have)
	}
	if have, want := ck.errorsMask(2, EMissingTest), uint64(10); have != want {
		tt.Errorf("want %#v, have %#v", want, have)
	}
}

func Test_QAChecker_checkOrder(tt *testing.T) {
	s := NewQATester(tt)
	defer s.Done()
	ck := s.ck

	ck.addTestee(code{"f.go", "T", "", 10})
	ck.addTestee(code{"f.go", "T", "M1", 11})
	ck.addTestee(code{"f.go", "T", "M2", 12})
	ck.addTestee(code{"f.go", "T", "M3", 13})
	ck.addTest(code{"a_test.go", "S", "Test_A", 98})        // different file, is skipped
	ck.addTest(code{"f_test.go", "S", "Test_Missing", 99})  // missing testee, is skipped
	ck.addTest(code{"f_test.go", "S", "Test_T_M1", 100})    // maxTestee = 11
	ck.addTest(code{"f_test.go", "S", "Test_T_M2", 101})    // maxTestee = 12
	ck.addTest(code{"f_test.go", "S", "Test_T", 102})       // testee 10 < maxTestee 12: insert before first [.testee > testee 10] == T_M1
	ck.addTest(code{"f_test.go", "S", "Test_T_M3", 103})    // maxTestee = 13
	ck.addTest(code{"f_test.go", "S", "Test_T__1", 104})    // testee < maxTestee: insert before first [testee > 10]
	ck.addTest(code{"f_test.go", "S", "Test_T__2", 105})    // testee < maxTestee: insert before first [testee > 10]
	ck.addTest(code{"f_test.go", "S", "Test_T_M2__1", 106}) // testee < maxTestee: insert before first [testee > 12] == T_M3
	ck.relate()

	ck.checkOrder()

	s.CheckErrors(
		"Test \"S.Test_T\" must be ordered before \"S.Test_T_M1\".",
		"Test \"S.Test_T__1\" must be ordered before \"S.Test_T_M1\".",
		"Test \"S.Test_T__2\" must be ordered before \"S.Test_T_M1\".",
		"Test \"S.Test_T_M2__1\" must be ordered before \"S.Test_T_M3\".")
}

func Test_QAChecker_addError(tt *testing.T) {
	s := NewQATester(tt)
	defer s.Done()
	ck := s.ck

	ck.Configure("ignored*", "*", "*", -EName)
	ok1 := ck.addError(EName, code{"ignored.go", "", "Func", 0}, "E1")
	ok2 := ck.addError(EName, code{"reported.go", "", "Func", 0}, "E2")

	if ok1 {
		tt.Error("want false")
	}
	if !ok2 {
		tt.Error("want true")
	}
	s.CheckErrors(
		"E2")
}

func Test_QAChecker_print__empty(tt *testing.T) {
	var out bytes.Buffer
	s := NewQATester(tt)
	defer s.Done()
	ck := s.ck
	ck.out = &out

	ck.print()

	if have, want := out.String(), ""; have != want {
		tt.Errorf("want %#v, have %#v", want, have)
	}
}

func Test_QAChecker_print__1_error(tt *testing.T) {
	var out bytes.Buffer
	s := NewQATester(tt)
	defer s.Done()
	ck := s.ck
	ck.out = &out
	ck.addError(EName, code{}, "1")

	ck.print()

	if have, want := out.String(), "1\n"; have != want {
		tt.Errorf("want %#v, have %#v", want, have)
	}
	s.CheckErrors("1")
	s.CheckSummary("1 error.")
}

func Test_QAChecker_print__2_errors(tt *testing.T) {
	var out bytes.Buffer
	s := NewQATester(tt)
	defer s.Done()
	ck := s.ck
	ck.out = &out
	ck.addError(EName, code{}, "1")
	ck.addError(EName, code{}, "2")

	ck.print()

	if have, want := out.String(), "1\n2\n"; have != want {
		tt.Errorf("want %#v, have %#v", want, have)
	}
	s.CheckErrors("1", "2")
	s.CheckSummary("2 errors.")
}

func Test_QAChecker_insertionSuggestion(tt *testing.T) {
	s := NewQATester(tt)
	defer s.Done()
	ck := s.ck

	ck.addTestee(code{"file1.go", "Type1", "", 1})
	ck.addTestee(code{"file2.go", "", "Func4", 2})
	ck.addTestee(code{"file2.go", "", "Func5", 3})
	ck.addTestee(code{"file2.go", "", "Func6", 4})
	ck.addTestee(code{"file3.go", "Type3", "Method", 5})
	ck.addTest(code{"file2_test.go", "", "Test_Func5", 6})
	ck.relate()

	_ = os.WriteFile("file1_test.go", nil, 0666)
	_ = os.WriteFile("file2_test.go", nil, 0666)
	_ = os.WriteFile("file3_test.go", nil, 0666)
	defer func() {
		_ = os.Remove("file3_test.go")
		_ = os.Remove("file2_test.go")
		_ = os.Remove("file1_test.go")
	}()

	testeeFor := func(name string) *testee {
		for _, testee := range ck.testees {
			if testee.fullName() == name {
				return testee
			}
		}
		panic(name)
	}

	test := func(testFile, testName, msg string) {
		testTarget := strings.SplitN(testName, "_", 2)[1]
		testeeName := strings.Replace(testTarget, "_", ".", -1)
		testee := testeeFor(testeeName)
		actual := ck.insertionSuggestion(
			&test{code{testFile, "", testName, 0}, testName, "", testee})
		if have, want := actual, msg; have != want {
			tt.Errorf("want %#v, have %#v", want, have)
		}
	}

	test(
		"file1_test.go", "Test_Type1",
		"Inserted it at the bottom of \"file1_test.go\".")

	test(
		"file2_test.go", "Test_Func4",
		"Insert it in \"file2_test.go\", above \"Test_Func5\".")

	test(
		"file2_test.go", "Test_Func5",
		"Insert it in \"file2_test.go\", above \"Test_Func5\".")

	test(
		"file2_test.go", "Test_Func6",
		"Inserted it at the bottom of \"file2_test.go\".")
}

func Test_code_fullName(tt *testing.T) {

	test := func(typeName, funcName, fullName string) {
		code := code{"filename", typeName, funcName, 0}
		if have, want := code.fullName(), fullName; have != want {
			tt.Errorf("want %#v, have %#v", want, have)
		}
	}

	test("Type", "", "Type")
	test("", "Func", "Func")
	test("Type", "Method", "Type.Method")
}

func Test_code_isFunc(tt *testing.T) {

	test := func(typeName, funcName string, isFunc bool) {
		code := code{"filename", typeName, funcName, 0}
		if have, want := code.isFunc(), isFunc; have != want {
			tt.Errorf("want %#v, have %#v", want, have)
		}
	}

	test("Type", "", false)
	test("", "Func", true)
	test("Type", "Method", false)
}

func Test_code_isType(tt *testing.T) {

	test := func(typeName, funcName string, isType bool) {
		code := code{"filename", typeName, funcName, 0}
		if have, want := code.isType(), isType; have != want {
			tt.Errorf("want %#v, have %#v", want, have)
		}
	}

	test("Type", "", true)
	test("", "Func", false)
	test("Type", "Method", false)
}

func Test_code_isMethod(tt *testing.T) {

	test := func(typeName, funcName string, isMethod bool) {
		code := code{"filename", typeName, funcName, 0}
		if have, want := code.isMethod(), isMethod; have != want {
			tt.Errorf("want %#v, have %#v", want, have)
		}
	}

	test("Type", "", false)
	test("", "Func", false)
	test("Type", "Method", true)
}

func Test_code_isTestScope(tt *testing.T) {

	test := func(filename string, isTestScope bool) {
		code := code{filename, "", "", 0}
		if have, want := code.isTestScope(), isTestScope; have != want {
			tt.Errorf("want %#v, have %#v", want, have)
		}
	}

	test("f.go", false)
	test("test.go", false)
	test("_test.go", true)
	test("file_test.go", true)
	test("file_linux_test.go", true)
}

func Test_code_testFile(tt *testing.T) {

	test := func(filename string, testFile string) {
		code := code{filename, "", "", 0}
		if have, want := code.testFile(), testFile; have != want {
			tt.Errorf("want %#v, have %#v", want, have)
		}
	}

	test("f.go", "f_test.go")
	test("test.go", "test_test.go")
	test("_test.go", "_test.go")
	test("file_test.go", "file_test.go")
	test("file_linux_test.go", "file_linux_test.go")
}

func Test_isCamelCase(tt *testing.T) {
	tests := []struct {
		s    string
		want bool
	}{
		{"", false},
		{"Word", false},
		{"Ada_Case", false},
		{"snake_case", false},
		{"CamelCase", true},
		{"lowerCamelCase", true},
		// After the first underscore of the description, any CamelCase
		// is ignored because there is no danger of confusing the method
		// name with the description.
		{"Word_CamelCase", false},
	}
	for _, test := range tests {
		tt.Run(test.s, func(t *testing.T) {
			have := isCamelCase(test.s)
			if have != test.want {
				tt.Errorf("want %#v, have %#v for %#v", test.want, have, test.s)
			}
		})
	}
}

func Test_join(t *testing.T) {
	tests := []struct {
		name string
		a    string
		sep  string
		b    string
		want string
	}{
		{"none", "", " and ", "", ""},
		{"first", "one", " and ", "", "one"},
		{"second", "", " and ", "two", "two"},
		{"both", "one", " and ", "two", "one and two"},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			if have := join(test.a, test.sep, test.b); have != test.want {
				t.Errorf("want %#v, have %#v", test.want, have)
			}
		})
	}
}

func Test_matches(tt *testing.T) {
	tests := []struct {
		s       string
		pattern string
		want    bool
	}{
		{"*", "*", true},
		{"anything", "*", true},
		{"*", "anything", false},
	}
	for _, test := range tests {
		tt.Run(test.s+" "+test.pattern, func(t *testing.T) {
			if have := matches(test.s, test.pattern); have != test.want {
				t.Errorf("want %#v, have %#v", test.want, have)
			}
		})
	}
}

func Test_matches__invalid_pattern(tt *testing.T) {
	var p any
	func() {
		defer func() { p = recover() }()
		matches("any", "[")
	}()
	if p != path.ErrBadPattern {
		tt.Error(p)
	}
}

func Test_sortedKeys(tt *testing.T) {

	m := make(map[string]uint8)
	m["first"] = 1
	m["second"] = 2
	m["third"] = 3
	m["fourth"] = 4

	have := sortedKeys(m)
	want := []string{"first", "fourth", "second", "third"}
	if !reflect.DeepEqual(have, want) {
		tt.Errorf("want %#v, have %#v", want, have)
	}
}

func Test_Value_Method(tt *testing.T) {
	// To make it appear actually used to golangci-lint.
	_ = Value{}

	// Just for code coverage of checkTestFile, to have a piece of code
	// that lives in the same file as its test.
}

type Value struct{}

// Method has no star on the receiver,
// for code coverage of QAChecker.loadDecl.
func (Value) Method() {}

func Test__qa(t *testing.T) {
	ck := NewQAChecker(t.Errorf)
	ck.Configure("*", "*", "", -EMissingTest)
	ck.Configure("*", "QAChecker", "isRelevant", -EMissingTest) // TODO
	ck.Configure("*", "QATester", "*", -EMissingTest)           // TODO
	ck.Configure("*", "", "NewQATester", -EMissingTest)         // TODO
	ck.Configure("*", "", "skeleton", -EMissingTest)            // TODO

	ck.Check()
}
