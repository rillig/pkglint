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
	if (pref eq "SCALAR" && !defined($pattern)) {
		$pattern = tree
		return true
	}
	if (aref eq "" && (pref eq "" || pref eq "SCALAR")) {
		return tree eq pattern
	}
	if (aref eq "ARRAY" && pref eq "ARRAY") {
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
	while (cond ne "") {
		if (cond =~ s/^!//) {
			return ["not", parse_mk_cond(line, cond)]
		} elsif (cond =~ s/^defined\((${re_simple_varname})\)$//) {
			return ["defined", 1]
		} elsif (cond =~ s/^empty\((${re_simple_varname})\)$//) {
			return ["empty", 1]
		} elsif (cond =~ s/^empty\((${re_simple_varname}):M([^\$:{})]+)\)$//) {
			return ["empty", ["match", 1, 2]]
		} elsif (cond =~ s/^\$\{(${re_simple_varname})\}\s+(==|!=)\s+"([^"\$\\]*)"$//) { #"
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
		line.explain_warning(
"Back in the old time, terminals with 80x25 characters were common.",
"And this is still the default size of many terminal emulators.",
"Moderately short lines also make reading easier.")
	}
}

sub checkline_valid_characters($$) {
	my (line, re_validchars) = @_
	my (rest)

	(rest = line.text) =~ s/re_validchars//g
	if (rest ne "") {
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
	if (rest ne "") {
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

	# In the GNU coding standards, DESTDIR is defined as a (usually
	# empty) prefix that can be used to install files to a different
	# location from what they have been built for. Therefore
	# everything following it is considered an absolute pathname.
	# Another commonly used context is in assignments like
	# "bindir=/bin".
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
	} elsif (!-e (((res_path =~ m"^/") ? "" : "${current_dir}/") . res_path)) {
		must_exist and line.logError("\"${res_path}\" does not exist.")
	} elsif (path =~ m"^\.\./\.\./([^/]+)/([^/]+)(.*)") {
		my (cat, pkg, rest) = (1, 2, 3)
	} elsif (path =~ m"^\.\./\.\./mk/") {
		# There need not be two directory levels for mk/ files.
	} elsif (path =~ m"^\.\./mk/" && cur_pkgsrcdir eq "..") {
		# That's fine for category Makefiles.
	} elsif (path =~ m"^\.\.") {
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
		line.explain_warning(
"A relative pathname always starts with \"../../\", followed",
"by a category, a slash and a the directory name of the package.",
"For example, \"../../misc/screen\" is a valid relative pathname.")
	}
}

// Some shell commands should not be used in the install phase.
//
sub checkline_mk_shellcmd_use($$) {
	my (line, shellcmd) = @_

	use constant allowed_install_commands => array_to_hash(qw(
		${INSTALL}
		${INSTALL_DATA} ${INSTALL_DATA_DIR}
		${INSTALL_LIB} ${INSTALL_LIB_DIR}
		${INSTALL_MAN} ${INSTALL_MAN_DIR}
		${INSTALL_PROGRAM} ${INSTALL_PROGRAM_DIR}
		${INSTALL_SCRIPT}
		${LIBTOOL}
		${LN}
		${PAX}
	))
	use constant discouraged_install_commands => array_to_hash(qw(
		sed ${SED}
		tr ${TR}
	))

	if (defined(mkctx_target) && mkctx_target =~ m"^(?:pre|do|post)-install") {

		if (exists(allowed_install_commands.{shellcmd})) {
			# Fine.

		} elsif (exists(discouraged_install_commands.{shellcmd})) {
			line.logWarning("The shell command \"${shellcmd}\" should not be used in the install phase.")
			line.explain_warning(
"In the install phase, the only thing that should be done is to install",
"the prepared files to their final location. The file's contents should",
"not be changed anymore.")

		} elsif (shellcmd eq "\${CP}") {
			line.logWarning("\${CP} should not be used to install files.")
			line.explain_warning(
"The \${CP} command is highly platform dependent and cannot overwrite",
"files that don't have write permission. Please use \${PAX} instead.",
"",
"For example, instead of",
"\t\${CP} -R \${WRKSRC}/* \${PREFIX}/foodir",
"you should use",
"\tcd \${WRKSRC} && \${PAX} -wr * \${PREFIX}/foodir")

		} else {
			opt_debug_misc and line.logDebug("May \"${shellcmd}\" be used in the install phase?")
		}
	}
}

sub checkline_mk_shelltext($$) {
	my (line, text) = @_
	my (vartools, state, rest, set_e_mode)

	if (rest =~ s///) {
		my (hidden, macro) = (1, 2)


		if (macro eq "\${RUN}") {
			set_e_mode = true
		}
	}

	state = SCST_START
	while (rest =~ s/^regex_shellword//) {
		my (shellword) = (1)

		opt_debug_shell and line.logDebug(scst_statename.[state] . ": ${shellword}")

		checkline_mk_shellword(line, shellword, !(
			state == SCST_CASE
			|| state == SCST_FOR_CONT
			|| state == SCST_SET_CONT
			|| (state == SCST_START && shellword =~ regex_sh_varassign)))

		#
		# Actions associated with the current state
		# and the symbol on the "input tape".
		#

		if (state == SCST_START || state == SCST_COND) {
			...
		}

		if (state == SCST_COND && shellword eq "cd") {
			...
		}

		if ((state != SCST_PAX_S && state != SCST_SED_E && state != SCST_CASE_LABEL)) {
			...
		}

		if ((state == SCST_INSTALL_D || state == SCST_MKDIR) && shellword =~ m"^(?:\$\{DESTDIR\})?\$\{PREFIX(?:|:Q)\}/") {
			...
		}

		if ((state == SCST_INSTALL_DIR || state == SCST_INSTALL_DIR2) && shellword !~ regex_mk_shellvaruse && shellword =~ m"") {
			...
		}

		if (state == SCST_INSTALL_DIR2 && shellword =~ m"^\$") {
			line.logWarning("The INSTALL_*_DIR commands can only handle one directory at a time.")
			line.explain_warning(
"Many implementations of install(1) can handle more, but pkgsrc aims at",
"maximum portability.")
		}

		if (state == SCST_PAX && shellword eq "-pe") {
			line.logWarning("Please use the -pp option to pax(1) instead of -pe.")
			line.explain_warning(
"The -pe option tells pax to preserve the ownership of the files, which",
"means that the installed files will belong to the user that has built",
"the package.")
		}

		if (state == SCST_PAX_S || state == SCST_SED_E) {
			if (false && shellword !~ m"^[\"\'].*[\"\']$") {
				line.logWarning("Substitution commands like \"${shellword}\" should always be quoted.")
				line.explain_warning(
"Usually these substitution commands contain characters like '*' or",
"other shell metacharacters that might lead to lookup of matching",
"filenames and then expand to more than one word.")
			}
		}

		if (state == SCST_ECHO && shellword eq "-n") {
			line.logWarning("Please use \${ECHO_N} instead of \"echo -n\".")
		}

		if (opt_warn_extra && state != SCST_CASE_LABEL_CONT && shellword eq "|") {
			line.logWarning("The exitcode of the left-hand-side command of the pipe operator is ignored.")
			line.explain_warning(
"If you need to detect the failure of the left-hand-side command, use",
"temporary files to save the output of the command.")
		}

		if (opt_warn_extra && shellword eq ";" && state != SCST_COND_CONT && state != SCST_FOR_CONT && !set_e_mode) {
			line.logWarning("Please switch to \"set -e\" mode before using a semicolon to separate commands.")
			line.explain_warning(
"Older versions of the NetBSD make(1) had run the shell commands using",
"the \"-e\" option of /bin/sh. In 2004, this behavior has been changed to",
"follow the POSIX conventions, which is to not use the \"-e\" option.",
"The consequence of this change is that shell programs don't terminate",
"as soon as an error occurs, but try to continue with the next command.",
"Imagine what would happen for these commands:",
"    cd \"\HOME\"; cd /nonexistent; rm -rf *",
"To fix this warning, either insert \"set -e\" at the beginning of this",
"line or use the \"&&\" operator instead of the semicolon.")
		}

		#
		# State transition.
		#

		if (state == SCST_SET && shellword =~ m"^-.*e") {
			set_e_mode = true
		}
		if (state == SCST_START && shellword eq "\${RUN}") {
			set_e_mode = true
		}

		state =  (shellword eq ";;") ? SCST_CASE_LABEL
			# Note: The order of the following two lines is important.
			: (state == SCST_CASE_LABEL_CONT && shellword eq "|") ? SCST_CASE_LABEL
			: (shellword =~ m"^[;&\|]+$") ? SCST_START
			: (state == SCST_START) ? (
				(shellword eq "\${INSTALL}") ? SCST_INSTALL
				: (shellword eq "\${MKDIR}") ? SCST_MKDIR
				: (shellword eq "\${PAX}") ? SCST_PAX
				: (shellword eq "\${SED}") ? SCST_SED
				: (shellword eq "\${ECHO}") ? SCST_ECHO
				: (shellword eq "\${RUN}") ? SCST_START
				: (shellword eq "echo") ? SCST_ECHO
				: (shellword eq "set") ? SCST_SET
				: (shellword =~ m"^(?:if|elif|while)$") ? SCST_COND
				: (shellword =~ m"^(?:then|else|do)$") ? SCST_START
				: (shellword eq "case") ? SCST_CASE
				: (shellword eq "for") ? SCST_FOR
				: (shellword eq "(") ? SCST_START
				: (shellword =~ m"^\$\{INSTALL_[A-Z]+_DIR\}$") ? SCST_INSTALL_DIR
				: (shellword =~ regex_sh_varassign) ? SCST_START
				: SCST_CONT)
			: (state == SCST_MKDIR) ? SCST_MKDIR
			: (state == SCST_INSTALL && shellword eq "-d") ? SCST_INSTALL_D
			: (state == SCST_INSTALL || state == SCST_INSTALL_D) ? (
				(shellword =~ m"^-[ogm]$") ? SCST_CONT
				: state)
			: (state == SCST_INSTALL_DIR) ? (
				(shellword =~ m"^-") ? SCST_CONT
				: (shellword =~ m"^\$") ? SCST_INSTALL_DIR2
				: state)
			: (state == SCST_INSTALL_DIR2) ? state
			: (state == SCST_PAX) ? (
				(shellword eq "-s") ? SCST_PAX_S
				: (shellword =~ m"^-") ? SCST_PAX
				: SCST_CONT)
			: (state == SCST_PAX_S) ? SCST_PAX
			: (state == SCST_SED) ? (
				(shellword eq "-e") ? SCST_SED_E
				: (shellword =~ m"^-") ? SCST_SED
				: SCST_CONT)
			: (state == SCST_SED_E) ? SCST_SED
			: (state == SCST_SET) ? SCST_SET_CONT
			: (state == SCST_SET_CONT) ? SCST_SET_CONT
			: (state == SCST_CASE) ? SCST_CASE_IN
			: (state == SCST_CASE_IN && shellword eq "in") ? SCST_CASE_LABEL
			: (state == SCST_CASE_LABEL && shellword eq "esac") ? SCST_CONT
			: (state == SCST_CASE_LABEL) ? SCST_CASE_LABEL_CONT
			: (state == SCST_CASE_LABEL_CONT && shellword eq ")") ? SCST_START
			: (state == SCST_CONT) ? SCST_CONT
			: (state == SCST_COND) ? SCST_COND_CONT
			: (state == SCST_COND_CONT) ? SCST_COND_CONT
			: (state == SCST_FOR) ? SCST_FOR_IN
			: (state == SCST_FOR_IN && shellword eq "in") ? SCST_FOR_CONT
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

	# If we are checking a whole package, add it to the package-wide
	# list of defined variables.
	if (defined(pkgctx_vardef) && !exists(pkgctx_vardef.{varname})) {
		pkgctx_vardef.{varname} = line
	}

	# Add it to the file-wide list of defined variables.
	if (!exists(mkctx_vardef.{varname})) {
		mkctx_vardef.{varname} = line
	}

	return unless opt_warn_perm

	my perms = get_variable_perms(line, varname)
	my needed = { "=" => "s", "!=" => "s", "?=" => "d", "+=" => "a", ":=" => "s" }.{op}
	if (index(perms, needed) == -1) {
		line.logWarning("Permission [" . expand_permission(needed) . "] requested for ${varname}, but only [" . expand_permission(perms) . "] is allowed.")
		line.explain_warning(
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
		AwkCommand => sub {
			opt_debug_unchecked and line.logDebug("Unchecked AWK command: ${value}")
		},

		BrokenIn => sub {
			if (value ne value_novar) {
				line.logError("${varname} must not refer to other variables.")

			} elsif (value =~ m"^pkgsrc-(\d\d\d\d)Q(\d)$") {
				my (year, quarter) = (1, 2)

				# Fine.

			} else {
				line.logWarning("Invalid value \"${value}\" for ${varname}.")
			}
			line.logNote("Please remove this line if the package builds for you.")
		},

		BuildlinkDepmethod => sub {
			# Note: this cannot be replaced with { build full } because
			# enumerations may not contain references to other variables.
			if (value ne value_novar) {
				# No checks yet.
			} elsif (value ne "build" && value ne "full") {
				line.logWarning("Invalid dependency method \"${value}\". Valid methods are \"build\" or \"full\".")
			}
		},

		BuildlinkDepth => sub {
			if (!(op eq "use" && value eq "+")
				&& value ne "\${BUILDLINK_DEPTH}+"
				&& value ne "\${BUILDLINK_DEPTH:S/+\$//}") {
				line.logWarning("Invalid value for ${varname}.")
			}
		},

		BuildlinkPackages => sub {
			my re_del = qr"\$\{BUILDLINK_PACKAGES:N(?:[+\-.0-9A-Z_a-z]|\$\{[^\}]+\})+\}"
			my re_add = qr"(?:[+\-.0-9A-Z_a-z]|\$\{[^\}]+\})+"

			if ((op eq ":=" && value =~ m"^${re_del}$") ||
				(op eq ":=" && value =~ m"^${re_del}\s+${re_add}$") ||
				(op eq "+=" && value =~ m"^${re_add}$")) {
				# Fine.

			} else {
				line.logWarning("Invalid value for ${varname}.")
			}
		},

		Category => sub {
			my allowed_categories = join("|", qw(
				archivers audio
				benchmarks biology
				cad chat chinese comms converters cross crosspkgtools
				databases devel
				editors emulators
				filesystems finance fonts
				games geography gnome gnustep graphics
				ham
				inputmethod
				japanese java
				kde korean
				lang linux local
				mail math mbone meta-pkgs misc multimedia
				net news
				packages parallel perl5 pkgtools plan9 print python
				ruby
				scm security shells sysutils
				tcl textproc time tk
				windowmaker wm www
				x11 xmms
			))
			if (value !~ m"^(?:${allowed_categories})$") {
				line.logError("Invalid category \"${value}\".")
			}
		},

		CFlag => sub {
			if (value =~ m"^-D([0-9A-Z_a-z]+)=(.*)") {
				my (macname, macval) = (1, 2)

				# No checks needed, since the macro definitions
				# are usually directory names, which don't need
				# any quoting.

			} elsif (value =~ m"^-[DU]([0-9A-Z_a-z]+)") {
				my (macname) = (1)

				opt_debug_unchecked and line.logDebug("Unchecked macro ${macname} in ${varname}.")

			} elsif (value =~ m"^-I(.*)") {
				my (dirname) = (1)

				opt_debug_unchecked and line.logDebug("Unchecked directory ${dirname} in ${varname}.")

			} elsif (value eq "-c99") {
				# Only works on IRIX, but is usually enclosed with
				# the proper preprocessor conditional.

			} elsif (value =~ m"^-[OWfgm]|^-std=.*") {
				opt_debug_unchecked and line.logDebug("Unchecked compiler flag ${value} in ${varname}.")

			} elsif (value =~ m"^-.*") {
				line.logWarning("Unknown compiler flag \"${value}\".")

			} elsif (value =~ regex_unresolved) {
				opt_debug_unchecked and line.logDebug("Unchecked CFLAG: ${value}")

			} else {
				line.logWarning("Compiler flag \"${value}\" does not start with a dash.")
			}
		},

		Comment => sub {
			if (value eq "SHORT_DESCRIPTION_OF_THE_PACKAGE") {
				line.logError("COMMENT must be set.")
			}
			if (value =~ m"^(a|an)\s+"i) {
				line.logWarning("COMMENT should not begin with '1'.")
			}
			if (value =~ m"^[a-z]") {
				line.logWarning("COMMENT should start with a capital letter.")
			}
			if (value =~ m"\.$") {
				line.logWarning("COMMENT should not end with a period.")
			}
			if (length(value) > 70) {
				line.logWarning("COMMENT should not be longer than 70 characters.")
			}
		},

		Dependency => sub {
			if (value =~ m"^(${regex_pkgbase})(<|=|>|<=|>=|!=|-)(${regex_pkgversion})$") {
				my (depbase, depop, depversion) = (1, 2, 3)

			} elsif (value =~ m"^(${regex_pkgbase})-(?:\[(.*)\]\*|(\d+(?:\.\d+)*(?:\.\*)?)(\{,nb\*\}|\*|)|(.*))?$") {
				my (depbase, bracket, version, version_wildcard, other) = (1, 2, 3, 4, 5)

				if (defined(bracket)) {
					if (bracket ne "0-9") {
						line.logWarning("Only [0-9]* is allowed in the numeric part of a dependency.")
					}

				} elsif (defined(version) && defined(version_wildcard) && version_wildcard ne "") {
					# Great.

				} elsif (defined(version)) {
					line.logWarning("Please append {,nb*} to the version number of this dependency.")
					line.explain_warning(
"Usually, a dependency should stay valid when the PKGREVISION is",
"increased, since those changes are most often editorial. In the",
"current form, the dependency only matches if the PKGREVISION is",
"undefined.")

				} elsif (other eq "*") {
					line.logWarning("Please use ${depbase}-[0-9]* instead of ${depbase}-*.")
					line.explain_warning(
"If you use a * alone, the package specification may match other",
"packages that have the same prefix, but a longer name. For example,",
"foo-* matches foo-1.2, but also foo-client-1.2 and foo-server-1.2.")

				} else {
					line.logError("Unknown dependency pattern \"${value}\".")
				}

			} elsif (value =~ m"\{") {
				# Dependency patterns containing alternatives
				# are just too hard to check.
				opt_debug_unchecked and line.logDebug("Unchecked dependency pattern: ${value}")

			} elsif (value ne value_novar) {
				opt_debug_unchecked and line.logDebug("Unchecked dependency: ${value}")

			} else {
				line.logWarning("Unknown dependency format: ${value}")
				line.explain_warning(
"Typical dependencies have the form \"package>=2.5\", \"package-[0-9]*\"",
"or \"package-3.141\".")
			}
		},

		DependencyWithPath => sub {
			if (value =~ regex_unresolved) {
				# don't even try to check anything
			} elsif (value =~ m"(.*):(\.\./\.\./([^/]+)/([^/]+))$") {
				my (pattern, relpath, cat, pkg) = (1, 2, 3, 4)

				checkline_relative_pkgdir(line, relpath)

				if (pkg eq "msgfmt" || pkg eq "gettext") {
					line.logWarning("Please use USE_TOOLS+=msgfmt instead of this dependency.")

				} elsif (pkg =~ m"^perl\d+") {
					line.logWarning("Please use USE_TOOLS+=perl:run instead of this dependency.")

				} elsif (pkg eq "gmake") {
					line.logWarning("Please use USE_TOOLS+=gmake instead of this dependency.")

				}

				if (pattern =~ regex_dependency_lge) {
//				(abi_pkg, abi_version) = (1, 2)
				} elsif (pattern =~ regex_dependency_wildcard) {
//				(abi_pkg) = (1)
				} else {
					line.logError("Unknown dependency pattern \"${pattern}\".")
				}

			} elsif (value =~ m":\.\./[^/]+$") {
				line.logWarning("Dependencies should have the form \"../../category/package\".")
				line.explain_warning(expl_relative_dirs)

			} else {
				line.logWarning("Unknown dependency format.")
				line.explain_warning(
"Examples for valid dependencies are:",
"  package-[0-9]*:../../category/package",
"  package>=3.41:../../category/package",
"  package-2.718:../../category/package")
			}
		},

		DistSuffix => sub {
			if (value eq ".tar.gz") {
				line.logNote("${varname} is \".tar.gz\" by default, so this definition may be redundant.")
			}
		},

		EmulPlatform => sub {
			if (value =~ m"^(\w+)-(\w+)$") {
				my (opsys, arch) = (1, 2)

				if (opsys !~ m"^(?:bsdos|cygwin|darwin|dragonfly|freebsd|haiku|hpux|interix|irix|linux|netbsd|openbsd|osf1|sunos|solaris)$") {
					line.logWarning("Unknown operating system: ${opsys}")
				}
				# no check for os_version
				if (arch !~ m"^(?:i386|alpha|amd64|arc|arm|arm32|cobalt|convex|dreamcast|hpcmips|hpcsh|hppa|ia64|m68k|m88k|mips|mips64|mipsel|mipseb|mipsn32|ns32k|pc532|pmax|powerpc|rs6000|s390|sparc|sparc64|vax|x86_64)$") {
					line.logWarning("Unknown hardware architecture: ${arch}")
				}

			} else {
				line.logWarning("\"${value}\" is not a valid emulation platform.")
				line.explain_warning(
"An emulation platform has the form <OPSYS>-<MACHINE_ARCH>.",
"OPSYS is the lower-case name of the operating system, and MACHINE_ARCH",
"is the hardware architecture.",
"",
"Examples: linux-i386, irix-mipsel.")
			}
		},

		FetchURL => sub {
			checkline_mk_vartype_basic(line, varname, "URL", op, value, comment, list_context, is_guessed)

			my sites = get_dist_sites()
			foreach my site (keys(%{sites})) {
				if (index(value, site) == 0) {
					my subdir = substr(value, length(site))
					my is_github = value =~ m"^https://github\.com/"
					if (is_github) {
						subdir =~ s|/.*|/|
					}
					line.logWarning(sprintf("Please use \${%s:=%s} instead of \"%s\".", sites.{site}, subdir, value))
					if (is_github) {
						line.logWarning("Run \"".conf_make." help topic=github\" for further tips.")
					}
					last
				}
			}
		},

		Filename => sub {
			if (value_novar =~ m"/") {
				line.logWarning("A filename should not contain a slash.")

			} elsif (value_novar !~ m"^[-0-9\@A-Za-z.,_~+%]*$") {
				line.logWarning("\"${value}\" is not a valid filename.")
			}
		},

		Filemask => sub {
			if (value_novar !~ m"^[-0-9A-Za-z._~+%*?]*$") {
				line.logWarning("\"${value}\" is not a valid filename mask.")
			}
		},

		FileMode => sub {
			if (value ne "" && value_novar eq "") {
				# Fine.
			} elsif (value =~ m"^[0-7]{3,4}") {
				# Fine.
			} else {
				line.logWarning("Invalid file mode ${value}.")
			}
		},

		Identifier => sub {
			if (value ne value_novar) {
				#line.logWarning("Identifiers should be given directly.")
			}
			if (value_novar =~ m"^[+\-.0-9A-Z_a-z]+$") {
				# Fine.
			} elsif (value ne "" && value_novar eq "") {
				# Don't warn here.
			} else {
				line.logWarning("Invalid identifier \"${value}\".")
			}
		},

		Integer => sub {
			if (value !~ m"^\d+$") {
				line.logWarning("${varname} must be a valid integer.")
			}
		},

		LdFlag => sub {
			if (value =~ m"^-L(.*)") {
				my (dirname) = (1)

				opt_debug_unchecked and line.logDebug("Unchecked directory ${dirname} in ${varname}.")

			} elsif (value =~ m"^-l(.*)") {
				my (libname) = (1)

				opt_debug_unchecked and line.logDebug("Unchecked library name ${libname} in ${varname}.")

			} elsif (value =~ m"^(?:-static)$") {
				# Assume that the wrapper framework catches these.

			} elsif (value =~ m"^(-Wl,(?:-R|-rpath|--rpath))") {
				my (rpath_flag) = (1)
				line.logWarning("Please use \${COMPILER_RPATH_FLAG} instead of ${rpath_flag}.")

			} elsif (value =~ m"^-.*") {
				line.logWarning("Unknown linker flag \"${value}\".")

			} elsif (value =~ regex_unresolved) {
				opt_debug_unchecked and line.logDebug("Unchecked LDFLAG: ${value}")

			} else {
				line.logWarning("Linker flag \"${value}\" does not start with a dash.")
			}
		},

		License => CheckvartypeLicense,

		Mail_Address => CheckvartypeMailAddress,

		Message => CheckvartypeMessage,

		Option => CheckvartypeOption,

		Pathlist => sub {

			if (value !~ m":" && is_guessed) {
				checkline_mk_vartype_basic(line, varname, "Pathname", op, value, comment, list_context, is_guessed)

			} else {

				# XXX: The splitting will fail if value contains any
				# variables with modifiers, for example :Q or :S/././.
				foreach my p (split(qr":", value)) {
					my p_novar = remove_variables(p)

					if (p_novar !~ m"^[-0-9A-Za-z._~+%/]*$") {
						line.logWarning("\"${p}\" is not a valid pathname.")
					}

					if (p !~ m"^[\$/]") {
						line.logWarning("All components of ${varname} (in this case \"${p}\") should be an absolute path.")
					}
				}
			}
		},

		Pathmask => sub {
			if (value_novar !~ m"^[#\-0-9A-Za-z._~+%*?/\[\]]*$") {
				line.logWarning("\"${value}\" is not a valid pathname mask.")
			}
			checkline_mk_absolute_pathname(line, value)
		},

		Pathname => sub {
			if (value_novar !~ m"^[#\-0-9A-Za-z._~+%/]*$") {
				line.logWarning("\"${value}\" is not a valid pathname.")
			}
			checkline_mk_absolute_pathname(line, value)
		},

		Perl5Packlist => sub {
			if (value ne value_novar) {
				line.logWarning("${varname} should not depend on other variables.")
			}
		},

		PkgName => sub {
			if (value eq value_novar && value !~ regex_pkgname) {
				line.logWarning("\"${value}\" is not a valid package name. A valid package name has the form packagename-version, where version consists only of digits, letters and dots.")
			}
		},

		PkgPath => sub {
			checkline_relative_pkgdir(line, "cur_pkgsrcdir/value")
		},

		PkgOptionsVar => sub {
			checkline_mk_vartype_basic(line, varname, "Varname", op, value, comment, false, is_guessed)
			if (value =~ m"\$\{PKGBASE[:\}]") {
				line.logError("PKGBASE must not be used in PKG_OPTIONS_VAR.")
				line.explain_error(
"PKGBASE is defined in bsd.pkg.mk, which is included as the",
"very last file, but PKG_OPTIONS_VAR is evaluated earlier.",
"Use \${PKGNAME:C/-[0-9].*//} instead.")
			}
		},

		PkgRevision => sub {
			if (value !~ m"^[1-9]\d*$") {
				line.logWarning("${varname} must be a positive integer number.")
			}
			if (line.fname !~ m"(?:^|/)Makefile$") {
				line.logError("${varname} only makes sense directly in the package Makefile.")
				line.explain_error(
"Usually, different packages using the same Makefile.common have",
"different dependencies and will be bumped at different times (e.g. for",
"shlib major bumps) and thus the PKGREVISIONs must be in the separate",
"Makefiles. There is no practical way of having this information in a",
"commonly used Makefile.")
			}
		},

		PlatformTriple => sub {
			my part = qr"(?:\[[^\]]+\]|[^-\[])+"
			if (value =~ m"^(${part})-(${part})-(${part})$") {
				my (opsys, os_version, arch) = (1, 2, 3)

				if (opsys !~ m"^(?:\*|BSDOS|Cygwin|Darwin|DragonFly|FreeBSD|Haiku|HPUX|Interix|IRIX|Linux|NetBSD|OpenBSD|OSF1|QNX|SunOS)$") {
					line.logWarning("Unknown operating system: ${opsys}")
				}
				# no check for os_version
				if (arch !~ m"^(?:\*|i386|alpha|amd64|arc|arm|arm32|cobalt|convex|dreamcast|hpcmips|hpcsh|hppa|ia64|m68k|m88k|mips|mips64|mipsel|mipseb|mipsn32|ns32k|pc532|pmax|powerpc|rs6000|s390|sparc|sparc64|vax|x86_64)$") {
					line.logWarning("Unknown hardware architecture: ${arch}")
				}

			} else {
				line.logWarning("\"${value}\" is not a valid platform triple.")
				line.explain_warning(
"A platform triple has the form <OPSYS>-<OS_VERSION>-<MACHINE_ARCH>.",
"Each of these components may be a shell globbing expression.",
"Examples: NetBSD-*-i386, *-*-*, Linux-*-*.")
			}
		},

		PrefixPathname => sub {
			if (value =~ m"^man/(.*)") {
				my (mansubdir) = (1)

				line.logWarning("Please use \"\${PKGMANDIR}/${mansubdir}\" instead of \"${value}\".")
			}
		},

		PythonDependency => sub {
			if (value ne value_novar) {
				line.logWarning("Python dependencies should not contain variables.")
			}
			if (value_novar !~ m"^[+\-.0-9A-Z_a-z]+(?:|:link|:build)$") {
				line.logWarning("Invalid Python dependency \"${value}\".")
				line.explain_warning(
"Python dependencies must be an identifier for a package, as specified",
"in lang/python/versioned_dependencies.mk. This identifier may be",
"followed by :build for a build-time only dependency, or by :link for",
"a run-time only dependency.")
			}
		},

		RelativePkgDir => sub {
			checkline_relative_pkgdir(line, value)
		},

		RelativePkgPath => sub {
			checkline_relative_path(line, value, true)
		},

		Restricted => sub {
			if (value ne "\${RESTRICTED}") {
				line.logWarning("The only valid value for ${varname} is \${RESTRICTED}.")
				line.explain_warning(
"These variables are used to control which files may be mirrored on FTP",
"servers or CD-ROM collections. They are not intended to mark packages",
"whose only MASTER_SITES are on ftp.NetBSD.org.")
			}
		},

		SedCommand => sub {
		},

		SedCommands => sub {
			my words = shell_split(value)
			if (!words) {
				line.logError("Invalid shell words in sed commands.")
				line.explain_error(
"If your sed commands have embedded \"#\" characters, you need to escape",
"them with a backslash, otherwise make(1) will interpret them as a",
"comment, no matter if they occur in single or double quotes or",
"whatever.")

			} else {
				my nwords = scalar(@{words})
				my ncommands = 0

				for (my i = 0; i < nwords; i++) {
					my word = words.[i]
					checkline_mk_shellword(line, word, true)

					if (word eq "-e") {
						if (i + 1 < nwords) {
							# Check the real sed command here.
							i++
							ncommands++
							if (ncommands > 1) {
								line.logWarning("Each sed command should appear in an assignment of its own.")
								line.explain_warning(
"For example, instead of",
"    SUBST_SED.foo+=        -e s,command1,, -e s,command2,,",
"use",
"    SUBST_SED.foo+=        -e s,command1,,",
"    SUBST_SED.foo+=        -e s,command2,,",
"",
"This way, short sed commands cannot be hidden at the end of a line.")
							}
							checkline_mk_shellword(line, words.[i - 1], true)
							checkline_mk_shellword(line, words.[i], true)
							checkline_mk_vartype_basic(line, varname, "SedCommand", op, words.[i], comment, list_context, is_guessed)
						} else {
							line.logError("The -e option to sed requires an argument.")
						}
					} elsif (word eq "-E") {
						# Switch to extended regular expressions mode.

					} elsif (word eq "-n") {
						# Don't print lines per default.

					} elsif (i == 0 && word =~ m"^([\"']?)(?:\d*|/.*/)s(.).*\2g?\1$") {
						line.logWarning("Please always use \"-e\" in sed commands, even if there is only one substitution.")

					} else {
						line.logWarning("Unknown sed command ${word}.")
					}
				}
			}
		},

		ShellCommand => sub {
			checkline_mk_shelltext(line, value)
		},

		ShellWord => sub {
			if (!list_context) {
				checkline_mk_shellword(line, value, true)
			}
		},

		Stage => sub {
			if (value !~ m"^(?:pre|do|post)-(?:extract|patch|configure|build|install)$") {
				line.logWarning("Invalid stage name. Use one of {pre,do,post}-{extract,patch,configure,build,install}.")
			}
		},

		String => sub {
			# No further checks possible.
		},

		Tool => sub {
			if (varname eq "TOOLS_NOOP" && op eq "+=") {
				# no warning for package-defined tool definitions

			} elsif (value =~ m"^([-\w]+|\[)(?::(\w+))?$") {
				my (toolname, tooldep) = (1, 2)
				if (!exists(get_tool_names().{toolname})) {
					line.logError("Unknown tool \"${toolname}\".")
				}
				if (defined(tooldep) && tooldep !~ m"^(?:bootstrap|build|pkgsrc|run)$") {
					line.logError("Unknown tool dependency \"${tooldep}\". Use one of \"build\", \"pkgsrc\" or \"run\".")
				}
			} else {
				line.logError("Invalid tool syntax: \"${value}\".")
			}
		},

		Unchecked => sub {
			# Do nothing, as the name says.
		},

		URL => sub {
			if (value eq "" && defined(comment) && comment =~ m"^#") {
				# Ok

			} elsif (value =~ m"\$\{(MASTER_SITE_[^:]*).*:=(.*)\}$") {
				my (name, subdir) = (1, 2)

				if (!exists(get_dist_sites_names().{name})) {
					line.logError("${name} does not exist.")
				}
				if (subdir !~ m"/$") {
					line.logError("The subdirectory in ${name} must end with a slash.")
				}

			} elsif (value =~ regex_unresolved) {
				# No further checks

			} elsif (value =~ m"^(https?|ftp|gopher)://([-0-9A-Za-z.]+)(?::(\d+))?/([-%&+,./0-9:=?\@A-Z_a-z~]|#)*$") {
				my (proto, host, port, path) = (1, 2, 3, 4)

				if (host =~ m"\.NetBSD\.org$"i && host !~ m"\.NetBSD\.org$") {
					line.logWarning("Please write NetBSD.org instead of ${host}.")
				}

			} elsif (value =~ m"^([0-9A-Za-z]+)://([^/]+)(.*)$") {
				my (scheme, host, abs_path) = (1, 2, 3)

				if (scheme ne "ftp" && scheme ne "http" && scheme ne "https" && scheme ne "gopher") {
					line.logWarning("\"${value}\" is not a valid URL. Only ftp, gopher, http, and https URLs are allowed here.")

				} elsif (abs_path eq "") {
					line.logNote("For consistency, please add a trailing slash to \"${value}\".")

				} else {
					line.logWarning("\"${value}\" is not a valid URL.")
				}

			} else {
				line.logWarning("\"${value}\" is not a valid URL.")
			}
		},

		UserGroupName => sub {
			if (value ne value_novar) {
				# No checks for now.
			} elsif (value !~ m"^[0-9_a-z]+$") {
				line.logWarning("Invalid user or group name \"${value}\".")
			}
		},

		Varname => sub {
			if (value ne "" && value_novar eq "") {
				# The value of another variable

			} elsif (value_novar !~ m"^[A-Z_][0-9A-Z_]*(?:[.].*)?$") {
				line.logWarning("\"${value}\" is not a valid variable name.")
			}
		},

		Version => sub {
			if (value !~ m"^([\d.])+$") {
				line.logWarning("Invalid version number \"${value}\".")
			}
		},

		WrapperReorder => sub {
			if (value =~ m"^reorder:l:([\w\-]+):([\w\-]+)$") {
				my (lib1, lib2) = (1, 2)
				# Fine.
			} else {
				line.logWarning("Unknown wrapper reorder command \"${value}\".")
			}
		},

		WrapperTransform => sub {
			if (value =~ m"^rm:(?:-[DILOUWflm].*|-std=.*)$") {
				# Fine.

			} elsif (value =~ m"^l:([^:]+):(.+)$") {
				my (lib, replacement_libs) = (1, 2)
				# Fine.

			} elsif (value =~ m"^'?(?:opt|rename|rm-optarg|rmdir):.*$") {
				# FIXME: This is cheated.
				# Fine.

			} elsif (value eq "-e" || value =~ m"^\"?'?s[|:,]") {
				# FIXME: This is cheated.
				# Fine.

			} else {
				line.logWarning("Unknown wrapper transform command \"${value}\".")
			}
		},

		WrkdirSubdirectory => sub {
			checkline_mk_vartype_basic(line, varname, "Pathname", op, value, comment, list_context, is_guessed)
			if (value eq "\${WRKDIR}") {
				# Fine.
			} else {
				opt_debug_unchecked and line.logDebug("Unchecked subdirectory \"${value}\" of \${WRKDIR}.")
			}
		},

		WrksrcSubdirectory => sub {
			if (value =~ m"^(\$\{WRKSRC\})(?:/(.*))?") {
				my (prefix, rest) = (1, 2)
				line.logNote("You can use \"" . (defined(rest) ? rest : ".") . "\" instead of \"${value}\".")

			} elsif (value ne "" && value_novar eq "") {
				# The value of another variable

			} elsif (value_novar !~ m"^(?:\.|[0-9A-Za-z_\@][-0-9A-Za-z_\@./+]*)$") {
				line.logWarning("\"${value}\" is not a valid subdirectory of \${WRKSRC}.")
			}
		},

		Yes => sub {
			if (value !~ m"^(?:YES|yes)(?:\s+#.*)?$") {
				line.logWarning("${varname} should be set to YES or yes.")
				line.explain_warning(
"This variable means \"yes\" if it is defined, and \"no\" if it is",
"undefined. Even when it has the value \"no\", this means \"yes\".",
"Therefore when it is defined, its value should correspond to its",
"meaning.")
			}
		},

		YesNo => sub {
			if (value !~ m"^(?:YES|yes|NO|no)(?:\s+#.*)?$") {
				line.logWarning("${varname} should be set to YES, yes, NO, or no.")
			}
		},

		YesNo_Indirectly => sub {
			if (value_novar ne "" && value !~ m"^(?:YES|yes|NO|no)(?:\s+#.*)?$") {
				line.logWarning("${varname} should be set to YES, yes, NO, or no.")
			}
		},
	)

	if (ref(type) eq "HASH") {
		if (!exists(type.{value})) {
			line.logWarning("\"${value}\" is not valid for ${varname}. Use one of { ".join(" ", sort(keys(%{type})))." } instead.")
		}

	} elsif (defined type_dispatch{type}) {
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
			line.explain_warning(
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

	if (op eq "+=") {
		if (defined(type)) {
			if (!type.may_use_plus_eq()) {
				line.logWarning("The \"+=\" operator should only be used with lists.")
			}
		} elsif (varbase !~ m"^_" && varbase !~ get_regex_plurals()) {
			line.logWarning("As ${varname} is modified using \"+=\", its name should indicate plural.")
		}
	}

	if (!defined(type)) {
		# Cannot check anything if the type is not known.
		opt_debug_unchecked and line.logDebug("Unchecked variable assignment for ${varname}.")

	} elsif (op eq "!=") {
		opt_debug_misc and line.logDebug("Use of !=: ${value}")

	} elsif (type.kind_of_list != LK_NONE) {
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

	if (op eq "?=" && defined(seen_bsd_prefs_mk) && !seen_bsd_prefs_mk) {
		if (varbase eq "BUILDLINK_PKGSRCDIR"
			|| varbase eq "BUILDLINK_DEPMETHOD"
			|| varbase eq "BUILDLINK_ABI_DEPENDS") {
			# FIXME: What about these ones? They occur quite often.
		} else {
			opt_warn_extra and line.logWarning("Please include \"../../mk/bsd.prefs.mk\" before using \"?=\".")
			opt_warn_extra and line.explain_warning(
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

	# If the variable is not used and is untyped, it may be a
	# spelling mistake.
	if (op eq ":=" && varname eq lc(varname)) {
		opt_debug_unchecked and line.logDebug("${varname} might be unused unless it is an argument to a procedure file.")
		# TODO: check varname against the list of "procedure files".

	} elsif (!var_is_used(varname)) {
		my vartypes = get_vartypes_map()
		my deprecated = get_deprecated_map()

		if (exists(vartypes.{varname}) || exists(vartypes.{varcanon})) {
			# Ok
		} elsif (exists(deprecated.{varname}) || exists(deprecated.{varcanon})) {
			# Ok
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

	if (varname eq "PERL5_PACKLIST" && defined(effective_pkgbase) && effective_pkgbase =~ m"^p5-(.*)") {
		my (guess) = (1)
		guess =~ s/-/\//g
		guess = "auto/${guess}/.packlist"

		my (ucvalue, ucguess) = (uc(value), uc(guess))
		if (ucvalue ne ucguess && ucvalue ne "\${PERL5_SITEARCH\}/${ucguess}") {
			line.logWarning("Unusual value for PERL5_PACKLIST -- \"${guess}\" expected.")
		}
	}

	if (varname eq "CONFIGURE_ARGS" && value =~ m"=\$\{PREFIX\}/share/kde") {
		line.logNote("Please .include \"../../meta-pkgs/kde3/kde3.mk\" instead of this line.")
		line.explain_note(
"That file probably does many things automatically and consistently that",
"this package also does. When using kde3.mk, you can probably also leave",
"out some explicit dependencies.")
	}

	if (varname eq "EVAL_PREFIX" && value =~ m"^([\w_]+)=") {
		my (eval_varname) = (1)

		# The variables mentioned in EVAL_PREFIX will later be
		# defined by find-prefix.mk. Therefore, they are marked
		# as known in the current file.
		mkctx_vardef.{eval_varname} = line
	}

	if (varname eq "PYTHON_VERSIONS_ACCEPTED") {
		checkline_decreasing_order(line, varname, value)
	}

	if (defined(comment) && comment eq "# defined" && varname !~ m".*(?:_MK|_COMMON)$") {
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
			# ok
		} elsif (varname =~ m"^(?:DIST_SUBDIR|WRKSRC)$") {
			line.logWarning("${pkgvarname} should not be used in ${varname}, as it sometimes includes the PKGREVISION. Please use ${pkgvarname}_NOREV instead.")
		} else {
			opt_debug_misc and line.logDebug("Use of PKGNAME in ${varname}.")
		}
	}

	if (exists(get_deprecated_map().{varname})) {
		line.logWarning("Definition of ${varname} is deprecated. ".get_deprecated_map().{varname})
	} elsif (exists(get_deprecated_map().{varcanon})) {
		line.logWarning("Definition of ${varname} is deprecated. ".get_deprecated_map().{varcanon})
	}

	if (varname =~ m"^SITES_") {
		line.logWarning("SITES_* is deprecated. Please use SITES.* instead.")
	}

	if (value =~ m"^[^=]\@comment") {
		line.logWarning("Please don't use \@comment in ${varname}.")
		line.explain_warning(
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

	# Mark the variable as PLIST condition. This is later used in
	# checkfile_PLIST.
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
		#line.logNote("tree_match: varname=varname, match=match")

		my type = get_variable_type(line, varname)
		my btype = defined(type) ? type.basic_type : undef
		if (defined(btype) && ref(type.basic_type) eq "HASH") {
			if (match !~ m"[\$\[*]" && !exists(btype.{match})) {
				line.logWarning("Invalid :M value \"match\". Only { " . join(" ", sort keys %btype) . " } are allowed.")
			}
		}

		# Currently disabled because the valid options can also be defined in PKG_OPTIONS_GROUP.*.
		# Additionally, all these variables may have multiple assigments (+=).
		if (false && varname eq "PKG_OPTIONS" && defined(pkgctx_vardef) && exists(pkgctx_vardef.{"PKG_SUPPORTED_OPTIONS"})) {
			my options = pkgctx_vardef.{"PKG_SUPPORTED_OPTIONS"}.get("value")

			if (match !~ m"[\$\[*]" && index(" options ", " match ") == -1) {
				line.logWarning("Invalid option \"match\". Only { options } are allowed.")
			}
		}

		# TODO: PKG_BUILD_OPTIONS. That requires loading the
		# complete package definitition for the package "pkgbase"
		# or some other database. If we could confine all option
		# definitions to options.mk, this would become easier.

	} elsif (tree_match(tree, [\op, ["var", \varname], ["string", \value]])) {
		checkline_mk_vartype(line, varname, "use", value, undef)

	}
	# XXX: When adding new cases, be careful that the variables may have
	# been partially initialized by previous calls to tree_match.
	# XXX: Maybe it is better to clear these references at the beginning
	# of tree_match.
}

//
// Procedures to check an array of lines.
//

sub checklines_trailing_empty_lines($) {
	my (lines) = @_
	my (last, max)

	max = $#{lines} + 1
	for (last = max; last > 1 && lines.[last - 1].text eq ""; ) {
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

	# Define global variables for the Makefile context.
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

	#
	# In the first pass, all additions to BUILD_DEFS and USE_TOOLS
	# are collected to make the order of the definitions irrelevant.
	#

	foreach my line (@{lines}) {
		next unless line.has("is_varassign")
		my varcanon = line.get("varcanon")

		if (varcanon eq "BUILD_DEFS" || varcanon eq "PKG_GROUPS_VARS" || varcanon eq "PKG_USERS_VARS") {
			foreach my varname (split(qr"\s+", line.get("value"))) {
				mkctx_build_defs.{varname} = true
				opt_debug_misc and line.logDebug("${varname} is added to BUILD_DEFS.")
			}

		} elsif (varcanon eq "PLIST_VARS") {
			foreach my id (split(qr"\s+", line.get("value"))) {
				mkctx_plist_vars.{"PLIST.id"} = true
				opt_debug_misc and line.logDebug("PLIST.${id} is added to PLIST_VARS.")
				use_var(line, "PLIST.id")
			}

		} elsif (varcanon eq "USE_TOOLS") {
			foreach my tool (split(qr"\s+", line.get("value"))) {
				tool =~ s/:(build|run)//
				mkctx_tools.{tool} = true
				opt_debug_misc and line.logDebug("${tool} is added to USE_TOOLS.")
			}

		} elsif (varcanon eq "SUBST_VARS.*") {
			foreach my svar (split(/\s+/, line.get("value"))) {
				use_var(svar, varname_canon(svar))
				opt_debug_misc and line.logDebug("varuse svar")
			}

		} elsif (varcanon eq "OPSYSVARS") {
			foreach my osvar (split(/\s+/, line.get("value"))) {
				use_var(line, "osvar.*")
				def_var(line, osvar)
			}
		}
	}

	#
	# In the second pass, all "normal" checks are done.
	#

	if (0 <= $#{lines}) {
		checkline_rcsid_regex(lines.[0], qr"^#\s+", "# ")
	}

	foreach my line (@{lines}) {
		my text = line.text

		checkline_trailing_whitespace(line)
		checkline_spellcheck(line)

		if (line.has("is_empty")) {
			substcontext.check_end(line)

		} elsif (line.has("is_comment")) {
			# No further checks.

		} elsif (text =~ regex_varassign) {
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

		} elsif (text =~ regex_mk_shellcmd) {
			my (shellcmd) = (1)
			checkline_mk_shellcmd(line, shellcmd)

		} elsif (text =~ regex_mk_include) {
			my (include, includefile) = (1, 2)

			opt_debug_include and line.logDebug("includefile=${includefile}")
			checkline_relative_path(line, includefile, include eq "include")

			if (includefile =~ m"../Makefile$") {
				line.logError("Other Makefiles must not be included directly.")
				line.explain_warning(
"If you want to include portions of another Makefile, extract",
"the common parts and put them into a Makefile.common. After",
"that, both this one and the other package should include the",
"Makefile.common.")
			}

			if (includefile eq "../../mk/bsd.prefs.mk") {
				if (line.fname =~ m"buildlink3\.mk$") {
					line.logNote("For efficiency reasons, please include bsd.fast.prefs.mk instead of bsd.prefs.mk.")
				}
				seen_bsd_prefs_mk = true
			} elsif (includefile eq "../../mk/bsd.fast.prefs.mk") {
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

		} elsif (text =~ regex_mk_sysinclude) {
			my (includefile, comment) = (1, 2)

			# No further action.

		} elsif (text =~ regex_mk_cond) {
			my (indent, directive, args, comment) = (1, 2, 3, 4)

			use constant regex_directives_with_args => qr"^(?:if|ifdef|ifndef|elif|for|undef)$"

			if (directive =~ m"^(?:endif|endfor|elif|else)$") {
				if ($#{mkctx_indentations} >= 1) {
					pop(@{mkctx_indentations})
				} else {
					line.logError("Unmatched .${directive}.")
				}
			}

			# Check the indentation
			if (indent ne " " x mkctx_indentations.[-1]) {
				opt_warn_space and line.logNote("This directive should be indented by ".mkctx_indentations.[-1]." spaces.")
			}

			if (directive eq "if" && args =~ m"^!defined\([\w]+_MK\)$") {
				push(@{mkctx_indentations}, mkctx_indentations.[-1])

			} elsif (directive =~ m"^(?:if|ifdef|ifndef|for|elif|else)$") {
				push(@{mkctx_indentations}, mkctx_indentations.[-1] + 2)
			}

			if (directive =~ regex_directives_with_args && !defined(args)) {
				line.logError("\".${directive}\" must be given some arguments.")

			} elsif (directive !~ regex_directives_with_args && defined(args)) {
				line.logError("\".${directive}\" does not take arguments.")

				if (directive eq "else") {
					line.logNote("If you meant \"else if\", use \".elif\".")
				}

			} elsif (directive eq "if" || directive eq "elif") {
				checkline_mk_cond(line, args)

			} elsif (directive eq "ifdef" || directive eq "ifndef") {
				if (args =~ m"\s") {
					line.logError("The \".${directive}\" directive can only handle _one_ argument.")
				} else {
					line.logWarning("The \".${directive}\" directive is deprecated. Please use \".if "
						. ((directive eq "ifdef" ? "" : "!"))
						. "defined(${args})\" instead.")
				}

			} elsif (directive eq "for") {
				if (args =~ m"^(\S+(?:\s*\S+)*?)\s+in\s+(.*)$") {
					my (vars, values) = (1, 2)

					foreach my var (split(qr"\s+", vars)) {
						if (!is_internal && var =~ m"^_") {
							line.logWarning("Variable names starting with an underscore are reserved for internal pkgsrc use.")
						}

						if (var =~ m"^[_a-z][_a-z0-9]*$") {
							# Fine.
						} elsif (var =~ m"[A-Z]") {
							line.logWarning(".for variable names should not contain uppercase letters.")
						} else {
							line.logError("Invalid variable name \"${var}\".")
						}

						mkctx_for_variables.{var} = true
					}

					# Check if any of the value's types is not guessed.
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

			} elsif (directive eq "undef" && defined(args)) {
				foreach my var (split(qr"\s+", args)) {
					if (exists(mkctx_for_variables.{var})) {
						line.logNote("Using \".undef\" after a \".for\" loop is unnecessary.")
					}
				}
			}

		} elsif (text =~ regex_mk_dependency) {
			my (targets, whitespace, dependencies, comment) = (1, 2, 3, 4)

			opt_debug_misc and line.logDebug("targets=${targets}, dependencies=${dependencies}")
			mkctx_target = targets

			foreach my source (split(/\s+/, dependencies)) {
				if (source eq ".PHONY") {
					foreach my target (split(/\s+/, targets)) {
						allowed_targets.{target} = true
					}
				}
			}

			foreach my target (split(/\s+/, targets)) {
				if (target eq ".PHONY") {
					foreach my dep (split(qr"\s+", dependencies)) {
						allowed_targets.{dep} = true
					}

				} elsif (target eq ".ORDER") {
					# TODO: Check for spelling mistakes.

				} elsif (!exists(allowed_targets.{target})) {
					line.logWarning("Unusual target \"${target}\".")
					line.explain_warning(
"If you really want to define your own targets, you can \"declare\"",
"them by inserting a \".PHONY: my-target\" line before this line. This",
"will tell make(1) to not interpret this target's name as a filename.")
				}
			}

		} elsif (text =~ m"^\.\s*(\S*)") {
			my (directive) = (1)

			line.logError("Unknown directive \".${directive}\".")

		} elsif (text =~ m"^ ") {
			line.logWarning("Makefile lines should not start with space characters.")
			line.explain_warning(
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

	# Clean up global variables.
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

	# Collect all the included buildlink3.mk files from the file.
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

	# Print debugging messages for all buildlink3.mk files that are
	# included by the package but not by this buildlink3.mk file.
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

	# Header comments
	while (lineno <= $#{lines} && (my text = lines.[lineno].text) =~ m"^#") {
		if (text =~ m"^# XXX") {
			lines.[lineno].logNote("Please read this comment and remove it if appropriate.")
		}
		lineno++
	}
	expect_empty_line(lines, \lineno)

	if (expect(lines, \lineno, qr"^BUILDLINK_DEPMETHOD\.(\S+)\?=.*$")) {
		lines.[lineno - 1].logWarning("This line belongs inside the .ifdef block.")
		while (lines.[lineno].text eq "") {
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

	# First paragraph: Introduction of the package identifier
	bl_pkgbase_line = lines.[lineno - 1]
	bl_pkgbase = pkgid
	opt_debug_misc and bl_pkgbase_line.logDebug("bl_pkgbase=${bl_pkgbase}")
	expect_empty_line(lines, \lineno)

	# Second paragraph: multiple inclusion protection and introduction
	# of the uppercase package identifier.
	return unless (m = expect_re(lines, \lineno, qr"^\.if !defined\((\S+)_BUILDLINK3_MK\)$"))
	bl_PKGBASE_line = lines.[lineno - 1]
	bl_PKGBASE = m.text(1)
	opt_debug_misc and bl_PKGBASE_line.logDebug("bl_PKGBASE=${bl_PKGBASE}")
	expect_re(lines, \lineno, qr"^\Qbl_PKGBASE\E_BUILDLINK3_MK:=$")
	expect_empty_line(lines, \lineno)

	my norm_bl_pkgbase = bl_pkgbase
	norm_bl_pkgbase =~ s/-/_/g
	norm_bl_pkgbase = uc(norm_bl_pkgbase)
	if (norm_bl_pkgbase ne bl_PKGBASE) {
		bl_PKGBASE_line.logError("Package name mismatch between ${bl_PKGBASE} ...")
		bl_pkgbase_line.logError("... and ${bl_pkgbase}.")
	}
	if (defined(effective_pkgbase) && effective_pkgbase ne bl_pkgbase) {
		bl_pkgbase_line.logError("Package name mismatch between ${bl_pkgbase} ...")
		effective_pkgname_line.logError("... and ${effective_pkgbase}.")
	}

	# Third paragraph: Package information.
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

			if (varname eq "BUILDLINK_ABI_DEPENDS.${bl_pkgbase}") {
				abi_line = line
				if (value =~ regex_dependency_lge) {
					(abi_pkg, abi_version) = (1, 2)
				} elsif (value =~ regex_dependency_wildcard) {
					(abi_pkg) = (1)
				} else {
					opt_debug_unchecked and line.logDebug("Unchecked dependency pattern \"${value}\".")
				}
				do_check = true
			}
			if (varname eq "BUILDLINK_API_DEPENDS.${bl_pkgbase}") {
				api_line = line
				if (value =~ regex_dependency_lge) {
					(api_pkg, api_version) = (1, 2)
				} elsif (value =~ regex_dependency_wildcard) {
					(api_pkg) = (1)
				} else {
					opt_debug_unchecked and line.logDebug("Unchecked dependency pattern \"${value}\".")
				}
				do_check = true
			}
			if (do_check && defined(abi_pkg) && defined(api_pkg)) {
				if (abi_pkg ne api_pkg) {
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

				if (varparam ne bl_pkgbase) {
					line.logWarning("Only buildlink variables for ${bl_pkgbase}, not ${varparam} may be set in this file.")
				}
			}

			if (varname eq "pkgbase") {
				expect_re(lines, \lineno, qr"^\.\s*include \"../../mk/pkg-build-options.mk\"$")
			}

			# TODO: More checks.

		} elsif (expect(lines, \lineno, qr"^(?:#.*)?$")) {
			# Comments and empty lines are fine here.

		} elsif (expect(lines, \lineno, qr"^\.\s*include \"\.\./\.\./([^/]+/[^/]+)/buildlink3\.mk\"$")
			|| expect(lines, \lineno, qr"^\.\s*include \"\.\./\.\./mk/(\S+)\.buildlink3\.mk\"$")) {
			# TODO: Maybe check dependency lines.

		} elsif (expect(lines, \lineno, qr"^\.if\s")) {
			if_level++

		} elsif (expect(lines, \lineno, qr"^\.endif.*$")) {
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

	# Fourth paragraph: Cleanup, corresponding to the first paragraph.
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
		line.explain_warning(
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
	if (1 <= $#{lines} && lines.[1].text ne "") {
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
				if (seen_algs ne "SHA1") {
					line.logError("Expected SHA1 checksum for ${current_fname}, got ${seen_algs}.")
				}
			} else {
				if (seen_algs ne "SHA1, RMD160, Size" && seen_algs ne "SHA1, RMD160, SHA512, Size") {
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

		if (!defined(current_fname) || chksum_fname ne current_fname) {
			on_filename_change.(line, chksum_fname)
		}

		if (chksum_fname !~ m"^\w") {
			line.logError("All file names should start with a letter.")
		}

		# Inter-package check for differing distfile checksums.
		if (opt_check_global && !is_patch) {
			# Note: Perl-specific auto-population.
			if (exists(ipc_distinfo.{alg}.{chksum_fname})) {
				my other = ipc_distinfo.{alg}.{chksum_fname}

				if (other.[1] eq sum) {
					# Fine.
				} else {
					line.logError("The ${alg} checksum for ${chksum_fname} differs ...")
					other.[0].logError("... from this one.")
				}
			} else {
				ipc_distinfo.{alg}.{chksum_fname} = [line, sum]
			}
		}

		push(@seen_algs, alg)

		if (is_patch && defined(patches_dir) && !(defined(distinfo_file) && distinfo_file eq "./../../lang/php5/distinfo")) {
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
				if (sum ne chksum) {
					line.logError("${alg} checksum of ${chksum_fname} differs (expected ${sum}, got ${chksum}). Rerun '".conf_make." makepatchsum'.")
				}
			} else {
				line.logWarning("${chksum_fname} does not exist.")
				line.explain_warning(
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
		explain_warning(fname, NO_LINE_NUMBER, @explanation)
		return
	}
	if (lines.[0].text ne "=" x 75) {
		lines.[0].logWarning("Expected a line of exactly 75 \"=\" characters.")
		explain_warning(fname, NO_LINE_NUMBER, @explanation)
	}
	checkline_rcsid(lines.[1], "")
	foreach my line (@{lines}) {
		checkline_length(line, 80)
		checkline_trailing_whitespace(line)
		checkline_valid_characters(line, regex_validchars)
		checkline_spellcheck(line)
	}
	if (lines.[-1].text ne "=" x 75) {
		lines.[-1].logWarning("Expected a line of exactly 75 \"=\" characters.")
		explain_warning(fname, NO_LINE_NUMBER, @explanation)
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
			# Don't emit a warning, since the comment
			# probably contains a statement that C is
			# really not needed.

		} elsif (value !~ m"(?:^|\s+)(?:c|c99|objc)(?:\s+|$)") {
			pkgctx_vardef.{"GNU_CONFIGURE"}.logWarning("GNU_CONFIGURE almost always needs a C compiler, ...")
			languages_line.logWarning("... but \"c\" is not added to USE_LANGUAGES.")
		}
	}

	my distname_line = pkgctx_vardef.{"DISTNAME"}
	my pkgname_line = pkgctx_vardef.{"PKGNAME"}

	my distname = defined(distname_line) ? distname_line.get("value") : undef
	my pkgname = defined(pkgname_line) ? pkgname_line.get("value") : undef
	my nbpart = get_nbpart()

	# Let's do some tricks to get the proper value of the package
	# name more often.
	if (defined(distname) && defined(pkgname)) {
		pkgname =~ s/\$\{DISTNAME\}/distname/

		if (pkgname =~ m"^(.*)\$\{DISTNAME:S(.)([^:]*)\2([^:]*)\2(g?)\}(.*)$") {
			my (before, separator, old, new, mod, after) = (1, 2, 3, 4, 5, 6)
			my newname = distname
			old = quotemeta(old)
			old =~ s/^\\\^/^/
			old =~ s/\\\$$/\$/
			if (mod eq "g") {
				newname =~ s/old/new/g
			} else {
				newname =~ s/old/new/
			}
			opt_debug_misc and pkgname_line.logDebug("old pkgname=pkgname")
			pkgname = before . newname . after
			opt_debug_misc and pkgname_line.logDebug("new pkgname=pkgname")
		}
	}

	if (defined(pkgname) && defined(distname) && pkgname eq distname) {
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
		# XXX: too many false positives
		if (false && pkgpath =~ m"/([^/]+)$" && effective_pkgbase ne 1) {
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

			next unless effective_pkgbase eq suggbase

			if (dewey_cmp(effective_pkgversion, "<", suggver)) {
				effective_pkgname_line.logWarning("This package should be updated to ${suggver}${comment}.")
				effective_pkgname_line.explain_warning(
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
		if (basename eq "files" || basename eq "patches" || basename eq "CVS") {
			# Ok
		} elsif (fname =~ m"(?:^|/)files/[^/]*$") {
			# Ok

		} elsif (!is_emptydir(fname)) {
			logWarning(fname, NO_LINE_NUMBER, "Unknown directory name.")
		}

	} elsif (S_ISLNK(st.mode)) {
		if (basename !~ m"^work") {
			logWarning(fname, NO_LINE_NUMBER, "Unknown symlink name.")
		}

	} elsif (!S_ISREG(st.mode)) {
		logError(fname, NO_LINE_NUMBER, "Only files and directories are allowed in pkgsrc.")

	} elsif (basename eq "ALTERNATIVES") {
		opt_check_ALTERNATIVES and checkfile_ALTERNATIVES(fname)

	} elsif (basename eq "buildlink3.mk") {
		opt_check_bl3 and checkfile_buildlink3_mk(fname)

	} elsif (basename =~ m"^DESCR") {
		opt_check_DESCR and checkfile_DESCR(fname)

	} elsif (basename =~ m"^distinfo") {
		opt_check_distinfo and checkfile_distinfo(fname)

	} elsif (basename eq "DEINSTALL" || basename eq "INSTALL") {
		opt_check_INSTALL and checkfile_INSTALL(fname)

	} elsif (basename =~ m"^MESSAGE") {
		opt_check_MESSAGE and checkfile_MESSAGE(fname)

	} elsif (basename =~ m"^patch-[-A-Za-z0-9_.~+]*[A-Za-z0-9_]$") {
		opt_check_patches and checkfile_patch(fname)

	} elsif (fname =~ m"(?:^|/)patches/manual[^/]*$") {
		opt_debug_unchecked and logDebug(fname, NO_LINE_NUMBER, "Unchecked file \"${fname}\".")

	} elsif (fname =~ m"(?:^|/)patches/[^/]*$") {
		logWarning(fname, NO_LINE_NUMBER, "Patch files should be named \"patch-\", followed by letters, '-', '_', '.', and digits only.")

	} elsif (basename =~ m"^(?:.*\.mk|Makefile.*)$" and not fname =~ m,files/, and not fname =~ m,patches/,) {
		opt_check_mk and checkfile_mk(fname)

	} elsif (basename =~ m"^PLIST") {
		opt_check_PLIST and checkfile_PLIST(fname)

	} elsif (basename eq "TODO" || basename eq "README") {
		# Ok

	} elsif (basename =~ m"^CHANGES-.*") {
		load_doc_CHANGES(fname)

	} elsif (!-T fname) {
		logWarning(fname, NO_LINE_NUMBER, "Unexpectedly found a binary file.")

	} elsif (fname =~ m"(?:^|/)files/[^/]*$") {
		# Ok
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
		next if (type eq "D" && !defined(fname))
		assert(false, "Unknown line format: " . line.text)
			unless type eq "" || type eq "D"
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

	# The first line must contain the RCS Id
	if (lineno <= $#{lines} && checkline_rcsid_regex(lines.[lineno], qr"#\s+", "# ")) {
		lineno++
	}

	# Then, arbitrary comments may follow
	while (lineno <= $#{lines} && lines.[lineno].text =~ m"^#") {
		lineno++
	}

	# Then we need an empty line
	expect_empty_line(lines, \lineno)

	# Then comes the COMMENT line
	if (lineno <= $#{lines} && lines.[lineno].text =~ m"^COMMENT=\t*(.*)") {
		my (comment) = (1)

		checkline_valid_characters_in_variable(lines.[lineno], qr"[-\040'(),/0-9A-Za-z]")
		lineno++
	} else {
		lines.[lineno].logError("COMMENT= line expected.")
	}

	# Then we need an empty line
	expect_empty_line(lines, \lineno)

	# And now to the most complicated part of the category Makefiles,
	# the (hopefully) sorted list of SUBDIRs. The first step is to
	# collect the SUBDIRs in the Makefile and in the file system.

	my (@f_subdirs, @m_subdirs)

	@f_subdirs = sort(get_subdirs(current_dir))

	my prev_subdir = undef
	while (lineno <= $#{lines}) {
		my line = lines.[lineno]

		if (line.text =~ m"^(#?)SUBDIR\+=(\s*)(\S+)\s*(?:#\s*(.*?)\s*|)$") {
			my (comment_flag, indentation, subdir, comment) = (1, 2, 3, 4)

			if (comment_flag eq "#" && (!defined(comment) || comment eq "")) {
				line.logWarning("${subdir} commented out without giving a reason.")
			}

			if (indentation ne "\t") {
				line.logWarning("Indentation should be a single tab character.")
			}

			if (defined(prev_subdir) && subdir eq prev_subdir) {
				line.logError("${subdir} must only appear once.")
			} elsif (defined(prev_subdir) && subdir lt prev_subdir) {
				line.logWarning("${subdir} should come before ${prev_subdir}.")
			} else {
				# correctly ordered
			}

			push(@m_subdirs, [subdir, line, comment_flag ? false : true])
			prev_subdir = subdir
			lineno++

		} else {
			if (line.text ne "") {
				line.logError("SUBDIR+= line or empty line expected.")
			}
			last
		}
	}

	# To prevent unnecessary warnings about subdirectories that are
	# in one list, but not in the other, we generate the sets of
	# subdirs of each list.
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

		} elsif (!m_atend && (f_atend || m_current lt f_current)) {
			if (!exists(f_check{m_current})) {
				line.logError("${m_current} exists in the Makefile, but not in the file system.")
				line.delete()
			}
			m_neednext = true

		} else { # f_current eq m_current
			f_neednext = true
			m_neednext = true
			if (m_recurse) {
				push(@subdirs, "${current_dir}/${m_current}")
			}
		}
	}

	# the wip category Makefile may have its own targets for generating
	# indexes and READMEs. Just skip them.
	if (is_wip) {
		while (lineno <= $#{lines} - 2) {
			lineno++
		}
	}

	expect_empty_line(lines, \lineno)

	# And, last but not least, the .include line
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

	# Initialize global variables
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

	# we need to handle the Makefile first to get some variables
	if (!load_package_Makefile("${current_dir}/Makefile", \lines)) {
		logError("${current_dir}/Makefile", NO_LINE_NUMBER, "Cannot be read.")
		goto cleanup
	}

	my @files = glob("${current_dir}/*")
	if (pkgdir ne ".") {
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

	# Determine the used variables before checking any of the
	# Makefile fragments.
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
		if (fname eq "${current_dir}/Makefile") {
			opt_check_Makefile and checkfile_package_Makefile(fname, lines)
		} else {
			checkfile(fname)
		}
		if (fname =~ m"/patches/patch-*$") {
			have_patches = true
		} elsif (fname =~ m"/distinfo$") {
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
	# Clean up global variables.
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
