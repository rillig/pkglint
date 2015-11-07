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

sub parse_licenses($) {
	my (licenses) = @_

	licenses =~ s,\$\{PERL5_LICENSE},gnu-gpl-v2 OR artistic,g
	licenses =~ s,[()]|AND|OR,,g; # XXX: treats OR like AND
	my @licenses = split(/\s+/, licenses)
	return \@licenses
}

//
// Subroutines to check a single line.
//

sub checkline_relative_pkgdir($$) {
	my (line, path) = @_

	checkline_relative_path(line, path, true)
	path = resolve_relative_path(path, false)

	if (path =~ `^(?:\./)?\.\./\.\./([^/]+/[^/]+)$`) {
		my otherpkgpath = 1
		if (! -f "cwd_pkgsrcdir/otherpkgpath/Makefile") {
			line.logError("There is no package in otherpkgpath.")
		}

	} else {
		line.logWarning("\"${path}\" is not a valid relative package directory.")
		line.explainWarning(
"A relative pathname always starts with \"../../\", followed",
"by a category, a slash and a the directory name of the package.",
"For example, \"../../misc/screen\" is a valid relative pathname.")
	}
}


// Checks whether the list of version numbers that are given as the
// C<value> of the variable C<varname> are in decreasing order.
sub checkline_decreasing_order($$$) {
	my (line, varname, value) = @_

	my @pyver = split(qr"\s+", value)
	if (!@pyver) {
		line.logError("There must be at least one value for ${varname}.")
		return
	}

	my ver = shift(@pyver)
	if (ver !~ `^\d+$`) {
		line.logError("All values for ${varname} must be numeric.")
		return
	}

	while (@pyver) {
		my nextver = shift(@pyver)
		if (nextver !~ `^\d+$`) {
			line.logError("All values for ${varname} must be numeric.")
			return
		}

		if (nextver >= ver) {
			line.logWarning("The values for ${varname} should be in decreasing order.")
			line.explainWarning(
"If they aren't, it may be possible that needless versions of packages",
"are installed.")
		}
		ver = nextver
	}
}

sub checkline_mk_vartype($$$$$) {
	my (line, varname, op, value, comment) = @_

	return unless opt_warn_types

	my vartypes = get_vartypes_map()
	my varbase = varname_base(varname)
	my varcanon = varname_canon(varname)

	my type = get_variable_type(line, varname)

	if (op == "+=") {
		if (defined(type)) {
			if (!type.may_use_plus_eq()) {
				line.logWarning("The \"+=\" operator should only be used with lists.")
			}
		} else if (varbase !~ `^_` && varbase !~ get_regex_plurals()) {
			line.logWarning("As ${varname} is modified using \"+=\", its name should indicate plural.")
		}
	}

	if (!defined(type)) {
		// Cannot check anything if the type is not known.
		opt_debug_unchecked and line.logDebug("Unchecked variable assignment for ${varname}.")

	} else if (op == "!=") {
		opt_debug_misc and line.logDebug("Use of !=: ${value}")

	} else if (type.kind_of_list != LK_NONE) {
		my (@words, rest)

		if (type.kind_of_list == LK_INTERNAL) {
			@words = split(qr"\s+", value)
			rest = ""
		} else {
			@words = ()
			rest = value
			while (rest =~ s/^regex_shellword//) {
				my (word) = (1)
				last if (word =~ `^#`)
				push(@words, 1)
			}
		}

		foreach my word (@words) {
			checkline_mk_vartype_basic(line, varname, type.basic_type, op, word, comment, true, type.is_guessed)
			if (type.kind_of_list != LK_INTERNAL) {
				checkline_mk_shellword(line, word, true)
			}
		}

		if (rest !~ `^\s*$`) {
			line.logError("Internal pkglint error: rest=${rest}")
		}

	} else {
		checkline_mk_vartype_basic(line, varname, type.basic_type, op, value, comment, type.is_practically_a_list(), type.is_guessed)
	}
}

sub checkline_mk_varassign($$$$$) {
	my (line, varname, op, value, comment) = @_
	my (used_vars)
	my varbase = varname_base(varname)
	my varcanon = varname_canon(varname)

	opt_debug_trace and line.logDebug("checkline_mk_varassign(varname, op, value)")

	checkline_mk_vardef(line, varname, op)

	if (op == "?=" && defined(seen_bsd_prefs_mk) && !seen_bsd_prefs_mk) {
		if (varbase == "BUILDLINK_PKGSRCDIR"
			|| varbase == "BUILDLINK_DEPMETHOD"
			|| varbase == "BUILDLINK_ABI_DEPENDS") {
			// FIXME: What about these ones? They occur quite often.
		} else {
			opt_warn_extra and line.logWarning("Please include \"../../mk/bsd.prefs.mk\" before using \"?=\".")
			opt_warn_extra and line.explainWarning(
"The ?= operator is used to provide a default value to a variable. In",
"pkgsrc, many variables can be set by the pkgsrc user in the mk.conf",
"file. This file must be included explicitly. If a ?= operator appears",
"before mk.conf has been included, it will not care about the user's",
"preferences, which can result in unexpected behavior. The easiest way",
"to include the mk.conf file is by including the bsd.prefs.mk file,",
"which will take care of everything.")
		}
	}

	checkline_mk_text(line, value)
	checkline_mk_vartype(line, varname, op, value, comment)

	// If the variable is not used and is untyped, it may be a
	// spelling mistake.
	if (op == ":=" && varname == lc(varname)) {
		opt_debug_unchecked and line.logDebug("${varname} might be unused unless it is an argument to a procedure file.")
		// TODO: check varname against the list of "procedure files".

	} else if (!var_is_used(varname)) {
		my vartypes = get_vartypes_map()
		my deprecated = get_deprecated_map()

		if (exists(vartypes.{varname}) || exists(vartypes.{varcanon})) {
			// Ok
		} else if (exists(deprecated.{varname}) || exists(deprecated.{varcanon})) {
			// Ok
		} else {
			line.logWarning("${varname} is defined but not used. Spelling mistake?")
		}
	}

	if (value =~ `/etc/rc\.d`) {
		line.logWarning("Please use the RCD_SCRIPTS mechanism to install rc.d scripts automatically to \${RCD_SCRIPTS_EXAMPLEDIR}.")
	}

	if (!is_internal && varname =~ `^_`) {
		line.logWarning("Variable names starting with an underscore are reserved for internal pkgsrc use.")
	}

	if (varname == "PERL5_PACKLIST" && defined(effective_pkgbase) && effective_pkgbase =~ `^p5-(.*)`) {
		my (guess) = (1)
		guess =~ s/-/\//g
		guess = "auto/${guess}/.packlist"

		my (ucvalue, ucguess) = (uc(value), uc(guess))
		if (ucvalue != ucguess && ucvalue != "\${PERL5_SITEARCH\}/${ucguess}") {
			line.logWarning("Unusual value for PERL5_PACKLIST -- \"${guess}\" expected.")
		}
	}

	if (varname == "CONFIGURE_ARGS" && value =~ `=\$\{PREFIX\}/share/kde`) {
		line.logNote("Please .include \"../../meta-pkgs/kde3/kde3.mk\" instead of this line.")
		line.explain_note(
"That file probably does many things automatically and consistently that",
"this package also does. When using kde3.mk, you can probably also leave",
"out some explicit dependencies.")
	}

	if (varname == "EVAL_PREFIX" && value =~ `^([\w_]+)=`) {
		my (eval_varname) = (1)

		// The variables mentioned in EVAL_PREFIX will later be
		// defined by find-prefix.mk. Therefore, they are marked
		// as known in the current file.
		mkctx_vardef.{eval_varname} = line
	}

	if (varname == "PYTHON_VERSIONS_ACCEPTED") {
		checkline_decreasing_order(line, varname, value)
	}

	if (defined(comment) && comment == "# defined" && varname !~ `.*(?:_MK|_COMMON)$`) {
		line.logNote("Please use \"# empty\", \"# none\" or \"yes\" instead of \"# defined\".")
		line.explain_note(
"The value #defined says something about the state of the variable, but",
"not what that _means_. In some cases a variable that is defined means",
"\"yes\", in other cases it is an empty list (which is also only the",
"state of the variable), whose meaning could be described with \"none\".",
"It is this meaning that should be described.")
	}

	if (value =~ `\$\{(PKGNAME|PKGVERSION)[:\}]`) {
		my (pkgvarname) = (1)
		if (varname =~ `^PKG_.*_REASON$`) {
			// ok
		} else if (varname =~ `^(?:DIST_SUBDIR|WRKSRC)$`) {
			line.logWarning("${pkgvarname} should not be used in ${varname}, as it sometimes includes the PKGREVISION. Please use ${pkgvarname}_NOREV instead.")
		} else {
			opt_debug_misc and line.logDebug("Use of PKGNAME in ${varname}.")
		}
	}

	if (exists(get_deprecated_map().{varname})) {
		line.logWarning("Definition of ${varname} is deprecated. ".get_deprecated_map().{varname})
	} else if (exists(get_deprecated_map().{varcanon})) {
		line.logWarning("Definition of ${varname} is deprecated. ".get_deprecated_map().{varcanon})
	}

	if (varname =~ `^SITES_`) {
		line.logWarning("SITES_* is deprecated. Please use SITES.* instead.")
	}

	if (value =~ `^[^=]\@comment`) {
		line.logWarning("Please don't use \@comment in ${varname}.")
		line.explainWarning(
"Here you are defining a variable containing \@comment. As this value",
"typically includes a space as the last character you probably also used",
"quotes around the variable. This can lead to confusion when adding this",
"variable to PLIST_SUBST, as all other variables are quoted using the :Q",
"operator when they are appended. As it is hard to check whether a",
"variable that is appended to PLIST_SUBST is already quoted or not, you",
"should not have pre-quoted variables at all. To solve this, you should",
"directly use PLIST_SUBST+= ${varname}=${value} or use any other",
"variable for collecting the list of PLIST substitutions and later",
"append that variable with PLIST_SUBST+= \${MY_PLIST_SUBST}.")
	}

	// Mark the variable as PLIST condition. This is later used in
	// checkfile_PLIST.
	if (defined(pkgctx_plist_subst_cond) && value =~ `(.+)=.*\@comment.*`) {
		pkgctx_plist_subst_cond.{1}++
	}

	use constant op_to_use_time => {
		":="	=> VUC_TIME_LOAD,
		"!="	=> VUC_TIME_LOAD,
		"="	=> VUC_TIME_RUN,
		"+="	=> VUC_TIME_RUN,
		"?="	=> VUC_TIME_RUN
	}

	used_vars = extract_used_variables(line, value)
	my vuc = PkgLint::VarUseContext.new(
		op_to_use_time.{op},
		get_variable_type(line, varname),
		VUC_SHELLWORD_UNKNOWN,		# XXX: maybe PLAIN?
		VUC_EXTENT_UNKNOWN
	)
	foreach my used_var (@{used_vars}) {
		checkline_mk_varuse(line, used_var, "", vuc)
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

//
// Procedures to check a directory including the files in it.
//

sub checkdir_category() {
	my fname = "${current_dir}/Makefile"
	my (lines, lineno)

	opt_debug_trace and logDebug(fname, NO_LINES, "checkdir_category()")

	if (!(lines = load_lines(fname, true))) {
		logError(fname, NO_LINE_NUMBER, "Cannot be read.")
		return
	}
	parselines_mk(lines)

	lineno = 0

	// The first line must contain the RCS Id
	if (lineno <= $#{lines} && checkline_rcsid_regex(lines.[lineno], qr"#\s+", "# ")) {
		lineno++
	}

	// Then, arbitrary comments may follow
	while (lineno <= $#{lines} && lines.[lineno].text =~ `^#`) {
		lineno++
	}

	// Then we need an empty line
	expect_empty_line(lines, \lineno)

	// Then comes the COMMENT line
	if (lineno <= $#{lines} && lines.[lineno].text =~ `^COMMENT=\t*(.*)`) {
		my (comment) = (1)

		checkline_valid_characters_in_variable(lines.[lineno], qr"[-\040'(),/0-9A-Za-z]")
		lineno++
	} else {
		lines.[lineno].logError("COMMENT= line expected.")
	}

	// Then we need an empty line
	expect_empty_line(lines, \lineno)

	// And now to the most complicated part of the category Makefiles,
	// the (hopefully) sorted list of SUBDIRs. The first step is to
	// collect the SUBDIRs in the Makefile and in the file system.

	my (@f_subdirs, @m_subdirs)

	@f_subdirs = sort(get_subdirs(current_dir))

	my prev_subdir = undef
	while (lineno <= $#{lines}) {
		my line = lines.[lineno]

		if (line.text =~ `^(#?)SUBDIR\+=(\s*)(\S+)\s*(?:#\s*(.*?)\s*|)$`) {
			my (comment_flag, indentation, subdir, comment) = (1, 2, 3, 4)

			if (comment_flag == "#" && (!defined(comment) || comment == "")) {
				line.logWarning("${subdir} commented out without giving a reason.")
			}

			if (indentation != "\t") {
				line.logWarning("Indentation should be a single tab character.")
			}

			if (defined(prev_subdir) && subdir == prev_subdir) {
				line.logError("${subdir} must only appear once.")
			} else if (defined(prev_subdir) && subdir lt prev_subdir) {
				line.logWarning("${subdir} should come before ${prev_subdir}.")
			} else {
				// correctly ordered
			}

			push(@m_subdirs, [subdir, line, comment_flag ? false : true])
			prev_subdir = subdir
			lineno++

		} else {
			if (line.text != "") {
				line.logError("SUBDIR+= line or empty line expected.")
			}
			last
		}
	}

	// To prevent unnecessary warnings about subdirectories that are
	// in one list, but not in the other, we generate the sets of
	// subdirs of each list.
	my (%f_check, %m_check)
	foreach my f (@f_subdirs) { f_check{f} = true; }
	foreach my m (@m_subdirs) { m_check{m.[0]} = true; }

	my (f_index, f_atend, f_neednext, f_current) = (0, false, true, undef, undef)
	my (m_index, m_atend, m_neednext, m_current) = (0, false, true, undef, undef)
	my (line, m_recurse)
	my (@subdirs)

	while (!(m_atend && f_atend)) {

		if (!m_atend && m_neednext) {
			m_neednext = false
			if (m_index > $#m_subdirs) {
				m_atend = true
				line = lines.[lineno]
				next
			} else {
				m_current = m_subdirs[m_index].[0]
				line = m_subdirs[m_index].[1]
				m_recurse = m_subdirs[m_index].[2]
				m_index++
			}
		}

		if (!f_atend && f_neednext) {
			f_neednext = false
			if (f_index > $#f_subdirs) {
				f_atend = true
				next
			} else {
				f_current = f_subdirs[f_index++]
			}
		}

		if (!f_atend && (m_atend || f_current lt m_current)) {
			if (!exists(m_check{f_current})) {
				line.logError("${f_current} exists in the file system, but not in the Makefile.")
				line.append_before("SUBDIR+=\t${f_current}")
			}
			f_neednext = true

		} else if (!m_atend && (f_atend || m_current lt f_current)) {
			if (!exists(f_check{m_current})) {
				line.logError("${m_current} exists in the Makefile, but not in the file system.")
				line.delete()
			}
			m_neednext = true

		} else { # f_current == m_current
			f_neednext = true
			m_neednext = true
			if (m_recurse) {
				push(@subdirs, "${current_dir}/${m_current}")
			}
		}
	}

	// the wip category Makefile may have its own targets for generating
	// indexes and READMEs. Just skip them.
	if (is_wip) {
		while (lineno <= $#{lines} - 2) {
			lineno++
		}
	}

	expect_empty_line(lines, \lineno)

	// And, last but not least, the .include line
	my final_line = ".include \"../mk/bsd.pkg.subdir.mk\""
	expect(lines, \lineno, qr"\Qfinal_line\E")
	|| expect_text(lines, \lineno, ".include \"../mk/misc/category.mk\"")

	if (lineno <= $#{lines}) {
		lines.[lineno].logError("The file should end here.")
	}

	checklines_mk(lines)

	autofix(lines)

	if (opt_recursive) {
		unshift(@todo_items, @subdirs)
	}
}

sub checkdir_package() {
	my (lines, have_distinfo, have_patches)

	// Initialize global variables
	pkgdir = undef
	filesdir = "files"
	patchdir = "patches"
	distinfo_file = "distinfo"
	effective_pkgname = undef
	effective_pkgbase = undef
	effective_pkgversion = undef
	effective_pkgname_line = undef
	seen_bsd_prefs_mk = false
	pkgctx_vardef = {%{get_userdefined_variables()}}
	pkgctx_varuse = {}
	pkgctx_bl3 = {}
	pkgctx_plist_subst_cond = {}
	pkgctx_included = {}
	seen_Makefile_common = false

	// we need to handle the Makefile first to get some variables
	if (!load_package_Makefile("${current_dir}/Makefile", \lines)) {
		logError("${current_dir}/Makefile", NO_LINE_NUMBER, "Cannot be read.")
		goto cleanup
	}

	my @files = glob("${current_dir}/*")
	if (pkgdir != ".") {
		push(@files, glob("${current_dir}/${pkgdir}/*"))
	}
	if (opt_check_extra) {
		push(@files, glob("${current_dir}/${filesdir}/*"))
	}
	push(@files, glob("${current_dir}/${patchdir}/*"))
	if (distinfo_file !~ `^(?:\./)?distinfo$`) {
		push(@files, "${current_dir}/${distinfo_file}")
	}
	have_distinfo = false
	have_patches = false

	// Determine the used variables before checking any of the
	// Makefile fragments.
	foreach my fname (@files) {
		if ((fname =~ `^((?:.*/)?Makefile\..*|.*\.mk)$`)
		&& (not fname =~ `patch-`)
		&& (not fname =~ `${pkgdir}/`)
		&& (not fname =~ `${filesdir}/`)
		&& (defined(my lines = load_lines(fname, true)))) {
			parselines_mk(lines)
			determine_used_variables(lines)
		}
	}

	foreach my fname (@files) {
		if (fname == "${current_dir}/Makefile") {
			opt_check_Makefile and checkfile_package_Makefile(fname, lines)
		} else {
			checkfile(fname)
		}
		if (fname =~ `/patches/patch-*$`) {
			have_patches = true
		} else if (fname =~ `/distinfo$`) {
			have_distinfo = true
		}
	}

	if (opt_check_distinfo && opt_check_patches) {
		if (have_patches && ! have_distinfo) {
			logWarning("${current_dir}/distinfo_file", NO_LINE_NUMBER, "File not found. Please run '".conf_make." makepatchsum'.")
		}
	}

	if (!is_emptydir("${current_dir}/scripts")) {
		logWarning("${current_dir}/scripts", NO_LINE_NUMBER, "This directory and its contents are deprecated! Please call the script(s) explicitly from the corresponding target(s) in the pkg's Makefile.")
	}

cleanup:
	// Clean up global variables.
	pkgdir = undef
	filesdir = undef
	patchdir = undef
	distinfo_file = undef
	effective_pkgname = undef
	effective_pkgbase = undef
	effective_pkgversion = undef
	effective_pkgname_line = undef
	seen_bsd_prefs_mk = undef
	pkgctx_vardef = undef
	pkgctx_varuse = undef
	pkgctx_bl3 = undef
	pkgctx_plist_subst_cond = undef
	pkgctx_included = undef
	seen_Makefile_common = undef
}
