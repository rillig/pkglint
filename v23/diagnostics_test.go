package pkglint

import "testing"

func Test_Diagnostics_Defer(tt *testing.T) {
	t := NewTester(tt)
	defer t.Done()
	t.SetUpCommandLine("--explain")

	line := t.NewLine("filename.mk", 123, "Text")
	var d Diagnostics
	d.Defer(line).Errorf("Line must contain %q.", "error text")
	d.Defer(line).Explain("Explanation")

	d.Emit(line)

	t.CheckOutputLines(
		"ERROR: filename.mk:123: Line must contain \"error text\".",
		"",
		"\tExplanation",
		"",
	)
}

func Test_Diagnostics_add(tt *testing.T) {
	t := NewTester(tt)
	defer t.Done()

	line1 := t.NewLine("filename.mk", 1, "Line 1")
	line2 := t.NewLine("filename.mk", 2, "Line 2")

	var d Diagnostics
	d.add(line1, Error, "Line must contain %q.", "error text")
	d.add(line2, Warn, "Line should contain %q.", "warning text")
	d.add(line1, Note, "Line could contain %q.", "note text")

	t.CheckEquals(len(d.diagnostics[line1]), 2)
	t.CheckEquals(len(d.diagnostics[line2]), 1)
}

func Test_Diagnostics_explain(tt *testing.T) {
	t := NewTester(tt)
	defer t.Done()

	t.SetUpCommandLine("--explain")

	line1 := t.NewLine("filename.mk", 1, "Line 1")

	var d Diagnostics
	t.ExpectPanic(func() {
		d.explain(line1, "Explanation 1.")
	}, "Pkglint internal error: explanation without corresponding diagnostic")
	d.add(line1, Warn, "Line should contain \"%s\".", "warning text")
	d.explain(line1, "Explanation 1.")

	d.Emit(line1)
	d.AssertEmpty()

	t.CheckOutputLines(
		"WARN: filename.mk:1: Line should contain \"warning text\".",
		"",
		"\tExplanation 1.",
		"",
	)
}

func Test_Diagnostics_Emit(tt *testing.T) {
	t := NewTester(tt)
	defer t.Done()

	line1 := t.NewLine("filename.mk", 1, "Line 1")
	line2 := t.NewLine("filename.mk", 2, "Line 2")

	var d Diagnostics
	d.add(line1, Error, "Line must contain %q.", "error text")
	d.add(line2, Warn, "Line should contain %q.", "warning text")
	d.add(line1, Note, "Line could contain %q.", "note text")
	d.add(line1, Error, "Line must equal %q.", "replacement")

	t.SetUpCommandLine("--only", "contain")

	for _, line := range []*Line{line1, line2} {
		d.Emit(line)
	}
	d.AssertEmpty()

	// First the diagnostics for line 1, then for line 2.
	// For each line, the diagnostics are in insertion order.
	t.CheckOutputLines(
		"ERROR: filename.mk:1: Line must contain \"error text\".",
		"NOTE: filename.mk:1: Line could contain \"note text\".",
		"WARN: filename.mk:2: Line should contain \"warning text\".",
	)
}

func Test_Diagnostics_AssertEmpty(tt *testing.T) {
	t := NewTester(tt)
	defer t.Done()

	line1 := t.NewLine("filename.mk", 1, "Line 1")
	line2 := t.NewLine("filename.mk", 2, "Line 2")

	var d Diagnostics
	d.add(line1, Error, "Line must contain %q.", "error text")
	d.add(line2, Warn, "Line should contain %q.", "warning text")
	d.add(line1, Note, "Line could contain %q.", "note text")

	d.Emit(line1)
	// But not d.Emit(line2)
	_ = t.Output()

	t.ExpectPanic(
		d.AssertEmpty,
		"filename.mk:2: Line 2")
}

func Test_DeferredDiagnoser_Errorf(tt *testing.T) {
	t := NewTester(tt)
	defer t.Done()

	line := t.NewLine("filename.mk", 123, "Text")
	var d Diagnostics
	d.Defer(line).Errorf("Line must contain %q.", "error text")

	d.Emit(line)

	t.CheckOutputLines(
		"ERROR: filename.mk:123: Line must contain \"error text\".")
}

func Test_DeferredDiagnoser_Warnf(tt *testing.T) {
	t := NewTester(tt)
	defer t.Done()

	line := t.NewLine("filename.mk", 123, "Text")
	var d Diagnostics
	d.Defer(line).Warnf("Line should contain %q.", "warning text")

	d.Emit(line)

	t.CheckOutputLines(
		"WARN: filename.mk:123: Line should contain \"warning text\".")
}

func Test_DeferredDiagnoser_Notef(tt *testing.T) {
	t := NewTester(tt)
	defer t.Done()

	line := t.NewLine("filename.mk", 123, "Text")
	var d Diagnostics
	d.Defer(line).Notef("Line may also contain %q.", "note text")

	d.Emit(line)

	t.CheckOutputLines(
		"NOTE: filename.mk:123: Line may also contain \"note text\".")
}

func Test_DeferredDiagnoser_Explain(tt *testing.T) {
	t := NewTester(tt)
	defer t.Done()
	t.SetUpCommandLine("--explain")

	line := t.NewLine("filename.mk", 123, "Text")
	var d Diagnostics
	d.Defer(line).Errorf("Line must contain %q.", "error text")
	d.Defer(line).Explain("Explanation")

	d.Emit(line)

	t.CheckOutputLines(
		"ERROR: filename.mk:123: Line must contain \"error text\".",
		"",
		"\tExplanation",
		"",
	)
}
