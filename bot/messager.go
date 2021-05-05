package bot

import (
	"context"

	"github.com/andersfylling/disgord"
)

// Mocks for testing
//go:generate mockgen -destination=mock_bot/mock_messager.go . Messager

// Messager wraps a disgord.Session to do message stuff. Used by widgets.
// Disgord's fluent interface is nice to use but such a pain to mock. Maybe switch to discordgo.
type Messager interface {
	Send(ctx context.Context, channelID disgord.Snowflake, content string, embed *disgord.Embed) (*disgord.Message, error)
	Edit(ctx context.Context, msg *disgord.Message, content string, embed *disgord.Embed) (*disgord.Message, error)
	React(ctx context.Context, msg *disgord.Message, reaction string) error
	Unreact(ctx context.Context, msg *disgord.Message, reaction string) error
	UnreactUser(ctx context.Context, msg *disgord.Message, reaction string, userID disgord.Snowflake) error
	Delete(ctx context.Context, msg *disgord.Message) error
	AddReactListener(filter disgord.Middleware, ctrl disgord.HandlerCtrl, handler disgord.HandlerMessageReactionAdd)
}

type disgordMessager struct {
	session disgord.Session
}

func (m *disgordMessager) Send(
	ctx context.Context,
	channelID disgord.Snowflake,
	content string,
	embed *disgord.Embed,
) (*disgord.Message, error) {
	return m.session.WithContext(ctx).SendMsg(channelID, content, embed)
}

func (m *disgordMessager) Edit(
	ctx context.Context,
	msg *disgord.Message,
	content string,
	embed *disgord.Embed,
) (*disgord.Message, error) {
	return MsgQueryBuilder(m.session, msg).
		WithContext(ctx).
		Update().
		SetContent(content).
		SetEmbed(embed).
		Execute()
}

func (m *disgordMessager) React(ctx context.Context, msg *disgord.Message, reaction string) error {
	// It is possible to get rate limited on react/unreact because react and unreact use the same
	// bucket but disgord doesn't know this until one request of each has been made. If the first
	// request fails by exceeding the rate limit, disgord doesn't auto-retry.
	// After the first request, disgord identifies the bucket from the response headers and rate
	// limits future requests correctly.
	return retryOnRateLimit(func() error { return msg.React(ctx, m.session, reaction) })
}

func (m *disgordMessager) Unreact(
	ctx context.Context,
	msg *disgord.Message,
	reaction string,
) error {
	return retryOnRateLimit(func() error { return msg.Unreact(ctx, m.session, reaction) })
}

func (m *disgordMessager) UnreactUser(
	ctx context.Context,
	msg *disgord.Message,
	reaction string,
	userID disgord.Snowflake,
) error {
	return MsgQueryBuilder(m.session, msg).
		Reaction(reaction).
		WithContext(ctx).
		DeleteUser(userID)
}

func (m *disgordMessager) Delete(ctx context.Context, msg *disgord.Message) error {
	return MsgQueryBuilder(m.session, msg).WithContext(ctx).Delete()
}

func (m *disgordMessager) AddReactListener(
	filter disgord.Middleware,
	ctrl disgord.HandlerCtrl,
	handler disgord.HandlerMessageReactionAdd,
) {
	m.session.Gateway().WithMiddleware(filter).WithCtrl(ctrl).MessageReactionAdd(handler)
}

var _ Messager = (*disgordMessager)(nil)
