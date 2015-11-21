package main

// Self-written getopt to support multi-argument options.

import (
	"fmt"
	"io"
	"strings"
	"text/tabwriter"
)

type Options struct {
	options []*Option
	out     io.Writer
}

func NewOptions(out io.Writer) *Options {
	return &Options{nil, out}
}

func (self *Options) AddFlagGroup(shortName rune, longName, argDescription, description string) *FlagGroup {
	grp := new(FlagGroup)
	opt := &Option{shortName, longName, argDescription, description, nil, nil, grp}
	self.options = append(self.options, opt)
	return grp
}

func (self *Options) AddFlagVar(shortName rune, longName string, flag *bool, defval bool, description string) {
	*flag = defval
	opt := &Option{shortName, longName, "", description, &flag, nil, nil}
	self.options = append(self.options, opt)
}

func (self *Options) Parse(args []string) ([]string, error) {
	for i := 1; i < len(args); i++ {
		arg := args[i]
		if arg == "--" {
			return args[i+1:], nil
		} else if hasPrefix(arg, "--") {
			i += self.parseLongOption(args, i, arg[2:])
		} else if hasPrefix(arg, "-") {
			i += self.parseShortOptions(args, i, arg[1:])
		} else {
			return args[i:], nil
		}
	}
	return args[len(args):], nil
}

func (self *Options) parseLongOption(args []string, i int, arg string) int {
	for _, opt := range self.options {
		if arg == opt.longName {
			if opt.flagGroup != nil {
				opt.flagGroup.parse(args[i+1])
				return 1
			}
			if opt.flag != nil {
				**opt.flag = true
				return 0
			}
			panic("not implemented: " + opt.longName)
		} else if prefix := opt.longName + "="; hasPrefix(arg, prefix) {
			if opt.flagGroup != nil {
				opt.flagGroup.parse(arg[len(prefix):])
				return 0
			}
			panic("not implemented: " + opt.longName)
		}
	}
	panic("not implemented: " + arg)
}

func (self *Options) parseShortOptions(args []string, i int, arg string) int {
	for ai, optchar := range arg {
		for _, opt := range self.options {
			if optchar == opt.shortName {
				if opt.flag != nil {
					**opt.flag = true
				} else if opt.str != nil {
					argarg := strings.TrimPrefix(arg, sprintf("%s%c", arg[:ai], optchar))
					if argarg != "" {
						**opt.str = argarg
						return 0
					}
					**opt.str = args[i+1]
					return 1
				} else if opt.flagGroup != nil {
					argarg := strings.TrimPrefix(arg, sprintf("%s%c", arg[:ai], optchar))
					if argarg != "" {
						opt.flagGroup.parse(argarg)
						return 0
					}
					opt.flagGroup.parse(args[i+1])
					return 1
				} else {
					panic("not implemented: " + arg[ai:])
				}
			}
		}
	}
	return 0
}

func (self *Options) Help(generalUsage string) {
	wr := tabwriter.NewWriter(self.out, 1, 0, 2, ' ', tabwriter.TabIndent)

	fmt.Fprintf(wr, "usage: %s\n", generalUsage)
	fmt.Fprintln(wr)
	wr.Flush()

	for _, opt := range self.options {
		if opt.argDescription == "" {
			fmt.Fprintf(wr, "  -%c, --%s\t %s\n",
				opt.shortName, opt.longName, opt.description)
		} else {
			fmt.Fprintf(wr, "  -%c, --%s=%s\t %s\n",
				opt.shortName, opt.longName, opt.argDescription, opt.description)
		}
	}
	wr.Flush()

	hasFlagGroups := false
	for _, opt := range self.options {
		if opt.flagGroup != nil {
			hasFlagGroups = true
			fmt.Fprintln(wr)
			fmt.Fprintf(wr, "  Flags for -%c, --%s:\n", opt.shortName, opt.longName)
			fmt.Fprintf(wr, "    all\t all of the following\n")
			fmt.Fprintf(wr, "    none\t none of the following\n")
			for _, flag := range opt.flagGroup.flags {
				fmt.Fprintf(wr, "    %s\t %s (%v)\n", flag.name, flag.help, ifelseStr(*flag.value, "enabled", "disabled"))
			}
			wr.Flush()
		}
	}
	if hasFlagGroups {
		fmt.Fprintln(wr)
		fmt.Fprint(wr, "  (Prefix a flag with \"no-\" to disable it.)\n")
		wr.Flush()
	}
}

type Option struct {
	shortName      rune
	longName       string
	argDescription string
	description    string
	flag           **bool
	str            **string
	flagGroup      *FlagGroup
}

type FlagGroup struct {
	flags []*GroupFlag
}

func (self *FlagGroup) AddFlagVar(name string, flag *bool, defval bool, help string) {
	opt := &GroupFlag{name, flag, help}
	self.flags = append(self.flags, opt)
	*flag = defval
}

func (self *FlagGroup) parse(arg string) {
argopt:
	for _, argopt := range strings.Split(arg, ",") {
		if argopt == "none" || argopt == "all" {
			for _, opt := range self.flags {
				*opt.value = argopt == "all"
			}
			continue argopt
		}
		for _, opt := range self.flags {
			if argopt == opt.name {
				*opt.value = true
				continue argopt
			}
			if argopt == "no-"+opt.name {
				*opt.value = false
				continue argopt
			}
		}
		panic("FlagGroup.parse: " + argopt)
	}
}

type GroupFlag struct {
	name  string
	value *bool
	help  string
}
