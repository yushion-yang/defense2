# 代码审查清单

> 基于已修复 bug 的模式归纳 + 代码实际扫描推导而成。
> 最近审查日期：2026-04-11

---

## 审查结果总览

| 结论 | 数量 | 说明 |
|------|------|------|
| PASS | 10 | 检查通过，无问题 |
| WARN | 14 | 存在隐患或待优化，非紧急 |
| FAIL | 14 | 确认存在问题，需修复 |

---

## 一、配置-代码一致性

| # | 结论 | 审查项 | 证据 |
|---|------|--------|------|
| 1.1 | **PASS** | abilities.json 每个 type 在 config_ability.go 有对应 case | 31 个 type 全覆盖。攻击方式类(scatter/wideBeam/spinAoe/radial)走 pipeline 分发不走 OnHit/OnTick，符合设计 |
| 1.2 | **WARN** | config_ability.go 每个 case 在 abilities.json 中有定义 | `"stun"` (config_ability.go:123) 是废弃别名；`"deathMark"` (config_ability.go:181) 无 JSON 定义，apply_hit.go:310 查表返回 nil 后走硬编码 fallback |
| 1.3 | **PASS** | attackStyleLabel 覆盖所有攻击方式 | 5 个 AttackStyle 常量(tower.go:19-23)全部有中文标签(stage_info_vm.go:225-231)。bounce/multiTarget/splash 不是独立 AttackStyle，用 projectile |
| 1.4 | **PASS** | enemies-core.json 每个 archetype 在 spawner 可生成 | 17/18 战斗原型覆盖（wave 15+ 全出）。`dummy` 为测试桩（hpScale=10000, speed=0），仅 EnemyFilter 可触发，符合设计 |
| 1.5 | **WARN** | balance.json 参数与 settings.json 无重复 | 11+ 参数重复定义，settings.json 标记为 `_legacy_*` 但仍保留。一处值冲突：minSpeedRatio(0.2) vs slowCap(0.30)。代码只读 balance.json |
| 1.6 | **PASS** | vfx.json 每个 VFX ID 有函数 + preview 注册 | 80/80 全覆盖，零缺口 |

## 二、死代码与未使用导出

| # | 结论 | 审查项 | 证据 |
|---|------|--------|------|
| 2.1 | **FAIL** | vfx/ 包所有导出函数有外部调用者 | `DrawFlyingShadow` (vfx_stage.go:215) 零外部调用者 |
| 2.2 | **FAIL** | combat/ 包导出函数有调用者 | `ApplyRoot` (crowd_control.go:92)、`ApplyDamageUp` (:216)、`ApplyDamageDown` (:224)、`ApplyControlImmunity` (:120) 零外部调用 |
| 2.3 | **FAIL** | stage_info_vm.go 辅助函数有调用者 | `fmtAttr` (:178)、`buildAttrSegs` (:190) 无调用者 |
| 2.4 | **FAIL** | theme/colors.go 所有常量有引用 | 10 个未使用：FactionBase/Output/Control/Support(:83-86), ToneDanger(:72), MapPathDash(:108), SlotHintPulse(:154), EnemySwarmTri(:200), BuffCrit(:181), BuffStr(:182) |
| 2.5 | **FAIL** | audio SFX 常量有 Play 调用 | 8 个从未播放：SFXHit(:205), SFXEnemyDeathElite(:207), SFXExplode(:217), SFXKnockback(:239), SFXShieldBreak(:241), SFXHitShield(:242), SFXChoiceAppear(:232), SFXChoiceSelect(:233)。3 个仅 fallback 间接引用 |
| 2.6 | **FAIL** | scaling.go 注册能力有实际效果 | 3/5 纯 TODO stub 无效果：KillUpgrade(:62), WaveScale(:86), NeighborBoost(:191)。2/5 部分功能：PeriodicCast case 2 是 stub, ElementSwitch 火/雷是 stub |

## 三、展示与 HUD

| # | 结论 | 审查项 | 证据 |
|---|------|--------|------|
| 3.1 | **FAIL** | potential=0 的能力 {s%} 显示有意义 | `bleedDot`(base=0.01, potential=0) 显示 `1%+(0%)→1%`，零成长段误导玩家。`enhance`(base=0.2, potential=0) 同理显示 `20%+(0%)→20%` |
| 3.2 | **PASS** | 概率/减速类展示 clamp 到 100% | `isPercentCapped` (stage_info_vm.go:289) 正确处理 chance/factor 维度，FormatAbilityDisplay 和 buildAbilitySegments 两处均已 clamp |
| 3.3 | **WARN** | 所有用户可见文本中文化 | 玩家可见英文：`"MISS"` (apply_hit.go:47)、`"CAP"` (damage_pipeline.go:164)。可接受的键盘标签：`Esc`/`S`/`B`/`Del` (pause_menu.go:86)。调试面板英文忽略 |
| 3.4 | **PASS** | 100 强度时属性颜色全白 | `scaledColor` (stage_info_vm.go:262) 在 scaled==potential 时返回 `theme.TextBody`(0xe2e8f0 近白色)，非绿非红 |
| 3.5 | **FAIL** | potential=0 隐藏 +(0%) 展示 | `buildAbilitySegments` 的 {s%} 分支(stage_info_vm.go:446-450) 在 base>0 && potential=0 时仍输出三段 `"base%+(0%)→base%"`，零成长段无意义 |
| 3.6 | **FAIL** | CC/DoT 命中有浮字反馈 | ApplyStun/ApplySlow 成功路径无浮字，仅免疫路径有。bleed/burn/poison/weaken 施加时均无浮字 |
| 3.7 | **FAIL** | 免疫浮字声明具体类型 | 全部显示通用 `"免疫"` 无类型区分。仅通过颜色微调区分：红色=控制免疫(crowd_control.go:21,95)，蓝色=减速免疫(:52) |

## 四、性能热路径

| # | 结论 | 审查项 | 证据 |
|---|------|--------|------|
| 4.1 | **WARN** | core/render 的 Tick/Draw 中无 fmt.Sprintf | 15+ 处 Sprintf 在每帧路径。最高频：aura 能力 srcKey(config_ability.go:207-266) 每光环塔每帧；chain.go:143 每帧生成稳定字符串 |
| 4.2 | **WARN** | sprite cache key 无逐帧分配 | cacheKey() (sprite/cache.go:51) 用 fmt.Sprintf，每次 Get 调用均触发。draw_enemy.go 有 spritePathCache 部分缓解 |
| 4.3 | **WARN** | Draw 函数中无堆分配 slice | draw_projectile.go:20 每弹道每帧 `make([]TrailPt, N)`；draw_enemy.go:276 每敌每帧 `var dots []StatusDot` + append |
| 4.4 | **FAIL** | 战灵 ID 键无逐帧生成 | chain.go:143 `fmt.Sprintf("chain_warden_%d", w.ID)` 每帧执行，ID 不变应在 Init 时缓存 |
| 4.5 | **FAIL** | Telemetry 锁竞争 | damage_pipeline.go 单次伤害事件 10 次 tel.T.Record()，每次 sync.Mutex lock/unlock。多塔战斗每帧数百次 mutex 操作 |

## 五、游戏逻辑完整性

| # | 结论 | 审查项 | 证据 |
|---|------|--------|------|
| 5.1 | **FAIL** | 沉默有免疫检查和视觉反馈 | silenceZone (config_ability.go:282-292) 直接 `e.Silenced=true` 无免疫检查、无 tenacity、无浮字、无 VFX 覆层、无 SFX |
| 5.2 | **FAIL** | 所有 CC 类型有完整管线 | stun/slow/root 有 Apply 函数+免疫+tenacity+VFX，但均无成功浮字。silence 完全缺失管线（无 ApplySilence 函数、无免疫、无 VFX、无 SFX）。root 有完整实现但零调用者 |
| 5.3 | **FAIL** | scaling.go package-level map 游戏结束后清理 | `ResetScalingState()` (scaling.go:312) 零生产调用者，仅 test 调用。多局游戏 6 个 map 无限增长 |
| 5.4 | **FAIL** | 战灵 cleanup 路径有 nil guard | envoy.go:127,158 调用 `Strength.RemoveTemp(key)` 无 nil 检查；apply 路径(envoy.go:161)有 `ensureStrength()`，不一致 |
| 5.5 | **WARN** | tick_abilities.go 对 Buffs 有 nil 检查 | tick_abilities.go:46 `t.Buffs.Tick(dt)` 无 nil 检查，RecalcStats (tower.go:132) 有。pool.go:70 初始化保障当前安全，但防御模式不一致 |
| 5.6 | **FAIL** | 核心机甲 AoE 模式有视觉区分 | drawCoreEffects (draw_warden.go:196) 仅绘制通用 shoot flash，AoE 模式与单目标无区分。CoreState 无 AoEActive 标志暴露给渲染层 |

## 六、硬编码魔数

| # | 结论 | 审查项 | 证据 |
|---|------|--------|------|
| 6.1 | **WARN** | scaling.go 能力参数从配置读取 | 15 个魔数无配置来源。活跃代码：冰减速 0.7/0.5s(scaling.go:269-270)、毒伤 2DPS(:289)、stunAoe 0.6s(:150)。TODO stub 中的魔数暂不紧急 |
| 6.2 | **FAIL** | combat handler fallback 速度从 balance.json 读 | radial 350(handler_radial.go:39)、scatter 400(handler_scatter.go:21) 硬编码，balance.json defaultProjectileSpeed=300 未引用 |
| 6.3 | **WARN** | handler_spinaoe.go 旋转参数从配置读取 | spinSpeed=3.0/0.3(:27)、innerRatio=0.5(:45)、innerBonusMul=1.5(:46)、视觉时间 0.3/0.4(:84-85) 硬编码。innerRatio 与 abilities.json 一致但为复制值 |
| 6.4 | **WARN** | apply_hit.go deathMark 默认值 | deathMark 不在 abilities.json → 查表永远返回 nil → fallback 公式 `10+15*(str/100)` 和 radius=50 实际是唯一路径 |

## 七、资源完整性

| # | 结论 | 审查项 | 证据 |
|---|------|--------|------|
| 7.1 | **FAIL** | 10 种塔视觉变体都有 sprite 目录和映射 | `assets/towers/railgun/` 缺失，无 spriteKey 映射，渲染 fallback 为 sentinel。其余 9 种完整 |
| 7.2 | **WARN** | sfx.json 映射的 WAV 文件都存在 | 126 个 WAV 全存在。`fire-pierce.wav`/`hit-pierce.wav` 引用不存在的 `pierce` 攻击方式未标记废弃。`root-apply.wav` 存在但未被 sfx.json 引用 |
| 7.3 | **WARN** | 颜色字面量使用 theme 常量 | 28 处 `color.RGBA{}` 硬编码（draw_enemy 14 + draw_tower 5 + draw_warden 9），约 19 处可替换为 theme 常量 |

## 八、错误处理

| # | 结论 | 审查项 | 证据 |
|---|------|--------|------|
| 8.1 | **WARN** | 持久化 load 处理错误 | progress.go:107 `_ = s.Get(...)` 静默忽略错误。首次启动无文件时预期，但存档损坏时也会静默丢失 |
| 8.2 | **WARN** | lifecycle hook panic 有日志 | 4 处 recover 块中 3 处有 log。tower/lifecycle.go:54 `_ = r` 静默吞掉 panic，注释说"生产环境可替换为日志" |
| 8.3 | **PASS** | 自动播放 package-level map 线程安全 | autoplay 顺序执行(cmd/autoplay/main.go:162 for 循环)，无并发。单线程注释准确 |

---

## 新发现汇总（本次审查新增）

### 新增 Bug（→ record.md）

| 来源 | 问题 |
|------|------|
| 5.1/5.2 | silenceZone 无免疫检查，控制免疫敌人也被沉默 |
| 5.3 | scaling.go 全局 map 多局游戏不清理，内存泄漏 |
| 5.4 | envoy.go:127,158 Strength 无 nil 检查，潜在 panic |
| 6.2 | radial/scatter handler 用硬编码速度 350/400，不用 balance.json 的 300 |
| 7.1 | railgun 无 sprite 资源，渲染为 sentinel 外观 |

### 新增优化项（→ optimization.md）

| 来源 | 问题 |
|------|------|
| 2.1-2.5 | 死代码清理：DrawFlyingShadow、ApplyRoot/DamageUp/Down/ControlImmunity、fmtAttr/buildAttrSegs、10 theme 常量、8 SFX 常量 |
| 2.6 | 3 个 TODO stub 能力(KillUpgrade/WaveScale/NeighborBoost)对玩家无效果 |
| 3.1/3.5 | potential=0 能力展示 +(0%) 零成长段 |
| 3.3 | "MISS"/"CAP" 英文浮字 |
| 4.1-4.5 | 热路径 fmt.Sprintf 和 slice 分配优化 |
| 7.3 | 28 处颜色硬编码可提取到 theme |
