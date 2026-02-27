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
	// Wait until the cluster is ready
	err = cluster.WaitUntilReady(30*time.Second, nil)
	if err != nil {
		return nil, err
	}
	// To create the bucket and scope if not exist
	bucket, err := createBucketAndScope(cluster, bucketName, scopeName, ramQuota)
	if err != nil {
		return nil, err
	}
	scope := bucket.Scope(scopeName)
	scopeIdentifier := fmt.Sprintf("%s.%s", bucketName, scopeName)
	v := reflect.ValueOf(schemas.Collections)
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
		// TODO: find how to fix this sleep time.
		// Add wait time for successful collection creation
		time.Sleep(5 * time.Second)
		indexQuery := fmt.Sprintf("CREATE PRIMARY INDEX ON %s.%s", scopeIdentifier, collectionName.String())
		_, err = scope.Query(indexQuery, nil)
		if err != nil && !strings.Contains(err.Error(), "The index #primary already exists") {
			return nil, err
		}
	}

	indices := getIndex(scopeIdentifier)
	for i := 0; i < v.NumField(); i++ {
		field := v.Field(i)
		for _, indexQuery := range indices[field.String()] {
			scope.Query(indexQuery, nil)
		}
	}
	return &provider{
		config:       config,
		dependencies: deps,
		db:           scope,
		scopeName:    scopeIdentifier,
	}, nil
}

func createBucketAndScope(cluster *gocb.Cluster, bucketName string, scopeName string, ramQuota string) (*gocb.Bucket, error) {
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
	// Wait until bucket is ready
	err = bucket.WaitUntilReady(30*time.Second, nil)
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
	indices[schemas.Collections.Webhook] = []string{webhookLogIndex1}

	// WebhookLog index
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

	return indices
}
