// ── ZH Translation Table ──
const ZH = {
  // Enemy archetypes
  normal: '步兵', runner: '疾行', swarm: '虫群', tank: '重甲', shielded: '护盾兵',
  medic: '治疗兵', regenerator: '回血体', berserker: '狂热兵', elite: '精英', boss: 'Boss',
  stealth: '隐身兵', splitter: '分裂体', banner: '旗手', armored: '护甲兵',
  timewarp: '净化者', teleporter: '传送兵', mirror: '镜像兵', devoter: '奉献者',
  reflector: '反射兵', ironhide: '铁皮兵', colossus: '巨像', steadfast: '稳行者', ironwill: '铁意兵',
  healer: '治疗兵', buffer: '旗手', flying: '飞行斥候', dummy: '木桩',
  // Faction enemies
  'st-plating': '附甲工兵', 'st-bunker': '掩体车', 'st-overload': '过载体', 'st-welder': '焊接兵', 'st-magnetize': '磁化兵',
  'en-phase': '相位兵', 'en-disrupt': '干扰者', 'en-overcharge': '超载体', 'en-siphon': '虹吸兵', 'en-flicker': '闪烁虫',
  'na-spore': '孢子母体', 'na-bloom': '绽放体', 'na-rootwalker': '藤行者', 'na-seedling': '种子兵', 'na-symbiont': '共生体',
  'or-phalanx': '方阵兵', 'or-martyr': '殉道者', 'or-warden': '守望者', 'or-inquisitor': '审判官', 'or-herald': '传令官',
  'su-hive': '母巢虫', 'su-revenant': '亡魂兵', 'su-cocoon': '虫茧', 'su-leech': '寄附虫', 'su-totem': '图腾兽',
  'ch-backlash': '反噬体', 'ch-drain': '汲魂兵', 'ch-goldrot': '蚀金虫', 'ch-corruptor': '腐化先驱', 'ch-abyssal': '深渊行者',
  'fly-scout': '飞行斥候', 'fly-heavy': '飞行重甲', 'fly-dart': '疾风翼', 'fly-shield': '飞盾卫',
  'fly-ghost': '幽灵翼', 'fly-bomber': '投弹手', 'fly-medic': '飞行奶妈', 'fly-split': '裂翼虫',
  'boss-iron': '铁壁将军', 'boss-storm': '风暴领主', 'boss-grove': '生命古树', 'boss-legion': '军团统帅',
  'boss-hive': '虫潮之母', 'boss-shadow': '暗影主宰', 'boss-abyss': '虚空巨兽', 'boss-chaos': '混沌化身',
  'boss-sky': '天空霸主', 'boss-omega': '终焉审判',
  // Tower types
  basic: '基础塔', laser: '激光塔', freeze: '冰冻塔', targeter: '增益塔',
  judicator: '裁决塔', poison: '毒素塔', electric: '电击塔', hunter: '猎手塔',
  // Attack modes
  balanced: '均衡', rapid: '速射', sniper: '重炮',
  // Attack styles
  projectile: '投射物', aura_dot: '范围持续', summon: '召唤',
  wideBeam: '宽束', scatter: '散射', charge: '蓄力', spin_aoe: '旋转AOE', pierce: '贯穿',
  // Ability types
  bounce: '弹射', stackDamage: '叠加伤害', splash: '溅射',
  distanceDamage: '距离伤害', percentHpDamage: '百分比血量伤害', executionBonus: '斩杀加成',
  onHitSlow: '命中减速', flatDamage: '固定伤害', burn: '灼烧', bleed: '流血',
  stun: '眩晕', shieldIgnore: '忽略护盾', instantHit: '瞬时命中',
  // Field names
  hpScale: 'HP倍率', speedScale: '速度倍率', rewardScale: '奖励倍率',
  radius: '体型', shieldScale: '护盾倍率', stealthDuration: '隐身时长',
  splitCount: '分裂数', teleportInterval: '传送间隔', healScale: '治疗倍率',
  healRadius: '治疗半径', healInterval: '治疗间隔', auraRange: '光环范围',
  auraSpeedUp: '光环加速', auraArmor: '光环护甲', movementType: '移动方式',
  buildCost: '建造费', baseRange: '基础射程', baseDamage: '基础伤害',
  baseFireRate: '基础攻速', projectileSpeed: '弹速', attackStyle: '攻击方式',
  // Difficulty
  easy: '简单', normal: '普通', hard: '困难', extreme: '极限',
  // Mode modifier fields
  range: '射程', damage: '伤害', fireRate: '攻速',
  // Tower factions
  base: '基础', steel: '钢铁', energy: '能量', nature: '自然',
  order: '秩序', summon: '召唤', chaos: '混沌', other: '其他',
};
function t(key) { return ZH[key] || key; }

// ── Utilities ──
const $ = (sel) => document.querySelector(sel);
const $$ = (sel) => document.querySelectorAll(sel);
function escapeHtml(v) {
  const d = document.createElement('div');
  d.textContent = String(v ?? '');
  return d.innerHTML;
}
function q(text) { return String(text ?? '').toLowerCase(); }
function matchesSearch(...parts) {
  if (!searchText) return true;
  const haystack = parts.map(q).join(' ');
  return searchText.split(/\s+/).every(w => haystack.includes(w));
}
function fmt(v) {
  if (v == null) return '-';
  if (typeof v === 'boolean') return v ? '是' : '否';
  if (typeof v === 'number') return Number.isInteger(v) ? String(v) : v.toFixed(2);
  return String(v);
}

// ── State ──
let config = null;
let searchText = '';
let activeTab = 'overview';

// ── Config Loader ──
async function loadConfig() {
  const base = '../../config';
  const [enemies, towers, settings, allyEvents, enemyEvents] = await Promise.all([
    fetch(`${base}/enemies/enemies-core.json`).then(r => r.json()),
    fetch(`${base}/towers/towers.json`).then(r => r.json()),
    fetch(`${base}/settings.json`).then(r => r.json()),
    fetch(`${base}/events/events-ally-v2.json`).then(r => r.json()),
    fetch(`${base}/events/events-enemy-v2.json`).then(r => r.json()),
  ]);
  const stripMeta = obj => Object.fromEntries(
    Object.entries(obj).filter(([k]) => !k.startsWith('_'))
  );
  return {
    enemies: { types: stripMeta(enemies), specialHints: settings.enemies?.specialHints || {} },
    towers: { types: stripMeta(towers), defaultType: settings.towers?.defaultType || 'laser' },
    economy: settings.economy,
    waves: settings.waves,
    difficulty: settings.difficulty,
    world: settings.world,
    allyEventPool: allyEvents,
    enemyEventPool: enemyEvents,
    combat: settings.combat,
  };
}

// ── Asset Paths ──
function towerAsset(key) { return `../../assets/towers/${key}/tower-${key}.png`; }
function towerFrame(key, state, n) { return `../../assets/towers/${key}/tower-${key}-${state}-${n}.png`; }
function enemyAsset(key) { return `../../assets/enemies/${key}.png`; }
function enemyFrame(key, state, n) { return `../../assets/enemies/${key}-${state}-${n}.png`; }

// ── Display Helpers ──
function displayField(label, value) {
  return `<label class="field"><span>${escapeHtml(label)}</span><span class="field-value">${escapeHtml(fmt(value))}</span></label>`;
}
function imgTag(src, size = 92) {
  return `<img src="${src}" width="${size}" height="${size}" style="object-fit:contain" onerror="this.style.opacity=0.15">`;
}
function spriteStrip(urls, size = 40) {
  return `<div class="gallery-sprites">${urls.map(u => `<img src="${u}" width="${size}" height="${size}" onerror="this.style.opacity=0.15">`).join('')}</div>`;
}

// ══════════════════════════════════════
// BUILD FUNCTIONS
// ══════════════════════════════════════

function buildOverview() {
  const el = $('#overview');
  const enemyCount = Object.keys(config.enemies.types).length;
  const towerCount = Object.keys(config.towers.types).length;
  const allyEventCount = Array.isArray(config.allyEventPool) ? config.allyEventPool.length : 0;
  const enemyEventCount = Array.isArray(config.enemyEventPool) ? config.enemyEventPool.length : 0;
  const eco = config.economy;
  const w = config.world;

  el.innerHTML = `
    <div class="card">
      <h2>总览</h2>
      <div class="kpi-grid" style="margin-top:12px">
        <div class="kpi"><div class="kpi-label">敌人原型</div><div class="kpi-value">${enemyCount}</div></div>
        <div class="kpi"><div class="kpi-label">炮塔类型</div><div class="kpi-value">${towerCount}</div></div>
        <div class="kpi"><div class="kpi-label">友方事件</div><div class="kpi-value">${allyEventCount}</div></div>
        <div class="kpi"><div class="kpi-label">敌方事件</div><div class="kpi-value">${enemyEventCount}</div></div>
        <div class="kpi"><div class="kpi-label">路径点</div><div class="kpi-value">${w?.path?.length ?? 0}</div></div>
        <div class="kpi"><div class="kpi-label">塔位</div><div class="kpi-value">${w?.towerSlots?.length ?? 0}</div></div>
      </div>
    </div>
    <div class="card">
      <h3>经济</h3>
      <div class="stat-grid" style="margin-top:10px">
        ${displayField('初始金币', eco?.startingGold)}
        ${displayField('初始生命', eco?.startingLives)}
        ${displayField('胜利波次', eco?.victoryWaveTarget)}
        ${displayField('建造费用', eco?.buildCost)}
        ${displayField('升级费用', eco?.baseUpgradeCost)}
        ${displayField('卖出退款', eco?.sellRefundRate)}
      </div>
    </div>
    <div class="card">
      <h3>难度模式</h3>
      <div class="diff-grid" style="margin-top:10px">
        ${Object.entries(config.difficulty?.modes || {}).map(([id, m]) => `
          <div class="diff-card">
            <h4>${escapeHtml(m.label || id)}</h4>
            <div class="stat">生命: <span>${m.hpScale}x</span></div>
            <div class="stat">速度: <span>${m.speedScale}x</span></div>
            <div class="stat">奖励: <span>${m.rewardScale}x</span></div>
            <div class="stat">初始金币: <span>${m.startGold}</span></div>
          </div>
        `).join('')}
      </div>
    </div>
  `;
}

function buildEnemies() {
  const el = $('#enemies');
  const entries = Object.entries(config.enemies.types);
  const hints = config.enemies.specialHints || {};

  const cards = entries.filter(([key, e]) =>
    matchesSearch(key, e.label, e.description, t(key))
  ).map(([key, e]) => {
    const specialFields = [];
    if (e.shieldScale != null) specialFields.push(displayField('护盾', `${e.shieldScale}x`));
    if (e.stealthDuration != null) specialFields.push(displayField('隐身', `${e.stealthDuration}s`));
    if (e.splitCount != null) specialFields.push(displayField('分裂', `x${e.splitCount}`));
    if (e.teleportInterval != null) specialFields.push(displayField('传送', `${e.teleportInterval}s`));
    if (e.healScale != null) specialFields.push(displayField('治疗', `${e.healScale}x r=${e.healRadius}`));
    if (e.auraRange != null) specialFields.push(displayField('光环', `r=${e.auraRange} 速+${e.auraSpeedUp} 甲+${e.auraArmor}`));
    if (e.movementType) specialFields.push(displayField('移动', e.movementType));

    const tags = [];
    if (key.startsWith('boss-')) tags.push('<span class="tag bad">Boss</span>');
    if (key.startsWith('fly-') || e.movementType === 'flying') tags.push('<span class="tag info">飞行</span>');
    if (e.speedScale >= 1.5) tags.push('<span class="tag warn">高速</span>');
    if (e.hpScale >= 2) tags.push('<span class="tag good">肉盾</span>');

    const hint = hints[key] ? `<div class="muted" style="margin-top:6px;font-size:12px">${escapeHtml(hints[key])}</div>` : '';

    return `
      <div class="card entity-card">
        <div class="card-header">
          <h3 style="color:${e.color || '#fff'}">${escapeHtml(e.label || key)} <span class="muted mono" style="font-size:12px">${key}</span></h3>
          <div class="tag-row">${tags.join('')}</div>
        </div>
        <div class="entity-head">
          <div class="preview-box">${imgTag(enemyAsset(key))}</div>
          <div>
            <div class="muted" style="font-size:12px;margin-bottom:8px">${escapeHtml(e.description || '')}</div>
            <div class="stat-grid">
              ${displayField('生命', `${e.hpScale}x`)}
              ${displayField('速度', `${e.speedScale}x`)}
              ${displayField('奖励', `${e.rewardScale}x`)}
              ${displayField('体型', e.radius)}
              ${specialFields.join('')}
            </div>
            ${hint}
          </div>
        </div>
      </div>
    `;
  });

  el.innerHTML = cards.length ? cards.join('') : '<div class="card empty-state">没有匹配的敌人</div>';
}

function buildTowers() {
  const el = $('#towers');
  const entries = Object.entries(config.towers.types);

  const cards = entries.filter(([key, tw]) =>
    matchesSearch(key, tw.label, tw.description, t(key), (tw.tags || []).join(' '))
  ).map(([key, tw]) => {
    const tags = (tw.tags || []).map(tag => `<span class="tag info">${escapeHtml(t(tag))}</span>`).join('');
    const styleTag = tw.attackStyle ? `<span class="tag">${escapeHtml(t(tw.attackStyle))}</span>` : '';
    const modeTag = tw.supportsModeSwitch ? '<span class="tag good">模式切换</span>' : '';

    // Mode modifiers table
    let modeTable = '';
    if (tw.modeModifiers) {
      const modes = Object.keys(tw.modeModifiers);
      const fields = ['range', 'damage', 'fireRate'];
      modeTable = `
        <h4 style="margin-top:12px">模式修正</h4>
        <table class="mode-table">
          <tr><th></th>${modes.map(m => `<th>${escapeHtml(t(m))}</th>`).join('')}</tr>
          ${fields.map(f => `<tr><td class="row-label">${escapeHtml(t(f))}</td>${modes.map(m => `<td>${fmt(tw.modeModifiers[m]?.[f])}</td>`).join('')}</tr>`).join('')}
        </table>
      `;
    }

    // Abilities
    let abilitiesHtml = '';
    if (tw.abilities && tw.abilities.length) {
      abilitiesHtml = `<div class="ability-chips" style="margin-top:8px">${tw.abilities.map(a =>
        `<span class="ability-chip">${escapeHtml(a.type || a.name || JSON.stringify(a))}</span>`
      ).join('')}</div>`;
    }

    // Ability unlocks
    let unlocksHtml = '';
    if (tw.abilityUnlocks && tw.abilityUnlocks.length) {
      unlocksHtml = `<div class="ability-chips" style="margin-top:4px">${tw.abilityUnlocks.map(u =>
        `<span class="ability-chip">Lv${u.level}: ${escapeHtml(u.name || u.type)}</span>`
      ).join('')}</div>`;
    }

    // Beam info
    let beamHtml = '';
    if (tw.beam) {
      beamHtml = `<div style="margin-top:8px"><span class="tag" style="border-color:${tw.beam.color}; color:${tw.beam.glow || tw.beam.color}">光束: ${tw.beam.duration}s 宽=${tw.beam.width}</span></div>`;
    }

    // Branches
    let branchHtml = '';
    if (tw.branches) {
      branchHtml = Object.entries(tw.branches).map(([bk, br]) =>
        `<div style="margin-top:8px"><span class="tag good">${escapeHtml(br.label || bk)}</span> <span class="muted" style="font-size:12px">${escapeHtml(br.description || '')}</span></div>`
      ).join('');
    }

    return `
      <div class="card entity-card">
        <div class="card-header">
          <h3>${escapeHtml(tw.label || key)} <span class="muted mono" style="font-size:12px">${key}</span></h3>
          <div class="tag-row">${tags} ${styleTag} ${modeTag}</div>
        </div>
        <div class="entity-head">
          <div class="preview-box">${imgTag(towerAsset(key))}</div>
          <div>
            <div class="muted" style="font-size:12px;margin-bottom:8px">${escapeHtml(tw.description || '')}</div>
            <div class="stat-grid">
              ${displayField('费用', tw.buildCost)}
              ${displayField('射程', tw.baseRange)}
              ${displayField('伤害', tw.baseDamage)}
              ${displayField('攻速', tw.baseFireRate)}
              ${displayField('弹速', tw.projectileSpeed)}
              ${displayField('固定模式', tw.fixedMode ? t(tw.fixedMode) : '-')}
            </div>
          </div>
        </div>
        ${modeTable}
        ${abilitiesHtml}
        ${unlocksHtml}
        ${beamHtml}
        ${branchHtml}
      </div>
    `;
  });

  el.innerHTML = cards.length ? cards.join('') : '<div class="card empty-state">没有匹配的炮塔</div>';
}

function buildWaves() {
  const el = $('#waves');
  const w = config.waves;
  const d = w?.difficulty || {};

  // Spawn multipliers table
  const mult = w?.spawnMultipliers || {};
  const multKeys = Object.keys(mult).sort((a, b) => mult[a] - mult[b]);

  el.innerHTML = `
    <div class="card">
      <h3>波次难度曲线</h3>
      <div class="stat-grid" style="margin-top:10px">
        ${displayField('基础HP', d.hpBase)}
        ${displayField('每波HP增量', d.hpPerWave)}
        ${displayField('基础速度', d.speedBase)}
        ${displayField('每波速度增量', d.speedPerWave)}
        ${displayField('基础奖励', d.rewardBase)}
        ${displayField('每波奖励增量', d.rewardPerWave)}
        ${displayField('后期波次起始', d.lateWaveStart)}
        ${displayField('后期HP加成/波', d.lateHpBonusPerWave)}
        ${displayField('后期速度加成/波', d.lateSpeedBonusPerWave)}
        ${displayField('后期奖励衰减/波', d.lateRewardPenaltyPerWave)}
      </div>
    </div>
    <div class="card">
      <h3>生成设置</h3>
      <div class="stat-grid" style="margin-top:10px">
        ${displayField('自动开波', w?.autoStart)}
        ${displayField('波间间隔(秒)', w?.intermissionSeconds)}
        ${displayField('基础间隔', w?.spawnBaseInterval)}
        ${displayField('最小间隔', w?.spawnMinInterval)}
        ${displayField('每波衰减', w?.spawnDecayPerWave)}
      </div>
    </div>
    <div class="card">
      <h3>生成倍率</h3>
      <table class="spawn-table" style="margin-top:10px">
        <tr><th>类型</th><th>倍率</th></tr>
        ${multKeys.map(k => `<tr><td class="row-label">${escapeHtml(t(k))}<span class="muted"> ${k}</span></td><td>${fmt(mult[k])}</td></tr>`).join('')}
      </table>
    </div>
    <div class="card">
      <h3>波次组成</h3>
      <div class="stat-grid" style="margin-top:10px">
        ${displayField('阶段阈值', JSON.stringify(w?.tierThresholds))}
        ${displayField('每阶最大Buff', JSON.stringify(w?.maxBuffsPerTier))}
        ${displayField('基础数量', `${w?.baseTotalFormula?.base} + ${w?.baseTotalFormula?.perWave}/波`)}
        ${displayField('普通加强上限', w?.normalBoostCap)}
        ${displayField('精英间隔', w?.eliteInterval)}
        ${displayField('压力基数', w?.pressureBase)}
      </div>
    </div>
  `;
}

function buildTowerGallery() {
  const el = $('#towerGallery');
  const entries = Object.entries(config.towers.types);

  // Group by first tag (faction)
  const groups = {};
  for (const [key, tw] of entries) {
    if (!matchesSearch(key, tw.label, (tw.tags || []).join(' '))) continue;
    const faction = tw.tags?.[0] || 'other';
    (groups[faction] = groups[faction] || []).push([key, tw]);
  }

  const factionColors = {
    base: '#94a3b8', steel: '#78716c', energy: '#60a5fa', nature: '#4ade80',
    order: '#fbbf24', summon: '#c084fc', chaos: '#f87171', other: '#9ca3af',
  };

  let html = '';
  for (const [faction, items] of Object.entries(groups)) {
    const color = factionColors[faction] || '#9ca3af';
    html += `
      <div class="gallery-faction">
        <div class="gallery-faction-header">
          <div style="width:12px;height:12px;border-radius:50%;background:${color}"></div>
          <h3>${escapeHtml(t(faction))}</h3>
          <span class="muted">${items.length} 座</span>
        </div>
        <div class="gallery-grid">
          ${items.map(([key, tw]) => {
            const frames = [
              towerFrame(key, 'idle', 0), towerFrame(key, 'idle', 1),
              towerFrame(key, 'attack', 0), towerFrame(key, 'attack', 1), towerFrame(key, 'attack', 2),
            ];
            const tagHtml = (tw.tags || []).map(tag => `<span>${escapeHtml(t(tag))}</span>`).join('');
            return `
              <div class="gallery-card">
                <img class="gallery-img-lg" src="${towerAsset(key)}" onerror="this.style.opacity=0.15">
                <div class="gallery-info">
                  <div class="name">${escapeHtml(tw.label || key)}</div>
                  <div class="key">${key}</div>
                  <div class="desc">${escapeHtml(tw.description || '')}</div>
                  <div class="tags">${tagHtml}</div>
                  <div style="margin-top:6px">${spriteStrip(frames, 36)}</div>
                </div>
              </div>
            `;
          }).join('')}
        </div>
      </div>
    `;
  }

  el.innerHTML = html || '<div class="card empty-state">没有匹配的炮塔</div>';
}

function buildEnemyGallery() {
  const el = $('#enemyGallery');
  const entries = Object.entries(config.enemies.types);

  // Group by key prefix
  const factionMap = {
    'st-': { name: '钢铁', color: '#78716c' },
    'en-': { name: '能量', color: '#60a5fa' },
    'na-': { name: '自然', color: '#4ade80' },
    'or-': { name: '秩序', color: '#fbbf24' },
    'su-': { name: '召唤', color: '#c084fc' },
    'ch-': { name: '混沌', color: '#f87171' },
    'fly-': { name: '飞行', color: '#93c5fd' },
    'boss-': { name: 'Boss', color: '#ef4444' },
  };

  const groups = { '基础': [] };
  for (const [key, e] of entries) {
    if (!matchesSearch(key, e.label, e.description, t(key))) continue;
    let placed = false;
    for (const [prefix, info] of Object.entries(factionMap)) {
      if (key.startsWith(prefix)) {
        (groups[info.name] = groups[info.name] || []).push([key, e]);
        placed = true;
        break;
      }
    }
    if (!placed) groups['基础'].push([key, e]);
  }

  const groupColors = { '基础': '#94a3b8' };
  for (const [, info] of Object.entries(factionMap)) groupColors[info.name] = info.color;

  let html = '';
  for (const [groupName, items] of Object.entries(groups)) {
    if (!items.length) continue;
    const color = groupColors[groupName] || '#9ca3af';
    html += `
      <div class="gallery-faction">
        <div class="gallery-faction-header">
          <div style="width:12px;height:12px;border-radius:50%;background:${color}"></div>
          <h3>${escapeHtml(groupName)}</h3>
          <span class="muted">${items.length} 只</span>
        </div>
        <div class="gallery-grid">
          ${items.map(([key, e]) => {
            const frames = [
              enemyFrame(key, 'walk', 0), enemyFrame(key, 'walk', 1),
              enemyFrame(key, 'walk', 2), enemyFrame(key, 'walk', 3),
              enemyFrame(key, 'hit', 0), enemyFrame(key, 'hit', 1),
            ];
            const roleTags = [];
            if (key.startsWith('boss-')) roleTags.push('Boss');
            if (e.movementType === 'flying') roleTags.push('飞行');
            if (e.speedScale >= 1.5) roleTags.push('高速');
            if (e.hpScale >= 2) roleTags.push('肉盾');
            if (e.shieldScale) roleTags.push('护盾');
            if (e.splitCount) roleTags.push('分裂');
            if (e.stealthDuration) roleTags.push('隐身');
            if (e.healScale) roleTags.push('治疗');
            const tagHtml = roleTags.map(tag => `<span>${tag}</span>`).join('');
            return `
              <div class="gallery-card">
                <img class="gallery-img-lg" src="${enemyAsset(key)}" onerror="this.style.opacity=0.15">
                <div class="gallery-info">
                  <div class="name" style="color:${e.color || '#fff'}">${escapeHtml(e.label || key)}</div>
                  <div class="key">${key}</div>
                  <div class="stat-line">生命 ${e.hpScale}x / 速度 ${e.speedScale}x / 体型 ${e.radius}</div>
                  <div class="tags">${tagHtml}</div>
                  <div style="margin-top:6px">${spriteStrip(frames, 32)}</div>
                </div>
              </div>
            `;
          }).join('')}
        </div>
      </div>
    `;
  }

  el.innerHTML = html || '<div class="card empty-state">没有匹配的敌人</div>';
}

// ══════════════════════════════════════
// TAB NAVIGATION
// ══════════════════════════════════════

const builders = {
  overview: buildOverview,
  enemies: buildEnemies,
  towers: buildTowers,
  waves: buildWaves,
  towerGallery: buildTowerGallery,
  enemyGallery: buildEnemyGallery,
};

function renderPanel(name) {
  if (builders[name]) builders[name]();
}

function activateTab(name) {
  activeTab = name;
  $$('.tab').forEach(btn => btn.classList.toggle('active', btn.dataset.tab === name));
  $$('.panel').forEach(p => p.classList.toggle('active', p.id === name));
  renderPanel(name);
}

function setStatus(text) { $('#status').textContent = text; }

// ── Search ──
$('#globalSearch').addEventListener('input', (e) => {
  searchText = q(e.target.value);
  renderPanel(activeTab);
});

// ── Tab clicks ──
$$('.tab').forEach(btn => btn.addEventListener('click', () => activateTab(btn.dataset.tab)));

// ── Boot ──
async function boot() {
  setStatus('加载配置中...');
  try {
    config = await loadConfig();
    renderPanel('overview');
    setStatus(`已加载: ${Object.keys(config.enemies.types).length} 种敌人, ${Object.keys(config.towers.types).length} 种炮塔`);
  } catch (e) {
    setStatus('加载失败: ' + e.message);
    console.error(e);
  }
}
boot();
