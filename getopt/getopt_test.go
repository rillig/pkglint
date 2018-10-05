package getopt

import (
	"gopkg.in/check.v1"
	"strings"
	"testing"
)

type Suite struct{}

var _ = check.Suite(new(Suite))

func Test(t *testing.T) { check.TestingT(t) }

func (s *Suite) Test_Options_Parse_short(c *check.C) {
	opts := NewOptions()
	var help bool
	opts.AddFlagVar('h', "help", &help, false, "prints a help page")

	args, err := opts.Parse([]string{"progname", "-h"})

	c.Assert(err, check.IsNil)
	c.Check(args, check.IsNil)
	c.Check(help, check.Equals, true)
}

func (s *Suite) Test_Options_Parse_unknown_short(c *check.C) {
	opts := NewOptions()

	_, err := opts.Parse([]string{"progname", "-z"})

	c.Check(err.Error(), check.Equals, "progname: unknown option: -z")
}

func (s *Suite) Test_Options_Parse_unknown_long(c *check.C) {
	opts := NewOptions()

	_, err := opts.Parse([]string{"progname", "--unknown-long"})

	c.Check(err.Error(), check.Equals, "progname: unknown option: --unknown-long")
}

func (s *Suite) Test_Options_Parse_unknown_flag_in_group(c *check.C) {
	opts := NewOptions()
	opts.AddFlagGroup('W', "warnings", "", "")

	_, err := opts.Parse([]string{"progname", "-Wall", "-Werror"})

	c.Check(err.Error(), check.Equals, "progname: unknown option: -Werror")

	_, err = opts.Parse([]string{"progname", "--warnings=all", "--warnings=no-error"})

	c.Check(err.Error(), check.Equals, "progname: unknown option: --warnings=no-error")

	_, err = opts.Parse([]string{"progname", "-W"})

	c.Check(err.Error(), check.Equals, "progname: option requires an argument: -W")
}

func (s *Suite) Test_Options_Parse_abbreviated_long(c *check.C) {
	opts := NewOptions()
	var longFlag, longerFlag bool
	opts.AddFlagVar('?', "long", &longFlag, false, "")
	opts.AddFlagVar('?', "longer", &longerFlag, false, "")

	_, err := opts.Parse([]string{"progname", "--lo"})

	c.Check(err.Error(), check.Equals, "progname: ambiguous option: --lo could mean --long or --longer")

	args, err := opts.Parse([]string{"progname", "--long"})

	c.Assert(err, check.IsNil)
	c.Check(args, check.IsNil)
	c.Check(longFlag, check.Equals, true)
	c.Check(longerFlag, check.Equals, false)

	longFlag = false
	args, err = opts.Parse([]string{"progname", "--longe"})

	c.Assert(err, check.IsNil)
	c.Check(args, check.IsNil)
	c.Check(longFlag, check.Equals, false)
	c.Check(longerFlag, check.Equals, true)
}

func (s *Suite) Test_Options_Parse_mixed_args_and_options(c *check.C) {
	opts := NewOptions()
	var aflag, bflag bool
	opts.AddFlagVar('a', "aflag", &aflag, false, "")
	opts.AddFlagVar('b', "bflag", &bflag, false, "")

	args, err := opts.Parse([]string{"progname", "-a", "arg1", "-b", "arg2"})

	c.Assert(err, check.IsNil)
	c.Check(args, check.DeepEquals, []string{"arg1", "arg2"})
	c.Check(aflag, check.Equals, true)
	c.Check(bflag, check.Equals, true)

	aflag = false
	bflag = false
	args, err = opts.Parse([]string{"progname", "-a", "--", "arg1", "-b", "arg2"})

	c.Assert(err, check.IsNil)
	c.Check(args, check.DeepEquals, []string{"arg1", "-b", "arg2"})
	c.Check(aflag, check.Equals, true)
	c.Check(bflag, check.Equals, false)
}

func (s *Suite) Test_Options_Parse_string_list(c *check.C) {
	opts := NewOptions()
	var verbose bool
	var includes []string
	var excludes []string
	opts.AddStrList('e', "exclude", &excludes, "")
	opts.AddStrList('i', "include", &includes, "")
	opts.AddFlagVar('v', "verbose", &verbose, false, "")

	args, err := opts.Parse([]string{"progname",
		"-viincluded1",
		"-i", "included2",
		"--include=included3",
		"--include", "included4",
		"-eexcluded1",
		"-e", "excluded2",
		"--exclude=excluded3",
		"--exclude", "excluded4"})

	c.Check(args, check.IsNil)
	c.Check(err, check.IsNil)
	c.Check(includes, check.DeepEquals, []string{"included1", "included2", "included3", "included4"})
	c.Check(excludes, check.DeepEquals, []string{"excluded1", "excluded2", "excluded3", "excluded4"})

	args, err = opts.Parse([]string{"progname", "-i"})

	c.Check(err.Error(), check.Equals, "progname: option requires an argument: -i")

	args, err = opts.Parse([]string{"progname", "--include"})

	c.Check(err.Error(), check.Equals, "progname: option requires an argument: --include")
}

func (s *Suite) Test_Options_Parse__long_flags(c *check.C) {
	var aflag, bflag, cflag, dflag, eflag, fflag, gflag, hflag, iflag, jflag bool

	opts := NewOptions()
	opts.AddFlagVar('a', "aflag", &aflag, false, "")
	opts.AddFlagVar('b', "bflag", &bflag, false, "")
	opts.AddFlagVar('c', "cflag", &cflag, false, "")
	opts.AddFlagVar('d', "dflag", &dflag, false, "")
	opts.AddFlagVar('e', "eflag", &eflag, true, "")
	opts.AddFlagVar('f', "fflag", &fflag, true, "")
	opts.AddFlagVar('g', "gflag", &gflag, true, "")
	opts.AddFlagVar('h', "hflag", &hflag, true, "")
	opts.AddFlagVar('i', "iflag", &iflag, false, "")
	opts.AddFlagVar('j', "jflag", &jflag, false, "")

	args, err := opts.Parse([]string{"progname",
		"--aflag=true",
		"--bflag=on",
		"--cflag=enabled",
		"--dflag=1",
		"--eflag=false",
		"--fflag=off",
		"--gflag=disabled",
		"--hflag=0",
		"--iflag",
		"--jflag=unknown"})

	c.Check(args, check.HasLen, 0)
	c.Check(aflag, check.Equals, true)
	c.Check(bflag, check.Equals, true)
	c.Check(cflag, check.Equals, true)
	c.Check(dflag, check.Equals, true)
	c.Check(eflag, check.Equals, false)
	c.Check(fflag, check.Equals, false)
	c.Check(gflag, check.Equals, false)
	c.Check(hflag, check.Equals, false)
	c.Check(iflag, check.Equals, true)
	c.Check(err, check.ErrorMatches, `^progname: invalid argument for option --jflag$`)
}

func (s *Suite) Test_Options_handleLongOption__flag_group_without_argument(c *check.C) {
	var extra bool

	opts := NewOptions()
	group := opts.AddFlagGroup('W', "warnings", "warning,...", "Print selected warnings")
	group.AddFlagVar("extra", &extra, false, "Print extra warnings")

	args, err := opts.Parse([]string{"progname", "--warnings"})

	c.Check(args, check.IsNil)
	c.Check(err.Error(), check.Equals, "progname: option requires an argument: --warnings")
	c.Check(extra, check.Equals, false)
}

func (s *Suite) Test_Options_handleLongOption__flag_group_separate_argument(c *check.C) {
	var extra bool

	opts := NewOptions()
	group := opts.AddFlagGroup('W', "warnings", "warning,...", "Print selected warnings")
	group.AddFlagVar("extra", &extra, false, "Print extra warnings")

	args, err := opts.Parse([]string{"progname", "--warnings", "extra,unknown"})

	c.Check(args, check.IsNil)
	c.Check(err.Error(), check.Equals, "progname: unknown option: --warnings=unknown")
	c.Check(extra, check.Equals, true)
}

func (s *Suite) Test_Options_handleLongOption__flag_group_negated(c *check.C) {
	var extra bool

	opts := NewOptions()
	group := opts.AddFlagGroup('W', "warnings", "warning,...", "Print selected warnings")
	group.AddFlagVar("extra", &extra, true, "Print extra warnings")

	args, err := opts.Parse([]string{"progname", "--warnings", "all,no-extra"})

	c.Check(args, check.IsNil)
	c.Check(err, check.IsNil)
	c.Check(extra, check.Equals, false)
}

func (s *Suite) Test_Options_handleLongOption__internal_error(c *check.C) {
	var extra bool

	opts := NewOptions()
	group := opts.AddFlagGroup('W', "warnings", "warning,...", "Print selected warnings")
	group.AddFlagVar("extra", &extra, false, "Print extra warnings")
	opts.options[0].data = "unexpected value"

	c.Check(
		func() { _, _ = opts.Parse([]string{"progname", "--warnings"}) },
		check.Panics,
		"getopt: unknown option type")
}

func (s *Suite) Test_Options_parseShortOptions__flag_group_separate_argument(c *check.C) {
	var extra bool

	opts := NewOptions()
	group := opts.AddFlagGroup('W', "warnings", "warning,...", "Print selected warnings")
	group.AddFlagVar("extra", &extra, false, "Print extra warnings")

	args, err := opts.Parse([]string{"progname", "-W", "extra,unknown"})

	c.Check(args, check.IsNil)
	c.Check(err.Error(), check.Equals, "progname: unknown option: -Wunknown")
	c.Check(extra, check.Equals, true)
}

func (s *Suite) Test_Options_Help(c *check.C) {
	var verbose, basic, extra bool

	opts := NewOptions()
	opts.AddFlagVar('v', "verbose", &verbose, false, "Print a detailed log")
	group := opts.AddFlagGroup('W', "warnings", "warning,...", "Print selected warnings")
	group.AddFlagVar("basic", &basic, true, "Print basic warnings")
	group.AddFlagVar("extra", &extra, false, "Print extra warnings")

	var out strings.Builder
	opts.Help(&out, "progname [options] args")

	c.Check(out.String(), check.Equals, ""+
		"usage: progname [options] args\n"+
		"\n"+
		"  -v, --verbose                Print a detailed log\n"+
		"  -W, --warnings=warning,...   Print selected warnings\n"+
		"\n"+
		"  Flags for -W, --warnings:\n"+
		"    all     all of the following\n"+
		"    none    none of the following\n"+
		"    basic   Print basic warnings (enabled)\n"+
		"    extra   Print extra warnings (disabled)\n"+
		"\n"+
		"  (Prefix a flag with \"no-\" to disable it.)\n")
}
