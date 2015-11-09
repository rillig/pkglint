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
