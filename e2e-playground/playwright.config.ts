import { defineConfig, devices } from '@playwright/test';

export default defineConfig({
  testDir: '.',
  testIgnore: ['**/node_modules/**', '**/mocks/**'],
  timeout: 30_000,
  retries: 0,
  reporter: [['html', { outputFolder: 'playwright-report', open: 'never' }]],
  use: {
    baseURL: process.env.AUTHORIZER_BASE_URL || 'http://localhost:8080',
    trace: 'retain-on-failure',
    screenshot: 'only-on-failure',
    video: 'retain-on-failure',
  },
  projects: [
    {
      name: 'mfa-off',
      testIgnore: [
        /mfa-routing-matrix\.spec\.ts/,
        /oidc-sso-rp\.spec\.ts/,
        /sso-discovery\.spec\.ts/,
        /webauthn\.spec\.ts/,
        /magic-link\.spec\.ts/,
        '**/mocks/**',
        '**/node_modules/**',
      ],
      use: { ...devices['Desktop Chrome'] },
    },
    {
      // Runs against authorizer-mfa-enforced (docker-compose.yml), the only
      // service with --enforce-mfa=true - see that service's comment in
      // docker-compose.yml for why EnforceMFA can't be toggled at runtime
      // on the shared `authorizer` service (the _update_env mutation
      // fixtures/adminClient.ts's setEnforceMFA calls is a stub that always
      // errors - "deprecated. please configure env via cli args"). The one
      // test in this spec needing magic-link login too talks to a second
      // dedicated instance, authorizer-mfa-magic-link, via an explicit
      // absolute URL rather than this project's baseURL.
      name: 'mfa-on',
      testMatch: /mfa-routing-matrix\.spec\.ts/,
      use: {
        ...devices['Desktop Chrome'],
        baseURL: process.env.AUTHORIZER_MFA_ENFORCED_BASE_URL || 'http://localhost:8084',
      },
    },
    {
      // Runs against authorizer-sso (docker-compose.yml), the only service
      // with --enable-org-discovery=true. That flag is a global login-UX
      // toggle, so it can't be turned on for the `mfa-off` project's service
      // without breaking tests/oidc-provider.spec.ts's plain PKCE flow — see
      // that service's comment in docker-compose.yml.
      name: 'sso-discovery',
      testMatch: [/oidc-sso-rp\.spec\.ts/, /sso-discovery\.spec\.ts/],
      use: {
        ...devices['Desktop Chrome'],
        baseURL: process.env.AUTHORIZER_SSO_BASE_URL || 'http://localhost:8081',
      },
    },
    {
      // Runs against authorizer-webauthn (docker-compose.yml), the only
      // service configured with a dotted --url hostname - required for
      // go-webauthn's RPID validation to accept it at all (see that
      // service's comment in docker-compose.yml). Can't share the `authorizer`
      // service's single-label hostname the way most other specs do.
      name: 'webauthn',
      testMatch: /webauthn\.spec\.ts/,
      use: {
        ...devices['Desktop Chrome'],
        baseURL: process.env.AUTHORIZER_WEBAUTHN_BASE_URL || 'http://localhost:8082',
      },
    },
    {
      // Runs against authorizer-magic-link (docker-compose.yml), the only
      // service with --enable-magic-link-login=true AND
      // --enable-email-verification=true - see that service's comment in
      // docker-compose.yml for why those can't live on the shared
      // `authorizer` service.
      name: 'magic-link',
      testMatch: /magic-link\.spec\.ts/,
      use: {
        ...devices['Desktop Chrome'],
        baseURL: process.env.AUTHORIZER_MAGIC_LINK_BASE_URL || 'http://localhost:8083',
      },
    },
  ],
});
