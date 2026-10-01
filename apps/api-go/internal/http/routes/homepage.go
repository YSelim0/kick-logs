package routes

import (
	"net/http"
	"time"

	"github.com/YSelim0/kick-logs/apps/api-go/internal/http/schemas"
	homepageusecase "github.com/YSelim0/kick-logs/apps/api-go/internal/usecase/homepage"
)

func RegisterHomepageRoutes(mux *http.ServeMux, service *homepageusecase.Service) {
	mux.HandleFunc("GET /analytics/homepage", func(response http.ResponseWriter, request *http.Request) {
		response.Header().Set("Cache-Control", "no-store")
		if service == nil {
			writeError(response, http.StatusServiceUnavailable, "Analytics unavailable.")
			return
		}
		snapshot, err := service.Get()
		if err != nil {
			response.Header().Set("Retry-After", "5")
			writeJSON(response, http.StatusAccepted, schemas.HomepagePendingResponse{Status: "initializing", RetryAfterSeconds: 5})
			return
		}
		writeJSON(response, http.StatusOK, schemas.HomepageResponse{
			Status: "ready", AsOf: snapshot.AsOf.Format(time.RFC3339),
			Start: snapshot.Start.Format(time.RFC3339), End: snapshot.End.Format(time.RFC3339),
			Timezone: "UTC", Stale: time.Since(snapshot.AsOf) >= homepageusecase.RefreshInterval,
			Overview:      analyticsOverviewResponse(snapshot.Overview),
			MessageVolume: messageVolumeResponse(snapshot.Volume).Items,
			TopChannels:   topChannelsResponse(snapshot.TopChannels).Items,
			TopSenders:    topSendersResponse(snapshot.TopSenders).Items,
			TopEmotes:     topEmotesResponse(snapshot.TopEmotes).Items,
		})
	})
}
