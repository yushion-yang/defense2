# 代码审查清单

> 基于已修复 bug 的模式归纳 + 代码实际扫描推导而成。
> 用途：新 session 中让 AI 逐项执行审查，发现未知 bug 和待优化项。

---

## 一、配置-代码一致性

> 历史教训：JSON tag 猜测、写了函数没调用、配置字段名不匹配。

| # | 审查项 | 执行方法 | 已知发现 |
|---|--------|---------|---------|
| 1.1 | abilities.json 每个 type 在 config_ability.go 的 OnHit/OnTick 中有对应 case | grep abilities.json 的 type → grep config_ability.go 的 case | `deathMark` 不在 abilities.json 中，代码用硬编码 fallback |
| 1.2 | config_ability.go 每个 case 在 abilities.json 中有定义 | 反向检查 | `spinAoe` 的 OnHit/OnTick 是 no-op（逻辑在 handler_spinaoe.go） |
| 1.3 | attackStyleLabel 覆盖所有攻击方式 | 对比 pool.go spriteKeyForStyle 的 style 列表 vs stage_info_vm.go attackStyleLabel | 缺 `bounce` 和 `multiTarget` 中文标签 |
| 1.4 | enemies-core.json 每个 archetype 在 spawner 中可生成 | 检查 phase 权重表是否覆盖所有 18 种 | — |
| 1.5 | balance.json 参数与 settings.json 无重复 | grep 同名参数 | killReward/sellRefundRatio 两处定义（值一致但双源） |
| 1.6 | vfx.json 每个 VFX ID 在 vfx 包中有对应函数，且在 preview registry 注册 | 三方交叉检查 | — |

## 二、死代码与未使用导出

> 历史教训：写了函数但没调用，旧系统残留。

| # | 审查项 | 执行方法 | 已知发现 |
|---|--------|---------|---------|
| 2.1 | vfx/ 包所有导出函数有外部调用者 | grep 每个 Draw/Apply 函数 | `DrawFlyingShadow` 无调用者 |
| 2.2 | combat/ 包导出函数有调用者 | grep ApplyXxx 函数 | `ApplyRoot` 零调用者（root CC 完全孤立）；`ApplyDamageUp`/`ApplyDamageDown` 无生产调用 |
| 2.3 | stage_info_vm.go 辅助函数有调用者 | grep fmtAttr/buildAttrSegs 等 | `fmtAttr`、`buildAttrSegs` 无调用者 |
| 2.4 | theme/colors.go 所有常量有引用 | grep 每个常量名 | 13 个未使用常量（EnemySwarmTri, EnemyHPBar*, TowerRangeFill, Buff*, MapDotGrid） |
| 2.5 | audio SFX 常量有 Play 调用 | grep 每个 SFX 常量 | 6 个孤立常量（SFXExplode, SFXEnemyDeathElite, SFXSpeedToggle, SFXKnockback, SFXShieldBreak, SFXHitShield） |
| 2.6 | scaling.go 中注册的能力是否有实际效果 | 检查 OnTick 是否 return 空 TickResult | `KillUpgrade`/`WaveScale`/`NeighborBoost`/火元素/雷元素 是 TODO stub，玩家获取后零效果 |

## 三、展示与 HUD

> 历史教训：描述显示错误、颜色不对、占位符暴露、英文残留。

| # | 审查项 | 执行方法 | 已知发现 |
|---|--------|---------|---------|
| 3.1 | potential=0 的能力，{s%} 显示是否有意义 | 遍历 abilities.json 中 potential=0 的条目 | `bleedDot`(1%) / `enhance`(20%) 永远固定值，描述暗示会成长 |
| 3.2 | 概率/减速类展示是否 clamp 到 100% | 检查 FormatAbilityDisplay 和 buildAbilitySegments | 已修复（isPercentCapped） |
| 3.3 | 所有用户可见文本是否中文化 | grep 英文字符串 in HUD/toast/panel | — |
| 3.4 | 100 强度时属性颜色是否全白 | 检查 scaledColor 在 scaled==potential 时的返回 | 已修复 |
| 3.5 | potential=0 的部分是否隐藏 +（0%）展示 | 检查 buildAbilitySegments 对 potential=0 的处理 | 待修复（optimization.md 已记录） |
| 3.6 | CC/DoT 命中时是否有浮字反馈 | 检查 ApplyStun/ApplySlow 成功路径 | 成功施加 stun/slow/root/bleed/burn/poison 时**无浮字** |
| 3.7 | 免疫浮字是否声明具体类型 | 检查"免疫"文本是否区分减速/控制/伤害 | 待修复（record.md 已记录） |

## 四、性能热路径

> 历史教训：config_ability.go 热路径 fmt.Printf 导致帧率暴跌。

| # | 审查项 | 执行方法 | 已知发现 |
|---|--------|---------|---------|
| 4.1 | core/ 和 render/ 的 Tick/Update/Draw 中无 fmt.Sprintf | grep fmt.Sprintf 排除 test 文件 | `towerAccKey()` 每帧每塔 8 次 Sprintf（应用 InstanceKey）；5 个 aura 每帧 Sprintf srcKey |
| 4.2 | sprite cache key 无逐帧分配 | 检查 cache.go cacheKey() | 每帧每实体 fmt.Sprintf 生成缓存键 |
| 4.3 | Draw 函数中无堆分配 slice | 检查 var xxx []Type + append 模式 | draw_enemy.go 的 `var dots []vfx.StatusDot` 逐敌逐帧分配 |
| 4.4 | 战灵 ID 键无逐帧生成 | 检查 envoy.go/chain.go 的 Sprintf | `envoy_buff_%d` 和 `chain_warden_%d` 每 tick Sprintf |
| 4.5 | Telemetry 锁竞争 | 检查 tel.T.Record() 调用频率 | 单帧内 damage_pipeline 14 次 mutex lock |

## 五、游戏逻辑完整性

> 历史教训：buff 加错属性、沉默对加速无效、削强连同攻速压制。

| # | 审查项 | 执行方法 | 已知发现 |
|---|--------|---------|---------|
| 5.1 | 沉默（silence）是否有免疫检查和视觉反馈 | 检查 config_ability.go silenceZone case | 无 IsSilenceImmune 检查，无浮字，无 VFX 覆层，无音效 |
| 5.2 | 所有 CC 类型是否有完整管线（apply → immunity check → tenacity → float text → VFX → SFX） | 逐一检查 stun/slow/root/silence | root 无调用者；silence 缺免疫/浮字/VFX/SFX |
| 5.3 | scaling.go 的 package-level map 是否在游戏结束后清理 | 检查 ResetScalingState 调用点 | 仅 test 调用，生产从不调用 → 多局游戏 map 无限增长 |
| 5.4 | 战灵 cleanup 路径是否有 nil guard | 检查 envoy cleanup 对 Strength/Buffs 的访问 | envoy.go 清理非活跃塔时无 Strength nil 检查 |
| 5.5 | tick_abilities.go 对 Buffs 是否有 nil 检查 | 与 RecalcStats 的防御模式对比 | 不一致（RecalcStats 检查了，tick_abilities 没检查） |
| 5.6 | 核心机甲战灵 AoE 模式是否有视觉区分 | 检查 draw_warden.go 对 AoE 模式的渲染 | 无区分，AoE 与单体射击视觉相同 |

## 六、硬编码魔数

> 历史教训：hardcoded 常量应从配置读取。

| # | 审查项 | 执行方法 | 已知发现 |
|---|--------|---------|---------|
| 6.1 | scaling.go 能力参数是否从配置读取 | 逐行检查数字字面量 | 14+ 处硬编码（killUpgrade 1%/stack, waveScale 5%/wave, periodicCast 8s/120px, neighborBoost 20%, elementCycle 5s, 冰减速 0.7/0.5s, 毒 2DPS 等） |
| 6.2 | combat handler fallback 速度是否从 balance.json 读取 | 检查 handler_radial.go/handler_scatter.go | radial fallback 350, scatter fallback 400 硬编码 |
| 6.3 | handler_spinaoe.go 旋转参数是否从配置读取 | 检查 spinSpeed/innerRatio/innerBonusMul | spinSpeed=3.0, innerRatio=0.5, innerBonusMul=1.5 硬编码 |
| 6.4 | apply_hit.go deathMark 默认值 | 检查 fallback 路径 | explodeDmg=10+15*(str/100), explodeR=50 硬编码 |

## 七、资源完整性

> 历史教训：sprite 缺失导致白圈 fallback。

| # | 审查项 | 执行方法 | 已知发现 |
|---|--------|---------|---------|
| 7.1 | 10 种塔视觉变体是否都有 sprite 目录和映射 | ls assets/towers/ + 检查 spriteKeyForStyle | `railgun` 无 sprite 资源、无映射，渲染为 sentinel |
| 7.2 | sfx.json 中映射的 WAV 文件是否都存在 | 交叉检查 config/audio/sfx.json vs assets/audio/ | pierce 相关 WAV 可能为废弃资源 |
| 7.3 | 颜色字面量是否应使用 theme 常量 | grep `color.RGBA{` in draw_*.go | ~49 处硬编码颜色应引用 theme 常量 |

## 八、错误处理

> 历史教训：静默失败导致难以排查。

| # | 审查项 | 执行方法 | 已知发现 |
|---|--------|---------|---------|
| 8.1 | 持久化 load 是否处理错误 | 检查 progress.go 的 Get 调用 | `_ = s.Get(progressKey, ...)` 静默忽略错误 |
| 8.2 | lifecycle hook panic 是否有日志 | 检查 recover 块 | `_ = r` 完全吞掉 panic，开发时隐藏 bug |
| 8.3 | 自动播放 package-level map 是否线程安全 | 检查 scaling.go 全局变量注释 | 单线程注释，但 autoplay 并行实例会竞争 |

---

## 使用方法

```
# 新 session 中执行
请按照 docs/bug/review-checklist.md 的审查项逐一执行检查，
对每个审查项给出 PASS / FAIL / WARN 结论和具体证据（文件:行号）。
将新发现的 bug 追加到 docs/bug/record.md，优化项追加到 docs/bug/optimization.md。
```
