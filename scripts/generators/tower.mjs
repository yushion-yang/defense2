/**
 * tower.mjs — Tower SVG multi-frame generator.
 *
 * Reads config/visuals/towers.json and produces multiple SVG frames per tower:
 *   idle-0, idle-1 (breathing), attack-0, attack-1, attack-2 (fire sequence).
 *
 * SVG viewBox: 0 0 128 128.
 */

let _gradId = 0;
function gid() { return `tg${_gradId++}`; }

// ── SVG gradient helpers ──

function svgRadialGrad(id, stops) {
  const s = stops.map(st =>
    `<stop offset="${st.offset}" stop-color="${st.color}"/>`
  ).join('');
  return `<radialGradient id="${id}" cx="50%" cy="50%" r="50%">${s}</radialGradient>`;
}

function svgLinearGrad(id, grad) {
  const x1 = grad.x1 || '0%';
  const y1 = grad.y1 || '0%';
  const x2 = grad.x2 || '0%';
  const y2 = grad.y2 || '100%';
  const s = grad.stops.map(st =>
    `<stop offset="${st.offset}" stop-color="${st.color}"/>`
  ).join('');
  return `<linearGradient id="${id}" x1="${x1}" y1="${y1}" x2="${x2}" y2="${y2}">${s}</linearGradient>`;
}

function svgGlowGrad(id, color, opacity) {
  return `<radialGradient id="${id}" cx="50%" cy="50%" r="50%">` +
    `<stop offset="0%" stop-color="${color}" stop-opacity="${opacity}"/>` +
    `<stop offset="100%" stop-color="${color}" stop-opacity="0"/>` +
    `</radialGradient>`;
}

// ── Geometry helpers ──

function hexPoints(cx, cy, r) {
  const pts = [];
  for (let i = 0; i < 6; i++) {
    const a = (Math.PI / 3) * i - Math.PI / 2;
    pts.push(`${(cx + r * Math.cos(a)).toFixed(1)},${(cy + r * Math.sin(a)).toFixed(1)}`);
  }
  return pts.join(' ');
}

function fourPointedStar(cx, cy, outerR, innerR, rotDeg = 0) {
  const pts = [];
  const rotRad = rotDeg * Math.PI / 180;
  for (let i = 0; i < 8; i++) {
    const a = (Math.PI / 4) * i - Math.PI / 2 + rotRad;
    const r = i % 2 === 0 ? outerR : innerR;
    pts.push(`${(cx + r * Math.cos(a)).toFixed(1)},${(cy + r * Math.sin(a)).toFixed(1)}`);
  }
  return pts.join(' ');
}

function arcPath(cx, cy, r, startDeg, sweepDeg) {
  const s = startDeg * Math.PI / 180;
  const e = (startDeg + sweepDeg) * Math.PI / 180;
  const x1 = cx + r * Math.cos(s);
  const y1 = cy + r * Math.sin(s);
  const x2 = cx + r * Math.cos(e);
  const y2 = cy + r * Math.sin(e);
  const large = sweepDeg > 180 ? 1 : 0;
  return `M${x1.toFixed(1)},${y1.toFixed(1)} A${r},${r} 0 ${large} 1 ${x2.toFixed(1)},${y2.toFixed(1)}`;
}

// ── Base renderers ──

function renderBase(base, palette, defs) {
  const out = [];
  const gradId = gid();

  if (base.gradient) {
    if (base.gradient.type === 'radial') {
      defs.push(svgRadialGrad(gradId, base.gradient.stops));
    } else {
      defs.push(svgLinearGrad(gradId, base.gradient));
    }
  }
  const fill = base.gradient ? `url(#${gradId})` : (palette.dark || '#333');
  const stroke = base.stroke ? ` stroke="${base.stroke}" stroke-width="${base.strokeWidth || 1}"` : '';

  switch (base.type) {
    case 'circle': {
      out.push(`<circle cx="${base.cx}" cy="${base.cy}" r="${base.r}" fill="${fill}"${stroke}/>`);
      if (base.innerRing) {
        const ir = base.innerRing;
        out.push(`<circle cx="${base.cx}" cy="${base.cy}" r="${ir.r}" fill="none" stroke="${ir.stroke}" stroke-width="${ir.strokeWidth}" opacity="${ir.opacity}"/>`);
      }
      break;
    }
    case 'hexagon': {
      out.push(`<polygon points="${hexPoints(base.cx, base.cy, base.r)}" fill="${fill}"${stroke}/>`);
      if (base.innerHex) {
        const ih = base.innerHex;
        out.push(`<polygon points="${hexPoints(base.cx, base.cy, ih.r)}" fill="none" stroke="${ih.stroke}" stroke-width="${ih.strokeWidth}" opacity="${ih.opacity}"/>`);
      }
      break;
    }
    case 'square': {
      const hs = base.size / 2;
      const x = base.cx - hs;
      const y = base.cy - hs;
      out.push(`<rect x="${x}" y="${y}" width="${base.size}" height="${base.size}" rx="${base.rx || 0}" fill="${fill}"${stroke}/>`);
      if (base.cornerBrackets) {
        const cb = base.cornerBrackets;
        const s = cb.size;
        const w = cb.width;
        const c = cb.color;
        const op = cb.opacity;
        // Four corners: TL, TR, BL, BR
        const corners = [
          [x, y], [x + base.size, y], [x, y + base.size], [x + base.size, y + base.size]
        ];
        const dirs = [
          [[1, 0], [0, 1]], [[-1, 0], [0, 1]], [[1, 0], [0, -1]], [[-1, 0], [0, -1]]
        ];
        corners.forEach(([cx2, cy2], i) => {
          const [d1, d2] = dirs[i];
          out.push(`<line x1="${cx2}" y1="${cy2}" x2="${cx2 + d1[0] * s}" y2="${cy2 + d1[1] * s}" stroke="${c}" stroke-width="${w}" opacity="${op}"/>`);
          out.push(`<line x1="${cx2}" y1="${cy2}" x2="${cx2 + d2[0] * s}" y2="${cy2 + d2[1] * s}" stroke="${c}" stroke-width="${w}" opacity="${op}"/>`);
        });
      }
      break;
    }
    case 'oval': {
      out.push(`<ellipse cx="${base.cx}" cy="${base.cy}" rx="${base.rx}" ry="${base.ry}" fill="${fill}"${stroke}/>`);
      if (base.innerOval) {
        const io = base.innerOval;
        out.push(`<ellipse cx="${base.cx}" cy="${base.cy}" rx="${io.rx}" ry="${io.ry}" fill="none" stroke="${io.stroke}" stroke-width="${io.strokeWidth}" opacity="${io.opacity}"/>`);
      }
      break;
    }
    case 'triangle': {
      const pts = base.points.join(',');
      // Convert pairs to "x,y x,y" format
      const formatted = [];
      for (let i = 0; i < base.points.length; i += 2) {
        formatted.push(`${base.points[i]},${base.points[i + 1]}`);
      }
      out.push(`<polygon points="${formatted.join(' ')}" fill="${fill}"${stroke}/>`);
      if (base.innerTriangle) {
        const it = base.innerTriangle;
        const fmt2 = [];
        for (let i = 0; i < it.points.length; i += 2) {
          fmt2.push(`${it.points[i]},${it.points[i + 1]}`);
        }
        out.push(`<polygon points="${fmt2.join(' ')}" fill="none" stroke="${it.stroke}" stroke-width="${it.strokeWidth}" opacity="${it.opacity}"/>`);
      }
      break;
    }
    case 'circle-with-ring': {
      // Outer ring
      out.push(`<circle cx="${base.cx}" cy="${base.cy}" r="${base.ringR}" fill="none" stroke="${base.ringColor}" stroke-width="${base.ringStrokeWidth}" opacity="${base.ringOpacity}"/>`);
      // Inner filled circle
      out.push(`<circle cx="${base.cx}" cy="${base.cy}" r="${base.r}" fill="${fill}"${stroke}/>`);
      break;
    }
    case 'double-ring': {
      // Outer ring
      out.push(`<circle cx="${base.cx}" cy="${base.cy}" r="${base.outerR}" fill="none" stroke="${base.outerStroke}" stroke-width="${base.strokeWidth}"/>`);
      // Inner filled
      out.push(`<circle cx="${base.cx}" cy="${base.cy}" r="${base.innerR}" fill="${fill}"/>`);
      // Inner ring stroke
      out.push(`<circle cx="${base.cx}" cy="${base.cy}" r="${base.innerR}" fill="none" stroke="${base.innerStroke}" stroke-width="${base.innerStrokeWidth}"/>`);
      break;
    }
    default:
      out.push(`<circle cx="64" cy="80" r="30" fill="${palette.dark}" opacity="0.8"/>`);
  }

  return out.join('\n  ');
}

// ── Turret renderers ──

function renderTurret(turret, palette, defs, offsetY = 0) {
  if (!turret) return '';
  const out = [];

  // Turret body gradient
  let turretFill = palette.primary;
  if (turret.gradient) {
    const tgId = gid();
    if (turret.gradient.type === 'linear') {
      defs.push(svgLinearGrad(tgId, turret.gradient));
    } else {
      defs.push(svgRadialGrad(tgId, turret.gradient.stops));
    }
    turretFill = `url(#${tgId})`;
  }

  switch (turret.type) {
    case 'barrel': {
      const r = turret.rect;
      const y = r.y + offsetY;
      // Muzzle flare (behind barrel)
      if (turret.muzzleFlare) {
        const mf = turret.muzzleFlare;
        out.push(`<rect x="${mf.x}" y="${mf.y + offsetY}" width="${mf.width}" height="${mf.height}" rx="${mf.rx}" fill="${mf.color}" opacity="0.6"/>`);
      }
      // Main barrel
      out.push(`<rect x="${r.x}" y="${y}" width="${r.width}" height="${r.height}" rx="${r.rx}" fill="${turretFill}"/>`);
      // Barrel highlight stripe
      out.push(`<rect x="${r.x + 2}" y="${y}" width="${r.width - 4}" height="${r.height}" rx="${r.rx - 1}" fill="${palette.accent}" opacity="0.12"/>`);
      // Lens
      if (turret.lens) {
        const l = turret.lens;
        const ly = l.cy + offsetY;
        out.push(`<circle cx="${l.cx}" cy="${ly}" r="${l.r}" fill="${l.color}" opacity="${l.opacity}"/>`);
        if (l.innerR) {
          out.push(`<circle cx="${l.cx}" cy="${ly}" r="${l.innerR}" fill="${l.innerColor}" opacity="0.8"/>`);
        }
      }
      break;
    }
    case 'barrel-with-crystal': {
      const r = turret.rect;
      const y = r.y + offsetY;
      // Barrel
      out.push(`<rect x="${r.x}" y="${y}" width="${r.width}" height="${r.height}" rx="${r.rx}" fill="${turretFill}"/>`);
      // Crystal
      if (turret.crystal) {
        const c = turret.crystal;
        const pts = [];
        for (let i = 0; i < c.points.length; i += 2) {
          pts.push(`${c.points[i]},${c.points[i + 1] + offsetY}`);
        }
        out.push(`<polygon points="${pts.join(' ')}" fill="${c.color}" opacity="${c.opacity}" stroke="${c.stroke}" stroke-width="${c.strokeWidth}"/>`);
        // Inner diamond
        if (c.innerDiamond) {
          const id = c.innerDiamond;
          const ipts = [];
          for (let i = 0; i < id.points.length; i += 2) {
            ipts.push(`${id.points[i]},${id.points[i + 1] + offsetY}`);
          }
          out.push(`<polygon points="${ipts.join(' ')}" fill="${id.color}" opacity="${id.opacity}"/>`);
        }
      }
      break;
    }
    case 'tesla-coil': {
      // Spine
      if (turret.spine) {
        const sp = turret.spine;
        out.push(`<rect x="${sp.x - sp.width / 2}" y="${sp.y2 + offsetY}" width="${sp.width}" height="${sp.y1 - sp.y2}" rx="2" fill="${sp.color}"/>`);
      }
      // Coils
      const cColor = turret.coilColor || palette.primary;
      const cStroke = turret.coilStroke || palette.accent;
      for (const coil of turret.coils) {
        out.push(`<ellipse cx="${coil.cx}" cy="${coil.cy + offsetY}" rx="${coil.rx}" ry="${coil.ry}" fill="none" stroke="${cStroke}" stroke-width="${coil.strokeWidth}" opacity="0.7"/>`);
        // Filled inner for depth
        out.push(`<ellipse cx="${coil.cx}" cy="${coil.cy + offsetY}" rx="${coil.rx - 1}" ry="${coil.ry - 0.5}" fill="${cColor}" opacity="0.3"/>`);
      }
      // Sphere
      if (turret.sphere) {
        const s = turret.sphere;
        const sy = s.cy + offsetY;
        out.push(`<circle cx="${s.cx}" cy="${sy}" r="${s.r}" fill="${s.color}" opacity="${s.opacity}"/>`);
        if (s.innerR) {
          out.push(`<circle cx="${s.cx}" cy="${sy}" r="${s.innerR}" fill="${s.innerColor}" opacity="0.85"/>`);
        }
      }
      break;
    }
    case 'twin-barrels': {
      for (const b of turret.barrels) {
        const by = b.y + offsetY;
        out.push(`<rect x="${b.x}" y="${by}" width="${b.width}" height="${b.height}" rx="${b.rx}" fill="${turretFill}"/>`);
        // Highlight stripe
        out.push(`<rect x="${b.x + 1}" y="${by}" width="${b.width - 2}" height="${b.height}" rx="${b.rx}" fill="${palette.accent}" opacity="0.1"/>`);
      }
      // Scope
      if (turret.scope) {
        const sc = turret.scope;
        const scy = sc.cy + offsetY;
        out.push(`<circle cx="${sc.cx}" cy="${scy}" r="${sc.r}" fill="${sc.fill || 'none'}" stroke="${sc.stroke}" stroke-width="${sc.strokeWidth}"/>`);
        // Cross lines
        if (sc.crossLines) {
          for (const cl of sc.crossLines) {
            out.push(`<line x1="${cl.x1}" y1="${cl.y1 + offsetY}" x2="${cl.x2}" y2="${cl.y2 + offsetY}" stroke="${sc.crossColor}" stroke-width="${sc.crossWidth}" opacity="${sc.crossOpacity}"/>`);
          }
        }
      }
      break;
    }
    case 'wide-emitter': {
      // Trapezoid body
      if (turret.trapezoid) {
        const tp = turret.trapezoid;
        const pts = [];
        for (let i = 0; i < tp.points.length; i += 2) {
          pts.push(`${tp.points[i]},${tp.points[i + 1] + offsetY}`);
        }
        out.push(`<polygon points="${pts.join(' ')}" fill="${turretFill}"/>`);
      }
      // Emitter slot
      if (turret.emitterSlot) {
        const es = turret.emitterSlot;
        out.push(`<rect x="${es.x}" y="${es.y + offsetY}" width="${es.width}" height="${es.height}" rx="${es.rx}" fill="${es.color}" opacity="${es.opacity}"/>`);
        if (es.innerSlot) {
          const is2 = es.innerSlot;
          out.push(`<rect x="${is2.x}" y="${is2.y + offsetY}" width="${is2.width}" height="${is2.height}" rx="${is2.rx}" fill="${is2.color}" opacity="${is2.opacity}"/>`);
        }
      }
      break;
    }
    case 'triple-barrel-fan': {
      // Hub
      if (turret.hub) {
        const h = turret.hub;
        out.push(`<circle cx="${h.cx}" cy="${h.cy + offsetY}" r="${h.r}" fill="${h.color}"/>`);
      }
      // Barrels as thick lines with round caps
      for (const b of turret.barrels) {
        out.push(`<line x1="${b.x1}" y1="${b.y1 + offsetY}" x2="${b.x2}" y2="${b.y2 + offsetY}" stroke="${turretFill}" stroke-width="${b.width}" stroke-linecap="round"/>`);
      }
      break;
    }
    case 'heavy-barrel': {
      const r = turret.rect;
      const ry = r.y + offsetY;
      // Side rails
      if (turret.sideRails) {
        for (const sr of turret.sideRails) {
          out.push(`<rect x="${sr.x}" y="${sr.y + offsetY}" width="${sr.width}" height="${sr.height}" rx="${sr.rx}" fill="${sr.color}" opacity="0.7"/>`);
        }
      }
      // Main barrel
      out.push(`<rect x="${r.x}" y="${ry}" width="${r.width}" height="${r.height}" rx="${r.rx}" fill="${turretFill}"/>`);
      // Barrel highlight
      out.push(`<rect x="${r.x + 3}" y="${ry}" width="${r.width - 6}" height="${r.height}" rx="${r.rx - 1}" fill="${palette.accent}" opacity="0.1"/>`);
      // Charging ring
      if (turret.chargingRing) {
        const cr = turret.chargingRing;
        out.push(`<circle cx="${cr.cx}" cy="${cr.cy + offsetY}" r="${cr.r}" fill="none" stroke="${cr.stroke}" stroke-width="${cr.strokeWidth}" stroke-dasharray="${cr.dasharray}" opacity="${cr.opacity}"/>`);
      }
      break;
    }
    case 'spinning-blade': {
      if (turret.blade) {
        const b = turret.blade;
        const cx = b.cx;
        const cy = b.cy + offsetY;
        const outerR = b.outerR || 30;
        const innerR = b.innerR || 12;
        const rotDeg = turret._rotDeg || 0;

        // Blade gradient
        let bladeFill = palette.primary;
        if (b.gradient) {
          const bgId = gid();
          defs.push(svgLinearGrad(bgId, b.gradient));
          bladeFill = `url(#${bgId})`;
        }

        const pts = fourPointedStar(cx, cy, outerR, innerR, rotDeg);
        const bladeStroke = b.stroke ? ` stroke="${b.stroke}" stroke-width="${b.strokeWidth}"` : '';
        out.push(`<polygon points="${pts}" fill="${bladeFill}"${bladeStroke} opacity="0.85"/>`);

        // Spin sweep arcs (attack only) — 4 arc segments showing rotation
        if (turret._spinArcs) {
          for (const sa of turret._spinArcs) {
            const d = arcPath(cx, cy, sa.r, sa.startAngle + rotDeg, sa.sweepAngle);
            out.push(`<path d="${d}" fill="none" stroke="${sa.color}" stroke-width="${sa.strokeWidth}" stroke-linecap="round" opacity="${sa.opacity}"/>`);
          }
        }
      }
      break;
    }
    default:
      out.push(`<rect x="56" y="${28 + offsetY}" width="16" height="40" rx="3" fill="${turretFill}"/>`);
  }

  return out.join('\n  ');
}

// ── Core renderer ──

function renderCore(core, palette, offsetY = 0) {
  if (!core) return '';
  const out = [];
  const cy = (core.cy || 64) + offsetY;

  if (core.type === 'snowflake-cross') {
    // Lines
    for (const l of core.lines) {
      out.push(`<line x1="${l.x1}" y1="${l.y1 + offsetY}" x2="${l.x2}" y2="${l.y2 + offsetY}" stroke="${core.color}" stroke-width="${core.lineWidth}" stroke-linecap="round" opacity="${core.opacity}"/>`);
    }
  } else if (core.type === 'crosshair') {
    // Ring
    out.push(`<circle cx="${core.cx}" cy="${cy}" r="${core.ringR}" fill="none" stroke="${core.color}" stroke-width="${core.lineWidth}" opacity="${core.opacity}"/>`);
    // Cross lines
    for (const l of core.crossLines) {
      out.push(`<line x1="${l.x1}" y1="${l.y1 + offsetY}" x2="${l.x2}" y2="${l.y2 + offsetY}" stroke="${core.color}" stroke-width="${core.lineWidth}" opacity="${core.opacity}"/>`);
    }
  } else if (core.type === 'horizontal-glow') {
    out.push(`<ellipse cx="${core.cx}" cy="${cy}" rx="${core.rx}" ry="${core.ry}" fill="${core.color}" opacity="${core.opacity}"/>`);
  } else {
    // Default: circle core
    out.push(`<circle cx="${core.cx}" cy="${cy}" r="${core.r}" fill="${core.color}" opacity="${core.opacity}"/>`);
    if (core.innerR) {
      out.push(`<circle cx="${core.cx}" cy="${cy}" r="${core.innerR}" fill="${core.innerColor || palette.highlight || '#fff'}" opacity="0.8"/>`);
    }
    if (core.ring) {
      out.push(`<circle cx="${core.cx}" cy="${cy}" r="${core.ring.r}" fill="none" stroke="${core.ring.stroke}" stroke-width="${core.ring.strokeWidth}"/>`);
    }
  }

  return out.join('\n  ');
}

// ── Effects renderer ──

function renderEffects(effects, palette, defs) {
  if (!effects || effects.length === 0) return '';
  const out = [];

  for (const eff of effects) {
    switch (eff.type) {
      case 'glow-halo': {
        const hid = gid();
        defs.push(svgGlowGrad(hid, eff.color, eff.opacity));
        out.push(`<circle cx="${eff.cx}" cy="${eff.cy}" r="${eff.r}" fill="url(#${hid})"/>`);
        break;
      }
      case 'ice-crystals': {
        for (const c of eff.crystals) {
          const pts = [];
          for (let i = 0; i < c.points.length; i += 2) {
            pts.push(`${c.points[i]},${c.points[i + 1]}`);
          }
          out.push(`<polygon points="${pts.join(' ')}" fill="${eff.color}" opacity="${c.opacity}"/>`);
        }
        break;
      }
      case 'lightning-arcs': {
        for (const arc of eff.arcs) {
          out.push(`<path d="${arc.path}" fill="none" stroke="${eff.color}" stroke-width="${arc.strokeWidth}" stroke-linecap="round" opacity="${arc.opacity}"/>`);
        }
        break;
      }
      case 'spread-indicators':
      case 'targeting-lines': {
        for (const l of eff.lines) {
          out.push(`<line x1="${l.x1}" y1="${l.y1}" x2="${l.x2}" y2="${l.y2}" stroke="${eff.color}" stroke-width="${l.strokeWidth || eff.lineWidth || 1}" stroke-linecap="round" opacity="${l.opacity || eff.opacity || 0.5}"/>`);
        }
        break;
      }
      case 'pellet-dots': {
        for (const p of eff.pellets) {
          out.push(`<circle cx="${p.cx}" cy="${p.cy}" r="${p.r}" fill="${eff.color}" opacity="${p.opacity}"/>`);
        }
        break;
      }
      case 'energy-vortex': {
        for (const a of eff.arcs) {
          const d = arcPath(eff.cx, eff.cy, eff.r, a.startAngle, a.sweepAngle);
          out.push(`<path d="${d}" fill="none" stroke="${eff.color}" stroke-width="${a.strokeWidth}" stroke-linecap="round" opacity="${a.opacity}"/>`);
        }
        break;
      }
      case 'motion-arcs': {
        for (const a of eff.arcs) {
          const d = arcPath(a.cx, a.cy, a.r, a.startAngle, a.sweepAngle);
          out.push(`<path d="${d}" fill="none" stroke="${eff.color}" stroke-width="${a.strokeWidth}" stroke-linecap="round" opacity="${a.opacity}"/>`);
        }
        break;
      }
      default:
        out.push(`<!-- unknown effect: ${eff.type} -->`);
    }
  }

  return out.join('\n  ');
}

// ── Frame generation ──

function getAnimValue(anims, state, frameIdx, target, property, defaultVal) {
  if (!anims || !anims[state]) return defaultVal;
  const transforms = anims[state].transforms || [];
  for (const t of transforms) {
    if (t.target === target && t.property === property) {
      const vals = t.values;
      if (frameIdx < vals.length) return vals[frameIdx];
      return vals[vals.length - 1];
    }
  }
  return defaultVal;
}

function generateTowerFrame(key, def, state, frameIdx) {
  _gradId = 0;
  const palette = def.palette;
  const defs = [];
  const anims = def.animations;

  // Get animation offsets
  const turretOffsetY = getAnimValue(anims, state, frameIdx, 'turret', 'translateY', 0);
  const effectsOpacity = getAnimValue(anims, state, frameIdx, 'effects', 'opacity', 1.0);
  const coreOpacity = getAnimValue(anims, state, frameIdx, 'core', 'opacity', 1.0);
  const turretRotate = getAnimValue(anims, state, frameIdx, 'turret', 'rotate', 0);

  // ── Spinning blade: bake rotation into vertex positions + add sweep arcs ──
  const turretCopy = def.turret ? JSON.parse(JSON.stringify(def.turret)) : null;
  if (turretCopy?.type === 'spinning-blade') {
    turretCopy._rotDeg = turretRotate;
    if (state === 'attack') {
      const bladeCx = turretCopy.blade?.cx || 64;
      const bladeCy = turretCopy.blade?.cy || 52;
      const bladeR = (turretCopy.blade?.outerR || 30) + 4;
      // 4 sweep arcs at 90-degree intervals, matching JS version's spin_aoe visual
      const sweepOpacity = [0.15, 0.6, 0.3][frameIdx] || 0.3;
      turretCopy._spinArcs = [0, 90, 180, 270].map(base => ({
        r: bladeR,
        startAngle: base,
        sweepAngle: 35,
        strokeWidth: 2.5,
        color: palette.accent,
        opacity: sweepOpacity
      }));
    }
  }

  // ── en-08 charge effect: growing energy sphere during attack ──
  let chargeSvg = '';
  if (key === 'en-08' && state === 'attack') {
    const ccy = (def.core?.cy || 24) + turretOffsetY;
    const ccx = def.core?.cx || 64;
    // frame 0: small charge, frame 1: full charge + ring, frame 2: dissipating
    const chargeProgress = [0.4, 1.0, 0.2][frameIdx];
    const chargeR = 4 + chargeProgress * 12;
    const chargeOp = 0.15 + chargeProgress * 0.4;
    // Outer glow
    const chGlowId = gid();
    defs.push(svgGlowGrad(chGlowId, palette.primary, chargeOp));
    chargeSvg += `<circle cx="${ccx}" cy="${ccy}" r="${chargeR + 8}" fill="url(#${chGlowId})"/>`;
    // Core sphere
    chargeSvg += `\n  <circle cx="${ccx}" cy="${ccy}" r="${chargeR.toFixed(1)}" fill="${palette.accent}" opacity="${(0.3 + chargeProgress * 0.6).toFixed(2)}"/>`;
    if (chargeProgress >= 0.5) {
      // Pulsing ring
      const ringR = chargeR * 1.4;
      chargeSvg += `\n  <circle cx="${ccx}" cy="${ccy}" r="${ringR.toFixed(1)}" fill="none" stroke="${palette.accent}" stroke-width="1.5" opacity="${(chargeProgress * 0.6).toFixed(2)}"/>`;
      // White-hot center
      const hotR = 2 + (chargeProgress - 0.5) * 6;
      chargeSvg += `\n  <circle cx="${ccx}" cy="${ccy}" r="${hotR.toFixed(1)}" fill="${palette.highlight}" opacity="${((chargeProgress - 0.5) * 1.5).toFixed(2)}"/>`;
    }
  }

  // Background glow
  const glowCx = def.core?.cx || 64;
  const glowCy = def.core?.cy || 55;
  const bgGlowId = gid();
  defs.push(svgGlowGrad(bgGlowId, palette.glow, 0.3));

  // Render layers
  const baseSvg = renderBase(def.base, palette, defs);
  const turretSvg = renderTurret(turretCopy || def.turret, palette, defs, turretOffsetY);
  const coreSvg = renderCore(def.core, palette, turretOffsetY);
  const effectSvg = renderEffects(def.effects, palette, defs);

  // Apply opacity modifiers
  const coreGroup = coreOpacity < 1.0
    ? `<g opacity="${coreOpacity.toFixed(2)}">\n    ${coreSvg}\n  </g>`
    : coreSvg;

  const effectGroup = effectsOpacity !== 1.0
    ? `<g opacity="${Math.min(effectsOpacity, 1.0).toFixed(2)}">\n    ${effectSvg}\n  </g>`
    : effectSvg;

  // ── Muzzle flash — different per tower type ──
  let flashSvg = '';
  if (state === 'attack' && frameIdx === 1) {
    if (key === 'wl-02') {
      // Spin blade: shockwave ring at center instead of muzzle flash
      const bladeCy = (def.turret?.blade?.cy || 52) + turretOffsetY;
      const shockId = gid();
      defs.push(svgGlowGrad(shockId, palette.accent, 0.5));
      flashSvg = `<circle cx="64" cy="${bladeCy}" r="36" fill="url(#${shockId})"/>`;
      flashSvg += `\n  <circle cx="64" cy="${bladeCy}" r="28" fill="none" stroke="${palette.accent}" stroke-width="2" opacity="0.6"/>`;
    } else if (key === 'en-08') {
      // Charge cannon: big muzzle burst
      const fcy = (def.core?.cy || 24) + turretOffsetY;
      const flashGlowId = gid();
      defs.push(svgGlowGrad(flashGlowId, palette.accent, 0.95));
      flashSvg = `<circle cx="64" cy="${fcy}" r="22" fill="url(#${flashGlowId})"/>`;
      flashSvg += `\n  <circle cx="64" cy="${fcy}" r="8" fill="${palette.highlight}" opacity="0.95"/>`;
      flashSvg += `\n  <circle cx="64" cy="${fcy}" r="4" fill="#fff" opacity="0.9"/>`;
    } else if (def.turret?.type === 'triple-barrel-fan') {
      // Scatter: flash at each muzzle
      const barrels = def.turret.barrels || [];
      for (const b of barrels) {
        const bx = b.x2;
        const by = b.y2 + turretOffsetY;
        const mfId = gid();
        defs.push(svgGlowGrad(mfId, palette.accent, 0.8));
        flashSvg += `<circle cx="${bx}" cy="${by}" r="10" fill="url(#${mfId})"/>`;
        flashSvg += `\n  <circle cx="${bx}" cy="${by}" r="3" fill="${palette.highlight || palette.accent}" opacity="0.9"/>`;
      }
    } else if (def.turret?.type === 'wide-emitter') {
      // Wide beam: horizontal flash bar
      const fcy = (def.turret.emitterSlot?.y || 28) - 2 + turretOffsetY;
      const flashGlowId = gid();
      defs.push(svgGlowGrad(flashGlowId, palette.accent, 0.85));
      flashSvg = `<ellipse cx="64" cy="${fcy}" rx="20" ry="8" fill="url(#${flashGlowId})"/>`;
      flashSvg += `\n  <rect x="48" y="${fcy - 2}" width="32" height="4" rx="2" fill="${palette.accent}" opacity="0.9"/>`;
    } else if (def.turret?.type === 'tesla-coil') {
      // Tesla: bright sphere flash
      const spy = (def.turret.sphere?.cy || 28) + turretOffsetY;
      const flashGlowId = gid();
      defs.push(svgGlowGrad(flashGlowId, palette.accent, 0.9));
      flashSvg = `<circle cx="64" cy="${spy}" r="16" fill="url(#${flashGlowId})"/>`;
      flashSvg += `\n  <circle cx="64" cy="${spy}" r="6" fill="${palette.highlight}" opacity="0.95"/>`;
    } else {
      // Default barrel towers: standard muzzle flash
      const fcy = (def.turret?.rect?.y || def.turret?.lens?.cy || 28) - 4 + turretOffsetY;
      const flashGlowId = gid();
      defs.push(svgGlowGrad(flashGlowId, palette.accent, 0.85));
      flashSvg = `<circle cx="64" cy="${fcy}" r="14" fill="url(#${flashGlowId})"/>`;
      flashSvg += `\n  <circle cx="64" cy="${fcy}" r="5" fill="${palette.accent}" opacity="0.95"/>`;
    }
  }

  // Also add a subtle flash on attack frame 0 (charge-up) for most towers
  if (state === 'attack' && frameIdx === 0 && key !== 'wl-02' && key !== 'en-08') {
    const fcy = (def.turret?.rect?.y || def.turret?.lens?.cy || def.turret?.sphere?.cy || 28) - 2 + turretOffsetY;
    const preFlashId = gid();
    defs.push(svgGlowGrad(preFlashId, palette.glow, 0.3));
    flashSvg = `<circle cx="64" cy="${fcy}" r="8" fill="url(#${preFlashId})"/>`;
  }

  return `<svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 128 128">
  <defs>
    ${defs.join('\n    ')}
  </defs>
  <!-- ${def.name || key}: ${state}-${frameIdx} -->
  <circle cx="${glowCx}" cy="${glowCy}" r="44" fill="url(#${bgGlowId})"/>
  ${baseSvg}
  ${turretSvg}
  ${coreGroup}
  ${effectGroup}
  ${chargeSvg}
  ${flashSvg}
</svg>`;
}

// ── Exports ──

/**
 * Generate all tower SVGs from a visuals config object.
 * Returns array of {key, name, svg} where key includes frame info: "laser-idle-0".
 */
export function generateAllTowerSvgs(config) {
  const meta = config._meta || {};
  const idleFrames = meta.animations?.idle?.frames || 2;
  const attackFrames = meta.animations?.attack?.frames || 3;

  const results = [];
  for (const [key, def] of Object.entries(config)) {
    if (key.startsWith('_')) continue;

    for (let i = 0; i < idleFrames; i++) {
      results.push({
        key: `${key}-idle-${i}`,
        name: def.name || def.label || key,
        svg: generateTowerFrame(key, def, 'idle', i),
      });
    }

    for (let i = 0; i < attackFrames; i++) {
      results.push({
        key: `${key}-attack-${i}`,
        name: def.name || def.label || key,
        svg: generateTowerFrame(key, def, 'attack', i),
      });
    }
  }
  return results;
}
