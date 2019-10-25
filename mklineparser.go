package pkglint

import (
	"netbsd.org/pkglint/textproc"
	"strings"
)

type MkLineParser struct{}

// Parse parses the text of a Makefile line to see what kind of line
// it is: variable assignment, include, comment, etc.
//
// See devel/bmake/parse.c:/^Parse_File/
func (p MkLineParser) Parse(line *Line) *MkLine {
	text := line.Text

	// XXX: This check should be moved somewhere else. NewMkLine should only be concerned with parsing.
	if hasPrefix(text, " ") && line.Basename != "bsd.buildlink3.mk" {
		line.Warnf("Makefile lines should not start with space characters.")
		line.Explain(
			"If this line should be a shell command connected to a target, use a tab character for indentation.",
			"Otherwise remove the leading whitespace.")
	}

	// Check for shell commands first because these cannot have comments
	// at the end of the line.
	if hasPrefix(text, "\t") {
		lex := textproc.NewLexer(text)
		for lex.SkipByte('\t') {
		}

		// Just for the side effects of the warnings.
		_ = p.split(line, lex.Rest())

		return p.parseShellcmd(line)
	}

	data := p.split(line, text)

	if mkline := p.parseVarassign(line); mkline != nil {
		return mkline
	}
	if mkline := p.parseCommentOrEmpty(line); mkline != nil {
		return mkline
	}
	if mkline := p.parseDirective(line, data); mkline != nil {
		return mkline
	}
	if mkline := p.parseInclude(line); mkline != nil {
		return mkline
	}
	if mkline := p.parseSysinclude(line); mkline != nil {
		return mkline
	}
	if mkline := p.parseDependency(line); mkline != nil {
		return mkline
	}
	if mkline := p.parseMergeConflict(line); mkline != nil {
		return mkline
	}

	// The %q is deliberate here since it shows possible strange characters.
	line.Errorf("Unknown Makefile line format: %q.", text)
	return &MkLine{line, nil}
}

func (p MkLineParser) parseVarassign(line *Line) *MkLine {
	m, a := p.MatchVarassign(line, line.Text)
	if !m {
		return nil
	}

	if a.spaceAfterVarname != "" {
		varname := a.varname
		op := a.op
		switch {
		case hasSuffix(varname, "+") && (op == opAssign || op == opAssignAppend):
			break
		case matches(varname, `^[a-z]`) && op == opAssignEval:
			break
		default:
			fix := line.Autofix()
			fix.Notef("Unnecessary space after variable name %q.", varname)
			fix.Replace(varname+a.spaceAfterVarname+op.String(), varname+op.String())
			fix.Apply()
		}
	}

	if a.comment != "" && a.value != "" && a.spaceAfterValue == "" {
		line.Warnf("The # character starts a Makefile comment.")
		line.Explain(
			"In a variable assignment, an unescaped # starts a comment that",
			"continues until the end of the line.",
			"To escape the #, write \\#.",
			"",
			"If this # character intentionally starts a comment,",
			"it should be preceded by a space in order to make it more visible.")
	}

	return &MkLine{line, a}
}

func (p MkLineParser) MatchVarassign(line *Line, text string) (bool, *mkLineAssign) {

	// A commented variable assignment does not have leading whitespace.
	// Otherwise line 1 of almost every Makefile fragment would need to
	// be scanned for a variable assignment even though it only contains
	// the $NetBSD CVS Id.
	clex := textproc.NewLexer(text)
	commented := clex.SkipByte('#')
	if commented && clex.SkipHspace() || clex.EOF() {
		return false, nil
	}

	withoutLeadingComment := text
	if commented {
		withoutLeadingComment = withoutLeadingComment[1:]
	}

	data := p.split(nil, withoutLeadingComment)

	lexer := NewMkTokensLexer(data.tokens)
	mainStart := lexer.Mark()

	for !commented && lexer.SkipByte(' ') {
	}

	varnameStart := lexer.Mark()
	// TODO: duplicated code in MkParser.Varname
	for lexer.NextBytesSet(VarbaseBytes) != "" || lexer.NextVarUse() != nil {
	}
	if lexer.SkipByte('.') || hasPrefix(data.main, "SITES_") {
		for lexer.NextBytesSet(VarparamBytes) != "" || lexer.NextVarUse() != nil {
		}
	}

	varname := lexer.Since(varnameStart)

	if varname == "" {
		return false, nil
	}

	spaceAfterVarname := lexer.NextHspace()

	opStart := lexer.Mark()
	switch lexer.PeekByte() {
	case '!', '+', ':', '?':
		lexer.Skip(1)
	}
	if !lexer.SkipByte('=') {
		return false, nil
	}
	op := NewMkOperator(lexer.Since(opStart))

	if hasSuffix(varname, "+") && op == opAssign && spaceAfterVarname == "" {
		varname = varname[:len(varname)-1]
		op = opAssignAppend
	}

	lexer.SkipHspace()

	value := trimHspace(lexer.Rest())
	parsedValueAlign := condStr(commented, "#", "") + lexer.Since(mainStart)
	valueAlign := p.getRawValueAlign(line.raw[0].orignl, parsedValueAlign)
	spaceBeforeComment := data.spaceBeforeComment
	if value == "" {
		valueAlign += spaceBeforeComment
		spaceBeforeComment = ""
	}

	return true, &mkLineAssign{
		commented:         commented,
		varname:           varname,
		varcanon:          varnameCanon(varname),
		varparam:          varnameParam(varname),
		spaceAfterVarname: spaceAfterVarname,
		op:                op,
		valueAlign:        valueAlign,
		value:             value,
		valueMk:           nil, // filled in lazily
		valueMkRest:       "",  // filled in lazily
		fields:            nil, // filled in lazily
		spaceAfterValue:   spaceBeforeComment,
		comment:           condStr(data.hasComment, "#", "") + data.comment,
	}
}

func (p MkLineParser) parseShellcmd(line *Line) *MkLine {
	return &MkLine{line, mkLineShell{line.Text[1:]}}
}

func (p MkLineParser) parseCommentOrEmpty(line *Line) *MkLine {
	trimmedText := trimHspace(line.Text)

	if strings.HasPrefix(trimmedText, "#") {
		return &MkLine{line, mkLineComment{}}
	}

	if trimmedText == "" {
		return &MkLine{line, mkLineEmpty{}}
	}

	return nil
}

func (p MkLineParser) parseDirective(line *Line, data mkLineSplitResult) *MkLine {
	text := line.Text
	if !hasPrefix(text, ".") {
		return nil
	}

	lexer := textproc.NewLexer(data.main[1:])

	indent := lexer.NextHspace()
	directive := lexer.NextBytesSet(LowerDash)
	switch directive {
	case "if", "else", "elif", "endif",
		"ifdef", "ifndef",
		"for", "endfor", "undef",
		"error", "warning", "info",
		"export", "export-env", "unexport", "unexport-env":
		break
	default:
		// Intentionally not supported are: ifmake ifnmake elifdef elifndef elifmake elifnmake.
		return nil
	}

	lexer.SkipHspace()

	args := lexer.Rest()

	// In .if and .endif lines the space surrounding the comment is irrelevant.
	// Especially for checking that the .endif comment matches the .if condition,
	// it must be trimmed.
	trimmedComment := trimHspace(data.comment)

	return &MkLine{line, &mkLineDirective{indent, directive, args, trimmedComment, nil, nil, nil}}
}

func (p MkLineParser) parseInclude(line *Line) *MkLine {
	m, indent, directive, includedFile, comment := MatchMkInclude(line.Text)
	if !m {
		return nil
	}

	return &MkLine{line, &mkLineInclude{directive == "include", false, indent, includedFile, nil, comment}}
}

func (p MkLineParser) parseSysinclude(line *Line) *MkLine {
	m, indent, directive, includedFile, comment := match4(line.Text, `^\.([\t ]*)(s?include)[\t ]+<([^>]+)>[\t ]*(#.*)?$`)
	if !m {
		return nil
	}

	return &MkLine{line, &mkLineInclude{directive == "include", true, indent, includedFile, nil, strings.TrimPrefix(comment, "#")}}
}

func (p MkLineParser) parseDependency(line *Line) *MkLine {
	// XXX: Replace this regular expression with proper parsing.
	// There might be a ${VAR:M*.c} in these variables, which the below regular expression cannot handle.
	m, targets, whitespace, sources := match3(line.Text, `^([^\t :]+(?:[\t ]*[^\t :]+)*)([\t ]*):[\t ]*([^#]*?)(?:[\t ]*#.*)?$`)
	if !m {
		return nil
	}

	if whitespace != "" {
		line.Notef("Space before colon in dependency line.")
	}
	return &MkLine{line, mkLineDependency{targets, sources}}
}

func (p MkLineParser) parseMergeConflict(line *Line) *MkLine {
	if !matches(line.Text, `^(<<<<<<<|=======|>>>>>>>)`) {
		return nil
	}

	return &MkLine{line, nil}
}

// split parses a logical line from a Makefile (that is, after joining
// the lines that end in a backslash) into two parts: the main part and the
// comment.
//
// This applies to all line types except those starting with a tab, which
// contain the shell commands to be associated with make targets. These cannot
// have comments.
//
// If line is given, it is used for logging parse errors and warnings
// about round parentheses instead of curly braces, as well as ambiguous
// variables of the form $v instead of ${v}.
func (MkLineParser) split(line *Line, text string) mkLineSplitResult {
	assert(!hasPrefix(text, "\t"))

	mainWithSpaces, comment := MkLineParser{}.unescapeComment(text)

	parser := NewMkParser(line, mainWithSpaces)
	lexer := parser.lexer

	parseOther := func() string {
		var sb strings.Builder

		for !lexer.EOF() {
			if lexer.SkipString("$$") {
				sb.WriteString("$$")
				continue
			}

			other := lexer.NextBytesFunc(func(b byte) bool { return b != '$' })
			if other == "" {
				break
			}

			sb.WriteString(other)
		}

		return sb.String()
	}

	var tokens []*MkToken
	for !lexer.EOF() {
		mark := lexer.Mark()

		if varUse := parser.VarUse(); varUse != nil {
			tokens = append(tokens, &MkToken{lexer.Since(mark), varUse})

		} else if other := parseOther(); other != "" {
			tokens = append(tokens, &MkToken{other, nil})

		} else {
			assert(lexer.SkipByte('$'))
			tokens = append(tokens, &MkToken{"$", nil})
		}
	}

	hasComment := comment != ""
	if hasComment {
		comment = comment[1:]
	}

	mainTrimmed := rtrimHspace(mainWithSpaces)
	spaceBeforeComment := mainWithSpaces[len(mainTrimmed):]
	if spaceBeforeComment != "" {
		tokenText := &tokens[len(tokens)-1].Text
		*tokenText = rtrimHspace(*tokenText)
		if *tokenText == "" {
			if len(tokens) == 1 {
				tokens = nil
			} else {
				tokens = tokens[:len(tokens)-1]
			}
		}
	}

	return mkLineSplitResult{mainTrimmed, tokens, spaceBeforeComment, hasComment, comment}
}

// unescapeComment takes a Makefile line, as written in a file, and splits
// it into the main part and the comment.
//
// The comment starts at the first #. Except if it is preceded by an odd number
// of backslashes. Or by an opening bracket.
//
// The main text is returned including leading and trailing whitespace. Any
// escaped # is returned in its unescaped form, that is, \# becomes #.
//
// The comment is returned including the leading "#", if any. If the line has
// no comment, it is an empty string.
func (MkLineParser) unescapeComment(text string) (main, comment string) {
	var sb strings.Builder

	lexer := textproc.NewLexer(text)

again:
	if plain := lexer.NextBytesSet(unescapeMkCommentSafeChars); plain != "" {
		sb.WriteString(plain)
		goto again
	}

	switch {
	case lexer.SkipString("\\#"):
		sb.WriteByte('#')

	case lexer.PeekByte() == '\\' && len(lexer.Rest()) >= 2:
		sb.WriteString(lexer.Rest()[:2])
		lexer.Skip(2)

	case lexer.SkipByte('\\'):
		sb.WriteByte('\\')

	case lexer.SkipString("[#"):
		// See devel/bmake/files/parse.c:/as in modifier/
		sb.WriteString("[#")

	case lexer.SkipByte('['):
		sb.WriteByte('[')

	default:
		main = sb.String()
		if lexer.PeekByte() == '#' {
			return main, lexer.Rest()
		}

		assert(lexer.EOF())
		return main, ""
	}

	goto again
}

func (MkLineParser) getRawValueAlign(raw, parsed string) string {
	r := textproc.NewLexer(raw)
	p := textproc.NewLexer(parsed)
	mark := r.Mark()

	for !p.EOF() {
		pch := p.PeekByte()
		rch := r.PeekByte()

		switch {
		case pch == rch:
			p.Skip(1)
			r.Skip(1)

		case pch == ' ', pch == '\t':
			p.SkipHspace()
			r.SkipHspace()

		default:
			assert(pch == '#')
			assert(r.SkipString("\\#"))
			p.Skip(1)
		}
	}

	return r.Since(mark)
}

type mkLineSplitResult struct {
	// The text of the line, without the comment at the end of the line,
	// and with # signs unescaped.
	main               string
	tokens             []*MkToken
	spaceBeforeComment string
	hasComment         bool
	comment            string
}
