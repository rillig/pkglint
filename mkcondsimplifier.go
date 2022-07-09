package pkglint

import "netbsd.org/pkglint/textproc"

// MkCondSimplifier replaces unnecessarily complex conditions with simpler yet
// equivalent conditions.
type MkCondSimplifier struct {
	MkLines *MkLines
	MkLine  *MkLine
}

// SimplifyVarUse replaces an unnecessarily complex condition with
// a simpler condition that's still equivalent.
//
// * fromEmpty is true for the form empty(VAR...), and false for ${VAR...}.
//
// * neg is true for the form !empty(VAR...), and false for empty(VAR...).
func (s *MkCondSimplifier) SimplifyVarUse(varuse *MkVarUse, fromEmpty bool, neg bool) {
	varname := varuse.varname
	modifiers := varuse.modifiers

	n := len(modifiers)
	if n == 0 {
		return
	}
	modsExceptLast := NewMkVarUse("", modifiers[:n-1]...).Mod()
	vartype := G.Pkgsrc.VariableType(s.MkLines, varname)

	isDefined := func() bool {
		if vartype.IsAlwaysInScope() && vartype.IsDefinedIfInScope() {
			return true
		}

		// For run time expressions, such as ${${VAR} == value:?yes:no},
		// the scope would need to be changed to ck.MkLines.allVars.
		if s.MkLines.checkAllData.vars.IsDefined(varname) {
			return true
		}

		return s.MkLines.Tools.SeenPrefs &&
			vartype.Union().Contains(aclpUseLoadtime) &&
			vartype.IsDefinedIfInScope()
	}

	// replace constructs the state before and after the autofix.
	// The before state is constructed to ensure that only very simple
	// patterns get replaced automatically.
	//
	// Before putting any cases involving special characters into
	// production, there need to be more tests for the edge cases.
	replace := func(positive bool, pattern string) (bool, string, string) {
		defined := isDefined()
		if !defined && !positive {
			// TODO: This is a double negation, maybe even triple.
			//  There is an :N pattern, and the variable may be undefined.
			//  If it is indeed undefined, should the whole condition
			//  evaluate to true or false?
			//  The cases to be distinguished are: undefined, empty, filled.

			// For now, be conservative and don't suggest anything wrong.
			return false, "", ""
		}
		uMod := condStr(!defined && !varuse.HasModifier("U"), ":U", "")

		op := condStr(neg == positive, "==", "!=")

		from := sprintf("%s%s%s%s%s%s%s",
			condStr(neg != fromEmpty, "", "!"),
			condStr(fromEmpty, "empty(", "${"),
			varname,
			modsExceptLast,
			condStr(positive, ":M", ":N"),
			pattern,
			condStr(fromEmpty, ")", "}"))

		needsQuotes := textproc.NewLexer(pattern).NextBytesSet(mkCondStringLiteralUnquoted) != pattern ||
			pattern == "" ||
			matches(pattern, `^\d+\.?\d*$`)
		quote := condStr(needsQuotes, "\"", "")

		to := sprintf(
			"${%s%s%s} %s %s%s%s",
			varname, uMod, modsExceptLast, op, quote, pattern, quote)

		return true, from, to
	}

	modifier := modifiers[n-1]
	ok, positive, pattern, exact := modifier.MatchMatch()
	if !ok || !positive && n != 1 {
		return
	}

	// Replace !empty(VAR:M*.c) with ${VAR:M*.c}.
	// Replace empty(VAR:M*.c) with !${VAR:M*.c}.
	if fromEmpty && positive && !exact && vartype != nil && isDefined() &&
		// Restrict replacements to very simple patterns with only few
		// special characters, for now.
		// Before generalizing this to arbitrary strings, there has to be
		// a proper code generator for these conditions that handles all
		// possible escaping.
		// The same reasoning applies to the variable name, even though the
		// variable name typically only uses a restricted character set.
		matches(varuse.Mod(), `^[*.:\w\[\]]+$`) {

		fixedPart := varname + modsExceptLast + ":M" + pattern
		from := condStr(neg, "!", "") + "empty(" + fixedPart + ")"
		to := condStr(neg, "", "!") + "${" + fixedPart + "}"

		fix := s.MkLine.Autofix()
		fix.Notef("%q can be simplified to %q.", from, to)
		fix.Explain(
			"This variable is guaranteed to be defined at this point.",
			"Therefore it may occur on the left-hand side of a comparison",
			"and doesn't have to be guarded by the function 'empty'.")
		fix.Replace(from, to)
		fix.Apply()

		return
	}

	switch {
	case !exact,
		vartype == nil,
		vartype.IsList(),
		textproc.NewLexer(pattern).NextBytesSet(mkCondModifierPatternLiteral) != pattern:
		return
	}

	ok, from, to := replace(positive, pattern)
	if !ok {
		return
	}

	fix := s.MkLine.Autofix()
	fix.Notef("%s can be compared using the simpler \"%s\" "+
		"instead of matching against %q.",
		varname, to, ":"+modifier.String()) // TODO: Quoted
	fix.Explain(
		"This variable has a single value, not a list of values.",
		"Therefore it feels strange to apply list operators like :M and :N onto it.",
		"A more direct approach is to use the == and != operators.",
		"",
		"An entirely different case is when the pattern contains",
		"wildcards like *, ?, [].",
		"In such a case, using the :M or :N modifiers is useful and preferred.")
	fix.Replace(from, to)
	fix.Apply()
}
