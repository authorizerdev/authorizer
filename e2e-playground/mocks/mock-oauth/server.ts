import express from "express";
import * as jose from "jose";
import crypto from "node:crypto";

export const app = express();
app.use(express.json());
app.use(express.urlencoded({ extended: true }));

// In-memory per-provider state: current profile to return, and issued tokens.
const profiles: Record<string, Record<string, unknown>> = {};
const issuedTokens = new Map<string, { provider: string; nonce?: string }>();

let signingKey: jose.KeyLike;
let publicJwk: jose.JWK;
const keyReady = jose
  .generateKeyPair("RS256")
  .then(async ({ privateKey, publicKey }) => {
    signingKey = privateKey;
    publicJwk = await jose.exportJWK(publicKey);
    publicJwk.kid = "mock-oauth-key-1";
    publicJwk.alg = "RS256";
    publicJwk.use = "sig";
  });

function defaultProfile(provider: string): Record<string, unknown> {
  const email = `mock-user@${provider}.example.com`;
  switch (provider) {
    case "github":
      // Mixed types on purpose: GitHub's real GET /user carries a numeric id
      // and boolean flags alongside the strings.
      return {
        id: 583231,
        login: "mockuser",
        name: "Mock User",
        email,
        avatar_url: "https://example.com/avatar.png",
        public_repos: 8,
        site_admin: false,
        company: null,
      };
    case "facebook":
      return {
        first_name: "Mock",
        last_name: "User",
        email,
        picture: { data: { url: "https://example.com/avatar.png" } },
      };
    case "linkedin":
      // OIDC userinfo shape (api.linkedin.com/v2/userinfo), which replaced the
      // legacy /v2/me + /v2/emailAddress pair.
      return {
        sub: "mock-linkedin-sub",
        name: "Mock User",
        given_name: "Mock",
        family_name: "User",
        picture: "https://example.com/a.png",
        email,
        email_verified: true,
      };
    case "discord":
      // Flat shape matching Discord's real GET /users/@me response
      // (processDiscordUserInfo, internal/http_handlers/oauth_callback.go,
      // reads id/username/avatar/email directly - no "user" wrapper, that
      // was /oauth2/@me's shape, which never includes email).
      // `verified` is Discord's email-confirmation flag; Authorizer refuses to
      // resolve a local account from an address the provider hasn't attested.
      return {
        id: "123",
        username: "mockuser",
        avatar: "abc",
        email,
        verified: true,
      };
    case "twitter":
      return {
        data: {
          id: "123",
          name: "Mock User",
          username: "mockuser",
          profile_image_url: "https://example.com/a.png",
        },
      };
    case "roblox":
      return {
        name: "Mock User",
        nickname: "mockuser",
        picture: "https://example.com/a.png",
        email,
        email_verified: true,
      };
    default:
      // OIDC `email_verified` (Core §5.1). Google/Apple/Twitch/Microsoft all
      // route through here, and the callback rejects an unattested address.
      return {
        sub: `mock-${provider}-sub`,
        email,
        email_verified: true,
        given_name: "Mock",
        family_name: "User",
      };
  }
}

app.post("/:provider/__configure", (req, res) => {
  profiles[req.params.provider] = req.body.profile;
  res.sendStatus(204);
});

app.get("/:provider/.well-known/openid-configuration", (req, res) => {
  const base = `${req.protocol}://${req.get("host")}/${req.params.provider}`;
  res.json({
    issuer: base,
    authorization_endpoint: `${base}/authorize`,
    token_endpoint: `${base}/token`,
    jwks_uri: `${base}/jwks`,
    userinfo_endpoint: `${base}/userinfo`,
    response_types_supported: ["code"],
    subject_types_supported: ["public"],
    id_token_signing_alg_values_supported: ["RS256"],
  });
});

app.get("/:provider/jwks", async (_req, res) => {
  await keyReady;
  res.json({ keys: [publicJwk] });
});

// Mint a workload-identity assertion: a JWT shaped like a Kubernetes projected
// ServiceAccount token, signed by this mock's own key so it verifies against
// the JWKS above.
//
// This stands in for the cluster in the workload-identity e2e. Authorizer must
// FETCH the signing keys to verify it, which is the whole point of the test —
// every other test of that path stubs the fetch, and the fetch is what decides
// whether the feature works in production.
app.get("/:provider/workload-token", async (req, res) => {
  await keyReady;
  const base = `${req.protocol}://${req.get("host")}/${req.params.provider}`;
  const sub = String(req.query.sub || "system:serviceaccount:default:worker");
  const aud = String(req.query.aud || "");
  if (!aud) {
    res.status(400).json({ error: "aud query parameter is required" });
    return;
  }
  const now = Math.floor(Date.now() / 1000);
  const token = await new jose.SignJWT({ sub, aud })
    .setProtectedHeader({ alg: "RS256" })
    .setIssuer(base)
    .setSubject(sub)
    .setAudience(aud)
    // A real projected ServiceAccount token carries a jti — verified against a
    // live cluster, whose claims are aud/exp/iat/iss/jti/kubernetes.io/nbf/sub.
    // It is what makes each mint distinct, and it is the key Authorizer's
    // single-use replay check uses, so omitting it here would both misrepresent
    // Kubernetes and make two mints in the same second collide.
    .setJti(crypto.randomUUID())
    .setIssuedAt(now)
    // Well inside Authorizer's 1-hour declared-lifetime ceiling.
    .setExpirationTime(now + 600)
    .sign(signingKey);
  res.json({ token, issuer: base, jwks_uri: `${base}/jwks`, sub, aud });
});

app.all("/:provider/authorize", (req, res) => {
  const redirectUri = String(req.query.redirect_uri);
  const state = String(req.query.state || "");
  const nonce = req.query.nonce ? String(req.query.nonce) : undefined;
  const code = crypto.randomUUID();
  issuedTokens.set(code, { provider: req.params.provider, nonce });
  const url = new URL(redirectUri);
  url.searchParams.set("code", code);
  if (state) url.searchParams.set("state", state);
  if (req.params.provider === "apple") {
    // Real Apple sends a `user` field (JSON: {"name":{"firstName","lastName"}})
    // alongside the code on first authorization only - it's constructed by
    // Apple's own hosted consent page, not by Authorizer or its frontend, and
    // isn't part of the id_token. Authorizer's OAuthCallbackHandler
    // (processAppleUserInfo, internal/http_handlers/oauth_callback.go) reads
    // it via ctx.Request.FormValue("user"), which Go resolves from either a
    // POST body or - as here - the URL query string, so mirroring it as a
    // query param on this redirect (rather than an auto-submitted form POST)
    // reaches the same code path.
    //
    // Real Apple omits this field entirely on every login after the first
    // (one-time grant, not re-sent). Tests exercising that returning-user
    // path set `omit_user_field: true` on the configured profile so this
    // mock matches - see __configure below.
    const profile = (profiles["apple"] || defaultProfile("apple")) as {
      given_name?: string;
      family_name?: string;
      omit_user_field?: boolean;
    };
    if (!profile.omit_user_field) {
      url.searchParams.set(
        "user",
        JSON.stringify({
          name: {
            firstName: profile.given_name || "",
            lastName: profile.family_name || "",
          },
        }),
      );
    }
  }
  res.redirect(302, url.toString());
});

app.post("/:provider/token", async (req, res) => {
  const provider = req.params.provider;
  // Recover the nonce captured at /authorize time (RFC-required round-trip
  // through the id_token) — keyed by the authorization code presented here.
  const nonce = req.body?.code
    ? issuedTokens.get(String(req.body.code))?.nonce
    : undefined;
  const accessToken = crypto.randomUUID();
  issuedTokens.set(accessToken, { provider });

  const body: Record<string, unknown> = {
    access_token: accessToken,
    token_type: "bearer",
    expires_in: 3600,
  };

  // Always issue a signed id_token, not just for the 4 named OIDC-verified
  // social providers: SSO/home-realm-discovery tests register a per-org OIDC
  // connection against a synthetic realm name (e.g. sso-org-<id>), which
  // still needs a real id_token for Authorizer's SSO broker to complete the
  // flow. Harmless for the REST-profile social providers too — their code
  // path in oauth_callback.go never reads token.Extra("id_token"), so an
  // extra field in the response is ignored.
  {
    await keyReady;
    const base = `${req.protocol}://${req.get("host")}/${provider}`;
    const profile = profiles[provider] || defaultProfile(provider);
    const idToken = await new jose.SignJWT({
      ...profile,
      ...(nonce ? { nonce } : {}),
    })
      .setProtectedHeader({ alg: "RS256", kid: "mock-oauth-key-1" })
      .setIssuer(base)
      .setAudience("mock-client-id")
      .setSubject(
        String(
          (profile as Record<string, unknown>).sub || `mock-${provider}-sub`,
        ),
      )
      .setIssuedAt()
      .setExpirationTime("10m")
      .sign(signingKey);
    body.id_token = idToken;
  }

  res.json(body);
});

app.get(
  [
    "/:provider/userinfo",
    "/:provider/user",
    "/:provider/@me",
    "/:provider/2/users/me",
  ],
  (req, res) => {
    const provider = req.params.provider;
    res.json(profiles[provider] || defaultProfile(provider));
  },
);

app.get("/:provider/user/emails", (req, res) => {
  const profile = (profiles[req.params.provider] ||
    defaultProfile(req.params.provider)) as { email?: string };
  res.json([
    {
      email: profile.email || "mock-user@github.example.com",
      primary: true,
      verified: true,
    },
  ]);
});

if (require.main === module) {
  app.listen(4000, () => console.log("mock-oauth listening on :4000"));
}
