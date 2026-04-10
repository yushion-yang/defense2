# Release vs Debug Build Separation

> Date: 2026-04-10
> Status: Planned (发版前实施)
> Goal: 正式版二进制排除所有测试/调试内容，零运行时开销

## 方案：Go Build Tags

```bash
# 开发版（默认）
make run

# 正式版（排除测试内容）
go build -tags release -o defense2 cmd/game/main.go
```

文件顶部加 `//go:build !release`，编译器直接跳过，不进入二进制。

## 需要隔离的文件

### P0：测试场景（必须排除）

| 文件 | 说明 |
|------|------|
| `internal/scene/vfx_preview.go` | VFX 预览场景 |
| `internal/scene/audio_preview.go` | 音频预览场景 |
| `internal/scene/test_select.go` | 测试场景选择入口 |

注意：test_select 被隔离后，需要确保 Select 场景中的测试入口按钮也被条件编译隐藏。可能需要拆分一个 `select_debug.go` 文件。

### P1：开发工具（独立 binary，不影响主程序）

| 路径 | 说明 |
|------|------|
| `cmd/autoplay/` | 自动对战独立入口 |
| `internal/autoplay/` | 自动对战逻辑 |
| `cmd/llmtools/` | LLM 工具 |

这些已经是独立 `main.go`，正式版不编译对应 cmd 即可。

### P2：可选隔离

| 文件/功能 | 建议 |
|----------|------|
| F2 PerfTracker | 可保留（玩家反馈性能问题时有用）或 build tag |
| `internal/llm/` | build tag 隔离（正式版不需要 LLM 推理） |
| `internal/core/telemetry/` | build tag（仅 autoplay 使用） |
| 散落的 `log.Printf` | 用 log level 控制，不需要 build tag |

### 不隔离

| 内容 | 原因 |
|------|------|
| 设置场景 | 玩家需要 |
| BGM/SFX 系统 | 核心功能 |
| BuffList 等游戏逻辑 | 核心功能 |
| config/ JSON | 嵌入体积可忽略 |
| assets/ 音频图片 | 核心资源 |

## 实施步骤（发版前执行）

### Step 1: 添加 build tag 到测试文件

```go
//go:build !release

package scene
```

加到: vfx_preview.go, audio_preview.go, test_select.go

### Step 2: 条件编译测试入口

从 Select 场景中拆分测试入口按钮：

```go
// select_debug.go
//go:build !release

package scene

func init() {
    showTestButton = true
}

// select.go
var showTestButton = false // release 版默认隐藏
```

### Step 3: Makefile 添加 release target

```makefile
build-release:
	go build -tags release -trimpath -ldflags="-s -w" -o defense2 cmd/game/main.go

build-wasm-release:
	GOOS=js GOARCH=wasm go build -tags release -trimpath -ldflags="-s -w" -o game.wasm cmd/game/main.go
```

`-trimpath` 移除本地路径信息，`-ldflags="-s -w"` 减小二进制体积。

### Step 4: CI 验证

```bash
# CI 中同时编译两个版本，确保 release 版不会因缺少符号报错
go build -tags release ./...
go build ./...
```

### Step 5: 可选 — 版本号注入

```go
// internal/core/game/version.go
var Version = "dev"
var BuildTime = "unknown"
```

```makefile
build-release:
	go build -tags release \
		-ldflags="-s -w -X defense2/internal/core/game.Version=$(VERSION) -X defense2/internal/core/game.BuildTime=$(shell date -u +%Y%m%d%H%M%S)" \
		-o defense2 cmd/game/main.go
```

## 预期效果

| 指标 | 开发版 | 正式版 |
|------|--------|--------|
| 测试场景 | 可用 | 不存在 |
| Autoplay | 可用 | 不编译 |
| PerfTracker | F2 可用 | 可选 |
| 二进制体积 | ~30MB (估) | ~25MB (估) |
| 运行时开销 | 零额外 | 零额外 |
