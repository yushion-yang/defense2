package buff

// BuffList manages active buffs with stacking rules.
type BuffList struct {
	active   []Buff
	rules    map[string]StackRule
	dotTimer float64 // DoT tick accumulator (used by TickDoT)
}

// NewBuffList creates a BuffList with the given stacking rules.
func NewBuffList(rules map[string]StackRule) *BuffList {
	return &BuffList{
		active: make([]Buff, 0, 8),
		rules:  rules,
	}
}

// Add applies a buff according to stacking rules. Returns true if buff was applied.
func (bl *BuffList) Add(b Buff) bool {
	rule := bl.getRule(b.ID)

	switch rule.Mode {
	case Override:
		for i := range bl.active {
			if bl.active[i].ID == b.ID {
				bl.active[i] = b
				return true
			}
		}
		bl.active = append(bl.active, b)

	case Strongest:
		for i := range bl.active {
			if bl.active[i].ID == b.ID {
				// Replace if new Value is strictly higher,
				// or same Value but longer Remaining.
				if b.Value > bl.active[i].Value ||
					(b.Value == bl.active[i].Value && b.Remaining > bl.active[i].Remaining) {
					bl.active[i] = b
				}
				return true
			}
		}
		bl.active = append(bl.active, b)

	case IndependentPerSource:
		for i := range bl.active {
			if bl.active[i].ID == b.ID && bl.active[i].Source == b.Source {
				bl.active[i] = b
				return true
			}
		}
		bl.active = append(bl.active, b)

	case Additive, Multiplicative, Independent:
		bl.active = append(bl.active, b)

	default:
		// Unknown mode: treat as override
		for i := range bl.active {
			if bl.active[i].ID == b.ID {
				bl.active[i] = b
				return true
			}
		}
		bl.active = append(bl.active, b)
	}
	return true
}

// Has returns true if any buff with the given ID is active.
func (bl *BuffList) Has(id string) bool {
	for i := range bl.active {
		if bl.active[i].ID == id {
			return true
		}
	}
	return false
}

// Get returns the effective buff for the given ID, merged by stacking rules.
//   - Override / Strongest: single buff (already resolved in Add)
//   - Additive: synthetic buff with summed Value
//   - Multiplicative: synthetic buff with multiplied Value
//   - IndependentPerSource: first instance (use GetAll for all)
//
// Cap/Floor from rules are applied after aggregation.
func (bl *BuffList) Get(id string) (Buff, bool) {
	rule := bl.getRule(id)
	var found bool
	var result Buff

	for i := range bl.active {
		if bl.active[i].ID != id {
			continue
		}
		if !found {
			result = bl.active[i]
			found = true
			continue
		}
		switch rule.Mode {
		case Additive:
			result.Value += bl.active[i].Value
		case Multiplicative:
			result.Value *= bl.active[i].Value
		default:
			// Override/Strongest: already single from Add.
			// IndependentPerSource: return first.
		}
	}

	if !found {
		return Buff{}, false
	}

	// Apply cap/floor
	if rule.Cap > 0 && result.Value > rule.Cap {
		result.Value = rule.Cap
	}
	if rule.Floor > 0 && result.Value < rule.Floor {
		result.Value = rule.Floor
	}

	return result, true
}

// GetAll returns all active buffs with the given ID (useful for IndependentPerSource).
func (bl *BuffList) GetAll(id string) []Buff {
	var result []Buff
	for i := range bl.active {
		if bl.active[i].ID == id {
			result = append(result, bl.active[i])
		}
	}
	return result
}

// Remove removes a buff by ID and Source.
func (bl *BuffList) Remove(id, source string) {
	n := 0
	for i := range bl.active {
		if bl.active[i].ID == id && bl.active[i].Source == source {
			continue
		}
		bl.active[n] = bl.active[i]
		n++
	}
	bl.active = bl.active[:n]
}

// RemoveByID removes all buffs with the given ID.
func (bl *BuffList) RemoveByID(id string) {
	n := 0
	for i := range bl.active {
		if bl.active[i].ID == id {
			continue
		}
		bl.active[n] = bl.active[i]
		n++
	}
	bl.active = bl.active[:n]
}

// ClearByCategory removes all buffs matching any of the given categories.
func (bl *BuffList) ClearByCategory(cats ...Category) {
	catSet := make(map[Category]bool, len(cats))
	for _, c := range cats {
		catSet[c] = true
	}
	n := 0
	for i := range bl.active {
		if catSet[bl.active[i].Category] {
			continue
		}
		bl.active[n] = bl.active[i]
		n++
	}
	bl.active = bl.active[:n]
}

// Clear removes all buffs.
func (bl *BuffList) Clear() {
	bl.active = bl.active[:0]
}

// Active returns a snapshot (copy) of all active buffs for HUD display.
func (bl *BuffList) Active() []Buff {
	result := make([]Buff, len(bl.active))
	copy(result, bl.active)
	return result
}

// Count returns the number of active buffs.
func (bl *BuffList) Count() int {
	return len(bl.active)
}

// getRule returns the stacking rule for a buff ID, defaulting to Override.
func (bl *BuffList) getRule(id string) StackRule {
	if r, ok := bl.rules[id]; ok {
		return r
	}
	return StackRule{Mode: Override}
}
