package webhookprocessor

// Kick subscription event payload shapes.
// Field names are best-effort based on Kick API conventions;
// adjust if the real payloads differ.

type subscriptionPayload struct {
	Broadcaster broadcasterPayload `json:"broadcaster"`
	Subscriber  userPayload        `json:"subscriber"`
	CreatedAt   string             `json:"created_at"`
	ExpiresAt   string             `json:"expires_at"`
}

type giftPayload struct {
	Broadcaster broadcasterPayload `json:"broadcaster"`
	Gifter      *gifterPayload     `json:"gifter"`
	Recipients  []userPayload      `json:"recipients"`
	Giftees     []userPayload      `json:"giftees"`
	CreatedAt   string             `json:"created_at"`
	ExpiresAt   string             `json:"expires_at"`
}

type broadcasterPayload struct {
	UserID      int64  `json:"user_id"`
	Username    string `json:"username"`
	ChannelSlug string `json:"channel_slug"`
}

type userPayload struct {
	UserID         int64  `json:"user_id"`
	Username       string `json:"username"`
	ChannelSlug    string `json:"channel_slug"`
	ProfilePicture string `json:"profile_picture"`
}

type gifterPayload struct {
	UserID         int64  `json:"user_id"`
	Username       string `json:"username"`
	ChannelSlug    string `json:"channel_slug"`
	ProfilePicture string `json:"profile_picture"`
	IsAnonymous    bool   `json:"is_anonymous"`
}

type broadcasterEnvelope struct {
	Broadcaster struct {
		UserID int64 `json:"user_id"`
	} `json:"broadcaster"`
}
