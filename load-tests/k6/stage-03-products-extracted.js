import http from 'k6/http';
import { check, sleep } from 'k6';

// Stage 03: NGINX roteia /users → users-service, /products → products-service, resto → monolith-partial
const BASE_URL = __ENV.BASE_URL || 'http://localhost:8080';
const HEADERS = { 'Content-Type': 'application/json' };

export const options = {
  stages: [
    { duration: '30s', target: 50  },
    { duration: '5m',  target: 50  },
    { duration: '30s', target: 200 },
    { duration: '5m',  target: 200 },
    { duration: '30s', target: 500 },
    { duration: '5m',  target: 500 },
    { duration: '30s', target: 0   },
  ],
  thresholds: {
    'http_req_duration{scenario:default}': ['p(95)<500'],
    'http_req_failed':                     ['rate<0.01'],
  },
};

export function setup() {
  const users = [];
  const products = [];

  for (let i = 0; i < 20; i++) {
    const res = http.post(`${BASE_URL}/users`,
      JSON.stringify({ name: `User ${i}`, email: `user${i}@k6stage03.com` }),
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

  return { users, products };
}

export default function ({ users, products }) {
  const r = Math.random();

  if (r < 0.30) {
    // GET user — NGINX → users-service
    const id = users[Math.floor(Math.random() * users.length)];
    const res = http.get(`${BASE_URL}/users/${id}`);
    check(res, { 'get user 200': (r) => r.status === 200 });

  } else if (r < 0.50) {
    // LIST products — NGINX → products-service
    const res = http.get(`${BASE_URL}/products`);
    check(res, { 'list products 200': (r) => r.status === 200 });

  } else if (r < 0.65) {
    // GET product — NGINX → products-service
    const id = products[Math.floor(Math.random() * products.length)];
    const res = http.get(`${BASE_URL}/products/${id}`);
    check(res, { 'get product 200': (r) => r.status === 200 });

  } else if (r < 0.80) {
    // POST order — NGINX → monolith-partial (consulta DB direto, sem HTTP inter-serviços)
    const userId    = users[Math.floor(Math.random() * users.length)];
    const productId = products[Math.floor(Math.random() * products.length)];
    const res = http.post(`${BASE_URL}/orders`,
      JSON.stringify({ user_id: userId, items: [{ product_id: productId, quantity: 1 }] }),
      { headers: HEADERS },
    );
    check(res, { 'create order 201': (r) => r.status === 201 });

  } else if (r < 0.90) {
    // LIST orders — NGINX → monolith-partial
    const res = http.get(`${BASE_URL}/orders`);
    check(res, { 'list orders 200': (r) => r.status === 200 });

  } else {
    // GET inventory — NGINX → monolith-partial
    const id = products[Math.floor(Math.random() * products.length)];
    const res = http.get(`${BASE_URL}/inventory/${id}`);
    check(res, { 'get inventory 200': (r) => r.status === 200 });
  }

  sleep(0.1);
}
