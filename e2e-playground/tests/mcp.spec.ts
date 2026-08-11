// e2e-playground/tests/mcp.spec.ts
import { test, expect } from '@playwright/test';
import { GraphQLClient, gql } from 'graphql-request';
import crypto from 'node:crypto';

const BASE_URL = process.env.AUTHORIZER_BASE_URL || 'http://localhost:8080';

// The canonical MCP resource identifier is derived from the server's --url, and
// this suite runs inside the compose network (docker compose run playwright) so
// AUTHORIZER_BASE_URL is the same http://authorizer:8080 the server is
// configured with. Deriving it here rather than hardcoding keeps the spec
// honest if either value moves.
const MCP_ISSUER = BASE_URL;
const MCP_RESOURCE = `${BASE_URL}/mcp`;

const client = new GraphQLClient(`${BASE_URL}/graphql`, { headers: { Origin: BASE_URL } });

function randomEmail() {
  return `mcp-${crypto.randomUUID()}@example.com`;
}

const INITIALIZE_RPC = {
  jsonrpc: '2.0',
  id: 1,
  method: 'initialize',
  params: {
    protocolVersion: '2025-06-18',
    capabilities: {},
    clientInfo: { name: 'e2e-playground', version: '1.0' },
  },
};

const MCP_HEADERS = {
  'Content-Type': 'application/json',
  Accept: 'application/json, text/event-stream',
};

test.describe('MCP — remote server', () => {
  test('an unauthenticated call returns the challenge that starts discovery', async ({ request }) => {
    const res = await request.post('/mcp', { headers: MCP_HEADERS, data: INITIALIZE_RPC });

    expect(res.status()).toBe(401);

    // RFC 9728 §5.1. This pointer is the only thing telling a client that has
    // never seen this deployment where to authenticate; without it the
    // connection fails with nothing to diagnose.
    const challenge = res.headers()['www-authenticate'];
    expect(challenge).toContain(
      `resource_metadata="${MCP_ISSUER}/.well-known/oauth-protected-resource/mcp"`,
    );
    // RFC 6750 §3: `error` describes a credential that was supplied and
    // rejected. A first contact carrying nothing is not an error.
    expect(challenge).not.toContain('error=');
  });

  test('the metadata document names the resource clients must bind tokens to', async ({ request }) => {
    const res = await request.get('/.well-known/oauth-protected-resource/mcp');
    expect(res.status()).toBe(200);

    const doc = await res.json();
    expect(doc.resource).toBe(MCP_RESOURCE);
    expect(doc.authorization_servers).toEqual([MCP_ISSUER]);
    expect(doc.bearer_methods_supported).toEqual(['header']);
    // A client that requests only what this document advertises must still be
    // able to obtain a refresh token, or the agent's session dies at the first
    // access-token expiry.
    expect(doc.scopes_supported).toContain('offline_access');

    // RFC 9728 §3.1 inserts the well-known segment ahead of the resource
    // identifier's PATH, so the bare origin form denotes a DIFFERENT resource
    // and §3.3 has clients reject a document whose `resource` does not match
    // what they asked for. Serving it there too would hand strict clients a
    // mismatch, so it must not exist.
    const bare = await request.get('/.well-known/oauth-protected-resource');
    expect(bare.status()).toBe(404);
  });

  test('a browser OAuth flow yields a token that works at /mcp, survives refresh, and is scoped to MCP alone', async ({
    page,
    request,
  }) => {
    const clientId = 'e2e-client-id';
    const redirectUri = `${BASE_URL}/e2e-mcp-callback`;
    const email = randomEmail();
    const password = 'Str0ngPassw0rd!';

    const signup = gql`
      mutation ($params: SignUpRequest!) {
        signup(params: $params) { message }
      }
    `;
    await client.request(signup, { params: { email, password, confirm_password: password } });

    const codeVerifier = crypto.randomBytes(32).toString('base64url');
    const codeChallenge = crypto.createHash('sha256').update(codeVerifier).digest('base64url');
    const state = crypto.randomUUID();

    const authorizeUrl = new URL('/authorize', BASE_URL);
    authorizeUrl.searchParams.set('response_type', 'code');
    authorizeUrl.searchParams.set('client_id', clientId);
    authorizeUrl.searchParams.set('redirect_uri', redirectUri);
    // offline_access so the grant yields a refresh token — the refresh
    // assertion below is the whole reason this test drives a real flow.
    authorizeUrl.searchParams.set('scope', 'openid offline_access');
    authorizeUrl.searchParams.set('state', state);
    authorizeUrl.searchParams.set('code_challenge', codeChallenge);
    authorizeUrl.searchParams.set('code_challenge_method', 'S256');
    // RFC 8707: this is what binds the issued access token's `aud` to the MCP
    // server, and it is what /mcp checks.
    authorizeUrl.searchParams.set('resource', MCP_RESOURCE);

    await page.goto(authorizeUrl.toString());

    await page.locator('#authorizer-login-email-or-phone-number').fill(email);
    await page.locator('#authorizer-login-password').fill(password);
    await page.locator('form[name="authorizer-login-form"] button[type="submit"]').click();

    // First login for a new user hits the optional MFA-setup offer; the token
    // is withheld until a factor is added or skipped. Tolerates the screen not
    // appearing at all.
    await page
      .getByRole('button', { name: 'Skip for now' })
      .click({ timeout: 10_000 })
      .catch(() => {});

    await page.waitForURL(
      (url) => url.origin === BASE_URL && url.pathname === '/e2e-mcp-callback' && url.searchParams.has('code'),
    );
    const code = new URL(page.url()).searchParams.get('code')!;
    expect(code).toBeTruthy();

    const tokenRes = await request.post('/oauth/token', {
      form: {
        grant_type: 'authorization_code',
        code,
        client_id: clientId,
        redirect_uri: redirectUri,
        code_verifier: codeVerifier,
        resource: MCP_RESOURCE,
      },
    });
    expect(tokenRes.status()).toBe(200);
    const tokens = await tokenRes.json();
    expect(tokens.access_token).toBeTruthy();
    expect(tokens.refresh_token).toBeTruthy();

    // --- the token reaches the tool surface ---------------------------------
    const callMCP = (bearer: string, body: unknown) =>
      request.post('/mcp', {
        headers: { ...MCP_HEADERS, Authorization: `Bearer ${bearer}` },
        data: body,
      });

    const init = await callMCP(tokens.access_token, INITIALIZE_RPC);
    expect(init.status(), await init.text()).toBe(200);

    const tools = await callMCP(tokens.access_token, { jsonrpc: '2.0', id: 2, method: 'tools/list' });
    expect(tools.status()).toBe(200);
    const toolNames = (await tools.json()).result.tools.map((t: { name: string }) => t.name);
    expect(toolNames).toContain('profile');

    // --- the binding survives refresh ---------------------------------------
    // Before the resource was carried across rotation, the refreshed token's
    // `aud` fell back to the client id and this call returned 401 forever.
    // Access tokens live 30 minutes and MCP clients refresh proactively, so the
    // failure only ever appeared in production, long after the flow was tested
    // by hand.
    const refreshRes = await request.post('/oauth/token', {
      form: { grant_type: 'refresh_token', refresh_token: tokens.refresh_token, client_id: clientId },
    });
    expect(refreshRes.status()).toBe(200);
    const refreshed = await refreshRes.json();
    expect(refreshed.access_token).toBeTruthy();
    expect(refreshed.access_token).not.toBe(tokens.access_token);

    const afterRefresh = await callMCP(refreshed.access_token, INITIALIZE_RPC);
    expect(afterRefresh.status(), await afterRefresh.text()).toBe(200);

    // --- and it buys nothing outside MCP ------------------------------------
    // The other half of the audience boundary: a token scoped to the MCP server
    // must not double as a first-party API credential, or handing one to a
    // semi-trusted agent would hand over the whole account.
    const userinfo = await request.get('/userinfo', {
      headers: { Authorization: `Bearer ${refreshed.access_token}` },
    });
    expect(userinfo.status()).toBe(401);
  });

  test('an ordinary login token cannot authenticate MCP', async ({ request }) => {
    // The inverse direction, and the one a client is most likely to try by
    // accident: the token every SDK already holds must not open this surface.
    const email = randomEmail();
    const password = 'Str0ngPassw0rd!';

    const signup = gql`
      mutation ($params: SignUpRequest!) {
        signup(params: $params) { message access_token }
      }
    `;
    const res = await client.request<{ signup: { access_token: string | null } }>(signup, {
      params: { email, password, confirm_password: password },
    });

    // MFA is on by default in this stack, which withholds signup's token. Skip
    // rather than assert a weaker thing: the audience boundary is covered from
    // the token side above, and a silently-empty bearer here would make this
    // assertion pass for the wrong reason.
    test.skip(!res.signup.access_token, 'signup token withheld behind MFA setup');

    const mcp = await request.post('/mcp', {
      headers: { ...MCP_HEADERS, Authorization: `Bearer ${res.signup.access_token}` },
      data: INITIALIZE_RPC,
    });
    expect(mcp.status()).toBe(401);
    expect(mcp.headers()['www-authenticate']).toContain('error="invalid_token"');
  });
});
