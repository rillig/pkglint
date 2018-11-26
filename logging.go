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

	pkglint.logOut.Separate()
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
	out   io.Writer
	state uint8 // 0 = beginning of line, 1 = in line, 2 = separator wanted, 3 = paragraph
}

func NewSeparatorWriter(out io.Writer) *SeparatorWriter {
	return &SeparatorWriter{out, 0}
}

func (wr *SeparatorWriter) WriteLine(text string) {
	wr.Write(text)
	wr.write('\n')
}

func (wr *SeparatorWriter) Write(text string) {
	for _, b := range []byte(text) {
		wr.write(b)
	}
}

func (wr *SeparatorWriter) Printf(format string, args ...interface{}) {
	wr.Write(fmt.Sprintf(format, args...))
}

// Separate remembers to output an empty line before the next character.
// If the writer is currently in the middle of a line, that line is terminated immediately.
func (wr *SeparatorWriter) Separate() {
	if wr.state == 1 {
		_, _ = wr.out.Write([]byte{'\n'})
	}
	if wr.state < 2 {
		wr.state = 2
	}
}

func (wr *SeparatorWriter) write(b byte) {
	switch {
	case b == '\n':
		if wr.state == 1 {
			wr.state = 0
		} else {
			wr.state = 3
		}
	default:
		if wr.state == 2 {
			_, _ = wr.out.Write([]byte{'\n'})
		}
		wr.state = 1
	}
	_, _ = wr.out.Write([]byte{b})
}
