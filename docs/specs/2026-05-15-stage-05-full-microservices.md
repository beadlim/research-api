# Spec: Stage 05 — Full Microservices com Schema-per-Service

**Quality Score:** 4.7/5.0 (Completeness: 5.0 | Clarity: 4.5 | Consistency: 4.5 | Feasibility: 5.0 | Testability: 4.5)

---

## Problem Statement

Stage 05 completes the Strangler Pattern migration do research-api TCC. O monolito já foi removido no Stage 04. A única variável nova é **schema isolation**: cada um dos 4 microsserviços (users, products, orders, inventory) passa a ter um schema PostgreSQL dedicado no servidor compartilhado. Isso isola o efeito de schema-per-service na performance medida (P95, throughput, error rate) para a tabela comparativa do TCC.

## Constraints

- Stack idêntica a todos os stages anteriores: Go 1.25 + chi + pgx v5 + NGINX 1.25 + k6 + Docker Compose
- PostgreSQL 16 único servidor, schemas separados (não DB-per-service) — mantém variável de infra constante
- Nenhuma mudança em lógica de negócio dos serviços — apenas configuração de conexão DB e SQL de migration
- Perfil de carga k6 idêntico aos stages 01–04 (50→200→500 VUs, mesmo mix de cenários, mesmos thresholds)
- `results/stage-05/summary.json` com mesmo schema JSON dos stages anteriores
- Nomes de schema e credenciais DB via env vars, não hardcoded

## Non-Functional Requirements

- P95 < 500ms sob carga de 500 VUs
- Taxa de erro < 1%
- `pool_max_conns=10` por serviço (40 total / 100 max_connections do PG)
- 4 serviços iniciam healthy antes do NGINX rotear tráfego
- Health endpoints retornam 503 se DB indisponível (`pool.Ping()` check)

## Architecture Decisions

1. **Schemas:** `users_schema`, `products_schema`, `orders_schema`, `inventory_schema`
2. **DATABASE_URL por serviço:** `postgres://postgres:postgres@postgres:5432/research?sslmode=disable&search_path=<schema>&pool_max_conns=10`
3. **search_path via DSN** — pgx v5 aplica no startup da conexão; zero mudança em lógica de query Go
4. **Migration SQL** — cada `001_init.sql` recebe `CREATE SCHEMA IF NOT EXISTS <nome>; SET search_path = <nome>;` no início
5. **products-service migration: remover `inventory`** — a migration atual (do stage 03) cria `products` + `inventory`. Stage 05 move `inventory` exclusivamente para `inventory_schema`
6. **FKs:** `orders_schema` mantém FK intra-schema (`order_items.order_id → orders.id`); nenhum FK cross-schema
7. **NGINX keepalive** — `keepalive 32` + `proxy_http_version 1.1` + `proxy_set_header Connection ""` (os 3 são obrigatórios juntos; silenciosamente inativo sem eles)
8. **Health check com `pool.Ping()`** — retorna 503 se DB down; única mudança de comportamento Go
9. **k6 setup()** — adicionar seeding de inventory via `PATCH /inventory/:id` por produto criado
10. **docker-compose.05:** volumes `postgres_data_05`/`grafana_data_05`, DATABASE_URLs com schema, build de `stages/05-full-microservices/`

## Acceptance Criteria

- **AC1:** `docker compose ps` mostra 4 app services + nginx + postgres + prometheus + grafana + cadvisor; sem container monolito
- **AC2:** `\dt <schema>.*` por schema: `users_schema` → só `users`; `products_schema` → só `products`; `orders_schema` → só `orders,order_items`; `inventory_schema` → só `inventory`; `public` → sem tabelas de app
- **AC3:** `SELECT current_schema()` em cada DSN retorna o schema correto
- **AC4:** Smoke test completo passa: POST user → POST product → PATCH inventory → POST order → GET inventory (todos 2xx)
- **AC5:** k6 termina com P95 < 500ms, erro < 1%, `get inventory 200` > 99%, `inventory has product_id` > 99%
- **AC6:** `results/stage-05/summary.json` existe com métricas k6 válidas

## Edge Cases

- Postgres inicia depois dos containers de serviço → pgxpool retry automático, sem crash-loop
- products-service migration inclui `inventory` acidentalmente → check A3 falha com erro explícito
- `search_path` não aplicado → tabelas vão para `public`, check A1 captura
- NGINX keepalive mal configurado → silencioso; verificado por grep B1/B2/B3
- k6 inventory não seeded → `GET /inventory/:id` retorna 404, checks E4/E5 falham visivelmente

## Impact Analysis

**Criar:**
- `stages/05-full-microservices/{users,products,orders,inventory}-service/` (Go sources + Dockerfile + migration)
- `docker-compose.05-full-microservices.yml`
- `gateway/nginx/stage-05.conf`
- `load-tests/k6/stage-05-full-microservices.js`
- `observability/prometheus/prometheus-stage05.yml`
- `results/stage-05/` (diretório)
- `docs/specs/` (este arquivo)

**Modificar:**
- `go.work` — 4 novos module paths
- `RESEARCH_PLAN.md` — resultados pós-teste

**Mudanças Go:** health check handler em `main.go` de cada serviço (`pool.Ping()` → 503 se down)

**Não modificar:** nenhum artefato dos stages 01–04

## Risks

1. products-service migration cria `inventory` em `products_schema` (falha de isolamento silenciosa)
2. `search_path` não aplicado → tabelas em `public` (silencioso, capturado por probe SQL)
3. NGINX keepalive mal configurado (silencioso, capturado por grep de config)
4. k6 inventory não seeded → `GET /inventory` retorna 404 infla error rate
5. FK cross-schema em migration de orders (acoplamento de schema)
6. Health check retorna 200 com DB down (race condition na inicialização)

## Test Strategy

- **Unit:** `TestMigration_*` por serviço (verifica ownership de tabela por schema)
- **Unit:** `TestHealth_DBDown` por serviço (verifica 503 quando pool indisponível)
- **Integration:** probes SQL de isolamento de schema (Bloco A do rubric)
- **Integration:** grep de config NGINX (Bloco B)
- **Smoke:** round-trip CRUD completo com seeding de inventory (Bloco D)
- **Load:** k6 com script stage-05 (Bloco E)
- **Observabilidade:** targets Prometheus UP (Bloco F)

## Implementation Steps

1. Criar estrutura de diretórios `stages/05-full-microservices/`
2. Copiar fontes Go do stage 04; atualizar `go.mod` com module paths stage-05
3. Adicionar `pool.Ping()` nos handlers `/health` de cada `main.go`
4. Escrever `001_init.sql` por serviço (`CREATE SCHEMA` + tabelas; products omite inventory)
5. Copiar Dockerfiles do stage 04
6. Criar `docker-compose.05-full-microservices.yml` com DATABASE_URLs por schema
7. Criar `gateway/nginx/stage-05.conf` (keepalive 32 + proxy_http_version 1.1)
8. Atualizar `go.work` com 4 novos module paths
9. Criar `load-tests/k6/stage-05-full-microservices.js` (+ inventory seeding no setup)
10. Copiar `observability/prometheus/prometheus-stage05.yml`
11. `docker compose up --build -d` → probes de isolamento SQL (A1–A10) + smoke tests (D1–D7)
12. `k6 run` → salvar `results/stage-05/summary.json`
13. Atualizar `RESEARCH_PLAN.md` com resultados

## Verification Rubric

### Bloco A — Isolamento de Schema (pré-load)
- A1: Nenhuma tabela de app em `public`
- A2: `users_schema` contém só `users`
- A3: `products_schema` contém só `products`
- A4: `orders_schema` contém só `orders`, `order_items`
- A5: `inventory_schema` contém só `inventory`
- A6–A9: `SELECT current_schema()` retorna schema correto por serviço
- A10: Nenhum SQL cross-schema no código (`grep -r 'schema\.' stages/05-full-microservices/`)

### Bloco B — Config NGINX (pré-load)
- B1: `proxy_http_version 1.1` presente em 4 blocos location
- B2: `keepalive 32` presente em 4 blocos upstream
- B3: `proxy_set_header Connection ""` presente em 4 blocos location
- B4: Nenhuma referência a `monolith` na config

### Bloco C — Health Checks e Startup
- C1–C4: `/health` de cada serviço retorna 200
- C5: 4 serviços em estado `running` no docker compose ps
- C6: Sem restart loops nos logs

### Bloco D — Smoke Tests (pré-load)
- D1: POST /users → 201
- D2: GET /users/:id → 200
- D3: POST /products → 201
- D4: PATCH /inventory/:id → 200
- D5: GET /inventory/:id → 200 com dados
- D6: POST /orders → 201
- D7: GET /orders/:id → 200 com `total > 0` e `items` não vazio

### Bloco E — Load Test (k6)
- E1: k6 exit code 0
- E2: P95 < 500ms
- E3: Error rate < 1%
- E4: `get inventory 200` > 99%
- E5: `inventory has product_id` > 99%
- E6: Total requests dentro de ±20% do stage 04 (1.652.836)
- E7: Throughput registrado no RESEARCH_PLAN.md

### Bloco F — Observabilidade
- F1: 4 targets UP no Prometheus
- F2: `prometheus-stage05.yml` presente
- F3: cAdvisor mostra 4 containers de serviço
- F4: `go_goroutines` metric presente por serviço
- F5: `results/stage-05/summary.json` salvo com conteúdo válido

## Definition of Done

- [ ] Nenhum container monolito presente
- [ ] 4 schemas com tabelas corretas (A1–A5 passam)
- [ ] `SELECT current_schema()` retorna schema certo por serviço (A6–A9)
- [ ] Smoke test completo: POST user → POST product → PATCH inventory → POST order → GET inventory (D1–D7)
- [ ] k6: P95 < 500ms, erro < 1%, `get inventory 200` > 99% (E1–E5)
- [ ] `results/stage-05/summary.json` salvo (F5)
- [ ] RESEARCH_PLAN.md atualizado com resultados do stage 05
