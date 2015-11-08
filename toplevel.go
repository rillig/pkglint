package main

type Toplevel struct {
	previousSubdir string
	subdirs        []string
}

func checkdirToplevel() {
	trace("checkdirToplevel", G.currentDir)

	ctx := &Toplevel{}
	ctx.subdirs = make([]string, 0)

	fname := G.currentDir + "/Makefile"

	lines, err := loadLines(fname, true)
	if err != nil {
		logError(fname, NO_LINES, "Cannot be read.")
		return
	}

	parselinesMk(lines)
	if 0 < len(lines) {
		checklineRcsid(lines[0], `#\s+`, "# ")
	}

	for _, line := range lines {
		if m, commentedOut, indentation, subdir, comment := match4(line.text, `^(#?)SUBDIR\s*\+=(\s*)(\S+)\s*(?:#\s*(.*?)\s*|)$`); m {
			ctx.checkSubdir(line, commentedOut == "#", indentation, subdir, comment)
		}
	}

	checklinesMk(lines)

	if G.opts.optRecursive {
		G.ipcCheckingRootRecursively = true
		G.todo = append(G.todo, ctx.subdirs...)
	}
}

func (ctx *Toplevel) checkSubdir(line *Line, commentedOut bool, indentation, subdir, comment string) {
	if commentedOut && comment == "" {
		line.logWarning("%s commented out without giving a reason.", subdir)
	}

	if indentation != "\t" {
		line.logWarning("Indentation should be a single tab character.")
	}

	if contains(subdir, "$") || !fileExists(G.currentDir+"/"+subdir+"/Makefile") {
		return
	}

	prev := ctx.previousSubdir
	switch {
	case subdir > prev:
		// Correctly ordered
	case subdir == prev:
		line.logError("Each subdir must only appear once.")
	case subdir == "archivers" && prev == "x11":
		// This exception is documented in the top-level Makefile.
	default:
		line.logWarning("%s should come before %s", subdir, prev)
	}
	ctx.previousSubdir = subdir

	if !commentedOut {
		ctx.subdirs = append(ctx.subdirs, G.currentDir+"/"+subdir)
	}
}
