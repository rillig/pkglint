package main

// Whether a variable is used correctly depends on many things:
//
// * The variable’s data type, as defined in vardefs.go.
// * Whether the variable is accessed at loading time (when the
//   Makefiles are parsed) or at run time (when the shell commands are
//   run). Especially at load time, there are several points of time
//   (e.g. the bsd.pkg.mk file is loaded at the very end, therefore
//   the variables that are defined there cannot be used at load time.)
// * When used on the right-hand side of an assigment, the variable can
//   represent a list of words, a single word or even only part of a
//   word. This distinction decides upon the correct use of the :Q
//   operator.
// * When used in shell commands, the variable can appear inside single
//   quotes, double quotes, backticks or some combination thereof. This
//   also influences whether the variable is correctly used.
// * When used in preprocessing statements like .if or .for, the other
//   operands of that statement should fit to the variable and are
//   checked against the variable type. For example, comparing OPSYS to
//   x86_64 doesn’t make sense.

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
