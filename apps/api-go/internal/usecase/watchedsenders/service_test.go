package watchedsenders

import (
	"context"
	"database/sql"
	"errors"
	"testing"
	"time"

	"github.com/YSelim0/kick-logs/apps/api-go/internal/domain"
)

type fakeRepository struct {
	nextID  int64
	senders []domain.WatchedSender
}

func (f *fakeRepository) Create(_ context.Context, username string) (domain.WatchedSender, error) {
	f.nextID++
	sender := domain.WatchedSender{ID: f.nextID, Username: username, CreatedAt: time.Now().UTC()}
	f.senders = append(f.senders, sender)
	return sender, nil
}

func (f *fakeRepository) Delete(_ context.Context, id int64) error {
	for i, sender := range f.senders {
		if sender.ID == id {
			f.senders = append(f.senders[:i], f.senders[i+1:]...)
			return nil
		}
	}
	return sql.ErrNoRows
}

func (f *fakeRepository) List(_ context.Context) ([]domain.WatchedSender, error) {
	out := make([]domain.WatchedSender, len(f.senders))
	copy(out, f.senders)
	return out, nil
}

func (f *fakeRepository) ListUsernames(ctx context.Context) ([]string, error) {
	senders, err := f.List(ctx)
	if err != nil {
		return nil, err
	}
	usernames := make([]string, 0, len(senders))
	for _, sender := range senders {
		usernames = append(usernames, sender.Username)
	}
	return usernames, nil
}

type fakeSettingsRepository struct {
	settings domain.NotificationSettings
}

func (f *fakeSettingsRepository) GetNotificationSettings(_ context.Context) (domain.NotificationSettings, error) {
	return f.settings, nil
}

func (f *fakeSettingsRepository) UpdateNotificationSettings(_ context.Context, settings domain.NotificationSettings) (domain.NotificationSettings, error) {
	f.settings = settings
	return f.settings, nil
}

func TestService_Add_RejectsEmptyAndOverlongUsername(t *testing.T) {
	service := NewService(&fakeRepository{}, &fakeSettingsRepository{})

	if _, err := service.Add(context.Background(), "   "); !errors.Is(err, ErrValidation) {
		t.Fatalf("expected ErrValidation for empty username, got %v", err)
	}

	overlong := make([]byte, 61)
	for i := range overlong {
		overlong[i] = 'a'
	}
	if _, err := service.Add(context.Background(), string(overlong)); !errors.Is(err, ErrValidation) {
		t.Fatalf("expected ErrValidation for overlong username, got %v", err)
	}
}

func TestService_Add_RejectsDuplicateCaseInsensitive(t *testing.T) {
	repo := &fakeRepository{}
	service := NewService(repo, &fakeSettingsRepository{})

	if _, err := service.Add(context.Background(), "Nuriben"); err != nil {
		t.Fatalf("unexpected error on first add: %v", err)
	}
	if _, err := service.Add(context.Background(), "nuriben"); !errors.Is(err, ErrAlreadyWatched) {
		t.Fatalf("expected ErrAlreadyWatched, got %v", err)
	}
}

func TestService_Remove_NotFound(t *testing.T) {
	service := NewService(&fakeRepository{}, &fakeSettingsRepository{})

	if err := service.Remove(context.Background(), 999); !errors.Is(err, ErrSenderNotFound) {
		t.Fatalf("expected ErrSenderNotFound, got %v", err)
	}
}

func TestService_List_ReturnsAddedSenders(t *testing.T) {
	service := NewService(&fakeRepository{}, &fakeSettingsRepository{})

	if _, err := service.Add(context.Background(), "nuriben"); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	senders, err := service.List(context.Background())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(senders) != 1 || senders[0].Username != "nuriben" {
		t.Fatalf("unexpected senders: %+v", senders)
	}
}

func TestService_UpdateCooldownSeconds_RejectsOutOfRange(t *testing.T) {
	service := NewService(&fakeRepository{}, &fakeSettingsRepository{settings: domain.NotificationSettings{CooldownSeconds: 600}})

	if _, err := service.UpdateCooldownSeconds(context.Background(), 10); !errors.Is(err, ErrValidation) {
		t.Fatalf("expected ErrValidation for too-low cooldown, got %v", err)
	}
	if _, err := service.UpdateCooldownSeconds(context.Background(), 999999); !errors.Is(err, ErrValidation) {
		t.Fatalf("expected ErrValidation for too-high cooldown, got %v", err)
	}
}

func TestService_UpdateCooldownSeconds_PersistsValue(t *testing.T) {
	settingsRepo := &fakeSettingsRepository{settings: domain.NotificationSettings{CooldownSeconds: 600}}
	service := NewService(&fakeRepository{}, settingsRepo)

	updated, err := service.UpdateCooldownSeconds(context.Background(), 120)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if updated != 120 {
		t.Fatalf("updated cooldown = %d, want 120", updated)
	}

	got, err := service.GetCooldownSeconds(context.Background())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got != 120 {
		t.Fatalf("GetCooldownSeconds() = %d, want 120", got)
	}
}
