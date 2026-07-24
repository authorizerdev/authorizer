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
  opts: { name: string; idpEntityId: string; idpSsoUrl: string; idpCertificate: string; allowIdpInitiated?: boolean },
  baseUrl?: string
): Promise<{ id: string }> {
  const query = gql`
    mutation ($params: CreateOrgSAMLConnectionRequest!) {
      _create_org_saml_connection(params: $params) { id }
    }
  `;
  const res = await getClient(baseUrl).request<{ _create_org_saml_connection: { id: string } }>(query, {
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
  }>(listQuery, { params: { pagination: { limit: 1000 } } });
  const stale = res._trusted_issuers.trusted_issuers.find((t) => t.issuer_url === idpEntityId);
  if (!stale) return;

  const deleteQuery = gql`
    mutation ($params: TrustedIssuerRequest!) {
      _delete_trusted_issuer(params: $params) { message }
    }
  `;
  await client.request(deleteQuery, { params: { id: stale.id } });
}

// signupUser drives the public `signup` mutation (not an admin op — the
// x-authorizer-admin-secret header on `client` is simply ignored by it).
export async function signupUser(email: string, password: string): Promise<void> {
  const query = gql`
    mutation ($params: SignUpRequest!) {
      signup(params: $params) { message }
    }
  `;
  await client.request(query, { params: { email, password, confirm_password: password } });
}

// getUserIdByEmail resolves a user's id via the admin `_users` search query
// (query is a substring filter over email/given_name/family_name/nickname —
// see ListUsersRequest doc comment — so the exact-match find() below guards
// against a substring false-positive).
export async function getUserIdByEmail(email: string): Promise<string> {
  const query = gql`
    query ($params: ListUsersRequest) {
      _users(params: $params) { users { id email } }
    }
  `;
  const res = await client.request<{ _users: { users: { id: string; email: string | null }[] } }>(query, {
    params: { query: email },
  });
  const user = res._users.users.find((u) => u.email === email);
  if (!user) throw new Error(`user not found for email ${email}`);
  return user.id;
}

// getUserByEmail resolves a user's profile fields (given_name/family_name/
// signup_methods) via the same admin `_users` search query getUserIdByEmail
// uses. Needed by social-login specs (tests/social/*.spec.ts) to verify a
// provider's profile claims were actually mapped onto the stored user, not
// just that a session was established.
export async function getUserByEmail(
  email: string
): Promise<{ id: string; email: string | null; given_name: string | null; family_name: string | null; signup_methods: string }> {
  const query = gql`
    query ($params: ListUsersRequest) {
      _users(params: $params) { users { id email given_name family_name signup_methods } }
    }
  `;
  const res = await client.request<{
    _users: {
      users: { id: string; email: string | null; given_name: string | null; family_name: string | null; signup_methods: string }[];
    };
  }>(query, { params: { query: email } });
  const user = res._users.users.find((u) => u.email === email);
  if (!user) throw new Error(`user not found for email ${email}`);
  return user;
}

// getUserByNickname mirrors getUserByEmail for providers that never return
// an email address at all - currently only Twitter/X (processTwitterUserInfo,
// internal/http_handlers/oauth_callback.go, sets Nickname from the profile's
// `username` but leaves Email nil; real Twitter's API doesn't expose email,
// this isn't a mock artifact). tests/social/twitter.spec.ts uses this in
// place of getUserByEmail to look up the account it just created.
export async function getUserByNickname(
  nickname: string
): Promise<{ id: string; given_name: string | null; family_name: string | null; nickname: string | null; signup_methods: string }> {
  const query = gql`
    query ($params: ListUsersRequest) {
      _users(params: $params) { users { id given_name family_name nickname signup_methods } }
    }
  `;
  const res = await client.request<{
    _users: {
      users: { id: string; given_name: string | null; family_name: string | null; nickname: string | null; signup_methods: string }[];
    };
  }>(query, { params: { query: nickname } });
  const user = res._users.users.find((u) => u.nickname === nickname);
  if (!user) throw new Error(`user not found for nickname ${nickname}`);
  return user;
}

// verifyUserEmail force-verifies a user's email (and sets given/family name)
// via the admin `_update_user` mutation, standing in for clicking the real
// verification link — needed because SAML IdP-side issuance
// (internal/http_handlers/saml_idp.go authorizeSAMLIssuance) refuses to
// assert an unverified email as the Subject NameID.
//
// given_name/family_name are required here, not optional convenience: this
// mutation's "at least one param" gate (internal/service/admin_users.go
// UpdateUser) checks GivenName/FamilyName/etc. but NOT EmailVerified, so a
// call with only email_verified set is rejected with "please enter atleast
// one param to update" even though EmailVerified is applied further down
// when present — a real, pre-existing gap in that gate, unrelated to SAML.
// Supplying a name works around it (and doubles as the SAML firstName/
// lastName attribute source for tests/saml-idp.spec.ts).
export async function verifyUserEmail(
  userId: string,
  opts: { givenName: string; familyName: string }
): Promise<void> {
  const query = gql`
    mutation ($params: UpdateUserRequest!) {
      _update_user(params: $params) { id }
    }
  `;
  await client.request(query, {
    params: { id: userId, email_verified: true, given_name: opts.givenName, family_name: opts.familyName },
  });
}

// addOrgMember adds an existing user to an org. SAML IdP-side issuance
// requires org membership (authorizeSAMLIssuance) — any authenticated
// Authorizer user could otherwise obtain an assertion for any org's SP.
export async function addOrgMember(orgId: string, userId: string): Promise<void> {
  const query = gql`
    mutation ($params: AddOrgMemberRequest!) {
      _add_org_member(params: $params) { user_id }
    }
  `;
  await client.request(query, { params: { org_id: orgId, user_id: userId } });
}

// createSAMLServiceProvider registers a downstream SP on the IdP side (the
// inverse of createSAMLConnection, which registers an upstream IdP on the SP
// side) via `_create_saml_service_provider`.
export async function createSAMLServiceProvider(
  orgId: string,
  opts: { name: string; entityId: string; acsUrl: string }
): Promise<{ id: string; entity_id: string }> {
  const query = gql`
    mutation ($params: CreateSAMLServiceProviderRequest!) {
      _create_saml_service_provider(params: $params) { id entity_id }
    }
  `;
  const res = await client.request<{ _create_saml_service_provider: { id: string; entity_id: string } }>(query, {
    params: { org_id: orgId, name: opts.name, entity_id: opts.entityId, acs_url: opts.acsUrl },
  });
  return res._create_saml_service_provider;
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

// getUserPhoneNumberByEmail resolves a user's stored phone_number via the
// admin `_users` search query. Needed by tests/scim.spec.ts's full-attribute
// PATCH coverage: the SCIM PATCH response never echoes phoneNumbers back
// (scimUserResource, internal/http_handlers/scim/users.go, has no phone field),
// so persistence of a `phoneNumbers` PATCH can only be confirmed out-of-band.
export async function getUserPhoneNumberByEmail(email: string): Promise<string | null> {
  const query = gql`
    query ($params: ListUsersRequest) {
      _users(params: $params) { users { id email phone_number } }
    }
  `;
  const res = await client.request<{ _users: { users: { id: string; email: string | null; phone_number: string | null }[] } }>(
    query,
    { params: { query: email } }
  );
  const user = res._users.users.find((u) => u.email === email);
  if (!user) throw new Error(`user not found for email ${email}`);
  return user.phone_number;
}

// addWebhook registers a webhook via the admin `_add_webhook` mutation. Webhooks
// are global (not org-scoped): GetWebhookByEventName matches by event_name prefix
// across all orgs, so one webhook per event_name fires for every matching event.
// Endpoint may be a docker-private host only because the target `authorizer`
// instance runs with --test-allow-private-webhook-hosts=true (see docker-compose).
export async function addWebhook(opts: {
  eventName: string;
  endpoint: string;
  enabled?: boolean;
  headers?: Record<string, string>;
}): Promise<void> {
  const query = gql`
    mutation ($params: AddWebhookRequest!) {
      _add_webhook(params: $params) { message }
    }
  `;
  await client.request(query, {
    params: {
      event_name: opts.eventName,
      endpoint: opts.endpoint,
      enabled: opts.enabled ?? true,
      headers: opts.headers ?? {},
    },
  });
}

export async function setEnforceMFA(enabled: boolean): Promise<void> {
  const query = gql`
    mutation ($params: UpdateEnvRequest!) {
      _update_env(params: $params) { message }
    }
  `;
  await client.request(query, { params: { ENFORCE_MULTI_FACTOR_AUTHENTICATION: enabled } });
}
