// list.go — BuffList 核心容器，管理活跃 buff 的增删查改和持续时间衰减。
//
// BuffList 是 buff 系统的主要数据结构，每个实体（敌人/塔）持有一个实例。
// 所有堆叠逻辑集中在 Add 方法中，Get 负责按规则聚合查询结果。
//
// 典型每帧调用顺序：
//  1. TickDoT(dt, interval) — 累积 DoT 计时器，触发时返回总伤害
//  2. Tick(dt)              — 递减所有限时 buff 的 Remaining，移除过期者
//  3. Has/Get/GetAll        — 查询当前生效的 buff 状态
package buff

// BuffList 管理一个实体上所有活跃的 buff 实例。
// active 切片预分配 8 容量，覆盖绝大多数实体的 buff 数量（通常 2~6 个）。
type BuffList struct {
	active   []Buff               // 当前活跃的所有 buff 实例（可能有同 ID 多实例，取决于 StackMode）
	rules    map[string]StackRule // buff ID → 堆叠规则，从 buff-stack.json 加载
	dotTimer float64              // DoT 伤害计时器：累积帧间隔时间，达到 dotInterval 时触发一跳
}

// NewBuffList 创建一个 BuffList，使用指定的堆叠规则集。
// 通常通过 NewDefaultBuffList() 使用全局规则创建。
func NewBuffList(rules map[string]StackRule) *BuffList {
	return &BuffList{
		active: make([]Buff, 0, 8),
		rules:  rules,
	}
}

// Add 按堆叠规则添加一个 buff。始终返回 true（当前无拒绝场景）。
//
// 各 StackMode 的行为差异：
//   - Override:              同 ID 只保留一个，后来者直接替换旧实例
//   - Strongest:             同 ID 只保留一个，仅当新 Value 更高（或相同但 Remaining 更长）时替换
//   - IndependentPerSource:  同 ID+同 Source 覆盖，不同 Source 共存（塔光环的核心机制：每塔一个实例）
//   - Additive/Multiplicative/Independent: 无条件追加，聚合逻辑在 Get 中处理
//
// 注意：slow 使用 Strongest 模式，但 slow 的 Value 越小表示减速越强（0.2 比 0.5 更慢），
// 因此 slow 的比较需要调用方在 Add 前手动处理（crowd_control.go 中先 RemoveByID 再 Add）。
func (bl *BuffList) Add(b Buff) bool {
	rule := bl.getRule(b.ID)

	switch rule.Mode {
	case Override:
		// 覆盖模式：找到同 ID 直接替换，找不到则追加
		for i := range bl.active {
			if bl.active[i].ID == b.ID {
				bl.active[i] = b
				return true
			}
		}
		bl.active = append(bl.active, b)

	case Strongest:
		// 最强模式：只保留 Value 最高的实例
		// 平局时（Value 相同）保留剩余时间更长的，避免频繁施加的短 buff 刷掉长 buff
		for i := range bl.active {
			if bl.active[i].ID == b.ID {
				if b.Value > bl.active[i].Value ||
					(b.Value == bl.active[i].Value && b.Remaining > bl.active[i].Remaining) {
					bl.active[i] = b
				}
				return true
			}
		}
		bl.active = append(bl.active, b)

	case IndependentPerSource:
		// 按来源独立模式：同 ID + 同 Source 覆盖，不同 Source 追加共存
		// 典型用例：塔光环 — 每座塔向相邻塔施加自己的光环 buff，
		// 同一座塔的光环每帧刷新（覆盖），不同塔的光环叠加（共存）
		for i := range bl.active {
			if bl.active[i].ID == b.ID && bl.active[i].Source == b.Source {
				bl.active[i] = b
				return true
			}
		}
		bl.active = append(bl.active, b)

	case Additive, Multiplicative, Independent:
		// 累加/连乘/独立模式：直接追加，聚合在 Get() 中完成
		bl.active = append(bl.active, b)

	default:
		// 未知模式：退化为 Override 行为，保证安全
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

// Has 检查是否存在指定 ID 的活跃 buff。
// 常用于存在性判断（如 Has(IDStun) 判断是否被眩晕），不关心具体数值。
func (bl *BuffList) Has(id string) bool {
	for i := range bl.active {
		if bl.active[i].ID == id {
			return true
		}
	}
	return false
}

// Get 返回指定 ID 的"有效 buff"，按堆叠规则聚合多实例。
//
// 聚合逻辑：
//   - Override / Strongest: Add 时已确保只有一个实例，直接返回
//   - Additive:            构造合成 buff，Value 为所有实例之和
//   - Multiplicative:      构造合成 buff，Value 为所有实例之积
//   - IndependentPerSource: 返回第一个实例（需要全部实例请用 GetAll）
//
// 聚合后应用 buff-stack.json 中配置的 Cap（上限）和 Floor（下限）。
// 例如 slow 的 Cap=0.8 确保减速不超过 80%，damageReduce 的 Cap=0.8 确保减伤不超过 80%。
func (bl *BuffList) Get(id string) (Buff, bool) {
	rule := bl.getRule(id)
	var found bool
	var result Buff

	for i := range bl.active {
		if bl.active[i].ID != id {
			continue
		}
		if !found {
			// 第一个匹配的实例作为基础值
			result = bl.active[i]
			found = true
			continue
		}
		// 后续实例按模式聚合
		switch rule.Mode {
		case Additive:
			result.Value += bl.active[i].Value
		case Multiplicative:
			result.Value *= bl.active[i].Value
		default:
			// Override/Strongest: Add 保证了只有一个实例，这里不会进入
			// IndependentPerSource: 只返回第一个，忽略后续（需全部请用 GetAll）
		}
	}

	if !found {
		return Buff{}, false
	}

	// 应用上限/下限钳制（Cap/Floor 在 buff-stack.json 中配置，0 表示不限制）
	if rule.Cap > 0 && result.Value > rule.Cap {
		result.Value = rule.Cap
	}
	if rule.Floor > 0 && result.Value < rule.Floor {
		result.Value = rule.Floor
	}

	return result, true
}

// GetPtr 返回指向第一个匹配 ID 的 buff 的指针，允许直接修改字段。
// 返回 nil 表示未找到。指针在下次 Add/Remove/Tick 调用前有效（切片可能重分配）。
// 主要用于出生时调整 buff 参数（如 spawner.go 中按波次缩放 berserk 阈值），
// 游戏循环中应优先使用 Get() 的值拷贝。
func (bl *BuffList) GetPtr(id string) *Buff {
	for i := range bl.active {
		if bl.active[i].ID == id {
			return &bl.active[i]
		}
	}
	return nil
}

// GetAll 返回指定 ID 的所有活跃 buff 实例（值拷贝切片）。
// 主要用于 IndependentPerSource 模式，需要遍历每个来源的 buff 时使用。
// 例如：遍历所有塔光环 buff 来聚合属性加成。
func (bl *BuffList) GetAll(id string) []Buff {
	var result []Buff
	for i := range bl.active {
		if bl.active[i].ID == id {
			result = append(result, bl.active[i])
		}
	}
	return result
}

// SumByID 返回指定 ID 所有实例的 Value 之和，并应用 Cap/Floor 钳制。
// 与 Get().Value 的区别：SumByID 始终做加法求和，不看 StackMode。
// 适用于需要手动聚合 IndependentPerSource buff 的场景（如塔属性重算时累加所有光环加成）。
func (bl *BuffList) SumByID(id string) float64 {
	var sum float64
	for i := range bl.active {
		if bl.active[i].ID == id {
			sum += bl.active[i].Value
		}
	}
	rule := bl.getRule(id)
	if rule.Cap > 0 && sum > rule.Cap {
		sum = rule.Cap
	}
	if rule.Floor != 0 && sum < rule.Floor {
		sum = rule.Floor
	}
	return sum
}

// Remove 精确移除指定 ID + Source 的 buff 实例。
// 使用 swap-compact 模式原地过滤，避免分配新切片。
// 典型用例：塔被卖掉时移除它施加给相邻塔的光环 buff。
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

// RemoveByID 移除指定 ID 的所有 buff 实例，不论 Source。
// 典型用例：crowd_control.go 在施加新 slow 前先 RemoveByID(IDSlow) 清除旧减速，
// 因为 slow 用 Strongest 模式但 Value 越小越强，需要手动替换而非依赖 Add 的比较逻辑。
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

// purgePriority 定义净化优先级（数值越高越先被移除）。
// 对敌人越有利的 buff 越优先净化，CC/DoT 等 debuff 不会被净化。
var purgePriority = map[string]int{
	IDDamageReduce:  100, // 减伤：对敌人防御收益最高
	IDControlImmune: 90,  // 控制免疫：让敌人无法被控
	IDPhaseShift:    80,  // 相位偏移：免伤机制
	IDRegen:         70,  // 回血：持续治疗
	IDHealAura:      60,  // 治疗光环：群体治疗
	IDBufferAura:    50,  // 旗手光环：群体加速
	IDSpeedUp:       40,  // 加速：提升移动速度
	IDBerserk:       30,  // 狂暴：提升速度和伤害
	IDStealth:       20,  // 隐身：不可被选中
}

// PurgeN 移除最多 n 个对敌人有利的 buff，返回实际移除数量。
// 优先移除 purgePriority 中定义的高优先级 buff。
// CC(stun/slow/root) 和 DoT(bleed/burn/poison) 等对敌人不利的 debuff 不会被净化。
func (bl *BuffList) PurgeN(n int) int {
	if n <= 0 || len(bl.active) == 0 {
		return 0
	}

	// 收集可净化的 buff 索引及其优先级
	type candidate struct {
		idx      int
		priority int
	}
	var candidates []candidate
	for i := range bl.active {
		if p, ok := purgePriority[bl.active[i].ID]; ok {
			candidates = append(candidates, candidate{idx: i, priority: p})
		}
	}

	if len(candidates) == 0 {
		return 0
	}

	// 按优先级降序排序（高优先级先移除）
	for i := 0; i < len(candidates)-1; i++ {
		for j := i + 1; j < len(candidates); j++ {
			if candidates[j].priority > candidates[i].priority {
				candidates[i], candidates[j] = candidates[j], candidates[i]
			}
		}
	}

	// 标记要移除的索引
	removeCount := n
	if removeCount > len(candidates) {
		removeCount = len(candidates)
	}
	removeSet := make(map[int]bool, removeCount)
	for i := 0; i < removeCount; i++ {
		removeSet[candidates[i].idx] = true
	}

	// swap-compact 移除
	k := 0
	for i := range bl.active {
		if removeSet[i] {
			continue
		}
		bl.active[k] = bl.active[i]
		k++
	}
	bl.active = bl.active[:k]
	return removeCount
}

// ClearByCategory 移除所有属于指定分类的 buff。
// 典型用例：净化技能清除所有 CC（ClearByCategory(CatCC)），
// 或控制免疫触发时清除所有控制类 buff。
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

// Clear 清除所有 buff。保留底层切片容量以复用内存（[:0] 而非 nil）。
// 用于实体回收到对象池时重置状态。
func (bl *BuffList) Clear() {
	bl.active = bl.active[:0]
}

// Active 返回所有活跃 buff 的快照（值拷贝），供 HUD 展示用。
// 返回的是独立切片，修改不影响 BuffList 内部状态。
func (bl *BuffList) Active() []Buff {
	result := make([]Buff, len(bl.active))
	copy(result, bl.active)
	return result
}

// Count 返回当前活跃 buff 的总数量。
func (bl *BuffList) Count() int {
	return len(bl.active)
}

// Tick 递减所有限时 buff 的 Remaining，并移除过期（Remaining<=0）的实例。
// 永久 buff（Duration < 0）不受影响，不会被自然移除。
//
// 重要：必须在 TickDoT() 之后调用。
// 如果先 Tick 再 TickDoT，即将过期的 DoT 会在 Tick 中被移除，
// 导致 TickDoT 统计不到它的最后一跳伤害。
func (bl *BuffList) Tick(dt float64) {
	n := 0
	for i := range bl.active {
		b := &bl.active[i]
		if b.Duration >= 0 { // 限时 buff：递减剩余时间
			b.Remaining -= dt
			if b.Remaining <= 0 {
				continue // 已过期 — 跳过（等效移除）
			}
		}
		// 永久 buff 或尚未过期的 buff：保留
		bl.active[n] = bl.active[i]
		n++
	}
	bl.active = bl.active[:n]
}

// getRule 查找指定 buff ID 的堆叠规则。
// 若 buff-stack.json 中未配置该 ID，默认退化为 Override 模式（最安全的兜底策略）。
func (bl *BuffList) getRule(id string) StackRule {
	if r, ok := bl.rules[id]; ok {
		return r
	}
	return StackRule{Mode: Override}
}
