package main

import (
	"fmt"
	"io"
	"netbsd.org/pkglint/histogram"
	"path"
)

type Logger struct {
	Opts LoggerOpts

	out *SeparatorWriter
	err *SeparatorWriter

	suppressDiag bool
	suppressExpl bool

	logged    Once
	explained Once
	histo     *histogram.Histogram

	errors                int
	warnings              int
	explanationsAvailable bool
	autofixAvailable      bool
}

type LoggerOpts struct {
	ShowAutofix,
	Autofix,
	Explain,
	ShowSource,
	LogVerbose,
	GccOutput,
	Quiet bool
}

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

func (l *Logger) IsAutofix() bool { return l.Opts.Autofix || l.Opts.ShowAutofix }

// Relevant decides and remembers whether the given diagnostic is relevant and should be logged.
//
// The result of the decision affects all log items until Relevant is called for the next time.
func (l *Logger) Relevant(format string) bool {
	relevant := l.shallBeLogged(format)
	l.suppressDiag = !relevant
	l.suppressExpl = !relevant
	return relevant
}

func (l *Logger) FirstTime(filename, linenos, msg string) bool {
	if l.Opts.LogVerbose {
		return true
	}

	if !l.logged.FirstTimeSlice(path.Clean(filename), linenos, msg) {
		l.suppressDiag = true
		l.suppressExpl = true
		return false
	}

	return true
}

// Explain outputs an explanation for the preceding diagnostic
// if the --explain option is given. Otherwise it just records
// that an explanation is available.
func (l *Logger) Explain(explanation ...string) {
	if l.suppressExpl {
		l.suppressExpl = false
		return
	}

	l.explanationsAvailable = true
	if !l.Opts.Explain {
		return
	}

	if !l.explained.FirstTimeSlice(explanation...) {
		return
	}

	l.out.Separate()
	wrapped := wrap(68, explanation...)
	for _, explanationLine := range wrapped {
		l.out.Write("\t")
		l.out.WriteLine(explanationLine)
	}
	l.out.WriteLine("")
}

func (l *Logger) ShowSummary() {
	if !l.Opts.Quiet && !l.Opts.Autofix {
		if l.errors != 0 || l.warnings != 0 {
			l.out.Printf("%d %s and %d %s found.\n",
				l.errors, ifelseStr(l.errors == 1, "error", "errors"),
				l.warnings, ifelseStr(l.warnings == 1, "warning", "warnings"))
		} else {
			l.out.WriteLine("Looks fine.")
		}
		if l.explanationsAvailable && !l.Opts.Explain {
			l.out.WriteLine("(Run \"pkglint -e\" to show explanations.)")
		}
		if l.autofixAvailable && !l.Opts.ShowAutofix {
			l.out.WriteLine("(Run \"pkglint -fs\" to show what can be fixed automatically.)")
		}
		if l.autofixAvailable && !l.Opts.Autofix {
			l.out.WriteLine("(Run \"pkglint -F\" to automatically fix some issues.)")
		}
	}
}

// shallBeLogged tests whether a diagnostic with the given format should
// be logged.
//
// It only inspects the --only arguments; duplicates are handled in Logger.Logf.
func (l *Logger) shallBeLogged(format string) bool {
	if len(G.Opts.LogOnly) == 0 {
		return true
	}

	for _, substr := range G.Opts.LogOnly {
		if contains(format, substr) {
			return true
		}
	}
	return false
}

// Diag logs a diagnostic. These are filtered by the --only command line option,
// and duplicates are suppressed unless the --log-verbose command line option is given.
//
// See Logf for logging arbitrary messages.
func (l *Logger) Diag(line Line, level *LogLevel, format string, args ...interface{}) {
	if l.Opts.ShowAutofix || l.Opts.Autofix {
		// In these two cases, the only interesting diagnostics are those that can
		// be fixed automatically. These are logged by Autofix.Apply.
		return
	}

	if !l.Relevant(format) {
		return
	}

	filename := line.Filename
	linenos := line.Linenos()
	msg := fmt.Sprintf(format, args...)
	if !l.FirstTime(filename, linenos, msg) {
		l.suppressDiag = false
		return
	}

	if l.Opts.ShowSource {
		line.showSource(l.out)
		l.Logf(level, filename, linenos, format, msg)
		l.out.Separate()
	} else {
		l.Logf(level, filename, linenos, format, msg)
	}
}

func (l *Logger) Logf(level *LogLevel, filename, lineno, format, msg string) {
	if l.suppressDiag {
		l.suppressDiag = false
		return
	}

	// TODO: Only ever output ASCII, no matter what's in the message.

	if G.Testing && format != AutofixFormat && !hasSuffix(format, ": %s") && !hasSuffix(format, ". %s") {
		G.Assertf(hasSuffix(format, "."), "Diagnostic format %q must end in a period.", format)
	}

	if filename != "" {
		filename = cleanpath(filename)
	}
	if G.Opts.Profiling && format != AutofixFormat && level != Fatal {
		l.histo.Add(format, 1)
	}

	out := l.out
	if level == Fatal {
		out = l.err
	}

	filenameSep := ifelseStr(filename != "", ": ", "")
	linenoSep := ifelseStr(filename != "" && lineno != "", ":", "")
	if l.Opts.GccOutput {
		out.Printf("%s%s%s%s%s: %s\n", filename, linenoSep, lineno, filenameSep, level.GccName, msg)
	} else {
		out.Printf("%s%s%s%s%s: %s\n", level.TraditionalName, filenameSep, filename, linenoSep, lineno, msg)
	}

	switch level {
	case Fatal:
		panic(pkglintFatal{})
	case Error:
		l.errors++
	case Warn:
		l.warnings++
	}
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
