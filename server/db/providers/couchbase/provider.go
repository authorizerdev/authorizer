package couchbase

import (
	"fmt"
	"os"

	"github.com/authorizerdev/authorizer/server/constants"
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
	bucket := cluster.Bucket(bucketName).Scope(scopeName)
	scopeName = fmt.Sprintf("%s.%s", bucketName, scopeName)
	// v := reflect.ValueOf(models.Collections)
	// fmt.Println("called in v", v)

	// for i := 0; i < v.NumField(); i++ {

	// 	field := v.Field(i)
	// 	fmt.Println("called in v", field)

	// 	user := gocb.CollectionSpec{
	// 		Name:      field.String(),
	// 		ScopeName: scopeName,
	// 	}
	// 	collectionOpts := gocb.CreateCollectionOptions{
	// 		Context: context.TODO(),
	// 	}
	// 	err = bucket.Collections().CreateCollection(user, &collectionOpts)
	// 	fmt.Println("2 called in oprovuider", err)

	// }
	// fmt.Println("called in oprovuider")
	return &provider{
		db:        bucket,
		scopeName: scopeName,
	}, nil
}
