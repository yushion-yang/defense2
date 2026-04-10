// Package buff provides a unified status effect (buff/debuff) system.
// It manages stacking, duration, and aggregation for all entity buffs.
// This package has zero dependencies on other game packages.
package buff

// Category classifies a buff for filtering (e.g. Purge clears all CC).
type Category int

const (
	CatCC       Category = iota // stun, slow, root
	CatDoT                      // bleed, burn, poison
	CatDefense                  // controlImmune, invincible, damageImmune
	CatDebuff                   // weaken, silence
	CatBehavior                 // berserk, regen, healAura, buffer (Phase 2)
	CatAura                     // tower auras (Phase 3)
)

// StackMode determines how multiple instances of the same buff combine.
type StackMode int

const (
	Strongest          StackMode = iota // keep highest Value
	Additive                            // sum all Values
	Multiplicative                      // multiply all Values
	Override                            // latest replaces previous
	Independent                         // each calculated independently
	IndependentPerSource                // same Source overwrites, different Source coexist
)

// Buff represents a single active status effect.
type Buff struct {
	ID        string   // matches buff-stack.json key: "slow", "stun", "bleed"
	Category  Category // used for ClearByCategory (purge)
	Source    string   // e.g. "tower_3", "wave_buff", "ability_weaken"
	Value     float64  // primary value (slow factor, DPS, amplify ratio)
	Value2    float64  // secondary value (unused by most buffs)
	Duration  float64  // total duration (-1 = permanent)
	Remaining float64  // time left (-1 = permanent)
	Priority  int      // for Override mode tie-breaking
}
