package routes

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/YSelim0/kick-logs/apps/api-go/internal/domain"
	homepageusecase "github.com/YSelim0/kick-logs/apps/api-go/internal/usecase/homepage"
)

func TestHomepageDoesNotQueryWhileInitializing(t *testing.T) {
	service := homepageusecase.NewService(nil, nil, nil)
	mux := http.NewServeMux()
	RegisterHomepageRoutes(mux, service)
	response := httptest.NewRecorder()
	mux.ServeHTTP(response, httptest.NewRequest("GET", "/analytics/homepage", nil))
	if response.Code != http.StatusAccepted || response.Header().Get("Retry-After") != "5" || response.Header().Get("Cache-Control") != "no-store" {
		t.Fatalf("response: %d %v %s", response.Code, response.Header(), response.Body.String())
	}
	if response.Body.String() != "{\"status\":\"initializing\",\"retry_after_seconds\":5}\n" {
		t.Fatal(response.Body.String())
	}
}

func TestHomepageReturnsBoundedSnapshotWithExistingAnalyticsShapes(t *testing.T) {
	now := time.Now().UTC().Add(-20 * time.Minute)
	start := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, time.UTC).AddDate(0, 0, -13)
	value := domain.HomepageSnapshot{Version: 1, AsOf: now, Start: start, End: now, Overview: domain.AnalyticsOverview{TotalMessages: 123}}
	for i := range 14 {
		value.Volume = append(value.Volume, domain.MessageVolumePoint{BucketStart: start.AddDate(0, 0, i)})
	}
	service := homepageusecase.NewService(nil, snapshotFixture{value}, nil)
	if err := service.Restore(); err != nil {
		t.Fatal(err)
	}
	mux := http.NewServeMux()
	RegisterHomepageRoutes(mux, service)
	response := httptest.NewRecorder()
	mux.ServeHTTP(response, httptest.NewRequest("GET", "/analytics/homepage", nil))
	var body struct {
		Status   string `json:"status"`
		Stale    bool   `json:"stale"`
		Overview struct {
			TotalMessages int64 `json:"total_messages"`
		} `json:"overview"`
		Volume   []any  `json:"message_volume"`
		Channels []any  `json:"top_channels"`
		Timezone string `json:"timezone"`
	}
	if err := json.Unmarshal(response.Body.Bytes(), &body); err != nil {
		t.Fatal(err)
	}
	if response.Code != 200 || body.Status != "ready" || !body.Stale || body.Overview.TotalMessages != 123 || len(body.Volume) != 14 || body.Channels == nil || body.Timezone != "UTC" {
		t.Fatalf("response: %s", response.Body.String())
	}
}

type snapshotFixture struct{ value domain.HomepageSnapshot }

func (s snapshotFixture) Load() (domain.HomepageSnapshot, error) { return s.value, nil }
func (s snapshotFixture) Save(domain.HomepageSnapshot) error     { return nil }
