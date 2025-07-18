package pkglint

import (
	"errors"
	"fmt"
	"gopkg.in/check.v1"
	"os"
	"reflect"
	"sort"
	"testing"
	"time"
)

func (s *Suite) Test_YesNoUnknown_String(c *check.C) {
	t := s.Init(c)

	t.CheckEquals(yes.String(), "yes")
	t.CheckEquals(no.String(), "no")
	t.CheckEquals(unknown.String(), "unknown")
}

func (s *Suite) Test_trimHspace(c *check.C) {
	t := s.Init(c)

	t.CheckEquals(trimHspace("a b"), "a b")
	t.CheckEquals(trimHspace(" a b "), "a b")
	t.CheckEquals(trimHspace("\ta b\t"), "a b")
	t.CheckEquals(trimHspace(" \t a b\t \t"), "a b")
}

func (s *Suite) Test_trimCommon(c *check.C) {
	t := s.Init(c)

	test := func(a, b, trimmedA, trimmedB string) {
		ta, tb := trimCommon(a, b)
		t.CheckEquals(ta, trimmedA)
		t.CheckEquals(tb, trimmedB)
	}

	test("", "",
		"", "")

	test("equal", "equal",
		"", "")

	test("prefixA", "prefixB",
		"A", "B")

	test("ASuffix", "BSuffix",
		"A", "B")

	test("PreMiddlePost", "PreCenterPost",
		"Middle", "Center")

	test("", "b",
		"", "b")

	test("a", "",
		"a", "")
}

func (s *Suite) Test_replaceOnce(c *check.C) {
	t := s.Init(c)

	test := func(s, from, to, result string) {
		ok, actualResult := replaceOnce(s, from, to)

		t.CheckEquals(actualResult, result)
		t.CheckEquals(ok, result != s)
	}

	// The text does not occur at all.
	test("something else", "from", "to", "something else")

	// The text occurs exactly once.
	test("from", "from", "to", "to")

	// The text occurs at two places, non-overlapping.
	test("from from", "from", "to", "from from")

	// The text occurs at three places, non-overlapping.
	test("aaa", "a", "b", "aaa")

	// The text occurs at two places, the occurrences overlap.
	test("aaa", "aa", "b", "aaa")
}

func (s *Suite) Test_condStr(c *check.C) {
	t := s.Init(c)

	t.CheckEquals(condStr(true, "T", "F"), "T")
	t.CheckEquals(condStr(false, "T", "F"), "F")
}

func (s *Suite) Test_condInt(c *check.C) {
	t := s.Init(c)

	t.CheckEquals(condInt(true, 123, 456), 123)
	t.CheckEquals(condInt(false, 123, 456), 456)
}

func (s *Suite) Test_imax(c *check.C) {
	t := s.Init(c)

	t.CheckEquals(imax(2, 5), 5)
	t.CheckEquals(imax(5, 2), 5)
}

func (s *Suite) Test_imin(c *check.C) {
	t := s.Init(c)

	t.CheckEquals(imin(2, 5), 2)
	t.CheckEquals(imin(5, 2), 2)
}

func (s *Suite) Test_assertNil(c *check.C) {
	t := s.Init(c)

	assertNil(nil, "this is not an error")

	t.ExpectPanic(
		func() { assertNil(errors.New("unexpected error"), "Oops") },
		"Pkglint internal error: Oops: unexpected error")
}

func (s *Suite) Test_assertNotNil(c *check.C) {
	t := s.Init(c)

	assertNotNil("this string is not nil")

	t.ExpectPanic(
		func() { assertNotNil(nil) },
		"Pkglint internal error: unexpected nil pointer")
	t.ExpectPanic(
		func() { var ptr *string; assertNotNil(ptr) },
		"Pkglint internal error: unexpected nil pointer")
}

func (s *Suite) Test_assert(c *check.C) {
	t := s.Init(c)

	assert(true)
	t.ExpectAssert(func() { assert(false) })
}

func (s *Suite) Test_isEmptyDir(c *check.C) {
	t := s.Init(c)

	t.CreateFileLines("CVS/Entries",
		"dummy")
	t.CreateFileLines("subdir/CVS/Entries",
		"dummy")

	t.CheckEquals(isEmptyDir(t.File(".")), true)
	t.CheckEquals(isEmptyDir(t.File("CVS")), true)

	t.Chdir(".")

	t.CheckEquals(isEmptyDir("."), true)
	t.CheckEquals(isEmptyDir("CVS"), true)
}

func (s *Suite) Test_isEmptyDir__and_getSubdirs(c *check.C) {
	t := s.Init(c)

	t.CreateFileLines("CVS/Entries",
		"dummy")

	if dir := t.File("."); true {
		t.CheckEquals(isEmptyDir(dir), true)
		t.CheckDeepEquals(getSubdirs(dir), []RelPath(nil))

		t.CreateFileLines("somedir/file")

		t.CheckEquals(isEmptyDir(dir), false)
		t.CheckDeepEquals(getSubdirs(dir), []RelPath{"somedir"})
	}

	if absent := t.File("nonexistent"); true {
		t.CheckEquals(isEmptyDir(absent), true) // Counts as empty.

		// The last group from the error message is localized, therefore the matching.
		t.ExpectFatalMatches(
			func() { getSubdirs(absent) },
			`FATAL: ~/nonexistent: Cannot be read: open ~/nonexistent: (.+)\n`)
	}
}

func (s *Suite) Test_getSubdirs(c *check.C) {
	t := s.Init(c)

	t.CreateFileLines("subdir/file")
	t.CreateFileLines("empty/file")
	t.CheckNil(os.Remove(t.File("empty/file").String()))

	t.CheckDeepEquals(getSubdirs(t.File(".")), []RelPath{"subdir"})
}

func (s *Suite) Test_isIgnoredFilename(c *check.C) {
	t := s.Init(c)

	test := func(filename string, isIgnored bool) {
		t.CheckEquals(isIgnoredFilename(filename), isIgnored)
	}

	test("filename.mk", false)
	test(".github", true)
	test(".gitignore", true)
	test(".git", true)
	test(".gitattributes", true)
	test("CVS", true)
	test(".svn", true)
	test(".hg", true)

	// There is actually an IDEA plugin for pkgsrc.
	// See https://github.com/rillig/intellij-pkgsrc.
	test(".idea", true)

	// After editing a file, run 'cvs up -CA'.  This creates a backup.
	test(".#Makefile.1.689", true)

	// https://www.gnu.org/software/trans-coord/manual/cvs/cvs.html#cvsignore
	test("PLIST.~1.7.~", true)
}

func (s *Suite) Test_isLocallyModified(c *check.C) {
	t := s.Init(c)

	unmodified := t.CreateFileLines("unmodified")
	modTime := time.Unix(1136239445, 0).UTC()

	err := os.Chtimes(unmodified.String(), modTime, modTime)
	t.CheckNil(err)

	st, err := os.Lstat(unmodified.String())
	t.CheckNil(err)

	// Make sure that the file system has second precision and accuracy.
	t.CheckDeepEquals(st.ModTime().UTC(), modTime)

	modified := t.CreateFileLines("modified")

	t.CreateFileLines("CVS/Entries",
		"/unmodified//"+modTime.Format(time.ANSIC)+"//",
		"/modified//"+modTime.Format(time.ANSIC)+"//",
		"/enoent//"+modTime.Format(time.ANSIC)+"//")

	t.CheckEquals(isLocallyModified(unmodified), false)
	t.CheckEquals(isLocallyModified(modified), true)
	t.CheckEquals(isLocallyModified(t.File("enoent")), true)
	t.CheckEquals(isLocallyModified(t.File("not_mentioned")), false)
	t.CheckEquals(isLocallyModified(t.File("subdir/file")), false)

	t.DisableTracing()

	t.CheckEquals(isLocallyModified(t.File("unmodified")), false)
}

func (s *Suite) Test_tabWidth(c *check.C) {
	t := s.Init(c)

	t.CheckEquals(tabWidth("12345"), 5)
	t.CheckEquals(tabWidth("\t"), 8)
	t.CheckEquals(tabWidth("123\t"), 8)
	t.CheckEquals(tabWidth("1234567\t"), 8)
	t.CheckEquals(tabWidth("12345678\t"), 16)
}

// Since tabWidthAppend is used with logical lines (Line.Text) as well as with
// raw lines (RawLine.textnl or RawLine.orignl), and since the width only
// makes sense for a single line, better panic.
func (s *Suite) Test_tabWidthAppend__panic(c *check.C) {
	t := s.Init(c)

	t.ExpectAssert(func() { tabWidthAppend(0, "\n") })
}

func (s *Suite) Test_detab(c *check.C) {
	t := s.Init(c)

	t.CheckEquals(detab(""), "")
	t.CheckEquals(detab("\t"), "        ")
	t.CheckEquals(detab("1234\t9"), "1234    9")
	t.CheckEquals(detab("1234567\t"), "1234567 ")
	t.CheckEquals(detab("12345678\t"), "12345678        ")
}

func (s *Suite) Test_alignWith(c *check.C) {
	t := s.Init(c)

	test := func(str, other, expected string) {
		aligned := alignWith(str, other)
		t.CheckEquals(aligned, expected)
		t.CheckEquals(hasPrefix(aligned, str), true)

		// It would be unusual to call this function with a string
		// that itself ends with space.
		t.CheckEquals(rtrimHspace(aligned), str)
	}

	// The needed alignment may be empty.
	// In some contexts like the value of a variable assignment, this
	// should not happen. In other contexts it's ok.
	test("", "", "")

	test("VAR=", "1234567", "VAR=   ")
	test("VAR=", "12345678", "VAR=\t")
	test("VAR=", "123456789", "VAR=\t ")

	// If the other string is shorter, no extra tab is added.
	test("1234567890=", "V=", "1234567890=")
}

func (s *Suite) Test_alignmentToWidths(c *check.C) {
	t := s.Init(c)

	t.CheckEquals(alignmentToWidths(8, 72), "\t\t\t\t\t\t\t\t")
}

func (s *Suite) Test_indent(c *check.C) {
	t := s.Init(c)

	test := func(width int, ind string) {
		actual := indent(width)

		t.CheckEquals(actual, ind)
	}

	test(0, "")
	test(1, " ")
	test(7, "       ")
	test(8, "\t")
	test(15, "\t       ")
	test(16, "\t\t")
	test(72, "\t\t\t\t\t\t\t\t\t")
	test(79, "\t\t\t\t\t\t\t\t\t       ")
	test(80, "\t\t\t\t\t\t\t\t\t\t")
	test(87, "\t\t\t\t\t\t\t\t\t\t       ")
}

func (s *Suite) Test_alignmentAfter(c *check.C) {
	t := s.Init(c)

	test := func(prefix string, width int, ind string) {
		actual := alignmentAfter(prefix, width)

		t.CheckEquals(actual, ind)
	}

	test("", 0, "")
	test("", 15, "\t       ")

	test("  ", 5, "   ")
	test("      ", 10, "\t  ")

	test("\t", 15, "       ")
	test(" \t", 15, "       ")
	test("       \t", 15, "       ")
	test("\t    ", 15, "   ")

	test("    ", 16, "\t\t")

	// The desired width must be at least the width of the prefix.
	t.ExpectAssert(func() { test("\t", 7, "") })
}

func (s *Suite) Test_shorten(c *check.C) {
	t := s.Init(c)

	t.CheckEquals(shorten("aaaaa", 3), "aaa...")
	t.CheckEquals(shorten("aaaaa", 5), "aaaaa")
	t.CheckEquals(shorten("aaa", 5), "aaa")
}

func (s *Suite) Test_varnameBase(c *check.C) {
	t := s.Init(c)

	t.CheckEquals(varnameBase("VAR"), "VAR")
	t.CheckEquals(varnameBase("VAR.param"), "VAR")
	t.CheckEquals(varnameBase(".CURDIR"), ".CURDIR")
}

func (s *Suite) Test_varnameCanon(c *check.C) {
	t := s.Init(c)

	t.CheckEquals(varnameCanon("VAR"), "VAR")
	t.CheckEquals(varnameCanon("VAR.param"), "VAR.*")
	t.CheckEquals(varnameCanon(".CURDIR"), ".CURDIR")
}

func (s *Suite) Test_varnameParam(c *check.C) {
	t := s.Init(c)

	t.CheckEquals(varnameParam("VAR"), "")
	t.CheckEquals(varnameParam("VAR.param"), "param")
	t.CheckEquals(varnameParam(".CURDIR"), "")
}

func (s *Suite) Test__regex_ReplaceFirst(c *check.C) {
	t := s.Init(c)

	m, rest := G.res.ReplaceFirst("a+b+c+d", `(\w)(.)(\w)`, "X")

	t.AssertNotNil(m)
	t.CheckDeepEquals(m, []string{"a+b", "a", "+", "b"})
	t.CheckEquals(rest, "X+c+d")
}

const reMkIncludeBenchmark = `^\.([\t ]*)(s?include)[\t ]+\"([^\"]+)\"[\t ]*(?:#.*)?$`
const reMkIncludeBenchmarkPositive = `^\.([\t ]*)(s?include)[\t ]+\"(.+)\"[\t ]*(?:#.*)?$`

func Benchmark_match3_buildlink3(b *testing.B) {
	for i := 0; i < b.N; i++ {
		match3(".include \"../../category/package/buildlink3.mk\"", reMkIncludeBenchmark)
	}
}

func Benchmark_match3_bsd_pkg_mk(b *testing.B) {
	for i := 0; i < b.N; i++ {
		match3(".include \"../../mk/bsd.pkg.mk\"", reMkIncludeBenchmark)
	}
}

func Benchmark_match3_same_dir(b *testing.B) {
	for i := 0; i < b.N; i++ {
		match3(".include \"options.mk\"", reMkIncludeBenchmark)
	}
}

func Benchmark_match3_bsd_pkg_mk_comment(b *testing.B) {
	for i := 0; i < b.N; i++ {
		match3(".include \"../../mk/bsd.pkg.mk\"          # infrastructure     ", reMkIncludeBenchmark)
	}
}

func Benchmark_match3_buildlink3_positive(b *testing.B) {
	for i := 0; i < b.N; i++ {
		match3(".include \"../../category/package/buildlink3.mk\"", reMkIncludeBenchmarkPositive)
	}
}

func Benchmark_match3_bsd_pkg_mk_positive(b *testing.B) {
	for i := 0; i < b.N; i++ {
		match3(".include \"../../mk/bsd.pkg.mk\"", reMkIncludeBenchmarkPositive)
	}
}

func Benchmark_match3_same_dir_positive(b *testing.B) {
	for i := 0; i < b.N; i++ {
		match3(".include \"options.mk\"", reMkIncludeBenchmarkPositive)
	}
}

func Benchmark_match3_bsd_pkg_mk_comment_positive(b *testing.B) {
	for i := 0; i < b.N; i++ {
		match3(".include \"../../mk/bsd.pkg.mk\"          # infrastructure     ", reMkIncludeBenchmarkPositive)
	}
}

func Benchmark_match3_explicit(b *testing.B) {
	for i := 0; i < b.N; i++ {
		MatchMkInclude(".include \"../../mk/bsd.pkg.mk\"          # infrastructure     ")
	}
}

func emptyToNil(slice []string) []string {
	if len(slice) == 0 {
		return nil
	}
	return slice
}

func (s *Suite) Test_containsExpr(c *check.C) {
	t := s.Init(c)

	test := func(str string, containsVar bool) {
		t.CheckEquals(containsExpr(str), containsVar)
	}

	test("", false)
	test("$", false) // A syntax error.

	// See the bmake manual page.
	test("$>", true) // .ALLSRC
	test("$!", true) // .ARCHIVE
	test("$<", true) // .IMPSRC
	test("$%", true) // .MEMBER
	test("$?", true) // .OODATE
	test("$*", true) // .PREFIX
	test("$@", true) // .TARGET

	test("$V", true)
	test("$v", true)
	test("${Var}", true)
	test("${VAR.${param}}", true)
	test("$(VAR)", true)

	test("$$", false)      // An escaped dollar character.
	test("$$(VAR)", false) // An escaped dollar character; probably a subshell.
	test("$${VAR}", false) // An escaped dollar character; probably a shell variable.
	test("$$VAR", false)   // An escaped dollar character.
}

func (s *Suite) Test_OncePerStringSlice(c *check.C) {
	t := s.Init(c)

	var once OncePerStringSlice

	t.CheckEquals(once.FirstTime("str"), true)
	t.CheckEquals(once.FirstTime("str", "str2"), true)
	t.CheckEquals(once.FirstTime("str", "str2"), false)
}

func (s *Suite) Test_naturalLess(c *check.C) {
	t := s.Init(c)

	var elements = []string{
		"",
		// Numbers are always considered smaller than other characters.
		"0", "000", "0000", "5", "7", "00011", "12", "00012", "000111",
		"!", "a", "a0", "a ", "aa", "ab", "b"}

	test := func(i int, ie string, j int, je string) {
		actual := naturalLess(ie, je)
		expected := i < j
		if actual != expected {
			t.CheckDeepEquals(
				[]interface{}{i, ie, j, je, actual},
				[]interface{}{i, ie, j, je, expected})
		}
	}

	for i, ie := range elements {
		for j, je := range elements {
			test(i, ie, j, je)
		}
	}
}

func (s *Suite) Test_FileCache(c *check.C) {
	t := s.Init(c)

	t.EnableTracingToLog()

	cache := NewFileCache(3)

	lines := t.NewLines("Makefile",
		MkCvsID,
		"# line 2")

	t.CheckNil(cache.Get("Makefile", 0))
	t.CheckEquals(cache.hits, 0)
	t.CheckEquals(cache.misses, 1)

	cache.Put("Makefile", 0, lines)
	t.CheckNil(cache.Get("Makefile", MustSucceed|LogErrors)) // Wrong LoadOptions.

	linesFromCache := cache.Get("Makefile", 0)
	t.CheckEquals(linesFromCache.Filename, NewCurrPath("Makefile"))
	t.CheckLen(linesFromCache.Lines, 2)
	t.CheckEquals(linesFromCache.Lines[0].Filename(), NewCurrPath("Makefile"))

	// Cache keys are normalized using path.Clean.
	linesFromCache2 := cache.Get("./Makefile", 0)
	t.CheckEquals(linesFromCache2.Filename, NewCurrPath("./Makefile"))
	t.CheckLen(linesFromCache2.Lines, 2)
	t.CheckEquals(linesFromCache2.Lines[0].Filename(), NewCurrPath("./Makefile"))

	cache.Put("file1.mk", 0, lines)
	cache.Put("file2.mk", 0, lines)

	// Now the cache is full. All three entries can be retrieved.
	t.CheckNotNil(cache.Get("Makefile", 0))
	t.CheckNotNil(cache.Get("file1.mk", 0))
	t.CheckNotNil(cache.Get("file2.mk", 0))

	// Adding another entry removes all entries with minimum count,
	// which currently are file1.mk and file2.mk.
	// Makefile is still in the cache because it was accessed once.
	cache.Put("file3.mk", 0, lines)

	t.CheckNotNil(cache.Get("Makefile", 0))
	t.CheckNil(cache.Get("file1.mk", 0))
	t.CheckNil(cache.Get("file2.mk", 0))
	t.CheckNotNil(cache.Get("file3.mk", 0))

	cache.Evict("Makefile")

	t.CheckNil(cache.Get("Makefile", 0))
	t.CheckLen(cache.table, 1)
	t.CheckLen(cache.mapping, 1)
	t.CheckEquals(cache.hits, 7)
	t.CheckEquals(cache.misses, 5)

	t.CheckOutputLines(
		"TRACE:   FileCache \"Makefile\" with count 4.",
		"TRACE:   FileCache \"file1.mk\" with count 2.",
		"TRACE:   FileCache \"file2.mk\" with count 2.",
		"TRACE:   FileCache.Evict \"file2.mk\" with count 2.",
		"TRACE:   FileCache.Evict \"file1.mk\" with count 2.",
		"TRACE:   FileCache.Halve \"Makefile\" with count 4.")
}

func (s *Suite) Test_FileCache_removeOldEntries__branch_coverage(c *check.C) {
	t := s.Init(c)

	t.EnableTracingToLog()
	G.Testing = false

	lines := t.NewLines("filename.mk",
		MkCvsID)
	cache := NewFileCache(3)
	cache.Put("filename1.mk", 0, lines)
	cache.Put("filename2.mk", 0, lines)
	cache.Get("filename2.mk", 0)
	cache.Get("filename2.mk", 0)
	cache.Put("filename3.mk", 0, lines)
	cache.Put("filename4.mk", 0, lines)

	t.CheckOutputLines(
		"TRACE:   FileCache.Evict \"filename3.mk\" with count 1.",
		"TRACE:   FileCache.Evict \"filename1.mk\" with count 1.",
		"TRACE:   FileCache.Halve \"filename2.mk\" with count 3.")
}

func (s *Suite) Test_FileCache_removeOldEntries__no_tracing(c *check.C) {
	t := s.Init(c)

	t.DisableTracing()

	lines := t.NewLines("filename.mk",
		MkCvsID)
	cache := NewFileCache(3)
	cache.Put("filename1.mk", 0, lines)
	cache.Put("filename2.mk", 0, lines)
	cache.Get("filename2.mk", 0)
	cache.Get("filename2.mk", 0)
	cache.Put("filename3.mk", 0, lines)
	cache.Put("filename4.mk", 0, lines)

	t.CheckOutputEmpty()
}

// Covers the newLen > 0 condition.
func (s *Suite) Test_FileCache_removeOldEntries__zero_capacity(c *check.C) {
	t := s.Init(c)

	lines := t.NewLines("filename.mk",
		MkCvsID)
	cache := NewFileCache(1)
	cache.Put("filename1.mk", 0, lines)

	// This call removes all existing entries from the cache,
	// as the cache's capacity is only 1.
	cache.Put("filename2.mk", 0, lines)
}

func (s *Suite) Test_FileCache_Evict__sort(c *check.C) {
	t := s.Init(c)

	lines := t.NewLines("filename.mk",
		MkCvsID)
	cache := NewFileCache(10)
	cache.Put("filename0.mk", 0, lines)
	cache.Put("filename1.mk", 0, lines)
	cache.Put("filename2.mk", 0, lines)
	cache.Put("filename3.mk", 0, lines)
	cache.Put("filename4.mk", 0, lines)
	cache.Put("filename5.mk", 0, lines)
	cache.Put("filename6.mk", 0, lines)
	cache.Put("filename7.mk", 0, lines)
	cache.Put("filename8.mk", 0, lines)
	cache.Put("filename9.mk", 0, lines)

	cache.Evict("filename5.mk")

	t.CheckLen(cache.table, 9)
	t.CheckNil(cache.Get("filename5.mk", 0))
	t.CheckNotNil(cache.Get("filename6.mk", 0))
}

func (s *Suite) Test_bmakeHelp(c *check.C) {
	t := s.Init(c)

	t.CheckEquals(bmakeHelp("subst"), confMake+" help topic=subst")
}

func (s *Suite) Test_wrap(c *check.C) {
	t := s.Init(c)

	wrapped := wrap(20,
		"See the pkgsrc guide, section \"Package components, Makefile\":",
		"https://www.NetBSD.org/doc/pkgsrc/pkgsrc.html#components.Makefile.",
		"",
		"For more information, ask on the tech-pkg@NetBSD.org mailing list.",
		"",
		"\tpreformatted line 1",
		"\tpreformatted line 2",
		"",
		"    intentionally indented",
		"*   itemization",
		"",
		"Normal",
		"text",
		"continues",
		"here",
		"with",
		"linebreaks.",
		"",
		"Sentence one.  Sentence two.",
		"",
		"A\tB\tC\tD",
		"E\tveryVeryVeryVeryVeryVeryVeryVeryLong")

	expected := []string{
		"See the pkgsrc",
		"guide, section",
		"\"Package components,",
		"Makefile\":",
		"https://www.NetBSD.org/doc/pkgsrc/pkgsrc.html#components.Makefile.",
		"",
		"For more",
		"information, ask on",
		"the",
		"tech-pkg@NetBSD.org",
		"mailing list.",
		"",
		"\tpreformatted line 1",
		"\tpreformatted line 2",
		"",
		"    intentionally indented",
		"*   itemization",
		"",
		"Normal text",
		"continues here with",
		"linebreaks.",
		"",
		"Sentence one.",
		"Sentence two.",
		"",
		"A\tB\tC\tD E",
		"veryVeryVeryVeryVeryVeryVeryVeryLong"}

	t.CheckDeepEquals(wrapped, expected)
}

func (s *Suite) Test_escapePrintable(c *check.C) {
	t := s.Init(c)

	t.CheckEquals(escapePrintable(""), "")
	t.CheckEquals(escapePrintable("ASCII only~\n\t"), "ASCII only~\n\t")
	t.CheckEquals(escapePrintable("Beep \u0007 control \u001F"), "Beep <U+0007> control <U+001F>")
	t.CheckEquals(escapePrintable("Bad \xFF character"), "Bad <0xFF> character")
	t.CheckEquals(escapePrintable("Unicode \uFFFD replacement"), "Unicode <U+FFFD> replacement")
}

func (s *Suite) Test_stringSliceLess(c *check.C) {
	t := s.Init(c)

	var elements = [][][]string{
		{nil, {}},
		{{"a"}},
		{{"a", "a"}},
		{{"a", "b"}},
		{{"b"}},
		{{"b", "a"}}}

	test := func(i int, iElement []string, j int, jElement []string) {
		actual := stringSliceLess(iElement, jElement)
		expected := i < j
		if actual != expected {
			t.CheckDeepEquals(
				[]interface{}{i, iElement, j, jElement, actual},
				[]interface{}{i, iElement, j, jElement, expected})
		}
	}

	for i, iElements := range elements {
		for j, jElements := range elements {
			for _, iElement := range iElements {
				for _, jElement := range jElements {
					test(i, iElement, j, jElement)
				}
			}
		}
	}
}

func (s *Suite) Test_joinSkipEmpty(c *check.C) {
	t := s.Init(c)

	t.CheckDeepEquals(
		joinSkipEmpty(", ", "", "one", "", "", "two", "", "three"),
		"one, two, three")
}

func (s *Suite) Test_joinCambridge(c *check.C) {
	t := s.Init(c)

	t.CheckDeepEquals(
		joinCambridge("and", "", "one", "", "", "two", "", "three"),
		"one, two and three")

	t.CheckDeepEquals(
		joinCambridge("and", "", "one", "", ""),
		"one")
}

func (s *Suite) Test_joinOxford(c *check.C) {
	t := s.Init(c)

	t.CheckDeepEquals(
		joinOxford("and", "", "one", "", "", "two", "", "three"),
		"one, two, and three")
}

func Test_expandCurlyBraces(t *testing.T) {
	tests := []struct {
		s    string
		want []string
	}{
		{"}", []string{"}"}},
		{"{}{}{}{}{}{}{}", []string{""}},
		{"{}}", []string{"}"}},
		{"{}{}{}{}{}{}{}}", []string{"}"}},
		{"{thir,f{our,if}}teen", []string{"thirteen", "fourteen", "fifteen"}},
		{"pkgname<=1.0", []string{"pkgname<=1.0"}},
		{"{pkgname,pkgname-client}<=1.0", []string{"pkgname<=1.0", "pkgname-client<=1.0"}},
		{"a{b,c,{d,e}}f", []string{"abf", "acf", "adf", "aef"}},
	}
	for _, tt := range tests {
		t.Run(tt.s, func(t *testing.T) {
			got := expandCurlyBraces(tt.s)
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("got %q, want %q", got, tt.want)
			}
		})
	}
}

func (s *Suite) Test_newPathMatcher(c *check.C) {
	t := s.Init(c)

	test := func(pattern string, matchType pathMatchType, matchPattern string) {
		t.CheckEquals(*newPathMatcher(pattern), pathMatcher{matchType, matchPattern, pattern})
	}

	testPanic := func(pattern string) {
		t.ExpectPanic(
			func() { _ = newPathMatcher(pattern) },
			"Pkglint internal error")
	}

	testPanic("*.[0123456]")
	testPanic("file.???")
	testPanic("*.???")
	test("", pmExact, "")
	test("exact", pmExact, "exact")
	test("*.mk", pmSuffix, ".mk")
	test("Makefile.*", pmPrefix, "Makefile.")
	testPanic("*.*")
	testPanic("**")
	testPanic("a*b")
	testPanic("[")
	testPanic("malformed[")
}

func (s *Suite) Test_pathMatcher_matches(c *check.C) {
	t := s.Init(c)

	test := func(pattern string, subject string, expected bool) {
		matcher := newPathMatcher(pattern)
		t.CheckEquals(matcher.matches(subject), expected)
	}

	test("", "", true)
	test("", "any", false)
	test("exact", "exact", true)
	test("exact", "different", false)

	test("*.mk", "filename.mk", true)
	test("*.mk", "filename.txt", false)
	test("*.mk", "filename.mkx", false)
	test("*.mk", ".mk", true)

	test("Makefile.*", "Makefile", false)
	test("Makefile.*", "Makefile.", true)
	test("Makefile.*", "Makefile.txt", true)
	test("Makefile.*", "makefile.txt", false)
}

func (s *Suite) Test_StringInterner(c *check.C) {
	t := s.Init(c)

	si := NewStringInterner()

	t.CheckEquals(si.Intern(""), "")
	t.CheckEquals(si.Intern("Hello"), "Hello")
	t.CheckEquals(si.Intern("Hello, world"), "Hello, world")
	t.CheckEquals(si.Intern("Hello, world"[0:5]), "Hello")
}

func (s *Suite) Test_shquote(c *check.C) {
	t := s.Init(c)

	test := func(in, out string) {
		t.CheckEquals(shquote(in), out)
	}

	test("", "''")
	test("'", "''\\'''")
	test("simple", "simple")
	test("~", "'~'")
}

func (s *Suite) Test_LazyStringBuilder_WriteByte__exact_match(c *check.C) {
	t := s.Init(c)

	sb := NewLazyStringBuilder("word")

	sb.WriteByte('w')
	sb.WriteByte('o')
	sb.WriteByte('r')
	sb.WriteByte('d')

	t.CheckEquals(sb.String(), "word")
	t.CheckNil(sb.buf)
}

func (s *Suite) Test_LazyStringBuilder_WriteByte__longer_than_expected(c *check.C) {
	t := s.Init(c)

	sb := NewLazyStringBuilder("word")
	sb.WriteByte('w')
	sb.WriteByte('o')
	sb.WriteByte('r')
	sb.WriteByte('d')
	sb.WriteByte('s')

	t.CheckEquals(sb.String(), "words")
	t.CheckDeepEquals(sb.buf, []byte{'w', 'o', 'r', 'd', 's'})
}

func (s *Suite) Test_LazyStringBuilder_WriteByte__shorter_than_expected(c *check.C) {
	t := s.Init(c)

	sb := NewLazyStringBuilder("word")
	sb.WriteByte('w')
	sb.WriteByte('o')

	t.CheckEquals(sb.String(), "wo")
	t.CheckNil(sb.buf)

	sb.WriteByte('r')
	sb.WriteByte('d')

	t.CheckEquals(sb.String(), "word")
	t.CheckNil(sb.buf)
}

func (s *Suite) Test_LazyStringBuilder_WriteByte__other_than_expected(c *check.C) {
	t := s.Init(c)

	sb := NewLazyStringBuilder("word")
	sb.WriteByte('w')
	sb.WriteByte('o')
	sb.WriteByte('l')
	sb.WriteByte('f')

	t.CheckEquals(sb.String(), "wolf")
	t.CheckDeepEquals(sb.buf, []byte{'w', 'o', 'l', 'f'})
}

func (s *Suite) Test_LazyStringBuilder_writeToBuf__assertion(c *check.C) {
	t := s.Init(c)

	sb := NewLazyStringBuilder("0123456789abcdef0123456789abcdef")
	sb.WriteString("0123456789abcdef0123456789abcdeX")

	t.CheckEquals(cap(sb.buf), 32)

	sb.Reset("0123456789abcdef")
	sb.WriteString("01234567")

	// Intentionally violate the invariant of the LazyStringBuilder that
	// as long as sb.usingBuf is false, sb.len is at most len(sb.expected).
	sb.expected = ""
	t.ExpectAssert(func() { sb.writeToBuf('x') })
}

func (s *Suite) Test_LazyStringBuilder_Reset(c *check.C) {
	t := s.Init(c)

	sb := NewLazyStringBuilder("word")
	sb.WriteByte('w')

	sb.Reset("other")

	t.CheckEquals(sb.String(), "")

	sb.WriteString("word")

	t.CheckEquals(sb.String(), "word")
	t.CheckEquals(sb.usingBuf, true)
	t.CheckDeepEquals(sb.buf, []byte("word"))

	sb.Reset("")

	t.CheckEquals(sb.String(), "")
	t.CheckEquals(sb.usingBuf, false)
	t.CheckDeepEquals(sb.buf, []byte("word"))

	sb.WriteByte('x')

	// Ensure that the buffer is reset properly.
	t.CheckEquals(sb.String(), "x")
	t.CheckEquals(sb.usingBuf, true)
	t.CheckDeepEquals(sb.buf, []byte("x"))
}

// sortedKeys takes the keys from an arbitrary map,
// converts them to strings if necessary,
// and then returns them sorted.
//
// It is only available during tests since it uses reflection.
func keys(m interface{}) []string {
	var keys []string
	for _, key := range reflect.ValueOf(m).MapKeys() {
		switch key := key.Interface().(type) {
		case fmt.Stringer:
			keys = append(keys, key.String())
		default:
			keys = append(keys, key.(string))
		}
	}
	sort.Strings(keys)
	return keys
}

func (s *Suite) Test_interval(c *check.C) {
	t := s.Init(c)

	i := newInterval()

	t.CheckEquals(i.min > i.max, true)

	i.add(3)

	t.CheckEquals(i.min, 3)
	t.CheckEquals(i.max, 3)

	i.add(7)

	t.CheckEquals(i.min, 3)
	t.CheckEquals(i.max, 7)

	i.add(-5)

	t.CheckEquals(i.min, -5)
	t.CheckEquals(i.max, 7)
}

type relation struct {
	idx           map[interface{}]int
	elements      []interface{}
	pairs         []struct{ a, b interface{} }
	reflexive     bool
	transitive    bool
	antisymmetric bool
	onError       func(s string)
}

func (r *relation) add(a interface{}, b interface{}) {
	if r.idx == nil {
		r.idx = make(map[interface{}]int)
	}
	if _, ok := r.idx[a]; !ok {
		r.idx[a] = len(r.idx)
		r.elements = append(r.elements, a)
	}
	if _, ok := r.idx[b]; !ok {
		r.idx[b] = len(r.idx)
		r.elements = append(r.elements, b)
	}
	r.pairs = append(r.pairs, struct{ a, b interface{} }{a, b})
}

func (r *relation) check(actual func(interface{}, interface{}) bool) {
	n := len(r.idx)
	rel := make([][]bool, n)
	for i := 0; i < n; i++ {
		rel[i] = make([]bool, n)
	}

	if r.reflexive {
		for i := 0; i < n; i++ {
			rel[i][i] = true
		}
	}

	for _, pair := range r.pairs {
		rel[r.idx[pair.a]][r.idx[pair.b]] = true
	}

	if r.transitive {
		for {
			changed := false
			for i := 0; i < n; i++ {
				for j := 0; j < n; j++ {
					for k := 0; k < n; k++ {
						if rel[i][j] && rel[j][k] && !rel[i][k] {
							rel[i][k] = true
							changed = true
						}
					}
				}
			}
			if !changed {
				break
			}
		}
	}

	if r.antisymmetric {
		for i := 0; i < n; i++ {
			for j := 0; j < n; j++ {
				if i != j && rel[i][j] && rel[j][i] {
					r.onError(sprintf(
						"the antisymmetric relation must not contain "+
							"both (%#[1]v, %#[2]v) and (%#[2]v, %#[1]v)",
						r.elements[i], r.elements[j]))
				}
			}
		}
	}

	for i := 0; i < n; i++ {
		for j := 0; j < n; j++ {
			ei := r.elements[i]
			ej := r.elements[j]
			actualRel := actual(ei, ej)
			if actualRel != rel[i][j] {
				_ = actual(ei, ej)
				r.onError(sprintf("expected %#v <=> %#v to be %v, was %v",
					ei, ej, rel[i][j], actualRel))
			}
		}
	}
}
