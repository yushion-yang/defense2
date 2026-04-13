// Package buff 提供统一的状态效果（增益/减益）系统。
//
// 本包是零游戏依赖的通用容器：敌人 CC/DoT/行为 buff 和塔光环 buff 均使用同一个 BuffList。
// 不依赖 core/* 中的任何游戏对象（Tower/Enemy 等），只接受基本类型参数。
//
// 核心设计：
//   - 6 种堆叠模式（StackMode）控制同 ID buff 如何合并
//   - Value/Value2 的语义由 buff ID 决定（见 Buff 结构体注释）
//   - buff-stack.json 定义每个 buff ID 的堆叠规则、上限和下限
//   - Duration < 0 表示永久 buff，不会被 Tick 自然清除
//
// 调用顺序约定：每帧必须先 TickDoT() 再 Tick()，确保即将过期的 DoT 仍能造最后一跳伤害。
package buff

// Category 将 buff 按功能分类，用于批量清除（如净化清除所有 CatCC）。
type Category int

const (
	CatCC       Category = iota // 控制类: stun(眩晕), slow(减速), root(定身)
	CatDoT                      // 持续伤害类: bleed(流血), burn(灼烧), poison(中毒)
	CatDefense                  // 防御类: controlImmune(控制免疫), invincible(无敌), damageImmune(伤害免疫)
	CatDebuff                   // 减益类: weaken(虚弱/增伤), silence(沉默)
	CatBehavior                 // 行为类: berserk(狂暴), regen(回血), healAura(治疗光环), bufferAura(旗手光环)
	CatAura                     // 塔光环类: 相邻塔通过 BuffList 施加的属性加成
)

// StackMode 决定同一个 ID 的多个 buff 实例如何合并。
// 规则在 buff-stack.json 中按 buff ID 配置，默认 Override。
type StackMode int

const (
	Strongest            StackMode = iota // 保留最高 Value；Value 相同时保留剩余时间更长的
	Additive                              // 所有实例的 Value 累加（Get 返回合计值）
	Multiplicative                        // 所有实例的 Value 连乘
	Override                              // 后来居上：新实例直接覆盖旧实例
	Independent                           // 各实例独立存在、独立计算（如护盾）
	IndependentPerSource                  // 同 Source 覆盖，不同 Source 共存（如塔光环：每塔一个实例）
)

// Buff 表示一个活跃的状态效果实例。
//
// Value/Value2 的语义取决于 buff ID：
//
//	| ID            | Value 含义           | Value2 含义              |
//	|---------------|----------------------|--------------------------|
//	| slow          | 减速系数 (0~0.8)     | (未使用)                 |
//	| stun/root     | (无意义，仅存在即生效)| (未使用)                 |
//	| bleed/burn    | DPS (每秒伤害)       | (未使用)                 |
//	| poison        | DPS (每秒伤害)       | (未使用)                 |
//	| weaken        | 增伤倍率             | (未使用)                 |
//	| damageReduce  | 减伤比例 (0~0.8)     | (未使用)                 |
//	| berserk       | 速度/伤害倍率        | 血量触发阈值 (百分比)    |
//	| regen         | 每秒回血比例 (%maxHP) | (未使用)                 |
//	| healAura      | 治疗量缩放系数       | 光环半径                 |
//	| bufferAura    | 加速系数             | 光环半径                 |
//	| speedUp       | 加速系数 (cap=1.4)   | (未使用)                 |
//	| phaseShift    | 相位持续秒数         | 冷却秒数                 |
//	| stealth       | (存在即隐身)         | (未使用)                 |
//	| aura:*        | 属性加成值           | (未使用)                 |
type Buff struct {
	ID        string   // 与 buff-stack.json 的 key 对应: "slow", "stun", "bleed" 等（推荐用 ids.go 常量）
	Category  Category // 分类标识，用于 ClearByCategory 批量净化
	Source    string   // 来源标识，如 "tower_3", "wave_buff", "archetype"。IndependentPerSource 模式依赖此字段区分实例
	Value     float64  // 主数值（语义取决于 ID，见上表）
	Value2    float64  // 辅助数值（多数 buff 不使用；healAura/bufferAura 用作光环半径，berserk 用作触发阈值，phaseShift 用作冷却）
	Duration  float64  // 总持续时间；负数(如 -1)表示永久 buff，不会被 Tick 自然递减
	Remaining float64  // 剩余时间；负数表示永久。Tick() 每帧递减此值，归零后移除
	Priority  int      // 仅 Override 模式下用于优先级竞争（数值越高优先级越高）
}
