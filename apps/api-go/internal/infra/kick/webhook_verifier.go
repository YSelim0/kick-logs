package kick

import (
	"crypto"
	"crypto/rsa"
	"crypto/sha256"
	"crypto/x509"
	"encoding/base64"
	"encoding/pem"
	"fmt"
	"strings"
)

// WebhookVerifier verifies RSA-SHA256 signatures on incoming Kick webhook requests.
// The signed message is: messageID + "." + timestamp + "." + raw body.
type WebhookVerifier struct {
	publicKey *rsa.PublicKey
}

func NewWebhookVerifier(publicKeyStr string) (*WebhookVerifier, error) {
	pub, err := parseRSAPublicKey(publicKeyStr)
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
	digest := sha256.Sum256(msg)

	if err := rsa.VerifyPKCS1v15(v.publicKey, crypto.SHA256, digest[:], sigBytes); err != nil {
		return fmt.Errorf("signature verification failed")
	}
	return nil
}

func buildSignedMessage(messageID, timestamp string, body []byte) []byte {
	msg := make([]byte, 0, len(messageID)+len(timestamp)+len(body)+2)
	msg = append(msg, []byte(messageID)...)
	msg = append(msg, '.')
	msg = append(msg, []byte(timestamp)...)
	msg = append(msg, '.')
	msg = append(msg, body...)
	return msg
}

func decodeSignature(sig string) ([]byte, error) {
	sig = strings.TrimSpace(sig)
	if sig == "" {
		return nil, fmt.Errorf("signature is empty")
	}

	decoders := []func(string) ([]byte, error){
		func(s string) ([]byte, error) { return base64.StdEncoding.DecodeString(s) },
		func(s string) ([]byte, error) { return base64.RawStdEncoding.DecodeString(s) },
		func(s string) ([]byte, error) { return base64.URLEncoding.DecodeString(s) },
		func(s string) ([]byte, error) { return base64.RawURLEncoding.DecodeString(s) },
	}
	for _, decode := range decoders {
		if b, err := decode(sig); err == nil && len(b) > 0 {
			return b, nil
		}
	}

	return nil, fmt.Errorf("cannot decode signature as base64 (len=%d)", len(sig))
}

func parseRSAPublicKey(keyStr string) (*rsa.PublicKey, error) {
	keyStr = strings.TrimSpace(keyStr)
	if keyStr == "" {
		return nil, fmt.Errorf("public key is empty")
	}

	derBytes, err := publicKeyDERBytes(keyStr)
	if err != nil {
		return nil, err
	}

	pub, err := x509.ParsePKIXPublicKey(derBytes)
	if err != nil {
		return nil, fmt.Errorf("parse PKIX public key: %w", err)
	}
	rsaKey, ok := pub.(*rsa.PublicKey)
	if !ok {
		return nil, fmt.Errorf("key is not RSA")
	}
	return rsaKey, nil
}

func publicKeyDERBytes(keyStr string) ([]byte, error) {
	if strings.HasPrefix(keyStr, "-----") {
		block, _ := pem.Decode([]byte(keyStr))
		if block == nil {
			return nil, fmt.Errorf("failed to decode PEM block")
		}
		return block.Bytes, nil
	}

	decoders := []func(string) ([]byte, error){
		func(s string) ([]byte, error) { return base64.StdEncoding.DecodeString(s) },
		func(s string) ([]byte, error) { return base64.RawStdEncoding.DecodeString(s) },
		func(s string) ([]byte, error) { return base64.URLEncoding.DecodeString(s) },
		func(s string) ([]byte, error) { return base64.RawURLEncoding.DecodeString(s) },
	}
	for _, decode := range decoders {
		if b, err := decode(keyStr); err == nil && len(b) > 0 {
			return b, nil
		}
	}

	return nil, fmt.Errorf("cannot parse public key: not PEM or base64 (len=%d)", len(keyStr))
}
