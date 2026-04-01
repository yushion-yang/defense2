# 性能优化设计

> 生成日期: 2026-03-30 | 分支: claude/migration-44items
>
> **目标**: 移动端帧率稳定 60fps（中低端 Android/iOS）
>
> **现状**: 桌面端 OK，移动端尚未构建测试。属于预防性优化。
>
> **原则**: 先能测量，再按投入产出比优化。不做过度工程。

## 性能基线（桌面分析）

| 指标 | 当前值 | 目标 |
|------|-------|------|
| Draw primitives/帧 | ~3000（繁忙帧） | <1000 |
| 堆分配/帧 | ~150+ | <10 |
| 碰撞检测迭代/帧 | 20K（200弹×100敌） | 保持（early-exit 够用） |
| 性能基准测试 | 0 | 有 benchmark + 实时 overlay |
| 嵌入资源 | 23MB（15MB 字体） | 暂不动 |

### 分配热点 Top 8

| 排名 | 来源 | 次数/帧 | 类型 |
|------|------|---------|------|
| 1 | `&DrawImageOptions{}` in Sprite* | ~130 | struct escape |
| 2 | `fmt.Sprintf` in InfoPanel | ~15（选中时） | string |
| 3 | `fmt.Sprintf` in TopBar | 5 | string |
| 4 | `map[string]any` in pipeline.Apply | 1-3 | map |
| 5 | `fmt.Sprintf` in FloatText | 每次命中 | string |
| 6 | `scatterHits` map in TickProjectileHits | 1 | map |
| 7 | `make([][2]float32)` in drawJaggedBolt | 每闪电 | slice |
| 8 | `attackStyleLabel` map literal | 每帧（选中时） | map |

### Draw call 分布

| 类别 | 繁忙帧 draw calls |
|------|-------------------|
| 敌人（100个）| ~700（5-8 calls/个）|
| 弹射物（200个）| ~1600（trail 6 + body 2）|
| 塔（30个）| ~90 |
| HUD | ~50-80 |
| 地图 | ~50-80 |
| 其他（VFX/光束/战灵）| ~200 |
| 粒子（已批渲染）| 1 |

---

## 演进路线

```
Phase A: 观测基础设施 ——— 帧时间追踪 + debug overlay + benchmark
Phase B: 零分配热路径 ——— 消除每帧堆分配（GC 卡顿元凶）
Phase C: 渲染优化 ——— HUD 缓存 + 地图预渲染 + HP bar 批渲染
Phase D: 质量分级 ——— 移动端自适应帧率/特效/粒子数
```

不做（投入产出比低）:
- DrawTriangles 全量批渲染（需纹理图集工具链，改动巨大）
- O(P*E) 碰撞空间索引（tracking projectile early-exit 已经很有效）
- 资源懒加载/压缩（等移动端实测后再定）

---

## Phase A: 观测基础设施

**问题**: 零性能数据。没有数据就没有优化依据。

### A1: 帧时间追踪器

`internal/core/debug/perf.go`

```go
type PerfTracker struct {
    updateTimes [300]float64  // 环形缓冲，最近 300 帧
    drawTimes   [300]float64
    cursor      int

    // 每秒统计
    AvgUpdate   float64  // ms
    AvgDraw     float64  // ms
    P99Update   float64  // ms
    P99Draw     float64  // ms
    FPS         float64
    GCCount     uint32
    GCPauseMs   float64
    HeapAlloc   uint64   // bytes

    lastStatsTime float64
    lastGCNum     uint32
}
```

- `BeginUpdate()` / `EndUpdate()` 记录 Update 耗时
- `BeginDraw()` / `EndDraw()` 记录 Draw 耗时
- 每秒调一次 `runtime.ReadMemStats` 更新 GC 数据（不每帧调，开销大）
- stage.go 在 Update/Draw 开头结尾调用

### A2: Debug overlay 扩展

`hud/debug_overlay.go` 添加性能行：
```
FPS: 60 | U: 2.1ms D: 4.3ms | P99: 3.2/6.1 | GC: 2/s 0.3ms | Alloc: 12MB
```

按 F 键切换显示。

### A3: Benchmark test

`tests/bench/bench_test.go`

```go
func BenchmarkUpdateFullLoad(b *testing.B) {
    // 构造满载场景：100 敌人 + 30 塔 + 200 弹射物
    // b.ReportAllocs() 报告每帧分配
}
```

`go test -bench=. -benchmem ./tests/bench/` 提供基线数据。

---

## Phase B: 零分配热路径

**问题**: ~150+ 堆分配/帧 → Go GC pause → 帧时间尖刺。

### B1: DrawImageOptions 栈分配（-130 allocs/帧）

`draw/circle.go` Sprite* 函数：
```go
// BEFORE: escapes to heap
opts := &ebiten.DrawImageOptions{}

// AFTER: stays on stack
var opts ebiten.DrawImageOptions
screen.DrawImage(img, &opts)
```

确保 `opts` 不被存储到任何持久结构中（否则逃逸分析会把它放到堆上）。

### B2: TopBar fmt.Sprintf 消除（-5 allocs/帧）

```go
// BEFORE
text := fmt.Sprintf("%d", gold)

// AFTER
var buf [20]byte
text := strconv.AppendInt(buf[:0], int64(gold), 10)
fm.DrawText(screen, string(text), x, y, size, clr)
```

或者 FontManager 增加 `DrawInt(screen, val int, x, y, size, clr)` 方法直接接受数字。

### B3: InfoPanel fmt.Sprintf 消除（-15 allocs/帧）

同 B2 模式。`ui/scale_text.go` 的 DrawScaleText 等内部也需要去 Sprintf。

### B4: FloatText 去 Sprintf

`SpawnDamageText` 改为存 `float64` 值而非 `string`，DrawFloatTexts 时用 `strconv.FormatFloat` + stack buffer 渲染。

### B5: Pipeline uniform 预分配（-3 allocs/帧）

Pipeline 结构体上预分配 uniform maps：
```go
type Pipeline struct {
    // ... existing fields ...
    vignetteUniforms   map[string]any  // pre-allocated in NewPipeline
    colorGradeUniforms map[string]any
    lightingUniforms   map[string]any
}
```

Apply() 只更新 value，不创建新 map。

### B6: scatterHits 改数组（-1 alloc/帧）

`tick_combat.go` 的 `scatterHits` 从 `map[int64]*scatterHit` 改为固定数组或复用池。

### B7: drawJaggedBolt 栈数组

```go
// BEFORE
pts := make([][2]float32, segs+1)

// AFTER (segs max 12)
var pts [13][2]float32
```

### B8: attackStyleLabel switch 替换 map

```go
// BEFORE
labels := map[string]string{"projectile": "投射", ...}

// AFTER
func attackStyleLabel(s string) string {
    switch s { case "projectile": return "投射" ... }
}
```

---

## Phase C: 渲染优化

### C1: 地图背景预渲染（-80 draws/帧）

```go
type MapCache struct {
    image *ebiten.Image
    dirty bool
}
```

DrawMap 改为：地图加载时渲染到 offscreen image（渐变背景 + 网格 + 路径），之后每帧 blit。只在地图变化（事件修改路径）时标 dirty 重绘。

### C2: HUD dirty-flag 缓存（-50 draws/帧）

TopBar 和 WavePanel 数据每秒才变几次，不需要每帧重绘 25+ draw calls。

```go
type CachedPanel struct {
    image    *ebiten.Image
    lastData any  // 上次渲染的数据快照
}

func (c *CachedPanel) DrawIfChanged(screen *ebiten.Image, data any, drawFn func(*ebiten.Image)) {
    if !reflect.DeepEqual(c.lastData, data) {
        c.image.Clear()
        drawFn(c.image)
        c.lastData = data
    }
    screen.DrawImage(c.image, posOpts)
}
```

**注意**: 用 struct 比较替代 reflect.DeepEqual（零分配）。

### C3: HP bar 批渲染（-300 draws/帧）

所有敌人 HP bar 是相同模式（背景灰 + 填充绿/红 + 边框）。可以像粒子系统一样收集所有 bar 的顶点，一次 DrawTriangles 提交。

```go
type HPBarBatch struct {
    vertices []ebiten.Vertex
    indices  []uint16
}
```

DrawEnemies 时不直接画 HP bar，而是收集到 batch，最后一次提交。

### C4: 弹射物拖尾精简（可选，-600 draws/帧）

TrailLen 从 6 降到 3。视觉差异小，draw call 减半。通过 QualityLevel 控制。

---

## Phase D: 质量分级 + 自适应

### D1: QualityLevel 定义

```go
type QualityLevel int
const (
    QualityHigh   QualityLevel = iota
    QualityMedium
    QualityLow
)

type QualitySettings struct {
    PostProcessing  bool    // vignette/lighting
    MaxParticles    int     // 2048/1024/512
    MaxLights       int     // 4/2/0
    TrailLen        int     // 6/3/1
    HPBarDetail     int     // 0=full, 1=simple, 2=minimal
    TargetFPS       int     // 60/60/30
}
```

### D2: 自适应逻辑

```go
func (q *QualityAdaptive) Tick(frameTimeMs float64) {
    q.history[q.cursor] = frameTimeMs
    q.cursor = (q.cursor + 1) % 60

    // 连续 30 帧超标 → 降级
    if q.consecutiveSlow > 30 {
        q.Downgrade()
    }
    // 连续 60 帧余裕 → 升级
    if q.consecutiveFast > 60 {
        q.Upgrade()
    }
}
```

阈值：
- Slow: Update+Draw > 14ms（留 2ms 余量给 60fps 的 16.67ms）
- Fast: Update+Draw < 10ms

### D3: 设置页

在暂停菜单增加"画质"选项：高/中/低/自动。保存到 persistence。

---

## 依赖关系

```
Phase A (观测) ← 独立，最先做
   ↓
Phase B (零分配) ← 用 A 的 benchmark 验证效果
   ↓
Phase C (渲染) ← 用 A 的 overlay 观察 draw call 变化
   ↓
Phase D (分级) ← 依赖 A 的帧时间数据做自适应
```

## 预期收益

| Phase | 堆分配减少 | Draw call 减少 | 工作量 |
|-------|-----------|---------------|--------|
| A | 0 | 0 | 小（3 文件） |
| B | -150/帧 → <10/帧 | 0 | 中（8 处修改） |
| C | 0 | -430/帧 | 中（4 个新模式） |
| D | 0 | 按等级 | 小（配置驱动） |
| **合计** | 95% 减少 | 15% 减少 | |

B 阶段消除 GC 压力是对移动端帧率稳定性影响最大的。C 阶段减少 GPU 提交次数。D 阶段兜底。
