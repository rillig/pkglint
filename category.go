package main

import (
	"sort"
)

type Subdir struct {
	subdir string
	line   *Line
	active bool
}

func checkdirCategory() {
	trace("checkdirCategory", G.currentDir)

	fname := G.currentDir + "/Makefile"
	lines := loadNonemptyLines(fname, true)
	if lines == nil {
		return
	}
	parselinesMk(lines)

	exp := &ExpectContext{lines, 0}
	if checklineRcsid(exp.currentLine(), `#\s+`, "# ") {
		exp.advance()
	}

	for !exp.eof() && exp.advanceIfMatches(`^#`) != nil {
	}
	exp.expectEmptyLine()

	if exp.advanceIfMatches(`^COMMENT=\t*(.*)`) != nil {
		checklineValidCharactersInValue(exp.previousLine(), `[-\040'(),/0-9A-Za-z]`)
	} else {
		exp.currentLine().logError("COMMENT= line expected.")
	}
	exp.expectEmptyLine()

	// And now to the most complicated part of the category Makefiles,
	// the (hopefully) sorted list of SUBDIRs. The first step is to
	// collect the SUBDIRs in the Makefile and in the file system.

	fSubdirs := getSubdirs(G.currentDir)
	sort.Sort(sort.StringSlice(fSubdirs))
	mSubdirs := make([]Subdir, 0)

	prevSubdir := ""
	for !exp.eof() {
		line := exp.currentLine()
		text := line.text

		if m, commentFlag, indentation, subdir, comment := match4(text, `^(#?)SUBDIR\+=(\s*)(\S+)\s*(?:#\s*(.*?)\s*|)$`); m {
			commentedOut := commentFlag == "#"
			if commentedOut && comment == "" {
				line.logWarning("%q commented out without giving a reason.", subdir)
			}

			if indentation != "\t" {
				line.logWarning("Indentation should be a single tab character.")
			}

			if subdir == prevSubdir {
				line.logError("%q must only appear once.", subdir)
			} else if subdir < prevSubdir {
				line.logWarning("%q should come before %q.", subdir, prevSubdir)
			} else {
				// correctly ordered
			}

			mSubdirs = append(mSubdirs, Subdir{subdir, line, !commentedOut})
			prevSubdir = subdir
			exp.advance()

		} else {
			if line.text != "" {
				line.logError("SUBDIR+= line or empty line expected.")
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
		mCheck[msub.subdir] = true
	}

	f_index, f_atend, f_neednext, f_current := 0, false, true, ""
	m_index, m_atend, m_neednext, m_current := 0, false, true, ""

	subdirs := make([]string, 0)

	var line *Line
	m_active := false

	for !(m_atend && f_atend) {
		if !m_atend && m_neednext {
			m_neednext = false
			if m_index >= len(mSubdirs) {
				m_atend = true
				line = exp.currentLine()
				continue
			} else {
				m_current = mSubdirs[m_index].subdir
				line = mSubdirs[m_index].line
				m_active = mSubdirs[m_index].active
				m_index++
			}
		}

		if !f_atend && f_neednext {
			f_neednext = false
			if f_index >= len(fSubdirs) {
				f_atend = true
				continue
			} else {
				f_current = fSubdirs[f_index]
				f_index++
			}
		}

		if !f_atend && (m_atend || f_current < m_current) {
			if !mCheck[f_current] {
				line.logError("%q exists in the file system, but not in the Makefile.", f_current)
				line.appendBefore("SUBDIR+=\t" + f_current)
			}
			f_neednext = true

		} else if !m_atend && (f_atend || m_current < f_current) {
			if !fCheck[m_current] {
				line.logError("%q exists in the Makefile, but not in the file system.", m_current)
				line.delete()
			}
			m_neednext = true

		} else { // f_current == m_current
			f_neednext = true
			m_neednext = true
			if m_active {
				subdirs = append(subdirs, G.currentDir+"/"+m_current)
			}
		}
	}

	// the pkgsrc-wip category Makefile defines its own targets for
	// generating indexes and READMEs. Just skip them.
	if G.isWip {
		exp.index = len(exp.lines) - 2
	}

	exp.expectEmptyLine()

	if exp.currentLine().text == ".include \"../mk/bsd.pkg.subdir.mk\"" {
		exp.advance()
	} else {
		exp.expectText(".include \"../mk/misc/category.mk\"")
	}

	if !exp.eof() {
		exp.currentLine().logError("The file should end here.")
	}

	checklinesMk(lines)

	autofix(lines)

	if G.opts.optRecursive {
		G.todo = append(subdirs, G.todo...)
	}
}
