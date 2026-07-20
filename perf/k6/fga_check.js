// check_permissions throughput. Self-contained: creates its own user + one
// tuple so results are meaningful on a fresh store. Run seed_fga.js first
// (and set ADMIN_SECRET here too) to also load millions of unrelated tuples
// into the store, so Check resolves against realistic background volume
// instead of a nearly-empty one.
import http from 'k6/http';
import { check } from 'k6';

export const options = {
  vus: Number(__ENV.VUS || 50),
  duration: __ENV.DURATION || '30s',
};

const BASE_URL = __ENV.BASE_URL || 'http://localhost:8080';
const ADMIN_SECRET = __ENV.ADMIN_SECRET;
const PASSWORD = 'Perf-Test-Passw0rd!';
const OBJECT = 'document:perf-check';

// Matches perf/k6/seed_fga.js — a shared model means this script also works
// against a store seed_fga.js already seeded.
const MODEL_DSL = `model
  schema 1.1
type user
type document
  relations
    define viewer: [user]
    define can_view: viewer
`;

export function setup() {
  let token = __ENV.TOKEN;
  let userId = __ENV.USER_ID;

  if (!token) {
    const email = `perf-check-${Date.now()}-${Math.floor(Math.random() * 1e6)}@example.com`;
    http.post(
      `${BASE_URL}/v1/signup`,
      JSON.stringify({ email, password: PASSWORD, confirm_password: PASSWORD }),
      { headers: { 'Content-Type': 'application/json', Origin: BASE_URL } }
    );
    const login = http.post(
      `${BASE_URL}/v1/login`,
      JSON.stringify({ email, password: PASSWORD }),
      { headers: { 'Content-Type': 'application/json', Origin: BASE_URL } }
    );
    check(login, { 'login 200': (r) => r.status === 200 });
    token = login.json('access_token');
    userId = login.json('user.id');
  }

  if (ADMIN_SECRET) {
    const adminLogin = http.post(
      `${BASE_URL}/v1/admin/login`,
      JSON.stringify({ admin_secret: ADMIN_SECRET }),
      { headers: { 'Content-Type': 'application/json', Origin: BASE_URL } }
    );
    check(adminLogin, { 'admin login 200': (r) => r.status === 200 });
    const cookie = (adminLogin.headers['Set-Cookie'] || '').split(';')[0];

    http.post(`${BASE_URL}/v1/admin/fga/model`, JSON.stringify({ dsl: MODEL_DSL }), {
      headers: { 'Content-Type': 'application/json', Origin: BASE_URL, Cookie: cookie },
    });
    const tupleRes = http.post(
      `${BASE_URL}/v1/admin/fga/tuples`,
      JSON.stringify({ tuples: [{ user: `user:${userId}`, relation: 'viewer', object: OBJECT }] }),
      { headers: { 'Content-Type': 'application/json', Origin: BASE_URL, Cookie: cookie } }
    );
    check(tupleRes, { 'tuple written': (r) => r.status === 200 });
  }

  return { token };
}

export default function (data) {
  const res = http.post(
    `${BASE_URL}/v1/check_permissions`,
    JSON.stringify({ checks: [{ relation: 'can_view', object: OBJECT }] }),
    { headers: { 'Content-Type': 'application/json', Origin: BASE_URL, Authorization: `Bearer ${data.token}` } }
  );
  check(res, { 'check_permissions 200': (r) => r.status === 200 });
}
