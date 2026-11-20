package surrealdb

// migrations is the SurrealDB schema, in apply order.
//
// Rules for changing it:
//
//   - Append only. A shipped migration is never edited or removed: databases
//     that applied it will not run it again. Add a new version instead.
//   - Every statement must be safe to re-run (IF NOT EXISTS / IF EXISTS), so a
//     version two instances start at once is harmless.
//   - Indexes mirror the SQL models' GORM index tags exactly, under the same
//     names, so lookups and uniqueness behave the same on every backend.
//     TestSchemaMatchesSQLIndexes fails, with no database, until a model index
//     added on the SQL side has its SurrealDB migration.
//
// Tables are SCHEMALESS: field types come from the Go structs, as with the
// other document-store providers.
var migrations = []migration{
	{
		version: 1,
		name:    "initial schema",
		statements: []string{
			`DEFINE TABLE IF NOT EXISTS authorizer_users SCHEMALESS`,
			`DEFINE INDEX IF NOT EXISTS idx_authorizer_users_email ON authorizer_users FIELDS email`,
			`DEFINE INDEX IF NOT EXISTS idx_authorizer_users_phone_number ON authorizer_users FIELDS phone_number`,
			`DEFINE INDEX IF NOT EXISTS idx_authorizer_users_external_id ON authorizer_users FIELDS external_id`,

			`DEFINE TABLE IF NOT EXISTS authorizer_verification_requests SCHEMALESS`,
			`DEFINE INDEX IF NOT EXISTS idx_email_identifier ON authorizer_verification_requests FIELDS identifier, email UNIQUE`,

			`DEFINE TABLE IF NOT EXISTS authorizer_sessions SCHEMALESS`,
			`DEFINE TABLE IF NOT EXISTS authorizer_envs SCHEMALESS`,

			`DEFINE TABLE IF NOT EXISTS authorizer_webhooks SCHEMALESS`,
			`DEFINE INDEX IF NOT EXISTS uni_authorizer_webhooks_event_name ON authorizer_webhooks FIELDS event_name UNIQUE`,

			`DEFINE TABLE IF NOT EXISTS authorizer_webhook_logs SCHEMALESS`,

			`DEFINE TABLE IF NOT EXISTS authorizer_email_templates SCHEMALESS`,
			`DEFINE INDEX IF NOT EXISTS uni_authorizer_email_templates_event_name ON authorizer_email_templates FIELDS event_name UNIQUE`,

			`DEFINE TABLE IF NOT EXISTS authorizer_otps SCHEMALESS`,
			`DEFINE INDEX IF NOT EXISTS idx_authorizer_otps_email ON authorizer_otps FIELDS email`,
			`DEFINE INDEX IF NOT EXISTS idx_authorizer_otps_phone_number ON authorizer_otps FIELDS phone_number`,

			`DEFINE TABLE IF NOT EXISTS authorizer_authenticators SCHEMALESS`,
			`DEFINE INDEX IF NOT EXISTS idx_authenticator_user_id_method ON authorizer_authenticators FIELDS user_id, method UNIQUE`,

			`DEFINE TABLE IF NOT EXISTS authorizer_session_tokens SCHEMALESS`,
			`DEFINE INDEX IF NOT EXISTS idx_session_token_user_id_key ON authorizer_session_tokens FIELDS user_id, key_name`,
			`DEFINE INDEX IF NOT EXISTS idx_authorizer_session_tokens_expires_at ON authorizer_session_tokens FIELDS expires_at`,

			`DEFINE TABLE IF NOT EXISTS authorizer_mfa_sessions SCHEMALESS`,
			`DEFINE INDEX IF NOT EXISTS idx_mfa_session_user_id_key ON authorizer_mfa_sessions FIELDS user_id, key_name`,
			`DEFINE INDEX IF NOT EXISTS idx_authorizer_mfa_sessions_expires_at ON authorizer_mfa_sessions FIELDS expires_at`,

			`DEFINE TABLE IF NOT EXISTS authorizer_oauth_states SCHEMALESS`,
			`DEFINE INDEX IF NOT EXISTS idx_authorizer_oauth_states_state_key ON authorizer_oauth_states FIELDS state_key UNIQUE`,

			`DEFINE TABLE IF NOT EXISTS authorizer_audit_logs SCHEMALESS`,

			`DEFINE TABLE IF NOT EXISTS authorizer_clients SCHEMALESS`,
			`DEFINE INDEX IF NOT EXISTS idx_authorizer_clients_client_id ON authorizer_clients FIELDS client_id UNIQUE`,
			`DEFINE INDEX IF NOT EXISTS idx_authorizer_clients_org_id ON authorizer_clients FIELDS org_id`,

			`DEFINE TABLE IF NOT EXISTS authorizer_trusted_issuers SCHEMALESS`,
			`DEFINE INDEX IF NOT EXISTS idx_authorizer_trusted_issuers_org_id ON authorizer_trusted_issuers FIELDS org_id`,
			`DEFINE INDEX IF NOT EXISTS idx_authorizer_trusted_issuers_client_id ON authorizer_trusted_issuers FIELDS client_id`,
			`DEFINE INDEX IF NOT EXISTS idx_authorizer_trusted_issuers_issuer_url ON authorizer_trusted_issuers FIELDS issuer_url UNIQUE`,

			`DEFINE TABLE IF NOT EXISTS authorizer_organizations SCHEMALESS`,
			`DEFINE INDEX IF NOT EXISTS idx_authorizer_organizations_name ON authorizer_organizations FIELDS name UNIQUE`,

			`DEFINE TABLE IF NOT EXISTS authorizer_org_memberships SCHEMALESS`,
			`DEFINE INDEX IF NOT EXISTS idx_org_membership_org_user ON authorizer_org_memberships FIELDS org_id, user_id UNIQUE`,
			`DEFINE INDEX IF NOT EXISTS idx_authorizer_org_memberships_user_id ON authorizer_org_memberships FIELDS user_id`,

			`DEFINE TABLE IF NOT EXISTS authorizer_federated_identities SCHEMALESS`,
			`DEFINE INDEX IF NOT EXISTS idx_federated_identity_org_iss_sub ON authorizer_federated_identities FIELDS org_id, issuer, subject UNIQUE`,
			`DEFINE INDEX IF NOT EXISTS idx_authorizer_federated_identities_user_id ON authorizer_federated_identities FIELDS user_id`,

			`DEFINE TABLE IF NOT EXISTS authorizer_scim_endpoints SCHEMALESS`,
			`DEFINE INDEX IF NOT EXISTS idx_authorizer_scim_endpoints_org_id ON authorizer_scim_endpoints FIELDS org_id UNIQUE`,

			`DEFINE TABLE IF NOT EXISTS authorizer_scim_groups SCHEMALESS`,
			`DEFINE INDEX IF NOT EXISTS idx_authorizer_scim_groups_org_id ON authorizer_scim_groups FIELDS org_id`,

			`DEFINE TABLE IF NOT EXISTS authorizer_webauthn_credentials SCHEMALESS`,
			`DEFINE INDEX IF NOT EXISTS idx_authorizer_webauthn_credentials_user_id ON authorizer_webauthn_credentials FIELDS user_id`,
			`DEFINE INDEX IF NOT EXISTS idx_authorizer_webauthn_credentials_credential_id ON authorizer_webauthn_credentials FIELDS credential_id UNIQUE`,

			`DEFINE TABLE IF NOT EXISTS authorizer_org_domains SCHEMALESS`,
			`DEFINE INDEX IF NOT EXISTS idx_authorizer_org_domains_org_id ON authorizer_org_domains FIELDS org_id`,

			`DEFINE TABLE IF NOT EXISTS authorizer_saml_service_providers SCHEMALESS`,
			`DEFINE INDEX IF NOT EXISTS idx_authorizer_saml_service_providers_org_id ON authorizer_saml_service_providers FIELDS org_id`,

			`DEFINE TABLE IF NOT EXISTS authorizer_saml_id_p_keys SCHEMALESS`,
			`DEFINE INDEX IF NOT EXISTS idx_authorizer_saml_id_p_keys_org_id ON authorizer_saml_id_p_keys FIELDS org_id`,
		},
	},
}
