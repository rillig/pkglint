// Package intqa provides quality assurance for the pkglint code.
package intqa

import (
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"io"
	"path/filepath"
	"strings"
	"unicode"
)

// TestNameChecker ensures that all test names follow a common naming scheme:
//  Test_${Type}_${Method}__${description_using_underscores}
// Each of the variable parts may be omitted.
type TestNameChecker struct {
	errorf func(format string, args ...interface{})

	ignoredFiles []string
	order        int

	testees []*testee
	tests   []*test

	warn     bool
	errors   []string
	warnings []string
	out      io.Writer
}

func NewTestNameChecker(errorf func(format string, args ...interface{})) *TestNameChecker {
	return &TestNameChecker{errorf: errorf}
}

func (ck *TestNameChecker) IgnoreFiles(fileGlob string) {
	ck.ignoredFiles = append(ck.ignoredFiles, fileGlob)
}

func (ck *TestNameChecker) ShowWarnings(warn bool) { ck.warn = warn }

func (ck *TestNameChecker) Check() {
	ck.load()
	ck.checkTestees()
	ck.checkTests()
	ck.checkOrder()
	ck.print()
}

// load loads all type, function and method names from the current package.
func (ck *TestNameChecker) load() {
	fileSet := token.NewFileSet()
	pkgs, err := parser.ParseDir(fileSet, ".", nil, 0)
	if err != nil {
		panic(err)
	}

	for _, pkg := range pkgs {
		for filename, file := range pkg.Files {
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
				ck.addCode(code{filename, typeName, "", ck.nextOrder()})
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
		ck.addCode(code{filename, typeName, decl.Name.Name, ck.nextOrder()})
	}
}

func (ck *TestNameChecker) addCode(code code) {
	isTest := strings.HasSuffix(code.file, "_test.go") &&
		code.Type != "" &&
		strings.HasPrefix(code.Func, "Test")

	if isTest {
		ck.addTest(code)
	} else {
		ck.addTestee(code)
	}
}

func (ck *TestNameChecker) addTestee(code code) {
	ck.testees = append(ck.testees, &testee{code})
}

func (ck *TestNameChecker) addTest(code code) {
	if !strings.HasPrefix(code.Func, "Test_") {
		ck.addError("Test %q must start with %q.", code.fullName(), "Test_")
		return
	}

	parts := strings.SplitN(code.Func, "__", 2)
	testeeName := strings.TrimPrefix(strings.TrimPrefix(parts[0], "Test"), "_")
	descr := ""
	if len(parts) > 1 {
		if parts[1] == "" {
			ck.addError("Test %q must not have a nonempty description.", code.fullName())
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
		ck.checkTestName(test)
		ck.checkTestTestee(test)
	}
}

func (ck *TestNameChecker) checkTestFile(test *test) {
	testee := test.testee
	if testee == nil || testee.file == test.file {
		return
	}

	correctTestFile := strings.TrimSuffix(testee.file, ".go") + "_test.go"
	if correctTestFile != test.file {
		ck.addError("Test %q for %q must be in %s instead of %s.",
			test.fullName(), testee.fullName(), correctTestFile, test.file)
	}
}

func (ck *TestNameChecker) checkTestTestee(test *test) {
	testee := test.testee
	if testee != nil || test.testeeName == "" {
		return
	}

	testeeName := strings.ReplaceAll(test.testeeName, "_", ".")
	ck.addError("Missing testee %q for test %q.",
		testeeName, test.fullName())
}

// checkTestName ensures that the method name does not accidentally
// end up in the description of the test. This could happen if there is a
// double underscore instead of a single underscore.
func (ck *TestNameChecker) checkTestName(test *test) {
	testee := test.testee
	if testee == nil {
		return
	}
	if testee.Type != "" && testee.Func != "" {
		return
	}
	if !isCamelCase(test.descr) {
		return
	}

	ck.addError(
		"%s: Test description %q must not use CamelCase in the first word.",
		test.fullName(), test.descr)
}

func (ck *TestNameChecker) checkTestees() {
	tested := make(map[*testee]bool)
	for _, test := range ck.tests {
		tested[test.testee] = true
	}

	for _, testee := range ck.testees {
		if tested[testee] || testee.Func == "" {
			continue
		}

		testName := "Test_" + join(testee.Type, "_", testee.Func)
		ck.addWarning("Missing unit test %q for %q.",
			testName, testee.fullName())
	}
}

func (ck *TestNameChecker) isIgnored(filename string) bool {
	for _, mask := range ck.ignoredFiles {
		ok, err := filepath.Match(mask, filename)
		if err != nil {
			panic(err)
		}
		if ok {
			return true
		}
	}
	return false
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
			ck.addWarning("Test %q should be ordered before %q.",
				test.fullName(), maxOrder.fullName())
		}
	}
}

func (ck *TestNameChecker) addError(format string, args ...interface{}) {
	ck.errors = append(ck.errors, "E: "+fmt.Sprintf(format, args...))
}

func (ck *TestNameChecker) addWarning(format string, args ...interface{}) {
	if ck.warn {
		ck.warnings = append(ck.warnings, "W: "+fmt.Sprintf(format, args...))
	}
}

func (ck *TestNameChecker) print() {
	for _, msg := range ck.errors {
		_, _ = fmt.Fprintln(ck.out, msg)
	}
	for _, msg := range ck.warnings {
		_, _ = fmt.Fprintln(ck.out, msg)
	}

	errors := plural(len(ck.errors), "error", "errors")
	warnings := plural(len(ck.warnings), "warning", "warnings")

	if len(ck.errors) > 0 || (ck.warn && len(ck.warnings) > 0) {
		ck.errorf("%s.",
			join(errors, " and ", warnings))
	}
}

type code struct {
	file  string // The file containing the code
	Type  string // The type, e.g. MkLine
	Func  string // The function or method name, e.g. Warnf
	order int    // The relative order in the file
}

func (c *code) fullName() string { return join(c.Type, ".", c.Func) }

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
