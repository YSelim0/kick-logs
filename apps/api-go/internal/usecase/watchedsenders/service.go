// Package watchedsenders manages the admin-editable list of Kick usernames
// that trigger a watched-sender email alert (see internal/usecase/watchlist
// and internal/infra/notify for the processor-side consumer).
package watchedsenders

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strings"

	"github.com/YSelim0/kick-logs/apps/api-go/internal/domain"
	"github.com/YSelim0/kick-logs/apps/api-go/internal/ports"
)

var (
	ErrValidation     = errors.New("validation failed")
	ErrAlreadyWatched = errors.New("username already watched")
	ErrSenderNotFound = errors.New("watched sender not found")
)

const maxUsernameLength = 60

type Service struct {
	senders ports.WatchedSenderRepository
}

func NewService(senders ports.WatchedSenderRepository) *Service {
	return &Service{senders: senders}
}

func (service *Service) List(ctx context.Context) ([]domain.WatchedSender, error) {
	return service.senders.List(ctx)
}

func (service *Service) Add(ctx context.Context, username string) (domain.WatchedSender, error) {
	username = strings.TrimSpace(username)
	if username == "" || len(username) > maxUsernameLength {
		return domain.WatchedSender{}, fmt.Errorf("%w: invalid username", ErrValidation)
	}

	existing, err := service.senders.List(ctx)
	if err != nil {
		return domain.WatchedSender{}, err
	}
	for _, sender := range existing {
		if strings.EqualFold(sender.Username, username) {
			return domain.WatchedSender{}, ErrAlreadyWatched
		}
	}

	return service.senders.Create(ctx, username)
}

func (service *Service) Remove(ctx context.Context, id int64) error {
	err := service.senders.Delete(ctx, id)
	if errors.Is(err, sql.ErrNoRows) {
		return ErrSenderNotFound
	}
	return err
}
