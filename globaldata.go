package main

import (
	"fmt"
	"strings"
)

// Constant data that is loaded once.
type GlobalData struct {
	pkgsrcdir      string // Relative to the current working directory.
	masterSiteUrls map[string]string
	masterSiteVars map[string]bool
	pkgOptions     map[string]string
}

// A change entry from doc/CHANGES-*
type Change struct {
	line    *Line
	action  string
	pkgpath string
	version string
	author  string
	date    string
}

func (self *GlobalData) Initialize(pkgsrcdir string) {
	self.pkgsrcdir = pkgsrcdir
	self.loadDistSites()
	self.loadPkgOptions()
}

func (self *GlobalData) loadDistSites() {
	fname := self.pkgsrcdir + "/mk/fetch/sites.mk"
	lines := loadExistingLines(fname, true)

	varname := ""
	names := make(map[string]bool, 0)
	ignoring := false
	url2name := make(map[string]string, 0)
	for _, line := range lines {
		text := line.text
		if m := match(text, `^(MASTER_SITE_\w+)\+=\s*\\$`); m != nil {
			varname = m[1]
			names[varname] = true
			ignoring = false
		} else if text == "MASTER_SITE_BACKUP?=\t\\" {
			ignoring = true
		} else if m := match(text, `^\t((?:http://|https://|ftp://)\S+/)(?:|\s*\\)$`); m != nil {
			if !ignoring {
				if varname != "" {
					url2name[m[1]] = varname
				} else {
					line.logError("Lonely URL found.")
				}
			}
		} else if match(text, `^(?:#.*|\s*)$`) != nil || strings.Contains(text, "BSD_SITES_MK") {
		} else {
			line.logFatal("Unknown line type.")
		}
	}

	// Explicitly allowed, although not defined in mk/fetch/sites.mk.
	names["MASTER_SITE_SUSE_UPD"] = true
	names["MASTER_SITE_LOCAL"] = true

	if GlobalVars.opts.optDebugMisc {
		logDebug(fname, NO_LINES, fmt.Sprintf("Loaded %d MASTER_SITE_* definitions.", len(url2name)))
	}
	self.masterSiteUrls = url2name
	self.masterSiteVars = names
}

func (self *GlobalData) loadPkgOptions() {
	fname := self.pkgsrcdir + "/mk/defaults/options.description"
	lines := loadExistingLines(fname, false)

	options := make(map[string]string)
	for _, line := range lines {
		if m := match(line.text, `^([-0-9a-z_+]+)(?:\s+(.*))?$`); m != nil {
			optname, optdescr := m[1], m[2]
			options[optname] = optdescr
		} else {
			line.logFatal("Unknown line format.")
		}
	}
	self.pkgOptions = options
}

func (self *GlobalData) loadTools() {
	toolFiles := []string{"defaults.mk"}
	{
		fname := *GlobalVars.cwdPkgsrcdir + "/mk/tools/bsd.tools.mk"
		lines := loadExistingLines(fname, true)
		for _, line := range lines {
			if m := match(line.text, reMkInclude); m != nil {
				includefile := m[2]
				if m2 := match(includefile, `^(?:\$\{PKGSRCDIR\}/mk/tools/)?([^/]+)$`); m2 != nil {
					toolFiles = append(toolFiles, m2[1])
				}
			}
		}
	}
	if len(toolFiles) <= 1 {
		logFatal(toolFiles[0], NO_LINES, "Too few tool files files.")
	}

	tools := make(map[string]bool, 0)
	vartools := make(map[string]string, 0)
	predefinedTools := make(map[string]bool, 0)
	varnameToToolname := make(map[string]string, 0)
	systemBuildDefs := make(map[string]bool, 0)

	for _, basename := range toolFiles {
		fname := *GlobalVars.cwdPkgsrcdir + "/mk/tools/" + basename
		lines := loadExistingLines(fname, true)
		for _, line := range lines {
			if m := match(line.text, reVarassign); m != nil {
				varname, value := m[1], m[3]
				if varname == "TOOLS_CREATE" && match(value, `^([-\w.]+|\[)$`) != nil {
					tools[value] = true
				} else if mm := match(varname, `^(?:_TOOLS_VARNAME)\.([-\w.]+|\[)$`); mm != nil {
					tools[mm[1]] = true
					vartools[mm[1]] = value
					varnameToToolname[value] = mm[1]

				} else if mm := match(varname, `^(?:TOOLS_PATH|_TOOLS_DEPMETHOD)\.([-\w.]+|\[)$`); mm != nil {
					tools[mm[1]] = true

				} else if mm := match(varname, `_TOOLS\.(.*)`); mm != nil {
					tools[mm[1]] = true
					for _, tool := range splitOnSpace(value) {
						tools[tool] = true
					}
				}
			}
		}
	}

	{
		basename := "bsd.pkg.mk"
		fname := *GlobalVars.cwdPkgsrcdir + "/mk/" + basename
		condDepth := 0

		lines := loadExistingLines(fname, true)
		for _, line := range lines {
			text := line.text

			if m := match(text, reVarassign); m != nil {
				varname, value := m[1], m[3]

				if varname == "USE_TOOLS" {
					if GlobalVars.opts.optDebugTools {
						line.logDebug(fmt.Sprintf("[condDepth=%d] %s", condDepth, value))
					}
					if condDepth==0 {
						for _, tool := range splitOnSpace(value) {
							if match(tool, reUnresolvedVar) == nil && tools[tool] {
								predefinedTools[tool] = true
								predefinedTools["TOOLS_"+tool] = true
							}
						}
					}

				} else if varname == "_BUILD_DEFS" {
					for _, bdvar := range splitOnSpace(value) {
						systemBuildDefs[bdvar] = true
					}
				}

			} else if m := match(text, reMkCond); m != nil {
				cond := m[2]

				switch cond {
					case "if": case "ifdef": case "ifndef": case "for":
					condDepth++
					case "endif": case "endfor":
					condDepth--
				}
			}
		}
	}

	if GlobalVars.opts.optDebugTools {
		logDebug(NO_FILE, NO_LINES, fmt.Sprintf("tools: %v", tools))
		logDebug(NO_FILE, NO_LINES, fmt.Sprintf("vartools: %v", vartools))
		logDebug(NO_FILE, NO_LINES, fmt.Sprintf("predefinedTools: %v", predefinedTools))
		logDebug(NO_FILE, NO_LINES, fmt.Sprintf("varnameToToolname: %v", varnameToToolname))
	}
	if GlobalVars.opts.optDebugMisc {
		logDebug(NO_FILE, NO_LINES, fmt.Sprintf("systemBuildDefs: %v", systemBuildDefs))
	}
}
