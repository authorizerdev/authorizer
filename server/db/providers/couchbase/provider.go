package couchbase

import (
	"context"
	"os"
	"reflect"

	"github.com/authorizerdev/authorizer/server/constants"
	"github.com/authorizerdev/authorizer/server/db/models"
	"github.com/authorizerdev/authorizer/server/memorystore"
	"github.com/couchbase/gocb/v2"
)

// TODO change following provider to new db provider
type provider struct {
	db *gocb.Bucket
}

// NewProvider returns a new SQL provider
// TODO change following provider to new db provider
func NewProvider() (*provider, error) {
	scopeName := os.Getenv(constants.EnvCouchbaseScope)
	bucketName := os.Getenv(constants.EnvCouchbaseBucket)
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
	bucket := cluster.Bucket(bucketName)

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
	}

	return &provider{
		db: bucket,
	}, nil
}
