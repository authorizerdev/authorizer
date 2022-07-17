export const LOGO_URL =
	'https://user-images.githubusercontent.com/6964334/147834043-fc384cab-e7ca-40f8-9663-38fc25fd5f3a.png';

export const TextInputType = {
	ACCESS_TOKEN_EXPIRY_TIME: 'ACCESS_TOKEN_EXPIRY_TIME',
	CLIENT_ID: 'CLIENT_ID',
	GOOGLE_CLIENT_ID: 'GOOGLE_CLIENT_ID',
	GITHUB_CLIENT_ID: 'GITHUB_CLIENT_ID',
	FACEBOOK_CLIENT_ID: 'FACEBOOK_CLIENT_ID',
	LINKEDIN_CLIENT_ID: 'LINKEDIN_CLIENT_ID',
	APPLE_CLIENT_ID: 'APPLE_CLIENT_ID',
	JWT_ROLE_CLAIM: 'JWT_ROLE_CLAIM',
	REDIS_URL: 'REDIS_URL',
	SMTP_HOST: 'SMTP_HOST',
	SMTP_PORT: 'SMTP_PORT',
	SMTP_USERNAME: 'SMTP_USERNAME',
	SENDER_EMAIL: 'SENDER_EMAIL',
	ORGANIZATION_NAME: 'ORGANIZATION_NAME',
	ORGANIZATION_LOGO: 'ORGANIZATION_LOGO',
	DATABASE_NAME: 'DATABASE_NAME',
	DATABASE_TYPE: 'DATABASE_TYPE',
	DATABASE_URL: 'DATABASE_URL',
	GIVEN_NAME: 'given_name',
	MIDDLE_NAME: 'middle_name',
	FAMILY_NAME: 'family_name',
	NICKNAME: 'nickname',
	PHONE_NUMBER: 'phone_number',
	PICTURE: 'picture',
};

export const HiddenInputType = {
	CLIENT_SECRET: 'CLIENT_SECRET',
	GOOGLE_CLIENT_SECRET: 'GOOGLE_CLIENT_SECRET',
	GITHUB_CLIENT_SECRET: 'GITHUB_CLIENT_SECRET',
	FACEBOOK_CLIENT_SECRET: 'FACEBOOK_CLIENT_SECRET',
	LINKEDIN_CLIENT_SECRET: 'LINKEDIN_CLIENT_SECRET',
	APPLE_CLIENT_SECRET: 'APPLE_CLIENT_SECRET',
	JWT_SECRET: 'JWT_SECRET',
	SMTP_PASSWORD: 'SMTP_PASSWORD',
	ADMIN_SECRET: 'ADMIN_SECRET',
	OLD_ADMIN_SECRET: 'OLD_ADMIN_SECRET',
};

export const ArrayInputType = {
	ROLES: 'ROLES',
	DEFAULT_ROLES: 'DEFAULT_ROLES',
	PROTECTED_ROLES: 'PROTECTED_ROLES',
	ALLOWED_ORIGINS: 'ALLOWED_ORIGINS',
	USER_ROLES: 'roles',
};

export const SelectInputType = {
	JWT_TYPE: 'JWT_TYPE',
	GENDER: 'gender',
};

export const TextAreaInputType = {
	CUSTOM_ACCESS_TOKEN_SCRIPT: 'CUSTOM_ACCESS_TOKEN_SCRIPT',
	JWT_PRIVATE_KEY: 'JWT_PRIVATE_KEY',
	JWT_PUBLIC_KEY: 'JWT_PUBLIC_KEY',
};

export const SwitchInputType = {
	DISABLE_LOGIN_PAGE: 'DISABLE_LOGIN_PAGE',
	DISABLE_MAGIC_LINK_LOGIN: 'DISABLE_MAGIC_LINK_LOGIN',
	DISABLE_EMAIL_VERIFICATION: 'DISABLE_EMAIL_VERIFICATION',
	DISABLE_BASIC_AUTHENTICATION: 'DISABLE_BASIC_AUTHENTICATION',
	DISABLE_SIGN_UP: 'DISABLE_SIGN_UP',
	DISABLE_REDIS_FOR_ENV: 'DISABLE_REDIS_FOR_ENV',
	DISABLE_STRONG_PASSWORD: 'DISABLE_STRONG_PASSWORD',
};

export const DateInputType = {
	BIRTHDATE: 'birthdate',
};

export const ArrayInputOperations = {
	APPEND: 'APPEND',
	REMOVE: 'REMOVE',
};

export const HMACEncryptionType = {
	HS256: 'HS256',
	HS384: 'HS384',
	HS512: 'HS512',
};

export const RSAEncryptionType = {
	RS256: 'RS256',
	RS384: 'RS384',
	RS512: 'RS512',
};

export const ECDSAEncryptionType = {
	ES256: 'ES256',
	ES384: 'ES384',
	ES512: 'ES512',
};

export interface envVarTypes {
	GOOGLE_CLIENT_ID: string;
	GOOGLE_CLIENT_SECRET: string;
	GITHUB_CLIENT_ID: string;
	GITHUB_CLIENT_SECRET: string;
	FACEBOOK_CLIENT_ID: string;
	FACEBOOK_CLIENT_SECRET: string;
	LINKEDIN_CLIENT_ID: string;
	LINKEDIN_CLIENT_SECRET: string;
	APPLE_CLIENT_ID: string;
	APPLE_CLIENT_SECRET: string;
	ROLES: [string] | [];
	DEFAULT_ROLES: [string] | [];
	PROTECTED_ROLES: [string] | [];
	JWT_TYPE: string;
	JWT_SECRET: string;
	JWT_ROLE_CLAIM: string;
	JWT_PRIVATE_KEY: string;
	JWT_PUBLIC_KEY: string;
	REDIS_URL: string;
	SMTP_HOST: string;
	SMTP_PORT: string;
	SMTP_USERNAME: string;
	SMTP_PASSWORD: string;
	SENDER_EMAIL: string;
	ALLOWED_ORIGINS: [string] | [];
	ORGANIZATION_NAME: string;
	ORGANIZATION_LOGO: string;
	CUSTOM_ACCESS_TOKEN_SCRIPT: string;
	ADMIN_SECRET: string;
	DISABLE_LOGIN_PAGE: boolean;
	DISABLE_MAGIC_LINK_LOGIN: boolean;
	DISABLE_EMAIL_VERIFICATION: boolean;
	DISABLE_BASIC_AUTHENTICATION: boolean;
	DISABLE_SIGN_UP: boolean;
	DISABLE_STRONG_PASSWORD: boolean;
	OLD_ADMIN_SECRET: string;
	DATABASE_NAME: string;
	DATABASE_TYPE: string;
	DATABASE_URL: string;
	ACCESS_TOKEN_EXPIRY_TIME: string;
}

export const envSubViews = {
	INSTANCE_INFO: 'instance-info',
	ROLES: 'roles',
	JWT_CONFIG: 'jwt-config',
	SESSION_STORAGE: 'session-storage',
	EMAIL_CONFIG: 'email-config',
	WHITELIST_VARIABLES: 'whitelist-variables',
	ORGANIZATION_INFO: 'organization-info',
	ACCESS_TOKEN: 'access-token',
	FEATURES: 'features',
	ADMIN_SECRET: 'admin-secret',
	DB_CRED: 'db-cred',
};

export enum WebhookInputDataFields {
	ID = 'id',
	EVENT_NAME = 'event_name',
	ENDPOINT = 'endpoint',
	ENABLED = 'enabled',
	HEADERS = 'headers',
}

export enum WebhookInputHeaderFields {
	KEY = 'key',
	VALUE = 'value',
}

export enum UpdateWebhookModalViews {
	ADD = 'add',
	Edit = 'edit',
}

export const pageLimits: number[] = [5, 10, 15];

export const webhookEventNames = {
	USER_SIGNUP: 'user.signup',
	USER_CREATED: 'user.created',
	USER_LOGIN: 'user.login',
	USER_DELETED: 'user.deleted',
	USER_ACCESS_ENABLED: 'user.access_enabled',
	USER_ACCESS_REVOKED: 'user.access_revoked',
};
