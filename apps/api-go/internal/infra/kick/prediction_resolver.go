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
	"github.com/YSelim0/kick-logs/apps/api-go/internal/ports"
)

// WebPredictionResolver reads the undocumented Kick latest-prediction endpoint. Browser-like headers
// improve the chance of a JSON response instead of a "Request blocked by security policy" body.
type WebPredictionResolver struct {
	client  *http.Client
	baseURL string
}

func NewWebPredictionResolver() *WebPredictionResolver {
	return &WebPredictionResolver{
		client:  &http.Client{Timeout: 15 * time.Second},
		baseURL: "https://kick.com",
	}
}

func (resolver *WebPredictionResolver) LatestPrediction(ctx context.Context, slug string) (domain.Prediction, error) {
	slug = strings.TrimSpace(slug)
	if slug == "" {
		return domain.Prediction{}, fmt.Errorf("slug is required")
	}

	url := fmt.Sprintf("%s/api/v2/channels/%s/predictions/latest", strings.TrimRight(resolver.baseURL, "/"), slug)
	request, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return domain.Prediction{}, fmt.Errorf("build Kick prediction request: %w", err)
	}
	applyBrowserHeaders(request, slug)

	response, err := resolver.client.Do(request)
	if err != nil {
		return domain.Prediction{}, fmt.Errorf("fetch Kick prediction: %w", err)
	}
	defer response.Body.Close()

	if response.StatusCode == http.StatusNotFound {
		return domain.Prediction{}, ports.ErrPredictionChannelNotFound
	}
	if response.StatusCode < 200 || response.StatusCode >= 300 {
		return domain.Prediction{}, ports.ErrPredictionBlocked
	}

	rawPayload, err := io.ReadAll(response.Body)
	if err != nil {
		return domain.Prediction{}, fmt.Errorf("read Kick prediction response: %w", err)
	}

	var payload predictionEnvelope
	if err := json.Unmarshal(rawPayload, &payload); err != nil {
		return domain.Prediction{}, fmt.Errorf("decode Kick prediction response: %w", err)
	}

	// A non-empty top-level error field means Kick rejected the request even with a 2xx status.
	if strings.TrimSpace(payload.Error) != "" {
		return domain.Prediction{}, ports.ErrPredictionBlocked
	}
	if payload.Data.Prediction == nil {
		return domain.Prediction{}, ports.ErrPredictionNotFound
	}

	return payload.Data.Prediction.toDomain(), nil
}

func applyBrowserHeaders(request *http.Request, slug string) {
	request.Header.Set("accept", "application/json")
	request.Header.Set("accept-language", "en-US,en;q=0.9")
	request.Header.Set("referer", fmt.Sprintf("https://kick.com/%s", slug))
	request.Header.Set("user-agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/124.0 Safari/537.36")
	request.Header.Set("sec-ch-ua", `"Chromium";v="124", "Google Chrome";v="124", "Not-A.Brand";v="99"`)
	request.Header.Set("sec-ch-ua-mobile", "?0")
	request.Header.Set("sec-ch-ua-platform", `"Windows"`)
	request.Header.Set("sec-fetch-dest", "empty")
	request.Header.Set("sec-fetch-mode", "cors")
	request.Header.Set("sec-fetch-site", "same-origin")
}

type predictionEnvelope struct {
	Data    predictionData `json:"data"`
	Message string         `json:"message"`
	Error   string         `json:"error"`
}

type predictionData struct {
	Prediction *predictionPayload `json:"prediction"`
}

type predictionPayload struct {
	ID               string           `json:"id"`
	ChannelID        int64            `json:"channel_id"`
	Title            string           `json:"title"`
	Outcomes         []outcomePayload `json:"outcomes"`
	Duration         int              `json:"duration"`
	CreatedAt        time.Time        `json:"created_at"`
	UpdatedAt        time.Time        `json:"updated_at"`
	LockedAt         *time.Time       `json:"locked_at"`
	State            string           `json:"state"`
	WinningOutcomeID *string          `json:"winning_outcome_id"`
}

type outcomePayload struct {
	ID              string           `json:"id"`
	Title           string           `json:"title"`
	TotalVoteAmount int64            `json:"total_vote_amount"`
	VoteCount       int64            `json:"vote_count"`
	ReturnRate      float64          `json:"return_rate"`
	TopUsers        []topUserPayload `json:"top_users"`
}

type topUserPayload struct {
	ID       int64  `json:"id"`
	Username string `json:"username"`
	Amount   int64  `json:"amount"`
}

func (payload predictionPayload) toDomain() domain.Prediction {
	winningID := ""
	if payload.WinningOutcomeID != nil {
		winningID = strings.TrimSpace(*payload.WinningOutcomeID)
	}

	outcomes := make([]domain.PredictionOutcome, 0, len(payload.Outcomes))
	for _, outcome := range payload.Outcomes {
		topUsers := make([]domain.PredictionTopUser, 0, len(outcome.TopUsers))
		for _, user := range outcome.TopUsers {
			topUsers = append(topUsers, domain.PredictionTopUser{
				ID:       user.ID,
				Username: user.Username,
				Amount:   user.Amount,
			})
		}
		outcomes = append(outcomes, domain.PredictionOutcome{
			ID:              outcome.ID,
			Title:           outcome.Title,
			TotalVoteAmount: outcome.TotalVoteAmount,
			VoteCount:       outcome.VoteCount,
			ReturnRate:      outcome.ReturnRate,
			TopUsers:        topUsers,
		})
	}

	return domain.Prediction{
		ID:               payload.ID,
		ChannelID:        payload.ChannelID,
		Title:            payload.Title,
		DurationSeconds:  payload.Duration,
		State:            payload.State,
		WinningOutcomeID: winningID,
		CreatedAt:        payload.CreatedAt,
		LockedAt:         payload.LockedAt,
		UpdatedAt:        payload.UpdatedAt,
		Outcomes:         outcomes,
	}
}
