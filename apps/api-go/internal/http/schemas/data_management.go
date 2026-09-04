package schemas

type RetentionSettingsRequest struct {
	MessageRetentionDays  *int `json:"message_retention_days"`
	RawEventRetentionDays *int `json:"raw_event_retention_days"`
}

type RetentionSettingsResponse struct {
	MessageRetentionDays  *int    `json:"message_retention_days"`
	RawEventRetentionDays *int    `json:"raw_event_retention_days"`
	UpdatedAt             *string `json:"updated_at"`
}

type DataManagementCountsResponse struct {
	Channels  int64 `json:"channels"`
	Senders   int64 `json:"senders"`
	Messages  int64 `json:"messages"`
	RawEvents int64 `json:"raw_events"`
}

type DataManagementTableResponse struct {
	TableName  string `json:"table_name"`
	TotalBytes int64  `json:"total_bytes"`
	RowCount   int64  `json:"row_count"`
}

type DataManagementSummaryResponse struct {
	Counts            DataManagementCountsResponse  `json:"counts"`
	DatabaseBytes     int64                         `json:"database_bytes"`
	Tables            []DataManagementTableResponse `json:"tables"`
	RetentionSettings RetentionSettingsResponse     `json:"retention_settings"`
}

type DataCleanupRequest struct {
	Target      string  `json:"target"`
	ChannelSlug *string `json:"channel_slug"`
	Sender      *string `json:"sender"`
}

type DataCleanupConfirmRequest struct {
	Target           string  `json:"target"`
	ChannelSlug      *string `json:"channel_slug"`
	Sender           *string `json:"sender"`
	ConfirmationText string  `json:"confirmation_text"`
}

type DataCleanupCountsResponse struct {
	Messages  int64 `json:"messages"`
	RawEvents int64 `json:"raw_events"`
	Total     int64 `json:"total"`
}

type DataCleanupPreviewResponse struct {
	Target           string                    `json:"target"`
	Affected         DataCleanupCountsResponse `json:"affected"`
	ConfirmationText string                    `json:"confirmation_text"`
	CanExecute       bool                      `json:"can_execute"`
	CutoffAt         *string                   `json:"cutoff_at"`
	ChannelSlug      *string                   `json:"channel_slug"`
	Sender           *string                   `json:"sender"`
	RetentionDays    *int                      `json:"retention_days"`
	Reason           *string                   `json:"reason"`
}

type DataCleanupResultResponse struct {
	Target           string                    `json:"target"`
	Deleted          DataCleanupCountsResponse `json:"deleted"`
	ConfirmationText string                    `json:"confirmation_text"`
	CutoffAt         *string                   `json:"cutoff_at"`
	ChannelSlug      *string                   `json:"channel_slug"`
	Sender           *string                   `json:"sender"`
	RetentionDays    *int                      `json:"retention_days"`
}

type MessageImportInvalidReasonResponse struct {
	Reason  string `json:"reason"`
	Count   int    `json:"count"`
	Example string `json:"example"`
}

type MessageImportPreviewResponse struct {
	TotalInFile       int                                  `json:"total_in_file"`
	RecordsRead       int                                  `json:"records_read"`
	Limit             int                                  `json:"limit"`
	ToInsert          int                                  `json:"to_insert"`
	AlreadyExists     int                                  `json:"already_exists"`
	DuplicateInFile   int                                  `json:"duplicate_in_file"`
	Invalid           int                                  `json:"invalid"`
	InvalidReasons    []MessageImportInvalidReasonResponse `json:"invalid_reasons"`
	SampleToInsertIDs []string                             `json:"sample_to_insert_ids"`
	ConfirmationText  string                               `json:"confirmation_text"`
	CanExecute        bool                                 `json:"can_execute"`
	Reason            *string                              `json:"reason"`
}

type MessageImportResultResponse struct {
	Written          int    `json:"written"`
	AlreadyExists    int    `json:"already_exists"`
	DuplicateInFile  int    `json:"duplicate_in_file"`
	Invalid          int    `json:"invalid"`
	ConfirmationText string `json:"confirmation_text"`
}
