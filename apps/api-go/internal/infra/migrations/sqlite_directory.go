package migrations

func directorySQLiteMigration() SQLiteMigration {
	return SQLiteMigration{
		Version: 9,
		Name:    "index_identity_directory_prefixes",
		Statements: []string{
			`CREATE INDEX idx_sender_profiles_directory_name ON sender_profiles (lower(replace(trim(username), '_', '-')), id);`,
			`CREATE INDEX idx_sender_profiles_directory_slug ON sender_profiles (lower(replace(trim(slug), '_', '-')), id);`,
			`CREATE INDEX idx_followed_channels_directory_name ON followed_channels (lower(replace(trim(display_name), '_', '-')), id);`,
			`CREATE INDEX idx_followed_channels_directory_slug ON followed_channels (lower(replace(trim(slug), '_', '-')), id);`,
		},
	}
}
