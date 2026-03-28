/**
 * warden.mjs — Warden SVG generator.
 *
 * Reads config/visuals/wardens.json and produces one SVG per warden.
 * Each warden has a unique visual design described declaratively:
 *   body shape, core, glow, decorations, optional eyes.
 *
 * SVG viewBox: 0 0 128 128 (same as tower/enemy assets).
 */

// ── Gradient helpers ──

let _gradId = 0;
function nextGradId() { return `wg${_gradId++}`; }

function linearGradientSvg(id, cfg) {
  const x1 = cfg.x1 || '50%';
  const y1 = cfg.y1 || '0%';
  const x2 = cfg.x2 || '50%';
  const y2 = cfg.y2 || '100%';
  const stops = cfg.stops.map(s =>
    `<stop offset="${s.offset}" stop-color="${s.color}" ${s.opacity != null ? `stop-opacity="${s.opacity}"` : ''}/>`
  ).join('');
  return `<linearGradient id="${id}" x1="${x1}" y1="${y1}" x2="${x2}" y2="${y2}">${stops}</linearGradient>`;
}

function radialGradientSvg(id, cx, cy, r, color, opacity) {
  return `<radialGradient id="${id}" cx="50%" cy="50%" r="50%">` +
    `<stop offset="0%" stop-color="${color}" stop-opacity="${opacity}"/>` +
    `<stop offset="100%" stop-color="${color}" stop-opacity="0"/>` +
    `</radialGradient>`;
}

// ── Hexagon path ──
function hexagonPath(cx, cy, r) {
  const pts = [];
  for (let i = 0; i < 6; i++) {
    const angle = (Math.PI / 3) * i - Math.PI / 2;
    pts.push(`${cx + r * Math.cos(angle)},${cy + r * Math.sin(angle)}`);
  }
  return pts.join(' ');
}

// ── Body renderers ──

function renderBody(def, defs) {
  const b = def.body;
  if (!b) return '';

  switch (b.type) {
    case 'diamond':
    case 'polygon':
    case 'inverted-triangle': {
      const pts = b.points.join(',');
      let fill;
      if (b.gradient) {
        const gid = nextGradId();
        defs.push(linearGradientSvg(gid, b.gradient));
        fill = `url(#${gid})`;
      } else {
        fill = def.palette.primary;
      }
      let inner = '';
      if (b.innerPoints) {
        inner = `<polygon points="${b.innerPoints.join(',')}" fill="${def.palette.secondary}" opacity="${b.innerOpacity || 0.5}"/>`;
      }
      const stroke = b.stroke ? ` stroke="${b.stroke}" stroke-width="${b.strokeWidth || 2}"` : '';
      return `<polygon points="${pts}" fill="${fill}"${stroke}/>` + inner;
    }

    case 'hexagram': {
      const gidA = nextGradId();
      const gidB = nextGradId();
      defs.push(linearGradientSvg(gidA, b.gradientA));
      defs.push(linearGradientSvg(gidB, b.gradientB));
      return `<polygon points="${b.triangleA.join(',')}" fill="url(#${gidA})" opacity="${b.opacity || 0.85}"/>` +
        `<polygon points="${b.triangleB.join(',')}" fill="url(#${gidB})" opacity="${b.opacity || 0.85}"/>`;
    }

    case 'concentric-rings': {
      return b.rings.map(ring =>
        `<circle cx="64" cy="64" r="${ring.r}" fill="none" stroke="${def.palette.secondary}" ` +
        `stroke-width="${ring.strokeWidth}" stroke-dasharray="${ring.dasharray}" opacity="${ring.opacity}"/>`
      ).join('\n  ');
    }

    default:
      return '';
  }
}

// ── Core renderer ──

function renderCore(def) {
  const c = def.core;
  if (!c) return '';

  let shape;
  if (c.type === 'hexagon') {
    shape = `<polygon points="${hexagonPath(c.cx, c.cy, c.r)}" fill="${c.color}" opacity="${c.opacity || 1}"/>`;
  } else {
    shape = `<circle cx="${c.cx}" cy="${c.cy}" r="${c.r}" fill="${c.color}" opacity="${c.opacity || 1}"/>`;
  }

  let inner = '';
  if (c.innerR) {
    inner = `<circle cx="${c.cx}" cy="${c.cy}" r="${c.innerR}" fill="${c.innerColor || '#fff'}" opacity="${c.innerOpacity || 0.9}"/>`;
  }

  return shape + inner;
}

// ── Decoration renderers ──

function renderDecoration(deco, palette) {
  switch (deco.type) {
    case 'wings': {
      return deco.paths.map(p =>
        `<path d="${p}" fill="${deco.color}" opacity="${deco.opacity || 0.7}"/>`
      ).join('\n  ');
    }

    case 'tail': {
      return `<polygon points="${deco.points.join(',')}" fill="${deco.color}" opacity="${deco.opacity || 0.5}"/>`;
    }

    case 'thrusters': {
      return deco.positions.map(p =>
        `<rect x="${p.x}" y="${p.y}" width="${p.w}" height="${p.h}" rx="2" fill="${deco.color}" opacity="${deco.opacity || 0.7}"/>`
      ).join('\n  ');
    }

    case 'thruster-flames': {
      return deco.triangles.map(t =>
        `<polygon points="${t.join(',')}" fill="${deco.color}" opacity="${deco.opacity || 0.5}"/>`
      ).join('\n  ');
    }

    case 'orbit-ring': {
      return `<circle cx="${deco.cx}" cy="${deco.cy}" r="${deco.r}" fill="none" ` +
        `stroke="${deco.stroke}" stroke-width="${deco.strokeWidth}" ` +
        `stroke-dasharray="${deco.dasharray}" opacity="${deco.opacity}"/>`;
    }

    case 'radial-lines': {
      return deco.lines.map(l =>
        `<line x1="${l.x1}" y1="${l.y1}" x2="${l.x2}" y2="${l.y2}" ` +
        `stroke="${l.color}" stroke-width="${l.width}" opacity="${l.opacity}"/>`
      ).join('\n  ');
    }

    case 'chain-links': {
      const lines = deco.anchors.map(a =>
        `<line x1="${deco.from[0]}" y1="${deco.from[1]}" x2="${a[0]}" y2="${a[1]}" ` +
        `stroke="${deco.lineColor}" stroke-width="${deco.lineWidth}" opacity="${deco.opacity}"/>`
      ).join('\n  ');
      const nodes = deco.anchors.map(a =>
        `<circle cx="${a[0]}" cy="${a[1]}" r="${deco.nodeR}" fill="${deco.nodeColor}" opacity="${deco.opacity + 0.1}"/>`
      ).join('\n  ');
      return lines + '\n  ' + nodes;
    }

    case 'sparks': {
      return deco.positions.map(p =>
        `<circle cx="${p.cx}" cy="${p.cy}" r="${p.r}" fill="${deco.color}" opacity="${deco.opacity}"/>`
      ).join('\n  ');
    }

    case 'crosshair': {
      const r = deco.r;
      const cx = deco.cx;
      const cy = deco.cy;
      return `<circle cx="${cx}" cy="${cy}" r="${r}" fill="none" ` +
        `stroke="${deco.stroke}" stroke-width="${deco.strokeWidth}" opacity="${deco.opacity}"/>` +
        `<line x1="${cx}" y1="${cy - r - 4}" x2="${cx}" y2="${cy - r + 6}" stroke="${deco.stroke}" stroke-width="${deco.strokeWidth}" opacity="${deco.opacity}"/>` +
        `<line x1="${cx}" y1="${cy + r - 6}" x2="${cx}" y2="${cy + r + 4}" stroke="${deco.stroke}" stroke-width="${deco.strokeWidth}" opacity="${deco.opacity}"/>` +
        `<line x1="${cx - r - 4}" y1="${cy}" x2="${cx - r + 6}" y2="${cy}" stroke="${deco.stroke}" stroke-width="${deco.strokeWidth}" opacity="${deco.opacity}"/>` +
        `<line x1="${cx + r - 6}" y1="${cy}" x2="${cx + r + 4}" y2="${cy}" stroke="${deco.stroke}" stroke-width="${deco.strokeWidth}" opacity="${deco.opacity}"/>`;
    }

    case 'light-beams': {
      return deco.beams.map(b =>
        `<line x1="${b.x1}" y1="${b.y1}" x2="${b.x2}" y2="${b.y2}" ` +
        `stroke="${deco.color}" stroke-width="${b.width}" stroke-linecap="round" opacity="${deco.opacity}"/>`
      ).join('\n  ');
    }

    case 'impact-rings': {
      return deco.positions.map(p =>
        `<ellipse cx="${p.cx}" cy="${p.cy}" rx="${p.rx}" ry="${p.ry}" ` +
        `fill="none" stroke="${deco.color}" stroke-width="1.5" opacity="${deco.opacity}"/>`
      ).join('\n  ');
    }

    default:
      return `<!-- unknown decoration: ${deco.type} -->`;
  }
}

// ── Eyes renderer ──

function renderEyes(def) {
  const e = def.eyes;
  if (!e) return '';

  return e.positions.map(pos => {
    const outer = `<ellipse cx="${pos.cx}" cy="${pos.cy}" rx="${e.outerRx}" ry="${e.outerRy}" fill="${e.outerColor}" opacity="0.9"/>`;
    const pupil = `<circle cx="${pos.cx + (e.pupilOffset?.dx || 0)}" cy="${pos.cy + (e.pupilOffset?.dy || 0)}" r="${e.pupilR}" fill="${e.pupilColor}"/>`;
    return outer + pupil;
  }).join('\n  ');
}

// ── Main generator ──

/**
 * Generate SVG string for a single warden.
 * @param {string} key — warden id (e.g. "prince")
 * @param {object} def — visual definition from wardens.json
 * @returns {string} SVG markup
 */
export function generateWardenSvg(key, def) {
  _gradId = 0;
  const defs = [];
  const palette = def.palette;

  // Glow gradient
  const glowId = nextGradId();
  defs.push(radialGradientSvg(glowId, def.glow.cx, def.glow.cy, def.glow.r, palette.glow, def.glow.opacity));

  // Body gradient (if polygon with gradient)
  const bodySvg = renderBody(def, defs);
  const coreSvg = renderCore(def);

  const decoSvg = (def.decorations || []).map(d => renderDecoration(d, palette)).join('\n  ');
  const eyesSvg = renderEyes(def);

  return `<svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 128 128">
  <defs>
    ${defs.join('\n    ')}
  </defs>
  <!-- ${def.name}: ${def.description} -->
  <circle cx="${def.glow.cx}" cy="${def.glow.cy}" r="${def.glow.r}" fill="url(#${glowId})"/>
  ${bodySvg}
  ${coreSvg}
  ${decoSvg}
  ${eyesSvg}
</svg>`;
}

/**
 * Generate all warden SVGs from a visuals config object.
 * @param {object} config — parsed wardens.json
 * @returns {Array<{key: string, name: string, svg: string}>}
 */
export function generateAllWardenSvgs(config) {
  const results = [];
  for (const [key, def] of Object.entries(config)) {
    if (key.startsWith('_')) continue;
    results.push({
      key,
      name: def.name,
      svg: generateWardenSvg(key, def),
    });
  }
  return results;
}
