package main

type VarChecker struct {
	name    string
	checker func(*VartypeCheckContext)
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
	CheckvarAwkCommand             = &VarChecker{"AwkCommand", (*VartypeCheckContext).AwkCommand}
	CheckvarBasicRegularExpression = &VarChecker{"BasicRegularExpression", (*VartypeCheckContext).BasicRegularExpression}
	CheckvarBrokenIn               = &VarChecker{"BrokenIn", (*VartypeCheckContext).BrokenIn}
	CheckvarBuildlinkDepmethod     = &VarChecker{"BuildlinkDepmethod", (*VartypeCheckContext).BuildlinkDepmethod}
	CheckvarBuildlinkDepth         = &VarChecker{"BuildlinkDepth", (*VartypeCheckContext).BuildlinkDepth}
	CheckvarCategory               = &VarChecker{"Category", (*VartypeCheckContext).Category}
	CheckvarCFlag                  = &VarChecker{"CFlag", (*VartypeCheckContext).CFlag}
	CheckvarComment                = &VarChecker{"Comment", (*VartypeCheckContext).Comment}
	CheckvarDependency             = &VarChecker{"Dependency", (*VartypeCheckContext).Dependency}
	CheckvarDependencyWithPath     = &VarChecker{"DependencyWithPath", (*VartypeCheckContext).DependencyWithPath}
	CheckvarDistSuffix             = &VarChecker{"DistSuffix", (*VartypeCheckContext).DistSuffix}
	CheckvarEmulPlatform           = &VarChecker{"EmulPlatform", (*VartypeCheckContext).EmulPlatform}
	CheckvarFetchURL               = &VarChecker{"FetchURL", (*VartypeCheckContext).FetchURL}
	CheckvarFilename               = &VarChecker{"Filename", (*VartypeCheckContext).Filename}
	CheckvarFilemask               = &VarChecker{"Filemask", (*VartypeCheckContext).Filemask}
	CheckvarFileMode               = &VarChecker{"FileMode", (*VartypeCheckContext).FileMode}
	CheckvarIdentifier             = &VarChecker{"Identifier", (*VartypeCheckContext).Identifier}
	CheckvarInteger                = &VarChecker{"Integer", (*VartypeCheckContext).Integer}
	CheckvarLdFlag                 = &VarChecker{"LdFlag", (*VartypeCheckContext).LdFlag}
	CheckvarLicense                = &VarChecker{"License", (*VartypeCheckContext).License}
	CheckvarMailAddress            = &VarChecker{"MailAddress", (*VartypeCheckContext).MailAddress}
	CheckvarMessage                = &VarChecker{"Message", (*VartypeCheckContext).Message}
	CheckvarOption                 = &VarChecker{"Option", (*VartypeCheckContext).Option}
	CheckvarPathlist               = &VarChecker{"Pathlist", (*VartypeCheckContext).Pathlist}
	CheckvarPathmask               = &VarChecker{"Pathmask", (*VartypeCheckContext).Pathmask}
	CheckvarPathname               = &VarChecker{"Pathname", (*VartypeCheckContext).Pathname}
	CheckvarPerl5Packlist          = &VarChecker{"Perl5Packlist", (*VartypeCheckContext).Perl5Packlist}
	CheckvarPkgName                = &VarChecker{"PkgName", (*VartypeCheckContext).PkgName}
	CheckvarPkgPath                = &VarChecker{"PkgPath", (*VartypeCheckContext).PkgPath}
	CheckvarPkgOptionsVar          = &VarChecker{"PkgOptionsVar", (*VartypeCheckContext).PkgOptionsVar}
	CheckvarPkgRevision            = &VarChecker{"PkgRevision", (*VartypeCheckContext).PkgRevision}
	CheckvarPlatformTriple         = &VarChecker{"PlatformTriple", (*VartypeCheckContext).PlatformTriple}
	CheckvarPrefixPathname         = &VarChecker{"PrefixPathname", (*VartypeCheckContext).PrefixPathname}
	CheckvarPythonDependency       = &VarChecker{"PythonDependency", (*VartypeCheckContext).PythonDependency}
	CheckvarRelativePkgDir         = &VarChecker{"RelativePkgDir", (*VartypeCheckContext).RelativePkgDir}
	CheckvarRelativePkgPath        = &VarChecker{"RelativePkgPath", (*VartypeCheckContext).RelativePkgPath}
	CheckvarRestricted             = &VarChecker{"Restricted", (*VartypeCheckContext).Restricted}
	CheckvarSedCommand             = &VarChecker{"SedCommand", (*VartypeCheckContext).SedCommand}
	CheckvarSedCommands            = &VarChecker{"SedCommands", (*VartypeCheckContext).SedCommands}
	CheckvarShellCommand           = &VarChecker{"ShellCommand", nil}
	CheckvarShellWord              = &VarChecker{"ShellWord", nil}
	CheckvarStage                  = &VarChecker{"Stage", (*VartypeCheckContext).Stage}
	CheckvarString                 = &VarChecker{"String", (*VartypeCheckContext).String}
	CheckvarTool                   = &VarChecker{"Tool", (*VartypeCheckContext).Tool}
	CheckvarUnchecked              = &VarChecker{"Unchecked", (*VartypeCheckContext).Unchecked}
	CheckvarURL                    = &VarChecker{"URL", (*VartypeCheckContext).URL}
	CheckvarUserGroupName          = &VarChecker{"UserGroupName", (*VartypeCheckContext).UserGroupName}
	CheckvarVarname                = &VarChecker{"Varname", (*VartypeCheckContext).Varname}
	CheckvarVersion                = &VarChecker{"Version", (*VartypeCheckContext).Version}
	CheckvarWrapperReorder         = &VarChecker{"WrapperReorder", (*VartypeCheckContext).WrapperReorder}
	CheckvarWrapperTransform       = &VarChecker{"WrapperTransform", (*VartypeCheckContext).WrapperTransform}
	CheckvarWrkdirSubdirectory     = &VarChecker{"WrkdirSubdirectory", (*VartypeCheckContext).WrkdirSubdirectory}
	CheckvarWrksrcSubdirectory     = &VarChecker{"WrksrcSubdirectory", (*VartypeCheckContext).WrksrcSubdirectory}
	CheckvarYes                    = &VarChecker{"Yes", (*VartypeCheckContext).Yes}
	CheckvarYesNo                  = &VarChecker{"YesNo", (*VartypeCheckContext).YesNo}
	CheckvarYesNo_Indirectly       = &VarChecker{"YesNo_Indirectly", (*VartypeCheckContext).YesNo_Indirectly}
)

func init() {
	CheckvarShellCommand.checker = (*VartypeCheckContext).ShellCommand
	CheckvarShellWord.checker = (*VartypeCheckContext).ShellWord
}
