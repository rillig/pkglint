package pkglint

import "github.com/rillig/pkglint/v23/pkgver"

// PackagePattern is a pattern that matches zero or more packages including
// their versions.
//
// Examples are "pkg>=1<2" or "pkg-[0-9]*".
//
// Patterns like "{ssh>=1,openssh>=6}" need to be expanded by
// expandCurlyBraces before being split into their components,
// as the alternatives from inside the braces may span multiple components.
type PackagePattern struct {
	Pkgbase  string // "freeciv-client", "${EMACS_REQD}"
	LowerOp  string // ">=", ">"
	Lower    string // "2.5.0", "${PYVER}"
	UpperOp  string // "<", "<="
	Upper    string // "3.0", "${PYVER}"
	Wildcard string // "[0-9]*", "1.5.*", "${PYVER}"
}

func ParsePackagePattern(p *MkParser) *PackagePattern {
	lexer := p.lexer

	parseVersion := func() string {
		mark := lexer.Mark()

		if p.mklex.Expr() != nil {
			for p.mklex.Expr() != nil || lexer.SkipRegexp(regcomp(`^\.\w+`)) {
			}
			return lexer.Since(mark)
		}

		m := lexer.NextRegexp(regcomp(`^\d[\w.]*`))
		if m != nil {
			return m[0]
		}

		return ""
	}

	var pp PackagePattern
	mark := lexer.Mark()
	pp.Pkgbase = p.PkgbasePattern()
	if pp.Pkgbase == "" {
		return nil
	}

	mark2 := lexer.Mark()
	op := lexer.NextString(">=")
	if op == "" {
		op = lexer.NextString(">")
	}

	if op != "" {
		version := parseVersion()
		if version != "" {
			pp.LowerOp = op
			pp.Lower = version
		} else {
			lexer.Reset(mark2)
		}
	}

	op = lexer.NextString("<=")
	if op == "" {
		op = lexer.NextString("<")
	}

	if op != "" {
		version := parseVersion()
		if version != "" {
			pp.UpperOp = op
			pp.Upper = version
		} else {
			lexer.Reset(mark2)
		}
	}

	if pp.LowerOp != "" || pp.UpperOp != "" {
		return &pp
	}

	if lexer.SkipByte('-') && lexer.Rest() != "" && lexer.PeekByte() != '-' {
		versionMark := lexer.Mark()

		for p.mklex.Expr() != nil ||
			lexer.SkipRegexp(regcomp(`^\[[^\]]+]`)) ||
			lexer.SkipRegexp(regcomp(`^[\w*._]+`)) {
		}

		if !lexer.SkipString("{,nb*}") {
			lexer.SkipString("{,nb[0-9]*}")
		}

		pp.Wildcard = lexer.Since(versionMark)
		if pp.Wildcard != "" {
			return &pp
		}
	}

	if hasPrefix(pp.Pkgbase, "$") && hasSuffix(pp.Pkgbase, "}") {
		if !lexer.SkipString("{,nb*}") {
			lexer.SkipString("{,nb[0-9]*}")
		}
		return &pp
	}

	lexer.Reset(mark)
	return nil
}

type PackagePatternChecker struct {
	Varname string
	MkLine  *MkLine
	MkLines *MkLines
}

func (ck *PackagePatternChecker) Check(value string, valueNoVar string) {
	if contains(valueNoVar, "{") {
		if !hasBalancedBraces(value) {
			ck.MkLine.Errorf("Package pattern %q must have balanced braces.", value)
			return
		}

		mainValue, nbPart := value, ""
		if hasSuffix(value, "{,nb*}") {
			mainValue, nbPart = value[:len(value)-6], value[len(value)-6:]
		} else if hasSuffix(value, "{,nb[0-9]*}") {
			mainValue, nbPart = value[:len(value)-11], value[len(value)-11:]
		}

		if contains(ck.MkLine.WithoutMakeVariables(mainValue), "{") {
			if m, expr := match1(mainValue, `(\{[^{]*\bnb\b.*?})`); m {
				ck.MkLine.Warnf("The nb version part should have the form "+
					"\"{,nb*}\" or \"{,nb[0-9]*}\", not %q.", expr)
			}
			if value != valueNoVar {
				trace.Step1("Skipping checks for package pattern %q.", value)
				return
			}
			for _, p := range expandCurlyBraces(mainValue) {
				ck.checkSingle(p + nbPart)
			}
			return
		}
	}
	ck.checkSingle(value)
}

func (ck *PackagePatternChecker) checkSingle(value string) {
	parser := NewMkParser(nil, value)
	pp := ParsePackagePattern(parser)
	rest := parser.Rest()

	if pp != nil &&
		(pp.LowerOp != "" || pp.UpperOp != "") &&
		(rest == "{,nb*}" || rest == "{,nb[0-9]*}") {
		ck.MkLine.Warnf("Dependency patterns of the form pkgbase>=1.0 don't need the \"{,nb*}\" extension.")
		ck.MkLine.Explain(
			"The \"{,nb*}\" extension is only necessary for dependencies of the",
			"form \"pkgbase-1.2\", since the pattern \"pkgbase-1.2\" doesn't match",
			"the version \"pkgbase-1.2nb5\".",
			"For package patterns using the comparison operators,",
			"this is not necessary.")

	} else if pp == nil || rest != "" {
		if rest != "" && rest != value {
			ck.MkLine.Errorf("Package pattern %q is followed by extra text %q.",
				value[:len(value)-len(rest)], rest)
		} else {
			ck.MkLine.Errorf("Invalid package pattern %q.", value)
		}
		ck.MkLine.Explain(
			"Typical package patterns have the following forms:",
			"",
			"\tpackage>=2.5",
			"\tpackage-[0-9]*",
			"\tpackage-3.141",
			"\tpackage>=2.71828<=3.1415")
		return
	}

	if pp.Lower != "" && pp.Upper != "" &&
		!containsExpr(pp.Lower) && !containsExpr(pp.Upper) &&
		pkgver.Compare(pp.Lower, pp.Upper) > 0 {
		ck.MkLine.Errorf("The lower bound \"%s\" is greater than the upper bound \"%s\".", pp.Lower, pp.Upper)
	}

	wildcard := pp.Wildcard
	if m, inside := match1(wildcard, `^\[(.*)\]\*$`); m {
		if inside != "0-9" {
			ck.MkLine.Warnf("Only \"[0-9]*\" is allowed as the numeric part of a dependency, not \"%s\".", wildcard)
			ck.MkLine.Explain(
				"The pattern \"[0-9]*\" means any version.",
				"All other version patterns should be expressed using the",
				"comparison operators, such as <5.2.3 or >=1.0 or >=2<3.",
				"",
				"Patterns like \"[0-7]*\" only match the first digit of the",
				"version number and will do the wrong thing when the package",
				"reaches version 10.")
		}

	} else if m, ver, suffix := match2(wildcard, `^(\d\w*(?:\.\w+)*)(\.\*|\{,nb\*\}|\{,nb\[0-9\]\*\}|\*|)$`); m {
		if suffix == "" {
			ck.MkLine.Warnf("Use %q instead of %q as the version pattern.", ver+"{,nb*}", ver)
			ck.MkLine.Explain(
				"Without the \"{,nb*}\" suffix, this version pattern only matches",
				"package versions that don't have a PKGREVISION (which is the part",
				"after the \"nb\").")
		}
		if suffix == "*" {
			ck.MkLine.Warnf("Use %q instead of %q as the version pattern.", ver+".*", ver+"*")
			ck.MkLine.Explain(
				"For example, the version \"1*\" also matches \"10.0.0\", which is",
				"probably not intended.")
		}

	} else if wildcard == "*" {
		ck.MkLine.Warnf("Use \"%[1]s-[0-9]*\" instead of \"%[1]s-*\".", pp.Pkgbase)
		ck.MkLine.Explain(
			"If you use a * alone, the package specification may match other",
			"packages that have the same prefix but a longer name.",
			"For example, foo-* matches foo-1.2 but also",
			"foo-client-1.2 and foo-server-1.2.")
	}

	withoutCharClasses := replaceAll(wildcard, `\[[\d-]+\]`, "")
	if contains(withoutCharClasses, "-") {
		ck.MkLine.Warnf("The version pattern \"%s\" should not contain a hyphen.", wildcard)
		ck.MkLine.Explain(
			"Pkgsrc interprets package names with version numbers like this:",
			"",
			"\t\"foo-2.0-2.1.x\" => pkgbase \"foo\", version \"2.0-2.1.x\"",
			"",
			"To make the \"2.0\" above part of the package basename, the hyphen",
			"must be omitted, so the full package name becomes \"foo2.0-2.1.x\".")
	}

	ck.checkDepends(
		pp,
		"BUILDLINK_API_DEPENDS.",
		func(data *Buildlink3Data) *PackagePattern { return data.apiDepends },
		func(data *Buildlink3Data) *MkLine { return data.apiDependsLine })
	ck.checkDepends(
		pp,
		"BUILDLINK_ABI_DEPENDS.",
		func(data *Buildlink3Data) *PackagePattern { return data.abiDepends },
		func(data *Buildlink3Data) *MkLine { return data.abiDependsLine })
}

func (ck *PackagePatternChecker) checkDepends(
	pp *PackagePattern,
	prefix string,
	depends func(data *Buildlink3Data) *PackagePattern,
	dependsLine func(data *Buildlink3Data) *MkLine,
) {
	if pp.LowerOp == "" {
		return
	}
	pkg := ck.MkLines.pkg
	if pkg == nil {
		return
	}
	if !hasPrefix(ck.Varname, prefix) {
		return
	}
	bl3id := Buildlink3ID(varnameParam(ck.Varname))
	data := pkg.bl3Data[bl3id]
	if data == nil {
		return
	}
	defpat := depends(data)
	if defpat == nil || defpat.LowerOp == "" {
		return
	}
	if containsExpr(defpat.Lower) || containsExpr(pp.Lower) {
		return
	}
	limit := condInt(defpat.LowerOp == ">=" && pp.LowerOp == ">", 1, 0)
	if pkgver.Compare(pp.Lower, defpat.Lower) < limit {
		ck.MkLine.Notef("The requirement %s%s is already guaranteed by the %s%s from %s.",
			pp.LowerOp, pp.Lower, defpat.LowerOp, defpat.Lower,
			ck.MkLine.RelMkLine(dependsLine(data)))
	}
}
