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
