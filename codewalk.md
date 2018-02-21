# The pkglint tour

Note: I wish there were a tool for nicely rendering the below codewalk blocks.
If you know of such a tool, please tell me.
`godoc` doesn't count since [it is only supported for the Go distribution 
itself](https://github.com/golang/go/issues/14369).

## The entry points

### Running pkglint

```codewalk
file   pkglint.go
start  ^func main
end    ^\}
```

When running pkglint, the `G` variable is set up first.
It contains the whole global state of pkglint.
(Except for some of the subpackages, which have to be initialized separately.)
All the interesting code is in the `Pkglint` type.
This code structure makes it easy to test the code.

### Testing pkglint

Very similar code is used to set up the test and tear it down again:

```codewalk
file   check_test.go
start  ^func .* SetUpTest
end    ^\}
```

```codewalk
file   check_test.go
start  ^func .* TearDownTest
end    ^\}
```

## First contact: checking a single DESCR files

To learn how pkglint works internally, it is a good idea to start with
a very simple example.
Since the `DESCR` files have a very simple structure (they only contain
text for human consumption), they are the ideal target.


```codewalk
file   pkglint.go
start  /^func main/ upwhile /^\/\//
```

```codewalk
file   pkglint.go
start  ^[\t]if exitcode
end    ^\t\}$
```

```codewalk
file   pkglint.go
start  ^[\t]for _, arg
end    ^\}$
```

TODO

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
file   line.go
start  ^type Line =
```

```codewalk
file   line.go
start  ^type LineImpl struct
end    ^\}
```

### MkLine

```codewalk
file   mkline.go
start  ^type MkLine =
```

```codewalk
file   mkline.go
start  ^type MkLineImpl struct
end    ^\}
```

Most of the pkgsrc infrastructure is written in Makefiles. 
In these, there may be line continuations  (the ones ending in backslash).
Plus, they may contain Make variables of the form `${VARNAME}` or `${VARNAME:Modifiers}`,
and these are handled specially.

### ShellLine

The instructions for building and installing packages are written in Shell.
The `ShellLine` type provides methods for checking shell commands and their individual parts.

```codewalk
file   shell.go
start  ^type ShellLine struct
end    ^\}
```
