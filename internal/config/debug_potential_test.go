package config

import (
	"fmt"
	"testing"
)

func TestDebug_AbilityPotentialPopulated(t *testing.T) {
	// 模拟真实的 abilities.json 数据
	enemyAbilityTable = map[string]*EnemyAbilityDef{
		"armorPlating": {Type: "armorPlating", Base: 10, Potential: 2, Param: 0},
		"damageCap":    {Type: "damageCap", Base: 60, Potential: 4, Param: 0},
		"evasion":      {Type: "evasion", Base: 0.3, Potential: 0, Param: 0},
	}
	defer func() { enemyAbilityTable = nil }()

	// 模拟 enemies-core.json 中的字符串引用
	refs := []EnemyAbilityRef{
		{Type: "armorPlating"}, // 字符串引用，Potential=0（未覆盖）
	}

	for _, ref := range refs {
		// 1. ResolveEnemyAbility（wave=0 路径）
		def0 := ResolveEnemyAbility(ref)
		fmt.Printf("ResolveEnemyAbility(%s): base=%.1f\n", ref.Type, def0.Base)

		// 2. ResolveEffectivePotential
		p := ResolveEffectivePotential(ref)
		fmt.Printf("ResolveEffectivePotential(%s): %.1f\n", ref.Type, p)
		if ref.Type == "armorPlating" && p != 2 {
			t.Errorf("armorPlating potential should be 2, got %.1f", p)
		}

		// 3. ResolveEnemyAbilityAtWave（wave=10）
		def10 := ResolveEnemyAbilityAtWave(ref, 10)
		fmt.Printf("ResolveEnemyAbilityAtWave(%s, wave=10): base=%.1f\n", ref.Type, def10.Base)
		if ref.Type == "armorPlating" && def10.Base != 30 { // 10 + 2*10
			t.Errorf("armorPlating at wave 10 should be 30, got %.1f", def10.Base)
		}
	}

	// 测试 evasion（potential=0）不应缩放
	evasionP := ResolveEffectivePotential(EnemyAbilityRef{Type: "evasion"})
	if evasionP != 0 {
		t.Errorf("evasion potential should be 0, got %.1f", evasionP)
	}
}
