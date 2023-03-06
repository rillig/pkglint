// Package ast provides a parser that reads structured data from files,
// maintaining the exact relationship between the text stored in the file and
// the application-visible value.
//
// It is work in progress.
package ast

import (
	"io"
	"netbsd.org/pkglint/textproc"
	"strings"
)

type Pos uint32

func (p Pos) Plus(offset int) Pos  { return Pos(int(p) + offset) }
func (p Pos) PlusLen(s string) Pos { return p.Plus(len(s)) }

const NoPos Pos = 0

type FileSet struct {
	files map[string]*File
}

type File struct {
	text string
}

func NewFileSet() *FileSet {
	return &FileSet{map[string]*File{}}
}

func (fset *FileSet) Add(filename string, text string) *File {
	file := File{"\000" + text}
	fset.files[filename] = &file
	return &file
}

func (f *File) Text(n Node) string {
	return f.text[n.Start():n.End()]
}

// A Node in an abstract syntax tree represents a structural element of the
// file. Every single byte from the file must be represented in some node,
// even whitespace, linebreaks and comments.
type Node interface {
	Start() Pos
	End() Pos
}

// Literal represents literal text without escaping.
type Literal struct {
	start Pos
	Text  string
}

func (l *Literal) Start() Pos { return l.start }
func (l *Literal) End() Pos   { return l.start.PlusLen(l.Text) }

// EscapableText represents text that may contain escape sequences such as
// '\newline', '$$', '\$', '\#'. In that case, the represented text does not
// equal the text that is stored in the file.
type EscapableText struct {
	start Pos
	end   Pos
	// The text with all escaping removed.
	// The exact amount of escaping depends on the layer of the parser:
	//
	//  * '\#' and '\newline'
	//  * '$$' in make strings
	//  * '\$' in shell commands
	//  * '\.' in sed commands
	LogicalText string
}

func (t *EscapableText) Start() Pos { return t.start }
func (t *EscapableText) End() Pos   { return t.start.PlusLen(t.LogicalText) }

// Space represents whitespace. In case of backslash-newline sequences, the
// represented text does not equal the text that is stored in the file.
type Space struct {
	start Pos
	end   Pos
	// The text with all escaping removed.
	// Consists only of whitespace.
	LogicalText string
}

func (s *Space) Start() Pos { return s.start }
func (s *Space) End() Pos   { return s.end }

type MkCond interface {
	Node
}

type MkCondBinary struct {
	Left   MkCond
	OpText *Literal
	Op     MkCondBoolOp
	Right  MkCond
}

func (b *MkCondBinary) Start() Pos { return b.Left.Start() }
func (b *MkCondBinary) End() Pos   { return b.Right.End() }

type MkCondParen struct {
	Open   *Literal
	Space1 *Space
	X      MkCond
	Space2 *Space
	Close  *Literal
}

func (p *MkCondParen) Start() Pos { return p.Open.Start() }
func (p *MkCondParen) End() Pos   { return p.Close.End() }

type MkCondNot struct {
	Exclam *Literal
	Space  Space
	X      MkCond
}

func (n *MkCondNot) Start() Pos { return n.Exclam.Start() }
func (n *MkCondNot) End() Pos   { return n.X.End() }

// MkExpr represents an expression such as '$V', '${VAR:Mpattern}' or
// '$(PARENTHESIZED)'.
type MkExpr struct {
	Open      *Literal
	Varname   *MkVarname
	Modifiers []MkExprModifier
	Close     *Literal
}

func (e *MkExpr) Start() Pos { return e.Open.Start() }
func (e *MkExpr) End() Pos   { return e.Close.End() }

// MkExprModifier represents a single modifier in an expression, such as the
// ':Mpattern' in the expression '${VAR:Ufallback:Mpattern:S,from,to,}'.
type MkExprModifier interface {
	Node
	ModifierText()
}

type MkVarname EscapableText // TODO

type Reader interface {
	Read(rd io.Reader) (Node, error)
}

type Writer interface {
	Write(wr io.Writer, n Node) error
}

// Editor allows manipulating the AST in-memory.
type Editor interface {
	Remove(Node)
	Replace(Node, Node)
	InsertBefore(Node, Node)
	InsertAfter(Node, Node)
}

type MkCondCompareOp uint8

const (
	LT MkCondCompareOp = iota + 1
	LE
	EQ
	NE
	GE
	GT
)

func (op MkCondCompareOp) String() string { return [...]string{"<", "<=", "==", "!=", ">=", ">"}[op] }

type MkCondBoolOp uint8

const (
	NOT MkCondBoolOp = iota + 1
	AND
	OR
)

func (op MkCondBoolOp) String() string { return [...]string{"&&", "||", "!"}[op] }

type MkParser struct {
	lex     *textproc.Lexer
	textLen int
}

func NewMkParser(f *File) *MkParser {
	lex := textproc.NewLexer(f.text)
	lex.Skip(1)
	return &MkParser{lex, len(f.text)}
}

func (p *MkParser) ParseLine() MkLine {
	dot := p.ParseLiteral(".")
	s1 := p.ParseSpace()
	directive := p.ParseDirective()
	s2 := p.ParseSpace()
	cond := p.ParseMkCond()
	s3 := p.ParseSpace()
	comment := p.ParseComment()

	return &MkCondLine{
		dot,
		s1,
		directive,
		s2,
		cond,
		s3,
		comment,
	}
}

var directiveBytes = textproc.NewByteSet("a-z-")

func (p *MkParser) ParseDirective() *Literal {
	pos := p.Pos()
	dir := p.lex.NextBytesSet(directiveBytes)
	if dir == "" {
		return nil
	}
	return &Literal{pos, dir}
}

func (p *MkParser) ParseMkCond() *MkCond {
	return nil
}

func (p *MkParser) ParseComment() *EscapableText {
	return nil
}

func (p *MkParser) ParseSpace() *Space {
	lex := p.lex
	start := p.Pos()
	var sb strings.Builder
again:
	hspace := lex.NextHspace()
	if hspace != "" {
		sb.WriteString(hspace)
		goto again
	}
	if lex.SkipString("\\\n") {
		sb.WriteString(" ")
		goto again
	}
	return &Space{start, p.Pos(), sb.String()}
}

func (p *MkParser) ParseLiteral(s string) *Literal {
	lex := p.lex
	start := p.Pos()
	if !lex.SkipString(s) {
		return nil
	}
	return &Literal{start, s}
}

func (p *MkParser) Pos() Pos {
	return Pos(p.textLen - len(p.lex.Rest()))
}

type MkLine interface {
	Node
}

type MkCondLine struct {
	Dot       *Literal
	S0        *Space
	Directive *Literal
	S1        *Space
	Cond      *MkCond
	S2        *Space
	Comment   *EscapableText
}

func (l *MkCondLine) Start() Pos { return l.Dot.Start() }
func (l *MkCondLine) End() Pos   { return l.Comment.End() }

type MkAssignLine struct {
	S0      *Space
	Name    *MkVarname
	S1      *Space
	Op      *Literal
	S2      *Space
	Value   *MkString
	S3      *Space
	Comment *EscapableText
}

func (l *MkAssignLine) Start() Pos { return l.S0.Start() }
func (l *MkAssignLine) End() Pos   { return l.Comment.Start() }

type MkMessageLine struct {
	Dot     *Literal
	S0      *Space
	Message *EscapableText
	S1      *Space
	Comment *EscapableText
}

func (l *MkMessageLine) Start() Pos { return l.S0.Start() }
func (l *MkMessageLine) End() Pos   { return l.Comment.Start() }

type MkDependencyLine struct {
	S0      *Space
	Targets []*MkWord
	S1      *Space
	Op      *Literal
	S2      *Space
	Sources []*MkWord
	S3      *Space
	Comment *EscapableText
}

type MkString struct {
	Parts []MkStringPart
}

type MkStringPart interface {
}

type MkWord struct {
}
