#!/bin/bash
# generate-enemy-anim.sh
# Generate walk/hit animation frames from old high-quality enemy SVGs.
#
# Walk frames (0-3): vertical bounce via nested <svg> inside <g transform>
#   walk-0: 0px, walk-1: -3px, walk-2: 0px, walk-3: +3px
#
# Hit frames (0-1):
#   hit-0: original + white flash overlay
#   hit-1: original (recovering)
#
# Also regenerates the static .png from the old SVG.

set -euo pipefail

SRC_DIR="${1:-/tmp/old-enemy-svgs}"
OUT_DIR="${2:-$(dirname "$0")/../assets/enemies}"
TMP_DIR="/tmp/enemy-anim-build"
SIZE=128

WALK_OFFSETS=(0 -3 0 3)

mkdir -p "$TMP_DIR" "$OUT_DIR"

count=0
total=$(ls "$SRC_DIR"/*.svg 2>/dev/null | wc -l | tr -d ' ')

for svg_file in "$SRC_DIR"/*.svg; do
  key=$(basename "$svg_file" .svg)
  count=$((count + 1))
  echo "[$count/$total] $key"

  # --- Static PNG ---
  rsvg-convert -w "$SIZE" -h "$SIZE" "$svg_file" > "$OUT_DIR/$key.png"

  # --- Walk frames: wrap entire SVG in a translated <g> ---
  for i in 0 1 2 3; do
    offset=${WALK_OFFSETS[$i]}
    tmp_svg="$TMP_DIR/${key}-walk-${i}.svg"

    # Nest the original SVG inside a wrapper with translate transform
    cat > "$tmp_svg" <<SVGEOF
<svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 ${SIZE} ${SIZE}" width="${SIZE}" height="${SIZE}">
  <g transform="translate(0, ${offset})">
    $(cat "$svg_file")
  </g>
</svg>
SVGEOF

    rsvg-convert -w "$SIZE" -h "$SIZE" "$tmp_svg" > "$OUT_DIR/${key}-walk-${i}.png"
  done

  # --- Hit frame 0: original + white flash overlay ---
  tmp_svg="$TMP_DIR/${key}-hit-0.svg"
  cat > "$tmp_svg" <<SVGEOF
<svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 ${SIZE} ${SIZE}" width="${SIZE}" height="${SIZE}">
  $(cat "$svg_file")
  <rect x="0" y="0" width="${SIZE}" height="${SIZE}" fill="white" opacity="0.45"/>
</svg>
SVGEOF
  rsvg-convert -w "$SIZE" -h "$SIZE" "$tmp_svg" > "$OUT_DIR/${key}-hit-0.png"

  # --- Hit frame 1: original (recovering) ---
  rsvg-convert -w "$SIZE" -h "$SIZE" "$svg_file" > "$OUT_DIR/${key}-hit-1.png"
done

echo ""
echo "Done: generated $(( total * 7 )) files ($total enemies x 7 frames)"

rm -rf "$TMP_DIR"
