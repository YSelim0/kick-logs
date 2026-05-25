package domain

import "time"

// Prediction is a normalized Kick channel prediction. The Kick prediction endpoint is undocumented
// and unstable; derived totals and per-outcome share/winner flags are computed in the use case, not
// returned directly by Kick.
type Prediction struct {
	ID               string
	ChannelID        int64
	Title            string
	DurationSeconds  int
	State            string
	WinningOutcomeID string
	CreatedAt        time.Time
	LockedAt         *time.Time
	UpdatedAt        time.Time
	TotalPoints      int64
	TotalVotes       int64
	Outcomes         []PredictionOutcome
}

type PredictionOutcome struct {
	ID              string
	Title           string
	TotalVoteAmount int64
	VoteCount       int64
	ReturnRate      float64
	PointShare      float64
	IsWinner        bool
	TopUsers        []PredictionTopUser
}

type PredictionTopUser struct {
	ID       int64
	Username string
	Amount   int64
}
