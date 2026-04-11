# 优化与改进

## UI / HUD

- [ ] 独行加成的周围是多大？（需展示范围）
- [ ] 对于没有潜力增长的部分不需要展示在描述中，比如一直显示 +（0%）没有意义。
- [ ] 对于潜能为0部分不需要展示。
- [ ] 选择能力时就展示出全面的成长内容。
- [ ] 让沉默等buff也显示范围圈。
- [ ] 炮塔需要展示buff光环。
- [ ] 如何解决血条重叠的问题？
- [ ] 升级菱形（炮塔获取能力之后提示用户选取）。

## VFX / 特效

- [ ] 模型加大展示，制作动画帧。
- [ ] 预览特效功能，查看一个特效后再预览下一个特效时没有清除旧特效展示。
- [ ] 预览特效定身地面删除，代码中也删除。
- [ ] 加速光环优化。
- [ ] 火灵特效优化。
- [ ] 旋转弧刃优化。
- [ ] 光束太简陋。
- [ ] 弹道拖尾需要射击展示。
- [ ] 标准弹需要射击展示。
- [ ] 串联链没有动效。
- [ ] 水灵三技能都有圈。

## VFX 清理（未使用则删除）

- [ ] 护甲火花如果没有使用也删除。
- [ ] 增益光环如果没有使用则也删除。
- [ ] 状态原点如果没有使用则也删除。
- [ ] 命中闪红在哪用到？
- [ ] 增益圆点查看是否需要。
- [ ] 光环脉冲（谁在用）。
- [ ] 力量光环（炮塔的强度？），取消表达方式。
- [ ] 建造波纹在哪使用？
- [ ] 闪电链应该也是没有使用了。
- [ ] 原始冲击也是可以删除。
- [ ] 冲击圆环（给溅射用？）。

## 游戏设计

- [ ] 伤害管线。攻击力，攻击力加成，伤害加成，额外伤害加成，暴击，削弱对方。
- [ ] buff如何更好的表达增加攻速/射程/攻击力，应该增加哪个维度。
- [ ] 聚能战灵，以有向无环图形式链接。
- [ ] 溢出的能力如何处理？
- [ ] 基于小时数的随机数。每局随机9个已经有一个能力的炮塔，额外花费xx金币。
- [ ] 让能力的自动解锁换成金币解锁。
- [ ] 添加更多针对性的能力。
- [ ] 削弱一些炮塔的攻击方式伤害的幅度。
- [ ] 经济系统。

## 死代码清理（审查发现）

- [ ] `DrawFlyingShadow` (vfx_stage.go:215) 零调用者，删除
- [ ] `ApplyRoot`/`ApplyDamageUp`/`ApplyDamageDown`/`ApplyControlImmunity` (crowd_control.go/damage_pipeline.go) 零外部调用
- [ ] `fmtAttr`/`buildAttrSegs` (stage_info_vm.go:178,190) 零调用者
- [ ] 10 个未使用 theme 常量：FactionBase/Output/Control/Support, ToneDanger, MapPathDash, SlotHintPulse, EnemySwarmTri, BuffCrit, BuffStr
- [ ] 8 个从未播放 SFX 常量：SFXHit, SFXEnemyDeathElite, SFXExplode, SFXKnockback, SFXShieldBreak, SFXHitShield, SFXChoiceAppear, SFXChoiceSelect
- [ ] 3 个 TODO stub 能力对玩家无效果：KillUpgrade/WaveScale/NeighborBoost（scaling.go），需实现或从注册中移除
- [ ] deathMark 不在 abilities.json → 查表永远 nil → 代码走硬编码 fallback，需补配置或删除 case
- [ ] sfx.json 中 `fire-pierce.wav`/`hit-pierce.wav` 引用不存在的 pierce 攻击方式，应标记废弃
- [ ] settings.json 11+ 个 `_legacy_*` 区段参数与 balance.json 重复，考虑清理

## 性能优化（审查发现）

- [ ] aura 能力 srcKey 每帧 fmt.Sprintf（config_ability.go:207-266），应预计算缓存
- [ ] chain.go:143 每帧 Sprintf 稳定字符串 `chain_warden_%d`，应 Init 时缓存
- [ ] towerAccKey() (scaling.go:24) 每帧每塔 8 次 Sprintf，应直接用 t.InstanceKey
- [ ] sprite/cache.go:51 cacheKey() 每次 Get 调用 Sprintf，可改为字符串拼接
- [ ] draw_projectile.go:20 每弹道每帧 `make([]TrailPt, N)`，应栈分配或预分配
- [ ] draw_enemy.go:276 每敌每帧 `var dots []StatusDot` + append，应用固定数组
- [ ] telemetry Record() 每伤害事件 10 次 mutex lock/unlock，单线程游戏可改为无锁

## 展示优化（审查发现）

- [ ] "MISS"/"CAP" 英文浮字改中文（如"闪避"/"上限"）
- [ ] CC/DoT 成功施加无浮字反馈，建议添加"眩晕"/"减速"/"灼烧"等成功提示
- [ ] 核心机甲战灵 AoE 模式无视觉区分，需添加模式切换指示
- [ ] 28 处 draw_*.go 颜色硬编码可提取为 theme 常量（约 19 处有对应常量）
- [ ] tower/lifecycle.go:54 recover 块静默吞掉 panic（`_ = r`），需加 log
