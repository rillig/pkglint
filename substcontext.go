package pkglint

import "netbsd.org/pkglint/textproc"

// SubstContext records the state of a block of variable assignments
// that make up a SUBST class (see `mk/subst.mk`).
type SubstContext struct {
	id        string
	stage     string
	message   string
	filterCmd string
	vars      map[string]bool
	conds     []*SubstCond
}

func NewSubstContext() *SubstContext {
	return &SubstContext{conds: []*SubstCond{{SeenElse: true}}}
}

type SubstCond struct {
	Total    SubstContextStats
	Curr     SubstContextStats
	SeenElse bool
}

type SubstContextStats struct {
	seenFiles     bool
	seenSed       bool
	seenVars      bool
	seenTransform bool
}

func (st *SubstContextStats) SeenFiles() *bool     { return &st.seenFiles }
func (st *SubstContextStats) SeenSed() *bool       { return &st.seenSed }
func (st *SubstContextStats) SeenVars() *bool      { return &st.seenVars }
func (st *SubstContextStats) SeenTransform() *bool { return &st.seenTransform }

func (st *SubstContextStats) And(other SubstContextStats) {
	st.seenFiles = st.seenFiles && other.seenFiles
	st.seenSed = st.seenSed && other.seenSed
	st.seenVars = st.seenVars && other.seenVars
	st.seenTransform = st.seenTransform && other.seenTransform
}

func (st *SubstContextStats) Or(other SubstContextStats) {
	st.seenFiles = st.seenFiles || other.seenFiles
	st.seenSed = st.seenSed || other.seenSed
	st.seenVars = st.seenVars || other.seenVars
	st.seenTransform = st.seenTransform || other.seenTransform
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
			complete := ctx.IsComplete()
			id := ctx.id
			ctx.Finish(mkline)
			if !complete {
				mkline.Warnf("Subst block %q should be finished before adding the next class to SUBST_CLASSES.", id)
			}
		}
		ctx.id = classes[0]
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
		ctx.dupString(mkline, &ctx.stage, varname, value) // TODO: Make stage conditional as well.
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
		ctx.dupString(mkline, &ctx.message, varname, value)

	case "SUBST_FILES.*":
		ctx.dupBool(mkline, (*SubstContextStats).SeenFiles, varname, op)

	case "SUBST_SED.*":
		ctx.dupBool(mkline, (*SubstContextStats).SeenSed, varname, op)
		ctx.top().seenTransform = true

		ctx.suggestSubstVars(mkline)

	case "SUBST_VARS.*":
		ctx.dupBool(mkline, (*SubstContextStats).SeenVars, varname, op)
		ctx.top().seenTransform = true
		for _, substVar := range mkline.Fields() {
			if ctx.vars == nil {
				ctx.vars = make(map[string]bool)
			}
			ctx.vars[substVar] = true
		}

	case "SUBST_FILTER_CMD.*":
		ctx.dupString(mkline, &ctx.filterCmd, varname, value)
		ctx.top().seenTransform = true
	}
}

func (ctx *SubstContext) Directive(mkline *MkLine) {
	if ctx.id == "" {
		return
	}

	if trace.Tracing {
		trace.Stepf("+ SubstContext.Directive %v", *ctx.top())
	}

	dir := mkline.Directive()
	switch dir {
	case "if":
		top := SubstCond{Total: SubstContextStats{true, true, true, true}}
		ctx.conds = append(ctx.conds, &top)

	case "elif", "else":
		top := ctx.conds[len(ctx.conds)-1]
		top.Total.And(top.Curr)
		top.Curr = SubstContextStats{}
		top.SeenElse = dir == "else"

	case "endif":
		top := ctx.conds[len(ctx.conds)-1]
		top.Total.And(top.Curr)
		if !top.SeenElse {
			top.Total = SubstContextStats{}
		}
		if len(ctx.conds) > 1 {
			ctx.conds = ctx.conds[:len(ctx.conds)-1]
		}
		ctx.top().Or(top.Total)
	}

	if trace.Tracing {
		trace.Stepf("- SubstContext.Directive %v", *ctx.top())
	}
}

func (ctx *SubstContext) IsComplete() bool {
	return ctx.stage != "" && ctx.top().seenFiles && ctx.top().seenTransform
}

func (ctx *SubstContext) Finish(mkline *MkLine) {
	if ctx.id == "" {
		return
	}

	id := ctx.id
	if ctx.stage == "" {
		mkline.Warnf("Incomplete SUBST block: SUBST_STAGE.%s missing.", id)
	}
	if !ctx.top().seenFiles {
		mkline.Warnf("Incomplete SUBST block: SUBST_FILES.%s missing.", id)
	}
	if !ctx.top().seenTransform {
		mkline.Warnf("Incomplete SUBST block: SUBST_SED.%[1]s, SUBST_VARS.%[1]s or SUBST_FILTER_CMD.%[1]s missing.", id)
	}

	*ctx = *NewSubstContext()
}

func (*SubstContext) dupString(mkline *MkLine, pstr *string, varname, value string) {
	if *pstr != "" {
		mkline.Warnf("Duplicate definition of %q.", varname)
	}
	*pstr = value
}

func (ctx *SubstContext) dupBool(mkline *MkLine, flag func(stats *SubstContextStats) *bool,
	varname string, op MkOperator) {

	seen := false
	for _, cond := range ctx.conds {
		seen = seen || *flag(&cond.Curr)
	}

	if seen && op != opAssignAppend {
		mkline.Warnf("All but the first %q lines should use the \"+=\" operator.", varname)
	}
	*flag(ctx.top()) = true
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
			condStr(ctx.top().seenVars, "+=", "="))

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

		ctx.top().seenVars = true
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

func (ctx *SubstContext) top() *SubstContextStats {
	return &ctx.conds[len(ctx.conds)-1].Curr
}
