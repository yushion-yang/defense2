# V1 Release Checklist

## Already Done

- [x] **Interaction audit & fix** — 8 issues (Delete double-fire, UI slide-off cancel, variant card click, config sync, dead code cleanup)
- [x] **Android long-press hover** — `draw/hover.go` global tracker, 500ms threshold, all 13 scenes migrated to `HoverPos()`
- [x] **Mode locking** — 非战役模式显示"敬请期待"，测试模式完全隐藏，战役地图保持逐步解锁
- [x] **Test fixes** — 3 contract test failures fixed (speedPerWave/damageUp/flatDamage), economy.json synced, combat_test params updated
- [x] **Resource check** — BGM 三首已就绪, PlayBGM/PlaySafe 均有 graceful fallback

## Remaining

### P0: Must-Do Before Release

- [ ] **Autoplay sweep regression** — 桌面运行 `go run cmd/autoplay/main.go --sweep`，检查 68 局有无运行时异常/economy_stall/平衡崩坏。当前环境无显示器无法跑
- [ ] **Android build verification** — 触摸交互、长按悬浮、性能帧率、UI 适配（有 session 在做）
- [ ] **Tutorial flow** — 新玩家首次进入 map_01 的引导是否完整顺畅（建塔→选能力→开波→通关）
- [ ] **Critical path playthrough** — 手动完整走一遍：Title → Select → CampaignSelect → map_01 Stage → Victory → Result → 返回选关

### P1: Should-Do

- [ ] **War warden select UX** — 战灵选择界面对新玩家是否清晰（需要提示"这是你的英雄角色"）
- [ ] **Locked mode visual polish** — "敬请期待"卡片考虑加锁图标或半透明遮罩，目前仅暗色+文字
- [ ] **Result scene review** — 结算画面动画序列、星级评价、统计数据是否显示正确
- [ ] **Settings persistence** — SFX/BGM 音量和画质设置是否在重启后保留
- [ ] **Performance profiling** — F2 overlay 查看 FPS/P99/GC，确保 60fps 稳定（尤其大地图 map_04/05）
- [ ] **Edge case: 0 lives** — extreme 模式 10 lives，前几波大量泄漏时生命归零流程是否正常

### P2: Nice-to-Have

- [ ] **Loading scene** — 加载画面是否有进度提示，冷启动时间是否可接受
- [ ] **Version string** — `v0.1.0` 硬编码在 select.go/title.go，发布时统一管理
- [ ] **App icon & splash** — Android app icon 和启动画面
- [ ] **Crash reporting** — 生产环境 panic 是否有日志可查

## How to Run Autoplay

```bash
# Desktop terminal (needs display)
cd ~/Games/defense2

# Quick smoke test (1 game, 30s)
go run cmd/autoplay/main.go --scenario attack-style-coverage

# Full regression (68 games, ~3min)
go run cmd/autoplay/main.go --sweep --json-dir docs/autotest/M1 --png-dir docs/autotest/M2

# Check results
cat docs/autotest/M1/coverage_summary.json
```

## How to Test Android

```bash
make build-wasm   # WASM build for browser testing
# Or use gomobile for APK
```
