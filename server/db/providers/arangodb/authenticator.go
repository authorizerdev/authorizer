package arangodb

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"

	arangoDriver "github.com/arangodb/go-driver"

	"github.com/authorizerdev/authorizer/server/db/models"
)

func (p *provider) AddAuthenticator(ctx context.Context, authenticators *models.Authenticators) (*models.Authenticators, error) {
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

	authenticatorsCollection, _ := p.db.Collection(ctx, models.Collections.Authenticators)
	meta, err := authenticatorsCollection.CreateDocument(arangoDriver.WithOverwrite(ctx), authenticators)
	if err != nil {
		return authenticators, err
	}
	authenticators.Key = meta.Key
	authenticators.ID = meta.ID.String()

	return authenticators, nil
}

func (p *provider) UpdateAuthenticator(ctx context.Context, authenticators *models.Authenticators) (*models.Authenticators, error) {
	authenticators.UpdatedAt = time.Now().Unix()

	collection, _ := p.db.Collection(ctx, models.Collections.Authenticators)
	meta, err := collection.UpdateDocument(ctx, authenticators.Key, authenticators)
	if err != nil {
		return authenticators, err
	}

	authenticators.Key = meta.Key
	authenticators.ID = meta.ID.String()
	return authenticators, nil
}

func (p *provider) GetAuthenticatorDetailsByUserId(ctx context.Context, userId string, authenticatorType string) (*models.Authenticators, error) {
	var authenticators *models.Authenticators
	query := fmt.Sprintf("FOR d in %s FILTER d.user_id == @user_id AND d.method == @method LIMIT 1 RETURN d", models.Collections.Authenticators)
	bindVars := map[string]interface{}{
		"user_id": userId,
		"method":  authenticatorType,
	}
	cursor, err := p.db.Query(ctx, query, bindVars)
	if err != nil {
		return authenticators, err
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
			return authenticators, err
		}
	}
	return authenticators, nil
}
