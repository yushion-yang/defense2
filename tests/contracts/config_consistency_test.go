// config_consistency_test.go — 跨配置一致性契约测试。
// 验证所有配置文件之间的交叉引用有效性，确保不存在悬空引用。
package contracts_test

import (
	"encoding/json"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	defense2 "defense2"
	"defense2/internal/config"
	"defense2/internal/core/buff"
)

// ═══════════════════════════════════════
// 1. 原型一致性：波次组合中的每个原型都存在于 enemies-core.json
// ═══════════════════════════════════════

func TestConsistency_WaveArchetypesExistInEnemyCore(t *testing.T) {
	archetypes, err := config.LoadEnemyArchetypes()
	if err != nil {
		t.Fatalf("加载敌人原型失败: %v", err)
	}
	if len(archetypes) == 0 {
		t.Fatal("敌人原型表为空")
	}

	// 从 wave-compositions.json 动态读取所有引用的原型名
	if err := config.LoadWaveCompositions(); err != nil {
		t.Fatalf("加载波次组合配置失败: %v", err)
	}
	referencedArchetypes := config.WaveCompositionArchetypes()
	if len(referencedArchetypes) == 0 {
		t.Fatal("波次组合中无原型引用")
	}

	for _, arch := range referencedArchetypes {
		t.Run(arch, func(t *testing.T) {
			if _, ok := archetypes[arch]; !ok {
				t.Errorf("波次组合引用的原型 %q 不存在于 enemies-core.json", arch)
			}
		})
	}
}

func TestConsistency_EnemyCoreArchetypesHaveUniqueIDs(t *testing.T) {
	archetypes, err := config.LoadEnemyArchetypes()
	if err != nil {
		t.Fatalf("加载敌人原型失败: %v", err)
	}

	seen := make(map[string]bool)
	for id := range archetypes {
		if seen[id] {
			t.Errorf("重复的原型 ID: %q", id)
		}
		seen[id] = true
	}
}

// ═══════════════════════════════════════
// 2. 敌人能力一致性：enemies-core.json 引用的能力都存在于 abilities.json
// ═══════════════════════════════════════

func TestConsistency_EnemyAbilitiesExistInAbilityDefs(t *testing.T) {
	archetypes, err := config.LoadEnemyArchetypes()
	if err != nil {
		t.Fatalf("加载敌人原型失败: %v", err)
	}
	abilityDefs, err := config.LoadEnemyAbilities()
	if err != nil {
		t.Fatalf("加载敌人能力定义失败: %v", err)
	}
	if len(abilityDefs) == 0 {
		t.Fatal("敌人能力定义表为空")
	}

	for id, arch := range archetypes {
		for _, ref := range arch.Abilities {
			t.Run(id+"/"+ref.Type, func(t *testing.T) {
				if _, ok := abilityDefs[ref.Type]; !ok {
					t.Errorf("原型 %q 引用能力 %q 不存在于 enemies/abilities.json", id, ref.Type)
				}
			})
		}
	}
}

func TestConsistency_EnemyAbilityDefsHaveRequiredFields(t *testing.T) {
	abilityDefs, err := config.LoadEnemyAbilities()
	if err != nil {
		t.Fatalf("加载敌人能力定义失败: %v", err)
	}

	validCategories := map[string]bool{
		"defense": true, "passive": true, "resist": true,
		"movement": true, "offense": true, "support": true, "death": true,
	}

	for id, def := range abilityDefs {
		t.Run(id, func(t *testing.T) {
			if def.Type == "" {
				t.Error("type 字段为空")
			}
			if def.Label == "" {
				t.Error("label 字段为空")
			}
			if def.Category == "" {
				t.Error("category 字段为空")
			} else if !validCategories[def.Category] {
				t.Errorf("无效的 category %q", def.Category)
			}
		})
	}
}

// ═══════════════════════════════════════
// 3. 塔能力一致性：abilities.json 中每个能力有完整的必填字段
// ═══════════════════════════════════════

func TestConsistency_TowerAbilitiesHaveRequiredFields(t *testing.T) {
	table := config.GlobalAbilityTable()
	if table == nil || len(table) == 0 {
		// 尝试加载
		var err error
		table, err = config.LoadAbilityTable()
		if err != nil {
			t.Fatalf("加载能力表失败: %v", err)
		}
	}

	validCategories := map[string]bool{
		"attack": true, "cc": true, "damage": true,
		"buff": true, "dot": true, "zone": true,
	}

	for id, def := range table {
		t.Run(id, func(t *testing.T) {
			if def.Type == "" {
				t.Error("type 字段为空")
			}
			if def.Label == "" {
				t.Error("label 字段为空")
			}
			if def.Category == "" {
				t.Error("category 字段为空")
			} else if !validCategories[def.Category] {
				t.Errorf("无效的 category %q", def.Category)
			}
			if def.Icon == "" {
				t.Error("icon 字段为空")
			}
			if def.Display == "" {
				t.Error("display 字段为空")
			}
		})
	}
}

func TestConsistency_TowerAbilityTypesMatchKeys(t *testing.T) {
	table, err := config.LoadAbilityTable()
	if err != nil {
		t.Fatalf("加载能力表失败: %v", err)
	}

	for key, def := range table {
		t.Run(key, func(t *testing.T) {
			if def.Type != key {
				t.Errorf("能力 key=%q 与 type=%q 不匹配", key, def.Type)
			}
		})
	}
}

// ═══════════════════════════════════════
// 4. 音频文件一致性：sfx.json/bgm.json 引用的每个 WAV 文件都存在于 assets/audio/
// ═══════════════════════════════════════

// projectRoot 返回项目根目录路径。
func projectRoot() string {
	_, filename, _, _ := runtime.Caller(0)
	return filepath.Join(filepath.Dir(filename), "..", "..")
}

// sfxEntry 用于解析 sfx.json 中的音效条目。
type sfxEntry struct {
	File string `json:"file"`
}

// sfxCategory 用于解析 sfx.json 中的分类。
type sfxCategory struct {
	SFX []sfxEntry `json:"sfx"`
}

// sfxConfig 用于解析 sfx.json 顶层结构。
type sfxConfig struct {
	Categories []sfxCategory `json:"categories"`
}

// bgmTrack 用于解析 bgm.json 中的音轨条目。
type bgmTrack struct {
	File string `json:"file"`
}

// bgmConfig 用于解析 bgm.json 顶层结构。
type bgmConfig struct {
	Tracks []bgmTrack `json:"tracks"`
}

func TestConsistency_SFXFilesExistInAssets(t *testing.T) {
	root := projectRoot()
	sfxPath := filepath.Join(root, "config", "audio", "sfx.json")
	data, err := os.ReadFile(sfxPath)
	if err != nil {
		t.Fatalf("读取 sfx.json 失败: %v", err)
	}

	var cfg sfxConfig
	if err := json.Unmarshal(data, &cfg); err != nil {
		t.Fatalf("解析 sfx.json 失败: %v", err)
	}

	audioDir := filepath.Join(root, "assets", "audio")
	for _, cat := range cfg.Categories {
		for _, sfx := range cat.SFX {
			if sfx.File == "" {
				continue
			}
			t.Run(sfx.File, func(t *testing.T) {
				wavPath := filepath.Join(audioDir, sfx.File)
				if _, err := os.Stat(wavPath); os.IsNotExist(err) {
					t.Errorf("sfx.json 引用的音频文件 %q 不存在于 assets/audio/", sfx.File)
				}
			})
		}
	}
}

func TestConsistency_BGMFilesExistInAssets(t *testing.T) {
	root := projectRoot()
	bgmPath := filepath.Join(root, "config", "audio", "bgm.json")
	data, err := os.ReadFile(bgmPath)
	if err != nil {
		t.Fatalf("读取 bgm.json 失败: %v", err)
	}

	var cfg bgmConfig
	if err := json.Unmarshal(data, &cfg); err != nil {
		t.Fatalf("解析 bgm.json 失败: %v", err)
	}

	audioDir := filepath.Join(root, "assets", "audio")
	for _, track := range cfg.Tracks {
		if track.File == "" {
			continue
		}
		t.Run(track.File, func(t *testing.T) {
			wavPath := filepath.Join(audioDir, track.File)
			if _, err := os.Stat(wavPath); os.IsNotExist(err) {
				t.Errorf("bgm.json 引用的音频文件 %q 不存在于 assets/audio/", track.File)
			}
		})
	}
}

func TestConsistency_SFXFilesHaveWavExtension(t *testing.T) {
	root := projectRoot()
	sfxPath := filepath.Join(root, "config", "audio", "sfx.json")
	data, err := os.ReadFile(sfxPath)
	if err != nil {
		t.Fatalf("读取 sfx.json 失败: %v", err)
	}

	var cfg sfxConfig
	if err := json.Unmarshal(data, &cfg); err != nil {
		t.Fatalf("解析 sfx.json 失败: %v", err)
	}

	for _, cat := range cfg.Categories {
		for _, sfx := range cat.SFX {
			if sfx.File == "" {
				continue
			}
			if !strings.HasSuffix(sfx.File, ".wav") {
				t.Errorf("sfx 文件 %q 不是 .wav 格式", sfx.File)
			}
		}
	}
}

// ═══════════════════════════════════════
// 5. Buff 规则一致性：代码中使用的 buff ID 都有对应规则
// ═══════════════════════════════════════

func TestConsistency_RequiredBuffRulesExist(t *testing.T) {
	rules := buff.GlobalRules()
	if rules == nil {
		t.Fatal("buff 规则未加载（GlobalRules 返回 nil）")
	}

	// 代码中使用的 buff ID（从 crowd_control.go, damage_pipeline.go, behaviors.go 等提取）
	requiredBuffIDs := []struct {
		id   string
		desc string
	}{
		{"stun", "眩晕 (crowd_control.go ApplyStun)"},
		{"slow", "减速 (crowd_control.go ApplySlow)"},
		{"root", "定身 (enemy.go IsRooted)"},
		{"bleed", "流血 (enemy.go IsBleeding)"},
		{"burn", "灼烧 (enemy.go IsBurning)"},
		{"poison", "中毒 (enemy.go IsPoisoned)"},
		{"weaken", "虚弱 (enemy.go IsWeakened, damage_pipeline.go)"},
		{"controlImmune", "控制免疫 (crowd_control.go ApplyControlImmunity)"},
		{"damageUp", "增伤 (buff stack rules)"},
		{"damageDown", "减伤 (buff stack rules)"},
		{"speedUp", "加速 (buff stack rules)"},
		{"dot", "持续伤害 (buff stack rules)"},
	}

	for _, tc := range requiredBuffIDs {
		t.Run(tc.id, func(t *testing.T) {
			if _, ok := rules[tc.id]; !ok {
				t.Errorf("代码使用的 buff ID %q (%s) 在 buff-stack.json rules 中无规则定义", tc.id, tc.desc)
			}
		})
	}
}

func TestConsistency_BuffRulesHaveValidModes(t *testing.T) {
	rules := buff.GlobalRules()
	if rules == nil {
		t.Fatal("buff 规则未加载")
	}

	validModes := map[buff.StackMode]bool{
		buff.Strongest:          true,
		buff.Additive:           true,
		buff.Multiplicative:     true,
		buff.Override:            true,
		buff.Independent:         true,
		buff.IndependentPerSource: true,
	}

	for id, rule := range rules {
		t.Run(id, func(t *testing.T) {
			if !validModes[rule.Mode] {
				t.Errorf("buff %q 的 mode %d 不在有效模式列表中", id, rule.Mode)
			}
		})
	}
}

func TestConsistency_SlowBuffHasReasonableCap(t *testing.T) {
	rules := buff.GlobalRules()
	if rules == nil {
		t.Fatal("buff 规则未加载")
	}
	slow, ok := rules["slow"]
	if !ok {
		t.Fatal("缺少 slow 规则")
	}
	if slow.Cap <= 0 || slow.Cap > 1.0 {
		t.Errorf("slow cap=%f 应在 (0, 1.0] 范围内", slow.Cap)
	}
}

// ═══════════════════════════════════════
// 6. 战灵一致性：wardens.json 中每个战灵有有效参数
// ═══════════════════════════════════════

func TestConsistency_WardenConfigsHaveRequiredFields(t *testing.T) {
	wardens, err := config.LoadWardenConfigs()
	if err != nil {
		t.Fatalf("加载战灵配置失败: %v", err)
	}
	if len(wardens) == 0 {
		t.Fatal("战灵配置为空")
	}

	for key, wc := range wardens {
		t.Run(key, func(t *testing.T) {
			if wc.Key == "" {
				t.Error("key 字段为空")
			}
			if wc.Name == "" {
				t.Error("name 字段为空")
			}
			if wc.Damage <= 0 {
				t.Errorf("damage=%.1f 应 > 0", wc.Damage)
			}
			if wc.AttackInterval <= 0 {
				t.Errorf("attackInterval=%.2f 应 > 0", wc.AttackInterval)
			}
			if wc.Range <= 0 {
				t.Errorf("range=%.0f 应 > 0", wc.Range)
			}
			if wc.MoveSpeed <= 0 {
				t.Errorf("moveSpeed=%.0f 应 > 0", wc.MoveSpeed)
			}
		})
	}
}

func TestConsistency_WardenConfigsHaveValidCategory(t *testing.T) {
	wardens, err := config.LoadWardenConfigs()
	if err != nil {
		t.Fatalf("加载战灵配置失败: %v", err)
	}

	validCategories := map[string]bool{
		"mobile": true, "indirect": true,
	}

	for key, wc := range wardens {
		t.Run(key, func(t *testing.T) {
			if !validCategories[wc.Category] {
				t.Errorf("战灵 %q 的 category %q 不在有效类别中", key, wc.Category)
			}
		})
	}
}

func TestConsistency_WardenConfigsHaveParams(t *testing.T) {
	wardens, err := config.LoadWardenConfigs()
	if err != nil {
		t.Fatalf("加载战灵配置失败: %v", err)
	}

	for key, wc := range wardens {
		t.Run(key, func(t *testing.T) {
			if len(wc.Params) == 0 {
				t.Errorf("战灵 %q 缺少 params 配置", key)
			}
			// 所有战灵都应有 orbitDist
			if _, ok := wc.Params["orbitDist"]; !ok {
				t.Errorf("战灵 %q 缺少 orbitDist 参数", key)
			}
		})
	}
}

// ═══════════════════════════════════════
// 7. 关卡列表与地图文件一致性
// ═══════════════════════════════════════

func TestConsistency_LevelListMapsCanLoad(t *testing.T) {
	levels, err := config.LoadLevelList()
	if err != nil {
		t.Fatalf("加载关卡列表失败: %v", err)
	}
	if len(levels) == 0 {
		t.Fatal("关卡列表为空")
	}

	for _, lv := range levels {
		t.Run(lv.ID, func(t *testing.T) {
			m, err := config.LoadMap(lv.ID)
			if err != nil {
				t.Fatalf("加载地图 %q 失败: %v", lv.ID, err)
			}
			if m.Waves <= 0 {
				t.Errorf("地图 %q 的 waves=%d 应 > 0", lv.ID, m.Waves)
			}
			if m.Cols <= 0 || m.Rows <= 0 {
				t.Errorf("地图 %q 的 cols=%d rows=%d 应 > 0", lv.ID, m.Cols, m.Rows)
			}
			if len(m.Grid) == 0 {
				t.Errorf("地图 %q 的 grid 为空", lv.ID)
			}
			if len(m.PathOrder) == 0 && len(m.PathOrders) == 0 {
				t.Errorf("地图 %q 缺少 pathOrder 或 pathOrders", lv.ID)
			}
		})
	}
}

func TestConsistency_LevelListWavesMatchMapWaves(t *testing.T) {
	levels, err := config.LoadLevelList()
	if err != nil {
		t.Fatalf("加载关卡列表失败: %v", err)
	}

	for _, lv := range levels {
		t.Run(lv.ID, func(t *testing.T) {
			m, err := config.LoadMap(lv.ID)
			if err != nil {
				t.Fatalf("加载地图 %q 失败: %v", lv.ID, err)
			}
			if lv.Waves != m.Waves {
				t.Errorf("关卡列表 waves=%d != 地图文件 waves=%d", lv.Waves, m.Waves)
			}
		})
	}
}

// ═══════════════════════════════════════
// 8. 综合：所有配置互不引用悬空 ID
// ═══════════════════════════════════════

func TestConsistency_NoOrphanEnemyAbilityDefs(t *testing.T) {
	// 确认 abilities.json 中的每个能力至少被一个原型引用，或能独立存在
	archetypes, err := config.LoadEnemyArchetypes()
	if err != nil {
		t.Fatalf("加载敌人原型失败: %v", err)
	}
	abilityDefs, err := config.LoadEnemyAbilities()
	if err != nil {
		t.Fatalf("加载敌人能力定义失败: %v", err)
	}

	usedAbilities := make(map[string]bool)
	for _, arch := range archetypes {
		for _, ref := range arch.Abilities {
			usedAbilities[ref.Type] = true
		}
	}

	orphanCount := 0
	for id := range abilityDefs {
		if !usedAbilities[id] {
			orphanCount++
			// 不报错，只记录——未使用的能力可能是预留的
			t.Logf("INFO: 敌人能力 %q 未被任何原型装配", id)
		}
	}
	t.Logf("统计: %d/%d 个敌人能力被使用, %d 个未使用",
		len(usedAbilities), len(abilityDefs), orphanCount)
}

// ── 辅助：确保 init() 已设置 DataFS 和 AssetFS ──
var _ = func() struct{} {
	config.SetDataFS(&defense2.DataFS)
	config.SetAssetFS(&defense2.AssetFS)
	_ = config.LoadBuffRules()
	_, _ = config.LoadAbilityTable()
	_, _ = config.LoadEnemyAbilities()
	return struct{}{}
}()
