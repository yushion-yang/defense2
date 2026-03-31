# 塔防引擎抽象思考

> 2026-03-31 | 从 defense2 经验出发，反思专业引擎 vs 自研引擎的边界

## 1. 问题：我们在操心哪些"本不该操心"的事？

defense2 当前 **199 个 Go 文件、31,796 行代码**，分布在 42 个包中。stage.go 一个文件 2,707 行。

把这些代码按"谁该负责"分类：

### 1.1 引擎层该做的（我们手写了）

| 模块 | 行数 | 专业引擎中的对应 | 我们做了什么 |
|------|------|------------------|-------------|
| `render/draw/` | 607 | Canvas/Shape API | 手写 Line/Rect/Circle/Glow/Diamond/DashedLine |
| `render/postprocess/` | 741 | Shader Graph / Post-FX Stack | 手写 bloom/blur/vignette/color_grade/radial_blur + ping-pong 管线 |
| `render/particle/` | 356 | Particle System Editor | 手写 ring buffer + DrawTriangles 批渲染 |
| `render/anim/` | 338 | Animation Controller | 手写帧动画加载器 + 状态切换 |
| `render/sprite/` | 73 | Sprite Renderer | 手写 PNG 缓存 + 缩放绘制 |
| `render/ui/` | 789 | UI Framework (Canvas/Flex) | 手写按钮/面板/属性行/文本布局 |
| `render/theme/` | 517 | Theme/Skin System | 手写设计系统(颜色/间距/字体) |
| `audio/` | 320 | Audio Manager | 手写加载/播放/节流/音量 |
| `input/` | 222 | Input System | 手写手势识别/统一触摸鼠标 |
| `scene/` (框架部分) | ~500 | Scene Manager + Transition | 手写淡入淡出/状态机/场景切换 |
| `core/event/` | 397 | Event/Signal System | 手写 pub/sub 总线 |
| `core/physics/` | 165 | Physics / Spatial Query | 手写线段-圆碰撞/范围查询 |
| `core/debug/` | 227 | Debug Overlay / Profiler | 手写 perf tracker + overlay |

**合计约 5,252 行**（总代码的 16.5%）属于"通用引擎基础设施"。

### 1.2 塔防框架层该做的（我们也手写了）

| 模块 | 行数 | 通用塔防概念 | 我们做了什么 |
|------|------|------------|-------------|
| `core/enemy/` | 1,528 | 敌人实体 + 对象池 + 路径跟随 | 手写池/路径/原型/buff 槽 |
| `core/tower/` | 1,028 | 塔实体 + 放置 + 升级 + 能力 | 手写塔/技能树/强化 |
| `core/projectile/` | 475 | 弹射物系统 | 手写追踪/直飞/穿刺/散射 |
| `core/combat/` | 1,070 | 伤害计算 + 攻击方式 | 手写 8 步管线 + 8 种攻击 |
| `core/buff/` | 506 | Buff/Debuff 系统 | 手写 6 种堆叠 + 回调 |
| `core/skill/` | 1,557 | 主动/被动技能 | 手写框架 + 7 种技能 |
| `core/warden/` | 510+ | 召唤物/英雄单位 | 手写状态机 + 5 种战灵 |
| `core/strength/` | 229 | 属性/强化系统 | 手写三层属性 + Union-Find 链 |
| `core/gamemode/` | 959 | 游戏模式 + 波次 | 手写 6 种模式 + 波次组合 |
| `core/economy/` | 49 | 经济系统 | 金币/奖励 |
| `core/pipeline/` | 603 | Tick 调度 | 手写各子系统 tick 编排 |
| `render/hud/` | 3,167 | 塔防 UI 面板 | 手写信息面板/建造/波次/小地图 |
| `config/` | 804 | 数据驱动配置 | 手写 JSON 加载器 |

**合计约 12,485 行**（总代码的 39.3%）属于"塔防品类通用逻辑"。

### 1.3 真正的游戏特色（我们的核心价值）

| 模块 | 描述 |
|------|------|
| 具体的塔类型设计（属性/攻击/技能树） | JSON 配置 + 少量胶水代码 |
| 具体的敌人原型设计 | JSON 配置 |
| 具体的地图/关卡设计 | JSON 配置 |
| 具体的战灵类型实现 | ~300 行/种 |
| 波次编排策略 | ~200 行 |
| 难度曲线调参 | JSON 配置 |
| 美术风格和主题色 | theme 包 |
| 音效设计 | WAV 文件 + 映射表 |
| 游戏特有的 "feel"（juice/连杀/完美波次） | ~500 行 |

**核心创意部分只占代码的很小比例，大量时间花在了基础设施上。**

---

## 2. 专业引擎对比

### 2.1 Unity / Godot 做塔防会怎样

```
你不需要操心的事（引擎内置）：
├── 渲染管线（Sprite/Shader/Post-FX/Particle → 编辑器拖拽）
├── UI 系统（Canvas/Flex 布局/按钮/面板 → 编辑器可视化）
├── 动画系统（Animator Controller → 状态机编辑器）
├── 音频系统（AudioSource/Mixer → 编辑器）
├── 输入系统（Input System 包 → 配置映射）
├── 场景管理（SceneManager + 过渡 → 内置）
├── 物理/碰撞（Collider2D/Raycast → 内置）
├── 序列化/配置（ScriptableObject → 编辑器）
├── 调试工具（Profiler/Frame Debugger → 内置）
└── 多平台构建（一键导出 → 内置）

你需要写的（游戏逻辑）：
├── 敌人寻路（NavMesh 或路点系统）← ~100 行
├── 塔放置逻辑 ← ~200 行
├── 战斗计算 ← ~500 行
├── 波次管理 ← ~200 行
├── 经济系统 ← ~100 行
├── UI 数据绑定 ← ~300 行
└── 游戏特色系统 ← 取决于创意

估算总代码量：2,000-5,000 行 vs 我们的 31,796 行
```

### 2.2 为什么 Ebitengine 下需要写这么多

Ebitengine 定位是**极简 2D 游戏库**，不是引擎：

```
Ebitengine 提供的：
├── 窗口/上下文管理
├── Image 绘制（DrawImage + GeoM + ColorScale）
├── Shader（Kage 语言）
├── 音频解码/播放
├── 输入读取（鼠标/键盘/触摸/游戏手柄）
└── 基本数学（运行时调度 Update/Draw/Layout）

Ebitengine 不提供的（全部自己来）：
├── UI 框架
├── 粒子系统
├── 动画控制器
├── 场景管理
├── 碰撞系统
├── 资源管线（热重载/打包/缓存策略）
├── 后处理管线
├── 对象池
├── ECS / 组件系统
├── 数据驱动框架
└── 调试/性能分析工具
```

**我们选择 Ebitengine 本质上是选择了"从头搭一切"。好处是极致可控，代价是工程量巨大。**

---

## 3. 我们能抽象出什么？—— "defense-engine" 构想

### 3.1 分层架构

```
┌──────────────────────────────────────────────┐
│  Layer 4: Game Content（你的游戏）             │ ← JSON 配置 + 少量胶水
│  - 具体塔/敌人/地图/战灵/技能设计              │
│  - 关卡编排、难度曲线、美术风格                  │
├──────────────────────────────────────────────┤
│  Layer 3: TD Framework（塔防框架）             │ ← 可复用的塔防品类库
│  - Entity (Tower/Enemy/Projectile/Summon)    │
│  - Combat (DamagePipeline/AttackStyles/Buff) │
│  - Wave (Spawner/Composer/Scheduler)         │
│  - Economy (Gold/Reward/Shop)                │
│  - GameMode (Campaign/Endless/Challenge)     │
│  - Map (Grid/Path/Placement)                 │
├──────────────────────────────────────────────┤
│  Layer 2: Game Engine（通用 2D 引擎层）        │ ← 可复用到任何 2D 游戏
│  - Renderer (Sprite/Shape/Text/PostFX)       │
│  - UI (Panel/Button/Label/Layout)            │
│  - Audio (Manager/SFX/BGM/Spatial)           │
│  - Animation (FrameAnim/Tween/Timeline)      │
│  - Particle (Emitter/Pool/Preset)            │
│  - Input (Gesture/Unified Touch+Mouse)       │
│  - Scene (StateMachine/Transition/Loading)   │
│  - Physics (Collision/SpatialHash/Query)     │
│  - Event (Bus/TypedEvent/Priority)           │
│  - ECS-lite (Entity/Component/System)        │
│  - Debug (PerfOverlay/Inspector/Console)     │
│  - Config (Loader/Validator/HotReload)       │
├──────────────────────────────────────────────┤
│  Layer 1: Platform（底层库）                   │ ← Ebitengine / SDL / WebGPU
│  - Window/Context, DrawImage, Shader         │
│  - Audio decode, Input polling               │
└──────────────────────────────────────────────┘
```

### 3.2 Layer 2：通用 2D 引擎层（~5,000 行 → 独立 module）

从 defense2 可直接提取的：

```go
// github.com/user/oak2d (或任何名字)

oak2d/
├── render/
│   ├── sprite.go      // Sprite/SpriteScaled/SpriteRotated（从 draw/ 提取）
│   ├── shape.go       // Line/Rect/Circle/Glow/Diamond（从 draw/ 提取）
│   ├── text.go        // DrawText/Centered/Right/Measure（从 font.go 提取）
│   ├── camera.go      // 视口/缩放/震动（新建，替代 stage.go 中散落的逻辑）
│   └── postfx/        // Pipeline/Bloom/Blur/Vignette（从 postprocess/ 提取）
├── ui/
│   ├── panel.go       // 面板容器 + 自动布局
│   ├── button.go      // 按钮 + 状态（normal/hover/pressed/disabled）
│   ├── label.go       // 文本标签
│   ├── layout.go      // Row/Column/Grid 布局
│   └── theme.go       // 设计系统（颜色/间距/字体 token）
├── anim/
│   ├── frame.go       // 帧动画（从 anim/ 提取）
│   └── tween.go       // 缓动动画（新建，替代各处 hardcoded lerp）
├── particle/
│   ├── pool.go        // 粒子池（从 particle/ 提取）
│   ├── emitter.go     // 发射器
│   └── preset.go      // 预设（burst/stream/ambient）
├── audio/
│   ├── manager.go     // 音频管理器（从 audio/ 提取）
│   └── throttle.go    // SFX 节流
├── input/
│   ├── gesture.go     // 手势识别（从 input/ 提取）
│   └── unified.go     // 统一鼠标/触摸
├── scene/
│   ├── manager.go     // 场景状态机
│   └── transition.go  // 淡入淡出/滑动/自定义
├── physics/
│   ├── collision.go   // 2D 碰撞检测
│   └── spatial.go     // 空间哈希（新建，替代暴力遍历）
├── event/
│   └── bus.go         // 类型安全事件总线
├── debug/
│   ├── perf.go        // 性能追踪
│   └── overlay.go     // 调试覆盖层
└── config/
    ├── loader.go      // JSON/TOML 配置加载
    └── validator.go   // 配置校验
```

**关键设计原则：**
- 零 Ebitengine 导出类型泄漏（通过接口隔离，未来可换底层）
- 每个子包独立可用（不强制全量导入）
- 配置驱动 > 代码驱动

### 3.3 Layer 3：塔防框架层（~8,000 行 → 独立 module）

```go
// github.com/user/tdcore

tdcore/
├── entity/
│   ├── tower.go        // Tower 实体 + 生命周期
│   ├── enemy.go        // Enemy 实体 + 路径跟随 + 对象池
│   ├── projectile.go   // 弹射物 + 追踪/直飞
│   └── summon.go       // 召唤物/战灵 基础
├── combat/
│   ├── pipeline.go     // 伤害管线（可配置步骤链）
│   ├── attack.go       // AttackHandler 接口 + 注册表
│   ├── buff.go         // Buff 框架（堆叠/回调/模板）
│   ├── cc.go           // 控制效果（减速/眩晕/定身）
│   └── damage_type.go  // 伤害类型 + 穿透矩阵
├── wave/
│   ├── spawner.go      // 波次生成器
│   ├── composer.go     // 波次组合（阶段/Boss/混合）
│   └── scheduler.go    // 波间调度（倒计时/手动/自动）
├── map/
│   ├── grid.go         // 网格 + 放置规则
│   ├── path.go         // 路径定义 + 多路径
│   └── placement.go    // 建造验证
├── economy/
│   ├── wallet.go       // 金币/资源
│   └── reward.go       // 击杀/波次奖励
├── mode/
│   ├── mode.go         // GameMode 接口
│   ├── campaign.go     // 战役模式
│   ├── endless.go      // 无尽模式
│   └── challenge.go    // 挑战模式
├── skill/
│   ├── skill.go        // 技能接口 + 注册表
│   └── passive.go      // 被动技能框架
├── strength/
│   ├── attribute.go    // 属性系统（base/temp/debuff）
│   └── link.go         // 链网络（Union-Find）
├── tick/
│   ├── orchestrator.go // Tick 编排器（定义子系统执行顺序）
│   └── context.go      // TickContext（各系统共享数据）
├── hud/
│   ├── tower_info.go   // 塔信息面板（数据层，不含渲染）
│   ├── build_menu.go   // 建造菜单（数据层）
│   ├── wave_bar.go     // 波次状态栏（数据层）
│   └── minimap.go      // 小地图（数据层）
└── config/
    ├── tower.go        // 塔配置 schema
    ├── enemy.go        // 敌人配置 schema
    ├── map.go          // 地图配置 schema
    └── validate.go     // 配置一致性校验
```

**关键设计原则：**
- **纯逻辑，零渲染依赖**：HUD 只输出数据（"显示什么"），不调用任何 draw
- **注册表模式**：攻击方式/技能/模式/Buff 全部插件化
- **伤害管线可配置**：步骤链可增删重排（而非 hardcoded 8 步）
- **配置 schema 强类型**：编译时校验，不靠运行时 panic

### 3.4 Layer 4：你的游戏（~3,000 行 + 大量 JSON）

```
defense2-game/
├── content/
│   ├── towers/          # JSON: 每种塔的属性/攻击方式/技能树
│   ├── enemies/         # JSON: 每种敌人的原型/属性
│   ├── maps/            # JSON: 地图网格/路径/放置点
│   ├── wardens/         # JSON: 战灵属性/技能
│   ├── waves/           # JSON: 波次编排规则
│   ├── difficulties/    # JSON: 难度系数
│   └── audio/           # WAV/OGG 文件
├── theme/               # 主题色/字体/图标
├── assets/              # PNG 精灵/动画帧
├── warden_impl/         # 5 种战灵的特化行为（~1,500 行）
├── juice/               # 连杀/完美波次/入场动画等 "feel"（~500 行）
├── main.go              # 启动入口（~100 行）
└── config.go            # 注册自定义攻击方式/技能/模式（~200 行）
```

**目标：游戏开发者只写"什么让这个游戏独特"的部分。**

---

## 4. 这值得做吗？—— 成本收益分析

### 4.1 收益

| 收益 | 说明 |
|------|------|
| **第二款塔防零成本启动** | 换套 JSON + 主题色就是新游戏 |
| **分层测试** | 引擎层/框架层/游戏层各自独立测试，不再需要 GUI |
| **降低 stage.go 复杂度** | 2,707 行 → ~500 行胶水 |
| **团队协作** | 策划改 JSON，程序改框架，互不干扰 |
| **社区价值** | Go 生态没有成熟的塔防框架 |
| **可换底层** | 哪天 Ebitengine 不维护了，Layer 1 换掉即可 |

### 4.2 成本

| 成本 | 说明 |
|------|------|
| **重构工作量** | 估算 2-3 周全职工作（拆分+接口设计+测试） |
| **过度抽象风险** | 只做过 1 款塔防，抽象可能"猜错" |
| **性能开销** | 接口/注册表有间接调用成本（但 Go 内联很激进，影响微乎其微） |
| **维护两个 module** | 引擎层和框架层需要独立版本管理 |

### 4.3 我的建议：渐进式提取

**不要一步到位搞引擎。** 按以下顺序渐进提取：

```
Phase 0（现在就能做，0 成本）:
  → 在 internal/ 内部理清层级边界
  → 确保 core/ 不 import render/
  → 确保 render/ 不 import core/（通过接口）

Phase 1（做第二个游戏时）:
  → 提取 Layer 2 (oak2d) 为独立 Go module
  → defense2 和新游戏共用

Phase 2（做第二个塔防时）:
  → 提取 Layer 3 (tdcore) 为独立 Go module
  → 两款塔防共用

Phase 3（社区验证后）:
  → 开源 + 文档 + 示例
  → 根据反馈调整 API
```

**"第二次使用时才提取"是工程界的黄金法则。** 过早抽象比不抽象更危险。

---

## 5. 当前 defense2 可以立即做的改善

不动大架构，但可以减少 stage.go 的认知负担：

### 5.1 Tick Orchestrator 提取

stage.go 的 `updatePlaying()` 有 ~400 行调度代码，可提取为：

```go
// internal/core/pipeline/orchestrator.go
type Orchestrator struct {
    systems []System // 有序的子系统列表
}

func (o *Orchestrator) Tick(ctx *TickContext) {
    for _, sys := range o.systems {
        sys.Tick(ctx)
    }
}
```

### 5.2 HUD 数据/渲染分离

当前 `render/hud/` 既算数据又画 UI。分成：

```
core/hud/       → 纯数据（"面板应该显示什么"）
render/hud/     → 纯渲染（"怎么画到屏幕上"）
```

### 5.3 交互状态机提取

stage.go 的 `interactMode` 状态机（7 种模式 + 转换规则）可提取为独立包：

```
core/interact/  → 状态定义 + 转换规则 + 输入处理
```

---

## 6. 结论

> **我们在 Ebitengine 上从零搭建了一个 "隐式引擎"——只是没有把它命名、分层、复用。**

专业引擎（Unity/Godot）的优势不是"代码更好"，而是**分层更清晰 + 工具链更完整**。我们的代码质量不差，但全部揉在一个 module 的 internal/ 下，失去了复用性。

最务实的路径：
1. **现在**：理清 core/ 和 render/ 的单向依赖，减少 stage.go 体积
2. **下一个项目**：提取通用 2D 引擎层
3. **下一个塔防**：提取塔防框架层
4. **别急着造引擎**：先造游戏，引擎是游戏的副产品
