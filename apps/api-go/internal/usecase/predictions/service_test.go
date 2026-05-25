package predictions

import (
	"context"
	"errors"
	"testing"

	"github.com/YSelim0/kick-logs/apps/api-go/internal/domain"
	"github.com/YSelim0/kick-logs/apps/api-go/internal/ports"
)

type fakeFetcher struct {
	prediction domain.Prediction
	err        error
	gotSlug    string
}

func (fake *fakeFetcher) LatestPrediction(_ context.Context, slug string) (domain.Prediction, error) {
	fake.gotSlug = slug
	if fake.err != nil {
		return domain.Prediction{}, fake.err
	}
	return fake.prediction, nil
}

func TestLatestPredictionDerivesTotalsShareAndWinner(t *testing.T) {
	fake := &fakeFetcher{
		prediction: domain.Prediction{
			ID:               "pred-1",
			WinningOutcomeID: "out-1",
			Outcomes: []domain.PredictionOutcome{
				{ID: "out-1", TotalVoteAmount: 75, VoteCount: 3},
				{ID: "out-2", TotalVoteAmount: 25, VoteCount: 1},
			},
		},
	}
	service := NewService(fake)

	prediction, err := service.LatestPrediction(context.Background(), "NuriBen")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if fake.gotSlug != "nuriben" {
		t.Fatalf("expected slug normalized to lowercase, got %q", fake.gotSlug)
	}
	if prediction.TotalPoints != 100 {
		t.Fatalf("expected total points 100, got %d", prediction.TotalPoints)
	}
	if prediction.TotalVotes != 4 {
		t.Fatalf("expected total votes 4, got %d", prediction.TotalVotes)
	}
	if prediction.Outcomes[0].PointShare != 0.75 {
		t.Fatalf("expected point share 0.75, got %v", prediction.Outcomes[0].PointShare)
	}
	if !prediction.Outcomes[0].IsWinner {
		t.Fatalf("expected outcome out-1 to be winner")
	}
	if prediction.Outcomes[1].IsWinner {
		t.Fatalf("expected outcome out-2 not to be winner")
	}
}

func TestLatestPredictionZeroPointsDoesNotDivideByZero(t *testing.T) {
	fake := &fakeFetcher{
		prediction: domain.Prediction{
			Outcomes: []domain.PredictionOutcome{
				{ID: "out-1", TotalVoteAmount: 0, VoteCount: 0},
			},
		},
	}
	service := NewService(fake)

	prediction, err := service.LatestPrediction(context.Background(), "channel")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if prediction.Outcomes[0].PointShare != 0 {
		t.Fatalf("expected zero point share, got %v", prediction.Outcomes[0].PointShare)
	}
}

func TestLatestPredictionRejectsEmptySlug(t *testing.T) {
	service := NewService(&fakeFetcher{})
	if _, err := service.LatestPrediction(context.Background(), "   "); !errors.Is(err, ErrInvalidSlug) {
		t.Fatalf("expected ErrInvalidSlug, got %v", err)
	}
}

func TestLatestPredictionPropagatesFetcherError(t *testing.T) {
	service := NewService(&fakeFetcher{err: ports.ErrPredictionNotFound})
	if _, err := service.LatestPrediction(context.Background(), "channel"); !errors.Is(err, ErrNotFound) {
		t.Fatalf("expected ErrNotFound, got %v", err)
	}
}
