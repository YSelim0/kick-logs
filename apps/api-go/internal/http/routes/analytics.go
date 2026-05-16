package routes

import (
	"errors"
	"net/http"
	"strconv"
	"strings"

	"github.com/YSelim0/kick-logs/apps/api-go/internal/domain"
	analyticsusecase "github.com/YSelim0/kick-logs/apps/api-go/internal/usecase/analytics"
)

func RegisterAnalyticsRoutes(mux *http.ServeMux, deps Dependencies) {
	mux.HandleFunc("GET /analytics/overview", func(response http.ResponseWriter, request *http.Request) {
		getAnalyticsOverview(response, request, deps)
	})
	mux.HandleFunc("GET /analytics/message-volume", func(response http.ResponseWriter, request *http.Request) {
		getMessageVolume(response, request, deps)
	})
	mux.HandleFunc("GET /analytics/top-senders", func(response http.ResponseWriter, request *http.Request) {
		getTopSenders(response, request, deps)
	})
	mux.HandleFunc("GET /analytics/top-channels", func(response http.ResponseWriter, request *http.Request) {
		getTopChannels(response, request, deps)
	})
	mux.HandleFunc("GET /analytics/top-emotes", func(response http.ResponseWriter, request *http.Request) {
		getTopEmotes(response, request, deps)
	})
}

func getAnalyticsOverview(response http.ResponseWriter, request *http.Request, deps Dependencies) {
	if deps.Analytics == nil {
		writeError(response, http.StatusInternalServerError, "Internal server error.")
		return
	}
	filter, err := parseAnalyticsFilter(request)
	if err != nil {
		writeAnalyticsFilterError(response, err)
		return
	}
	overview, err := deps.Analytics.Overview(request.Context(), filter)
	if err != nil {
		writeAnalyticsUseCaseError(response, err)
		return
	}
	writeJSON(response, http.StatusOK, analyticsOverviewResponse(overview))
}

func getMessageVolume(response http.ResponseWriter, request *http.Request, deps Dependencies) {
	if deps.Analytics == nil {
		writeError(response, http.StatusInternalServerError, "Internal server error.")
		return
	}
	filter, err := parseAnalyticsFilter(request)
	if err != nil {
		writeAnalyticsFilterError(response, err)
		return
	}
	bucket, err := parseAnalyticsBucket(request)
	if err != nil {
		writeAnalyticsFilterError(response, err)
		return
	}
	points, err := deps.Analytics.MessageVolume(request.Context(), filter, bucket)
	if err != nil {
		writeAnalyticsUseCaseError(response, err)
		return
	}
	writeJSON(response, http.StatusOK, messageVolumeResponse(points))
}

func getTopSenders(response http.ResponseWriter, request *http.Request, deps Dependencies) {
	if deps.Analytics == nil {
		writeError(response, http.StatusInternalServerError, "Internal server error.")
		return
	}
	filter, limit, err := parseAnalyticsTopRequest(request)
	if err != nil {
		writeAnalyticsFilterError(response, err)
		return
	}
	senders, err := deps.Analytics.TopSenders(request.Context(), filter, limit)
	if err != nil {
		writeAnalyticsUseCaseError(response, err)
		return
	}
	writeJSON(response, http.StatusOK, topSendersResponse(senders))
}

func getTopChannels(response http.ResponseWriter, request *http.Request, deps Dependencies) {
	if deps.Analytics == nil {
		writeError(response, http.StatusInternalServerError, "Internal server error.")
		return
	}
	filter, limit, err := parseAnalyticsTopRequest(request)
	if err != nil {
		writeAnalyticsFilterError(response, err)
		return
	}
	channels, err := deps.Analytics.TopChannels(request.Context(), filter, limit)
	if err != nil {
		writeAnalyticsUseCaseError(response, err)
		return
	}
	writeJSON(response, http.StatusOK, topChannelsResponse(channels))
}

func getTopEmotes(response http.ResponseWriter, request *http.Request, deps Dependencies) {
	if deps.Analytics == nil {
		writeError(response, http.StatusInternalServerError, "Internal server error.")
		return
	}
	filter, limit, err := parseAnalyticsTopRequest(request)
	if err != nil {
		writeAnalyticsFilterError(response, err)
		return
	}
	emotes, err := deps.Analytics.TopEmotes(request.Context(), filter, limit)
	if err != nil {
		writeAnalyticsUseCaseError(response, err)
		return
	}
	writeJSON(response, http.StatusOK, topEmotesResponse(emotes))
}

func parseAnalyticsTopRequest(request *http.Request) (domain.AnalyticsFilter, uint64, error) {
	filter, err := parseAnalyticsFilter(request)
	if err != nil {
		return domain.AnalyticsFilter{}, 0, err
	}
	limit, err := parseAnalyticsLimit(request)
	if err != nil {
		return domain.AnalyticsFilter{}, 0, err
	}
	return filter, limit, nil
}

func parseAnalyticsFilter(request *http.Request) (domain.AnalyticsFilter, error) {
	query := request.URL.Query()
	filter := domain.AnalyticsFilter{}

	var err error
	if filter.Start, err = optionalTime(query.Get("start")); err != nil {
		return domain.AnalyticsFilter{}, err
	}
	if filter.End, err = optionalTime(query.Get("end")); err != nil {
		return domain.AnalyticsFilter{}, err
	}
	if filter.Channel, err = optionalText(query.Get("channel"), 160); err != nil {
		return domain.AnalyticsFilter{}, err
	}
	if filter.Sender, err = optionalText(query.Get("sender"), 160); err != nil {
		return domain.AnalyticsFilter{}, err
	}
	if !filter.Start.IsZero() && !filter.End.IsZero() && filter.Start.After(filter.End) {
		return domain.AnalyticsFilter{}, analyticsusecase.ErrInvalidRange
	}
	return filter, nil
}

func parseAnalyticsBucket(request *http.Request) (domain.AnalyticsBucket, error) {
	raw := strings.TrimSpace(request.URL.Query().Get("bucket"))
	if raw == "" {
		return domain.AnalyticsBucketDay, nil
	}
	bucket := domain.AnalyticsBucket(raw)
	if bucket != domain.AnalyticsBucketHour && bucket != domain.AnalyticsBucketDay {
		return "", errors.New("invalid analytics bucket")
	}
	return bucket, nil
}

func parseAnalyticsLimit(request *http.Request) (uint64, error) {
	raw := strings.TrimSpace(request.URL.Query().Get("limit"))
	if raw == "" {
		return 10, nil
	}
	limit, err := strconv.ParseUint(raw, 10, 64)
	if err != nil || limit < 1 || limit > 100 {
		return 0, errors.New("invalid analytics limit")
	}
	return limit, nil
}

func writeAnalyticsFilterError(response http.ResponseWriter, err error) {
	if errors.Is(err, analyticsusecase.ErrInvalidRange) {
		writeError(response, http.StatusUnprocessableEntity, "Analytics start datetime must be before end datetime.")
		return
	}
	writeError(response, http.StatusUnprocessableEntity, "Invalid query parameters.")
}

func writeAnalyticsUseCaseError(response http.ResponseWriter, err error) {
	if errors.Is(err, analyticsusecase.ErrInvalidRange) {
		writeError(response, http.StatusUnprocessableEntity, "Analytics start datetime must be before end datetime.")
		return
	}
	writeError(response, http.StatusInternalServerError, "Internal server error.")
}
