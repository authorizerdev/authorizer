import express from 'express';
import * as jose from 'jose';
import crypto from 'node:crypto';

export const app = express();
app.use(express.json());
app.use(express.urlencoded({ extended: true }));

// In-memory per-provider state: current profile to return, and issued tokens.
const profiles: Record<string, Record<string, unknown>> = {};
const issuedTokens = new Map<string, { provider: string }>();

// OIDC-verified providers (Task 2's route convention): Google, Apple, Microsoft, Twitch.
const OIDC_PROVIDERS = new Set(['google', 'apple', 'microsoft', 'twitch']);

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
      return { user: { id: '123', username: 'mockuser', avatar: 'abc' } };
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
  const code = crypto.randomUUID();
  issuedTokens.set(code, { provider: req.params.provider });
  const url = new URL(redirectUri);
  url.searchParams.set('code', code);
  if (state) url.searchParams.set('state', state);
  res.redirect(302, url.toString());
});

app.post('/:provider/token', async (req, res) => {
  const provider = req.params.provider;
  const accessToken = crypto.randomUUID();
  issuedTokens.set(accessToken, { provider });

  const body: Record<string, unknown> = {
    access_token: accessToken,
    token_type: 'bearer',
    expires_in: 3600,
  };

  if (OIDC_PROVIDERS.has(provider)) {
    await keyReady;
    const base = `${req.protocol}://${req.get('host')}/${provider}`;
    const profile = profiles[provider] || defaultProfile(provider);
    const idToken = await new jose.SignJWT({ ...profile })
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
