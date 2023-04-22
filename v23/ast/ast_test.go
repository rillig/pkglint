package ast

import (
	"fmt"
	"go/ast"
	"reflect"
	"testing"
)

func Test(t *testing.T) {
	f := NewFile("" +
		".\\\n" +
		"\tif ${COND}\n" +
		".endif\n")
	par := NewMkParser(f)

	line := par.ParseLine()

	switch line := line.(type) {
	case *MkCondLine:
		fmt.Println("directive", line.Directive)
	case *MkAssignLine:
		fmt.Println("assign", line.Name, "=", line.Value)
	default:
		fmt.Println("other", reflect.ValueOf(line).Type().Name())
	}

	ast.Print(nil, line)

	if want, got := ".\\\n\tif ${COND}\n", f.Text(line); got != want {
		t.Errorf("got %q, want %q", got, want)
	}
}
