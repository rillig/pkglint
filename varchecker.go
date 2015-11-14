package main

type VarChecker struct {
	name    string
	checker func(*CheckVartype)
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
	CheckvarAwkCommand             = &VarChecker{"AwkCommand", (*CheckVartype).AwkCommand}
	CheckvarBasicRegularExpression = &VarChecker{"BasicRegularExpression", (*CheckVartype).BasicRegularExpression}
	CheckvarBrokenIn               = &VarChecker{"BrokenIn", (*CheckVartype).BrokenIn}
	CheckvarBuildlinkDepmethod     = &VarChecker{"BuildlinkDepmethod", (*CheckVartype).BuildlinkDepmethod}
	CheckvarBuildlinkDepth         = &VarChecker{"BuildlinkDepth", (*CheckVartype).BuildlinkDepth}
	CheckvarCategory               = &VarChecker{"Category", (*CheckVartype).Category}
	CheckvarCFlag                  = &VarChecker{"CFlag", (*CheckVartype).CFlag}
	CheckvarComment                = &VarChecker{"Comment", (*CheckVartype).Comment}
	CheckvarDependency             = &VarChecker{"Dependency", (*CheckVartype).Dependency}
	CheckvarDependencyWithPath     = &VarChecker{"DependencyWithPath", (*CheckVartype).DependencyWithPath}
	CheckvarDistSuffix             = &VarChecker{"DistSuffix", (*CheckVartype).DistSuffix}
	CheckvarEmulPlatform           = &VarChecker{"EmulPlatform", (*CheckVartype).EmulPlatform}
	CheckvarFetchURL               = &VarChecker{"FetchURL", (*CheckVartype).FetchURL}
	CheckvarFilename               = &VarChecker{"Filename", (*CheckVartype).Filename}
	CheckvarFilemask               = &VarChecker{"Filemask", (*CheckVartype).Filemask}
	CheckvarFileMode               = &VarChecker{"FileMode", (*CheckVartype).FileMode}
	CheckvarIdentifier             = &VarChecker{"Identifier", (*CheckVartype).Identifier}
	CheckvarInteger                = &VarChecker{"Integer", (*CheckVartype).Integer}
	CheckvarLdFlag                 = &VarChecker{"LdFlag", (*CheckVartype).LdFlag}
	CheckvarLicense                = &VarChecker{"License", (*CheckVartype).License}
	CheckvarMailAddress            = &VarChecker{"MailAddress", (*CheckVartype).MailAddress}
	CheckvarMessage                = &VarChecker{"Message", (*CheckVartype).Message}
	CheckvarOption                 = &VarChecker{"Option", (*CheckVartype).Option}
	CheckvarPathlist               = &VarChecker{"Pathlist", (*CheckVartype).Pathlist}
	CheckvarPathmask               = &VarChecker{"Pathmask", (*CheckVartype).Pathmask}
	CheckvarPathname               = &VarChecker{"Pathname", (*CheckVartype).Pathname}
	CheckvarPerl5Packlist          = &VarChecker{"Perl5Packlist", (*CheckVartype).Perl5Packlist}
	CheckvarPkgName                = &VarChecker{"PkgName", (*CheckVartype).PkgName}
	CheckvarPkgPath                = &VarChecker{"PkgPath", (*CheckVartype).PkgPath}
	CheckvarPkgOptionsVar          = &VarChecker{"PkgOptionsVar", (*CheckVartype).PkgOptionsVar}
	CheckvarPkgRevision            = &VarChecker{"PkgRevision", (*CheckVartype).PkgRevision}
	CheckvarPlatformTriple         = &VarChecker{"PlatformTriple", (*CheckVartype).PlatformTriple}
	CheckvarPrefixPathname         = &VarChecker{"PrefixPathname", (*CheckVartype).PrefixPathname}
	CheckvarPythonDependency       = &VarChecker{"PythonDependency", (*CheckVartype).PythonDependency}
	CheckvarRelativePkgDir         = &VarChecker{"RelativePkgDir", (*CheckVartype).RelativePkgDir}
	CheckvarRelativePkgPath        = &VarChecker{"RelativePkgPath", (*CheckVartype).RelativePkgPath}
	CheckvarRestricted             = &VarChecker{"Restricted", (*CheckVartype).Restricted}
	CheckvarSedCommand             = &VarChecker{"SedCommand", (*CheckVartype).SedCommand}
	CheckvarSedCommands            = &VarChecker{"SedCommands", (*CheckVartype).SedCommands}
	CheckvarShellCommand           = &VarChecker{"ShellCommand", nil}
	CheckvarShellWord              = &VarChecker{"ShellWord", nil}
	CheckvarStage                  = &VarChecker{"Stage", (*CheckVartype).Stage}
	CheckvarString                 = &VarChecker{"String", (*CheckVartype).String}
	CheckvarTool                   = &VarChecker{"Tool", (*CheckVartype).Tool}
	CheckvarUnchecked              = &VarChecker{"Unchecked", (*CheckVartype).Unchecked}
	CheckvarURL                    = &VarChecker{"URL", (*CheckVartype).URL}
	CheckvarUserGroupName          = &VarChecker{"UserGroupName", (*CheckVartype).UserGroupName}
	CheckvarVarname                = &VarChecker{"Varname", (*CheckVartype).Varname}
	CheckvarVersion                = &VarChecker{"Version", (*CheckVartype).Version}
	CheckvarWrapperReorder         = &VarChecker{"WrapperReorder", (*CheckVartype).WrapperReorder}
	CheckvarWrapperTransform       = &VarChecker{"WrapperTransform", (*CheckVartype).WrapperTransform}
	CheckvarWrkdirSubdirectory     = &VarChecker{"WrkdirSubdirectory", (*CheckVartype).WrkdirSubdirectory}
	CheckvarWrksrcSubdirectory     = &VarChecker{"WrksrcSubdirectory", (*CheckVartype).WrksrcSubdirectory}
	CheckvarYes                    = &VarChecker{"Yes", (*CheckVartype).Yes}
	CheckvarYesNo                  = &VarChecker{"YesNo", (*CheckVartype).YesNo}
	CheckvarYesNo_Indirectly       = &VarChecker{"YesNo_Indirectly", (*CheckVartype).YesNo_Indirectly}
)

func init() {
	CheckvarShellCommand.checker = (*CheckVartype).ShellCommand
	CheckvarShellWord.checker = (*CheckVartype).ShellWord
}
