package histogram_test

import (
	"github.com/rillig/pkglint/v23/histogram"
	"github.com/rillig/pkglint/v23/intqa"
	"strings"
	"testing"
)

func Test_Histogram(t *testing.T) {
	hgr := histogram.New()
	hgr.Add("one", 1)
	hgr.Add("two", 2)
	hgr.Add("three", 3)

	var out strings.Builder
	hgr.PrintStats(&out, "caption", 2)

	if out.String() != ""+
		"caption      3 three\n"+
		"caption      2 two\n" {
		t.Error(out)
	}
}

func Test__qa(t *testing.T) {
	ck := intqa.NewQAChecker(t.Errorf)
	ck.Configure("*", "*", "*", -intqa.EMissingTest)
	ck.Check()
}
