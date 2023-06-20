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
type MkStmtBlock []MkStmt

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
		if mkline.IsEmpty() || mkline.IsComment() {
			return "ignore"
		}
		return ""
	}

	stack := []MkStmt{&MkStmtBlock{}}

	appendStmt := func(stmt MkStmt) {
		switch top := stack[len(stack)-1].(type) {
		case *MkStmtBlock:
			*top = append(*top, stmt)
		case *MkStmtCond:
			branch := top.Branches[len(top.Branches)-1]
			*branch = append(*branch, stmt)
		case *MkStmtLoop:
			*top.Body = append(*top.Body, stmt)
		}
	}

	for _, mkline := range mklines.mklines {
		switch kind(mkline) {
		case "if":
			var cond MkStmtCond
			cond.Conds = append(cond.Conds, mkline)
			cond.Branches = append(cond.Branches, &MkStmtBlock{})
			stack = append(stack, &cond)
		case "elif":
			cond, ok := stack[len(stack)-1].(*MkStmtCond)
			if !ok {
				return nil
			}
			if len(cond.Conds) > 0 && kind(cond.Conds[len(cond.Conds)-1]) == "else" {
				return nil
			}
			cond.Conds = append(cond.Conds, mkline)
			cond.Branches = append(cond.Branches, &MkStmtBlock{})
		case "else":
			cond, ok := stack[len(stack)-1].(*MkStmtCond)
			if !ok {
				return nil
			}
			if len(cond.Conds) > 0 && kind(cond.Conds[len(cond.Conds)-1]) == "else" {
				return nil
			}
			cond.Conds = append(cond.Conds, mkline)
			cond.Branches = append(cond.Branches, &MkStmtBlock{})
		case "endif":
			cond, ok := stack[len(stack)-1].(*MkStmtCond)
			if !ok {
				return nil
			}
			stack = stack[:len(stack)-1]
			appendStmt(cond)
		case "for":
			stack = append(stack, &MkStmtLoop{mkline, &MkStmtBlock{}})
		case "endfor":
			loop, ok := stack[len(stack)-1].(*MkStmtLoop)
			if !ok {
				return nil
			}
			stack = stack[:len(stack)-1]
			appendStmt(loop)
		case "ignore":
			// Do nothing.
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
		for _, blockStmt := range *stmt {
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
			for _, branchStmt := range *branch {
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
		for _, bodyStmt := range *stmt.Body {
			WalkMkStmt(bodyStmt, callback)
		}
	}
}

func (*MkStmtLine) isMkStmt()  {}
func (*MkStmtBlock) isMkStmt() {}
func (*MkStmtCond) isMkStmt()  {}
func (*MkStmtLoop) isMkStmt()  {}
