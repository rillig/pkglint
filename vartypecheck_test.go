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

	newVartypeCheck("MASTER_SITES", "=", "https://github.com/example/project/").FetchURL()

	c.Check(s.Output(), equals, ""+
		"WARN: fname:1: Please use ${MASTER_SITE_GITHUB:=example/} instead of \"https://github.com/example/project/\".\n"+
		"WARN: fname:1: Run \""+confMake+" help topic=github\" for further tips.\n")

	newVartypeCheck("MASTER_SITES", "=", "http://ftp.gnu.org/pub/gnu/bison").FetchURL() // Missing a slash at the end

	c.Check(s.Output(), equals, "WARN: fname:1: Please use ${MASTER_SITE_GNU:=bison} instead of \"http://ftp.gnu.org/pub/gnu/bison\".\n")

	newVartypeCheck("MASTER_SITES", "=", "${MASTER_SITE_GNU:=bison}").FetchURL()

	c.Check(s.Output(), equals, "ERROR: fname:1: The subdirectory in MASTER_SITE_GNU must end with a slash.\n")

	newVartypeCheck("MASTER_SITES", "=", "${MASTER_SITE_INVALID:=subdir/}").FetchURL()

	c.Check(s.Output(), equals, "ERROR: fname:1: MASTER_SITE_INVALID does not exist.\n")
}

func (s *Suite) TestVartypeCheckYes(c *check.C) {

	newVartypeCheck("APACHE_MODULE", "=", "yes").Yes()

	c.Check(s.Output(), equals, "")

	newVartypeCheck("APACHE_MODULE", "=", "no").Yes()

	c.Check(s.Output(), equals, "WARN: fname:1: APACHE_MODULE should be set to YES or yes.\n")

	newVartypeCheck("APACHE_MODULE", "=", "${YESVAR}").Yes()

	c.Check(s.Output(), equals, "WARN: fname:1: APACHE_MODULE should be set to YES or yes.\n")
}

func newVartypeCheck(varname, op, value string) *VartypeCheck {
	line := NewLine("fname", "1", "dummy", nil)
	valueNovar := withoutMakeVariables(line, value, true)
	return &VartypeCheck{line, varname, op, value, valueNovar, "", true, NOT_GUESSED}
}
