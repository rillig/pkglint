package main

import (
	"io/ioutil"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

// Constant data that is loaded once.
type GlobalData struct {
	pkgsrcdir           string // Relative to the current working directory.
	masterSiteUrls      map[string]string
	masterSiteVars      map[string]bool
	pkgOptions          map[string]string
	tools               map[string]bool   // Known tool names, e.g. "sed" and "gm4".
	vartools            map[string]string // Maps tool names to their respective variable, e.g. "sed" => "SED", "gzip" => "GZIP_CMD".
	predefinedTools     map[string]bool   // Tools that a package does not need to add to USE_TOOLS explicitly because they are used by the pkgsrc infrastructure, too.
	varnameToToolname   map[string]string // Maps the tool variable names to the tool name they use, e.g. "GZIP_CMD" => "gzip" and "SED" => "sed".
	systemBuildDefs     map[string]bool   // The set of user-defined variables that are added to BUILD_DEFS within the bsd.pkg.mk file.
	varRequiredTools    map[string]bool   // Tool variable names that may not be converted to their "direct" form, that is: ${CP} => cp.
	suggestedUpdates    []SuggestedUpdate
	suggestedWipUpdates []SuggestedUpdate
	lastChange          map[string]*Change
	userDefinedVars     map[string]*Line
	vartypes            map[string]*Type
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

type SuggestedUpdate struct {
	line    *Line
	pkgname string
	version string
	comment string
}

func (self *GlobalData) Initialize(pkgsrcdir string) {
	self.pkgsrcdir = pkgsrcdir
	self.loadDistSites()
	self.loadPkgOptions()
	self.loadDocChanges()
	self.loadUserDefinedVars()
}

func (self *GlobalData) loadDistSites() {
	fname := self.pkgsrcdir + "/mk/fetch/sites.mk"
	lines := loadExistingLines(fname, true)

	names := make(map[string]bool)
	ignoring := false
	url2name := make(map[string]string)
	for _, line := range lines {
		text := line.text
		if m, varname := match1(text, `^(MASTER_SITE_\w+)\+=\s*\\$`); m {
			names[varname] = true
			ignoring = false
		} else if text == "MASTER_SITE_BACKUP?=\t\\" {
			ignoring = true
		} else if m, url := match1(text, `^\t((?:http://|https://|ftp://)\S+/)(?:|\s*\\)$`); m {
			if !ignoring {
				if varname != "" {
					url2name[url] = varname
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

	_ = G.opts.optDebugMisc && logDebug(fname, NO_LINES, "Loaded %d MASTER_SITE_* definitions.", len(url2name))
	self.masterSiteUrls = url2name
	self.masterSiteVars = names
}

func (self *GlobalData) loadPkgOptions() {
	fname := self.pkgsrcdir + "/mk/defaults/options.description"
	lines := loadExistingLines(fname, false)

	options := make(map[string]string)
	for _, line := range lines {
		if m, optname, optdescr := match2(line.text, `^([-0-9a-z_+]+)(?:\s+(.*))?$`); m {
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
		fname := *G.cwdPkgsrcdir + "/mk/tools/bsd.tools.mk"
		lines := loadExistingLines(fname, true)
		for _, line := range lines {
			if m, _, includefile := match2(line.text, reMkInclude); m {
				if m, toolfile := match1(includefile, `^(?:\$\{PKGSRCDIR\}/mk/tools/)?([^/]+)$`); m {
					toolFiles = append(toolFiles, toolfile)
				}
			}
		}
	}
	if len(toolFiles) <= 1 {
		logFatal(toolFiles[0], NO_LINES, "Too few tool files files.")
	}

	tools := make(map[string]bool)
	vartools := make(map[string]string)
	predefinedTools := make(map[string]bool)
	varnameToToolname := make(map[string]string)
	systemBuildDefs := make(map[string]bool)

	for _, basename := range toolFiles {
		fname := *G.cwdPkgsrcdir + "/mk/tools/" + basename
		lines := loadExistingLines(fname, true)
		for _, line := range lines {
			if m, varname, _, value := match3(line.text, reVarassign); m {
				if varname == "TOOLS_CREATE" && match(value, `^([-\w.]+|\[)$`) != nil {
					tools[value] = true
				} else if m, toolname := match1(varname, `^(?:_TOOLS_VARNAME)\.([-\w.]+|\[)$`); m {
					tools[toolname] = true
					vartools[toolname] = value
					varnameToToolname[value] = toolname

				} else if m, toolname := match1(varname, `^(?:TOOLS_PATH|_TOOLS_DEPMETHOD)\.([-\w.]+|\[)$`); m {
					tools[toolname] = true

				} else if m, toolname := match1(varname, `_TOOLS\.(.*)`); m {
					tools[toolname] = true
					for _, tool := range splitOnSpace(value) {
						tools[tool] = true
					}
				}
			}
		}
	}

	{
		basename := "bsd.pkg.mk"
		fname := *G.cwdPkgsrcdir + "/mk/" + basename
		condDepth := 0

		lines := loadExistingLines(fname, true)
		for _, line := range lines {
			text := line.text

			if m, varname, _, value := match3(text, reVarassign); m {
				if varname == "USE_TOOLS" {
					_ = G.opts.optDebugTools && line.logDebug("[condDepth=%d] %s", condDepth, value)
					if condDepth == 0 {
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

			} else if m, _, cond := match2(text, reMkCond); m {
				switch cond {
				case "if":
				case "ifdef":
				case "ifndef":
				case "for":
					condDepth++
				case "endif":
				case "endfor":
					condDepth--
				}
			}
		}
	}

	if G.opts.optDebugTools {
		logDebug(NO_FILE, NO_LINES, "tools: %v", tools)
		logDebug(NO_FILE, NO_LINES, "vartools: %v", vartools)
		logDebug(NO_FILE, NO_LINES, "predefinedTools: %v", predefinedTools)
		logDebug(NO_FILE, NO_LINES, "varnameToToolname: %v", varnameToToolname)
	}
	_ = G.opts.optDebugMisc && logDebug(NO_FILE, NO_LINES, "systemBuildDefs: %v", systemBuildDefs)

	// Some user-defined variables do not influence the binary
	// package at all and therefore do not have to be added to
	// BUILD_DEFS; therefore they are marked as “already added”.
	systemBuildDefs["DISTDIR"] = true
	systemBuildDefs["FETCH_CMD"] = true
	systemBuildDefs["FETCH_OUTPUT_ARGS"] = true
	systemBuildDefs["GAMES_USER"] = true
	systemBuildDefs["GAMES_GROUP"] = true
	systemBuildDefs["GAMEDATAMODE"] = true
	systemBuildDefs["GAMEDIRMODE"] = true
	systemBuildDefs["GAMEMODE"] = true
	systemBuildDefs["GAMEOWN"] = true
	systemBuildDefs["GAMEGRP"] = true

	self.tools = tools
	self.vartools = vartools
	self.predefinedTools = predefinedTools
	self.varnameToToolname = varnameToToolname
	self.systemBuildDefs = systemBuildDefs
	self.varRequiredTools = map[string]bool{
		"ECHO":   true,
		"ECHO_N": true,
		"FALSE":  true,
		"TEST":   true,
		"TRUE":   true,
	}
}

func loadSuggestedUpdatesFile(fname string) []SuggestedUpdate {
	lines := loadExistingLines(fname, false)

	updates := make([]SuggestedUpdate, 0)
	state := 0
	for _, line := range lines {
		text := line.text

		if state == 0 && text == "Suggested package updates" {
			state = 1
		} else if state == 1 && text == "" {
			state = 2
		} else if state == 2 {
			state = 3
		} else if state == 3 && text == "" {
			state = 4
		}

		if state == 3 {
			if m, pkgname, comment := match2(text, `\to\s(\S+)(?:\s*(.+))?$`); m {
				if m, pkgbase, pkgversion := match2(pkgname, rePkgname); m {
					updates = append(updates, SuggestedUpdate{line, pkgbase, pkgversion, comment})
				} else {
					line.logWarning("Invalid package name %v", pkgname)
				}
			} else {
				line.logWarning("Invalid line format %v", text)
			}
		}
	}
	return updates
}

func (self *GlobalData) loadSuggestedUpdates() {
	self.suggestedUpdates = loadSuggestedUpdatesFile(*G.cwdPkgsrcdir + "/doc/TODO")
	wipFilename := *G.cwdPkgsrcdir + "/wip/TODO"
	if _, err := os.Stat(wipFilename); err != nil {
		self.suggestedWipUpdates = loadSuggestedUpdatesFile(wipFilename)
	}
}

func (self *GlobalData) loadDocChangesFromFile(fname string) []Change {
	lines := loadExistingLines(fname, false)

	changes := make([]Change, 0)
	for _, line := range lines {
		text := line.text
		if match(text, `^\t[A-Z]`) == nil {
			continue
		}

		if m, action, pkgpath, version, author, date := match5(text, `^\t(Updated) (\S+) to (\S+) \[(\S+) (\d\d\d\d-\d\d-\d\d)\]$`); m {
			changes = append(changes, Change{line, action, pkgpath, version, author, date})

		} else if m, action, pkgpath, version, author, date := match5(text, `^\t(Added) (\S+) version (\S+) \[(\S+) (\d\d\d\d-\d\d-\d\d)\]$`); m {
			changes = append(changes, Change{line, action, pkgpath, version, author, date})

		} else if m, action, pkgpath, author, date := match4(text, `^\t(Removed) (\S+) (?:successor (\S+) )?\[(\S+) (\d\d\d\d-\d\d-\d\d)\]$`); m {
			changes = append(changes, Change{line, action, pkgpath, "", author, date})

		} else if m, action, pkgpath, version, author, date := match5(text, `^\t(Downgraded) (\S+) to (\S+) \[(\S+) (\d\d\d\d-\d\d-\d\d)\]$`); m {
			changes = append(changes, Change{line, action, pkgpath, version, author, date})

		} else if m, action, pkgpath, version, author, date := match5(text, `^\t(Renamed|Moved) (\S+) to (\S+) \[(\S+) (\d\d\d\d-\d\d-\d\d)\]$`); m {
			changes = append(changes, Change{line, action, pkgpath, version, author, date})

		} else {
			line.logWarning("Unknown doc/CHANGES line: %v", text)
			line.explainWarning("See mk/misc/developer.mk for the rules.")
		}
	}
	return changes
}

func (self *GlobalData) getSuggestedPackageUpdates() []SuggestedUpdate {
	if G.isWip {
		return self.suggestedWipUpdates
	} else {
		return self.suggestedUpdates
	}
}

func (self *GlobalData) loadDocChanges() {
	docdir := *G.cwdPkgsrcdir + "/doc"
	files, err := ioutil.ReadDir(docdir)
	if err != nil {
		logFatal(docdir, NO_LINES, "Cannot be read.")
	}

	fnames := make([]string, 0)
	for _, file := range files {
		fname := file.Name()
		if match(fname, `^CHANGES-(20\d\d)$`) != nil && fname >= "CHANGES-2011" {
			fnames = append(fnames, fname)
		}
	}

	sort.Strings(fnames)
	self.lastChange = make(map[string]*Change)
	for _, fname := range fnames {
		changes := self.loadDocChangesFromFile(filepath.Join(docdir, fname))
		for _, change := range changes {
			c := change
			self.lastChange[change.pkgpath] = &c
		}
	}
}

func (self *GlobalData) loadUserDefinedVars() {
	lines := loadExistingLines(*G.cwdPkgsrcdir+"/mk/defaults/mk.conf", true)

	for _, line := range lines {
		if m := match(line.text, reVarassign); m != nil {
			self.userDefinedVars[m[1]] = line
		}
	}
}
