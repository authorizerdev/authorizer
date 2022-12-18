package couchbase

import (
	"context"
	"errors"
	"fmt"
	"os"
	"reflect"

	"github.com/authorizerdev/authorizer/server/constants"
	"github.com/authorizerdev/authorizer/server/db/models"
	"github.com/authorizerdev/authorizer/server/memorystore"
	"github.com/couchbase/gocb/v2"
)

// TODO change following provider to new db provider
type provider struct {
	db        *gocb.Scope
	scopeName string
}

// NewProvider returns a new SQL provider
// TODO change following provider to new db provider
func NewProvider() (*provider, error) {
	// scopeName := os.Getenv(constants.EnvCouchbaseScope)
	bucketName := os.Getenv(constants.EnvCouchbaseBucket)
	scopeName := os.Getenv(constants.EnvCouchbaseScope)

	dbURL := memorystore.RequiredEnvStoreObj.GetRequiredEnv().DatabaseURL
	userName := memorystore.RequiredEnvStoreObj.GetRequiredEnv().DatabaseUsername
	password := memorystore.RequiredEnvStoreObj.GetRequiredEnv().DatabasePassword

	opts := gocb.ClusterOptions{
		Username: userName,
		Password: password,
	}

	cluster, err := gocb.Connect(dbURL, opts)

	if err != nil {
		return nil, err
	}

	// To create the bucket and scope if not exist
	bucket, err := CreateBucketAndScope(cluster, bucketName, scopeName)

	if err != nil {
		return nil, err
	}

	scope := bucket.Scope(scopeName)

	scopeIdentifier := fmt.Sprintf("%s.%s", bucketName, scopeName)

	v := reflect.ValueOf(models.Collections)
	for i := 0; i < v.NumField(); i++ {
		field := v.Field(i)
		user := gocb.CollectionSpec{
			Name:      field.String(),
			ScopeName: scopeName,
		}
		collectionOpts := gocb.CreateCollectionOptions{
			Context: context.TODO(),
		}
		_ = bucket.Collections().CreateCollection(user, &collectionOpts)
		// if err != nil && !errors.Is(err, gocb.ErrCollectionExists) {
		// 	return nil, err
		// }
		indexQuery := fmt.Sprintf("CREATE PRIMARY INDEX ON %s.%s", scopeIdentifier, field.String())
		scope.Query(indexQuery, nil)
	}

	indices := GetIndex(scopeIdentifier)
	for i := 0; i < v.NumField(); i++ {
		field := v.Field(i)
		for _, indexQuery := range indices[field.String()] {
			scope.Query(indexQuery, nil)
		}
	}
	return &provider{
		db:        scope,
		scopeName: scopeIdentifier,
	}, nil
}

func CreateBucketAndScope(cluster *gocb.Cluster, bucketName string, scopeName string) (*gocb.Bucket, error) {
	settings := gocb.BucketSettings{
		Name:            bucketName,
		RAMQuotaMB:      1000,
		NumReplicas:     1,
		BucketType:      gocb.CouchbaseBucketType,
		EvictionPolicy:  gocb.EvictionPolicyTypeValueOnly,
		FlushEnabled:    true,
		CompressionMode: gocb.CompressionModeActive,
	}

	err := cluster.Buckets().CreateBucket(gocb.CreateBucketSettings{
		BucketSettings:         settings,
		ConflictResolutionType: gocb.ConflictResolutionTypeSequenceNumber,
	}, nil)

	bucket := cluster.Bucket(bucketName)

	if err != nil && !errors.Is(err, gocb.ErrBucketExists) {
		return bucket, err
	}

	err = bucket.Collections().CreateScope(scopeName, nil)

	if err != nil && !errors.Is(err, gocb.ErrScopeExists) {
		return bucket, err
	}

	return bucket, nil
}

func GetIndex(scopeName string) map[string][]string {
	indices := make(map[string][]string)

	// User Index
	userIndex1 := fmt.Sprintf("CREATE INDEX userEmailIndex ON %s.%s(email)", scopeName, models.Collections.User)
	userIndex2 := fmt.Sprintf("CREATE INDEX userPhoneIndex ON %s.%s(phone_number)", scopeName, models.Collections.User)
	indices[models.Collections.User] = []string{userIndex1, userIndex2}

	// VerificationRequest
	verificationIndex1 := fmt.Sprintf("CREATE INDEX verificationRequestTokenIndex ON %s.%s(token)", scopeName, models.Collections.VerificationRequest)
	verificationIndex2 := fmt.Sprintf("CREATE INDEX verificationRequestEmailAndIdentifierIndex ON %s.%s(email,identifier)", scopeName, models.Collections.VerificationRequest)
	indices[models.Collections.VerificationRequest] = []string{verificationIndex1, verificationIndex2}

	// Session index
	sessionIndex1 := fmt.Sprintf("CREATE INDEX SessionUserIdIndex ON %s.%s(user_id)", scopeName, models.Collections.Session)
	indices[models.Collections.Session] = []string{sessionIndex1}

	// Webhook index
	webhookIndex1 := fmt.Sprintf("CREATE INDEX webhookEventNameIndex ON %s.%s(event_name)", scopeName, models.Collections.Webhook)
	indices[models.Collections.Webhook] = []string{webhookIndex1}

	// WebhookLog index
	webhookLogIndex1 := fmt.Sprintf("CREATE INDEX webhookLogIdIndex ON %s.%s(webhook_id)", scopeName, models.Collections.WebhookLog)
	indices[models.Collections.Webhook] = []string{webhookLogIndex1}

	// WebhookLog index
	emailTempIndex1 := fmt.Sprintf("CREATE INDEX EmailTemplateEventNameIndex ON %s.%s(event_name)", scopeName, models.Collections.EmailTemplate)
	indices[models.Collections.EmailTemplate] = []string{emailTempIndex1}

	// OTP index
	otpIndex1 := fmt.Sprintf("CREATE INDEX OTPEmailIndex ON %s.%s(email)", scopeName, models.Collections.OTP)
	indices[models.Collections.OTP] = []string{otpIndex1}

	return indices
}
