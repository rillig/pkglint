package pkglint

// Project defines a larger collection of makefiles that establishes
// conventions on variable names, variable types, deprecated variables
// and so on.
//
// Pkglint initially started as a pkgsrc-only tool, but since it parses,
// analyzes and formats makefiles, these parts are useful outside pkgsrc
// as well.
type Project interface {

	// Deprecated determines whether the variable is deprecated,
	// and if so, what should be done instead.
	Deprecated(varname string) string

	// Types determines the types of variables.
	Types() *VarTypeRegistry
}

type NetBSDProject struct {
	types VarTypeRegistry
}

func NewNetBSDProject() *NetBSDProject {
	return &NetBSDProject{
		NewVarTypeRegistry(),
	}
}

func (p NetBSDProject) Deprecated(string) string {
	return ""
}

func (p NetBSDProject) Types() *VarTypeRegistry {
	return &p.types
}
