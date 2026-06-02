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
		handled, err := handlePusherProtocolMessage(conn, payload)
		if err != nil {
			return err
		}
		if handled {
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
	values := make([]subscriptionMessage, 0, 3)
	if channel.KickChatroomID > 0 {
		values = append(values, newSubscriptionMessage(fmt.Sprintf("chatrooms.%d.v2", channel.KickChatroomID)))
		values = append(values, newSubscriptionMessage(fmt.Sprintf("chatrooms.%d", channel.KickChatroomID)))
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

type pusherProtocolEnvelope struct {
	Event string          `json:"event"`
	Data  json.RawMessage `json:"data"`
}

func handlePusherProtocolMessage(conn *websocket.Conn, payload []byte) (bool, error) {
	var envelope pusherProtocolEnvelope
	if err := json.Unmarshal(payload, &envelope); err != nil {
		return false, nil
	}

	switch envelope.Event {
	case "pusher:ping":
		if err := conn.WriteJSON(map[string]any{
			"event": "pusher:pong",
			"data":  map[string]any{},
		}); err != nil {
			return true, fmt.Errorf("send Kick Pusher pong: %w", err)
		}
		return true, nil
	case "pusher_internal:subscription_succeeded", "pusher:connection_established":
		return true, nil
	case "pusher:error":
		return true, fmt.Errorf("Kick Pusher protocol error: %s", string(envelope.Data))
	default:
		return false, nil
	}
}
