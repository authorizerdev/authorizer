// JWT validation throughput — the highest-QPS path in real deployments
// (every downstream service call re-validates). Self-seeds a token unless
// TOKEN is passed in.
import http from 'k6/http';
import { check } from 'k6';

export const options = {
  vus: Number(__ENV.VUS || 50),
  duration: __ENV.DURATION || '30s',
};

const BASE_URL = __ENV.BASE_URL || 'http://localhost:8080';
const PASSWORD = 'Perf-Test-Passw0rd!';

export function setup() {
  if (__ENV.TOKEN) return { token: __ENV.TOKEN };

  const email = `perf-validate-${Date.now()}-${Math.floor(Math.random() * 1e6)}@example.com`;
  http.post(
    `${BASE_URL}/v1/signup`,
    JSON.stringify({ email, password: PASSWORD, confirm_password: PASSWORD }),
    { headers: { 'Content-Type': 'application/json' } }
  );
  const login = http.post(
    `${BASE_URL}/v1/login`,
    JSON.stringify({ email, password: PASSWORD }),
    { headers: { 'Content-Type': 'application/json' } }
  );
  check(login, { 'login 200': (r) => r.status === 200 });
  return { token: login.json('access_token') };
}

export default function (data) {
  const res = http.post(
    `${BASE_URL}/v1/validate_jwt_token`,
    JSON.stringify({ token_type: 'access_token', token: data.token }),
    { headers: { 'Content-Type': 'application/json' } }
  );
  check(res, {
    'validate 200': (r) => r.status === 200,
    'is_valid true': (r) => r.json('is_valid') === true,
  });
}
