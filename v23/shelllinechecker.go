package pkglint

import (
	"github.com/rillig/pkglint/v23/textproc"
	"strings"
)

// ShellLineChecker checks either a line from a makefile starting with a tab,
// thereby containing shell commands to be executed.
//
// Or it checks a variable assignment line from a makefile with a left-hand
// side variable that is of some shell-like type; see Vartype.IsShell.
type ShellLineChecker struct {
	MkLines *MkLines
	mkline  *MkLine

	// checkExpr is set to false when checking a single shell word
	// in order to skip duplicate warnings in variable assignments.
	checkExpr bool
}

func NewShellLineChecker(mklines *MkLines, mkline *MkLine) *ShellLineChecker {
	assertNotNil(mklines)
	return &ShellLineChecker{mklines, mkline, true}
}

// CheckShellCommands checks for a list of shell commands, of which each one
// is terminated with a semicolon. These are used in GENERATE_PLIST.
func (ck *ShellLineChecker) CheckShellCommands(shellcmds string, time ToolTime) {
	ck.CheckShellCommand(shellcmds, true, time)
	if !hasSuffix(shellcmds, ";") {
		ck.mkline.Warnf("This shell command list should end with a semicolon.")
	}
}

func (ck *ShellLineChecker) CheckShellCommandLine(shelltext string) {
	if trace.Tracing {
		defer trace.Call1(shelltext)()
	}

	line := ck.mkline.Line

	// TODO: Add sed and mv in addition to ${SED} and ${MV}.
	// TODO: Now that a shell command parser is available, be more precise in the condition.
	if contains(shelltext, "${SED}") && contains(shelltext, "${MV}") {
		line.Notef("Use the SUBST framework instead of ${SED} and ${MV}.")
		line.Explain(
			"Using the SUBST framework instead of explicit commands is easier",
			"to understand, since all the complexity of using sed and mv is",
			"hidden behind the scenes.",
			"",
			sprintf("Run %q for more information.", bmakeHelp("subst")))
		if contains(shelltext, "#") {
			line.Explain(
				"When migrating to the SUBST framework, pay attention to \"#\" characters.",
				"In shell commands, make(1) does not interpret them as",
				"comment character, but in variable assignments it does.",
				"Therefore, instead of the shell command",
				"",
				"\tsed -e 's,#define foo,,'",
				"",
				"you need to write",
				"",
				"\tSUBST_SED.foo+=\t's,\\#define foo,,'")
		}
	}

	lexer := textproc.NewLexer(shelltext)
	lexer.NextHspace()
	hiddenAndSuppress := lexer.NextBytesFunc(func(b byte) bool { return b == '-' || b == '@' })
	if hiddenAndSuppress != "" {
		ck.checkHiddenAndSuppress(hiddenAndSuppress, lexer.Rest())
	}
	setE := lexer.SkipString("${RUN}")
	if !setE {
		if lexer.NextString("${_PKG_SILENT}${_PKG_DEBUG}") != "" {
			line.Errorf("Use of _PKG_SILENT and _PKG_DEBUG is obsolete. Use ${RUN} instead.")
		}
	}
	lexer.SkipHspace()
	lexer.SkipString("${_ULIMIT_CMD}") // It brings its own semicolon, just like ${RUN}.

	if contains(lexer.Rest(), "${RUN}") {
		line.Errorf("The expression \"${RUN}\" must only occur at the beginning of a shell command line.")
		line.Explain(
			"The expression ${RUN} expands to special instructions for make",
			"that are only valid at the beginning of a shell command line,",
			"even before any \"@\" character.")
	}

	ck.CheckShellCommand(lexer.Rest(), setE, RunTime)
	ck.checkMultiLineComment()
	if hasSuffix(shelltext, ";") && !hasSuffix(shelltext, "\\;") && !contains(shelltext, "#") {
		fix := line.Autofix()
		fix.Notef("A trailing semicolon at the end of a shell command line is redundant.")
		if strings.Count(shelltext, ";") == 1 {
			fix.Replace(";", "")
		}
		fix.Apply()
	}
}

func (ck *ShellLineChecker) checkHiddenAndSuppress(hiddenAndSuppress, rest string) {
	if trace.Tracing {
		defer trace.Call(hiddenAndSuppress, rest)()
	}

	switch {
	case !contains(hiddenAndSuppress, "@"):
		// Nothing is hidden at all.

	case hasPrefix(ck.MkLines.checkAllData.target, "show-"),
		hasSuffix(ck.MkLines.checkAllData.target, "-message"):
		// In these targets, all commands may be hidden.

	case hasPrefix(rest, "#"):
		// Shell comments may be hidden, since they cannot have side effects.

	default:
		tokens, _ := splitIntoShellTokens(ck.mkline.Line, rest)
		if len(tokens) > 0 {
			cmd := tokens[0]
			switch cmd {
			case "${DELAYED_ERROR_MSG}",
				"${DELAYED_WARNING_MSG}",
				"${DO_NADA}",
				"${ECHO}", "echo", "${ECHO_N}",
				"${ECHO_MSG}",
				"${ERROR_CAT}", "${ERROR_MSG}",
				"${FAIL_MSG}",
				"${INFO_MSG}",
				"${PHASE_MSG}",
				"${PRINTF}", "printf",
				"${SHCOMMENT}",
				"${STEP_MSG}",
				"${WARNING_CAT}", "${WARNING_MSG}":
				break
			default:
				ck.mkline.Warnf("The shell command %q should not be hidden.", cmd)
				ck.mkline.Explain(
					"Hidden shell commands do not appear on the terminal",
					"or in the log file when they are executed.",
					"When they fail, the error message cannot be related to the command,",
					"which makes debugging more difficult.",
					"",
					"It is better to insert ${RUN} at the beginning of the whole command line.",
					"This will hide the command by default but shows it when PKG_DEBUG_LEVEL is set.")
			}
		}
	}

	if contains(hiddenAndSuppress, "-") {
		ck.mkline.Warnf("Using a leading \"-\" to suppress errors is deprecated.")
		ck.mkline.Explain(
			"If you really want to ignore any errors from this command, append \"|| ${TRUE}\" to the command.",
			"This is more visible than a single hyphen, and it should be.")
	}
}

var shellCommandsType = NewVartype(BtShellCommands, NoVartypeOptions, NewACLEntry("*", aclpAllRuntime))

var shellCommandsEctx = &ExprContext{shellCommandsType, EctxUnknownTime, EctxQuotPlain, false}

func (ck *ShellLineChecker) CheckWord(token string, checkQuoting bool, time ToolTime) {
	if trace.Tracing {
		defer trace.Call(token, checkQuoting)()
	}

	if token == "" {
		return
	}

	var line = ck.mkline.Line

	// Delegate check for shell words consisting of a single expression
	// to the MkLineChecker. Examples for these are ${VAR:Mpattern} or $@.
	if expr := ToExpr(token); expr != nil {
		if ck.checkExpr {
			exprChecker := NewMkExprChecker(expr, ck.MkLines, ck.mkline)
			exprChecker.Check(shellCommandsEctx)
		}
		return
	}

	if matches(token, `\$\{PREFIX\}/man(?:$|/)`) {
		line.Warnf("Use ${PKGMANDIR} instead of \"man\".")
	}

	if G.Pkgsrc != nil && contains(token, "etc/rc.d") {
		line.Warnf("Use the RCD_SCRIPTS mechanism to install rc.d scripts automatically to ${RCD_SCRIPTS_EXAMPLEDIR}.")
	}

	ck.checkWordQuoting(token, checkQuoting, time)
}

func (ck *ShellLineChecker) checkWordQuoting(token string, checkQuoting bool, time ToolTime) {
	tok := NewShTokenizer(ck.mkline.Line, token)

	atoms := tok.ShAtoms()
	quoting := shqPlain
outer:
	for len(atoms) > 0 {
		atom := atoms[0]
		// Cutting off the first atom is done at the end of the loop since in
		// some cases the called methods need to see the current atom.

		if trace.Tracing {
			trace.Stepf("shell state %s: %q", quoting, atom)
		}

		switch {
		case atom.Quoting == shqBackt || atom.Quoting == shqDquotBackt:
			backtCommand := ck.unescapeBackticks(&atoms, quoting)
			if backtCommand != "" {
				ck.CheckShellCommand(backtCommand, true, time)
			}
			continue

			// Make(1) variables have the same syntax, no matter in which state the shell parser is currently.
		case ck.checkExprToken(&atoms, quoting):
			continue

		case quoting == shqPlain:
			switch {
			case atom.Type == shtShExpr:
				ck.checkShExprPlain(atom, checkQuoting)

			case atom.Type == shtSubshell:
				// Early return to avoid further parse errors.
				// As of December 2018, it might be worth continuing again since the
				// shell parser has improved in 2018.
				return

			case atom.Type == shtText:
				break

			default:
				break outer
			}
		}

		quoting = atom.Quoting
		atoms = atoms[1:]
	}

	if trimHspace(tok.Rest()) != "" {
		ck.Warnf("Internal pkglint error in ShellLine.CheckWord at %q (quoting=%s), rest: %s",
			token, quoting.String(), tok.Rest())
	}
}

func (ck *ShellLineChecker) CheckShellCommand(shellcmd string, setE bool, time ToolTime) {
	if trace.Tracing {
		defer trace.Call0()()
	}

	line := ck.mkline.Line
	program, err := parseShellProgram(line, shellcmd)
	// XXX: This code is duplicated in checkWordQuoting.
	if err != nil && contains(shellcmd, "$$(") { // Hack until the shell parser can handle subshells.
		line.Warnf("Invoking subshells via $(...) is not portable enough.")
		return
	}
	if err != nil {
		line.Warnf("Pkglint ShellLine.CheckShellCommand: %s", err.Error())
		return
	}

	walker := NewMkShWalker()
	walker.Callback.SimpleCommand = func(command *MkShSimpleCommand) {
		scc := NewSimpleCommandChecker(command, time, ck.mkline, ck.MkLines)
		scc.Check()
		// TODO: Implement getopt parsing for StrCommand.
		if scc.strcmd.Name == "set" && scc.strcmd.AnyArgMatches(`^-.*e`) {
			setE = true
		}
	}
	walker.Callback.AndOr = func(andor *MkShAndOr) {
		if G.WarnExtra && !setE && walker.Current().Index != 0 {
			ck.checkSetE(walker.Parent(1).(*MkShList), walker.Current().Index)
		}
	}
	walker.Callback.Pipeline = func(pipeline *MkShPipeline) {
		ck.checkPipeExitcode(pipeline)
	}
	walker.Callback.Word = func(word *ShToken) {
		// TODO: Try to replace false with true here; it had been set to false
		//  in 2016 for no apparent reason.
		ck.CheckWord(word.MkText, false, time)
	}

	walker.Walk(program)
}

func (ck *ShellLineChecker) checkSetE(list *MkShList, index int) {
	if trace.Tracing {
		defer trace.Call0()()
	}

	command := list.AndOrs[index-1].Pipes[0].Cmds[0]
	if command.Simple == nil || !ck.canFail(command) {
		return
	}

	line := ck.mkline.Line
	if !line.warnedAboutSetE.FirstTime() {
		return
	}

	line.Warnf("Switch to \"set -e\" mode before using a semicolon (after %q) to separate commands.",
		NewStrCommand(command.Simple).String())
	line.Explain(
		"Normally, when a shell command fails (returns non-zero),",
		"the remaining commands are still executed.",
		"For example, the following commands would remove",
		"all files from the HOME directory:",
		"",
		"\tcd \"$HOME\"; cd /nonexistent; rm -rf *",
		"",
		"In \"set -e\" mode, the shell stops when a command fails.",
		"",
		"To fix this warning, you can:",
		"",
		"* insert ${RUN} at the beginning of the line",
		"  (which among other things does \"set -e\")",
		"* insert \"set -e\" explicitly at the beginning of the line",
		"* use \"&&\" instead of \";\" to separate the commands")
}

func (ck *ShellLineChecker) checkPipeExitcode(pipeline *MkShPipeline) {
	if trace.Tracing {
		defer trace.Call0()()
	}

	canFail := func() (bool, string) {
		for _, cmd := range pipeline.Cmds[:len(pipeline.Cmds)-1] {
			if ck.canFail(cmd) {
				if cmd.Simple != nil && cmd.Simple.Name != nil {
					return true, cmd.Simple.Name.MkText
				}
				return true, ""
			}
		}
		return false, ""
	}

	if G.WarnExtra && len(pipeline.Cmds) > 1 {
		if canFail, cmd := canFail(); canFail {
			if cmd != "" {
				ck.Warnf("The exitcode of %q at the left of the | operator is ignored.", cmd)
			} else {
				ck.Warnf("The exitcode of the command at the left of the | operator is ignored.")
			}
			ck.Explain(
				"In a shell command like \"cat *.txt | grep keyword\", if the command",
				"on the left side of the \"|\" fails, this failure is ignored.",
				"",
				"If you need to detect the failure of the left-hand-side command, use",
				"temporary files to save the output of the command.",
				"A good place to create those files is in ${WRKDIR}.")
		}
	}
}

// canFail returns true if the given shell command can fail.
// Most shell commands can fail for various reasons, such as missing
// files or invalid arguments.
//
// Commands that can fail:
//
//	echo "hello" > file
//	sed 's,$, world,,' < input > output
//	find . -print
//	wc -l *
//
// Commands that cannot fail:
//
//	echo "hello"
//	sed 's,$, world,,'
//	wc -l
func (ck *ShellLineChecker) canFail(cmd *MkShCommand) bool {
	simple := cmd.Simple
	if simple == nil {
		return true
	}

	if simple.Name == nil {
		for _, assignment := range simple.Assignments {
			text := assignment.MkText
			if contains(text, "`") || contains(text, "$(") {
				if !contains(text, "|| ${TRUE}") && !contains(text, "|| true") {
					return true
				}
			}
		}
		return false
	}

	for _, redirect := range simple.Redirections {
		if !hasSuffix(redirect.Op, "&") {
			return true
		}
	}

	cmdName := simple.Name.MkText
	switch cmdName {
	case "${ECHO_MSG}", "${PHASE_MSG}", "${STEP_MSG}",
		"${INFO_MSG}", "${WARNING_MSG}", "${ERROR_MSG}",
		"${WARNING_CAT}", "${ERROR_CAT}",
		"${DO_NADA}":
		return false
	case "${FAIL_MSG}":
		return true
	case "set":
	}

	tool, _ := G.Tool(ck.MkLines, cmdName, RunTime)
	if tool == nil {
		return true
	}

	toolName := tool.Name
	args := simple.Args
	argc := len(args)
	switch toolName {
	case "basename", "dirname", "echo", "env", "printf", "tr":
		return false
	case "sed", "gsed":
		if argc == 2 && args[0].MkText == "-e" {
			return false
		}
		return argc != 1
	case "grep", "ggrep":
		return argc != 1
	}

	return true
}

// unescapeBackticks takes a backticks expression like `echo \\"hello\\"` and
// returns the part inside the backticks, removing one level of backslashes.
//
// Backslashes are only removed before a dollar, a backslash or a backtick.
// Other backslashes generate a warning since it is easier to remember that
// all backslashes are unescaped.
//
// See http://www.opengroup.org/onlinepubs/009695399/utilities/xcu_chap02.html#tag_02_06_03
func (ck *ShellLineChecker) unescapeBackticks(atoms *[]*ShAtom, quoting ShQuoting) string {
	line := ck.mkline.Line

	// Skip the initial backtick.
	*atoms = (*atoms)[1:]

	var unescaped strings.Builder
	for len(*atoms) > 0 {
		atom := (*atoms)[0]
		*atoms = (*atoms)[1:]

		if atom.Quoting == quoting {
			return unescaped.String()
		}

		if atom.Type != shtText {
			unescaped.WriteString(atom.MkText)
			continue
		}

		lex := textproc.NewLexer(atom.MkText)
		for !lex.EOF() {
			unescaped.WriteString(lex.NextBytesFunc(func(b byte) bool { return b != '\\' }))
			if lex.SkipByte('\\') {
				switch lex.PeekByte() {
				case '"', '\\', '`', '$':
					unescaped.WriteByte(byte(lex.PeekByte()))
					lex.Skip(1)
				default:
					line.Warnf("Backslashes should be doubled inside backticks.")
					unescaped.WriteByte('\\')
				}
			}
		}

		// XXX: The regular expression is a bit cheated but is good enough until
		//  pkglint has a real parser for all shell constructs.
		if atom.Quoting == shqDquotBackt && matches(atom.MkText, `(^|[^\\])"`) {
			line.Warnf("Double quotes inside backticks inside double quotes are error prone.")
			line.Explain(
				"According to the SUSv3, they produce undefined results.",
				"",
				"See the paragraph starting \"Within the backquoted ...\" in",
				"http://www.opengroup.org/onlinepubs/009695399/utilities/xcu_chap02.html.",
				"",
				"To avoid this uncertainty, escape the double quotes using \\\".")
		}
	}

	line.Errorf("Unfinished backticks after %q.", unescaped.String())
	return unescaped.String()
}

func (ck *ShellLineChecker) checkShExprPlain(atom *ShAtom, checkQuoting bool) {
	shVarname := atom.ShVarname()

	if shVarname == "@" {
		ck.Warnf("The $@ shell variable should only be used in double quotes.")

	} else if G.WarnQuoting && checkQuoting && ck.variableNeedsQuoting(shVarname) {
		ck.Warnf("Unquoted shell variable %q.", shVarname)
		ck.Explain(
			"When a shell variable contains whitespace, it is expanded (split into multiple words)",
			"when it is written as $variable in a shell script.",
			"If that is not intended, it should be surrounded by quotation marks, like \"$variable\".",
			"This way it always expands to a single word, preserving all whitespace and other special characters.",
			"",
			"Example:",
			"\tfname=\"Curriculum vitae.doc\"",
			"\tcp $filename /tmp",
			"\t# tries to copy the two files \"Curriculum\" and \"Vitae.doc\"",
			"",
			"\tcp \"$filename\" /tmp",
			"\t# copies one file, as intended")
	}

	if shVarname == "?" {
		ck.Warnf("The $? shell variable is often not available in \"set -e\" mode.")
		// TODO: Explain how to properly fix this warning.
		// TODO: Make sure the warning is only shown when applicable.
	}
}

func (ck *ShellLineChecker) variableNeedsQuoting(shVarname string) bool {
	switch shVarname {
	case "#", "?", "$":
		return false // Definitely ok
	case "d", "f", "i", "id", "file", "src", "dst", "prefix":
		return false // Probably ok
	}
	return !hasSuffix(shVarname, "dir") // Probably ok
}

func (ck *ShellLineChecker) checkExprToken(atoms *[]*ShAtom, quoting ShQuoting) bool {
	expr := (*atoms)[0].Expr()
	if expr == nil {
		return false
	}

	*atoms = (*atoms)[1:]
	varname := expr.varname

	if varname == "@" {
		// No autofix here since it may be a simple typo.
		// Maybe the package developer meant the shell variable instead.
		ck.Warnf("Use \"${.TARGET}\" instead of \"$@\".")
		ck.Explain(
			"The variable $@ can easily be confused with the shell variable of",
			"the same name, which has a completely different meaning.")

		varname = ".TARGET"
		expr = NewMkExpr(varname, expr.modifiers...)
	}

	switch {
	case quoting == shqPlain && expr.IsQ():
		// Fine.

	case (quoting == shqSquot || quoting == shqDquot) && matches(varname, `^(?:.*DIR|.*FILE|.*PATH|.*_VAR|PREFIX|.*BASE|PKGNAME)$`):
		// This is ok as long as these variables don't have embedded [$\\"'`].

	case quoting != shqPlain && expr.IsQ():
		ck.Warnf("The :Q modifier should not be used inside quotes.")
		ck.Explain(
			"The :Q modifier is intended for embedding a string into a shell program.",
			"It escapes all characters that have a special meaning in shell programs.",
			"It only works correctly when it appears outside of \"double\" or 'single'",
			"quotes or `backticks`.",
			"",
			"When it is used inside double quotes or backticks, the resulting string may",
			"contain more backslashes than intended.",
			"",
			"When it is used inside single quotes and the string contains a single quote",
			"itself, it produces syntax errors in the shell.",
			"",
			"To fix this warning, either remove the :Q or the double quotes.",
			"In most cases, it is more appropriate to remove the double quotes.",
			"",
			"A special case is for empty strings.",
			"If the empty string should be preserved as an empty string,",
			"the correct form is ${VAR:Q}'' with either leading or trailing single or double quotes.",
			"If the empty string should just be skipped,",
			"a simple ${VAR:Q} without any surrounding quotes is correct.")
	}

	if ck.checkExpr {
		ectx := ExprContext{shellCommandsType, EctxUnknownTime, quoting.ToExprContext(), true}
		NewMkExprChecker(expr, ck.MkLines, ck.mkline).Check(&ectx)
	}

	return true
}

func (ck *ShellLineChecker) checkMultiLineComment() {
	mkline := ck.mkline
	if !mkline.IsMultiline() || !contains(mkline.Text, "#") {
		return
	}

	for rawIndex, rawLine := range mkline.raw[:len(mkline.raw)-1] {
		text := strings.TrimSuffix(mkline.RawText(rawIndex), "\\")
		tokens, rest := splitIntoShellTokens(nil, text)
		if rest != "" {
			return
		}

		for _, token := range tokens {
			if hasPrefix(token, "#") {
				ck.warnMultiLineComment(rawIndex, rawLine)
				return
			}
		}
	}
}

func (ck *ShellLineChecker) warnMultiLineComment(rawIndex int, raw *RawLine) {
	line := ck.mkline.Line
	singleLine := NewLine(
		line.Filename(),
		line.Location.Lineno(rawIndex),
		line.RawText(rawIndex),
		raw)

	singleLine.Warnf("The shell comment does not stop at the end of this line.")
	singleLine.Explain(
		"When a shell command is spread out on multiple lines that are",
		"continued with a backslash, they will nevertheless be converted to",
		"a single line before the shell sees them.",
		"",
		"This means that even if it looks as if the comment only spanned",
		"one line in the Makefile, in fact it spans until the end of the whole",
		"shell command.",
		"",
		"To insert a comment into shell code, you can write it like this:",
		"",
		"\t${SHCOMMENT} \"The following command might fail; this is ok.\"",
		"",
		"Note that any special characters in the comment are still",
		"interpreted by the shell.",
		"",
		"If that is not possible, you can apply the :D modifier to the",
		"variable with the empty name, which is guaranteed to be undefined:",
		"",
		"\t${:D this is commented out}")
}

func (ck *ShellLineChecker) Warnf(format string, args ...interface{}) {
	ck.mkline.Warnf(format, args...)
}

func (ck *ShellLineChecker) Explain(explanation ...string) {
	ck.mkline.Explain(explanation...)
}
