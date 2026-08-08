package licenses

import (
	"encoding/json"
	"github.com/rillig/pkglint/v23/intqa"
	"github.com/rillig/pkglint/v23/textproc"
	"reflect"
	"strings"
	"testing"
)

func Test_Parse(t *testing.T) {
	test := func(cond string, expected string) {
		have := toJSON(Parse(cond))
		if have != expected {
			t.Error(expected, have)
		}
	}

	testDeep := func(cond string, expected *Condition) {
		have := Parse(cond)
		if !reflect.DeepEqual(have, expected) {
			t.Error(expected, have)
		}
	}

	testDeep("gnu-gpl-v2", NewName("gnu-gpl-v2"))

	test("gnu-gpl-v2", "{Name:gnu-gpl-v2}")
	test("citrix_ica-license", "{Name:citrix_ica-license}")

	test("a AND b", "{And:true,Children:[{Name:a},{Name:b}]}")
	test("a OR b", "{Or:true,Children:[{Name:a},{Name:b}]}")

	test("a OR (b AND c)", "{Or:true,Children:[{Name:a},{Paren:{And:true,Children:[{Name:b},{Name:c}]}}]}")
	test("(a OR b) AND c", "{And:true,Children:[{Paren:{Or:true,Children:[{Name:a},{Name:b}]}},{Name:c}]}")

	test("a AND b AND c AND d", "{And:true,Children:[{Name:a},{Name:b},{Name:c},{Name:d}]}")
	testDeep(
		"a AND b AND c AND d",
		NewAnd(NewName("a"), NewName("b"), NewName("c"), NewName("d")))

	test("a OR b OR c OR d", "{Or:true,Children:[{Name:a},{Name:b},{Name:c},{Name:d}]}")
	testDeep(
		"a OR b OR c OR d",
		NewOr(NewName("a"), NewName("b"), NewName("c"), NewName("d")))

	test("(a OR b) AND (c AND d)", "{And:true,Children:[{Paren:{Or:true,Children:[{Name:a},{Name:b}]}},{Paren:{And:true,Children:[{Name:c},{Name:d}]}}]}")
	testDeep(
		"(a OR b) AND (c AND d)",
		NewAnd(
			NewParen(NewOr(NewName("a"), NewName("b"))),
			NewParen(NewAnd(NewName("c"), NewName("d")))))

	test("a AND b OR c AND d", "{And:true,Or:true,Children:[{Name:a},{Name:b},{Name:c},{Name:d}]}")
	test("((a AND (b AND c)))", "{Paren:{Children:[{Paren:{And:true,Children:[{Name:a},{Paren:{And:true,Children:[{Name:b},{Name:c}]}}]}}]}}")

	if have, want := Parse("a AND b OR c AND d").String(), "a MIXED b MIXED c MIXED d"; have != want {
		t.Errorf("want %#v, have %#v", want, have)
	}
	if have := Parse("AND artistic"); have != nil {
		t.Errorf("have %#v", have)
	}
	if have := Parse("invalid/character"); have != nil {
		t.Errorf("have %#v", have)
	}
}

func Test_Condition_String(t *testing.T) {
	if have, want := NewName("a").String(), "a"; have != want {
		t.Errorf("want %#v, have %#v", want, have)
	}

	if have, want := NewAnd(NewName("a"), NewName("b")).String(), "a AND b"; have != want {
		t.Errorf("want %#v, have %#v", want, have)
	}

	if have, want := NewOr(NewName("a"), NewName("b")).String(), "a OR b"; have != want {
		t.Errorf("want %#v, have %#v", want, have)
	}

	if have, want := NewAnd(
		NewParen(NewOr(NewName("a"), NewName("b"))),
		NewParen(NewOr(NewName("c"), NewName("d")))).String(), "(a OR b) AND (c OR d)"; have != want {
		t.Errorf("want %#v, have %#v", want, have)
	}

	mixed := NewAnd(NewName("a"), NewName("b"), NewName("c"))
	mixed.Or = true
	if have, want := mixed.String(), "a MIXED b MIXED c"; have != want {
		t.Errorf("want %#v, have %#v", want, have)
	}
}

func Test_Condition_Walk(t *testing.T) {
	condition := Parse("(a OR b) AND (c AND d)")

	var out []string
	condition.Walk(func(condition *Condition) {
		switch {
		case condition.Name != "":
			out = append(out, condition.Name)
		case condition.Paren != nil:
			out = append(out, "()")
		case condition.And && condition.Or:
			out = append(out, "MIXED")
		case condition.And:
			out = append(out, "AND")
		case condition.Or:
			out = append(out, "OR")
		default:
			panic("unexpected")
		}
	})

	if have, want := out, []string{"a", "b", "OR", "()", "c", "d", "AND", "()", "AND"}; !reflect.DeepEqual(have, want) {
		t.Errorf("want %#v, have %#v", want, have)
	}
}

func Test_licenseLexer_Lex(t *testing.T) {

	test := func(text string, tokenType int) {
		lexer := &licenseLexer{lexer: textproc.NewLexer(text)}
		var token liyySymType
		lex := lexer.Lex(&token)
		if have, want := lex, tokenType; have != want {
			t.Errorf("want %#v, have %#v", want, have)
		}
	}
	testName := func(text string, name string) {
		lexer := &licenseLexer{lexer: textproc.NewLexer(text)}
		var token liyySymType
		lex := lexer.Lex(&token)
		if have, want := lex, ltNAME; have != want {
			t.Errorf("want %#v, have %#v", want, have)
		}
		if have, want := token.Node.Name, name; have != want {
			t.Errorf("want %#v, have %#v", want, have)
		}
	}

	test("", 0)
	test("(", ltOPEN)
	test(")", ltCLOSE)
	test("AND", ltAND)
	test("OR", ltOR)
	test("?", -1)
	test("license-name", ltNAME)
	test("license_name", ltNAME)

	test("AND rest", ltAND)
	test("AND-rest", ltNAME)

	testName("license-name", "license-name")
	testName("license_name", "license_name")
	testName("AND-rest", "AND-rest")
}

func NewName(name string) *Condition {
	return &Condition{Name: name}
}
func NewParen(child *Condition) *Condition {
	return &Condition{Paren: child}
}
func NewAnd(parts ...*Condition) *Condition {
	return &Condition{Children: parts, And: true}
}
func NewOr(parts ...*Condition) *Condition {
	return &Condition{Children: parts, Or: true}
}

func toJSON(cond *Condition) string {
	jsonStr, _ := json.Marshal(cond)
	return strings.Replace(string(jsonStr), "\"", "", -1)
}

func Test__qa(t *testing.T) {
	ck := intqa.NewQAChecker(t.Errorf)
	ck.Configure("*", "*", "*", -intqa.EMissingTest)
	ck.Check()
}
