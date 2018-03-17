package main

import "gopkg.in/check.v1"

// Ensures that pkglint can handle MASTER_SITES definitions with and
// without line continuations.
//
// See https://mail-index.netbsd.org/tech-pkg/2017/01/18/msg017698.html.
func (s *Suite) Test_Pkgsrc_loadMasterSites(c *check.C) {
	t := s.Init(c)

	t.CreateFileLines("mk/fetch/sites.mk",
		MkRcsID,
		"",
		"MASTER_SITE_A+= https://example.org/distfiles/",
		"MASTER_SITE_B+= https://b.example.org/distfiles/ \\",
		"  https://b2.example.org/distfiles/",
		"MASTER_SITE_A+= https://a.example.org/distfiles/")

	G.Pkgsrc.loadMasterSites()

	c.Check(G.Pkgsrc.MasterSiteURLToVar["https://example.org/distfiles/"], equals, "MASTER_SITE_A")
	c.Check(G.Pkgsrc.MasterSiteURLToVar["https://b.example.org/distfiles/"], equals, "MASTER_SITE_B")
	c.Check(G.Pkgsrc.MasterSiteURLToVar["https://b2.example.org/distfiles/"], equals, "MASTER_SITE_B")
	c.Check(G.Pkgsrc.MasterSiteURLToVar["https://a.example.org/distfiles/"], equals, "MASTER_SITE_A")
	c.Check(G.Pkgsrc.MasterSiteVarToURL["MASTER_SITE_A"], equals, "https://example.org/distfiles/")
	c.Check(G.Pkgsrc.MasterSiteVarToURL["MASTER_SITE_B"], equals, "https://b.example.org/distfiles/")
}
