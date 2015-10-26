my $get_doc_CHANGES_docs = undef; # [ $fname, undef or { pkgpath -> @changes } ]
sub get_doc_CHANGES($) {
	my ($pkgpath) = @_;

	$opt_debug_trace and log_debug(NO_FILE, NO_LINES, "get_doc_CHANGES(\"$pkgpath\")");

	# Make a reversed list of all the CHANGES-* files, but don't load
	# them yet.
	if (!defined($get_doc_CHANGES_docs)) {
		opendir(DIR, "${cwd_pkgsrcdir}/doc") or die;
		my @files = readdir(DIR);
		closedir(DIR) or die;
		foreach my $file (reverse sort @files) {
			if ($file =~ m"^CHANGES-(\d+)$" && (0 + $1 >= 2008)) {
				push(@$get_doc_CHANGES_docs, [ $file, undef ]);
			}
		}
		$opt_debug_misc and log_debug(NO_FILE, NO_LINES, "Found " . (scalar @$get_doc_CHANGES_docs) . " changes files.");
	}

	# Scan the *-CHANGES files in reverse order until some action
	# matches the package directory.
	my @result = ();
	foreach my $doc (@$get_doc_CHANGES_docs) {
		if (!defined($doc->[1])) {
			$opt_debug_misc and log_debug(NO_FILE, NO_LINES, "loading $doc->[0]");
			$doc->[1] = load_doc_CHANGES("${cwd_pkgsrcdir}/doc/$doc->[0]");
		}

		foreach my $change (@{$doc->[1]->{$pkgpath}}) {
			next unless $pkgpath eq $change->pkgpath;
			push(@result, $change);
		}
		if (@result != 0) {
			return @result;
		}
	}
	return ();
}


# Load all variables from mk/defaults/mk.conf. Since pkglint does not
# load the infrastructure files during normal operation, these
# definitions have to be added explicitly.
sub load_userdefined_variables() {
	my $fname = "${cwd_pkgsrcdir}/mk/defaults/mk.conf";
	my $vars = {};

	my $lines = load_existing_lines($fname, true);
	foreach my $line (@{$lines}) {
		if ($line->text =~ regex_varassign) {
			my ($varname, $op, $value, $comment) = ($1, $2, $3, $4);

			$vars->{$varname} = $line;
		}
	}

	return $vars;
}

sub get_userdefined_variables() {
	state $result = load_userdefined_variables();
	return $result;
}

sub match_all($$);	# needed by load_shared_dirs()

my $load_shared_dirs_dir_to_varname = undef;
my $load_shared_dirs_varname_to_dirs = undef;
my $load_shared_dirs_dir_to_id = undef;
sub load_shared_dirs() {
	return if defined($load_shared_dirs_dir_to_varname);

	$opt_debug_trace and log_debug(NO_FILE, NO_LINES, "load_shared_dirs()");

	my $dir_to_varname = {};
	my $varname_to_dirs = {};
	my $dir_to_id = {};

	foreach my $pkg (qw(
		misc/gnome-dirs misc/gnome1-dirs misc/gnome2-dirs
		misc/theme-dirs
		misc/xdg-dirs misc/xdg-x11-dirs
		print/texmf-dirs)) {

		$opt_debug_trace and log_debug(NO_FILE, NO_LINES, "pkg=$pkg");
		my $dirs_mk = load_existing_lines("$cwd_pkgsrcdir/$pkg/dirs.mk", true);
		foreach my $line (@$dirs_mk) {
			parseline_mk($line);
			if ($line->has("is_varassign")) {
				my $varname = $line->get("varname");
				my $value = $line->get("value");

				if ($varname =~ m"^[A-Z]\w*_DIRS$" && $value ne "") {
					if (exists($dir_to_varname->{$value})) {
						# FIXME: misc/xdg-x11-dirs and misc/xdg-dirs conflict.
						#$line->log_warning("Duplicate directory, also appears in " . $dir_to_varname->{$value} . ".");
					} else {
						$dir_to_varname->{$value} = $varname;
					}
				}

			} elsif ($line->has("is_cond") && $line->get("directive") eq "for") {
				my $args = $line->get("args");
				while ($args =~ /\$\{(\w+_DIRS)\}/gc) {
					push(@{$varname_to_dirs->{$1}}, $pkg);
				}
			}
		}

		my $makefile = load_existing_lines("$cwd_pkgsrcdir/$pkg/Makefile", true);
		foreach my $line (@$makefile) {
			my $pkgname = undef;

			parseline_mk($line);
			if ($line->has("is_varassign") && $line->get("varname") eq "DISTNAME") {
				if ($line->get("value") =~ m"^(.*)-dirs-(.*)$") {
					$dir_to_id->{$pkg} = "$1-$2";
				} else {
					assert(false, "$pkg/Makefile does not define a proper DISTNAME.");
				}
			}
		}
	}
	$load_shared_dirs_dir_to_varname = $dir_to_varname;
	$load_shared_dirs_varname_to_dirs = $varname_to_dirs;
	$load_shared_dirs_dir_to_id = $dir_to_id;
}

#
# Miscellaneous functions
#

sub resolve_relative_path($$) {
	my ($relpath, $adjust_depth) = @_;

	my $arg = $relpath;
	$relpath =~ s,\$\{PKGSRCDIR\},$cur_pkgsrcdir,;
	$relpath =~ s,\$\{\.CURDIR\},.,;
	$relpath =~ s,\$\{\.PARSEDIR\},.,;
	$relpath =~ s,\$\{LUA_PKGSRCDIR\},../../lang/lua52,;
	$relpath =~ s,\$\{PHPPKGSRCDIR\},../../lang/php54,;
	$relpath =~ s,\$\{SUSE_DIR_PREFIX\},suse100,;
	$relpath =~ s,\$\{PYPKGSRCDIR\},../../lang/python27,;
	$relpath =~ s,\$\{FILESDIR\},$filesdir, if defined($filesdir);
	if ($adjust_depth && $relpath =~ m"^\.\./\.\./([^.].*)$") {
		$relpath = "${cur_pkgsrcdir}/$1";
	}
	if (defined($pkgdir)) {
		$relpath =~ s,\$\{PKGDIR\},$pkgdir,g;
	}

	$opt_debug_misc and log_debug(NO_FILE, NO_LINES, "resolve_relative_path: $arg => $relpath");
	return $relpath;
}

# Takes two pathnames and returns the relative pathname to get from
# the first to the second.
sub relative_path($$) {
	my ($from, $to) = @_;

	my $abs_from = Cwd::abs_path($from);
	my $abs_to = Cwd::abs_path($to);
	if ($abs_to =~ m"^\Q$abs_from/\E(.*)$") { #"
		return $1;
	} elsif ($abs_to eq $abs_from) {
		return ".";
	} else {
		assert(false, "$abs_from is not a prefix of $abs_to");
	}
}

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

sub determine_used_variables($) {
	my ($lines) = @_;
	my ($rest);

	foreach my $line (@{$lines}) {
		$rest = $line->text;
		while ($rest =~ s/(?:\$\{|\$\(|defined\(|empty\()([0-9+.A-Z_a-z]+)[:})]//) {
			my ($varname) = ($1);
			use_var($line, $varname);
			$opt_debug_unused and $line->log_debug("Variable ${varname} is used.");
		}
	}
}

sub extract_used_variables($$) {
	my ($line, $text) = @_;
	my ($rest, $result);

	$rest = $text;
	$result = [];
	while ($rest =~ s/^(?:[^\$]+|\$[\$*<>?\@]|\$\{([.0-9A-Z_a-z]+)(?::(?:[^\${}]|\$[^{])+)?\})//) {
		my ($varname) = ($1);

		if (defined($varname)) {
			push(@{$result}, $varname);
		}
	}

	if ($rest ne "") {
		$opt_debug_misc and $line->log_warning("Could not extract variables: ${rest}");
	}

	return $result;
}

sub get_nbpart() {
	my $line = $pkgctx_vardef->{"PKGREVISION"};
	return "" unless defined($line);
	my $pkgrevision = $line->get("value");
	return "" unless $pkgrevision =~ m"^\d+$";
	return "" unless $pkgrevision + 0 != 0;
	return "nb$pkgrevision";
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

# Returns an object of type Pkglint::Type that represents the type of
# the variable (maybe guessed based on the variable name), or undef if
# the type cannot even be guessed.
#
sub get_variable_type($$) {
	my ($line, $varname) = @_;
	my ($type);

	assert(defined($varname), "The varname parameter must be defined.");

	if (exists(get_vartypes_map()->{$varname})) {
		return get_vartypes_map()->{$varname};
	}

	my $varcanon = varname_canon($varname);
	if (exists(get_vartypes_map()->{$varcanon})) {
		return get_vartypes_map()->{$varcanon};
	}

	if (exists(get_varname_to_toolname()->{$varname})) {
		return PkgLint::Type->new(LK_NONE, "ShellCommand", [[ qr".*", "u" ]], NOT_GUESSED);
	}

	if ($varname =~ m"^TOOLS_(.*)" && exists(get_varname_to_toolname()->{$1})) {
		return PkgLint::Type->new(LK_NONE, "Pathname", [[ qr".*", "u" ]], NOT_GUESSED);
	}

	use constant allow_all => [[ qr".*", "adpsu" ]];
	use constant allow_runtime => [[ qr".*", "adsu" ]];

	# Guess the datatype of the variable based on
	# naming conventions.
	$type =	  ($varname =~ m"DIRS$") ? PkgLint::Type->new(LK_EXTERNAL, "Pathmask", allow_runtime, GUESSED)
		: ($varname =~ m"(?:DIR|_HOME)$") ? PkgLint::Type->new(LK_NONE, "Pathname", allow_runtime, GUESSED)
		: ($varname =~ m"FILES$") ? PkgLint::Type->new(LK_EXTERNAL, "Pathmask", allow_runtime, GUESSED)
		: ($varname =~ m"FILE$") ? PkgLint::Type->new(LK_NONE, "Pathname", allow_runtime, GUESSED)
		: ($varname =~ m"PATH$") ? PkgLint::Type->new(LK_NONE, "Pathlist", allow_runtime, GUESSED)
		: ($varname =~ m"PATHS$") ? PkgLint::Type->new(LK_EXTERNAL, "Pathname", allow_runtime, GUESSED)
		: ($varname =~ m"_USER$") ? PkgLint::Type->new(LK_NONE, "UserGroupName", allow_all, GUESSED)
		: ($varname =~ m"_GROUP$") ? PkgLint::Type->new(LK_NONE, "UserGroupName", allow_all, GUESSED)
		: ($varname =~ m"_ENV$") ? PkgLint::Type->new(LK_EXTERNAL, "ShellWord", allow_runtime, GUESSED)
		: ($varname =~ m"_CMD$") ? PkgLint::Type->new(LK_NONE, "ShellCommand", allow_runtime, GUESSED)
		: ($varname =~ m"_ARGS$") ? PkgLint::Type->new(LK_EXTERNAL, "ShellWord", allow_runtime, GUESSED)
		: ($varname =~ m"_(?:C|CPP|CXX|LD|)FLAGS$") ? PkgLint::Type->new(LK_EXTERNAL, "ShellWord", allow_runtime, GUESSED)
		: ($varname =~ m"_MK$") ? PkgLint::Type->new(LK_NONE, "Unchecked", allow_all, GUESSED)
		: ($varname =~ m"^PLIST.") ? PkgLint::Type->new(LK_NONE, "Yes", allow_all, GUESSED)
		: undef;

	if (defined($type)) {
		$opt_debug_vartypes and $line->log_debug("The guessed type of ${varname} is \"" . $type->to_string . "\".");
		return $type;
	}

	$opt_debug_vartypes and $line->log_debug("No type definition found for ${varcanon}.");
	return undef;
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

# This function returns whether a variable needs the :Q operator in a
# certain context. There are four possible outcomes:
#
# false:	The variable should not be quoted.
# true:		The variable should be quoted.
# doesnt_matter:
#		Since the values of the variable usually don't contain
#		special characters, it does not matter whether the
#		variable is quoted or not.
# dont_know:	pkglint cannot say whether the variable should be quoted
#		or not, most likely because type information is missing.
#
sub variable_needs_quoting($$$) {
	my ($line, $varname, $context) = @_;
	my $type = get_variable_type($line, $varname);
	my ($want_list, $have_list);

	$opt_debug_trace and $line->log_debug("variable_needs_quoting($varname, " . $context->to_string() . ")");

	use constant safe_types => array_to_hash(qw(
		DistSuffix
		FileMode Filename
		Identifier
		Option
		Pathname PkgName PkgOptionsVar PkgRevision
		RelativePkgDir RelativePkgPath
		UserGroupName
		Varname Version
		WrkdirSubdirectory
	));

	if (!defined($type) || !defined($context->type)) {
		return dont_know;
	}

	# Variables of certain predefined types, as well as all
	# enumerations, are expected to not require the :Q operator.
	if (ref($type->basic_type) eq "HASH" || exists(safe_types->{$type->basic_type})) {
		if ($type->kind_of_list == LK_NONE) {
			return doesnt_matter;

		} elsif ($type->kind_of_list == LK_EXTERNAL && $context->extent != VUC_EXTENT_WORD_PART) {
			return false;
		}
	}

	# In .for loops, the :Q operator is always misplaced, since
	# the items are broken up at white-space, not as shell words
	# like in all other parts of make(1).
	if ($context->shellword == VUC_SHELLWORD_FOR) {
		return false;
	}

	# Determine whether the context expects a list of shell words or
	# not.
	$want_list = $context->type->is_practically_a_list() && ($context->shellword == VUC_SHELLWORD_BACKT || $context->extent != VUC_EXTENT_WORD_PART);
	$have_list = $type->is_practically_a_list();

	$opt_debug_quoting and $line->log_debug("[variable_needs_quoting]"
		. " varname=$varname"
		. " context=" . $context->to_string()
		. " type=" . $type->to_string()
		. " want_list=" . ($want_list ? "yes" : "no")
		. " have_list=" . ($have_list ? "yes" : "no")
		. ".");

	# A shell word may appear as part of a shell word, for example
	# COMPILER_RPATH_FLAG.
	if ($context->extent == VUC_EXTENT_WORD_PART && $context->shellword == VUC_SHELLWORD_PLAIN) {
		if ($type->kind_of_list == LK_NONE && $type->basic_type eq "ShellWord") {
			return false;
		}
	}

	# Assume that the tool definitions don't include very special
	# characters, so they can safely be used inside any quotes.
	if (exists(get_varname_to_toolname()->{$varname})) {
		my $sw = $context->shellword;

		if ($sw == VUC_SHELLWORD_PLAIN && $context->extent != VUC_EXTENT_WORD_PART) {
			return false;

		} elsif ($sw == VUC_SHELLWORD_BACKT) {
			return false;

		} elsif ($sw == VUC_SHELLWORD_DQUOT || $sw == VUC_SHELLWORD_SQUOT) {
			return doesnt_matter;

		} else {
			# Let the other rules decide.
		}
	}

	# Variables that appear as parts of shell words generally need
	# to be quoted. An exception is in the case of backticks,
	# because the whole backticks expression is parsed as a single
	# shell word by pkglint.
	#
	# XXX: When the shell word parser gets rewritten the next time,
	# this test can be refined.
	if ($context->extent == VUC_EXTENT_WORD_PART && $context->shellword != VUC_SHELLWORD_BACKT) {
		return true;
	}

	# Assigning lists to lists does not require any quoting, though
	# there may be cases like "CONFIGURE_ARGS+= -libs ${LDFLAGS:Q}"
	# where quoting is necessary. So let's hope that no developer
	# ever makes the mistake of using :Q when appending a list to
	# a list.
	if ($want_list && $have_list) {
		return doesnt_matter;
	}

	# Appending elements to a list requires quoting, as well as
	# assigning a list value to a non-list variable.
	if ($want_list != $have_list) {
		return true;
	}

	$opt_debug_quoting and $line->log_debug("Don't know whether :Q is needed for ${varname}.");
	return dont_know;
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

# This procedure fills in the extra fields of a line, depending on the
# line type. These fields can later be queried without having to parse
# them again and again.
#
sub parseline_mk($) {
	my ($line) = @_;
	my $text = $line->text;

	if ($text =~ regex_varassign) {
		my ($varname, $op, $value, $comment) = ($1, $2, $3, $4);

		# In variable assignments, a '#' character is preceded
		# by a backslash. In shell commands, it is interpreted
		# literally.
		$value =~ s/\\\#/\#/g;

		$line->set("is_varassign", true);
		$line->set("varname", $varname);
		$line->set("varcanon", varname_canon($varname));
		my $varparam = varname_param($varname);
		defined($varparam) and $line->set("varparam", $varparam);
		$line->set("op", $op);
		$line->set("value", $value);
		defined($comment) and $line->set("comment", $comment);

	} elsif ($text =~ regex_mk_shellcmd) {
		my ($shellcmd) = ($1);

		# Shell command lines cannot have embedded comments.
		$line->set("is_shellcmd", true);
		$line->set("shellcmd", $shellcmd);

		my ($shellwords, $rest) = match_all($shellcmd, $regex_shellword);
		$line->set("shellwords", $shellwords);
		if ($rest !~ m"^\s*$") {
			$line->set("shellwords_rest", $rest);
		}

	} elsif ($text =~ regex_mk_comment) {
		my ($comment) = ($1);

		$line->set("is_comment", true);
		$line->set("comment", $comment);

	} elsif ($text =~ m"^\s*$") {

		$line->set("is_empty", true);

	} elsif ($text =~ regex_mk_cond) {
		my ($indent, $directive, $args, $comment) = ($1, $2, $3, $4);

		$line->set("is_cond", true);
		$line->set("indent", $indent);
		$line->set("directive", $directive);
		defined($args) and $line->set("args", $args);
		defined($comment) and $line->set("comment", $comment);

	} elsif ($text =~ regex_mk_include) {
		my (undef, $includefile, $comment) = ($1, $2, $3);

		$line->set("is_include", true);
		$line->set("includefile", $includefile);
		defined($comment) and $line->set("comment", $comment);

	} elsif ($text =~ regex_mk_sysinclude) {
		my ($includefile, $comment) = ($1, $2);

		$line->set("is_sysinclude", true);
		$line->set("includefile", $includefile);
		defined($comment) and $line->set("comment", $comment);

	} elsif ($text =~ regex_mk_dependency) {
		my ($targets, $whitespace, $sources, $comment) = ($1, $2, $3, $4);

		$line->set("is_dependency", true);
		$line->set("targets", $targets);
		$line->set("sources", $sources);
		$line->log_warning("Space before colon in dependency line: " . $line->to_string()) if ($whitespace);
		defined($comment) and $line->set("comment", $comment);

	} elsif ($text =~ regex_rcs_conflict) {
		# This line is useless

	} else {
		$line->log_fatal("Unknown line format: " . $line->to_string());
	}
}

sub parselines_mk($) {
	my ($lines) = @_;

	foreach my $line (@{$lines}) {
		parseline_mk($line);
	}
}

#
# Loading package-specific data from files.
#

sub readmakefile($$$$);
sub readmakefile($$$$) {
	my ($fname, $main_lines, $all_lines, $seen_Makefile_include) = @_;
	my ($includefile, $dirname, $lines, $is_main_Makefile);

	$lines = load_lines($fname, true);
	if (!$lines) {
		return false;
	}
	parselines_mk($lines);

	$is_main_Makefile = (@{$main_lines} == 0);

	foreach my $line (@{$lines}) {
		my $text = $line->text;

		if ($is_main_Makefile) {
			push(@{$main_lines}, $line);
		}
		push(@{$all_lines}, $line);

		# try to get any included file
		my $is_include_line = false;
		if ($text =~ m"^\.\s*include\s+\"(.*)\"$") {
			$includefile = resolve_relative_path($1, true);
			if ($includefile =~ regex_unresolved) {
				if ($fname !~ m"/mk/") {
					$line->log_note("Skipping include file \"${includefile}\". This may result in false warnings.");
				}

			} else {
				$is_include_line = true;
			}
		}

		if ($is_include_line) {
			if ($fname !~ m"buildlink3\.mk$" && $includefile =~ m"^\.\./\.\./(.*)/buildlink3\.mk$") {
				my ($bl3_file) = ($1);

				$pkgctx_bl3->{$bl3_file} = $line;
				$opt_debug_misc and $line->log_debug("Buildlink3 file in package: ${bl3_file}");
			}
		}

		if ($is_include_line && !exists($seen_Makefile_include->{$includefile})) {
			$seen_Makefile_include->{$includefile} = true;

			if ($includefile =~ m"^\.\./[^./][^/]*/[^/]+") {
				$line->log_warning("Relative directories should look like \"../../category/package\", not \"../package\".");
				$line->explain_warning(expl_relative_dirs);
			}
			if ($includefile =~ m"(?:^|/)Makefile.common$"
				|| ($includefile =~ m"^(?:\.\./(\.\./[^/]+/)?[^/]+/)?([^/]+)$"
				&& (!defined($1) || $1 ne "../mk")
				&& $2 ne "buildlink3.mk"
				&& $2 ne "options.mk")) {
				$opt_debug_include and $line->log_debug("including ${includefile} sets seen_Makefile_common.");
				$seen_Makefile_common = true;
			}
			if ($includefile =~ m"/mk/") {
				# skip these files

			} else {
				$dirname = dirname($fname);
				# Only look in the directory relative to the
				# current file and in the current working directory.
				# We don't have an include dir list, like make(1) does.
				if (!-f "$dirname/$includefile") {
					$dirname = $current_dir;
				}
				if (!-f "$dirname/$includefile") {
					$line->log_error("Cannot read $dirname/$includefile.");
				} else {
					$opt_debug_include and $line->log_debug("Including \"$dirname/$includefile\".");
					my $last_lineno = $#{$all_lines};
					readmakefile("$dirname/$includefile", $main_lines, $all_lines, $seen_Makefile_include) or return false;

					# Check that there is a comment in each
					# Makefile.common that says which files
					# include it.
					if ($includefile =~ m"/Makefile\.common$") {
						my @mc_lines = @{$all_lines}[$last_lineno+1 .. $#{$all_lines}];
						my $expected = "# used by " . relative_path($cwd_pkgsrcdir, $fname);

						if (!(grep { $_->text eq $expected } @mc_lines)) {
							my $lineno = 0;
							while ($lineno < $#mc_lines && $mc_lines[$lineno]->has("is_comment")) {
								$lineno++;
							}
							my $iline = $mc_lines[$lineno];
							$iline->log_warning("Please add a line \"$expected\" here.");
							$iline->explain_warning(
"Since Makefile.common files usually don't have any comments and",
"therefore not a clearly defined interface, they should at least contain",
"references to all files that include them, so that it is easier to see",
"what effects future changes may have.",
"",
"If there are more than five packages that use a Makefile.common,",
"you should think about giving it a proper name (maybe plugin.mk) and",
"documenting its interface.");
							$iline->append_before($expected);
							autofix(\@mc_lines);
						}
					}
				}
			}

		} elsif ($line->has("is_varassign")) {
			my ($varname, $op, $value) = ($line->get("varname"), $line->get("op"), $line->get("value"));

			# Record all variables that are defined in these lines, so that they
			# are not reported as "used but not defined".
			if ($op ne "?=" || !exists($pkgctx_vardef->{$varname})) {
				$opt_debug_misc and $line->log_debug("varassign(${varname}, ${op}, ${value})");
				$pkgctx_vardef->{$varname} = $line;
			}
		}
	}

	return true;
}
