//
// Parsing.
//

// Checks whether tree matches pattern, and if so, instanciates the
// variables in pattern. If they don't match, some variables may be
// instanciated nevertheless, but the exact behavior is unspecified.
//
sub tree_match($$)
sub tree_match($$) {
	my (tree, pattern) = @_

	my d1 = Data::Dumper.new([tree, pattern]).Terse(true).Indent(0)
	my d2 = Data::Dumper.new([pattern]).Terse(true).Indent(0)
	opt_debug_trace and logDebug(NO_FILE, NO_LINES, sprintf("tree_match(%s, %s)", d1.Dump, d2.Dump))

	return true if (!defined(tree) && !defined(pattern))
	return false if (!defined(tree) || !defined(pattern))
	my aref = ref(tree)
	my pref = ref(pattern)
	if (pref == "SCALAR" && !defined($pattern)) {
		$pattern = tree
		return true
	}
	if (aref == "" && (pref == "" || pref == "SCALAR")) {
		return tree == pattern
	}
	if (aref == "ARRAY" && pref == "ARRAY") {
		return false if scalar(@tree) != scalar(@pattern)
		for (my i = 0; i < scalar(@tree); i++) {
			return false unless tree_match(tree.[i], pattern.[i])
		}
		return true
	}
	return false
}

// TODO: Needs to be redesigned to handle more complex expressions.
sub parse_mk_cond($$)
sub parse_mk_cond($$) {
	my (line, cond) = @_

	opt_debug_trace and line.logDebug("parse_mk_cond(\"${cond}\")")

	my re_simple_varname = qr"[A-Z_][A-Z0-9_]*(?:\.[\w_+\-]+)?"
	while (cond != "") {
		if (cond =~ s/^!//) {
			return ["not", parse_mk_cond(line, cond)]
		} else if (cond =~ s/^defined\((${re_simple_varname})\)$//) {
			return ["defined", 1]
		} else if (cond =~ s/^empty\((${re_simple_varname})\)$//) {
			return ["empty", 1]
		} else if (cond =~ s/^empty\((${re_simple_varname}):M([^\$:{})]+)\)$//) {
			return ["empty", ["match", 1, 2]]
		} else if (cond =~ s/^\$\{(${re_simple_varname})\}\s+(==|!=)\s+"([^"\$\\]*)"$//) { #"
			return [2, ["var", 1], ["string", 3]]
		} else {
			opt_debug_unchecked and line.logDebug("parse_mk_cond: ${cond}")
			return ["unknown", cond]
		}
	}
}

// The bmake parser is way too sloppy about syntax, so we need to check
// that here.
//
sub checkline_mk_cond($$) {
	my (line, cond) = @_
	my (op, varname, match, value)

	opt_debug_trace and line.logDebug("checkline_mk_cond(cond)")
	my tree = parse_mk_cond(line, cond)
	if (tree_match(tree, ["not", ["empty", ["match", \varname, \match]]])) {
		//line.logNote("tree_match: varname=varname, match=match")

		my type = get_variable_type(line, varname)
		my btype = defined(type) ? type.basic_type : undef
		if (defined(btype) && ref(type.basic_type) == "HASH") {
			if (match !~ `[\$\[*]` && !exists(btype.{match})) {
				line.logWarning("Invalid :M value \"match\". Only { " . join(" ", sort keys %btype) . " } are allowed.")
			}
		}

		// Currently disabled because the valid options can also be defined in PKG_OPTIONS_GROUP.*.
		// Additionally, all these variables may have multiple assigments (+=).
		if (false && varname == "PKG_OPTIONS" && defined(pkgctx_vardef) && exists(pkgctx_vardef.{"PKG_SUPPORTED_OPTIONS"})) {
			my options = pkgctx_vardef.{"PKG_SUPPORTED_OPTIONS"}.get("value")

			if (match !~ `[\$\[*]` && index(" options ", " match ") == -1) {
				line.logWarning("Invalid option \"match\". Only { options } are allowed.")
			}
		}

		// TODO: PKG_BUILD_OPTIONS. That requires loading the
		// complete package definitition for the package "pkgbase"
		// or some other database. If we could confine all option
		// definitions to options.mk, this would become easier.

	} else if (tree_match(tree, [\op, ["var", \varname], ["string", \value]])) {
		checkline_mk_vartype(line, varname, "use", value, undef)

	}
	// XXX: When adding new cases, be careful that the variables may have
	// been partially initialized by previous calls to tree_match.
	// XXX: Maybe it is better to clear these references at the beginning
	// of tree_match.
}

sub checkfile_mk($) {
	my (fname) = @_

	opt_debug_trace and logDebug(fname, NO_LINES, "checkfile_mk()")

	checkperms(fname)
	my lines = load_lines(fname, true) or return logError(fname, NO_LINE_NUMBER, "Cannot be read.")

	parselines_mk(lines)
	checklines_mk(lines)
	autofix(lines)
}

sub checkfile_package_Makefile($$) {
	my (fname, lines) = @_

	opt_debug_trace and logDebug(fname, NO_LINES, "checkfile_package_Makefile(..., ...)")

	checkperms(fname)

	if (!exists(pkgctx_vardef.{"PLIST_SRC"})
		&& !exists(pkgctx_vardef.{"GENERATE_PLIST"})
		&& !exists(pkgctx_vardef.{"META_PACKAGE"})
		&& defined(pkgdir)
		&& !-f "${current_dir}/pkgdir/PLIST"
		&& !-f "${current_dir}/pkgdir/PLIST.common") {
		logWarning(fname, NO_LINE_NUMBER, "Neither PLIST nor PLIST.common exist, and PLIST_SRC is unset. Are you sure PLIST handling is ok?")
	}

	if ((exists(pkgctx_vardef.{"NO_CHECKSUM"}) || pkgctx_vardef.{"META_PACKAGE"}) && is_emptydir("${current_dir}/${patchdir}")) {
		if (-f "${current_dir}/${distinfo_file}") {
			logWarning("${current_dir}/${distinfo_file}", NO_LINE_NUMBER, "This file should not exist if NO_CHECKSUM or META_PACKAGE is set.")
		}
	} else {
		if (!-f "${current_dir}/${distinfo_file}") {
			logWarning("${current_dir}/${distinfo_file}", NO_LINE_NUMBER, "File not found. Please run '".conf_make." makesum'.")
		}
	}

	if (exists(pkgctx_vardef.{"REPLACE_PERL"}) && exists(pkgctx_vardef.{"NO_CONFIGURE"})) {
		pkgctx_vardef.{"REPLACE_PERL"}.logWarning("REPLACE_PERL is ignored when ...")
		pkgctx_vardef.{"NO_CONFIGURE"}.logWarning("... NO_CONFIGURE is set.")
	}

	if (!exists(pkgctx_vardef.{"LICENSE"})) {
		logError(fname, NO_LINE_NUMBER, "Each package must define its LICENSE.")
	}

	if (exists(pkgctx_vardef.{"GNU_CONFIGURE"}) && exists(pkgctx_vardef.{"USE_LANGUAGES"})) {
		my languages_line = pkgctx_vardef.{"USE_LANGUAGES"}
		my value = languages_line.get("value")

		if (languages_line.has("comment") && languages_line.get("comment") =~ `\b(?:c|empty|none)\b`i) {
			// Don't emit a warning, since the comment
			// probably contains a statement that C is
			// really not needed.

		} else if (value !~ `(?:^|\s+)(?:c|c99|objc)(?:\s+|$)`) {
			pkgctx_vardef.{"GNU_CONFIGURE"}.logWarning("GNU_CONFIGURE almost always needs a C compiler, ...")
			languages_line.logWarning("... but \"c\" is not added to USE_LANGUAGES.")
		}
	}

	my distname_line = pkgctx_vardef.{"DISTNAME"}
	my pkgname_line = pkgctx_vardef.{"PKGNAME"}

	my distname = defined(distname_line) ? distname_line.get("value") : undef
	my pkgname = defined(pkgname_line) ? pkgname_line.get("value") : undef
	my nbpart = get_nbpart()

	// Let's do some tricks to get the proper value of the package
	// name more often.
	if (defined(distname) && defined(pkgname)) {
		pkgname =~ s/\$\{DISTNAME\}/distname/

		if (pkgname =~ `^(.*)\$\{DISTNAME:S(.)([^:]*)\2([^:]*)\2(g?)\}(.*)$`) {
			my (before, separator, old, new, mod, after) = (1, 2, 3, 4, 5, 6)
			my newname = distname
			old = quotemeta(old)
			old =~ s/^\\\^/^/
			old =~ s/\\\$$/\$/
			if (mod == "g") {
				newname =~ s/old/new/g
			} else {
				newname =~ s/old/new/
			}
			opt_debug_misc and pkgname_line.logDebug("old pkgname=pkgname")
			pkgname = before . newname . after
			opt_debug_misc and pkgname_line.logDebug("new pkgname=pkgname")
		}
	}

	if (defined(pkgname) && defined(distname) && pkgname == distname) {
		pkgname_line.logNote("PKGNAME is \${DISTNAME} by default. You probably don't need to define PKGNAME.")
	}

	if (!defined(pkgname) && defined(distname) && distname !~ regex_unresolved && distname !~ regex_pkgname) {
		distname_line.logWarning("As DISTNAME is not a valid package name, please define the PKGNAME explicitly.")
	}

	(effective_pkgname, effective_pkgname_line, effective_pkgbase, effective_pkgversion)
		= (defined(pkgname) && pkgname !~ regex_unresolved && pkgname =~ regex_pkgname) ? (pkgname.nbpart, pkgname_line, 1, 2)
		: (defined(distname) && distname !~ regex_unresolved && distname =~ regex_pkgname) ? (distname.nbpart, distname_line, 1, 2)
		: (undef, undef, undef, undef)
	if (defined(effective_pkgname_line)) {
		opt_debug_misc and effective_pkgname_line.logDebug("Effective name=${effective_pkgname} base=${effective_pkgbase} version=${effective_pkgversion}.")
		// XXX: too many false positives
		if (false && pkgpath =~ `/([^/]+)$` && effective_pkgbase != 1) {
			effective_pkgname_line.logWarning("Mismatch between PKGNAME (effective_pkgname) and package directory (1).")
		}
	}

	checkpackage_possible_downgrade()

	if (!exists(pkgctx_vardef.{"COMMENT"})) {
		logWarning(fname, NO_LINE_NUMBER, "No COMMENT given.")
	}

	if (exists(pkgctx_vardef.{"USE_IMAKE"}) && exists(pkgctx_vardef.{"USE_X11"})) {
		pkgctx_vardef.{"USE_IMAKE"}.logNote("USE_IMAKE makes ...")
		pkgctx_vardef.{"USE_X11"}.logNote("... USE_X11 superfluous.")
	}

	if (defined(effective_pkgbase)) {

		foreach my suggested_update (@{get_suggested_package_updates()}) {
			my (line, suggbase, suggver, suggcomm) = @{suggested_update}
			my comment = (defined(suggcomm) ? " (${suggcomm})" : "")

			next unless effective_pkgbase == suggbase

			if (dewey_cmp(effective_pkgversion, "<", suggver)) {
				effective_pkgname_line.logWarning("This package should be updated to ${suggver}${comment}.")
				effective_pkgname_line.explainWarning(
"The wishlist for package updates in doc/TODO mentions that a newer",
"version of this package is available.")
			}
			if (dewey_cmp(effective_pkgversion, "==", suggver)) {
				effective_pkgname_line.logNote("The update request to ${suggver} from doc/TODO${comment} has been done.")
			}
			if (dewey_cmp(effective_pkgversion, ">", suggver)) {
				effective_pkgname_line.logNote("This package is newer than the update request to ${suggver}${comment}.")
			}
		}
	}

	checklines_mk(lines)
	checklines_package_Makefile_varorder(lines)
	autofix(lines)
}


sub checkfile($) {
	my (fname) = @_
	my (st, basename)

	opt_debug_trace and logDebug(fname, NO_LINES, "checkfile()")

	basename = basename(fname)
	if (basename =~ `^(?:work.*|.*~|.*\.orig|.*\.rej)$`) {
		if (opt_import) {
			logError(fname, NO_LINE_NUMBER, "Must be cleaned up before committing the package.")
		}
		return
	}

	if (!(st = lstat(fname))) {
		logError(fname, NO_LINE_NUMBER, "$!")
		return
	}
	if (S_ISDIR(st.mode)) {
		if (basename == "files" || basename == "patches" || basename == "CVS") {
			// Ok
		} else if (fname =~ `(?:^|/)files/[^/]*$`) {
			// Ok

		} else if (!is_emptydir(fname)) {
			logWarning(fname, NO_LINE_NUMBER, "Unknown directory name.")
		}

	} else if (S_ISLNK(st.mode)) {
		if (basename !~ `^work`) {
			logWarning(fname, NO_LINE_NUMBER, "Unknown symlink name.")
		}

	} else if (!S_ISREG(st.mode)) {
		logError(fname, NO_LINE_NUMBER, "Only files and directories are allowed in pkgsrc.")

	} else if (basename == "ALTERNATIVES") {
		opt_check_ALTERNATIVES and checkfile_ALTERNATIVES(fname)

	} else if (basename == "buildlink3.mk") {
		opt_check_bl3 and checkfile_buildlink3_mk(fname)

	} else if (basename =~ `^DESCR`) {
		opt_check_DESCR and checkfile_DESCR(fname)

	} else if (basename =~ `^distinfo`) {
		opt_check_distinfo and checkfile_distinfo(fname)

	} else if (basename == "DEINSTALL" || basename == "INSTALL") {
		opt_check_INSTALL and checkfile_INSTALL(fname)

	} else if (basename =~ `^MESSAGE`) {
		opt_check_MESSAGE and checkfile_MESSAGE(fname)

	} else if (basename =~ `^patch-[-A-Za-z0-9_.~+]*[A-Za-z0-9_]$`) {
		opt_check_patches and checkfile_patch(fname)

	} else if (fname =~ `(?:^|/)patches/manual[^/]*$`) {
		opt_debug_unchecked and logDebug(fname, NO_LINE_NUMBER, "Unchecked file \"${fname}\".")

	} else if (fname =~ `(?:^|/)patches/[^/]*$`) {
		logWarning(fname, NO_LINE_NUMBER, "Patch files should be named \"patch-\", followed by letters, '-', '_', '.', and digits only.")

	} else if (basename =~ `^(?:.*\.mk|Makefile.*)$` and not fname =~ m,files/, and not fname =~ m,patches/,) {
		opt_check_mk and checkfile_mk(fname)

	} else if (basename =~ `^PLIST`) {
		opt_check_PLIST and checkfile_PLIST(fname)

	} else if (basename == "TODO" || basename == "README") {
		// Ok

	} else if (basename =~ `^CHANGES-.*`) {
		load_doc_CHANGES(fname)

	} else if (!-T fname) {
		logWarning(fname, NO_LINE_NUMBER, "Unexpectedly found a binary file.")

	} else if (fname =~ `(?:^|/)files/[^/]*$`) {
		// Ok
	} else {
		logWarning(fname, NO_LINE_NUMBER, "Unexpected file found.")
		opt_check_extra and checkfile_extra(fname)
	}
}

sub my_split($$) {
	my (delimiter, s) = @_
	my (pos, next, @result)

	pos = 0
	for (pos = 0; pos != -1; pos = next) {
		next = index(s, delimiter, pos)
		push @result, ((next == -1) ? substr(s, pos) : substr(s, pos, next - pos))
		if (next != -1) {
			next += length(delimiter)
		}
	}
	return @result
}

// Checks that the files in the directory are in sync with CVS's status.
//
sub checkdir_CVS($) {
	my (fname) = @_

	my cvs_entries = load_file("fname/CVS/Entries")
	my cvs_entries_log = load_file("fname/CVS/Entries.Log")
	return unless cvs_entries

	foreach my line (@cvs_entries) {
		my (type, fname, mtime, date, keyword_mode, tag, undef) = my_split("/", line.text)
		next if (type == "D" && !defined(fname))
		assert(false, "Unknown line format: " . line.text)
			unless type == "" || type == "D"
		assert(false, "Unknown line format: " . line.text)
			unless defined(tag)
		assert(false, "Unknown line format: " . line.text)
			unless defined(keyword_mode)
		assert(false, "Unknown line format: " . line.text)
			if defined(undef)
	}
}
