package bot

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"time"

	"github.com/andersfylling/disgord"
)

// PageGetter is a type alias for a function that returns a page given the page number.
type PageGetter = func(int) (string, *disgord.Embed)

// DelCallbackType is a type alias for the callback function invoked on delete.
type DelCallbackType = func(*disgord.MessageReactionAdd)

// AllowPredicateType is a type alias for the predicate that returns whether the operation on react
// is allowed.
type AllowPredicateType = func(*disgord.MessageReactionAdd) bool

// PaginateParams aggregates the params required for a paginated message.
type PaginateParams struct {
	// Should return the page corresponding to the given page number. Will not be called
	// concurrently.
	GetPage PageGetter

	NumPages        int
	PageToShowFirst int

	// After this duration the message will not be monitored.
	DeactivateAfter time.Duration

	// Whether to add a delete button.
	DelBtn bool

	// Optional callback to be invoked when the message is deleted.
	DelCallback DelCallbackType

	// Optional check called before performing any operation (delete, prev, next). Defaults to
	// always allowed.
	AllowOp AllowPredicateType
}

// OnePageWithDelParams aggregates the params required for a single message with a delete button.
type OnePageWithDelParams struct {
	// The content and embed of the message to send.
	Content string
	Embed   *disgord.Embed

	// After this duration the message will not be monitored.
	DeactivateAfter time.Duration

	// Optional callback to be invoked when the message is deleted.
	DelCallback DelCallbackType

	// Optional check called before performing delete. Defaults to always allowed.
	AllowOp AllowPredicateType
}

const (
	delSymbol  = "🗑"
	prevSymbol = "◀"
	nextSymbol = "▶"
)

// SendPaginated sends a paginated message.
func SendPaginated(
	ctx context.Context,
	params PaginateParams,
	session disgord.Session,
	channelID disgord.Snowflake,
) (*disgord.Message, error) {
	if err := validateAndUpdate(&params, session); err != nil {
		return nil, err
	}

	currentPage := params.PageToShowFirst
	content, embed := params.GetPage(currentPage)
	msg, err := session.SendMsg(ctx, channelID, content, embed)
	if err != nil {
		return nil, err
	}
	if params.DelBtn {
		msg.React(ctx, session, delSymbol)
	}
	if params.NumPages > 1 {
		msg.React(ctx, session, prevSymbol)
		msg.React(ctx, session, nextSymbol)
	}

	// This mutex guards against concurrent attempts to update the currently shown page.
	var currentPageLock sync.Mutex

	showPage := func(ctx context.Context, delta int) {
		currentPageLock.Lock()
		defer currentPageLock.Unlock()
		newPage := currentPage + delta
		if newPage < 1 || newPage > params.NumPages {
			return
		}
		currentPage = newPage
		content, embed := params.GetPage(currentPage)
		session.UpdateMessage(ctx, channelID, msg.ID).SetContent(content).SetEmbed(embed).Execute()
	}

	ctx, cancel := context.WithCancel(ctx)
	reactMap := map[string]func(*disgord.MessageReactionAdd){
		delSymbol: func(evt *disgord.MessageReactionAdd) {
			go session.DeleteFromDiscord(evt.Ctx, msg)
			cancel()
			params.DelCallback(evt)
		},
		prevSymbol: func(evt *disgord.MessageReactionAdd) {
			go session.DeleteUserReaction(evt.Ctx, channelID, msg.ID, evt.UserID, prevSymbol)
			showPage(evt.Ctx, -1)
		},
		nextSymbol: func(evt *disgord.MessageReactionAdd) {
			go session.DeleteUserReaction(evt.Ctx, channelID, msg.ID, evt.UserID, nextSymbol)
			showPage(evt.Ctx, +1)
		},
	}

	reactionAddCh := make(chan *disgord.MessageReactionAdd)
	ctrl := &disgord.Ctrl{Channel: reactionAddCh}
	session.On(disgord.EvtMessageReactionAdd, reactionAddCh, ctrl)

	go func() {
		for {
			select {
			case evt := <-reactionAddCh:
				if evt.MessageID != msg.ID {
					break
				}
				if handler, ok := reactMap[evt.PartialEmoji.Name]; ok && params.AllowOp(evt) {
					go handler(evt)
				}
			case <-time.After(params.DeactivateAfter):
				ctrl.CloseChannel()
				if params.DelBtn {
					session.DeleteOwnReaction(ctx, channelID, msg.ID, delSymbol)
				}
				if params.NumPages > 1 {
					session.DeleteOwnReaction(ctx, channelID, msg.ID, prevSymbol)
					session.DeleteOwnReaction(ctx, channelID, msg.ID, nextSymbol)
				}
				return
			case <-ctx.Done():
				ctrl.CloseChannel()
				return
			}
		}
	}()
	return msg, nil
}

// SendWithDelBtn sends a message and adds a delete button to it.
func SendWithDelBtn(
	ctx context.Context,
	params OnePageWithDelParams,
	session disgord.Session,
	channelID disgord.Snowflake,
) (*disgord.Message, error) {
	return SendPaginated(
		ctx,
		PaginateParams{
			GetPage: func(int) (string, *disgord.Embed) {
				return params.Content, params.Embed
			},
			NumPages:        1,
			PageToShowFirst: 1,
			DeactivateAfter: params.DeactivateAfter,
			DelBtn:          true,
			DelCallback:     params.DelCallback,
			AllowOp:         params.AllowOp,
		},
		session,
		channelID,
	)
}

func validateAndUpdate(params *PaginateParams, session disgord.Session) error {
	if params.GetPage == nil {
		return errors.New("GetPage must not be nil")
	}
	if params.NumPages < 1 {
		return fmt.Errorf("NumPages must be positive, found %v", params.NumPages)
	}
	if params.PageToShowFirst < 1 || params.PageToShowFirst > params.NumPages {
		return fmt.Errorf(
			"PageToShowFirst must be between 1 and %v, found %v",
			params.NumPages, params.PageToShowFirst)
	}
	if params.DeactivateAfter < time.Second {
		return fmt.Errorf("DeactivateAfter must be at least 1s, found %v", params.DeactivateAfter)
	}
	if params.DelBtn && params.DelCallback == nil {
		params.DelCallback = func(*disgord.MessageReactionAdd) {}
	}
	if params.AllowOp == nil {
		params.AllowOp = func(*disgord.MessageReactionAdd) bool { return true }
	}
	allowCheck := params.AllowOp
	params.AllowOp = func(evt *disgord.MessageReactionAdd) bool {
		if user, err := session.GetUser(evt.Ctx, evt.UserID); err == nil {
			return !user.Bot && allowCheck(evt)
		}
		return false
	}
	return nil
}
