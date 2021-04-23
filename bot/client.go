package bot

import (
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

var _ disgord.HandlerCtrl = &manualCtrl{}
