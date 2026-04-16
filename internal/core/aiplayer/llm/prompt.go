// prompt.go -- 局势摘要 -> LLM prompt 构建。
//
// 将游戏状态快照转化为自然语言描述，作为 LLM 的输入。
// 包含短期记忆系统（最近 3 次交互），让 LLM 产生连贯对话。
// 关联：由 aiplayer.go 的 buildSituation() 填充 Situation，
//       connector.go 的 callAPI() 使用 BuildPrompt() 生成 prompt。
package llm

import (
	"fmt"
	"strings"
)

// ── 塔/敌人/道具的详细描述 ──

// TowerDesc 单座塔的详细描述（用于构建精确局势 prompt）。
type TowerDesc struct {
	Position  string   // "entrance" / "middle" / "exit"
	Abilities []string // 已有能力列表
	Damage    float64
	Range     float64
	Kills     int
	Style     string // 攻击方式（projectile/scatter/spinAoe 等）
	Strength  int
}

// EnemyGroup 敌人分组描述（按类型聚合）。
type EnemyGroup struct {
	Type  string  // "tank" / "runner" / "swarm" / "boss" / "normal"
	Count int
	AvgHP float64
}

// Situation 游戏局势摘要（纯值结构体）。
type Situation struct {
	Wave, MaxWaves    int
	AIGold, HumanGold int
	Lives, MaxLives   int
	AITowerCount      int
	HumanTowerCount   int
	LastAction        string // "built tower" / "upgraded" / "idle"
	ThreatLevel       string // "low" / "medium" / "high"

	// ── Phase 2-3 新增字段 ──
	PersonalityDesc string // 个性描述（如 "你性格激进，喜欢DPS塔"）
	Mood            string // 当前情绪（如 "excited" / "worried" / "calm"）

	// ── 协作+策略 字段 ──
	CoopDesc       string // 协作分析描述（如 "队友有控制，全队缺输出"）
	AdvicePriority string // 策略建议方向（如 "build_cc" / "build_dps"）

	// ── LLM 深度知识字段 ──
	AITowers         []TowerDesc  // AI 的塔详细描述
	EnemyGroups      []EnemyGroup // 敌人分组
	AvailableItems   []string     // 可用道具名称列表
	PendingAbilities int          // 有几座塔在等待能力选择
	NextWaveBoss     bool         // 下一波是否有 Boss
	CanBuild         bool         // 金币够造塔吗
	CanUpgrade       bool         // 金币够升级吗
}

// ── 短期记忆系统 ──

const memoryCapacity = 3 // 最多保留 3 条记忆

// MemoryEntry 记忆条目：一次事件和对应的 LLM 输出。
type MemoryEntry struct {
	Event    string // "built tower at row 5" / "player leaked" 等
	Response string // 上次 LLM 的输出
}

// Memory 短期记忆（环形缓冲区，最多 3 条）。
type Memory struct {
	entries []MemoryEntry
}

// NewMemory 创建空记忆。
func NewMemory() *Memory {
	return &Memory{
		entries: make([]MemoryEntry, 0, memoryCapacity),
	}
}

// Add 添加一条记忆。超过容量时丢弃最旧的。
func (m *Memory) Add(event, response string) {
	if len(m.entries) >= memoryCapacity {
		// 移除最旧的条目
		m.entries = m.entries[1:]
	}
	m.entries = append(m.entries, MemoryEntry{Event: event, Response: response})
}

// Entries 返回所有记忆条目（只读副本）。
func (m *Memory) Entries() []MemoryEntry {
	copied := make([]MemoryEntry, len(m.entries))
	copy(copied, m.entries)
	return copied
}

// Len 返回当前记忆条目数量。
func (m *Memory) Len() int {
	return len(m.entries)
}

// Clear 清空记忆。
func (m *Memory) Clear() {
	m.entries = m.entries[:0]
}

// BuildPrompt 根据局势生成 LLM prompt。
// 包含个性描述、情绪状态、短期记忆、详细塔/敌人信息和具体输出要求。
func BuildPrompt(s Situation, memory *Memory) string {
	var sb strings.Builder

	// ── 角色设定 ──
	sb.WriteString("你是塔防游戏中的 AI 玩家。你正在和一个人类队友一起守关。\n")

	// 个性描述
	if s.PersonalityDesc != "" {
		sb.WriteString(s.PersonalityDesc)
		sb.WriteString("\n")
	}

	// 情绪状态
	if s.Mood != "" {
		sb.WriteString(fmt.Sprintf("你现在的心情: %s\n", s.Mood))
	}

	// ── 短期记忆 ──
	if memory != nil && memory.Len() > 0 {
		sb.WriteString("\n你之前说过:\n")
		for _, e := range memory.Entries() {
			if e.Response != "" {
				sb.WriteString(fmt.Sprintf("- [%s时] \"%s\"\n", e.Event, e.Response))
			}
		}
	}

	// ── 当前局势 ──
	sb.WriteString(fmt.Sprintf(`
当前局势:
- 第 %d/%d 波
- 你的金币: %d, 队友金币: %d
- 共享生命: %d/%d
- 你有 %d 座塔, 队友有 %d 座塔
- 你刚刚: %s
- 威胁等级: %s
`,
		s.Wave, s.MaxWaves,
		s.AIGold, s.HumanGold,
		s.Lives, s.MaxLives,
		s.AITowerCount, s.HumanTowerCount,
		s.LastAction,
		s.ThreatLevel,
	))

	// ── 详细塔信息 ──
	if len(s.AITowers) > 0 {
		sb.WriteString("- 我的塔:\n")
		for i, t := range s.AITowers {
			abStr := "无能力"
			if len(t.Abilities) > 0 {
				abStr = strings.Join(t.Abilities, "+")
			}
			sb.WriteString(fmt.Sprintf("  塔%d(%s,%s,伤害%.0f,射程%.0f,%d击杀)\n",
				i+1, t.Position, abStr, t.Damage, t.Range, t.Kills))
		}
	}

	// ── 敌人组成 ──
	if len(s.EnemyGroups) > 0 {
		var parts []string
		for _, g := range s.EnemyGroups {
			parts = append(parts, fmt.Sprintf("%d%s(均HP%.0f)", g.Count, g.Type, g.AvgHP))
		}
		sb.WriteString(fmt.Sprintf("- 敌人: %s\n", strings.Join(parts, "+")))
	}

	// ── 待处理事项 ──
	if s.PendingAbilities > 0 {
		sb.WriteString(fmt.Sprintf("- %d座塔待选能力\n", s.PendingAbilities))
	}
	if s.NextWaveBoss {
		sb.WriteString("- 下一波有Boss!\n")
	}

	// 协作和策略上下文
	if s.CoopDesc != "" {
		sb.WriteString(fmt.Sprintf("- 队友配置: %s\n", s.CoopDesc))
	}
	if s.AdvicePriority != "" {
		sb.WriteString(fmt.Sprintf("- 建议策略: %s\n", s.AdvicePriority))
	}

	// ── 输出要求 ──
	sb.WriteString(`用一句话（15字以内）表达你当前的想法或对局势的反应。
要求：口语化中文、有个性、偶尔吐槽。不要用emoji。只输出这一句话，不要任何解释。`)

	return sb.String()
}

// BuildStrategicPrompt 构建结构化战略决策 prompt。
// 与 BuildPrompt 不同，此 prompt 要求 LLM 返回严格 JSON 操作数组，
// 支持多步行动计划（sell/build/upgrade/ability/item/wait）。
//
// 输入：
//   - s: 局势摘要（含详细塔/敌人/道具数据）
//   - advicePriority: 局势感知的建议方向（如 "build_cc"）
//   - coopDesc: 协作分析描述
//
// 输出 prompt 要求 LLM 返回格式：
//
//	{"actions":[{"type":"build|upgrade|sell|ability|item|wait","position":"entrance|middle|exit","priority":"cc|dps|aoe","target":"strongest|weakest","itemType":"BaseDamage|Speed|Range","reason":"一句话"}],"say":"弹幕(15字内)"}
func BuildStrategicPrompt(s Situation, advicePriority, coopDesc string) string {
	var sb strings.Builder

	sb.WriteString("根据局势制定多步行动计划。\n\n")

	// ── 精确局势 ──
	sb.WriteString(fmt.Sprintf("局势：第%d/%d波 金币:%d 生命:%d/%d 威胁:%s\n",
		s.Wave, s.MaxWaves, s.AIGold, s.Lives, s.MaxLives, s.ThreatLevel))

	// ── 详细塔描述 ──
	if len(s.AITowers) > 0 {
		sb.WriteString("我的塔:")
		for i, t := range s.AITowers {
			abStr := "无"
			if len(t.Abilities) > 0 {
				abStr = strings.Join(t.Abilities, "+")
			}
			sb.WriteString(fmt.Sprintf(" 塔%d(%s,%s,伤%.0f,%d杀)",
				i+1, t.Position, abStr, t.Damage, t.Kills))
		}
		sb.WriteString("\n")
	}

	// ── 敌人组成 ──
	if len(s.EnemyGroups) > 0 {
		sb.WriteString("敌人:")
		for _, g := range s.EnemyGroups {
			sb.WriteString(fmt.Sprintf(" %d%s(HP%.0f)", g.Count, g.Type, g.AvgHP))
		}
		sb.WriteString("\n")
	}

	// ── 可用操作 ──
	var availActions []string
	if s.CanBuild {
		availActions = append(availActions, "造塔(50金)")
	}
	if s.CanUpgrade {
		availActions = append(availActions, "升级(10金)")
	}
	if len(s.AvailableItems) > 0 {
		availActions = append(availActions, fmt.Sprintf("道具(%s)", strings.Join(s.AvailableItems, ",")))
	}
	if s.PendingAbilities > 0 {
		availActions = append(availActions, fmt.Sprintf("%d塔待选能力", s.PendingAbilities))
	}
	if len(availActions) > 0 {
		sb.WriteString(fmt.Sprintf("可选: %s\n", strings.Join(availActions, " / ")))
	}

	// ── 协作和策略上下文 ──
	if coopDesc != "" {
		sb.WriteString(fmt.Sprintf("队友: %s\n", coopDesc))
	}
	if advicePriority != "" {
		sb.WriteString(fmt.Sprintf("建议: %s\n", advicePriority))
	}
	if s.NextWaveBoss {
		sb.WriteString("注意: 下一波Boss!\n")
	}

	// ── JSON 输出格式 ──
	sb.WriteString(`
返回JSON(严格格式,不要其他文字):
{"actions":[{"type":"build|upgrade|sell|ability|item|wait","reason":"一句话"}],"say":"弹幕(15字内)"}
type=build时加position(entrance/middle/exit)+priority(cc/dps/aoe)
type=upgrade时加target(strongest/weakest)
type=sell时加target(weakest)
type=item时加itemType(BaseDamage/Speed/Range)+target(strongest)
type=ability时加preference(cc/damage/aoe/dot)
最多3个action,按执行顺序排列。`)

	return sb.String()
}
