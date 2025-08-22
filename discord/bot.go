package discord

import (
	"discordttsbot/config"
	"discordttsbot/logging"
	"discordttsbot/tts"
	"os"
	"sync"

	"github.com/bwmarrin/discordgo"
	"github.com/sirupsen/logrus"
)

type Bot struct {
	mutex   sync.Mutex
	session *discordgo.Session
	config  *config.Config
	tts     *tts.Service
	logger  *logrus.Logger
}

func NewBot(cfg *config.Config) (*Bot, error) {
	dg, err := discordgo.New("Bot " + cfg.Discord.Token)
	if err != nil {
		return nil, err
	}

	bot := &Bot{
		session: dg,
		config:  cfg,
		tts:     tts.NewService(cfg),
	}

	dg.AddHandler(bot.handleMesssage)
	dg.Identify.Intents = discordgo.IntentsGuilds | discordgo.IntentsGuildMessages | discordgo.IntentsGuildVoiceStates

	bot.logger = logrus.New()
	bot.logger.SetFormatter(&logging.ColorFormatter{Colors: true})
	bot.logger.SetLevel(logrus.DebugLevel)
	bot.logger.SetOutput(os.Stdout)

	return bot, nil
}

func (b *Bot) Start() error {
	return b.session.Open()
}

func (b *Bot) Stop() {
	b.session.Close()
}

func (b *Bot) getLoggerEntryWithCommandContext(s *discordgo.Session, m *discordgo.MessageCreate, command string) *logrus.Entry {
	return b.logger.WithFields(logrus.Fields{
		"command":    CmdTTS,
		"user_name":  m.Author.Username,
		"channel_id": m.ChannelID,
		"guild_id":   m.GuildID,
		"message_id": m.ID,
	})
}
