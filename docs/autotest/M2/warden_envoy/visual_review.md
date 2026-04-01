# Visual Review Manifest

> 使用方法: 将本文件和同目录的截图一起发给 Claude Vision，
> 让它逐张按检查项回答 Yes/No 并标记发现的问题。

共 57 张截图待审查。

## 塔外观 (1)

### tower_basic.png
![tower_basic.png](tower_basic.png)

**场景**: 塔类型 'basic' 首次建造时的外观

**检查项**:
1. 塔是否渲染为正常的精灵图像（而非纯色圆形/白色圆形 fallback）？ _(ref: 炮塔变白圈)_
2. 塔的精灵是否清晰、无明显缺失部件？

## 塔攻击特效 (1)

### attack_projectile.png
![attack_projectile.png](attack_projectile.png)

**场景**: 攻击方式 'projectile' 首次开火时的战斗画面

**检查项**:
1. 塔是否有可见的攻击动画或特效（粒子、光束、弹射物等）？ _(ref: 旋风塔无攻击展示)_
2. 弹射物/光束是否从塔的位置发出、指向敌人？
3. 塔周围是否有异常的白色圆环 artifact？ _(ref: 旋刃白圈)_
4. 是否能看到伤害浮字（数字）出现在敌人身上？ _(ref: 电磁炮无伤害浮字)_

## 敌人外观 (17)

### enemy_normal.png
![enemy_normal.png](enemy_normal.png)

**场景**: 敌人原型 'normal' 首次出现时的行走状态

**检查项**:
1. 敌人是否渲染为正常的精灵图像（而非占位符）？
2. 敌人是否处于正常的行走动画帧（而非 hit/受击 帧）？ _(ref: 怪物出现就播 hit 帧)_
3. 敌人周围是否有异常的白色方框/光晕？ _(ref: hit帧白方框)_
4. 敌人的血条是否正常显示在头顶？

### enemy_dying.png
![enemy_dying.png](enemy_dying.png)

**场景**: 敌人原型 'dying' 首次出现时的行走状态

**检查项**:
1. 敌人是否渲染为正常的精灵图像（而非占位符）？
2. 敌人是否处于正常的行走动画帧（而非 hit/受击 帧）？ _(ref: 怪物出现就播 hit 帧)_
3. 敌人周围是否有异常的白色方框/光晕？ _(ref: hit帧白方框)_
4. 敌人的血条是否正常显示在头顶？

### enemy_runner.png
![enemy_runner.png](enemy_runner.png)

**场景**: 敌人原型 'runner' 首次出现时的行走状态

**检查项**:
1. 敌人是否渲染为正常的精灵图像（而非占位符）？
2. 敌人是否处于正常的行走动画帧（而非 hit/受击 帧）？ _(ref: 怪物出现就播 hit 帧)_
3. 敌人周围是否有异常的白色方框/光晕？ _(ref: hit帧白方框)_
4. 敌人的血条是否正常显示在头顶？

### boss_wave_5.png
![boss_wave_5.png](boss_wave_5.png)

**检查项**:
1. 敌人是否渲染为正常的精灵图像（而非占位符）？
2. 敌人是否处于正常的行走动画帧（而非 hit/受击 帧）？ _(ref: 怪物出现就播 hit 帧)_
3. 敌人周围是否有异常的白色方框/光晕？ _(ref: hit帧白方框)_
4. 敌人的血条是否正常显示在头顶？

### enemy_swarm.png
![enemy_swarm.png](enemy_swarm.png)

**场景**: 敌人原型 'swarm' 首次出现时的行走状态

**检查项**:
1. 敌人是否渲染为正常的精灵图像（而非占位符）？
2. 敌人是否处于正常的行走动画帧（而非 hit/受击 帧）？ _(ref: 怪物出现就播 hit 帧)_
3. 敌人周围是否有异常的白色方框/光晕？ _(ref: hit帧白方框)_
4. 敌人的血条是否正常显示在头顶？

### enemy_tank.png
![enemy_tank.png](enemy_tank.png)

**场景**: 敌人原型 'tank' 首次出现时的行走状态

**检查项**:
1. 敌人是否渲染为正常的精灵图像（而非占位符）？
2. 敌人是否处于正常的行走动画帧（而非 hit/受击 帧）？ _(ref: 怪物出现就播 hit 帧)_
3. 敌人周围是否有异常的白色方框/光晕？ _(ref: hit帧白方框)_
4. 敌人的血条是否正常显示在头顶？

### boss_appear.png
![boss_appear.png](boss_appear.png)

**场景**: Boss 敌人首次出现的画面

**检查项**:
1. 敌人是否渲染为正常的精灵图像（而非占位符）？
2. 敌人是否处于正常的行走动画帧（而非 hit/受击 帧）？ _(ref: 怪物出现就播 hit 帧)_
3. 敌人周围是否有异常的白色方框/光晕？ _(ref: hit帧白方框)_
4. 敌人的血条是否正常显示在头顶？
5. Boss 敌人是否体型明显大于普通敌人？
6. Boss 是否有独特的视觉标识（颜色/光环）？

### enemy_shielded.png
![enemy_shielded.png](enemy_shielded.png)

**场景**: 敌人原型 'shielded' 首次出现时的行走状态

**检查项**:
1. 敌人是否渲染为正常的精灵图像（而非占位符）？
2. 敌人是否处于正常的行走动画帧（而非 hit/受击 帧）？ _(ref: 怪物出现就播 hit 帧)_
3. 敌人周围是否有异常的白色方框/光晕？ _(ref: hit帧白方框)_
4. 敌人的血条是否正常显示在头顶？

### enemy_armored.png
![enemy_armored.png](enemy_armored.png)

**场景**: 敌人原型 'armored' 首次出现时的行走状态

**检查项**:
1. 敌人是否渲染为正常的精灵图像（而非占位符）？
2. 敌人是否处于正常的行走动画帧（而非 hit/受击 帧）？ _(ref: 怪物出现就播 hit 帧)_
3. 敌人周围是否有异常的白色方框/光晕？ _(ref: hit帧白方框)_
4. 敌人的血条是否正常显示在头顶？

### boss_wave_10.png
![boss_wave_10.png](boss_wave_10.png)

**检查项**:
1. 敌人是否渲染为正常的精灵图像（而非占位符）？
2. 敌人是否处于正常的行走动画帧（而非 hit/受击 帧）？ _(ref: 怪物出现就播 hit 帧)_
3. 敌人周围是否有异常的白色方框/光晕？ _(ref: hit帧白方框)_
4. 敌人的血条是否正常显示在头顶？

### enemy_healer.png
![enemy_healer.png](enemy_healer.png)

**场景**: 敌人原型 'healer' 首次出现时的行走状态

**检查项**:
1. 敌人是否渲染为正常的精灵图像（而非占位符）？
2. 敌人是否处于正常的行走动画帧（而非 hit/受击 帧）？ _(ref: 怪物出现就播 hit 帧)_
3. 敌人周围是否有异常的白色方框/光晕？ _(ref: hit帧白方框)_
4. 敌人的血条是否正常显示在头顶？

### enemy_flying.png
![enemy_flying.png](enemy_flying.png)

**场景**: 敌人原型 'flying' 首次出现时的行走状态

**检查项**:
1. 敌人是否渲染为正常的精灵图像（而非占位符）？
2. 敌人是否处于正常的行走动画帧（而非 hit/受击 帧）？ _(ref: 怪物出现就播 hit 帧)_
3. 敌人周围是否有异常的白色方框/光晕？ _(ref: hit帧白方框)_
4. 敌人的血条是否正常显示在头顶？

### enemy_stealth.png
![enemy_stealth.png](enemy_stealth.png)

**场景**: 敌人原型 'stealth' 首次出现时的行走状态

**检查项**:
1. 敌人是否渲染为正常的精灵图像（而非占位符）？
2. 敌人是否处于正常的行走动画帧（而非 hit/受击 帧）？ _(ref: 怪物出现就播 hit 帧)_
3. 敌人周围是否有异常的白色方框/光晕？ _(ref: hit帧白方框)_
4. 敌人的血条是否正常显示在头顶？

### boss_wave_15.png
![boss_wave_15.png](boss_wave_15.png)

**检查项**:
1. 敌人是否渲染为正常的精灵图像（而非占位符）？
2. 敌人是否处于正常的行走动画帧（而非 hit/受击 帧）？ _(ref: 怪物出现就播 hit 帧)_
3. 敌人周围是否有异常的白色方框/光晕？ _(ref: hit帧白方框)_
4. 敌人的血条是否正常显示在头顶？

### enemy_splitter.png
![enemy_splitter.png](enemy_splitter.png)

**场景**: 敌人原型 'splitter' 首次出现时的行走状态

**检查项**:
1. 敌人是否渲染为正常的精灵图像（而非占位符）？
2. 敌人是否处于正常的行走动画帧（而非 hit/受击 帧）？ _(ref: 怪物出现就播 hit 帧)_
3. 敌人周围是否有异常的白色方框/光晕？ _(ref: hit帧白方框)_
4. 敌人的血条是否正常显示在头顶？

### enemy_buffer.png
![enemy_buffer.png](enemy_buffer.png)

**场景**: 敌人原型 'buffer' 首次出现时的行走状态

**检查项**:
1. 敌人是否渲染为正常的精灵图像（而非占位符）？
2. 敌人是否处于正常的行走动画帧（而非 hit/受击 帧）？ _(ref: 怪物出现就播 hit 帧)_
3. 敌人周围是否有异常的白色方框/光晕？ _(ref: hit帧白方框)_
4. 敌人的血条是否正常显示在头顶？

### enemy_teleporter.png
![enemy_teleporter.png](enemy_teleporter.png)

**场景**: 敌人原型 'teleporter' 首次出现时的行走状态

**检查项**:
1. 敌人是否渲染为正常的精灵图像（而非占位符）？
2. 敌人是否处于正常的行走动画帧（而非 hit/受击 帧）？ _(ref: 怪物出现就播 hit 帧)_
3. 敌人周围是否有异常的白色方框/光晕？ _(ref: hit帧白方框)_
4. 敌人的血条是否正常显示在头顶？

## 战灵 (2)

### warden_active.png
![warden_active.png](warden_active.png)

**场景**: 战灵激活后的首次画面

**检查项**:
1. 战灵精灵是否正常渲染（非占位符）？
2. 战灵的朝向是否与其移动方向一致？ _(ref: 战灵朝向不对)_
3. 战灵是否有可见的攻击动画/特效？
4. 战灵是否已离开原点、位于地图上的合理位置？ _(ref: 战灵出生闪现)_

### warden_combat.png
![warden_combat.png](warden_combat.png)

**场景**: 战灵正在战斗中（有敌人在附近）

**检查项**:
1. 战灵精灵是否正常渲染（非占位符）？
2. 战灵的朝向是否与其移动方向一致？ _(ref: 战灵朝向不对)_
3. 战灵是否有可见的攻击动画/特效？

## 场景全局 (36)

### start.png
![start.png](start.png)

**场景**: 游戏开始画面

**检查项**:
1. 整体画面是否正常渲染（无全黑/全白/严重撕裂）？
2. UI 元素（顶栏、按钮）是否可见且布局正常？

### anomaly_warden_range_limited_x_601.png
![anomaly_warden_range_limited_x_601.png](anomaly_warden_range_limited_x_601.png)

**检查项**:
1. 整体画面是否正常渲染（无全黑/全白/严重撕裂）？
2. UI 元素（顶栏、按钮）是否可见且布局正常？

### leak_wave_7_1900.png
![leak_wave_7_1900.png](leak_wave_7_1900.png)

**检查项**:
1. 整体画面是否正常渲染（无全黑/全白/严重撕裂）？
2. UI 元素（顶栏、按钮）是否可见且布局正常？

### leak_wave_7_1936.png
![leak_wave_7_1936.png](leak_wave_7_1936.png)

**检查项**:
1. 整体画面是否正常渲染（无全黑/全白/严重撕裂）？
2. UI 元素（顶栏、按钮）是否可见且布局正常？

### leak_wave_7_2008.png
![leak_wave_7_2008.png](leak_wave_7_2008.png)

**检查项**:
1. 整体画面是否正常渲染（无全黑/全白/严重撕裂）？
2. UI 元素（顶栏、按钮）是否可见且布局正常？

### leak_wave_7_2030.png
![leak_wave_7_2030.png](leak_wave_7_2030.png)

**检查项**:
1. 整体画面是否正常渲染（无全黑/全白/严重撕裂）？
2. UI 元素（顶栏、按钮）是否可见且布局正常？

### anomaly_economy_stall_2486.png
![anomaly_economy_stall_2486.png](anomaly_economy_stall_2486.png)

**检查项**:
1. 整体画面是否正常渲染（无全黑/全白/严重撕裂）？
2. UI 元素（顶栏、按钮）是否可见且布局正常？

### anomaly_build_silent_fail_2498.png
![anomaly_build_silent_fail_2498.png](anomaly_build_silent_fail_2498.png)

**检查项**:
1. 整体画面是否正常渲染（无全黑/全白/严重撕裂）？
2. UI 元素（顶栏、按钮）是否可见且布局正常？

### leak_wave_9_2508.png
![leak_wave_9_2508.png](leak_wave_9_2508.png)

**检查项**:
1. 整体画面是否正常渲染（无全黑/全白/严重撕裂）？
2. UI 元素（顶栏、按钮）是否可见且布局正常？

### leak_wave_9_2580.png
![leak_wave_9_2580.png](leak_wave_9_2580.png)

**检查项**:
1. 整体画面是否正常渲染（无全黑/全白/严重撕裂）？
2. UI 元素（顶栏、按钮）是否可见且布局正常？

### leak_wave_9_2584.png
![leak_wave_9_2584.png](leak_wave_9_2584.png)

**检查项**:
1. 整体画面是否正常渲染（无全黑/全白/严重撕裂）？
2. UI 元素（顶栏、按钮）是否可见且布局正常？

### leak_wave_9_2592.png
![leak_wave_9_2592.png](leak_wave_9_2592.png)

**检查项**:
1. 整体画面是否正常渲染（无全黑/全白/严重撕裂）？
2. UI 元素（顶栏、按钮）是否可见且布局正常？

### leak_wave_9_2618.png
![leak_wave_9_2618.png](leak_wave_9_2618.png)

**检查项**:
1. 整体画面是否正常渲染（无全黑/全白/严重撕裂）？
2. UI 元素（顶栏、按钮）是否可见且布局正常？

### leak_wave_9_2635.png
![leak_wave_9_2635.png](leak_wave_9_2635.png)

**检查项**:
1. 整体画面是否正常渲染（无全黑/全白/严重撕裂）？
2. UI 元素（顶栏、按钮）是否可见且布局正常？

### leak_wave_9_2643.png
![leak_wave_9_2643.png](leak_wave_9_2643.png)

**检查项**:
1. 整体画面是否正常渲染（无全黑/全白/严重撕裂）？
2. UI 元素（顶栏、按钮）是否可见且布局正常？

### leak_wave_9_2650.png
![leak_wave_9_2650.png](leak_wave_9_2650.png)

**检查项**:
1. 整体画面是否正常渲染（无全黑/全白/严重撕裂）？
2. UI 元素（顶栏、按钮）是否可见且布局正常？

### leak_wave_9_2674.png
![leak_wave_9_2674.png](leak_wave_9_2674.png)

**检查项**:
1. 整体画面是否正常渲染（无全黑/全白/严重撕裂）？
2. UI 元素（顶栏、按钮）是否可见且布局正常？

### leak_wave_9_2688.png
![leak_wave_9_2688.png](leak_wave_9_2688.png)

**检查项**:
1. 整体画面是否正常渲染（无全黑/全白/严重撕裂）？
2. UI 元素（顶栏、按钮）是否可见且布局正常？

### leak_wave_9_2694.png
![leak_wave_9_2694.png](leak_wave_9_2694.png)

**检查项**:
1. 整体画面是否正常渲染（无全黑/全白/严重撕裂）？
2. UI 元素（顶栏、按钮）是否可见且布局正常？

### leak_wave_9_2760.png
![leak_wave_9_2760.png](leak_wave_9_2760.png)

**检查项**:
1. 整体画面是否正常渲染（无全黑/全白/严重撕裂）？
2. UI 元素（顶栏、按钮）是否可见且布局正常？

### leak_wave_9_2794.png
![leak_wave_9_2794.png](leak_wave_9_2794.png)

**检查项**:
1. 整体画面是否正常渲染（无全黑/全白/严重撕裂）？
2. UI 元素（顶栏、按钮）是否可见且布局正常？

### leak_wave_9_2836.png
![leak_wave_9_2836.png](leak_wave_9_2836.png)

**检查项**:
1. 整体画面是否正常渲染（无全黑/全白/严重撕裂）？
2. UI 元素（顶栏、按钮）是否可见且布局正常？

### leak_wave_9_2860.png
![leak_wave_9_2860.png](leak_wave_9_2860.png)

**检查项**:
1. 整体画面是否正常渲染（无全黑/全白/严重撕裂）？
2. UI 元素（顶栏、按钮）是否可见且布局正常？

### leak_wave_9_2908.png
![leak_wave_9_2908.png](leak_wave_9_2908.png)

**检查项**:
1. 整体画面是否正常渲染（无全黑/全白/严重撕裂）？
2. UI 元素（顶栏、按钮）是否可见且布局正常？

### anomaly_build_silent_fail_2967.png
![anomaly_build_silent_fail_2967.png](anomaly_build_silent_fail_2967.png)

**检查项**:
1. 整体画面是否正常渲染（无全黑/全白/严重撕裂）？
2. UI 元素（顶栏、按钮）是否可见且布局正常？

### anomaly_build_silent_fail_3508.png
![anomaly_build_silent_fail_3508.png](anomaly_build_silent_fail_3508.png)

**检查项**:
1. 整体画面是否正常渲染（无全黑/全白/严重撕裂）？
2. UI 元素（顶栏、按钮）是否可见且布局正常？

### anomaly_projectile_orphan_burst_7994.png
![anomaly_projectile_orphan_burst_7994.png](anomaly_projectile_orphan_burst_7994.png)

**检查项**:
1. 整体画面是否正常渲染（无全黑/全白/严重撕裂）？
2. UI 元素（顶栏、按钮）是否可见且布局正常？

### anomaly_projectile_orphan_burst_8114.png
![anomaly_projectile_orphan_burst_8114.png](anomaly_projectile_orphan_burst_8114.png)

**检查项**:
1. 整体画面是否正常渲染（无全黑/全白/严重撕裂）？
2. UI 元素（顶栏、按钮）是否可见且布局正常？

### anomaly_projectile_orphan_burst_9271.png
![anomaly_projectile_orphan_burst_9271.png](anomaly_projectile_orphan_burst_9271.png)

**检查项**:
1. 整体画面是否正常渲染（无全黑/全白/严重撕裂）？
2. UI 元素（顶栏、按钮）是否可见且布局正常？

### anomaly_projectile_orphan_burst_9511.png
![anomaly_projectile_orphan_burst_9511.png](anomaly_projectile_orphan_burst_9511.png)

**检查项**:
1. 整体画面是否正常渲染（无全黑/全白/严重撕裂）？
2. UI 元素（顶栏、按钮）是否可见且布局正常？

### anomaly_projectile_orphan_burst_9707.png
![anomaly_projectile_orphan_burst_9707.png](anomaly_projectile_orphan_burst_9707.png)

**检查项**:
1. 整体画面是否正常渲染（无全黑/全白/严重撕裂）？
2. UI 元素（顶栏、按钮）是否可见且布局正常？

### anomaly_projectile_orphan_burst_9827.png
![anomaly_projectile_orphan_burst_9827.png](anomaly_projectile_orphan_burst_9827.png)

**检查项**:
1. 整体画面是否正常渲染（无全黑/全白/严重撕裂）？
2. UI 元素（顶栏、按钮）是否可见且布局正常？

### anomaly_projectile_orphan_burst_9947.png
![anomaly_projectile_orphan_burst_9947.png](anomaly_projectile_orphan_burst_9947.png)

**检查项**:
1. 整体画面是否正常渲染（无全黑/全白/严重撕裂）？
2. UI 元素（顶栏、按钮）是否可见且布局正常？

### anomaly_projectile_orphan_burst_10067.png
![anomaly_projectile_orphan_burst_10067.png](anomaly_projectile_orphan_burst_10067.png)

**检查项**:
1. 整体画面是否正常渲染（无全黑/全白/严重撕裂）？
2. UI 元素（顶栏、按钮）是否可见且布局正常？

### anomaly_projectile_orphan_burst_10187.png
![anomaly_projectile_orphan_burst_10187.png](anomaly_projectile_orphan_burst_10187.png)

**检查项**:
1. 整体画面是否正常渲染（无全黑/全白/严重撕裂）？
2. UI 元素（顶栏、按钮）是否可见且布局正常？

### anomaly_projectile_orphan_burst_10307.png
![anomaly_projectile_orphan_burst_10307.png](anomaly_projectile_orphan_burst_10307.png)

**检查项**:
1. 整体画面是否正常渲染（无全黑/全白/严重撕裂）？
2. UI 元素（顶栏、按钮）是否可见且布局正常？

