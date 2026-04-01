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

## Conventions

- Go idioms: accept interfaces, return structs
- Error wrapping: `fmt.Errorf("context: %w", err)`
- Tests: table-driven, `-race` flag always
- Ability registration: `init()` + blank import pattern
