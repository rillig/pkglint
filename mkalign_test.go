// +build ignore

package pkglint

import "gopkg.in/check.v1"

func (s *Suite) Test_MkAlignFile_AlignParas(c *check.C) {
	t := s.Init(c)

	test := func() {
		// TODO
		t.CheckEquals(true, true)
	}

	test()
}

func (s *Suite) Test_MkAlignPara_IsAligned(c *check.C) {
	t := s.Init(c)

	test := func() {
		// TODO
		t.CheckEquals(true, true)
	}

	test()
}

func (s *Suite) Test_MkAlignPara_IsOutlier(c *check.C) {
	t := s.Init(c)

	test := func() {
		// TODO
		t.CheckEquals(true, true)
	}

	test()
}

func (s *Suite) Test_MkAlignPara_ValueAlignment(c *check.C) {
	t := s.Init(c)

	test := func() {
		// TODO
		t.CheckEquals(true, true)
	}

	test()
}

func (s *Suite) Test_MkAlignPara_MinValueAlignment(c *check.C) {
	t := s.Init(c)

	test := func() {
		// TODO
		t.CheckEquals(true, true)
	}

	test()
}

func (s *Suite) Test_MkAlignPara_MaxValueAlignment(c *check.C) {
	t := s.Init(c)

	test := func() {
		// TODO
		t.CheckEquals(true, true)
	}

	test()
}

func (s *Suite) Test_MkAlignPara_MayAlignValuesTo(c *check.C) {
	t := s.Init(c)

	test := func() {
		// TODO
		t.CheckEquals(true, true)
	}

	test()
}

func (s *Suite) Test_MkAlignPara_AlignValuesTo(c *check.C) {
	t := s.Init(c)

	test := func() {
		// TODO
		t.CheckEquals(true, true)
	}

	test()
}

func (s *Suite) Test_MkAlignMkLine_RightMargin(c *check.C) {
	t := s.Init(c)

	test := func() {
		// TODO
		t.CheckEquals(true, true)
	}

	test()
}

func (s *Suite) Test_MkAlignMkLine_IsCanonical(c *check.C) {
	t := s.Init(c)

	test := func() {
		// TODO
		t.CheckEquals(true, true)
	}

	test()
}

func (s *Suite) Test_MkAlignMkLine_HasCanonicalRightMargin(c *check.C) {
	t := s.Init(c)

	test := func() {
		// TODO
		t.CheckEquals(true, true)
	}

	test()
}

func (s *Suite) Test_MkAlignMkLine_CurrentValueAlign(c *check.C) {
	t := s.Init(c)

	test := func() {
		// TODO
		t.CheckEquals(true, true)
	}

	test()
}

func (s *Suite) Test_NewMkAlignLine(c *check.C) {
	t := s.Init(c)

	// TODO

	t.CheckOutputEmpty()
}

func (s *Suite) Test_MkAlignLine_HasCanonicalRightMargin(c *check.C) {
	t := s.Init(c)

	test := func() {
		// TODO
		t.CheckEquals(true, true)
	}

	test()
}

func (s *Suite) Test_MkAlignLine_IsCanonicalSingle(c *check.C) {
	t := s.Init(c)

	test := func() {
		// TODO
		t.CheckEquals(true, true)
	}

	test()
}

func (s *Suite) Test_MkAlignLine_IsCanonicalLeadEmpty(c *check.C) {
	t := s.Init(c)

	test := func() {
		// TODO
		t.CheckEquals(true, true)
	}

	test()
}

func (s *Suite) Test_MkAlignLine_IsCanonicalLeadValue(c *check.C) {
	t := s.Init(c)

	test := func() {
		// TODO
		t.CheckEquals(true, true)
	}

	test()
}

func (s *Suite) Test_MkAlignLine_IsCanonicalFollowLead(c *check.C) {
	t := s.Init(c)

	test := func() {
		// TODO
		t.CheckEquals(true, true)
	}

	test()
}

func (s *Suite) Test_MkAlignLine_IsCanonicalFollow(c *check.C) {
	t := s.Init(c)

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
	// Therefore it counts as canonical.
	//
	// XXX: Is it really worth having this rule?
	//  It may be equally ok to just have the continuation
	//  backslash further to the right.
	test("\tv\t\t\t\t\t\t......64", 16, true)

	// This line already already overflows the right margin.
	// On an 80-column display it is not decidable whether this line
	// continues to the right or whether it stops there.
	// It wouldn't hurt to make the line even longer.
	// Still, it is ok to keep the indentation at a minimum.
	//
	// Splitting this line into several shorter lines would require
	// too much knowledge, therefore this task is left to the pkgsrc
	// developers.
	test("\tv\t\t\t\t\t\t\t\t.....79", 16, true)
}
