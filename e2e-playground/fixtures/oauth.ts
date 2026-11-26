// e2e-playground/fixtures/oauth.ts
import { APIRequestContext, Page } from '@playwright/test';
import crypto from 'node:crypto';

// Every instance in docker-compose.yml runs --client-id=e2e-client-id.
export const CLIENT_ID = 'e2e-client-id';

export type AuthorizeRequest = {
  url: string;
  state: string;
  verifier: string;
  redirectUri: string;
};

// newAuthorizeRequest builds a fresh RFC 7636 (S256) authorization-code
// request against `baseURL`, the same shape tests/oidc-provider.spec.ts builds
// inline.
export function newAuthorizeRequest(baseURL: string, redirectUri: string): AuthorizeRequest {
  const verifier = crypto.randomBytes(32).toString('base64url');
  const challenge = crypto.createHash('sha256').update(verifier).digest('base64url');
  const state = crypto.randomUUID();
  const u = new URL('/authorize', baseURL);
  u.searchParams.set('response_type', 'code');
  u.searchParams.set('client_id', CLIENT_ID);
  u.searchParams.set('redirect_uri', redirectUri);
  u.searchParams.set('scope', 'openid profile email');
  u.searchParams.set('state', state);
  u.searchParams.set('code_challenge', challenge);
  u.searchParams.set('code_challenge_method', 'S256');
  return { url: u.toString(), state, verifier, redirectUri };
}

// stubCallback answers every request to the RP callback path on `baseURL`
// with a tiny page, so the browser can land on it and the test can read the
// query the authorization server put there. The callback lives on the
// authorizer's own origin only because that origin is already in
// --allowed-origins (and, where configured, --redirect-uris); nothing on the
// server serves it.
export async function stubCallback(page: Page, baseURL: string, pathname: string): Promise<void> {
  const origin = new URL(baseURL).origin;
  await page.context().route(
    (url) => url.origin === origin && url.pathname === pathname,
    (route) => route.fulfill({ status: 200, contentType: 'text/html', body: '<!doctype html><title>rp callback</title>' })
  );
}

// redeemCode posts the authorization code to /oauth/token.
export async function redeemCode(
  request: APIRequestContext,
  baseURL: string,
  code: string,
  req: Pick<AuthorizeRequest, 'verifier' | 'redirectUri'>
) {
  return request.post(`${baseURL}/oauth/token`, {
    form: {
      grant_type: 'authorization_code',
      code,
      client_id: CLIENT_ID,
      redirect_uri: req.redirectUri,
      code_verifier: req.verifier,
    },
  });
}

// jwtClaims decodes a JWT's payload without verifying it (the tests only read
// claims the server just issued over a trusted connection).
export function jwtClaims(jwt: string): Record<string, unknown> {
  return JSON.parse(Buffer.from(jwt.split('.')[1], 'base64url').toString('utf8'));
}
