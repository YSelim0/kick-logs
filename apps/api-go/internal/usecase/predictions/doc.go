// Package predictions normalizes a channel's latest Kick prediction: it validates the slug, fetches
// the raw prediction through a ports.PredictionFetcher, and derives totals, per-outcome point share,
// and the winning-outcome flag. It owns no storage; every call fetches live data.
package predictions
