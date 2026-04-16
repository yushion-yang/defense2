// knowledge.go -- 游戏知识库。
//
// 将塔防游戏的核心机制编码为结构化文本，注入 LLM system prompt。
// 包含两个版本：完整版（用于参考）和压缩版（用于 API 调用，~800 tokens）。
// 关联：由 connector.go callStrategicAPI() 作为 system prompt 注入，
//       使 LLM 理解游戏机制后做出有深度的战略决策。
package llm

// GameKnowledge 完整游戏知识（供参考和测试，不直接用于 API）。
const GameKnowledge = `# 塔防游戏机制知识

## 塔系统
- 基础塔(basic)造价50金币，建造后随机获得S/B/D三档属性
- 每2波解锁一个能力槽(共6个)，每次3选1
- 升级(+10强度)花费10金币，公式: 属性 = Base + Potential * (Strength/100)
- 强度越高收益越大(线性增长)，但金币有限需要取舍

## 6种能力类别
1. 攻击方式(必选): 决定塔的身份
   - projectile(追踪弹): 单体精准
   - scatter(散射): 扇形多目标，克制密集群
   - wideBeam(宽光束): 穿透直线
   - spinAoe(旋转斩): 近距离360度
   - radial(环射): 全方位同时攻击
   - barrage(连击): 快速多段低伤，克制damageCap
2. CC(控制): slow(减速), stun(眩晕), root(定身)
3. 伤害加成: crit(暴击), bonusDamage(额外伤害), percentHp(百分比生命)
4. 光环/增益: damageAmp(增伤光环), speedAura(攻速光环), rangeAura(范围光环)
5. DoT(持续伤害): bleed(流血), burn(灼烧), poison(中毒)
6. 区域: zoneDamage(区域伤害), goldPassive(金币产出)

## 敌人类型与克制
- tank(坦克): 高HP低速 -> 用percentHp或持续升级主力
- runner(快跑): 高速低HP -> 用slow/root控住
- swarm(蜂群): 大量低HP -> 用scatter/spinAoe/splash范围清
- boss: 极高HP+伤害上限5% -> 用barrage多段突破上限
- stealth(隐身): 普通塔看不到 -> 需要特定能力reveal
- healer(治疗): 治疗周围友方 -> 优先击杀
- splitter(分裂): 死后分裂成小怪 -> 预留AOE清理

## 经济节奏
- 击杀收入: 15金/只(乘原型倍率)
- 波次奖励: 12+4*波次 (完美清波额外+2+2*波次)
- 建议: 前3波攒钱只造1-2塔，波4前补到3塔应对首个Boss
- 卖塔返还70%，差塔趁早卖

## 位置策略
- 入口区: 放CC塔(slow/root)，让敌人一进来就被控住
- 中段区: 放主力DPS塔，趁敌人被控时输出
- 出口区: 放cleanup塔(scatter/spinAoe)，清理漏网之鱼
- 交叉覆盖: 两塔射程重叠 = 协同伤害倍增

## 道具使用
- BaseDamage道具 -> 给低base高potential的塔(放大potential收益)
- PotentialDamage道具 -> 给高strength的塔(strength放大potential)
- Speed道具 -> 给攻速最慢的主力塔
- Range道具 -> 给覆盖关键路段但range不足的塔

## 协作原则
- 队友全DPS -> 我补CC，让敌人被控住给队友打
- 队友全CC -> 我补DPS，趁敌人被控时输出
- 队友防前段 -> 我守后段
- 共享生命，任何一边漏怪都扣血`

// CondensedKnowledge 压缩版游戏知识（~800 tokens，用于 API system prompt）。
// 精简要点，省略细节，让 LLM 用最少 token 理解核心机制。
const CondensedKnowledge = `你是塔防游戏AI。核心机制：
塔:造价50金,升级+10强度花10金,公式:属性=Base+Potential*(Strength/100)。每2波解锁能力槽(6个)3选1。
6种能力:攻击方式(projectile/scatter/wideBeam/spinAoe/radial/barrage)、CC(slow/stun/root)、伤害(crit/bonusDamage/percentHp)、光环(damageAmp/speedAura/rangeAura)、DoT(bleed/burn/poison)、区域(zoneDamage/goldPassive)。
敌人克制:tank用percentHp,runner用slow/root,swarm用scatter/spinAoe,boss用barrage突破5%伤害上限。
经济:击杀15金,波次奖励12+4*波次,完美清波额外+2+2*波次。前3波攒钱造1-2塔。卖塔返70%。
位置:入口放CC,中段放DPS,出口放cleanup。协同:CC塔旁放DPS效率倍增。
道具:Damage给高伤塔,Speed给慢速塔,Range给覆盖不足塔。
协作:队友全DPS我补CC,队友全CC我补DPS,共享生命。
返回JSON操作数组,每个含type/position/priority/target/reason。`
