// classic_waves_config.go — 经典模式逐波出怪配置加载。
//
// 经典模式要求全确定行为：每波的敌人序列、Boss 原型、附加 buff
// 全部由配置表精确指定，不使用任何随机逻辑。
//
// 配置文件：config/systems/classic-waves/{mapID}.json（每张地图独立配置）
// 关联：由 stage.go 在经典模式初始化时按地图 ID 加载，注入 spawner.ClassicWaves。
package config

import (
	"encoding/json"
	"fmt"
	"slices"
)

// ClassicEnemyEntry 单个敌人的出怪配置。
// JSON 支持两种格式：
//   - 简写: "normal" → {Archetype: "normal", Path: ""}
//   - 完整: {"archetype": "normal", "path": "top"} → 指定走哪条路径
//
// Path 为空时使用地图默认路径（单路径地图的唯一路径）。
type ClassicEnemyEntry struct {
	Archetype string `json:"archetype"` // 原型标识（如 "normal"、"runner"）
	Path      string `json:"path"`      // 路径 ID（如 "top"、"bottom"），空=默认路径
}

// UnmarshalJSON 支持字符串和对象两种 JSON 格式。
func (e *ClassicEnemyEntry) UnmarshalJSON(data []byte) error {
	// 尝试字符串格式: "normal"
	var s string
	if err := json.Unmarshal(data, &s); err == nil {
		e.Archetype = s
		e.Path = ""
		return nil
	}
	// 对象格式: {"archetype": "normal", "path": "top"}
	type alias ClassicEnemyEntry // 避免递归
	var a alias
	if err := json.Unmarshal(data, &a); err != nil {
		return fmt.Errorf("ClassicEnemyEntry: 期望字符串或对象，got %s", string(data))
	}
	*e = ClassicEnemyEntry(a)
	return nil
}

// ClassicBossEntry 经典模式 Boss 配置。
type ClassicBossEntry struct {
	Archetype string `json:"archetype"` // 固定原型（如 "tank"、"colossus"）
	Path      string `json:"path"`      // 路径 ID，空=默认路径
}

// ClassicWaveEntry 单波出怪配置。
//
// 两种互斥的出怪格式：
//   - 单路径: "enemies": ["normal", "runner", ...] — 所有敌人走默认路径
//   - 多路径: "paths": {"top": ["normal", ...], "bottom": ["runner", ...]} — 按路径分组
//
// 多路径格式在加载时展开为 spawnSeq（round-robin 交错），spawner 统一从 spawnSeq 读取。
type ClassicWaveEntry struct {
	Wave    int                                `json:"wave"`    // 波次号（1-based）
	Enemies []ClassicEnemyEntry                `json:"enemies"` // 单路径格式：精确出怪序列
	Paths   map[string][]ClassicEnemyEntry     `json:"paths"`   // 多路径格式：按路径 ID 分组
	Boss    *ClassicBossEntry                  `json:"boss"`    // Boss 配置，null=无 Boss
	Buffs   map[string][]string                `json:"buffs"`   // 按敌人索引(0-based)指定 buff
	spawnSeq []ClassicEnemyEntry                                // 内部：展开后的统一出怪序列
}

// SpawnSeq 返回展开后的出怪序列。
// 单路径模式直接返回 Enemies，多路径模式返回 round-robin 交错的序列。
func (e *ClassicWaveEntry) SpawnSeq() []ClassicEnemyEntry {
	return e.spawnSeq
}

// buildSpawnSeq 构建统一出怪序列。
// 单路径: 直接复用 Enemies（Path 为空，由 spawner 走默认路径）。
// 多路径: 各路径 round-robin 交错，每个 entry 带上对应的 Path。
//
// 例: paths={"top":["A","B","C"], "bottom":["D","E"]}
// → spawnSeq=[A(top), D(bottom), B(top), E(bottom), C(top)]
func (e *ClassicWaveEntry) buildSpawnSeq(pathOrder []string) {
	if len(e.Paths) > 0 {
		// 多路径：round-robin 交错
		// pathOrder 保证遍历顺序确定
		maxLen := 0
		for _, enemies := range e.Paths {
			if len(enemies) > maxLen {
				maxLen = len(enemies)
			}
		}
		e.spawnSeq = make([]ClassicEnemyEntry, 0, maxLen*len(pathOrder))
		for i := 0; i < maxLen; i++ {
			for _, pid := range pathOrder {
				enemies := e.Paths[pid]
				if i < len(enemies) {
					entry := enemies[i]
					entry.Path = pid
					e.spawnSeq = append(e.spawnSeq, entry)
				}
			}
		}
	} else {
		// 单路径：直接复用
		e.spawnSeq = e.Enemies
	}
}

// ClassicWavesScaling 经典模式独立的数值缩放参数。
// 与 spawner.json 的 scaling 结构相似但独立，允许经典模式有不同的增长曲线。
type ClassicWavesScaling struct {
	HpBase        float64 `json:"hpBase"`        // 血量基准值
	HpPerWave     float64 `json:"hpPerWave"`     // 每波血量线性增量
	HpQuadratic   float64 `json:"hpQuadratic"`   // 每波血量二次项系数（wave² 系数）
	SpeedBase     float64 `json:"speedBase"`      // 速度基准值（像素/秒）
	SpeedPerWave  float64 `json:"speedPerWave"`   // 每波速度增量
	SpawnInterval float64 `json:"spawnInterval"`  // 出怪间隔（秒），固定值（经典模式不衰减）
}

// ClassicWavesConfig 经典模式出怪完整配置。
type ClassicWavesConfig struct {
	TotalWaves int                  `json:"totalWaves"` // 总波数，覆盖地图配置
	Scaling    ClassicWavesScaling  `json:"scaling"`    // 独立缩放参数
	Waves      []ClassicWaveEntry   `json:"waves"`      // 逐波出怪定义
	waveMap    map[int]*ClassicWaveEntry               // wave number → entry 快速查找表（内部构建）
}

// GetWave 返回指定波次的出怪配置。未找到返回 nil。
func (c *ClassicWavesConfig) GetWave(n int) *ClassicWaveEntry {
	if c.waveMap == nil {
		c.buildWaveMap()
	}
	return c.waveMap[n]
}

// buildWaveMap 从 Waves 数组构建波次查找表。
func (c *ClassicWavesConfig) buildWaveMap() {
	c.waveMap = make(map[int]*ClassicWaveEntry, len(c.Waves))
	for i := range c.Waves {
		c.waveMap[c.Waves[i].Wave] = &c.Waves[i]
	}
}

// LoadClassicWavesConfig 从 config/systems/classic-waves/{mapID}.json 加载指定地图的经典出怪配置。
// 必须在 SetDataFS() 之后调用。
func LoadClassicWavesConfig(mapID string) (*ClassicWavesConfig, error) {
	if dataFS == nil {
		return nil, fmt.Errorf("load classic waves config: dataFS not initialized")
	}
	path := "config/systems/classic-waves/" + mapID + ".json"
	data, err := dataFS.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("load classic waves config for %s: %w", mapID, err)
	}

	var cfg ClassicWavesConfig
	if err := json.Unmarshal(data, &cfg); err != nil {
		return nil, fmt.Errorf("parse classic waves config: %w", err)
	}

	// 基本校验
	if cfg.TotalWaves <= 0 {
		return nil, fmt.Errorf("classic waves config: totalWaves must be > 0")
	}
	if len(cfg.Waves) == 0 {
		return nil, fmt.Errorf("classic waves config: empty waves array")
	}

	// 构建查找表
	cfg.buildWaveMap()

	// 验证波次完整性：1..totalWaves 每波都必须有定义
	for w := 1; w <= cfg.TotalWaves; w++ {
		if cfg.waveMap[w] == nil {
			return nil, fmt.Errorf("classic waves config: missing wave %d definition", w)
		}
	}

	// 验证每波出怪定义非空 + 构建 spawnSeq
	for i := range cfg.Waves {
		entry := &cfg.Waves[i]
		hasPaths := len(entry.Paths) > 0
		hasEnemies := len(entry.Enemies) > 0
		if !hasPaths && !hasEnemies {
			return nil, fmt.Errorf("classic waves config: wave %d has no enemies or paths", entry.Wave)
		}

		// 多路径格式：提取排序后的路径 key 列表，保证遍历顺序确定
		var pathOrder []string
		if hasPaths {
			for pid := range entry.Paths {
				pathOrder = append(pathOrder, pid)
			}
			slices.Sort(pathOrder)
			// 验证每条路径非空
			for _, pid := range pathOrder {
				if len(entry.Paths[pid]) == 0 {
					return nil, fmt.Errorf("classic waves config: wave %d paths[%s] is empty", entry.Wave, pid)
				}
			}
		}

		entry.buildSpawnSeq(pathOrder)
	}

	// 重新构建 waveMap（buildSpawnSeq 修改了 Waves 元素）
	cfg.buildWaveMap()

	return &cfg, nil
}
