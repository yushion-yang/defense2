# V1 Release Checklist

## All Done

- [x] Interaction audit & fix (8 issues)
- [x] Android long-press hover (`draw/hover.go`)
- [x] Mode locking (campaign only, others "coming soon" + [lock])
- [x] Test fixes (3 contract tests + economy.json + combat_test)
- [x] Resource check (BGM ready, all fallbacks safe)
- [x] Version string centralized (`game.Version`)
- [x] Result replay preserves WardenType
- [x] Result kills color fixed (neutral gray)
- [x] Tutorial done guard (once flag)
- [x] Settings debounce (save on release only)
- [x] Warden select UX (intro text)
- [x] Loading scene reviewed (progress bar, <1s)
- [x] Settings persistence verified
- [x] Edge case: 0 lives verified
- [x] Crash recovery verified (recover in Update/Draw)
- [x] Performance: buildModeCtx cached (zero alloc/frame)
- [x] TitleScene restored in flow (was dead code)
- [x] Autoplay smoke test passed (1 scenario, 0 anomalies)
- [x] Critical path code traced (all transitions safe)

## Remaining (need desktop/device)

- [ ] **Autoplay full sweep** — 68 scenarios running, check results: `cat autoplay-results/run_20260413_003244/coverage_summary.json`
- [ ] **Android build verification** — another session handling
- [ ] **App icon** — needs design assets (not code)

## Known Limitations (not blocking V1)

- Tutorial steps 5-6 (upgrade/item_use) rely on AutoAdvance timeout
- SFXEnabled has no UI toggle (volume=0 achieves same effect)
- `tests/core` panics in headless env (Ebitengine GLFW, not code issue)
- `dummy` archetype and `radial` attack style not covered by autoplay
