package main

import (
	check "gopkg.in/check.v1"
	"io/ioutil"
	"os"
	"path/filepath"
)

func (s *Suite) TestChecklinesDistinfo(c *check.C) {
	tmpdir := c.MkDir()
	patchesdir := tmpdir + "/patches"
	patchAa := patchesdir + "/patch-aa"
	patchContents := "" +
		"$" + "NetBSD$ line is ignored\n" +
		"patch contents\n"

	os.Mkdir(patchesdir, 0777)
	if err := ioutil.WriteFile(patchAa, []byte(patchContents), 0666); err != nil {
		c.Fatal(err)
	}
	G.currentDir = filepath.ToSlash(tmpdir)

	lines := s.NewLines("distinfo",
		"should be the RCS ID",
		"should be empty",
		"MD5 (distfile.tar.gz) = 12345678901234567890123456789012",
		"SHA1 (distfile.tar.gz) = 1234567890123456789012345678901234567890",
		"SHA1 (patch-aa) = 6b98dd609f85a9eb9c4c1e4e7055a6aaa62b7cc7")

	checklinesDistinfo(lines)

	c.Check(s.Output(), equals, ""+
		"ERROR: distinfo:1: Expected \"$"+"NetBSD$\".\n"+
		"NOTE: distinfo:2: Empty line expected.\n"+
		"ERROR: distinfo:5: Expected SHA1, RMD160, SHA512, Size checksums for \"distfile.tar.gz\", got MD5, SHA1.\n")
}

func (s *Suite) TestChecklinesDistinfo_GlobalHashMismatch(c *check.C) {
	otherLine := NewLine("other/Makefile", "1", "dummy", nil)
	G.ipcDistinfo = make(map[string]*Hash)
	G.ipcDistinfo["SHA512:pkgname-1.0.tar.gz"] = &Hash{"asdfasdf", otherLine}
	lines := s.NewLines("distinfo",
		"$"+"NetBSD$",
		"",
		"SHA512 (pkgname-1.0.tar.gz) = 12341234")

	checklinesDistinfo(lines)

	c.Check(s.Output(), equals, ""+
		"ERROR: distinfo:3: The hash SHA512 for pkgname-1.0.tar.gz is 12341234, ...\n"+
		"ERROR: other/Makefile:1: ... which differs from asdfasdf.\n"+
		"ERROR: distinfo:EOF: Expected SHA1, RMD160, SHA512, Size checksums for \"pkgname-1.0.tar.gz\", got SHA512.\n")
}
