package main

type MkShWalker struct {
	Callback struct {
		List               func(list *MkShList)
		AndOr              func(andor *MkShAndOr)
		Pipeline           func(pipeline *MkShPipeline)
		Command            func(command *MkShCommand)
		SimpleCommand      func(command *MkShSimpleCommand)
		CompoundCommand    func(command *MkShCompoundCommand)
		Case               func(caseClause *MkShCaseClause)
		CaseItem           func(caseItem *MkShCaseItem)
		FunctionDefinition func(funcdef *MkShFunctionDefinition)
		If                 func(ifClause *MkShIfClause)
		Loop               func(loop *MkShLoopClause)
		Words              func(words []*ShToken)
		Word               func(word *ShToken)
		Redirects          func(redirects []*MkShRedirection)
		Redirect           func(redirect *MkShRedirection)
		For                func(forClause *MkShForClause)
		Varname            func(varname string)
	}
}

func NewMkShWalker() *MkShWalker {
	return &MkShWalker{}
}

// Walk calls the given callback for each node of the parsed shell program,
// in visiting order from large to small.
func (w *MkShWalker) Walk(list *MkShList) {
	w.walkList(list)
}

func (w *MkShWalker) walkList(list *MkShList) {
	if callback := w.Callback.List; callback != nil {
		callback(list)
	}

	for _, andor := range list.AndOrs {
		w.walkAndOr(andor)
	}
}

func (w *MkShWalker) walkAndOr(andor *MkShAndOr) {
	if callback := w.Callback.AndOr; callback != nil {
		callback(andor)
	}

	for _, pipeline := range andor.Pipes {
		w.walkPipeline(pipeline)
	}
}

func (w *MkShWalker) walkPipeline(pipeline *MkShPipeline) {
	if callback := w.Callback.Pipeline; callback != nil {
		callback(pipeline)
	}

	for _, command := range pipeline.Cmds {
		w.walkCommand(command)
	}
}

func (w *MkShWalker) walkCommand(command *MkShCommand) {
	if callback := w.Callback.Command; callback != nil {
		callback(command)
	}

	switch {
	case command.Simple != nil:
		w.walkSimpleCommand(command.Simple)
	case command.Compound != nil:
		w.walkCompoundCommand(command.Compound)
		w.walkRedirects(command.Redirects)
	case command.FuncDef != nil:
		w.walkFunctionDefinition(command.FuncDef)
		w.walkRedirects(command.Redirects)
	}
}

func (w *MkShWalker) walkSimpleCommand(command *MkShSimpleCommand) {
	if callback := w.Callback.SimpleCommand; callback != nil {
		callback(command)
	}

	w.walkWords(command.Assignments)
	if command.Name != nil {
		w.walkWord(command.Name)
	}
	w.walkWords(command.Args)
	w.walkRedirects(command.Redirections)
}

func (w *MkShWalker) walkCompoundCommand(command *MkShCompoundCommand) {
	if callback := w.Callback.CompoundCommand; callback != nil {
		callback(command)
	}

	switch {
	case command.Brace != nil:
		w.walkList(command.Brace)
	case command.Case != nil:
		w.walkCase(command.Case)
	case command.For != nil:
		w.walkFor(command.For)
	case command.If != nil:
		w.walkIf(command.If)
	case command.Loop != nil:
		w.walkLoop(command.Loop)
	case command.Subshell != nil:
		w.walkList(command.Subshell)
	}
}

func (w *MkShWalker) walkCase(caseClause *MkShCaseClause) {
	if callback := w.Callback.Case; callback != nil {
		callback(caseClause)
	}

	w.walkWord(caseClause.Word)
	for _, caseItem := range caseClause.Cases {
		if callback := w.Callback.CaseItem; callback != nil {
			callback(caseItem)
		}
		w.walkWords(caseItem.Patterns)
		w.walkList(caseItem.Action)
	}
}

func (w *MkShWalker) walkFunctionDefinition(funcdef *MkShFunctionDefinition) {
	if callback := w.Callback.FunctionDefinition; callback != nil {
		callback(funcdef)
	}

	w.walkCompoundCommand(funcdef.Body)
}

func (w *MkShWalker) walkIf(ifClause *MkShIfClause) {
	if callback := w.Callback.If; callback != nil {
		callback(ifClause)
	}

	for i, cond := range ifClause.Conds {
		w.walkList(cond)
		w.walkList(ifClause.Actions[i])
	}
	if ifClause.Else != nil {
		w.walkList(ifClause.Else)
	}
}

func (w *MkShWalker) walkLoop(loop *MkShLoopClause) {
	if callback := w.Callback.Loop; callback != nil {
		callback(loop)
	}

	w.walkList(loop.Cond)
	w.walkList(loop.Action)
}

func (w *MkShWalker) walkWords(words []*ShToken) {
	if len(words) != 0 {
		if callback := w.Callback.Words; callback != nil {
			callback(words)
		}

		for _, word := range words {
			w.walkWord(word)
		}
	}
}

func (w *MkShWalker) walkWord(word *ShToken) {
	if callback := w.Callback.Word; callback != nil {
		callback(word)
	}
}

func (w *MkShWalker) walkRedirects(redirects []*MkShRedirection) {
	if len(redirects) != 0 {
		if callback := w.Callback.Redirects; callback != nil {
			callback(redirects)
		}

		for _, redirect := range redirects {
			if callback := w.Callback.Redirect; callback != nil {
				callback(redirect)
			}

			w.walkWord(redirect.Target)
		}
	}
}

func (w *MkShWalker) walkFor(forClause *MkShForClause) {
	if callback := w.Callback.For; callback != nil {
		callback(forClause)
	}
	if callback := w.Callback.Varname; callback != nil {
		callback(forClause.Varname)
	}

	w.walkWords(forClause.Values)
	w.walkList(forClause.Body)
}
