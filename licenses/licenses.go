package licenses

import "netbsd.org/pkglint/textproc"

// Condition describes a complex license condition.
// It has either `Name` or `And` or `Or` set.
// Malformed license conditions can have both `And` and `Or` set.
type Condition struct {
	Name string
	Main *Condition
	And  []*Condition
	Or   []*Condition
}

func Parse(licenses string) *Condition {
	lexer := &licenseLexer{repl: textproc.NewPrefixReplacer(licenses)}
	result := liyyNewParser().Parse(lexer)
	if result == 0 {
		return lexer.result
	}
	return nil
}

func (lc *Condition) String() string {
	s := lc.Name
	if lc.Main != nil {
		s += lc.Main.String()
	}
	if len(lc.And) == 0 && len(lc.Or) == 0 {
		return s
	}

	for _, and := range lc.And {
		s += " AND " + and.String()
	}
	for _, or := range lc.Or {
		s += " OR " + or.String()
	}
	return "(" + s + ")"
}

func (lc *Condition) Walk(callback func(*Condition)) {
	callback(lc)
	if lc.Main != nil {
		lc.Main.Walk(callback)
	}
	for _, and := range lc.And {
		and.Walk(callback)
	}
	for _, or := range lc.Or {
		or.Walk(callback)
	}
}

//go:generate go tool yacc -p liyy -o licensesyacc.go -v licensesyacc.log licenses.y

type licenseLexer struct {
	repl   *textproc.PrefixReplacer
	result *Condition
	error  string
}

func (lexer *licenseLexer) Lex(llval *liyySymType) int {
	repl := lexer.repl
	repl.AdvanceHspace()
	switch {
	case repl.EOF():
		return 0
	case repl.AdvanceStr("("):
		return ltOPEN
	case repl.AdvanceStr(")"):
		return ltCLOSE
	case repl.AdvanceRegexp(`^[\w-.]+`):
		word := repl.Group(0)
		switch word {
		case "AND":
			return ltAND
		case "OR":
			return ltOR
		default:
			llval.Node = &Condition{Name: word}
			return ltNAME
		}
	}
	return -1
}

func (lexer *licenseLexer) Error(s string) {
	lexer.error = s
}
