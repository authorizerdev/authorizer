export const MetaQuery = `
  query MetaQuery {
    meta {
      version
      client_id
    }
  }
`;

export const AdminSessionQuery = `
  query {
    _admin_session{
	    message
    }
  }
`;

export const UserDetailsQuery = `
  query($params: ListUsersRequest) {
    _users(params: $params) {
      pagination {
        limit
        page
        offset
        total
      }
      users {
        id
        email
        email_verified
        phone_number_verified
        given_name
        family_name
        middle_name
        nickname
        gender
        birthdate
        phone_number
        picture
        signup_methods
        roles
        created_at
        revoked_timestamp
        is_multi_factor_auth_enabled
        enrolled_mfa_methods
      }
    }
  }
`;

export const WebhooksDataQuery = `
  query getWebhooksData($params: PaginatedRequest!) {
    _webhooks(params: $params){
      webhooks{
        id
        event_description
        event_name
        endpoint
        enabled
        headers
      }
      pagination{
        limit
        page
        offset
        total
      }
    }
  }
`;

export const EmailTemplatesQuery = `
  query getEmailTemplates($params: PaginatedRequest!) {
    _email_templates(params: $params) {
      email_templates {
        id
        event_name
        subject
        created_at
        template
        design
      }
      pagination {
        limit
        page
        offset
        total
      }
    }
  }
`;

export const WebhookLogsQuery = `
  query getWebhookLogs($params: ListWebhookLogRequest!) {
    _webhook_logs(params: $params) {
      webhook_logs {
        id
        http_status
        request
        response
        created_at
      }
      pagination {
        limit
        page
        offset
        total
      }
    }
  }
`;

export const AuditLogsQuery = `
  query getAuditLogs($params: ListAuditLogRequest!) {
    _audit_logs(params: $params) {
      audit_logs {
        id
        actor_id
        actor_type
        actor_email
        action
        resource_type
        resource_id
        ip_address
        user_agent
        metadata
        created_at
      }
      pagination {
        limit
        page
        offset
        total
      }
    }
  }
`;

export const FgaGetModelQuery = `
  query fgaGetModel {
    _fga_get_model {
      id
      dsl
    }
  }
`;

export const FgaReadTuplesQuery = `
  query fgaReadTuples($params: FgaReadTuplesInput!) {
    _fga_read_tuples(params: $params) {
      tuples {
        user
        relation
        object
      }
      continuation_token
    }
  }
`;

// ListPermissionsQuery enumerates what a subject can access. relation and
// object_type are optional — omitting them lists ALL permissions the subject
// holds across the model. The optional user param is honored for the
// super-admin dashboard session.
export const ListPermissionsQuery = `
  query listPermissions($params: ListPermissionsInput!) {
    list_permissions(params: $params) {
      objects
      permissions {
        object
        relation
      }
      truncated
    }
  }
`;

// AdminRolesQuery fetches the instance's configured roles via the admin-only
// _admin_meta query so the FGA model builder can seed its matrix with the real
// roles, and the dashboard can flag FGA role references that aren't configured
// roles. (_env, the old source, is deprecated in v2.)
export const AdminRolesQuery = `
  query adminMeta {
    _admin_meta {
      roles
      is_multi_factor_auth_service_enabled
    }
  }
`;

export const ClientsQuery = `
  query getClients($params: ListClientsRequest) {
    _clients(params: $params) {
      clients {
        id
        client_id
        name
        description
        allowed_scopes
        is_active
        created_at
        updated_at
      }
      pagination {
        limit
        page
        offset
        total
      }
    }
  }
`;

export const TrustedIssuersQuery = `
  query getTrustedIssuers($params: ListTrustedIssuersRequest) {
    _trusted_issuers(params: $params) {
      trusted_issuers {
        id
        service_account_id
        name
        issuer_url
        key_source_type
        jwks_url
        expected_aud
        subject_claim
        allowed_subjects
        issuer_type
        is_active
        spiffe_refresh_hint_seconds
        created_at
        updated_at
      }
      pagination {
        limit
        page
        offset
        total
      }
    }
  }
`;

export const OrganizationsQuery = `
  query getOrganizations($params: ListOrganizationsRequest) {
    _organizations(params: $params) {
      organizations {
        id
        name
        display_name
        enabled
        created_at
        updated_at
      }
      pagination {
        limit
        page
        offset
        total
      }
    }
  }
`;

export const OrganizationQuery = `
  query getOrganization($params: OrganizationRequest!) {
    _organization(params: $params) {
      id
      name
      display_name
      enabled
      created_at
      updated_at
    }
  }
`;

export const UserOrganizationsQuery = `
  query getUserOrganizations($params: UserOrganizationsRequest!) {
    _user_organizations(params: $params) {
      user_organizations {
        organization {
          id
          name
          display_name
          enabled
        }
        roles
      }
      pagination {
        limit
        page
        offset
        total
      }
    }
  }
`;

export const OrgMembersQuery = `
  query getOrgMembers($params: ListOrgMembersRequest!) {
    _org_members(params: $params) {
      org_members {
        id
        org_id
        user_id
        email
        given_name
        family_name
        roles
        created_at
      }
      pagination {
        limit
        page
        offset
        total
      }
    }
  }
`;

export const OrgOIDCConnectionQuery = `
  query getOrgOIDCConnection($params: OrgOIDCConnectionRequest!) {
    _org_oidc_connection(params: $params) {
      id
      org_id
      name
      issuer_url
      sso_client_id
      scopes
      redirect_uri
      is_active
      created_at
      updated_at
    }
  }
`;

export const OrgSAMLConnectionQuery = `
  query getOrgSAMLConnection($params: OrgSAMLConnectionRequest!) {
    _org_saml_connection(params: $params) {
      id
      org_id
      name
      idp_entity_id
      idp_sso_url
      sp_entity_id
      acs_url
      attribute_mapping
      allow_idp_initiated
      is_active
      created_at
      updated_at
    }
  }
`;

export const ScimEndpointQuery = `
  query getScimEndpoint($params: ScimEndpointRequest!) {
    _scim_endpoint(params: $params) {
      id
      org_id
      enabled
      created_at
      updated_at
    }
  }
`;
