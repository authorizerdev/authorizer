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
      ROLES,
      DEFAULT_ROLES,
      PROTECTED_ROLES,
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
      CUSTOM_ACCESS_TOKEN_SCRIPT,
      DATABASE_NAME,
      DATABASE_TYPE,
      DATABASE_URL,
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
