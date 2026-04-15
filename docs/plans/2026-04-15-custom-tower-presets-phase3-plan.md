# Custom Tower Presets — Phase 3 Implementation Plan

> **For Claude:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to implement this plan task-by-task.

**Goal:** Build a primitive-level ability editor that lets players compose custom abilities from triggers/conditions/selectors/effects, and integrate it into the blueprint workflow. Players can create an ability from scratch, use it in a blueprint, and play with it in Campaign.

**Architecture:** A new AbilityEditScene implements a slot-based pipeline composer UI. Each pipeline has 4 slots (trigger/conditions/selector/effects) with dropdown-style pickers for each primitive. Custom abilities are persisted alongside blueprints, referenced by ID. The blueprint editor's Step 3 gains a "自定义能力" button that launches the ability editor.

**Tech Stack:** Go 1.24+, Ebitengine v2.9.9, existing ui/ components, descriptor engine from Phase 1-2

**Depends on:** Phase 1 + Phase 2 complete

---

## Task 1: Custom ability persistence (AbilityStore)

自定义能力需要持久化存储，类似 BlueprintStore。

**Files:**
- Create: `internal/core/tower/descriptor/ability_store.go`
- Test: `tests/core/descriptor_ability_store_test.go`

**Requirements:**

```go
type CustomAbility struct {
    ID        string            `json:"id"`
    Name      string            `json:"name"`
    Desc      AbilityDescriptor `json:"descriptor"` // 完整描述符（内含 pipelines）
    CreatedAt string            `json:"createdAt,omitempty"`
}

type AbilityStore struct {
    storage persistence.Storage
    data    abilityStoreData
}

const maxCustomAbilities = 50
const customAbilitiesKey = "custom_abilities"

func NewAbilityStore(s persistence.Storage) *AbilityStore
func (as *AbilityStore) Save(ca CustomAbility) error
func (as *AbilityStore) Get(id string) (*CustomAbility, error)
func (as *AbilityStore) Delete(id string) error
func (as *AbilityStore) List() []CustomAbility
func (as *AbilityStore) Count() int
```

**Tests:** CRUD cycle, max limit, update, persistence across instances, list returns copies.

**Commit:** `feat: add AbilityStore for custom ability persistence`

---

## Task 2: Primitive registry with metadata for UI

UI 需要知道每种原语的标签、参数 schema、费用。创建一个带元数据的注册表。

**Files:**
- Create: `internal/core/tower/descriptor/primitive_meta.go`
- Test: `tests/core/descriptor_primitive_meta_test.go`

**Requirements:**

```go
// PrimitiveMeta UI 展示用的原语元数据。
type PrimitiveMeta struct {
    ID       string       `json:"id"`
    Label    string       `json:"label"`    // 中文标签
    Category string       `json:"category"` // trigger/condition/selector/effect
    Cost     int          `json:"cost"`     // 费用
    Params   []ParamMeta  `json:"params"`   // 参数列表
}

type ParamMeta struct {
    Key      string  `json:"key"`      // 参数键名
    Label    string  `json:"label"`    // 中文标签
    Type     string  `json:"type"`     // scaler/float/int/string/bool
    Default  float64 `json:"default"`  // 默认值
    Min, Max float64 `json:"min,max"`  // 值域
}

// 全局原语元数据注册
func AllTriggerMeta() []PrimitiveMeta
func AllConditionMeta() []PrimitiveMeta
func AllSelectorMeta() []PrimitiveMeta
func AllEffectMeta() []PrimitiveMeta
```

为每个已实现的原语定义元数据。例如：
- `chance` condition: `{ID:"chance", Label:"概率", Cost:1, Params:[{Key:"rate", Label:"触发概率", Type:"scaler", Default:0.3, Min:0, Max:1}]}`
- `stun` effect: `{ID:"stun", Label:"眩晕", Cost:3, Params:[{Key:"duration", Label:"持续时间", Type:"scaler", Default:0.5, Min:0.1, Max:5}]}`

**Tests:** AllTriggerMeta returns 4+ items, AllConditionMeta returns 11+ items, all have non-empty Label/ID.

**Commit:** `feat: add primitive metadata registry for ability editor UI`

---

## Task 3: AbilityEditScene — skeleton + pipeline list

能力编辑器场景骨架：管线列表 + 添加/删除管线。

**Files:**
- Create: `internal/scene/ability_edit.go`
- Modify: `internal/scene/game.go` (add to currentSceneName)

**Requirements:**

场景布局：
```
┌──────────────────────────────────────────────┐
│  [能力名称: ___________]  费用: 12    [返回] │
├──────────────────────────────────────────────┤
│  管线 1                              [删除]  │
│  ┌──────────────────────────────────────┐    │
│  │ 触发: [命中时 ▼]                     │    │
│  │ 条件: [概率 30%]            [+条件]  │    │
│  │ 目标: [当前目标 ▼]                   │    │
│  │ 效果: [眩晕 0.5s]          [+效果]  │    │
│  └──────────────────────────────────────┘    │
│                                              │
│  [+添加管线]                                  │
├──────────────────────────────────────────────┤
│              [取消]  [保存]                    │
└──────────────────────────────────────────────┘
```

Scene struct:
```go
type AbilityEditScene struct {
    switcher     Switcher
    ability      descriptor.CustomAbility
    isNew        bool
    abilityStore *descriptor.AbilityStore

    // UI 状态
    pipelines    []pipelineEditState // 每条管线的编辑状态
    activePicker *pickerState        // 当前打开的选择器（nil=无）
    nameText     string
}

type pipelineEditState struct {
    TriggerID    string
    Conditions   []conditionEditState
    SelectorID   string
    SelectorParams map[string]interface{}
    Effects      []effectEditState
}
```

本 task 只实现：
- 场景骨架（标题、管线列表面板、底部按钮）
- 添加/删除管线
- 每条管线显示 4 行占位文本
- 返回/保存导航

**Commit:** `feat: add AbilityEditScene skeleton with pipeline list`

---

## Task 4: Primitive picker — 下拉选择器组件

通用的原语选择器弹窗，用于选择 trigger/condition/selector/effect 类型。

**Files:**
- Create: `internal/render/hud/primitive_picker.go`

**Requirements:**

```go
type PrimitivePickerData struct {
    Title   string
    Options []descriptor.PrimitiveMeta
    Current string // 当前选中的 ID
    X, Y    float32 // 弹窗位置
}

func DrawPrimitivePicker(screen *ebiten.Image, data PrimitivePickerData)
func PrimitivePickerHitTest(px, py float64, data PrimitivePickerData) int // 返回选中索引或 -1
```

渲染为浮动面板，列出可选项：每项显示 标签 + 费用 + 简要参数描述。当前选中项高亮。点击选中后关闭。

**Commit:** `feat: add PrimitivePicker dropdown component for ability editor`

---

## Task 5: Pipeline editor — trigger + selector slots

实现管线编辑器中的 trigger 和 selector 选择。

**Files:**
- Modify: `internal/scene/ability_edit.go`

**Requirements:**

每条管线渲染 4 行：
1. **触发器行**: `触发: [命中时 ▼]` — 点击打开 trigger picker
2. **条件行**: `条件: (无)  [+条件]` — 占位，Task 6 实现
3. **目标行**: `目标: [当前目标 ▼]` — 点击打开 selector picker
4. **效果行**: `效果: (无)  [+效果]` — 占位，Task 7 实现

点击 trigger/selector 的下拉按钮 → 设置 `s.activePicker`，渲染 PrimitivePicker。
选择后更新 `pipelineEditState.TriggerID` / `SelectorID`。

Selector 选择后，如果该 selector 有参数（如 aoeRadius 的 radius），显示参数行（Task 8 实现参数编辑）。

**Commit:** `feat: implement trigger + selector slots in pipeline editor`

---

## Task 6: Pipeline editor — conditions slot

实现条件列表：添加/删除条件，每条条件可选类型。

**Files:**
- Modify: `internal/scene/ability_edit.go`

**Requirements:**

条件行显示已选条件列表 + [+条件] 按钮：
```
条件: [概率 30%] [HP<50%] [+条件]
```

- [+条件] → 打开 condition picker
- 点击已有条件 → 弹出选择器更改类型 / 删除按钮
- 多条件为 AND 关系

`conditionEditState`:
```go
type conditionEditState struct {
    TypeID string
    Params map[string]interface{} // 参数值
}
```

**Commit:** `feat: implement conditions slot in pipeline editor`

---

## Task 7: Pipeline editor — effects slot

实现效果列表：添加/删除效果。

**Files:**
- Modify: `internal/scene/ability_edit.go`

**Requirements:**

效果行显示已选效果列表 + [+效果] 按钮：
```
效果: [眩晕 0.5s] [灼烧 10%] [+效果]
```

- [+效果] → 打开 effect picker
- 点击已有效果 → 弹出选择器更改类型 / 删除
- 一条管线可有多个效果

`effectEditState`:
```go
type effectEditState struct {
    TypeID string
    Params map[string]interface{}
}
```

**Commit:** `feat: implement effects slot in pipeline editor`

---

## Task 8: Parameter editor — Scaler 输入组件

原语参数的编辑 UI。每个参数显示为可调节的数值行。

**Files:**
- Create: `internal/render/hud/param_editor.go`
- Modify: `internal/scene/ability_edit.go`

**Requirements:**

参数编辑行：
```
持续时间: [linear ▼] base=[0.5] pot=[0.1]    // scaler 类型参数
触发概率: [fixed ▼]  value=[0.3]              // fixed scaler
冷却时间: [3.0] s                              // 纯 float 参数
```

组件：
```go
type ParamEditorData struct {
    Params []ParamEditRow
    X, Y   float32
    W      float32
}

type ParamEditRow struct {
    Meta        descriptor.ParamMeta
    ScalerType  string  // "linear"/"fixed"/"diminishing"/"capped" (scaler 类型参数)
    Values      map[string]float64 // base/potential/value/k/cap 等
}

func DrawParamEditor(screen *ebiten.Image, data ParamEditorData)
```

数值调节用 +/- 按钮（步长从 ParamMeta.Min/Max 推导），或直接点击数值区域输入。
本 phase 先实现 +/- 按钮方式，文本输入留给未来。

**Commit:** `feat: add parameter editor component with scaler type switching`

---

## Task 9: AbilityEditScene — 编辑状态 ↔ AbilityDescriptor 转换

将 UI 编辑状态序列化为 AbilityDescriptor（保存时），和从 AbilityDescriptor 反序列化为编辑状态（加载已有能力时）。

**Files:**
- Create: `internal/core/tower/descriptor/edit_state.go`
- Test: `tests/core/descriptor_edit_state_test.go`

**Requirements:**

```go
// EditStateToDescriptor 将编辑状态转为可执行的 AbilityDescriptor。
func EditStateToDescriptor(name string, pipelines []PipelineEditState) (*AbilityDescriptor, error)

// DescriptorToEditState 将 AbilityDescriptor 转为编辑状态。
func DescriptorToEditState(desc *AbilityDescriptor) []PipelineEditState

// PipelineEditState 管线编辑状态（与 scene 层共享的纯数据结构）。
type PipelineEditState struct {
    TriggerID      string
    Conditions     []ComponentEditState
    SelectorID     string
    SelectorParams map[string]float64
    Effects        []ComponentEditState
}

type ComponentEditState struct {
    TypeID string
    Params map[string]float64
}
```

**Tests:**
- Roundtrip: stunChance descriptor → EditState → Descriptor → 验证等价
- Multi-pipeline roundtrip
- Empty conditions/effects

**Commit:** `feat: add EditState ↔ AbilityDescriptor conversion`

---

## Task 10: Blueprint editor Step 3 — 集成自定义能力

升级 blueprint editor 的 Step 3，添加「自定义能力」入口。

**Files:**
- Modify: `internal/scene/blueprint_edit.go` (drawAbilitiesStep, handleAbilitiesInput)

**Requirements:**

在可选能力列表顶部添加一个特殊区域：
```
[我的自定义能力]  →  列出 AbilityStore 中的自定义能力
[+创建新能力]     →  跳转 AbilityEditScene
```

自定义能力和预制能力在同一列表混排，但用不同背景色区分。
自定义能力也有 cost，参与预算计算。

点击 [+创建新能力] → 保存当前 blueprint 临时状态 → 跳转 AbilityEditScene → 返回时恢复状态 + 刷新能力列表。

需要 BlueprintEditScene 持有 AbilityStore 引用。

**Commit:** `feat: integrate custom abilities into blueprint editor Step 3`

---

## Task 11: 自定义能力注册到描述符引擎

让自定义能力在运行时可被 Interpreter 执行。

**Files:**
- Modify: `internal/core/tower/descriptor/init.go`
- Modify: `internal/scene/stage.go` (init 流程)

**Requirements:**

在 `InitDescriptorAbilities` 之后，加载 AbilityStore 中的自定义能力并注册：

```go
func RegisterCustomAbilities(store *AbilityStore) {
    for _, ca := range store.List() {
        desc := ca.Desc
        desc.ID = ca.ID // 确保 ID 一致
        ability := NewDescriptorAbility(&desc)
        tower.Register(ability)
    }
}
```

蓝图引用自定义能力 ID 时，Interpreter 能在 Registry 中找到对应的 DescriptorAbility。

**Commit:** `feat: register custom abilities into descriptor engine at init`

---

## Task 12: New primitives — stepped scaler + teleport effect

Phase 3 设计文档中规划的最后两个原语。

**Files:**
- Modify: `internal/core/tower/descriptor/scaler.go`
- Modify: `internal/core/tower/descriptor/effect.go` + `effect_result.go`
- Modify: `internal/core/tower/descriptor/descriptor.go`
- Test: `tests/core/descriptor_scaler_v3_test.go`
- Test: `tests/core/descriptor_effect_v3_test.go`

**Requirements:**

**SteppedScaler** — 按阈值分段：
```go
type SteppedScaler struct {
    Steps []StepThreshold // 升序排列的 {Strength, Value} 对
}
// 找到 strength 所在区间，线性插值
```

**TeleportEffect** — 将敌人沿路径回推：
```go
type TeleportEffect struct {
    Distance Scaler // 回推距离
}
// 返回 EffTypeTeleport
```

Update ParseScaler ("stepped") and parseEffect ("teleport").

**Commit:** `feat: add stepped scaler and teleport effect primitives`

---

## Task 13: Autoplay 验证 + 最终清理

**Step 1:** Full build + test

```bash
go build ./...
go test ./tests/core/ -count=1 -race
go test ./tests/contracts/ -count=1 -race
go test ./tests/regression/ -count=1 -race
```

**Step 2:** Autoplay smoke test

```bash
go run cmd/autoplay/main.go --scenario attack-style-coverage
```

**Step 3:** Final commit

```
chore: Phase 3 complete — ability editor, custom ability persistence, primitive composition UI
```

---

## 依赖图

```
Task 1 (AbilityStore) ──────────────────────┐
Task 2 (PrimitiveMeta) ─┐                   │
                         ├→ Task 4 (Picker)  │
Task 3 (Scene skeleton) ─┤                   │
                         └→ Task 5 (trigger/selector)
                              → Task 6 (conditions)
                              → Task 7 (effects)
                              → Task 8 (param editor)
                                    ↓
                              Task 9 (EditState↔Descriptor)
                                    ↓
Task 1 ──────────────────→ Task 10 (Step 3 集成)
                                    ↓
                              Task 11 (运行时注册)
                                    ↓
Task 12 (新原语) ──────→ Task 13 (最终验证)
```

Tasks 1, 2, 3, 12 可并行（无依赖）。
Tasks 5-8 串行（依赖 3+4）。
Task 10 依赖 1+9。
