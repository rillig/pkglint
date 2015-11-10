package main

import (
	"os"
)

func checkperms(fname string) {
	st, err := os.Stat(fname)
	if err == nil && st.Mode().IsRegular() && (st.Mode().Perm()&0111 != 0) {
		line := NewLine(fname, NO_LINES, "", nil)
		line.logWarning("Should not be executable.")
		line.explainWarning(
			"No package file should ever be executable. Even the INSTALL and",
			"DEINSTALL scripts are usually not usable in the form they have in the",
			"package, as the pathnames get adjusted during installation. So there is",
			"no need to have any file executable.")
	}
}
