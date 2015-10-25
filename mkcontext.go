package main

// Context of the Makefile that is currently checked.
type MkContext struct {
	forVars     map[string]bool  // The variables currently used in .for loops
	indentation []int            // Indentation depth of preprocessing directives
	target      string           // Current make(1) target
	vardef      map[string]*Line // varname => line; for all variables that are defined in the current file
	varuse      map[string]*Line // varname => line; for all variables that are used in the current file
	buildDefs   map[string]bool  // Variables that are registered in BUILD_DEFS, to ensure that all user-defined variables are added to it.
	plistVars   map[string]bool  // Variables that are registered in PLIST_VARS, to ensure that all user-defined variables are added to it.
	tools       map[string]bool  // Set of tools that are declared to be used.
}
