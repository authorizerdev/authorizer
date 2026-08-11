// Stands in for an MCP client that identifies itself with a Client ID Metadata
// Document instead of a pre-registered client_id.
//
// Serves HTTPS, and that is not incidental: the spec requires a CIMD client_id
// to use the https scheme, and the AUTHORIZER SERVER — not the test process —
// is what fetches it. So this needs a certificate the server trusts, which the
// tls-certs service generates into a shared volume.
//
// `client_id` MUST equal the URL it is served from. That equality is the spec's
// central requirement and the only thing stopping any host from serving a
// document claiming to be some other client.
import express from 'express';
import https from 'node:https';
import fs from 'node:fs';

const BASE = process.env.SELF_BASE_URL || 'https://cimd-client:4300';
const app = express();

app.get('/client.json', (_req, res) => {
  res.json({
    client_id: `${BASE}/client.json`,
    client_name: 'E2E Playground Client',
    client_uri: BASE,
    redirect_uris: [`${BASE}/callback`],
    grant_types: ['authorization_code'],
    response_types: ['code'],
    token_endpoint_auth_method: 'none',
  });
});

// A document whose client_id names a DIFFERENT URL — the impersonation case.
// Accepting this would let any host claim to be any client.
app.get('/mismatched.json', (_req, res) => {
  res.json({
    client_id: `${BASE}/client.json`,
    client_name: 'Impersonator',
    redirect_uris: [`${BASE}/callback`],
  });
});

// Where the authorization code lands. Echoes the query so the test can read it.
app.get('/callback', (req, res) => {
  // HTML, not JSON: this is a NAVIGATION target, and a browser handles a
  // document far more predictably than an application/json body.
  res.type('html').send(`<!doctype html><title>callback</title><pre id="q">${JSON.stringify(req.query)}</pre>`);
});

app.get('/healthz', (_req, res) => res.sendStatus(204));

https
  .createServer(
    { key: fs.readFileSync('/certs/server.key'), cert: fs.readFileSync('/certs/server.crt') },
    app,
  )
  .listen(4300, () => console.log(`cimd-client listening on ${BASE}`));
