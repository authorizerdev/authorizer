import express from 'express';

const app = express();
// Capture the raw request bytes (not a re-serialized copy) so a test can verify
// the X-Authorizer-Signature HMAC over the EXACT payload authorizer signed —
// re-marshalling JSON would reorder/reformat keys and break the signature.
app.use(express.raw({ type: '*/*', limit: '1mb' }));

interface Delivery {
  eventName: string;
  rawBody: string;
  signature: string;
  body: any;
}

// Keyed by user email (unique per test), then by event name, so a single global
// webhook receiving many SCIM events across parallel tests can be queried for one
// specific user's provisioned / scim_updated / deprovisioned deliveries.
const byEmail = new Map<string, Map<string, Delivery>>();

app.post('/webhook', (req, res) => {
  const rawBody = Buffer.isBuffer(req.body) ? req.body.toString('utf8') : '';
  let body: any;
  try {
    body = JSON.parse(rawBody);
  } catch {
    res.sendStatus(400);
    return;
  }
  const eventName: string = body.event_name;
  const email: string | undefined = body.user?.email;
  if (email && eventName) {
    if (!byEmail.has(email)) byEmail.set(email, new Map());
    byEmail.get(email)!.set(eventName, {
      eventName,
      rawBody,
      signature: (req.header('X-Authorizer-Signature') as string) || '',
      body,
    });
  }
  res.sendStatus(200);
});

// Returns every event delivered for one user's email:
//   { email, events: { "user.provisioned": { signature, rawBody, body }, ... } }
app.get('/webhook/:email', (req, res) => {
  const events = byEmail.get(req.params.email);
  if (!events) {
    res.sendStatus(404);
    return;
  }
  res.json({ email: req.params.email, events: Object.fromEntries(events) });
});

app.listen(4200, () => console.log('webhook-sink listening on :4200'));
