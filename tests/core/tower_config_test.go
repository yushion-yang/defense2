package core_test

import (
	"testing"

	"defense2/internal/config"
)

func TestLoadTowersJSON(t *testing.T) {
	fd, err := config.LoadTowerFile("config/towers/towers.json")
	if err != nil {
		t.Fatalf("加载失败: %v", err)
	}
	if len(fd.Towers) < 1 {
		t.Fatalf("预期至少 1 座塔, 实际 %d", len(fd.Towers))
	}
	// 检查 basic 塔存在
	basic, ok := fd.Towers["basic"]
	if !ok {
		t.Fatal("应包含 basic 塔")
	}
	// basic 塔的 damage/range/attackSpeed 通过 tier-presets.json 动态生成（RollTowerStats），
	// JSON 中不含 baseRange 字段，所以 BaseRange=0 是预期行为。
	if basic.BuildCost <= 0 {
		t.Fatalf("basic 建造费用应 > 0, 实际 %d", basic.BuildCost)
	}
}

func TestLoadAllTowers(t *testing.T) {
	all, err := config.LoadAllTowers()
	if err != nil {
		t.Fatalf("加载全部塔失败: %v", err)
	}
	if len(all) < 1 {
		t.Fatalf("预期至少 1 座塔, 实际 %d", len(all))
	}
	if _, ok := all["basic"]; !ok {
		t.Fatal("应包含 basic 塔")
	}
}
