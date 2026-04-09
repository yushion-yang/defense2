// assertion.go — 场景断言框架。
// 为测试场景提供行为检查和结果收集。
package autoplay

// Assertion defines a behavioral check for a test scenario.
type Assertion struct {
	Name      string                        // human-readable name
	AfterTick int                           // minimum tick before checking (0=immediate)
	Check     func(state *GameState) bool   // returns true if assertion passes
	Detail    func(state *GameState) string // failure description
}

// AssertionResult is the outcome of a single assertion check.
type AssertionResult struct {
	Name   string `json:"name"`
	Passed bool   `json:"passed"`
	Tick   int    `json:"tick"`
	Detail string `json:"detail,omitempty"`
}

// AssertionChecker manages per-scenario assertions.
type AssertionChecker struct {
	assertions []Assertion
	results    []AssertionResult
	checked    map[string]bool // already checked (for one-shot assertions)
}

// NewAssertionChecker creates a new assertion checker.
func NewAssertionChecker(assertions []Assertion) *AssertionChecker {
	return &AssertionChecker{
		assertions: assertions,
		checked:    make(map[string]bool),
	}
}

// Check runs all pending assertions against the current state.
func (ac *AssertionChecker) Check(state *GameState) {
	for i := range ac.assertions {
		a := &ac.assertions[i]
		if ac.checked[a.Name] {
			continue
		}
		if state.Tick < a.AfterTick {
			continue
		}
		if a.Check(state) {
			ac.results = append(ac.results, AssertionResult{
				Name: a.Name, Passed: true, Tick: state.Tick,
			})
			ac.checked[a.Name] = true
		}
	}
}

// Finalize marks unchecked assertions as failed.
func (ac *AssertionChecker) Finalize(lastTick int) []AssertionResult {
	for i := range ac.assertions {
		a := &ac.assertions[i]
		if ac.checked[a.Name] {
			continue
		}
		detail := a.Name + " never passed"
		if a.Detail != nil {
			detail = a.Detail(nil)
		}
		ac.results = append(ac.results, AssertionResult{
			Name: a.Name, Passed: false, Tick: lastTick, Detail: detail,
		})
	}
	return ac.results
}

// Results returns current assertion results (passed ones so far).
func (ac *AssertionChecker) Results() []AssertionResult {
	return ac.results
}
