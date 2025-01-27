package pkglint

import "gopkg.in/check.v1"

func (s *Suite) Test_NewVulnerabilities(c *check.C) {
	t := s.Init(c)

	v := NewVulnerabilities()

	t.CheckNotNil(v.byPkgbase)
}

func (s *Suite) Test_Vulnerabilities_read(c *check.C) {
	t := s.Init(c)

	f := t.CreateFileLines("pkg-vulnerabilities",
		"#FORMAT 1.0.0",
		"pkgbase<5.6.7\tbuffer-overflow\thttps://example.org/SA-2025-00001",
		"pkgbase-5<5.6.7\tbuffer-overflow\thttps://example.org/SA-2025-00001")
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
		"ERROR: ~/pkg-vulnerabilities:3: " +
			"Package pattern \"pkgbase-5\" is followed by " +
			"extra text \"<5.6.7\".")
}
