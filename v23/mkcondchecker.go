package pkglint

import (
	"github.com/rillig/pkglint/v23/makepat"
	"github.com/rillig/pkglint/v23/textproc"
)

// MkCondChecker checks conditions in Makefiles.
// These conditions occur in .if and .elif clauses, as well as the
// :? modifier.
type MkCondChecker struct {
	MkLine  *MkLine
	MkLines *MkLines
}

func NewMkCondChecker(mkLine *MkLine, mkLines *MkLines) *MkCondChecker {
	return &MkCondChecker{MkLine: mkLine, MkLines: mkLines}
}

func (ck *MkCondChecker) Check() {
	mkline := ck.MkLine
	if trace.Tracing {
		defer trace.Call1(mkline.Args())()
	}

	p := NewMkParser(nil, mkline.Args()) // No emitWarnings here, see the code below.
	cond := p.MkCond()
	if !p.EOF() || cond == nil {
		mkline.Warnf("Invalid condition, unrecognized part: %q.", p.Rest())
		return
	}

	ck.checkRedundantParentheses(cond)

	checkExpr := func(expr *MkExpr) {
		var vartype *Vartype // TODO: Insert a better type guess here.
		// See Test_MkExprChecker_checkAssignable__shell_command_in_exists.
		ectx := ExprContext{vartype, EctxLoadTime, EctxQuotPlain, false}
		NewMkExprChecker(expr, ck.MkLines, mkline).Check(&ectx)
	}

	// Skip subconditions that have already been handled as part of the !(...).
	done := make(map[interface{}]bool)

	checkNot := func(not *MkCond) {
		empty := not.Empty
		if empty != nil {
			ck.checkNotEmpty(not)
			ck.checkEmpty(empty, true, true)
			done[empty] = true
		}

		if not.Term != nil && not.Term.Expr != nil {
			expr := not.Term.Expr
			ck.checkEmpty(expr, false, false)
			done[expr] = true
		}

		ck.checkNotCompare(not)
	}

	checkEmpty := func(empty *MkExpr) {
		if !done[empty] {
			ck.checkEmpty(empty, true, false)
		}
	}

	checkVar := func(expr *MkExpr) {
		if !done[expr] {
			ck.checkEmpty(expr, false, true)
		}
	}

	cond.Walk(&MkCondCallback{
		And:     ck.checkAnd,
		Not:     checkNot,
		Empty:   checkEmpty,
		Var:     checkVar,
		Compare: ck.checkCompare,
		Expr:    checkExpr})

	ck.checkContradictions()
}

func (ck *MkCondChecker) checkRedundantParentheses(cond *MkCond) {
	if cond.Paren != nil {
		fix := ck.MkLine.Autofix()
		fix.Notef("Parentheses around the outermost condition are redundant.")
		fix.Explain(
			"BSD make does not require parentheses around conditions.",
			"In this regard, it works like the C preprocessor.")
		fix.Apply()
	}
}

func (ck *MkCondChecker) checkAnd(conds []*MkCond) {
	if len(conds) == 2 &&
		conds[0].Defined != "" &&
		conds[1].Not != nil &&
		conds[1].Not.Empty != nil &&
		conds[0].Defined == conds[1].Not.Empty.varname {
		fix := ck.MkLine.Autofix()
		fix.Notef("Checking \"defined\" before \"!empty\" is redundant.")
		fix.Explain(
			"The \"empty\" function treats an undefined variable",
			"as empty, so there is no need to write the \"defined\"",
			"explicitly.")
		fix.Replace("defined("+conds[0].Defined+") && ", "")
		fix.Apply()
	}
}

func (ck *MkCondChecker) checkNotEmpty(not *MkCond) {
	// Consider suggesting ${VAR} instead of !empty(VAR) since it is
	// shorter and avoids unnecessary negation, which makes the
	// expression less confusing.
	//
	// This applies especially to the ${VAR:Mpattern} form.
	//
	// See MkCondSimplifier.
	if !hasPrefix(not.Empty.varname, "PKG_BUILD_OPTIONS.") {
		return
	}

	fix := ck.MkLine.Autofix()
	from := sprintf("!empty(%s%s)", not.Empty.varname, not.Empty.Mod())
	to := not.Empty.String()
	fix.Notef("%s can be replaced with %s.", from, to)
	fix.Explain(
		"Besides being simpler to read, the expression will also fail",
		"quickly with a \"Malformed conditional\" error from bmake",
		"if it should ever be undefined at this point.",
		"This catches typos and other programming mistakes.")
	fix.Replace(from, to)
	fix.Apply()
}

// checkEmpty checks a condition of the form empty(VAR),
// empty(VAR:Mpattern) or ${VAR:Mpattern} in an .if directive.
func (ck *MkCondChecker) checkEmpty(expr *MkExpr, fromEmpty bool, neg bool) {
	ck.checkEmptyExpr(expr, neg)
	ck.checkEmptyType(expr)

	s := MkCondSimplifier{ck.MkLines, ck.MkLine}
	s.SimplifyExpr(expr, fromEmpty, neg)
}

// checkEmptyExpr warns about 'empty(${VARNAME:Mpattern})', which should be
// 'empty(VARNAME:Mpattern)' instead, without the '${...}'.
func (ck *MkCondChecker) checkEmptyExpr(expr *MkExpr, neg bool) {
	if !matches(expr.varname, `^\$.*:[MN]`) {
		return
	}

	ck.MkLine.Warnf("The empty() function takes a variable name as parameter, " +
		"not an expression.")
	if neg {
		ck.MkLine.Explain(
			"Instead of !empty(${VARNAME:Mpattern}),",
			"you should write either of the following:",
			"",
			"\t!empty(VARNAME:Mpattern)",
			"\t${VARNAME:U:Mpattern} != \"\"",
			"",
			"If the pattern cannot match the number zero,",
			"you can omit the '!= \"\"', resulting in:",
			"",
			"\t${VARNAME:Mpattern}")
	} else {
		ck.MkLine.Explain(
			"Instead of empty(${VARNAME:Mpattern}),",
			"you should write either of the following:",
			"",
			"\tempty(VARNAME:Mpattern)",
			"\t${VARNAME:Mpattern} == \"\"",
			"",
			"If the pattern cannot match the number zero,",
			"you can replace the '== \"\"' with a '!',",
			"resulting in:",
			"",
			"\t!${VARNAME:Mpattern}")
	}
}

// checkEmptyType warns if there is a ':M' modifier in which the pattern
// doesn't match the type of the expression.
func (ck *MkCondChecker) checkEmptyType(expr *MkExpr) {
	for _, modifier := range expr.modifiers {
		ok, _, pattern, _ := modifier.MatchMatch()
		if ok {
			mkLineChecker := NewMkLineChecker(ck.MkLines, ck.MkLine)
			mkLineChecker.checkVartype(expr.varname, opUseMatch, pattern, "")
			continue
		}

		switch modifier.String() {
		default:
			return
		case "O", "u":
		}
	}
}

// mkCondStringLiteralUnquoted contains a safe subset of the characters
// that may be used without surrounding quotes in a comparison such as
// ${PKGPATH} == category/package.
// TODO: Check whether the ',' really needs to be here.
var mkCondStringLiteralUnquoted = textproc.NewByteSet("-+,./0-9@A-Z_a-z")

// mkCondModifierPatternLiteral contains a safe subset of the characters
// that are interpreted literally in the :M and :N modifiers.
// TODO: Check whether the ',' really needs to be here.
var mkCondModifierPatternLiteral = textproc.NewByteSet("-+,./0-9<=>@A-Z_a-z")

func (ck *MkCondChecker) checkCompare(left *MkCondTerm, op string, right *MkCondTerm) {
	switch {
	case right.Num != "":
		ck.checkCompareWithNum(left, op, right.Num)
	case left.Expr != nil && right.Expr == nil:
		ck.checkCompareExprStr(left.Expr, op, right.Str)
	}
}

func (ck *MkCondChecker) checkCompareExprStr(expr *MkExpr, op string, str string) {
	varname := expr.varname
	mods := expr.modifiers
	switch len(mods) {
	case 0:
		mkLineChecker := NewMkLineChecker(ck.MkLines, ck.MkLine)
		mkLineChecker.checkVartype(varname, opUseCompare, str, "")

		if varname == "PKGSRC_COMPILER" {
			ck.checkCompareExprStrCompiler(op, str)
		}

	case 1:
		if m, _, pattern, _ := mods[0].MatchMatch(); m {
			mkLineChecker := NewMkLineChecker(ck.MkLines, ck.MkLine)
			mkLineChecker.checkVartype(varname, opUseMatch, pattern, "")

			// After applying the :M or :N modifier, every expression may end up empty,
			// regardless of its data type. Therefore, there's no point in type-checking that case.
			if str != "" {
				mkLineChecker.checkVartype(varname, opUseCompare, str, "")
			}
		}

	default:
		// This case covers ${VAR:Mfilter:O:u} or similar uses in conditions.
		// To check these properly, pkglint first needs to know the most common
		// modifiers and how they interact.
		// As of March 2019, the modifiers are not modeled.
		// The following tracing statement makes it easy to discover these cases,
		// in order to decide whether checking them is worthwhile.
		if trace.Tracing {
			trace.Stepf("checkCompareExprStr ${%s%s} %s %s",
				expr.varname, expr.Mod(), op, str)
		}
	}
}

func (ck *MkCondChecker) checkCompareWithNum(left *MkCondTerm, op string, num string) {
	ck.checkCompareWithNumVersion(op, num)
	ck.checkCompareWithNumPython(left, op, num)
}

func (ck *MkCondChecker) checkCompareWithNumVersion(op string, num string) {
	if !contains(num, ".") {
		return
	}

	mkline := ck.MkLine
	mkline.Warnf("Numeric comparison %s %s.", op, num)
	mkline.Explain(
		"The numeric comparison of bmake is not suitable for version numbers",
		"since 5.1 == 5.10 == 5.1000000.",
		"",
		"To fix this, either enclose the number in double quotes,",
		"or use pattern matching:",
		"",
		"\t${OS_VERSION} == \"6.5\"",
		"\t${OS_VERSION:M1.[1-9]} || ${OS_VERSION:M1.[1-9].*}",
		"",
		"The second example needs to be split into two parts",
		"since with a single comparison of the form ${OS_VERSION:M1.[1-9]*},",
		"the version number 1.11 would also match, which is not intended.")
}

func (ck *MkCondChecker) checkCompareWithNumPython(left *MkCondTerm, op string, num string) {
	if left.Expr != nil && left.Expr.varname == "_PYTHON_VERSION" &&
		op != "==" && op != "!=" &&
		matches(num, `^\d+$`) {

		fixedNum := replaceAll(num, `^([0-9])([0-9])$`, `${1}0$2`)
		fix := ck.MkLine.Autofix()
		fix.Errorf("_PYTHON_VERSION must not be compared numerically.")
		fix.Explain(
			"The variable _PYTHON_VERSION must not be compared",
			"against an integer number, as these comparisons are",
			"not meaningful.",
			"For example, 27 < 39 < 40 < 41 < 310, which means that",
			"Python 3.10 would be considered newer than a",
			"possible future Python 4.0.",
			"",
			"In addition, _PYTHON_VERSION can be \"none\",",
			"which is not a number.")
		fix.Replace("${_PYTHON_VERSION} "+op+" "+num, "${PYTHON_VERSION} "+op+" "+fixedNum)
		fix.Apply()
	}
}

func (ck *MkCondChecker) checkCompareExprStrCompiler(op string, value string) {
	if !matches(value, `^\w+$`) {
		return
	}

	// It would be nice if original text of the whole comparison expression
	// were available at this point, to avoid guessing how much whitespace
	// the package author really used.

	matchOp := condStr(op == "==", "M", "N")

	fix := ck.MkLine.Autofix()
	fix.Errorf("Use ${PKGSRC_COMPILER:%s%s} instead of the %s operator.", matchOp, value, op)
	fix.Explain(
		"The PKGSRC_COMPILER can be a list of chained compilers, e.g. \"ccache distcc clang\".",
		"Therefore, comparing it using == or != leads to wrong results in these cases.")
	fix.Replace("${PKGSRC_COMPILER} "+op+" "+value, "${PKGSRC_COMPILER:"+matchOp+value+"}")
	fix.Replace("${PKGSRC_COMPILER} "+op+" \""+value+"\"", "${PKGSRC_COMPILER:"+matchOp+value+"}")
	fix.Apply()
}

func (ck *MkCondChecker) checkNotCompare(not *MkCond) {
	if not.Compare == nil {
		return
	}

	ck.MkLine.Warnf("The ! should use parentheses or be merged into the comparison operator.")
}

func (ck *MkCondChecker) checkContradictions() {
	mkline := ck.MkLine

	byVarname := make(map[string][]VarFact)
	levels := ck.MkLines.checkAllData.conditions.levels
	for _, level := range levels {
		if !level.current.NeedsCond() || level.current == mkline {
			continue
		}
		prevFacts := ck.collectFacts(level.current)
		for _, prevFact := range prevFacts {
			varname := prevFact.Varname
			byVarname[varname] = append(byVarname[varname], prevFact)
		}
	}

	facts := ck.collectFacts(mkline)
	for _, curr := range facts {
		varname := curr.Varname
		for _, prev := range byVarname[varname] {
			both := makepat.Intersect(prev.Pattern, curr.Pattern)
			if !both.CanMatch() {
				if prev.MkLine != mkline {
					mkline.Errorf("The patterns %q from %s and %q cannot match at the same time.",
						prev.PatternText, mkline.RelMkLine(prev.MkLine), curr.PatternText)
				} else {
					mkline.Errorf("The patterns %q and %q cannot match at the same time.",
						prev.PatternText, curr.PatternText)
				}
			}
		}
		byVarname[varname] = append(byVarname[varname], curr)
	}
}

// VarFact is a statement about a variable that is true in the current
// conditional branch.
type VarFact struct {
	Varname     string
	PatternText string
	Pattern     *makepat.Pattern
	MkLine      *MkLine
}

// collectFacts extracts those basic conditions that must definitely be true
// to make the whole condition evaluate to true.
// For example, in 'A && B', both A and B are facts, while in 'A || B',
// neither is a fact.
func (ck *MkCondChecker) collectFacts(mkline *MkLine) []VarFact {
	var facts []VarFact

	collectExpr := func(expr *MkExpr) {
		if expr == nil || len(expr.modifiers) != 1 {
			return
		}

		ok, positive, pattern, _ := expr.modifiers[0].MatchMatch()
		if !ok || !positive || containsExpr(pattern) {
			return
		}

		vartype := G.Pkgsrc.VariableType(ck.MkLines, expr.varname)
		if vartype.IsList() != no {
			return
		}

		m, err := makepat.Compile(pattern)
		if err != nil {
			return
		}

		facts = append(facts, VarFact{expr.varname, pattern, m, mkline})
	}

	var collectCond func(cond *MkCond)
	collectCond = func(cond *MkCond) {
		if cond == nil {
			// TODO: This is a workaround for a panic in '.ifndef VARNAME'.
			// Seen in net/wget/hacks.mk 1.1.
			// The parser for conditions needs to accept bare words.
			return
		}
		if cond.Term != nil {
			collectExpr(cond.Term.Expr)
		}
		if cond.Not != nil {
			collectExpr(cond.Not.Empty)
		}
		for _, cond := range cond.And {
			collectCond(cond)
		}
		if cond.Paren != nil {
			collectCond(cond.Paren)
		}
	}

	collectCond(mkline.Cond())

	return facts
}
