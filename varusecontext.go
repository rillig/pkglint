package main

import (
	"fmt"
)

// VarUseContext defines the context in which a variable is defined
// or used. Whether that is allowed depends on:
//
// * The variable’s data type, as defined in vardefs.go.
// * When used on the right-hand side of an assigment, the variable can
//   represent a list of words, a single word or even only part of a
//   word. This distinction decides upon the correct use of the :Q
//   operator.
// * When used in preprocessing statements like .if or .for, the other
//   operands of that statement should fit to the variable and are
//   checked against the variable type. For example, comparing OPSYS to
//   x86_64 doesn’t make sense.
type VarUseContext struct {
	vartype *Vartype
	time    vucTime
	quoting vucQuoting
	extent  vucExtent
}

type vucTime uint8

const (
	vucTimeUnknown vucTime = iota

	// When Makefiles are loaded, the operators := and != are evaluated,
	// as well as the conditionals .if, .elif and .for.
	// During loading, not all variables are available yet.
	// Variable values are still subject to change, especially lists.
	vucTimeParse

	// All files have been read, all variables can be referenced.
	// Variable values don’t change anymore.
	vucTimeRun
)

func (t vucTime) String() string { return [...]string{"unknown", "parse", "run"}[t] }

// The quoting context in which the variable is used.
// Depending on this context, the modifiers :Q or :M can be allowed or not.
type vucQuoting uint8

const (
	vucQuotUnknown vucQuoting = iota
	vucQuotPlain              // Example: echo LOCALBASE=${LOCALBASE}
	vucQuotDquot              // Example: echo "The version is ${PKGVERSION}."
	vucQuotSquot              // Example: echo 'The version is ${PKGVERSION}.'
	vucQuotBackt              // Example: echo \`sed 1q ${WRKSRC}/README\`

	// The .for loop in Makefiles. This is the only place where
	// variables are split on whitespace. Everywhere else (:Q, :M)
	// they are split like in the shell.
	//
	// Example: .for f in ${EXAMPLE_FILES}
	vucQuotFor
)

func (q vucQuoting) String() string {
	return [...]string{"unknown", "plain", "dquot", "squot", "backt", "mk-for"}[q]
}

type vucExtent uint8

const (
	vucExtentUnknown  vucExtent = iota
	vucExtentWord               // Example: echo ${LOCALBASE}
	vucExtentWordpart           // Example: echo LOCALBASE=${LOCALBASE}
)

func (e vucExtent) String() string {
	return [...]string{"unknown", "word", "wordpart"}[e]
}

func (vuc *VarUseContext) String() string {
	typename := "no-type"
	if vuc.vartype != nil {
		typename = vuc.vartype.String()
	}
	return fmt.Sprintf("(%s %s %s %s)", vuc.time, typename, vuc.quoting, vuc.extent)
}
