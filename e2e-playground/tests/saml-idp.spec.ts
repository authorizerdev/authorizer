// e2e-playground/tests/saml-idp.spec.ts
import { test, expect } from '@playwright/test';
import crypto from 'node:crypto';
import {
  createOrg,
  signupUser,
  getUserIdByEmail,
  verifyUserEmail,
  addOrgMember,
  createSAMLServiceProvider,
} from '../fixtures/adminClient';

// Same host-reachable-or-container-internal fallback pattern as the other
// SAML/OIDC specs. AUTHORIZER_BASE_URL is wired to the docker-internal
// `http://authorizer:8080` for the containerized `playwright` run
// (docker-compose.yml); mock-saml-idp's fake-SP endpoints (below) need this
// passed explicitly since they run in a separate Node process that doesn't
// see Playwright's own baseURL config.
const AUTHORIZER_BASE = process.env.AUTHORIZER_BASE_URL || 'http://localhost:8080';
// https, not http: mock-saml-idp terminates TLS with a self-signed cert (see
// its server.ts) purely as a stand-in external SP host — nothing here
// requires https, but reusing the one mock keeps this file simple instead of
// standing up a second mock service. `ignoreHTTPSErrors` below trusts it.
const MOCK_SAML_BASE = process.env.MOCK_SAML_BASE_URL || 'https://localhost:4001';

test.describe('SAML — IdP side', () => {
  test.use({ ignoreHTTPSErrors: true });

  test('SP-initiated login: external SP gets a signed assertion posted to its ACS URL', async ({
    page,
    request,
  }) => {
    const org = await createOrg(`saml-idp-${Date.now()}`);
    const email = `saml-idp-user-${org.id}@example.com`;
    const password = 'Str0ngPassw0rd!';
    await signupUser(email, password);
    const userId = await getUserIdByEmail(email);
    // authorizeSAMLIssuance (internal/http_handlers/saml_idp.go) refuses to
    // assert an unverified email as the Subject NameID — force-verify it
    // rather than looping this test through a real verification email.
    await verifyUserEmail(userId, { givenName: 'Grace', familyName: 'Hopper' });
    // Org membership is the other half of authorizeSAMLIssuance's guard:
    // only a member of the org an SP belongs to can receive an assertion
    // for it.
    await addOrgMember(org.id, userId);

    // Register this run's fake SP as a real downstream SP on the org (the
    // inverse of createSAMLConnection from the SP-side spec): entity_id and
    // acs_url must match exactly what mock-saml-idp's fake-SP role builds
    // into the AuthnRequest below, since crewjam's samlSPRegistry resolves
    // the SP by entity_id and validates the ACS URL against this record —
    // never against anything the request itself supplies.
    const entityId = `fake-sp-${org.id}`;
    const acsUrl = `${MOCK_SAML_BASE}/fake-sp/acs`;
    await createSAMLServiceProvider(org.id, { name: 'fake-sp', entityId, acsUrl });

    // relayState is this test's own correlation key into mock-saml-idp's
    // per-flow state (keyed, not a single shared slot, so concurrent runs
    // against the same long-lived mock container don't collide).
    const relayState = crypto.randomUUID();
    const startUrl =
      `${MOCK_SAML_BASE}/fake-sp/start?` +
      `authorizer_base=${encodeURIComponent(AUTHORIZER_BASE)}&org=${encodeURIComponent(org.name)}` +
      `&entity_id=${encodeURIComponent(entityId)}&acs_url=${encodeURIComponent(acsUrl)}` +
      `&relay_state=${relayState}`;

    // mock-saml-idp's /fake-sp/start builds a REAL AuthnRequest (samlify SP
    // role, HTTP-Redirect binding) and 302s the browser straight to
    // Authorizer's actual SP-initiated SSO route
    // (GET /saml/idp/:org_slug/sso). This is a fresh user with no
    // Authorizer session yet, so Authorizer's SAMLIDPSSOHandler bounces the
    // browser to its own /app login UI (bounceSAMLIDPToLogin), stashing the
    // pending AuthnRequest server-side.
    await page.goto(startUrl);

    await page.locator('#authorizer-login-email-or-phone-number').fill(email);
    await page.locator('#authorizer-login-password').fill(password);
    await page.locator('form[name="authorizer-login-form"] button[type="submit"]').click();
    // First-time login for a brand-new user hits the optional MFA-setup
    // offer screen (withheld-token first-time-setup redesign, commit
    // 992b3bf4) — same tolerant skip as tests/oidc-provider.spec.ts's PKCE
    // test; click() itself already waits, so this also tolerates the offer
    // screen never appearing.
    await page
      .getByRole('button', { name: 'Skip for now' })
      .click({ timeout: 5_000 })
      .catch(() => undefined);

    // Login resumes the stashed SAML flow (?saml_continue=), which emits a
    // real signed assertion and auto-submits it (crewjam's WriteResponse) to
    // our fake SP's ACS.
    await page.waitForURL(/\/fake-sp\/acs(\?|$)/, { timeout: 10_000 });

    const last = await request.get(`${MOCK_SAML_BASE}/fake-sp/last?relay_state=${relayState}`);
    expect(last.status()).toBe(200);
    const result = await last.json();
    // Default NameID format is emailAddress with a verified email
    // (buildSAMLSession / samlNameIDWouldBeEmail) — the Subject really is
    // this user, not a placeholder.
    expect(result.nameID).toBe(email);
    // Audience isolation: the assertion is scoped to THIS SP's entity id.
    expect(result.audience).toBe(entityId);
    // Issuer is Authorizer's real, per-org IdP entity id (buildIDPMetadata /
    // samlIDPEntityID) — proves the signature samlify verified really came
    // from this org's Authorizer IdP key, not a self-signed stand-in.
    expect(result.issuer).toBe(`${AUTHORIZER_BASE}/saml/idp/${org.name}/metadata`);
    // Mapped attributes (buildMappedAttributes -> samlDefaultAttributeMapping)
    // carry the verified email and name under Authorizer's default SAML
    // attribute names ("email", "firstName", "lastName").
    expect(result.attributes.email).toBe(email);
    expect(result.attributes.firstName).toBe('Grace');
    expect(result.attributes.lastName).toBe('Hopper');
  });

  test('unregistered SP requesting an assertion is rejected', async ({ request }) => {
    const org = await createOrg(`saml-idp-abuse-${Date.now()}`);
    const relayState = crypto.randomUUID();
    // entity_id here is deliberately never registered via
    // _create_saml_service_provider for this org.
    const startUrl =
      `${MOCK_SAML_BASE}/fake-sp/start?` +
      `authorizer_base=${encodeURIComponent(AUTHORIZER_BASE)}&org=${encodeURIComponent(org.name)}` +
      `&entity_id=${encodeURIComponent('never-registered-sp')}` +
      `&acs_url=${encodeURIComponent('https://not-registered.example.com/acs')}&relay_state=${relayState}`;

    // The SP-resolution check (crewjam's IdpAuthnRequest.Validate ->
    // samlSPRegistry.GetServiceProvider) runs before any authentication
    // check, so an unauthenticated request is enough to exercise it —
    // follow /fake-sp/start's redirect by hand rather than letting a
    // browser bounce through the login UI, since this rejection never gets
    // that far.
    const startRes = await request.get(startUrl, { maxRedirects: 0 });
    expect(startRes.status()).toBe(302);
    const ssoUrl = startRes.headers()['location'];
    expect(ssoUrl).toBeTruthy();

    const ssoRes = await request.get(ssoUrl!);
    expect(ssoRes.status()).toBe(400);
    const body = await ssoRes.json();
    expect(body.error).toBe('invalid_authn_request');
  });
});
