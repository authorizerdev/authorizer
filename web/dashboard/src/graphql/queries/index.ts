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

export const FgaCheckQuery = `
  query fgaCheck($params: FgaCheckInput!) {
    fga_check(params: $params) {
      allowed
    }
  }
`;

// AdminRolesQuery fetches the instance's configured roles (admin-only _env) so
// the authorization-model builder can offer a template using the real roles.
export const AdminRolesQuery = `
  query adminRoles {
    _env {
      ROLES
    }
  }
`;
