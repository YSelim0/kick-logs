package httpapi

import (
	"context"
	"database/sql"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/YSelim0/kick-logs/apps/api-go/internal/config"
	"github.com/YSelim0/kick-logs/apps/api-go/internal/domain"
	"github.com/YSelim0/kick-logs/apps/api-go/internal/http/routes"
	analyticsusecase "github.com/YSelim0/kick-logs/apps/api-go/internal/usecase/analytics"
	profilesusecase "github.com/YSelim0/kick-logs/apps/api-go/internal/usecase/profiles"
)

func TestAnalyticsRoutesReturnPublicAggregates(t *testing.T) {
	handler := newAnalyticsProfileTestRouter()

	overviewResponse := httptest.NewRecorder()
	handler.ServeHTTP(
		overviewResponse,
		httptest.NewRequest(http.MethodGet, "/analytics/overview?sender=profile-user&channel=hype", nil),
	)
	if overviewResponse.Code != http.StatusOK {
		t.Fatalf("overview status = %d body = %s", overviewResponse.Code, overviewResponse.Body.String())
	}
	if !strings.Contains(overviewResponse.Body.String(), `"total_messages":3`) {
		t.Fatalf("overview body = %s", overviewResponse.Body.String())
	}

	volumeResponse := httptest.NewRecorder()
	handler.ServeHTTP(volumeResponse, httptest.NewRequest(http.MethodGet, "/analytics/message-volume?bucket=hour", nil))
	if volumeResponse.Code != http.StatusOK || !strings.Contains(volumeResponse.Body.String(), `"message_count":2`) {
		t.Fatalf("volume status = %d body = %s", volumeResponse.Code, volumeResponse.Body.String())
	}

	topResponse := httptest.NewRecorder()
	handler.ServeHTTP(topResponse, httptest.NewRequest(http.MethodGet, "/analytics/top-emotes?limit=1", nil))
	if topResponse.Code != http.StatusOK || !strings.Contains(topResponse.Body.String(), `"usage_count":2`) {
		t.Fatalf("top status = %d body = %s", topResponse.Code, topResponse.Body.String())
	}
}

func TestAnalyticsRoutesRejectInvalidRange(t *testing.T) {
	handler := newAnalyticsProfileTestRouter()

	response := httptest.NewRecorder()
	handler.ServeHTTP(
		response,
		httptest.NewRequest(
			http.MethodGet,
			"/analytics/overview?start=2040-01-02T00:00:00Z&end=2040-01-01T00:00:00Z",
			nil,
		),
	)
	if response.Code != http.StatusUnprocessableEntity {
		t.Fatalf("status = %d body = %s", response.Code, response.Body.String())
	}
	if !strings.Contains(response.Body.String(), "Analytics start datetime must be before end datetime.") {
		t.Fatalf("body = %s", response.Body.String())
	}
}

func TestProfileRoutesReturnProfilesAndNotFound(t *testing.T) {
	handler := newAnalyticsProfileTestRouter()

	userResponse := httptest.NewRecorder()
	handler.ServeHTTP(userResponse, httptest.NewRequest(http.MethodGet, "/users/profile-user/analytics", nil))
	if userResponse.Code != http.StatusOK {
		t.Fatalf("user status = %d body = %s", userResponse.Code, userResponse.Body.String())
	}
	if !strings.Contains(userResponse.Body.String(), `"username":"profile_user"`) ||
		!strings.Contains(userResponse.Body.String(), `"latest_messages"`) {
		t.Fatalf("user body = %s", userResponse.Body.String())
	}

	channelResponse := httptest.NewRecorder()
	handler.ServeHTTP(channelResponse, httptest.NewRequest(http.MethodGet, "/channels/hype/analytics", nil))
	if channelResponse.Code != http.StatusOK {
		t.Fatalf("channel status = %d body = %s", channelResponse.Code, channelResponse.Body.String())
	}
	if !strings.Contains(channelResponse.Body.String(), `"display_name":"Hype"`) ||
		!strings.Contains(channelResponse.Body.String(), `"top_senders"`) {
		t.Fatalf("channel body = %s", channelResponse.Body.String())
	}

	missingResponse := httptest.NewRecorder()
	handler.ServeHTTP(missingResponse, httptest.NewRequest(http.MethodGet, "/users/missing/analytics", nil))
	if missingResponse.Code != http.StatusNotFound ||
		!strings.Contains(missingResponse.Body.String(), "Sender profile not found.") {
		t.Fatalf("missing status = %d body = %s", missingResponse.Code, missingResponse.Body.String())
	}
}

func newAnalyticsProfileTestRouter() http.Handler {
	cfg := config.Config{BackendCORSOrigins: []string{"http://localhost:3000"}}
	analyticsRepo := newFakeAnalyticsRepository()
	channelRepo := newFakeProfileChannelRepository()
	senderRepo := newFakeProfileSenderRepository()
	return NewRouter(cfg, slog.New(slog.NewTextHandler(io.Discard, nil)), routes.Dependencies{
		Config:    cfg,
		Analytics: analyticsusecase.NewService(analyticsRepo),
		Profiles:  profilesusecase.NewService(analyticsRepo, channelRepo, senderRepo),
	})
}

type fakeAnalyticsRepository struct {
	now time.Time
}

func newFakeAnalyticsRepository() *fakeAnalyticsRepository {
	return &fakeAnalyticsRepository{now: time.Date(2040, 5, 1, 10, 0, 0, 0, time.UTC)}
}

func (repo *fakeAnalyticsRepository) Overview(
	_ context.Context,
	_ domain.AnalyticsFilter,
) (domain.AnalyticsOverview, error) {
	return domain.AnalyticsOverview{
		TotalMessages:    3,
		TotalSenders:     1,
		TotalChannels:    1,
		TotalEmoteUsages: 2,
		FirstMessageAt:   repo.now,
		LatestMessageAt:  repo.now.Add(time.Hour),
	}, nil
}

func (repo *fakeAnalyticsRepository) MessageVolume(
	_ context.Context,
	_ domain.AnalyticsFilter,
	_ domain.AnalyticsBucket,
) ([]domain.MessageVolumePoint, error) {
	return []domain.MessageVolumePoint{{BucketStart: repo.now, MessageCount: 2}}, nil
}

func (repo *fakeAnalyticsRepository) TopSenders(
	_ context.Context,
	_ domain.AnalyticsFilter,
	_ uint64,
) ([]domain.TopSenderAnalytics, error) {
	return []domain.TopSenderAnalytics{{
		SenderID:        10,
		KickUserID:      100,
		Username:        "profile_user",
		Slug:            "profile_user",
		ProfileImageURL: "https://example.com/profile.png",
		MessageCount:    3,
		FirstMessageAt:  repo.now,
		LatestMessageAt: repo.now.Add(time.Hour),
	}}, nil
}

func (repo *fakeAnalyticsRepository) TopChannels(
	_ context.Context,
	_ domain.AnalyticsFilter,
	_ uint64,
) ([]domain.TopChannelAnalytics, error) {
	return []domain.TopChannelAnalytics{{
		ChannelID:       1,
		Slug:            "hype",
		DisplayName:     "Hype",
		ProfileImageURL: "https://example.com/hype.png",
		BannerImageURL:  "https://example.com/hype-banner.png",
		MessageCount:    3,
		FirstMessageAt:  repo.now,
		LatestMessageAt: repo.now.Add(time.Hour),
	}}, nil
}

func (repo *fakeAnalyticsRepository) TopEmotes(
	_ context.Context,
	_ domain.AnalyticsFilter,
	_ uint64,
) ([]domain.TopEmoteAnalytics, error) {
	return []domain.TopEmoteAnalytics{{
		ID:           "111",
		Name:         "Kappa",
		Token:        "[emote:111:Kappa]",
		ImageURL:     "https://files.kick.com/emotes/111/fullsize",
		UsageCount:   2,
		MessageCount: 2,
	}}, nil
}

func (repo *fakeAnalyticsRepository) LatestMessages(
	_ context.Context,
	_ domain.AnalyticsFilter,
	_ uint64,
) ([]domain.ChatMessage, error) {
	return []domain.ChatMessage{{
		ID:                 500,
		KickMessageID:      "profile-message",
		ChannelID:          1,
		ChannelKickID:      1001,
		ChannelChatroomID:  2001,
		ChannelSlug:        "hype",
		ChannelDisplayName: "Hype",
		SenderID:           10,
		SenderKickID:       100,
		SenderUsername:     "profile_user",
		SenderSlug:         "profile_user",
		SenderBadgesJSON:   "[]",
		MessageType:        "message",
		Content:            "hello profile",
		ReplyMetadataJSON:  "{}",
		RawPayloadJSON:     "{}",
		MessageCreatedAt:   repo.now.Add(time.Hour),
		IngestedAt:         repo.now.Add(time.Hour + time.Second),
	}}, nil
}

type fakeProfileSenderRepository struct {
	sender domain.SenderProfile
}

func newFakeProfileSenderRepository() *fakeProfileSenderRepository {
	return &fakeProfileSenderRepository{sender: domain.SenderProfile{
		ID:              10,
		KickUserID:      100,
		Username:        "profile_user",
		Slug:            "profile_user",
		ProfileImageURL: "https://example.com/profile.png",
	}}
}

func (repo *fakeProfileSenderRepository) Upsert(
	_ context.Context,
	sender domain.SenderProfile,
) (domain.SenderProfile, error) {
	repo.sender = sender
	return sender, nil
}

func (repo *fakeProfileSenderRepository) GetByKickUserID(
	_ context.Context,
	kickUserID int64,
) (domain.SenderProfile, error) {
	if repo.sender.KickUserID == kickUserID {
		return repo.sender, nil
	}
	return domain.SenderProfile{}, sql.ErrNoRows
}

func (repo *fakeProfileSenderRepository) GetBySlug(
	_ context.Context,
	slug string,
) (domain.SenderProfile, error) {
	if repo.sender.Slug == slug {
		return repo.sender, nil
	}
	return domain.SenderProfile{}, sql.ErrNoRows
}

type fakeProfileChannelRepository struct {
	channel domain.FollowedChannel
}

func newFakeProfileChannelRepository() *fakeProfileChannelRepository {
	return &fakeProfileChannelRepository{channel: domain.FollowedChannel{
		ID:              1,
		KickChannelID:   1001,
		KickChatroomID:  2001,
		Slug:            "hype",
		DisplayName:     "Hype",
		ProfileImageURL: "https://example.com/hype.png",
		BannerImageURL:  "https://example.com/hype-banner.png",
		IsEnabled:       true,
	}}
}

func (repo *fakeProfileChannelRepository) Upsert(
	_ context.Context,
	channel domain.FollowedChannel,
) (domain.FollowedChannel, error) {
	repo.channel = channel
	return channel, nil
}

func (repo *fakeProfileChannelRepository) GetByID(
	_ context.Context,
	id int64,
) (domain.FollowedChannel, error) {
	if repo.channel.ID == id {
		return repo.channel, nil
	}
	return domain.FollowedChannel{}, sql.ErrNoRows
}

func (repo *fakeProfileChannelRepository) GetBySlug(
	_ context.Context,
	slug string,
) (domain.FollowedChannel, error) {
	if repo.channel.Slug == slug {
		return repo.channel, nil
	}
	return domain.FollowedChannel{}, sql.ErrNoRows
}

func (repo *fakeProfileChannelRepository) GetByChatroomID(
	_ context.Context,
	chatroomID int64,
) (domain.FollowedChannel, error) {
	if repo.channel.KickChatroomID == chatroomID {
		return repo.channel, nil
	}
	return domain.FollowedChannel{}, sql.ErrNoRows
}

func (repo *fakeProfileChannelRepository) GetByBroadcasterUserID(
	_ context.Context,
	broadcasterUserID int64,
) (domain.FollowedChannel, error) {
	if repo.channel.BroadcasterUserID == broadcasterUserID {
		return repo.channel, nil
	}
	return domain.FollowedChannel{}, sql.ErrNoRows
}

func (repo *fakeProfileChannelRepository) List(_ context.Context) ([]domain.FollowedChannel, error) {
	return []domain.FollowedChannel{repo.channel}, nil
}

func (repo *fakeProfileChannelRepository) ListEnabled(
	_ context.Context,
) ([]domain.FollowedChannel, error) {
	return []domain.FollowedChannel{repo.channel}, nil
}

func (repo *fakeProfileChannelRepository) Disable(
	_ context.Context,
	_ int64,
) (domain.FollowedChannel, error) {
	repo.channel.IsEnabled = false
	return repo.channel, nil
}
