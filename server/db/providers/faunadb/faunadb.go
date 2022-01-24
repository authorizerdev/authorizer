package faunadb

import (
	"errors"
	"log"
	"strings"

	"github.com/authorizerdev/authorizer/server/constants"
	"github.com/authorizerdev/authorizer/server/db/models"
	"github.com/authorizerdev/authorizer/server/envstore"
	f "github.com/fauna/faunadb-go/v5/faunadb"
)

type provider struct {
	db *f.FaunaClient
}

// NewProvider returns a new faunadb provider
func NewProvider() (*provider, error) {
	secret := ""
	dbURL := "https://db.fauna.com"

	// secret,url is stored in DATABASE_URL
	dbURLSplit := strings.Split(envstore.EnvInMemoryStoreObj.GetStringStoreEnvVariable(constants.EnvKeyDatabaseURL), ":")
	secret = dbURLSplit[0]

	if len(dbURLSplit) > 1 {
		dbURL = dbURLSplit[1]
	}

	client := f.NewFaunaClient(secret, f.Endpoint(dbURL))
	if client == nil {
		return nil, errors.New("failed to create faunadb client")
	}

	_, err := client.Query(
		f.CreateCollection(f.Obj{"name": models.Collections.Env}))
	if err != nil {
		log.Println("error:", err)
	}

	_, err = client.Query(
		f.CreateIndex(
			f.Obj{
				"name":   "env_id",
				"source": f.Collection(models.Collections.Env),
				"values": "_id",
				"unique": true,
			}))
	if err != nil {
		log.Println("error:", err)
	}

	_, err = client.Query(
		f.CreateIndex(
			f.Obj{
				"name":   "env_key",
				"source": f.Collection(models.Collections.Env),
				"values": "_key",
				"unique": true,
			}))
	if err != nil {
		log.Println("error:", err)
	}

	_, err = client.Query(
		f.CreateCollection(f.Obj{"name": models.Collections.User}))
	if err != nil {
		log.Println("error:", err)
	}

	_, err = client.Query(
		f.CreateIndex(
			f.Obj{
				"name":   "_id",
				"source": f.Collection(models.Collections.User),
				"unique": true,
			}))
	if err != nil {
		log.Println("error:", err)
	}

	_, err = client.Query(
		f.CreateIndex(
			f.Obj{
				"name":   "_key",
				"source": f.Collection(models.Collections.User),
				"unique": true,
			}))
	if err != nil {
		log.Println("error:", err)
	}

	_, err = client.Query(
		f.CreateIndex(
			f.Obj{
				"name":   "email",
				"source": f.Collection(models.Collections.User),
				"unique": true,
			}))
	if err != nil {
		log.Println("error:", err)
	}

	_, err = client.Query(
		f.CreateCollection(f.Obj{"name": models.Collections.Session}))
	if err != nil {
		log.Println("error:", err)
	}

	_, err = client.Query(
		f.CreateIndex(
			f.Obj{
				"name":   "_id",
				"source": f.Collection(models.Collections.Session),
				"unique": true,
			}))
	if err != nil {
		log.Println("error:", err)
	}

	_, err = client.Query(
		f.CreateIndex(
			f.Obj{
				"name":   "_key",
				"source": f.Collection(models.Collections.Session),
				"unique": true,
			}))
	if err != nil {
		log.Println("error:", err)
	}

	_, err = client.Query(
		f.CreateCollection(f.Obj{"name": models.Collections.VerificationRequest}))
	if err != nil {
		log.Println("error:", err)
	}

	_, err = client.Query(
		f.CreateIndex(
			f.Obj{
				"name":   "_id",
				"source": f.Collection(models.Collections.VerificationRequest),
				"unique": true,
			}))
	if err != nil {
		log.Println("error:", err)
	}

	_, err = client.Query(
		f.CreateIndex(
			f.Obj{
				"name":   "_key",
				"source": f.Collection(models.Collections.VerificationRequest),
				"unique": true,
			}))
	if err != nil {
		log.Println("error:", err)
	}

	return &provider{
		db: client,
	}, nil
}
