package item

import "image/color"

// Kind identifies an item type.
type Kind int

const (
	KindBaseDamage      Kind = iota // 攻击磨石 — +BaseDamage
	KindPotentialDamage             // 攻击秘卷 — +PotentialDamage
	KindBaseSpeed                   // 速射齿轮 — +BaseSpeed
	KindPotentialSpeed              // 速射秘卷 — +PotentialSpeed
	KindBaseRange                   // 瞄准镜片 — +BaseRange
	KindPotentialRange              // 瞄准秘卷 — +PotentialRange
	KindCount                       // sentinel
)

// AllKinds enumerates every valid Kind.
var AllKinds = [...]Kind{
	KindBaseDamage, KindPotentialDamage,
	KindBaseSpeed, KindPotentialSpeed,
	KindBaseRange, KindPotentialRange,
}

// Def describes an item's static properties.
type Def struct {
	Kind     Kind
	Name     string
	BoostVal float64
	Color    color.RGBA
}

// Defs holds the definition for every item kind.
var Defs = [KindCount]Def{
	KindBaseDamage:      {KindBaseDamage, "攻击磨石", 2, color.RGBA{R: 239, G: 68, B: 68, A: 255}},
	KindPotentialDamage: {KindPotentialDamage, "攻击秘卷", 3, color.RGBA{R: 185, G: 28, B: 28, A: 255}},
	KindBaseSpeed:       {KindBaseSpeed, "速射齿轮", 0.15, color.RGBA{R: 250, G: 204, B: 21, A: 255}},
	KindPotentialSpeed:  {KindPotentialSpeed, "速射秘卷", 0.2, color.RGBA{R: 202, G: 138, B: 4, A: 255}},
	KindBaseRange:       {KindBaseRange, "瞄准镜片", 12, color.RGBA{R: 59, G: 130, B: 246, A: 255}},
	KindPotentialRange:  {KindPotentialRange, "瞄准秘卷", 18, color.RGBA{R: 30, G: 64, B: 175, A: 255}},
}

// Inventory tracks how many of each item the player owns.
type Inventory struct {
	counts [KindCount]int
}

// NewInventory creates an inventory with n of each item kind.
func NewInventory(n int) *Inventory {
	inv := &Inventory{}
	for i := range inv.counts {
		inv.counts[i] = n
	}
	return inv
}

// Count returns the remaining count for item kind k.
func (inv *Inventory) Count(k Kind) int { return inv.counts[k] }

// Use consumes one item of kind k. Returns false if none remain.
func (inv *Inventory) Use(k Kind) bool {
	if inv.counts[k] <= 0 {
		return false
	}
	inv.counts[k]--
	return true
}

// TotalCount returns the sum of all item counts.
func (inv *Inventory) TotalCount() int {
	total := 0
	for _, c := range inv.counts {
		total += c
	}
	return total
}
