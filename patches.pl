sub checkfile_patch($) {

	# [ regex, to state, action ]
	my $transitions = {
		PST_START() =>
		[   [re_patch_rcsid, PST_CENTER, sub() {
			checkline_rcsid($line, "");
		}], [undef, PST_CENTER, sub() {
			checkline_rcsid($line, "");
		}]],
		PST_CENTER() =>
		[   [re_patch_empty, PST_TEXT, sub() {
			#
		}], [re_patch_cfd, PST_CFA, sub() {
			if ($seen_comment) {
				$opt_warn_space and $line->log_note("Empty line expected.");
			} else {
				$line->log_error("Comment expected.");
				$line->explain_error(@comment_explanation);
			}
			$line->log_warning("Please use unified diffs (diff -u) for patches.");
		}], [re_patch_ufd, PST_UFA, sub() {
			if ($seen_comment) {
				$opt_warn_space and $line->log_note("Empty line expected.");
			} else {
				$line->log_error("Comment expected.");
				$line->explain_error(@comment_explanation);
			}
		}], [undef, PST_TEXT, sub() {
			$opt_warn_space and $line->log_note("Empty line expected.");
		}]],
		PST_TEXT() =>
		[   [re_patch_cfd, PST_CFA, sub() {
			if (!$seen_comment) {
				$line->log_error("Comment expected.");
				$line->explain_error(@comment_explanation);
			}
			$line->log_warning("Please use unified diffs (diff -u) for patches.");
		}], [re_patch_ufd, PST_UFA, sub() {
			if (!$seen_comment) {
				$line->log_error("Comment expected.");
				$line->explain_error(@comment_explanation);
			}
		}], [re_patch_text, PST_TEXT, sub() {
			$seen_comment = true;
		}], [re_patch_empty, PST_TEXT, sub() {
			#
		}], [undef, PST_TEXT, sub() {
			#
		}]],
		PST_CFA() =>
		[   [re_patch_cfa, PST_CH, sub() {
			$current_fname = $m->text(1);
			$current_ftype = get_filetype($line, $current_fname);
			$opt_debug_patches and $line->log_debug("fname=$current_fname ftype=$current_ftype");
			$patched_files++;
			$hunks = 0;
		}]],
		PST_CH() =>
		[   [re_patch_ch, PST_CHD, sub() {
			$hunks++;
		}]],
		PST_CHD() =>
		[   [re_patch_chd, PST_CLD0, sub() {
			$dellines = ($m->has(2))
				? (1 + $m->text(2) - $m->text(1))
				: ($m->text(1));
		}]],
		PST_CLD0() =>
		[   [re_patch_clc, PST_CLD, sub() {
			$check_hunk_line->(1, 0, PST_CLD0);
		}], [re_patch_cld, PST_CLD, sub() {
			$check_hunk_line->(1, 0, PST_CLD0);
		}], [re_patch_clm, PST_CLD, sub() {
			$check_hunk_line->(1, 0, PST_CLD0);
		}], [re_patch_cha, PST_CLA0, sub() {
			$dellines = undef;
			$addlines = ($m->has(2))
				? (1 + $m->text(2) - $m->text(1))
				: ($m->text(1));
		}]],
		PST_CLD() =>
		[   [re_patch_clc, PST_CLD, sub() {
			$check_hunk_line->(1, 0, PST_CLD0);
		}], [re_patch_cld, PST_CLD, sub() {
			$check_hunk_line->(1, 0, PST_CLD0);
		}], [re_patch_clm, PST_CLD, sub() {
			$check_hunk_line->(1, 0, PST_CLD0);
		}], [undef, PST_CLD0, sub() {
			if ($dellines != 0) {
				$line->log_warning("Invalid number of deleted lines (${dellines} missing).");
			}
		}]],
		PST_CLA0() =>
		[   [re_patch_clc, PST_CLA, sub() {
			$check_hunk_line->(0, 1, PST_CH);
		}], [re_patch_clm, PST_CLA, sub() {
			$check_hunk_line->(0, 1, PST_CH);
		}], [re_patch_cla, PST_CLA, sub() {
			$check_hunk_line->(0, 1, PST_CH);
		}], [undef, PST_CH, sub() {
			#
		}]],
		PST_CLA() =>
		[   [re_patch_clc, PST_CLA, sub() {
			$check_hunk_line->(0, 1, PST_CH);
		}], [re_patch_clm, PST_CLA, sub() {
			$check_hunk_line->(0, 1, PST_CH);
		}], [re_patch_cla, PST_CLA, sub() {
			$check_hunk_line->(0, 1, PST_CH);
		}], [undef, PST_CLA0, sub() {
			if ($addlines != 0) {
				$line->log_warning("Invalid number of added lines (${addlines} missing).");
			}
		}]],
		PST_CH() =>
		[   [undef, PST_TEXT, sub() {
			#
		}]],
		PST_UFA() =>
		[   [re_patch_ufa, PST_UH, sub() {
			$current_fname = $m->text(1);
			$current_ftype = get_filetype($line, $current_fname);
			$opt_debug_patches and $line->log_debug("fname=$current_fname ftype=$current_ftype");
			$patched_files++;
			$hunks = 0;
		}]],
		PST_UH() =>
		[   [re_patch_uh, PST_UL, sub() {
			$dellines = ($m->has(1) ? $m->text(2) : 1);
			$addlines = ($m->has(3) ? $m->text(4) : 1);
			$check_text->($line->text);
			if ($line->text =~ m"\r$") {
				$line->log_error("The hunk header must not end with a CR character.");
				$line->explain_error(
"The MacOS X patch utility cannot handle these.");
			}
			$hunks++;
			$context_scanning_leading = (($m->has(1) && $m->text(1) ne "1") ? true : undef);
			$leading_context_lines = 0;
			$trailing_context_lines = 0;
		}], [undef, PST_TEXT, sub() {
			($hunks != 0) || $line->log_warning("No hunks for file ${current_fname}.");
		}]],
		PST_UL() =>
		[   [re_patch_uld, PST_UL, sub() {
			$check_hunk_line->(1, 0, PST_UH);
		}], [re_patch_ula, PST_UL, sub() {
			$check_hunk_line->(0, 1, PST_UH);
		}], [re_patch_ulc, PST_UL, sub() {
			$check_hunk_line->(1, 1, PST_UH);
		}], [re_patch_ulnonl, PST_UL, sub() {
			#
		}], [re_patch_empty, PST_UL, sub() {
			$opt_warn_space and $line->log_note("Leading white-space missing in hunk.");
			$check_hunk_line->(1, 1, PST_UH);
		}], [undef, PST_UH, sub() {
			if ($dellines != 0 || $addlines != 0) {
				$line->log_warning("Unexpected end of hunk (-${dellines},+${addlines} expected).");
			}
		}]]};

	$state = PST_START;
	$dellines = undef;
	$addlines = undef;
	$patched_files = 0;
	$seen_comment = false;
	$current_fname = undef;
	$current_ftype = undef;
	$hunks = undef;

	for (my $lineno = 0; $lineno <= $#{$lines}; ) {
		$line = $lines->[$lineno];
		my $text = $line->text;

		$opt_debug_patches and $line->log_debug("[${state} ${patched_files}/".($hunks||0)."/-".($dellines||0)."+".($addlines||0)."] $text");

		my $found = false;
		foreach my $t (@{$transitions->{$state}}) {
				if (!defined($t->[0])) {
					$m = undef;
				} elsif ($text =~ $t->[0]) {
					$opt_debug_patches and $line->log_debug($t->[0]);
					$m = PkgLint::SimpleMatch->new($text, \@-, \@+);
				} else {
					next;
				}
				$redostate = undef;
				$nextstate = $t->[1];
				$t->[2]->();
				if (defined($redostate)) {
					$state = $redostate;
				} else {
					$state = $nextstate;
					if (defined($t->[0])) {
						$lineno++;
					}
				}
				$found = true;
				last;
		}

		if (!$found) {
			$line->log_error("Parse error: state=${state}");
			$state = PST_TEXT;
			$lineno++;
		}
	}

	while ($state != PST_TEXT) {
		$opt_debug_patches and log_debug($fname, "EOF", "[${state} ${patched_files}/".($hunks||0)."/-".($dellines||0)."+".($addlines||0)."]");

		my $found = false;
		foreach my $t (@{$transitions->{$state}}) {
			if (!defined($t->[0])) {
				my $newstate;

				$m = undef;
				$redostate = undef;
				$nextstate = $t->[1];
				$t->[2]->();
				$newstate = (defined($redostate)) ? $redostate : $nextstate;
				if ($newstate == $state) {
					log_fatal($fname, "EOF", "Internal error in the patch transition table.");
				}
				$state = $newstate;
				$found = true;
				last;
			}
		}

		if (!$found) {
			log_error($fname, "EOF", "Parse error: state=${state}");
			$state = PST_TEXT;
		}
	}

	if ($patched_files > 1) {
		log_warning($fname, NO_LINE_NUMBER, "Contains patches for $patched_files files, should be only one.");

	} elsif ($patched_files == 0) {
		log_error($fname, NO_LINE_NUMBER, "Contains no patch.");
	}

	checklines_trailing_empty_lines($lines);
}
