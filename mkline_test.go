package main

import (
	check "gopkg.in/check.v1"
)

func (s *Suite) TestChecklineMkVartype_SimpleType(c *check.C) {
	s.UseCommandLine(c, "-Wtypes", "-Dunchecked")
	G.globalData.InitVartypes()
	mkline := NewMkLine(NewLine("fname", 1, "COMMENT=\tA nice package", nil))

	vartype1 := G.globalData.vartypes["COMMENT"]
	c.Assert(vartype1, check.NotNil)
	c.Check(vartype1.guessed, equals, guNotGuessed)

	vartype := mkline.getVariableType("COMMENT")

	c.Assert(vartype, check.NotNil)
	c.Check(vartype.checker.name, equals, "Comment")
	c.Check(vartype.guessed, equals, guNotGuessed)
	c.Check(vartype.kindOfList, equals, lkNone)

	mkline.checkVartype("COMMENT", "=", "A nice package", "")

	c.Check(s.Stdout(), equals, "WARN: fname:1: COMMENT should not begin with \"A\".\n")
}

func (s *Suite) TestChecklineMkVartype(c *check.C) {
	G.globalData.InitVartypes()
	mkline := NewMkLine(NewLine("fname", 1, "DISTNAME=gcc-${GCC_VERSION}", nil))

	mkline.checkVartype("DISTNAME", "=", "gcc-${GCC_VERSION}", "")
}

func (s *Suite) TestChecklineMkVaralign(c *check.C) {
	s.UseCommandLine(c, "-Wspace", "-f")
	lines := s.NewLines("file.mk",
		"VAR=   value",    // Indentation 7, fixed to 8.
		"VAR=    value",   // Indentation 8, fixed to 8.
		"VAR=     value",  // Indentation 9, fixed to 16.
		"VAR= \tvalue",    // Mixed indentation 8, fixed to 8.
		"VAR=   \tvalue",  // Mixed indentation 8, fixed to 8.
		"VAR=    \tvalue", // Mixed indentation 16, fixed to 16.
		"VAR=\tvalue")     // Already aligned with tabs only, left unchanged.

	for _, line := range lines {
		NewMkLine(line).checkVaralign()
	}

	c.Check(lines[0].changed, equals, true)
	c.Check(lines[0].rawLines()[0].String(), equals, "1:VAR=\tvalue\n")
	c.Check(lines[1].changed, equals, true)
	c.Check(lines[1].rawLines()[0].String(), equals, "2:VAR=\tvalue\n")
	c.Check(lines[2].changed, equals, true)
	c.Check(lines[2].rawLines()[0].String(), equals, "3:VAR=\t\tvalue\n")
	c.Check(lines[3].changed, equals, true)
	c.Check(lines[3].rawLines()[0].String(), equals, "4:VAR=\tvalue\n")
	c.Check(lines[4].changed, equals, true)
	c.Check(lines[4].rawLines()[0].String(), equals, "5:VAR=\tvalue\n")
	c.Check(lines[5].changed, equals, true)
	c.Check(lines[5].rawLines()[0].String(), equals, "6:VAR=\t\tvalue\n")
	c.Check(lines[6].changed, equals, false)
	c.Check(lines[6].rawLines()[0].String(), equals, "7:VAR=\tvalue\n")
	c.Check(s.Output(), equals, ""+
		"NOTE: file.mk:1: Alignment of variable values should be done with tabs, not spaces.\n"+
		"NOTE: file.mk:1: Autofix: replacing \"VAR=   \" with \"VAR=\\t\".\n"+
		"NOTE: file.mk:2: Alignment of variable values should be done with tabs, not spaces.\n"+
		"NOTE: file.mk:2: Autofix: replacing \"VAR=    \" with \"VAR=\\t\".\n"+
		"NOTE: file.mk:3: Alignment of variable values should be done with tabs, not spaces.\n"+
		"NOTE: file.mk:3: Autofix: replacing \"VAR=     \" with \"VAR=\\t\\t\".\n"+
		"NOTE: file.mk:4: Alignment of variable values should be done with tabs, not spaces.\n"+
		"NOTE: file.mk:4: Autofix: replacing \"VAR= \\t\" with \"VAR=\\t\".\n"+
		"NOTE: file.mk:5: Alignment of variable values should be done with tabs, not spaces.\n"+
		"NOTE: file.mk:5: Autofix: replacing \"VAR=   \\t\" with \"VAR=\\t\".\n"+
		"NOTE: file.mk:6: Alignment of variable values should be done with tabs, not spaces.\n"+
		"NOTE: file.mk:6: Autofix: replacing \"VAR=    \\t\" with \"VAR=\\t\\t\".\n")
	c.Check(tabLength("VAR=    \t"), equals, 16)
}

func (s *Suite) TestMkLine_fields(c *check.C) {
	mklines := NewMkLines(s.NewLines("test.mk",
		"VARNAME.param?=value # varassign comment",
		"\tshell command # shell comment",
		"# whole line comment",
		"",
		".  if !empty(PKGNAME:M*-*) # cond comment",
		".include \"../../mk/bsd.prefs.mk\" # include comment",
		".include <subdir.mk> # sysinclude comment",
		"target1 target2: source1 source2",
		"target : source",
		"VARNAME+=value"))
	ln := mklines.mklines

	c.Check(ln[0].IsVarassign(), equals, true)
	c.Check(ln[0].Varname(), equals, "VARNAME.param")
	c.Check(ln[0].Varcanon(), equals, "VARNAME.*")
	c.Check(ln[0].Varparam(), equals, "param")
	c.Check(ln[0].Op(), equals, "?=")
	c.Check(ln[0].Value(), equals, "value")
	c.Check(ln[0].Comment(), equals, "# varassign comment")

	c.Check(ln[1].IsShellcmd(), equals, true)
	c.Check(ln[1].Shellcmd(), equals, "shell command # shell comment")

	c.Check(ln[2].IsComment(), equals, true)
	c.Check(ln[2].Comment(), equals, " whole line comment")

	c.Check(ln[3].IsEmpty(), equals, true)

	c.Check(ln[4].IsCond(), equals, true)
	c.Check(ln[4].Indent(), equals, "  ")
	c.Check(ln[4].Directive(), equals, "if")
	c.Check(ln[4].Args(), equals, "!empty(PKGNAME:M*-*)")
	c.Check(ln[4].Comment(), equals, "") // Not needed

	c.Check(ln[5].IsInclude(), equals, true)
	c.Check(ln[5].MustExist(), equals, true)
	c.Check(ln[5].Includefile(), equals, "../../mk/bsd.prefs.mk")
	c.Check(ln[5].Comment(), equals, "") // Not needed

	c.Check(ln[6].IsSysinclude(), equals, true)
	c.Check(ln[6].MustExist(), equals, true)
	c.Check(ln[6].Includefile(), equals, "subdir.mk")
	c.Check(ln[6].Comment(), equals, "") // Not needed

	c.Check(ln[7].IsDependency(), equals, true)
	c.Check(ln[7].Targets(), equals, "target1 target2")
	c.Check(ln[7].Sources(), equals, "source1 source2")
	c.Check(ln[7].Comment(), equals, "") // Not needed

	c.Check(ln[9].IsVarassign(), equals, true)
	c.Check(ln[9].Varname(), equals, "VARNAME")
	c.Check(ln[9].Varcanon(), equals, "VARNAME")
	c.Check(ln[9].Varparam(), equals, "")

	c.Check(s.Output(), equals, "WARN: test.mk:9: Space before colon in dependency line.\n")
}

func (s *Suite) TestMkLine_checkVarassign(c *check.C) {
	G.pkg = NewPackage("graphics/gimp-fix-ca")
	G.globalData.InitVartypes()
	mkline := NewMkLine(NewLine("fname", 10, "MASTER_SITES=http://registry.gimp.org/file/fix-ca.c?action=download&id=9884&file=", nil))

	mkline.checkVarassign()

	c.Check(s.Output(), equals, "")
}

func (s *Suite) TestParseMkCond_NotEmptyMatch(c *check.C) {
	mkline := NewMkLine(NewLine("fname", 1, ".if !empty(USE_LIBTOOL:M[Yy][Ee][Ss])", nil))

	cond := mkline.parseMkCond(mkline.Args())

	c.Check(cond, check.DeepEquals, NewTree("not", NewTree("empty", NewTree("match", "USE_LIBTOOL", "[Yy][Ee][Ss]"))))
}

func (s *Suite) TestParseMkCond_Compare(c *check.C) {
	mkline := NewMkLine(NewLine("fname", 1, ".if ${VARNAME} != \"Value\"", nil))

	cond := mkline.parseMkCond(mkline.Args())

	c.Check(cond, check.DeepEquals, NewTree("compareVarStr", "VARNAME", "!=", "Value"))
}

func (s *Suite) TestChecklineMkCondition(c *check.C) {
	s.UseCommandLine(c, "-Wtypes")
	G.globalData.InitVartypes()

	NewMkLine(NewLine("fname", 1, ".if !empty(PKGSRC_COMPILER:Mmycc)", nil)).checkIf()

	c.Check(s.Stdout(), equals, "WARN: fname:1: Invalid :M value \"mycc\". "+
		"Only { ccache ccc clang distcc f2c gcc hp icc ido gcc mipspro "+
		"mipspro-ucode pcc sunpro xlc } are allowed.\n")

	NewMkLine(NewLine("fname", 1, ".elif ${A} != ${B}", nil)).checkIf()

	c.Check(s.Stdout(), equals, "") // Unknown condition types are silently ignored

	NewMkLine(NewLine("fname", 1, ".if ${HOMEPAGE} == \"mailto:someone@example.org\"", nil)).checkIf()

	c.Check(s.Output(), equals, "WARN: fname:1: \"mailto:someone@example.org\" is not a valid URL.\n")
}

func (s *Suite) TestMkLine_variableNeedsQuoting(c *check.C) {
	mkline := NewMkLine(NewLine("fname", 1, "PKGNAME := ${UNKNOWN}", nil))
	G.globalData.InitVartypes()
	pkgnameType := G.globalData.vartypes["PKGNAME"]

	vuc := &VarUseContext{pkgnameType, vucTimeParse, vucQuotUnknown, vucExtentUnknown}
	nq := mkline.variableNeedsQuoting("UNKNOWN", vuc)

	c.Check(nq, equals, nqDontKnow)
}

func (s *Suite) TestMkLine_variableNeedsQuoting_Varbase(c *check.C) {
	mkline := NewMkLine(NewLine("fname", 1, "# dummy", nil))
	G.globalData.InitVartypes()

	t1 := mkline.getVariableType("FONT_DIRS")

	c.Assert(t1, check.NotNil)
	c.Check(t1.String(), equals, "ShellList of Pathmask")

	t2 := mkline.getVariableType("FONT_DIRS.ttf")

	c.Assert(t2, check.NotNil)
	c.Check(t2.String(), equals, "ShellList of Pathmask")
}

func (s *Suite) TestVarUseContext_ToString(c *check.C) {
	G.globalData.InitVartypes()
	mkline := NewMkLine(NewLine("fname", 1, "# dummy", nil))
	vartype := mkline.getVariableType("PKGNAME")
	vuc := &VarUseContext{vartype, vucTimeUnknown, vucQuotBackt, vucExtentWord}

	c.Check(vuc.String(), equals, "(unknown PkgName backt word)")
}

func (s *Suite) TestMkLine_(c *check.C) {
	G.globalData.InitVartypes()

	G.mk = s.NewMkLines("Makefile",
		"# $"+"NetBSD$",
		"ac_cv_libpari_libs+=\t-L${BUILDLINK_PREFIX.pari}/lib", // From math/clisp-pari/Makefile, rev. 1.8
		"var+=value")

	G.mk.mklines[1].checkVarassign()
	G.mk.mklines[2].checkVarassign()

	c.Check(s.Output(), equals, ""+
		"WARN: Makefile:2: ac_cv_libpari_libs is defined but not used. Spelling mistake?\n"+
		"WARN: Makefile:3: As var is modified using \"+=\", its name should indicate plural.\n"+
		"WARN: Makefile:3: var is defined but not used. Spelling mistake?\n")
}
