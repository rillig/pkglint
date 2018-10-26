package main

import (
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"gopkg.in/check.v1"
	"os"
	"sort"
	"strings"
)

// Ensures that all test names follow a common naming scheme:
//
//  Test_${Type}_${Method}__${description_using_underscores}
func (s *Suite) Test__test_names(c *check.C) {

	type Element struct {
		Type string
		Func string
		File string
	}

	var errors []string
	var warnings []string
	addError := func(format string, args ...interface{}) {
		errors = append(errors, "E: "+sprintf(format, args...))
	}
	addWarning := func(format string, args ...interface{}) {
		warnings = append(warnings, "W: "+sprintf(format, args...))
	}

	elementString := func(e *Element) string {
		sep := ifelseStr(e.Type != "" && e.Func != "", ".", "")
		return fmt.Sprintf("%s: %s%s%s", e.File, e.Type, sep, e.Func)
	}

	isTest := func(e *Element) bool {
		return hasSuffix(e.File, "_test.go") && e.Type != "" && hasPrefix(e.Func, "Test")
	}

	// addElement adds a single type or function declaration
	// to the known elements.
	addElement := func(elements *[]*Element, decl ast.Decl, fileName string) {
		switch decl := decl.(type) {

		case *ast.GenDecl:
			for _, spec := range decl.Specs {
				switch spec := spec.(type) {
				case *ast.TypeSpec:
					*elements = append(*elements, &Element{spec.Name.Name, "", fileName})
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
				typeName = strings.TrimSuffix(typeName, "Impl")
			}
			*elements = append(*elements, &Element{typeName, decl.Name.Name, fileName})
		}
	}

	// loadAllElements returns all type, function and method names
	// from the current package, in the form FunctionName or
	// TypeName.MethodName (omitting the * from the type name).
	loadAllElements := func() []*Element {
		fileSet := token.NewFileSet()
		pkgs, err := parser.ParseDir(fileSet, ".", func(fi os.FileInfo) bool { return true }, 0)
		if err != nil {
			panic(err)
		}

		var elements []*Element
		for fileName, file := range pkgs["main"].Files {
			for _, decl := range file.Decls {
				addElement(&elements, decl, fileName)
			}
		}

		sort.Slice(elements, func(i, j int) bool {
			ti := elements[i]
			tj := elements[j]
			switch {
			case ti.Type != tj.Type:
				return ti.Type < tj.Type
			case ti.Func != tj.Func:
				return ti.Func < tj.Func
			default:
				return ti.File < tj.File
			}
		})

		return elements
	}

	testName := func(element *Element) string {
		if isTest(element) {
			return ""
		}
		sep := ifelseStr(element.Type != "" && element.Func != "", "_", "")
		return element.Type + sep + element.Func
	}

	// collectTesteeByName generates a map containing the names of all
	// testable elements, as used in the test names. Examples:
	//
	//  Autofix
	//  Line_Warnf
	//  match5
	collectTesteeByName := func(elements []*Element) map[string]*Element {
		prefixes := make(map[string]*Element)
		for _, element := range elements {
			if prefix := testName(element); prefix != "" {
				prefixes[prefix] = element
			}
		}

		// Allow some special test name testeeByName.
		prefixes["Varalign"] = &Element{"Varalign", "", "mklines_varalign.go"}
		prefixes["ShellParser"] = &Element{"ShellParser", "", "mkshparser.go"}
		return prefixes
	}

	checkTestName := func(test *Element, testee string, descr string, testeeByName map[string]*Element) {
		if testeeByName[testee] == nil {
			addError("%s: Testee %q not found.", elementString(test), testee)
		}
		if matches(descr, `\p{Ll}\p{Lu}`) {
			switch descr {
			case "comparing_YesNo_variable_to_string",
				"GitHub",
				"enumFrom",
				"enumFromDirs",
				"dquotBacktDquot",
				"and_getSubdirs":
				// These exceptions are ok.

			default:
				addError("%s: Test description must not use CamelCase.", elementString(test))
			}
		}
	}

	checkAll := func(elements []*Element, testeeByName map[string]*Element) {
		testNames := make(map[string]bool)

		for _, test := range elements {
			if isTest(test) {
				method := test.Func
				switch {
				case hasPrefix(method, "Test__"):
					// OK

				case hasPrefix(method, "Test_"):
					refAndDescr := strings.SplitN(method[5:], "__", 2)
					descr := ""
					if len(refAndDescr) > 1 {
						descr = refAndDescr[1]
					}
					testNames[refAndDescr[0]] = true
					checkTestName(test, refAndDescr[0], descr, testeeByName)

				default:
					addError("%s: Missing underscore.", elementString(test))
				}
			}
		}

		for _, element := range elements {
			if !hasSuffix(element.File, "_test.go") && !hasSuffix(element.File, "yacc.go") {
				testNamePrefix := testName(element)
				if !testNames[testNamePrefix] {
					addWarning("%s: Does not have a unit test (Test_%s).", elementString(element), testNamePrefix)
				}
			}
		}
	}

	elements := loadAllElements()
	testeeByName := collectTesteeByName(elements)
	checkAll(elements, testeeByName)

	printWarnings := "no"[0] == 'y' // Set this to "yes" to enable warnings.

	for _, err := range errors {
		fmt.Println(err)
	}
	for _, warning := range warnings {
		if printWarnings {
			fmt.Println(warning)
		}
	}
	if len(errors) > 0 || (printWarnings && len(warnings) > 0) {
		fmt.Printf("%d errors and %d warnings.", len(errors), len(warnings))
	}
}
