# 03 敌人系统审核指导

## 审核目标

验证敌人原型配置、行为实现、Boss 机制、buff 模板、生命周期管理。

## 必读文件

| 文件 | 读取内容 |
|------|---------|
| `config/enemies/enemies-core.json` | 敌人原型配置（hpScale/speedScale/radius 等） |
| `config/enemies/boss-templates.json` | Boss 行为模板 |
| `internal/core/enemy/enemy.go` | Enemy struct、TickStatusEffects()、常量 |
| `internal/core/enemy/pool.go` | Spawn()、Kill()、FinishDying()、KillImmediate() |
| `internal/core/enemy/spawner.go` | 波次生成逻辑、原型选择、Boss 生成 |
| `internal/core/enemy/spawn_config.go` | SpawnConfig struct |
| `internal/core/enemy/behaviors.go` | 特殊行为（berserk/regen/split/healer/teleport） |
| `internal/core/enemy/boss_behavior.go` | Boss 专属行为 |
| `internal/core/enemy/buff_templates.go` | 14+ buff 模板 |
| `internal/core/enemy/movement.go` | 路径移动 + 眩晕处理 |
| `internal/core/enemy/flying.go` | 飞行路径 |

## 检查项

### A. 原型配置

| # | 检查 | 方法 | 预期 |
|---|------|------|------|
| A1 | 每个原型有 label | 读 enemies-core.json | 非空 |
| A2 | hpScale > 0 | 读 enemies-core.json | 所有原型 > 0.1 |
| A3 | speedScale ≥ 0 | 读 enemies-core.json | 不为负 |
| A4 | rewardScale ≥ 0 | 读 enemies-core.json | 不为负 |
| A5 | runner speedScale > 1 | 读 enemies-core.json | runner 应该比 normal 快 |
| A6 | tank hpScale > 1 | 读 enemies-core.json | tank 应该比 normal 肉 |

### B. Spawn 流程

| # | 检查 | 方法 | 预期 |
|---|------|------|------|
| B1 | HP 缩放 | 读 pool.go Spawn | `hp = baseHP * cfg.HpScale` |
| B2 | Speed 缩放 | 读 pool.go Spawn | `speed = baseSpeed * cfg.SpeedScale` |
| B3 | 字段全量重置 | 读 pool.go Spawn | HitFlash/Age/SlowTimer/StunTimer/BleedTimer/BurnTimer 等全部归零 |
| B4 | 池复用安全 | 读 pool.go Spawn | 旧 enemy 的所有状态不会残留到新 enemy |
| B5 | Boss 标记 | 读 spawner.go | bossCfg.Boss = true，HpScale 递增 |

### C. 特殊行为

| # | 检查 | 方法 | 预期 |
|---|------|------|------|
| C1 | splitter 分裂 | 读 pool.go Kill | SplitCount > 0 时生成子怪 |
| C2 | 子怪路径继承 | 读 SpawnSplitChildren | 子怪从父怪当前路径位置继续 |
| C3 | stealth 揭隐 | 搜索 stealth 相关逻辑 | AoE/splash 应能命中隐身怪 |
| C4 | healer 治疗 | 读 behaviors.go | 范围内友方 HP 回复 |
| C5 | teleporter 传送 | 读 behaviors.go | 周期性位置跳跃 |
| C6 | berserk 狂暴 | 读 buff_templates.go | 低 HP 时攻速/移速提升 |
| C7 | 复活机制 | 读 pool.go Kill | ReviveHPPercent > 0 时首次死亡恢复 HP |

### D. Boss 行为

| # | 检查 | 方法 | 预期 |
|---|------|------|------|
| D1 | Boss HP 递增 | 读 spawner.go | `HpScale *= 8.0 + wave`（wave5=13x, wave10=18x） |
| D2 | boss-templates 完整 | 读 boss-templates.json + boss_behavior.go | 每种 Boss 模板有对应行为实现 |
| D3 | FullBossState | 读 spawner.go | Boss 出生时调用 FullBossState(wave) |

### E. Buff 模板

| # | 检查 | 方法 | 预期 |
|---|------|------|------|
| E1 | ApplyBuffTemplate switch | 读 buff_templates.go | 每种模板 ID 有对应 case |
| E2 | 波次注入 | 读 spawner.go applyWaveBuffs | wave≥6 才注入，Boss/Elite 跳过，30% 概率 |
| E3 | 遥测记录 | 读 buff_templates.go | `tel.T.Record("enemy_template", templateID)` |

### F. 死亡流程

| # | 检查 | 方法 | 预期 |
|---|------|------|------|
| F1 | Kill → DyingTimer | 读 pool.go Kill | 普通 0.3s，Boss 0.5s |
| F2 | Count 递减时机 | 读 pool.go Kill | Kill 时 Count--（不是 FinishDying 时） |
| F3 | KillImmediate | 读 pool.go KillImmediate | 泄漏用，无 dying 动画，立即 Active=false |
| F4 | IsDying 跳过 | 全局搜索 IsDying() | targeting/collision/movement 应跳过 dying 敌人 |

## 跨系统关联

- Spawn ← spawner.Update ← stage.updatePlaying step 1
- Kill → emitKill → EventBus(EvtEnemyKilled) → session.OnEnemyKilled → gold 奖励
- 泄漏 → lives-- → EventBus(EvtEnemyLeaked) → session.OnEnemyLeaked
- buff_templates ← applyWaveBuffs ← spawner.Update（出怪时注入）
