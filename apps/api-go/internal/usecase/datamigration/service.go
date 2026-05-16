package datamigration

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
	"golang.org/x/crypto/bcrypt"

	"github.com/YSelim0/kick-logs/apps/api-go/internal/domain"
)

const migrationName = "postgres_to_clickhouse_sqlite"

type Source interface {
	Counts(ctx context.Context) (domain.MigrationCounts, error)
	AdminUsers(ctx context.Context, limit int, offset int) ([]domain.AdminUser, error)
	FollowedChannels(ctx context.Context, limit int, offset int) ([]domain.FollowedChannel, error)
	SenderProfiles(ctx context.Context, limit int, offset int) ([]domain.SenderProfile, error)
	RetentionSettings(ctx context.Context, limit int, offset int) ([]domain.RetentionSettings, error)
	WorkerHeartbeats(ctx context.Context, limit int, offset int) ([]domain.ListenerHeartbeat, error)
	ChatMessages(ctx context.Context, limit int, offset int) ([]domain.ChatMessage, error)
	RawEvents(ctx context.Context, limit int, offset int) ([]domain.RawKickEvent, error)
}

type ControlDestination interface {
	UpsertAdminUser(ctx context.Context, user domain.AdminUser) error
	UpsertFollowedChannel(ctx context.Context, channel domain.FollowedChannel) error
	UpsertSenderProfile(ctx context.Context, sender domain.SenderProfile) error
	UpsertRetentionSettings(ctx context.Context, settings domain.RetentionSettings) error
	UpsertWorkerHeartbeat(ctx context.Context, heartbeat domain.ListenerHeartbeat) error
	ControlCounts(ctx context.Context) (domain.MigrationCounts, error)
	FindAdminUser(ctx context.Context, id int64) (domain.AdminUser, error)
	FindFollowedChannel(ctx context.Context, id int64) (domain.FollowedChannel, error)
	FindSenderProfile(ctx context.Context, id int64) (domain.SenderProfile, error)
	RecordRun(ctx context.Context, run domain.DataMigrationRun) error
}

type DataDestination interface {
	UpsertChatMessage(ctx context.Context, message domain.ChatMessage) (bool, error)
	UpsertChatMessages(ctx context.Context, messages []domain.ChatMessage) (int64, error)
	UpsertRawEvent(ctx context.Context, event domain.RawKickEvent) (bool, error)
	UpsertRawEvents(ctx context.Context, events []domain.RawKickEvent) (int64, error)
	UpsertRawEventAttempt(ctx context.Context, attempt domain.RawEventAttempt) (bool, error)
	UpsertRawEventAttempts(ctx context.Context, attempts []domain.RawEventAttempt) (int64, error)
	DataCounts(ctx context.Context) (domain.MigrationCounts, error)
	FindChatMessage(ctx context.Context, id int64, kickMessageID string) (domain.ChatMessage, error)
	FindRawEvent(ctx context.Context, id string) (domain.RawKickEvent, error)
}

type Service struct {
	source  Source
	control ControlDestination
	data    DataDestination
}

type Dependencies struct {
	Source  Source
	Control ControlDestination
	Data    DataDestination
}

func NewService(deps Dependencies) *Service {
	return &Service{source: deps.Source, control: deps.Control, data: deps.Data}
}

type Options struct {
	DryRun         bool
	Execute        bool
	ValidationOnly bool
	BatchSize      int
	SampleSize     int
}

type Report struct {
	RunID             string
	Mode              string
	SourceCounts      domain.MigrationCounts
	DestinationCounts domain.MigrationCounts
	WrittenCounts     domain.MigrationCounts
	Validation        map[string]string
	StartedAt         time.Time
	FinishedAt        time.Time
}

func (service *Service) Run(ctx context.Context, options Options) (Report, error) {
	if service.source == nil {
		return Report{}, fmt.Errorf("postgres source is required")
	}
	if service.control == nil {
		return Report{}, fmt.Errorf("sqlite control destination is required")
	}
	if service.data == nil {
		return Report{}, fmt.Errorf("clickhouse data destination is required")
	}

	options = options.withDefaults()
	report := Report{
		RunID:      uuid.NewString(),
		Mode:       options.mode(),
		Validation: make(map[string]string),
		StartedAt:  time.Now().UTC(),
	}

	sourceCounts, err := service.source.Counts(ctx)
	if err != nil {
		return report, err
	}
	report.SourceCounts = sourceCounts

	if err := service.validateSource(ctx, options); err != nil {
		report.FinishedAt = time.Now().UTC()
		if options.DryRun {
			return report, err
		}
		return report, service.recordFailed(ctx, report, err)
	}
	report.Validation["source"] = "ok"

	if options.DryRun {
		report.FinishedAt = time.Now().UTC()
		return report, nil
	}

	if options.Execute {
		written, err := service.execute(ctx, options)
		report.WrittenCounts = written
		if err != nil {
			report.FinishedAt = time.Now().UTC()
			return report, service.recordFailed(ctx, report, err)
		}
	}

	destinationCounts, err := service.destinationCounts(ctx)
	if err != nil {
		report.FinishedAt = time.Now().UTC()
		return report, service.recordFailed(ctx, report, err)
	}
	report.DestinationCounts = destinationCounts

	if err := service.validateDestination(ctx, options, sourceCounts, destinationCounts); err != nil {
		report.FinishedAt = time.Now().UTC()
		return report, service.recordFailed(ctx, report, err)
	}
	report.Validation["destination"] = "ok"

	report.FinishedAt = time.Now().UTC()
	if err := service.control.RecordRun(ctx, report.toRun("succeeded", "")); err != nil {
		return report, err
	}
	return report, nil
}

func (service *Service) validateSource(ctx context.Context, options Options) error {
	return service.forEachAdminUser(ctx, options.BatchSize, func(user domain.AdminUser) error {
		if strings.TrimSpace(user.Email) == "" {
			return fmt.Errorf("source user %d has empty email", user.ID)
		}
		if user.Role != domain.AdminRoleAdmin && user.Role != domain.AdminRoleSuperAdmin {
			return fmt.Errorf("source user %d has unsupported role %q", user.ID, user.Role)
		}
		if _, err := bcrypt.Cost([]byte(user.PasswordHash)); err != nil {
			return fmt.Errorf("source user %d has Go-incompatible bcrypt hash: %w", user.ID, err)
		}
		return nil
	})
}

func (service *Service) execute(ctx context.Context, options Options) (domain.MigrationCounts, error) {
	written := domain.MigrationCounts{}

	if err := service.forEachAdminUser(ctx, options.BatchSize, func(user domain.AdminUser) error {
		if err := service.control.UpsertAdminUser(ctx, user); err != nil {
			return err
		}
		written.AdminUsers++
		return nil
	}); err != nil {
		return written, err
	}

	if err := service.forEachFollowedChannel(ctx, options.BatchSize, func(channel domain.FollowedChannel) error {
		if err := service.control.UpsertFollowedChannel(ctx, channel); err != nil {
			return err
		}
		written.FollowedChannels++
		return nil
	}); err != nil {
		return written, err
	}

	if err := service.forEachSenderProfile(ctx, options.BatchSize, func(sender domain.SenderProfile) error {
		if err := service.control.UpsertSenderProfile(ctx, sender); err != nil {
			return err
		}
		written.SenderProfiles++
		return nil
	}); err != nil {
		return written, err
	}

	if err := service.forEachRetentionSettings(ctx, options.BatchSize, func(settings domain.RetentionSettings) error {
		if err := service.control.UpsertRetentionSettings(ctx, settings); err != nil {
			return err
		}
		written.RetentionSettings++
		return nil
	}); err != nil {
		return written, err
	}

	if err := service.forEachWorkerHeartbeat(ctx, options.BatchSize, func(heartbeat domain.ListenerHeartbeat) error {
		if err := service.control.UpsertWorkerHeartbeat(ctx, heartbeat); err != nil {
			return err
		}
		written.WorkerHeartbeats++
		return nil
	}); err != nil {
		return written, err
	}

	for offset := 0; ; offset += options.BatchSize {
		messages, err := service.source.ChatMessages(ctx, options.BatchSize, offset)
		if err != nil {
			return written, err
		}
		if len(messages) == 0 {
			break
		}
		for index := range messages {
			messages[index] = normalizeMessage(messages[index])
		}
		inserted, err := service.data.UpsertChatMessages(ctx, messages)
		if err != nil {
			return written, err
		}
		written.ChatMessages += inserted
	}

	for offset := 0; ; offset += options.BatchSize {
		events, err := service.source.RawEvents(ctx, options.BatchSize, offset)
		if err != nil {
			return written, err
		}
		if len(events) == 0 {
			break
		}
		attempts := make([]domain.RawEventAttempt, 0, len(events))
		for index := range events {
			events[index] = normalizeRawEvent(events[index])
			attempts = append(attempts, AttemptsForRawEvent(events[index])...)
		}
		insertedEvents, err := service.data.UpsertRawEvents(ctx, events)
		if err != nil {
			return written, err
		}
		written.RawEvents += insertedEvents
		insertedAttempts, err := service.data.UpsertRawEventAttempts(ctx, attempts)
		if err != nil {
			return written, err
		}
		written.RawEventAttempts += insertedAttempts
	}

	return written, nil
}

func (service *Service) validateDestination(
	ctx context.Context,
	options Options,
	sourceCounts domain.MigrationCounts,
	destinationCounts domain.MigrationCounts,
) error {
	if destinationCounts.AdminUsers != sourceCounts.AdminUsers {
		return fmt.Errorf("admin user count mismatch: source=%d destination=%d", sourceCounts.AdminUsers, destinationCounts.AdminUsers)
	}
	if destinationCounts.FollowedChannels != sourceCounts.FollowedChannels {
		return fmt.Errorf("followed channel count mismatch: source=%d destination=%d", sourceCounts.FollowedChannels, destinationCounts.FollowedChannels)
	}
	if destinationCounts.SenderProfiles != sourceCounts.SenderProfiles {
		return fmt.Errorf("sender profile count mismatch: source=%d destination=%d", sourceCounts.SenderProfiles, destinationCounts.SenderProfiles)
	}
	if destinationCounts.RetentionSettings != sourceCounts.RetentionSettings {
		return fmt.Errorf("retention settings count mismatch: source=%d destination=%d", sourceCounts.RetentionSettings, destinationCounts.RetentionSettings)
	}
	if destinationCounts.WorkerHeartbeats != sourceCounts.WorkerHeartbeats {
		return fmt.Errorf("worker heartbeat count mismatch: source=%d destination=%d", sourceCounts.WorkerHeartbeats, destinationCounts.WorkerHeartbeats)
	}
	if destinationCounts.ChatMessages != sourceCounts.ChatMessages {
		return fmt.Errorf("chat message count mismatch: source=%d destination=%d", sourceCounts.ChatMessages, destinationCounts.ChatMessages)
	}
	if destinationCounts.RawEvents != sourceCounts.RawEvents {
		return fmt.Errorf("raw event count mismatch: source=%d destination=%d", sourceCounts.RawEvents, destinationCounts.RawEvents)
	}
	if destinationCounts.RawEventAttempts != sourceCounts.RawEventAttempts {
		return fmt.Errorf("raw event attempt count mismatch: source=%d destination=%d", sourceCounts.RawEventAttempts, destinationCounts.RawEventAttempts)
	}

	return service.validateSamples(ctx, options.SampleSize)
}

func (service *Service) validateSamples(ctx context.Context, sampleSize int) error {
	admins, err := service.source.AdminUsers(ctx, sampleSize, 0)
	if err != nil {
		return err
	}
	for _, sourceUser := range admins {
		destinationUser, err := service.control.FindAdminUser(ctx, sourceUser.ID)
		if err != nil {
			return err
		}
		if destinationUser.Email != strings.ToLower(strings.TrimSpace(sourceUser.Email)) ||
			destinationUser.PasswordHash != sourceUser.PasswordHash ||
			destinationUser.Role != sourceUser.Role {
			return fmt.Errorf("admin user sample mismatch for id %d", sourceUser.ID)
		}
	}

	channels, err := service.source.FollowedChannels(ctx, sampleSize, 0)
	if err != nil {
		return err
	}
	for _, sourceChannel := range channels {
		destinationChannel, err := service.control.FindFollowedChannel(ctx, sourceChannel.ID)
		if err != nil {
			return err
		}
		if destinationChannel.Slug != strings.ToLower(strings.TrimSpace(sourceChannel.Slug)) ||
			destinationChannel.KickChannelID != sourceChannel.KickChannelID ||
			destinationChannel.KickChatroomID != sourceChannel.KickChatroomID {
			return fmt.Errorf("followed channel sample mismatch for id %d", sourceChannel.ID)
		}
	}

	senders, err := service.source.SenderProfiles(ctx, sampleSize, 0)
	if err != nil {
		return err
	}
	for _, sourceSender := range senders {
		destinationSender, err := service.control.FindSenderProfile(ctx, sourceSender.ID)
		if err != nil {
			return err
		}
		if destinationSender.KickUserID != sourceSender.KickUserID ||
			destinationSender.Username != sourceSender.Username ||
			destinationSender.Slug != strings.ToLower(strings.TrimSpace(sourceSender.Slug)) {
			return fmt.Errorf("sender profile sample mismatch for id %d", sourceSender.ID)
		}
	}

	messages, err := service.source.ChatMessages(ctx, sampleSize, 0)
	if err != nil {
		return err
	}
	for _, sourceMessage := range messages {
		destinationMessage, err := service.data.FindChatMessage(ctx, sourceMessage.ID, sourceMessage.KickMessageID)
		if err != nil {
			return err
		}
		if destinationMessage.Content != sourceMessage.Content ||
			destinationMessage.SenderUsername != sourceMessage.SenderUsername ||
			len(destinationMessage.Emotes) != len(sourceMessage.Emotes) {
			return fmt.Errorf("chat message sample mismatch for id %d", sourceMessage.ID)
		}
	}

	rawEvents, err := service.source.RawEvents(ctx, sampleSize, 0)
	if err != nil {
		return err
	}
	for _, sourceEvent := range rawEvents {
		destinationEvent, err := service.data.FindRawEvent(ctx, sourceEvent.ID)
		if err != nil {
			return err
		}
		if destinationEvent.EventName != sourceEvent.EventName ||
			destinationEvent.Status != sourceEvent.Status ||
			destinationEvent.PayloadJSON != sourceEvent.PayloadJSON {
			return fmt.Errorf("raw event sample mismatch for id %s", sourceEvent.ID)
		}
	}

	return nil
}

func AttemptsForRawEvent(event domain.RawKickEvent) []domain.RawEventAttempt {
	count := int(event.Attempts)
	if count == 0 && (event.Status == "processed" || event.Status == "failed" || event.Status == "processing") {
		count = 1
	}
	if count == 0 {
		return nil
	}

	attempts := make([]domain.RawEventAttempt, 0, count)
	for index := 1; index <= count; index++ {
		status := migratedAttemptStatus(event.Status, index, count)
		startedAt := event.ReceivedAt.Add(time.Duration(index) * time.Millisecond).UTC()
		if index == count && !event.ProcessingStartedAt.IsZero() {
			startedAt = event.ProcessingStartedAt.UTC()
		}
		finishedAt := startedAt
		if status == "processed" && !event.ProcessedAt.IsZero() {
			finishedAt = event.ProcessedAt.UTC()
		}
		if status == "processing" {
			finishedAt = time.Time{}
		}

		errorMessage := ""
		if status == "failed" {
			errorMessage = event.ErrorMessage
			if errorMessage == "" && index < count {
				errorMessage = "migrated historical failed attempt"
			}
		}
		attempts = append(attempts, domain.RawEventAttempt{
			ID:           fmt.Sprintf("postgres-raw-event:%s:attempt:%d", event.ID, index),
			RawEventID:   event.ID,
			Attempt:      uint16(index),
			Status:       status,
			ErrorMessage: errorMessage,
			StartedAt:    startedAt,
			FinishedAt:   finishedAt,
		})
	}
	return attempts
}

func migratedAttemptStatus(eventStatus string, index int, count int) string {
	if index < count {
		return "failed"
	}
	switch eventStatus {
	case "processed", "failed", "processing":
		return eventStatus
	case "pending":
		return "failed"
	default:
		if eventStatus != "" {
			return eventStatus
		}
		return "failed"
	}
}

func (service *Service) destinationCounts(ctx context.Context) (domain.MigrationCounts, error) {
	controlCounts, err := service.control.ControlCounts(ctx)
	if err != nil {
		return domain.MigrationCounts{}, err
	}
	dataCounts, err := service.data.DataCounts(ctx)
	if err != nil {
		return domain.MigrationCounts{}, err
	}
	controlCounts.ChatMessages = dataCounts.ChatMessages
	controlCounts.RawEvents = dataCounts.RawEvents
	controlCounts.RawEventAttempts = dataCounts.RawEventAttempts
	return controlCounts, nil
}

func normalizeMessage(message domain.ChatMessage) domain.ChatMessage {
	message.MessageCreatedAt = message.MessageCreatedAt.UTC()
	message.IngestedAt = message.IngestedAt.UTC()
	if message.ChannelPublicURL == "" && message.ChannelSlug != "" {
		message.ChannelPublicURL = "https://kick.com/" + strings.TrimSpace(message.ChannelSlug)
	}
	if message.SenderPublicURL == "" && message.SenderSlug != "" {
		message.SenderPublicURL = "https://kick.com/" + strings.TrimSpace(message.SenderSlug)
	}
	if message.SenderBadgesJSON == "" {
		message.SenderBadgesJSON = "[]"
	}
	if message.ReplyMetadataJSON == "" {
		message.ReplyMetadataJSON = "{}"
	}
	if message.RawPayloadJSON == "" {
		message.RawPayloadJSON = "{}"
	}
	if message.MessageType == "" {
		message.MessageType = "message"
	}
	return message
}

func normalizeRawEvent(event domain.RawKickEvent) domain.RawKickEvent {
	event.ReceivedAt = event.ReceivedAt.UTC()
	event.ProcessedAt = event.ProcessedAt.UTC()
	event.ProcessingStartedAt = event.ProcessingStartedAt.UTC()
	if event.EventType == "" {
		event.EventType = "pusher"
	}
	if event.PayloadJSON == "" {
		event.PayloadJSON = "{}"
	}
	if event.MetadataJSON == "" {
		event.MetadataJSON = "{}"
	}
	if event.Status == "" {
		event.Status = "pending"
	}
	return event
}

func (service *Service) forEachAdminUser(ctx context.Context, batchSize int, visit func(domain.AdminUser) error) error {
	for offset := 0; ; offset += batchSize {
		items, err := service.source.AdminUsers(ctx, batchSize, offset)
		if err != nil {
			return err
		}
		if len(items) == 0 {
			return nil
		}
		for _, item := range items {
			if err := visit(item); err != nil {
				return err
			}
		}
	}
}

func (service *Service) forEachFollowedChannel(ctx context.Context, batchSize int, visit func(domain.FollowedChannel) error) error {
	for offset := 0; ; offset += batchSize {
		items, err := service.source.FollowedChannels(ctx, batchSize, offset)
		if err != nil {
			return err
		}
		if len(items) == 0 {
			return nil
		}
		for _, item := range items {
			if err := visit(item); err != nil {
				return err
			}
		}
	}
}

func (service *Service) forEachSenderProfile(ctx context.Context, batchSize int, visit func(domain.SenderProfile) error) error {
	for offset := 0; ; offset += batchSize {
		items, err := service.source.SenderProfiles(ctx, batchSize, offset)
		if err != nil {
			return err
		}
		if len(items) == 0 {
			return nil
		}
		for _, item := range items {
			if err := visit(item); err != nil {
				return err
			}
		}
	}
}

func (service *Service) forEachRetentionSettings(ctx context.Context, batchSize int, visit func(domain.RetentionSettings) error) error {
	for offset := 0; ; offset += batchSize {
		items, err := service.source.RetentionSettings(ctx, batchSize, offset)
		if err != nil {
			return err
		}
		if len(items) == 0 {
			return nil
		}
		for _, item := range items {
			if err := visit(item); err != nil {
				return err
			}
		}
	}
}

func (service *Service) forEachWorkerHeartbeat(ctx context.Context, batchSize int, visit func(domain.ListenerHeartbeat) error) error {
	for offset := 0; ; offset += batchSize {
		items, err := service.source.WorkerHeartbeats(ctx, batchSize, offset)
		if err != nil {
			return err
		}
		if len(items) == 0 {
			return nil
		}
		for _, item := range items {
			if err := visit(item); err != nil {
				return err
			}
		}
	}
}

func (service *Service) forEachChatMessage(ctx context.Context, batchSize int, visit func(domain.ChatMessage) error) error {
	for offset := 0; ; offset += batchSize {
		items, err := service.source.ChatMessages(ctx, batchSize, offset)
		if err != nil {
			return err
		}
		if len(items) == 0 {
			return nil
		}
		for _, item := range items {
			if err := visit(item); err != nil {
				return err
			}
		}
	}
}

func (service *Service) forEachRawEvent(ctx context.Context, batchSize int, visit func(domain.RawKickEvent) error) error {
	for offset := 0; ; offset += batchSize {
		items, err := service.source.RawEvents(ctx, batchSize, offset)
		if err != nil {
			return err
		}
		if len(items) == 0 {
			return nil
		}
		for _, item := range items {
			if err := visit(item); err != nil {
				return err
			}
		}
	}
}

func (options Options) withDefaults() Options {
	selectedModes := 0
	if options.DryRun {
		selectedModes++
	}
	if options.Execute {
		selectedModes++
	}
	if options.ValidationOnly {
		selectedModes++
	}
	if selectedModes == 0 {
		options.DryRun = true
	}
	if options.BatchSize < 1 {
		options.BatchSize = 500
	}
	if options.SampleSize < 1 {
		options.SampleSize = 5
	}
	return options
}

func (options Options) mode() string {
	if options.Execute {
		return "execute"
	}
	if options.ValidationOnly {
		return "validation-only"
	}
	return "dry-run"
}

func (report Report) toRun(status string, message string) domain.DataMigrationRun {
	return domain.DataMigrationRun{
		RunID:                 report.RunID,
		Name:                  migrationName,
		Mode:                  report.Mode,
		Status:                status,
		SourceCountsJSON:      jsonString(report.SourceCounts),
		DestinationCountsJSON: jsonString(report.DestinationCounts),
		ValidationJSON:        jsonString(report.Validation),
		ErrorMessage:          message,
		StartedAt:             report.StartedAt,
		FinishedAt:            report.FinishedAt,
	}
}

func (service *Service) recordFailed(ctx context.Context, report Report, cause error) error {
	if report.FinishedAt.IsZero() {
		report.FinishedAt = time.Now().UTC()
	}
	_ = service.control.RecordRun(ctx, report.toRun("failed", cause.Error()))
	return cause
}

func jsonString(value any) string {
	encoded, err := json.Marshal(value)
	if err != nil {
		return "{}"
	}
	return string(encoded)
}
