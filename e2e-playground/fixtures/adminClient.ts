// e2e-playground/fixtures/adminClient.ts
import { GraphQLClient, gql } from 'graphql-request';

const BASE_URL = process.env.AUTHORIZER_BASE_URL || 'http://localhost:8080';
const ADMIN_SECRET = process.env.AUTHORIZER_ADMIN_SECRET || 'e2e-admin-secret';

// Admin auth is the x-authorizer-admin-secret header (see
// internal/token/admin_token.go IsSuperAdmin / internal/e2e/smoke_test.go),
// not a Bearer token. Origin is required too: the CSRF middleware rejects
// state-changing requests with no Origin/Referer (internal/graph/schema.graphqls
// comment on AllowedOrigins; must match the server's --allowed-origins).
const client = new GraphQLClient(`${BASE_URL}/graphql`, {
  headers: { 'x-authorizer-admin-secret': ADMIN_SECRET, Origin: BASE_URL },
});

// getClient returns the default (AUTHORIZER_BASE_URL / :8080) admin client,
// or a one-off client scoped to baseUrl when given. Needed because this
// module's BASE_URL is a single process-wide constant, independent of
// Playwright's per-project `use.baseURL` — the `sso-discovery` project points
// page navigation at :8081 (authorizer-sso) via its own baseURL, but without
// this, admin calls below would still silently hit :8080's `authorizer`
// service instead, so the org/connection the JIT test creates would exist on
// the wrong server.
function getClient(baseUrl?: string): GraphQLClient {
  if (!baseUrl || baseUrl === BASE_URL) return client;
  return new GraphQLClient(`${baseUrl}/graphql`, {
    headers: { 'x-authorizer-admin-secret': ADMIN_SECRET, Origin: baseUrl },
  });
}

export async function createOrg(name: string, baseUrl?: string): Promise<{ id: string; name: string }> {
  const query = gql`
    mutation ($params: CreateOrganizationRequest!) {
      _create_organization(params: $params) { id name }
    }
  `;
  const res = await getClient(baseUrl).request<{ _create_organization: { id: string; name: string } }>(query, {
    params: { name, display_name: name },
  });
  return res._create_organization;
}

export async function createOIDCConnection(
  orgId: string,
  opts: { name: string; issuerUrl: string; clientId: string; clientSecret: string },
  baseUrl?: string
): Promise<{ id: string }> {
  const query = gql`
    mutation ($params: CreateOrgOIDCConnectionRequest!) {
      _create_org_oidc_connection(params: $params) { id }
    }
  `;
  const res = await getClient(baseUrl).request<{ _create_org_oidc_connection: { id: string } }>(query, {
    params: {
      org_id: orgId,
      name: opts.name,
      issuer_url: opts.issuerUrl,
      client_id: opts.clientId,
      client_secret: opts.clientSecret,
    },
  });
  return res._create_org_oidc_connection;
}

export async function createSAMLConnection(
  orgId: string,
  opts: { name: string; idpEntityId: string; idpSsoUrl: string; idpCertificate: string; allowIdpInitiated?: boolean }
): Promise<{ id: string }> {
  const query = gql`
    mutation ($params: CreateOrgSAMLConnectionRequest!) {
      _create_org_saml_connection(params: $params) { id }
    }
  `;
  const res = await client.request<{ _create_org_saml_connection: { id: string } }>(query, {
    params: {
      org_id: orgId,
      name: opts.name,
      idp_entity_id: opts.idpEntityId,
      idp_sso_url: opts.idpSsoUrl,
      idp_certificate: opts.idpCertificate,
      allow_idp_initiated: opts.allowIdpInitiated ?? false,
    },
  });
  return res._create_org_saml_connection;
}

// deleteSAMLConnectionByEntityID removes any existing OrgSAMLConnection whose
// idp_entity_id matches (a no-op if none exists). idp_entity_id is enforced
// globally-unique across all orgs (internal/service/admin_org_saml.go
// GetTrustedIssuerByIssuerURL check), so a test that must use a fixed entity
// ID (matching a mock IdP's own hardcoded entityID) needs to clean up any
// stale row left by a prior unclean run before creating a fresh connection.
// There's no admin query keyed on idp_entity_id directly, so this lists
// trusted issuers (the shared backing table for SAML/OIDC connections and
// machine-identity issuers, exposed via issuer_url on TrustedIssuer) and
// deletes the generic way (_delete_trusted_issuer works on any issuer kind).
export async function deleteSAMLConnectionByEntityID(idpEntityId: string): Promise<void> {
  const listQuery = gql`
    query ($params: ListTrustedIssuersRequest) {
      _trusted_issuers(params: $params) {
        trusted_issuers { id issuer_url }
      }
    }
  `;
  const res = await client.request<{
    _trusted_issuers: { trusted_issuers: { id: string; issuer_url: string }[] };
  }>(listQuery, { params: { pagination: { pagination: { limit: 1000 } } } });
  const stale = res._trusted_issuers.trusted_issuers.find((t) => t.issuer_url === idpEntityId);
  if (!stale) return;

  const deleteQuery = gql`
    mutation ($params: TrustedIssuerRequest!) {
      _delete_trusted_issuer(params: $params) { message }
    }
  `;
  await client.request(deleteQuery, { params: { id: stale.id } });
}

export async function addVerifiedDomain(orgId: string, domain: string, baseUrl?: string): Promise<void> {
  const query = gql`
    mutation ($params: AddVerifiedOrgDomainRequest!) {
      _add_verified_org_domain(params: $params) { domain }
    }
  `;
  await getClient(baseUrl).request(query, { params: { org_id: orgId, domain } });
}

export async function createSCIMEndpoint(orgId: string): Promise<{ token: string; endpoint: string }> {
  const query = gql`
    mutation ($params: CreateScimEndpointRequest!) {
      _create_scim_endpoint(params: $params) {
        token
        scim_endpoint { id }
      }
    }
  `;
  const res = await client.request<{ _create_scim_endpoint: { token: string; scim_endpoint: { id: string } } }>(
    query,
    { params: { org_id: orgId } }
  );
  return { token: res._create_scim_endpoint.token, endpoint: `/scim/v2` };
}

export async function setEnforceMFA(enabled: boolean): Promise<void> {
  const query = gql`
    mutation ($params: UpdateEnvRequest!) {
      _update_env(params: $params) { message }
    }
  `;
  await client.request(query, { params: { ENFORCE_MULTI_FACTOR_AUTHENTICATION: enabled } });
}
