import http from 'k6/http';
import { check, sleep } from 'k6';
import { options as loadProfile, tagged } from './lib/profile.js';

// Stage 04: microsserviços completos — monolito removido
// POST /orders envolve 2 chamadas HTTP inter-serviços (users + products)
const BASE_URL = __ENV.BASE_URL || 'http://localhost:8080';
const HEADERS = { 'Content-Type': 'application/json' };
const RUN_ID = Date.now();

export const options = loadProfile;

export function setup() {
  const users = [];
  const products = [];

  for (let i = 0; i < 20; i++) {
    const res = http.post(`${BASE_URL}/users`,
      JSON.stringify({ name: `User ${i}`, email: `user${i}@k6-${RUN_ID}.com` }),
      { headers: HEADERS },
    );
    if (res.status === 201) users.push(res.json('id'));
  }

  for (let i = 0; i < 20; i++) {
    const res = http.post(`${BASE_URL}/products`,
      JSON.stringify({ name: `Product ${i}`, price: parseFloat(((i + 1) * 9.99).toFixed(2)) }),
      { headers: HEADERS },
    );
    if (res.status === 201) products.push(res.json('id'));
  }
  // Sem este guarda, um banco nao reiniciado faz o setup falhar em silencio
  // (e-mails duplicados), o teste roda com identificadores indefinidos e toda a
  // execucao e descartavel sem que isso apareca no sumario.
  if (users.length === 0 || products.length === 0) {
    throw new Error(
      `setup falhou: ${users.length} usuarios e ${products.length} produtos criados. ` +
        'Reinicie o banco antes do teste (scripts/run-stage.sh ja faz isso).',
    );
  }

  return { users, products };
}

export default function ({ users, products }) {
  const r = Math.random();

  if (r < 0.30) {
    const id = users[Math.floor(Math.random() * users.length)];
    const res = http.get(`${BASE_URL}/users/${id}`, tagged('get_user'));
    check(res, { 'get user 200': (r) => r.status === 200 });

  } else if (r < 0.50) {
    const res = http.get(`${BASE_URL}/products`, tagged('list_products'));
    check(res, { 'list products 200': (r) => r.status === 200 });

  } else if (r < 0.65) {
    const id = products[Math.floor(Math.random() * products.length)];
    const res = http.get(`${BASE_URL}/products/${id}`, tagged('get_product'));
    check(res, { 'get product 200': (r) => r.status === 200 });

  } else if (r < 0.80) {
    // POST /orders → orders-service → HTTP call users-service + HTTP call products-service
    // Este é o cenário que evidencia o overhead de comunicação inter-serviços
    const userId    = users[Math.floor(Math.random() * users.length)];
    const productId = products[Math.floor(Math.random() * products.length)];
    const res = http.post(`${BASE_URL}/orders`,
      JSON.stringify({ user_id: userId, items: [{ product_id: productId, quantity: 1 }] }),
      tagged('post_order', { headers: HEADERS }),
    );
    check(res, { 'create order 201': (r) => r.status === 201 });

  } else if (r < 0.90) {
    const res = http.get(`${BASE_URL}/orders`, tagged('list_orders'));
    check(res, { 'list orders 200': (r) => r.status === 200 });

  } else {
    const id = products[Math.floor(Math.random() * products.length)];
    const res = http.get(`${BASE_URL}/inventory/${id}`, tagged('get_inventory'));
    check(res, { 'get inventory 200': (r) => r.status === 200 });
  }

  sleep(0.1);
}
