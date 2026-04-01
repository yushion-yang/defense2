# 性能优化 — 剩余项

> 已完成项归档见 `docs/archive/2026-03-31-performance-optimization-completed.md`
>
> 已完成: PerfTracker + debug overlay / DrawImageOptions 栈化(-130 allocs) / TopBar strconv(-10 allocs) / Pipeline uniform 预分配(-3 allocs) / 地图缓存(-80 draws) / TopBar dirty-flag(-25 draws) / Quality 三档 + 自适应

## Phase B 剩余：零分配热路径

| 项 | 预估收益 | 说明 |
|----|---------|------|
| B3: InfoPanel fmt.Sprintf 消除 | -15 allocs/帧 | `ui/scale_text.go` DrawScaleText 等内部去 Sprintf |
| B4: FloatText 去 Sprintf | 每次命中 | SpawnDamageText 存 float64，DrawFloatTexts 时格式化 |
| B6: scatterHits 改数组 | -1 alloc/帧 | tick_combat.go 的 map 改固定数组或复用池 |
| B7: drawJaggedBolt 栈数组 | 每闪电 | `make([][2]float32)` 改栈数组 `var pts [13][2]float32` |
| B8: attackStyleLabel switch | 每帧(选中时) | map literal 改 switch 函数 |

## Phase C 剩余：渲染优化

### C3: HP bar 批渲染（-300 draws/帧）
- 所有敌人 HP bar 收集顶点后一次 DrawTriangles 提交
- 类似粒子系统架构

### C4: 弹射物拖尾精简（-600 draws/帧）
- TrailLen 从 6 降到 3，通过 QualityLevel 控制
- Quality Low 时可完全禁用拖尾

## Phase D 剩余

### D3: 设置页
- 暂停菜单增加"画质"选项：高/中/低/自动
- 保存到 persistence

## Benchmark（未开始）

- `tests/bench/bench_test.go` — 满载场景 benchmark + `ReportAllocs`
- 用于验证优化效果和回归检测
