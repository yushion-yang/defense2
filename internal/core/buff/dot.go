package buff

// TickDoT processes DoT damage accumulation on a fixed interval.
// Returns total damage to deal this frame (0 if no tick fired).
// Call this BEFORE Tick() so expiring DoTs still deal their final tick.
//
// dotInterval is the tick period (e.g. 0.5 seconds).
// Value on each DoT buff is treated as DPS; damage per tick = Value * dotInterval.
func (bl *BuffList) TickDoT(dt, dotInterval float64) float64 {
	// Check if any DoT buff is active.
	hasDoT := false
	for i := range bl.active {
		if bl.active[i].Category == CatDoT {
			hasDoT = true
			break
		}
	}
	if !hasDoT {
		bl.dotTimer = 0
		return 0
	}

	// Accumulate time since last DoT tick.
	bl.dotTimer += dt
	if bl.dotTimer < dotInterval {
		return 0 // no tick this frame
	}

	// Tick fired — consume one interval (preserve remainder for next cycle).
	bl.dotTimer -= dotInterval

	// Sum damage from all DoT buffs.
	totalDmg := 0.0
	for i := range bl.active {
		b := &bl.active[i]
		if b.Category == CatDoT && b.Value > 0 {
			totalDmg += b.Value * dotInterval // DPS * interval
		}
	}
	return totalDmg
}
