package zammad

import (
	"crypto/hmac"
	"crypto/sha1"
	"encoding/hex"
	"fmt"
	"time"
)

// WebhookPayload is the inbound payload from a Zammad webhook.
type WebhookPayload struct {
	Event   string         `json:"event"`
	Ticket  *WebhookTicket `json:"ticket"`
	Article *Article       `json:"article,omitempty"`
}

// WebhookTicket is the ticket data included in webhook payloads.
type WebhookTicket struct {
	ID        int       `json:"id"`
	Number    string    `json:"number"`
	StateID   int       `json:"state_id"`
	State     string    `json:"state"`
	Priority  string    `json:"priority"`
	Group     string    `json:"group"`
	UpdatedAt time.Time `json:"updated_at"`
}

// ValidateWebhookSignature validates the HMAC-SHA1 signature from Zammad.
// The signature header format is "sha1=<hex>".
func ValidateWebhookSignature(body []byte, secret, signature string) error {
	if len(signature) < 5 || signature[:5] != "sha1=" {
		return fmt.Errorf("invalid signature format")
	}

	expectedSig := signature[5:]

	mac := hmac.New(sha1.New, []byte(secret))
	mac.Write(body)
	computedSig := hex.EncodeToString(mac.Sum(nil))

	if !hmac.Equal([]byte(computedSig), []byte(expectedSig)) {
		return fmt.Errorf("signature mismatch")
	}

	return nil
}

// ComputeWebhookSignature computes the HMAC-SHA1 signature for a payload.
// Returns the full header value: "sha1=<hex>".
func ComputeWebhookSignature(body []byte, secret string) string {
	mac := hmac.New(sha1.New, []byte(secret))
	mac.Write(body)
	return "sha1=" + hex.EncodeToString(mac.Sum(nil))
}
