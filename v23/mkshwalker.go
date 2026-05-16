package pkglint

type MkShWalker struct {
	Callback struct {
		List               func(list *MkShList)
		AndOr              func(andor *MkShAndOr)
		Pipeline           func(pipeline *MkShPipeline)
		Command            func(command *MkShCommand)
		SimpleCommand      func(command *MkShSimpleCommand)
		CompoundCommand    func(command *MkShCompoundCommand)
		Case               func(caseClause *MkShCase)
		CaseItem           func(caseItem *MkShCaseItem)
		FunctionDefinition func(funcdef *MkShFunctionDefinition)
		If                 func(ifClause *MkShIf)
		Loop               func(loop *MkShLoop)
		Words              func(words []*ShToken)
		Word               func(word *ShToken)
		Redirects          func(redirects []*MkShRedirection)
		Redirect           func(redirect *MkShRedirection)
		For                func(forClause *MkShFor)

		// For variable definition in a for loop.
		Varname func(varname string)
	}

	// Context[0] is the currently visited element,
	// Context[1] is its immediate parent element, and so on.
	// This is useful when the check for a CaseItem needs to look at the enclosing Case.
	Context []MkShWalkerPathElement
}

type MkShWalkerPathElement struct {

	// For fields that can be repeated, this is the index as seen from the parent element.
	// For fields that cannot be repeated, it is -1.
	//
	// For example, in the SimpleCommand "var=value cmd arg1 arg2",
	// there are multiple child elements of type Words.
	//
	// The first Words are the variable assignments, which have index 0.
	//
	// The command "cmd" has type Word, therefore it cannot be confused
	// with either of the Words lists and has index -1.
	//
	// The second Words are the arguments, which have index 1.
	// In this example, there are two arguments, so when visiting the
	// arguments individually, arg1 will have index 0 and arg2 will have index 1.
	//
	// TODO: It might be worth defining negative indexes to correspond
	//  to the fields "Cond", "Action", "Else", etc.
	Index  int
	Length int

	Element interface{}
}

func NewMkShWalker() *MkShWalker {
	return &MkShWalker{}
}

// Walk calls the given callback for each node of the parsed shell program,
// in visiting order from large to small.
func (w *MkShWalker) Walk(list *MkShList) {
	w.walkList(-1, -1, list)

	// The calls to w.push and w.pop must be balanced.
	assert(len(w.Context) == 0)
}

func (w *MkShWalker) walkList(index int, length int, list *MkShList) {
	w.push(index, length, list)

	if callback := w.Callback.List; callback != nil {
		callback(list)
	}

	for i, andor := range list.AndOrs {
		w.walkAndOr(i, len(list.AndOrs), andor)
	}

	w.pop()
}

func (w *MkShWalker) walkAndOr(index int, length int, andor *MkShAndOr) {
	w.push(index, length, andor)

	if callback := w.Callback.AndOr; callback != nil {
		callback(andor)
	}

	for i, pipeline := range andor.Pipes {
		w.walkPipeline(i, len(andor.Pipes), pipeline)
	}

	w.pop()
}

func (w *MkShWalker) walkPipeline(index int, length int, pipeline *MkShPipeline) {
	w.push(index, length, pipeline)

	if callback := w.Callback.Pipeline; callback != nil {
		callback(pipeline)
	}

	for i, command := range pipeline.Cmds {
		w.walkCommand(i, len(pipeline.Cmds), command)
	}

	w.pop()
}

func (w *MkShWalker) walkCommand(index int, length int, command *MkShCommand) {
	w.push(index, length, command)

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

	w.pop()
}

func (w *MkShWalker) walkSimpleCommand(command *MkShSimpleCommand) {
	w.push(-1, -1, command)

	if callback := w.Callback.SimpleCommand; callback != nil {
		callback(command)
	}

	w.walkWords(0, 2, command.Assignments)
	if command.Name != nil {
		w.walkWord(-1, -1, command.Name)
	}
	w.walkWords(1, 2, command.Args)
	w.walkRedirects(command.Redirections)

	w.pop()
}

func (w *MkShWalker) walkCompoundCommand(command *MkShCompoundCommand) {
	w.push(-1, -1, command)

	if callback := w.Callback.CompoundCommand; callback != nil {
		callback(command)
	}

	switch {
	case command.Brace != nil:
		w.walkList(-1, -1, command.Brace)
	case command.Case != nil:
		w.walkCase(command.Case)
	case command.For != nil:
		w.walkFor(command.For)
	case command.If != nil:
		w.walkIf(command.If)
	case command.Loop != nil:
		w.walkLoop(command.Loop)
	case command.Subshell != nil:
		w.walkList(-1, -1, command.Subshell)
	}

	w.pop()
}

func (w *MkShWalker) walkCase(caseClause *MkShCase) {
	w.push(-1, -1, caseClause)

	if callback := w.Callback.Case; callback != nil {
		callback(caseClause)
	}

	w.walkWord(-1, -1, caseClause.Word)
	for i, caseItem := range caseClause.Cases {
		w.push(i, len(caseClause.Cases), caseItem)
		if callback := w.Callback.CaseItem; callback != nil {
			callback(caseItem)
		}
		w.walkWords(-1, -1, caseItem.Patterns)
		if caseItem.Action != nil {
			w.walkList(-1, -1, caseItem.Action)
		}
		w.pop()
	}

	w.pop()
}

func (w *MkShWalker) walkFunctionDefinition(funcdef *MkShFunctionDefinition) {
	w.push(-1, -1, funcdef)

	if callback := w.Callback.FunctionDefinition; callback != nil {
		callback(funcdef)
	}

	w.walkCompoundCommand(funcdef.Body)

	w.pop()
}

func (w *MkShWalker) walkIf(ifClause *MkShIf) {
	w.push(-1, -1, ifClause)

	if callback := w.Callback.If; callback != nil {
		callback(ifClause)
	}

	// TODO: Replace these indices with proper field names; see MkShWalkerPathElement.Index.
	length := len(ifClause.Conds) + condInt(ifClause.Else != nil, 1, 0)
	for i, cond := range ifClause.Conds {
		w.walkList(2*i, length, cond)
		w.walkList(2*i+1, length, ifClause.Actions[i])
	}
	if ifClause.Else != nil {
		w.walkList(2*len(ifClause.Conds), length, ifClause.Else)
	}

	w.pop()
}

func (w *MkShWalker) walkLoop(loop *MkShLoop) {
	w.push(-1, -1, loop)

	if callback := w.Callback.Loop; callback != nil {
		callback(loop)
	}

	w.walkList(0, 2, loop.Cond)
	w.walkList(1, 2, loop.Action)

	w.pop()
}

func (w *MkShWalker) walkWords(index int, length int, words []*ShToken) {
	if len(words) == 0 {
		return
	}

	w.push(index, length, words)

	if callback := w.Callback.Words; callback != nil {
		callback(words)
	}

	for i, word := range words {
		w.walkWord(i, len(words), word)
	}

	w.pop()
}

func (w *MkShWalker) walkWord(index int, length int, word *ShToken) {
	w.push(index, length, word)

	if callback := w.Callback.Word; callback != nil {
		callback(word)
	}

	w.pop()
}

func (w *MkShWalker) walkRedirects(redirects []*MkShRedirection) {
	if len(redirects) == 0 {
		return
	}

	w.push(-1, -1, redirects)

	if callback := w.Callback.Redirects; callback != nil {
		callback(redirects)
	}

	for i, redirect := range redirects {
		w.push(i, len(redirects), redirect)
		if callback := w.Callback.Redirect; callback != nil {
			callback(redirect)
		}

		w.walkWord(i, len(redirects), redirect.Target)
		w.pop()
	}

	w.pop()
}

func (w *MkShWalker) walkFor(forClause *MkShFor) {
	w.push(-1, -1, forClause)

	if callback := w.Callback.For; callback != nil {
		callback(forClause)
	}
	if callback := w.Callback.Varname; callback != nil {
		callback(forClause.Varname)
	}

	w.walkWords(-1, -1, forClause.Values)
	w.walkList(-1, -1, forClause.Body)

	w.pop()
}

// Current provides access to the element that the walker is currently
// processing, especially its index as seen from its parent element.
func (w *MkShWalker) Current() MkShWalkerPathElement {
	return w.Context[len(w.Context)-1]
}

// Parent returns an ancestor element from the currently visited path.
// Parent(0) is the element that is currently visited,
// Parent(1) is its direct parent, and so on.
func (w *MkShWalker) Parent(steps int) interface{} {
	index := len(w.Context) - 1 - steps
	if index >= 0 {
		return w.Context[index].Element
	}
	return nil
}

func (w *MkShWalker) push(index int, length int, element interface{}) {
	w.Context = append(w.Context, MkShWalkerPathElement{index, length, element})
}

func (w *MkShWalker) pop() {
	w.Context = w.Context[:len(w.Context)-1]
}
