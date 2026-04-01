# Event Bus System Audit (2026-03-30)

## Overview

事件总线 `event.Bus` 是一个同步发布/订阅中心，位于 `internal/core/event/`，用于解耦游戏系统间的通信。

**生命周期**: `Game.NewBus()` 创建 → `Switcher.EventBus()` 注入场景 → `subscribeBus()` 注册 → `bus.Clear()` 在场景淡出时清理。

**注意**: `event` 包内同时存在两套不相关的系统：
1. **事件总线** (`bus.go` + `typed.go` + `payloads.go`) — pub/sub 消息总线
2. **游戏内事件** (`event.go` + `handler.go`) — 波次间选择增益的玩家机制

本文档仅审查事件总线（第 1 套）。

---

## Audit Summary

| Event | Emit | Subscribe | Payload | Status | Verdict |
|-------|------|-----------|---------|--------|---------|
| `EvtTowerBuilt` | 1 (stage.go:1051) | 1 (stage.go:401) | `TowerBuiltPayload` | Active | KEEP |
| `EvtTowerUpgraded` | 1 (stage.go:927) | 1 (stage.go:406) | `TowerUpgradedPayload` | Active | KEEP |
| `EvtTowerSold` | 1 (stage.go:1078) | 1 (stage.go:409) | `TowerSoldPayload` | Active | KEEP |
| `EvtEnemyKilled` | 3 via emitKill | 1 (stage.go:440) | `EnemyKilledPayload` | Active | KEEP |
| `EvtEnemyLeaked` | 1 (stage.go:1335) | 1 (stage.go:436) | `EnemyLeakedPayload` | Active | KEEP |
| `EvtWaveStarted` | 1 (stage.go:1313) | 1 (stage.go:414) | `WaveStartedPayload` | Active | KEEP |
| `EvtWaveCleared` | 1 (stage.go:1610) | 1 (stage.go:423) | `WaveClearedPayload` | Active | KEEP |
| `EvtDamageDealt` | 0 | 0 | None | Dead | **DELETE** |
| `EvtGoldChanged` | 0 | 0 | None | Dead | **DELETE** |

---

## Detailed Analysis

### Active Events (7/9) — All KEEP

#### 1. EvtTowerBuilt

- **Emit**: `tryBuildTower()` — 建塔成功后
- **Subscribe**: session.OnTowerBuilt() + SFX(build) + tutorial("towerBuilt")
- **Payload fields**: `TowerKey` (used by session), `Cost` (available but subscribe 端未使用)
- **Assessment**: 功能完整。`Cost` 字段留作未来统计扩展（合理）。

#### 2. EvtTowerUpgraded

- **Emit**: 升级塔强度后
- **Subscribe**: SFX(upgrade)
- **Payload fields**: `TowerKey`, `Spent` — subscribe 端均未使用（仅播音效）
- **Assessment**: 功能完整。Payload 携带的信息供未来 session 统计使用。当前只需音效通知。

#### 3. EvtTowerSold

- **Emit**: 卖塔动画触发后
- **Subscribe**: SFX(towerSell)
- **Payload fields**: `TowerKey`, `Refund` — subscribe 端均未使用
- **Assessment**: 同上，功能完整。

#### 4. EvtEnemyKilled

- **Emit**: 3 个击杀路径统一走 `emitKill()` — projectile/warden/skill
- **Subscribe**: kills++, gold+=GoldValue, session.OnEnemyKilled(), tutorial, warden.OnKill()
- **Payload fields**: `IsBoss` (used), `KillerID` (available, 未在 subscribe 端使用), `GoldValue` (used)
- **Assessment**: **核心事件**，功能完整。统一了 3 条击杀路径，修复了之前战灵/技能击杀漏回调的 bug。`KillerID` 可用于未来击杀统计面板。

#### 5. EvtEnemyLeaked

- **Emit**: 敌人到达路径终点时
- **Subscribe**: session.OnEnemyLeaked() + SFX(enemyLeak)
- **Payload fields**: Empty struct
- **Assessment**: 功能完整。空 payload 合理——泄漏只需通知，不需额外数据。

#### 6. EvtWaveStarted

- **Emit**: spawner 波次号递增时
- **Subscribe**: session.OnWaveStart() + SFX(waveStart/bossEnter) + waveAnnounce UI + tutorial
- **Payload fields**: `Wave` (used), `IsBoss` (used)
- **Assessment**: 功能完整，所有字段均被使用。

#### 7. EvtWaveCleared

- **Emit**: 波次清除后（利息计算和完美判定之后）
- **Subscribe**: warden.OnWaveClear() + SFX(waveClearPerfect/waveClear) + tutorial
- **Payload fields**: `Wave` (available, 未直接使用), `Perfect` (used)
- **Assessment**: 功能完整。`Wave` 字段供未来波次统计使用。

### Dead Events (2/9) — DELETE

#### 8. EvtDamageDealt

- **Status**: 常量定义于 `bus.go:17`，**无 Emit、无 Subscribe、无 Payload 类型**
- **Origin**: 迁移设计时规划的高频事件，但按 MEMORY.md 记载的设计决策"高频事件保留直接回调"，伤害事件走的是 `HitCallback` 直接回调而非 Bus
- **Verdict**: **删除**。设计上已明确排除，留着只是噪声。

#### 9. EvtGoldChanged

- **Status**: 常量定义于 `bus.go:18`，**无 Emit、无 Subscribe、无 Payload 类型**
- **Origin**: 迁移设计时规划，但金币变更直接在 `s.gold += p.GoldValue`（EvtEnemyKilled handler）和其他直接赋值中完成
- **Verdict**: **删除**。金币变更是高频操作（每次击杀），走 Bus 不合理。当前直接赋值模式正确。

---

## Integration Verification

### All Emit→Subscribe chains verified

| Chain | Emit Site | Subscribe Handler | Data Flow |
|-------|-----------|-------------------|-----------|
| Tower build | tryBuildTower → bus.Emit | session + SFX + tutorial | gold deducted before emit |
| Tower upgrade | strength upgrade → bus.Emit | SFX | gold deducted before emit |
| Tower sell | trySellTower → bus.Emit | SFX | gold refunded + sell anim before emit |
| Enemy killed | emitKill (3 callers) → bus.Emit | kills/gold/session/tutorial/warden | gold calc in emitKill |
| Enemy leaked | moveEnemies → bus.Emit | session + SFX | lives decremented before emit |
| Wave started | tickSpawning → bus.Emit | session/SFX/announce/tutorial | wave number valid |
| Wave cleared | tickWaveClear → bus.Emit | warden/SFX/tutorial | perfect flag computed before emit |

### Architecture Observations

1. **所有生产代码的 Emit 和 Subscribe 集中在 `stage.go` 一个文件**。这不是问题——塔防游戏的核心循环本身就集中在 stage 场景。
2. **Bus 是跨场景的**（挂在 Game 上），但只有 StageScene 使用它。其他场景（Title/Select/Result）无需事件通信。
3. **`bus.Clear()` 在场景切换时正确调用**，防止跨场景的 stale subscriptions。
4. **`emitKill()` 统一 3 条击杀路径**是关键架构——确保 session/tutorial/warden 回调不会因路径不同而遗漏。

---

## Changes Made

### Deleted
- `EvtDamageDealt` constant — dead code, high-frequency events use direct callbacks
- `EvtGoldChanged` constant — dead code, gold mutations are direct assignments

### No changes needed for active events
- All 7 active events have complete Emit→Subscribe→Handler chains
- All payloads are correctly structured and type-safe via `OnTyped[T]`
- Test coverage exists in `tests/core/event_typed_test.go` (7 test cases covering all payload types)
