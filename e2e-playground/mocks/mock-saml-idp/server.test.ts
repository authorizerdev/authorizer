import assert from 'node:assert';
import { test } from 'node:test';
import { app } from './server';
import type { Server } from 'node:http';

let server: Server;

test.before(() => {
  server = app.listen(0);
});
test.after(() => server.close());

test('/sso responds 400 (not a crash) on a malformed SAMLRequest', async () => {
  const addr = server.address();
  const base = `http://127.0.0.1:${typeof addr === 'object' && addr ? addr.port : 0}`;

  const res = await fetch(`${base}/sso?SAMLRequest=not-valid-base64-deflated-xml&RelayState=test`);
  assert.strictEqual(res.status, 400);

  // Process (and this same server instance) must still be alive and serving
  // requests afterwards — a crash would make this fetch fail outright.
  const metadataRes = await fetch(`${base}/metadata`);
  assert.strictEqual(metadataRes.status, 200);
});
