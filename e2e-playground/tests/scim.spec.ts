// e2e-playground/tests/scim.spec.ts
import { test, expect } from '@playwright/test';
import { createOrg, createSCIMEndpoint, getUserPhoneNumberByEmail } from '../fixtures/adminClient';

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
// scim_updated, internal/constants/webhook_event_scim.go) are fired correctly
// from internal/service/scim/scim.go — confirmed by reading the source. But
// there is no way to exercise them end-to-end here, or via any real customer
// flow today, because of a real product bug found while building this test:
//
//   internal/validators/webhook.go's IsValidWebhookEventName allow-list was
//   never updated for the six SCIM webhook events added alongside this
//   feature. `_add_webhook` (internal/service/admin_webhooks.go) rejects
//   event_name: "user.provisioned" (and deprovisioned/scim_updated/group.*)
//   with "invalid event name user.provisioned" — verified live against this
//   stack's running authorizer container. No admin can register a webhook
//   for any SCIM lifecycle event via the only supported API, so the whole
//   feature is currently unreachable.
//
// Even with that fixed, a live delivery test in this docker-compose stack
// would hit a second, structural wall: webhook delivery
// (internal/events/events.go's deliver()) always calls
// validators.SafeHTTPClient, which unconditionally rejects private-network
// IPs with no override (unlike the SSO broker's
// SafeHTTPClientAllowPrivate/--test-allow-private-sso-hosts) — and every
// e2e-playground service, including a hypothetical webhook-sink mock, only
// has a private docker-network address. Running the authorizer with --env
// test would skip that check, but internal/events/events.go's deliver() also
// short-circuits ALL real HTTP delivery when Env == TestEnv (it just logs a
// fake 200 "test" response) — so there is no configuration of this stack that
// both allows registering a private endpoint and actually sends the HTTP
// request to it.
//
// Filed as a real bug rather than routed around; not building a webhook-sink
// mock until IsValidWebhookEventName is fixed and (separately) a private-host
// delivery escape hatch exists for e2e use, analogous to the SSO one.
test.describe('SCIM provisioning webhooks', () => {
  test('provisioning webhook delivery (user.provisioned / user.deprovisioned / user.scim_updated)', async () => {
    test.skip(
      true,
      'blocked by a real bug: IsValidWebhookEventName (internal/validators/webhook.go) rejects ' +
        'user.provisioned/deprovisioned/scim_updated, so _add_webhook cannot register a listener for ' +
        'any SCIM lifecycle event today — see the comment above this describe block for details and ' +
        'the separate SSRF/test-env constraint that would also block a live delivery test even once ' +
        'that is fixed.'
    );
  });
});
