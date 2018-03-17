package main

import "netbsd.org/pkglint/trace"

// Pkgsrc describes a pkgsrc installation.
// In each pkglint run, only a single pkgsrc installation is ever loaded.
// It just doesn't make sense to check multiple pkgsrc installations at once.
type Pkgsrc = *PkgsrcImpl

type PkgsrcImpl struct {

	// The top directory (PKGSRCDIR), either absolute or relative to
	// the current working directory.
	topdir string

	// The set of user-defined variables that are added to BUILD_DEFS
	// within the bsd.pkg.mk file.
	buildDefs map[string]bool

	Tools ToolRegistry

	MasterSiteURLToVar map[string]string // "https://github.com/" => "MASTER_SITE_GITHUB"
	MasterSiteVarToURL map[string]string // "MASTER_SITE_GITHUB" => "https://github.com/"
}

func NewPkgsrc(dir string) Pkgsrc {
	src := &PkgsrcImpl{
		dir,
		make(map[string]bool),
		NewToolRegistry(),
		make(map[string]string),
		make(map[string]string)}

	// Some user-defined variables do not influence the binary
	// package at all and therefore do not have to be added to
	// BUILD_DEFS; therefore they are marked as "already added".
	src.AddBuildDef("DISTDIR")
	src.AddBuildDef("FETCH_CMD")
	src.AddBuildDef("FETCH_OUTPUT_ARGS")

	// The following variables are not expected to be modified
	// by the pkgsrc user. They are added here to prevent unnecessary
	// warnings by pkglint.
	src.AddBuildDef("GAMES_USER")
	src.AddBuildDef("GAMES_GROUP")
	src.AddBuildDef("GAMEDATAMODE")
	src.AddBuildDef("GAMEDIRMODE")
	src.AddBuildDef("GAMEMODE")
	src.AddBuildDef("GAMEOWN")
	src.AddBuildDef("GAMEGRP")

	return src
}

// LoadExistingLines loads the file relative to the pkgsrc top directory.
func (src *PkgsrcImpl) LoadExistingLines(fileName string, joinBackslashLines bool) []Line {
	return LoadExistingLines(src.topdir+"/"+fileName, joinBackslashLines)
}

// File resolves a file name relative to the pkgsrc top directory.
//
// Example:
//  NewPkgsrc("/usr/pkgsrc").File("distfiles") => "/usr/pkgsrc/distfiles"
func (src *PkgsrcImpl) File(relativeName string) string {
	return src.topdir + "/" + relativeName
}

// ToRel returns the path of `fileName`, relative to the pkgsrc top directory.
//
// Example:
//  NewPkgsrc("/usr/pkgsrc").ToRel("/usr/pkgsrc/distfiles") => "distfiles"
func (src *PkgsrcImpl) ToRel(fileName string) string {
	return relpath(src.topdir, fileName)
}

func (src *PkgsrcImpl) AddBuildDef(varname string) {
	src.buildDefs[varname] = true
}

func (src *PkgsrcImpl) IsBuildDef(varname string) bool {
	return src.buildDefs[varname]
}

func (src *PkgsrcImpl) loadMasterSites() {
	lines := src.LoadExistingLines("mk/fetch/sites.mk", true)

	nameToUrl := src.MasterSiteVarToURL
	urlToName := src.MasterSiteURLToVar
	for _, line := range lines {
		if m, commented, varname, _, _, _, urls, _, _ := MatchVarassign(line.Text); m {
			if !commented && hasPrefix(varname, "MASTER_SITE_") && varname != "MASTER_SITE_BACKUP" {
				for _, url := range splitOnSpace(urls) {
					if matches(url, `^(?:http://|https://|ftp://)`) {
						if nameToUrl[varname] == "" {
							nameToUrl[varname] = url
						}
						urlToName[url] = varname
					}
				}
			}
		}
	}

	// Explicitly allowed, although not defined in mk/fetch/sites.mk.
	nameToUrl["MASTER_SITE_LOCAL"] = "ftp://ftp.NetBSD.org/pub/pkgsrc/distfiles/LOCAL_PORTS/"

	if trace.Tracing {
		trace.Stepf("Loaded %d MASTER_SITE_* URLs.", len(urlToName))
	}
}
