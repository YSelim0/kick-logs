package kick

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/YSelim0/kick-logs/apps/api-go/internal/domain"
)

type WebChannelResolver struct {
	client  *http.Client
	baseURL string
}

func NewWebChannelResolver() *WebChannelResolver {
	return &WebChannelResolver{
		client:  &http.Client{Timeout: 15 * time.Second},
		baseURL: "https://kick.com",
	}
}

func (resolver *WebChannelResolver) ResolveChannel(ctx context.Context, slug string) (domain.FollowedChannel, error) {
	slug = strings.TrimSpace(slug)
	if slug == "" {
		return domain.FollowedChannel{}, fmt.Errorf("slug is required")
	}

	url := fmt.Sprintf("%s/api/v2/channels/%s", strings.TrimRight(resolver.baseURL, "/"), slug)
	request, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return domain.FollowedChannel{}, fmt.Errorf("build Kick channel request: %w", err)
	}
	request.Header.Set("accept", "application/json")
	request.Header.Set("user-agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/124.0 Safari/537.36")

	response, err := resolver.client.Do(request)
	if err != nil {
		return domain.FollowedChannel{}, fmt.Errorf("resolve Kick channel: %w", err)
	}
	defer response.Body.Close()

	if response.StatusCode == http.StatusNotFound {
		return domain.FollowedChannel{}, fmt.Errorf("Kick channel not found")
	}
	if response.StatusCode < 200 || response.StatusCode >= 300 {
		return domain.FollowedChannel{}, fmt.Errorf("Kick channel returned status %d", response.StatusCode)
	}

	rawPayload, err := io.ReadAll(response.Body)
	if err != nil {
		return domain.FollowedChannel{}, fmt.Errorf("read Kick channel response: %w", err)
	}

	var payload channelPayload
	if err := json.Unmarshal(rawPayload, &payload); err != nil {
		return domain.FollowedChannel{}, fmt.Errorf("decode Kick channel response: %w", err)
	}

	resolvedSlug := strings.TrimSpace(payload.Slug)
	if resolvedSlug == "" {
		resolvedSlug = slug
	}
	displayName := strings.TrimSpace(payload.User.Username)
	if displayName == "" {
		displayName = resolvedSlug
	}

	return domain.FollowedChannel{
		KickChannelID:     payload.ID,
		KickChatroomID:    payload.Chatroom.ID,
		BroadcasterUserID: payload.broadcasterUserID(),
		Slug:              strings.ToLower(resolvedSlug),
		DisplayName:       displayName,
		ProfileImageURL:   payload.profileImageURL(),
		BannerImageURL:    payload.bannerImageURL(),
		IsEnabled:         true,
		RawPayloadJSON:    string(rawPayload),
	}, nil
}

type channelPayload struct {
	ID          int64           `json:"id"`
	UserID      int64           `json:"user_id"`
	Slug        string          `json:"slug"`
	User        userPayload     `json:"user"`
	Chatroom    chatroomPayload `json:"chatroom"`
	ProfilePic  string          `json:"profilepic"`
	ProfilePic2 string          `json:"profile_pic"`
	ProfileURL  string          `json:"profile_image_url"`
	BannerImage any             `json:"banner_image"`
	BannerURL   string          `json:"banner_image_url"`
	Banner      string          `json:"banner"`
}

type userPayload struct {
	ID          int64  `json:"id"`
	Username    string `json:"username"`
	ProfilePic  string `json:"profilepic"`
	ProfilePic2 string `json:"profile_pic"`
	ProfileURL  string `json:"profile_image_url"`
}

type chatroomPayload struct {
	ID int64 `json:"id"`
}

func (payload channelPayload) broadcasterUserID() int64 {
	if payload.UserID != 0 {
		return payload.UserID
	}
	return payload.User.ID
}

func (payload channelPayload) profileImageURL() string {
	return firstNonEmptyString(
		payload.ProfilePic,
		payload.ProfilePic2,
		payload.ProfileURL,
		payload.User.ProfilePic,
		payload.User.ProfilePic2,
		payload.User.ProfileURL,
	)
}

func (payload channelPayload) bannerImageURL() string {
	if value, ok := payload.BannerImage.(string); ok && strings.TrimSpace(value) != "" {
		return strings.TrimSpace(value)
	}
	if value, ok := payload.BannerImage.(map[string]any); ok {
		if url, ok := value["url"].(string); ok && strings.TrimSpace(url) != "" {
			return strings.TrimSpace(url)
		}
	}
	return firstNonEmptyString(payload.BannerURL, payload.Banner)
}

func firstNonEmptyString(values ...string) string {
	for _, value := range values {
		if trimmed := strings.TrimSpace(value); trimmed != "" {
			return trimmed
		}
	}
	return ""
}
