// primitive_meta.go — 基元元数据注册表，供能力编辑器 UI 使用。
//
// 职责：为所有 trigger/condition/selector/effect 基元提供静态元数据，
//       包括 ID、中文标签、描述、费用、参数定义（类型/范围/默认值）。
// 关联：由 UI 层调用 AllXxxMeta() 获取可用基元列表及其参数控件信息。
//       各基元的实际实现分布在 trigger.go / condition.go / selector.go / effect.go 中。
package descriptor

// PrimitiveMeta 基元元数据，描述一个可用的 trigger/condition/selector/effect。
type PrimitiveMeta struct {
	ID     string      `json:"id"`
	Label  string      `json:"label"`
	Desc   string      `json:"desc"`
	Cost   int         `json:"cost"`
	Params []ParamMeta `json:"params"`
}

// ParamMeta 参数元数据，描述基元的一个可配置参数。
// Type 取值: "scaler"/"float"/"int"/"string"/"bool"
// 对于 string/bool 类型，Min/Max/Default 无意义。
type ParamMeta struct {
	Key     string  `json:"key"`
	Label   string  `json:"label"`
	Type    string  `json:"type"`
	Default float64 `json:"default"`
	Min     float64 `json:"min"`
	Max     float64 `json:"max"`
}

// ── Trigger 元数据 ──────────────────────────────────────────

// AllTriggerMeta 返回全部 4 种触发器的元数据。
func AllTriggerMeta() []PrimitiveMeta {
	return []PrimitiveMeta{
		{
			ID:    "onHit",
			Label: "命中时",
			Desc:  "塔的攻击命中敌人时触发",
			Cost:  0,
		},
		{
			ID:    "onTick",
			Label: "每帧",
			Desc:  "每帧持续触发",
			Cost:  0,
		},
		{
			ID:    "onKill",
			Label: "击杀时",
			Desc:  "塔击杀敌人时触发",
			Cost:  0,
		},
		{
			ID:    "onPlace",
			Label: "放置时",
			Desc:  "塔被放置到地图上时触发一次",
			Cost:  0,
		},
	}
}

// ── Condition 元数据 ────────────────────────────────────────

// AllConditionMeta 返回全部 11 种条件门的元数据。
func AllConditionMeta() []PrimitiveMeta {
	return []PrimitiveMeta{
		{
			ID:    "chance",
			Label: "概率",
			Desc:  "以指定概率通过，概率可随强度缩放",
			Cost:  1,
			Params: []ParamMeta{
				{Key: "rate", Label: "触发率", Type: "scaler", Default: 0.3, Min: 0, Max: 1},
			},
		},
		{
			ID:    "cooldown",
			Label: "冷却",
			Desc:  "触发后进入冷却，冷却结束前不再触发",
			Cost:  1,
			Params: []ParamMeta{
				{Key: "seconds", Label: "冷却秒数", Type: "float", Default: 3, Min: 0.1, Max: 30},
			},
		},
		{
			ID:    "hpBelow",
			Label: "HP低于",
			Desc:  "目标血量比例低于阈值时通过",
			Cost:  1,
			Params: []ParamMeta{
				{Key: "threshold", Label: "血量阈值", Type: "scaler", Default: 0.5, Min: 0, Max: 1},
			},
		},
		{
			ID:    "hpAbove",
			Label: "HP高于",
			Desc:  "目标血量比例高于阈值时通过",
			Cost:  1,
			Params: []ParamMeta{
				{Key: "threshold", Label: "血量阈值", Type: "scaler", Default: 0.5, Min: 0, Max: 1},
			},
		},
		{
			ID:    "distanceMin",
			Label: "最小距离",
			Desc:  "目标距离不小于指定值时通过",
			Cost:  1,
			Params: []ParamMeta{
				{Key: "distance", Label: "最小距离", Type: "float", Default: 150, Min: 0, Max: 500},
			},
		},
		{
			ID:    "noNearbyTower",
			Label: "无邻塔",
			Desc:  "附近无友方塔时通过",
			Cost:  1,
			Params: []ParamMeta{
				{Key: "radius", Label: "检测半径", Type: "float", Default: 120, Min: 50, Max: 300},
			},
		},
		{
			ID:    "isBoss",
			Label: "是Boss",
			Desc:  "目标为 Boss 时通过",
			Cost:  1,
		},
		{
			ID:    "notBoss",
			Label: "非Boss",
			Desc:  "目标非 Boss 时通过",
			Cost:  1,
		},
		{
			ID:    "every",
			Label: "每N次",
			Desc:  "每 N 次触发通过一次",
			Cost:  1,
			Params: []ParamMeta{
				{Key: "n", Label: "触发间隔", Type: "int", Default: 3, Min: 1, Max: 20},
			},
		},
		{
			ID:    "buffActive",
			Label: "有Buff",
			Desc:  "目标身上存在指定 buff 时通过",
			Cost:  1,
			Params: []ParamMeta{
				{Key: "buffID", Label: "Buff ID", Type: "string"},
			},
		},
		{
			ID:    "buffAbsent",
			Label: "无Buff",
			Desc:  "目标身上不存在指定 buff 时通过",
			Cost:  1,
			Params: []ParamMeta{
				{Key: "buffID", Label: "Buff ID", Type: "string"},
			},
		},
	}
}

// ── Selector 元数据 ─────────────────────────────────────────

// AllSelectorMeta 返回全部 9 种选择器的元数据。
func AllSelectorMeta() []PrimitiveMeta {
	return []PrimitiveMeta{
		{
			ID:    "currentTarget",
			Label: "当前目标",
			Desc:  "选择当前命中的目标",
			Cost:  0,
		},
		{
			ID:    "aoeRadius",
			Label: "范围选择",
			Desc:  "选择命中点半径内的所有敌人",
			Cost:  2,
			Params: []ParamMeta{
				{Key: "radius", Label: "范围半径", Type: "scaler", Default: 50, Min: 10, Max: 200},
			},
		},
		{
			ID:    "chain",
			Label: "链式弹跳",
			Desc:  "从命中点向最近未命中敌人依次弹跳",
			Cost:  3,
			Params: []ParamMeta{
				{Key: "maxBounce", Label: "弹跳次数", Type: "scaler", Default: 2, Min: 1, Max: 10},
				{Key: "range", Label: "弹跳范围", Type: "float", Default: 80, Min: 50, Max: 200},
				{Key: "decayRatio", Label: "衰减系数", Type: "float", Default: 0.8, Min: 0.1, Max: 1},
			},
		},
		{
			ID:    "allInRange",
			Label: "全范围",
			Desc:  "选择塔攻击范围内所有敌人",
			Cost:  2,
		},
		{
			ID:    "nearbyAllies",
			Label: "友方塔",
			Desc:  "选择塔周围指定半径内的友方塔",
			Cost:  1,
			Params: []ParamMeta{
				{Key: "radius", Label: "搜索半径", Type: "float", Default: 150, Min: 50, Max: 300},
			},
		},
		{
			ID:    "selfTower",
			Label: "自身",
			Desc:  "选择自身塔",
			Cost:  0,
		},
		{
			ID:    "cone",
			Label: "扇形",
			Desc:  "以塔位置为起点，选取锥角范围内的敌人",
			Cost:  2,
			Params: []ParamMeta{
				{Key: "angle", Label: "锥角(度)", Type: "float", Default: 60, Min: 10, Max: 180},
				{Key: "radius", Label: "扇形半径", Type: "scaler", Default: 100, Min: 50, Max: 300},
			},
		},
		{
			ID:    "ring360",
			Label: "环形",
			Desc:  "360度均匀生成方向性目标点",
			Cost:  2,
			Params: []ParamMeta{
				{Key: "count", Label: "方向数", Type: "scaler", Default: 6, Min: 2, Max: 16},
			},
		},
		{
			ID:    "random",
			Label: "随机",
			Desc:  "从塔周围随机选取指定数量的敌人",
			Cost:  1,
			Params: []ParamMeta{
				{Key: "count", Label: "选取数量", Type: "scaler", Default: 3, Min: 1, Max: 10},
				{Key: "radius", Label: "搜索半径", Type: "float", Default: 150, Min: 50, Max: 300},
			},
		},
	}
}

// ── Effect 元数据 ───────────────────────────────────────────

// AllEffectMeta 返回全部 14 种效果的元数据。
func AllEffectMeta() []PrimitiveMeta {
	return []PrimitiveMeta{
		{
			ID:    "damage",
			Label: "伤害",
			Desc:  "对目标造成直接伤害",
			Cost:  3,
			Params: []ParamMeta{
				{Key: "mode", Label: "伤害模式", Type: "string"},
				{Key: "value", Label: "伤害值", Type: "scaler", Default: 1, Min: 0, Max: 9999},
			},
		},
		{
			ID:    "slow",
			Label: "减速",
			Desc:  "降低目标移动速度",
			Cost:  3,
			Params: []ParamMeta{
				{Key: "factor", Label: "减速系数", Type: "scaler", Default: 0.3, Min: 0, Max: 1},
				{Key: "duration", Label: "持续时间", Type: "scaler", Default: 1, Min: 0.1, Max: 5},
			},
		},
		{
			ID:    "stun",
			Label: "眩晕",
			Desc:  "使目标无法移动和行动",
			Cost:  6,
			Params: []ParamMeta{
				{Key: "duration", Label: "持续时间", Type: "scaler", Default: 0.5, Min: 0.1, Max: 3},
			},
		},
		{
			ID:    "root",
			Label: "定身",
			Desc:  "使目标无法移动但仍可行动",
			Cost:  5,
			Params: []ParamMeta{
				{Key: "duration", Label: "持续时间", Type: "scaler", Default: 1, Min: 0.1, Max: 5},
			},
		},
		{
			ID:    "dot",
			Label: "持续伤害",
			Desc:  "对目标施加持续伤害效果",
			Cost:  3,
			Params: []ParamMeta{
				{Key: "subtype", Label: "DoT类型", Type: "string"},
				{Key: "mode", Label: "伤害模式", Type: "string"},
				{Key: "value", Label: "每跳伤害", Type: "scaler", Default: 1, Min: 0, Max: 9999},
				{Key: "duration", Label: "持续时间", Type: "scaler", Default: 3, Min: 0.1, Max: 30},
			},
		},
		{
			ID:    "weaken",
			Label: "易伤",
			Desc:  "增加目标受到的伤害",
			Cost:  3,
			Params: []ParamMeta{
				{Key: "amplify", Label: "增伤比例", Type: "scaler", Default: 0.15, Min: 0, Max: 1},
				{Key: "duration", Label: "持续时间", Type: "scaler", Default: 3, Min: 0.5, Max: 5},
			},
		},
		{
			ID:    "silence",
			Label: "沉默",
			Desc:  "禁用目标的特殊能力",
			Cost:  6,
		},
		{
			ID:    "buff",
			Label: "友方增益",
			Desc:  "增强友方塔的指定属性",
			Cost:  2,
			Params: []ParamMeta{
				{Key: "stat", Label: "属性", Type: "string"},
				{Key: "bonus", Label: "增益值", Type: "scaler", Default: 0.1, Min: 0, Max: 9999},
			},
		},
		{
			ID:    "selfBuff",
			Label: "自身增益",
			Desc:  "增强自身塔的指定属性",
			Cost:  2,
			Params: []ParamMeta{
				{Key: "stat", Label: "属性", Type: "string"},
				{Key: "bonus", Label: "增益值", Type: "scaler", Default: 0.1, Min: 0, Max: 9999},
			},
		},
		{
			ID:    "gold",
			Label: "产金",
			Desc:  "获得额外金币",
			Cost:  2,
			Params: []ParamMeta{
				{Key: "amount", Label: "金币量", Type: "scaler", Default: 1, Min: 0.5, Max: 10},
			},
		},
		{
			ID:    "modifyStat",
			Label: "改属性",
			Desc:  "按倍率修改目标属性",
			Cost:  3,
			Params: []ParamMeta{
				{Key: "stat", Label: "属性", Type: "string"},
				{Key: "multiplier", Label: "倍率", Type: "float", Default: 1.5, Min: 0.5, Max: 3},
			},
		},
		{
			ID:    "crit",
			Label: "暴击",
			Desc:  "独立暴击效果，可在条件管线中单独控制",
			Cost:  3,
			Params: []ParamMeta{
				{Key: "multiplier", Label: "暴击倍率", Type: "float", Default: 2, Min: 1.5, Max: 5},
			},
		},
		{
			ID:    "purge",
			Label: "净化",
			Desc:  "移除敌人身上的 buff",
			Cost:  4,
			Params: []ParamMeta{
				{Key: "count", Label: "净化数量", Type: "int", Default: 1, Min: 1, Max: 5},
			},
		},
		{
			ID:    "teleport",
			Label: "传送回推",
			Desc:  "将目标沿路径回推指定距离",
			Cost:  3,
			Params: []ParamMeta{
				{Key: "distance", Label: "回推距离", Type: "scaler", Default: 100, Min: 10, Max: 500},
			},
		},
	}
}
