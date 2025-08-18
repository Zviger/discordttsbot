package discord

import (
	"discordttsbot/config"
	"discordttsbot/tts"

	"github.com/bwmarrin/discordgo"
)

type Bot struct {
	session *discordgo.Session
	config  *config.Config
	tts     *tts.Service
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

	return bot, nil
}

func (b *Bot) Start() error {
	return b.session.Open()
}

func (b *Bot) Stop() {
	b.session.Close()
}
