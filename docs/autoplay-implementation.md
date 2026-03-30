# AutoPlay 自动对局系统 — 实施文档

**日期**: 2026-03-31
**基于设计**: `docs/plans/2026-03-30-autoplay-design.md`

## 一、目标

构建自动化游戏运行系统，能够：
1. 系统性覆盖所有游戏内容和代码路径
2. 每局收集结构化 JSON 数据 + 关键帧截图
3. 检测平衡性问题和运行时异常
4. 输出可供 AI 分析的制品

## 二、实施概览

### 新增文件（17 个）

| 文件 | 行数 | 用途 |
|------|------|------|
| `internal/autoplay/strategy.go` | ~95 | 核心类型（GameState/Action/Strategy 接口） |
| `internal/autoplay/anomaly.go` | ~170 | 运行时异常检测器（10 种异常） |
| `internal/autoplay/screenshotter.go` | ~90 | 关键帧截图管理器 |
| `internal/autoplay/recorder.go` | ~210 | JSON 数据收集器 |
| `internal/autoplay/controller.go` | ~200 | AutoPlay 控制器（编排核心） |
| `internal/autoplay/strategy_random.go` | ~100 | 随机模糊策略 |
| `internal/autoplay/strategy_greedy.go` | ~120 | 贪心启发策略 |
| `internal/autoplay/strategy_focus.go` | ~90 | 单塔极限策略 |
| `internal/autoplay/strategy_scenario.go` | ~200 | 脚本化场景策略 |
| `internal/autoplay/coverage.go` | ~200 | 覆盖矩阵 + 测试计划生成 |
| `internal/autoplay/report.go` | ~160 | 汇总报告生成器 |
| `internal/autoplay/strategy_test.go` | ~130 | 策略单元测试 |
| `internal/autoplay/anomaly_test.go` | ~120 | 异常检测测试 |
| `internal/autoplay/coverage_test.go` | ~60 | 覆盖矩阵测试 |
| `internal/autoplay/recorder_test.go` | ~80 | 录制器测试 |
| `internal/core/gamemode/autoplay.go` | ~65 | AutoPlay 游戏模式 |
| `internal/core/gamemode/autoplay_test.go` | ~55 | 游戏模式测试 |

### 修改文件（3 个）

| 文件 | 改动 | 用途 |
|------|------|------|
| `internal/scene/autoplay_types.go` | 新建 ~95 行 | AutoPlayer 接口 + 快照/操作类型 |
| `internal/scene/stage.go` | 新增 ~130 行 | AutoPlayer 集成（钩子+快照+执行） |
| `internal/core/gamemode/register.go` | +1 行 | 注册 AutoPlay 模式 |
| `cmd/autoplay/main.go` | 新建 ~95 行 | CLI 入口 |

## 三、架构决策

### 3.1 循环导入避免

关键挑战：`scene` 包（StageScene）和 `autoplay` 包（Controller）互相依赖。

**解决方案**：双类型映射模式
```
scene.AutoPlayer     (接口，定义在 scene 包)
     ↑ 实现
autoplay.Controller  (在 autoplay 包)
     ↓ 使用
autoplay.Strategy    (纯逻辑，无 scene 依赖)
```

- `scene.AutoPlaySnapshot` / `scene.AutoPlayAction` — scene 包内的数据类型
- `autoplay.GameState` / `autoplay.Action` — autoplay 包内的对应类型
- `Controller.snapshotToGameState()` 负责映射

### 3.2 StageScene 最小侵入

StageScene 是 ~2400 行的单体私有结构体。所有游戏操作（建塔/卖塔/升级）都是私有方法。

**集成方式**：
1. 新增 `autoPlayer AutoPlayer` 字段
2. `SetAutoPlayer(ap)` 公共方法注入
3. `buildAutoPlaySnapshot()` 构建只读快照
4. `executeAutoPlayAction(a)` 分发到现有私有方法
5. `runAutoPlayFrame()` 在 `updatePlaying()` 末尾调用

**特殊处理**：
- `modeEvent` 和 `modeWardenSelect` 状态下 `updatePlaying()` 不执行，需要在 `Update()` 的 switch 分支中单独处理
- `stateVictory/stateDefeat` 时返回 `ebiten.Termination` 终止游戏循环

### 3.3 游戏模式

基于 `testmode.go` 模式创建 `autoplay.go`：
- 永不失败（`CheckDefeat` 返回 false）
- 自动开波（`ShouldAutoStart` 返回 true）
- 短间歇（2 秒）
- 启用事件（测试事件代码路径）

### 3.4 截图实现

使用最小化窗口模式：
- Ebitengine 正常渲染到窗口
- `Draw()` 完成后通过 `screen.ReadPixels()` 读取像素
- 异步 PNG 编码保存，避免阻塞渲染

### 3.5 测试环境

Ebitengine 在无 GUI 的环境下初始化 GLFW 会 panic。
- `controller.go` 使用 `//go:build !unittest` 标签
- 纯逻辑测试用 `go test -tags unittest` 运行
- 游戏模式测试（`internal/core/gamemode/`）不依赖 Ebitengine，直接运行

## 四、策略实现

### 4.1 RandomStrategy（随机模糊）
- 每帧 2% 概率建塔、0.5% 升级、0.1% 卖塔
- 随机选择塔类型和位置
- 用于广覆盖和发现 panic

### 4.2 GreedyStrategy（贪心启发）
- 三阶段：早期建塔 → 中期升级 → 后期全力升级
- 按性价比排序塔类型（`damage × range / cost`）
- 优先靠近地图中心的位置

### 4.3 FocusStrategy（单塔极限）
- 只建一种塔，填满所有位置后轮询升级
- 构造函数接受 `towerKey` 参数
- 8 种塔 × 每种一次 = 8 个测试用例

### 4.4 ScenarioStrategy（脚本化场景）
- 步骤式 FSM：`WaitUntil` 条件 + `Actions` 操作
- 4 个预定义场景：
  - `build-flow` — 建塔流程
  - `tower-lifecycle` — 建 → 升级 → 卖
  - `zero-gold-build` — 零金币边界
  - `rapid-actions` — 同帧多操作压力测试

## 五、覆盖矩阵

### 测试计划生成

| 维度 | 用例数 | 方法 |
|------|--------|------|
| Pairwise 组合（地图×难度×战灵×策略） | ~32 | 确保每对维度组合至少出现一次 |
| 单塔极限 | 8 | 每种塔一次 FocusStrategy |
| 敌人定向 | 13 | 每种原型一次 |
| 交互场景 | 4 | 预定义脚本 |
| 边界测试 | 4 | 极端条件 |
| **合计** | **~61** | |

### 覆盖追踪

每局记录：
- 使用的塔类型
- 遇到的敌人原型
- 选择的事件
- 触发的能力

汇总报告输出覆盖缺口。

## 六、异常检测

| 异常类型 | 条件 | 严重程度 |
|----------|------|----------|
| `enemy_stuck` | 位置 60 帧不变，速度非零 | HIGH |
| `gold_negative` | 金币 < 0 | CRITICAL |
| `gold_spike` | 单帧金币增 >500 | HIGH |
| `lives_drop` | 单帧生命减 >5 | MEDIUM |
| `fps_drop` | 连续 10 帧 >50ms | MEDIUM |
| `panic_recovered` | panic 恢复 | CRITICAL |
| `dead_enemy_walking` | HP≤0 但 Active 且非 Dying | CRITICAL |

每个异常触发截图和 JSON 记录。

## 七、运行方式

```bash
# 单局测试
go run cmd/autoplay/main.go \
  --runs 1 --strategies random \
  --map map_01 --difficulty normal \
  --output ./autoplay-results/

# 完整覆盖扫描
go run cmd/autoplay/main.go --sweep --output ./autoplay-results/

# 单个场景
go run cmd/autoplay/main.go --scenario build-flow --map map_01

# 多策略
go run cmd/autoplay/main.go --runs 3 --strategies random,greedy,focus
```

### 输出目录结构

```
autoplay-results/
  ├── 2026-03-31_random_map_01_normal_000/
  │   ├── report.json
  │   ├── start.png
  │   ├── wave_5.png
  │   └── result.png
  ├── 2026-03-31_greedy_map_01_normal_000/
  │   └── ...
  └── coverage_summary.json
```

## 八、测试结果

### 单元测试

```
go test -tags unittest ./internal/autoplay/... -count=1  → PASS (26 tests)
go test ./internal/core/gamemode/... -count=1            → PASS (8 tests)
go build ./...                                           → PASS
go vet ./...                                             → PASS (无新增警告)
```

### 已知限制

1. **Ebitengine GLFW 限制**：`internal/autoplay/` 的测试因 `controller.go` 导入 `scene` 包触发 GLFW 初始化，需要使用 `-tags unittest` 跳过 controller 编译
2. **单窗口限制**：Ebitengine 每进程只能有一个窗口，多局测试只能串行执行
3. **截图依赖 GUI**：截图需要有可渲染的窗口，无头环境不可用

## 九、后续扩展

- [ ] 更多脚本化场景（暂停恢复、事件选择、BuildSellRace 等）
- [ ] LLM 驱动的策略（v2）
- [ ] 自动发现和填充覆盖缺口
- [ ] 与 CI 集成（GPU runner）
- [ ] 多线程/多进程并行执行
