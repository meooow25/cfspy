package bot

import (
	"encoding/json"
	"net/http"
	"time"

	"github.com/andersfylling/disgord"
)

// This file contains some disgord helpers.

// MsgQueryBuilder returns a message query builder for the given message.
func MsgQueryBuilder(s disgord.Session, msg *disgord.Message) disgord.MessageQueryBuilder {
	return s.Channel(msg.ChannelID).Message(msg.ID)
}

// SuppressEmbeds suppresses all embeds on the given message.
func SuppressEmbeds(session disgord.Session, msg *disgord.Message) (*disgord.Message, error) {
	return MsgQueryBuilder(session, msg).Update().
		Set("flags", msg.Flags|disgord.MessageFlagSupressEmbeds).Execute()
}

// UnsuppressEmbeds unsuppresses all embeds on the given message.
func UnsuppressEmbeds(session disgord.Session, msg *disgord.Message) (*disgord.Message, error) {
	return MsgQueryBuilder(session, msg).Update().
		Set("flags", msg.Flags&^disgord.MessageFlagSupressEmbeds).Execute()
}

// A disgord.HandlerCtrl that can be killed manually.
type manualCtrl struct {
	dead bool
}

func (m *manualCtrl) kill() {
	m.dead = true
}

func (m *manualCtrl) IsDead() bool                     { return m.dead }
func (m *manualCtrl) OnInsert(s disgord.Session) error { return nil }
func (m *manualCtrl) OnRemove(s disgord.Session) error { return nil }
func (m *manualCtrl) Update()                          {}

var _ disgord.HandlerCtrl = (*manualCtrl)(nil)

// https://discord.com/developers/docs/topics/rate-limits#exceeding-a-rate-limit-rate-limit-response-structure
type rateLimitResponse struct {
	RetryAfter float64 `json:"retry_after"`
}

// Runs the given function and retries once if the returned error is caused by exceeding Discord's
// rate limit.
func retryOnRateLimit(do func() error) error {
	err := do()
	if err == nil {
		return nil
	}
	errRest, ok := err.(*disgord.ErrRest)
	if !ok || errRest.HTTPCode != http.StatusTooManyRequests {
		return err
	}
	var r rateLimitResponse
	// Depends on the response body being present as errRest.Suggestion
	if json.Unmarshal([]byte(errRest.Suggestion), &r) != nil || r.RetryAfter <= 0 {
		return err
	}
	wait := time.Duration(r.RetryAfter * float64(time.Second))
	time.Sleep(wait)
	return do()
}
