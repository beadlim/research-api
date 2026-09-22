#!/usr/bin/env bash
# Executa um estagio da pesquisa de ponta a ponta: sobe o ambiente, aguarda a
# aplicacao responder, roda o teste de carga do k6, exporta as metricas de
# recursos do Prometheus para CSV e derruba o ambiente.
#
# Uso:
#   scripts/run-stage.sh <01|01b|02|03|04|05|05b> [repeticoes]
#
# Exemplo (3 repeticoes do baseline):
#   scripts/run-stage.sh 01 3

set -euo pipefail

STAGE="${1:?informe o estagio: 01, 01b, 02, 03, 04, 05 ou 05b}"
REPS="${2:-1}"
# START_REP permite retomar de uma repeticao intermediaria sem sobrescrever as
# anteriores. Ex.: START_REP=2 scripts/run-stage.sh 04 3 roda apenas 2 e 3.
START_REP="${START_REP:-1}"

ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
cd "$ROOT"

case "$STAGE" in
  01) COMPOSE=docker-compose.01-monolith.yml;            SCRIPT=baseline.js;                        OUTDIR=results/baseline ;;
  01b) COMPOSE=docker-compose.01b-monolith-nginx.yml;    SCRIPT=baseline.js;                        OUTDIR=results/stage-01b ;;
  02) COMPOSE=docker-compose.02-users-extracted.yml;     SCRIPT=stage-02-users-extracted.js;        OUTDIR=results/stage-02 ;;
  03) COMPOSE=docker-compose.03-products-extracted.yml;  SCRIPT=stage-03-products-extracted.js;     OUTDIR=results/stage-03 ;;
  04) COMPOSE=docker-compose.04-orders-inventory.yml;    SCRIPT=stage-04-orders-inventory.js;       OUTDIR=results/stage-04 ;;
  05) COMPOSE=docker-compose.05-full-microservices.yml;  SCRIPT=stage-05-full-microservices.js;     OUTDIR=results/stage-05 ;;
  05b) COMPOSE=docker-compose.05b-full-microservices-control.yml; SCRIPT=stage-05-full-microservices.js; OUTDIR=results/stage-05b ;;
  *)  echo "estagio invalido: $STAGE" >&2; exit 1 ;;
esac

export COMPOSE_PROJECT_NAME=research-api

# SMOKE=1 roda uma versao curta (2 min, 30 VUs) apenas para validar que o
# pipeline inteiro funciona antes de investir nas execucoes de verdade.
if [[ "${SMOKE:-0}" == "1" ]]; then
  K6_EXTRA="--duration 120s --vus 30"
  COLLECT_EXTRA="--smoke"
  echo "### modo de validacao rapida: os resultados NAO valem para o TCC"
else
  K6_EXTRA=""
  COLLECT_EXTRA=""
fi
BASE_URL="${BASE_URL:-http://localhost:8080}"

# O Grafana e opcional no fluxo de coleta (os dados saem em CSV). Se a porta
# padrao estiver ocupada, escolhe outra livre em vez de abortar a execucao.
if [[ -z "${GRAFANA_PORT:-}" ]]; then
  GRAFANA_PORT=3000
  for candidate in 3000 3001 3002 3003; do
    if ! (exec 3<>/dev/tcp/127.0.0.1/$candidate) 2>/dev/null; then
      GRAFANA_PORT=$candidate
      break
    fi
  done
fi
export GRAFANA_PORT
echo "Grafana em http://localhost:${GRAFANA_PORT}"
# Constroi as imagens uma unica vez, fora do laco: dentro dele o "up" fica mais
# leve e menos sujeito a travar.
echo "construindo as imagens do estagio ${STAGE} ..."
timeout "${BUILD_TIMEOUT:-900}" docker compose -f "$COMPOSE" build >/dev/null 2>&1 \
  || echo "aviso: a construcao das imagens retornou erro; o up tentara novamente"
PROMETHEUS="${PROMETHEUS:-http://localhost:9090}"

wait_ready() {
  echo "  aguardando a aplicacao responder em ${BASE_URL}/products ..."
  for _ in $(seq 1 60); do
    if curl -fsS -o /dev/null "${BASE_URL}/products" 2>/dev/null; then
      echo "  aplicacao pronta"
      return 0
    fi
    sleep 2
  done
  echo "  aplicacao nao respondeu em 120s" >&2
  return 1
}

# O "docker compose up" ja travou indefinidamente com todos os containers no ar
# e saudaveis, consumindo uma noite inteira de coleta. A causa exata nao foi
# identificada, entao a protecao aqui e por limite de tempo: se estourar,
# verifica se os containers subiram mesmo assim e, se sim, segue em frente.
subir_ambiente() {
  local tentativa
  for tentativa in 1 2; do
    echo "  subindo o ambiente (tentativa ${tentativa}) ..."
    set +e
    timeout "${UP_TIMEOUT:-420}" docker compose -f "$COMPOSE" up -d --build \
      >>"${RUN_DIR}/compose.log" 2>&1
    local status=$?
    set -e
    if [[ $status -eq 124 ]]; then
      echo "  o comando de subida estourou o tempo limite; verificando o estado real ..."
    elif [[ $status -ne 0 ]]; then
      echo "  o comando de subida falhou com status ${status}; verificando o estado real ..."
    fi

    # O que importa nao e o status do compose, e a aplicacao responder.
    if wait_ready; then
      return 0
    fi

    echo "  ambiente nao respondeu; derrubando e tentando de novo ..."
    docker compose -f "$COMPOSE" down --remove-orphans >>"${RUN_DIR}/compose.log" 2>&1 || true
    sleep 10
  done
  return 1
}

for rep in $(seq "$START_REP" "$REPS"); do
  RUN_DIR="${OUTDIR}/run-${rep}"
  mkdir -p "${RUN_DIR}/metrics"

  echo ""
  echo "=============================================================="
  echo " Estagio ${STAGE} — repeticao ${rep} de ${REPS}"
  echo "=============================================================="

  # Ambiente limpo a cada repeticao: o volume do PostgreSQL e recriado para que
  # todas as execucoes partam do mesmo volume de dados. O volume do Prometheus e
  # preservado, pois guarda o historico de metricas ja exportado.
  docker compose -f "$COMPOSE" down --remove-orphans >/dev/null 2>&1 || true
  docker volume ls -q --filter "name=${COMPOSE_PROJECT_NAME}_postgres_data" \
    | xargs -r docker volume rm >/dev/null 2>&1 || true

  if ! subir_ambiente; then
    echo "  FALHA: ambiente nao subiu na repeticao ${rep}; seguindo para a proxima" >&2
    echo "$(date -Is) estagio ${STAGE} repeticao ${rep}: ambiente nao subiu" >> "${OUTDIR}/falhas.log"
    docker compose -f "$COMPOSE" down --remove-orphans >/dev/null 2>&1 || true
    continue
  fi

  # Margem para o Prometheus registrar ao menos um scrape antes da carga.
  sleep 15

  # amostrador de CPU e memoria por container, em paralelo ao teste de carga
  python3 scripts/sample_container_stats.py \
    --outfile "${RUN_DIR}/metrics/container_stats.csv" \
    --interval 5 --project "$COMPOSE_PROJECT_NAME" &
  SAMPLER_PID=$!

  START=$(date +%s)
  set +e
  k6 run ${K6_EXTRA} \
    --summary-export "${RUN_DIR}/summary.json" \
    --env BASE_URL="${BASE_URL}" \
    "load-tests/k6/${SCRIPT}" | tee "${RUN_DIR}/k6-console.log"
  K6_STATUS=${PIPESTATUS[0]}
  set -e
  END=$(date +%s)

  kill -TERM "$SAMPLER_PID" 2>/dev/null || true
  wait "$SAMPLER_PID" 2>/dev/null || true

  echo "  k6 encerrou com status ${K6_STATUS} (status != 0 indica threshold nao atingido, nao falha de execucao)"
  echo "  exportando metricas de recursos do Prometheus ..."

  # Margem para o ultimo scrape entrar na janela consultada.
  sleep 15

  python3 scripts/collect_metrics.py \
    --stage "$STAGE" \
    --run "$rep" \
    --start "$START" \
    --end "$((END + 15))" \
    --outdir "${RUN_DIR}/metrics" \
    --prometheus "$PROMETHEUS" \
    --container-stats "${RUN_DIR}/metrics/container_stats.csv" \
    ${COLLECT_EXTRA}

  printf '{"stage":"%s","run":%s,"start":%s,"end":%s,"k6_exit":%s}\n' \
    "$STAGE" "$rep" "$START" "$END" "$K6_STATUS" > "${RUN_DIR}/run-info.json"

  docker compose -f "$COMPOSE" down --remove-orphans >/dev/null 2>&1 || true
  echo "  resultados em ${RUN_DIR}"
done

echo ""
echo "Estagio ${STAGE} concluido. Consolide com: python3 scripts/consolidate.py"
