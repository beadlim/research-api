#!/usr/bin/env python3
"""Consolida os resultados de todas as execucoes em tabelas prontas para o TCC.

Le os summary.json do k6 e os CSVs de recursos exportados do Prometheus em
results/<estagio>/run-*/ e escreve, em results/consolidated/:

  latency_by_stage.csv    latencia, vazao e erro por estagio, patamar e endpoint
  resources_by_stage.csv  CPU, memoria, runtime Go e pool de conexoes
  tcc_tables.md           as mesmas tabelas ja formatadas para conferencia

Uso:
    python3 scripts/consolidate.py
"""

import csv
import glob
import json
import os
import re
import statistics
from collections import defaultdict

ROOT = os.path.dirname(os.path.dirname(os.path.abspath(__file__)))
OUT = os.path.join(ROOT, "results", "consolidated")

STAGES = [
    ("results/baseline", "Stage 01", "Monolito completo"),
    ("results/stage-01b", "Stage 01b", "Monolito atras do NGINX (controle)"),
    ("results/stage-02", "Stage 02", "Users extraido"),
    ("results/stage-03", "Stage 03", "Products extraido"),
    ("results/stage-04", "Stage 04", "Orders e Inventory extraidos"),
    ("results/stage-05", "Stage 05", "Microsservicos completos"),
    ("results/stage-05b", "Stage 05b", "Microsservicos com gateway pareado (controle)"),
]

TREND_KEYS = [("med", "p50"), ("p(90)", "p90"), ("p(95)", "p95"), ("p(99)", "p99"),
              ("avg", "avg"), ("max", "max")]


def parse_tags(name):
    """'http_req_duration{endpoint:get_user,load_level:low}' -> (base, {tags})"""
    m = re.match(r"^([^{]+)(?:\{(.*)\})?$", name)
    base, raw = m.group(1), m.group(2)
    tags = {}
    if raw:
        for part in raw.split(","):
            if ":" in part:
                k, v = part.split(":", 1)
                tags[k.strip()] = v.strip()
    return base, tags


def load_latency():
    rows = []
    for path, stage, desc in STAGES:
        for summary in sorted(glob.glob(os.path.join(ROOT, path, "run-*", "summary.json"))):
            run = re.search(r"run-(\d+)", summary).group(1)
            metrics = json.load(open(summary)).get("metrics", {})

            overall_reqs = metrics.get("http_reqs", {})
            overall_fail = metrics.get("http_req_failed", {})

            for name, vals in metrics.items():
                base, tags = parse_tags(name)
                if base != "http_req_duration":
                    continue
                if tags.get("scenario") or tags.get("expected_response"):
                    continue
                level = tags.get("load_level", "todos")
                if level == "ramp":
                    continue
                # o k6 materializa a submetrica mesmo quando nenhuma requisicao
                # caiu naquele recorte; descarta para nao reportar zero como dado
                if not vals.get("count", 1) and not vals.get("max"):
                    continue
                if vals.get("max") in (0, None) and vals.get("med") in (0, None):
                    continue
                row = {
                    "stage": stage,
                    "descricao": desc,
                    "run": run,
                    "load_level": level,
                    "endpoint": tags.get("endpoint", "todos"),
                }
                for src, dst in TREND_KEYS:
                    v = vals.get(src)
                    row[dst + "_ms"] = round(v, 2) if isinstance(v, (int, float)) else ""
                # vazao e erro so existem de forma agregada por execucao
                if row["endpoint"] == "todos" and level == "todos":
                    row["throughput_req_s"] = round(overall_reqs.get("rate", 0), 1)
                    row["total_requests"] = overall_reqs.get("count", "")
                    row["error_rate"] = round(overall_fail.get("value", 0) * 100, 3)
                else:
                    ep_fail = metrics.get(f"http_req_failed{{endpoint:{tags.get('endpoint')}}}", {})
                    row["throughput_req_s"] = ""
                    row["total_requests"] = ""
                    row["error_rate"] = (
                        round(ep_fail.get("value", 0) * 100, 3) if ep_fail else ""
                    )
                rows.append(row)
    return rows


def load_resources():
    rows = []
    for path, stage, desc in STAGES:
        for csv_path in sorted(
            glob.glob(os.path.join(ROOT, path, "run-*", "metrics", "resources_by_load_level.csv"))
        ):
            for r in csv.DictReader(open(csv_path)):
                r["stage"] = stage
                r["descricao"] = desc
                rows.append(r)
    return rows


def write_csv(path, rows, fields):
    os.makedirs(os.path.dirname(path), exist_ok=True)
    with open(path, "w", newline="") as fh:
        w = csv.DictWriter(fh, fieldnames=fields, extrasaction="ignore")
        w.writeheader()
        w.writerows(rows)


def mean_sd(values):
    values = [v for v in values if isinstance(v, (int, float))]
    if not values:
        return "", ""
    m = statistics.mean(values)
    sd = statistics.stdev(values) if len(values) > 1 else 0.0
    return round(m, 2), round(sd, 2)


def markdown_tables(lat, res):
    lines = ["# Tabelas consolidadas", ""]

    lines += ["## Desempenho geral por estagio (media das repeticoes)", "",
              "| Estagio | Execucoes | Vazao (req/s) | P50 (ms) | P95 (ms) | P99 (ms) | Erro (%) |",
              "|---|---|---|---|---|---|---|"]
    by_stage = defaultdict(list)
    for r in lat:
        if r["endpoint"] == "todos" and r["load_level"] == "todos":
            by_stage[r["stage"]].append(r)
    for stage in sorted(by_stage):
        rs = by_stage[stage]
        f = lambda k: mean_sd([x[k] for x in rs if x[k] != ""])
        tp, tp_sd = f("throughput_req_s")
        p50, p50_sd = f("p50_ms")
        p95, p95_sd = f("p95_ms")
        p99, p99_sd = f("p99_ms")
        er, _ = f("error_rate")
        cell = lambda m, sd: f"{m} ± {sd}" if m != "" else "—"
        lines.append(
            f"| {stage} | {len(rs)} | {cell(tp, tp_sd)} | {cell(p50, p50_sd)} | "
            f"{cell(p95, p95_sd)} | {cell(p99, p99_sd)} | {er if er != '' else '—'} |"
        )

    lines += ["", "## P95 por patamar de carga (ms)", "",
              "| Estagio | 50 VUs | 200 VUs | 500 VUs |", "|---|---|---|---|"]
    grid = defaultdict(dict)
    for r in lat:
        if r["endpoint"] == "todos" and r["load_level"] in ("low", "medium", "high"):
            grid[r["stage"]].setdefault(r["load_level"], []).append(r["p95_ms"])
    for stage in sorted(grid):
        cells = []
        for lvl in ("low", "medium", "high"):
            m, sd = mean_sd(grid[stage].get(lvl, []))
            cells.append(f"{m} ± {sd}" if m != "" else "—")
        lines.append(f"| {stage} | " + " | ".join(cells) + " |")

    lines += ["", "## P95 do POST /orders por estagio (ms)", "",
              "| Estagio | 50 VUs | 200 VUs | 500 VUs |", "|---|---|---|---|"]
    grid = defaultdict(dict)
    for r in lat:
        if r["endpoint"] == "post_order" and r["load_level"] in ("low", "medium", "high"):
            grid[r["stage"]].setdefault(r["load_level"], []).append(r["p95_ms"])
    for stage in sorted(grid):
        cells = []
        for lvl in ("low", "medium", "high"):
            m, sd = mean_sd(grid[stage].get(lvl, []))
            cells.append(f"{m} ± {sd}" if m != "" else "—")
        lines.append(f"| {stage} | " + " | ".join(cells) + " |")

    for metric, titulo, unidade in [
        ("cpu_percent", "CPU por servico sob carga alta", "% de um nucleo"),
        ("memory_mb", "Memoria por servico sob carga alta", "MB"),
        ("pool_wait_per_acquire_ms", "Espera media por conexao no pool sob carga alta", "ms"),
    ]:
        lines += ["", f"## {titulo} ({unidade})", "",
                  "| Estagio | Servico | Media | P95 | Maximo |", "|---|---|---|---|---|"]
        sel = [r for r in res if r["metric"] == metric and r["load_level"] == "high"]
        agg = defaultdict(lambda: defaultdict(list))
        for r in sel:
            agg[r["stage"]][r["target"]].append(r)
        for stage in sorted(agg):
            for target in sorted(agg[stage]):
                rs = agg[stage][target]
                m, _ = mean_sd([float(x["mean"]) for x in rs])
                p, _ = mean_sd([float(x["p95"]) for x in rs])
                mx, _ = mean_sd([float(x["max"]) for x in rs])
                lines.append(f"| {stage} | {target} | {m} | {p} | {mx} |")
        if not sel:
            lines.append("| — | — | — | — | — |")

    return "\n".join(lines) + "\n"


def main():
    lat = load_latency()
    res = load_resources()

    write_csv(
        os.path.join(OUT, "latency_by_stage.csv"), lat,
        ["stage", "descricao", "run", "load_level", "endpoint",
         "p50_ms", "p90_ms", "p95_ms", "p99_ms", "avg_ms", "max_ms",
         "throughput_req_s", "total_requests", "error_rate"],
    )
    write_csv(
        os.path.join(OUT, "resources_by_stage.csv"), res,
        ["stage", "descricao", "run", "metric", "unit", "target",
         "load_level", "samples", "mean", "p95", "max"],
    )
    with open(os.path.join(OUT, "tcc_tables.md"), "w") as fh:
        fh.write(markdown_tables(lat, res))

    print(f"{len(lat)} linhas de latencia e {len(res)} linhas de recursos consolidadas.")
    print(f"Saida em {OUT}")
    if not lat:
        print("\nNenhum summary.json encontrado em results/<estagio>/run-*/. "
              "Rode scripts/run-stage.sh antes de consolidar.")


if __name__ == "__main__":
    main()
