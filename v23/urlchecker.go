package pkglint

import "strings"

type UrlChecker struct {
	varname string
	op      MkOperator
	mkline  *MkLine
	mklines *MkLines
}

func NewUrlChecker(cv *VartypeCheck) *UrlChecker {
	return &UrlChecker{cv.Varname, cv.Op, cv.MkLine, cv.MkLines}
}

func (ck *UrlChecker) CheckFetchURL(fetchURL string) {
	url := strings.TrimPrefix(fetchURL, "-")
	hyphen := condStr(len(fetchURL) > len(url), "-", "")
	hyphenSubst := condStr(hyphen != "", ":S,^,-,", "")

	ck.CheckURL(url)

	trimURL := url[len(url)-len(replaceAll(url, `^\w+://`, "")):]
	protoLen := len(url) - len(trimURL)

	for trimSiteURL, siteName := range G.Pkgsrc.MasterSiteURLToVar {
		if !hasPrefix(trimURL, trimSiteURL) {
			continue
		}
		if siteName == "MASTER_SITE_GITHUB" &&
			hasPrefix(ck.varname, "SITES.") {
			continue
		}

		subdir := trimURL[len(trimSiteURL):]
		if hasPrefix(trimURL, "github.com/") {
			subdir = strings.SplitAfter(subdir, "/")[0]
			commonPrefix := hyphen + url[:protoLen+len(trimSiteURL)+len(subdir)]
			ck.mkline.Warnf("Use ${%s%s:=%s} instead of %q and run %q for further instructions.",
				siteName, hyphenSubst, subdir, commonPrefix, bmakeHelp("github"))
		} else {
			ck.mkline.Warnf("Use ${%s%s:=%s} instead of %q.", siteName, hyphenSubst, subdir, hyphen+url)
		}
		return
	}

	tokens := ck.mkline.Tokenize(url, false)
	for _, token := range tokens {
		expr := token.Expr
		if expr == nil {
			continue
		}

		name := expr.varname
		if !hasPrefix(name, "MASTER_SITE_") {
			continue
		}

		if name == "MASTER_SITE_BACKUP" {
			if !ck.mkline.HasRationale() {
				ck.mkline.Warnf("The site MASTER_SITE_BACKUP should not be used.")
				ck.mkline.Explain(
					"MASTER_SITE_BACKUP is hosted by NetBSD",
					"and is only for backup purposes.",
					"Each package should have a primary MASTER_SITE",
					"outside the NetBSD Foundation.")
			}
		} else if G.Pkgsrc.MasterSiteVarToURL[name] == "" {
			if ck.mklines.pkg == nil || !ck.mklines.pkg.vars.IsDefined(name) {
				ck.mkline.Errorf("The site %s does not exist.", name)
			}
		}

		if len(expr.modifiers) != 1 || !expr.modifiers[0].HasPrefix("=") {
			continue
		}

		subdir := expr.modifiers[0].String()[1:]
		if !hasSuffix(subdir, "/") {
			ck.mkline.Errorf("The subdirectory in %s must end with a slash.", name)
		}
	}

	switch {
	case ck.op == opUseMatch,
		hasSuffix(fetchURL, "/"),
		hasSuffix(fetchURL, "="),
		hasSuffix(fetchURL, ":"),
		hasPrefix(fetchURL, "-"),
		tokens[len(tokens)-1].Expr != nil:
		break

	default:
		ck.mkline.Warnf("The fetch URL %q should end with a slash.", fetchURL)
		ck.mkline.Explain(
			"The filename from DISTFILES is appended directly to this base URL.",
			"Therefore, it should typically end with a slash, or sometimes with",
			"an equals sign or a colon.",
			"",
			"To specify a full URL directly, prefix it with a hyphen, such as in",
			"-https://example.org/distfile-1.0.tar.gz.")
	}
}

func (ck *UrlChecker) CheckURL(url string) {
	value := url

	if value == "" && ck.mkline.HasComment() {
		// Ok

	} else if containsExpr(value) {
		// No further checks

	} else if m, host := match1(value, `^(?:https?|ftp|gopher)://([-0-9A-Za-z.]+)(?::\d+)?/[-#%&+,./0-9:;=?@A-Z_a-z~]*$`); m {
		if matches(host, `(?i)\.NetBSD\.org$`) && !matches(host, `\.NetBSD\.org$`) {
			prefix := host[:len(host)-len(".NetBSD.org")]
			fix := ck.mkline.Autofix()
			fix.Warnf("Write NetBSD.org instead of %s.", host)
			fix.Replace(host, prefix+".NetBSD.org")
			fix.Apply()
		}

	} else if m, scheme, _, absPath := match3(value, `^([0-9A-Za-z]+)://([^/]+)(.*)$`); m {
		switch {
		case scheme != "ftp" && scheme != "http" && scheme != "https" && scheme != "gopher":
			ck.mkline.Warnf("%q is not a valid URL. Only ftp, gopher, http, and https URLs are allowed here.", value)

		case absPath == "":
			ck.mkline.Notef("For consistency, add a trailing slash to %q.", value)

		default:
			ck.mkline.Warnf("%q is not a valid URL.", value)
		}

	} else {
		ck.mkline.Warnf("%q is not a valid URL.", value)
	}
}
