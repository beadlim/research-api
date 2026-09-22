#!/usr/bin/env bash
# Coleta noturna sem supervisao. Grava tudo em log, nao aborta a noite inteira
# quando um estagio falha e consolida os resultados no fim.
#
# Uso:
#   scripts/run-night.sh                    # o que falta hoje: 04 a partir da
#                                           # repeticao 2, e 05 a partir da 1
#   scripts/run-night.sh "01 02 03" 3       # estagios e repeticoes especificos
#
# A sintaxe "estagio:repeticao_inicial" retoma sem sobrescrever o que ja existe.
# Ex.: "04:2" roda apenas as repeticoes 2 e 3 do estagio 04.
#
# Para sobreviver ao fechamento do terminal:
#   nohup scripts/run-night.sh > /dev/null 2>&1 &

set -uo pipefail

ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
cd "$ROOT"

ESTAGIOS="${1:-04:2 05:1}"
REPS="${2:-3}"
LOG="results/coleta-$(date +%Y%m%d-%H%M%S).log"
mkdir -p results

{
  echo "inicio: $(date -Is)"
  echo "estagios: ${ESTAGIOS} | repeticoes ate: ${REPS}"
  echo

  for spec in $ESTAGIOS; do
    estagio="${spec%%:*}"
    inicio="${spec#*:}"
    [[ "$inicio" == "$estagio" ]] && inicio=1

    echo "############ estagio ${estagio} (repeticoes ${inicio} a ${REPS}) ############"
    if START_REP="$inicio" bash scripts/run-stage.sh "$estagio" "$REPS"; then
      echo ">>> estagio ${estagio} concluido"
    else
      echo ">>> estagio ${estagio} terminou com erro (status $?); seguindo para o proximo"
    fi
    echo
  done

  echo "############ consolidacao ############"
  python3 scripts/consolidate.py

  echo
  echo "############ inventario ############"
  for d in results/baseline results/stage-01b results/stage-02 results/stage-03 \
           results/stage-04 results/stage-05 results/stage-05b; do
    n=$(ls -d "$d"/run-*/summary.json 2>/dev/null | wc -l)
    printf '%-16s %s repeticao(oes) com sumario\n' "$(basename "$d")" "$n"
  done

  echo
  echo "fim: $(date -Is)"
} 2>&1 | tee "$LOG"

echo
echo "log completo em ${LOG}"
