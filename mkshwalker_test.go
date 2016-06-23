package main

import (
	"fmt"
	"gopkg.in/check.v1"
)

func (s *Suite) Test_MkShWalker_ForEachSimpleCommand(c *check.C) {
	list, err := parseShellProgram("if condition; then action; else case selector in pattern) case-item-action ;; esac; fi")
	if c.Check(err, check.IsNil) && c.Check(list, check.NotNil) {
		var commands []string
		(*MkShWalker).ForEachSimpleCommand(nil, list, func(cmd *MkShSimpleCommand) {
			cmdDescr := fmt.Sprintf("%v", cmd)
			commands = append(commands, string(cmdDescr))
		})
		c.Check(commands, deepEquals, []string{
			"&{[] ShToken([\"condition\"]) [] []}",
			"&{[] ShToken([\"action\"]) [] []}",
			"&{[] ShToken([\"case-item-action\"]) [] []}"})
	}
}
