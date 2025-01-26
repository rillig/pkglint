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

// None of the relevant variables is defined,
// so the varorder check is skipped.
func (s *Suite) Test_VarorderChecker_Check__none(c *check.C) {
	t := s.Init(c)

	mklines := t.NewMkLines("Makefile",
		MkCvsID,
		"",
		"IRRELEVANT=",
		"",
		".include \"../../mk/bsd.pkg.mk\"")

	NewVarorderChecker(mklines).Check()

	t.CheckOutputEmpty()
}

// All required variables are defined,
// all optional variables are undefined.
func (s *Suite) Test_VarorderChecker_Check__once(c *check.C) {
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

// All required and optional variables are defined,
// but all repeatable variables are undefined.
func (s *Suite) Test_VarorderChecker_Check__once_and_optional(c *check.C) {
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

// All required and repeatable variables are defined,
// but all optional variables are undefined.
func (s *Suite) Test_VarorderChecker_Check__once_and_many(c *check.C) {
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

// No test for a required variable at the beginning of a section,
// as this constellation does not occur in varorderVariables.

// A required variable in the middle of a section is missing.
func (s *Suite) Test_VarorderChecker_Check__missing_once_middle(c *check.C) {
	t := s.Init(c)

	mklines := t.NewMkLines("Makefile",
		MkCvsID,
		"",
		"CATEGORIES=",
		"",
		// COMMENT is missing.
		".include \"../../mk/bsd.pkg.mk\"")

	NewVarorderChecker(mklines).Check()

	t.CheckOutputLines(
		"WARN: Makefile:5: The variable \"COMMENT\" " +
			"should be defined here.")
}

// A required variable at the end of the section is missing.
func (s *Suite) Test_VarorderChecker_Check__missing_once_end(c *check.C) {
	t := s.Init(c)

	mklines := t.NewMkLines("Makefile",
		MkCvsID,
		"",
		"CATEGORIES=",
		"",
		"COMMENT=",
		// LICENSE is missing.
		"",
		"LICENSE=", // Too late.
		".include \"../../mk/bsd.pkg.mk\"")

	NewVarorderChecker(mklines).Check()

	t.CheckOutputLines(
		"WARN: Makefile:6: The variable \"LICENSE\" " +
			"should be defined here.")
}

func (s *Suite) Test_VarorderChecker_Check__once_repeated(c *check.C) {
	t := s.Init(c)

	mklines := t.NewMkLines("Makefile",
		MkCvsID,
		"",
		"DISTNAME=",
		"PKGNAME=",
		"PKGREVISION=",
		"CATEGORIES=",
		"CATEGORIES=",
		"#MASTER_SITES=",
		"",
		"MAINTAINER=",
		"#HOMEPAGE=",
		"COMMENT=",
		"",
		".include \"../../mk/bsd.pkg.mk\"")

	NewVarorderChecker(mklines).Check()

	// FIXME: wrong line, must be 7.
	t.CheckOutputLines(
		"WARN: Makefile:8: The variable \"CATEGORIES\" " +
			"should only occur once.")
}

// Two optional variables from the same section occur in the wrong order.
// Both variables should be placed in the middle of the section.
func (s *Suite) Test_VarorderChecker_Check__swapped_optional_middle(c *check.C) {
	t := s.Init(c)

	mklines := t.NewMkLines("Makefile",
		MkCvsID,
		"",
		"R_PKGVER=",
		"R_PKGNAME=", // Should be above R_PKGVER.
		"CATEGORIES=",
		"",
		"COMMENT=",
		"LICENSE=",
		"",
		".include \"../../mk/bsd.pkg.mk\"")

	NewVarorderChecker(mklines).Check()

	t.CheckOutputLines(
		"WARN: Makefile:4: The variable \"R_PKGNAME\" " +
			"is misplaced, should be in line 3.")
}

func (s *Suite) Test_VarorderChecker_Check__too_early(c *check.C) {
	t := s.Init(c)

	mklines := t.NewMkLines("Makefile",
		MkCvsID,
		"",
		"GITHUB_PROJECT=",
		"GITHUB_TAG=",
		"DISTNAME=",
		"PKGNAME=",
		"CATEGORIES=",
		"MASTER_SITES=",
		"DIST_SUBDIR=",
		"",
		".include \"../../mk/bsd.pkg.mk\"")

	NewVarorderChecker(mklines).Check()

	t.CheckOutputLines(
		"WARN: Makefile:3: The variable \"GITHUB_PROJECT\" " +
			"occurs too early, should be after \"CATEGORIES\".")
}

func (s *Suite) Test_VarorderChecker_Check__too_late(c *check.C) {
	t := s.Init(c)

	mklines := t.NewMkLines("Makefile",
		MkCvsID,
		"",
		"VERSION=",
		"PKGNAME=",
		"PKGREVISION=",
		"DISTNAME=",
		"CATEGORIES=",
		"",
		"COMMENT=",
		"LICENSE=",
		"",
		".include \"../../mk/bsd.pkg.mk\"")

	NewVarorderChecker(mklines).Check()

	t.CheckOutputLines(
		"WARN: Makefile:6: The variable \"DISTNAME\" " +
			"occurs too late, should be in line 4.")
}

// Two optional variables from the same section occur in the wrong order.
// The second of these variables should be placed at the end of the section.
func (s *Suite) Test_VarorderChecker_Check__swapped_optional_end(c *check.C) {
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

	t.CheckOutputLines(
		"WARN: Makefile:5: The variable \"MASTER_SITES\" is misplaced, " +
			"should be in line 4.")
}

// Two optional variables from different sections occur in the wrong order.
func (s *Suite) Test_VarorderChecker_Check__swapped_across_sections(c *check.C) {
	t := s.Init(c)

	mklines := t.NewMkLines("Makefile",
		MkCvsID,
		"",
		"CATEGORIES=",
		"",
		"COMMENT=",
		"LICENSE=",
		"",
		"NOT_FOR_UNPRIVILEGED=",
		"",
		"DISTNAME=",
		"",
		".include \"../../mk/bsd.pkg.mk\"")

	NewVarorderChecker(mklines).Check()

	t.CheckOutputLines(
		"WARN: Makefile:10: The variable \"DISTNAME\" is misplaced, " +
			"should be in line 3.")
}

// A variable is in the wrong place, it should be at the very top.
func (s *Suite) Test_VarorderChecker_Check__move_to_top(c *check.C) {
	t := s.Init(c)

	mklines := t.NewMkLines("Makefile",
		MkCvsID,
		"",
		"CATEGORIES=",
		"",
		"COMMENT=",
		"LICENSE=",
		"",
		"DISTNAME=", // Should be at the very top.
		"",
		".include \"../../mk/bsd.pkg.mk\"")

	NewVarorderChecker(mklines).Check()

	t.CheckOutputLines(
		"WARN: Makefile:8: The variable \"DISTNAME\" is misplaced, " +
			"should be in line 3.")
}

// A variable is in the wrong place, it should be at the very bottom.
func (s *Suite) Test_VarorderChecker_Check__move_to_bottom(c *check.C) {
	t := s.Init(c)

	mklines := t.NewMkLines("Makefile",
		MkCvsID,
		"",
		"CATEGORIES=",
		"DEPENDS=", // Should be at the very bottom.
		"",
		"COMMENT=",
		"LICENSE=",
		"",
		".include \"../../mk/bsd.pkg.mk\"")

	NewVarorderChecker(mklines).Check()

	t.CheckOutputLines(
		"WARN: Makefile:4: Missing empty line.")
}

// In the middle of a section, there is an extra empty line.
//
// The sections are designed to be small enough to not need these empty lines.
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

	t.CheckOutputLines(
		"WARN: Makefile:9: The variable \"LICENSE\" " +
			"should be defined here.")
}

// Between two sections, there should be one empty line.
//
// The CATEGORIES variable is in a different section from COMMENT and
// LICENSE, with an entirely optional section in-between.
func (s *Suite) Test_VarorderChecker_Check__missing_empty_line(c *check.C) {
	t := s.Init(c)

	mklines := t.NewMkLines("Makefile",
		MkCvsID,
		"",
		"CATEGORIES=",
		"COMMENT=",
		// Empty line missing.
		"LICENSE=",
		"",
		".include \"../../mk/bsd.pkg.mk\"")

	NewVarorderChecker(mklines).Check()

	t.CheckOutputLines(
		"WARN: Makefile:4: Missing empty line.")
}

func (s *Suite) Test_VarorderChecker_Check__commented_once(c *check.C) {
	t := s.Init(c)

	mklines := t.NewMkLines("Makefile",
		MkCvsID,
		"",
		"CATEGORIES=",
		"MASTER_SITES=",
		"",
		"MAINTAINER=",
		"HOMEPAGE=",
		"#COMMENT=",
		"#COMMENT=",
		"COMMENT=",
		"LICENSE=",
		"",
		"CONFIGURE_ARGS=",
		"",
		".include \"../../mk/bsd.pkg.mk\"")

	NewVarorderChecker(mklines).Check()

	// The duplicate COMMENT is not an error since all but one assignment
	// are commented out.
	t.CheckOutputEmpty()
}

// A commented variable assignment satisfies an "optional" as well as a "once"
// requirement.  There may be multiple commented variable assignments, as
// they are not active.
func (s *Suite) Test_VarorderChecker_Check__commented_optional(c *check.C) {
	t := s.Init(c)

	mklines := t.NewMkLines("Makefile",
		MkCvsID,
		"",
		"#CATEGORIES=",
		"",
		"MAINTAINER=",
		"#HOMEPAGE=",
		"#HOMEPAGE=",
		"#HOMEPAGE=",
		"COMMENT=",
		"LICENSE=",
		"",
		".include \"../../mk/bsd.pkg.mk\"")

	NewVarorderChecker(mklines).Check()

	// The duplicate COMMENT is not an error since all but one assignment
	// are commented out.
	t.CheckOutputEmpty()
}

// When extracting the relevant lines, comments are skipped,
// but commented variable assignments are kept.
func (s *Suite) Test_VarorderChecker_relevantLines__comments(c *check.C) {
	t := s.Init(c)

	t.SetUpVartypes()
	mklines := t.NewMkLines("Makefile",
		MkCvsID,
		"",
		"# A comment at the beginning of a section.",
		"#CATEGORIES=",
		"# A comment at the end of a section.",
		"",
		"# A comment between sections.",
		"",
		"# A second comment between sections.",
		"",
		"COMMENT=",
		"LICENSE=",
		"",
		".include \"../../mk/bsd.pkg.mk\"")

	NewVarorderChecker(mklines).Check()

	// The empty line between the comments is not treated as a section
	// separator, so no warning about an extra empty line.
	t.CheckOutputEmpty()
}

// USE_TOOLS is not in the list of varorder variables and is thus skipped when
// collecting the relevant lines.
func (s *Suite) Test_VarorderChecker_relevantLines__foreign(c *check.C) {
	t := s.Init(c)

	mklines := t.NewMkLines("Makefile",
		MkCvsID,
		"",
		"DISTNAME=",
		"USE_TOOLS+=",
		"CATEGORIES=",
		"",
		"MAINTAINER=",
		"#HOMEPAGE=",
		// COMMENT is missing to force a warning,
		// showing that the varorder check is run.
		"LICENSE=",
		"",
		".include \"../../mk/bsd.pkg.mk\"")

	NewVarorderChecker(mklines).Check()

	t.CheckOutputLines(
		"WARN: Makefile:9: The variable \"LICENSE\" " +
			"occurs too early, should be after \"COMMENT\".")
}

// A package makefile that contains conditionals may have good reason to
// deviate from the standard variable order, or may define a "once" variable
// twice, in separate branches of the conditional.  Skip the check in these
// cases, as covering all possible cases would become too complicated.
func (s *Suite) Test_VarorderChecker_relevantLines__directive(c *check.C) {
	t := s.Init(c)

	mklines := t.NewMkLines("Makefile",
		MkCvsID,
		"",
		"DISTNAME=",
		"CATEGORIES=",
		"",
		".if ${DISTNAME:Mdistname-*}",
		"MAINTAINER=",
		".endif",
		"LICENSE=",
		"",
		".include \"../../mk/bsd.pkg.mk\"")

	NewVarorderChecker(mklines).Check()

	// No warning about the missing COMMENT since the .if directive
	// causes the whole check to be skipped.
	t.CheckOutputEmpty()
}

// A package that doesn't include bsd.pkg.mk in the last line of the package
// makefile is not considered simple enough for the varorder check.
func (s *Suite) Test_VarorderChecker_relevantLines__incomplete(c *check.C) {
	t := s.Init(c)

	mklines := t.NewMkLines("Makefile",
		MkCvsID,
		"",
		"DISTNAME=",
		"CATEGORIES=",
		"",
		"LICENSE=")

	NewVarorderChecker(mklines).Check()

	t.CheckOutputEmpty()
}

// A package may include buildlink3.mk files at the end and still be
// considered simple enough for the varorder check.
func (s *Suite) Test_VarorderChecker_relevantLines__buildlink(c *check.C) {
	t := s.Init(c)

	mklines := t.NewMkLines("Makefile",
		MkCvsID,
		"",
		"DISTNAME=",
		"CATEGORIES=",
		"",
		// COMMENT is missing to show that the varorder check is active.
		"LICENSE=",
		"",
		".include \"../../category/package/buildlink3.mk\"",
		".include \"../../mk/bsd.pkg.mk\"")

	NewVarorderChecker(mklines).Check()

	t.CheckOutputLines(
		"WARN: Makefile:6: The variable \"LICENSE\" " +
			"occurs too early, should be after \"COMMENT\".")
}

// A package that includes an arbitrary other makefile may define the
// variables from the varorder check there, which is common for
// Makefile.common files.  In such a case, skip the varorder check.
func (s *Suite) Test_VarorderChecker_relevantLines__include(c *check.C) {
	t := s.Init(c)

	mklines := t.NewMkLines("Makefile",
		MkCvsID,
		"",
		"DISTNAME=",
		"CATEGORIES=",
		"",
		// COMMENT is missing to show that the varorder check is skipped.
		"LICENSE=",
		"",
		".include \"../../category/package/Makefile.common\"",
		".include \"../../mk/bsd.pkg.mk\"")

	NewVarorderChecker(mklines).Check()

	t.CheckOutputEmpty()
}

// A variable that is "once" or "optional" should not occur more than once.
func (s *Suite) Test_VarorderChecker_check(c *check.C) {
	t := s.Init(c)

	mklines := t.NewMkLines("Makefile",
		MkCvsID,
		"",
		"DISTNAME=",
		"CATEGORIES=",
		"CATEGORIES=",
		"",
		"LICENSE=",
		"",
		".include \"../../mk/bsd.pkg.mk\"")

	NewVarorderChecker(mklines).Check()

	t.CheckOutputLines(
		"WARN: Makefile:6: The variable \"CATEGORIES\" should only occur once.")
}

// This package does not declare its LICENSE.
// Since this error is grave enough, treat the LICENSE variable as optional,
// to prevent issuing two very similar diagnostics.
func (s *Suite) Test_VarorderChecker_skipLicenseCheck(c *check.C) {
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

func (s *Suite) Test_VarorderChecker_explain(c *check.C) {
	t := s.Init(c)
	t.SetUpCommandLine("--explain")

	mklines := t.NewMkLines("Makefile",
		MkCvsID,
		"",
		"CATEGORIES=",
		"",
		".include \"../../mk/bsd.pkg.mk\"")

	NewVarorderChecker(mklines).Check()

	t.CheckOutputLines(
		"WARN: Makefile:5: The variable \"COMMENT\" "+
			"should be defined here.",
		"",
		"\tIn simple package Makefiles, some common variables should be",
		"\tarranged in a specific order.",
		"",
		"\tSee doc/Makefile-example for an example Makefile. See the pkgsrc",
		"\tguide, section \"Package components, Makefile\":",
		"\thttps://www.NetBSD.org/docs/pkgsrc/pkgsrc.html#components.Makefile",
		"")
}
