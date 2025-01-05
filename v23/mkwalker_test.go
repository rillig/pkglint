package pkglint

import "gopkg.in/check.v1"

func (s *Suite) Test_NewMkWalker(c *check.C) {
	t := s.Init(c)

	mkline := t.NewMkLine("filename.mk", 123, "LEFT.$p=\t${RIGHT}")
	action := func(expr *MkExpr, time EctxTime) {
		mkline.Notef("Expression \"%s\" at \"%s\" time.", expr.String(), time.String())
	}

	walker := NewMkWalker(mkline, action)
	walker.WalkLine(mkline)

	t.CheckOutputLines(
		"WARN: filename.mk:123: $p is ambiguous. Use ${p} if you mean a Make variable or $$p if you mean a shell variable.",
		"NOTE: filename.mk:123: Expression \"${p}\" at \"load\" time.",
		"NOTE: filename.mk:123: Expression \"${RIGHT}\" at \"run\" time.")
}

func (s *Suite) Test_MkWalker_WalkLine(c *check.C) {
	t := s.Init(c)

	mklines := t.NewMkLines("filename.mk",
		MkCvsID,
		"LEFT.$p=\t${RIGHT}",
		".for i in ${LEFT.$p}",
		".if ${left}",
		"\t${COMMAND}",
		"${TARGET}: ${SOURCE}",
		".include \"${DIR}/Makefile\"")

	mklines.ForEach(func(mkline *MkLine) {
		action := func(expr *MkExpr, time EctxTime) {
			mkline.Notef("Expression \"%s\" at \"%s\" time.", expr.String(), time.String())
		}
		walker := NewMkWalker(mkline, action)
		walker.WalkLine(mkline)
	})

	t.CheckOutputLines(
		"WARN: filename.mk:2: $p is ambiguous. Use ${p} if you mean a Make variable or $$p if you mean a shell variable.",
		"NOTE: filename.mk:2: Expression \"${p}\" at \"load\" time.",
		"NOTE: filename.mk:2: Expression \"${RIGHT}\" at \"run\" time.",
		"NOTE: filename.mk:3: Expression \"${LEFT.$p}\" at \"load\" time.",
		"NOTE: filename.mk:3: Expression \"${p}\" at \"load\" time.",
		"NOTE: filename.mk:4: Expression \"${left}\" at \"load\" time.",
		"NOTE: filename.mk:5: Expression \"${COMMAND}\" at \"run\" time.",
		"NOTE: filename.mk:6: Expression \"${TARGET}\" at \"load\" time.",
		"NOTE: filename.mk:6: Expression \"${SOURCE}\" at \"load\" time.",
		"NOTE: filename.mk:7: Expression \"${DIR}\" at \"load\" time.")
}

func (s *Suite) Test_MkWalker_WalkText(c *check.C) {
	t := s.Init(c)

	mklines := t.NewMkLines("filename.mk",
		MkCvsID,
		"\t${COMMAND}")

	mklines.ForEach(func(mkline *MkLine) {
		action := func(expr *MkExpr, time EctxTime) {
			mkline.Notef("Expression \"%s\" at \"%s\" time.", expr.String(), time.String())
		}
		walker := NewMkWalker(mkline, action)
		walker.WalkLine(mkline)
	})

	t.CheckOutputLines(
		"NOTE: filename.mk:2: Expression \"${COMMAND}\" at \"run\" time.")
}

func (s *Suite) Test_MkWalker_walkDirective(c *check.C) {
	t := s.Init(c)

	mklines := t.NewMkLines("filename.mk",
		".info ${INFO:S<from$<to<}",
		".if ${COND:S<from$<to<}",
		".endif")

	mklines.ForEach(func(mkline *MkLine) {
		action := func(expr *MkExpr, time EctxTime) {
			mkline.Notef("Expression \"%s\" at \"%s\" time.", expr.String(), time.String())
		}
		walker := NewMkWalker(mkline, action)
		walker.walkDirective(mkline)
	})

	t.CheckOutputLines(
		"NOTE: filename.mk:1: Expression \"${INFO:S<from$<to<}\" at \"load\" time.",
		"NOTE: filename.mk:2: Expression \"${COND:S<from$<to<}\" at \"load\" time.")
}

func (s *Suite) Test_MkWalker_walkExpr(c *check.C) {
	t := s.Init(c)

	mklines := t.NewMkLines("filename.mk",
		"${LEFT:S<from$<to<}=\t${RIGHT:S<from$<to<}")

	mklines.ForEach(func(mkline *MkLine) {
		action := func(expr *MkExpr, time EctxTime) {
			mkline.Notef("Expression \"%s\" at \"%s\" time.", expr.String(), time.String())
		}
		walker := NewMkWalker(mkline, action)
		walker.WalkLine(mkline)
	})

	t.CheckOutputLines(
		"NOTE: filename.mk:1: Expression \"${LEFT:S<from$<to<}\" at \"load\" time.",
		"NOTE: filename.mk:1: Expression \"${RIGHT:S<from$<to<}\" at \"run\" time.")
}

func (s *Suite) Test_MkWalker_walkModifier(c *check.C) {
	t := s.Init(c)

	mklines := t.NewMkLines("filename.mk",
		"MOD.S=\t${VAR:S<from$<to<}",
		"MOD.S=\t${VAR:S<${from}$<${to}<}",
	)

	mklines.ForEach(func(mkline *MkLine) {
		action := func(expr *MkExpr, time EctxTime) {
			mkline.Notef("Expression \"%s\" at \"%s\" time.", expr.String(), time.String())
		}
		walker := NewMkWalker(mkline, action)
		walker.WalkLine(mkline)
	})

	t.CheckOutputLines(
		"NOTE: filename.mk:1: Expression \"${VAR:S<from$<to<}\" at \"run\" time.",
		"NOTE: filename.mk:2: Expression \"${VAR:S<${from}$<${to}<}\" at \"run\" time.",
		"NOTE: filename.mk:2: Expression \"${from}\" at \"run\" time.",
		"NOTE: filename.mk:2: Expression \"${to}\" at \"run\" time.")
}
