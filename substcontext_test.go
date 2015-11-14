package main

import (
	check "gopkg.in/check.v1"
)

func (s *Suite) TestSubstContext(c *check.C) {
	G.opts.optWarnExtra = true
	line := NewLine("Makefile", "1", "dummy", nil)
	ctx := new(SubstContext)

	ctx.checkVarassign(line, "PKGNAME", "=", "pkgname-1.0")

	c.Check(ctx.id, equals, "")

	ctx.checkVarassign(line, "SUBST_CLASSES", "+=", "interp")

	c.Check(ctx.id, equals, "interp")

	ctx.checkVarassign(line, "SUBST_FILES.interp", "=", "Makefile")

	c.Check(ctx.isComplete(), equals, false)

	ctx.checkVarassign(line, "SUBST_SED.interp", "=", "s,@PREFIX@,${PREFIX},g")

	c.Check(ctx.isComplete(), equals, true)

	ctx.finish(line)

	c.Check(s.Stdout(), equals, "WARN: Makefile:1: Incomplete SUBST block: SUBST_STAGE missing.\n")
}
