package dynamodb

import (
	"context"
	"fmt"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	awsconfig "github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/credentials"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb"
	"github.com/rs/zerolog"

	"github.com/authorizerdev/authorizer/internal/config"
)

// Dependencies struct the dynamodb data store provider
type Dependencies struct {
	Log *zerolog.Logger
}

type provider struct {
	config       *config.Config
	dependencies *Dependencies
	client       *dynamodb.Client
}

// NewProvider returns a new Dynamo provider using AWS SDK for Go v2.
func NewProvider(cfg *config.Config, deps *Dependencies) (*provider, error) {
	dbURL := cfg.DatabaseURL
	awsRegion := cfg.AWSRegion
	awsAccessKeyID := cfg.AWSAccessKeyID
	awsSecretAccessKey := cfg.AWSSecretAccessKey

	region := awsRegion
	if region == "" {
		region = "us-east-1"
	}

	loadOpts := []func(*awsconfig.LoadOptions) error{
		awsconfig.WithRegion(region),
	}

	if awsAccessKeyID != "" && awsSecretAccessKey != "" {
		loadOpts = append(loadOpts, awsconfig.WithCredentialsProvider(
			credentials.NewStaticCredentialsProvider(awsAccessKeyID, awsSecretAccessKey, "")))
	} else if dbURL != "" {
		deps.Log.Info().Msg("Using DB URL for dynamodb")
		loadOpts = append(loadOpts, awsconfig.WithCredentialsProvider(
			credentials.NewStaticCredentialsProvider("key", "key", "")))
	} else {
		deps.Log.Info().Msg("Using default AWS credentials config from system for dynamodb")
	}

	if dbURL != "" {
		resolver := aws.EndpointResolverWithOptionsFunc(func(service, region string, options ...interface{}) (aws.Endpoint, error) {
			if service == dynamodb.ServiceID {
				return aws.Endpoint{
					URL:               dbURL,
					HostnameImmutable: true,
				}, nil
			}
			return aws.Endpoint{}, &aws.EndpointNotFoundError{}
		})
		loadOpts = append(loadOpts, awsconfig.WithEndpointResolverWithOptions(resolver))
	}

	awsCfg, err := awsconfig.LoadDefaultConfig(context.Background(), loadOpts...)
	if err != nil {
		return nil, fmt.Errorf("aws config: %w", err)
	}

	client := dynamodb.NewFromConfig(awsCfg, func(o *dynamodb.Options) {
		o.RetryMaxAttempts = 3
	})

	p := &provider{
		client:       client,
		config:       cfg,
		dependencies: deps,
	}

	createCtx, cancel := context.WithTimeout(context.Background(), 3*time.Minute)
	defer cancel()

	if err := p.ensureTables(createCtx); err != nil {
		return nil, err
	}

	return p, nil
}

// Close is a no-op; the AWS SDK v2 client needs no explicit shutdown for typical use.
func (p *provider) Close() error {
	return nil
}
