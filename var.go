package pkglint

// Var describes a variable in a Makefile snippet.
//
// It provides information about the possible values and all places where the
// variable is accessed and modified.
//
// TODO: Remove this type in June 2019 if it is still a stub.
type Var struct {
	Name string
	Type *Vartype

	//  0 = not yet assigned
	//  1 = literal
	//  2 = not anymore
	//  3 = not yet assigned, but already read
	//  4 = literal and read
	// TODO: The exact definition of "read", "accessed", "referenced" is important here.
	literalValueState uint8
	literalValue      string
	readLocations     []MkLine
	writeLocations    []MkLine
	conditionalVars   StringSet
}

func NewVar(name string) *Var { return &Var{name, nil, 0, "", nil, nil, NewStringSet()} }

// Conditional returns whether the variable value depends on other variables.
func (v *Var) Conditional() bool {
	return v.conditionalVars.Size() > 0
}

// ConditionalVars returns all variables in conditions on which the value of
// this variable depends.
//
// The returned slice must not be modified.
func (v *Var) ConditionalVars() []string {
	return v.conditionalVars.Elements
}

// TODO: Refs
//
// Refs returns all variables on which this variable depends. These are:
//
// Variables that are referenced in the value, such as in VAR=${OTHER}.
//
// Variables that are used in conditions that enclose one of the assignments
// to this variable, such as .if ${OPSYS} == NetBSD.
//
// Variables that are used in .for loops in which this variable is assigned
// a value, such as DIRS in:
//  .for dir in ${DIRS}
//  VAR+=${dir}
//  .endfor

// Literal returns whether the variable's value is a constant,
// without being dependent on any other variable.
//
// Multiple assignments (such as VAR=1, VAR+=2, VAR+=3) are considered literals
// as well.
//
// TODO: As long as the variable is not used in-between these assignments.
//  That is, no .include or .if may appear there, and no ::= modifier may
//  access this variable.
//  Note: being referenced in other variables is not the same as the value
//  being actually used. The check for being actually used would need to
//  be able to check transitive references.
//
// TODO: Simple .for loops that append to the variable are ok as well.
//  (This needs to be worded more precisely since that part potentially
//  adds a lot of complexity to the whole data structure.)
func (v *Var) Literal() bool {
	return v.literalValueState == 1 || v.literalValueState == 4
}

// LiteralValue returns the value of the literal.
// It is only allowed when Literal() returns true.
func (v *Var) LiteralValue() string {
	G.Assertf(v.Literal(), "Variable must have a literal value.")
	return v.literalValue
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

// ReadLocations returns the locations where the variable is read, such as
// in ${VAR} or defined(VAR) or empty(VAR).
//
// Indirect uses through other variables (such as VAR2=${VAR}, VAR3=${VAR2})
// are not listed.
func (v *Var) ReadLocations() []MkLine {
	return v.readLocations
}

// WriteLocations returns the locations where the variable is modified.
func (v *Var) WriteLocations() []MkLine {
	return v.writeLocations
}

func (v *Var) Read(mkline MkLine) {
	v.readLocations = append(v.readLocations, mkline)
	v.literalValueState = [...]uint8{3, 4, 2, 3, 4}[v.literalValueState]
}

func (v *Var) Write(mkline MkLine, conditionVarnames ...string) {
	G.Assertf(mkline.Varname() == v.Name, "wrong variable name")

	v.writeLocations = append(v.writeLocations, mkline)
	for _, cond := range conditionVarnames {
		v.conditionalVars.Add(cond)
	}

	if v.literalValueState == 2 {
		return
	}

	// For now, just mark the variable as being non-literal if it depends
	// on other variables. Later this can be made more sophisticated, but
	// then the current value needs to be resolved, and for that this method
	// would need to be passed the proper scope for resolving variable references.
	// Plus, the documentation of Literal needs to be adjusted.
	value := mkline.Value()
	if len(conditionVarnames) > 0 || value != mkline.WithoutMakeVariables(value) {
		v.literalValueState = 2
		v.literalValue = ""
		return
	}

	switch mkline.Op() {
	case opAssign, opAssignEval:
		v.literalValue = value

	case opAssignDefault:
		if v.literalValueState == 0 {
			v.literalValue = value
		}

	case opAssignAppend:
		v.literalValue += " " + value

	case opAssignShell:
		v.literalValueState = 2
		v.literalValue = ""
	}

	v.literalValueState = [...]uint8{1, 1, 2, 2, 2}[v.literalValueState]
}
