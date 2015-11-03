package main

func parseAclEntries(args []string) []AclEntry {
	result := make([]AclEntry, 0)
	for _, arg := range args {
		m := mustMatch(`^([\w.*]+|_):([adpsu]*)$`, arg)
		glob, perms := m[1], m[2]
		result = append(result, AclEntry{glob, perms})
	}
	return result
}

func loadVartypesBasictypes() {
	panic("not implemented; don’t use self-grep")
}

type NeedsQuoting int

const (
	NQ_NO NeedsQuoting = iota
	NQ_YES
	NQ_DOESNT_MATTER
	NQ_DONT_KNOW
)

func variableNeedsQuoting(line *Line, varname string, context *VarUseContext) NeedsQuoting {
	_ = GlobalVars.opts.optDebugTrace && line.logDebugF("variableNeedsQuoting: %s, %#v", varname, context)

	vartype := getVariableType(line, varname)
	if vartype == nil || context.vartype == nil {
		return NQ_DONT_KNOW
	}

	switch vartype.basicType {
	case "DistSuffix",
		"enum",
		"FileMode", "Filename",
		"Identifier",
		"Option",
		"Pathname", "PkgName", "PkgOptionsVar", "PkgRevision",
		"RelativePkgDir", "RelativePkgPath",
		"UserGroupName",
		"Varname", "Version",
		"WrkdirSubdirectory":
		if vartype.kindOfList == LK_NONE {
			return NQ_DOESNT_MATTER
		}
		if vartype.kindOfList == LK_EXTERNAL && context.extent != VUC_EXT_WORDPART {
			return NQ_NO
		}
	}

	// In .for loops, the :Q operator is always misplaced, since
	// the items are broken up at white-space, not as shell words
	// like in all other parts of make(1).
	if context.shellword == VUC_SHW_FOR {
		return NQ_NO
	}

	// Determine whether the context expects a list of shell words or not.
	wantList := context.vartype.isConsideredList() && (context.shellword == VUC_SHW_BACKT || context.extent != VUC_EXT_WORDPART)
	haveList := vartype.isConsideredList()

	_ = GlobalVars.opts.optDebugQuoting && line.logDebugF(
		"variableNeedsQuoting: varname=%v, context=%v, type=%v, wantList=%v, haveList=%v",
		varname, context, vartype, wantList, haveList)

	// A shell word may appear as part of a shell word, for example COMPILER_RPATH_FLAG.
	if context.extent == VUC_EXT_WORDPART && context.shellword == VUC_SHW_PLAIN {
		if vartype.kindOfList == LK_NONE && vartype.basicType == "ShellWord" {
			return NQ_NO
		}
	}

	// Assuming the tool definitions don't include very special characters,
	// so they can safely be used inside any quotes.
	if GlobalVars.globalData.varnameToToolname[varname] != "" {
		shellword := context.shellword
		switch {
		case shellword == VUC_SHW_PLAIN && context.extent != VUC_EXT_WORDPART:
			return NQ_NO
		case shellword == VUC_SHW_BACKT:
			return NQ_NO
		case shellword == VUC_SHW_DQUOT || shellword == VUC_SHW_SQUOT:
			return NQ_DOESNT_MATTER
		}
	}

	// Variables that appear as parts of shell words generally need
	// to be quoted. An exception is in the case of backticks,
	// because the whole backticks expression is parsed as a single
	// shell word by pkglint.
	if context.extent == VUC_EXT_WORDPART && context.shellword != VUC_SHW_BACKT {
		return NQ_YES
	}

	// Assigning lists to lists does not require any quoting, though
	// there may be cases like "CONFIGURE_ARGS+= -libs ${LDFLAGS:Q}"
	// where quoting is necessary.
	if wantList && haveList {
		return NQ_DOESNT_MATTER
	}

	if wantList != haveList {
		return NQ_YES
	}

	_ = GlobalVars.opts.optDebugQuoting && line.logDebugF("Don't know whether :Q is needed for %v", varname)
	return NQ_DONT_KNOW
}
