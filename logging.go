package main

import (
	"fmt"
	"io"
	"path"
	"strings"
)

type LogLevel struct {
	TraditionalName string
	GccName         string
}

var (
	Fatal           = &LogLevel{"FATAL", "fatal"}
	Error           = &LogLevel{"ERROR", "error"}
	Warn            = &LogLevel{"WARN", "warning"}
	Note            = &LogLevel{"NOTE", "note"}
	AutofixLogLevel = &LogLevel{"AUTOFIX", "autofix"}
)

var dummyLine = NewLineMulti("", 0, 0, "", nil)
var dummyMkLine = NewMkLine(dummyLine)

// shallBeLogged tests whether a diagnostic with the given format should
// be logged. It only inspects the --only arguments.
//
// Duplicates are handled in main.logf.
func shallBeLogged(format string) bool {
	if len(G.Opts.LogOnly) > 0 {
		found := false
		for _, substr := range G.Opts.LogOnly {
			if contains(format, substr) {
				found = true
				break
			}
		}
		if !found {
			return false
		}
	}

	return true
}

func loggedAlready(filename, lineno, msg string) bool {
	uniq := path.Clean(filename) + ":" + lineno + ":" + msg
	if G.logged[uniq] {
		return true
	}

	if G.logged == nil {
		G.logged = make(map[string]bool)
	}
	G.logged[uniq] = true
	return false
}

// Explain outputs an explanation for the preceding diagnostic
// if the --explain option is given. Otherwise it just records
// that an explanation is available.
func Explain(explanation ...string) {
	if G.Testing {
		for _, s := range explanation {
			if l := tabWidth(s); l > 68 && contains(s, " ") {
				lastSpace := strings.LastIndexByte(s[:68], ' ')
				G.logErr.Printf("Long explanation line: %s\nBreak after: %s\n", s, s[:lastSpace])
			}
			if m, before := match1(s, `(.+)\. [^ ]`); m {
				if !matches(before, `\d$|e\.g`) {
					G.logErr.Printf("Short space after period: %s\n", s)
				}
			}
			if hasSuffix(s, " ") || hasSuffix(s, "\t") {
				G.logErr.Printf("Trailing whitespace: %q\n", s)
			}
		}
	}

	if !G.explainNext {
		return
	}
	G.explanationsAvailable = true
	if !G.Opts.Explain {
		return
	}

	complete := strings.Join(explanation, "\n")
	if G.explanationsGiven[complete] {
		return
	}
	if G.explanationsGiven == nil {
		G.explanationsGiven = make(map[string]bool)
	}
	G.explanationsGiven[complete] = true

	G.logOut.WriteLine("")
	for _, explanationLine := range explanation {
		G.logOut.WriteLine("\t" + explanationLine)
	}
	G.logOut.WriteLine("")

}

type pkglintFatal struct{}

// SeparatorWriter writes output, occasionally separated by an
// empty line. This is used for separating the diagnostics when
// --source is combined with --show-autofix, where each
// log message consists of multiple lines.
type SeparatorWriter struct {
	out            io.Writer
	needSeparator  bool
	wroteSomething bool
}

func NewSeparatorWriter(out io.Writer) *SeparatorWriter {
	return &SeparatorWriter{out, false, false}
}

func (wr *SeparatorWriter) WriteLine(text string) {
	wr.Write(text)
	_, _ = io.WriteString(wr.out, "\n")
}

func (wr *SeparatorWriter) Write(text string) {
	if wr.needSeparator && wr.wroteSomething {
		_, _ = io.WriteString(wr.out, "\n")
		wr.needSeparator = false
	}
	n, err := io.WriteString(wr.out, text)
	if err == nil && n > 0 {
		wr.wroteSomething = true
	}
}

func (wr *SeparatorWriter) Printf(format string, args ...interface{}) {
	wr.Write(fmt.Sprintf(format, args...))
}

func (wr *SeparatorWriter) Separate() {
	wr.needSeparator = true
}
