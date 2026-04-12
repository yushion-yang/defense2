#!/usr/bin/env python3
"""Generate gatling tower sprites (128x128 RGBA) matching the game's visual style.

Style reference: hexagonal base, dark blue/teal color scheme, glowing center eye,
weapon apparatus on top. Attack frames have bright energy glow effects.

Gatling design: 3-barrel rotary cannon on a heavy hexagonal base.
"""

import math
from PIL import Image, ImageDraw, ImageFilter

W, H = 128, 128
CX, CY = 64, 68  # base center (slightly lower to leave room for barrels)

# Color palette (matching existing tower style)
BASE_DARK = (30, 45, 65, 255)
BASE_MID = (45, 70, 100, 255)
BASE_LIGHT = (60, 90, 130, 255)
BASE_EDGE = (35, 55, 80, 255)
EYE_CORE = (60, 200, 255, 255)
EYE_GLOW = (80, 180, 240, 180)
BARREL_DARK = (40, 55, 75, 255)
BARREL_MID = (55, 75, 100, 255)
BARREL_TIP = (70, 90, 120, 255)
MUZZLE_FLASH = (180, 230, 255, 255)
MUZZLE_GLOW = (100, 200, 255, 200)
GLOW_SOFT = (60, 180, 255, 40)


def draw_hexagon(draw, cx, cy, r, fill, outline=None, width=1):
    """Draw a filled hexagon (flat-top)."""
    pts = []
    for i in range(6):
        angle = math.radians(60 * i + 30)  # flat-top hex
        pts.append((cx + r * math.cos(angle), cy + r * math.sin(angle)))
    draw.polygon(pts, fill=fill, outline=outline, width=width)


def draw_base(draw, cx, cy):
    """Draw the heavy hexagonal base."""
    # Shadow/depth
    draw_hexagon(draw, cx, cy + 3, 32, fill=(20, 30, 45, 200))
    # Main base
    draw_hexagon(draw, cx, cy, 30, fill=BASE_DARK, outline=BASE_EDGE, width=2)
    # Inner hex highlight
    draw_hexagon(draw, cx, cy - 1, 24, fill=BASE_MID)
    # Top face bevel
    draw_hexagon(draw, cx, cy - 2, 20, fill=BASE_LIGHT)
    draw_hexagon(draw, cx, cy - 2, 16, fill=BASE_MID)


def draw_eye(draw, cx, cy, intensity=1.0):
    """Draw the central glowing eye/core."""
    r = 8
    # Outer glow ring
    for i in range(4):
        ri = r + 4 - i
        alpha = int(40 * intensity)
        draw.ellipse([cx - ri, cy - ri, cx + ri, cy + ri],
                     fill=(60, 180, 255, alpha))
    # Core
    draw.ellipse([cx - r, cy - r, cx + r, cy + r],
                 fill=(40, 60, 90, 255))
    draw.ellipse([cx - r + 2, cy - r + 2, cx + r - 2, cy + r - 2],
                 fill=EYE_CORE if intensity > 0.5 else (40, 120, 180, 255))
    # Highlight
    draw.ellipse([cx - 3, cy - 4, cx + 1, cy - 1],
                 fill=(200, 240, 255, int(180 * intensity)))


def draw_barrel(draw, cx, cy, angle, length, firing=False, flash_intensity=0.0):
    """Draw a single barrel at given angle from center."""
    # Barrel origin (offset from center)
    ox = cx + math.cos(angle) * 4
    oy = cy + math.sin(angle) * 4
    # Barrel end
    ex = cx + math.cos(angle) * length
    ey = cy + math.sin(angle) * length

    # Barrel body (thick line)
    perp_x = -math.sin(angle) * 3
    perp_y = math.cos(angle) * 3
    barrel_pts = [
        (ox + perp_x, oy + perp_y),
        (ox - perp_x, oy - perp_y),
        (ex - perp_x * 0.7, ey - perp_y * 0.7),
        (ex + perp_x * 0.7, ey + perp_y * 0.7),
    ]
    draw.polygon(barrel_pts, fill=BARREL_DARK, outline=BARREL_MID)

    # Barrel tip highlight
    tip_r = 2
    draw.ellipse([ex - tip_r, ey - tip_r, ex + tip_r, ey + tip_r],
                 fill=BARREL_TIP)

    if firing and flash_intensity > 0:
        # Muzzle flash
        fr = int(6 * flash_intensity)
        for i in range(fr, 0, -1):
            alpha = int(200 * flash_intensity * (i / fr))
            draw.ellipse([ex - i, ey - i, ex + i, ey + i],
                         fill=(180, 230, 255, alpha))


def draw_mount(draw, cx, cy):
    """Draw the barrel mounting ring."""
    r = 10
    draw.ellipse([cx - r, cy - r, cx + r, cy + r],
                 fill=BARREL_DARK, outline=BASE_EDGE, width=1)
    draw.ellipse([cx - r + 2, cy - r + 2, cx + r - 2, cy + r - 2],
                 fill=BARREL_MID)


def create_frame(rotation_deg=0, firing=False, flash_intensity=0.0, glow_intensity=1.0):
    """Create a single sprite frame."""
    img = Image.new("RGBA", (W, H), (0, 0, 0, 0))
    draw = ImageDraw.Draw(img)

    base_cy = CY
    barrel_cy = CY - 18  # barrels above base

    # Base
    draw_base(draw, CX, base_cy)

    # Central mount
    draw_mount(draw, CX, barrel_cy)

    # 3 barrels, evenly spaced 120 degrees, pointing upward
    barrel_length = 28
    base_angle = math.radians(-90 + rotation_deg)  # -90 = pointing up

    for i in range(3):
        angle = base_angle + math.radians(120 * i)
        is_front = i == 0  # only front barrel has flash
        draw_barrel(draw, CX, barrel_cy, angle, barrel_length,
                    firing=firing and is_front,
                    flash_intensity=flash_intensity if is_front else 0)

    # Eye (on base, below mount)
    draw_eye(draw, CX, base_cy - 2, intensity=glow_intensity)

    # Overall glow effect for firing
    if firing and flash_intensity > 0.3:
        glow_layer = Image.new("RGBA", (W, H), (0, 0, 0, 0))
        glow_draw = ImageDraw.Draw(glow_layer)
        gr = int(20 * flash_intensity)
        glow_draw.ellipse([CX - gr, barrel_cy - barrel_length - gr,
                           CX + gr, barrel_cy - barrel_length + gr],
                          fill=(100, 200, 255, int(60 * flash_intensity)))
        glow_layer = glow_layer.filter(ImageFilter.GaussianBlur(radius=8))
        img = Image.alpha_composite(img, glow_layer)

    return img


def main():
    out_dir = "assets/towers/gatling"

    # Idle frames: gentle rotation cycle
    idle_0 = create_frame(rotation_deg=0, firing=False, glow_intensity=0.8)
    idle_1 = create_frame(rotation_deg=15, firing=False, glow_intensity=1.0)

    idle_0.save(f"{out_dir}/tower-gatling-idle-0.png")
    idle_1.save(f"{out_dir}/tower-gatling-idle-1.png")

    # Attack frames: rotation + muzzle flash
    attack_0 = create_frame(rotation_deg=0, firing=False, glow_intensity=1.0)
    attack_1 = create_frame(rotation_deg=40, firing=True, flash_intensity=1.0, glow_intensity=1.2)
    attack_2 = create_frame(rotation_deg=80, firing=True, flash_intensity=0.5, glow_intensity=0.9)

    attack_0.save(f"{out_dir}/tower-gatling-attack-0.png")
    attack_1.save(f"{out_dir}/tower-gatling-attack-1.png")
    attack_2.save(f"{out_dir}/tower-gatling-attack-2.png")

    print(f"Generated 5 gatling sprites in {out_dir}/")


if __name__ == "__main__":
    main()
