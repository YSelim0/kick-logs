package kick

import (
	"crypto/ed25519"
	"crypto/x509"
	"encoding/base64"
	"encoding/hex"
	"encoding/pem"
	"fmt"
	"strings"
)

// WebhookVerifier verifies Ed25519 signatures on incoming Kick webhook requests.
// The signed message is: messageID + timestamp + raw body (concatenated bytes).
// The signature is hex-encoded; base64 is accepted as a fallback.
type WebhookVerifier struct {
	publicKey ed25519.PublicKey
}

func NewWebhookVerifier(publicKeyStr string) (*WebhookVerifier, error) {
	pub, err := parseEd25519PublicKey(publicKeyStr)
	if err != nil {
		return nil, fmt.Errorf("parse Kick webhook public key: %w", err)
	}
	return &WebhookVerifier{publicKey: pub}, nil
}

func (v *WebhookVerifier) Verify(messageID, timestamp string, body []byte, signature string) error {
	sigBytes, err := decodeSignature(signature)
	if err != nil {
		return fmt.Errorf("decode signature: %w", err)
	}

	msg := buildSignedMessage(messageID, timestamp, body)

	if !ed25519.Verify(v.publicKey, msg, sigBytes) {
		return fmt.Errorf("signature verification failed")
	}
	return nil
}

func buildSignedMessage(messageID, timestamp string, body []byte) []byte {
	msg := make([]byte, 0, len(messageID)+len(timestamp)+len(body))
	msg = append(msg, []byte(messageID)...)
	msg = append(msg, []byte(timestamp)...)
	msg = append(msg, body...)
	return msg
}

func decodeSignature(sig string) ([]byte, error) {
	sig = strings.TrimSpace(sig)

	if b, err := hex.DecodeString(sig); err == nil {
		if len(b) == ed25519.SignatureSize {
			return b, nil
		}
	}

	decoders := []func(string) ([]byte, error){
		func(s string) ([]byte, error) { return base64.StdEncoding.DecodeString(s) },
		func(s string) ([]byte, error) { return base64.RawStdEncoding.DecodeString(s) },
		func(s string) ([]byte, error) { return base64.URLEncoding.DecodeString(s) },
		func(s string) ([]byte, error) { return base64.RawURLEncoding.DecodeString(s) },
	}
	for _, decode := range decoders {
		if b, err := decode(sig); err == nil && len(b) == ed25519.SignatureSize {
			return b, nil
		}
	}

	return nil, fmt.Errorf("cannot decode signature as hex or base64 (len=%d)", len(sig))
}

func parseEd25519PublicKey(keyStr string) (ed25519.PublicKey, error) {
	keyStr = strings.TrimSpace(keyStr)
	if keyStr == "" {
		return nil, fmt.Errorf("public key is empty")
	}

	if strings.HasPrefix(keyStr, "-----") {
		block, _ := pem.Decode([]byte(keyStr))
		if block == nil {
			return nil, fmt.Errorf("failed to decode PEM block")
		}
		pub, err := x509.ParsePKIXPublicKey(block.Bytes)
		if err != nil {
			return nil, fmt.Errorf("parse PKIX public key: %w", err)
		}
		ed, ok := pub.(ed25519.PublicKey)
		if !ok {
			return nil, fmt.Errorf("key is not Ed25519")
		}
		return ed, nil
	}

	decoders := []func(string) ([]byte, error){
		func(s string) ([]byte, error) { return base64.StdEncoding.DecodeString(s) },
		func(s string) ([]byte, error) { return base64.RawStdEncoding.DecodeString(s) },
		func(s string) ([]byte, error) { return base64.URLEncoding.DecodeString(s) },
		func(s string) ([]byte, error) { return base64.RawURLEncoding.DecodeString(s) },
		func(s string) ([]byte, error) { return hex.DecodeString(s) },
	}
	for _, decode := range decoders {
		if b, err := decode(keyStr); err == nil && len(b) == ed25519.PublicKeySize {
			return ed25519.PublicKey(b), nil
		}
	}

	return nil, fmt.Errorf("cannot parse public key: not PEM, base64, or hex (len=%d)", len(keyStr))
}
