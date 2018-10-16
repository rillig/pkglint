package licenses

import (
	"netbsd.org/pkglint/textproc"
)

// Condition describes a complex license condition.
// It has either `Name` or `Paren` or `Children` set.
// In the `Children` case, `And` and `Or` specify the operators used.
// Malformed license conditions can have both `And` and `Or` set.
type Condition struct {
	Name     string       `json:",omitempty"`
	Paren    *Condition   `json:",omitempty"`
	And      bool         `json:",omitempty"`
	Or       bool         `json:",omitempty"`
	Children []*Condition `json:",omitempty"`
}

func Parse(licenses string) *Condition {
	lexer := &licenseLexer{lexer: textproc.NewLexer(licenses)}
	result := liyyNewParser().Parse(lexer)
	if result == 0 && lexer.lexer.EOF() {
		return lexer.result
	}
	return nil
}

func (cond *Condition) String() string {
	if cond.Name != "" {
		return cond.Name
	}
	if cond.Paren != nil {
		return "(" + cond.Paren.String() + ")"
	}
	s := ""
	separator := [...]string{"", " AND ", " OR ", " MIXED "}[b2i(cond.And)+2*b2i(cond.Or)]
	for i, child := range cond.Children {
		if i != 0 {
			s += separator
		}
		s += child.String()
	}
	return s
}

func (cond *Condition) Walk(callback func(*Condition)) {
	if cond.Paren != nil {
		cond.Paren.Walk(callback)
	}
	for _, child := range cond.Children {
		child.Walk(callback)
	}
	callback(cond)
}

//go:generate goyacc -p liyy -o licensesyacc.go -v licensesyacc.log licenses.y

type licenseLexer struct {
	lexer  *textproc.Lexer
	result *Condition
	error  string
}

func (lexer *licenseLexer) Lex(llval *liyySymType) int {
	repl := lexer.lexer
	repl.NextHspace()
	switch {
	case repl.EOF():
		return 0
	case repl.NextByte('('):
		return ltOPEN
	case repl.NextByte(')'):
		return ltCLOSE
	}

	word := repl.NextBytesFunc(func(b byte) bool {
		return 'a' <= b && b <= 'z' || '0' <= b && b <= '9' || 'A' <= b && b <= 'Z' || b == '-' || b == '.'
	})
	switch word {
	case "AND":
		return ltAND
	case "OR":
		return ltOR
	case "":
		return -1
	default:
		llval.Node = &Condition{Name: word}
		return ltNAME
	}
}

func (lexer *licenseLexer) Error(s string) {
	lexer.error = s
}

func b2i(x bool) int {
	if x {
		return 1
	}
	return 0
}
