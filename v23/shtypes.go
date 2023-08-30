package pkglint

//go:generate goyacc -o shellyacc.go -v shellyacc.log -p shyy shell.y

type ShAtomType uint8

const (
	shtSpace    ShAtomType = iota
	shtExpr                // ${PREFIX}
	shtShExpr              // $${var:-pol}
	shtText                // while, cat, ENV=value
	shtOperator            // (, ;, |
	shtComment             // # ...
	shtSubshell            // $$(
)

func (t ShAtomType) String() string {
	return [...]string{
		"space",
		"expr",
		"shexpr",
		"text",
		"operator",
		"comment",
		"subshell",
	}[t]
}

// IsWord checks whether the atom counts as text.
// Makefile variables, shell variables and other text counts,
// but keywords, operators and separators don't.
func (t ShAtomType) IsWord() bool {
	switch t {
	case shtExpr, shtShExpr, shtText:
		return true
	}
	return false
}

type ShAtom struct {
	Type    ShAtomType
	MkText  string
	Quoting ShQuoting // The quoting state at the end of the token

	//  * usually nil
	//  * for shtExpr a *MkExpr
	//  * for shtShExpr a string
	data interface{}
}

func (atom *ShAtom) String() string {
	if atom.Type == shtText && atom.Quoting == shqPlain {
		return sprintf("%q", atom.MkText)
	}
	if atom.Type == shtExpr {
		expr := atom.Expr()
		return sprintf("expr(%q)", expr.varname+expr.Mod())
	}
	return sprintf("ShAtom(%v, %q, %s)", atom.Type, atom.MkText, atom.Quoting)
}

// Expr returns a read access to a makefile variable, or nil for plain shell tokens.
func (atom *ShAtom) Expr() *MkExpr {
	if atom.Type == shtExpr {
		return atom.data.(*MkExpr)
	}
	return nil
}

// ShVarname applies to shell variable atoms like $$varname or $${varname:-modifier}
// and returns the name of the shell variable.
func (atom *ShAtom) ShVarname() string {
	return atom.data.(string)
}

// ShQuoting describes the context in which a string appears
// and how it must be unescaped to get its literal value.
type ShQuoting uint8

const (
	shqPlain           ShQuoting = iota // e.g. word
	shqDquot                            // e.g. "word"
	shqSquot                            // e.g. 'word'
	shqBackt                            // e.g. `word`
	shqSubsh                            // e.g. $(word)
	shqDquotBackt                       // e.g. "`word`"
	shqBacktDquot                       // e.g. `"word"`
	shqBacktSquot                       // e.g. `'word'`
	shqSubshDquot                       // e.g. $("word")
	shqSubshSquot                       // e.g. $('word')
	shqSubshBackt                       // e.g. $(`word`)
	shqDquotBacktDquot                  // e.g. "`"word"`"
	shqDquotBacktSquot                  // e.g. "`'word'`"
)

// String returns a very short identifier for the quoting state.
// In this identifier, d means double quotes, s means single quotes,
// b means backticks and S means subshell.
func (q ShQuoting) String() string {
	return [...]string{
		"plain",
		"d", "s", "b", "S",
		"db", "bd", "bs", "Sd", "Ss", "Sb",
		"dbd", "dbs",
	}[q]
}

func (q ShQuoting) ToExprContext() EctxQuoting {
	switch q {
	case shqPlain:
		return EctxQuotPlain
	case shqDquot:
		return EctxQuotDquot
	case shqSquot:
		return EctxQuotSquot
	case shqBackt:
		return EctxQuotBackt
	}
	return EctxQuotUnknown
}

// ShToken is an operator or a keyword or some text intermingled with variables.
//
// Examples:
//
//	;
//	then
//	"The number of pkgsrc packages in ${PREFIX} is $$packages."
//
// See http://pubs.opengroup.org/onlinepubs/9699919799/utilities/V3_chap02.html#tag_18_10_02
type ShToken struct {
	MkText string // The text as it appeared in the Makefile, after replacing `\#` with `#`
	Atoms  []*ShAtom
}

func NewShToken(mkText string, atoms ...*ShAtom) *ShToken {
	assert(mkText != "")
	assert(len(atoms) > 0)
	return &ShToken{mkText, atoms}
}

func (token *ShToken) String() string {
	return sprintf("ShToken(%v)", token.Atoms)
}
