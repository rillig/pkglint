package pkglint

// MkWalker walks through a makefile line or a text snippet from such a line,
// visiting the expressions and their subexpressions.
type MkWalker struct {
	Diag Autofixer
	Expr func(expr *MkExpr, time EctxTime)
}

func NewMkWalker(diag Autofixer, expr func(expr *MkExpr, time EctxTime)) *MkWalker {
	return &MkWalker{diag, expr}
}

// WalkLine calls the action for each variable that is used in the line.
func (w *MkWalker) WalkLine(mkline *MkLine) {
	switch {

	case mkline.IsVarassign():
		w.WalkText(mkline.Varname(), EctxLoadTime)
		w.WalkText(mkline.Value(), mkline.Op().Time())

	case mkline.IsDirective():
		w.walkDirective(mkline)

	case mkline.IsShellCommand():
		w.WalkText(mkline.ShellCommand(), EctxRunTime)

	case mkline.IsDependency():
		w.WalkText(mkline.Targets(), EctxLoadTime)
		w.WalkText(mkline.Sources(), EctxLoadTime)

	case mkline.IsInclude():
		w.WalkText(mkline.IncludedFile().String(), EctxLoadTime)
	}
}

func (w *MkWalker) WalkText(text string, time EctxTime) {
	if !contains(text, "$") {
		return
	}

	tokens, _ := NewMkLexer(text, nil).MkTokens()
	for _, token := range tokens {
		if token.Expr != nil {
			w.walkExpr(token.Expr, time)
		}
	}
}

func (w *MkWalker) walkDirective(mkline *MkLine) {
	switch mkline.Directive() {
	case "error", "for", "info", "warning":
		w.WalkText(mkline.Args(), EctxLoadTime)
	case "if", "elif":
		if cond := mkline.Cond(); cond != nil {
			cond.Walk(&MkCondCallback{
				Expr: func(expr *MkExpr) {
					w.walkExpr(expr, EctxLoadTime)
				}})
		}
	}
}

func (w *MkWalker) walkExpr(expr *MkExpr, time EctxTime) {
	varname := expr.varname
	if !expr.IsExpression() {
		w.Expr(expr, time)
	}
	w.WalkText(varname, time)
	for _, mod := range expr.modifiers {
		w.walkModifier(mod, time)
	}
}

func (w *MkWalker) walkModifier(mod MkExprModifier, time EctxTime) {
	if !contains(mod.String(), "$") {
		return
	}
	if mod.HasPrefix("@") {
		// XXX: Probably close enough for most practical cases.
		// If not, implement bmake's ParseModifierPartBalanced.
		w.WalkText(mod.String(), time)
		return
	}
	if mod.HasPrefix("?") || mod.HasPrefix("D") || mod.HasPrefix("U") || mod.HasPrefix("!") {
		// XXX: Probably close enough, but \$ and $$ differ.
		w.WalkText(mod.String(), time)
		return
	}
	if ok, _, from, to, _ := mod.MatchSubst(); ok {
		// XXX: Probably close enough, but \$ and $$ differ.
		w.WalkText(from, time)
		// XXX: Probably close enough, but \$ and $$ differ.
		w.WalkText(to, time)
		return
	}
	if ok, _, pattern, _ := mod.MatchMatch(); ok {
		// XXX: Probably close enough, but \$ and $$ differ.
		w.WalkText(pattern, time)
		return
	}
	// XXX: Assume that all other modifiers behave similarly to each other.
	// See the ApplyModifier_* functions in bmake's var.c for details.
	w.WalkText(mod.String(), time)
}
