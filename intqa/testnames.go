// Package intqa provides quality assurance for the pkglint code.
package intqa

import (
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"io"
	"os"
	"path"
	"sort"
	"strings"
	"unicode"
)

type Error int

const (
	ENone Error = iota + 1
	EAll

	// A function or method does not have a corresponding test.
	EMissingTest

	// The name of a test function does not correspond to a program
	// element to be tested.
	EMissingTestee

	// The tests are not in the same order as their corresponding
	// testees in the main code.
	EOrder

	// The test method does not have a valid name.
	EName

	// The file of the test method does not correspond to the
	// file of the testee.
	EFile
)

// TestNameChecker ensures that all test names follow a common naming scheme:
//  Test_${Type}_${Method}__${description_using_underscores}
// Each of the variable parts may be omitted.
type TestNameChecker struct {
	errorf func(format string, args ...interface{})

	filters []filter
	order   int

	testees []*testee
	tests   []*test

	errors []string
	out    io.Writer
}

// NewTestNameChecker creates a new checker.
// By default, all errors are enabled;
// call Configure to disable them selectively.
func NewTestNameChecker(errorf func(format string, args ...interface{})) *TestNameChecker {
	ck := TestNameChecker{errorf: errorf, out: os.Stderr}

	// This function is created by https://github.com/rillig/gobco when
	// measuring the branch coverage of a package.
	ck.Configure("*_test.go", "", "TestMain", ENone)

	return &ck
}

// Configure sets the errors that are activated for the given code,
// specified by shell patterns like in path.Match.
//
// All rules are applied in order. Later rules overwrite earlier rules.
//
// Individual errors can be enabled by giving their constant and disabled
// by negating them, such as -EMissingTestee. To reset everything, use
// either EAll or ENone.
func (ck *TestNameChecker) Configure(filenames, typeNames, funcNames string, errors ...Error) {
	ck.filters = append(ck.filters, filter{filenames, typeNames, funcNames, errors})
}

func (ck *TestNameChecker) Check() {
	ck.load()
	ck.checkTestees()
	ck.checkTests()
	ck.checkOrder()
	ck.print()
}

// load loads all type, function and method names from the current package.
func (ck *TestNameChecker) load() {
	isRelevant := func(info os.FileInfo) bool {
		return ck.isRelevant(info.Name(), "*", "*", EAll)
	}

	fileSet := token.NewFileSet()
	pkgs, err := parser.ParseDir(fileSet, ".", isRelevant, 0)
	if err != nil {
		panic(err)
	}

	var pkgnames []string
	for pkgname := range pkgs {
		pkgnames = append(pkgnames, pkgname)
	}
	sort.Strings(pkgnames)

	for _, pkgname := range pkgnames {
		files := pkgs[pkgname].Files

		var filenames []string
		for filename := range files {
			filenames = append(filenames, filename)
		}
		sort.Strings(filenames)

		for _, filename := range filenames {
			file := files[filename]
			for _, decl := range file.Decls {
				ck.loadDecl(decl, filename)
			}
		}
	}

	ck.relate()
}

// loadDecl adds a single type or function declaration to the known elements.
func (ck *TestNameChecker) loadDecl(decl ast.Decl, filename string) {
	switch decl := decl.(type) {

	case *ast.GenDecl:
		for _, spec := range decl.Specs {
			switch spec := spec.(type) {
			case *ast.TypeSpec:
				typeName := spec.Name.Name
				if ck.isRelevant(filename, typeName, "*", EAll) {
					ck.addCode(code{filename, typeName, "", ck.nextOrder()})
				}
			}
		}

	case *ast.FuncDecl:
		typeName := ""
		if decl.Recv != nil {
			typeExpr := decl.Recv.List[0].Type.(ast.Expr)
			if star, ok := typeExpr.(*ast.StarExpr); ok {
				typeName = star.X.(*ast.Ident).Name
			} else {
				typeName = typeExpr.(*ast.Ident).Name
			}
		}
		funcName := decl.Name.Name
		if ck.isRelevant(filename, typeName, funcName, EAll) {
			ck.addCode(code{filename, typeName, funcName, ck.nextOrder()})
		}
	}
}

func (ck *TestNameChecker) addCode(code code) {
	if code.isTest() {
		ck.addTest(code)
	} else {
		ck.addTestee(code)
	}
}

func (ck *TestNameChecker) addTestee(code code) {
	ck.testees = append(ck.testees, &testee{code})
}

func (ck *TestNameChecker) addTest(code code) {
	if !strings.HasPrefix(code.Func, "Test_") && code.Func != "Test" {
		ck.addError(
			EName,
			code,
			"Test %q must start with %q.",
			code.fullName(), "Test_")
		return
	}

	parts := strings.SplitN(code.Func, "__", 2)
	testeeName := strings.TrimPrefix(strings.TrimPrefix(parts[0], "Test"), "_")
	descr := ""
	if len(parts) > 1 {
		if parts[1] == "" {
			ck.addError(
				EName,
				code,
				"Test %q must not have a nonempty description.",
				code.fullName())
			return
		}
		descr = parts[1]
	}

	ck.tests = append(ck.tests, &test{code, testeeName, descr, nil})
}

func (ck *TestNameChecker) nextOrder() int {
	id := ck.order
	ck.order++
	return id
}

// relate connects the tests to their testees.
func (ck *TestNameChecker) relate() {
	testeesByPrefix := make(map[string]*testee)
	for _, testee := range ck.testees {
		prefix := join(testee.Type, "_", testee.Func)
		testeesByPrefix[prefix] = testee
	}

	for _, test := range ck.tests {
		test.testee = testeesByPrefix[test.testeeName]
	}
}

func (ck *TestNameChecker) checkTests() {
	for _, test := range ck.tests {
		ck.checkTestFile(test)
		ck.checkTestTestee(test)
		ck.checkTestDescr(test)
	}
}

func (ck *TestNameChecker) checkTestFile(test *test) {
	testee := test.testee
	if testee == nil || testee.file == test.file {
		return
	}

	correctTestFile := strings.TrimSuffix(testee.file, ".go") + "_test.go"
	if correctTestFile == test.file {
		return
	}

	ck.addError(
		EFile,
		test.code,
		"Test %q for %q must be in %s instead of %s.",
		test.fullName(), testee.fullName(), correctTestFile, test.file)
}

func (ck *TestNameChecker) checkTestTestee(test *test) {
	testee := test.testee
	if testee != nil || test.testeeName == "" {
		return
	}

	testeeName := strings.Replace(test.testeeName, "_", ".", -1)
	ck.addError(
		EMissingTestee,
		test.code,
		"Missing testee %q for test %q.",
		testeeName, test.fullName())
}

// checkTestDescr ensures that the type or function name of the testee
// does not accidentally end up in the description of the test. This could
// happen if there is a double underscore instead of a single underscore.
func (ck *TestNameChecker) checkTestDescr(test *test) {
	testee := test.testee
	if testee == nil || testee.isMethod() || !isCamelCase(test.descr) {
		return
	}

	ck.addError(
		EName,
		testee.code,
		"%s: Test description %q must not use CamelCase in the first word.",
		test.fullName(), test.descr)
}

func (ck *TestNameChecker) checkTestees() {
	tested := make(map[*testee]bool)
	for _, test := range ck.tests {
		tested[test.testee] = true
	}

	for _, testee := range ck.testees {
		ck.checkTesteeTest(testee, tested)
	}
}

func (ck *TestNameChecker) checkTesteeTest(testee *testee, tested map[*testee]bool) {
	if tested[testee] || testee.isType() {
		return
	}

	testName := "Test_" + join(testee.Type, "_", testee.Func)
	ck.addError(
		EMissingTest,
		testee.code,
		"Missing unit test %q for %q.",
		testName, testee.fullName())
}

// isRelevant checks whether the given error is enabled.
// In each of the filter arguments, "*" means any.
func (ck *TestNameChecker) isRelevant(filename, typeName, funcName string, e Error) bool {

	matches := func(subj string, pattern string) bool {
		ok, err := path.Match(pattern, subj)
		if err != nil {
			panic(err)
		}
		return subj == "*" || ok
	}

	mask := ^uint64(0)
	for _, filter := range ck.filters {
		if matches(filename, filter.filenames) &&
			matches(typeName, filter.typeNames) &&
			matches(funcName, filter.funcNames) {
			mask = ck.errorsMask(mask, filter.errors...)
		}
	}
	return mask&ck.errorsMask(0, e) != 0
}

func (ck *TestNameChecker) errorsMask(mask uint64, errors ...Error) uint64 {
	for _, err := range errors {
		if err == ENone {
			mask = 0
		} else if err == EAll {
			mask = ^uint64(0)
		} else if err < 0 {
			mask &= ^(uint64(1) << -uint(err))
		} else {
			mask |= uint64(1) << uint(err)
		}
	}
	return mask
}

// checkOrder ensures that the tests appear in the same order as their
// counterparts in the main code.
func (ck *TestNameChecker) checkOrder() {
	maxOrderByFile := make(map[string]*test)

	for _, test := range ck.tests {
		testee := test.testee
		if testee == nil {
			continue
		}

		maxOrder := maxOrderByFile[testee.file]
		if maxOrder == nil || testee.order > maxOrder.testee.order {
			maxOrderByFile[testee.file] = test
		}

		if maxOrder != nil && testee.order < maxOrder.testee.order {
			insertBefore := maxOrder
			for _, before := range ck.tests {
				if before.file == test.file && before.testee != nil && before.testee.order > testee.order {
					insertBefore = before
					break
				}
			}
			ck.addError(
				EOrder,
				test.code,
				"Test %q must be ordered before %q.",
				test.fullName(), insertBefore.fullName())
		}
	}
}

func (ck *TestNameChecker) addError(e Error, c code, format string, args ...interface{}) {
	if ck.isRelevant(c.file, c.Type, c.Func, e) {
		ck.errors = append(ck.errors, fmt.Sprintf(format, args...))
	}
}

func (ck *TestNameChecker) print() {
	for _, msg := range ck.errors {
		_, _ = fmt.Fprintln(ck.out, msg)
	}

	if len(ck.errors) > 0 {
		ck.errorf("%s.", plural(len(ck.errors), "error", "errors"))
	}
}

type filter struct {
	filenames string
	typeNames string
	funcNames string
	errors    []Error
}

// testee is an element of the source code that can be tested.
// It is either a type, a function or a method.
type testee struct {
	code
}

type test struct {
	code

	testeeName string // The method name without the "Test_" prefix and description
	descr      string // The part after the "__" in the method name
	testee     *testee
}

type code struct {
	file  string // the file containing the code
	Type  string // the type, e.g. MkLine
	Func  string // the function or method name, e.g. Warnf
	order int    // the relative order in the file
}

func (c *code) fullName() string { return join(c.Type, ".", c.Func) }
func (c *code) isType() bool     { return c.Func == "" }
func (c *code) isMethod() bool   { return c.Type != "" && c.Func != "" }

func (c *code) isTest() bool {
	return c.isTestScope() && strings.HasPrefix(c.Func, "Test")
}
func (c *code) isTestScope() bool {
	return strings.HasSuffix(c.file, "_test.go")
}

func plural(n int, sg, pl string) string {
	if n == 0 {
		return ""
	}
	form := pl
	if n == 1 {
		form = sg
	}
	return fmt.Sprintf("%d %s", n, form)
}

func isCamelCase(str string) bool {
	for i := 0; i+1 < len(str); i++ {
		if str[i] == '_' {
			return false
		}
		if unicode.IsLower(rune(str[i])) && unicode.IsUpper(rune(str[i+1])) {
			return true
		}
	}
	return false
}

func join(a, sep, b string) string {
	if a == "" || b == "" {
		sep = ""
	}
	return a + sep + b
}
