package bot

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"time"

	"github.com/andersfylling/disgord"
)

// PageGetter is a function type that returns a page given the page number.
type PageGetter func(int) (string, *disgord.Embed)

// MsgCallbackType is the callback function type invoked on message create.
type MsgCallbackType func(*disgord.Message)

// DelCallbackType is the callback function type invoked on delete.
type DelCallbackType func(*disgord.MessageReactionAdd)

// AllowPredicateType is the predicate type that returns whether the operation on react is allowed.
type AllowPredicateType func(*disgord.MessageReactionAdd) bool

// PaginateParams aggregates the params required for a paginated message.
type PaginateParams struct {
	// Should return the page corresponding to the given page number.
	GetPage PageGetter

	NumPages        int
	PageToShowFirst int

	// Optional callback invoked when the message is created.
	MsgCallback MsgCallbackType

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

	// Optional callback invoked when the message is created.
	MsgCallback MsgCallbackType

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
) error {
	if err := validateAndUpdate(&params, session); err != nil {
		return err
	}

	currentPage := params.PageToShowFirst
	content, embed := params.GetPage(currentPage)
	msg, err := session.WithContext(ctx).SendMsg(channelID, content, embed)
	if err != nil {
		return err
	}
	params.MsgCallback(msg)

	if params.DelBtn {
		msg.React(ctx, session, delSymbol)
	}
	if params.NumPages > 1 {
		msg.React(ctx, session, prevSymbol)
		msg.React(ctx, session, nextSymbol)
	}
	cleanupReacts := func() {
		if params.DelBtn {
			msg.Unreact(ctx, session, delSymbol)
		}
		if params.NumPages > 1 {
			msg.Unreact(ctx, session, prevSymbol)
			msg.Unreact(ctx, session, nextSymbol)
		}
	}

	// This mutex guards against concurrent attempts to update the currently shown page.
	var currentPageLock sync.Mutex

	ctxWidgetActive, cancelWidgetCtx := context.WithTimeout(ctx, params.DeactivateAfter)
	defer cancelWidgetCtx()

	showPage := func(delta int) {
		currentPageLock.Lock()
		defer currentPageLock.Unlock()
		newPage := currentPage + delta
		if newPage < 1 || newPage > params.NumPages {
			return
		}
		content, embed := params.GetPage(newPage)
		_, err := QueryBuilderFor(session, msg).WithContext(ctxWidgetActive).Update().
			SetContent(content).SetEmbed(embed).Execute()
		if err == nil {
			currentPage = newPage
		}
	}

	reactMap := map[string]func(*disgord.MessageReactionAdd){
		delSymbol: func(evt *disgord.MessageReactionAdd) {
			QueryBuilderFor(session, msg).WithContext(ctxWidgetActive).Delete()
			params.DelCallback(evt)
			cancelWidgetCtx()
		},
		prevSymbol: func(evt *disgord.MessageReactionAdd) {
			go QueryBuilderFor(session, msg).WithContext(ctxWidgetActive).
				Reaction(prevSymbol).DeleteUser(evt.UserID)
			showPage(-1)
		},
		nextSymbol: func(evt *disgord.MessageReactionAdd) {
			go QueryBuilderFor(session, msg).WithContext(ctxWidgetActive).
				Reaction(nextSymbol).DeleteUser(evt.UserID)
			showPage(+1)
		},
	}

	reactionAddCh := make(chan *disgord.MessageReactionAdd)
	ctrl := &disgord.Ctrl{Channel: reactionAddCh}
	session.Gateway().
		WithMiddleware(filterReactionAddForMsg(msg.ID)).
		WithCtrl(ctrl).
		MessageReactionAddChan(reactionAddCh)

	for {
		select {
		case evt := <-reactionAddCh:
			if handler, ok := reactMap[evt.PartialEmoji.Name]; ok && params.AllowOp(evt) {
				go handler(evt)
			}
		case <-ctxWidgetActive.Done():
			ctrl.CloseChannel()
			if ctxWidgetActive.Err() == context.DeadlineExceeded && ctx.Err() == nil {
				cleanupReacts()
				return nil
			}
			return ctx.Err()
		}
	}
}

// SendWithDelBtn sends a message and adds a delete button to it.
func SendWithDelBtn(
	ctx context.Context,
	params OnePageWithDelParams,
	session disgord.Session,
	channelID disgord.Snowflake,
) error {
	return SendPaginated(
		ctx,
		PaginateParams{
			GetPage: func(int) (string, *disgord.Embed) {
				return params.Content, params.Embed
			},
			NumPages:        1,
			PageToShowFirst: 1,
			MsgCallback:     params.MsgCallback,
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
	if params.MsgCallback == nil {
		params.MsgCallback = func(*disgord.Message) {}
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
	return nil
}
