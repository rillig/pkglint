package main

import (
	"fmt"
	"netbsd.org/pkglint/textproc"
)

func CheckdirCategory(dir string) {
	if trace.Tracing {
		defer trace.Call1(dir)()
	}

	mklines := LoadMk(dir+"/Makefile", NotEmpty|LogErrors)
	if mklines == nil {
		return
	}

	mklines.Check()

	exp := NewMkExpecter(mklines)
	for exp.AdvanceIfPrefix("#") {
	}
	exp.ExpectEmptyLine()

	if exp.AdvanceIf(func(mkline MkLine) bool { return mkline.IsVarassign() && mkline.Varname() == "COMMENT" }) {
		mkline := exp.PreviousMkLine()

		lex := textproc.NewLexer(mkline.Value())
		valid := textproc.NewByteSet("--- '(),/0-9A-Za-z")
		invalid := valid.Inverse()
		uni := ""

		for !lex.EOF() {
			_ = lex.NextBytesSet(valid)
			ch := lex.NextByteSet(invalid)
			if ch != -1 {
				uni += fmt.Sprintf(" %U", ch)
			}
		}

		if uni != "" {
			mkline.Warnf("%s contains invalid characters (%s).", mkline.Varname(), uni[1:])
		}

	} else {
		exp.CurrentLine().Errorf("COMMENT= line expected.")
	}
	exp.ExpectEmptyLine()

	type subdir struct {
		name string
		line MkLine
	}

	// And now to the most complicated part of the category Makefiles,
	// the (hopefully) sorted list of SUBDIRs. The first step is to
	// collect the SUBDIRs in the Makefile and in the file system.

	fSubdirs := getSubdirs(dir)
	var mSubdirs []subdir

	seen := make(map[string]MkLine)
	for !exp.EOF() {
		mkline := exp.CurrentMkLine()

		if (mkline.IsVarassign() || mkline.IsCommentedVarassign()) && mkline.Varname() == "SUBDIR" {
			exp.Advance()

			name := mkline.Value()
			if mkline.IsCommentedVarassign() && mkline.VarassignComment() == "" {
				mkline.Warnf("%q commented out without giving a reason.", name)
			}

			if prev := seen[name]; prev != nil {
				mkline.Errorf("%q must only appear once, already seen in %s.", name, prev.ReferenceFrom(mkline.Line))
			}
			seen[name] = mkline

			if len(mSubdirs) > 0 {
				if prev := mSubdirs[len(mSubdirs)-1].name; name < prev {
					mkline.Warnf("%q should come before %q.", name, prev)
				}
			}

			mSubdirs = append(mSubdirs, subdir{name, mkline})

		} else {
			if !mkline.IsEmpty() {
				mkline.Errorf("SUBDIR+= line or empty line expected.")
			}
			break
		}
	}

	// To prevent unnecessary warnings about subdirectories that are
	// in one list but not in the other, generate the sets of
	// subdirs of each list.
	fCheck := make(map[string]bool)
	mCheck := make(map[string]bool)
	for _, fsub := range fSubdirs {
		fCheck[fsub] = true
	}
	for _, msub := range mSubdirs {
		mCheck[msub.name] = true
	}

	fRest := fSubdirs[:]
	mRest := mSubdirs[:]
	mAtend, mNeednext, mCurrent := false, true, ""

	var subdirs []string

	var line MkLine
	commented := false

	for !(mAtend && len(fRest) == 0) {
		if !mAtend && mNeednext {
			mNeednext = false
			if len(mRest) == 0 {
				mAtend = true
				line = exp.CurrentMkLine()
				continue
			} else {
				mCurrent = mRest[0].name
				line = mRest[0].line
				commented = mRest[0].line.IsCommentedVarassign()
				mRest = mRest[1:]
			}
		}

		if len(fRest) > 0 && (mAtend || fRest[0] < mCurrent) {
			fCurrent := fRest[0]
			if !mCheck[fCurrent] {
				fix := line.Autofix()
				fix.Errorf("%q exists in the file system, but not in the Makefile.", fCurrent)
				fix.InsertBefore("SUBDIR+=\t" + fCurrent)
				fix.Apply()
			}
			fRest = fRest[1:]

		} else if !mAtend && (len(fRest) == 0 || mCurrent < fRest[0]) {
			if !fCheck[mCurrent] {
				fix := line.Autofix()
				fix.Errorf("%q exists in the Makefile, but not in the file system.", mCurrent)
				fix.Delete()
				fix.Apply()
			}
			mNeednext = true

		} else { // f_current == m_current
			fRest = fRest[1:]
			mNeednext = true
			if !commented {
				subdirs = append(subdirs, dir+"/"+mCurrent)
			}
		}
	}

	// the pkgsrc-wip category Makefile defines its own targets for
	// generating indexes and READMEs. Just skip them.
	if !G.Wip {
		exp.ExpectEmptyLine()
		exp.ExpectText(".include \"../mk/misc/category.mk\"")
		if !exp.EOF() {
			exp.CurrentLine().Errorf("The file should end here.")
		}
	}

	mklines.SaveAutofixChanges()

	if G.Opts.Recursive {
		G.Todo = append(append([]string(nil), subdirs...), G.Todo...)
	}
}
