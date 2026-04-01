package item

import "defense2/internal/core/tower"

// ApplyItem boosts the tower's stat corresponding to item kind k,
// then recalculates derived stats.
func ApplyItem(t *tower.Tower, k Kind) {
	v := Defs[k].BoostVal
	switch k {
	case KindBaseDamage:
		t.BaseDamage += v
	case KindPotentialDamage:
		t.PotentialDamage += v
	case KindBaseSpeed:
		t.BaseSpeed += v
	case KindPotentialSpeed:
		t.PotentialSpeed += v
	case KindBaseRange:
		t.BaseRange += v
	case KindPotentialRange:
		t.PotentialRange += v
	}
	t.RecalcStats()
}
