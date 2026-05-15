import http from 'k6/http';
import { check, sleep } from 'k6';

// Stage 05: microsserviços completos com schema-per-service
// Cada serviço usa seu próprio schema PostgreSQL (users_schema, products_schema, orders_schema, inventory_schema)
// POST /orders continua com 2 chamadas HTTP inter-serviços (users + products)
const BASE_URL = __ENV.BASE_URL || 'http://localhost:8080';
const HEADERS = { 'Content-Type': 'application/json' };
const RUN_ID = Date.now();

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
    if (res.status === 201) {
      const id = res.json('id');
      products.push(id);
      // Seed inventory — schema isolation means inventory-service owns its own table
      // Without seeding, GET /inventory/:id returns 404 for every call
      http.patch(`${BASE_URL}/inventory/${id}`,
        JSON.stringify({ quantity: 10000 }),
        { headers: HEADERS },
      );
    }
  }

  return { users, products };
}

export default function ({ users, products }) {
  const r = Math.random();

  if (r < 0.30) {
    const id = users[Math.floor(Math.random() * users.length)];
    const res = http.get(`${BASE_URL}/users/${id}`);
    check(res, { 'get user 200': (r) => r.status === 200 });

  } else if (r < 0.50) {
    const res = http.get(`${BASE_URL}/products`);
    check(res, { 'list products 200': (r) => r.status === 200 });

  } else if (r < 0.65) {
    const id = products[Math.floor(Math.random() * products.length)];
    const res = http.get(`${BASE_URL}/products/${id}`);
    check(res, { 'get product 200': (r) => r.status === 200 });

  } else if (r < 0.80) {
    // POST /orders → orders-service → HTTP call users-service + HTTP call products-service
    const userId    = users[Math.floor(Math.random() * users.length)];
    const productId = products[Math.floor(Math.random() * products.length)];
    const res = http.post(`${BASE_URL}/orders`,
      JSON.stringify({ user_id: userId, items: [{ product_id: productId, quantity: 1 }] }),
      { headers: HEADERS },
    );
    check(res, { 'create order 201': (r) => r.status === 201 });

  } else if (r < 0.90) {
    const res = http.get(`${BASE_URL}/orders`);
    check(res, { 'list orders 200': (r) => r.status === 200 });

  } else {
    const id = products[Math.floor(Math.random() * products.length)];
    const res = http.get(`${BASE_URL}/inventory/${id}`);
    check(res, {
      'get inventory 200':        (r) => r.status === 200,
      'inventory has product_id': (r) => r.json('product_id') > 0,
    });
  }

  sleep(0.1);
}
