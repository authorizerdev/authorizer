// S2S throughput via the client_credentials grant — no user in the loop,
// pure machine-to-machine token issuance. Needs a client created via
// seed_fga.js (it prints client_id/client_secret) or POST /v1/admin/create_client.
import http from 'k6/http';
import { check } from 'k6';

export const options = {
  vus: Number(__ENV.VUS || 20),
  duration: __ENV.DURATION || '30s',
};

const BASE_URL = __ENV.BASE_URL || 'http://localhost:8080';
const CLIENT_ID = __ENV.CLIENT_ID;
const CLIENT_SECRET = __ENV.CLIENT_SECRET;

export function setup() {
  if (!CLIENT_ID || !CLIENT_SECRET) {
    throw new Error('CLIENT_ID and CLIENT_SECRET env vars are required (from seed_fga.js output or /v1/admin/create_client)');
  }
}

export default function () {
  const res = http.post(`${BASE_URL}/oauth/token`, {
    grant_type: 'client_credentials',
    client_id: CLIENT_ID,
    client_secret: CLIENT_SECRET,
  });
  check(res, {
    'token 200': (r) => r.status === 200,
    'has access_token': (r) => !!r.json('access_token'),
  });
}
