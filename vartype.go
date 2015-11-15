package main

import (
	"path"
)

// A Vartype in pkglint is a combination of a data type and a permission
// specification. Further details can be found in the chapter ``The pkglint
// type system'' of the pkglint book.
type Vartype struct {
	kindOfList KindOfList
	checker    *VarChecker
	aclEntries []AclEntry
	guessed    Guessed
}

type KindOfList int

const (
	LK_NONE KindOfList = iota
	LK_SPACE
	LK_SHELL
)

type AclEntry struct {
	glob        string
	permissions string
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

type VarChecker struct {
	name    string
	checker func(*VartypeCheck)
}

func (vc *VarChecker) IsEnum() bool {
	return hasPrefix(vc.name, "enum:")
}
func (vc *VarChecker) HasEnum(value string) bool {
	return !matches(value, `\s`) && contains(vc.name, " "+value+" ")
}
func (vc *VarChecker) AllowedEnums() string {
	return vc.name[5:]
}

var (
	CheckvarAwkCommand             = &VarChecker{"AwkCommand", (*VartypeCheck).AwkCommand}
	CheckvarBasicRegularExpression = &VarChecker{"BasicRegularExpression", (*VartypeCheck).BasicRegularExpression}
	CheckvarBuildlinkDepmethod     = &VarChecker{"BuildlinkDepmethod", (*VartypeCheck).BuildlinkDepmethod}
	CheckvarBuildlinkDepth         = &VarChecker{"BuildlinkDepth", (*VartypeCheck).BuildlinkDepth}
	CheckvarCategory               = &VarChecker{"Category", (*VartypeCheck).Category}
	CheckvarCFlag                  = &VarChecker{"CFlag", (*VartypeCheck).CFlag}
	CheckvarComment                = &VarChecker{"Comment", (*VartypeCheck).Comment}
	CheckvarDependency             = &VarChecker{"Dependency", (*VartypeCheck).Dependency}
	CheckvarDependencyWithPath     = &VarChecker{"DependencyWithPath", (*VartypeCheck).DependencyWithPath}
	CheckvarDistSuffix             = &VarChecker{"DistSuffix", (*VartypeCheck).DistSuffix}
	CheckvarEmulPlatform           = &VarChecker{"EmulPlatform", (*VartypeCheck).EmulPlatform}
	CheckvarFetchURL               = &VarChecker{"FetchURL", (*VartypeCheck).FetchURL}
	CheckvarFilename               = &VarChecker{"Filename", (*VartypeCheck).Filename}
	CheckvarFilemask               = &VarChecker{"Filemask", (*VartypeCheck).Filemask}
	CheckvarFileMode               = &VarChecker{"FileMode", (*VartypeCheck).FileMode}
	CheckvarIdentifier             = &VarChecker{"Identifier", (*VartypeCheck).Identifier}
	CheckvarInteger                = &VarChecker{"Integer", (*VartypeCheck).Integer}
	CheckvarLdFlag                 = &VarChecker{"LdFlag", (*VartypeCheck).LdFlag}
	CheckvarLicense                = &VarChecker{"License", (*VartypeCheck).License}
	CheckvarMailAddress            = &VarChecker{"MailAddress", (*VartypeCheck).MailAddress}
	CheckvarMessage                = &VarChecker{"Message", (*VartypeCheck).Message}
	CheckvarOption                 = &VarChecker{"Option", (*VartypeCheck).Option}
	CheckvarPathlist               = &VarChecker{"Pathlist", (*VartypeCheck).Pathlist}
	CheckvarPathmask               = &VarChecker{"Pathmask", (*VartypeCheck).Pathmask}
	CheckvarPathname               = &VarChecker{"Pathname", (*VartypeCheck).Pathname}
	CheckvarPerl5Packlist          = &VarChecker{"Perl5Packlist", (*VartypeCheck).Perl5Packlist}
	CheckvarPkgName                = &VarChecker{"PkgName", (*VartypeCheck).PkgName}
	CheckvarPkgPath                = &VarChecker{"PkgPath", (*VartypeCheck).PkgPath}
	CheckvarPkgOptionsVar          = &VarChecker{"PkgOptionsVar", (*VartypeCheck).PkgOptionsVar}
	CheckvarPkgRevision            = &VarChecker{"PkgRevision", (*VartypeCheck).PkgRevision}
	CheckvarPlatformTriple         = &VarChecker{"PlatformTriple", (*VartypeCheck).PlatformTriple}
	CheckvarPrefixPathname         = &VarChecker{"PrefixPathname", (*VartypeCheck).PrefixPathname}
	CheckvarPythonDependency       = &VarChecker{"PythonDependency", (*VartypeCheck).PythonDependency}
	CheckvarRelativePkgDir         = &VarChecker{"RelativePkgDir", (*VartypeCheck).RelativePkgDir}
	CheckvarRelativePkgPath        = &VarChecker{"RelativePkgPath", (*VartypeCheck).RelativePkgPath}
	CheckvarRestricted             = &VarChecker{"Restricted", (*VartypeCheck).Restricted}
	CheckvarSedCommand             = &VarChecker{"SedCommand", (*VartypeCheck).SedCommand}
	CheckvarSedCommands            = &VarChecker{"SedCommands", (*VartypeCheck).SedCommands}
	CheckvarShellCommand           = &VarChecker{"ShellCommand", nil}
	CheckvarShellWord              = &VarChecker{"ShellWord", nil}
	CheckvarStage                  = &VarChecker{"Stage", (*VartypeCheck).Stage}
	CheckvarString                 = &VarChecker{"String", (*VartypeCheck).String}
	CheckvarTool                   = &VarChecker{"Tool", (*VartypeCheck).Tool}
	CheckvarUnchecked              = &VarChecker{"Unchecked", (*VartypeCheck).Unchecked}
	CheckvarURL                    = &VarChecker{"URL", (*VartypeCheck).URL}
	CheckvarUserGroupName          = &VarChecker{"UserGroupName", (*VartypeCheck).UserGroupName}
	CheckvarVarname                = &VarChecker{"Varname", (*VartypeCheck).Varname}
	CheckvarVersion                = &VarChecker{"Version", (*VartypeCheck).Version}
	CheckvarWrapperReorder         = &VarChecker{"WrapperReorder", (*VartypeCheck).WrapperReorder}
	CheckvarWrapperTransform       = &VarChecker{"WrapperTransform", (*VartypeCheck).WrapperTransform}
	CheckvarWrkdirSubdirectory     = &VarChecker{"WrkdirSubdirectory", (*VartypeCheck).WrkdirSubdirectory}
	CheckvarWrksrcSubdirectory     = &VarChecker{"WrksrcSubdirectory", (*VartypeCheck).WrksrcSubdirectory}
	CheckvarYes                    = &VarChecker{"Yes", (*VartypeCheck).Yes}
	CheckvarYesNo                  = &VarChecker{"YesNo", (*VartypeCheck).YesNo}
	CheckvarYesNo_Indirectly       = &VarChecker{"YesNo_Indirectly", (*VartypeCheck).YesNo_Indirectly}
)

func init() {
	CheckvarShellCommand.checker = (*VartypeCheck).ShellCommand
	CheckvarShellWord.checker = (*VartypeCheck).ShellWord
}
