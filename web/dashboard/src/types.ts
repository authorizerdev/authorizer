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

export interface FgaModel {
	id: string;
	dsl: string;
}

export interface FgaTuple {
	user: string;
	relation: string;
	object: string;
}

export interface FgaGetModelResponse {
	_fga_get_model: FgaModel;
}

export interface FgaWriteModelResponse {
	_fga_write_model: FgaModel;
}

export interface FgaReadTuplesResponse {
	_fga_read_tuples: {
		tuples: FgaTuple[];
		continuation_token?: string | null;
	};
}

export interface FgaWriteTuplesResponse {
	_fga_write_tuples: {
		message: string;
	};
}

export interface FgaDeleteTuplesResponse {
	_fga_delete_tuples: {
		message: string;
	};
}

export interface FgaResetResponse {
	_fga_reset: {
		message: string;
	};
}

// Permission is one (object, relation) pair a subject holds.
export interface Permission {
	object: string;
	relation: string;
}

export interface ListPermissionsResponse {
	list_permissions: {
		objects: string[];
		permissions: Permission[];
		// True when the server capped the result set and more permissions exist.
		truncated: boolean;
	};
}

export interface AdminRolesResponse {
	_admin_meta: {
		roles?: string[] | null;
	} | null;
}

export interface Client {
	id: string;
	client_id: string;
	name: string;
	description?: string | null;
	allowed_scopes: string[];
	is_active: boolean;
	created_at?: number | null;
	updated_at?: number | null;
}

export interface ClientsResponse {
	_clients: {
		pagination: PaginationInfo;
		clients: Client[];
	};
}

export interface CreateClientResponse {
	client: Client;
	// Returned exactly once at creation/rotation; never retrievable again.
	client_secret: string;
}

export interface TrustedIssuer {
	id: string;
	service_account_id: string;
	name: string;
	issuer_url: string;
	key_source_type: string;
	jwks_url?: string | null;
	expected_aud: string;
	subject_claim: string;
	allowed_subjects?: string | null;
	issuer_type: string;
	is_active: boolean;
	spiffe_refresh_hint_seconds?: number | null;
	created_at?: number | null;
	updated_at?: number | null;
}

export interface TrustedIssuersResponse {
	_trusted_issuers: {
		pagination: PaginationInfo;
		trusted_issuers: TrustedIssuer[];
	};
}

export interface Organization {
	id: string;
	// name is a unique, URL-safe slug identifying the organization.
	name: string;
	display_name?: string | null;
	enabled: boolean;
	created_at?: number | null;
	updated_at?: number | null;
}

export interface OrganizationsResponse {
	_organizations: {
		pagination: PaginationInfo;
		organizations: Organization[];
	};
}

export interface OrgMember {
	id: string;
	org_id: string;
	user_id: string;
	roles: string[];
	created_at?: number | null;
	updated_at?: number | null;
}

export interface OrgMembersResponse {
	_org_members: {
		pagination: PaginationInfo;
		org_members: OrgMember[];
	};
}

export interface UserOrganization {
	organization: Organization;
	roles: string[];
}

export interface UserOrganizationsResponse {
	_user_organizations: {
		pagination: PaginationInfo;
		user_organizations: UserOrganization[];
	};
}

export interface OrgOIDCConnection {
	id: string;
	org_id: string;
	name: string;
	issuer_url: string;
	sso_client_id: string;
	scopes?: string | null;
	redirect_uri?: string | null;
	is_active: boolean;
	created_at?: number | null;
	updated_at?: number | null;
}

export interface OrgSAMLConnection {
	id: string;
	org_id: string;
	name: string;
	idp_entity_id: string;
	idp_sso_url?: string | null;
	sp_entity_id?: string | null;
	acs_url?: string | null;
	attribute_mapping?: string | null;
	allow_idp_initiated: boolean;
	is_active: boolean;
	created_at?: number | null;
	updated_at?: number | null;
}

export interface ScimEndpoint {
	id: string;
	org_id: string;
	enabled: boolean;
	created_at?: number | null;
	updated_at?: number | null;
}

export interface CreateScimEndpointResponse {
	scim_endpoint: ScimEndpoint;
	// Returned exactly once at creation/rotation; never retrievable again.
	token: string;
}
