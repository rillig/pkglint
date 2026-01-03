package pkglint

// Diagnostics collects diagnostics by line, so that they can be output later
// in source code order, no matter in which order they were generated.
//
// The typical workflow is:
// First, Add is called to collect the diagnostics.
// Then Emit is called for each line that might be affected.
// Finally, AssertEmpty ensures that all diagnostics have been emitted.
type Diagnostics struct {
	diagnostics map[*Line][]Diagnostic
}

type Diagnostic struct {
	level       *LogLevel
	format      string
	arguments   []interface{}
	explanation []string
}

func (d *Diagnostics) Defer(line *Line) Diagnoser {
	return &DeferredDiagnoser{d, line}
}

// Add remembers a diagnostic as belonging to a particular line.
func (d *Diagnostics) Add(line *Line, level *LogLevel, format string, args ...interface{}) {
	if d.diagnostics == nil {
		d.diagnostics = make(map[*Line][]Diagnostic)
	}
	d.diagnostics[line] = append(d.diagnostics[line], Diagnostic{level, format, args, nil})
}

// Emit outputs the diagnostics for the line, in creation order.
func (d *Diagnostics) Emit(line *Line) {
	logger := G.Logger
	for _, diagnostic := range d.diagnostics[line] {
		if logger.shallBeLogged(diagnostic.format) {
			logger.Diag(line, diagnostic.level, diagnostic.format, diagnostic.arguments...)
			if len(diagnostic.explanation) > 0 {
				logger.Explain(diagnostic.explanation...)
			}
		}
	}
	delete(d.diagnostics, line)
}

// AssertEmpty ensures that all diagnostics have been emitted.
func (d *Diagnostics) AssertEmpty() {
	for line := range d.diagnostics {
		panic(line.String())
	}
}

type DeferredDiagnoser struct {
	diagnostics *Diagnostics
	line        *Line
}

func (d *DeferredDiagnoser) Errorf(format string, args ...interface{}) {
	d.diagnostics.Add(d.line, Error, format, args...)
}

func (d *DeferredDiagnoser) Warnf(format string, args ...interface{}) {
	d.diagnostics.Add(d.line, Warn, format, args...)
}

func (d *DeferredDiagnoser) Notef(format string, args ...interface{}) {
	d.diagnostics.Add(d.line, Note, format, args...)
}

func (d *DeferredDiagnoser) Explain(explanation ...string) {
	diagnostics := d.diagnostics.diagnostics[d.line]
	last := &diagnostics[len(diagnostics)-1]
	last.explanation = append(last.explanation, explanation...)
	d.diagnostics.diagnostics[d.line] = diagnostics
}
