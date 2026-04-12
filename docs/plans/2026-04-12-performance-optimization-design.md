# Performance Optimization Design

## Problem

The game needs to maintain 60 FPS under heavy combat (150+ enemies, 200+ projectiles, multiple beams, VFX).
Current worst-case estimate: ~2,100 draw calls + 262K collision checks per frame.

## Analysis Summary

### Current Bottlenecks (by severity)

| # | Issue | Category | Impact |
|---|-------|----------|--------|
| 1 | Projectile-enemy collision O(P*E) = 262K checks/frame | CPU | Critical |
| 2 | Enemy pool scanned 10-12x/frame, always 256 slots | CPU | High |
| 3 | LinearGradientV allocates ~10MB GPU texture/frame (non-Stage scenes) | GC+GPU | High |
| 4 | Projectile trail `make([]TrailPt, 6)` per projectile per frame | GC | Medium |
| 5 | TickProjectileHits `make(map)` per frame | GC | Medium |
| 6 | RecalcStats on ALL towers every frame | CPU | Medium |
| 7 | No viewport culling on camera-enabled maps | GPU | Medium |
| 8 | HUD 18+ fmt.Sprintf per frame | GC | Low |
| 9 | Beam 25+ draw calls per beam | GPU | Low |
| 10 | FillPath(nil,nil) allocates vertex/index per call | GC | Low |
| 11 | hover.go AppendTouchIDs allocates per frame | GC | Low |

### Already Optimized

- Map rendering: offscreen cache, single blit
- Particle system: batched DrawTriangles (2048 particles, 1 GPU call)
- Object pools: fixed-size arrays (enemy 256, tower 64, projectile 1024)
- Shader uniforms: pre-allocated
- Quality adaptive: 3-tier with auto-switch
- Sprite DrawImageOptions: stack-allocated

## Design

### Phase A: Zero-alloc Hot Paths (7 items, low risk)

#### A1. LinearGradientV -> CachedGradient
- Scenes: select, campaign_select, test_select, warden_select, settings, map_editor
- Each scene creates a `CachedGradient` in its init/constructor
- `Draw()` calls `cachedGrad.Draw(screen)` instead of `LinearGradientV()`
- Eliminates ~10MB/frame GPU texture + pixel buffer allocation in 6 scenes

#### A2. Projectile trail pre-allocation
- In `draw_projectile.go`: change from `make([]vfx.TrailPt, TrailLen)` to fixed `[6]vfx.TrailPt` array
- Pass as slice of the array: `pts[:n]`
- Eliminates 12,000 alloc/s at 200 active projectiles

#### A3. TickProjectileHits map reuse
- Add a `towerByKey map[string]*tower.Tower` field to the pipeline context or combat struct
- Clear and reuse each frame instead of `make()` each frame
- Or: pass tower pool directly and use `Lookup()` instead of building a map

#### A4. fmt.Sprintf -> strconv in HUD
- Replace `fmt.Sprintf("%d", n)` with `strconv.Itoa(n)` or `strconv.AppendInt` patterns
- Target files: wave_panel.go, build_menu.go, info_panel.go, item_panel.go, debug_overlay.go
- ~18 call sites

#### A5. RecalcStats dirty flag
- Add `statsDirty bool` to tower pool or warden struct
- Only set when warden actually applies/removes a buff
- `RecalcStats` only runs on towers where `statsDirty == true`
- Reset flag after recalc

#### A6. FillPath pre-allocated vertex buffers
- Package-level `var fillVs []ebiten.Vertex` and `var fillIs []uint16`
- `FillPath` passes these to `AppendVerticesAndIndicesForFilling(fillVs[:0], fillIs[:0])`
- Ebitengine appends to the provided slices, reusing their backing arrays

#### A7. hover.go AppendTouchIDs pre-allocation
- Package-level `var touchBuf []ebiten.TouchID`
- Pass to `AppendTouchIDs(touchBuf[:0])` and `AppendJustPressedTouchIDs(touchBuf[:0])`

### Phase B: Structural Optimizations (4 items, medium risk)

#### B8. Enemy pool ActiveList
- Add `activeIndices []int` (cap 256) to enemy pool
- Maintain on `Spawn()` (append index) and `Kill()` (swap-remove)
- New method `EachActive(fn)` iterates only `activeIndices`
- Migrate all `Each()` call sites to `EachActive()`
- Reduces 12 scans x 256 to 12 scans x actual count

#### B9. Merge enemy update passes
- Combine Steps 4-8 (status effects, teleport, movement, spawn anim, dying anim) into a single `Each()` call
- Create a `tickEnemyFrame(e *Enemy, dt float64)` function that runs all 5 checks per enemy
- Reduces 5 x N iterations to 1 x N

#### B10. Viewport culling
- When camera is active, compute visible rect in world coords
- `DrawEnemies` / `DrawTowers` / `DrawProjectiles` skip entities outside visible rect (with margin)
- Simple AABB check per entity: `if e.X < viewLeft-margin || e.X > viewRight+margin || ...`
- Margin = 100 logical pixels (for trails, glow bleeding)

#### B11. Beam rendering simplification
- Define beam detail levels: Full (25+ calls), Reduced (12 calls), Minimal (5 calls)
- Map to quality settings: High=Full, Medium=Reduced, Low=Minimal
- Reduced: skip energy nodes, edge sparks, atmospheric glow
- Minimal: core line + impact flare only

### Phase C: Spatial Indexing (1 item, high reward)

#### C12. Grid-based spatial partitioning for collision
- Create `internal/core/physics/grid.go`
- Grid cell size = 64 logical pixels (roughly enemy radius + projectile range)
- Grid dimensions: ceil(1200/64) x ceil(540/64) = 19 x 9 = 171 cells
- Each cell holds a `[]int` of enemy indices (pre-allocated, cap ~16)
- **Rebuild** every frame at start of combat step (single enemy pass)
- **Query** in TickProjectileHits: for each projectile, check only cells within projectile radius
  - Typical: 1-4 cells checked instead of 256 enemies
  - O(P * k) where k ≈ 4-8 enemies per cell region
- Expected reduction: 262K -> ~4K distance checks at peak

## Execution Order

1. A1-A7 (independent, can parallelize)
2. C12 (spatial grid — biggest single improvement)
3. B8 (ActiveList — enables B9)
4. B9 (merge passes — depends on B8)
5. B10 (viewport culling)
6. B11 (beam simplification)

## Verification

- `make test` passes after each phase
- `go run cmd/autoplay/main.go --scenario attack-style-coverage` for runtime validation
- Compare PerfTracker P99 before/after on a heavy scenario
- Check `runtime.ReadMemStats` alloc/s reduction
- Manual play-test on worst-case map (map_04 or map_05, late waves)

## Risk Assessment

- **A changes**: Pure optimization, no behavior change. Lowest risk.
- **B8/B9**: Changes iteration order. Must verify all side effects are order-independent.
- **B10**: Could cause pop-in if margin too small. Use generous margin (100px).
- **B11**: Visual quality reduction on lower settings. Acceptable trade-off.
- **C12**: New data structure. Must match exact collision semantics. Test with edge cases (enemies at cell boundaries, fast projectiles crossing cells).
