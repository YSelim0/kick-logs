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

const (
	maxUsernameLength = 60
	// minCooldownSeconds keeps an active watched chatter from flooding the
	// recipient's inbox; maxCooldownSeconds keeps a fat-fingered value from
	// silencing the feature for an unreasonable stretch.
	minCooldownSeconds = 30
	maxCooldownSeconds = 86400
)

type Service struct {
	senders  ports.WatchedSenderRepository
	settings ports.NotificationSettingsRepository
}

func NewService(senders ports.WatchedSenderRepository, settings ports.NotificationSettingsRepository) *Service {
	return &Service{senders: senders, settings: settings}
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

func (service *Service) GetCooldownSeconds(ctx context.Context) (int, error) {
	settings, err := service.settings.GetNotificationSettings(ctx)
	if err != nil {
		return 0, err
	}
	return settings.CooldownSeconds, nil
}

func (service *Service) UpdateCooldownSeconds(ctx context.Context, cooldownSeconds int) (int, error) {
	if cooldownSeconds < minCooldownSeconds || cooldownSeconds > maxCooldownSeconds {
		return 0, fmt.Errorf("%w: cooldown must be between %d and %d seconds", ErrValidation, minCooldownSeconds, maxCooldownSeconds)
	}
	settings, err := service.settings.UpdateNotificationSettings(ctx, domain.NotificationSettings{CooldownSeconds: cooldownSeconds})
	if err != nil {
		return 0, err
	}
	return settings.CooldownSeconds, nil
}
