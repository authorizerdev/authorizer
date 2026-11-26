package dynamodb

import (
	"context"
	"errors"
	"net"
	"testing"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/credentials"
	"github.com/aws/aws-sdk-go-v2/feature/dynamodb/expression"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb/types"
	"github.com/aws/smithy-go"
	"github.com/google/uuid"
)

// TestEnsureGSIOnExistingTable covers the upgrade path for deployments whose
// verification-request table predates the email GSI: createTable is a no-op
// for an existing table, so ensureGSI must add the index, and until it is
// usable a query against it must fail with the ValidationException that
// verificationRequestsByEmail falls back to a scan on. Needs DynamoDB Local on
// :8000 (make test-dynamodb); skipped otherwise.
func TestEnsureGSIOnExistingTable(t *testing.T) {
	conn, err := net.DialTimeout("tcp", "127.0.0.1:8000", time.Second)
	if err != nil {
		t.Skipf("DynamoDB Local not reachable on :8000: %v", err)
	}
	_ = conn.Close()

	ctx := context.Background()
	client := dynamodb.New(dynamodb.Options{
		Region:       "us-east-1",
		BaseEndpoint: aws.String("http://127.0.0.1:8000"),
		Credentials:  credentials.NewStaticCredentialsProvider("key", "key", ""),
	})
	table := "gsi_upgrade_test_" + uuid.New().String()
	idAttr := types.AttributeDefinition{AttributeName: aws.String("id"), AttributeType: types.ScalarAttributeTypeS}
	if err := createTable(ctx, client, table, "id", []types.AttributeDefinition{idAttr}, nil); err != nil {
		t.Fatalf("create table: %v", err)
	}
	t.Cleanup(func() {
		_, _ = client.DeleteTable(context.Background(), &dynamodb.DeleteTableInput{TableName: aws.String(table)})
	})

	// Before the index exists, querying it must be a ValidationException.
	kc := expression.Key("email").Equal(expression.Value("a@b.c"))
	expr, err := expression.NewBuilder().WithKeyCondition(kc).Build()
	if err != nil {
		t.Fatal(err)
	}
	_, err = client.Query(ctx, &dynamodb.QueryInput{
		TableName:                 aws.String(table),
		IndexName:                 aws.String(verificationRequestEmailIndex),
		KeyConditionExpression:    expr.KeyCondition(),
		ExpressionAttributeNames:  expr.Names(),
		ExpressionAttributeValues: expr.Values(),
	})
	var apiErr smithy.APIError
	if !errors.As(err, &apiErr) || apiErr.ErrorCode() != "ValidationException" {
		t.Fatalf("query on a missing index must be a ValidationException, got %v", err)
	}

	emailAttr := types.AttributeDefinition{AttributeName: aws.String("email"), AttributeType: types.ScalarAttributeTypeS}
	if err := ensureGSI(ctx, client, table, emailAttr, gsi(verificationRequestEmailIndex, "email")); err != nil {
		t.Fatalf("ensureGSI: %v", err)
	}
	desc, err := client.DescribeTable(ctx, &dynamodb.DescribeTableInput{TableName: aws.String(table)})
	if err != nil {
		t.Fatal(err)
	}
	found := false
	for _, idx := range desc.Table.GlobalSecondaryIndexes {
		if aws.ToString(idx.IndexName) == verificationRequestEmailIndex {
			found = true
		}
	}
	if !found {
		t.Fatalf("ensureGSI did not add the %q index", verificationRequestEmailIndex)
	}
	// Idempotent: a second call (every restart) must be a no-op, not an error.
	if err := ensureGSI(ctx, client, table, emailAttr, gsi(verificationRequestEmailIndex, "email")); err != nil {
		t.Fatalf("second ensureGSI must be a no-op: %v", err)
	}
}
