// e2e-playground/fixtures/graphqlRequest.ts
import { APIRequestContext } from '@playwright/test';

// Shared by any pure-GraphQL (no browser UI) MFA spec that goes through the
// mfa_session cookie flow: tests/totp.spec.ts (Task 24) hit this first, and
// tests/sms-otp.spec.ts (Task 25) needs the exact same thing, so it's
// extracted here rather than copy-pasted a second time. The MFA flow is
// cookie-based (mfa_session / mfa_session_domain, set by login's Set-Cookie
// header and read back by totp_mfa_setup/sms_otp_mfa_setup/verify_otp - see
// internal/cookie/mfa_session.go and internal/service/otp_mfa_setup.go's
// resolveOTPSetupCaller), and neither graphql-request nor Node's fetch
// maintain a cookie jar across separate client.request() calls. Playwright's
// own `request` fixture (APIRequestContext) does maintain a cookie jar
// automatically across calls made through it within one test - confirmed by
// totp.spec.ts - so login's Set-Cookie is replayed on the following
// totp_mfa_setup/sms_otp_mfa_setup/verify_otp calls with no manual Cookie
// passthrough needed.
export async function graphql<T = any>(
  request: APIRequestContext,
  baseURL: string,
  query: string,
  variables: Record<string, unknown>
): Promise<T> {
  const res = await request.post('/graphql', {
    data: { query, variables },
    // CSRF middleware requires Origin/Referer on state-changing requests
    // (internal/http_handlers/csrf.go) - same rationale as the GraphQLClient
    // Origin header in tests/oidc-provider.spec.ts. Playwright's request
    // fixture doesn't send Origin the way a real browser would, so it's set
    // explicitly here. The other CSRF requirement (Content-Type:
    // application/json) is satisfied automatically: passing a plain object
    // as `data` makes Playwright JSON-encode the body and set that header.
    headers: { Origin: baseURL },
  });
  const body = await res.json();
  if (body.errors) {
    throw new Error(`GraphQL error: ${JSON.stringify(body.errors)}`);
  }
  return body.data as T;
}
