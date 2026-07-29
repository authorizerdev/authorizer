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
  const res = await request.post(`${baseURL}/graphql`, {
    // Absolute URL, not a relative '/graphql' path: request.post resolves a
    // relative path against the `request` fixture's own configured baseURL
    // (the Playwright project's use.baseURL), silently ignoring this
    // function's `baseURL` argument for anything but the Origin header
    // below - a real bug found in Task 28 (mfa-routing-matrix.spec.ts),
    // which needs to target a second instance
    // (authorizer-mfa-magic-link) from within the `mfa-on` project
    // (baseURL authorizer-mfa-enforced). Every caller up to that point
    // happened to pass the same baseURL as their project's, so a relative
    // path resolved to the right host by coincidence and this never
    // surfaced. Worse, the wrong-host request was rejected by CSRF
    // (Origin not in that host's --allowed-origins) with a non-GraphQL-shaped
    // body ({error, error_description}, no `errors` array) that the check
    // below didn't catch either - silently returning undefined instead of
    // failing loudly. See the `body.error` check below for that half of the
    // fix.
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
  if (body.error) {
    // Non-GraphQL-shaped error (e.g. CSRF middleware's {error,
    // error_description}, returned before the request ever reaches the
    // GraphQL handler) - not caught by the body.errors check below.
    throw new Error(`Request error: ${JSON.stringify(body)}`);
  }
  if (body.errors) {
    throw new Error(`GraphQL error: ${JSON.stringify(body.errors)}`);
  }
  return body.data as T;
}
