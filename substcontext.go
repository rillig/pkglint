package pkglint

import "netbsd.org/pkglint/textproc"

// SubstContext records the state of a block of variable assignments
// that make up a SUBST class (see `mk/subst.mk`).
type SubstContext struct {
	id    string
	vars  map[string]bool
	conds []*substCond
}

func NewSubstContext() *SubstContext {
	return &SubstContext{conds: []*substCond{{seenElse: true}}}
}

type substCond struct {
	total    substSeen
	curr     substSeen
	seenElse bool
}

// substSeen contains all variables that depend on a particular SUBST
// class ID. These variables can be set in conditional branches, and
// pkglint keeps track whether they are set in all branches or only
// in some of them.
type substSeen struct {
	// The ID of the SUBST class is included here to track nested SUBST blocks.
	// It marks the conditional level at which a block has started.
	id bool

	stage     bool
	message   bool
	files     bool
	sed       bool
	vars      bool
	cmd       bool
	transform bool
}

func (st *substSeen) And(other substSeen) {
	st.id = st.id && other.id
	st.stage = st.stage && other.stage
	st.message = st.message && other.message
	st.files = st.files && other.files
	st.sed = st.sed && other.sed
	st.vars = st.vars && other.vars
	st.cmd = st.cmd && other.cmd
	st.transform = st.transform && other.transform
}

func (st *substSeen) Or(other substSeen) {
	st.id = st.id || other.id
	st.stage = st.stage || other.stage
	st.message = st.message || other.message
	st.files = st.files || other.files
	st.sed = st.sed || other.sed
	st.vars = st.vars || other.vars
	st.cmd = st.cmd || other.cmd
	st.transform = st.transform || other.transform
}

func (ctx *SubstContext) Process(mkline *MkLine) {
	switch {
	case mkline.IsEmpty():
		ctx.Finish(mkline)
	case mkline.IsVarassign():
		ctx.Varassign(mkline)
	case mkline.IsDirective():
		ctx.Directive(mkline)
	}
}

func (ctx *SubstContext) Varassign(mkline *MkLine) {
	if trace.Tracing {
		trace.Stepf("SubstContext.Varassign curr=%v", *ctx.top())
	}

	varname := mkline.Varname()
	varcanon := mkline.Varcanon()
	varparam := mkline.Varparam()
	op := mkline.Op()
	value := mkline.Value()

	if varcanon == "SUBST_CLASSES" || varcanon == "SUBST_CLASSES.*" {
		classes := mkline.ValueFields(value)
		if len(classes) > 1 {
			mkline.Warnf("Please add only one class at a time to SUBST_CLASSES.")
		}
		if ctx.id != "" && ctx.id != classes[0] {
			for len(ctx.conds) > 1 {
				// This will be confusing for the outer SUBST block,
				// but since that block is assumed to be finished,
				// this doesn't matter.
				ctx.directiveEndif(mkline)
			}
			complete := ctx.IsComplete()
			id := ctx.id
			ctx.Finish(mkline)
			if !complete {
				mkline.Warnf("Subst block %q should be finished before adding the next class to SUBST_CLASSES.", id)
			}
		}
		ctx.id = classes[0]
		ctx.top().id = true
		return
	}

	foreign := true
	switch varcanon {
	case
		"SUBST_STAGE.*",
		"SUBST_MESSAGE.*",
		"SUBST_FILES.*",
		"SUBST_SED.*",
		"SUBST_VARS.*",
		"SUBST_FILTER_CMD.*":
		foreign = false
	}

	if foreign && ctx.vars[varname] {
		foreign = false
	}

	if foreign {
		if ctx.id != "" {
			mkline.Warnf("Foreign variable %q in SUBST block.", varname)
		}
		return
	}

	if ctx.id == "" {
		mkline.Warnf("SUBST_CLASSES should come before the definition of %q.", varname)
		ctx.id = varparam
	}

	if hasPrefix(varname, "SUBST_") && varparam != ctx.id {
		if ctx.IsComplete() {
			// XXX: This code sometimes produces weird warnings. See
			// meta-pkgs/xorg/Makefile.common 1.41 for an example.
			ctx.Finish(mkline)

			// The following assignment prevents an additional warning,
			// but from a technically viewpoint, it is incorrect.
			ctx.id = varparam
		} else {
			mkline.Warnf("Variable %q does not match SUBST class %q.", varname, ctx.id)
			return
		}
	}

	switch varcanon {
	case "SUBST_STAGE.*":
		seen := func(s *substSeen) *bool { return &s.stage }
		ctx.dupString(mkline, seen, varname)
		if value == "pre-patch" || value == "post-patch" {
			fix := mkline.Autofix()
			fix.Warnf("Substitutions should not happen in the patch phase.")
			fix.Explain(
				"Performing substitutions during post-patch breaks tools such as",
				"mkpatches, making it very difficult to regenerate correct patches",
				"after making changes, and often leading to substituted string",
				"replacements being committed.",
				"",
				"Instead of pre-patch, use post-extract.",
				"Instead of post-patch, use pre-configure.")
			fix.Replace("pre-patch", "post-extract")
			fix.Replace("post-patch", "pre-configure")
			fix.Apply()
			// XXX: Add test that has "SUBST_STAGE.id=pre-patch # or rather post-patch?"
		}

		if G.Pkg != nil && (value == "pre-configure" || value == "post-configure") {
			if noConfigureLine := G.Pkg.vars.FirstDefinition("NO_CONFIGURE"); noConfigureLine != nil {
				mkline.Warnf("SUBST_STAGE %s has no effect when NO_CONFIGURE is set (in %s).",
					value, mkline.RelMkLine(noConfigureLine))
				mkline.Explain(
					"To fix this properly, remove the definition of NO_CONFIGURE.")
			}
		}

	case "SUBST_MESSAGE.*":
		seen := func(s *substSeen) *bool { return &s.message }
		ctx.dupString(mkline, seen, varname)

	case "SUBST_FILES.*":
		seen := func(s *substSeen) *bool { return &s.files }
		ctx.dupList(mkline, seen, varname, op)

	case "SUBST_SED.*":
		seen := func(s *substSeen) *bool { return &s.sed }
		ctx.dupList(mkline, seen, varname, op)
		ctx.top().transform = true

		ctx.suggestSubstVars(mkline)

	case "SUBST_VARS.*":
		seen := func(s *substSeen) *bool { return &s.vars }
		prev := ctx.seen(seen)
		ctx.dupList(mkline, seen, varname, op)
		ctx.top().transform = true
		for _, substVar := range mkline.Fields() {
			if ctx.vars == nil {
				ctx.vars = make(map[string]bool)
			}
			ctx.vars[substVar] = true
		}

		if prev && mkline.Op() == opAssign {
			before := mkline.ValueAlign()
			after := alignWith(mkline.Varname()+"+=", before)
			fix := mkline.Autofix()
			fix.Notef("Adjust.")
			fix.Replace(before, after)
			fix.Apply()
		}

	case "SUBST_FILTER_CMD.*":
		seen := func(s *substSeen) *bool { return &s.cmd }
		ctx.dupString(mkline, seen, varname)
		ctx.top().transform = true
	}
}

func (ctx *SubstContext) Directive(mkline *MkLine) {
	if trace.Tracing {
		trace.Stepf("+ SubstContext.Directive %v", *ctx.top())
	}

	dir := mkline.Directive()
	switch dir {
	case "if":
		top := substCond{total: substSeen{true, true, true, true, true, true, true, true}}
		ctx.conds = append(ctx.conds, &top)

	case "elif", "else":
		top := ctx.conds[len(ctx.conds)-1]
		top.total.And(top.curr)
		if top.curr.id {
			ctx.Finish(mkline)
		}
		top.curr = substSeen{}
		top.seenElse = dir == "else"

	case "endif":
		ctx.directiveEndif(mkline)
	}

	if trace.Tracing {
		trace.Stepf("- SubstContext.Directive %v", *ctx.top())
	}
}

func (ctx *SubstContext) directiveEndif(diag Diagnoser) {
	top := ctx.conds[len(ctx.conds)-1]
	top.total.And(top.curr)
	if top.curr.id {
		ctx.Finish(diag)
	}
	if !top.seenElse {
		top.total = substSeen{}
	}
	if len(ctx.conds) > 1 {
		ctx.conds = ctx.conds[:len(ctx.conds)-1]
	}
	ctx.top().Or(top.total)
}

func (ctx *SubstContext) IsComplete() bool {
	top := ctx.top()
	return top.stage && top.files && top.transform
}

func (ctx *SubstContext) Finish(diag Diagnoser) {
	if ctx.id == "" {
		return
	}

	id := ctx.id
	top := ctx.top()
	if !top.stage {
		diag.Warnf("Incomplete SUBST block: SUBST_STAGE.%s missing.", id)
	}
	if !top.files {
		diag.Warnf("Incomplete SUBST block: SUBST_FILES.%s missing.", id)
	}
	if !top.transform {
		diag.Warnf("Incomplete SUBST block: SUBST_SED.%[1]s, SUBST_VARS.%[1]s or SUBST_FILTER_CMD.%[1]s missing.", id)
	}

	*ctx = *NewSubstContext()
}

func (ctx *SubstContext) dupString(mkline *MkLine, flag func(stats *substSeen) *bool, varname string) {
	if ctx.seen(flag) {
		mkline.Warnf("Duplicate definition of %q.", varname)
	}
	*flag(ctx.top()) = true
}

func (ctx *SubstContext) dupList(mkline *MkLine, flag func(stats *substSeen) *bool,
	varname string, op MkOperator) {

	if ctx.seen(flag) && op != opAssignAppend {
		mkline.Warnf("All but the first %q lines should use the \"+=\" operator.", varname)
	}
	*flag(ctx.top()) = true
}

func (ctx *SubstContext) seen(flag func(seen *substSeen) *bool) bool {
	for _, cond := range ctx.conds {
		if *flag(&cond.curr) {
			return true
		}
	}
	return false
}

func (ctx *SubstContext) suggestSubstVars(mkline *MkLine) {

	tokens, _ := splitIntoShellTokens(mkline.Line, mkline.Value())
	for _, token := range tokens {
		varname := ctx.extractVarname(mkline.UnquoteShell(token, false))
		if varname == "" {
			continue
		}

		varop := sprintf("SUBST_VARS.%s%s%s",
			ctx.id,
			condStr(hasSuffix(ctx.id, "+"), " ", ""),
			condStr(ctx.top().vars, "+=", "="))

		fix := mkline.Autofix()
		fix.Notef("The substitution command %q can be replaced with \"%s %s\".",
			token, varop, varname)
		fix.Explain(
			"Replacing @VAR@ with ${VAR} is such a typical pattern that pkgsrc has built-in support for it,",
			"requiring only the variable name instead of the full sed command.")
		if !mkline.HasComment() && len(tokens) == 2 && tokens[0] == "-e" {
			fix.Replace(mkline.Text, alignWith(varop, mkline.ValueAlign())+varname)
		}
		fix.Apply()

		ctx.top().vars = true
	}
}

// extractVarname extracts the variable name from a sed command of the form
// s,@VARNAME@,${VARNAME}, and some related variants thereof.
func (*SubstContext) extractVarname(token string) string {
	parser := NewMkLexer(token, nil)
	lexer := parser.lexer
	if !lexer.SkipByte('s') {
		return ""
	}

	separator := lexer.NextByteSet(textproc.XPrint) // Really any character works
	if separator == -1 {
		return ""
	}

	if !lexer.SkipByte('@') {
		return ""
	}

	varname := parser.Varname()
	if !lexer.SkipByte('@') || !lexer.SkipByte(byte(separator)) {
		return ""
	}

	varuse := parser.VarUse()
	if varuse == nil || varuse.varname != varname {
		return ""
	}

	switch varuse.Mod() {
	case "", ":Q":
		break
	default:
		return ""
	}

	if !lexer.SkipByte(byte(separator)) {
		return ""
	}

	return varname
}

func (ctx *SubstContext) top() *substSeen {
	return &ctx.conds[len(ctx.conds)-1].curr
}
