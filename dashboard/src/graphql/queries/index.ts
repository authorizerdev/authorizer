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
    }
  }
`;
