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
	t := s.Init(c)

	t.SetupCommandLine("--autofix") // For fixTabs.

	type Element struct {
		Type string // The type, e.g. MkLine
		Func string // The function or method name, e.g. Warnf
		File string // The file containing the element

		FullName string // Type + separator + Func

		// Whether the element is a test or a testee
		Test bool
		// For a test, its name without the description,
		// otherwise the prefix for the corresponding tests
		Prefix string
	}

	newElement := func(typeName, funcName, fileName string) *Element {
		e := &Element{File: fileName, Type: typeName, Func: funcName}

		sep := ifelseStr(e.Type != "" && e.Func != "", ".", "")
		e.FullName = e.Type + sep + e.Func

		e.Test = hasSuffix(e.File, "_test.go") && e.Type != "" && hasPrefix(e.Func, "Test")
		if e.Test {
			e.Prefix = strings.Split(strings.TrimPrefix(e.Func, "Test"), "__")[0]
		} else {
			sep := ifelseStr(e.Type != "" && e.Func != "", "_", "")
			e.Prefix = e.Type + sep + e.Func
		}
		return e
	}

	var errors []string
	var warnings []string
	addError := func(format string, args ...interface{}) {
		errors = append(errors, "E: "+sprintf(format, args...))
	}
	addWarning := func(format string, args ...interface{}) {
		warnings = append(warnings, "W: "+sprintf(format, args...))
	}

	// addElement adds a single type or function declaration
	// to the known elements.
	addElement := func(elements *[]*Element, decl ast.Decl, fileName string) {
		switch decl := decl.(type) {

		case *ast.GenDecl:
			for _, spec := range decl.Specs {
				switch spec := spec.(type) {
				case *ast.TypeSpec:
					*elements = append(*elements, newElement(spec.Name.Name, "", fileName))
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
			*elements = append(*elements, newElement(typeName, decl.Name.Name, fileName))
		}
	}

	// Replace tabs with spaces, except for the indentation tabs.
	fixTabs := func(fileName string) {
		if hasSuffix(fileName, "yacc.go") {
			return
		}

		lines := Load(fileName, MustSucceed|NotEmpty|LogErrors)
		for _, line := range lines.Lines {
			_, rest := match1(line.Text, `^\t*(.*)`)
			if contains(rest, "\t") {
				fix := line.Autofix()
				fix.Warnf("Tabs should only be used for indentation.")
				fix.Replace(rest, detab(rest))
				fix.Apply()
			}
		}

		SaveAutofixChanges(lines)
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
			fixTabs(fileName)
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

	// collectTesteeByName generates a map containing the names of all
	// testable elements, as used in the test names. Examples:
	//
	//  Autofix
	//  Line_Warnf
	//  match5
	collectTesteeByName := func(elements []*Element) map[string]*Element {
		prefixes := make(map[string]*Element)
		for _, element := range elements {
			if element.Prefix != "" {
				prefixes[element.Prefix] = element
			}
		}

		// Allow some special test name testeeByName.
		prefixes["Varalign"] = newElement("Varalign", "", "mklines_varalign.go")
		prefixes["ShellParser"] = newElement("ShellParser", "", "mkshparser.go")
		return prefixes
	}

	checkTestName := func(test *Element, prefix string, descr string, testeeByName map[string]*Element) {
		testee := testeeByName[prefix]
		if testee == nil {
			addError("Test %q for missing testee %q.", test.FullName, prefix)

		} else if !hasSuffix(testee.File, "_test.go") {
			correctTestFile := strings.TrimSuffix(testee.File, ".go") + "_test.go"
			if correctTestFile != test.File {
				addWarning("Test %q for %q should be in %s instead of %s.",
					test.FullName, testee.FullName, correctTestFile, test.File)
			}
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
				addError("%s: Test description must not use CamelCase.", test.FullName)
			}
		}
	}

	checkAll := func(elements []*Element, testeeByName map[string]*Element) {
		testNames := make(map[string]bool)

		for _, element := range elements {
			if element.Test {
				method := element.Func
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
					checkTestName(element, refAndDescr[0], descr, testeeByName)

				default:
					addError("Test name %q must contain an underscore.", element.FullName)
				}
			}
		}

		for _, element := range elements {
			if !hasSuffix(element.File, "_test.go") && !hasSuffix(element.File, "yacc.go") {
				if !testNames[element.Prefix] {
					addWarning("Missing unit test %q for %q.",
						"Test_"+element.FullName, element.Prefix)
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
