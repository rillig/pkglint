package pkglint

// TODO: empty slice returns EOF
// TODO: a slice of a single token behaves like textproc.Lexer
// TODO: a slice of a single MkVarUse does not match any plain text patterns
// TODO: a slice of [plain token, MkVarUse] parses as expected
// TODO: a slice of [MkVarUse, MkVarUse, MkVarUse] parses as 3 variables
// TODO: mark, reset and since work in the initial state
// TODO: mark, reset and since work after parsing part of a plain text token
// TODO: mark, reset and since work after parsing a complete plain text token
// TODO: mark, reset and since work after parsing a MkVarUse token
// TODO: marks are independent of each other, especially when parsing parts of plain text tokens
// TODO: marks are independent of each other; the remaining tokens are copied properly
// TODO: EOF works in case of a trailing MkVarUse
// TODO: EOF works in case of a trailing plain text token
// TODO: the constructor copies the tokens so that parsing them multiple times is possible
// TODO: even without the append() calls the lexer and the marks are independent of each other
// TODO: MkTokensLexer is documented to assume that the underlying token array does not change
// TODO: all inherited methods from textproc.Lexer make sense
// TODO: all exported methods of MkTokensLexer are documented
// TODO: directly after a successful NextVarUse, PeekByte returns -1
