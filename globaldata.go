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
