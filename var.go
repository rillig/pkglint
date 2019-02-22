package pkglint

// Var describes a variable in a Makefile snippet.
//
// It provides information about the possible values and all places where the
// variable is accessed.
//
// TODO: Remove this type in June 2019 if it is still a stub.
type Var struct {
	Name         string
	Type         *Vartype
	literalValue *string
}

func NewVar(name string) *Var { return &Var{name, nil, nil} }

// Conditional returns whether the variable value depends on other variables.
func (v *Var) Conditional() bool {
	G.Assertf(false, "Not implemented.")
	return false
}

// ConditionalVars returns the variables on which the value of this variable depends.
// Typical cases are:
//
// * references in the value (VAR=${OTHER}),
//
// * conditions (.if ${OPSYS} == NetBSD), and
//
// * loops (.for dir in ${DIRS}; VAR+=${dir}; .endfor).
func (v *Var) ConditionalVars() []*Var {
	G.Assertf(false, "Not implemented.")
	return nil
}

// Literal returns whether the variable is only ever assigned a single value,
// without being dependent on any other variable.
//
// Multiple assignments (such as VAR=1, VAR+=2, VAR+=3) are considered literals
// as well, as long as the variable is not used in-between these assignments.
// That is, no .include or .if may appear there, and no ::= modifier may
// be involved.
//
// Simple .for loops that append to the variable are ok as well.
func (v *Var) Literal() bool {
	return v.literalValue != nil
}

// LiteralValue returns the value of the literal.
// It is only allowed when Literal() returns true.
func (v *Var) LiteralValue() string {
	G.Assertf(v.Literal(), "Variable must have a literal value.")
	return *v.literalValue
}

// Value returns the (approximated) value of the variable, taking into account
// all variable assignments that happen outside the pkgsrc infrastructure.
//
// For variables that are conditionally assigned (as in .if/.else), the returned
// value is not reliable. It may be the value from either branch, or even the
// combined value of both branches.
//
// See Literal and LiteralValue for more reliable information.
func (v *Var) Value() string {
	G.Assertf(false, "Not implemented.")
	return ""
}

// ValueInfra returns the (approximated) value of the variable, taking into
// account all variable assignments from the package, the user and the pkgsrc
// infrastructure.
//
// For variables that are conditionally assigned (as in .if/.else), the returned
// value is not reliable. It may be the value from either branch, or even the
// combined value of both branches.
//
// See Literal and LiteralValue for more reliable information, but these ignore
// assignments from the infrastructure.
func (v *Var) ValueInfra() string {
	G.Assertf(false, "Not implemented.")
	return ""
}

// Uses returns the locations where the variable is used.
func (v *Var) Uses() []Location {
	G.Assertf(false, "Not implemented.")
	return nil
}

// Defs returns the locations where the variable is defined.
func (v *Var) Defs() []Location {
	G.Assertf(false, "Not implemented.")
	return nil
}
