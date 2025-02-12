package pkglint

// Vulnerabilities collects the vulnerabilites from the
// doc/pkg-vulnerabilities file.
type Vulnerabilities struct {
	byPkgbase map[string][]Vulnerability
}

type Vulnerability struct {
	line    *Line
	pattern *PackagePattern
	kind    string
	url     string
}

func NewVulnerabilities() *Vulnerabilities {
	return &Vulnerabilities{
		map[string][]Vulnerability{},
	}
}

func (vs *Vulnerabilities) read(filename CurrPath) {
	file := Load(filename, MustSucceed|NotEmpty)
	lines := file.Lines
	format := ""
	for len(lines) > 0 && hasPrefix(lines[0].Text, "#") {
		if hasPrefix(lines[0].Text, "#FORMAT ") {
			format = lines[0].Text[8:]
		}
		lines = lines[1:]
	}
	if format != "1.0.0" {
		file.Whole().Errorf("Invalid file format \"%s\".", format)
		return
	}

	for _, line := range lines {
		text := line.Text
		if hasPrefix(text, "#") {
			continue
		}
		m, pattern, kindOfExploit, url := match3(text, `^(\S+)\s+(\S+)\s+(\S+)$`)
		if !m {
			line.Errorf("Invalid line format \"%s\".", text)
			continue
		}
		if !hasBalancedBraces(pattern) {
			line.Errorf("Package pattern \"%s\" must have balanced braces.", pattern)
			continue
		}
		for _, pat := range expandCurlyBraces(pattern) {
			parser := NewMkParser(nil, pat)
			pp := ParsePackagePattern(parser)
			rest := parser.Rest()

			switch {
			case pp == nil && contains(pattern, "{"):
				line.Errorf("Package pattern \"%s\" expands to the invalid package pattern \"%s\".", pattern, pat)
				continue
			case pp == nil:
				line.Errorf("Invalid package pattern \"%s\".", pat)
				continue
			case hasPrefix(rest, "-") && contains(pattern, "{"):
				line.Errorf("Package pattern \"%s\" expands to \"%s\", which has a \"-\" in the version number.",
					pattern, pat)
				continue
			case hasPrefix(rest, "-"):
				line.Errorf("Package pattern \"%s\" has a \"-\" in the version number.", pat)
				continue
			case rest != "" && contains(pattern, "{"):
				line.Errorf("Package pattern \"%s\" expands to \"%s\", which is followed by extra text \"%s\".",
					pattern, pat[:len(pat)-len(rest)], rest)
				continue
			case rest != "":
				line.Errorf("Package pattern \"%s\" is followed by extra text \"%s\".", pat[:len(pat)-len(rest)], rest)
				continue
			}

			vs.byPkgbase[pp.Pkgbase] = append(vs.byPkgbase[pp.Pkgbase],
				Vulnerability{line, pp, kindOfExploit, url})
		}
	}
}
