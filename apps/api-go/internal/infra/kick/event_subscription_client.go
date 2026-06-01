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

// ResolveBroadcasterUserID fetches the Kick broadcaster user ID for a channel slug.
// It tries the official public API first and falls back to Kick's web channel endpoint.
func (c *EventSubscriptionClient) ResolveBroadcasterUserID(ctx context.Context, slug string) (int64, error) {
	slug = strings.TrimSpace(slug)
	if slug == "" {
		return 0, fmt.Errorf("slug is required")
	}

	if broadcasterID, err := c.resolveBroadcasterUserIDFromPublicAPI(ctx, slug); err == nil && broadcasterID != 0 {
		return broadcasterID, nil
	}
	return c.resolveBroadcasterUserIDFromWebAPI(ctx, slug)
}

func (c *EventSubscriptionClient) resolveBroadcasterUserIDFromPublicAPI(ctx context.Context, slug string) (int64, error) {
	token, err := c.getAccessToken(ctx)
	if err != nil {
		return 0, err
	}

	reqURL := c.apiBaseURL + "/public/v1/channels?slug=" + url.QueryEscape(slug)
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, reqURL, nil)
	if err != nil {
		return 0, fmt.Errorf("build public channel request: %w", err)
	}
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("Accept", "application/json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return 0, fmt.Errorf("fetch public channel for broadcaster user id: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return 0, fmt.Errorf("public channel lookup returned status %d", resp.StatusCode)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return 0, fmt.Errorf("read public channel response: %w", err)
	}

	var payload struct {
		Data []struct {
			BroadcasterUserID int64 `json:"broadcaster_user_id"`
			UserID            int64 `json:"user_id"`
			User              struct {
				ID int64 `json:"id"`
			} `json:"user"`
		} `json:"data"`
	}
	if err := json.Unmarshal(body, &payload); err != nil {
		return 0, fmt.Errorf("decode public channel response: %w", err)
	}
	for _, item := range payload.Data {
		switch {
		case item.BroadcasterUserID != 0:
			return item.BroadcasterUserID, nil
		case item.UserID != 0:
			return item.UserID, nil
		case item.User.ID != 0:
			return item.User.ID, nil
		}
	}
	return 0, fmt.Errorf("broadcaster_user_id not present in public Kick channel response for %q", slug)
}

func (c *EventSubscriptionClient) resolveBroadcasterUserIDFromWebAPI(ctx context.Context, slug string) (int64, error) {
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
	subs, err := c.CreateEventSubscriptions(ctx, broadcasterUserID, []string{eventType})
	if err != nil {
		return domain.KickAPIEventSub{}, err
	}
	if len(subs) == 0 {
		return domain.KickAPIEventSub{}, fmt.Errorf("empty response from create event subscription")
	}
	return subs[0], nil
}

func (c *EventSubscriptionClient) CreateEventSubscriptions(ctx context.Context, broadcasterUserID int64, eventTypes []string) ([]domain.KickAPIEventSub, error) {
	token, err := c.getAccessToken(ctx)
	if err != nil {
		return nil, err
	}

	events := make([]kickSubCreateEvent, 0, len(eventTypes))
	for _, eventType := range eventTypes {
		eventType = strings.TrimSpace(eventType)
		if eventType == "" {
			continue
		}
		events = append(events, kickSubCreateEvent{Name: eventType, Version: 1})
	}
	if len(events) == 0 {
		return nil, fmt.Errorf("at least one event type is required")
	}

	body, err := json.Marshal(struct {
		BroadcasterUserID int64                `json:"broadcaster_user_id"`
		Events            []kickSubCreateEvent `json:"events"`
		Method            string               `json:"method"`
	}{
		BroadcasterUserID: broadcasterUserID,
		Events:            events,
		Method:            "webhook",
	})
	if err != nil {
		return nil, fmt.Errorf("marshal create subscription request: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.apiBaseURL+"/public/v1/events/subscriptions", bytes.NewReader(body))
	if err != nil {
		return nil, fmt.Errorf("build create subscription request: %w", err)
	}
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("create event subscription: %w", err)
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("read create subscription response: %w", err)
	}

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil, fmt.Errorf("create event subscription returned status %d: %s", resp.StatusCode, trimResponseBody(respBody))
	}

	var result struct {
		Data []kickSubResponse `json:"data"`
	}
	if err := json.Unmarshal(respBody, &result); err != nil {
		return nil, fmt.Errorf("decode create subscription response: %w", err)
	}

	subs := toAPIEventSubs(result.Data)
	for i := range subs {
		if subs[i].BroadcasterUserID == 0 {
			subs[i].BroadcasterUserID = broadcasterUserID
		}
		if subs[i].Method == "" {
			subs[i].Method = "webhook"
		}
	}
	if len(subs) == 0 {
		return nil, fmt.Errorf("empty response from create event subscription: %s", trimResponseBody(respBody))
	}
	return subs, nil
}

// FetchWebhookPublicKey retrieves the app's webhook signing public key from the Kick API.
// The returned PEM string can be passed to NewWebhookVerifier.
func (c *EventSubscriptionClient) FetchWebhookPublicKey(ctx context.Context) (string, error) {
	token, err := c.getAccessToken(ctx)
	if err != nil {
		return "", err
	}

	key, err := c.tryFetchPublicKey(ctx, token, "/public/v1/public-key")
	if err != nil {
		return "", fmt.Errorf("fetch webhook public key: %w", err)
	}
	return key, nil
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
		Data      struct {
			PublicKey string `json:"public_key"`
		} `json:"data"`
	}
	if err := json.Unmarshal(body, &result); err != nil {
		return "", err
	}
	if result.PublicKey == "" {
		result.PublicKey = result.Data.PublicKey
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

	reqURL := c.apiBaseURL + "/public/v1/events/subscriptions?id=" + url.QueryEscape(subscriptionID)
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

type kickSubCreateEvent struct {
	Name    string `json:"name"`
	Version int    `json:"version"`
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
	ID                string         `json:"id"`
	SubscriptionID    string         `json:"subscription_id"` // alternate ID field name
	BroadcasterUserID int64          `json:"broadcaster_user_id"`
	UserID            int64          `json:"user_id"` // alternate broadcaster field name
	Type              string         `json:"type"`
	Name              string         `json:"name"`  // alternate event type field name
	Event             eventNameValue `json:"event"` // another alternate; may be a string or object
	Method            string         `json:"method"`
	CreatedAt         string         `json:"created_at"`
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
	return r.Event.String()
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

func trimResponseBody(body []byte) string {
	text := strings.TrimSpace(string(body))
	if text == "" {
		return "<empty>"
	}
	const maxLen = 512
	if len(text) > maxLen {
		return text[:maxLen] + "..."
	}
	return text
}

type eventNameValue string

func (v *eventNameValue) UnmarshalJSON(data []byte) error {
	var s string
	if err := json.Unmarshal(data, &s); err == nil {
		*v = eventNameValue(s)
		return nil
	}

	var obj struct {
		Name  string `json:"name"`
		Type  string `json:"type"`
		Event string `json:"event"`
	}
	if err := json.Unmarshal(data, &obj); err != nil {
		return err
	}
	switch {
	case obj.Name != "":
		*v = eventNameValue(obj.Name)
	case obj.Type != "":
		*v = eventNameValue(obj.Type)
	default:
		*v = eventNameValue(obj.Event)
	}
	return nil
}

func (v eventNameValue) String() string {
	return string(v)
}
