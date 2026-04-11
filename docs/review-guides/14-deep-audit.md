# 14 深度审查指导 — 基于历史 Bug 模式的系统性排查

> 从 40+ 个已知 bug 和优化项中提炼出 7 类模式，衍生出 50+ 个审查项。
> AI 在新 session 中按本文档逐项审核，输出发现的问题（P0-P3 分级）。

## 使用方法

```
请阅读 docs/review-guides/14-deep-audit.md，按照 A-G 七个类别逐项审核源码，
每个检查项给出 PASS / FAIL / WARN，FAIL 和 WARN 项附上具体文件:行号和问题描述。
```

---

## A. 功能接入完整性 — "写了但没调用"

> 历史教训：金灵没加强度、串联没加属性、散弹数量不增长

### 必读文件

| 文件 | 关注点 |
|------|--------|
| `internal/core/tower/abilities/scaling.go` | 6 个 scaling 能力是否真正生效 |
| `internal/core/tower/abilities/config_ability.go` | OnHit/OnTick 返回值是否被使用 |
| `internal/core/warden/types/*.go` | 5 种战灵特殊能力是否产生实际效果 |
| `internal/render/vfx/*.go` | VFX 函数是否被 draw_*.go 调用 |

### 检查项

| # | 检查 | 方法 | 预期 |
|---|------|------|------|
| A1 | killUpgrade 能力是否生效 | 读 scaling.go `killUpgrade` 的 OnHit，检查返回值是否被使用 | 计算的 bonusPerStack 必须写入塔属性，不能赋给 `_` |
| A2 | waveScale 能力是否生效 | 读 scaling.go `waveScale` 的 OnTick，检查 bonus 是否被应用 | 同上 |
| A3 | neighborBoost 能力是否生效 | 读 scaling.go `neighborBoost` OnTick，检查 boostRatio 去向 | 必须影响塔的 Damage/Range/Speed |
| A4 | elementSwitch fire/lightning 是否生效 | 读 scaling.go elementSwitch 各 case | fire 应加伤，lightning 应加攻速 |
| A5 | periodicCast buff AoE 是否生效 | 读 scaling.go periodicCast case 2 | 应给范围内塔施加 buff |
| A6 | 每个 VFX 函数有调用者 | `grep -r "vfx.DrawXxx" internal/render/draw_*.go` 对比 `vfx/*.go` 导出函数 | 每个导出函数至少 1 个调用点 |
| A7 | 5 种战灵 buff/damage 生效 | 读各 warden type 的 Tick，验证 ApplyDamage/ApplyBuff 被调用且参数正确 | prince 火球有伤害，envoy 有强度 buff，chain 有链网络，core 有 AoE，skystrike 有减速 |
| A8 | 所有 event bus 事件有订阅者 | grep `Emit(` 和 `OnTyped(` 配对 | 每个 Emit 有对应 OnTyped |
| A9 | OnSplit / OnDeathSpawn 回调已注册 | 读 stage.go 中 `enemies.OnSplit` 和 `enemies.OnDeathSpawn` 赋值 | 两个回调都不为 nil |

---

## B. 数值一致性 — "配置不匹配"

> 历史教训：攻速 buff 实际加射程、能力描述显示 0+1%=2%、散弹子弹数不增长

### 必读文件

| 文件 | 关注点 |
|------|--------|
| `internal/core/tower/tower.go` | RecalcStats 公式 |
| `internal/core/tower/branch.go` | 分支属性修改 |
| `internal/core/strength/strength.go` | Effective() 计算 |
| `internal/core/warden/state.go` | ApplyStrength 缩放 |
| `config/abilities/abilities.json` | 能力数值定义 |
| `config/towers/towers.json` | 塔 Base/Potential 定义 |

### 检查项

| # | 检查 | 方法 | 预期 |
|---|------|------|------|
| B1 | RecalcStats 三属性公式一致 | 读 tower.go RecalcStats | Damage/Range/AttackSpeed 都用 `Base + Potential * ratio` |
| B2 | Mods 正确应用 | 读 RecalcStats 中 PctDamage/PctSpeed/PctRange 应用处 | 乘法应用在 base 之后，不能先乘后加 |
| B3 | DPS() 包含所有加成 | 读 tower.go DPS() | 应包含 DamageAmp，否则 HUD 和 AI 决策用错误值 |
| B4 | branch.go 修改不被覆盖 | 读 branch.go ApplyBranch | 修改后必须调 RecalcStats 或在 RecalcStats 中考虑分支 |
| B5 | 卖塔退款比例正确 | 读 stage.go 卖塔逻辑 | 退款 = 建造成本 * sellRatio（从 balance.json 读） |
| B6 | 能力 CalcScale(str) 使用正确 | 全局搜索 CalcScale 调用 | 每次调用传入的 str 是 Effective() 而非 Permanent |
| B7 | 战灵只缩放 Damage 是否符合设计 | 读 warden/state.go ApplyStrength | 确认 Range/AttackInterval 不缩放是有意为之 |
| B8 | 敌人 reward 与难度匹配 | 读 spawner.go 奖励计算 | reward * rewardScale * difficultyRewardMul 链路完整 |
| B9 | DoT DPS 从配置读取 | 读 apply_hit.go bleed/burn 赋值 | DPS 值来自能力 CalcScale，不是硬编码 |
| B10 | 减速 factor 下限生效 | 读 crowd_control.go ApplySlow | MinSpeedRatio 从 balance.json 读取并 clamp |

---

## C. 状态同步 — "视觉与数据不一致"

> 历史教训：沉默对加速 buff 无效、削强连攻速也降了、浮字免疫无说明

### 必读文件

| 文件 | 关注点 |
|------|--------|
| `internal/scene/stage.go` | 模式转换、HUD 同步 |
| `internal/scene/stage_input.go` | 输入处理、模式切换 |
| `internal/core/enemy/behaviors.go` | 行为系统状态 |
| `internal/render/draw_enemy.go` | 敌人渲染状态读取 |

### 检查项

| # | 检查 | 方法 | 预期 |
|---|------|------|------|
| C1 | modeNames 数组与枚举对齐 | 对比 stage_types.go 枚举与 stage.go modeNames | 索引、名称、数量完全一致 |
| C2 | AbilitySilenced 每帧重置 | 读 tick_abilities.go Phase 1.5 | 每帧清零后由 zone 能力重设 |
| C3 | Silenced 阻断所有应阻断的能力 | 读 behaviors.go 每个能力检查 AbilitySilenced | healer/buffer/phaseShift/strDrain 都检查 |
| C4 | Purge 清除所有负面效果 | 读 behaviors.go purge 代码 | BuffList.ClearByCategory 覆盖 CC+DoT+Debuff，ZoneDmgAccum 也清零 |
| C5 | 敌人死亡动画期间不被攻击 | 读 targeting.go | IsDying() 检查在索敌时排除 |
| C6 | 出生动画期间无敌 | 读 IsSpawning() 使用处 | 出生中敌人被 targeting 和碰撞跳过 |
| C7 | HP bar 拖尾同步 | 读 enemy.go DisplayHP 逻辑 | DisplayHP >= HP（不能低于实际 HP） |
| C8 | 速度恢复：slow 过期后速度回 BaseSpeed | 读 TickStatusEffects slow 过期处理 | slow buff 过期时 `e.Speed = e.BaseSpeed` |

---

## D. 生命周期 — "初始化/清理遗漏"

> 历史教训：战灵卡原点、白圈精灵、特效不跟随移动

### 必读文件

| 文件 | 关注点 |
|------|--------|
| `internal/core/enemy/pool.go` | Spawn/Kill/ClearAll |
| `internal/core/tower/pool.go` | 塔的 Spawn/Remove |
| `internal/core/projectile/pool.go` | 弹射物生命周期 |
| `internal/core/warden/state.go` | 战灵初始化 |

### 检查项

| # | 检查 | 方法 | 预期 |
|---|------|------|------|
| D1 | Enemy Spawn 清零所有字段 | 读 pool.go Spawn 的 `*e = Enemy{}` | 复用槽位前所有字段清零 |
| D2 | Enemy Kill 不遗漏 | 读 pool.go Kill | 分裂/死亡召唤在 dying 标记前执行 |
| D3 | Tower Remove 清理 scaling 状态 | 读 tower/pool.go Remove + scaling.go ClearTowerScalingState | 卖塔时清除 6 个全局 map 中的对应 key |
| D4 | 弹射物 TTL 到期清理 | 读 projectile/pool.go Update | 所有弹射物有 TTL，过期后 Active=false |
| D5 | 弹射物目标死亡处理 | 读 projectile pool Update | 追踪弹目标死亡后立即失活 |
| D6 | 战灵初始位置合理 | 读 warden/state.go MoveOrbit + Wander 的首帧处理 | X==0&&Y==0 时 teleport 到合理位置 |
| D7 | BuffList 在 Spawn 时初始化 | 读 enemy/pool.go Spawn | `e.Buffs = buff.NewDefaultBuffList()` 存在 |
| D8 | 场景切换时资源清理 | 读 stage.go 退出逻辑 | BGM 停止、粒子清空、对象池 ClearAll |

---

## E. HUD / 文本 — "展示不准确"

> 历史教训：英文残留、描述显示 0+1%=2%、属性颜色不对

### 必读文件

| 文件 | 关注点 |
|------|--------|
| `internal/render/hud/*.go` | 所有 HUD 面板 |
| `internal/render/floattext.go` | 浮字系统 |
| `internal/core/gamemode/*.go` | 波次消息 |
| `internal/core/combat/apply_hit.go` | 战斗浮字 |

### 检查项

| # | 检查 | 方法 | 预期 |
|---|------|------|------|
| E1 | 所有 SetFloatText 内容为中文 | grep `SetFloatText` 全局 | 无英文（MISS→闪避, CAP→上限等） |
| E2 | 所有 gamemode 消息为中文 | 读 gamemode/*.go 的 WaveClearMessage | "第N波清除! +$X" 而非 "Wave N clear!" |
| E3 | 能力描述不显示零增长 | 读 HUD 能力展示逻辑 | Potential=0 的属性不显示 "+0%" |
| E4 | 属性颜色正确 | 读 HUD 三段配色逻辑 | 100 强度=白色，>100=绿色，<100=红色 |
| E5 | 免疫浮字区分类型 | 读所有 "免疫" SetFloatText | 应区分"减速免疫"/"控制免疫"/"伤害免疫" |
| E6 | 升级菱形提示 | 读 stage.go 能力解锁提示逻辑 | 获取能力后有视觉提示用户选取 |
| E7 | 独行加成范围可见 | 读 soloBoost 能力 | HUD 或游戏中展示检查范围圈 |

---

## F. 性能 — "帧率下降"

> 历史教训：fmt.Printf 在热路径导致帧率暴跌

### 必读文件

| 文件 | 关注点 |
|------|--------|
| `internal/render/hud/*.go` | Draw() 中的 fmt.Sprintf |
| `internal/render/draw_projectile.go` | 每帧分配 |
| `internal/render/draw_warden.go` | 每帧分配 |
| `internal/core/tower/abilities/scaling.go` | 全局 map 清理 |

### 检查项

| # | 检查 | 方法 | 预期 |
|---|------|------|------|
| F1 | 无 fmt.Sprintf 在 Draw() 热路径 | grep `fmt.Sprintf` 在 `internal/render/` | 改用 strconv + 栈 buffer |
| F2 | 无 fmt.Printf 在 Update() 热路径 | grep `fmt.Printf` 在 `internal/core/` | 只允许在错误路径或初始化中 |
| F3 | 弹射物渲染无每帧分配 | 读 draw_projectile.go trail 创建 | 应预分配或复用 slice |
| F4 | 战灵 VFX 数据无每帧分配 | 读 draw_warden.go trails/fireballs 转换 | 同上 |
| F5 | scaling 全局 map 有清理 | 读 ClearTowerScalingState 调用链 | 卖塔/重置时调用 |
| F6 | collectAlive 预分配 | 读 skystrike.go collectAlive | 使用 pool-level buffer 而非 append |
| F7 | 粒子池不超限 | 读 particle.go pool 上限 | Quality 对应 maxParticles 严格执行 |

---

## G. 音频完整性

> 历史教训：SFX 被关闭后无法重新开启、造塔音效消失

### 必读文件

| 文件 | 关注点 |
|------|--------|
| `internal/audio/manager.go` | SFX 常量、播放方法 |
| `internal/scene/stage.go` | 音效触发点 |
| `config/audio/sfx.json` | 音效定义 |

### 检查项

| # | 检查 | 方法 | 预期 |
|---|------|------|------|
| G1 | 所有 SFX 常量有调用点 | grep 每个 SFX 常量在 stage*.go 中的使用 | 无孤立常量 |
| G2 | BGM 切换完整 | 读 stage.go BGM 转换逻辑 | menu→battle→boss→battle→silence→menu 完整链路 |
| G3 | 音量设置持久化 | 读 settings_persist.go | sfxEnabled 默认 true，volume 范围 [0,1] |
| G4 | 节流间隔合理 | 读所有 PlayThrottledAt 调用 | 高频音效 (fire/hit) ≤100ms，低频事件 (wave/build) 不节流 |
| G5 | 能力触发有音效 | 对比 abilities.json 中所有能力与 stage.go 音效触发 | 至少 attack 类能力有对应开火音效 |

---

## 已发现待修的问题（本次审查产出）

> 以下是编写本文档过程中已确认的问题，需后续修复：

| 优先级 | ID | 问题 | 文件 |
|--------|-----|------|------|
| P0 | A1 | 6 个 scaling 能力是 no-op（killUpgrade/waveScale/neighborBoost/elementSwitch/periodicCast buff） | scaling.go |
| P0 | C1 | modeNames 数组与枚举不对齐（缺 3 个 mode，索引错位） | stage.go:3077 |
| P1 | B4 | branch.go 属性修改被下一帧 RecalcStats 覆盖 | branch.go |
| P1 | B3 | DPS() 不含 DamageAmp，HUD 和 AI 决策用错误值 | tower.go:245 |
| P1 | E1 | 浮字 "MISS"/"CAP" 为英文 | apply_hit.go:47, damage_pipeline.go:164 |
| P1 | E2 | 所有 gamemode 波次消息为英文 | gamemode/*.go |
| P2 | F1 | HUD Draw() 中 fmt.Sprintf | hud/wave_panel.go, build_menu.go |
| P2 | F3 | draw_projectile.go 每帧 make([]TrailPt) | draw_projectile.go:20 |
