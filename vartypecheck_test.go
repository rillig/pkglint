package main

import (
	check "gopkg.in/check.v1"
)

func (s *Suite) TestVartypeCheck_AwkCommand(c *check.C) {
	runVartypeChecks("PLIST_AWK", "+=", (*VartypeCheck).AwkCommand,
		"{print $0}")

	c.Check(s.Output(), equals, "")
}

func (s *Suite) TestVartypeCheck_BasicRegularExpression(c *check.C) {
	runVartypeChecks("REPLACE_FILES.pl", "=", (*VartypeCheck).BasicRegularExpression,
		".*\\.pl$")

	c.Check(s.Output(), equals, "")
}

func (s *Suite) TestVartypeCheck_BuildlinkDepmethod(c *check.C) {
	runVartypeChecks("BUILDLINK_DEPMETHOD.libc", "?=", (*VartypeCheck).BuildlinkDepmethod,
		"full",
		"unknown")

	c.Check(s.Output(), equals, "WARN: fname:2: Invalid dependency method \"unknown\". Valid methods are \"build\" or \"full\".\n")
}

func (s *Suite) TestVartypeCheck_Category(c *check.C) {
	s.CreateTmpFile(c, "filesyscategory/Makefile", "# empty\n")
	G.currentDir = s.tmpdir
	G.curPkgsrcdir = "."

	runVartypeChecks("CATEGORIES", "=", (*VartypeCheck).Category,
		"chinese",
		"arabic",
		"filesyscategory")

	c.Check(s.Output(), equals, "ERROR: fname:2: Invalid category \"arabic\".\n")
}

func (s *Suite) TestVartypeCheck_CFlag(c *check.C) {
	runVartypeChecks("CFLAGS", "+=", (*VartypeCheck).CFlag,
		"-Wall",
		"/W3",
		"target:sparc64",
		"-std=c99",
		"-XX:+PrintClassHistogramAfterFullGC")

	c.Check(s.Output(), equals, ""+
		"WARN: fname:2: Compiler flag \"/W3\" should start with a hyphen.\n"+
		"WARN: fname:3: Compiler flag \"target:sparc64\" should start with a hyphen.\n"+
		"WARN: fname:5: Unknown compiler flag \"-XX:+PrintClassHistogramAfterFullGC\".\n")
}

func (s *Suite) TestVartypeCheck_Comment(c *check.C) {
	runVartypeChecks("COMMENT", "=", (*VartypeCheck).Comment,
		"Versatile Programming Language",
		"SHORT_DESCRIPTION_OF_THE_PACKAGE",
		"A great package.",
		"some packages need a very very long comment to explain their basic usefulness")

	c.Check(s.Output(), equals, ""+
		"ERROR: fname:2: COMMENT must be set.\n"+
		"WARN: fname:3: COMMENT should not begin with \"A\".\n"+
		"WARN: fname:3: COMMENT should not end with a period.\n"+
		"WARN: fname:4: COMMENT should start with a capital letter.\n"+
		"WARN: fname:4: COMMENT should not be longer than 70 characters.\n")
}

func (s *Suite) TestVartypeCheck_Dependency(c *check.C) {
	runVartypeChecks("CONFLICTS", "+=", (*VartypeCheck).Dependency,
		"Perl",
		"perl5>=5.22",
		"perl5-*",
		"perl5-5.22.*",
		"perl5-[5.10-5.22]*",
		"py-docs",
		"perl5-5.22.*{,nb*}")

	c.Check(s.Output(), equals, ""+
		"WARN: fname:1: Unknown dependency format: Perl\n"+
		"WARN: fname:3: Please use \"perl5-[0-9]*\" instead of \"perl5-*\".\n"+
		"WARN: fname:4: Please append \"{,nb*}\" to the version number of this dependency.\n"+
		"WARN: fname:5: Only [0-9]* is allowed in the numeric part of a dependency.\n"+
		"ERROR: fname:6: Unknown dependency pattern \"py-docs\".\n")
}

func (s *Suite) TestVartypeCheck_DependencyWithPatch(c *check.C) {
	s.CreateTmpFile(c, "x11/alacarte/Makefile", "# empty\n")
	s.CreateTmpFile(c, "category/package/Makefile", "# empty\n")
	G.globalData.pkgsrcdir = s.tmpdir
	G.currentDir = s.tmpdir + "/category/package"
	G.curPkgsrcdir = "../.."

	runVartypeChecks("DEPENDS", "+=", (*VartypeCheck).DependencyWithPath,
		"Perl",
		"perl5>=5.22:../perl5",
		"perl5>=5.24:../../lang/perl5",
		"broken0.12.1:../../x11/alacarte",
		"broken[0-9]*:../../x11/alacarte",
		"broken[0-9]*../../x11/alacarte",
		"broken>=:../../x11/alacarte",
		"broken=0:../../x11/alacarte",
		"broken=:../../x11/alacarte",
		"broken-:../../x11/alacarte",
		"broken>:../../x11/alacarte")

	c.Check(s.Output(), equals, ""+
		"WARN: fname:1: Unknown dependency format.\n"+
		"WARN: fname:2: Dependencies should have the form \"../../category/package\".\n"+
		"ERROR: fname:3: \"../../lang/perl5\" does not exist.\n"+
		"ERROR: fname:3: There is no package in \"lang/perl5\".\n"+
		"WARN: fname:3: Please use USE_TOOLS+=perl:run instead of this dependency.\n"+
		"ERROR: fname:4: Unknown dependency pattern \"broken0.12.1\".\n"+
		"ERROR: fname:5: Unknown dependency pattern \"broken[0-9]*\".\n"+
		"WARN: fname:6: Unknown dependency format.\n"+
		"ERROR: fname:7: Unknown dependency pattern \"broken>=\".\n"+
		"ERROR: fname:8: Unknown dependency pattern \"broken=0\".\n"+
		"ERROR: fname:9: Unknown dependency pattern \"broken=\".\n"+
		"ERROR: fname:10: Unknown dependency pattern \"broken-\".\n"+
		"ERROR: fname:11: Unknown dependency pattern \"broken>\".\n")
}

func (s *Suite) TestVartypeCheck_DistSuffix(c *check.C) {
	runVartypeChecks("EXTRACT_SUFX", "=", (*VartypeCheck).DistSuffix,
		".tar.gz",
		".tar.bz2")

	c.Check(s.Output(), equals, "NOTE: fname:1: EXTRACT_SUFX is \".tar.gz\" by default, so this definition may be redundant.\n")
}

func (s *Suite) TestVartypeCheck_EmulPlatform(c *check.C) {
	runVartypeChecks("EMUL_PLATFORM", "=", (*VartypeCheck).EmulPlatform,
		"linux-i386",
		"nextbsd-8087",
		"${LINUX}")

	c.Check(s.Output(), equals, ""+
		"WARN: fname:2: Unknown operating system: nextbsd\n"+
		"WARN: fname:2: Unknown hardware architecture: 8087\n"+
		"WARN: fname:3: \"${LINUX}\" is not a valid emulation platform.\n")
}

func (s *Suite) TestVartypeCheck_FetchURL(c *check.C) {
	G.globalData.masterSiteUrls = map[string]string{
		"https://github.com/":         "MASTER_SITE_GITHUB",
		"http://ftp.gnu.org/pub/gnu/": "MASTER_SITE_GNU",
	}
	G.globalData.masterSiteVars = map[string]bool{
		"MASTER_SITE_GITHUB": true,
		"MASTER_SITE_GNU":    true,
	}

	runVartypeChecks("MASTER_SITES", "=", (*VartypeCheck).FetchURL,
		"https://github.com/example/project/",
		"http://ftp.gnu.org/pub/gnu/bison", // Missing a slash at the end
		"${MASTER_SITE_GNU:=bison}",
		"${MASTER_SITE_INVALID:=subdir/}")

	c.Check(s.Output(), equals, ""+
		"WARN: fname:1: Please use ${MASTER_SITE_GITHUB:=example/} instead of \"https://github.com/example/project/\".\n"+
		"WARN: fname:1: Run \""+confMake+" help topic=github\" for further tips.\n"+
		"WARN: fname:2: Please use ${MASTER_SITE_GNU:=bison} instead of \"http://ftp.gnu.org/pub/gnu/bison\".\n"+
		"ERROR: fname:3: The subdirectory in MASTER_SITE_GNU must end with a slash.\n"+
		"ERROR: fname:4: MASTER_SITE_INVALID does not exist.\n")
}

func (s *Suite) TestVartypeCheck_Filename(c *check.C) {
	runVartypeChecks("FNAME", "=", (*VartypeCheck).Filename,
		"Filename with spaces.docx",
		"OS/2-manual.txt")

	c.Check(s.Output(), equals, ""+
		"WARN: fname:1: \"Filename with spaces.docx\" is not a valid filename.\n"+
		"WARN: fname:2: A filename should not contain a slash.\n")
}

func (s *Suite) TestVartypeCheck_MailAddress(c *check.C) {
	runVartypeChecks("MAINTAINER", "=", (*VartypeCheck).MailAddress,
		"pkgsrc-users@netbsd.org")

	c.Check(s.Output(), equals, "WARN: fname:1: Please write \"NetBSD.org\" instead of \"netbsd.org\".\n")
}

func (s *Suite) TestVartypeCheck_Message(c *check.C) {
	runVartypeChecks("SUBST_MESSAGE.id", "=", (*VartypeCheck).Message,
		"\"Correct paths\"",
		"Correct paths")

	c.Check(s.Output(), equals, "WARN: fname:1: SUBST_MESSAGE.id should not be quoted.\n")
}

func (s *Suite) TestVartypeCheck_Pathlist(c *check.C) {
	runVartypeChecks("PATH", "=", (*VartypeCheck).Pathlist,
		"/usr/bin:/usr/sbin:.:${LOCALBASE}/bin")

	c.Check(s.Output(), equals, "WARN: fname:1: All components of PATH (in this case \".\") should be absolute paths.\n")
}

func (s *Suite) TestVartypeCheck_PkgRevision(c *check.C) {
	runVartypeChecks("PKGREVISION", "=", (*VartypeCheck).PkgRevision,
		"3a")

	c.Check(s.Output(), equals, ""+
		"WARN: fname:1: PKGREVISION must be a positive integer number.\n"+
		"ERROR: fname:1: PKGREVISION only makes sense directly in the package Makefile.\n")

	runVartypeChecksFname("Makefile", "PKGREVISION", "=", (*VartypeCheck).PkgRevision,
		"3")

	c.Check(s.Output(), equals, "")
}

func (s *Suite) TestVartypeCheck_PlatformTriple(c *check.C) {
	runVartypeChecks("ONLY_FOR_PLATFORM", "=", (*VartypeCheck).PlatformTriple,
		"linux-i386",
		"nextbsd-5.0-8087",
		"${LINUX}")

	c.Check(s.Output(), equals, ""+
		"WARN: fname:1: \"linux-i386\" is not a valid platform triple.\n"+
		"WARN: fname:2: Unknown operating system: nextbsd\n"+
		"WARN: fname:2: Unknown hardware architecture: 8087\n")
}

func (s *Suite) TestVartypeCheck_SedCommands(c *check.C) {

	runVartypeChecks("SUBST_SED.dummy", "=", (*VartypeCheck).SedCommands,
		"s,@COMPILER@,gcc,g",
		"-e s,a,b, -e a,b,c,")

	c.Check(s.Output(), equals, ""+
		"NOTE: fname:1: Please always use \"-e\" in sed commands, even if there is only one substitution.\n"+
		"NOTE: fname:2: Each sed command should appear in an assignment of its own.\n")
}

func (s *Suite) TestVartypeCheck_Stage(c *check.C) {
	runVartypeChecks("SUBST_STAGE.dummy", "=", (*VartypeCheck).Stage,
		"post-patch",
		"post-modern",
		"pre-test")

	c.Check(s.Output(), equals, "WARN: fname:2: Invalid stage name \"post-modern\". Use one of {pre,do,post}-{extract,patch,configure,build,test,install}.\n")
}

func (s *Suite) TestVartypeCheck_Yes(c *check.C) {
	runVartypeChecks("APACHE_MODULE", "=", (*VartypeCheck).Yes,
		"yes",
		"no",
		"${YESVAR}")

	c.Check(s.Output(), equals, ""+
		"WARN: fname:2: APACHE_MODULE should be set to YES or yes.\n"+
		"WARN: fname:3: APACHE_MODULE should be set to YES or yes.\n")
}

func runVartypeChecks(varname, op string, checker func(*VartypeCheck), values ...string) {
	for i, value := range values {
		mkline := NewMkLine(NewLine("fname", i+1, varname+op+value, nil))
		valueNovar := mkline.withoutMakeVariables(value, true)
		vc := &VartypeCheck{mkline, varname, op, value, valueNovar, "", true, guNotGuessed}
		checker(vc)
	}
}

func runVartypeChecksFname(fname, varname, op string, checker func(*VartypeCheck), values ...string) {
	for i, value := range values {
		mkline := NewMkLine(NewLine(fname, i+1, varname+op+value, nil))
		valueNovar := mkline.withoutMakeVariables(value, true)
		vc := &VartypeCheck{mkline, varname, op, value, valueNovar, "", true, guNotGuessed}
		checker(vc)
	}
}
