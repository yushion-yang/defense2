// upgrade.go — 塔能力槽解锁与选择系统。
//
// 本文件是塔系统中最复杂的部分，管理能力的解锁时机、候选生成、玩家选择。
//
// ═══ 5 个核心交互字段 ═══
//
//	UnlockOrder [6]int
//	  能力类别的解锁顺序（建塔时由 RollUnlockOrder 随机生成）。
//	  [0] 始终是攻击模式（AbilityCatAttack），后 5 个类别随机排列。
//	  决定了"第 N 个解锁的是哪个类别"。
//
//	AbilitySlots [6]string
//	  每个类别已选择的能力（""=未选择）。
//	  AbilitySlots[cat] = "scatter" 表示攻击类别已选择散射能力。
//	  一旦选择不可更改（没有换能力机制）。
//
//	PendingChoices map[int][]AbilityDef
//	  待选的候选能力缓存（key=类别索引，value=3 个候选）。
//	  生命周期：解锁时 roll → 玩家选择后 delete → 新波次解锁时追加。
//	  已有缓存的类别不会重新 roll（保证玩家看到的选项稳定）。
//
//	PaidUnlocks int
//	  通过花钱购买解锁的累计次数（用于索引 UpgradeCosts 计算下次费用）。
//	  与波次解锁独立——花钱解锁不受 wavesCleared 限制。
//
//	wavesCleared int（外部传入参数）
//	  全局已完成波次数。解锁公式：解锁数 = 1 + wavesCleared / WavesPerUnlock。
//	  第 0 波建塔时解锁 1 个（攻击模式），每 2 波再解锁 1 个，最多 6 个。
//
// ═══ 解锁流程 ═══
//
//	波次解锁（自动）：
//	  1. 新波次开始 → RollAndCachePendingChoices(t, wavesCleared)
//	  2. 计算 UnlockedSlots(wavesCleared) 得到应解锁数
//	  3. 按 UnlockOrder 顺序，为每个已解锁但未选择且无缓存的类别 roll 3 个候选
//	  4. UI 显示待选提示 → 玩家点击选择 → AddAbility + ClearPendingChoice
//
//	付费解锁（主动）：
//	  1. 玩家点击升级按钮 → UnlockNextSlot(t)
//	  2. 找到 UnlockOrder 中首个空且无待选的类别 → roll 候选 → 加入 PendingChoices
//	  3. PaidUnlocks++ 由调用方处理（stage 层）
//
// 与其他文件的关系：
//   - tower.go: AbilitySlots/UnlockOrder/PendingChoices 字段定义
//   - randomize.go: RollUnlockOrder 生成解锁顺序
//   - ability.go: Registry 查找能力实例
//   - config/: AbilityDef/AbilityTable 提供能力元数据
package tower

import (
	"cmp"
	"math/rand"
	"slices"

	"defense2/internal/config"
	"defense2/internal/i18n"
)

// MaxAbilitySlots 最大能力槽位数（6 大类别各一个）。
const MaxAbilitySlots = config.AbilityCatCount

// WavesPerUnlock 返回每隔多少波解锁 1 个能力位（从 balance.json 实时读取）。
func WavesPerUnlock() int { return config.GlobalBalance().Tower.WavesPerUnlock }

// ChoicesPerUnlock 返回每次解锁提供的候选能力数（从 balance.json 实时读取）。
func ChoicesPerUnlock() int { return config.GlobalBalance().Tower.ChoicesPerUnlock }

// ── 能力位解锁 ──

// UnlockedSlots 根据已完成的波次数计算已解锁的能力位数。
// 公式：n = 1 + wavesCleared / WavesPerUnlock（整除），上限 MaxAbilitySlots=6。
// 示例（WavesPerUnlock=2）：波 0→1 个, 波 2→2 个, 波 4→3 个, ..., 波 10→6 个（满）。
func UnlockedSlots(wavesCleared int) int {
	n := 1 + wavesCleared/WavesPerUnlock()
	if n > MaxAbilitySlots {
		n = MaxAbilitySlots
	}
	return n
}

// PendingSlots 返回塔有多少个已解锁但未选择能力的槽位。
// 计算方式：已解锁数 - 已选择数 = 待选数。
// 注意：这里只统计波次解锁的空位，不包含花钱解锁的（花钱解锁直接进 PendingChoices）。
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

// HasPendingUpgrade 返回塔是否有待选择的能力（用于 UI 显示升级提示图标）。
// 仅当 PendingChoices 中有实际候选时才返回 true。
// 空槽但没有可选能力（该类别能力池耗尽）不显示提示。
func (t *Tower) HasPendingUpgrade() bool {
	return PendingCount(t) > 0
}

// NextUpgradeCost 返回塔下一次付费解锁能力槽的金币费用。
// 以 PaidUnlocks（累计付费次数）为索引查 TowerDef.UpgradeCosts 数组。
// 安全处理：索引超出数组时复用最后一档费用（这样新增能力类别不需要改 JSON 配置）。
func NextUpgradeCost(t *Tower, def TowerDef) int {
	if len(def.UpgradeCosts) == 0 {
		return 0
	}
	idx := t.PaidUnlocks
	if idx >= len(def.UpgradeCosts) {
		idx = len(def.UpgradeCosts) - 1
	}
	return def.UpgradeCosts[idx]
}

// ── 选项缓存 ──

// UnlockNextSlot 付费解锁下一个空能力槽位并 roll 候选选项。
// 不受 wavesCleared 限制（花钱就能提前解锁）。
// 返回解锁的类别索引（0-5），全部槽位已满或已有待选时返回 -1。
// 调用方（stage 层）负责扣金币和递增 PaidUnlocks。
func UnlockNextSlot(t *Tower) int {
	cat := t.NextUnlockCategory()
	if cat < 0 {
		return -1
	}
	if t.PendingChoices == nil {
		t.PendingChoices = make(map[int][]config.AbilityDef)
	}
	if _, exists := t.PendingChoices[cat]; !exists {
		choices := rollChoicesForCategory(cat, ChoicesPerUnlock())
		if len(choices) > 0 {
			t.PendingChoices[cat] = choices
		}
	}
	return cat
}

// CanUnlockMore 返回塔是否还有空能力槽位可以解锁（不要求先选完待选能力）。
func CanUnlockMore(t *Tower) bool {
	return t.NextUnlockCategory() >= 0
}

// RollAndCachePendingChoices 为塔 roll 所有已解锁但未选择的能力位的候选选项并缓存。
//
// 调用时机：建塔时（初始化第 1 个攻击模式候选）+ 每波结束检查新解锁。
// 关键行为：
//   - 已有缓存的类别不会重新 roll（保证玩家看到的选项稳定，不会因为切换 UI 而变化）
//   - 已选择能力的类别跳过
//   - 按 UnlockOrder 顺序处理，只处理 wavesCleared 已解锁的前 N 个类别
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
// 从该类别的全部可用能力中随机洗牌，取前 count 个。
// 如果该类别可用能力不足 count 个，则全部返回（不会重复）。
// 返回值是 AbilityDef 的值拷贝（非指针），确保缓存独立不受全局表影响。
func rollChoicesForCategory(cat, count int) []config.AbilityDef {
	pool := AbilitiesForCategory(cat) // 获取该类别所有可用能力（已排除禁用的）
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
		result[i] = *pool[i] // 值拷贝，PendingChoices 缓存持有独立副本
	}
	return result
}

// ── 能力选择 ──

// AddAbility 为塔添加一个能力到对应类别的槽位（玩家选择确认后调用）。
//
// 流程：
//  1. 从全局能力表查找 abilityType 的元数据（获取所属类别）
//  2. 检查该类别槽位是否为空（已占用则拒绝）
//  3. 写入 AbilitySlots[cat] 并同步更新 Abilities 列表
//  4. Level++（塔等级随能力数增长）
//  5. 特殊处理：
//     - 攻击类别(cat=0)：更新 AttackStyleID + SpriteKey + Label（塔外观变形）
//     - enhance 能力：一次性提升 Base/Potential 属性（applyEnhance）
//
// 返回 false 表示添加失败（类别无效/已占用/能力表中不存在）。
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

	// 攻击类别(slot[0])是特殊的：选择攻击能力会改变塔的攻击方式和外观（变形）
	if cat == config.AbilityCatAttack {
		t.AttackStyleID = t.ResolveAttackStyle()
		t.SpriteKey = AbilitySpriteKey(abilityType)
		t.Label = SpriteLabelFor(t.SpriteKey)
	}

	// enhance 是特殊的非攻击能力：一次性永久提升塔的 Base 和 Potential 属性
	if abilityType == AbilityEnhance {
		applyEnhance(t, def)
	}

	return true
}

// applyEnhance 强化能力：一次性永久提升塔的 Base 和 Potential 属性。
// 使用 AbilityDef 的 Param/Param2 固定参数（不受强度影响），确保 UI 描述与实际效果一致。
// 伤害和攻速使用 Param（如 0.2 = +20%），射程使用 Param2（如 0.1 = +10%）。
// 乘法增幅（非加法），所以多次 enhance 是复利效果（1.2 × 1.2 = 1.44）。
func applyEnhance(t *Tower, def *config.AbilityDef) {
	boost := def.Param       // 伤害/攻速增幅比例（如 0.2 = +20%）
	rangeBoost := def.Param2 // 射程增幅比例（独立配置，通常比伤害增幅小）
	if rangeBoost <= 0 {
		rangeBoost = boost / 2 // 兼容旧配置：Param2 未设置时回退到 Param 的一半
	}

	t.BaseDamage *= 1 + boost
	t.PotentialDamage *= 1 + boost
	t.BaseSpeed *= 1 + boost
	t.PotentialSpeed *= 1 + boost
	t.BaseRange *= 1 + rangeBoost
	t.PotentialRange *= 1 + rangeBoost
	t.RecalcStats()
}

// NextUnlockCategory 返回下一个应该解锁的类别索引（按 UnlockOrder 顺序，首个空且无待选的槽）。
// 跳过两种槽：已选择能力的（AbilitySlots[cat] != ""）和已有 PendingChoices 的（已解锁待选中）。
// 这保证了连续两次 UnlockNextSlot 会解锁不同的类别。
// 全部类别都已占用或有待选时返回 -1。
func (t *Tower) NextUnlockCategory() int {
	for _, cat := range t.UnlockOrder {
		if t.AbilitySlots[cat] != "" {
			continue // 已选择
		}
		if t.PendingChoices != nil {
			if _, exists := t.PendingChoices[cat]; exists {
				continue // 已解锁待选
			}
		}
		return cat
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

// disabledAbilities 禁用能力黑名单（从候选池中排除）。
// 这些能力的代码已写但效果不完整，玩家选择后无法正常工作。
// 从候选池排除比删除代码更安全——后续只需从此 map 移除即可启用。
var disabledAbilities = map[string]bool{
	AbilityElementSwitch: true, // 元素增强计划后续通过 BuffList 集成
	AbilityPeriodicCast:  true, // case 2 (buffAoe) is a no-op; disable until all modes work
}

// AbilitiesForCategory 返回指定类别中所有可用的能力定义，按 Type 字母序排列。
// 过滤掉 disabledAbilities 黑名单中的能力。排序保证测试结果稳定。
// rollChoicesForCategory 在此基础上随机洗牌取 N 个。
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
	slices.SortFunc(result, func(a, b *config.AbilityDef) int {
		return cmp.Compare(a.Type, b.Type)
	})
	return result
}

// CategoryName 返回类别的本地化名称。
func CategoryName(cat int) string {
	switch cat {
	case config.AbilityCatAttack:
		return i18n.T("tower.category.attack")
	case config.AbilityCatCC:
		return i18n.T("tower.category.cc")
	case config.AbilityCatDamage:
		return i18n.T("tower.category.damage")
	case config.AbilityCatBuff:
		return i18n.T("tower.category.buff")
	case config.AbilityCatDoT:
		return i18n.T("tower.category.dot")
	case config.AbilityCatZone:
		return i18n.T("tower.category.zone")
	default:
		return i18n.T("tower.category.unknown")
	}
}
