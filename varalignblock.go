package pkglint

import "strings"

// VaralignBlock checks that all variable assignments from a paragraph
// use the same indentation depth for their values.
// It also checks that the indentation uses tabs instead of spaces.
//
// In general, all values should be aligned using tabs.
// As an exception, a single very long line (called an outlier) may be
// aligned with a single space.
// A typical example is a SITES.very-long-file-name.tar.gz variable
// between HOMEPAGE and DISTFILES.
//
// TODO: Document how exactly the continuation lines are handled.
type VaralignBlock struct {
	infos []*varalignBlockInfo
	skip  bool
}

type varalignBlockInfo struct {
	mkline         *MkLine
	varnameOp      string // Variable name + assignment operator
	varnameOpWidth int    // Screen width of varnameOp
	space          string // Whitespace between varnameOp and the variable value
	totalWidth     int    // Screen width of varnameOp + current space

	// The line is a multiline and the first actual value is in the initial line.
	//
	// Example:
	//  VAR=    value1 \
	//          value2
	multiInitial bool

	// The line is a multiline and the initial line is essentially empty.
	//
	// Example:
	//  VAR= \
	//          value1 \
	//          value2
	multiEmpty bool

	// A multiline that is properly aligned, when seen on its own. This means:
	//
	// For a multiEmpty line, the first actual value may be indented in column 8.
	// Otherwise the first actual value determines the indentation.
	// Each later continuation line must be at least as indented as the first actual value.
	multiAligned bool
}

func (va *VaralignBlock) Process(mkline *MkLine) {
	switch {
	case !G.Opts.WarnSpace:
		return

	case mkline.IsEmpty():
		va.Finish()
		return

	case mkline.IsVarassign(), mkline.IsCommentedVarassign():
		va.processVarassign(mkline)

	case mkline.IsComment(), mkline.IsDirective():
		return

	default:
		trace.Stepf("Skipping varalign block because of line %s", &mkline.Location)
		va.skip = true
		return
	}
}

func (va *VaralignBlock) processVarassign(mkline *MkLine) {
	switch {
	case mkline.Op() == opAssignEval && matches(mkline.Varname(), `^[a-z]`):
		// Arguments to procedures do not take part in block alignment.
		//
		// Example:
		// pkgpath := ${PKGPATH}
		// .include "../../mk/pkg-build-options.mk"
		return

	case mkline.Value() == "" && mkline.VarassignComment() == "":
		// Multiple-inclusion guards usually appear in a block of
		// their own and therefore do not need alignment.
		//
		// Example:
		// .if !defined(INCLUSION_GUARD_MK)
		// INCLUSION_GUARD_MK:=
		// # ...
		// .endif
		return
	}

	multiEarly := mkline.IsMultiline() && mkline.FirstLineContainsValue()
	multiLate := mkline.IsMultiline() && !multiEarly && mkline.Value() != ""
	multiAligned := mkline.IsMultiline() && mkline.IsMultiAligned()

	valueAlign := mkline.ValueAlign()
	varnameOp := strings.TrimRight(valueAlign, " \t")
	info := varalignBlockInfo{
		mkline,
		varnameOp,
		tabWidth(varnameOp),
		valueAlign[len(varnameOp):],
		tabWidth(valueAlign),
		multiEarly,
		multiLate,
		multiAligned}
	va.infos = append(va.infos, &info)
}

func (va *VaralignBlock) Finish() {
	infos := va.infos
	skip := va.skip
	*va = VaralignBlock{} // overwrites infos and skip

	if len(infos) == 0 || skip {
		return
	}

	if trace.Tracing {
		defer trace.Call(infos[0].mkline.Line)()
	}

	newWidth := va.optimalWidth(infos)
	if newWidth == 0 {
		return
	}

	for _, info := range infos {
		va.realign(info.mkline, info.varnameOp, info.space, info.multiEmpty, newWidth)
	}
}

// optimalWidth computes the desired screen width for the variable assignment
// lines. If the paragraph is already indented consistently, it is kept as-is.
//
// There may be a single line sticking out from the others (called outlier).
// This is to prevent a single SITES.* variable from forcing the rest of the
// paragraph to be indented too far to the right.
func (va *VaralignBlock) optimalWidth(infos []*varalignBlockInfo) int {
	longest := 0 // The longest seen varnameOpWidth
	var longestLine *MkLine
	secondLongest := 0 // The second-longest seen varnameOpWidth
	for _, info := range infos {
		if info.multiEmpty {
			// TODO: what if this info is never added to infos?
			continue
		}

		width := info.varnameOpWidth
		if width >= longest {
			secondLongest = longest
			longest = width
			longestLine = info.mkline
		} else if width > secondLongest {
			secondLongest = width
		}
	}

	haveOutlier := secondLongest != 0 &&
		longest/8 >= secondLongest/8+2 &&
		!longestLine.IsMultiline()
	// Minimum required width of varnameOp, without the trailing whitespace.
	minVarnameOpWidth := condInt(haveOutlier, secondLongest, longest)
	outlier := condInt(haveOutlier, longest, 0)

	// Widths of the current indentation (including whitespace)
	minTotalWidth := 0
	maxTotalWidth := 0
	for _, info := range infos {
		if info.multiEmpty {
			continue
		}

		if width := info.totalWidth; info.varnameOpWidth != outlier {
			if minTotalWidth == 0 || width < minTotalWidth {
				minTotalWidth = width
			}
			maxTotalWidth = imax(maxTotalWidth, width)
		}
	}

	if trace.Tracing {
		trace.Stepf("Indentation including whitespace is between %d and %d.",
			minTotalWidth, maxTotalWidth)
		trace.Stepf("Minimum required indentation is %d + 1.", minVarnameOpWidth)
		if outlier != 0 {
			trace.Stepf("The outlier is at indentation %d.", outlier)
		}
	}

	if minTotalWidth > minVarnameOpWidth && minTotalWidth == maxTotalWidth && minTotalWidth%8 == 0 {
		// The whole paragraph is already indented to the same width.
		return minTotalWidth
	}

	if minVarnameOpWidth == 0 {
		// Only continuation lines in this paragraph.
		return 0
	}

	return (minVarnameOpWidth & -8) + 8
}

func (va *VaralignBlock) realign(mkline *MkLine, varnameOp, oldSpace string, continuation bool, newWidth int) {
	hasSpace := contains(oldSpace, " ")

	newSpace := ""
	for tabWidth(varnameOp+newSpace) < newWidth {
		newSpace += "\t"
	}
	// Indent the outlier with a space instead of a tab to keep the line short.
	if newSpace == "" {
		if hasPrefix(oldSpace, "\t") {
			// Even though it is an outlier, it uses a tab and therefore
			// didn't seem to be too long to the original developer.
			// Therefore, leave it as-is but still fix any continuation lines.
			newSpace = oldSpace
		} else {
			newSpace = " "
		}
	}

	va.realignInitialLine(mkline, varnameOp, oldSpace, newSpace, hasSpace, newWidth)
	if mkline.IsMultiline() {
		va.realignContinuationLines(mkline, newWidth)
	}
}

func (va *VaralignBlock) realignInitialLine(mkline *MkLine, varnameOp string, oldSpace string, newSpace string, hasSpace bool, newWidth int) {
	wrongColumn := tabWidth(varnameOp+oldSpace) != tabWidth(varnameOp+newSpace)

	fix := mkline.Autofix()

	switch {
	case hasSpace && wrongColumn:
		fix.Notef("This variable value should be aligned with tabs, not spaces, to column %d.", newWidth+1)
	case hasSpace && oldSpace != newSpace:
		fix.Notef("Variable values should be aligned with tabs, not spaces.")
	case wrongColumn:
		fix.Notef("This variable value should be aligned to column %d.", newWidth+1)
	default:
		return
	}

	if wrongColumn {
		fix.Explain(
			"Normally, all variable values in a block should start at the same column.",
			"This provides orientation, especially for sequences",
			"of variables that often appear in the same order.",
			"For these it suffices to look at the variable values only.",
			"",
			"There are some exceptions to this rule:",
			"",
			"Definitions for long variable names may be indented with a single space instead of tabs,",
			"but only if they appear in a block that is otherwise indented using tabs.",
			"",
			"Variable definitions that span multiple lines are not checked for alignment at all.",
			"",
			"When the block contains something else than variable definitions",
			"and directives like .if or .for, it is not checked at all.")
	}

	fix.ReplaceAfter(varnameOp, oldSpace, newSpace)
	fix.Apply()
}

func (va *VaralignBlock) realignContinuationLines(mkline *MkLine, newWidth int) {
	indentation := strings.Repeat("\t", newWidth/8) + strings.Repeat(" ", newWidth%8)
	fix := mkline.Autofix()
	fix.Notef("This line should be aligned with %q.", indentation)
	fix.RealignContinuation(mkline, newWidth)
	fix.Apply()
}
