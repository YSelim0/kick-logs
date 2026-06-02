package operations

import (
	"testing"

	"github.com/YSelim0/kick-logs/apps/api-go/internal/domain"
)

func TestApplyIngestionMetadataPreservesFieldsMissingFromLaterHeartbeats(t *testing.T) {
	var ingestion domain.IngestionHealth

	applyIngestionMetadata(`{"captured_raw_events":12,"recent_message_poll_captured":5,"recent_message_poll_errors":1,"write_drop_count":3}`, &ingestion)
	applyIngestionMetadata(`{"breaker_state":"closed"}`, &ingestion)

	if ingestion.CapturedRawEvents != 12 {
		t.Fatalf("captured raw events = %d, want 12", ingestion.CapturedRawEvents)
	}
	if ingestion.WriteDropCount != 3 {
		t.Fatalf("write drop count = %d, want 3", ingestion.WriteDropCount)
	}
	if ingestion.RecentMessagePollCaptured != 5 {
		t.Fatalf("recent poll captured = %d, want 5", ingestion.RecentMessagePollCaptured)
	}
	if ingestion.RecentMessagePollErrors != 1 {
		t.Fatalf("recent poll errors = %d, want 1", ingestion.RecentMessagePollErrors)
	}
	if ingestion.BreakerState != "closed" {
		t.Fatalf("breaker state = %q, want closed", ingestion.BreakerState)
	}
}
