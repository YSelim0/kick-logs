package routes

import (
	"strings"
	"testing"
	"time"

	"github.com/YSelim0/kick-logs/apps/api-go/internal/domain"
)

func TestChannelSubscribersTXTLocales(t *testing.T) {
	now := time.Date(2026, 10, 1, 12, 0, 0, 0, time.UTC)
	items := []domain.ChannelSubscriber{{SubscriberKickUserID: 42, Username: "example_user", Slug: "example-user", IsGift: true, GifterUsername: "Heaven", StartedAt: now, ExpiresAt: now.Add(31 * 24 * time.Hour)}}
	for _, tc := range []struct{ locale, title, gift, empty string }{
		{"tr", "Kick Logs Aktif Abone Listesi", "Hediye aboneler", "Bu kanal için henüz aktif abonelik kaydı yok."},
		{"en", "Kick Logs Active Subscriber List", "Gift subscribers", "No active subscriptions have been recorded for this channel yet."},
		{"de", "Kick Logs Liste aktiver Abonnenten", "Geschenkabonnenten", "Für diesen Kanal sind noch keine aktiven Abonnements erfasst."},
	} {
		t.Run(tc.locale, func(t *testing.T) {
			output := channelSubscribersTXTForLocale("Heaven", true, now, items, tc.locale)
			for _, want := range []string{tc.title, tc.gift, "#Heaven", "example_user", "Heaven", "kick.com/example-user", "2026-10-01T12:00:00Z", "2026-11-01T12:00:00Z"} {
				if !strings.Contains(output, want) {
					t.Errorf("missing %q in %s", want, output)
				}
			}
			if !strings.Contains(channelSubscribersTXTForLocale("Heaven", false, now, nil, tc.locale), tc.empty) {
				t.Fatal("missing localized empty state")
			}
		})
	}
	legacy := channelSubscribersTXT("Heaven", false, now, nil)
	expected := "Kick Logs Aktif Abone Listesi\nKanal: #Heaven\nFiltre: Tüm aktif aboneler\nOlusturulma: 2026-10-01T12:00:00Z\nToplam: 0\n\nBu kanal için henüz aktif abonelik kaydı yok.\n"
	if legacy != expected || legacy != channelSubscribersTXTForLocale("Heaven", false, now, nil, "tr") {
		t.Fatalf("legacy output changed: %s", legacy)
	}
}
