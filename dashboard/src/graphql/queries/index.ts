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

export const EnvVariablesQuery = `
  query {
    _env{
      CLIENT_ID,
      CLIENT_SECRET,
	    GOOGLE_CLIENT_ID,
      GOOGLE_CLIENT_SECRET,
      GITHUB_CLIENT_ID,
      GITHUB_CLIENT_SECRET,
      FACEBOOK_CLIENT_ID,
      FACEBOOK_CLIENT_SECRET,
      LINKEDIN_CLIENT_ID,
      LINKEDIN_CLIENT_SECRET,
      APPLE_CLIENT_ID,
      APPLE_CLIENT_SECRET,
      DEFAULT_ROLES,
      PROTECTED_ROLES,
      ROLES,
      JWT_TYPE,
      JWT_SECRET,
      JWT_ROLE_CLAIM,
      JWT_PRIVATE_KEY,
      JWT_PUBLIC_KEY,
      REDIS_URL,
      SMTP_HOST,
      SMTP_PORT,
      SMTP_USERNAME,
      SMTP_PASSWORD,
      SENDER_EMAIL,
      ALLOWED_ORIGINS,
      ORGANIZATION_NAME,
      ORGANIZATION_LOGO,
      ADMIN_SECRET,
      DISABLE_LOGIN_PAGE,
      DISABLE_MAGIC_LINK_LOGIN,
      DISABLE_EMAIL_VERIFICATION,
      DISABLE_BASIC_AUTHENTICATION,
      DISABLE_SIGN_UP,
      DISABLE_STRONG_PASSWORD,
      DISABLE_REDIS_FOR_ENV,
      CUSTOM_ACCESS_TOKEN_SCRIPT,
      DATABASE_NAME,
      DATABASE_TYPE,
      DATABASE_URL,
      ACCESS_TOKEN_EXPIRY_TIME,
    }
  }
`;

export const UserDetailsQuery = `
  query($params: PaginatedInput) {
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

export const EmailVerificationQuery = `
  query {
    _env{
      DISABLE_EMAIL_VERIFICATION
    }
  }
`;

export const WebhooksDataQuery = `
  query getWebhooksData($params: PaginatedInput!) {
    _webhooks(params: $params){
      webhooks{
        id
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
  query getEmailTemplates($params: PaginatedInput!) {
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
