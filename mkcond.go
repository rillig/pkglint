package main

func parseMkCond(line *Line, cond string) *Tree {
	defer tracecall("parseMkCond", cond)()

	const (
		repartVarname = `[A-Z_][A-Z0-9_]*(?:\.[\w_+\-]+)?`
		reDefined     = `^defined\((` + repartVarname + `)\)`
		reEmpty       = `^empty\((` + repartVarname + `)\)`
		reEmptyMatch  = `^empty\((` + repartVarname + `):M([^\$:{})]+)\)`
		reCompare     = `^\$\{(` + repartVarname + `)\}\s+(==|!=)\s+"([^"\$\\]*)"`
	)

	if m, rest := replaceFirst(cond, `^!`, ""); m != nil {
		return NewTree("not", parseMkCond(line, rest))
	}
	if m, rest := replaceFirst(cond, reDefined, ""); m != nil {
		return NewTree("defined", parseMkCond(line, rest))
	}
	if m, _ := replaceFirst(cond, reEmpty, ""); m != nil {
		return NewTree("empty", m[1])
	}
	if m, _ := replaceFirst(cond, reEmptyMatch, ""); m != nil {
		return NewTree("empty", NewTree("match", m[1], m[2]))
	}
	if m, _ := replaceFirst(cond, reCompare, ""); m != nil {
		return NewTree(m[2], NewTree("var", m[1]), NewTree("string", m[3]))
	}
	return NewTree("unknown", cond)
}
