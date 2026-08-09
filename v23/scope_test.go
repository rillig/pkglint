package pkglint

import "testing"

func Test_Scope__no_tracing(tt *testing.T) {
	t := NewTester(tt)
	defer t.Done()

	scope := NewScope()
	scope.Define("VAR.param", t.NewMkLine("fname.mk", 3, "VAR.param=\tvalue"))
	t.DisableTracing()

	t.CheckEquals(scope.IsDefinedSimilar("VAR.param"), true)
	t.CheckEquals(scope.IsDefinedSimilar("VAR.other"), true)
	t.CheckEquals(scope.IsDefinedSimilar("OTHER"), false)
}

func Test_Scope__commented_varassign(tt *testing.T) {
	t := NewTester(tt)
	defer t.Done()

	mkline := t.NewMkLine("mk/defaults/mk.conf", 3, "#VAR=default")
	scope := NewScope()
	scope.Define("VAR", mkline)

	t.CheckEquals(scope.IsDefined("VAR"), false)
	t.CheckNil(scope.FirstDefinition("VAR"))
	t.CheckNil(scope.LastDefinition("VAR"))

	t.CheckEquals(scope.Mentioned("VAR"), mkline)
	t.CheckEquals(scope.Commented("VAR"), mkline)

	value, found, indeterminate := scope.LastValueFound("VAR")
	t.CheckEquals(value, "")
	t.CheckEquals(found, false)
	t.CheckEquals(indeterminate, false)
}

func Test_NewScope(tt *testing.T) {
	t := NewTester(tt)
	defer t.Done()

	scope := NewScope()

	t.CheckNil(scope.names)
	t.CheckNotNil(scope.vs)
	t.CheckEquals(len(scope.vs), 0)
}

func Test_Scope_varnames(tt *testing.T) {
	t := NewTester(tt)
	defer t.Done()

	scope := NewScope()
	mkline := t.NewMkLine("filename.mk", 3, "DEFINED=\t${USED}")

	t.CheckNil(scope.varnames())

	scope.Define("DEFINED", mkline)
	scope.Use("USED", mkline, EctxRunTime)

	t.CheckDeepEquals(scope.varnames(), []string{"DEFINED", "USED"})
	scope.varnames()[0] = "modified" // just to demonstrate the caching
	t.CheckDeepEquals(scope.varnames(), []string{"modified", "USED"})
}

func Test_Scope_create(tt *testing.T) {
	t := NewTester(tt)
	defer t.Done()

	scope := NewScope()

	v1 := scope.create("VAR")
	v2 := scope.create("VAR")

	t.CheckEquals(v1, v2)
}

func Test_Scope_Define(tt *testing.T) {
	t := NewTester(tt)
	defer t.Done()

	scope := NewScope()

	test := func(line string, found, indeterminate bool, value string) {
		mkline := t.NewMkLine("file.mk", 123, line)
		scope.Define("BUILD_DIRS", mkline)

		actualValue, actualFound, actualIndeterminate := scope.LastValueFound("BUILD_DIRS")

		t.CheckDeepEquals(
			[]any{actualFound, actualIndeterminate, actualValue},
			[]any{found, indeterminate, value})
		t.CheckEquals(scope.vs["BUILD_DIRS"].value.String(), value)
	}

	test("BUILD_DIRS?=\tdefault",
		true, false, "default")

	test(
		"BUILD_DIRS=\tone two three",
		true, false, "one two three")

	test(
		"BUILD_DIRS+=\tfour",
		true, false, "one two three four")

	// Later default assignments do not have an effect.
	test("BUILD_DIRS?=\tdefault",
		true, false, "one two three four")

	test("BUILD_DIRS!=\techo dynamic",
		true, true, "")

	// The shell assignment above sets the variable to an indeterminate
	// value, after which all further default assignments are ignored.
	test("BUILD_DIRS?=\tdefault after shell assignment",
		true, true, "")
}

func Test_Scope_def(tt *testing.T) {
	t := NewTester(tt)
	defer t.Done()

	mkline := t.NewMkLine("filename.mk", 3, "VAR=\tvalue")
	scope := NewScope()

	scope.def("VAR.param", mkline)

	t.CheckNil(scope.FirstDefinition("VAR"))
	t.CheckEquals(scope.FirstDefinition("VAR.param"), mkline)
	t.CheckNil(scope.FirstDefinition("VAR.*"))
}

func Test_Scope_Fallback(tt *testing.T) {
	t := NewTester(tt)
	defer t.Done()

	mkline := t.NewMkLine("filename.mk", 3, "VAR=\tvalue")
	scope := NewScope()
	scope.def("VAR.param", mkline)

	scope.Fallback("FALLBACK", "fallback")

	t.CheckEquals(scope.LastValue("VAR.param"), "value")
	t.CheckEquals(scope.LastValue("FALLBACK"), "fallback")
	t.CheckEquals(scope.LastValue("UNDEFINED"), "")
}

func Test_Scope_Use(tt *testing.T) {
	t := NewTester(tt)
	defer t.Done()

	mkline := t.NewMkLine("filename.mk", 3, "VAR=\t${USED}")
	scope := NewScope()
	scope.Define("VAR", mkline)
	scope.Use("USED", mkline, EctxRunTime)

	t.CheckEquals(scope.LastValue("VAR"), "${USED}")
	t.CheckEquals(scope.LastValue("USED"), "")
}

func Test_Scope_Mentioned(tt *testing.T) {
	t := NewTester(tt)
	defer t.Done()

	assigned := t.NewMkLine("filename.mk", 3, "VAR=\tvalue")
	commented := t.NewMkLine("filename.mk", 4, "#COMMENTED=\tvalue")
	documented := t.NewMkLine("filename.mk", 5, "# DOCUMENTED is a variable.")

	scope := NewScope()
	scope.Define("VAR", assigned)
	scope.Define("COMMENTED", commented)
	scope.Define("DOCUMENTED", documented)

	t.CheckEquals(scope.Mentioned("VAR"), assigned)
	t.CheckEquals(scope.Mentioned("COMMENTED"), commented)
	t.CheckEquals(scope.Mentioned("DOCUMENTED"), documented)
	t.CheckNil(scope.Mentioned("UNKNOWN"))
}

func Test_Scope_IsDefined(tt *testing.T) {
	t := NewTester(tt)
	defer t.Done()

	scope := NewScope()
	scope.Define("VAR.param", t.NewMkLine("file.mk", 1, "VAR.param=value"))

	t.CheckEquals(scope.IsDefined("VAR.param"), true)
	t.CheckEquals(scope.IsDefined("VAR.other"), false)
	t.CheckEquals(scope.IsDefined("VARIABLE.*"), false)
}

func Test_Scope_IsDefinedSimilar(tt *testing.T) {
	t := NewTester(tt)
	defer t.Done()

	scope := NewScope()
	scope.Define("VAR.param", t.NewMkLine("file.mk", 1, "VAR.param=value"))

	t.CheckEquals(scope.IsDefinedSimilar("VAR.param"), true)
	t.CheckEquals(scope.IsDefinedSimilar("VAR.other"), true)
	t.CheckEquals(scope.IsDefinedSimilar("VARIABLE.*"), false)
}

func Test_Scope_IsUsed(tt *testing.T) {
	t := NewTester(tt)
	defer t.Done()

	scope := NewScope()
	mkline := t.NewMkLine("file.mk", 1, "\techo ${VAR.param}")
	scope.Use("VAR.param", mkline, EctxRunTime)

	t.CheckEquals(scope.IsUsed("VAR.param"), true)
	t.CheckEquals(scope.IsUsed("VAR.other"), false)
	t.CheckEquals(scope.IsUsed("VARIABLE.*"), false)
}

func Test_Scope_IsUsedSimilar(tt *testing.T) {
	t := NewTester(tt)
	defer t.Done()

	scope := NewScope()
	mkline := t.NewMkLine("file.mk", 1, "\techo ${VAR.param}")
	scope.Use("VAR.param", mkline, EctxRunTime)

	t.CheckEquals(scope.IsUsedSimilar("VAR.param"), true)
	t.CheckEquals(scope.IsUsedSimilar("VAR.other"), true)
	t.CheckEquals(scope.IsUsedSimilar("VARIABLE.*"), false)
}

func Test_Scope_IsUsedAtLoadTime(tt *testing.T) {
	t := NewTester(tt)
	defer t.Done()

	scope := NewScope()
	mkline := t.NewMkLine("file.mk", 1, "\techo ${VAR.param}")

	scope.Use("LOAD_TIME", mkline, EctxLoadTime)
	scope.Use("RUN_TIME", mkline, EctxRunTime)

	t.CheckEquals(scope.IsUsedAtLoadTime("LOAD_TIME"), true)
	t.CheckEquals(scope.IsUsedAtLoadTime("RUN_TIME"), false)
	t.CheckEquals(scope.IsUsedAtLoadTime("UNDEFINED"), false)
}

func Test_Scope_FirstDefinition(tt *testing.T) {
	t := NewTester(tt)
	defer t.Done()

	mkline3 := t.NewMkLine("fname.mk", 3, "VAR=\tvalue")
	mkline4 := t.NewMkLine("fname.mk", 4, ".if ${SNEAKY::=value}")
	mkline5 := t.NewMkLine("fname.mk", 5, ".if ${USED}")

	scope := NewScope()
	scope.Define("VAR", mkline3)
	scope.Define("SNEAKY", mkline4)
	scope.Use("USED", mkline5, EctxLoadTime)

	t.CheckEquals(scope.FirstDefinition("VAR"), mkline3)

	// This call returns nil because it's not a variable assignment
	// and the calling code typically assumes a variable definition.
	// These sneaky variables with implicit definition are an edge
	// case that only few people actually know. It's better that way.
	t.CheckNil(scope.FirstDefinition("SNEAKY"))

	t.CheckNil(scope.FirstDefinition("USED"))

	t.CheckOutputEmpty()
}

func Test_Scope_LastDefinition(tt *testing.T) {
	t := NewTester(tt)
	defer t.Done()

	mkline3 := t.NewMkLine("fname.mk", 3, "VAR=\tfirst")
	mkline4 := t.NewMkLine("fname.mk", 4, "VAR=\t${USED}")

	scope := NewScope()
	scope.Define("VAR", mkline3)
	scope.Define("VAR", mkline4)
	scope.Use("USED", mkline4, EctxRunTime)

	t.CheckEquals(scope.LastDefinition("VAR"), mkline4)
	t.CheckNil(scope.LastDefinition("UNDEFINED"))
	t.CheckNil(scope.LastDefinition("USED"))
}

func Test_Scope_Commented(tt *testing.T) {
	t := NewTester(tt)
	defer t.Done()

	assigned := t.NewMkLine("filename.mk", 3, "VAR=\tvalue")
	commented := t.NewMkLine("filename.mk", 4, "#COMMENTED=\tvalue")
	documented := t.NewMkLine("filename.mk", 5, "# DOCUMENTED is a variable.")
	used := t.NewMkLine("filename.mk", 6, "ANY=\t${USED}")

	scope := NewScope()
	scope.Define("VAR", assigned)
	scope.Define("COMMENTED", commented)
	scope.Define("DOCUMENTED", documented)
	scope.Use("USED", used, EctxRunTime)

	t.CheckNil(scope.Commented("VAR"))
	t.CheckEquals(scope.Commented("COMMENTED"), commented)
	t.CheckNil(scope.Commented("DOCUMENTED"))
	t.CheckNil(scope.Commented("UNKNOWN"))
	t.CheckNil(scope.Commented("USED"))
}

func Test_Scope_FirstUse(tt *testing.T) {
	t := NewTester(tt)
	defer t.Done()

	mklines := t.NewMkLines("file.mk",
		MkCvsID,
		"VAR1=\t${USED}",
		"VAR2=\t${USED}")

	mklines.Check()

	scope := mklines.allVars
	t.CheckEquals(scope.FirstUse("USED"), mklines.mklines[1])
	t.CheckNil(scope.FirstUse("UNUSED"))

	t.CheckOutputLines(
		"WARN: file.mk:2: Variable \"VAR1\" is defined but not used.",
		"WARN: file.mk:2: Variable \"USED\" is used but not defined.",
		"WARN: file.mk:3: Variable \"VAR2\" is defined but not used.")
}

func Test_Scope_LastValue(tt *testing.T) {
	t := NewTester(tt)
	defer t.Done()

	mklines := t.NewMkLines("file.mk",
		MkCvsID,
		"VAR=\tfirst",
		"VAR=\tsecond",
		".if 1",
		"VAR=\tthird (conditional)",
		".endif")

	mklines.Check()

	// TODO: At load time, use loadVars instead of allVars.
	t.CheckEquals(mklines.allVars.LastValue("VAR"), "third (conditional)")

	t.CheckOutputLines(
		"WARN: file.mk:2: Variable \"VAR\" is defined but not used.")
}

// Up to 2020-01-06, pkglint wrongly returned "one" as the variable value,
// even though Makefile.common is included before appending "two".
func Test_Scope_LastValue__append_in_multiple_files(tt *testing.T) {
	t := NewTester(tt)
	defer t.Done()

	t.SetUpPackage("category/package",
		".include \"Makefile.common\"",
		"PLIST_VARS+=\ttwo",
		"PLIST.two=\tyes")
	t.Chdir("category/package")
	t.CreateFileLines("PLIST",
		PlistCvsID,
		"${PLIST.one}${PLIST.two}bin/program")
	t.CreateFileLines("Makefile.common",
		MkCvsID,
		"PLIST_VARS=\tone",
		"PLIST.one=\tyes")
	pkg := NewPackage(".")
	t.FinishSetUp()

	pkg.Check()

	t.CheckEquals(pkg.vars.LastValue("PLIST_VARS"), "one two")
}

func Test_Scope_LastValueFound(tt *testing.T) {
	t := NewTester(tt)
	defer t.Done()

	mklines := t.NewMkLines("file.mk",
		MkCvsID,
		"VAR=\tfirst",
		"VAR=\tsecond",
		".if 1",
		"VAR=\tthird (conditional)",
		".endif")

	mklines.Check()

	value, found, indeterminate := mklines.allVars.LastValueFound("VAR")
	t.CheckEquals(value, "third (conditional)")
	t.CheckEquals(found, true)
	t.CheckEquals(indeterminate, false) // TODO: why?

	t.CheckOutputLines(
		"WARN: file.mk:2: Variable \"VAR\" is defined but not used.")
}

// Scope.DefineAll copies only the variable definitions,
// but not the uses of variables.
func Test_Scope_DefineAll(tt *testing.T) {
	t := NewTester(tt)
	defer t.Done()

	mkline := t.NewMkLine("filename.mk", 123, "VAR=\t${USED}")

	src := NewScope()
	src.Use("USED", mkline, EctxRunTime)

	dst := NewScope()
	dst.DefineAll(&src)

	t.CheckEquals(len(dst.vs), 0)

	src.Define("VAR", t.NewMkLine("file.mk", 1, "VAR=value"))
	dst.DefineAll(&src)

	t.CheckEquals(dst.IsDefined("VAR"), true)
}

func Test_Scope_forEach(tt *testing.T) {
	t := NewTester(tt)
	defer t.Done()

	mkline := t.NewMkLine("filename.mk", 123, "VAR=\t${USED}")

	scope := NewScope()
	scope.Define("VAR", mkline)
	scope.Use("USED", mkline, EctxRunTime)

	var result []string
	scope.forEach(func(varname string, data *scopeVar) {
		result = append(result, varname+"="+data.value.String())
	})

	t.CheckDeepEquals(result, []string{
		"USED=",
		"VAR=${USED}"})
}

func Test_scopeVar_append(tt *testing.T) {
	t := NewTester(tt)
	defer t.Done()

	var v scopeVar
	v.append("first")
	v.append("second")

	t.CheckEquals(v.value.String(), " first second")
}
