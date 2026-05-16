package datamanagement

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/YSelim0/kick-logs/apps/api-go/internal/domain"
	"github.com/YSelim0/kick-logs/apps/api-go/internal/ports"
)

var (
	ErrValidation             = errors.New("validation failed")
	ErrCleanupCannotExecute   = errors.New("cleanup cannot be executed")
	ErrCleanupConfirmation    = errors.New("cleanup confirmation mismatch")
	allowedRetentionDayValues = map[int]bool{30: true, 90: true}
)

type Service struct {
	repository ports.DataManagementRepository
}

func NewService(repository ports.DataManagementRepository) *Service {
	return &Service{repository: repository}
}

func (service *Service) Summary(ctx context.Context) (domain.DataManagementSummary, error) {
	return service.repository.Summary(ctx)
}

func (service *Service) UpdateRetentionSettings(
	ctx context.Context,
	messageRetentionDays *int,
	rawEventRetentionDays *int,
) (domain.RetentionSettings, error) {
	if !validRetentionDays(messageRetentionDays) || !validRetentionDays(rawEventRetentionDays) {
		return domain.RetentionSettings{}, fmt.Errorf("%w: retention days must be one of: 30, 90", ErrValidation)
	}
	return service.repository.UpdateRetentionSettings(ctx, domain.RetentionSettings{
		ID:                    1,
		MessageRetentionDays:  messageRetentionDays,
		RawEventRetentionDays: rawEventRetentionDays,
		UpdatedAt:             time.Now().UTC(),
	})
}

func (service *Service) PreviewCleanup(
	ctx context.Context,
	request domain.DataCleanupRequest,
) (domain.DataCleanupPreview, error) {
	settings, err := service.repository.GetRetentionSettings(ctx)
	if err != nil {
		return domain.DataCleanupPreview{}, err
	}
	criteria, err := buildCleanupCriteria(request, settings)
	if err != nil {
		return domain.DataCleanupPreview{}, err
	}
	affected, err := service.repository.CountCleanup(ctx, criteria)
	if err != nil {
		return domain.DataCleanupPreview{}, err
	}
	return buildPreview(criteria, affected), nil
}

func (service *Service) ConfirmCleanup(
	ctx context.Context,
	request domain.DataCleanupRequest,
	confirmationText string,
) (domain.DataCleanupResult, error) {
	preview, err := service.PreviewCleanup(ctx, request)
	if err != nil {
		return domain.DataCleanupResult{}, err
	}
	if !preview.CanExecute {
		return domain.DataCleanupResult{}, fmt.Errorf("%w: %s", ErrCleanupCannotExecute, preview.Reason)
	}
	if strings.TrimSpace(confirmationText) != preview.ConfirmationText {
		return domain.DataCleanupResult{}, ErrCleanupConfirmation
	}

	deleted, err := service.repository.ExecuteCleanup(ctx, domain.DataCleanupCriteria{
		Target:        preview.Target,
		CutoffAt:      preview.CutoffAt,
		ChannelSlug:   preview.ChannelSlug,
		Sender:        preview.Sender,
		RetentionDays: preview.RetentionDays,
	})
	if err != nil {
		return domain.DataCleanupResult{}, err
	}
	return domain.DataCleanupResult{
		Target:           preview.Target,
		Deleted:          deleted,
		ConfirmationText: preview.ConfirmationText,
		CutoffAt:         preview.CutoffAt,
		ChannelSlug:      preview.ChannelSlug,
		Sender:           preview.Sender,
		RetentionDays:    preview.RetentionDays,
	}, nil
}

func buildCleanupCriteria(
	request domain.DataCleanupRequest,
	settings domain.RetentionSettings,
) (domain.DataCleanupCriteria, error) {
	switch request.Target {
	case domain.DataCleanupTargetOldMessages:
		return oldCriteria(request.Target, settings.MessageRetentionDays)
	case domain.DataCleanupTargetOldRawEvents:
		return oldCriteria(request.Target, settings.RawEventRetentionDays)
	case domain.DataCleanupTargetChannel:
		channelSlug := strings.TrimSpace(request.ChannelSlug)
		if channelSlug == "" {
			return domain.DataCleanupCriteria{}, fmt.Errorf("%w: channel slug is required", ErrValidation)
		}
		return domain.DataCleanupCriteria{Target: request.Target, ChannelSlug: channelSlug}, nil
	case domain.DataCleanupTargetSender:
		sender := strings.TrimSpace(request.Sender)
		if sender == "" {
			return domain.DataCleanupCriteria{}, fmt.Errorf("%w: sender is required", ErrValidation)
		}
		return domain.DataCleanupCriteria{Target: request.Target, Sender: sender}, nil
	default:
		return domain.DataCleanupCriteria{}, fmt.Errorf("%w: unsupported cleanup target", ErrValidation)
	}
}

func oldCriteria(target domain.DataCleanupTarget, retentionDays *int) (domain.DataCleanupCriteria, error) {
	if !validRetentionDays(retentionDays) {
		return domain.DataCleanupCriteria{}, fmt.Errorf("%w: retention days must be one of: 30, 90", ErrValidation)
	}
	criteria := domain.DataCleanupCriteria{Target: target, RetentionDays: retentionDays}
	if retentionDays == nil {
		return criteria, nil
	}
	criteria.CutoffAt = time.Now().UTC().Add(-time.Duration(*retentionDays) * 24 * time.Hour)
	return criteria, nil
}

func buildPreview(criteria domain.DataCleanupCriteria, affected domain.DataCleanupCounts) domain.DataCleanupPreview {
	canExecute := true
	reason := ""
	if (criteria.Target == domain.DataCleanupTargetOldMessages ||
		criteria.Target == domain.DataCleanupTargetOldRawEvents) && criteria.CutoffAt.IsZero() {
		canExecute = false
		reason = "Retention is set to keep forever."
	}
	return domain.DataCleanupPreview{
		Target:           criteria.Target,
		Affected:         affected,
		ConfirmationText: confirmationText(criteria),
		CanExecute:       canExecute,
		CutoffAt:         criteria.CutoffAt,
		ChannelSlug:      criteria.ChannelSlug,
		Sender:           criteria.Sender,
		RetentionDays:    criteria.RetentionDays,
		Reason:           reason,
	}
}

func confirmationText(criteria domain.DataCleanupCriteria) string {
	switch criteria.Target {
	case domain.DataCleanupTargetOldMessages:
		return "DELETE OLD MESSAGES"
	case domain.DataCleanupTargetOldRawEvents:
		return "DELETE OLD RAW EVENTS"
	case domain.DataCleanupTargetChannel:
		return "DELETE CHANNEL " + criteria.ChannelSlug
	default:
		return "DELETE SENDER " + criteria.Sender
	}
}

func validRetentionDays(value *int) bool {
	return value == nil || allowedRetentionDayValues[*value]
}
