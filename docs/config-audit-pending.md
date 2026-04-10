# Config 审查 — 待决项

> 来源: docs/config-audit-2026-04-10.md 中未修复的项目
> 请在「决定」列填写意见，我会据此执行。

---

## 一、致命/严重 — 需要明确方向

| # | 级别 | 问题 | 文件 | 建议选项 | 决定 |
|---|------|------|------|---------|------|
| 1 | P0 | map_04 pathOrder 在 [9,31]→[4,35] 断裂，敌人瞬移 | levels/map_04.json | A) 修复路径补全缺失节点 B) 整条螺旋路径重新生成 C) 暂不处理(地图未上线) | |
| 2 | P0 | map_06 pathOrder 两处断裂: [10,2]→[12,2] 缺 [11,2]; [16,1]→[16,19] 跳 18 列 | levels/map_06.json | A) 逐点修复 B) 重新生成 C) 暂不处理 | |
| 3 | P0 | map_08 entries topRight/botRight cell 指向 CellBase(5) 非 CellSpawn(4) | levels/map_08.json | A) 修正 cell 指向实际 CellSpawn B) 这是双向地图设计(左进右出+右进左出) C) 暂不处理 | |
| 4 | P1 | 敌人颜色冲突: tank(紫 vs 灰)/healer(绿 vs 粉)/splitter(紫 vs 橙)，enemies-core 和 visuals/enemies 完全不同 | enemies-core.json + visuals/enemies.json | A) 以 enemies-core 为准，更新 visuals B) 以 visuals 为准，更新 core C) 重新定义两者统一的新色板 | |
| 5 | P1 | 6 个 sprite 缺视觉定义: colossus/ironwill/steadfast/devoter/mirror/dummy | visuals/enemies.json | A) 补全视觉定义(需要设计输入) B) 复用已有相近 sprite C) 暂不处理(代码有 fallback) | |
| 6 | P1 | map_04~08 网格 2400×1080 是逻辑分辨率 1200×540 的 2 倍 | levels/map_04~08.json | A) 有摄像机缩放(已实现?), 仅补文档说明 B) 无摄像机, 需要缩小地图 C) 暂不处理 | |

---

## 二、中等 — 需要确认处理方式

| # | 级别 | 问题 | 文件 | 建议选项 | 决定 |
|---|------|------|------|---------|------|
| 7 | P2 | damageDownFloor=0.2 在三处重复定义(damage-pipeline/buff-stack/balance)，无明确 source of truth | 三个 systems JSON | A) 指定 balance.json 为权威，其余引用 B) 指定 buff-stack.json 为权威 C) 保持现状(值一致就行) | |
| 8 | P2 | towers.json 的 baseDamage/potentialDamage 等 6 个属性是死数据(被 tier-presets 完全覆盖) | towers/towers.json | A) 删除死字段，加注释说明 B) 保留作为 fallback/文档参考 C) 改代码让它们生效(作为默认值) | |
| 9 | P2 | wardens.json specialDesc 模板变量({fireballDmg}等)无对应数值 | wardens/wardens.json | A) 在 wardens.json 中补数值字段 B) 移除模板变量，写死文案 C) 暂不处理(战灵系统未完善) | |
| 10 | P2 | boss-templates.json 7 个模板全部未实现，bossRotateWeakness 引用不存在的元素系统 | enemies/boss-templates.json + systems/boss.json | A) 移到 config/draft/ 标注未实现 B) 删除 C) 保留原样 | |
| 11 | P2 | cc.json 定义了 root(定身)但无能力施加它 | systems/cc.json | A) 标注 reserved B) 删除 root 定义 C) 保留(基础设施先行) | |
| 12 | P2 | abilities.json 多个能力 paramDim 非空但 display 不用 {p}(spinAoe/splash/crit/各aura/soloBoost) | abilities/abilities.json | A) 在 display 中加 {p} 展示参数 B) 清空不需要展示的 paramDim C) 暂不处理 | |
| 13 | P2 | abilities.json 图标复用: burn/bleedDot 共用 burn; goldPassive 用 stat-dps | abilities/abilities.json | A) 分配独立图标(需新图标资源) B) 暂不处理(图标资源不足) | |
| 14 | P2 | settings.json 废弃字段: towers.defaultType="laser", towers.modeOrder, world.towerSlots(y>540) | settings.json | A) 删除废弃字段 B) 保留(可能还有代码引用) C) 标注 deprecated | |
| 15 | P2 | vocab.json 位置网格 R0-R5/C0-C11 无法覆盖实际 row=10/col=22；缺 bounce/splash/multiTarget token；塔 token 含义不明 | llm/vocab.json | A) 扩展 vocab 覆盖实际范围 B) 暂不处理(LLM 系统未完善) | |
| 16 | P2 | enemies/abilities.json potential 全部为 0，敌人能力无成长 | enemies/abilities.json | A) 这是设计意图(敌人能力固定)，删除 potential 字段 B) 部分能力加 potential(哪些?) C) 保持现状 | |
| 17 | P2 | map_02 有两个 CellBase(5) 出口 | levels/map_02.json | A) 这是双出口设计，补文档说明 B) 修正为单出口 | |
| 18 | P2 | projectile-defaults.json 自承认 scatter/radial/warden 不一致 | systems/projectile-defaults.json | A) 统一到配置 B) 暂不处理 | |
| 19 | P2 | 控制_低强度.json 0 个敌人，测试无效 | scenarios/控制_低强度.json | A) 删除 B) 补充敌人 C) 保留(可能是快照中间态) | |
| 20 | P2 | 全部怪物静止 vs 所有怪物静止 功能重叠 | scenarios/ | A) 合并为一个 B) 保留两个(有细微差异) | |

---

## 三、平衡性 — 需要设计判断

| # | 级别 | 问题 | 当前值 | 建议选项 | 决定 |
|---|------|------|--------|---------|------|
| 21 | P3 | core(机甲)战灵全面碾压: DPS/射程/移速/成长均最高 | DPS 16.7 vs 次强 10.0 | A) 削弱 core B) 增强其他战灵 C) 保持(core 定位就是全能) | |
| 22 | P3 | skystrike(水灵)基础面板最弱，DPS 仅 core 的 40% | DPS 6.7, interval 1.5s | A) 提升基础属性 B) 保持(靠百分比伤害弥补) | |
| 23 | P3 | envoy(金灵)成长最慢，Lv4/5 正常游戏不可达 | growthOnKill=0, growthOnWaveClear=5 | A) 提升成长速度 B) 降低升级门槛 C) 保持(辅助定位不需高级) | |
| 24 | P3 | speedAura buff 缺半径限制，可能全图加速 20% | 无 radius 字段 | A) 加 radius 限制(参考 healAura 80px) B) 确认代码已有限制 | |
| 25 | P3 | wideBeam rangeMult=3，光束打 600px(半屏) | param=3 | A) 降低到 2 或 1.5 B) 保持(贯穿光束就该这么远) | |
| 26 | P3 | soloBoost(+30%) vs damageUpAura(+15% 惠及多塔)，soloBoost 条件苛刻收益低 | solo+30% vs aura+15% | A) 提升 soloBoost 倍率 B) 保持(niche 定位) | |
| 27 | P3 | armored 固定减伤 5，potential=0，后期无感 | base=5, potential=0 | A) 加 potential 使其随波次成长 B) 保持(前期特色型) | |
| 28 | P3 | colossus 4x HP + 5% 伤害上限，双重防御可能过强 | hpScale=4.0 + damageCapPercent 5% | A) 调高 cap 到 8-10% B) 降低 hpScale C) 保持(Boss 级精英就该难杀) | |
| 29 | P3 | purifier 不可沉默 + 3.5x HP + 每 4s 净化+免疫 2s | hpScale=3.5, silenceable=false | A) 允许沉默 B) 降低 HP C) 保持(终极精英定位) | |
| 30 | P3 | bleedDot potential=0，高强度无额外收益 | base=0.01, potential=0 | A) 加 potential B) 保持(按%HP 扣血天然强力) | |
| 31 | P3 | 减速上限 80% + 速度下限 20%，敌人几乎停下 | slow cap 0.8, minSpeed 20% | A) 降低 cap 到 60-70% B) 保持(需要多塔配合才达上限) | |
| 32 | P3 | Boss %HP 伤害上限 5%，加上多种免疫可能过难 | bossPercentHpCap=0.05 | A) 提高到 8-10% B) 保持 | |
| 33 | P3 | 波次 buff 注入概率 30%，大部分敌人是白板 | buffChance=0.3, wave6-25 max 1 buff | A) 提高概率或 buff 数 B) 保持(buff 是调味不是主菜) | |
| 34 | P3 | deathSplit 子体 30% HP + 速度 x1.4 + 不递归，作为高阶 buff 偏弱 | childHpRatio=0.3 | A) 提升子体属性 B) 保持 | |
| 35 | P3 | 强度购买 10 金/+10，击杀奖励 15 金，Wave 1 即可大幅提升 | 10G/+10str, killReward=15G | A) 提高强度成本 B) 降低前期收益 C) 保持(让玩家有成长感) | |
| 36 | P3 | fortress 塔配色与 shielded 敌人完全相同，视觉混淆 | 两者都是蓝色系 | A) 调整 fortress 色板 B) 调整 shielded 色板 C) 保持 | |
| 37 | P3 | stunDuration 期望眩晕/攻击 比 stunChance 低 30% | 6%×0.7s vs 20%×0.3s | A) 提升 stunDuration 效果 B) 保持(两种定位不同) | |
