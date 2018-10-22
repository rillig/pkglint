package main

import (
	"gopkg.in/check.v1"
)

func NewMkVarUse(varname string, modifiers ...string) *MkVarUse {
	var mods []MkVarUseModifier
	for _, modifier := range modifiers {
		mods = append(mods, MkVarUseModifier{modifier})
	}
	return &MkVarUse{varname, mods}
}

func (s *Suite) Test_MkVarUse_Mod(c *check.C) {
	varuse := NewMkVarUse("varname", "Q")

	c.Check(varuse.Mod(), equals, ":Q")
}

func (list *MkShList) AddCommand(command *MkShCommand) *MkShList {
	pipeline := NewMkShPipeline(false, command)
	andOr := NewMkShAndOr(pipeline)
	return list.AddAndOr(andOr)
}
