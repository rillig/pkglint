package pkglint

import "testing"

func Test_Var_ConditionalVars(tt *testing.T) {
	t := NewTester(tt)
	defer t.Done()

	v := NewVar("VARNAME")

	t.CheckEquals(v.IsConditional(), false)
	t.CheckNil(v.ConditionalVars())

	v.Write(t.NewMkLine("write.mk", 123, "VARNAME=\tconditional"), true, "OPSYS")

	t.CheckEquals(v.IsConstant(), false)
	t.CheckEquals(v.IsConditional(), true)
	t.CheckDeepEquals(v.ConditionalVars(), []string{"OPSYS"})

	v.Write(t.NewMkLine("write.mk", 124, "VARNAME=\tconditional"), true, "OPSYS")

	t.CheckEquals(v.IsConditional(), true)
	t.CheckDeepEquals(v.ConditionalVars(), []string{"OPSYS"})
}

func Test_Var_Refs(tt *testing.T) {
	t := NewTester(tt)
	defer t.Done()

	v := NewVar("VAR")

	t.CheckNil(v.Refs())

	// The referenced variables are taken from the mkline.
	// They don't need to be passed separately.
	v.Write(t.NewMkLine("write.mk", 123, "VAR=${OTHER} ${${OPSYS} == NetBSD :? ${THEN} : ${ELSE}}"), true, "COND")

	v.AddRef("FOR")

	t.CheckDeepEquals(v.Refs(), []string{"OTHER", "OPSYS", "THEN", "ELSE", "COND", "FOR"})
}

func Test_Var_ConstantValue__assign(tt *testing.T) {
	t := NewTester(tt)
	defer t.Done()

	v := NewVar("VARNAME")

	v.Write(t.NewMkLine("write.mk", 123, "VARNAME=\tvalue"), false)

	t.CheckEquals(v.ConstantValue(), "value")

	v.Write(t.NewMkLine("write.mk", 124, "VARNAME=\toverwritten"), false)

	t.CheckEquals(v.ConstantValue(), "overwritten")
}

// Variables that reference other variable are considered constants.
// Even if these referenced variables change their value in-between,
// this does not affect the constant-ness of this variable, since the
// references are resolved lazily.
func Test_Var_ConstantValue__assign_reference(tt *testing.T) {
	t := NewTester(tt)
	defer t.Done()

	v := NewVar("VARNAME")

	v.Write(t.NewMkLine("write.mk", 123, "VARNAME=\tvalue"), false)

	t.CheckEquals(v.ConstantValue(), "value")

	v.Write(t.NewMkLine("write.mk", 124, "VARNAME=\t${OTHER}"), false)

	t.CheckEquals(v.IsConstant(), true)
}

func Test_Var_ConstantValue__assign_eval_reference(tt *testing.T) {
	t := NewTester(tt)
	defer t.Done()

	v := NewVar("VARNAME")

	v.Write(t.NewMkLine("write.mk", 123, "VARNAME=\tvalue"), false)

	t.CheckEquals(v.ConstantValue(), "value")

	v.Write(t.NewMkLine("write.mk", 124, "VARNAME:=\t${OTHER}"), false)

	// To analyze this case correctly, pkglint would have to know
	// the current value of ${OTHER} in line 124. For that it would
	// need the complete scope including all other variables.
	//
	// As of March 2019 this is not implemented, therefore pkglint
	// doesn't treat the variable as constant, to prevent wrong warnings.
	t.CheckEquals(v.IsConstant(), false)
}

func Test_Var_ConstantValue__assign_conditional(tt *testing.T) {
	t := NewTester(tt)
	defer t.Done()

	v := NewVar("VARNAME")

	t.CheckNil(v.ConditionalVars())

	v.Write(t.NewMkLine("write.mk", 123, "VARNAME=\tconditional"), true, "OPSYS")

	t.CheckEquals(v.IsConstant(), false)
}

func Test_Var_ConstantValue__default(tt *testing.T) {
	t := NewTester(tt)
	defer t.Done()

	v := NewVar("VARNAME")

	v.Write(t.NewMkLine("write.mk", 123, "VARNAME?=\tvalue"), false)

	t.CheckEquals(v.ConstantValue(), "value")

	v.Write(t.NewMkLine("write.mk", 124, "VARNAME?=\tignored"), false)

	t.CheckEquals(v.ConstantValue(), "value")
}

func Test_Var_ConstantValue__eval_then_default(tt *testing.T) {
	t := NewTester(tt)
	defer t.Done()

	v := NewVar("VARNAME")

	v.Write(t.NewMkLine("buildlink3.mk", 123, "VARNAME:=\tvalue"), false)

	t.CheckEquals(v.ConstantValue(), "value")

	v.Write(t.NewMkLine("builtin.mk", 124, "VARNAME?=\tignored"), false)

	t.CheckEquals(v.ConstantValue(), "value")
}

func Test_Var_ConstantValue__append(tt *testing.T) {
	t := NewTester(tt)
	defer t.Done()

	v := NewVar("VARNAME")

	v.Write(t.NewMkLine("write.mk", 123, "VARNAME+=\tvalue"), false)

	t.CheckEquals(v.ConstantValue(), "value")

	v.Write(t.NewMkLine("write.mk", 124, "VARNAME+=\tappended"), false)

	t.CheckEquals(v.ConstantValue(), "value appended")
}

func Test_Var_ConstantValue__eval(tt *testing.T) {
	t := NewTester(tt)
	defer t.Done()

	v := NewVar("VARNAME")

	v.Write(t.NewMkLine("write.mk", 123, "VARNAME:=\tvalue"), false)

	t.CheckEquals(v.ConstantValue(), "value")

	v.Write(t.NewMkLine("write.mk", 124, "VARNAME:=\toverwritten"), false)

	t.CheckEquals(v.ConstantValue(), "overwritten")
}

// Variables that are based on running shell commands are never constant.
func Test_Var_ConstantValue__shell(tt *testing.T) {
	t := NewTester(tt)
	defer t.Done()

	v := NewVar("VARNAME")

	v.Write(t.NewMkLine("write.mk", 123, "VARNAME=\tvalue"), false)

	t.CheckEquals(v.ConstantValue(), "value")

	v.Write(t.NewMkLine("write.mk", 124, "VARNAME!=\techo hello"), false)

	t.CheckEquals(v.IsConstant(), false)
}

func Test_Var_ConstantValue__referenced_before(tt *testing.T) {
	t := NewTester(tt)
	defer t.Done()

	v := NewVar("VARNAME")

	// Since the value of VARNAME escapes here, the value is not
	// guaranteed to be the same in all evaluations of ${VARNAME}.
	// For example, OTHER may be used at load time in an .if
	// condition.
	v.Read(t.NewMkLine("readwrite.mk", 123, "OTHER=\t${VARNAME}"))

	t.CheckEquals(v.IsConstant(), false)

	v.Write(t.NewMkLine("readwrite.mk", 124, "VARNAME=\tvalue"), false)

	t.CheckEquals(v.IsConstant(), false)
}

func Test_Var_ConstantValue__referenced_in_between(tt *testing.T) {
	t := NewTester(tt)
	defer t.Done()

	v := NewVar("VARNAME")

	v.Write(t.NewMkLine("readwrite.mk", 123, "VARNAME=\tvalue"), false)

	t.CheckEquals(v.ConstantValue(), "value")

	// Since the value of VARNAME escapes here, the value is not
	// guaranteed to be the same in all evaluations of ${VARNAME}.
	// For example, OTHER may be used at load time in an .if
	// condition.
	v.Read(t.NewMkLine("readwrite.mk", 124, "OTHER=\t${VARNAME}"))

	t.CheckEquals(v.ConstantValue(), "value")

	v.Write(t.NewMkLine("write.mk", 125, "VARNAME=\toverwritten"), false)

	t.CheckEquals(v.IsConstant(), false)
}

func Test_Var_Value__conditional_write_after_unconditional(tt *testing.T) {
	t := NewTester(tt)
	defer t.Done()

	v := NewVar("VARNAME")

	v.Write(t.NewMkLine("write.mk", 123, "VARNAME=\tvalue"), false)

	t.CheckEquals(v.Value(), "value")

	v.Write(t.NewMkLine("write.mk", 124, "VARNAME+=\tappended"), false)

	t.CheckEquals(v.Value(), "value appended")

	v.Write(t.NewMkLine("write.mk", 124, "VARNAME:=\toverwritten conditionally"), true, "OPSYS")

	// When there is a previous value, it's probably best to keep
	// that value since this way the following code results in the
	// most generic value:
	//  VAR=    generic
	//  .if ${OPSYS} == NetBSD
	//  VAR=    specific
	//  .endif
	// The value stays the same, still it is marked as conditional and therefore
	// not constant anymore.
	t.CheckEquals(v.IsConditional(), true)
	t.CheckEquals(v.IsConstant(), false)
	t.CheckEquals(v.Value(), "value appended")
}

func Test_Var_Value__infrastructure(tt *testing.T) {
	t := NewTester(tt)
	defer t.Done()

	v := NewVar("VARNAME")

	v.Write(t.NewMkLine(t.File("write.mk"), 123, "VARNAME=\tvalue"), false)

	t.CheckEquals(v.Value(), "value")

	v.Write(t.NewMkLine(t.File("mk/write.mk"), 123, "VARNAME=\tinfra"), false)

	t.CheckEquals(v.Value(), "value")

	v.Write(t.NewMkLine(t.File("wip/mk/write.mk"), 123, "VARNAME=\twip infra"), false)

	t.CheckEquals(v.Value(), "value")
}

func Test_Var_Value__initial_conditional_write(tt *testing.T) {
	t := NewTester(tt)
	defer t.Done()

	v := NewVar("VARNAME")

	v.Write(t.NewMkLine("write.mk", 124, "VARNAME:=\toverwritten conditionally"), true, "OPSYS")

	// Since there is no previous value, the simplest choice is to just
	// take the first seen value, no matter if that value is conditional
	// or not.
	t.CheckEquals(v.IsConditional(), true)
	t.CheckEquals(v.IsConstant(), false)
	t.CheckEquals(v.Value(), "overwritten conditionally")
}

func Test_Var_ValueInfra(tt *testing.T) {
	t := NewTester(tt)
	defer t.Done()

	v := NewVar("VARNAME")

	v.Write(t.NewMkLine(t.File("write.mk"), 123, "VARNAME=\tvalue"), false)

	t.CheckEquals(v.ValueInfra(), "value")

	v.Write(t.NewMkLine(t.File("mk/write.mk"), 123, "VARNAME=\tinfra"), false)

	t.CheckEquals(v.ValueInfra(), "infra")

	v.Write(t.NewMkLine(t.File("wip/mk/write.mk"), 123, "VARNAME=\twip infra"), false)

	t.CheckEquals(v.ValueInfra(), "wip infra")
}

func Test_Var_ReadLocations(tt *testing.T) {
	t := NewTester(tt)
	defer t.Done()

	v := NewVar("VAR")

	t.CheckNil(v.ReadLocations())

	mkline123 := t.NewMkLine("read.mk", 123, "OTHER=\t${VAR}")
	v.Read(mkline123)

	t.CheckDeepEquals(v.ReadLocations(), []*MkLine{mkline123})

	mkline124 := t.NewMkLine("read.mk", 124, "OTHER=\t${VAR} ${VAR}")
	v.Read(mkline124)
	v.Read(mkline124)

	// For now, count every read of the variable. I'm not yet sure
	// whether that's the best way or whether to make the lines unique.
	t.CheckDeepEquals(v.ReadLocations(), []*MkLine{mkline123, mkline124, mkline124})
}

func Test_Var_WriteLocations(tt *testing.T) {
	t := NewTester(tt)
	defer t.Done()

	v := NewVar("VAR")

	t.CheckNil(v.WriteLocations())

	mkline123 := t.NewMkLine("write.mk", 123, "VAR=\tvalue")
	v.Write(mkline123, false)

	t.CheckDeepEquals(v.WriteLocations(), []*MkLine{mkline123})

	// Multiple writes from the same line may happen because of a .for loop.
	mkline125 := t.NewMkLine("write.mk", 125, "VAR+=\t${var}")
	v.Write(mkline125, false)
	v.Write(mkline125, false)

	// For now, count every write of the variable. I'm not yet sure
	// whether that's the best way or whether to make the lines unique.
	t.CheckDeepEquals(v.WriteLocations(), []*MkLine{mkline123, mkline125, mkline125})
}

func Test_Var_Write__conditional_without_variables(tt *testing.T) {
	t := NewTester(tt)
	defer t.Done()

	mklines := t.NewMkLines("filename.mk",
		MkCvsID,
		".if exists(/usr/bin)",
		"VAR=\tvalue",
		".endif")

	scope := NewRedundantScope()
	mklines.ForEach(func(mkline *MkLine) {
		if mkline.IsVarassign() {
			t.CheckEquals(scope.get("VAR").vari.IsConditional(), false)
		}

		scope.checkLine(mklines, mkline)

		if mkline.IsVarassign() {
			t.CheckEquals(scope.get("VAR").vari.IsConditional(), true)
		}
	})
}

func Test_Var_Write__assertion(tt *testing.T) {
	t := NewTester(tt)
	defer t.Done()

	v := NewVar("VAR")
	t.ExpectAssert(
		func() { v.Write(t.NewMkLine("filename.mk", 1, "OTHER=value"), false, nil...) })
}
