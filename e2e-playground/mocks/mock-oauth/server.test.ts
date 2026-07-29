import assert from 'node:assert';
import { test } from 'node:test';
import { app } from './server';
import type { Server } from 'node:http';

let server: Server;

test.before(() => {
  server = app.listen(0);
});
test.after(() => server.close());

test('token endpoint issues an access token, userinfo returns configured profile', async () => {
  const addr = server.address();
  const base = `http://127.0.0.1:${typeof addr === 'object' && addr ? addr.port : 0}`;

  await fetch(`${base}/github/__configure`, {
    method: 'POST',
    headers: { 'content-type': 'application/json' },
    body: JSON.stringify({ profile: { name: 'Ada Lovelace', email: 'ada@example.com', avatar_url: 'https://example.com/a.png' } }),
  });

  const tokenRes = await fetch(`${base}/github/token`, {
    method: 'POST',
    headers: { 'content-type': 'application/x-www-form-urlencoded' },
    body: 'grant_type=authorization_code&code=any-code',
  });
  assert.strictEqual(tokenRes.status, 200);
  const { access_token } = await tokenRes.json();
  assert.ok(access_token);

  const userRes = await fetch(`${base}/github/userinfo`, { headers: { authorization: `Bearer ${access_token}` } });
  assert.strictEqual(userRes.status, 200);
  const profile = await userRes.json();
  assert.strictEqual(profile.name, 'Ada Lovelace');
  assert.strictEqual(profile.email, 'ada@example.com');
});
