/**
 * svgToPng.mjs — Convert SVG files/strings to PNG using sharp.
 *
 * Usage:
 *   import { svgToPng, batchSvgToPng } from './svgToPng.mjs';
 *   await svgToPng(svgString, outputPath, { size: 256 });
 *   await batchSvgToPng(items, { size: 256 });
 */
import sharp from 'sharp';

/**
 * Convert a single SVG string to PNG.
 * @param {string} svgStr — SVG markup
 * @param {string} outPath — output PNG file path
 * @param {object} [opts]
 * @param {number} [opts.size=256] — output width & height in px
 * @returns {Promise<{path: string, bytes: number}>}
 */
export async function svgToPng(svgStr, outPath, opts = {}) {
  const size = opts.size || 256;
  const buf = Buffer.from(svgStr, 'utf8');

  const info = await sharp(buf)
    .resize(size, size, { fit: 'contain', background: { r: 0, g: 0, b: 0, alpha: 0 } })
    .png()
    .toFile(outPath);

  return { path: outPath, bytes: info.size };
}

/**
 * Batch convert SVG strings to PNG.
 * @param {Array<{svg: string, outPath: string}>} items
 * @param {object} [opts]
 * @param {number} [opts.size=256]
 * @returns {Promise<Array<{path: string, bytes: number}>>}
 */
export async function batchSvgToPng(items, opts = {}) {
  const results = [];
  for (const item of items) {
    const r = await svgToPng(item.svg, item.outPath, opts);
    results.push(r);
  }
  return results;
}
