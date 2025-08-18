package discord

import (
	"discordttsbot/utils"
	"fmt"
	"log"
	"net/url"
	"os"
	"strings"
	"time"

	"github.com/bwmarrin/discordgo"
)

const (
	CommandPrefix = "!"

	CmdTTS        = "tts"
	CmdUploadFile = "upload_file"
	CmdListFiles  = "list_files"
	CmdDeleteFile = "delete_file"
	CmdHelp       = "help"
)

// FullCommand returns the prefixed command string
func FullCommand(cmd string) string {
	return CommandPrefix + cmd
}

func (b *Bot) handleMesssage(s *discordgo.Session, m *discordgo.MessageCreate) {
	// Ignore all messages created by the bot itself
	if m.Author.ID == s.State.User.ID {
		return
	}

	content := strings.TrimSpace(m.Content)
	if !strings.HasPrefix(content, CommandPrefix) {
		return
	}

	parts := strings.Fields(content)
	commandName := strings.TrimPrefix(parts[0], CommandPrefix)

	switch commandName {
	case CmdTTS:
		b.handleTTSCommand(s, m)
	case CmdUploadFile:
		b.handleUploadCommand(s, m)
	case CmdListFiles:
		b.handleListCommand(s, m)
	case CmdDeleteFile:
		b.handleDeleteCommand(s, m)
	case CmdHelp:
		b.handleHelpCommand(s, m)
	}
}

func (b *Bot) handleTTSCommand(s *discordgo.Session, m *discordgo.MessageCreate) {
	// Find the channel that the message came from.
	c, err := s.State.Channel(m.ChannelID)
	if err != nil {
		// Could not find channel.
		return
	}

	// Find the guild for that channel.
	g, err := s.State.Guild(c.GuildID)
	if err != nil {
		return
	}

	args := strings.Split(strings.TrimSpace(strings.TrimPrefix(m.Content, FullCommand(CmdTTS))), "|")

	if len(args) != 3 {
		_, _ = s.ChannelMessageSend(
			m.ChannelID,
			fmt.Sprintf("This command should be written as '%s ru|fileName|Text'", FullCommand(CmdTTS)),
		)
		return
	}

	voiceFileName := args[1]

	fileStats, err := os.Stat(fmt.Sprintf("./tmp/assets/input_%s.wav", voiceFileName))

	if os.IsNotExist(err) {
		_, _ = s.ChannelMessageSend(m.ChannelID, "This speaker file doesn't exists!")
		return
	}

	if fileStats.IsDir() {
		_, _ = s.ChannelMessageSend(m.ChannelID, "It's a directory!")
		return
	}

	text := url.QueryEscape(args[2])

	voiceLanguage := args[0]

	if utils.NotInSlice(voiceLanguage, SupportedTTSLanguages) {
		_, _ = s.ChannelMessageSend(m.ChannelID, fmt.Sprintf("This language is not supported! Choose one from: %s", strings.Join(SupportedTTSLanguages, ", ")))
		return
	}

	speech, err := b.tts.GenerateSpeech(text, voiceFileName, voiceLanguage, 0.1)
	if err != nil {
		_, _ = s.ChannelMessageSend(m.ChannelID, fmt.Sprintf("Can't run TTS: %s", err))
		return
	}

	// Look for the message sender in that guild's current voice states.
	for _, vs := range g.VoiceStates {
		if vs.UserID == m.Author.ID {

			b.mutex.Lock()
			defer b.mutex.Unlock()

			// Join the provided voice channel.
			vc, err := s.ChannelVoiceJoin(g.ID, vs.ChannelID, false, true)
			if err != nil {
				_, _ = s.ChannelMessageSend(m.ChannelID, fmt.Sprintf("Can't join channel: %s", err))
				return
			}

			// Sleep for a specified amount of time before playing the sound
			time.Sleep(250 * time.Millisecond)

			// Start speaking.
			vc.Speaking(true)

			for _, speech_part := range *speech {
				// Send to Discord
				vc.OpusSend <- speech_part
			}

			// Stop speaking
			vc.Speaking(false)

			// Sleep for a specificed amount of time before ending.
			time.Sleep(250 * time.Millisecond)

			// Disconnect from the provided voice channel.
			vc.Disconnect()

			return
		}
	}
}

func (b *Bot) handleUploadCommand(s *discordgo.Session, m *discordgo.MessageCreate) {
	file_name := strings.TrimSpace(strings.TrimPrefix(m.Content, FullCommand(CmdUploadFile)))

	if file_name == "" {
		_, _ = s.ChannelMessageSend(m.ChannelID, "Speaker file name is not provided!")
		return
	}

	if len(m.Attachments) < 1 {
		_, _ = s.ChannelMessageSend(m.ChannelID, "Speaker file is not provided!")
		return
	}

	err := b.tts.UploadVoiceFile(m.Attachments[0].URL, file_name)

	if err != nil {
		_, _ = s.ChannelMessageSend(m.ChannelID, fmt.Sprintf("Can't upload file: %s", err))
	} else {
		_, _ = s.ChannelMessageSend(m.ChannelID, fmt.Sprintf("File with name - %s uploaded", file_name))
	}
}

func (b *Bot) handleListCommand(s *discordgo.Session, m *discordgo.MessageCreate) {
	files, err := b.tts.ListVoiceFiles()
	if err != nil {
		_, _ = s.ChannelMessageSend(m.ChannelID, fmt.Sprintf("Can't list files: %s", err))
	}

	var builder strings.Builder
	builder.WriteString("**Speaker Files**\n")
	builder.WriteString("```css\n")

	for i, file := range files {
		builder.WriteString(fmt.Sprintf("%2d. %s\n", i+1, file))
	}

	builder.WriteString("```")

	_, _ = s.ChannelMessageSend(m.ChannelID, builder.String())
}

func (b *Bot) handleDeleteCommand(s *discordgo.Session, m *discordgo.MessageCreate) {
	file_name := strings.TrimSpace(strings.TrimPrefix(m.Content, FullCommand(CmdDeleteFile)))

	if file_name == "" {
		_, _ = s.ChannelMessageSend(m.ChannelID, "Speaker file name is not provided!")
		return
	}

	err := b.tts.DeleteVoiceFile(file_name)

	if err != nil {
		_, _ = s.ChannelMessageSend(m.ChannelID, fmt.Sprintf("Can't delete file: %s", err))
	} else {
		_, _ = s.ChannelMessageSend(m.ChannelID, fmt.Sprintf("File with name - %s deleted", file_name))
	}
}

func (b *Bot) handleHelpCommand(s *discordgo.Session, m *discordgo.MessageCreate) {
	helpMessage := fmt.Sprintf(`**TTS Bot Help Menu**

**Main Commands:**
%s <language>|<file\_name>|<text> - Convert text to speech using specified voice
Example: !tts en|announcer|Hello world

**Voice Management:**
%s <file\_name> - Upload new voice (attach WAV file to message)
%s - Show available voices
%s <file\_name> - Delete voice

**Supported Languages:** `+strings.Join(SupportedTTSLanguages, ", ")+`

**Notes:**
1. For !tts you must be in a voice channel
2. Voice files must be in WAV format`,
		FullCommand(CmdTTS),
		FullCommand(CmdUploadFile),
		FullCommand(CmdListFiles),
		FullCommand(CmdDeleteFile),
	)

	_, err := s.ChannelMessageSend(m.ChannelID, helpMessage)
	if err != nil {
		log.Println("Error sending help message:", err)
	}
}
