package pkglint

import "gopkg.in/check.v1"

func (s *Suite) Test_VarorderChecker_Check__only_required_variables(c *check.C) {
	t := s.Init(c)

	mklines := t.NewMkLines("Makefile",
		MkCvsID,
		"",
		"DISTNAME=9term",
		"CATEGORIES=x11",
		"",
		".include \"../../mk/bsd.pkg.mk\"")

	(&VarorderChecker{mklines}).Check()

	t.CheckOutputLines(
		"WARN: Makefile:3: The canonical order of the variables is " +
			"DISTNAME, CATEGORIES, empty line, COMMENT, LICENSE.")
}

func (s *Suite) Test_VarorderChecker_Check__with_optional_variables(c *check.C) {
	t := s.Init(c)

	mklines := t.NewMkLines("Makefile",
		MkCvsID,
		"",
		"GITHUB_PROJECT=project",
		"DISTNAME=9term",
		"CATEGORIES=x11")

	(&VarorderChecker{mklines}).Check()

	// TODO: Make this warning more specific to the actual situation.

	// Before 2022-03-11, the GitHub variables were allowed above DISTNAME,
	// which allowed more variation than necessary and made the warning longer.
	t.CheckOutputLines(
		"WARN: Makefile:3: The canonical order of the variables is " +
			"DISTNAME, CATEGORIES, GITHUB_PROJECT, empty line, " +
			"COMMENT, LICENSE.")
}

// Ensure that comments and empty lines do not lead to panics.
// This had been the case when the code accessed fields like Varname from the
// MkLine without checking the line type before.
func (s *Suite) Test_VarorderChecker_relevantLines__comments_do_not_crash(c *check.C) {
	t := s.Init(c)

	mklines := t.NewMkLines("Makefile",
		MkCvsID,
		"",
		"GITHUB_PROJECT=project",
		"",
		"# comment",
		"",
		"DISTNAME=9term",
		"# comment",
		"CATEGORIES=x11")

	(&VarorderChecker{mklines}).Check()

	t.CheckOutputLines(
		"WARN: Makefile:3: The canonical order of the variables is " +
			"DISTNAME, CATEGORIES, GITHUB_PROJECT, empty line, " +
			"COMMENT, LICENSE.")
}

func (s *Suite) Test_VarorderChecker_relevantLines__comments_are_ignored(c *check.C) {
	t := s.Init(c)

	mklines := t.NewMkLines("Makefile",
		MkCvsID,
		"",
		"DISTNAME=\tdistname-1.0",
		"CATEGORIES=\tsysutils",
		"",
		"MAINTAINER=\tpkgsrc-users@NetBSD.org",
		"# comment",
		"COMMENT=\tComment",
		"LICENSE=\tgnu-gpl-v2")

	(&VarorderChecker{mklines}).Check()

	t.CheckOutputEmpty()
}

func (s *Suite) Test_VarorderChecker_relevantLines__commented_variable_assignment(c *check.C) {
	t := s.Init(c)

	mklines := t.NewMkLines("Makefile",
		MkCvsID,
		"",
		"DISTNAME=\tdistname-1.0",
		"CATEGORIES=\tsysutils",
		"",
		"MAINTAINER=\tpkgsrc-users@NetBSD.org",
		"#HOMEPAGE=\thttps://example.org/",
		"COMMENT=\tComment",
		"LICENSE=\tgnu-gpl-v2")

	(&VarorderChecker{mklines}).Check()

	t.CheckOutputEmpty()
}

func (s *Suite) Test_VarorderChecker_relevantLines__GITHUB_PROJECT_at_the_top(c *check.C) {
	t := s.Init(c)

	mklines := t.NewMkLines("Makefile",
		MkCvsID,
		"",
		"GITHUB_PROJECT=\t\tautocutsel",
		"DISTNAME=\t\tautocutsel-0.10.0",
		"CATEGORIES=\t\tx11",
		"MASTER_SITES=\t\t${MASTER_SITE_GITHUB:=sigmike/}",
		"GITHUB_TAG=\t\t${PKGVERSION_NOREV}",
		"",
		"COMMENT=\tComment",
		"LICENSE=\tgnu-gpl-v2")

	(&VarorderChecker{mklines}).Check()

	// Before 2022-03-11, the GitHub variables were allowed above DISTNAME,
	// which allowed more variation than necessary and made the warning longer.
	t.CheckOutputLines(
		"WARN: Makefile:3: The canonical order of the variables is " +
			"DISTNAME, CATEGORIES, MASTER_SITES, GITHUB_PROJECT, " +
			"GITHUB_TAG, empty line, COMMENT, LICENSE.")
}

func (s *Suite) Test_VarorderChecker_relevantLines__GITHUB_PROJECT_at_the_bottom(c *check.C) {
	t := s.Init(c)

	mklines := t.NewMkLines("Makefile",
		MkCvsID,
		"",
		"DISTNAME=\t\tautocutsel-0.10.0",
		"CATEGORIES=\t\tx11",
		"MASTER_SITES=\t\t${MASTER_SITE_GITHUB:=sigmike/}",
		"GITHUB_PROJECT=\t\tautocutsel",
		"GITHUB_TAG=\t\t${PKGVERSION_NOREV}",
		"",
		"COMMENT=\tComment",
		"LICENSE=\tgnu-gpl-v2")

	(&VarorderChecker{mklines}).Check()

	t.CheckOutputEmpty()
}

// TODO: Add more tests like skip_if_there_are_directives for other line types.

// https://mail-index.netbsd.org/tech-pkg/2017/01/18/msg017698.html
func (s *Suite) Test_VarorderChecker_relevantLines__MASTER_SITES(c *check.C) {
	t := s.Init(c)

	mklines := t.NewMkLines("Makefile",
		MkCvsID,
		"",
		"PKGNAME=\tpackage-1.0",
		"CATEGORIES=\tcategory",
		"MASTER_SITES=\thttp://example.org/",
		"MASTER_SITES+=\thttp://mirror.example.org/",
		"",
		"COMMENT=\tComment",
		"LICENSE=\tgnu-gpl-v2")

	(&VarorderChecker{mklines}).Check()

	// No warning that "MASTER_SITES appears too late"
	t.CheckOutputEmpty()
}

func (s *Suite) Test_VarorderChecker_relevantLines__comment_at_end_of_section(c *check.C) {
	t := s.Init(c)

	t.SetUpVartypes()
	mklines := t.NewMkLines("Makefile",
		MkCvsID,
		"",
		"CATEGORIES=     net",
		"SITES.*=        # none",
		"# comment after the last variable of a section",
		"",
		"MAINTAINER=     maintainer@example.org",
		"HOMEPAGE=       https://github.com/project/pkgbase/",
		"COMMENT=        Comment",
		"LICENSE=        gnu-gpl-v3",
		"",
		".include \"../../mk/bsd.pkg.mk\"")

	t.EnableTracingToLog()
	(&VarorderChecker{mklines}).Check()

	// The varorder code is not skipped, not even because of the comment
	// after SITES.*.
	t.CheckOutputLinesMatching(`.*varorder.*`,
		nil...)
}

func (s *Suite) Test_VarorderChecker_relevantLines__comments_between_sections(c *check.C) {
	t := s.Init(c)

	t.SetUpVartypes()
	mklines := t.NewMkLines("Makefile",
		MkCvsID,
		"",
		"CATEGORIES=     net",
		"",
		"# comment 1",
		"",
		"# comment 2",
		"",
		"MAINTAINER=     maintainer@example.org",
		"HOMEPAGE=       https://github.com/project/pkgbase/",
		"COMMENT=        Comment",
		"LICENSE=        gnu-gpl-v3",
		"",
		".include \"../../mk/bsd.pkg.mk\"")

	(&VarorderChecker{mklines}).Check()

	// The empty line between the comments is not treated as a section separator.
	t.CheckOutputEmpty()
}

func (s *Suite) Test_VarorderChecker_relevantLines__commented_varassign(c *check.C) {
	t := s.Init(c)

	t.SetUpVartypes()
	mklines := t.NewMkLines("Makefile",
		MkCvsID,
		"",
		"CATEGORIES=     net",
		"#MASTER_SITES=  # none",
		"",
		"HOMEPAGE=       https://github.com/project/pkgbase/",
		"#HOMEPAGE=      https://github.com/project/pkgbase/",
		"#HOMEPAGE=      https://github.com/project/pkgbase/",
		"#HOMEPAGE=      https://github.com/project/pkgbase/",
		"#HOMEPAGE=      https://github.com/project/pkgbase/",
		"LICENSE=        gnu-gpl-v3",
		"COMMENT=        Comment",
		"",
		".include \"../../mk/bsd.pkg.mk\"")

	(&VarorderChecker{mklines}).Check()

	// The order of the variables LICENSE and COMMENT is intentionally
	// wrong to force the warning.
	//
	// Up to June 2019 (308099138a62) pkglint mentioned in the warning
	// each commented variable assignment, even repeatedly for the same
	// variable name.
	//
	// These variable assignments should be in the correct order, even
	// if they are commented out. It's not necessary though to list a
	// variable more than once.
	t.CheckOutputLines(
		"WARN: Makefile:3: The canonical order of the variables is " +
			"CATEGORIES, MASTER_SITES, empty line, HOMEPAGE, COMMENT, LICENSE.")
}

func (s *Suite) Test_VarorderChecker_relevantLines__DEPENDS(c *check.C) {
	t := s.Init(c)

	t.SetUpVartypes()
	mklines := t.NewMkLines("Makefile",
		MkCvsID,
		"",
		"CATEGORIES=     net",
		"",
		"COMMENT=        Comment",
		"LICENSE=        license",
		"MAINTAINER=     maintainer@example.org", // In wrong order
		"",
		"DEPENDS+=       dependency>=1.0:../../category/dependency",
		"",
		".include \"../../mk/bsd.pkg.mk\"")

	(&VarorderChecker{mklines}).Check()

	t.CheckOutputLines(
		"WARN: Makefile:3: The canonical order of the variables is " +
			"CATEGORIES, empty line, MAINTAINER, COMMENT, LICENSE, empty line, DEPENDS.")
}

func (s *Suite) Test_VarorderChecker_skip__skip_because_of_foreign_variable(c *check.C) {
	t := s.Init(c)

	mklines := t.NewMkLines("Makefile",
		MkCvsID,
		"",
		"DISTNAME=\tdistname-1.0",
		"USE_TOOLS+=gmake",
		"CATEGORIES=\tsysutils",
		"",
		"MAINTAINER=\tpkgsrc-users@NetBSD.org",
		"#HOMEPAGE=\thttps://example.org/",
		"COMMENT=\tComment",
		"LICENSE=\tgnu-gpl-v2")

	t.EnableTracingToLog()
	(&VarorderChecker{mklines}).Check()

	t.CheckOutputLinesMatching(`.*varorder.*`,
		"TRACE:   Skipping varorder because of line 4.")
}

func (s *Suite) Test_VarorderChecker_skip__skip_if_there_are_directives(c *check.C) {
	t := s.Init(c)

	mklines := t.NewMkLines("Makefile",
		MkCvsID,
		"",
		"DISTNAME=\tdistname-1.0",
		"CATEGORIES=\tsysutils",
		"",
		".if ${DISTNAME:Mdistname-*}",
		"MAINTAINER=\tpkgsrc-users@NetBSD.org",
		".endif",
		"LICENSE=\tgnu-gpl-v2")

	(&VarorderChecker{mklines}).Check()

	// No warning about the missing COMMENT since the .if directive
	// causes the whole check to be skipped.
	t.CheckOutputEmpty()

	// Just for code coverage.
	t.DisableTracing()
	(&VarorderChecker{mklines}).Check()
	t.CheckOutputEmpty()
}

func (s *Suite) Test_VarorderChecker_skip__LICENSE(c *check.C) {
	t := s.Init(c)

	t.CreateFileLines("mk/bsd.pkg.mk", "# dummy")
	t.CreateFileLines("x11/Makefile", MkCvsID)
	t.CreateFileLines("x11/9term/DESCR", "Terminal")
	t.CreateFileLines("x11/9term/PLIST", PlistCvsID, "bin/9term")
	t.CreateFileLines("x11/9term/Makefile",
		MkCvsID,
		"",
		"DISTNAME=\t9term-1.0",
		"CATEGORIES=\tx11",
		"",
		"COMMENT=\tTerminal",
		"",
		"NO_CHECKSUM=\tyes",
		"",
		".include \"../../mk/bsd.pkg.mk\"")

	t.SetUpVartypes()

	G.Check(t.File("x11/9term"))

	// Since the error is grave enough, the warning about the correct position is suppressed.
	// TODO: Knowing the correct position helps, though.
	t.CheckOutputLines(
		"ERROR: ~/x11/9term/Makefile: Each package must define its LICENSE.")
}

func (s *Suite) Test_VarorderChecker_canonical__diagnostics(c *check.C) {
	t := s.Init(c)

	t.SetUpVartypes()
	mklines := t.NewMkLines("Makefile",
		MkCvsID,
		"",
		"CATEGORIES=     net",
		"",
		"COMMENT=        Comment",
		"LICENSE=        gnu-gpl-v3",
		"",
		"GITHUB_PROJECT= pkgbase",
		"DISTNAME=       v1.0",
		"PKGNAME=        ${GITHUB_PROJECT}-${DISTNAME}",
		"MASTER_SITES=   ${MASTER_SITE_GITHUB:=project/}",
		"DIST_SUBDIR=    ${GITHUB_PROJECT}",
		"",
		"MAINTAINER=     maintainer@example.org",
		"HOMEPAGE=       https://github.com/project/pkgbase/",
		"",
		".include \"../../mk/bsd.pkg.mk\"")

	(&VarorderChecker{mklines}).Check()

	t.CheckOutputLines(
		"WARN: Makefile:3: The canonical order of the variables is " +
			"DISTNAME, PKGNAME, CATEGORIES, " +
			"MASTER_SITES, GITHUB_PROJECT, DIST_SUBDIR, empty line, " +
			"MAINTAINER, HOMEPAGE, COMMENT, LICENSE.")

	// After moving the variables according to the warning:
	mklines = t.NewMkLines("Makefile",
		MkCvsID,
		"",
		"DISTNAME=       v1.0",
		"PKGNAME=        ${GITHUB_PROJECT}-${DISTNAME}",
		"CATEGORIES=     net",
		"MASTER_SITES=   ${MASTER_SITE_GITHUB:=project/}",
		"GITHUB_PROJECT= pkgbase",
		"DIST_SUBDIR=    ${GITHUB_PROJECT}",
		"",
		"MAINTAINER=     maintainer@example.org",
		"HOMEPAGE=       https://github.com/project/pkgbase/",
		"COMMENT=        Comment",
		"LICENSE=        gnu-gpl-v3",
		"",
		".include \"../../mk/bsd.pkg.mk\"")

	(&VarorderChecker{mklines}).Check()

	t.CheckOutputEmpty()
}
