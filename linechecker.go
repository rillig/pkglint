package main

import (
	"fmt"
	"strings"
)

type LineChecker struct {
	line Line
}

// CheckAbsolutePathname checks whether any absolute pathnames occur in the line.
//
// XXX: Is this check really useful? It had been added 10 years ago because some
// style guide said that "absolute pathnames should be avoided", but there was no
// evidence for that.
func (ck LineChecker) CheckAbsolutePathname(text string) {
	if trace.Tracing {
		defer trace.Call1(text)()
	}

	// In the GNU coding standards, DESTDIR is defined as a (usually
	// empty) prefix that can be used to install files to a different
	// location from what they have been built for. Therefore
	// everything following it is considered an absolute pathname.
	//
	// Another context where absolute pathnames usually appear is in
	// assignments like "bindir=/bin".
	if m, path := match1(text, `(?:^|[\t ]|\$[{(]DESTDIR[)}]|[\w_]+[\t ]*=[\t ]*)(/(?:[^"' \t\\]|"[^"*]"|'[^']*')*)`); m {
		if matches(path, `^/\w`) {
			ck.CheckWordAbsolutePathname(path)
		}
	}
}

func (ck LineChecker) CheckLength(maxlength int) {
	if len(ck.line.Text) > maxlength {
		ck.line.Warnf("Line too long (should be no more than %d characters).", maxlength)
		Explain(
			"Back in the old time, terminals with 80x25 characters were common.",
			"And this is still the default size of many terminal emulators.",
			"Moderately short lines also make reading easier.")
	}
}

func (ck LineChecker) CheckValidCharacters() {
	uni := ""
	for _, r := range ck.line.Text {
		if r != '\t' && !(' ' <= r && r <= '~') {
			uni += fmt.Sprintf(" %U", r)
		}
	}
	if uni != "" {
		ck.line.Warnf("Line contains invalid characters (%s).", uni[1:])
	}
}

func (ck LineChecker) CheckTrailingWhitespace() {
	if strings.HasSuffix(ck.line.Text, " ") || strings.HasSuffix(ck.line.Text, "\t") {
		fix := ck.line.Autofix()
		fix.Notef("Trailing white-space.")
		fix.Explain(
			"When a line ends with some white-space, that space is in most cases",
			"irrelevant and can be removed.")
		fix.ReplaceRegex(`[ \t\r]+\n$`, "\n", 1)
		fix.Apply()
	}
}

// CheckWordAbsolutePathname checks the given word (which is often part of a
// shell command) for absolute pathnames.
//
// XXX: Is this check really useful? It had been added 10 years ago because some
// style guide said that "absolute pathnames should be avoided", but there was no
// evidence for that.
func (ck LineChecker) CheckWordAbsolutePathname(word string) {
	if trace.Tracing {
		defer trace.Call1(word)()
	}

	if !G.Opts.WarnAbsname {
		return
	}

	switch {
	case matches(word, `^/dev/(?:null|tty|zero)$`):
		// These are defined by POSIX.

	case word == "/bin/sh":
		// This is usually correct, although on Solaris, it's pretty feature-crippled.

	case matches(word, `/s\W`):
		// Probably a sed(1) command, e.g. /find/s,replace,with,

	case matches(word, `^/(?:[a-z]|\$[({])`):
		// Absolute paths probably start with a lowercase letter.
		ck.line.Warnf("Found absolute pathname: %s", word)
		if contains(ck.line.Text, "DESTDIR") {
			Explain(
				"Absolute pathnames are often an indicator for unportable code.  As",
				"pkgsrc aims to be a portable system, absolute pathnames should be",
				"avoided whenever possible.",
				"",
				"A special variable in this context is ${DESTDIR}, which is used in",
				"GNU projects to specify a different directory for installation than",
				"what the programs see later when they are executed.  Usually it is",
				"empty, so if anything after that variable starts with a slash, it is",
				"considered an absolute pathname.")
		} else {
			Explain(
				"Absolute pathnames are often an indicator for unportable code.  As",
				"pkgsrc aims to be a portable system, absolute pathnames should be",
				"avoided whenever possible.")
		}
	}
}
