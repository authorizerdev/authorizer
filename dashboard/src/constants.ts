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
	DISABLE_MULTI_FACTOR_AUTHENTICATION: 'DISABLE_MULTI_FACTOR_AUTHENTICATION',
	ENFORCE_MULTI_FACTOR_AUTHENTICATION: 'ENFORCE_MULTI_FACTOR_AUTHENTICATION',
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
	DISABLE_MULTI_FACTOR_AUTHENTICATION: boolean;
	ENFORCE_MULTI_FACTOR_AUTHENTICATION: boolean;
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

export enum EmailTemplateInputDataFields {
	ID = 'id',
	EVENT_NAME = 'event_name',
	SUBJECT = 'subject',
	CREATED_AT = 'created_at',
	TEMPLATE = 'template',
	DESIGN = 'design',
}

export enum WebhookInputHeaderFields {
	KEY = 'key',
	VALUE = 'value',
}

export enum UpdateModalViews {
	ADD = 'add',
	Edit = 'edit',
}

export const pageLimits: number[] = [5, 10, 15];

export const webhookEventNames = {
	'User signup': 'user.signup',
	'User created': 'user.created',
	'User login': 'user.login',
	'User deleted': 'user.deleted',
	'User access enabled': 'user.access_enabled',
	'User access revoked': 'user.access_revoked',
};

export const emailTemplateEventNames = {
	Signup: 'basic_auth_signup',
	'Magic Link Login': 'magic_link_login',
	'Update Email': 'update_email',
	'Forgot Password': 'forgot_password',
	'Verify Otp': 'verify_otp',
	'Invite member': 'invite_member',
};

export enum webhookVerifiedStatus {
	VERIFIED = 'verified',
	NOT_VERIFIED = 'not_verified',
	PENDING = 'verification_pending',
}

export const emailTemplateVariables = {
	'user.id': {
		description: `User identifier`,
		value: '{.user.id}}',
	},
	'user.email': {
		description: 'User email address',
		value: '{.user.email}}',
	},
	'user.given_name': {
		description: `User first name`,
		value: '{.user.given_name}}',
	},
	'user.family_name': {
		description: `User last name`,
		value: '{.user.family_name}}',
	},
	'user.middle_name': {
		description: `Middle name of user`,
		value: '{.user.middle_name}}',
	},
	'user.nickname': {
		description: `Nick name of user`,
		value: '{.user.nickname}}',
	},
	'user.preferred_username': {
		description: `Username, by default it is email`,
		value: '{.user.preferred_username}}',
	},
	'user.signup_methods': {
		description: `Comma separated list of methods using which user has signed up`,
		value: '{.user.signup_methods}}',
	},
	'user.email_verified': {
		description: `Whether email is verified or not`,
		value: '{.user.email_verified}}',
	},
	'user.picture': {
		description: `URL of the user profile picture`,
		value: '{.user.picture}}',
	},
	'user.roles': {
		description: `Comma separated list of roles assigned to user`,
		value: '{.user.roles}}',
	},
	'user.gender': {
		description: `Gender of user`,
		value: '{.user.gender}}',
	},
	'user.birthdate': {
		description: `BirthDate of user`,
		value: '{.user.birthdate}}',
	},
	'user.phone_number': {
		description: `Phone number of user`,
		value: '{.user.phone_number}}',
	},
	'user.phone_number_verified': {
		description: `Whether phone number is verified or not`,
		value: '{.user.phone_number_verified}}',
	},
	'user.created_at': {
		description: `User created at time`,
		value: '{.user.created_at}}',
	},
	'user.updated_at': {
		description: `Last updated time at user`,
		value: '{.user.updated_at}}',
	},
	'organization.name': {
		description: `Organization name`,
		value: '{.organization.name}}',
	},
	'organization.logo': {
		description: `Organization logo`,
		value: '{.organization.logo}}',
	},
	verification_url: {
		description: `Verification URL in case of events other than verify otp`,
		value: '{.verification_url}}',
	},
	otp: {
		description: `OTP sent during login with Multi factor authentication`,
		value: '{.otp}}',
	},
};

export const webhookPayloadExample: string = `{
	"event_name":"user.login",
	"user":{
	   "birthdate":null,
	   "created_at":1657524721,
	   "email":"lakhan.m.samani@gmail.com",
	   "email_verified":true,
	   "family_name":"Samani",
	   "gender":null,
	   "given_name":"Lakhan",
	   "id":"466d0b31-1b87-420e-bea5-09d05d79c586",
	   "middle_name":null,
	   "nickname":null,
	   "phone_number":null,
	   "phone_number_verified":false,
	   "picture":"https://lh3.googleusercontent.com/a-/AFdZucppvU6a2zIDkX0wvhhapVjT0ZMKDlYCkQDi3NxcUg=s96-c",
	   "preferred_username":"lakhan.m.samani@gmail.com",
	   "revoked_timestamp":null,
	   "roles":[
		  "user"
	   ],
	   "signup_methods":"google",
	   "updated_at":1657526492
	},
	"auth_recipe":"google"
 }`;
