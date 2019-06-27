# The pkglint tour

## The entry points

### Running pkglint

As is common in Go, each executable command is implemented in its own directory.
This directory is commonly called `cmd`.
 
> from [cmd/pkglint/main.go](cmd/pkglint/main.go#L10):

```go
func main() {
	exit(pkglint.Main())
}
```

From there on, everything interesting happens in the `netbsd.org/pkglint` package.
The below `Main` function already uses some implementation details (like `G.out` and `G.err`),
therefore it is currently not possible to write that code outside of this package.

Making all the pkglint code exportable is a good idea in general, but as of January 2019,
no one has asked to use any of the pkglint code as a library,
therefore the decision whether each element should be exported or not is not carved in stone yet.
If you want to use some of the code in your own pkgsrc programs,
[just ask](mailto:%72%69%6C%6C%69%67%40NetBSD.org).

> from [pkglint.go](pkglint.go#L161):

```go
func Main() int {
	G.Logger.out = NewSeparatorWriter(os.Stdout)
	G.Logger.err = NewSeparatorWriter(os.Stderr)
	trace.Out = os.Stdout
	exitCode := G.Main(os.Args...)
	if G.Opts.Profiling {
		G = unusablePkglint() // Free all memory.
		runtime.GC()          // For detecting possible memory leaks; see qa-pkglint.
	}
	return exitCode
}
```

> from [pkglint.go](pkglint.go#L173):

```go
// Main runs the main program with the given arguments.
// argv[0] is the program name.
//
// Note: during tests, calling this method disables tracing
// because the getopt parser resets all options before the actual parsing.
// One of these options is trace.Tracing, which is connected to --debug.
//
// It also discards the -Wall option that is used by default in other tests.
func (pkglint *Pkglint) Main(argv ...string) (exitCode int) {
	defer func() {
		if r := recover(); r != nil {
			if _, ok := r.(pkglintFatal); ok {
				exitCode = 1
			} else {
				panic(r)
			}
		}
	}()

	if exitcode := pkglint.ParseCommandLine(argv); exitcode != -1 {
		return exitcode
	}

	if pkglint.Opts.Profiling {
		defer pkglint.setUpProfiling()()
	}

	for _, arg := range pkglint.Opts.args {
		pkglint.Todo = append(pkglint.Todo, filepath.ToSlash(arg))
	}
	if len(pkglint.Todo) == 0 {
		pkglint.Todo = []string{"."}
	}

	firstDir := pkglint.Todo[0]
	if fileExists(firstDir) {
		firstDir = path.Dir(firstDir)
	}

	relTopdir := findPkgsrcTopdir(firstDir)
	if relTopdir == "" {
		// If the first argument to pkglint is not inside a pkgsrc tree,
		// pkglint doesn't know where to load the infrastructure files from,
		// and these are needed for virtually every single check.
		// Therefore, the only sensible thing to do is to quit immediately.
		dummyLine.Fatalf("%q must be inside a pkgsrc tree.", firstDir)
	}

	pkglint.Pkgsrc = NewPkgsrc(firstDir + "/" + relTopdir)
	pkglint.Wip = matches(pkglint.Pkgsrc.ToRel(firstDir), `^wip(/|$)`) // Same as in Pkglint.Check.
	pkglint.Pkgsrc.LoadInfrastructure()

	currentUser, err := user.Current()
	assertNil(err, "user.Current")
	// On Windows, this is `Computername\Username`.
	pkglint.Username = replaceAll(currentUser.Username, `^.*\\`, "")

	for len(pkglint.Todo) > 0 {
		item := pkglint.Todo[0]
		pkglint.Todo = pkglint.Todo[1:]
		pkglint.Check(item)
	}

	pkglint.Pkgsrc.checkToplevelUnusedLicenses()

	pkglint.Logger.ShowSummary()
	if pkglint.Logger.errors != 0 {
		return 1
	}
	return 0
}
```

When running pkglint, the `G` variable is set up first.
It contains the whole global state of pkglint.
(Except for some of the subpackages, which have to be initialized separately.)
All the interesting code is in the `Pkglint` type.
Having only a single global variable makes it easy to reset the global state during testing.

### Testing pkglint

Very similar code is used to set up the test and tear it down again:

> from [check_test.go](check_test.go#L59):

```go
func (s *Suite) SetUpTest(c *check.C) {
	t := Tester{c: c, testName: c.TestName()}
	s.Tester = &t

	G = NewPkglint()
	G.Testing = true
	G.Logger.out = NewSeparatorWriter(&t.stdout)
	G.Logger.err = NewSeparatorWriter(&t.stderr)
	trace.Out = &t.stdout

	// XXX: Maybe the tests can run a bit faster when they don't
	// create a temporary directory each.
	G.Pkgsrc = NewPkgsrc(t.File("."))

	t.c = c
	t.SetUpCommandLine("-Wall") // To catch duplicate warnings
	t.c = nil

	// To improve code coverage and ensure that trace.Result works
	// in all cases. The latter cannot be ensured at compile time.
	t.EnableSilentTracing()

	prevdir, err := os.Getwd()
	if err != nil {
		c.Fatalf("Cannot get current working directory: %s", err)
	}
	t.prevdir = prevdir
}
```

> from [check_test.go](check_test.go#L88):

```go
func (s *Suite) TearDownTest(c *check.C) {
	t := s.Tester
	t.c = nil // No longer usable; see https://github.com/go-check/check/issues/22

	if err := os.Chdir(t.prevdir); err != nil {
		t.Errorf("Cannot chdir back to previous dir: %s", err)
	}

	if t.seenSetupPkgsrc > 0 && !t.seenFinish && !t.seenMain {
		t.Errorf("After t.SetupPkgsrc(), either t.FinishSetUp() or t.Main() must be called.")
	}

	if out := t.Output(); out != "" {
		var msg strings.Builder
		msg.WriteString("\n")
		_, _ = fmt.Fprintf(&msg, "Unchecked output in %s; check with:\n", c.TestName())
		msg.WriteString("\n")
		msg.WriteString("t.CheckOutputLines(\n")
		lines := strings.Split(strings.TrimSpace(out), "\n")
		for i, line := range lines {
			_, _ = fmt.Fprintf(&msg, "\t%q%s\n", line, ifelseStr(i == len(lines)-1, ")", ","))
		}
		_, _ = fmt.Fprintf(&msg, "\n")
		_, _ = os.Stderr.WriteString(msg.String())
	}

	t.tmpdir = ""
	t.DisableTracing()

	G = unusablePkglint()
}
```

## First contact: checking a single DESCR file

To learn how pkglint works internally, it is a good idea to start with
a very simple example.
Since the `DESCR` files have a very simple structure (they only contain
text for human consumption), they are the ideal target.
Let's trace an invocation of the command `pkglint DESCR` down to where
the actual checks happen.

> from [cmd/pkglint/main.go](cmd/pkglint/main.go#L10):

```go
func main() {
	exit(pkglint.Main())
}
```

> from [pkglint.go](pkglint.go#L192):

```go
	if exitcode := pkglint.ParseCommandLine(argv); exitcode != -1 {
		return exitcode
	}
```

Since there are no command line options starting with a hyphen, we can
skip the command line parsing for this example.

The argument `DESCR` is saved in the `TODO` list.
The default use case for pkglint is to check the package from the
current working directory, therefore this is done if no arguments are given.

> from [pkglint.go](pkglint.go#L200):

```go
	for _, arg := range pkglint.Opts.args {
		pkglint.Todo = append(pkglint.Todo, filepath.ToSlash(arg))
	}
	if len(pkglint.Todo) == 0 {
		pkglint.Todo = []string{"."}
	}
```

Next, the files from the pkgsrc infrastructure are loaded to parse the
known variable names (like PREFIX, TOOLS_CREATE.*, the MASTER_SITEs).

The path to the pkgsrc root directory is determined from the first command line argument,
therefore the arguments had to be processed in the code above.

In this example run, the first (and only) argument is `DESCR`.
From there, the pkgsrc root is usually reachable via `../../`,
and this is what pkglint tries.

> from [pkglint.go](pkglint.go#L207):

```go
	firstDir := pkglint.Todo[0]
	if fileExists(firstDir) {
		firstDir = path.Dir(firstDir)
	}

	relTopdir := findPkgsrcTopdir(firstDir)
	if relTopdir == "" {
		// If the first argument to pkglint is not inside a pkgsrc tree,
		// pkglint doesn't know where to load the infrastructure files from,
		// and these are needed for virtually every single check.
		// Therefore, the only sensible thing to do is to quit immediately.
		dummyLine.Fatalf("%q must be inside a pkgsrc tree.", firstDir)
	}

	pkglint.Pkgsrc = NewPkgsrc(firstDir + "/" + relTopdir)
	pkglint.Wip = matches(pkglint.Pkgsrc.ToRel(firstDir), `^wip(/|$)`) // Same as in Pkglint.Check.
	pkglint.Pkgsrc.LoadInfrastructure()
```

Now the information from pkgsrc is loaded, and the main work can start.
The items from the TODO list are worked off and handed over to `Pkglint.Check`,
one after another. When pkglint is called with the `-r` option,
some entries may be added to the Todo list,
but that doesn't happen in this simple example run.

> from [pkglint.go](pkglint.go#L230):

```go
	for len(pkglint.Todo) > 0 {
		item := pkglint.Todo[0]
		pkglint.Todo = pkglint.Todo[1:]
		pkglint.Check(item)
	}
```

The main work is done in `Pkglint.Check`:

> from [pkglint.go](pkglint.go#L397):

```go
	if isReg {
		depth := strings.Count(pkgsrcRel, "/")
		pkglint.checkExecutable(dirent, mode)
		pkglint.checkReg(dirent, basename, depth)
		return
	}
```

Since `DESCR` is a regular file, the next function to call is `checkReg`.
For directories, the next function would depend on the depth from the
pkgsrc root directory.

> from [pkglint.go](pkglint.go#L606):

```go
func (pkglint *Pkglint) checkReg(filename, basename string, depth int) {

	if depth == 2 && !pkglint.Wip {
		if contains(basename, "README") || contains(basename, "TODO") {
			NewLineWhole(filename).Errorf("Packages in main pkgsrc must not have a %s file.", basename)
			// TODO: Add a convincing explanation.
			return
		}
	}

	switch {
	case hasSuffix(basename, "~"),
		hasSuffix(basename, ".orig"),
		hasSuffix(basename, ".rej"),
		contains(basename, "README") && depth == 2,
		contains(basename, "TODO") && depth == 2:
		if pkglint.Opts.Import {
			NewLineWhole(filename).Errorf("Must be cleaned up before committing the package.")
		}
		return
	}

	switch {
	case basename == "ALTERNATIVES":
		CheckFileAlternatives(filename)

	case basename == "buildlink3.mk":
		if mklines := LoadMk(filename, NotEmpty|LogErrors); mklines != nil {
			CheckLinesBuildlink3Mk(mklines)
		}

	case hasPrefix(basename, "DESCR"):
		if lines := Load(filename, NotEmpty|LogErrors); lines != nil {
			CheckLinesDescr(lines)
		}

	case basename == "distinfo":
		if lines := Load(filename, NotEmpty|LogErrors); lines != nil {
			CheckLinesDistinfo(G.Pkg, lines)
		}

	case basename == "DEINSTALL" || basename == "INSTALL":
		CheckFileOther(filename)

	case hasPrefix(basename, "MESSAGE"):
		if lines := Load(filename, NotEmpty|LogErrors); lines != nil {
			CheckLinesMessage(lines)
		}

	case basename == "options.mk":
		if mklines := LoadMk(filename, NotEmpty|LogErrors); mklines != nil {
			CheckLinesOptionsMk(mklines)
		}

	case matches(basename, `^patch-[-\w.~+]*\w$`):
		if lines := Load(filename, NotEmpty|LogErrors); lines != nil {
			CheckLinesPatch(lines)
		}

	case matches(filename, `(?:^|/)patches/manual[^/]*$`):
		if trace.Tracing {
			trace.Step1("Unchecked file %q.", filename)
		}

	case matches(filename, `(?:^|/)patches/[^/]*$`):
		NewLineWhole(filename).Warnf("Patch files should be named \"patch-\", followed by letters, '-', '_', '.', and digits only.")

	case (hasPrefix(basename, "Makefile") || hasSuffix(basename, ".mk")) &&
		!pathContainsDir(filename, "files"):
		CheckFileMk(filename)

	case hasPrefix(basename, "PLIST"):
		if lines := Load(filename, NotEmpty|LogErrors); lines != nil {
			CheckLinesPlist(G.Pkg, lines)
		}

	case hasPrefix(basename, "CHANGES-"):
		// This only checks the file but doesn't register the changes globally.
		_ = pkglint.Pkgsrc.loadDocChangesFromFile(filename)

	case matches(filename, `(?:^|/)files/[^/]*$`):
		// Skip files directly in the files/ directory, but not those further down.

	case basename == "spec":
		if !hasPrefix(pkglint.Pkgsrc.ToRel(filename), "regress/") {
			NewLineWhole(filename).Warnf("Only packages in regress/ may have spec files.")
		}

	case pkglint.matchesLicenseFile(basename):
		break

	default:
		NewLineWhole(filename).Warnf("Unexpected file found.")
	}
}
```

> from [pkglint.go](pkglint.go#L632):

```go
	case basename == "buildlink3.mk":
		if mklines := LoadMk(filename, NotEmpty|LogErrors); mklines != nil {
			CheckLinesBuildlink3Mk(mklines)
		}

	case hasPrefix(basename, "DESCR"):
		if lines := Load(filename, NotEmpty|LogErrors); lines != nil {
			CheckLinesDescr(lines)
		}

	case basename == "distinfo":
		if lines := Load(filename, NotEmpty|LogErrors); lines != nil {
			CheckLinesDistinfo(G.Pkg, lines)
		}
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

> from [pkglint.go](pkglint.go#L496):

```go
func CheckLinesDescr(lines *Lines) {
	if trace.Tracing {
		defer trace.Call1(lines.Filename)()
	}

	for _, line := range lines.Lines {
		ck := LineChecker{line}
		ck.CheckLength(80)
		ck.CheckTrailingWhitespace()
		ck.CheckValidCharacters()

		if contains(line.Text, "${") {
			for _, token := range NewMkParser(nil, line.Text, false).MkTokens() {
				if token.Varuse != nil && G.Pkgsrc.VariableType(nil, token.Varuse.varname) != nil {
					line.Notef("Variables are not expanded in the DESCR file.")
				}
			}
		}
	}

	CheckLinesTrailingEmptyLines(lines)

	if maxLines := 24; lines.Len() > maxLines {
		line := lines.Lines[maxLines]

		line.Warnf("File too long (should be no more than %d lines).", maxLines)
		line.Explain(
			"The DESCR file should fit on a traditional terminal of 80x25 characters.",
			"It is also intended to give a _brief_ summary about the package's contents.")
	}

	SaveAutofixChanges(lines)
}
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

> from [linechecker.go](linechecker.go#L42):

```go
func (ck LineChecker) CheckTrailingWhitespace() {

	// XXX: Markdown files may need trailing whitespace. If there should ever
	// be Markdown files in pkgsrc, this code has to be adjusted.

	if strings.HasSuffix(ck.line.Text, " ") || strings.HasSuffix(ck.line.Text, "\t") {
		fix := ck.line.Autofix()
		fix.Notef("Trailing whitespace.")
		fix.Explain(
			"When a line ends with some whitespace, that space is in most cases",
			"irrelevant and can be removed.")
		fix.ReplaceRegex(`[ \t\r]+\n$`, "\n", 1)
		fix.Apply()
	}
}
```

This code is a typical example for using the autofix feature.
Some more possibilities are described at the `Autofix` type itself
and at its typical call site `Line.Autofix()`:

> from [autofix.go](autofix.go#L11):

```go
// Autofix handles all modifications to a single line,
// describes them in a human-readable form and formats the output.
// The modifications are kept in memory only,
// until they are written to disk by SaveAutofixChanges.
type Autofix struct {
	line        *Line
	linesBefore []string // Newly inserted lines, including \n
	linesAfter  []string // Newly inserted lines, including \n
	// Whether an actual fix has been applied (or, without --show-autofix,
	// whether a fix is applicable)
	modified bool

	autofixShortTerm
}
```

> from [line.go](line.go#L196):

```go
// Autofix returns the autofix instance belonging to the line.
//
// Usage:
//
//  fix := line.Autofix()
//
//  fix.Errorf("Must not be ...")
//  fix.Warnf("Should not be ...")
//  fix.Notef("It is also possible ...")
//
//  fix.Explain(
//      "Explanation ...",
//      "... end of explanation.")
//
//  fix.Replace("from", "to")
//  fix.ReplaceAfter("prefix", "from", "to")
//  fix.ReplaceRegex(`[\t ]+`, "space", -1)
//  fix.InsertBefore("new line")
//  fix.InsertAfter("new line")
//  fix.Delete()
//  fix.Custom(func(showAutofix, autofix bool) {})
//
//  fix.Apply()
func (line *Line) Autofix() *Autofix {
	if line.autofix == nil {
		line.autofix = NewAutofix(line)
	}
	return line.autofix
}
```

The journey ends here, and it hasn't been that difficult.
If that was too easy, have a look at the complex cases here:

> from [mkline.go](mkline.go#L886):

```go
// VariableNeedsQuoting determines whether the given variable needs the :Q operator
// in the given context.
//
// This decision depends on many factors, such as whether the type of the context is
// a list of things, whether the variable is a list, whether it can contain only
// safe characters, and so on.
func (mkline *MkLine) VariableNeedsQuoting(mklines *MkLines, varuse *MkVarUse, vartype *Vartype, vuc *VarUseContext) (needsQuoting YesNoUnknown) {
	if trace.Tracing {
		defer trace.Call(varuse, vartype, vuc, trace.Result(&needsQuoting))()
	}

	// TODO: Systematically test this function, each and every case, from top to bottom.
	// TODO: Re-check the order of all these if clauses whether it really makes sense.

	vucVartype := vuc.vartype
	if vartype == nil || vucVartype == nil || vartype.basicType == BtUnknown {
		return unknown
	}

	if !vartype.basicType.NeedsQ() {
		if !vartype.List() {
			if vartype.Guessed() {
				return unknown
			}
			return no
		}
		if !vuc.IsWordPart {
			return no
		}
	}

	// A shell word may appear as part of a shell word, for example COMPILER_RPATH_FLAG.
	if vuc.IsWordPart && vuc.quoting == VucQuotPlain {
		if !vartype.List() && vartype.basicType == BtShellWord {
			return no
		}
	}

	// Determine whether the context expects a list of shell words or not.
	wantList := vucVartype.MayBeAppendedTo()
	haveList := vartype.MayBeAppendedTo()
	if trace.Tracing {
		trace.Stepf("wantList=%v, haveList=%v", wantList, haveList)
	}

	// Both of these can be correct, depending on the situation:
	// 1. echo ${PERL5:Q}
	// 2. xargs ${PERL5}
	if !vuc.IsWordPart && wantList && haveList {
		return unknown
	}

	// Pkglint assumes that the tool definitions don't include very
	// special characters, so they can safely be used inside any quotes.
	if tool := G.ToolByVarname(mklines, varuse.varname); tool != nil {
		switch vuc.quoting {
		case VucQuotPlain:
			if !vuc.IsWordPart {
				return no
			}
			// XXX: Should there be a return here? It looks as if it could have been forgotten.
		case VucQuotBackt:
			return no
		case VucQuotDquot, VucQuotSquot:
			return unknown
		}
	}

	// Variables that appear as parts of shell words generally need to be quoted.
	//
	// An exception is in the case of backticks, because the whole backticks expression
	// is parsed as a single shell word by pkglint. (XXX: This comment may be outdated.)
	if vuc.IsWordPart && vucVartype.IsShell() && vuc.quoting != VucQuotBackt {
		return yes
	}

	// SUBST_MESSAGE.perl= Replacing in ${REPLACE_PERL}
	if vucVartype.basicType == BtMessage {
		return no
	}

	if wantList != haveList {
		if vucVartype.basicType == BtFetchURL && vartype.basicType == BtHomepage {
			return no
		}
		if vucVartype.basicType == BtHomepage && vartype.basicType == BtFetchURL {
			return no // Just for HOMEPAGE=${MASTER_SITE_*:=subdir/}.
		}

		// .for dir in ${PATH:C,:, ,g}
		for _, modifier := range varuse.modifiers {
			if modifier.ChangesWords() {
				return unknown
			}
		}

		return yes
	}

	// Bad: LDADD+= -l${LIBS}
	// Good: LDADD+= ${LIBS:S,^,-l,}
	if wantList {
		return yes
	}

	if trace.Tracing {
		trace.Step1("Don't know whether :Q is needed for %q", varuse.varname)
	}
	return unknown
}
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

> from [line.go](line.go#L62):

```go
// Line represents a line of text from a file.
type Line struct {
	// TODO: Consider storing pointers to the Filename and Basename instead of strings to save memory.
	//  But first find out where and why pkglint needs so much memory (200 MB for a full recursive run over pkgsrc + wip).
	Location
	Basename string // the last component of the Filename

	// the text of the line, without the trailing newline character;
	// in Makefiles, also contains the text from the continuation lines,
	// joined by single spaces
	Text string

	raw     []*RawLine // contains the original text including trailing newline
	autofix *Autofix   // any changes that pkglint would like to apply to the line
	Once

	// XXX: Filename and Basename could be replaced with a pointer to a Lines object.
}
```

### MkLine

Most of the pkgsrc infrastructure is written in Makefiles. 
In these, there may be line continuations  (the ones ending in backslash).
Plus, they may contain Make variables of the form `${VARNAME}` or `${VARNAME:Modifiers}`,
and these are handled specially.

> from [mkline.go](mkline.go#L11):

```go
// MkLine is a line from a Makefile fragment.
// There are several types of lines.
// The most common types in pkgsrc are variable assignments,
// shell commands and directives like .if and .for.
type MkLine struct {
	*Line

	// One of the following mkLine* types.
	//
	// For the larger of these types, a pointer is used instead of a direct
	// struct because of https://github.com/golang/go/issues/28045.
	data interface{}
}
```

### ShellLine

The instructions for building and installing packages are written in shell commands,
which are embedded in Makefile fragments.
The `ShellLineChecker` type provides methods for checking shell commands and their individual parts.

> from [shell.go](shell.go#L13):

```go
// ShellLineChecker is either a line from a Makefile starting with a tab,
// thereby containing shell commands to be executed.
//
// Or it is a variable assignment line from a Makefile with a left-hand
// side variable that is of some shell-like type; see Vartype.IsShell.
type ShellLineChecker struct {
	MkLines *MkLines
	mkline  *MkLine

	// checkVarUse is set to false when checking a single shell word
	// in order to skip duplicate warnings in variable assignments.
	checkVarUse bool
}
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

> from [check_test.go](check_test.go#L124):

```go
// Tester provides utility methods for testing pkglint.
// It is separated from the Suite since the latter contains
// all the test methods, which makes it difficult to find
// a method by auto-completion.
type Tester struct {
	c        *check.C // Only usable during the test method itself
	testName string

	stdout  bytes.Buffer
	stderr  bytes.Buffer
	tmpdir  string
	prevdir string // The current working directory before the test started
	relCwd  string // See Tester.Chdir

	seenSetupPkgsrc int
	seenFinish      bool
	seenMain        bool
}
```

The `s` variable is not used in tests.
The only purpose of its type `Suite` is to group the tests so they are all run together.

The `c` variable comes from [gocheck](https://godoc.org/gopkg.in/check.v1),
which is the underlying testing framework.
Most pkglint tests don't need this variable.
Low-level tests call `c.Check` to compare their results to the expected values.

> from [util_test.go](util_test.go#L82):

```go
func (s *Suite) Test_tabWidth(c *check.C) {
	c.Check(tabWidth("12345"), equals, 5)
	c.Check(tabWidth("\t"), equals, 8)
	c.Check(tabWidth("123\t"), equals, 8)
	c.Check(tabWidth("1234567\t"), equals, 8)
	c.Check(tabWidth("12345678\t"), equals, 16)
}
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

> from [pkglint_test.go](pkglint_test.go#L119):

```go
// Demonstrates which infrastructure files are necessary to actually run
// pkglint in a realistic scenario.
//
// Especially covers Pkglint.ShowSummary and Pkglint.checkReg.
func (s *Suite) Test_Pkglint_Main__complete_package(c *check.C) {
	t := s.Init(c)

	// Since the general infrastructure setup is useful for several tests,
	// it is available as a separate method.
	//
	// In this test, several of the infrastructure files are later
	// overwritten with more realistic and interesting content.
	// This is typical of the pkglint tests.
	t.SetUpPkgsrc()

	t.CreateFileLines("doc/CHANGES-2018",
		CvsID,
		"",
		"Changes to the packages collection and infrastructure in 2018:",
		"",
		"\tUpdated sysutils/checkperms to 1.10 [rillig 2018-01-05]")

	// See Pkgsrc.loadSuggestedUpdates.
	t.CreateFileLines("doc/TODO",
		CvsID,
		"",
		"Suggested package updates",
		"",
		"\to checkperms-1.13 [supports more file formats]")

	// The MASTER_SITES in the package Makefile are searched here.
	// See Pkgsrc.loadMasterSites.
	t.CreateFileLines("mk/fetch/sites.mk",
		MkCvsID,
		"",
		"MASTER_SITE_GITHUB+=\thttps://github.com/")

	// After setting up the pkgsrc infrastructure, the files for
	// a complete pkgsrc package are created individually.
	//
	// In this test each file is created manually for demonstration purposes.
	// Other tests typically call t.SetUpPackage, which does most of the work
	// shown here while allowing to adjust the package Makefile a little bit.

	// The existence of this file makes the category "sysutils" valid,
	// so that it can be used in CATEGORIES in the package Makefile.
	// The category "tools" on the other hand is not valid.
	t.CreateFileLines("sysutils/Makefile",
		MkCvsID)

	// The package Makefile in this test is quite simple, containing just the
	// standard variable definitions. The data for checking the variable
	// values is partly defined in the pkgsrc infrastructure files
	// (as defined in the previous lines), and partly in the pkglint
	// code directly. Many details can be found in vartypecheck.go.
	t.CreateFileLines("sysutils/checkperms/Makefile",
		MkCvsID,
		"",
		"DISTNAME=\tcheckperms-1.11",
		"CATEGORIES=\tsysutils tools",
		"MASTER_SITES=\t${MASTER_SITE_GITHUB:=rillig/}",
		"",
		"MAINTAINER=\tpkgsrc-users@NetBSD.org",
		"HOMEPAGE=\thttps://github.com/rillig/checkperms/",
		"COMMENT=\tCheck file permissions",
		"LICENSE=\t2-clause-bsd",
		"",
		".include \"../../mk/bsd.pkg.mk\"")

	t.CreateFileLines("sysutils/checkperms/MESSAGE",
		"===========================================================================",
		CvsID,
		"",
		"After installation, this package has to be configured in a special way.",
		"",
		"===========================================================================")

	t.CreateFileLines("sysutils/checkperms/PLIST",
		PlistCvsID,
		"bin/checkperms",
		"man/man1/checkperms.1")

	t.CreateFileLines("sysutils/checkperms/README",
		"When updating this package, test the pkgsrc bootstrap.")

	t.CreateFileLines("sysutils/checkperms/TODO",
		"Make the package work on MS-DOS")

	t.CreateFileLines("sysutils/checkperms/patches/patch-checkperms.c",
		CvsID,
		"",
		"A simple patch demonstrating that pkglint checks for missing",
		"removed lines. The hunk headers says that one line is to be",
		"removed, but in fact, there is no deletion line below it.",
		"",
		"--- a/checkperms.c",
		"+++ b/checkperms.c",
		"@@ -1,1 +1,3 @@", // at line 1, delete 1 line; at line 1, add 3 lines
		"+// Header 1",
		"+// Header 2",
		"+// Header 3")
	t.CreateFileLines("sysutils/checkperms/distinfo",
		CvsID,
		"",
		"SHA1 (checkperms-1.12.tar.gz) = 34c084b4d06bcd7a8bba922ff57677e651eeced5",
		"RMD160 (checkperms-1.12.tar.gz) = cd95029aa930b6201e9580b3ab7e36dd30b8f925",
		"SHA512 (checkperms-1.12.tar.gz) = "+
			"43e37b5963c63fdf716acdb470928d7e21a7bdfddd6c85cf626a11acc7f45fa5"+
			"2a53d4bcd83d543150328fe8cec5587987d2d9a7c5f0aaeb02ac1127ab41f8ae",
		"Size (checkperms-1.12.tar.gz) = 6621 bytes",
		"SHA1 (patch-checkperms.c) = asdfasdf") // Invalid SHA-1 checksum

	t.Main("-Wall", "-Call", t.File("sysutils/checkperms"))

	t.CheckOutputLines(
		"NOTE: ~/sysutils/checkperms/Makefile:3: "+
			"Package version \"1.11\" is greater than the latest \"1.10\" "+
			"from ../../doc/CHANGES-2018:5.",
		"WARN: ~/sysutils/checkperms/Makefile:3: "+
			"This package should be updated to 1.13 ([supports more file formats]).",
		"ERROR: ~/sysutils/checkperms/Makefile:4: Invalid category \"tools\".",
		"ERROR: ~/sysutils/checkperms/README: Packages in main pkgsrc must not have a README file.",
		"ERROR: ~/sysutils/checkperms/TODO: Packages in main pkgsrc must not have a TODO file.",
		"ERROR: ~/sysutils/checkperms/distinfo:7: SHA1 hash of patches/patch-checkperms.c differs "+
			"(distinfo has asdfasdf, patch file has e775969de639ec703866c0336c4c8e0fdd96309c).",
		"WARN: ~/sysutils/checkperms/patches/patch-checkperms.c:12: Premature end of patch hunk "+
			"(expected 1 lines to be deleted and 0 lines to be added).",
		"4 errors and 2 warnings found.",
		"(Run \"pkglint -e\" to show explanations.)",
		"(Run \"pkglint -fs\" to show what can be fixed automatically.)",
		"(Run \"pkglint -F\" to automatically fix some issues.)")
}
```
