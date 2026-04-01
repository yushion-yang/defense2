// generate-skill-icons.mjs — 为 9 个技能生成 64x64 手绘风 PNG 图标。
import sharp from 'sharp';
import { mkdirSync } from 'fs';

const OUT = 'assets/icons';
mkdirSync(OUT, { recursive: true });

// 手绘 SVG 图标定义
const skills = [
  {
    name: 'skill-lightning',
    bg: ['#0a1a3a', '#1a3a6b'],
    svg: `<!-- 闪电 -->
      <path d="M34 8 L28 28 L36 28 L26 56 L40 30 L32 30 Z" fill="#7dcfff" stroke="#aee4ff" stroke-width="1"/>
      <path d="M34 8 L28 28 L36 28 L26 56 L40 30 L32 30 Z" fill="url(#lightGlow)" opacity="0.4"/>`,
  },
  {
    name: 'skill-nuke',
    bg: ['#3a0a0a', '#5a1a1a'],
    svg: `<!-- 核弹爆炸 -->
      <circle cx="32" cy="32" r="16" fill="#ff6030" opacity="0.6"/>
      <circle cx="32" cy="32" r="10" fill="#ff9040"/>
      <circle cx="32" cy="32" r="5" fill="#ffdd80"/>
      <line x1="32" y1="10" x2="32" y2="54" stroke="#ff8040" stroke-width="1.5" opacity="0.5"/>
      <line x1="10" y1="32" x2="54" y2="32" stroke="#ff8040" stroke-width="1.5" opacity="0.5"/>
      <line x1="16" y1="16" x2="48" y2="48" stroke="#ff8040" stroke-width="1" opacity="0.4"/>
      <line x1="48" y1="16" x2="16" y2="48" stroke="#ff8040" stroke-width="1" opacity="0.4"/>`,
  },
  {
    name: 'skill-wind',
    bg: ['#0a2a1a', '#1a4a3a'],
    svg: `<!-- 风刃旋涡 -->
      <path d="M32 14 Q48 20 42 32 Q48 44 32 50 Q16 44 22 32 Q16 20 32 14Z" fill="none" stroke="#80ffc0" stroke-width="1.5"/>
      <path d="M32 20 C40 22 40 30 32 32 C24 34 24 42 32 44" fill="none" stroke="#c0ffe0" stroke-width="2" stroke-linecap="round"/>
      <circle cx="32" cy="32" r="3" fill="#c0ffe0"/>`,
  },
  {
    name: 'skill-laser',
    bg: ['#3a0a1a', '#4a1a2a'],
    svg: `<!-- 激光束 -->
      <rect x="10" y="29" width="44" height="6" rx="3" fill="#ff4060" opacity="0.5"/>
      <rect x="10" y="30" width="44" height="4" rx="2" fill="#ff6080"/>
      <rect x="10" y="31" width="44" height="2" rx="1" fill="#ffb0c0"/>
      <circle cx="10" cy="32" r="4" fill="#ff6080" opacity="0.6"/>
      <circle cx="54" cy="32" r="3" fill="#ffb0c0" opacity="0.8"/>`,
  },
  {
    name: 'skill-missile',
    bg: ['#2a1a0a', '#3a2a1a'],
    svg: `<!-- 导弹齐射 -->
      <path d="M32 18 L36 26 L32 24 L28 26 Z" fill="#ffaa40"/>
      <path d="M18 32 L26 28 L24 32 L26 36 Z" fill="#ffaa40"/>
      <path d="M46 32 L38 28 L40 32 L38 36 Z" fill="#ffaa40"/>
      <path d="M32 46 L36 38 L32 40 L28 38 Z" fill="#ffaa40"/>
      <path d="M20 20 L26 24 L24 26 Z" fill="#ffe080" opacity="0.7"/>
      <path d="M44 44 L38 40 L40 38 Z" fill="#ffe080" opacity="0.7"/>
      <circle cx="32" cy="32" r="4" fill="#ffcc60"/>`,
  },
  {
    name: 'skill-beam',
    bg: ['#3a2a0a', '#4a3a1a'],
    svg: `<!-- 审判光束 -->
      <rect x="18" y="8" width="6" height="48" rx="3" fill="#ffd060" opacity="0.6"/>
      <rect x="19" y="8" width="4" height="48" rx="2" fill="#ffe080"/>
      <rect x="20" y="8" width="2" height="48" rx="1" fill="#fff0a0"/>
      <rect x="38" y="8" width="6" height="48" rx="3" fill="#ffd060" opacity="0.6"/>
      <rect x="39" y="8" width="4" height="48" rx="2" fill="#ffe080"/>
      <rect x="40" y="8" width="2" height="48" rx="1" fill="#fff0a0"/>
      <path d="M16 32 L48 32" stroke="#ffd060" stroke-width="0.8" stroke-dasharray="3 3" opacity="0.5"/>`,
  },
  {
    name: 'skill-storm',
    bg: ['#0a1a3a', '#1a2a4a'],
    svg: `<!-- 闪电风暴 -->
      <path d="M20 12 L18 22 L22 22 L16 36" fill="none" stroke="#80a0ff" stroke-width="2" stroke-linecap="round"/>
      <path d="M44 12 L46 22 L42 22 L48 36" fill="none" stroke="#80a0ff" stroke-width="2" stroke-linecap="round"/>
      <path d="M28 28 L26 38 L30 38 L24 52" fill="none" stroke="#b0c8ff" stroke-width="2.5" stroke-linecap="round"/>
      <path d="M40 28 L38 38 L42 38 L36 52" fill="none" stroke="#b0c8ff" stroke-width="2.5" stroke-linecap="round"/>
      <line x1="10" y1="20" x2="54" y2="20" stroke="#4060a0" stroke-width="0.8" stroke-dasharray="2 4" opacity="0.5"/>`,
  },
  {
    name: 'skill-rain',
    bg: ['#1a0a3a', '#2a1a4a'],
    svg: `<!-- 审判之雨 -->
      <line x1="14" y1="12" x2="14" y2="28" stroke="#c080ff" stroke-width="1.5" stroke-linecap="round"/>
      <line x1="24" y1="8" x2="24" y2="24" stroke="#c080ff" stroke-width="1.5" stroke-linecap="round"/>
      <line x1="34" y1="14" x2="34" y2="30" stroke="#c080ff" stroke-width="1.5" stroke-linecap="round"/>
      <line x1="44" y1="10" x2="44" y2="26" stroke="#c080ff" stroke-width="1.5" stroke-linecap="round"/>
      <line x1="50" y1="16" x2="50" y2="32" stroke="#c080ff" stroke-width="1.5" stroke-linecap="round"/>
      <line x1="19" y1="30" x2="19" y2="46" stroke="#e0b0ff" stroke-width="1.5" stroke-linecap="round"/>
      <line x1="29" y1="34" x2="29" y2="50" stroke="#e0b0ff" stroke-width="1.5" stroke-linecap="round"/>
      <line x1="39" y1="28" x2="39" y2="44" stroke="#e0b0ff" stroke-width="1.5" stroke-linecap="round"/>
      <circle cx="14" cy="28" r="2" fill="#e0b0ff" opacity="0.8"/>
      <circle cx="29" cy="50" r="2" fill="#e0b0ff" opacity="0.8"/>
      <circle cx="44" cy="26" r="2" fill="#e0b0ff" opacity="0.8"/>`,
  },
  {
    name: 'skill-thunder',
    bg: ['#0a0a2a', '#1a1a3a'],
    svg: `<!-- 天罚雷击 -->
      <path d="M22 8 L18 24 L24 24 L16 44" fill="none" stroke="#a0c0ff" stroke-width="2.5" stroke-linecap="round"/>
      <path d="M32 10 L28 26 L34 26 L26 48" fill="none" stroke="#a0c0ff" stroke-width="3" stroke-linecap="round"/>
      <path d="M42 8 L38 24 L44 24 L36 44" fill="none" stroke="#a0c0ff" stroke-width="2.5" stroke-linecap="round"/>
      <circle cx="16" cy="44" r="3" fill="#d0e0ff" opacity="0.6"/>
      <circle cx="26" cy="48" r="4" fill="#d0e0ff" opacity="0.7"/>
      <circle cx="36" cy="44" r="3" fill="#d0e0ff" opacity="0.6"/>
      <line x1="8" y1="52" x2="56" y2="52" stroke="#4060a0" stroke-width="0.5" opacity="0.4"/>`,
  },
];

for (const s of skills) {
  const svg = `<svg xmlns="http://www.w3.org/2000/svg" width="64" height="64" viewBox="0 0 64 64">
    <defs>
      <linearGradient id="bg" x1="0" y1="0" x2="0" y2="1">
        <stop offset="0%" stop-color="${s.bg[0]}"/>
        <stop offset="100%" stop-color="${s.bg[1]}"/>
      </linearGradient>
      <radialGradient id="lightGlow" cx="50%" cy="40%" r="50%">
        <stop offset="0%" stop-color="white" stop-opacity="0.5"/>
        <stop offset="100%" stop-color="white" stop-opacity="0"/>
      </radialGradient>
    </defs>
    <rect width="64" height="64" rx="10" fill="url(#bg)"/>
    <rect x="1" y="1" width="62" height="62" rx="9" fill="none" stroke="white" stroke-width="0.8" stroke-opacity="0.15"/>
    ${s.svg}
  </svg>`;

  await sharp(Buffer.from(svg)).png().toFile(`${OUT}/${s.name}.png`);
  console.log(`  ✓ ${s.name}.png`);
}
console.log(`Done: ${skills.length} skill icons generated.`);
