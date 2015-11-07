package main

import (
	"io/ioutil"
)

func checkglobalUnusedLicenses() {
	licensedir := *G.cwdPkgsrcdir + "/licenses"
	files, _ := ioutil.ReadDir(licensedir)
	for _, licensefile := range files {
		licensename := licensefile.Name()
		licensepath := licensedir + "/" + licensename
		if fileExists(licensepath) {
			if !G.ipcUsedLicenses[licensename] {
				logWarning(licensepath, NO_LINES, "This license seems to be unused.")
			}
		}
	}
}
