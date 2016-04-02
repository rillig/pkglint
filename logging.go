package main

import (
	"fmt"
	"io"
	"strings"
)

type LogLevel struct {
	TraditionalName string
	GccName         string
}

var (
	llFatal   = &LogLevel{"FATAL", "fatal"}
	llError   = &LogLevel{"ERROR", "error"}
	llWarn    = &LogLevel{"WARN", "warning"}
	llNote    = &LogLevel{"NOTE", "note"}
	llDebug   = &LogLevel{"DEBUG", "debug"}
	llAutofix = &LogLevel{"AUTOFIX", "autofix"}
)

var dummyLine = NewLine("", 0, "", nil)

func logs(out io.Writer, level *LogLevel, fname, lineno, format, msg string) bool {
	if fname != "" {
		fname = cleanpath(fname)
	}

	var text, sep string
	if !G.opts.GccOutput {
		text += sep + level.TraditionalName + ":"
		sep = " "
	}
	if fname != "" {
		text += sep + fname
		sep = ": "
		if lineno != "" {
			text += ":" + lineno
		}
	}
	if G.opts.GccOutput {
		text += sep + level.GccName + ":"
		sep = " "
	}
	if G.opts.Profiling {
		G.loghisto.Add(format, 1)
	}
	text += sep + msg + "\n"
	io.WriteString(out, text)
	return true
}

func Fatals(fname, lineno, format, msg string) {
	logs(G.logErr, llFatal, fname, lineno, format, msg)
	panic(pkglintFatal{})
}
func Errors(fname, lineno, format, msg string) bool {
	G.errors++
	return logs(G.logOut, llError, fname, lineno, format, msg)
}
func Warns(fname, lineno, format, msg string) bool {
	G.warnings++
	return logs(G.logOut, llWarn, fname, lineno, format, msg)
}
func Notes(fname, lineno, format, msg string) bool {
	return logs(G.logOut, llNote, fname, lineno, format, msg)
}
func autofixs(fname, lineno, format, msg string) bool {
	return logs(G.logOut, llAutofix, fname, lineno, format, msg)
}
func Debugs(fname, lineno, format, msg string) bool {
	return logs(G.debugOut, llDebug, fname, lineno, format, msg)
}

func Explain(explanation ...string) {
	if G.opts.Explain {
		complete := strings.Join(explanation, "\n")
		if G.explanationsGiven[complete] {
			return
		}
		if G.explanationsGiven == nil {
			G.explanationsGiven = make(map[string]bool)
		}
		G.explanationsGiven[complete] = true

		io.WriteString(G.logOut, "\n")
		for _, explanationLine := range explanation {
			io.WriteString(G.logOut, "\t"+explanationLine+"\n")
		}
		io.WriteString(G.logOut, "\n")
	} else if G.Testing {
		for _, s := range explanation {
			if l := tabLength(s); l > 68 && contains(s, " ") {
				print(fmt.Sprintf("Long explanation line (%d): %s\n", l, s))
			}
			if m, before := match1(s, `(.+)\. [^ ]`); m {
				if !matches(before, `\d$`) {
					print(fmt.Sprintf("Short space after period: %s\n", s))
				}
			}
		}
	}
	G.explanationsAvailable = true
}
func Explain1(e1 string)             { Explain(e1) }
func Explain2(e1, e2 string)         { Explain(e1, e2) }
func Explain3(e1, e2, e3 string)     { Explain(e1, e2, e3) }
func Explain4(e1, e2, e3, e4 string) { Explain(e1, e2, e3, e4) }

type pkglintFatal struct{}
