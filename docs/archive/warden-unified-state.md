# 战灵 State 统一重构方案

## 1. 当前问题

5 种战灵各自有完全不同的 State 结构体，没有共享基座，导致：

1. **渲染分发**: `draw_warden.go` 必须对 5 种具体类型做 type-assert
2. **配置断联**: JSON 配置有 5 级属性，但 Go 代码硬编码 Lv.1 数值；升级系统未实现
3. **代码重复**: 轨道运动逻辑在 core/chain/skystrike 中复制了 3 份
4. **字段不一致**: 相同概念（位置、伤害、渲染数据）在不同类型中命名和结构各异
5. **JSON 冗余**: chain/skystrike 每级都填充 `effectDps:0, effectDuration:0, moveSpeed:0`；envoy 完全没有 damage 字段
6. **成长硬编码**: `warden.go` 对所有类型使用 `+2/kill +5/wave`，忽略了 JSON 中的分类型成长配置

## 2. 当前字段清单

| 字段 | Prince | Core | Chain | Skystrike | Envoy |
|------|--------|------|-------|-----------|-------|
| **位置 (X, Y)** | State | State | State | State | State |
| **MoveSpeed** | State | State | State | State | - |
| **OrbitAngle** | - | State | State | State | - |
| **Damage** | State | State | State | State | - |
| **AttackInterval** | State | State | State | State | - |
| **AttackTimer** | - | State | State | State | - |
| **Range** | - | State | State | State | - |
| **AoERadius** | State | State | - | State | - |
| **EffectDPS** | State | - | - | - | - |
| **EffectDuration** | State | - | - | - | - |
| **AoE 特殊字段** | - | - | - | State(4个) | - |
| **TowerBonus** | - | - | State | - | - |
| **BoostedSet** | - | - | State | - | - |
| **PossessedTower** | - | - | - | - | State |
| **PossessDur/CdDur** | - | - | - | - | State |
| **DamageBonus** | - | - | - | - | State |
| **Phase + Timer** | State | - | - | - | State |
| **冲刺字段** | State(5个) | - | - | - | - |
| **Trails** | State | - | - | - | - |
| **渲染 (LastTarget/ShootTimer)** | - | State | State | State | - |
| **打击渲染** | - | - | - | State(3个) | - |

## 3. 统一设计

### 3.1 核心原则

拆分为 **共享基座**（嵌入所有 State 结构体）+ **类型特有扩展**。基座处理所有公共字段；扩展只持有独有机制。

### 3.2 共享基座: `WardenState`

所有战灵都是地图上可见的实体。

```go
// WardenState 嵌入到每种战灵类型的状态中。
// 包含 3 种及以上战灵类型共享的所有字段。
type WardenState struct {
    // --- 位置与移动 ---
    X, Y       float64 // 当前像素位置（5 种战灵都有）
    MoveSpeed  float64 // 移动速度 px/s（0 = 不移动/瞬移）
    OrbitAngle float64 // 弧度，轨道运动角度（不使用的类型保持 0）

    // --- 战斗（可选，0 表示无攻击能力）---
    Damage         float64 // 单次伤害
    AttackInterval float64 // 攻击间隔秒数（0 = 不攻击）
    AttackTimer    float64 // 攻击冷却倒计时
    Range          float64 // 攻击范围像素（0 = 不检查范围）
    AoERadius      float64 // AoE 半径（0 = 单体攻击）

    // --- 阶段机（可选，"" 表示常驻模式）---
    Phase string  // 当前阶段（如 "idle"/"dashing"/"cooldown"/"possessing"）
    Timer float64 // 当前阶段倒计时

    // --- 渲染 ---
    LastTargetX float64 // 上次射击目标 X
    LastTargetY float64 // 上次射击目标 Y
    ShootTimer  float64 // 射击线视觉计时器（0 = 不可见）
}
```

**设计决策**:
- **所有战灵都有 X/Y**: 即使 envoy（悬浮在塔上方）。每个战灵都是地图上的可见实体。
- **战斗字段可选**: Envoy 保持 `Damage=0, AttackInterval=0`——行为代码通过 `AttackInterval > 0` 判断是否需要攻击。
- **Phase/Timer 可选**: Core/Chain/Skystrike 不使用阶段机（行为常驻）。Prince 和 Envoy 使用阶段机。不使用阶段机的类型保持 `Phase=""`。
- **渲染字段统一**: 所有能攻击的战灵都需要；不攻击的战灵保持为 0，渲染器跳过射击线。

### 3.3 类型扩展（仅包含真正独有的字段）

```
PrinceExt {
    DashStartX, DashStartY float64  // 冲刺起点
    DashEndX, DashEndY     float64  // 冲刺终点
    DashProgress           float64  // 冲刺进度 0-1
    HitSet                 map[*enemy.Enemy]bool  // 本次冲刺已命中敌人
    Trails                 []FireTrail             // 活跃火焰痕迹
    EffectDPS              float64  // 火焰痕迹每秒伤害
    EffectDuration         float64  // 火焰痕迹持续时间
}

CoreExt {
    // （当前为空——所有字段已移入基座）
    // 未来：进化标志、模式切换状态
}

ChainExt {
    TowerBonus float64               // 每座塔的战力加成值
    BoostedSet map[*tower.Tower]bool // 已加成的塔集合
}

SkystrikeExt {
    AoEDamage   float64  // 特殊 AoE 伤害（不同于基座的 Damage）
    AoEInterval float64  // 特殊 AoE 冷却间隔
    AoETimer    float64  // 特殊 AoE 冷却倒计时
    StrikeX     float64  // AoE 视觉效果中心 X
    StrikeY     float64  // AoE 视觉效果中心 Y
    StrikeTimer float64  // AoE 视觉效果倒计时
}

EnvoyExt {
    PossessedTower   *tower.Tower  // 当前附身的塔
    PossessDuration  float64       // 附身持续时间
    CooldownDuration float64       // 冷却持续时间
    DamageBonus      float64       // 附身时给塔的战力加成
}
```

### 3.4 具体 State 结构体

每种类型嵌入 `WardenState` + 自身扩展：

```go
type PrinceState struct {
    WardenState  // 嵌入基座
    PrinceExt    // 小王子独有
}

type CoreState struct {
    WardenState
    // 当前无扩展字段
}

type ChainState struct {
    WardenState
    ChainExt
}

type SkystrikeState struct {
    WardenState
    SkystrikeExt
}

type EnvoyState struct {
    WardenState
    EnvoyExt
}
```

### 3.5 行为层改动

**基座提供公共辅助方法**（放在 `warden` 包内）：

```go
// MoveOrbit 围绕 (cx, cy) 以 idealDist 为理想距离进行轨道运动。
func (s *WardenState) MoveOrbit(cx, cy, idealDist, dt float64) { ... }

// FindNearest 查找 Range 范围内最近的敌人。
func (s *WardenState) FindNearest(enemies *enemy.Pool) *enemy.Enemy { ... }

// BasicAttack 对最近敌人造成 Damage 伤害，返回目标或 nil。
func (s *WardenState) BasicAttack(ctx *TickContext) *enemy.Enemy { ... }

// DecayShootTimer 按 dt 递减 ShootTimer。
func (s *WardenState) DecayShootTimer(dt float64) { ... }

// CanAttack 返回该战灵是否具有攻击能力（AttackInterval > 0）。
func (s *WardenState) CanAttack() bool { return s.AttackInterval > 0 }
```

**消除重复代码**:
- `moveOrbit` / `chainMoveOrbit` / `skystrikeMoveOrbit` → `WardenState.MoveOrbit(idealDist)`
- `performAttack` / `chainAttack` / `skystrikeAttack` → `WardenState.BasicAttack(ctx)`
- `computeClusterCenter` / `findDensestEnemy` → 包级共享函数

**各类型 Tick 逻辑保持不变**:
- **Prince**: 阶段机（idle→dash→cooldown）+ 火焰痕迹逻辑。行为无变化。
- **Core**: `MoveOrbit(110)` + `BasicAttack` 含斩杀加成。斩杀加成保留在类型代码中。
- **Chain**: 塔增强循环 + `MoveOrbit(100)` + `BasicAttack`。塔增强保留在类型代码中。
- **Skystrike**: `MoveOrbit(120)` + `BasicAttack` + 周期性 AoE。AoE 特殊攻击保留在类型代码中。
- **Envoy**: 阶段机（idle→possessing）。`CanAttack()` 返回 false，无攻击逻辑。

### 3.6 渲染层改动

```go
func DrawWarden(screen *ebiten.Image, w *warden.Warden) {
    // 所有类型：通过接口获取基座状态
    base := w.BaseState() // 返回 *WardenState
    if base == nil { return }

    // 通用：位置检查
    if base.X == 0 && base.Y == 0 { return }

    // 通用：射击线（ShootTimer > 0 时绘制）
    drawShootLine(screen, base, typeColor(w.Type))

    // 类型特有：本体形状
    switch s := w.State.(type) { ... }
}
```

### 3.7 JSON 配置清理

**改动前**（chain Lv.1）：
```json
{ "level": 1, "damage": 15, "attackInterval": 2.0, "moveSpeed": 0, "range": 9999, "aoeRadius": 0, "effectDuration": 0, "effectDps": 0 }
```

**改动后**（chain Lv.1 — 仅保留非零/有意义字段）：
```json
{ "level": 1, "damage": 15, "attackInterval": 2.0, "range": 9999, "towerBonus": 10 }
```

**规则**: JSON 中缺失的字段默认为 `0`。Go 的 `json.Unmarshal` 本身就有这个行为。移除所有显式零值字段。

**Envoy** 保持不写 damage/interval 字段（反序列化后自然为 0 → `CanAttack()` 返回 false）。

### 3.8 等级系统实现（新增）

将 JSON 配置接入运行时属性：

```go
// 在 warden.go 或新文件 warden/level.go 中
func (w *Warden) ApplyLevel(cfg *config.WardenLevel) {
    base := w.BaseState()
    if base == nil { return }
    base.Damage = cfg.Damage
    base.AttackInterval = cfg.AttackInterval
    base.Range = cfg.Range
    base.MoveSpeed = cfg.MoveSpeed
    base.AoERadius = cfg.AoERadius
    // 类型特有字段由各 Behavior 自行应用
}
```

在 `Warden.Tick()` 中当 `PerceivedStrength` 跨过阈值时调用：

```go
func (w *Warden) Tick(ctx *TickContext) {
    w.CalcStrength(ctx.Towers)
    newLevel := w.computeLevel()  // 从 _meta.json 阈值计算
    if newLevel != w.Level {
        w.Level = newLevel
        w.applyLevelStats(newLevel)  // 从 WardenConfig 读取对应等级属性
    }
    behaviors[w.Type].Tick(w, ctx)
}
```

### 3.9 成长参数从配置读取（新增）

替换硬编码的 `+2/+5`：

```go
func (w *Warden) OnKill() {
    w.SelfStrength += w.GrowthOnKill     // 从 JSON 加载
}
func (w *Warden) OnWaveClear() {
    w.SelfStrength += w.GrowthOnWaveClear // 从 JSON 加载
}
```

在 `Warden` 结构体上新增 `GrowthOnKill` / `GrowthOnWaveClear` 字段，创建时从配置填充。

## 4. 行为等价验证表

确认每种战灵的可观测行为在重构后保持一致：

| 战灵 | 移动方式 | 攻击方式 | 特殊能力 | 阶段机 |
|------|---------|---------|---------|--------|
| **Prince** | 向敌群中心冲刺 | 冲刺穿透 AoE | 火焰痕迹（持续灼伤区域） | idle → dashing → cooldown |
| **Core** | 轨道运动(110) 围绕敌群质心 | 基础攻击 + 斩杀加成 | 范围内 4+ 敌人且 AoERadius>0 时 AoE | 常驻 |
| **Chain** | 轨道运动(100) 围绕敌群质心 | 基础攻击（单体） | 全场塔 +战力增强 | 常驻 |
| **Skystrike** | 轨道运动(120) 围绕敌群质心 | 基础攻击（单体） | 周期性 AoE 轰炸 | 常驻 |
| **Envoy** | 瞬移到塔位置 | 无（CanAttack=false） | 附身塔（+战力加成） | idle → possessing |

所有行为保持不变。重构是纯结构性的。

## 5. 渲染器访问接口

让渲染器不需要类型分发就能访问 `WardenState` 的公共字段：

```go
// Stateful 由所有战灵 State 结构体实现。
type Stateful interface {
    Base() *WardenState
}

// 每个 State 结构体通过嵌入自动实现：
func (s *PrinceState) Base() *WardenState    { return &s.WardenState }
func (s *CoreState) Base() *WardenState      { return &s.WardenState }
// ... 其他类型同理

// Warden 便捷方法：
func (w *Warden) BaseState() *WardenState {
    if s, ok := w.State.(Stateful); ok {
        return s.Base()
    }
    return nil
}
```

## 6. 文件改动清单

| 文件 | 操作 | 说明 |
|------|------|------|
| `warden/state.go` | **新建** | `WardenState` + `Stateful` 接口 + 公共辅助方法（`MoveOrbit`、`FindNearest`、`BasicAttack`） |
| `warden/warden.go` | **修改** | 新增 `Level`、`GrowthOnKill`、`GrowthOnWaveClear` 字段；`BaseState()` 方法；Tick 中等级检测与升级；成长参数从配置读取 |
| `types/prince.go` | **修改** | 嵌入 `WardenState`；X/Y/Phase/Timer/Damage 等移入基座；保留冲刺/火焰痕迹字段 |
| `types/core_mech.go` | **修改** | 嵌入 `WardenState`；删除 `moveOrbit`/`performAttack`，改用 `s.MoveOrbit()`/`s.BasicAttack()` |
| `types/chain.go` | **修改** | 嵌入 `WardenState`；删除 `chainMoveOrbit`/`chainAttack`，改用基座辅助方法 |
| `types/skystrike.go` | **修改** | 嵌入 `WardenState`；删除 `skystrikeMoveOrbit`/`skystrikeAttack`，改用基座辅助方法 |
| `types/envoy.go` | **修改** | 嵌入 `WardenState`；保留附身字段；`AttackInterval=0` 使 `CanAttack()=false` |
| `render/draw_warden.go` | **修改** | 提取公共射击线逻辑到 `BaseState()`；保留各类型特有的本体形状绘制 |
| `config/warden_config.go` | **修改** | `WardenLevel` 新增 `TowerBonus` 字段 |
| `config/wardens/wardens.json` | **修改** | 移除零值填充；chain 新增 `towerBonus` |
| `config/wardens/_meta.json` | **不变** | 等级阈值保持不变 |

## 7. 迁移步骤

1. 新建 `warden/state.go`，定义 `WardenState`、`Stateful` 接口和公共辅助方法
2. 逐个类型重构为嵌入 `WardenState`（每改一个类型测试一次）
3. 将轨道运动/基础攻击逻辑合并到基座辅助方法，删除重复代码
4. 更新渲染器，用 `BaseState()` 处理公共绘制逻辑
5. 实现等级系统（读取配置 → 强度跨阈值时应用对应等级属性）
6. 实现分类型成长参数（从 JSON 配置读取替代硬编码）
7. 清理 JSON 配置（移除零值填充字段）
8. 运行全量测试，验证视觉行为一致
