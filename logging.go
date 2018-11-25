package main

import (
	"fmt"
	"io"
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

// Explain outputs an explanation for the preceding diagnostic
// if the --explain option is given. Otherwise it just records
// that an explanation is available.
func (pkglint *Pkglint) Explain(explanation ...string) {

	if !pkglint.explainNext {
		return
	}
	pkglint.explanationsAvailable = true
	if !pkglint.Opts.Explain {
		return
	}

	if !pkglint.explained.FirstTimeSlice(explanation...) {
		return
	}

	pkglint.logOut.WriteLine("")
	wrapped := wrap(68, explanation...)
	for _, explanationLine := range wrapped {
		pkglint.logOut.Write("\t")
		pkglint.logOut.WriteLine(explanationLine)
	}
	pkglint.logOut.WriteLine("")

}

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
