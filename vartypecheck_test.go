package main

import (
	check "gopkg.in/check.v1"
)

func (s *Suite) TestVartypeCheckFetchURL(c *check.C) {
	G.globalData.masterSiteUrls = map[string]string{
		"https://github.com/":         "MASTER_SITE_GITHUB",
		"http://ftp.gnu.org/pub/gnu/": "MASTER_SITE_GNU",
	}
	G.globalData.masterSiteVars = map[string]bool{
		"MASTER_SITE_GITHUB": true,
		"MASTER_SITE_GNU":    true,
	}
	line := NewLine("fname", "1", "dummy", nil)
	vc := &VartypeCheck{
		line,
		"MASTER_SITES",
		"=",
		"https://github.com/example/project/",
		"https://github.com/example/project/",
		"",
		true,
		NOT_GUESSED}

	vc.FetchURL()

	c.Check(s.Output(), equals, ""+
		"WARN: fname:1: Please use ${MASTER_SITE_GITHUB:=example/} instead of \"https://github.com/example/project/\".\n"+
		"WARN: fname:1: Run \"@BMAKE@ help topic=github\" for further tips.\n")

	vc.value = "http://ftp.gnu.org/pub/gnu/bison" // Missing a slash at the end
	vc.valueNovar = vc.value

	vc.FetchURL()

	c.Check(s.Output(), equals, "WARN: fname:1: Please use ${MASTER_SITE_GNU:=bison} instead of \"http://ftp.gnu.org/pub/gnu/bison\".\n")

	vc.value = "${MASTER_SITE_GNU:=bison}"
	vc.valueNovar = ""

	vc.FetchURL()

	c.Check(s.Output(), equals, "ERROR: fname:1: The subdirectory in MASTER_SITE_GNU must end with a slash.\n")

	vc.value = "${MASTER_SITE_INVALID:=subdir/}"
	vc.valueNovar = ""

	vc.FetchURL()

	c.Check(s.Output(), equals, "ERROR: fname:1: MASTER_SITE_INVALID does not exist.\n")
}
