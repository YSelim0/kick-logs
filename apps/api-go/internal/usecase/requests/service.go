package requests

import (
	"context"
	"crypto/rand"
	"database/sql"
	"encoding/hex"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/YSelim0/kick-logs/apps/api-go/internal/domain"
	"github.com/YSelim0/kick-logs/apps/api-go/internal/ports"
)

var (
	ErrValidation = errors.New("validation failed")
	ErrNotFound   = errors.New("request not found")
)

type Service struct {
	repository ports.UserRequestRepository
}

func NewService(repository ports.UserRequestRepository) *Service {
	return &Service{repository: repository}
}

type CreateInput struct {
	Type               string
	Title              string
	Message            string
	ChannelSlug        string
	ChannelDisplayName string
	Contact            string
	IPHash             string
	UserAgentHash      string
}

func (service *Service) Create(ctx context.Context, input CreateInput) (domain.UserRequest, error) {
	if service.repository == nil {
		return domain.UserRequest{}, errors.New("user request repository is not configured")
	}

	requestType, err := parseRequestType(input.Type)
	if err != nil {
		return domain.UserRequest{}, err
	}

	title, err := requiredText(input.Title, 3, 120, "title")
	if err != nil {
		return domain.UserRequest{}, err
	}
	message, err := requiredText(input.Message, 5, 2000, "message")
	if err != nil {
		return domain.UserRequest{}, err
	}
	contact, err := optionalText(input.Contact, 120, "contact")
	if err != nil {
		return domain.UserRequest{}, err
	}
	displayName, err := optionalText(input.ChannelDisplayName, 120, "channel display name")
	if err != nil {
		return domain.UserRequest{}, err
	}

	channelSlug := ""
	if requestType == domain.UserRequestTypeChannelRequest {
		channelSlug, err = normalizeChannelSlug(input.ChannelSlug)
		if err != nil {
			return domain.UserRequest{}, err
		}
	} else if strings.TrimSpace(input.ChannelSlug) != "" {
		channelSlug, err = normalizeChannelSlug(input.ChannelSlug)
		if err != nil {
			return domain.UserRequest{}, err
		}
	}

	requestID, err := newID("req")
	if err != nil {
		return domain.UserRequest{}, err
	}

	request := domain.UserRequest{
		ID:                 requestID,
		Type:               requestType,
		Title:              title,
		Message:            message,
		ChannelSlug:        channelSlug,
		ChannelDisplayName: displayName,
		Contact:            contact,
		IPHash:             strings.TrimSpace(input.IPHash),
		UserAgentHash:      strings.TrimSpace(input.UserAgentHash),
		CreatedAt:          time.Now().UTC(),
	}
	if err := service.repository.Create(ctx, request); err != nil {
		return domain.UserRequest{}, err
	}
	return request, nil
}

func (service *Service) List(
	ctx context.Context,
	filter domain.UserRequestListFilter,
) ([]domain.UserRequestState, error) {
	if service.repository == nil {
		return nil, errors.New("user request repository is not configured")
	}
	normalized, err := normalizeListFilter(filter)
	if err != nil {
		return nil, err
	}
	return service.repository.List(ctx, normalized)
}

func (service *Service) Detail(ctx context.Context, requestID string) (domain.UserRequestDetail, error) {
	if service.repository == nil {
		return domain.UserRequestDetail{}, errors.New("user request repository is not configured")
	}
	requestID = strings.TrimSpace(requestID)
	if requestID == "" || len(requestID) > 80 {
		return domain.UserRequestDetail{}, fmt.Errorf("%w: invalid request id", ErrValidation)
	}

	state, err := service.repository.Get(ctx, requestID)
	if errors.Is(err, sql.ErrNoRows) {
		return domain.UserRequestDetail{}, ErrNotFound
	}
	if err != nil {
		return domain.UserRequestDetail{}, err
	}
	events, err := service.repository.ListEvents(ctx, requestID)
	if err != nil {
		return domain.UserRequestDetail{}, err
	}
	return domain.UserRequestDetail{State: state, Events: events}, nil
}

func (service *Service) ChangeStatus(
	ctx context.Context,
	requestID string,
	status string,
	adminID int64,
) (domain.UserRequestDetail, error) {
	parsedStatus, err := parseStatus(status)
	if err != nil {
		return domain.UserRequestDetail{}, err
	}
	return service.appendAdminEvent(ctx, requestID, domain.UserRequestEvent{
		EventType: domain.UserRequestEventStatusChanged,
		Status:    parsedStatus,
		AdminID:   adminID,
	})
}

func (service *Service) AddNote(
	ctx context.Context,
	requestID string,
	note string,
	adminID int64,
) (domain.UserRequestDetail, error) {
	trimmedNote, err := requiredText(note, 2, 1000, "note")
	if err != nil {
		return domain.UserRequestDetail{}, err
	}
	return service.appendAdminEvent(ctx, requestID, domain.UserRequestEvent{
		EventType: domain.UserRequestEventNoteAdded,
		Note:      trimmedNote,
		AdminID:   adminID,
	})
}

func (service *Service) Archive(
	ctx context.Context,
	requestID string,
	adminID int64,
) (domain.UserRequestDetail, error) {
	return service.appendAdminEvent(ctx, requestID, domain.UserRequestEvent{
		EventType: domain.UserRequestEventArchived,
		AdminID:   adminID,
	})
}

func (service *Service) appendAdminEvent(
	ctx context.Context,
	requestID string,
	event domain.UserRequestEvent,
) (domain.UserRequestDetail, error) {
	if service.repository == nil {
		return domain.UserRequestDetail{}, errors.New("user request repository is not configured")
	}
	requestID = strings.TrimSpace(requestID)
	if requestID == "" || len(requestID) > 80 {
		return domain.UserRequestDetail{}, fmt.Errorf("%w: invalid request id", ErrValidation)
	}

	if _, err := service.repository.Get(ctx, requestID); errors.Is(err, sql.ErrNoRows) {
		return domain.UserRequestDetail{}, ErrNotFound
	} else if err != nil {
		return domain.UserRequestDetail{}, err
	}

	eventID, err := newID("evt")
	if err != nil {
		return domain.UserRequestDetail{}, err
	}
	event.ID = eventID
	event.RequestID = requestID
	event.CreatedAt = time.Now().UTC()
	if err := service.repository.AppendEvent(ctx, event); err != nil {
		return domain.UserRequestDetail{}, err
	}
	return service.Detail(ctx, requestID)
}

func normalizeListFilter(filter domain.UserRequestListFilter) (domain.UserRequestListFilter, error) {
	var err error
	if filter.Type != "" {
		filter.Type, err = parseRequestType(string(filter.Type))
		if err != nil {
			return domain.UserRequestListFilter{}, err
		}
	}
	if filter.Status != "" {
		filter.Status, err = parseStatus(string(filter.Status))
		if err != nil {
			return domain.UserRequestListFilter{}, err
		}
	}
	filter.Query, err = optionalText(filter.Query, 200, "query")
	if err != nil {
		return domain.UserRequestListFilter{}, err
	}
	if !filter.Start.IsZero() && !filter.End.IsZero() && filter.Start.After(filter.End) {
		return domain.UserRequestListFilter{}, fmt.Errorf("%w: start must be before end", ErrValidation)
	}
	if filter.Limit == 0 {
		filter.Limit = 50
	}
	if filter.Limit > 200 {
		filter.Limit = 200
	}
	return filter, nil
}

func parseRequestType(value string) (domain.UserRequestType, error) {
	switch domain.UserRequestType(strings.TrimSpace(value)) {
	case domain.UserRequestTypeChannelRequest:
		return domain.UserRequestTypeChannelRequest, nil
	case domain.UserRequestTypeFeedback:
		return domain.UserRequestTypeFeedback, nil
	default:
		return "", fmt.Errorf("%w: unsupported request type", ErrValidation)
	}
}

func parseStatus(value string) (domain.UserRequestStatus, error) {
	switch domain.UserRequestStatus(strings.TrimSpace(value)) {
	case domain.UserRequestStatusNew:
		return domain.UserRequestStatusNew, nil
	case domain.UserRequestStatusReviewing:
		return domain.UserRequestStatusReviewing, nil
	case domain.UserRequestStatusApproved:
		return domain.UserRequestStatusApproved, nil
	case domain.UserRequestStatusRejected:
		return domain.UserRequestStatusRejected, nil
	case domain.UserRequestStatusDone:
		return domain.UserRequestStatusDone, nil
	case domain.UserRequestStatusDuplicate:
		return domain.UserRequestStatusDuplicate, nil
	default:
		return "", fmt.Errorf("%w: unsupported request status", ErrValidation)
	}
}

func requiredText(value string, minLength int, maxLength int, fieldName string) (string, error) {
	trimmed := strings.TrimSpace(value)
	if len(trimmed) < minLength || len(trimmed) > maxLength {
		return "", fmt.Errorf("%w: invalid %s", ErrValidation, fieldName)
	}
	return trimmed, nil
}

func optionalText(value string, maxLength int, fieldName string) (string, error) {
	trimmed := strings.TrimSpace(value)
	if len(trimmed) > maxLength {
		return "", fmt.Errorf("%w: invalid %s", ErrValidation, fieldName)
	}
	return trimmed, nil
}

func normalizeChannelSlug(value string) (string, error) {
	slug := strings.TrimSpace(strings.ToLower(value))
	slug = strings.TrimPrefix(slug, "@")
	slug = strings.TrimPrefix(slug, "https://")
	slug = strings.TrimPrefix(slug, "http://")
	slug = strings.TrimPrefix(slug, "www.")
	slug = strings.TrimPrefix(slug, "kick.com/")
	if index := strings.IndexAny(slug, "?#"); index >= 0 {
		slug = slug[:index]
	}
	slug = strings.Trim(slug, "/")
	if index := strings.IndexByte(slug, '/'); index >= 0 {
		slug = slug[:index]
	}

	if len(slug) < 2 || len(slug) > 80 {
		return "", fmt.Errorf("%w: invalid channel slug", ErrValidation)
	}
	for _, r := range slug {
		if (r >= 'a' && r <= 'z') || (r >= '0' && r <= '9') || r == '-' || r == '_' {
			continue
		}
		return "", fmt.Errorf("%w: invalid channel slug", ErrValidation)
	}
	return slug, nil
}

func newID(prefix string) (string, error) {
	var raw [16]byte
	if _, err := rand.Read(raw[:]); err != nil {
		return "", fmt.Errorf("generate id: %w", err)
	}
	return prefix + "_" + hex.EncodeToString(raw[:]), nil
}
