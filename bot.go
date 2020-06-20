package main

import (
	"context"
	"fmt"
	"regexp"
	"strings"
	"time"

	"github.com/andersfylling/disgord"
)

// Bot is a simple wrapper over a disgord Client. It is not protected by a mutex. Is is ok to
// concurrently modify the internal disgord client.
type Bot struct {
	Client      *disgord.Client
	Info        BotInfo
	commands    map[string]*Command
	helpCommand *Command
}

// BotInfo wraps some bot info.
type BotInfo struct {
	Name   string
	Token  string
	Prefix string
	Desc   string
	Logger disgord.Logger
}

// BotContext is passed to all command handlers and contains fields relevant to the current command
// invocation.
type BotContext struct {
	Session disgord.Session
	Message *disgord.Message
	Command *Command
	Args    []string
	Ctx     context.Context
	Logger  disgord.Logger
}

// Command represents a bot command.
type Command struct {
	ID      string
	Usage   string
	Desc    string
	Handler func(BotContext)
}

// Used for parsing args, the string `hello "wor ld"` will be parsed to ["hello", "wor ld"]
var argsRe = regexp.MustCompile(`"([^"]+)"|([^\s]+)`)

// NewBot creates a new bot with the given BotInfo.
func NewBot(info BotInfo) *Bot {
	bot := Bot{
		Client: disgord.New(
			disgord.Config{
				BotToken: info.Token,
				Logger:   info.Logger,
			},
		),
		Info:     info,
		commands: make(map[string]*Command),
	}
	bot.helpCommand = &Command{
		ID:      "help",
		Desc:    "Shows the bot help message",
		Handler: bot.sendHelp,
	}
	bot.Client.On(disgord.EvtMessageCreate, bot.maybeHandleCommand)
	return &bot
}

// OnMessageCreate attaches a handler that is called on the message create event.
func (bot *Bot) OnMessageCreate(handler func(BotContext, *disgord.MessageCreate)) {
	wrapped := func(s disgord.Session, evt *disgord.MessageCreate) {
		ctx := BotContext{
			Session: s,
			Message: evt.Message,
			Ctx:     evt.Ctx,
			Logger:  s.Logger(),
		}
		handler(ctx, evt)
	}
	bot.Client.On(disgord.EvtMessageCreate, wrapped)
}

// AddCommand adds a Command to the bot.
func (bot *Bot) AddCommand(command *Command) {
	bot.commands[command.ID] = command
}

func (bot *Bot) maybeHandleCommand(s disgord.Session, evt *disgord.MessageCreate) {
	msg := evt.Message
	if msg.Author == nil || msg.Author.Bot {
		return
	}
	if !strings.HasPrefix(msg.Content, bot.Info.Prefix) {
		return
	}
	argsNested := argsRe.FindAllStringSubmatch(msg.Content, -1)
	args := make([]string, len(argsNested))
	for i := range argsNested {
		args[i] = argsNested[i][0]
	}
	commandID := args[0][len(bot.Info.Prefix):]
	ctx := BotContext{
		Session: s,
		Message: msg,
		Args:    args,
		Ctx:     evt.Ctx,
		Logger:  s.Logger(),
	}
	if command, ok := bot.commands[commandID]; ok {
		ctx.Logger.Info("Dispatching command: ", commandID)
		ctx.Command = command
		command.Handler(ctx)
	} else if commandID == bot.helpCommand.ID {
		ctx.Logger.Info("Dispatching help command")
		ctx.Command = bot.helpCommand
		bot.helpCommand.Handler(ctx)
	} else {
		bot.rejectCommand(ctx, commandID)
	}
}

func (bot *Bot) sendHelp(ctx BotContext) {
	go func() {
		if len(ctx.Args) > 1 {
			ctx.SendIncorrectUsageMsg()
			return
		}
		ctx.Send(bot.buildHelpEmbed())
	}()
}

func (bot *Bot) rejectCommand(ctx BotContext, commandID string) {
	var content string
	switch commandID {
	case "":
		content = fmt.Sprintf(
			"Missing command, send `%v` for help",
			bot.addPrefix(bot.helpCommand))
	default:
		content = fmt.Sprintf(
			"Unknown command `%v`, send `%v` for help",
			commandID, bot.addPrefix(bot.helpCommand))
	}
	ctx.Send(content)
}

func (bot *Bot) addPrefix(command *Command) string {
	return bot.Info.Prefix + command.FullUsage()
}

func (bot *Bot) buildHelpEmbed() *disgord.Embed {
	var fields []*disgord.EmbedField
	for _, command := range bot.commands {
		fields = append(fields, &disgord.EmbedField{
			Name:  command.FullUsage(),
			Value: command.Desc,
		})
	}
	fields = append(fields, &disgord.EmbedField{
		Name:  bot.helpCommand.FullUsage(),
		Value: bot.helpCommand.Desc,
	})
	embed := disgord.Embed{
		Title:       bot.Info.Name,
		Description: bot.Info.Desc,
		Fields:      fields,
	}
	return &embed
}

// Send sends a message in the current channel.
func (ctx *BotContext) Send(data ...interface{}) (*disgord.Message, error) {
	return ctx.Session.SendMsg(ctx.Ctx, ctx.Message.ChannelID, data...)
}

// SendTimed sends a message and deletes it after a delay. Ignores any error if the delete fails.
func (ctx *BotContext) SendTimed(
	deleteAfter time.Duration,
	data ...interface{},
) (*disgord.Message, error) {
	msg, err := ctx.Send(data...)
	if err == nil {
		time.AfterFunc(deleteAfter, func() { ctx.DeleteMsg(msg) })
	}
	return msg, err
}

// EditMsg edits a message to set the given string as content.
func (ctx *BotContext) EditMsg(msg *disgord.Message, content string) (*disgord.Message, error) {
	return ctx.Session.SetMsgContent(ctx.Ctx, msg.ChannelID, msg.ID, content)
}

// SuppressEmbeds suppresses all embeds on the given message.
func (ctx *BotContext) SuppressEmbeds(msg *disgord.Message) (*disgord.Message, error) {
	return ctx.Session.UpdateMessage(
		ctx.Ctx,
		msg.ChannelID,
		msg.ID,
	).Set(
		"flags",
		msg.Flags|disgord.MessageFlagSupressEmbeds,
	).Execute()
}

// UnsuppressEmbeds unsuppresses all embeds on the given message.
func (ctx *BotContext) UnsuppressEmbeds(msg *disgord.Message) (*disgord.Message, error) {
	return ctx.Session.UpdateMessage(
		ctx.Ctx,
		msg.ChannelID,
		msg.ID,
	).Set(
		"flags",
		msg.Flags&^disgord.MessageFlagSupressEmbeds,
	).Execute()
}

// DeleteMsg deletes the given message.
func (ctx *BotContext) DeleteMsg(msg *disgord.Message) error {
	return ctx.Session.DeleteFromDiscord(ctx.Ctx, msg)
}

// React reacts on the given message with the given emoji.
func (ctx *BotContext) React(msg *disgord.Message, emoji interface{}) error {
	return msg.React(ctx.Ctx, ctx.Session, emoji)
}

// SendIncorrectUsageMsg sends the incorrect usage message for the current command.
func (ctx *BotContext) SendIncorrectUsageMsg() (*disgord.Message, error) {
	return ctx.Send(ctx.Command.IncorrectUsageMsg())
}

// FullUsage returns the "ID Usage" form of the command.
func (com *Command) FullUsage() string {
	if com.Usage == "" {
		return com.ID
	}
	return com.ID + " " + com.Usage
}

// IncorrectUsageMsg returns a message that may be sent if the command is invoked with incorrect
// arguments.
func (com *Command) IncorrectUsageMsg() string {
	return fmt.Sprintf("Incorrect usage, expected `%v`", com.FullUsage())
}
