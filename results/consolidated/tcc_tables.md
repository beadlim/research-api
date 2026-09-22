# Tabelas consolidadas

## Desempenho geral por estagio (media das repeticoes)

| Estagio | Execucoes | Vazao (req/s) | P50 (ms) | P95 (ms) | P99 (ms) | Erro (%) |
|---|---|---|---|---|---|---|
| Stage 01 | 3 | 2375.37 ± 0.95 | 1.12 ± 0.02 | 4.6 ± 0.14 | 7.9 ± 0.31 | 0 |
| Stage 01b | 3 | 1534.6 ± 10.12 | 6.21 ± 0.51 | 248.81 ± 1.9 | 307.97 ± 2.39 | 0 |
| Stage 02 | 3 | 1771.97 ± 3.13 | 4.99 ± 0.56 | 185.67 ± 2.04 | 240.67 ± 5.5 | 0 |
| Stage 03 | 3 | 2160.17 ± 2.85 | 5.94 ± 0.08 | 39.3 ± 0.47 | 70.04 ± 0.63 | 0 |
| Stage 04 | 3 | 1836.2 ± 7.25 | 7.22 ± 0.51 | 142.16 ± 2.77 | 396.03 ± 26.82 | 0.64 |
| Stage 05 | 3 | 1940.0 ± 14.73 | 4.54 ± 0.17 | 100.41 ± 8.9 | 417.45 ± 5.07 | 1.3 |
| Stage 05b | 3 | 1829.37 ± 19.42 | 7.37 ± 0.11 | 148.54 ± 18.52 | 385.03 ± 9.43 | 0.59 |

## P95 por patamar de carga (ms)

| Estagio | 50 VUs | 200 VUs | 500 VUs |
|---|---|---|---|
| Stage 01 | 6.12 ± 0.03 | 5.7 ± 0.2 | 4.1 ± 0.11 |
| Stage 01b | 7.67 ± 0.06 | 39.9 ± 22.76 | 279.34 ± 2.29 |
| Stage 02 | 7.57 ± 0.09 | 3.77 ± 0.07 | 210.51 ± 2.7 |
| Stage 03 | 7.61 ± 0.01 | 4.05 ± 0.15 | 48.82 ± 0.53 |
| Stage 04 | 8.86 ± 0.36 | 11.24 ± 1.13 | 211.04 ± 6.18 |
| Stage 05 | 8.49 ± 0.12 | 7.65 ± 0.15 | 166.11 ± 9.71 |
| Stage 05b | 8.99 ± 0.05 | 11.32 ± 0.29 | 218.22 ± 25.47 |

## P95 do POST /orders por estagio (ms)

| Estagio | 50 VUs | 200 VUs | 500 VUs |
|---|---|---|---|
| Stage 01 | 9.12 ± 0.08 | 9.17 ± 0.3 | 6.61 ± 0.34 |
| Stage 01b | 11.24 ± 0.1 | 43.89 ± 22.71 | 285.08 ± 2.2 |
| Stage 02 | 11.35 ± 0.25 | 5.69 ± 0.11 | 218.25 ± 2.82 |
| Stage 03 | 11.22 ± 0.22 | 6.27 ± 0.49 | 60.61 ± 0.86 |
| Stage 04 | 14.43 ± 0.17 | 21.93 ± 2.27 | 586.31 ± 38.72 |
| Stage 05 | 13.39 ± 0.23 | 13.42 ± 0.42 | 640.45 ± 8.1 |
| Stage 05b | 15.06 ± 0.12 | 21.12 ± 0.45 | 570.37 ± 36.23 |

## CPU por servico sob carga alta (% de um nucleo)

| Estagio | Servico | Media | P95 | Maximo |
|---|---|---|---|---|
| Stage 01 | api | 174.1 | 183.7 | 185.64 |
| Stage 01 | cadvisor | 0.21 | 0.7 | 1.04 |
| Stage 01 | grafana | 0.05 | 0.08 | 0.09 |
| Stage 01 | postgres | 146.41 | 152.16 | 155.32 |
| Stage 01 | prometheus | 0.18 | 0.36 | 0.47 |
| Stage 01b | api | 147.74 | 234.83 | 257.22 |
| Stage 01b | cadvisor | 0.12 | 0.75 | 2.02 |
| Stage 01b | grafana | 0.1 | 0.24 | 0.41 |
| Stage 01b | nginx | 591.17 | 770.65 | 795.85 |
| Stage 01b | postgres | 88.42 | 146.24 | 159.4 |
| Stage 01b | prometheus | 0.05 | 0.13 | 0.77 |
| Stage 02 | cadvisor | 0.29 | 1.25 | 1.95 |
| Stage 02 | grafana | 0.1 | 0.1 | 1.79 |
| Stage 02 | monolith-partial | 151.56 | 214.69 | 221.32 |
| Stage 02 | nginx | 443.36 | 689.14 | 708.44 |
| Stage 02 | postgres | 117.88 | 166.53 | 177.12 |
| Stage 02 | prometheus | 0.21 | 0.5 | 1.03 |
| Stage 02 | users-service | 51.8 | 77.92 | 84.08 |
| Stage 03 | cadvisor | 0.66 | 1.89 | 3.11 |
| Stage 03 | grafana | 0.11 | 0.15 | 1.14 |
| Stage 03 | monolith-partial | 141.17 | 152.74 | 160.26 |
| Stage 03 | nginx | 167.39 | 176.65 | 178.62 |
| Stage 03 | postgres | 172.15 | 192.83 | 215.12 |
| Stage 03 | products-service | 91.38 | 98.75 | 101.71 |
| Stage 03 | prometheus | 0.32 | 0.8 | 1.24 |
| Stage 03 | users-service | 72.18 | 78.22 | 80.55 |
| Stage 04 | cadvisor | 0.13 | 0.85 | 2.57 |
| Stage 04 | grafana | 0.27 | 0.99 | 2.22 |
| Stage 04 | inventory-service | 23.83 | 29.92 | 30.8 |
| Stage 04 | nginx | 128.42 | 160.66 | 166.34 |
| Stage 04 | orders-service | 386.88 | 679.45 | 725.33 |
| Stage 04 | postgres | 120.86 | 153.07 | 168.22 |
| Stage 04 | products-service | 85.76 | 105.38 | 110.9 |
| Stage 04 | prometheus | 0.1 | 0.37 | 1.11 |
| Stage 04 | users-service | 73.97 | 90.85 | 96.92 |
| Stage 05 | cadvisor | 0.17 | 1.14 | 3.36 |
| Stage 05 | grafana | 0.24 | 0.87 | 2.51 |
| Stage 05 | inventory-service | 17.87 | 23.7 | 24.91 |
| Stage 05 | nginx | 76.81 | 105.51 | 109.65 |
| Stage 05 | orders-service | 460.99 | 810.04 | 897.48 |
| Stage 05 | postgres | 122.23 | 169.45 | 182.99 |
| Stage 05 | products-service | 72.03 | 95.75 | 101.21 |
| Stage 05 | prometheus | 0.12 | 0.46 | 1.32 |
| Stage 05 | users-service | 61.5 | 85.9 | 90.65 |
| Stage 05b | cadvisor | 0.61 | 1.9 | 3.64 |
| Stage 05b | grafana | 0.3 | 1.18 | 2.38 |
| Stage 05b | inventory-service | 23.68 | 29.56 | 30.82 |
| Stage 05b | nginx | 129.17 | 161.16 | 166.96 |
| Stage 05b | orders-service | 373.87 | 666.47 | 734.85 |
| Stage 05b | postgres | 119.62 | 152.06 | 158.13 |
| Stage 05b | products-service | 86.15 | 107.3 | 111.18 |
| Stage 05b | prometheus | 0.28 | 0.58 | 1.93 |
| Stage 05b | users-service | 73.91 | 90.83 | 96.46 |

## Memoria por servico sob carga alta (MB)

| Estagio | Servico | Media | P95 | Maximo |
|---|---|---|---|---|
| Stage 01 | api | 36.12 | 36.85 | 37.12 |
| Stage 01 | cadvisor | 20.0 | 20.9 | 21.34 |
| Stage 01 | grafana | 43.79 | 43.81 | 44.03 |
| Stage 01 | postgres | 111.82 | 132.62 | 134.73 |
| Stage 01 | prometheus | 38.06 | 39.07 | 39.31 |
| Stage 01b | api | 26.76 | 28.83 | 30.34 |
| Stage 01b | cadvisor | 19.87 | 21.91 | 22.04 |
| Stage 01b | grafana | 43.91 | 44.04 | 44.36 |
| Stage 01b | nginx | 24.23 | 26.12 | 26.77 |
| Stage 01b | postgres | 97.38 | 107.35 | 108.33 |
| Stage 01b | prometheus | 36.51 | 37.41 | 37.84 |
| Stage 02 | cadvisor | 19.87 | 21.57 | 21.71 |
| Stage 02 | grafana | 44.75 | 44.76 | 45.05 |
| Stage 02 | monolith-partial | 24.17 | 25.87 | 27.8 |
| Stage 02 | nginx | 22.58 | 24.49 | 25.29 |
| Stage 02 | postgres | 124.92 | 137.42 | 138.8 |
| Stage 02 | prometheus | 36.27 | 37.27 | 37.37 |
| Stage 02 | users-service | 16.57 | 17.92 | 18.27 |
| Stage 03 | cadvisor | 20.11 | 21.72 | 21.9 |
| Stage 03 | grafana | 45.47 | 45.48 | 45.69 |
| Stage 03 | monolith-partial | 21.55 | 22.57 | 23.16 |
| Stage 03 | nginx | 21.64 | 22.53 | 23.04 |
| Stage 03 | postgres | 155.5 | 173.41 | 176.03 |
| Stage 03 | products-service | 18.21 | 19.14 | 19.62 |
| Stage 03 | prometheus | 36.62 | 37.39 | 37.47 |
| Stage 03 | users-service | 15.35 | 16.24 | 16.69 |
| Stage 04 | cadvisor | 20.81 | 22.47 | 22.65 |
| Stage 04 | grafana | 67.56 | 67.57 | 67.57 |
| Stage 04 | inventory-service | 13.05 | 13.98 | 14.91 |
| Stage 04 | nginx | 24.13 | 26.7 | 27.55 |
| Stage 04 | orders-service | 41.28 | 48.89 | 54.81 |
| Stage 04 | postgres | 175.8 | 187.86 | 189.13 |
| Stage 04 | products-service | 18.48 | 19.42 | 20.51 |
| Stage 04 | prometheus | 48.69 | 49.73 | 49.81 |
| Stage 04 | users-service | 16.74 | 18.14 | 19.02 |
| Stage 05 | cadvisor | 20.38 | 22.2 | 22.72 |
| Stage 05 | grafana | 43.39 | 43.41 | 43.63 |
| Stage 05 | inventory-service | 14.32 | 15.4 | 16.0 |
| Stage 05 | nginx | 21.35 | 23.29 | 24.15 |
| Stage 05 | orders-service | 50.0 | 56.28 | 58.8 |
| Stage 05 | postgres | 141.11 | 153.06 | 154.53 |
| Stage 05 | products-service | 20.33 | 21.65 | 22.05 |
| Stage 05 | prometheus | 39.8 | 40.86 | 41.04 |
| Stage 05 | users-service | 19.72 | 21.64 | 23.43 |
| Stage 05b | cadvisor | 20.92 | 22.63 | 22.75 |
| Stage 05b | grafana | 43.86 | 43.87 | 44.12 |
| Stage 05b | inventory-service | 13.27 | 14.37 | 14.88 |
| Stage 05b | nginx | 23.44 | 26.07 | 26.71 |
| Stage 05b | orders-service | 41.75 | 50.36 | 54.48 |
| Stage 05b | postgres | 170.97 | 183.4 | 184.93 |
| Stage 05b | products-service | 18.74 | 19.88 | 20.96 |
| Stage 05b | prometheus | 38.1 | 38.79 | 38.81 |
| Stage 05b | users-service | 16.59 | 17.83 | 18.67 |

## Espera media por conexao no pool sob carga alta (ms)

| Estagio | Servico | Media | P95 | Maximo |
|---|---|---|---|---|
| Stage 01 | monolith | 0.07 | 0.1 | 0.13 |
| Stage 01b | monolith | 3.12 | 3.87 | 3.98 |
| Stage 02 | monolith-partial | 2.44 | 3.09 | 3.17 |
| Stage 02 | users-service | 0.03 | 0.05 | 0.05 |
| Stage 03 | monolith-partial-03 | 1.97 | 2.43 | 2.5 |
| Stage 03 | products-service | 0.05 | 0.07 | 0.07 |
| Stage 03 | users-service | 0.03 | 0.04 | 0.04 |
| Stage 04 | inventory-service | 0.01 | 0.02 | 0.02 |
| Stage 04 | orders-service | 51.39 | 68.62 | 70.3 |
| Stage 04 | products-service | 0.16 | 0.21 | 0.23 |
| Stage 04 | users-service | 0.2 | 0.29 | 0.33 |
| Stage 05 | inventory-service | 0.01 | 0.02 | 0.02 |
| Stage 05 | orders-service | 38.97 | 52.0 | 54.49 |
| Stage 05 | products-service | 0.32 | 0.42 | 0.44 |
| Stage 05 | users-service | 0.39 | 0.55 | 0.58 |
| Stage 05b | inventory-service | 0.01 | 0.02 | 0.02 |
| Stage 05b | orders-service | 54.88 | 72.75 | 74.43 |
| Stage 05b | products-service | 0.15 | 0.2 | 0.22 |
| Stage 05b | users-service | 0.2 | 0.3 | 0.32 |
