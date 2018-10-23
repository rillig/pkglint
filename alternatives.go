package main

import "strings"

func CheckfileAlternatives(fileName string) {
	lines := Load(fileName, NotEmpty|LogErrors)
	if lines == nil {
		return
	}

	var plist PlistContent
	if G.Pkg != nil {
		plist = G.Pkg.Plist
	}

	for _, line := range lines {
		if m, wrapper, space, implementation := match3(line.Text, `^(\S+)([ \t]+)(\S+)`); m {
			if plist.Files != nil {
				if plist.Files[wrapper] {
					line.Errorf("Alternative wrapper %q must not appear in the PLIST.", wrapper)
				}

				relImplementation := strings.Replace(implementation, "@PREFIX@/", "", 1)
				plistName := replaceAll(relImplementation, `@(\w+)@`, "${$1}")
				if !plist.Files[plistName] && !G.Pkg.vars.Defined("ALTERNATIVES_SRC") {
					if plistName != implementation {
						line.Errorf("Alternative implementation %q must appear in the PLIST as %q.", implementation, plistName)
					} else {
						line.Errorf("Alternative implementation %q must appear in the PLIST.", implementation)
					}
				}
			}

			fix := line.Autofix()
			fix.Notef("@PREFIX@/ can be omitted from the file name.")
			fix.Explain(
				"The alternative implementation is always interpreted relative to",
				"${PREFIX}.")
			fix.ReplaceAfter(space, "@PREFIX@/", "")
			fix.Apply()
		} else {
			line.Errorf("Invalid ALTERNATIVES line %q.", line.Text)
			Explain(
				"Run \"" + confMake + " help topic=alternatives\" for more information.")
		}
	}
}
