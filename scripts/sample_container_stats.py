#!/usr/bin/env python3
"""Amostra CPU e memoria por container durante um teste de carga.

Fonte alternativa ao cAdvisor, que em hosts com Docker recente e cgroups v2
frequentemente enumera apenas o cgroup raiz e nao os containers individuais.
Le "docker stats" em intervalos regulares e grava uma serie temporal em CSV.

Uso:
    python3 scripts/sample_container_stats.py --outfile results/.../container_stats.csv \
        [--interval 5] [--project research-api]

Encerra de forma limpa em SIGTERM ou SIGINT, gravando tudo que ja coletou.
"""

import argparse
import csv
import json
import signal
import subprocess
import sys
import time

STOP = False


def _stop(signum, frame):
    global STOP
    STOP = True


def parse_mem(usage):
    """'66.82MiB / 30.8GiB' -> 66.82 (em MB)"""
    raw = usage.split("/")[0].strip()
    units = {"B": 1 / 1024 / 1024, "KiB": 1 / 1024, "MiB": 1.0, "GiB": 1024.0,
             "kB": 1 / 1000 / 1000 * 1000, "MB": 1.0, "GB": 1024.0}
    for suffix, factor in sorted(units.items(), key=lambda kv: -len(kv[0])):
        if raw.endswith(suffix):
            try:
                return round(float(raw[: -len(suffix)]) * factor, 3)
            except ValueError:
                return None
    return None


def service_of(name, project):
    """'research-api-users-service-1' -> 'users-service'"""
    base = name
    if project and base.startswith(project + "-"):
        base = base[len(project) + 1:]
    if "-" in base and base.rsplit("-", 1)[1].isdigit():
        base = base.rsplit("-", 1)[0]
    return base


def sample(project):
    out = subprocess.run(
        ["docker", "stats", "--no-stream", "--format", "{{json .}}"],
        capture_output=True, text=True, timeout=60,
    )
    rows = []
    now = time.time()
    for line in out.stdout.splitlines():
        line = line.strip()
        if not line:
            continue
        try:
            d = json.loads(line)
        except json.JSONDecodeError:
            continue
        name = d.get("Name", "")
        if project and not name.startswith(project + "-"):
            continue
        try:
            cpu = float(d.get("CPUPerc", "0%").rstrip("%"))
        except ValueError:
            cpu = None
        rows.append(
            {
                "timestamp": round(now, 3),
                "container": name,
                "service": service_of(name, project),
                "cpu_percent": cpu,
                "memory_mb": parse_mem(d.get("MemUsage", "")),
            }
        )
    return rows


def main():
    ap = argparse.ArgumentParser()
    ap.add_argument("--outfile", required=True)
    ap.add_argument("--interval", type=float, default=5.0)
    ap.add_argument("--project", default="research-api")
    args = ap.parse_args()

    signal.signal(signal.SIGTERM, _stop)
    signal.signal(signal.SIGINT, _stop)

    with open(args.outfile, "w", newline="") as fh:
        w = csv.DictWriter(
            fh, fieldnames=["timestamp", "container", "service", "cpu_percent", "memory_mb"]
        )
        w.writeheader()
        n = 0
        while not STOP:
            started = time.time()
            try:
                rows = sample(args.project)
                w.writerows(rows)
                fh.flush()
                n += len(rows)
            except Exception as exc:  # pragma: no cover
                print(f"amostragem falhou: {exc}", file=sys.stderr)
            elapsed = time.time() - started
            # dorme em fatias para reagir rapido ao sinal de parada
            remaining = max(0.0, args.interval - elapsed)
            while remaining > 0 and not STOP:
                step = min(0.5, remaining)
                time.sleep(step)
                remaining -= step
    print(f"{n} amostras gravadas em {args.outfile}")


if __name__ == "__main__":
    main()
