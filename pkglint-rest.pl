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

sub checkline_length($$) {
	my (line, maxlength) = @_

	if (length(line.text) > maxlength) {
		line.logWarning("Line too long (should be no more than maxlength characters).")
		line.explainWarning(
"Back in the old time, terminals with 80x25 characters were common.",
"And this is still the default size of many terminal emulators.",
"Moderately short lines also make reading easier.")
	}
}

sub checkline_valid_characters($$) {
	my (line, re_validchars) = @_
	my (rest)

	(rest = line.text) =~ s/re_validchars//g
	if (rest != "") {
		my @chars = map { sprintf("0x%02x", ord(_)); } split(//, rest)
		line.logWarning("Line contains invalid characters (" . join(", ", @chars) . ").")
	}
}

sub checkline_valid_characters_in_variable($$) {
	my (line, re_validchars) = @_
	my (varname, rest)

	varname = line.get("varname")
	rest = line.get("value")

	rest =~ s/re_validchars//g
	if (rest != "") {
		my @chars = map { sprintf("0x%02x", ord(_)); } split(//, rest)
		line.logWarning("${varname} contains invalid characters (" . join(", ", @chars) . ").")
	}
}

sub checkline_trailing_whitespace($) {
	my (line) = @_

	opt_debug_trace and line.logDebug("checkline_trailing_whitespace()")

	if (line.text =~ /\s+$/) {
		line.logNote("Trailing white-space.")
		line.explain_note(
"When a line ends with some white-space, that space is in most cases",
"irrelevant and can be removed, leading to a \"normal form\" syntax.",
"",
"Note: This is mostly for aesthetic reasons.")
		line.replace_regex(qr"\s+\n$", "\n")
	}
}

sub checkline_rcsid_regex($$$) {
	my (line, prefix_regex, prefix) = @_
	my (id) = (opt_rcsidstring . (is_wip ? "|Id" : ""))

	opt_debug_trace and line.logDebug("checkline_rcsid_regex(${prefix_regex}, ${prefix})")

	if (line.text !~ m"^${prefix_regex}\$(${id})(?::[^\$]+|)\$$") {
		line.logError("\"${prefix}\$${opt_rcsidstring}\$\" expected.")
		line.explain_error(
"Several files in pkgsrc must contain the CVS Id, so that their current",
"version can be traced back later from a binary package. This is to",
"ensure reproducible builds, for example for finding bugs.",
"",
"Please insert the text from the above error message (without the quotes)",
"at this position in the file.")
		return false
	}
	return true
}

sub checkline_rcsid($$) {
	my (line, prefix) = @_

	checkline_rcsid_regex(line, quotemeta(prefix), prefix)
}

sub checkline_mk_absolute_pathname($$) {
	my (line, text) = @_
	my abspath

	opt_debug_trace and line.logDebug("checkline_mk_absolute_pathname(${text})")

	// In the GNU coding standards, DESTDIR is defined as a (usually
	// empty) prefix that can be used to install files to a different
	// location from what they have been built for. Therefore
	// everything following it is considered an absolute pathname.
	// Another commonly used context is in assignments like
	// "bindir=/bin".
	if (text =~ m"(?:^|\$\{DESTDIR\}|\$\(DESTDIR\)|[\w_]+\s*=\s*)(/(?:[^\"'\`\s]|\"[^\"*]\"|'[^']*'|\`[^\`]*\`)*)") {
		my path = 1

		if (path =~ m"^/\w") {
			abspath = path
		}
	}

	if (defined(abspath)) {
		checkword_absolute_pathname(line, abspath)
	}
}

sub checkline_relative_path($$$) {
	my (line, path, must_exist) = @_
	my (res_path)

	if (!is_wip && path =~ m"/wip/") {
		line.logError("A pkgsrc package must not depend on any outside package.")
	}
	res_path = resolve_relative_path(path, true)
	if (res_path =~ regex_unresolved) {
		opt_debug_unchecked and line.logDebug("Unchecked path: \"${path}\".")
	} else if (!-e (((res_path =~ m"^/") ? "" : "${current_dir}/") . res_path)) {
		must_exist and line.logError("\"${res_path}\" does not exist.")
	} else if (path =~ m"^\.\./\.\./([^/]+)/([^/]+)(.*)") {
		my (cat, pkg, rest) = (1, 2, 3)
	} else if (path =~ m"^\.\./\.\./mk/") {
		// There need not be two directory levels for mk/ files.
	} else if (path =~ m"^\.\./mk/" && cur_pkgsrcdir == "..") {
		// That's fine for category Makefiles.
	} else if (path =~ m"^\.\.") {
		line.logWarning("Invalid relative path \"${path}\".")
	}
}

sub checkline_relative_pkgdir($$) {
	my (line, path) = @_

	checkline_relative_path(line, path, true)
	path = resolve_relative_path(path, false)

	if (path =~ m"^(?:\./)?\.\./\.\./([^/]+/[^/]+)$") {
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

sub checkline_mk_shelltext($$) {
	my (line, text) = @_
	my (vartools, state, rest, set_e_mode)

	if (rest =~ s///) {
		my (hidden, macro) = (1, 2)


		if (macro == "\${RUN}") {
			set_e_mode = true
		}
	}

	state = SCST_START
	while (rest =~ s/^regex_shellword//) {
		my (shellword) = (1)

		opt_debug_shell and line.logDebug(scst_statename.[state] . ": ${shellword}")

AAAAAASDFDHFJDFSDGSDGSDGSD



		//
		// State transition.
		//


		state =  (shellword == ";;") ? SCST_CASE_LABEL
			// Note: The order of the following two lines is important.
			: (state == SCST_CASE_LABEL_CONT && shellword == "|") ? SCST_CASE_LABEL
			: (shellword =~ m"^[;&\|]+$") ? SCST_START
			: (state == SCST_START) ? (
				(shellword == "\${INSTALL}") ? SCST_INSTALL
				: (shellword == "\${MKDIR}") ? SCST_MKDIR
				: (shellword == "\${PAX}") ? SCST_PAX
				: (shellword == "\${SED}") ? SCST_SED
				: (shellword == "\${ECHO}") ? SCST_ECHO
				: (shellword == "\${RUN}") ? SCST_START
				: (shellword == "echo") ? SCST_ECHO
				: (shellword == "set") ? SCST_SET
				: (shellword =~ m"^(?:if|elif|while)$") ? SCST_COND
				: (shellword =~ m"^(?:then|else|do)$") ? SCST_START
				: (shellword == "case") ? SCST_CASE
				: (shellword == "for") ? SCST_FOR
				: (shellword == "(") ? SCST_START
				: (shellword =~ m"^\$\{INSTALL_[A-Z]+_DIR\}$") ? SCST_INSTALL_DIR
				: (shellword =~ regex_sh_varassign) ? SCST_START
				: SCST_CONT)
			: (state == SCST_MKDIR) ? SCST_MKDIR
			: (state == SCST_INSTALL && shellword == "-d") ? SCST_INSTALL_D
			: (state == SCST_INSTALL || state == SCST_INSTALL_D) ? (
				(shellword =~ m"^-[ogm]$") ? SCST_CONT
				: state)
			: (state == SCST_INSTALL_DIR) ? (
				(shellword =~ m"^-") ? SCST_CONT
				: (shellword =~ m"^\$") ? SCST_INSTALL_DIR2
				: state)
			: (state == SCST_INSTALL_DIR2) ? state
			: (state == SCST_PAX) ? (
				(shellword == "-s") ? SCST_PAX_S
				: (shellword =~ m"^-") ? SCST_PAX
				: SCST_CONT)
			: (state == SCST_PAX_S) ? SCST_PAX
			: (state == SCST_SED) ? (
				(shellword == "-e") ? SCST_SED_E
				: (shellword =~ m"^-") ? SCST_SED
				: SCST_CONT)
			: (state == SCST_SED_E) ? SCST_SED
			: (state == SCST_SET) ? SCST_SET_CONT
			: (state == SCST_SET_CONT) ? SCST_SET_CONT
			: (state == SCST_CASE) ? SCST_CASE_IN
			: (state == SCST_CASE_IN && shellword == "in") ? SCST_CASE_LABEL
			: (state == SCST_CASE_LABEL && shellword == "esac") ? SCST_CONT
			: (state == SCST_CASE_LABEL) ? SCST_CASE_LABEL_CONT
			: (state == SCST_CASE_LABEL_CONT && shellword == ")") ? SCST_START
			: (state == SCST_CONT) ? SCST_CONT
			: (state == SCST_COND) ? SCST_COND_CONT
			: (state == SCST_COND_CONT) ? SCST_COND_CONT
			: (state == SCST_FOR) ? SCST_FOR_IN
			: (state == SCST_FOR_IN && shellword == "in") ? SCST_FOR_CONT
			: (state == SCST_FOR_CONT) ? SCST_FOR_CONT
			: (state == SCST_ECHO) ? SCST_CONT
			: do {
				line.logWarning("[" . scst_statename.[state] . " ${shellword}] Keeping the current state.")
				state
			}
	}

	if (rest !~ m"^\s*$") {
		line.logError("Internal pkglint error: " . scst_statename.[state] . ": rest=${rest}")
	}
}

sub checkline_mk_shellcmd($$) {
	my (line, shellcmd) = @_

	checkline_mk_text(line, shellcmd)
	checkline_mk_shelltext(line, shellcmd)
}


sub expand_permission($) {
	my (perm) = @_
	my %fullperm = ( "a" => "append", "d" => "default", "p" => "preprocess", "s" => "set", "u" => "runtime", "?" => "unknown" )
	my result = join(", ", map { fullperm{_} } split //, perm)
	result =~ s/, $//g

	return result
}

sub checkline_mk_vardef($$$) {
	my (line, varname, op) = @_

	opt_debug_trace and line.logDebug("checkline_mk_vardef(${varname}, ${op})")

	// If we are checking a whole package, add it to the package-wide
	// list of defined variables.
	if (defined(pkgctx_vardef) && !exists(pkgctx_vardef.{varname})) {
		pkgctx_vardef.{varname} = line
	}

	// Add it to the file-wide list of defined variables.
	if (!exists(mkctx_vardef.{varname})) {
		mkctx_vardef.{varname} = line
	}

	return unless opt_warn_perm

	my perms = get_variable_perms(line, varname)
	my needed = { "=" => "s", "!=" => "s", "?=" => "d", "+=" => "a", ":=" => "s" }.{op}
	if (index(perms, needed) == -1) {
		line.logWarning("Permission [" . expand_permission(needed) . "] requested for ${varname}, but only [" . expand_permission(perms) . "] is allowed.")
		line.explainWarning(
"The available permissions are:",
"\tappend\t\tappend something using +=",
"\tdefault\t\tset a default value using ?=",
"\tpreprocess\tuse a variable during preprocessing",
"\truntime\t\tuse a variable at runtime",
"\tset\t\tset a variable using :=, =, !=",
"",
"A \"?\" means that it is not yet clear which permissions are allowed",
"and which aren't.")
	}
}

// @param op
//	The operator that is used for reading or writing to the variable.
//	One of: "=", "+=", ":=", "!=", "?=", "use", "pp-use", "".
//	For some variables (like BuildlinkDepth or BuildlinkPackages), the
//	operator influences the valid values.
// @param comment
//	In assignments, a part of the line may be commented out. If there
//	is no comment, pass C<undef>.
//
sub checkline_mk_vartype_basic($$$$$$$$)
sub checkline_mk_vartype_basic($$$$$$$$) {
	my (line, varname, type, op, value, comment, list_context, is_guessed) = @_
	my (value_novar)

	opt_debug_trace and line.logDebug(sprintf("checkline_mk_vartype_basic(%s, %s, %s, %s, %s, %s, %s)",
		varname, type, op, value, defined(comment) ? comment : "<undef>", list_context, is_guessed))

	value_novar = value
	while (value_novar =~ s/\$\{([^{}]*)\}//g) {
		my (varuse) = (1)
		if (!list_context && varuse =~ m":Q$") {
			line.logWarning("The :Q operator should only be used in lists and shell commands.")
		}
	}

	my %type_dispatch = (
	)

	if (ref(type) == "HASH") {
		if (!exists(type.{value})) {
			line.logWarning("\"${value}\" is not valid for ${varname}. Use one of { ".join(" ", sort(keys(%{type})))." } instead.")
		}

	} else if (defined type_dispatch{type}) {
		type_dispatch{type}.()

	} else {
		line.log_fatal("Type ${type} unknown.")
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
	if (ver !~ m"^\d+$") {
		line.logError("All values for ${varname} must be numeric.")
		return
	}

	while (@pyver) {
		my nextver = shift(@pyver)
		if (nextver !~ m"^\d+$") {
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
		} else if (varbase !~ m"^_" && varbase !~ get_regex_plurals()) {
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
				last if (word =~ m"^#")
				push(@words, 1)
			}
		}

		foreach my word (@words) {
			checkline_mk_vartype_basic(line, varname, type.basic_type, op, word, comment, true, type.is_guessed)
			if (type.kind_of_list != LK_INTERNAL) {
				checkline_mk_shellword(line, word, true)
			}
		}

		if (rest !~ m"^\s*$") {
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

	if (value =~ m"/etc/rc\.d") {
		line.logWarning("Please use the RCD_SCRIPTS mechanism to install rc.d scripts automatically to \${RCD_SCRIPTS_EXAMPLEDIR}.")
	}

	if (!is_internal && varname =~ m"^_") {
		line.logWarning("Variable names starting with an underscore are reserved for internal pkgsrc use.")
	}

	if (varname == "PERL5_PACKLIST" && defined(effective_pkgbase) && effective_pkgbase =~ m"^p5-(.*)") {
		my (guess) = (1)
		guess =~ s/-/\//g
		guess = "auto/${guess}/.packlist"

		my (ucvalue, ucguess) = (uc(value), uc(guess))
		if (ucvalue != ucguess && ucvalue != "\${PERL5_SITEARCH\}/${ucguess}") {
			line.logWarning("Unusual value for PERL5_PACKLIST -- \"${guess}\" expected.")
		}
	}

	if (varname == "CONFIGURE_ARGS" && value =~ m"=\$\{PREFIX\}/share/kde") {
		line.logNote("Please .include \"../../meta-pkgs/kde3/kde3.mk\" instead of this line.")
		line.explain_note(
"That file probably does many things automatically and consistently that",
"this package also does. When using kde3.mk, you can probably also leave",
"out some explicit dependencies.")
	}

	if (varname == "EVAL_PREFIX" && value =~ m"^([\w_]+)=") {
		my (eval_varname) = (1)

		// The variables mentioned in EVAL_PREFIX will later be
		// defined by find-prefix.mk. Therefore, they are marked
		// as known in the current file.
		mkctx_vardef.{eval_varname} = line
	}

	if (varname == "PYTHON_VERSIONS_ACCEPTED") {
		checkline_decreasing_order(line, varname, value)
	}

	if (defined(comment) && comment == "# defined" && varname !~ m".*(?:_MK|_COMMON)$") {
		line.logNote("Please use \"# empty\", \"# none\" or \"yes\" instead of \"# defined\".")
		line.explain_note(
"The value #defined says something about the state of the variable, but",
"not what that _means_. In some cases a variable that is defined means",
"\"yes\", in other cases it is an empty list (which is also only the",
"state of the variable), whose meaning could be described with \"none\".",
"It is this meaning that should be described.")
	}

	if (value =~ m"\$\{(PKGNAME|PKGVERSION)[:\}]") {
		my (pkgvarname) = (1)
		if (varname =~ m"^PKG_.*_REASON$") {
			// ok
		} else if (varname =~ m"^(?:DIST_SUBDIR|WRKSRC)$") {
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

	if (varname =~ m"^SITES_") {
		line.logWarning("SITES_* is deprecated. Please use SITES.* instead.")
	}

	if (value =~ m"^[^=]\@comment") {
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
	if (defined(pkgctx_plist_subst_cond) && value =~ m"(.+)=.*\@comment.*") {
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
			if (match !~ m"[\$\[*]" && !exists(btype.{match})) {
				line.logWarning("Invalid :M value \"match\". Only { " . join(" ", sort keys %btype) . " } are allowed.")
			}
		}

		// Currently disabled because the valid options can also be defined in PKG_OPTIONS_GROUP.*.
		// Additionally, all these variables may have multiple assigments (+=).
		if (false && varname == "PKG_OPTIONS" && defined(pkgctx_vardef) && exists(pkgctx_vardef.{"PKG_SUPPORTED_OPTIONS"})) {
			my options = pkgctx_vardef.{"PKG_SUPPORTED_OPTIONS"}.get("value")

			if (match !~ m"[\$\[*]" && index(" options ", " match ") == -1) {
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

//
// Procedures to check an array of lines.
//

sub checklines_trailing_empty_lines($) {
	my (lines) = @_
	my (last, max)

	max = $#{lines} + 1
	for (last = max; last > 1 && lines.[last - 1].text == ""; ) {
		last--
	}
	if (last != max) {
		lines.[last].logNote("Trailing empty lines.")
	}
}

sub checklines_mk($) {
	my (lines) = @_
	my (allowed_targets) = ({})
	my (substcontext) = PkgLint::SubstContext.new()

	assert(@{lines} != 0, "checklines_mk may only be called with non-empty lines.")
	opt_debug_trace and logDebug(lines.[0].fname, NO_LINES, "checklines_mk()")

	// Define global variables for the Makefile context.
	mkctx_indentations = [0]
	mkctx_target = undef
	mkctx_for_variables = {}
	mkctx_vardef = {}
	mkctx_build_defs = {}
	mkctx_plist_vars = {}
	mkctx_tools = {%{get_predefined_tool_names()}}
	mkctx_varuse = {}

	determine_used_variables(lines)

	foreach my prefix (qw(pre do post)) {
		foreach my action (qw(fetch extract patch tools wrapper configure build test install package clean)) {
			allowed_targets.{"${prefix}-${action}"} = true
		}
	}

	//
	// In the first pass, all additions to BUILD_DEFS and USE_TOOLS
	// are collected to make the order of the definitions irrelevant.
	//

	foreach my line (@{lines}) {
		next unless line.has("is_varassign")
		my varcanon = line.get("varcanon")

		if (varcanon == "BUILD_DEFS" || varcanon == "PKG_GROUPS_VARS" || varcanon == "PKG_USERS_VARS") {
			foreach my varname (split(qr"\s+", line.get("value"))) {
				mkctx_build_defs.{varname} = true
				opt_debug_misc and line.logDebug("${varname} is added to BUILD_DEFS.")
			}

		} else if (varcanon == "PLIST_VARS") {
			foreach my id (split(qr"\s+", line.get("value"))) {
				mkctx_plist_vars.{"PLIST.id"} = true
				opt_debug_misc and line.logDebug("PLIST.${id} is added to PLIST_VARS.")
				use_var(line, "PLIST.id")
			}

		} else if (varcanon == "USE_TOOLS") {
			foreach my tool (split(qr"\s+", line.get("value"))) {
				tool =~ s/:(build|run)//
				mkctx_tools.{tool} = true
				opt_debug_misc and line.logDebug("${tool} is added to USE_TOOLS.")
			}

		} else if (varcanon == "SUBST_VARS.*") {
			foreach my svar (split(/\s+/, line.get("value"))) {
				use_var(svar, varname_canon(svar))
				opt_debug_misc and line.logDebug("varuse svar")
			}

		} else if (varcanon == "OPSYSVARS") {
			foreach my osvar (split(/\s+/, line.get("value"))) {
				use_var(line, "osvar.*")
				def_var(line, osvar)
			}
		}
	}

	//
	// In the second pass, all "normal" checks are done.
	//

	if (0 <= $#{lines}) {
		checkline_rcsid_regex(lines.[0], qr"^#\s+", "# ")
	}

	foreach my line (@{lines}) {
		my text = line.text

		checkline_trailing_whitespace(line)
		checkline_spellcheck(line)

		if (line.has("is_empty")) {
			substcontext.check_end(line)

		} else if (line.has("is_comment")) {
			// No further checks.

		} else if (text =~ regex_varassign) {
			my (varname, op, undef, comment) = (1, 2, 3, 4)
			my space1 = substr(text, $+[1], $-[2] - $+[1])
			my align = substr(text, $+[2], $-[3] - $+[2])
			my value = line.get("value")

			if (align !~ m"^(\t*|[ ])$") {
				opt_warn_space && line.logNote("Alignment of variable values should be done with tabs, not spaces.")
				my prefix = "${varname}${space1}${op}"
				my aligned_len = tablen("${prefix}${align}")
				if (aligned_len % 8 == 0) {
					my tabalign = ("\t" x ((aligned_len - tablen(prefix) + 7) / 8))
					line.replace("${prefix}${align}", "${prefix}${tabalign}")
				}
			}
			checkline_mk_varassign(line, varname, op, value, comment)
			substcontext.check_varassign(line, varname, op, value)

		} else if (text =~ regex_mk_shellcmd) {
			my (shellcmd) = (1)
			checkline_mk_shellcmd(line, shellcmd)

		} else if (text =~ regex_mk_include) {
			my (include, includefile) = (1, 2)

			opt_debug_include and line.logDebug("includefile=${includefile}")
			checkline_relative_path(line, includefile, include == "include")

			if (includefile =~ m"../Makefile$") {
				line.logError("Other Makefiles must not be included directly.")
				line.explainWarning(
"If you want to include portions of another Makefile, extract",
"the common parts and put them into a Makefile.common. After",
"that, both this one and the other package should include the",
"Makefile.common.")
			}

			if (includefile == "../../mk/bsd.prefs.mk") {
				if (line.fname =~ m"buildlink3\.mk$") {
					line.logNote("For efficiency reasons, please include bsd.fast.prefs.mk instead of bsd.prefs.mk.")
				}
				seen_bsd_prefs_mk = true
			} else if (includefile == "../../mk/bsd.fast.prefs.mk") {
				seen_bsd_prefs_mk = true
			}

			if (includefile =~ m"/x11-links/buildlink3\.mk$") {
				line.logError("${includefile} must not be included directly. Include \"../../mk/x11.buildlink3.mk\" instead.")
			}
			if (includefile =~ m"/jpeg/buildlink3\.mk$") {
				line.logError("${includefile} must not be included directly. Include \"../../mk/jpeg.buildlink3.mk\" instead.")
			}
			if (includefile =~ m"/intltool/buildlink3\.mk$") {
				line.logWarning("Please say \"USE_TOOLS+= intltool\" instead of this line.")
			}
			if (includefile =~ m"(.*)/builtin\.mk$") {
				my (dir) = (1)
				line.logError("${includefile} must not be included directly. Include \"${dir}/buildlink3.mk\" instead.")
			}

		} else if (text =~ regex_mk_sysinclude) {
			my (includefile, comment) = (1, 2)

			// No further action.

		} else if (text =~ regex_mk_cond) {
			my (indent, directive, args, comment) = (1, 2, 3, 4)

			use constant regex_directives_with_args => qr"^(?:if|ifdef|ifndef|elif|for|undef)$"

			if (directive =~ m"^(?:endif|endfor|elif|else)$") {
				if ($#{mkctx_indentations} >= 1) {
					pop(@{mkctx_indentations})
				} else {
					line.logError("Unmatched .${directive}.")
				}
			}

			// Check the indentation
			if (indent != " " x mkctx_indentations.[-1]) {
				opt_warn_space and line.logNote("This directive should be indented by ".mkctx_indentations.[-1]." spaces.")
			}

			if (directive == "if" && args =~ m"^!defined\([\w]+_MK\)$") {
				push(@{mkctx_indentations}, mkctx_indentations.[-1])

			} else if (directive =~ m"^(?:if|ifdef|ifndef|for|elif|else)$") {
				push(@{mkctx_indentations}, mkctx_indentations.[-1] + 2)
			}

			if (directive =~ regex_directives_with_args && !defined(args)) {
				line.logError("\".${directive}\" must be given some arguments.")

			} else if (directive !~ regex_directives_with_args && defined(args)) {
				line.logError("\".${directive}\" does not take arguments.")

				if (directive == "else") {
					line.logNote("If you meant \"else if\", use \".elif\".")
				}

			} else if (directive == "if" || directive == "elif") {
				checkline_mk_cond(line, args)

			} else if (directive == "ifdef" || directive == "ifndef") {
				if (args =~ m"\s") {
					line.logError("The \".${directive}\" directive can only handle _one_ argument.")
				} else {
					line.logWarning("The \".${directive}\" directive is deprecated. Please use \".if "
						. ((directive == "ifdef" ? "" : "!"))
						. "defined(${args})\" instead.")
				}

			} else if (directive == "for") {
				if (args =~ m"^(\S+(?:\s*\S+)*?)\s+in\s+(.*)$") {
					my (vars, values) = (1, 2)

					foreach my var (split(qr"\s+", vars)) {
						if (!is_internal && var =~ m"^_") {
							line.logWarning("Variable names starting with an underscore are reserved for internal pkgsrc use.")
						}

						if (var =~ m"^[_a-z][_a-z0-9]*$") {
							// Fine.
						} else if (var =~ m"[A-Z]") {
							line.logWarning(".for variable names should not contain uppercase letters.")
						} else {
							line.logError("Invalid variable name \"${var}\".")
						}

						mkctx_for_variables.{var} = true
					}

					// Check if any of the value's types is not guessed.
					my guessed = true
					foreach my value (split(qr"\s+", values)) { # XXX: too simple
						if (value =~ m"^\$\{(.*)\}") {
							my type = get_variable_type(line, 1)
							if (defined(type) && !type.is_guessed()) {
								guessed = false
							}
						}
					}

					my for_loop_type = PkgLint::Type.new(
						LK_INTERNAL,
						"Unchecked",
						[[qr".*", "pu"]],
						guessed
					)
					my for_loop_context = PkgLint::VarUseContext.new(
						VUC_TIME_LOAD,
						for_loop_type,
						VUC_SHELLWORD_FOR,
						VUC_EXTENT_WORD
					)
					foreach my var (@{extract_used_variables(line, values)}) {
						checkline_mk_varuse(line, var, "", for_loop_context)
					}

				}

			} else if (directive == "undef" && defined(args)) {
				foreach my var (split(qr"\s+", args)) {
					if (exists(mkctx_for_variables.{var})) {
						line.logNote("Using \".undef\" after a \".for\" loop is unnecessary.")
					}
				}
			}

		} else if (text =~ regex_mk_dependency) {
			my (targets, whitespace, dependencies, comment) = (1, 2, 3, 4)

			opt_debug_misc and line.logDebug("targets=${targets}, dependencies=${dependencies}")
			mkctx_target = targets

			foreach my source (split(/\s+/, dependencies)) {
				if (source == ".PHONY") {
					foreach my target (split(/\s+/, targets)) {
						allowed_targets.{target} = true
					}
				}
			}

			foreach my target (split(/\s+/, targets)) {
				if (target == ".PHONY") {
					foreach my dep (split(qr"\s+", dependencies)) {
						allowed_targets.{dep} = true
					}

				} else if (target == ".ORDER") {
					// TODO: Check for spelling mistakes.

				} else if (!exists(allowed_targets.{target})) {
					line.logWarning("Unusual target \"${target}\".")
					line.explainWarning(
"If you really want to define your own targets, you can \"declare\"",
"them by inserting a \".PHONY: my-target\" line before this line. This",
"will tell make(1) to not interpret this target's name as a filename.")
				}
			}

		} else if (text =~ m"^\.\s*(\S*)") {
			my (directive) = (1)

			line.logError("Unknown directive \".${directive}\".")

		} else if (text =~ m"^ ") {
			line.logWarning("Makefile lines should not start with space characters.")
			line.explainWarning(
"If you want this line to contain a shell program, use a tab",
"character for indentation. Otherwise please remove the leading",
"white-space.")

		} else {
			line.logError("[Internal] Unknown line format: text")
		}
	}
	if (@{lines} > 0) {
		substcontext.check_end(lines.[-1])
	}

	checklines_trailing_empty_lines(lines)

	if ($#{mkctx_indentations} != 0) {
		lines.[-1].logError("Directive indentation is not 0, but ".mkctx_indentations.[-1]." at EOF.")
	}

	// Clean up global variables.
	mkctx_for_variables = undef
	mkctx_indentations = undef
	mkctx_target = undef
	mkctx_vardef = undef
	mkctx_build_defs = undef
	mkctx_plist_vars = undef
	mkctx_tools = undef
	mkctx_varuse = undef
}

sub checklines_buildlink3_inclusion($) {
	my (lines) = @_
	my (included_files)

	assert(@{lines} != 0, "The lines array must be non-empty.")
	opt_debug_trace and logDebug(lines.[0].fname, NO_LINES, "checklines_buildlink3_inclusion()")

	if (!defined(pkgctx_bl3)) {
		return
	}

	// Collect all the included buildlink3.mk files from the file.
	included_files = {}
	foreach my line (@{lines}) {
		if (line.text =~ regex_mk_include) {
			my (undef, file, comment) = (1, 2, 3)

			if (file =~ m"^\.\./\.\./(.*)/buildlink3\.mk") {
				my (bl3) = (1)

				included_files.{bl3} = line
				if (!exists(pkgctx_bl3.{bl3})) {
					line.logWarning("${bl3}/buildlink3.mk is included by this file but not by the package.")
				}
			}
		}
	}

	// Print debugging messages for all buildlink3.mk files that are
	// included by the package but not by this buildlink3.mk file.
	foreach my package_bl3 (sort(keys(%{pkgctx_bl3}))) {
		if (!exists(included_files.{package_bl3})) {
			opt_debug_misc and pkgctx_bl3.{package_bl3}.logDebug("${package_bl3}/buildlink3.mk is included by the package but not by the buildlink3.mk file.")
		}
	}
}

//
// Procedures to check a single file.
//

sub checkfile_ALTERNATIVES($) {
	my (fname) = @_
	my (lines)

	opt_debug_trace and logDebug(fname, NO_LINES, "checkfile_ALTERNATIVES()")

	checkperms(fname)
	if (!(lines = load_file(fname))) {
		logError(fname, NO_LINE_NUMBER, "Cannot be read.")
		return
	}
}

sub checkfile_buildlink3_mk($) {
	my (fname) = @_
	my (lines, lineno, m)

	opt_debug_trace and logDebug(fname, NO_LINES, "checkfile_buildlink3_mk()")

	checkperms(fname)
	if (!(lines = load_lines(fname, true))) {
		logError(fname, NO_LINE_NUMBER, "Cannot be read.")
		return
	}
	if (@{lines} == 0) {
		logError(fname, NO_LINES, "Must not be empty.")
		return
	}

	parselines_mk(lines)
	checklines_mk(lines)

	lineno = 0

	// Header comments
	while (lineno <= $#{lines} && (my text = lines.[lineno].text) =~ m"^#") {
		if (text =~ m"^# XXX") {
			lines.[lineno].logNote("Please read this comment and remove it if appropriate.")
		}
		lineno++
	}
	expect_empty_line(lines, \lineno)

	if (expect(lines, \lineno, qr"^BUILDLINK_DEPMETHOD\.(\S+)\?=.*$")) {
		lines.[lineno - 1].logWarning("This line belongs inside the .ifdef block.")
		while (lines.[lineno].text == "") {
			lineno++
		}
	}

	if (!(m = expect(lines, \lineno, qr"^BUILDLINK_TREE\+=\s*(\S+)$"))) {
		lines.[lineno].logWarning("Expected a BUILDLINK_TREE line.")
		return
	}

	checklines_buildlink3_mk(lines, lineno, m.text(1))
}

sub checklines_buildlink3_mk($$$) {
	my (lines, lineno, pkgid) = @_
	my (m)
	my (bl_PKGBASE_line, bl_PKGBASE)
	my (bl_pkgbase_line, bl_pkgbase)
	my (abi_line, abi_pkg, abi_version)
	my (api_line, api_pkg, api_version)

	// First paragraph: Introduction of the package identifier
	bl_pkgbase_line = lines.[lineno - 1]
	bl_pkgbase = pkgid
	opt_debug_misc and bl_pkgbase_line.logDebug("bl_pkgbase=${bl_pkgbase}")
	expect_empty_line(lines, \lineno)

	// Second paragraph: multiple inclusion protection and introduction
	// of the uppercase package identifier.
	return unless (m = expect_re(lines, \lineno, qr"^\.if !defined\((\S+)_BUILDLINK3_MK\)$"))
	bl_PKGBASE_line = lines.[lineno - 1]
	bl_PKGBASE = m.text(1)
	opt_debug_misc and bl_PKGBASE_line.logDebug("bl_PKGBASE=${bl_PKGBASE}")
	expect_re(lines, \lineno, qr"^\Qbl_PKGBASE\E_BUILDLINK3_MK:=$")
	expect_empty_line(lines, \lineno)

	my norm_bl_pkgbase = bl_pkgbase
	norm_bl_pkgbase =~ s/-/_/g
	norm_bl_pkgbase = uc(norm_bl_pkgbase)
	if (norm_bl_pkgbase != bl_PKGBASE) {
		bl_PKGBASE_line.logError("Package name mismatch between ${bl_PKGBASE} ...")
		bl_pkgbase_line.logError("... and ${bl_pkgbase}.")
	}
	if (defined(effective_pkgbase) && effective_pkgbase != bl_pkgbase) {
		bl_pkgbase_line.logError("Package name mismatch between ${bl_pkgbase} ...")
		effective_pkgname_line.logError("... and ${effective_pkgbase}.")
	}

	// Third paragraph: Package information.
	my if_level = 1; # the first .if is from the second paragraph.
	while (true) {

		if (lineno > $#{lines}) {
			lines_logWarning(lines, lineno, "Expected .endif")
			return
		}

		my line = lines.[lineno]

		if ((m = expect(lines, \lineno, regex_varassign))) {
			my (varname, value) = (m.text(1), m.text(3))
			my do_check = false

			if (varname == "BUILDLINK_ABI_DEPENDS.${bl_pkgbase}") {
				abi_line = line
				if (value =~ regex_dependency_lge) {
					(abi_pkg, abi_version) = (1, 2)
				} else if (value =~ regex_dependency_wildcard) {
					(abi_pkg) = (1)
				} else {
					opt_debug_unchecked and line.logDebug("Unchecked dependency pattern \"${value}\".")
				}
				do_check = true
			}
			if (varname == "BUILDLINK_API_DEPENDS.${bl_pkgbase}") {
				api_line = line
				if (value =~ regex_dependency_lge) {
					(api_pkg, api_version) = (1, 2)
				} else if (value =~ regex_dependency_wildcard) {
					(api_pkg) = (1)
				} else {
					opt_debug_unchecked and line.logDebug("Unchecked dependency pattern \"${value}\".")
				}
				do_check = true
			}
			if (do_check && defined(abi_pkg) && defined(api_pkg)) {
				if (abi_pkg != api_pkg) {
					abi_line.logWarning("Package name mismatch between ${abi_pkg} ...")
					api_line.logWarning("... and ${api_pkg}.")
				}
			}
			if (do_check && defined(abi_version) && defined(api_version)) {
				if (!dewey_cmp(abi_version, ">=", api_version)) {
					abi_line.logWarning("ABI version (${abi_version}) should be at least ...")
					api_line.logWarning("... API version (${api_version}).")
				}
			}

			if (varname =~ m"^BUILDLINK_[\w_]+\.(.*)$") {
				my (varparam) = (1)

				if (varparam != bl_pkgbase) {
					line.logWarning("Only buildlink variables for ${bl_pkgbase}, not ${varparam} may be set in this file.")
				}
			}

			if (varname == "pkgbase") {
				expect_re(lines, \lineno, qr"^\.\s*include \"../../mk/pkg-build-options.mk\"$")
			}

			// TODO: More checks.

		} else if (expect(lines, \lineno, qr"^(?:#.*)?$")) {
			// Comments and empty lines are fine here.

		} else if (expect(lines, \lineno, qr"^\.\s*include \"\.\./\.\./([^/]+/[^/]+)/buildlink3\.mk\"$")
			|| expect(lines, \lineno, qr"^\.\s*include \"\.\./\.\./mk/(\S+)\.buildlink3\.mk\"$")) {
			// TODO: Maybe check dependency lines.

		} else if (expect(lines, \lineno, qr"^\.if\s")) {
			if_level++

		} else if (expect(lines, \lineno, qr"^\.endif.*$")) {
			if_level--
			last if if_level == 0

		} else {
			opt_debug_unchecked and lines_logWarning(lines, lineno, "Unchecked line in third paragraph.")
			lineno++
		}
	}
	if (!defined(api_line)) {
		lines.[lineno - 1].logWarning("Definition of BUILDLINK_API_DEPENDS is missing.")
	}
	expect_empty_line(lines, \lineno)

	// Fourth paragraph: Cleanup, corresponding to the first paragraph.
	return unless expect_re(lines, \lineno, qr"^BUILDLINK_TREE\+=\s*-\Qbl_pkgbase\E$")

	if (lineno <= $#{lines}) {
		lines.[lineno].logWarning("The file should end here.")
	}

	checklines_buildlink3_inclusion(lines)
}

sub checkfile_DESCR($) {
	my (fname) = @_
	my (maxchars, maxlines) = (80, 24)
	my (lines)

	opt_debug_trace and logDebug(fname, NO_LINES, "checkfile_DESCR()")

	checkperms(fname)
	if (!(lines = load_file(fname))) {
		logError(fname, NO_LINE_NUMBER, "Cannot be read.")
		return
	}
	if (@{lines} == 0) {
		logError(fname, NO_LINE_NUMBER, "Must not be empty.")
		return
	}

	foreach my line (@{lines}) {
		checkline_length(line, maxchars)
		checkline_trailing_whitespace(line)
		checkline_valid_characters(line, regex_validchars)
		checkline_spellcheck(line)
		if (line.text =~ m"\$\{") {
			line.logWarning("Variables are not expanded in the DESCR file.")
		}
	}
	checklines_trailing_empty_lines(lines)

	if (@{lines} > maxlines) {
		my line = lines.[maxlines]

		line.logWarning("File too long (should be no more than maxlines lines).")
		line.explainWarning(
"A common terminal size is 80x25 characters. The DESCR file should",
"fit on one screen. It is also intended to give a _brief_ summary",
"about the package's contents.")
	}
	autofix(lines)
}

sub checkfile_distinfo($) {
	my (fname) = @_
	my (lines, %in_distinfo, patches_dir, di_is_committed, current_fname, is_patch, @seen_algs)

	opt_debug_trace and logDebug(fname, NO_LINES, "checkfile_distinfo()")

	di_is_committed = is_committed(fname)

	checkperms(fname)
	if (!(lines = load_file(fname))) {
		logError(fname, NO_LINE_NUMBER, "Cannot be read.")
		return
	}

	if (@{lines} == 0) {
		logError(fname, NO_LINE_NUMBER, "Must not be empty.")
		return
	}

	checkline_rcsid(lines.[0], "")
	if (1 <= $#{lines} && lines.[1].text != "") {
		lines.[1].logNote("Empty line expected.")
		lines.[1].explain_note("This is merely for aesthetical purposes.")
	}

	patches_dir = patchdir
	if (!defined(patches_dir) && -d "${current_dir}/patches") {
		patches_dir = "patches"
	}

	my on_filename_change = sub($$) {
		my (line, new_fname) = @_

		if (defined(current_fname)) {
			my seen_algs = join(", ", @seen_algs)
			opt_debug_misc and line.logDebug("File ${current_fname} has checksums ${seen_algs}.")
			if (is_patch) {
				if (seen_algs != "SHA1") {
					line.logError("Expected SHA1 checksum for ${current_fname}, got ${seen_algs}.")
				}
			} else {
				if (seen_algs != "SHA1, RMD160, Size" && seen_algs != "SHA1, RMD160, SHA512, Size") {
					line.logError("Expected SHA1, RMD160, Size checksums for ${current_fname}, got ${seen_algs}.")
				}
			}
		}

		is_patch = defined(new_fname) && new_fname =~ m"^patch-.+$" ? true : false
		current_fname = new_fname
		@seen_algs = ()
	}

	foreach my line (@{lines}[2..$#{lines}]) {
		if (line.text !~ m"^(\w+) \(([^)]+)\) = (.*)(?: bytes)?$") {
			line.logError("Unknown line type.")
			next
		}
		my (alg, chksum_fname, sum) = (1, 2, 3)

		if (!defined(current_fname) || chksum_fname != current_fname) {
			on_filename_change.(line, chksum_fname)
		}

		if (chksum_fname !~ m"^\w") {
			line.logError("All file names should start with a letter.")
		}

		// Inter-package check for differing distfile checksums.
		if (opt_check_global && !is_patch) {
			// Note: Perl-specific auto-population.
			if (exists(ipc_distinfo.{alg}.{chksum_fname})) {
				my other = ipc_distinfo.{alg}.{chksum_fname}

				if (other.[1] == sum) {
					// Fine.
				} else {
					line.logError("The ${alg} checksum for ${chksum_fname} differs ...")
					other.[0].logError("... from this one.")
				}
			} else {
				ipc_distinfo.{alg}.{chksum_fname} = [line, sum]
			}
		}

		push(@seen_algs, alg)

		if (is_patch && defined(patches_dir) && !(defined(distinfo_file) && distinfo_file == "./../../lang/php5/distinfo")) {
			my fname = "${current_dir}/${patches_dir}/${chksum_fname}"
			if (di_is_committed && !is_committed(fname)) {
				line.logWarning("${patches_dir}/${chksum_fname} is registered in distinfo but not added to CVS.")
			}

			if (open(my patchfile, "<", fname)) {
				my sha1 = Digest::SHA1.new()
				while (defined(my patchline = <patchfile>)) {
					sha1.add(patchline) unless patchline =~ m"\$[N]etBSD"
				}
				close(patchfile)
				my chksum = sha1.hexdigest()
				if (sum != chksum) {
					line.logError("${alg} checksum of ${chksum_fname} differs (expected ${sum}, got ${chksum}). Rerun '".conf_make." makepatchsum'.")
				}
			} else {
				line.logWarning("${chksum_fname} does not exist.")
				line.explainWarning(
"All patches that are mentioned in a distinfo file should actually exist.",
"What's the use of a checksum if there is no file to check?")
			}
		}
		in_distinfo{chksum_fname} = true

	}
	on_filename_change.(PkgLint::Line.new(fname, NO_LINE_NUMBER, "", []), undef)
	checklines_trailing_empty_lines(lines)

	if (defined(patches_dir)) {
		foreach my patch (glob("${current_dir}/${patches_dir}/patch-*")) {
			patch = basename(patch)
			if (!exists(in_distinfo{patch})) {
				logError(fname, NO_LINE_NUMBER, "patch is not recorded. Rerun '".conf_make." makepatchsum'.")
			}
		}
	}
}

sub checkfile_extra($) {
	my (fname) = @_

	opt_debug_trace and logDebug(fname, NO_LINES, "checkfile_extra()")

	my lines = load_file(fname) or return logError(fname, NO_LINE_NUMBER, "Could not be read.")
	checklines_trailing_empty_lines(lines)
	checkperms(fname)
}

sub checkfile_INSTALL($) {
	my (fname) = @_

	opt_debug_trace and logDebug(fname, NO_LINES, "checkfile_INSTALL()")

	checkperms(fname)
	my lines = load_file(fname) or return logError(fname, NO_LINE_NUMBER, "Cannot be read.")
}

sub checkfile_MESSAGE($) {
	my (fname) = @_

	my @explanation = (
		"A MESSAGE file should consist of a header line, having 75 \"=\"",
		"characters, followed by a line containing only the RCS Id, then an",
		"empty line, your text and finally the footer line, which is the",
		"same as the header line.")

	opt_debug_trace and logDebug(fname, NO_LINES, "checkfile_MESSAGE()")

	checkperms(fname)
	my lines = load_file(fname) or return logError(fname, NO_LINE_NUMBER, "Cannot be read.")

	if (@{lines} < 3) {
		logWarning(fname, NO_LINE_NUMBER, "File too short.")
		explainWarning(fname, NO_LINE_NUMBER, @explanation)
		return
	}
	if (lines.[0].text != "=" x 75) {
		lines.[0].logWarning("Expected a line of exactly 75 \"=\" characters.")
		explainWarning(fname, NO_LINE_NUMBER, @explanation)
	}
	checkline_rcsid(lines.[1], "")
	foreach my line (@{lines}) {
		checkline_length(line, 80)
		checkline_trailing_whitespace(line)
		checkline_valid_characters(line, regex_validchars)
		checkline_spellcheck(line)
	}
	if (lines.[-1].text != "=" x 75) {
		lines.[-1].logWarning("Expected a line of exactly 75 \"=\" characters.")
		explainWarning(fname, NO_LINE_NUMBER, @explanation)
	}
	checklines_trailing_empty_lines(lines)
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

		if (languages_line.has("comment") && languages_line.get("comment") =~ m"\b(?:c|empty|none)\b"i) {
			// Don't emit a warning, since the comment
			// probably contains a statement that C is
			// really not needed.

		} else if (value !~ m"(?:^|\s+)(?:c|c99|objc)(?:\s+|$)") {
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

		if (pkgname =~ m"^(.*)\$\{DISTNAME:S(.)([^:]*)\2([^:]*)\2(g?)\}(.*)$") {
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
		if (false && pkgpath =~ m"/([^/]+)$" && effective_pkgbase != 1) {
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
	if (basename =~ m"^(?:work.*|.*~|.*\.orig|.*\.rej)$") {
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
		} else if (fname =~ m"(?:^|/)files/[^/]*$") {
			// Ok

		} else if (!is_emptydir(fname)) {
			logWarning(fname, NO_LINE_NUMBER, "Unknown directory name.")
		}

	} else if (S_ISLNK(st.mode)) {
		if (basename !~ m"^work") {
			logWarning(fname, NO_LINE_NUMBER, "Unknown symlink name.")
		}

	} else if (!S_ISREG(st.mode)) {
		logError(fname, NO_LINE_NUMBER, "Only files and directories are allowed in pkgsrc.")

	} else if (basename == "ALTERNATIVES") {
		opt_check_ALTERNATIVES and checkfile_ALTERNATIVES(fname)

	} else if (basename == "buildlink3.mk") {
		opt_check_bl3 and checkfile_buildlink3_mk(fname)

	} else if (basename =~ m"^DESCR") {
		opt_check_DESCR and checkfile_DESCR(fname)

	} else if (basename =~ m"^distinfo") {
		opt_check_distinfo and checkfile_distinfo(fname)

	} else if (basename == "DEINSTALL" || basename == "INSTALL") {
		opt_check_INSTALL and checkfile_INSTALL(fname)

	} else if (basename =~ m"^MESSAGE") {
		opt_check_MESSAGE and checkfile_MESSAGE(fname)

	} else if (basename =~ m"^patch-[-A-Za-z0-9_.~+]*[A-Za-z0-9_]$") {
		opt_check_patches and checkfile_patch(fname)

	} else if (fname =~ m"(?:^|/)patches/manual[^/]*$") {
		opt_debug_unchecked and logDebug(fname, NO_LINE_NUMBER, "Unchecked file \"${fname}\".")

	} else if (fname =~ m"(?:^|/)patches/[^/]*$") {
		logWarning(fname, NO_LINE_NUMBER, "Patch files should be named \"patch-\", followed by letters, '-', '_', '.', and digits only.")

	} else if (basename =~ m"^(?:.*\.mk|Makefile.*)$" and not fname =~ m,files/, and not fname =~ m,patches/,) {
		opt_check_mk and checkfile_mk(fname)

	} else if (basename =~ m"^PLIST") {
		opt_check_PLIST and checkfile_PLIST(fname)

	} else if (basename == "TODO" || basename == "README") {
		// Ok

	} else if (basename =~ m"^CHANGES-.*") {
		load_doc_CHANGES(fname)

	} else if (!-T fname) {
		logWarning(fname, NO_LINE_NUMBER, "Unexpectedly found a binary file.")

	} else if (fname =~ m"(?:^|/)files/[^/]*$") {
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
	while (lineno <= $#{lines} && lines.[lineno].text =~ m"^#") {
		lineno++
	}

	// Then we need an empty line
	expect_empty_line(lines, \lineno)

	// Then comes the COMMENT line
	if (lineno <= $#{lines} && lines.[lineno].text =~ m"^COMMENT=\t*(.*)") {
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

		if (line.text =~ m"^(#?)SUBDIR\+=(\s*)(\S+)\s*(?:#\s*(.*?)\s*|)$") {
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
	if (distinfo_file !~ m"^(?:\./)?distinfo$") {
		push(@files, "${current_dir}/${distinfo_file}")
	}
	have_distinfo = false
	have_patches = false

	// Determine the used variables before checking any of the
	// Makefile fragments.
	foreach my fname (@files) {
		if ((fname =~ m"^((?:.*/)?Makefile\..*|.*\.mk)$")
		&& (not fname =~ m"patch-")
		&& (not fname =~ m"${pkgdir}/")
		&& (not fname =~ m"${filesdir}/")
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
		if (fname =~ m"/patches/patch-*$") {
			have_patches = true
		} else if (fname =~ m"/distinfo$") {
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
