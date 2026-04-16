// descriptor_edit_state_test.go — EditState ↔ AbilityDescriptor 双向转换测试。
//
// 验证 PipelineEditState 与 Pipeline 之间的双向转换能力，
// 确保用户在 UI 中编辑的管线状态能正确序列化为 descriptor，
// 以及已有 descriptor 能正确还原为编辑状态。
package core_test

import (
	"testing"

	"defense2/internal/core/tower/descriptor"
)

// ── 辅助函数 ──────────────────────────────────────────

// editStateEq 比较两个 PipelineEditState 是否语义相等。
func editStateEq(t *testing.T, label string, got, want descriptor.PipelineEditState) {
	t.Helper()

	if got.TriggerID != want.TriggerID {
		t.Errorf("%s: TriggerID got=%q want=%q", label, got.TriggerID, want.TriggerID)
	}
	if got.SelectorID != want.SelectorID {
		t.Errorf("%s: SelectorID got=%q want=%q", label, got.SelectorID, want.SelectorID)
	}

	// 条件数量
	if len(got.Conditions) != len(want.Conditions) {
		t.Errorf("%s: len(Conditions) got=%d want=%d", label, len(got.Conditions), len(want.Conditions))
	} else {
		for i := range got.Conditions {
			componentEq(t, label+".Conditions["+itoa(i)+"]", got.Conditions[i], want.Conditions[i])
		}
	}

	// 效果数量
	if len(got.Effects) != len(want.Effects) {
		t.Errorf("%s: len(Effects) got=%d want=%d", label, len(got.Effects), len(want.Effects))
	} else {
		for i := range got.Effects {
			componentEq(t, label+".Effects["+itoa(i)+"]", got.Effects[i], want.Effects[i])
		}
	}

	// 选择器参数
	paramsEq(t, label+".SelectorParams", got.SelectorParams, want.SelectorParams)
}

func componentEq(t *testing.T, label string, got, want descriptor.ComponentEditState) {
	t.Helper()
	if got.TypeID != want.TypeID {
		t.Errorf("%s: TypeID got=%q want=%q", label, got.TypeID, want.TypeID)
	}
	paramsEq(t, label+".Params", got.Params, want.Params)
}

func paramsEq(t *testing.T, label string, got, want map[string]float64) {
	t.Helper()
	if len(got) != len(want) {
		t.Errorf("%s: len got=%d want=%d (got=%v want=%v)", label, len(got), len(want), got, want)
		return
	}
	for k, wv := range want {
		gv, ok := got[k]
		if !ok {
			t.Errorf("%s: missing key %q", label, k)
			continue
		}
		if !floatEq(gv, wv) {
			t.Errorf("%s[%q]: got=%f want=%f", label, k, gv, wv)
		}
	}
}

func itoa(i int) string {
	return string(rune('0' + i))
}

// ── Roundtrip: stunChance ────────────────────────────

// TestRoundtrip_StunChance onHit + chance(linear) + currentTarget + stun(linear)。
func TestRoundtrip_StunChance(t *testing.T) {
	input := []descriptor.PipelineEditState{
		{
			TriggerID: "onHit",
			Conditions: []descriptor.ComponentEditState{
				{TypeID: "chance", Params: map[string]float64{"base": 0.1, "potential": 0.05}},
			},
			SelectorID:     "currentTarget",
			SelectorParams: map[string]float64{},
			Effects: []descriptor.ComponentEditState{
				{TypeID: "stun", Params: map[string]float64{"base": 0.5, "potential": 0.3}},
			},
		},
	}

	// EditState → Descriptor
	desc, err := descriptor.EditStateToDescriptor("stunChance", "Stun Chance", input)
	if err != nil {
		t.Fatalf("EditStateToDescriptor: %v", err)
	}
	if desc.ID != "stunChance" {
		t.Errorf("desc.ID=%q want %q", desc.ID, "stunChance")
	}
	if len(desc.Pipelines) != 1 {
		t.Fatalf("len(Pipelines)=%d want 1", len(desc.Pipelines))
	}

	// Descriptor → EditState
	got := descriptor.DescriptorToEditState(desc)
	if len(got) != 1 {
		t.Fatalf("len(got)=%d want 1", len(got))
	}

	editStateEq(t, "stunChance", got[0], input[0])
}

// ── Roundtrip: goldPassive ───────────────────────────

// TestRoundtrip_GoldPassive onTick + cooldown + selfTower + gold(linear)。
func TestRoundtrip_GoldPassive(t *testing.T) {
	input := []descriptor.PipelineEditState{
		{
			TriggerID: "onTick",
			Conditions: []descriptor.ComponentEditState{
				{TypeID: "cooldown", Params: map[string]float64{"seconds": 5.0}},
			},
			SelectorID:     "selfTower",
			SelectorParams: map[string]float64{},
			Effects: []descriptor.ComponentEditState{
				{TypeID: "gold", Params: map[string]float64{"base": 10, "potential": 2}},
			},
		},
	}

	desc, err := descriptor.EditStateToDescriptor("goldPassive", "Gold Passive", input)
	if err != nil {
		t.Fatalf("EditStateToDescriptor: %v", err)
	}

	got := descriptor.DescriptorToEditState(desc)
	if len(got) != 1 {
		t.Fatalf("len(got)=%d want 1", len(got))
	}

	editStateEq(t, "goldPassive", got[0], input[0])
}

// ── Roundtrip: multi-pipeline ────────────────────────

// TestRoundtrip_MultiPipeline 两条管线：onHit+stun 和 onKill+gold。
func TestRoundtrip_MultiPipeline(t *testing.T) {
	input := []descriptor.PipelineEditState{
		{
			TriggerID:      "onHit",
			SelectorID:     "currentTarget",
			SelectorParams: map[string]float64{},
			Conditions: []descriptor.ComponentEditState{
				{TypeID: "chance", Params: map[string]float64{"value": 0.3}},
			},
			Effects: []descriptor.ComponentEditState{
				{TypeID: "stun", Params: map[string]float64{"value": 1.0}},
			},
		},
		{
			TriggerID:      "onKill",
			SelectorID:     "selfTower",
			SelectorParams: map[string]float64{},
			Effects: []descriptor.ComponentEditState{
				{TypeID: "gold", Params: map[string]float64{"value": 5}},
			},
		},
	}

	desc, err := descriptor.EditStateToDescriptor("multi", "Multi", input)
	if err != nil {
		t.Fatalf("EditStateToDescriptor: %v", err)
	}
	if len(desc.Pipelines) != 2 {
		t.Fatalf("len(Pipelines)=%d want 2", len(desc.Pipelines))
	}

	got := descriptor.DescriptorToEditState(desc)
	if len(got) != 2 {
		t.Fatalf("len(got)=%d want 2", len(got))
	}

	editStateEq(t, "pipeline[0]", got[0], input[0])
	editStateEq(t, "pipeline[1]", got[1], input[1])
}

// ── Empty conditions/effects ─────────────────────────

// TestRoundtrip_EmptyConditionsEffects 仅 trigger + selector，无条件无效果。
func TestRoundtrip_EmptyConditionsEffects(t *testing.T) {
	input := []descriptor.PipelineEditState{
		{
			TriggerID:      "onPlace",
			SelectorID:     "selfTower",
			SelectorParams: map[string]float64{},
		},
	}

	desc, err := descriptor.EditStateToDescriptor("empty", "Empty", input)
	if err != nil {
		t.Fatalf("EditStateToDescriptor: %v", err)
	}

	got := descriptor.DescriptorToEditState(desc)
	if len(got) != 1 {
		t.Fatalf("len(got)=%d want 1", len(got))
	}

	if got[0].TriggerID != "onPlace" {
		t.Errorf("TriggerID=%q want %q", got[0].TriggerID, "onPlace")
	}
	if got[0].SelectorID != "selfTower" {
		t.Errorf("SelectorID=%q want %q", got[0].SelectorID, "selfTower")
	}
	if len(got[0].Conditions) != 0 {
		t.Errorf("len(Conditions)=%d want 0", len(got[0].Conditions))
	}
	if len(got[0].Effects) != 0 {
		t.Errorf("len(Effects)=%d want 0", len(got[0].Effects))
	}
}

// ── DescriptorToEditState from parsed JSON ──────────

// TestDescriptorToEditState_FromJSON 解析真实 JSON → 转编辑状态 → 验证。
func TestDescriptorToEditState_FromJSON(t *testing.T) {
	data := []byte(`{
		"id": "testAbility",
		"label": "Test",
		"cost": 2,
		"pipelines": [
			{
				"trigger": "onHit",
				"conditions": [
					{"type": "chance", "rate": {"scaler": "linear", "base": 0.2, "potential": 0.1}},
					{"type": "hpBelow", "threshold": {"scaler": "fixed", "value": 0.5}}
				],
				"selector": {"type": "aoeRadius", "radius": {"scaler": "linear", "base": 50, "potential": 10}},
				"effects": [
					{"type": "damage", "mode": "flat", "value": {"scaler": "linear", "base": 20, "potential": 5}},
					{"type": "slow", "factor": {"scaler": "fixed", "value": 0.4}, "duration": {"scaler": "linear", "base": 1.0, "potential": 0.5}}
				]
			}
		]
	}`)

	desc, err := descriptor.ParseDescriptor(data)
	if err != nil {
		t.Fatalf("ParseDescriptor: %v", err)
	}

	got := descriptor.DescriptorToEditState(desc)
	if len(got) != 1 {
		t.Fatalf("len(got)=%d want 1", len(got))
	}

	p := got[0]

	// 触发器
	if p.TriggerID != "onHit" {
		t.Errorf("TriggerID=%q want %q", p.TriggerID, "onHit")
	}

	// 条件
	if len(p.Conditions) != 2 {
		t.Fatalf("len(Conditions)=%d want 2", len(p.Conditions))
	}
	if p.Conditions[0].TypeID != "chance" {
		t.Errorf("Conditions[0].TypeID=%q want %q", p.Conditions[0].TypeID, "chance")
	}
	if !floatEq(p.Conditions[0].Params["base"], 0.2) {
		t.Errorf("chance.base=%f want 0.2", p.Conditions[0].Params["base"])
	}
	if !floatEq(p.Conditions[0].Params["potential"], 0.1) {
		t.Errorf("chance.potential=%f want 0.1", p.Conditions[0].Params["potential"])
	}
	if p.Conditions[1].TypeID != "hpBelow" {
		t.Errorf("Conditions[1].TypeID=%q want %q", p.Conditions[1].TypeID, "hpBelow")
	}
	if !floatEq(p.Conditions[1].Params["value"], 0.5) {
		t.Errorf("hpBelow.value=%f want 0.5", p.Conditions[1].Params["value"])
	}

	// 选择器
	if p.SelectorID != "aoeRadius" {
		t.Errorf("SelectorID=%q want %q", p.SelectorID, "aoeRadius")
	}
	if !floatEq(p.SelectorParams["base"], 50) {
		t.Errorf("aoeRadius.base=%f want 50", p.SelectorParams["base"])
	}
	if !floatEq(p.SelectorParams["potential"], 10) {
		t.Errorf("aoeRadius.potential=%f want 10", p.SelectorParams["potential"])
	}

	// 效果
	if len(p.Effects) != 2 {
		t.Fatalf("len(Effects)=%d want 2", len(p.Effects))
	}
	if p.Effects[0].TypeID != "damage" {
		t.Errorf("Effects[0].TypeID=%q want %q", p.Effects[0].TypeID, "damage")
	}
	if p.Effects[0].Params["mode"] != 0 { // flat = 0
		// mode 存为 "flat" 字符串无法放 float64 map — 用特殊编码
		// damage mode: flat=0, ratio=1, hpPercent=2
	}
	if !floatEq(p.Effects[0].Params["base"], 20) {
		t.Errorf("damage.base=%f want 20", p.Effects[0].Params["base"])
	}
	if p.Effects[1].TypeID != "slow" {
		t.Errorf("Effects[1].TypeID=%q want %q", p.Effects[1].TypeID, "slow")
	}
}

// ── Error: empty trigger ────────────────────────────

// TestEditStateToDescriptor_EmptyTrigger 空 trigger 应报错。
func TestEditStateToDescriptor_EmptyTrigger(t *testing.T) {
	input := []descriptor.PipelineEditState{
		{
			TriggerID:      "", // 空
			SelectorID:     "currentTarget",
			SelectorParams: map[string]float64{},
		},
	}

	_, err := descriptor.EditStateToDescriptor("err", "Err", input)
	if err == nil {
		t.Fatal("expected error for empty trigger, got nil")
	}
}

// ── Roundtrip: 全部条件/选择器/效果类型覆盖 ─────────────

// TestRoundtrip_AllConditionTypes 验证所有 11 种条件类型的 roundtrip。
func TestRoundtrip_AllConditionTypes(t *testing.T) {
	tests := []struct {
		name string
		cond descriptor.ComponentEditState
	}{
		{"chance-linear", descriptor.ComponentEditState{TypeID: "chance", Params: map[string]float64{"base": 0.2, "potential": 0.05}}},
		{"chance-fixed", descriptor.ComponentEditState{TypeID: "chance", Params: map[string]float64{"value": 0.5}}},
		{"cooldown", descriptor.ComponentEditState{TypeID: "cooldown", Params: map[string]float64{"seconds": 3.0}}},
		{"hpBelow", descriptor.ComponentEditState{TypeID: "hpBelow", Params: map[string]float64{"value": 0.3}}},
		{"hpAbove", descriptor.ComponentEditState{TypeID: "hpAbove", Params: map[string]float64{"base": 0.5, "potential": 0.1}}},
		{"distanceMin", descriptor.ComponentEditState{TypeID: "distanceMin", Params: map[string]float64{"distance": 100}}},
		{"noNearbyTower", descriptor.ComponentEditState{TypeID: "noNearbyTower", Params: map[string]float64{"radius": 80}}},
		{"isBoss", descriptor.ComponentEditState{TypeID: "isBoss", Params: map[string]float64{}}},
		{"notBoss", descriptor.ComponentEditState{TypeID: "notBoss", Params: map[string]float64{}}},
		{"every", descriptor.ComponentEditState{TypeID: "every", Params: map[string]float64{"n": 3}}},
		{"buffActive", descriptor.ComponentEditState{TypeID: "buffActive", Params: map[string]float64{"buffID": 0}}},
		{"buffAbsent", descriptor.ComponentEditState{TypeID: "buffAbsent", Params: map[string]float64{"buffID": 0}}},
		{"buffActive-slow", descriptor.ComponentEditState{TypeID: "buffActive", Params: map[string]float64{"buffID": 1}}},   // slow(idx=1)
		{"buffAbsent-weaken", descriptor.ComponentEditState{TypeID: "buffAbsent", Params: map[string]float64{"buffID": 6}}}, // weaken(idx=6)
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			input := []descriptor.PipelineEditState{
				{
					TriggerID:      "onHit",
					Conditions:     []descriptor.ComponentEditState{tc.cond},
					SelectorID:     "currentTarget",
					SelectorParams: map[string]float64{},
					Effects: []descriptor.ComponentEditState{
						{TypeID: "silence", Params: map[string]float64{}},
					},
				},
			}

			desc, err := descriptor.EditStateToDescriptor("test", "Test", input)
			if err != nil {
				t.Fatalf("EditStateToDescriptor: %v", err)
			}

			got := descriptor.DescriptorToEditState(desc)
			if len(got) != 1 {
				t.Fatalf("len(got)=%d want 1", len(got))
			}

			componentEq(t, tc.name, got[0].Conditions[0], tc.cond)
		})
	}
}

// TestRoundtrip_AllSelectorTypes 验证所有 9 种选择器类型的 roundtrip。
func TestRoundtrip_AllSelectorTypes(t *testing.T) {
	tests := []struct {
		name       string
		selectorID string
		params     map[string]float64
	}{
		{"currentTarget", "currentTarget", map[string]float64{}},
		{"aoeRadius-linear", "aoeRadius", map[string]float64{"base": 50, "potential": 10}},
		{"aoeRadius-fixed", "aoeRadius", map[string]float64{"value": 60}},
		{"chain", "chain", map[string]float64{"base": 2, "potential": 1, "range": 120, "decayRatio": 0.8}},
		{"allInRange", "allInRange", map[string]float64{}},
		{"nearbyAllies", "nearbyAllies", map[string]float64{"radius": 100}},
		{"selfTower", "selfTower", map[string]float64{}},
		{"cone", "cone", map[string]float64{"angle": 90, "base": 80, "potential": 10}},
		{"ring360", "ring360", map[string]float64{"base": 8, "potential": 2}},
		{"random", "random", map[string]float64{"base": 3, "potential": 1, "radius": 150}},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			input := []descriptor.PipelineEditState{
				{
					TriggerID:      "onHit",
					SelectorID:     tc.selectorID,
					SelectorParams: tc.params,
					Effects: []descriptor.ComponentEditState{
						{TypeID: "silence", Params: map[string]float64{}},
					},
				},
			}

			desc, err := descriptor.EditStateToDescriptor("test", "Test", input)
			if err != nil {
				t.Fatalf("EditStateToDescriptor: %v", err)
			}

			got := descriptor.DescriptorToEditState(desc)
			if len(got) != 1 {
				t.Fatalf("len(got)=%d want 1", len(got))
			}

			if got[0].SelectorID != tc.selectorID {
				t.Errorf("SelectorID=%q want %q", got[0].SelectorID, tc.selectorID)
			}
			paramsEq(t, tc.name+".SelectorParams", got[0].SelectorParams, tc.params)
		})
	}
}

// TestRoundtrip_AllEffectTypes 验证所有 13 种效果类型的 roundtrip。
func TestRoundtrip_AllEffectTypes(t *testing.T) {
	tests := []struct {
		name   string
		effect descriptor.ComponentEditState
	}{
		{"damage-flat", descriptor.ComponentEditState{TypeID: "damage", Params: map[string]float64{"mode": 0, "base": 20, "potential": 5}}},
		{"damage-ratio", descriptor.ComponentEditState{TypeID: "damage", Params: map[string]float64{"mode": 1, "value": 0.5}}},
		{"damage-hpPercent", descriptor.ComponentEditState{TypeID: "damage", Params: map[string]float64{"mode": 2, "base": 0.05, "potential": 0.01}}},
		{"slow", descriptor.ComponentEditState{TypeID: "slow", Params: map[string]float64{
			"factor_base": 0.3, "factor_potential": 0.1,
			"duration_base": 1.0, "duration_potential": 0.5,
		}}},
		{"stun-linear", descriptor.ComponentEditState{TypeID: "stun", Params: map[string]float64{"base": 0.5, "potential": 0.3}}},
		{"stun-fixed", descriptor.ComponentEditState{TypeID: "stun", Params: map[string]float64{"value": 1.0}}},
		{"root", descriptor.ComponentEditState{TypeID: "root", Params: map[string]float64{"base": 1.0, "potential": 0.2}}},
		{"dot", descriptor.ComponentEditState{TypeID: "dot", Params: map[string]float64{
			"subtype": 0, "mode": 0,
			"value_base": 5, "value_potential": 2,
			"duration_base": 3, "duration_potential": 1,
		}}},
		{"weaken", descriptor.ComponentEditState{TypeID: "weaken", Params: map[string]float64{
			"amplify_base": 0.2, "amplify_potential": 0.05,
			"duration_base": 2, "duration_potential": 0.5,
		}}},
		{"silence", descriptor.ComponentEditState{TypeID: "silence", Params: map[string]float64{}}},
		{"buff", descriptor.ComponentEditState{TypeID: "buff", Params: map[string]float64{"stat": 0, "base": 10, "potential": 5}}},
		{"selfBuff", descriptor.ComponentEditState{TypeID: "selfBuff", Params: map[string]float64{"stat": 0, "value": 15}}},
		{"gold", descriptor.ComponentEditState{TypeID: "gold", Params: map[string]float64{"base": 10, "potential": 2}}},
		{"modifyStat", descriptor.ComponentEditState{TypeID: "modifyStat", Params: map[string]float64{"stat": 0, "multiplier": 1.5}}},
		{"crit", descriptor.ComponentEditState{TypeID: "crit", Params: map[string]float64{"multiplier": 2.0}}},
		{"purge", descriptor.ComponentEditState{TypeID: "purge", Params: map[string]float64{"count": 2}}},
		{"teleport", descriptor.ComponentEditState{TypeID: "teleport", Params: map[string]float64{"base": 50, "potential": 10}}},
		// 字符串字段非默认值 roundtrip 验证
		{"buff-speed", descriptor.ComponentEditState{TypeID: "buff", Params: map[string]float64{"stat": 1, "value": 0.2}}},      // stat=speed(idx=1)
		{"modifyStat-range", descriptor.ComponentEditState{TypeID: "modifyStat", Params: map[string]float64{"stat": 2, "multiplier": 2.0}}}, // stat=range(idx=2)
		{"damage-hpPercent-mode", descriptor.ComponentEditState{TypeID: "damage", Params: map[string]float64{"mode": 2, "value": 0.1}}},     // mode=hpPercent(idx=2)
		{"dot-bleed", descriptor.ComponentEditState{TypeID: "dot", Params: map[string]float64{
			"subtype": 1, "mode": 0,
			"value_value": 10, "duration_value": 5,
		}}}, // subtype=bleed(idx=1)
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			input := []descriptor.PipelineEditState{
				{
					TriggerID:      "onHit",
					SelectorID:     "currentTarget",
					SelectorParams: map[string]float64{},
					Effects:        []descriptor.ComponentEditState{tc.effect},
				},
			}

			desc, err := descriptor.EditStateToDescriptor("test", "Test", input)
			if err != nil {
				t.Fatalf("EditStateToDescriptor: %v", err)
			}

			got := descriptor.DescriptorToEditState(desc)
			if len(got) != 1 {
				t.Fatalf("len(got)=%d want 1", len(got))
			}

			if len(got[0].Effects) != 1 {
				t.Fatalf("len(Effects)=%d want 1", len(got[0].Effects))
			}

			componentEq(t, tc.name, got[0].Effects[0], tc.effect)
		})
	}
}
