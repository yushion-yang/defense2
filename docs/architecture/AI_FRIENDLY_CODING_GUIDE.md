# AI-Friendly Coding Guide

> 让代码的「执行语义」和「文本结构」一致——消除需要脑补才能理解的代码。
>
> 这些原则对 AI 和人类新成员同样有效。

## 1. 显式频率：区分事件 vs 过程

### 问题

Flat fields 架构中，"施加效果（一次性）"和"处理效果（每帧）"对同一组字段做赋值，代码形态完全一致。AI 无法从单行代码判断它的执行频率。

```go
// 这是「施加」还是「每帧处理」？看不出来
e.BleedDPS += 5.0
```

### 真实案例

毒雾塔设计为每次攻击间隔叠一层毒，但施毒逻辑写到了 tick 循环中，导致每帧叠加。AI 逐行审查每行都"正确"，无法定位 bug。

### 规则

**用命名约定区分调用频率：**

| 前缀/命名 | 含义 | 调用频率 |
|-----------|------|---------|
| `On` + 事件名 | 事件回调 | 触发时一次 |
| `Tick` / `Update` | 帧循环处理 | 每帧 |
| `Init` / `Setup` | 初始化 | 生命周期一次 |
| `Apply` + 效果名 | 施加效果 | 触发时一次 |

```go
// GOOD: 命名明确表达频率
func OnHit(source *Tower, target *Enemy)  { ... }  // 事件，一次性
func TickStatusEffects(e *Enemy, dt float64) { ... } // 过程，每帧
func ApplyPoison(e *Enemy, stacks int)    { ... }  // 施加，一次性

// BAD: 模糊命名
func HandlePoison(e *Enemy, dt float64) { ... } // 是施加还是处理？
func ProcessTower(t *Tower, enemies []*Enemy) { ... } // 是每帧还是事件触发？
```

### 效果系统：当 Flat Fields 不够用时

当出现以下信号时，从 flat fields 迁移到 Effect List：

- 同类效果需要叠层（如毒叠 3 层）
- 需要追踪效果来源（谁施加的）
- 效果种类超过 6-7 种
- 效果有复杂行为（计数器、条件触发）

```go
// Flat Fields: 简单场景（3-5 种固定效果，不叠层）
type Enemy struct {
    StunTimer  float64
    SlowTimer  float64
    SlowFactor float64
}

// Effect List: 复杂场景（叠层、来源追踪、动态增减）
type StatusEffect struct {
    Type     EffectType
    Stacks   int
    DPS      float64
    Timer    float64
    SourceID EntityID
}

type Enemy struct {
    Effects []StatusEffect
}

// 施加和处理是完全不同的函数，不可能混淆
func ApplyPoison(e *Enemy, dps float64, dur float64) { ... } // 事件
func TickEffects(e *Enemy, dt float64)               { ... } // 每帧
```

---

## 2. 显式时序：声明执行顺序依赖

### 问题

主循环中多个子系统的调用顺序有隐式依赖，人靠记忆维护，AI 认为可以随意重排。

```go
// AI 不知道这三行不能换顺序
moveTowers()
checkCollisions()
applyDamage()
```

### 规则

**方案 A（最小成本）：注释标注顺序约束**

```go
// === Update Pipeline（顺序关键，不可重排）===
// Phase 1: Movement — 更新所有实体位置
moveEnemies(dt)
moveTowers(dt)

// Phase 2: Collision — 基于新位置检测碰撞（依赖 Phase 1）
checkCollisions()

// Phase 3: Damage — 处理碰撞产生的伤害（依赖 Phase 2）
applyDamage()

// Phase 4: Cleanup — 移除死亡实体（依赖 Phase 3）
cleanup()
```

**方案 B（强保障）：Phase 枚举 + 注册制**

```go
type Phase int
const (
    PhaseMovement Phase = iota
    PhaseCollision
    PhaseDamage
    PhaseCleanup
)

type System struct {
    Phase Phase
    Name  string
    Fn    func(dt float64)
}

// 注册时声明 phase，运行时自动排序
func (g *Game) RegisterSystem(s System)
func (g *Game) Update(dt float64) {
    sort.Slice(g.systems, func(i, j int) bool {
        return g.systems[i].Phase < g.systems[j].Phase
    })
    for _, s := range g.systems {
        s.Fn(dt)
    }
}
```

---

## 3. 显式关联：消灭魔法字符串

### 问题

用字符串做跨文件匹配，重命名时编译器不报错，AI 容易漏改。

```go
// tower.go
tower.Tag = "poison"      // 字符串

// damage.go（另一个文件）
if target.Tag == "poison"  // 改了一处漏了另一处 → 静默失败
```

### 规则

**所有跨模块引用的标识符必须定义为常量/枚举：**

```go
// tags.go — 唯一真实来源
type TowerTag string
const (
    TagPoison TowerTag = "poison"
    TagFreeze TowerTag = "freeze"
    TagFire   TowerTag = "fire"
)

// 使用处
tower.Tag = TagPoison           // 编译器可追踪
if target.Tag == TagPoison { }  // 重命名自动生效
```

同样适用于：
- 事件名（bus event names）
- 配置 key
- 状态名
- 效果类型

---

## 4. 显式状态机：替代 bool 组合

### 问题

多个 bool 字段表示状态，合法组合只在开发者脑中，AI 无法判断哪些组合是 bug。

```go
type Tower struct {
    IsCharging    bool
    IsFiring      bool
    CooldownTimer float64
}

// 散落各处的条件判断
if t.IsCharging && !t.IsFiring { ... }      // 合法？
if t.IsFiring && t.CooldownTimer > 0 { ... } // bug？
```

### 规则

**有 3 个以上互斥状态时，使用显式状态机：**

```go
type TowerState int
const (
    TowerIdle TowerState = iota
    TowerCharging
    TowerFiring
    TowerCooldown
)

type Tower struct {
    State      TowerState
    StateTimer float64
}

// 状态转换集中管理，可加合法性校验
func (t *Tower) TransitionTo(next TowerState) {
    t.State = next
    t.StateTimer = 0
}
```

好处：
- AI 可以枚举所有 `TransitionTo` 调用点，验证状态转换图完整性
- 不可能出现 `IsCharging=true && IsFiring=true` 的非法状态
- switch-case 强制处理所有状态（加 `default: panic`）

---

## 5. 声明式策略：配置与逻辑的桥梁

### 问题

配置文件中的字段含义需要看实现代码才能理解，AI 无法从配置推断运行时行为。

```json
{ "poisonTower": { "layers": 3, "dpsPerLayer": 5 } }
```

`layers: 3` 是什么意思？最多叠 3 层？初始 3 层？AI 必须找到消费这个配置的代码才知道。

### 规则

**在配置或类型定义中声明行为策略：**

```go
type EffectConfig struct {
    StackMode   string  `json:"stackMode"`   // "replace" | "add" | "max"
    MaxStacks   int     `json:"maxStacks"`
    DPSPerStack float64 `json:"dpsPerStack"`
    Duration    float64 `json:"duration"`
}
```

```json
{
  "poisonTower": {
    "effect": {
      "stackMode": "add",
      "maxStacks": 3,
      "dpsPerStack": 5,
      "duration": 3.0
    }
  }
}
```

AI 看到 `stackMode: "add"` + `maxStacks: 3` 就能推断叠加逻辑，不需要追踪运行时代码。

---

## 6. 显式依赖：注入替代全局

### 问题

全局变量/闭包捕获的依赖关系在代码中不可见，AI 无法追踪数据流。

```go
var allEnemies []*Enemy  // 包级全局

func (t *Tower) Attack() {
    for _, e := range allEnemies { ... } // 依赖从哪来？
}
```

### 规则

**函数的所有依赖必须出现在参数列表中：**

```go
func (t *Tower) Attack(enemies []*Enemy) {
    for _, e := range enemies { ... } // 来源一目了然
}
```

对于需要访问多个子系统的场景，使用 context/session 结构体：

```go
type GameSession struct {
    Enemies   []*Enemy
    Towers    []*Tower
    Spawner   *Spawner
    EventBus  *EventBus
}

func (t *Tower) Attack(session *GameSession) { ... }
```

---

## 7. 对象池：生命周期与 ABA 安全

### 问题

游戏中大量实体高频创建/销毁（弹射物每秒数十个），GC 压力大。对象池复用内存，但带来一个隐蔽 bug：**ABA 问题**——实体死亡后槽位被新实体复用，持有旧引用的代码误操作新实体。

```go
// 弹射物瞄准 enemies[5] 的 "装甲兵"
// 装甲兵死亡，enemies[5] 被一只 "治疗兵" 复用
// 弹射物命中 enemies[5]——打到了治疗兵！
```

### 真实案例

回归测试 `TestRegression_ProjectileSlotReuse_ABA`：弹射物命中已死亡敌人的复用槽位，伤害被错误施加到新生成的敌人身上。

### 规则

**A. 全零重置——杜绝幽灵状态**

```go
func (p *EnemyPool) Spawn(proto string) *Enemy {
    for i := range p.enemies {
        if !p.enemies[i].Active {
            e := &p.enemies[i]
            *e = Enemy{}  // 整体清零，不靠逐字段重置
            e.Active = true
            e.ID = p.nextID  // 单调递增 ID
            p.nextID++
            // ... 初始化新实体
            return e
        }
    }
    return nil // 池满
}
```

**B. 单调 ID 校验——检测 ABA**

```go
type Projectile struct {
    Target   *Enemy
    TargetID int  // 发射时记录的 target.ID
}

// 每帧追踪目标时校验
func (p *Projectile) IsTargetValid() bool {
    return p.Target != nil &&
           p.Target.Active &&
           p.Target.ID == p.TargetID  // ID 不匹配 → 槽位已被复用
}
```

**C. 两阶段死亡——渲染与逻辑解耦**

```go
// Kill(): 开始死亡动画
// - Dying = true, HP = 0, Count-- (逻辑上已死)
// - Active 仍为 true (渲染系统仍可绘制死亡动画)
// - 索敌系统用 !Active || Dying 跳过

// FinishDying(): 动画播完
// - Active = false (槽位可被复用)

// KillImmediate(): 跳过动画 (敌人到达终点)
// - 直接 Active = false
```

| 池类型 | 分配策略 | 适用场景 |
|--------|---------|---------|
| 固定数组 + 线性扫描 | `for range` 找空槽 | Tower/Enemy（低频创建，需空间索引） |
| 环形缓冲 + 游标 | 游标前进，满时强制回收最旧 | Projectile（高频创建/销毁，FIFO 特性） |

---

## 8. 配置三阶段管线：Embed → 单例 → 运行时

### 问题

配置加载涉及多个时机（编译期嵌入 → 运行时注入 → 按需加载），AI 不清楚"这个配置值从哪里来"、"什么时候可用"。

### 规则

**三阶段，每阶段有明确边界：**

```
阶段 1: //go:embed          编译期嵌入所有 JSON/PNG/WAV 到 embed.FS
阶段 2: config.SetDataFS()  main() 启动时注入 FS 引用，解锁所有 Load* 函数
阶段 3: GlobalXxx()         首次调用时加载+缓存，后续返回缓存值
```

```go
// 阶段 1: data.go — 编译期
//go:embed config/* assets/*
var DataFS embed.FS

// 阶段 2: main.go — 运行时启动
func main() {
    config.SetDataFS(&defense2.DataFS)  // 必须在任何 Load 之前
}

// 阶段 3: balance_config.go — 按需加载
var globalBalance *BalanceConfig

func GlobalBalance() *BalanceConfig {
    if globalBalance == nil {
        LoadBalance()  // 首次加载
    }
    return globalBalance
}

func LoadBalance() {
    b := defaultBalance()       // 先填充安全默认值
    data, _ := dataFS.ReadFile("config/balance.json")
    json.Unmarshal(data, &b)    // JSON 只覆盖存在的字段
    globalBalance = &b
}
```

**关键约束：**

- `SetDataFS()` 必须在任何 `Load*()` 或 `Global*()` 之前调用
- `defaultBalance()` 提供完整的安全默认值——JSON 字段缺失不会导致零值 bug
- 全局单例用包级变量 + 函数封装，不用 `sync.Once`（单线程初始化）

### ParamOr 模式：配置层的防御性读取

```go
// warden 从 JSON 读参数，缺失时用硬编码默认值
func ParamOr(params map[string]float64, key string, fallback float64) float64 {
    if v, ok := params[key]; ok {
        return v
    }
    return fallback
}

// 使用处 — 每个参数都有明确的 fallback
cooldown := ParamOr(w.Params, "cooldown", 2.0)
damage   := ParamOr(w.Params, "damage", 50.0)
```

---

## 9. 分层通信：事件频率决定通信机制

### 问题

游戏中不同子系统间的通信频率差异巨大：击杀成就每局几十次，伤害计算每帧上百次。用同一种机制处理所有通信要么性能差，要么耦合紧。

### 规则

**按频率选择通信机制，不要一刀切：**

| 通信机制 | 频率 | 耦合度 | 本项目实例 |
|----------|------|--------|-----------|
| Event Bus | 低频（每波/每击杀） | 最低 | `event.Bus`: TowerBuilt / EnemyKilled / WaveCleared |
| TickCallbacks 结构体 | 每帧 | 中等 | `pipeline.TickCallbacks`: OnDamageText / OnKill / OnCC |
| Pool Hooks | 生命周期 | 中等 | `EnemyPool.OnSplit` / `TowerPool.RemoveHook` |
| 直接函数调用 | 热路径 | 最高 | `combat.ProcessDamage()` 直接操作 Enemy 字段 |

```go
// LOW-FREQ: Event Bus — 跨场景，有 payload 结构体
bus.Emit(event.EvtEnemyKilled, event.EnemyKilledPayload{
    IsBoss: e.IsBoss, KillerID: tower.ID, GoldValue: e.GoldValue,
})

// PER-FRAME: TickCallbacks — 管线系统触发，场景层填充闭包
type TickCallbacks struct {
    OnDamageText func(x, y float64, text string, clr color.RGBA)
    OnKill       func(e *Enemy, tower *Tower)
    OnCC         func(x, y float64, ccType string)
    // ... 12+ 回调
}

// LIFECYCLE: Pool Hooks — 避免循环依赖
pool.RemoveHook = func(key string) {
    renderCache.Invalidate(key)  // 场景层注入，pool 不 import render
}
```

**回调安全规则：**

```go
// 所有回调必须 nil 检查（场景层可能未填充）
if ctx.CB.OnWaveStart != nil {
    ctx.CB.OnWaveStart(waveNum)
}
```

---

## 10. init() 注册与空白导入

### 问题

Go 的 `init()` + 空白导入 (`_ "pkg"`) 模式实现了"声明即注册"，但 AI 容易遗漏：

1. 新增的 warden/ability 类型没有被空白导入 → 注册表为空 → 运行时静默失败
2. `init()` 依赖其他包的初始化顺序 → 隐式耦合

### 本项目的三种注册策略

```
策略 A: 同包 init() — 自包含，无需导入
  ├── combat/attack.go init()    注册 5 种攻击方式
  └── gamemode/mode.go init()    注册 7 种游戏模式

策略 B: 跨包 init() + 空白导入 — 需要导入方触发
  ├── warden/types/envoy.go init()  → 自注册到 warden.RegisterBehavior()
  ├── warden/types/chain.go init()  → 同上
  └── 必须在 stage.go 空白导入:  _ "defense2/internal/core/warden/types"

策略 C: 显式 Load → Register — 有初始化顺序依赖
  └── abilities.InitConfigAbilities()  先 Load JSON，再 Register
      （依赖 config.SetDataFS() 已执行，不能放 init()）
```

**空白导入的唯一入口（stage.go）：**

```go
import (
    _ "defense2/internal/core/tower/abilities" // init() 注册所有能力
    _ "defense2/internal/core/warden/types"    // init() 注册所有战灵类型
)
```

**防遗漏措施——契约测试：**

```go
// tests/contracts/system_contracts_test.go
func TestAllWardenTypesRegistered(t *testing.T) {
    // init() 触发需要空白导入
    expected := []string{"prince", "skystrike", "envoy", "chain", "core_mech"}
    for _, name := range expected {
        if warden.GetBehavior(name) == nil {
            t.Errorf("warden type %q not registered — missing blank import?", name)
        }
    }
}
```

### 新增 ability/warden 类型的完整检查清单

1. 在对应 `types/` 目录新增 `.go` 文件，实现接口 + `init()` 自注册
2. 在 `stage.go` 确认已有空白导入（或手动添加）
3. 在契约测试中添加新类型的名称
4. `make test` 通过

---

## 11. 对象池空间索引：O(1) 查找

### 问题

"点击了 (row=3, col=5) 的格子上有没有塔？"——遍历 64 个塔挨个比较坐标是 O(n)，AI 可能写出这样的低效代码。

### 规则

**池附带空间索引时，查找用索引不用遍历：**

```go
type TowerPool struct {
    towers [MaxTowers]Tower
    grid   [gridMaxRows][gridMaxCols]*Tower  // 空间索引
}

// O(1) 查找 — 用这个
func (p *TowerPool) At(row, col int) *Tower {
    return p.grid[row][col]
}

// O(n) 遍历 — 只在需要"全部塔"时使用
func (p *TowerPool) Each(fn func(*Tower)) {
    for i := range p.towers {
        if p.towers[i].Active { fn(&p.towers[i]) }
    }
}
```

**Spawn/Remove 必须同步更新索引：**

```go
func (p *TowerPool) Spawn(row, col int) *Tower {
    t := p.findFreeSlot()
    t.Row, t.Col = row, col
    p.grid[row][col] = t        // 同步更新索引
    return t
}

func (p *TowerPool) Remove(t *Tower) {
    p.grid[t.Row][t.Col] = nil  // 同步清除索引
    t.Active = false
}
```

---

## 12. 六层测试防线

### 问题

AI 容易只写 happy-path 单元测试，遗漏配置映射、API 规范、回归 bug 等维度。

### 规则

**本项目六类测试，各有明确职责：**

```
tests/
├── contracts/    配置→代码的结构性断言（JSON 字段必须有对应代码）
├── core/         单元测试（单个函数/模块的行为）
├── regression/   Bug 回归测试（每个测试对应一个已修复 bug）
├── integration/  跨系统集成测试（多个模块协作）
├── fuzz/         模糊测试（随机输入发现边界）
├── lint/         源码 lint（禁止直接调用 ebiten API）
```

**A. 回归测试：必须有 BUG 注释**

```go
// BUG: SlowFactor=0 caused enemy speed to drop to 0 (full stop).
// Fix: combat.ApplySlow clamps factor to MinSpeedRatio (0.2).
func TestRegression_CC_SlowMinSpeedClamp(t *testing.T) {
    s := sim.New().
        WithStraightPath(500).
        WithEnemy("normal", 100, 100).
        Build()
    s.ApplySlow(0, 0.0, 3.0)  // factor=0 → should clamp
    s.RunTicks(60)
    s.AssertEnemySpeed(t, 0, ">", 0)  // 不能完全停止
}
```

**B. 契约测试：验证配置完整性**

```go
func TestEveryAbilityInJSON_HasCodeHandler(t *testing.T) {
    table := config.GlobalAbilityTable()
    for name := range table {
        if tower.GetAbility(name) == nil {
            t.Errorf("ability %q in JSON but no code handler", name)
        }
    }
}
```

**C. Lint 测试：禁止直接调用底层 API**

```go
func TestNoDirectEbitenCalls(t *testing.T) {
    // 遍历所有 .go 文件（排除 draw/ 包本身）
    // 正则匹配 ebiten.CursorPosition / vector.StrokeLine 等
    // 强制所有渲染通过 HiDPI 感知的 draw.* 包装函数
}
```

**D. Headless Sim 框架：无 Ebitengine 依赖的集成测试**

```go
s := sim.New().
    WithStraightPath(500).
    WithEnemy("tank", 500, 80).
    WithTower("basic", 3, 2, 50).
    Build()

s.RunTicks(300)
s.AssertEnemyHP(t, 0, "<", 400)   // 确认受到伤害
s.AssertEnemyDead(t, 0)            // 确认最终死亡
```

---

## 13. TickCallbacks：纯逻辑管线的副作用出口

### 问题

管线系统（pipeline/）需要触发音效、VFX、飘字等表现层效果，但不能 import render/audio 包——否则形成循环依赖，且逻辑不可测试。

### 规则

**管线系统只通过 TickCallbacks 结构体"喊话"，不直接操作表现层：**

```go
// pipeline/orchestrator.go — 纯数据结构，零 render/audio import
type TickCallbacks struct {
    OnDamageText   func(x, y float64, text string, clr color.RGBA)
    OnKill         func(e *Enemy, tower *Tower)
    OnTowerFire    func(tower *Tower, target *Enemy)
    OnProjectileHit func(x, y float64, style string)
    OnCC           func(x, y float64, ccType string)
    OnWaveStart    func(waveNum int)
    OnWardenFire   func(w *Warden, x, y float64)
    // ...
}

// pipeline 内部 — 只管逻辑，触发回调
if enemy.HP <= 0 {
    if ctx.CB.OnKill != nil {
        ctx.CB.OnKill(enemy, tower)
    }
}

// scene/stage.go — 场景层填充回调闭包
ctx.CB.OnKill = func(e *Enemy, t *Tower) {
    audioMgr.PlaySFX("enemy-death")
    vfxMgr.SpawnDeathEffect(e.X, e.Y)
    floatingText.Add(e.X, e.Y, fmt.Sprintf("+%d", e.GoldValue))
}
```

**好处：**

- pipeline 包可 headless 测试（sim 框架中 callbacks 设为 nil 或 mock）
- 表现层替换不影响逻辑（如切换 audio 后端、禁用 VFX）
- AI 看到 `ctx.CB.OnKill` 就知道"这里产生副作用"，不用追踪到 render 包

---

## 检查清单

新增/修改代码时对照检查：

**命名与结构**
- [ ] 函数命名是否明确表达调用频率（On/Tick/Apply/Init）
- [ ] 主循环中的调用顺序是否有注释或 phase 标注
- [ ] 跨文件引用是否使用常量/枚举（无魔法字符串）
- [ ] 互斥状态是否用状态机表示（而非多个 bool）
- [ ] 配置字段是否自解释（声明式策略 > 隐式约定）
- [ ] 函数依赖是否通过参数注入（无隐式全局访问）

**对象池与生命周期**
- [ ] 池对象复用时是否全零重置（`*e = Enemy{}`）
- [ ] 跨帧持有的引用是否有 ID 校验（防 ABA 问题）
- [ ] 两阶段死亡：逻辑死亡（Dying）与物理回收（Active=false）是否分离
- [ ] 空间索引是否在 Spawn/Remove 时同步更新

**配置与注册**
- [ ] 新增的 ability/warden 类型是否已空白导入
- [ ] 契约测试是否覆盖新增类型
- [ ] 配置读取是否有 fallback 默认值（ParamOr 模式）
- [ ] `SetDataFS()` 是否在所有 `Load*()` 之前

**通信与副作用**
- [ ] 副作用是否通过 TickCallbacks 触发（pipeline 不 import render/audio）
- [ ] 回调是否有 nil 检查
- [ ] 低频事件用 Event Bus，高频事件用 TickCallbacks 或直接调用

**测试**
- [ ] 回归测试是否有 `// BUG:` + `// Fix:` 注释
- [ ] 新增系统是否有对应的契约测试
- [ ] 是否使用 sim 框架做 headless 集成测试（无 Ebitengine 依赖）
