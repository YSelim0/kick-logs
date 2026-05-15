package kick

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/gorilla/websocket"

	"github.com/YSelim0/kick-logs/apps/api-go/internal/domain"
)

type PusherClient struct {
	url    string
	dialer *websocket.Dialer
}

func NewPusherClient(url string) *PusherClient {
	return &PusherClient{
		url: url,
		dialer: &websocket.Dialer{
			HandshakeTimeout: 15 * time.Second,
			Proxy:            http.ProxyFromEnvironment,
		},
	}
}

func (client *PusherClient) Listen(
	ctx context.Context,
	channels []domain.ListenerChannel,
	handle func(string) error,
) error {
	conn, _, err := client.dialer.DialContext(ctx, client.url, nil)
	if err != nil {
		return fmt.Errorf("connect Kick Pusher websocket: %w", err)
	}
	defer conn.Close()

	go func() {
		<-ctx.Done()
		_ = conn.WriteControl(
			websocket.CloseMessage,
			websocket.FormatCloseMessage(websocket.CloseNormalClosure, "context done"),
			time.Now().Add(time.Second),
		)
		_ = conn.Close()
	}()

	for _, channel := range channels {
		for _, subscription := range subscriptionMessages(channel) {
			if err := conn.WriteJSON(subscription); err != nil {
				return fmt.Errorf("subscribe Kick Pusher channel: %w", err)
			}
		}
	}

	for {
		messageType, payload, err := conn.ReadMessage()
		if err != nil {
			if ctx.Err() != nil {
				return ctx.Err()
			}
			return fmt.Errorf("read Kick Pusher websocket: %w", err)
		}
		if messageType != websocket.TextMessage && messageType != websocket.BinaryMessage {
			continue
		}
		if err := handle(string(payload)); err != nil {
			return err
		}
	}
}

type subscriptionMessage struct {
	Event string `json:"event"`
	Data  struct {
		Auth    string `json:"auth"`
		Channel string `json:"channel"`
	} `json:"data"`
}

func subscriptionMessages(channel domain.ListenerChannel) []subscriptionMessage {
	values := make([]subscriptionMessage, 0, 2)
	if channel.KickChatroomID > 0 {
		values = append(values, newSubscriptionMessage(fmt.Sprintf("chatrooms.%d.v2", channel.KickChatroomID)))
	}
	if channel.KickChannelID > 0 {
		values = append(values, newSubscriptionMessage(fmt.Sprintf("channel.%d", channel.KickChannelID)))
	}
	return values
}

func newSubscriptionMessage(channel string) subscriptionMessage {
	var message subscriptionMessage
	message.Event = "pusher:subscribe"
	message.Data.Auth = ""
	message.Data.Channel = channel
	return message
}

func marshalSubscription(message subscriptionMessage) string {
	encoded, _ := json.Marshal(message)
	return string(encoded)
}
