package kick

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"sync"
	"time"

	"github.com/YSelim0/kick-logs/apps/api-go/internal/domain"
)

type EventSubscriptionClient struct {
	httpClient   *http.Client
	apiBaseURL   string
	oauthURL     string
	clientID     string
	clientSecret string

	mu          sync.Mutex
	accessToken string
	tokenExpiry time.Time
}

func NewEventSubscriptionClient(apiBaseURL, oauthURL, clientID, clientSecret string) *EventSubscriptionClient {
	return &EventSubscriptionClient{
		httpClient:   &http.Client{Timeout: 15 * time.Second},
		apiBaseURL:   strings.TrimRight(apiBaseURL, "/"),
		oauthURL:     oauthURL,
		clientID:     clientID,
		clientSecret: clientSecret,
	}
}

// ResolveBroadcasterUserID fetches the Kick broadcaster user ID for a channel slug
// using the Kick web API (kick.com/api/v2/channels/{slug}).
func (c *EventSubscriptionClient) ResolveBroadcasterUserID(ctx context.Context, slug string) (int64, error) {
	slug = strings.TrimSpace(slug)
	if slug == "" {
		return 0, fmt.Errorf("slug is required")
	}

	reqURL := fmt.Sprintf("https://kick.com/api/v2/channels/%s", slug)
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, reqURL, nil)
	if err != nil {
		return 0, fmt.Errorf("build channel request: %w", err)
	}
	req.Header.Set("Accept", "application/json")
	req.Header.Set("User-Agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/124.0 Safari/537.36")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return 0, fmt.Errorf("fetch channel for broadcaster user id: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusNotFound {
		return 0, fmt.Errorf("channel %q not found on Kick", slug)
	}
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return 0, fmt.Errorf("channel lookup returned status %d", resp.StatusCode)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return 0, fmt.Errorf("read channel response: %w", err)
	}

	var payload struct {
		UserID int64 `json:"user_id"`
		User   struct {
			ID int64 `json:"id"`
		} `json:"user"`
	}
	if err := json.Unmarshal(body, &payload); err != nil {
		return 0, fmt.Errorf("decode channel response: %w", err)
	}

	if payload.UserID != 0 {
		return payload.UserID, nil
	}
	if payload.User.ID != 0 {
		return payload.User.ID, nil
	}
	return 0, fmt.Errorf("broadcaster_user_id not present in Kick channel response for %q", slug)
}

func (c *EventSubscriptionClient) ListEventSubscriptions(ctx context.Context) ([]domain.KickAPIEventSub, error) {
	token, err := c.getAccessToken(ctx)
	if err != nil {
		return nil, err
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, c.apiBaseURL+"/public/v1/events/subscriptions", nil)
	if err != nil {
		return nil, fmt.Errorf("build list subscriptions request: %w", err)
	}
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("Accept", "application/json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("list event subscriptions: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil, fmt.Errorf("list event subscriptions returned status %d", resp.StatusCode)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("read list subscriptions response: %w", err)
	}

	var result struct {
		Data []kickSubResponse `json:"data"`
	}
	if err := json.Unmarshal(body, &result); err != nil {
		return nil, fmt.Errorf("decode list subscriptions response: %w", err)
	}

	return toAPIEventSubs(result.Data), nil
}

func (c *EventSubscriptionClient) CreateEventSubscription(ctx context.Context, broadcasterUserID int64, eventType string) (domain.KickAPIEventSub, error) {
	token, err := c.getAccessToken(ctx)
	if err != nil {
		return domain.KickAPIEventSub{}, err
	}

	body, err := json.Marshal(struct {
		BroadcasterUserID int64  `json:"broadcaster_user_id"`
		Type              string `json:"type"`
		Method            string `json:"method"`
	}{
		BroadcasterUserID: broadcasterUserID,
		Type:              eventType,
		Method:            "webhook",
	})
	if err != nil {
		return domain.KickAPIEventSub{}, fmt.Errorf("marshal create subscription request: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.apiBaseURL+"/public/v1/events/subscriptions", bytes.NewReader(body))
	if err != nil {
		return domain.KickAPIEventSub{}, fmt.Errorf("build create subscription request: %w", err)
	}
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return domain.KickAPIEventSub{}, fmt.Errorf("create event subscription: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return domain.KickAPIEventSub{}, fmt.Errorf("create event subscription returned status %d", resp.StatusCode)
	}

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return domain.KickAPIEventSub{}, fmt.Errorf("read create subscription response: %w", err)
	}

	// Kick returns {"data": [{...}]} (array), not {"data": {...}}.
	var result struct {
		Data []kickSubResponse `json:"data"`
	}
	if err := json.Unmarshal(respBody, &result); err != nil {
		return domain.KickAPIEventSub{}, fmt.Errorf("decode create subscription response: %w", err)
	}

	subs := toAPIEventSubs(result.Data)
	if len(subs) == 0 {
		return domain.KickAPIEventSub{}, fmt.Errorf("empty response from create event subscription")
	}
	return subs[0], nil
}

// FetchWebhookPublicKey retrieves the app's webhook signing public key from the Kick API.
// Tries multiple known endpoint paths. The returned string can be passed to NewWebhookVerifier.
func (c *EventSubscriptionClient) FetchWebhookPublicKey(ctx context.Context) (string, error) {
	token, err := c.getAccessToken(ctx)
	if err != nil {
		return "", err
	}

	candidates := []string{
		"/public/v1/webhooks/public-key",
		"/public/v1/events/subscriptions/public-key",
		"/public/v1/events/public-key",
	}

	for _, path := range candidates {
		key, err := c.tryFetchPublicKey(ctx, token, path)
		if err == nil {
			return key, nil
		}
	}
	return "", fmt.Errorf("could not fetch webhook public key from any known Kick API path (set KICK_WEBHOOK_PUBLIC_KEY manually)")
}

func (c *EventSubscriptionClient) tryFetchPublicKey(ctx context.Context, token, path string) (string, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, c.apiBaseURL+path, nil)
	if err != nil {
		return "", err
	}
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("Accept", "application/json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return "", fmt.Errorf("status %d", resp.StatusCode)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}

	var result struct {
		PublicKey string `json:"public_key"`
	}
	if err := json.Unmarshal(body, &result); err != nil {
		return "", err
	}
	if result.PublicKey == "" {
		return "", fmt.Errorf("empty public_key")
	}
	return result.PublicKey, nil
}

func (c *EventSubscriptionClient) DeleteEventSubscription(ctx context.Context, subscriptionID string) error {
	token, err := c.getAccessToken(ctx)
	if err != nil {
		return err
	}

	reqURL := fmt.Sprintf("%s/public/v1/events/subscriptions/%s", c.apiBaseURL, url.PathEscape(subscriptionID))
	req, err := http.NewRequestWithContext(ctx, http.MethodDelete, reqURL, nil)
	if err != nil {
		return fmt.Errorf("build delete subscription request: %w", err)
	}
	req.Header.Set("Authorization", "Bearer "+token)

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("delete event subscription: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusNotFound {
		return nil
	}
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return fmt.Errorf("delete event subscription returned status %d", resp.StatusCode)
	}
	return nil
}

func (c *EventSubscriptionClient) getAccessToken(ctx context.Context) (string, error) {
	c.mu.Lock()
	defer c.mu.Unlock()

	if c.accessToken != "" && time.Now().Before(c.tokenExpiry) {
		return c.accessToken, nil
	}

	data := url.Values{
		"grant_type":    []string{"client_credentials"},
		"client_id":     []string{c.clientID},
		"client_secret": []string{c.clientSecret},
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.oauthURL, strings.NewReader(data.Encode()))
	if err != nil {
		return "", fmt.Errorf("build token request: %w", err)
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Set("Accept", "application/json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return "", fmt.Errorf("fetch access token: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return "", fmt.Errorf("fetch access token returned status %d", resp.StatusCode)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("read token response: %w", err)
	}

	var tokenResp struct {
		AccessToken string `json:"access_token"`
		ExpiresIn   int    `json:"expires_in"`
	}
	if err := json.Unmarshal(body, &tokenResp); err != nil {
		return "", fmt.Errorf("decode token response: %w", err)
	}
	if tokenResp.AccessToken == "" {
		return "", fmt.Errorf("empty access token in response")
	}

	c.accessToken = tokenResp.AccessToken
	expiresIn := tokenResp.ExpiresIn
	if expiresIn <= 60 {
		expiresIn = 60
	}
	c.tokenExpiry = time.Now().Add(time.Duration(expiresIn-60) * time.Second)
	return c.accessToken, nil
}

type kickSubResponse struct {
	ID                string `json:"id"`
	SubscriptionID    string `json:"subscription_id"` // alternate ID field name
	BroadcasterUserID int64  `json:"broadcaster_user_id"`
	UserID            int64  `json:"user_id"` // alternate broadcaster field name
	Type              string `json:"type"`
	Name              string `json:"name"`  // alternate event type field name
	Event             string `json:"event"` // another alternate
	Method            string `json:"method"`
	CreatedAt         string `json:"created_at"`
}

func (r kickSubResponse) subID() string {
	if r.ID != "" {
		return r.ID
	}
	return r.SubscriptionID
}

func (r kickSubResponse) broadcasterID() int64 {
	if r.BroadcasterUserID != 0 {
		return r.BroadcasterUserID
	}
	return r.UserID
}

func (r kickSubResponse) eventType() string {
	if r.Type != "" {
		return r.Type
	}
	if r.Name != "" {
		return r.Name
	}
	return r.Event
}

func toAPIEventSubs(responses []kickSubResponse) []domain.KickAPIEventSub {
	subs := make([]domain.KickAPIEventSub, 0, len(responses))
	for _, r := range responses {
		var createdAt time.Time
		if r.CreatedAt != "" {
			createdAt, _ = time.Parse(time.RFC3339, r.CreatedAt)
		}
		subs = append(subs, domain.KickAPIEventSub{
			SubscriptionID:    r.subID(),
			BroadcasterUserID: r.broadcasterID(),
			EventType:         r.eventType(),
			Method:            r.Method,
			CreatedAt:         createdAt,
		})
	}
	return subs
}
