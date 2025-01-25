package pkglint

import "gopkg.in/check.v1"

func (s *Suite) Test_NewVarorderChecker(c *check.C) {
	t := s.Init(c)

	mklines := t.NewMkLines("filename.mk",
		MkCvsID,
		"",
		"VAR=")

	ck := NewVarorderChecker(mklines)

	t.CheckEquals(ck.mklines, mklines)
	t.CheckNotNil(ck.relevant)
	t.CheckEquals(len(ck.relevant), 42)
}

func (s *Suite) Test_VarorderChecker_Check__missing_required(c *check.C) {
	t := s.Init(c)

	mklines := t.NewMkLines("Makefile",
		MkCvsID,
		"",
		"DISTNAME=",
		"CATEGORIES=",
		"",
		".include \"../../mk/bsd.pkg.mk\"")

	NewVarorderChecker(mklines).Check()

	// TODO: Make the warning more specific,
	//  mentioning that COMMENT and LICENSE are missing.
	t.CheckOutputLines(
		"WARN: Makefile:3: The canonical order of the variables is " +
			"DISTNAME, CATEGORIES, empty line, COMMENT, LICENSE.")
}

func (s *Suite) Test_VarorderChecker_Check__only_required(c *check.C) {
	t := s.Init(c)

	mklines := t.NewMkLines("Makefile",
		MkCvsID,
		"",
		"DISTNAME=",
		"CATEGORIES=",
		"",
		"COMMENT=",
		"LICENSE=",
		"",
		".include \"../../mk/bsd.pkg.mk\"")

	NewVarorderChecker(mklines).Check()

	t.CheckOutputEmpty()
}

func (s *Suite) Test_VarorderChecker_Check__all_optional(c *check.C) {
	t := s.Init(c)

	mklines := t.NewMkLines("Makefile",
		MkCvsID,
		"",
		"DISTNAME=",
		"PKGNAME=",
		"R_PKGNAME=",
		"R_PKGVER=",
		"PKGREVISION=",
		"CATEGORIES=",
		"GITHUB_PROJECT=",
		"GITHUB_TAG=",
		"GITHUB_RELEASE=",
		"DIST_SUBDIR=",
		"EXTRACT_SUFX=",
		"",
		"PATCH_SITES=",
		"PATCH_SITE_SUBDIR=",
		"PATCHFILES=",
		"PATCH_DIST_ARGS=",
		"PATCH_DIST_STRIP=",
		"PATCH_DIST_CAT=",
		"",
		"MAINTAINER=",
		"OWNER=",
		"HOMEPAGE=",
		"COMMENT=",
		"LICENSE=",
		"",
		"LICENSE_FILE=",
		"RESTRICTED=",
		"NO_BIN_ON_CDROM=",
		"NO_BIN_ON_FTP=",
		"NO_SRC_ON_CDROM=",
		"NO_SRC_ON_FTP=",
		"",
		"NOT_FOR_UNPRIVILEGED=",
		"",
		".include \"../../mk/bsd.pkg.mk\"")

	NewVarorderChecker(mklines).Check()

	t.CheckOutputEmpty()
}

func (s *Suite) Test_VarorderChecker_Check__all_many(c *check.C) {
	t := s.Init(c)

	mklines := t.NewMkLines("Makefile",
		MkCvsID,
		"",
		"CATEGORIES=",
		"MASTER_SITES=",
		"MASTER_SITES=",
		"DISTFILES=",
		"DISTFILES=",
		"SITES.*=",
		"SITES.*=",
		"",
		"COMMENT=",
		"LICENSE=",
		"",
		"BROKEN_EXCEPT_ON_PLATFORM=",
		"BROKEN_EXCEPT_ON_PLATFORM=",
		"BROKEN_ON_PLATFORM=",
		"BROKEN_ON_PLATFORM=",
		"NOT_FOR_PLATFORM=",
		"NOT_FOR_PLATFORM=",
		"ONLY_FOR_PLATFORM=",
		"ONLY_FOR_PLATFORM=",
		"NOT_FOR_COMPILER=",
		"NOT_FOR_COMPILER=",
		"ONLY_FOR_COMPILER=",
		"ONLY_FOR_COMPILER=",
		"",
		"BUILD_DEPENDS=",
		"BUILD_DEPENDS=",
		"TOOL_DEPENDS=",
		"TOOL_DEPENDS=",
		"DEPENDS=",
		"DEPENDS=",
		"",
		".include \"../../mk/bsd.pkg.mk\"")

	NewVarorderChecker(mklines).Check()

	t.CheckOutputEmpty()
}

func (s *Suite) Test_VarorderChecker_Check__extra_empty_line(c *check.C) {
	t := s.Init(c)

	mklines := t.NewMkLines("Makefile",
		MkCvsID,
		"",
		"CATEGORIES=",
		"MASTER_SITES=",
		"",
		"MAINTAINER=",
		"HOMEPAGE=",
		"COMMENT=",
		"",
		"LICENSE=",
		"",
		".include \"../../mk/bsd.pkg.mk\"")

	NewVarorderChecker(mklines).Check()

	// TODO: Be more specific, mentioning that the empty line above
	//  LICENSE should be removed.
	t.CheckOutputLines(
		"WARN: Makefile:3: The canonical order of the variables is " +
			"CATEGORIES, MASTER_SITES, empty line, " +
			"MAINTAINER, HOMEPAGE, COMMENT, LICENSE.")
}

func (s *Suite) Test_VarorderChecker_Check__swapped(c *check.C) {
	t := s.Init(c)

	mklines := t.NewMkLines("Makefile",
		MkCvsID,
		"",
		"CATEGORIES=",
		"DIST_SUBDIR=",
		"MASTER_SITES=",
		"",
		"MAINTAINER=",
		"HOMEPAGE=",
		"COMMENT=",
		"LICENSE=",
		"",
		".include \"../../mk/bsd.pkg.mk\"")

	NewVarorderChecker(mklines).Check()

	// TODO: Be more specific, mentioning that DIST_SUBDIR and
	//  MASTER_SITES should be swapped.
	t.CheckOutputLines(
		"WARN: Makefile:3: The canonical order of the variables is " +
			"CATEGORIES, MASTER_SITES, DIST_SUBDIR, empty line, " +
			"MAINTAINER, HOMEPAGE, COMMENT, LICENSE.")
}

func (s *Suite) Test_VarorderChecker_Check__in_wrong_section(c *check.C) {
	t := s.Init(c)

	mklines := t.NewMkLines("Makefile",
		MkCvsID,
		"",
		"CATEGORIES=",
		"MASTER_SITES=",
		"",
		"MAINTAINER=",
		"HOMEPAGE=",
		"COMMENT=",
		"",
		"DIST_SUBDIR=", // Is in the wrong section.
		"WRKSRC=",
		"INSTALLATION_DIRS=",
		"",
		".include \"../../mk/bsd.pkg.mk\"")

	NewVarorderChecker(mklines).Check()

	// TODO: Be more specific, mentioning that DIST_SUBDIR should
	//  be at the end of the first section, below MASTER_SITES.
	t.CheckOutputLines(
		"WARN: Makefile:3: The canonical order of the variables is " +
			"CATEGORIES, MASTER_SITES, DIST_SUBDIR, empty line, " +
			"MAINTAINER, HOMEPAGE, COMMENT, LICENSE.")
}

func (s *Suite) Test_VarorderChecker_Check__duplicate_in_comment(c *check.C) {
	t := s.Init(c)

	mklines := t.NewMkLines("Makefile",
		MkCvsID,
		"",
		"CATEGORIES=",
		"MASTER_SITES=",
		"",
		"MAINTAINER=",
		"#HOMEPAGE=",
		"HOMEPAGE=",
		"COMMENT=",
		"LICENSE=",
		"",
		"CONFIGURE_ARGS=",
		"",
		".include \"../../mk/bsd.pkg.mk\"")

	NewVarorderChecker(mklines).Check()

	// FIXME: The duplicate HOMEPAGE is not an error since at most one
	//  of them is active.
	t.CheckOutputLines(
		"WARN: Makefile:3: The canonical order of the variables is " +
			"CATEGORIES, MASTER_SITES, " +
			"empty line, " +
			"MAINTAINER, HOMEPAGE, COMMENT, LICENSE.")
}

func (s *Suite) Test_VarorderChecker_relevantLines__comments(c *check.C) {
	t := s.Init(c)

	t.SetUpVartypes()
	mklines := t.NewMkLines("Makefile",
		MkCvsID,
		"",
		"# A comment at the beginning of a section.",
		"CATEGORIES=     net",
		"# A comment at the end of a section.",
		"",
		"# A comment between sections.",
		"",
		"# A second comment between sections.",
		"",
		"MAINTAINER=     maintainer@example.org",
		"HOMEPAGE=       https://github.com/project/pkgbase/",
		"COMMENT=        Comment",
		"LICENSE=        gnu-gpl-v3",
		"",
		".include \"../../mk/bsd.pkg.mk\"")

	NewVarorderChecker(mklines).Check()

	// The empty line between the comments is not treated as a section separator.
	t.CheckOutputEmpty()
}

// A commented variable assignment is treated in the same way as an active
// variable assignment, as the varorder check only ensures the correct
// ordering, and not that each required variable is actually defined.
// Ensuring that every package has a homepage would be a suitable runtime
// check instead.
//
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
func (s *Suite) Test_VarorderChecker_relevantLines__commented_variable_assignment(c *check.C) {
	t := s.Init(c)

	mklines := t.NewMkLines("Makefile",
		MkCvsID,
		"",
		"DISTNAME=\tdistname-1.0",
		// CATEGORIES is missing to force the warning.
		"",
		"MAINTAINER=\tpkgsrc-users@NetBSD.org",
		"#HOMEPAGE=\thttps://example.org/",
		"#HOMEPAGE=\thttps://example.org/",
		"#HOMEPAGE=\thttps://example.org/",
		"#HOMEPAGE=\thttps://example.org/",
		"#HOMEPAGE=\thttps://example.org/",
		"#HOMEPAGE=\thttps://example.org/",
		"COMMENT=\tComment",
		"LICENSE=\tgnu-gpl-v2")

	NewVarorderChecker(mklines).Check()

	t.CheckOutputLines(
		"WARN: Makefile:3: The canonical order of the variables is " +
			"DISTNAME, CATEGORIES, " +
			"empty line, " +
			"MAINTAINER, HOMEPAGE, COMMENT, LICENSE.")
}

// USE_TOOLS is not in the list of varorder variables and is thus skipped when
// collecting the relevant lines.
func (s *Suite) Test_VarorderChecker_relevantLines__foreign_variable(c *check.C) {
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
		"LICENSE=\tgnu-gpl-v2")

	NewVarorderChecker(mklines).Check()

	// TODO: Be more specific in the warning,
	//  mentioning that COMMENT is missing.
	t.CheckOutputLines(
		"WARN: Makefile:3: The canonical order of the variables is " +
			"DISTNAME, CATEGORIES, empty line, " +
			"MAINTAINER, HOMEPAGE, COMMENT, LICENSE.")
}

// Package makefiles that contain conditionals may have good reason to deviate
// from the standard variable order, or may define a "once" variable twice, in
// separate branches of the conditional.  Skip the check in these cases.
func (s *Suite) Test_VarorderChecker_skip__directive(c *check.C) {
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

	NewVarorderChecker(mklines).Check()

	// No warning about the missing COMMENT since the .if directive
	// causes the whole check to be skipped.
	t.CheckOutputEmpty()
}

// This package does not declare its LICENSE.
// Since this error is grave enough, skip the varorder check,
// as its warning would look redundant.
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

	// TODO: Be more specific in the warning,
	//  mentioning in which line the LICENSE should be inserted.
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

	NewVarorderChecker(mklines).Check()

	t.CheckOutputLines(
		"WARN: Makefile:3: The canonical order of the variables is " +
			"DISTNAME, PKGNAME, CATEGORIES, " +
			"MASTER_SITES, GITHUB_PROJECT, DIST_SUBDIR, empty line, " +
			"MAINTAINER, HOMEPAGE, COMMENT, LICENSE.")

	// After ordering the variables according to the warning:
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

	NewVarorderChecker(mklines).Check()

	t.CheckOutputEmpty()
}
