package main

import (
	"fmt"
	"os"
	"testing"
)

type Asserter struct {
	t *testing.T
}
func (a *Asserter) assertStringEqual(expected, actual string) {
	if actual != expected {
		a.t.Fatalf("Expected %#v, got %#v", expected, actual)
	} else {
		fmt.Printf("assertStringEqual: %#v\n", actual)
	}
}

func TestConvertToLogicalLines_nocont(t *testing.T) {
	a := Asserter{t}
	phys := []PhysLine{
		{1, "first line\n"},
		{2, "second line\n"},
	}
	lines := convertToLogicalLines("fname", phys, false)
	a.assertStringEqual("fname", lines[0].fname)
	a.assertStringEqual("1", lines[0].lines)
	a.assertStringEqual("first line", lines[0].text)
	a.assertStringEqual("fname", lines[1].fname)
	a.assertStringEqual("2", lines[1].lines)
	a.assertStringEqual("second line", lines[1].text)
	a.assertStringEqual("a","b")
}

func TestConvertToLogicalLines_contInLastLine(t *testing.T) {
	a := Asserter{t}
	physlines := []PhysLine{
		{1, "last line\\"},
	}
	lines := convertToLogicalLines("fname", physlines, true)
	a.assertStringEqual("fname", lines[0].fname)
	a.assertStringEqual("1", lines[0].lines)
	a.assertStringEqual("last line", lines[0].text)
}

func TestMe(t *testing.T) {
	fmt.Fprintf(os.Stderr, "hello from test\n")
}

func TestMain(m *testing.M) {
	GlobalVars.opts = &CmdOpts{}
}
