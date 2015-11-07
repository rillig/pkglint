package main

// Context of the package that is currently checked.
type PkgContext struct {
	pkgpath                *string // The relative path to the package within PKGSRC
	pkgdir                 *string // PKGDIR from the package Makefile
	filesdir               string // FILESDIR from the package Makefile
	patchdir               string // PATCHDIR from the package Makefile
	distinfoFile           string // DISTINFO_FILE from the package Makefile
	effective_pkgname      *string // PKGNAME or DISTNAME from the package Makefile
	effective_pkgbase      *string // The effective PKGNAME without the version
	effective_pkgversion   *string // The version part of the effective PKGNAME
	effective_pkgname_line *Line   // The origin of the three effective_* values
	seen_bsd_prefs_mk      bool    // Has bsd.prefs.mk already been included?

	vardef             map[string]*Line // varname => line
	varuse             map[string]*Line // varname => line
	bl3                map[string]*Line // buildlink3.mk name => line; contains only buildlink3.mk files that are directly included.
	plistSubstCond     map[string]bool  // varname => true; list of all variables that are used as conditionals (@comment or nothing) in PLISTs.
	included           map[string]*Line // fname => line
	seenMakefileCommon bool             // Does the package have any .includes?
	isWip              bool             // Is the current to-be-checked item from pkgsrc-wip?
}
