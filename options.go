package main

import "netbsd.org/pkglint/trace"

func ChecklinesOptionsMk(mklines *MkLines) {
	if trace.Tracing {
		defer trace.Call1(mklines.lines[0].Filename)()
	}

	mklines.Check()

	lines := mklines.mklines
	var mkline MkLine
	var logLine Line

	next := func() {
		if len(lines) == 0 {
			logLine = NewLineEOF(mkline.Filename)
			mkline = nil
		} else {
			mkline = lines[0]
			logLine = mkline.Line
			lines = lines[1:]
		}
	}
	next()

	skipWhile := func(pred func(MkLine) bool) {
		for mkline != nil && pred(mkline) {
			next()
		}
	}

	skipWhile(func(mkline MkLine) bool { return mkline.IsComment() || mkline.IsEmpty() })

	if !(mkline != nil && mkline.IsVarassign() && mkline.Varname() == "PKG_OPTIONS_VAR") {
		logLine.Warnf("Expected definition of PKG_OPTIONS_VAR.")
		return
	}
	next()

	declaredOptions := make(map[string]MkLine)
	handledOptions := make(map[string]MkLine)
	var optionsInDeclarationOrder []string

	// The conditionals are typically for OPSYS and MACHINE_ARCH.
loop:
	for mkline != nil {
		switch {
		case mkline.IsComment():
		case mkline.IsEmpty():
		case mkline.IsVarassign():
			varname := mkline.Varname()
			if varname == "PKG_SUPPORTED_OPTIONS" || hasPrefix(varname, "PKG_OPTIONS_GROUP.") {
				for _, option := range splitOnSpace(mkline.Value()) {
					declaredOptions[option] = mkline
					optionsInDeclarationOrder = append(optionsInDeclarationOrder, option)
				}
			}
		case mkline.IsCond():
		case mkline.IsInclude():
			includedFile := mkline.IncludeFile()
			switch {
			case matches(includedFile, `/[^/]+\.buildlink3\.mk$`):
			case matches(includedFile, `/[^/]+\.builtin\.mk$`):
			case includedFile == "../../mk/bsd.prefs.mk":
			case includedFile == "../../mk/bsd.fast.prefs.mk":
			case includedFile == "../../mk/bsd.options.mk":
				break loop
			}
		default:
			logLine.Warnf("Unexpected line type.")
		}

		next()
	}

	if !(mkline != nil && mkline.IsInclude() && mkline.IncludeFile() == "../../mk/bsd.options.mk") {
		logLine.Warnf("Expected inclusion of bsd.options.mk.")
		return
	}
	next()

	for mkline != nil {
		switch {
		case mkline.IsCond() && mkline.Directive() == "if":
			cond := NewMkParser(mkline.Line, mkline.Args(), false).MkCond()
			cond.Visit("empty", func(t *Tree) {
				varuse := t.args[0].(MkVarUse)
				if varuse.varname == "PKG_OPTIONS" && len(varuse.modifiers) == 1 && hasPrefix(varuse.modifiers[0], "M") {
					option := varuse.modifiers[0][1:]
					handledOptions[option] = mkline
				}
			})
		}
		next()
	}

	for _, option := range optionsInDeclarationOrder {
		declared := declaredOptions[option]
		handled := handledOptions[option]
		if declared != nil && handled == nil {
			declared.Warnf("Option %q should be handled below in an .if block.", option)
		}
		if declared == nil && handled != nil {
			handled.Warnf("Option %q is handled but not declared above.", option)
		}
	}

	SaveAutofixChanges(mklines.lines)
}
