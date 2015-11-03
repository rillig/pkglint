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
