package pkglint

import (
	"gopkg.in/check.v1"
	"io"
	"os"
	"runtime"
	"strings"
)

func (s *Suite) Test_NewPath(c *check.C) {
	t := s.Init(c)

	t.CheckEquals(NewPath("filename"), NewPath("filename"))
	t.CheckEquals(NewPath("\\"), NewPath("\\"))
	c.Check(NewPath("\\"), check.Not(check.Equals), NewPath("/"))
}

func (s *Suite) Test_NewPathSlash(c *check.C) {
	t := s.Init(c)

	t.CheckEquals(NewPathSlash("filename"), NewPathSlash("filename"))
	t.CheckEquals(NewPathSlash("\\"), NewPathSlash("\\"))

	t.CheckEquals(
		NewPathSlash("\\"),
		NewPathSlash(condStr(runtime.GOOS == "windows", "/", "\\")))
}

func (s *Suite) Test_Path_String(c *check.C) {
	t := s.Init(c)

	for _, p := range []string{"", "filename", "a/b", "c\\d"} {
		t.CheckEquals(NewPath(p).String(), p)
	}
}

func (s *Suite) Test_Path_GoString(c *check.C) {
	t := s.Init(c)

	test := func(p Path, s string) {
		t.CheckEquals(p.GoString(), s)
	}

	test("", "\"\"")
	test("filename", "\"filename\"")
	test("a/b", "\"a/b\"")
	test("c\\d", "\"c\\\\d\"")
}

func (s *Suite) Test_Path_IsEmpty(c *check.C) {
	t := s.Init(c)

	test := func(p Path, isEmpty bool) {
		t.CheckEquals(p.IsEmpty(), isEmpty)
	}

	test("", true)
	test(".", false)
	test("/", false)
}

func (s *Suite) Test_Path_Dir(c *check.C) {
	t := s.Init(c)

	test := func(p, dir Path) {
		t.CheckEquals(p.Dir(), dir)
	}

	test("", ".")
	test("././././", ".")
	test("/root", "/")
	test("filename", ".")
	test("dir/filename", "dir")
	test("dir/filename\\with\\backslash", "dir")

	// TODO: I didn't expect that Dir would return the cleaned path.
	test("././././dir/filename", "dir")
}

func (s *Suite) Test_Path_Base(c *check.C) {
	t := s.Init(c)

	test := func(p Path, base string) {
		t.CheckEquals(p.Base(), base)
	}

	test("", ".") // That's a bit surprising
	test("././././", ".")
	test("/root", "root")
	test("filename", "filename")
	test("dir/filename", "filename")
	test("dir/filename\\with\\backslash", "filename\\with\\backslash")
}

func (s *Suite) Test_Path_Split(c *check.C) {
	t := s.Init(c)

	test := func(p, dir, base string) {
		actualDir, actualBase := NewPath(p).Split()

		t.CheckDeepEquals(
			[]string{actualDir.String(), actualBase},
			[]string{dir, base})
	}

	test("", "", "")
	test("././././", "././././", "")
	test("/root", "/", "root")
	test("filename", "", "filename")
	test("dir/filename", "dir/", "filename")
	test("dir/filename\\with\\backslash", "dir/", "filename\\with\\backslash")
}

func (s *Suite) Test_Path_Parts(c *check.C) {
	t := s.Init(c)

	test := func(p string, parts ...string) {
		t.CheckDeepEquals(NewPath(p).Parts(), parts)
	}

	// Only the empty path returns an empty slice.
	test("", nil...)

	// The standard cases for relative paths.
	test("relative", "relative")
	test("relative/subdir", "relative", "subdir")
	test("relative////subdir", "relative", "subdir")
	test("relative/..", "relative", "..")
	test("relative/.", "relative")

	// Leading dots are removed when they are followed by something.
	test("./relative", "relative")

	// A path consisting of only dots produces a single dot.
	test("./././.", ".")

	// Slashes at the end are treated like a single dot.
	test("././././", ".")
	test(".///////", ".")

	// Absolute paths have an empty first component.
	test("/", "")
	test("/.", "")
	test("/root", "", "root")

	// The backslash is not a path separator.
	test("dir/filename\\with\\backslash", "dir", "filename\\with\\backslash")
}

func (s *Suite) Test_Path_Count(c *check.C) {
	t := s.Init(c)

	test := func(p string, count int) {
		t.CheckEquals(NewPath(p).Count(), count)
	}

	test("././././", 1)
	test("/root", 2)
	test("filename", 1)
	test("dir/filename", 2)
	test("dir/filename\\with\\backslash", 2)

	// Only the empty path returns an empty slice.
	test("", 0)

	// The standard cases for canonical relative paths.
	test("relative", 1)
	test("relative/subdir", 2)
	test("relative////subdir", 2)
	test("relative/..", 2)
	test("relative/.", 1)

	// A path consisting of only dots produces a single dot.
	test("./././.", 1)

	// Slashes at the end are treated like a single dot.
	test("././././", 1)
	test(".///////", 1)

	// Absolute paths have an empty first component.
	test("/", 1)
	test("/.", 1)
	test("/root", 2)

	// The backslash is not a path separator.
	test("dir/filename\\with\\backslash", 2)
}

func (s *Suite) Test_Path_HasPrefixText(c *check.C) {
	t := s.Init(c)

	test := func(p, prefix string, hasPrefix bool) {
		t.CheckEquals(NewPath(p).HasPrefixText(prefix), hasPrefix)
	}

	test("", "", true)
	test("filename", "", true)
	test("", "x", false)
	test("/root", "/r", true)
	test("/root", "/root", true)
	test("/root", "/root/", false)
	test("/root", "root/", false)
}

func (s *Suite) Test_Path_HasPrefixPath(c *check.C) {
	t := s.Init(c)

	test := func(p, prefix Path, hasPrefix bool) {
		t.CheckEquals(p.HasPrefixPath(prefix), hasPrefix)
	}

	test("", "", true)
	test("filename", "", false)
	test("", "x", false)
	test("/root", "/r", false)
	test("/root", "/root", true)

	// Even though the textual representation of the prefix is longer than
	// the path. The trailing slash marks the path as a directory, and
	// there are only a few cases where the difference matters, such as
	// in rsync and mkdir.
	test("/root", "/root/", true)

	test("/root/", "/root", true)
	test("/root/", "root", false)
	test("/root/subdir", "/root", true)
	test("filename", ".", true)
	test("filename", "./filename", true)
	test("filename", "./file", false)
	test("filename", "./filename/sub", false)
	test("/anything", ".", false)
}

func (s *Suite) Test_Path_ContainsText(c *check.C) {
	t := s.Init(c)

	test := func(p Path, text string, contains bool) {
		t.CheckEquals(p.ContainsText(text), contains)
	}

	test("", "", true)
	test("filename", "", true)
	test("filename", ".", false)
	test("a.b", ".", true)
	test("..", ".", true)
	test("", "x", false)
	test("/root", "/r", true)
	test("/root", "/root", true)
	test("/root", "/root/", false)
	test("/root", "root/", false)
	test("/root", "ro", true)
	test("/root", "ot", true)
}

func (s *Suite) Test_Path_ContainsPath(c *check.C) {
	t := s.Init(c)

	test := func(p, sub Path, contains bool) {
		t.CheckEquals(p.ContainsPath(sub), contains)
	}

	test("", "", true)          // It doesn't make sense to search for empty paths.
	test(".", "", false)        // It doesn't make sense to search for empty paths.
	test("filename", ".", true) // Every relative path contains "." implicitly at the beginning
	test("a.b", ".", true)
	test("..", ".", true)
	test("filename", "", false)
	test("filename", "filename", true)
	test("a/b/c", "a", true)
	test("a/b/c", "b", true)
	test("a/b/c", "c", true)
	test("a/b/c", "a/b", true)
	test("a/b/c", "b/c", true)
	test("a/b/c", "a/b/c", true)
	test("aa/b/c", "a", false)
	test("a/bb/c", "b", false)
	test("a/bb/c", "b/c", false)
	test("mk/fetch/fetch.mk", "mk", true)
	test("category/package/../../wip/mk/../..", "mk", true)

	test("a", "a", true)
	test("a", "b", false)
	test("a", "A", false)
	test("a/b/c", "a", true)
	test("a/b/c", "b", true)
	test("a/b/c", "c", true)
	test("a/b/c", "a/b", true)
	test("a/b/c", "b/c", true)
	test("a/b/c", "a/b/c", true)
	test("aa/bb/cc", "a/b", false)
	test("aa/bb/cc", "a/bb", false)
	test("aa/bb/cc", "aa/b", false)
	test("aa/bb/cc", "aa/bb", true)
	test("aa/bb/cc", "a", false)
	test("aa/bb/cc", "b", false)
	test("aa/bb/cc", "c", false)
}

func (s *Suite) Test_Path_ContainsPathCanonical(c *check.C) {
	t := s.Init(c)

	test := func(p, sub Path, contains bool) {
		t.CheckEquals(p.ContainsPathCanonical(sub), contains)
	}

	test("", "", false)
	test(".", "", false)
	test("filename", "", false)
	test("filename", "filename", true)
	test("a/b/c", "a", true)
	test("a/b/c", "b", true)
	test("a/b/c", "c", true)
	test("a/b/c", "a/b", true)
	test("a/b/c", "b/c", true)
	test("a/b/c", "a/b/c", true)
	test("aa/b/c", "a", false)
	test("a/bb/c", "b", false)
	test("a/bb/c", "b/c", false)
	test("mk/fetch/fetch.mk", "mk", true)
	test("category/package/../../wip/mk", "mk", true)
	test("category/package/../../wip/mk/..", "mk", true) // FIXME
	test("category/package/../../wip/mk/../..", "mk", false)
}

func (s *Suite) Test_Path_HasSuffixText(c *check.C) {
	t := s.Init(c)

	test := func(p Path, suffix string, has bool) {
		t.CheckEquals(p.HasSuffixText(suffix), has)
	}

	test("", "", true)
	test("a/bb/c", "", true)
	test("a/bb/c", "c", true)
	test("a/bb/c", "/c", true)
	test("a/bb/c", "b/c", true)
	test("aa/b/c", "bb", false)
}

func (s *Suite) Test_Path_HasSuffixPath(c *check.C) {
	t := s.Init(c)

	test := func(p, suffix Path, has bool) {
		t.CheckEquals(p.HasSuffixPath(suffix), has)
	}

	test("", "", true)
	test("a/bb/c", "", false)
	test("a/bb/c", "c", true)
	test("a/bb/c", "/c", false)
	test("a/bb/c", "b/c", false)
	test("aa/b/c", "bb", false)
}

func (s *Suite) Test_Path_HasBase(c *check.C) {
	t := s.Init(c)

	test := func(p Path, suffix string, hasBase bool) {
		t.CheckEquals(p.HasBase(suffix), hasBase)
	}

	test("dir/file", "e", false)
	test("dir/file", "file", true)
	test("dir/file", "file.ext", false)
	test("dir/file", "/file", false)
	test("dir/file", "dir/file", false)
}

func (s *Suite) Test_Path_TrimSuffix(c *check.C) {
	t := s.Init(c)

	test := func(p Path, suffix string, result Path) {
		t.CheckEquals(p.TrimSuffix(suffix), result)
	}

	test("dir/file", "e", "dir/fil")
	test("dir/file", "file", "dir/")
	test("dir/file", "/file", "dir")
	test("dir/file", "dir/file", "")
	test("dir/file", "subdir/file", "dir/file")
}

func (s *Suite) Test_Path_Replace(c *check.C) {
	t := s.Init(c)

	test := func(p Path, from, to string, result Path) {
		t.CheckEquals(p.Replace(from, to), result)
	}

	test("dir/file", "dir", "other", "other/file")
	test("dir/file", "r", "sk", "disk/file")
	test("aaa/file", "a", "sub/", "sub/sub/sub//file")
}

func (s *Suite) Test_Path_JoinClean(c *check.C) {
	t := s.Init(c)

	test := func(p Path, suffix Path, result Path) {
		t.CheckEquals(p.JoinClean(suffix), result)
	}

	test("dir", "file", "dir/file")
	test("dir", "///file", "dir/file")
	test("dir/./../dir/", "///file", "dir/file")
	test("dir", "..", ".")
}

func (s *Suite) Test_Path_JoinNoClean(c *check.C) {
	t := s.Init(c)

	test := func(p, suffix Path, result Path) {
		t.CheckEquals(p.JoinNoClean(suffix), result)
	}

	test("dir", "file", "dir/file")
	test("dir", "///file", "dir////file")
	test("dir/./../dir/", "///file", "dir/./../dir/////file")
	test("dir", "..", "dir/..")
}

func (s *Suite) Test_Path_Clean(c *check.C) {
	t := s.Init(c)

	test := func(p, result Path) {
		t.CheckEquals(p.Clean(), result)
	}

	test("", ".")
	test(".", ".")
	test("./././", ".")
	test("a/bb///../c", "a/c")
}

func (s *Suite) Test_Path_CleanDot(c *check.C) {
	t := s.Init(c)

	test := func(p, result Path) {
		t.CheckEquals(p.CleanDot(), result)
	}

	test("", "")
	test(".", ".")
	test("./././", ".")
	test("a/bb///../c", "a/bb/../c")
	test("./filename", "filename")
	test("/absolute", "/absolute")
	test("/usr/pkgsrc/wip/package", "/usr/pkgsrc/wip/package")
	test("/usr/pkgsrc/wip/package/../mk/git-package.mk", "/usr/pkgsrc/wip/package/../mk/git-package.mk")
}

func (s *Suite) Test_Path_CleanPath(c *check.C) {
	t := s.Init(c)

	test := func(from, to Path) {
		t.CheckEquals(from.CleanPath(), to)
	}

	test("simple/path", "simple/path")
	test("/absolute/path", "/absolute/path")

	// Single dot components are removed, unless it's the only component of the path.
	test("./././.", ".")
	test("./././", ".")
	test("dir/multi/././/file", "dir/multi/file")
	test("dir/", "dir")

	test("dir/", "dir")

	// Components like aa/bb/../.. are removed, but not in the initial part of the path,
	// and only if they are not followed by another "..".
	test("dir/../dir/../dir/../dir/subdir/../../Makefile", "dir/../dir/../dir/../Makefile")
	test("111/222/../../333/444/../../555/666/../../777/888/9", "111/222/../../777/888/9")
	test("1/2/3/../../4/5/6/../../7/8/9/../../../../10", "1/2/3/../../4/7/8/9/../../../../10")
	test("cat/pkg.v1/../../cat/pkg.v2/Makefile", "cat/pkg.v1/../../cat/pkg.v2/Makefile")
	test("aa/../../../../../a/b/c/d", "aa/../../../../../a/b/c/d")
	test("aa/bb/../../../../a/b/c/d", "aa/bb/../../../../a/b/c/d")
	test("aa/bb/cc/../../../a/b/c/d", "aa/bb/cc/../../../a/b/c/d")
	test("aa/bb/cc/dd/../../a/b/c/d", "aa/bb/a/b/c/d")
	test("aa/bb/cc/dd/ee/../a/b/c/d", "aa/bb/cc/dd/ee/../a/b/c/d")
	test("../../../../../a/b/c/d", "../../../../../a/b/c/d")
	test("aa/../../../../a/b/c/d", "aa/../../../../a/b/c/d")
	test("aa/bb/../../../a/b/c/d", "aa/bb/../../../a/b/c/d")
	test("aa/bb/cc/../../a/b/c/d", "aa/bb/cc/../../a/b/c/d")
	test("aa/bb/cc/dd/../a/b/c/d", "aa/bb/cc/dd/../a/b/c/d")
	test("aa/../cc/../../a/b/c/d", "aa/../cc/../../a/b/c/d")

	// The initial 2 components of the path are typically category/package, when
	// pkglint is called from the pkgsrc top-level directory.
	// This path serves as the context and therefore is always kept.
	test("aa/bb/../../cc/dd/../../ee/ff", "aa/bb/../../ee/ff")
	test("aa/bb/../../cc/dd/../..", "aa/bb/../..")
	test("aa/bb/cc/dd/../..", "aa/bb")
	test("aa/bb/../../cc/dd/../../ee/ff/buildlink3.mk", "aa/bb/../../ee/ff/buildlink3.mk")
	test("./aa/bb/../../cc/dd/../../ee/ff/buildlink3.mk", "aa/bb/../../ee/ff/buildlink3.mk")

	test("../.", "..")
	test("../././././././.", "..")
	test(".././././././././", "..")
}

func (s *Suite) Test_Path_IsAbs(c *check.C) {
	t := s.Init(c)

	test := func(p Path, abs bool) {
		t.CheckEquals(p.IsAbs(), abs)
	}

	test("", false)
	test(".", false)
	test("a/b", false)
	test("/a", true)
	test("C:/", runtime.GOOS == "windows")
	test("c:/", runtime.GOOS == "windows")
}

func (s *Suite) Test_Path_Rel(c *check.C) {
	t := s.Init(c)

	base := NewPath(".")
	abc := NewPath("a/b/c")
	defFile := NewPath("d/e/f/file")

	t.CheckEquals(abc.Rel(defFile), NewPath("../../../d/e/f/file"))
	t.CheckEquals(base.Rel(base), NewPath("."))
	t.CheckEquals(abc.Rel(base), NewPath("../../../."))
}

func (s *Suite) Test_Path_Rename(c *check.C) {
	t := s.Init(c)

	f := t.CreateFileLines("filename.old",
		"line 1")
	t.CheckEquals(f.IsFile(), true)
	dst := NewPath(f.TrimSuffix(".old").String() + ".new")

	err := f.Rename(dst)

	assertNil(err, "Rename")
	t.CheckEquals(f.IsFile(), false)
	t.CheckFileLines("filename.new",
		"line 1")
}

func (s *Suite) Test_Path_Lstat(c *check.C) {
	t := s.Init(c)

	testDir := func(f Path, isDir bool) {
		st, err := f.Lstat()
		assertNil(err, "Lstat")
		t.CheckEquals(st.Mode()&os.ModeDir != 0, isDir)
	}

	t.CreateFileLines("subdir/file")
	t.CreateFileLines("file")

	testDir(t.File("subdir"), true)
	testDir(t.File("file"), false)
}

func (s *Suite) Test_Path_Stat(c *check.C) {
	t := s.Init(c)

	testDir := func(f Path, isDir bool) {
		st, err := f.Stat()
		assertNil(err, "Stat")
		t.CheckEquals(st.Mode()&os.ModeDir != 0, isDir)
	}

	t.CreateFileLines("subdir/file")
	t.CreateFileLines("file")

	testDir(t.File("subdir"), true)
	testDir(t.File("file"), false)
}

func (s *Suite) Test_Path_Exists(c *check.C) {
	t := s.Init(c)

	test := func(f Path, exists bool) {
		t.CheckEquals(f.Exists(), exists)
	}

	t.CreateFileLines("subdir/file")
	t.CreateFileLines("file")

	test(t.File("subdir"), true)
	test(t.File("file"), true)
	test(t.File("enoent"), false)
}

func (s *Suite) Test_Path_IsFile(c *check.C) {
	t := s.Init(c)

	t.CreateFileLines("dir/file")

	t.CheckEquals(t.File("nonexistent").IsFile(), false)
	t.CheckEquals(t.File("dir").IsFile(), false)
	t.CheckEquals(t.File("dir/nonexistent").IsFile(), false)
	t.CheckEquals(t.File("dir/file").IsFile(), true)
}

func (s *Suite) Test_Path_IsDir(c *check.C) {
	t := s.Init(c)

	t.CreateFileLines("dir/file")

	t.CheckEquals(t.File("nonexistent").IsDir(), false)
	t.CheckEquals(t.File("dir").IsDir(), true)
	t.CheckEquals(t.File("dir/nonexistent").IsDir(), false)
	t.CheckEquals(t.File("dir/file").IsDir(), false)
}

func (s *Suite) Test_Path_Chmod(c *check.C) {
	t := s.Init(c)

	testWritable := func(f Path, writable bool) {
		lstat, err := f.Lstat()
		assertNil(err, "Lstat")
		t.CheckEquals(lstat.Mode().Perm()&0200 != 0, writable)
	}

	f := t.CreateFileLines("file")
	testWritable(f, true)

	err := f.Chmod(0444)
	assertNil(err, "Chmod")

	testWritable(f, false)
}

func (s *Suite) Test_Path_ReadDir(c *check.C) {
	t := s.Init(c)

	t.CreateFileLines("subdir/file")
	t.CreateFileLines("file")
	t.CreateFileLines("CVS/Entries")
	t.CreateFileLines(".git/info/exclude")

	infos, err := t.File(".").ReadDir()

	assertNil(err, "ReadDir")
	var names []string
	for _, info := range infos {
		names = append(names, info.Name())
	}

	t.CheckDeepEquals(names, []string{".git", "CVS", "file", "subdir"})
}

func (s *Suite) Test_Path_Open(c *check.C) {
	t := s.Init(c)

	t.CreateFileLines("filename",
		"line 1",
		"line 2")

	f, err := t.File("filename").Open()

	assertNil(err, "Open")
	defer func() { assertNil(f.Close(), "Close") }()
	var sb strings.Builder
	n, err := io.Copy(&sb, f)
	assertNil(err, "Copy")
	t.CheckEquals(n, int64(14))
	t.CheckEquals(sb.String(), "line 1\nline 2\n")
}

func (s *Suite) Test_Path_ReadString(c *check.C) {
	t := s.Init(c)

	t.CreateFileLines("filename",
		"line 1",
		"line 2")

	text, err := t.File("filename").ReadString()

	assertNil(err, "ReadString")
	t.CheckEquals(text, "line 1\nline 2\n")
}

func (s *Suite) Test_Path_WriteString(c *check.C) {
	t := s.Init(c)

	err := t.File("filename").WriteString("line 1\nline 2\n")

	assertNil(err, "WriteString")
	t.CheckFileLines("filename",
		"line 1",
		"line 2")
}

func (s *Suite) Test_NewPkgsrcPath(c *check.C) {
	t := s.Init(c)

	p := NewPkgsrcPath("category/package")

	t.CheckEquals(p.AsPath().Dir(), NewPath("category"))
}

func (s *Suite) Test_PkgsrcPath_String(c *check.C) {
	t := s.Init(c)

	p := NewPkgsrcPath("any string..././")

	str := p.String()

	// No normalization takes place because it is typically not needed.
	t.CheckEquals(str, "any string..././")
}

func (s *Suite) Test_PkgsrcPath_AsPath(c *check.C) {
	t := s.Init(c)

	pp := NewPkgsrcPath("./category/package/Makefile")

	p := pp.AsPath()

	t.CheckEquals(p.String(), "./category/package/Makefile")
}

func (s *Suite) Test_PkgsrcPath_Dir(c *check.C) {
	t := s.Init(c)

	pp := NewPkgsrcPath("dir/base///.")

	dir := pp.Dir()

	t.CheckEquals(dir, NewPkgsrcPath("dir/base"))
}

func (s *Suite) Test_PkgsrcPath_Base(c *check.C) {
	t := s.Init(c)

	pp := NewPkgsrcPath("dir/base///.")

	base := pp.Base()

	t.CheckEquals(base, ".")
}

func (s *Suite) Test_PkgsrcPath_Count(c *check.C) {
	t := s.Init(c)

	pp := NewPkgsrcPath("./...////dir")

	count := pp.Count()

	t.CheckEquals(count, 2)
}

func (s *Suite) Test_PkgsrcPath_HasPrefixPath(c *check.C) {
	t := s.Init(c)

	pp := NewPkgsrcPath("./././///prefix/suffix")

	hasPrefixPath := pp.HasPrefixPath("prefix")

	t.CheckEquals(hasPrefixPath, true)
}

func (s *Suite) Test_PkgsrcPath_JoinNoClean(c *check.C) {
	t := s.Init(c)

	pp := NewPkgsrcPath("base///.")

	joined := pp.JoinNoClean("./../rel")

	t.CheckEquals(joined, NewPkgsrcPath("base///././../rel"))
}

func (s *Suite) Test_PkgsrcPath_JoinRel(c *check.C) {
	t := s.Init(c)

	pp := NewPkgsrcPath("base///.")

	joined := pp.JoinRel("./../rel")

	t.CheckEquals(joined, NewPkgsrcPath("base///././../rel"))
}

func (s *Suite) Test_NewRelPath(c *check.C) {
	t := s.Init(c)

	rel := NewRelPath("dir/file")

	t.CheckEquals(rel.String(), "dir/file")
}

func (s *Suite) Test_RelPath_AsPath(c *check.C) {
	t := s.Init(c)

	rel := NewRelPath("relative")

	path := rel.AsPath()

	t.CheckEquals(path.String(), "relative")
}

func (s *Suite) Test_RelPath_String(c *check.C) {
	t := s.Init(c)

	rel := NewRelPath(".///rel")

	str := rel.String()

	t.CheckEquals(str, ".///rel")
}

func (s *Suite) Test_RelPath_Dir(c *check.C) {
	t := s.Init(c)

	rel := NewRelPath("./dir////./file")

	dir := rel.Dir()

	t.CheckEquals(dir, NewRelPath("dir"))
}

func (s *Suite) Test_RelPath_HasBase(c *check.C) {
	t := s.Init(c)

	test := func(rel RelPath, base string, hasBase bool) {
		t.CheckEquals(rel.HasBase(base), hasBase)
	}

	test("./dir/Makefile", "Makefile", true)
	test("./dir/Makefile", "Make", false)
	test("./dir/Makefile", "file", false)
	test("./dir/Makefile", "dir/Makefile", false)
}

func (s *Suite) Test_RelPath_Parts(c *check.C) {
	t := s.Init(c)

	rel := NewRelPath("./dir/.///base")

	parts := rel.Parts()

	t.CheckDeepEquals(parts, []string{"dir", "base"})
}

func (s *Suite) Test_RelPath_CleanPath(c *check.C) {
	t := s.Init(c)

	rel := NewRelPath("a/b/../../c/d/../../e/../f")

	cleaned := rel.CleanPath()

	t.CheckEquals(cleaned, NewRelPath("a/b/../../e/../f"))
}

func (s *Suite) Test_RelPath_JoinNoClean(c *check.C) {
	t := s.Init(c)

	rel := NewRelPath("basedir/.//")

	joined := rel.JoinNoClean("./other")

	t.CheckEquals(joined, NewRelPath("basedir/.///./other"))
}

func (s *Suite) Test_RelPath_Replace(c *check.C) {
	t := s.Init(c)

	rel := NewRelPath("dir/subdir/file")

	replaced := rel.Replace("/", ":")

	t.CheckEquals(replaced, NewRelPath("dir:subdir:file"))
}

func (s *Suite) Test_RelPath_HasPrefixPath(c *check.C) {
	t := s.Init(c)

	rel := NewRelPath("dir/subdir/file")

	t.CheckEquals(rel.HasPrefixPath("dir"), true)
	t.CheckEquals(rel.HasPrefixPath("dir/sub"), false)
	t.CheckEquals(rel.HasPrefixPath("subdir"), false)
}

func (s *Suite) Test_RelPath_ContainsPath(c *check.C) {
	t := s.Init(c)

	rel := NewRelPath("dir/subdir/file")

	t.CheckEquals(rel.ContainsPath("dir"), true)
	t.CheckEquals(rel.ContainsPath("dir/sub"), false)
	t.CheckEquals(rel.ContainsPath("subdir"), true)
}

func (s *Suite) Test_RelPath_ContainsText(c *check.C) {
	t := s.Init(c)

	rel := NewRelPath("dir/subdir/file")

	t.CheckEquals(rel.ContainsText("dir"), true)
	t.CheckEquals(rel.ContainsText("dir/sub"), true)
	t.CheckEquals(rel.ContainsText("subdir"), true)
	t.CheckEquals(rel.ContainsText("super"), false)
}

func (s *Suite) Test_RelPath_HasSuffixPath(c *check.C) {
	t := s.Init(c)

	rel := NewRelPath("dir/subdir/file")

	t.CheckEquals(rel.HasSuffixPath("file"), true)
	t.CheckEquals(rel.HasSuffixPath("subdir/file"), true)
	t.CheckEquals(rel.HasSuffixPath("subdir"), false)
}
