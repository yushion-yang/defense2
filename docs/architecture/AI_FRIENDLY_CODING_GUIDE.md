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

## 检查清单

新增/修改代码时对照检查：

- [ ] 函数命名是否明确表达调用频率（On/Tick/Apply/Init）
- [ ] 主循环中的调用顺序是否有注释或 phase 标注
- [ ] 跨文件引用是否使用常量/枚举（无魔法字符串）
- [ ] 互斥状态是否用状态机表示（而非多个 bool）
- [ ] 配置字段是否自解释（声明式策略 > 隐式约定）
- [ ] 函数依赖是否通过参数注入（无隐式全局访问）
- [ ] Flat fields 效果是否仍在可控范围（<6 种，无叠层需求）
