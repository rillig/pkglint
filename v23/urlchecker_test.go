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
	t.SetUpMasterSite("MASTER_SITE_CPAN", "https://example.org/")

	mklines := t.NewMkLines("filename.mk",
		MkCvsID,
		"",
		"MASTER_SITES=\thttps://example.org/downloads",
		"MASTER_SITES+=\thttps://example.org/${:U}",
		"MASTER_SITES+=\t${MASTER_SITE_CPAN:=subdir/}",
		"MASTER_SITES+=\t${MASTER_SITE_CPAN:=subdir}",
	)

	mklines.Check()

	t.CheckOutputLines(
		"WARN: filename.mk:3: Use ${MASTER_SITE_CPAN:=downloads} instead of \"https://example.org/downloads\".",
		"WARN: filename.mk:4: Use ${MASTER_SITE_CPAN:=${:U}} instead of \"https://example.org/${:U}\".",
		"WARN: filename.mk:5: MASTER_SITE_CPAN should not be used indirectly at load time (via MASTER_SITES).",
		"ERROR: filename.mk:6: The fetch URL \"${MASTER_SITE_CPAN:=subdir}\" must end with a slash.",
	)
}

func (s *Suite) Test_UrlChecker_endsWithSlash(c *check.C) {
	t := s.Init(c)

	test := func(url string, expected YesNoUnknown) {
		mklines := t.NewMkLines("filename.mk",
			"MASTER_SITES=\t"+url)
		mkline := mklines.mklines[0]
		ck := UrlChecker{mkline.Varname(), mkline.Op(), mkline, mklines}
		tokens := mkline.Tokenize(url, false)
		actual := ck.endsWithSlash(tokens)
		t.CheckEquals(actual, expected)
	}

	test("https://example.org", no)
	test("https://example.org/", yes)
	test("https://example.org/${PKGNAME}", no)
	test("https://example.org/${PKGNAME}/", yes)
	test("${ANY:=anything}", no)
	test("${ANY:=anything/}", yes)
	test("${MASTER_SITE_CPAN:S,^,-,}", unknown)

	t.CheckOutputEmpty()
}
