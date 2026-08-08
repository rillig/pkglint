package trace

import (
	"bytes"
	"github.com/rillig/pkglint/v23/intqa"
	"testing"
)

func (t *Tracer) onlyArguments(args ...any) {
	defer t.Call(args...)()
	t.Stepf("Running %q", "code")
}

func (t *Tracer) argumentsAndResult(arg0 string, arg1 int) (result string) {
	defer t.Call(arg0, arg1, t.Result(&result))()
	t.Stepf("Running %q", "code")
	return "the result"
}

func (t *Tracer) argumentsAndResultWrong(arg0 string, arg1 int) (result string) {
	defer t.Call(arg0, arg1, result)() // "result" is evaluated too early and only once.
	t.Stepf("Running %q", "code")
	return "the result"
}

func Test_Tracer_Call__only_arguments(t *testing.T) {
	var tracer Tracer

	output := captureTracingOutput(&tracer, func() {
		tracer.onlyArguments("arg0", 1234)
	})

	if have, want := output, ""+
		"TRACE: + github.com/rillig/pkglint/v23/trace.(*Tracer).onlyArguments(\"arg0\", 1234)\n"+
		"TRACE: 1   Running \"code\"\n"+
		"TRACE: - github.com/rillig/pkglint/v23/trace.(*Tracer).onlyArguments(\"arg0\", 1234)\n"; have != want {
		t.Errorf("want %#v, have %#v", want, have)
	}
}

func Test_Tracer_Call__arguments_and_result(t *testing.T) {
	var tracer Tracer

	output := captureTracingOutput(&tracer, func() {
		tracer.argumentsAndResult("arg0", 1234)
	})

	if have, want := output, ""+
		"TRACE: + github.com/rillig/pkglint/v23/trace.(*Tracer).argumentsAndResult(\"arg0\", 1234)\n"+
		"TRACE: 1   Running \"code\"\n"+
		"TRACE: - github.com/rillig/pkglint/v23/trace.(*Tracer).argumentsAndResult(\"arg0\", 1234, \"=>\", \"the result\")\n"; have != want {
		t.Errorf("want %#v, have %#v", want, have)
	}
}

func Test_Tracer_Call__arguments_and_result_wrong(t *testing.T) {
	var tracer Tracer

	output := captureTracingOutput(&tracer, func() {
		tracer.argumentsAndResultWrong("arg0", 1234)
	})

	if have, want := output, ""+
		"TRACE: + github.com/rillig/pkglint/v23/trace.(*Tracer).argumentsAndResultWrong(\"arg0\", 1234, \"\")\n"+
		"TRACE: 1   Running \"code\"\n"+
		"TRACE: - github.com/rillig/pkglint/v23/trace.(*Tracer).argumentsAndResultWrong(\"arg0\", 1234, \"\")\n"; have != want {
		t.Errorf("want %#v, have %#v", want, have)
	}
}

func Test__fixed_argument_variants(t *testing.T) {
	var tracer Tracer

	output := captureTracingOutput(&tracer, func() {
		defer tracer.Call0()()
		defer tracer.Call1("x")()
		defer tracer.Call2("x", "y")()
		tracer.Step1("step %s", "a")
		tracer.Step2("step %s, %s", "a", "b")
	})

	if have, want := output, ""+
		"TRACE: + github.com/rillig/pkglint/v23/trace.Test__fixed_argument_variants.func1()\n"+
		"TRACE: 1 + github.com/rillig/pkglint/v23/trace.Test__fixed_argument_variants.func1(\"x\")\n"+
		"TRACE: 1 2 + github.com/rillig/pkglint/v23/trace.Test__fixed_argument_variants.func1(\"x\", \"y\")\n"+
		"TRACE: 1 2 3   step a\n"+
		"TRACE: 1 2 3   step a, b\n"+
		"TRACE: 1 2 - github.com/rillig/pkglint/v23/trace.Test__fixed_argument_variants.func1(\"x\", \"y\")\n"+
		"TRACE: 1 - github.com/rillig/pkglint/v23/trace.Test__fixed_argument_variants.func1(\"x\")\n"+
		"TRACE: - github.com/rillig/pkglint/v23/trace.Test__fixed_argument_variants.func1()\n"; have != want {
		t.Errorf("want %#v, have %#v", want, have)
	}
}

func Test_Tracer_Call__Stringer_arg(t *testing.T) {
	var tracer Tracer

	output := captureTracingOutput(&tracer, func() {
		defer tracer.Call(stringer{}, &stringer{})()
	})

	if have, want := output, ""+
		"TRACE: + github.com/rillig/pkglint/v23/trace.Test_Tracer_Call__Stringer_arg.func1(It's a string, It's a string)\n"+
		"TRACE: - github.com/rillig/pkglint/v23/trace.Test_Tracer_Call__Stringer_arg.func1(It's a string, It's a string)\n"; have != want {
		t.Errorf("want %#v, have %#v", want, have)
	}
}

func Test_Tracer_Call__GoStringer_arg(t *testing.T) {
	var tracer Tracer

	output := captureTracingOutput(&tracer, func() {
		defer tracer.Call(goStringer{}, &goStringer{})()
	})

	if have, want := output, ""+
		"TRACE: + github.com/rillig/pkglint/v23/trace.Test_Tracer_Call__GoStringer_arg.func1(\"It's a string\", \"It's a string\")\n"+
		"TRACE: - github.com/rillig/pkglint/v23/trace.Test_Tracer_Call__GoStringer_arg.func1(\"It's a string\", \"It's a string\")\n"; have != want {
		t.Errorf("want %#v, have %#v", want, have)
	}
}

func Test_Tracer_traceCall__panic(t *testing.T) {
	var tracer Tracer

	var p any
	func() {
		defer func() {
			p = recover()
		}()
		tracer.traceCall()
	}()
	if p != "Internal pkglint error: calls to trace.Call must only occur in tracing mode" {
		t.Error(p)
	}
}

func Test_Tracer_Result__panic(t *testing.T) {
	var tracer Tracer

	var p any
	func() {
		defer func() {
			p = recover()
		}()
		tracer.Result("invalid argument")
	}()
	if p != "Result must be called with a pointer to the result, not \"invalid argument\"." {
		t.Error(p)
	}
}

func captureTracingOutput(tracer *Tracer, action func()) string {
	out := bytes.Buffer{}
	tracer.Out = &out
	tracer.Tracing = true

	action()

	tracer.Tracing = false
	tracer.Out = nil
	return out.String()
}

type stringer struct{}

func (stringer) String() string { return "It's a string" }

type goStringer struct{}

func (goStringer) GoString() string { return "\"It's a string\"" }

func Test__qa(t *testing.T) {
	ck := intqa.NewQAChecker(t.Errorf)
	ck.Configure("*", "*", "*", -intqa.EMissingTest)
	ck.Check()
}
