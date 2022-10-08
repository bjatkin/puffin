package puffin

// Pattern is a pattern matcher for a FuncCmd
type Pattern interface {
	Match(*FuncCmd) bool
}

// muxMatcher maps a pat.Pattern to a specific CmdFunc handler
type muxMatcher struct {
	pat     Pattern
	handler CmdFunc
}

// Mux routs incoming commands to the appropriate handler functions
type Mux struct {
	matchers []muxMatcher
}

// NewMux returns a new mux with no configured handlers
func NewMux() *Mux {
	return &Mux{}
}

// HandleFunc adds a new handler func to the mux with the given Pattern matcher
func (m *Mux) HandleFunc(pat Pattern, handler CmdFunc) {
	m.matchers = append(m.matchers, muxMatcher{
		pat:     pat,
		handler: handler,
	})
}

// findHandler searches through all the mux's Patterns to find a match
// if a match is found the corresponding CmdFunc is also returned
func (m *Mux) findHandler(cmd *FuncCmd) CmdFunc {
	if m == nil {
		return nil
	}

	for _, matcher := range m.matchers {
		if matcher.pat.Match(cmd) {
			return matcher.handler
		}
	}

	return nil
}
