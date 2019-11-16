// Package intqa provides quality assurance for the pkglint code.
package intqa

import (
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"gopkg.in/check.v1"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"unicode"
)

// TestNameChecker ensures that all test names follow a common naming scheme:
//  Test_${Type}_${Method}__${description_using_underscores}
// Each of the variable parts may be omitted.
type TestNameChecker struct {
	c *check.C

	ignoredFiles []string
	prefixes     []testeePrefix

	warn     bool
	errors   []string
	warnings []string
}

func NewTestNameChecker(c *check.C) *TestNameChecker {
	return &TestNameChecker{c: c}
}

func (ck *TestNameChecker) IgnoreFiles(fileGlob string) {
	ck.ignoredFiles = append(ck.ignoredFiles, fileGlob)
}

func (ck *TestNameChecker) ShowWarnings(warn bool) { ck.warn = warn }

func (ck *TestNameChecker) Check() {
	elements := ck.loadAllElements()
	testeeByName := ck.collectTesteeByName(elements)
	ck.checkAll(elements, testeeByName)

	for _, err := range ck.errors {
		fmt.Println(err)
	}
	for _, warning := range ck.warnings {
		if ck.warn {
			fmt.Println(warning)
		}
	}
	if len(ck.errors) > 0 || (ck.warn && len(ck.warnings) > 0) {
		ck.c.Errorf("%d %s and %d %s.",
			len(ck.errors),
			plural(len(ck.errors), "error", "errors"),
			len(ck.warnings),
			plural(len(ck.warnings), "warning", "warnings"))
	}
}

// loadAllElements returns all type, function and method names
// from the current package, in the form FunctionName or
// TypeName.MethodName (omitting the * from the type name).
func (ck *TestNameChecker) loadAllElements() testees {
	fileSet := token.NewFileSet()
	pkgs, err := parser.ParseDir(fileSet, ".", func(fi os.FileInfo) bool { return true }, 0)
	if err != nil {
		panic(err)
	}

	var elements testees
	for _, pkg := range pkgs {
		for filename, file := range pkg.Files {
			for _, decl := range file.Decls {
				ck.addElement(&elements, decl, filename)
			}
		}
	}

	elements.sort()

	return elements
}

// addElement adds a single type or function declaration
// to the known elements.
func (ck *TestNameChecker) addElement(elements *testees, decl ast.Decl, filename string) {
	switch decl := decl.(type) {

	case *ast.GenDecl:
		for _, spec := range decl.Specs {
			switch spec := spec.(type) {
			case *ast.TypeSpec:
				typeName := spec.Name.Name
				elements.add(ck.newElement(typeName, "", filename))
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
		elements.add(ck.newElement(typeName, decl.Name.Name, filename))
	}
}

// collectTesteeByName generates a map containing the names of all
// testable elements, as used in the test names. Examples:
//
//  Autofix
//  Line_Warnf
//  match5
func (ck *TestNameChecker) collectTesteeByName(elements testees) map[string]*testeeElement {
	prefixes := make(map[string]*testeeElement)
	for _, element := range elements.elements {
		if element.prefix != "" { // Ignore tests named Test__description.
			prefixes[element.prefix] = element
		}
	}
	return prefixes
}

func (ck *TestNameChecker) checkAll(elements testees, testeeByName map[string]*testeeElement) {
	testNames := make(map[string]bool)

	for _, element := range elements.elements {
		if element.test {
			method := element.Func
			switch {
			case strings.HasPrefix(method, "Test__"):
				// OK

			case strings.HasPrefix(method, "Test_"):
				refAndDescr := strings.SplitN(method[5:], "__", 2)
				descr := ""
				if len(refAndDescr) > 1 {
					descr = refAndDescr[1]
				}
				testNames[refAndDescr[0]] = true
				ck.checkTestName(element, refAndDescr[0], descr, testeeByName)

			default:
				ck.addError("Test name %q must contain an underscore.", element.fullName)
			}
		}
	}

	for _, element := range elements.elements {
		if !strings.HasSuffix(element.file, "_test.go") && !ck.isIgnored(element.file) {
			if !testNames[element.prefix] {
				ck.addWarning("Missing unit test %q for %q.",
					"Test_"+element.prefix, element.fullName)
			}
		}
	}
}

func (ck *TestNameChecker) checkTestName(test *testeeElement, prefix string, descr string, testeeByName map[string]*testeeElement) {
	testee := testeeByName[prefix]
	if testee == nil {
		ck.addError("Test %q for missing testee %q.", test.fullName, prefix)

	} else if !strings.HasSuffix(testee.file, "_test.go") {
		correctTestFile := strings.TrimSuffix(testee.file, ".go") + "_test.go"
		if correctTestFile != test.file {
			ck.addError("Test %q for %q must be in %s instead of %s.",
				test.fullName, testee.fullName, correctTestFile, test.file)
		}
	}

	ck.checkTestNameCamelCase(descr, test)
}

// checkTestNameCamelCase ensures that the method name does not accidentally
// end up in the description of the test. This could happen if there is a
// double underscore instead of a single underscore.
func (ck *TestNameChecker) checkTestNameCamelCase(descr string, test *testeeElement) {
	if test.Type != "" && test.Func != "" {
		return // There's no way to accidentally write __ instead of _.
	}
	if !isCamelCase(descr) {
		return
	}

	ck.addError("%s: Test description %q must not use CamelCase.", test.fullName, descr)
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

func (ck *TestNameChecker) newElement(typeName, funcName, filename string) *testeeElement {
	typeName = strings.TrimSuffix(typeName, "Impl")

	fullName := join(typeName, ".", funcName)
	isTest := strings.HasSuffix(filename, "_test.go") && typeName != "" && strings.HasPrefix(funcName, "Test")

	var prefix string
	if isTest {
		testeeAndDescr := strings.TrimPrefix(funcName, "Test_")
		if testeeAndDescr == funcName {
			ck.addError("Test %q must start with %q.", fullName, "Test_")
		}
		parts := strings.SplitN(testeeAndDescr, "__", 2)
		if len(parts) > 1 && parts[1] == "" {
			ck.addError("Test %q must not have an empty description.", fullName)
		}
		prefix = parts[0]

	} else {
		prefix = join(typeName, "_", funcName)
	}

	return &testeeElement{filename, typeName, funcName, fullName, isTest, prefix}
}

func (ck *TestNameChecker) addError(format string, args ...interface{}) {
	ck.errors = append(ck.errors, "E: "+fmt.Sprintf(format, args...))
}

func (ck *TestNameChecker) addWarning(format string, args ...interface{}) {
	ck.warnings = append(ck.warnings, "W: "+fmt.Sprintf(format, args...))
}

type testeePrefix struct {
	prefix   string
	filename string
}

type testees struct {
	elements []*testeeElement
}

func (t *testees) add(element *testeeElement) {
	t.elements = append(t.elements, element)
}

func (t *testees) sort() {
	less := func(i, j int) bool {
		ei := t.elements[i]
		ej := t.elements[j]
		switch {
		case ei.Type != ej.Type:
			return ei.Type < ej.Type
		case ei.Func != ej.Func:
			return ei.Func < ej.Func
		default:
			return ei.file < ej.file
		}
	}

	sort.Slice(t.elements, less)
}

// testeeElement is an element of the source code that can be tested.
// It is either a type, a function or a method.
// The test methods are also testeeElements.
type testeeElement struct {
	file string // The file containing the testeeElement
	Type string // The type, e.g. MkLine
	Func string // The function or method name, e.g. Warnf

	fullName string // Type + "." + Func

	test bool // Whether the testeeElement is a test or a testee

	// For a test, its name without the description,
	// otherwise the prefix (Type + "_" + Func) for the corresponding tests
	prefix string
}

func plural(n int, sg, pl string) string {
	if n == 1 {
		return sg
	}
	return pl
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
