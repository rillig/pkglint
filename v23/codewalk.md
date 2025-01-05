# The pkglint tour

## The entry points

### Running pkglint

As is common in Go, each executable command is implemented in its own directory.
This directory is commonly called `cmd`.

> from [cmd/pkglint/main.go](cmd/pkglint/main.go#L10):

```go
func main() {
	exit(pkglint.G.Main(os.Stdout, os.Stderr, os.Args))
}
```

From there on, everything interesting happens in the `github.com/rillig/pkglint/v23` package.
The below `Main` function already uses some implementation details (like `G.Logger.out` and `G.Logger.err`),
therefore it is currently not possible to write that code outside of this package.

Making all the pkglint code exportable is a good idea in general, but as of June 2019,
no one has asked to use any of the pkglint code as a library,
therefore the decision whether each element should be exported or not is not carved in stone yet.
If you want to use some of the code in your own pkgsrc programs,
[just ask](mailto:%72%69%6C%6C%69%67%40NetBSD.org?subject=using%20pkglint%20as%20a%20library).

> from [pkglint.go](pkglint.go#L98):

```go
// Main runs the main program with the given arguments.
// args[0] is the program name.
//
// Note: during tests, calling this method disables tracing
// because the getopt parser resets all options before the actual parsing.
// One of these options is trace.Tracing, which is connected to --debug.
//
// It also discards the -Wall option that is used by default in other tests.
func (p *Pkglint) Main(stdout io.Writer, stderr io.Writer, args []string) (exitCode int) {
```

When running pkglint, the `G` variable is set up first.
It contains the whole global state of pkglint:

> from [pkglint.go](pkglint.go#L91):

```go
// G is the abbreviation for "global state";
// this and the tracer are the only global variables in this Go package.
var (
	G     = NewPkglint(os.Stdout, os.Stderr)
	trace tracePkg.Tracer
)
```

All the interesting code is in the `Pkglint` type.
Having only two global variables makes it easy to reset the global state during testing.

> from [pkglint.go](pkglint.go#L98):

```go
// Main runs the main program with the given arguments.
// args[0] is the program name.
//
// Note: during tests, calling this method disables tracing
// because the getopt parser resets all options before the actual parsing.
// One of these options is trace.Tracing, which is connected to --debug.
//
// It also discards the -Wall option that is used by default in other tests.
func (p *Pkglint) Main(stdout io.Writer, stderr io.Writer, args []string) (exitCode int) {
	G.Logger.out = NewSeparatorWriter(stdout)
	G.Logger.err = NewSeparatorWriter(stderr)
	trace.Out = stdout

	defer func() {
		if r := recover(); r != nil {
			_ = r.(pkglintFatal)
			exitCode = 1
		}
	}()

	if exitcode := p.ParseCommandLine(args); exitcode != -1 {
		return exitcode
	}

	if p.Profiling {
		defer p.setUpProfiling()()
	}

	p.prepareMainLoop()

	for !p.Todo.IsEmpty() {
		p.Check(p.Todo.Pop())
	}

	p.Pkgsrc.checkToplevelUnusedLicenses()

	p.Logger.ShowSummary(args)
	if p.WarnError && p.Logger.warnings != 0 {
		return 1
	}
	if p.Logger.errors != 0 {
		return 1
	}
	return 0
}
```

### Testing pkglint

The code for setting up the tests looks similar to the main code:

> from [check_test.go](check_test.go#L56):

```go
func (s *Suite) SetUpTest(c *check.C) {
	t := Tester{c: c, testName: c.TestName()}
	s.Tester = &t

	G = NewPkglint(&t.stdout, &t.stderr)
	G.Testing = true
	trace.Out = &t.stdout

	G.Pkgsrc = NewPkgsrc(t.File("."))
	G.Project = G.Pkgsrc

	t.c = c
	t.SetUpCommandLine("-Wall")    // To catch duplicate warnings
	G.Todo.Pop()                   // The "." was inserted by default.
	t.seenSetUpCommandLine = false // This default call doesn't count.

	// To improve code coverage and ensure that trace.Result works
	// in all cases. The latter cannot be ensured at compile time.
	t.EnableSilentTracing()

	prevdir, err := os.Getwd()
	assertNil(err, "Cannot get current working directory: %s", err)
	t.prevdir = NewCurrPathString(prevdir)

	// No longer usable; see https://github.com/go-check/check/issues/22
	t.c = nil
}
```

## First contact: checking a single DESCR file

To learn how pkglint works internally, it is a good idea to start with
a small example.

Since the `DESCR` files have a very simple structure (they only contain
text for human consumption), they are the ideal target.
Let's trace an invocation of the command `pkglint DESCR` down to where
the actual checks happen.

> from [cmd/pkglint/main.go](cmd/pkglint/main.go#L10):

```go
func main() {
	exit(pkglint.G.Main(os.Stdout, os.Stderr, os.Args))
}
```

> from [pkglint.go](pkglint.go#L98):

```go
// Main runs the main program with the given arguments.
// args[0] is the program name.
//
// Note: during tests, calling this method disables tracing
// because the getopt parser resets all options before the actual parsing.
// One of these options is trace.Tracing, which is connected to --debug.
//
// It also discards the -Wall option that is used by default in other tests.
func (p *Pkglint) Main(stdout io.Writer, stderr io.Writer, args []string) (exitCode int) {
```

> from [pkglint.go](pkglint.go#L118):

```go
	if exitcode := p.ParseCommandLine(args); exitcode != -1 {
		return exitcode
	}
```

In this example, there are no command line options starting with a hyphen.
Therefore, the main part of `ParseCommandLine` can be skipped.
The one remaining command line argument is `DESCR`,
and that is saved in `pkglint.Todo`, which contains all items that still need to be checked.
The default use case for pkglint is to check the package from the
current working directory, therefore this is done if no arguments are given.

> from [pkglint.go](pkglint.go#L280):

```go
	for _, arg := range remainingArgs {
		p.Todo.Push(NewCurrPathSlash(arg))
	}
	if p.Todo.IsEmpty() {
		p.Todo.Push(".")
	}
```

Next, the files from the pkgsrc infrastructure are loaded to parse the
known variable names (like PREFIX, TOOLS_CREATE.*, the MASTER_SITEs).

The path to the pkgsrc root directory is determined from the first command line argument,
therefore the arguments had to be processed before loading the pkgsrc infrastructure.

In this example run, the first and only argument is `DESCR`.
From there, the pkgsrc root is usually reachable via `../../`,
and this is what pkglint tries.

> from [pkglint.go](pkglint.go#L196):

```go
	firstDir := p.Todo.Front()
	isFile := firstDir.IsFile()
	if isFile {
		firstDir = firstDir.Dir()
	}

	relTopdir := p.findPkgsrcTopdir(firstDir)
	if relTopdir.IsEmpty() {
		// If the first argument to pkglint is not inside a pkgsrc tree,
		// pkglint doesn't know where to load the infrastructure files from.
		if isFile {
			// Allow this mode nevertheless, for checking the basic syntax
			// and for formatting individual makefiles outside pkgsrc.
		} else {
			G.Logger.TechFatalf(firstDir, "Must be inside a pkgsrc tree.")
		}
		p.Project = NewNetBSDProject()
	} else {
		p.Pkgsrc = NewPkgsrc(firstDir.JoinNoClean(relTopdir))
		p.Wip = p.Pkgsrc.IsWip(firstDir) // See Pkglint.checkMode.
		p.Pkgsrc.LoadInfrastructure()
```

Now the information from pkgsrc is loaded into `pkglint.Pkgsrc`, and the main work can start.
The items from the TODO list are worked off and handed over to `Pkglint.Check`,
one after another. When pkglint is called with the `-r` option,
some entries may be added to the `Todo` list,
but that doesn't happen in this simple example run.

> from [pkglint.go](pkglint.go#L128):

```go
	for !p.Todo.IsEmpty() {
		p.Check(p.Todo.Pop())
	}
```

The main work is done in `Pkglint.Check` and `Pkglint.checkMode`:

> from [pkglint.go](pkglint.go#L325):

```go
	if isReg && p.Pkgsrc == nil {
		CheckFileMk(dirent, nil)
		return
	}
```

Since `DESCR` is a regular file, the next function to call is `checkReg`.
For directories, the next function would depend on the depth from the
pkgsrc root directory.

> from [pkglint.go](pkglint.go#L601):

```go
// checkReg checks the given regular file.
// depth is 3 for files in the package directory, and 4 or more for files
// deeper in the directory hierarchy, such as in files/ or patches/.
func (p *Pkglint) checkReg(filename CurrPath, basename RelPath, depth int, pkg *Package) {
```

The relevant part of `Pkglint.checkReg` is:

> from [pkglint.go](pkglint.go#L631):

```go
	case basename == "buildlink3.mk":
		if mklines := LoadMk(filename, pkg, NotEmpty|LogErrors); mklines != nil {
			CheckLinesBuildlink3Mk(mklines)
		}

	case p.Wip && basename == "COMMIT_MSG":
		// https://mail-index.netbsd.org/pkgsrc-users/2020/05/10/msg031174.html

	case basename.HasPrefixText("DESCR"):
		if lines := Load(filename, NotEmpty|LogErrors); lines != nil {
			CheckLinesDescr(lines)
			G.InterPackage.CheckDuplicateDescr(filename)
		}

	case basename == "distinfo":
		if lines := Load(filename, NotEmpty|LogErrors); lines != nil {
			CheckLinesDistinfo(pkg, lines)
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

> from [pkglint.go](pkglint.go#L474):

```go
func CheckLinesDescr(lines *Lines) {
	if trace.Tracing {
		defer trace.Call(lines.Filename)()
	}

	checkVarRefs := func(line *Line) {
		tokens, _ := NewMkLexer(line.Text, nil).MkTokens()
		for _, token := range tokens {
			switch {
			case token.Expr == nil,
				!hasPrefix(token.Text, "${"),
				G.Pkgsrc.VariableType(nil, token.Expr.varname) == nil:
			default:
				line.Notef("Variables like %q are not expanded in the DESCR file.",
					token.Text)
			}
		}
	}

	checkTodo := func(line *Line) {
		if hasPrefix(line.Text, "TODO:") {
			line.Errorf("DESCR files must not have TODO lines.")
		}
	}

	for _, line := range lines.Lines {
		ck := LineChecker{line}
		ck.CheckLength(80)
		ck.CheckTrailingWhitespace()
		ck.CheckValidCharacters()
		checkVarRefs(line)
		checkTodo(line)
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
Therefore, the checks for individual lines happen before the other check.

The call to `SaveAutofixChanges` at the end looks a bit strange
since none of the visible checks fixes anything.
The autofix feature must be hidden in one of the line checks,
and indeed, the code for `CheckTrailingWhitespace` says:

> from [linechecker.go](linechecker.go#L39):

```go
func (ck LineChecker) CheckTrailingWhitespace() {

	// Markdown files may need trailing whitespace. If there should ever
	// be Markdown files in pkgsrc, this code has to be adjusted.

	rawIndex := len(ck.line.raw) - 1
	text := ck.line.RawText(rawIndex)
	trimmedLen := len(rtrimHspace(text))
	if trimmedLen == len(text) {
		return
	}

	fix := ck.line.Autofix()
	fix.Notef("Trailing whitespace.")
	fix.Explain(
		"This whitespace is irrelevant and can be removed.")
	fix.ReplaceAt(rawIndex, trimmedLen, text[trimmedLen:], "")
	fix.Apply()
}
```

This code is a typical example for using the autofix feature.
Some more possibilities are described at the `Autofix` type itself
and at its typical call site `Line.Autofix()`:

> from [autofix.go](autofix.go#L14):

```go
// Autofix handles all modifications to a single line,
// possibly spanning multiple physical lines in case of makefile lines,
// describes them in a human-readable form and formats the output.
// The modifications are kept in memory only,
// until they are written to disk by SaveAutofixChanges.
type Autofix struct {
	line  *Line
	above []string // Newly inserted lines, including \n
	texts []string // Modified lines, including \n
	below []string // Newly inserted lines, including \n
	// Whether an actual fix has been applied to the text of the raw lines
	modified bool

	autofixShortTerm
}
```

> from [line.go](line.go#L178):

```go
// Autofix returns the autofix instance belonging to the line.
//
// Usage:
//
//	fix := line.Autofix()
//
//	fix.Errorf("Must not be ...")
//	fix.Warnf("Should not be ...")
//	fix.Notef("It is also possible ...")
//	fix.Silent()
//
//	fix.Explain(
//	    "Explanation ...",
//	    "... end of explanation.")
//
//	fix.Replace("from", "to")
//	fix.ReplaceAfter("prefix", "from", "to")
//	fix.InsertAbove("new line")
//	fix.InsertBelow("new line")
//	fix.Delete()
//	fix.Custom(func(showAutofix, autofix bool) {})
//
//	fix.Apply()
func (line *Line) Autofix() *Autofix {
```

The journey ends here, and it hasn't been that difficult.

If that was too easy, have a look at the code that decides whether an
expression such as `${CFLAGS}` needs to be quoted using the `:Q` modifier
when it is used in a shell command:

> from [mkline.go](mkline.go#L728):

```go
// VariableNeedsQuoting determines whether the given variable needs the :Q
// modifier in the given context.
//
// This decision depends on many factors, such as whether the type of the
// context is a list of things, whether the variable is a list, whether it
// can contain only safe characters, and so on.
func (mkline *MkLine) VariableNeedsQuoting(mklines *MkLines, expr *MkExpr, vartype *Vartype, ectx *ExprContext) (needsQuoting YesNoUnknown) {
	if trace.Tracing {
		defer trace.Call(expr, vartype, ectx, trace.Result(&needsQuoting))()
	}

	// TODO: Systematically test this function, each and every case, from top to bottom.
	// TODO: Re-check the order of all these if clauses whether it really makes sense.

	if expr.HasModifier("D") {
		// The :D modifier discards the value of the original variable and
		// replaces it with the expression from the :D modifier.
		// Therefore, the original variable does not need to be quoted.
		return unknown
	}

	ectxVartype := ectx.vartype
	if vartype == nil || ectxVartype == nil || vartype.basicType == BtUnknown {
		return unknown
	}

	if !vartype.basicType.NeedsQ() {
		if vartype.IsList() == no {
			if vartype.IsGuessed() {
				return unknown
			}
			return no
		}
		if !ectx.IsWordPart {
			return no
		}
	}

	// A shell word may appear as part of a shell word, for example COMPILER_RPATH_FLAG.
	if ectx.IsWordPart && ectx.quoting == EctxQuotPlain {
		if vartype.IsList() == no && vartype.basicType == BtShellWord {
			return no
		}
	}

	// Determine whether the context expects a list of shell words or not.
	wantList := ectxVartype.MayBeAppendedTo()
	haveList := vartype.MayBeAppendedTo()
	if trace.Tracing {
		trace.Stepf("wantList=%v, haveList=%v", wantList, haveList)
	}

	// Both of these can be correct, depending on the situation:
	// 1. echo ${PERL5:Q}
	// 2. xargs ${PERL5}
	if !ectx.IsWordPart && wantList && haveList {
		return unknown
	}

	// Pkglint assumes that the tool definitions don't include very
	// special characters, so they can safely be used inside any quotes.
	if tool := G.ToolByVarname(mklines, expr.varname); tool != nil {
		switch ectx.quoting {
		case EctxQuotPlain:
			if !ectx.IsWordPart {
				return no
			}
			// XXX: Should there be a return here? It looks as if it could have been forgotten.
		case EctxQuotBackt:
			return no
		case EctxQuotDquot, EctxQuotSquot:
			return unknown
		}
	}

	// Variables that appear as parts of shell words generally need to be quoted.
	//
	// An exception is in the case of backticks, because the whole backticks expression
	// is parsed as a single shell word by pkglint. (XXX: This comment may be outdated.)
	if ectx.IsWordPart && ectxVartype.IsShell() && ectx.quoting != EctxQuotBackt {
		return yes
	}

	// SUBST_MESSAGE.perl= Replacing in ${REPLACE_PERL}
	if ectxVartype.basicType == BtMessage {
		return no
	}

	if wantList != haveList {
		if ectxVartype.basicType == BtFetchURL && vartype.basicType == BtHomepage {
			return no
		}
		if ectxVartype.basicType == BtHomepage && vartype.basicType == BtFetchURL {
			return no // Just for HOMEPAGE=${MASTER_SITE_*:=subdir/}.
		}

		// .for dir in ${PATH:C,:, ,g}
		for _, modifier := range expr.modifiers {
			if modifier.ChangesList() {
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
		trace.Step1("Don't know whether :Q is needed for %q", expr.varname)
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

> from [line.go](line.go#L54):

```go
// Line represents a line of text from a file.
// In makefiles, a single "logical" line can consist of multiple "raw" lines,
// which happens when a line ends with an odd number of backslashes.
type Line struct {
	Location Location
	Basename RelPath // the basename from Location, for fast access

	// the text of the line, without the trailing newline character;
	// in Makefiles, also contains the text from the continuation lines,
	// joined by single spaces
	Text string

	raw  []*RawLine // contains the original text including trailing newline
	fix  *Autofix   // any changes that pkglint would like to apply to the line
	once Once
}
```

### MkLine

Most of the pkgsrc infrastructure is written in makefiles.
In these, there may be line continuations  (the ones ending in backslash).
Plus, they may contain Make variables of the form `${VARNAME}` or `${VARNAME:Modifiers}`,
and these are handled specially.

> from [mkline.go](mkline.go#L11):

```go
// MkLine is a line from a makefile fragment.
// There are several types of lines.
// The most common types in pkgsrc are variable assignments,
// shell commands and directives like .if and .for.
// The line types can be distinguished by IsVarassign,
// IsDirective and so on.
type MkLine struct {
	*Line

	splitResult mkLineSplitResult

	// One of the following mkLine* types.
	//
	// For the larger of these types, a pointer is used instead of a direct
	// struct because of https://github.com/golang/go/issues/28045.
	data interface{}
}
```

There are several types of lines in a makefile:

* comments and empty lines (trivial)
* variable assignments
* directives like `.if` and `.for`
* file inclusion, like `.include "../../mk/bsd.pkg.mk"`
* make targets like `pre-configure:` or `do-install:`
* shell commands for these targets, indented by a tab character

For each of these types, there is a corresponding type test,
such as `MkLine.IsVarassign()` or `MkLine.IsInclude()`.

Depending on this type, the individual properties of the line
can be accessed using `MkLine.Varname()` (for variable assignments only)
or `MkLine.DirectiveComment()` (for directives only).

### ShellLineChecker

The instructions for building and installing packages are written in shell commands,
which are embedded in makefile fragments.
The `ShellLineChecker` type provides methods for checking shell commands and their individual parts.

> from [shell.go](shell.go#L386):

```go
// ShellLineChecker checks either a line from a makefile starting with a tab,
// thereby containing shell commands to be executed.
//
// Or it checks a variable assignment line from a makefile with a left-hand
// side variable that is of some shell-like type; see Vartype.IsShell.
type ShellLineChecker struct {
	MkLines *MkLines
	mkline  *MkLine

	// checkExpr is set to false when checking a single shell word
	// in order to skip duplicate warnings in variable assignments.
	checkExpr bool
}
```

### Paths

Pkglint deals with all kinds of paths.
To avoid confusing these paths (which was more than easy as long as they
were all represented by simple strings), pkglint distinguishes these types
of paths:

* `CurrPath` is for paths given on the command line
    * these are used at the beginning of the diagnostics
* `PkgsrcPath` is for paths relative to the pkgsrc directory
    * `PKGPATH`
* `PackagePath` is for paths relative to the package directory
    * `PATCHDIR`
    * `DEPENDS`
* `RelPath` is for all other relative paths
    * paths that appear in the text of a diagnostic,
      these are relative to the line of a diagnostic
    * paths relative to the `PREFIX`
        * paths in `PLIST` files
        * paths in `ALTERNATIVES` files

All these path types are defined in `path.go`:

> from [path.go](path.go#L10):

```go
// Path is a slash-separated path.
// It may or may not resolve to an existing file.
// It may be absolute or relative.
// Some paths may contain placeholders like @VAR@ or ${VAR}.
// The base directory of relative paths is unspecified.
type Path string
```

> from [path.go](path.go#L231):

```go
// CurrPath is a path that is either absolute or relative to the current
// working directory. It is used in command line arguments and for
// loading files from the file system, and later in the diagnostics.
type CurrPath string
```

> from [path.go](path.go#L459):

```go
// RelPath is a path that is relative to some base directory that is not
// further specified.
type RelPath string
```

> from [path.go](path.go#L380):

```go
// PkgsrcPath is a path relative to the pkgsrc root.
type PkgsrcPath string
```

> from [path.go](path.go#L410):

```go
// PackagePath is a path relative to the package directory. It is used
// for the PATCHDIR and PKGDIR variables, as well as dependencies and
// conflicts on other packages.
//
// It can have two forms:
//   - patches (further down)
//   - ../../category/package/* (up to the pkgsrc root, then down again)
type PackagePath string
```

To convert between these paths, several of the pkglint types provide methods
called `File` and `Rel`:

* `File` converts a relative path to a `CurrPath`
* `Rel` converts a path to a relative path

Some types that provide these methods are `Pkgsrc`, `Package`, `Line`.

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

> from [check_test.go](check_test.go#L178):

```go
// Tester provides utility methods for testing pkglint.
// It is separated from the Suite since the latter contains
// all the test methods, which makes it difficult to find
// a method by auto-completion.
type Tester struct {
	c        *check.C // Only usable during the test method itself
	testName string
	argv     []string // from the last invocation of Tester.SetUpCommandLine

	stdout  bytes.Buffer
	stderr  bytes.Buffer
	tmpdir  CurrPath
	prevdir CurrPath // The current working directory before the test started
	cwd     RelPath  // relative to tmpdir; see Tester.Chdir

	seenSetUpCommandLine bool
	seenSetupPkgsrc      int
	seenFinish           bool
	seenMain             bool
}
```

The `s` variable is not used in tests.
The only purpose of its type `Suite` is to group the tests so they are all run together.

The `c` variable comes from [gocheck](https://godoc.org/gopkg.in/check.v1),
which is the underlying testing framework.
Most pkglint tests don't need this variable.

> from [util_test.go](util_test.go#L259):

```go
func (s *Suite) Test_tabWidth(c *check.C) {
	t := s.Init(c)

	t.CheckEquals(tabWidth("12345"), 5)
	t.CheckEquals(tabWidth("\t"), 8)
	t.CheckEquals(tabWidth("123\t"), 8)
	t.CheckEquals(tabWidth("1234567\t"), 8)
	t.CheckEquals(tabWidth("12345678\t"), 16)
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

To see how to set up complicated tests, have a look at the following test,
which sets up a realistic environment to run the tests in.

> from [pkglint_test.go](pkglint_test.go#L120):

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

	t.CreateFileLines("sysutils/checkperms/DESCR",
		"Description")

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
		"--- checkperms.c",
		"+++ checkperms.c",
		"@@ -1,1 +1,3 @@", // at line 1, delete 1 line; at line 1, add 3 lines
		"+// Header 1",
		"+// Header 2",
		"+// Header 3")
	t.CreateFileLines("sysutils/checkperms/distinfo",
		CvsID,
		"",
		"BLAKE2s (checkperms-1.12.tar.gz) = cd95029aa930b6201e9580b3ab7e36dd30b8f925",
		"SHA512 (checkperms-1.12.tar.gz) = "+
			"43e37b5963c63fdf716acdb470928d7e21a7bdfddd6c85cf626a11acc7f45fa5"+
			"2a53d4bcd83d543150328fe8cec5587987d2d9a7c5f0aaeb02ac1127ab41f8ae",
		"Size (checkperms-1.12.tar.gz) = 6621 bytes",
		"SHA1 (patch-checkperms.c) = asdfasdf") // Invalid SHA-1 checksum

	t.Main("-Wall", "-Call", "sysutils/checkperms")

	t.CheckOutputLines(
		"NOTE: ~/sysutils/checkperms/Makefile:3: "+
			"Package version \"1.11\" is greater than the latest \"1.10\" "+
			"from ../../doc/CHANGES-2018:5.",
		"WARN: ~/sysutils/checkperms/Makefile:3: "+
			"This package should be updated to 1.13 (supports more file formats; see ../../doc/TODO:5).",
		"ERROR: ~/sysutils/checkperms/Makefile:4: Invalid category \"tools\".",
		"ERROR: ~/sysutils/checkperms/TODO: Packages in main pkgsrc must not have a TODO file.",
		"ERROR: ~/sysutils/checkperms/distinfo:6: SHA1 hash of patches/patch-checkperms.c differs "+
			"(distinfo has asdfasdf, patch file has bcfb79696cb6bf4d2222a6d78a530e11bf1c0cea).",
		"WARN: ~/sysutils/checkperms/patches/patch-checkperms.c:12: Premature end of patch hunk "+
			"(expected 1 line to be deleted and 0 lines to be added).",
		"3 errors, 2 warnings and 1 note found.",
		t.Shquote("(Run \"pkglint -e -Wall -Call %s\" to show explanations.)", "sysutils/checkperms"),
		t.Shquote("(Run \"pkglint -fs -Wall -Call %s\" to show what can be fixed automatically.)", "sysutils/checkperms"),
		t.Shquote("(Run \"pkglint -F -Wall -Call %s\" to automatically fix some issues.)", "sysutils/checkperms"))
}
```

### Typical mistakes during a test

When running a newly written pkglint test, it may output more warnings than
necessary or interesting for the current test. Here are the most frequent
warnings and how to repair them properly:

#### Unknown shell command %q

* Load the standard variables using `t.SetUpVartypes()`
* Define the corresponding tool using `t.SetUpTool("tool", "TOOL", AtRunTime)`

#### Variable "%s" is used but not defined

* Load the standard variables using `t.SetUpVartypes()`

#### Variable "%s" is defined but not used

* Load the standard variables using `t.SetUpVartypes()`

#### The created MkLines are not found

Check whether you have created the lines using `t.NewLines`
instead of `t.CreateFileLines`.
The former creates the lines only in memory,
and the result of that method must be used,
otherwise the call doesn't make sense.

#### Test failure because of differing paths

If a test fails like this:

~~~text
obtained: file ../../../../AppData/Local/Temp/check-.../licenses/gpl-v2
expected: file ~/licenses/gpl-v2
~~~

Check whether you have created the lines using `t.NewLines`
instead of `t.CreateFileLines`.
The former creates the lines only in memory,
and the result of that method must be used,
otherwise the call doesn't make sense.
