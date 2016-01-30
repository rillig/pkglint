package main

import (
	"fmt"
)

type Tree struct {
	name string
	args []interface{}
}

func NewTree(name string, args ...interface{}) *Tree {
	return &Tree{name, args}
}

func (t *Tree) String() string {
	s := "(" + t.name
	for _, arg := range t.args {
		if arg, ok := arg.(*Tree); ok {
			s += " " + (*arg).String()
			continue
		}
		if arg, ok := arg.(string); ok {
			s += fmt.Sprintf(" %q", arg)
			continue
		} else {
			s += fmt.Sprintf(" %v", arg)
		}
	}
	return s + ")"
}
