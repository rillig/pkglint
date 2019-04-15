# The pkglint tour

## The entry points

### Running pkglint

As is common in Go, each executable command is implemented in its own directory.
This directory is commonly called `cmd`.
 
```codewalk
file     cmd/pkglint/main.go
go:func  main
```

From there on, everything interesting happens in the `netbsd.org/pkglint` package.
The below `Main` function already uses some implementation details (like `G.out` and `G.err`),
therefore it is currently not possible to write that code outside of this package.

Making all the pkglint code exportable is a good idea in general, but as of January 2019,
no one has asked to use any of the pkglint code as a library,
therefore the decision whether each element should be exported or not is not carved in stone yet.
If you want to use some of the code in your own pkgsrc programs,
[just ask](mailto:%72%69%6C%6C%69%67%40NetBSD.org).

```codewalk
file     pkglint.go
go:func  Main
```

```codewalk
file     pkglint.go
go:func  Pkglint.Main
```

When running pkglint, the `G` variable is set up first.
It contains the whole global state of pkglint.
(Except for some of the subpackages, which have to be initialized separately.)
All the interesting code is in the `Pkglint` type.
Having only a single global variable makes it easy to reset the global state during testing.

### Testing pkglint

Very similar code is used to set up the test and tear it down again:

```codewalk
file     check_test.go
go:func  Suite.SetUpTest
```

```codewalk
file     check_test.go
go:func  Suite.TearDownTest
```

## First contact: checking a single DESCR file

To learn how pkglint works internally, it is a good idea to start with
a very simple example.
Since the `DESCR` files have a very simple structure (they only contain
text for human consumption), they are the ideal target.
Let's trace an invocation of the command `pkglint DESCR` down to where
the actual checks happen.

```codewalk
file     cmd/pkglint/main.go
go:func  main
```

```codewalk
file   pkglint.go
start  ^[\t]if exitcode :=
end    ^\t\}$
```

Since there are no command line options starting with a hyphen, we can
skip the command line parsing for this example.

The argument `DESCR` is saved in the `TODO` list.
The default use case for pkglint is to check the package from the
current working directory, therefore this is done if no arguments are given.

```codewalk
file   pkglint.go
start  ^[\t]for _, arg
end    ^$
endUp 1
```

Next, the files from the pkgsrc infrastructure are loaded to parse the
known variable names (like PREFIX, TOOLS_CREATE.*, the MASTER_SITEs).

The path to the pkgsrc root directory is determined from the first command line argument,
therefore the arguments had to be processed in the code above.

In this example run, the first (and only) argument is `DESCR`.
From there, the pkgsrc root is usually reachable via `../../`,
and this is what pkglint tries.

```codewalk
file   pkglint.go
start  ^[\t]firstDir :=
end    LoadInfrastructure
```

Now the information from pkgsrc is loaded, and the main work can start.
The items from the TODO list are worked off and handed over to `Pkglint.Check`,
one after another. When pkglint is called with the `-r` option,
some entries may be added to the Todo list,
but that doesn't happen in this simple example run.

```codewalk
file   pkglint.go
start  ^[\t]for len\(pkglint\.Todo\)
end    ^\t}
```

The main work is done in `Pkglint.Check`:

```codewalk
file     pkglint.go
start    ^\tif isReg
end      ^\t\}
```

Since `DESCR` is a regular file, the next function to call is `checkReg`.
For directories, the next function would depend on the depth from the
pkgsrc root directory.

```codewalk
file     pkglint.go
go:func  Pkglint.checkReg
```

```codewalk
file   pkglint.go
start  basename == "buildlink3.mk"
end    case basename == "DEINSTALL"
endUp  2
```

When compared to the code blocks around this one, it looks strange that
this one uses `hasPrefix` and the others use a direct string comparison.
But indeed, there are a few packages that actually have `DESCR.common`
files. So everything's fine here.

At this point, the file is loaded and converted to lines.
For DESCR files, this is very simple, so there's no need to dive into that.

The actual checks usually work on `Line` objects instead of files
because the lines offer nice methods for logging the diagnostics
and for automatically fixing the text (in pkglint's `--autofix` mode).

```codewalk
file     pkglint.go
go:func  CheckLinesDescr
```

Now we are where the actual action takes place.
The code looks straight-forward here.
First, each line is checked on its own,
and the final check is for too long files.
Pkglint takes great care to output all diagnostics in a logical order,
that is file by file, and top to bottom within each file.
Therefore the checks for individual lines happen before the other check.

The call to `SaveAutofixChanges` at the end looks a bit strange
since none of the visible checks fixes anything.
The autofix feature must be hidden in one of the line checks,
and indeed, the code for `CheckTrailingWhitespace` says:

```codewalk
file     linechecker.go
go:func  LineChecker.CheckTrailingWhitespace
```

This code is a typical example for using the autofix feature.
Some more possibilities are described at the `Autofix` type itself
and at its typical call site `Line.Autofix()`:

```codewalk
file autofix.go
go:type Autofix
```

```codewalk
file line.go
go:func LineImpl.Autofix
```

The journey ends here, and it hasn't been that difficult.
If that was too easy, have a look at the complex cases here:

```codewalk
file mkline.go
go:func MkLineImpl.VariableNeedsQuoting
```

## Basic ingredients

Pkglint checks packages, and a package consists of several different files.
All pkgsrc files are text files, which are organized in lines.

Most pkglint diagnostics refer to a specific line,
therefore the `Line` type is responsible for producing the diagnostics.

### Line

Most checks in pkgsrc only need to look at a single line.
Lines that are independent of the file type are implemented in the `Line` type.
This type contains the methods `Errorf`, `Warnf` and `Notef` to produce diagnostics
of the following form:

```text
WARN: Makefile:3: COMMENT should not start with "A" or "An".
```

The definition for the `Line` type is:

```codewalk
file     line.go
go:type  Line
```

```codewalk
file     line.go
go:type  LineImpl
```

### MkLine

Most of the pkgsrc infrastructure is written in Makefiles. 
In these, there may be line continuations  (the ones ending in backslash).
Plus, they may contain Make variables of the form `${VARNAME}` or `${VARNAME:Modifiers}`,
and these are handled specially.

```codewalk
file     mkline.go
go:type  MkLine
```

```codewalk
file    mkline.go
go:type MkLineImpl
```

### ShellLine

The instructions for building and installing packages are written in shell commands,
which are embedded in Makefile fragments.
The `ShellLineChecker` type provides methods for checking shell commands and their individual parts.

```codewalk
file     shell.go
go:type  ShellLineChecker
```

## Testing pkglint

### Standard shape of a test

```go
func (s *Suite) Test_Type_Method__description(c *check.C) {
	t := s.Init(c)       // Every test needs this.

	t.SetUp…(…)          // Set up the testing environment.

	lines := t.New…(…)   // Set up the test data.

	CodeToBeTested()     // The code to be tested.

	t.Check…(…)          // Check the result (typically diagnostics).
}
```

The `t` variable is the center of most tests.
It is of type `Tester` and provides a high-level interface
for setting up tests and checking the results.

```codewalk
file     check_test.go
go:type  Tester
```

The `s` variable is not used in tests.
The only purpose of its type `Suite` is to group the tests so they are all run together.

The `c` variable comes from [gocheck](https://godoc.org/gopkg.in/check.v1),
which is the underlying testing framework.
Most pkglint tests don't need this variable.
Low-level tests call `c.Check` to compare their results to the expected values.

```codewalk
file     util_test.go
go:func  Suite.Test_tabWidth
```

### Logging detailed information during tests

When testing complicated code, it sometimes helps to have a detailed trace
of the code that is run. This is done via these two methods:

```go
t.EnableTracing()
t.DisableTracing()
```

### Setting up a realistic pkgsrc environment

To see how to setup complicated tests, have a look at the following test,
which sets up a realistic environment to run the tests in.

```codewalk
file     pkglint_test.go
go:func  Suite.Test_Pkglint_Main__complete_package
```

### Typical warnings during a test

When running a newly written pkglint test, it may output more warnings than 
necessary or interesting for the current test. Here are the most frequent
warnings and how to repair them properly:

#### Unknown shell command %q

* Load the standard variables using `t.SetUpVartypes()`
* Define the corresponding tool using `t.SetUpTool("tool", "TOOL", AtRunTime)`

#### %s is used but not defined

* Load the standard variables using `t.SetUpVartypes()`

#### %s is defined but not used

* Load the standard variables using `t.SetUpVartypes()`

### Traps and pitfalls during a test

If a file is not checked although it should be, check whether you have created
the lines using `t.NewLines` instead of `t.CreateFileLines`. The former creates
the lines only in memory, and the result of that method must be used, otherwise
the call doesn't make sense.
