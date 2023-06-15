package pkglint

// MkStmt provides a view on a makefile that focuses on the conditions and
// loops and their nesting.
type MkStmt interface {
	isMkStmt()
}

// MkStmtLine is a single makefile lines that is neither a condition nor a
// loop.
type MkStmtLine struct {
	Line *MkLine
}

// MkStmtBlock groups regular lines and nested conditions or loops.
type MkStmtBlock struct {
	Stmts []MkStmt
}

// MkStmtCond is a single level of .if/.elif/.else/.endif.
type MkStmtCond struct {
	Conds    []*MkLine
	Branches []*MkStmtBlock
}

// MkStmtLoop is a single level of a .for loop.
type MkStmtLoop struct {
	Head *MkLine
	Body *MkStmtBlock
}

type MkStmtCallback struct {
	Line  func(line *MkLine)
	Block func(block *MkStmtBlock)
	Cond  func(cond *MkStmtCond)
	Loop  func(loop *MkStmtLoop)
}

func ParseMkStmts(mklines *MkLines) MkStmt {

	kind := func(mkline *MkLine) string {
		if mkline.IsDirective() {
			dir := mkline.Directive()
			if hasPrefix(dir, "if") {
				return "if"
			}
			if hasPrefix(dir, "elif") {
				return "elif"
			}
			switch dir {
			case "else", "endif", "for", "endfor":
				return dir
			}
		}
		return ""
	}

	stack := []MkStmt{&MkStmtBlock{}}

	appendStmt := func(stmt MkStmt) {
		switch top := stack[len(stack)-1].(type) {
		case *MkStmtBlock:
			top.Stmts = append(top.Stmts, stmt)
		case *MkStmtCond:
			branch := top.Branches[len(top.Branches)-1]
			branch.Stmts = append(branch.Stmts, stmt)
		case *MkStmtLoop:
			top.Body.Stmts = append(top.Body.Stmts, stmt)
		}
	}

	for _, mkline := range mklines.mklines {
		kind := kind(mkline)
		switch kind {
		case "if":
			var cond MkStmtCond
			cond.Conds = append(cond.Conds, mkline)
			cond.Branches = append(cond.Branches, &MkStmtBlock{})
			stack = append(stack, &cond)
		case "elif":
			cond := stack[len(stack)-1].(*MkStmtCond)
			if len(cond.Conds) != len(cond.Branches) {
				return nil
			}
			cond.Conds = append(cond.Conds, mkline)
			cond.Branches = append(cond.Branches, &MkStmtBlock{})
		case "else":
			cond := stack[len(stack)-1].(*MkStmtCond)
			if len(cond.Conds) != len(cond.Branches) {
				return nil
			}
			cond.Branches = append(cond.Branches, &MkStmtBlock{})
		case "endif":
			cond := stack[len(stack)-1].(*MkStmtCond)
			stack = stack[:len(stack)-1]
			appendStmt(cond)
		case "for":
			stack = append(stack, &MkStmtLoop{mkline, &MkStmtBlock{}})
		case "endfor":
			loop := stack[len(stack)-1].(*MkStmtLoop)
			stack = stack[:len(stack)-1]
			appendStmt(loop)
		default:
			appendStmt(&MkStmtLine{mkline})
		}
	}
	if len(stack) != 1 {
		return nil
	}
	return stack[0]
}

func WalkMkStmt(stmt MkStmt, callback MkStmtCallback) {
	switch stmt := stmt.(type) {
	case *MkStmtLine:
		if callback.Line != nil {
			callback.Line(stmt.Line)
		}
	case *MkStmtBlock:
		if callback.Block != nil {
			callback.Block(stmt)
		}
		for _, blockStmt := range stmt.Stmts {
			WalkMkStmt(blockStmt, callback)
		}
	case *MkStmtCond:
		if callback.Cond != nil {
			callback.Cond(stmt)
		}
		for i, branch := range stmt.Branches {
			if i < len(stmt.Conds) && callback.Line != nil {
				callback.Line(stmt.Conds[i])
			}
			for _, branchStmt := range branch.Stmts {
				WalkMkStmt(branchStmt, callback)
			}
		}
	case *MkStmtLoop:
		if callback.Loop != nil {
			callback.Loop(stmt)
		}
		if callback.Line != nil {
			callback.Line(stmt.Head)
		}
		for _, bodyStmt := range stmt.Body.Stmts {
			WalkMkStmt(bodyStmt, callback)
		}
	}
}

func (*MkStmtLine) isMkStmt()  {}
func (*MkStmtBlock) isMkStmt() {}
func (*MkStmtCond) isMkStmt()  {}
func (*MkStmtLoop) isMkStmt()  {}
