import express from 'express';
import * as jose from 'jose';
import crypto from 'node:crypto';

export const app = express();
app.use(express.json());
app.use(express.urlencoded({ extended: true }));

// In-memory per-provider state: current profile to return, and issued tokens.
const profiles: Record<string, Record<string, unknown>> = {};
const issuedTokens = new Map<string, { provider: string; nonce?: string }>();

let signingKey: jose.KeyLike;
let publicJwk: jose.JWK;
const keyReady = jose.generateKeyPair('RS256').then(async ({ privateKey, publicKey }) => {
  signingKey = privateKey;
  publicJwk = await jose.exportJWK(publicKey);
  publicJwk.kid = 'mock-oauth-key-1';
  publicJwk.alg = 'RS256';
  publicJwk.use = 'sig';
});

function defaultProfile(provider: string): Record<string, unknown> {
  const email = `mock-user@${provider}.example.com`;
  switch (provider) {
    case 'github':
      return { name: 'Mock User', email, avatar_url: 'https://example.com/avatar.png' };
    case 'facebook':
      return { first_name: 'Mock', last_name: 'User', email, picture: { data: { url: 'https://example.com/avatar.png' } } };
    case 'linkedin':
      return { localizedFirstName: 'Mock', localizedLastName: 'User' };
    case 'discord':
      // Flat shape matching Discord's real GET /users/@me response
      // (processDiscordUserInfo, internal/http_handlers/oauth_callback.go,
      // reads id/username/avatar/email directly - no "user" wrapper, that
      // was /oauth2/@me's shape, which never includes email).
      return { id: '123', username: 'mockuser', avatar: 'abc', email };
    case 'twitter':
      return { data: { id: '123', name: 'Mock User', username: 'mockuser', profile_image_url: 'https://example.com/a.png' } };
    case 'roblox':
      return { name: 'Mock User', nickname: 'mockuser', picture: 'https://example.com/a.png', email };
    default:
      return { sub: `mock-${provider}-sub`, email, given_name: 'Mock', family_name: 'User' };
  }
}

app.post('/:provider/__configure', (req, res) => {
  profiles[req.params.provider] = req.body.profile;
  res.sendStatus(204);
});

app.get('/:provider/.well-known/openid-configuration', (req, res) => {
  const base = `${req.protocol}://${req.get('host')}/${req.params.provider}`;
  res.json({
    issuer: base,
    authorization_endpoint: `${base}/authorize`,
    token_endpoint: `${base}/token`,
    jwks_uri: `${base}/jwks`,
    userinfo_endpoint: `${base}/userinfo`,
    response_types_supported: ['code'],
    subject_types_supported: ['public'],
    id_token_signing_alg_values_supported: ['RS256'],
  });
});

app.get('/:provider/jwks', async (_req, res) => {
  await keyReady;
  res.json({ keys: [publicJwk] });
});

app.all('/:provider/authorize', (req, res) => {
  const redirectUri = String(req.query.redirect_uri);
  const state = String(req.query.state || '');
  const nonce = req.query.nonce ? String(req.query.nonce) : undefined;
  const code = crypto.randomUUID();
  issuedTokens.set(code, { provider: req.params.provider, nonce });
  const url = new URL(redirectUri);
  url.searchParams.set('code', code);
  if (state) url.searchParams.set('state', state);
  if (req.params.provider === 'apple') {
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
    const profile = (profiles['apple'] || defaultProfile('apple')) as {
      given_name?: string;
      family_name?: string;
      omit_user_field?: boolean;
    };
    if (!profile.omit_user_field) {
      url.searchParams.set(
        'user',
        JSON.stringify({ name: { firstName: profile.given_name || '', lastName: profile.family_name || '' } })
      );
    }
  }
  res.redirect(302, url.toString());
});

app.post('/:provider/token', async (req, res) => {
  const provider = req.params.provider;
  // Recover the nonce captured at /authorize time (RFC-required round-trip
  // through the id_token) — keyed by the authorization code presented here.
  const nonce = req.body?.code ? issuedTokens.get(String(req.body.code))?.nonce : undefined;
  const accessToken = crypto.randomUUID();
  issuedTokens.set(accessToken, { provider });

  const body: Record<string, unknown> = {
    access_token: accessToken,
    token_type: 'bearer',
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
    const base = `${req.protocol}://${req.get('host')}/${provider}`;
    const profile = profiles[provider] || defaultProfile(provider);
    const idToken = await new jose.SignJWT({ ...profile, ...(nonce ? { nonce } : {}) })
      .setProtectedHeader({ alg: 'RS256', kid: 'mock-oauth-key-1' })
      .setIssuer(base)
      .setAudience('mock-client-id')
      .setSubject(String((profile as Record<string, unknown>).sub || `mock-${provider}-sub`))
      .setIssuedAt()
      .setExpirationTime('10m')
      .sign(signingKey);
    body.id_token = idToken;
  }

  res.json(body);
});

app.get(['/:provider/userinfo', '/:provider/user', '/:provider/@me', '/:provider/2/users/me'], (req, res) => {
  const provider = req.params.provider;
  res.json(profiles[provider] || defaultProfile(provider));
});

app.get('/:provider/user/emails', (req, res) => {
  const profile = (profiles[req.params.provider] || defaultProfile(req.params.provider)) as { email?: string };
  res.json([{ email: profile.email || 'mock-user@github.example.com', primary: true }]);
});

app.get('/:provider/emailAddress', (req, res) => {
  const profile = (profiles[req.params.provider] || defaultProfile(req.params.provider)) as { email?: string };
  res.json({ elements: [{ 'handle~': { emailAddress: profile.email || 'mock-user@linkedin.example.com' } }] });
});

if (require.main === module) {
  app.listen(4000, () => console.log('mock-oauth listening on :4000'));
}
