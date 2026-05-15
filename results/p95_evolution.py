from PIL import Image, ImageDraw, ImageFont
import os

stages = [
    ("Stage 01\nMonolito", 10.12),
    ("Stage 02\n+Users", 160.89),
    ("Stage 03\n+Products", 90.19),
    ("Stage 04\n+Orders/Inv", 286.41),
    ("Stage 05\nFull µS", 216.99),
]

W, H = 900, 540
PAD_L, PAD_R, PAD_T, PAD_B = 80, 40, 50, 100
CHART_W = W - PAD_L - PAD_R
CHART_H = H - PAD_T - PAD_B

BG      = (18, 18, 18)
GRID    = (50, 50, 50)
LINE    = (99, 179, 237)   # blue
DOT     = (99, 179, 237)
LABEL   = (200, 200, 200)
TITLE   = (240, 240, 240)
ANNOT   = (255, 220, 100)  # yellow for annotations
AXIS    = (120, 120, 120)

img  = Image.new("RGB", (W, H), BG)
draw = ImageDraw.Draw(img)

# try to load a font, fall back to default
try:
    font_sm   = ImageFont.truetype("/usr/share/fonts/truetype/dejavu/DejaVuSans.ttf", 13)
    font_md   = ImageFont.truetype("/usr/share/fonts/truetype/dejavu/DejaVuSans.ttf", 15)
    font_bold = ImageFont.truetype("/usr/share/fonts/truetype/dejavu/DejaVuSans-Bold.ttf", 16)
    font_title= ImageFont.truetype("/usr/share/fonts/truetype/dejavu/DejaVuSans-Bold.ttf", 18)
except:
    font_sm = font_md = font_bold = font_title = ImageFont.load_default()

max_val = 320
y_ticks = [0, 50, 100, 150, 200, 250, 300]

def to_px(val):
    return PAD_T + CHART_H - int((val / max_val) * CHART_H)

def to_x(i):
    return PAD_L + int((i / (len(stages) - 1)) * CHART_W)

# grid lines + y labels
for t in y_ticks:
    y = to_px(t)
    draw.line([(PAD_L, y), (PAD_L + CHART_W, y)], fill=GRID, width=1)
    draw.text((PAD_L - 50, y - 8), f"{t}ms", font=font_sm, fill=AXIS)

# axes
draw.line([(PAD_L, PAD_T), (PAD_L, PAD_T + CHART_H)], fill=AXIS, width=2)
draw.line([(PAD_L, PAD_T + CHART_H), (PAD_L + CHART_W, PAD_T + CHART_H)], fill=AXIS, width=2)

# line segments
points = [(to_x(i), to_px(v)) for i, (_, v) in enumerate(stages)]
for i in range(len(points) - 1):
    draw.line([points[i], points[i+1]], fill=LINE, width=3)

# dots + value labels + stage labels
annotations = {
    1: ("pior P95\n+1491% vs baseline", "above"),
    3: ("pior P95 µS\n2 HTTP calls síncronas", "above"),
    2: ("schema isolation\nreduz contenção", "below"),
    4: ("schema-per-service\n-24% vs Stage 04", "below_left"),
}

for i, (label, val) in enumerate(stages):
    x, y = points[i]
    r = 7
    draw.ellipse([(x-r, y-r), (x+r, y+r)], fill=DOT, outline=TITLE, width=2)

    # value above dot
    val_txt = f"{val:.0f}ms"
    bbox = draw.textbbox((0,0), val_txt, font=font_bold)
    tw = bbox[2] - bbox[0]
    draw.text((x - tw//2, y - 28), val_txt, font=font_bold, fill=TITLE)

    # stage label below axis
    for j, line in enumerate(label.split("\n")):
        bbox = draw.textbbox((0,0), line, font=font_sm)
        tw = bbox[2] - bbox[0]
        draw.text((x - tw//2, PAD_T + CHART_H + 12 + j*16), line, font=font_sm, fill=LABEL)

    # annotations for key points
    if i in annotations:
        note, pos = annotations[i]
        for j, line in enumerate(note.split("\n")):
            bbox = draw.textbbox((0,0), line, font=font_sm)
            tw = bbox[2] - bbox[0]
            if pos == "above":
                ny = y - 58 + j*16
                nx = x - tw//2
            elif pos == "below_left":
                ny = y + 20 + j*16
                nx = x - tw - 8
            else:
                ny = y + 20 + j*16
                nx = x - tw//2
            draw.text((nx, ny), line, font=font_sm, fill=ANNOT)

# title + y-axis label
title = "Evolução do P95 — Migração Monolito → Microsserviços (Strangler Pattern)"
bbox = draw.textbbox((0,0), title, font=font_title)
tw = bbox[2] - bbox[0]
draw.text(((W - tw)//2, 12), title, font=font_title, fill=TITLE)

y_label = "Latência P95 (ms)"
bbox = draw.textbbox((0,0), y_label, font=font_md)
th = bbox[2] - bbox[0]
label_img = Image.new("RGBA", (th + 4, 20), (0,0,0,0))
ld = ImageDraw.Draw(label_img)
ld.text((0, 0), y_label, font=font_md, fill=AXIS)
label_rot = label_img.rotate(90, expand=True)
img.paste(label_rot, (8, PAD_T + CHART_H//2 - label_rot.height//2), label_rot)

out = os.path.join(os.path.dirname(__file__), "p95_evolution.png")
img.save(out, dpi=(150, 150))
print(f"Saved: {out}")
