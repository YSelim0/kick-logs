package datamigration_test

import (
	"context"
	"database/sql"
	"errors"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/YSelim0/kick-logs/apps/api-go/internal/domain"
	authinfra "github.com/YSelim0/kick-logs/apps/api-go/internal/infra/auth"
	"github.com/YSelim0/kick-logs/apps/api-go/internal/infra/migrations"
	sqliteinfra "github.com/YSelim0/kick-logs/apps/api-go/internal/infra/sqlite"
	"github.com/YSelim0/kick-logs/apps/api-go/internal/usecase/datamigration"
	"golang.org/x/crypto/bcrypt"
)

func TestServiceMigratesEmptySource(t *testing.T) {
	ctx := context.Background()
	source := newFakeSource()
	control := newFakeControlDestination()
	data := newFakeDataDestination()

	report, err := datamigration.NewService(datamigration.Dependencies{
		Source:  source,
		Control: control,
		Data:    data,
	}).Run(ctx, datamigration.Options{Execute: true, BatchSize: 2})
	if err != nil {
		t.Fatalf("Run() error = %v", err)
	}
	if report.SourceCounts != (domain.MigrationCounts{}) || report.DestinationCounts != (domain.MigrationCounts{}) {
		t.Fatalf("empty report counts = %#v", report)
	}
	if len(control.runs) != 1 || control.runs[0].Status != "succeeded" {
		t.Fatalf("migration runs = %#v", control.runs)
	}
}

func TestServiceMigratesRepresentativeDataAndIsIdempotent(t *testing.T) {
	ctx := context.Background()
	source := representativeSource(t)
	control := newFakeControlDestination()
	data := newFakeDataDestination()
	service := datamigration.NewService(datamigration.Dependencies{
		Source:  source,
		Control: control,
		Data:    data,
	})

	firstReport, err := service.Run(ctx, datamigration.Options{Execute: true, BatchSize: 1, SampleSize: 3})
	if err != nil {
		t.Fatalf("first Run() error = %v", err)
	}
	if firstReport.DestinationCounts.ChatMessages != 1 || firstReport.DestinationCounts.RawEventAttempts != 2 {
		t.Fatalf("first destination counts = %#v", firstReport.DestinationCounts)
	}
	if len(data.messages) != 1 || data.messages["kick-message-1"].ReplyToSender != "other_user" {
		t.Fatalf("migrated messages = %#v", data.messages)
	}
	if len(data.rawAttempts) != 2 || data.rawAttempts["postgres-raw-event:88:attempt:2"].Status != "processed" {
		t.Fatalf("migrated raw attempts = %#v", data.rawAttempts)
	}
	if !messageMatchesSearch(data.messages["kick-message-1"], "yavuz", "hype", "needle") {
		t.Fatalf("migrated message does not satisfy source-equivalent search fixture")
	}

	secondReport, err := service.Run(ctx, datamigration.Options{Execute: true, BatchSize: 1, SampleSize: 3})
	if err != nil {
		t.Fatalf("second Run() error = %v", err)
	}
	if secondReport.DestinationCounts != firstReport.DestinationCounts {
		t.Fatalf("destination counts changed on rerun: first=%#v second=%#v", firstReport.DestinationCounts, secondReport.DestinationCounts)
	}
	if secondReport.WrittenCounts.ChatMessages != 0 || secondReport.WrittenCounts.RawEvents != 0 || secondReport.WrittenCounts.RawEventAttempts != 0 {
		t.Fatalf("data rows duplicated on rerun: %#v", secondReport.WrittenCounts)
	}
}

func TestServiceValidationFailsOnDestinationMismatch(t *testing.T) {
	ctx := context.Background()
	source := representativeSource(t)
	control := newFakeControlDestination()
	data := newFakeDataDestination()
	service := datamigration.NewService(datamigration.Dependencies{
		Source:  source,
		Control: control,
		Data:    data,
	})
	if _, err := service.Run(ctx, datamigration.Options{Execute: true}); err != nil {
		t.Fatalf("seed Run() error = %v", err)
	}

	forcedMessageCount := int64(0)
	data.forceMessageCount = &forcedMessageCount
	_, err := service.Run(ctx, datamigration.Options{ValidationOnly: true})
	if err == nil || !strings.Contains(err.Error(), "chat message count mismatch") {
		t.Fatalf("validation error = %v", err)
	}
	if len(control.runs) < 2 || control.runs[len(control.runs)-1].Status != "failed" {
		t.Fatalf("failed validation run was not recorded: %#v", control.runs)
	}
}

func TestServiceRejectsGoIncompatiblePasswordHash(t *testing.T) {
	ctx := context.Background()
	source := newFakeSource()
	source.users = []domain.AdminUser{{
		ID:           1,
		Email:        "admin@kicklogs.local",
		PasswordHash: "not-a-bcrypt-hash",
		Role:         domain.AdminRoleSuperAdmin,
		IsActive:     true,
		CreatedAt:    time.Now().UTC(),
		UpdatedAt:    time.Now().UTC(),
	}}
	source.refreshCounts()

	_, err := datamigration.NewService(datamigration.Dependencies{
		Source:  source,
		Control: newFakeControlDestination(),
		Data:    newFakeDataDestination(),
	}).Run(ctx, datamigration.Options{DryRun: true})
	if err == nil || !strings.Contains(err.Error(), "Go-incompatible bcrypt hash") {
		t.Fatalf("Run() error = %v", err)
	}
}

func TestMigratedAdminHashWorksWithSQLiteLoginHasher(t *testing.T) {
	ctx := context.Background()
	source := newFakeSource()
	hash := bcryptHash(t, "admin123")
	now := time.Date(2026, 5, 16, 12, 0, 0, 0, time.UTC)
	source.users = []domain.AdminUser{{
		ID:           42,
		Email:        "ADMIN@kicklogs.local",
		PasswordHash: hash,
		Role:         domain.AdminRoleSuperAdmin,
		IsActive:     true,
		CreatedAt:    now,
		UpdatedAt:    now,
	}}
	source.refreshCounts()

	db, control := openSQLiteControl(t, ctx)
	defer db.Close()
	adminRepo := sqliteinfra.NewAdminUserRepository(db)
	if _, err := sqliteinfra.SeedSuperAdmin(ctx, adminRepo, "admin@kicklogs.local", "temporary123"); err != nil {
		t.Fatalf("SeedSuperAdmin() error = %v", err)
	}
	data := newFakeDataDestination()
	if _, err := datamigration.NewService(datamigration.Dependencies{
		Source:  source,
		Control: control,
		Data:    data,
	}).Run(ctx, datamigration.Options{Execute: true}); err != nil {
		t.Fatalf("Run() error = %v", err)
	}

	user, err := adminRepo.GetByEmail(ctx, "admin@kicklogs.local")
	if err != nil {
		t.Fatalf("GetByEmail() error = %v", err)
	}
	if user.ID != 42 {
		t.Fatalf("migrated user id = %d", user.ID)
	}
	if !authinfra.NewBcryptPasswordHasher().Verify("admin123", user.PasswordHash) {
		t.Fatal("migrated bcrypt hash does not verify")
	}
}

type fakeSource struct {
	users      []domain.AdminUser
	channels   []domain.FollowedChannel
	senders    []domain.SenderProfile
	retention  []domain.RetentionSettings
	heartbeats []domain.ListenerHeartbeat
	messages   []domain.ChatMessage
	rawEvents  []domain.RawKickEvent
	counts     domain.MigrationCounts
}

func newFakeSource() *fakeSource {
	source := &fakeSource{}
	source.refreshCounts()
	return source
}

func (source *fakeSource) Counts(context.Context) (domain.MigrationCounts, error) {
	return source.counts, nil
}

func (source *fakeSource) AdminUsers(_ context.Context, limit int, offset int) ([]domain.AdminUser, error) {
	return page(source.users, limit, offset), nil
}

func (source *fakeSource) FollowedChannels(_ context.Context, limit int, offset int) ([]domain.FollowedChannel, error) {
	return page(source.channels, limit, offset), nil
}

func (source *fakeSource) SenderProfiles(_ context.Context, limit int, offset int) ([]domain.SenderProfile, error) {
	return page(source.senders, limit, offset), nil
}

func (source *fakeSource) RetentionSettings(_ context.Context, limit int, offset int) ([]domain.RetentionSettings, error) {
	return page(source.retention, limit, offset), nil
}

func (source *fakeSource) WorkerHeartbeats(_ context.Context, limit int, offset int) ([]domain.ListenerHeartbeat, error) {
	return page(source.heartbeats, limit, offset), nil
}

func (source *fakeSource) ChatMessages(_ context.Context, limit int, offset int) ([]domain.ChatMessage, error) {
	return page(source.messages, limit, offset), nil
}

func (source *fakeSource) RawEvents(_ context.Context, limit int, offset int) ([]domain.RawKickEvent, error) {
	return page(source.rawEvents, limit, offset), nil
}

func (source *fakeSource) refreshCounts() {
	source.counts = domain.MigrationCounts{
		AdminUsers:        int64(len(source.users)),
		FollowedChannels:  int64(len(source.channels)),
		SenderProfiles:    int64(len(source.senders)),
		RetentionSettings: int64(len(source.retention)),
		WorkerHeartbeats:  int64(len(source.heartbeats)),
		ChatMessages:      int64(len(source.messages)),
		RawEvents:         int64(len(source.rawEvents)),
	}
	for _, event := range source.rawEvents {
		source.counts.RawEventAttempts += int64(len(datamigration.AttemptsForRawEvent(event)))
	}
}

type fakeControlDestination struct {
	users      map[int64]domain.AdminUser
	channels   map[int64]domain.FollowedChannel
	senders    map[int64]domain.SenderProfile
	retention  map[int64]domain.RetentionSettings
	heartbeats map[string]domain.ListenerHeartbeat
	runs       []domain.DataMigrationRun
}

func newFakeControlDestination() *fakeControlDestination {
	return &fakeControlDestination{
		users:      map[int64]domain.AdminUser{},
		channels:   map[int64]domain.FollowedChannel{},
		senders:    map[int64]domain.SenderProfile{},
		retention:  map[int64]domain.RetentionSettings{},
		heartbeats: map[string]domain.ListenerHeartbeat{},
	}
}

func (dest *fakeControlDestination) UpsertAdminUser(_ context.Context, user domain.AdminUser) error {
	user.Email = strings.ToLower(strings.TrimSpace(user.Email))
	dest.users[user.ID] = user
	return nil
}

func (dest *fakeControlDestination) UpsertFollowedChannel(_ context.Context, channel domain.FollowedChannel) error {
	channel.Slug = strings.ToLower(strings.TrimSpace(channel.Slug))
	dest.channels[channel.ID] = channel
	return nil
}

func (dest *fakeControlDestination) UpsertSenderProfile(_ context.Context, sender domain.SenderProfile) error {
	sender.Slug = strings.ToLower(strings.TrimSpace(sender.Slug))
	dest.senders[sender.ID] = sender
	return nil
}

func (dest *fakeControlDestination) UpsertRetentionSettings(_ context.Context, settings domain.RetentionSettings) error {
	dest.retention[settings.ID] = settings
	return nil
}

func (dest *fakeControlDestination) UpsertWorkerHeartbeat(_ context.Context, heartbeat domain.ListenerHeartbeat) error {
	dest.heartbeats[heartbeat.ServiceName] = heartbeat
	return nil
}

func (dest *fakeControlDestination) ControlCounts(context.Context) (domain.MigrationCounts, error) {
	return domain.MigrationCounts{
		AdminUsers:        int64(len(dest.users)),
		FollowedChannels:  int64(len(dest.channels)),
		SenderProfiles:    int64(len(dest.senders)),
		RetentionSettings: int64(len(dest.retention)),
		WorkerHeartbeats:  int64(len(dest.heartbeats)),
	}, nil
}

func (dest *fakeControlDestination) FindAdminUser(_ context.Context, id int64) (domain.AdminUser, error) {
	user, ok := dest.users[id]
	if !ok {
		return domain.AdminUser{}, sql.ErrNoRows
	}
	return user, nil
}

func (dest *fakeControlDestination) FindFollowedChannel(_ context.Context, id int64) (domain.FollowedChannel, error) {
	channel, ok := dest.channels[id]
	if !ok {
		return domain.FollowedChannel{}, sql.ErrNoRows
	}
	return channel, nil
}

func (dest *fakeControlDestination) FindSenderProfile(_ context.Context, id int64) (domain.SenderProfile, error) {
	sender, ok := dest.senders[id]
	if !ok {
		return domain.SenderProfile{}, sql.ErrNoRows
	}
	return sender, nil
}

func (dest *fakeControlDestination) RecordRun(_ context.Context, run domain.DataMigrationRun) error {
	dest.runs = append(dest.runs, run)
	return nil
}

type fakeDataDestination struct {
	messages          map[string]domain.ChatMessage
	rawEvents         map[string]domain.RawKickEvent
	rawAttempts       map[string]domain.RawEventAttempt
	forceMessageCount *int64
}

func newFakeDataDestination() *fakeDataDestination {
	return &fakeDataDestination{
		messages:    map[string]domain.ChatMessage{},
		rawEvents:   map[string]domain.RawKickEvent{},
		rawAttempts: map[string]domain.RawEventAttempt{},
	}
}

func (dest *fakeDataDestination) UpsertChatMessage(_ context.Context, message domain.ChatMessage) (bool, error) {
	if _, ok := dest.messages[message.KickMessageID]; ok {
		return false, nil
	}
	dest.messages[message.KickMessageID] = message
	return true, nil
}

func (dest *fakeDataDestination) UpsertRawEvent(_ context.Context, event domain.RawKickEvent) (bool, error) {
	if _, ok := dest.rawEvents[event.ID]; ok {
		return false, nil
	}
	dest.rawEvents[event.ID] = event
	return true, nil
}

func (dest *fakeDataDestination) UpsertRawEventAttempt(_ context.Context, attempt domain.RawEventAttempt) (bool, error) {
	if _, ok := dest.rawAttempts[attempt.ID]; ok {
		return false, nil
	}
	dest.rawAttempts[attempt.ID] = attempt
	return true, nil
}

func (dest *fakeDataDestination) DataCounts(context.Context) (domain.MigrationCounts, error) {
	messageCount := int64(len(dest.messages))
	if dest.forceMessageCount != nil {
		messageCount = *dest.forceMessageCount
	}
	return domain.MigrationCounts{
		ChatMessages:     messageCount,
		RawEvents:        int64(len(dest.rawEvents)),
		RawEventAttempts: int64(len(dest.rawAttempts)),
	}, nil
}

func (dest *fakeDataDestination) FindChatMessage(_ context.Context, id int64, kickMessageID string) (domain.ChatMessage, error) {
	message, ok := dest.messages[kickMessageID]
	if !ok || message.ID != id {
		return domain.ChatMessage{}, errors.New("message not found")
	}
	return message, nil
}

func (dest *fakeDataDestination) FindRawEvent(_ context.Context, id string) (domain.RawKickEvent, error) {
	event, ok := dest.rawEvents[id]
	if !ok {
		return domain.RawKickEvent{}, errors.New("raw event not found")
	}
	return event, nil
}

func representativeSource(t *testing.T) *fakeSource {
	t.Helper()

	now := time.Date(2026, 5, 16, 12, 0, 0, 0, time.UTC)
	source := newFakeSource()
	source.users = []domain.AdminUser{{
		ID:           1,
		Email:        "admin@kicklogs.local",
		PasswordHash: bcryptHash(t, "admin123"),
		Role:         domain.AdminRoleSuperAdmin,
		IsActive:     true,
		CreatedAt:    now,
		UpdatedAt:    now,
	}}
	source.channels = []domain.FollowedChannel{{
		ID:              11,
		KickChannelID:   111,
		KickChatroomID:  222,
		Slug:            "hype",
		DisplayName:     "Hype",
		ProfileImageURL: "https://example.com/channel.png",
		IsEnabled:       true,
		RawPayloadJSON:  `{"slug":"hype"}`,
		CreatedAt:       now,
		UpdatedAt:       now,
	}}
	source.senders = []domain.SenderProfile{{
		ID:                    22,
		KickUserID:            333,
		Username:              "yavuz",
		Slug:                  "yavuz",
		ProfileImageURL:       "https://example.com/avatar.png",
		LastSeenColor:         "#fff600",
		RawProfilePayloadJSON: `{"id":333}`,
		CreatedAt:             now,
		UpdatedAt:             now,
		LastSeenAt:            now,
	}}
	retentionDays := 90
	source.retention = []domain.RetentionSettings{{
		ID:                    1,
		MessageRetentionDays:  &retentionDays,
		RawEventRetentionDays: nil,
		CreatedAt:             now,
		UpdatedAt:             now,
	}}
	source.heartbeats = []domain.ListenerHeartbeat{{
		ServiceName:  "listener",
		LastSeenAt:   now,
		MetadataJSON: `{"worker_count":4}`,
	}}
	source.messages = []domain.ChatMessage{{
		ID:                     77,
		KickMessageID:          "kick-message-1",
		ChannelID:              11,
		ChannelKickID:          111,
		ChannelChatroomID:      222,
		ChannelSlug:            "hype",
		ChannelDisplayName:     "Hype",
		ChannelProfileImageURL: "https://example.com/channel.png",
		ChannelPublicURL:       "https://kick.com/hype",
		SenderID:               22,
		SenderKickID:           333,
		SenderUsername:         "yavuz",
		SenderSlug:             "yavuz",
		SenderDisplayColor:     "#fff600",
		SenderProfileImageURL:  "https://example.com/avatar.png",
		SenderPublicURL:        "https://kick.com/yavuz",
		SenderBadgesJSON:       `[{"text":"mod"}]`,
		MessageType:            "reply",
		Content:                "hello needle [emote:123:wave]",
		Emotes: []domain.ChatEmote{{
			ID:       "123",
			Name:     "wave",
			Token:    "[emote:123:wave]",
			ImageURL: "https://files.kick.com/emotes/123/fullsize",
		}},
		ReplyToSender:     "other_user",
		ReplyToContent:    "older message",
		ReplyToMessageID:  "parent-message",
		ThreadParentID:    "parent-message",
		ReplyMetadataJSON: `{"original_sender":{"username":"other_user"},"original_message":{"content":"older message"},"message_ref":"parent-message"}`,
		RawPayloadJSON:    `{"id":"kick-message-1"}`,
		MessageCreatedAt:  now,
		IngestedAt:        now.Add(time.Second),
	}}
	source.rawEvents = []domain.RawKickEvent{{
		ID:                  "88",
		ChannelSlug:         "hype",
		EventType:           "pusher",
		EventName:           `App\Events\ChatMessageEvent`,
		KickMessageID:       "kick-message-1",
		ChatroomID:          222,
		ChannelID:           11,
		PayloadJSON:         `{"id":"kick-message-1"}`,
		MetadataJSON:        `{"source":"postgres"}`,
		Status:              "processed",
		Attempts:            2,
		ReceivedAt:          now,
		ProcessingStartedAt: now.Add(time.Second),
		ProcessedAt:         now.Add(2 * time.Second),
	}}
	source.refreshCounts()
	return source
}

func openSQLiteControl(t *testing.T, ctx context.Context) (*sql.DB, *sqliteinfra.DataMigrationRepository) {
	t.Helper()

	db, err := sqliteinfra.Open(ctx, filepath.Join(t.TempDir(), "kick-logs.sqlite3"))
	if err != nil {
		t.Fatalf("sqlite Open() error = %v", err)
	}
	if err := migrations.ApplySQLite(ctx, db); err != nil {
		db.Close()
		t.Fatalf("ApplySQLite() error = %v", err)
	}
	return db, sqliteinfra.NewDataMigrationRepository(db)
}

func bcryptHash(t *testing.T, password string) string {
	t.Helper()
	hash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.MinCost)
	if err != nil {
		t.Fatalf("GenerateFromPassword() error = %v", err)
	}
	return string(hash)
}

func page[T any](items []T, limit int, offset int) []T {
	if offset >= len(items) {
		return nil
	}
	end := offset + limit
	if end > len(items) {
		end = len(items)
	}
	return append([]T(nil), items[offset:end]...)
}

func messageMatchesSearch(message domain.ChatMessage, sender string, channel string, query string) bool {
	return strings.EqualFold(message.SenderUsername, sender) &&
		strings.Contains(strings.ToLower(message.ChannelSlug), strings.ToLower(channel)) &&
		strings.Contains(strings.ToLower(message.Content), strings.ToLower(query))
}
