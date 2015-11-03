package main

import (
	"testing"
)

func TestDetermineUsedVariables(t *testing.T) {
	a := Asserter{t}

	GlobalVars = GlobalVarsType{}
	GlobalVars.opts = &CmdOpts{}
	GlobalVars.mkContext = &MkContext{}
	GlobalVars.mkContext.varuse = make(map[string]*Line)

	line := NewLine("fname", "1", "${VAR}", nil)
	lines := make([]*Line, 1)
	lines[0] = line
	determineUsedVariables(lines)
	a.assertEqual(1, len(GlobalVars.mkContext.varuse))
	a.assertEqual(line, GlobalVars.mkContext.varuse["VAR"])

	GlobalVars.mkContext.varuse = make(map[string]*Line)

	line = NewLine("fname", "2", "${outer.${inner}}", nil)
	lines[0] = line
	determineUsedVariables(lines)
	a.assertEqual(3, len(GlobalVars.mkContext.varuse))
	a.assertEqual(line, GlobalVars.mkContext.varuse["inner"])
	a.assertEqual(line, GlobalVars.mkContext.varuse["outer."])
	a.assertEqual(line, GlobalVars.mkContext.varuse["outer.*"])
}
