package bot

import (
	"fmt"
	"regexp"
	"strings"

	"github.com/andersfylling/disgord"
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
	Name       string
	Token      string
	Prefix     string
	Desc       string
	SupportURL string
	Logger     disgord.Logger
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
func (bot *Bot) OnMessageCreate(handler func(Context, *disgord.MessageCreate)) {
	wrapped := func(s disgord.Session, evt *disgord.MessageCreate) {
		if evt.Message.Author == nil || evt.Message.Author.Bot {
			return
		}
		ctx := Context{
			Bot:     bot,
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
	bot.commandList = append(bot.commandList, command)
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
	ctx := Context{
		Bot:     bot,
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
