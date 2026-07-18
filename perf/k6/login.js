// Login throughput: password verify + token issuance, repeated against one
// seeded user (bcrypt cost is per-request regardless of which user).
import http from 'k6/http';
import { check } from 'k6';

export const options = {
  vus: Number(__ENV.VUS || 10),
  duration: __ENV.DURATION || '30s',
};

const BASE_URL = __ENV.BASE_URL || 'http://localhost:8080';
const PASSWORD = 'Perf-Test-Passw0rd!';

export function setup() {
  const email = `perf-login-${exec_id()}@example.com`;
  const res = http.post(
    `${BASE_URL}/v1/signup`,
    JSON.stringify({ email, password: PASSWORD, confirm_password: PASSWORD }),
    { headers: { 'Content-Type': 'application/json', Origin: BASE_URL } }
  );
  check(res, { 'signup 200': (r) => r.status === 200 });
  return { email };
}

export default function (data) {
  const res = http.post(
    `${BASE_URL}/v1/login`,
    JSON.stringify({ email: data.email, password: PASSWORD }),
    { headers: { 'Content-Type': 'application/json', Origin: BASE_URL } }
  );
  check(res, {
    'login 200': (r) => r.status === 200,
    'has access_token': (r) => !!r.json('access_token'),
  });
}

function exec_id() {
  return `${Date.now()}-${Math.floor(Math.random() * 1e6)}`;
}
