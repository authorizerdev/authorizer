package dynamodb

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/authorizerdev/authorizer/internal/storage/schemas"
	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb/types"
	"github.com/aws/smithy-go"
)

func gsi(name, hashAttr string) types.GlobalSecondaryIndex {
	return types.GlobalSecondaryIndex{
		IndexName: aws.String(name),
		KeySchema: []types.KeySchemaElement{
			{AttributeName: aws.String(hashAttr), KeyType: types.KeyTypeHash},
		},
		Projection: &types.Projection{ProjectionType: types.ProjectionTypeAll},
	}
}

func createTable(ctx context.Context, client *dynamodb.Client, name string, hashAttr string, attrs []types.AttributeDefinition, gsis []types.GlobalSecondaryIndex) error {
	in := &dynamodb.CreateTableInput{
		TableName:            aws.String(name),
		BillingMode:          types.BillingModePayPerRequest,
		AttributeDefinitions: attrs,
		KeySchema: []types.KeySchemaElement{
			{AttributeName: aws.String(hashAttr), KeyType: types.KeyTypeHash},
		},
	}
	if len(gsis) > 0 {
		in.GlobalSecondaryIndexes = gsis
	}
	_, err := client.CreateTable(ctx, in)
	if err != nil {
		var apiErr smithy.APIError
		if errors.As(err, &apiErr) && apiErr.ErrorCode() == "ResourceInUseException" {
			return nil
		}
		return err
	}
	w := dynamodb.NewTableExistsWaiter(client)
	return w.Wait(ctx, &dynamodb.DescribeTableInput{TableName: aws.String(name)}, 5*time.Minute)
}

func (p *provider) ensureTables(ctx context.Context) error {
	tables := []struct {
		name string
		hash string
		attr []types.AttributeDefinition
		gsi  []types.GlobalSecondaryIndex
	}{
		{
			name: schemas.Collections.User,
			hash: "id",
			attr: []types.AttributeDefinition{
				{AttributeName: aws.String("id"), AttributeType: types.ScalarAttributeTypeS},
				{AttributeName: aws.String("email"), AttributeType: types.ScalarAttributeTypeS},
				{AttributeName: aws.String("external_id"), AttributeType: types.ScalarAttributeTypeS},
			},
			gsi: []types.GlobalSecondaryIndex{
				gsi("email", "email"),
				gsi("external_id", "external_id"),
			},
		},
		{
			name: schemas.Collections.Session,
			hash: "id",
			attr: []types.AttributeDefinition{
				{AttributeName: aws.String("id"), AttributeType: types.ScalarAttributeTypeS},
				{AttributeName: aws.String("user_id"), AttributeType: types.ScalarAttributeTypeS},
			},
			gsi: []types.GlobalSecondaryIndex{gsi("user_id", "user_id")},
		},
		{
			name: schemas.Collections.Env,
			hash: "id",
			attr: []types.AttributeDefinition{
				{AttributeName: aws.String("id"), AttributeType: types.ScalarAttributeTypeS},
			},
		},
		{
			name: schemas.Collections.Webhook,
			hash: "id",
			attr: []types.AttributeDefinition{
				{AttributeName: aws.String("id"), AttributeType: types.ScalarAttributeTypeS},
				{AttributeName: aws.String("event_name"), AttributeType: types.ScalarAttributeTypeS},
			},
			gsi: []types.GlobalSecondaryIndex{gsi("event_name", "event_name")},
		},
		{
			name: schemas.Collections.WebhookLog,
			hash: "id",
			attr: []types.AttributeDefinition{
				{AttributeName: aws.String("id"), AttributeType: types.ScalarAttributeTypeS},
				{AttributeName: aws.String("webhook_id"), AttributeType: types.ScalarAttributeTypeS},
			},
			gsi: []types.GlobalSecondaryIndex{gsi("webhook_id", "webhook_id")},
		},
		{
			name: schemas.Collections.EmailTemplate,
			hash: "id",
			attr: []types.AttributeDefinition{
				{AttributeName: aws.String("id"), AttributeType: types.ScalarAttributeTypeS},
				{AttributeName: aws.String("event_name"), AttributeType: types.ScalarAttributeTypeS},
			},
			gsi: []types.GlobalSecondaryIndex{gsi("event_name", "event_name")},
		},
		{
			name: schemas.Collections.OTP,
			hash: "id",
			attr: []types.AttributeDefinition{
				{AttributeName: aws.String("id"), AttributeType: types.ScalarAttributeTypeS},
				{AttributeName: aws.String("email"), AttributeType: types.ScalarAttributeTypeS},
			},
			gsi: []types.GlobalSecondaryIndex{gsi("email", "email")},
		},
		{
			name: schemas.Collections.VerificationRequest,
			hash: "id",
			attr: []types.AttributeDefinition{
				{AttributeName: aws.String("id"), AttributeType: types.ScalarAttributeTypeS},
				{AttributeName: aws.String("token"), AttributeType: types.ScalarAttributeTypeS},
			},
			gsi: []types.GlobalSecondaryIndex{gsi("token", "token")},
		},
		{
			name: schemas.Collections.Authenticators,
			hash: "id",
			attr: []types.AttributeDefinition{
				{AttributeName: aws.String("id"), AttributeType: types.ScalarAttributeTypeS},
				{AttributeName: aws.String("user_id"), AttributeType: types.ScalarAttributeTypeS},
			},
			gsi: []types.GlobalSecondaryIndex{gsi("user_id", "user_id")},
		},
		{
			name: schemas.Collections.SessionToken,
			hash: "id",
			attr: []types.AttributeDefinition{
				{AttributeName: aws.String("id"), AttributeType: types.ScalarAttributeTypeS},
				{AttributeName: aws.String("user_id"), AttributeType: types.ScalarAttributeTypeS},
			},
			gsi: []types.GlobalSecondaryIndex{gsi("user_id", "user_id")},
		},
		{
			name: schemas.Collections.MFASession,
			hash: "id",
			attr: []types.AttributeDefinition{
				{AttributeName: aws.String("id"), AttributeType: types.ScalarAttributeTypeS},
				{AttributeName: aws.String("user_id"), AttributeType: types.ScalarAttributeTypeS},
			},
			gsi: []types.GlobalSecondaryIndex{gsi("user_id", "user_id")},
		},
		{
			name: schemas.Collections.OAuthState,
			hash: "id",
			attr: []types.AttributeDefinition{
				{AttributeName: aws.String("id"), AttributeType: types.ScalarAttributeTypeS},
				{AttributeName: aws.String("state_key"), AttributeType: types.ScalarAttributeTypeS},
			},
			gsi: []types.GlobalSecondaryIndex{gsi("state_key", "state_key")},
		},
		{
			name: schemas.Collections.AuditLog,
			hash: "id",
			attr: []types.AttributeDefinition{
				{AttributeName: aws.String("id"), AttributeType: types.ScalarAttributeTypeS},
				{AttributeName: aws.String("actor_id"), AttributeType: types.ScalarAttributeTypeS},
				{AttributeName: aws.String("action"), AttributeType: types.ScalarAttributeTypeS},
			},
			gsi: []types.GlobalSecondaryIndex{
				gsi("actor_id", "actor_id"),
				gsi("action", "action"),
			},
		},
		{
			name: schemas.Collections.Client,
			hash: "id",
			attr: []types.AttributeDefinition{
				{AttributeName: aws.String("id"), AttributeType: types.ScalarAttributeTypeS},
				{AttributeName: aws.String("client_id"), AttributeType: types.ScalarAttributeTypeS},
			},
			gsi: []types.GlobalSecondaryIndex{gsi("client_id", "client_id")},
		},
		{
			name: schemas.Collections.TrustedIssuer,
			hash: "id",
			attr: []types.AttributeDefinition{
				{AttributeName: aws.String("id"), AttributeType: types.ScalarAttributeTypeS},
				{AttributeName: aws.String("issuer_url"), AttributeType: types.ScalarAttributeTypeS},
				{AttributeName: aws.String("client_id"), AttributeType: types.ScalarAttributeTypeS},
			},
			gsi: []types.GlobalSecondaryIndex{
				gsi("issuer_url", "issuer_url"),
				gsi("client_id", "client_id"),
			},
		},
		{
			name: schemas.Collections.WebauthnCredential,
			hash: "id",
			attr: []types.AttributeDefinition{
				{AttributeName: aws.String("id"), AttributeType: types.ScalarAttributeTypeS},
				{AttributeName: aws.String("credential_id"), AttributeType: types.ScalarAttributeTypeS},
				{AttributeName: aws.String("user_id"), AttributeType: types.ScalarAttributeTypeS},
			},
			gsi: []types.GlobalSecondaryIndex{
				gsi("credential_id", "credential_id"),
				gsi("user_id", "user_id"),
			},
		},
		{
			name: schemas.Collections.Organization,
			hash: "id",
			attr: []types.AttributeDefinition{
				{AttributeName: aws.String("id"), AttributeType: types.ScalarAttributeTypeS},
				{AttributeName: aws.String("name"), AttributeType: types.ScalarAttributeTypeS},
			},
			gsi: []types.GlobalSecondaryIndex{gsi("name", "name")},
		},
		{
			name: schemas.Collections.OrgMembership,
			hash: "id",
			attr: []types.AttributeDefinition{
				{AttributeName: aws.String("id"), AttributeType: types.ScalarAttributeTypeS},
				{AttributeName: aws.String("org_id"), AttributeType: types.ScalarAttributeTypeS},
				{AttributeName: aws.String("user_id"), AttributeType: types.ScalarAttributeTypeS},
			},
			gsi: []types.GlobalSecondaryIndex{
				gsi("org_id", "org_id"),
				gsi("user_id", "user_id"),
			},
		},
		{
			name: schemas.Collections.FederatedIdentity,
			hash: "id",
			attr: []types.AttributeDefinition{
				{AttributeName: aws.String("id"), AttributeType: types.ScalarAttributeTypeS},
				{AttributeName: aws.String("org_id"), AttributeType: types.ScalarAttributeTypeS},
				{AttributeName: aws.String("user_id"), AttributeType: types.ScalarAttributeTypeS},
			},
			gsi: []types.GlobalSecondaryIndex{
				gsi("org_id", "org_id"),
				gsi("user_id", "user_id"),
			},
		},
		{
			name: schemas.Collections.ScimEndpoint,
			hash: "id",
			attr: []types.AttributeDefinition{
				{AttributeName: aws.String("id"), AttributeType: types.ScalarAttributeTypeS},
				{AttributeName: aws.String("org_id"), AttributeType: types.ScalarAttributeTypeS},
			},
			gsi: []types.GlobalSecondaryIndex{gsi("org_id", "org_id")},
		},
		{
			// OrgDomain: partition key "id" holds the normalized domain, so a
			// conditional PutItem (attribute_not_exists) is a race-free unique insert.
			name: schemas.Collections.OrgDomain,
			hash: "id",
			attr: []types.AttributeDefinition{
				{AttributeName: aws.String("id"), AttributeType: types.ScalarAttributeTypeS},
				{AttributeName: aws.String("org_id"), AttributeType: types.ScalarAttributeTypeS},
			},
			gsi: []types.GlobalSecondaryIndex{gsi("org_id", "org_id")},
		},
		{
			name: schemas.Collections.SAMLServiceProvider,
			hash: "id",
			attr: []types.AttributeDefinition{
				{AttributeName: aws.String("id"), AttributeType: types.ScalarAttributeTypeS},
				{AttributeName: aws.String("org_id"), AttributeType: types.ScalarAttributeTypeS},
			},
			gsi: []types.GlobalSecondaryIndex{gsi("org_id", "org_id")},
		},
		{
			name: schemas.Collections.SAMLIDPKey,
			hash: "id",
			attr: []types.AttributeDefinition{
				{AttributeName: aws.String("id"), AttributeType: types.ScalarAttributeTypeS},
				{AttributeName: aws.String("org_id"), AttributeType: types.ScalarAttributeTypeS},
			},
			gsi: []types.GlobalSecondaryIndex{gsi("org_id", "org_id")},
		},
	}

	for _, t := range tables {
		if err := createTable(ctx, p.client, t.name, t.hash, t.attr, t.gsi); err != nil {
			return fmt.Errorf("create table %s: %w", t.name, err)
		}
	}
	return nil
}
