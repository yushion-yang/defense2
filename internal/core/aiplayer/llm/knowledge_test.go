// knowledge_test.go -- 游戏知识库和增强 prompt 测试。
//
// 覆盖：知识文本非空、富数据 prompt 构建、多操作 JSON 解析、
//       战略 prompt 包含所有 section。
package llm

import (
	"encoding/json"
	"strings"
	"testing"
)

// ── GameKnowledge 内容测试 ──

func TestGameKnowledge_NonEmpty(t *testing.T) {
	if GameKnowledge == "" {
		t.Fatal("GameKnowledge should not be empty")
	}
	// 检查关键机制段落存在
	sections := []string{
		"塔系统", "能力类别", "敌人类型", "经济节奏", "位置策略", "道具使用", "协作原则",
	}
	for _, s := range sections {
		if !strings.Contains(GameKnowledge, s) {
			t.Errorf("GameKnowledge missing section: %s", s)
		}
	}
}

func TestCondensedKnowledge_NonEmpty(t *testing.T) {
	if CondensedKnowledge == "" {
		t.Fatal("CondensedKnowledge should not be empty")
	}
	// 压缩版也应包含核心关键字
	keywords := []string{
		"塔", "CC", "DPS", "敌人", "经济", "协作", "JSON",
	}
	for _, kw := range keywords {
		if !strings.Contains(CondensedKnowledge, kw) {
			t.Errorf("CondensedKnowledge missing keyword: %s", kw)
		}
	}
}

func TestCondensedKnowledge_TokenBudget(t *testing.T) {
	// 粗略估计：1 token ~= 1.5 个中文字符 或 ~4 个英文字符
	// 800 tokens ≈ 1200 中文字符 ≈ 3200 英文字符
	// 用 rune 数量作为上限参考（中文占主体时 ~1200 rune）
	runes := []rune(CondensedKnowledge)
	if len(runes) > 1500 {
		t.Errorf("CondensedKnowledge too long: %d runes (target: <1500 for ~800 tokens)", len(runes))
	}
}

// ── BuildPrompt 富数据测试 ──

func TestBuildPrompt_WithRichData(t *testing.T) {
	s := Situation{
		Wave: 8, MaxWaves: 30,
		AIGold: 150, HumanGold: 200,
		Lives: 15, MaxLives: 20,
		AITowerCount: 3, HumanTowerCount: 2,
		LastAction:  "built tower",
		ThreatLevel: "high",
		Mood:        "tense",
		AITowers: []TowerDesc{
			{Position: "entrance", Abilities: []string{"slow", "scatter"}, Damage: 35, Range: 100, Kills: 12, Style: "scatter"},
			{Position: "middle", Abilities: []string{"barrage", "crit"}, Damage: 55, Range: 80, Kills: 20, Style: "barrage"},
		},
		EnemyGroups: []EnemyGroup{
			{Type: "tank", Count: 3, AvgHP: 500},
			{Type: "runner", Count: 5, AvgHP: 100},
			{Type: "boss", Count: 1, AvgHP: 2000},
		},
		PendingAbilities: 1,
		NextWaveBoss:     true,
		CoopDesc:         "队友有控制，全队缺输出",
		AdvicePriority:   "build_dps",
	}
	mem := NewMemory()
	mem.Add("built tower", "又一座搞起来!")

	prompt := BuildPrompt(s, mem)

	// 验证详细塔信息出现在 prompt 中
	if !strings.Contains(prompt, "塔1(entrance,slow+scatter") {
		t.Error("prompt should contain tower 1 description with abilities")
	}
	if !strings.Contains(prompt, "塔2(middle,barrage+crit") {
		t.Error("prompt should contain tower 2 description with abilities")
	}

	// 验证敌人组成
	if !strings.Contains(prompt, "3tank") {
		t.Error("prompt should contain tank enemy group")
	}
	if !strings.Contains(prompt, "1boss") {
		t.Error("prompt should contain boss enemy group")
	}

	// 验证待处理事项
	if !strings.Contains(prompt, "1座塔待选能力") {
		t.Error("prompt should mention pending ability choices")
	}
	if !strings.Contains(prompt, "Boss") {
		t.Error("prompt should mention next wave boss")
	}

	// 验证记忆
	if !strings.Contains(prompt, "又一座搞起来") {
		t.Error("prompt should include memory")
	}
}

func TestBuildPrompt_MinimalData(t *testing.T) {
	// 最小数据：无塔无敌人
	s := Situation{
		Wave: 1, MaxWaves: 30,
		AIGold: 100, HumanGold: 100,
		Lives: 20, MaxLives: 20,
		ThreatLevel: "low",
	}
	prompt := BuildPrompt(s, nil)

	if !strings.Contains(prompt, "第 1/30 波") {
		t.Error("prompt should contain wave info")
	}
	// 不应崩溃
	if prompt == "" {
		t.Fatal("prompt should not be empty even with minimal data")
	}
}

// ── BuildStrategicPrompt 测试 ──

func TestBuildStrategicPrompt_IncludesAllSections(t *testing.T) {
	s := Situation{
		Wave: 10, MaxWaves: 30,
		AIGold: 200, Lives: 15, MaxLives: 20,
		ThreatLevel: "high",
		AITowers: []TowerDesc{
			{Position: "entrance", Abilities: []string{"slow"}, Damage: 30, Range: 100, Kills: 8},
		},
		EnemyGroups: []EnemyGroup{
			{Type: "boss", Count: 1, AvgHP: 3000},
		},
		AvailableItems:   []string{"BaseDamage", "Speed"},
		PendingAbilities: 2,
		NextWaveBoss:     true,
		CanBuild:         true,
		CanUpgrade:       true,
	}

	prompt := BuildStrategicPrompt(s, "build_dps", "队友有控制")

	// 验证各 section 存在
	checks := map[string]string{
		"wave info":        "第10/30波",
		"gold":             "金币:200",
		"threat":           "威胁:high",
		"tower desc":       "塔1(",
		"enemy desc":       "1boss",
		"available items":  "道具(BaseDamage,Speed)",
		"pending":          "2塔待选能力",
		"coop desc":        "队友有控制",
		"advice":           "build_dps",
		"boss warning":     "下一波Boss",
		"json format":      "actions",
		"multi-action":     "最多3个action",
		"can build":        "造塔(50金)",
		"can upgrade":      "升级(10金)",
	}
	for name, substr := range checks {
		if !strings.Contains(prompt, substr) {
			t.Errorf("strategic prompt missing %s (expected substring: %q)", name, substr)
		}
	}
}

func TestBuildStrategicPrompt_NoItems(t *testing.T) {
	s := Situation{
		Wave: 5, MaxWaves: 30,
		AIGold: 50, Lives: 20, MaxLives: 20,
		ThreatLevel: "low",
	}

	prompt := BuildStrategicPrompt(s, "", "")
	// 无道具时不应出现道具段
	if strings.Contains(prompt, "道具(") {
		t.Error("prompt should not mention items when none available")
	}
}

// ── ParseLLMDecision 多操作解析测试 ──

func TestParseLLMDecision_MultiAction(t *testing.T) {
	input := `{"actions":[
		{"type":"sell","target":"weakest","reason":"先卖差塔"},
		{"type":"build","position":"entrance","priority":"cc","reason":"补个控制"},
		{"type":"upgrade","target":"strongest","reason":"强化主力"}
	],"say":"大换血！"}`

	d := ParseLLMDecision(input)
	if d == nil {
		t.Fatal("ParseLLMDecision should parse multi-action JSON")
	}
	if len(d.Actions) != 3 {
		t.Fatalf("expected 3 actions, got %d", len(d.Actions))
	}

	// 验证第一个操作
	if d.Actions[0].Type != "sell" {
		t.Errorf("action 0 type: expected sell, got %s", d.Actions[0].Type)
	}
	if d.Actions[0].Target != "weakest" {
		t.Errorf("action 0 target: expected weakest, got %s", d.Actions[0].Target)
	}

	// 验证第二个操作
	if d.Actions[1].Type != "build" {
		t.Errorf("action 1 type: expected build, got %s", d.Actions[1].Type)
	}
	if d.Actions[1].Position != "entrance" {
		t.Errorf("action 1 position: expected entrance, got %s", d.Actions[1].Position)
	}
	if d.Actions[1].Priority != "cc" {
		t.Errorf("action 1 priority: expected cc, got %s", d.Actions[1].Priority)
	}

	// 验证第三个操作
	if d.Actions[2].Type != "upgrade" {
		t.Errorf("action 2 type: expected upgrade, got %s", d.Actions[2].Type)
	}

	// 验证弹幕
	if d.Say != "大换血！" {
		t.Errorf("say: expected '大换血！', got %s", d.Say)
	}
}

func TestParseLLMDecision_SingleActionCompat(t *testing.T) {
	// 旧版单操作格式应仍然可用
	input := `{"action":"build","priority":"dps","reason":"加火力"}`

	d := ParseLLMDecision(input)
	if d == nil {
		t.Fatal("ParseLLMDecision should parse single-action JSON")
	}

	first := d.FirstAction()
	if first.Type != "build" {
		t.Errorf("expected type=build, got %s", first.Type)
	}
	if first.Priority != "dps" {
		t.Errorf("expected priority=dps, got %s", first.Priority)
	}
	if first.Reason != "加火力" {
		t.Errorf("expected reason='加火力', got %s", first.Reason)
	}
}

func TestParseLLMDecision_InvalidJSON(t *testing.T) {
	cases := []string{
		"not json at all",
		`{"action":"invalid_type"}`,
		`{"actions":[{"type":"fly"}]}`,
		`{}`,
		`{"actions":[]}`,
	}
	for _, c := range cases {
		d := ParseLLMDecision(c)
		if d != nil {
			t.Errorf("expected nil for input %q, got %+v", c, d)
		}
	}
}

func TestParseLLMDecision_TruncatesLongChain(t *testing.T) {
	// 超过 3 个操作应被截断
	input := `{"actions":[
		{"type":"sell","reason":"1"},
		{"type":"build","reason":"2"},
		{"type":"upgrade","reason":"3"},
		{"type":"build","reason":"4"},
		{"type":"upgrade","reason":"5"}
	]}`

	d := ParseLLMDecision(input)
	if d == nil {
		t.Fatal("should parse")
	}
	if len(d.Actions) != 3 {
		t.Fatalf("expected 3 actions (truncated), got %d", len(d.Actions))
	}
}

func TestParseLLMDecision_FiltersInvalid(t *testing.T) {
	// 混合有效和无效的操作类型
	input := `{"actions":[
		{"type":"build","reason":"ok"},
		{"type":"fly","reason":"bad"},
		{"type":"upgrade","reason":"ok2"}
	]}`

	d := ParseLLMDecision(input)
	if d == nil {
		t.Fatal("should parse with valid actions present")
	}
	if len(d.Actions) != 2 {
		t.Fatalf("expected 2 valid actions, got %d", len(d.Actions))
	}
	if d.Actions[0].Type != "build" {
		t.Errorf("action 0: expected build, got %s", d.Actions[0].Type)
	}
	if d.Actions[1].Type != "upgrade" {
		t.Errorf("action 1: expected upgrade, got %s", d.Actions[1].Type)
	}
}

func TestParseLLMDecision_NewActionTypes(t *testing.T) {
	// 测试新增的 action type
	types := []string{"sell", "ability", "item"}
	for _, typ := range types {
		input := `{"actions":[{"type":"` + typ + `","reason":"test"}]}`
		d := ParseLLMDecision(input)
		if d == nil {
			t.Fatalf("should parse action type %q", typ)
		}
		if d.Actions[0].Type != typ {
			t.Errorf("expected type=%s, got %s", typ, d.Actions[0].Type)
		}
	}
}

func TestParseLLMDecision_ItemAction(t *testing.T) {
	input := `{"actions":[{"type":"item","itemType":"BaseDamage","target":"strongest","reason":"强化主力"}],"say":"吃道具！"}`

	d := ParseLLMDecision(input)
	if d == nil {
		t.Fatal("should parse item action")
	}
	if d.Actions[0].ItemType != "BaseDamage" {
		t.Errorf("expected itemType=BaseDamage, got %s", d.Actions[0].ItemType)
	}
	if d.Actions[0].Target != "strongest" {
		t.Errorf("expected target=strongest, got %s", d.Actions[0].Target)
	}
}

func TestParseLLMDecision_AbilityAction(t *testing.T) {
	input := `{"actions":[{"type":"ability","preference":"cc","reason":"补控制"}]}`

	d := ParseLLMDecision(input)
	if d == nil {
		t.Fatal("should parse ability action")
	}
	if d.Actions[0].Preference != "cc" {
		t.Errorf("expected preference=cc, got %s", d.Actions[0].Preference)
	}
}

// ── FirstAction 兼容测试 ──

func TestFirstAction_NewFormat(t *testing.T) {
	d := &LLMDecision{
		Actions: []LLMAction{
			{Type: "build", Priority: "cc", Reason: "补控制"},
			{Type: "upgrade", Target: "strongest"},
		},
	}
	first := d.FirstAction()
	if first.Type != "build" || first.Priority != "cc" {
		t.Errorf("FirstAction wrong: %+v", first)
	}
}

func TestFirstAction_OldFormat(t *testing.T) {
	d := &LLMDecision{
		Action:   "upgrade",
		Priority: "",
		Target:   "weakest",
		Reason:   "强化弱塔",
	}
	first := d.FirstAction()
	if first.Type != "upgrade" || first.Target != "weakest" {
		t.Errorf("FirstAction wrong for old format: %+v", first)
	}
}

// ── TowerDesc/EnemyGroup 结构验证 ──

func TestTowerDescFields(t *testing.T) {
	td := TowerDesc{
		Position: "entrance", Abilities: []string{"slow", "scatter"},
		Damage: 35.5, Range: 100, Kills: 12, Style: "scatter", Strength: 150,
	}
	if td.Position != "entrance" {
		t.Error("position mismatch")
	}
	if len(td.Abilities) != 2 {
		t.Error("abilities count mismatch")
	}
	if td.Strength != 150 {
		t.Error("strength mismatch")
	}
}

func TestEnemyGroupFields(t *testing.T) {
	eg := EnemyGroup{Type: "boss", Count: 1, AvgHP: 3000}
	if eg.Type != "boss" || eg.Count != 1 || eg.AvgHP != 3000 {
		t.Errorf("EnemyGroup mismatch: %+v", eg)
	}
}

// ── LLMAction JSON 序列化测试 ──

func TestLLMAction_JSONRoundtrip(t *testing.T) {
	original := LLMAction{
		Type:     "build",
		Position: "entrance",
		Priority: "cc",
		Reason:   "补控制",
	}
	data, err := json.Marshal(original)
	if err != nil {
		t.Fatalf("marshal error: %v", err)
	}

	var parsed LLMAction
	if err := json.Unmarshal(data, &parsed); err != nil {
		t.Fatalf("unmarshal error: %v", err)
	}
	if parsed.Type != original.Type || parsed.Position != original.Position ||
		parsed.Priority != original.Priority || parsed.Reason != original.Reason {
		t.Errorf("roundtrip mismatch: %+v vs %+v", original, parsed)
	}
}
