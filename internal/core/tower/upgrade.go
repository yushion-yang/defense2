// upgrade.go — 塔升级系统。
// 建塔时立即解锁第 1 个能力位（攻击模式），之后每 2 波再解锁 1 个。
// 玩家从 3 个候选能力中选 1 个。可保留不选。
package tower

import (
	"math/rand"
	"sort"

	"defense2/internal/config"
)

// MaxAbilitySlots 最大能力槽位数（6 大类别各一个）。
const MaxAbilitySlots = config.AbilityCatCount

// WavesPerUnlock 返回每隔多少波解锁 1 个能力位（从 balance.json 实时读取）。
func WavesPerUnlock() int { return config.GlobalBalance().Tower.WavesPerUnlock }

// ChoicesPerUnlock 返回每次解锁提供的候选能力数（从 balance.json 实时读取）。
func ChoicesPerUnlock() int { return config.GlobalBalance().Tower.ChoicesPerUnlock }

// ── 能力位解锁 ──

// UnlockedSlots 根据已完成的波次数计算已解锁的能力位数。
// 第一个槽位（攻击模式）建塔时立即解锁，后续每 WavesPerUnlock 波再解锁一个。
func UnlockedSlots(wavesCleared int) int {
	n := 1 + wavesCleared/WavesPerUnlock()
	if n > MaxAbilitySlots {
		n = MaxAbilitySlots
	}
	return n
}

// PendingSlots 返回塔有多少个已解锁但未选择能力的槽位。
func (t *Tower) PendingSlots(wavesCleared int) int {
	unlocked := UnlockedSlots(wavesCleared)
	used := 0
	for _, a := range t.AbilitySlots {
		if a != "" {
			used++
		}
	}
	pending := unlocked - used
	if pending < 0 {
		pending = 0
	}
	return pending
}

// HasPendingUpgrade 返回塔是否有待选择的能力。
func (t *Tower) HasPendingUpgrade(wavesCleared int) bool {
	return t.PendingSlots(wavesCleared) > 0
}

// ── 选项缓存 ──

// RollAndCachePendingChoices 为塔 roll 所有已解锁但未选择能力位的 3 选项并缓存。
// 已有缓存的位不会重新 roll。建塔时和新波次解锁时调用。
func RollAndCachePendingChoices(t *Tower, wavesCleared int) {
	if t.PendingChoices == nil {
		t.PendingChoices = make(map[int][]config.AbilityDef)
	}
	unlocked := UnlockedSlots(wavesCleared)
	for i := 0; i < unlocked && i < len(t.UnlockOrder); i++ {
		cat := t.UnlockOrder[i]
		if t.AbilitySlots[cat] != "" {
			continue // 已选择
		}
		if _, exists := t.PendingChoices[cat]; exists {
			continue // 已缓存
		}
		choices := rollChoicesForCategory(cat, ChoicesPerUnlock())
		if len(choices) > 0 {
			t.PendingChoices[cat] = choices
		}
	}
}

// ClearPendingChoice 选择能力后清除该类别的缓存选项。
func ClearPendingChoice(t *Tower, cat int) {
	if t.PendingChoices != nil {
		delete(t.PendingChoices, cat)
	}
}

// NextPendingCategory 返回下一个有缓存选项待选的类别（按 UnlockOrder 顺序）。
// 返回 -1 表示没有待选。
func NextPendingCategory(t *Tower) int {
	if t.PendingChoices == nil {
		return -1
	}
	for _, cat := range t.UnlockOrder {
		if t.AbilitySlots[cat] == "" {
			if _, exists := t.PendingChoices[cat]; exists {
				return cat
			}
		}
	}
	return -1
}

// PendingCount 返回有缓存选项待选的能力位数量。
func PendingCount(t *Tower) int {
	if t.PendingChoices == nil {
		return 0
	}
	count := 0
	for cat, choices := range t.PendingChoices {
		if t.AbilitySlots[cat] == "" && len(choices) > 0 {
			count++
		}
	}
	return count
}

// rollChoicesForCategory 为指定类别 roll N 个候选能力。
func rollChoicesForCategory(cat, count int) []config.AbilityDef {
	pool := AbilitiesForCategory(cat)
	if len(pool) == 0 {
		return nil
	}
	rand.Shuffle(len(pool), func(i, j int) {
		pool[i], pool[j] = pool[j], pool[i]
	})
	if count > len(pool) {
		count = len(pool)
	}
	result := make([]config.AbilityDef, count)
	for i := 0; i < count; i++ {
		result[i] = *pool[i]
	}
	return result
}

// ── 能力选择 ──

// AddAbility 为塔添加一个能力到对应类别的槽位。
// 返回 false 如果该类别已被占用或类别无效。
func (t *Tower) AddAbility(abilityType string) bool {
	table := config.GlobalAbilityTable()
	if table == nil {
		return false
	}
	def, ok := table[abilityType]
	if !ok {
		return false
	}
	cat := def.CategoryIndex()
	if cat < 0 || cat >= config.AbilityCatCount {
		return false
	}
	if t.AbilitySlots[cat] != "" {
		return false // 该类别已有能力
	}
	t.AbilitySlots[cat] = abilityType
	t.Abilities = t.AllAbilities()
	t.Level++

	// 攻击模式 → 更新 AttackStyleID + 外观
	if cat == config.AbilityCatAttack {
		t.AttackStyleID = t.ResolveAttackStyle()
		t.SpriteKey = AbilitySpriteKey(abilityType)
		t.Label = SpriteLabelFor(t.SpriteKey)
	}

	// 强化 → 提升基础属性
	if abilityType == "enhance" {
		applyEnhance(t, def)
	}

	return true
}

// applyEnhance 强化能力：一次性提升塔的基础和潜力属性。
// 固定使用 base 值（不受强度影响），确保行为与描述一致。
func applyEnhance(t *Tower, def *config.AbilityDef) {
	boost := def.Base // 固定 20%，不走 CalcScale(str)

	t.BaseDamage *= 1 + boost
	t.PotentialDamage *= 1 + boost
	t.BaseSpeed *= 1 + boost
	t.PotentialSpeed *= 1 + boost
	t.BaseRange *= 1 + boost/2 // 射程提升减半避免过强
	t.PotentialRange *= 1 + boost/2
	t.RecalcStats()
}

// NextUnlockCategory 返回下一个应该解锁的类别（按 UnlockOrder 中首个空槽）。
// 如果全部已满，返回 -1。
func (t *Tower) NextUnlockCategory() int {
	for _, cat := range t.UnlockOrder {
		if t.AbilitySlots[cat] == "" {
			return cat
		}
	}
	return -1
}

// ── 查询 ──

// AllChoicesForCategory 返回指定类别的全部能力（不随机不截断），用于测试模式。
func AllChoicesForCategory(cat int) []config.AbilityDef {
	pool := AbilitiesForCategory(cat)
	result := make([]config.AbilityDef, len(pool))
	for i, p := range pool {
		result[i] = *p
	}
	return result
}

// disabledAbilities 禁用能力黑名单。
// 这些能力的实现尚未完成（TODO: 待 BuffList 集成后启用），
// 玩家选择后无任何效果，因此从候选池中排除。
var disabledAbilities = map[string]bool{
	"killUpgrade":   true, // TODO: integrate kill-based scaling via BuffList
	"waveScale":     true, // TODO: integrate wave-scale boosts via BuffList
	"neighborBoost": true, // TODO: integrate neighbor boost via BuffList
	"elementSwitch": true, // TODO: integrate element boosts via BuffList
	"periodicCast":  true, // case 2 (buffAoe) is a no-op; disable until all modes work
}

// AbilitiesForCategory 返回指定类别中可选择的能力列表，按 Type 字母序排列。
// 已禁用的能力（disabledAbilities）不会出现在候选池中。
func AbilitiesForCategory(category int) []*config.AbilityDef {
	table := config.GlobalAbilityTable()
	if table == nil {
		return nil
	}
	var result []*config.AbilityDef
	for _, def := range table {
		if def.CategoryIndex() == category && !disabledAbilities[def.Type] {
			result = append(result, def)
		}
	}
	sort.Slice(result, func(i, j int) bool {
		return result[i].Type < result[j].Type
	})
	return result
}

// CategoryName 返回类别的中文名称。
func CategoryName(cat int) string {
	switch cat {
	case config.AbilityCatAttack:
		return "攻击模式"
	case config.AbilityCatCC:
		return "控制效果"
	case config.AbilityCatDamage:
		return "命中加伤"
	case config.AbilityCatBuff:
		return "增益光环"
	case config.AbilityCatDoT:
		return "持续伤害"
	case config.AbilityCatZone:
		return "范围效果"
	default:
		return "未知"
	}
}
