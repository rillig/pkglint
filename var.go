package pkglint

// Var describes a variable in a Makefile snippet.
//
// It keeps track of all places where the variable is accessed or modified (see
// ReadLocations, WriteLocations) and provides information for further static
// analysis, such as:
//
// * Whether the variable value is constant, and if so, what the constant value
// is (see Constant, ConstantValue).
//
// * What its (approximated) value is, either including values from the pkgsrc
// infrastructure (see ValueInfra) or excluding them (Value).
//
// * On which other variables this variable depends (see Conditional,
// ConditionalVars).
type Var struct {
	Name string

	//  0 = neither written nor read
	//  1 = constant
	//  2 = constant and read; further writes will make it non-constant
	//  3 = not constant anymore
	constantState uint8
	constantValue string

	value      string
	valueInfra string

	readLocations   []MkLine
	writeLocations  []MkLine
	conditionalVars StringSet
	refs            StringSet
}

func NewVar(name string) *Var {
	return &Var{name, 0, "", "", "", nil, nil, NewStringSet(), NewStringSet()}
}

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
func (v *Var) Refs() []string {
	return v.refs.Elements
}

// AddRef marks this variable as being dependent on the given variable name.
// This can be used for the .for loops mentioned in Refs.
func (v *Var) AddRef(varname string) {
	v.refs.Add(varname)
}

// Constant returns whether the variable's value is a constant,
// without being dependent on any other variable.
//
// Multiple assignments (such as VAR=1, VAR+=2, VAR+=3) are considered to
// form a single constant as well, as long as the variable is not read before
// or in-between these assignments. The definition of "read" is very strict
// here since every mention of the variable counts. This may prevent some
// essentially constant values from being detected as such, but these can
// be added later.
//
// TODO: Simple .for loops that append to the variable are ok as well.
//  (This needs to be worded more precisely since that part potentially
//  adds a lot of complexity to the whole data structure.)
func (v *Var) Constant() bool {
	return v.constantState == 1 || v.constantState == 2
}

// ConstantValue returns the constant value of the variable.
// It is only allowed when Constant() returns true.
func (v *Var) ConstantValue() string {
	G.Assertf(v.Constant(), "Variable must be constant.")
	return v.constantValue
}

// Value returns the (approximated) value of the variable, taking into account
// all variable assignments that happen outside the pkgsrc infrastructure.
//
// For variables that are conditionally assigned (as in .if/.else), the
// returned value is not reliable. It may be the value from either branch, or
// even the combined value of both branches.
//
// See Constant and ConstantValue for more reliable information.
func (v *Var) Value() string {
	return v.value
}

// ValueInfra returns the (approximated) value of the variable, taking into
// account all variable assignments from the package, the user and the pkgsrc
// infrastructure.
//
// For variables that are conditionally assigned (as in .if/.else), the
// returned value is not reliable. It may be the value from either branch, or
// even the combined value of both branches.
//
// See Constant and ConstantValue for more reliable information, but these
// ignore assignments from the infrastructure.
func (v *Var) ValueInfra() string {
	return v.valueInfra
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
	v.constantState = [...]uint8{3, 2, 2, 3}[v.constantState]
}

func (v *Var) Write(mkline MkLine, conditionVarnames ...string) {
	G.Assertf(mkline.Varname() == v.Name, "wrong variable name")

	v.writeLocations = append(v.writeLocations, mkline)
	v.conditionalVars.AddAll(conditionVarnames)
	v.refs.AddAll(mkline.DetermineUsedVariables())
	v.refs.AddAll(conditionVarnames)

	v.update(mkline, &v.valueInfra)
	if !v.isInfra(mkline) {
		v.update(mkline, &v.value)
	}
	v.updateConstantValue(mkline)
}

func (v *Var) isInfra(mkline MkLine) bool {
	rel := G.Pkgsrc.ToRel(mkline.Filename)
	return hasPrefix(rel, "mk/") || hasPrefix(rel, "wip/mk/")
}

func (v *Var) update(mkline MkLine, update *string) {
	firstWrite := len(v.writeLocations) == 1
	if v.Conditional() && !firstWrite {
		return
	}

	value := mkline.Value()
	switch mkline.Op() {
	case opAssign, opAssignEval:
		*update = value

	case opAssignDefault:
		if firstWrite {
			*update = value
		}

	case opAssignAppend:
		*update += " " + value

	case opAssignShell:
		// Ignore these for now.
		// Later it might be useful to parse the shell commands to
		// evaluate simple commands like "test && echo yes || echo no".
	}
}

func (v *Var) updateConstantValue(mkline MkLine) {
	if v.constantState == 3 {
		return
	}

	// For now, just mark the variable as being non-constant if it depends
	// on other variables. Later this can be made more sophisticated, but
	// needs a few precautions:
	// * For the := operator, the current value needs to be resolved.
	//   This in turn requires the proper scope for resolving variable
	//   references. Furthermore, the variable must be constant at this
	//   point, while later changes can be ignored.
	// * For the other operators, the referenced variables must be still
	//   be constant at the end of loading the complete package.
	// * The documentation of Constant would need to be adjusted.
	value := mkline.Value()
	if v.Conditional() || value != mkline.WithoutMakeVariables(value) {
		v.constantState = 3
		v.constantValue = ""
		return
	}

	switch mkline.Op() {
	case opAssign, opAssignEval:
		v.constantValue = value

	case opAssignDefault:
		if v.constantState == 0 {
			v.constantValue = value
		}

	case opAssignAppend:
		v.constantValue += " " + value

	case opAssignShell:
		v.constantState = 2
		v.constantValue = ""
	}

	v.constantState = [...]uint8{1, 1, 3, 3}[v.constantState]
}
