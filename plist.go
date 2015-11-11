package main

import (
	"path"
)

type PlistContext struct {
	allFiles  map[string]*Line
	allDirs   map[string]*Line
	lastFname string
}

func checkfilePlist(fname string) {
	defer tracecall("checkfilePlist", fname)()

	checkperms(fname)
	lines, err := loadLines(fname, false)
	if err != nil {
		logError(fname, NO_LINES, "Cannot be read.")
		return
	}

	if len(lines) == 0 {
		logError(fname, NO_LINES, "Must not be empty.")
		return
	}
	checklineRcsid(lines[0], `@comment `, "@comment ")

	if len(lines) == 1 {
		lines[0].logWarning("PLIST files shouldn't be empty.")
		lines[0].explainWarning(
			"One reason for empty PLISTs is that this is a newly created package",
			"and that the author didn't run \"bmake print-PLIST\" after installing",
			"the files.",
			"",
			"Another reason, common for Perl packages, is that the final PLIST is",
			"automatically generated. Since the source PLIST is not used at all,",
			"you can remove it.",
			"",
			"Meta packages also don't need a PLIST file.")
	}

	pctx := &PlistContext{}
	pctx.allFiles = make(map[string]*Line)
	pctx.allDirs = make(map[string]*Line)

	extraLines := make([]*Line, 0)
	if path.Base(fname) == "PLIST.common_end" {
		commonLines, err := loadLines(path.Dir(fname)+"/PLIST.common", false)
		if err == nil {
			extraLines = commonLines
		}
	}

	// Collect all files and directories that appear in the PLIST file.
	for _, line := range append(extraLines, lines...) {
		text := line.text

		if hasPrefix(text, "${") {
			if m, varname, rest := match2(text, `^\$\{([\w_]+)\}(.*)`); m {
				if G.pkgContext.plistSubstCond[varname] {
					_ = G.opts.optDebugMisc && line.logDebug("Removed PLIST_SUBST conditional %q.", varname)
					text = rest
				}
			}
		}

		if match0(text, `^[\w$]`) {
			pctx.allFiles[text] = line
			for dir := path.Dir(text); dir != "."; dir = path.Dir(dir) {
				pctx.allDirs[dir] = line
			}
		}

		if hasPrefix(text, "@") {
			if m, dirname := match1(text, `^\@exec \$\{MKDIR\} %D/(.*)$`); m {
				for dir := dirname; dir != "."; dir = path.Dir(dir) {
					pctx.allDirs[dir] = line
				}
			}
		}
	}

	for _, line := range lines {
		text := line.text
		pline := &PlistLine{line}
		pline.checkTrailingWhitespace()

		if m, cmd, arg := match2(text, `^(?:\$\{[\w_]+\})?\@([a-z-]+)\s+(.*)`); m {
			pline.checkDirective(cmd, arg)
		} else if m, dirname, basename := match2(text, `^([A-Za-z0-9\$].*)/([^/]+)$`); m {
			pline.checkPathname(pctx, dirname, basename)
		} else if match0(text, `^\$\{[\w_]+\}$`) {
			// A variable on its own line.
		} else {
			line.logWarning("Unknown line type.")
		}
	}

	checklinesTrailingEmptyLines(lines)
	autofix(lines)
}

type PlistLine struct {
	line *Line
}

func (pline *PlistLine) checkTrailingWhitespace() {
	line := pline.line

	if match0(line.text, `\s$`) {
		line.logError("pkgsrc does not support filenames ending in white-space.")
		line.explainError(
			"Each character in the PLIST is relevant, even trailing white-space.")
	}
}

func (pline *PlistLine) checkDirective(cmd, arg string) {
	line := pline.line

	if cmd == "unexec" {
		if m, arg := match1(arg, `^(?:rmdir|\$\{RMDIR\} \%D/)(.*)`); m {
			if !contains(arg, "true") && !contains(arg, "${TRUE}") {
				line.logWarning("Please remove this line. It is no longer necessary.")
			}
		}
	}

	switch cmd {
	case "exec", "unexec":
		switch {
		case contains(arg, "install-info"),
			contains(arg, "${INSTALL_INFO}"):
			line.logWarning("@exec/unexec install-info is deprecated.")
		case contains(arg, "ldconfig") && !contains(arg, "/usr/bin/true"):
			line.logError("ldconfig must be used with \"||/usr/bin/true\".")
		}

	case "comment":
		// Nothing to do.

	case "dirrm":
		line.logWarning("@dirrm is obsolete. Please remove this line.")
		line.explainWarning(
			"Directories are removed automatically when they are empty.",
			"When a package needs an empty directory, it can use the @pkgdir",
			"command in the PLIST")

	case "imake-man":
		args := splitOnSpace(arg)
		if len(args) != 3 {
			line.logWarning("Invalid number of arguments for imake-man.")
		} else {
			if args[2] == "${IMAKE_MANNEWSUFFIX}" {
				pline.warnAboutPlistImakeMannewsuffix()
			}
		}

	case "pkgdir":
		// Nothing to check.

	default:
		line.logWarning("Unknown PLIST directive \"@%s\".", cmd)
	}
}

func (pline *PlistLine) checkPathname(pctx *PlistContext, dirname, basename string) {
	line := pline.line
	text := line.text

	if G.opts.optWarnPlistSort && match0(text, `^\w`) && !match0(text, reUnresolvedVar) {
		if pctx.lastFname != "" {
			if pctx.lastFname > text {
				line.logWarning("%q should be sorted before %q.", text, pctx.lastFname)
				line.explainWarning(
					"For aesthetic reasons, the files in the PLIST should be sorted",
					"alphabetically.")
			} else if pctx.lastFname == text {
				line.logError("Duplicate filename.")
			}
		}
		pctx.lastFname = text
	}

	if contains(basename, "${IMAKE_MANNEWSUFFIX}") {
		pline.warnAboutPlistImakeMannewsuffix()
	}

	switch {
	case hasPrefix(dirname, "bin/"):
		line.logWarning("The bin/ directory should not have subdirectories.")

	case dirname == "bin":
		switch {
		case pctx.allFiles["man/man1/"+basename+".1"] != nil:
		case pctx.allFiles["man/man6/"+basename+".6"] != nil:
		case pctx.allFiles["${IMAKE_MAN_DIR}/"+basename+".${IMAKE_MANNEWSUFFIX}"] != nil:
		default:
			if G.opts.optWarnExtra {
				line.logWarning("Manual page missing for bin/%s.", basename)
				line.explainWarning(
					"All programs that can be run directly by the user should have a manual",
					"page for quick reference. The programs in the bin/ directory should have",
					"corresponding manual pages in section 1 (filename program.1), not in",
					"section 8.")
			}
		}

	case hasPrefix(text, "doc/"):
		line.logError("Documentation must be installed under share/doc, not doc.")

	case hasPrefix(text, "etc/rc.d/"):
		line.logError("RCD_SCRIPTS must not be registered in the PLIST. Please use the RCD_SCRIPTS framework.")

	case hasPrefix(text, "etc/"):
		f := "mk/pkginstall/bsd.pkginstall.mk"
		line.logError("Configuration files must not be registered in the PLIST. "+
			"Please use the CONF_FILES framework, which is described in %s.", f)

	case hasPrefix(text, "include/") && match0(text, `^include/.*\.(?:h|hpp)$`):
		// Fine.

	case text == "info/dir":
		line.logError("\"info/dir\" must not be listed. Use install-info to add/remove an entry.")

	case hasPrefix(text, "info/"):
		if G.pkgContext.vardef["INFO_FILES"] == nil {
			line.logWarning("Packages that install info files should set INFO_FILES.")
		}

	case G.pkgContext.effectivePkgbase != nil && hasPrefix(text, "lib/"+*G.pkgContext.effectivePkgbase+"/"):
		// Fine.

	case hasPrefix(text, "lib/locale/"):
		line.logError("\"lib/locale\" must not be listed. Use ${PKGLOCALEDIR}/locale and set USE_PKGLOCALEDIR instead.")

	case hasPrefix(text, "lib/"):
		if m, dir, lib, ext := match3(text, `^(lib/(?:.*/)*)([^/]+)\.(so|a|la)$`); m {
			if dir == "lib/" && !hasPrefix(lib, "lib") {
				_ = G.opts.optWarnExtra && line.logWarning("Library filename does not start with \"lib\".")
			}
			if ext == "la" {
				if G.pkgContext.vardef["USE_LIBTOOL"] == nil {
					line.logWarning("Packages that install libtool libraries should define USE_LIBTOOL.")
				}
			}
		}

	case hasPrefix(text, "man/"):
		if m, catOrMan, section, manpage, ext, gz := match5(text, `^man/(cat|man)(\w+)/(.*?)\.(\w+)(\.gz)?$`); m {

			if !match0(section, `^[\dln]$`) {
				line.logWarning("Unknown section %q for manual page.", section)
			}

			if catOrMan == "cat" && pctx.allFiles["man/man"+section+"/"+manpage+"."+section] == nil {
				line.logWarning("Preformatted manual page without unformatted one.")
			}

			if catOrMan == "cat" {
				if ext != "0" {
					line.logWarning("Preformatted manual pages should end in \".0\".")
				}
			} else {
				if section != ext {
					line.logWarning("Mismatch between the section (%s) and extension (%s) of the manual page.", section, ext)
				}
			}

			if gz != "" {
				line.logNote("The .gz extension is unnecessary for manual pages.")
				line.explainNote(
					"Whether the manual pages are installed in compressed form or not is",
					"configured by the pkgsrc user. Compression and decompression takes place",
					"automatically, no matter if the .gz extension is mentioned in the PLIST",
					"or not.")
			}
		} else {
			line.logWarning("Invalid filename %q for manual page.", text)
		}

	case hasPrefix(text, "sbin/"):
		binname := text[5:]

		if pctx.allFiles["man/man8/"+binname+".8"] == nil && G.opts.optWarnExtra {
			line.logWarning("Manual page missing for sbin/%s.", binname)
			line.explainWarning(
				"All programs that can be run directly by the user should have a manual",
				"page for quick reference. The programs in the sbin/ directory should have",
				"corresponding manual pages in section 8 (filename program.8), not in",
				"section 1.")
		}

	case hasPrefix(text, "share/applications/") && hasSuffix(text, ".desktop"):
		f := "../../sysutils/desktop-file-utils/desktopdb.mk"
		if G.pkgContext.included[f] == nil {
			line.logWarning("Packages that install a .desktop entry should .include %q.", f)
			line.explainWarning(
				"If *.desktop files contain MimeType keys, the global MIME type registry",
				"must be updated by desktop-file-utils. Otherwise, this warning is harmless.")
		}

	case hasPrefix(text, "share/icons/hicolor/") && G.pkgContext.pkgpath != "graphics/hicolor-icon-theme":
		f := "../../graphics/hicolor-icon-theme/buildlink3.mk"
		if G.pkgContext.included[f] == nil {
			line.logError("Packages that install hicolor icons must include %q in the Makefile.", f)
		}

	case hasPrefix(text, "share/icons/gnome") && G.pkgContext.pkgpath != "graphics/gnome-icon-theme":
		f := "../../graphics/gnome-icon-theme/buildlink3.mk"
		if G.pkgContext.included[f] == nil {
			line.logError("The package Makefile must include %q.", f)
			line.explainError(
				"Packages that install GNOME icons must maintain the icon theme cache.")
		}

	case dirname == "share/aclocal" && hasSuffix(basename, ".m4"):
		// Fine.

	case hasPrefix(text, "share/doc/html/"):
		_ = G.opts.optWarnPlistDepr && line.logWarning("Use of \"share/doc/html\" is deprecated. Use \"share/doc/${PKGBASE}\" instead.")

	case G.pkgContext.effectivePkgbase != nil && (hasPrefix(text, "share/doc/"+*G.pkgContext.effectivePkgbase+"/") ||
		hasPrefix(text, "share/examples/"+*G.pkgContext.effectivePkgbase+"/")):
		// Fine.

	case text == "share/icons/hicolor/icon-theme.cache" && G.pkgContext.pkgpath != "graphics/hicolor-icon-theme":
		line.logError("This file must not appear in any PLIST file.")
		line.explainError(
			"Remove this line and add the following line to the package Makefile.",
			"",
			".include \"../../graphics/hicolor-icon-theme/buildlink3.mk\"")

	case hasPrefix(text, "share/info/"):
		line.logWarning("Info pages should be installed into info/, not share/info/.")
		line.explainWarning(
			"To fix this, you should add INFO_FILES=yes to the package Makefile.")

	case hasPrefix(text, "share/locale/") && hasSuffix(text, ".mo"):
		// Fine.

	case hasPrefix(text, "share/man/"):
		line.logWarning("Man pages should be installed into man/, not share/man/.")

	default:
		_ = G.opts.optDebugUnchecked && line.logDebug("Unchecked pathname %q.", text)
	}

	if contains(text, "${PKGLOCALEDIR}") && G.pkgContext.vardef["USE_PKGLOCALEDIR"] == nil {
		line.logWarning("PLIST contains ${PKGLOCALEDIR}, but USE_PKGLOCALEDIR was not found.")
	}

	if contains(text, "/CVS/") {
		line.logWarning("CVS files should not be in the PLIST.")
	}
	if hasSuffix(text, ".orig") {
		line.logWarning(".orig files should not be in the PLIST.")
	}
	if hasSuffix(text, "/perllocal.pod") {
		line.logWarning("perllocal.pod files should not be in the PLIST.")
		line.explainWarning(
			"This file is handled automatically by the INSTALL/DEINSTALL scripts,",
			"since its contents changes frequently.")
	}

	if m, basename, _ := match2(text, `^(.*)(\.a|\.so[0-9.]*)$`); m {
		if laLine := pctx.allFiles[basename+".la"]; laLine != nil {
			line.logWarning("Redundant library found. The libtool library is in line %s.", laLine)
		}
	}
}

func (pline *PlistLine) warnAboutPlistImakeMannewsuffix() {
	line := pline.line

	line.logWarning("IMAKE_MANNEWSUFFIX is not meant to appear in PLISTs.")
	line.explainWarning(
		"This is the result of a print-PLIST call that has not been edited",
		"manually by the package maintainer. Please replace the",
		"IMAKE_MANNEWSUFFIX with:",
		"",
		"\tIMAKE_MAN_SUFFIX for programs,",
		"\tIMAKE_LIBMAN_SUFFIX for library functions,",
		"\tIMAKE_FILEMAN_SUFFIX for file formats,",
		"\tIMAKE_GAMEMAN_SUFFIX for games,",
		"\tIMAKE_MISCMAN_SUFFIX for other man pages.")
}
