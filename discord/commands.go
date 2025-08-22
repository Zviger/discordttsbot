package discord

import (
	"bytes"
	"discordttsbot/utils"
	"fmt"
	"log"
	"net/url"
	"os"
	"strings"
	"time"

	"github.com/bwmarrin/discordgo"
	"github.com/sirupsen/logrus"
)

const (
	CommandPrefix = "!"

	CmdTTS        = "tts"
	CmdTTSFile    = "tts_file"
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
	case CmdTTSFile:
		b.handleTTSFileCommand(s, m)
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

func (b *Bot) parseTTSCommandArgs(command_args string) (text, voiceFileName, voiceLanguage string, err error) {
	args := strings.Split(strings.TrimSpace(command_args), "|")

	if len(args) != 3 {
		return "", "", "", fmt.Errorf("This command should be written as '%s language|fileName|text'", FullCommand(CmdTTS))
	}

	voiceFileName = args[1]

	fileStats, err := os.Stat(fmt.Sprintf("./tmp/assets/input_%s.wav", voiceFileName))

	if os.IsNotExist(err) {
		return "", "", "", fmt.Errorf("This speaker file doesn't exists!")
	}

	if fileStats.IsDir() {
		return "", "", "", fmt.Errorf("It's a directory!")
	}

	text = url.QueryEscape(args[2])

	voiceLanguage = args[0]

	if utils.NotInSlice(voiceLanguage, SupportedTTSLanguages) {
		return "", "", "", fmt.Errorf("This language is not supported! Choose one from: %s", strings.Join(SupportedTTSLanguages, ", "))
	}

	return text, voiceFileName, voiceLanguage, nil
}

func (b *Bot) handleTTSCommand(s *discordgo.Session, m *discordgo.MessageCreate) {
	logEntry := b.getLoggerEntryWithCommandContext(s, m, CmdTTS)
	logEntry.Debug("Command received")

	text, voiceFileName, voiceLanguage, err := b.parseTTSCommandArgs(strings.TrimPrefix(m.Content, FullCommand(CmdTTS)))
	if err != nil {
		errorMsg := fmt.Sprintf("Wrong params: %s", err)
		logEntry.Warn(errorMsg)
		_, _ = s.ChannelMessageSend(m.ChannelID, errorMsg)
		return
	}

	logEntry.WithFields(logrus.Fields{
		"text_length":    len(text),
		"voice_file":     voiceFileName,
		"voice_language": voiceLanguage,
	}).Debug("Command arguments parsed successfully")

	c, err := s.State.Channel(m.ChannelID)
	if err != nil {
		logEntry.Warn("Can't find the channel the message came from")
		return
	}

	g, err := s.State.Guild(c.GuildID)
	if err != nil {
		logEntry.Warn("Can't find the guild the message came from")
		return
	}

	speech, err := b.tts.GenerateSpeechInOpusFrames(text, voiceFileName, voiceLanguage, 0.1)
	if err != nil {
		errorMsg := fmt.Sprintf("Can't run TTS: %s", err)
		logEntry.Error(errorMsg)
		_, _ = s.ChannelMessageSend(m.ChannelID, errorMsg)
		return
	}

	logEntry.Debug("TTS file is generated and received")

	// Look for the message sender in that guild's current voice states.
	for _, vs := range g.VoiceStates {
		if vs.UserID == m.Author.ID {
			b.mutex.Lock()
			defer b.mutex.Unlock()

			vc, err := s.ChannelVoiceJoin(g.ID, vs.ChannelID, false, true)
			if err != nil {
				errorMsg := fmt.Sprintf("Can't join channel: %s", err)
				logEntry.Error(errorMsg)
				_, _ = s.ChannelMessageSend(m.ChannelID, errorMsg)
				return
			}

			// Sleep for a specified amount of time before playing the sound
			time.Sleep(250 * time.Millisecond)

			vc.Speaking(true)

			for _, speech_part := range speech {
				vc.OpusSend <- speech_part
			}

			vc.Speaking(false)

			// Sleep for a specificed amount of time before ending.
			time.Sleep(250 * time.Millisecond)

			vc.Disconnect()

			logEntry.Info("Command processed successfully")

			return
		}
	}

	errorMsg := "Can't join a voice channel. Looks like you are not in any voice channel"
	logEntry.Warn(errorMsg)
	_, _ = s.ChannelMessageSend(m.ChannelID, errorMsg)
}

func (b *Bot) handleTTSFileCommand(s *discordgo.Session, m *discordgo.MessageCreate) {
	logEntry := b.getLoggerEntryWithCommandContext(s, m, CmdTTSFile)
	logEntry.Debug("Command received")

	text, voiceFileName, voiceLanguage, err := b.parseTTSCommandArgs(strings.TrimPrefix(m.Content, FullCommand(CmdTTSFile)))
	if err != nil {
		errorMsg := fmt.Sprintf("Wrong params: %s", err)
		logEntry.Warn(errorMsg)
		_, _ = s.ChannelMessageSend(m.ChannelID, errorMsg)
		return
	}

	logEntry.WithFields(logrus.Fields{
		"text_length":    len(text),
		"voice_file":     voiceFileName,
		"voice_language": voiceLanguage,
	}).Debug("Command arguments parsed successfully")

	speech, err := b.tts.GenerateSpeechInWAV(text, voiceFileName, voiceLanguage)
	if err != nil {
		errorMsg := fmt.Sprintf("Can't generate TTS file: %s", err)
		logEntry.Error(errorMsg)
		_, _ = s.ChannelMessageSend(m.ChannelID, errorMsg)
		return
	}

	logEntry.Debug("TTS file is generated and received")

	message := &discordgo.MessageSend{
		Content: "Here's your file!",
		Files: []*discordgo.File{
			{
				Name:   fmt.Sprintf("%s.wav", voiceFileName),
				Reader: bytes.NewReader(speech),
			},
		},
	}

	_, err = s.ChannelMessageSendComplex(m.ChannelID, message)
	if err != nil {
		errorMsg := fmt.Sprintf("Can't send TTS file: %s", err)
		logEntry.Error(errorMsg)
		_, _ = s.ChannelMessageSend(m.ChannelID, errorMsg)
		return
	}

	logEntry.Info("Command processed successfully")
}

func (b *Bot) handleUploadCommand(s *discordgo.Session, m *discordgo.MessageCreate) {
	logEntry := b.getLoggerEntryWithCommandContext(s, m, CmdUploadFile)
	logEntry.Debug("Command received")

	file_name := strings.TrimSpace(strings.TrimPrefix(m.Content, FullCommand(CmdUploadFile)))

	if file_name == "" {
		errorMsg := "Speaker file name is not provided!"
		logEntry.Warn(errorMsg)
		_, _ = s.ChannelMessageSend(m.ChannelID, errorMsg)
		return
	}

	if len(m.Attachments) < 1 {
		errorMsg := "Speaker file is not provided!"
		logEntry.Warn(errorMsg)
		_, _ = s.ChannelMessageSend(m.ChannelID, errorMsg)
		return
	}

	err := b.tts.UploadVoiceFile(m.Attachments[0].URL, file_name)

	if err != nil {
		errorMsg := fmt.Sprintf("Can't upload file: %s", err)
		logEntry.Warn(errorMsg)
		_, _ = s.ChannelMessageSend(m.ChannelID, errorMsg)
		return
	}

	_, _ = s.ChannelMessageSend(m.ChannelID, fmt.Sprintf("File with name - %s uploaded", file_name))

	logEntry.Info("Command processed successfully")
}

func (b *Bot) handleListCommand(s *discordgo.Session, m *discordgo.MessageCreate) {
	logEntry := b.getLoggerEntryWithCommandContext(s, m, CmdListFiles)
	logEntry.Debug("Command received")

	files, err := b.tts.ListVoiceFiles()
	if err != nil {
		errorMsg := fmt.Sprintf("Can't list files: %s", err)
		logEntry.Warn(errorMsg)
		_, _ = s.ChannelMessageSend(m.ChannelID, errorMsg)
		return
	}

	var builder strings.Builder
	builder.WriteString("**Speaker Files**\n")
	builder.WriteString("```css\n")

	for i, file := range files {
		builder.WriteString(fmt.Sprintf("%2d. %s\n", i+1, file))
	}

	builder.WriteString("```")

	_, _ = s.ChannelMessageSend(m.ChannelID, builder.String())

	logEntry.Info("Command processed successfully")
}

func (b *Bot) handleDeleteCommand(s *discordgo.Session, m *discordgo.MessageCreate) {
	logEntry := b.getLoggerEntryWithCommandContext(s, m, CmdDeleteFile)
	logEntry.Debug("Command received")

	file_name := strings.TrimSpace(strings.TrimPrefix(m.Content, FullCommand(CmdDeleteFile)))

	if file_name == "" {
		errorMsg := "Speaker file name is not provided!"
		logEntry.Warn(errorMsg)
		_, _ = s.ChannelMessageSend(m.ChannelID, errorMsg)
		return
	}

	err := b.tts.DeleteVoiceFile(file_name)

	if err != nil {
		errorMsg := fmt.Sprintf("Can't delete file: %s", err)
		logEntry.Warn(errorMsg)
		_, _ = s.ChannelMessageSend(m.ChannelID, errorMsg)
		return
	}

	_, _ = s.ChannelMessageSend(m.ChannelID, fmt.Sprintf("File with name - %s deleted", file_name))

	logEntry.Info("Command processed successfully")
}

func (b *Bot) handleHelpCommand(s *discordgo.Session, m *discordgo.MessageCreate) {
	helpMessage := fmt.Sprintf(`**TTS Bot Help Menu**

**Main Commands:**
%s <language>|<file\_name>|<text> - Convert text to speech using specified voice
Example: !tts en|announcer|Hello world
%s <language>|<file\_name>|<text> - Does the same as previous command but returns a file with result.


**Voice Management:**
%s <file\_name> - Upload new voice (attach WAV file to message)
%s - Show available voices
%s <file\_name> - Delete voice

**Supported Languages:** `+strings.Join(SupportedTTSLanguages, ", ")+`

**Notes:**
1. For !tts you must be in a voice channel
2. Voice files must be in WAV format`,
		FullCommand(CmdTTS),
		FullCommand(CmdTTSFile),
		FullCommand(CmdUploadFile),
		FullCommand(CmdListFiles),
		FullCommand(CmdDeleteFile),
	)

	_, err := s.ChannelMessageSend(m.ChannelID, helpMessage)
	if err != nil {
		log.Println("Error sending help message:", err)
	}
}
