package httpapi

import (
	"context"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/YSelim0/kick-logs/apps/api-go/internal/config"
	"github.com/YSelim0/kick-logs/apps/api-go/internal/domain"
	"github.com/YSelim0/kick-logs/apps/api-go/internal/http/routes"
	"github.com/YSelim0/kick-logs/apps/api-go/internal/ports"
	predictionsusecase "github.com/YSelim0/kick-logs/apps/api-go/internal/usecase/predictions"
)

type fakePredictionFetcher struct {
	prediction domain.Prediction
	err        error
}

func (fake *fakePredictionFetcher) LatestPrediction(_ context.Context, _ string) (domain.Prediction, error) {
	if fake.err != nil {
		return domain.Prediction{}, fake.err
	}
	return fake.prediction, nil
}

func newPredictionTestRouter(fetcher ports.PredictionFetcher) http.Handler {
	cfg := config.Config{BackendCORSOrigins: []string{"http://localhost:3000"}}
	return NewRouter(cfg, slog.New(slog.NewTextHandler(io.Discard, nil)), routes.Dependencies{
		Config:      cfg,
		Predictions: predictionsusecase.NewService(fetcher),
	})
}

func TestPredictionRouteReturnsNormalizedPrediction(t *testing.T) {
	handler := newPredictionTestRouter(&fakePredictionFetcher{
		prediction: domain.Prediction{
			ID:               "pred-1",
			ChannelID:        12440103,
			Title:            "mac bitis suresi",
			State:            "RESOLVED",
			WinningOutcomeID: "out-1",
			Outcomes: []domain.PredictionOutcome{
				{ID: "out-1", Title: "Evet", TotalVoteAmount: 75, VoteCount: 3, ReturnRate: 1.33},
				{ID: "out-2", Title: "Hayir", TotalVoteAmount: 25, VoteCount: 1, ReturnRate: 4.0},
			},
		},
	})

	response := httptest.NewRecorder()
	handler.ServeHTTP(response, httptest.NewRequest(http.MethodGet, "/channels/nuriben/prediction", nil))

	if response.Code != http.StatusOK {
		t.Fatalf("status = %d body = %s", response.Code, response.Body.String())
	}
	body := response.Body.String()
	for _, want := range []string{
		`"totalPoints":100`,
		`"totalVotes":4`,
		`"pointShare":0.75`,
		`"isWinner":true`,
		`"winningOutcomeId":"out-1"`,
	} {
		if !strings.Contains(body, want) {
			t.Fatalf("body missing %q: %s", want, body)
		}
	}
}

func TestPredictionRouteMapsFetchErrors(t *testing.T) {
	cases := []struct {
		name       string
		err        error
		wantStatus int
		wantDetail string
	}{
		{"no prediction", ports.ErrPredictionNotFound, http.StatusNotFound, "No active prediction found"},
		{"channel missing", ports.ErrPredictionChannelNotFound, http.StatusNotFound, "Channel not found."},
		{"blocked", ports.ErrPredictionBlocked, http.StatusBadGateway, "blocked"},
	}
	for _, testCase := range cases {
		t.Run(testCase.name, func(t *testing.T) {
			handler := newPredictionTestRouter(&fakePredictionFetcher{err: testCase.err})
			response := httptest.NewRecorder()
			handler.ServeHTTP(response, httptest.NewRequest(http.MethodGet, "/channels/nuriben/prediction", nil))
			if response.Code != testCase.wantStatus {
				t.Fatalf("status = %d body = %s", response.Code, response.Body.String())
			}
			if !strings.Contains(response.Body.String(), testCase.wantDetail) {
				t.Fatalf("body = %s", response.Body.String())
			}
		})
	}
}
