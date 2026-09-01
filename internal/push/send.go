package push

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/base64"
	"errors"
	"fmt"
	"net/http"
	"strconv"
	"time"
)

// topicFor turns any tag into a legal Topic header (RFC 8030 §5.4: at
// most 32 characters from the URL-safe base64 alphabet). Apple's push
// service answers 400 to anything else — "inbox:<slug>" included — so
// the tag is hashed rather than sent as-is. Same tag, same topic, so the
// service still collapses repeats.
func topicFor(tag string) string {
	if tag == "" {
		return ""
	}
	sum := sha256.Sum256([]byte(tag))
	return base64.RawURLEncoding.EncodeToString(sum[:24])
}

// ErrGone is a subscription the push service no longer knows (404/410):
// the caller deletes it, the browser will re-subscribe on its next visit.
var ErrGone = errors.New("push: subscription gone")

// Target is one browser's subscription as PushManager.subscribe reported it.
type Target struct {
	Endpoint string
	P256dh   string
	Auth     string
}

// Message is what the service worker shows.
type Message struct {
	Title   string `json:"title"`
	Body    string `json:"body,omitempty"`
	Hash    string `json:"hash,omitempty"` // where a tap lands: "#/agent/<id>", "#/inbox/<id>"
	Tag     string `json:"tag,omitempty"`  // collapses repeats of the same subject
	Urgency string `json:"-"`              // very-low | low | normal | high (RFC 8030 §5.3)
}

// Sender posts encrypted messages to push services.
type Sender struct {
	Keys    *Keys
	Subject string // mailto: or https: contact for the push service (RFC 8292 §2.1)
	Client  *http.Client
	Now     func() time.Time
}

// Send encrypts payload for t and posts it. ttl is how long the service
// keeps it for an offline device (seconds).
func (s *Sender) Send(ctx context.Context, t Target, payload []byte, ttl int, urgency, topic string) error {
	if s == nil || s.Keys == nil {
		return errors.New("push: no keys")
	}
	body, err := Encrypt(t.P256dh, t.Auth, payload)
	if err != nil {
		return err
	}
	now := time.Now
	if s.Now != nil {
		now = s.Now
	}
	auth, err := s.Keys.Authorization(t.Endpoint, s.Subject, now())
	if err != nil {
		return err
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, t.Endpoint, bytes.NewReader(body))
	if err != nil {
		return err
	}
	req.Header.Set("Authorization", auth)
	req.Header.Set("Content-Encoding", "aes128gcm")
	req.Header.Set("Content-Type", "application/octet-stream")
	req.Header.Set("TTL", strconv.Itoa(ttl))
	if urgency != "" {
		req.Header.Set("Urgency", urgency)
	}
	if t := topicFor(topic); t != "" {
		req.Header.Set("Topic", t)
	}
	client := s.Client
	if client == nil {
		client = &http.Client{Timeout: 15 * time.Second}
	}
	res, err := client.Do(req)
	if err != nil {
		return err
	}
	defer res.Body.Close()
	switch {
	case res.StatusCode == http.StatusNotFound || res.StatusCode == http.StatusGone:
		return ErrGone
	case res.StatusCode >= 200 && res.StatusCode < 300:
		return nil
	default:
		return fmt.Errorf("push: service answered %d", res.StatusCode)
	}
}
