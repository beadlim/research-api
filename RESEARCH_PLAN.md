# Plano de Pesquisa — Migração Monolito → Microsserviços

**Repositório:** https://github.com/beadlim/research-api  
**Objetivo:** Medir impactos de desempenho da migração incremental via Strangler Pattern  
**Metodologia:** Estudo de caso único, abordagem quantitativa, coleta em 4 estágios

---

## Stack Técnico

| Camada | Tecnologia |
|---|---|
| Linguagem | Go 1.25 |
| Roteamento | chi v5 |
| Banco de dados | PostgreSQL 16 |
| Gateway (Strangler) | NGINX 1.25 |
| Métricas | Prometheus + Grafana |
| Tracing inter-serviços | OpenTelemetry + Jaeger (a partir do stage 03) |
| Testes de carga | k6 v1.7 |
| Containers | Docker + Docker Compose |

---

## Domínio da API — Orders System

4 módulos extraídos incrementalmente:

| Módulo | Endpoints | Estágio de extração |
|---|---|---|
| Users | `POST /users` `GET /users` `GET /users/:id` | Stage 02 ✅ |
| Products | `POST /products` `GET /products` `GET /products/:id` | Stage 03 ✅ |
| Orders | `POST /orders` `GET /orders` `GET /orders/:id` | Stage 04 ✅ |
| Inventory | `GET /inventory/:id` `PATCH /inventory/:id` | Stage 04 ✅ |

---

## Perfil de Carga k6 (idêntico em todos os estágios)

```
30s  ramp-up  →  50 VUs
5min steady      50 VUs   (baixa carga)
30s  ramp-up  → 200 VUs
5min steady     200 VUs   (média carga)
30s  ramp-up  → 500 VUs
5min steady     500 VUs   (alta carga)
30s  ramp-down →   0 VUs
```

Thresholds: P95 < 500ms | taxa de erro < 1%

Mix de cenários:
- 30% GET /users/:id
- 20% LIST /products
- 15% GET /products/:id
- 15% POST /orders  ← operação mais custosa (transação multi-query)
- 10% LIST /orders
- 10% GET /inventory/:id

---

## Métricas Coletadas por Estágio

- Latência: P50, P90, P95, P99, máximo, média
- Throughput: req/s
- Taxa de erro: % por código HTTP
- Recursos: CPU%, memória MB por container (cAdvisor)
- Go runtime: goroutines, heap alloc, GC duration
- Latência inter-serviços: via Jaeger/OTEL (estágios 03+)

---

## Progresso dos Estágios

### ✅ Stage 01 — Monolito Completo (baseline)

**Arquivo:** `docker-compose.01-monolith.yml`  
**k6:** `load-tests/k6/baseline.js`  
**Resultados:** `results/baseline/summary.json`

| Métrica | Resultado |
|---|---|
| Throughput | 2.327 req/s |
| P50 | 1,98 ms |
| P90 | 6,92 ms |
| P95 | 10,12 ms |
| Máximo | 127,98 ms |
| Taxa de erro | 0% |
| Total requisições | 2.374.271 |

**Como rodar:**
```bash
docker compose -f docker-compose.01-monolith.yml up --build -d
k6 run --summary-export results/baseline/summary.json load-tests/k6/baseline.js
docker compose -f docker-compose.01-monolith.yml down
```

---

### ✅ Stage 02 — Users Service Extraído (Strangler Pattern)

**Arquitetura:**
- NGINX gateway na porta 8080
- `/users` → `users-service` (Go, porta 8081)
- todo o resto → `monolith-partial` (Go, porta 8080 interno)
- Banco compartilhado (PostgreSQL único)

**Arquivo:** `docker-compose.02-users-extracted.yml`  
**NGINX config:** `gateway/nginx/stage-02.conf`  
**k6:** `load-tests/k6/stage-02-users-extracted.js`  
**Resultados:** `results/stage-02/summary.json`

| Métrica | Resultado | vs Baseline |
|---|---|---|
| Throughput | 1.793 req/s | -23% |
| P50 | 9,43 ms | +376% |
| P90 | 102,05 ms | +1.375% |
| P95 | 160,89 ms | +1.491% |
| Máximo | 902,82 ms | +605% |
| Taxa de erro | 0% | = |
| Total requisições | 1.829.498 | -23% |

**Causa do overhead:** hop NGINX + dois connection pools competindo no PostgreSQL.

**Como rodar:**
```bash
docker compose -f docker-compose.02-users-extracted.yml up --build -d
k6 run --summary-export results/stage-02/summary.json load-tests/k6/stage-02-users-extracted.js
docker compose -f docker-compose.02-users-extracted.yml down
```

---

### ✅ Stage 03 — Products Service Extraído

**Arquitetura:**
- NGINX gateway na porta 8080
- `/users` → `users-service` (Go, porta 8081)
- `/products` → `products-service` (Go, porta 8082)
- todo o resto → `monolith-partial` (Go, porta 8080 interno)
- Banco compartilhado (PostgreSQL único)

**Arquivo:** `docker-compose.03-products-extracted.yml`  
**NGINX config:** `gateway/nginx/stage-03.conf`  
**k6:** `load-tests/k6/stage-03-products-extracted.js`  
**Resultados:** `results/stage-03/summary.json`

| Métrica | Resultado | vs Stage 02 | vs Baseline |
|---|---|---|---|
| Throughput | 1.964 req/s | +10% | -16% |
| P50 | 9,13 ms | -3% | +361% |
| P90 | 58,83 ms | -42% | +750% |
| P95 | 90,19 ms | -44% | +791% |
| Máximo | 424,60 ms | -53% | +232% |
| Taxa de erro | 0% | = | = |
| Total requisições | 2.003.676 | — | — |

**Observação:** extração do products-service *melhorou* P95 em relação ao stage-02 (90ms vs 161ms). Monolito menor reduz contenção no connection pool do PostgreSQL.

**Como rodar:**
```bash
docker compose -f docker-compose.03-products-extracted.yml up --build -d
k6 run --summary-export results/stage-03/summary.json load-tests/k6/stage-03-products-extracted.js
docker compose -f docker-compose.03-products-extracted.yml down
```

---

### ✅ Stage 04 — Orders + Inventory Extraídos

**Arquitetura:**
- NGINX gateway na porta 8080
- `/users` → `users-service` (Go, porta 8081)
- `/products` → `products-service` (Go, porta 8082)
- `/orders` → `orders-service` (Go, porta 8083)
- `/inventory` → `inventory-service` (Go, porta 8084)
- Banco compartilhado (PostgreSQL único)
- `POST /orders` realiza 2 chamadas HTTP inter-serviços: users-service (validação) + products-service (preço)

**Arquivo:** `docker-compose.04-orders-inventory.yml`  
**NGINX config:** `gateway/nginx/stage-04.conf`  
**k6:** `load-tests/k6/stage-04-orders-inventory.js`  
**Resultados:** `results/stage-04/summary.json`

| Métrica | Resultado | vs Stage 03 | vs Baseline |
|---|---|---|---|
| Throughput | 1.620 req/s | -18% | -30% |
| P50 | 11,42 ms | +25% | +477% |
| P90 | 163,34 ms | +178% | +2.261% |
| P95 | 286,41 ms | +218% | +2.731% |
| Máximo | 1.030,00 ms | +143% | +705% |
| Taxa de erro | 0% | = | = |
| Total requisições | 1.652.836 | — | — |

**Observação:** P95 sobe 218% em relação ao stage-03, confirmando o overhead de comunicação inter-serviços no `POST /orders` (2 chamadas HTTP síncronas adicionais por transação). Throughput cai 18% por maior tempo de processamento por requisição.

**Como rodar:**
```bash
docker compose -f docker-compose.04-orders-inventory.yml up --build -d
k6 run --summary-export results/stage-04/summary.json load-tests/k6/stage-04-orders-inventory.js
docker compose -f docker-compose.04-orders-inventory.yml down
```

---

### ⏳ Stage 05 — Microsserviços Completos

**O que fazer:**
1. Monolito completamente removido
2. Cada serviço com seu próprio schema no PostgreSQL (ou banco separado)
3. NGINX roteia 100% do tráfego para os microsserviços
4. Testar com e sem escalonamento horizontal
5. Salvar em `results/stage-05/summary.json`

---

## Estrutura do Repositório

```
research-api/
├── stages/
│   ├── 01-monolith/               ✅ monolito completo
│   ├── 02-users-extracted/        ✅ users-service + monolith-partial
│   │   ├── users-service/
│   │   └── monolith-partial/
│   ├── 03-products-extracted/     ✅ products-service + monolith-partial
│   ├── 04-orders-inventory/       ✅ orders-service + inventory-service
│   └── 05-full-microservices/     ⏳ a implementar
├── gateway/
│   └── nginx/
│       ├── stage-02.conf          ✅
│       ├── stage-03.conf          ✅
│       ├── stage-04.conf          ✅
│       └── stage-05.conf          ⏳
├── load-tests/k6/
│   ├── baseline.js                ✅
│   ├── stage-02-users-extracted.js ✅
│   ├── stage-03-products-extracted.js ✅
│   ├── stage-04-orders-inventory.js   ✅
│   └── stage-05-full-microservices.js ⏳
├── observability/
│   ├── prometheus/
│   │   ├── prometheus.yml         ✅ stage 01
│   │   ├── prometheus-stage02.yml ✅ stage 02
│   │   └── prometheus-stage0X.yml ⏳ stages seguintes
│   └── grafana/
│       └── provisioning/
│           └── dashboards/json/
│               └── monolith.json  ✅ (usar para todos os estágios)
├── results/
│   ├── baseline/summary.json      ✅
│   ├── stage-02/summary.json      ✅
│   ├── stage-03/summary.json      ✅
│   ├── stage-04/summary.json      ✅
│   └── stage-05/summary.json      ⏳
├── docker-compose.01-monolith.yml          ✅
├── docker-compose.02-users-extracted.yml   ✅
├── docker-compose.03-products-extracted.yml ✅
├── docker-compose.04-orders-inventory.yml   ✅
├── docker-compose.05-full-microservices.yml ⏳
└── go.work                        ✅ (adicionar módulos novos em cada estágio)
```

---

## Tabela Comparativa Final (preencher ao longo dos estágios)

| Métrica | Stage 01 Monolito | Stage 02 +Users | Stage 03 +Products | Stage 04 +Orders/Inv | Stage 05 Full µS |
|---|---|---|---|---|---|
| Throughput (req/s) | 2.327 | 1.793 | 1.964 | 1.620 | ⏳ |
| P50 (ms) | 1,98 | 9,43 | 9,13 | 11,42 | ⏳ |
| P90 (ms) | 6,92 | 102,05 | 58,83 | 163,34 | ⏳ |
| P95 (ms) | 10,12 | 160,89 | 90,19 | 286,41 | ⏳ |
| Máximo (ms) | 127,98 | 902,82 | 424,60 | 1.030,00 | ⏳ |
| Taxa de erro | 0% | 0% | 0% | 0% | ⏳ |
| Serviços ativos | 1 | 2 + nginx | 3 + nginx | 4 + nginx | ⏳ |

---

## Observações e Pendências

- [ ] **CPU/Memory no Grafana** — filtro cAdvisor corrigido no dashboard, validar no Stage 03
- [ ] **Jaeger/OTEL** — adicionar a partir do Stage 03 para medir latência inter-serviços
- [ ] **Banco separado por serviço** — a partir do Stage 05 (cada serviço com seu schema)
- [ ] **Screenshots Grafana** — capturar de cada estágio para usar como figuras no TCC
- [ ] **Análise estatística final** — estatística descritiva com todos os 5 estágios para a seção de Resultados

---

## Comandos Úteis

```bash
# Ver containers rodando
docker compose -f docker-compose.0X-NOME.yml ps

# Ver logs de um serviço
docker compose -f docker-compose.0X-NOME.yml logs -f users-service

# Verificar métricas Prometheus
curl http://localhost:8080/metrics | grep http_request

# Checar progresso do k6 em background
# (o ID do task é exibido ao rodar com run_in_background)

# Acessar Grafana
open http://localhost:3000  # admin/admin

# Acessar Prometheus
open http://localhost:9090
```
