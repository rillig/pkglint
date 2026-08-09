package pkglint

import (
	"io"
	"os"
	"runtime"
	"strings"
	"testing"
)

func Test_NewPath(tt *testing.T) {
	t := NewTester(tt)
	defer t.Done()

	t.CheckEquals(NewPath("filename"), NewPath("filename"))
	t.CheckEquals(NewPath("\\"), NewPath("\\"))
	if NewPath("\\") == NewPath("/") {
		t.Errorf("path separators are unexpectedly normalized")
	}
}

func Test_Path_String(t *testing.T) {
	tests := []struct {
		path string
	}{
		{""},
		{"filename"},
		{"a/b"},

		// No normalization takes place in Path(string).
		// That's what NewPathSlash is for.
		{"c\\d"},
	}
	for _, test := range tests {
		t.Run(test.path, func(t *testing.T) {
			p := Path(test.path)
			have := p.String()
			if have != test.path || have != string(p) {
				t.Errorf("want %#v, have %#v", test.path, have)
			}
		})
	}
}

func Test_Path_GoString(t *testing.T) {
	tests := []struct {
		path string
		want string
	}{
		{"", "\"\""},
		{"filename", "\"filename\""},
		{"a/b", "\"a/b\""},
		{"c\\d", "\"c\\\\d\""},
	}
	for _, test := range tests {
		t.Run(test.path, func(t *testing.T) {
			p := Path(test.path)
			have := p.GoString()
			if have != test.want {
				t.Errorf("want %#v, have %#v for %#v",
					test.want, have, test.path)
			}
		})
	}
}

func Test_Path_IsEmpty(t *testing.T) {
	tests := []struct {
		path string
		want bool
	}{
		{"", true},
		{".", false},
		{"/", false},
	}
	for _, test := range tests {
		t.Run(test.path, func(t *testing.T) {
			p := Path(test.path)
			have := p.IsEmpty()
			if have != test.want {
				t.Errorf("want %#v, have %#v for %#v",
					test.want, have, test.path)
			}
		})
	}
}

func Test_Path_Dir(t *testing.T) {
	tests := []struct {
		path string
		want Path
	}{
		{"", "."},
		// Dir only works textual, it does not know about "..".
		{".", "."},
		// Dir only works textual, it does not know about "..".
		{"dir/..", "dir"},
		{"././././", "."},
		{"/", "/"}, // ?
		{"/root", "/"},
		{"filename", "."},
		{"dir/filename", "dir"},
		{"dir/./file", "dir"},
		{"./file", "."},
	}
	for _, test := range tests {
		t.Run(test.path, func(t *testing.T) {
			p := Path(test.path)
			have := p.Dir()
			if have != test.want {
				t.Errorf("want %#v, have %#v for %#v",
					test.want, have, test.path)
			}
		})
	}
}

func Test_Path_Base(t *testing.T) {
	tests := []struct {
		path string
		want RelPath
	}{
		// That's a bit surprising
		{"", "."},
		// Base only works textual, it does not know about "..".
		{".", "."},
		// Base only works textual, it does not know about "..".
		{"dir/..", ".."},
		{"././././", "."},
		// {"/", "/"}, // Panics because "/" is not relative.
		{"/root", "root"},
		{"filename", "filename"},
		{"dir/filename", "filename"},
		{"dir/./file", "file"},
		{"./file", "file"},
		{"dir/filename\\with\\backslash", "filename\\with\\backslash"},
	}
	for _, test := range tests {
		t.Run(test.path, func(t *testing.T) {
			p := Path(test.path)
			have := p.Base()
			if have != test.want {
				t.Errorf("want %#v, have %#v for %#v",
					test.want, have, test.path)
			}
		})
	}
}

func Test_Path_Split(t *testing.T) {
	tests := []struct {
		path     string
		wantDir  Path
		wantBase string
	}{
		{"", "", ""},
		{"././././", "././././", ""},
		{"/root", "/", "root"},
		{"filename", "", "filename"},
		{"dir/filename", "dir/", "filename"},
		{"dir/filename\\with\\backslash", "dir/", "filename\\with\\backslash"},
		{"/", "/", ""},
	}
	for _, test := range tests {
		t.Run(test.path, func(t *testing.T) {
			p := NewPath(test.path)
			haveDir, haveBase := p.Split()
			if haveDir != test.wantDir || haveBase != test.wantBase {
				t.Errorf("want %#v|%#v, have %#v|%#v for %#v",
					test.wantDir, test.wantBase, haveDir, haveBase, test.path)
			}
		})
	}
}

func Test_Path_Parts(tt *testing.T) {
	t := NewTester(tt)
	defer t.Done()

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

func Test_Path_Count(tt *testing.T) {
	t := NewTester(tt)
	defer t.Done()

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

func Test_Path_HasPrefixText(tt *testing.T) {
	t := NewTester(tt)
	defer t.Done()

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

func Test_Path_HasPrefixPath(tt *testing.T) {
	t := NewTester(tt)
	defer t.Done()

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
	test("filename", "/filename", false)
	test("filename", ".", true)
	test("filename", "./filename", true)
	test("filename", "./file", false)
	test("filename", "./filename/sub", false)
	test("/anything", ".", false)
}

func Test_Path_ContainsText(tt *testing.T) {
	t := NewTester(tt)
	defer t.Done()

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

func Test_Path_ContainsPath(tt *testing.T) {
	t := NewTester(tt)
	defer t.Done()

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

func Test_Path_HasSuffixText(tt *testing.T) {
	t := NewTester(tt)
	defer t.Done()

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

func Test_Path_HasSuffixPath(tt *testing.T) {
	t := NewTester(tt)
	defer t.Done()

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

func Test_Path_HasBase(tt *testing.T) {
	t := NewTester(tt)
	defer t.Done()

	test := func(p Path, suffix string, hasBase bool) {
		t.CheckEquals(p.HasBase(suffix), hasBase)
	}

	test("dir/file", "e", false)
	test("dir/file", "file", true)
	test("dir/file", "file.ext", false)
	test("dir/file", "/file", false)
	test("dir/file", "dir/file", false)
}

func Test_Path_TrimSuffix(tt *testing.T) {
	t := NewTester(tt)
	defer t.Done()

	test := func(p Path, suffix string, result Path) {
		t.CheckEquals(p.TrimSuffix(suffix), result)
	}

	test("dir/file", "e", "dir/fil")
	test("dir/file", "file", "dir/")
	test("dir/file", "/file", "dir")
	test("dir/file", "dir/file", "")
	test("dir/file", "subdir/file", "dir/file")
}

func Test_Path_Replace(tt *testing.T) {
	t := NewTester(tt)
	defer t.Done()

	test := func(p Path, from, to string, result Path) {
		t.CheckEquals(p.Replace(from, to), result)
	}

	test("dir/file", "dir", "other", "other/file")
	test("dir/file", "r", "sk", "disk/file")
	test("aaa/file", "a", "sub/", "sub/sub/sub//file")
}

func Test_Path_JoinClean(tt *testing.T) {
	t := NewTester(tt)
	defer t.Done()

	test := func(p Path, rel RelPath, result Path) {
		t.CheckEquals(p.JoinClean(rel), result)
	}

	test("dir", "file", "dir/file")
	test("dir", "///file", "dir/file")
	test("dir/./../dir/", "///file", "dir/file")
	test("dir", "..", ".")
}

func Test_Path_JoinNoClean(tt *testing.T) {
	t := NewTester(tt)
	defer t.Done()

	test := func(p Path, rel RelPath, result Path) {
		t.CheckEquals(p.JoinNoClean(rel), result)
	}

	test("dir", "file", "dir/file")
	test("dir", "///file", "dir////file")
	test("dir/./../dir/", "///file", "dir/./../dir/////file")
	test("dir", "..", "dir/..")
	test(".", "sub", "./sub")
}

func Test_Path_Clean(tt *testing.T) {
	t := NewTester(tt)
	defer t.Done()

	test := func(p, result Path) {
		t.CheckEquals(p.Clean(), result)
	}

	test("", ".")
	test(".", ".")
	test("./././", ".")
	test("a/bb///../c", "a/c")
}

func Test_Path_CleanDot(tt *testing.T) {
	t := NewTester(tt)
	defer t.Done()

	test := func(p, result Path) {
		t.CheckEquals(p.CleanDot(), result)
	}

	test("", "")
	test(".", ".")
	test("./././", ".")
	test("dir/", "dir/") // TODO: Or maybe "dir/."?
	test("a/bb///../c", "a/bb/../c")
	test("./filename", "filename")
	test("/absolute", "/absolute")
	test("/usr/pkgsrc/wip/package", "/usr/pkgsrc/wip/package")
	test("/usr/pkgsrc/wip/package/../mk/git-package.mk", "/usr/pkgsrc/wip/package/../mk/git-package.mk")
	test("a//b", "a/b")
}

func Test_Path_CleanPath(t *testing.T) {
	tests := []struct {
		path string
		want Path
	}{
		{"simple/path", "simple/path"},
		{"/absolute/path", "/absolute/path"},

		// Single dot components are removed, unless it's the only component of the path.
		{"./././.", "."},
		{"./././", "."},
		{"dir/multi/././/file", "dir/multi/file"},
		{"dir/", "dir"},

		// Components like aa/bb/../.. are removed, but not in the initial part of the path,
		// and only if they are not followed by another "..".
		{"dir/../dir/../dir/../dir/subdir/../../Makefile", "dir/../dir/../dir/../Makefile"},
		{"111/222/../../333/444/../../555/666/../../777/888/9", "111/222/../../777/888/9"},
		{"1/2/3/../../4/5/6/../../7/8/9/../../../../10", "1/2/3/../../4/7/8/9/../../../../10"},
		{"cat/pkg.v1/../../cat/pkg.v2/Makefile", "cat/pkg.v1/../../cat/pkg.v2/Makefile"},
		{"aa/../../../../../a/b/c/d", "aa/../../../../../a/b/c/d"},
		{"aa/bb/../../../../a/b/c/d", "aa/bb/../../../../a/b/c/d"},
		{"aa/bb/cc/../../../a/b/c/d", "aa/bb/cc/../../../a/b/c/d"},
		{"aa/bb/cc/dd/../../a/b/c/d", "aa/bb/a/b/c/d"},
		{"aa/bb/cc/dd/ee/../a/b/c/d", "aa/bb/cc/dd/ee/../a/b/c/d"},
		{"../../../../../a/b/c/d", "../../../../../a/b/c/d"},
		{"aa/../../../../a/b/c/d", "aa/../../../../a/b/c/d"},
		{"aa/bb/../../../a/b/c/d", "aa/bb/../../../a/b/c/d"},
		{"aa/bb/cc/../../a/b/c/d", "aa/bb/cc/../../a/b/c/d"},
		{"aa/bb/cc/dd/../a/b/c/d", "aa/bb/cc/dd/../a/b/c/d"},
		{"aa/../cc/../../a/b/c/d", "aa/../cc/../../a/b/c/d"},

		// The initial 2 components of the path are typically category/package, when
		// pkglint is called from the pkgsrc top-level directory.
		// This path serves as the context and therefore is always kept.
		{"aa/bb/../../cc/dd/../../ee/ff", "aa/bb/../../ee/ff"},
		{"aa/bb/../../cc/dd/../..", "aa/bb/../.."},
		{"aa/bb/cc/dd/../..", "aa/bb"},
		{"aa/bb/../../cc/dd/../../ee/ff/buildlink3.mk", "aa/bb/../../ee/ff/buildlink3.mk"},
		{"./aa/bb/../../cc/dd/../../ee/ff/buildlink3.mk", "aa/bb/../../ee/ff/buildlink3.mk"},

		{"../.", ".."},
		{"../././././././.", ".."},
		{".././././././././", ".."},

		{
			"x11/kde-runtime4/../../misc/kdepimlibs4/../../databases/openldap-client/buildlink3.mk",
			"x11/kde-runtime4/../../databases/openldap-client/buildlink3.mk"},
	}
	for _, test := range tests {
		t.Run(test.path, func(t *testing.T) {
			p := NewPath(test.path)
			have := p.CleanPath()
			if have != test.want {
				t.Errorf("want %#v, have %#v for %#v",
					test.want, have, test.path)
			}
		})
	}
}

func Test_Path_IsAbs(t *testing.T) {
	tests := []struct {
		path string
		want bool
	}{
		{"", false},
		{".", false},
		{"a/b", false},
		{"/a", true},
		{"C:/", true},
		{"c:/", true},
		{"c::", false},
		{"c:relative", false},
	}
	for _, test := range tests {
		t.Run(test.path, func(t *testing.T) {
			p := NewPath(test.path)
			have := p.IsAbs()
			if have != test.want {
				t.Errorf("want %#v, have %#v for %#v",
					test.want, have, test.path)
			}
		})
	}
}

func Test_Path_Rel(t *testing.T) {
	tests := []struct {
		base  string
		other Path
		want  RelPath
	}{
		{"a/b/c", "d/e/f/file", "../../../d/e/f/file"},
		{".", ".", "."},

		// The trailing dot marks the difference between a file and a directory.
		// This is the same behavior as with filepath.Rel.
		{"a/b/c", ".", "../../../."},

		// Intermediate dotdot components are removed.
		{"a/../b", "c/../d", "../d"},

		{".", "dir/file", "dir/file"},
		// XXX: maybe the /. is missing at the end
		{".", "dir/subdir/", "dir/subdir"},
		// XXX: maybe the /. is missing at the end
		{".", "dir/subdir/.", "dir/subdir"},
	}
	for _, test := range tests {
		t.Run(test.base+","+test.other.String(), func(t *testing.T) {
			p := NewPath(test.base)
			have := p.Rel(test.other)
			if have != test.want {
				t.Errorf("want %#v, have %#v for %#v + %#v",
					test.want, have, test.base, test.other)
			}
		})
	}
}

func Test_NewCurrPath(tt *testing.T) {
	t := NewTester(tt)
	defer t.Done()

	curr := NewCurrPath("dir/.///file")

	t.CheckEquals(curr.String(), "dir/.///file")
}

func Test_NewCurrPathString(tt *testing.T) {
	t := NewTester(tt)
	defer t.Done()

	curr := NewCurrPathString("dir/.///file")

	t.CheckEquals(curr.String(), "dir/.///file")
}

func Test_NewCurrPathSlash(tt *testing.T) {
	t := NewTester(tt)
	defer t.Done()

	test := func(path, curr string) {
		t.CheckEquals(NewCurrPathSlash(path).String(), curr)
	}

	test("filename", "filename")
	test("dir/.///file", "dir/.///file")
}

func Test_NewCurrPathSlash__windows(tt *testing.T) {
	t := NewTester(tt)
	defer t.Done()

	test := func(path, currWindows, currOther string) {
		t.CheckEquals(
			NewCurrPathSlash(path).String(),
			condStr(runtime.GOOS == "windows", currWindows, currOther))
	}

	test("\\", "/", "\\")
	test("dir\\.\\\\\\file", "dir/.///file", "dir\\.\\\\\\file")
}

func Test_CurrPath_GoString(tt *testing.T) {
	t := NewTester(tt)
	defer t.Done()

	test := func(p CurrPath, str string) {
		t.CheckEquals(p.GoString(), str)
	}

	// Tabs in filenames are rare, probably typos.
	test("dir/file\t", "\"dir/file\\t\"")
}

func Test_CurrPath_String(tt *testing.T) {
	t := NewTester(tt)
	defer t.Done()

	test := func(p CurrPath, str string) {
		t.CheckEquals(p.String(), str)
	}

	// Tabs in filenames are rare, probably typos.
	test("dir/file\t", "dir/file\t")
}

func Test_CurrPath_AsPath(tt *testing.T) {
	t := NewTester(tt)
	defer t.Done()

	test := func(curr CurrPath, asPath Path) {
		t.CheckEquals(curr.AsPath(), asPath)
	}

	// Tabs in filenames are rare, probably typos.
	test("dir/file\t", "dir/file\t")
}

func Test_CurrPath_IsEmpty(tt *testing.T) {
	t := NewTester(tt)
	defer t.Done()

	test := func(curr CurrPath, isEmpty bool) {
		t.CheckEquals(curr.IsEmpty(), isEmpty)
	}

	test("", true)
	test(".", false)
	test("/", false)
}

func Test_CurrPath_Dir(tt *testing.T) {
	t := NewTester(tt)
	defer t.Done()

	test := func(curr, dir CurrPath) {
		t.CheckEquals(curr.Dir(), dir)
	}

	test("./dir/../dir///./file", "./dir/../dir")
}

func Test_CurrPath_Base(tt *testing.T) {
	t := NewTester(tt)
	defer t.Done()

	test := func(curr CurrPath, base RelPath) {
		t.CheckEquals(curr.Base(), base)
	}

	test("dir/file", "file")
}

func Test_CurrPath_Split(tt *testing.T) {
	t := NewTester(tt)
	defer t.Done()

	test := func(curr, dir CurrPath, base string) {
		actualDir, actualBase := curr.Split()
		t.CheckEquals(actualDir, dir)
		t.CheckEquals(actualBase, base)
	}

	test("dir/file", "dir/", "file")
}

func Test_CurrPath_Parts(tt *testing.T) {
	t := NewTester(tt)
	defer t.Done()

	test := func(curr CurrPath, parts ...string) {
		t.CheckDeepEquals(curr.Parts(), parts)
	}

	test("dir/file", "dir", "file")
}

func Test_CurrPath_IsAbs(tt *testing.T) {
	t := NewTester(tt)
	defer t.Done()

	test := func(curr CurrPath, isAbs bool) {
		t.CheckDeepEquals(curr.IsAbs(), isAbs)
	}

	test("/", true)
	test("./", false)
	test("C:/", true)
}

func Test_CurrPath_HasPrefixPath(tt *testing.T) {
	t := NewTester(tt)
	defer t.Done()

	test := func(curr, prefix CurrPath, hasPrefix bool) {
		t.CheckEquals(curr.HasPrefixPath(prefix), hasPrefix)
	}

	test("dir/file", "dir", true)
	test("dir/file", "file", false)
	test("dir", ".", true)
}

func Test_CurrPath_ContainsPath(tt *testing.T) {
	t := NewTester(tt)
	defer t.Done()

	test := func(curr CurrPath, sub Path, hasPrefix bool) {
		t.CheckEquals(curr.ContainsPath(sub), hasPrefix)
	}

	test("dir/file", "dir", true)
	test("dir/file", "file", true)
	test("dir/file", "fi", false)
	test("dir", ".", true)
}

func Test_CurrPath_ContainsText(tt *testing.T) {
	t := NewTester(tt)
	defer t.Done()

	test := func(curr CurrPath, sub string, hasPrefix bool) {
		t.CheckEquals(curr.ContainsText(sub), hasPrefix)
	}

	test("dir/file", "dir", true)
	test("dir/file", "r/f", true)
}

func Test_CurrPath_HasSuffixPath(tt *testing.T) {
	t := NewTester(tt)
	defer t.Done()

	test := func(curr CurrPath, suffix Path, hasPrefix bool) {
		t.CheckEquals(curr.HasSuffixPath(suffix), hasPrefix)
	}

	test("dir/file", "dir", false)
	test("dir/file", "file", true)
	test("dir/file", "le", false)

	// In contrast to HasPrefixPath, it doesn't really make sense to
	// ask whether a path ends with the current directory.
	test("dir", ".", false)
}

func Test_CurrPath_HasSuffixText(tt *testing.T) {
	t := NewTester(tt)
	defer t.Done()

	test := func(curr CurrPath, suffix string, hasPrefix bool) {
		t.CheckEquals(curr.HasSuffixText(suffix), hasPrefix)
	}

	test("dir/file", "dir", false)
	test("dir/file", "file", true)
	test("dir/file", "le", true)
}

func Test_CurrPath_HasBase(tt *testing.T) {
	t := NewTester(tt)
	defer t.Done()

	test := func(curr CurrPath, base string, hasPrefix bool) {
		t.CheckEquals(curr.HasBase(base), hasPrefix)
	}

	test("dir/file", "dir", false)
	test("dir/file", "file", true)
	test("dir/file", "le", false)
}

func Test_CurrPath_TrimSuffix(tt *testing.T) {
	t := NewTester(tt)
	defer t.Done()

	test := func(curr CurrPath, suffix string, trimmed CurrPath) {
		t.CheckEquals(curr.TrimSuffix(suffix), trimmed)
	}

	test("dir/file", "dir", "dir/file")
	test("dir/file", "file", "dir/")
	test("dir/file", "le", "dir/fi")
}

func Test_CurrPath_ReplaceSuffix(tt *testing.T) {
	t := NewTester(tt)
	defer t.Done()

	test := func(curr CurrPath, from, to string, replaced CurrPath) {
		t.CheckEquals(curr.ReplaceSuffix(from, to), replaced)
	}

	test("dir/file", "file", "subdir", "dir/subdir")

	// The path must actually end with the suffix, otherwise there is
	// the risk of creating unintended paths.
	t.ExpectAssert(
		func() { test("dir/file", "no-match", "anything", "dir/file") })
}

func Test_CurrPath_Clean(tt *testing.T) {
	t := NewTester(tt)
	defer t.Done()

	test := func(curr, cleaned CurrPath) {
		t.CheckEquals(curr.Clean(), cleaned)
	}

	test("dir/file", "dir/file")
	test("dir/.////../file", "file")
}

func Test_CurrPath_CleanDot(tt *testing.T) {
	t := NewTester(tt)
	defer t.Done()

	test := func(curr, cleaned CurrPath) {
		t.CheckEquals(curr.CleanDot(), cleaned)
	}

	test("dir/file", "dir/file")
	test("dir/.////../file", "dir/../file")
}

func Test_CurrPath_CleanPath(tt *testing.T) {
	t := NewTester(tt)
	defer t.Done()

	test := func(curr, cleaned CurrPath) {
		t.CheckEquals(curr.CleanPath(), cleaned)
	}

	test("a/b/../../c/d/../../e/../f", "a/b/../../e/../f")
}

func Test_CurrPath_JoinNoClean(tt *testing.T) {
	t := NewTester(tt)
	defer t.Done()

	test := func(curr CurrPath, rel RelPath, joined CurrPath) {
		t.CheckEquals(curr.JoinNoClean(rel), joined)
	}

	test("", "", "/")
	test(".", "file", "./file")
	test("dir", "subdir/file", "dir/subdir/file")
}

func Test_CurrPath_JoinClean(tt *testing.T) {
	t := NewTester(tt)
	defer t.Done()

	test := func(curr CurrPath, rel RelPath, joined CurrPath) {
		t.CheckEquals(curr.JoinClean(rel), joined)
	}

	test("", "", "")
	test(".", "./////file", "file")
	test("dir/./.", "../subdir/file", "subdir/file")
}

func Test_CurrPath_Rel(tt *testing.T) {
	t := NewTester(tt)
	defer t.Done()

	test := func(curr, rel CurrPath, result RelPath) {
		t.CheckEquals(curr.Rel(rel), result)
	}

	test("dir/subdir", "dir", "..")
	test("dir/subdir", "file", "../../file")
}

func Test_CurrPath_Rename(tt *testing.T) {
	t := NewTester(tt)
	defer t.Done()

	f := t.CreateFileLines("filename.old",
		"line 1")
	t.CheckEquals(f.IsFile(), true)
	dst := f.ReplaceSuffix(".old", ".new")

	err := f.Rename(dst)

	assertNil(err, "Rename")
	t.CheckEquals(f.IsFile(), false)
	t.CheckFileLines("filename.new",
		"line 1")
}

func Test_CurrPath_Lstat(tt *testing.T) {
	t := NewTester(tt)
	defer t.Done()

	test := func(f CurrPath, isDir bool) {
		st, err := f.Lstat()
		assertNil(err, "Lstat")
		t.CheckEquals(st.Mode()&os.ModeDir != 0, isDir)
	}

	t.CreateFileLines("subdir/file")
	t.CreateFileLines("file")

	test(t.File("subdir"), true)
	test(t.File("file"), false)
}

func Test_CurrPath_Stat(tt *testing.T) {
	t := NewTester(tt)
	defer t.Done()

	test := func(f CurrPath, isDir bool) {
		st, err := f.Stat()
		assertNil(err, "Stat")
		t.CheckEquals(st.Mode()&os.ModeDir != 0, isDir)
	}

	t.CreateFileLines("subdir/file")
	t.CreateFileLines("file")

	test(t.File("subdir"), true)
	test(t.File("file"), false)
}

func Test_CurrPath_Exists(tt *testing.T) {
	t := NewTester(tt)
	defer t.Done()

	test := func(f CurrPath, exists bool) {
		t.CheckEquals(f.Exists(), exists)
	}

	t.CreateFileLines("subdir/file")
	t.CreateFileLines("file")

	test(t.File("subdir"), true)
	test(t.File("file"), true)
	test(t.File("enoent"), false)
}

func Test_CurrPath_IsFile(tt *testing.T) {
	t := NewTester(tt)
	defer t.Done()

	t.CreateFileLines("dir/file")
	t.Chdir(".")

	test := func(curr CurrPath, isFile bool) {
		t.CheckEquals(curr.IsFile(), isFile)
	}

	test("nonexistent", false)
	test("dir", false)
	test("dir/nonexistent", false)
	test("dir/file", true)
}

func Test_CurrPath_IsDir(tt *testing.T) {
	t := NewTester(tt)
	defer t.Done()

	t.CreateFileLines("dir/file")
	t.Chdir(".")

	test := func(curr CurrPath, isFile bool) {
		t.CheckEquals(curr.IsDir(), isFile)
	}

	test("nonexistent", false)
	test("dir", true)
	test("dir/nonexistent", false)
	test("dir/file", false)
}

func Test_CurrPath_Chmod(tt *testing.T) {
	t := NewTester(tt)
	defer t.Done()

	testWritable := func(f CurrPath, writable bool) {
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

func Test_CurrPath_ReadDir(tt *testing.T) {
	t := NewTester(tt)
	defer t.Done()

	t.CreateFileLines("subdir/file")
	t.CreateFileLines("file")
	t.CreateFileLines("CVS/Entries")
	t.CreateFileLines(".git/info/exclude")
	t.Chdir(".")

	test := func(curr CurrPath, entryNames ...string) {
		entries, err := curr.ReadDir()
		assertNil(err, "ReadDir")

		var names []string
		for _, entry := range entries {
			names = append(names, entry.Name())
		}

		t.CheckDeepEquals(names, entryNames)
	}

	test(".",
		".git", "CVS", "file", "subdir")
	test("subdir",
		"file")
	test("CVS",
		"Entries")
}

func Test_CurrPath_ReadPaths(tt *testing.T) {
	t := NewTester(tt)
	defer t.Done()

	t.CreateFileLines("dir/subdir/file")
	t.CreateFileLines("dir/CVS/Entries")
	t.CreateFileLines("dir/file")
	t.Chdir(".")

	test := func(dir CurrPath, entries ...CurrPath) {
		t.CheckDeepEquals(dir.ReadPaths(), entries)
	}

	test(".",
		"dir")

	test("dir",
		"dir/file", "dir/subdir")
}

func Test_CurrPath_Open(tt *testing.T) {
	t := NewTester(tt)
	defer t.Done()

	t.CreateFileLines("filename",
		"line 1",
		"line 2")
	t.Chdir(".")

	test := func(curr CurrPath, content string) {
		f, err := curr.Open()
		assertNil(err, "Open")
		defer func() { assertNil(f.Close(), "Close") }()

		var sb strings.Builder
		n, err := io.Copy(&sb, f)
		assertNil(err, "Copy")

		t.CheckEquals(n, int64(len(content)))
		t.CheckEquals(sb.String(), content)
	}

	test("filename", "line 1\nline 2\n")
}

func Test_CurrPath_ReadString(tt *testing.T) {
	t := NewTester(tt)
	defer t.Done()

	t.Chdir(".")
	t.CreateFileLines("empty")
	t.CreateFileLines("filename",
		"line 1",
		"line 2")

	test := func(curr CurrPath, content string) {
		text, err := curr.ReadString()

		assertNil(err, "ReadString")
		t.CheckEquals(text, content)
	}

	test("empty", "")
	test("filename", "line 1\nline 2\n")
}

func Test_CurrPath_WriteString(tt *testing.T) {
	t := NewTester(tt)
	defer t.Done()

	t.Chdir(".")

	test := func(curr CurrPath, content string, lines ...string) {
		err := curr.WriteString(content)
		assertNil(err, "WriteString")

		t.CheckFileLines(NewRelPath(curr.AsPath()),
			lines...)
	}

	test("empty", "",
		nil...)

	test("filename", "line 1\nline 2\n",
		"line 1",
		"line 2")
}

func Test_NewPkgsrcPath(tt *testing.T) {
	t := NewTester(tt)
	defer t.Done()

	p := NewPkgsrcPath("category/package")

	t.CheckEquals(p.AsPath(), NewPath("category/package"))
}

func Test_PkgsrcPath_String(tt *testing.T) {
	t := NewTester(tt)
	defer t.Done()

	p := NewPkgsrcPath("any string..././")

	str := p.String()

	// No normalization takes place because it is typically not needed.
	t.CheckEquals(str, "any string..././")
}

func Test_PkgsrcPath_AsPath(tt *testing.T) {
	t := NewTester(tt)
	defer t.Done()

	pp := NewPkgsrcPath("./category/package/Makefile")

	p := pp.AsPath()

	t.CheckEquals(p.String(), "./category/package/Makefile")
}

func Test_PkgsrcPath_AsRelPath(tt *testing.T) {
	t := NewTester(tt)
	defer t.Done()

	pp := NewPkgsrcPath("./category/package/Makefile")

	rel := pp.AsRelPath()

	t.CheckEquals(rel.String(), "./category/package/Makefile")
}

func Test_PkgsrcPath_Dir(tt *testing.T) {
	t := NewTester(tt)
	defer t.Done()

	test := func(pp, cleaned PkgsrcPath) {
		t.CheckEquals(pp.Dir(), cleaned)
	}

	test("./dir/../dir/base///.", "./dir/../dir/base")
}

func Test_PkgsrcPath_Base(tt *testing.T) {
	t := NewTester(tt)
	defer t.Done()

	test := func(pp PkgsrcPath, base RelPath) {
		t.CheckEquals(pp.Base(), base)
	}

	test("./dir/../dir/base///.", ".")
}

func Test_PkgsrcPath_Count(tt *testing.T) {
	t := NewTester(tt)
	defer t.Done()

	test := func(pp PkgsrcPath, count int) {
		t.CheckEquals(pp.Count(), count)
	}

	test("./...////dir", 2)
}

func Test_PkgsrcPath_HasPrefixPath(tt *testing.T) {
	t := NewTester(tt)
	defer t.Done()

	test := func(pp PkgsrcPath, prefix Path, hasPrefixPath bool) {
		t.CheckEquals(pp.HasPrefixPath(prefix), hasPrefixPath)
	}

	test("./././///prefix/suffix", "prefix", true)
}

func Test_PkgsrcPath_JoinNoClean(tt *testing.T) {
	t := NewTester(tt)
	defer t.Done()

	test := func(pp PkgsrcPath, rel RelPath, joined PkgsrcPath) {
		t.CheckEquals(pp.JoinNoClean(rel), joined)
	}

	test("base///.", "./../rel", "base///././../rel")
}

func Test_NewPackagePath(tt *testing.T) {
	t := NewTester(tt)
	defer t.Done()

	p := NewPackagePath("../../category/package")

	t.CheckEquals(p.AsPath(), NewPath("../../category/package"))
}

func Test_NewPackagePathString(tt *testing.T) {
	t := NewTester(tt)
	defer t.Done()

	p := NewPackagePathString("../../category/package")

	t.CheckEquals(p.AsPath(), NewPath("../../category/package"))
}

func Test_PackagePath_AsPath(tt *testing.T) {
	t := NewTester(tt)
	defer t.Done()

	pp := NewPackagePath("../../category/package/Makefile")

	p := pp.AsPath()

	t.CheckEquals(p.String(), "../../category/package/Makefile")
}

func Test_PackagePath_AsRelPath(tt *testing.T) {
	t := NewTester(tt)
	defer t.Done()

	pp := NewPackagePath("./category/package/Makefile")

	rel := pp.AsRelPath()

	t.CheckEquals(rel.String(), "./category/package/Makefile")
}

func Test_PackagePath_String(tt *testing.T) {
	t := NewTester(tt)
	defer t.Done()

	pp := NewPackagePath("../../category/package/Makefile")

	str := pp.String()

	t.CheckEquals(str, "../../category/package/Makefile")
}

func Test_PackagePath_JoinNoClean(tt *testing.T) {
	t := NewTester(tt)
	defer t.Done()

	test := func(pp PackagePath, other RelPath, joined PackagePath) {
		t.CheckEquals(pp.JoinNoClean(other), joined)

	}

	test("../../category/package/patches", "patch-aa",
		"../../category/package/patches/patch-aa")
}

func Test_PackagePath_CleanPath(tt *testing.T) {
	t := NewTester(tt)
	defer t.Done()

	test := func(p PackagePath, cleaned PackagePath) {
		t.CheckEquals(p.CleanPath(), cleaned)
	}

	test("a/b/../../c/d/../.././e/../f", "a/b/../../e/../f")
}

func Test_PackagePath_IsEmpty(tt *testing.T) {
	t := NewTester(tt)
	defer t.Done()

	test := func(p PackagePath, isEmpty bool) {
		t.CheckEquals(p.IsEmpty(), isEmpty)
	}

	test("", true)
	test(".", false)
}

func Test_PackagePath_HasPrefixPath(tt *testing.T) {
	t := NewTester(tt)
	defer t.Done()

	test := func(p PackagePath, sub Path, hasPrefix bool) {
		t.CheckEquals(p.HasPrefixPath(sub), hasPrefix)
	}

	test("/root/subdir", "subdir", false)
	test("/root/subdir", "/root", true)
	test("/root/subdir", "/r", false)
}

func Test_PackagePath_ContainsPath(tt *testing.T) {
	t := NewTester(tt)
	defer t.Done()

	test := func(p PackagePath, sub Path, hasPrefix bool) {
		t.CheckEquals(p.ContainsPath(sub), hasPrefix)
	}

	test("/root/subdir", "subdir", true)
	test("/root/subdir", "/root", true)
	test("/root/subdir", "/r", false)
}

func Test_PackagePath_ContainsText(tt *testing.T) {
	t := NewTester(tt)
	defer t.Done()

	test := func(p PackagePath, sub string, hasPrefix bool) {
		t.CheckEquals(p.ContainsText(sub), hasPrefix)
	}

	test("/root/subdir", "subdir", true)
	test("/root/subdir", "/root", true)
	test("/root/subdir", "/r", true)
	test("/root/subdir", "t//sub", false)
}

func Test_PackagePath_Replace(tt *testing.T) {
	t := NewTester(tt)
	defer t.Done()

	test := func(p PackagePath, from, to string, result PackagePath) {
		t.CheckEquals(p.Replace(from, to), result)
	}

	test("dir/file", "dir", "other", "other/file")
	test("dir/file", "r", "sk", "disk/file")
	test("aaa/file", "a", "sub/", "sub/sub/sub//file")
}

func Test_NewRelPath(tt *testing.T) {
	t := NewTester(tt)
	defer t.Done()

	rel := NewRelPath("dir/file")

	t.CheckEquals(rel.String(), "dir/file")
}

func Test_NewRelPathString(tt *testing.T) {
	t := NewTester(tt)
	defer t.Done()

	rel := NewRelPathString("dir/file")

	t.CheckEquals(rel.String(), "dir/file")
}

func Test_RelPath_AsPath(tt *testing.T) {
	t := NewTester(tt)
	defer t.Done()

	rel := NewRelPath("relative")

	path := rel.AsPath()

	t.CheckEquals(path.String(), "relative")
}

func Test_RelPath_String(tt *testing.T) {
	t := NewTester(tt)
	defer t.Done()

	rel := NewRelPath(".///rel")

	str := rel.String()

	t.CheckEquals(str, ".///rel")
}

func Test_RelPath_IsEmpty(tt *testing.T) {
	t := NewTester(tt)
	defer t.Done()

	test := func(rel RelPath, isEmpty bool) {
		t.CheckEquals(rel.IsEmpty(), isEmpty)
	}

	test("", true)
	test(".", false)
	test("/", false)
}

func Test_RelPath_Split(tt *testing.T) {
	t := NewTester(tt)
	defer t.Done()

	test := func(rel RelPath, dir RelPath, base string) {
		actualDir, actualBase := rel.Split()
		t.CheckEquals(actualDir, dir)
		t.CheckEquals(actualBase, base)
	}

	test("dir/file", "dir/", "file")
	test("././///file", "././///", "file")

	t.ExpectAssert(
		func() { test("/", "/", "") })

}

func Test_RelPath_Dir(tt *testing.T) {
	t := NewTester(tt)
	defer t.Done()

	test := func(rel RelPath, dir RelPath) {
		t.CheckEquals(rel.Dir(), dir)
	}

	test("./dir/../dir///./file", "./dir/../dir")
}

func Test_RelPath_Base(tt *testing.T) {
	t := NewTester(tt)
	defer t.Done()

	test := func(rel RelPath, base RelPath) {
		t.CheckEquals(rel.Base(), base)
	}

	test("./dir/../dir///./file", "file")
}

func Test_RelPath_HasBase(tt *testing.T) {
	t := NewTester(tt)
	defer t.Done()

	test := func(rel RelPath, base string, hasBase bool) {
		t.CheckEquals(rel.HasBase(base), hasBase)
	}

	test("./dir/Makefile", "Makefile", true)
	test("./dir/Makefile", "Make", false)
	test("./dir/Makefile", "file", false)
	test("./dir/Makefile", "dir/Makefile", false)
}

func Test_RelPath_Parts(tt *testing.T) {
	t := NewTester(tt)
	defer t.Done()

	test := func(rel RelPath, parts ...string) {
		t.CheckDeepEquals(rel.Parts(), parts)
	}

	test("./dir/.///base", "dir", "base")
}

func Test_RelPath_Count(tt *testing.T) {
	t := NewTester(tt)
	defer t.Done()

	test := func(rel RelPath, count int) {
		t.CheckEquals(rel.Count(), count)
	}

	test("./dir/.///base", 2)
}

func Test_RelPath_Clean(tt *testing.T) {
	t := NewTester(tt)
	defer t.Done()

	test := func(rel RelPath, cleaned RelPath) {
		t.CheckDeepEquals(rel.Clean(), cleaned)
	}

	test("a/b/../../c/d/../.././e/../f", "f")
}

func Test_RelPath_CleanDot(tt *testing.T) {
	t := NewTester(tt)
	defer t.Done()

	test := func(rel RelPath, cleaned RelPath) {
		t.CheckEquals(rel.CleanDot(), cleaned)
	}

	test("a/b/../../c/d/../.././e/../f", "a/b/../../c/d/../../e/../f")
}

func Test_RelPath_CleanPath(tt *testing.T) {
	t := NewTester(tt)
	defer t.Done()

	test := func(rel RelPath, cleaned RelPath) {
		t.CheckEquals(rel.CleanPath(), cleaned)
	}

	test("a/b/../../c/d/../.././e/../f", "a/b/../../e/../f")
}

func Test_RelPath_JoinNoClean(tt *testing.T) {
	t := NewTester(tt)
	defer t.Done()

	test := func(rel, other, joined RelPath) {
		t.CheckEquals(rel.JoinNoClean(other), joined)
	}

	test("basedir/.//", "./other", "basedir/.///./other")
}

func Test_RelPath_Replace(tt *testing.T) {
	t := NewTester(tt)
	defer t.Done()

	test := func(rel RelPath, from, to string, result RelPath) {
		t.CheckEquals(rel.Replace(from, to), result)
	}

	test("dir/subdir/file", "/", ":", "dir:subdir:file")
}

func Test_RelPath_HasPrefixPath(tt *testing.T) {
	t := NewTester(tt)
	defer t.Done()

	test := func(rel RelPath, prefix Path, hasPrefixPath bool) {
		t.CheckEquals(rel.HasPrefixPath(prefix), hasPrefixPath)
	}

	test("dir/subdir/file", "dir", true)
	test("dir/subdir/file", "dir/sub", false)
	test("dir/subdir/file", "subdir", false)
}

func Test_RelPath_HasPrefixText(tt *testing.T) {
	t := NewTester(tt)
	defer t.Done()

	test := func(rel RelPath, prefix string, hasPrefixPath bool) {
		t.CheckEquals(rel.HasPrefixText(prefix), hasPrefixPath)
	}

	test("dir/subdir/file", "dir", true)
	test("dir/subdir/file", "dir/sub", true)
	test("dir/subdir/file", "subdir", false)
	test("dir/subdir/file", "super", false)
}

func Test_RelPath_ContainsPath(tt *testing.T) {
	t := NewTester(tt)
	defer t.Done()

	test := func(rel RelPath, prefix Path, hasPrefixPath bool) {
		t.CheckEquals(rel.ContainsPath(prefix), hasPrefixPath)
	}

	test("dir/subdir/file", "dir", true)
	test("dir/subdir/file", "dir/sub", false)
	test("dir/subdir/file", "subdir", true)
}

func Test_RelPath_ContainsText(tt *testing.T) {
	t := NewTester(tt)
	defer t.Done()

	test := func(rel RelPath, prefix string, hasPrefixPath bool) {
		t.CheckEquals(rel.ContainsText(prefix), hasPrefixPath)
	}

	test("dir/subdir/file", "dir", true)
	test("dir/subdir/file", "dir/sub", true)
	test("dir/subdir/file", "subdir", true)
	test("dir/subdir/file", "super", false)
}

func Test_RelPath_HasSuffixPath(tt *testing.T) {
	t := NewTester(tt)
	defer t.Done()

	test := func(rel RelPath, prefix Path, hasPrefixPath bool) {
		t.CheckEquals(rel.HasSuffixPath(prefix), hasPrefixPath)
	}

	test("dir/subdir/file", "dir", false)
	test("dir/subdir/file", "file", true)
	test("dir/subdir/file", "le", false)
	test("dir/subdir/file", "subdir/file", true)
	test("dir/subdir/file", "subdir", false)
}

func Test_RelPath_HasSuffixText(tt *testing.T) {
	t := NewTester(tt)
	defer t.Done()

	test := func(rel RelPath, prefix string, hasPrefixPath bool) {
		t.CheckEquals(rel.HasSuffixText(prefix), hasPrefixPath)
	}

	test("dir/subdir/file", "dir", false)
	test("dir/subdir/file", "file", true)
	test("dir/subdir/file", "le", true)
	test("dir/subdir/file", "subdir/file", true)
	test("dir/subdir/file", "subdir", false)
}

func Test_RelPath_Rel(tt *testing.T) {
	t := NewTester(tt)
	defer t.Done()

	test := func(base RelPath, other Path, result RelPath) {
		t.CheckEquals(base.Rel(other), result)
	}

	test("a/b/c", "d/e/f/file", "../../../d/e/f/file")
	test(".", ".", ".")

	// The trailing dot marks the difference between a file and a directory.
	// This is the same behavior as with filepath.Rel.
	test("a/b/c", ".", "../../../.")
}
