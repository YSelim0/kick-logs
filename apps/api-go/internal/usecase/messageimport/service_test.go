package messageimport_test

import (
	"context"
	"errors"
	"testing"

	"github.com/YSelim0/kick-logs/apps/api-go/internal/domain"
	"github.com/YSelim0/kick-logs/apps/api-go/internal/usecase/messageimport"
)

type fakeMessageRepository struct {
	existing  map[string]bool
	inserted  []domain.ChatMessage
	insertErr error
}

func (repository *fakeMessageRepository) Insert(ctx context.Context, message domain.ChatMessage) error {
	return repository.InsertMessagesBatch(ctx, []domain.ChatMessage{message})
}

func (repository *fakeMessageRepository) InsertMessagesBatch(_ context.Context, messages []domain.ChatMessage) error {
	if repository.insertErr != nil {
		return repository.insertErr
	}
	repository.inserted = append(repository.inserted, messages...)
	return nil
}

func (repository *fakeMessageRepository) ExistsByKickMessageID(_ context.Context, kickMessageID string) (bool, error) {
	return repository.existing[kickMessageID], nil
}

func (repository *fakeMessageRepository) ExistingKickMessageIDs(
	_ context.Context,
	kickMessageIDs []string,
) (map[string]bool, error) {
	result := make(map[string]bool)
	for _, id := range kickMessageIDs {
		if repository.existing[id] {
			result[id] = true
		}
	}
	return result, nil
}

func (repository *fakeMessageRepository) Search(
	_ context.Context,
	_ domain.MessageSearchFilter,
) ([]domain.ChatMessage, error) {
	return nil, nil
}

const exportPayload = `{
  "items": [
    {
      "id": 1,
      "kick_message_id": "msg-new",
      "chatroom_id": 7552533,
      "content": "merhaba [emote:123:selam]",
      "message_type": "reply",
      "sender_color_snapshot": "#00CCB3",
      "sender_badges": [],
      "emotes": [{"id": "123", "name": "selam", "token": "[emote:123:selam]", "image_url": "https://files.kick.com/emotes/123/fullsize"}],
      "reply_metadata": {
        "message_ref": "1788151087288",
        "original_message": {"content": "onceki mesaj", "id": "orig-1"},
        "original_sender": {"id": 26302661, "username": "Savas353"}
      },
      "thread_parent_id": "thread-1",
      "message_created_at": "2026-08-31T04:38:07Z",
      "ingested_at": "2026-08-31T04:38:10Z",
      "sender": {"id": 74431315, "kick_user_id": 74431315, "username": "prenses_elif", "slug": "prenses-elif", "profile_image_url": null},
      "channel": {"id": 10, "slug": "sinasi", "display_name": "Sinasi", "profile_image_url": null, "banner_image_url": null}
    },
    {
      "id": 2,
      "kick_message_id": "msg-existing",
      "content": "zaten var",
      "message_type": "message",
      "message_created_at": "2026-08-31T04:39:07Z",
      "sender": {"id": 1, "kick_user_id": 1, "username": "someone", "slug": "someone"},
      "channel": {"id": 10, "slug": "sinasi", "display_name": "Sinasi"}
    },
    {
      "id": 3,
      "kick_message_id": "msg-new",
      "content": "dosya ici tekrar",
      "message_type": "message",
      "message_created_at": "2026-08-31T04:40:07Z",
      "sender": {"id": 1, "kick_user_id": 1, "username": "someone", "slug": "someone"},
      "channel": {"id": 10, "slug": "sinasi", "display_name": "Sinasi"}
    },
    {
      "id": 4,
      "kick_message_id": "",
      "content": "kick_message_id yok",
      "message_created_at": "2026-08-31T04:41:07Z",
      "sender": {"id": 1, "kick_user_id": 1, "username": "someone", "slug": "someone"},
      "channel": {"id": 10, "slug": "sinasi", "display_name": "Sinasi"}
    },
    {
      "id": 5,
      "kick_message_id": "msg-bad-time",
      "content": "tarih bozuk",
      "message_created_at": "not-a-time",
      "sender": {"id": 1, "kick_user_id": 1, "username": "someone", "slug": "someone"},
      "channel": {"id": 10, "slug": "sinasi", "display_name": "Sinasi"}
    }
  ],
  "count": 5,
  "max_rows": 1000,
  "truncated": false
}`

func newService(existing ...string) (*messageimport.Service, *fakeMessageRepository) {
	repository := &fakeMessageRepository{existing: map[string]bool{}}
	for _, id := range existing {
		repository.existing[id] = true
	}
	return messageimport.NewService(repository, 0), repository
}

func TestPreviewCountsEachOutcome(t *testing.T) {
	service, repository := newService("msg-existing")

	preview, err := service.Preview(context.Background(), []byte(exportPayload), 0)
	if err != nil {
		t.Fatalf("preview: %v", err)
	}

	if preview.RecordsRead != 5 {
		t.Fatalf("records read = %d, want 5", preview.RecordsRead)
	}
	if preview.ToInsert != 1 {
		t.Fatalf("to insert = %d, want 1", preview.ToInsert)
	}
	if preview.AlreadyExists != 1 {
		t.Fatalf("already exists = %d, want 1", preview.AlreadyExists)
	}
	if preview.DuplicateInFile != 1 {
		t.Fatalf("duplicate in file = %d, want 1", preview.DuplicateInFile)
	}
	if preview.Invalid != 2 {
		t.Fatalf("invalid = %d, want 2", preview.Invalid)
	}
	if len(preview.InvalidReasons) != 2 {
		t.Fatalf("invalid reasons = %d, want 2", len(preview.InvalidReasons))
	}
	if !preview.CanExecute {
		t.Fatalf("preview should be executable when there is something to insert")
	}
	if preview.ConfirmationText != messageimport.ConfirmationText {
		t.Fatalf("confirmation text = %q", preview.ConfirmationText)
	}
	if len(repository.inserted) != 0 {
		t.Fatalf("preview must not write, inserted %d rows", len(repository.inserted))
	}
}

func TestPreviewRespectsLimit(t *testing.T) {
	service, _ := newService()

	preview, err := service.Preview(context.Background(), []byte(exportPayload), 2)
	if err != nil {
		t.Fatalf("preview: %v", err)
	}
	if preview.RecordsRead != 2 {
		t.Fatalf("records read = %d, want 2", preview.RecordsRead)
	}
	if preview.TotalInFile != 5 {
		t.Fatalf("total in file = %d, want 5", preview.TotalInFile)
	}
	if preview.ToInsert != 2 {
		t.Fatalf("to insert = %d, want 2", preview.ToInsert)
	}
}

func TestPreviewRejectsFileOverMaxRows(t *testing.T) {
	repository := &fakeMessageRepository{existing: map[string]bool{}}
	service := messageimport.NewService(repository, 2)

	_, err := service.Preview(context.Background(), []byte(exportPayload), 0)
	if !errors.Is(err, messageimport.ErrValidation) {
		t.Fatalf("err = %v, want validation error", err)
	}
}

func TestPreviewRejectsNonExportJSON(t *testing.T) {
	service, _ := newService()

	if _, err := service.Preview(context.Background(), []byte(`{"foo":1}`), 0); !errors.Is(err, messageimport.ErrValidation) {
		t.Fatalf("err = %v, want validation error for missing items", err)
	}
	if _, err := service.Preview(context.Background(), []byte(`not json`), 0); !errors.Is(err, messageimport.ErrValidation) {
		t.Fatalf("err = %v, want validation error for malformed json", err)
	}
}

func TestConfirmRequiresExactConfirmationText(t *testing.T) {
	service, repository := newService("msg-existing")

	_, err := service.Confirm(context.Background(), []byte(exportPayload), 0, "import messages")
	if !errors.Is(err, messageimport.ErrConfirmation) {
		t.Fatalf("err = %v, want confirmation error", err)
	}
	if len(repository.inserted) != 0 {
		t.Fatalf("confirm must not write on mismatched text, inserted %d rows", len(repository.inserted))
	}
}

func TestConfirmInsertsOnlyNewMessages(t *testing.T) {
	service, repository := newService("msg-existing")

	result, err := service.Confirm(
		context.Background(),
		[]byte(exportPayload),
		0,
		messageimport.ConfirmationText,
	)
	if err != nil {
		t.Fatalf("confirm: %v", err)
	}
	if result.Written != 1 {
		t.Fatalf("written = %d, want 1", result.Written)
	}
	if len(repository.inserted) != 1 {
		t.Fatalf("inserted %d rows, want 1", len(repository.inserted))
	}

	inserted := repository.inserted[0]
	if inserted.KickMessageID != "msg-new" {
		t.Fatalf("inserted kick message id = %q, want msg-new", inserted.KickMessageID)
	}
	if inserted.ID != messageimport.DeterministicMessageID("msg-new") {
		t.Fatalf("inserted id = %d, want deterministic id", inserted.ID)
	}
	if inserted.ChannelSlug != "sinasi" || inserted.ChannelChatroomID != 7552533 {
		t.Fatalf("channel fields not preserved: %+v", inserted)
	}
	if inserted.SenderUsername != "prenses_elif" || inserted.SenderSlug != "prenses-elif" {
		t.Fatalf("sender fields not preserved: %+v", inserted)
	}
	if inserted.SenderKickID != 74431315 {
		t.Fatalf("sender kick id = %d", inserted.SenderKickID)
	}
	if len(inserted.Emotes) != 1 || inserted.Emotes[0].Name != "selam" {
		t.Fatalf("emotes not preserved: %+v", inserted.Emotes)
	}
	if inserted.ReplyToSender != "Savas353" || inserted.ReplyToContent != "onceki mesaj" {
		t.Fatalf("reply fields not derived: %+v", inserted)
	}
	if inserted.ReplyToMessageID != "orig-1" || inserted.ThreadParentID != "thread-1" {
		t.Fatalf("reply/thread ids not preserved: %+v", inserted)
	}
	if inserted.ReplyMetadataJSON == "" || inserted.ReplyMetadataJSON == "{}" {
		t.Fatalf("reply metadata json not preserved: %q", inserted.ReplyMetadataJSON)
	}
	if inserted.MessageCreatedAt.Format("2006-01-02T15:04:05Z") != "2026-08-31T04:38:07Z" {
		t.Fatalf("message_created_at not preserved: %v", inserted.MessageCreatedAt)
	}
	if inserted.IngestedAt.Format("2006-01-02T15:04:05Z") != "2026-08-31T04:38:10Z" {
		t.Fatalf("ingested_at not preserved: %v", inserted.IngestedAt)
	}
}

func TestConfirmRefusesWhenNothingToInsert(t *testing.T) {
	service, repository := newService("msg-new", "msg-existing")

	_, err := service.Confirm(
		context.Background(),
		[]byte(exportPayload),
		0,
		messageimport.ConfirmationText,
	)
	if !errors.Is(err, messageimport.ErrCannotExecute) {
		t.Fatalf("err = %v, want cannot-execute error", err)
	}
	if len(repository.inserted) != 0 {
		t.Fatalf("nothing should be written, inserted %d rows", len(repository.inserted))
	}
}

func TestDeterministicMessageIDMatchesListenerHash(t *testing.T) {
	// Value taken from a real export row produced by the live pipeline, which
	// is what makes backfilled ids line up with live-ingested ones.
	const kickMessageID = "7c49c171-3381-47ba-abb3-49fe4a3ae43b"
	const want = int64(5558432954209888718)

	if got := messageimport.DeterministicMessageID(kickMessageID); got != want {
		t.Fatalf("deterministic id = %d, want %d", got, want)
	}
}
