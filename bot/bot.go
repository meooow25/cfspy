package bot

import (
	"fmt"
	"regexp"

	"github.com/andersfylling/disgord"
	"github.com/andersfylling/disgord/std"
)

// Bot is a simple wrapper over a disgord Client. It is not protected by a mutex. Is is ok to
// concurrently modify the internal disgord client.
type Bot struct {
	Client      *disgord.Client
	Info        Info
	commands    map[string]*Command
	commandList []*Command
	helpCommand *Command
}

// Info wraps some bot info.
type Info struct {
	disgord.Config
	Name       string
	Prefix     string
	Desc       string
	SupportURL string
}

// Command represents a bot command.
type Command struct {
	ID      string
	Usage   string
	Desc    string
	Handler func(Context)
}

// Used for parsing args, the string `hello "wor ld"` will be parsed to ["hello", "wor ld"]
var argsRe = regexp.MustCompile(`"([^"]+)"|([^\s]+)`)

// New creates a new bot with the given BotInfo.
func New(info Info) *Bot {
	bot := Bot{
		Client:   disgord.New(info.Config),
		Info:     info,
		commands: make(map[string]*Command),
	}
	bot.helpCommand = &Command{
		ID:      "help",
		Desc:    "Shows the bot help message",
		Handler: bot.sendHelp,
	}
	bot.Client.Gateway().
		WithMiddleware(
			filterMsgCreateNotBot, std.CopyMsgEvt, filterMsgCreateStripPrefix(bot.Info.Prefix)).
		MessageCreate(bot.maybeHandleCommand)
	return &bot
}

// OnMessageCreate attaches a handler that is called on the message create event.
func (bot *Bot) OnMessageCreate(handler func(Context, *disgord.MessageCreate)) {
	wrapped := func(s disgord.Session, evt *disgord.MessageCreate) {
		ctx := Context{
			Bot:     bot,
			Session: s,
			Message: evt.Message,
			Logger:  s.Logger(),
		}
		handler(ctx, evt)
	}
	bot.Client.Gateway().WithMiddleware(filterMsgCreateNotBot).MessageCreate(wrapped)
}

// AddCommand adds a Command to the bot.
func (bot *Bot) AddCommand(command *Command) {
	bot.commands[command.ID] = command
	bot.commandList = append(bot.commandList, command)
}

func (bot *Bot) maybeHandleCommand(s disgord.Session, evt *disgord.MessageCreate) {
	ctx := Context{
		Bot:     bot,
		Session: s,
		Message: evt.Message,
		Logger:  s.Logger(),
	}
	if ctx.Args = argsRe.FindAllString(evt.Message.Content, -1); ctx.Args == nil {
		bot.rejectCommand(ctx, "")
		return
	}
	commandID := ctx.Args[0]
	var ok bool
	if ctx.Command, ok = bot.commands[commandID]; ok {
		ctx.Logger.Info("Dispatching command: ", commandID)
		ctx.Command.Handler(ctx)
	} else if commandID == bot.helpCommand.ID {
		ctx.Logger.Info("Dispatching help command")
		bot.helpCommand.Handler(ctx)
	} else {
		bot.rejectCommand(ctx, commandID)
	}
}

func (bot *Bot) sendHelp(ctx Context) {
	go func() {
		if len(ctx.Args) > 1 {
			ctx.SendIncorrectUsageMsg()
			return
		}
		ctx.Send(bot.buildHelpEmbed())
	}()
}

func (bot *Bot) rejectCommand(ctx Context, commandID string) {
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
	for _, command := range bot.commandList {
		fields = append(fields, &disgord.EmbedField{
			Name:   command.FullUsage(),
			Value:  command.Desc,
			Inline: true,
		})
	}
	fields = append(fields, &disgord.EmbedField{
		Name:   bot.helpCommand.FullUsage(),
		Value:  bot.helpCommand.Desc,
		Inline: true,
	})
	embed := disgord.Embed{
		Title:       bot.Info.Name,
		Description: bot.Info.Desc + "\nSupported commands:",
		Fields:      fields,
	}
	return &embed
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
