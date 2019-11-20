package pkglint

import (
	"io/ioutil"
	"os"
	"path"
	"path/filepath"
	"strings"
)

// Path is a slash-separated path in the filesystem.
// It may be absolute or relative.
// Some paths may contain placeholders like @VAR@ or ${VAR}.
// The base directory of relative paths depends on the context
// in which the path is used.
type Path string

func NewPath(name string) Path { return Path(name) }

func NewPathSlash(name string) Path { return Path(filepath.ToSlash(name)) }

func (p Path) String() string { return string(p) }

func (p Path) Dir() Path { return Path(path.Dir(string(p))) }

func (p Path) Base() string { return path.Base(string(p)) }

func (p Path) Split() (dir Path, base string) {
	strDir, strBase := path.Split(string(p))
	return Path(strDir), strBase
}

func (p Path) Parts() []string {
	return strings.FieldsFunc(string(p), func(r rune) bool { return r == '/' })
}

func (p Path) Count() int { return len(p.Parts()) }

func (p Path) HasPrefixText(prefix string) bool {
	return hasPrefix(string(p), prefix)
}

func (p Path) HasPrefixPath(prefix Path) bool {
	return hasPrefix(string(p), string(prefix)) &&
		(len(p) == len(prefix) || p[len(prefix)] == '/')
}

func (p Path) ContainsText(contained string) bool {
	return contains(string(p), contained)
}

func (p Path) ContainsPath(contained Path) bool {
	limit := len(p) - len(contained)
	for i := 0; i <= limit; i++ {
		if (i == 0 || p[i-1] == '/') &&
			(i == limit || p[i+len(contained)] == '/') &&
			hasPrefix(string(p)[i:], string(contained)) {
			return true
		}
	}
	return false
}

func (p Path) HasSuffixText(suffix string) bool {
	return hasSuffix(string(p), suffix)
}

func (p Path) HasSuffixPath(suffix Path) bool {
	return hasSuffix(string(p), string(suffix)) &&
		(len(p) == len(suffix) || p[len(p)-len(suffix)-1] == '/')
}

func (p Path) TrimSuffix(suffix string) Path {
	return Path(strings.TrimSuffix(string(p), suffix))
}

func (p Path) HasBase(base string) bool { return p.Base() == base }

func (p Path) JoinClean(s string) Path {
	return Path(path.Join(string(p), s))
}

func (p Path) JoinNoClean(s string) Path { return Path(string(p) + "/" + s) }

func (p Path) Clean() Path { return NewPath(path.Clean(string(p))) }

func (p Path) IsAbs() bool {
	return filepath.IsAbs(filepath.FromSlash(string(p)))
}

func (p Path) Rel(other Path) Path {
	fp := filepath.FromSlash(p.String())
	fpOther := filepath.FromSlash(other.String())
	rel, err := filepath.Rel(fp, fpOther)
	assertNil(err, "relpath from %q to %q", p, other)
	return NewPath(filepath.ToSlash(rel))
}

func (p Path) Rename(newName Path) error {
	return os.Rename(string(p), string(newName))
}

func (p Path) Lstat() (os.FileInfo, error) { return os.Lstat(string(p)) }

func (p Path) Stat() (os.FileInfo, error) { return os.Stat(string(p)) }

func (p Path) Chmod(mode os.FileMode) error {
	return os.Chmod(string(p), mode)
}

func (p Path) IsFile() bool {
	info, err := p.Lstat()
	return err == nil && info.Mode().IsRegular()
}

func (p Path) IsDir() bool {
	info, err := p.Lstat()
	return err == nil && info.IsDir()
}

func (p Path) ReadDir() ([]os.FileInfo, error) {
	return ioutil.ReadDir(string(p))
}

func (p Path) Open() (*os.File, error) { return os.Open(string(p)) }

func (p Path) ReadString() (string, error) {
	bytes, err := ioutil.ReadFile(string(p))
	return string(bytes), err
}

func (p Path) WriteString(s string) error {
	return ioutil.WriteFile(string(p), []byte(s), 0666)
}
