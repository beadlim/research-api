# Pipeline de medição

Executa um estágio, coleta as métricas e exporta tudo em CSV. Substitui a coleta
manual usada nas primeiras execuções, na qual as métricas de CPU, memória e pool
de conexões se perdiam junto com o ambiente.

## Uso

```bash
# validação rápida do pipeline (2 min, resultados descartáveis)
SMOKE=1 scripts/run-stage.sh 01

# coleta noturna sem supervisão, sobrevive ao fechamento do terminal
nohup scripts/run-night.sh > /dev/null 2>&1 &

# acompanhar
tail -f results/coleta-*.log

# consolidar manualmente, se precisar
python3 scripts/consolidate.py
```

Cada repetição leva cerca de 17 minutos.

`run-night.sh` aceita a sintaxe `estagio:repeticao_inicial`, que retoma sem
sobrescrever o que já foi coletado. Sem argumentos, roda `04:2 05:1`, que é o
que falta. Para tudo do zero: `scripts/run-night.sh "01 01b 02 03 04 05" 3`.

O mesmo vale para uma execução avulsa: `START_REP=2 scripts/run-stage.sh 04 3`.

## Estágios

| Estágio | Arquitetura |
|---|---|
| `01` | Monolito completo (linha de base) |
| `01b` | Monolito íntegro atrás do NGINX (controle) |
| `02` | Users extraído |
| `03` | Products extraído |
| `04` | Orders e Inventory extraídos |
| `05` | Microsserviços completos, schema por serviço |

O estágio `01b` isola o custo do salto de rede do gateway do custo da extração de
serviço, que no `02` ocorrem ao mesmo tempo.

## O que é coletado

| Fonte | Métricas |
|---|---|
| k6 | Latência P50/P90/P95/**P99**, vazão e taxa de erro, por endpoint e por patamar de carga |
| `docker stats` | CPU e memória por container |
| Prometheus, `/metrics` da aplicação | CPU e memória do processo, goroutines, heap, pausa de GC |
| Prometheus, coletor `db_pool_*` | Conexões em uso e ociosas, espera por conexão, aquisições sem conexão livre |

As métricas de pool medem diretamente a contenção no banco, que nas análises
anteriores era inferida a partir da latência e não observada.

Apenas as janelas de regime estável entram nas estatísticas. As rampas recebem o
rótulo `ramp` e são descartadas, de modo que os três patamares (50, 200 e 500 VUs)
sejam comparáveis entre si.

## Saída

```
results/<estágio>/run-<n>/
  summary.json                        sumário do k6
  k6-console.log                      saída completa do k6
  run-info.json                       janela de tempo da execução
  metrics/container_stats.csv         série temporal de CPU e memória
  metrics/<métrica>.csv               séries do Prometheus
  metrics/resources_by_load_level.csv agregados por patamar

results/consolidated/
  latency_by_stage.csv                todas as execuções
  resources_by_stage.csv              todas as execuções
  tcc_tables.md                       tabelas prontas para conferência
```

## Observações

O ambiente é recriado a cada repetição e o volume do PostgreSQL é removido, para
que todas as execuções partam do mesmo volume de dados. Sem isso o `setup()` do
k6 falha por e-mail duplicado e o teste roda com identificadores indefinidos; os
scripts agora interrompem a execução nesse caso em vez de produzir um resultado
silenciosamente inválido.

O volume do Prometheus é preservado entre execuções.

Se a porta do Grafana estiver ocupada, o script escolhe outra automaticamente.
Para fixar: `GRAFANA_PORT=3005 scripts/run-stage.sh 01`.

O `docker compose up` já travou indefinidamente com todos os containers no ar e
saudáveis, consumindo uma noite inteira de coleta. A causa não foi identificada.
A proteção é por limite de tempo (`UP_TIMEOUT`, padrão 420 s): se estourar, o
script verifica se a aplicação responde mesmo assim e segue; se não responder,
derruba e tenta mais uma vez; se ainda assim falhar, registra em
`results/<estágio>/falhas.log` e passa para a repetição seguinte, em vez de
travar a noite inteira. As imagens passaram a ser construídas uma única vez,
fora do laço.

A medição é sensível a uso concorrente da máquina, sobretudo nos percentis sob
500 usuários virtuais. Rode com a máquina ociosa e compare as repetições: nos
estágios coletados com a máquina parada a dispersão ficou abaixo de 0,2%.

O cAdvisor continua no ambiente, mas em hosts com Docker recente e cgroups v2 ele
costuma enumerar apenas o cgroup raiz. Por isso CPU e memória por container vêm de
`docker stats`, e não dele.
