package main

import (
	"testing"
)

type Asserter struct {
	t *testing.T
}

func (a *Asserter) assertEqual(expected, actual interface{}) {
	if actual != expected {
		a.t.Fatalf("Expected %#v, got %#v", expected, actual)
	}
}

func TestConvertToLogicalLines_nocont(t *testing.T) {
	G.opts = &CmdOpts{}

	a := Asserter{t}
	phys := []PhysLine{
		{1, "first line\n"},
		{2, "second line\n"},
	}
	lines := convertToLogicalLines("fname", phys, false)
	a.assertEqual("fname", lines[0].fname)
	a.assertEqual("1", lines[0].lines)
	a.assertEqual("first line", lines[0].text)
	a.assertEqual("fname", lines[1].fname)
	a.assertEqual("2", lines[1].lines)
	a.assertEqual("second line", lines[1].text)
}

func TestConvertToLogicalLines_contInLastLine(t *testing.T) {
	G.opts = &CmdOpts{}

	a := Asserter{t}
	physlines := []PhysLine{
		{1, "last line\\"},
	}
	lines := convertToLogicalLines("fname", physlines, true)
	a.assertEqual("fname", lines[0].fname)
	a.assertEqual("1", lines[0].lines)
	a.assertEqual("last line ", lines[0].text)
}
