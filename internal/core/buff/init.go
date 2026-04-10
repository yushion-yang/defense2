// init.go — global buff stacking rules singleton.
// InitGlobalRules must be called once at startup (by the config package)
// before any BuffList is created via NewDefaultBuffList.
package buff

import "sync"

var (
	globalRules     map[string]StackRule
	globalRulesOnce sync.Once
)

// InitGlobalRules parses buff-stack.json bytes and stores the resulting
// rules as a process-wide singleton. Protected by sync.Once — subsequent
// calls are no-ops.
func InitGlobalRules(jsonData []byte) error {
	var initErr error
	globalRulesOnce.Do(func() {
		rules, err := LoadRules(jsonData)
		if err != nil {
			initErr = err
			return
		}
		globalRules = rules
	})
	return initErr
}

// GlobalRules returns the loaded stacking rules map.
// Returns nil if InitGlobalRules has not been called.
func GlobalRules() map[string]StackRule {
	return globalRules
}

// NewDefaultBuffList creates a BuffList using the global stacking rules.
// If InitGlobalRules has not been called, falls back to an empty rule set
// (all buffs default to Override mode).
func NewDefaultBuffList() *BuffList {
	rules := globalRules
	if rules == nil {
		rules = map[string]StackRule{}
	}
	return NewBuffList(rules)
}
