package messageimport

import "encoding/json"

// ExportFile is the JSON message export shape produced by the app's own
// message export/search paths.
type ExportFile struct {
	Items     []ExportItem `json:"items"`
	Count     int          `json:"count"`
	MaxRows   int          `json:"max_rows"`
	Truncated bool         `json:"truncated"`
}

type ExportItem struct {
	ID                     int64           `json:"id"`
	KickMessageID          string          `json:"kick_message_id"`
	ChatroomID             int64           `json:"chatroom_id"`
	Content                string          `json:"content"`
	MessageType            string          `json:"message_type"`
	SenderUsernameSnapshot string          `json:"sender_username_snapshot"`
	SenderSlugSnapshot     string          `json:"sender_slug_snapshot"`
	SenderColorSnapshot    string          `json:"sender_color_snapshot"`
	SenderBadges           json.RawMessage `json:"sender_badges"`
	Emotes                 []ExportEmote   `json:"emotes"`
	ReplyMetadata          json.RawMessage `json:"reply_metadata"`
	ThreadParentID         *string         `json:"thread_parent_id"`
	MessageCreatedAt       string          `json:"message_created_at"`
	IngestedAt             string          `json:"ingested_at"`
	Sender                 ExportSender    `json:"sender"`
	Channel                ExportChannel   `json:"channel"`
}

type ExportEmote struct {
	ID       string `json:"id"`
	Name     string `json:"name"`
	Token    string `json:"token"`
	ImageURL string `json:"image_url"`
}

type ExportSender struct {
	ID              int64   `json:"id"`
	KickUserID      int64   `json:"kick_user_id"`
	Username        string  `json:"username"`
	Slug            string  `json:"slug"`
	ProfileImageURL *string `json:"profile_image_url"`
}

type ExportChannel struct {
	ID              int64   `json:"id"`
	Slug            string  `json:"slug"`
	DisplayName     string  `json:"display_name"`
	ProfileImageURL *string `json:"profile_image_url"`
	BannerImageURL  *string `json:"banner_image_url"`
}

type replyMetadata struct {
	OriginalMessage struct {
		Content string `json:"content"`
		ID      string `json:"id"`
	} `json:"original_message"`
	OriginalSender struct {
		Username string `json:"username"`
	} `json:"original_sender"`
}
