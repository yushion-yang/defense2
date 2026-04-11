package item

import (
	"fmt"
	"image/color"
	"strings"

	"defense2/internal/config"
	"defense2/internal/core/tower"
)

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
	Kind       Kind
	Name       string
	Desc       string // 描述文字（已替换 {v} 占位符）
	Icon       string // 图标名（用于 IconManager.Get）
	BoostVal   float64
	Color      color.RGBA
	StartCount int
}

// itemColors 每种道具的显示颜色（固定，不受 balance 影响）。
var itemColors = [KindCount]color.RGBA{
	KindBaseDamage:      {R: 239, G: 68, B: 68, A: 255},
	KindPotentialDamage: {R: 185, G: 28, B: 28, A: 255},
	KindBaseSpeed:       {R: 250, G: 204, B: 21, A: 255},
	KindPotentialSpeed:  {R: 202, G: 138, B: 4, A: 255},
	KindBaseRange:       {R: 59, G: 130, B: 246, A: 255},
	KindPotentialRange:  {R: 30, G: 64, B: 175, A: 255},
}

// kindFromString 将 balance.json 中的 kind 字符串映射为 Kind 枚举。
var kindFromString = map[string]Kind{
	"baseDamage":      KindBaseDamage,
	"potentialDamage": KindPotentialDamage,
	"baseSpeed":       KindBaseSpeed,
	"potentialSpeed":  KindPotentialSpeed,
	"baseRange":       KindBaseRange,
	"potentialRange":  KindPotentialRange,
}

// itemIcons 每种道具对应的图标名。
var itemIcons = [KindCount]string{
	KindBaseDamage:      "item-stone",
	KindPotentialDamage: "item-scroll-damage",
	KindBaseSpeed:       "item-gear",
	KindPotentialSpeed:  "item-scroll-speed",
	KindBaseRange:       "item-lens",
	KindPotentialRange:  "item-scroll-range",
}

// Defs holds the definition for every item kind (populated from balance.json).
var Defs = initDefs()

// initDefs 从 balance.json 构建道具定义表。
func initDefs() [KindCount]Def {
	var defs [KindCount]Def
	items := config.GlobalBalance().Items
	for _, it := range items {
		k, ok := kindFromString[it.Kind]
		if !ok {
			continue
		}
		// 将 {v} 占位符替换为实际数值
		desc := it.Description
		if it.Boost == float64(int(it.Boost)) {
			desc = strings.ReplaceAll(desc, "{v}", fmt.Sprintf("%d", int(it.Boost)))
		} else {
			desc = strings.ReplaceAll(desc, "{v}", fmt.Sprintf("%.2f", it.Boost))
		}
		defs[k] = Def{
			Kind:       k,
			Name:       it.Label,
			Desc:       desc,
			Icon:       itemIcons[k],
			BoostVal:   it.Boost,
			Color:      itemColors[k],
			StartCount: it.StartCount,
		}
	}
	return defs
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

// NewInventoryFromConfig creates an inventory using startCount from balance.json.
func NewInventoryFromConfig() *Inventory {
	inv := &Inventory{}
	for _, k := range AllKinds {
		inv.counts[k] = Defs[k].StartCount
	}
	return inv
}

// Add grants one item of kind k.
func (inv *Inventory) Add(k Kind) { inv.counts[k]++ }

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
