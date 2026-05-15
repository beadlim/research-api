from PIL import Image, ImageDraw, ImageFont
import os

stages = [
    ("Stage 01\nMonolito", 2327),
    ("Stage 02\n+Users", 1793),
    ("Stage 03\n+Products", 1964),
    ("Stage 04\n+Orders/Inv", 1620),
    ("Stage 05\nFull µS", 1769),
]

W, H = 900, 540
PAD_L, PAD_R, PAD_T, PAD_B = 80, 40, 50, 100
CHART_W = W - PAD_L - PAD_R
CHART_H = H - PAD_T - PAD_B

BG    = (18, 18, 18)
GRID  = (50, 50, 50)
LINE  = (80, 200, 120)   # green
DOT   = (80, 200, 120)
LABEL = (200, 200, 200)
TITLE = (240, 240, 240)
ANNOT = (255, 220, 100)
AXIS  = (120, 120, 120)

img  = Image.new("RGB", (W, H), BG)
draw = ImageDraw.Draw(img)

try:
    font_sm    = ImageFont.truetype("/usr/share/fonts/truetype/dejavu/DejaVuSans.ttf", 13)
    font_md    = ImageFont.truetype("/usr/share/fonts/truetype/dejavu/DejaVuSans.ttf", 15)
    font_bold  = ImageFont.truetype("/usr/share/fonts/truetype/dejavu/DejaVuSans-Bold.ttf", 16)
    font_title = ImageFont.truetype("/usr/share/fonts/truetype/dejavu/DejaVuSans-Bold.ttf", 18)
except:
    font_sm = font_md = font_bold = font_title = ImageFont.load_default()

min_val, max_val = 1400, 2500
y_ticks = [1400, 1600, 1800, 2000, 2200, 2400]

def to_px(val):
    return PAD_T + CHART_H - int(((val - min_val) / (max_val - min_val)) * CHART_H)

def to_x(i):
    return PAD_L + int((i / (len(stages) - 1)) * CHART_W)

# grid + y labels
for t in y_ticks:
    y = to_px(t)
    draw.line([(PAD_L, y), (PAD_L + CHART_W, y)], fill=GRID, width=1)
    draw.text((PAD_L - 52, y - 8), f"{t}", font=font_sm, fill=AXIS)

# axes
draw.line([(PAD_L, PAD_T), (PAD_L, PAD_T + CHART_H)], fill=AXIS, width=2)
draw.line([(PAD_L, PAD_T + CHART_H), (PAD_L + CHART_W, PAD_T + CHART_H)], fill=AXIS, width=2)

# line segments
points = [(to_x(i), to_px(v)) for i, (_, v) in enumerate(stages)]
for i in range(len(points) - 1):
    draw.line([points[i], points[i+1]], fill=LINE, width=3)

annotations = {
    0: ("baseline\n2.327 req/s", "above"),
    1: ("-23% vs baseline\nNGINX + pool contention", "below"),
    2: ("+10% vs Stage 02\nmonolito menor", "above"),
    3: ("-18% vs Stage 03\n2 HTTP calls síncronas", "below"),
    4: ("+9% vs Stage 04\nschema isolation", "above"),
}

for i, (label, val) in enumerate(stages):
    x, y = points[i]
    r = 7
    draw.ellipse([(x-r, y-r), (x+r, y+r)], fill=DOT, outline=TITLE, width=2)

    val_txt = f"{val:,}".replace(",", ".")
    bbox = draw.textbbox((0,0), val_txt, font=font_bold)
    tw = bbox[2] - bbox[0]
    draw.text((x - tw//2, y - 28), val_txt, font=font_bold, fill=TITLE)

    for j, line in enumerate(label.split("\n")):
        bbox = draw.textbbox((0,0), line, font=font_sm)
        tw = bbox[2] - bbox[0]
        draw.text((x - tw//2, PAD_T + CHART_H + 12 + j*16), line, font=font_sm, fill=LABEL)

    if i in annotations:
        note, pos = annotations[i]
        for j, line in enumerate(note.split("\n")):
            bbox = draw.textbbox((0,0), line, font=font_sm)
            tw = bbox[2] - bbox[0]
            if pos == "above":
                ny = y - 56 + j*16
                nx = x - tw//2
            else:
                ny = y + 18 + j*16
                nx = x - tw//2
            # clamp to canvas
            nx = max(PAD_L + 2, min(nx, W - PAD_R - tw - 2))
            draw.text((nx, ny), line, font=font_sm, fill=ANNOT)

# title
title = "Evolução do Throughput — Migração Monolito → Microsserviços (Strangler Pattern)"
bbox = draw.textbbox((0,0), title, font=font_title)
tw = bbox[2] - bbox[0]
draw.text(((W - tw)//2, 12), title, font=font_title, fill=TITLE)

# y-axis label
y_label = "Throughput (req/s)"
bbox = draw.textbbox((0,0), y_label, font=font_md)
th = bbox[2] - bbox[0]
label_img = Image.new("RGBA", (th + 4, 20), (0,0,0,0))
ld = ImageDraw.Draw(label_img)
ld.text((0, 0), y_label, font=font_md, fill=AXIS)
label_rot = label_img.rotate(90, expand=True)
img.paste(label_rot, (8, PAD_T + CHART_H//2 - label_rot.height//2), label_rot)

out = os.path.join(os.path.dirname(__file__), "throughput_evolution.png")
img.save(out, dpi=(150, 150))
print(f"Saved: {out}")
