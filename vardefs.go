package pkglint

import (
	"netbsd.org/pkglint/regex"
	"path"
	"strings"
)

// This file defines the type and the access permissions of most pkgsrc
// variables.
//
// Some types are plain values, some are lists. Lists are split like in the
// shell, using "double" and 'single' quotes to enclose spaces.
//
// See vartypecheck.go for how these types are checked.
//
// The permissions depend on the name of the file where the variable is
// either assigned or used. There are several types of Makefile fragments
// in pkgsrc, and some of them have very specific tasks, like buildlink3.mk,
// builtin.mk and options.mk.
//
// TODO: There are separate permission rules for files from the pkgsrc
//  infrastructure since the infrastructure basically provides the API, and
//  the packages use the API.
//
// Variables that are defined by packages are usually used by the
// infrastructure, and vice versa. There are also user-defined variables,
// which from the view point of a package, are the same as variables
// defined by the infrastructure.

// InitVartypes initializes the long list of predefined pkgsrc variables.
// After this is done, PKGNAME, MAKE_ENV and all the other variables
// can be used in Makefiles without triggering warnings about typos.
func (src *Pkgsrc) InitVartypes() {

	// acl defines a variable with the given type and permissions.
	acl := func(varname string, kindOfList KindOfList, basicType *BasicType, aclEntries ...string) {
		m, varbase, varparam := match2(varname, `^([A-Z_.][A-Z0-9_]*|@)(|\*|\.\*)$`)
		G.Assertf(m, "invalid variable name")

		vartype := Vartype{kindOfList, basicType, parseACLEntries(varname, aclEntries...), false}

		if varparam == "" || varparam == "*" {
			src.vartypes[varbase] = &vartype
		}
		if varparam == "*" || varparam == ".*" {
			src.vartypes[varbase+".*"] = &vartype
		}
	}

	// A package-settable variable may be set in all Makefiles except buildlink3.mk and builtin.mk.
	pkg := func(varname string, checker *BasicType) {
		acl(varname, lkNone, checker,
			"buildlink3.mk, builtin.mk: none",
			"Makefile, Makefile.*, *.mk: default, set, use")
	}

	// pkgload is the same as pkg, except that the variable may be accessed at load time.
	pkgload := func(varname string, kindOfList KindOfList, checker *BasicType) {
		acl(varname, kindOfList, checker,
			"buildlink3.mk, builtin.mk: none",
			"Makefile, Makefile.*, *.mk: default, set, use, use-loadtime")
	}

	// A package-defined list may be defined and appended to in all Makefiles
	// except buildlink3.mk and builtin.mk. Simple assignment (instead of
	// appending) is also allowed. If this leads of an unconditional
	// assignment overriding a previous value, the redundancy check will
	// catch it.
	pkglist := func(varname string, checker *BasicType) {
		acl(varname, lkShell, checker,
			"Makefile, Makefile.*, options.mk: append, default, set, use",
			"buildlink3.mk, builtin.mk: none",
			"*.mk: append, default, use")
	}

	// Some variable types look like lists, but their values cannot be checked
	// by looking at a single one of them. An example is CONF_FILES, which is
	// a list of filename pairs.
	pkglistSpecial := func(varname string, checker *BasicType) {
		acl(varname, lkNone, checker,
			"Makefile, Makefile.*, options.mk: append, default, set, use",
			"buildlink3.mk, builtin.mk: none",
			"*.mk: append, default, use")
	}

	// Some package-defined lists may also be appended in buildlink3.mk files,
	// for example platform-specific CFLAGS and LDFLAGS.
	pkglistbl3 := func(varname string, kindOfList KindOfList, checker *BasicType) {
		acl(varname, kindOfList, checker,
			"Makefile, Makefile.common, options.mk: append, default, set, use",
			"buildlink3.mk, builtin.mk, *.mk: append, default, use")
	}

	// sys declares a user-defined or system-defined variable that must not
	// be modified by packages.
	//
	// It also must not be used in buildlink3.mk and builtin.mk files or at
	// load time since the system/user preferences may not have been loaded
	// when these files are included.
	//
	// TODO: These timing issues should be handled separately from the permissions.
	//  They can be made more precise.
	sys := func(varname string, kindOfList KindOfList, checker *BasicType) {
		acl(varname, kindOfList, checker,
			"buildlink3.mk: none",
			"*: use")
	}

	// usr declares a user-defined variable that must not be modified by packages.
	usr := func(varname string, checker *BasicType) {
		acl(varname, lkNone, checker,
			// TODO: why is builtin.mk missing here?
			"buildlink3.mk: none",
			"*: use-loadtime, use")
	}

	// usr declares a user-defined list variable that must not be modified by packages.
	usrlist := func(varname string, checker *BasicType) {
		acl(varname, lkShell, checker,
			// TODO: why is builtin.mk missing here?
			"buildlink3.mk: none",
			"*: use-loadtime, use")
	}

	// sysload declares a system-provided variable that may already be used at load time.
	sysload := func(varname string, kindOfList KindOfList, checker *BasicType) {
		acl(varname, kindOfList, checker,
			"*: use-loadtime, use")
	}

	// bl3list declares a list variable that is defined by buildlink3.mk and
	// builtin.mk and can later be used by the package.
	bl3list := func(varname string, kindOfList KindOfList, checker *BasicType) {
		acl(varname, kindOfList, checker,
			"buildlink3.mk, builtin.mk: append",
			"*: use")
	}

	// cmdline declares a variable that is defined on the command line. There
	// are only few variables of this type, such as PKG_DEBUG_LEVEL.
	cmdline := func(varname string, kindOfList KindOfList, checker *BasicType) {
		acl(varname, kindOfList, checker,
			"buildlink3.mk, builtin.mk: none",
			"*: use-loadtime, use")
	}

	// compilerLanguages reads the available languages that are typically
	// bundled in a single compiler framework, such as GCC or Clang.
	compilerLanguages := enum(
		func() string {
			mklines := LoadMk(src.File("mk/compiler.mk"), NotEmpty)
			languages := make(map[string]bool)
			if mklines != nil {
				for _, mkline := range mklines.mklines {
					if mkline.IsDirective() && mkline.Directive() == "for" {
						words := mkline.ValueFields(mkline.Args())
						if len(words) > 2 && words[0] == "_version_" {
							for _, word := range words[2:] {
								languages[intern(word)] = true
							}
						}
					}
				}
			}
			alwaysAvailable := [...]string{
				"ada", "c", "c99", "c++", "c++11", "c++14",
				"fortran", "fortran77", "java", "objc", "obj-c++"}
			for _, language := range alwaysAvailable {
				languages[language] = true
			}

			joined := keysJoined(languages)
			if trace.Tracing {
				trace.Stepf("Languages from mk/compiler.mk: %s", joined)
			}
			return joined
		}())

	// enumFrom parses all variable definitions for the given file,
	// and for all variables matching one of the varcanons, all values
	// are added as allowed values.
	//
	// If the file cannot be found, the allowed values are taken from
	// defval. This is mostly useful when testing pkglint.
	enumFrom := func(filename string, defval string, varcanons ...string) *BasicType {
		mklines := LoadMk(src.File(filename), NotEmpty)
		if mklines == nil {
			return enum(defval)
		}

		values := make(map[string]bool)
		for _, mkline := range mklines.mklines {
			if !mkline.IsVarassign() {
				continue
			}

			varcanon := mkline.Varcanon()
			for _, vc := range varcanons {
				if vc != varcanon {
					continue
				}

				words := mkline.ValueFields(mkline.Value())
				for _, word := range words {
					if !contains(word, "$") {
						values[intern(word)] = true
					}
				}
			}
		}

		if len(values) > 0 {
			joined := keysJoined(values)
			if trace.Tracing {
				trace.Stepf("Enum from %s in %s with values: %s",
					strings.Join(varcanons, " "), filename, joined)
			}
			return enum(joined)
		}

		if trace.Tracing {
			trace.Stepf("Enum from default value: %s", defval)
		}
		return enum(defval)
	}

	// enumFromDirs reads the package directories from category, takes all
	// that have a single number in them (such as php72) and ranks them
	// from earliest to latest.
	//
	// If the directories cannot be found, the allowed values are taken
	// from defval. This is mostly useful when testing pkglint.
	enumFromDirs := func(category string, re regex.Pattern, repl string, defval string) *BasicType {
		versions := src.ListVersions(category, re, repl, false)
		if len(versions) == 0 {
			return enum(defval)
		}
		return enum(strings.Join(versions, " "))
	}

	compilers := enumFrom(
		"mk/compiler.mk",
		"ccache ccc clang distcc f2c gcc hp icc ido mipspro mipspro-ucode pcc sunpro xlc",
		"_COMPILERS",
		"_PSEUDO_COMPILERS")

	emacsVersions := enumFrom(
		"editors/emacs/modules.mk",
		"emacs25 emacs21 emacs21nox emacs20 xemacs215 xemacs215nox xemacs214 xemacs214nox",
		"_EMACS_VERSIONS_ALL")

	mysqlVersions := enumFrom(
		"mk/mysql.buildlink3.mk",
		"57 56 55 51 MARIADB55",
		"MYSQL_VERSIONS_ACCEPTED")

	pgsqlVersions := enumFrom(
		"mk/pgsql.buildlink3.mk",
		"10 96 95 94 93",
		"PGSQL_VERSIONS_ACCEPTED")

	jvms := enumFrom(
		"mk/java-vm.mk",
		"openjdk8 oracle-jdk8 openjdk7 sun-jdk7 sun-jdk6 jdk16 jdk15 kaffe",
		"_PKG_JVMS.*")

	// Last synced with mk/defaults/mk.conf revision 1.300 (fe3d998769f).
	usr("USE_CWRAPPERS", enum("yes no auto"))
	usr("ALLOW_VULNERABLE_PACKAGES", BtYes)
	usrlist("AUDIT_PACKAGES_FLAGS", BtShellWord)
	usrlist("MANINSTALL", enum("maninstall catinstall"))
	usr("MANZ", BtYes)
	usrlist("GZIP", BtShellWord)
	usr("MAKE_JOBS", BtInteger)
	usr("OBJHOSTNAME", BtYes)
	usr("OBJMACHINE", BtYes)
	usr("SIGN_PACKAGES", enum("gpg x509"))
	usr("X509_KEY", BtPathname)
	usr("X509_CERTIFICATE", BtPathname)
	usr("PATCH_DEBUG", BtYes)
	usr("PKG_COMPRESSION", enum("gzip bzip2 xz none"))
	usr("PKGSRC_LOCKTYPE", enum("none sleep once"))
	usr("PKGSRC_SLEEPSECS", BtInteger)
	usr("ABI", enum("32 64"))
	usr("PKG_DEVELOPER", BtYesNo)
	usr("USE_ABI_DEPENDS", BtYesNo)
	usr("PKG_REGISTER_SHELLS", enum("YES NO"))
	usrlist("PKGSRC_COMPILER", compilers)
	usr("PKGSRC_KEEP_BIN_PKGS", BtYesNo)
	usrlist("PKGSRC_MESSAGE_RECIPIENTS", BtMailAddress)
	usr("PKGSRC_SHOW_BUILD_DEFS", BtYesNo)
	usr("PKGSRC_RUN_TEST", BtYesNo)
	usr("PKGSRC_MKPIE", BtYesNo)
	usr("PKGSRC_MKREPRO", BtYesNo)
	usr("PKGSRC_USE_CTF", BtYesNo)
	usr("PKGSRC_USE_FORTIFY", enum("no weak strong"))
	usr("PKGSRC_USE_RELRO", enum("no partial full"))
	usr("PKGSRC_USE_SSP", enum("no yes strong all"))
	usr("PKGSRC_USE_STACK_CHECK", enum("no yes"))
	usr("PREFER.*", enum("pkgsrc native"))
	usrlist("PREFER_PKGSRC", BtIdentifier)
	usrlist("PREFER_NATIVE", BtIdentifier)
	usr("PREFER_NATIVE_PTHREADS", BtYesNo)
	usr("WRKOBJDIR", BtPathname)
	usr("LOCALBASE", BtPathname)
	usr("CROSSBASE", BtPathname)
	usr("VARBASE", BtPathname)
	acl("X11_TYPE", lkNone, enum("modular native"),
		"*: use-loadtime, use")
	acl("X11BASE", lkNone, BtPathname,
		"*: use-loadtime, use")
	usr("MOTIFBASE", BtPathname)
	usr("PKGINFODIR", BtPathname)
	usr("PKGMANDIR", BtPathname)
	usr("PKGGNUDIR", BtPathname)
	usr("BSDSRCDIR", BtPathname)
	usr("BSDXSRCDIR", BtPathname)
	usr("DISTDIR", BtPathname)
	usr("DIST_PATH", BtPathlist)
	usr("DEFAULT_VIEW", BtUnknown) // XXX: deprecate? pkgviews has been removed
	usr("FETCH_CMD", BtShellCommand)
	usr("FETCH_USING", enum("auto curl custom fetch ftp manual wget"))
	usrlist("FETCH_BEFORE_ARGS", BtShellWord)
	usrlist("FETCH_AFTER_ARGS", BtShellWord)
	usrlist("FETCH_RESUME_ARGS", BtShellWord)
	usrlist("FETCH_OUTPUT_ARGS", BtShellWord)
	usr("FIX_SYSTEM_HEADERS", BtYes)
	usr("LIBTOOLIZE_PLIST", BtYesNo)
	usr("PKG_RESUME_TRANSFERS", BtYesNo)
	usr("PKG_SYSCONFBASE", BtPathname)
	usr("INIT_SYSTEM", enum("rc.d smf"))
	usr("RCD_SCRIPTS_DIR", BtPathname)
	usr("PACKAGES", BtPathname)
	usr("PASSIVE_FETCH", BtYes)
	usr("PATCH_FUZZ_FACTOR", enum("none -F0 -F1 -F2 -F3"))
	usrlist("ACCEPTABLE_LICENSES", BtIdentifier)
	usr("SPECIFIC_PKGS", BtYes)
	usrlist("SITE_SPECIFIC_PKGS", BtPkgPath)
	usrlist("HOST_SPECIFIC_PKGS", BtPkgPath)
	usrlist("GROUP_SPECIFIC_PKGS", BtPkgPath)
	usrlist("USER_SPECIFIC_PKGS", BtPkgPath)
	usr("EXTRACT_USING", enum("bsdtar gtar nbtar pax"))
	usr("FAILOVER_FETCH", BtYes)
	usrlist("MASTER_SORT", BtUnknown)
	usrlist("MASTER_SORT_REGEX", BtUnknown)
	usr("MASTER_SORT_RANDOM", BtYes)
	usr("PATCH_DEBUG", BtYes)
	usr("PKG_FC", BtShellCommand)
	usrlist("IMAKEOPTS", BtShellWord)
	usr("PRE_ROOT_CMD", BtShellCommand)
	usr("SU_CMD", BtShellCommand)
	usr("SU_CMD_PATH_APPEND", BtPathlist)
	usr("FATAL_OBJECT_FMT_SKEW", BtYesNo)
	usr("WARN_NO_OBJECT_FMT", BtYesNo)
	usr("SMART_MESSAGES", BtYes)
	usrlist("BINPKG_SITES", BtURL)
	usrlist("BIN_INSTALL_FLAGS", BtShellWord)
	usr("LOCALPATCHES", BtPathname)

	// The remaining variables from mk/defaults/mk.conf may be overridden by packages.
	// Therefore they need a separate definition of "user-settable".
	//
	// It is debatable whether packages should be allowed to override these
	// variables at all since then there are two competing sources for the
	// default values. Current practice is to have exactly this ambiguity,
	// combined with some package Makefiles including bsd.prefs.mk and others
	// omitting this necessary inclusion.
	//
	// TODO: parse all the below information directly from mk/defaults/mk.conf.
	usrpkg := func(varname string, checker *BasicType) {
		acl(varname, lkNone, checker,
			"Makefile: default, set, use, use-loadtime",
			"buildlink3.mk, builtin.mk: none",
			"Makefile.*, *.mk: default, set, use, use-loadtime",
			"*: use-loadtime, use")
	}
	usrpkglist := func(varname string, checker *BasicType) {
		acl(varname, lkShell, checker,
			"Makefile: default, set, use, use-loadtime",
			"buildlink3.mk, builtin.mk: none",
			"Makefile.*, *.mk: default, set, use, use-loadtime",
			"*: use-loadtime, use")
	}

	usrpkg("ACROREAD_FONTPATH", BtPathlist)
	usrpkg("AMANDA_USER", BtUserGroupName)
	usrpkg("AMANDA_TMP", BtPathname)
	usrpkg("AMANDA_VAR", BtPathname)
	usrpkg("APACHE_USER", BtUserGroupName)
	usrpkg("APACHE_GROUP", BtUserGroupName)
	usrpkglist("APACHE_SUEXEC_CONFIGURE_ARGS", BtShellWord)
	usrpkglist("APACHE_SUEXEC_DOCROOT", BtPathname)
	usrpkg("ARLA_CACHE", BtPathname)
	usrpkg("BIND_DIR", BtPathname)
	usrpkg("BIND_GROUP", BtUserGroupName)
	usrpkg("BIND_USER", BtUserGroupName)
	usrpkg("CACTI_GROUP", BtUserGroupName)
	usrpkg("CACTI_USER", BtUserGroupName)
	usrpkg("CANNA_GROUP", BtUserGroupName)
	usrpkg("CANNA_USER", BtUserGroupName)
	usrpkg("CDRECORD_CONF", BtPathname)
	usrpkg("CLAMAV_GROUP", BtUserGroupName)
	usrpkg("CLAMAV_USER", BtUserGroupName)
	usrpkg("CLAMAV_DBDIR", BtPathname)
	usrpkg("CONSERVER_DEFAULTHOST", BtIdentifier)
	usrpkg("CONSERVER_DEFAULTPORT", BtInteger)
	usrpkg("CUPS_GROUP", BtUserGroupName)
	usrpkg("CUPS_USER", BtUserGroupName)
	usrpkglist("CUPS_SYSTEM_GROUPS", BtUserGroupName)
	usrpkg("CYRUS_IDLE", enum("poll idled no"))
	usrpkg("CYRUS_GROUP", BtUserGroupName)
	usrpkg("CYRUS_USER", BtUserGroupName)
	usrpkg("DAEMONTOOLS_LOG_USER", BtUserGroupName)
	usrpkg("DAEMONTOOLS_GROUP", BtUserGroupName)
	usrpkg("DBUS_GROUP", BtUserGroupName)
	usrpkg("DBUS_USER", BtUserGroupName)
	usrpkg("DEFANG_GROUP", BtUserGroupName)
	usrpkg("DEFANG_USER", BtUserGroupName)
	usrpkg("DEFANG_SPOOLDIR", BtPathname)
	usrpkg("DEFAULT_IRC_SERVER", BtIdentifier)
	usrpkg("DEFAULT_SERIAL_DEVICE", BtPathname)
	usrpkg("DIALER_GROUP", BtUserGroupName)
	usrpkg("DJBDNS_AXFR_USER", BtUserGroupName)
	usrpkg("DJBDNS_CACHE_USER", BtUserGroupName)
	usrpkg("DJBDNS_LOG_USER", BtUserGroupName)
	usrpkg("DJBDNS_RBL_USER", BtUserGroupName)
	usrpkg("DJBDNS_TINY_USER", BtUserGroupName)
	usrpkg("DJBDNS_DJBDNS_GROUP", BtUserGroupName)
	usrpkg("DT_LAYOUT", enum("US FI FR GER DV"))
	usrpkglist("ELK_GUI", enum("none xaw motif"))
	usrpkg("EMACS_TYPE", emacsVersions)
	usrpkg("EXIM_GROUP", BtUserGroupName)
	usrpkg("EXIM_USER", BtUserGroupName)
	usrpkg("FLUXBOX_USE_XINERAMA", enum("YES NO"))
	usrpkg("FLUXBOX_USE_KDE", enum("YES NO"))
	usrpkg("FLUXBOX_USE_GNOME", enum("YES NO"))
	usrpkg("FLUXBOX_USE_XFT", enum("YES NO"))
	usrpkg("FOX_USE_XUNICODE", enum("YES NO"))
	usrpkg("FREEWNN_USER", BtUserGroupName)
	usrpkg("FREEWNN_GROUP", BtUserGroupName)
	usrpkg("GAMES_USER", BtUserGroupName)
	usrpkg("GAMES_GROUP", BtUserGroupName)
	usrpkg("GAMEMODE", BtFileMode)
	usrpkg("GAMEDIRMODE", BtFileMode)
	usrpkg("GAMEDATAMODE", BtFileMode)
	usrpkg("GAMEGRP", BtUserGroupName)
	usrpkg("GAMEOWN", BtUserGroupName)
	usrpkg("GRUB_NETWORK_CARDS", BtIdentifier)
	usrpkg("GRUB_PRESET_COMMAND", enum("bootp dhcp rarp"))
	usrpkglist("GRUB_SCAN_ARGS", BtShellWord)
	usrpkg("HASKELL_COMPILER", enum("ghc"))
	usrpkg("HOWL_GROUP", BtUserGroupName)
	usrpkg("HOWL_USER", BtUserGroupName)
	usrpkg("ICECAST_CHROOTDIR", BtPathname)
	usrpkg("ICECAST_CHUNKLEN", BtInteger)
	usrpkg("ICECAST_SOURCE_BUFFSIZE", BtInteger)
	usrpkg("IMAP_UW_CCLIENT_MBOX_FMT", enum("mbox mbx mh mmdf mtx mx news phile tenex unix"))
	usrpkg("IMAP_UW_MAILSPOOLHOME", BtFileName)
	usrpkg("IMDICTDIR", BtPathname)
	usrpkg("INN_DATA_DIR", BtPathname)
	usrpkg("INN_USER", BtUserGroupName)
	usrpkg("INN_GROUP", BtUserGroupName)
	usrpkg("IRCD_HYBRID_NICLEN", BtInteger)
	usrpkg("IRCD_HYBRID_TOPICLEN", BtInteger)
	usrpkg("IRCD_HYBRID_SYSLOG_EVENTS", BtUnknown)
	usrpkg("IRCD_HYBRID_SYSLOG_FACILITY", BtIdentifier)
	usrpkg("IRCD_HYBRID_MAXCONN", BtInteger)
	usrpkg("IRCD_HYBRID_IRC_USER", BtUserGroupName)
	usrpkg("IRCD_HYBRID_IRC_GROUP", BtUserGroupName)
	usrpkg("IRRD_USE_PGP", enum("5 2"))
	usrpkg("JABBERD_USER", BtUserGroupName)
	usrpkg("JABBERD_GROUP", BtUserGroupName)
	usrpkg("JABBERD_LOGDIR", BtPathname)
	usrpkg("JABBERD_SPOOLDIR", BtPathname)
	usrpkg("JABBERD_PIDDIR", BtPathname)
	usrpkg("JAKARTA_HOME", BtPathname)
	usrpkg("KERBEROS", BtYes)
	usrpkg("KERMIT_SUID_UUCP", BtYes)
	usrpkg("KJS_USE_PCRE", BtYes)
	usrpkg("KNEWS_DOMAIN_FILE", BtPathname)
	usrpkg("KNEWS_DOMAIN_NAME", BtIdentifier)
	usrpkg("LIBDVDCSS_HOMEPAGE", BtHomepage)
	usrpkglist("LIBDVDCSS_MASTER_SITES", BtFetchURL)
	usrpkg("LIBUSB_TYPE", enum("compat native"))
	usrpkg("LATEX2HTML_ICONPATH", BtURL)
	usrpkg("LEAFNODE_DATA_DIR", BtPathname)
	usrpkg("LEAFNODE_USER", BtUserGroupName)
	usrpkg("LEAFNODE_GROUP", BtUserGroupName)
	usrpkglist("LINUX_LOCALES", BtIdentifier)
	usrpkg("MAILAGENT_DOMAIN", BtIdentifier)
	usrpkg("MAILAGENT_EMAIL", BtMailAddress)
	usrpkg("MAILAGENT_FQDN", BtIdentifier)
	usrpkg("MAILAGENT_ORGANIZATION", BtUnknown)
	usrpkg("MAJORDOMO_HOMEDIR", BtPathname)
	usrpkglist("MAKEINFO_ARGS", BtShellWord)
	usrpkg("MECAB_CHARSET", BtIdentifier)
	usrpkg("MEDIATOMB_GROUP", BtUserGroupName)
	usrpkg("MEDIATOMB_USER", BtUserGroupName)
	usrpkg("MIREDO_USER", BtUserGroupName)
	usrpkg("MIREDO_GROUP", BtUserGroupName)
	usrpkg("MLDONKEY_GROUP", BtUserGroupName)
	usrpkg("MLDONKEY_HOME", BtPathname)
	usrpkg("MLDONKEY_USER", BtUserGroupName)
	usrpkg("MONOTONE_GROUP", BtUserGroupName)
	usrpkg("MONOTONE_USER", BtUserGroupName)
	usrpkg("MOTIF_TYPE", enum("motif openmotif lesstif dt"))
	usrpkg("MOTIF_TYPE_DEFAULT", enum("motif openmotif lesstif dt"))
	usrpkg("MTOOLS_ENABLE_FLOPPYD", BtYesNo)
	usrpkg("MYSQL_USER", BtUserGroupName)
	usrpkg("MYSQL_GROUP", BtUserGroupName)
	usrpkg("MYSQL_DATADIR", BtPathname)
	usrpkg("MYSQL_CHARSET", BtIdentifier)
	usrpkglist("MYSQL_EXTRA_CHARSET", BtIdentifier)
	usrpkg("NAGIOS_GROUP", BtUserGroupName)
	usrpkg("NAGIOS_USER", BtUserGroupName)
	usrpkg("NAGIOSCMD_GROUP", BtUserGroupName)
	usrpkg("NAGIOSDIR", BtPathname)
	usrpkg("NBPAX_PROGRAM_PREFIX", BtUnknown)
	usrpkg("NMH_EDITOR", BtIdentifier)
	usrpkg("NMH_MTA", enum("smtp sendmail"))
	usrpkg("NMH_PAGER", BtIdentifier)
	usrpkg("NS_PREFERRED", enum("communicator navigator mozilla"))
	usrpkg("NULLMAILER_USER", BtUserGroupName)
	usrpkg("NULLMAILER_GROUP", BtUserGroupName)
	usrpkg("OPENSSH_CHROOT", BtPathname)
	usrpkg("OPENSSH_USER", BtUserGroupName)
	usrpkg("OPENSSH_GROUP", BtUserGroupName)
	usrpkg("P4USER", BtUserGroupName)
	usrpkg("P4GROUP", BtUserGroupName)
	usrpkg("P4ROOT", BtPathname)
	usrpkg("P4PORT", BtInteger)
	usrpkg("PALMOS_DEFAULT_SDK", enum("1 2 3.1 3.5"))
	usrpkg("PAPERSIZE", enum("A4 Letter"))
	usrpkg("PGGROUP", BtUserGroupName)
	usrpkg("PGUSER", BtUserGroupName)
	usrpkg("PGHOME", BtPathname)
	usrpkg("PILRC_USE_GTK", BtYesNo)
	usrpkg("PKG_JVM_DEFAULT", jvms)
	usrpkg("POPTOP_USE_MPPE", BtYes)
	usrpkg("PROCMAIL_MAILSPOOLHOME", BtFileName)
	// Comma-separated list of string or integer literals.
	usrpkglist("PROCMAIL_TRUSTED_IDS", BtUnknown)
	usrpkg("PVM_SSH", BtPathname)
	usrpkg("QMAILDIR", BtPathname)
	usrpkg("QMAIL_ALIAS_USER", BtUserGroupName)
	usrpkg("QMAIL_DAEMON_USER", BtUserGroupName)
	usrpkg("QMAIL_LOG_USER", BtUserGroupName)
	usrpkg("QMAIL_ROOT_USER", BtUserGroupName)
	usrpkg("QMAIL_PASSWD_USER", BtUserGroupName)
	usrpkg("QMAIL_QUEUE_USER", BtUserGroupName)
	usrpkg("QMAIL_REMOTE_USER", BtUserGroupName)
	usrpkg("QMAIL_SEND_USER", BtUserGroupName)
	usrpkg("QMAIL_QMAIL_GROUP", BtUserGroupName)
	usrpkg("QMAIL_NOFILES_GROUP", BtUserGroupName)
	usrpkg("QMAIL_QFILTER_TMPDIR", BtPathname)
	usrpkg("QMAIL_QUEUE_DIR", BtPathname)
	usrpkg("QMAIL_QUEUE_EXTRA", BtMailAddress)
	usrpkg("QPOPPER_FAC", BtIdentifier)
	usrpkg("QPOPPER_USER", BtUserGroupName)
	usrpkg("QPOPPER_SPOOL_DIR", BtPathname)
	usrpkg("RASMOL_DEPTH", enum("8 16 32"))
	usrpkg("RELAY_CTRL_DIR", BtPathname)
	usrpkg("RPM_DB_PREFIX", BtPathname)
	usrpkg("RSSH_SCP_PATH", BtPathname)
	usrpkg("RSSH_SFTP_SERVER_PATH", BtPathname)
	usrpkg("RSSH_CVS_PATH", BtPathname)
	usrpkg("RSSH_RDIST_PATH", BtPathname)
	usrpkg("RSSH_RSYNC_PATH", BtPathname)
	usrpkglist("SAWFISH_THEMES", BtFileName)
	usrpkg("SCREWS_GROUP", BtUserGroupName)
	usrpkg("SCREWS_USER", BtUserGroupName)
	usrpkg("SDIST_PAWD", enum("pawd pwd"))
	usrpkglist("SERIAL_DEVICES", BtPathname)
	usrpkg("SILC_CLIENT_WITH_PERL", BtYesNo)
	usrpkg("SNIPROXY_USER", BtUserGroupName)
	usrpkg("SNIPROXY_GROUP", BtUserGroupName)
	usrpkg("SSH_SUID", BtYesNo)
	usrpkg("SSYNC_PAWD", enum("pawd pwd"))
	usrpkg("SUSE_PREFER", enum("13.1 12.1 10.0"))
	usrpkg("TEXMFSITE", BtPathname)
	usrpkg("THTTPD_LOG_FACILITY", BtIdentifier)
	usrpkg("UCSPI_SSL_USER", BtUserGroupName)
	usrpkg("UCSPI_SSL_GROUP", BtUserGroupName)
	usrpkg("UNPRIVILEGED", BtYesNo)
	usrpkg("USE_CROSS_COMPILE", BtYesNo)
	usrpkg("USERPPP_GROUP", BtUserGroupName)
	usrpkg("UUCP_GROUP", BtUserGroupName)
	usrpkg("UUCP_USER", BtUserGroupName)
	usrpkglist("VIM_EXTRA_OPTS", BtShellWord)
	usrpkg("WCALC_HTMLDIR", BtPathname)
	usrpkg("WCALC_HTMLPATH", BtPathname) // URL path
	usrpkg("WCALC_CGIDIR", BtPrefixPathname)
	usrpkg("WCALC_CGIPATH", BtPathname) // URL path
	usrpkglist("WDM_MANAGERS", BtIdentifier)
	usrpkg("X10_PORT", BtPathname)
	usrpkg("XAW_TYPE", enum("standard 3d xpm neXtaw"))
	usrpkg("XLOCK_DEFAULT_MODE", BtIdentifier)
	usrpkg("ZSH_STATIC", BtYes)

	// some other variables, sorted alphabetically

	acl(".CURDIR", lkNone, BtPathname,
		"buildlink3.mk: none",
		"*: use, use-loadtime")
	acl(".IMPSRC", lkShell, BtPathname,
		"buildlink3.mk: none",
		"*: use, use-loadtime")
	acl(".TARGET", lkNone, BtPathname,
		"buildlink3.mk: none",
		"*: use, use-loadtime")
	acl("@", lkNone, BtPathname,
		"buildlink3.mk: none",
		"*: use, use-loadtime")
	pkglist("ALL_ENV", BtShellWord)
	pkg("ALTERNATIVES_FILE", BtFileName)
	pkglist("ALTERNATIVES_SRC", BtPathname)
	pkg("APACHE_MODULE", BtYes)
	sys("AR", lkNone, BtShellCommand)
	sys("AS", lkNone, BtShellCommand)
	pkglist("AUTOCONF_REQD", BtVersion)
	pkglist("AUTOMAKE_OVERRIDE", BtPathmask)
	pkglist("AUTOMAKE_REQD", BtVersion)
	pkg("AUTO_MKDIRS", BtYesNo)
	usr("BATCH", BtYes)
	usr("BDB185_DEFAULT", BtUnknown)
	sys("BDBBASE", lkNone, BtPathname)
	pkglist("BDB_ACCEPTED", enum("db1 db2 db3 db4 db5 db6"))
	usr("BDB_DEFAULT", enum("db1 db2 db3 db4 db5 db6"))
	sys("BDB_LIBS", lkShell, BtLdFlag)
	sys("BDB_TYPE", lkNone, enum("db1 db2 db3 db4 db5 db6"))
	sys("BIGENDIANPLATFORMS", lkShell, BtMachinePlatformPattern)
	sys("BINGRP", lkNone, BtUserGroupName)
	sys("BINMODE", lkNone, BtFileMode)
	sys("BINOWN", lkNone, BtUserGroupName)
	acl("BOOTSTRAP_DEPENDS", lkShell, BtDependencyWithPath,
		"Makefile, Makefile.common, *.mk: append")
	pkg("BOOTSTRAP_PKG", BtYesNo)
	pkg("BROKEN", BtMessage)
	pkg("BROKEN_GETTEXT_DETECTION", BtYesNo)
	pkglist("BROKEN_EXCEPT_ON_PLATFORM", BtMachinePlatformPattern)
	pkglist("BROKEN_ON_PLATFORM", BtMachinePlatformPattern)
	sys("BSD_MAKE_ENV", lkShell, BtShellWord)
	// TODO: Align the permissions of the various BUILDLINK_*.* variables with each other.
	acl("BUILDLINK_ABI_DEPENDS.*", lkShell, BtDependency,
		"buildlink3.mk, builtin.mk: append, use-loadtime",
		"*: append")
	acl("BUILDLINK_API_DEPENDS.*", lkShell, BtDependency,
		"buildlink3.mk, builtin.mk: append, use-loadtime",
		"*: append")
	acl("BUILDLINK_AUTO_DIRS.*", lkNone, BtYesNo,
		"buildlink3.mk: append",
		"Makefile: set")
	sys("BUILDLINK_CFLAGS", lkShell, BtCFlag)
	bl3list("BUILDLINK_CFLAGS.*", lkShell, BtCFlag)
	acl("BUILDLINK_CONTENTS_FILTER.*", lkNone, BtShellCommand,
		"buildlink3.mk: set")
	sys("BUILDLINK_CPPFLAGS", lkShell, BtCFlag)
	bl3list("BUILDLINK_CPPFLAGS.*", lkShell, BtCFlag)
	acl("BUILDLINK_DEPENDS", lkShell, BtIdentifier,
		"buildlink3.mk: append")
	acl("BUILDLINK_DEPMETHOD.*", lkShell, BtBuildlinkDepmethod,
		"buildlink3.mk: default, append, use",
		"Makefile: set, append",
		"Makefile.common, *.mk: append")
	acl("BUILDLINK_DIR", lkNone, BtPathname,
		"*: use")
	bl3list("BUILDLINK_FILES.*", lkShell, BtPathmask)
	pkg("BUILDLINK_FILES_CMD.*", BtShellCommand)
	acl("BUILDLINK_INCDIRS.*", lkShell, BtPathname,
		"buildlink3.mk: default, append",
		"Makefile, Makefile.common, *.mk: use")
	acl("BUILDLINK_JAVA_PREFIX.*", lkNone, BtPathname,
		"buildlink3.mk: set, use")
	acl("BUILDLINK_LDADD.*", lkShell, BtLdFlag,
		"builtin.mk: set, default, append, use",
		"buildlink3.mk: append, use",
		"Makefile, Makefile.common, *.mk: use")
	acl("BUILDLINK_LDFLAGS", lkShell, BtLdFlag,
		"*: use")
	bl3list("BUILDLINK_LDFLAGS.*", lkShell, BtLdFlag)
	acl("BUILDLINK_LIBDIRS.*", lkShell, BtPathname,
		"buildlink3.mk, builtin.mk: append",
		"Makefile, Makefile.common, *.mk: use")
	acl("BUILDLINK_LIBS.*", lkShell, BtLdFlag,
		"Makefile: set, append",
		"buildlink3.mk: append")
	acl("BUILDLINK_PASSTHRU_DIRS", lkShell, BtPathname,
		"Makefile, Makefile.common, buildlink3.mk, hacks.mk: append")
	acl("BUILDLINK_PASSTHRU_RPATHDIRS", lkShell, BtPathname,
		"Makefile, Makefile.common, buildlink3.mk, hacks.mk: append")
	acl("BUILDLINK_PKGSRCDIR.*", lkNone, BtRelativePkgDir,
		"buildlink3.mk: default, use-loadtime")
	acl("BUILDLINK_PREFIX.*", lkNone, BtPathname,
		"builtin.mk: set, use",
		"Makefile, Makefile.common, *.mk: use")
	acl("BUILDLINK_RPATHDIRS.*", lkShell, BtPathname,
		"buildlink3.mk: append")
	acl("BUILDLINK_TARGETS", lkShell, BtIdentifier,
		"Makefile, Makefile.*, *.mk: append")
	acl("BUILDLINK_FNAME_TRANSFORM.*", lkNone, BtSedCommands,
		"Makefile, buildlink3.mk, builtin.mk, hacks.mk, options.mk: append")
	acl("BUILDLINK_TRANSFORM", lkShell, BtWrapperTransform,
		"*: append")
	acl("BUILDLINK_TRANSFORM.*", lkShell, BtWrapperTransform,
		"*: append")
	acl("BUILDLINK_TREE", lkShell, BtIdentifier,
		"buildlink3.mk: append")
	acl("BUILDLINK_X11_DIR", lkNone, BtPathname,
		"*: use")
	acl("BUILD_DEFS", lkShell, BtVariableName,
		"Makefile, Makefile.common, *.mk: append")
	pkglist("BUILD_DEFS_EFFECTS", BtVariableName)
	acl("BUILD_DEPENDS", lkShell, BtDependencyWithPath,
		"Makefile, Makefile.common, *.mk: append")
	pkglist("BUILD_DIRS", BtWrksrcSubdirectory)
	pkglist("BUILD_ENV", BtShellWord)
	sys("BUILD_MAKE_CMD", lkNone, BtShellCommand)
	pkglist("BUILD_MAKE_FLAGS", BtShellWord)
	pkglist("BUILD_TARGET", BtIdentifier)
	pkglist("BUILD_TARGET.*", BtIdentifier)
	pkg("BUILD_USES_MSGFMT", BtYes)
	acl("BUILTIN_PKG", lkNone, BtIdentifier,
		"builtin.mk: set, use-loadtime, use")
	acl("BUILTIN_PKG.*", lkNone, BtPkgName,
		"builtin.mk: set, use-loadtime, use")
	acl("BUILTIN_FIND_FILES_VAR", lkShell, BtVariableName,
		"builtin.mk: set")
	acl("BUILTIN_FIND_FILES.*", lkShell, BtPathname,
		"builtin.mk: set")
	acl("BUILTIN_FIND_GREP.*", lkNone, BtUnknown,
		"builtin.mk: set")
	acl("BUILTIN_FIND_HEADERS_VAR", lkShell, BtVariableName,
		"builtin.mk: set")
	acl("BUILTIN_FIND_HEADERS.*", lkShell, BtPathname,
		"builtin.mk: set")
	acl("BUILTIN_FIND_LIBS", lkShell, BtPathname,
		"builtin.mk: set")
	sys("BUILTIN_X11_TYPE", lkNone, BtUnknown)
	sys("BUILTIN_X11_VERSION", lkNone, BtUnknown)
	acl("CATEGORIES", lkShell, BtCategory,
		"Makefile: set, append",
		"Makefile.common: set, default, append")
	sysload("CC_VERSION", lkNone, BtMessage)
	sysload("CC", lkNone, BtShellCommand)
	pkglistbl3("CFLAGS", lkShell, BtCFlag)   // may also be changed by the user
	pkglistbl3("CFLAGS.*", lkShell, BtCFlag) // may also be changed by the user
	acl("CHECK_BUILTIN", lkNone, BtYesNo,
		"builtin.mk: default",
		"Makefile: set")
	acl("CHECK_BUILTIN.*", lkNone, BtYesNo,
		"Makefile, options.mk, buildlink3.mk: set",
		"builtin.mk: default, use-loadtime",
		"*: use-loadtime")
	acl("CHECK_FILES_SKIP", lkShell, BtBasicRegularExpression,
		"Makefile, Makefile.common: append")
	pkg("CHECK_FILES_SUPPORTED", BtYesNo)
	usr("CHECK_HEADERS", BtYesNo)
	pkglist("CHECK_HEADERS_SKIP", BtPathmask)
	usr("CHECK_INTERPRETER", BtYesNo)
	pkglist("CHECK_INTERPRETER_SKIP", BtPathmask)
	usr("CHECK_PERMS", BtYesNo)
	pkglist("CHECK_PERMS_SKIP", BtPathmask)
	usr("CHECK_PORTABILITY", BtYesNo)
	pkglist("CHECK_PORTABILITY_SKIP", BtPathmask)
	usr("CHECK_RELRO", BtYesNo)
	pkglist("CHECK_RELRO_SKIP", BtPathmask)
	pkg("CHECK_RELRO_SUPPORTED", BtYesNo)
	acl("CHECK_SHLIBS", lkNone, BtYesNo,
		"Makefile: set")
	pkglist("CHECK_SHLIBS_SKIP", BtPathmask)
	acl("CHECK_SHLIBS_SUPPORTED", lkNone, BtYesNo,
		"Makefile: set")
	pkglist("CHECK_WRKREF_SKIP", BtPathmask)
	pkg("CMAKE_ARG_PATH", BtPathname)
	pkglist("CMAKE_ARGS", BtShellWord)
	pkglist("CMAKE_ARGS.*", BtShellWord)
	pkglist("CMAKE_DEPENDENCIES_REWRITE", BtPathmask) // Relative to WRKSRC
	pkglist("CMAKE_MODULE_PATH_OVERRIDE", BtPathmask) // Relative to WRKSRC
	pkg("CMAKE_PKGSRC_BUILD_FLAGS", BtYesNo)
	pkglist("CMAKE_PREFIX_PATH", BtPathmask)
	pkg("CMAKE_USE_GNU_INSTALL_DIRS", BtYesNo)
	pkg("CMAKE_INSTALL_PREFIX", BtPathname) // The default is ${PREFIX}.
	acl("COMMENT", lkNone, BtComment,
		"Makefile, Makefile.common, *.mk: set, append")
	sys("COMPILE.*", lkNone, BtShellCommand)
	acl("COMPILER_RPATH_FLAG", lkNone, enum("-Wl,-rpath"),
		"*: use")
	pkglist("CONFIGURE_ARGS", BtShellWord)
	pkglist("CONFIGURE_ARGS.*", BtShellWord)
	pkglist("CONFIGURE_DIRS", BtWrksrcSubdirectory)
	acl("CONFIGURE_ENV", lkShell, BtShellWord,
		"Makefile, Makefile.common: append, set, use",
		"buildlink3.mk, builtin.mk: append",
		"*.mk: append, use")
	acl("CONFIGURE_ENV.*", lkShell, BtShellWord,
		"Makefile, Makefile.common: append, set, use",
		"buildlink3.mk, builtin.mk: append",
		"*.mk: append, use")
	pkg("CONFIGURE_HAS_INFODIR", BtYesNo)
	pkg("CONFIGURE_HAS_LIBDIR", BtYesNo)
	pkg("CONFIGURE_HAS_MANDIR", BtYesNo)
	pkg("CONFIGURE_SCRIPT", BtPathname)
	acl("CONFIG_GUESS_OVERRIDE", lkShell, BtPathmask,
		"Makefile, Makefile.common: set, append")
	acl("CONFIG_STATUS_OVERRIDE", lkShell, BtPathmask,
		"Makefile, Makefile.common: set, append")
	acl("CONFIG_SHELL", lkNone, BtPathname,
		"Makefile, Makefile.common, hacks.mk: set")
	acl("CONFIG_SUB_OVERRIDE", lkShell, BtPathmask,
		"Makefile, Makefile.common: set, append")
	pkglist("CONFLICTS", BtDependency)
	pkglistSpecial("CONF_FILES", BtConfFiles)
	pkg("CONF_FILES_MODE", enum("0644 0640 0600 0400"))
	pkglist("CONF_FILES_PERMS", BtPerms)
	sys("COPY", lkNone, enum("-c")) // The flag that tells ${INSTALL} to copy a file
	sys("CPP", lkNone, BtShellCommand)
	pkglist("CPPFLAGS", BtCFlag)
	pkglist("CPPFLAGS.*", BtCFlag)
	sys("CXX", lkNone, BtShellCommand)
	pkglist("CXXFLAGS", BtCFlag)
	pkglist("CXXFLAGS.*", BtCFlag)
	pkglist("CWRAPPERS_APPEND.*", BtShellWord)
	sys("DEFAULT_DISTFILES", lkShell, BtFetchURL) // From mk/fetch/bsd.fetch-vars.mk.
	acl("DEINSTALL_FILE", lkNone, BtPathname,
		"Makefile: set")
	acl("DEINSTALL_SRC", lkShell, BtPathname,
		"Makefile: set",
		"Makefile.common: default, set")
	acl("DEINSTALL_TEMPLATES", lkShell, BtPathname,
		"Makefile: set, append",
		"Makefile.common: set, default, append")
	sys("DELAYED_ERROR_MSG", lkNone, BtShellCommand)
	sys("DELAYED_WARNING_MSG", lkNone, BtShellCommand)
	pkglist("DEPENDS", BtDependencyWithPath)
	usrlist("DEPENDS_TARGET", BtIdentifier)
	pkglist("DESCR_SRC", BtPathname)
	sys("DESTDIR", lkNone, BtPathname)
	acl("DESTDIR_VARNAME", lkNone, BtVariableName,
		"Makefile, Makefile.common: set")
	sys("DEVOSSAUDIO", lkNone, BtPathname)
	sys("DEVOSSSOUND", lkNone, BtPathname)
	pkglist("DISTFILES", BtFileName)
	pkg("DISTINFO_FILE", BtRelativePkgPath)
	pkg("DISTNAME", BtFileName)
	pkg("DIST_SUBDIR", BtPathname)
	pkglist("DJB_BUILD_ARGS", BtShellWord)
	pkglist("DJB_BUILD_TARGETS", BtIdentifier)
	pkglistSpecial("DJB_CONFIG_CMDS", BtShellCommands)
	pkglist("DJB_CONFIG_DIRS", BtWrksrcSubdirectory)
	pkg("DJB_CONFIG_HOME", BtFileName)
	pkg("DJB_CONFIG_PREFIX", BtPathname)
	pkglist("DJB_INSTALL_TARGETS", BtIdentifier)
	pkg("DJB_MAKE_TARGETS", BtYesNo)
	pkg("DJB_RESTRICTED", BtYesNo)
	pkg("DJB_SLASHPACKAGE", BtYesNo)
	pkg("DLOPEN_REQUIRE_PTHREADS", BtYesNo)
	acl("DL_AUTO_VARS", lkNone, BtYes,
		"Makefile, Makefile.common, options.mk: set")
	acl("DL_LIBS", lkShell, BtLdFlag,
		"*: use")
	sys("DOCOWN", lkNone, BtUserGroupName)
	sys("DOCGRP", lkNone, BtUserGroupName)
	sys("DOCMODE", lkNone, BtFileMode)
	sys("DOWNLOADED_DISTFILE", lkNone, BtPathname)
	sys("DO_NADA", lkNone, BtShellCommand)
	pkg("DYNAMIC_SITES_CMD", BtShellCommand)
	pkg("DYNAMIC_SITES_SCRIPT", BtPathname)
	acl("ECHO", lkNone, BtShellCommand,
		"*: use")
	sys("ECHO_MSG", lkNone, BtShellCommand)
	sys("ECHO_N", lkNone, BtShellCommand)
	pkg("EGDIR", BtPathname) // Not defined anywhere but used in many places like this.
	sys("EMACS_BIN", lkNone, BtPathname)
	sys("EMACS_ETCPREFIX", lkNone, BtPathname)
	sys("EMACS_FLAVOR", lkNone, enum("emacs xemacs"))
	sys("EMACS_INFOPREFIX", lkNone, BtPathname)
	sys("EMACS_LISPPREFIX", lkNone, BtPathname)
	acl("EMACS_MODULES", lkShell, BtIdentifier,
		"Makefile, Makefile.common: set, append")
	sys("EMACS_PKGNAME_PREFIX", lkNone, BtIdentifier) // Or the empty string.
	sys("EMACS_TYPE", lkNone, enum("emacs xemacs"))
	acl("EMACS_VERSIONS_ACCEPTED", lkShell, emacsVersions,
		"Makefile: set")
	sys("EMACS_VERSION_MAJOR", lkNone, BtInteger)
	sys("EMACS_VERSION_MINOR", lkNone, BtInteger)
	acl("EMACS_VERSION_REQD", lkShell, emacsVersions,
		"Makefile: set, append")
	sys("EMULDIR", lkNone, BtPathname)
	sys("EMULSUBDIR", lkNone, BtPathname)
	sys("OPSYS_EMULDIR", lkNone, BtPathname)
	sys("EMULSUBDIRSLASH", lkNone, BtPathname)
	sys("EMUL_ARCH", lkNone, enum("arm i386 m68k none ns32k sparc vax x86_64"))
	sys("EMUL_DISTRO", lkNone, BtIdentifier)
	sys("EMUL_IS_NATIVE", lkNone, BtYes)
	pkglist("EMUL_MODULES.*", BtIdentifier)
	sys("EMUL_OPSYS", lkNone, enum("darwin freebsd hpux irix linux osf1 solaris sunos none"))
	pkg("EMUL_PKG_FMT", enum("plain rpm"))
	usr("EMUL_PLATFORM", BtEmulPlatform)
	pkglist("EMUL_PLATFORMS", BtEmulPlatform)
	usrlist("EMUL_PREFER", BtEmulPlatform)
	pkglist("EMUL_REQD", BtDependency)
	usr("EMUL_TYPE.*", enum("native builtin suse suse-10.0 suse-12.1 suse-13.1"))
	sys("ERROR_CAT", lkNone, BtShellCommand)
	sys("ERROR_MSG", lkNone, BtShellCommand)
	sys("EXPORT_SYMBOLS_LDFLAGS", lkShell, BtLdFlag)
	sys("EXTRACT_CMD", lkNone, BtShellCommand)
	pkg("EXTRACT_DIR", BtPathname)
	pkg("EXTRACT_DIR.*", BtPathname)
	pkglist("EXTRACT_ELEMENTS", BtPathmask)
	pkglist("EXTRACT_ENV", BtShellWord)
	pkglist("EXTRACT_ONLY", BtPathname)
	acl("EXTRACT_OPTS", lkShell, BtShellWord,
		"Makefile, Makefile.common: set, append")
	acl("EXTRACT_OPTS_BIN", lkShell, BtShellWord,
		"Makefile, Makefile.common: set, append")
	acl("EXTRACT_OPTS_LHA", lkShell, BtShellWord,
		"Makefile, Makefile.common: set, append")
	acl("EXTRACT_OPTS_PAX", lkShell, BtShellWord,
		"Makefile, Makefile.common: set, append")
	acl("EXTRACT_OPTS_RAR", lkShell, BtShellWord,
		"Makefile, Makefile.common: set, append")
	acl("EXTRACT_OPTS_TAR", lkShell, BtShellWord,
		"Makefile, Makefile.common: set, append")
	acl("EXTRACT_OPTS_ZIP", lkShell, BtShellWord,
		"Makefile, Makefile.common: set, append")
	acl("EXTRACT_OPTS_ZOO", lkShell, BtShellWord,
		"Makefile, Makefile.common: set, append")
	pkg("EXTRACT_SUFX", BtDistSuffix)
	pkg("EXTRACT_USING", enum("bsdtar gtar nbtar pax"))
	sys("FAIL_MSG", lkNone, BtShellCommand)
	sys("FAMBASE", lkNone, BtPathname)
	pkglist("FAM_ACCEPTED", enum("fam gamin"))
	usr("FAM_DEFAULT", enum("fam gamin"))
	sys("FAM_TYPE", lkNone, enum("fam gamin"))
	acl("FETCH_BEFORE_ARGS", lkShell, BtShellWord,
		"Makefile: set, append")
	pkglist("FETCH_MESSAGE", BtShellWord)
	pkg("FILESDIR", BtRelativePkgPath)
	pkglist("FILES_SUBST", BtShellWord)
	sys("FILES_SUBST_SED", lkShell, BtShellWord)
	pkglist("FIX_RPATH", BtVariableName)
	pkglist("FLEX_REQD", BtVersion)
	acl("FONTS_DIRS.*", lkShell, BtPathname,
		"Makefile: set, append, use",
		"Makefile.common: append, use")
	sys("GAMEDATAMODE", lkNone, BtFileMode)
	sys("GAMES_GROUP", lkNone, BtUserGroupName)
	sys("GAMEDATA_PERMS", lkShell, BtPerms)
	sys("GAMEDIR_PERMS", lkShell, BtPerms)
	sys("GAMEMODE", lkNone, BtFileMode)
	sys("GAMES_USER", lkNone, BtUserGroupName)
	pkglist("GCC_REQD", BtGccReqd)
	pkglistSpecial("GENERATE_PLIST", BtShellCommands)
	pkg("GITHUB_PROJECT", BtIdentifier)
	pkg("GITHUB_TAG", BtIdentifier)
	pkg("GITHUB_RELEASE", BtFileName)
	pkg("GITHUB_TYPE", enum("tag release"))
	pkg("GMAKE_REQD", BtVersion)
	acl("GNU_ARCH.*", lkNone, BtIdentifier,
		"buildlink3.mk: none",
		"*: set, use")
	pkgload("GNU_CONFIGURE", lkNone, BtYes)
	pkg("GNU_CONFIGURE_INFODIR", BtPathname)
	pkg("GNU_CONFIGURE_LIBDIR", BtPathname)
	pkg("GNU_CONFIGURE_LIBSUBDIR", BtPathname)
	acl("GNU_CONFIGURE_MANDIR", lkNone, BtPathname,
		"Makefile, Makefile.common: set")
	pkg("GNU_CONFIGURE_PREFIX", BtPathname)
	pkg("GOPATH", BtPathname)
	pkgload("HAS_CONFIGURE", lkNone, BtYes)
	pkglist("HEADER_TEMPLATES", BtPathname)
	pkg("HOMEPAGE", BtHomepage)
	pkg("ICON_THEMES", BtYes)
	acl("IGNORE_PKG.*", lkNone, BtYes,
		"*: set, use-loadtime")
	sys("IMAKE", lkNone, BtShellCommand)
	acl("INCOMPAT_CURSES", lkShell, BtMachinePlatformPattern,
		"Makefile: set, append")
	sys("INFO_DIR", lkNone, BtPathname) // relative to PREFIX
	pkg("INFO_FILES", BtYes)
	sys("INSTALL", lkNone, BtShellCommand)
	pkglist("INSTALLATION_DIRS", BtPrefixPathname)
	pkg("INSTALLATION_DIRS_FROM_PLIST", BtYes)
	sys("INSTALL_DATA", lkNone, BtShellCommand)
	sys("INSTALL_DATA_DIR", lkNone, BtShellCommand)
	pkglist("INSTALL_DIRS", BtWrksrcSubdirectory)
	pkglist("INSTALL_ENV", BtShellWord)
	acl("INSTALL_FILE", lkNone, BtPathname,
		"Makefile: set")
	sys("INSTALL_GAME", lkNone, BtShellCommand)
	sys("INSTALL_GAME_DATA", lkNone, BtShellCommand)
	sys("INSTALL_LIB", lkNone, BtShellCommand)
	sys("INSTALL_LIB_DIR", lkNone, BtShellCommand)
	pkglist("INSTALL_MAKE_FLAGS", BtShellWord)
	sys("INSTALL_MAN", lkNone, BtShellCommand)
	sys("INSTALL_MAN_DIR", lkNone, BtShellCommand)
	sys("INSTALL_PROGRAM", lkNone, BtShellCommand)
	sys("INSTALL_PROGRAM_DIR", lkNone, BtShellCommand)
	sys("INSTALL_SCRIPT", lkNone, BtShellCommand)
	sys("INSTALL_SCRIPTS_ENV", lkShell, BtShellWord)
	sys("INSTALL_SCRIPT_DIR", lkNone, BtShellCommand)
	acl("INSTALL_SRC", lkShell, BtPathname,
		"Makefile: set",
		"Makefile.common: default, set")
	pkglist("INSTALL_TARGET", BtIdentifier)
	acl("INSTALL_TEMPLATES", lkShell, BtPathname,
		"Makefile: set, append",
		"Makefile.common, *.mk: set, default, append")
	acl("INSTALL_UNSTRIPPED", lkNone, BtYesNo,
		"Makefile, Makefile.common, options.mk: default, set")
	pkglist("INTERACTIVE_STAGE", enum("fetch extract configure build test install"))
	acl("IS_BUILTIN.*", lkNone, BtYesNoIndirectly,
		"builtin.mk: set, use-loadtime, use")
	sys("JAVA_BINPREFIX", lkNone, BtPathname)
	pkg("JAVA_CLASSPATH", BtShellWord)
	pkg("JAVA_HOME", BtPathname)
	pkg("JAVA_NAME", BtFileName)
	pkglist("JAVA_UNLIMIT", enum("cmdsize datasize stacksize"))
	pkglist("JAVA_WRAPPERS", BtFileName)
	pkg("JAVA_WRAPPER_BIN.*", BtPathname)
	sys("KRB5BASE", lkNone, BtPathname)
	pkglist("KRB5_ACCEPTED", enum("heimdal mit-krb5"))
	usr("KRB5_DEFAULT", enum("heimdal mit-krb5"))
	sys("KRB5_TYPE", lkNone, BtIdentifier)
	sys("LD", lkNone, BtShellCommand)
	pkglistbl3("LDFLAGS", lkShell, BtLdFlag)      // May also be changed by the user.
	pkglistbl3("LDFLAGS.*", lkShell, BtLdFlag)    // May also be changed by the user.
	sysload("LIBABISUFFIX", lkNone, BtIdentifier) // Can also be empty.
	sys("LIBGRP", lkNone, BtUserGroupName)
	sys("LIBMODE", lkNone, BtFileMode)
	sys("LIBOWN", lkNone, BtUserGroupName)
	sys("LIBOSSAUDIO", lkNone, BtPathname)
	pkglist("LIBS", BtLdFlag)
	pkglist("LIBS.*", BtLdFlag)
	sys("LIBTOOL", lkNone, BtShellCommand)
	acl("LIBTOOL_OVERRIDE", lkShell, BtPathmask,
		"Makefile: set, append")
	pkglist("LIBTOOL_REQD", BtVersion)
	acl("LICENCE", lkNone, BtLicense,
		"buildlink3.mk, builtin.mk: none",
		"Makefile: set, append",
		"*: default, set, append")
	acl("LICENSE", lkNone, BtLicense,
		"buildlink3.mk, builtin.mk: none",
		"Makefile: set, append",
		"*: default, set, append")
	pkg("LICENSE_FILE", BtPathname)
	sys("LINK.*", lkNone, BtShellCommand)
	sys("LINKER_RPATH_FLAG", lkNone, BtShellWord)
	sys("LITTLEENDIANPLATFORMS", lkShell, BtMachinePlatformPattern)
	sys("LOWER_OPSYS", lkNone, BtIdentifier)
	sys("LOWER_VENDOR", lkNone, BtIdentifier)
	sys("LP64PLATFORMS", lkShell, BtMachinePlatformPattern)
	acl("LTCONFIG_OVERRIDE", lkShell, BtPathmask,
		"Makefile: set, append",
		"Makefile.common: append")
	sysload("MACHINE_ARCH", lkNone, enumMachineArch)
	sysload("MACHINE_GNU_ARCH", lkNone, enumMachineGnuArch)
	sysload("MACHINE_GNU_PLATFORM", lkNone, BtMachineGnuPlatform)
	sysload("MACHINE_PLATFORM", lkNone, BtMachinePlatform)
	acl("MAINTAINER", lkNone, BtMailAddress,
		"Makefile: set",
		"Makefile.common: default")
	sysload("MAKE", lkNone, BtShellCommand)
	pkglist("MAKEFLAGS", BtShellWord)
	acl("MAKEVARS", lkShell, BtVariableName,
		"Makefile, buildlink3.mk, builtin.mk, hacks.mk: append")
	pkglist("MAKE_DIRS", BtPathname)
	pkglist("MAKE_DIRS_PERMS", BtPerms)
	acl("MAKE_ENV", lkShell, BtShellWord,
		"Makefile, Makefile.common: append, set, use",
		"buildlink3.mk, builtin.mk: append",
		"*.mk: append, use")
	acl("MAKE_ENV.*", lkShell, BtShellWord,
		"Makefile, Makefile.common: append, set, use",
		"buildlink3.mk, builtin.mk: append",
		"*.mk: append, use")
	pkg("MAKE_FILE", BtPathname)
	pkglist("MAKE_FLAGS", BtShellWord)
	pkglist("MAKE_FLAGS.*", BtShellWord)
	usr("MAKE_JOBS", BtInteger)
	pkg("MAKE_JOBS_SAFE", BtYesNo)
	pkg("MAKE_PROGRAM", BtShellCommand)
	acl("MANCOMPRESSED", lkNone, BtYesNo,
		"Makefile: set",
		"Makefile.common: default, set")
	acl("MANCOMPRESSED_IF_MANZ", lkNone, BtYes,
		"Makefile: set",
		"Makefile.common: default, set")
	sys("MANGRP", lkNone, BtUserGroupName)
	sys("MANMODE", lkNone, BtFileMode)
	sys("MANOWN", lkNone, BtUserGroupName)
	pkglist("MASTER_SITES", BtFetchURL)
	// TODO: Extract the MASTER_SITE_* definitions from mk/fetch/sites.mk
	//  instead of listing them here.
	sys("MASTER_SITE_APACHE", lkShell, BtFetchURL)
	sys("MASTER_SITE_BACKUP", lkShell, BtFetchURL)
	sys("MASTER_SITE_CRATESIO", lkShell, BtFetchURL)
	sys("MASTER_SITE_CYGWIN", lkShell, BtFetchURL)
	sys("MASTER_SITE_DEBIAN", lkShell, BtFetchURL)
	sys("MASTER_SITE_FREEBSD", lkShell, BtFetchURL)
	sys("MASTER_SITE_FREEBSD_LOCAL", lkShell, BtFetchURL)
	sys("MASTER_SITE_GENTOO", lkShell, BtFetchURL)
	sys("MASTER_SITE_GITHUB", lkShell, BtFetchURL)
	sys("MASTER_SITE_GNOME", lkShell, BtFetchURL)
	sys("MASTER_SITE_GNU", lkShell, BtFetchURL)
	sys("MASTER_SITE_GNUSTEP", lkShell, BtFetchURL)
	sys("MASTER_SITE_IFARCHIVE", lkShell, BtFetchURL)
	sys("MASTER_SITE_HASKELL_HACKAGE", lkShell, BtFetchURL)
	sys("MASTER_SITE_KDE", lkShell, BtFetchURL)
	sys("MASTER_SITE_LOCAL", lkShell, BtFetchURL)
	sys("MASTER_SITE_MOZILLA", lkShell, BtFetchURL)
	sys("MASTER_SITE_MOZILLA_ALL", lkShell, BtFetchURL)
	sys("MASTER_SITE_MOZILLA_ESR", lkShell, BtFetchURL)
	sys("MASTER_SITE_MYSQL", lkShell, BtFetchURL)
	sys("MASTER_SITE_NETLIB", lkShell, BtFetchURL)
	sys("MASTER_SITE_OPENBSD", lkShell, BtFetchURL)
	sys("MASTER_SITE_OPENOFFICE", lkShell, BtFetchURL)
	sys("MASTER_SITE_OSDN", lkShell, BtFetchURL)
	sys("MASTER_SITE_PERL_CPAN", lkShell, BtFetchURL)
	sys("MASTER_SITE_PGSQL", lkShell, BtFetchURL)
	sys("MASTER_SITE_PYPI", lkShell, BtFetchURL)
	sys("MASTER_SITE_R_CRAN", lkShell, BtFetchURL)
	sys("MASTER_SITE_RUBYGEMS", lkShell, BtFetchURL)
	sys("MASTER_SITE_SOURCEFORGE", lkShell, BtFetchURL)
	sys("MASTER_SITE_SUNSITE", lkShell, BtFetchURL)
	sys("MASTER_SITE_SUSE", lkShell, BtFetchURL)
	sys("MASTER_SITE_TEX_CTAN", lkShell, BtFetchURL)
	sys("MASTER_SITE_XCONTRIB", lkShell, BtFetchURL)
	sys("MASTER_SITE_XEMACS", lkShell, BtFetchURL)
	sys("MASTER_SITE_XORG", lkShell, BtFetchURL)
	pkglist("MESSAGE_SRC", BtPathname)
	acl("MESSAGE_SUBST", lkShell, BtShellWord,
		"Makefile, Makefile.common, options.mk: append")
	pkg("META_PACKAGE", BtYes)
	sys("MISSING_FEATURES", lkShell, BtIdentifier)
	acl("MYSQL_VERSIONS_ACCEPTED", lkShell, mysqlVersions,
		"Makefile: set")
	usr("MYSQL_VERSION_DEFAULT", BtVersion)
	sys("NATIVE_CC", lkNone, BtShellCommand) // See mk/platform/tools.NetBSD.mk (and some others).
	sys("NM", lkNone, BtShellCommand)
	sys("NONBINMODE", lkNone, BtFileMode)
	pkglist("NOT_FOR_COMPILER", compilers)
	pkglist("NOT_FOR_BULK_PLATFORM", BtMachinePlatformPattern)
	pkglist("NOT_FOR_PLATFORM", BtMachinePlatformPattern)
	pkg("NOT_FOR_UNPRIVILEGED", BtYesNo)
	pkglist("NOT_PAX_ASLR_SAFE", BtPathmask)
	pkglist("NOT_PAX_MPROTECT_SAFE", BtPathmask)
	acl("NO_BIN_ON_CDROM", lkNone, BtRestricted,
		"Makefile, Makefile.common, *.mk: set")
	acl("NO_BIN_ON_FTP", lkNone, BtRestricted,
		"Makefile, Makefile.common, *.mk: set")
	pkgload("NO_BUILD", lkNone, BtYes)
	pkg("NO_CHECKSUM", BtYes)
	pkg("NO_CONFIGURE", BtYes)
	acl("NO_EXPORT_CPP", lkNone, BtYes,
		"Makefile: set")
	pkg("NO_EXTRACT", BtYes)
	pkg("NO_INSTALL_MANPAGES", BtYes) // only has an effect for Imake packages.
	acl("NO_PKGTOOLS_REQD_CHECK", lkNone, BtYes,
		"Makefile: set")
	acl("NO_SRC_ON_CDROM", lkNone, BtRestricted,
		"Makefile, Makefile.common, *.mk: set")
	acl("NO_SRC_ON_FTP", lkNone, BtRestricted,
		"Makefile, Makefile.common, *.mk: set")
	sysload("OBJECT_FMT", lkNone, enum("COFF ECOFF ELF SOM XCOFF Mach-O PE a.out"))
	pkglist("ONLY_FOR_COMPILER", compilers)
	pkglist("ONLY_FOR_PLATFORM", BtMachinePlatformPattern)
	pkg("ONLY_FOR_UNPRIVILEGED", BtYesNo)
	sysload("OPSYS", lkNone, BtIdentifier)
	acl("OPSYSVARS", lkShell, BtVariableName,
		"Makefile, Makefile.common: append")
	acl("OSVERSION_SPECIFIC", lkNone, BtYes,
		"Makefile, Makefile.common: set")
	sysload("OS_VERSION", lkNone, BtVersion)
	sysload("OSX_VERSION", lkNone, BtVersion) // See mk/platform/Darwin.mk.
	pkg("OVERRIDE_DIRDEPTH*", BtInteger)
	pkg("OVERRIDE_GNU_CONFIG_SCRIPTS", BtYes)
	acl("OWNER", lkNone, BtMailAddress,
		"Makefile: set",
		"Makefile.common: default")
	pkglist("OWN_DIRS", BtPathname)
	pkglist("OWN_DIRS_PERMS", BtPerms)
	sys("PAMBASE", lkNone, BtPathname)
	usr("PAM_DEFAULT", enum("linux-pam openpam solaris-pam"))
	pkg("PATCHDIR", BtRelativePkgPath)
	pkglist("PATCHFILES", BtFileName)
	pkglist("PATCH_ARGS", BtShellWord)
	acl("PATCH_DIST_ARGS", lkShell, BtShellWord,
		"Makefile: set, append")
	pkg("PATCH_DIST_CAT", BtShellCommand)
	acl("PATCH_DIST_STRIP*", lkNone, BtShellWord,
		"buildlink3.mk, builtin.mk: none",
		"Makefile, Makefile.common, *.mk: set")
	acl("PATCH_SITES", lkShell, BtFetchURL,
		"Makefile, Makefile.common, options.mk: set")
	pkg("PATCH_STRIP", BtShellWord)
	sys("PATH", lkNone, BtPathlist)       // From the PATH environment variable.
	sys("PAXCTL", lkNone, BtShellCommand) // See mk/pax.mk.
	acl("PERL5_PACKLIST", lkShell, BtPerl5Packlist,
		"Makefile: set",
		"options.mk: set, append")
	pkg("PERL5_PACKLIST_DIR", BtPathname)
	pkglist("PERL5_REQD", BtVersion)
	sys("PERL5_INSTALLARCHLIB", lkNone, BtPathname) // See lang/perl5/vars.mk
	sys("PERL5_INSTALLSCRIPT", lkNone, BtPathname)
	sys("PERL5_INSTALLVENDORBIN", lkNone, BtPathname)
	sys("PERL5_INSTALLVENDORSCRIPT", lkNone, BtPathname)
	sys("PERL5_INSTALLVENDORARCH", lkNone, BtPathname)
	sys("PERL5_INSTALLVENDORLIB", lkNone, BtPathname)
	sys("PERL5_INSTALLVENDORMAN1DIR", lkNone, BtPathname)
	sys("PERL5_INSTALLVENDORMAN3DIR", lkNone, BtPathname)
	sys("PERL5_SUB_INSTALLARCHLIB", lkNone, BtPrefixPathname) // See lang/perl5/vars.mk
	sys("PERL5_SUB_INSTALLSCRIPT", lkNone, BtPrefixPathname)
	sys("PERL5_SUB_INSTALLVENDORBIN", lkNone, BtPrefixPathname)
	sys("PERL5_SUB_INSTALLVENDORSCRIPT", lkNone, BtPrefixPathname)
	sys("PERL5_SUB_INSTALLVENDORARCH", lkNone, BtPrefixPathname)
	sys("PERL5_SUB_INSTALLVENDORLIB", lkNone, BtPrefixPathname)
	sys("PERL5_SUB_INSTALLVENDORMAN1DIR", lkNone, BtPrefixPathname)
	sys("PERL5_SUB_INSTALLVENDORMAN3DIR", lkNone, BtPrefixPathname)
	pkg("PERL5_USE_PACKLIST", BtYesNo)
	sys("PGSQL_PREFIX", lkNone, BtPathname)
	pkglist("PGSQL_VERSIONS_ACCEPTED", pgsqlVersions)
	usr("PGSQL_VERSION_DEFAULT", BtVersion)
	sys("PG_LIB_EXT", lkNone, enum("dylib so"))
	sys("PGSQL_TYPE", lkNone, enumFrom("mk/pgsql.buildlink3.mk",
		"postgresql11-client",
		"PGSQL_TYPE"))
	sys("PGPKGSRCDIR", lkNone, BtPathname)
	sys("PHASE_MSG", lkNone, BtShellCommand)
	usr("PHP_VERSION_REQD", BtVersion)
	acl("PHP_PKG_PREFIX", lkNone,
		enumFromDirs("lang", `^php(\d+)$`, "php$1", "php56 php71 php72 php73"),
		"special:phpversion.mk: set",
		"*: use-loadtime, use")
	sys("PKGBASE", lkNone, BtIdentifier)
	acl("PKGCONFIG_FILE.*", lkShell, BtPathname,
		"builtin.mk: set, append",
		"special:pkgconfig-builtin.mk: use-loadtime")
	acl("PKGCONFIG_OVERRIDE", lkShell, BtPathmask,
		"Makefile: set, append",
		"Makefile.common: append")
	pkg("PKGCONFIG_OVERRIDE_STAGE", BtStage)
	pkg("PKGDIR", BtRelativePkgDir)
	sys("PKGDIRMODE", lkNone, BtFileMode)
	sys("PKGLOCALEDIR", lkNone, BtPathname)
	pkg("PKGNAME", BtPkgName)
	sys("PKGNAME_NOREV", lkNone, BtPkgName)
	sysload("PKGPATH", lkNone, BtPathname)
	sys("PKGREPOSITORY", lkNone, BtUnknown)
	acl("PKGREVISION", lkNone, BtPkgRevision,
		"Makefile: set",
		"*: none")
	sys("PKGSRCDIR", lkNone, BtPathname)
	acl("PKGSRCTOP", lkNone, BtYes,
		"Makefile: set")
	sys("PKGSRC_SETENV", lkNone, BtShellCommand)
	sys("PKGTOOLS_ENV", lkShell, BtShellWord)
	sys("PKGVERSION", lkNone, BtVersion)
	sys("PKGVERSION_NOREV", lkNone, BtVersion) // Without the nb* part.
	sys("PKGWILDCARD", lkNone, BtFileMask)
	sys("PKG_ADMIN", lkNone, BtShellCommand)
	sys("PKG_APACHE", lkNone, enum("apache24"))
	pkglist("PKG_APACHE_ACCEPTED", enum("apache24"))
	usr("PKG_APACHE_DEFAULT", enum("apache24"))
	sysload("PKG_BUILD_OPTIONS.*", lkShell, BtOption)
	usr("PKG_CONFIG", BtYes)
	// ^^ No, this is not the popular command from GNOME, but the setting
	// whether the pkgsrc user wants configuration files automatically
	// installed or not.
	sys("PKG_CREATE", lkNone, BtShellCommand)
	sys("PKG_DBDIR", lkNone, BtPathname)
	cmdline("PKG_DEBUG_LEVEL", lkNone, BtInteger)
	usrlist("PKG_DEFAULT_OPTIONS", BtOption)
	sys("PKG_DELETE", lkNone, BtShellCommand)
	acl("PKG_DESTDIR_SUPPORT", lkShell, enum("destdir user-destdir"),
		"Makefile, Makefile.common: set")
	pkglist("PKG_FAIL_REASON", BtShellWord)
	sysload("PKG_FORMAT", lkNone, BtIdentifier)
	acl("PKG_GECOS.*", lkNone, BtMessage,
		"Makefile: set")
	acl("PKG_GID.*", lkNone, BtInteger,
		"Makefile: set")
	pkglist("PKG_GROUPS", BtShellWord)
	pkglist("PKG_GROUPS_VARS", BtVariableName)
	pkg("PKG_HOME.*", BtPathname)
	acl("PKG_HACKS", lkShell, BtIdentifier,
		"hacks.mk: append")
	sys("PKG_INFO", lkNone, BtShellCommand)
	sys("PKG_JAVA_HOME", lkNone, BtPathname)
	sys("PKG_JVM", lkNone, jvms)
	acl("PKG_JVMS_ACCEPTED", lkShell, jvms,
		"Makefile: set",
		"Makefile.common: default, set")
	usr("PKG_JVM_DEFAULT", jvms)
	acl("PKG_LEGACY_OPTIONS", lkShell, BtOption,
		"options.mk: set, append")
	pkg("PKG_LIBTOOL", BtPathname)
	sysload("PKG_OPTIONS", lkShell, BtOption)
	usrlist("PKG_OPTIONS.*", BtOption)
	opt := pkg         // TODO: force package options to only be set in options.mk
	optlist := pkglist // TODO: force package options to only be set in options.mk
	optlist("PKG_OPTIONS_DEPRECATED_WARNINGS", BtShellWord)
	optlist("PKG_OPTIONS_GROUP.*", BtOption)
	optlist("PKG_OPTIONS_LEGACY_OPTS", BtUnknown)
	optlist("PKG_OPTIONS_LEGACY_VARS", BtUnknown)
	optlist("PKG_OPTIONS_NONEMPTY_SETS", BtIdentifier)
	optlist("PKG_OPTIONS_OPTIONAL_GROUPS", BtIdentifier)
	optlist("PKG_OPTIONS_REQUIRED_GROUPS", BtIdentifier)
	optlist("PKG_OPTIONS_SET.*", BtOption)
	opt("PKG_OPTIONS_VAR", BtPkgOptionsVar)
	acl("PKG_PRESERVE", lkNone, BtYes,
		"Makefile: set")
	acl("PKG_SHELL", lkNone, BtPathname,
		"Makefile, Makefile.common: set")
	acl("PKG_SHELL.*", lkNone, BtPathname,
		"Makefile, Makefile.common: set")
	sys("PKG_SHLIBTOOL", lkNone, BtPathname)
	pkglist("PKG_SKIP_REASON", BtShellWord)
	optlist("PKG_SUGGESTED_OPTIONS", BtOption)
	optlist("PKG_SUGGESTED_OPTIONS.*", BtOption)
	optlist("PKG_SUPPORTED_OPTIONS", BtOption)
	acl("PKG_SYSCONFDIR*", lkNone, BtPathname,
		"Makefile: set, use, use-loadtime",
		"buildlink3.mk, builtin.mk: use-loadtime",
		"Makefile.*, *.mk: default, set, use, use-loadtime")
	pkglist("PKG_SYSCONFDIR_PERMS", BtPerms)
	sys("PKG_SYSCONFBASEDIR", lkNone, BtPathname)
	pkg("PKG_SYSCONFSUBDIR", BtPathname)
	pkg("PKG_SYSCONFVAR", BtIdentifier)
	acl("PKG_UID", lkNone, BtInteger,
		"Makefile: set")
	pkglist("PKG_USERS", BtShellWord)
	pkglist("PKG_USERS_VARS", BtVariableName)
	acl("PKG_USE_KERBEROS", lkNone, BtYes,
		"Makefile, Makefile.common: set")
	pkgload("PLIST.*", lkNone, BtYes)
	pkglist("PLIST_VARS", BtIdentifier)
	pkglist("PLIST_SRC", BtRelativePkgPath)
	pkglist("PLIST_SUBST", BtShellWord)
	pkg("PLIST_TYPE", enum("dynamic static"))
	acl("PREPEND_PATH", lkShell, BtPathname,
		"*: append")
	acl("PREFIX", lkNone, BtPathname,
		"*: use")
	acl("PREV_PKGPATH", lkNone, BtPathname,
		"*: use") // doesn't exist any longer
	acl("PRINT_PLIST_AWK", lkNone, BtAwkCommand,
		"*: append")
	pkglist("PRIVILEGED_STAGES", enum("build install package clean"))
	pkg("PTHREAD_AUTO_VARS", BtYesNo)
	sys("PTHREAD_CFLAGS", lkShell, BtCFlag)
	sys("PTHREAD_LDFLAGS", lkShell, BtLdFlag)
	sys("PTHREAD_LIBS", lkShell, BtLdFlag)
	acl("PTHREAD_OPTS", lkShell, enum("native optional require"),
		"Makefile, Makefile.*, *.mk: default, set, append")
	sysload("PTHREAD_TYPE", lkNone, BtIdentifier) // Or "native" or "none".
	pkg("PY_PATCHPLIST", BtYes)
	acl("PYPKGPREFIX", lkNone,
		enumFromDirs("lang", `^python(\d+)$`, "py$1", "py27 py36"),
		"special:pyversion.mk: set",
		"*: use-loadtime, use")
	// See lang/python/pyversion.mk
	pkg("PYTHON_FOR_BUILD_ONLY", enum("yes no test tool YES"))
	pkglist("REPLACE_PYTHON", BtPathmask)
	pkglist("PYTHON_VERSIONS_ACCEPTED", BtVersion)
	pkglist("PYTHON_VERSIONS_INCOMPATIBLE", BtVersion)
	usr("PYTHON_VERSION_DEFAULT", BtVersion)
	usr("PYTHON_VERSION_REQD", BtVersion)
	pkglist("PYTHON_VERSIONED_DEPENDENCIES", BtPythonDependency)
	sys("RANLIB", lkNone, BtShellCommand)
	pkglist("RCD_SCRIPTS", BtFileName)
	acl("RCD_SCRIPT_SRC.*", lkNone, BtPathname,
		"Makefile: set")
	acl("RCD_SCRIPT_WRK.*", lkNone, BtPathname,
		"Makefile: set")
	usr("REAL_ROOT_USER", BtUserGroupName)
	usr("REAL_ROOT_GROUP", BtUserGroupName)

	// Example:
	//  REPLACE.sys-AWK.old=    .*awk
	//  REPLACE.sys-AWK.new=    ${AWK}
	// BtUnknown since one of them is a regular expression and the other
	// is a plain string.
	pkg("REPLACE.*", BtUnknown)

	pkglist("REPLACE_AWK", BtPathmask)
	pkglist("REPLACE_BASH", BtPathmask)
	pkglist("REPLACE_CSH", BtPathmask)
	acl("REPLACE_FILES.*", lkShell, BtPathmask,
		"Makefile, Makefile.common, *.mk: default, set, append")
	acl("REPLACE_INTERPRETER", lkShell, BtIdentifier,
		"Makefile, Makefile.common, *.mk: default, set, append")
	pkglist("REPLACE_KSH", BtPathmask)
	pkglist("REPLACE_LOCALEDIR_PATTERNS", BtFileMask)
	pkglist("REPLACE_LUA", BtPathmask)
	pkglist("REPLACE_PERL", BtPathmask)
	pkglist("REPLACE_PYTHON", BtPathmask)
	pkglist("REPLACE_SH", BtPathmask)
	pkglist("REQD_DIRS", BtPathname)
	pkglist("REQD_DIRS_PERMS", BtPerms)
	pkglist("REQD_FILES", BtPathname)
	pkg("REQD_FILES_MODE", enum("0644 0640 0600 0400"))
	pkglist("REQD_FILES_PERMS", BtPerms)
	pkg("RESTRICTED", BtMessage)
	usr("ROOT_USER", BtUserGroupName)
	usr("ROOT_GROUP", BtUserGroupName)
	pkglist("RPMIGNOREPATH", BtPathmask)
	acl("RUBY_BASE", lkNone,
		enumFromDirs("lang", `^ruby(\d+)$`, "ruby$1", "ruby22 ruby23 ruby24 ruby25"),
		"special:rubyversion.mk: set",
		"*: use-loadtime, use")
	usr("RUBY_VERSION_REQD", BtVersion)
	acl("RUBY_PKGPREFIX", lkNone,
		enumFromDirs("lang", `^ruby(\d+)$`, "ruby$1", "ruby22 ruby23 ruby24 ruby25"),
		"special:rubyversion.mk: set, default, use",
		"*: use-loadtime, use")
	sys("RUN", lkNone, BtShellCommand)
	sys("RUN_LDCONFIG", lkNone, BtYesNo)
	acl("SCRIPTS_ENV", lkShell, BtShellWord,
		"Makefile, Makefile.common: append")
	usrlist("SETGID_GAMES_PERMS", BtPerms)
	usrlist("SETUID_ROOT_PERMS", BtPerms)
	pkg("SET_LIBDIR", BtYes)
	sys("SHAREGRP", lkNone, BtUserGroupName)
	sys("SHAREMODE", lkNone, BtFileMode)
	sys("SHAREOWN", lkNone, BtUserGroupName)
	sys("SHCOMMENT", lkNone, BtShellCommand)
	acl("SHLIBTOOL", lkNone, BtShellCommand,
		"Makefile: use")
	acl("SHLIBTOOL_OVERRIDE", lkShell, BtPathmask,
		"Makefile: set, append",
		"Makefile.common: append")
	sysload("SHLIB_TYPE", lkNone,
		enum("COFF ECOFF ELF SOM XCOFF Mach-O PE PEwin a.out aixlib dylib none"))
	acl("SITES.*", lkShell, BtFetchURL,
		"Makefile, Makefile.common, options.mk: set, append, use")
	usr("SMF_PREFIS", BtPathname)
	pkg("SMF_SRCDIR", BtPathname)
	pkg("SMF_NAME", BtFileName)
	pkg("SMF_MANIFEST", BtPathname)
	pkglist("SMF_INSTANCES", BtIdentifier)
	pkglist("SMF_METHODS", BtFileName)
	pkg("SMF_METHOD_SRC.*", BtPathname)
	pkg("SMF_METHOD_SHELL", BtShellCommand)
	pkglist("SPECIAL_PERMS", BtPerms)
	sys("STEP_MSG", lkNone, BtShellCommand)
	sys("STRIP", lkNone, BtShellCommand) // see mk/tools/strip.mk
	acl("SUBDIR", lkShell, BtFileName,
		"Makefile: append",
		"*: none")
	acl("SUBST_CLASSES", lkShell, BtIdentifier,
		"Makefile: set, append",
		"*: append")
	acl("SUBST_CLASSES.*", lkShell, BtIdentifier,
		"Makefile: set, append",
		"*: append") // OPSYS-specific
	acl("SUBST_FILES.*", lkShell, BtPathmask,
		"Makefile, Makefile.*, *.mk: set, append")
	acl("SUBST_FILTER_CMD.*", lkNone, BtShellCommand,
		"Makefile, Makefile.*, *.mk: set")
	acl("SUBST_MESSAGE.*", lkNone, BtMessage,
		"Makefile, Makefile.*, *.mk: set")
	acl("SUBST_SED.*", lkNone, BtSedCommands,
		"Makefile, Makefile.*, *.mk: set, append")
	pkg("SUBST_STAGE.*", BtStage)
	acl("SUBST_VARS.*", lkShell, BtVariableName,
		"Makefile, Makefile.*, *.mk: set, append")
	pkglist("SUPERSEDES", BtDependency)
	acl("TEST_DEPENDS", lkShell, BtDependencyWithPath,
		"Makefile, Makefile.common, *.mk: append")
	pkglist("TEST_DIRS", BtWrksrcSubdirectory)
	pkglist("TEST_ENV", BtShellWord)
	acl("TEST_TARGET", lkShell, BtIdentifier,
		"Makefile: set",
		"Makefile.common: default, set",
		"options.mk: set, append")
	pkglist("TEXINFO_REQD", BtVersion)
	acl("TOOL_DEPENDS", lkShell, BtDependencyWithPath,
		"Makefile, Makefile.common, *.mk: append")
	sys("TOOLS_ALIASES", lkShell, BtFileName)
	sys("TOOLS_BROKEN", lkShell, BtTool)
	sys("TOOLS_CMD.*", lkNone, BtPathname)
	acl("TOOLS_CREATE", lkShell, BtTool,
		"Makefile, Makefile.common, *.mk: append")
	acl("TOOLS_DEPENDS.*", lkShell, BtDependencyWithPath,
		"buildlink3.mk: none",
		"Makefile, Makefile.*: set, default",
		"*: use")
	sys("TOOLS_GNU_MISSING", lkShell, BtTool)
	sys("TOOLS_NOOP", lkShell, BtTool)
	sys("TOOLS_PATH.*", lkNone, BtPathname)
	sysload("TOOLS_PLATFORM.*", lkNone, BtShellCommand)
	sys("TOUCH_FLAGS", lkShell, BtShellWord)
	pkglist("UAC_REQD_EXECS", BtPrefixPathname)
	acl("UNLIMIT_RESOURCES", lkShell,
		enum("cputime datasize memorysize stacksize"),
		"Makefile: set, append",
		"Makefile.common: append")
	usr("UNPRIVILEGED_USER", BtUserGroupName)
	usr("UNPRIVILEGED_GROUP", BtUserGroupName)
	pkglist("UNWRAP_FILES", BtPathmask)
	usrlist("UPDATE_TARGET", BtIdentifier)
	pkg("USERGROUP_PHASE", enum("configure build pre-install"))
	usrlist("USER_ADDITIONAL_PKGS", BtPkgPath)
	pkg("USE_BSD_MAKEFILE", BtYes)
	acl("USE_BUILTIN.*", lkNone, BtYesNoIndirectly,
		"builtin.mk: set, use, use-loadtime",
		"Makefile, Makefile.common, *.mk: use-loadtime")
	pkg("USE_CMAKE", BtYes)
	usr("USE_DESTDIR", BtYes)
	pkglist("USE_FEATURES", BtIdentifier)
	acl("USE_GAMESGROUP", lkNone, BtYesNo,
		"buildlink3.mk, builtin.mk: none",
		"*: set, default, use")
	pkg("USE_GCC_RUNTIME", BtYesNo)
	pkg("USE_GNU_CONFIGURE_HOST", BtYesNo)
	acl("USE_GNU_ICONV", lkNone, BtYes,
		"Makefile, Makefile.common, options.mk: set, use-loadtime, use")
	acl("USE_IMAKE", lkNone, BtYes,
		"Makefile: set")
	pkg("USE_JAVA", enum("run yes build"))
	pkg("USE_JAVA2", enum("YES yes no 1.4 1.5 6 7 8"))
	acl("USE_LANGUAGES", lkShell, compilerLanguages,
		"Makefile, Makefile.common, options.mk: set, append")
	pkg("USE_LIBTOOL", BtYes)
	pkg("USE_MAKEINFO", BtYes)
	pkg("USE_MSGFMT_PLURALS", BtYes)
	pkg("USE_NCURSES", BtYes)
	pkg("USE_OLD_DES_API", BtYesNo)
	pkg("USE_PKGINSTALL", BtYes)
	pkg("USE_PKGLOCALEDIR", BtYesNo)
	usr("USE_PKGSRC_GCC", BtYes)
	acl("USE_TOOLS", lkShell, BtTool,
		"*: append, use-loadtime")
	acl("USE_TOOLS.*", lkShell, BtTool,
		"*: append, use-loadtime")
	pkg("USE_X11", BtYes)
	sys("WARNINGS", lkShell, BtShellWord)
	sys("WARNING_MSG", lkNone, BtShellCommand)
	sys("WARNING_CAT", lkNone, BtShellCommand)
	sysload("WRAPPER_DIR", lkNone, BtPathname)
	acl("WRAPPER_REORDER_CMDS", lkShell, BtWrapperReorder,
		"Makefile, Makefile.common, buildlink3.mk: append")
	pkg("WRAPPER_SHELL", BtShellCommand)
	acl("WRAPPER_TRANSFORM_CMDS", lkShell, BtWrapperTransform,
		"Makefile, Makefile.common, buildlink3.mk: append")
	sys("WRKDIR", lkNone, BtPathname)
	pkg("WRKSRC", BtWrkdirSubdirectory)
	pkglist("X11_LDFLAGS", BtLdFlag)
	sys("X11_PKGSRCDIR.*", lkNone, BtPathname)
	usr("XAW_TYPE", enum("3d neXtaw standard xpm"))
	pkglist("XMKMF_FLAGS", BtShellWord)
	pkglist("_WRAP_EXTRA_ARGS.*", BtShellWord)

	// Only for infrastructure files; see mk/misc/show.mk
	acl("_VARGROUPS", lkShell, BtIdentifier,
		"*: append")
	acl("_USER_VARS.*", lkShell, BtIdentifier,
		"*: append")
	acl("_PKG_VARS.*", lkShell, BtIdentifier,
		"*: append")
	acl("_SYS_VARS.*", lkShell, BtIdentifier,
		"*: append")
	acl("_DEF_VARS.*", lkShell, BtIdentifier,
		"*: append")
	acl("_USE_VARS.*", lkShell, BtIdentifier,
		"*: append")
}

func enum(values string) *BasicType {
	valueMap := make(map[string]bool)
	for _, value := range strings.Fields(values) {
		valueMap[value] = true
	}
	name := "enum: " + values + " " // See IsEnum
	basicType := BasicType{name, nil}
	basicType.checker = func(check *VartypeCheck) {
		check.Enum(valueMap, &basicType)
	}
	return &basicType
}

func parseACLEntries(varname string, aclEntries ...string) []ACLEntry {
	if len(aclEntries) == 0 {
		return []ACLEntry{{"*", aclpNone}}
	}

	var result []ACLEntry
	prevperms := "(first)"
	for _, arg := range aclEntries {
		fields := strings.SplitN(arg, ": ", 2)
		G.Assertf(len(fields) == 2, "Invalid ACL entry %q", arg)
		globs, perms := fields[0], ifelseStr(fields[1] == "none", "", fields[1])

		G.Assertf(perms != prevperms, "Repeated permissions %q for %q.", perms, varname)
		prevperms = perms

		var permissions ACLPermissions
		for _, perm := range strings.Split(perms, ", ") {
			switch perm {
			case "append":
				permissions |= aclpAppend
			case "default":
				permissions |= aclpSetDefault
			case "set":
				permissions |= aclpSet
			case "use":
				permissions |= aclpUse
			case "use-loadtime":
				permissions |= aclpUseLoadtime
			case "":
				break
			default:
				G.Assertf(false, "Invalid ACL permission %q for %q. "+
					"Valid permissions are append, default, set, use, use-loadtime, none.",
					perm, varname)
			}
		}

		for _, glob := range strings.Split(globs, ", ") {
			switch glob {
			case "*",
				"Makefile", "Makefile.common", "Makefile.*",
				"buildlink3.mk", "builtin.mk", "options.mk", "hacks.mk", "*.mk":
				break
			default:
				withoutSpecial := strings.TrimPrefix(glob, "special:")
				if withoutSpecial == glob {
					G.Assertf(false, "Invalid ACL glob %q for %q.", glob, varname)
				} else {
					glob = withoutSpecial
				}
			}
			for _, prev := range result {
				matched, err := path.Match(prev.glob, glob)
				G.AssertNil(err, "Invalid ACL pattern %q for %q", glob, varname)
				G.Assertf(!matched, "Unreachable ACL pattern %q for %q.", glob, varname)
			}
			result = append(result, ACLEntry{glob, permissions})
		}
	}

	if result[len(result)-1].glob != "*" {
		//println("default permissions missing for " + varname)
		result = append(result, ACLEntry{"*", aclpNone})
	}

	return result
}
