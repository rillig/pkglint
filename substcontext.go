package main

// Records the state of a block of variable assignments that make up a SUBST
// class (see mk/subst.mk).
type SubstContext struct {
	id        string
	stage     string
	message   string
	files     []string
	sed       []string
	vars      []string
	filterCmd string
}

func (ctx *SubstContext) isComplete() bool {
	return ctx.id != "" &&
		len(ctx.files) != 0 &&
		(len(ctx.sed) != 0 || len(ctx.vars) != 0 || ctx.filterCmd != "")
}

func (self *SubstContext) checkVarassign(line *Line, varname, op, value string) {
	if !G.opts.optWarnExtra {
		return
	}

	if varname == "SUBST_CLASSES" {
		classes := splitOnSpace(value)
		if len(classes) > 1 {
			line.warnf("Please add only one class at a time to SUBST_CLASSES.")
		}
		if self.id != "" {
			line.warnf("SUBST_CLASSES should only appear once in a SUBST block.")
		}
		self.id = classes[0]
		return
	}

	m, varbase, varparam := match2(varname, `^(SUBST_(?:STAGE|MESSAGE|FILES|SED|VARS|FILTER_CMD))\.([\-\w_]+)$`)
	if !m {
		if self.id != "" {
			line.warnf("Foreign variable %q in SUBST block.", varname)
		}
		return
	}

	if self.id == "" {
		line.warnf("SUBST_CLASSES should come before the definition of %q.", varname)
		self.id = varparam
	}

	if self.id != "" && varparam != self.id {
		if self.isComplete() {
			// XXX: This code sometimes produces weird warnings. See
			// meta-pkgs/xorg/Makefile.common 1.41 for an example.
			self.finish(line)

			// The following assignment prevents an additional warning,
			// but from a technically viewpoint, it is incorrect.
			self.id = varparam
		} else {
			line.warnf("Variable parameter %q does not match SUBST class %q.", varparam, self.id)
		}
		return
	}

	switch varbase {
	case "SUBST_STAGE":
		if self.stage != "" {
			line.warnf("Duplicate definition of %q.", varname)
		}
		self.stage = value
	case "SUBST_MESSAGE":
		if self.message != "" {
			line.warnf("Duplicate definition of %q.", varname)
		}
		self.message = value
	case "SUBST_FILES":
		if len(self.files) > 0 && op != "+=" {
			line.warnf("All but the first SUBST_FILES line should use the \"+=\" operator.")
		}
		self.files = append(self.files, value)
	case "SUBST_SED":
		if len(self.sed) > 0 && op != "+=" {
			line.warnf("All but the first SUBST_SED line should use the \"+=\" operator.")
		}
		self.sed = append(self.sed, value)
	case "SUBST_FILTER_CMD":
		if self.filterCmd != "" {
			line.warnf("Duplicate definition of %q.", varname)
		}
		self.filterCmd = value
	case "SUBST_VARS":
		if len(self.vars) > 0 && op != "+=" {
			line.warnf("All but the first SUBST_VARS line should use the \"+=\" operator.")
		}
		self.vars = append(self.vars, value)
	default:
		line.warnf("Foreign variable in SUBST block.")
	}
}

func (self *SubstContext) finish(line *Line) {
	if self.id == "" || !G.opts.optWarnExtra {
		return
	}
	if self.id == "" {
		line.warnf("Incomplete SUBST block: SUBST_CLASSES missing.")
	}
	if self.stage == "" {
		line.warnf("Incomplete SUBST block: SUBST_STAGE missing.")
	}
	if len(self.files) == 0 {
		line.warnf("Incomplete SUBST block: SUBST_FILES missing.")
	}
	if len(self.sed) == 0 && len(self.vars) == 0 && self.filterCmd == "" {
		line.warnf("Incomplete SUBST block: SUBST_SED, SUBST_VARS or SUBST_FILTER_CMD missing.")
	}
	self.id = ""
	self.stage = ""
	self.message = ""
	self.files = nil
	self.sed = nil
	self.vars = nil
	self.filterCmd = ""
}
