package predictions

import (
	"context"
	"errors"
	"strings"

	"github.com/YSelim0/kick-logs/apps/api-go/internal/domain"
	"github.com/YSelim0/kick-logs/apps/api-go/internal/ports"
)

// Re-export the fetch error contract so the HTTP layer can map failures without importing ports.
var (
	ErrInvalidSlug     = errors.New("invalid prediction slug")
	ErrNotFound        = ports.ErrPredictionNotFound
	ErrChannelNotFound = ports.ErrPredictionChannelNotFound
	ErrBlocked         = ports.ErrPredictionBlocked
)

const maxSlugLength = 160

type Service struct {
	fetcher ports.PredictionFetcher
}

func NewService(fetcher ports.PredictionFetcher) *Service {
	return &Service{fetcher: fetcher}
}

func (service *Service) LatestPrediction(ctx context.Context, slug string) (domain.Prediction, error) {
	slug = strings.ToLower(strings.TrimSpace(slug))
	if slug == "" || len(slug) > maxSlugLength {
		return domain.Prediction{}, ErrInvalidSlug
	}

	prediction, err := service.fetcher.LatestPrediction(ctx, slug)
	if err != nil {
		return domain.Prediction{}, err
	}

	return derive(prediction), nil
}

// derive fills the totals and per-outcome share/winner fields the Kick endpoint does not return.
func derive(prediction domain.Prediction) domain.Prediction {
	var totalPoints, totalVotes int64
	for _, outcome := range prediction.Outcomes {
		totalPoints += outcome.TotalVoteAmount
		totalVotes += outcome.VoteCount
	}
	prediction.TotalPoints = totalPoints
	prediction.TotalVotes = totalVotes

	for index := range prediction.Outcomes {
		outcome := &prediction.Outcomes[index]
		if totalPoints > 0 {
			outcome.PointShare = float64(outcome.TotalVoteAmount) / float64(totalPoints)
		} else {
			outcome.PointShare = 0
		}
		outcome.IsWinner = prediction.WinningOutcomeID != "" && outcome.ID == prediction.WinningOutcomeID
	}

	return prediction
}
