package pkglint

// Example: "word1 word2;;;" => "word1", "word2", ";;", ";"
//
// TODO: Document what this function should be used for.
func splitIntoShellTokens(line Autofixer, text string) (tokens []string, rest string) {
	if trace.Tracing {
		defer trace.Call(line, text)()
	}

	// TODO: Check whether this function is used correctly by all callers.
	//  It may be better to use a proper shell parser instead of this tokenizer.

	p := NewShTokenizer(line, text)
	for {
		token := p.ShToken()
		if token == nil {
			break
		}
		tokens = append(tokens, token.MkText)
	}

	rest = p.parser.Rest()

	return
}
