package main

func mklines(fname string, lines ...string) []*Line {
	result := make([]*Line, len(lines))
	for i, line := range lines {
		result[i] = NewLine(fname, sprintf("%d", i+1), line, nil)
	}
	return result
}

func UseCommandLine(args ...string) {
	G.opts = ParseCommandLine(append([]string{"pkglint"}, args...), G.logOut)
}
