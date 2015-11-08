package main

import (
	"testing"
)

func TestDetermineUsedVariables(t *testing.T) {
	a := Asserter{t}

	G = &GlobalVarsType{}
	defer func() { G = nil }()
	G.mkContext = newMkContext()

	line := NewLine("fname", "1", "${VAR}", nil)
	lines := make([]*Line, 1)
	lines[0] = line
	determineUsedVariables(lines)
	a.assertEqual(1, len(G.mkContext.varuse))
	a.assertEqual(line, G.mkContext.varuse["VAR"])

	G.mkContext.varuse = make(map[string]*Line)

	line = NewLine("fname", "2", "${outer.${inner}}", nil)
	lines[0] = line
	determineUsedVariables(lines)
	a.assertEqual(3, len(G.mkContext.varuse))
	a.assertEqual(line, G.mkContext.varuse["inner"])
	a.assertEqual(line, G.mkContext.varuse["outer."])
	a.assertEqual(line, G.mkContext.varuse["outer.*"])
}
