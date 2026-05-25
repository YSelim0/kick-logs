package ports

import (
	"context"
	"errors"

	"github.com/YSelim0/kick-logs/apps/api-go/internal/domain"
)

// Prediction fetch error contract. The infra adapter returns these so the use case and HTTP layer
// can map Kick failures to stable responses without depending on Kick internals.
var (
	// ErrPredictionNotFound means the channel exists but has no latest prediction
	// (Kick returned a null prediction payload).
	ErrPredictionNotFound = errors.New("prediction not found")
	// ErrPredictionChannelNotFound means Kick returned 404 for the channel.
	ErrPredictionChannelNotFound = errors.New("prediction channel not found")
	// ErrPredictionBlocked means Kick rejected the request (security policy or non-2xx status).
	ErrPredictionBlocked = errors.New("prediction request blocked")
)

type PredictionFetcher interface {
	LatestPrediction(ctx context.Context, slug string) (domain.Prediction, error)
}
