package ports

import (
	"context"

	"github.com/YSelim0/kick-logs/apps/api-go/internal/domain"
)

// SenderMessageNotifier delivers an out-of-band alert when a watched sender
// posts a chat message. Implementations must be safe to call from the
// ingestion hot path: a notification failure is a delivery problem for the
// notifier, not a reason to fail chat message processing.
type SenderMessageNotifier interface {
	NotifySenderMessage(ctx context.Context, message domain.ChatMessage) error
}
