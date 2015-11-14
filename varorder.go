package main

func checklinesPackageMakefileVarorder(lines []*Line) {
	if !G.opts.optWarnOrder {
		return
	}

	type OccCount int
	const (
		once OccCount = iota
		optional
		many
	)
	type OccDef struct {
		varname string
		count   OccCount
	}
	type OccGroup struct {
		name  string
		count OccCount
		occ   []OccDef
	}

	var sections = []OccGroup{
		{"Initial comments", once,
			[]OccDef{},
		},
		{"Unsorted stuff, part 1", once,
			[]OccDef{
				{"DISTNAME", optional},
				{"PKGNAME", optional},
				{"PKGREVISION", optional},
				{"CATEGORIES", once},
				{"MASTER_SITES", optional},
				{"DIST_SUBDIR", optional},
				{"EXTRACT_SUFX", optional},
				{"DISTFILES", many},
				{"SITES.*", many},
			},
		},
		{"Distribution patches", optional,
			[]OccDef{
				{"PATCH_SITES", optional}, // or once?
				{"PATCH_SITE_SUBDIR", optional},
				{"PATCHFILES", optional}, // or once?
				{"PATCH_DIST_ARGS", optional},
				{"PATCH_DIST_STRIP", optional},
				{"PATCH_DIST_CAT", optional},
			},
		},
		{"Unsorted stuff, part 2", once,
			[]OccDef{
				{"MAINTAINER", optional},
				{"OWNER", optional},
				{"HOMEPAGE", optional},
				{"COMMENT", once},
				{"LICENSE", once},
			},
		},
		{"Legal issues", optional,
			[]OccDef{
				{"LICENSE_FILE", optional},
				{"RESTRICTED", optional},
				{"NO_BIN_ON_CDROM", optional},
				{"NO_BIN_ON_FTP", optional},
				{"NO_SRC_ON_CDROM", optional},
				{"NO_SRC_ON_FTP", optional},
			},
		},
		{"Technical restrictions", optional,
			[]OccDef{
				{"BROKEN_EXCEPT_ON_PLATFORM", many},
				{"BROKEN_ON_PLATFORM", many},
				{"NOT_FOR_PLATFORM", many},
				{"ONLY_FOR_PLATFORM", many},
				{"NOT_FOR_COMPILER", many},
				{"ONLY_FOR_COMPILER", many},
				{"NOT_FOR_UNPRIVILEGED", optional},
				{"ONLY_FOR_UNPRIVILEGED", optional},
			},
		},
		{"Dependencies", optional,
			[]OccDef{
				{"BUILD_DEPENDS", many},
				{"TOOL_DEPENDS", many},
				{"DEPENDS", many},
			},
		},
	}

	if G.pkgContext == nil || G.pkgContext.seenMakefileCommon {
		return
	}

	lineno := 0
	sectindex := -1
	varindex := 0
	nextSection := true
	var vars []OccDef
	below := make(map[string]*string)
	var belowWhat *string

	// If the current section is optional but contains non-optional
	// fields, the complete section may be skipped as long as there
	// has not been a non-optional variable.
	maySkipSection := false

	// In each iteration, one of the following becomes true:
	// - new lineno > old lineno
	// - new sectindex > old sectindex
	// - new sectindex == old sectindex && new varindex > old varindex
	// - new next_section == true && old next_section == false
	for lineno <= len(lines) {
		line := lines[lineno]
		text := line.text

		_ = G.opts.optDebugMisc && line.debugf("[varorder] section %d variable %d", sectindex, varindex)

		if nextSection {
			nextSection = false
			sectindex++
			if !(sectindex < len(sections)) {
				break
			}
			vars = sections[sectindex].occ
			maySkipSection = sections[sectindex].count == optional
			varindex = 0
		}

		switch {
		case hasPrefix(text, "#"):
			lineno++

		case line.extra["varcanon"] != nil:
			varcanon := line.extra["varcanon"].(string)

			if belowText, exists := below[varcanon]; exists {
				if belowText != nil {
					line.warnf("%s appears too late. Please put it below %s.", varcanon, belowText)
				} else {
					line.warnf("%s appears too late. It should be the very first definition.", varcanon)
				}
				lineno++
				continue
			}

			for varindex < len(vars) && varcanon != vars[varindex].varname && (vars[varindex].count != once || maySkipSection) {
				if vars[varindex].count == once {
					maySkipSection = false
				}
				below[vars[varindex].varname] = belowWhat
				varindex++
			}
			switch {
			case !(varindex < len(vars)):
				if sections[sectindex].count != optional {
					line.warnf("Empty line expected.")
				}
				nextSection = true

			case varcanon != vars[varindex].varname:
				line.warnf("Expected %s, but found %s.", vars[varindex].varname, varcanon)
				lineno++

			default:
				if vars[varindex].count != many {
					below[vars[varindex].varname] = belowWhat
					varindex++
				}
				lineno++
			}
			belowWhat = newStr(varcanon)

		default:
			for varindex < len(vars) {
				if vars[varindex].count == once && !maySkipSection {
					line.warnf("%s should be set here.", vars[varindex].varname)
				}
				below[vars[varindex].varname] = belowWhat
				varindex++
			}
			nextSection = true
			if text == "" {
				belowWhat = newStr("the previous empty line")
				lineno++
			}
		}
	}
}
