package main

import (
	"io/ioutil"
)

func checkglobalUnusedLicenses() {
	licensedir := *GlobalVars.cwdPkgsrcdir + "/licenses"
	files, _ := ioutil.ReadDir(licensedir)
	for _, licensefile := range files {
		licensename := licensefile.Name()
		licensepath := licensedir + "/" + licensename
		if fileExists(licensepath) {
			if !GlobalVars.ipcUsedLicenses[licensename] {
				logWarning(licensepath, NO_LINES, "This license seems to be unused.")
			}
		}
	}
}
