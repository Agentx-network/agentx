#!/usr/bin/env python3
"""
Generate the DMG installer background image.

The macOS DMG release workflow (.github/workflows/release-macos-desktop.yml)
auto-detects dmg-background.png at this path and passes it to create-dmg as
the mounted volume's background. The app icon and Applications shortcut get
drawn on top of this image at the coordinates configured in that workflow:

    APP_ICON     at (165, 200)   ← --icon "...app" 165 200
    APPLICATIONS at (495, 200)   ← --app-drop-link 495 200

So the arrow in this image is drawn in the clear strip between those two
icon zones, pointing from the app toward the install target.

Regenerate after tweaking colors or layout:
    python3 cmd/agentx-desktop/build/darwin/generate-dmg-background.py
"""
from PIL import Image, ImageDraw, ImageFilter, ImageFont
import os

WIDTH, HEIGHT = 660, 400

# Must stay in sync with create-dmg's --icon and --app-drop-link positions
# in .github/workflows/release-macos-desktop.yml.
APP_X, APP_Y = 165, 200
DROP_X, DROP_Y = 495, 200
ICON_HALF = 64  # 128px icons

# AgentX brand palette (matches the website + README badges).
COLOR_TOP = (10, 0, 21)         # #0a0015 — deep purple/black
COLOR_BOT = (26, 0, 48)         # #1a0030 — slightly lighter purple
COLOR_ARROW = (174, 0, 255)     # #AE00FF — neon purple
COLOR_GLOW = (255, 0, 146)      # #FF0092 — magenta glow
COLOR_LABEL = (200, 150, 230)   # soft lavender for caption

OUT_PATH = os.path.join(os.path.dirname(os.path.abspath(__file__)), 'dmg-background.png')


def vertical_gradient(w, h, top, bot):
    """Top-to-bottom RGB gradient via per-row interpolation."""
    img = Image.new('RGB', (w, h))
    d = ImageDraw.Draw(img)
    for y in range(h):
        t = y / (h - 1)
        r = int(top[0] + (bot[0] - top[0]) * t)
        g = int(top[1] + (bot[1] - top[1]) * t)
        b = int(top[2] + (bot[2] - top[2]) * t)
        d.line([(0, y), (w, y)], fill=(r, g, b))
    return img.convert('RGBA')


def draw_arrow(size):
    """Draw the arrow on a transparent overlay so we can glow it later."""
    overlay = Image.new('RGBA', size, (0, 0, 0, 0))
    d = ImageDraw.Draw(overlay)

    # Clear of the icon footprints, with a small visual gap.
    gap = 24
    start_x = APP_X + ICON_HALF + gap   # 253
    end_x = DROP_X - ICON_HALF - gap    # 407
    y = APP_Y

    shaft_thickness = 9
    head_w, head_h = 28, 28

    # Shaft — stop short of the arrowhead so the join is clean.
    d.line(
        [(start_x, y), (end_x - head_w + 2, y)],
        fill=COLOR_ARROW + (255,),
        width=shaft_thickness,
    )
    # Rounded tail.
    r = shaft_thickness // 2
    d.ellipse([start_x - r, y - r, start_x + r, y + r], fill=COLOR_ARROW + (255,))

    # Arrowhead.
    d.polygon(
        [
            (end_x, y),
            (end_x - head_w, y - head_h // 2),
            (end_x - head_w, y + head_h // 2),
        ],
        fill=COLOR_ARROW + (255,),
    )
    return overlay


def glow_layer(arrow_overlay, color, radius=14, alpha=150):
    """Re-color the arrow's alpha into a glow color and blur it."""
    r, g, b = color
    # Take just the alpha channel and tint it.
    alpha_ch = arrow_overlay.split()[-1]
    glow = Image.new('RGBA', arrow_overlay.size, (r, g, b, 0))
    # Scale arrow's alpha to the desired glow intensity.
    scaled = alpha_ch.point(lambda a: min(alpha, a))
    glow.putalpha(scaled)
    return glow.filter(ImageFilter.GaussianBlur(radius))


def add_caption(img):
    """Subtle 'DRAG TO INSTALL' caption below the arrow."""
    d = ImageDraw.Draw(img)
    text = "DRAG TO INSTALL"

    # Try a few common system fonts; fall back to PIL's bitmap default.
    font = None
    for candidate in (
        "/System/Library/Fonts/Helvetica.ttc",
        "/System/Library/Fonts/SFNS.ttf",
        "/Library/Fonts/Arial.ttf",
    ):
        if os.path.exists(candidate):
            try:
                font = ImageFont.truetype(candidate, 13)
                break
            except OSError:
                continue
    if font is None:
        font = ImageFont.load_default()

    bbox = d.textbbox((0, 0), text, font=font)
    tw = bbox[2] - bbox[0]
    th = bbox[3] - bbox[1]
    tx = (WIDTH - tw) // 2
    ty = APP_Y + 50

    # Soft shadow then the text.
    d.text((tx + 1, ty + 1), text, fill=(0, 0, 0, 180), font=font)
    d.text((tx, ty), text, fill=COLOR_LABEL + (255,), font=font)


def main():
    bg = vertical_gradient(WIDTH, HEIGHT, COLOR_TOP, COLOR_BOT)
    arrow = draw_arrow(bg.size)

    # Two-pass glow: wide soft halo + tight bright halo, then the solid arrow.
    halo = glow_layer(arrow, COLOR_GLOW, radius=22, alpha=110)
    inner = glow_layer(arrow, COLOR_ARROW, radius=8, alpha=180)

    composited = Image.alpha_composite(bg, halo)
    composited = Image.alpha_composite(composited, inner)
    composited = Image.alpha_composite(composited, arrow)
    add_caption(composited)

    composited.convert('RGB').save(OUT_PATH, 'PNG', optimize=True)
    print(f"Wrote {OUT_PATH} ({os.path.getsize(OUT_PATH)} bytes)")


if __name__ == '__main__':
    main()
