# 03 敌人系统审核指导

## 审核目标

验证敌人原型配置、**能力装配系统**、行为实现、Boss 机制、buff 模板、生命周期管理。
怪物能力系统是近期重构的重点，详见 `13-ability-interaction.md`。

## 必读文件

| 文件 | 读取内容 |
|------|---------|
| `config/enemies/enemies-core.json` | 敌人原型配置（hpScale/speedScale/radius 等） |
| `config/enemies/boss-templates.json` | Boss 行为模板 |
| `internal/core/enemy/enemy.go` | Enemy struct、TickStatusEffects()、常量 |
| `internal/core/enemy/pool.go` | Spawn()、Kill()、FinishDying()、KillImmediate() |
| `internal/core/enemy/spawner.go` | 波次生成逻辑、原型选择、Boss 生成 |
| `internal/core/enemy/spawn_config.go` | SpawnConfig struct |
| `config/enemies/abilities.json` | 15 种怪物能力配置（7 类别） |
| `internal/core/enemy/behaviors.go` | TickBehaviors（healer/stealth/buffer + 能力 tick：冲刺/相位/削强/净化） |
| `internal/core/enemy/boss_behavior.go` | Boss 专属行为（**当前全是 stub/no-op，待迁移到能力系统**） |
| `internal/core/enemy/buff_templates.go` | 8 buff 模板（reflect/revive 已迁移到能力系统） |
| `internal/core/enemy/movement.go` | 路径移动 + 眩晕/定身/冲刺/SpeedBuff 处理 |

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

### C. 能力装配系统（新）

| # | 检查 | 方法 | 预期 |
|---|------|------|------|
| C1 | 能力配置表 | 读 abilities.json | 15 种能力，7 类别(defense/passive/resist/movement/offense/support/death) |
| C2 | 原型→能力映射 | 读 enemies-core.json | 每种原型的 abilities 字段引用有效的能力 type |
| C3 | SpawnConfig 拷贝 | 读 pool.go Spawn | 所有 15 种能力字段从 SpawnConfig 正确拷贝到 Enemy |
| C4 | AbilityIDs 拷贝 | 读 pool.go Spawn | AbilityIDs []string 从 SpawnConfig 拷贝（渲染依赖） |
| C5 | 池复用重置 | 读 pool.go Spawn | PhaseActive/DashActiveT/StrDrainActiveT/PurgeTimer/AbilitySilenced 全部清零 |
| C6 | 沉默响应 | 读 behaviors.go + apply_hit.go | silenceable=true 的能力检查 AbilitySilenced；purge 不检查 |

### D. 旧行为系统（Behavior 字段）

| # | 检查 | 方法 | 预期 |
|---|------|------|------|
| D1 | healer 治疗 | 读 behaviors.go tickHealer | 范围内友方按 MaxHP% 回复，被沉默时停止 |
| D2 | stealth 揭隐 | 读 behaviors.go tickStealth | 计时到期或被击中时破隐 |
| D3 | buffer 加速 | 读 behaviors.go tickBuffer | SpeedBuff 每帧清零→buffer 重写，被沉默时停止 |
| D4 | splitter 分裂 | 读 pool.go Kill | SplitCount > 0 时 OnSplitterDeath，子体不递归 |
| D5 | deathSpawn 召唤 | 读 pool.go Kill | DeathSpawnCount > 0 时生成 DeathSpawnArch 原型 |
| D6 | berserk 狂暴 | 读 behaviors.go TickBerserk | 低 HP 时永久提升 BaseSpeed（一次性） |
| D7 | teleporter 传送 | 读 behaviors.go TickTeleport | 周期性位置跳跃，跳过 N 个路径段 |

### E. Boss 行为

| # | 检查 | 方法 | 预期 |
|---|------|------|------|
| E1 | Boss HP 递增 | 读 spawner.go | `HpScale *= 8.0 + wave`（wave5=13x, wave10=18x） |
| E2 | boss-templates 配置 | 读 boss-templates.json | 7 种 Boss 模板（phase/teleport/spawnMinions/reflect/rotateWeakness/goldSteal/aura） |
| E3 | boss_behavior.go 状态 | 读 boss_behavior.go | **当前全是 stub/no-op**：FullBossState 返回 nil，所有 Tick 函数为空 |
| E4 | FullBossState | 读 spawner.go | Boss 出生时调用，但因为是 no-op，**Boss 行为实际未激活** |

### F. Buff 模板

| # | 检查 | 方法 | 预期 |
|---|------|------|------|
| F1 | ApplyBuffTemplate switch | 读 buff_templates.go | 8 种模板（berserk/regen/healAura/speedAura/damageReduce/deathSplit/deathSlow/spawnMinions），reflect/revive 已迁移 |
| F2 | 波次注入 | 读 spawner.go applyWaveBuffs | wave≥6 才注入，Boss/Elite 跳过，30% 概率 |
| F3 | 遥测记录 | 读 buff_templates.go | `tel.T.Record("enemy_template", templateID)` |

### G. 死亡流程

| # | 检查 | 方法 | 预期 |
|---|------|------|------|
| G1 | Kill → DyingTimer | 读 pool.go Kill | 普通 0.3s，Boss 0.5s |
| G2 | Kill → 分裂/召唤 | 读 pool.go Kill | SplitCount > 0 触发 OnSplitterDeath；DeathSpawnCount > 0 触发 deathSpawn |
| G3 | Count 递减时机 | 读 pool.go Kill | Kill 时 Count--（不是 FinishDying 时） |
| G4 | KillImmediate | 读 pool.go KillImmediate | 泄漏用，无 dying 动画，立即 Active=false |
| G5 | IsDying 跳过 | 全局搜索 IsDying() | targeting/collision/movement 应跳过 dying 敌人 |

## 跨系统关联

- Spawn ← spawner.Tick ← stage.updatePlaying step 1
- Kill → emitKill → EventBus(EvtEnemyKilled) → session.OnEnemyKilled → gold 奖励
- 泄漏 → lives-- → EventBus(EvtEnemyLeaked) → session.OnEnemyLeaked
- buff_templates ← applyWaveBuffs ← spawner.Tick（出怪时注入）
