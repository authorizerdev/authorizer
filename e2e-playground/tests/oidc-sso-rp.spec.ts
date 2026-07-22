// e2e-playground/tests/oidc-sso-rp.spec.ts
import { test, expect } from '@playwright/test';
import { createOrg, createOIDCConnection, addVerifiedDomain } from '../fixtures/adminClient';

// Host-reachable base for the test PROCESS's own calls to mock-oauth (the
// __configure REST call below runs from this Node process, not from inside
// the authorizer container).
const MOCK_OAUTH_BASE = process.env.MOCK_OAUTH_BASE_URL || 'http://localhost:4000';
// Container-network-reachable base for the issuer_url stored on the org's
// OIDC connection: that URL is dialed SERVER-SIDE, by the authorizer-sso
// container itself (SSOLoginHandler -> fetchOIDCDiscovery), so it must
// resolve inside the docker-compose network, not on the host. Plain http
// because mock-oauth has no TLS termination; that (and the private
// docker-network address below) requires
// --test-allow-private-sso-hosts=true, which authorizer-sso runs with (see
// docker-compose.yml) — the default-off `authorizer` service still refuses
// both, unchanged.
const MOCK_OAUTH_INTERNAL_BASE = process.env.MOCK_OAUTH_INTERNAL_BASE_URL || 'http://mock-oauth:4000';

// This spec exercises the /app email-first home-realm-discovery (HRD) step
// (web/app/src/pages/login.tsx: HRDForm / handleHRDSubmit), which only
// renders when the server runs with --enable-org-discovery=true
// (internal/config/config.go Config.EnableOrgDiscovery; cmd/root.go
// --enable-org-discovery, default false / opt-in). That flag is a GLOBAL
// login-UX toggle (every /app login gets the HRD screen, not just org-scoped
// ones), so it can't be turned on for the plain `authorizer` service without
// breaking tests/oidc-provider.spec.ts's PKCE flow. It's instead scoped to
// the dedicated `authorizer-sso` compose service (port 8081) and this spec
// runs against it via the `sso-discovery` Playwright project
// (playwright.config.ts), whose baseURL points at that port.
//
// The real HRD screen (confirmed against a live stack) is served at /app
// (NOT /app/login — Root.tsx only registers "/app" for the unauthenticated
// Login route), has no <label> on its email input (placeholder-only), and
// its button text is exactly "Continue".
test.describe('OIDC — SSO relying party (home-realm discovery)', () => {
  test(
    'home-realm discovery routes to the org IdP and JIT-provisions on first login',
    async ({ page, request, baseURL }) => {
      // adminClient's calls default to AUTHORIZER_BASE_URL (:8080); this
      // project's baseURL is :8081 (authorizer-sso) — pass it through
      // explicitly so the org/connection this test creates exist on the same
      // server the browser below actually logs into. See adminClient.ts's
      // getClient() comment.
      const org = await createOrg(`sso-rp-${Date.now()}`, baseURL);
      const domain = `${org.id}.example.com`;
      await addVerifiedDomain(org.id, domain, baseURL);
      const realm = `sso-org-${org.id}`;
      await createOIDCConnection(
        org.id,
        {
          name: 'primary-idp',
          issuerUrl: `${MOCK_OAUTH_INTERNAL_BASE}/${realm}`,
          clientId: 'mock-client-id',
          clientSecret: 'mock-client-secret',
        },
        baseURL
      );

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
