package storage

import (
	"context"
	"errors"

	"github.com/authorizerdev/authorizer/internal/graph/model"
	"github.com/authorizerdev/authorizer/internal/storage/schemas"
)

// Error marks an error as having come out of the storage layer. Its text is
// whatever the backend produced (driver messages, SQL fragments, table and
// constraint names), which is useful in server logs and must never be shown
// to API clients verbatim. Transports use IsStorageError to replace such
// errors with a generic "internal server error" (see
// service.MaskInternalError).
//
// Error() and Unwrap() are transparent: the message is unchanged and
// errors.Is / errors.As / IsNotFound see straight through the wrapper, so
// callers that branch on driver sentinels keep working.
type Error struct {
	// Err is the backend's original error.
	Err error
}

// Error returns the underlying storage error's text unchanged.
func (e *Error) Error() string { return e.Err.Error() }

// Unwrap exposes the underlying storage error.
func (e *Error) Unwrap() error { return e.Err }

// IsStorageError reports whether err (or anything it wraps) originated from
// the storage layer.
func IsStorageError(err error) bool {
	var se *Error
	return errors.As(err, &se)
}

func wrapErr(err error) error {
	if err == nil {
		return nil
	}
	return &Error{Err: err}
}

// errorBoundary decorates a backend Provider so every error it returns is
// marked as a storage error. It is applied once, in New, so no backend has to
// remember to do it. The compile-time assertion below forces every new
// Provider method to be added here as well.
type errorBoundary struct {
	next Provider
}

var _ Provider = (*errorBoundary)(nil)

func (p *errorBoundary) AddUser(ctx context.Context, user *schemas.User) (*schemas.User, error) {
	r0, err := p.next.AddUser(ctx, user)
	return r0, wrapErr(err)
}

func (p *errorBoundary) UpdateUser(ctx context.Context, user *schemas.User) (*schemas.User, error) {
	r0, err := p.next.UpdateUser(ctx, user)
	return r0, wrapErr(err)
}

func (p *errorBoundary) DeleteUser(ctx context.Context, user *schemas.User) error {
	return wrapErr(p.next.DeleteUser(ctx, user))
}

func (p *errorBoundary) ListUsers(ctx context.Context, pagination *model.Pagination, opts schemas.ListUsersOptions) ([]*schemas.User, *model.Pagination, error) {
	r0, r1, err := p.next.ListUsers(ctx, pagination, opts)
	return r0, r1, wrapErr(err)
}

func (p *errorBoundary) GetUserByEmail(ctx context.Context, email string) (*schemas.User, error) {
	r0, err := p.next.GetUserByEmail(ctx, email)
	return r0, wrapErr(err)
}

func (p *errorBoundary) GetUserByPhoneNumber(ctx context.Context, phoneNumber string) (*schemas.User, error) {
	r0, err := p.next.GetUserByPhoneNumber(ctx, phoneNumber)
	return r0, wrapErr(err)
}

func (p *errorBoundary) GetUserByID(ctx context.Context, id string) (*schemas.User, error) {
	r0, err := p.next.GetUserByID(ctx, id)
	return r0, wrapErr(err)
}

func (p *errorBoundary) GetUserByExternalID(ctx context.Context, orgID string, externalID string) (*schemas.User, error) {
	r0, err := p.next.GetUserByExternalID(ctx, orgID, externalID)
	return r0, wrapErr(err)
}

func (p *errorBoundary) UpdateUsers(ctx context.Context, data map[string]interface{}, ids []string) error {
	return wrapErr(p.next.UpdateUsers(ctx, data, ids))
}

func (p *errorBoundary) AddVerificationRequest(ctx context.Context, verificationRequest *schemas.VerificationRequest) (*schemas.VerificationRequest, error) {
	r0, err := p.next.AddVerificationRequest(ctx, verificationRequest)
	return r0, wrapErr(err)
}

func (p *errorBoundary) GetVerificationRequestByToken(ctx context.Context, token string) (*schemas.VerificationRequest, error) {
	r0, err := p.next.GetVerificationRequestByToken(ctx, token)
	return r0, wrapErr(err)
}

func (p *errorBoundary) GetVerificationRequestByEmail(ctx context.Context, email string, identifier string) (*schemas.VerificationRequest, error) {
	r0, err := p.next.GetVerificationRequestByEmail(ctx, email, identifier)
	return r0, wrapErr(err)
}

func (p *errorBoundary) ListVerificationRequests(ctx context.Context, pagination *model.Pagination) ([]*schemas.VerificationRequest, *model.Pagination, error) {
	r0, r1, err := p.next.ListVerificationRequests(ctx, pagination)
	return r0, r1, wrapErr(err)
}

func (p *errorBoundary) DeleteVerificationRequest(ctx context.Context, verificationRequest *schemas.VerificationRequest) error {
	return wrapErr(p.next.DeleteVerificationRequest(ctx, verificationRequest))
}

func (p *errorBoundary) AddSession(ctx context.Context, session *schemas.Session) error {
	return wrapErr(p.next.AddSession(ctx, session))
}

func (p *errorBoundary) DeleteSession(ctx context.Context, userId string) error {
	return wrapErr(p.next.DeleteSession(ctx, userId))
}

func (p *errorBoundary) AddWebhook(ctx context.Context, webhook *schemas.Webhook) (*schemas.Webhook, error) {
	r0, err := p.next.AddWebhook(ctx, webhook)
	return r0, wrapErr(err)
}

func (p *errorBoundary) UpdateWebhook(ctx context.Context, webhook *schemas.Webhook) (*schemas.Webhook, error) {
	r0, err := p.next.UpdateWebhook(ctx, webhook)
	return r0, wrapErr(err)
}

func (p *errorBoundary) ListWebhook(ctx context.Context, pagination *model.Pagination) ([]*schemas.Webhook, *model.Pagination, error) {
	r0, r1, err := p.next.ListWebhook(ctx, pagination)
	return r0, r1, wrapErr(err)
}

func (p *errorBoundary) GetWebhookByID(ctx context.Context, webhookID string) (*schemas.Webhook, error) {
	r0, err := p.next.GetWebhookByID(ctx, webhookID)
	return r0, wrapErr(err)
}

func (p *errorBoundary) GetWebhookByEventName(ctx context.Context, eventName string) ([]*schemas.Webhook, error) {
	r0, err := p.next.GetWebhookByEventName(ctx, eventName)
	return r0, wrapErr(err)
}

func (p *errorBoundary) DeleteWebhook(ctx context.Context, webhook *schemas.Webhook) error {
	return wrapErr(p.next.DeleteWebhook(ctx, webhook))
}

func (p *errorBoundary) AddWebhookLog(ctx context.Context, webhookLog *schemas.WebhookLog) (*schemas.WebhookLog, error) {
	r0, err := p.next.AddWebhookLog(ctx, webhookLog)
	return r0, wrapErr(err)
}

func (p *errorBoundary) ListWebhookLogs(ctx context.Context, pagination *model.Pagination, webhookID string) ([]*schemas.WebhookLog, *model.Pagination, error) {
	r0, r1, err := p.next.ListWebhookLogs(ctx, pagination, webhookID)
	return r0, r1, wrapErr(err)
}

func (p *errorBoundary) AddEmailTemplate(ctx context.Context, emailTemplate *schemas.EmailTemplate) (*schemas.EmailTemplate, error) {
	r0, err := p.next.AddEmailTemplate(ctx, emailTemplate)
	return r0, wrapErr(err)
}

func (p *errorBoundary) UpdateEmailTemplate(ctx context.Context, emailTemplate *schemas.EmailTemplate) (*schemas.EmailTemplate, error) {
	r0, err := p.next.UpdateEmailTemplate(ctx, emailTemplate)
	return r0, wrapErr(err)
}

func (p *errorBoundary) ListEmailTemplate(ctx context.Context, pagination *model.Pagination) ([]*schemas.EmailTemplate, *model.Pagination, error) {
	r0, r1, err := p.next.ListEmailTemplate(ctx, pagination)
	return r0, r1, wrapErr(err)
}

func (p *errorBoundary) GetEmailTemplateByID(ctx context.Context, emailTemplateID string) (*schemas.EmailTemplate, error) {
	r0, err := p.next.GetEmailTemplateByID(ctx, emailTemplateID)
	return r0, wrapErr(err)
}

func (p *errorBoundary) GetEmailTemplateByEventName(ctx context.Context, eventName string) (*schemas.EmailTemplate, error) {
	r0, err := p.next.GetEmailTemplateByEventName(ctx, eventName)
	return r0, wrapErr(err)
}

func (p *errorBoundary) DeleteEmailTemplate(ctx context.Context, emailTemplate *schemas.EmailTemplate) error {
	return wrapErr(p.next.DeleteEmailTemplate(ctx, emailTemplate))
}

func (p *errorBoundary) UpsertOTP(ctx context.Context, otp *schemas.OTP) (*schemas.OTP, error) {
	r0, err := p.next.UpsertOTP(ctx, otp)
	return r0, wrapErr(err)
}

func (p *errorBoundary) GetOTPByEmail(ctx context.Context, emailAddress string) (*schemas.OTP, error) {
	r0, err := p.next.GetOTPByEmail(ctx, emailAddress)
	return r0, wrapErr(err)
}

func (p *errorBoundary) GetOTPByPhoneNumber(ctx context.Context, phoneNumber string) (*schemas.OTP, error) {
	r0, err := p.next.GetOTPByPhoneNumber(ctx, phoneNumber)
	return r0, wrapErr(err)
}

func (p *errorBoundary) DeleteOTP(ctx context.Context, otp *schemas.OTP) error {
	return wrapErr(p.next.DeleteOTP(ctx, otp))
}

func (p *errorBoundary) AddAuthenticator(ctx context.Context, totp *schemas.Authenticator) (*schemas.Authenticator, error) {
	r0, err := p.next.AddAuthenticator(ctx, totp)
	return r0, wrapErr(err)
}

func (p *errorBoundary) UpdateAuthenticator(ctx context.Context, totp *schemas.Authenticator) (*schemas.Authenticator, error) {
	r0, err := p.next.UpdateAuthenticator(ctx, totp)
	return r0, wrapErr(err)
}

func (p *errorBoundary) UpdateAuthenticatorSecretAndVerifiedAt(ctx context.Context, id string, secret string, verifiedAt int64) error {
	return wrapErr(p.next.UpdateAuthenticatorSecretAndVerifiedAt(ctx, id, secret, verifiedAt))
}

func (p *errorBoundary) ConsumeAuthenticatorRecoveryCode(ctx context.Context, id string, oldCodes string, newCodes string) (bool, error) {
	r0, err := p.next.ConsumeAuthenticatorRecoveryCode(ctx, id, oldCodes, newCodes)
	return r0, wrapErr(err)
}

func (p *errorBoundary) GetAuthenticatorDetailsByUserId(ctx context.Context, userId string, authenticatorType string) (*schemas.Authenticator, error) {
	r0, err := p.next.GetAuthenticatorDetailsByUserId(ctx, userId, authenticatorType)
	return r0, wrapErr(err)
}

func (p *errorBoundary) DeleteAuthenticatorsByUserID(ctx context.Context, userID string) error {
	return wrapErr(p.next.DeleteAuthenticatorsByUserID(ctx, userID))
}

func (p *errorBoundary) AddSessionToken(ctx context.Context, token *schemas.SessionToken) error {
	return wrapErr(p.next.AddSessionToken(ctx, token))
}

func (p *errorBoundary) GetSessionTokenByUserIDAndKey(ctx context.Context, userId string, key string) (*schemas.SessionToken, error) {
	r0, err := p.next.GetSessionTokenByUserIDAndKey(ctx, userId, key)
	return r0, wrapErr(err)
}

func (p *errorBoundary) DeleteSessionToken(ctx context.Context, id string) error {
	return wrapErr(p.next.DeleteSessionToken(ctx, id))
}

func (p *errorBoundary) DeleteSessionTokenByUserIDAndKey(ctx context.Context, userId string, key string) (bool, error) {
	r0, err := p.next.DeleteSessionTokenByUserIDAndKey(ctx, userId, key)
	return r0, wrapErr(err)
}

func (p *errorBoundary) DeleteAllSessionTokensByUserID(ctx context.Context, userId string) error {
	return wrapErr(p.next.DeleteAllSessionTokensByUserID(ctx, userId))
}

func (p *errorBoundary) DeleteSessionTokensByNamespace(ctx context.Context, namespace string) error {
	return wrapErr(p.next.DeleteSessionTokensByNamespace(ctx, namespace))
}

func (p *errorBoundary) CleanExpiredSessionTokens(ctx context.Context) error {
	return wrapErr(p.next.CleanExpiredSessionTokens(ctx))
}

func (p *errorBoundary) GetAllSessionTokens(ctx context.Context) ([]*schemas.SessionToken, error) {
	r0, err := p.next.GetAllSessionTokens(ctx)
	return r0, wrapErr(err)
}

func (p *errorBoundary) AddMFASession(ctx context.Context, session *schemas.MFASession) error {
	return wrapErr(p.next.AddMFASession(ctx, session))
}

func (p *errorBoundary) GetMFASessionByUserIDAndKey(ctx context.Context, userId string, key string) (*schemas.MFASession, error) {
	r0, err := p.next.GetMFASessionByUserIDAndKey(ctx, userId, key)
	return r0, wrapErr(err)
}

func (p *errorBoundary) DeleteMFASession(ctx context.Context, id string) error {
	return wrapErr(p.next.DeleteMFASession(ctx, id))
}

func (p *errorBoundary) DeleteMFASessionByUserIDAndKey(ctx context.Context, userId string, key string) error {
	return wrapErr(p.next.DeleteMFASessionByUserIDAndKey(ctx, userId, key))
}

func (p *errorBoundary) GetAllMFASessionsByUserID(ctx context.Context, userId string) ([]*schemas.MFASession, error) {
	r0, err := p.next.GetAllMFASessionsByUserID(ctx, userId)
	return r0, wrapErr(err)
}

func (p *errorBoundary) CleanExpiredMFASessions(ctx context.Context) error {
	return wrapErr(p.next.CleanExpiredMFASessions(ctx))
}

func (p *errorBoundary) GetAllMFASessions(ctx context.Context) ([]*schemas.MFASession, error) {
	r0, err := p.next.GetAllMFASessions(ctx)
	return r0, wrapErr(err)
}

func (p *errorBoundary) AddOAuthState(ctx context.Context, state *schemas.OAuthState) error {
	return wrapErr(p.next.AddOAuthState(ctx, state))
}

func (p *errorBoundary) GetOAuthStateByKey(ctx context.Context, key string) (*schemas.OAuthState, error) {
	r0, err := p.next.GetOAuthStateByKey(ctx, key)
	return r0, wrapErr(err)
}

func (p *errorBoundary) DeleteOAuthStateByKey(ctx context.Context, key string) (bool, error) {
	r0, err := p.next.DeleteOAuthStateByKey(ctx, key)
	return r0, wrapErr(err)
}

func (p *errorBoundary) GetAllOAuthStates(ctx context.Context) ([]*schemas.OAuthState, error) {
	r0, err := p.next.GetAllOAuthStates(ctx)
	return r0, wrapErr(err)
}

func (p *errorBoundary) AddAuditLog(ctx context.Context, log *schemas.AuditLog) error {
	return wrapErr(p.next.AddAuditLog(ctx, log))
}

func (p *errorBoundary) ListAuditLogs(ctx context.Context, pagination *model.Pagination, filter map[string]interface{}) ([]*schemas.AuditLog, *model.Pagination, error) {
	r0, r1, err := p.next.ListAuditLogs(ctx, pagination, filter)
	return r0, r1, wrapErr(err)
}

func (p *errorBoundary) DeleteAuditLogsBefore(ctx context.Context, before int64) error {
	return wrapErr(p.next.DeleteAuditLogsBefore(ctx, before))
}

func (p *errorBoundary) HealthCheck(ctx context.Context) error {
	return wrapErr(p.next.HealthCheck(ctx))
}

func (p *errorBoundary) Close() error {
	return wrapErr(p.next.Close())
}

func (p *errorBoundary) AddClient(ctx context.Context, sa *schemas.Client) (*schemas.Client, error) {
	r0, err := p.next.AddClient(ctx, sa)
	return r0, wrapErr(err)
}

func (p *errorBoundary) UpdateClient(ctx context.Context, sa *schemas.Client) (*schemas.Client, error) {
	r0, err := p.next.UpdateClient(ctx, sa)
	return r0, wrapErr(err)
}

func (p *errorBoundary) DeleteClient(ctx context.Context, sa *schemas.Client) error {
	return wrapErr(p.next.DeleteClient(ctx, sa))
}

func (p *errorBoundary) GetClientByID(ctx context.Context, id string) (*schemas.Client, error) {
	r0, err := p.next.GetClientByID(ctx, id)
	return r0, wrapErr(err)
}

func (p *errorBoundary) GetClientByClientID(ctx context.Context, clientID string) (*schemas.Client, error) {
	r0, err := p.next.GetClientByClientID(ctx, clientID)
	return r0, wrapErr(err)
}

func (p *errorBoundary) ListClients(ctx context.Context, pagination *model.Pagination) ([]*schemas.Client, *model.Pagination, error) {
	r0, r1, err := p.next.ListClients(ctx, pagination)
	return r0, r1, wrapErr(err)
}

func (p *errorBoundary) AddTrustedIssuer(ctx context.Context, issuer *schemas.TrustedIssuer) (*schemas.TrustedIssuer, error) {
	r0, err := p.next.AddTrustedIssuer(ctx, issuer)
	return r0, wrapErr(err)
}

func (p *errorBoundary) UpdateTrustedIssuer(ctx context.Context, issuer *schemas.TrustedIssuer) (*schemas.TrustedIssuer, error) {
	r0, err := p.next.UpdateTrustedIssuer(ctx, issuer)
	return r0, wrapErr(err)
}

func (p *errorBoundary) DeleteTrustedIssuer(ctx context.Context, issuer *schemas.TrustedIssuer) error {
	return wrapErr(p.next.DeleteTrustedIssuer(ctx, issuer))
}

func (p *errorBoundary) GetTrustedIssuerByID(ctx context.Context, id string) (*schemas.TrustedIssuer, error) {
	r0, err := p.next.GetTrustedIssuerByID(ctx, id)
	return r0, wrapErr(err)
}

func (p *errorBoundary) GetTrustedIssuerByIssuerURL(ctx context.Context, issuerURL string) (*schemas.TrustedIssuer, error) {
	r0, err := p.next.GetTrustedIssuerByIssuerURL(ctx, issuerURL)
	return r0, wrapErr(err)
}

func (p *errorBoundary) GetTrustedIssuerByOrgIDAndKind(ctx context.Context, orgID string, kind string) (*schemas.TrustedIssuer, error) {
	r0, err := p.next.GetTrustedIssuerByOrgIDAndKind(ctx, orgID, kind)
	return r0, wrapErr(err)
}

func (p *errorBoundary) ListTrustedIssuers(ctx context.Context, serviceAccountID string, pagination *model.Pagination) ([]*schemas.TrustedIssuer, *model.Pagination, error) {
	r0, r1, err := p.next.ListTrustedIssuers(ctx, serviceAccountID, pagination)
	return r0, r1, wrapErr(err)
}

func (p *errorBoundary) AddSAMLServiceProvider(ctx context.Context, sp *schemas.SAMLServiceProvider) (*schemas.SAMLServiceProvider, error) {
	r0, err := p.next.AddSAMLServiceProvider(ctx, sp)
	return r0, wrapErr(err)
}

func (p *errorBoundary) UpdateSAMLServiceProvider(ctx context.Context, sp *schemas.SAMLServiceProvider) (*schemas.SAMLServiceProvider, error) {
	r0, err := p.next.UpdateSAMLServiceProvider(ctx, sp)
	return r0, wrapErr(err)
}

func (p *errorBoundary) DeleteSAMLServiceProvider(ctx context.Context, sp *schemas.SAMLServiceProvider) error {
	return wrapErr(p.next.DeleteSAMLServiceProvider(ctx, sp))
}

func (p *errorBoundary) GetSAMLServiceProviderByID(ctx context.Context, id string) (*schemas.SAMLServiceProvider, error) {
	r0, err := p.next.GetSAMLServiceProviderByID(ctx, id)
	return r0, wrapErr(err)
}

func (p *errorBoundary) GetSAMLServiceProviderByOrgAndEntityID(ctx context.Context, orgID string, entityID string) (*schemas.SAMLServiceProvider, error) {
	r0, err := p.next.GetSAMLServiceProviderByOrgAndEntityID(ctx, orgID, entityID)
	return r0, wrapErr(err)
}

func (p *errorBoundary) ListSAMLServiceProviders(ctx context.Context, orgID string, pagination *model.Pagination) ([]*schemas.SAMLServiceProvider, *model.Pagination, error) {
	r0, r1, err := p.next.ListSAMLServiceProviders(ctx, orgID, pagination)
	return r0, r1, wrapErr(err)
}

func (p *errorBoundary) AddSAMLIDPKey(ctx context.Context, key *schemas.SAMLIDPKey) (*schemas.SAMLIDPKey, error) {
	r0, err := p.next.AddSAMLIDPKey(ctx, key)
	return r0, wrapErr(err)
}

func (p *errorBoundary) UpdateSAMLIDPKey(ctx context.Context, key *schemas.SAMLIDPKey) (*schemas.SAMLIDPKey, error) {
	r0, err := p.next.UpdateSAMLIDPKey(ctx, key)
	return r0, wrapErr(err)
}

func (p *errorBoundary) DeleteSAMLIDPKey(ctx context.Context, key *schemas.SAMLIDPKey) error {
	return wrapErr(p.next.DeleteSAMLIDPKey(ctx, key))
}

func (p *errorBoundary) GetSAMLIDPKeyByID(ctx context.Context, id string) (*schemas.SAMLIDPKey, error) {
	r0, err := p.next.GetSAMLIDPKeyByID(ctx, id)
	return r0, wrapErr(err)
}

func (p *errorBoundary) ListSAMLIDPKeys(ctx context.Context, orgID string) ([]*schemas.SAMLIDPKey, error) {
	r0, err := p.next.ListSAMLIDPKeys(ctx, orgID)
	return r0, wrapErr(err)
}

func (p *errorBoundary) AddWebauthnCredential(ctx context.Context, cred *schemas.WebauthnCredential) (*schemas.WebauthnCredential, error) {
	r0, err := p.next.AddWebauthnCredential(ctx, cred)
	return r0, wrapErr(err)
}

func (p *errorBoundary) UpdateWebauthnCredential(ctx context.Context, cred *schemas.WebauthnCredential) (*schemas.WebauthnCredential, error) {
	r0, err := p.next.UpdateWebauthnCredential(ctx, cred)
	return r0, wrapErr(err)
}

func (p *errorBoundary) DeleteWebauthnCredential(ctx context.Context, cred *schemas.WebauthnCredential) error {
	return wrapErr(p.next.DeleteWebauthnCredential(ctx, cred))
}

func (p *errorBoundary) GetWebauthnCredentialByID(ctx context.Context, id string) (*schemas.WebauthnCredential, error) {
	r0, err := p.next.GetWebauthnCredentialByID(ctx, id)
	return r0, wrapErr(err)
}

func (p *errorBoundary) GetWebauthnCredentialByCredentialID(ctx context.Context, credentialID string) (*schemas.WebauthnCredential, error) {
	r0, err := p.next.GetWebauthnCredentialByCredentialID(ctx, credentialID)
	return r0, wrapErr(err)
}

func (p *errorBoundary) ListWebauthnCredentialsByUserID(ctx context.Context, userID string) ([]*schemas.WebauthnCredential, error) {
	r0, err := p.next.ListWebauthnCredentialsByUserID(ctx, userID)
	return r0, wrapErr(err)
}

func (p *errorBoundary) AddOrganization(ctx context.Context, org *schemas.Organization) (*schemas.Organization, error) {
	r0, err := p.next.AddOrganization(ctx, org)
	return r0, wrapErr(err)
}

func (p *errorBoundary) GetOrganizationByID(ctx context.Context, id string) (*schemas.Organization, error) {
	r0, err := p.next.GetOrganizationByID(ctx, id)
	return r0, wrapErr(err)
}

func (p *errorBoundary) GetOrganizationByName(ctx context.Context, name string) (*schemas.Organization, error) {
	r0, err := p.next.GetOrganizationByName(ctx, name)
	return r0, wrapErr(err)
}

func (p *errorBoundary) UpdateOrganization(ctx context.Context, org *schemas.Organization) (*schemas.Organization, error) {
	r0, err := p.next.UpdateOrganization(ctx, org)
	return r0, wrapErr(err)
}

func (p *errorBoundary) DeleteOrganization(ctx context.Context, org *schemas.Organization) error {
	return wrapErr(p.next.DeleteOrganization(ctx, org))
}

func (p *errorBoundary) ListOrganizations(ctx context.Context, pagination *model.Pagination) ([]*schemas.Organization, *model.Pagination, error) {
	r0, r1, err := p.next.ListOrganizations(ctx, pagination)
	return r0, r1, wrapErr(err)
}

func (p *errorBoundary) AddOrgMembership(ctx context.Context, membership *schemas.OrgMembership) (*schemas.OrgMembership, error) {
	r0, err := p.next.AddOrgMembership(ctx, membership)
	return r0, wrapErr(err)
}

func (p *errorBoundary) GetOrgMembership(ctx context.Context, orgID string, userID string) (*schemas.OrgMembership, error) {
	r0, err := p.next.GetOrgMembership(ctx, orgID, userID)
	return r0, wrapErr(err)
}

func (p *errorBoundary) UpdateOrgMembership(ctx context.Context, membership *schemas.OrgMembership) (*schemas.OrgMembership, error) {
	r0, err := p.next.UpdateOrgMembership(ctx, membership)
	return r0, wrapErr(err)
}

func (p *errorBoundary) DeleteOrgMembership(ctx context.Context, membership *schemas.OrgMembership) error {
	return wrapErr(p.next.DeleteOrgMembership(ctx, membership))
}

func (p *errorBoundary) ListOrgMembershipsByOrg(ctx context.Context, orgID string, pagination *model.Pagination) ([]*schemas.OrgMembership, *model.Pagination, error) {
	r0, r1, err := p.next.ListOrgMembershipsByOrg(ctx, orgID, pagination)
	return r0, r1, wrapErr(err)
}

func (p *errorBoundary) ListOrgMembershipsByUser(ctx context.Context, userID string, pagination *model.Pagination) ([]*schemas.OrgMembership, *model.Pagination, error) {
	r0, r1, err := p.next.ListOrgMembershipsByUser(ctx, userID, pagination)
	return r0, r1, wrapErr(err)
}

func (p *errorBoundary) AddFederatedIdentity(ctx context.Context, identity *schemas.FederatedIdentity) (*schemas.FederatedIdentity, error) {
	r0, err := p.next.AddFederatedIdentity(ctx, identity)
	return r0, wrapErr(err)
}

func (p *errorBoundary) GetFederatedIdentity(ctx context.Context, orgID string, issuer string, subject string) (*schemas.FederatedIdentity, error) {
	r0, err := p.next.GetFederatedIdentity(ctx, orgID, issuer, subject)
	return r0, wrapErr(err)
}

func (p *errorBoundary) AddScimEndpoint(ctx context.Context, endpoint *schemas.ScimEndpoint) (*schemas.ScimEndpoint, error) {
	r0, err := p.next.AddScimEndpoint(ctx, endpoint)
	return r0, wrapErr(err)
}

func (p *errorBoundary) GetScimEndpointByID(ctx context.Context, id string) (*schemas.ScimEndpoint, error) {
	r0, err := p.next.GetScimEndpointByID(ctx, id)
	return r0, wrapErr(err)
}

func (p *errorBoundary) GetScimEndpointByOrgID(ctx context.Context, orgID string) (*schemas.ScimEndpoint, error) {
	r0, err := p.next.GetScimEndpointByOrgID(ctx, orgID)
	return r0, wrapErr(err)
}

func (p *errorBoundary) UpdateScimEndpoint(ctx context.Context, endpoint *schemas.ScimEndpoint) (*schemas.ScimEndpoint, error) {
	r0, err := p.next.UpdateScimEndpoint(ctx, endpoint)
	return r0, wrapErr(err)
}

func (p *errorBoundary) DeleteScimEndpoint(ctx context.Context, endpoint *schemas.ScimEndpoint) error {
	return wrapErr(p.next.DeleteScimEndpoint(ctx, endpoint))
}

func (p *errorBoundary) AddScimGroup(ctx context.Context, group *schemas.ScimGroup) (*schemas.ScimGroup, error) {
	r0, err := p.next.AddScimGroup(ctx, group)
	return r0, wrapErr(err)
}

func (p *errorBoundary) GetScimGroupByID(ctx context.Context, id string) (*schemas.ScimGroup, error) {
	r0, err := p.next.GetScimGroupByID(ctx, id)
	return r0, wrapErr(err)
}

func (p *errorBoundary) GetScimGroupByOrgAndDisplayName(ctx context.Context, orgID string, displayName string) (*schemas.ScimGroup, error) {
	r0, err := p.next.GetScimGroupByOrgAndDisplayName(ctx, orgID, displayName)
	return r0, wrapErr(err)
}

func (p *errorBoundary) GetScimGroupByOrgAndExternalID(ctx context.Context, orgID string, externalID string) (*schemas.ScimGroup, error) {
	r0, err := p.next.GetScimGroupByOrgAndExternalID(ctx, orgID, externalID)
	return r0, wrapErr(err)
}

func (p *errorBoundary) UpdateScimGroup(ctx context.Context, group *schemas.ScimGroup) (*schemas.ScimGroup, error) {
	r0, err := p.next.UpdateScimGroup(ctx, group)
	return r0, wrapErr(err)
}

func (p *errorBoundary) DeleteScimGroup(ctx context.Context, group *schemas.ScimGroup) error {
	return wrapErr(p.next.DeleteScimGroup(ctx, group))
}

func (p *errorBoundary) AddOrgDomain(ctx context.Context, domain *schemas.OrgDomain) (*schemas.OrgDomain, error) {
	r0, err := p.next.AddOrgDomain(ctx, domain)
	return r0, wrapErr(err)
}

func (p *errorBoundary) GetOrgDomainByDomain(ctx context.Context, domain string) (*schemas.OrgDomain, error) {
	r0, err := p.next.GetOrgDomainByDomain(ctx, domain)
	return r0, wrapErr(err)
}

func (p *errorBoundary) ListOrgDomainsByOrg(ctx context.Context, orgID string, pagination *model.Pagination) ([]*schemas.OrgDomain, *model.Pagination, error) {
	r0, r1, err := p.next.ListOrgDomainsByOrg(ctx, orgID, pagination)
	return r0, r1, wrapErr(err)
}

func (p *errorBoundary) DeleteOrgDomain(ctx context.Context, domain string) error {
	return wrapErr(p.next.DeleteOrgDomain(ctx, domain))
}

func (p *errorBoundary) DeleteOrgDomainsByOrg(ctx context.Context, orgID string) error {
	return wrapErr(p.next.DeleteOrgDomainsByOrg(ctx, orgID))
}
