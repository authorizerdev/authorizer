package storage

import (
	"context"
	"fmt"

	"github.com/rs/zerolog"

	"github.com/authorizerdev/authorizer/internal/config"
	"github.com/authorizerdev/authorizer/internal/constants"
	"github.com/authorizerdev/authorizer/internal/graph/model"
	"github.com/authorizerdev/authorizer/internal/storage/db/arangodb"
	"github.com/authorizerdev/authorizer/internal/storage/db/cassandradb"
	"github.com/authorizerdev/authorizer/internal/storage/db/couchbase"
	"github.com/authorizerdev/authorizer/internal/storage/db/dynamodb"
	"github.com/authorizerdev/authorizer/internal/storage/db/mongodb"
	"github.com/authorizerdev/authorizer/internal/storage/db/sql"
	"github.com/authorizerdev/authorizer/internal/storage/schemas"
)

// Dependencies carries shared resources for constructing a storage Provider.
type Dependencies struct {
	Log *zerolog.Logger
}

// Provider is the interface which defines the methods for the database provider.
//
// Delete methods are idempotent: deleting a non-existent id returns nil, not an
// error. Callers that rely on delete-confirms-existence must check existence
// separately first.
type Provider interface {
	// AddUser to save user information in database
	AddUser(ctx context.Context, user *schemas.User) (*schemas.User, error)
	// UpdateUser to update user information in database
	UpdateUser(ctx context.Context, user *schemas.User) (*schemas.User, error)
	// DeleteUser to delete user information from database
	DeleteUser(ctx context.Context, user *schemas.User) error
	// ListUsers to get list of users from database
	ListUsers(ctx context.Context, pagination *model.Pagination) ([]*schemas.User, *model.Pagination, error)
	// GetUserByEmail to get user information from database using email address
	GetUserByEmail(ctx context.Context, email string) (*schemas.User, error)
	// GetUserByPhoneNumber to get user information from database using phone number
	GetUserByPhoneNumber(ctx context.Context, phoneNumber string) (*schemas.User, error)
	// GetUserByID to get user information from database using user ID
	GetUserByID(ctx context.Context, id string) (*schemas.User, error)
	// GetUserByExternalID fetches an IdP-provisioned user by its org-namespaced
	// external id. The lookup key is composed as "<orgID>:<externalID>" so one
	// org's SCIM/SSO connection can never resolve another org's user by external
	// id (design §4.4 H6). Provisioning stores User.ExternalID in the same
	// namespaced form.
	GetUserByExternalID(ctx context.Context, orgID, externalID string) (*schemas.User, error)
	// UpdateUsers to update multiple users, identified by the ids slice.
	// If ids is nil / empty NO update is performed: global updates are disabled,
	// so implementations return an error (SQL: gorm.ErrMissingWhereClause) rather
	// than silently updating every user.
	UpdateUsers(ctx context.Context, data map[string]interface{}, ids []string) error

	// AddVerificationRequest to save verification request in database
	AddVerificationRequest(ctx context.Context, verificationRequest *schemas.VerificationRequest) (*schemas.VerificationRequest, error)
	// GetVerificationRequestByToken to get verification request from database using token
	GetVerificationRequestByToken(ctx context.Context, token string) (*schemas.VerificationRequest, error)
	// GetVerificationRequestByEmail to get verification request by email from database
	GetVerificationRequestByEmail(ctx context.Context, email string, identifier string) (*schemas.VerificationRequest, error)
	// ListVerificationRequests to get list of verification requests from database
	ListVerificationRequests(ctx context.Context, pagination *model.Pagination) ([]*schemas.VerificationRequest, *model.Pagination, error)
	// DeleteVerificationRequest to delete verification request from database
	DeleteVerificationRequest(ctx context.Context, verificationRequest *schemas.VerificationRequest) error

	// AddSession to save session information in database
	AddSession(ctx context.Context, session *schemas.Session) error
	// DeleteSession to delete session information from database
	DeleteSession(ctx context.Context, userId string) error

	// // AddEnv to save environment information in database
	// AddEnv(ctx context.Context, env *schemas.Env) (*schemas.Env, error)
	// // UpdateEnv to update environment information in database
	// UpdateEnv(ctx context.Context, env *schemas.Env) (*schemas.Env, error)
	// // GetEnv to get environment information from database
	// GetEnv(ctx context.Context) (*schemas.Env, error)

	// AddWebhook to add webhook
	AddWebhook(ctx context.Context, webhook *schemas.Webhook) (*schemas.Webhook, error)
	// UpdateWebhook to update webhook
	UpdateWebhook(ctx context.Context, webhook *schemas.Webhook) (*schemas.Webhook, error)
	// ListWebhook to list webhook
	ListWebhook(ctx context.Context, pagination *model.Pagination) ([]*schemas.Webhook, *model.Pagination, error)
	// GetWebhookByID to get webhook by id
	GetWebhookByID(ctx context.Context, webhookID string) (*schemas.Webhook, error)
	// GetWebhookByEventName to get webhook by event_name
	GetWebhookByEventName(ctx context.Context, eventName string) ([]*schemas.Webhook, error)
	// DeleteWebhook to delete webhook
	DeleteWebhook(ctx context.Context, webhook *schemas.Webhook) error

	// AddWebhookLog to add webhook log
	AddWebhookLog(ctx context.Context, webhookLog *schemas.WebhookLog) (*schemas.WebhookLog, error)
	// ListWebhookLogs to list webhook logs
	ListWebhookLogs(ctx context.Context, pagination *model.Pagination, webhookID string) ([]*schemas.WebhookLog, *model.Pagination, error)

	// AddEmailTemplate to add EmailTemplate
	AddEmailTemplate(ctx context.Context, emailTemplate *schemas.EmailTemplate) (*schemas.EmailTemplate, error)
	// UpdateEmailTemplate to update EmailTemplate
	UpdateEmailTemplate(ctx context.Context, emailTemplate *schemas.EmailTemplate) (*schemas.EmailTemplate, error)
	// ListEmailTemplate to list EmailTemplate
	ListEmailTemplate(ctx context.Context, pagination *model.Pagination) ([]*schemas.EmailTemplate, *model.Pagination, error)
	// GetEmailTemplateByID to get EmailTemplate by id
	GetEmailTemplateByID(ctx context.Context, emailTemplateID string) (*schemas.EmailTemplate, error)
	// GetEmailTemplateByEventName to get EmailTemplate by event_name
	GetEmailTemplateByEventName(ctx context.Context, eventName string) (*schemas.EmailTemplate, error)
	// DeleteEmailTemplate to delete EmailTemplate
	DeleteEmailTemplate(ctx context.Context, emailTemplate *schemas.EmailTemplate) error

	// UpsertOTP to add or update otp
	UpsertOTP(ctx context.Context, otp *schemas.OTP) (*schemas.OTP, error)
	// GetOTPByEmail to get otp for a given email address
	GetOTPByEmail(ctx context.Context, emailAddress string) (*schemas.OTP, error)
	// GetOTPByPhoneNumber to get otp for a given phone number
	GetOTPByPhoneNumber(ctx context.Context, phoneNumber string) (*schemas.OTP, error)
	// DeleteOTP to delete otp
	DeleteOTP(ctx context.Context, otp *schemas.OTP) error

	// AddAuthenticator adds a new authenticator document to the database.
	// If the authenticator doesn't have an ID, a new one is generated.
	// The created document is returned, or an error if the operation fails.
	AddAuthenticator(ctx context.Context, totp *schemas.Authenticator) (*schemas.Authenticator, error)
	// UpdateAuthenticator updates an existing authenticator document in the database.
	// The updated document is returned, or an error if the operation fails.
	UpdateAuthenticator(ctx context.Context, totp *schemas.Authenticator) (*schemas.Authenticator, error)
	// GetAuthenticatorDetailsByUserId retrieves details of an authenticator document based on user ID and authenticator type.
	// If found, the authenticator document is returned, or an error if not found or an error occurs during the retrieval.
	GetAuthenticatorDetailsByUserId(ctx context.Context, userId string, authenticatorType string) (*schemas.Authenticator, error)

	// Session Token methods (for database-backed memory store)
	// AddSessionToken adds a session token to the database
	AddSessionToken(ctx context.Context, token *schemas.SessionToken) error
	// GetSessionTokenByUserIDAndKey retrieves a session token by user ID and key
	GetSessionTokenByUserIDAndKey(ctx context.Context, userId, key string) (*schemas.SessionToken, error)
	// DeleteSessionToken deletes a session token by ID
	DeleteSessionToken(ctx context.Context, id string) error
	// DeleteSessionTokenByUserIDAndKey deletes a session token by user ID and key
	DeleteSessionTokenByUserIDAndKey(ctx context.Context, userId, key string) error
	// DeleteAllSessionTokensByUserID deletes all session tokens for a user ID
	DeleteAllSessionTokensByUserID(ctx context.Context, userId string) error
	// DeleteSessionTokensByNamespace deletes all session tokens for a namespace (e.g., "auth_provider")
	DeleteSessionTokensByNamespace(ctx context.Context, namespace string) error
	// CleanExpiredSessionTokens removes expired session tokens from the database
	CleanExpiredSessionTokens(ctx context.Context) error
	// GetAllSessionTokens retrieves all session tokens (for testing)
	GetAllSessionTokens(ctx context.Context) ([]*schemas.SessionToken, error)

	// MFA Session methods (for database-backed memory store)
	// AddMFASession adds an MFA session to the database
	AddMFASession(ctx context.Context, session *schemas.MFASession) error
	// GetMFASessionByUserIDAndKey retrieves an MFA session by user ID and key
	GetMFASessionByUserIDAndKey(ctx context.Context, userId, key string) (*schemas.MFASession, error)
	// DeleteMFASession deletes an MFA session by ID
	DeleteMFASession(ctx context.Context, id string) error
	// DeleteMFASessionByUserIDAndKey deletes an MFA session by user ID and key
	DeleteMFASessionByUserIDAndKey(ctx context.Context, userId, key string) error
	// GetAllMFASessionsByUserID retrieves all MFA sessions for a user ID
	GetAllMFASessionsByUserID(ctx context.Context, userId string) ([]*schemas.MFASession, error)
	// CleanExpiredMFASessions removes expired MFA sessions from the database
	CleanExpiredMFASessions(ctx context.Context) error
	// GetAllMFASessions retrieves all MFA sessions (for testing)
	GetAllMFASessions(ctx context.Context) ([]*schemas.MFASession, error)

	// OAuth State methods (for database-backed memory store)
	// AddOAuthState adds an OAuth state to the database
	AddOAuthState(ctx context.Context, state *schemas.OAuthState) error
	// GetOAuthStateByKey retrieves an OAuth state by key
	GetOAuthStateByKey(ctx context.Context, key string) (*schemas.OAuthState, error)
	// DeleteOAuthStateByKey deletes an OAuth state by key
	DeleteOAuthStateByKey(ctx context.Context, key string) error
	// GetAllOAuthStates retrieves all OAuth states (for testing)
	GetAllOAuthStates(ctx context.Context) ([]*schemas.OAuthState, error)

	// Audit Log methods
	// AddAuditLog adds an audit log entry
	AddAuditLog(ctx context.Context, log *schemas.AuditLog) error
	// ListAuditLogs queries audit logs with filters and pagination
	ListAuditLogs(ctx context.Context, pagination *model.Pagination, filter map[string]interface{}) ([]*schemas.AuditLog, *model.Pagination, error)
	// DeleteAuditLogsBefore removes logs older than a timestamp (retention)
	DeleteAuditLogsBefore(ctx context.Context, before int64) error

	// HealthCheck verifies that the storage backend is reachable and responsive.
	HealthCheck(ctx context.Context) error

	// Close releases resources held by the provider (e.g. database connection pools).
	Close() error

	// Client methods.

	// AddClient creates a new service account record.
	AddClient(ctx context.Context, sa *schemas.Client) (*schemas.Client, error)
	// UpdateClient updates name, description, allowed_scopes, or is_active.
	UpdateClient(ctx context.Context, sa *schemas.Client) (*schemas.Client, error)
	// DeleteClient removes a service account. Callers must delete
	// associated TrustedIssuers before or within the same logical operation.
	DeleteClient(ctx context.Context, sa *schemas.Client) error
	// GetClientByID fetches a client by its surrogate primary key.
	GetClientByID(ctx context.Context, id string) (*schemas.Client, error)
	// GetClientByClientID fetches a client by its public, unique client_id
	// (distinct from the surrogate ID). This is the lookup the token/authorize
	// endpoints and the boot-time reserved-client seed use.
	GetClientByClientID(ctx context.Context, clientID string) (*schemas.Client, error)
	// ListClients returns a paginated list of all service accounts.
	ListClients(ctx context.Context, pagination *model.Pagination) ([]*schemas.Client, *model.Pagination, error)

	// TrustedIssuer methods.

	// AddTrustedIssuer creates a new trusted issuer record.
	AddTrustedIssuer(ctx context.Context, issuer *schemas.TrustedIssuer) (*schemas.TrustedIssuer, error)
	// UpdateTrustedIssuer updates mutable fields: jwks_url, expected_aud,
	// is_active, spiffe_refresh_hint_seconds, enable_token_review,
	// kubernetes_api_server_url, trusted_proxy_header, trusted_proxy_cidrs.
	UpdateTrustedIssuer(ctx context.Context, issuer *schemas.TrustedIssuer) (*schemas.TrustedIssuer, error)
	// DeleteTrustedIssuer removes a trusted issuer.
	DeleteTrustedIssuer(ctx context.Context, issuer *schemas.TrustedIssuer) error
	// GetTrustedIssuerByID fetches a trusted issuer by primary key.
	GetTrustedIssuerByID(ctx context.Context, id string) (*schemas.TrustedIssuer, error)
	// GetTrustedIssuerByIssuerURL fetches by issuer URL (unique index).
	// This is called on every client_assertion validation — keep it fast.
	//
	// SECURITY (CR1): issuer_url is globally unique, so at most one row exists per
	// URL. The client_assertion resolver additionally rejects any row whose
	// EffectiveKind is not client_assertion_trust (or whose OrgID is non-empty), so
	// an sso_oidc row registered at the same URL can never authenticate a client.
	GetTrustedIssuerByIssuerURL(ctx context.Context, issuerURL string) (*schemas.TrustedIssuer, error)
	// GetTrustedIssuerByOrgIDAndKind fetches the single trusted issuer for an
	// organization of a given kind — used to resolve an org's sso_oidc connection.
	// Returns an error when no matching row exists.
	GetTrustedIssuerByOrgIDAndKind(ctx context.Context, orgID, kind string) (*schemas.TrustedIssuer, error)
	// ListTrustedIssuers returns trusted issuers filtered by serviceAccountID.
	// Pass an empty serviceAccountID to list all issuers.
	ListTrustedIssuers(ctx context.Context, serviceAccountID string, pagination *model.Pagination) ([]*schemas.TrustedIssuer, *model.Pagination, error)

	// Organization methods.

	// AddOrganization creates a new organization record.
	AddOrganization(ctx context.Context, org *schemas.Organization) (*schemas.Organization, error)
	// GetOrganizationByID fetches an organization by its primary key.
	GetOrganizationByID(ctx context.Context, id string) (*schemas.Organization, error)
	// GetOrganizationByName fetches an organization by its unique name slug.
	GetOrganizationByName(ctx context.Context, name string) (*schemas.Organization, error)
	// UpdateOrganization updates name, display_name, or enabled.
	UpdateOrganization(ctx context.Context, org *schemas.Organization) (*schemas.Organization, error)
	// DeleteOrganization removes an organization and cascade-deletes its
	// memberships. Mirrors the DeleteClient cascade pattern.
	DeleteOrganization(ctx context.Context, org *schemas.Organization) error
	// ListOrganizations returns a paginated list of all organizations.
	ListOrganizations(ctx context.Context, pagination *model.Pagination) ([]*schemas.Organization, *model.Pagination, error)

	// OrgMembership methods.

	// AddOrgMembership creates a new membership. The (org_id, user_id) pair is
	// unique — adding a duplicate returns an error.
	AddOrgMembership(ctx context.Context, membership *schemas.OrgMembership) (*schemas.OrgMembership, error)
	// GetOrgMembership fetches the membership for a (orgID, userID) pair.
	GetOrgMembership(ctx context.Context, orgID, userID string) (*schemas.OrgMembership, error)
	// UpdateOrgMembership updates the roles of an existing membership.
	UpdateOrgMembership(ctx context.Context, membership *schemas.OrgMembership) (*schemas.OrgMembership, error)
	// DeleteOrgMembership removes a membership.
	DeleteOrgMembership(ctx context.Context, membership *schemas.OrgMembership) error
	// ListOrgMembershipsByOrg returns paginated memberships of an organization.
	ListOrgMembershipsByOrg(ctx context.Context, orgID string, pagination *model.Pagination) ([]*schemas.OrgMembership, *model.Pagination, error)
	// ListOrgMembershipsByUser returns paginated memberships held by a user.
	ListOrgMembershipsByUser(ctx context.Context, userID string, pagination *model.Pagination) ([]*schemas.OrgMembership, *model.Pagination, error)

	// FederatedIdentity methods (SSO JIT provisioning, account-takeover defense).

	// AddFederatedIdentity records a JIT-provisioned upstream identity. The
	// (org_id, issuer, subject) triple is unique — adding a duplicate returns an
	// error.
	AddFederatedIdentity(ctx context.Context, identity *schemas.FederatedIdentity) (*schemas.FederatedIdentity, error)
	// GetFederatedIdentity fetches the identity for a (orgID, issuer, subject)
	// triple. Returns an error when no matching row exists.
	GetFederatedIdentity(ctx context.Context, orgID, issuer, subject string) (*schemas.FederatedIdentity, error)

	// ScimEndpoint methods (per-org inbound SCIM 2.0 connection credential).

	// AddScimEndpoint creates a new SCIM endpoint. OrgID is unique — one
	// endpoint per org.
	AddScimEndpoint(ctx context.Context, endpoint *schemas.ScimEndpoint) (*schemas.ScimEndpoint, error)
	// GetScimEndpointByID fetches an endpoint by primary key (the id embedded in
	// the presented bearer token).
	GetScimEndpointByID(ctx context.Context, id string) (*schemas.ScimEndpoint, error)
	// GetScimEndpointByOrgID fetches an org's endpoint (admin surface, uniqueness
	// pre-check).
	GetScimEndpointByOrgID(ctx context.Context, orgID string) (*schemas.ScimEndpoint, error)
	// UpdateScimEndpoint updates an existing endpoint (token rotation, enable).
	// Callers MUST load-then-mutate — Save writes every column.
	UpdateScimEndpoint(ctx context.Context, endpoint *schemas.ScimEndpoint) (*schemas.ScimEndpoint, error)
	// DeleteScimEndpoint removes an endpoint.
	DeleteScimEndpoint(ctx context.Context, endpoint *schemas.ScimEndpoint) error

	// OrgDomain methods (verified domain → org mapping for home-realm discovery).

	// AddOrgDomain atomically inserts a verified domain row, keyed by the
	// normalized domain (the primary/partition key). First-writer-wins:
	//   - domain unclaimed → inserts and returns the new row.
	//   - domain already held by the SAME org → returns the existing row (idempotent).
	//   - domain already held by a DIFFERENT org → returns schemas.ErrOrgDomainConflict.
	// ID and Domain MUST both be set to the normalized domain by the caller.
	AddOrgDomain(ctx context.Context, domain *schemas.OrgDomain) (*schemas.OrgDomain, error)
	// GetOrgDomainByDomain fetches the verified row for a normalized domain
	// (the home-realm-discovery reverse lookup — a primary-key GET).
	GetOrgDomainByDomain(ctx context.Context, domain string) (*schemas.OrgDomain, error)
	// ListOrgDomainsByOrg returns an org's verified domains, paginated.
	ListOrgDomainsByOrg(ctx context.Context, orgID string, pagination *model.Pagination) ([]*schemas.OrgDomain, *model.Pagination, error)
	// DeleteOrgDomain removes a verified domain mapping by normalized domain.
	DeleteOrgDomain(ctx context.Context, domain string) error
	// DeleteOrgDomainsByOrg removes all of an org's verified domains (cascade on
	// org delete — otherwise the domain becomes permanently unclaimable).
	DeleteOrgDomainsByOrg(ctx context.Context, orgID string) error
}

// New creates a new database provider based on the configuration
func New(config *config.Config, deps *Dependencies) (Provider, error) {
	var provider Provider
	var err error
	if config.DatabaseType == "" {
		return nil, fmt.Errorf("database type is required")
	}

	switch config.DatabaseType {
	case constants.DbTypePostgres,
		constants.DbTypeSqlite,
		constants.DbTypeLibSQL,
		constants.DbTypeMysql,
		constants.DbTypeSqlserver,
		constants.DbTypeYugabyte,
		constants.DbTypeMariaDB,
		constants.DbTypeCockroachDB,
		constants.DbTypePlanetScaleDB:
		provider, err = sql.NewProvider(config, &sql.Dependencies{
			Log: deps.Log,
		})
	case constants.DbTypeMongoDB:
		provider, err = mongodb.NewProvider(config, &mongodb.Dependencies{
			Log: deps.Log,
		})
	case constants.DbTypeArangoDB:
		provider, err = arangodb.NewProvider(config, &arangodb.Dependencies{
			Log: deps.Log,
		})
	case constants.DbTypeCassandraDB,
		constants.DbTypeScyllaDB:
		provider, err = cassandradb.NewProvider(config, &cassandradb.Dependencies{
			Log: deps.Log,
		})
	case constants.DbTypeCouchbaseDB:
		provider, err = couchbase.NewProvider(config, &couchbase.Dependencies{
			Log: deps.Log,
		})
	case constants.DbTypeDynamoDB:
		provider, err = dynamodb.NewProvider(config, &dynamodb.Dependencies{
			Log: deps.Log,
		})
	default:
		err = fmt.Errorf("unsupported database type: %s", config.DatabaseType)

	}
	if err != nil {
		return nil, err
	}
	return provider, nil
}
