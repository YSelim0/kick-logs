package migrations

const clickHouseMigrationTableSQL = `
CREATE TABLE IF NOT EXISTS schema_migrations (
	version UInt32,
	name String,
	applied_at DateTime64(3, 'UTC')
)
ENGINE = ReplacingMergeTree(applied_at)
ORDER BY version;`

func ClickHouseMigrations() []ClickHouseMigration {
	return []ClickHouseMigration{
		{
			Version: 1,
			Name:    "create_event_and_message_tables",
			Statements: []string{
				`CREATE TABLE IF NOT EXISTS chat_messages (
					id Int64,
					kick_message_id String,
					channel_id Nullable(Int64),
					channel_kick_id Nullable(Int64),
					channel_chatroom_id Nullable(Int64),
					channel_slug String,
					channel_slug_lower String,
					channel_display_name String,
					channel_display_name_lower String,
					channel_profile_image_url Nullable(String),
					channel_banner_image_url Nullable(String),
					channel_public_url String,
					sender_id Nullable(Int64),
					sender_kick_id Nullable(Int64),
					sender_username String,
					sender_username_lower String,
					sender_slug String,
					sender_slug_lower String,
					sender_display_color Nullable(String),
					sender_profile_image_url Nullable(String),
					sender_public_url String,
					sender_badges_json String DEFAULT '[]',
					message_type LowCardinality(String),
					content String,
					content_lower String,
					emote_count UInt16,
					emote_ids Array(String),
					emote_names Array(String),
					emote_tokens Array(String),
					emote_image_urls Array(String),
					reply_to_sender Nullable(String),
					reply_to_sender_lower Nullable(String),
					reply_to_content Nullable(String),
					reply_to_message_id Nullable(String),
					thread_parent_id Nullable(String),
					reply_metadata_json String DEFAULT '{}',
					raw_payload_json String,
					message_created_at DateTime64(3, 'UTC'),
					ingested_at DateTime64(3, 'UTC'),
					is_deleted UInt8 DEFAULT 0
				)
				ENGINE = ReplacingMergeTree(ingested_at)
				PARTITION BY toYYYYMM(message_created_at)
				ORDER BY (message_created_at, id, kick_message_id);`,
				`CREATE TABLE IF NOT EXISTS raw_kick_events (
					id String,
					channel_slug String,
					event_type LowCardinality(String),
					event_name String,
					payload_json String,
					status LowCardinality(String),
					received_at DateTime64(3, 'UTC'),
					processed_at Nullable(DateTime64(3, 'UTC')),
					error_message Nullable(String)
				)
				ENGINE = MergeTree
				PARTITION BY toYYYYMM(received_at)
				ORDER BY (received_at, id);`,
				`CREATE TABLE IF NOT EXISTS raw_event_attempts (
					id String,
					raw_event_id String,
					attempt UInt16,
					status LowCardinality(String),
					error_message Nullable(String),
					started_at DateTime64(3, 'UTC'),
					finished_at Nullable(DateTime64(3, 'UTC'))
				)
				ENGINE = MergeTree
				PARTITION BY toYYYYMM(started_at)
				ORDER BY (started_at, raw_event_id, attempt);`,
			},
		},
		{
			Version: 2,
			Name:    "add_message_response_snapshot_columns",
			Statements: []string{
				`ALTER TABLE chat_messages ADD COLUMN IF NOT EXISTS channel_id Nullable(Int64);`,
				`ALTER TABLE chat_messages ADD COLUMN IF NOT EXISTS sender_id Nullable(Int64);`,
				`ALTER TABLE chat_messages ADD COLUMN IF NOT EXISTS sender_badges_json String DEFAULT '[]';`,
				`ALTER TABLE chat_messages ADD COLUMN IF NOT EXISTS reply_metadata_json String DEFAULT '{}';`,
				`ALTER TABLE chat_messages ADD COLUMN IF NOT EXISTS channel_banner_image_url Nullable(String);`,
			},
		},
		{
			Version: 3,
			Name:    "add_raw_event_message_metadata",
			Statements: []string{
				`ALTER TABLE raw_kick_events ADD COLUMN IF NOT EXISTS kick_message_id Nullable(String);`,
				`ALTER TABLE raw_kick_events ADD COLUMN IF NOT EXISTS chatroom_id Nullable(Int64);`,
				`ALTER TABLE raw_kick_events ADD COLUMN IF NOT EXISTS channel_id Nullable(Int64);`,
			},
		},
		{
			Version: 4,
			Name:    "add_raw_event_metadata_json",
			Statements: []string{
				`ALTER TABLE raw_kick_events ADD COLUMN IF NOT EXISTS metadata_json String DEFAULT '{}';`,
			},
		},
		{
			Version: 5,
			Name:    "create_channel_subscription_periods",
			Statements: []string{
				`CREATE TABLE IF NOT EXISTS channel_subscription_periods (
					id String,
					event_message_id String,
					event_type LowCardinality(String),
					followed_channel_id Int64,
					broadcaster_user_id Int64,
					channel_slug String,
					channel_display_name String,
					subscriber_kick_user_id Int64,
					subscriber_username String,
					subscriber_slug String,
					subscriber_profile_image_url String DEFAULT '',
					gifter_kick_user_id Nullable(Int64),
					gifter_username Nullable(String),
					gifter_slug Nullable(String),
					gifter_profile_image_url Nullable(String),
					is_gift UInt8 DEFAULT 0,
					started_at DateTime64(3, 'UTC'),
					expires_at DateTime64(3, 'UTC'),
					raw_payload_json String DEFAULT '{}',
					ingested_at DateTime64(3, 'UTC')
				)
				ENGINE = ReplacingMergeTree(ingested_at)
				PARTITION BY toYYYYMM(started_at)
				ORDER BY id;`,
			},
		},
	}
}
