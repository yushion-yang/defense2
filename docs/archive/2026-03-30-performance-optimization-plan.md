# Performance Optimization Implementation Plan

> **For Claude:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to implement this plan task-by-task.

**Goal:** Eliminate per-frame heap allocations and add performance observability for stable 60fps on mobile.

**Architecture:** 4-phase approach: observability first (perf tracker + debug overlay + benchmarks), then zero-alloc hot paths (DrawImageOptions stack, strconv replacing fmt.Sprintf, pre-allocated uniforms), then render optimization (map cache, HUD dirty-flag), then quality tier system.

**Tech Stack:** Go runtime/metrics, strconv, fixed-size buffers, ebiten.Image offscreen caching.

---

## Overview

| Task | Phase | Feature | Est. alloc reduction |
|------|-------|---------|---------------------|
| 1 | A | Perf tracker + debug overlay | 0 (observability) |
| 2 | B | DrawImageOptions stack alloc | -130/frame |
| 3 | B | TopBar + FloatText zero-alloc | -10/frame |
| 4 | B | Pipeline uniform pre-alloc | -3/frame |
| 5 | C | Map background cache | -80 draws/frame |
| 6 | C | HUD dirty-flag cache (TopBar) | -25 draws/frame |
| 7 | D | Quality level system | adaptive |

---

### Task 1: Perf Tracker + Debug Overlay

**Files:**
- Create: `internal/core/debug/perf.go`
- Modify: `internal/render/hud/debug_overlay.go`
- Modify: `internal/scene/stage.go` (wire perf tracker)
- Test: `tests/core/perf_test.go`

**Step 1: Create perf.go**

```go
// internal/core/debug/perf.go
package debug

import (
    "runtime"
    "sort"
    "time"
)

const historySize = 300

type PerfTracker struct {
    updateTimes [historySize]float64
    drawTimes   [historySize]float64
    cursor      int
    frameCount  int

    updateStart time.Time
    drawStart   time.Time

    // Public stats (updated once per second)
    FPS         float64
    AvgUpdateMs float64
    AvgDrawMs   float64
    P99UpdateMs float64
    P99DrawMs   float64
    GCCount     uint32
    GCPauseUs   uint64
    HeapMB      float64
    AllocsPerSec uint64

    lastTick    time.Time
    lastGCNum   uint32
    lastAllocs  uint64
}

func NewPerfTracker() *PerfTracker {
    return &PerfTracker{lastTick: time.Now()}
}

func (p *PerfTracker) BeginUpdate()          { p.updateStart = time.Now() }
func (p *PerfTracker) EndUpdate()            { p.updateTimes[p.cursor] = float64(time.Since(p.updateStart).Microseconds()) / 1000 }
func (p *PerfTracker) BeginDraw()            { p.drawStart = time.Now() }
func (p *PerfTracker) EndDraw() {
    p.drawTimes[p.cursor] = float64(time.Since(p.drawStart).Microseconds()) / 1000
    p.cursor = (p.cursor + 1) % historySize
    p.frameCount++
    if time.Since(p.lastTick) >= time.Second {
        p.computeStats()
        p.lastTick = time.Now()
    }
}

func (p *PerfTracker) computeStats() {
    n := p.frameCount
    if n > historySize { n = historySize }
    if n == 0 { return }

    p.FPS = float64(p.frameCount) / time.Since(p.lastTick).Seconds()
    p.frameCount = 0

    // Compute avg and p99 from ring buffer
    p.AvgUpdateMs = avg(p.updateTimes[:n])
    p.AvgDrawMs = avg(p.drawTimes[:n])
    p.P99UpdateMs = percentile(p.updateTimes[:n], 99)
    p.P99DrawMs = percentile(p.drawTimes[:n], 99)

    // GC stats (sampled once per second, not per frame)
    var mem runtime.MemStats
    runtime.ReadMemStats(&mem)
    p.HeapMB = float64(mem.HeapAlloc) / (1024 * 1024)
    gcDelta := mem.NumGC - p.lastGCNum
    p.GCCount = gcDelta
    p.lastGCNum = mem.NumGC
    allocDelta := mem.Mallocs - p.lastAllocs
    p.AllocsPerSec = allocDelta
    p.lastAllocs = mem.Mallocs
    if gcDelta > 0 && len(mem.PauseNs) > 0 {
        var totalPause uint64
        for i := uint32(0); i < gcDelta && i < 256; i++ {
            idx := (mem.NumGC - 1 - i) % 256
            totalPause += mem.PauseNs[idx]
        }
        p.GCPauseUs = totalPause / 1000
    } else {
        p.GCPauseUs = 0
    }
}

func avg(vals []float64) float64 {
    if len(vals) == 0 { return 0 }
    sum := 0.0
    for _, v := range vals { sum += v }
    return sum / float64(len(vals))
}

func percentile(vals []float64, pct int) float64 {
    if len(vals) == 0 { return 0 }
    sorted := make([]float64, len(vals))
    copy(sorted, vals)
    sort.Float64s(sorted)
    idx := len(sorted) * pct / 100
    if idx >= len(sorted) { idx = len(sorted) - 1 }
    return sorted[idx]
}
```

**Step 2: Write tests**

```go
// tests/core/perf_test.go
package core_test

import (
    "testing"
    "time"
    "defense2/internal/core/debug"
)

func TestPerfTrackerCreation(t *testing.T) {
    pt := debug.NewPerfTracker()
    if pt == nil {
        t.Fatal("expected non-nil")
    }
    if pt.FPS != 0 {
        t.Error("FPS should start at 0")
    }
}

func TestPerfTrackerRecording(t *testing.T) {
    pt := debug.NewPerfTracker()
    // Simulate 10 frames
    for i := 0; i < 10; i++ {
        pt.BeginUpdate()
        time.Sleep(100 * time.Microsecond)
        pt.EndUpdate()
        pt.BeginDraw()
        time.Sleep(100 * time.Microsecond)
        pt.EndDraw()
    }
    // Stats won't be computed until 1s passes, but no crash
}
```

**Step 3: Extend debug_overlay.go**

Add a `DrawPerf` method that accepts `*debug.PerfTracker` and renders the stats bar. Use `strconv` instead of `fmt.Sprintf` (practice what we preach).

```go
func (o *DebugOverlay) DrawPerf(screen *ebiten.Image, pt *debug.PerfTracker) {
    if !o.Enabled || pt == nil { return }
    // Format: "FPS:60 U:2.1 D:4.3 P99:3.2/6.1 GC:2 Heap:12M"
    // Use fixed buffer + strconv
}
```

**Step 4: Wire into stage.go**

Add `perfTracker *debug.PerfTracker` to StageScene. Call Begin/End around Update and Draw.

**Step 5: Commit**

```bash
git commit -m "feat: add perf tracker with frame time recording and debug overlay"
```

---

### Task 2: DrawImageOptions Stack Allocation (-130 allocs/frame)

**Files:**
- Modify: `internal/render/draw/circle.go:97-159` (4 Sprite functions)
- Test: verify via `go test -benchmem`

**Step 1: Fix all 4 Sprite functions**

Change from heap-escaping pointer to stack-local struct:

```go
// BEFORE (line 104): escapes to heap
op := &ebiten.DrawImageOptions{}

// AFTER: stays on stack
var op ebiten.DrawImageOptions
op.GeoM.Translate(-w/2, -h/2)
op.GeoM.Scale(s, s)
op.GeoM.Translate(cx*Scale, cy*Scale)
screen.DrawImage(img, &op)
```

Apply to all 4 functions: `Sprite` (line 97), `SpriteRotated` (line 113), `SpriteScaled` (line 130), `SpriteScaledRotated` (line 146).

**Key**: The `&op` passed to `DrawImage` is fine — Go escape analysis sees it doesn't outlive the call.

**Step 2: Run tests**

```bash
make test
```

**Step 3: Commit**

```bash
git commit -m "perf: stack-allocate DrawImageOptions in Sprite functions (-130 allocs/frame)"
```

---

### Task 3: TopBar + FloatText Zero-Alloc (-10 allocs/frame)

**Files:**
- Modify: `internal/render/hud/top_bar.go:71-94` (5 fmt.Sprintf)
- Modify: `internal/render/floattext.go:38,48` (2 fmt.Sprintf)
- Create: `internal/render/hud/numfmt.go` (shared int/float formatting helpers)
- Test: `tests/core/numfmt_test.go`

**Step 1: Create numfmt.go with zero-alloc helpers**

```go
// internal/render/hud/numfmt.go
package hud

import "strconv"

// Reusable buffer for number formatting (not concurrent — only used in Draw goroutine).
var numBuf [32]byte

// fmtInt formats an int to string using the shared stack buffer. Zero allocation.
// IMPORTANT: result is only valid until next fmtInt/fmtFloat call.
func fmtInt(v int) string {
    b := strconv.AppendInt(numBuf[:0], int64(v), 10)
    return string(b)
}

// fmtFloat0 formats a float with 0 decimal places.
func fmtFloat0(v float64) string {
    b := strconv.AppendFloat(numBuf[:0], v, 'f', 0, 64)
    return string(b)
}

// fmtWave formats "wave/max" like "3/20".
func fmtWave(wave, max int) string {
    b := strconv.AppendInt(numBuf[:0], int64(wave), 10)
    b = append(b, '/')
    b = strconv.AppendInt(b, int64(max), 10)
    return string(b)
}
```

Note: `string(b)` still allocates a string header, but the `strconv.Append*` avoids the fmt parsing machinery and is ~10x cheaper. For true zero-alloc, the FontManager would need a `DrawBytes` method — but that's over-engineering for now.

**Step 2: Replace fmt.Sprintf in top_bar.go**

```go
// Line 71: fmt.Sprintf("%d", d.Lives) → fmtInt(d.Lives)
// Line 78: fmt.Sprintf("%d", d.Gold) → fmtInt(d.Gold)
// Line 85: fmt.Sprintf("%d/%d", d.Wave, d.MaxWaves) → fmtWave(d.Wave, d.MaxWaves)
// Line 94: fmt.Sprintf("%d", d.Kills) → fmtInt(d.Kills)
```

Remove `"fmt"` import from top_bar.go if no longer needed.

**Step 3: Replace fmt.Sprintf in floattext.go**

```go
// Line 38: fmt.Sprintf("%.0f", damage) → use strconv.FormatFloat
// Line 48: fmt.Sprintf("+%d", amount) → "+" + strconv.Itoa(amount)
```

**Step 4: Write tests for numfmt**

```go
func TestFmtInt(t *testing.T) {
    tests := []struct{ v int; want string }{
        {0, "0"}, {42, "42"}, {-7, "-7"}, {999999, "999999"},
    }
    for _, tt := range tests {
        if got := hud.FmtInt(tt.v); got != tt.want {
            t.Errorf("fmtInt(%d) = %q, want %q", tt.v, got, tt.want)
        }
    }
}
```

Note: Need to export for testing or use internal test.

**Step 5: Commit**

```bash
git commit -m "perf: replace fmt.Sprintf with strconv in TopBar and FloatText"
```

---

### Task 4: Pipeline Uniform Pre-Allocation (-3 allocs/frame)

**Files:**
- Modify: `internal/render/postprocess/pipeline.go` (pre-allocate uniform maps)
- Test: existing postprocess tests must still pass

**Step 1: Add pre-allocated uniform maps to Pipeline struct**

```go
type Pipeline struct {
    // ... existing fields ...

    // Pre-allocated uniform maps (reused each frame).
    uVignette   map[string]any
    uColorGrade map[string]any
    uRadialBlur map[string]any
    uLighting   map[string]any
    uBloomExt   map[string]any
    uBlurH      map[string]any
    uBlurV      map[string]any
    uBloomComb  map[string]any
}
```

In `NewPipeline()`, initialize all maps with their keys:
```go
p.uVignette = map[string]any{"Strength": float32(0)}
p.uColorGrade = map[string]any{"TintR": float32(0), "TintG": float32(0), "TintB": float32(0), "TintA": float32(0)}
// etc.
```

**Step 2: Replace inline map literals in Apply()**

```go
// BEFORE:
Uniforms: map[string]any{"Strength": float32(fx.VignetteStrength)},

// AFTER:
p.uVignette["Strength"] = float32(fx.VignetteStrength)
// ...
Uniforms: p.uVignette,
```

**Step 3: Same for buildLightingUniforms — update values in pre-allocated map**

**Step 4: Run tests**

```bash
make test
```

**Step 5: Commit**

```bash
git commit -m "perf: pre-allocate shader uniform maps (zero alloc in Apply)"
```

---

### Task 5: Map Background Cache (-80 draws/frame)

**Files:**
- Modify: `internal/render/draw_map.go` (add caching layer)
- Modify: `internal/scene/stage.go` (pass dirty flag)

**Step 1: Add cache to DrawMap**

```go
var mapCache *ebiten.Image
var mapCacheDirty = true

func InvalidateMapCache() { mapCacheDirty = true }

func DrawMap(screen *ebiten.Image, gm *gamemap.GameMap, ...) {
    w, h := screen.Bounds().Dx(), screen.Bounds().Dy()
    if mapCacheDirty || mapCache == nil || mapCache.Bounds().Dx() != w {
        if mapCache != nil { mapCache.Deallocate() }
        mapCache = ebiten.NewImage(w, h)
        drawMapInternal(mapCache, gm, ...)  // existing logic
        mapCacheDirty = false
    }
    screen.DrawImage(mapCache, nil)
}
```

Move existing DrawMap logic into `drawMapInternal`.

**Step 2: Call InvalidateMapCache on map load/event path change**

In stage.go, after map events that modify paths.

**Step 3: Commit**

```bash
git commit -m "perf: cache map background rendering (redraw only on change)"
```

---

### Task 6: HUD Dirty-Flag Cache — TopBar (-25 draws/frame)

**Files:**
- Modify: `internal/render/hud/top_bar.go` (add dirty-flag cache)

**Step 1: Add cache to TopBar**

```go
var topBarCache *ebiten.Image
var topBarLastData TopBarData

func DrawTopBar(screen *ebiten.Image, d TopBarData) {
    if topBarCache == nil || d != topBarLastData {
        w := screen.Bounds().Dx()
        if topBarCache == nil || topBarCache.Bounds().Dx() != w {
            if topBarCache != nil { topBarCache.Deallocate() }
            topBarCache = ebiten.NewImage(w, int(theme.TopBarH)+20)
        }
        topBarCache.Clear()
        drawTopBarInternal(topBarCache, d)
        topBarLastData = d
    }
    screen.DrawImage(topBarCache, nil)
}
```

TopBarData is already a struct — `d != topBarLastData` works if all fields are comparable (they are: ints, bools, floats).

**Step 2: Move existing draw logic to drawTopBarInternal**

**Step 3: Commit**

```bash
git commit -m "perf: dirty-flag cache for TopBar HUD (redraw only on data change)"
```

---

### Task 7: Quality Level System

**Files:**
- Create: `internal/core/game/quality.go`
- Modify: `internal/render/postprocess/pipeline.go` (apply quality)
- Modify: `internal/render/particle/particle.go` (respect quality max)
- Modify: `internal/scene/stage.go` (adaptive logic)
- Test: `tests/core/quality_test.go`

**Step 1: Define quality levels**

```go
// internal/core/game/quality.go
package game

type QualityLevel int

const (
    QualityHigh QualityLevel = iota
    QualityMedium
    QualityLow
)

type QualitySettings struct {
    PostProcessing bool
    MaxParticles   int
    MaxLights      int
    TrailLen       int
    TargetTPS      int
}

var QualityPresets = map[QualityLevel]QualitySettings{
    QualityHigh:   {true, 2048, 4, 6, 60},
    QualityMedium: {true, 1024, 2, 3, 60},
    QualityLow:    {false, 512, 0, 1, 30},
}

// Current is the active quality level (default High for desktop).
var Current = QualityHigh
func Settings() QualitySettings { return QualityPresets[Current] }
```

**Step 2: Add adaptive ticker**

```go
type QualityAdaptive struct {
    slowFrames int
    fastFrames int
}

func (q *QualityAdaptive) Tick(frameMs float64) {
    if frameMs > 14 {
        q.slowFrames++
        q.fastFrames = 0
    } else if frameMs < 10 {
        q.fastFrames++
        q.slowFrames = 0
    } else {
        q.slowFrames = 0
        q.fastFrames = 0
    }
    if q.slowFrames > 30 && Current > QualityLow {
        Current--
        q.slowFrames = 0
    }
    if q.fastFrames > 120 && Current < QualityHigh {
        Current++
        q.fastFrames = 0
    }
}
```

**Step 3: Wire quality into pipeline and particle pool**

Pipeline: `if !game.Settings().PostProcessing { skip vignette/lighting }`
Particle: `if pool.ActiveCount() > game.Settings().MaxParticles { skip spawn }`

**Step 4: Write tests**

```go
func TestQualityPresets(t *testing.T) {
    for level, s := range game.QualityPresets {
        if s.MaxParticles <= 0 {
            t.Errorf("level %d: MaxParticles should be positive", level)
        }
    }
}

func TestAdaptiveDowngrade(t *testing.T) {
    game.Current = game.QualityHigh
    qa := &game.QualityAdaptive{}
    for i := 0; i < 35; i++ {
        qa.Tick(15.0) // slow frames
    }
    if game.Current >= game.QualityHigh {
        t.Error("should have downgraded from High")
    }
}
```

**Step 5: Commit**

```bash
git commit -m "feat: add quality level system with adaptive frame-rate downgrade"
```

---

## Verification

After all tasks, run:

```bash
make test                           # all tests pass
go test -bench=. -benchmem ./tests/bench/  # allocation baseline
make run                            # visual verification with F key debug overlay
```

Expected observable improvements:
- Debug overlay shows real FPS/Update/Draw/GC data
- `go test -benchmem` shows <10 allocs per Update frame
- Map and TopBar only redraw when data changes
- Quality auto-adjusts if frame time spikes
