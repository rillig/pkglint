package pkglint

import "strings"

// VarorderChecker checks that in simple package Makefiles,
// the most common variables appear in a fixed order.
// The order itself is a little arbitrary but provides
// at least a bit of consistency.
type VarorderChecker struct {
}

func (ck *VarorderChecker) Check(mklines *MkLines) {

	// TODO: Generalize this code
	//  since it is equally useful for PKG_OPTIONS in options.mk,
	//  and it is related to SubstContext.

	type Repetition uint8
	const (
		optional Repetition = iota
		once
		many
	)

	type Variable struct {
		Name       string
		Repetition Repetition
	}

	emptyLine := Variable{"", once}

	// See doc/Makefile-example.
	// See https://netbsd.org/docs/pkgsrc/pkgsrc.html#components.Makefile.
	var variables = []Variable{
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
		emptyLine,
		{"PATCH_SITES", optional},
		{"PATCH_SITE_SUBDIR", optional},
		{"PATCHFILES", optional},
		{"PATCH_DIST_ARGS", optional},
		{"PATCH_DIST_STRIP", optional},
		{"PATCH_DIST_CAT", optional},
		emptyLine,
		{"MAINTAINER", optional},
		{"OWNER", optional},
		{"HOMEPAGE", optional},
		{"COMMENT", once},
		{"LICENSE", once},
		emptyLine,
		{"LICENSE_FILE", optional},
		{"RESTRICTED", optional},
		{"NO_BIN_ON_CDROM", optional},
		{"NO_BIN_ON_FTP", optional},
		{"NO_SRC_ON_CDROM", optional},
		{"NO_SRC_ON_FTP", optional},
		emptyLine,
		{"BROKEN_EXCEPT_ON_PLATFORM", many},
		{"BROKEN_ON_PLATFORM", many},
		{"NOT_FOR_PLATFORM", many},
		{"ONLY_FOR_PLATFORM", many},
		{"NOT_FOR_COMPILER", many},
		{"ONLY_FOR_COMPILER", many},
		{"NOT_FOR_UNPRIVILEGED", optional},
		emptyLine,
		{"BUILD_DEPENDS", many},
		{"TOOL_DEPENDS", many},
		{"DEPENDS", many}}

	relevantLines := func() []*MkLine {
		firstRelevant := -1
		lastRelevant := -1

		relevantVars := make(map[string]bool)
		for _, variable := range variables {
			if variable != emptyLine {
				relevantVars[variable.Name] = true
			}
		}

		firstIrrelevant := -1
		for i, mkline := range mklines.mklines {
			switch {
			case mkline.IsVarassignMaybeCommented():
				varcanon := mkline.Varcanon()
				if relevantVars[varcanon] {
					if firstRelevant == -1 {
						firstRelevant = i
					}
					if firstIrrelevant != -1 {
						if trace.Tracing {
							trace.Stepf("Skipping varorder because of line %s.",
								mklines.mklines[firstIrrelevant].Linenos())
						}
						return nil
					}
					lastRelevant = i
				} else {
					if firstIrrelevant == -1 {
						firstIrrelevant = i
					}
				}

			case mkline.IsComment(), mkline.IsEmpty():
				break

			default:
				if firstIrrelevant == -1 {
					firstIrrelevant = i
				}
			}
		}

		if firstRelevant == -1 {
			return nil
		}
		return mklines.mklines[firstRelevant : lastRelevant+1]
	}()

	// If there are foreign variables, skip the whole check.
	// The check is only intended for the most simple packages.
	skip := func() bool {
		interesting := relevantLines

		varcanon := func() string {
			for len(interesting) > 0 && interesting[0].IsComment() {
				interesting = interesting[1:]
			}

			if len(interesting) > 0 && interesting[0].IsVarassign() {
				return interesting[0].Varcanon()
			}
			return ""
		}

		for _, variable := range variables {
			if variable == emptyLine {
				for len(interesting) > 0 && (interesting[0].IsEmpty() || interesting[0].IsComment()) {
					interesting = interesting[1:]
				}
				continue
			}

			switch variable.Repetition {
			case optional:
				if varcanon() == variable.Name {
					interesting = interesting[1:]
				}
			case once:
				if varcanon() == variable.Name {
					interesting = interesting[1:]
				} else if variable.Name != "LICENSE" {
					if trace.Tracing {
						trace.Stepf("Wrong varorder because %s is missing.", variable.Name)
					}
					return false
				}
			default:
				for varcanon() == variable.Name {
					interesting = interesting[1:]
				}
			}
		}

		return len(interesting) == 0
	}

	// canonical returns the canonical ordering of the variables. It mentions all the
	// variables that occur in the relevant section, as well as the "once" variables.
	canonical := func() string {
		var canonical []string
		for _, variable := range variables {
			if variable == emptyLine {
				if canonical[len(canonical)-1] != "empty line" {
					canonical = append(canonical, "empty line")
				}
				continue
			}

			found := false
			for _, mkline := range relevantLines {
				if mkline.IsVarassignMaybeCommented() &&
					mkline.Varcanon() == variable.Name {

					canonical = append(canonical, mkline.Varname())
					found = true
					break
				}
			}

			if !found && variable.Repetition == once {
				canonical = append(canonical, variable.Name)
			}
		}

		if canonical[len(canonical)-1] == "empty line" {
			canonical = canonical[:len(canonical)-1]
		}
		return strings.Join(canonical, ", ")
	}

	if len(relevantLines) == 0 || skip() {
		return
	}

	// TODO: This leads to very long and complicated warnings.
	//  Those parts that are correct should not be mentioned,
	//  except if they are helpful for locating the mistakes.
	mkline := relevantLines[0]
	mkline.Warnf("The canonical order of the variables is %s.", canonical())
	mkline.Explain(
		"In simple package Makefiles, some common variables should be",
		"arranged in a specific order.",
		"",
		"See doc/Makefile-example for an example Makefile.",
		seeGuide("Package components, Makefile", "components.Makefile"))
}
