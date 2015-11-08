package main

import(
	"fmt"
)

type VarUseContext struct {
	time      VarUseContextTime
	vartype   *Vartype
	shellword VarUseContextShellword
	extent    VarUseContextExtent
}

func (self *VarUseContext) String() string {
	typename := "no-type"
	if self.vartype != nil {
		typename = self.vartype.String()
	}
	return fmt.Sprintf("(%s %s %s %s)",
		[]string{"unknown-time", "load-time", "run-time"}[self.time],
		typename,
		[]string{"none", "plain", "dquot", "squot", "backt", "for"}[self.shellword],
		[]string{"unknown", "full", "word", "word-part"}[self.extent])
}
