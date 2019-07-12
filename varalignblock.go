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
// Continuation lines are also aligned to the single-line assignments.
// There are two types of continuation lines: follow-lines and initial-lines.
//
//  FOLLOW_LINE= \
//          The value starts in the second line.
//
// The backslash in the first line is usually aligned to the other variables
// in the same paragraph. If the variable name is so long that it is an
// outlier, it may be indented using a single space, just like a single-line
// variable. In multi-line shell commands or AWK programs, the backslash is
// often indented to column 73, as are the backslashes from the follow-up
// lines, to act as a visual guideline.
//
// Since this type is often used for URLs or other long values, the first
// follow-up line may be indented using a single tab, even if the other
// variables in the paragraph are aligned further to the right. If the
// indentation is not a single tab, it must match the indentation of the
// other lines in the paragraph.
//
//  INITIAL_LINE=   The value starts in the first line \
//                  and continues in the second line.
//
// In lists or plain text, like in the INITIAL_LINE above, all values are
// aligned in the same column. Some variables also contain code, and in
// these variables, the line containing the first word defines how deep
// the follow-up lines must be indented at least.
//
//  SHELL_CMD=                                                              \
//          if ${PKG_ADMIN} pmatch ${PKGNAME} ${dependency}; then           \
//                  ${ECHO} yes;                                            \
//          else                                                            \
//                  ${ECHO} no;                                             \
//          fi
//
// In the continuation lines, each follow-up line is indented using at least
// one tab, to avoid confusing them with regular single-lines. This is
// especially true for CONFIGURE_ENV, since the environment variables are
// typically uppercase as well.
//
// TODO: An initial line has this form:
//  comment? varname+op space? value? space? comment? space? backslash?
//
// TODO: A follow-up line has the form:
//  comment? space? value? space? comment? space? backslash?
//
// TODO: The alignment checks are performed on the raw lines instead of
//  the logical lines, since this check is about the visual appearance
//  as opposed to the meaning of the variable assignment.
//
// FIXME: Implement each requirement from the above documentation.
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
	multiFollow bool

	// A multiline that is properly aligned, when seen in isolation.
	// See MkLine.IsMultiAligned.
	//
	// Just because a multiline is properly aligned on its own, this
	// does not automatically mean it is properly aligned in the surrounding
	// paragraph.
	multiAligned bool
}

type varalignSplitResult struct {
	// All of the following strings can be empty.

	leadingComment    string
	varnameOp         string
	spaceBeforeValue  string
	value             string
	spaceAfterValue   string
	trailingComment   string
	spaceAfterComment string
	continuation      string
}

func (va *VaralignBlock) Process(mkline *MkLine) {
	switch {
	case !G.Opts.WarnSpace:

	case mkline.IsEmpty():
		va.Finish()

	case mkline.IsVarassignMaybeCommented():
		va.processVarassign(mkline)

	case mkline.IsComment(), mkline.IsDirective():

	default:
		trace.Stepf("Skipping varalign block because of line %s", &mkline.Location)
		va.skip = true
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

	valueAlign := mkline.ValueAlign()
	varnameOp := strings.TrimRight(valueAlign, " \t")
	varnameOpWidth := tabWidth(varnameOp)
	space := valueAlign[len(varnameOp):]
	totalWidth := tabWidth(valueAlign)
	multiInitial := mkline.IsMultiline() && mkline.FirstLineContainsValue()
	multiFollow := mkline.IsMultiline() && !multiInitial && mkline.Value() != ""
	multiAligned := mkline.IsMultiline() && mkline.IsMultiAligned()

	va.infos = append(va.infos, &varalignBlockInfo{
		mkline, varnameOp, varnameOpWidth, space, totalWidth,
		multiInitial, multiFollow, multiAligned})
}

func (*VaralignBlock) split(textnl string, initial bool) varalignSplitResult {

	// See MkLineParser.unescapeComment for very similar code.

	p := NewMkParser(nil, textnl)
	lexer := p.lexer

	parseVarnameOp := func() string {
		if !initial {
			return ""
		}

		mark := lexer.Mark()
		_ = p.Varname()
		lexer.SkipHspace()
		ok, _ := p.Op()
		assert(ok)
		return lexer.Since(mark)
	}

	parseValue := func() (string, string) {
		mark := lexer.Mark()

		for !lexer.EOF() && lexer.PeekByte() != '#' && lexer.PeekByte() != '\n' {
			switch {
			case lexer.NextBytesSet(unescapeMkCommentSafeChars) != "",
				lexer.SkipString("[#"),
				lexer.SkipByte('['):
				break

			default:
				assert(lexer.SkipByte('\\'))
				if !lexer.EOF() {
					lexer.Skip(1)
				}
			}
		}

		valueSpace := lexer.Since(mark)
		value := rtrimHspace(valueSpace)
		space := valueSpace[len(value):]
		return value, space
	}

	parseComment := func() (string, string, string) {
		rest := lexer.Rest()

		newline := len(rest)
		for newline > 0 && rest[newline-1] == '\n' {
			newline--
		}

		backslash := newline
		for backslash > 0 && rest[backslash-1] == '\\' {
			backslash--
		}

		if (newline-backslash)%2 == 1 {
			continuation := rest[backslash:]
			commentSpace := rest[:backslash]
			comment := rtrimHspace(commentSpace)
			space := commentSpace[len(comment):]
			return comment, space, continuation
		}

		return rest[:newline], "", rest[newline:]
	}

	leadingComment := lexer.NextString("#")
	varnameOp := parseVarnameOp()
	spaceBeforeValue := lexer.NextHspace()
	value, spaceAfterValue := parseValue()
	trailingComment, spaceAfterComment, continuation := parseComment()

	return varalignSplitResult{
		leadingComment,
		varnameOp,
		spaceBeforeValue,
		value,
		spaceAfterValue,
		trailingComment,
		spaceAfterComment,
		continuation,
	}
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
		va.realign(info.mkline, info.varnameOp, info.space, info.multiFollow, newWidth)
	}
}

// optimalWidth computes the desired screen width for the variable assignment
// lines. If the paragraph is already indented consistently, it is kept as-is.
//
// There may be a single line sticking out from the others (called outlier).
// This is to prevent a single SITES.* variable from forcing the rest of the
// paragraph to be indented too far to the right.
func (*VaralignBlock) optimalWidth(infos []*varalignBlockInfo) int {
	longest := 0 // The longest seen varnameOpWidth
	var longestLine *MkLine
	secondLongest := 0 // The second-longest seen varnameOpWidth
	for _, info := range infos {
		if info.multiFollow {
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
		if info.multiFollow {
			continue
		}

		if info.varnameOpWidth != outlier {
			width := info.totalWidth
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
	oldWidth := tabWidth(varnameOp + oldSpace)
	if oldWidth == 72 {
		return
	}

	wrongColumn := oldWidth != tabWidth(varnameOp+newSpace)

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
