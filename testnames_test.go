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

	type Testee struct {
		Type string
		Func string
		File string
	}

	elementString := func(testee Testee) string {
		return fmt.Sprintf("%s: %s", testee.File, strings.Join([]string{testee.Type, testee.Func}, "."))
	}

	// addTestee adds a single type or function declaration
	// to the testees.
	addTestee := func(testees *[]Testee, decl ast.Decl, fileName string) {
		switch decl := decl.(type) {

		case *ast.GenDecl:
			for _, spec := range decl.Specs {
				switch spec := spec.(type) {
				case *ast.TypeSpec:
					*testees = append(*testees, Testee{spec.Name.Name, "", fileName})
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
			*testees = append(*testees, Testee{typeName, decl.Name.Name, fileName})
		}
	}

	// loadAllTestees returns all type, function and method names
	// from the current package, in the form FunctionName or
	// TypeName.MethodName (omitting the * from the type name).
	loadAllTestees := func() []Testee {
		fset := token.NewFileSet()
		pkgs, err := parser.ParseDir(fset, ".", func(fi os.FileInfo) bool { return true }, 0)
		if err != nil {
			panic(err)
		}

		var testees []Testee
		for fileName, file := range pkgs["main"].Files {
			for _, decl := range file.Decls {
				addTestee(&testees, decl, fileName)
			}
		}

		sort.Slice(testees, func(i, j int) bool {
			ti := testees[i]
			tj := testees[j]
			switch {
			case ti.Type != tj.Type:
				return ti.Type < tj.Type
			case ti.Func != tj.Func:
				return ti.Func < tj.Func
			default:
				return ti.File < tj.File
			}
		})
		return testees
	}

	// generateTesteeNames generates a map containing all names for
	// testees as used in the test names. Examples:
	//
	//  Autofix
	//  Line_Warnf
	//  match5
	generateTesteeNames := func(testees []Testee) map[string]bool {
		prefixes := make(map[string]bool)
		for _, testee := range testees {
			var prefix string
			switch {
			case testee.Type != "" && testee.Func != "":
				prefix = testee.Type + "_" + testee.Func
			default:
				prefix = testee.Type + testee.Func
			}
			prefixes[prefix] = true
		}

		// Allow some special test name prefixes.
		prefixes["Varalign"] = true
		prefixes["ShellParser"] = true
		return prefixes
	}

	checkTestName := func(test Testee, testee string, descr string, prefixes map[string]bool) {
		if !prefixes[testee] {
			c.Errorf("%s: Testee %q not found.\n", elementString(test), testee)
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
				c.Errorf("%s: Test description must not use CamelCase.\n", elementString(test))
			}
		}
	}

	checkAll := func(testees []Testee, prefixes map[string]bool) {
		for _, test := range testees {
			if test.Type != "" && test.Func != "" {
				method := test.Func
				switch {
				case !hasPrefix(method, "Test"):
					// Ignore

				case hasPrefix(method, "Test__"):
					// OK

				case hasPrefix(method, "Test_"):
					refAndDescr := strings.SplitN(method[5:], "__", 2)
					descr := ""
					if len(refAndDescr) > 1 {
						descr = refAndDescr[1]
					}
					checkTestName(test, refAndDescr[0], descr, prefixes)

				default:
					c.Errorf("%s: Missing underscore.\n", elementString(test))
				}
			}
		}
	}

	testees := loadAllTestees()
	prefixes := generateTesteeNames(testees)
	checkAll(testees, prefixes)
}
