package requests

import (
	"context"
	"database/sql"
	"errors"
	"testing"
	"time"

	"github.com/YSelim0/kick-logs/apps/api-go/internal/domain"
)

func TestCreateChannelRequestNormalizesSlug(t *testing.T) {
	repo := newMemoryRequestRepo()
	service := NewService(repo)

	created, err := service.Create(context.Background(), CreateInput{
		Type:        string(domain.UserRequestTypeChannelRequest),
		Title:       "Kanal eklensin",
		Message:     "Bu kanali takip listesine ekleyelim.",
		ChannelSlug: "https://kick.com/NuriBen?ref=test",
		Contact:     "mod@example.com",
	})
	if err != nil {
		t.Fatalf("Create() error = %v", err)
	}
	if created.ID == "" {
		t.Fatal("request id is empty")
	}
	if created.ChannelSlug != "nuriben" {
		t.Fatalf("ChannelSlug = %q", created.ChannelSlug)
	}
	if created.Type != domain.UserRequestTypeChannelRequest {
		t.Fatalf("Type = %q", created.Type)
	}
}

func TestCreateRejectsInvalidPayloads(t *testing.T) {
	service := NewService(newMemoryRequestRepo())

	tests := []CreateInput{
		{Type: "unknown", Title: "Valid title", Message: "valid message"},
		{Type: string(domain.UserRequestTypeFeedback), Title: "x", Message: "valid message"},
		{Type: string(domain.UserRequestTypeFeedback), Title: "Valid title", Message: "bad"},
		{Type: string(domain.UserRequestTypeChannelRequest), Title: "Valid title", Message: "valid message"},
		{Type: string(domain.UserRequestTypeChannelRequest), Title: "Valid title", Message: "valid message", ChannelSlug: "bad slug"},
	}

	for index, input := range tests {
		if _, err := service.Create(context.Background(), input); !errors.Is(err, ErrValidation) {
			t.Fatalf("case %d error = %v, want ErrValidation", index, err)
		}
	}
}

func TestAdminWorkflowAppendsEvents(t *testing.T) {
	repo := newMemoryRequestRepo()
	service := NewService(repo)
	ctx := context.Background()

	created, err := service.Create(ctx, CreateInput{
		Type:    string(domain.UserRequestTypeFeedback),
		Title:   "Feature request",
		Message: "Lutfen yeni bir rapor ekleyin.",
	})
	if err != nil {
		t.Fatalf("Create() error = %v", err)
	}

	detail, err := service.ChangeStatus(ctx, created.ID, string(domain.UserRequestStatusReviewing), 10)
	if err != nil {
		t.Fatalf("ChangeStatus() error = %v", err)
	}
	if detail.State.CurrentStatus != domain.UserRequestStatusReviewing {
		t.Fatalf("CurrentStatus = %q", detail.State.CurrentStatus)
	}

	detail, err = service.AddNote(ctx, created.ID, "Kontrol edilecek.", 10)
	if err != nil {
		t.Fatalf("AddNote() error = %v", err)
	}
	if len(detail.Events) != 2 || detail.Events[1].Note != "Kontrol edilecek." {
		t.Fatalf("events after note = %#v", detail.Events)
	}

	detail, err = service.Archive(ctx, created.ID, 10)
	if err != nil {
		t.Fatalf("Archive() error = %v", err)
	}
	if !detail.State.IsArchived {
		t.Fatal("request should be archived")
	}
	if len(detail.Events) != 3 || detail.Events[2].EventType != domain.UserRequestEventArchived {
		t.Fatalf("events after archive = %#v", detail.Events)
	}
}

func TestDetailReturnsNotFound(t *testing.T) {
	service := NewService(newMemoryRequestRepo())

	if _, err := service.Detail(context.Background(), "missing"); !errors.Is(err, ErrNotFound) {
		t.Fatalf("Detail() error = %v, want ErrNotFound", err)
	}
}

type memoryRequestRepo struct {
	requests map[string]domain.UserRequest
	events   map[string][]domain.UserRequestEvent
}

func newMemoryRequestRepo() *memoryRequestRepo {
	return &memoryRequestRepo{
		requests: map[string]domain.UserRequest{},
		events:   map[string][]domain.UserRequestEvent{},
	}
}

func (repo *memoryRequestRepo) Create(_ context.Context, request domain.UserRequest) error {
	repo.requests[request.ID] = request
	return nil
}

func (repo *memoryRequestRepo) List(
	_ context.Context,
	filter domain.UserRequestListFilter,
) ([]domain.UserRequestState, error) {
	states := make([]domain.UserRequestState, 0, len(repo.requests))
	for _, request := range repo.requests {
		state := repo.state(request.ID)
		if filter.Type != "" && state.Request.Type != filter.Type {
			continue
		}
		if filter.Status != "" && state.CurrentStatus != filter.Status {
			continue
		}
		if filter.Archived != nil && state.IsArchived != *filter.Archived {
			continue
		}
		states = append(states, state)
	}
	return states, nil
}

func (repo *memoryRequestRepo) Get(_ context.Context, requestID string) (domain.UserRequestState, error) {
	if _, ok := repo.requests[requestID]; !ok {
		return domain.UserRequestState{}, sql.ErrNoRows
	}
	return repo.state(requestID), nil
}

func (repo *memoryRequestRepo) ListEvents(_ context.Context, requestID string) ([]domain.UserRequestEvent, error) {
	events := append([]domain.UserRequestEvent(nil), repo.events[requestID]...)
	return events, nil
}

func (repo *memoryRequestRepo) AppendEvent(_ context.Context, event domain.UserRequestEvent) error {
	if _, ok := repo.requests[event.RequestID]; !ok {
		return sql.ErrNoRows
	}
	repo.events[event.RequestID] = append(repo.events[event.RequestID], event)
	return nil
}

func (repo *memoryRequestRepo) state(requestID string) domain.UserRequestState {
	request := repo.requests[requestID]
	state := domain.UserRequestState{
		Request:       request,
		CurrentStatus: domain.UserRequestStatusNew,
		LatestEventAt: request.CreatedAt,
	}
	for _, event := range repo.events[requestID] {
		if event.EventType == domain.UserRequestEventStatusChanged && event.Status != "" {
			state.CurrentStatus = event.Status
		}
		if event.EventType == domain.UserRequestEventArchived {
			state.IsArchived = true
		}
		if event.CreatedAt.After(state.LatestEventAt) || state.LatestEventAt.IsZero() {
			state.LatestEventAt = event.CreatedAt
		}
	}
	if state.LatestEventAt.IsZero() {
		state.LatestEventAt = time.Now().UTC()
	}
	return state
}
