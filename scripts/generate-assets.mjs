#!/usr/bin/env node
/**
 * generate-assets.mjs — Unified asset generation pipeline.
 *
 * Reads visual description JSON files, generates SVG, converts to PNG.
 *
 * Usage:
 *   node scripts/generate-assets.mjs <type> [--force] [--svg-only] [--png-only] [--size N]
 *
 * Types:
 *   wardens   — Generate warden assets from config/visuals/wardens.json
 *   towers    — Generate tower assets from config/visuals/towers.json (multi-frame)
 *   enemies   — Generate enemy assets from config/visuals/enemies.json (multi-frame)
 *   all       — Generate all asset types
 *
 * Options:
 *   --force     Overwrite existing files
 *   --svg-only  Only generate SVG (skip PNG conversion)
 *   --png-only  Only convert existing SVG to PNG (skip SVG generation)
 *   --size N    PNG output size in pixels (default: from config or 256)
 *
 * Examples:
 *   node scripts/generate-assets.mjs wardens
 *   node scripts/generate-assets.mjs wardens --force --size 512
 *   node scripts/generate-assets.mjs all
 */
import fs from 'node:fs';
import path from 'node:path';
import { fileURLToPath } from 'node:url';
import { generateAllWardenSvgs } from './generators/warden.mjs';
import { generateAllTowerSvgs } from './generators/tower.mjs';
import { generateAllEnemySvgs } from './generators/enemy.mjs';
import { svgToPng } from './generators/svgToPng.mjs';

const __dirname = path.dirname(fileURLToPath(import.meta.url));
const ROOT = path.resolve(__dirname, '..');

// ── Parse CLI args ──

const args = process.argv.slice(2);
const typeArg = args.find(a => !a.startsWith('-'));
const force = args.includes('--force');
const svgOnly = args.includes('--svg-only');
const pngOnly = args.includes('--png-only');
const sizeIdx = args.indexOf('--size');
const cliSize = sizeIdx >= 0 ? parseInt(args[sizeIdx + 1], 10) : null;

if (!typeArg) {
  console.log(`Usage: node scripts/generate-assets.mjs <type> [options]

Types: wardens, towers, enemies, all

Options:
  --force      Overwrite existing files
  --svg-only   Only generate SVG (skip PNG)
  --png-only   Only convert existing SVG to PNG
  --size N     PNG size in pixels (default: 256)`);
  process.exit(0);
}

// ── Generator registry ──

const GENERATORS = {
  wardens: {
    configPath: 'config/visuals/wardens.json',
    outDir: 'assets/wardens',
    filenamePrefix: 'warden-',
    generate: generateAllWardenSvgs,
  },
  towers: {
    configPath: 'config/visuals/towers.json',
    outDir: 'assets/towers',
    filenamePrefix: 'tower-',
    generate: generateAllTowerSvgs,
    // Each tower gets its own subdirectory: assets/towers/{towerKey}/tower-{towerKey}-{state}-{frame}.ext
    subdirFromKey: (itemKey) => itemKey.replace(/-(idle|attack)-\d+$/, ''),
  },
  enemies: {
    configPath: 'config/visuals/enemies.json',
    outDir: 'assets/enemies',
    filenamePrefix: '',
    generate: generateAllEnemySvgs,
  },
};

// ── Run a single generator ──

async function runGenerator(name, gen) {
  const configFile = path.resolve(ROOT, gen.configPath);
  if (!fs.existsSync(configFile)) {
    console.error(`  Config not found: ${gen.configPath}`);
    return;
  }

  const config = JSON.parse(fs.readFileSync(configFile, 'utf8'));
  const meta = config._meta || {};
  const outputSize = cliSize || meta.outputSize || 256;
  const outDir = path.resolve(ROOT, gen.outDir);
  fs.mkdirSync(outDir, { recursive: true });

  // Resolve per-item output directory (supports subdirFromKey for per-tower dirs)
  const itemOutDir = (itemKey) => {
    if (gen.subdirFromKey) {
      const sub = path.join(outDir, gen.subdirFromKey(itemKey));
      fs.mkdirSync(sub, { recursive: true });
      return sub;
    }
    return outDir;
  };

  let svgCount = 0;
  let pngCount = 0;

  if (!pngOnly) {
    // Generate SVGs
    const items = gen.generate(config);
    for (const item of items) {
      const dir = itemOutDir(item.key);
      const svgPath = path.join(dir, `${gen.filenamePrefix}${item.key}.svg`);
      if (force || !fs.existsSync(svgPath)) {
        fs.writeFileSync(svgPath, item.svg, 'utf8');
        svgCount++;
      }
    }
    console.log(`  [${name}] SVG: ${svgCount} generated`);

    if (!svgOnly) {
      // Convert to PNG
      for (const item of items) {
        const dir = itemOutDir(item.key);
        const svgPath = path.join(dir, `${gen.filenamePrefix}${item.key}.svg`);
        const pngPath = path.join(dir, `${gen.filenamePrefix}${item.key}.png`);
        if (force || !fs.existsSync(pngPath)) {
          const svgStr = fs.readFileSync(svgPath, 'utf8');
          const result = await svgToPng(svgStr, pngPath, { size: outputSize });
          pngCount++;
        }
      }
      console.log(`  [${name}] PNG: ${pngCount} generated (${outputSize}x${outputSize})`);
    }
  } else {
    // PNG-only mode: convert all existing SVGs in subdirs
    const scanDirs = gen.subdirFromKey
      ? fs.readdirSync(outDir).map(d => path.join(outDir, d)).filter(d => fs.statSync(d).isDirectory())
      : [outDir];
    for (const dir of scanDirs) {
      const svgFiles = fs.readdirSync(dir).filter(f => f.startsWith(gen.filenamePrefix) && f.endsWith('.svg'));
      for (const svgFile of svgFiles) {
        const svgPath = path.join(dir, svgFile);
        const pngFile = svgFile.replace(/\.svg$/, '.png');
        const pngPath = path.join(dir, pngFile);
        if (force || !fs.existsSync(pngPath)) {
          const svgStr = fs.readFileSync(svgPath, 'utf8');
          await svgToPng(svgStr, pngPath, { size: outputSize });
          pngCount++;
        }
      }
    }
    console.log(`  [${name}] PNG: ${pngCount} converted (${outputSize}x${outputSize})`);
  }
}

// ── Main ──

async function main() {
  console.log('generate-assets: starting...');

  const types = typeArg === 'all' ? Object.keys(GENERATORS) : [typeArg];

  for (const t of types) {
    const gen = GENERATORS[t];
    if (!gen) {
      console.error(`  Unknown type: ${t}`);
      console.error(`  Available: ${Object.keys(GENERATORS).join(', ')}`);
      process.exit(1);
    }
    await runGenerator(t, gen);
  }

  console.log('generate-assets: done.');
}

main().catch(err => {
  console.error('generate-assets failed:', err);
  process.exit(1);
});
