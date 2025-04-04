package sql

import (
	"context"
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"

	"github.com/authorizerdev/authorizer/internal/graph/model"
	"github.com/authorizerdev/authorizer/internal/storage/schemas"
)

// AddWebhookLog to add webhook log
func (p *provider) AddWebhookLog(ctx context.Context, webhookLog *schemas.WebhookLog) (*schemas.WebhookLog, error) {
	if webhookLog.ID == "" {
		webhookLog.ID = uuid.New().String()
	}

	webhookLog.Key = webhookLog.ID
	webhookLog.CreatedAt = time.Now().Unix()
	webhookLog.UpdatedAt = time.Now().Unix()
	res := p.db.Clauses(
		clause.OnConflict{
			DoNothing: true,
		}).Create(&webhookLog)
	if res.Error != nil {
		return nil, res.Error
	}

	return webhookLog, nil
}

// ListWebhookLogs to list webhook logs
func (p *provider) ListWebhookLogs(ctx context.Context, pagination *model.Pagination, webhookID string) ([]*schemas.WebhookLog, *model.Pagination, error) {
	var webhookLogs []*schemas.WebhookLog
	var result *gorm.DB
	var totalRes *gorm.DB
	var total int64

	if webhookID != "" {
		result = p.db.Where("webhook_id = ?", webhookID).Limit(int(pagination.Limit)).Offset(int(pagination.Offset)).Order("created_at DESC").Find(&webhookLogs)
		totalRes = p.db.Where("webhook_id = ?", webhookID).Model(&schemas.WebhookLog{}).Count(&total)
	} else {
		result = p.db.Limit(int(pagination.Limit)).Offset(int(pagination.Offset)).Order("created_at DESC").Find(&webhookLogs)
		totalRes = p.db.Model(&schemas.WebhookLog{}).Count(&total)
	}

	if result.Error != nil {
		return nil, nil, result.Error
	}

	if totalRes.Error != nil {
		return nil, nil, totalRes.Error
	}

	paginationClone := pagination
	paginationClone.Total = total

	return webhookLogs, paginationClone, nil
}
