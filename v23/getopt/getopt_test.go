package getopt

import (
	"github.com/rillig/pkglint/v23/intqa"
	"reflect"
	"strings"
	"testing"
)

func Test_Options_Parse__short(t *testing.T) {
	opts := NewOptions()
	var help bool
	opts.AddFlagVar('h', "help", &help, false, "prints a help page")

	args, err := opts.Parse([]string{"progname", "-h"})
	if err != nil {
		t.Fatal(err)
	}
	if args != nil {
		t.Error(args)
	}
	if !help {
		t.Error(help)
	}
}

func Test_Options_Parse__short_string(t *testing.T) {
	opts := NewOptions()
	var help bool
	var src string
	var dst string
	opts.AddFlagVar('h', "help", &help, false, "prints a help page")
	opts.AddStrVar('s', "src", &src, "", "source of the copy operation")
	opts.AddStrVar('d', "dst", &dst, "", "destination of the copy operation")

	args, err := opts.Parse([]string{"copy", "-hssource", "-d", "destination"})

	if err != nil {
		t.Fatal(err)
	}
	if args != nil {
		t.Error(args)
	}
	if !help {
		t.Error(help)
	}
	if src != "source" {
		t.Error(src)
	}
	if dst != "destination" {
		t.Error(dst)
	}
}

func Test_Options_Parse__short_string_unfinished(t *testing.T) {
	opts := NewOptions()
	var unfinished string
	opts.AddStrVar('u', "unfinished", &unfinished, "", "demo")

	args, err := opts.Parse([]string{"program", "-u"})

	if err.Error() != "program: option requires an argument: -u" {
		t.Error(err)
	}
	if args != nil {
		t.Error(args)
	}
	if unfinished != "" {
		t.Error(unfinished)
	}
}

func Test_Options_Parse__unknown_short(t *testing.T) {
	opts := NewOptions()

	_, err := opts.Parse([]string{"progname", "-z"})

	if err.Error() != "progname: unknown option: -z" {
		t.Error(err)
	}
}

func Test_Options_Parse__unknown_short_with_argument(t *testing.T) {
	opts := NewOptions()

	_, err := opts.Parse([]string{"progname", "-z", "arg"})

	if err.Error() != "progname: unknown option: -z" {
		t.Error(err)
	}
}

func Test_Options_Parse__unknown_long(t *testing.T) {
	opts := NewOptions()

	_, err := opts.Parse([]string{"progname", "--unknown-long"})

	if err.Error() != "progname: unknown option: --unknown-long" {
		t.Error(err)
	}
}

func Test_Options_Parse__unknown_flag_in_group(t *testing.T) {
	opts := NewOptions()
	opts.AddFlagGroup('W', "warnings", "", "")

	_, err := opts.Parse([]string{"progname", "-Wall", "-Werror"})

	if err.Error() != "progname: unknown option: -Werror" {
		t.Error(err)
	}

	_, err = opts.Parse([]string{"progname", "--warnings=all", "--warnings=no-error"})

	if err.Error() != "progname: unknown option: --warnings=no-error" {
		t.Error(err)
	}

	_, err = opts.Parse([]string{"progname", "-W"})

	if err.Error() != "progname: option requires an argument: -W" {
		t.Error(err)
	}
}

func Test_Options_Parse__abbreviated_long(t *testing.T) {
	opts := NewOptions()
	var longFlag, longerFlag bool
	opts.AddFlagVar('?', "long", &longFlag, false, "")
	opts.AddFlagVar('?', "longer", &longerFlag, false, "")

	_, err := opts.Parse([]string{"progname", "--lo"})

	if err.Error() != "progname: ambiguous option: --lo could mean --long or --longer" {
		t.Error(err)
	}

	args, err := opts.Parse([]string{"progname", "--long"})

	if err != nil {
		t.Fatal(err)
	}
	if args != nil {
		t.Error(args)
	}
	if longFlag != true {
		t.Error(longFlag)
	}
	if longerFlag != false {
		t.Error(longerFlag)
	}

	longFlag = false
	args, err = opts.Parse([]string{"progname", "--longe"})

	if err != nil {
		t.Fatal(err)
	}
	if args != nil {
		t.Error(args)
	}
	if longFlag != false {
		t.Error(longFlag)
	}
	if longerFlag != true {
		t.Error(longerFlag)
	}
}

func Test_Options_Parse__mixed_args_and_options(t *testing.T) {
	opts := NewOptions()
	var aflag, bflag bool
	opts.AddFlagVar('a', "aflag", &aflag, false, "")
	opts.AddFlagVar('b', "bflag", &bflag, false, "")

	args, err := opts.Parse([]string{"progname", "-a", "arg1", "-b", "arg2"})

	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(args, []string{"arg1", "arg2"}) {
		t.Error(args)
	}
	if aflag != true {
		t.Error(aflag)
	}
	if bflag != true {
		t.Error(bflag)
	}

	aflag = false
	bflag = false
	args, err = opts.Parse([]string{"progname", "-a", "--", "arg1", "-b", "arg2"})

	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(args, []string{"arg1", "-b", "arg2"}) {
		t.Error(args)
	}
	if aflag != true {
		t.Error(aflag)
	}
	if bflag != false {
		t.Error(bflag)
	}
}

func Test_Options_Parse__string_list(t *testing.T) {
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

	if args != nil {
		t.Error(args)
	}
	if err != nil {
		t.Error(err)
	}
	if !reflect.DeepEqual(includes, []string{"included1", "included2", "included3", "included4"}) {
		t.Error(includes)
	}
	if !reflect.DeepEqual(excludes, []string{"excluded1", "excluded2", "excluded3", "excluded4"}) {
		t.Error(excludes)
	}

	_, err = opts.Parse([]string{"progname", "-i"})

	if err == nil || err.Error() != "progname: option requires an argument: -i" {
		t.Error(err)
	}

	_, err = opts.Parse([]string{"progname", "--include"})

	if err == nil || err.Error() != "progname: option requires an argument: --include" {
		t.Error(err)
	}
}

func Test_Options_Parse__long_flags(t *testing.T) {
	var posFlags [5]bool
	var negFlags [5]bool
	var otherFlags [2]bool

	opts := NewOptions()
	opts.AddFlagVar(0, "pos0", &posFlags[0], false, "")
	opts.AddFlagVar(0, "pos1", &posFlags[1], false, "")
	opts.AddFlagVar(0, "pos2", &posFlags[2], false, "")
	opts.AddFlagVar(0, "pos3", &posFlags[3], false, "")
	opts.AddFlagVar(0, "pos4", &posFlags[4], false, "")
	opts.AddFlagVar(0, "neg0", &negFlags[0], true, "")
	opts.AddFlagVar(0, "neg1", &negFlags[1], true, "")
	opts.AddFlagVar(0, "neg2", &negFlags[2], true, "")
	opts.AddFlagVar(0, "neg3", &negFlags[3], true, "")
	opts.AddFlagVar(0, "neg4", &negFlags[4], true, "")
	opts.AddFlagVar(0, "other0", &otherFlags[0], false, "")
	opts.AddFlagVar(0, "other1", &otherFlags[1], false, "")

	args, err := opts.Parse([]string{"progname",
		"--pos0=true",
		"--pos1=on",
		"--pos2=enabled",
		"--pos3=1",
		"--pos4=yes",
		"--neg0=false",
		"--neg1=off",
		"--neg2=disabled",
		"--neg3=0",
		"--neg4=no",
		"--other0",
		"--other1=unknown"})

	if len(args) > 0 {
		t.Error(len(args))
	}
	if posFlags != [5]bool{true, true, true, true, true} {
		t.Error(posFlags)
	}
	if negFlags != [5]bool{false, false, false, false, false} {
		t.Error(negFlags)
	}
	if otherFlags != [2]bool{true, false} {
		t.Error(otherFlags)
	}
	if err.Error() != "progname: invalid argument for option --other1" {
		t.Error(err)
	}
}

func Test_Options_Parse__long_string(t *testing.T) {
	opts := NewOptions()
	var src, dst string
	opts.AddStrVar('s', "src", &src, "", "source of the copy operation")
	opts.AddStrVar('d', "dst", &dst, "", "destination of the copy operation")

	args, err := opts.Parse([]string{"copy", "--src=source", "--dst", "destination", "arg"})

	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(args, []string{"arg"}) {
		t.Error(args)
	}
	if src != "source" {
		t.Error(src)
	}
	if dst != "destination" {
		t.Error(dst)
	}
}

func Test_Options_Parse__long_string_unfinished(t *testing.T) {
	opts := NewOptions()
	var unfinished string
	opts.AddStrVar('u', "unfinished", &unfinished, "", "unfinished option")

	args, err := opts.Parse([]string{"program", "--unfinished"})

	if err.Error() != "program: option requires an argument: --unfinished" {
		t.Error(err)
	}
	if args != nil {
		t.Error(args)
	}
	if unfinished != "" {
		t.Error(unfinished)
	}
}

// From an implementation standpoint, it would be a likely bug to interpret
// the "--" as the long name of the option, and that would set the flag
// to true.
func Test_Options_Parse__only_short(t *testing.T) {
	var onlyShort bool
	opts := NewOptions()
	opts.AddFlagVar('s', "", &onlyShort, false, "only short")

	args, err := opts.Parse([]string{"program", "--", "arg"})

	if err != nil {
		t.Error(err)
	}
	if !reflect.DeepEqual(args, []string{"arg"}) {
		t.Error(args)
	}
	if onlyShort != false {
		t.Error(onlyShort)
	}
}

func Test_Options_Parse__only_long(t *testing.T) {
	var onlyLong bool
	opts := NewOptions()
	opts.AddFlagVar(0, "long", &onlyLong, false, "only long")

	args, err := opts.Parse([]string{"program", "-", "arg"})

	if err != nil {
		t.Error(err)
	}
	if !reflect.DeepEqual(args, []string{"-", "arg"}) {
		t.Error(args)
	}
	if onlyLong != false {
		t.Error(onlyLong)
	}
}

func Test_Options_handleLongOption__string(t *testing.T) {
	var extra bool

	opts := NewOptions()

	group := opts.AddFlagGroup('W', "warnings", "warning,...", "Print selected warnings")
	group.AddFlagVar("extra", &extra, false, "Print extra warnings")

	args, err := opts.Parse([]string{"progname", "--warnings"})

	if args != nil {
		t.Error(args)
	}
	if err.Error() != "progname: option requires an argument: --warnings" {
		t.Error(err)
	}
	if extra != false {
		t.Error(extra)
	}
}

func Test_Options_handleLongOption__flag_group_without_argument(t *testing.T) {
	var extra bool

	opts := NewOptions()
	group := opts.AddFlagGroup('W', "warnings", "warning,...", "Print selected warnings")
	group.AddFlagVar("extra", &extra, false, "Print extra warnings")

	args, err := opts.Parse([]string{"progname", "--warnings"})

	if args != nil {
		t.Error(args)
	}
	if err.Error() != "progname: option requires an argument: --warnings" {
		t.Error(err)
	}
	if extra != false {
		t.Error(extra)
	}
}

func Test_Options_handleLongOption__flag_group_separate_argument(t *testing.T) {
	var extra bool

	opts := NewOptions()
	group := opts.AddFlagGroup('W', "warnings", "warning,...", "Print selected warnings")
	group.AddFlagVar("extra", &extra, false, "Print extra warnings")

	args, err := opts.Parse([]string{"progname", "--warnings", "extra,unknown"})

	if args != nil {
		t.Error(args)
	}
	if err.Error() != "progname: unknown option: --warnings=unknown" {
		t.Error(err)
	}
	if extra != true {
		t.Error(extra)
	}
}

func Test_Options_handleLongOption__flag_group_all_then_disable(t *testing.T) {
	var false1, false2, true1, true2 bool

	opts := NewOptions()
	group := opts.AddFlagGroup('a', "answers", "answer,...", "Choose the answers")
	group.AddFlagVar("false1", &false1, false, "A")
	group.AddFlagVar("false2", &false2, false, "B")
	group.AddFlagVar("true1", &true1, true, "C")
	group.AddFlagVar("true2", &true2, true, "C")

	args, err := opts.Parse([]string{"progname", "--answers", "all,no-false1,no-true1", "arg"})

	if err != nil {
		t.Error(err)
	}
	if !reflect.DeepEqual(args, []string{"arg"}) {
		t.Error(args)
	}
	if false1 != false {
		t.Error(false1)
	}
	if false2 != true {
		t.Error(false2)
	}
	if true1 != false {
		t.Error(true1)
	}
	if true2 != true {
		t.Error(true2)
	}
}

func Test_Options_handleLongOption__flag_group_none_then_enable(t *testing.T) {
	var false1, false2, true1, true2 bool

	opts := NewOptions()
	group := opts.AddFlagGroup('a', "answers", "answer,...", "Choose the answers")
	group.AddFlagVar("false1", &false1, false, "A")
	group.AddFlagVar("false2", &false2, false, "B")
	group.AddFlagVar("true1", &true1, true, "C")
	group.AddFlagVar("true2", &true2, true, "C")

	args, err := opts.Parse([]string{"progname", "--answers", "none,false1,true1", "arg"})

	if err != nil {
		t.Error(err)
	}
	if !reflect.DeepEqual(args, []string{"arg"}) {
		t.Error(args)
	}
	if !false1 {
		t.Error(false1)
	}
	if false2 {
		t.Error(false2)
	}
	if !true1 {
		t.Error(true1)
	}
	if true2 {
		t.Error(true2)
	}
}

func Test_Options_handleLongOption__internal_error(t *testing.T) {
	var extra bool

	opts := NewOptions()
	group := opts.AddFlagGroup('W', "warnings", "warning,...", "Print selected warnings")
	group.AddFlagVar("extra", &extra, false, "Print extra warnings")

	// Intentionally damage internal structure to reach full code coverage.
	opts.options[0].data = "unexpected value"

	var p any
	func() {
		defer func() {
			p = recover()
		}()
		_, _ = opts.Parse([]string{"progname", "--warnings"})
	}()

	if p != "getopt: internal error: unknown option type" {
		t.Error(p)
	}
}

func Test_Options_parseShortOptions__flag_group_separate_argument(t *testing.T) {
	var extra bool

	opts := NewOptions()
	group := opts.AddFlagGroup('W', "warnings", "warning,...", "Print selected warnings")
	group.AddFlagVar("extra", &extra, false, "Print extra warnings")

	args, err := opts.Parse([]string{"progname", "-W", "extra,unknown"})

	if args != nil {
		t.Error(args)
	}
	if err.Error() != "progname: unknown option: -Wunknown" {
		t.Error(err)
	}
	if !extra {
		t.Error(extra)
	}
}

func Test_Options_Help(t *testing.T) {
	var verbose bool
	var name string

	opts := NewOptions()
	opts.AddFlagVar('v', "verbose", &verbose, false, "Print a detailed log")
	opts.AddStrVar('n', "name", &name, "", "Name of the print job")

	var out strings.Builder
	opts.Help(&out, "progname [options] args")

	if out.String() != ""+
		"usage: progname [options] args\n"+
		"\n"+
		"  -v, --verbose   Print a detailed log\n"+
		"  -n, --name      Name of the print job\n" {
		t.Error(out)
	}
}

func Test_Options_Help__with_flag_group(t *testing.T) {
	var verbose, basic, extra bool
	var name string

	opts := NewOptions()
	opts.AddFlagVar('v', "verbose", &verbose, false, "Print a detailed log")
	opts.AddStrVar('n', "name", &name, "", "Name of the print job")
	group := opts.AddFlagGroup('W', "warnings", "warning,...", "Print selected warnings")
	group.AddFlagVar("basic", &basic, true, "Print basic warnings")
	group.AddFlagVar("extra", &extra, false, "Print extra warnings")

	var out strings.Builder
	opts.Help(&out, "progname [options] args")

	if out.String() != ""+
		"usage: progname [options] args\n"+
		"\n"+
		"  -v, --verbose                Print a detailed log\n"+
		"  -n, --name                   Name of the print job\n"+
		"  -W, --warnings=warning,...   Print selected warnings\n"+
		"\n"+
		"  Flags for -W, --warnings:\n"+
		"    all     all of the following\n"+
		"    none    none of the following\n"+
		"    basic   Print basic warnings (enabled)\n"+
		"    extra   Print extra warnings (disabled)\n"+
		"\n"+
		"  (Prefix a flag with \"no-\" to disable it.)\n" {
		t.Error(out)
	}
}

func Test_Options_Help__partial(t *testing.T) {
	var onlyShort, onlyLong bool

	opts := NewOptions()
	opts.AddFlagVar('s', "", &onlyShort, false, "Only short option")
	opts.AddFlagVar(0, "long", &onlyLong, false, "Only long option")

	var out strings.Builder
	opts.Help(&out, "progname [options] args")

	if out.String() != ""+
		"usage: progname [options] args\n"+
		"\n"+
		"  -s       Only short option\n"+
		"  --long   Only long option\n" {
		t.Error(out)
	}
}

func Test_FlagGroup_AddFlagVarNoAll(t *testing.T) {
	opts := NewOptions()

	var warnCommon, warnError bool
	group := opts.AddFlagGroup('W', "warn", "warnings", "descr")
	group.AddFlagVar("common", &warnCommon, false, "Enable common warnings")
	group.AddFlagVarNoAll("error", &warnError, false, "Treat warnings as errors")

	args, err := opts.Parse([]string{"program", "-Wall"})
	if err != nil {
		t.Error(err)
	}
	if args != nil {
		t.Error(args)
	}
	if !warnCommon {
		t.Error(warnCommon)
	}
	if warnError {
		t.Error(warnError)
	}
}

func Test__qa(t *testing.T) {
	ck := intqa.NewQAChecker(t.Errorf)
	ck.Configure("*", "*", "*", -intqa.EMissingTest)
	ck.Check()
}
