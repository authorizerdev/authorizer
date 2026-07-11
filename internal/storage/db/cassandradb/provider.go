// Package cassandradb implements the storage provider backed by Cassandra/ScyllaDB.
package cassandradb

import (
	"crypto/tls"
	"crypto/x509"
	"encoding/json"
	"fmt"
	"reflect"
	"strings"
	"time"

	cansandraDriver "github.com/gocql/gocql"
	"github.com/rs/zerolog"

	"github.com/authorizerdev/authorizer/internal/config"
	"github.com/authorizerdev/authorizer/internal/crypto"
	"github.com/authorizerdev/authorizer/internal/storage/schemas"
)

// Dependencies struct the cassandradb data store provider
type Dependencies struct {
	Log *zerolog.Logger
}

type provider struct {
	config       *config.Config
	dependencies *Dependencies
	db           *cansandraDriver.Session
}

// KeySpace for the cassandra database
var KeySpace string

// NewProvider to initialize arangodb connection
func NewProvider(cfg *config.Config, deps *Dependencies) (*provider, error) {
	dbURL := cfg.DatabaseURL
	if dbURL == "" {
		dbHost := cfg.DatabaseHost
		dbPort := cfg.DatabasePort
		if dbPort != 0 && dbHost != "" {
			dbURL = fmt.Sprintf("%s:%d", dbHost, dbPort)
		} else if dbHost != "" {
			dbURL = dbHost
		}
	}

	KeySpace = cfg.DatabaseName
	if KeySpace == "" {
		return nil, fmt.Errorf("database name is required for cassandra. It is used as keyspace in case of cassandra")
	}
	clusterURL := []string{}
	if strings.Contains(dbURL, ",") {
		clusterURL = strings.Split(dbURL, ",")
	} else {
		clusterURL = append(clusterURL, dbURL)
	}
	cassandraClient := cansandraDriver.NewCluster(clusterURL...)
	dbUsername := cfg.DatabaseUsername
	dbPassword := cfg.DatabasePassword

	if dbUsername != "" && dbPassword != "" {
		cassandraClient.Authenticator = &cansandraDriver.PasswordAuthenticator{
			Username: dbUsername,
			Password: dbPassword,
		}
	}

	dbCert := cfg.DatabaseCert
	dbCACert := cfg.DatabaseCACert
	dbCertKey := cfg.DatabaseCertKey
	if dbCert != "" && dbCACert != "" && dbCertKey != "" {
		certString, err := crypto.DecryptB64(dbCert)
		if err != nil {
			return nil, err
		}

		keyString, err := crypto.DecryptB64(dbCertKey)
		if err != nil {
			return nil, err
		}

		caString, err := crypto.DecryptB64(dbCACert)
		if err != nil {
			return nil, err
		}

		cert, err := tls.X509KeyPair([]byte(certString), []byte(keyString))
		if err != nil {
			return nil, err
		}

		caCertPool := x509.NewCertPool()
		caCertPool.AppendCertsFromPEM([]byte(caString))

		cassandraClient.SslOpts = &cansandraDriver.SslOptions{
			Config: &tls.Config{
				Certificates: []tls.Certificate{cert},
				RootCAs:      caCertPool,
			},
			EnableHostVerification: true,
		}
	}

	cassandraClient.RetryPolicy = &cansandraDriver.SimpleRetryPolicy{
		NumRetries: 3,
	}
	cassandraClient.Consistency = cansandraDriver.LocalQuorum
	cassandraClient.ConnectTimeout = 10 * time.Second
	cassandraClient.ProtoVersion = 4
	cassandraClient.Timeout = 30 * time.Second

	session, err := cassandraClient.CreateSession()
	if err != nil {
		return nil, err
	}

	// Note for astra keyspaces can only be created from there console
	// https://docs.datastax.com/en/astra/docs/datastax-astra-faq.html#_i_am_trying_to_create_a_keyspace_in_the_cql_shell_and_i_am_running_into_an_error_how_do_i_fix_this
	getKeyspaceQuery := "SELECT keyspace_name FROM system_schema.keyspaces;"
	scanner := session.Query(getKeyspaceQuery).Iter().Scanner()
	hasAuthorizerKeySpace := false
	for scanner.Next() {
		var keySpace string
		err := scanner.Scan(&keySpace)
		if err != nil {
			return nil, err
		}
		if keySpace == KeySpace {
			hasAuthorizerKeySpace = true
			break
		}
	}

	if !hasAuthorizerKeySpace {
		createKeySpaceQuery := fmt.Sprintf("CREATE KEYSPACE %s WITH REPLICATION = {'class': 'SimpleStrategy', 'replication_factor': 1};", KeySpace)
		err = session.Query(createKeySpaceQuery).Exec()
		if err != nil {
			return nil, err
		}
	}

	// make sure collections are present
	envCollectionQuery := fmt.Sprintf("CREATE TABLE IF NOT EXISTS %s.%s (id text, env text, hash text, updated_at bigint, created_at bigint, PRIMARY KEY (id))",
		KeySpace, schemas.Collections.Env)
	err = session.Query(envCollectionQuery).Exec()
	if err != nil {
		return nil, err
	}

	sessionCollectionQuery := fmt.Sprintf("CREATE TABLE IF NOT EXISTS %s.%s (id text, user_id text, user_agent text, ip text, updated_at bigint, created_at bigint, PRIMARY KEY (id))", KeySpace, schemas.Collections.Session)
	err = session.Query(sessionCollectionQuery).Exec()
	if err != nil {
		return nil, err
	}
	sessionIndexQuery := fmt.Sprintf("CREATE INDEX IF NOT EXISTS authorizer_session_user_id ON %s.%s (user_id)", KeySpace, schemas.Collections.Session)
	err = session.Query(sessionIndexQuery).Exec()
	if err != nil {
		return nil, err
	}

	userCollectionQuery := fmt.Sprintf("CREATE TABLE IF NOT EXISTS %s.%s (id text, email text, email_verified_at bigint, password text, signup_methods text, given_name text, family_name text, middle_name text, nickname text, gender text, birthdate text, phone_number text, phone_number_verified_at bigint, picture text, roles text, updated_at bigint, created_at bigint, revoked_timestamp bigint, external_id text, is_active boolean, PRIMARY KEY (id))", KeySpace, schemas.Collections.User)
	err = session.Query(userCollectionQuery).Exec()
	if err != nil {
		return nil, err
	}
	userIndexQuery := fmt.Sprintf("CREATE INDEX IF NOT EXISTS authorizer_user_email ON %s.%s (email)", KeySpace, schemas.Collections.User)
	err = session.Query(userIndexQuery).Exec()
	if err != nil {
		return nil, err
	}

	userPhoneNumberIndexQuery := fmt.Sprintf("CREATE INDEX IF NOT EXISTS authorizer_user_phone_number ON %s.%s (phone_number)", KeySpace, schemas.Collections.User)
	err = session.Query(userPhoneNumberIndexQuery).Exec()
	if err != nil {
		return nil, err
	}
	// add is_multi_factor_auth_enabled on users table
	userTableAlterQuery := fmt.Sprintf(`ALTER TABLE %s.%s ADD is_multi_factor_auth_enabled boolean`, KeySpace, schemas.Collections.User)
	err = session.Query(userTableAlterQuery).Exec()
	if err != nil {
		deps.Log.Debug().Err(err).Msg("Failed to alter table as is_multi_factor_auth_enabled column exists")
		// continue
	}
	// add external_id and is_active on users table (SCIM provisioning)
	userExternalIDAlterQuery := fmt.Sprintf(`ALTER TABLE %s.%s ADD external_id text`, KeySpace, schemas.Collections.User)
	err = session.Query(userExternalIDAlterQuery).Exec()
	if err != nil {
		deps.Log.Debug().Err(err).Msg("Failed to alter table as external_id column exists")
		// continue
	}
	userIsActiveAlterQuery := fmt.Sprintf(`ALTER TABLE %s.%s ADD is_active boolean`, KeySpace, schemas.Collections.User)
	err = session.Query(userIsActiveAlterQuery).Exec()
	if err != nil {
		deps.Log.Debug().Err(err).Msg("Failed to alter table as is_active column exists")
		// continue
	}
	// external_id is looked up by GetUserByExternalID (SCIM), needs an index
	userExternalIDIndexQuery := fmt.Sprintf("CREATE INDEX IF NOT EXISTS authorizer_user_external_id ON %s.%s (external_id)", KeySpace, schemas.Collections.User)
	err = session.Query(userExternalIDIndexQuery).Exec()
	if err != nil {
		return nil, err
	}

	// token is reserved keyword in cassandra, hence we need to use jwt_token
	verificationRequestCollectionQuery := fmt.Sprintf("CREATE TABLE IF NOT EXISTS %s.%s (id text, jwt_token text, identifier text, expires_at bigint, email text, nonce text, redirect_uri text, created_at bigint, updated_at bigint, PRIMARY KEY (id))", KeySpace, schemas.Collections.VerificationRequest)
	err = session.Query(verificationRequestCollectionQuery).Exec()
	if err != nil {
		return nil, err
	}
	verificationRequestIndexQuery := fmt.Sprintf("CREATE INDEX IF NOT EXISTS authorizer_verification_request_email ON %s.%s (email)", KeySpace, schemas.Collections.VerificationRequest)
	err = session.Query(verificationRequestIndexQuery).Exec()
	if err != nil {
		return nil, err
	}
	verificationRequestIndexQuery = fmt.Sprintf("CREATE INDEX IF NOT EXISTS authorizer_verification_request_identifier ON %s.%s (identifier)", KeySpace, schemas.Collections.VerificationRequest)
	err = session.Query(verificationRequestIndexQuery).Exec()
	if err != nil {
		return nil, err
	}
	verificationRequestIndexQuery = fmt.Sprintf("CREATE INDEX IF NOT EXISTS authorizer_verification_request_jwt_token ON %s.%s (jwt_token)", KeySpace, schemas.Collections.VerificationRequest)
	err = session.Query(verificationRequestIndexQuery).Exec()
	if err != nil {
		return nil, err
	}

	webhookCollectionQuery := fmt.Sprintf("CREATE TABLE IF NOT EXISTS %s.%s (id text, event_name text, endpoint text, enabled boolean, headers text, updated_at bigint, created_at bigint, PRIMARY KEY (id))", KeySpace, schemas.Collections.Webhook)
	err = session.Query(webhookCollectionQuery).Exec()
	if err != nil {
		return nil, err
	}
	webhookIndexQuery := fmt.Sprintf("CREATE INDEX IF NOT EXISTS authorizer_webhook_event_name ON %s.%s (event_name)", KeySpace, schemas.Collections.Webhook)
	err = session.Query(webhookIndexQuery).Exec()
	if err != nil {
		return nil, err
	}
	// add event_description to webhook table
	webhookAlterQuery := fmt.Sprintf(`ALTER TABLE %s.%s ADD (event_description text);`, KeySpace, schemas.Collections.Webhook)
	err = session.Query(webhookAlterQuery).Exec()
	if err != nil {
		deps.Log.Debug().Err(err).Msg("Failed to alter table as event_description column exists")
		// continue
	}

	webhookLogCollectionQuery := fmt.Sprintf("CREATE TABLE IF NOT EXISTS %s.%s (id text, http_status bigint, response text, request text, webhook_id text,updated_at bigint, created_at bigint, PRIMARY KEY (id))", KeySpace, schemas.Collections.WebhookLog)
	err = session.Query(webhookLogCollectionQuery).Exec()
	if err != nil {
		return nil, err
	}
	webhookLogIndexQuery := fmt.Sprintf("CREATE INDEX IF NOT EXISTS authorizer_webhook_log_webhook_id ON %s.%s (webhook_id)", KeySpace, schemas.Collections.WebhookLog)
	err = session.Query(webhookLogIndexQuery).Exec()
	if err != nil {
		return nil, err
	}

	emailTemplateCollectionQuery := fmt.Sprintf("CREATE TABLE IF NOT EXISTS %s.%s (id text, event_name text, template text, updated_at bigint, created_at bigint, PRIMARY KEY (id))", KeySpace, schemas.Collections.EmailTemplate)
	err = session.Query(emailTemplateCollectionQuery).Exec()
	if err != nil {
		return nil, err
	}
	emailTemplateIndexQuery := fmt.Sprintf("CREATE INDEX IF NOT EXISTS authorizer_email_template_event_name ON %s.%s (event_name)", KeySpace, schemas.Collections.EmailTemplate)
	err = session.Query(emailTemplateIndexQuery).Exec()
	if err != nil {
		return nil, err
	}
	// add subject on email_templates table
	emailTemplateAlterQuery := fmt.Sprintf(`ALTER TABLE %s.%s ADD (subject text, design text);`, KeySpace, schemas.Collections.EmailTemplate)
	err = session.Query(emailTemplateAlterQuery).Exec()
	if err != nil {
		deps.Log.Debug().Err(err).Msg("Failed to alter table as subject & design column exists")
		// continue
	}

	otpCollection := fmt.Sprintf("CREATE TABLE IF NOT EXISTS %s.%s (id text, email text, otp text, expires_at bigint, updated_at bigint, created_at bigint, PRIMARY KEY (id))", KeySpace, schemas.Collections.OTP)
	err = session.Query(otpCollection).Exec()
	if err != nil {
		return nil, err
	}
	otpIndexQuery := fmt.Sprintf("CREATE INDEX IF NOT EXISTS authorizer_otp_email ON %s.%s (email)", KeySpace, schemas.Collections.OTP)
	err = session.Query(otpIndexQuery).Exec()
	if err != nil {
		return nil, err
	}
	// Add phone_number column to otp table
	otpAlterQuery := fmt.Sprintf(`ALTER TABLE %s.%s ADD (phone_number text);`, KeySpace, schemas.Collections.OTP)
	err = session.Query(otpAlterQuery).Exec()
	if err != nil {
		deps.Log.Debug().Err(err).Msg("Failed to alter table as phone_number column exists")
		// continue
	}
	// Add app_data column to users table
	appDataAlterQuery := fmt.Sprintf(`ALTER TABLE %s.%s ADD (app_data text);`, KeySpace, schemas.Collections.User)
	err = session.Query(appDataAlterQuery).Exec()
	if err != nil {
		deps.Log.Debug().Err(err).Msg("Failed to alter table as app_data column exists")
		// continue
	}
	// Add phone number index
	otpIndexQueryPhoneNumber := fmt.Sprintf("CREATE INDEX IF NOT EXISTS authorizer_otp_phone_number ON %s.%s (phone_number)", KeySpace, schemas.Collections.OTP)
	err = session.Query(otpIndexQueryPhoneNumber).Exec()
	if err != nil {
		return nil, err
	}
	// add authenticators table
	totpCollectionQuery := fmt.Sprintf("CREATE TABLE IF NOT EXISTS %s.%s (id text, user_id text, method text, secret text, recovery_codes text, verified_at bigint, updated_at bigint, created_at bigint, PRIMARY KEY (id))", KeySpace, schemas.Collections.Authenticators)
	err = session.Query(totpCollectionQuery).Exec()
	if err != nil {
		return nil, err
	}

	// SessionToken table and indexes
	// Note: 'token' is a reserved keyword in CQL, so we use 'token_value' as the column name
	sessionTokenCollectionQuery := fmt.Sprintf("CREATE TABLE IF NOT EXISTS %s.%s (id text, user_id text, key_name text, token_value text, expires_at bigint, updated_at bigint, created_at bigint, PRIMARY KEY (id))", KeySpace, schemas.Collections.SessionToken)
	err = session.Query(sessionTokenCollectionQuery).Exec()
	if err != nil {
		return nil, err
	}
	sessionTokenIndex1 := fmt.Sprintf("CREATE INDEX IF NOT EXISTS authorizer_session_token_user_id ON %s.%s (user_id)", KeySpace, schemas.Collections.SessionToken)
	err = session.Query(sessionTokenIndex1).Exec()
	if err != nil {
		return nil, err
	}
	sessionTokenIndex1b := fmt.Sprintf("CREATE INDEX IF NOT EXISTS authorizer_session_token_key_name ON %s.%s (key_name)", KeySpace, schemas.Collections.SessionToken)
	err = session.Query(sessionTokenIndex1b).Exec()
	if err != nil {
		return nil, err
	}
	sessionTokenIndex2 := fmt.Sprintf("CREATE INDEX IF NOT EXISTS authorizer_session_token_expires_at ON %s.%s (expires_at)", KeySpace, schemas.Collections.SessionToken)
	err = session.Query(sessionTokenIndex2).Exec()
	if err != nil {
		return nil, err
	}

	// MFASession table and indexes
	mfaSessionCollectionQuery := fmt.Sprintf("CREATE TABLE IF NOT EXISTS %s.%s (id text, user_id text, key_name text, expires_at bigint, updated_at bigint, created_at bigint, PRIMARY KEY (id))", KeySpace, schemas.Collections.MFASession)
	err = session.Query(mfaSessionCollectionQuery).Exec()
	if err != nil {
		return nil, err
	}
	mfaSessionIndex1 := fmt.Sprintf("CREATE INDEX IF NOT EXISTS authorizer_mfa_session_user_id ON %s.%s (user_id)", KeySpace, schemas.Collections.MFASession)
	err = session.Query(mfaSessionIndex1).Exec()
	if err != nil {
		return nil, err
	}
	mfaSessionIndex1b := fmt.Sprintf("CREATE INDEX IF NOT EXISTS authorizer_mfa_session_key_name ON %s.%s (key_name)", KeySpace, schemas.Collections.MFASession)
	err = session.Query(mfaSessionIndex1b).Exec()
	if err != nil {
		return nil, err
	}
	mfaSessionIndex2 := fmt.Sprintf("CREATE INDEX IF NOT EXISTS authorizer_mfa_session_expires_at ON %s.%s (expires_at)", KeySpace, schemas.Collections.MFASession)
	err = session.Query(mfaSessionIndex2).Exec()
	if err != nil {
		return nil, err
	}

	// OAuthState table and indexes
	oauthStateCollectionQuery := fmt.Sprintf("CREATE TABLE IF NOT EXISTS %s.%s (id text, state_key text, state text, updated_at bigint, created_at bigint, PRIMARY KEY (id))", KeySpace, schemas.Collections.OAuthState)
	err = session.Query(oauthStateCollectionQuery).Exec()
	if err != nil {
		return nil, err
	}
	oauthStateIndex := fmt.Sprintf("CREATE INDEX IF NOT EXISTS authorizer_oauth_state_key ON %s.%s (state_key)", KeySpace, schemas.Collections.OAuthState)
	err = session.Query(oauthStateIndex).Exec()
	if err != nil {
		return nil, err
	}

	// AuditLog table and indexes
	auditLogCollectionQuery := fmt.Sprintf("CREATE TABLE IF NOT EXISTS %s.%s (id text, actor_id text, actor_type text, actor_email text, action text, resource_type text, resource_id text, ip_address text, user_agent text, metadata text, created_at bigint, PRIMARY KEY (id))", KeySpace, schemas.Collections.AuditLog)
	err = session.Query(auditLogCollectionQuery).Exec()
	if err != nil {
		return nil, err
	}
	auditLogActorIdIndex := fmt.Sprintf("CREATE INDEX IF NOT EXISTS authorizer_audit_log_actor_id ON %s.%s (actor_id)", KeySpace, schemas.Collections.AuditLog)
	err = session.Query(auditLogActorIdIndex).Exec()
	if err != nil {
		return nil, err
	}
	auditLogActionIndex := fmt.Sprintf("CREATE INDEX IF NOT EXISTS authorizer_audit_log_action ON %s.%s (action)", KeySpace, schemas.Collections.AuditLog)
	err = session.Query(auditLogActionIndex).Exec()
	if err != nil {
		return nil, err
	}
	auditLogTimestampIndex := fmt.Sprintf("CREATE INDEX IF NOT EXISTS authorizer_audit_log_created_at ON %s.%s (created_at)", KeySpace, schemas.Collections.AuditLog)
	err = session.Query(auditLogTimestampIndex).Exec()
	if err != nil {
		return nil, err
	}
	// ScyllaDB builds secondary indexes asynchronously. Poll with a probe query
	// that requires the actor_id index until it succeeds instead of a fixed sleep.
	waitForCassandraIndexes(session, KeySpace, schemas.Collections.AuditLog, 30*time.Second)

	// Client table
	clientCollectionQuery := fmt.Sprintf("CREATE TABLE IF NOT EXISTS %s.%s (id text, client_id text, kind text, name text, description text, client_secret text, allowed_scopes text, redirect_uris text, grant_types text, token_endpoint_auth_method text, is_active boolean, org_id text, created_at bigint, updated_at bigint, PRIMARY KEY (id))", KeySpace, schemas.Collections.Client)
	err = session.Query(clientCollectionQuery).Exec()
	if err != nil {
		return nil, err
	}
	clientClientIDIndex := fmt.Sprintf("CREATE INDEX IF NOT EXISTS authorizer_client_client_id ON %s.%s (client_id)", KeySpace, schemas.Collections.Client)
	err = session.Query(clientClientIDIndex).Exec()
	if err != nil {
		return nil, err
	}
	clientOrgIDIndex := fmt.Sprintf("CREATE INDEX IF NOT EXISTS authorizer_client_org_id ON %s.%s (org_id)", KeySpace, schemas.Collections.Client)
	err = session.Query(clientOrgIDIndex).Exec()
	if err != nil {
		return nil, err
	}
	// ScyllaDB builds secondary indexes asynchronously; wait for the client_id
	// index (used by GetClientByClientID and the boot-time reserved-client seed)
	// to become queryable.
	waitForCassandraSecondaryIndex(session, KeySpace, schemas.Collections.Client, "client_id", 30*time.Second)

	// TrustedIssuer table and indexes
	trustedIssuerCollectionQuery := fmt.Sprintf("CREATE TABLE IF NOT EXISTS %s.%s (id text, client_id text, name text, issuer_url text, key_source_type text, jwks_url text, expected_aud text, subject_claim text, allowed_subjects text, issuer_type text, auth_method text, is_active boolean, enable_token_review boolean, kubernetes_api_server_url text, spiffe_refresh_hint_seconds bigint, trusted_proxy_header text, trusted_proxy_cidrs text, kind text, org_id text, sso_client_id text, sso_client_secret_enc text, sso_scopes text, sso_redirect_uri text, saml_sso_url text, saml_idp_cert_pem text, saml_sp_entity_id text, saml_acs_url text, saml_attribute_mapping text, saml_allow_idp_initiated boolean, created_at bigint, updated_at bigint, PRIMARY KEY (id))", KeySpace, schemas.Collections.TrustedIssuer)
	err = session.Query(trustedIssuerCollectionQuery).Exec()
	if err != nil {
		return nil, err
	}
	// Add allowed_subjects column for keyspaces created before the client_assertion
	// subject-pin (§5.2 C1) landed. Tolerated if the column already exists.
	trustedIssuerAlterQuery := fmt.Sprintf(`ALTER TABLE %s.%s ADD (allowed_subjects text);`, KeySpace, schemas.Collections.TrustedIssuer)
	if err = session.Query(trustedIssuerAlterQuery).Exec(); err != nil {
		deps.Log.Debug().Err(err).Msg("Failed to alter trusted_issuers table as allowed_subjects column exists")
		// continue
	}
	// Add SSO broker columns for keyspaces created before the sso_oidc kind (§4.3)
	// landed. Tolerated if the columns already exist.
	trustedIssuerSSOAlterQuery := fmt.Sprintf(`ALTER TABLE %s.%s ADD (kind text, org_id text, sso_client_id text, sso_client_secret_enc text, sso_scopes text, sso_redirect_uri text);`, KeySpace, schemas.Collections.TrustedIssuer)
	if err = session.Query(trustedIssuerSSOAlterQuery).Exec(); err != nil {
		deps.Log.Debug().Err(err).Msg("Failed to alter trusted_issuers table as SSO broker columns exist")
		// continue
	}
	// Add SAML SP columns for keyspaces created before the sso_saml kind (§4.4)
	// landed. Tolerated if the columns already exist.
	trustedIssuerSAMLAlterQuery := fmt.Sprintf(`ALTER TABLE %s.%s ADD (saml_sso_url text, saml_idp_cert_pem text, saml_sp_entity_id text, saml_acs_url text, saml_attribute_mapping text, saml_allow_idp_initiated boolean);`, KeySpace, schemas.Collections.TrustedIssuer)
	if err = session.Query(trustedIssuerSAMLAlterQuery).Exec(); err != nil {
		deps.Log.Debug().Err(err).Msg("Failed to alter trusted_issuers table as SAML SP columns exist")
		// continue
	}
	trustedIssuerIssuerURLIndex := fmt.Sprintf("CREATE INDEX IF NOT EXISTS authorizer_trusted_issuer_issuer_url ON %s.%s (issuer_url)", KeySpace, schemas.Collections.TrustedIssuer)
	err = session.Query(trustedIssuerIssuerURLIndex).Exec()
	if err != nil {
		return nil, err
	}
	trustedIssuerClientIndex := fmt.Sprintf("CREATE INDEX IF NOT EXISTS authorizer_trusted_issuer_client_id ON %s.%s (client_id)", KeySpace, schemas.Collections.TrustedIssuer)
	err = session.Query(trustedIssuerClientIndex).Exec()
	if err != nil {
		return nil, err
	}
	// ScyllaDB builds secondary indexes asynchronously; wait for the issuer_url
	// index (hot path for client_assertion validation) to become queryable.
	waitForCassandraSecondaryIndex(session, KeySpace, schemas.Collections.TrustedIssuer, "issuer_url", 30*time.Second)

	// WebauthnCredential table and indexes
	webauthnCredentialCollectionQuery := fmt.Sprintf("CREATE TABLE IF NOT EXISTS %s.%s (id text, user_id text, credential_id text, public_key text, sign_count bigint, flags bigint, transports text, aaguid text, name text, created_at bigint, updated_at bigint, last_used_at bigint, PRIMARY KEY (id))", KeySpace, schemas.Collections.WebauthnCredential)
	err = session.Query(webauthnCredentialCollectionQuery).Exec()
	if err != nil {
		return nil, err
	}
	webauthnCredentialCredentialIDIndex := fmt.Sprintf("CREATE INDEX IF NOT EXISTS authorizer_webauthn_credential_credential_id ON %s.%s (credential_id)", KeySpace, schemas.Collections.WebauthnCredential)
	err = session.Query(webauthnCredentialCredentialIDIndex).Exec()
	if err != nil {
		return nil, err
	}
	webauthnCredentialUserIDIndex := fmt.Sprintf("CREATE INDEX IF NOT EXISTS authorizer_webauthn_credential_user_id ON %s.%s (user_id)", KeySpace, schemas.Collections.WebauthnCredential)
	err = session.Query(webauthnCredentialUserIDIndex).Exec()
	if err != nil {
		return nil, err
	}
	// ScyllaDB builds secondary indexes asynchronously; wait for the credential_id
	// index (hot path for usernameless login) to become queryable.
	waitForCassandraSecondaryIndex(session, KeySpace, schemas.Collections.WebauthnCredential, "credential_id", 30*time.Second)

	// Organization table and index
	organizationCollectionQuery := fmt.Sprintf("CREATE TABLE IF NOT EXISTS %s.%s (id text, name text, display_name text, enabled boolean, created_at bigint, updated_at bigint, PRIMARY KEY (id))", KeySpace, schemas.Collections.Organization)
	if err = session.Query(organizationCollectionQuery).Exec(); err != nil {
		return nil, err
	}
	organizationNameIndex := fmt.Sprintf("CREATE INDEX IF NOT EXISTS authorizer_organization_name ON %s.%s (name)", KeySpace, schemas.Collections.Organization)
	if err = session.Query(organizationNameIndex).Exec(); err != nil {
		return nil, err
	}

	// OrgMembership table and indexes
	orgMembershipCollectionQuery := fmt.Sprintf("CREATE TABLE IF NOT EXISTS %s.%s (id text, org_id text, user_id text, roles text, created_at bigint, updated_at bigint, PRIMARY KEY (id))", KeySpace, schemas.Collections.OrgMembership)
	if err = session.Query(orgMembershipCollectionQuery).Exec(); err != nil {
		return nil, err
	}
	orgMembershipOrgIndex := fmt.Sprintf("CREATE INDEX IF NOT EXISTS authorizer_org_membership_org_id ON %s.%s (org_id)", KeySpace, schemas.Collections.OrgMembership)
	if err = session.Query(orgMembershipOrgIndex).Exec(); err != nil {
		return nil, err
	}
	orgMembershipUserIndex := fmt.Sprintf("CREATE INDEX IF NOT EXISTS authorizer_org_membership_user_id ON %s.%s (user_id)", KeySpace, schemas.Collections.OrgMembership)
	if err = session.Query(orgMembershipUserIndex).Exec(); err != nil {
		return nil, err
	}

	// FederatedIdentity table and indexes
	federatedIdentityCollectionQuery := fmt.Sprintf("CREATE TABLE IF NOT EXISTS %s.%s (id text, org_id text, issuer text, subject text, user_id text, created_at bigint, updated_at bigint, PRIMARY KEY (id))", KeySpace, schemas.Collections.FederatedIdentity)
	if err = session.Query(federatedIdentityCollectionQuery).Exec(); err != nil {
		return nil, err
	}
	federatedIdentityOrgIndex := fmt.Sprintf("CREATE INDEX IF NOT EXISTS authorizer_federated_identity_org_id ON %s.%s (org_id)", KeySpace, schemas.Collections.FederatedIdentity)
	if err = session.Query(federatedIdentityOrgIndex).Exec(); err != nil {
		return nil, err
	}
	federatedIdentityUserIndex := fmt.Sprintf("CREATE INDEX IF NOT EXISTS authorizer_federated_identity_user_id ON %s.%s (user_id)", KeySpace, schemas.Collections.FederatedIdentity)
	if err = session.Query(federatedIdentityUserIndex).Exec(); err != nil {
		return nil, err
	}

	// ScyllaDB builds secondary indexes asynchronously; wait for the lookup
	// columns used by the uniqueness guard and membership listings.
	waitForCassandraSecondaryIndex(session, KeySpace, schemas.Collections.Organization, "name", 30*time.Second)
	waitForCassandraSecondaryIndex(session, KeySpace, schemas.Collections.OrgMembership, "org_id", 30*time.Second)
	waitForCassandraSecondaryIndex(session, KeySpace, schemas.Collections.OrgMembership, "user_id", 30*time.Second)
	waitForCassandraSecondaryIndex(session, KeySpace, schemas.Collections.FederatedIdentity, "org_id", 30*time.Second)
	waitForCassandraSecondaryIndex(session, KeySpace, schemas.Collections.FederatedIdentity, "user_id", 30*time.Second)

	// ScimEndpoint table and index
	scimEndpointCollectionQuery := fmt.Sprintf("CREATE TABLE IF NOT EXISTS %s.%s (id text, org_id text, token_hash text, enabled boolean, created_at bigint, updated_at bigint, PRIMARY KEY (id))", KeySpace, schemas.Collections.ScimEndpoint)
	if err = session.Query(scimEndpointCollectionQuery).Exec(); err != nil {
		return nil, err
	}
	scimEndpointOrgIndex := fmt.Sprintf("CREATE INDEX IF NOT EXISTS authorizer_scim_endpoint_org_id ON %s.%s (org_id)", KeySpace, schemas.Collections.ScimEndpoint)
	if err = session.Query(scimEndpointOrgIndex).Exec(); err != nil {
		return nil, err
	}
	// ScyllaDB builds secondary indexes asynchronously; wait for org_id (used by
	// GetScimEndpointByOrgID and the uniqueness guard) to become queryable.
	waitForCassandraSecondaryIndex(session, KeySpace, schemas.Collections.ScimEndpoint, "org_id", 30*time.Second)

	return &provider{
		config:       cfg,
		dependencies: deps,
		db:           session,
	}, err
}

// waitForCassandraSecondaryIndex polls a probe query that requires the given
// column's secondary index until it succeeds or the timeout is reached.
// ScyllaDB builds secondary indexes asynchronously; queries on indexed columns
// fail until the index is ready.
func waitForCassandraSecondaryIndex(session *cansandraDriver.Session, keyspace, table, column string, timeout time.Duration) {
	probe := fmt.Sprintf("SELECT id FROM %s.%s WHERE %s='' LIMIT 1 ALLOW FILTERING", keyspace, table, column)
	deadline := time.Now().Add(timeout)
	delay := 500 * time.Millisecond
	for {
		if err := session.Query(probe).Exec(); err == nil {
			return
		}
		if time.Now().After(deadline) {
			return
		}
		time.Sleep(delay)
		if delay < 3*time.Second {
			delay += 500 * time.Millisecond
		}
	}
}

// waitForCassandraIndexes polls a probe query that requires the actor_id secondary
// index until it succeeds or the timeout is reached. ScyllaDB builds secondary
// indexes asynchronously; queries on indexed columns fail until the index is ready.
func waitForCassandraIndexes(session *cansandraDriver.Session, keyspace, table string, timeout time.Duration) {
	probe := fmt.Sprintf("SELECT id FROM %s.%s WHERE actor_id='' LIMIT 1", keyspace, table)
	deadline := time.Now().Add(timeout)
	delay := 500 * time.Millisecond
	for {
		if err := session.Query(probe).Exec(); err == nil {
			return
		}
		if time.Now().After(deadline) {
			return
		}
		time.Sleep(delay)
		if delay < 3*time.Second {
			delay += 500 * time.Millisecond
		}
	}
}

// Close closes the Cassandra session.
func (p *provider) Close() error {
	if p == nil || p.db == nil {
		return nil
	}
	p.db.Close()
	return nil
}

// convertMapValues converts json.Number values in a map to native Go types
// (int64 or float64) so gocql can marshal them into CQL bigint/double columns.
func convertMapValues(m map[string]interface{}) {
	for key, value := range m {
		if num, ok := value.(json.Number); ok {
			if i, err := num.Int64(); err == nil {
				m[key] = i
			} else if f, err := num.Float64(); err == nil {
				m[key] = f
			}
		}
	}
}

// buildCQLColumnMap reflects over a schema struct and returns a map keyed by each
// field's `cql` tag name to its native Go value, ready for a gocql INSERT/UPDATE.
//
// It replaces the json.Marshal→decode→map construction on secret-bearing write
// paths (User, Client). encoding/json honors `json:"-"`, which is set on
// User.Password and Client.ClientSecret purely to keep those secrets out of
// API/GraphQL/log JSON. As a side effect the JSON-based builder silently dropped
// them from the CQL statement — password was never written at signup, and secret
// rotation silently no-op'd. The `cql` tag is never set to "-" for API-safety, so
// sourcing column names from it persists every field the table actually has.
//
// Value semantics mirror the json.Marshal→decode→convertMapValues path it replaces:
//   - nil pointer fields map to a nil value; callers skip nil on INSERT and emit
//     "col = null" on UPDATE, preserving null handling.
//   - non-nil pointers are dereferenced to their element value (native
//     int64/string/bool), so no json.Number coercion is needed.
//   - `omitempty` fields with a nil pointer or zero non-pointer value are omitted
//     entirely, matching json omitempty on User.Key / Client.Key (`_key`).
//   - `cql:"-"` and untagged fields are skipped.
func buildCQLColumnMap(v interface{}) map[string]interface{} {
	rv := reflect.Indirect(reflect.ValueOf(v))
	rt := rv.Type()
	out := make(map[string]interface{}, rt.NumField())
	for i := 0; i < rt.NumField(); i++ {
		f := rt.Field(i)
		parts := strings.Split(f.Tag.Get("cql"), ",")
		name := parts[0]
		if name == "" || name == "-" {
			continue
		}
		omitempty := false
		for _, opt := range parts[1:] {
			if opt == "omitempty" {
				omitempty = true
			}
		}
		fv := rv.Field(i)
		if fv.Kind() == reflect.Ptr {
			if fv.IsNil() {
				if omitempty {
					continue
				}
				out[name] = nil
				continue
			}
			out[name] = fv.Elem().Interface()
			continue
		}
		if omitempty && fv.IsZero() {
			continue
		}
		out[name] = fv.Interface()
	}
	return out
}
