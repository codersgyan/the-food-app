// prepare.js logs in the customers and every restaurant's tablet once, and
// saves their tokens to a file for shop.js. Logging in 20,000 tablets during
// the test itself would turn the first minute into a login storm that no real
// dinner rush has.
//
//     k6 run -e BASE_URL=https://food-api.sys0.in -e OUT=tokens.json prepare.js
//
// The phone numbers are the ones cmd/seed makes (see cmd/seed/main.go).

import http from 'k6/http';

const BASE_URL = __ENV.BASE_URL || 'http://localhost:8080';
const CUSTOMERS = Number(__ENV.CUSTOMERS || 2000); // logged in up front
const RESTAURANTS = Number(__ENV.RESTAURANTS || 20000); // one tablet each
const BATCH = Number(__ENV.BATCH || 100); // logins sent at once

export const options = {
  vus: 1,
  iterations: 1,
  setupTimeout: '30m',
  batch: BATCH,
  batchPerHost: BATCH,
};

function staffPhone(restaurantId) {
  // The first 20 restaurants are the hand-made ones, with their own numbers.
  if (restaurantId <= 20) {
    return '+9198100000' + String(restaurantId).padStart(2, '0');
  }
  return '+9197' + String(restaurantId).padStart(8, '0');
}

function customerPhone(n) {
  return '+9196' + String(n).padStart(8, '0');
}

function loginAll(phones) {
  const tokens = [];
  for (let i = 0; i < phones.length; i += BATCH) {
    const requests = phones.slice(i, i + BATCH).map((phone) => [
      'POST',
      `${BASE_URL}/v1/login/verify`,
      JSON.stringify({ phone, code: '123456' }),
      { headers: { 'Content-Type': 'application/json' } },
    ]);
    http.batch(requests).forEach((res, j) => {
      if (res.status !== 200) {
        throw new Error(`login ${phones[i + j]}: ${res.status} ${res.body}`);
      }
      tokens.push(res.json('token'));
    });
  }
  return tokens;
}

function range(from, to) {
  const list = [];
  for (let n = from; n <= to; n++) list.push(n);
  return list;
}

export function setup() {
  const customers = loginAll(range(1, CUSTOMERS).map(customerPhone));
  const tablets = loginAll(range(1, RESTAURANTS).map(staffPhone)).map((token, i) => ({
    restaurantId: i + 1,
    token,
  }));
  console.log(`logged in ${customers.length} customers and ${tablets.length} tablets`);
  return { customers, tablets };
}

export default function () {}

export function handleSummary(data) {
  return { [__ENV.OUT || 'tokens.json']: JSON.stringify(data.setup_data) };
}
