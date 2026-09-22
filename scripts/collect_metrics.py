#!/usr/bin/env python3
"""Exporta as metricas de recursos do Prometheus para CSV ao final de um teste de carga.

Sem esta exportacao o TSDB do Prometheus e descartado junto com o ambiente e as
metricas de CPU, memoria e pool de conexoes se perdem, que foi o que ocorreu nas
execucoes iniciais da pesquisa.

Uso:
    python3 scripts/collect_metrics.py --stage 01-monolith --start <epoch> --end <epoch> \
        --outdir results/01-monolith/run-1/metrics
"""

import argparse
import csv
import json
import os
import sys
import urllib.parse
import urllib.request

# Janelas de regime estavel, em segundos a partir do inicio do teste.
# As rampas ficam de fora para que as estatisticas reflitam apenas carga estavel.
STEADY_WINDOWS = [
    ("low", 30, 330),
    ("medium", 360, 660),
    ("high", 690, 990),
]

# Janela unica usada na validacao rapida do pipeline (SMOKE=1 em run-stage.sh).
STEADY_WINDOWS_SMOKE = [("low", 30, 115)]

PROJECT = os.environ.get("COMPOSE_PROJECT_NAME", "research-api")
SVC = f'container_label_com_docker_compose_service'
SEL = f'{{container_label_com_docker_compose_project="{PROJECT}",{SVC}!=""}}'

# Cada entrada: nome do arquivo -> (expressao PromQL, rotulo que identifica a serie, unidade)
# CPU e memoria por container vem de scripts/sample_container_stats.py, e nao do
# cAdvisor: em hosts com Docker recente e cgroups v2 o cAdvisor frequentemente
# enumera apenas o cgroup raiz, sem series por container.
QUERIES = {
    # Recursos do processo Go, via /metrics da propria aplicacao
    "process_cpu_percent": (
        "rate(process_cpu_seconds_total[1m]) * 100",
        "job",
        "percentual de um nucleo",
    ),
    "process_memory_mb": (
        "process_resident_memory_bytes / 1024 / 1024",
        "job",
        "MB",
    ),
    "goroutines": ("go_goroutines", "job", "goroutines"),
    "heap_alloc_mb": ("go_memstats_heap_alloc_bytes / 1024 / 1024", "job", "MB"),
    "gc_pause_ms": (
        "rate(go_gc_duration_seconds_sum[1m]) / "
        "clamp_min(rate(go_gc_duration_seconds_count[1m]), 0.0001) * 1000",
        "job",
        "ms",
    ),
    # Contencao no pool de conexoes do PostgreSQL
    "pool_acquired_conns": ("db_pool_acquired_conns", "service", "conexoes"),
    "pool_idle_conns": ("db_pool_idle_conns", "service", "conexoes"),
    "pool_total_conns": ("db_pool_total_conns", "service", "conexoes"),
    "pool_wait_per_acquire_ms": (
        "rate(db_pool_acquire_duration_seconds_total[1m]) / "
        "clamp_min(rate(db_pool_acquire_total[1m]), 0.0001) * 1000",
        "service",
        "ms",
    ),
    "pool_empty_acquire_per_s": (
        "rate(db_pool_empty_acquire_total[1m])",
        "service",
        "por segundo",
    ),
}


def query_range(base, expr, start, end, step):
    params = urllib.parse.urlencode(
        {"query": expr, "start": start, "end": end, "step": step}
    )
    url = f"{base}/api/v1/query_range?{params}"
    with urllib.request.urlopen(url, timeout=120) as resp:
        payload = json.load(resp)
    if payload.get("status") != "success":
        raise RuntimeError(f"Prometheus recusou a consulta: {payload}")
    return payload["data"]["result"]


def percentile(values, p):
    if not values:
        return None
    ordered = sorted(values)
    k = (len(ordered) - 1) * p
    lo, hi = int(k), min(int(k) + 1, len(ordered) - 1)
    return ordered[lo] + (ordered[hi] - ordered[lo]) * (k - lo)


def aggregate_container_stats(path, args):
    """Agrega o CSV de docker stats nas mesmas janelas de carga estavel."""
    series = {}
    with open(path) as fh:
        for r in csv.DictReader(fh):
            try:
                ts = float(r["timestamp"])
            except (TypeError, ValueError):
                continue
            for metric, unit in (("cpu_percent", "percentual de um nucleo"),
                                 ("memory_mb", "MB")):
                raw = r.get(metric)
                if raw in (None, "", "None"):
                    continue
                series.setdefault((metric, unit, r["service"]), []).append((ts, float(raw)))

    out = []
    for (metric, unit, service), points in series.items():
        for level, off_from, off_to in STEADY_WINDOWS:
            lo, hi = args.start + off_from, args.start + off_to
            vals = [v for ts, v in points if lo <= ts < hi]
            if not vals:
                continue
            out.append({
                "stage": args.stage, "run": args.run, "metric": metric, "unit": unit,
                "target": service, "load_level": level, "samples": len(vals),
                "mean": round(sum(vals) / len(vals), 3),
                "p95": round(percentile(vals, 0.95), 3),
                "max": round(max(vals), 3),
            })
    return out


def main():
    ap = argparse.ArgumentParser()
    ap.add_argument("--stage", required=True)
    ap.add_argument("--run", default="1")
    ap.add_argument("--start", type=float, required=True, help="epoch do inicio do teste")
    ap.add_argument("--end", type=float, required=True, help="epoch do fim do teste")
    ap.add_argument("--outdir", required=True)
    ap.add_argument("--prometheus", default="http://localhost:9090")
    ap.add_argument("--step", default="10s")
    ap.add_argument("--smoke", action="store_true",
                    help="usa a janela curta da validacao de pipeline")
    ap.add_argument("--container-stats", default=None,
                    help="CSV gerado por scripts/sample_container_stats.py")
    args = ap.parse_args()

    if args.smoke:
        global STEADY_WINDOWS
        STEADY_WINDOWS = STEADY_WINDOWS_SMOKE

    os.makedirs(args.outdir, exist_ok=True)
    rows = []

    for name, (expr, label, unit) in QUERIES.items():
        try:
            series = query_range(args.prometheus, expr, args.start, args.end, args.step)
        except Exception as exc:  # pragma: no cover - diagnostico em tempo de execucao
            print(f"  ! {name}: {exc}", file=sys.stderr)
            continue

        # serie temporal bruta, para eventual replotagem
        raw_path = os.path.join(args.outdir, f"{name}.csv")
        with open(raw_path, "w", newline="") as fh:
            w = csv.writer(fh)
            w.writerow(["timestamp", label, "value"])
            for s in series:
                ident = s["metric"].get(label, "?")
                for ts, val in s["values"]:
                    w.writerow([ts, ident, val])

        # agregados por patamar de carga, que alimentam as tabelas do TCC
        for s in series:
            ident = s["metric"].get(label, "?")
            for level, off_from, off_to in STEADY_WINDOWS:
                lo, hi = args.start + off_from, args.start + off_to
                vals = [
                    float(v)
                    for ts, v in s["values"]
                    if lo <= float(ts) < hi and v not in ("NaN", "+Inf", "-Inf")
                ]
                if not vals:
                    continue
                rows.append(
                    {
                        "stage": args.stage,
                        "run": args.run,
                        "metric": name,
                        "unit": unit,
                        "target": ident,
                        "load_level": level,
                        "samples": len(vals),
                        "mean": round(sum(vals) / len(vals), 3),
                        "p95": round(percentile(vals, 0.95), 3),
                        "max": round(max(vals), 3),
                    }
                )
        print(f"  - {name}: {len(series)} serie(s)")

    if args.container_stats and os.path.exists(args.container_stats):
        rows.extend(aggregate_container_stats(args.container_stats, args))
        print(f"  - container_stats: {os.path.basename(args.container_stats)}")
    elif args.container_stats:
        print(f"  ! container_stats ausente: {args.container_stats}", file=sys.stderr)

    agg_path = os.path.join(args.outdir, "resources_by_load_level.csv")
    with open(agg_path, "w", newline="") as fh:
        w = csv.DictWriter(
            fh,
            fieldnames=[
                "stage", "run", "metric", "unit", "target",
                "load_level", "samples", "mean", "p95", "max",
            ],
        )
        w.writeheader()
        w.writerows(rows)

    print(f"\n{len(rows)} agregados escritos em {agg_path}")
    if not rows:
        print("ATENCAO: nenhum dado retornado. Verifique se o Prometheus ainda "
              "estava no ar e se o intervalo informado cobre o teste.", file=sys.stderr)
        return 1
    return 0


if __name__ == "__main__":
    sys.exit(main())
