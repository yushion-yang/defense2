# Release Strategy Design

Date: 2026-04-12

## Overview

Defense2 is a Go/Ebitengine tower defense game approaching feature completeness. This document outlines the phased release strategy across 4 platforms with a hybrid monetization model.

## Current State

### Ready
- Core gameplay: 8 maps, 18 enemy archetypes, 32 abilities, 5 wardens, 4 game modes
- Rendering: HiDPI pipeline, 9 shaders, 105 VFX, particle system
- Audio: 131 WAV (SFX + 3 BGM)
- Persistence: high scores, unlocks, 15 achievements, settings, tutorial
- Mascot system with 110+ dialogue lines
- Desktop + WASM builds working
- Autoplay regression testing (68 scenarios, 26 anomaly rules)

### P0 Blockers (Must Fix Before Any Release)
1. **Test failures**: 4 contract tests fail, core tests don't compile (API signature drift)
2. **Dead-code abilities**: 5 abilities selectable but non-functional (killUpgrade, waveScale, neighborBoost, elementSwitch fire/lightning, periodicCast buff)
3. **Railgun sprite missing**: renders as sentinel fallback
4. **Memory leak**: `scaling.go` global maps not cleaned between games
5. **BuffList HUD regression**: many buffs no longer display post-refactor

### P1 Gaps (Before Mobile/Desktop Release)
1. No ad SDK or IAP integration
2. No Android build pipeline (APK/AAB)
3. No crash reporting (Sentry/Crashlytics)
4. No analytics/telemetry (Firebase)
5. No privacy policy / GDPR consent
6. No app icon or store materials
7. No i18n — hardcoded Chinese, English fragments remain
8. Performance optimization: 12-item plan documented but not implemented
9. 28 hardcoded colors to extract to theme

## Release Strategy: Phased Multi-Platform

```
Phase 1 (Web)  →  Phase 2 (Android)  →  Phase 3 (Steam)  →  Phase 4 (iOS, optional)
```

### Monetization Model (Hybrid)
| Platform | Model | Details |
|----------|-------|---------|
| Web (itch.io) | Free | No ads, optional tip jar |
| Android (Google Play) | Free + Ads | Interstitial (between levels) + Rewarded video (revive/double reward) |
| Steam | Paid (¥18-28) | No ads, includes all content |
| iOS | Free + Ads | Same as Android (if pursued) |

---

## Phase 1: Bug Fix + Web Launch

**Goal**: Fix all P0 code issues, deploy to itch.io for early feedback.

**Duration**: 1-2 weeks

### Tasks

#### 1.1 Fix Test Failures
- Update `pipeline.TickProjectileHits` / `pipeline.TickTowerCombat` test signatures
- Fix 4 contract test assertions (speedPerWave, damageUp cap, flatDamage, killReward consistency)

#### 1.2 Remove Dead-Code Abilities
- Remove 5 non-functional abilities from candidate pool in config
- Or: implement them if feasible (evaluate effort per ability)

#### 1.3 Fix Railgun Sprite
- Add railgun tower sprite set to `assets/towers/railgun/`
- Or: use a valid fallback and mark as known limitation

#### 1.4 Fix Memory Leak
- Clean `scaling.go` global maps on game session reset

#### 1.5 Fix BuffList HUD Regression
- Restore buff icon display in HUD after BuffList refactor

#### 1.6 Web Deployment
- `make build-wasm` → test in browser
- Deploy to itch.io
- Write itch.io page description (Chinese + English summary)
- Share on social media / indie game communities for feedback

### Exit Criteria
- `make check-all` passes (lint + all tests green)
- Autoplay sweep: 0 anomalies on all 68 scenarios
- Web version playable end-to-end in Chrome/Firefox/Safari

---

## Phase 2: Android Release (Main Revenue)

**Goal**: Ship on Google Play with ad monetization.

**Duration**: 3-4 weeks

### Tasks

#### 2.1 Android Build Pipeline
- Set up Gradle project scaffold for `ebitenmobile`
- Configure APK/AAB signing
- Add `make android` Makefile target
- Test on physical device (touch input, performance)

#### 2.2 Ad SDK Integration (AdMob)
- Add AdMob SDK via Go bindings or JNI bridge
- Design ad trigger points:
  - **Interstitial**: between levels (after result screen, before next map select)
  - **Rewarded video**: on defeat (revive with 5 lives) + after victory (double coin reward)
- Implement ad abstraction layer (interface) for platform-specific backends
- Web/Steam builds: no-op ad implementation

#### 2.3 i18n Foundation (Chinese + English)
- Extract all hardcoded UI strings to JSON resource files
- Implement locale detection + fallback
- Translate core UI to English (menus, HUD, ability names, enemy names)
- Mascot dialogs: Chinese-only initially (too many lines to translate at once)

#### 2.4 Crash Reporting + Analytics
- Integrate Sentry (Go SDK) for crash reports
- Integrate Firebase Analytics for:
  - Session duration, retention
  - Level completion rate per map/difficulty
  - Tower/ability usage distribution
  - Ad impression/click rates

#### 2.5 Privacy & Compliance
- Write privacy policy (data collection: analytics + ad identifiers)
- Add GDPR consent dialog on first launch (EU users)
- Add COPPA compliance note (if targeting under 13)

#### 2.6 App Store Materials
- Design app icon (1024x1024)
- Capture 4-6 screenshots per form factor (phone + tablet)
- Write store description (Chinese + English)
- Create feature graphic (1024x500)
- Set content rating questionnaire

#### 2.7 Performance Optimization
- Execute Phase A from performance plan (7 zero-alloc items, low risk)
- Execute Phase B selectively (ActiveList, viewport culling)
- Target: stable 60fps on mid-range Android (Snapdragon 6 series)

### Exit Criteria
- APK installs and runs on 3+ Android devices
- Ad flow works: interstitial shows between levels, rewarded video grants reward
- Crash-free rate > 99%
- Store listing approved on Google Play

---

## Phase 3: Steam Release (Premium)

**Goal**: Ship on Steam as paid title.

**Duration**: 2-3 weeks

### Tasks

#### 3.1 Steamworks Integration
- Register on Steamworks ($100 fee)
- Integrate Steam SDK (achievements → map to existing 15 achievements)
- Cloud saves via Steam Cloud (map persistence data)

#### 3.2 Desktop Polish
- Keyboard shortcuts review (already partial)
- Window resize / fullscreen toggle
- Controller support (optional, low priority)

#### 3.3 Store Page
- Steam capsule images (header 460x215, small capsule 231x87, hero 3840x1240)
- 4-6 screenshots with captions
- Short description + about section
- Tags: Tower Defense, Strategy, Indie, Anime
- Release trailer (30-60s gameplay capture)

#### 3.4 Pricing
- Base price: ¥18-28 (comparable to indie TD games on Steam)
- Launch discount: 10-15% for first week
- No DLC planned initially

### Exit Criteria
- Steam build review approved
- Achievement integration working
- Store page live with all required assets

---

## Phase 4: iOS (Optional)

**Prerequisite**: Android version has positive retention metrics (D1 > 30%, D7 > 10%).

### Tasks
- Apple Developer Program ($99/year)
- Xcode project scaffold for `ebitenmobile`
- AdMob iOS integration
- App Store Connect listing
- App Review compliance (more stringent than Google Play)

### Decision Point
Evaluate after Android has been live for 1-2 months. If DAU < 100, defer iOS indefinitely.

---

## Cross-Cutting Concerns (All Phases)

### Feature Flags / Build Tags
Use Go build tags to separate platform concerns:
```go
//go:build android
//go:build steam
//go:build web
```
- Ad code: only in android/ios builds
- Steam SDK: only in steam build
- Analytics: all builds except web

### Testing Strategy
- Continue autoplay sweep for balance regression
- Add integration tests for ad trigger points (mock ad SDK)
- Manual QA checklist per platform before each release

### Localization Roadmap
| Phase | Languages |
|-------|-----------|
| Phase 1 | Chinese only |
| Phase 2 | Chinese + English |
| Phase 3+ | Japanese, Korean (if demand exists) |

### Art Assets Still Needed
| Asset | Priority | Notes |
|-------|----------|-------|
| App icon (1024x1024) | P0 for Phase 2 | Required for all app stores |
| Railgun tower sprite | P0 for Phase 1 | Currently renders wrong |
| Store screenshots | P0 for Phase 2/3 | Per platform requirements |
| Multi-frame warden PNGs | P2 | Animator framework ready, single-frame works |
| Release trailer | P1 for Phase 3 | 30-60s gameplay capture |

---

## Risk Assessment

| Risk | Impact | Mitigation |
|------|--------|------------|
| AdMob Go/Ebitengine integration complexity | High | Research existing examples; fallback to JNI bridge |
| Steam SDK Go bindings maturity | Medium | Use steamworks-go or CGo wrapper |
| i18n effort explosion | Medium | Start with UI strings only; mascot dialogs later |
| Low user acquisition on itch.io | Low | Web is for feedback, not revenue |
| Apple rejection on first submission | Medium | Study rejection reasons beforehand; keep UI clean |

## Success Metrics

| Metric | Phase 1 | Phase 2 | Phase 3 |
|--------|---------|---------|---------|
| Players | 100+ plays | 1K+ installs | 100+ sales |
| Revenue | $0 | $50+/month (ads) | $500+ first month |
| Crash-free | N/A | >99% | >99.5% |
| Rating | N/A | >4.0 stars | >80% positive |
