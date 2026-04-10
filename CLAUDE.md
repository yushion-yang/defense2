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
- **Pattern**: Scene state machine (title -> select -> stage -> result)
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

1. **禁止在 worktree 中操作主仓库工作区。** 在 worktree 工作时，不得对主仓库执行 `git -C 主仓库 merge/stash/checkout/reset` 等任何修改工作区的命令。主仓库可能有大量未 commit 的工作文件（文档、截图、配置），`git stash` 会静默丢失无法干净恢复的文件。合入主干必须由用户在主仓库中手动执行，或等用户明确确认后再操作。曾因 worktree 合入时 stash push→merge→stash pop 丢失 700+ 个文件（review-guides/、visual_review、截图）。

2. **会话结束前 commit 所有变更。** 新建或修改了文件就 `git add` + `git commit`，commit message 根据实际内容写。worktree 是隔离分支，commit 零风险；未 commit 的文件在后续操作中可能丢失。

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
