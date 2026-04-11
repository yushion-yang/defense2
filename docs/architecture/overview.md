# 项目架构全览

> 所有图表使用 Mermaid 语法，GitHub / VSCode 原生渲染。

---

## 1. 场景流

```mermaid
stateDiagram-v2
    [*] --> Title
    Title --> Select : tap

    Select --> CampaignSelect : mode=campaign
    Select --> TestSelect : mode=test
    Select --> Stage : 其他模式
    Select --> Settings : 设置按钮

    CampaignSelect --> Stage : 选择地图
    CampaignSelect --> Select : 返回

    TestSelect --> Stage : 选择场景
    TestSelect --> Select : 返回

    Settings --> Select : 返回
    Settings --> Stage : 返回(从暂停进入时)

    Stage --> Result : 胜利/失败
    Stage --> Settings : 暂停>设置
    Stage --> Stage : 暂停>重开

    Result --> Stage : 重玩
    Result --> Select : 菜单
```

---

## 2. 每帧 Tick 管线（updatePlaying）

```mermaid
flowchart TD
    Start([每帧开始]) --> Paused{暂停?}
    Paused -->|是| Return([跳过])
    Paused -->|否| DT[计算 gameDT = dt × gameSpeed]
    DT --> HitStop{命中暂停?}
    HitStop -->|是| VFXOnly[仅更新飘字+特效]
    HitStop -->|否| Session[Session.Tick]

    Session --> WardenCheck{需要选战灵?}
    WardenCheck -->|是| ShowWarden[显示战灵选择]
    WardenCheck -->|否| Step1

    Step1["1. Spawn 出怪"] --> Step1b["1.5 波次公告"]
    Step1b --> Step2["2. 状态效果 Tick\n(DoT/减速/虚弱/免疫)"]
    Step2 --> Step2b["2.5 个体行为\n(狂暴/回血/传送)"]
    Step2b --> Step3["3. 敌人移动\n(到终点→扣命)"]
    Step3 --> Step3b["3.5 死亡动画 Tick"]
    Step3b --> Step3c["3.6 群体行为\n(治疗/隐身/加速/相位/削强/净化)"]
    Step3c --> Step4["4. 战灵 Tick\n(攻击/移动/技能)"]
    Step4 --> Step5["5.5 塔动画\n(建造/卖塔)"]
    Step5 --> Step6["6. 能力 Tick\n(光环/区域/经济/重置属性)"]
    Step6 --> Step6b["6.1 削强连接"]
    Step6b --> Step7["7. 塔战斗\n(索敌/开火)"]
    Step7 --> Step7b["7.5 弹射物移动"]
    Step7b --> Step8["8. 弹射物命中\n(碰撞/伤害/击杀)"]
    Step8 --> Step9["9. VFX 更新\n(飘字/震屏/粒子)"]
    Step9 --> Step10["10.5 安全网 HP<=0 清理"]
    Step10 --> Step11["11. 胜负判定"]
    Step11 --> Step12["12. AutoPlay AI"]
```

---

## 3. 伤害管线（ApplyDamage 8 步）

```mermaid
flowchart TD
    Input([输入伤害 + 类型]) --> S1

    S1{"1 免疫检查\n不可选中/无敌/伤害免疫"}
    S1 -->|pure 穿透| S2
    S1 -->|被阻挡| Blocked([伤害=0, 飘字'免伤'])
    S1 -->|通过| S2

    S2{"2 Boss %HP 上限\n(仅 %HP 伤害)"}
    S2 --> S3

    S3{"3 攻击者增伤\ntrue/pure 跳过"}
    S3 --> S4

    S4{"4 目标减伤\ntrue/pure 跳过"}
    S4 --> S41

    S41{"4.1 减伤比例\nDamageReduceRatio"}
    S41 --> S425

    S425{"4.25 虚弱增伤\nDamageAmplify\n上限 50%"}
    S425 --> S45

    S45{"4.5 伤害上限\n固定值 / %HP\n沉默时失效"}
    S45 -->|触发| CAP([飘字'CAP'])
    S45 --> Floor

    Floor["最低保底 1 点"] --> S5

    S5["5 HP 扣减"] --> S6
    S6["6 阈值触发"] --> S7
    S7{"7 死亡检查\nHP <= 0?"}
    S7 -->|是| Dead([Killed=true])
    S7 -->|否| Alive([返回伤害值])
```

---

## 4. 交互状态机（11 模式）

```mermaid
stateDiagram-v2
    [*] --> Idle

    Idle --> BuildMenu : B键 / 造塔按钮
    Idle --> TowerSel : 点击塔
    Idle --> SpawnMenu : 造怪按钮(测试)
    Idle --> Paused : ESC / P
    Idle --> WardenSelect : 首波倒计时结束
    Idle --> ItemPanel : I键 / 道具按钮

    BuildMenu --> Idle : ESC / B / 点外部
    BuildMenu --> BuildPlace : 选择塔类型

    BuildPlace --> Idle : ESC / 放塔成功
    BuildPlace --> Paused : P

    TowerSel --> Idle : ESC / 点空白 / 卖塔
    TowerSel --> Upgrade : 能力按钮
    TowerSel --> TowerSel : 点另一座塔

    Upgrade --> TowerSel : 选择能力 / ESC

    SpawnMenu --> Idle : ESC
    SpawnMenu --> SpawnPlace : 选类型

    SpawnPlace --> Idle : ESC

    Paused --> Idle : 恢复(→prePauseMode)
    Paused --> Settings : 设置
    Paused --> NewGame : 重开/退出

    WardenSelect --> Idle : 选定 / ESC跳过

    ItemPanel --> Idle : ESC / I
    ItemPanel --> ItemDrag : 按住道具卡

    ItemDrag --> ItemPanel : 释放(成功/取消)
```

---

## 5. 包依赖层次

```mermaid
graph BT
    subgraph "Layer 0 — 叶子包"
        config[config]
        game[core/game]
        telemetry[core/telemetry]
        economy[core/economy]
        event[core/event]
        strength[core/strength]
    end

    subgraph "Layer 1 — 基础实体"
        gamemap[core/gamemap]
        enemy[core/enemy]
        projectile[core/projectile]
        gamemode[core/gamemode]
    end

    subgraph "Layer 2 — 复合实体"
        tower[core/tower]
        buff[core/buff]
    end

    subgraph "Layer 3 — 系统逻辑"
        combat[core/combat]
        warden[core/warden]
        item[core/item]
    end

    subgraph "Layer 4 — 数据驱动"
        abilities[tower/abilities]
        wtypes[warden/types]
    end

    subgraph "Layer 5 — 管线"
        pipeline[core/pipeline]
    end

    subgraph "Layer 6 — 表现层"
        render[render]
        hud[render/hud]
        input[input]
    end

    subgraph "Layer 7 — 场景"
        scene[scene]
    end

    %% Layer 1
    enemy --> game
    enemy --> gamemap
    enemy --> telemetry
    gamemap --> config
    gamemode --> config

    %% Layer 2
    tower --> config
    tower --> enemy
    tower --> projectile
    tower --> strength

    %% Layer 3
    combat --> enemy
    combat --> projectile
    combat --> tower
    combat --> telemetry
    warden --> combat
    warden --> enemy
    warden --> tower
    item --> tower

    %% Layer 4
    abilities --> combat
    abilities --> tower
    abilities --> strength
    wtypes --> warden
    wtypes --> strength

    %% Layer 5
    pipeline --> combat
    pipeline --> enemy
    pipeline --> tower
    pipeline --> projectile
    pipeline --> warden

    %% Layer 6
    render --> enemy
    render --> tower
    render --> projectile
    render --> warden
    hud --> game
    hud --> render

    %% Layer 7
    scene --> pipeline
    scene --> render
    scene --> hud
    scene --> config
    scene --> input
```

---

## 6. ApplyHit 命中处理流程

```mermaid
flowchart TD
    Hit([命中事件]) --> Evasion{"1 闪避检查\n概率 EvasionChance\n(沉默时失效)"}
    Evasion -->|闪避| Miss([MISS 飘字, 返回空])
    Evasion -->|未闪避| Shield

    Shield["2 弹幕盾标记\nbounce/scatter/radial\n(仅标记,不改伤害)"]
    Shield --> Armor

    Armor["3 装甲减免\n固定减免 ArmorFlat\n(最低1,沉默时失效)"]
    Armor --> Dash

    Dash["4 受击冲刺触发\n(沉默时失效)"]
    Dash --> OnHit

    OnHit["5 遍历塔能力 OnHit\nCC/DoT/溅射/弹射\nweaken 在此设置"]
    OnHit --> Crit

    Crit{"6 CritBonus 暴击\n(critAura 独立触发)\n固定 2x"}
    Crit --> Amp

    Amp["7 DamageAmp 全伤害增幅\n(damageUpAura)"]
    Amp --> Pipeline

    Pipeline["8 ApplyDamage\n→ 8步伤害管线"]
    Pipeline --> Callback

    Callback["9 命中回调\n(飘字/音效)"]
    Callback --> Kill{"击杀?"}

    Kill -->|是| Death["tower.Kills++\ndeathMark AoE\nenemies.Kill()"]
    Kill -->|否| ShieldVFX

    Death --> ShieldVFX
    ShieldVFX["10 弹幕盾视觉\n(正常受伤后触发 BlockFlash)"]
    ShieldVFX --> Result([返回 HitOutput])
```
