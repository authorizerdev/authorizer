// e2e-playground/tests/sso-discovery.spec.ts
import { test, expect } from '@playwright/test';
import fs from 'node:fs';
import path from 'node:path';
import { createOrg, addVerifiedDomain, createOIDCConnection, createSAMLConnection } from '../fixtures/adminClient';

// This spec exercises the public, unauthenticated GET /api/v1/org-discovery
// route directly (internal/http_handlers/org_discovery.go) via the `request`
// fixture — no browser needed. The browser-driven /app home-realm-discovery
// UI flow that CONSUMES this route is already covered end-to-end by
// tests/oidc-sso-rp.spec.ts.
//
// The route is gated behind Config.EnableOrgDiscovery, which defaults to
// FALSE (cmd/root.go --enable-org-discovery, opt-in — "off keeps the login
// page unchanged"). The plain `authorizer` compose service does not pass that
// flag, so it 404s there; only `authorizer-sso` sets
// --enable-org-discovery=true. This file therefore runs under the
// `sso-discovery` Playwright project (playwright.config.ts), whose baseURL
// points at authorizer-sso (:8081) — same project as oidc-sso-rp.spec.ts.
//
// Response contract (read directly from org_discovery.go, mirrored by
// internal/integration_tests/org_discovery_test.go):
//   - match:    200 {"connection": {"type": "saml"|"oidc", "login_url": "..."}}
//   - no-match: 200 {"connection": null} — deliberately identical for
//     unknown domain, unverified/pending domain, disabled org, and a verified
//     domain with no SSO connection (privacy: no tenant-enumeration signal).
//   - malformed email: 400 {"error":"invalid_request",...}
//   - SAML beats OIDC when an org has both active connections (spec-locked).
const idpCert = fs.readFileSync(path.join(__dirname, '../fixtures/certs/idp-cert.pem'), 'utf8');

// Server-side issuer URLs below are dialed (or, for SAML, just format-checked
// at creation time — no live handshake) by the authorizer-sso container, so
// they use the docker-network-internal host, not localhost. Matches the
// pattern established in oidc-sso-rp.spec.ts / saml-sp.spec.ts.
const MOCK_OAUTH_INTERNAL_BASE = process.env.MOCK_OAUTH_INTERNAL_BASE_URL || 'http://mock-oauth:4000';

test.describe('SSO — home-realm discovery API', () => {
  test('verified domain with an OIDC-only connection routes to the oidc login_url', async ({ request, baseURL }) => {
    const org = await createOrg(`discovery-oidc-${Date.now()}`, baseURL);
    const domain = `${org.id}.example.com`;
    await addVerifiedDomain(org.id, domain, baseURL);
    await createOIDCConnection(
      org.id,
      {
        name: 'primary-idp',
        issuerUrl: `${MOCK_OAUTH_INTERNAL_BASE}/discovery-realm-${org.id}`,
        clientId: 'mock-client-id',
        clientSecret: 'mock-client-secret',
      },
      baseURL
    );

    const res = await request.get(`/api/v1/org-discovery?email=${encodeURIComponent(`someone@${domain}`)}`);
    expect(res.status()).toBe(200);
    const body = await res.json();
    expect(body).toEqual({ connection: { type: 'oidc', login_url: `/oauth/sso/${org.name}/login` } });
  });

  test('verified domain with a SAML-only connection routes to the saml login_url', async ({ request, baseURL }) => {
    const org = await createOrg(`discovery-saml-${Date.now()}`, baseURL);
    const domain = `${org.id}.example.com`;
    await addVerifiedDomain(org.id, domain, baseURL);
    await createSAMLConnection(
      org.id,
      {
        name: 'primary-idp',
        idpEntityId: `https://idp-${org.id}.example.com/metadata`,
        idpSsoUrl: `https://idp-${org.id}.example.com/sso`,
        idpCertificate: idpCert,
      },
      baseURL
    );

    const res = await request.get(`/api/v1/org-discovery?email=${encodeURIComponent(`someone@${domain}`)}`);
    expect(res.status()).toBe(200);
    const body = await res.json();
    expect(body).toEqual({ connection: { type: 'saml', login_url: `/oauth/saml/${org.name}/login` } });
  });

  test('SAML takes precedence over OIDC when an org has both active connections', async ({ request, baseURL }) => {
    const org = await createOrg(`discovery-both-${Date.now()}`, baseURL);
    const domain = `${org.id}.example.com`;
    await addVerifiedDomain(org.id, domain, baseURL);
    await createOIDCConnection(
      org.id,
      {
        name: 'primary-idp',
        issuerUrl: `${MOCK_OAUTH_INTERNAL_BASE}/discovery-both-realm-${org.id}`,
        clientId: 'mock-client-id',
        clientSecret: 'mock-client-secret',
      },
      baseURL
    );
    await createSAMLConnection(
      org.id,
      {
        name: 'secondary-idp',
        idpEntityId: `https://idp-both-${org.id}.example.com/metadata`,
        idpSsoUrl: `https://idp-both-${org.id}.example.com/sso`,
        idpCertificate: idpCert,
      },
      baseURL
    );

    const res = await request.get(`/api/v1/org-discovery?email=${encodeURIComponent(`someone@${domain}`)}`);
    expect(res.status()).toBe(200);
    const body = await res.json();
    expect(body).toEqual({ connection: { type: 'saml', login_url: `/oauth/saml/${org.name}/login` } });
  });

  test('a verified domain with no SSO connection returns the indistinguishable null response', async ({
    request,
    baseURL,
  }) => {
    const org = await createOrg(`discovery-noconn-${Date.now()}`, baseURL);
    const domain = `${org.id}.example.com`;
    await addVerifiedDomain(org.id, domain, baseURL);

    const res = await request.get(`/api/v1/org-discovery?email=${encodeURIComponent(`someone@${domain}`)}`);
    expect(res.status()).toBe(200);
    expect(await res.json()).toEqual({ connection: null });
  });

  test('an unknown domain returns the same null response (no tenant-enumeration signal)', async ({ request }) => {
    const res = await request.get(
      `/api/v1/org-discovery?email=${encodeURIComponent(`someone@totally-unrecognized-${Date.now()}.example`)}`
    );
    expect(res.status()).toBe(200);
    expect(await res.json()).toEqual({ connection: null });
  });

  test('malformed email returns 400 with the invalid_request error', async ({ request }) => {
    const res = await request.get(`/api/v1/org-discovery?email=${encodeURIComponent('not-an-email')}`);
    expect(res.status()).toBe(400);
    expect(await res.json()).toEqual({ error: 'invalid_request', error_description: 'invalid email' });
  });

  test('a domain already verified by one org cannot be claimed by a second org', async ({ request, baseURL }) => {
    const orgA = await createOrg(`discovery-ambig-a-${Date.now()}`, baseURL);
    const orgB = await createOrg(`discovery-ambig-b-${Date.now()}`, baseURL);
    const sharedDomain = `shared-${Date.now()}.example.com`;
    await addVerifiedDomain(orgA.id, sharedDomain, baseURL);

    // GetOrgDomainByDomain is a primary-key lookup keyed on the domain
    // (admin_org_domains.go AddVerifiedOrgDomain), so a second org claiming an
    // already-verified domain must fail, not silently take it over.
    await expect(addVerifiedDomain(orgB.id, sharedDomain, baseURL)).rejects.toThrow(
      /domain_already_verified_by_another_org/
    );

    // Discovery must still route to the original owner, orgA.
    const res = await request.get(`/api/v1/org-discovery?email=${encodeURIComponent(`someone@${sharedDomain}`)}`);
    expect(res.status()).toBe(200);
    expect(await res.json()).toEqual({ connection: null }); // orgA has no SSO connection configured
  });
});
