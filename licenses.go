package main

import (
	"io/ioutil"
	"strings"
)

//go:generate go tool yacc -p liyy -o licenseyacc.go -v licenseyacc.log license.y

type LicenseCondition struct {
	Name string
	And  []LicenseCondition
	Or   []LicenseCondition
}

func (lc *LicenseCondition) Walk(callback func(*LicenseCondition)) {
	callback(lc)
	for _, and := range lc.And {
		and.Walk(callback)
	}
	for _, or := range lc.Or {
		or.Walk(callback)
	}
}

type licenseLexer struct {
	repl   *PrefixReplacer
	result LicenseCondition
	error  string
}

func (lexer *licenseLexer) Lex(llval *liyySymType) int {
	repl := lexer.repl
	repl.AdvanceHspace()
	switch {
	case repl.rest == "":
		return 0
	case repl.AdvanceStr("("):
		return ltOPEN
	case repl.AdvanceStr(")"):
		return ltCLOSE
	case repl.AdvanceRegexp(`^[\w-.]+`):
		word := repl.m[0]
		switch word {
		case "AND":
			return ltAND
		case "OR":
			return ltOR
		default:
			llval.Node.Name = word
			return ltNAME
		}
	}
	return -1
}

func (lexer *licenseLexer) Error(s string) {
	lexer.error = s
}

func parseLicenses(licenses string) *LicenseCondition {
	expanded := strings.Replace(licenses, "${PERL5_LICENSE}", "gnu-gpl-v2 OR artistic", -1)
	lexer := &licenseLexer{repl: NewPrefixReplacer(expanded)}
	result := liyyNewParser().Parse(lexer)
	if result == 0 {
		return &lexer.result
	}
	return nil
}

func checkToplevelUnusedLicenses() {
	if G.UsedLicenses == nil {
		return
	}

	licensedir := G.globalData.Pkgsrcdir + "/licenses"
	files, _ := ioutil.ReadDir(licensedir)
	for _, licensefile := range files {
		licensename := licensefile.Name()
		licensepath := licensedir + "/" + licensename
		if fileExists(licensepath) {
			if !G.UsedLicenses[licensename] {
				NewLineWhole(licensepath).Warn0("This license seems to be unused.")
			}
		}
	}
}

type LicenseChecker struct {
	MkLine *MkLine
}

func (lc *LicenseChecker) Check(value string) {
	licenses := parseLicenses(value)

	if licenses == nil {
		lc.MkLine.Line.Error1("Parse error for license condition %q.", value)
		return
	}

	licenses.Walk(lc.checkNode)
}

func (lc *LicenseChecker) checkNode(cond *LicenseCondition) {
	license := cond.Name
	var licenseFile string
	if G.Pkg != nil {
		if licenseFileValue, ok := G.Pkg.varValue("LICENSE_FILE"); ok {
			licenseFile = G.CurrentDir + "/" + resolveVarsInRelativePath(licenseFileValue, false)
		}
	}
	if licenseFile == "" {
		licenseFile = G.globalData.Pkgsrcdir + "/licenses/" + license
		if G.UsedLicenses != nil {
			G.UsedLicenses[license] = true
		}
	}

	if !fileExists(licenseFile) {
		lc.MkLine.Warn1("License file %s does not exist.", cleanpath(licenseFile))
	}

	switch license {
	case "fee-based-commercial-use",
		"no-commercial-use",
		"no-profit",
		"no-redistribution",
		"shareware":
		lc.MkLine.Error1("License %q must not be used.", license)
		Explain(
			"Instead of using these deprecated licenses, extract the actual",
			"license from the package into the pkgsrc/licenses/ directory",
			"and define LICENSE to that file name.  See the pkgsrc guide,",
			"keyword LICENSE, for more information.")
	}

	if len(cond.And) > 0 && len(cond.Or) > 0 {
		lc.MkLine.Line.Error0("AND and OR operators in license conditions can only be combined using parentheses.")
		Explain(
			"Examples for valid license conditions are:",
			"",
			"\tlicense1 AND license2 AND (license3 OR license4)",
			"\t(((license1 OR license2) AND (license3 OR license4)))")
	}
}
