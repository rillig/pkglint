package main

import (
	"go/ast"
	"go/parser"
	"go/token"
	"gopkg.in/check.v1"
	"os"
	"sort"
	"strings"

	"netbsd.org/pkglint/regex"
)

// Ensures that all test names follow a common naming scheme:
//
//  Test_${Type}_${Method}__${description_using_underscores}
func (s *Suite) Test__test_names(c *check.C) {
	var allowed []string

	handleDecl := func(decl ast.Decl) {
		switch decl := decl.(type) {
		case *ast.FuncDecl:
			typeName := ""
			if decl.Recv != nil {
				typeExpr := decl.Recv.List[0].Type.(ast.Expr)
				if star, ok := typeExpr.(*ast.StarExpr); ok {
					typeName = star.X.(*ast.Ident).Name + "."
				} else {
					typeName = typeExpr.(*ast.Ident).Name + "."
				}
			}
			allowed = append(allowed, strings.Replace(typeName, "Impl.", ".", 1)+decl.Name.Name)

		case *ast.GenDecl:
			for _, spec := range decl.Specs {
				switch spec := spec.(type) {
				case *ast.TypeSpec:
					allowed = append(allowed, spec.Name.Name)
				}
			}
		}
	}

	collect := func() {
		fset := token.NewFileSet()
		pkgs, err := parser.ParseDir(fset, ".", func(fi os.FileInfo) bool { return true }, 0)
		if err != nil {
			panic(err)
		}

		for _, file := range pkgs["main"].Files {
			for _, decl := range file.Decls {
				handleDecl(decl)
			}
		}

		sort.SliceStable(allowed, func(i, j int) bool { return allowed[i] < allowed[j] })
	}

	collect()

	prefixes := make(map[string]bool)
	for _, funcName := range allowed {
		prefix := strings.Replace(funcName, ".", "_", 1)
		prefixes[prefix] = true
	}
	prefixes["Varalign"] = true
	prefixes["ShellParser"] = true

	for _, funcName := range allowed {
		typeAndMethod := strings.SplitN(funcName, ".", 2)
		if len(typeAndMethod) == 2 {
			method := typeAndMethod[1]
			switch {
			case !strings.HasPrefix(method, "Test"):
				// Ignore

			case strings.HasPrefix(method, "Test__"):
				// OK

			case strings.HasPrefix(method, "Test_"):
				refAndDescr := strings.SplitN(method[5:], "__", 2)
				testee := refAndDescr[0]
				if !prefixes[testee] {
					c.Errorf("%s: Testee %q not found.\n", funcName, testee)
				}
				if len(refAndDescr) == 1 {
					break
				}
				descr := refAndDescr[1]
				if regex.Compile(`\p{Ll}\p{Lu}`).MatchString(descr) {
					switch descr {
					case "comparing_YesNo_variable_to_string",
						"GitHub",
						"enumFrom",
						"dquotBacktDquot",
						"and_getSubdirs":
						// These exceptions are ok.

					default:
						c.Errorf("%s: Test description must not use CamelCase.\n", funcName)
					}
				}

			default:
				c.Errorf("%s: Missing underscore.\n", funcName)
			}
		}
	}
}
