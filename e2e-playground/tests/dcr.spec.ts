// e2e-playground/tests/dcr.spec.ts
//
// RFC 7591 dynamic client registration: a client with no prior relationship
// POSTs its own metadata and gets a client_id back, then runs a normal OAuth
// flow with it. CIMD is the preferred mechanism (see cimd.spec.ts) — this path
// exists for clients that predate it, which today includes Claude Code.
//
// Two properties can ONLY be proven here, not in the Go tests:
//
//   1. The registration POST survives the real middleware chain. It carries no
//      Origin and no Referer, because it comes from a CLI rather than a page,
//      and the CSRF middleware rejects exactly that shape unless the endpoint is
//      exempted. Every Go test calls the handler directly and so never sees the
//      middleware — which is how the equivalent bug reached the consent form.
//   2. A real user is shown the consent screen for a self-registered client,
//      rendered by the real template after a real login, and approving it yields
//      a code for the registered redirect. The Go tests assert the handler's
//      rules against a fabricated request; only this proves the page a person
//      actually sees carries the client's name, the redirect host, the warning
//      and a working form.
//
// The final hop into the client's own callback is asserted from the redirect the
// server issues rather than by letting the browser follow it — see the note at
// the approval step for why Chromium refuses that hop in this topology.
//
// The callback is a loopback URI rather than a path on the authorizer, and that
// is forced rather than chosen: the server refuses to register a plain-http
// redirect_uri unless it is loopback (MCP requires redirect URIs to be localhost
// or https), so http://authorizer:8080/... — the same-origin trick mcp.spec.ts
// uses with a pre-registered client — is correctly rejected for a DCR client.
// Loopback is also what a real MCP client binds.
import { test, expect } from "@playwright/test";
import { GraphQLClient, gql } from "graphql-request";
import crypto from "node:crypto";

const BASE_URL = process.env.AUTHORIZER_BASE_URL || "http://localhost:8080";
const MCP_RESOURCE = `${BASE_URL}/mcp`;

const client = new GraphQLClient(`${BASE_URL}/graphql`, {
  headers: { Origin: BASE_URL },
});

const MCP_HEADERS = {
  "Content-Type": "application/json",
  Accept: "application/json, text/event-stream",
};

const INITIALIZE_RPC = {
  jsonrpc: "2.0",
  id: 1,
  method: "initialize",
  params: {
    protocolVersion: "2025-06-18",
    capabilities: {},
    clientInfo: { name: "e2e-dcr", version: "1.0" },
  },
};

// The loopback callback this client registers. Nothing listens on it, and
// nothing needs to: the flow below asserts the redirect the SERVER issues rather
// than following it into the client's process (see the note at the approval
// step). A high, arbitrary port is deliberate — an MCP client binds an ephemeral
// one, which is precisely the case the server's port-agnostic loopback matching
// (RFC 8252 §7.3) exists for, and which no operator could allow-list in advance.
const redirectURI = "http://127.0.0.1:47821/callback";

async function register(
  request: import("@playwright/test").APIRequestContext,
  body: unknown,
) {
  // No Origin, no Referer, no cookies — the shape a CLI sends. If the CSRF
  // exemption regresses this returns 403 and every case below fails loudly.
  return request.post("/oauth/register", {
    headers: { "Content-Type": "application/json" },
    data: body,
  });
}

test.describe("DCR — dynamically registered clients", () => {
  test("the authorization server advertises the registration endpoint", async ({
    request,
  }) => {
    // This exact field is what a client looks for before it will attempt DCR;
    // its absence is the "Incompatible auth server: does not support dynamic
    // client registration" refusal that made the MCP surface unreachable.
    const res = await request.get("/.well-known/oauth-authorization-server");
    expect(res.status()).toBe(200);
    const doc = await res.json();
    expect(doc.registration_endpoint).toBe(`${BASE_URL}/oauth/register`);
    // Both self-registration mechanisms are offered; a client that supports
    // CIMD picks it first and never reaches the DCR path.
    expect(doc.client_id_metadata_document_supported).toBe(true);
    expect(doc.token_endpoint_auth_methods_supported).toContain("none");
  });

  test("registration succeeds without an Origin header and yields a public client", async ({
    request,
  }) => {
    const res = await register(request, {
      client_name: "E2E DCR Client",
      redirect_uris: [redirectURI],
      grant_types: ["authorization_code", "refresh_token"],
      token_endpoint_auth_method: "none",
    });
    expect(res.status(), await res.text()).toBe(201);

    const body = await res.json();
    expect(body.client_id).toBeTruthy();
    expect(body.token_endpoint_auth_method).toBe("none");
    expect(body.redirect_uris).toEqual([redirectURI]);
    // No secret is issued: an anonymous caller must never be able to create a
    // confidential client.
    expect(body).not.toHaveProperty("client_secret");
    expect(body).not.toHaveProperty("client_secret_expires_at");
  });

  test("a confidential registration is refused", async ({ request }) => {
    const res = await register(request, {
      client_name: "Wants A Secret",
      redirect_uris: [redirectURI],
      token_endpoint_auth_method: "client_secret_basic",
    });
    expect(res.status()).toBe(400);
    expect((await res.json()).error).toBe("invalid_client_metadata");
  });

  test("a non-loopback http redirect is refused", async ({ request }) => {
    // MCP: "All redirect URIs MUST be either localhost or use HTTPS." Without
    // this every code issued to the client would cross the network in the clear.
    const res = await register(request, {
      client_name: "Cleartext",
      redirect_uris: ["http://app.example.com/cb"],
    });
    expect(res.status()).toBe(400);
    expect((await res.json()).error).toBe("invalid_redirect_uri");
  });

  test("a self-registered client is consented to, then reaches the MCP surface", async ({
    page,
    request,
  }) => {
    const regRes = await register(request, {
      client_name: "E2E DCR Browser Client",
      redirect_uris: [redirectURI],
      grant_types: ["authorization_code", "refresh_token"],
      token_endpoint_auth_method: "none",
    });
    expect(regRes.status(), await regRes.text()).toBe(201);
    const clientId = (await regRes.json()).client_id as string;

    const email = `dcr-${crypto.randomUUID()}@example.com`;
    const password = "Str0ngPassw0rd!";
    await client.request(
      gql`
        mutation ($params: SignUpRequest!) {
          signup(params: $params) {
            message
          }
        }
      `,
      { params: { email, password, confirm_password: password } },
    );

    const codeVerifier = crypto.randomBytes(32).toString("base64url");
    const codeChallenge = crypto
      .createHash("sha256")
      .update(codeVerifier)
      .digest("base64url");

    const authorizeUrl = new URL("/authorize", BASE_URL);
    authorizeUrl.searchParams.set("response_type", "code");
    authorizeUrl.searchParams.set("client_id", clientId);
    authorizeUrl.searchParams.set("redirect_uri", redirectURI);
    authorizeUrl.searchParams.set("scope", "openid offline_access");
    authorizeUrl.searchParams.set("state", crypto.randomUUID());
    authorizeUrl.searchParams.set("response_mode", "query");
    authorizeUrl.searchParams.set("code_challenge", codeChallenge);
    authorizeUrl.searchParams.set("code_challenge_method", "S256");
    // RFC 8707: binds the issued token's `aud` to the MCP server.
    authorizeUrl.searchParams.set("resource", MCP_RESOURCE);

    await page.goto(authorizeUrl.toString());
    await page.locator("#authorizer-login-email-or-phone-number").fill(email);
    await page.locator("#authorizer-login-password").fill(password);
    await page
      .locator('form[name="authorizer-login-form"] button[type="submit"]')
      .click();
    // First login for a new user hits the optional MFA-setup offer.
    await page
      .getByRole("button", { name: "Skip for now" })
      .click({ timeout: 10_000 })
      .catch(() => {});

    // --- the consent screen ------------------------------------------------
    // RFC 7591 §5 warns that "a rogue client might use the name and logo of a
    // legitimate client" and tells servers to warn users about dynamically
    // registered clients. The name below was chosen by the caller at
    // registration and verified by nobody, which is what the page must say.
    await expect(
      page.getByRole("heading", { name: /E2E DCR Browser Client/ }),
    ).toBeVisible();
    // The redirect host is the only fact about this client the server verified.
    await expect(page.locator(".host")).toHaveText(new URL(redirectURI).host);
    await expect(page.getByText(/runs on your own computer/i)).toBeVisible();

    // The form must be wired to submit the single-use id the server issued —
    // the page carries nothing else, so a broken action, method or hidden field
    // is the difference between consent working and silently doing nothing.
    const form = page.locator('form[action="/authorize/consent"]');
    await expect(form).toHaveAttribute("method", /post/i);
    await expect(
      page.getByRole("button", { name: "Allow access" }),
    ).toBeEnabled();
    const consentID = await form
      .locator('input[name="consent_id"]')
      .inputValue();
    expect(consentID).toBeTruthy();

    // Approve through the PAGE's own request context, which shares this
    // browser's cookies, and walk the redirects one hop at a time.
    //
    // The click itself cannot be used to reach the callback here: Chromium
    // aborts a navigation when a redirect chain that began on a private-network
    // origin (authorizer:8080 resolves to a private address) targets a loopback
    // address — its Private Network Access rule. A direct navigation to the same
    // loopback URL succeeds, and the server log shows POST /authorize/consent
    // 302 → GET /authorize 302 exactly as intended, so the abort is a browser
    // policy about redirect chains rather than anything this server does. A real
    // MCP client starts the flow from localhost and never trips it. Neither
    // page.route nor waitForResponse can observe past the abort, so the hops are
    // driven explicitly instead of pretending the browser completed them.
    const consentRes = await page.request.post("/authorize/consent", {
      form: { consent_id: consentID, action: "approve" },
      maxRedirects: 0,
    });
    expect(consentRes.status(), await consentRes.text()).toBe(302);

    const resumed = await page.request.get(
      new URL(consentRes.headers()["location"], BASE_URL).toString(),
      { maxRedirects: 0 },
    );
    expect(resumed.status(), await resumed.text()).toBe(302);

    // The code must go to the URI this client registered, and nowhere else.
    const landed = new URL(resumed.headers()["location"]);
    expect(landed.origin + landed.pathname).toBe(redirectURI);
    const code = landed.searchParams.get("code")!;
    expect(code).toBeTruthy();

    // --- redeem it as a public client --------------------------------------
    // No client_secret anywhere: PKCE alone binds the code to the instance that
    // started the flow.
    const tokenRes = await request.post("/oauth/token", {
      form: {
        grant_type: "authorization_code",
        code,
        client_id: clientId,
        redirect_uri: redirectURI,
        code_verifier: codeVerifier,
        resource: MCP_RESOURCE,
      },
    });
    expect(tokenRes.status(), await tokenRes.text()).toBe(200);
    const tokens = await tokenRes.json();
    expect(tokens.access_token).toBeTruthy();
    expect(tokens.refresh_token).toBeTruthy();

    // --- the token reaches the tool surface --------------------------------
    const init = await request.post("/mcp", {
      headers: {
        ...MCP_HEADERS,
        Authorization: `Bearer ${tokens.access_token}`,
      },
      data: INITIALIZE_RPC,
    });
    expect(init.status(), await init.text()).toBe(200);
    const sessionId = init.headers()["mcp-session-id"];
    expect(sessionId).toBeTruthy();

    const tools = await request.post("/mcp", {
      headers: {
        ...MCP_HEADERS,
        Authorization: `Bearer ${tokens.access_token}`,
        "Mcp-Session-Id": sessionId,
      },
      data: { jsonrpc: "2.0", id: 2, method: "tools/list" },
    });
    expect(tools.status()).toBe(200);
    expect(await tools.text()).toContain("profile");

    // --- the refresh a long-lived connection depends on --------------------
    // Claude Code registers refresh_token, so this runs on every rotation. The
    // resource binding must survive it, or connections die at the first refresh
    // rather than at connect time.
    const refreshRes = await request.post("/oauth/token", {
      form: {
        grant_type: "refresh_token",
        refresh_token: tokens.refresh_token,
        client_id: clientId,
      },
    });
    expect(refreshRes.status(), await refreshRes.text()).toBe(200);
    const refreshed = await refreshRes.json();
    expect(refreshed.access_token).toBeTruthy();
    expect(refreshed.access_token).not.toBe(tokens.access_token);

    const afterRefresh = await request.post("/mcp", {
      headers: {
        ...MCP_HEADERS,
        Authorization: `Bearer ${refreshed.access_token}`,
        "Mcp-Session-Id": sessionId,
      },
      data: { jsonrpc: "2.0", id: 3, method: "tools/list" },
    });
    expect(afterRefresh.status(), await afterRefresh.text()).toBe(200);
  });

  test("PKCE is required of a self-registered client", async ({ request }) => {
    // RFC 9700 §2.1.1 "Public clients MUST use PKCE"; OAuth 2.1 §4.1.1 has the
    // authorization server MUST enforce it. Refused at /authorize so the user is
    // never asked to log in and approve a request that could never complete.
    const regRes = await register(request, {
      client_name: "No PKCE Client",
      redirect_uris: [redirectURI],
    });
    expect(regRes.status()).toBe(201);
    const clientId = (await regRes.json()).client_id as string;

    const u = new URL("/authorize", BASE_URL);
    u.searchParams.set("response_type", "code");
    u.searchParams.set("client_id", clientId);
    u.searchParams.set("redirect_uri", redirectURI);
    u.searchParams.set("scope", "openid");
    u.searchParams.set("state", crypto.randomUUID());

    const res = await request.get(u.toString(), { maxRedirects: 0 });
    expect(res.status()).toBe(400);
    expect(await res.text()).toContain("code_challenge is required");
  });
});
