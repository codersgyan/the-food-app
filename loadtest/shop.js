// shop.js sends the traffic of a dinner rush: customers browsing and ordering,
// and every restaurant's tablet asking for new orders.
//
//     k6 run -e BASE_URL=https://food-api.sys0.in -e MULT=1 -e TABLETS=20000 shop.js
//
// The numbers come from the lessons. 1x is 17 orders a second (120,000 orders
// in the two-hour dinner peak), and about 20 requests for each order. MULT
// multiplies the customers. The tablets don't change with MULT: a restaurant's
// tablet asks for new orders every few seconds, however busy it is.

import http from 'k6/http';
import { check, sleep } from 'k6';
import { SharedArray } from 'k6/data';
import { Counter, Trend } from 'k6/metrics';

const BASE_URL = __ENV.BASE_URL || 'http://localhost:8080';
const MULT = Number(__ENV.MULT || 1);
const DURATION = __ENV.DURATION || '3m';
const ORDERS_PER_SEC = Number(__ENV.ORDERS_PER_SEC || 17) * MULT;

// One visit to the app.
const BUY_SHARE = 0.4; // 4 in 10 visits end in an order; the rest only browse
const LOGIN_SHARE = 0.1; // 1 in 10 visits starts with a login
const MENUS = 4; // menus opened in a visit
const ALL_CUSTOMERS = Number(__ENV.ALL_CUSTOMERS || 10000); // seeded customers

// The restaurants' tablets. 0 turns them off.
const TABLETS = Number(__ENV.TABLETS || 0);
const TABLET_INTERVAL = Number(__ENV.TABLET_INTERVAL || 10); // seconds between polls

const VISITS_PER_SEC = ORDERS_PER_SEC / BUY_SHARE;
const TOKENS = __ENV.TOKENS || './tokens.json';
const customers = new SharedArray('customers', () => JSON.parse(open(TOKENS)).customers);
const tablets = new SharedArray('tablets', () => JSON.parse(open(TOKENS)).tablets);

const ordersPlaced = new Counter('orders_placed');
const ordersPaid = new Counter('orders_paid');
const ordersCooked = new Counter('orders_cooked');
const paymentTime = new Trend('payment_time', true); // from Pay to "paid"

// Every request is named, so the results show each kind on its own.
const NAMES = ['login', 'verify', 'me', 'near', 'menu', 'order', 'pay', 'status', 'tablet_poll', 'accept', 'ready'];

const scenarios = {
  customers: {
    executor: 'constant-arrival-rate',
    exec: 'customer',
    rate: Math.round(VISITS_PER_SEC * 10),
    timeUnit: '10s',
    duration: DURATION,
    // A visit lasts about 10 seconds, with the pauses, so about 10 visits run
    // at once for every visit that starts each second.
    preAllocatedVUs: Math.ceil(VISITS_PER_SEC * 15),
    maxVUs: Math.ceil(VISITS_PER_SEC * 30),
  },
};
if (TABLETS > 0) {
  scenarios.tablets = {
    executor: 'constant-arrival-rate',
    exec: 'tablet',
    rate: TABLETS,
    timeUnit: `${TABLET_INTERVAL}s`,
    duration: DURATION,
    preAllocatedVUs: Math.max(10, Math.ceil(TABLETS / TABLET_INTERVAL / 2)),
    maxVUs: Math.max(20, Math.ceil(TABLETS / TABLET_INTERVAL) * 2),
  };
}

const thresholds = { http_req_failed: ['rate<0.01'], http_req_duration: ['p(99)<1000'] };
for (const name of NAMES) {
  thresholds[`http_req_duration{name:${name}}`] = ['p(99)<1000'];
  thresholds[`http_req_failed{name:${name}}`] = ['rate<0.01'];
}

export const options = {
  scenarios,
  thresholds,
  summaryTrendStats: ['count', 'avg', 'med', 'p(90)', 'p(95)', 'p(99)', 'max'],
  discardResponseBodies: false,
};

// Phones ask for compressed responses, so nginx gzips them. k6 doesn't ask by
// default, so we ask for it here.
const JSON_HEADERS = { 'Content-Type': 'application/json', 'Accept-Encoding': 'gzip' };

function auth(token) {
  return { Authorization: `Bearer ${token}` };
}

function req(method, path, body, token, name) {
  const params = { headers: { ...JSON_HEADERS, ...(token ? auth(token) : {}) }, tags: { name } };
  return http.request(method, `${BASE_URL}${path}`, body === null ? null : JSON.stringify(body), params);
}

function think() {
  sleep(0.5 + Math.random()); // 0.5 to 1.5 seconds, like a person reading the screen
}

function pick(list) {
  return list[Math.floor(Math.random() * list.length)];
}

// A random spot in a box around Bengaluru, like the seeded restaurants.
function randomSpot() {
  return `${(12.85 + Math.random() * 0.25).toFixed(4)},${(77.5 + Math.random() * 0.25).toFixed(4)}`;
}

export function customer() {
  let token;
  if (Math.random() < LOGIN_SHARE) {
    const phone = '+9196' + String(1 + Math.floor(Math.random() * ALL_CUSTOMERS)).padStart(8, '0');
    check(req('POST', '/v1/login', { phone }, null, 'login'), { 'login 200': (r) => r.status === 200 });
    think();
    const res = req('POST', '/v1/login/verify', { phone, code: '123456' }, null, 'verify');
    if (!check(res, { 'verify 200': (r) => r.status === 200 })) return;
    token = res.json('token');
  } else {
    token = pick(customers);
  }

  const me = req('GET', '/v1/me', null, token, 'me');
  if (!check(me, { 'me 200': (r) => r.status === 200 })) return;
  const addressId = me.json('addresses.0.id');
  think();

  // The nearest restaurants come first. Only read the start of the list:
  // a visitor looks at the top of the screen, not at all 4,000 restaurants.
  const near = req('GET', `/v1/restaurants?near=${randomSpot()}`, null, token, 'near');
  if (!check(near, { 'near 200': (r) => r.status === 200 })) return;
  const nearby = [...near.body.substring(0, 4000).matchAll(/"id":(\d+)/g)].map((m) => Number(m[1]));
  if (nearby.length === 0) return;
  think();

  let menu = null;
  let restaurantId = 0;
  for (let i = 0; i < MENUS; i++) {
    restaurantId = pick(nearby);
    const res = req('GET', `/v1/restaurants/${restaurantId}/menu`, null, token, 'menu');
    if (!check(res, { 'menu 200': (r) => r.status === 200 })) return;
    menu = res;
    think();
  }

  if (Math.random() >= BUY_SHARE) return; // only browsing this time

  const dishes = menu.json('dishes').filter((d) => !d.sold_out);
  if (dishes.length === 0) return;
  const items = [];
  const count = 1 + Math.floor(Math.random() * Math.min(3, dishes.length));
  for (const d of dishes.sort(() => Math.random() - 0.5).slice(0, count)) {
    items.push({ dish_id: d.id, quantity: 1 + Math.floor(Math.random() * 2) });
  }
  const order = req('POST', '/v1/orders', { restaurant_id: restaurantId, address_id: addressId, items }, token, 'order');
  if (!check(order, { 'order 201': (r) => r.status === 201 })) return;
  ordersPlaced.add(1);
  const orderId = order.json('id');
  think();

  const pay = req('POST', `/v1/orders/${orderId}/pay`, null, token, 'pay');
  if (!check(pay, { 'pay 200': (r) => r.status === 200 })) return;
  const paidFrom = Date.now();

  // The app checks the order every second until the payment goes through.
  for (let i = 0; i < 10; i++) {
    sleep(1);
    const status = req('GET', `/v1/orders/${orderId}`, null, token, 'status');
    if (!check(status, { 'status 200': (r) => r.status === 200 })) return;
    if (status.json('status') !== 'paying') {
      ordersPaid.add(1);
      paymentTime.add(Date.now() - paidFrom);
      return;
    }
  }
}

export function tablet() {
  const t = pick(tablets);
  const res = req('GET', `/v1/restaurants/${t.restaurantId}/orders?status=paid`, null, t.token, 'tablet_poll');
  if (!check(res, { 'tablet_poll 200': (r) => r.status === 200 })) return;
  for (const o of res.json('orders')) {
    // Accept the order, and mark it ready. The test doesn't wait for the cooking.
    const accept = req('POST', `/v1/orders/${o.id}/accept`, null, t.token, 'accept');
    check(accept, { 'accept 200 or 409': (r) => r.status === 200 || r.status === 409 });
    if (accept.status !== 200) continue;
    const ready = req('POST', `/v1/orders/${o.id}/ready`, null, t.token, 'ready');
    if (check(ready, { 'ready 200': (r) => r.status === 200 })) ordersCooked.add(1);
  }
}

// The summary: a table of every kind of request, and the whole k6 result as
// JSON in OUT.
export function handleSummary(data) {
  const m = data.metrics;
  const ms = (v) => (v === undefined ? '-' : v < 10 ? v.toFixed(1) : Math.round(v).toString());
  const lines = [
    `MULT=${MULT}  orders/s=${ORDERS_PER_SEC}  visits/s=${VISITS_PER_SEC.toFixed(1)}  tablets=${TABLETS} every ${TABLET_INTERVAL}s  duration=${DURATION}`,
    '',
    'request        count     req/s    p50 ms   p95 ms   p99 ms   max ms   failed',
  ];
  const seconds = data.state.testRunDurationMs / 1000;
  for (const name of [...NAMES, null]) {
    const d = m[name ? `http_req_duration{name:${name}}` : 'http_req_duration'];
    const f = m[name ? `http_req_failed{name:${name}}` : 'http_req_failed'];
    if (!d || !d.values.count) continue;
    lines.push(
      [
        (name || 'ALL').padEnd(12),
        String(d.values.count).padStart(8),
        (d.values.count / seconds).toFixed(1).padStart(9),
        ms(d.values.med).padStart(9),
        ms(d.values['p(95)']).padStart(8),
        ms(d.values['p(99)']).padStart(8),
        ms(d.values.max).padStart(8),
        `${((f ? f.values.rate : 0) * 100).toFixed(2)}%`.padStart(9),
      ].join(' '),
    );
  }
  const val = (k, f) => (m[k] ? m[k].values[f] : 0);
  lines.push(
    '',
    `orders placed ${val('orders_placed', 'count')}, paid ${val('orders_paid', 'count')}, cooked ${val('orders_cooked', 'count')}`,
    `payment time p50 ${ms(val('payment_time', 'med'))} ms, p99 ${ms(val('payment_time', 'p(99)'))} ms`,
    `dropped iterations (k6 couldn't start a visit on time) ${val('dropped_iterations', 'count')}`,
    `max VUs ${val('vus_max', 'max')}`,
    '',
  );
  return {
    stdout: lines.join('\n') + '\n',
    [__ENV.OUT || 'summary.json']: JSON.stringify(data, null, 1),
  };
}
