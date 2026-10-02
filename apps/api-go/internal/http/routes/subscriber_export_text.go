package routes

import (
	"strconv"
	"strings"
	"time"

	"github.com/YSelim0/kick-logs/apps/api-go/internal/domain"
)

type subscriberExportText struct {
	title, channel, filter, gifted, all, generated, total, empty, gifter, start, end string
}

func subscriberTextForLocale(locale string) subscriberExportText {
	switch locale {
	case "en":
		return subscriberExportText{
			title: "Kick Logs Active Subscriber List\n", channel: "Channel: #", filter: "Filter: ",
			gifted: "Gift subscribers", all: "All active subscribers", generated: "Generated: ", total: "Total: ",
			empty:  "No active subscriptions have been recorded for this channel yet.\n",
			gifter: "   Gifted by: ", start: "   Start: ", end: "   End: ",
		}
	case "de":
		return subscriberExportText{
			title: "Kick Logs Liste aktiver Abonnenten\n", channel: "Kanal: #", filter: "Filter: ",
			gifted: "Geschenkabonnenten", all: "Alle aktiven Abonnenten", generated: "Erstellt: ", total: "Gesamt: ",
			empty:  "Für diesen Kanal sind noch keine aktiven Abonnements erfasst.\n",
			gifter: "   Geschenk von: ", start: "   Beginn: ", end: "   Ende: ",
		}
	default:
		return subscriberExportText{
			title: "Kick Logs Aktif Abone Listesi\n", channel: "Kanal: #", filter: "Filtre: ",
			gifted: "Hediye aboneler", all: "Tüm aktif aboneler", generated: "Olusturulma: ", total: "Toplam: ",
			empty:  "Bu kanal için henüz aktif abonelik kaydı yok.\n",
			gifter: "   Hediye eden: ", start: "   Baslangic: ", end: "   Bitis: ",
		}
	}
}

// Keep the historical report byte-for-byte for callers without a locale.
func channelSubscribersTXT(channelSlug string, giftOnly bool, generatedAt time.Time, items []domain.ChannelSubscriber) string {
	return channelSubscribersTXTForLocale(channelSlug, giftOnly, generatedAt, items, "tr")
}

func channelSubscribersTXTForLocale(
	channelSlug string,
	giftOnly bool,
	generatedAt time.Time,
	items []domain.ChannelSubscriber,
	locale string,
) string {
	copy := subscriberTextForLocale(locale)
	var builder strings.Builder
	builder.WriteString(copy.title)
	builder.WriteString(copy.channel)
	builder.WriteString(channelSlug)
	builder.WriteString("\n")
	builder.WriteString(copy.filter)
	if giftOnly {
		builder.WriteString(copy.gifted)
	} else {
		builder.WriteString(copy.all)
	}
	builder.WriteString("\n")
	builder.WriteString(copy.generated)
	builder.WriteString(generatedAt.UTC().Format(time.RFC3339))
	builder.WriteString("\n")
	builder.WriteString(copy.total)
	builder.WriteString(strconv.Itoa(len(items)))
	builder.WriteString("\n\n")

	if len(items) == 0 {
		builder.WriteString(copy.empty)
		return builder.String()
	}

	for i, item := range items {
		builder.WriteString(strconv.Itoa(i + 1))
		builder.WriteString(". ")
		builder.WriteString(item.Username)
		builder.WriteString(" (ID: ")
		builder.WriteString(strconv.FormatInt(item.SubscriberKickUserID, 10))
		builder.WriteString(")")
		if item.Slug != "" {
			builder.WriteString(" - kick.com/")
			builder.WriteString(item.Slug)
		}
		builder.WriteString("\n")
		if item.IsGift && item.GifterUsername != "" {
			builder.WriteString(copy.gifter)
			builder.WriteString(item.GifterUsername)
			builder.WriteString("\n")
		}
		builder.WriteString(copy.start)
		builder.WriteString(item.StartedAt.UTC().Format(time.RFC3339))
		builder.WriteString("\n")
		builder.WriteString(copy.end)
		builder.WriteString(item.ExpiresAt.UTC().Format(time.RFC3339))
		builder.WriteString("\n\n")
	}
	return builder.String()
}
