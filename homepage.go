package pkglint

import (
	"net"
	"net/http"
	"syscall"
	"time"
)

// HomepageChecker runs the checks for a HOMEPAGE definition.
//
// When pkglint is in network mode (which has to be enabled explicitly using
// --network), it checks whether the homepage is actually reachable.
//
// The homepage URLs should use https as far as possible.
// To achieve this goal, the HomepageChecker can migrate homepages
// from less preferred URLs to preferred URLs.
//
// For most sites, the list of possible URLs is:
//  - https://$rest (preferred)
//  - http://$rest (less preferred)
//
// For SourceForge, it's a little more complicated:
//  - https://$project.sourceforge.io/$path
//  - http://$project.sourceforge.net/$path
//  - http://$project.sourceforge.io/$path (not officially supported)
//  - https://$project.sourceforge.net/$path (not officially supported)
//  - https://sourceforge.net/projects/$project/
//  - http://sourceforge.net/projects/$project/
//  - https://sf.net/projects/$project/
//  - http://sf.net/projects/$project/
//  - https://sf.net/p/$project/
//  - http://sf.net/p/$project/
//
// TODO: implement complete homepage migration for SourceForge.
// TODO: allow to suppress the automatic migration for SourceForge,
//  even if it is not about https vs. http.
type HomepageChecker struct {
	Value      string
	ValueNoVar string
	MkLine     *MkLine
	MkLines    *MkLines
}

func NewHomepageChecker(value string, valueNoVar string, mkline *MkLine, mklines *MkLines) *HomepageChecker {
	return &HomepageChecker{value, valueNoVar, mkline, mklines}
}

func (ck *HomepageChecker) Check() {
	ck.checkBasedOnMasterSites()
	ck.checkFtp()
	ck.checkHttp()
	ck.checkBadUrls()
	ck.checkReachable()
}

func (ck *HomepageChecker) checkBasedOnMasterSites() {
	m, wrong, sitename, subdir := match3(ck.Value, `^(\$\{(MASTER_SITE\w+)(?::=([\w\-/]+))?\})`)
	if !m {
		return
	}

	baseURL := G.Pkgsrc.MasterSiteVarToURL[sitename]
	if sitename == "MASTER_SITES" && ck.MkLines.pkg != nil {
		mkline := ck.MkLines.pkg.vars.FirstDefinition("MASTER_SITES")
		if mkline != nil {
			if !containsVarUse(mkline.Value()) {
				masterSites := ck.MkLine.ValueFields(mkline.Value())
				if len(masterSites) > 0 {
					baseURL = masterSites[0]
				}
			}
		}
	}

	fixedURL := baseURL + subdir

	fix := ck.MkLine.Autofix()
	if baseURL != "" {
		// TODO: Don't suggest any of checkBadUrls.
		fix.Warnf("HOMEPAGE should not be defined in terms of MASTER_SITEs. Use %s directly.", fixedURL)
	} else {
		fix.Warnf("HOMEPAGE should not be defined in terms of MASTER_SITEs.")
	}
	fix.Explain(
		"The HOMEPAGE is a single URL, while MASTER_SITES is a list of URLs.",
		"As long as this list has exactly one element, this works, but as",
		"soon as another site is added, the HOMEPAGE would not be a valid",
		"URL anymore.",
		"",
		"Defining MASTER_SITES=${HOMEPAGE} is ok, though.")
	if baseURL != "" {
		fix.Replace(wrong, fixedURL)
	}
	fix.Apply()
}

func (ck *HomepageChecker) checkFtp() {
	if !hasPrefix(ck.Value, "ftp://") {
		return
	}

	mkline := ck.MkLine
	if mkline.HasRationale("ftp", "FTP", "http", "https", "HTTP") {
		return
	}

	mkline.Warnf("An FTP URL does not represent a user-friendly homepage.")
	mkline.Explain(
		"This homepage URL has probably been generated by url2pkg",
		"and not been reviewed by the package author.",
		"",
		"In most cases there exists a more welcoming URL,",
		"which is usually served via HTTP.")
}

func (ck *HomepageChecker) checkHttp() {
	if ck.MkLine.HasRationale("http", "https") {
		return
	}

	shouldAutofix, from, to := ck.toHttps(ck.Value)
	if from == "" {
		return
	}

	fix := ck.MkLine.Autofix()
	fix.Warnf("HOMEPAGE should migrate from %s to %s.", from, to)
	if shouldAutofix {
		fix.Replace(from, to)
	}
	fix.Explain(
		"To provide secure communication by default,",
		"the HOMEPAGE URL should use the https protocol if available.",
		"",
		"If the HOMEPAGE really does not support https,",
		"add a comment near the HOMEPAGE variable stating this clearly.")
	fix.Apply()
}

// toHttps checks whether the homepage should be migrated from http to https
// and which part of the homepage URL needs to be modified for that.
//
// If for some reason the https URL should not be reachable but the
// corresponding http URL is, the homepage is changed back to http.
func (ck *HomepageChecker) toHttps(url string) (bool, string, string) {
	m, scheme, host, port := match3(url, `(https?)://([A-Za-z0-9-.]+)(:[0-9]+)?`)
	if !m {
		return false, "", ""
	}

	if ck.hasAnySuffix(host,
		"www.gnustep.org",           // 2020-01-18
		"aspell.net",                // 2020-01-18
		"downloads.sourceforge.net", // gets another warning already
		".dl.sourceforge.net",       // gets another warning already
	) {
		return false, "", ""
	}

	if scheme == "http" && ck.hasAnySuffix(host,
		"apache.org",
		"archive.org",
		"ctan.org",
		"freedesktop.org",
		"github.com",
		"github.io",
		"gnome.org",
		"gnu.org",
		"kde.org",
		"kldp.net",
		"linuxfoundation.org",
		"NetBSD.org",
		"nongnu.org",
		"tryton.org",
		"tug.org") {
		return port == "", "http", "https"
	}

	if scheme == "http" && host == "sf.net" {
		return port == "", "http://sf.net", "https://sourceforge.net"
	}

	from := scheme
	to := "https"
	toReachable := unknown

	// SourceForge projects use either http://project.sourceforge.net or
	// https://project.sourceforge.io (not net).
	if m, project := match1(host, `^([\w-]+)\.(?:sf|sourceforge)\.net$`); m {
		if scheme == "http" {
			from = scheme + "://" + host
			// See https://sourceforge.net/p/forge/documentation/Custom%20VHOSTs
			to = "https://" + project + ".sourceforge.io"
		} else {
			from = "sourceforge.net"
			to = "sourceforge.io"

			// Roll back wrong https SourceForge homepages generated by:
			// https://mail-index.netbsd.org/pkgsrc-changes/2020/01/18/msg205146.html
			if port == "" && G.Opts.Network {
				_, migrated := replaceOnce(url, from, to)
				if ck.isReachable(migrated) == no {
					ok, httpOnly := replaceOnce(url, "https://", "http://")
					if ok && ck.isReachable(httpOnly) == yes && ck.isReachable(url) == no {
						from = "https"
						to = "http"
						toReachable = yes
					}
				}
			}
		}
	}

	if from == to {
		return false, "", ""
	}

	shouldAutofix := toReachable == yes
	if port == "" && G.Opts.Network && toReachable == unknown {
		_, migrated := replaceOnce(url, from, to)
		toReachable = ck.isReachable(migrated)
		if toReachable == yes {
			shouldAutofix = true
		} else {
			return false, "", ""
		}
	}
	return shouldAutofix, from, to
}

func (ck *HomepageChecker) checkBadUrls() {
	m, host := match1(ck.Value, `https?://([A-Za-z0-9-.]+)`)
	if !m {
		return
	}

	if !ck.hasAnySuffix(host,
		".dl.sourceforge.net",
		"downloads.sourceforge.net") {
		return
	}

	mkline := ck.MkLine
	mkline.Warnf("A direct download URL is not a user-friendly homepage.")
	mkline.Explain(
		"This homepage URL has probably been generated by url2pkg",
		"and not been reviewed by the package author.",
		"",
		"In most cases there exists a more welcoming URL.")
}

func (ck *HomepageChecker) checkReachable() {
	mkline := ck.MkLine
	url := ck.Value

	if !G.Opts.Network || url != ck.ValueNoVar {
		return
	}
	if !matches(url, `^https?://[A-Za-z0-9-.]+(?::[0-9]+)?/[!-~]*$`) {
		return
	}

	var client http.Client
	client.Timeout = 3 * time.Second
	client.CheckRedirect = func(req *http.Request, via []*http.Request) error {
		return http.ErrUseLastResponse
	}

	request, err := http.NewRequest("GET", url, nil)
	if err != nil {
		mkline.Errorf("Invalid URL %q.", url)
		return
	}

	response, err := client.Do(request)
	if err != nil {
		networkError := ck.classifyNetworkError(err)
		mkline.Warnf("Homepage %q cannot be checked: %s", url, networkError)
		return
	}
	defer func() { _ = response.Body.Close() }()

	location, err := response.Location()
	if err == nil {
		mkline.Warnf("Homepage %q redirects to %q.", url, location.String())
		return
	}

	if response.StatusCode != 200 {
		mkline.Warnf("Homepage %q returns HTTP status %q.", url, response.Status)
		return
	}
}

func (*HomepageChecker) isReachable(url string) YesNoUnknown {
	switch {
	case !G.Opts.Network,
		containsVarRefLong(url),
		!matches(url, `^https?://[A-Za-z0-9-.]+(?::[0-9]+)?/[!-~]*$`):
		return unknown
	}

	var client http.Client
	client.Timeout = 3 * time.Second
	client.CheckRedirect = func(req *http.Request, via []*http.Request) error {
		return http.ErrUseLastResponse
	}

	request, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return no
	}
	response, err := client.Do(request)
	if err != nil {
		return no
	}
	_ = response.Body.Close()
	if response.StatusCode != 200 {
		return no
	}
	return yes
}

func (*HomepageChecker) hasAnySuffix(s string, suffixes ...string) bool {
	for _, suffix := range suffixes {
		if hasSuffix(s, suffix) {
			dotIndex := len(s) - len(suffix)
			if dotIndex == 0 || s[dotIndex-1] == '.' || suffix[0] == '.' {
				return true
			}
		}
	}
	return false
}

func (*HomepageChecker) classifyNetworkError(err error) string {
	cause := err
	for {
		// Unwrap was added in Go 1.13.
		// See https://github.com/golang/go/issues/36781
		if unwrap, ok := cause.(interface{ Unwrap() error }); ok {
			cause = unwrap.Unwrap()
			continue
		}
		break
	}

	// DNSError.IsNotFound was added in Go 1.13.
	// See https://github.com/golang/go/issues/28635
	if cause, ok := cause.(*net.DNSError); ok && cause.Err == "no such host" {
		return "name not found"
	}

	if cause, ok := cause.(syscall.Errno); ok {
		if cause == 10061 || cause == syscall.ECONNREFUSED {
			return "connection refused"
		}
	}

	if cause, ok := cause.(net.Error); ok && cause.Timeout() {
		return "timeout"
	}

	return sprintf("unknown network error: %s", err)
}
