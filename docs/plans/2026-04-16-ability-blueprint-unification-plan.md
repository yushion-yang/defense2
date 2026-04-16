# Ability/Blueprint Unification Implementation Plan

> **For Claude:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to implement this plan task-by-task.

**Goal:** Unify ability and tower definition systems — descriptors become the single source of truth for abilities, blueprints become the single source of truth for towers.

**Architecture:** Replace `abilities.json` with metadata in `ability-descriptors.json`, replace `classic-presets.json` with `prebuilt-blueprints.json` (reusing TowerBlueprint format). Auto-derive `AbilityDef` from descriptors at load time. Expand workshop to 4 tabs showing pre-built content read-only with "copy to custom" support.

**Tech Stack:** Go 1.24+, Ebitengine, JSON config, TDD

**Design doc:** `docs/plans/2026-04-16-ability-blueprint-unification-design.md`

---

## Phase 1: Descriptor as Single Ability Source

### Task 1: Extend ability-descriptors.json with icon and display fields

**Files:**
- Modify: `config/towers/ability-descriptors.json`
- Reference: `config/towers/abilities.json` (source of icon/display values)

**Step 1: Add `icon`, `display`, `prebuilt` fields to every descriptor**

For each of the 32 descriptors, add three fields from abilities.json. The icon mapping:
- Most abilities: `icon` = same as `id`
- Exceptions: `goldPassive` → `icon: "gold"`, `bleedDot` → `icon: "bleed"`

The `display` field is copied verbatim from abilities.json. Add `"prebuilt": true` to all 32.

Example diff for `slowPower`:
```json
{
  "id": "slowPower",
  "label": "凝滞",
  "icon": "slowPower",
  "display": "攻击命中对敌人减速{s%}, 持续{p}秒",
  "prebuilt": true,
  "cost": 8,
  "tags": ["cc"],
  "pipelines": [...]
}
```

**Step 2: Validate JSON is valid**

Run: `python3 -c "import json; json.load(open('config/towers/ability-descriptors.json'))"`
Expected: no error

**Step 3: Commit**

```bash
git add config/towers/ability-descriptors.json
git commit -m "feat: add icon/display/prebuilt fields to ability descriptors"
```

---

### Task 2: Extend AbilityDescriptor struct with new fields

**Files:**
- Modify: `internal/core/tower/descriptor/descriptor.go:20-30` (AbilityDescriptor struct)
- Modify: `internal/core/tower/descriptor/descriptor.go:43-53` (rawDescriptor struct)

**Step 1: Write contract test for new fields**

**Test file:** `tests/contracts/descriptor_metadata_test.go`

```go
package contracts

import (
	"testing"

	"defense2/internal/config"
	"defense2/internal/core/tower/abilities"
	"defense2/internal/core/tower/descriptor"
)

// TestDescriptorMetadataComplete 验证所有预制描述符包含完整的展示元数据。
func TestDescriptorMetadataComplete(t *testing.T) {
	config.InitDataFS()
	abilities.InitConfigAbilities()
	descriptor.InitDescriptorAbilities(config.DataFS())

	table := descriptor.GlobalDescriptorTable()
	if len(table) < 32 {
		t.Fatalf("expected >= 32 descriptors, got %d", len(table))
	}

	for _, desc := range table {
		if desc.Icon == "" {
			t.Errorf("descriptor %q missing icon", desc.ID)
		}
		if desc.Display == "" {
			t.Errorf("descriptor %q missing display", desc.ID)
		}
		if !desc.Prebuilt {
			t.Errorf("descriptor %q should be prebuilt", desc.ID)
		}
		if len(desc.Tags) == 0 {
			t.Errorf("descriptor %q missing tags", desc.ID)
		}
	}
}
```

**Step 2: Run test to verify it fails**

Run: `go test ./tests/contracts/ -run TestDescriptorMetadataComplete -v`
Expected: FAIL (fields don't exist in struct yet)

**Step 3: Add fields to AbilityDescriptor and rawDescriptor**

In `internal/core/tower/descriptor/descriptor.go`, add to AbilityDescriptor (line ~23):
```go
type AbilityDescriptor struct {
	ID           string            `json:"id"`
	Label        string            `json:"label"`
	Icon         string            `json:"icon,omitempty"`
	Display      string            `json:"display,omitempty"`      // ← 新增
	Prebuilt     bool              `json:"prebuilt,omitempty"`     // ← 新增
	Cost         int               `json:"cost"`
	Tags         []string          `json:"tags,omitempty"`
	AttackStyle  string            `json:"attackStyle,omitempty"`
	SpriteKey    string            `json:"spriteKey,omitempty"`
	AttackParams json.RawMessage   `json:"attackParams,omitempty"`
	Pipelines    []Pipeline        `json:"pipelines"`
}
```

Add same fields to `rawDescriptor` (line ~43):
```go
type rawDescriptor struct {
	ID           string            `json:"id"`
	Label        string            `json:"label"`
	Icon         string            `json:"icon,omitempty"`
	Display      string            `json:"display,omitempty"`      // ← 新增
	Prebuilt     bool              `json:"prebuilt,omitempty"`     // ← 新增
	Cost         int               `json:"cost"`
	Tags         []string          `json:"tags,omitempty"`
	AttackStyle  string            `json:"attackStyle,omitempty"`
	SpriteKey    string            `json:"spriteKey,omitempty"`
	AttackParams json.RawMessage   `json:"attackParams,omitempty"`
	Pipelines    []json.RawMessage `json:"pipelines"`
}
```

Also update the field copy in the parsing function (where rawDescriptor fields are copied to AbilityDescriptor) to include `Display` and `Prebuilt`.

**Step 4: Run test to verify it passes**

Run: `go test ./tests/contracts/ -run TestDescriptorMetadataComplete -v`
Expected: PASS

**Step 5: Run full test suite**

Run: `make test`
Expected: all pass

**Step 6: Commit**

```bash
git add internal/core/tower/descriptor/descriptor.go tests/contracts/descriptor_metadata_test.go
git commit -m "feat: add Display/Prebuilt fields to AbilityDescriptor"
```

---

### Task 3: Implement DeriveAbilityTable — generate AbilityDef from descriptors

**Files:**
- Create: `internal/core/tower/descriptor/derive_ability_table.go`
- Test: `tests/contracts/derive_ability_table_test.go`

**Step 1: Write contract test — derived table matches old abilities.json**

```go
package contracts

import (
	"testing"

	"defense2/internal/config"
	"defense2/internal/core/tower/abilities"
	"defense2/internal/core/tower/descriptor"
)

// TestDeriveAbilityTableMatchesOld 验证从描述符派生的 AbilityTable 与旧 abilities.json 一致。
func TestDeriveAbilityTableMatchesOld(t *testing.T) {
	config.InitDataFS()
	abilities.InitConfigAbilities()

	oldTable := config.GlobalAbilityTable()
	if len(oldTable) == 0 {
		t.Fatal("old ability table is empty")
	}

	descriptor.InitDescriptorAbilities(config.DataFS())
	derived := descriptor.DeriveAbilityTable()

	// 每个旧能力必须在派生表中存在且元数据匹配
	for id, oldDef := range oldTable {
		newDef, ok := derived[id]
		if !ok {
			t.Errorf("ability %q missing from derived table", id)
			continue
		}
		if newDef.Category != oldDef.Category {
			t.Errorf("%s: category mismatch: got %q, want %q", id, newDef.Category, oldDef.Category)
		}
		if newDef.Icon != oldDef.Icon {
			t.Errorf("%s: icon mismatch: got %q, want %q", id, newDef.Icon, oldDef.Icon)
		}
		if newDef.Label != oldDef.Label {
			t.Errorf("%s: label mismatch: got %q, want %q", id, newDef.Label, oldDef.Label)
		}
	}
}
```

**Step 2: Run test to verify it fails**

Run: `go test ./tests/contracts/ -run TestDeriveAbilityTableMatchesOld -v`
Expected: FAIL (`DeriveAbilityTable` undefined)

**Step 3: Implement DeriveAbilityTable**

Create `internal/core/tower/descriptor/derive_ability_table.go`:

```go
// derive_ability_table.go — 从描述符自动派生 AbilityDef 元数据表。
//
// 将描述符中的 label/icon/tags/display 映射为 config.AbilityDef，
// 使 config.GlobalAbilityTable() 能同时包含预制和自定义能力。
//
// 关联：
//   - descriptor.go — AbilityDescriptor 数据结构
//   - config/ability_config.go — AbilityDef / AbilityTable 类型
package descriptor

import "defense2/internal/config"

// DeriveAbilityTable 从全局描述符表生成 AbilityTable。
// 每个描述符的 tags[0] 映射为 category，管线的主 scaler 提取为 base/potential。
func DeriveAbilityTable() config.AbilityTable {
	table := config.AbilityTable{}
	for _, desc := range GlobalDescriptorTable() {
		table[desc.ID] = deriveAbilityDef(desc)
	}
	return table
}

// DeriveAbilityDefFromDescriptor 从单个描述符生成 AbilityDef（供自定义能力注册用）。
func DeriveAbilityDefFromDescriptor(desc *AbilityDescriptor) *config.AbilityDef {
	return deriveAbilityDef(desc)
}

func deriveAbilityDef(desc *AbilityDescriptor) *config.AbilityDef {
	def := &config.AbilityDef{
		Type:    desc.ID,
		Label:   desc.Label,
		Icon:    desc.Icon,
		Display: desc.Display,
	}

	// category 从 tags[0] 取
	if len(desc.Tags) > 0 {
		def.Category = desc.Tags[0]
	}

	// 从第一条管线的第一个 effect 提取 scaler 作为 base/potential
	if len(desc.Pipelines) > 0 && len(desc.Pipelines[0].Effects) > 0 {
		extractScalerParams(desc.Pipelines[0].Effects[0], def)
	}

	// attack 类能力有 attackParams，从中提取主 scaler
	if desc.AttackParams != nil && len(desc.Pipelines) == 0 {
		extractAttackParams(desc, def)
	}

	return def
}
```

`extractScalerParams` 和 `extractAttackParams` 用类型断言从 effect/attackParams 中提取 FixedScaler 或 LinearScaler 的 base/potential 值。需要为每种 effect 类型定义提取规则。

注意：这些函数的具体实现需要遍历不同的 effect 类型来获取它们的主 scaler。可参考 `describe.go` 中 `GenerateDescription` 的逻辑来确定每种 effect 的主参数。

**Step 4: Run test to verify it passes**

Run: `go test ./tests/contracts/ -run TestDeriveAbilityTableMatchesOld -v`
Expected: PASS

**Step 5: Commit**

```bash
git add internal/core/tower/descriptor/derive_ability_table.go tests/contracts/derive_ability_table_test.go
git commit -m "feat: DeriveAbilityTable generates AbilityDef from descriptors"
```

---

### Task 4: Wire DeriveAbilityTable into loading pipeline

**Files:**
- Modify: `internal/core/tower/descriptor/init.go:45-60` (InitDescriptorAbilities)
- Modify: `internal/scene/loading.go:87-95` (phaseConfigs)

**Step 1: Write test — GlobalAbilityTable contains derived data after init**

Add to `tests/contracts/derive_ability_table_test.go`:

```go
// TestGlobalAbilityTableAfterInit 验证初始化完成后 GlobalAbilityTable 包含所有描述符能力。
func TestGlobalAbilityTableAfterInit(t *testing.T) {
	config.InitDataFS()
	abilities.InitConfigAbilities()
	descriptor.InitDescriptorAbilities(config.DataFS())

	table := config.GlobalAbilityTable()
	// 应至少包含 32 个预制能力
	if len(table) < 32 {
		t.Fatalf("expected >= 32 abilities in table, got %d", len(table))
	}
	// 随机抽查一个
	if def, ok := table["slowPower"]; !ok {
		t.Error("slowPower missing from GlobalAbilityTable")
	} else if def.Category != "cc" {
		t.Errorf("slowPower category: got %q, want cc", def.Category)
	}
}
```

**Step 2: Run test — should fail (derived table not yet wired)**

Run: `go test ./tests/contracts/ -run TestGlobalAbilityTableAfterInit -v`

**Step 3: Wire in InitDescriptorAbilities**

In `internal/core/tower/descriptor/init.go`, after registering to tower.Registry, add:
```go
// 派生 AbilityTable 并注入到 config.GlobalAbilityTable
derived := DeriveAbilityTable()
config.MergeAbilityTable(derived)
```

Need to add `config.MergeAbilityTable()` function in `internal/config/ability_config.go`:
```go
// MergeAbilityTable 将 entries 合并到全局能力表（覆盖同 key）。
func MergeAbilityTable(entries AbilityTable) {
	if globalAbilityTable == nil {
		globalAbilityTable = AbilityTable{}
	}
	for k, v := range entries {
		globalAbilityTable[k] = v
	}
}
```

**Step 4: Run test to verify it passes**

Run: `go test ./tests/contracts/ -run TestGlobalAbilityTableAfterInit -v`
Expected: PASS

**Step 5: Run full test suite**

Run: `make test`
Expected: all pass

**Step 6: Commit**

```bash
git add internal/core/tower/descriptor/init.go internal/config/ability_config.go tests/contracts/derive_ability_table_test.go
git commit -m "feat: wire DeriveAbilityTable into loading pipeline"
```

---

### Task 5: Register custom abilities into AbilityTable

**Files:**
- Modify: `internal/core/tower/descriptor/init.go:25-41` (RegisterCustomAbilities)

**Step 1: Write test — custom ability appears in AbilityTable after registration**

Add to `tests/contracts/derive_ability_table_test.go`:

```go
// TestCustomAbilityInAbilityTable 验证自定义能力注册后出现在 AbilityTable 中。
func TestCustomAbilityInAbilityTable(t *testing.T) {
	config.InitDataFS()
	abilities.InitConfigAbilities()
	descriptor.InitDescriptorAbilities(config.DataFS())

	// 创建内存存储和自定义能力
	store := descriptor.NewAbilityStore(persistence.NewMemoryStorage())
	ca := descriptor.CustomAbility{
		ID:   "ca_test_001",
		Name: "测试能力",
		Desc: descriptor.AbilityDescriptor{
			ID:    "ca_test_001",
			Label: "测试能力",
			Icon:  "crit",
			Tags:  []string{"damage"},
			Pipelines: []descriptor.Pipeline{},
		},
	}
	store.Save(ca)

	descriptor.RegisterCustomAbilities(store)

	table := config.GlobalAbilityTable()
	def, ok := table["ca_test_001"]
	if !ok {
		t.Fatal("custom ability ca_test_001 not in AbilityTable")
	}
	if def.Category != "damage" {
		t.Errorf("category: got %q, want damage", def.Category)
	}
}
```

**Step 2: Run test — should fail**

**Step 3: In RegisterCustomAbilities, also inject AbilityDef**

In `internal/core/tower/descriptor/init.go`, inside the loop in RegisterCustomAbilities:
```go
for _, ca := range store.List() {
	desc := ca.Desc
	desc.ID = ca.ID
	ability := NewDescriptorAbility(&desc)
	tower.Register(ability)
	// 同步注入 AbilityTable，使 AddAbility 能识别自定义能力
	config.MergeAbilityTable(config.AbilityTable{
		ca.ID: DeriveAbilityDefFromDescriptor(&desc),
	})
	registered++
}
```

**Step 4: Run test to verify it passes**

**Step 5: Run `make test`**

**Step 6: Commit**

```bash
git add internal/core/tower/descriptor/init.go tests/contracts/derive_ability_table_test.go
git commit -m "feat: register custom abilities into AbilityTable"
```

> **Phase 1 完成后**：自定义能力蓝图塔的 bug 已修复。abilities.json 仍存在但可以不再被引用。

---

## Phase 2: Blueprint Unification

### Task 6: Create prebuilt-blueprints.json

**Files:**
- Create: `config/towers/prebuilt-blueprints.json`
- Reference: `config/towers/classic-presets.json` (source data)

**Step 1: Convert classic-presets.json to blueprint format**

将 11 个经典塔转换为 `TowerBlueprint` 格式的 JSON。每个塔添加 `"prebuilt": true`。

关键字段映射：
- `key` → `id`
- `name` → `name`
- `category` → `category`（新增字段，蓝图格式原本没有）
- `abilities` → `abilities`
- `tiers` → `tiers`
- `specialty` → `specialty`
- `buildCost`：使用 defaults.buildCost (50) 或自行指定
- `strength`：使用 defaults.strength

**Step 2: Validate JSON**

Run: `python3 -c "import json; json.load(open('config/towers/prebuilt-blueprints.json'))"`

**Step 3: Commit**

```bash
git add config/towers/prebuilt-blueprints.json
git commit -m "feat: add prebuilt-blueprints.json (classic towers as blueprints)"
```

---

### Task 7: Add Prebuilt field to TowerBlueprint and loader

**Files:**
- Modify: `internal/core/tower/descriptor/blueprint.go:22-47` (TowerBlueprint struct)
- Create: `internal/core/tower/descriptor/prebuilt_blueprints.go`
- Modify: `internal/config/` (add loader for new JSON)
- Test: `tests/contracts/prebuilt_blueprint_test.go`

**Step 1: Write contract test — prebuilt blueprints generate same TowerDef as old classic path**

```go
package contracts

import (
	"testing"

	"defense2/internal/config"
	"defense2/internal/core/tower/abilities"
	"defense2/internal/core/tower/descriptor"
)

// TestPrebuiltBlueprintMatchesClassic 验证预制蓝图和经典预设生成的 TowerDef 属性一致。
func TestPrebuiltBlueprintMatchesClassic(t *testing.T) {
	config.InitDataFS()
	abilities.InitConfigAbilities()
	descriptor.InitDescriptorAbilities(config.DataFS())

	// 旧路径
	oldDefs := loadClassicTowerDefsForTest()
	// 新路径
	newDefs := descriptor.LoadPrebuiltBlueprintDefs()

	if len(newDefs) != len(oldDefs) {
		t.Fatalf("count mismatch: old=%d new=%d", len(oldDefs), len(newDefs))
	}

	for i, oldDef := range oldDefs {
		newDef := newDefs[i]
		if oldDef.Key != newDef.Key {
			t.Errorf("[%d] key mismatch: %s vs %s", i, oldDef.Key, newDef.Key)
			continue
		}
		assertFloatEqual(t, oldDef.Key+".CfgBaseDamage", oldDef.CfgBaseDamage, newDef.CfgBaseDamage)
		assertFloatEqual(t, oldDef.Key+".PotentialDamage", oldDef.PotentialDamage, newDef.PotentialDamage)
		assertFloatEqual(t, oldDef.Key+".CfgBaseSpeed", oldDef.CfgBaseSpeed, newDef.CfgBaseSpeed)
		assertFloatEqual(t, oldDef.Key+".PotentialSpeed", oldDef.PotentialSpeed, newDef.PotentialSpeed)
		assertFloatEqual(t, oldDef.Key+".CfgBaseRange", oldDef.CfgBaseRange, newDef.CfgBaseRange)
		assertFloatEqual(t, oldDef.Key+".PotentialRange", oldDef.PotentialRange, newDef.PotentialRange)
		if oldDef.Cost != newDef.Cost {
			t.Errorf("%s cost: %d vs %d", oldDef.Key, oldDef.Cost, newDef.Cost)
		}
	}
}
```

**Step 2: Run test — should fail**

**Step 3: Add `Prebuilt` field to TowerBlueprint**

In `internal/core/tower/descriptor/blueprint.go`, add to TowerBlueprint struct:
```go
// ── 元信息 ──
Prebuilt    bool   `json:"prebuilt,omitempty"`
Category    string `json:"category,omitempty"`
Description string `json:"description,omitempty"`
```

**Step 4: Implement LoadPrebuiltBlueprintDefs**

Create `internal/core/tower/descriptor/prebuilt_blueprints.go`:
- Load `config/towers/prebuilt-blueprints.json` via `config.DataFS()`
- Parse into `[]TowerBlueprint`
- Call `BlueprintToTowerDef` for each → return `[]tower.TowerDef`

**Step 5: Run test to verify it passes**

**Step 6: Commit**

```bash
git add internal/core/tower/descriptor/blueprint.go internal/core/tower/descriptor/prebuilt_blueprints.go tests/contracts/prebuilt_blueprint_test.go
git commit -m "feat: LoadPrebuiltBlueprintDefs with classic parity test"
```

---

### Task 8: Replace loadClassicTowerDefs with blueprint path

**Files:**
- Modify: `internal/scene/stage.go:3897-3918` (loadTowerDefsForMode)
- Modify: `internal/scene/stage.go:3940-3999` (loadClassicTowerDefs — delete)
- Modify: `internal/scene/loading.go:95` (remove LoadClassicPresets call)

**Step 1: In loadTowerDefsForMode, replace loadClassicTowerDefs() calls**

At line 3899:
```go
// 旧: return loadClassicTowerDefs()
return descriptor.LoadPrebuiltBlueprintDefs()
```

At line 3907:
```go
// 旧: defs = append(defs, loadClassicTowerDefs()...)
defs = append(defs, descriptor.LoadPrebuiltBlueprintDefs()...)
```

**Step 2: Delete `loadClassicTowerDefs()` function (lines 3940-3999)**

**Step 3: Remove `config.LoadClassicPresets()` from loading.go line 95**

**Step 4: Run `make test`**

Expected: all pass (contract test from Task 7 ensures parity)

**Step 5: Commit**

```bash
git add internal/scene/stage.go internal/scene/loading.go
git commit -m "refactor: replace loadClassicTowerDefs with blueprint path"
```

---

## Phase 3: Workshop 4 Tab

### Task 9: Expand TowerWorkshopScene to 4 tabs

**Files:**
- Modify: `internal/scene/tower_workshop.go:41-44` (tab constants)
- Modify: `internal/scene/tower_workshop.go:75-89` (struct fields)
- Modify: `internal/scene/tower_workshop.go` (Update/Draw methods)

**Step 1: Update tab constants**

```go
const (
	wsTabPrebuiltAbilities = 0 // 预制能力（只读）
	wsTabCustomAbilities   = 1 // 我的自定义能力
	wsTabPrebuiltTowers    = 2 // 预制塔（只读）
	wsTabCustomBlueprints  = 3 // 我的自定义蓝图
)
```

**Step 2: Add prebuilt data fields to struct**

```go
type TowerWorkshopScene struct {
	switcher       Switcher
	blueprintStore *descriptor.BlueprintStore
	abilityStore   *descriptor.AbilityStore

	// 预制数据（只读，从配置加载）
	prebuiltAbilities []*descriptor.AbilityDescriptor
	prebuiltBlueprints []descriptor.TowerBlueprint

	tab     int
	hover   int
	scrollY float64
	// ...
}
```

**Step 3: Load prebuilt data in constructor**

In `NewTowerWorkshopScene`, load prebuilt abilities from `descriptor.GlobalDescriptorTable()` (filter `Prebuilt == true`) and prebuilt blueprints from `descriptor.LoadPrebuiltBlueprints()`.

**Step 4: Update tab rendering**

Tab labels array: `[4]string{"预制能力", "我的能力", "预制塔", "我的蓝图"}`

Adjust `wsTabW` if needed to fit 4 tabs (may need to reduce to ~100 or use dynamic sizing).

**Step 5: Update card list rendering per tab**

- Tab 0 (预制能力): iterate `prebuiltAbilities`, show label + icon + category badge + "只读" marker
- Tab 1 (我的能力): existing logic (from `abilityStore.List()`)
- Tab 2 (预制塔): iterate `prebuiltBlueprints`, show name + category badge + abilities summary + "只读" marker
- Tab 3 (我的蓝图): existing logic (from `blueprintStore.List()`)

**Step 6: Disable edit/delete for prebuilt tabs (0 and 2)**

Prebuilt cards: click → show detail panel (read-only). No edit/delete buttons.

**Step 7: Run `make test` and `make run` to verify visually**

**Step 8: Commit**

```bash
git add internal/scene/tower_workshop.go
git commit -m "feat: expand workshop to 4 tabs with prebuilt content"
```

---

### Task 10: Implement "Copy to Custom" functionality

**Files:**
- Modify: `internal/scene/tower_workshop.go` (add copy button and logic)

**Step 1: Add "复制为自定义" button to prebuilt card detail panel**

When a prebuilt ability card is clicked (Tab 0), show a detail overlay with:
- Ability name, icon, category
- Pipeline composition visualization (reuse from ability_edit.go's read-only view)
- Bottom button: "复制为自定义"

**Step 2: Implement copy logic for abilities**

```go
func (s *TowerWorkshopScene) copyAbilityToCustom(desc *descriptor.AbilityDescriptor) {
	ca := descriptor.CustomAbility{
		ID:   fmt.Sprintf("ca_%d", time.Now().UnixMilli()),
		Name: desc.Label + "（副本）",
		Desc: *desc, // deep copy
	}
	ca.Desc.Prebuilt = false
	ca.Desc.ID = ca.ID
	if err := s.abilityStore.Save(ca); err != nil {
		hud.ShowToast("保存失败: " + err.Error())
		return
	}
	s.tab = wsTabCustomAbilities
	hud.ShowToast("已复制到我的能力")
}
```

**Step 3: Implement copy logic for blueprints (same pattern)**

```go
func (s *TowerWorkshopScene) copyBlueprintToCustom(bp *descriptor.TowerBlueprint) {
	copy := *bp
	copy.ID = fmt.Sprintf("bp_%d", time.Now().UnixMilli())
	copy.Name = bp.Name + "（副本）"
	copy.Prebuilt = false
	if err := s.blueprintStore.Save(copy); err != nil {
		hud.ShowToast("保存失败: " + err.Error())
		return
	}
	s.tab = wsTabCustomBlueprints
	hud.ShowToast("已复制到我的蓝图")
}
```

**Step 4: Run `make test` and visual verification**

**Step 5: Commit**

```bash
git add internal/scene/tower_workshop.go
git commit -m "feat: copy-to-custom for prebuilt abilities and blueprints"
```

---

## Phase 4: Cleanup

### Task 11: Remove old abilities.json dependency

**Files:**
- Modify: `internal/scene/loading.go:87` (remove `abilities.InitConfigAbilities()`)
- Modify: `internal/scene/game.go:137` (remove HeadlessMode path)
- Modify: `internal/config/ability_config.go` (remove `LoadAbilityTable`, keep types and `GlobalAbilityTable`)
- Delete: `internal/core/tower/abilities/config_ability.go` (entire file)

**Step 1: Ensure all tests still pass with InitConfigAbilities removed**

`InitDescriptorAbilities` now fully populates both `tower.Registry` and `AbilityTable`. Removing `InitConfigAbilities` should be safe since descriptors override all ConfigAbility registrations anyway.

**Step 2: Remove `abilities.InitConfigAbilities()` from loading.go:87 and game.go:137**

**Step 3: Remove `LoadAbilityTable()` from config (keep `GlobalAbilityTable()` and `MergeAbilityTable()`)**

**Step 4: Delete `config_ability.go`**

**Step 5: Run `make test`**

Fix any compilation errors from removed imports. Tests that directly called `InitConfigAbilities` need updating to only call `InitDescriptorAbilities`.

**Step 6: Commit**

```bash
git add -A
git commit -m "refactor: remove config_ability.go and abilities.json dependency"
```

---

### Task 12: Remove old classic-presets.json dependency

**Files:**
- Delete: `config/towers/classic-presets.json`
- Modify: `internal/config/classic_preset_config.go` — delete or gut (remove `LoadClassicPresets`, keep nothing)
- Remove from `loading.go` if not already done in Task 8

**Step 1: Remove file and loader**

**Step 2: Run `make test`**

Fix compilation errors. Update any tests that referenced classic presets.

**Step 3: Commit**

```bash
git add -A
git commit -m "refactor: remove classic-presets.json, replaced by prebuilt-blueprints.json"
```

---

### Task 13: Delete abilities.json

**Files:**
- Delete: `config/towers/abilities.json`
- Grep for any remaining references and fix

**Step 1: Delete file**

**Step 2: Run `make test`**

**Step 3: Run `make lint`**

**Step 4: Commit**

```bash
git add -A
git commit -m "chore: delete abilities.json, single source is ability-descriptors.json"
```

---

### Task 14: Update indexes and docs

**Step 1: Run `make index`**

**Step 2: Update design doc status**

**Step 3: Commit**

```bash
git add -A
git commit -m "docs: update indexes after ability/blueprint unification"
```
