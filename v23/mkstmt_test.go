package pkglint

import "gopkg.in/check.v1"

func (s *Suite) Test_ParseMkStmts(c *check.C) {
	t := s.Init(c)
	mklines := t.NewMkLines("filename.mk",
		MkCvsID,
		".if 1",
		".info one",
		".elif 2",
		".else",
		".  for i in value",
		".  endfor",
		".endif")

	stmts := ParseMkStmts(mklines)

	t.CheckDeepEquals(stmts,
		&MkStmtBlock{[]MkStmt{
			&MkStmtLine{mklines.mklines[0]},
			&MkStmtCond{
				[]*MkLine{
					mklines.mklines[1],
					mklines.mklines[3],
				},
				[]*MkStmtBlock{
					{[]MkStmt{
						&MkStmtLine{mklines.mklines[2]},
					}},
					{},
					{[]MkStmt{
						&MkStmtLoop{
							mklines.mklines[5],
							&MkStmtBlock{nil},
						},
					}},
				},
			},
		}})
}

func (s *Suite) Test_WalkMkStmt(c *check.C) {
	t := s.Init(c)
	mklines := t.NewMkLines("filename.mk",
		MkCvsID,
		".if 1",
		".info one",
		".elif 2",
		".else",
		".  for i in value",
		".  endfor",
		".endif")

	line := NewLineWhole("")
	stmts := ParseMkStmts(mklines)
	if stmts != nil {
		WalkMkStmt(stmts, MkStmtCallback{
			Line: func(mkline *MkLine) {
				mkline.Notef("Line.")
			},
			Block: func(block *MkStmtBlock) {
				line.Notef("Block with %d statements.", len(block.Stmts))
			},
			Cond: func(cond *MkStmtCond) {
				line.Notef("Cond with %d conditions and %d branches.",
					len(cond.Conds), len(cond.Branches))
			},
			Loop: func(loop *MkStmtLoop) {
				line.Notef("Loop with head %q and %d body statements.",
					loop.Head.Args(), len(loop.Body.Stmts))
			},
		})
	}
	WalkMkStmt(stmts, MkStmtCallback{})

	t.CheckOutputLines(
		"NOTE: Block with 2 statements.",
		"NOTE: filename.mk:1: Line.",
		"NOTE: Cond with 2 conditions and 3 branches.",
		"NOTE: filename.mk:2: Line.",
		"NOTE: filename.mk:3: Line.",
		"NOTE: filename.mk:4: Line.",
		"NOTE: Loop with head \"i in value\" and 0 body statements.",
		"NOTE: filename.mk:6: Line.")
}

func (s *Suite) Test_WalkMkStmt__invalid(c *check.C) {
	t := s.Init(c)

	test := func(lines ...string) {
		mklines := t.NewMkLines("filename.mk", lines...)
		stmts := ParseMkStmts(mklines)
		t.CheckNil(stmts)
	}

	// '.if' without '.endif'
	test(MkCvsID,
		".if 1")

	// '.elif' without '.if'
	test(MkCvsID,
		".elif 2")

	// '.else' without '.if'
	test(MkCvsID,
		".else")

	// '.endif' without '.if
	test(MkCvsID,
		".endif")

	// '.for' without '.endfor'
	test(MkCvsID,
		".for i in value")

	// '.endfor' without '.for'
	test(MkCvsID,
		".endfor")

	// '.if' ended by '.endfor'
	test(MkCvsID,
		".if 1",
		".endfor")

	// '.for' ended by '.endif'
	test(MkCvsID,
		".for i in value",
		".endif")
}
