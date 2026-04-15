# Custom Tower Presets — Phase 2 Implementation Plan

> **For Claude:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to implement this plan task-by-task.

**Goal:** Add a "Custom" tab to the build menu that displays player blueprints, allow blueprint creation via a simple editing scene, and extend the primitive set for finer-grained ability composition.

**Architecture:** Build menu gets tab bar (All / Custom), blueprint towers loaded from BlueprintStore and appended to towerDefs. A new BlueprintEditScene lets players pick attack style, tiers, and abilities from the existing pool. New primitives (conditions, selectors, effects, scalers) expand the descriptor engine's expressiveness.

**Tech Stack:** Go 1.24+, Ebitengine v2.9.9, existing ui/ components (DrawButtonRow, CardGrid, PanelBox, ScrollRegion)

**Design doc:** `docs/plans/2026-04-15-custom-tower-presets-design.md`

**Depends on:** Phase 1 complete (descriptor engine, blueprint CRUD, budget system)

---

## Task 1: Build menu tab bar — rendering

给建塔面板添加 tab 切换栏（全部 / 自定义）。纯渲染，不含数据过滤。

**Files:**
- Modify: `internal/render/hud/build_menu.go`
- Test: `tests/core/hud_build_menu_test.go`

**Step 1: Write failing tests**

验证 `DrawBuildMenu` 在 `BuildMenuData.Tabs` 非空时渲染 tab 按钮。
测试 `BuildMenuHitTest` 在 tab 区域返回 tab index。

**Step 2: Implement**

在 `BuildMenuData` 中添加：

```go
type BuildMenuData struct {
    // ... 现有字段 ...
    Tabs      []string // tab 名称列表（如 ["全部", "自定义"]）
    ActiveTab int      // 当前激活 tab 索引
}
```

在 `DrawBuildMenu` 的面板顶部（Cards 上方）使用 `ui.DrawButtonRow` 渲染 tab 栏：

```go
// tab 栏渲染（面板标题下方）
if len(data.Tabs) > 1 {
    tabBtns := make([]ui.ButtonState, len(data.Tabs))
    for i, name := range data.Tabs {
        tabBtns[i] = ui.ButtonState{
            Label:  name,
            Active: i == data.ActiveTab,
        }
    }
    ui.DrawButtonRow(screen, tabX, tabY, tabBtns, theme.BuildFactionTabH, 0, theme.BuildFactionTabGap)
}
```

在 `BuildMenuHitTest` 中添加 tab 点击检测，返回 `(cardIdx int, tabIdx int)` 或用负值约定区分。

**Step 3: Commit**

```
feat: add tab bar rendering to build menu
```

---

## Task 2: Build menu tab state + card filtering

将 tab 状态接入 stage 输入循环，按 tab 过滤显示的 cards。

**Files:**
- Modify: `internal/scene/stage.go` (buildBuildMenuData, ~line 3087)
- Modify: `internal/scene/stage_input.go` (modeBuildMenu 输入处理)
- Modify: `internal/scene/stage_types.go` (新字段 buildMenuTab)

**Step 1: Write failing tests**

```go
func TestBuildMenuData_TabFiltering(t *testing.T) {
    // 给定 3 个 standard + 2 个 blueprint towers
    // ActiveTab=0 ("全部") → 5 cards
    // ActiveTab=1 ("自定义") → 2 cards
}
```

**Step 2: Implement**

StageScene 添加 `buildMenuTab int` 字段。

`buildBuildMenuData()` 修改：
1. 检查 `ruleset.AllowCustomBlueprints()` 且 blueprintStore 非空 → 设 Tabs
2. `ActiveTab == 0`（全部）→ 显示所有 towerDefs
3. `ActiveTab == 1`（自定义）→ 只显示 blueprint 来源的 TowerDef（通过 Key 前缀 "bp_" 识别）

`modeBuildMenu` 输入处理：tab 点击时更新 `s.buildMenuTab`，不关闭面板。

**Step 3: Commit**

```
feat: add tab state and card filtering to build menu
```

---

## Task 3: Blueprint towers 接入 loadTowerDefsForMode

让 BlueprintStore 中的蓝图作为 TowerDef 追加到可建造列表。

**Files:**
- Modify: `internal/scene/stage.go` (loadTowerDefsForMode, NewStageSceneWithOpts)
- Test: `tests/core/stage_blueprint_integration_test.go`

**Step 1: Write failing tests**

```go
func TestLoadTowerDefsForMode_WithBlueprints(t *testing.T) {
    // Campaign 模式 + 2 个蓝图 → towerDefs 包含 standard + 2 个 blueprint defs
    // Classic 模式 + 2 个蓝图 → towerDefs 不包含 blueprint defs
}
```

**Step 2: Implement**

修改 `loadTowerDefsForMode`：

```go
func loadTowerDefsForMode(ruleset gamemode.TowerRuleset, pm *persistence.ProgressManager, bpStore *descriptor.BlueprintStore) []tower.TowerDef {
    // ... 现有逻辑 ...

    // 追加蓝图塔（如果允许）
    if ruleset.AllowCustomBlueprints() && bpStore != nil {
        tierPresets := config.GlobalTierPresets()
        budgetRules := descriptor.GlobalBudgetRules()
        abilityCosts := descriptor.GlobalAbilityCosts()
        for _, bp := range bpStore.List() {
            def := descriptor.BlueprintToTowerDef(&bp, tierPresets, budgetRules, abilityCosts)
            defs = append(defs, def)
        }
    }
    return defs
}
```

同时需要在 `NewStageSceneWithOpts` 中传入 BlueprintStore（从 Game struct 获取或 DefaultBlueprintStore）。

**Step 3: Commit**

```
feat: integrate blueprint towers into loadTowerDefsForMode
```

---

## Task 4: BlueprintEditScene — 场景骨架

新建蓝图编辑场景的最小骨架：进入/退出、空面板渲染。

**Files:**
- Create: `internal/scene/blueprint_edit.go`
- Modify: `internal/scene/scene.go` (SceneID 注册)

**Step 1: Implement scene skeleton**

```go
type BlueprintEditScene struct {
    switcher  Switcher
    blueprint *descriptor.TowerBlueprint
    isNew     bool
    // UI state
    step      int // 0=attackStyle, 1=tiers, 2=abilities, 3=preview
}

func (s *BlueprintEditScene) Update() error { ... }
func (s *BlueprintEditScene) Draw(screen *ebiten.Image) { ... }
```

场景流程分 4 步：
1. 选攻击模式（scatter/wideBeam/...）
2. 选属性档位（S/B/D）+ 专精
3. 选能力（从预制能力池挑选，最多 maxSlots 个）
4. 预览 + 命名 + 保存

**Step 2: Wire up navigation**

从建塔面板的「+新建」按钮或蓝图管理入口进入。ESC 返回。

**Step 3: Commit**

```
feat: add BlueprintEditScene skeleton with 4-step wizard
```

---

## Task 5: BlueprintEditScene — Step 1: 攻击模式选择

**Files:**
- Modify: `internal/scene/blueprint_edit.go`

**Step 1: Implement**

使用 `ui.CardGrid` 显示 7 个攻击模式卡片（projectile + 6 个带 attackStyle 的能力）。
每个卡片显示：精灵预览 + 名称 + 费用 + 简要描述。
点击选择后进入 Step 2。

卡片数据从 `descriptor.GlobalDescriptorTable()` 筛选 `attackStyle != ""` 的描述符。

**Step 2: Commit**

```
feat: add attack style selection (Step 1) to blueprint editor
```

---

## Task 6: BlueprintEditScene — Step 2: 属性档位 + 专精

**Files:**
- Modify: `internal/scene/blueprint_edit.go`

**Step 1: Implement**

3 行属性选择器：伤害 / 攻速 / 射程，每行 3 个按钮 (S/B/D)。
底部专精选择：3 个 radio 按钮 (伤害/攻速/射程)。
实时显示预算消耗。

**Step 2: Commit**

```
feat: add tier + specialty selection (Step 2) to blueprint editor
```

---

## Task 7: BlueprintEditScene — Step 3: 能力选择

**Files:**
- Modify: `internal/scene/blueprint_edit.go`

**Step 1: Implement**

使用 `ui.CardGrid` + `ui.ScrollRegion` 显示可选能力池。
排除攻击类能力（已在 Step 1 选定）和 disabled abilities。
每个卡片：图标 + 名称 + 费用 + 缩放描述。
点击添加到已选列表，再点击移除。
实时预算条（used / cap）。
超预算时新增卡片灰显不可选。

**Step 2: Commit**

```
feat: add ability selection (Step 3) to blueprint editor
```

---

## Task 8: BlueprintEditScene — Step 4: 预览 + 保存

**Files:**
- Modify: `internal/scene/blueprint_edit.go`

**Step 1: Implement**

显示完整蓝图摘要：
- 攻击模式 + 精灵预览
- 属性档位 + 专精
- 已选能力列表
- 预算使用 / 建造费用
- 名称输入框（或默认生成名称）
- [保存] [返回修改] 按钮

保存调用 `blueprintStore.Save(bp)`，返回上一场景。

**Step 2: Commit**

```
feat: add preview + save (Step 4) to blueprint editor
```

---

## Task 9: Build menu「+新建」入口

在建塔面板「自定义」tab 末尾添加一个「+」卡片，点击进入 BlueprintEditScene。

**Files:**
- Modify: `internal/render/hud/build_menu.go`
- Modify: `internal/scene/stage_input.go`
- Modify: `internal/scene/stage.go`

**Step 1: Implement**

`buildBuildMenuData()` 在 custom tab 末尾追加一个特殊 card：
```go
BuildCardVM{Key: "__new_blueprint__", Label: "+新建", Buildable: false, IsCreateBtn: true}
```

`BuildMenuHitTest` 返回该 card 时，stage_input 切换到 BlueprintEditScene。

**Step 2: Commit**

```
feat: add "+create" card in custom tab to launch blueprint editor
```

---

## Task 10: New primitives — Conditions

扩展条件门：isBoss, notBoss, every(n), buffActive, buffAbsent。

**Files:**
- Modify: `internal/core/tower/descriptor/condition.go`
- Modify: `internal/core/tower/descriptor/descriptor.go` (parseCondition 新分支)
- Test: `tests/core/descriptor_condition_v2_test.go`

**Step 1: Write failing tests**

```go
func TestIsBossCondition(t *testing.T) { ... }
func TestNotBossCondition(t *testing.T) { ... }
func TestEveryCondition(t *testing.T) { ... }
func TestBuffActiveCondition(t *testing.T) { ... }
func TestBuffAbsentCondition(t *testing.T) { ... }
```

**Step 2: Implement**

ConditionCtx 新增 `IsBoss bool` 和 `ActiveBuffIDs []string`。

5 个新 Condition 类型 + parseCondition 新增 5 个 case。

**Step 3: Commit**

```
feat: add 5 new condition primitives (isBoss/notBoss/every/buffActive/buffAbsent)
```

---

## Task 11: New primitives — Selectors

扩展选择器：cone, ring360, random。

**Files:**
- Modify: `internal/core/tower/descriptor/selector.go`
- Modify: `internal/core/tower/descriptor/descriptor.go` (parseSelector 新分支)
- Test: `tests/core/descriptor_selector_v2_test.go`

**Step 1: Write failing tests + implement + commit**

```
feat: add 3 new selector primitives (cone/ring360/random)
```

---

## Task 12: New primitives — Effects + Scalers

扩展效果：crit (独立 effect), purge。
扩展缩放器：diminishing, capped。

**Files:**
- Modify: `internal/core/tower/descriptor/effect.go`
- Modify: `internal/core/tower/descriptor/scaler.go`
- Modify: `internal/core/tower/descriptor/descriptor.go`
- Test: `tests/core/descriptor_effect_v2_test.go`
- Test: `tests/core/descriptor_scaler_v2_test.go`

**Step 1: Write failing tests + implement + commit**

```
feat: add crit/purge effects and diminishing/capped scalers
```

---

## Task 13: Blueprint management — 编辑 + 删除

在建塔面板中长按/右键蓝图卡片弹出管理菜单（编辑/删除/复制）。

**Files:**
- Modify: `internal/render/hud/build_menu.go`
- Modify: `internal/scene/stage_input.go`

**Step 1: Implement**

长按 blueprint card（500ms）或右键 → 弹出 context menu：
- 编辑 → 进入 BlueprintEditScene（传入已有 blueprint）
- 删除 → 确认对话框 → blueprintStore.Delete
- 复制 → blueprintStore.Save(clone with new ID)

**Step 2: Commit**

```
feat: add blueprint management (edit/delete/copy) via long-press menu
```

---

## Task 14: Autoplay 验证 + 最终清理

**Step 1: Autoplay smoke test**

```bash
go run cmd/autoplay/main.go --scenario attack-style-coverage
```

**Step 2: 全量测试**

```bash
make check-all
go test ./tests/contracts/ -v -race
go test ./tests/regression/ -v -race
```

**Step 3: Final commit**

```
chore: Phase 2 complete — build menu tabs, blueprint editor, new primitives
```

---

## 依赖图

```
Task 1 (Tab rendering) → Task 2 (Tab state + filtering) → Task 9 (+新建入口)
                                    ↑
Task 3 (Blueprint→towerDefs) ───────┘

Task 4 (Scene skeleton) → Task 5 (攻击模式) → Task 6 (属性档位) → Task 7 (能力选择) → Task 8 (预览保存)

Task 10 (Conditions) ┐
Task 11 (Selectors)  ├─ 独立，可并行
Task 12 (Effects+Scalers) ┘

Task 13 (Blueprint管理) → depends on Tasks 4-8

Task 14 (Final验证) → depends on all
```

Tasks 1-3 (建塔菜单) 和 Tasks 4-8 (编辑场景) 可并行推进。
Tasks 10-12 (新原语) 完全独立，可随时并行。
