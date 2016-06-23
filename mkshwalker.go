package main

type MkShWalker struct {
}

func (w *MkShWalker) Walk(list *MkShList, callback func(node interface{})) {
	for element := range w.iterate(list) {
		callback(element)
	}
}

func (w *MkShWalker) ForEachSimpleCommand(list *MkShList, callback func(cmd *MkShSimpleCommand)) {
	for element := range w.iterate(list) {
		if cmd, ok := element.(*MkShSimpleCommand); ok {
			callback(cmd)
		}
	}
}

func (w *MkShWalker) ForEachConditionalSimpleCommand(list *MkShList, callback func(cmd *MkShSimpleCommand)) {
	getSimple := func(list *MkShList) *MkShSimpleCommand {
		if len(list.AndOrs) == 1 {
			if len(list.AndOrs[0].Pipes) == 1 {
				if len(list.AndOrs[0].Pipes[0].Cmds) == 1 {
					return list.AndOrs[0].Pipes[0].Cmds[0].Simple
				}
			}
		}
		return nil
	}

	for element := range w.iterate(list) {
		if cmd, ok := element.(*MkShIfClause); ok {
			for _, cond := range cmd.Conds {
				if simple := getSimple(cond); simple != nil {
					callback(simple)
				}
			}
		}
		if cmd, ok := element.(*MkShLoopClause); ok {
			if simple := getSimple(cmd.Cond); simple != nil {
				callback(simple)
			}
		}
	}
}

func (w *MkShWalker) iterate(list *MkShList) chan interface{} {
	elements := make(chan interface{})

	go func() {
		w.walkList(list, elements)
		close(elements)
	}()

	return elements
}

func (w *MkShWalker) walkList(list *MkShList, collector chan interface{}) {
	collector <- list

	for _, andor := range list.AndOrs {
		w.walkAndOr(andor, collector)
	}
}

func (w *MkShWalker) walkAndOr(andor *MkShAndOr, collector chan interface{}) {
	collector <- andor

	for _, pipeline := range andor.Pipes {
		w.walkPipeline(pipeline, collector)
	}
}

func (w *MkShWalker) walkPipeline(pipeline *MkShPipeline, collector chan interface{}) {
	collector <- pipeline

	for _, command := range pipeline.Cmds {
		w.walkCommand(command, collector)
	}
}

func (w *MkShWalker) walkCommand(command *MkShCommand, collector chan interface{}) {
	collector <- command

	switch {
	case command.Simple != nil:
		w.walkSimpleCommand(command.Simple, collector)
	case command.Compound != nil:
		w.walkCompoundCommand(command.Compound, collector)
	case command.FuncDef != nil:
		w.walkFunctionDefinition(command.FuncDef, collector)
	}
}

func (w *MkShWalker) walkSimpleCommand(command *MkShSimpleCommand, collector chan interface{}) {
	collector <- command
}

func (w *MkShWalker) walkCompoundCommand(command *MkShCompoundCommand, collector chan interface{}) {
	collector <- command

	switch {
	case command.Brace != nil:
		w.walkList(command.Brace, collector)
	case command.Case != nil:
		for _, item := range command.Case.Cases {
			w.walkList(item.Action, collector)
		}
	case command.For != nil:
		w.walkList(command.For.Body, collector)
	case command.If != nil:
		for i, cond := range command.If.Conds {
			w.walkList(cond, collector)
			w.walkList(command.If.Actions[i], collector)
		}
		if command.If.Else != nil {
			w.walkList(command.If.Else, collector)
		}
	case command.Loop != nil:
		w.walkList(command.Loop.Cond, collector)
		w.walkList(command.Loop.Action, collector)
	case command.Subshell != nil:
		w.walkList(command.Subshell, collector)
	}
}

func (w *MkShWalker) walkFunctionDefinition(funcdef *MkShFunctionDefinition, collector chan interface{}) {
	collector <- funcdef

	w.walkCompoundCommand(funcdef.Body, collector)
}
