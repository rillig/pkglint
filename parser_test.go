package pkglint

import (
	"gopkg.in/check.v1"
)

func (s *Suite) Test_Parser_PkgbasePattern(c *check.C) {

	testRest := func(pattern, expected, rest string) {
		parser := NewParser(dummyLine, pattern, false)
		actual := parser.PkgbasePattern()
		c.Check(actual, equals, expected)
		c.Check(parser.Rest(), equals, rest)
	}

	testRest("fltk", "fltk", "")
	testRest("fltk|", "fltk", "|")
	testRest("boost-build-1.59.*", "boost-build", "-1.59.*")
	testRest("${PHP_PKG_PREFIX}-pdo-5.*", "${PHP_PKG_PREFIX}-pdo", "-5.*")
	testRest("${PYPKGPREFIX}-metakit-[0-9]*", "${PYPKGPREFIX}-metakit", "-[0-9]*")
}
