# V1 Release Checklist

## Already Done

- [x] **Interaction audit & fix** — 8 issues (Delete double-fire, UI slide-off cancel, variant card click, config sync, dead code cleanup)
- [x] **Android long-press hover** — `draw/hover.go` global tracker, 500ms threshold, all 13 scenes migrated to `HoverPos()`
- [x] **Mode locking** — 非战役模式显示"敬请期待"+[锁]标记，测试模式完全隐藏，战役地图保持逐步解锁
- [x] **Test fixes** — 3 contract test failures fixed (speedPerWave/damageUp/flatDamage), economy.json synced, combat_test params updated
- [x] **Resource check** — BGM 三首已就绪, PlayBGM/PlaySafe 均有 graceful fallback
- [x] **Version string** — 集中到 `game.Version` 常量，4 个场景统一引用
- [x] **Result replay** — 重玩按钮保留战灵选择（WardenType），不再重新选
- [x] **Result kills color** — 击杀统计改为中性色（不再用红色）
- [x] **Tutorial done guard** — SetTutorialDone 加 once 标记，不再每帧写入
- [x] **Settings debounce** — 滑条拖动中不写盘，松开时保存一次
- [x] **Warden select UX** — 增加"战灵是与你并肩作战的英雄"说明文案
- [x] **Loading scene** — 审查通过：进度条+百分比+状态文字+脉冲版本号，<1秒加载
- [x] **Settings persistence** — 审查通过：SFX/BGM/Quality 存 UserConfigDir/defense2/settings.json，启动时恢复
- [x] **Edge case: 0 lives** — 审查通过：lives clamp 到 0，defeat 检测正确，多敌同帧泄漏处理正确
- [x] **Crash reporting** — 审查通过：Game.Update/Draw 均有 recover()，audio/lifecycle 也有防护

## Remaining (need desktop/device)

- [ ] **Autoplay sweep regression** — 桌面运行 `go run cmd/autoplay/main.go --sweep`，检查 68 局有无运行时异常
- [ ] **Android build verification** — 触摸交互、长按悬浮、性能帧率、UI 适配（有 session 在做）
- [ ] **Critical path playthrough** — 手动完整走一遍：Title → Select → CampaignSelect → map_01 Stage → Victory → Result → 返回选关
- [ ] **Performance profiling** — F2 overlay 查看 FPS/P99/GC，确保 60fps 稳定
- [ ] **App icon & splash** — Android app icon 和启动画面（需设计资源）

## Known Limitations (not blocking V1)

- Tutorial steps 5-6 (upgrade/item_use) 依赖 AutoAdvance 超时推进，实际操作很少自然触发
- SFXEnabled 有持久化但无 UI 开关（默认 true，玩家可拉 volume=0 达到同等效果）
- `tests/core` 在无显示器环境编译失败（Ebitengine GLFW 初始化 panic），非代码问题

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
