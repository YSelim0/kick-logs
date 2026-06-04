package natsstream

import (
	"testing"
	"time"

	"github.com/YSelim0/kick-logs/apps/api-go/internal/config"
)

func TestConfigFromAppConfigUsesDefaults(t *testing.T) {
	cfg := ConfigFromAppConfig(config.Config{})

	if cfg.URL != "nats://127.0.0.1:4222" {
		t.Fatalf("URL = %q", cfg.URL)
	}
	if cfg.StreamName != "KICK_RAW_EVENTS" {
		t.Fatalf("StreamName = %q", cfg.StreamName)
	}
	if cfg.Subject != "kick.raw.chat" {
		t.Fatalf("Subject = %q", cfg.Subject)
	}
	if cfg.ConsumerName != "kick-raw-event-processor" {
		t.Fatalf("ConsumerName = %q", cfg.ConsumerName)
	}
	if cfg.AckWait != 60*time.Second {
		t.Fatalf("AckWait = %s", cfg.AckWait)
	}
	if cfg.FetchBatchSize != 500 {
		t.Fatalf("FetchBatchSize = %d", cfg.FetchBatchSize)
	}
	if cfg.FetchTimeout != 2*time.Second {
		t.Fatalf("FetchTimeout = %s", cfg.FetchTimeout)
	}
}

func TestConfigFromAppConfigParsesOverrides(t *testing.T) {
	cfg := ConfigFromAppConfig(config.Config{
		NATSURL:                         "nats://nats:4222",
		NATSRawEventStream:              "CUSTOM_STREAM",
		NATSRawEventSubject:             "custom.raw.chat",
		NATSRawEventConsumer:            "custom-consumer",
		NATSRawEventAckWaitSeconds:      90,
		NATSRawEventFetchBatchSize:      250,
		NATSRawEventFetchTimeoutSeconds: 5,
	})

	if cfg.URL != "nats://nats:4222" {
		t.Fatalf("URL = %q", cfg.URL)
	}
	if cfg.StreamName != "CUSTOM_STREAM" {
		t.Fatalf("StreamName = %q", cfg.StreamName)
	}
	if cfg.Subject != "custom.raw.chat" {
		t.Fatalf("Subject = %q", cfg.Subject)
	}
	if cfg.ConsumerName != "custom-consumer" {
		t.Fatalf("ConsumerName = %q", cfg.ConsumerName)
	}
	if cfg.AckWait != 90*time.Second {
		t.Fatalf("AckWait = %s", cfg.AckWait)
	}
	if cfg.FetchBatchSize != 250 {
		t.Fatalf("FetchBatchSize = %d", cfg.FetchBatchSize)
	}
	if cfg.FetchTimeout != 5*time.Second {
		t.Fatalf("FetchTimeout = %s", cfg.FetchTimeout)
	}
}

func TestStreamAndConsumerConfigPreserveDurabilityRules(t *testing.T) {
	client := &Client{cfg: NormalizeConfig(Config{})}

	stream := client.streamConfig()
	if stream.Storage.String() != "File" {
		t.Fatalf("Storage = %s", stream.Storage.String())
	}
	if stream.Retention.String() != "WorkQueue" {
		t.Fatalf("Retention = %s", stream.Retention.String())
	}
	if stream.Discard.String() != "DiscardNew" {
		t.Fatalf("Discard = %s", stream.Discard.String())
	}

	consumer := client.consumerConfig()
	if consumer.AckWait != 60*time.Second {
		t.Fatalf("AckWait = %s", consumer.AckWait)
	}
	if consumer.AckPolicy.String() != "AckExplicit" {
		t.Fatalf("AckPolicy = %s", consumer.AckPolicy.String())
	}
	if consumer.MaxDeliver != -1 {
		t.Fatalf("MaxDeliver = %d", consumer.MaxDeliver)
	}
}
