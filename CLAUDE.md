# defense2

Go/Ebitengine tower defense game. Full port from JS version.

## Commands

- `make run` — desktop dev
- `make test` — run tests
- `make lint` — golangci-lint
- `make check-all` — lint + test
- `make build-wasm` — WASM build

## Architecture

- **Engine**: Ebitengine v2.9.9
- **Pattern**: Scene state machine: Title → Select → {CampaignSelect | TestSelect | Settings} → Stage(含 WardenSelect 覆盖层) → Result
- **Layout**: Landscape 1200x540
- **Config**: JSON via `//go:embed`, reused from JS version

## Project Structure

- `cmd/game/` — desktop entry
- `cmd/mobile/` — Android entry
- `internal/scene/` — scene management
- `internal/core/` — game logic (tower, enemy, projectile, hero, warden, event, physics, pipeline)
- `internal/config/` — JSON config loading
- `internal/render/` — rendering (svg parser, draw_*, hud/)
- `internal/input/` — unified input
- `config/` — JSON data files
- `assets/` — SVG models, audio, fonts
- `tests/` — unit / integration / design tests

## Rendering (HiDPI)

`LayoutF` 返回原生物理分辨率，`draw` 包内部自动处理坐标缩放（等同于浏览器 Canvas 的 devicePixelRatio 行为）。
**所有代码使用逻辑坐标 (1200×540)，不需要关注设备缩放。** 详见 `docs/rendering-hidpi.md`。

### 规则：用 draw.* 不用 vector.*

```go
draw.Line(screen, x1, y1, x2, y2, width, clr, aa)
draw.FilledRect(screen, x, y, w, h, clr, aa)
draw.FilledCircle(screen, cx, cy, r, clr)
draw.CircleOutline(screen, cx, cy, r, width, clr)
draw.RoundRect(screen, x, y, w, h, radius, clr)
draw.Glow(screen, cx, cy, innerR, outerR, clr)
draw.Diamond(screen, cx, cy, r, width, clr)
draw.ThickLine(screen, x1, y1, x2, y2, width, clr)
draw.DashedLine / draw.DashedCircle
```

### 规则：精灵用 draw.Sprite*

```go
draw.Sprite(screen, img, cx, cy, 64)                            // 居中绘制
draw.SpriteScaled(screen, img, cx, cy, logicalScale)            // 自定义缩放
draw.SpriteRotated(screen, img, cx, cy, 24, rotation, offsetY)  // 带旋转
```

### 规则：鼠标/触摸用 draw.CursorPos / draw.TouchPos

```go
mx, my := draw.CursorPos()           // 返回逻辑坐标，不要用 ebiten.CursorPosition()
tx, ty := draw.TouchPos(touchID)     // 返回逻辑坐标，不要用 ebiten.TouchPosition()
```

### Font

- `assets/fonts/NotoSansSC-Regular.ttf` (Noto Sans SC, simplified Chinese)
- Global singleton: `render.InitGlobalFont()` → `render.GlobalFont()`
- `DrawText / DrawCenteredText / DrawRightText / MeasureText` — 传逻辑坐标，内部自动缩放

## 最高优先级规则（违反即事故）

1. **禁止在 worktree 中合入代码到主仓库。** 默认不得从 worktree 向主仓库执行 merge/stash/checkout/reset 等任何操作，除非用户明确要求。主仓库可能有大量未 commit 的工作文件（文档、截图、配置），`git stash` 会静默丢失无法干净恢复的文件。曾因 worktree 合入时 stash push→merge→stash pop 丢失 700+ 个文件。

2. **会话结束前 commit 所有变更。** 新建或修改了文件就 `git add` + `git commit`，commit message 根据实际内容写。worktree 是隔离分支，commit 零风险；未 commit 的文件在后续操作中可能丢失。

3. **禁止写入 config/scenarios/ 目录。** 该目录存放用户手动保存的测试场景快照，仅由游戏内 Stage 调试面板的「保存场景」功能写入。AI 会话不得在此目录创建、修改或删除任何文件。曾因 AI 直接写入 8 个 autoplay 场景文件污染整个目录。

## 架构原则（强制）

1. **问题处理必须追根因。** 每次修复问题时，先回答三个问题再动手：(1) 根因是什么（设计/架构层面，不是症状）；(2) 当前设计是否需要调整才能杜绝同类问题；(3) 如果需要调整，先提出方案等确认，再修具体 bug。修表面症状之前，先判断是否该修地基。

2. **HUD 文本必须自适应容器宽度。** 禁止假设文本长度。新增任何 DrawText 调用时，必须确认容器边界处理（截断/缩放/换行）。

3. **配置即一切。** 如果 JSON 配置中已有对应字段，代码中禁止硬编码同类值。新增游戏参数时，先加配置字段，再从配置读取，禁止代码内 magic number。

## VFX 解耦规范（强制）

所有视觉效果必须实现在 `internal/render/vfx/` 包中：

1. **零 core 依赖**：vfx 包只 import `render/draw`、`render/theme`、标准库，禁止 import `core/*`
2. **纯值参数**：函数只接受基本类型（float, int, color, bool, 小值 struct），不接受 Tower/Enemy 等游戏对象
3. **可独立预览**：每个效果必须能在 VFX Preview 场景中单独触发
4. **调用方解析**：`draw_*.go` 负责从游戏对象提取参数，然后调用 `vfx.DrawXxx()`
5. **新效果必须同步更新**：
   - `internal/render/vfx/` 中添加函数
   - `config/visuals/vfx.json` 中添加元数据（id/name/label/description）
   - `vfx_preview.go` 的 trigger registry 中注册

## Conventions

- Go idioms: accept interfaces, return structs
- Error wrapping: `fmt.Errorf("context: %w", err)`
- Tests: table-driven, `-race` flag always
- Ability registration: `init()` + blank import pattern

## HUD 组件规范（强制）

hud/ 包禁止直接调用底层渲染 API，必须通过 ui/ 组件模板：

| 需求 | 组件 | 禁止 |
|------|------|------|
| 单行文本 | `ui.Label` / `ui.LabelV` | `fm.DrawText` 等 |
| 多行文本 | `ui.Paragraph` | 手动 WrapText 循环 |
| 图标+文本 | `ui.IconLabel` | 手动 Sprite+DrawText |
| 面板背景 | `ui.Panel` / `ui.PanelBox` | `draw.RoundRect` |
| 全屏遮罩 | `ui.Overlay` | `draw.FilledRect` 全屏 |
| 浮动提示 | `ui.Tooltip` | 手动 RoundRect+StrokeRoundRect |
| 卡片网格 | `ui.CardGrid` | 手动坐标计算 |
| 垂直堆叠 | `ui.VStack` | 手动 Y 累加 |
| 分隔线 | `ui.Divider` | `draw.Line` |

- 精灵渲染（`draw.Sprite*`）和特殊图形（minimap 路径点、动画边框闪光）用 `//nolint:hud` 豁免
- 字号必须用 `theme.Font*` 常量，禁止数字字面量
- 屏幕尺寸必须用 `theme.CanvasW/H`，禁止 `game.ScreenWidth/Height`
- `tests/lint/hud_lint_test.go` 自动检测违规，`make test` 拦截

## 代码注释规范（强制）

### 语言与详细度

- **语言**: 中文
- **详细度**: 中级 — 解释本项目特有的设计决策和非显而易见的逻辑，不解释通用编程模式（如"什么是 for 循环"）

### 必须添加注释的位置

| 位置 | 内容 |
|------|------|
| 文件头部 | 说明文件在系统中的角色、与其他文件的关系 |
| 大型函数开头(>50行) | 流程概览注释（编号步骤列表） |
| 复杂 struct(>10字段) | 按逻辑分组添加分隔注释 |
| 关键分支/算法 | 说明"为什么"而非"做了什么" |
| 非显而易见的设计决策 | 说明选型理由、替代方案、权衡取舍 |

### 不需要注释的位置

- 通用编程模式（error handling、for loop、type assertion）
- 单行显而易见的赋值/return
- 已有良好命名的短函数(<20行)
- godoc 格式的英文导出注释（本项目不需要）

### 风格示例

```go
// spawner.go — 波次出怪管理器。
//
// 职责：管理波次推进、敌人原型选择、Boss 注入、wave buff 施加。
// 关联：由 pipeline/sys_spawn.go 每帧调用 Tick()，
//       从 config/spawner_config.go 读取波次配置。

// ApplyHit 处理一次命中的完整流程：
//   1. 闪避判定（evasion check）
//   2. 护甲减免
//   3. 触发 OnHit 能力（弹射/溅射/流血/灼烧/减速/眩晕）
//   4. 暴击判定
//   5. 调用 ApplyDamage 进入 8 步伤害管线
//   6. 处理击杀和死亡爆炸
func ApplyHit(...) {

// ── 身份标识 ──────────────────────────
Key       string  // 塔类型键名，如 "basic"
Row, Col  int     // 网格位置

// ── 战斗属性 ──────────────────────────
Damage    float64 // 最终伤害 = Base + Potential * (Strength/100)

// 用环形缓冲区而非 slice：弹道生命周期极短（<2秒），
// FIFO 覆盖比 GC 回收更高效，且保证零分配。
type Pool struct {
```

## Development Workflow（强制）

### 新功能实现流程

**每次实现新功能必须按此顺序，不可跳步：**

```
1. 理解需求 → 确认设计意图（有疑问先问，不要猜）
2. 先写契约测试 → tests/contracts/ 或 tests/regression/
   - 从设计文档/口述提取可验证的断言
   - 测试必须先 FAIL（红灯）
3. 再写实现代码
4. make test → 测试通过（绿灯）
5. 验证集成：
   - 新函数是否被调用（不是"写了就有用"）
   - JSON tag 是否匹配配置文件
   - 接口方法是否全实现
```

### Bug 修复流程

```
1. 先写回归测试复现 bug（测试必须先 FAIL）
2. 修复代码
3. make test → 测试通过
4. 全局搜索同模式代码（确认无同类问题）
5. 如果适合运行时检测 → 加 anomaly 规则到 anomaly.go
6. 更新 memory（记录根因 + 修复方式）
```

### 关键防线

| 层级 | 工具 | 拦截什么 |
|------|------|---------|
| 编译器 | interface / 类型系统 | 接口不全、类型不匹配 |
| 契约测试 | tests/contracts/ | 配置→代码映射错误、数值范围 |
| 回归测试 | tests/regression/ | 已修复 bug 复发 |
| Lint | tests/lint/ | 直接调用 ebiten API |
| Autoplay | cmd/autoplay/ --sweep | 运行时异常(26条规则)、平衡问题 |
| AI 审核 | docs/review-guides/ | 深层逻辑、跨系统一致性 |

### 常见陷阱（AI 必须避免）

1. **写了函数但没调用** — 实现完必须 grep 确认调用链完整
2. **JSON tag 猜测** — 必须打开 JSON 文件核实字段名，不要凭记忆
3. **硬编码常量** — 必须从配置读取，不要写 magic number
4. **只测 happy path** — 必须覆盖：空输入、边界值、池满、目标死亡
5. **改了签名没改调用点** — 改函数签名后必须 grep 所有调用者

### Autoplay 使用

```bash
# 快速验证（1 局，30 秒）
go run cmd/autoplay/main.go --scenario attack-style-coverage

# 全量回归（68 局，~3 分钟）
go run cmd/autoplay/main.go --sweep --json-dir docs/autotest/M1 --png-dir docs/autotest/M2

# 查看结果
cat docs/autotest/M1/coverage_summary.json
```

### AI 源码审核

```bash
# 新 session 中让 AI 按指导文档逐项审核
# 文档位于 docs/review-guides/01~10-*.md（共 192 个检查项）
```
