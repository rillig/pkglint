package main

import (
	"fmt"
	"netbsd.org/pkglint/textproc"
	"sort"
	"strings"
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
		name   string
		line   MkLine
		active bool
	}

	// And now to the most complicated part of the category Makefiles,
	// the (hopefully) sorted list of SUBDIRs. The first step is to
	// collect the SUBDIRs in the Makefile and in the file system.

	fSubdirs := getSubdirs(dir)
	sort.Strings(fSubdirs)
	var mSubdirs []subdir

	prevSubdir := ""
	for !exp.EOF() {
		mkline := exp.CurrentMkLine()

		if (mkline.IsVarassign() || mkline.IsCommentedVarassign()) && mkline.Varname() == "SUBDIR" {
			name := mkline.Value()
			commentedOut := mkline.IsCommentedVarassign()
			if commentedOut && mkline.VarassignComment() == "" {
				mkline.Warnf("%q commented out without giving a reason.", name)
			}

			valueAlign := mkline.ValueAlign()
			indent := valueAlign[len(strings.TrimRightFunc(valueAlign, isHspaceRune)):]
			if indent != "\t" {
				mkline.Warnf("Indentation should be a single tab character.")
			}

			if name == prevSubdir {
				mkline.Errorf("%q must only appear once.", name)
			} else if name < prevSubdir {
				mkline.Warnf("%q should come before %q.", name, prevSubdir)
			} else {
				// correctly ordered
			}

			mSubdirs = append(mSubdirs, subdir{name, mkline, !commentedOut})
			prevSubdir = name
			exp.Advance()

		} else {
			if !mkline.IsEmpty() {
				mkline.Errorf("SUBDIR+= line or empty line expected.")
			}
			break
		}
	}

	// To prevent unnecessary warnings about subdirectories that are
	// in one list, but not in the other, we generate the sets of
	// subdirs of each list.
	fCheck := make(map[string]bool)
	mCheck := make(map[string]bool)
	for _, fsub := range fSubdirs {
		fCheck[fsub] = true
	}
	for _, msub := range mSubdirs {
		mCheck[msub.name] = true
	}

	fIndex, fAtend, fNeednext, fCurrent := 0, false, true, ""
	mIndex, mAtend, mNeednext, mCurrent := 0, false, true, ""

	var subdirs []string

	var line MkLine
	mActive := false

	for !(mAtend && fAtend) {
		if !mAtend && mNeednext {
			mNeednext = false
			if mIndex >= len(mSubdirs) {
				mAtend = true
				line = exp.CurrentMkLine()
				continue
			} else {
				mCurrent = mSubdirs[mIndex].name
				line = mSubdirs[mIndex].line
				mActive = mSubdirs[mIndex].active
				mIndex++
			}
		}

		if !fAtend && fNeednext {
			fNeednext = false
			if fIndex >= len(fSubdirs) {
				fAtend = true
				continue
			} else {
				fCurrent = fSubdirs[fIndex]
				fIndex++
			}
		}

		if !fAtend && (mAtend || fCurrent < mCurrent) {
			if !mCheck[fCurrent] {
				fix := line.Autofix()
				fix.Errorf("%q exists in the file system, but not in the Makefile.", fCurrent)
				fix.InsertBefore("SUBDIR+=\t" + fCurrent)
				fix.Apply()
			}
			fNeednext = true

		} else if !mAtend && (fAtend || mCurrent < fCurrent) {
			if !fCheck[mCurrent] {
				fix := line.Autofix()
				fix.Errorf("%q exists in the Makefile, but not in the file system.", mCurrent)
				fix.Delete()
				fix.Apply()
			}
			mNeednext = true

		} else { // f_current == m_current
			fNeednext = true
			mNeednext = true
			if mActive {
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
