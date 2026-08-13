package dynamodb

import (
	"context"
	"errors"
	"fmt"
	"strconv"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/feature/dynamodb/expression"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb/types"
	"github.com/google/uuid"

	"github.com/authorizerdev/authorizer/internal/storage/schemas"
)

func (p *provider) AddAuthenticator(ctx context.Context, authenticators *schemas.Authenticator) (*schemas.Authenticator, error) {
	exists, _ := p.GetAuthenticatorDetailsByUserId(ctx, authenticators.UserID, authenticators.Method)
	if exists != nil {
		return authenticators, nil
	}
	if authenticators.ID == "" {
		authenticators.ID = uuid.New().String()
	}
	authenticators.CreatedAt = time.Now().Unix()
	authenticators.UpdatedAt = time.Now().Unix()
	if err := p.putItem(ctx, schemas.Collections.Authenticators, authenticators); err != nil {
		return nil, err
	}
	return authenticators, nil
}

func (p *provider) UpdateAuthenticator(ctx context.Context, authenticators *schemas.Authenticator) (*schemas.Authenticator, error) {
	if authenticators.ID != "" {
		authenticators.UpdatedAt = time.Now().Unix()
		if err := p.updateByHashKey(ctx, schemas.Collections.Authenticators, "id", authenticators.ID, authenticators); err != nil {
			return nil, err
		}
	}
	return authenticators, nil
}

// UpdateAuthenticatorSecretAndVerifiedAt writes only secret and verified_at,
// leaving recovery_codes to ConsumeAuthenticatorRecoveryCode.
//
// attribute_exists(id) is load-bearing, not decoration: DynamoDB's UpdateItem is
// an upsert, so without the condition an admin MFA reset that deletes the item
// mid-flight would be followed by this write RESURRECTING it — a ghost item
// carrying a secret and no recovery codes, which would then look like an
// enrolled factor. A missing item is the documented no-op.
func (p *provider) UpdateAuthenticatorSecretAndVerifiedAt(ctx context.Context, id, secret string, verifiedAt int64) error {
	_, err := p.client.UpdateItem(ctx, &dynamodb.UpdateItemInput{
		TableName: aws.String(schemas.Collections.Authenticators),
		Key: map[string]types.AttributeValue{
			"id": &types.AttributeValueMemberS{Value: id},
		},
		UpdateExpression:    aws.String("SET #s = :secret, #va = :verified_at, #ua = :updated_at"),
		ConditionExpression: aws.String("attribute_exists(id)"),
		ExpressionAttributeNames: map[string]string{
			"#s":  "secret",
			"#va": "verified_at",
			"#ua": "updated_at",
		},
		ExpressionAttributeValues: map[string]types.AttributeValue{
			":secret":      &types.AttributeValueMemberS{Value: secret},
			":verified_at": &types.AttributeValueMemberN{Value: strconv.FormatInt(verifiedAt, 10)},
			":updated_at":  &types.AttributeValueMemberN{Value: strconv.FormatInt(time.Now().Unix(), 10)},
		},
	})
	if err != nil {
		var ccf *types.ConditionalCheckFailedException
		if errors.As(err, &ccf) {
			return nil
		}
		return err
	}
	return nil
}

// ConsumeAuthenticatorRecoveryCode swaps the recovery-code blob only while the
// item still holds oldCodes. DynamoDB evaluates the ConditionExpression as part
// of the same UpdateItem, so exactly one caller writes under concurrent
// redemption and the rest get ConditionalCheckFailed — a lost race, returned as
// (false, nil) rather than an error. An item whose recovery_codes attribute is
// absent or different fails the same way, which is the intended answer.
func (p *provider) ConsumeAuthenticatorRecoveryCode(ctx context.Context, id, oldCodes, newCodes string) (bool, error) {
	_, err := p.client.UpdateItem(ctx, &dynamodb.UpdateItemInput{
		TableName: aws.String(schemas.Collections.Authenticators),
		Key: map[string]types.AttributeValue{
			"id": &types.AttributeValueMemberS{Value: id},
		},
		UpdateExpression:    aws.String("SET #rc = :new_codes, #ua = :updated_at"),
		ConditionExpression: aws.String("#rc = :old_codes"),
		ExpressionAttributeNames: map[string]string{
			"#rc": "recovery_codes",
			"#ua": "updated_at",
		},
		ExpressionAttributeValues: map[string]types.AttributeValue{
			":new_codes":  &types.AttributeValueMemberS{Value: newCodes},
			":old_codes":  &types.AttributeValueMemberS{Value: oldCodes},
			":updated_at": &types.AttributeValueMemberN{Value: strconv.FormatInt(time.Now().Unix(), 10)},
		},
	})
	if err != nil {
		var ccf *types.ConditionalCheckFailedException
		if errors.As(err, &ccf) {
			return false, nil
		}
		return false, err
	}
	return true, nil
}

func (p *provider) GetAuthenticatorDetailsByUserId(ctx context.Context, userId string, authenticatorType string) (*schemas.Authenticator, error) {
	f := expression.Name("user_id").Equal(expression.Value(userId)).And(expression.Name("method").Equal(expression.Value(authenticatorType)))
	items, err := p.scanFilteredAll(ctx, schemas.Collections.Authenticators, nil, &f)
	if err != nil {
		return nil, err
	}
	if len(items) == 0 {
		// Absent MUST be an error, matching every other backend. totp.Validate
		// and ValidateRecoveryCode branch on err alone before dereferencing.
		return nil, fmt.Errorf("authenticator not found: %w", ErrNotFound)
	}
	var a schemas.Authenticator
	if err := unmarshalItem(items[0], &a); err != nil {
		return nil, err
	}
	return &a, nil
}

// DeleteAuthenticatorsByUserID removes every authenticator row for a user.
// Used by admin MFA reset.
func (p *provider) DeleteAuthenticatorsByUserID(ctx context.Context, userID string) error {
	f := expression.Name("user_id").Equal(expression.Value(userID))
	items, err := p.scanFilteredAll(ctx, schemas.Collections.Authenticators, nil, &f)
	if err != nil {
		return err
	}
	for _, item := range items {
		var a schemas.Authenticator
		if err := unmarshalItem(item, &a); err != nil {
			return err
		}
		if err := p.deleteItemByHash(ctx, schemas.Collections.Authenticators, "id", a.ID); err != nil {
			return err
		}
	}
	return nil
}
