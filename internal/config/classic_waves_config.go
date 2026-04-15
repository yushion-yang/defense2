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
type ClassicWaveEntry struct {
	Wave    int                 `json:"wave"`    // 波次号（1-based）
	Enemies []ClassicEnemyEntry `json:"enemies"` // 精确出怪序列，按数组顺序逐个生成
	Boss    *ClassicBossEntry   `json:"boss"`    // Boss 配置，null=无 Boss
	Buffs   map[string][]string `json:"buffs"`   // 按敌人索引(0-based)指定 buff，key="索引"，value=buff ID 列表
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

	// 验证每波 enemies 非空
	for _, entry := range cfg.Waves {
		if len(entry.Enemies) == 0 {
			return nil, fmt.Errorf("classic waves config: wave %d has empty enemies", entry.Wave)
		}
	}

	return &cfg, nil
}
