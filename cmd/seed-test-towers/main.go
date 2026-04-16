// seed-test-towers — 写入 OP 测试用自定义能力和蓝图到 ~/.defense2/。
//
// 用途：开发测试时快速生成一组超强能力和蓝图，供玩家在游戏内直接使用。
// 直接写入原始 JSON（不经过 AbilityStore/BlueprintStore），
// 因为当前 store 的 Pipeline 接口字段无法正确反序列化。
//
// 用法：go run cmd/seed-test-towers/main.go [--clean]
//   --clean  删除已有的 custom_abilities.json 和 tower_blueprints.json 后重写
package main

import (
	"encoding/json"
	"fmt"
	"log"
	"os"
	"path/filepath"
)

func main() {
	clean := len(os.Args) > 1 && os.Args[1] == "--clean"

	home, err := os.UserHomeDir()
	if err != nil {
		log.Fatalf("获取用户目录失败: %v", err)
	}
	dir := filepath.Join(home, ".defense2")
	if err := os.MkdirAll(dir, 0755); err != nil {
		log.Fatalf("创建 .defense2 目录失败: %v", err)
	}

	abPath := filepath.Join(dir, "custom_abilities.json")
	bpPath := filepath.Join(dir, "tower_blueprints.json")

	if clean {
		os.Remove(abPath)
		os.Remove(bpPath)
		fmt.Println("已清理旧数据")
	}

	// ── 写入自定义能力 ──────────────────────────────────
	abData := buildAbilities()
	if err := writeJSON(abPath, abData); err != nil {
		log.Fatalf("写入 custom_abilities.json 失败: %v", err)
	}
	fmt.Printf("已写入 %d 个自定义能力 → %s\n", len(abData["abilities"].([]any)), abPath)

	// ── 写入蓝图 ────────────────────────────────────────
	bpData := buildBlueprints()
	if err := writeJSON(bpPath, bpData); err != nil {
		log.Fatalf("写入 tower_blueprints.json 失败: %v", err)
	}
	fmt.Printf("已写入 %d 个蓝图 → %s\n", len(bpData["blueprints"].([]any)), bpPath)

	fmt.Println("\n运行 'make run' 即可在游戏内测试")
}

// writeJSON 将数据序列化为格式化 JSON 并写入文件。
func writeJSON(path string, data map[string]any) error {
	bytes, err := json.MarshalIndent(data, "", "  ")
	if err != nil {
		return fmt.Errorf("marshal: %w", err)
	}
	// 原子写入：先写临时文件再重命名
	tmp := path + ".tmp"
	if err := os.WriteFile(tmp, bytes, 0644); err != nil {
		return fmt.Errorf("write: %w", err)
	}
	return os.Rename(tmp, path)
}

// ── 能力定义 ──────────────────────────────────────────

func buildAbilities() map[string]any {
	return map[string]any{
		"version": 1,
		"abilities": []any{
			permaFreezeZone(),
			totalParalysis(),
			killBomb(),
			bossHunter(),
			tripleAura(),
		},
	}
}

// permaFreezeZone — 永久冻结场（cost ~6）。
// 每帧对射程内所有敌人施加 0.1 秒眩晕，等效永久定身。
func permaFreezeZone() map[string]any {
	return map[string]any{
		"id":   "ca_perma_freeze",
		"name": "永久冻结场",
		"descriptor": map[string]any{
			"id":    "ca_perma_freeze",
			"label": "永久冻结场",
			"cost":  6,
			"tags":  []string{"cc", "zone"},
			"pipelines": []any{
				map[string]any{
					"trigger":    "onTick",
					"conditions": []any{},
					"selector":   map[string]any{"type": "allInRange"},
					"effects": []any{
						map[string]any{
							"type":     "stun",
							"duration": map[string]any{"scaler": "fixed", "value": 0.1},
						},
					},
				},
			},
		},
	}
}

// totalParalysis — 全域瘫痪（cost ~16）。
// 每帧眩晕 + 减速 + 削弱，三重控制叠加。
func totalParalysis() map[string]any {
	return map[string]any{
		"id":   "ca_total_paralysis",
		"name": "全域瘫痪",
		"descriptor": map[string]any{
			"id":    "ca_total_paralysis",
			"label": "全域瘫痪",
			"cost":  16,
			"tags":  []string{"cc", "zone"},
			"pipelines": []any{
				map[string]any{
					"trigger":    "onTick",
					"conditions": []any{},
					"selector":   map[string]any{"type": "allInRange"},
					"effects": []any{
						map[string]any{
							"type":     "stun",
							"duration": map[string]any{"scaler": "fixed", "value": 0.1},
						},
					},
				},
				map[string]any{
					"trigger":    "onTick",
					"conditions": []any{},
					"selector":   map[string]any{"type": "allInRange"},
					"effects": []any{
						map[string]any{
							"type":     "slow",
							"factor":   map[string]any{"scaler": "fixed", "value": 0.3},
							"duration": map[string]any{"scaler": "fixed", "value": 0.5},
						},
					},
				},
				map[string]any{
					"trigger":    "onTick",
					"conditions": []any{},
					"selector":   map[string]any{"type": "allInRange"},
					"effects": []any{
						map[string]any{
							"type":     "weaken",
							"amplify":  map[string]any{"scaler": "fixed", "value": 0.3},
							"duration": map[string]any{"scaler": "fixed", "value": 0.5},
						},
					},
				},
			},
		},
	}
}

// killBomb — 击杀核爆（cost ~12）。
// 击杀敌人时对 80 半径内所有敌人造成 150%+50% 伤害并眩晕 0.8 秒。
func killBomb() map[string]any {
	return map[string]any{
		"id":   "ca_kill_bomb",
		"name": "击杀核爆",
		"descriptor": map[string]any{
			"id":    "ca_kill_bomb",
			"label": "击杀核爆",
			"cost":  12,
			"tags":  []string{"damage", "cc"},
			"pipelines": []any{
				map[string]any{
					"trigger":    "onKill",
					"conditions": []any{},
					"selector": map[string]any{
						"type":   "aoeRadius",
						"radius": map[string]any{"scaler": "fixed", "value": 80.0},
					},
					"effects": []any{
						map[string]any{
							"type":  "damage",
							"mode":  "ratio",
							"value": map[string]any{"scaler": "linear", "base": 1.5, "potential": 0.5},
						},
						map[string]any{
							"type":     "stun",
							"duration": map[string]any{"scaler": "fixed", "value": 0.8},
						},
					},
				},
			},
		},
	}
}

// bossHunter — Boss猎手（cost ~20）。
// 命中 Boss 时净化 3 层 + 沉默；Boss 半血以下时额外比例伤害 + 3 倍暴击。
func bossHunter() map[string]any {
	return map[string]any{
		"id":   "ca_boss_hunter",
		"name": "Boss猎手",
		"descriptor": map[string]any{
			"id":    "ca_boss_hunter",
			"label": "Boss猎手",
			"cost":  20,
			"tags":  []string{"damage", "cc"},
			"pipelines": []any{
				// 管线 1：命中 Boss → 净化 + 沉默
				map[string]any{
					"trigger": "onHit",
					"conditions": []any{
						map[string]any{"type": "isBoss"},
					},
					"selector": map[string]any{"type": "currentTarget"},
					"effects": []any{
						map[string]any{"type": "purge", "count": 3},
						map[string]any{"type": "silence"},
					},
				},
				// 管线 2：命中半血以下 Boss → 额外伤害 + 暴击
				map[string]any{
					"trigger": "onHit",
					"conditions": []any{
						map[string]any{"type": "isBoss"},
						map[string]any{
							"type":      "hpBelow",
							"threshold": map[string]any{"scaler": "fixed", "value": 0.5},
						},
					},
					"selector": map[string]any{"type": "currentTarget"},
					"effects": []any{
						map[string]any{
							"type":  "damage",
							"mode":  "ratio",
							"value": map[string]any{"scaler": "linear", "base": 0.5, "potential": 0.1},
						},
						map[string]any{"type": "crit", "multiplier": 3.0},
					},
				},
			},
		},
	}
}

// tripleAura — 三光环合一（cost ~8）。
// 每帧为 150 半径内友方塔提供 damage/speed/crit 三重增益。
func tripleAura() map[string]any {
	return map[string]any{
		"id":   "ca_triple_aura",
		"name": "三光环合一",
		"descriptor": map[string]any{
			"id":    "ca_triple_aura",
			"label": "三光环合一",
			"cost":  8,
			"tags":  []string{"buff"},
			"pipelines": []any{
				map[string]any{
					"trigger":    "onTick",
					"conditions": []any{},
					"selector":   map[string]any{"type": "nearbyAllies", "radius": 150.0},
					"effects": []any{
						map[string]any{
							"type":  "buff",
							"stat":  "damage",
							"bonus": map[string]any{"scaler": "linear", "base": 0.05, "potential": 0.02},
						},
					},
				},
				map[string]any{
					"trigger":    "onTick",
					"conditions": []any{},
					"selector":   map[string]any{"type": "nearbyAllies", "radius": 150.0},
					"effects": []any{
						map[string]any{
							"type":  "buff",
							"stat":  "speed",
							"bonus": map[string]any{"scaler": "linear", "base": 0.08, "potential": 0.02},
						},
					},
				},
				map[string]any{
					"trigger":    "onTick",
					"conditions": []any{},
					"selector":   map[string]any{"type": "nearbyAllies", "radius": 150.0},
					"effects": []any{
						map[string]any{
							"type":  "buff",
							"stat":  "crit",
							"bonus": map[string]any{"scaler": "linear", "base": 0.04, "potential": 0.02},
						},
					},
				},
			},
		},
	}
}

// ── 蓝图定义 ──────────────────────────────────────────

func buildBlueprints() map[string]any {
	return map[string]any{
		"version": 1,
		"blueprints": []any{
			absoluteZone(),
			doomsday(),
			totalControl(),
		},
	}
}

// absoluteZone — 绝对禁区。
// 永久冻结 + 三光环 + 金币被动，S 射程。
func absoluteZone() map[string]any {
	return map[string]any{
		"id":          "bp_absolute_zone",
		"name":        "绝对禁区",
		"author":      "test",
		"attackStyle": "projectile",
		"tiers": map[string]string{
			"damage":   "D",
			"atkSpeed": "D",
			"range":    "S",
		},
		"specialty": "range",
		"abilities": []string{"ca_perma_freeze", "ca_triple_aura", "goldPassive"},
		"strength": map[string]any{
			"cost":         50,
			"amount":       50.0,
			"maxPurchases": -1,
		},
	}
}

// doomsday — 末日审判。
// 强化 + Boss 猎手 + 击杀核爆，S 伤害。
func doomsday() map[string]any {
	return map[string]any{
		"id":          "bp_doomsday",
		"name":        "末日审判",
		"author":      "test",
		"attackStyle": "projectile",
		"tiers": map[string]string{
			"damage":   "S",
			"atkSpeed": "B",
			"range":    "D",
		},
		"specialty": "damage",
		"abilities": []string{"enhance", "ca_boss_hunter", "ca_kill_bomb"},
		"strength": map[string]any{
			"cost":         50,
			"amount":       50.0,
			"maxPurchases": -1,
		},
	}
}

// totalControl — 全域瘫痪者。
// 全域瘫痪 + 毒区，旋转 AOE + S 射程。
func totalControl() map[string]any {
	return map[string]any{
		"id":          "bp_total_control",
		"name":        "全域瘫痪者",
		"author":      "test",
		"attackStyle": "spin_aoe",
		"tiers": map[string]string{
			"damage":   "D",
			"atkSpeed": "B",
			"range":    "S",
		},
		"specialty": "range",
		"abilities": []string{"ca_total_paralysis", "poisonZone"},
		"strength": map[string]any{
			"cost":         50,
			"amount":       50.0,
			"maxPurchases": -1,
		},
	}
}
