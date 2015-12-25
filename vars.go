package main

type NeedsQuoting uint8

const (
	nqNo NeedsQuoting = iota
	nqYes
	nqDoesntMatter
	nqDontKnow
)

func variableNeedsQuoting(line *Line, varname string, vuc *VarUseContext) NeedsQuoting {
	defer tracecall("variableNeedsQuoting", varname, *vuc)()

	vartype := getVariableType(line, varname)
	if vartype == nil || vuc.vartype == nil {
		return nqDontKnow
	}

	isPlainWord := vartype.checker.IsEnum()
	if c := vartype.checker; false ||
		c == CheckvarDistSuffix ||
		c == CheckvarFileMode ||
		c == CheckvarFilename ||
		c == CheckvarIdentifier ||
		c == CheckvarOption ||
		c == CheckvarPathname ||
		c == CheckvarPkgName ||
		c == CheckvarPkgOptionsVar ||
		c == CheckvarPkgRevision ||
		c == CheckvarRelativePkgDir ||
		c == CheckvarRelativePkgPath ||
		c == CheckvarUserGroupName ||
		c == CheckvarVarname ||
		c == CheckvarVersion ||
		c == CheckvarWrkdirSubdirectory {
		isPlainWord = true
	}
	if isPlainWord {
		if vartype.kindOfList == lkNone {
			return nqDoesntMatter
		}
		if vartype.kindOfList == lkShell && vuc.extent != vucExtentWordpart {
			return nqNo
		}
	}

	// In .for loops, the :Q operator is always misplaced, since
	// the items are broken up at white-space, not as shell words
	// like in all other parts of make(1).
	if vuc.quoting == vucQuotFor {
		return nqNo
	}

	// Determine whether the context expects a list of shell words or not.
	wantList := vuc.vartype.isConsideredList() && (vuc.quoting == vucQuotBackt || vuc.extent != vucExtentWordpart)
	haveList := vartype.isConsideredList()

	if G.opts.DebugQuoting {
		line.debugf("variableNeedsQuoting: varname=%q, context=%v, type=%v, wantList=%v, haveList=%v",
			varname, vuc, vartype, wantList, haveList)
	}

	// A shell word may appear as part of a shell word, for example COMPILER_RPATH_FLAG.
	if vuc.extent == vucExtentWordpart && vuc.quoting == vucQuotPlain {
		if vartype.kindOfList == lkNone && vartype.checker == CheckvarShellWord {
			return nqNo
		}
	}

	// Assuming the tool definitions don't include very special characters,
	// so they can safely be used inside any quotes.
	if G.globalData.varnameToToolname[varname] != "" {
		switch vuc.quoting {
		case vucQuotPlain:
			if vuc.extent != vucExtentWordpart {
				return nqNo
			}
		case vucQuotBackt:
			return nqNo
		case vucQuotDquot, vucQuotSquot:
			return nqDoesntMatter
		}
	}

	// Variables that appear as parts of shell words generally need
	// to be quoted. An exception is in the case of backticks,
	// because the whole backticks expression is parsed as a single
	// shell word by pkglint.
	if vuc.extent == vucExtentWordpart && vuc.quoting != vucQuotBackt {
		return nqYes
	}

	// Assigning lists to lists does not require any quoting, though
	// there may be cases like "CONFIGURE_ARGS+= -libs ${LDFLAGS:Q}"
	// where quoting is necessary.
	if wantList && haveList {
		return nqDoesntMatter
	}

	if wantList != haveList {
		return nqYes
	}

	if G.opts.DebugQuoting {
		line.debug1("Don't know whether :Q is needed for %q", varname)
	}
	return nqDontKnow
}
