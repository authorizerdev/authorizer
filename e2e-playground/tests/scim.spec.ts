// e2e-playground/tests/scim.spec.ts
import { test, expect } from '@playwright/test';
import { createHmac } from 'crypto';
import { addWebhook, createOrg, createSCIMEndpoint, getUserPhoneNumberByEmail } from '../fixtures/adminClient';

test.describe('SCIM', () => {
  test('provision, update, and deprovision a user via SCIM', async ({ request }) => {
    const org = await createOrg(`scim-${Date.now()}`);
    const { token } = await createSCIMEndpoint(org.id);
    const headers = { Authorization: `Bearer ${token}`, 'Content-Type': 'application/scim+json' };
    const email = `scim-user-${org.id}@example.com`;

    const createRes = await request.post('/scim/v2/Users', {
      headers,
      data: {
        schemas: ['urn:ietf:params:scim:schemas:core:2.0:User'],
        userName: email,
        name: { givenName: 'Katherine', familyName: 'Johnson' },
        emails: [{ value: email, primary: true }],
        active: true,
      },
    });
    expect(createRes.status()).toBe(201);
    const created = await createRes.json();
    expect(created.userName).toBe(email);

    const patchRes = await request.patch(`/scim/v2/Users/${created.id}`, {
      headers,
      data: { schemas: ['urn:ietf:params:scim:api:messages:2.0:PatchOp'], Operations: [{ op: 'replace', value: { active: false } }] },
    });
    expect(patchRes.status()).toBe(200);
    const patched = await patchRes.json();
    expect(patched.active).toBe(false);

    const deleteRes = await request.delete(`/scim/v2/Users/${created.id}`, { headers });
    expect(deleteRes.status()).toBe(204);
  });

  test('cross-org isolation: org A token cannot read org B users', async ({ request }) => {
    const orgA = await createOrg(`scim-a-${Date.now()}`);
    const orgB = await createOrg(`scim-b-${Date.now()}`);
    const { token: tokenA } = await createSCIMEndpoint(orgA.id);
    const { token: tokenB } = await createSCIMEndpoint(orgB.id);
    const headersB = { Authorization: `Bearer ${tokenB}`, 'Content-Type': 'application/scim+json' };

    const createRes = await request.post('/scim/v2/Users', {
      headers: headersB,
      data: {
        schemas: ['urn:ietf:params:scim:schemas:core:2.0:User'],
        userName: `org-b-only-${orgB.id}@example.com`,
        active: true,
      },
    });
    const orgBUser = await createRes.json();

    const crossOrgRes = await request.get(`/scim/v2/Users/${orgBUser.id}`, {
      headers: { Authorization: `Bearer ${tokenA}` },
    });
    expect(crossOrgRes.status()).toBe(404);
  });
});

// parseUserFilter (internal/http_handlers/scim/users.go) supports exactly
// eq/ne/co/sw/pr over userName, emails.value, name.givenName, name.familyName,
// active, externalId — a single-term filter only. Compound expressions
// (and/or/not/parens) are rejected outright with 400 invalidFilter, never
// silently matched as everything or nothing (a connector doing directory sync
// must be able to tell "unsupported filter" apart from "no matches").
test.describe('SCIM filter operators', () => {
  test('eq, co, sw, and pr all match the expected user', async ({ request }) => {
    const org = await createOrg(`scim-filter-${crypto.randomUUID()}`);
    const { token } = await createSCIMEndpoint(org.id);
    const headers = { Authorization: `Bearer ${token}`, 'Content-Type': 'application/scim+json' };

    const emailA = `filter-eq-${org.id}@example.com`;
    const emailB = `filter-other-${org.id}@example.com`;

    const userA = await (
      await request.post('/scim/v2/Users', {
        headers,
        data: {
          schemas: ['urn:ietf:params:scim:schemas:core:2.0:User'],
          userName: emailA,
          externalId: 'ext-alpha',
          name: { givenName: 'Ada', familyName: 'Lovelace' },
          emails: [{ value: emailA, primary: true }],
          active: true,
        },
      })
    ).json();

    // userB has no externalId — distinguishes the `pr` (present) case below.
    await request.post('/scim/v2/Users', {
      headers,
      data: {
        schemas: ['urn:ietf:params:scim:schemas:core:2.0:User'],
        userName: emailB,
        name: { givenName: 'Grace', familyName: 'Hopper' },
        emails: [{ value: emailB, primary: true }],
        active: true,
      },
    });

    const eqRes = await request.get('/scim/v2/Users', { headers, params: { filter: `userName eq "${emailA}"` } });
    expect(eqRes.status()).toBe(200);
    const eqBody = await eqRes.json();
    expect(eqBody.totalResults).toBe(1);
    expect(eqBody.Resources[0].id).toBe(userA.id);

    // "Ada" contains "da"; "Grace"/"Hopper" do not.
    const coRes = await request.get('/scim/v2/Users', { headers, params: { filter: 'name.givenName co "da"' } });
    expect(coRes.status()).toBe(200);
    const coBody = await coRes.json();
    expect(coBody.totalResults).toBe(1);
    expect(coBody.Resources[0].id).toBe(userA.id);

    const swRes = await request.get('/scim/v2/Users', { headers, params: { filter: 'userName sw "filter-eq-"' } });
    expect(swRes.status()).toBe(200);
    const swBody = await swRes.json();
    expect(swBody.totalResults).toBe(1);
    expect(swBody.Resources[0].id).toBe(userA.id);

    const prRes = await request.get('/scim/v2/Users', { headers, params: { filter: 'externalId pr' } });
    expect(prRes.status()).toBe(200);
    const prBody = await prRes.json();
    expect(prBody.totalResults).toBe(1);
    expect(prBody.Resources[0].id).toBe(userA.id);
  });

  test('compound and/or filters are rejected with 400, not silently matched', async ({ request }) => {
    const org = await createOrg(`scim-filter-compound-${crypto.randomUUID()}`);
    const { token } = await createSCIMEndpoint(org.id);
    const headers = { Authorization: `Bearer ${token}`, 'Content-Type': 'application/scim+json' };
    const email = `filter-compound-${org.id}@example.com`;

    await request.post('/scim/v2/Users', {
      headers,
      data: { schemas: ['urn:ietf:params:scim:schemas:core:2.0:User'], userName: email, active: true },
    });

    const andRes = await request.get('/scim/v2/Users', {
      headers,
      params: { filter: `userName eq "${email}" and active eq true` },
    });
    expect(andRes.status()).toBe(400);

    const orRes = await request.get('/scim/v2/Users', {
      headers,
      params: { filter: 'userName eq "a@example.com" or userName eq "b@example.com"' },
    });
    expect(orRes.status()).toBe(400);
  });
});

// applyNoPathUserPatch / parseUserPatch (internal/http_handlers/scim/users.go)
// support both the path-qualified PatchOp shape ({"path":"name.givenName",...})
// and the no-path attribute-map shape ({"value":{"name":{...}}}). This covers
// the attributes beyond `active` that the original spec didn't touch:
// name.givenName/familyName, phoneNumbers, and externalId.
test.describe('SCIM full-attribute PATCH', () => {
  test('PATCH updates name, externalId (path-qualified) and phoneNumbers (no-path shape)', async ({ request }) => {
    const org = await createOrg(`scim-patch-full-${crypto.randomUUID()}`);
    const { token } = await createSCIMEndpoint(org.id);
    const headers = { Authorization: `Bearer ${token}`, 'Content-Type': 'application/scim+json' };
    const email = `patch-full-${org.id}@example.com`;

    const created = await (
      await request.post('/scim/v2/Users', {
        headers,
        data: {
          schemas: ['urn:ietf:params:scim:schemas:core:2.0:User'],
          userName: email,
          name: { givenName: 'Original', familyName: 'Name' },
          emails: [{ value: email, primary: true }],
          active: true,
        },
      })
    ).json();

    // Path-qualified PatchOp shape.
    const pathPatchRes = await request.patch(`/scim/v2/Users/${created.id}`, {
      headers,
      data: {
        schemas: ['urn:ietf:params:scim:api:messages:2.0:PatchOp'],
        Operations: [
          { op: 'replace', path: 'name.givenName', value: 'Augusta' },
          { op: 'replace', path: 'name.familyName', value: 'King' },
          { op: 'replace', path: 'externalId', value: `ext-${org.id}` },
        ],
      },
    });
    expect(pathPatchRes.status()).toBe(200);
    const pathPatched = await pathPatchRes.json();
    expect(pathPatched.name.givenName).toBe('Augusta');
    expect(pathPatched.name.familyName).toBe('King');
    expect(pathPatched.externalId).toBe(`ext-${org.id}`);

    // No-path attribute-map shape, for phoneNumbers. The SCIM response never
    // echoes phoneNumbers back (scimUserResource has no phone field), so
    // persistence is verified out-of-band via the admin _users query.
    const phone = `+1555${Math.floor(1000000 + Math.random() * 9000000)}`;
    const phonePatchRes = await request.patch(`/scim/v2/Users/${created.id}`, {
      headers,
      data: {
        schemas: ['urn:ietf:params:scim:api:messages:2.0:PatchOp'],
        Operations: [{ op: 'replace', value: { phoneNumbers: [{ value: phone }] } }],
      },
    });
    expect(phonePatchRes.status()).toBe(200);

    const storedPhone = await getUserPhoneNumberByEmail(email);
    expect(storedPhone).toBe(phone);
  });
});

// SCIM provisioning-lifecycle webhooks (user.provisioned/deprovisioned/
// scim_updated, internal/constants/webhook_event_scim.go) are fired from
// internal/service/scim/scim.go and now exercised end-to-end below. Getting
// here took clearing TWO walls that previously made this test un-writable:
//
//   1. internal/validators/webhook.go's IsValidWebhookEventName allow-list did
//      not include the SCIM events, so `_add_webhook` rejected them with
//      "invalid event name user.provisioned". Fixed (allow-list now lists all
//      six SCIM events) — a webhook for a SCIM lifecycle event can be registered.
//
//   2. Both SSRF chokepoints unconditionally rejected private-network IPs, so no
//      webhook could target the docker-private webhook-sink mock: the
//      registration-time check (validators.ValidateEndpointURL in
//      internal/service/admin_webhooks.go AddWebhook) AND the delivery-time check
//      (validators.SafeHTTPClient in internal/events/events.go deliver()). Both
//      now take an allowPrivate flag threaded from Config.TestAllowPrivateWebhookHosts
//      (CLI: --test-allow-private-webhook-hosts, set true only on the `authorizer`
//      service in docker-compose.yml — least privilege). The flag relaxes ONLY the
//      private-IP rejection; the http/https scheme allow-list and DNS-rebinding
//      host pin stay enforced, and it is a true no-op when unset (production
//      default) — reachable solely via the operator CLI flag, never a request /
//      GraphQL / per-webhook field. Mirrors the SSO broker's existing
//      --test-allow-private-sso-hosts precedent exactly.
//
// The Env==TestEnv fake-200 short-circuit the old comment worried about is not a
// factor: the e2e-playground authorizer never runs with --env test, so deliver()
// makes a real HTTP POST.
test.describe('SCIM provisioning webhooks', () => {
  const CLIENT_SECRET = 'e2e-client-secret'; // matches --client-secret in docker-compose.yml
  const WEBHOOK_SINK = process.env.WEBHOOK_SINK_BASE_URL || 'http://localhost:4200';

  test('delivers user.provisioned / user.scim_updated / user.deprovisioned with a valid HMAC signature', async ({
    request,
  }) => {
    // Register listeners for all three lifecycle events BEFORE provisioning, so
    // the provisioned delivery is captured. Webhooks are global; each fires on
    // its event_name across every org on this instance. The mock keys deliveries
    // by user email (unique per test), so parallel specs never collide.
    const endpoint = `${WEBHOOK_SINK}/webhook`;
    await addWebhook({ eventName: 'user.provisioned', endpoint });
    await addWebhook({ eventName: 'user.scim_updated', endpoint });
    await addWebhook({ eventName: 'user.deprovisioned', endpoint });

    const org = await createOrg(`scim-webhook-${crypto.randomUUID()}`);
    const { token } = await createSCIMEndpoint(org.id);
    const headers = { Authorization: `Bearer ${token}`, 'Content-Type': 'application/scim+json' };
    const email = `scim-webhook-user-${org.id}@example.com`;

    // create (active) -> user.provisioned
    const created = await (
      await request.post('/scim/v2/Users', {
        headers,
        data: {
          schemas: ['urn:ietf:params:scim:schemas:core:2.0:User'],
          userName: email,
          name: { givenName: 'Katherine', familyName: 'Johnson' },
          emails: [{ value: email, primary: true }],
          active: true,
        },
      })
    ).json();

    // attribute PATCH while still active -> user.scim_updated (not deprovisioned)
    const patchRes = await request.patch(`/scim/v2/Users/${created.id}`, {
      headers,
      data: {
        schemas: ['urn:ietf:params:scim:api:messages:2.0:PatchOp'],
        Operations: [{ op: 'replace', path: 'name.givenName', value: 'Kate' }],
      },
    });
    expect(patchRes.status()).toBe(200);

    // DELETE -> user.deprovisioned
    const deleteRes = await request.delete(`/scim/v2/Users/${created.id}`, { headers });
    expect(deleteRes.status()).toBe(204);

    // Delivery is async (detached goroutine); poll the sink until all three land.
    let events: Record<string, { signature: string; rawBody: string; body: any }> = {};
    await expect
      .poll(
        async () => {
          const res = await request.get(`${WEBHOOK_SINK}/webhook/${encodeURIComponent(email)}`);
          if (res.status() !== 200) return [];
          events = (await res.json()).events;
          return Object.keys(events).sort();
        },
        { timeout: 15000, intervals: [250, 500, 1000] }
      )
      .toEqual(['user.deprovisioned', 'user.provisioned', 'user.scim_updated']);

    // Each delivery: correct event fired, payload carries the right user, and the
    // X-Authorizer-Signature is a valid HMAC-SHA256 of the EXACT received bytes
    // under the client secret (matches deliver()'s hmac.New(sha256.New, ClientSecret)).
    for (const eventName of ['user.provisioned', 'user.scim_updated', 'user.deprovisioned']) {
      const d = events[eventName];
      expect(d, `missing delivery for ${eventName}`).toBeTruthy();
      expect(d.body.event_name).toBe(eventName);
      expect(d.body.user.email).toBe(email);
      const expectedSig = createHmac('sha256', CLIENT_SECRET).update(d.rawBody).digest('hex');
      expect(d.signature, `HMAC mismatch for ${eventName}`).toBe(expectedSig);
    }
  });
});
