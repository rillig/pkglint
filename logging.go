package main

import (
	"fmt"
	"io"
	"os"
	"path"
)

const NO_FILE string = ""
const NO_LINES string = ""

type LogLevel struct{ traditionalName, gccName string }

var (
	LL_FATAL = LogLevel{"FATAL", "fatal"}
	LL_ERROR = LogLevel{"ERROR", "error"}
	LL_WARN  = LogLevel{"WARN", "warning"}
	LL_NOTE  = LogLevel{"NOTE", "note"}
	LL_DEBUG = LogLevel{"DEBUG", "debug"}
)

func logMessage(level LogLevel, fname, lineno, message string) {
	if fname != NO_FILE {
		fname = path.Clean(fname)
	}

	text, sep := "", ""
	if !G.opts.optGccOutput {
		text += sep + level.traditionalName + ":"
		sep = " "
	}
	if fname != NO_FILE {
		text += sep + fname
		sep = ": "
		if lineno != NO_LINES {
			text += ":" + lineno
		}
	}
	if G.opts.optGccOutput {
		text += sep + level.gccName + ":"
		sep = " "
	}
	text += sep + message + "\n"
	if level == LL_FATAL {
		io.WriteString(os.Stderr, text)
	} else {
		io.WriteString(os.Stdout, text)
	}
}

func logFatal(fname, lineno, format string, args ...interface{}) bool {
	message := fmt.Sprintf(format, args...)
	logMessage(LL_FATAL, fname, lineno, message)
	os.Exit(1)
	return false
}
func logError(fname, lineno, format string, args ...interface{}) bool {
	message := fmt.Sprintf(format, args...)
	logMessage(LL_ERROR, fname, lineno, message)
	G.errors++
	return true
}
func logWarning(fname, lineno, format string, args ...interface{}) bool {
	message := fmt.Sprintf(format, args...)
	logMessage(LL_WARN, fname, lineno, message)
	G.warnings++
	return true
}
func logNote(fname, lineno, format string, args ...interface{}) bool {
	message := fmt.Sprintf(format, args...)
	logMessage(LL_NOTE, fname, lineno, message)
	return true
}
func logDebug(fname, lineno, format string, args ...interface{}) bool {
	message := fmt.Sprintf(format, args...)
	logMessage(LL_DEBUG, fname, lineno, message)
	return true
}

func explain(level LogLevel, fname, lineno string, explanation []string) {
	if G.opts.optExplain {
		out := os.Stdout
		if level == LL_FATAL {
			out = os.Stderr
		}
		for _, explanationLine := range explanation {
			io.WriteString(out, "\t"+explanationLine+"\n")
		}
	}
}

func printSummary() {
	if !G.opts.optQuiet {
		if G.errors != 0 && G.warnings != 0 {
			fmt.Printf("%d errors and %d warnings found.", G.errors, G.warnings)
			if !G.opts.optExplain {
				fmt.Printf("(Use -e for more details.)")
			}
			fmt.Printf("\n")
		} else {
			fmt.Printf("looks fine.\n")
		}
	}
}
