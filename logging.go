package main

import (
	"fmt"
	"io"
	"strings"
)

const noFile = ""
const noLines = ""

type LogLevel struct {
	traditionalName string
	gccName         string
}

var (
	llFatal   = &LogLevel{"FATAL", "fatal"}
	llError   = &LogLevel{"ERROR", "error"}
	llWarn    = &LogLevel{"WARN", "warning"}
	llNote    = &LogLevel{"NOTE", "note"}
	llDebug   = &LogLevel{"DEBUG", "debug"}
	llAutofix = &LogLevel{"AUTOFIX", "autofix"}
)

var dummyLine = NewLine(noFile, 0, "", nil)

func logf(out io.Writer, level *LogLevel, fname, lineno, format string, args ...interface{}) bool {
	if fname != noFile {
		fname = cleanpath(fname)
	}

	var text, sep string
	if !G.opts.GccOutput {
		text += sep + level.traditionalName + ":"
		sep = " "
	}
	if fname != noFile {
		text += sep + fname
		sep = ": "
		if lineno != noLines {
			text += ":" + lineno
		}
	}
	if G.opts.GccOutput {
		text += sep + level.gccName + ":"
		sep = " "
	}
	text += sep + fmt.Sprintf(format, args...) + "\n"
	io.WriteString(out, text)
	return true
}

func fatalf(fname, lineno, format string, args ...interface{}) {
	logf(G.logErr, llFatal, fname, lineno, format, args...)
	panic(pkglintFatal{})
}
func errorf(fname, lineno, format string, args ...interface{}) bool {
	G.errors++
	return logf(G.logOut, llError, fname, lineno, format, args...)
}
func warnf(fname, lineno, format string, args ...interface{}) bool {
	G.warnings++
	return logf(G.logOut, llWarn, fname, lineno, format, args...)
}
func notef(fname, lineno, format string, args ...interface{}) bool {
	return logf(G.logOut, llNote, fname, lineno, format, args...)
}
func autofixf(fname, lineno, format string, args ...interface{}) bool {
	return logf(G.logOut, llAutofix, fname, lineno, format, args...)
}
func debugf(fname, lineno, format string, args ...interface{}) bool {
	return logf(G.debugOut, llDebug, fname, lineno, format, args...)
}

func explain(explanation ...string) {
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
	}
	G.explanationsAvailable = true
}
func explain1(e1 string)             { explain(e1) }
func explain2(e1, e2 string)         { explain(e1, e2) }
func explain3(e1, e2, e3 string)     { explain(e1, e2, e3) }
func explain4(e1, e2, e3, e4 string) { explain(e1, e2, e3, e4) }

type pkglintFatal struct{}
