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
	if !GlobalVars.opts.optGccOutput {
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
	if GlobalVars.opts.optGccOutput {
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

func logFatalF(fname, lineno, format string, args ...interface{}) bool {
	message := fmt.Sprintf(format, args...)
	logMessage(LL_FATAL, fname, lineno, message)
	os.Exit(1)
	return false
}
func logErrorF(fname, lineno, format string, args ...interface{}) bool {
	message := fmt.Sprintf(format, args...)
	logMessage(LL_ERROR, fname, lineno, message)
	GlobalVars.errors++
	return true
}
func logWarningF(fname, lineno, format string, args ...interface{}) bool {
	message := fmt.Sprintf(format, args...)
	logMessage(LL_WARN, fname, lineno, message)
	GlobalVars.warnings++
	return true
}
func logNoteF(fname, lineno, format string, args ...interface{}) bool {
	message := fmt.Sprintf(format, args...)
	logMessage(LL_NOTE, fname, lineno, message)
	return true
}
func logDebugF(fname, lineno, format string, args ...interface{}) bool {
	message := fmt.Sprintf(format, args...)
	logMessage(LL_DEBUG, fname, lineno, message)
	return true
}

func explain(level LogLevel, fname, lineno string, explanation []string) {
	if GlobalVars.opts.optExplain {
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
	if !GlobalVars.opts.optQuiet {
		if GlobalVars.errors != 0 && GlobalVars.warnings != 0 {
			fmt.Printf("%d errors and %d warnings found.", GlobalVars.errors, GlobalVars.warnings)
			if !GlobalVars.opts.optExplain {
				fmt.Printf("(Use -e for more details.)")
			}
			fmt.Printf("\n")
		} else {
			fmt.Printf("looks fine.\n")
		}
	}
}
