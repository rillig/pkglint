package pkglint

import "gopkg.in/check.v1"

func (s *Suite) Test_NewUrlChecker(c *check.C) {
	t := s.Init(c)

	mklines := t.NewMkLines("filename.mk",
		MkCvsID,
		"",
		"MASTER_SITES=\thttps://example.org/")
	mkline := mklines.mklines[2]

	ck := NewUrlChecker(&VartypeCheck{
		mklines,
		mkline,
		"MASTER_SITES",
		opAssign,
		"https://example.org/",
		"",
		"",
		false,
	})

	t.CheckEquals(ck.varname, "MASTER_SITES")
}

func (s *Suite) Test_UrlChecker_CheckFetchURL(c *check.C) {
	t := s.Init(c)
	t.SetUpVarType("HOMEPAGE", BtURL, NoVartypeOptions)

	mklines := t.NewMkLines("filename.mk",
		MkCvsID,
		"",
		"HOMEPAGE=\thttps://example.org/package#download")

	mklines.Check()

	t.CheckOutputLines(
		"WARN: filename.mk:3: The # character starts a makefile comment.",
	)
}

func (s *Suite) Test_UrlChecker_CheckURL(c *check.C) {
	t := s.Init(c)
	t.SetUpVarType("MASTER_SITES", BtFetchURL, List)

	mklines := t.NewMkLines("filename.mk",
		MkCvsID,
		"",
		"MASTER_SITES=\thttps://example.org/downloads")

	mklines.Check()

	t.CheckOutputLines(
		"WARN: filename.mk:3: The fetch URL \"https://example.org/downloads\" should end with a slash.")
}
