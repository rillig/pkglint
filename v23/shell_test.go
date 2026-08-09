package pkglint

import "testing"

func Test_splitIntoShellTokens__line_continuation(tt *testing.T) {
	t := NewTester(tt)
	defer t.Done()

	line := t.NewLine("filename.mk", 1, "")
	words, rest := splitIntoShellTokens(line, "if true; then \\")

	t.CheckDeepEquals(words, []string{"if", "true", ";", "then"})
	t.CheckEquals(rest, "\\")

	t.CheckOutputLines(
		"WARN: filename.mk:1: Internal pkglint error in ShTokenizer.ShAtom at \"\\\\\" (quoting=plain).")
}

func Test_splitIntoShellTokens__dollar_slash(tt *testing.T) {
	t := NewTester(tt)
	defer t.Done()

	line := t.NewLine("filename.mk", 1, "")
	words, rest := splitIntoShellTokens(line, "pax -s /.*~$$//g")

	t.CheckDeepEquals(words, []string{"pax", "-s", "/.*~$$//g"})
	t.CheckEquals(rest, "")
}

func Test_splitIntoShellTokens__dollar_subshell(tt *testing.T) {
	t := NewTester(tt)
	defer t.Done()

	line := t.NewLine("filename.mk", 1, "")
	words, rest := splitIntoShellTokens(line, "id=$$(${AWK} '{print}' < ${WRKSRC}/idfile) && echo \"$$id\"")

	t.CheckDeepEquals(words, []string{"id=$$(${AWK} '{print}' < ${WRKSRC}/idfile)", "&&", "echo", "\"$$id\""})
	t.CheckEquals(rest, "")
}

func Test_splitIntoShellTokens__semicolons(tt *testing.T) {
	t := NewTester(tt)
	defer t.Done()

	line := t.NewLine("filename.mk", 1, "")
	words, rest := splitIntoShellTokens(line, "word1 word2;;;")

	t.CheckDeepEquals(words, []string{"word1", "word2", ";;", ";"})
	t.CheckEquals(rest, "")
}

func Test_splitIntoShellTokens__whitespace(tt *testing.T) {
	t := NewTester(tt)
	defer t.Done()

	text := "\t${RUN} cd ${WRKSRC}&&(${ECHO} ${PERL5:Q};${ECHO})|${BASH} ./install"
	line := t.NewLine("filename.mk", 1, "")
	words, rest := splitIntoShellTokens(line, text)

	t.CheckDeepEquals(words, []string{
		"${RUN}",
		"cd", "${WRKSRC}",
		"&&", "(", "${ECHO}", "${PERL5:Q}", ";", "${ECHO}", ")",
		"|", "${BASH}", "./install"})
	t.CheckEquals(rest, "")
}

func Test_splitIntoShellTokens__finished_dquot(tt *testing.T) {
	t := NewTester(tt)
	defer t.Done()

	text := "\"\""
	line := t.NewLine("filename.mk", 1, "")
	words, rest := splitIntoShellTokens(line, text)

	t.CheckDeepEquals(words, []string{"\"\""})
	t.CheckEquals(rest, "")
}

func Test_splitIntoShellTokens__unfinished_dquot(tt *testing.T) {
	t := NewTester(tt)
	defer t.Done()

	text := "\t\""
	line := t.NewLine("filename.mk", 1, "")
	words, rest := splitIntoShellTokens(line, text)

	t.CheckNil(words)
	t.CheckEquals(rest, "\"")
}

func Test_splitIntoShellTokens__unescaped_dollar_in_dquot(tt *testing.T) {
	t := NewTester(tt)
	defer t.Done()

	text := "echo \"$$\""
	line := t.NewLine("filename.mk", 1, "")
	words, rest := splitIntoShellTokens(line, text)

	t.CheckDeepEquals(words, []string{"echo", "\"$$\""})
	t.CheckEquals(rest, "")

	t.CheckOutputEmpty()
}

func Test_splitIntoShellTokens__expr_with_embedded_space_and_other_vars(tt *testing.T) {
	t := NewTester(tt)
	defer t.Done()

	exprWord := "${GCONF_SCHEMAS:@.s.@${INSTALL_DATA} ${WRKSRC}/src/common/dbus/${.s.} ${DESTDIR}${GCONF_SCHEMAS_DIR}/@}"
	line := t.NewLine("filename.mk", 1, "")
	words, rest := splitIntoShellTokens(line, exprWord)

	t.CheckDeepEquals(words, []string{exprWord})
	t.CheckEquals(rest, "")
}

// Two shell variables, next to each other,
// are two separate atoms but count as a single token.
func Test_splitIntoShellTokens__two_shell_variables(tt *testing.T) {
	t := NewTester(tt)
	defer t.Done()

	code := "echo $$i$$j"
	line := t.NewLine("filename.mk", 1, "")
	words, rest := splitIntoShellTokens(line, code)

	t.CheckDeepEquals(words, []string{"echo", "$$i$$j"})
	t.CheckEquals(rest, "")
}

func Test_splitIntoShellTokens__expr_with_embedded_space(tt *testing.T) {
	t := NewTester(tt)
	defer t.Done()

	line := t.NewLine("filename.mk", 1, "")
	words, rest := splitIntoShellTokens(line, "${VAR:S/ /_/g}")

	t.CheckDeepEquals(words, []string{"${VAR:S/ /_/g}"})
	t.CheckEquals(rest, "")
}

func Test_splitIntoShellTokens__redirect(tt *testing.T) {
	t := NewTester(tt)
	defer t.Done()

	line := t.NewLine("filename.mk", 1, "")
	words, rest := splitIntoShellTokens(line, "echo 1>output 2>>append 3>|clobber 4>&5 6<input >>append")

	t.CheckDeepEquals(words, []string{
		"echo",
		"1>", "output",
		"2>>", "append",
		"3>|", "clobber",
		"4>&", "5",
		"6<", "input",
		">>", "append"})
	t.CheckEquals(rest, "")

	words, rest = splitIntoShellTokens(line, "echo 1> output 2>> append 3>| clobber 4>& 5 6< input >> append")

	t.CheckDeepEquals(words, []string{
		"echo",
		"1>", "output",
		"2>>", "append",
		"3>|", "clobber",
		"4>&", "5",
		"6<", "input",
		">>", "append"})
	t.CheckEquals(rest, "")
}

func Test_splitIntoShellTokens__expr(tt *testing.T) {
	t := NewTester(tt)
	defer t.Done()

	test := func(text string, tokens ...string) {
		line := t.NewLine("filename.mk", 1, "")

		words, rest := splitIntoShellTokens(line, text)

		t.CheckDeepEquals(words, tokens)
		t.CheckEquals(rest, "")
	}

	test(
		"sed -e s#@PREFIX@#${PREFIX}#g filename",

		"sed",
		"-e",
		"s#@PREFIX@#${PREFIX}#g",
		"filename")
}
