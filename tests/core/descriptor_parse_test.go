// descriptor_parse_test.go — AbilityDescriptor JSON 解析测试。
//
// 验证 ParseDescriptor 能正确将 JSON 解析为含预编译管线的 AbilityDescriptor，
// 包括各类型的条件/选择器/效果的分派解析，以及错误输入的处理。
package core_test

import (
	"math"
	"strings"
	"testing"

	"defense2/internal/core/tower/descriptor"
)

// floatEq 浮点近似相等比较，容差 1e-9。
func floatEq(a, b float64) bool {
	return math.Abs(a-b) < 1e-9
}

// TestParseDescriptor_StunChance 解析 onHit + chance + currentTarget + stun 管线。
func TestParseDescriptor_StunChance(t *testing.T) {
	data := []byte(`{
		"id": "stunChance",
		"label": "Stun Chance",
		"cost": 3,
		"tags": ["cc"],
		"pipelines": [
			{
				"trigger": "onHit",
				"conditions": [
					{"type": "chance", "rate": {"scaler": "linear", "base": 0.1, "potential": 0.05}}
				],
				"selector": {"type": "currentTarget"},
				"effects": [
					{"type": "stun", "duration": {"scaler": "fixed", "value": 0.5}}
				]
			}
		]
	}`)

	desc, err := descriptor.ParseDescriptor(data)
	if err != nil {
		t.Fatalf("ParseDescriptor() error: %v", err)
	}

	// 顶层字段
	if desc.ID != "stunChance" {
		t.Errorf("ID = %q, want %q", desc.ID, "stunChance")
	}
	if desc.Label != "Stun Chance" {
		t.Errorf("Label = %q, want %q", desc.Label, "Stun Chance")
	}
	if desc.Cost != 3 {
		t.Errorf("Cost = %d, want %d", desc.Cost, 3)
	}
	if len(desc.Tags) != 1 || desc.Tags[0] != "cc" {
		t.Errorf("Tags = %v, want [cc]", desc.Tags)
	}

	// 管线数量
	if len(desc.Pipelines) != 1 {
		t.Fatalf("Pipelines count = %d, want 1", len(desc.Pipelines))
	}

	p := desc.Pipelines[0]

	// Trigger
	if p.Trigger != descriptor.TriggerOnHit {
		t.Errorf("Trigger = %v, want onHit", p.Trigger)
	}

	// Conditions
	if len(p.Conditions) != 1 {
		t.Fatalf("Conditions count = %d, want 1", len(p.Conditions))
	}
	chance, ok := p.Conditions[0].(descriptor.ChanceCondition)
	if !ok {
		t.Fatalf("Conditions[0] type = %T, want ChanceCondition", p.Conditions[0])
	}
	// strength=100 → rate = 0.1 + 0.05*(100/100) = 0.15
	got := chance.Rate.Calc(100)
	if !floatEq(got, 0.15) {
		t.Errorf("chance.Rate.Calc(100) = %v, want 0.15", got)
	}

	// Selector
	if _, ok := p.Selector.(descriptor.CurrentTargetSelector); !ok {
		t.Errorf("Selector type = %T, want CurrentTargetSelector", p.Selector)
	}

	// Effects
	if len(p.Effects) != 1 {
		t.Fatalf("Effects count = %d, want 1", len(p.Effects))
	}
	stun, ok := p.Effects[0].(descriptor.StunEffect)
	if !ok {
		t.Fatalf("Effects[0] type = %T, want StunEffect", p.Effects[0])
	}
	if stun.Duration.Calc(100) != 0.5 {
		t.Errorf("stun.Duration.Calc(100) = %v, want 0.5", stun.Duration.Calc(100))
	}
}

// TestParseDescriptor_Splash 解析 onHit + 无条件 + aoeRadius + damage ratio 管线。
func TestParseDescriptor_Splash(t *testing.T) {
	data := []byte(`{
		"id": "splash",
		"label": "Splash Damage",
		"cost": 2,
		"pipelines": [
			{
				"trigger": "onHit",
				"selector": {"type": "aoeRadius", "radius": {"scaler": "linear", "base": 50, "potential": 10}},
				"effects": [
					{"type": "damage", "mode": "ratio", "value": {"scaler": "fixed", "value": 0.5}}
				]
			}
		]
	}`)

	desc, err := descriptor.ParseDescriptor(data)
	if err != nil {
		t.Fatalf("ParseDescriptor() error: %v", err)
	}

	if desc.ID != "splash" {
		t.Errorf("ID = %q, want %q", desc.ID, "splash")
	}

	p := desc.Pipelines[0]

	// 无条件
	if len(p.Conditions) != 0 {
		t.Errorf("Conditions count = %d, want 0", len(p.Conditions))
	}

	// AoeRadiusSelector
	aoe, ok := p.Selector.(descriptor.AoeRadiusSelector)
	if !ok {
		t.Fatalf("Selector type = %T, want AoeRadiusSelector", p.Selector)
	}
	// strength=100 → radius = 50 + 10*(100/100) = 60
	if aoe.Radius.Calc(100) != 60 {
		t.Errorf("aoe.Radius.Calc(100) = %v, want 60", aoe.Radius.Calc(100))
	}

	// DamageEffect
	dmg, ok := p.Effects[0].(descriptor.DamageEffect)
	if !ok {
		t.Fatalf("Effects[0] type = %T, want DamageEffect", p.Effects[0])
	}
	if dmg.Mode != descriptor.DmgRatio {
		t.Errorf("dmg.Mode = %v, want DmgRatio", dmg.Mode)
	}
	if dmg.Value.Calc(100) != 0.5 {
		t.Errorf("dmg.Value.Calc(100) = %v, want 0.5", dmg.Value.Calc(100))
	}
}

// TestParseDescriptor_MultiPipeline 解析含 2 条管线的描述符（slow + burn）。
func TestParseDescriptor_MultiPipeline(t *testing.T) {
	data := []byte(`{
		"id": "slowBurn",
		"label": "Slow & Burn",
		"cost": 4,
		"tags": ["cc", "dot"],
		"pipelines": [
			{
				"trigger": "onHit",
				"conditions": [
					{"type": "chance", "rate": {"scaler": "fixed", "value": 0.3}}
				],
				"selector": {"type": "currentTarget"},
				"effects": [
					{"type": "slow", "factor": {"scaler": "fixed", "value": 0.4}, "duration": {"scaler": "fixed", "value": 2.0}}
				]
			},
			{
				"trigger": "onHit",
				"selector": {"type": "currentTarget"},
				"effects": [
					{"type": "dot", "subtype": "burn", "mode": "flat", "value": {"scaler": "linear", "base": 5, "potential": 2}, "duration": {"scaler": "fixed", "value": 3.0}}
				]
			}
		]
	}`)

	desc, err := descriptor.ParseDescriptor(data)
	if err != nil {
		t.Fatalf("ParseDescriptor() error: %v", err)
	}

	if len(desc.Pipelines) != 2 {
		t.Fatalf("Pipelines count = %d, want 2", len(desc.Pipelines))
	}

	// 管线 0: slow
	p0 := desc.Pipelines[0]
	if len(p0.Conditions) != 1 {
		t.Errorf("Pipeline[0] conditions count = %d, want 1", len(p0.Conditions))
	}
	slow, ok := p0.Effects[0].(descriptor.SlowEffect)
	if !ok {
		t.Fatalf("Pipeline[0].Effects[0] type = %T, want SlowEffect", p0.Effects[0])
	}
	if slow.Factor.Calc(100) != 0.4 {
		t.Errorf("slow.Factor.Calc(100) = %v, want 0.4", slow.Factor.Calc(100))
	}
	if slow.Duration.Calc(100) != 2.0 {
		t.Errorf("slow.Duration.Calc(100) = %v, want 2.0", slow.Duration.Calc(100))
	}

	// 管线 1: burn dot
	p1 := desc.Pipelines[1]
	if len(p1.Conditions) != 0 {
		t.Errorf("Pipeline[1] conditions count = %d, want 0", len(p1.Conditions))
	}
	dot, ok := p1.Effects[0].(descriptor.DotEffect)
	if !ok {
		t.Fatalf("Pipeline[1].Effects[0] type = %T, want DotEffect", p1.Effects[0])
	}
	if dot.Subtype != "burn" {
		t.Errorf("dot.Subtype = %q, want %q", dot.Subtype, "burn")
	}
	if dot.Mode != descriptor.DmgFlat {
		t.Errorf("dot.Mode = %v, want DmgFlat", dot.Mode)
	}
	// strength=100 → value = 5 + 2*(100/100) = 7
	if dot.Value.Calc(100) != 7 {
		t.Errorf("dot.Value.Calc(100) = %v, want 7", dot.Value.Calc(100))
	}
}

// TestParseDescriptor_GoldPassive 解析 onTick + cooldown + selfTower + gold 管线。
func TestParseDescriptor_GoldPassive(t *testing.T) {
	data := []byte(`{
		"id": "goldPassive",
		"label": "Gold Generator",
		"cost": 5,
		"pipelines": [
			{
				"trigger": "onTick",
				"conditions": [
					{"type": "cooldown", "seconds": 5.0}
				],
				"selector": {"type": "selfTower"},
				"effects": [
					{"type": "gold", "amount": {"scaler": "linear", "base": 1, "potential": 0.5}}
				]
			}
		]
	}`)

	desc, err := descriptor.ParseDescriptor(data)
	if err != nil {
		t.Fatalf("ParseDescriptor() error: %v", err)
	}

	p := desc.Pipelines[0]
	if p.Trigger != descriptor.TriggerOnTick {
		t.Errorf("Trigger = %v, want onTick", p.Trigger)
	}

	// CooldownCondition
	if len(p.Conditions) != 1 {
		t.Fatalf("Conditions count = %d, want 1", len(p.Conditions))
	}
	cd, ok := p.Conditions[0].(*descriptor.CooldownCondition)
	if !ok {
		t.Fatalf("Conditions[0] type = %T, want *CooldownCondition", p.Conditions[0])
	}
	if cd.Seconds != 5.0 {
		t.Errorf("cd.Seconds = %v, want 5.0", cd.Seconds)
	}

	// SelfTowerSelector
	if _, ok := p.Selector.(descriptor.SelfTowerSelector); !ok {
		t.Errorf("Selector type = %T, want SelfTowerSelector", p.Selector)
	}

	// GoldEffect
	gold, ok := p.Effects[0].(descriptor.GoldEffect)
	if !ok {
		t.Fatalf("Effects[0] type = %T, want GoldEffect", p.Effects[0])
	}
	// strength=200 → amount = 1 + 0.5*(200/100) = 2
	if gold.Amount.Calc(200) != 2 {
		t.Errorf("gold.Amount.Calc(200) = %v, want 2", gold.Amount.Calc(200))
	}
}

// TestParseDescriptor_AttackStyleAndSpriteKey 解析含 attackStyle/spriteKey 的描述符。
func TestParseDescriptor_AttackStyleAndSpriteKey(t *testing.T) {
	data := []byte(`{
		"id": "scatterShot",
		"label": "Scatter Shot",
		"cost": 2,
		"attackStyle": "scatter",
		"spriteKey": "shotgun",
		"pipelines": [
			{
				"trigger": "onHit",
				"selector": {"type": "currentTarget"},
				"effects": [
					{"type": "damage", "mode": "flat", "value": {"scaler": "fixed", "value": 10}}
				]
			}
		]
	}`)

	desc, err := descriptor.ParseDescriptor(data)
	if err != nil {
		t.Fatalf("ParseDescriptor() error: %v", err)
	}

	if desc.AttackStyle != "scatter" {
		t.Errorf("AttackStyle = %q, want %q", desc.AttackStyle, "scatter")
	}
	if desc.SpriteKey != "shotgun" {
		t.Errorf("SpriteKey = %q, want %q", desc.SpriteKey, "shotgun")
	}
}

// ── 所有条件类型解析测试 ────────────────────────────────────

func TestParseDescriptor_AllConditionTypes(t *testing.T) {
	data := []byte(`{
		"id": "condTest",
		"label": "Condition Test",
		"cost": 1,
		"pipelines": [
			{
				"trigger": "onHit",
				"conditions": [
					{"type": "hpBelow", "threshold": {"scaler": "fixed", "value": 0.3}},
					{"type": "hpAbove", "threshold": {"scaler": "fixed", "value": 0.1}},
					{"type": "distanceMin", "distance": 100},
					{"type": "noNearbyTower", "radius": 80}
				],
				"selector": {"type": "currentTarget"},
				"effects": [
					{"type": "silence"}
				]
			}
		]
	}`)

	desc, err := descriptor.ParseDescriptor(data)
	if err != nil {
		t.Fatalf("ParseDescriptor() error: %v", err)
	}

	conds := desc.Pipelines[0].Conditions
	if len(conds) != 4 {
		t.Fatalf("Conditions count = %d, want 4", len(conds))
	}

	// hpBelow
	hpb, ok := conds[0].(descriptor.HpBelowCondition)
	if !ok {
		t.Errorf("conds[0] type = %T, want HpBelowCondition", conds[0])
	} else if hpb.Threshold.Calc(100) != 0.3 {
		t.Errorf("hpBelow.Threshold.Calc(100) = %v, want 0.3", hpb.Threshold.Calc(100))
	}

	// hpAbove
	hpa, ok := conds[1].(descriptor.HpAboveCondition)
	if !ok {
		t.Errorf("conds[1] type = %T, want HpAboveCondition", conds[1])
	} else if hpa.Threshold.Calc(100) != 0.1 {
		t.Errorf("hpAbove.Threshold.Calc(100) = %v, want 0.1", hpa.Threshold.Calc(100))
	}

	// distanceMin
	dm, ok := conds[2].(descriptor.DistanceMinCondition)
	if !ok {
		t.Errorf("conds[2] type = %T, want DistanceMinCondition", conds[2])
	} else if dm.Distance != 100 {
		t.Errorf("distanceMin.Distance = %v, want 100", dm.Distance)
	}

	// noNearbyTower
	nnt, ok := conds[3].(descriptor.NoNearbyTowerCondition)
	if !ok {
		t.Errorf("conds[3] type = %T, want NoNearbyTowerCondition", conds[3])
	} else if nnt.Radius != 80 {
		t.Errorf("noNearbyTower.Radius = %v, want 80", nnt.Radius)
	}
}

// ── 所有选择器类型解析测试 ──────────────────────────────────

func TestParseDescriptor_AllSelectorTypes(t *testing.T) {
	cases := []struct {
		name     string
		json     string
		checkFn  func(t *testing.T, sel descriptor.Selector)
	}{
		{
			name: "chain",
			json: `{"type": "chain", "maxBounce": {"scaler": "linear", "base": 2, "potential": 1}, "range": 120, "decayRatio": 0.7}`,
			checkFn: func(t *testing.T, sel descriptor.Selector) {
				cs, ok := sel.(descriptor.ChainSelector)
				if !ok {
					t.Fatalf("type = %T, want ChainSelector", sel)
				}
				if cs.MaxBounce.Calc(100) != 3 {
					t.Errorf("MaxBounce.Calc(100) = %v, want 3", cs.MaxBounce.Calc(100))
				}
				if cs.ChainRange != 120 {
					t.Errorf("ChainRange = %v, want 120", cs.ChainRange)
				}
				if cs.DecayRatio != 0.7 {
					t.Errorf("DecayRatio = %v, want 0.7", cs.DecayRatio)
				}
			},
		},
		{
			name: "allInRange",
			json: `{"type": "allInRange"}`,
			checkFn: func(t *testing.T, sel descriptor.Selector) {
				if _, ok := sel.(descriptor.AllInRangeSelector); !ok {
					t.Fatalf("type = %T, want AllInRangeSelector", sel)
				}
			},
		},
		{
			name: "nearbyAllies",
			json: `{"type": "nearbyAllies", "radius": 150}`,
			checkFn: func(t *testing.T, sel descriptor.Selector) {
				na, ok := sel.(descriptor.NearbyAlliesSelector)
				if !ok {
					t.Fatalf("type = %T, want NearbyAlliesSelector", sel)
				}
				if na.Radius != 150 {
					t.Errorf("Radius = %v, want 150", na.Radius)
				}
			},
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			json := `{
				"id": "test", "label": "Test", "cost": 1,
				"pipelines": [{
					"trigger": "onHit",
					"selector": ` + tc.json + `,
					"effects": [{"type": "silence"}]
				}]
			}`
			desc, err := descriptor.ParseDescriptor([]byte(json))
			if err != nil {
				t.Fatalf("ParseDescriptor() error: %v", err)
			}
			tc.checkFn(t, desc.Pipelines[0].Selector)
		})
	}
}

// ── 所有效果类型解析测试 ────────────────────────────────────

func TestParseDescriptor_AllEffectTypes(t *testing.T) {
	cases := []struct {
		name    string
		json    string
		checkFn func(t *testing.T, eff descriptor.Effect)
	}{
		{
			name: "damage_hpPercent",
			json: `{"type": "damage", "mode": "hpPercent", "value": {"scaler": "fixed", "value": 0.1}}`,
			checkFn: func(t *testing.T, eff descriptor.Effect) {
				d, ok := eff.(descriptor.DamageEffect)
				if !ok {
					t.Fatalf("type = %T, want DamageEffect", eff)
				}
				if d.Mode != descriptor.DmgHpPercent {
					t.Errorf("Mode = %v, want DmgHpPercent", d.Mode)
				}
			},
		},
		{
			name: "root",
			json: `{"type": "root", "duration": {"scaler": "fixed", "value": 1.5}}`,
			checkFn: func(t *testing.T, eff descriptor.Effect) {
				r, ok := eff.(descriptor.RootEffect)
				if !ok {
					t.Fatalf("type = %T, want RootEffect", eff)
				}
				if r.Duration.Calc(100) != 1.5 {
					t.Errorf("Duration.Calc(100) = %v, want 1.5", r.Duration.Calc(100))
				}
			},
		},
		{
			name: "weaken",
			json: `{"type": "weaken", "amplify": {"scaler": "fixed", "value": 0.2}, "duration": {"scaler": "fixed", "value": 3}}`,
			checkFn: func(t *testing.T, eff descriptor.Effect) {
				w, ok := eff.(descriptor.WeakenEffect)
				if !ok {
					t.Fatalf("type = %T, want WeakenEffect", eff)
				}
				if w.Amplify.Calc(100) != 0.2 {
					t.Errorf("Amplify.Calc(100) = %v, want 0.2", w.Amplify.Calc(100))
				}
				if w.Duration.Calc(100) != 3 {
					t.Errorf("Duration.Calc(100) = %v, want 3", w.Duration.Calc(100))
				}
			},
		},
		{
			name: "buff",
			json: `{"type": "buff", "stat": "damage", "bonus": {"scaler": "fixed", "value": 10}}`,
			checkFn: func(t *testing.T, eff descriptor.Effect) {
				b, ok := eff.(descriptor.BuffEffect)
				if !ok {
					t.Fatalf("type = %T, want BuffEffect", eff)
				}
				if b.Stat != "damage" {
					t.Errorf("Stat = %q, want %q", b.Stat, "damage")
				}
				if b.Bonus.Calc(100) != 10 {
					t.Errorf("Bonus.Calc(100) = %v, want 10", b.Bonus.Calc(100))
				}
			},
		},
		{
			name: "selfBuff",
			json: `{"type": "selfBuff", "stat": "range", "bonus": {"scaler": "linear", "base": 5, "potential": 2}}`,
			checkFn: func(t *testing.T, eff descriptor.Effect) {
				sb, ok := eff.(descriptor.SelfBuffEffect)
				if !ok {
					t.Fatalf("type = %T, want SelfBuffEffect", eff)
				}
				if sb.Stat != "range" {
					t.Errorf("Stat = %q, want %q", sb.Stat, "range")
				}
				// strength=100 → 5 + 2*(100/100) = 7
				if sb.Bonus.Calc(100) != 7 {
					t.Errorf("Bonus.Calc(100) = %v, want 7", sb.Bonus.Calc(100))
				}
			},
		},
		{
			name: "modifyStat",
			json: `{"type": "modifyStat", "stat": "atkSpeed", "multiplier": 1.5}`,
			checkFn: func(t *testing.T, eff descriptor.Effect) {
				ms, ok := eff.(descriptor.ModifyStatEffect)
				if !ok {
					t.Fatalf("type = %T, want ModifyStatEffect", eff)
				}
				if ms.Stat != "atkSpeed" {
					t.Errorf("Stat = %q, want %q", ms.Stat, "atkSpeed")
				}
				if ms.Multiplier != 1.5 {
					t.Errorf("Multiplier = %v, want 1.5", ms.Multiplier)
				}
			},
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			json := `{
				"id": "test", "label": "Test", "cost": 1,
				"pipelines": [{
					"trigger": "onHit",
					"selector": {"type": "currentTarget"},
					"effects": [` + tc.json + `]
				}]
			}`
			desc, err := descriptor.ParseDescriptor([]byte(json))
			if err != nil {
				t.Fatalf("ParseDescriptor() error: %v", err)
			}
			tc.checkFn(t, desc.Pipelines[0].Effects[0])
		})
	}
}

// ── 错误输入测试 ────────────────────────────────────────────

func TestParseDescriptor_Errors(t *testing.T) {
	cases := []struct {
		name    string
		json    string
		wantErr string
	}{
		{
			name:    "invalid JSON",
			json:    `{not json`,
			wantErr: "unmarshal",
		},
		{
			name: "unknown trigger",
			json: `{
				"id": "test", "label": "Test", "cost": 1,
				"pipelines": [{
					"trigger": "onDance",
					"selector": {"type": "currentTarget"},
					"effects": [{"type": "silence"}]
				}]
			}`,
			wantErr: "unknown trigger",
		},
		{
			name: "unknown condition type",
			json: `{
				"id": "test", "label": "Test", "cost": 1,
				"pipelines": [{
					"trigger": "onHit",
					"conditions": [{"type": "weather"}],
					"selector": {"type": "currentTarget"},
					"effects": [{"type": "silence"}]
				}]
			}`,
			wantErr: "unknown condition type",
		},
		{
			name: "unknown selector type",
			json: `{
				"id": "test", "label": "Test", "cost": 1,
				"pipelines": [{
					"trigger": "onHit",
					"selector": {"type": "randomEnemy"},
					"effects": [{"type": "silence"}]
				}]
			}`,
			wantErr: "unknown selector type",
		},
		{
			name: "unknown effect type",
			json: `{
				"id": "test", "label": "Test", "cost": 1,
				"pipelines": [{
					"trigger": "onHit",
					"selector": {"type": "currentTarget"},
					"effects": [{"type": "teleport"}]
				}]
			}`,
			wantErr: "unknown effect type",
		},
		{
			name: "unknown damage mode",
			json: `{
				"id": "test", "label": "Test", "cost": 1,
				"pipelines": [{
					"trigger": "onHit",
					"selector": {"type": "currentTarget"},
					"effects": [{"type": "damage", "mode": "magic", "value": {"scaler": "fixed", "value": 1}}]
				}]
			}`,
			wantErr: "unknown damage mode",
		},
		{
			name: "empty pipelines",
			json: `{
				"id": "test", "label": "Test", "cost": 1,
				"pipelines": []
			}`,
			wantErr: "at least one pipeline",
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			_, err := descriptor.ParseDescriptor([]byte(tc.json))
			if err == nil {
				t.Fatal("expected error, got nil")
			}
			if !strings.Contains(strings.ToLower(err.Error()), strings.ToLower(tc.wantErr)) {
				t.Errorf("error = %q, want it to contain %q", err.Error(), tc.wantErr)
			}
		})
	}
}

// TestParseDescriptor_IconField 解析含 icon 字段的描述符。
func TestParseDescriptor_IconField(t *testing.T) {
	data := []byte(`{
		"id": "test",
		"label": "Test",
		"icon": "stun_icon",
		"cost": 1,
		"pipelines": [
			{
				"trigger": "onHit",
				"selector": {"type": "currentTarget"},
				"effects": [{"type": "silence"}]
			}
		]
	}`)

	desc, err := descriptor.ParseDescriptor(data)
	if err != nil {
		t.Fatalf("ParseDescriptor() error: %v", err)
	}
	if desc.Icon != "stun_icon" {
		t.Errorf("Icon = %q, want %q", desc.Icon, "stun_icon")
	}
}
