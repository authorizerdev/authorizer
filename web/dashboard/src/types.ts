export interface User {
	id: string;
	email: string;
	email_verified: boolean;
	given_name?: string;
	family_name?: string;
	middle_name?: string;
	nickname?: string;
	gender?: string;
	birthdate?: string;
	phone_number?: string;
	phone_number_verified?: boolean;
	picture?: string;
	signup_methods: string;
	roles: string[];
	created_at: number;
	updated_at?: number;
	revoked_timestamp?: number;
	is_multi_factor_auth_enabled?: boolean;
	preferred_username?: string;
}

export interface Webhook {
	id: string;
	event_name: string;
	event_description?: string;
	endpoint: string;
	enabled: boolean;
	headers?: Record<string, string>;
}

export interface WebhookLog {
	id: string;
	http_status: number;
	request: string;
	response: string;
	webhook_id: string;
	created_at: number;
}

export interface EmailTemplate {
	id: string;
	event_name: string;
	subject: string;
	template: string;
	design: string;
	created_at: number;
	updated_at?: number;
}

export interface AuditLog {
	id: string;
	actor_id: string;
	actor_type: string;
	actor_email: string;
	action: string;
	resource_type: string;
	resource_id: string;
	ip_address: string;
	user_agent: string;
	metadata: string;
	created_at: number;
}

export interface PaginationInfo {
	offset: number;
	total: number;
	page: number;
	limit: number;
}

export interface Pagination {
	pagination: PaginationInfo;
}

export interface UsersResponse {
	_users: {
		pagination: PaginationInfo;
		users: User[];
	};
}

export interface WebhooksResponse {
	_webhooks: {
		pagination: PaginationInfo;
		webhooks: Webhook[];
	};
}

export interface EmailTemplatesResponse {
	_email_templates: {
		pagination: PaginationInfo;
		email_templates: EmailTemplate[];
	};
}

export interface WebhookLogsResponse {
	_webhook_logs: {
		pagination: PaginationInfo;
		webhook_logs: WebhookLog[];
	};
}

export interface AuditLogsResponse {
	_audit_logs: {
		pagination: PaginationInfo;
		audit_logs: AuditLog[];
	};
}

export interface MetaResponse {
	meta: {
		version: string;
		client_id: string;
	};
}

export interface AdminSessionResponse {
	_admin_session: {
		message: string;
	};
}
