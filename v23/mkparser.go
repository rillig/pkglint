package pkglint

import "github.com/rillig/pkglint/v23/textproc"

// MkParser wraps a MkLexer and provides methods for parsing
// things related to Makefiles.
type MkParser struct {
	diag         Diagnoser
	mklex        *MkLexer
	lexer        *textproc.Lexer
	EmitWarnings bool
}

func (p *MkParser) EOF() bool {
	return p.lexer.EOF()
}

func (p *MkParser) Rest() string {
	return p.lexer.Rest()
}

// NewMkParser creates a new parser for the given text.
//
// If diag is given, it is used for reporting parse errors and warnings.
// Otherwise, parsing is silent.
//
// The text argument is assumed to be after unescaping the # character,
// which means the # is a normal character and does not introduce a makefile comment.
// For Expr, this distinction is irrelevant.
func NewMkParser(diag Autofixer, text string) *MkParser {
	mklex := NewMkLexer(text, diag)
	return &MkParser{diag, mklex, mklex.lexer, diag != nil}
}

// MkCond parses a condition like ${OPSYS} == "NetBSD".
//
// See devel/bmake/files/cond.c.
func (p *MkParser) MkCond() *MkCond {
	and := p.mkCondAnd()
	if and == nil {
		return nil
	}

	ands := []*MkCond{and}
	for {
		mark := p.lexer.Mark()
		p.lexer.SkipHspace()
		if !p.lexer.SkipString("||") {
			break
		}
		next := p.mkCondAnd()
		if next == nil {
			p.lexer.Reset(mark)
			break
		}
		ands = append(ands, next)
	}
	if len(ands) == 1 {
		return and
	}
	return &MkCond{Or: ands}
}

func (p *MkParser) mkCondAnd() *MkCond {
	atom := p.mkCondCompare()
	if atom == nil {
		return nil
	}

	atoms := []*MkCond{atom}
	for {
		mark := p.lexer.Mark()
		p.lexer.SkipHspace()
		if p.lexer.NextString("&&") == "" {
			break
		}
		next := p.mkCondCompare()
		if next == nil {
			p.lexer.Reset(mark)
			break
		}
		atoms = append(atoms, next)
	}
	if len(atoms) == 1 {
		return atom
	}
	return &MkCond{And: atoms}
}

func (p *MkParser) mkCondCompare() *MkCond {
	if trace.Tracing {
		defer trace.Call1(p.Rest())()
	}

	lexer := p.lexer
	mark := lexer.Mark()
	lexer.SkipHspace()
	switch {
	case lexer.SkipByte('!'):
		notMark := lexer.Mark()
		cond := p.mkCondCompare()
		if cond != nil {
			return &MkCond{Not: cond}
		}
		lexer.Reset(notMark)
		return nil

	case lexer.SkipByte('('):
		cond := p.MkCond()
		if cond != nil {
			lexer.SkipHspace()
			if lexer.SkipByte(')') {
				return &MkCond{Paren: cond}
			}
		}
		lexer.Reset(mark)
		return nil

	case lexer.TestByteSet(textproc.Alpha):
		// This can only be a function name, not a string literal like in
		// amd64 == ${MACHINE_ARCH}, since bmake interprets it in the same
		// way, reporting a malformed conditional.
		return p.mkCondFunc()
	}

	lhs := p.mkCondTerm()

	if lhs != nil {
		lexer.SkipHspace()

		if m := lexer.NextRegexp(regcomp(`^(<|<=|==|!=|>=|>)[\t ]*(0x[0-9A-Fa-f]+|\d+(?:\.\d+)?)`)); m != nil {
			return &MkCond{Compare: &MkCondCompare{*lhs, m[1], MkCondTerm{Num: m[2]}}}
		}

		m := lexer.NextRegexp(regcomp(`^(?:<|<=|==|!=|>=|>)`))
		if len(m) == 0 {
			// See devel/bmake/files/cond.c:/\* For \.if \$/
			return &MkCond{Term: lhs}
		}
		lexer.SkipHspace()

		op := m[0]
		if op == "==" || op == "!=" {
			if mrhs := lexer.NextRegexp(regcomp(`^"([^"\$\\]*)"`)); mrhs != nil {
				return &MkCond{Compare: &MkCondCompare{*lhs, op, MkCondTerm{Str: mrhs[1]}}}
			}
		}

		rhs := p.mkCondTerm()
		if rhs != nil {
			return &MkCond{Compare: &MkCondCompare{*lhs, op, *rhs}}
		}

		if str := lexer.NextBytesSet(mkCondStringLiteralUnquoted); str != "" {
			return &MkCond{Compare: &MkCondCompare{*lhs, op, MkCondTerm{Str: str}}}
		}
	}

	// See devel/bmake/files/cond.c:/^CondCvtArg
	if m := lexer.NextRegexp(regcomp(`^(?:0x[0-9A-Fa-f]+|\d+(?:\.\d+)?)`)); m != nil {
		return &MkCond{Term: &MkCondTerm{Num: m[0]}}
	}

	lexer.Reset(mark)
	return nil
}

// mkCondTerm parses the following:
//
//	${VAR}
//	"${VAR}"
//	"text${VAR}text"
//	"text"
//
// It does not parse unquoted string literals since these are only allowed
// at the right-hand side of a comparison expression.
func (p *MkParser) mkCondTerm() *MkCondTerm {
	lexer := p.lexer

	if rhs := p.mklex.Expr(); rhs != nil {
		return &MkCondTerm{Expr: rhs}
	}

	if lexer.PeekByte() != '"' {
		return nil
	}

	mark := lexer.Mark()
	lexer.Skip(1)
	if quotedRHS := p.mklex.Expr(); quotedRHS != nil {
		if lexer.SkipByte('"') {
			return &MkCondTerm{Expr: quotedRHS}
		}
	}
	lexer.Reset(mark)

	lexer.Skip(1)
	rhsText := NewLazyStringBuilder(lexer.Rest())
loop:
	for {
		m := lexer.Mark()
		switch {
		case p.mklex.Expr() != nil,
			lexer.NextBytesSet(textproc.Alnum) != "",
			lexer.SkipBytesFunc(func(b byte) bool { return b != '"' && b != '\\' }):
			rhsText.WriteString(lexer.Since(m))

		case lexer.SkipString("\\\""),
			lexer.SkipString("\\\\"):
			rhsText.WriteByte(lexer.Since(m)[1])

		case lexer.SkipByte('"'):
			return &MkCondTerm{Str: rhsText.String()}
		default:
			break loop
		}
	}
	lexer.Reset(mark)

	return nil
}

func (p *MkParser) mkCondFunc() *MkCond {
	lexer := p.lexer
	mark := lexer.Mark()

	funcName := lexer.NextBytesSet(textproc.Lower)
	lexer.SkipHspace()
	if !lexer.SkipByte('(') {
		lexer.Reset(mark)
		return nil
	}

	switch funcName {
	case "defined":
		varname := p.mklex.Varname()
		if varname != "" && lexer.SkipByte(')') {
			return &MkCond{Defined: varname}
		}

	case "empty":
		if varname := p.mklex.Varname(); varname != "" {
			modifiers := p.mklex.ExprModifiers(varname, ')')
			if lexer.SkipByte(')') {
				return &MkCond{Empty: NewMkExpr(varname, modifiers...)}
			}
		}

	case "commands", "exists", "make", "target":
		argMark := lexer.Mark()
		for p.mklex.Expr() != nil || lexer.SkipBytesFunc(func(b byte) bool { return b != '$' && b != ')' }) {
		}
		arg := lexer.Since(argMark)
		if lexer.SkipByte(')') {
			return &MkCond{Call: &MkCondCall{funcName, arg}}
		}
	}

	lexer.Reset(mark)
	return nil
}

func (p *MkParser) Op() (bool, MkOperator) {
	lexer := p.lexer
	switch {
	case lexer.SkipString("!="):
		return true, opAssignShell
	case lexer.SkipString(":="):
		return true, opAssignEval
	case lexer.SkipString("+="):
		return true, opAssignAppend
	case lexer.SkipString("?="):
		return true, opAssignDefault
	case lexer.SkipString("="):
		return true, opAssign
	}
	return false, 0
}

func (p *MkParser) PkgbasePattern() string {

	lexer := p.lexer
	start := lexer.Mark()

	for {
		if p.mklex.Expr() != nil ||
			(!hasPrefix(lexer.Rest(), "{,nb") && lexer.SkipRegexp(regcomp(`^[\w.*+,{}]+`))) ||
			lexer.SkipRegexp(regcomp(`^\[[\w-]+\]`)) {
			continue
		}

		if lexer.PeekByte() == '-' && p.isPkgbasePart(lexer.Rest()[1:]) {
			lexer.Skip(1)
		} else {
			break
		}
	}

	pkgbase := lexer.Since(start)
	if !hasBalancedBraces(pkgbase) { // as in "{ssh{,6}-[0-9]"
		lexer.Reset(start)
		return ""
	}
	return pkgbase
}

// isPkgbasePart returns whether str, when following a hyphen,
// continues the package base (as in "mysql-client"), or whether it
// starts the version (as in "mysql-1.0").
func (*MkParser) isPkgbasePart(str string) bool {
	lexer := textproc.NewLexer(str)

	lexer.SkipByte('{')
	lexer.SkipByte('[')
	if lexer.NextByteSet(textproc.Alpha) != -1 {
		return true
	}

	if expr := NewMkLexer(lexer.Rest(), nil).Expr(); expr != nil {
		return !contains(expr.varname, "VER") && len(expr.modifiers) == 0
	}

	return false
}

type DependencyPattern struct {
	Pkgbase  string // "freeciv-client", "{gcc48,gcc48-libs}", "${EMACS_REQD}"
	LowerOp  string // ">=", ">"
	Lower    string // "2.5.0", "${PYVER}"
	UpperOp  string // "<", "<="
	Upper    string // "3.0", "${PYVER}"
	Wildcard string // "[0-9]*", "1.5.*", "${PYVER}"
}

// DependencyPattern parses a dependency pattern like "pkg>=1<2" or "pkg-[0-9]*".
func (p *MkParser) DependencyPattern() *DependencyPattern {
	lexer := p.lexer

	parseVersion := func() string {
		mark := lexer.Mark()

		if p.mklex.Expr() != nil {
			for p.mklex.Expr() != nil || lexer.SkipRegexp(regcomp(`^\.\w+`)) {
			}
			return lexer.Since(mark)
		}

		m := lexer.NextRegexp(regcomp(`^\d[\w.]*`))
		if m != nil {
			return m[0]
		}

		return ""
	}

	var dp DependencyPattern
	mark := lexer.Mark()
	dp.Pkgbase = p.PkgbasePattern()
	if dp.Pkgbase == "" {
		return nil
	}

	mark2 := lexer.Mark()
	op := lexer.NextString(">=")
	if op == "" {
		op = lexer.NextString(">")
	}

	if op != "" {
		version := parseVersion()
		if version != "" {
			dp.LowerOp = op
			dp.Lower = version
		} else {
			lexer.Reset(mark2)
		}
	}

	op = lexer.NextString("<=")
	if op == "" {
		op = lexer.NextString("<")
	}

	if op != "" {
		version := parseVersion()
		if version != "" {
			dp.UpperOp = op
			dp.Upper = version
		} else {
			lexer.Reset(mark2)
		}
	}

	if dp.LowerOp != "" || dp.UpperOp != "" {
		return &dp
	}

	if lexer.SkipByte('-') && lexer.Rest() != "" {
		versionMark := lexer.Mark()

		for p.mklex.Expr() != nil ||
			lexer.SkipRegexp(regcomp(`^\[[^\]]+]`)) ||
			lexer.SkipRegexp(regcomp(`^[\w\[\]*_.\-]+`)) {
		}

		if !lexer.SkipString("{,nb*}") {
			lexer.SkipString("{,nb[0-9]*}")
		}

		dp.Wildcard = lexer.Since(versionMark)
		if dp.Wildcard != "" {
			return &dp
		}
	}

	if hasPrefix(dp.Pkgbase, "$") && hasSuffix(dp.Pkgbase, "}") {
		if !lexer.SkipString("{,nb*}") {
			lexer.SkipString("{,nb[0-9]*}")
		}
		return &dp
	}

	lexer.Reset(mark)
	return nil
}

// ToExpr converts the given string into a MkExpr, or returns nil
// if there is a parse error or some trailing text.
// Parse errors do not produce any warnings.
func ToExpr(str string) *MkExpr {
	p := NewMkLexer(str, nil)
	if expr := p.Expr(); expr != nil && p.EOF() {
		return expr
	}
	return nil
}

// MkCond is a condition in a makefile, such as ${OPSYS} == NetBSD.
//
// The representation is somewhere between syntactic and semantic.
// The condition '!empty(VARNAME)' is represented differently from the
// condition '${VARNAME} != ""'.
// For higher level analysis, a unified representation might be better.
// See MkParser.MkCond.
type MkCond struct {
	Or  []*MkCond
	And []*MkCond
	Not *MkCond

	Defined string
	Empty   *MkExpr
	Term    *MkCondTerm
	Compare *MkCondCompare
	Call    *MkCondCall
	Paren   *MkCond
}
type MkCondCompare struct {
	Left MkCondTerm
	// For numeric comparison: one of <, <=, ==, !=, >=, >.
	//
	// For string comparison: one of ==, !=.
	//
	// For not-empty test: "".
	Op    string
	Right MkCondTerm
}
type MkCondTerm struct {
	Str  string
	Num  string
	Expr *MkExpr
}
type MkCondCall struct {
	Name string
	Arg  string
}

// MkCondCallback defines the actions for walking a makefile condition
// using MkCondWalker.Walk.
type MkCondCallback struct {
	And     func(conds []*MkCond)
	Not     func(cond *MkCond)
	Defined func(varname string)
	Empty   func(empty *MkExpr)
	Compare func(left *MkCondTerm, op string, right *MkCondTerm)
	Call    func(name string, arg string)
	Paren   func(cond *MkCond)

	// Var is called for every terminal condition that consists solely
	// of an expression, possibly enclosed in double quotes, but without
	// any surrounding string literal parts.
	Var func(expr *MkExpr)

	// Expr is called for each variable that is used in some expression.
	Expr func(expr *MkExpr)
}

func (cond *MkCond) Walk(callback *MkCondCallback) {
	(&MkCondWalker{}).Walk(cond, callback)
}

type MkCondWalker struct{}

func (w *MkCondWalker) Walk(cond *MkCond, callback *MkCondCallback) {
	switch {
	case cond.Or != nil:
		for _, or := range cond.Or {
			w.Walk(or, callback)
		}

	case cond.And != nil:
		if callback.And != nil {
			callback.And(cond.And)
		}
		for _, and := range cond.And {
			w.Walk(and, callback)
		}

	case cond.Not != nil:
		if callback.Not != nil {
			callback.Not(cond.Not)
		}
		w.Walk(cond.Not, callback)

	case cond.Defined != "":
		if callback.Defined != nil {
			callback.Defined(cond.Defined)
		}
		if callback.Expr != nil {
			// This is not really an Expr, it's more an ExprDefined.
			// But in practice, they are similar enough to be treated the same.
			callback.Expr(NewMkExpr(cond.Defined))
		}

	case cond.Term != nil && cond.Term.Expr != nil:
		if callback.Var != nil {
			callback.Var(cond.Term.Expr)
		}
		if callback.Expr != nil {
			callback.Expr(cond.Term.Expr)
		}

	case cond.Term != nil && cond.Term.Str != "":
		w.walkStr(cond.Term.Str, callback)

	case cond.Empty != nil:
		if callback.Empty != nil {
			callback.Empty(cond.Empty)
		}
		if callback.Expr != nil {
			callback.Expr(cond.Empty)
		}

	case cond.Compare != nil:
		cmp := cond.Compare
		if callback.Compare != nil {
			callback.Compare(&cmp.Left, cmp.Op, &cmp.Right)
		}
		w.walkAtom(&cmp.Left, callback)
		w.walkAtom(&cmp.Right, callback)

	case cond.Call != nil:
		if callback.Call != nil {
			call := cond.Call
			callback.Call(call.Name, call.Arg)
		}
		w.walkStr(cond.Call.Arg, callback)

	case cond.Paren != nil:
		if callback.Paren != nil {
			callback.Paren(cond.Paren)
		}
		w.Walk(cond.Paren, callback)
	}
}

func (w *MkCondWalker) walkAtom(atom *MkCondTerm, callback *MkCondCallback) {
	switch {
	case atom.Expr != nil:
		if callback.Expr != nil {
			callback.Expr(atom.Expr)
		}
	case atom.Num != "":
		break
	default:
		w.walkStr(atom.Str, callback)
	}
}

func (w *MkCondWalker) walkStr(str string, callback *MkCondCallback) {
	if callback.Expr != nil {
		tokens, _ := NewMkLexer(str, nil).MkTokens()
		for _, token := range tokens {
			if token.Expr != nil {
				callback.Expr(token.Expr)
			}
		}
	}
}
