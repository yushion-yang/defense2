/**
 * enemy.mjs — Enemy SVG multi-frame generator.
 *
 * Reads config/visuals/enemies.json and produces multiple SVG frames per enemy:
 *   walk-0..3 (walking cycle), hit-0..1 (damage flash).
 *
 * SVG viewBox: 0 0 128 128.
 */

let _gradId = 0;
function nextGradId() { return `eg${_gradId++}`; }

function radialGradientSvg(id, cx, cy, color, opacity) {
  return `<radialGradient id="${id}" cx="50%" cy="50%" r="50%">` +
    `<stop offset="0%" stop-color="${color}" stop-opacity="${opacity}"/>` +
    `<stop offset="100%" stop-color="${color}" stop-opacity="0"/>` +
    `</radialGradient>`;
}

// ── Body renderers ──

function renderBody(body, palette, offsetY = 0) {
  if (!body) return '';
  const cy = (body.cy || 64) + offsetY;

  switch (body.type) {
    case 'circle':
      return `<circle cx="${body.cx || 64}" cy="${cy}" r="${body.r || 32}" fill="${palette.primary}"/>` +
        `<circle cx="${body.cx || 64}" cy="${cy}" r="${(body.r || 32) - 6}" fill="${palette.secondary}" opacity="0.5"/>`;
    case 'ellipse':
      return `<ellipse cx="${body.cx || 64}" cy="${cy}" rx="${body.rx || 36}" ry="${body.ry || 24}" fill="${palette.primary}"/>`;
    case 'diamond': {
      const cx = body.cx || 64;
      const s = body.size || 40;
      const hs = s / 2;
      return `<polygon points="${cx},${cy - hs} ${cx + hs},${cy} ${cx},${cy + hs} ${cx - hs},${cy}" fill="${palette.primary}"/>`;
    }
    case 'rounded-rect': {
      const cx = body.cx || 64;
      const s = body.size || 50;
      const hs = s / 2;
      return `<rect x="${cx - hs}" y="${cy - hs}" width="${s}" height="${s}" rx="10" fill="${palette.primary}"/>`;
    }
    default:
      return `<circle cx="${body.cx || 64}" cy="${cy}" r="${body.r || 28}" fill="${palette.primary}"/>`;
  }
}

function renderHead(head, palette, offsetY = 0) {
  if (!head) return '';
  const cy = (head.cy || 38) + offsetY;

  switch (head.type) {
    case 'circle':
      return `<circle cx="${head.cx || 64}" cy="${cy}" r="${head.r || 14}" fill="${palette.primary}"/>` +
        `<circle cx="${head.cx || 64}" cy="${cy}" r="${(head.r || 14) - 3}" fill="${palette.secondary}" opacity="0.4"/>`;
    case 'pointed':
      return `<polygon points="${head.cx || 64},${cy - 12} ${(head.cx || 64) + 10},${cy + 6} ${(head.cx || 64) - 10},${cy + 6}" fill="${palette.primary}"/>`;
    case 'angular':
      return `<polygon points="${head.cx || 64},${cy - 14} ${(head.cx || 64) + 16},${cy + 4} ${(head.cx || 64) - 16},${cy + 4}" fill="${palette.secondary}"/>`;
    default:
      return '';
  }
}

function renderDecorations(decorations, palette, rotateAngle = 0) {
  if (!decorations || decorations.length === 0) return '';

  return decorations.map(deco => {
    switch (deco.type) {
      case 'eyes': {
        return deco.positions.map(p =>
          `<circle cx="${p.cx}" cy="${p.cy}" r="${deco.r || 3}" fill="${deco.color || '#fff'}"/>` +
          `<circle cx="${p.cx + 1}" cy="${p.cy}" r="${deco.pupilR || 1.5}" fill="${deco.pupilColor || '#000'}"/>`
        ).join('\n  ');
      }
      case 'speed-lines': {
        return (deco.lines || []).map(l =>
          `<line x1="${l.x1}" y1="${l.y1}" x2="${l.x2}" y2="${l.y2}" stroke="${palette.accent}" stroke-width="${l.w || 1.5}" stroke-linecap="round" opacity="0.5"/>`
        ).join('\n  ');
      }
      case 'armor-plates': {
        return (deco.rects || []).map(r =>
          `<rect x="${r.x}" y="${r.y}" width="${r.w}" height="${r.h}" rx="2" fill="${deco.color || palette.accent}" opacity="${deco.opacity || 0.6}"/>`
        ).join('\n  ');
      }
      case 'shield': {
        const r = deco.rect || deco;
        const x = r.x || 0, y = r.y || 0, w = r.width || r.w || 20, h = r.height || r.h || 30;
        const rx = r.rx || 4;
        const color = deco.gradient ? palette.accent : (deco.color || palette.accent);
        return `<rect x="${x}" y="${y}" width="${w}" height="${h}" rx="${rx}" fill="${color}" opacity="${deco.opacity || 0.8}"/>`;
      }
      case 'shield-emblem': {
        const c = deco.cross;
        if (!c) return '';
        const hz = c.horizontal || {};
        const vt = c.vertical || {};
        const color = deco.color || '#fff';
        const op = deco.opacity || 0.7;
        return `<rect x="${hz.x || 0}" y="${hz.y || 0}" width="${hz.width || 14}" height="${hz.height || 4}" rx="${hz.rx || 1}" fill="${color}" opacity="${op}"/>` +
          `<rect x="${vt.x || 0}" y="${vt.y || 0}" width="${vt.width || 4}" height="${vt.height || 22}" rx="${vt.rx || 1}" fill="${color}" opacity="${op}"/>`;
      }
      case 'antennae': {
        return (deco.lines || []).map(l =>
          `<line x1="${l.x1}" y1="${l.y1}" x2="${l.x2}" y2="${l.y2}" stroke="${palette.primary}" stroke-width="2" stroke-linecap="round"/>` +
          `<circle cx="${l.x2}" cy="${l.y2}" r="2" fill="${palette.accent}"/>`
        ).join('\n  ');
      }
      case 'cloak': {
        return `<circle cx="${deco.cx || 64}" cy="${deco.cy || 64}" r="${deco.r || 36}" fill="none" stroke="${palette.accent}" stroke-width="1.5" stroke-dasharray="${deco.dasharray || '6 4'}" opacity="0.4"/>`;
      }
      case 'cracks': {
        return (deco.paths || []).map(p =>
          `<path d="${p}" fill="none" stroke="${palette.accent}" stroke-width="2" opacity="0.6"/>`
        ).join('\n  ');
      }
      case 'ring': {
        return `<circle cx="${deco.cx || 64}" cy="${deco.cy || 64}" r="${deco.r || 40}" fill="none" stroke="${palette.accent}" stroke-width="1.5" stroke-dasharray="${deco.dasharray || '5 3'}" opacity="${deco.opacity || 0.5}"/>`;
      }
      case 'cross': {
        const cx = deco.cx || 64;
        const cy = deco.cy || 64;
        const s = deco.size || 16;
        const w = deco.width || 6;
        return `<rect x="${cx - w / 2}" y="${cy - s / 2}" width="${w}" height="${s}" rx="2" fill="${palette.accent}" opacity="0.7"/>` +
          `<rect x="${cx - s / 2}" y="${cy - w / 2}" width="${s}" height="${w}" rx="2" fill="${palette.accent}" opacity="0.7"/>`;
      }
      case 'flag': {
        const pole = deco.pole || {};
        const banner = deco.banner || {};
        let result = `<line x1="${pole.x1 || 80}" y1="${pole.y1 || 28}" x2="${pole.x2 || 80}" y2="${pole.y2 || 56}" stroke="${pole.stroke || palette.dark}" stroke-width="${pole.strokeWidth || 2}"/>`;
        if (banner.points) {
          const pts = Array.isArray(banner.points) ? banner.points.join(',') : banner.points;
          result += `<polygon points="${pts}" fill="${banner.color || palette.accent}" opacity="0.8"/>`;
        }
        return result;
      }
      case 'wings': {
        return (deco.paths || []).map(p =>
          `<path d="${p}" fill="${palette.accent}" opacity="${deco.opacity || 0.6}"/>`
        ).join('\n  ');
      }
      case 'crown': {
        return (deco.triangles || []).map(t =>
          `<polygon points="${t.join(',')}" fill="${deco.color || '#eab308'}" opacity="0.9"/>`
        ).join('\n  ');
      }
      case 'aura': {
        return `<circle cx="${deco.cx || 64}" cy="${deco.cy || 64}" r="${deco.r || 46}" fill="none" stroke="${palette.glow || palette.accent}" stroke-width="2" opacity="0.3"/>`;
      }
      case 'rivets':
      case 'dots': {
        return (deco.positions || []).map(p =>
          `<circle cx="${p.cx}" cy="${p.cy}" r="${p.r || 2}" fill="${deco.color || palette.accent}" opacity="${deco.opacity || 0.6}"/>`
        ).join('\n  ');
      }
      case 'armor-band':
      case 'armor-plates': {
        return (deco.rects || [deco.rect]).filter(Boolean).map(r =>
          `<rect x="${r.x || 0}" y="${r.y || 0}" width="${r.width || r.w || 30}" height="${r.height || r.h || 6}" rx="${r.rx || 2}" fill="${deco.color || palette.accent}" opacity="${deco.opacity || 0.5}"/>`
        ).join('\n  ');
      }
      case 'shoulder-pads': {
        return (deco.rects || []).map(r =>
          `<rect x="${r.x || 0}" y="${r.y || 0}" width="${r.width || r.w || 10}" height="${r.height || r.h || 8}" rx="2" fill="${deco.color || palette.accent}" opacity="${deco.opacity || 0.6}"/>`
        ).join('\n  ');
      }
      case 'legs': {
        return (deco.lines || []).map(l =>
          `<line x1="${l.x1}" y1="${l.y1}" x2="${l.x2}" y2="${l.y2}" stroke="${deco.color || palette.dark}" stroke-width="${l.w || 2}" stroke-linecap="round"/>`
        ).join('\n  ');
      }
      case 'cloak-outline': {
        return `<circle cx="${deco.cx || 64}" cy="${deco.cy || 64}" r="${deco.r || 34}" fill="none" stroke="${deco.color || palette.accent}" stroke-width="1.5" stroke-dasharray="${deco.dasharray || '8 4'}" opacity="${deco.opacity || 0.35}"/>`;
      }
      case 'eye-slits': {
        return (deco.rects || []).map(r =>
          `<rect x="${r.x || 0}" y="${r.y || 0}" width="${r.width || r.w || 12}" height="${r.height || r.h || 3}" rx="1" fill="${deco.color || '#fff'}" opacity="${deco.opacity || 0.8}"/>`
        ).join('\n  ');
      }
      case 'crack-lines': {
        return (deco.paths || []).map(p =>
          `<path d="${p}" fill="none" stroke="${deco.color || palette.accent}" stroke-width="${deco.strokeWidth || 2}" opacity="${deco.opacity || 0.6}"/>`
        ).join('\n  ');
      }
      case 'child-preview': {
        return (deco.circles || []).map(c =>
          `<circle cx="${c.cx}" cy="${c.cy}" r="${c.r || 8}" fill="${deco.color || palette.secondary}" opacity="${deco.opacity || 0.4}"/>`
        ).join('\n  ');
      }
      case 'teleport-ring': {
        return `<circle cx="${deco.cx || 64}" cy="${deco.cy || 64}" r="${deco.r || 38}" fill="none" stroke="${deco.color || palette.accent}" stroke-width="${deco.strokeWidth || 2}" stroke-dasharray="${deco.dasharray || '6 4'}" opacity="${deco.opacity || 0.5}"/>`;
      }
      case 'energy-sparkles': {
        return (deco.positions || []).map(p =>
          `<circle cx="${p.cx}" cy="${p.cy}" r="${p.r || 2}" fill="${deco.color || palette.accent}" opacity="${deco.opacity || 0.7}"/>`
        ).join('\n  ');
      }
      case 'medical-cross': {
        const cx = deco.cx || 64, cy = deco.cy || 64, s = deco.size || 16, w = deco.width || 6;
        return `<rect x="${cx - w / 2}" y="${cy - s / 2}" width="${w}" height="${s}" rx="2" fill="${deco.color || palette.accent}" opacity="0.8"/>` +
          `<rect x="${cx - s / 2}" y="${cy - w / 2}" width="${s}" height="${w}" rx="2" fill="${deco.color || palette.accent}" opacity="0.8"/>`;
      }
      case 'heal-particles': {
        return (deco.positions || []).map(p =>
          `<circle cx="${p.cx}" cy="${p.cy}" r="${p.r || 2}" fill="${deco.color || palette.accent}" opacity="${deco.opacity || 0.5}"/>`
        ).join('\n  ');
      }
      case 'aura-ring':
      case 'aura-ring-inner': {
        return `<circle cx="${deco.cx || 64}" cy="${deco.cy || 64}" r="${deco.r || 42}" fill="none" stroke="${deco.color || palette.accent}" stroke-width="${deco.strokeWidth || 1.5}" opacity="${deco.opacity || 0.3}"/>`;
      }
      case 'tail-feather': {
        const p = deco.path || deco.points;
        if (typeof p === 'string') {
          return `<path d="${p}" fill="${deco.color || palette.accent}" opacity="${deco.opacity || 0.5}"/>`;
        }
        if (Array.isArray(p)) {
          return `<polygon points="${p.join(',')}" fill="${deco.color || palette.accent}" opacity="${deco.opacity || 0.5}"/>`;
        }
        return '';
      }
      case 'glowing-eyes': {
        return (deco.positions || []).map(p =>
          `<circle cx="${p.cx}" cy="${p.cy}" r="${p.r || 4}" fill="${deco.color || '#ef4444'}" opacity="${deco.opacity || 0.9}"/>` +
          `<circle cx="${p.cx}" cy="${p.cy}" r="${(p.r || 4) + 3}" fill="${deco.color || '#ef4444'}" opacity="0.2"/>`
        ).join('\n  ');
      }
      default:
        return `<!-- unknown decoration: ${deco.type} -->`;
    }
  }).join('\n  ');
}

// ── Frame generation ──

function generateEnemyFrame(key, def, state, frameIdx) {
  _gradId = 0;
  const palette = def.palette;
  const defs = [];

  // Glow gradient
  const glowId = nextGradId();
  defs.push(radialGradientSvg(glowId, 64, 64, palette.primary, 0.3));

  // Apply animation transforms
  let bodyOffsetY = 0;
  let decoRotate = 0;
  let hitOverlay = '';

  if (state === 'walk') {
    // 4-frame walk cycle: bounce pattern 0, -2, 0, 2
    const bouncePattern = [0, -2, 0, 2];
    bodyOffsetY = bouncePattern[frameIdx % 4];
    decoRotate = bouncePattern[frameIdx % 4] * 1.5;
  } else if (state === 'hit') {
    // Frame 0: flash white overlay, frame 1: normal (recovering)
    if (frameIdx === 0) {
      hitOverlay = `<rect x="0" y="0" width="128" height="128" fill="#fff" opacity="0.5"/>`;
    }
  }

  const bodySvg = renderBody(def.body, palette, bodyOffsetY);
  const headSvg = renderHead(def.head, palette, bodyOffsetY);
  const decoSvg = renderDecorations(def.decorations, palette, decoRotate);

  // Shadow
  const shadowCy = (def.body?.cy || 64) + (def.body?.r || 28) + 8;
  const shadowRx = (def.body?.r || 28) * 0.8;

  return `<svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 128 128">
  <defs>
    ${defs.join('\n    ')}
  </defs>
  <!-- ${def.name}: ${state}-${frameIdx} -->
  <ellipse cx="64" cy="${shadowCy}" rx="${shadowRx}" ry="6" fill="rgba(0,0,0,0.15)"/>
  <circle cx="64" cy="64" r="38" fill="url(#${glowId})"/>
  ${bodySvg}
  ${headSvg}
  ${decoSvg}
  ${hitOverlay}
</svg>`;
}

// ── Exports ──

/**
 * Generate all enemy SVGs from a visuals config object.
 * Returns array of {key, svg} where key includes frame info: "runner-walk-0".
 */
export function generateAllEnemySvgs(config) {
  const meta = config._meta || {};
  const walkFrames = meta.animations?.walk?.frames || 4;
  const hitFrames = meta.animations?.hit?.frames || 2;

  const results = [];
  for (const [key, def] of Object.entries(config)) {
    if (key.startsWith('_')) continue;

    for (let i = 0; i < walkFrames; i++) {
      results.push({
        key: `${key}-walk-${i}`,
        name: def.name,
        svg: generateEnemyFrame(key, def, 'walk', i),
      });
    }

    for (let i = 0; i < hitFrames; i++) {
      results.push({
        key: `${key}-hit-${i}`,
        name: def.name,
        svg: generateEnemyFrame(key, def, 'hit', i),
      });
    }
  }
  return results;
}
