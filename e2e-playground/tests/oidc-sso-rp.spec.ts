// e2e-playground/tests/oidc-sso-rp.spec.ts
import { test, expect } from '@playwright/test';
import { createOrg, createOIDCConnection, addVerifiedDomain } from '../fixtures/adminClient';

// Host-reachable base for the test PROCESS's own calls to mock-oauth (the
// __configure REST call below runs from this Node process, not from inside
// the authorizer container).
const MOCK_OAUTH_BASE = process.env.MOCK_OAUTH_BASE_URL || 'http://localhost:4000';
// Container-network-reachable base for the issuer_url stored on the org's
// OIDC connection: that URL is dialed SERVER-SIDE, by the authorizer
// container itself (SSOLoginHandler -> fetchOIDCDiscovery), so it must
// resolve inside the docker-compose network, not on the host. See the
// blocked test below for why this currently can never succeed regardless.
const MOCK_OAUTH_INTERNAL_BASE = process.env.MOCK_OAUTH_INTERNAL_BASE_URL || 'https://mock-oauth:4000';

// This spec exercises the /app email-first home-realm-discovery (HRD) step
// (web/app/src/pages/login.tsx: HRDForm / handleHRDSubmit), which only
// renders when the server runs with --enable-org-discovery=true
// (internal/config/config.go Config.EnableOrgDiscovery; cmd/root.go
// --enable-org-discovery, default false / opt-in). The committed
// e2e-playground/docker-compose.yml does NOT currently pass this flag, so
// running this spec against a plain `docker compose up` will render the
// standard password-first login UI directly (no email/Continue step) and
// both tests below will fail at the very first HRD interaction. Add
// "--enable-org-discovery=true" to the authorizer service's command list in
// docker-compose.yml before running this spec — see the Task 8 report for
// verification detail (confirmed by hitting a live stack with the flag
// injected via a scratch, non-committed compose override).
//
// The real HRD screen (confirmed against a live stack) is served at /app
// (NOT /app/login — Root.tsx only registers "/app" for the unauthenticated
// Login route), has no <label> on its email input (placeholder-only), and
// its button text is exactly "Continue".
test.describe('OIDC — SSO relying party (home-realm discovery)', () => {
  // BLOCKED: cannot be made to pass against this docker-compose stack. Two
  // independent, unconditional server-side checks stand in the way, neither
  // of which has a test/dev bypass (verified live, not by inspection alone):
  //
  // 1. internal/service/admin_org_oidc.go validateSSOIssuerURL() rejects any
  //    issuer_url whose scheme isn't exactly "https" -- confirmed: creating a
  //    connection with issuer_url="http://mock-oauth:4000/<realm>" (mock-oauth
  //    only ever serves plain HTTP) fails immediately with GraphQL error
  //    "issuer_url must be a valid https URL". No Env/test flag relaxes this
  //    (contrast internal/config/config.go SkipTestEndpointSSRFValidation,
  //    which exists ONLY for the admin webhook TestEndpoint mutation).
  //
  // 2. Even switching the scheme to "https" (which passes step 1, since it
  //    only checks scheme+host, not reachability) does not help: at login
  //    time SSOLoginHandler -> fetchOIDCDiscovery calls
  //    validators.SafeHTTPClient, which resolves the host and unconditionally
  //    rejects any private/loopback/internal IP. mock-oauth's docker-compose
  //    network address is unavoidably private (confirmed via `docker exec
  //    <authorizer> getent hosts mock-oauth` -> 172.25.0.2, inside
  //    172.16.0.0/12). Driving a real browser through the HRD flow with such
  //    a connection configured redirects to
  //    /oauth/sso/<org>/login?... and lands on the literal JSON body
  //    {"error":"sso_upstream_error","error_description":"could not reach the
  //    identity provider"} -- confirmed live.
  //
  // Together these mean no mock IdP reachable from inside this
  // docker-compose network (which is all of them: mock-oauth, mock-saml-idp,
  // etc. all sit on the same private bridge) can ever be registered+dialed
  // as an org OIDC connection in this stack. Closing this gap needs either a
  // test-mode SSRF bypass for the SSO broker's outbound fetches (mirroring
  // SkipTestEndpointSSRFValidation, scoped to fetchOIDCDiscovery /
  // exchangeSSOCode / fetchSSOJWKS) or a TLS-terminated, non-private-IP mock
  // IdP -- both are product/infra decisions outside a single test-spec task.
  // The body below is the intended test, kept ready to enable once one of
  // those lands.
  test.fixme(
    'home-realm discovery routes to the org IdP and JIT-provisions on first login',
    async ({ page, request }) => {
      const org = await createOrg(`sso-rp-${Date.now()}`);
      const domain = `${org.id}.example.com`;
      await addVerifiedDomain(org.id, domain);
      const realm = `sso-org-${org.id}`;
      await createOIDCConnection(org.id, {
        name: 'primary-idp',
        issuerUrl: `${MOCK_OAUTH_INTERNAL_BASE}/${realm}`,
        clientId: 'mock-client-id',
        clientSecret: 'mock-client-secret',
      });

      const employeeEmail = `employee@${domain}`;
      await request.post(`${MOCK_OAUTH_BASE}/${realm}/__configure`, {
        data: { profile: { sub: 'employee-1', email: employeeEmail, given_name: 'Ada', family_name: 'Lovelace' } },
      });

      await page.goto('/app');
      await page.getByPlaceholder('Enter your email').fill(employeeEmail);
      await page.getByRole('button', { name: 'Continue' }).click();

      // Home-realm discovery should have redirected through mock-oauth back
      // to Authorizer with a session, landing back on /app.
      await page.waitForURL((url) => url.pathname === '/app' && !url.pathname.includes('error'), { timeout: 10_000 });
      await expect(page.locator('body')).not.toContainText(/error/i);
    }
  );

  test('unrecognized domain falls back to standard login without leaking org existence', async ({ page }) => {
    await page.goto('/app');
    await page.getByPlaceholder('Enter your email').fill('someone@totally-unrecognized-domain.example');
    await page.getByRole('button', { name: 'Continue' }).click();

    // Falls through to the standard password field (id confirmed in
    // web/app/src/pages/login.tsx -> AuthorizerBasicAuthLogin, same as
    // oidc-provider.spec.ts) rather than any org-specific redirect or error.
    await expect(page.locator('#authorizer-login-password')).toBeVisible({ timeout: 10_000 });
  });
});
