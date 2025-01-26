package pkglint

// VarorderChecker checks that in simple package Makefiles,
// the most common variables appear in a fixed order,
// as detailed in the pkgsrc guide.
type VarorderChecker struct {
	mklines  *MkLines
	relevant map[string]bool
}

func NewVarorderChecker(mklines *MkLines) *VarorderChecker {
	ck := VarorderChecker{mklines, map[string]bool{}}
	for _, variable := range varorderVariables {
		ck.relevant[variable.canon] = true
	}
	return &ck
}

func (ck *VarorderChecker) Check() {

	// TODO: Generalize this code
	//  since it is equally useful for PKG_OPTIONS in options.mk,
	//  and it is related to SubstContext.

	relevant, bottom := ck.relevantLines()
	if len(relevant) == 0 {
		return
	}

	ck.check(relevant, bottom)
}

// relevantLines returns the variable assignments and the empty lines
// from the top of the makefile, until there is a different kind of line.
// If there is another relevant variable assignment later in the file,
// the makefile is not considered simple enough to enforce the order of the
// variable assignments.
func (ck *VarorderChecker) relevantLines() (relevant []*MkLine, bottom *MkLine) {
	mklines := ck.mklines.mklines

	i := 0
	for ; i < len(mklines); i++ {
		mkline := mklines[i]
		if mkline.IsVarassignMaybeCommented() {
			if ck.relevant[mkline.Varcanon()] {
				relevant = append(relevant, mkline)
			}
		} else if mkline.IsEmpty() {
			if len(relevant) > 0 && !relevant[len(relevant)-1].IsEmpty() {
				relevant = append(relevant, mkline)
			}
		} else if mkline.IsComment() {
			continue
		} else {
			break
		}
	}
	for len(relevant) > 0 && relevant[len(relevant)-1].IsEmpty() {
		relevant = relevant[:len(relevant)-1]
	}

	if i == len(mklines) {
		return nil, nil
	}
	bottom = mklines[i]

	for ; i < len(mklines); i++ {
		switch mkline := mklines[i]; {
		case mkline.IsVarassignMaybeCommented():
			if ck.relevant[mkline.Varcanon()] {
				return nil, nil
			}
		case mkline.IsInclude():
			if !mkline.IncludedFile().HasBase("buildlink3.mk") &&
				!mkline.IncludedFile().ContainsPath("mk") {
				return nil, nil
			}
		}
	}

	return
}

func (ck *VarorderChecker) check(mklines []*MkLine, bottom *MkLine) {
	location := map[string]*MkLine{}
	mi, mn := 0, len(mklines)
	for _, v := range varorderVariables {

		if v.canon == "" {
			pi := mi
			for mi < mn && mklines[mi].IsEmpty() {
				mi++
			}
			if mi == pi && mi > 0 && mi < mn && !mklines[mi-1].IsEmpty() {
				mkline := mklines[mi]
				mkline.Warnf("Missing empty line.")
				ck.explain(mkline)
				return
			}
			continue
		}

		commented := 0
		uncommented := 0
		for mi < mn &&
			mklines[mi].IsVarassignMaybeCommented() &&
			mklines[mi].Varcanon() == v.canon {
			if mklines[mi].IsComment() {
				commented++
			} else {
				uncommented++
				if uncommented > 1 && v.repetition != many {
					mkline := mklines[mi]
					mkline.Warnf("The variable \"%s\" should only occur once.", v.canon)
					ck.explain(mkline)
					return
				}
			}
			mi++
		}

		if v.repetition == once && commented+uncommented == 0 &&
			!(v.canon == "LICENSE" && ck.skipLicenseCheck(mklines)) {
			mkline := bottom
			if mi < mn {
				mkline = mklines[mi]
			}
			if mkline.IsVarassignMaybeCommented() && location[mkline.Varcanon()] != nil {
				mkline.Warnf("The variable \"%s\" occurs too late, should be in %s.",
					mkline.Varname(), mkline.RelMkLine(location[mkline.Varcanon()]))
			} else if mi < mn && mkline.IsVarassignMaybeCommented() {
				mkline.Warnf("The variable \"%s\" occurs too early, should be after \"%s\".",
					mkline.Varname(), v.canon)
			} else {
				mkline.Warnf("The variable \"%s\" should be defined here.", v.canon)
			}
			ck.explain(mkline)
			return
		}

		if commented+uncommented == 0 && mi < mn &&
			mklines[mi].IsVarassignMaybeCommented() &&
			location[mklines[mi].Varcanon()] != nil {
			mkline := mklines[mi]
			mkline.Warnf("The variable \"%s\" is misplaced, should be in %s.",
				mklines[mi].Varname(), mkline.RelMkLine(location[mklines[mi].Varcanon()]))
			ck.explain(mkline)
			return
		}
		if mi < mn {
			location[v.canon] = mklines[mi]
		}
	}
}

func (ck *VarorderChecker) skipLicenseCheck(mklines []*MkLine) bool {
	for _, mkline := range mklines {
		if mkline.IsVarassignMaybeCommented() && mkline.Varcanon() == "LICENSE" {
			return false
		}
	}
	return true
}

func (*VarorderChecker) explain(mkline *MkLine) {
	mkline.Explain(
		"In simple package Makefiles, some common variables should be",
		"arranged in a specific order.",
		"",
		"See doc/Makefile-example for an example Makefile.",
		seeGuide("Package components, Makefile", "components.Makefile"))
}

type varorderRepetition uint8

const (
	optional varorderRepetition = iota
	once
	many
)

type varorderVariable struct {
	canon      string
	repetition varorderRepetition
}

// See doc/Makefile-example.
// See https://netbsd.org/docs/pkgsrc/pkgsrc.html#components.Makefile.
var varorderVariables = []varorderVariable{
	{"DISTNAME", optional},
	{"PKGNAME", optional},
	{"R_PKGNAME", optional},
	{"R_PKGVER", optional},
	{"PKGREVISION", optional},
	{"CATEGORIES", once},
	{"MASTER_SITES", many},
	{"GITHUB_PROJECT", optional},
	{"GITHUB_TAG", optional},
	{"GITHUB_RELEASE", optional},
	{"DIST_SUBDIR", optional},
	{"EXTRACT_SUFX", optional},
	{"DISTFILES", many},
	{"SITES.*", many},
	{"", once},
	{"PATCH_SITES", optional},
	{"PATCH_SITE_SUBDIR", optional},
	{"PATCHFILES", many},
	{"PATCH_DIST_ARGS", optional},
	{"PATCH_DIST_STRIP", optional},
	{"PATCH_DIST_CAT", optional},
	{"", once},
	{"MAINTAINER", optional},
	{"OWNER", optional},
	{"HOMEPAGE", optional},
	{"COMMENT", once},
	{"LICENSE", once},
	{"", once},
	{"LICENSE_FILE", optional},
	{"RESTRICTED", optional},
	{"NO_BIN_ON_CDROM", optional},
	{"NO_BIN_ON_FTP", optional},
	{"NO_SRC_ON_CDROM", optional},
	{"NO_SRC_ON_FTP", optional},
	{"", once},
	{"BROKEN_EXCEPT_ON_PLATFORM", many},
	{"BROKEN_ON_PLATFORM", many},
	{"NOT_FOR_PLATFORM", many},
	{"ONLY_FOR_PLATFORM", many},
	{"NOT_FOR_COMPILER", many},
	{"ONLY_FOR_COMPILER", many},
	{"NOT_FOR_UNPRIVILEGED", optional},
	{"", once},
	{"BUILD_DEPENDS", many},
	{"TOOL_DEPENDS", many},
	{"DEPENDS", many},
}
