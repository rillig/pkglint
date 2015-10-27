package main

import (
	"fmt"
)

func parseAclEntries(args []string) []AclEntry {
	result := make([]AclEntry, 0)
	for _, arg := range args {
		m := mustMatch(`^([\w.*]+|_):([adpsu]*)$`, arg)
		glob, perms := m[1], m[2]
		result = append(result, AclEntry{glob, perms})
	}
	return result
}

func loadVartypesBasictypes() {
	panic("not implemented; don’t use self-grep")
}

type NeedsQuoting int

const (
	NQ_NO NeedsQuoting = iota
	NQ_YES
	NQ_DOESNT_MATTER
	NQ_DONT_KNOW
)

var vnqSafeTypes = map[string]bool{
	"DistSuffix": true,
	"FileMode":   true, "Filename": true,
	"Identifier": true,
	"Option":     true,
	"Pathname":   true, "PkgName": true, "PkgOptionsVar": true, "PkgRevision": true,
	"RelativePkgDir": true, "RelativePkgPath": true,
	"UserGroupName": true,
	"Varname":       true, "Version": true,
	"WrkdirSubdirectory": true,
}

func variableNeedsQuoting(line *Line, varname string, context *VarUseContext) NeedsQuoting {
	_ = GlobalVars.opts.optDebugTrace && line.logDebug(fmt.Sprintf("variableNeedsQuoting: %s, %#v", varname, context))

	vartype := getVariableType(line, varname)
	if vartype == nil || context.vartype == nil {
		return NQ_DONT_KNOW
	}
	logError(NO_FILE, NO_LINES, "not implemented")
	return NQ_DONT_KNOW
}
