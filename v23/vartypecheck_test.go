package pkglint

import "gopkg.in/check.v1"

func (s *Suite) Test_VartypeCheck_Errorf(c *check.C) {
	t := s.Init(c)

	mkline := t.NewMkLine("filename.mk", 123, "")
	cv := VartypeCheck{MkLine: mkline}

	cv.Errorf("Error %q.", "message")

	t.CheckOutputLines(
		"ERROR: filename.mk:123: Error \"message\".")
}

func (s *Suite) Test_VartypeCheck_Warnf(c *check.C) {
	t := s.Init(c)

	mkline := t.NewMkLine("filename.mk", 123, "")
	cv := VartypeCheck{MkLine: mkline}

	cv.Warnf("Warning %q.", "message")

	t.CheckOutputLines(
		"WARN: filename.mk:123: Warning \"message\".")
}

func (s *Suite) Test_VartypeCheck_Notef(c *check.C) {
	t := s.Init(c)

	mkline := t.NewMkLine("filename.mk", 123, "")
	cv := VartypeCheck{MkLine: mkline}

	cv.Notef("Note %q.", "message")

	t.CheckOutputLines(
		"NOTE: filename.mk:123: Note \"message\".")
}

func (s *Suite) Test_VartypeCheck_Explain(c *check.C) {
	t := s.Init(c)

	t.SetUpCommandLine("--explain")
	mkline := t.NewMkLine("filename.mk", 123, "")
	cv := VartypeCheck{MkLine: mkline}

	cv.Notef("Note %q.", "message")
	cv.Explain("Explanation.")

	t.CheckOutputLines(
		"NOTE: filename.mk:123: Note \"message\".",
		"",
		"\tExplanation.",
		"")
}

func (s *Suite) Test_VartypeCheck_Autofix(c *check.C) {
	t := s.Init(c)

	mkline := t.NewMkLine("filename.mk", 123, "")
	cv := VartypeCheck{MkLine: mkline}

	t.CheckEquals(cv.Autofix(), mkline.Autofix())
}

func (s *Suite) Test_VartypeCheck_WithValue(c *check.C) {
	t := s.Init(c)

	cv := VartypeCheck{
		Varname:    "OLD",
		Value:      "oldValue${VAR}",
		ValueNoVar: "oldValue",
	}

	copied := cv.WithValue("newValue${NEW_VAR}")

	t.CheckEquals(copied.Varname, "OLD")
	t.CheckEquals(copied.Value, "newValue${NEW_VAR}")
	t.CheckEquals(copied.ValueNoVar, "newValue")
	t.CheckEquals(cv.Value, "oldValue${VAR}")
	t.CheckEquals(cv.ValueNoVar, "oldValue")
}

func (s *Suite) Test_VartypeCheck_WithVarnameValue(c *check.C) {
	t := s.Init(c)

	cv := VartypeCheck{
		Varname:    "OLD",
		Value:      "oldValue${VAR}",
		ValueNoVar: "oldValue",
	}

	copied := cv.WithVarnameValue("NEW", "newValue${NEW_VAR}")

	t.CheckEquals(copied.Varname, "NEW")
	t.CheckEquals(copied.Value, "newValue${NEW_VAR}")
	t.CheckEquals(copied.ValueNoVar, "newValue")
	t.CheckEquals(cv.Value, "oldValue${VAR}")
	t.CheckEquals(cv.ValueNoVar, "oldValue")
}

func (s *Suite) Test_VartypeCheck_WithVarnameValueMatch(c *check.C) {
	t := s.Init(c)

	cv := VartypeCheck{
		Varname:    "OLD",
		Op:         opAssign,
		Value:      "oldValue${VAR}",
		ValueNoVar: "oldValue",
	}

	copied := cv.WithVarnameValueMatch("NEW", "newValue${NEW_VAR}")

	t.CheckEquals(copied.Varname, "NEW")
	t.CheckEquals(copied.Op, opUseMatch)
	t.CheckEquals(copied.Value, "newValue${NEW_VAR}")
	t.CheckEquals(copied.ValueNoVar, "newValue")
	t.CheckEquals(cv.Varname, "OLD")
	t.CheckEquals(cv.Op, opAssign)
	t.CheckEquals(cv.Value, "oldValue${VAR}")
	t.CheckEquals(cv.ValueNoVar, "oldValue")
}

func (s *Suite) Test_VartypeCheck_AwkCommand(c *check.C) {
	t := s.Init(c)
	vt := NewVartypeCheckTester(t, BtAwkCommand)

	vt.Varname("PRINT_PLIST_AWK")
	vt.Op(opAssignAppend)
	vt.Values(
		"{print $0}",
		"{print $$0}")
	t.DisableTracing()
	vt.Values(
		"{print $0}",
		"{print $$0}")

	// TODO: In this particular context of AWK programs, $$0 is not a shell variable.
	//  The warning should be adjusted to reflect this.

	vt.Output(
		"WARN: filename.mk:1: $0 is ambiguous. "+
			"Use ${0} if you mean a Make variable or $$0 if you mean a shell variable.",
		"WARN: filename.mk:11: $0 is ambiguous. "+
			"Use ${0} if you mean a Make variable or $$0 if you mean a shell variable.")
}

func (s *Suite) Test_VartypeCheck_BasicRegularExpression(c *check.C) {
	t := s.Init(c)
	vt := NewVartypeCheckTester(t, BtBasicRegularExpression)

	vt.Varname("CHECK_FILES_SKIP")
	vt.Values(
		".*\\.pl$",
		".*\\.pl$$",
		"\u1E9E",
		"\\(capture\\)\\1",
		"\\+")

	vt.Output(
		"WARN: filename.mk:1: Internal pkglint error in MkLine.Tokenize at \"$\".",
		"WARN: filename.mk:5: In a basic regular expression, a backslash followed by \"+\" is undefined.")

	// Check for special characters that appear outside of character classes.
	vt.Values(
		"\u0007",
		" !\"\"\\#$$%&''()*+,-./09:;<=>?",
		"@AZ[\\\\]^_``az{|}~",
		"\t")

	vt.OutputEmpty()

	vt.Values(
		"?",
		"\\?",
		"\\\\?",
		"\\\\\\?")

	vt.Output(
		"WARN: filename.mk:22: In a basic regular expression, a backslash followed by \"?\" is undefined.",
		"WARN: filename.mk:24: In a basic regular expression, a backslash followed by \"?\" is undefined.")

	vt.Values(
		"package-[0-9][0-9.]*",
		"unclosed-[",
		// TODO: Warn about the unclosed character class.
		"backslash-[\\")

	vt.OutputEmpty()

	vt.Values(
		// TODO: Warn about incomplete regular expression escape
		"\\",
		"\\\\")

	vt.OutputEmpty()

	vt.Values(
		"${VAR}*",
		"\\?${VAR}\\/")

	vt.Output(
		"WARN: filename.mk:52: In a basic regular expression, a backslash followed by \"?\" is undefined.",
		"WARN: filename.mk:52: In a basic regular expression, a backslash followed by \"/\" is undefined.")
}

func (s *Suite) Test_VartypeCheck_BuildlinkDepmethod(c *check.C) {
	vt := NewVartypeCheckTester(s.Init(c), BtBuildlinkDepmethod)

	vt.Varname("BUILDLINK_DEPMETHOD.libc")
	vt.Op(opAssignDefault)
	vt.Values(
		"full",
		"unknown",
		"${BUILDLINK_DEPMETHOD.kernel}")

	vt.Output(
		"WARN: filename.mk:2: Invalid dependency method \"unknown\". Valid methods are \"build\" or \"full\".")
}

func (s *Suite) Test_VartypeCheck_Category(c *check.C) {
	t := s.Init(c)
	vt := NewVartypeCheckTester(t, BtCategory)

	t.CreateFileLines("filesyscategory/Makefile",
		"# empty")
	t.CreateFileLines("wip/Makefile",
		"# empty")

	vt.Varname("CATEGORIES")
	vt.Values(
		"chinese",
		"arabic",
		"filesyscategory",
		"wip",
		"gnome",
		"gnustep",
		"java",
		"kde",
		"korean",
		"linux",
		"local",
		"lua",
		"plan9",
		"R",
		"ruby",
		"scm",
		"tcl",
		"tk",
		"windowmaker",
		"xmms")

	vt.Output(
		"ERROR: filename.mk:2: Invalid category \"arabic\".",
		"ERROR: filename.mk:4: Invalid category \"wip\".")
}

func (s *Suite) Test_VartypeCheck_CFlag(c *check.C) {
	vt := NewVartypeCheckTester(s.Init(c), BtCFlag)

	vt.tester.SetUpTool("pkg-config", "", AtRunTime)

	vt.Varname("CFLAGS")
	vt.Op(opAssignAppend)
	vt.Values(
		"-Wall",
		"/W3",
		"target:sparc64",
		"-std=c99",
		"-XX:+PrintClassHistogramAfterFullGC",
		"`pkg-config pidgin --cflags`",
		"-c99",
		"-c",
		"-no-integrated-as",
		"-pthread",
		"`pkg-config`_plus")
	vt.OutputEmpty()

	vt.Values(
		"-L${PREFIX}/lib",
		"-L${PREFIX}/lib64",
		"-lncurses",
		"-DMACRO=\\\"",
		"-DMACRO=\\'")

	vt.Output(
		"WARN: filename.mk:21: \"-L${PREFIX}/lib\" is a linker flag "+
			"and belong to LDFLAGS, LIBS or LDADD instead of CFLAGS.",
		"WARN: filename.mk:22: \"-L${PREFIX}/lib64\" is a linker flag "+
			"and belong to LDFLAGS, LIBS or LDADD instead of CFLAGS.",
		"WARN: filename.mk:23: \"-lncurses\" is a linker flag "+
			"and belong to LDFLAGS, LIBS or LDADD instead of CFLAGS.",
		"WARN: filename.mk:24: Compiler flag \"-DMACRO=\\\\\\\"\" "+
			"has unbalanced double quotes.",
		"WARN: filename.mk:25: Compiler flag \"-DMACRO=\\\\'\" "+
			"has unbalanced single quotes.")

	vt.Op(opUseMatch)
	vt.Values(
		"anything")

	vt.OutputEmpty()
}

func (s *Suite) Test_VartypeCheck_Comment(c *check.C) {
	t := s.Init(c)

	pkg := NewPackage(t.File("category/converter"))
	pkg.EffectivePkgbase = "converter"
	vt := NewVartypeCheckTester(t, BtComment)
	vt.Package(pkg)

	vt.Varname("COMMENT")
	vt.Values(
		"Versatile Programming Language",
		"TODO: Short description of the package",
		"A great package.",
		"some packages need a very very long comment to explain their basic usefulness",
		"\"Quoting the comment is wrong\"",
		"'Quoting the comment is wrong'",
		"Package is a great package",
		"Package is an awesome package",
		"The Big New Package is a great package",
		"Converter converts between measurement units",
		"Converter is a unit converter",
		"\"Official\" office suite",
		"'SQL injection fuzzer",
		"TCR (Test && Commit || Revert) utility")

	vt.Output(
		"ERROR: filename.mk:2: COMMENT must be set.",
		"WARN: filename.mk:3: COMMENT should not begin with \"A\".",
		"WARN: filename.mk:3: COMMENT should not end with a period.",
		"WARN: filename.mk:4: COMMENT should start with a capital letter.",
		"WARN: filename.mk:4: COMMENT should not be longer than 70 characters.",
		"ERROR: filename.mk:5: COMMENT must not be enclosed in quotes.",
		"ERROR: filename.mk:6: COMMENT must not be enclosed in quotes.",
		"WARN: filename.mk:7: COMMENT should not contain \"is a\".",
		"WARN: filename.mk:8: COMMENT should not contain \"is an\".",
		"WARN: filename.mk:9: COMMENT should not contain \"is a\".",
		"WARN: filename.mk:10: COMMENT should not start with the package name.",
		"WARN: filename.mk:11: COMMENT should not start with the package name.",
		"WARN: filename.mk:11: COMMENT should not contain \"is a\".",
		"ERROR: filename.mk:14: COMMENT must not contain \"|\".")
}

func (s *Suite) Test_VartypeCheck_ConfFiles(c *check.C) {
	vt := NewVartypeCheckTester(s.Init(c), BtConfFiles)

	vt.Varname("CONF_FILES")
	vt.Op(opAssignAppend)
	vt.Values(
		"single/file",
		"share/etc/config ${PKG_SYSCONFDIR}/etc/config",
		"share/etc/config ${PKG_SYSCONFBASE}/etc/config file",
		"share/etc/config ${PREFIX}/etc/config share/etc/config2 ${VARBASE}/config2",
		"share/etc/bootrc /etc/bootrc")

	vt.Output(
		"WARN: filename.mk:1: Values for CONF_FILES should always be pairs of paths.",
		"WARN: filename.mk:3: Values for CONF_FILES should always be pairs of paths.",
		"WARN: filename.mk:5: The destination file \"/etc/bootrc\" should start with a variable reference.")

	// See pkgsrc/regress/conf-files-spaces.
	vt.Values(
		"back\\ slash.conf ${PKG_SYSCONFDIR}/back\\ slash.conf",
		"\"d quot.conf\" \"${PKG_SYSCONFDIR}/d quot.conf\"",
		"'s quot.conf' '${PKG_SYSCONFDIR}/''s quot.conf'")
	vt.OutputEmpty()

	vt.Values(
		"\\*.conf ${PKG_SYSCONFDIR}/\\*.conf")
	vt.Output(
		"WARN: filename.mk:21: The pathname \"\\\\*.conf\" contains the invalid character \"*\".",
		"WARN: filename.mk:21: The pathname \"${PKG_SYSCONFDIR}/\\\\*.conf\" contains the invalid character \"*\".")

}

func (s *Suite) Test_VartypeCheck_DependencyWithPath(c *check.C) {
	t := s.Init(c)

	t.CreateFileLines("category/package/Makefile")
	t.CreateFileLines("category/package/files/dummy")
	t.CreateFileLines("databases/py-sqlite3/Makefile")
	t.CreateFileLines("devel/gettext/Makefile")
	t.CreateFileLines("devel/gmake/Makefile")
	t.CreateFileLines("devel/py-module/Makefile")
	t.CreateFileLines("x11/alacarte/Makefile")
	pkg := NewPackage(t.File("category/package"))
	vt := NewVartypeCheckTester(t, BtDependencyWithPath)
	vt.Package(pkg)

	vt.Varname("DEPENDS")
	vt.Op(opAssignAppend)
	vt.File(pkg.File("filename.mk"))
	vt.Values(
		"Perl",
		"perl5>=5.22:../perl5",
		"perl5>=5.24:../../lang/perl5",
		"gtk2+>=2.16:../../x11/alacarte",
		"gettext-[0-9]*:../../devel/gettext",
		"gmake-[0-9]*:../../devel/gmake")

	vt.Output(
		"ERROR: ~/category/package/filename.mk:1: Invalid dependency pattern \"Perl\".",
		"WARN: ~/category/package/filename.mk:2: Dependency paths should have the form \"../../category/package\".",
		"ERROR: ~/category/package/filename.mk:2: Relative path \"../perl5/Makefile\" does not exist.",
		"WARN: ~/category/package/filename.mk:2: \"../perl5\" is not a valid relative package directory.",
		"WARN: ~/category/package/filename.mk:2: Use USE_TOOLS+=perl:run instead of this dependency.",
		"ERROR: ~/category/package/filename.mk:3: Relative path \"../../lang/perl5/Makefile\" does not exist.",
		"WARN: ~/category/package/filename.mk:3: Use USE_TOOLS+=perl:run instead of this dependency.",
		"WARN: ~/category/package/filename.mk:5: Use USE_TOOLS+=msgfmt instead of this dependency.",
		"WARN: ~/category/package/filename.mk:6: Use USE_TOOLS+=gmake instead of this dependency.")

	vt.Values(
		"broken0.12.1:../../x11/alacarte", // missing version
		"broken[0-9]*:../../x11/alacarte", // missing version
		"broken[0-9]*../../x11/alacarte",  // missing colon
		"broken>=:../../x11/alacarte",     // incomplete comparison
		"broken=0:../../x11/alacarte",     // invalid comparison operator
		"broken=:../../x11/alacarte",      // incomplete comparison
		"broken-:../../x11/alacarte",      // incomplete pattern
		"broken>:../../x11/alacarte")      // incomplete comparison

	vt.Output(
		"ERROR: ~/category/package/filename.mk:11: Invalid package pattern \"broken0.12.1\".",
		"ERROR: ~/category/package/filename.mk:12: Invalid package pattern \"broken[0-9]*\".",
		"ERROR: ~/category/package/filename.mk:13: Invalid dependency pattern \"broken[0-9]*../../x11/alacarte\".",
		"ERROR: ~/category/package/filename.mk:14: Invalid package pattern \"broken>=\".",
		"ERROR: ~/category/package/filename.mk:15: Invalid package pattern \"broken=0\".",
		"ERROR: ~/category/package/filename.mk:16: Invalid package pattern \"broken=\".",
		"ERROR: ~/category/package/filename.mk:17: Invalid package pattern \"broken-\".",
		"ERROR: ~/category/package/filename.mk:18: Invalid package pattern \"broken>\".")

	vt.Values(
		"${PYPKGPREFIX}-sqlite3:../../${MY_PKGPATH.py-sqlite3}",
		"${PYPKGPREFIX}-sqlite3:../../databases/py-sqlite3",
		"${DEPENDS.NetBSD}",
		"${DEPENDENCY_PATTERN.py-sqlite3}:${DEPENDENCY_PATH.py-sqlite}",
		"${PYPKGPREFIX}-module>=0:../../devel/py-module",
		"${EMACS_PACKAGE}>=${EMACS_MAJOR}:${EMACS_PKGDIR}",
		"{${NETSCAPE_PREFERRED:C/:/,/g}}-[0-9]*:../../www/${NETSCAPE_PREFERRED:C/:.*//}")

	vt.Output(
		"ERROR: ~/category/package/filename.mk:21: "+
			"Invalid package pattern \"${PYPKGPREFIX}-sqlite3\".",
		"ERROR: ~/category/package/filename.mk:22: "+
			"Invalid package pattern \"${PYPKGPREFIX}-sqlite3\".")

	vt.Values(
		"gettext-[0-9]*:files/../../../databases/py-sqlite3")

	vt.Output(
		"ERROR: ~/category/package/filename.mk:31: "+
			"Relative package directories like "+
			"\"files/../../../databases/py-sqlite3\" must be canonical.",
		"WARN: ~/category/package/filename.mk:31: "+
			"\"files/../../../databases/py-sqlite3\" is "+
			"not a valid relative package directory.")

	// The path has a trailing slash.
	// https://mail-index.netbsd.org/pkgsrc-changes/2020/03/26/msg209490.html
	vt.Values(
		"py-sqlite3-[0-9]*:../../databases/py-sqlite3/",
		"py-sqlite3-[0-9]*:../../././databases/py-sqlite3")

	vt.Output(
		"ERROR: ~/category/package/filename.mk:41: "+
			"Relative package directories like "+
			"\"../../databases/py-sqlite3/\" must not end with a slash.",
		"ERROR: ~/category/package/filename.mk:42: "+
			"Relative package directories like "+
			"\"../../././databases/py-sqlite3\" must be canonical.")

	vt.Values("py-sqlite3>=0:/usr/pkg")

	vt.Output(
		"ERROR: ~/category/package/filename.mk:51: " +
			"Dependency paths like \"/usr/pkg\" must be relative.")

	vt.Values(
		"py-sqlite3>=0:../package/../../category/package")

	// These warnings are quite redundant. It's an edge case anyway.
	vt.Output(
		"WARN: ~/category/package/filename.mk:61: "+
			"Dependency paths should have the form \"../../category/package\".",
		"WARN: ~/category/package/filename.mk:61: "+
			"References to other packages should look like \"../../category/package\", not \"../package\".",
		"ERROR: ~/category/package/filename.mk:61: "+
			"Relative package directories like "+
			"\"../package/../../category/package\" must be canonical.",
		"WARN: ~/category/package/filename.mk:61: "+
			"\"../package/../../category/package\" is not a valid relative package directory.")

	// The "empty" field after the colon is not even counted as a field.
	vt.Values(
		"py-sqlite3>=0:")

	vt.Output(
		"ERROR: ~/category/package/filename.mk:71: " +
			"Invalid dependency pattern \"py-sqlite3>=0:\".")
}

func (s *Suite) Test_VartypeCheck_DistSuffix(c *check.C) {
	vt := NewVartypeCheckTester(s.Init(c), BtDistSuffix)

	vt.Varname("EXTRACT_SUFX")
	vt.Values(
		".tar.gz",
		".tar.bz2",
		".tar.gz # overrides a definition from a Makefile.common")

	vt.Output(
		"NOTE: filename.mk:1: EXTRACT_SUFX is \".tar.gz\" by default, so this definition may be redundant.")
}

func (s *Suite) Test_VartypeCheck_EmulPlatform(c *check.C) {
	vt := NewVartypeCheckTester(s.Init(c), BtEmulPlatform)

	vt.Varname("EMUL_PLATFORM")
	vt.Values(
		"linux-i386",
		"nextbsd-8087",
		"${LINUX}")

	vt.Output(
		"WARN: filename.mk:2: \"nextbsd\" is not valid for the operating system part of EMUL_PLATFORM. "+
			"Use one of "+
			"{ bitrig bsdos cygwin darwin dragonfly freebsd haiku hpux "+
			"interix irix linux mirbsd netbsd openbsd osf1 solaris sunos "+
			"} instead.",
		"WARN: filename.mk:2: \"8087\" is not valid for the hardware architecture part of EMUL_PLATFORM. "+
			"Use one of { "+
			"aarch64 aarch64eb alpha amd64 arc arm arm26 arm32 "+
			"cobalt coldfire convex dreamcast "+
			"earm earmeb earmhf earmhfeb earmv4 earmv4eb earmv5 earmv5eb "+
			"earmv6 earmv6eb earmv6hf earmv6hfeb "+
			"earmv7 earmv7eb earmv7hf earmv7hfeb evbarm "+
			"hpcmips hpcsh hppa hppa64 "+
			"i386 i586 i686 ia64 m68000 m68k m88k "+
			"mips mips64 mips64eb mips64el mipseb mipsel mipsn32 "+
			"mlrisc ns32k pc532 pmax powerpc powerpc64 riscv32 riscv64 rs6000 "+
			"s390 sh3eb sh3el sparc sparc64 vax x86_64 "+
			"} instead.",
		"WARN: filename.mk:3: \"${LINUX}\" is not a valid emulation platform.")
}

func (s *Suite) Test_VartypeCheck_Enum(c *check.C) {
	basicType := enum("jdk1 jdk2 jdk4")
	G.Pkgsrc.Types().Define("JDK", basicType, UserSettable, []ACLEntry{})
	vt := NewVartypeCheckTester(s.Init(c), basicType)

	vt.Varname("JDK")
	vt.Op(opUseMatch)
	vt.Values(
		"*",
		"jdk*",
		"sun-jdk*",
		"${JDKNAME}",
		"[")

	vt.Output(
		"WARN: filename.mk:3: The pattern \"sun-jdk*\" cannot match any of { jdk1 jdk2 jdk4 } for JDK.",
		"WARN: filename.mk:5: Invalid match pattern \"[\".")
}

func (s *Suite) Test_VartypeCheck_Enum__use_match(c *check.C) {
	t := s.Init(c)

	t.SetUpPkgsrc()
	t.Chdir("category/package")
	t.FinishSetUp()
	t.SetUpCommandLine("-Wall", "--explain")

	mklines := t.NewMkLines("module.mk",
		MkCvsID,
		"",
		".include \"../../mk/bsd.prefs.mk\"",
		"",
		".if !empty(MACHINE_ARCH:Mi386) || ${MACHINE_ARCH} == i386",
		".endif",
		".if !empty(PKGSRC_COMPILER:Mclang) || ${PKGSRC_COMPILER} == clang",
		".endif",
		".if ${MACHINE_ARCH:Ni386:Nx86_64:Nsparc64}",
		".endif")

	mklines.Check()

	t.CheckOutputLines(
		"NOTE: module.mk:5: MACHINE_ARCH can be "+
			"compared using the simpler \"${MACHINE_ARCH} == i386\" "+
			"instead of matching against \":Mi386\".",
		"",
		"\tThis variable has a single value, not a list of values. Therefore,",
		"\tit feels strange to apply list operators like :M and :N onto it. A",
		"\tmore direct approach is to use the == and != operators.",
		"",
		"\tAn entirely different case is when the pattern contains wildcards",
		"\tlike *, ?, []. In such a case, using the :M or :N modifiers is",
		"\tuseful and preferred.",
		"",
		"ERROR: module.mk:7: Use ${PKGSRC_COMPILER:Mclang} instead of the == operator.",
		"",
		"\tThe PKGSRC_COMPILER can be a list of chained compilers, e.g. \"ccache",
		"\tdistcc clang\". Therefore, comparing it using == or != leads to wrong",
		"\tresults in these cases.",
		"")
}

func (s *Suite) Test_VartypeCheck_FetchURL(c *check.C) {
	t := s.Init(c)

	t.SetUpPackage("category/own-master-site",
		"MASTER_SITE_OWN=\thttps://example.org/")
	t.FinishSetUp()

	t.SetUpMasterSite("MASTER_SITE_GNU", "http://ftp.gnu.org/pub/gnu/")
	t.SetUpMasterSite("MASTER_SITE_GITHUB", "https://github.com/")

	pkg := NewPackage(t.File("category/own-master-site"))
	pkg.load()

	vt := NewVartypeCheckTester(t, BtFetchURL)
	vt.Package(pkg)

	vt.Varname("MASTER_SITES")
	vt.Values(
		"https://github.com/example/project/",
		"http://ftp.gnu.org/pub/gnu/bison", // Missing a slash at the end
		"${MASTER_SITE_GNU:=bison}",
		"${MASTER_SITE_INVALID:=subdir/}",
		"${MASTER_SITE_OWN}",
		"${MASTER_SITE_OWN:=subdir/}")

	vt.Output(
		"WARN: filename.mk:1: Use ${MASTER_SITE_GITHUB:=example/} "+
			"instead of \"https://github.com/example/\" "+
			"and run \""+confMake+" help topic=github\" for further instructions.",
		"WARN: filename.mk:2: Use ${MASTER_SITE_GNU:=bison} "+
			"instead of \"http://ftp.gnu.org/pub/gnu/bison\".",
		"ERROR: filename.mk:3: The fetch URL \"${MASTER_SITE_GNU:=bison}\" must end with a slash.",
		"ERROR: filename.mk:4: The site MASTER_SITE_INVALID does not exist.")

	// PR 46570, keyword gimp-fix-ca
	vt.Values(
		"https://example.org/download.cgi?filename=filename&sha1=12341234")

	vt.Output(
		"ERROR: filename.mk:11: The fetch URL \"https://example.org/download.cgi" +
			"?filename=filename&sha1=12341234\" must end with a slash.")

	vt.Values(
		"http://example.org/distfiles/",
		"http://example.org/download?filename=distfile;version=1.0",
		"http://example.org/download?filename=<distfile>;version=<version>")

	vt.Output(
		"ERROR: filename.mk:22: The fetch URL \"http://example.org/download"+
			"?filename=distfile;version=1.0\" must end with a slash.",
		"WARN: filename.mk:23: \"http://example.org/download"+
			"?filename=<distfile>;version=<version>\" is not a valid URL.",
		"ERROR: filename.mk:23: The fetch URL \"http://example.org/download"+
			"?filename=<distfile>;version=<version>\" must end with a slash.")

	vt.Values(
		"${MASTER_SITE_GITHUB:S,^,-,:=project/archive/${DISTFILE}}")

	// No warning that the part after the := must end with a slash,
	// since there is another modifier in the expression, in this case :S.
	//
	// That modifier adds a hyphen at the beginning (but pkglint doesn't
	// inspect this), therefore the URL is not required to end with a slash anymore.
	vt.OutputEmpty()

	// As of June 2019, the :S modifier is not analyzed since it is unusual.
	vt.Values(
		"${MASTER_SITE_GNU:S,$,subdir,}",
		"${MASTER_SITE_GNU:S,$,subdir/,}")
	vt.OutputEmpty()

	vt.Values(
		"https://github.com/transmission/transmission-releases/raw/master/")
	vt.Output(
		"WARN: filename.mk:51: Use ${MASTER_SITE_GITHUB:=transmission/} " +
			"instead of \"https://github.com/transmission/\" " +
			"and run \"" + confMake + " help topic=github\" for further instructions.")

	vt.Values(
		"-https://example.org/distfile.tar.gz",
		"-http://ftp.gnu.org/pub/gnu/bash-5.0.tar.gz",
		"-http://ftp.gnu.org/pub/gnu/bash/bash-5.0.tar.gz")

	vt.Output(
		"WARN: filename.mk:62: Use ${MASTER_SITE_GNU:S,^,-,:=bash-5.0.tar.gz} "+
			"instead of \"-http://ftp.gnu.org/pub/gnu/bash-5.0.tar.gz\".",
		"WARN: filename.mk:63: Use ${MASTER_SITE_GNU:S,^,-,:=bash/bash-5.0.tar.gz} "+
			"instead of \"-http://ftp.gnu.org/pub/gnu/bash/bash-5.0.tar.gz\".")

	vt.Values(
		"https://example.org/pub",
		"https://example.org/$@",
		"https://example.org/?f=",
		"https://example.org/download:",
		"https://example.org/download?",
		"https://example.org/$$")

	vt.Output(
		"ERROR: filename.mk:71: The fetch URL \"https://example.org/pub\" must end with a slash.",
		"ERROR: filename.mk:75: The fetch URL \"https://example.org/download?\" must end with a slash.",
		"WARN: filename.mk:76: \"https://example.org/$$\" is not a valid URL.",
		"ERROR: filename.mk:76: The fetch URL \"https://example.org/$$\" must end with a slash.")

	// The transport protocol doesn't matter for matching the MASTER_SITEs.
	// See url2pkg.py, function adjust_site_from_sites_mk.
	vt.Values(
		"http://ftp.gnu.org/pub/gnu/bash/",
		"ftp://ftp.gnu.org/pub/gnu/bash/",
		"https://ftp.gnu.org/pub/gnu/bash/",
		"-http://ftp.gnu.org/pub/gnu/bash/bash-5.0.tar.gz",
		"-ftp://ftp.gnu.org/pub/gnu/bash/bash-5.0.tar.gz",
		"-https://ftp.gnu.org/pub/gnu/bash/bash-5.0.tar.gz")

	vt.Output(
		"WARN: filename.mk:81: Use ${MASTER_SITE_GNU:=bash/} "+
			"instead of \"http://ftp.gnu.org/pub/gnu/bash/\".",
		"WARN: filename.mk:82: Use ${MASTER_SITE_GNU:=bash/} "+
			"instead of \"ftp://ftp.gnu.org/pub/gnu/bash/\".",
		"WARN: filename.mk:83: Use ${MASTER_SITE_GNU:=bash/} "+
			"instead of \"https://ftp.gnu.org/pub/gnu/bash/\".",
		"WARN: filename.mk:84: Use ${MASTER_SITE_GNU:S,^,-,:=bash/bash-5.0.tar.gz} "+
			"instead of \"-http://ftp.gnu.org/pub/gnu/bash/bash-5.0.tar.gz\".",
		"WARN: filename.mk:85: Use ${MASTER_SITE_GNU:S,^,-,:=bash/bash-5.0.tar.gz} "+
			"instead of \"-ftp://ftp.gnu.org/pub/gnu/bash/bash-5.0.tar.gz\".",
		"WARN: filename.mk:86: Use ${MASTER_SITE_GNU:S,^,-,:=bash/bash-5.0.tar.gz} "+
			"instead of \"-https://ftp.gnu.org/pub/gnu/bash/bash-5.0.tar.gz\".")

	// The ${.TARGET} variable doesn't make sense at all in a URL.
	// Other variables might, and there could be checks for them.
	// As of December 2019 these are skipped completely,
	// see containsExpr in VartypeCheck.URL.
	vt.Values(
		"https://example.org/$@")

	vt.OutputEmpty()

	// For secondary distfiles, it does not make sense to refer to GitHub
	// since pulling in the whole github.mk infrastructure is too much
	// effort.
	//
	// Seen in net/unifi on 2021-08-14.
	vt.Varname("SITES.secondary-distfile")
	vt.Values("-https://github.com/org/proj/archive/v1.0.0.tar.gz")

	vt.OutputEmpty()

	vt.Varname("MASTER_SITES")
	vt.Values(
		"${MASTER_SITE_BACKUP}",
		"${MASTER_SITE_BACKUP} # no alternative")

	vt.Output(
		"WARN: filename.mk:111: The site MASTER_SITE_BACKUP should not be used.")
}

func (s *Suite) Test_VartypeCheck_FetchURL__without_package(c *check.C) {
	t := s.Init(c)

	vt := NewVartypeCheckTester(t, BtFetchURL)

	vt.Varname("MASTER_SITES")
	vt.Values(
		"https://github.com/example/project/",
		"${MASTER_SITE_OWN}")

	vt.Output(
		"ERROR: filename.mk:2: The site MASTER_SITE_OWN does not exist.")
}

func (s *Suite) Test_VartypeCheck_Filename(c *check.C) {
	vt := NewVartypeCheckTester(s.Init(c), BtFilename)

	vt.Varname("JAVA_NAME")
	vt.Values(
		"Filename with spaces.docx",
		"OS/2-manual.txt")

	vt.Output(
		"WARN: filename.mk:1: The filename \"Filename with spaces.docx\" contains the invalid characters \"  \".",
		"WARN: filename.mk:2: The filename \"OS/2-manual.txt\" contains the invalid character \"/\".")

	vt.Op(opUseMatch)
	vt.Values(
		"Filename with spaces.docx")

	// There's no guarantee that a filename only contains [A-Za-z0-9.].
	// Therefore, there are no useful checks in this situation.
	vt.Output(
		"WARN: filename.mk:11: The filename pattern \"Filename with spaces.docx\" " +
			"contains the invalid characters \"  \".")
}

func (s *Suite) Test_VartypeCheck_FilePattern(c *check.C) {
	vt := NewVartypeCheckTester(s.Init(c), BtFilePattern)

	vt.Varname("PKGWILDCARD")
	vt.Values(
		"filename.txt",
		"*.txt",
		"[12345].txt",
		"[0-9].txt",
		"???.txt",
		"FilePattern with spaces.docx",
		"OS/2-manual.txt")

	vt.Output(
		"WARN: filename.mk:6: The filename pattern \"FilePattern with spaces.docx\" "+
			"contains the invalid characters \"  \".",
		"WARN: filename.mk:7: The filename pattern \"OS/2-manual.txt\" "+
			"contains the invalid character \"/\".")

	vt.Op(opUseMatch)
	vt.Values(
		"FilePattern with spaces.docx")

	// There's no guarantee that a filename only contains [A-Za-z0-9.].
	// Therefore, it might be necessary to allow all characters here.
	// TODO: Investigate whether this restriction is useful in practice.
	vt.Output(
		"WARN: filename.mk:11: The filename pattern \"FilePattern with spaces.docx\" " +
			"contains the invalid characters \"  \".")
}

func (s *Suite) Test_VartypeCheck_FileMode(c *check.C) {
	vt := NewVartypeCheckTester(s.Init(c), BtFileMode)

	vt.Varname("GAMEMODE")
	vt.Values(
		"u+rwx",
		"0600",
		"1234",
		"12345",
		"${OTHER_PERMS}",
		"")

	vt.Output(
		"WARN: filename.mk:1: Invalid file mode \"u+rwx\".",
		"WARN: filename.mk:4: Invalid file mode \"12345\".",
		"WARN: filename.mk:6: Invalid file mode \"\".")

	vt.Op(opUseMatch)
	vt.Values(
		"u+rwx")

	// There's no guarantee that a filename only contains [A-Za-z0-9.].
	// Therefore, there are no useful checks in this situation.
	vt.Output(
		"WARN: filename.mk:11: Invalid file mode \"u+rwx\".")
}

func (s *Suite) Test_VartypeCheck_GccReqd(c *check.C) {
	vt := NewVartypeCheckTester(s.Init(c), BtGccReqd)

	vt.Varname("GCC_REQD")
	vt.Op(opAssignAppend)
	vt.Values(
		"2.95",
		"3.1.5",
		"4.7",
		"4.8",
		"5.1",
		"6",
		"7.3")
	vt.Output(
		"WARN: filename.mk:5: GCC version numbers should only contain the major version (5).",
		"WARN: filename.mk:7: GCC version numbers should only contain the major version (7).")
}

func (s *Suite) Test_VartypeCheck_GitHubSubmodule(c *check.C) {
	vt := NewVartypeCheckTester(s.Init(c), BtGitHubSubmodule)

	vt.Varname("GITHUB_SUBMODULES")
	vt.Values("user project tag place")
	vt.Values("user project tag")
	vt.Values("user project tag place extra")

	vt.Output(
		"WARN: filename.mk:11: Appending to GITHUB_SUBMODULES "+
			"should happen in groups of 4 words each, not 3.",
		"WARN: filename.mk:21: Appending to GITHUB_SUBMODULES "+
			"should happen in groups of 4 words each, not 5.")
}

func (s *Suite) Test_VartypeCheck_GitHubSubmodule__realistic(c *check.C) {
	t := s.Init(c)
	t.SetUpVartypes()

	mklines := t.NewMkLines("filename.mk",
		MkCvsID,
		"GITHUB_SUBMODULES+=\tuser project tag",
		"GITHUB_SUBMODULES+=\tuser project tag place",
		"GITHUB_SUBMODULES+=\tuser project tag place extra")

	mklines.Check()

	t.CheckOutputLines(
		"WARN: filename.mk:2: Appending to GITHUB_SUBMODULES "+
			"should happen in groups of 4 words each, not 3.",
		"WARN: filename.mk:4: Appending to GITHUB_SUBMODULES "+
			"should happen in groups of 4 words each, not 5.")
}

func (s *Suite) Test_VartypeCheck_GitTag(c *check.C) {
	vt := NewVartypeCheckTester(s.Init(c), BtGitTag)

	vt.Varname("GITHUB_TAG")
	vt.Values(
		"master", // Bad since it is a moving target.
		"v1.2.3",
		"refs/heads/devel", // Bad since it is a moving target.
		"refs/tags/v1.2.3",
		"v${PKGVERSION_NOREV}",
		"1234567812345678123456781234567812345678",
		"1234567",
		"123456", // Too short in practice.
		"${DISTNAME}",
		"invalid:char",  // Bad since ':' is not supported.
		"invalid:;char", // Bad since neither ':' nor ';' is supported.
		"jdk-11.0.10+9-1",
	)
	vt.Output(
		"WARN: filename.mk:1: The Git tag \"master\" refers to a moving target.",
		"WARN: filename.mk:3: The Git tag \"refs/heads/devel\" refers to a moving target.",
		"WARN: filename.mk:8: The git commit name \"123456\" is too short to be reliable.",
		"WARN: filename.mk:10: Invalid character \":\" in Git tag.",
		"WARN: filename.mk:11: Invalid characters \": ;\" in Git tag.")
}

func (s *Suite) Test_VartypeCheck_GoModuleFile(c *check.C) {
	vt := NewVartypeCheckTester(s.Init(c), BtGoModuleFile)

	vt.Varname("GO_MODULE_FILES")
	vt.Values(
		"github.com/!azure/azure-pipeline-go/@v/v0.1.8.mod",
		"github.com/<org>/<proj>",
		"git.sr.ht/~user/gg/@v/v0.3.1.mod",
	)
	vt.Output(
		"WARN: filename.mk:2: Go module \"github.com/<org>/<proj>\" " +
			"contains invalid characters \"< > < >\".",
	)
}

func (s *Suite) Test_VartypeCheck_Homepage(c *check.C) {
	t := s.Init(c)
	vt := NewVartypeCheckTester(t, BtHomepage)

	vt.Varname("HOMEPAGE")
	vt.Values(
		"http://www.pkgsrc.org/",
		"https://www.pkgsrc.org/",
		"${MASTER_SITES}")

	vt.Output(
		"WARN: filename.mk:3: HOMEPAGE should not be defined in terms of MASTER_SITEs.")

	// For more tests, see HomepageChecker.
}

func (s *Suite) Test_VartypeCheck_IdentifierDirect(c *check.C) {
	t := s.Init(c)
	vt := NewVartypeCheckTester(t, BtIdentifierDirect)

	vt.Varname("PKGBASE")
	vt.Values(
		"${OTHER_VAR}",
		"identifiers cannot contain spaces",
		"id/cannot/contain/slashes",
		"id-${OTHER_VAR}",
		"")

	vt.Output(
		"ERROR: filename.mk:1: Identifiers for PKGBASE "+
			"must not refer to other variables.",
		"WARN: filename.mk:2: Invalid identifier \"identifiers cannot contain spaces\".",
		"WARN: filename.mk:3: Invalid identifier \"id/cannot/contain/slashes\".",
		"ERROR: filename.mk:4: Identifiers for PKGBASE "+
			"must not refer to other variables.",
		"WARN: filename.mk:5: Invalid identifier \"\".")

	vt.Op(opUseMatch)
	vt.Values(
		"[A-Z]",
		"[A-Z.]",
		"${PKG_OPTIONS:Moption}",
		"A*B")

	vt.Output(
		"WARN: filename.mk:12: Invalid identifier pattern \"[A-Z.]\" for PKGBASE.")
}

func (s *Suite) Test_VartypeCheck_IdentifierIndirect(c *check.C) {
	t := s.Init(c)
	vt := NewVartypeCheckTester(t, BtIdentifierIndirect)

	vt.Varname("MYSQL_CHARSET")
	vt.Values(
		"${OTHER_VAR}",
		"identifiers cannot contain spaces",
		"id/cannot/contain/slashes",
		"id-${OTHER_VAR}",
		"")

	vt.Output(
		"WARN: filename.mk:2: Invalid identifier \"identifiers cannot contain spaces\".",
		"WARN: filename.mk:3: Invalid identifier \"id/cannot/contain/slashes\".",
		"WARN: filename.mk:5: Invalid identifier \"\".")

	vt.Op(opUseMatch)
	vt.Values(
		"[A-Z]",
		"[A-Z.]",
		"${PKG_OPTIONS:Moption}",
		"A*B")

	vt.Output(
		"WARN: filename.mk:12: Invalid identifier pattern \"[A-Z.]\" for MYSQL_CHARSET.")
}

func (s *Suite) Test_VartypeCheck_Integer(c *check.C) {
	t := s.Init(c)
	vt := NewVartypeCheckTester(t, BtInteger)

	vt.Varname("MAKE_JOBS")
	vt.Values(
		"${OTHER_VAR}",
		"123",
		"-13",
		"11111111111111111111111111111111111111111111111")

	vt.Output(
		"WARN: filename.mk:1: Invalid integer \"${OTHER_VAR}\".",
		"WARN: filename.mk:3: Invalid integer \"-13\".")
}

func (s *Suite) Test_VartypeCheck_LdFlag(c *check.C) {
	vt := NewVartypeCheckTester(s.Init(c), BtLdFlag)

	vt.tester.SetUpTool("pkg-config", "", AtRunTime)

	vt.Varname("LDFLAGS")
	vt.Op(opAssignAppend)
	vt.Values(
		"-lc",
		"-L/usr/lib64",
		"`pkg-config pidgin --ldflags`",
		"-unknown",
		"no-hyphen",
		"-Wl,--rpath,/usr/lib64",
		"-pthread",
		"-static",
		"-static-something",
		"${LDFLAGS.NetBSD}",
		"-l${LIBNCURSES}",
		"`pkg-config`_plus",
		"-DMACRO",
		"-UMACRO",
		"-P",
		"-E",
		"-I${PREFIX}/include")
	vt.Op(opUseMatch)
	vt.Values(
		"anything")

	vt.Output(
		"WARN: filename.mk:6: Use ${COMPILER_RPATH_FLAG} instead of \"-Wl,--rpath,\".",
		"WARN: filename.mk:13: \"-DMACRO\" is a compiler flag "+
			"and belongs on CFLAGS, CPPFLAGS, CXXFLAGS or FFLAGS instead of LDFLAGS.",
		"WARN: filename.mk:14: \"-UMACRO\" is a compiler flag "+
			"and belongs on CFLAGS, CPPFLAGS, CXXFLAGS or FFLAGS instead of LDFLAGS.",
		"WARN: filename.mk:15: \"-P\" is a compiler flag "+
			"and belongs on CFLAGS, CPPFLAGS, CXXFLAGS or FFLAGS instead of LDFLAGS.",
		"WARN: filename.mk:16: \"-E\" is a compiler flag "+
			"and belongs on CFLAGS, CPPFLAGS, CXXFLAGS or FFLAGS instead of LDFLAGS.",
		"WARN: filename.mk:17: \"-I${PREFIX}/include\" is a compiler flag "+
			"and belongs on CFLAGS, CPPFLAGS, CXXFLAGS or FFLAGS instead of LDFLAGS.")
}

func (s *Suite) Test_VartypeCheck_License(c *check.C) {
	t := s.Init(c)

	t.Chdir(".")
	// Adds the gnu-gpl-v2 and 2-clause-bsd licenses
	t.SetUpPackage("category/package",
		".include \"perl5.mk\"")
	t.CreateFileLines("licenses/mit",
		"Permission is hereby granted, ...")
	t.CreateFileLines("category/package/perl5.mk",
		MkCvsID,
		"PERL5_LICENSE= gnu-gpl-v2 OR artistic")
	t.FinishSetUp()
	pkg := NewPackage(t.File("category/package"))
	// This registers the PERL5_LICENSE variable in the package,
	// since Makefile includes perl5.mk.
	pkg.load()
	vt := NewVartypeCheckTester(t, BtLicense)

	vt.Package(pkg)
	vt.Varname("LICENSE")
	vt.Values(
		"gnu-gpl-v2",
		"AND mit",
		"${PERL5_LICENSE}", // Is properly resolved, see perl5.mk above.
		"${UNKNOWN_LICENSE}")

	vt.Output(
		"ERROR: filename.mk:2: Parse error for license condition \"AND mit\".",
		"ERROR: filename.mk:3: License file licenses/artistic does not exist.",
		"ERROR: filename.mk:4: Parse error for license condition \"${UNKNOWN_LICENSE}\".")

	vt.Op(opAssignAppend)
	vt.Values(
		"gnu-gpl-v2",
		"AND mit")

	vt.Output(
		"ERROR: filename.mk:11: Parse error for appended license condition \"gnu-gpl-v2\".")
}

func (s *Suite) Test_VartypeCheck_MachineGnuPlatform(c *check.C) {
	vt := NewVartypeCheckTester(s.Init(c), BtMachineGnuPlatform)

	vt.Varname("MACHINE_GNU_PLATFORM")
	vt.Op(opUseMatch)
	vt.Values(
		"x86_64-pc-cygwin",
		"Cygwin-*-amd64",
		"x86_64-*",
		"*-*-*-*",
		"${OTHER_VAR}",
		"x86_64-pc") // Just for code coverage.

	vt.Output(
		"WARN: filename.mk:2: The pattern \"Cygwin\" cannot match any of "+
			"{ aarch64 aarch64_be alpha amd64 arc arm armeb armv4 armv4eb armv6 armv6eb armv7 armv7eb "+
			"cobalt convex dreamcast hpcmips hpcsh hppa hppa64 i386 i486 ia64 m5407 m68010 m68k m88k "+
			"mips mips64 mips64el mipseb mipsel mipsn32 mlrisc ns32k pc532 pmax powerpc powerpc64 "+
			"rs6000 s390 sh shle sparc sparc64 vax x86_64 "+
			"} for the hardware architecture part of MACHINE_GNU_PLATFORM.",
		"WARN: filename.mk:2: The pattern \"amd64\" cannot match any of "+
			"{ bitrig bsdos cygwin darwin dragonfly freebsd haiku hpux interix irix linux mirbsd "+
			"netbsd openbsd osf1 solaris sunos } "+
			"for the operating system part of MACHINE_GNU_PLATFORM.",
		"WARN: filename.mk:4: \"*-*-*-*\" is not a valid platform pattern.",
		"WARN: filename.mk:6: \"x86_64-pc\" is not a valid platform pattern.")
}

func (s *Suite) Test_VartypeCheck_MachinePlatform(c *check.C) {
	vt := NewVartypeCheckTester(s.Init(c), BtMachinePlatform)

	// There is no need to test the assignment operators since the
	// only variable of this type is read-only.

	vt.Varname("MACHINE_PLATFORM")
	vt.Op(opUseMatch)
	vt.Values(
		"linux-i386",
		"nextbsd-5.0-8087",
		"netbsd-7.0-l*",
		"NetBSD-1.6.2-i386",
		"FreeBSD*",
		"FreeBSD-*",
		"${LINUX}",
		"NetBSD-[0-1]*-*")

	vt.Output(
		"WARN: filename.mk:1: \"linux-i386\" is not a valid platform pattern.",
		"WARN: filename.mk:2: The pattern \"nextbsd\" cannot match any of "+
			"{ Cygwin DragonFly FreeBSD Linux NetBSD SunOS "+
			"} for the operating system part of MACHINE_PLATFORM.",
		"WARN: filename.mk:2: The pattern \"8087\" cannot match any of "+
			"{ aarch64 aarch64eb alpha amd64 arc arm arm26 arm32 "+
			"cobalt coldfire convex dreamcast "+
			"earm earmeb earmhf earmhfeb earmv4 earmv4eb "+
			"earmv5 earmv5eb earmv6 earmv6eb earmv6hf "+
			"earmv6hfeb earmv7 earmv7eb earmv7hf earmv7hfeb evbarm hpcmips hpcsh hppa hppa64 "+
			"i386 i586 i686 ia64 m68000 m68k m88k "+
			"mips mips64 mips64eb mips64el mipseb mipsel mipsn32 "+
			"mlrisc ns32k pc532 pmax powerpc powerpc64 riscv32 riscv64 "+
			"rs6000 s390 sh3eb sh3el sparc sparc64 vax x86_64 "+
			"} for the hardware architecture part of MACHINE_PLATFORM.",
		"WARN: filename.mk:3: The pattern \"netbsd\" cannot match any of "+
			"{ Cygwin DragonFly FreeBSD Linux NetBSD SunOS "+
			"} for the operating system part of MACHINE_PLATFORM.",
		"WARN: filename.mk:3: The pattern \"l*\" cannot match any of "+
			"{ aarch64 aarch64eb alpha amd64 arc arm arm26 arm32 "+
			"cobalt coldfire convex dreamcast "+
			"earm earmeb earmhf earmhfeb earmv4 earmv4eb "+
			"earmv5 earmv5eb earmv6 earmv6eb earmv6hf "+
			"earmv6hfeb earmv7 earmv7eb earmv7hf earmv7hfeb evbarm hpcmips hpcsh hppa hppa64 "+
			"i386 i586 i686 ia64 m68000 m68k m88k "+
			"mips mips64 mips64eb mips64el mipseb mipsel mipsn32 "+
			"mlrisc ns32k pc532 pmax powerpc powerpc64 riscv32 riscv64 "+
			"rs6000 s390 sh3eb sh3el sparc sparc64 vax x86_64 "+
			"} for the hardware architecture part of MACHINE_PLATFORM.",
		"WARN: filename.mk:5: \"FreeBSD*\" is not a valid platform pattern.",
		"WARN: filename.mk:8: Use \"[0-1].*\" instead of \"[0-1]*\" as the version pattern.")
}

func (s *Suite) Test_VartypeCheck_MachinePlatformPattern(c *check.C) {
	vt := NewVartypeCheckTester(s.Init(c), BtMachinePlatformPattern)

	vt.Varname("ONLY_FOR_PLATFORM")
	vt.Op(opUseMatch)
	vt.Values(
		"linux-i386",
		"nextbsd-5.0-8087",
		"netbsd-7.0-l*",
		"NetBSD-1.6.2-i386",
		"FreeBSD*",
		"FreeBSD-*",
		"${LINUX}",
		"NetBSD-[0-1]*-*")

	vt.Output(
		"WARN: filename.mk:1: \"linux-i386\" is not a valid platform pattern.",
		"WARN: filename.mk:2: The pattern \"nextbsd\" cannot match any of "+
			"{ Cygwin DragonFly FreeBSD Linux NetBSD SunOS "+
			"} for the operating system part of ONLY_FOR_PLATFORM.",
		"WARN: filename.mk:2: The pattern \"8087\" cannot match any of "+
			"{ aarch64 aarch64eb alpha amd64 arc arm arm26 arm32 "+
			"cobalt coldfire convex dreamcast "+
			"earm earmeb earmhf earmhfeb earmv4 earmv4eb "+
			"earmv5 earmv5eb earmv6 earmv6eb earmv6hf "+
			"earmv6hfeb earmv7 earmv7eb earmv7hf earmv7hfeb evbarm hpcmips hpcsh hppa hppa64 "+
			"i386 i586 i686 ia64 m68000 m68k m88k "+
			"mips mips64 mips64eb mips64el mipseb mipsel mipsn32 "+
			"mlrisc ns32k pc532 pmax powerpc powerpc64 riscv32 riscv64 "+
			"rs6000 s390 sh3eb sh3el sparc sparc64 vax x86_64 "+
			"} for the hardware architecture part of ONLY_FOR_PLATFORM.",
		"WARN: filename.mk:3: The pattern \"netbsd\" cannot match any of "+
			"{ Cygwin DragonFly FreeBSD Linux NetBSD SunOS "+
			"} for the operating system part of ONLY_FOR_PLATFORM.",
		"WARN: filename.mk:3: The pattern \"l*\" cannot match any of "+
			"{ aarch64 aarch64eb alpha amd64 arc arm arm26 arm32 "+
			"cobalt coldfire convex dreamcast "+
			"earm earmeb earmhf earmhfeb earmv4 earmv4eb "+
			"earmv5 earmv5eb earmv6 earmv6eb earmv6hf "+
			"earmv6hfeb earmv7 earmv7eb earmv7hf earmv7hfeb evbarm hpcmips hpcsh hppa hppa64 "+
			"i386 i586 i686 ia64 m68000 m68k m88k "+
			"mips mips64 mips64eb mips64el mipseb mipsel mipsn32 "+
			"mlrisc ns32k pc532 pmax powerpc powerpc64 riscv32 riscv64 "+
			"rs6000 s390 sh3eb sh3el sparc sparc64 vax x86_64 "+
			"} for the hardware architecture part of ONLY_FOR_PLATFORM.",
		"WARN: filename.mk:5: \"FreeBSD*\" is not a valid platform pattern.",
		"WARN: filename.mk:8: Use \"[0-1].*\" instead of \"[0-1]*\" as the version pattern.")
}

func (s *Suite) Test_VartypeCheck_MailAddress(c *check.C) {
	vt := NewVartypeCheckTester(s.Init(c), BtMailAddress)

	vt.Varname("MAINTAINER")
	vt.Values(
		"pkgsrc-users@netbsd.org",
		"tech-pkg@NetBSD.org",
		"packages@NetBSD.org",
		"user1@example.org,user2@example.org")

	vt.Output(
		"WARN: filename.mk:1: Write \"NetBSD.org\" instead of \"netbsd.org\".",
		"ERROR: filename.mk:2: This mailing list address is obsolete. Use pkgsrc-users@NetBSD.org instead.",
		"ERROR: filename.mk:3: This mailing list address is obsolete. Use pkgsrc-users@NetBSD.org instead.",
		"WARN: filename.mk:4: \"user1@example.org,user2@example.org\" is not a valid mail address.")
}

func (s *Suite) Test_VartypeCheck_MakeTarget(c *check.C) {
	t := s.Init(c)
	vt := NewVartypeCheckTester(t, BtMakeTarget)

	vt.Varname("BUILD_TARGET")
	vt.Values(
		"${OTHER_VAR}",
		"spaces in target lists are ok",
		"target/may/contain/slashes",
		"target:must:not:contain:colons",
		"id-${OTHER_VAR}",
		"")

	vt.Output(
		"WARN: filename.mk:4: Invalid make target " +
			"\"target:must:not:contain:colons\".")

	vt.Op(opUseMatch)
	vt.Values(
		"[A-Z]",
		"[A-Z.]",
		"${PKG_OPTIONS:Moption}",
		"A*B")

	vt.OutputEmpty()
}

func (s *Suite) Test_VartypeCheck_Message(c *check.C) {
	vt := NewVartypeCheckTester(s.Init(c), BtMessage)

	vt.Varname("SUBST_MESSAGE.id")
	vt.Values(
		"\"Correct paths\"",
		"Correct paths")

	vt.Output(
		"WARN: filename.mk:1: SUBST_MESSAGE.id should not be quoted.")
}

func (s *Suite) Test_VartypeCheck_Option(c *check.C) {
	vt := NewVartypeCheckTester(s.Init(c), BtOption)

	G.Pkgsrc.PkgOptions["documented"] = "Option description"
	G.Pkgsrc.PkgOptions["undocumented"] = ""

	vt.Varname("PKG_SUPPORTED_OPTIONS")
	vt.Values(
		"documented",
		"undocumented",
		"unknown",
		"underscore_is_deprecated",
		"UPPER",
		"-invalid")

	vt.Output(
		"WARN: filename.mk:3: Undocumented option \"unknown\".",
		"WARN: filename.mk:4: Use of the underscore character in option names is deprecated.",
		"ERROR: filename.mk:5: Invalid option name \"UPPER\". "+
			"Option names must start with a lowercase letter and be all-lowercase.",
		"ERROR: filename.mk:6: Invalid option name \"-invalid\". "+
			"Option names must start with a lowercase letter and be all-lowercase.")
}

func (s *Suite) Test_VartypeCheck_PackagePattern(c *check.C) {
	t := s.Init(c)

	// See Test_PackagePatternChecker_Check.

	t.CheckOutputEmpty()
}

func (s *Suite) Test_VartypeCheck_Pathlist(c *check.C) {
	vt := NewVartypeCheckTester(s.Init(c), BtPathlist)

	vt.Varname("PATH")
	vt.Values(
		"/usr/bin:/usr/sbin:.::${LOCALBASE}/bin:${HOMEPAGE:S,https://,,}:${TMPDIR}:${PREFIX}/!!!",
		"/directory with spaces")

	vt.Output(
		"ERROR: filename.mk:1: The component \".\" of PATH must be an absolute path.",
		"ERROR: filename.mk:1: The component \"\" of PATH must be an absolute path.",
		"WARN: filename.mk:1: \"${PREFIX}/!!!\" is not a valid pathname.",
		"WARN: filename.mk:2: \"/directory with spaces\" is not a valid pathname.")
}

func (s *Suite) Test_VartypeCheck_PathPattern(c *check.C) {
	vt := NewVartypeCheckTester(s.Init(c), BtPathPattern)

	vt.Varname("DISTDIRS")
	vt.Values(
		"/home/user/*",
		"src/*&*",
		"src/*&&*",
		"src/*/*")

	vt.Output(
		"WARN: filename.mk:2: The pathname pattern \"src/*&*\" "+
			"contains the invalid character \"&\".",
		"WARN: filename.mk:3: The pathname pattern \"src/*&&*\" "+
			"contains the invalid characters \"&&\".")

	vt.Op(opUseMatch)
	vt.Values("any")

	vt.OutputEmpty()
}

func (s *Suite) Test_VartypeCheck_Pathname(c *check.C) {
	vt := NewVartypeCheckTester(s.Init(c), BtPathname)

	vt.Varname("EGDIR")
	vt.Values(
		"${PREFIX}/*",
		"${PREFIX}/share/locale",
		"share/locale",
		"/bin",
		"/path with spaces")
	vt.Output(
		"WARN: filename.mk:1: The pathname \"${PREFIX}/*\" "+
			"contains the invalid character \"*\".",
		"WARN: filename.mk:5: The pathname \"/path with spaces\" "+
			"contains the invalid characters \"  \".")

	vt.Op(opUseMatch)
	vt.Values(
		"anything",
		"/path with *spaces")
	vt.Output(
		"WARN: filename.mk:12: The pathname pattern \"/path with *spaces\" " +
			"contains the invalid characters \"  \".")
}

func (s *Suite) Test_VartypeCheck_PathnameSpace(c *check.C) {
	t := s.Init(c)
	// Invent a variable name since this data type is only used as part
	// of CONF_FILES.
	t.SetUpVarType("CONFIG_FILE", BtPathnameSpace,
		NoVartypeOptions, "*.mk: set, use")
	vt := NewVartypeCheckTester(t, BtPathnameSpace)

	vt.Varname("CONFIG_FILE")
	vt.Values(
		"${PREFIX}/*",
		"${PREFIX}/share/locale",
		"share/locale",
		"/bin",
		"/path with spaces")
	vt.Output(
		"WARN: filename.mk:1: The pathname \"${PREFIX}/*\" " +
			"contains the invalid character \"*\".")

	vt.Op(opUseMatch)
	vt.Values(
		"anything",
		"/path with *spaces&",
		"/path with spaces and ;several, other &characters.",
	)
	vt.Output(
		"WARN: filename.mk:12: The pathname pattern \"/path with *spaces&\" "+
			"contains the invalid character \"&\".",
		"WARN: filename.mk:13: The pathname pattern "+
			"\"/path with spaces and ;several, other &characters.\" "+
			"contains the invalid characters \";&\".")
}

func (s *Suite) Test_VartypeCheck_Perl5Packlist(c *check.C) {
	vt := NewVartypeCheckTester(s.Init(c), BtPerl5Packlist)

	vt.Varname("PERL5_PACKLIST")
	vt.Values(
		"${PKGBASE}",
		"anything else")

	vt.Output(
		"WARN: filename.mk:1: PERL5_PACKLIST should not depend on other variables.")
}

func (s *Suite) Test_VartypeCheck_Perms(c *check.C) {
	vt := NewVartypeCheckTester(s.Init(c), BtPerms)

	vt.Varname("CONF_FILES_PERMS")
	vt.Op(opAssignAppend)
	vt.Values(
		"root",
		"${ROOT_USER}",
		"ROOT_USER",
		"${REAL_ROOT_USER}",
		"${ROOT_GROUP}",
		"${REAL_ROOT_GROUP}")

	vt.Output(
		"ERROR: filename.mk:2: ROOT_USER must not be used in permission definitions. Use REAL_ROOT_USER instead.",
		"ERROR: filename.mk:5: ROOT_GROUP must not be used in permission definitions. Use REAL_ROOT_GROUP instead.")
}

func (s *Suite) Test_VartypeCheck_Pkgname(c *check.C) {
	vt := NewVartypeCheckTester(s.Init(c), BtPkgname)

	vt.Varname("PKGNAME")
	vt.Values(
		"pkgbase-0",
		"pkgbase-1.0",
		"pkgbase-1.1234567890",
		"pkgbase-1z",
		"pkgbase-client-11a",
		"pkgbase-client-1.a",
		"pkgbase-client-1_20180101",
		"pkgbase-z1",
		"pkgbase-3.1.4.1.5.9.2.6.5.3.5.8.9.7.9")

	vt.Output(
		"WARN: filename.mk:8: \"pkgbase-z1\" is not a valid package name.")

	vt.Values(
		"pkgbase-1.0nb17")

	vt.Output(
		"ERROR: filename.mk:11: The \"nb\" part of the version number belongs in PKGREVISION.")

	vt.Op(opUseMatch)
	vt.Values(
		"pkgbase-[0-9]*")

	vt.OutputEmpty()
}

func (s *Suite) Test_VartypeCheck_PkgOptionsVar(c *check.C) {
	vt := NewVartypeCheckTester(s.Init(c), BtPkgOptionsVar)

	vt.Varname("PKG_OPTIONS_VAR")
	vt.Values(
		"PKG_OPTIONS.${PKGBASE}",
		"PKG_OPTIONS.anypkgbase",
		"PKG_OPTS.mc")

	vt.Output(
		"ERROR: filename.mk:1: PKGBASE must not be used in PKG_OPTIONS_VAR.",
		"ERROR: filename.mk:3: PKG_OPTIONS_VAR must be "+
			"of the form \"PKG_OPTIONS.*\", not \"PKG_OPTS.mc\".")
}

func (s *Suite) Test_VartypeCheck_Pkgpath(c *check.C) {
	t := s.Init(c)
	vt := NewVartypeCheckTester(t, BtPkgpath)

	t.CreateFileLines("category/Makefile")
	t.CreateFileLines("category/other-package/Makefile")
	t.CreateFileLines("wip/package/Makefile")
	t.Chdir("category/package")

	vt.Varname("PKGPATH")
	vt.Values(
		"category/other-package",
		"${OTHER_VAR}",
		"invalid",
		"../../invalid/relative",
		"wip/package",
		"category",
		"&&")

	vt.Output(
		"ERROR: filename.mk:3: There is no package in \"../../invalid\".",
		"ERROR: filename.mk:4: There is no package in \"../../../../invalid/relative\".",
		"ERROR: filename.mk:5: A main pkgsrc package must not depend on a pkgsrc-wip package.",
		"ERROR: filename.mk:6: \"category\" is not a valid path to a package.",
		"WARN: filename.mk:7: The pathname \"&&\" contains the invalid characters \"&&\".",
		"ERROR: filename.mk:7: There is no package in \"../../&&\".")

	G.Wip = true

	vt.Values(
		"wip/package")

	vt.OutputEmpty()

	vt.Op(opUseMatch)

	vt.Values(
		"pattern",
		"&special&")

	vt.Output(
		"WARN: filename.mk:22: The pathname pattern \"&special&\" contains the invalid characters \"&&\".")
}

func (s *Suite) Test_VartypeCheck_Pkgrevision(c *check.C) {
	vt := NewVartypeCheckTester(s.Init(c), BtPkgrevision)

	vt.Varname("PKGREVISION")
	vt.Values(
		"3a")

	vt.Output(
		"ERROR: filename.mk:1: PKGREVISION must be a positive integer number.",
		"ERROR: filename.mk:1: PKGREVISION only makes sense directly in the package Makefile.")

	vt.File("Makefile")
	vt.Values(
		"3")

	vt.OutputEmpty()
}

func (s *Suite) Test_VartypeCheck_PlistIdentifier(c *check.C) {
	vt := NewVartypeCheckTester(s.Init(c), BtPlistIdentifier)

	vt.Varname("PLIST_VARS")
	vt.Values(
		"gtk",
		"gtk+",
		"gcc-cxx",
		"gcc-c__",
		"package1.5")

	vt.Output(
		"ERROR: filename.mk:2: "+
			"PLIST identifier \"gtk+\" contains invalid character \"+\".",
		"ERROR: filename.mk:5: "+
			"PLIST identifier \"package1.5\" contains invalid character \".\".")

	vt.Op(opUseMatch)
	vt.Values(
		"*",
		"/",
		"-",
		"[A-Z]",
		"gtk",
		"***+")

	vt.Output(
		"WARN: filename.mk:12: PLIST identifier pattern \"/\" "+
			"contains invalid character \"/\".",
		"WARN: filename.mk:16: PLIST identifier pattern \"***+\" "+
			"contains invalid character \"+\".")
}

func (s *Suite) Test_VartypeCheck_PrefixPathname(c *check.C) {
	vt := NewVartypeCheckTester(s.Init(c), BtPrefixPathname)

	vt.Varname("PKGMANDIR")
	vt.Values(
		"man/man1",
		"share/locale",
		"/absolute")

	vt.Output(
		"WARN: filename.mk:1: "+
			"Use \"${PKGMANDIR}/man1\" instead of \"man/man1\".",
		"ERROR: filename.mk:3: The pathname \"/absolute\" in PKGMANDIR "+
			"must be relative to ${PREFIX}.")

	vt.Varname("INSTALLATION_DIRS")
	vt.Values(
		"bin ${PKG_SYSCONFDIR} ${VARBASE}")

	vt.Output(
		"ERROR: filename.mk:11: PKG_SYSCONFDIR must not be used in INSTALLATION_DIRS "+
			"since it is not relative to PREFIX.",
		"ERROR: filename.mk:11: VARBASE must not be used in INSTALLATION_DIRS "+
			"since it is not relative to PREFIX.")

	// INSTALLATION_DIRS automatically replaces "man" with "${PKGMANDIR}".
	vt.Varname("INSTALLATION_DIRS")
	vt.Values(
		"man/man1")

	vt.OutputEmpty()
}

func (s *Suite) Test_VartypeCheck_PythonDependency(c *check.C) {
	vt := NewVartypeCheckTester(s.Init(c), BtPythonDependency)

	vt.Varname("PYTHON_VERSIONED_DEPENDENCIES")
	vt.Values(
		"cairo",
		"${PYDEP}",
		"cairo,X")

	vt.Output(
		"WARN: filename.mk:2: Python dependencies should not contain variables.",
		"WARN: filename.mk:3: Invalid Python dependency \"cairo,X\".")
}

func (s *Suite) Test_VartypeCheck_RPkgName(c *check.C) {
	vt := NewVartypeCheckTester(s.Init(c), BtRPkgName)

	vt.Varname("R_PKGNAME")
	vt.Values(
		"package",
		"${VAR}",
		"a,b,c",
		"under_score",
		"R-package")

	vt.Output(
		"WARN: filename.mk:2: The R package name should not contain variables.",
		"WARN: filename.mk:3: The R package name contains the invalid characters \",,\".",
		"WARN: filename.mk:5: The R_PKGNAME does not need the \"R-\" prefix.")

	vt.Op(opUseMatch)
	vt.Values(
		"R-package")

	vt.OutputEmpty()
}

func (s *Suite) Test_VartypeCheck_RPkgVer(c *check.C) {
	vt := NewVartypeCheckTester(s.Init(c), BtRPkgVer)

	vt.Varname("R_PKGVER")
	vt.Values(
		"1.0",
		"1-2-3",
		"${VERSION}",
		"1-:")

	vt.Output(
		"WARN: filename.mk:4: Invalid R version number \"1-:\".")

	vt.Op(opUseMatch)
	vt.Values(
		"1-:")

	vt.OutputEmpty()
}

func (s *Suite) Test_VartypeCheck_PackageDir(c *check.C) {
	t := s.Init(c)
	vt := NewVartypeCheckTester(t, BtPackageDir)

	t.CreateFileLines("category/other-package/Makefile")
	t.Chdir("category/package")

	vt.Varname("PKGDIR")
	vt.Values(
		"category/other-package",
		"../../category/other-package",
		"${OTHER_VAR}",
		"invalid",
		"../../invalid/relative",
		"/absolute")

	vt.Output(
		"ERROR: filename.mk:1: Relative path \"category/other-package/Makefile\" does not exist.",
		"WARN: filename.mk:1: \"category/other-package\" is not a valid relative package directory.",
		"ERROR: filename.mk:4: Relative path \"invalid/Makefile\" does not exist.",
		"WARN: filename.mk:4: \"invalid\" is not a valid relative package directory.",
		"ERROR: filename.mk:5: Relative path \"../../invalid/relative/Makefile\" does not exist.",
		"ERROR: filename.mk:6: The path \"/absolute\" must be relative.")
}

func (s *Suite) Test_VartypeCheck_PackagePath(c *check.C) {
	t := s.Init(c)
	vt := NewVartypeCheckTester(t, BtPackagePath)

	t.CreateFileLines("category/other-package/Makefile")
	t.Chdir("category/package")

	vt.Varname("DISTINFO_FILE")
	vt.Values(
		"category/other-package",
		"../../category/other-package",
		"${OTHER_VAR}",
		"invalid",
		"../../invalid/relative",
		"/absolute")

	vt.Output(
		"ERROR: filename.mk:1: Relative path \"category/other-package\" does not exist.",
		"ERROR: filename.mk:4: Relative path \"invalid\" does not exist.",
		"ERROR: filename.mk:5: Relative path \"../../invalid/relative\" does not exist.",
		"ERROR: filename.mk:6: The path \"/absolute\" must be relative.")

	vt.File("../../mk/infra.mk")
	vt.Values(
		"../package",
		"../../category/other-package",
		"../../missing/package",
		"../../category/missing")

	vt.Output(
		"ERROR: ../../mk/infra.mk:1: Relative path \"../package\" does not exist.",
		"ERROR: ../../mk/infra.mk:3: Relative path \"../../missing/package\" does not exist.",
		"ERROR: ../../mk/infra.mk:4: Relative path \"../../category/missing\" does not exist.")
}

func (s *Suite) Test_VartypeCheck_Restricted(c *check.C) {
	vt := NewVartypeCheckTester(s.Init(c), BtRestricted)

	vt.Varname("NO_BIN_ON_CDROM")
	vt.Values(
		"May only be distributed free of charge")

	vt.Output(
		"WARN: filename.mk:1: The only valid value for NO_BIN_ON_CDROM is ${RESTRICTED}.")
}

func (s *Suite) Test_VartypeCheck_SedCommands(c *check.C) {
	vt := NewVartypeCheckTester(s.Init(c), BtSedCommands)

	vt.Varname("SUBST_SED.dummy")
	vt.Values(
		"s,@COMPILER@,gcc,g",
		"-e s,a,b, -e a,b,c,",
		"-e \"s,#,comment ,\"",
		"-e \"s,\\#,comment ,\"",
		"-E",
		"-n",
		"-e 1d",
		"1d",
		"-e",
		"-i s,from,to,",
		"-e s,$${unclosedShellVar", // Just for code coverage.
		"-e s,...")                 // Syntactically invalid sed command.

	vt.Output(
		"NOTE: filename.mk:1: Always use \"-e\" in sed commands, even if there is only one substitution.",
		"WARN: filename.mk:2: Each sed command should appear in an assignment of its own.",
		"WARN: filename.mk:3: The # character starts a makefile comment.",
		"ERROR: filename.mk:3: Invalid shell words \"\\\"s,\" in sed commands.",
		"WARN: filename.mk:8: Unknown sed command \"1d\".",
		"ERROR: filename.mk:9: The -e option to sed requires an argument.",
		"WARN: filename.mk:10: Unknown sed command \"-i\".",
		"NOTE: filename.mk:10: Always use \"-e\" in sed commands, even if there is only one substitution.",
		// XXX: duplicate warning
		"WARN: filename.mk:11: Unclosed shell variable starting at \"$${unclosedShellVar\".",
		"WARN: filename.mk:11: Unclosed shell variable starting at \"$${unclosedShellVar\".")
}

func (s *Suite) Test_VartypeCheck_SedCommands__experimental(c *check.C) {
	vt := NewVartypeCheckTester(s.Init(c), BtSedCommands)

	vt.Varname("SUBST_SED.dummy")

	vt.Values(
		"-e s,???,questions,",
		"-e 's?from?to?g'",
		"-E -e 's,from,to,g'")

	vt.Output(
		"WARN: filename.mk:1: The \"?\" in the word \"s,???,questions,\" may lead to unintended file globbing.")

	vt.Values(
		"-e s,?,replacement,",
		"-e s,\\?,replacement,",
		"-e s,\\\\?,replacement,",
		"-e s,\\\\\\?,replacement,")

	vt.Output(
		"WARN: filename.mk:11: The \"?\" in the word \"s,?,replacement,\" may lead to unintended file globbing.",
		"WARN: filename.mk:13: The \"?\" in the word \"s,\\\\\\\\?,replacement,\" may lead to unintended file globbing.",
		"WARN: filename.mk:13: In a basic regular expression, a backslash followed by \"?\" is undefined.",
		"WARN: filename.mk:14: In a basic regular expression, a backslash followed by \"?\" is undefined.")

	vt.Values(
		"-e s/dir\\\\/file/other-file/")

	// No warning about backslash followed by "/" being undefined.
	vt.OutputEmpty()

	vt.Values(
		"-e 's, ,space,g'",
		"-e 's,\t,tab,g'")

	vt.OutputEmpty()
}

func (s *Suite) Test_VartypeCheck_ShellCommand(c *check.C) {
	t := s.Init(c)
	t.SetUpVartypes()
	vt := NewVartypeCheckTester(t, BtShellCommand)

	vt.Varname("INSTALL_CMD")
	vt.Values(
		"${INSTALL_DATA} -m 0644 ${WRKDIR}/source ${DESTDIR}${PREFIX}/target")

	vt.Op(opUseMatch)
	vt.Values("*")

	vt.OutputEmpty()

	vt.Varname("CC")
	vt.Op(opAssignAppend)
	vt.Values("-ggdb")

	vt.OutputEmpty()
}

func (s *Suite) Test_VartypeCheck_ShellCommands(c *check.C) {
	t := s.Init(c)
	t.SetUpVartypes()
	t.SetUpTool("echo", "ECHO", AtRunTime)
	vt := NewVartypeCheckTester(t, BtShellCommands)

	vt.Varname("GENERATE_PLIST")
	vt.Values(
		"echo bin/program",
		"echo bin/program;")

	vt.Output(
		"WARN: filename.mk:1: This shell command list should end with a semicolon.")
}

func (s *Suite) Test_VartypeCheck_ShellWord(c *check.C) {
	t := s.Init(c)
	t.SetUpVartypes()
	vt := NewVartypeCheckTester(t, BtShellWord)

	vt.Varname("PKG_FAIL_REASON")
	vt.Values(
		"The package does not work here.",
		"\"Properly quoted reason.\"")

	// At this level, there can be no warning for line 1 since each word
	// is analyzed on its own.
	//
	// See Test_MkLineChecker_checkVartype__one_per_line.
	vt.OutputEmpty()
}

func (s *Suite) Test_VartypeCheck_Stage(c *check.C) {
	vt := NewVartypeCheckTester(s.Init(c), BtStage)

	vt.Varname("SUBST_STAGE.dummy")
	vt.Values(
		"post-patch",
		"post-modern",
		"pre-test")

	vt.Output(
		"WARN: filename.mk:2: Invalid stage name \"post-modern\". " +
			"Use one of {pre,do,post}-{extract,patch,configure,build,test,install}.")
}

func (s *Suite) Test_VartypeCheck_ToolDependency(c *check.C) {
	t := s.Init(c)
	vt := NewVartypeCheckTester(t, BtToolDependency)

	t.SetUpTool("tool1", "", AtRunTime)
	t.SetUpTool("tool2", "", AtRunTime)
	t.SetUpTool("tool3", "", AtRunTime)

	vt.Varname("USE_TOOLS")
	vt.Op(opAssignAppend)
	vt.Values(
		"tool3:run",
		"tool2:unknown",
		"${t}",
		"mal:formed:tool",
		"unknown")

	vt.Output(
		"ERROR: filename.mk:2: Invalid tool dependency \"unknown\". "+
			"Use one of \"bootstrap\", \"build\", \"pkgsrc\", \"run\" or \"test\".",
		"ERROR: filename.mk:4: Invalid tool dependency \"mal:formed:tool\".",
		"ERROR: filename.mk:5: Unknown tool \"unknown\".")

	vt.Varname("USE_TOOLS.NetBSD")
	vt.Op(opAssignAppend)
	vt.Values(
		"tool3:run",
		"tool2:unknown")

	vt.Output(
		"ERROR: filename.mk:12: Invalid tool dependency \"unknown\". " +
			"Use one of \"bootstrap\", \"build\", \"pkgsrc\", \"run\" or \"test\".")

	vt.Varname("USE_TOOLS")
	vt.Op(opUseMatch)
	vt.Values(
		"tool1",
		"tool1\\:build",
		"tool1\\:*",
		"${t}\\:build")

	vt.OutputEmpty()

	vt.Op(opAssignAppend)
	vt.Values(
		"tool1:bootstrap",
		"tool1:build",
		"tool1:pkgsrc",
		"tool1:run",
		"tool1:test")

	vt.OutputEmpty()
}

func (s *Suite) Test_VartypeCheck_ToolName(c *check.C) {
	t := s.Init(c)
	vt := NewVartypeCheckTester(t, BtToolName)

	t.SetUpTool("tool1", "", AtRunTime)
	t.SetUpTool("tool2", "", AtRunTime)
	t.SetUpTool("tool3", "", AtRunTime)

	vt.Varname("TOOLS_BROKEN")
	vt.Op(opAssignAppend)
	vt.Values(
		"tool1",
		"tool3:anything",
		"${t}",
		"mal:formed:tool",
		"unknown",
		"c++")

	vt.Output(
		"ERROR: filename.mk:2: TOOLS_BROKEN accepts only plain tool names, "+
			"without any colon.",
		"ERROR: filename.mk:4: TOOLS_BROKEN accepts only plain tool names, "+
			"without any colon.",
		"ERROR: filename.mk:6: Invalid tool name \"c++\".")

	vt.Varname("TOOLS_NOOP")
	vt.Op(opUseMatch)
	vt.Values(
		"tool1",
		"tool1\\:build",
		"${t}\\:build")

	vt.Output(
		"ERROR: filename.mk:12: TOOLS_NOOP accepts only plain tool names, without any colon.",
		"ERROR: filename.mk:13: TOOLS_NOOP accepts only plain tool names, without any colon.")
}

func (s *Suite) Test_VartypeCheck_Unknown(c *check.C) {
	t := s.Init(c)
	vt := NewVartypeCheckTester(t, BtUnknown)

	vt.Varname("BDB185_DEFAULT")
	vt.Values(
		"# empty",
		"Something",
		"'quotes are ok'",
		"!\"#$%&/()*+,-./0-9:;<=>?@A-Z[\\]^_a-z{|}~")

	// This warning is produced as a side effect of parsing the lines.
	// It is not specific to the BtUnknown type.
	vt.Output(
		"WARN: filename.mk:4: The # character starts a makefile comment.")
}

func (s *Suite) Test_VartypeCheck_URL(c *check.C) {
	t := s.Init(c)
	vt := NewVartypeCheckTester(t, BtURL)

	vt.Varname("LATEX2HTML_ICONPATH")
	vt.Values(
		"# none",
		"${OTHER_VAR}",
		"https://www.NetBSD.org/",
		"https://www.netbsd.org/",
		"https://www.example.org",
		"ftp://example.org/pub/",
		"gopher://example.org/")

	vt.Output(
		"WARN: filename.mk:4: Write NetBSD.org instead of www.netbsd.org.",
		"NOTE: filename.mk:5: For consistency, add a trailing slash to \"https://www.example.org\".")

	vt.Values(
		"",
		"ftp://example.org/<",
		"gopher://example.org/<",
		"http://example.org/<",
		"https://example.org/<",
		"https://www.example.org/path with spaces",
		"httpxs://www.example.org",
		"mailto:someone@example.org",
		"string with spaces")

	vt.Output(
		"WARN: filename.mk:11: \"\" is not a valid URL.",
		"WARN: filename.mk:12: \"ftp://example.org/<\" is not a valid URL.",
		"WARN: filename.mk:13: \"gopher://example.org/<\" is not a valid URL.",
		"WARN: filename.mk:14: \"http://example.org/<\" is not a valid URL.",
		"WARN: filename.mk:15: \"https://example.org/<\" is not a valid URL.",
		"WARN: filename.mk:16: \"https://www.example.org/path with spaces\" is not a valid URL.",
		"WARN: filename.mk:17: \"httpxs://www.example.org\" is not a valid URL. Only ftp, gopher, http, and https URLs are allowed here.",
		"WARN: filename.mk:18: \"mailto:someone@example.org\" is not a valid URL.",
		"WARN: filename.mk:19: \"string with spaces\" is not a valid URL.")

	// Yes, even in 2019, some pkgsrc-wip packages really use a gopher HOMEPAGE.
	vt.Values(
		"gopher://bitreich.org/1/scm/geomyidae")
	vt.OutputEmpty()

	G.Logger.Opts.Autofix = true
	vt.Values(
		"# none",
		"${OTHER_VAR}",
		"https://www.NetBSD.org/",
		"https://www.netbsd.org/")
	vt.Output(
		"AUTOFIX: filename.mk:34: " +
			"Replacing \"www.netbsd.org\" with \"www.NetBSD.org\".")
}

func (s *Suite) Test_VartypeCheck_UserGroupName(c *check.C) {
	t := s.Init(c)
	vt := NewVartypeCheckTester(t, BtUserGroupName)

	vt.Varname("APACHE_USER")
	vt.Values(
		"user with spaces",
		"user\twith\ttabs",
		"typical_username",
		"user123",
		"domain\\user",
		"${OTHER_VAR}",
		"r",
		"-rf",
		"rf-")

	vt.Output(
		"WARN: filename.mk:1: User or group name \"user with spaces\" "+
			"contains invalid characters \"space space\".",
		"WARN: filename.mk:2: User or group name \"user\\twith\\ttabs\" "+
			"contains invalid characters \"tab tab\".",
		"WARN: filename.mk:5: User or group name \"domain\\\\user\" "+
			"contains invalid character \"\\\".",
		"ERROR: filename.mk:8: User or group name \"-rf\" "+
			"must not start with a hyphen.",
		"ERROR: filename.mk:9: User or group name \"rf-\" "+
			"must not end with a hyphen.")
}

func (s *Suite) Test_VartypeCheck_VariableName(c *check.C) {
	vt := NewVartypeCheckTester(s.Init(c), BtVariableName)

	vt.Varname("BUILD_DEFS")
	vt.Values(
		"VARBASE",
		"VarBase",
		"PKG_OPTIONS_VAR.pkgbase",
		"${INDIRECT}")

	vt.Output(
		"WARN: filename.mk:2: \"VarBase\" is not a valid variable name.")
}

func (s *Suite) Test_VartypeCheck_VariableNamePattern(c *check.C) {
	vt := NewVartypeCheckTester(s.Init(c), BtVariableNamePattern)

	vt.Varname("_SORTED_VARS.group")
	vt.Values(
		"VARBASE",
		"VarBase",
		"PKG_OPTIONS_VAR.pkgbase",
		"${INDIRECT}",
		"*_DIRS",
		"VAR.*",
		"***")

	vt.Output(
		"WARN: filename.mk:2: \"VarBase\" is not a valid variable name pattern.")
}

func (s *Suite) Test_VartypeCheck_Version(c *check.C) {
	vt := NewVartypeCheckTester(s.Init(c), BtVersion)

	vt.Varname("PERL5_REQD")
	vt.Op(opAssignAppend)
	vt.Values(
		"0",
		"1.2.3.4.5.6",
		"4.1nb17",
		"4.1-SNAPSHOT",
		"4pre7",
		"${VER}")
	vt.Output(
		"WARN: filename.mk:4: Invalid version number \"4.1-SNAPSHOT\".")

	vt.Op(opUseMatch)
	vt.Values(
		"a*",
		"1.2/456",
		"4*",
		"?.??",
		"1.[234]*",
		"1.[2-7].*",
		"[0-9]*")
	vt.Output(
		"WARN: filename.mk:11: Invalid version number pattern \"a*\".",
		"WARN: filename.mk:12: Invalid version number pattern \"1.2/456\".",
		"WARN: filename.mk:13: Use \"4.*\" instead of \"4*\" as the version pattern.",
		"WARN: filename.mk:15: Use \"1.[234].*\" instead of \"1.[234]*\" as the version pattern.")
}

func (s *Suite) Test_VartypeCheck_WrapperReorder(c *check.C) {
	vt := NewVartypeCheckTester(s.Init(c), BtWrapperReorder)

	vt.Varname("WRAPPER_REORDER_CMDS")
	vt.Op(opAssignAppend)
	vt.Values(
		"reorder:l:first:second",
		"reorder:l:first",
		"omit:first")
	vt.Output(
		"WARN: filename.mk:2: Unknown wrapper reorder command \"reorder:l:first\".",
		"WARN: filename.mk:3: Unknown wrapper reorder command \"omit:first\".")
}

func (s *Suite) Test_VartypeCheck_WrapperTransform(c *check.C) {
	vt := NewVartypeCheckTester(s.Init(c), BtWrapperTransform)

	vt.Varname("WRAPPER_TRANSFORM_CMDS")
	vt.Op(opAssignAppend)
	vt.Values(
		"rm:-O3",
		"opt:-option",
		"rename:src:dst",
		"rm-optarg:-option",
		"rmdir:/usr/include",
		"rpath:/usr/lib:/usr/pkg/lib",
		"rpath:/usr/lib",
		"unknown",
		"-e 's,-Wall,-Wall -Wextra,'")
	vt.Output(
		"WARN: filename.mk:7: Unknown wrapper transform command \"rpath:/usr/lib\".",
		"WARN: filename.mk:8: Unknown wrapper transform command \"unknown\".")
}

func (s *Suite) Test_VartypeCheck_WrkdirSubdirectory(c *check.C) {
	t := s.Init(c)
	pkg := NewPackage(t.SetUpPackage("category/package"))
	t.FinishSetUp()
	vt := NewVartypeCheckTester(t, BtWrkdirSubdirectory)
	pkg.Check() // To initialize pkg.redundant.

	vt.Package(pkg)
	vt.Varname("WRKSRC")
	vt.Op(opAssign)
	vt.Values(
		"${WRKDIR}",
		"${WRKDIR}/",
		"${WRKDIR}/.",
		"${WRKDIR}/subdir",
		".",
		"${DISTNAME}",
		"${PKGNAME_NOREV}",
		"two words",
		"../other",
		"${WRKSRC}", // Recursive definition.
		"${PKGDIR}/files")

	// XXX: Many more consistency checks are possible here.
	vt.Output(
		"WARN: filename.mk:8: The pathname \"two words\" " +
			"contains the invalid character \" \".")

	vt.Values(
		// TODO: Note the redundant definition.
		"${WRKDIR}/package-1.0",
		"${WRKDIR}/pkg-1.0",       // different package base
		"${WRKDIR}/package-1.000", // different version string
		"${WRKDIR}/package-1.1",   // different version
	)

	vt.Output(
		"NOTE: filename.mk:21: " +
			"Setting WRKSRC to \"${WRKDIR}/package-1.0\" is redundant.")

	// When the makefile is checked independently of a package, there
	// cannot be any redundancy check.
	vt.Package(nil)

	vt.Values(
		"${WRKDIR}/package-1.0")

	vt.OutputEmpty()
}

// If the package has a non-constant DISTNAME, pkglint cannot reliably
// determine the actual value of WRKSRC, therefore no redundancy note.
func (s *Suite) Test_VartypeCheck_WrkdirSubdirectory__non_constant_DISTNAME(c *check.C) {
	t := s.Init(c)
	pkg := NewPackage(t.SetUpPackage("category/package",
		"DISTNAME=\tpackage-1.0",
		".if 1",
		"DISTNAME=\tpackage-1.1",
		".endif"))
	t.FinishSetUp()
	vt := NewVartypeCheckTester(t, BtWrkdirSubdirectory)
	pkg.Check() // To initialize pkg.redundant.

	vt.Package(pkg)
	vt.Varname("WRKSRC")
	vt.Op(opAssign)

	vt.Values(
		"${WRKDIR}/package-1.0")

	vt.OutputEmpty()
}

func (s *Suite) Test_VartypeCheck_WrksrcPathPattern(c *check.C) {
	t := s.Init(c)
	vt := NewVartypeCheckTester(t, BtWrksrcPathPattern)

	vt.Varname("SUBST_FILES.class")
	vt.Op(opAssign)
	vt.Values(
		"relative/*.sh",
		"${WRKSRC}/relative/*.sh")

	vt.Output(
		"NOTE: filename.mk:2: The pathname patterns in SUBST_FILES.class " +
			"don't need to mention ${WRKSRC}.")

	t.SetUpCommandLine("--autofix")

	vt.Values(
		"relative/*.sh",
		"${WRKSRC}/relative/*.sh")

	vt.Output(
		"AUTOFIX: filename.mk:12: Replacing \"${WRKSRC}/\" with \"\".")

	t.SetUpCommandLine("-Wall")

	// Seen in devel/meson/Makefile.
	vt.Varname("REPLACE_PYTHON")
	vt.Op(opAssign)
	vt.Values(
		"test\\ cases/*/*/*.py",
		"test\" \"cases/*/*/*.py",
		"test' 'cases/*/*/*.py",
		// This matches the single file literally named '*.py'.
		"'test cases/*/*/*.py'")

	vt.OutputEmpty()
}

func (s *Suite) Test_VartypeCheck_WrksrcSubdirectory(c *check.C) {
	vt := NewVartypeCheckTester(s.Init(c), BtWrksrcSubdirectory)

	vt.Varname("BUILD_DIRS")
	vt.Op(opAssignAppend)
	vt.Values(
		"${WRKSRC}",
		"${WRKSRC}/",
		"${WRKSRC}/.",
		"${WRKSRC}/subdir",
		"${CONFIGURE_DIRS}",
		"${WRKSRC}/directory with spaces", // This is a list of 3 directories.
		"directory with spaces",           // This is a list of 3 directories.
		"../other",
		"${WRKDIR}/sub",
		"${SRCDIR}/sub")
	vt.Output(
		"NOTE: filename.mk:1: You can use \".\" instead of \"${WRKSRC}\".",
		"NOTE: filename.mk:2: You can use \".\" instead of \"${WRKSRC}/\".",
		"NOTE: filename.mk:3: You can use \".\" instead of \"${WRKSRC}/.\".",
		"NOTE: filename.mk:4: You can use \"subdir\" instead of \"${WRKSRC}/subdir\".",
		"NOTE: filename.mk:6: You can use \"directory\" instead of \"${WRKSRC}/directory\".",
		"WARN: filename.mk:8: \"../other\" is not a valid subdirectory of ${WRKSRC}.",
		"WARN: filename.mk:9: \"${WRKDIR}/sub\" is not a valid subdirectory of ${WRKSRC}.",
		"WARN: filename.mk:10: \"${SRCDIR}/sub\" is not a valid subdirectory of ${WRKSRC}.")
}

func (s *Suite) Test_VartypeCheck_Yes(c *check.C) {
	vt := NewVartypeCheckTester(s.Init(c), BtYes)

	vt.Varname("APACHE_MODULE")
	vt.Values(
		"yes",
		"YES",
		"no",
		"NO",
		"${YESVAR}")

	vt.Output(
		"WARN: filename.mk:3: APACHE_MODULE should be set to YES or yes.",
		"WARN: filename.mk:4: APACHE_MODULE should be set to YES or yes.",
		"WARN: filename.mk:5: APACHE_MODULE should be set to YES or yes.")

	vt.Varname("BUILD_USES_MSGFMT")
	vt.Op(opUseMatch)
	vt.Values(
		"yes",
		"no",
		"${YESVAR}")

	vt.Output(
		"WARN: filename.mk:11: BUILD_USES_MSGFMT should only be used in a \".if defined(...)\" condition.",
		"WARN: filename.mk:12: BUILD_USES_MSGFMT should only be used in a \".if defined(...)\" condition.",
		"WARN: filename.mk:13: BUILD_USES_MSGFMT should only be used in a \".if defined(...)\" condition.")

	vt.Op(opAssign)
	vt.Values(
		// This was accidentally accepted until 2019-12-09.
		"yes \\# this is not a comment")

	vt.Output(
		"WARN: filename.mk:21: BUILD_USES_MSGFMT should be set to YES or yes.")
}

func (s *Suite) Test_VartypeCheck_YesNo(c *check.C) {
	vt := NewVartypeCheckTester(s.Init(c), BtYesNo)

	vt.Varname("PKG_DEVELOPER")
	vt.Values(
		"yes",
		"YES",
		"no",
		"NO",
		"ja",
		"${YESVAR}",
		"yes # comment",
		"no # comment",
		"Yes indeed")

	vt.Output(
		"WARN: filename.mk:5: PKG_DEVELOPER should be set to YES, yes, NO, or no.",
		"WARN: filename.mk:6: PKG_DEVELOPER should be set to YES, yes, NO, or no.",
		"WARN: filename.mk:9: PKG_DEVELOPER should be set to YES, yes, NO, or no.")

	vt.Op(opUseMatch)
	vt.Values(
		"yes",
		"[Yy]es",
		"[Yy][Ee][Ss]",
		"[yY][eE][sS]",
		"[Nn]o",
		"[Nn][Oo]",
		"[nN][oO]")

	vt.Output(
		"WARN: filename.mk:11: PKG_DEVELOPER should be matched against "+
			"\"[yY][eE][sS]\" or \"[nN][oO]\", not \"yes\".",
		"WARN: filename.mk:12: PKG_DEVELOPER should be matched against "+
			"\"[yY][eE][sS]\" or \"[nN][oO]\", not \"[Yy]es\".",
		"WARN: filename.mk:15: PKG_DEVELOPER should be matched against "+
			"\"[yY][eE][sS]\" or \"[nN][oO]\", not \"[Nn]o\".")

	vt.Op(opAssign)
	vt.Values(
		// This was accidentally accepted until 2019-12-09.
		"yes \\# this is not a comment")

	vt.Output(
		"WARN: filename.mk:21: PKG_DEVELOPER should be set to YES, yes, NO, or no.")
}

func (s *Suite) Test_VartypeCheck_YesNoIndirectly(c *check.C) {
	vt := NewVartypeCheckTester(s.Init(c), BtYesNoIndirectly)

	vt.Varname("IS_BUILTIN.pkgbase")
	vt.Values(
		"yes",
		"no",
		"ja",
		"${YESVAR}")

	vt.Output(
		"WARN: filename.mk:3: IS_BUILTIN.pkgbase should be set to YES, yes, NO, or no.")
}

// VartypeCheckTester helps to test the many different checks in VartypeCheck.
// It keeps track of the current variable, operator, filename, line number,
// so that the test can focus on the interesting details.
type VartypeCheckTester struct {
	tester    *Tester
	basicType *BasicType
	filename  CurrPath
	lineno    int
	varname   string
	op        MkOperator
	pkg       *Package
}

// NewVartypeCheckTester starts the test with a filename of "filename.mk",
// at line 1, with "=" as the operator. The variable has to be initialized
// explicitly.
func NewVartypeCheckTester(t *Tester, basicType *BasicType) *VartypeCheckTester {

	// This is necessary to know whether the variable name is a list type
	// since in such a case each value is split into the list elements.
	if G.Pkgsrc.VariableType(nil, "USE_CWRAPPERS") == nil {
		t.SetUpVartypes()
	}

	return &VartypeCheckTester{t, basicType, "filename.mk", 1, "", opAssign, nil}
}

// Package sets the package that gives context to the MkLines that are
// temporarily created in all following calls to Values.
//
// Depending on the test case at hand, it may be enough to have a bare
// package created by NewPackage, in other cases the package data needs to be
// loaded using Package.load.
func (vt *VartypeCheckTester) Package(pkg *Package) { vt.pkg = pkg }

// Varname sets the variable name that will be used in all following calls to
// Values.
func (vt *VartypeCheckTester) Varname(varname string) {
	vartype := G.Pkgsrc.VariableType(nil, varname)
	assertNotNil(vartype)
	assert(vartype.basicType == vt.basicType)

	vt.varname = varname
	vt.nextSection()
}

// File sets the filename that will be used in all following calls to Values.
// This is useful when testing the permissions of the variable, see
// VarTypeRegistry.
func (vt *VartypeCheckTester) File(filename CurrPath) {
	vt.filename = filename
	vt.lineno = 1
}

// Op sets the operator for the following tests.
// The line number is advanced to the next number ending in 1, e.g. 11, 21, 31.
func (vt *VartypeCheckTester) Op(op MkOperator) {
	vt.op = op
	vt.nextSection()
}

// Values feeds each of the values to the actual check.
// Each value is interpreted as if it were written verbatim into a makefile line.
// That is, # starts a comment.
//
// For the opUseMatch operator, all colons and closing braces must be escaped.
func (vt *VartypeCheckTester) Values(values ...string) {

	toText := func(value string) string {
		op := vt.op
		opStr := op.String()
		varname := vt.varname

		if op == opUseMatch {
			return sprintf(".if ${%s:M%s} == \"\"", varname, value)
		}

		if !contains(opStr, "=") {
			panic("Invalid operator: " + opStr)
		}

		space := condStr(hasSuffix(varname, "+") && opStr == "=", " ", "")
		return varname + space + opStr + value
	}

	test := func(mklines *MkLines, mkline *MkLine, value string) {
		varname := vt.varname
		comment := condStr(mkline.HasComment(), "#", "") + mkline.Comment()
		if mkline.IsVarassign() {
			_ = mkline.Tokenize(value, true) // Produce some warnings as side-effects.
		}

		effectiveValue := value
		if mkline.IsVarassign() {
			effectiveValue = mkline.Value()
		}

		vartype := G.Pkgsrc.VariableType(nil, varname)

		// See MkLineChecker.checkVartype.
		var lineValues []string
		if vartype.IsList() == yes {
			lineValues = mkline.ValueFields(effectiveValue)
		} else {
			lineValues = []string{effectiveValue}
		}

		for _, lineValue := range lineValues {
			valueNovar := mkline.WithoutMakeVariables(lineValue)
			vc := VartypeCheck{mklines, mkline, varname, vt.op, lineValue, valueNovar, comment, false}
			vt.basicType.checker(&vc)
		}
	}

	for _, value := range values {
		text := toText(value)

		line := vt.tester.NewLine(vt.filename, vt.lineno, text)
		mklines := NewMkLines(NewLines(vt.filename, []*Line{line}), vt.pkg, nil)
		mklines.collectRationale()
		vt.lineno++

		mklines.ForEach(func(mkline *MkLine) { test(mklines, mkline, value) })
	}

	vt.nextSection()
}

// Output checks that the output from all previous steps is
// the same as the expectedLines.
func (vt *VartypeCheckTester) Output(expectedLines ...string) {
	vt.tester.CheckOutputLines(expectedLines...)
	vt.nextSection()
}

func (vt *VartypeCheckTester) OutputEmpty() {
	vt.tester.CheckOutputEmpty()
	vt.nextSection()
}

func (vt *VartypeCheckTester) nextSection() {
	if vt.lineno%10 != 1 {
		vt.lineno += 9 - (vt.lineno+8)%10
	}
}
