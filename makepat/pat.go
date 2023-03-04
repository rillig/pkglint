package makepat

import (
	"errors"
	"netbsd.org/pkglint/textproc"
)

// Pattern is a compiled pattern like "*.c" or "NetBSD-??.[^0-9]".
// It behaves exactly like in bmake,
// see devel/bmake/files/str.c, function Str_Match.
type Pattern struct {
	states []state
}

type state struct {
	transitions []transition
	end         bool
}

type transition struct {
	min, max byte
	to       StateID
}

type StateID uint16

// Compile parses a pattern, including the error checking that is missing
// from bmake.
func Compile(pattern string) (*Pattern, error) {
	var p Pattern
	s := p.addState(false)

	lex := textproc.NewLexer(pattern)
	for !lex.EOF() {
		ch := lex.NextByte()

		switch ch {
		case '*':
			p.addTransition(s, 0, 255, s)
		case '?':
			next := p.addState(false)
			p.addTransition(s, 0, 255, next)
			s = next
		case '\\':
			if lex.EOF() {
				return nil, errors.New("unfinished escape sequence")
			}
			ch := lex.NextByte()
			next := p.addState(false)
			p.addTransition(s, ch, ch, next)
			s = next
		case '[':
			next, err := compileCharClass(&p, lex, ch, s)
			if err != nil {
				return nil, err
			}
			s = next
		default:
			next := p.addState(false)
			p.addTransition(s, ch, ch, next)
			s = next
		}
	}

	p.states[s].end = true
	return &p, nil
}

func compileCharClass(p *Pattern, lex *textproc.Lexer, ch byte, s StateID) (StateID, error) {
	negate := lex.SkipByte('^')
	var chars [256]bool
	next := p.addState(false)
	for {
		if lex.EOF() {
			return 0, errors.New("unfinished character class")
		}
		ch = lex.NextByte()
		if ch == ']' {
			break
		}
		if lex.SkipByte('-') {
			if lex.EOF() {
				return 0, errors.New("unfinished character range")
			}
			max := lex.NextByte()
			if ch > max {
				ch, max = max, ch
			}
			for i := int(ch); i <= int(max); i++ {
				chars[i] = true
			}
		} else {
			chars[ch] = true
		}
	}
	if negate {
		for i, b := range chars {
			chars[i] = !b
		}
	}

	p.addTransitions(s, &chars, next)
	return next, nil
}

func (p *Pattern) addTransitions(from StateID, chars *[256]bool, to StateID) {
	start := 0
	for start < len(chars) && !chars[start] {
		start++
	}

	for start < len(chars) {
		end := start
		for end < len(chars) && chars[end] {
			end++
		}

		if start < end {
			p.addTransition(from, byte(start), byte(end-1), to)
		}

		start = end
		for start < len(chars) && !chars[start] {
			start++
		}
	}
}

func (p *Pattern) addState(end bool) StateID {
	p.states = append(p.states, state{nil, end})
	return StateID(len(p.states) - 1)
}

func (p *Pattern) addTransition(from StateID, min, max byte, to StateID) {
	state := &p.states[from]
	state.transitions = append(state.transitions, transition{min, max, to})
}

// Match tests whether a pattern matches the given string.
func (p *Pattern) Match(s string) bool {
	if len(p.states) == 0 {
		return false
	}

	curr := make([]bool, len(p.states))
	next := make([]bool, len(p.states))

	curr[0] = true
	for _, ch := range []byte(s) {
		ok := false
		for i := range next {
			next[i] = false
		}

		for si := range curr {
			if !curr[si] {
				continue
			}
			for _, tr := range p.states[si].transitions {
				if tr.min <= ch && ch <= tr.max {
					next[tr.to] = true
					ok = true
				}
			}
		}
		if !ok {
			return false
		}
		curr, next = next, curr
	}

	for i, curr := range curr {
		if curr && p.states[i].end {
			return true
		}
	}
	return false
}

// Intersect computes a pattern that only matches if both given patterns
// match at the same time.
func Intersect(p1, p2 *Pattern) *Pattern {
	var res Pattern

	newState := make(map[[2]StateID]StateID)

	// stateFor returns the state ID in the intersection,
	// creating it if necessary.
	stateFor := func(s1, s2 StateID) StateID {
		key := [2]StateID{s1, s2}
		ns, ok := newState[key]
		if !ok {
			ns = res.addState(p1.states[s1].end && p2.states[s2].end)
			newState[key] = ns
		}
		return ns
	}

	// Each pattern needs a start node.
	stateFor(0, 0)

	for i1, s1 := range p1.states {
		for i2, s2 := range p2.states {
			for _, t1 := range s1.transitions {
				for _, t2 := range s2.transitions {
					min := bmax(t1.min, t2.min)
					max := bmin(t1.max, t2.max)
					if min <= max {
						from := stateFor(StateID(i1), StateID(i2))
						to := stateFor(t1.to, t2.to)
						res.addTransition(from, min, max, to)
					}
				}
			}
		}
	}

	return res.optimized()
}

func (p *Pattern) optimized() *Pattern {
	reachable := p.reachable()
	relevant := p.relevant(reachable)
	return p.compressed(relevant)
}

// reachable returns all states that are reachable from the start state.
// In optimized patterns, each state is reachable.
func (p *Pattern) reachable() []bool {
	reachable := make([]bool, len(p.states))

	progress := make([]int, len(p.states)) // 0 = unseen, 1 = to do, 2 = done

	progress[0] = 1

	for {
		changed := false
		for i, pr := range progress {
			if pr == 1 {
				reachable[i] = true
				progress[i] = 2
				changed = true
				for _, tr := range p.states[i].transitions {
					if progress[tr.to] == 0 {
						progress[tr.to] = 1
					}
				}
			}
		}

		if !changed {
			break
		}
	}

	return reachable
}

// relevant returns all states from which an end state is reachable.
// In optimized patterns, each state is relevant.
func (p *Pattern) relevant(reachable []bool) []bool {
	relevant := make([]bool, len(p.states))

	progress := make([]int, len(p.states)) // 0 = unseen, 1 = to do, 2 = done

	for i, state := range p.states {
		if state.end && reachable[i] {
			progress[i] = 1
		}
	}

	for {
		changed := false
		for to, pr := range progress {
			if pr != 1 {
				continue
			}
			progress[to] = 2
			relevant[to] = true
			changed = true
			for from, st := range p.states {
				for _, tr := range st.transitions {
					if tr.to == StateID(to) && reachable[from] &&
						progress[from] == 0 {
						progress[from] = 1
					}
				}
			}
		}

		if !changed {
			break
		}
	}

	return relevant
}

// compressed creates a pattern that contains only the relevant states.
func (p *Pattern) compressed(relevant []bool) *Pattern {
	var opt Pattern

	newIDs := make([]StateID, len(p.states))
	for i, r := range relevant {
		if r {
			newIDs[i] = opt.addState(p.states[i].end)
		}
	}

	for from, s := range p.states {
		for _, t := range s.transitions {
			if relevant[from] && relevant[t.to] {
				opt.addTransition(newIDs[from], t.min, t.max, newIDs[t.to])
			}
		}
	}

	return &opt
}

// CanMatch tests whether the pattern can match some string.
// Most patterns can do that.
// Typical counterexamples are:
//
//	[^]
//	Intersect("*.c", "*.h")
func (p *Pattern) CanMatch() bool {
	if len(p.states) == 0 {
		return false
	}

	reachable := p.reachable()

	for i, s := range p.states {
		if reachable[i] && s.end {
			return true
		}
	}
	return false
}

// CompileLimited creates a pattern that matches all strings consisting of the
// given bytes only. Such a pattern cannot be expressed in the string form.
func CompileLimited(limitedTo string) *Pattern {
	var p Pattern
	s := p.addState(true)

	var chars [256]bool
	for _, b := range []byte(limitedTo) {
		chars[b] = true
	}

	p.addTransitions(s, &chars, s)
	return &p
}

// Float creates a pattern that matches integer or floating point constants,
// as in C99, both decimal and hex.
func Float() *Pattern {
	var p Pattern

	// The states and transitions are taken from a manually constructed
	// hand-drawn state diagram, based on the syntax rules from C99 6.4.4.

	start := p.addState(false)
	sign := p.addState(false)
	p.addTransition(start, '+', '+', sign)
	p.addTransition(start, '-', '-', sign)

	dec := p.addState(true)
	decDotUnfinished := p.addState(false)
	decDot := p.addState(true)
	decDotDec := p.addState(true)
	p.addTransition(start, '0', '9', dec)
	p.addTransition(sign, '0', '9', dec)
	p.addTransition(dec, '0', '9', dec)
	p.addTransition(start, '.', '.', decDotUnfinished)
	p.addTransition(sign, '.', '.', decDotUnfinished)
	p.addTransition(dec, '.', '.', decDot)
	p.addTransition(start, '0', '9', decDotDec)
	p.addTransition(sign, '0', '9', decDotDec)
	p.addTransition(decDotUnfinished, '0', '9', decDotDec)
	p.addTransition(decDot, '0', '9', decDotDec)
	p.addTransition(decDotDec, '0', '9', decDotDec)

	decExp := p.addState(false)
	decExpSign := p.addState(true)
	decExpDec := p.addState(true)
	p.addTransition(decDotDec, 'E', 'E', decExp)
	p.addTransition(decDotDec, 'e', 'e', decExp)
	p.addTransition(decExp, '+', '+', decExpSign)
	p.addTransition(decExp, '-', '-', decExpSign)
	p.addTransition(decExp, '0', '9', decExpDec)
	p.addTransition(decExpSign, '0', '9', decExpDec)
	p.addTransition(decExpDec, '0', '9', decExpDec)

	z := p.addState(false)
	zx := p.addState(false)
	p.addTransition(start, '0', '0', z)
	p.addTransition(z, 'X', 'X', zx)
	p.addTransition(z, 'x', 'x', zx)

	hex := p.addState(true)
	hexDotUnfinished := p.addState(false)
	hexDot := p.addState(false)
	hexDotHex := p.addState(false)
	p.addTransition(zx, '0', '9', hex)
	p.addTransition(zx, 'A', 'F', hex)
	p.addTransition(zx, 'a', 'f', hex)
	p.addTransition(hex, '0', '9', hex)
	p.addTransition(hex, 'A', 'F', hex)
	p.addTransition(hex, 'a', 'f', hex)
	p.addTransition(zx, '.', '.', hexDotUnfinished)
	p.addTransition(hex, '.', '.', hexDot)
	p.addTransition(zx, '0', '9', hexDotHex)
	p.addTransition(zx, 'A', 'F', hexDotHex)
	p.addTransition(zx, 'a', 'f', hexDotHex)
	p.addTransition(hexDotUnfinished, '0', '9', hexDotHex)
	p.addTransition(hexDotUnfinished, 'A', 'F', hexDotHex)
	p.addTransition(hexDotUnfinished, 'a', 'f', hexDotHex)
	p.addTransition(hexDot, '0', '9', hexDotHex)
	p.addTransition(hexDot, 'A', 'F', hexDotHex)
	p.addTransition(hexDot, 'a', 'f', hexDotHex)
	p.addTransition(hexDotHex, '0', '9', hexDotHex)
	p.addTransition(hexDotHex, 'A', 'F', hexDotHex)
	p.addTransition(hexDotHex, 'a', 'f', hexDotHex)

	hexExp := p.addState(false)
	hexExpSign := p.addState(false)
	hexExpDec := p.addState(true)
	p.addTransition(hexDotHex, 'P', 'P', hexExp)
	p.addTransition(hexDotHex, 'p', 'p', hexExp)
	p.addTransition(hexExp, '+', '+', hexExpSign)
	p.addTransition(hexExp, '-', '-', hexExpSign)
	p.addTransition(hexExpSign, '0', '9', hexExpDec)
	p.addTransition(hexExpDec, '0', '9', hexExpDec)

	return &p
}

func bmin(a, b byte) byte {
	if a < b {
		return a
	}
	return b
}

func bmax(a, b byte) byte {
	if a > b {
		return a
	}
	return b
}
