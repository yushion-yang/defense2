// mode_config.go — 配置驱动模式系统的 JSON 数据结构与加载。
//
// 将 gamemodes.json 映射为 Go 结构体，供 UniversalMode 使用。
// 所有模式参数（胜负条件、经济、HUD、分数公式、塔规则）
// 均从 JSON 配置读取，消除 10 个模式文件的样板代码。
//
// 加载流程：LoadModeConfigs() → map[string]ModeConfig → NewUniversalMode(id, cfg)
package gamemode

import (
	"encoding/json"
	"fmt"
	"sync"

	"defense2/internal/config"
)

// ModeConfig 单个游戏模式的完整配置。
type ModeConfig struct {
	AutoStart    bool          `json:"autoStart"`
	Intermission float64       `json:"intermission"` // 0 = 使用 spawner 默认值
	InitMaxWaves int           `json:"initMaxWaves"` // 0 = 使用地图默认值
	Victory      string        `json:"victory"`      // allWaves|never|hook:countdown|hook:bossCounter
	Defeat       string        `json:"defeat"`       // livesZero|never
	EnableEvents bool          `json:"enableEvents"`
	EconID       string        `json:"econID"`
	PerfectBonus bool          `json:"perfectBonus"`
	CoopEnabled  bool          `json:"coopEnabled"`
	Ruleset      RulesetConfig `json:"ruleset"`
	Score        ScoreConfig   `json:"score"`
	Hooks        []HookConfig  `json:"hooks"`
	EndFields    []string      `json:"endFields"`
	UI           UIConfig      `json:"ui"`
}

// RulesetConfig 塔建造规则配置（对应 TowerRuleset 接口）。
type RulesetConfig struct {
	PresetTowers   bool   `json:"presetTowers"`   // 仅使用预设塔列表
	IncludePresets bool   `json:"includePresets"` // 追加预设塔（标准+预设共存）
	ClassicWaves   bool   `json:"classicWaves"`   // 使用经典确定性出怪
	WardenEnabled  bool   `json:"wardenEnabled"`  // 启用战灵选择
	ItemDrop       string `json:"itemDrop"`       // probability|everyKill|byWave|none
	Blueprints     bool   `json:"blueprints"`     // 允许自定义蓝图
	BudgetCap      int    `json:"budgetCap"`      // 蓝图预算上限（-1=默认）
}

// ScoreConfig 分数计算权重。
// 最终分数 = wave*Wave + kill*Kill + leaked*Leaked + lives*Lives。
// 零值字段不参与计算，leaked 通常为负值表示惩罚。
type ScoreConfig struct {
	Wave   int `json:"wave,omitempty"`
	Kill   int `json:"kill,omitempty"`
	Leaked int `json:"leaked,omitempty"`
	Lives  int `json:"lives,omitempty"`
}

// HookConfig 模式钩子配置（倒计时/Boss计数等运行时扩展）。
// Type 决定使用哪个 Hook 实现，其余字段为该 Hook 的参数。
type HookConfig struct {
	Type          string  `json:"type"`                    // countdown|bossCounter
	TargetSeconds float64 `json:"targetSeconds,omitempty"` // countdown 专用
	TotalBosses   int     `json:"totalBosses,omitempty"`   // bossCounter 专用
	TimeBase      int     `json:"timeBase,omitempty"`      // bossCounter 时间分基数
	TimeDecay     float64 `json:"timeDecay,omitempty"`     // bossCounter 每秒扣分
}

// UIConfig 模式在选择界面的展示配置。
type UIConfig struct {
	NameKey    string `json:"nameKey"`    // i18n 键（模式名称）
	DescKey    string `json:"descKey"`    // i18n 键（模式描述）
	Icon       string `json:"icon"`       // 图标标识
	DefaultMap string `json:"defaultMap"` // 默认地图 ID（直接进 Stage 时使用）
	Flow       string `json:"flow"`       // 进入流程：campaignSelect|testSelect|direct
	ComingSoon bool   `json:"comingSoon"` // 是否显示"敬请期待"
	DevOnly    bool   `json:"devOnly"`    // 是否仅开发模式可见
	Order      int    `json:"order"`      // 显示排序（越小越靠前）
}

// ── 加载 ──────────────────────────────────────────────

var (
	modeConfigCache map[string]ModeConfig
	modeConfigOnce  sync.Once
	modeConfigErr   error
)

// LoadModeConfigs 从 embed FS 加载 gamemodes.json，返回模式ID→配置的映射。
// 使用 sync.Once 保证只加载一次，后续调用返回缓存。
func LoadModeConfigs() (map[string]ModeConfig, error) {
	modeConfigOnce.Do(func() {
		fs := config.GetDataFS()
		if fs == nil {
			modeConfigErr = fmt.Errorf("load gamemodes: dataFS not initialized")
			return
		}
		data, err := fs.ReadFile("config/gamemodes.json")
		if err != nil {
			modeConfigErr = fmt.Errorf("load gamemodes: %w", err)
			return
		}
		var cfgs map[string]ModeConfig
		if err := json.Unmarshal(data, &cfgs); err != nil {
			modeConfigErr = fmt.Errorf("parse gamemodes: %w", err)
			return
		}
		modeConfigCache = cfgs
	})
	return modeConfigCache, modeConfigErr
}

// GetModeConfig 获取单个模式的配置，未找到返回零值和 false。
func GetModeConfig(id string) (ModeConfig, bool) {
	cfgs, err := LoadModeConfigs()
	if err != nil || cfgs == nil {
		return ModeConfig{}, false
	}
	cfg, ok := cfgs[id]
	return cfg, ok
}

// ResetModeConfigCache 重置缓存（仅用于测试）。
func ResetModeConfigCache() {
	modeConfigOnce = sync.Once{}
	modeConfigCache = nil
	modeConfigErr = nil
}
