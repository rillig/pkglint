package main

import (
	"path"
)

type Vartype struct {
	kindOfList KindOfList
	checker    *VarChecker
	aclEntries []AclEntry
	guessed    Guessed
}

type Guessed bool

const (
	NOT_GUESSED Guessed = false
	GUESSED     Guessed = true
)

func (self *Vartype) effectivePermissions(fname string) string {
	for _, aclEntry := range self.aclEntries {
		if m, _ := path.Match(aclEntry.glob, path.Base(fname)); m {
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
	case self.kindOfList == LK_SHELL:
		return true
	case self.kindOfList == LK_SPACE:
		return false
	case self.checker.name == "BuildlinkPackages":
		return true
	case self.checker.name == "SedCommands":
		return true
	case self.checker.name == "ShellCommand":
		return true
	default:
		return false
	}
}

func (self *Vartype) mayBeAppendedTo() bool {
	return self.kindOfList != LK_NONE ||
		self.checker.name == "AwkCommand" ||
		self.checker.name == "BuildlinkPackages" ||
		self.checker.name == "SedCommands"
}

func (self *Vartype) String() string {
	switch self.kindOfList {
	case LK_NONE:
		return self.checker.name
	case LK_SPACE:
		return "SpaceList of " + self.checker.name
	case LK_SHELL:
		return "ShellList of " + self.checker.name
	default:
		panic("")
	}
}
