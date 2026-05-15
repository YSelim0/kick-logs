package messages

import (
	"context"
	"errors"

	"github.com/YSelim0/kick-logs/apps/api-go/internal/domain"
	"github.com/YSelim0/kick-logs/apps/api-go/internal/ports"
)

var (
	ErrInvalidCursor = errors.New("invalid cursor")
	ErrInvalidRange  = errors.New("invalid date range")
)

type Service struct {
	repository ports.MessageRepository
}

func NewService(repository ports.MessageRepository) *Service {
	return &Service{repository: repository}
}

func (service *Service) Search(ctx context.Context, filter domain.MessageSearchFilter) (SearchPage, error) {
	if service.repository == nil {
		return SearchPage{}, errors.New("message repository is not configured")
	}
	if !filter.Start.IsZero() && !filter.End.IsZero() && filter.Start.After(filter.End) {
		return SearchPage{}, ErrInvalidRange
	}
	if filter.Limit == 0 {
		filter.Limit = 50
	}

	messages, err := service.repository.Search(ctx, filter)
	if err != nil {
		return SearchPage{}, err
	}

	var nextCursor *domain.MessageCursor
	if uint64(len(messages)) >= filter.Limit && len(messages) > 0 {
		lastMessage := messages[len(messages)-1]
		nextCursor = &domain.MessageCursor{
			MessageCreatedAt: lastMessage.MessageCreatedAt,
			MessageID:        lastMessage.ID,
		}
	}

	return SearchPage{Items: messages, NextCursor: nextCursor}, nil
}

func (service *Service) Export(
	ctx context.Context,
	filter domain.MessageSearchFilter,
	maxRows uint64,
) (MessageExport, error) {
	if maxRows == 0 {
		maxRows = 1
	}
	filter.Cursor = nil
	filter.Limit = maxRows + 1

	page, err := service.Search(ctx, filter)
	if err != nil {
		return MessageExport{}, err
	}

	items := page.Items
	truncated := uint64(len(items)) > maxRows
	if truncated {
		items = items[:maxRows]
	}

	return MessageExport{
		Items:     items,
		Count:     len(items),
		MaxRows:   int(maxRows),
		Truncated: truncated,
	}, nil
}

type SearchPage struct {
	Items      []domain.ChatMessage
	NextCursor *domain.MessageCursor
}

type MessageExport struct {
	Items     []domain.ChatMessage
	Count     int
	MaxRows   int
	Truncated bool
}
