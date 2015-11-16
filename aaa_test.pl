sub test_pkglint_main {
	my $unit = \&pkglint::main;

	@ARGV = ('-h');
	test_unit($unit, undef, 0, '^usage: pkglint ', '^$');

	@ARGV = ('..');
	test_unit($unit, undef, 0, '^looks fine', '^$');

	@ARGV = ('.');
	test_unit($unit, undef, 1, '^ERROR:.+how to check', '^$');

	@ARGV = ();
	test_unit($unit, undef, 1, '^ERROR:.+how to check', '^$');

	@ARGV = ('/does/not/exist');
	test_unit($unit, undef, 1, '^ERROR:.+not exist', '^$');

	@ARGV = ($ENV{HOME});
	test_unit($unit, undef, 1, '^ERROR:.+outside a pkgsrc', '^$');
}

sub test_lint_some_reference_packages {
	my %reference_packages = (
		'devel/syncdir' => {
			stdout_re => <<EOT,
^ERROR: .*Makefile: Each package must define its LICENSE\.
ERROR: .*patches/patch-aa:[0-9]+: Comment expected\.
2 errors and 0 warnings found\..*\$
EOT
			stderr_re => undef,
			exitcode => 1,
		},
		'mail/qmail' => {
			stdout_re => <<EOT,
^WARN: .*Makefile:[0-9]+: USERGROUP_PHASE is defined but not used\. Spelling mistake\\?
0 errors and 1 warnings found\..*\$
EOT
			stderr_re => undef,
			exitcode => 0,
		},
		'mail/getmail' => {
			stdout_re => <<EOT,
^looks fine\.\$
EOT
			stderr_re => undef,
			exitcode => 0,
		},
	);

	my $dirprefix = dirname($0) || '.';
	my $pkglint = "$dirprefix/pkglint.pl";
	my $perl = $Config{perlpath};
	for my $package (keys %reference_packages) {
		test_program($perl, [ $pkglint, "@PKGSRCDIR@/$package" ],
			$reference_packages{$package}->{exitcode},
			$reference_packages{$package}->{stdout_re},
			$reference_packages{$package}->{stderr_re});
	}
	# XXX this is JUST like test_unit(), when the tests work, refactor!

}

sub main {
	test_get_vartypes_basictypes();
	test_get_vartypes_map();
	test_checkline_mk_vartype_basic();
	test_pkglint_main();
	test_lint_some_reference_packages();
}

main();
