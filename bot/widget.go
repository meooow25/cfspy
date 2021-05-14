package bot

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"time"

	"github.com/andersfylling/disgord"
)

// Message is a single Discord message.
type Message struct {
	Content string
	Embed   *disgord.Embed
}

// Page is a single widget page.
type Page struct {
	// The message for this page.
	Default *Message

	// The expanded message for this page, optional.
	Expanded *Message
}

// Pages is a set of pages, numbered 1 to Total. First is shown first.
type Pages struct {
	Get   func(pageNum int) *Page
	Total int
	First int
	Files []disgord.CreateMessageFileParams
}

// MsgCallbackType is the callback function type invoked on message create.
type MsgCallbackType func(*disgord.Message)

// DelCallbackType is the callback function type invoked on delete.
type DelCallbackType func(*disgord.MessageReactionAdd)

// AllowPredicateType is the predicate type that returns whether the operation on react is allowed.
type AllowPredicateType func(*disgord.MessageReactionAdd) bool

// WidgetParams aggregates the params required for a paginated widget.
type WidgetParams struct {
	Pages *Pages

	// Optional callback invoked when the message is created.
	MsgCallback MsgCallbackType

	// After this duration the message will not be monitored.
	Lifetime time.Duration

	// Optional callback to be invoked when the message is deleted.
	DelCallback DelCallbackType

	// Optional check called before performing any operation (delete, prev, next). Defaults to
	// always allowed.
	AllowOp AllowPredicateType
}

const (
	delSymbol  = "🗑"
	prevSymbol = "◀"
	nextSymbol = "▶"
	moreSymbol = "🔽"
	lessSymbol = "🔼"
)

var allSymbols = map[string]bool{
	delSymbol:  true,
	prevSymbol: true,
	nextSymbol: true,
	moreSymbol: true,
	lessSymbol: true,
}

// NewPage returns a Page with a default message and no expanded message.
func NewPage(content string, embed *disgord.Embed) *Page {
	return &Page{
		Default: &Message{
			Content: content,
			Embed:   embed,
		},
	}
}

// NewPageWithExpansion returns a Page with a default message and an expanded message.
func NewPageWithExpansion(
	content string,
	embed *disgord.Embed,
	expandedContent string,
	expandedEmbed *disgord.Embed,
) *Page {
	return &Page{
		Default: &Message{
			Content: content,
			Embed:   embed,
		},
		Expanded: &Message{
			Content: expandedContent,
			Embed:   expandedEmbed,
		},
	}
}

// SendPaginated sends a paginated message.
func SendPaginated(
	ctx context.Context,
	params *WidgetParams,
	session disgord.Session,
	channelID disgord.Snowflake,
) error {
	w := widget{
		params:   params,
		messager: &disgordMessager{session: session},
		logger:   session.Logger(),
	}
	return w.run(ctx, channelID)
}

type widget struct {
	params   *WidgetParams
	messager Messager
	logger   disgord.Logger

	// The active context for the widget
	ctx    context.Context
	cancel context.CancelFunc

	// The widget message
	msg *disgord.Message

	// The current state of the widget
	sync.Mutex
	currentPageNum int
	currentPage    *Page
	expanded       bool
	currentReacts  map[string]bool
}

func (w *widget) run(ctx context.Context, channelID disgord.Snowflake) error {
	if err := w.validateAndUpdateParams(); err != nil {
		return err
	}

	// Initialize context
	w.ctx, w.cancel = context.WithTimeout(ctx, w.params.Lifetime)
	defer w.cancel()

	// Initialize state
	w.currentPageNum = w.params.Pages.First
	w.currentPage = w.params.Pages.Get(w.currentPageNum)
	w.expanded = false
	w.currentReacts = make(map[string]bool)

	// Send page to show first, add reacts
	params := &disgord.CreateMessageParams{
		Content: w.currentPage.Default.Content,
		Embed:   w.currentPage.Default.Embed,
		Files:   w.params.Pages.Files,
	}
	var err error
	w.msg, err = w.messager.Send(w.ctx, channelID, params)
	if err != nil {
		return err
	}
	w.params.MsgCallback(w.msg)
	w.reactOnMsg(delSymbol)
	if w.params.Pages.Total > 1 {
		w.reactOnMsg(prevSymbol)
		w.reactOnMsg(nextSymbol)
	}
	w.fixMoreLessReactsForCurrentPage()

	// Listen for reacts on the message
	var ctrl manualCtrl
	w.messager.AddReactListener(
		filterReactionAddForMsg(w.msg.ID),
		&ctrl,
		func(_ disgord.Session, evt *disgord.MessageReactionAdd) {
			if allSymbols[evt.PartialEmoji.Name] && w.params.AllowOp(evt) {
				go w.handleControlReact(evt)
			}
		},
	)

	<-w.ctx.Done()
	ctrl.kill()
	if w.ctx.Err() == context.DeadlineExceeded && ctx.Err() == nil {
		w.cleanupReacts(ctx)
		return nil
	}
	return ctx.Err()
}

func (w *widget) validateAndUpdateParams() error {
	if w.params.Pages == nil {
		return errors.New("Pages must not be nil")
	}
	if w.params.Pages.Get == nil {
		return errors.New("Pages.Get must not be nil")
	}
	if w.params.Pages.Total < 1 {
		return fmt.Errorf("Pages.Total must be positive, found %v", w.params.Pages.Total)
	}
	if w.params.Pages.First < 1 || w.params.Pages.First > w.params.Pages.Total {
		return fmt.Errorf(
			"Pages.First must be between 1 and %v, found %v",
			w.params.Pages.Total, w.params.Pages.First)
	}
	if w.params.MsgCallback == nil {
		w.params.MsgCallback = func(*disgord.Message) {}
	}
	if w.params.Lifetime <= 0 {
		return fmt.Errorf("Lifetime must be positive, found %v", w.params.Lifetime)
	}
	if w.params.DelCallback == nil {
		w.params.DelCallback = func(*disgord.MessageReactionAdd) {}
	}
	if w.params.AllowOp == nil {
		w.params.AllowOp = func(*disgord.MessageReactionAdd) bool { return true }
	}
	return nil
}

func (w *widget) reactOnMsg(react string) {
	if err := w.messager.React(w.ctx, w.msg, react); err != nil {
		w.logger.Error(fmt.Errorf("React %q failed: %w", react, err))
		return
	}
	w.currentReacts[react] = true
}

func (w *widget) unreactOnMsg(react string) {
	if err := w.messager.Unreact(w.ctx, w.msg, react); err != nil {
		w.logger.Error(fmt.Errorf("Unreact %q failed: %w", react, err))
		return
	}
	delete(w.currentReacts, react)
}

func (w *widget) cleanupReacts(ctx context.Context) {
	for react := range w.currentReacts {
		if err := w.messager.Unreact(ctx, w.msg, react); err != nil {
			w.logger.Error(fmt.Errorf("Clean up react %q failed: %w", react, err))
		}
	}
}

func (w *widget) fixMoreLessReactsForCurrentPage() {
	reacts := []string{moreSymbol, lessSymbol}
	want := make(map[string]bool)
	if w.currentPage.Expanded != nil {
		if w.expanded {
			want[lessSymbol] = true
		} else {
			want[moreSymbol] = true
		}
	}
	// Remove first, then add
	for _, react := range reacts {
		if w.currentReacts[react] && !want[react] {
			w.unreactOnMsg(react)
		}
	}
	for _, react := range reacts {
		if !w.currentReacts[react] && want[react] {
			w.reactOnMsg(react)
		}
	}
}

func (w *widget) expandCurrentPage() {
	if w.currentPage.Expanded == nil || w.expanded {
		return
	}
	_, err := w.messager.Edit(
		w.ctx, w.msg, w.currentPage.Expanded.Content, w.currentPage.Expanded.Embed)
	if err != nil {
		w.logger.Error(fmt.Errorf("Failed to expand page: %w", err))
		return
	}
	w.expanded = true
	w.fixMoreLessReactsForCurrentPage()
}

func (w *widget) contractCurrentPage() {
	if w.currentPage.Expanded == nil || !w.expanded {
		return
	}
	_, err := w.messager.Edit(
		w.ctx, w.msg, w.currentPage.Default.Content, w.currentPage.Default.Embed)
	if err != nil {
		w.logger.Error(fmt.Errorf("Failed to contract page: %w", err))
		return
	}
	w.expanded = false
	w.fixMoreLessReactsForCurrentPage()
}

func (w *widget) showPage(delta int) {
	newPageNum := w.currentPageNum + delta
	if newPageNum < 1 || newPageNum > w.params.Pages.Total {
		return
	}
	newPage := w.params.Pages.Get(newPageNum)
	_, err := w.messager.Edit(w.ctx, w.msg, newPage.Default.Content, newPage.Default.Embed)
	if err != nil {
		w.logger.Error(fmt.Errorf("Failed to show page: %w", err))
		return
	}
	w.currentPageNum = newPageNum
	w.currentPage = newPage
	w.expanded = false
	w.fixMoreLessReactsForCurrentPage()
}

func (w *widget) handleControlReact(evt *disgord.MessageReactionAdd) {
	w.Lock()
	defer w.Unlock()

	react := evt.PartialEmoji.Name
	if react == delSymbol {
		w.messager.Delete(w.ctx, w.msg)
		w.params.DelCallback(evt)
		w.cancel()
		return
	}

	go w.messager.UnreactUser(w.ctx, w.msg, react, evt.UserID)

	switch evt.PartialEmoji.Name {
	case prevSymbol:
		w.showPage(-1)
	case nextSymbol:
		w.showPage(+1)
	case moreSymbol:
		w.expandCurrentPage()
	case lessSymbol:
		w.contractCurrentPage()
	default:
		w.logger.Error(fmt.Errorf("Unexpected react %v", react))
	}
}
