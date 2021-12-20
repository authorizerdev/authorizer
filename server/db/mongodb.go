package db

import (
	"context"
	"log"
	"time"

	"github.com/authorizerdev/authorizer/server/constants"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
	"go.mongodb.org/mongo-driver/mongo/readpref"
)

func initMongodb() (*mongo.Database, error) {
	log.Println("=> connecting to:", constants.DATABASE_URL)
	mongodbOptions := options.Client().ApplyURI(constants.DATABASE_URL)
	maxWait := time.Duration(5 * time.Second)
	mongodbOptions.ConnectTimeout = &maxWait
	mongoClient, err := mongo.NewClient(mongodbOptions)
	if err != nil {
		log.Println("=> err...:", err)
		return nil, err
	}
	ctx, _ := context.WithTimeout(context.Background(), 30*time.Second)
	err = mongoClient.Connect(ctx)
	if err != nil {
		return nil, err
	}
	defer mongoClient.Disconnect(ctx)

	err = mongoClient.Ping(ctx, readpref.Primary())
	if err != nil {
		return nil, err
	}

	mongodb := mongoClient.Database(constants.DATABASE_NAME, &options.DatabaseOptions{})
	return mongodb, nil
}
