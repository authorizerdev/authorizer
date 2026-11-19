package serenedb

// migrations is SereneDB's schema, as hand-written DDL, in apply order.
//
// Rules for changing it:
//
//   - Append only. A migration that has shipped is never edited or removed:
//     databases that already applied it will not run it again, so an edit
//     silently forks the schema between old and new installs. Add a new
//     version instead.
//   - Every statement must be safe to re-run (IF NOT EXISTS / IF EXISTS).
//     Versions are recorded in the same transaction as their DDL, but two
//     instances booting at once can both start a version; idempotent DDL is
//     what makes the loser's attempt harmless.
//   - A field added to a model in internal/storage/schemas needs a migration
//     here too. TestSchemaDDLMatchesModels fails until it has one, without a
//     database, so `make test` catches the drift.
//
// Strings are declared as text. SereneDB stores every string type as text and
// reports varchar(n) as text, which is what broke GORM AutoMigrate: it saw a
// type difference on every boot and issued an ALTER COLUMN ... TYPE that
// SereneDB refuses on an indexed column. Declaring the type SereneDB actually
// has keeps the catalog and this file in agreement. Column lengths are not
// relied on by the application on any backend.
//
// Unique keys are inline table constraints, named after the GORM index they
// mirror so the parity test can match them by name. Column DEFAULTs must match
// the model tags: GORM omits a zero-valued field that has a default from the
// INSERT, so the database default is what the row actually gets.
var migrations = []migration{
	{
		version: 1,
		name:    "initial schema",
		statements: []string{
			`CREATE TABLE IF NOT EXISTS authorizer_users (
				key text,
				id text,
				email text,
				email_verified_at bigint,
				password text,
				signup_methods text,
				given_name text,
				family_name text,
				middle_name text,
				nickname text,
				gender text,
				birthdate text,
				phone_number text,
				phone_number_verified_at bigint,
				picture text,
				roles text,
				revoked_timestamp bigint,
				is_multi_factor_auth_enabled boolean,
				has_skipped_mfa_setup_at bigint,
				mfa_locked_at bigint,
				updated_at bigint,
				created_at bigint,
				app_data text,
				external_id text,
				is_active boolean DEFAULT true,
				PRIMARY KEY (id)
			)`,
			`CREATE INDEX IF NOT EXISTS idx_authorizer_users_external_id ON authorizer_users (external_id)`,
			`CREATE INDEX IF NOT EXISTS idx_authorizer_users_phone_number ON authorizer_users (phone_number)`,
			`CREATE INDEX IF NOT EXISTS idx_authorizer_users_email ON authorizer_users (email)`,

			`CREATE TABLE IF NOT EXISTS authorizer_verification_requests (
				key text,
				id text,
				token text,
				identifier text,
				expires_at bigint,
				email text,
				nonce text,
				redirect_uri text,
				created_at bigint,
				updated_at bigint,
				PRIMARY KEY (id),
				CONSTRAINT idx_email_identifier UNIQUE (identifier, email)
			)`,

			`CREATE TABLE IF NOT EXISTS authorizer_sessions (
				key text,
				id text,
				user_id text,
				user_agent text,
				ip text,
				created_at bigint,
				updated_at bigint,
				PRIMARY KEY (id)
			)`,

			`CREATE TABLE IF NOT EXISTS authorizer_envs (
				key text,
				id text,
				env_data text,
				hash text,
				encryption_key text,
				updated_at bigint,
				created_at bigint,
				PRIMARY KEY (id)
			)`,

			`CREATE TABLE IF NOT EXISTS authorizer_webhooks (
				key text,
				id text,
				event_name text,
				event_description text,
				end_point text,
				headers text,
				enabled boolean,
				created_at bigint,
				updated_at bigint,
				PRIMARY KEY (id),
				CONSTRAINT uni_authorizer_webhooks_event_name UNIQUE (event_name)
			)`,

			`CREATE TABLE IF NOT EXISTS authorizer_webhook_logs (
				key text,
				id text,
				http_status bigint,
				response text,
				request text,
				webhook_id text,
				created_at bigint,
				updated_at bigint,
				PRIMARY KEY (id)
			)`,

			`CREATE TABLE IF NOT EXISTS authorizer_email_templates (
				key text,
				id text,
				event_name text,
				subject text,
				template text,
				design text,
				created_at bigint,
				updated_at bigint,
				PRIMARY KEY (id),
				CONSTRAINT uni_authorizer_email_templates_event_name UNIQUE (event_name)
			)`,

			`CREATE TABLE IF NOT EXISTS authorizer_otps (
				key text,
				id text,
				email text,
				phone_number text,
				otp text,
				expires_at bigint,
				created_at bigint,
				updated_at bigint,
				PRIMARY KEY (id)
			)`,
			`CREATE INDEX IF NOT EXISTS idx_authorizer_otps_phone_number ON authorizer_otps (phone_number)`,
			`CREATE INDEX IF NOT EXISTS idx_authorizer_otps_email ON authorizer_otps (email)`,

			`CREATE TABLE IF NOT EXISTS authorizer_authenticators (
				key text,
				id text,
				user_id text,
				method text,
				secret text,
				recovery_codes text,
				verified_at bigint,
				created_at bigint,
				updated_at bigint,
				PRIMARY KEY (id),
				CONSTRAINT idx_authenticator_user_id_method UNIQUE (user_id, method)
			)`,

			`CREATE TABLE IF NOT EXISTS authorizer_session_tokens (
				key text,
				id text,
				user_id text,
				key_name text,
				token text,
				expires_at bigint,
				created_at bigint,
				updated_at bigint,
				PRIMARY KEY (id)
			)`,
			`CREATE INDEX IF NOT EXISTS idx_authorizer_session_tokens_expires_at ON authorizer_session_tokens (expires_at)`,
			`CREATE INDEX IF NOT EXISTS idx_session_token_user_id_key ON authorizer_session_tokens (user_id, key_name)`,

			`CREATE TABLE IF NOT EXISTS authorizer_mfa_sessions (
				key text,
				id text,
				user_id text,
				key_name text,
				expires_at bigint,
				created_at bigint,
				updated_at bigint,
				PRIMARY KEY (id)
			)`,
			`CREATE INDEX IF NOT EXISTS idx_authorizer_mfa_sessions_expires_at ON authorizer_mfa_sessions (expires_at)`,
			`CREATE INDEX IF NOT EXISTS idx_mfa_session_user_id_key ON authorizer_mfa_sessions (user_id, key_name)`,

			`CREATE TABLE IF NOT EXISTS authorizer_oauth_states (
				key text,
				id text,
				state_key text,
				state text,
				created_at bigint,
				updated_at bigint,
				PRIMARY KEY (id),
				CONSTRAINT idx_authorizer_oauth_states_state_key UNIQUE (state_key)
			)`,

			`CREATE TABLE IF NOT EXISTS authorizer_audit_logs (
				key text,
				id text,
				actor_id text,
				actor_type text,
				actor_email text,
				action text,
				resource_type text,
				resource_id text,
				ip_address text,
				user_agent text,
				metadata text,
				created_at bigint,
				PRIMARY KEY (id)
			)`,

			`CREATE TABLE IF NOT EXISTS authorizer_clients (
				key text,
				id text,
				client_id text,
				kind text,
				name text,
				description text,
				client_secret text,
				allowed_scopes text,
				redirect_uris text,
				grant_types text,
				token_endpoint_auth_method text,
				is_active boolean DEFAULT true,
				org_id text,
				created_at bigint,
				updated_at bigint,
				PRIMARY KEY (id),
				CONSTRAINT idx_authorizer_clients_client_id UNIQUE (client_id)
			)`,
			`CREATE INDEX IF NOT EXISTS idx_authorizer_clients_org_id ON authorizer_clients (org_id)`,

			`CREATE TABLE IF NOT EXISTS authorizer_trusted_issuers (
				key text,
				id text,
				kind text DEFAULT 'client_assertion_trust',
				org_id text,
				client_id text,
				name text,
				issuer_url text,
				key_source_type text,
				jwks_url text,
				expected_aud text,
				subject_claim text,
				allowed_subjects text,
				issuer_type text,
				auth_method text DEFAULT 'jwt_assertion',
				is_active boolean DEFAULT true,
				enable_token_review boolean,
				kubernetes_api_server_url text,
				spiffe_refresh_hint_seconds bigint,
				trusted_proxy_header text,
				trusted_proxy_c_id_rs text,
				sso_client_id text,
				sso_client_secret_enc text,
				sso_scopes text,
				sso_redirect_uri text,
				samlsso_url text,
				saml_id_p_cert_pem text,
				samlsp_entity_id text,
				samlacs_url text,
				saml_attribute_mapping text,
				saml_allow_id_p_initiated boolean,
				created_at bigint,
				updated_at bigint,
				PRIMARY KEY (id),
				CONSTRAINT idx_authorizer_trusted_issuers_issuer_url UNIQUE (issuer_url)
			)`,
			`CREATE INDEX IF NOT EXISTS idx_authorizer_trusted_issuers_client_id ON authorizer_trusted_issuers (client_id)`,
			`CREATE INDEX IF NOT EXISTS idx_authorizer_trusted_issuers_org_id ON authorizer_trusted_issuers (org_id)`,

			`CREATE TABLE IF NOT EXISTS authorizer_organizations (
				key text,
				id text,
				name text,
				display_name text,
				enabled boolean DEFAULT true,
				created_at bigint,
				updated_at bigint,
				PRIMARY KEY (id),
				CONSTRAINT idx_authorizer_organizations_name UNIQUE (name)
			)`,

			`CREATE TABLE IF NOT EXISTS authorizer_org_memberships (
				key text,
				id text,
				org_id text,
				user_id text,
				roles text,
				deprovisioned_at bigint,
				created_at bigint,
				updated_at bigint,
				PRIMARY KEY (id),
				CONSTRAINT idx_org_membership_org_user UNIQUE (org_id, user_id)
			)`,
			`CREATE INDEX IF NOT EXISTS idx_authorizer_org_memberships_user_id ON authorizer_org_memberships (user_id)`,

			`CREATE TABLE IF NOT EXISTS authorizer_federated_identities (
				key text,
				id text,
				org_id text,
				issuer text,
				subject text,
				user_id text,
				created_at bigint,
				updated_at bigint,
				PRIMARY KEY (id),
				CONSTRAINT idx_federated_identity_org_iss_sub UNIQUE (org_id, issuer, subject)
			)`,
			`CREATE INDEX IF NOT EXISTS idx_authorizer_federated_identities_user_id ON authorizer_federated_identities (user_id)`,

			`CREATE TABLE IF NOT EXISTS authorizer_scim_endpoints (
				key text,
				id text,
				org_id text,
				token_hash text,
				enabled boolean DEFAULT true,
				created_at bigint,
				updated_at bigint,
				PRIMARY KEY (id),
				CONSTRAINT idx_authorizer_scim_endpoints_org_id UNIQUE (org_id)
			)`,

			`CREATE TABLE IF NOT EXISTS authorizer_scim_groups (
				key text,
				id text,
				org_id text,
				display_name text,
				external_id text,
				created_at bigint,
				updated_at bigint,
				PRIMARY KEY (id)
			)`,
			`CREATE INDEX IF NOT EXISTS idx_authorizer_scim_groups_org_id ON authorizer_scim_groups (org_id)`,

			`CREATE TABLE IF NOT EXISTS authorizer_webauthn_credentials (
				key text,
				id text,
				user_id text,
				credential_id text,
				public_key text,
				sign_count bigint,
				flags bigint,
				transports text,
				aa_guid text,
				name text,
				created_at bigint,
				updated_at bigint,
				last_used_at bigint,
				PRIMARY KEY (id),
				CONSTRAINT idx_authorizer_webauthn_credentials_credential_id UNIQUE (credential_id)
			)`,
			`CREATE INDEX IF NOT EXISTS idx_authorizer_webauthn_credentials_user_id ON authorizer_webauthn_credentials (user_id)`,

			`CREATE TABLE IF NOT EXISTS authorizer_org_domains (
				key text,
				id text,
				org_id text,
				domain text,
				verified_at bigint,
				created_at bigint,
				updated_at bigint,
				PRIMARY KEY (id)
			)`,
			`CREATE INDEX IF NOT EXISTS idx_authorizer_org_domains_org_id ON authorizer_org_domains (org_id)`,

			`CREATE TABLE IF NOT EXISTS authorizer_saml_service_providers (
				key text,
				id text,
				org_id text,
				name text,
				entity_id text,
				acs_url text,
				sp_cert_pem text,
				name_id_format text,
				mapped_attributes text,
				allow_id_p_initiated boolean,
				is_active boolean DEFAULT true,
				created_at bigint,
				updated_at bigint,
				PRIMARY KEY (id)
			)`,
			`CREATE INDEX IF NOT EXISTS idx_authorizer_saml_service_providers_org_id ON authorizer_saml_service_providers (org_id)`,

			`CREATE TABLE IF NOT EXISTS authorizer_saml_id_p_keys (
				key text,
				id text,
				org_id text,
				cert_pem text,
				private_key_enc text,
				algorithm text,
				status text DEFAULT 'current',
				created_at bigint,
				updated_at bigint,
				PRIMARY KEY (id)
			)`,
			`CREATE INDEX IF NOT EXISTS idx_authorizer_saml_id_p_keys_org_id ON authorizer_saml_id_p_keys (org_id)`,
		},
	},
}
