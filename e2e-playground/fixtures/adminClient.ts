// e2e-playground/fixtures/adminClient.ts
import { GraphQLClient, gql } from 'graphql-request';

const BASE_URL = process.env.AUTHORIZER_BASE_URL || 'http://localhost:8080';
const ENDPOINT = `${BASE_URL}/graphql`;
const ADMIN_SECRET = process.env.AUTHORIZER_ADMIN_SECRET || 'e2e-admin-secret';

// Admin auth is the x-authorizer-admin-secret header (see
// internal/token/admin_token.go IsSuperAdmin / internal/e2e/smoke_test.go),
// not a Bearer token. Origin is required too: the CSRF middleware rejects
// state-changing requests with no Origin/Referer (internal/graph/schema.graphqls
// comment on AllowedOrigins; must match the server's --allowed-origins).
const client = new GraphQLClient(ENDPOINT, {
  headers: { 'x-authorizer-admin-secret': ADMIN_SECRET, Origin: BASE_URL },
});

export async function createOrg(name: string): Promise<{ id: string; name: string }> {
  const query = gql`
    mutation ($params: CreateOrganizationRequest!) {
      _create_organization(params: $params) { id name }
    }
  `;
  const res = await client.request<{ _create_organization: { id: string; name: string } }>(query, {
    params: { name, display_name: name },
  });
  return res._create_organization;
}

export async function createOIDCConnection(
  orgId: string,
  opts: { name: string; issuerUrl: string; clientId: string; clientSecret: string }
): Promise<{ id: string }> {
  const query = gql`
    mutation ($params: CreateOrgOIDCConnectionRequest!) {
      _create_org_oidc_connection(params: $params) { id }
    }
  `;
  const res = await client.request<{ _create_org_oidc_connection: { id: string } }>(query, {
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

export async function addVerifiedDomain(orgId: string, domain: string): Promise<void> {
  const query = gql`
    mutation ($params: AddVerifiedOrgDomainRequest!) {
      _add_verified_org_domain(params: $params) { domain }
    }
  `;
  await client.request(query, { params: { org_id: orgId, domain } });
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
