// item.go — 道具系统。
//
// 道具是塔防游戏中的一次性消耗品，玩家将道具拖放到塔上以永久增强属性。
// 设计核心：
//   - 6 种道具 = 3 属性(Damage/Speed/Range) × 2 类型(Base/Potential)
//   - Base 型：直接加属性基础值，立即见效
//   - Potential 型：加属性潜力值，需要高 Strength 才能发挥（attr = Base + Potential × Strength/100）
//   - 所有数值从 balance.json items 区段读取，不硬编码
//
// 使用流程：ActionBar [道具]按钮 → 选择道具 → 拖拽到目标塔 → ApplyItem 修改塔属性。
package item

import (
	"fmt"
	"image/color"
	"strings"

	"defense2/internal/config"
	"defense2/internal/core/tower"
)

// Kind 道具类型枚举，对应 balance.json items[].kind 字符串。
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

// AllKinds 所有有效道具类型的枚举数组，用于遍历（不含 KindCount 哨兵值）。
var AllKinds = [...]Kind{
	KindBaseDamage, KindPotentialDamage,
	KindBaseSpeed, KindPotentialSpeed,
	KindBaseRange, KindPotentialRange,
}

// Def 道具的静态定义（从 balance.json 加载后不可变）。
type Def struct {
	Kind       Kind       // 道具类型
	Name       string     // 显示名称（如 "攻击磨石"）
	Desc       string     // 描述文字（已将 {v} 替换为实际数值）
	Icon       string     // 图标名（用于 IconManager.Get 获取 SVG 精灵）
	BoostVal   float64    // 属性加成数值
	Color      color.RGBA // UI 显示颜色（红=攻击、黄=速度、蓝=射程）
	StartCount int        // 每局初始持有数量
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

// Defs 全局道具定义表（按 Kind 索引），从 balance.json items 区段加载。
// 包初始化时一次性构建，运行时只读。
var Defs = initDefs()

// initDefs 从 balance.json 构建道具定义表。
// 将 JSON 配置中的字符串 kind 映射为枚举，合并颜色和图标等硬编码元数据。
// {v} 占位符处理：整数显示为整数（如 "+5"），小数保留两位（如 "+0.10"）。
func initDefs() [KindCount]Def {
	var defs [KindCount]Def
	items := config.GlobalBalance().Items
	for _, it := range items {
		k, ok := kindFromString[it.Kind]
		if !ok {
			continue // 跳过未知类型（防御性处理，允许 JSON 新增未实现道具）
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

// Inventory 玩家道具背包，追踪每种道具的剩余数量。
// 使用固定大小数组（6 种）而非 map，避免热路径堆分配。
type Inventory struct {
	counts [KindCount]int
}

// NewInventory 创建每种道具各 n 个的背包（主要用于测试）。
func NewInventory(n int) *Inventory {
	inv := &Inventory{}
	for i := range inv.counts {
		inv.counts[i] = n
	}
	return inv
}

// NewInventoryFromConfig 从 balance.json 的 startCount 创建背包（正式游戏用）。
func NewInventoryFromConfig() *Inventory {
	inv := &Inventory{}
	for _, k := range AllKinds {
		inv.counts[k] = Defs[k].StartCount
	}
	return inv
}

// Add 增加一个道具（用于事件奖励等场景）。
func (inv *Inventory) Add(k Kind) { inv.counts[k]++ }

// Count 返回指定道具的剩余数量（用于 UI 灰显判断）。
func (inv *Inventory) Count(k Kind) int { return inv.counts[k] }

// Use 消耗一个道具。数量不足返回 false，调用方应检查并提示玩家。
func (inv *Inventory) Use(k Kind) bool {
	if inv.counts[k] <= 0 {
		return false
	}
	inv.counts[k]--
	return true
}

// TotalCount 返回所有道具总数（用于判断背包是否为空）。
func (inv *Inventory) TotalCount() int {
	total := 0
	for _, c := range inv.counts {
		total += c
	}
	return total
}

// ApplyItem 将道具效果应用到目标塔上。
// 直接修改塔的 Base/Potential 原始属性，然后调用 RecalcStats 重算派生属性。
// 效果是永久的——不是 buff，不会过期，不可撤销。
// 这也是道具系统唯一修改塔属性的入口（遵循单一职责原则）。
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
