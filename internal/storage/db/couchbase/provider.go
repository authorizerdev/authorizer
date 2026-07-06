package couchbase

import (
	"context"
	"errors"
	"fmt"
	"reflect"
	"strconv"
	"strings"
	"time"

	"github.com/couchbase/gocb/v2"
	"github.com/rs/zerolog"

	"github.com/authorizerdev/authorizer/internal/config"
	"github.com/authorizerdev/authorizer/internal/storage/schemas"
)

// Dependencies struct the couchbase data store provider
type Dependencies struct {
	Log *zerolog.Logger
}

const (
	defaultBucketName = "authorizer"
	defaultScope      = "_default"
)

type provider struct {
	config       *config.Config
	dependencies *Dependencies

	cluster   *gocb.Cluster
	db        *gocb.Scope
	scopeName string
}

// NewProvider returns a new Couchbase provider
func NewProvider(config *config.Config, deps *Dependencies) (*provider, error) {
	bucketName := config.CouchBaseBucket
	ramQuota := config.CouchBaseRamQuota
	scopeName := config.CouchBaseScope
	dbURL := config.DatabaseURL
	userName := config.DatabaseUsername
	password := config.DatabasePassword
	waitTimeout := 30 * time.Second
	if config.CouchBaseWaitTimeout > 0 {
		waitTimeout = time.Duration(config.CouchBaseWaitTimeout) * time.Second
	}
	opts := gocb.ClusterOptions{
		Username: userName,
		Password: password,
	}
	if bucketName == "" {
		bucketName = defaultBucketName
	}
	if scopeName == "" {
		scopeName = defaultScope
	}
	cluster, err := gocb.Connect(dbURL, opts)
	if err != nil {
		return nil, err
	}
	// Wait until the cluster (including Query service) is ready
	clusterWaitOpts := &gocb.WaitUntilReadyOptions{
		ServiceTypes: []gocb.ServiceType{
			gocb.ServiceTypeQuery,
		},
	}
	err = cluster.WaitUntilReady(waitTimeout, clusterWaitOpts)
	if err != nil {
		return nil, err
	}
	// To create the bucket and scope if not exist
	bucket, err := createBucketAndScope(cluster, bucketName, scopeName, ramQuota, waitTimeout)
	if err != nil {
		return nil, err
	}
	scope := bucket.Scope(scopeName)
	scopeIdentifier := fmt.Sprintf("%s.%s", bucketName, scopeName)
	v := reflect.ValueOf(schemas.Collections)
	indexQueryTimeout := 30 * time.Second
	if waitTimeout > indexQueryTimeout {
		indexQueryTimeout = waitTimeout / 4
	}

	for i := 0; i < v.NumField(); i++ {
		collectionName := v.Field(i)
		user := gocb.CollectionSpec{
			Name:      collectionName.String(),
			ScopeName: scopeName,
		}
		collectionOpts := gocb.CreateCollectionOptions{
			Context: context.TODO(),
		}
		err = bucket.Collections().CreateCollection(user, &collectionOpts)
		if err != nil && !errors.Is(err, gocb.ErrCollectionExists) {
			return nil, err
		}
	}

	for i := 0; i < v.NumField(); i++ {
		collectionName := v.Field(i).String()
		indexQuery := fmt.Sprintf("CREATE PRIMARY INDEX ON %s.%s", scopeIdentifier, collectionName)
		if err := execIndexQuery(scope, indexQuery, indexQueryTimeout); err != nil {
			return nil, err
		}
	}

	indices := getIndex(scopeIdentifier)
	for i := 0; i < v.NumField(); i++ {
		field := v.Field(i)
		for _, indexQuery := range indices[field.String()] {
			if err := execIndexQuery(scope, indexQuery, indexQueryTimeout); err != nil {
				return nil, fmt.Errorf("couchbase secondary index: %s: %w", indexQuery, err)
			}
		}
	}
	return &provider{
		config:       config,
		dependencies: deps,
		cluster:      cluster,
		db:           scope,
		scopeName:    scopeIdentifier,
	}, nil
}

// Close shuts down the Couchbase cluster connection.
func (p *provider) Close() error {
	if p.cluster == nil {
		return nil
	}
	return p.cluster.Close(&gocb.ClusterCloseOptions{})
}

func isIndexExistsErr(msg string) bool {
	return strings.Contains(msg, "already exists") ||
		strings.Contains(msg, "The index #primary already exists") ||
		(strings.Contains(msg, "Index") && strings.Contains(msg, "already"))
}

func isTransientQueryErr(err error) bool {
	if err == nil {
		return false
	}
	msg := strings.ToLower(err.Error())
	return strings.Contains(msg, "eof") ||
		strings.Contains(msg, "connection reset") ||
		strings.Contains(msg, "not_ready") ||
		strings.Contains(msg, "collection_not_found") ||
		strings.Contains(msg, "indexnotfound") ||
		strings.Contains(msg, "servicenotavailable") ||
		strings.Contains(msg, "unambiguous timeout") ||
		strings.Contains(msg, "temporary failure")
}

// execIndexQuery runs a CREATE INDEX statement with retries for transient query-service errors.
func execIndexQuery(scope *gocb.Scope, query string, timeout time.Duration) error {
	deadline := time.Now().Add(timeout)
	delay := 500 * time.Millisecond
	for {
		_, err := scope.Query(query, nil)
		if err == nil {
			return nil
		}
		if isIndexExistsErr(err.Error()) {
			return nil
		}
		if time.Now().After(deadline) || !isTransientQueryErr(err) {
			return err
		}
		time.Sleep(delay)
		if delay < 3*time.Second {
			delay += 500 * time.Millisecond
		}
	}
}

func createBucketAndScope(cluster *gocb.Cluster, bucketName string, scopeName string, ramQuota string, waitTimeout time.Duration) (*gocb.Bucket, error) {
	if ramQuota == "" {
		ramQuota = "1000"
	}
	bucketRAMQuota, err := strconv.ParseInt(ramQuota, 10, 64)
	if err != nil {
		return nil, err
	}
	settings := gocb.BucketSettings{
		Name:            bucketName,
		RAMQuotaMB:      uint64(bucketRAMQuota),
		BucketType:      gocb.CouchbaseBucketType,
		EvictionPolicy:  gocb.EvictionPolicyTypeValueOnly,
		FlushEnabled:    true,
		CompressionMode: gocb.CompressionModeActive,
	}
	shouldCreateBucket := false
	// check if bucket exists
	_, err = cluster.Buckets().GetBucket(bucketName, nil)
	if err != nil {
		// bucket not found
		shouldCreateBucket = true
	}
	if shouldCreateBucket {
		err = cluster.Buckets().CreateBucket(gocb.CreateBucketSettings{
			BucketSettings:         settings,
			ConflictResolutionType: gocb.ConflictResolutionTypeSequenceNumber,
		}, nil)
		if err != nil {
			return nil, err
		}
	}
	bucket := cluster.Bucket(bucketName)
	// Wait until bucket (including KV and Query services) is ready
	bucketWaitOpts := &gocb.WaitUntilReadyOptions{
		ServiceTypes: []gocb.ServiceType{
			gocb.ServiceTypeKeyValue,
			gocb.ServiceTypeQuery,
		},
	}
	err = bucket.WaitUntilReady(waitTimeout, bucketWaitOpts)
	if err != nil {
		return nil, err
	}
	if scopeName != defaultScope {
		err = bucket.Collections().CreateScope(scopeName, nil)
		if err != nil && !errors.Is(err, gocb.ErrScopeExists) {
			return nil, err
		}
	}
	return bucket, nil
}

func getIndex(scopeName string) map[string][]string {
	indices := make(map[string][]string)

	// User Index
	userIndex1 := fmt.Sprintf("CREATE INDEX userEmailIndex ON %s.%s(email)", scopeName, schemas.Collections.User)
	userIndex2 := fmt.Sprintf("CREATE INDEX userPhoneIndex ON %s.%s(phone_number)", scopeName, schemas.Collections.User)
	indices[schemas.Collections.User] = []string{userIndex1, userIndex2}

	// VerificationRequest
	verificationIndex1 := fmt.Sprintf("CREATE INDEX verificationRequestTokenIndex ON %s.%s(token)", scopeName, schemas.Collections.VerificationRequest)
	verificationIndex2 := fmt.Sprintf("CREATE INDEX verificationRequestEmailAndIdentifierIndex ON %s.%s(email,identifier)", scopeName, schemas.Collections.VerificationRequest)
	indices[schemas.Collections.VerificationRequest] = []string{verificationIndex1, verificationIndex2}

	// Session index
	sessionIndex1 := fmt.Sprintf("CREATE INDEX SessionUserIdIndex ON %s.%s(user_id)", scopeName, schemas.Collections.Session)
	indices[schemas.Collections.Session] = []string{sessionIndex1}

	// Webhook index
	webhookIndex1 := fmt.Sprintf("CREATE INDEX webhookEventNameIndex ON %s.%s(event_name)", scopeName, schemas.Collections.Webhook)
	indices[schemas.Collections.Webhook] = []string{webhookIndex1}

	// WebhookLog index
	webhookLogIndex1 := fmt.Sprintf("CREATE INDEX webhookLogIdIndex ON %s.%s(webhook_id)", scopeName, schemas.Collections.WebhookLog)
	indices[schemas.Collections.WebhookLog] = []string{webhookLogIndex1}

	// EmailTemplate index
	emailTempIndex1 := fmt.Sprintf("CREATE INDEX EmailTemplateEventNameIndex ON %s.%s(event_name)", scopeName, schemas.Collections.EmailTemplate)
	indices[schemas.Collections.EmailTemplate] = []string{emailTempIndex1}

	// OTP index
	otpIndex1 := fmt.Sprintf("CREATE INDEX OTPEmailIndex ON %s.%s(email)", scopeName, schemas.Collections.OTP)
	indices[schemas.Collections.OTP] = []string{otpIndex1}

	// OTP index
	otpIndex2 := fmt.Sprintf("CREATE INDEX OTPPhoneNumberIndex ON %s.%s(phone_number)", scopeName, schemas.Collections.OTP)
	indices[schemas.Collections.OTP] = []string{otpIndex1, otpIndex2}

	// SessionToken indexes
	sessionTokenIndex1 := fmt.Sprintf("CREATE INDEX SessionTokenUserIdKeyIndex ON %s.%s(user_id, key_name)", scopeName, schemas.Collections.SessionToken)
	sessionTokenIndex2 := fmt.Sprintf("CREATE INDEX SessionTokenExpiresAtIndex ON %s.%s(expires_at)", scopeName, schemas.Collections.SessionToken)
	indices[schemas.Collections.SessionToken] = []string{sessionTokenIndex1, sessionTokenIndex2}

	// MFASession indexes
	mfaSessionIndex1 := fmt.Sprintf("CREATE INDEX MFASessionUserIdKeyIndex ON %s.%s(user_id, key_name)", scopeName, schemas.Collections.MFASession)
	mfaSessionIndex2 := fmt.Sprintf("CREATE INDEX MFASessionExpiresAtIndex ON %s.%s(expires_at)", scopeName, schemas.Collections.MFASession)
	indices[schemas.Collections.MFASession] = []string{mfaSessionIndex1, mfaSessionIndex2}

	// OAuthState index
	oauthStateIndex1 := fmt.Sprintf("CREATE INDEX OAuthStateKeyIndex ON %s.%s(state_key)", scopeName, schemas.Collections.OAuthState)
	indices[schemas.Collections.OAuthState] = []string{oauthStateIndex1}

	// AuditLog indexes
	auditLogIndex1 := fmt.Sprintf("CREATE INDEX AuditLogActorIdIndex ON %s.%s(actor_id)", scopeName, schemas.Collections.AuditLog)
	auditLogIndex2 := fmt.Sprintf("CREATE INDEX AuditLogActionIndex ON %s.%s(action)", scopeName, schemas.Collections.AuditLog)
	auditLogIndex3 := fmt.Sprintf("CREATE INDEX AuditLogCreatedAtIndex ON %s.%s(created_at)", scopeName, schemas.Collections.AuditLog)
	indices[schemas.Collections.AuditLog] = []string{auditLogIndex1, auditLogIndex2, auditLogIndex3}

	// TrustedIssuer indexes
	trustedIssuerIndex1 := fmt.Sprintf("CREATE INDEX TrustedIssuerIssuerURLIndex ON %s.%s(issuer_url)", scopeName, schemas.Collections.TrustedIssuer)
	trustedIssuerIndex2 := fmt.Sprintf("CREATE INDEX TrustedIssuerServiceAccountIdIndex ON %s.%s(service_account_id)", scopeName, schemas.Collections.TrustedIssuer)
	indices[schemas.Collections.TrustedIssuer] = []string{trustedIssuerIndex1, trustedIssuerIndex2}

	return indices
}
