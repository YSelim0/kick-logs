package migrations

const sqliteMigrationTableSQL = `
CREATE TABLE IF NOT EXISTS schema_migrations (
	version INTEGER PRIMARY KEY,
	name TEXT NOT NULL,
	applied_at TEXT NOT NULL
);`

func SQLiteMigrations() []SQLiteMigration {
	return []SQLiteMigration{
		{
			Version: 1,
			Name:    "create_control_plane_tables",
			Statements: []string{
				`CREATE TABLE IF NOT EXISTS admin_users (
					id INTEGER PRIMARY KEY AUTOINCREMENT,
					email TEXT NOT NULL UNIQUE,
					password_hash TEXT NOT NULL,
					role TEXT NOT NULL CHECK (role IN ('admin', 'super_admin')),
					is_active INTEGER NOT NULL DEFAULT 1,
					created_at TEXT NOT NULL,
					updated_at TEXT NOT NULL
				);`,
				`CREATE INDEX IF NOT EXISTS idx_admin_users_active ON admin_users (is_active);`,
				`CREATE TABLE IF NOT EXISTS followed_channels (
					id INTEGER PRIMARY KEY AUTOINCREMENT,
					kick_channel_id INTEGER,
					kick_chatroom_id INTEGER,
					slug TEXT NOT NULL UNIQUE,
					display_name TEXT NOT NULL,
					profile_image_url TEXT NOT NULL DEFAULT '',
					banner_image_url TEXT NOT NULL DEFAULT '',
					is_enabled INTEGER NOT NULL DEFAULT 1,
					raw_payload_json TEXT NOT NULL DEFAULT '{}',
					created_at TEXT NOT NULL,
					updated_at TEXT NOT NULL,
					last_resolved_at TEXT NOT NULL DEFAULT '',
					last_message_at TEXT NOT NULL DEFAULT '',
					last_listener_error TEXT NOT NULL DEFAULT ''
				);`,
				`CREATE UNIQUE INDEX IF NOT EXISTS idx_followed_channels_kick_channel_id ON followed_channels (kick_channel_id) WHERE kick_channel_id IS NOT NULL;`,
				`CREATE UNIQUE INDEX IF NOT EXISTS idx_followed_channels_kick_chatroom_id ON followed_channels (kick_chatroom_id) WHERE kick_chatroom_id IS NOT NULL;`,
				`CREATE INDEX IF NOT EXISTS idx_followed_channels_enabled ON followed_channels (is_enabled, slug);`,
				`CREATE TABLE IF NOT EXISTS sender_profiles (
					id INTEGER PRIMARY KEY AUTOINCREMENT,
					kick_user_id INTEGER NOT NULL UNIQUE,
					username TEXT NOT NULL,
					slug TEXT NOT NULL UNIQUE,
					profile_image_url TEXT NOT NULL DEFAULT '',
					last_seen_color TEXT NOT NULL DEFAULT '',
					raw_profile_payload_json TEXT NOT NULL DEFAULT '{}',
					created_at TEXT NOT NULL,
					updated_at TEXT NOT NULL,
					last_seen_at TEXT NOT NULL DEFAULT ''
				);`,
				`CREATE INDEX IF NOT EXISTS idx_sender_profiles_username ON sender_profiles (username);`,
				`CREATE TABLE IF NOT EXISTS retention_settings (
					id INTEGER PRIMARY KEY CHECK (id = 1),
					message_retention_days INTEGER CHECK (message_retention_days IS NULL OR message_retention_days IN (30, 90)),
					raw_event_retention_days INTEGER CHECK (raw_event_retention_days IS NULL OR raw_event_retention_days IN (30, 90)),
					created_at TEXT NOT NULL,
					updated_at TEXT NOT NULL
				);`,
				`CREATE TABLE IF NOT EXISTS worker_heartbeats (
					service_name TEXT PRIMARY KEY,
					last_seen_at TEXT NOT NULL,
					metadata_json TEXT NOT NULL DEFAULT '{}',
					created_at TEXT NOT NULL,
					updated_at TEXT NOT NULL
				);`,
				`CREATE TABLE IF NOT EXISTS data_migrations (
					version INTEGER PRIMARY KEY,
					name TEXT NOT NULL,
					applied_at TEXT NOT NULL
				);`,
			},
		},
		{
			Version: 2,
			Name:    "create_data_migration_runs",
			Statements: []string{
				`CREATE TABLE IF NOT EXISTS data_migration_runs (
					run_id TEXT PRIMARY KEY,
					name TEXT NOT NULL,
					mode TEXT NOT NULL,
					status TEXT NOT NULL,
					source_counts_json TEXT NOT NULL DEFAULT '{}',
					destination_counts_json TEXT NOT NULL DEFAULT '{}',
					validation_json TEXT NOT NULL DEFAULT '{}',
					error_message TEXT NOT NULL DEFAULT '',
					started_at TEXT NOT NULL,
					finished_at TEXT NOT NULL
				);`,
				`CREATE INDEX IF NOT EXISTS idx_data_migration_runs_started_at ON data_migration_runs (started_at);`,
			},
		},
		{
			Version: 3,
			Name:    "create_raw_event_claims",
			Statements: []string{
				`CREATE TABLE IF NOT EXISTS raw_event_claims (
					raw_event_id TEXT PRIMARY KEY,
					worker_id TEXT NOT NULL,
					status TEXT NOT NULL CHECK (status IN ('claimed', 'released', 'completed')),
					lease_expires_at TEXT NOT NULL DEFAULT '',
					claimed_at TEXT NOT NULL,
					completed_at TEXT NOT NULL DEFAULT '',
					updated_at TEXT NOT NULL
				);`,
				`CREATE INDEX IF NOT EXISTS idx_raw_event_claims_status_lease ON raw_event_claims (status, lease_expires_at);`,
			},
		},
		{
			Version: 4,
			Name:    "create_raw_event_queue",
			Statements: []string{
				`CREATE TABLE IF NOT EXISTS raw_event_queue (
					raw_event_id TEXT PRIMARY KEY,
					channel_id INTEGER NOT NULL DEFAULT 0,
					chatroom_id INTEGER NOT NULL DEFAULT 0,
					channel_slug TEXT NOT NULL DEFAULT '',
					kick_message_id TEXT NOT NULL DEFAULT '',
					status TEXT NOT NULL CHECK (status IN ('pending', 'claimed', 'processed', 'failed')),
					attempts INTEGER NOT NULL DEFAULT 0,
					claimed_by TEXT NOT NULL DEFAULT '',
					claimed_at TEXT NOT NULL DEFAULT '',
					enqueued_at TEXT NOT NULL,
					last_error TEXT NOT NULL DEFAULT '',
					updated_at TEXT NOT NULL
				);`,
				`CREATE INDEX IF NOT EXISTS idx_raw_event_queue_pending ON raw_event_queue (status, enqueued_at);`,
				`CREATE INDEX IF NOT EXISTS idx_raw_event_queue_claimed ON raw_event_queue (status, claimed_at);`,
			},
		},
		{
			Version: 5,
			Name:    "add_broadcaster_user_id_to_followed_channels",
			Statements: []string{
				`ALTER TABLE followed_channels ADD COLUMN broadcaster_user_id INTEGER;`,
			},
		},
		{
			Version: 6,
			Name:    "create_kick_webhook_events",
			Statements: []string{
				`CREATE TABLE IF NOT EXISTS kick_webhook_events (
					message_id TEXT PRIMARY KEY,
					subscription_id TEXT NOT NULL DEFAULT '',
					event_type TEXT NOT NULL DEFAULT '',
					event_version TEXT NOT NULL DEFAULT '',
					raw_payload_json TEXT NOT NULL DEFAULT '{}',
					status TEXT NOT NULL CHECK (status IN ('pending', 'processed', 'failed', 'ignored')) DEFAULT 'pending',
					attempts INTEGER NOT NULL DEFAULT 0,
					received_at TEXT NOT NULL,
					processed_at TEXT NOT NULL DEFAULT '',
					error_message TEXT NOT NULL DEFAULT ''
				);`,
				`CREATE INDEX IF NOT EXISTS idx_kick_webhook_events_pending ON kick_webhook_events (status, received_at) WHERE status = 'pending';`,
			},
		},
		{
			Version: 7,
			Name:    "create_kick_event_subscriptions",
			Statements: []string{
				`CREATE TABLE IF NOT EXISTS kick_event_subscriptions (
					id INTEGER PRIMARY KEY AUTOINCREMENT,
					followed_channel_id INTEGER NOT NULL,
					broadcaster_user_id INTEGER NOT NULL DEFAULT 0,
					event_type TEXT NOT NULL DEFAULT '',
					event_version TEXT NOT NULL DEFAULT 'v1',
					method TEXT NOT NULL DEFAULT 'webhook',
					kick_subscription_id TEXT NOT NULL DEFAULT '',
					status TEXT NOT NULL CHECK (status IN ('active', 'deleted', 'error')) DEFAULT 'active',
					latest_sync_error TEXT NOT NULL DEFAULT '',
					created_at TEXT NOT NULL,
					updated_at TEXT NOT NULL,
					synced_at TEXT NOT NULL DEFAULT '',
					UNIQUE (followed_channel_id, event_type, event_version, method)
				);`,
				`CREATE INDEX IF NOT EXISTS idx_kick_event_subscriptions_channel ON kick_event_subscriptions (followed_channel_id, status);`,
			},
		},
		{
			Version: 8,
			Name:    "prune_processed_raw_event_queue_rows",
			Statements: []string{
				`DELETE FROM raw_event_queue WHERE status = 'processed';`,
			},
		},
	}
}
