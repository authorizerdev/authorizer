package service

import (
	"context"
	"fmt"
	"strings"

	"github.com/authorizerdev/authorizer/internal/audit"
	"github.com/authorizerdev/authorizer/internal/constants"
	"github.com/authorizerdev/authorizer/internal/graph/model"
	"github.com/authorizerdev/authorizer/internal/refs"
	"github.com/authorizerdev/authorizer/internal/storage/schemas"
	"github.com/authorizerdev/authorizer/internal/utils"
	"github.com/authorizerdev/authorizer/internal/validators"
)

// AddEmailTemplate creates a new email template for an event and returns a
// status message. Requires super-admin auth. Logic migrated from
// internal/graphql/add_email_template.go.
func (p *provider) AddEmailTemplate(ctx context.Context, meta RequestMetadata, params *model.AddEmailTemplateRequest) (*model.Response, *ResponseSideEffects, error) {
	log := p.Log.With().Str("func", "AddEmailTemplate").Logger()
	if err := p.requireSuperAdmin(ctx, meta); err != nil {
		return nil, nil, err
	}

	if !validators.IsValidEmailTemplateEventName(params.EventName) {
		log.Debug().Str("EventName", params.EventName).Msg("Invalid Event Name")
		return nil, nil, fmt.Errorf("invalid event name %s", params.EventName)
	}

	if strings.TrimSpace(params.Subject) == "" {
		log.Debug().Msg("subject is missing")
		return nil, nil, fmt.Errorf("empty subject not allowed")
	}

	if strings.TrimSpace(params.Template) == "" {
		log.Debug().Msg("template is missing")
		return nil, nil, fmt.Errorf("empty template not allowed")
	}

	var design string
	if params.Design == nil || strings.TrimSpace(refs.StringValue(params.Design)) == "" {
		design = ""
	} else {
		design = refs.StringValue(params.Design)
	}

	emailTemplate, err := p.StorageProvider.AddEmailTemplate(ctx, &schemas.EmailTemplate{
		EventName: params.EventName,
		Template:  params.Template,
		Subject:   params.Subject,
		Design:    design,
	})
	if err != nil {
		log.Debug().Err(err).Msg("Failed to add email template in db")
		return nil, nil, err
	}

	p.AuditProvider.LogEvent(audit.Event{
		Action:   constants.AuditAdminEmailTemplateCreatedEvent,
		Protocol: meta.Protocol, ActorType: constants.AuditActorTypeAdmin,
		ResourceType: constants.AuditResourceTypeEmailTemplate,
		ResourceID:   emailTemplate.ID,
		IPAddress:    meta.IPAddress,
		UserAgent:    meta.UserAgent,
	})

	return &model.Response{
		Message: `Email template added successfully`,
	}, nil, nil
}

// UpdateEmailTemplate updates an existing email template's event, subject,
// body, or design and returns a status message. Requires super-admin auth.
// Logic migrated from internal/graphql/update_email_template.go.
func (p *provider) UpdateEmailTemplate(ctx context.Context, meta RequestMetadata, params *model.UpdateEmailTemplateRequest) (*model.Response, *ResponseSideEffects, error) {
	log := p.Log.With().Str("func", "UpdateEmailTemplate").Logger()
	if err := p.requireSuperAdmin(ctx, meta); err != nil {
		return nil, nil, err
	}

	emailTemplate, err := p.StorageProvider.GetEmailTemplateByID(ctx, params.ID)
	if err != nil {
		log.Debug().Err(err).Msg("failed GetEmailTemplateByID")
		return nil, nil, err
	}

	emailTemplateDetails := &schemas.EmailTemplate{
		ID:        emailTemplate.ID,
		Key:       emailTemplate.ID,
		EventName: emailTemplate.EventName,
		Subject:   emailTemplate.Subject,
		Template:  emailTemplate.Template,
		Design:    emailTemplate.Design,
		CreatedAt: emailTemplate.CreatedAt,
	}

	if params.EventName != nil && emailTemplateDetails.EventName != refs.StringValue(params.EventName) {
		if isValid := validators.IsValidEmailTemplateEventName(refs.StringValue(params.EventName)); !isValid {
			log.Debug().Str("event_name", refs.StringValue(params.EventName)).Msg("invalid event name")
			return nil, nil, fmt.Errorf("invalid event name %s", refs.StringValue(params.EventName))
		}
		emailTemplateDetails.EventName = refs.StringValue(params.EventName)
	}

	if params.Subject != nil && emailTemplateDetails.Subject != refs.StringValue(params.Subject) {
		if strings.TrimSpace(refs.StringValue(params.Subject)) == "" {
			log.Debug().Msg("empty subject not allowed")
			return nil, nil, fmt.Errorf("empty subject not allowed")
		}
		emailTemplateDetails.Subject = refs.StringValue(params.Subject)
	}

	if params.Template != nil && emailTemplateDetails.Template != refs.StringValue(params.Template) {
		if strings.TrimSpace(refs.StringValue(params.Template)) == "" {
			log.Debug().Msg("empty template not allowed")
			return nil, nil, fmt.Errorf("empty template not allowed")
		}
		emailTemplateDetails.Template = refs.StringValue(params.Template)
	}

	if params.Design != nil && emailTemplateDetails.Design != refs.StringValue(params.Design) {
		if strings.TrimSpace(refs.StringValue(params.Design)) == "" {
			log.Debug().Msg("empty design not allowed")
			return nil, nil, fmt.Errorf("empty design not allowed")
		}
		emailTemplateDetails.Design = refs.StringValue(params.Design)
	}

	if _, err := p.StorageProvider.UpdateEmailTemplate(ctx, emailTemplateDetails); err != nil {
		log.Debug().Err(err).Msg("failed UpdateEmailTemplate")
		return nil, nil, err
	}

	p.AuditProvider.LogEvent(audit.Event{
		Action:   constants.AuditAdminEmailTemplateUpdatedEvent,
		Protocol: meta.Protocol, ActorType: constants.AuditActorTypeAdmin,
		ResourceType: constants.AuditResourceTypeEmailTemplate,
		ResourceID:   params.ID,
		IPAddress:    meta.IPAddress,
		UserAgent:    meta.UserAgent,
	})

	return &model.Response{
		Message: `Email template updated successfully.`,
	}, nil, nil
}

// DeleteEmailTemplate deletes an email template by id and returns a status
// message. Requires super-admin auth. Logic migrated from
// internal/graphql/delete_email_template.go.
func (p *provider) DeleteEmailTemplate(ctx context.Context, meta RequestMetadata, params *model.DeleteEmailTemplateRequest) (*model.Response, *ResponseSideEffects, error) {
	log := p.Log.With().Str("func", "DeleteEmailTemplate").Logger()
	if err := p.requireSuperAdmin(ctx, meta); err != nil {
		return nil, nil, err
	}

	if strings.TrimSpace(params.ID) == "" {
		log.Debug().Msg("email template ID required")
		return nil, nil, fmt.Errorf("email template ID required")
	}

	log = log.With().Str("emailTemplateID", params.ID).Logger()

	emailTemplate, err := p.StorageProvider.GetEmailTemplateByID(ctx, params.ID)
	if err != nil {
		log.Debug().Err(err).Msg("Failed to get email template by id")
		return nil, nil, err
	}

	if err := p.StorageProvider.DeleteEmailTemplate(ctx, emailTemplate); err != nil {
		log.Debug().Err(err).Msg("Failed to delete email template")
		return nil, nil, err
	}

	p.AuditProvider.LogEvent(audit.Event{
		Action:   constants.AuditAdminEmailTemplateDeletedEvent,
		Protocol: meta.Protocol, ActorType: constants.AuditActorTypeAdmin,
		ResourceType: constants.AuditResourceTypeEmailTemplate,
		ResourceID:   params.ID,
		IPAddress:    meta.IPAddress,
		UserAgent:    meta.UserAgent,
	})

	return &model.Response{
		Message: "Email templated deleted successfully",
	}, nil, nil
}

// EmailTemplates returns a paginated list of email templates. Requires
// super-admin auth. Logic migrated from internal/graphql/email_templates.go.
func (p *provider) EmailTemplates(ctx context.Context, meta RequestMetadata, params *model.PaginatedRequest) (*model.EmailTemplates, *ResponseSideEffects, error) {
	log := p.Log.With().Str("func", "EmailTemplates").Logger()
	if err := p.requireSuperAdmin(ctx, meta); err != nil {
		return nil, nil, err
	}

	pagination := utils.GetPagination(params)
	emailTemplates, pagination, err := p.StorageProvider.ListEmailTemplate(ctx, pagination)
	if err != nil {
		log.Debug().Err(err).Msg("Failed to get email templates")
		return nil, nil, err
	}
	resItems := make([]*model.EmailTemplate, 0, len(emailTemplates))
	for _, emailTemplate := range emailTemplates {
		resItems = append(resItems, emailTemplate.AsAPIEmailTemplate())
	}

	return &model.EmailTemplates{
		Pagination:     pagination,
		EmailTemplates: resItems,
	}, nil, nil
}
