//go:build ignore
// +build ignore

package pkglint

import "testing"

func Test_MkAlignFile_AlignParas(tt *testing.T) {
	t := NewTester(tt)
	defer t.Done()

	test := func() {
		// TODO
		t.CheckEquals(true, true)
	}

	test()
}

func Test_MkAlignPara_IsAligned(tt *testing.T) {
	t := NewTester(tt)
	defer t.Done()

	test := func() {
		// TODO
		t.CheckEquals(true, true)
	}

	test()
}

func Test_MkAlignPara_IsOutlier(tt *testing.T) {
	t := NewTester(tt)
	defer t.Done()

	test := func() {
		// TODO
		t.CheckEquals(true, true)
	}

	test()
}

func Test_MkAlignPara_ValueAlignment(tt *testing.T) {
	t := NewTester(tt)
	defer t.Done()

	test := func() {
		// TODO
		t.CheckEquals(true, true)
	}

	test()
}

func Test_MkAlignPara_MinValueAlignment(tt *testing.T) {
	t := NewTester(tt)
	defer t.Done()

	test := func() {
		// TODO
		t.CheckEquals(true, true)
	}

	test()
}

func Test_MkAlignPara_MaxValueAlignment(tt *testing.T) {
	t := NewTester(tt)
	defer t.Done()

	test := func() {
		// TODO
		t.CheckEquals(true, true)
	}

	test()
}

func Test_MkAlignPara_MayAlignValuesTo(tt *testing.T) {
	t := NewTester(tt)
	defer t.Done()

	test := func() {
		// TODO
		t.CheckEquals(true, true)
	}

	test()
}

func Test_MkAlignPara_AlignValuesTo(tt *testing.T) {
	t := NewTester(tt)
	defer t.Done()

	test := func() {
		// TODO
		t.CheckEquals(true, true)
	}

	test()
}

func Test_MkAlignMkLine_RightMargin(tt *testing.T) {
	t := NewTester(tt)
	defer t.Done()

	test := func() {
		// TODO
		t.CheckEquals(true, true)
	}

	test()
}

func Test_MkAlignMkLine_IsCanonical(tt *testing.T) {
	t := NewTester(tt)
	defer t.Done()

	test := func() {
		// TODO
		t.CheckEquals(true, true)
	}

	test()
}

func Test_MkAlignMkLine_HasCanonicalRightMargin(tt *testing.T) {
	t := NewTester(tt)
	defer t.Done()

	test := func() {
		// TODO
		t.CheckEquals(true, true)
	}

	test()
}

func Test_MkAlignMkLine_CurrentValueAlign(tt *testing.T) {
	t := NewTester(tt)
	defer t.Done()

	test := func() {
		// TODO
		t.CheckEquals(true, true)
	}

	test()
}

func Test_NewMkAlignLine(tt *testing.T) {
	t := NewTester(tt)
	defer t.Done()

	// TODO

	t.CheckOutputEmpty()
}

func Test_MkAlignLine_HasCanonicalRightMargin(tt *testing.T) {
	t := NewTester(tt)
	defer t.Done()

	test := func() {
		// TODO
		t.CheckEquals(true, true)
	}

	test()
}

func Test_MkAlignLine_IsCanonicalSingle(tt *testing.T) {
	t := NewTester(tt)
	defer t.Done()

	test := func() {
		// TODO
		t.CheckEquals(true, true)
	}

	test()
}

func Test_MkAlignLine_IsCanonicalLeadEmpty(tt *testing.T) {
	t := NewTester(tt)
	defer t.Done()

	test := func() {
		// TODO
		t.CheckEquals(true, true)
	}

	test()
}

func Test_MkAlignLine_IsCanonicalLeadValue(tt *testing.T) {
	t := NewTester(tt)
	defer t.Done()

	test := func() {
		// TODO
		t.CheckEquals(true, true)
	}

	test()
}

func Test_MkAlignLine_IsCanonicalFollowLead(tt *testing.T) {
	t := NewTester(tt)
	defer t.Done()

	test := func() {
		// TODO
		t.CheckEquals(true, true)
	}

	test()
}

func Test_MkAlignLine_IsCanonicalFollow(tt *testing.T) {
	t := NewTester(tt)
	defer t.Done()

	test := func(line string, minAlign int, isCanonical bool) {
		t.CheckDotColumns(line)

		parts := NewVaralignSplitter().split(line, false)
		al := NewMkAlignLine(
			parts.leadingComment, parts.varnameOp, parts.spaceBeforeValue,
			parts.value, parts.spaceAfterValue, parts.continuation)

		actualIsCanonical := al.IsCanonicalFollow(minAlign)

		t.CheckEquals(actualIsCanonical, isCanonical)
	}

	test("\tvalue", 0, true)
	test("\tvalue", 8, true)
	test("\tvalue", 9, false)

	// The indentation width is correct, but there is an additional
	// space in the indentation. That space must be replaced with tabs,
	// which in this case means it is simply removed.
	test("\t \tvalue", 16, false)

	// TODO: Why should pkglint care about the right margin?
	//  If there is an existing right margin, it should be kept as-is,
	//  but otherwise, why not let the pkgsrc developers fix this
	//  themselves?

	// This line is 63 characters wide.
	// It should be indented with one more tab.
	//
	// After indenting it, it is 71 characters wide,
	// which is exactly the maximum right border
	// for lines without a continuation backslash.
	test("\tv\t\t\t\t\t\t.....63", 16, false)

	// This line is 64 characters wide.
	// It should be indented with one more tab.
	//
	// After indenting it, it is 72 characters wide,
	// which is just beyond the maximum right border
	// for lines without a continuation backslash.
	// Therefore, it counts as canonical.
	//
	// XXX: Is it really worth having this rule?
	//  It may be equally ok to just have the continuation
	//  backslash further to the right.
	test("\tv\t\t\t\t\t\t......64", 16, true)

	// This line already overflows the right margin.
	// On an 80-column display, it is not decidable whether this line
	// continues to the right or whether it stops there.
	// It wouldn't hurt to make the line even longer.
	// Still, it is ok to keep the indentation at a minimum.
	//
	// Splitting this line into several shorter lines would require
	// too much knowledge, therefore this task is left to the pkgsrc
	// developers.
	test("\tv\t\t\t\t\t\t\t\t.....79", 16, true)
}
