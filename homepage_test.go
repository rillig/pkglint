package pkglint

import (
	"context"
	"gopkg.in/check.v1"
	"net/http"
	"strconv"
	"time"
)

func (s *Suite) Test_NewHomepageChecker(c *check.C) {
	t := s.Init(c)

	mklines := t.NewMkLines("filename.mk",
		"HOMEPAGE=\t# none")
	mkline := mklines.mklines[0]

	ck := NewHomepageChecker("value", "valueNoVar", mkline, mklines)

	t.CheckEquals(ck.Value, "value")
	t.CheckEquals(ck.ValueNoVar, "valueNoVar")
}

func (s *Suite) Test_HomepageChecker_Check(c *check.C) {
	t := s.Init(c)

	mklines := t.NewMkLines("filename.mk",
		"HOMEPAGE=\tftp://example.org/")
	mkline := mklines.mklines[0]
	value := mkline.Value()

	ck := NewHomepageChecker(value, value, mkline, mklines)

	ck.Check()

	t.CheckOutputLines(
		"WARN: filename.mk:1: An FTP URL does not represent a user-friendly homepage.")
}

func (s *Suite) Test_HomepageChecker_checkBasedOnMasterSites(c *check.C) {
	t := s.Init(c)
	vt := NewVartypeCheckTester(t, BtHomepage)

	vt.Varname("HOMEPAGE")
	vt.Values(
		"${MASTER_SITES}")

	vt.Output(
		"WARN: filename.mk:1: HOMEPAGE should not be defined in terms of MASTER_SITEs.")

	pkg := NewPackage(t.File("category/package"))
	vt.Package(pkg)

	vt.Values(
		"${MASTER_SITES}")

	// When this assignment occurs while checking a package, but the package
	// doesn't define MASTER_SITES, that variable cannot be expanded, which means
	// the warning cannot suggest a replacement value.
	vt.Output(
		"WARN: filename.mk:11: HOMEPAGE should not be defined in terms of MASTER_SITEs.")

	delete(pkg.vars.firstDef, "MASTER_SITES")
	delete(pkg.vars.lastDef, "MASTER_SITES")
	pkg.vars.Define("MASTER_SITES", t.NewMkLine(pkg.File("Makefile"), 5,
		"MASTER_SITES=\thttps://cdn.NetBSD.org/pub/pkgsrc/distfiles/"))

	vt.Values(
		"${MASTER_SITES}")

	vt.Output(
		"WARN: filename.mk:21: HOMEPAGE should not be defined in terms of MASTER_SITEs. " +
			"Use https://cdn.NetBSD.org/pub/pkgsrc/distfiles/ directly.")

	delete(pkg.vars.firstDef, "MASTER_SITES")
	delete(pkg.vars.lastDef, "MASTER_SITES")
	pkg.vars.Define("MASTER_SITES", t.NewMkLine(pkg.File("Makefile"), 5,
		"MASTER_SITES=\t${MASTER_SITE_GITHUB}"))

	vt.Values(
		"${MASTER_SITES}")

	// When MASTER_SITES itself makes use of another variable, pkglint doesn't
	// resolve that variable and just outputs the simple variant of this warning.
	vt.Output(
		"WARN: filename.mk:31: HOMEPAGE should not be defined in terms of MASTER_SITEs.")

	delete(pkg.vars.firstDef, "MASTER_SITES")
	delete(pkg.vars.lastDef, "MASTER_SITES")
	pkg.vars.Define("MASTER_SITES", t.NewMkLine(pkg.File("Makefile"), 5,
		"MASTER_SITES=\t# none"))

	vt.Values(
		"${MASTER_SITES}")

	// When MASTER_SITES is empty, pkglint cannot extract the first of the URLs
	// for using it in the HOMEPAGE.
	vt.Output(
		"WARN: filename.mk:41: HOMEPAGE should not be defined in terms of MASTER_SITEs.")
}

func (s *Suite) Test_HomepageChecker_checkFtp(c *check.C) {
	t := s.Init(c)
	vt := NewVartypeCheckTester(t, BtHomepage)

	vt.Varname("HOMEPAGE")
	vt.Values(
		"ftp://example.org/",
		"ftp://example.org/ # no HTTP homepage available")

	vt.Output(
		"WARN: filename.mk:1: " +
			"An FTP URL does not represent a user-friendly homepage.")
}

func (s *Suite) Test_HomepageChecker_checkHttp(c *check.C) {
	t := s.Init(c)
	vt := NewVartypeCheckTester(t, BtHomepage)

	vt.Varname("HOMEPAGE")
	vt.Values(
		"http://www.gnustep.org/",
		"http://www.pkgsrc.org/",
		"http://project.sourceforge.net/",
		"http://sf.net/p/project/",
		"http://sourceforge.net/p/project/",
		"http://example.org/ # doesn't support https",
		"http://example.org/ # only supports http",
		"http://asf.net/")

	vt.Output(
		"WARN: filename.mk:2: HOMEPAGE should migrate from http to https.",
		"WARN: filename.mk:4: HOMEPAGE should migrate from http://sf.net to https://sourceforge.net.",
		"WARN: filename.mk:5: HOMEPAGE should migrate from http to https.",
		"WARN: filename.mk:8: HOMEPAGE should migrate from http to https.")

	t.SetUpCommandLine("--autofix")
	vt.Values(
		"http://www.gnustep.org/",
		"http://www.pkgsrc.org/",
		"http://project.sourceforge.net/",
		"http://sf.net/p/project/",
		"http://sourceforge.net/p/project/",
		"http://example.org/ # doesn't support https",
		"http://example.org/ # only supports http",
		"http://kde.org/",
		"http://asf.net/")

	// www.gnustep.org does not support https at all.
	// www.pkgsrc.org is not in the (short) list of known https domains,
	// therefore pkglint does not dare to change it automatically.
	vt.Output(
		"AUTOFIX: filename.mk:18: Replacing \"http\" with \"https\".")
}

func (s *Suite) Test_HomepageChecker_checkBadUrls(c *check.C) {
	t := s.Init(c)
	vt := NewVartypeCheckTester(t, BtHomepage)

	vt.Varname("HOMEPAGE")
	vt.Values(
		"http://garr.dl.sourceforge.net/project/name/dir/subdir/",
		"https://downloads.sourceforge.net/project/name/dir/subdir/")

	vt.Output(
		"WARN: filename.mk:1: A direct download URL is not a user-friendly homepage.",
		"WARN: filename.mk:2: A direct download URL is not a user-friendly homepage.")
}

func (s *Suite) Test_HomepageChecker_checkReachable(c *check.C) {
	t := s.Init(c)
	vt := NewVartypeCheckTester(t, BtHomepage)

	t.SetUpCommandLine("--network")

	mux := http.NewServeMux()
	mux.HandleFunc("/status/", func(writer http.ResponseWriter, request *http.Request) {
		location := request.URL.Query().Get("location")
		if location != "" {
			writer.Header().Set("Location", location)
		}

		status, err := strconv.Atoi(request.URL.Path[len("/status/"):])
		assertNil(err, "")
		writer.WriteHeader(status)
	})
	mux.HandleFunc("/timeout", func(writer http.ResponseWriter, request *http.Request) {
		time.Sleep(5 * time.Second)
	})

	// 28780 = 256 * 'p' + 'l'
	srv := http.Server{Addr: "localhost:28780", Handler: mux}
	shutdown := make(chan error)

	go func() {
		err := srv.ListenAndServe()
		assertf(err == http.ErrServerClosed, "%s", err)
		shutdown <- err
	}()

	defer func() {
		err := srv.Shutdown(context.Background())
		assertNil(err, "")
		<-shutdown
	}()

	vt.Varname("HOMEPAGE")
	vt.Values(
		"http://localhost:28780/status/200",
		"http://localhost:28780/status/301?location=/redirect301",
		"http://localhost:28780/status/302?location=/redirect302",
		"http://localhost:28780/status/307?location=/redirect307",
		"http://localhost:28780/status/404",
		"http://localhost:28780/status/500")

	vt.Output(
		"WARN: filename.mk:1: HOMEPAGE should migrate from http to https.",
		"WARN: filename.mk:2: HOMEPAGE should migrate from http to https.",
		"WARN: filename.mk:2: Status: 301 Moved Permanently, "+
			"location: http://localhost:28780/redirect301",
		"WARN: filename.mk:3: HOMEPAGE should migrate from http to https.",
		"WARN: filename.mk:3: Status: 302 Found, "+
			"location: http://localhost:28780/redirect302",
		"WARN: filename.mk:4: HOMEPAGE should migrate from http to https.",
		"WARN: filename.mk:4: Status: 307 Temporary Redirect, "+
			"location: http://localhost:28780/redirect307",
		"WARN: filename.mk:5: HOMEPAGE should migrate from http to https.",
		"WARN: filename.mk:5: Status: 404 Not Found",
		"WARN: filename.mk:6: HOMEPAGE should migrate from http to https.",
		"WARN: filename.mk:6: Status: 500 Internal Server Error")

	vt.Values(
		"http://localhost:28780/timeout",
		"http://localhost:28780/%invalid",
		"http://localhost:28781/",
		"https://no-such-name.example.org/")

	vt.Output(
		"WARN: filename.mk:11: HOMEPAGE should migrate from http to https.",
		"WARN: filename.mk:11: Homepage \"http://localhost:28780/timeout\" "+
			"cannot be checked: timeout",
		"WARN: filename.mk:12: HOMEPAGE should migrate from http to https.",
		"ERROR: filename.mk:12: Invalid URL \"http://localhost:28780/%invalid\".",
		"WARN: filename.mk:13: HOMEPAGE should migrate from http to https.",
		"WARN: filename.mk:13: Homepage \"http://localhost:28781/\" "+
			"cannot be checked: connection refused",
		"WARN: filename.mk:14: Homepage \"https://no-such-name.example.org/\" "+
			"cannot be checked: name not found")
}

func (s *Suite) Test_HomepageChecker_hasAnySuffix(c *check.C) {
	t := s.Init(c)

	test := func(s string, hasAnySuffix bool, suffixes ...string) {
		actual := (*HomepageChecker).hasAnySuffix(nil, s, suffixes...)

		t.CheckEquals(actual, hasAnySuffix)
	}

	test("example.org", true, "org")
	test("example.com", false, "org")
	test("example.org", true, "example.org")
	test("example.org", false, ".example.org")
	test("example.org", true, ".org")
}
