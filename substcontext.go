package pkglint

import "netbsd.org/pkglint/textproc"

// SubstContext records the state of a block of variable assignments
// that make up a SUBST class (see `mk/subst.mk`).
type SubstContext struct {
	queuedIds map[string]bool
	doneIds   map[string]bool
	id        string
	vars      map[string]bool
	conds     []*substCond

	once Once
}

func NewSubstContext() *SubstContext {
	ctx := SubstContext{}
	ctx.reset()
	return &ctx
}

func (ctx *SubstContext) reset() {
	ctx.id = ""
	ctx.vars = nil
	ctx.conds = []*substCond{{seenElse: true}}
}

type substCond struct {
	// Tells whether a SUBST block has started at this conditional level.
	// All variable assignments that belong to this class must happen at
	// this conditional level or below it.
	top bool

	// Collects the parts of the SUBST block that have been defined in all
	// branches that have been parsed completely.
	total substSeen

	// Collects the parts of the SUBST block that are defined in the current
	// branch of the conditional. At the end of the branch, they are merged
	// into the total.
	curr substSeen

	// Marks whether the current conditional statement has
	// an .else branch. If it doesn't, this means that all variables
	// are potentially unset in that branch.
	seenElse bool
}

// substSeen contains all variables that depend on a particular SUBST
// class ID. These variables can be set in conditional branches, and
// pkglint keeps track whether they are set in all branches or only
// in some of them.
type substSeen struct {
	stage     bool
	message   bool
	files     bool
	sed       bool
	vars      bool
	cmd       bool
	transform bool
}

func (st *substSeen) And(other substSeen) {
	st.stage = st.stage && other.stage
	st.message = st.message && other.message
	st.files = st.files && other.files
	st.sed = st.sed && other.sed
	st.vars = st.vars && other.vars
	st.cmd = st.cmd && other.cmd
	st.transform = st.transform && other.transform
}

func (st *substSeen) Or(other substSeen) {
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
		trace.Stepf("SubstContext.Varassign curr=%v", *ctx.seen())
	}

	varcanon := mkline.Varcanon()
	if varcanon == "SUBST_CLASSES" || varcanon == "SUBST_CLASSES.*" {
		ctx.varassignClasses(mkline)
		return
	}

	if ctx.isForeign(mkline.Varcanon()) && !ctx.isSubstVar(mkline.Varname()) {
		if ctx.isActive() {
			mkline.Warnf("Foreign variable %q in SUBST block.", mkline.Varname())
		}
		return
	}

	// TODO: Move before the previous if clause.
	if !ctx.isActive() {
		ctx.varassignOutOfScope(mkline)
		return
	}

	if hasPrefix(mkline.Varname(), "SUBST_") && !ctx.isActiveId(mkline.Varparam()) {
		if !ctx.varassignDifferentClass(mkline) {
			return
		}
	}

	switch varcanon {
	case "SUBST_STAGE.*":
		ctx.varassignStage(mkline)
	case "SUBST_MESSAGE.*":
		ctx.varassignMessages(mkline)
	case "SUBST_FILES.*":
		ctx.varassignFiles(mkline)
	case "SUBST_SED.*":
		ctx.varassignSed(mkline)
	case "SUBST_VARS.*":
		ctx.varassignVars(mkline)
	case "SUBST_FILTER_CMD.*":
		ctx.varassignFilterCmd(mkline)
	}
}

func (ctx *SubstContext) varassignClasses(mkline *MkLine) {
	classes := mkline.ValueFields(mkline.Value())

	if len(classes) > 1 {
		mkline.Notef("Please add only one class at a time to SUBST_CLASSES.")
		mkline.Explain(
			"This way, each substitution class forms a block in the package Makefile,",
			"and to delete this block, it is not necessary to look anywhere else.")
		for _, class := range classes {
			ctx.queue(class)
		}
	}

	id := classes[0]
	if ctx.isActive() && !ctx.isActiveId(id) {
		id := ctx.activeId() // since ctx.condEndif may reset it

		for ctx.isConditional() {
			// This will be confusing for the outer SUBST block,
			// but since that block is assumed to be finished,
			// this doesn't matter.
			ctx.condEndif(mkline)
		}

		complete := ctx.IsComplete() // since ctx.Finish will reset it
		ctx.Finish(mkline)
		if !complete {
			mkline.Warnf("Subst block %q should be finished before adding the next class to SUBST_CLASSES.", id)
		}
	}

	ctx.setActiveId(id)

	return
}

func (ctx *SubstContext) varassignOutOfScope(mkline *MkLine) {
	varparam := mkline.Varparam()

	if ctx.isListCanon(mkline.Varcanon()) && ctx.isDone(varparam) {
		if mkline.Op() != opAssignAppend {
			mkline.Warnf("Late additions to a SUBST variable should use the += operator.")
		}
		return
	}
	if containsWord(mkline.Rationale(), varparam) {
		return
	}

	if ctx.start(varparam) {
		return
	}

	if ctx.once.FirstTimeSlice("SubstContext.Varassign", varparam) {
		mkline.Warnf("Before defining %s, the SUBST class "+
			"should be declared using \"SUBST_CLASSES+= %s\".",
			mkline.Varname(), varparam)
	}
	return
}

func (ctx *SubstContext) varassignDifferentClass(mkline *MkLine) (ok bool) {
	varname := mkline.Varname()
	varparam := mkline.Varparam()

	if !ctx.IsComplete() {
		mkline.Warnf("Variable %q does not match SUBST class %q.", varname, ctx.activeId())
		return false
	}

	ctx.Finish(mkline)

	ctx.start(varparam)
	return true
}

func (ctx *SubstContext) varassignStage(mkline *MkLine) {
	if ctx.isConditional() {
		mkline.Warnf("%s should not be defined conditionally.", mkline.Varname())
	}

	accessor := func(s *substSeen) *bool { return &s.stage }
	ctx.dupString(mkline, accessor)

	value := mkline.Value()
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
	}

	if G.Pkg != nil && (value == "pre-configure" || value == "post-configure") {
		if noConfigureLine := G.Pkg.vars.FirstDefinition("NO_CONFIGURE"); noConfigureLine != nil {
			mkline.Warnf("SUBST_STAGE %s has no effect when NO_CONFIGURE is set (in %s).",
				value, mkline.RelMkLine(noConfigureLine))
			mkline.Explain(
				"To fix this properly, remove the definition of NO_CONFIGURE.")
		}
	}
}

func (ctx *SubstContext) varassignMessages(mkline *MkLine) {
	varname := mkline.Varname()

	if ctx.isConditional() {
		mkline.Warnf("%s should not be defined conditionally.", varname)
	}

	accessor := func(s *substSeen) *bool { return &s.message }
	ctx.dupString(mkline, accessor)
}

func (ctx *SubstContext) varassignFiles(mkline *MkLine) {
	accessor := func(s *substSeen) *bool { return &s.files }
	ctx.dupList(mkline, accessor)
}

func (ctx *SubstContext) varassignSed(mkline *MkLine) {
	accessor := func(s *substSeen) *bool { return &s.sed }
	ctx.dupList(mkline, accessor)
	ctx.seen().transform = true

	ctx.suggestSubstVars(mkline)
}

func (ctx *SubstContext) varassignVars(mkline *MkLine) {
	accessor := func(s *substSeen) *bool { return &s.vars }
	seen := ctx.seenInBranch(accessor) // since ctx.dupList modifies it
	ctx.dupList(mkline, accessor)
	ctx.seen().transform = true

	for _, substVar := range mkline.Fields() {
		// TODO: What about variables that are defined before the SUBST_VARS line?
		ctx.allowVar(substVar)
	}

	if seen && mkline.Op() == opAssign {
		before := mkline.ValueAlign()
		after := alignWith(mkline.Varname()+"+=", before)

		fix := mkline.Autofix()
		fix.Notef("All but the first assignment should use the += operator.")
		fix.Replace(before, after)
		fix.Apply()
	}
}

func (ctx *SubstContext) varassignFilterCmd(mkline *MkLine) {
	accessor := func(s *substSeen) *bool { return &s.cmd }
	ctx.dupString(mkline, accessor)
	ctx.seen().transform = true
}

func (ctx *SubstContext) isForeign(varcanon string) bool {
	switch varcanon {
	case
		"SUBST_STAGE.*",
		"SUBST_MESSAGE.*",
		"SUBST_FILES.*",
		"SUBST_SED.*",
		"SUBST_VARS.*",
		"SUBST_FILTER_CMD.*":
		return false
	}
	return true
}

func (ctx *SubstContext) isListCanon(varcanon string) bool {
	switch varcanon {
	case
		"SUBST_FILES.*",
		"SUBST_SED.*",
		"SUBST_VARS.*":
		return true
	}
	return false
}

func (ctx *SubstContext) Directive(mkline *MkLine) {
	if trace.Tracing {
		trace.Stepf("+ SubstContext.Directive %v", *ctx.seen())
	}

	dir := mkline.Directive()
	switch dir {
	case "if":
		ctx.condIf()

	case "elif", "else":
		ctx.condElse(mkline, dir)

	case "endif":
		ctx.condEndif(mkline)
	}

	if trace.Tracing {
		trace.Stepf("- SubstContext.Directive %v", *ctx.seen())
	}
}

func (ctx *SubstContext) IsComplete() bool {
	seen := ctx.seen()
	return seen.stage && seen.files && seen.transform
}

func (ctx *SubstContext) Finish(diag Diagnoser) {
	if !ctx.isActive() {
		return
	}

	// TODO: Extract these warnings into a separate method,
	//  to decouple them from the state manipulation.
	id := ctx.activeId()
	seen := ctx.seen()
	if !seen.stage {
		diag.Warnf("Incomplete SUBST block: SUBST_STAGE.%s missing.", id)
	}
	if !seen.files {
		diag.Warnf("Incomplete SUBST block: SUBST_FILES.%s missing.", id)
	}
	if !seen.transform {
		diag.Warnf("Incomplete SUBST block: SUBST_SED.%[1]s, SUBST_VARS.%[1]s or SUBST_FILTER_CMD.%[1]s missing.", id)
	}

	ctx.reset()
}

func (ctx *SubstContext) dupString(mkline *MkLine, flag func(stats *substSeen) *bool) {
	if ctx.seenInBranch(flag) {
		mkline.Warnf("Duplicate definition of %q.", mkline.Varname())
	}
	*flag(ctx.seen()) = true
}

func (ctx *SubstContext) dupList(mkline *MkLine, flag func(stats *substSeen) *bool) {

	if ctx.seenInBranch(flag) && mkline.Op() != opAssignAppend {
		mkline.Warnf("All but the first %q lines should use the \"+=\" operator.", mkline.Varname())
	}
	*flag(ctx.seen()) = true
}

func (ctx *SubstContext) suggestSubstVars(mkline *MkLine) {

	tokens, _ := splitIntoShellTokens(mkline.Line, mkline.Value())
	for _, token := range tokens {
		varname := ctx.extractVarname(mkline.UnquoteShell(token, false))
		if varname == "" {
			continue
		}

		id := ctx.activeId()
		varop := sprintf("SUBST_VARS.%s%s%s",
			id,
			condStr(hasSuffix(id, "+"), " ", ""),
			// FIXME: ctx.anyVars sounds more correct.
			condStr(ctx.seen().vars, "+=", "="))

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

		// At this point the number of SUBST_SED assignments is one
		// less than before. Therefore it is possible to adjust the
		// assignment operators on them. It's probably not worth the
		// effort, though.

		ctx.seen().vars = true
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

func (ctx *SubstContext) isActive() bool { return ctx.id != "" }

func (ctx *SubstContext) isActiveId(id string) bool { return ctx.id == id }

func (ctx *SubstContext) activeId() string {
	assert(ctx.isActive())
	return ctx.id
}

func (ctx *SubstContext) setActiveId(id string) {
	ctx.id = id
	ctx.cond().top = true
	ctx.markAsDone(id)
}

func (ctx *SubstContext) queue(id string) {
	if ctx.queuedIds == nil {
		ctx.queuedIds = map[string]bool{}
	}
	ctx.queuedIds[id] = true
}

func (ctx *SubstContext) start(id string) bool {
	if ctx.queuedIds[id] {
		ctx.queuedIds[id] = false
		ctx.setActiveId(id)
		return true
	}
	return false
}

func (ctx *SubstContext) markAsDone(id string) {
	if ctx.doneIds == nil {
		ctx.doneIds = map[string]bool{}
	}
	ctx.doneIds[id] = true
}

func (ctx *SubstContext) isDone(varparam string) bool {
	return ctx.doneIds[varparam]
}

func (ctx *SubstContext) allowVar(substVar string) {
	if ctx.vars == nil {
		ctx.vars = make(map[string]bool)
	}
	ctx.vars[substVar] = true
}

func (ctx *SubstContext) isSubstVar(varname string) bool {
	return ctx.vars[varname]
}

// isConditional returns whether the current line is at a deeper conditional
// level than the corresponding start of the SUBST block.
//
// The start of the block is most often the SUBST_CLASSES line.
// When more than one class is added to SUBST_CLASSES in a single line,
// the start of the block is the first variable assignment that uses the
// corresponding class ID.
//
// TODO: Adjust the implementation to this description.
func (ctx *SubstContext) isConditional() bool {
	return !ctx.cond().top
}

// cond returns information about the current branch of conditionals.
func (ctx *SubstContext) cond() *substCond {
	return ctx.conds[len(ctx.conds)-1]
}

// cond returns information about the parts of the SUBST block that
// have already been seen in the current leaf branch of the conditionals.
func (ctx *SubstContext) seen() *substSeen {
	return &ctx.cond().curr
}

func (ctx *SubstContext) condIf() {
	top := substCond{total: substSeen{true, true, true, true, true, true, true}}
	ctx.conds = append(ctx.conds, &top)
}

func (ctx *SubstContext) condElse(mkline *MkLine, dir string) {
	top := ctx.cond()
	top.total.And(top.curr)
	if !ctx.isConditional() {
		// XXX: This is a higher-level method
		ctx.Finish(mkline)
	}
	top.curr = substSeen{}
	top.seenElse = dir == "else"
}

func (ctx *SubstContext) condEndif(diag Diagnoser) {
	top := ctx.cond()
	top.total.And(top.curr)
	if !ctx.isConditional() {
		// XXX: This is a higher-level method
		ctx.Finish(diag)
	}
	if !top.seenElse {
		top.total = substSeen{}
	}
	if len(ctx.conds) > 1 {
		ctx.conds = ctx.conds[:len(ctx.conds)-1]
	}
	ctx.seen().Or(top.total)
}

// Returns true if the given flag from substSeen has been seen
// somewhere in the conditional path of the current line.
func (ctx *SubstContext) seenInBranch(flag func(seen *substSeen) *bool) bool {
	for _, cond := range ctx.conds {
		if *flag(&cond.curr) {
			return true
		}
	}
	return false
}
