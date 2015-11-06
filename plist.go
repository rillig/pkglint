package main

import (
	"path"
	"regexp"
	"strings"
)

func checkfilePlist(fname string) {
	_ = GlobalVars.opts.optDebugTrace && logDebugF(fname, NO_LINES, "checkfilePlist")

	checkperms(fname)
	lines, err := loadLines(fname, false)
	if err != nil {
		logErrorF(fname, NO_LINES, "Cannot be read.")
		return
	}

	if len(lines) == 0 {
		logErrorF(fname, NO_LINES, "Must not be empty.")
		return
	}
	checklineRcsid(lines[0], "@comment ")

	if len(lines) == 1 {
		lines[0].logWarningF("PLIST files shouldn't be empty.")
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

	allFiles := make(map[string]*Line)
	allDirs := make(map[string]*Line)
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

		if strings.HasPrefix(text, "${") {
			if m, varname, rest := match2(text, `^\$\{([\w_]+)\}(.*)`); m {
				if GlobalVars.pkgContext.plistSubstCond[varname] {
					_ = GlobalVars.opts.optDebugMisc && line.logDebugF("Removed PLIST_SUBST conditional %q.", varname)
					text = rest
				}
			}
		}

		if match(text, `^[\w$]`) != nil {
			allFiles[text] = line
			for dir := path.Dir(text); dir != "."; dir = path.Dir(dir) {
				allDirs[dir] = line
			}
		}

		if strings.HasPrefix(text, "@") {
			if m, dirname := match1(text, `^\@exec \$\{MKDIR\} %D/(.*)$`); m {
				for dir := dirname; dir != "."; dir = path.Dir(dir) {
					allDirs[dir] = line
				}
			}
		}
	}

	for _, line := range lines {
		pline := &PlistLine{line}
		pline.checkTrailingWhitespace()

		if m, cmd, arg := match2(text, `^(?:\$\{[\w_]+\})?\@([a-z-]+)\s+(.*)`); m {
			pline.checkDirective(cmd, arg)
		} else if m, dirname, basename := match2(text, `^([A-Za-z0-9\$].*)/([^/]+)$`); m {
			pline.checkPathname(dirname, basename)
		} else if match0(text, `^\$\{[\w_]+\}$`) {
			// A variable on its own line.
		}		else {
			line.logWarningF("Unknown line type.")
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

	if match(line.text, `\s$`) != nil {
		line.logErrorF("pkgsrc does not support filenames ending in white-space.")
		line.explainError(
			"Each character in the PLIST is relevant, even trailing white-space.")
	}
}

func (pline *PlistLine) checkDirective(cmd, arg string) {
	line := pline.line
	text := line.text

	if cmd == "unexec" {
		if m, arg := match1(arg, `^(?:rmdir|\$\{RMDIR\} \%D/)(.*)`); m {
			if !strings.Contains(arg, "true") && !strings.Contains(arg, "${TRUE}") {
				line.logWarningF("Please remove this line. It is no longer necessary.")
			}
		}
	}

	switch cmd {
	case "exec", "unexec":
		switch {
		case strings.Contains(arg, "install-info"),
			strings.Contains(arg, "${INSTALL_INFO}"):
			line.logWarningF("@exec/unexec install-info is deprecated.")
		case strings.Contains(arg, "ldconfig") && !strings.Contains(arg, "/usr/bin/true"):
			line.logErrorF("ldconfig must be used with \"||/usr/bin/true\".")
		}

	case "comment":
		// Nothing to do.

	case "dirrm":
		line.logWarningF("@dirrm is obsolete. Please remove this line.")
		line.explainWarning(
			"Directories are removed automatically when they are empty.",
			"When a package needs an empty directory, it can use the @pkgdir",
			"command in the PLIST")

	case "imake-man":
		args := regexp.MustCompile(`\s+`).Split(arg, -1)
		if len(args) != 3 {
			line.logWarningF("Invalid number of arguments for imake-man.")
		} else {
			if args[2] == "${IMAKE_MANNEWSUFFIX}" {
				pline.warnAboutPlistImakeMannewsuffix()
			}
		}

	case "pkgdir":
		// Nothing to check.

	default:
		line.logWarningF("Unknown PLIST directive \"@%s\".", cmd)
	}
}

func (pline *PlistLine) checkPathname(dirname, basename string, lastFileSeen *string) {
	line := pline.line
	text := line.text
	
	if GlobalVars.opts.optWarnPlistSort && match0(text, `^\w`) && !match0(text, reUnresolvedVar) {
		if lastFileSeen != "" {
			if *lastFileSeen > text {
				line.logWarningF("%q should be sorted before %q.", text, lastFileSeen)
				line.explainWarning(
					"For aesthetic reasons, the files in the PLIST should be sorted",
"alphabetically.")
			} else if *lastFileSeen == text {
				line.logError("Duplicate filename.")
			}	
		}
		*lastFileSeen = text
	}

	if strings.Contains(basename, "${IMAKE_MANNEWSUFFIX}") {
		pline.warnAboutPlistImakeMannewsuffix()
	}

	switch {	
	case strings.HasPrefix(dirname, "bin/"):
		line.logWarningF("The bin/ directory should not have subdirectories.")

	case dirname == "bin":
		switch {
		case allFiles["man/man1/" + basename + ".1"] != nil:
		case allFiles["man/man6/" + basename + ".6"] != nil:
		case allFiles["${IMAKE_MAN_DIR}/" + basename + ".${IMAKE_MANNEWSUFFIX}"}] != nil:
		default:
			if GlobalVars.opts.optWarnExtra {
				line.logWarningF("Manual page missing for bin/${basename}.")
				line.explainWarning(
"All programs that can be run directly by the user should have a manual",
"page for quick reference. The programs in the bin/ directory should have",
"corresponding manual pages in section 1 (filename program.1), not in",
"section 8.")
		}
	
	case strings.HasPrefix(text, "doc/"):
		line.logErrorF("Documentation must be installed under share/doc, not doc.")

	case strings.HasPrefix(text, "etc/rc.d/":
		line.logErrorF("RCD_SCRIPTS must not be registered in the PLIST. Please use the RCD_SCRIPTS framework.")

	case strings.HasPrefix(text, "etc/":
		f := "mk/pkginstall/bsd.pkginstall.mk"
		line.logErrorF("Configuration files must not be registered in the PLIST. " +
			"Please use the CONF_FILES framework, which is described in %s.", f)

	case strings.HasPrefix(text, "include/") && match0(text, `^include/.*\.(?:h|hpp)$`):
		// Fine.

	case text == "info/dir":
		line.logErrorF("\"info/dir\" must not be listed. Use install-info to add/remove an entry.")

	case strings.HasPrefix(text, "info/" && length($text) > 5:
		if GlobalVars.pkgContext.vardef["INFO_FILES"] == nil {
			line.logWarningF("Packages that install info files should set INFO_FILES.")
		}
		
	case GlobalVars.pkgContext.effective_pkgbase != nil && strings.HasPrefix(text, "lib/" + GlobalVars.pkgContext.effective_pkgbase + "/") {
		// Fine.

	case strings.HasPrefix(text, "lib/locale/":
		line.logErrorF("\"lib/locale\" must not be listed. Use ${PKGLOCALEDIR}/locale and set USE_PKGLOCALEDIR instead.")

	case strings.HasPrefix(text, "lib/"):
		if m, dir, lib, ext := match3(text, `^(lib/(?:.*/)*)([^/]+)\.(so|a|la)$`); m {
			if dir == "lib/" && !strings.HasPrefix(lib, "lib") {
				_ = GlobalVars.opts.optWarnExtra && line.logWarningF("Library filename does not start with \"lib\".")
			}
			if ext=="la"{
				if GlobalVars.pkgContext.vardef["USE_LIBTOOL"] == nil {
					line.logWarningF("Packages that install libtool libraries should define USE_LIBTOOL.")
				}
}
			}
		
	case strings.HasPrefix(text, "man/"):
		if m, catOrMan, section, manpage, ext, gz := match5(text, `^man/(cat|man)(\w+)/(.*?)\.(\w+)(\.gz)?$`) {

				if (!match0(section, `^[\dln]$`) {
					line.logWarningF("Unknown section %q for manual page.", section)
				}

				if (catOrMan == "cat" CONT_HERE && !exists($all_files->{"man/man${section}/${manpage}.${section}"})) {
					line.logWarningF("Preformatted manual page without unformatted one.");
				}

				if ($cat_or_man eq "cat") {
					if ($ext ne "0") {
						line.logWarningF("Preformatted manual pages should end in \".0\".");
					}
				} else {
					if ($section ne $ext) {
						line.logWarningF("Mismatch between the section (${section}) and extension (${ext}) of the manual page.");
					}
				}

				if (defined($gz)) {
					$line->log_note("The .gz extension is unnecessary for manual pages.");
					$line->explain_note(
"Whether the manual pages are installed in compressed form or not is",
"configured by the pkgsrc user. Compression and decompression takes place",
"automatically, no matter if the .gz extension is mentioned in the PLIST",
"or not.");
				}

			} elsif (substr($text, 0, 7) eq "man/cat") {
				line.logWarningF("Invalid filename \"${text}\" for preformatted manual page.");

			} elsif (substr($text, 0, 7) eq "man/man") {
				line.logWarningF("Invalid filename \"${text}\" for unformatted manual page.");

			} elsif (substr($text, 0, 5) eq "sbin/") {
				my $binname = substr($text, 5);

				if (!exists($all_files->{"man/man8/${binname}.8"})) {
					$opt_warn_extra and line.logWarningF("Manual page missing for sbin/${binname}.");
					$opt_warn_extra and $line->explain_warning(
"All programs that can be run directly by the user should have a manual",
"page for quick reference. The programs in the sbin/ directory should have",
"corresponding manual pages in section 8 (filename program.8), not in",
"section 1.");
				}

			} elsif (substr($text, 0, 6) eq "share/" && $text =~ m"^share/applications/.*\.desktop$") {
				my $f = "../../sysutils/desktop-file-utils/desktopdb.mk";
				if (defined($pkgctx_included) && !exists($pkgctx_included->{$f})) {
					line.logWarningF("Packages that install a .desktop entry may .include \"$f\".");
					$line->explain_warning(
"If *.desktop files contain MimeType keys, global MIME Type registory DB",
"must be updated by desktop-file-utils.",
"Otherwise, this warning is harmless.");
				}

			} elsif (substr($text, 0, 6) eq "share/" && $pkgpath ne "graphics/hicolor-icon-theme" && $text =~ m"^share/icons/hicolor(?:$|/)") {
				my $f = "../../graphics/hicolor-icon-theme/buildlink3.mk";
				if (defined($pkgctx_included) && !exists($pkgctx_included->{$f})) {
					$line->log_error("Please .include \"$f\" in the Makefile");
					$line->explain_error(
"If hicolor icon themes are installed, icon theme cache must be",
"maintained. The hicolor-icon-theme package takes care of that.");
				}

			} elsif (substr($text, 0, 6) eq "share/" && $pkgpath ne "graphics/gnome-icon-theme" && $text =~ m"^share/icons/gnome(?:$|/)") {
				my $f = "../../graphics/gnome-icon-theme/buildlink3.mk";
				if (defined($pkgctx_included) && !exists($pkgctx_included->{$f})) {
					$line->log_error("Please .include \"$f\"");
					$line->explain_error(
"If Gnome icon themes are installed, icon theme cache must be maintained.");
				}
			} elsif ($dirname eq "share/aclocal" && $basename =~ m"\.m4$") {
				# Fine.

			} elsif (substr($text, 0, 15) eq "share/doc/html/") {
				$opt_warn_plist_depr and line.logWarningF("Use of \"share/doc/html\" is deprecated. Use \"share/doc/\${PKGBASE}\" instead.");

			} elsif (defined($effective_pkgbase) && $text =~ m"^share/(?:doc/|examples/|)\Q${effective_pkgbase}\E/") {
				# Fine.

			} elsif ($pkgpath ne "graphics/hicolor-icon-theme" && $text =~ m"^share/icons/hicolor/icon-theme\.cache") {
				$line->log_error("Please .include \"../../graphics/hicolor-icon-theme/buildlink3.mk\" and remove this line.");

			} elsif (substr($text, 0, 11) eq "share/info/") {
				line.logWarningF("Info pages should be installed into info/, not share/info/.");
				$line->explain_warning(
"To fix this, you should add INFO_FILES=yes to the package Makefile.");

			} elsif (substr($text, -3) eq ".mo" && $text =~ m"^share/locale/[\w\@_]+/LC_MESSAGES/[^/]+\.mo$") {
				# Fine.

			} elsif (substr($text, 0, 10) eq "share/man/") {
				line.logWarningF("Man pages should be installed into man/, not share/man/.");

			} else {
				$opt_debug_unchecked and $line->log_debug("Unchecked pathname \"${text}\".");
			}

			if ($text =~ /\$\{PKGLOCALEDIR}/ && defined($pkgctx_vardef) && !exists($pkgctx_vardef->{"USE_PKGLOCALEDIR"})) {
				line.logWarningF("PLIST contains \${PKGLOCALEDIR}, but USE_PKGLOCALEDIR was not found.");
			}

			if (index($text, "/CVS/") != -1) {
				line.logWarningF("CVS files should not be in the PLIST.");
			}
			if (substr($text, -5) eq ".orig") {
				line.logWarningF(".orig files should not be in the PLIST.");
			}
			if (substr($text, -14) eq "/perllocal.pod") {
				line.logWarningF("perllocal.pod files should not be in the PLIST.");
				$line->explain_warning(
"This file is handled automatically by the INSTALL/DEINSTALL scripts,",
"since its contents changes frequently.");
			}

			if ($text =~ m"^(.*)(\.a|\.so[0-9.]*)$") {
				my ($basename, $ext) = ($1, $2);

				if (exists($all_files->{"${basename}.la"})) {
					line.logWarningF("Redundant library found. The libtool library is in line " . $all_files->{"${basename}.la"}->lines . ".");
				}
			}

-------------	
	
}

func (pline *PlistLine) warnAboutPlistImakeMannewsuffix() {
	line := pline.line

	line.logWarningF("IMAKE_MANNEWSUFFIX is not meant to appear in PLISTs.")
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
