package routes

import (
	"errors"
	"net/http"
	"time"

	"github.com/YSelim0/kick-logs/apps/api-go/internal/domain"
	predictionsusecase "github.com/YSelim0/kick-logs/apps/api-go/internal/usecase/predictions"
)

func RegisterPredictionRoutes(mux *http.ServeMux, deps Dependencies) {
	mux.HandleFunc("GET /channels/{slug}/prediction", func(response http.ResponseWriter, request *http.Request) {
		getChannelPrediction(response, request, deps)
	})
}

func getChannelPrediction(response http.ResponseWriter, request *http.Request, deps Dependencies) {
	if deps.Predictions == nil {
		writeError(response, http.StatusInternalServerError, "Internal server error.")
		return
	}
	slug, ok := profileSlug(response, request)
	if !ok {
		return
	}
	prediction, err := deps.Predictions.LatestPrediction(request.Context(), slug)
	if err != nil {
		writePredictionUseCaseError(response, err)
		return
	}
	writeJSON(response, http.StatusOK, predictionResponse(prediction))
}

func writePredictionUseCaseError(response http.ResponseWriter, err error) {
	switch {
	case errors.Is(err, predictionsusecase.ErrInvalidSlug):
		writeError(response, http.StatusUnprocessableEntity, "Invalid channel slug.")
	case errors.Is(err, predictionsusecase.ErrChannelNotFound):
		writeError(response, http.StatusNotFound, "Channel not found.")
	case errors.Is(err, predictionsusecase.ErrNotFound):
		writeError(response, http.StatusNotFound, "No active prediction found for this channel.")
	case errors.Is(err, predictionsusecase.ErrBlocked):
		writeError(response, http.StatusBadGateway, "Kick prediction request was blocked. Try again later.")
	default:
		writeError(response, http.StatusInternalServerError, "Internal server error.")
	}
}

type predictionResponseBody struct {
	ID               string                  `json:"id"`
	ChannelID        int64                   `json:"channelId"`
	Title            string                  `json:"title"`
	DurationSeconds  int                     `json:"durationSeconds"`
	State            string                  `json:"state"`
	WinningOutcomeID *string                 `json:"winningOutcomeId"`
	CreatedAt        *string                 `json:"createdAt"`
	LockedAt         *string                 `json:"lockedAt"`
	UpdatedAt        *string                 `json:"updatedAt"`
	TotalPoints      int64                   `json:"totalPoints"`
	TotalVotes       int64                   `json:"totalVotes"`
	Outcomes         []predictionOutcomeBody `json:"outcomes"`
}

type predictionOutcomeBody struct {
	ID              string                  `json:"id"`
	Title           string                  `json:"title"`
	TotalVoteAmount int64                   `json:"totalVoteAmount"`
	VoteCount       int64                   `json:"voteCount"`
	ReturnRate      float64                 `json:"returnRate"`
	PointShare      float64                 `json:"pointShare"`
	IsWinner        bool                    `json:"isWinner"`
	TopUsers        []predictionTopUserBody `json:"topUsers"`
}

type predictionTopUserBody struct {
	ID       int64  `json:"id"`
	Username string `json:"username"`
	Amount   int64  `json:"amount"`
}

func predictionResponse(prediction domain.Prediction) predictionResponseBody {
	outcomes := make([]predictionOutcomeBody, 0, len(prediction.Outcomes))
	for _, outcome := range prediction.Outcomes {
		topUsers := make([]predictionTopUserBody, 0, len(outcome.TopUsers))
		for _, user := range outcome.TopUsers {
			topUsers = append(topUsers, predictionTopUserBody{
				ID:       user.ID,
				Username: user.Username,
				Amount:   user.Amount,
			})
		}
		outcomes = append(outcomes, predictionOutcomeBody{
			ID:              outcome.ID,
			Title:           outcome.Title,
			TotalVoteAmount: outcome.TotalVoteAmount,
			VoteCount:       outcome.VoteCount,
			ReturnRate:      outcome.ReturnRate,
			PointShare:      outcome.PointShare,
			IsWinner:        outcome.IsWinner,
			TopUsers:        topUsers,
		})
	}

	return predictionResponseBody{
		ID:               prediction.ID,
		ChannelID:        prediction.ChannelID,
		Title:            prediction.Title,
		DurationSeconds:  prediction.DurationSeconds,
		State:            prediction.State,
		WinningOutcomeID: nullableString(prediction.WinningOutcomeID),
		CreatedAt:        nullableTime(prediction.CreatedAt),
		LockedAt:         nullableTimePointer(prediction.LockedAt),
		UpdatedAt:        nullableTime(prediction.UpdatedAt),
		TotalPoints:      prediction.TotalPoints,
		TotalVotes:       prediction.TotalVotes,
		Outcomes:         outcomes,
	}
}

func nullableTimePointer(value *time.Time) *string {
	if value == nil {
		return nil
	}
	return nullableTime(*value)
}
