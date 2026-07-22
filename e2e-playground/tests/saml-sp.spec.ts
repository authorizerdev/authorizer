// e2e-playground/tests/saml-sp.spec.ts
import { test, expect } from '@playwright/test';
import { URL } from 'node:url';
import { createOrg, createSAMLConnection, deleteSAMLConnectionByEntityID } from '../fixtures/adminClient';
import fs from 'node:fs';
import path from 'node:path';

// Host-reachable-or-container-internal base, same fallback pattern as the
// other specs — this whole suite runs via the containerized `playwright`
// compose service (see docker-compose.yml), where MOCK_SAML_BASE_URL is
// wired to the docker-internal `https://mock-saml-idp:4001`. The fallback
// lets a developer also run it against the published host port. https, not
// http: mock-saml-idp terminates TLS with a self-signed cert specifically so
// this URL passes Authorizer's idp_sso_url validation (admin_org_saml.go
// validateSAMLHTTPSURL rejects non-https outright, with no test bypass like
// OIDC's --test-allow-private-sso-hosts) — see `ignoreHTTPSErrors` below.
const MOCK_SAML_BASE = process.env.MOCK_SAML_BASE_URL || 'https://localhost:4001';

// mock-saml-idp/server.ts hardcodes its own entityID to this literal value
// regardless of how a caller reaches it (it's never derived from the
// request's Host) — so the org's SAML connection must trust this exact
// string, not a value built from MOCK_SAML_BASE. crewjam's ParseXMLResponse
// rejects the assertion (`issuer is not %q`) if IssuerURL doesn't match the
// Issuer the IdP actually signs (internal/http_handlers/saml_sp.go
// buildSAMLServiceProvider -> IDPMetadata.EntityID; crewjam service_provider.go
// checks assertion.Issuer.Value against it).
const MOCK_IDP_ENTITY_ID = 'http://mock-saml-idp:4001/metadata';

const idpCert = fs.readFileSync(path.join(__dirname, '../fixtures/certs/idp-cert.pem'), 'utf8');

test.describe('SAML — SP side', () => {
  // mock-saml-idp's TLS cert is self-signed (see server.ts) — trust it here
  // for both the browser (page) and the request fixture rather than adding
  // it to a real trust store, scoped to just this file's tests.
  test.use({ ignoreHTTPSErrors: true });

  test('SP-initiated login: redirect to IdP, POST-binding assertion back, JIT provisioning', async ({
    page,
    baseURL,
    request,
  }) => {
    const org = await createOrg(`saml-sp-${Date.now()}`);
    // idp_entity_id is globally unique and fixed (see MOCK_IDP_ENTITY_ID above)
    // — unlike the org name, it can't be randomized per run, so a stale row
    // from a prior unclean run (CI retry, or a local run without `down -v`)
    // would otherwise collide here. Clean it up first to make this test safe
    // to re-run against a non-fresh database.
    await deleteSAMLConnectionByEntityID(MOCK_IDP_ENTITY_ID);
    await createSAMLConnection(org.id, {
      name: 'primary-idp',
      idpEntityId: MOCK_IDP_ENTITY_ID,
      idpSsoUrl: `${MOCK_SAML_BASE}/sso`,
      idpCertificate: idpCert,
    });

    const email = `saml-user-${org.id}@example.com`;
    await request.post(`${MOCK_SAML_BASE}/__configure`, {
      data: { email, givenName: 'Grace', familyName: 'Hopper' },
    });

    await page.goto(`/oauth/saml/${org.name}/login?redirect_uri=${encodeURIComponent(baseURL + '/app')}`);

    // The mock IdP's /sso response is a self-submitting POST-binding form
    // (onload -> submit) back to the ACS, which redirects to /app on
    // success. Match loosely on the path (not a baseURL-anchored string):
    // the app's static server 301s the bare "/app" to "/app/", same as
    // tests/oidc-sso-rp.spec.ts's confirmed real redirect behavior.
    await page.waitForURL((url) => /^\/app\/?$/.test(url.pathname), { timeout: 10_000 });
    await expect(page.locator('body')).not.toContainText(/error/i);
    // JIT provisioning: the dashboard renders "Signed in as <email>" for the
    // now-authenticated session (web/app/src/pages/dashboard.tsx).
    await expect(page.getByText(email)).toBeVisible();
  });

  test('malformed SAMLResponse at ACS is rejected', async ({ request, baseURL }) => {
    const org = await createOrg(`saml-sp-abuse-${Date.now()}`);
    // idp_entity_id is stored in a globally-unique column across ALL orgs
    // (internal/service/admin_org_saml.go: GetTrustedIssuerByIssuerURL check),
    // not just this one — so it can't reuse MOCK_IDP_ENTITY_ID, which the
    // first test already registered. This test never validates a real signed
    // assertion (it fails on XML parsing first), so any distinct value works.
    await createSAMLConnection(org.id, {
      name: 'primary-idp',
      idpEntityId: `mock-saml-idp-${org.id}`,
      idpSsoUrl: `${MOCK_SAML_BASE}/sso`,
      idpCertificate: idpCert,
    });

    // A bare, unassociated RelayState would fail context resolution before
    // ever reaching assertion parsing (no pending SP-initiated request ->
    // IdP-initiated path -> rejected with idp_initiated_disabled, since this
    // connection doesn't opt in). Drive a real SP-initiated login start to
    // get a genuine, single-use RelayState bound to a pending AuthnRequest,
    // then feed it a garbage SAMLResponse — this isolates the assertion
    // parsing failure the test is actually about.
    const loginRes = await request.get(
      `/oauth/saml/${org.name}/login?redirect_uri=${encodeURIComponent(baseURL + '/app')}`,
      { maxRedirects: 0 }
    );
    expect(loginRes.status()).toBe(302);
    const location = loginRes.headers()['location'];
    const relayState = new URL(location).searchParams.get('RelayState');
    expect(relayState).toBeTruthy();

    const res = await request.post(`/oauth/saml/${org.name}/acs`, {
      form: { SAMLResponse: Buffer.from('<not-valid-xml>').toString('base64'), RelayState: relayState! },
    });
    expect(res.status()).toBe(400);
    const body = await res.json();
    expect(body.error).toBe('saml_assertion_invalid');
  });

  test('metadata endpoint is well-formed SAML metadata', async ({ request }) => {
    const org = await createOrg(`saml-sp-meta-${Date.now()}`);
    // Same global-uniqueness reasoning as the malformed-response test above —
    // metadata generation never validates a signed assertion either.
    await createSAMLConnection(org.id, {
      name: 'primary-idp',
      idpEntityId: `mock-saml-idp-${org.id}`,
      idpSsoUrl: `${MOCK_SAML_BASE}/sso`,
      idpCertificate: idpCert,
    });

    const res = await request.get(`/oauth/saml/${org.name}/metadata`);
    expect(res.status()).toBe(200);
    expect(res.headers()['content-type']).toContain('samlmetadata+xml');
    const body = await res.text();
    expect(body).toContain('EntityDescriptor');
  });
});
