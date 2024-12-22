package pkglint

import "gopkg.in/check.v1"

func (s *Suite) Test_NewNetBSDProject(c *check.C) {
	t := s.Init(c)

	project := NewNetBSDProject()

	t.CheckEquals(project.Types().IsDefinedCanon("PKGNAME"), false)
}

func (s *Suite) Test_NetBSDProject_Deprecated(c *check.C) {
	t := s.Init(c)

	G.Pkgsrc = nil
	G.Project = NewNetBSDProject()
	mklines := t.NewMkLines("filename.mk",
		MkCvsID,
		"DEPRECATED=\t${DEPRECATED}")

	mklines.Check()

	t.CheckOutputEmpty()
}

func (s *Suite) Test_NetBSDProject_Types(c *check.C) {
	t := s.Init(c)

	project := NewNetBSDProject()
	project.Types().acl("VAR", BtUnknown, NoVartypeOptions, "*.mk: append")

	t.CheckEquals(project.Types().Canon("VAR").basicType, BtUnknown)
	t.CheckEquals(project.Types().Canon("UNDEFINED"), (*Vartype)(nil))
}
