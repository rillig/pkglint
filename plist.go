package main

import (
	"path"
	"strings"
)

func checklinesPlist(lines []*Line) {
	defer tracecall1("checklinesPlist", lines[0].fname)()

	checklineRcsid(lines[0], `@comment `, "@comment ")

	if len(lines) == 1 {
		lines[0].warn0("PLIST files shouldn't be empty.")
		explain(
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

	ck := &PlistChecker{
		make(map[string]*PlistLine),
		make(map[string]*PlistLine),
		""}
	ck.check(lines)
}

type PlistChecker struct {
	allFiles  map[string]*PlistLine
	allDirs   map[string]*PlistLine
	lastFname string
}

type PlistLine struct {
	line        *Line
	conditional string
	text        string
}

func (ck *PlistChecker) check(plainLines []*Line) {
	plines := ck.newLines(plainLines)
	ck.collectFilesAndDirs(plines)

	if fname := plines[0].line.fname; path.Base(fname) == "PLIST.common_end" {
		commonLines, err := readLines(strings.TrimSuffix(fname, "_end"), false)
		if err == nil {
			ck.collectFilesAndDirs(ck.newLines(commonLines))
		}
	}

	for _, pline := range plines {
		ck.checkline(pline)
		pline.checkTrailingWhitespace()
	}

	checklinesTrailingEmptyLines(plainLines)
	saveAutofixChanges(plainLines)
}

func (ck *PlistChecker) newLines(lines []*Line) []*PlistLine {
	plines := make([]*PlistLine, len(lines))
	for i, line := range lines {
		conditional, text := "", line.text
		if hasPrefix(text, "${PLIST.") {
			if m, cond, rest := match2(text, `^\$\{([\w.]+)\}(.*)`); m {
				conditional, text = cond, rest
			}
		}
		plines[i] = &PlistLine{line, conditional, text}
	}
	return plines
}

func (ck *PlistChecker) collectFilesAndDirs(plines []*PlistLine) {
	for _, pline := range plines {
		if text := pline.text; len(text) > 0 {
			first := text[0]
			switch {
			case 'a' <= first && first <= 'z',
				first == '$',
				'A' <= first && first <= 'Z',
				'0' <= first && first <= '9':
				if ck.allFiles[text] == nil {
					ck.allFiles[text] = pline
				}
				for dir := path.Dir(text); dir != "."; dir = path.Dir(dir) {
					ck.allDirs[dir] = pline
				}
			case first == '@':
				if m, dirname := match1(text, `^@exec \$\{MKDIR\} %D/(.*)$`); m {
					for dir := dirname; dir != "."; dir = path.Dir(dir) {
						ck.allDirs[dir] = pline
					}
				}
			}

		}
	}
}

func (ck *PlistChecker) checkline(pline *PlistLine) {
	text := pline.text
	if hasAlnumPrefix(text) {
		ck.checkpath(pline)
	} else if m, cmd, arg := match2(text, `^(?:\$\{[\w.]+\})?@([a-z-]+)\s+(.*)`); m {
		pline.checkDirective(cmd, arg)
	} else if hasPrefix(text, "$") {
		ck.checkpath(pline)
	} else {
		pline.line.warn0("Unknown line type.")
	}
}

func (ck *PlistChecker) checkpath(pline *PlistLine) {
	line, text := pline.line, pline.text
	sdirname, basename := path.Split(text)
	dirname := strings.TrimSuffix(sdirname, "/")

	ck.checkSorted(pline)

	if strings.Contains(basename, "${IMAKE_MANNEWSUFFIX}") {
		pline.warnAboutPlistImakeMannewsuffix()
	}

	topdir := ""
	if firstSlash := strings.IndexByte(text, '/'); firstSlash != -1 {
		topdir = text[:firstSlash]
	}

	switch topdir {
	case "bin":
		ck.checkpathBin(pline, dirname, basename)
	case "doc":
		line.error0("Documentation must be installed under share/doc, not doc.")
	case "etc":
		ck.checkpathEtc(pline, dirname, basename)
	case "info":
		ck.checkpathInfo(pline, dirname, basename)
	case "lib":
		ck.checkpathLib(pline, dirname, basename)
	case "man":
		ck.checkpathMan(pline)
	case "sbin":
		ck.checkpathSbin(pline)
	case "share":
		ck.checkpathShare(pline)
	}

	if strings.Contains(text, "${PKGLOCALEDIR}") && G.pkg != nil && G.pkg.vardef["USE_PKGLOCALEDIR"] == nil {
		line.warn0("PLIST contains ${PKGLOCALEDIR}, but USE_PKGLOCALEDIR was not found.")
	}

	if strings.Contains(text, "/CVS/") {
		line.warn0("CVS files should not be in the PLIST.")
	}
	if hasSuffix(text, ".orig") {
		line.warn0(".orig files should not be in the PLIST.")
	}
	if hasSuffix(text, "/perllocal.pod") {
		line.warn0("perllocal.pod files should not be in the PLIST.")
		explain2(
			"This file is handled automatically by the INSTALL/DEINSTALL scripts,",
			"since its contents changes frequently.")
	}
}

func (ck *PlistChecker) checkSorted(pline *PlistLine) {
	if text := pline.text; G.opts.WarnPlistSort && hasAlnumPrefix(text) && !containsVarRef(text) {
		if ck.lastFname != "" {
			if ck.lastFname > text {
				pline.line.warn2("%q should be sorted before %q.", text, ck.lastFname)
				explain1(
					"The files in the PLIST should be sorted alphabetically.")
			}
			if prev := ck.allFiles[text]; prev != nil && prev != pline {
				if !pline.line.autofixDelete() {
					pline.line.errorf("Duplicate filename %q, already appeared in %s:%s.", text, prev.line.fname, prev.line.linenos())
				}
			}
		}
		ck.lastFname = text
	}
}

func (ck *PlistChecker) checkpathBin(pline *PlistLine, dirname, basename string) {
	if strings.Contains(dirname, "/") {
		pline.line.warn0("The bin/ directory should not have subdirectories.")
		return
	}

	if G.opts.WarnExtra &&
		ck.allFiles["man/man1/"+basename+".1"] == nil &&
		ck.allFiles["man/man6/"+basename+".6"] == nil &&
		ck.allFiles["${IMAKE_MAN_DIR}/"+basename+".${IMAKE_MANNEWSUFFIX}"] == nil {
		pline.line.warn1("Manual page missing for bin/%s.", basename)
		explain(
			"All programs that can be run directly by the user should have a manual",
			"page for quick reference. The programs in the bin/ directory should have",
			"corresponding manual pages in section 1 (filename program.1), not in",
			"section 8.")
	}
}

func (ck *PlistChecker) checkpathEtc(pline *PlistLine, dirname, basename string) {
	if hasPrefix(pline.text, "etc/rc.d/") {
		pline.line.error0("RCD_SCRIPTS must not be registered in the PLIST. Please use the RCD_SCRIPTS framework.")
		return
	}

	pline.line.error0("Configuration files must not be registered in the PLIST. " +
		"Please use the CONF_FILES framework, which is described in mk/pkginstall/bsd.pkginstall.mk.")
}

func (ck *PlistChecker) checkpathInfo(pline *PlistLine, dirname, basename string) {
	if pline.text == "info/dir" {
		pline.line.error0("\"info/dir\" must not be listed. Use install-info to add/remove an entry.")
		return
	}

	if G.pkg != nil && G.pkg.vardef["INFO_FILES"] == nil {
		pline.line.warn0("Packages that install info files should set INFO_FILES.")
	}
}

func (ck *PlistChecker) checkpathLib(pline *PlistLine, dirname, basename string) {
	switch {
	case G.pkg != nil && G.pkg.effectivePkgbase != "" && hasPrefix(pline.text, "lib/"+G.pkg.effectivePkgbase+"/"):
		return

	case hasPrefix(pline.text, "lib/locale/"):
		pline.line.error0("\"lib/locale\" must not be listed. Use ${PKGLOCALEDIR}/locale and set USE_PKGLOCALEDIR instead.")
		return
	}

	switch ext := path.Ext(basename); ext {
	case ".a", ".la", ".so":
		if G.opts.WarnExtra && dirname == "lib" && !hasPrefix(basename, "lib") {
			pline.line.warn1("Library filename %q should start with \"lib\".", basename)
		}
		if ext == "la" {
			if G.pkg != nil && G.pkg.vardef["USE_LIBTOOL"] == nil {
				pline.line.warn0("Packages that install libtool libraries should define USE_LIBTOOL.")
			}
		}
	}

	if strings.Contains(basename, ".a") || strings.Contains(basename, ".so") {
		if m, noext := match1(pline.text, `^(.*)(?:\.a|\.so[0-9.]*)$`); m {
			if laLine := ck.allFiles[noext+".la"]; laLine != nil {
				pline.line.warnf("Redundant library found. The libtool library is in line %d.", laLine.line.firstLine)
			}
		}
	}
}

func (ck *PlistChecker) checkpathMan(pline *PlistLine) {
	line := pline.line

	m, catOrMan, section, manpage, ext, gz := match5(pline.text, `^man/(cat|man)(\w+)/(.*?)\.(\w+)(\.gz)?$`)
	if !m {
		// maybe: line.warn1("Invalid filename %q for manual page.", text)
		return
	}

	if !matches(section, `^[\dln]$`) {
		line.warn1("Unknown section %q for manual page.", section)
	}

	if catOrMan == "cat" && ck.allFiles["man/man"+section+"/"+manpage+"."+section] == nil {
		line.warn0("Preformatted manual page without unformatted one.")
	}

	if catOrMan == "cat" {
		if ext != "0" {
			line.warn0("Preformatted manual pages should end in \".0\".")
		}
	} else {
		if section != ext {
			line.warn2("Mismatch between the section (%s) and extension (%s) of the manual page.", section, ext)
		}
	}

	if gz != "" {
		line.note0("The .gz extension is unnecessary for manual pages.")
		explain(
			"Whether the manual pages are installed in compressed form or not is",
			"configured by the pkgsrc user. Compression and decompression takes place",
			"automatically, no matter if the .gz extension is mentioned in the PLIST",
			"or not.")
	}
}

func (ck *PlistChecker) checkpathSbin(pline *PlistLine) {
	binname := pline.text[5:]

	if ck.allFiles["man/man8/"+binname+".8"] == nil && G.opts.WarnExtra {
		pline.line.warn1("Manual page missing for sbin/%s.", binname)
		explain(
			"All programs that can be run directly by the user should have a manual",
			"page for quick reference. The programs in the sbin/ directory should have",
			"corresponding manual pages in section 8 (filename program.8), not in",
			"section 1.")
	}
}

func (ck *PlistChecker) checkpathShare(pline *PlistLine) {
	line, text := pline.line, pline.text
	switch {
	case hasPrefix(text, "share/applications/") && hasSuffix(text, ".desktop"):
		f := "../../sysutils/desktop-file-utils/desktopdb.mk"
		if G.pkg != nil && G.pkg.included[f] == nil {
			line.warn1("Packages that install a .desktop entry should .include %q.", f)
			explain2(
				"If *.desktop files contain MimeType keys, the global MIME type registry",
				"must be updated by desktop-file-utils. Otherwise, this warning is harmless.")
		}

	case hasPrefix(text, "share/icons/hicolor/") && G.pkg != nil && G.pkg.pkgpath != "graphics/hicolor-icon-theme":
		f := "../../graphics/hicolor-icon-theme/buildlink3.mk"
		if G.pkg.included[f] == nil {
			line.error1("Packages that install hicolor icons must include %q in the Makefile.", f)
		}

	case hasPrefix(text, "share/icons/gnome") && G.pkg != nil && G.pkg.pkgpath != "graphics/gnome-icon-theme":
		f := "../../graphics/gnome-icon-theme/buildlink3.mk"
		if G.pkg.included[f] == nil {
			line.error1("The package Makefile must include %q.", f)
			explain1(
				"Packages that install GNOME icons must maintain the icon theme cache.")
		}

	case hasPrefix(text, "share/doc/html/"):
		if G.opts.WarnPlistDepr {
			line.warn0("Use of \"share/doc/html\" is deprecated. Use \"share/doc/${PKGBASE}\" instead.")
		}

	case G.pkg != nil && G.pkg.effectivePkgbase != "" && (hasPrefix(text, "share/doc/"+G.pkg.effectivePkgbase+"/") ||
		hasPrefix(text, "share/examples/"+G.pkg.effectivePkgbase+"/")):
		// Fine.

	case text == "share/icons/hicolor/icon-theme.cache" && G.pkg != nil && G.pkg.pkgpath != "graphics/hicolor-icon-theme":
		line.error0("This file must not appear in any PLIST file.")
		explain3(
			"Remove this line and add the following line to the package Makefile.",
			"",
			".include \"../../graphics/hicolor-icon-theme/buildlink3.mk\"")

	case hasPrefix(text, "share/info/"):
		line.warn0("Info pages should be installed into info/, not share/info/.")
		explain1(
			"To fix this, you should add INFO_FILES=yes to the package Makefile.")

	case hasPrefix(text, "share/locale/") && hasSuffix(text, ".mo"):
		// Fine.

	case hasPrefix(text, "share/man/"):
		line.warn0("Man pages should be installed into man/, not share/man/.")
	}
}

func (pline *PlistLine) checkTrailingWhitespace() {
	if hasSuffix(pline.text, " ") || hasSuffix(pline.text, "\t") {
		pline.line.error0("pkgsrc does not support filenames ending in white-space.")
		explain1(
			"Each character in the PLIST is relevant, even trailing white-space.")
	}
}

func (pline *PlistLine) checkDirective(cmd, arg string) {
	line := pline.line

	if cmd == "unexec" {
		if m, arg := match1(arg, `^(?:rmdir|\$\{RMDIR\} \%D/)(.*)`); m {
			if !strings.Contains(arg, "true") && !strings.Contains(arg, "${TRUE}") {
				pline.line.warn0("Please remove this line. It is no longer necessary.")
			}
		}
	}

	switch cmd {
	case "exec", "unexec":
		switch {
		case strings.Contains(arg, "install-info"),
			strings.Contains(arg, "${INSTALL_INFO}"):
			line.warn0("@exec/unexec install-info is deprecated.")
		case strings.Contains(arg, "ldconfig") && !strings.Contains(arg, "/usr/bin/true"):
			pline.line.error0("ldconfig must be used with \"||/usr/bin/true\".")
		}

	case "comment":
		// Nothing to do.

	case "dirrm":
		line.warn0("@dirrm is obsolete. Please remove this line.")
		explain3(
			"Directories are removed automatically when they are empty.",
			"When a package needs an empty directory, it can use the @pkgdir",
			"command in the PLIST")

	case "imake-man":
		args := splitOnSpace(arg)
		switch {
		case len(args) != 3:
			line.warn0("Invalid number of arguments for imake-man.")
		case args[2] == "${IMAKE_MANNEWSUFFIX}":
			pline.warnAboutPlistImakeMannewsuffix()
		}

	case "pkgdir":
		// Nothing to check.

	default:
		line.warn1("Unknown PLIST directive \"@%s\".", cmd)
	}
}

func (pline *PlistLine) warnAboutPlistImakeMannewsuffix() {
	pline.line.warn0("IMAKE_MANNEWSUFFIX is not meant to appear in PLISTs.")
	explain(
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
