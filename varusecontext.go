package main

type VarUseContext struct {
	time      VarUseContextTime
	vartype   *Vartype
	shellword VarUseContextShellword
	extent    VarUseContextExtent
}

// The various contexts in which make(1) variables can appear in pkgsrc.
// Further details can be found in the chapter “The pkglint type system”
// of the pkglint book.
type VarUseContextTime int

const (
	VUC_TIME_UNKNOWN VarUseContextTime = iota
	VUC_TIME_LOAD
	VUC_TIME_RUN
)

type VarUseContextShellword int

const (
	VUC_SHW_UNKNOWN VarUseContextShellword = iota
	VUC_SHW_PLAIN
	VUC_SHW_DQUOT
	VUC_SHW_SQUOT
	VUC_SHW_BACKT
	VUC_SHW_FOR
)

type VarUseContextExtent int

const (
	VUC_EXTENT_UNKNOWN VarUseContextExtent = iota
	VUC_EXT_FULL
	VUC_EXT_WORD
	VUC_EXT_WORDPART
)

func (self *VarUseContext) String() string {
	typename := "no-type"
	if self.vartype != nil {
		typename = self.vartype.String()
	}
	return sprintf("(%s %s %s %s)",
		[]string{"unknown-time", "load-time", "run-time"}[self.time],
		typename,
		[]string{"none", "plain", "dquot", "squot", "backt", "for"}[self.shellword],
		[]string{"unknown", "full", "word", "word-part"}[self.extent])
}
