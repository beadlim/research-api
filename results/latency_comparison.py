from PIL import Image, ImageDraw, ImageFont
import os

stages_labels = ["Stage 01\nMonolito", "Stage 02\n+Users", "Stage 03\n+Products", "Stage 04\n+Orders/Inv", "Stage 05\nFull µS"]

data = {
    "P50":  [1.98,   9.43,   9.13,   11.42,   6.52],
    "P90":  [6.92,  102.05,  58.83,  163.34,  97.98],
    "P95":  [10.12, 160.89,  90.19,  286.41, 216.99],
}

COLORS = {
    "P50": (100, 220, 130),   # green
    "P90": (99,  179, 237),   # blue
    "P95": (255, 140,  80),   # orange
}

W, H = 960, 560
PAD_L, PAD_R, PAD_T, PAD_B = 80, 160, 60, 100
CHART_W = W - PAD_L - PAD_R
CHART_H = H - PAD_T - PAD_B

BG    = (18, 18, 18)
GRID  = (50, 50, 50)
TITLE = (240, 240, 240)
LABEL = (200, 200, 200)
AXIS  = (120, 120, 120)

img  = Image.new("RGB", (W, H), BG)
draw = ImageDraw.Draw(img)

try:
    font_sm    = ImageFont.truetype("/usr/share/fonts/truetype/dejavu/DejaVuSans.ttf", 12)
    font_md    = ImageFont.truetype("/usr/share/fonts/truetype/dejavu/DejaVuSans.ttf", 14)
    font_bold  = ImageFont.truetype("/usr/share/fonts/truetype/dejavu/DejaVuSans-Bold.ttf", 14)
    font_title = ImageFont.truetype("/usr/share/fonts/truetype/dejavu/DejaVuSans-Bold.ttf", 17)
except:
    font_sm = font_md = font_bold = font_title = ImageFont.load_default()

min_val, max_val = 0, 320
y_ticks = [0, 50, 100, 150, 200, 250, 300]

def to_px(val):
    return PAD_T + CHART_H - int(((val - min_val) / (max_val - min_val)) * CHART_H)

def to_x(i):
    return PAD_L + int((i / (len(stages_labels) - 1)) * CHART_W)

# grid + y labels
for t in y_ticks:
    y = to_px(t)
    draw.line([(PAD_L, y), (PAD_L + CHART_W, y)], fill=GRID, width=1)
    draw.text((PAD_L - 52, y - 8), f"{t}ms", font=font_sm, fill=AXIS)

# axes
draw.line([(PAD_L, PAD_T), (PAD_L, PAD_T + CHART_H)], fill=AXIS, width=2)
draw.line([(PAD_L, PAD_T + CHART_H), (PAD_L + CHART_W, PAD_T + CHART_H)], fill=AXIS, width=2)

# lines + dots per metric
for metric, values in data.items():
    color = COLORS[metric]
    points = [(to_x(i), to_px(v)) for i, v in enumerate(values)]
    for i in range(len(points) - 1):
        draw.line([points[i], points[i+1]], fill=color, width=2)
    for x, y in points:
        r = 5
        draw.ellipse([(x-r, y-r), (x+r, y+r)], fill=color, outline=TITLE, width=1)

# value labels — only P95 to avoid clutter, offset others slightly
label_offsets = {"P50": -2, "P90": -18, "P95": -32}
for metric, values in data.items():
    color = COLORS[metric]
    for i, v in enumerate(values):
        x, y = to_x(i), to_px(v)
        txt = f"{v:.0f}"
        bbox = draw.textbbox((0,0), txt, font=font_sm)
        tw = bbox[2] - bbox[0]
        offset = label_offsets[metric]
        draw.text((x - tw//2, y + offset - 14), txt, font=font_sm, fill=color)

# stage labels on x axis
for i, label in enumerate(stages_labels):
    x = to_x(i)
    for j, line in enumerate(label.split("\n")):
        bbox = draw.textbbox((0,0), line, font=font_sm)
        tw = bbox[2] - bbox[0]
        draw.text((x - tw//2, PAD_T + CHART_H + 12 + j*16), line, font=font_sm, fill=LABEL)

# legend (right side)
legend_x = W - PAD_R + 10
legend_y = PAD_T + 40
for metric, color in COLORS.items():
    draw.rectangle([(legend_x, legend_y), (legend_x + 24, legend_y + 12)], fill=color)
    draw.text((legend_x + 30, legend_y - 1), metric, font=font_bold, fill=color)
    legend_y += 30

# title
title = "Latência por Percentil — Evolução ao longo da Migração"
bbox = draw.textbbox((0,0), title, font=font_title)
tw = bbox[2] - bbox[0]
draw.text(((W - PAD_R - PAD_L - tw)//2 + PAD_L, 14), title, font=font_title, fill=TITLE)

# y-axis label
y_label = "Latência (ms)"
bbox = draw.textbbox((0,0), y_label, font=font_md)
th = bbox[2] - bbox[0]
label_img = Image.new("RGBA", (th + 4, 20), (0,0,0,0))
ld = ImageDraw.Draw(label_img)
ld.text((0, 0), y_label, font=font_md, fill=AXIS)
label_rot = label_img.rotate(90, expand=True)
img.paste(label_rot, (8, PAD_T + CHART_H//2 - label_rot.height//2), label_rot)

out = os.path.join(os.path.dirname(__file__), "latency_comparison.png")
img.save(out, dpi=(150, 150))
print(f"Saved: {out}")
