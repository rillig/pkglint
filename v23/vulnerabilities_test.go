package pkglint

import "gopkg.in/check.v1"

func (s *Suite) Test_NewVulnerabilities(c *check.C) {
	t := s.Init(c)

	v := NewVulnerabilities()

	t.CheckNotNil(v.byPkgbase)
}

func (s *Suite) Test_Vulnerabilities_read(c *check.C) {
	t := s.Init(c)
	t.Chdir(".")

	f := t.CreateFileLines("pkg-vulnerabilities",
		"#FORMAT 1.0.0",
		"pkgbase<5.6.7\tbuffer-overflow\thttps://example.org/SA-2025-00001",
		"pkgbase-5<5.6.7\tbuffer-overflow\thttps://example.org/SA-2025-00001",
		"# comment",
		"invalid",
		"}{\tunbalanced-braces\thttps://example.org/",
		"{invalid-package-pattern}\tinvalid\thttps://example.org/",
		"invalid-package-pattern\tinvalid\thttps://example.org/",
		"{package-1.0-2}\tinvalid\thttps://example.org/",
		"package-1.0-2\tinvalid\thttps://example.org/",
		"{package-1.0:extra}\tinvalid\thttps://example.org/",
		"package-1.0:extra\tinvalid\thttps://example.org/")
	v := NewVulnerabilities()
	v.read(f)

	t.CheckEquals(len(v.byPkgbase), 1)
	vs := v.byPkgbase["pkgbase"]
	if t.CheckNotNil(vs) {
		t.CheckEquals(len(vs), 1)
		t.CheckEquals(*vs[0].pattern, PackagePattern{"pkgbase", "", "", "<", "5.6.7", ""})
		t.CheckEquals(vs[0].kind, "buffer-overflow")
		t.CheckEquals(vs[0].url, "https://example.org/SA-2025-00001")
	}

	t.CheckOutputLines(
		"ERROR: pkg-vulnerabilities:3: Package pattern \"pkgbase-5\" is followed by extra text \"<5.6.7\".",
		"ERROR: pkg-vulnerabilities:5: Invalid line format \"invalid\".",
		"ERROR: pkg-vulnerabilities:6: Package pattern \"}{\" must have balanced braces.",
		"ERROR: pkg-vulnerabilities:7: Package pattern \"{invalid-package-pattern}\" expands to the invalid package pattern \"invalid-package-pattern\".",
		"ERROR: pkg-vulnerabilities:8: Invalid package pattern \"invalid-package-pattern\".",
		"ERROR: pkg-vulnerabilities:9: Package pattern \"{package-1.0-2}\" expands to \"package-1.0-2\", which has a \"-\" in the version number.",
		"ERROR: pkg-vulnerabilities:10: Package pattern \"package-1.0-2\" has a \"-\" in the version number.",
		"ERROR: pkg-vulnerabilities:11: Package pattern \"{package-1.0:extra}\" expands to \"package-1.0\", which is followed by extra text \":extra\".",
		"ERROR: pkg-vulnerabilities:12: Package pattern \"package-1.0\" is followed by extra text \":extra\".",
	)
}

func (s *Suite) Test_Vulnerabilities_read__empty(c *check.C) {
	t := s.Init(c)
	t.Chdir(".")

	f := t.CreateFileLines("pkg-vulnerabilities",
		"# comment")
	v := NewVulnerabilities()
	v.read(f)

	t.CheckOutputLines(
		"ERROR: pkg-vulnerabilities: Invalid file format \"\".")
}
