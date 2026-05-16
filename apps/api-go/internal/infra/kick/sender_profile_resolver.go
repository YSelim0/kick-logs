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

type WebSenderProfileResolver struct {
	client  *http.Client
	baseURL string
}

func NewWebSenderProfileResolver() *WebSenderProfileResolver {
	return &WebSenderProfileResolver{
		client:  &http.Client{Timeout: 15 * time.Second},
		baseURL: "https://kick.com/api/v2/channels",
	}
}

func (resolver *WebSenderProfileResolver) ResolveSender(ctx context.Context, slug string) (domain.SenderProfile, error) {
	slug = strings.ReplaceAll(strings.ToLower(strings.TrimSpace(slug)), "_", "-")
	if slug == "" {
		return domain.SenderProfile{}, fmt.Errorf("sender slug is required")
	}

	request, err := http.NewRequestWithContext(ctx, http.MethodGet, strings.TrimRight(resolver.baseURL, "/")+"/"+slug, nil)
	if err != nil {
		return domain.SenderProfile{}, fmt.Errorf("build Kick sender request: %w", err)
	}
	request.Header.Set("accept", "application/json")
	request.Header.Set("user-agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/124.0 Safari/537.36")

	response, err := resolver.client.Do(request)
	if err != nil {
		return domain.SenderProfile{}, fmt.Errorf("resolve Kick sender profile: %w", err)
	}
	defer response.Body.Close()
	if response.StatusCode < 200 || response.StatusCode >= 300 {
		return domain.SenderProfile{}, fmt.Errorf("Kick sender profile returned status %d", response.StatusCode)
	}

	rawPayload, err := io.ReadAll(response.Body)
	if err != nil {
		return domain.SenderProfile{}, fmt.Errorf("read Kick sender response: %w", err)
	}
	var payload senderProfilePayload
	if err := json.Unmarshal(rawPayload, &payload); err != nil {
		return domain.SenderProfile{}, fmt.Errorf("decode Kick sender response: %w", err)
	}
	username := strings.TrimSpace(payload.User.Username)
	if username == "" {
		username = slug
	}
	resolvedSlug := strings.ReplaceAll(strings.ToLower(strings.TrimSpace(payload.Slug)), "_", "-")
	if resolvedSlug == "" {
		resolvedSlug = slug
	}
	return domain.SenderProfile{
		Username:              username,
		Slug:                  resolvedSlug,
		ProfileImageURL:       firstNonEmptyString(payload.User.ProfilePic, payload.User.ProfilePic2, payload.User.ProfileURL, payload.ProfilePic, payload.ProfilePic2, payload.ProfileURL),
		RawProfilePayloadJSON: string(rawPayload),
	}, nil
}

type senderProfilePayload struct {
	Slug        string      `json:"slug"`
	User        userPayload `json:"user"`
	ProfilePic  string      `json:"profilepic"`
	ProfilePic2 string      `json:"profile_pic"`
	ProfileURL  string      `json:"profile_image_url"`
}
