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
  query($params: PaginatedRequest) {
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

// Authorization queries
export const ResourcesQuery = `
  query getResources($params: PaginatedRequest) {
    _resources(params: $params) {
      resources {
        id
        name
        description
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

export const ScopesQuery = `
  query getScopes($params: PaginatedRequest) {
    _scopes(params: $params) {
      scopes {
        id
        name
        description
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

export const PoliciesQuery = `
  query getPolicies($params: PaginatedRequest) {
    _policies(params: $params) {
      policies {
        id
        name
        description
        type
        logic
        decision_strategy
        targets {
          id
          target_type
          target_value
        }
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

export const PermissionsQuery = `
  query getPermissions($params: PaginatedRequest) {
    _permissions(params: $params) {
      permissions {
        id
        name
        description
        resource {
          id
          name
          description
        }
        scopes {
          id
          name
          description
        }
        policies {
          id
          name
          type
          logic
          decision_strategy
          targets {
            id
            target_type
            target_value
          }
        }
        decision_strategy
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

export const CheckPermissionQuery = `
  query checkPermission($params: CheckPermissionInput!) {
    check_permission(params: $params) {
      allowed
      matched_policy
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
