package main

import (
	"fmt"
	"io/ioutil"
	"os"
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
	lastChange          map[string]Change
	userDefinedVars     map[string]*Line
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

	_ = GlobalVars.opts.optDebugMisc && logDebug(fname, NO_LINES, fmt.Sprintf("Loaded %d MASTER_SITE_* definitions.", len(url2name)))
	self.masterSiteUrls = url2name
	self.masterSiteVars = names
}

func (self *GlobalData) loadPkgOptions() {
	fname := self.pkgsrcdir + "/mk/defaults/options.description"
	lines := loadExistingLines(fname, false)

	options := make(map[string]string)
	for _, line := range lines {
		fmt.Printf("line=%#v\n", line)
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
					_ = GlobalVars.opts.optDebugTools && line.logDebug(fmt.Sprintf("[condDepth=%d] %s", condDepth, value))
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

			} else if m := match(text, reMkCond); m != nil {
				cond := m[2]

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

	if GlobalVars.opts.optDebugTools {
		logDebug(NO_FILE, NO_LINES, fmt.Sprintf("tools: %v", tools))
		logDebug(NO_FILE, NO_LINES, fmt.Sprintf("vartools: %v", vartools))
		logDebug(NO_FILE, NO_LINES, fmt.Sprintf("predefinedTools: %v", predefinedTools))
		logDebug(NO_FILE, NO_LINES, fmt.Sprintf("varnameToToolname: %v", varnameToToolname))
	}
	_ = GlobalVars.opts.optDebugMisc && logDebug(NO_FILE, NO_LINES, fmt.Sprintf("systemBuildDefs: %v", systemBuildDefs))

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
			if m := match(text, `\to\s(\S+)(?:\s*(.+))?$`); m != nil {
				pkgname, comment := m[1], m[2]
				if m = match(pkgname, rePkgname); m != nil {
					updates = append(updates, SuggestedUpdate{line, m[1], m[2], comment})
				} else {
					line.logWarning("Invalid package name " + pkgname)
				}
			} else {
				line.logWarning("Invalid line format " + text)
			}
		}
	}
	return updates
}

func (self *GlobalData) loadSuggestedUpdates() {
	self.suggestedUpdates = loadSuggestedUpdatesFile(*GlobalVars.cwdPkgsrcdir + "/doc/TODO")
	wipFilename := *GlobalVars.cwdPkgsrcdir + "/wip/TODO"
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

		if m := match(text, `^\t(Updated) (\S+) to (\S+) \[(\S+) (\d\d\d\d-\d\d-\d\d)\]$`); m != nil {
			changes = append(changes, Change{line, m[1], m[2], m[3], m[4], m[5]})
		} else if m := match(text, `^\t(Added) (\S+) version (\S+) \[(\S+) (\d\d\d\d-\d\d-\d\d)\]$`); m != nil {
			changes = append(changes, Change{line, m[1], m[2], m[3], m[4], m[5]})
		} else if m := match(text, `^\t(Removed) (\S+) (?:successor (\S+) )?\[(\S+) (\d\d\d\d-\d\d-\d\d)\]$`); m != nil {
			changes = append(changes, Change{line, m[1], m[2], "", m[3], m[4]})
		} else if m := match(text, `^\t(Downgraded) (\S+) to (\S+) \[(\S+) (\d\d\d\d-\d\d-\d\d)\]$`); m != nil {
			changes = append(changes, Change{line, m[1], m[2], m[3], m[4], m[5]})
		} else if m := match(text, `^\t(Renamed|Moved) (\S+) to (\S+) \[(\S+) (\d\d\d\d-\d\d-\d\d)\]$`); m != nil {
			changes = append(changes, Change{line, m[1], m[2], m[3], m[4], m[5]})
		} else {
			line.logWarning("Unknown doc/CHANGES line: " + text)
			line.explainWarning("See mk/misc/developer.mk for the rules.")
		}
	}
	return changes
}

func (self *GlobalData) getSuggestedPackageUpdates() []SuggestedUpdate {
	if GlobalVars.isWip {
		return self.suggestedWipUpdates
	} else {
		return self.suggestedUpdates
	}
}

func (self *GlobalData) loadDocChanges() {
	docdir := *GlobalVars.cwdPkgsrcdir + "/doc"
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
	for _, fname := range fnames {
		changes := self.loadDocChangesFromFile(fname)
		for _, change := range changes {
			self.lastChange[change.pkgpath] = change
		}
	}
}

func (self *GlobalData) loadUserDefinedVars() {
	lines := loadExistingLines(*GlobalVars.cwdPkgsrcdir+"/mk/defaults/mk.conf", true)

	for _, line := range lines {
		if m := match(line.text, reVarassign); m != nil {
			self.userDefinedVars[m[1]] = line
		}
	}
}
