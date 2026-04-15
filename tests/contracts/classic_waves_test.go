// classic_waves_test.go — 经典模式出怪配置契约测试。
// 验证 config/systems/classic-waves/{mapID}.json 的完整性和一致性。
package contracts_test

import (
	"fmt"
	"testing"

	"defense2/internal/config"
)

// classicMapIDs 所有需要经典出怪配置的地图。
var classicMapIDs = []string{
	"map_01", "map_02", "map_03", "map_04",
	"map_05", "map_06", "map_07", "map_08",
}

// TestClassicWavesLoad 验证每张地图的经典出怪配置加载成功。
func TestClassicWavesLoad(t *testing.T) {
	for _, mapID := range classicMapIDs {
		t.Run(mapID, func(t *testing.T) {
			cwc, err := config.LoadClassicWavesConfig(mapID)
			if err != nil {
				t.Fatalf("LoadClassicWavesConfig(%s) 失败: %v", mapID, err)
			}
			if cwc.TotalWaves <= 0 {
				t.Errorf("totalWaves=%d, want > 0", cwc.TotalWaves)
			}
			if len(cwc.Waves) == 0 {
				t.Error("waves 数组为空")
			}
		})
	}
}

// TestClassicWavesCoverage 验证每波 1..totalWaves 都有配置条目。
func TestClassicWavesCoverage(t *testing.T) {
	for _, mapID := range classicMapIDs {
		t.Run(mapID, func(t *testing.T) {
			cwc, err := config.LoadClassicWavesConfig(mapID)
			if err != nil {
				t.Fatalf("加载失败: %v", err)
			}
			for w := 1; w <= cwc.TotalWaves; w++ {
				entry := cwc.GetWave(w)
				if entry == nil {
					t.Errorf("wave %d 缺少配置条目", w)
					continue
				}
				if len(entry.SpawnSeq()) == 0 {
					t.Errorf("wave %d spawnSeq 为空", w)
				}
			}
		})
	}
}

// TestClassicWavesArchetypesExist 验证所有引用的原型名存在于 enemies-core.json。
func TestClassicWavesArchetypesExist(t *testing.T) {
	archetypes, err := config.LoadEnemyArchetypes()
	if err != nil {
		t.Fatalf("LoadEnemyArchetypes() 失败: %v", err)
	}
	for _, mapID := range classicMapIDs {
		t.Run(mapID, func(t *testing.T) {
			cwc, err := config.LoadClassicWavesConfig(mapID)
			if err != nil {
				t.Fatalf("加载失败: %v", err)
			}
			for _, entry := range cwc.Waves {
				for i, ee := range entry.SpawnSeq() {
					if _, ok := archetypes[ee.Archetype]; !ok {
						t.Errorf("wave %d spawnSeq[%d]: 原型 %q 不存在", entry.Wave, i, ee.Archetype)
					}
				}
				if entry.Boss != nil {
					if _, ok := archetypes[entry.Boss.Archetype]; !ok {
						t.Errorf("wave %d boss: 原型 %q 不存在", entry.Wave, entry.Boss.Archetype)
					}
				}
			}
		})
	}
}

// TestClassicWavesBuffsValid 验证 buff 索引不越界且 buff ID 合法。
// 支持两种格式：Buffs（全局索引）和 PathBuffs（路径内局部索引）。
func TestClassicWavesBuffsValid(t *testing.T) {
	config.LoadEnemyAbilities()
	abilTable := config.GlobalEnemyAbilityTable()

	checkBuffID := func(t *testing.T, wave int, context, bid string) {
		t.Helper()
		if abilTable != nil && abilTable[bid] == nil {
			t.Errorf("wave %d %s: buff ID %q 不存在", wave, context, bid)
		}
	}

	for _, mapID := range classicMapIDs {
		t.Run(mapID, func(t *testing.T) {
			cwc, err := config.LoadClassicWavesConfig(mapID)
			if err != nil {
				t.Fatalf("加载失败: %v", err)
			}
			for _, entry := range cwc.Waves {
				// 检查 Buffs（单路径格式：全局索引）
				seqLen := len(entry.SpawnSeq())
				for idxStr, buffIDs := range entry.Buffs {
					var idx int
					if _, err := fmt.Sscanf(idxStr, "%d", &idx); err != nil {
						t.Errorf("wave %d buffs key %q 不是合法整数", entry.Wave, idxStr)
						continue
					}
					if idx < 0 || idx >= seqLen {
						t.Errorf("wave %d buffs[%d]: 索引越界 (spawnSeq 长度=%d)", entry.Wave, idx, seqLen)
					}
					for _, bid := range buffIDs {
						checkBuffID(t, entry.Wave, fmt.Sprintf("buffs[%d]", idx), bid)
					}
				}
				// 检查 PathBuffs（多路径格式：路径内局部索引）
				for pid, pathMap := range entry.PathBuffs {
					pathEnemies := entry.Paths[pid]
					pathLen := len(pathEnemies)
					for idxStr, buffIDs := range pathMap {
						var idx int
						if _, err := fmt.Sscanf(idxStr, "%d", &idx); err != nil {
							t.Errorf("wave %d pathBuffs[%s] key %q 不是合法整数", entry.Wave, pid, idxStr)
							continue
						}
						if idx < 0 || idx >= pathLen {
							t.Errorf("wave %d pathBuffs[%s][%d]: 索引越界 (path 长度=%d)", entry.Wave, pid, idx, pathLen)
						}
						for _, bid := range buffIDs {
							checkBuffID(t, entry.Wave, fmt.Sprintf("pathBuffs[%s][%d]", pid, idx), bid)
						}
					}
				}
			}
		})
	}
}

// TestClassicWavesScalingValid 验证每张地图的缩放参数合理。
func TestClassicWavesScalingValid(t *testing.T) {
	for _, mapID := range classicMapIDs {
		t.Run(mapID, func(t *testing.T) {
			cwc, err := config.LoadClassicWavesConfig(mapID)
			if err != nil {
				t.Fatalf("加载失败: %v", err)
			}
			s := cwc.Scaling
			if s.HpBase <= 0 {
				t.Errorf("scaling.hpBase=%.1f, want > 0", s.HpBase)
			}
			if s.HpPerWave <= 0 {
				t.Errorf("scaling.hpPerWave=%.1f, want > 0", s.HpPerWave)
			}
			if s.SpeedBase <= 0 {
				t.Errorf("scaling.speedBase=%.1f, want > 0", s.SpeedBase)
			}
			if s.SpawnInterval <= 0 {
				t.Errorf("scaling.spawnInterval=%.1f, want > 0", s.SpawnInterval)
			}
		})
	}
}
