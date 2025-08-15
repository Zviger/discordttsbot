package main

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/url"
	"os"
	"os/signal"
	"path/filepath"
	"regexp"
	"syscall"

	"strings"
	"time"

	"github.com/bwmarrin/discordgo"
	"gopkg.in/hraban/opus.v2"
)

var config *Config

func main() {
	var err error
	config, err = LoadConfig("config.json")
	if err != nil {
		log.Fatal("Failed to load config:", err)
	}

	// Create a new Discord session using the provided bot token.
	dg, err := discordgo.New("Bot " + config.Discord.Token)
	if err != nil {
		fmt.Println("Error creating Discord session: ", err)
		return
	}

	// Register ready as a callback for the ready events.
	dg.AddHandler(ready)

	// Register messageCreate as a callback for the messageCreate events.
	dg.AddHandler(messageCreate)

	// We need information about guilds (which includes their channels),
	// messages and voice states.
	dg.Identify.Intents = discordgo.IntentsGuilds | discordgo.IntentsGuildMessages | discordgo.IntentsGuildVoiceStates

	// Open the websocket and begin listening.
	err = dg.Open()
	if err != nil {
		fmt.Println("Error opening Discord session: ", err)
	}

	// Wait here until CTRL-C or other term signal is received.
	fmt.Println("TTS is now running.  Press CTRL-C to exit.")
	sc := make(chan os.Signal, 1)
	signal.Notify(sc, syscall.SIGINT, syscall.SIGTERM, os.Interrupt)
	<-sc

	// Cleanly close down the Discord session.
	dg.Close()
}

func ready(s *discordgo.Session, event *discordgo.Ready) {
	s.UpdateGameStatus(0, "!tts")
}

func messageCreate(s *discordgo.Session, m *discordgo.MessageCreate) {
	// Ignore all messages created by the bot itself
	if m.Author.ID == s.State.User.ID {
		return
	}

	if strings.HasPrefix(m.Content, "!tts") {
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

		args := strings.Split(strings.TrimSpace(strings.TrimPrefix(m.Content, "!tts")), "|")

		if len(args) != 3 {
			_, _ = s.ChannelMessageSend(m.ChannelID, "This command should be written as '!tts ru|fileName|Text'")
			return
		}

		speakerFileName := args[1]

		fileStats, err := os.Stat(fmt.Sprintf("./tmp/assets/input_%s.wav", speakerFileName))

		if os.IsNotExist(err) {
			_, _ = s.ChannelMessageSend(m.ChannelID, "This speaker file doesn't exists!")
			return
		}

		if fileStats.IsDir() {
			_, _ = s.ChannelMessageSend(m.ChannelID, "It's a directory!")
			return
		}

		text := url.QueryEscape(args[2])

		speakerLanguage := args[0]

		if NotInSlice(speakerLanguage, SupportedTTSLanguages) {
			_, _ = s.ChannelMessageSend(m.ChannelID, fmt.Sprintf("This language is not supported! Choose one from: %s", strings.Join(SupportedTTSLanguages, ", ")))
			return
		}

		// Look for the message sender in that guild's current voice states.
		for _, vs := range g.VoiceStates {
			if vs.UserID == m.Author.ID {
				err = playTTS(s, g.ID, vs.ChannelID, text, fmt.Sprintf("/root/assets/input_%s.wav", speakerFileName), speakerLanguage, 0.1)
				if err != nil {
					_, _ = s.ChannelMessageSend(m.ChannelID, fmt.Sprintf("Can't run TTS: %s", err))
				}
				return
			}
		}
	} else if strings.HasPrefix(m.Content, "!upload_file") {
		file_name := strings.TrimSpace(strings.TrimPrefix(m.Content, "!upload_file"))

		if file_name == "" {
			_, _ = s.ChannelMessageSend(m.ChannelID, "Speaker file name is not provided!")
			return
		}

		if len(m.Attachments) < 1 {
			_, _ = s.ChannelMessageSend(m.ChannelID, "Speaker file is not provided!")
			return
		}

		err := uploadSpeakerFile(m.Attachments[0].URL, file_name)

		if err != nil {
			_, _ = s.ChannelMessageSend(m.ChannelID, fmt.Sprintf("Can't run TTS: %s", err))
		} else {
			_, _ = s.ChannelMessageSend(m.ChannelID, fmt.Sprintf("File with name - %s uploaded", file_name))
		}

		return
	} else if strings.HasPrefix(m.Content, "!list_files") {
		err := listSpeakerFiles(s, m.ChannelID)
		if err != nil {
			_, _ = s.ChannelMessageSend(m.ChannelID, fmt.Sprintf("Can't list files: %s", err))
		}

		return
	}
}

// playTTS plays the current buffer to the provided channel.
func playTTS(s *discordgo.Session, guildID, channelID, text, speakerFilePath, language string, volume float64) (err error) {
	if volume < 0 {
		volume = 0
	} else if volume > 2.0 {
		volume = 2.0
	}

	url := fmt.Sprintf(
		"%s/api/tts?text=%s&speaker_wav=%s&language_id=%s",
		config.TTS.ApiUrl,
		text,
		speakerFilePath,
		language,
	)

	// Download the WAV file
	resp, err := http.Get(url)
	if err != nil {
		log.Println("Error downloading audio:", err)
		return err
	}
	defer resp.Body.Close()

	// Read the WAV file into memory
	wavData, err := io.ReadAll(resp.Body)
	if err != nil {
		log.Println("Error reading audio data:", err)
		return err
	}

	// Join the provided voice channel.
	vc, err := s.ChannelVoiceJoin(guildID, channelID, false, true)
	if err != nil {
		return err
	}

	// Sleep for a specified amount of time before playing the sound
	time.Sleep(250 * time.Millisecond)

	// Start speaking.
	vc.Speaking(true)

	// Convert WAV to PCM (simplified - you may need proper WAV parsing)
	// This assumes the WAV is 16-bit 48kHz mono
	pcmData := bytes.NewBuffer(wavData[44:]) // Skip WAV header (44 bytes)

	// Create Opus encoder
	encoder, err := opus.NewEncoder(24000, 1, opus.AppAudio)
	if err != nil {
		log.Println("Error creating encoder:", err)
		return err
	}

	// Encode PCM to Opus and send to Discord
	frameSize := 480 // 20ms frame at 48kHz
	pcm := make([]int16, frameSize)
	adjustedPcm := make([]int16, frameSize)
	opusFrame := make([]byte, 1000) // Sufficient size for Opus frame

	for {
		// Read PCM data
		err := binary.Read(pcmData, binary.LittleEndian, &pcm)
		if err == io.EOF {
			break
		} else if err != nil {
			break
		}

		for i, sample := range pcm {
			adjusted := float64(sample) * volume

			if adjusted > 32767 {
				adjusted = 32767
			} else if adjusted < -32768 {
				adjusted = -32768
			}

			adjustedPcm[i] = int16(adjusted)
		}

		// Encode to Opus
		n, err := encoder.Encode(adjustedPcm, opusFrame)
		if err != nil {
			log.Println("Error encoding:", err)
			break
		}

		// Send to Discord
		vc.OpusSend <- opusFrame[:n]
	}

	// Disconnect when done
	vc.Disconnect()

	// Stop speaking
	vc.Speaking(false)

	// Sleep for a specificed amount of time before ending.
	time.Sleep(250 * time.Millisecond)

	// Disconnect from the provided voice channel.
	vc.Disconnect()

	return nil
}

func uploadSpeakerFile(file_url string, file_name string) error {
	filePath := fmt.Sprintf("./tmp/assets/input_%s.wav", file_name)

	out, err := os.Create(filePath)
	if err != nil {
		return err
	}
	defer out.Close()

	resp, err := http.Get(file_url)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	_, err = io.Copy(out, resp.Body)
	if err != nil {
		return err
	}

	return nil
}

func listSpeakerFiles(s *discordgo.Session, channelID string) error {
	files, err := filepath.Glob("./tmp/assets/input_*.wav")
	if err != nil {
		return err
	}

	var builder strings.Builder
	builder.WriteString("**Speaker Files**\n")
	builder.WriteString("```css\n")

	for i, file := range files {
		re := regexp.MustCompile(`.*input_(.*).wav`)
		mathces := re.FindStringSubmatch(file)
		builder.WriteString(fmt.Sprintf("%2d. %s\n", i+1, mathces[1]))
	}

	builder.WriteString("```")

	_, err = s.ChannelMessageSend(channelID, builder.String())
	if err != nil {
		return err
	}

	return nil
}
