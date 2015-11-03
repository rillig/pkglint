sub resolve_variable_rec1($$);
sub resolve_variable_rec2($$);

sub resolve_variable_rec1($$) {
	my ($varname, $visited) = @_;
	$opt_debug_trace and log_debug(NO_FILE, NO_LINES, "resolve_variable_rec1($varname)");

	if (!exists($visited->{$varname})) {
		$visited->{$varname} = true;
		if (defined($pkgctx_vardef) && exists($pkgctx_vardef->{$varname})) {
			return resolve_variable_rec2($pkgctx_vardef->{$varname}->get("value"), $visited);
		}
		if (defined($mkctx_vardef) && exists($mkctx_vardef->{$varname})) {
			return resolve_variable_rec2($mkctx_vardef->{$varname}->get("value"), $visited);
		}
	}
	return "\${$varname}";
}

sub resolve_variable_rec2($$) {
	my ($string, $visited) = @_;
	$opt_debug_trace and log_debug(NO_FILE, NO_LINES, "resolve_variable_rec2(\"$string\")");

	my $expanded = $string;
	$expanded =~ s/\$\{(\w+)\}/resolve_variable_rec1($1, $visited)/eg;
	return $expanded;
}

sub expand_variable($) {
	my ($varname) = @_;

	return unless exists($pkgctx_vardef->{$varname});
	my $line = $pkgctx_vardef->{$varname};
	my $value = $line->get("value");

	$value = resolve_relative_path($value, true);
	if ($value =~ regex_unresolved) {
		$opt_debug_misc and log_debug(NO_FILE, NO_LINES, "[expand_variable] Trying harder to resolve variable references in ${varname}=\"${value}\".");
		$value = resolve_variable_rec2($value, {});
		if ($value =~ regex_unresolved) {
			$opt_debug_misc and log_debug(NO_FILE, NO_LINES, "[expand_variable] Failed to resolve ${varname}=\"${value}\".");
		}
	}
	return $value;
}

sub set_default_value($$) {
	my ($varref, $value) = @_;

	if (!defined(${$varref}) || ${$varref} =~ regex_unresolved) {
		${$varref} = $value;
	}
}

sub shell_split($) {
	my ($text) = @_;
	my ($words);

	$words = [];
	while ($text =~ s/^$regex_shellword//) {
		push(@{$words}, $1);
	}
	return (($text =~ m"^\s*$") ? $words : false);
}

# When processing a file using the expect* subroutines below, it may
# happen that $lineno points past the end of the file. In that case,
# print the warning without associated source code.
sub lines_log_warning($$$) {
	my ($lines, $lineno, $msg) = @_;

	assert(false, "The line number is negative (${lineno}).")
		unless 0 <= $lineno;
	assert(@{$lines} != 0, "The lines may not be empty.");

	if ($lineno <= $#{$lines}) {
		$lines->[$lineno]->log_warning($msg);
	} else {
		log_warning($lines->[0]->fname, "EOF", $msg);
	}
}

# Checks if the current line ($lines->{${$lineno_ref}}) matches the
# regular expression, and if it does, increments ${${lineno_ref}}.
# @param $lines
#	The lines that are checked.
# @param $lineno_ref
#	A reference to the line number, an integer variable.
# @param $regex
#	The regular expression to be checked.
# @return
#	The result of the regular expression match or false.
sub expect($$$) {
	my ($lines, $lineno_ref, $regex) = @_;
	my $lineno = ${$lineno_ref};

	if ($lineno <= $#{$lines} && $lines->[$lineno]->text =~ $regex) {
		${$lineno_ref}++;
		return PkgLint::SimpleMatch->new($lines->[$lineno]->text, \@-, \@+);
	} else {
		return false;
	}
}

sub expect_empty_line($$) {
	my ($lines, $lineno_ref) = @_;

	if (expect($lines, $lineno_ref, qr"^$")) {
		return true;
	} else {
		$opt_warn_space and $lines->[${$lineno_ref}]->log_note("Empty line expected.");
		return false;
	}
}

sub expect_text($$$) {
	my ($lines, $lineno_ref, $text) = @_;

	my $rv = expect($lines, $lineno_ref, qr"^\Q${text}\E$");
	$rv or lines_log_warning($lines, ${$lineno_ref}, "Expected \"${text}\".");
	return $rv;
}

sub expect_re($$$) {
	my ($lines, $lineno_ref, $re) = @_;

	my $rv = expect($lines, $lineno_ref, $re);
	$rv or lines_log_warning($lines, ${$lineno_ref}, "Expected text matching $re.");
	return $rv;
}


sub get_variable_perms($$) {
	my ($line, $varname) = @_;

	my $type = get_variable_type($line, $varname);
	if (!defined($type)) {
		$opt_debug_misc and $line->log_debug("No type definition found for ${varname}.");
		return "adpsu";
	}

	my $perms = $type->perms($line->fname, $varname);
	if (!defined($perms)) {
		$opt_debug_misc and $line->log_debug("No permissions specified for ${varname}.");
		return "?";
	}
	return $perms;
}

#
# Parsing.
#

# Checks whether $tree matches $pattern, and if so, instanciates the
# variables in $pattern. If they don't match, some variables may be
# instanciated nevertheless, but the exact behavior is unspecified.
#
sub tree_match($$);
sub tree_match($$) {
	my ($tree, $pattern) = @_;

	my $d1 = Data::Dumper->new([$tree, $pattern])->Terse(true)->Indent(0);
	my $d2 = Data::Dumper->new([$pattern])->Terse(true)->Indent(0);
	$opt_debug_trace and log_debug(NO_FILE, NO_LINES, sprintf("tree_match(%s, %s)", $d1->Dump, $d2->Dump));

	return true if (!defined($tree) && !defined($pattern));
	return false if (!defined($tree) || !defined($pattern));
	my $aref = ref($tree);
	my $pref = ref($pattern);
	if ($pref eq "SCALAR" && !defined($$pattern)) {
		$$pattern = $tree;
		return true;
	}
	if ($aref eq "" && ($pref eq "" || $pref eq "SCALAR")) {
		return $tree eq $pattern;
	}
	if ($aref eq "ARRAY" && $pref eq "ARRAY") {
		return false if scalar(@$tree) != scalar(@$pattern);
		for (my $i = 0; $i < scalar(@$tree); $i++) {
			return false unless tree_match($tree->[$i], $pattern->[$i]);
		}
		return true;
	}
	return false;
}

# TODO: Needs to be redesigned to handle more complex expressions.
sub parse_mk_cond($$);
sub parse_mk_cond($$) {
	my ($line, $cond) = @_;

	$opt_debug_trace and $line->log_debug("parse_mk_cond(\"${cond}\")");

	my $re_simple_varname = qr"[A-Z_][A-Z0-9_]*(?:\.[\w_+\-]+)?";
	while ($cond ne "") {
		if ($cond =~ s/^!//) {
			return ["not", parse_mk_cond($line, $cond)];
		} elsif ($cond =~ s/^defined\((${re_simple_varname})\)$//) {
			return ["defined", $1];
		} elsif ($cond =~ s/^empty\((${re_simple_varname})\)$//) {
			return ["empty", $1];
		} elsif ($cond =~ s/^empty\((${re_simple_varname}):M([^\$:{})]+)\)$//) {
			return ["empty", ["match", $1, $2]];
		} elsif ($cond =~ s/^\$\{(${re_simple_varname})\}\s+(==|!=)\s+"([^"\$\\]*)"$//) { #"
			return [$2, ["var", $1], ["string", $3]];
		} else {
			$opt_debug_unchecked and $line->log_debug("parse_mk_cond: ${cond}");
			return ["unknown", $cond];
		}
	}
}

sub parse_licenses($) {
	my ($licenses) = @_;

	$licenses =~ s,\$\{PERL5_LICENSE},gnu-gpl-v2 OR artistic,g;
	$licenses =~ s,[()]|AND|OR,,g; # XXX: treats OR like AND
	my @licenses = split(/\s+/, $licenses);
	return \@licenses;
}

sub warn_about_PLIST_imake_mannewsuffix($) {
	my ($line) = @_;

	$line->log_warning("IMAKE_MANNEWSUFFIX is not meant to appear in PLISTs.");
	$line->explain_warning(
"This is the result of a print-PLIST call that has _not_ been checked",
"thoroughly by the developer. Please replace the IMAKE_MANNEWSUFFIX with",
"",
"\tIMAKE_MAN_SUFFIX for programs,",
"\tIMAKE_LIBMAN_SUFFIX for library functions,",
"\tIMAKE_FILEMAN_SUFFIX for file formats,",
"\tIMAKE_GAMEMAN_SUFFIX for games,",
"\tIMAKE_MISCMAN_SUFFIX for other man pages.");
}

#
# Subroutines to check a single line.
#

sub checkline_length($$) {
	my ($line, $maxlength) = @_;

	if (length($line->text) > $maxlength) {
		$line->log_warning("Line too long (should be no more than $maxlength characters).");
		$line->explain_warning(
"Back in the old time, terminals with 80x25 characters were common.",
"And this is still the default size of many terminal emulators.",
"Moderately short lines also make reading easier.");
	}
}

sub checkline_valid_characters($$) {
	my ($line, $re_validchars) = @_;
	my ($rest);

	($rest = $line->text) =~ s/$re_validchars//g;
	if ($rest ne "") {
		my @chars = map { sprintf("0x%02x", ord($_)); } split(//, $rest);
		$line->log_warning("Line contains invalid characters (" . join(", ", @chars) . ").");
	}
}

sub checkline_valid_characters_in_variable($$) {
	my ($line, $re_validchars) = @_;
	my ($varname, $rest);

	$varname = $line->get("varname");
	$rest = $line->get("value");

	$rest =~ s/$re_validchars//g;
	if ($rest ne "") {
		my @chars = map { sprintf("0x%02x", ord($_)); } split(//, $rest);
		$line->log_warning("${varname} contains invalid characters (" . join(", ", @chars) . ").");
	}
}

sub checkline_trailing_whitespace($) {
	my ($line) = @_;

	$opt_debug_trace and $line->log_debug("checkline_trailing_whitespace()");

	if ($line->text =~ /\s+$/) {
		$line->log_note("Trailing white-space.");
		$line->explain_note(
"When a line ends with some white-space, that space is in most cases",
"irrelevant and can be removed, leading to a \"normal form\" syntax.",
"",
"Note: This is mostly for aesthetic reasons.");
		$line->replace_regex(qr"\s+\n$", "\n");
	}
}

sub checkline_rcsid_regex($$$) {
	my ($line, $prefix_regex, $prefix) = @_;
	my ($id) = ($opt_rcsidstring . ($is_wip ? "|Id" : ""));

	$opt_debug_trace and $line->log_debug("checkline_rcsid_regex(${prefix_regex}, ${prefix})");

	if ($line->text !~ m"^${prefix_regex}\$(${id})(?::[^\$]+|)\$$") {
		$line->log_error("\"${prefix}\$${opt_rcsidstring}\$\" expected.");
		$line->explain_error(
"Several files in pkgsrc must contain the CVS Id, so that their current",
"version can be traced back later from a binary package. This is to",
"ensure reproducible builds, for example for finding bugs.",
"",
"Please insert the text from the above error message (without the quotes)",
"at this position in the file.");
		return false;
	}
	return true;
}

sub checkline_rcsid($$) {
	my ($line, $prefix) = @_;

	checkline_rcsid_regex($line, quotemeta($prefix), $prefix);
}

sub checkline_mk_absolute_pathname($$) {
	my ($line, $text) = @_;
	my $abspath;

	$opt_debug_trace and $line->log_debug("checkline_mk_absolute_pathname(${text})");

	# In the GNU coding standards, DESTDIR is defined as a (usually
	# empty) prefix that can be used to install files to a different
	# location from what they have been built for. Therefore
	# everything following it is considered an absolute pathname.
	# Another commonly used context is in assignments like
	# "bindir=/bin".
	if ($text =~ m"(?:^|\$\{DESTDIR\}|\$\(DESTDIR\)|[\w_]+\s*=\s*)(/(?:[^\"'\`\s]|\"[^\"*]\"|'[^']*'|\`[^\`]*\`)*)") {
		my $path = $1;

		if ($path =~ m"^/\w") {
			$abspath = $path;
		}
	}

	if (defined($abspath)) {
		checkword_absolute_pathname($line, $abspath);
	}
}

sub checkline_relative_path($$$) {
	my ($line, $path, $must_exist) = @_;
	my ($res_path);

	if (!$is_wip && $path =~ m"/wip/") {
		$line->log_error("A pkgsrc package must not depend on any outside package.");
	}
	$res_path = resolve_relative_path($path, true);
	if ($res_path =~ regex_unresolved) {
		$opt_debug_unchecked and $line->log_debug("Unchecked path: \"${path}\".");
	} elsif (!-e ((($res_path =~ m"^/") ? "" : "${current_dir}/") . $res_path)) {
		$must_exist and $line->log_error("\"${res_path}\" does not exist.");
	} elsif ($path =~ m"^\.\./\.\./([^/]+)/([^/]+)(.*)") {
		my ($cat, $pkg, $rest) = ($1, $2, $3);
	} elsif ($path =~ m"^\.\./\.\./mk/") {
		# There need not be two directory levels for mk/ files.
	} elsif ($path =~ m"^\.\./mk/" && $cur_pkgsrcdir eq "..") {
		# That's fine for category Makefiles.
	} elsif ($path =~ m"^\.\.") {
		$line->log_warning("Invalid relative path \"${path}\".");
	}
}

sub checkline_relative_pkgdir($$) {
	my ($line, $path) = @_;

	checkline_relative_path($line, $path, true);
	$path = resolve_relative_path($path, false);

	if ($path =~ m"^(?:\./)?\.\./\.\./([^/]+/[^/]+)$") {
		my $otherpkgpath = $1;
		if (! -f "$cwd_pkgsrcdir/$otherpkgpath/Makefile") {
			$line->log_error("There is no package in $otherpkgpath.");
		}

	} else {
		$line->log_warning("\"${path}\" is not a valid relative package directory.");
		$line->explain_warning(
"A relative pathname always starts with \"../../\", followed",
"by a category, a slash and a the directory name of the package.",
"For example, \"../../misc/screen\" is a valid relative pathname.");
	}
}

sub checkline_spellcheck($) {
	my ($line) = @_;

	if ($line->text =~ m"existant") {
		$line->log_warning("The word \"existant\" is nonexistent in the m-w dictionary.");
		$line->explain_warning("Please use \"existent\" instead.");
	}
}

sub checkline_mk_varuse($$$$) {
	my ($line, $varname, $mod, $context) = @_;

	assert(defined($varname), "The varname parameter must be defined");
	assert(defined($context), "The context parameter must be defined");
	$opt_debug_trace and $line->log_debug("checkline_mk_varuse(\"${varname}\", \"${mod}\", ".$context->to_string().")");

	# Check for spelling mistakes.
	my $type = get_variable_type($line, $varname);
	if (defined($type) && !($type->is_guessed)) {
		# Great.

	} elsif (var_is_used($varname)) {
		# Fine.

	} elsif (defined($mkctx_for_variables) && exists($mkctx_for_variables->{$varname})) {
		# Variables defined in .for loops are also ok.

	} else {
		$opt_warn_extra and $line->log_warning("${varname} is used but not defined. Spelling mistake?");
	}

	if ($opt_warn_perm) {
		my $perms = get_variable_perms($line, $varname);
		my $is_load_time;	# Will the variable be used at load time?
		my $is_indirect;	# Might the variable be used indirectly at load time,
					# for example by assigning it to another variable
					# which then gets evaluated?

		# Don't warn about variables that are not recorded in the
		# pkglint variable definition.
		if (defined($context->type) && $context->type->is_guessed()) {
			$is_load_time = false;

		} elsif ($context->time == VUC_TIME_LOAD && $perms !~ m"p") {
			$is_load_time = true;
			$is_indirect = false;

		} elsif (defined($context->type) && $context->type->perms_union() =~ m"p" && $perms !~ m"p") {
			$is_load_time = true;
			$is_indirect = true;

		} else {
			$is_load_time = false;
		}

		if ($is_load_time && !$is_indirect) {
			$line->log_warning("${varname} should not be evaluated at load time.");
			$line->explain_warning(
"Many variables, especially lists of something, get their values",
"incrementally. Therefore it is generally unsafe to rely on their value",
"until it is clear that it will never change again. This point is",
"reached when the whole package Makefile is loaded and execution of the",
"shell commands starts, in some cases earlier.",
"",
"Additionally, when using the \":=\" operator, each \$\$ is replaced",
"with a single \$, so variables that have references to shell variables",
"or regular expressions are modified in a subtle way.");
		}
		if ($is_load_time && $is_indirect) {
			$line->log_warning("${varname} should not be evaluated indirectly at load time.");
			$line->explain_warning(
"The variable on the left-hand side may be evaluated at load time, but",
"the variable on the right-hand side may not. Due to this assignment, it",
"might be used indirectly at load-time, when it is not guaranteed to be",
"properly defined.");
		}

		if ($perms !~ m"p" && $perms !~ m"u") {
			$line->log_warning("${varname} may not be used in this file.");
		}
	}

	if ($varname eq "LOCALBASE" && !$is_internal) {
		$line->log_warning("The LOCALBASE variable should not be used by packages.");
		$line->explain_warning(
# from jlam via private mail.
"Currently, LOCALBASE is typically used in these cases:",
"",
"(1) To locate a file or directory from another package.",
"(2) To refer to own files after installation.",
"",
"In the first case, the example is:",
"",
"	STRLIST=        \${LOCALBASE}/bin/strlist",
"	do-build:",
"		cd \${WRKSRC} && \${STRLIST} *.str",
"",
"This should really be:",
"",
"	EVAL_PREFIX=    STRLIST_PREFIX=strlist",
"	STRLIST=        \${STRLIST_PREFIX}/bin/strlist",
"	do-build:",
"		cd \${WRKSRC} && \${STRLIST} *.str",
"",
"In the second case, the example is:",
"",
"	CONFIGURE_ENV+= --with-datafiles=\${LOCALBASE}/share/battalion",
"",
"This should really be:",
"",
"	CONFIGURE_ENV+= --with-datafiles=\${PREFIX}/share/battalion");
	}

	my $needs_quoting = variable_needs_quoting($line, $varname, $context);

	if ($context->shellword == VUC_SHELLWORD_FOR) {
		if (!defined($type)) {
			# Cannot check anything here.

		} elsif ($type->kind_of_list == LK_INTERNAL) {
			# Fine.

		} elsif ($needs_quoting == doesnt_matter || $needs_quoting == false) {
			# Fine, these variables are assumed to not
			# contain special characters.

		} else {
			$line->log_warning("The variable ${varname} should not be used in .for loops.");
			$line->explain_warning(
"The .for loop splits its argument at sequences of white-space, as",
"opposed to all other places in make(1), which act like the shell.",
"Therefore only variables that are specifically designed to match this",
"requirement should be used here.");
		}
	}

	if ($opt_warn_quoting && $context->shellword != VUC_SHELLWORD_UNKNOWN && $needs_quoting != dont_know) {

		# In GNU configure scripts, a few variables need to be
		# passed through the :M* operator before they reach the
		# configure scripts.
		my $need_mstar = false;
		if ($varname =~ regex_gnu_configure_volatile_vars) {
			# When we are not checking a package, but some other file,
			# the :M* operator is needed for safety.
			if (!defined($pkgctx_vardef) || exists($pkgctx_vardef->{"GNU_CONFIGURE"})) {
				$need_mstar = true;
			}
		}

		my $stripped_mod = ($mod =~ m"(.*?)(?::M\*)?(?::Q)?$") ? $1 : $mod;
		my $correct_mod = $stripped_mod . ($need_mstar ? ":M*:Q" : ":Q");

		if ($mod eq ":M*:Q" && !$need_mstar) {
			$line->log_note("The :M* modifier is not needed here.");

		} elsif ($mod ne $correct_mod && $needs_quoting == true) {
			if ($context->shellword == VUC_SHELLWORD_PLAIN) {
				$line->log_warning("Please use \${${varname}${correct_mod}} instead of \${${varname}${mod}}.");
				#$line->replace("\${${varname}}", "\${${varname}:Q}");
			} else {
				$line->log_warning("Please use \${${varname}${correct_mod}} instead of \${${varname}${mod}} and make sure the variable appears outside of any quoting characters.");
			}
			$line->explain_warning("See the pkgsrc guide, section \"quoting guideline\", for details.");
		}

		if ($mod =~ m":Q$") {
			my @expl = (
"Many variables in pkgsrc do not need the :Q operator, since they",
"are not expected to contain white-space or other evil characters.",
"",
"Another case is when a variable of type ShellWord appears in a context",
"that expects a shell word, it does not need to have a :Q operator. Even",
"when it is concatenated with another variable, it still stays _one_ word.",
"",
"Example:",
"\tWORD1=  Have\\ fun             # 1 word",
"\tWORD2=  \"with BSD Make\"       # 1 word, too",
"",
"\tdemo:",
"\t\techo \${WORD1}\${WORD2} # still 1 word");

			if ($needs_quoting == false) {
				$line->log_warning("The :Q operator should not be used for \${${varname}} here.");
				$line->explain_warning(@expl);
			} elsif ($needs_quoting == doesnt_matter) {
				$line->log_note("The :Q operator isn't necessary for \${${varname}} here.");
				$line->explain_note(@expl);
			}
		}
	}

	assert(defined($mkctx_build_defs), "The build_defs variable must be defined here.");
	if (exists(get_userdefined_variables()->{$varname}) && !exists(get_system_build_defs()->{$varname}) && !exists($mkctx_build_defs->{$varname})) {
		$line->log_warning("The user-defined variable ${varname} is used but not added to BUILD_DEFS.");
		$line->explain_warning(
"When a pkgsrc package is built, many things can be configured by the",
"pkgsrc user in the mk.conf file. All these configurations should be",
"recorded in the binary package, so the package can be reliably rebuilt.",
"The BUILD_DEFS variable contains a list of all these user-settable",
"variables, so please add your variable to it, too.");
	}
}

sub checkline_mk_shellword($$$) {
	my ($line, $shellword, $check_quoting) = @_;
	my ($rest, $state, $value);


CONT_HERE
			my $ctx = PkgLint::VarUseContext->new_from_pool(
				VUC_TIME_UNKNOWN,
				shellcommand_context_type,
				($state == SWST_PLAIN) ? VUC_SHELLWORD_PLAIN
				: ($state == SWST_DQUOT) ? VUC_SHELLWORD_DQUOT
				: ($state == SWST_SQUOT) ? VUC_SHELLWORD_SQUOT
				: ($state == SWST_BACKT) ? VUC_SHELLWORD_BACKT
				: VUC_SHELLWORD_UNKNOWN,
				VUC_EXTENT_WORD_PART
			);
			if ($varname ne "\@") {
				checkline_mk_varuse($line, $varname, defined($mod) ? $mod : "", $ctx);
			}

		# The syntax of the variable modifiers can get quite
		# hairy. In lack of motivation, we just skip anything
		# complicated, hoping that at least the braces are
		# balanced.
		} elsif ($rest =~ s/^\$\{//) {
			my $braces = 1;
			while ($rest ne "" && $braces > 0) {
				if ($rest =~ s/^\}//) {
					$braces--;
				} elsif ($rest =~ s/^\{//) {
					$braces++;
				} elsif ($rest =~ s/^[^{}]+//) {
					# Nothing to do here.
				} else {
					last;
				}
			}

		} elsif ($state == SWST_PLAIN) {

			# XXX: This is only true for the "first" word in a
			# shell command, not for every word. For example,
			# FOO_ENV+= VAR=`command` may be underquoted.
			if (false && $rest =~ m"([\w_]+)=\"\`") {
				$line->log_note("In the assignment to \"$1\", you don't need double quotes around backticks.");
				$line->explain_note(
"Assignments are a special context, where no double quotes are needed",
"around backticks. In other contexts, the double quotes are necessary.");
			}

			if ($rest =~ s/^[!#\%&\(\)*+,\-.\/0-9:;<=>?\@A-Z\[\]^_a-z{|}~]+//) {
			} elsif ($rest =~ s/^\'//) {
				$state = SWST_SQUOT;
			} elsif ($rest =~ s/^\"//) {
				$state = SWST_DQUOT;
			} elsif ($rest =~ s/^\`//) { #`
				$state = SWST_BACKT;
			} elsif ($rest =~ s/^\\(?:[ !"#'\(\)*;?\\^{|}]|\$\$)//) {
			} elsif ($rest =~ s/^\$\$([0-9A-Z_a-z]+|\#)//
				|| $rest =~ s/^\$\$\{([0-9A-Z_a-z]+|\#)\}//
				|| $rest =~ s/^\$\$(\$)\$//) {
				my ($shvarname) = ($1);
				if ($opt_warn_quoting && $check_quoting) {
					$line->log_warning("Unquoted shell variable \"${shvarname}\".");
					$line->explain_warning(
"When a shell variable contains white-space, it is expanded (split into",
"multiple words) when it is written as \$variable in a shell script.",
"If that is not intended, you should add quotation marks around it,",
"like \"\$variable\". Then, the variable will always expand to a single",
"word, preserving all white-space and other special characters.",
"",
"Example:",
"\tfname=\"Curriculum vitae.doc\"",
"\tcp \$fname /tmp",
"\t# tries to copy the two files \"Curriculum\" and \"Vitae.doc\"",
"\tcp \"\$fname\" /tmp",
"\t# copies one file, as intended");
				}

			} elsif ($rest =~ s/^\$\@//) {
				$line->log_warning("Please use \"\${.TARGET}\" instead of \"\$@\".");
				$line->explain_warning(
"It is more readable and prevents confusion with the shell variable of",
"the same name.");

			} elsif ($rest =~ s/^\$\$\@//) {
				$line->log_warning("The \$@ shell variable should only be used in double quotes.");

			} elsif ($rest =~ s/^\$\$\?//) {
				$line->log_warning("The \$? shell variable is often not available in \"set -e\" mode.");

			} elsif ($rest =~ s/^\$\$\(/(/) {
				$line->log_warning("Invoking subshells via \$(...) is not portable enough.");
				$line->explain_warning(
"The Solaris /bin/sh does not know this way to execute a command in a",
"subshell. Please use backticks (\`...\`) as a replacement.");

			} else {
				last;
			}

		} elsif ($state == SWST_SQUOT) {
			if ($rest =~ s/^\'//) {
				$state = SWST_PLAIN;
			} elsif ($rest =~ s/^[^\$\']+//) { #'
			} elsif ($rest =~ s/^\$\$//) {
			} else {
				last;
			}

		} elsif ($state == SWST_DQUOT) {
			if ($rest =~ s/^\"//) {
				$state = SWST_PLAIN;
			} elsif ($rest =~ s/^\`//) { #`
				$state = SWST_DQUOT_BACKT;
			} elsif ($rest =~ s/^[^\$"\\\`]+//) { #`
			} elsif ($rest =~ s/^\\(?:[\\\"\`]|\$\$)//) { #`
			} elsif ($rest =~ s/^\$\$\{([0-9A-Za-z_]+)\}//
				|| $rest =~ s/^\$\$([0-9A-Z_a-z]+|[!#?\@]|\$\$)//) {
				my ($shvarname) = ($1);
				$opt_debug_shell and $line->log_debug("[checkline_mk_shellword] Found double-quoted variable ${shvarname}.");
			} elsif ($rest =~ s/^\$\$//) {
				$line->log_warning("Unquoted \$ or strange shell variable found.");
			} elsif ($rest =~ s/^\\(.)//) {
				my ($char) = ($1);
				$line->log_warning("Please use \"\\\\${char}\" instead of \"\\${char}\".");
				$line->explain_warning(
"Although the current code may work, it is not good style to rely on",
"the shell passing \"\\${char}\" exactly as is, and not discarding the",
"backslash. Alternatively you can use single quotes instead of double",
"quotes.");
			} else {
				last;
			}

		} else {
			last;
		}
	}
	if ($rest !~ m"^\s*$") {
		$line->log_error("Internal pkglint error: " . statename->[$state] . ": rest=${rest}");
	}
}

# Some shell commands should not be used in the install phase.
#
sub checkline_mk_shellcmd_use($$) {
	my ($line, $shellcmd) = @_;

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
	));
	use constant discouraged_install_commands => array_to_hash(qw(
		sed ${SED}
		tr ${TR}
	));

	if (defined($mkctx_target) && $mkctx_target =~ m"^(?:pre|do|post)-install") {

		if (exists(allowed_install_commands->{$shellcmd})) {
			# Fine.

		} elsif (exists(discouraged_install_commands->{$shellcmd})) {
			$line->log_warning("The shell command \"${shellcmd}\" should not be used in the install phase.");
			$line->explain_warning(
"In the install phase, the only thing that should be done is to install",
"the prepared files to their final location. The file's contents should",
"not be changed anymore.");

		} elsif ($shellcmd eq "\${CP}") {
			$line->log_warning("\${CP} should not be used to install files.");
			$line->explain_warning(
"The \${CP} command is highly platform dependent and cannot overwrite",
"files that don't have write permission. Please use \${PAX} instead.",
"",
"For example, instead of",
"\t\${CP} -R \${WRKSRC}/* \${PREFIX}/foodir",
"you should use",
"\tcd \${WRKSRC} && \${PAX} -wr * \${PREFIX}/foodir");

		} else {
			$opt_debug_misc and $line->log_debug("May \"${shellcmd}\" be used in the install phase?");
		}
	}
}

sub checkline_mk_shelltext($$) {
	my ($line, $text) = @_;
	my ($vartools, $state, $rest, $set_e_mode);

	$opt_debug_trace and $line->log_debug("checkline_mk_shelltext(\"$text\")");

	# Note: SCST is the abbreviation for [S]hell [C]ommand [ST]ate.
	use constant scst => qw(
		START CONT
		INSTALL INSTALL_D
		MKDIR
		PAX PAX_S
		SED SED_E
		SET SET_CONT
		COND COND_CONT
		CASE CASE_IN CASE_LABEL CASE_LABEL_CONT CASE_PAREN
		FOR FOR_IN FOR_CONT
		ECHO
		INSTALL_DIR INSTALL_DIR2
	);
	use enum (":SCST_", scst);
	use constant scst_statename => [ map { "SCST_$_" } scst ];

	use constant forbidden_commands => array_to_hash(qw(
		ktrace
		mktexlsr
		strace
		texconfig truss
	));

	if ($text =~ m"\$\{SED\}" && $text =~ m"\$\{MV\}") {
		$line->log_note("Please use the SUBST framework instead of \${SED} and \${MV}.");
		$line->explain_note(
"When converting things, pay attention to \"#\" characters. In shell",
"commands make(1) does not interpret them as comment character, but",
"in other lines it does. Therefore, instead of the shell command",
"",
"\tsed -e 's,#define foo,,'",
"",
"you need to write",
"",
"\tSUBST_SED.foo+=\t's,\\#define foo,,'");
	}

	if ($text =~ m"^\@*-(.*(MKDIR|INSTALL.*-d|INSTALL_.*_DIR).*)") {
		my ($mkdir_cmd) = ($1);

		$line->log_note("You don't need to use \"-\" before ${mkdir_cmd}.");
	}

	$vartools = get_vartool_names();
	$rest = $text;

	use constant hidden_shell_commands => array_to_hash(qw(
		${DELAYED_ERROR_MSG} ${DELAYED_WARNING_MSG}
		${DO_NADA}
		${ECHO} ${ECHO_MSG} ${ECHO_N} ${ERROR_CAT} ${ERROR_MSG}
		${FAIL_MSG}
		${PHASE_MSG} ${PRINTF}
		${SHCOMMENT} ${STEP_MSG}
		${WARNING_CAT} ${WARNING_MSG}
	));

	$set_e_mode = false;

	if ($rest =~ s/^\s*([-@]*)(\$\{_PKG_SILENT\}\$\{_PKG_DEBUG\}|\$\{RUN\}|)//) {
		my ($hidden, $macro) = ($1, $2);

		if ($hidden !~ m"\@") {
			# Nothing is hidden at all.

		} elsif (defined($mkctx_target) && $mkctx_target =~ m"^(?:show-.*|.*-message)$") {
			# In some targets commands may be hidden.

		} elsif ($rest =~ m"^#") {
			# Shell comments may be hidden, as they have no side effects

		} elsif ($rest =~ $regex_shellword) {
			my ($cmd) = ($1);

			if (!exists(hidden_shell_commands->{$cmd})) {
				$line->log_warning("The shell command \"${cmd}\" should not be hidden.");
				$line->explain_warning(
"Hidden shell commands do not appear on the terminal or in the log file",
"when they are executed. When they fail, the error message cannot be",
"assigned to the command, which is very difficult to debug.");
			}
		}

		if ($hidden =~ m"-") {
			$line->log_warning("The use of a leading \"-\" to suppress errors is deprecated.");
			$line->explain_warning(
"If you really want to ignore any errors from this command (including",
"all errors you never thought of), append \"|| \${TRUE}\" to the",
"command.");
		}

		if ($macro eq "\${RUN}") {
			$set_e_mode = true;
		}
	}

	$state = SCST_START;
	while ($rest =~ s/^$regex_shellword//) {
		my ($shellword) = ($1);

		$opt_debug_shell and $line->log_debug(scst_statename->[$state] . ": ${shellword}");

		checkline_mk_shellword($line, $shellword, !(
			$state == SCST_CASE
			|| $state == SCST_FOR_CONT
			|| $state == SCST_SET_CONT
			|| ($state == SCST_START && $shellword =~ regex_sh_varassign)));

		#
		# Actions associated with the current state
		# and the symbol on the "input tape".
		#

		if ($state == SCST_START || $state == SCST_COND) {
			my ($type);

			if ($shellword eq "\${RUN}") {
				# Just skip this one.

			} elsif (exists(forbidden_commands->{basename($shellword)})) {
				$line->log_error("${shellword} must not be used in Makefiles.");
				$line->explain_error(
"This command must appear in INSTALL scripts, not in the package",
"Makefile, so that the package also works if it is installed as a binary",
"package via pkg_add.");

			} elsif (exists(get_tool_names()->{$shellword})) {
				if (!exists($mkctx_tools->{$shellword}) && !exists($mkctx_tools->{"g$shellword"})) {
					$line->log_warning("The \"${shellword}\" tool is used but not added to USE_TOOLS.");
				}

				if (exists(get_required_vartools()->{$shellword})) {
					$line->log_warning("Please use \"\${" . get_vartool_names()->{$shellword} . "}\" instead of \"${shellword}\".");
				}

				checkline_mk_shellcmd_use($line, $shellword);

			} elsif ($shellword =~ m"^\$\{([\w_]+)\}$" && exists(get_varname_to_toolname()->{$1})) {
				my ($vartool) = ($1);
				my $plain_tool = get_varname_to_toolname()->{$vartool};

				if (!exists($mkctx_tools->{$plain_tool})) {
					$line->log_warning("The \"${plain_tool}\" tool is used but not added to USE_TOOLS.");
				}

				# Deactivated to allow package developers to write
				# consistent code that uses ${TOOL} in all places.
				if (false && defined($mkctx_target) && $mkctx_target =~ m"^(?:pre|do|post)-(?:extract|patch|wrapper|configure|build|install|package|clean)$") {
					if (!exists(get_required_vartool_varnames()->{$vartool})) {
						$opt_warn_extra and $line->log_note("You can write \"${plain_tool}\" instead of \"${shellword}\".");
						$opt_warn_extra and $line->explain_note(
"The wrapper framework from pkgsrc takes care that a sufficiently",
"capable implementation of that tool will be selected.",
"",
"Calling the commands by their plain name instead of the macros is",
"only available in the {pre,do,post}-* targets. For all other targets,",
"you should still use the macros.");
					}
				}

				checkline_mk_shellcmd_use($line, $shellword);

			} elsif ($shellword =~ m"^\$\{([\w_]+)\}$" && defined($type = get_variable_type($line, $1)) && $type->basic_type eq "ShellCommand") {
				checkline_mk_shellcmd_use($line, $shellword);

			} elsif ($shellword =~ m"^\$\{(\w+)\}$" && defined($pkgctx_vardef) && exists($pkgctx_vardef->{$1})) {
				# When the package author has explicitly
				# defined a command variable, assume it
				# to be valid.

			} elsif ($shellword =~ m"^(?:\(|\)|:|;|;;|&&|\|\||\{|\}|break|case|cd|continue|do|done|elif|else|esac|eval|exec|exit|export|fi|for|if|read|set|shift|then|umask|unset|while)$") {
				# Shell builtins are fine.

			} elsif ($shellword =~ m"^[\w_]+=.*$") {
				# Variable assignment.

			} elsif ($shellword =~ m"^\./.*$") {
				# All commands from the current directory are fine.

			} elsif ($shellword =~ m"^#") {
				my $semicolon = ($shellword =~ m";");
				my $multiline = ($line->lines =~ m"--");

				if ($semicolon) {
					$line->log_warning("A shell comment should not contain semicolons.");
				}
				if ($multiline) {
					$line->log_warning("A shell comment does not stop at the end of line.");
				}

				if ($semicolon || $multiline) {
					$line->explain_warning(
"When you split a shell command into multiple lines that are continued",
"with a backslash, they will nevertheless be converted to a single line",
"before the shell sees them. That means that even if it _looks_ like that",
"the comment only spans one line in the Makefile, in fact it spans until",
"the end of the whole shell command. To insert a comment into shell code,",
"you can pass it as an argument to the \${SHCOMMENT} macro, which expands",
"to a command doing nothing. Note that any special characters are",
"nevertheless interpreted by the shell.");
				}

			} else {
				$opt_warn_extra and $line->log_warning("Unknown shell command \"${shellword}\".");
				$opt_warn_extra and $line->explain_warning(
"If you want your package to be portable to all platforms that pkgsrc",
"supports, you should only use shell commands that are covered by the",
"tools framework.");

			}
		}

		if ($state == SCST_COND && $shellword eq "cd") {
			$line->log_error("The Solaris /bin/sh cannot handle \"cd\" inside conditionals.");
			$line->explain_error(
"When the Solaris shell is in \"set -e\" mode and \"cd\" fails, the",
"shell will exit, no matter if it is protected by an \"if\" or the",
"\"||\" operator.");
		}

		if (($state != SCST_PAX_S && $state != SCST_SED_E && $state != SCST_CASE_LABEL)) {
			checkline_mk_absolute_pathname($line, $shellword);
		}

		if (($state == SCST_INSTALL_D || $state == SCST_MKDIR) && $shellword =~ m"^(?:\$\{DESTDIR\})?\$\{PREFIX(?:|:Q)\}/") {
			$line->log_warning("Please use AUTO_MKDIRS instead of "
				. (($state == SCST_MKDIR) ? "\${MKDIR}" : "\${INSTALL} -d")
				. ".");
			$line->explain_warning(
"Setting AUTO_MKDIRS=yes automatically creates all directories that are",
"mentioned in the PLIST. If you need additional directories, specify",
"them in INSTALLATION_DIRS, which is a list of directories relative to",
"\${PREFIX}.");
		}

		if (($state == SCST_INSTALL_DIR || $state == SCST_INSTALL_DIR2) && $shellword !~ regex_mk_shellvaruse && $shellword =~ m"^(?:\$\{DESTDIR\})?\$\{PREFIX(?:|:Q)\}/(.*)") {
			my ($dirname) = ($1);

			$line->log_note("You can use AUTO_MKDIRS=yes or INSTALLATION_DIRS+= ${dirname} instead of this command.");
			$line->explain_note(
"This saves you some typing. You also don't have to think about which of",
"the many INSTALL_*_DIR macros is appropriate, since INSTALLATION_DIRS",
"takes care of that.",
"",
"Note that you should only do this if the package creates _all_",
"directories it needs before trying to install files into them.",
"",
"Many packages include a list of all needed directories in their PLIST",
"file. In that case, you can just set AUTO_MKDIRS=yes and be done.");
		}

		if ($state == SCST_INSTALL_DIR2 && $shellword =~ m"^\$") {
			$line->log_warning("The INSTALL_*_DIR commands can only handle one directory at a time.");
			$line->explain_warning(
"Many implementations of install(1) can handle more, but pkgsrc aims at",
"maximum portability.");
		}

		if ($state == SCST_PAX && $shellword eq "-pe") {
			$line->log_warning("Please use the -pp option to pax(1) instead of -pe.");
			$line->explain_warning(
"The -pe option tells pax to preserve the ownership of the files, which",
"means that the installed files will belong to the user that has built",
"the package. That's a Bad Thing.");
		}

		if ($state == SCST_PAX_S || $state == SCST_SED_E) {
			if (false && $shellword !~ m"^[\"\'].*[\"\']$") {
				$line->log_warning("Substitution commands like \"${shellword}\" should always be quoted.");
				$line->explain_warning(
"Usually these substitution commands contain characters like '*' or",
"other shell metacharacters that might lead to lookup of matching",
"filenames and then expand to more than one word.");
			}
		}

		if ($state == SCST_ECHO && $shellword eq "-n") {
			$line->log_warning("Please use \${ECHO_N} instead of \"echo -n\".");
		}

		if ($opt_warn_extra && $state != SCST_CASE_LABEL_CONT && $shellword eq "|") {
			$line->log_warning("The exitcode of the left-hand-side command of the pipe operator is ignored.");
			$line->explain_warning(
"If you need to detect the failure of the left-hand-side command, use",
"temporary files to save the output of the command.");
		}

		if ($opt_warn_extra && $shellword eq ";" && $state != SCST_COND_CONT && $state != SCST_FOR_CONT && !$set_e_mode) {
			$line->log_warning("Please switch to \"set -e\" mode before using a semicolon to separate commands.");
			$line->explain_warning(
"Older versions of the NetBSD make(1) had run the shell commands using",
"the \"-e\" option of /bin/sh. In 2004, this behavior has been changed to",
"follow the POSIX conventions, which is to not use the \"-e\" option.",
"The consequence of this change is that shell programs don't terminate",
"as soon as an error occurs, but try to continue with the next command.",
"Imagine what would happen for these commands:",
"    cd \"\$HOME\"; cd /nonexistent; rm -rf *",
"To fix this warning, either insert \"set -e\" at the beginning of this",
"line or use the \"&&\" operator instead of the semicolon.");
		}

		#
		# State transition.
		#

		if ($state == SCST_SET && $shellword =~ m"^-.*e") {
			$set_e_mode = true;
		}
		if ($state == SCST_START && $shellword eq "\${RUN}") {
			$set_e_mode = true;
		}

		$state =  ($shellword eq ";;") ? SCST_CASE_LABEL
			# Note: The order of the following two lines is important.
			: ($state == SCST_CASE_LABEL_CONT && $shellword eq "|") ? SCST_CASE_LABEL
			: ($shellword =~ m"^[;&\|]+$") ? SCST_START
			: ($state == SCST_START) ? (
				($shellword eq "\${INSTALL}") ? SCST_INSTALL
				: ($shellword eq "\${MKDIR}") ? SCST_MKDIR
				: ($shellword eq "\${PAX}") ? SCST_PAX
				: ($shellword eq "\${SED}") ? SCST_SED
				: ($shellword eq "\${ECHO}") ? SCST_ECHO
				: ($shellword eq "\${RUN}") ? SCST_START
				: ($shellword eq "echo") ? SCST_ECHO
				: ($shellword eq "set") ? SCST_SET
				: ($shellword =~ m"^(?:if|elif|while)$") ? SCST_COND
				: ($shellword =~ m"^(?:then|else|do)$") ? SCST_START
				: ($shellword eq "case") ? SCST_CASE
				: ($shellword eq "for") ? SCST_FOR
				: ($shellword eq "(") ? SCST_START
				: ($shellword =~ m"^\$\{INSTALL_[A-Z]+_DIR\}$") ? SCST_INSTALL_DIR
				: ($shellword =~ regex_sh_varassign) ? SCST_START
				: SCST_CONT)
			: ($state == SCST_MKDIR) ? SCST_MKDIR
			: ($state == SCST_INSTALL && $shellword eq "-d") ? SCST_INSTALL_D
			: ($state == SCST_INSTALL || $state == SCST_INSTALL_D) ? (
				($shellword =~ m"^-[ogm]$") ? SCST_CONT
				: $state)
			: ($state == SCST_INSTALL_DIR) ? (
				($shellword =~ m"^-") ? SCST_CONT
				: ($shellword =~ m"^\$") ? SCST_INSTALL_DIR2
				: $state)
			: ($state == SCST_INSTALL_DIR2) ? $state
			: ($state == SCST_PAX) ? (
				($shellword eq "-s") ? SCST_PAX_S
				: ($shellword =~ m"^-") ? SCST_PAX
				: SCST_CONT)
			: ($state == SCST_PAX_S) ? SCST_PAX
			: ($state == SCST_SED) ? (
				($shellword eq "-e") ? SCST_SED_E
				: ($shellword =~ m"^-") ? SCST_SED
				: SCST_CONT)
			: ($state == SCST_SED_E) ? SCST_SED
			: ($state == SCST_SET) ? SCST_SET_CONT
			: ($state == SCST_SET_CONT) ? SCST_SET_CONT
			: ($state == SCST_CASE) ? SCST_CASE_IN
			: ($state == SCST_CASE_IN && $shellword eq "in") ? SCST_CASE_LABEL
			: ($state == SCST_CASE_LABEL && $shellword eq "esac") ? SCST_CONT
			: ($state == SCST_CASE_LABEL) ? SCST_CASE_LABEL_CONT
			: ($state == SCST_CASE_LABEL_CONT && $shellword eq ")") ? SCST_START
			: ($state == SCST_CONT) ? SCST_CONT
			: ($state == SCST_COND) ? SCST_COND_CONT
			: ($state == SCST_COND_CONT) ? SCST_COND_CONT
			: ($state == SCST_FOR) ? SCST_FOR_IN
			: ($state == SCST_FOR_IN && $shellword eq "in") ? SCST_FOR_CONT
			: ($state == SCST_FOR_CONT) ? SCST_FOR_CONT
			: ($state == SCST_ECHO) ? SCST_CONT
			: do {
				$line->log_warning("[" . scst_statename->[$state] . " ${shellword}] Keeping the current state.");
				$state;
			};
	}

	if ($rest !~ m"^\s*$") {
		$line->log_error("Internal pkglint error: " . scst_statename->[$state] . ": rest=${rest}");
	}
}

sub checkline_mk_shellcmd($$) {
	my ($line, $shellcmd) = @_;

	checkline_mk_text($line, $shellcmd);
	checkline_mk_shelltext($line, $shellcmd);
}


sub expand_permission($) {
	my ($perm) = @_;
	my %fullperm = ( "a" => "append", "d" => "default", "p" => "preprocess", "s" => "set", "u" => "runtime", "?" => "unknown" );
	my $result = join(", ", map { $fullperm{$_} } split //, $perm);
	$result =~ s/, $//g;

	return $result;
}

sub checkline_mk_vardef($$$) {
	my ($line, $varname, $op) = @_;

	$opt_debug_trace and $line->log_debug("checkline_mk_vardef(${varname}, ${op})");

	# If we are checking a whole package, add it to the package-wide
	# list of defined variables.
	if (defined($pkgctx_vardef) && !exists($pkgctx_vardef->{$varname})) {
		$pkgctx_vardef->{$varname} = $line;
	}

	# Add it to the file-wide list of defined variables.
	if (!exists($mkctx_vardef->{$varname})) {
		$mkctx_vardef->{$varname} = $line;
	}

	return unless $opt_warn_perm;

	my $perms = get_variable_perms($line, $varname);
	my $needed = { "=" => "s", "!=" => "s", "?=" => "d", "+=" => "a", ":=" => "s" }->{$op};
	if (index($perms, $needed) == -1) {
		$line->log_warning("Permission [" . expand_permission($needed) . "] requested for ${varname}, but only [" . expand_permission($perms) . "] is allowed.");
		$line->explain_warning(
"The available permissions are:",
"\tappend\t\tappend something using +=",
"\tdefault\t\tset a default value using ?=",
"\tpreprocess\tuse a variable during preprocessing",
"\truntime\t\tuse a variable at runtime",
"\tset\t\tset a variable using :=, =, !=",
"",
"A \"?\" means that it is not yet clear which permissions are allowed",
"and which aren't.");
	}
}

# @param $op
#	The operator that is used for reading or writing to the variable.
#	One of: "=", "+=", ":=", "!=", "?=", "use", "pp-use", "".
#	For some variables (like BuildlinkDepth or BuildlinkPackages), the
#	operator influences the valid values.
# @param $comment
#	In assignments, a part of the line may be commented out. If there
#	is no comment, pass C<undef>.
#
sub checkline_mk_vartype_basic($$$$$$$$);
sub checkline_mk_vartype_basic($$$$$$$$) {
	my ($line, $varname, $type, $op, $value, $comment, $list_context, $is_guessed) = @_;
	my ($value_novar);

	$opt_debug_trace and $line->log_debug(sprintf("checkline_mk_vartype_basic(%s, %s, %s, %s, %s, %s, %s)",
		$varname, $type, $op, $value, defined($comment) ? $comment : "<undef>", $list_context, $is_guessed));

	$value_novar = $value;
	while ($value_novar =~ s/\$\{([^{}]*)\}//g) {
		my ($varuse) = ($1);
		if (!$list_context && $varuse =~ m":Q$") {
			$line->log_warning("The :Q operator should only be used in lists and shell commands.");
		}
	}

	my %type_dispatch = (
		AwkCommand => sub {
			$opt_debug_unchecked and $line->log_debug("Unchecked AWK command: ${value}");
		},

		BrokenIn => sub {
			if ($value ne $value_novar) {
				$line->log_error("${varname} must not refer to other variables.");

			} elsif ($value =~ m"^pkgsrc-(\d\d\d\d)Q(\d)$") {
				my ($year, $quarter) = ($1, $2);

				# Fine.

			} else {
				$line->log_warning("Invalid value \"${value}\" for ${varname}.");
			}
			$line->log_note("Please remove this line if the package builds for you.");
		},

		BuildlinkDepmethod => sub {
			# Note: this cannot be replaced with { build full } because
			# enumerations may not contain references to other variables.
			if ($value ne $value_novar) {
				# No checks yet.
			} elsif ($value ne "build" && $value ne "full") {
				$line->log_warning("Invalid dependency method \"${value}\". Valid methods are \"build\" or \"full\".");
			}
		},

		BuildlinkDepth => sub {
			if (!($op eq "use" && $value eq "+")
				&& $value ne "\${BUILDLINK_DEPTH}+"
				&& $value ne "\${BUILDLINK_DEPTH:S/+\$//}") {
				$line->log_warning("Invalid value for ${varname}.");
			}
		},

		BuildlinkPackages => sub {
			my $re_del = qr"\$\{BUILDLINK_PACKAGES:N(?:[+\-.0-9A-Z_a-z]|\$\{[^\}]+\})+\}";
			my $re_add = qr"(?:[+\-.0-9A-Z_a-z]|\$\{[^\}]+\})+";

			if (($op eq ":=" && $value =~ m"^${re_del}$") ||
				($op eq ":=" && $value =~ m"^${re_del}\s+${re_add}$") ||
				($op eq "+=" && $value =~ m"^${re_add}$")) {
				# Fine.

			} else {
				$line->log_warning("Invalid value for ${varname}.");
			}
		},

		Category => sub {
			my $allowed_categories = join("|", qw(
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
			));
			if ($value !~ m"^(?:${allowed_categories})$") {
				$line->log_error("Invalid category \"${value}\".");
			}
		},

		CFlag => sub {
			if ($value =~ m"^-D([0-9A-Z_a-z]+)=(.*)") {
				my ($macname, $macval) = ($1, $2);

				# No checks needed, since the macro definitions
				# are usually directory names, which don't need
				# any quoting.

			} elsif ($value =~ m"^-[DU]([0-9A-Z_a-z]+)") {
				my ($macname) = ($1);

				$opt_debug_unchecked and $line->log_debug("Unchecked macro ${macname} in ${varname}.");

			} elsif ($value =~ m"^-I(.*)") {
				my ($dirname) = ($1);

				$opt_debug_unchecked and $line->log_debug("Unchecked directory ${dirname} in ${varname}.");

			} elsif ($value eq "-c99") {
				# Only works on IRIX, but is usually enclosed with
				# the proper preprocessor conditional.

			} elsif ($value =~ m"^-[OWfgm]|^-std=.*") {
				$opt_debug_unchecked and $line->log_debug("Unchecked compiler flag ${value} in ${varname}.");

			} elsif ($value =~ m"^-.*") {
				$line->log_warning("Unknown compiler flag \"${value}\".");

			} elsif ($value =~ regex_unresolved) {
				$opt_debug_unchecked and $line->log_debug("Unchecked CFLAG: ${value}");

			} else {
				$line->log_warning("Compiler flag \"${value}\" does not start with a dash.");
			}
		},

		Comment => sub {
			if ($value eq "SHORT_DESCRIPTION_OF_THE_PACKAGE") {
				$line->log_error("COMMENT must be set.");
			}
			if ($value =~ m"^(a|an)\s+"i) {
				$line->log_warning("COMMENT should not begin with '$1'.");
			}
			if ($value =~ m"^[a-z]") {
				$line->log_warning("COMMENT should start with a capital letter.");
			}
			if ($value =~ m"\.$") {
				$line->log_warning("COMMENT should not end with a period.");
			}
			if (length($value) > 70) {
				$line->log_warning("COMMENT should not be longer than 70 characters.");
			}
		},

		Dependency => sub {
			if ($value =~ m"^(${regex_pkgbase})(<|=|>|<=|>=|!=|-)(${regex_pkgversion})$") {
				my ($depbase, $depop, $depversion) = ($1, $2, $3);

			} elsif ($value =~ m"^(${regex_pkgbase})-(?:\[(.*)\]\*|(\d+(?:\.\d+)*(?:\.\*)?)(\{,nb\*\}|\*|)|(.*))?$") {
				my ($depbase, $bracket, $version, $version_wildcard, $other) = ($1, $2, $3, $4, $5);

				if (defined($bracket)) {
					if ($bracket ne "0-9") {
						$line->log_warning("Only [0-9]* is allowed in the numeric part of a dependency.");
					}

				} elsif (defined($version) && defined($version_wildcard) && $version_wildcard ne "") {
					# Great.

				} elsif (defined($version)) {
					$line->log_warning("Please append {,nb*} to the version number of this dependency.");
					$line->explain_warning(
"Usually, a dependency should stay valid when the PKGREVISION is",
"increased, since those changes are most often editorial. In the",
"current form, the dependency only matches if the PKGREVISION is",
"undefined.");

				} elsif ($other eq "*") {
					$line->log_warning("Please use ${depbase}-[0-9]* instead of ${depbase}-*.");
					$line->explain_warning(
"If you use a * alone, the package specification may match other",
"packages that have the same prefix, but a longer name. For example,",
"foo-* matches foo-1.2, but also foo-client-1.2 and foo-server-1.2.");

				} else {
					$line->log_error("Unknown dependency pattern \"${value}\".");
				}

			} elsif ($value =~ m"\{") {
				# Dependency patterns containing alternatives
				# are just too hard to check.
				$opt_debug_unchecked and $line->log_debug("Unchecked dependency pattern: ${value}");

			} elsif ($value ne $value_novar) {
				$opt_debug_unchecked and $line->log_debug("Unchecked dependency: ${value}");

			} else {
				$line->log_warning("Unknown dependency format: ${value}");
				$line->explain_warning(
"Typical dependencies have the form \"package>=2.5\", \"package-[0-9]*\"",
"or \"package-3.141\".");
			}
		},

		DependencyWithPath => sub {
			if ($value =~ regex_unresolved) {
				# don't even try to check anything
			} elsif ($value =~ m"(.*):(\.\./\.\./([^/]+)/([^/]+))$") {
				my ($pattern, $relpath, $cat, $pkg) = ($1, $2, $3, $4);

				checkline_relative_pkgdir($line, $relpath);

				if ($pkg eq "msgfmt" || $pkg eq "gettext") {
					$line->log_warning("Please use USE_TOOLS+=msgfmt instead of this dependency.");

				} elsif ($pkg =~ m"^perl\d+") {
					$line->log_warning("Please use USE_TOOLS+=perl:run instead of this dependency.");

				} elsif ($pkg eq "gmake") {
					$line->log_warning("Please use USE_TOOLS+=gmake instead of this dependency.");

				}

				if ($pattern =~ regex_dependency_lge) {
#				($abi_pkg, $abi_version) = ($1, $2);
				} elsif ($pattern =~ regex_dependency_wildcard) {
#				($abi_pkg) = ($1);
				} else {
					$line->log_error("Unknown dependency pattern \"${pattern}\".");
				}

			} elsif ($value =~ m":\.\./[^/]+$") {
				$line->log_warning("Dependencies should have the form \"../../category/package\".");
				$line->explain_warning(expl_relative_dirs);

			} else {
				$line->log_warning("Unknown dependency format.");
				$line->explain_warning(
"Examples for valid dependencies are:",
"  package-[0-9]*:../../category/package",
"  package>=3.41:../../category/package",
"  package-2.718:../../category/package");
			}
		},

		DistSuffix => sub {
			if ($value eq ".tar.gz") {
				$line->log_note("${varname} is \".tar.gz\" by default, so this definition may be redundant.");
			}
		},

		EmulPlatform => sub {
			if ($value =~ m"^(\w+)-(\w+)$") {
				my ($opsys, $arch) = ($1, $2);

				if ($opsys !~ m"^(?:bsdos|cygwin|darwin|dragonfly|freebsd|haiku|hpux|interix|irix|linux|netbsd|openbsd|osf1|sunos|solaris)$") {
					$line->log_warning("Unknown operating system: ${opsys}");
				}
				# no check for $os_version
				if ($arch !~ m"^(?:i386|alpha|amd64|arc|arm|arm32|cobalt|convex|dreamcast|hpcmips|hpcsh|hppa|ia64|m68k|m88k|mips|mips64|mipsel|mipseb|mipsn32|ns32k|pc532|pmax|powerpc|rs6000|s390|sparc|sparc64|vax|x86_64)$") {
					$line->log_warning("Unknown hardware architecture: ${arch}");
				}

			} else {
				$line->log_warning("\"${value}\" is not a valid emulation platform.");
				$line->explain_warning(
"An emulation platform has the form <OPSYS>-<MACHINE_ARCH>.",
"OPSYS is the lower-case name of the operating system, and MACHINE_ARCH",
"is the hardware architecture.",
"",
"Examples: linux-i386, irix-mipsel.");
			}
		},

		FetchURL => sub {
			checkline_mk_vartype_basic($line, $varname, "URL", $op, $value, $comment, $list_context, $is_guessed);

			my $sites = get_dist_sites();
			foreach my $site (keys(%{$sites})) {
				if (index($value, $site) == 0) {
					my $subdir = substr($value, length($site));
					my $is_github = $value =~ m"^https://github\.com/";
					if ($is_github) {
						$subdir =~ s|/.*|/|;
					}
					$line->log_warning(sprintf("Please use \${%s:=%s} instead of \"%s\".", $sites->{$site}, $subdir, $value));
					if ($is_github) {
						$line->log_warning("Run \"".conf_make." help topic=github\" for further tips.");
					}
					last;
				}
			}
		},

		Filename => sub {
			if ($value_novar =~ m"/") {
				$line->log_warning("A filename should not contain a slash.");

			} elsif ($value_novar !~ m"^[-0-9\@A-Za-z.,_~+%]*$") {
				$line->log_warning("\"${value}\" is not a valid filename.");
			}
		},

		Filemask => sub {
			if ($value_novar !~ m"^[-0-9A-Za-z._~+%*?]*$") {
				$line->log_warning("\"${value}\" is not a valid filename mask.");
			}
		},

		FileMode => sub {
			if ($value ne "" && $value_novar eq "") {
				# Fine.
			} elsif ($value =~ m"^[0-7]{3,4}") {
				# Fine.
			} else {
				$line->log_warning("Invalid file mode ${value}.");
			}
		},

		Identifier => sub {
			if ($value ne $value_novar) {
				#$line->log_warning("Identifiers should be given directly.");
			}
			if ($value_novar =~ m"^[+\-.0-9A-Z_a-z]+$") {
				# Fine.
			} elsif ($value ne "" && $value_novar eq "") {
				# Don't warn here.
			} else {
				$line->log_warning("Invalid identifier \"${value}\".");
			}
		},

		Integer => sub {
			if ($value !~ m"^\d+$") {
				$line->log_warning("${varname} must be a valid integer.");
			}
		},

		LdFlag => sub {
			if ($value =~ m"^-L(.*)") {
				my ($dirname) = ($1);

				$opt_debug_unchecked and $line->log_debug("Unchecked directory ${dirname} in ${varname}.");

			} elsif ($value =~ m"^-l(.*)") {
				my ($libname) = ($1);

				$opt_debug_unchecked and $line->log_debug("Unchecked library name ${libname} in ${varname}.");

			} elsif ($value =~ m"^(?:-static)$") {
				# Assume that the wrapper framework catches these.

			} elsif ($value =~ m"^(-Wl,(?:-R|-rpath|--rpath))") {
				my ($rpath_flag) = ($1);
				$line->log_warning("Please use \${COMPILER_RPATH_FLAG} instead of ${rpath_flag}.");

			} elsif ($value =~ m"^-.*") {
				$line->log_warning("Unknown linker flag \"${value}\".");

			} elsif ($value =~ regex_unresolved) {
				$opt_debug_unchecked and $line->log_debug("Unchecked LDFLAG: ${value}");

			} else {
				$line->log_warning("Linker flag \"${value}\" does not start with a dash.");
			}
		},

		License => CheckvartypeLicense,

		Mail_Address => CheckvartypeMailAddress,

		Message => CheckvartypeMessage,

		Option => CheckvartypeOption,

		Pathlist => sub {

			if ($value !~ m":" && $is_guessed) {
				checkline_mk_vartype_basic($line, $varname, "Pathname", $op, $value, $comment, $list_context, $is_guessed);

			} else {

				# XXX: The splitting will fail if $value contains any
				# variables with modifiers, for example :Q or :S/././.
				foreach my $p (split(qr":", $value)) {
					my $p_novar = remove_variables($p);

					if ($p_novar !~ m"^[-0-9A-Za-z._~+%/]*$") {
						$line->log_warning("\"${p}\" is not a valid pathname.");
					}

					if ($p !~ m"^[\$/]") {
						$line->log_warning("All components of ${varname} (in this case \"${p}\") should be an absolute path.");
					}
				}
			}
		},

		Pathmask => sub {
			if ($value_novar !~ m"^[#\-0-9A-Za-z._~+%*?/\[\]]*$") {
				$line->log_warning("\"${value}\" is not a valid pathname mask.");
			}
			checkline_mk_absolute_pathname($line, $value);
		},

		Pathname => sub {
			if ($value_novar !~ m"^[#\-0-9A-Za-z._~+%/]*$") {
				$line->log_warning("\"${value}\" is not a valid pathname.");
			}
			checkline_mk_absolute_pathname($line, $value);
		},

		Perl5Packlist => sub {
			if ($value ne $value_novar) {
				$line->log_warning("${varname} should not depend on other variables.");
			}
		},

		PkgName => sub {
			if ($value eq $value_novar && $value !~ regex_pkgname) {
				$line->log_warning("\"${value}\" is not a valid package name. A valid package name has the form packagename-version, where version consists only of digits, letters and dots.");
			}
		},

		PkgPath => sub {
			checkline_relative_pkgdir($line, "$cur_pkgsrcdir/$value");
		},

		PkgOptionsVar => sub {
			checkline_mk_vartype_basic($line, $varname, "Varname", $op, $value, $comment, false, $is_guessed);
			if ($value =~ m"\$\{PKGBASE[:\}]") {
				$line->log_error("PKGBASE must not be used in PKG_OPTIONS_VAR.");
				$line->explain_error(
"PKGBASE is defined in bsd.pkg.mk, which is included as the",
"very last file, but PKG_OPTIONS_VAR is evaluated earlier.",
"Use \${PKGNAME:C/-[0-9].*//} instead.");
			}
		},

		PkgRevision => sub {
			if ($value !~ m"^[1-9]\d*$") {
				$line->log_warning("${varname} must be a positive integer number.");
			}
			if ($line->fname !~ m"(?:^|/)Makefile$") {
				$line->log_error("${varname} only makes sense directly in the package Makefile.");
				$line->explain_error(
"Usually, different packages using the same Makefile.common have",
"different dependencies and will be bumped at different times (e.g. for",
"shlib major bumps) and thus the PKGREVISIONs must be in the separate",
"Makefiles. There is no practical way of having this information in a",
"commonly used Makefile.");
			}
		},

		PlatformTriple => sub {
			my $part = qr"(?:\[[^\]]+\]|[^-\[])+";
			if ($value =~ m"^(${part})-(${part})-(${part})$") {
				my ($opsys, $os_version, $arch) = ($1, $2, $3);

				if ($opsys !~ m"^(?:\*|BSDOS|Cygwin|Darwin|DragonFly|FreeBSD|Haiku|HPUX|Interix|IRIX|Linux|NetBSD|OpenBSD|OSF1|QNX|SunOS)$") {
					$line->log_warning("Unknown operating system: ${opsys}");
				}
				# no check for $os_version
				if ($arch !~ m"^(?:\*|i386|alpha|amd64|arc|arm|arm32|cobalt|convex|dreamcast|hpcmips|hpcsh|hppa|ia64|m68k|m88k|mips|mips64|mipsel|mipseb|mipsn32|ns32k|pc532|pmax|powerpc|rs6000|s390|sparc|sparc64|vax|x86_64)$") {
					$line->log_warning("Unknown hardware architecture: ${arch}");
				}

			} else {
				$line->log_warning("\"${value}\" is not a valid platform triple.");
				$line->explain_warning(
"A platform triple has the form <OPSYS>-<OS_VERSION>-<MACHINE_ARCH>.",
"Each of these components may be a shell globbing expression.",
"Examples: NetBSD-*-i386, *-*-*, Linux-*-*.");
			}
		},

		PrefixPathname => sub {
			if ($value =~ m"^man/(.*)") {
				my ($mansubdir) = ($1);

				$line->log_warning("Please use \"\${PKGMANDIR}/${mansubdir}\" instead of \"${value}\".");
			}
		},

		PythonDependency => sub {
			if ($value ne $value_novar) {
				$line->log_warning("Python dependencies should not contain variables.");
			}
			if ($value_novar !~ m"^[+\-.0-9A-Z_a-z]+(?:|:link|:build)$") {
				$line->log_warning("Invalid Python dependency \"${value}\".");
				$line->explain_warning(
"Python dependencies must be an identifier for a package, as specified",
"in lang/python/versioned_dependencies.mk. This identifier may be",
"followed by :build for a build-time only dependency, or by :link for",
"a run-time only dependency.");
			}
		},

		RelativePkgDir => sub {
			checkline_relative_pkgdir($line, $value);
		},

		RelativePkgPath => sub {
			checkline_relative_path($line, $value, true);
		},

		Restricted => sub {
			if ($value ne "\${RESTRICTED}") {
				$line->log_warning("The only valid value for ${varname} is \${RESTRICTED}.");
				$line->explain_warning(
"These variables are used to control which files may be mirrored on FTP",
"servers or CD-ROM collections. They are not intended to mark packages",
"whose only MASTER_SITES are on ftp.NetBSD.org.");
			}
		},

		SedCommand => sub {
		},

		SedCommands => sub {
			my $words = shell_split($value);
			if (!$words) {
				$line->log_error("Invalid shell words in sed commands.");
				$line->explain_error(
"If your sed commands have embedded \"#\" characters, you need to escape",
"them with a backslash, otherwise make(1) will interpret them as a",
"comment, no matter if they occur in single or double quotes or",
"whatever.");

			} else {
				my $nwords = scalar(@{$words});
				my $ncommands = 0;

				for (my $i = 0; $i < $nwords; $i++) {
					my $word = $words->[$i];
					checkline_mk_shellword($line, $word, true);

					if ($word eq "-e") {
						if ($i + 1 < $nwords) {
							# Check the real sed command here.
							$i++;
							$ncommands++;
							if ($ncommands > 1) {
								$line->log_warning("Each sed command should appear in an assignment of its own.");
								$line->explain_warning(
"For example, instead of",
"    SUBST_SED.foo+=        -e s,command1,, -e s,command2,,",
"use",
"    SUBST_SED.foo+=        -e s,command1,,",
"    SUBST_SED.foo+=        -e s,command2,,",
"",
"This way, short sed commands cannot be hidden at the end of a line.");
							}
							checkline_mk_shellword($line, $words->[$i - 1], true);
							checkline_mk_shellword($line, $words->[$i], true);
							checkline_mk_vartype_basic($line, $varname, "SedCommand", $op, $words->[$i], $comment, $list_context, $is_guessed);
						} else {
							$line->log_error("The -e option to sed requires an argument.");
						}
					} elsif ($word eq "-E") {
						# Switch to extended regular expressions mode.

					} elsif ($word eq "-n") {
						# Don't print lines per default.

					} elsif ($i == 0 && $word =~ m"^([\"']?)(?:\d*|/.*/)s(.).*\2g?\1$") {
						$line->log_warning("Please always use \"-e\" in sed commands, even if there is only one substitution.");

					} else {
						$line->log_warning("Unknown sed command ${word}.");
					}
				}
			}
		},

		ShellCommand => sub {
			checkline_mk_shelltext($line, $value);
		},

		ShellWord => sub {
			if (!$list_context) {
				checkline_mk_shellword($line, $value, true);
			}
		},

		Stage => sub {
			if ($value !~ m"^(?:pre|do|post)-(?:extract|patch|configure|build|install)$") {
				$line->log_warning("Invalid stage name. Use one of {pre,do,post}-{extract,patch,configure,build,install}.");
			}
		},

		String => sub {
			# No further checks possible.
		},

		Tool => sub {
			if ($varname eq "TOOLS_NOOP" && $op eq "+=") {
				# no warning for package-defined tool definitions

			} elsif ($value =~ m"^([-\w]+|\[)(?::(\w+))?$") {
				my ($toolname, $tooldep) = ($1, $2);
				if (!exists(get_tool_names()->{$toolname})) {
					$line->log_error("Unknown tool \"${toolname}\".");
				}
				if (defined($tooldep) && $tooldep !~ m"^(?:bootstrap|build|pkgsrc|run)$") {
					$line->log_error("Unknown tool dependency \"${tooldep}\". Use one of \"build\", \"pkgsrc\" or \"run\".");
				}
			} else {
				$line->log_error("Invalid tool syntax: \"${value}\".");
			}
		},

		Unchecked => sub {
			# Do nothing, as the name says.
		},

		URL => sub {
			if ($value eq "" && defined($comment) && $comment =~ m"^#") {
				# Ok

			} elsif ($value =~ m"\$\{(MASTER_SITE_[^:]*).*:=(.*)\}$") {
				my ($name, $subdir) = ($1, $2);

				if (!exists(get_dist_sites_names()->{$name})) {
					$line->log_error("${name} does not exist.");
				}
				if ($subdir !~ m"/$") {
					$line->log_error("The subdirectory in ${name} must end with a slash.");
				}

			} elsif ($value =~ regex_unresolved) {
				# No further checks

			} elsif ($value =~ m"^(https?|ftp|gopher)://([-0-9A-Za-z.]+)(?::(\d+))?/([-%&+,./0-9:=?\@A-Z_a-z~]|#)*$") {
				my ($proto, $host, $port, $path) = ($1, $2, $3, $4);

				if ($host =~ m"\.NetBSD\.org$"i && $host !~ m"\.NetBSD\.org$") {
					$line->log_warning("Please write NetBSD.org instead of ${host}.");
				}

			} elsif ($value =~ m"^([0-9A-Za-z]+)://([^/]+)(.*)$") {
				my ($scheme, $host, $abs_path) = ($1, $2, $3);

				if ($scheme ne "ftp" && $scheme ne "http" && $scheme ne "https" && $scheme ne "gopher") {
					$line->log_warning("\"${value}\" is not a valid URL. Only ftp, gopher, http, and https URLs are allowed here.");

				} elsif ($abs_path eq "") {
					$line->log_note("For consistency, please add a trailing slash to \"${value}\".");

				} else {
					$line->log_warning("\"${value}\" is not a valid URL.");
				}

			} else {
				$line->log_warning("\"${value}\" is not a valid URL.");
			}
		},

		UserGroupName => sub {
			if ($value ne $value_novar) {
				# No checks for now.
			} elsif ($value !~ m"^[0-9_a-z]+$") {
				$line->log_warning("Invalid user or group name \"${value}\".");
			}
		},

		Varname => sub {
			if ($value ne "" && $value_novar eq "") {
				# The value of another variable

			} elsif ($value_novar !~ m"^[A-Z_][0-9A-Z_]*(?:[.].*)?$") {
				$line->log_warning("\"${value}\" is not a valid variable name.");
			}
		},

		Version => sub {
			if ($value !~ m"^([\d.])+$") {
				$line->log_warning("Invalid version number \"${value}\".");
			}
		},

		WrapperReorder => sub {
			if ($value =~ m"^reorder:l:([\w\-]+):([\w\-]+)$") {
				my ($lib1, $lib2) = ($1, $2);
				# Fine.
			} else {
				$line->log_warning("Unknown wrapper reorder command \"${value}\".");
			}
		},

		WrapperTransform => sub {
			if ($value =~ m"^rm:(?:-[DILOUWflm].*|-std=.*)$") {
				# Fine.

			} elsif ($value =~ m"^l:([^:]+):(.+)$") {
				my ($lib, $replacement_libs) = ($1, $2);
				# Fine.

			} elsif ($value =~ m"^'?(?:opt|rename|rm-optarg|rmdir):.*$") {
				# FIXME: This is cheated.
				# Fine.

			} elsif ($value eq "-e" || $value =~ m"^\"?'?s[|:,]") {
				# FIXME: This is cheated.
				# Fine.

			} else {
				$line->log_warning("Unknown wrapper transform command \"${value}\".");
			}
		},

		WrkdirSubdirectory => sub {
			checkline_mk_vartype_basic($line, $varname, "Pathname", $op, $value, $comment, $list_context, $is_guessed);
			if ($value eq "\${WRKDIR}") {
				# Fine.
			} else {
				$opt_debug_unchecked and $line->log_debug("Unchecked subdirectory \"${value}\" of \${WRKDIR}.");
			}
		},

		WrksrcSubdirectory => sub {
			if ($value =~ m"^(\$\{WRKSRC\})(?:/(.*))?") {
				my ($prefix, $rest) = ($1, $2);
				$line->log_note("You can use \"" . (defined($rest) ? $rest : ".") . "\" instead of \"${value}\".");

			} elsif ($value ne "" && $value_novar eq "") {
				# The value of another variable

			} elsif ($value_novar !~ m"^(?:\.|[0-9A-Za-z_\@][-0-9A-Za-z_\@./+]*)$") {
				$line->log_warning("\"${value}\" is not a valid subdirectory of \${WRKSRC}.");
			}
		},

		Yes => sub {
			if ($value !~ m"^(?:YES|yes)(?:\s+#.*)?$") {
				$line->log_warning("${varname} should be set to YES or yes.");
				$line->explain_warning(
"This variable means \"yes\" if it is defined, and \"no\" if it is",
"undefined. Even when it has the value \"no\", this means \"yes\".",
"Therefore when it is defined, its value should correspond to its",
"meaning.");
			}
		},

		YesNo => sub {
			if ($value !~ m"^(?:YES|yes|NO|no)(?:\s+#.*)?$") {
				$line->log_warning("${varname} should be set to YES, yes, NO, or no.");
			}
		},

		YesNo_Indirectly => sub {
			if ($value_novar ne "" && $value !~ m"^(?:YES|yes|NO|no)(?:\s+#.*)?$") {
				$line->log_warning("${varname} should be set to YES, yes, NO, or no.");
			}
		},
	);

	if (ref($type) eq "HASH") {
		if (!exists($type->{$value})) {
			$line->log_warning("\"${value}\" is not valid for ${varname}. Use one of { ".join(" ", sort(keys(%{$type})))." } instead.");
		}

	} elsif (defined $type_dispatch{$type}) {
		$type_dispatch{$type}->();

	} else {
		$line->log_fatal("Type ${type} unknown.");
	}
}

# Checks whether the list of version numbers that are given as the
# C<value> of the variable C<varname> are in decreasing order.
sub checkline_decreasing_order($$$) {
	my ($line, $varname, $value) = @_;

	my @pyver = split(qr"\s+", $value);
	if (!@pyver) {
		$line->log_error("There must be at least one value for ${varname}.");
		return;
	}

	my $ver = shift(@pyver);
	if ($ver !~ m"^\d+$") {
		$line->log_error("All values for ${varname} must be numeric.");
		return;
	}

	while (@pyver) {
		my $nextver = shift(@pyver);
		if ($nextver !~ m"^\d+$") {
			$line->log_error("All values for ${varname} must be numeric.");
			return;
		}

		if ($nextver >= $ver) {
			$line->log_warning("The values for ${varname} should be in decreasing order.");
			$line->explain_warning(
"If they aren't, it may be possible that needless versions of packages",
"are installed.");
		}
		$ver = $nextver;
	}
}

sub checkline_mk_vartype($$$$$) {
	my ($line, $varname, $op, $value, $comment) = @_;

	return unless $opt_warn_types;

	my $vartypes = get_vartypes_map();
	my $varbase = varname_base($varname);
	my $varcanon = varname_canon($varname);

	my $type = get_variable_type($line, $varname);

	if ($op eq "+=") {
		if (defined($type)) {
			if (!$type->may_use_plus_eq()) {
				$line->log_warning("The \"+=\" operator should only be used with lists.");
			}
		} elsif ($varbase !~ m"^_" && $varbase !~ get_regex_plurals()) {
			$line->log_warning("As ${varname} is modified using \"+=\", its name should indicate plural.");
		}
	}

	if (!defined($type)) {
		# Cannot check anything if the type is not known.
		$opt_debug_unchecked and $line->log_debug("Unchecked variable assignment for ${varname}.");

	} elsif ($op eq "!=") {
		$opt_debug_misc and $line->log_debug("Use of !=: ${value}");

	} elsif ($type->kind_of_list != LK_NONE) {
		my (@words, $rest);

		if ($type->kind_of_list == LK_INTERNAL) {
			@words = split(qr"\s+", $value);
			$rest = "";
		} else {
			@words = ();
			$rest = $value;
			while ($rest =~ s/^$regex_shellword//) {
				my ($word) = ($1);
				last if ($word =~ m"^#");
				push(@words, $1);
			}
		}

		foreach my $word (@words) {
			checkline_mk_vartype_basic($line, $varname, $type->basic_type, $op, $word, $comment, true, $type->is_guessed);
			if ($type->kind_of_list != LK_INTERNAL) {
				checkline_mk_shellword($line, $word, true);
			}
		}

		if ($rest !~ m"^\s*$") {
			$line->log_error("Internal pkglint error: rest=${rest}");
		}

	} else {
		checkline_mk_vartype_basic($line, $varname, $type->basic_type, $op, $value, $comment, $type->is_practically_a_list(), $type->is_guessed);
	}
}

sub checkline_mk_varassign($$$$$) {
	my ($line, $varname, $op, $value, $comment) = @_;
	my ($used_vars);
	my $varbase = varname_base($varname);
	my $varcanon = varname_canon($varname);

	$opt_debug_trace and $line->log_debug("checkline_mk_varassign($varname, $op, $value)");

	checkline_mk_vardef($line, $varname, $op);

	if ($op eq "?=" && defined($seen_bsd_prefs_mk) && !$seen_bsd_prefs_mk) {
		if ($varbase eq "BUILDLINK_PKGSRCDIR"
			|| $varbase eq "BUILDLINK_DEPMETHOD"
			|| $varbase eq "BUILDLINK_ABI_DEPENDS") {
			# FIXME: What about these ones? They occur quite often.
		} else {
			$opt_warn_extra and $line->log_warning("Please include \"../../mk/bsd.prefs.mk\" before using \"?=\".");
			$opt_warn_extra and $line->explain_warning(
"The ?= operator is used to provide a default value to a variable. In",
"pkgsrc, many variables can be set by the pkgsrc user in the mk.conf",
"file. This file must be included explicitly. If a ?= operator appears",
"before mk.conf has been included, it will not care about the user's",
"preferences, which can result in unexpected behavior. The easiest way",
"to include the mk.conf file is by including the bsd.prefs.mk file,",
"which will take care of everything.");
		}
	}

	checkline_mk_text($line, $value);
	checkline_mk_vartype($line, $varname, $op, $value, $comment);

	# If the variable is not used and is untyped, it may be a
	# spelling mistake.
	if ($op eq ":=" && $varname eq lc($varname)) {
		$opt_debug_unchecked and $line->log_debug("${varname} might be unused unless it is an argument to a procedure file.");
		# TODO: check $varname against the list of "procedure files".

	} elsif (!var_is_used($varname)) {
		my $vartypes = get_vartypes_map();
		my $deprecated = get_deprecated_map();

		if (exists($vartypes->{$varname}) || exists($vartypes->{$varcanon})) {
			# Ok
		} elsif (exists($deprecated->{$varname}) || exists($deprecated->{$varcanon})) {
			# Ok
		} else {
			$line->log_warning("${varname} is defined but not used. Spelling mistake?");
		}
	}

	if ($value =~ m"/etc/rc\.d") {
		$line->log_warning("Please use the RCD_SCRIPTS mechanism to install rc.d scripts automatically to \${RCD_SCRIPTS_EXAMPLEDIR}.");
	}

	if (!$is_internal && $varname =~ m"^_") {
		$line->log_warning("Variable names starting with an underscore are reserved for internal pkgsrc use.");
	}

	if ($varname eq "PERL5_PACKLIST" && defined($effective_pkgbase) && $effective_pkgbase =~ m"^p5-(.*)") {
		my ($guess) = ($1);
		$guess =~ s/-/\//g;
		$guess = "auto/${guess}/.packlist";

		my ($ucvalue, $ucguess) = (uc($value), uc($guess));
		if ($ucvalue ne $ucguess && $ucvalue ne "\${PERL5_SITEARCH\}/${ucguess}") {
			$line->log_warning("Unusual value for PERL5_PACKLIST -- \"${guess}\" expected.");
		}
	}

	if ($varname eq "CONFIGURE_ARGS" && $value =~ m"=\$\{PREFIX\}/share/kde") {
		$line->log_note("Please .include \"../../meta-pkgs/kde3/kde3.mk\" instead of this line.");
		$line->explain_note(
"That file probably does many things automatically and consistently that",
"this package also does. When using kde3.mk, you can probably also leave",
"out some explicit dependencies.");
	}

	if ($varname eq "EVAL_PREFIX" && $value =~ m"^([\w_]+)=") {
		my ($eval_varname) = ($1);

		# The variables mentioned in EVAL_PREFIX will later be
		# defined by find-prefix.mk. Therefore, they are marked
		# as known in the current file.
		$mkctx_vardef->{$eval_varname} = $line;
	}

	if ($varname eq "PYTHON_VERSIONS_ACCEPTED") {
		checkline_decreasing_order($line, $varname, $value);
	}

	if (defined($comment) && $comment eq "# defined" && $varname !~ m".*(?:_MK|_COMMON)$") {
		$line->log_note("Please use \"# empty\", \"# none\" or \"yes\" instead of \"# defined\".");
		$line->explain_note(
"The value #defined says something about the state of the variable, but",
"not what that _means_. In some cases a variable that is defined means",
"\"yes\", in other cases it is an empty list (which is also only the",
"state of the variable), whose meaning could be described with \"none\".",
"It is this meaning that should be described.");
	}

	if ($value =~ m"\$\{(PKGNAME|PKGVERSION)[:\}]") {
		my ($pkgvarname) = ($1);
		if ($varname =~ m"^PKG_.*_REASON$") {
			# ok
		} elsif ($varname =~ m"^(?:DIST_SUBDIR|WRKSRC)$") {
			$line->log_warning("${pkgvarname} should not be used in ${varname}, as it sometimes includes the PKGREVISION. Please use ${pkgvarname}_NOREV instead.");
		} else {
			$opt_debug_misc and $line->log_debug("Use of PKGNAME in ${varname}.");
		}
	}

	if (exists(get_deprecated_map()->{$varname})) {
		$line->log_warning("Definition of ${varname} is deprecated. ".get_deprecated_map()->{$varname});
	} elsif (exists(get_deprecated_map()->{$varcanon})) {
		$line->log_warning("Definition of ${varname} is deprecated. ".get_deprecated_map()->{$varcanon});
	}

	if ($varname =~ m"^SITES_") {
		$line->log_warning("SITES_* is deprecated. Please use SITES.* instead.");
	}

	if ($value =~ m"^[^=]\@comment") {
		$line->log_warning("Please don't use \@comment in ${varname}.");
		$line->explain_warning(
"Here you are defining a variable containing \@comment. As this value",
"typically includes a space as the last character you probably also used",
"quotes around the variable. This can lead to confusion when adding this",
"variable to PLIST_SUBST, as all other variables are quoted using the :Q",
"operator when they are appended. As it is hard to check whether a",
"variable that is appended to PLIST_SUBST is already quoted or not, you",
"should not have pre-quoted variables at all. To solve this, you should",
"directly use PLIST_SUBST+= ${varname}=${value} or use any other",
"variable for collecting the list of PLIST substitutions and later",
"append that variable with PLIST_SUBST+= \${MY_PLIST_SUBST}.");
	}

	# Mark the variable as PLIST condition. This is later used in
	# checkfile_PLIST.
	if (defined($pkgctx_plist_subst_cond) && $value =~ m"(.+)=.*\@comment.*") {
		$pkgctx_plist_subst_cond->{$1}++;
	}

	use constant op_to_use_time => {
		":="	=> VUC_TIME_LOAD,
		"!="	=> VUC_TIME_LOAD,
		"="	=> VUC_TIME_RUN,
		"+="	=> VUC_TIME_RUN,
		"?="	=> VUC_TIME_RUN
	};

	$used_vars = extract_used_variables($line, $value);
	my $vuc = PkgLint::VarUseContext->new(
		op_to_use_time->{$op},
		get_variable_type($line, $varname),
		VUC_SHELLWORD_UNKNOWN,		# XXX: maybe PLAIN?
		VUC_EXTENT_UNKNOWN
	);
	foreach my $used_var (@{$used_vars}) {
		checkline_mk_varuse($line, $used_var, "", $vuc);
	}
}

# The bmake parser is way too sloppy about syntax, so we need to check
# that here.
#
sub checkline_mk_cond($$) {
	my ($line, $cond) = @_;
	my ($op, $varname, $match, $value);

	$opt_debug_trace and $line->log_debug("checkline_mk_cond($cond)");
	my $tree = parse_mk_cond($line, $cond);
	if (tree_match($tree, ["not", ["empty", ["match", \$varname, \$match]]])) {
		#$line->log_note("tree_match: varname=$varname, match=$match");

		my $type = get_variable_type($line, $varname);
		my $btype = defined($type) ? $type->basic_type : undef;
		if (defined($btype) && ref($type->basic_type) eq "HASH") {
			if ($match !~ m"[\$\[*]" && !exists($btype->{$match})) {
				$line->log_warning("Invalid :M value \"$match\". Only { " . join(" ", sort keys %$btype) . " } are allowed.");
			}
		}

		# Currently disabled because the valid options can also be defined in PKG_OPTIONS_GROUP.*.
		# Additionally, all these variables may have multiple assigments (+=).
		if (false && $varname eq "PKG_OPTIONS" && defined($pkgctx_vardef) && exists($pkgctx_vardef->{"PKG_SUPPORTED_OPTIONS"})) {
			my $options = $pkgctx_vardef->{"PKG_SUPPORTED_OPTIONS"}->get("value");

			if ($match !~ m"[\$\[*]" && index(" $options ", " $match ") == -1) {
				$line->log_warning("Invalid option \"$match\". Only { $options } are allowed.");
			}
		}

		# TODO: PKG_BUILD_OPTIONS. That requires loading the
		# complete package definitition for the package "pkgbase"
		# or some other database. If we could confine all option
		# definitions to options.mk, this would become easier.

	} elsif (tree_match($tree, [\$op, ["var", \$varname], ["string", \$value]])) {
		checkline_mk_vartype($line, $varname, "use", $value, undef);

	}
	# XXX: When adding new cases, be careful that the variables may have
	# been partially initialized by previous calls to tree_match.
	# XXX: Maybe it is better to clear these references at the beginning
	# of tree_match.
}

#
# Procedures to check an array of lines.
#

sub checklines_trailing_empty_lines($) {
	my ($lines) = @_;
	my ($last, $max);

	$max = $#{$lines} + 1;
	for ($last = $max; $last > 1 && $lines->[$last - 1]->text eq ""; ) {
		$last--;
	}
	if ($last != $max) {
		$lines->[$last]->log_note("Trailing empty lines.");
	}
}

sub checklines_package_Makefile_varorder($) {
	my ($lines) = @_;

	return unless $opt_warn_varorder;

	use enum qw(once optional many);
	my (@sections) = (
		[ "Initial comments", once,
			[
			]
		],
		[ "Unsorted stuff, part 1", once,
			[
				[ "DISTNAME", optional ],
				[ "PKGNAME",  optional ],
				[ "PKGREVISION", optional ],
				[ "CATEGORIES", once ],
				[ "MASTER_SITES", optional ],
				[ "DIST_SUBDIR", optional ],
				[ "EXTRACT_SUFX", optional ],
				[ "DISTFILES", many ],
				[ "SITES.*", many ],
			]
		],
		[ "Distribution patches", optional,
			[
				[ "PATCH_SITES", optional ], # or once?
				[ "PATCH_SITE_SUBDIR", optional ],
				[ "PATCHFILES", optional ], # or once?
				[ "PATCH_DIST_ARGS", optional ],
				[ "PATCH_DIST_STRIP", optional ],
				[ "PATCH_DIST_CAT", optional ],
			]
		],
		[ "Unsorted stuff, part 2", once,
			[
				[ "MAINTAINER", optional ],
				[ "OWNER", optional ],
				[ "HOMEPAGE", optional ],
				[ "COMMENT", once ],
				[ "LICENSE", once ],
			]
		],
		[ "Legal issues", optional,
			[
				[ "LICENSE_FILE", optional ],
				[ "RESTRICTED", optional ],
				[ "NO_BIN_ON_CDROM", optional ],
				[ "NO_BIN_ON_FTP", optional ],
				[ "NO_SRC_ON_CDROM", optional ],
				[ "NO_SRC_ON_FTP", optional ],
			]
		],
		[ "Technical restrictions", optional,
			[
				[ "BROKEN_EXCEPT_ON_PLATFORM", many ],
				[ "BROKEN_ON_PLATFORM", many ],
				[ "NOT_FOR_PLATFORM", many ],
				[ "ONLY_FOR_PLATFORM", many ],
				[ "NOT_FOR_COMPILER", many ],
				[ "ONLY_FOR_COMPILER", many ],
				[ "NOT_FOR_UNPRIVILEGED", optional ],
				[ "ONLY_FOR_UNPRIVILEGED", optional ],
			]
		],
		[ "Dependencies", optional,
			[
				[ "BUILD_DEPENDS", many ],
				[ "TOOL_DEPENDS", many ],
				[ "DEPENDS", many ],
			]
		]
	);

	if (!defined($seen_Makefile_common) || $seen_Makefile_common) {
		return;
	}

	my ($lineno, $sectindex, $varindex) = (0, -1, 0);
	my ($next_section, $vars, $below, $below_what) = (true, undef, {}, undef);

	# If the current section is optional but contains non-optional
	# fields, the complete section may be skipped as long as there
	# has not been a non-optional variable.
	my $may_skip_section = false;

	# In each iteration, one of the following becomes true:
	# - new.lineno > old.lineno
	# - new.sectindex > old.sectindex
	# - new.sectindex == old.sectindex && new.varindex > old.varindex
	# - new.next_section == true && old.next_section == false
	while ($lineno <= $#{$lines}) {
		my $line = $lines->[$lineno];
		my $text = $line->text;

		$opt_debug_misc and $line->log_debug("[varorder] section ${sectindex} variable ${varindex}.");

		if ($next_section) {
			$next_section = false;
			$sectindex++;
			last if ($sectindex > $#sections);
			$vars = $sections[$sectindex]->[2];
			$may_skip_section = ($sections[$sectindex]->[1] == optional);
			$varindex = 0;
		}

		if ($text =~ m"^#") {
			$lineno++;

		} elsif ($line->has("varcanon")) {
			my $varcanon = $line->get("varcanon");

			if (exists($below->{$varcanon})) {
				if (defined($below->{$varcanon})) {
					$line->log_warning("${varcanon} appears too late. Please put it below $below->{$varcanon}.");
				} else {
					$line->log_warning("${varcanon} appears too late. It should be the very first definition.");
				}
				$lineno++;
				next;
			}

			while ($varindex <= $#{$vars} && $varcanon ne $vars->[$varindex]->[0] && ($vars->[$varindex]->[1] != once || $may_skip_section)) {
				if ($vars->[$varindex]->[1] == once) {
					$may_skip_section = false;
				}
				$below->{$vars->[$varindex]->[0]} = $below_what;
				$varindex++;
			}
			if ($varindex > $#{$vars}) {
				if ($sections[$sectindex]->[1] != optional) {
					$line->log_warning("Empty line expected.");
				}
				$next_section = true;

			} elsif ($varcanon ne $vars->[$varindex]->[0]) {
				$line->log_warning("Expected " . $vars->[$varindex]->[0] . ", but found " . $varcanon . ".");
				$lineno++;

			} else {
				if ($vars->[$varindex]->[1] != many) {
					$below->{$vars->[$varindex]->[0]} = $below_what;
					$varindex++;
				}
				$lineno++;
			}
			$below_what = $varcanon;

		} else {
			while ($varindex <= $#{$vars}) {
				if ($vars->[$varindex]->[1] == once && !$may_skip_section) {
					$line->log_warning($vars->[$varindex]->[0] . " should be set here.");
				}
				$below->{$vars->[$varindex]->[0]} = $below_what;
				$varindex++;
			}
			$next_section = true;
			if ($text eq "") {
				$below_what = "the previous empty line";
				$lineno++;
			}
		}
	}
}

sub checklines_mk($) {
	my ($lines) = @_;
	my ($allowed_targets) = ({});
	my ($substcontext) = PkgLint::SubstContext->new();

	assert(@{$lines} != 0, "checklines_mk may only be called with non-empty lines.");
	$opt_debug_trace and log_debug($lines->[0]->fname, NO_LINES, "checklines_mk()");

	# Define global variables for the Makefile context.
	$mkctx_indentations = [0];
	$mkctx_target = undef;
	$mkctx_for_variables = {};
	$mkctx_vardef = {};
	$mkctx_build_defs = {};
	$mkctx_plist_vars = {};
	$mkctx_tools = {%{get_predefined_tool_names()}};
	$mkctx_varuse = {};

	determine_used_variables($lines);

	foreach my $prefix (qw(pre do post)) {
		foreach my $action (qw(fetch extract patch tools wrapper configure build test install package clean)) {
			$allowed_targets->{"${prefix}-${action}"} = true;
		}
	}

	#
	# In the first pass, all additions to BUILD_DEFS and USE_TOOLS
	# are collected to make the order of the definitions irrelevant.
	#

	foreach my $line (@{$lines}) {
		next unless $line->has("is_varassign");
		my $varcanon = $line->get("varcanon");

		if ($varcanon eq "BUILD_DEFS" || $varcanon eq "PKG_GROUPS_VARS" || $varcanon eq "PKG_USERS_VARS") {
			foreach my $varname (split(qr"\s+", $line->get("value"))) {
				$mkctx_build_defs->{$varname} = true;
				$opt_debug_misc and $line->log_debug("${varname} is added to BUILD_DEFS.");
			}

		} elsif ($varcanon eq "PLIST_VARS") {
			foreach my $id (split(qr"\s+", $line->get("value"))) {
				$mkctx_plist_vars->{"PLIST.$id"} = true;
				$opt_debug_misc and $line->log_debug("PLIST.${id} is added to PLIST_VARS.");
				use_var($line, "PLIST.$id");
			}

		} elsif ($varcanon eq "USE_TOOLS") {
			foreach my $tool (split(qr"\s+", $line->get("value"))) {
				$tool =~ s/:(build|run)//;
				$mkctx_tools->{$tool} = true;
				$opt_debug_misc and $line->log_debug("${tool} is added to USE_TOOLS.");
			}

		} elsif ($varcanon eq "SUBST_VARS.*") {
			foreach my $svar (split(/\s+/, $line->get("value"))) {
				use_var($svar, varname_canon($svar));
				$opt_debug_misc and $line->log_debug("varuse $svar");
			}

		} elsif ($varcanon eq "OPSYSVARS") {
			foreach my $osvar (split(/\s+/, $line->get("value"))) {
				use_var($line, "$osvar.*");
				def_var($line, $osvar);
			}
		}
	}

	#
	# In the second pass, all "normal" checks are done.
	#

	if (0 <= $#{$lines}) {
		checkline_rcsid_regex($lines->[0], qr"^#\s+", "# ");
	}

	foreach my $line (@{$lines}) {
		my $text = $line->text;

		checkline_trailing_whitespace($line);
		checkline_spellcheck($line);

		if ($line->has("is_empty")) {
			$substcontext->check_end($line);

		} elsif ($line->has("is_comment")) {
			# No further checks.

		} elsif ($text =~ regex_varassign) {
			my ($varname, $op, undef, $comment) = ($1, $2, $3, $4);
			my $space1 = substr($text, $+[1], $-[2] - $+[1]);
			my $align = substr($text, $+[2], $-[3] - $+[2]);
			my $value = $line->get("value");

			if ($align !~ m"^(\t*|[ ])$") {
				$opt_warn_space && $line->log_note("Alignment of variable values should be done with tabs, not spaces.");
				my $prefix = "${varname}${space1}${op}";
				my $aligned_len = tablen("${prefix}${align}");
				if ($aligned_len % 8 == 0) {
					my $tabalign = ("\t" x (($aligned_len - tablen($prefix) + 7) / 8));
					$line->replace("${prefix}${align}", "${prefix}${tabalign}");
				}
			}
			checkline_mk_varassign($line, $varname, $op, $value, $comment);
			$substcontext->check_varassign($line, $varname, $op, $value);

		} elsif ($text =~ regex_mk_shellcmd) {
			my ($shellcmd) = ($1);
			checkline_mk_shellcmd($line, $shellcmd);

		} elsif ($text =~ regex_mk_include) {
			my ($include, $includefile) = ($1, $2);

			$opt_debug_include and $line->log_debug("includefile=${includefile}");
			checkline_relative_path($line, $includefile, $include eq "include");

			if ($includefile =~ m"../Makefile$") {
				$line->log_error("Other Makefiles must not be included directly.");
				$line->explain_warning(
"If you want to include portions of another Makefile, extract",
"the common parts and put them into a Makefile.common. After",
"that, both this one and the other package should include the",
"Makefile.common.");
			}

			if ($includefile eq "../../mk/bsd.prefs.mk") {
				if ($line->fname =~ m"buildlink3\.mk$") {
					$line->log_note("For efficiency reasons, please include bsd.fast.prefs.mk instead of bsd.prefs.mk.");
				}
				$seen_bsd_prefs_mk = true;
			} elsif ($includefile eq "../../mk/bsd.fast.prefs.mk") {
				$seen_bsd_prefs_mk = true;
			}

			if ($includefile =~ m"/x11-links/buildlink3\.mk$") {
				$line->log_error("${includefile} must not be included directly. Include \"../../mk/x11.buildlink3.mk\" instead.");
			}
			if ($includefile =~ m"/jpeg/buildlink3\.mk$") {
				$line->log_error("${includefile} must not be included directly. Include \"../../mk/jpeg.buildlink3.mk\" instead.");
			}
			if ($includefile =~ m"/intltool/buildlink3\.mk$") {
				$line->log_warning("Please say \"USE_TOOLS+= intltool\" instead of this line.");
			}
			if ($includefile =~ m"(.*)/builtin\.mk$") {
				my ($dir) = ($1);
				$line->log_error("${includefile} must not be included directly. Include \"${dir}/buildlink3.mk\" instead.");
			}

		} elsif ($text =~ regex_mk_sysinclude) {
			my ($includefile, $comment) = ($1, $2);

			# No further action.

		} elsif ($text =~ regex_mk_cond) {
			my ($indent, $directive, $args, $comment) = ($1, $2, $3, $4);

			use constant regex_directives_with_args => qr"^(?:if|ifdef|ifndef|elif|for|undef)$";

			if ($directive =~ m"^(?:endif|endfor|elif|else)$") {
				if ($#{$mkctx_indentations} >= 1) {
					pop(@{$mkctx_indentations});
				} else {
					$line->log_error("Unmatched .${directive}.");
				}
			}

			# Check the indentation
			if ($indent ne " " x $mkctx_indentations->[-1]) {
				$opt_warn_space and $line->log_note("This directive should be indented by ".$mkctx_indentations->[-1]." spaces.");
			}

			if ($directive eq "if" && $args =~ m"^!defined\([\w]+_MK\)$") {
				push(@{$mkctx_indentations}, $mkctx_indentations->[-1]);

			} elsif ($directive =~ m"^(?:if|ifdef|ifndef|for|elif|else)$") {
				push(@{$mkctx_indentations}, $mkctx_indentations->[-1] + 2);
			}

			if ($directive =~ regex_directives_with_args && !defined($args)) {
				$line->log_error("\".${directive}\" must be given some arguments.");

			} elsif ($directive !~ regex_directives_with_args && defined($args)) {
				$line->log_error("\".${directive}\" does not take arguments.");

				if ($directive eq "else") {
					$line->log_note("If you meant \"else if\", use \".elif\".");
				}

			} elsif ($directive eq "if" || $directive eq "elif") {
				checkline_mk_cond($line, $args);

			} elsif ($directive eq "ifdef" || $directive eq "ifndef") {
				if ($args =~ m"\s") {
					$line->log_error("The \".${directive}\" directive can only handle _one_ argument.");
				} else {
					$line->log_warning("The \".${directive}\" directive is deprecated. Please use \".if "
						. (($directive eq "ifdef" ? "" : "!"))
						. "defined(${args})\" instead.");
				}

			} elsif ($directive eq "for") {
				if ($args =~ m"^(\S+(?:\s*\S+)*?)\s+in\s+(.*)$") {
					my ($vars, $values) = ($1, $2);

					foreach my $var (split(qr"\s+", $vars)) {
						if (!$is_internal && $var =~ m"^_") {
							$line->log_warning("Variable names starting with an underscore are reserved for internal pkgsrc use.");
						}

						if ($var =~ m"^[_a-z][_a-z0-9]*$") {
							# Fine.
						} elsif ($var =~ m"[A-Z]") {
							$line->log_warning(".for variable names should not contain uppercase letters.");
						} else {
							$line->log_error("Invalid variable name \"${var}\".");
						}

						$mkctx_for_variables->{$var} = true;
					}

					# Check if any of the value's types is not guessed.
					my $guessed = true;
					foreach my $value (split(qr"\s+", $values)) { # XXX: too simple
						if ($value =~ m"^\$\{(.*)\}") {
							my $type = get_variable_type($line, $1);
							if (defined($type) && !$type->is_guessed()) {
								$guessed = false;
							}
						}
					}

					my $for_loop_type = PkgLint::Type->new(
						LK_INTERNAL,
						"Unchecked",
						[[qr".*", "pu"]],
						$guessed
					);
					my $for_loop_context = PkgLint::VarUseContext->new(
						VUC_TIME_LOAD,
						$for_loop_type,
						VUC_SHELLWORD_FOR,
						VUC_EXTENT_WORD
					);
					foreach my $var (@{extract_used_variables($line, $values)}) {
						checkline_mk_varuse($line, $var, "", $for_loop_context);
					}

				}

			} elsif ($directive eq "undef" && defined($args)) {
				foreach my $var (split(qr"\s+", $args)) {
					if (exists($mkctx_for_variables->{$var})) {
						$line->log_note("Using \".undef\" after a \".for\" loop is unnecessary.");
					}
				}
			}

		} elsif ($text =~ regex_mk_dependency) {
			my ($targets, $whitespace, $dependencies, $comment) = ($1, $2, $3, $4);

			$opt_debug_misc and $line->log_debug("targets=${targets}, dependencies=${dependencies}");
			$mkctx_target = $targets;

			foreach my $source (split(/\s+/, $dependencies)) {
				if ($source eq ".PHONY") {
					foreach my $target (split(/\s+/, $targets)) {
						$allowed_targets->{$target} = true;
					}
				}
			}

			foreach my $target (split(/\s+/, $targets)) {
				if ($target eq ".PHONY") {
					foreach my $dep (split(qr"\s+", $dependencies)) {
						$allowed_targets->{$dep} = true;
					}

				} elsif ($target eq ".ORDER") {
					# TODO: Check for spelling mistakes.

				} elsif (!exists($allowed_targets->{$target})) {
					$line->log_warning("Unusual target \"${target}\".");
					$line->explain_warning(
"If you really want to define your own targets, you can \"declare\"",
"them by inserting a \".PHONY: my-target\" line before this line. This",
"will tell make(1) to not interpret this target's name as a filename.");
				}
			}

		} elsif ($text =~ m"^\.\s*(\S*)") {
			my ($directive) = ($1);

			$line->log_error("Unknown directive \".${directive}\".");

		} elsif ($text =~ m"^ ") {
			$line->log_warning("Makefile lines should not start with space characters.");
			$line->explain_warning(
"If you want this line to contain a shell program, use a tab",
"character for indentation. Otherwise please remove the leading",
"white-space.");

		} else {
			$line->log_error("[Internal] Unknown line format: $text");
		}
	}
	if (@{$lines} > 0) {
		$substcontext->check_end($lines->[-1]);
	}

	checklines_trailing_empty_lines($lines);

	if ($#{$mkctx_indentations} != 0) {
		$lines->[-1]->log_error("Directive indentation is not 0, but ".$mkctx_indentations->[-1]." at EOF.");
	}

	# Clean up global variables.
	$mkctx_for_variables = undef;
	$mkctx_indentations = undef;
	$mkctx_target = undef;
	$mkctx_vardef = undef;
	$mkctx_build_defs = undef;
	$mkctx_plist_vars = undef;
	$mkctx_tools = undef;
	$mkctx_varuse = undef;
}

sub checklines_buildlink3_inclusion($) {
	my ($lines) = @_;
	my ($included_files);

	assert(@{$lines} != 0, "The lines array must be non-empty.");
	$opt_debug_trace and log_debug($lines->[0]->fname, NO_LINES, "checklines_buildlink3_inclusion()");

	if (!defined($pkgctx_bl3)) {
		return;
	}

	# Collect all the included buildlink3.mk files from the file.
	$included_files = {};
	foreach my $line (@{$lines}) {
		if ($line->text =~ regex_mk_include) {
			my (undef, $file, $comment) = ($1, $2, $3);

			if ($file =~ m"^\.\./\.\./(.*)/buildlink3\.mk") {
				my ($bl3) = ($1);

				$included_files->{$bl3} = $line;
				if (!exists($pkgctx_bl3->{$bl3})) {
					$line->log_warning("${bl3}/buildlink3.mk is included by this file but not by the package.");
				}
			}
		}
	}

	# Print debugging messages for all buildlink3.mk files that are
	# included by the package but not by this buildlink3.mk file.
	foreach my $package_bl3 (sort(keys(%{$pkgctx_bl3}))) {
		if (!exists($included_files->{$package_bl3})) {
			$opt_debug_misc and $pkgctx_bl3->{$package_bl3}->log_debug("${package_bl3}/buildlink3.mk is included by the package but not by the buildlink3.mk file.");
		}
	}
}

#
# Procedures to check a single file.
#

sub checkfile_ALTERNATIVES($) {
	my ($fname) = @_;
	my ($lines);

	$opt_debug_trace and log_debug($fname, NO_LINES, "checkfile_ALTERNATIVES()");

	checkperms($fname);
	if (!($lines = load_file($fname))) {
		log_error($fname, NO_LINE_NUMBER, "Cannot be read.");
		return;
	}
}

sub checkfile_buildlink3_mk($) {
	my ($fname) = @_;
	my ($lines, $lineno, $m);

	$opt_debug_trace and log_debug($fname, NO_LINES, "checkfile_buildlink3_mk()");

	checkperms($fname);
	if (!($lines = load_lines($fname, true))) {
		log_error($fname, NO_LINE_NUMBER, "Cannot be read.");
		return;
	}
	if (@{$lines} == 0) {
		log_error($fname, NO_LINES, "Must not be empty.");
		return;
	}

	parselines_mk($lines);
	checklines_mk($lines);

	$lineno = 0;

	# Header comments
	while ($lineno <= $#{$lines} && (my $text = $lines->[$lineno]->text) =~ m"^#") {
		if ($text =~ m"^# XXX") {
			$lines->[$lineno]->log_note("Please read this comment and remove it if appropriate.");
		}
		$lineno++;
	}
	expect_empty_line($lines, \$lineno);

	if (expect($lines, \$lineno, qr"^BUILDLINK_DEPMETHOD\.(\S+)\?=.*$")) {
		$lines->[$lineno - 1]->log_warning("This line belongs inside the .ifdef block.");
		while ($lines->[$lineno]->text eq "") {
			$lineno++;
		}
	}

	if (!($m = expect($lines, \$lineno, qr"^BUILDLINK_TREE\+=\s*(\S+)$"))) {
		$lines->[$lineno]->log_warning("Expected a BUILDLINK_TREE line.");
		return;
	}

	checklines_buildlink3_mk($lines, $lineno, $m->text(1));
}

sub checklines_buildlink3_mk($$$) {
	my ($lines, $lineno, $pkgid) = @_;
	my ($m);
	my ($bl_PKGBASE_line, $bl_PKGBASE);
	my ($bl_pkgbase_line, $bl_pkgbase);
	my ($abi_line, $abi_pkg, $abi_version);
	my ($api_line, $api_pkg, $api_version);

	# First paragraph: Introduction of the package identifier
	$bl_pkgbase_line = $lines->[$lineno - 1];
	$bl_pkgbase = $pkgid;
	$opt_debug_misc and $bl_pkgbase_line->log_debug("bl_pkgbase=${bl_pkgbase}");
	expect_empty_line($lines, \$lineno);

	# Second paragraph: multiple inclusion protection and introduction
	# of the uppercase package identifier.
	return unless ($m = expect_re($lines, \$lineno, qr"^\.if !defined\((\S+)_BUILDLINK3_MK\)$"));
	$bl_PKGBASE_line = $lines->[$lineno - 1];
	$bl_PKGBASE = $m->text(1);
	$opt_debug_misc and $bl_PKGBASE_line->log_debug("bl_PKGBASE=${bl_PKGBASE}");
	expect_re($lines, \$lineno, qr"^\Q$bl_PKGBASE\E_BUILDLINK3_MK:=$");
	expect_empty_line($lines, \$lineno);

	my $norm_bl_pkgbase = $bl_pkgbase;
	$norm_bl_pkgbase =~ s/-/_/g;
	$norm_bl_pkgbase = uc($norm_bl_pkgbase);
	if ($norm_bl_pkgbase ne $bl_PKGBASE) {
		$bl_PKGBASE_line->log_error("Package name mismatch between ${bl_PKGBASE} ...");
		$bl_pkgbase_line->log_error("... and ${bl_pkgbase}.");
	}
	if (defined($effective_pkgbase) && $effective_pkgbase ne $bl_pkgbase) {
		$bl_pkgbase_line->log_error("Package name mismatch between ${bl_pkgbase} ...");
		$effective_pkgname_line->log_error("... and ${effective_pkgbase}.");
	}

	# Third paragraph: Package information.
	my $if_level = 1; # the first .if is from the second paragraph.
	while (true) {

		if ($lineno > $#{$lines}) {
			lines_log_warning($lines, $lineno, "Expected .endif");
			return;
		}

		my $line = $lines->[$lineno];

		if (($m = expect($lines, \$lineno, regex_varassign))) {
			my ($varname, $value) = ($m->text(1), $m->text(3));
			my $do_check = false;

			if ($varname eq "BUILDLINK_ABI_DEPENDS.${bl_pkgbase}") {
				$abi_line = $line;
				if ($value =~ regex_dependency_lge) {
					($abi_pkg, $abi_version) = ($1, $2);
				} elsif ($value =~ regex_dependency_wildcard) {
					($abi_pkg) = ($1);
				} else {
					$opt_debug_unchecked and $line->log_debug("Unchecked dependency pattern \"${value}\".");
				}
				$do_check = true;
			}
			if ($varname eq "BUILDLINK_API_DEPENDS.${bl_pkgbase}") {
				$api_line = $line;
				if ($value =~ regex_dependency_lge) {
					($api_pkg, $api_version) = ($1, $2);
				} elsif ($value =~ regex_dependency_wildcard) {
					($api_pkg) = ($1);
				} else {
					$opt_debug_unchecked and $line->log_debug("Unchecked dependency pattern \"${value}\".");
				}
				$do_check = true;
			}
			if ($do_check && defined($abi_pkg) && defined($api_pkg)) {
				if ($abi_pkg ne $api_pkg) {
					$abi_line->log_warning("Package name mismatch between ${abi_pkg} ...");
					$api_line->log_warning("... and ${api_pkg}.");
				}
			}
			if ($do_check && defined($abi_version) && defined($api_version)) {
				if (!dewey_cmp($abi_version, ">=", $api_version)) {
					$abi_line->log_warning("ABI version (${abi_version}) should be at least ...");
					$api_line->log_warning("... API version (${api_version}).");
				}
			}

			if ($varname =~ m"^BUILDLINK_[\w_]+\.(.*)$") {
				my ($varparam) = ($1);

				if ($varparam ne $bl_pkgbase) {
					$line->log_warning("Only buildlink variables for ${bl_pkgbase}, not ${varparam} may be set in this file.");
				}
			}

			if ($varname eq "pkgbase") {
				expect_re($lines, \$lineno, qr"^\.\s*include \"../../mk/pkg-build-options.mk\"$");
			}

			# TODO: More checks.

		} elsif (expect($lines, \$lineno, qr"^(?:#.*)?$")) {
			# Comments and empty lines are fine here.

		} elsif (expect($lines, \$lineno, qr"^\.\s*include \"\.\./\.\./([^/]+/[^/]+)/buildlink3\.mk\"$")
			|| expect($lines, \$lineno, qr"^\.\s*include \"\.\./\.\./mk/(\S+)\.buildlink3\.mk\"$")) {
			# TODO: Maybe check dependency lines.

		} elsif (expect($lines, \$lineno, qr"^\.if\s")) {
			$if_level++;

		} elsif (expect($lines, \$lineno, qr"^\.endif.*$")) {
			$if_level--;
			last if $if_level == 0;

		} else {
			$opt_debug_unchecked and lines_log_warning($lines, $lineno, "Unchecked line in third paragraph.");
			$lineno++;
		}
	}
	if (!defined($api_line)) {
		$lines->[$lineno - 1]->log_warning("Definition of BUILDLINK_API_DEPENDS is missing.");
	}
	expect_empty_line($lines, \$lineno);

	# Fourth paragraph: Cleanup, corresponding to the first paragraph.
	return unless expect_re($lines, \$lineno, qr"^BUILDLINK_TREE\+=\s*-\Q$bl_pkgbase\E$");

	if ($lineno <= $#{$lines}) {
		$lines->[$lineno]->log_warning("The file should end here.");
	}

	checklines_buildlink3_inclusion($lines);
}

sub checkfile_DESCR($) {
	my ($fname) = @_;
	my ($maxchars, $maxlines) = (80, 24);
	my ($lines);

	$opt_debug_trace and log_debug($fname, NO_LINES, "checkfile_DESCR()");

	checkperms($fname);
	if (!($lines = load_file($fname))) {
		log_error($fname, NO_LINE_NUMBER, "Cannot be read.");
		return;
	}
	if (@{$lines} == 0) {
		log_error($fname, NO_LINE_NUMBER, "Must not be empty.");
		return;
	}

	foreach my $line (@{$lines}) {
		checkline_length($line, $maxchars);
		checkline_trailing_whitespace($line);
		checkline_valid_characters($line, regex_validchars);
		checkline_spellcheck($line);
		if ($line->text =~ m"\$\{") {
			$line->log_warning("Variables are not expanded in the DESCR file.");
		}
	}
	checklines_trailing_empty_lines($lines);

	if (@{$lines} > $maxlines) {
		my $line = $lines->[$maxlines];

		$line->log_warning("File too long (should be no more than $maxlines lines).");
		$line->explain_warning(
"A common terminal size is 80x25 characters. The DESCR file should",
"fit on one screen. It is also intended to give a _brief_ summary",
"about the package's contents.");
	}
	autofix($lines);
}

sub checkfile_distinfo($) {
	my ($fname) = @_;
	my ($lines, %in_distinfo, $patches_dir, $di_is_committed, $current_fname, $is_patch, @seen_algs);

	$opt_debug_trace and log_debug($fname, NO_LINES, "checkfile_distinfo()");

	$di_is_committed = is_committed($fname);

	checkperms($fname);
	if (!($lines = load_file($fname))) {
		log_error($fname, NO_LINE_NUMBER, "Cannot be read.");
		return;
	}

	if (@{$lines} == 0) {
		log_error($fname, NO_LINE_NUMBER, "Must not be empty.");
		return;
	}

	checkline_rcsid($lines->[0], "");
	if (1 <= $#{$lines} && $lines->[1]->text ne "") {
		$lines->[1]->log_note("Empty line expected.");
		$lines->[1]->explain_note("This is merely for aesthetical purposes.");
	}

	$patches_dir = $patchdir;
	if (!defined($patches_dir) && -d "${current_dir}/patches") {
		$patches_dir = "patches";
	}

	my $on_filename_change = sub($$) {
		my ($line, $new_fname) = @_;

		if (defined($current_fname)) {
			my $seen_algs = join(", ", @seen_algs);
			$opt_debug_misc and $line->log_debug("File ${current_fname} has checksums ${seen_algs}.");
			if ($is_patch) {
				if ($seen_algs ne "SHA1") {
					$line->log_error("Expected SHA1 checksum for ${current_fname}, got ${seen_algs}.");
				}
			} else {
				if ($seen_algs ne "SHA1, RMD160, Size" && $seen_algs ne "SHA1, RMD160, SHA512, Size") {
					$line->log_error("Expected SHA1, RMD160, Size checksums for ${current_fname}, got ${seen_algs}.");
				}
			}
		}

		$is_patch = defined($new_fname) && $new_fname =~ m"^patch-.+$" ? true : false;
		$current_fname = $new_fname;
		@seen_algs = ();
	};

	foreach my $line (@{$lines}[2..$#{$lines}]) {
		if ($line->text !~ m"^(\w+) \(([^)]+)\) = (.*)(?: bytes)?$") {
			$line->log_error("Unknown line type.");
			next;
		}
		my ($alg, $chksum_fname, $sum) = ($1, $2, $3);

		if (!defined($current_fname) || $chksum_fname ne $current_fname) {
			$on_filename_change->($line, $chksum_fname);
		}

		if ($chksum_fname !~ m"^\w") {
			$line->log_error("All file names should start with a letter.");
		}

		# Inter-package check for differing distfile checksums.
		if ($opt_check_global && !$is_patch) {
			# Note: Perl-specific auto-population.
			if (exists($ipc_distinfo->{$alg}->{$chksum_fname})) {
				my $other = $ipc_distinfo->{$alg}->{$chksum_fname};

				if ($other->[1] eq $sum) {
					# Fine.
				} else {
					$line->log_error("The ${alg} checksum for ${chksum_fname} differs ...");
					$other->[0]->log_error("... from this one.");
				}
			} else {
				$ipc_distinfo->{$alg}->{$chksum_fname} = [$line, $sum];
			}
		}

		push(@seen_algs, $alg);

		if ($is_patch && defined($patches_dir) && !(defined($distinfo_file) && $distinfo_file eq "./../../lang/php5/distinfo")) {
			my $fname = "${current_dir}/${patches_dir}/${chksum_fname}";
			if ($di_is_committed && !is_committed($fname)) {
				$line->log_warning("${patches_dir}/${chksum_fname} is registered in distinfo but not added to CVS.");
			}

			if (open(my $patchfile, "<", $fname)) {
				my $sha1 = Digest::SHA1->new();
				while (defined(my $patchline = <$patchfile>)) {
					$sha1->add($patchline) unless $patchline =~ m"\$[N]etBSD";
				}
				close($patchfile);
				my $chksum = $sha1->hexdigest();
				if ($sum ne $chksum) {
					$line->log_error("${alg} checksum of ${chksum_fname} differs (expected ${sum}, got ${chksum}). Rerun '".conf_make." makepatchsum'.");
				}
			} else {
				$line->log_warning("${chksum_fname} does not exist.");
				$line->explain_warning(
"All patches that are mentioned in a distinfo file should actually exist.",
"What's the use of a checksum if there is no file to check?");
			}
		}
		$in_distinfo{$chksum_fname} = true;

	}
	$on_filename_change->(PkgLint::Line->new($fname, NO_LINE_NUMBER, "", []), undef);
	checklines_trailing_empty_lines($lines);

	if (defined($patches_dir)) {
		foreach my $patch (glob("${current_dir}/${patches_dir}/patch-*")) {
			$patch = basename($patch);
			if (!exists($in_distinfo{$patch})) {
				log_error($fname, NO_LINE_NUMBER, "$patch is not recorded. Rerun '".conf_make." makepatchsum'.");
			}
		}
	}
}

sub checkfile_extra($) {
	my ($fname) = @_;

	$opt_debug_trace and log_debug($fname, NO_LINES, "checkfile_extra()");

	my $lines = load_file($fname) or return log_error($fname, NO_LINE_NUMBER, "Could not be read.");
	checklines_trailing_empty_lines($lines);
	checkperms($fname);
}

sub checkfile_INSTALL($) {
	my ($fname) = @_;

	$opt_debug_trace and log_debug($fname, NO_LINES, "checkfile_INSTALL()");

	checkperms($fname);
	my $lines = load_file($fname) or return log_error($fname, NO_LINE_NUMBER, "Cannot be read.");
}

sub checkfile_MESSAGE($) {
	my ($fname) = @_;

	my @explanation = (
		"A MESSAGE file should consist of a header line, having 75 \"=\"",
		"characters, followed by a line containing only the RCS Id, then an",
		"empty line, your text and finally the footer line, which is the",
		"same as the header line.");

	$opt_debug_trace and log_debug($fname, NO_LINES, "checkfile_MESSAGE()");

	checkperms($fname);
	my $lines = load_file($fname) or return log_error($fname, NO_LINE_NUMBER, "Cannot be read.");

	if (@{$lines} < 3) {
		log_warning($fname, NO_LINE_NUMBER, "File too short.");
		explain_warning($fname, NO_LINE_NUMBER, @explanation);
		return;
	}
	if ($lines->[0]->text ne "=" x 75) {
		$lines->[0]->log_warning("Expected a line of exactly 75 \"=\" characters.");
		explain_warning($fname, NO_LINE_NUMBER, @explanation);
	}
	checkline_rcsid($lines->[1], "");
	foreach my $line (@{$lines}) {
		checkline_length($line, 80);
		checkline_trailing_whitespace($line);
		checkline_valid_characters($line, regex_validchars);
		checkline_spellcheck($line);
	}
	if ($lines->[-1]->text ne "=" x 75) {
		$lines->[-1]->log_warning("Expected a line of exactly 75 \"=\" characters.");
		explain_warning($fname, NO_LINE_NUMBER, @explanation);
	}
	checklines_trailing_empty_lines($lines);
}

sub checkfile_mk($) {
	my ($fname) = @_;

	$opt_debug_trace and log_debug($fname, NO_LINES, "checkfile_mk()");

	checkperms($fname);
	my $lines = load_lines($fname, true) or return log_error($fname, NO_LINE_NUMBER, "Cannot be read.");

	parselines_mk($lines);
	checklines_mk($lines);
	autofix($lines);
}

sub checkfile_package_Makefile($$) {
	my ($fname, $lines) = @_;

	$opt_debug_trace and log_debug($fname, NO_LINES, "checkfile_package_Makefile(..., ...)");

	checkperms($fname);

	if (!exists($pkgctx_vardef->{"PLIST_SRC"})
		&& !exists($pkgctx_vardef->{"GENERATE_PLIST"})
		&& !exists($pkgctx_vardef->{"META_PACKAGE"})
		&& defined($pkgdir)
		&& !-f "${current_dir}/$pkgdir/PLIST"
		&& !-f "${current_dir}/$pkgdir/PLIST.common") {
		log_warning($fname, NO_LINE_NUMBER, "Neither PLIST nor PLIST.common exist, and PLIST_SRC is unset. Are you sure PLIST handling is ok?");
	}

	if ((exists($pkgctx_vardef->{"NO_CHECKSUM"}) || $pkgctx_vardef->{"META_PACKAGE"}) && is_emptydir("${current_dir}/${patchdir}")) {
		if (-f "${current_dir}/${distinfo_file}") {
			log_warning("${current_dir}/${distinfo_file}", NO_LINE_NUMBER, "This file should not exist if NO_CHECKSUM or META_PACKAGE is set.");
		}
	} else {
		if (!-f "${current_dir}/${distinfo_file}") {
			log_warning("${current_dir}/${distinfo_file}", NO_LINE_NUMBER, "File not found. Please run '".conf_make." makesum'.");
		}
	}

	if (exists($pkgctx_vardef->{"REPLACE_PERL"}) && exists($pkgctx_vardef->{"NO_CONFIGURE"})) {
		$pkgctx_vardef->{"REPLACE_PERL"}->log_warning("REPLACE_PERL is ignored when ...");
		$pkgctx_vardef->{"NO_CONFIGURE"}->log_warning("... NO_CONFIGURE is set.");
	}

	if (!exists($pkgctx_vardef->{"LICENSE"})) {
		log_error($fname, NO_LINE_NUMBER, "Each package must define its LICENSE.");
	}

	if (exists($pkgctx_vardef->{"GNU_CONFIGURE"}) && exists($pkgctx_vardef->{"USE_LANGUAGES"})) {
		my $languages_line = $pkgctx_vardef->{"USE_LANGUAGES"};
		my $value = $languages_line->get("value");

		if ($languages_line->has("comment") && $languages_line->get("comment") =~ m"\b(?:c|empty|none)\b"i) {
			# Don't emit a warning, since the comment
			# probably contains a statement that C is
			# really not needed.

		} elsif ($value !~ m"(?:^|\s+)(?:c|c99|objc)(?:\s+|$)") {
			$pkgctx_vardef->{"GNU_CONFIGURE"}->log_warning("GNU_CONFIGURE almost always needs a C compiler, ...");
			$languages_line->log_warning("... but \"c\" is not added to USE_LANGUAGES.");
		}
	}

	my $distname_line = $pkgctx_vardef->{"DISTNAME"};
	my $pkgname_line = $pkgctx_vardef->{"PKGNAME"};

	my $distname = defined($distname_line) ? $distname_line->get("value") : undef;
	my $pkgname = defined($pkgname_line) ? $pkgname_line->get("value") : undef;
	my $nbpart = get_nbpart();

	# Let's do some tricks to get the proper value of the package
	# name more often.
	if (defined($distname) && defined($pkgname)) {
		$pkgname =~ s/\$\{DISTNAME\}/$distname/;

		if ($pkgname =~ m"^(.*)\$\{DISTNAME:S(.)([^:]*)\2([^:]*)\2(g?)\}(.*)$") {
			my ($before, $separator, $old, $new, $mod, $after) = ($1, $2, $3, $4, $5, $6);
			my $newname = $distname;
			$old = quotemeta($old);
			$old =~ s/^\\\^/^/;
			$old =~ s/\\\$$/\$/;
			if ($mod eq "g") {
				$newname =~ s/$old/$new/g;
			} else {
				$newname =~ s/$old/$new/;
			}
			$opt_debug_misc and $pkgname_line->log_debug("old pkgname=$pkgname");
			$pkgname = $before . $newname . $after;
			$opt_debug_misc and $pkgname_line->log_debug("new pkgname=$pkgname");
		}
	}

	if (defined($pkgname) && defined($distname) && $pkgname eq $distname) {
		$pkgname_line->log_note("PKGNAME is \${DISTNAME} by default. You probably don't need to define PKGNAME.");
	}

	if (!defined($pkgname) && defined($distname) && $distname !~ regex_unresolved && $distname !~ regex_pkgname) {
		$distname_line->log_warning("As DISTNAME is not a valid package name, please define the PKGNAME explicitly.");
	}

	($effective_pkgname, $effective_pkgname_line, $effective_pkgbase, $effective_pkgversion)
		= (defined($pkgname) && $pkgname !~ regex_unresolved && $pkgname =~ regex_pkgname) ? ($pkgname.$nbpart, $pkgname_line, $1, $2)
		: (defined($distname) && $distname !~ regex_unresolved && $distname =~ regex_pkgname) ? ($distname.$nbpart, $distname_line, $1, $2)
		: (undef, undef, undef, undef);
	if (defined($effective_pkgname_line)) {
		$opt_debug_misc and $effective_pkgname_line->log_debug("Effective name=${effective_pkgname} base=${effective_pkgbase} version=${effective_pkgversion}.");
		# XXX: too many false positives
		if (false && $pkgpath =~ m"/([^/]+)$" && $effective_pkgbase ne $1) {
			$effective_pkgname_line->log_warning("Mismatch between PKGNAME ($effective_pkgname) and package directory ($1).");
		}
	}

	checkpackage_possible_downgrade();

	if (!exists($pkgctx_vardef->{"COMMENT"})) {
		log_warning($fname, NO_LINE_NUMBER, "No COMMENT given.");
	}

	if (exists($pkgctx_vardef->{"USE_IMAKE"}) && exists($pkgctx_vardef->{"USE_X11"})) {
		$pkgctx_vardef->{"USE_IMAKE"}->log_note("USE_IMAKE makes ...");
		$pkgctx_vardef->{"USE_X11"}->log_note("... USE_X11 superfluous.");
	}

	if (defined($effective_pkgbase)) {

		foreach my $suggested_update (@{get_suggested_package_updates()}) {
			my ($line, $suggbase, $suggver, $suggcomm) = @{$suggested_update};
			my $comment = (defined($suggcomm) ? " (${suggcomm})" : "");

			next unless $effective_pkgbase eq $suggbase;

			if (dewey_cmp($effective_pkgversion, "<", $suggver)) {
				$effective_pkgname_line->log_warning("This package should be updated to ${suggver}${comment}.");
				$effective_pkgname_line->explain_warning(
"The wishlist for package updates in doc/TODO mentions that a newer",
"version of this package is available.");
			}
			if (dewey_cmp($effective_pkgversion, "==", $suggver)) {
				$effective_pkgname_line->log_note("The update request to ${suggver} from doc/TODO${comment} has been done.");
			}
			if (dewey_cmp($effective_pkgversion, ">", $suggver)) {
				$effective_pkgname_line->log_note("This package is newer than the update request to ${suggver}${comment}.");
			}
		}
	}

	checklines_mk($lines);
	checklines_package_Makefile_varorder($lines);
	autofix($lines);
}


sub checkfile_PLIST($) {
	my ($fname) = @_;
	my ($lines, $last_file_seen);

	$opt_debug_trace and log_debug($fname, NO_LINES, "checkfile_PLIST()");

	checkperms($fname);
	if (!($lines = load_file($fname))) {
		log_error($fname, NO_LINE_NUMBER, "Cannot be read.");
		return;
	}
	if (@{$lines} == 0) {
		log_error($fname, NO_LINE_NUMBER, "Must not be empty.");
		return;
	}
	checkline_rcsid($lines->[0], "\@comment ");

	if (@$lines == 1) {
		$lines->[0]->log_warning("PLIST files shouldn't be empty.");
		$lines->[0]->explain_warning(
"One reason for empty PLISTs is that this is a newly created package",
"and that the author didn't run \"bmake print-PLIST\" after installing",
"the files.",
"",
"Another reason, common for Perl packages, is that the final PLIST is",
"automatically generated. Since the source PLIST is not used at all,",
"you can remove it.",
"",
"Meta packages also don't need a PLIST file.");
	}

	# Get the list of all files from the PLIST.
	my $all_files = {};
	my $all_dirs = {};
	my $extra_lines = [];
	if (basename($fname) eq "PLIST.common_end") {
		my $common_lines = load_file(dirname($fname) . "/PLIST.common");
		if ($common_lines) {
			$extra_lines = $common_lines;
		}
	}

	foreach my $line (@{$extra_lines}, @{$lines}) {
		my $text = $line->text;

		if (index($text, '${') != -1 && $text =~ m"\$\{([\w_]+)\}(.*)") {
			if (defined($pkgctx_plist_subst_cond) && exists($pkgctx_plist_subst_cond->{$1})) {
				$opt_debug_misc and $line->log_debug("Removed PLIST_SUBST conditional $1.");
				$text = $2;
			}
		}

		if ($text =~ m"^[\w\$]") {
			$all_files->{$text} = $line;
			my $dir = $text;
			while ($dir =~ s,/[^/]+$,,) {
				$all_dirs->{$dir} = $line;
			}
		}
		if (substr($text, 0, 1) eq '@' && $text =~ m"^\@exec \$\{MKDIR\} %D/(.*)$") {
			my $dir = $1;
			do {
				$all_dirs->{$dir} = $line;
			} while ($dir =~ s,/[^/]+$,,);
		}
	}

	foreach my $line (@{$lines}) {
		my $text = $line->text;

		if ($text =~ /\s$/) {
			$line->log_error("pkgsrc does not support filenames ending in white-space.");
			$line->explain_error(
"Each character in the PLIST is relevant, even trailing white-space.");
		}

		# @foo directives.
		if (index($text, '@') != -1 && $text =~ /^(?:\$\{[\w_]+\})?\@([a-z-]+)\s+(.*)/) {
			my ($cmd, $arg) = ($1, $2);

			if ($cmd eq "unexec" && $arg =~ m"^(rmdir|\$\{RMDIR\} \%D/)(.*)") {
				my ($rmdir, $rest) = ($1, $2);
				if ($rest !~ m"(?:true|\$\{TRUE\})") {
					$line->log_warning("Please remove this line. It is no longer necessary.");
				}

			} elsif (($cmd eq "exec" || $cmd eq "unexec")) {
				if ($arg =~ /(?:install-info|\$\{INSTALL_INFO\})/) {
					$line->log_warning("\@exec/unexec install-info is deprecated.");

				} elsif ($arg =~ /ldconfig/ && $arg !~ m"/usr/bin/true") {
					$line->log_error("ldconfig must be used with \"||/usr/bin/true\".");
				}

			} elsif ($cmd eq "comment") {
				# nothing to do

			} elsif ($cmd eq "dirrm") {
				$line->log_warning("\@dirrm is obsolete. Please remove this line.");
				$line->explain_warning(
"Directories are removed automatically when they are empty.",
"When a package needs an empty directory, it can use the \@pkgdir",
"command in the PLIST");
			} elsif ($cmd eq "imake-man") {
				my (@args) = split(/\s+/, $arg);
				if (@args != 3) {
					$line->log_warning("Invalid number of arguments for imake-man.");
				} else {
					if ($args[2] eq "\${IMAKE_MANNEWSUFFIX}") {
						warn_about_PLIST_imake_mannewsuffix($line);
					}
				}

			} elsif ($cmd eq "pkgdir") {
				# TODO: What can we check here?

			} else {
				$line->log_warning("Unknown PLIST directive \"\@$cmd\".");
			}

		# Pathnames.
		} elsif ($text =~ m"^([A-Za-z0-9\$].*)/([^/]+)$") {
			my ($dirname, $basename) = ($1, $2);

			if ($opt_warn_plist_sort && $text =~ m"^\w" && $text !~ regex_unresolved) {
				if (defined($last_file_seen)) {
					if ($last_file_seen gt $text) {
						$line->log_warning("${text} should be sorted before ${last_file_seen}.");
						$line->explain_warning(
"For aesthetic reasons, the files in the PLIST should be sorted",
"alphabetically.");
					} elsif ($last_file_seen eq $text) {
						$line->log_warning("Duplicate filename.");
					}
				}
				$last_file_seen = $text;
			}

			if (index($basename, '${IMAKE_MANNEWSUFFIX}') != -1) {
				warn_about_PLIST_imake_mannewsuffix($line);
			}

			if (substr($dirname, 0, 4) eq "bin/") {
				$line->log_warning("The bin/ directory should not have subdirectories.");

			} elsif ($dirname eq "bin") {

				if (exists($all_files->{"man/man1/${basename}.1"})) {
					# Fine.
				} elsif (exists($all_files->{"man/man6/${basename}.6"})) {
					# Fine.
				} elsif (exists($all_files->{"\${IMAKE_MAN_DIR}/${basename}.\${IMAKE_MANNEWSUFFIX}"})) {
					# Fine.
				} else {
					$opt_warn_extra and $line->log_warning("Manual page missing for bin/${basename}.");
					$opt_warn_extra and $line->explain_warning(
"All programs that can be run directly by the user should have a manual",
"page for quick reference. The programs in the bin/ directory should have",
"corresponding manual pages in section 1 (filename program.1), not in",
"section 8.");
				}

			} elsif (substr($text, 0, 4) eq "doc/") {
				$line->log_error("Documentation must be installed under share/doc, not doc.");

			} elsif (substr($text, 0, 9) eq "etc/rc.d/") {
				$line->log_error("RCD_SCRIPTS must not be registered in the PLIST. Please use the RCD_SCRIPTS framework.");

			} elsif (substr($text, 0, 4) eq "etc/") {
				my $f = "mk/pkginstall/bsd.pkginstall.mk";

				assert(-f "${cwd_pkgsrcdir}/${f}", "${cwd_pkgsrcdir}/${f} is not a regular file.");
				$line->log_error("Configuration files must not be registered in the PLIST. Please use the CONF_FILES framework, which is described in ${f}.");

			} elsif (substr($text, 0, 8) eq "include/" && $text =~ m"^include/.*\.(?:h|hpp)$") {
				# Fine.

			} elsif ($text eq "info/dir") {
				$line->log_error("\"info/dir\" must not be listed. Use install-info to add/remove an entry.");

			} elsif (substr($text, 0, 5) eq "info/" && length($text) > 5) {
				if (defined($pkgctx_vardef) && !exists($pkgctx_vardef->{"INFO_FILES"})) {
					$line->log_warning("Packages that install info files should set INFO_FILES.");
				}

			} elsif (defined($effective_pkgbase) && $text =~ m"^lib/\Q${effective_pkgbase}\E/") {
				# Fine.

			} elsif (substr($text, 0, 11) eq "lib/locale/") {
				$line->log_error("\"lib/locale\" must not be listed. Use \${PKGLOCALEDIR}/locale and set USE_PKGLOCALEDIR instead.");

			} elsif (substr($text, 0, 4) eq "lib/" && $text =~ m"^(lib/(?:.*/)*)([^/]+)\.(so|a|la)$") {
				my ($dir, $lib, $ext) = ($1, $2, $3);

				if ($dir eq "lib/" && $lib !~ m"^lib") {
					$opt_warn_extra and $line->log_warning("Library filename does not start with \"lib\".");
				}
				if ($ext eq "la") {
					if (defined($pkgctx_vardef) && !exists($pkgctx_vardef->{"USE_LIBTOOL"})) {
						$line->log_warning("Packages that install libtool libraries should define USE_LIBTOOL.");
					}
				}

			} elsif (substr($text, 0, 4) eq "man/" && $text =~ m"^man/(cat|man)(\w+)/(.*?)\.(\w+)(\.gz)?$") {
				my ($cat_or_man, $section, $manpage, $ext, $gz) = ($1, $2, $3, $4, $5);

				if ($section !~ m"^[\dln]$") {
					$line->log_warning("Unknown section \"${section}\" for manual page.");
				}

				if ($cat_or_man eq "cat" && !exists($all_files->{"man/man${section}/${manpage}.${section}"})) {
					$line->log_warning("Preformatted manual page without unformatted one.");
				}

				if ($cat_or_man eq "cat") {
					if ($ext ne "0") {
						$line->log_warning("Preformatted manual pages should end in \".0\".");
					}
				} else {
					if ($section ne $ext) {
						$line->log_warning("Mismatch between the section (${section}) and extension (${ext}) of the manual page.");
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
				$line->log_warning("Invalid filename \"${text}\" for preformatted manual page.");

			} elsif (substr($text, 0, 7) eq "man/man") {
				$line->log_warning("Invalid filename \"${text}\" for unformatted manual page.");

			} elsif (substr($text, 0, 5) eq "sbin/") {
				my $binname = substr($text, 5);

				if (!exists($all_files->{"man/man8/${binname}.8"})) {
					$opt_warn_extra and $line->log_warning("Manual page missing for sbin/${binname}.");
					$opt_warn_extra and $line->explain_warning(
"All programs that can be run directly by the user should have a manual",
"page for quick reference. The programs in the sbin/ directory should have",
"corresponding manual pages in section 8 (filename program.8), not in",
"section 1.");
				}

			} elsif (substr($text, 0, 6) eq "share/" && $text =~ m"^share/applications/.*\.desktop$") {
				my $f = "../../sysutils/desktop-file-utils/desktopdb.mk";
				if (defined($pkgctx_included) && !exists($pkgctx_included->{$f})) {
					$line->log_warning("Packages that install a .desktop entry may .include \"$f\".");
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
				$opt_warn_plist_depr and $line->log_warning("Use of \"share/doc/html\" is deprecated. Use \"share/doc/\${PKGBASE}\" instead.");

			} elsif (defined($effective_pkgbase) && $text =~ m"^share/(?:doc/|examples/|)\Q${effective_pkgbase}\E/") {
				# Fine.

			} elsif ($pkgpath ne "graphics/hicolor-icon-theme" && $text =~ m"^share/icons/hicolor/icon-theme\.cache") {
				$line->log_error("Please .include \"../../graphics/hicolor-icon-theme/buildlink3.mk\" and remove this line.");

			} elsif (substr($text, 0, 11) eq "share/info/") {
				$line->log_warning("Info pages should be installed into info/, not share/info/.");
				$line->explain_warning(
"To fix this, you should add INFO_FILES=yes to the package Makefile.");

			} elsif (substr($text, -3) eq ".mo" && $text =~ m"^share/locale/[\w\@_]+/LC_MESSAGES/[^/]+\.mo$") {
				# Fine.

			} elsif (substr($text, 0, 10) eq "share/man/") {
				$line->log_warning("Man pages should be installed into man/, not share/man/.");

			} else {
				$opt_debug_unchecked and $line->log_debug("Unchecked pathname \"${text}\".");
			}

			if ($text =~ /\$\{PKGLOCALEDIR}/ && defined($pkgctx_vardef) && !exists($pkgctx_vardef->{"USE_PKGLOCALEDIR"})) {
				$line->log_warning("PLIST contains \${PKGLOCALEDIR}, but USE_PKGLOCALEDIR was not found.");
			}

			if (index($text, "/CVS/") != -1) {
				$line->log_warning("CVS files should not be in the PLIST.");
			}
			if (substr($text, -5) eq ".orig") {
				$line->log_warning(".orig files should not be in the PLIST.");
			}
			if (substr($text, -14) eq "/perllocal.pod") {
				$line->log_warning("perllocal.pod files should not be in the PLIST.");
				$line->explain_warning(
"This file is handled automatically by the INSTALL/DEINSTALL scripts,",
"since its contents changes frequently.");
			}

			if ($text =~ m"^(.*)(\.a|\.so[0-9.]*)$") {
				my ($basename, $ext) = ($1, $2);

				if (exists($all_files->{"${basename}.la"})) {
					$line->log_warning("Redundant library found. The libtool library is in line " . $all_files->{"${basename}.la"}->lines . ".");
				}
			}

		} elsif ($text =~ m"^\$\{[\w_]+\}$") {
			# A variable on its own line.

		} else {
			$line->log_error("Unknown line type.");
		}
	}
	checklines_trailing_empty_lines($lines);
	autofix($lines);
}

sub checkfile($) {
	my ($fname) = @_;
	my ($st, $basename);

	$opt_debug_trace and log_debug($fname, NO_LINES, "checkfile()");

	$basename = basename($fname);
	if ($basename =~ m"^(?:work.*|.*~|.*\.orig|.*\.rej)$") {
		if ($opt_import) {
			log_error($fname, NO_LINE_NUMBER, "Must be cleaned up before committing the package.");
		}
		return;
	}

	if (!($st = lstat($fname))) {
		log_error($fname, NO_LINE_NUMBER, "$!");
		return;
	}
	if (S_ISDIR($st->mode)) {
		if ($basename eq "files" || $basename eq "patches" || $basename eq "CVS") {
			# Ok
		} elsif ($fname =~ m"(?:^|/)files/[^/]*$") {
			# Ok

		} elsif (!is_emptydir($fname)) {
			log_warning($fname, NO_LINE_NUMBER, "Unknown directory name.");
		}

	} elsif (S_ISLNK($st->mode)) {
		if ($basename !~ m"^work") {
			log_warning($fname, NO_LINE_NUMBER, "Unknown symlink name.");
		}

	} elsif (!S_ISREG($st->mode)) {
		log_error($fname, NO_LINE_NUMBER, "Only files and directories are allowed in pkgsrc.");

	} elsif ($basename eq "ALTERNATIVES") {
		$opt_check_ALTERNATIVES and checkfile_ALTERNATIVES($fname);

	} elsif ($basename eq "buildlink3.mk") {
		$opt_check_bl3 and checkfile_buildlink3_mk($fname);

	} elsif ($basename =~ m"^DESCR") {
		$opt_check_DESCR and checkfile_DESCR($fname);

	} elsif ($basename =~ m"^distinfo") {
		$opt_check_distinfo and checkfile_distinfo($fname);

	} elsif ($basename eq "DEINSTALL" || $basename eq "INSTALL") {
		$opt_check_INSTALL and checkfile_INSTALL($fname);

	} elsif ($basename =~ m"^MESSAGE") {
		$opt_check_MESSAGE and checkfile_MESSAGE($fname);

	} elsif ($basename =~ m"^patch-[-A-Za-z0-9_.~+]*[A-Za-z0-9_]$") {
		$opt_check_patches and checkfile_patch($fname);

	} elsif ($fname =~ m"(?:^|/)patches/manual[^/]*$") {
		$opt_debug_unchecked and log_debug($fname, NO_LINE_NUMBER, "Unchecked file \"${fname}\".");

	} elsif ($fname =~ m"(?:^|/)patches/[^/]*$") {
		log_warning($fname, NO_LINE_NUMBER, "Patch files should be named \"patch-\", followed by letters, '-', '_', '.', and digits only.");

	} elsif ($basename =~ m"^(?:.*\.mk|Makefile.*)$" and not $fname =~ m,files/, and not $fname =~ m,patches/,) {
		$opt_check_mk and checkfile_mk($fname);

	} elsif ($basename =~ m"^PLIST") {
		$opt_check_PLIST and checkfile_PLIST($fname);

	} elsif ($basename eq "TODO" || $basename eq "README") {
		# Ok

	} elsif ($basename =~ m"^CHANGES-.*") {
		load_doc_CHANGES($fname);

	} elsif (!-T $fname) {
		log_warning($fname, NO_LINE_NUMBER, "Unexpectedly found a binary file.");

	} elsif ($fname =~ m"(?:^|/)files/[^/]*$") {
		# Ok
	} else {
		log_warning($fname, NO_LINE_NUMBER, "Unexpected file found.");
		$opt_check_extra and checkfile_extra($fname);
	}
}

sub my_split($$) {
	my ($delimiter, $s) = @_;
	my ($pos, $next, @result);

	$pos = 0;
	for ($pos = 0; $pos != -1; $pos = $next) {
		$next = index($s, $delimiter, $pos);
		push @result, (($next == -1) ? substr($s, $pos) : substr($s, $pos, $next - $pos));
		if ($next != -1) {
			$next += length($delimiter);
		}
	}
	return @result;
}

# Checks that the files in the directory are in sync with CVS's status.
#
sub checkdir_CVS($) {
	my ($fname) = @_;

	my $cvs_entries = load_file("$fname/CVS/Entries");
	my $cvs_entries_log = load_file("$fname/CVS/Entries.Log");
	return unless $cvs_entries;

	foreach my $line (@$cvs_entries) {
		my ($type, $fname, $mtime, $date, $keyword_mode, $tag, $undef) = my_split("/", $line->text);
		next if ($type eq "D" && !defined($fname));
		assert(false, "Unknown line format: " . $line->text)
			unless $type eq "" || $type eq "D";
		assert(false, "Unknown line format: " . $line->text)
			unless defined($tag);
		assert(false, "Unknown line format: " . $line->text)
			unless defined($keyword_mode);
		assert(false, "Unknown line format: " . $line->text)
			if defined($undef);
	}
}

#
# Procedures to check a directory including the files in it.
#

sub checkdir_root() {
	my ($fname) = "${current_dir}/Makefile";
	my ($lines, $prev_subdir, @subdirs);

	$opt_debug_trace and log_debug($fname, NO_LINES, "checkdir_root()");

	if (!($lines = load_lines($fname, true))) {
		log_error($fname, NO_LINE_NUMBER, "Cannot be read.");
		return;
	}

	parselines_mk($lines);
	if (0 <= $#{$lines}) {
		checkline_rcsid_regex($lines->[0], qr"#\s+", "# ");
	}

	foreach my $line (@{$lines}) {
		if ($line->text =~ m"^(#?)SUBDIR\s*\+=(\s*)(\S+)\s*(?:#\s*(.*?)\s*|)$") {
			my ($comment_flag, $indentation, $subdir, $comment) = ($1, $2, $3, $4);

			if ($comment_flag eq "#" && (!defined($comment) || $comment eq "")) {
				$line->log_warning("${subdir} commented out without giving a reason.");
			}

			if ($indentation ne "\t") {
				$line->log_warning("Indentation should be a single tab character.");
			}

			if ($subdir =~ m"\$" || !-f "${current_dir}/${subdir}/Makefile") {
				next;
			}

			if (!defined($prev_subdir) || $subdir gt $prev_subdir) {
				# correctly ordered
			} elsif ($subdir eq $prev_subdir) {
				$line->log_error("${subdir} must only appear once.");
			} elsif ($prev_subdir eq "x11" && $subdir eq "archivers") {
				# ignore that one, since it is documented in the top-level Makefile
			} else {
				$line->log_warning("${subdir} should come before ${prev_subdir}.");
			}

			$prev_subdir = $subdir;

			if ($comment_flag eq "") {
				push(@subdirs, "${current_dir}/${subdir}");
			}
		}
	}

	checklines_mk($lines);

	if ($opt_recursive) {
		$ipc_checking_root_recursively = true;
		push(@todo_items, @subdirs);
	}
}

sub checkdir_category() {
	my $fname = "${current_dir}/Makefile";
	my ($lines, $lineno);

	$opt_debug_trace and log_debug($fname, NO_LINES, "checkdir_category()");

	if (!($lines = load_lines($fname, true))) {
		log_error($fname, NO_LINE_NUMBER, "Cannot be read.");
		return;
	}
	parselines_mk($lines);

	$lineno = 0;

	# The first line must contain the RCS Id
	if ($lineno <= $#{$lines} && checkline_rcsid_regex($lines->[$lineno], qr"#\s+", "# ")) {
		$lineno++;
	}

	# Then, arbitrary comments may follow
	while ($lineno <= $#{$lines} && $lines->[$lineno]->text =~ m"^#") {
		$lineno++;
	}

	# Then we need an empty line
	expect_empty_line($lines, \$lineno);

	# Then comes the COMMENT line
	if ($lineno <= $#{$lines} && $lines->[$lineno]->text =~ m"^COMMENT=\t*(.*)") {
		my ($comment) = ($1);

		checkline_valid_characters_in_variable($lines->[$lineno], qr"[-\040'(),/0-9A-Za-z]");
		$lineno++;
	} else {
		$lines->[$lineno]->log_error("COMMENT= line expected.");
	}

	# Then we need an empty line
	expect_empty_line($lines, \$lineno);

	# And now to the most complicated part of the category Makefiles,
	# the (hopefully) sorted list of SUBDIRs. The first step is to
	# collect the SUBDIRs in the Makefile and in the file system.

	my (@f_subdirs, @m_subdirs);

	@f_subdirs = sort(get_subdirs($current_dir));

	my $prev_subdir = undef;
	while ($lineno <= $#{$lines}) {
		my $line = $lines->[$lineno];

		if ($line->text =~ m"^(#?)SUBDIR\+=(\s*)(\S+)\s*(?:#\s*(.*?)\s*|)$") {
			my ($comment_flag, $indentation, $subdir, $comment) = ($1, $2, $3, $4);

			if ($comment_flag eq "#" && (!defined($comment) || $comment eq "")) {
				$line->log_warning("${subdir} commented out without giving a reason.");
			}

			if ($indentation ne "\t") {
				$line->log_warning("Indentation should be a single tab character.");
			}

			if (defined($prev_subdir) && $subdir eq $prev_subdir) {
				$line->log_error("${subdir} must only appear once.");
			} elsif (defined($prev_subdir) && $subdir lt $prev_subdir) {
				$line->log_warning("${subdir} should come before ${prev_subdir}.");
			} else {
				# correctly ordered
			}

			push(@m_subdirs, [$subdir, $line, $comment_flag ? false : true]);
			$prev_subdir = $subdir;
			$lineno++;

		} else {
			if ($line->text ne "") {
				$line->log_error("SUBDIR+= line or empty line expected.");
			}
			last;
		}
	}

	# To prevent unnecessary warnings about subdirectories that are
	# in one list, but not in the other, we generate the sets of
	# subdirs of each list.
	my (%f_check, %m_check);
	foreach my $f (@f_subdirs) { $f_check{$f} = true; }
	foreach my $m (@m_subdirs) { $m_check{$m->[0]} = true; }

	my ($f_index, $f_atend, $f_neednext, $f_current) = (0, false, true, undef, undef);
	my ($m_index, $m_atend, $m_neednext, $m_current) = (0, false, true, undef, undef);
	my ($line, $m_recurse);
	my (@subdirs);

	while (!($m_atend && $f_atend)) {

		if (!$m_atend && $m_neednext) {
			$m_neednext = false;
			if ($m_index > $#m_subdirs) {
				$m_atend = true;
				$line = $lines->[$lineno];
				next;
			} else {
				$m_current = $m_subdirs[$m_index]->[0];
				$line = $m_subdirs[$m_index]->[1];
				$m_recurse = $m_subdirs[$m_index]->[2];
				$m_index++;
			}
		}

		if (!$f_atend && $f_neednext) {
			$f_neednext = false;
			if ($f_index > $#f_subdirs) {
				$f_atend = true;
				next;
			} else {
				$f_current = $f_subdirs[$f_index++];
			}
		}

		if (!$f_atend && ($m_atend || $f_current lt $m_current)) {
			if (!exists($m_check{$f_current})) {
				$line->log_error("${f_current} exists in the file system, but not in the Makefile.");
				$line->append_before("SUBDIR+=\t${f_current}");
			}
			$f_neednext = true;

		} elsif (!$m_atend && ($f_atend || $m_current lt $f_current)) {
			if (!exists($f_check{$m_current})) {
				$line->log_error("${m_current} exists in the Makefile, but not in the file system.");
				$line->delete();
			}
			$m_neednext = true;

		} else { # $f_current eq $m_current
			$f_neednext = true;
			$m_neednext = true;
			if ($m_recurse) {
				push(@subdirs, "${current_dir}/${m_current}");
			}
		}
	}

	# the wip category Makefile may have its own targets for generating
	# indexes and READMEs. Just skip them.
	if ($is_wip) {
		while ($lineno <= $#{$lines} - 2) {
			$lineno++;
		}
	}

	expect_empty_line($lines, \$lineno);

	# And, last but not least, the .include line
	my $final_line = ".include \"../mk/bsd.pkg.subdir.mk\"";
	expect($lines, \$lineno, qr"\Q$final_line\E")
	|| expect_text($lines, \$lineno, ".include \"../mk/misc/category.mk\"");

	if ($lineno <= $#{$lines}) {
		$lines->[$lineno]->log_error("The file should end here.");
	}

	checklines_mk($lines);

	autofix($lines);

	if ($opt_recursive) {
		unshift(@todo_items, @subdirs);
	}
}

sub checkdir_package() {
	my ($lines, $have_distinfo, $have_patches);

	# Initialize global variables
	$pkgdir = undef;
	$filesdir = "files";
	$patchdir = "patches";
	$distinfo_file = "distinfo";
	$effective_pkgname = undef;
	$effective_pkgbase = undef;
	$effective_pkgversion = undef;
	$effective_pkgname_line = undef;
	$seen_bsd_prefs_mk = false;
	$pkgctx_vardef = {%{get_userdefined_variables()}};
	$pkgctx_varuse = {};
	$pkgctx_bl3 = {};
	$pkgctx_plist_subst_cond = {};
	$pkgctx_included = {};
	$seen_Makefile_common = false;

	# we need to handle the Makefile first to get some variables
	if (!load_package_Makefile("${current_dir}/Makefile", \$lines)) {
		log_error("${current_dir}/Makefile", NO_LINE_NUMBER, "Cannot be read.");
		goto cleanup;
	}

	my @files = glob("${current_dir}/*");
	if ($pkgdir ne ".") {
		push(@files, glob("${current_dir}/${pkgdir}/*"));
	}
	if ($opt_check_extra) {
		push(@files, glob("${current_dir}/${filesdir}/*"));
	}
	push(@files, glob("${current_dir}/${patchdir}/*"));
	if ($distinfo_file !~ m"^(?:\./)?distinfo$") {
		push(@files, "${current_dir}/${distinfo_file}");
	}
	$have_distinfo = false;
	$have_patches = false;

	# Determine the used variables before checking any of the
	# Makefile fragments.
	foreach my $fname (@files) {
		if (($fname =~ m"^((?:.*/)?Makefile\..*|.*\.mk)$")
		&& (not $fname =~ m"patch-")
		&& (not $fname =~ m"${pkgdir}/")
		&& (not $fname =~ m"${filesdir}/")
		&& (defined(my $lines = load_lines($fname, true)))) {
			parselines_mk($lines);
			determine_used_variables($lines);
		}
	}

	foreach my $fname (@files) {
		if ($fname eq "${current_dir}/Makefile") {
			$opt_check_Makefile and checkfile_package_Makefile($fname, $lines);
		} else {
			checkfile($fname);
		}
		if ($fname =~ m"/patches/patch-*$") {
			$have_patches = true;
		} elsif ($fname =~ m"/distinfo$") {
			$have_distinfo = true;
		}
	}

	if ($opt_check_distinfo && $opt_check_patches) {
		if ($have_patches && ! $have_distinfo) {
			log_warning("${current_dir}/$distinfo_file", NO_LINE_NUMBER, "File not found. Please run '".conf_make." makepatchsum'.");
		}
	}

	if (!is_emptydir("${current_dir}/scripts")) {
		log_warning("${current_dir}/scripts", NO_LINE_NUMBER, "This directory and its contents are deprecated! Please call the script(s) explicitly from the corresponding target(s) in the pkg's Makefile.");
	}

cleanup:
	# Clean up global variables.
	$pkgdir = undef;
	$filesdir = undef;
	$patchdir = undef;
	$distinfo_file = undef;
	$effective_pkgname = undef;
	$effective_pkgbase = undef;
	$effective_pkgversion = undef;
	$effective_pkgname_line = undef;
	$seen_bsd_prefs_mk = undef;
	$pkgctx_vardef = undef;
	$pkgctx_varuse = undef;
	$pkgctx_bl3 = undef;
	$pkgctx_plist_subst_cond = undef;
	$pkgctx_included = undef;
	$seen_Makefile_common = undef;
}
