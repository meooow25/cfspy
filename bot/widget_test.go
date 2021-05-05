package bot

import (
	"context"
	"testing"
	"time"

	"github.com/andersfylling/disgord"
	"github.com/golang/mock/gomock"

	"github.com/meooow25/cfspy/bot/mock_bot"
)

var (
	testChannelID = disgord.Snowflake(8723564)
	testMsg       = &disgord.Message{ID: 238648}
	testUserID    = disgord.Snowflake(5982314)
	testPages     = []*Page{
		nil, // 1-indexed
		NewPage("test1", &disgord.Embed{Title: "embed1"}),
		NewPage("test2", &disgord.Embed{Title: "embed2"}),
		NewPageWithExpansion(
			"test3", &disgord.Embed{Title: "embed3"},
			"test3 expanded", &disgord.Embed{Title: "embed3 expanded"},
		),
		NewPageWithExpansion(
			"test4", &disgord.Embed{Title: "embed4"},
			"test4 expanded", &disgord.Embed{Title: "embed4 expanded"},
		),
	}
)

func newWidget(
	pages int,
	lifetime time.Duration,
	msgCallback MsgCallbackType,
	delCallback DelCallbackType,
	allowOp AllowPredicateType,
	messager *mock_bot.MockMessager,
) *widget {
	return &widget{
		params: &WidgetParams{
			Pages: &Pages{
				Get:   func(i int) *Page { return testPages[i] },
				Total: pages,
				First: pages,
			},
			Lifetime:    lifetime,
			MsgCallback: msgCallback,
			DelCallback: delCallback,
			AllowOp:     allowOp,
		},
		messager: messager,
	}
}

// Runs the widget in a separate goroutine.
func runWidget(ctx context.Context, w *widget) <-chan error {
	done := make(chan error)
	go func() { done <- w.run(ctx, testChannelID) }()
	return done
}

func msgReactionAdd(reaction string) *disgord.MessageReactionAdd {
	return &disgord.MessageReactionAdd{
		PartialEmoji: &disgord.Emoji{Name: reaction},
		UserID:       testUserID,
	}
}

func registerCallCheckCleanup(t *testing.T, got *int, want int) {
	t.Cleanup(func() {
		if *got != want {
			t.Fatalf("got %v calls, want %v", *got, want)
		}
	})
}

func newMsgCallback(t *testing.T, expectedCalls int) MsgCallbackType {
	calls := 0
	cb := func(m *disgord.Message) {
		if m != testMsg {
			t.Fatalf("got %v, want testMsg", m)
		}
		calls++
	}
	registerCallCheckCleanup(t, &calls, expectedCalls)
	return cb
}

func newDelCallback(t *testing.T, expectedCalls int) DelCallbackType {
	calls := 0
	cb := func(evt *disgord.MessageReactionAdd) {
		reaction := evt.PartialEmoji.Name
		if reaction != delSymbol {
			t.Fatalf("got %v, want %v", reaction, delSymbol)
		}
		calls++
	}
	registerCallCheckCleanup(t, &calls, expectedCalls)
	return cb
}

func newAllowOp(t *testing.T, expectedCalls int) AllowPredicateType {
	calls := 0
	cb := func(evt *disgord.MessageReactionAdd) bool {
		calls++
		return true
	}
	registerCallCheckCleanup(t, &calls, expectedCalls)
	return cb
}

// Matches a context.Context that has not crossed its deadline or been cancelled.
type activeContextMatcher struct{}

func (m activeContextMatcher) Matches(x interface{}) bool {
	if ctx, ok := x.(context.Context); ok {
		return ctx.Err() == nil
	}
	return false
}

func (m activeContextMatcher) String() string {
	return "is an active context"
}

var _ gomock.Matcher = activeContextMatcher{}

// Exposes convenient functions to record calls to a mock messager.
type messagerCalls struct {
	messager *mock_bot.MockMessager
}

func (c *messagerCalls) send(content string, embed *disgord.Embed) *gomock.Call {
	return c.messager.EXPECT().
		Send(activeContextMatcher{}, testChannelID, content, embed).
		Return(testMsg, nil)
}

func (c *messagerCalls) edit(content string, embed *disgord.Embed) *gomock.Call {
	return c.messager.EXPECT().
		Edit(activeContextMatcher{}, testMsg, content, embed).
		Return(testMsg, nil)
}

func (c *messagerCalls) react(reaction string) *gomock.Call {
	return c.messager.EXPECT().React(activeContextMatcher{}, testMsg, reaction)
}

func (c *messagerCalls) unreact(reaction string) *gomock.Call {
	return c.messager.EXPECT().Unreact(activeContextMatcher{}, testMsg, reaction)
}

func (c *messagerCalls) unreactUser(reaction string, userID disgord.Snowflake) *gomock.Call {
	return c.messager.EXPECT().UnreactUser(activeContextMatcher{}, testMsg, reaction, userID)
}

func (c *messagerCalls) delete() *gomock.Call {
	return c.messager.EXPECT().Delete(activeContextMatcher{}, testMsg)
}

func (c *messagerCalls) reactListener(ch chan<- disgord.HandlerMessageReactionAdd) *gomock.Call {
	return c.messager.EXPECT().
		AddReactListener(gomock.Any(), gomock.Any(), gomock.Any()).
		Do(func(_ disgord.Middleware, _ disgord.HandlerCtrl, handler disgord.HandlerMessageReactionAdd) {
			if ch != nil {
				ch <- handler
			}
		})
}

func TestWidgetOnePage(t *testing.T) {
	ctrl := gomock.NewController(t)
	messager := mock_bot.NewMockMessager(ctrl)
	calls := messagerCalls{messager}
	gomock.InOrder(
		calls.send(testPages[1].Default.Content, testPages[1].Default.Embed),
		calls.react(delSymbol),
		calls.reactListener(nil),
		calls.unreact(delSymbol),
	)
	msgCallback := newMsgCallback(t, 1)
	delCallback := newDelCallback(t, 0)
	allowOp := newAllowOp(t, 0)

	w := newWidget(1, time.Millisecond, msgCallback, delCallback, allowOp, messager)
	if err := w.run(context.Background(), testChannelID); err != nil {
		t.Fatal(err)
	}
}

func TestWidgetMultiPage(t *testing.T) {
	ctrl := gomock.NewController(t)
	messager := mock_bot.NewMockMessager(ctrl)
	calls := messagerCalls{messager}
	inOrder(
		calls.send(testPages[2].Default.Content, testPages[2].Default.Embed),
		calls.react(delSymbol),
		calls.react(prevSymbol),
		calls.react(nextSymbol),
		calls.reactListener(nil),
		anyOrder(
			calls.unreact(nextSymbol),
			calls.unreact(prevSymbol),
			calls.unreact(delSymbol),
		),
	)
	msgCallback := newMsgCallback(t, 1)
	delCallback := newDelCallback(t, 0)
	allowOp := newAllowOp(t, 0)

	w := newWidget(2, time.Millisecond, msgCallback, delCallback, allowOp, messager)
	if err := w.run(context.Background(), testChannelID); err != nil {
		t.Fatal(err)
	}
}

func TestWidgetCancel(t *testing.T) {
	ctrl := gomock.NewController(t)
	messager := mock_bot.NewMockMessager(ctrl)
	calls := messagerCalls{messager}
	chk, _, _, _, _ := newCheckpoints()
	inOrder(
		calls.send(testPages[2].Default.Content, testPages[2].Default.Embed),
		calls.react(delSymbol),
		calls.react(prevSymbol),
		calls.react(nextSymbol),
		calls.reactListener(nil),
		chk,
	)
	msgCallback := newMsgCallback(t, 1)
	delCallback := newDelCallback(t, 0)
	allowOp := newAllowOp(t, 0)

	w := newWidget(2, time.Minute, msgCallback, delCallback, allowOp, messager)

	ctx, cancel := context.WithCancel(context.Background())
	done := runWidget(ctx, w)
	<-chk
	cancel()
	select {
	case err := <-done:
		if err != context.Canceled {
			t.Fatalf("got %v, want context.Canceled", err)
		}
	case <-time.After(time.Second):
		t.Fatal("widget did not stop when canceled")
	}
}

func TestWidgetDelete(t *testing.T) {
	ctrl := gomock.NewController(t)
	messager := mock_bot.NewMockMessager(ctrl)
	calls := messagerCalls{messager}
	handlerCh := make(chan disgord.HandlerMessageReactionAdd, 1)
	gomock.InOrder(
		calls.send(testPages[2].Default.Content, testPages[2].Default.Embed),
		calls.react(delSymbol),
		calls.react(prevSymbol),
		calls.react(nextSymbol),
		calls.reactListener(handlerCh),
		calls.delete(),
	)
	msgCallback := newMsgCallback(t, 1)
	delCallback := newDelCallback(t, 1)
	allowOp := newAllowOp(t, 1)

	w := newWidget(2, time.Minute, msgCallback, delCallback, allowOp, messager)

	done := runWidget(context.Background(), w)

	handler := <-handlerCh
	handler(nil, msgReactionAdd(delSymbol))

	select {
	case err := <-done:
		if err != nil {
			t.Fatal(err)
		}
	case <-time.After(time.Second):
		t.Fatal("widget did not stop when deleted")
	}
}

func TestWidgetAllowOp(t *testing.T) {
	ctrl := gomock.NewController(t)
	messager := mock_bot.NewMockMessager(ctrl)
	calls := messagerCalls{messager}
	handlerCh := make(chan disgord.HandlerMessageReactionAdd, 1)
	chk, _, _, _, _ := newCheckpoints()

	inOrder(
		calls.send(testPages[2].Default.Content, testPages[2].Default.Embed),
		calls.react(delSymbol),
		calls.react(prevSymbol),
		calls.react(nextSymbol),
		calls.reactListener(handlerCh),

		// Previous, 2 -> 1
		anyOrder(
			calls.unreactUser(prevSymbol, testUserID),
			calls.edit(testPages[1].Default.Content, testPages[1].Default.Embed),
		),
		chk,

		// Delete
		calls.delete(),
	)

	msgCallback := newMsgCallback(t, 1)
	delCallback := newDelCallback(t, 1)
	allow := false
	allowOp := func(*disgord.MessageReactionAdd) bool {
		return allow
	}

	w := newWidget(2, time.Minute, msgCallback, delCallback, allowOp, messager)
	done := runWidget(context.Background(), w)

	handler := <-handlerCh

	// Not allowed
	handler(nil, msgReactionAdd(prevSymbol))
	handler(nil, msgReactionAdd(nextSymbol))
	handler(nil, msgReactionAdd(delSymbol))

	// Previous, 2 -> 1
	allow = true
	handler(nil, msgReactionAdd(prevSymbol))
	<-chk

	// Not allowed
	allow = false
	handler(nil, msgReactionAdd(prevSymbol))
	handler(nil, msgReactionAdd(nextSymbol))
	handler(nil, msgReactionAdd(delSymbol))

	// Delete
	allow = true
	handler(nil, msgReactionAdd(delSymbol))

	if err := <-done; err != nil {
		t.Fatal(err)
	}
}

func TestWidgetChangePage(t *testing.T) {
	ctrl := gomock.NewController(t)
	messager := mock_bot.NewMockMessager(ctrl)
	calls := messagerCalls{messager}
	handlerCh := make(chan disgord.HandlerMessageReactionAdd, 1)
	chk1, chk2, chk3, chk4, _ := newCheckpoints()

	inOrder(
		calls.send(testPages[2].Default.Content, testPages[2].Default.Embed),
		calls.react(delSymbol),
		calls.react(prevSymbol),
		calls.react(nextSymbol),
		calls.reactListener(handlerCh),

		// Previous, 2 -> 1
		anyOrder(
			calls.unreactUser(prevSymbol, testUserID),
			calls.edit(testPages[1].Default.Content, testPages[1].Default.Embed),
		),
		chk1,

		// Previous, 1 -> 1
		calls.unreactUser(prevSymbol, testUserID),
		chk2,

		// Next, 1 -> 2
		anyOrder(
			calls.unreactUser(nextSymbol, testUserID),
			calls.edit(testPages[2].Default.Content, testPages[2].Default.Embed),
		),
		chk3,

		// Next, 2 -> 2
		calls.unreactUser(nextSymbol, testUserID),
		chk4,
	)

	msgCallback := newMsgCallback(t, 1)
	delCallback := newDelCallback(t, 0)
	allowOp := newAllowOp(t, 4)

	w := newWidget(2, time.Minute, msgCallback, delCallback, allowOp, messager)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	runWidget(ctx, w)

	handler := <-handlerCh

	// Previous, 2 -> 1
	handler(nil, msgReactionAdd(prevSymbol))
	<-chk1

	// Previous, 1 -> 1
	handler(nil, msgReactionAdd(prevSymbol))
	<-chk2

	// Next, 1 -> 2
	handler(nil, msgReactionAdd(nextSymbol))
	<-chk3

	// Next, 2 -> 2
	handler(nil, msgReactionAdd(nextSymbol))
	<-chk4
}

func TestWidgetExpandContract(t *testing.T) {
	ctrl := gomock.NewController(t)
	messager := mock_bot.NewMockMessager(ctrl)
	calls := messagerCalls{messager}
	handlerCh := make(chan disgord.HandlerMessageReactionAdd, 1)
	chk1, chk2, chk3, chk4, _ := newCheckpoints()

	inOrder(
		calls.send(testPages[3].Default.Content, testPages[3].Default.Embed),
		calls.react(delSymbol),
		calls.react(prevSymbol),
		calls.react(nextSymbol),
		calls.react(moreSymbol),
		calls.reactListener(handlerCh),

		// Expand, default -> expanded
		anyOrder(
			calls.unreactUser(moreSymbol, testUserID),
			inOrder(
				calls.edit(testPages[3].Expanded.Content, testPages[3].Expanded.Embed),
				calls.unreact(moreSymbol),
				calls.react(lessSymbol),
			),
		),
		chk1,

		// Expand, expanded -> expanded
		calls.unreactUser(moreSymbol, testUserID),
		chk2,

		// Contract, expanded -> default
		anyOrder(
			calls.unreactUser(lessSymbol, testUserID),
			inOrder(
				calls.edit(testPages[3].Default.Content, testPages[3].Default.Embed),
				calls.unreact(lessSymbol),
				calls.react(moreSymbol),
			),
		),
		chk3,

		// Contract, default -> default
		calls.unreactUser(lessSymbol, testUserID),
		chk4,
	)

	msgCallback := newMsgCallback(t, 1)
	delCallback := newDelCallback(t, 0)
	allowOp := newAllowOp(t, 4)

	w := newWidget(3, time.Minute, msgCallback, delCallback, allowOp, messager)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	runWidget(ctx, w)

	handler := <-handlerCh

	// Expand, default -> expanded
	handler(nil, msgReactionAdd(moreSymbol))
	<-chk1

	// Expand, expanded -> expanded
	handler(nil, msgReactionAdd(moreSymbol))
	<-chk2

	// Contract, expanded -> default
	handler(nil, msgReactionAdd(lessSymbol))
	<-chk3

	// Contract, default -> default
	handler(nil, msgReactionAdd(lessSymbol))
	<-chk4
}

func TestWidgetMixedActions(t *testing.T) {
	ctrl := gomock.NewController(t)
	messager := mock_bot.NewMockMessager(ctrl)
	calls := messagerCalls{messager}
	handlerCh := make(chan disgord.HandlerMessageReactionAdd, 1)
	chk1, chk2, chk3, chk4, chk5 := newCheckpoints()

	inOrder(
		calls.send(testPages[4].Default.Content, testPages[4].Default.Embed),
		calls.react(delSymbol),
		calls.react(prevSymbol),
		calls.react(nextSymbol),
		calls.react(moreSymbol),
		calls.reactListener(handlerCh),

		// Expand 4, default -> expanded
		anyOrder(
			calls.unreactUser(moreSymbol, testUserID),
			inOrder(
				calls.edit(testPages[4].Expanded.Content, testPages[4].Expanded.Embed),
				calls.unreact(moreSymbol),
				calls.react(lessSymbol),
			),
		),
		chk1,

		// Previous, 4 -> 3
		anyOrder(
			calls.unreactUser(prevSymbol, testUserID),
			inOrder(
				calls.edit(testPages[3].Default.Content, testPages[3].Default.Embed),
				calls.unreact(lessSymbol),
				calls.react(moreSymbol),
			),
		),
		chk2,

		// Previous, 3 -> 2
		anyOrder(
			calls.unreactUser(prevSymbol, testUserID),
			inOrder(
				calls.edit(testPages[2].Default.Content, testPages[2].Default.Embed),
				calls.unreact(moreSymbol),
			),
		),
		chk3,

		// Next, 2 -> 3
		anyOrder(
			calls.unreactUser(nextSymbol, testUserID),
			inOrder(
				calls.edit(testPages[3].Default.Content, testPages[3].Default.Embed),
				calls.react(moreSymbol),
			),
		),
		chk4,

		// Next, 3 -> 4
		anyOrder(
			calls.unreactUser(nextSymbol, testUserID),
			calls.edit(testPages[4].Default.Content, testPages[4].Default.Embed),
		),
		chk5,

		calls.delete(),
	)

	msgCallback := newMsgCallback(t, 1)
	delCallback := newDelCallback(t, 1)
	allowOp := newAllowOp(t, 6)

	w := newWidget(4, time.Minute, msgCallback, delCallback, allowOp, messager)
	done := runWidget(context.Background(), w)

	handler := <-handlerCh

	// Expand 4, default -> expanded
	handler(nil, msgReactionAdd(moreSymbol))
	<-chk1

	// Previous, 4 -> 3
	handler(nil, msgReactionAdd(prevSymbol))
	<-chk2

	// Previous, 3 -> 2
	handler(nil, msgReactionAdd(prevSymbol))
	<-chk3

	// Next, 2 -> 3
	handler(nil, msgReactionAdd(nextSymbol))
	<-chk4

	// Next, 3 -> 4
	handler(nil, msgReactionAdd(nextSymbol))
	<-chk5

	// Delete
	handler(nil, msgReactionAdd(delSymbol))

	if err := <-done; err != nil {
		t.Fatal(err)
	}
}
