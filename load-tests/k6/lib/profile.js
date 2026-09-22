import exec from 'k6/execution';

// Perfil de carga compartilhado pelos cinco estagios da pesquisa.
// A forma da carga e identica a das execucoes originais (30s de rampa + 5min de
// patamar, em 50/200/500 VUs). O que muda e a instrumentacao: cada requisicao
// passa a ser rotulada por endpoint e por patamar de carga, e o sumario passa a
// incluir o percentil 99.

export const ENDPOINTS = [
  'get_user',
  'list_products',
  'get_product',
  'post_order',
  'list_orders',
  'get_inventory',
];

export const LOAD_LEVELS = ['low', 'medium', 'high'];

// Janelas de tempo, em segundos, de cada fase do teste.
//   0-30    rampa   0 -> 50
//   30-330  patamar baixo   (50 VUs)
//   330-360 rampa  50 -> 200
//   360-660 patamar medio   (200 VUs)
//   660-690 rampa 200 -> 500
//   690-990 patamar alto    (500 VUs)
//   990-1020 rampa de descida
const WINDOWS = [
  { level: 'low', from: 30, to: 330 },
  { level: 'medium', from: 360, to: 660 },
  { level: 'high', from: 690, to: 990 },
];

// Classifica a requisicao no patamar de carga vigente. As requisicoes emitidas
// durante as rampas recebem o rotulo "ramp" e sao descartadas na analise, de
// modo que as estatisticas por patamar reflitam apenas o regime estavel.
export function loadLevel() {
  const t = exec.instance.currentTestRunDuration / 1000;
  for (const w of WINDOWS) {
    if (t >= w.from && t < w.to) return w.level;
  }
  return 'ramp';
}

// Monta o objeto de parametros da requisicao com os rotulos de analise.
export function tagged(endpoint, extra) {
  const params = Object.assign({}, extra);
  params.tags = Object.assign({}, params.tags, {
    endpoint: endpoint,
    load_level: loadLevel(),
  });
  return params;
}

// O k6 so materializa submetricas por rotulo no sumario quando existe um
// threshold declarado para aquele rotulo. Os limites abaixo sao propositalmente
// permissivos: servem para expor as submetricas, nao para reprovar a execucao.
function observationalThresholds() {
  const t = {};
  const OBSERVE = ['p(95)<600000'];

  for (const level of LOAD_LEVELS) {
    t[`http_req_duration{load_level:${level}}`] = OBSERVE;
    t[`http_req_failed{load_level:${level}}`] = ['rate<1'];
  }
  for (const ep of ENDPOINTS) {
    t[`http_req_duration{endpoint:${ep}}`] = OBSERVE;
    t[`http_req_failed{endpoint:${ep}}`] = ['rate<1'];
    for (const level of LOAD_LEVELS) {
      t[`http_req_duration{endpoint:${ep},load_level:${level}}`] = OBSERVE;
    }
  }
  return t;
}

export const options = {
  stages: [
    { duration: '30s', target: 50 },
    { duration: '5m', target: 50 },
    { duration: '30s', target: 200 },
    { duration: '5m', target: 200 },
    { duration: '30s', target: 500 },
    { duration: '5m', target: 500 },
    { duration: '30s', target: 0 },
  ],
  // Percentil 99 incluido no sumario, conforme previsto na metodologia.
  summaryTrendStats: ['min', 'avg', 'med', 'p(90)', 'p(95)', 'p(99)', 'max'],
  thresholds: Object.assign(
    {
      // Limiares de qualidade da pesquisa.
      'http_req_duration{scenario:default}': ['p(95)<500'],
      http_req_failed: ['rate<0.01'],
    },
    observationalThresholds(),
  ),
};
