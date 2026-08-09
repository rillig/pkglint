package pkglint

import "testing"

func Test_NewNetBSDProject(tt *testing.T) {
	t := NewTester(tt)
	defer t.Done()

	project := NewNetBSDProject()

	t.CheckEquals(project.Types().IsDefinedCanon("PKGNAME"), false)
}

func Test_NetBSDProject_Deprecated(tt *testing.T) {
	t := NewTester(tt)
	defer t.Done()

	G.Pkgsrc = nil
	G.Project = NewNetBSDProject()
	mklines := t.NewMkLines("filename.mk",
		MkCvsID,
		"DEPRECATED=\t${DEPRECATED}")

	mklines.Check()

	t.CheckOutputEmpty()
}

func Test_NetBSDProject_Types(tt *testing.T) {
	t := NewTester(tt)
	defer t.Done()

	project := NewNetBSDProject()
	project.Types().acl("VAR", BtUnknown, NoVartypeOptions, "*.mk: append")

	t.CheckEquals(project.Types().Canon("VAR").basicType, BtUnknown)
	t.CheckEquals(project.Types().Canon("UNDEFINED"), (*Vartype)(nil))
}
