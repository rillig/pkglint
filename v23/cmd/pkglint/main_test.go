package main

import (
	"os"
	"regexp"
	"testing"
)

// This test goes into great lengths to bring the code coverage to 100%.
// Without this test, the main function would be a trivial one-liner.
// To make that one-liner testable, the call to os.Exit must be mockable.
func Test_main(t *testing.T) {
	tmpdir := t.TempDir()
	out, err := os.Create(tmpdir + "/out")
	if err != nil {
		t.Fatal(err)
	}

	prevStdout := os.Stdout
	prevArgs := os.Args
	prevExit := exit
	defer func() {
		os.Stdout = prevStdout
		os.Args = prevArgs
		exit = prevExit
	}()

	os.Stdout = out
	os.Args = []string{"pkglint", "--version"}
	exit = func(code int) {
		if code != 0 {
			t.Fatalf("unexpectedly exited with %#v", code)
		}
	}

	main()

	err = out.Close()
	if err != nil {
		t.Fatal(err)
	}

	output, err := os.ReadFile(out.Name())
	if err != nil {
		t.Fatal(err)
	}

	re := regexp.MustCompile(`^(@VERSION@|\d+(\.\d+)+(nb\d+)?)\n$`)
	if !re.Match(output) {
		t.Errorf("want %#v, have %#v", re, output)
	}
}
