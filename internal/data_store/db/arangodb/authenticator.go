package arangodb

import (
	"context"
	"fmt"
	"time"

	arangoDriver "github.com/arangodb/go-driver"
	"github.com/google/uuid"

	"github.com/authorizerdev/authorizer/internal/data_store/schemas"
)

func (p *provider) AddAuthenticator(ctx context.Context, authenticators *schemas.Authenticator) (*schemas.Authenticator, error) {
	exists, _ := p.GetAuthenticatorDetailsByUserId(ctx, authenticators.UserID, authenticators.Method)
	if exists != nil {
		return authenticators, nil
	}
	if authenticators.ID == "" {
		authenticators.ID = uuid.New().String()
	}

	authenticators.Key = authenticators.ID
	authenticators.CreatedAt = time.Now().Unix()
	authenticators.UpdatedAt = time.Now().Unix()

	authenticatorsCollection, _ := p.db.Collection(ctx, schemas.Collections.Authenticators)
	meta, err := authenticatorsCollection.CreateDocument(arangoDriver.WithOverwrite(ctx), authenticators)
	if err != nil {
		return nil, err
	}
	authenticators.Key = meta.Key
	authenticators.ID = meta.ID.String()

	return authenticators, nil
}

func (p *provider) UpdateAuthenticator(ctx context.Context, authenticators *schemas.Authenticator) (*schemas.Authenticator, error) {
	authenticators.UpdatedAt = time.Now().Unix()

	collection, _ := p.db.Collection(ctx, schemas.Collections.Authenticators)
	meta, err := collection.UpdateDocument(ctx, authenticators.Key, authenticators)
	if err != nil {
		return nil, err
	}

	authenticators.Key = meta.Key
	authenticators.ID = meta.ID.String()
	return authenticators, nil
}

func (p *provider) GetAuthenticatorDetailsByUserId(ctx context.Context, userId string, authenticatorType string) (*schemas.Authenticator, error) {
	var authenticators *schemas.Authenticator
	query := fmt.Sprintf("FOR d in %s FILTER d.user_id == @user_id AND d.method == @method LIMIT 1 RETURN d", schemas.Collections.Authenticators)
	bindVars := map[string]interface{}{
		"user_id": userId,
		"method":  authenticatorType,
	}
	cursor, err := p.db.Query(ctx, query, bindVars)
	if err != nil {
		return nil, err
	}
	defer cursor.Close()
	for {
		if !cursor.HasMore() {
			if authenticators == nil {
				return authenticators, fmt.Errorf("authenticator not found")
			}
			break
		}
		_, err := cursor.ReadDocument(ctx, &authenticators)
		if err != nil {
			return nil, err
		}
	}
	return authenticators, nil
}
