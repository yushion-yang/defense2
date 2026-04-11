# 08 事件系统与生命周期审核指导

## 审核目标

验证事件总线的发布/订阅完整性、关键数据流无断裂、生命周期管理正确。

## 必读文件

| 文件 | 读取内容 |
|------|---------|
| `internal/core/event/bus.go` | Bus struct、事件常量、On/Emit/Clear |
| `internal/core/event/payloads.go` | 事件载荷类型 |
| `internal/core/event/typed.go` | OnTyped 泛型订阅 |
| `internal/scene/stage.go` | 搜索 `bus.Emit` 和 `event.OnTyped` 的所有调用点 |

## 检查项

### A. 事件定义完整性

| # | 事件 | Emit 位置 | 订阅者 |
|---|------|---------|--------|
| A1 | EvtTowerBuilt | tryPlaceTower 成功后 | session.Stats.TowersBuilt++ |
| A2 | EvtTowerUpgraded | AddAbility 成功后 | （验证是否有订阅者） |
| A3 | EvtTowerSold | trySellTower 成功后 | session.Stats.TowersSold++ |
| A4 | EvtEnemyKilled | emitKill() | session.OnEnemyKilled, gold+=, 战灵 OnKill |
| A5 | EvtEnemyLeaked | 敌人到终点 | session.OnEnemyLeaked, lives-- 已在 Emit 前完成 |
| A6 | EvtWaveStarted | onWaveTransition | waveLivesSnapshot, waveAnnounce, tutorial |
| A7 | EvtWaveCleared | onWaveTransition | wavesCleared++, 能力解锁, 奖金 |

### B. 击杀事件三路径

| # | 路径 | 位置 | 验证 |
|---|------|------|------|
| B1 | 弹射物击杀 | TickProjectileHits → kill → emitKill | emitKill 调用 bus.Emit(EvtEnemyKilled) |
| B2 | 战灵击杀 | warden.Tick → OnKill → emitKill | TickContext.OnKill 绑定到 emitKill |
| B3 | 技能击杀 | skill.Tick → onKill → emitKill | 验证是否正确绑定 |

### C. 波次变化事件

| # | 检查 | 方法 | 预期 |
|---|------|------|------|
| C1 | 自动开波路径 | 读 spawner.Tick → wave 变化 | stage.updatePlaying 检测 prevWave < spawner.Wave → onWaveTransition |
| C2 | 手动开波路径 | 读 tryStartWave | prevWave 记录 → StartNextWave → onWaveTransition(prevWave) |
| C3 | autoplay 开波路径 | 读 executeAutoPlayAction APActionStartWave | 同手动开波，包裹 prevWave 检测 |
| C4 | onWaveTransition | 读完整函数 | WaveStarted 事件 + WaveCleared 奖金 + 能力解锁 |

### D. 生命周期管理

| # | 检查 | 方法 | 预期 |
|---|------|------|------|
| D1 | bus.Clear 时机 | 搜索 bus.Clear 调用 | 场景切换淡出完成时调用，清除旧订阅 |
| D2 | 订阅不泄漏 | 读 stage.go 订阅注册 | 所有 OnTyped 在 subscribeEvents 中，不在 Update 循环内重复注册 |
| D3 | Bus 实例共享 | 读 game.go | Bus 在 Game struct 上，通过 Switcher.EventBus() 注入 |

### E. 关键数据流验证

**伤害→击杀→奖金 完整链**：
```
tower.Fire → projectile.Hit → ApplyHit → ApplyDamage → Killed=true
→ enemies.Kill(e) → emitKill(boss, killerID)
→ bus.Emit(EvtEnemyKilled, {GoldValue, IsBoss})
→ stage handler: kills++, gold += GoldValue, session.OnEnemyKilled
```

验证：每个箭头处是否有数据丢失。特别关注 `GoldValue` 的来源（应从 economy.KillGold 计算）。

**波次清除→能力解锁 完整链**：
```
spawner.WaveActive=false → prevWave 检测 → onWaveTransition(prevWave)
→ bus.Emit(EvtWaveCleared) → handler: wavesCleared++
→ tower.UnlockedSlots(wavesCleared) → RollAndCachePendingChoices
```

验证：手动开波和自动开波两条路径是否都正确触发。

## 跨系统关联

- EventBus 实例在 Game 上，跨场景共享
- stage.subscribeEvents 在首次 Update 时注册（避免被场景切换的 Clear 清掉）
- 击杀 GoldValue ← economy.KillGold × rewardScale × archetype.RewardScale
