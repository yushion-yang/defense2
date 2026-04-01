# 05 战灵系统审核指导

## 审核目标

验证 5 种战灵的行为实现、成长系统、伤害路径、运动逻辑。

## 必读文件

| 文件 | 读取内容 |
|------|---------|
| `config/wardens/wardens.json` | 战灵配置（damage/attackInterval/range/moveSpeed/growth） |
| `internal/core/warden/warden.go` | Warden struct、Behavior 接口、TickContext |
| `internal/core/warden/state.go` | WardenState 基座、MoveOrbit、BasicAttack、ApplyDamage |
| `internal/core/warden/types/prince.go` | Prince（近战跳跃） |
| `internal/core/warden/types/core_mech.go` | Core（AoE） |
| `internal/core/warden/types/chain.go` | Chain（连锁） |
| `internal/core/warden/types/skystrike.go` | Skystrike（远程） |
| `internal/core/warden/types/envoy.go` | Envoy（辅助） |

## 检查项

### A. 配置完整性

| # | 检查 | 方法 | 预期 |
|---|------|------|------|
| A1 | 5 种战灵配置 | 读 wardens.json | prince/core/chain/skystrike/envoy 各有配置 |
| A2 | damage > 0 | 读 wardens.json | 所有战灵基础伤害 > 0 |
| A3 | attackInterval > 0 | 读 wardens.json | 不为 0（否则除零） |
| A4 | GrowthOnKill/WaveClear | 读 wardens.json | 成长参数 ≥ 0 |

### B. 行为实现

| # | 检查 | 方法 | 预期 |
|---|------|------|------|
| B1 | prince Tick | 读 prince.go | 近战跳跃：idle→dash→attack→return 状态机 |
| B2 | core Tick | 读 core_mech.go | AoE 攻击逻辑 |
| B3 | chain Tick | 读 chain.go | 连锁攻击 + ChainLinks 记录 |
| B4 | skystrike Tick | 读 skystrike.go | 远程打击逻辑 |
| B5 | envoy Tick | 读 envoy.go | 辅助增益逻辑 |
| B6 | 行为注册 | 各 types/ 文件的 init() | 5 种都调用 RegisterBehavior |

### C. 伤害路径

| # | 检查 | 方法 | 预期 |
|---|------|------|------|
| C1 | ApplyDamage 走管线 | 读 state.go ApplyDamage | 调用 combat.ProcessDamage |
| C2 | 击杀回调 | 读 ApplyDamage | r.Killed → ctx.OnKill(e) |
| C3 | OnKill → emitKill | 读 stage.go 战灵 TickContext 构建 | OnKill 内调用 emitKill("warden") |

### D. 运动系统

| # | 检查 | 方法 | 预期 |
|---|------|------|------|
| D1 | 轨道运动 | 读 state.go MoveOrbit | 角速度合理，不超出地图边界 |
| D2 | 初始位置 | 读 state.go | 初始 (0,0)，首次 MoveOrbit 时 teleport 到轨道位置 |
| D3 | 大地图覆盖 | 读 MoveOrbit | 轨道中心是否随相机/敌群移动 |
| D4 | TrailHistory | 读 RecordTrail | 8 帧环形缓冲，在 MoveOrbit/Wander 尾部调用 |

## 跨系统关联

- 战灵选择 → stage.activateWarden → wardenReady=true
- Tick ← stage.updatePlaying step 4 → TickContext{Enemies,Towers,DT,OnKill,OnFire}
- 成长 ← EventBus(EvtEnemyKilled/EvtWaveCleared) → GrowthOnKill/GrowthOnWaveClear
- ApplyDamage → combat.ProcessDamage → 遥测 pipeline/damage_type
