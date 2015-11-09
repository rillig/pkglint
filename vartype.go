package main

import (
	"path"
)

type Vartype struct {
	kindOfList    KindOfList
	basicType     string
	enumValues    map[string]bool
	enumValuesStr string
	aclEntries    []AclEntry
	guessed       Guessed
}

type Guessed bool
const (
	GUESSED Guessed = false
	NOT_GUESSED Guessed = true
)

func newBasicVartype(kindOfList KindOfList, basicType string, aclEntries []AclEntry, guessed Guessed) *Vartype {
	return &Vartype{kindOfList, basicType, nil, "", aclEntries, guessed}
}

func newEnumVartype(kindOfList KindOfList, enumValues string, aclEntries []AclEntry, guessed Guessed) *Vartype {
	emap := make(map[string]bool)
	for _, evalue := range splitOnSpace(enumValues) {
		emap[evalue] = true
	}
	return &Vartype{kindOfList, "", emap, enumValues, aclEntries, guessed}
}

func (self *Vartype) effectivePermissions(fname string) string {
	for _, aclEntry := range self.aclEntries {
		if m, _ := path.Match(aclEntry.glob, fname); m {
			return aclEntry.permissions
		}
	}
	return ""
}

// Returns the union of all possible permissions. This can be used to
// check whether a variable may be defined or used at all, or if it is
// read-only.
func (self *Vartype) union() (perms string) {
	for _, aclEntry := range self.aclEntries {
		perms += aclEntry.permissions
	}
	return
}

// This distinction between “real lists” and “considered a list” makes
// the implementation of checklineMkVartype easier.
func (self *Vartype) isConsideredList() bool {
	switch {
	case self.kindOfList == LK_EXTERNAL:
		return true
	case self.kindOfList == LK_INTERNAL:
		return false
	case self.basicType == "BuildlinkPackages":
		return true
	case self.basicType == "SedCommands":
		return true
	case self.basicType == "ShellCommand":
		return true
	default:
		return false
	}
}

func (self *Vartype) mayBeAppendedTo() bool {
	return self.kindOfList != LK_NONE ||
		self.basicType == "AwkCommand" ||
		self.basicType == "BuildlinkPackages" ||
		self.basicType == "SedCommands"
}

func (self *Vartype) String() string {
	switch self.kindOfList {
	case LK_NONE:
		return self.basicType
	case LK_INTERNAL:
		return "InternalList of " + self.basicType
	case LK_EXTERNAL:
		return "List of " + self.basicType
	default:
		panic("")
	}
}
