package schemas

type CreateUserRequestRequest struct {
	Type               string `json:"type"`
	Title              string `json:"title"`
	Message            string `json:"message"`
	ChannelSlug        string `json:"channel_slug"`
	ChannelDisplayName string `json:"channel_display_name"`
	Contact            string `json:"contact"`
	Website            string `json:"website"`
}

type CreateUserRequestResponse struct {
	RequestID string `json:"request_id"`
}

type UserRequestResponse struct {
	RequestID          string  `json:"request_id"`
	Type               string  `json:"type"`
	Title              string  `json:"title"`
	Message            string  `json:"message"`
	ChannelSlug        *string `json:"channel_slug"`
	ChannelDisplayName *string `json:"channel_display_name"`
	Contact            *string `json:"contact"`
	CurrentStatus      string  `json:"current_status"`
	IsArchived         bool    `json:"is_archived"`
	CreatedAt          string  `json:"created_at"`
	LatestEventAt      string  `json:"latest_event_at"`
}

type UserRequestEventResponse struct {
	EventID   string `json:"event_id"`
	RequestID string `json:"request_id"`
	EventType string `json:"event_type"`
	Status    string `json:"status"`
	Note      string `json:"note"`
	AdminID   int64  `json:"admin_id"`
	CreatedAt string `json:"created_at"`
}

type UserRequestsResponse struct {
	Items []UserRequestResponse `json:"items"`
	Count int                   `json:"count"`
}

type UserRequestDetailResponse struct {
	Request UserRequestResponse        `json:"request"`
	Events  []UserRequestEventResponse `json:"events"`
}

type UpdateUserRequestStatusRequest struct {
	Status string `json:"status"`
}

type AddUserRequestNoteRequest struct {
	Note string `json:"note"`
}
