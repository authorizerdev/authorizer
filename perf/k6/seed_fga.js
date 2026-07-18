// Bulk-loads the FGA store so check_permissions/list_permissions benchmarks
// run against realistic background volume instead of a near-empty store.
// Writes model + TUPLES tuples in batches of 100 (OpenFGA's own per-Write
// limit), parallelized across VUS. Also creates one client_credentials
// client and one password user, printed at the end for the other scripts.
//
// Usage: ADMIN_SECRET=... TUPLES=1000000 k6 run perf/k6/seed_fga.js
import http from 'k6/http';
import { check } from 'k6';
import exec from 'k6/execution';

const BASE_URL = __ENV.BASE_URL || 'http://localhost:8080';
const ADMIN_SECRET = __ENV.ADMIN_SECRET;
const TOTAL_TUPLES = Number(__ENV.TUPLES || 100000);
const BATCH = 100; // OpenFGA's default max tuple ops per Write call.
const PASSWORD = 'Perf-Test-Passw0rd!';

if (!ADMIN_SECRET) {
  throw new Error('ADMIN_SECRET env var is required');
}

const MODEL_DSL = `model
  schema 1.1
type user
type document
  relations
    define viewer: [user]
    define can_view: viewer
`;

export const options = {
  scenarios: {
    seed: {
      executor: 'shared-iterations',
      vus: Number(__ENV.VUS || 20),
      iterations: Math.ceil(TOTAL_TUPLES / BATCH),
      maxDuration: __ENV.MAX_DURATION || '30m',
    },
  },
};

export function setup() {
  const adminLogin = http.post(
    `${BASE_URL}/v1/admin/login`,
    JSON.stringify({ admin_secret: ADMIN_SECRET }),
    { headers: { 'Content-Type': 'application/json' } }
  );
  check(adminLogin, { 'admin login 200': (r) => r.status === 200 });
  const cookie = (adminLogin.headers['Set-Cookie'] || '').split(';')[0];

  const model = http.post(`${BASE_URL}/v1/admin/fga/model`, JSON.stringify({ dsl: MODEL_DSL }), {
    headers: { 'Content-Type': 'application/json', Cookie: cookie },
  });
  check(model, { 'model written': (r) => r.status === 200 });

  const clientRes = http.post(
    `${BASE_URL}/v1/admin/create_client`,
    JSON.stringify({ name: `perf-s2s-${Date.now()}`, allowed_scopes: ['openid'] }),
    { headers: { 'Content-Type': 'application/json', Cookie: cookie } }
  );
  check(clientRes, { 'client created': (r) => r.status === 200 });

  const email = `perf-seed-${Date.now()}@example.com`;
  http.post(
    `${BASE_URL}/v1/signup`,
    JSON.stringify({ email, password: PASSWORD, confirm_password: PASSWORD }),
    { headers: { 'Content-Type': 'application/json' } }
  );

  return {
    cookie,
    client_id: clientRes.json('client.client_id'),
    client_secret: clientRes.json('client_secret'),
    user_email: email,
  };
}

export default function (data) {
  const start = exec.scenario.iterationInTest * BATCH;
  const tuples = [];
  for (let i = 0; i < BATCH; i++) {
    const n = start + i;
    tuples.push({ user: `user:perf-user-${n}`, relation: 'viewer', object: `document:perf-doc-${n}` });
  }
  const res = http.post(`${BASE_URL}/v1/admin/fga/tuples`, JSON.stringify({ tuples }), {
    headers: { 'Content-Type': 'application/json', Cookie: data.cookie },
  });
  check(res, { 'tuples written': (r) => r.status === 200 });
}

export function teardown(data) {
  console.log(
    `seed complete: client_id=${data.client_id} client_secret=${data.client_secret} user_email=${data.user_email} password=${PASSWORD}`
  );
}
