package notify

import (
	"strings"
	"testing"
)

func TestBuildMIMEMessage(t *testing.T) {
	message := string(buildMIMEMessage("bot@example.com", "owner@example.com", "Subj", "Body line"))

	for _, want := range []string{
		"From: bot@example.com\r\n",
		"To: owner@example.com\r\n",
		"Subject: Subj\r\n",
		"Content-Type: text/plain; charset=\"utf-8\"\r\n",
		"\r\n\r\nBody line",
	} {
		if !strings.Contains(message, want) {
			t.Fatalf("expected message to contain %q, got:\n%s", want, message)
		}
	}
}
